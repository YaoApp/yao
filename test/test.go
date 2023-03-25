package test

import (
	"fmt"
	"os"
	"testing"

	"github.com/yaoapp/gou/application"
	"github.com/yaoapp/gou/model"
	"github.com/yaoapp/gou/query"
	"github.com/yaoapp/gou/query/gou"
	v8 "github.com/yaoapp/gou/runtime/v8"
	"github.com/yaoapp/kun/exception"
	"github.com/yaoapp/xun/capsule"
	"github.com/yaoapp/yao/config"
	"github.com/yaoapp/yao/fs"
	"github.com/yaoapp/yao/runtime"
	"github.com/yaoapp/yao/share"
)

// Prepare test environment
func Prepare(t *testing.T, cfg config.Config) {
	root := os.Getenv("YAO_TEST_APPLICATION")
	var app application.Application
	var err error

	if root == "bin:application.pkg" {
		key := os.Getenv("YAO_TEST_PRIVATE_KEY")
		app, err = application.OpenFromBin(root, key) // Load app from Bin
		if err != nil {
			t.Fatal(err)
		}
		application.Load(app)
		return
	}

	app, err = application.OpenFromDisk(root) // Load app from Disk
	if err != nil {
		t.Fatal(err)
	}
	application.Load(app)

	dbconnect(t, cfg)
	load(t, cfg)
	start(t, cfg)
}

// Clean the test environment
func Clean() {
	dbclose()
	runtime.Stop()
}

func dbclose() {
	if capsule.Global != nil {
		capsule.Global.Connections.Range(func(key, value any) bool {
			if conn, ok := value.(*capsule.Connection); ok {
				conn.Close()
			}
			return true
		})
	}
}

func dbconnect(t *testing.T, cfg config.Config) {

	// connect db
	switch cfg.DB.Driver {
	case "sqlite3":
		capsule.AddConn("primary", "sqlite3", cfg.DB.Primary[0]).SetAsGlobal()
		break
	default:
		capsule.AddConn("primary", "mysql", cfg.DB.Primary[0]).SetAsGlobal()
		break
	}

}

func start(t *testing.T, cfg config.Config) {
	err := runtime.Start(cfg)
	if err != nil {
		t.Fatal(err)
	}
}

func load(t *testing.T, cfg config.Config) {
	loadFS(t, cfg)
	loadScript(t, cfg)
	loadModel(t, cfg)
	loadQuery(t, cfg)
}

func loadFS(t *testing.T, cfg config.Config) {
	err := fs.Load(cfg)
	if err != nil {
		t.Fatal(err)
	}
}

func loadScript(t *testing.T, cfg config.Config) {
	exts := []string{"*.js"}
	err := application.App.Walk("scripts", func(root, file string, isdir bool) error {
		if isdir {
			return nil
		}
		_, err := v8.Load(file, share.ID(root, file))
		return err
	}, exts...)

	if err != nil {
		t.Fatal(err)
	}
}

func loadModel(t *testing.T, cfg config.Config) {
	model.WithCrypt([]byte(fmt.Sprintf(`{"key":"%s"}`, cfg.DB.AESKey)), "AES")
	model.WithCrypt([]byte(`{}`), "PASSWORD")

	exts := []string{"*.mod.yao", "*.mod.json", "*.mod.jsonc"}
	err := application.App.Walk("models", func(root, file string, isdir bool) error {
		if isdir {
			return nil
		}
		_, err := model.Load(file, share.ID(root, file))
		return err
	}, exts...)

	if err != nil {
		t.Fatal(err)
	}
}

func loadQuery(t *testing.T, cfg config.Config) {

	// query engine
	query.Register("query-test", &gou.Query{
		Query: capsule.Query(),
		GetTableName: func(s string) string {
			if mod, has := model.Models[s]; has {
				return mod.MetaData.Table.Name
			}
			exception.New("[query] %s not found", 404, s).Throw()
			return s
		},
		AESKey: cfg.DB.AESKey,
	})
}
