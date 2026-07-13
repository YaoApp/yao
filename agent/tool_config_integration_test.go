//go:build integration

package agent_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/yaoapp/yao/agent"
	"github.com/yaoapp/yao/setting"
	"github.com/yaoapp/yao/unit-test/agent/testprepare"
)

func TestSyncSearchDefaults(t *testing.T) {
	testprepare.PrepareSandbox(t)
	appDir := testprepare.AgentAppDir()
	agentDir := filepath.Join(appDir, "agent")

	t.Run("websearch_and_webfetch", func(t *testing.T) {
		wsFile := filepath.Join(agentDir, "websearch.yml")
		wfFile := filepath.Join(agentDir, "webfetch.yml")
		t.Cleanup(func() {
			os.Remove(wsFile)
			os.Remove(wfFile)
		})

		os.Setenv("TEST_SERPER_KEY", "sk-test-serper-123")
		os.Setenv("TEST_BD_KEY", "sk-test-bd-456")
		t.Cleanup(func() {
			os.Unsetenv("TEST_SERPER_KEY")
			os.Unsetenv("TEST_BD_KEY")
		})

		wsContent := []byte("default: serper\nproviders:\n  serper:\n    api_key: $ENV.TEST_SERPER_KEY\n")
		if err := os.WriteFile(wsFile, wsContent, 0644); err != nil {
			t.Fatalf("write websearch.yml: %v", err)
		}

		wfContent := []byte("default: brightdata\nproviders:\n  brightdata:\n    api_key: $ENV.TEST_BD_KEY\n    zone: web_unlocker1\n")
		if err := os.WriteFile(wfFile, wfContent, 0644); err != nil {
			t.Fatalf("write webfetch.yml: %v", err)
		}

		if err := agent.SyncSearchDefaults(); err != nil {
			t.Fatalf("SyncSearchDefaults: %v", err)
		}

		systemScope := setting.ScopeID{Scope: setting.ScopeSystem}

		assignment, err := setting.Global.Get(systemScope, "search.tool_assignment")
		if err != nil {
			t.Fatalf("get search.tool_assignment: %v", err)
		}
		if v, ok := assignment["web_search"].(string); !ok || v != "serper" {
			t.Errorf("web_search = %v, want 'serper'", assignment["web_search"])
		}
		if v, ok := assignment["web_scrape"].(string); !ok || v != "brightdata" {
			t.Errorf("web_scrape = %v, want 'brightdata'", assignment["web_scrape"])
		}

		serperData, err := setting.Global.Get(systemScope, "search.providers.serper")
		if err != nil {
			t.Fatalf("get search.providers.serper: %v", err)
		}
		fieldValues, ok := serperData["field_values"].(map[string]interface{})
		if !ok {
			t.Fatalf("serper field_values not a map: %T", serperData["field_values"])
		}
		apiKey, _ := fieldValues["api_key"].(string)
		if apiKey == "" {
			t.Fatal("serper api_key is empty")
		}
		if apiKey == "sk-test-serper-123" {
			t.Error("serper api_key should be encrypted, got plaintext")
		}

		bdData, err := setting.Global.Get(systemScope, "search.providers.brightdata")
		if err != nil {
			t.Fatalf("get search.providers.brightdata: %v", err)
		}
		bdFields, ok := bdData["field_values"].(map[string]interface{})
		if !ok {
			t.Fatalf("brightdata field_values not a map: %T", bdData["field_values"])
		}
		if zone, _ := bdFields["zone"].(string); zone != "web_unlocker1" {
			t.Errorf("brightdata zone = %q, want 'web_unlocker1'", zone)
		}
	})

	t.Run("empty_api_key_skipped", func(t *testing.T) {
		wsFile := filepath.Join(agentDir, "websearch.yml")
		t.Cleanup(func() { os.Remove(wsFile) })

		os.Unsetenv("TEST_EMPTY_KEY")

		wsContent := []byte("default: tavily\nproviders:\n  tavily:\n    api_key: $ENV.TEST_EMPTY_KEY\n")
		if err := os.WriteFile(wsFile, wsContent, 0644); err != nil {
			t.Fatalf("write websearch.yml: %v", err)
		}

		// Delete any existing tavily provider data
		systemScope := setting.ScopeID{Scope: setting.ScopeSystem}
		_ = setting.Global.Delete(systemScope, "search.providers.tavily")

		if err := agent.SyncSearchDefaults(); err != nil {
			t.Fatalf("SyncSearchDefaults: %v", err)
		}

		_, err := setting.Global.Get(systemScope, "search.providers.tavily")
		if err == nil {
			t.Error("tavily provider should not be written when api_key is empty")
		}
	})

	t.Run("direct_default_no_providers", func(t *testing.T) {
		wfFile := filepath.Join(agentDir, "webfetch.yml")
		t.Cleanup(func() { os.Remove(wfFile) })

		wfContent := []byte("default: direct\n")
		if err := os.WriteFile(wfFile, wfContent, 0644); err != nil {
			t.Fatalf("write webfetch.yml: %v", err)
		}

		if err := agent.SyncSearchDefaults(); err != nil {
			t.Fatalf("SyncSearchDefaults: %v", err)
		}

		systemScope := setting.ScopeID{Scope: setting.ScopeSystem}
		assignment, err := setting.Global.Get(systemScope, "search.tool_assignment")
		if err != nil {
			t.Fatalf("get search.tool_assignment: %v", err)
		}
		if v, ok := assignment["web_scrape"].(string); !ok || v != "direct" {
			t.Errorf("web_scrape = %v, want 'direct'", assignment["web_scrape"])
		}
	})

	t.Run("missing_api_key_skipped", func(t *testing.T) {
		wfFile := filepath.Join(agentDir, "webfetch.yml")
		t.Cleanup(func() { os.Remove(wfFile) })

		wfContent := []byte("default: brightdata\nproviders:\n  brightdata:\n    zone: web_unlocker1\n")
		if err := os.WriteFile(wfFile, wfContent, 0644); err != nil {
			t.Fatalf("write webfetch.yml: %v", err)
		}

		systemScope := setting.ScopeID{Scope: setting.ScopeSystem}
		_ = setting.Global.Delete(systemScope, "search.providers.brightdata")

		if err := agent.SyncSearchDefaults(); err != nil {
			t.Fatalf("SyncSearchDefaults: %v", err)
		}

		_, err := setting.Global.Get(systemScope, "search.providers.brightdata")
		if err == nil {
			t.Error("brightdata provider should not be written when api_key field is missing")
		}
	})

	t.Run("unknown_default_ignored", func(t *testing.T) {
		wsFile := filepath.Join(agentDir, "websearch.yml")
		t.Cleanup(func() { os.Remove(wsFile) })

		wsContent := []byte("default: unknown_provider\n")
		if err := os.WriteFile(wsFile, wsContent, 0644); err != nil {
			t.Fatalf("write websearch.yml: %v", err)
		}

		// Set a known value first
		systemScope := setting.ScopeID{Scope: setting.ScopeSystem}
		setting.Global.Set(systemScope, "search.tool_assignment", map[string]interface{}{
			"web_search": "serper",
		})

		if err := agent.SyncSearchDefaults(); err != nil {
			t.Fatalf("SyncSearchDefaults: %v", err)
		}

		assignment, err := setting.Global.Get(systemScope, "search.tool_assignment")
		if err != nil {
			t.Fatalf("get search.tool_assignment: %v", err)
		}
		if v, _ := assignment["web_search"].(string); v != "serper" {
			t.Errorf("web_search = %v, want 'serper' (should not be overwritten by unknown default)", v)
		}
	})
}
