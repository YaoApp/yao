package agent

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/yaoapp/gou/application"
	"github.com/yaoapp/gou/store"
	"github.com/yaoapp/yao/config"
	"github.com/yaoapp/yao/setting"
)

func TestSyncParallelSearchDefaults(t *testing.T) {
	oldApp, oldGlobal, oldConfig := application.App, setting.Global, config.Conf
	oldStore, oldCache := store.Pools["__yao.store"], store.Pools["__yao.cache"]
	t.Cleanup(func() {
		application.App, setting.Global, config.Conf = oldApp, oldGlobal, oldConfig
		store.Pools["__yao.store"], store.Pools["__yao.cache"] = oldStore, oldCache
	})
	config.Conf.DB.AESKey = "parallel-test-encryption-key"
	t.Setenv("TEST_PARALLEL_KEY", "configured-key")
	appDir := t.TempDir()
	if err := os.Mkdir(filepath.Join(appDir, "agent"), 0755); err != nil {
		t.Fatal(err)
	}
	app, err := application.OpenFromDisk(appDir)
	if err != nil {
		t.Fatal(err)
	}
	application.App = app
	for _, tc := range []struct{ name, providers string }{
		{"anonymous", ""},
		{"authenticated", "providers:\n  parallel:\n    api_key: $ENV.TEST_PARALLEL_KEY\n"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s, err := store.New(nil, store.Option{"size": 100})
			if err != nil {
				t.Fatal(err)
			}
			store.Pools["__yao.store"] = s
			delete(store.Pools, "__yao.cache")
			if err := setting.Init(); err != nil {
				t.Fatal(err)
			}
			userScope := setting.ScopeID{Scope: setting.ScopeUser, UserID: "existing-user"}
			if _, err := setting.Global.Set(userScope, "search.tool_assignment", map[string]interface{}{"web_search": "serper"}); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(appDir, "agent", "websearch.yml"), []byte("default: parallel\n"+tc.providers), 0644); err != nil {
				t.Fatal(err)
			}
			if err := SyncSearchDefaults(); err != nil {
				t.Fatal(err)
			}
			assignment, err := setting.Global.GetMerged("", "", "search.tool_assignment")
			if err != nil || assignment["web_search"] != "parallel" {
				t.Fatalf("system assignment = %v, error = %v", assignment, err)
			}
			assignment, err = setting.Global.GetMerged("existing-user", "", "search.tool_assignment")
			if err != nil || assignment["web_search"] != "serper" {
				t.Fatalf("user assignment = %v, error = %v", assignment, err)
			}
			saved, err := setting.Global.GetMerged("", "", "search.providers.parallel")
			if tc.providers == "" {
				if err == nil {
					t.Fatalf("anonymous config unexpectedly wrote credentials: %v", saved)
				}
				return
			}
			if err != nil {
				t.Fatalf("explicit Parallel key was not saved: %v", err)
			}
			fields, ok := saved["field_values"].(map[string]interface{})
			if !ok {
				t.Fatalf("missing fields: %v", saved)
			}
			key, _ := fields["api_key"].(string)
			if !setting.IsEncrypted(key) || setting.Decrypt(key) != "configured-key" {
				t.Fatal("explicit Parallel key was not encrypted and preserved")
			}
		})
	}
}
