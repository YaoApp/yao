package test

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	jsoniter "github.com/json-iterator/go"
	"github.com/yaoapp/gou/api"
	"github.com/yaoapp/gou/application"
	"github.com/yaoapp/gou/connector"
	"github.com/yaoapp/gou/model"
	"github.com/yaoapp/gou/query"
	"github.com/yaoapp/gou/query/gou"
	v8 "github.com/yaoapp/gou/runtime/v8"
	"github.com/yaoapp/gou/server/http"
	"github.com/yaoapp/gou/store"
	"github.com/yaoapp/kun/exception"
	"github.com/yaoapp/kun/log"
	"github.com/yaoapp/xun/capsule"
	"github.com/yaoapp/yao/config"
	"github.com/yaoapp/yao/data"
	"github.com/yaoapp/yao/fs"
	"github.com/yaoapp/yao/helper"
	"github.com/yaoapp/yao/runtime"
	"github.com/yaoapp/yao/share"
	"github.com/yaoapp/yao/utils"
)

var testServer *http.Server = nil

// SystemModels system models for testing
var testSystemModels = map[string]string{
	"__yao.assistant":          "yao/models/assistant.mod.yao",
	"__yao.attachment":         "yao/models/attachment.mod.yao",
	"__yao.audit":              "yao/models/audit.mod.yao",
	"__yao.chat":               "yao/models/chat.mod.yao",
	"__yao.config":             "yao/models/config.mod.yao",
	"__yao.dsl":                "yao/models/dsl.mod.yao",
	"__yao.history":            "yao/models/history.mod.yao",
	"__yao.job.category":       "yao/models/job/category.mod.yao",
	"__yao.job":                "yao/models/job/job.mod.yao",
	"__yao.job.execution":      "yao/models/job/execution.mod.yao",
	"__yao.job.log":            "yao/models/job/log.mod.yao",
	"__yao.kb.collection":      "yao/models/kb/collection.mod.yao",
	"__yao.kb.document":        "yao/models/kb/document.mod.yao",
	"__yao.user":               "yao/models/user.mod.yao",
	"__yao.user_role":          "yao/models/user_role.mod.yao",
	"__yao.user_type":          "yao/models/user_type.mod.yao",
	"__yao.user_oauth_account": "yao/models/user_oauth_account.mod.yao",
}

var testSystemStores = map[string]string{
	"__yao.store":        "yao/stores/store.badger.yao",
	"__yao.cache":        "yao/stores/cache.lru.yao",
	"__yao.oauth.store":  "yao/stores/oauth/store.badger.yao",
	"__yao.oauth.client": "yao/stores/oauth/client.badger.yao",
	"__yao.oauth.cache":  "yao/stores/oauth/cache.lru.yao",
	"__yao.agent.memory": "yao/stores/agent/memory.badger.yao",
	"__yao.kb.store":     "yao/stores/kb/store.badger.yao",
	"__yao.kb.cache":     "yao/stores/kb/cache.lru.yao",
}

func loadSystemStores(t *testing.T, cfg config.Config) error {
	for id, path := range testSystemStores {
		raw, err := data.Read(path)
		if err != nil {
			return err
		}

		// Replace template variables in the JSON string
		source := string(raw)
		if strings.Contains(source, "YAO_APP_ROOT") || strings.Contains(source, "YAO_DATA_ROOT") {
			vars := map[string]string{
				"YAO_APP_ROOT":  cfg.Root,
				"YAO_DATA_ROOT": cfg.DataRoot,
			}
			source = replaceVars(source, vars)
		}

		// Load store with the processed source
		_, err = store.LoadSource([]byte(source), id, filepath.Join("__system", path))
		if err != nil {
			log.Error("load system store %s error: %s", id, err.Error())
			return err
		}
	}
	return nil
}

// replaceVars replaces template variables in the JSON string
// Supports {{ VAR_NAME }} syntax
func replaceVars(jsonStr string, vars map[string]string) string {
	result := jsonStr
	for key, value := range vars {
		// Replace both {{ KEY }} and {{KEY}} patterns
		patterns := []string{
			"{{ " + key + " }}",
			"{{" + key + "}}",
		}
		for _, pattern := range patterns {
			result = strings.ReplaceAll(result, pattern, value)
		}
	}
	return result
}

// loadSystemModels load system models for testing
func loadSystemModels(t *testing.T, cfg config.Config) error {
	for id, path := range testSystemModels {
		content, err := data.Read(path)
		if err != nil {
			return err
		}

		// Parse model
		var data map[string]interface{}
		err = application.Parse(path, content, &data)
		if err != nil {
			return err
		}

		// Set prefix
		if table, ok := data["table"].(map[string]interface{}); ok {
			if name, ok := table["name"].(string); ok {
				table["name"] = share.App.Prefix + name
				content, err = jsoniter.Marshal(data)
				if err != nil {
					log.Error("failed to marshal model data: %v", err)
					return fmt.Errorf("failed to marshal model data: %v", err)
				}
			}
		}

		// Load Model
		mod, err := model.LoadSource(content, id, filepath.Join("__system", path))
		if err != nil {
			log.Error("load system model %s error: %s", id, err.Error())
			return err
		}

		// Auto migrate
		err = mod.Migrate(false, model.WithDonotInsertValues(true))
		if err != nil {
			log.Error("migrate system model %s error: %s", id, err.Error())
			return err
		}
	}

	return nil
}

// Prepare test environment
func Prepare(t *testing.T, cfg config.Config, rootEnv ...string) {

	appRootEnv := "YAO_TEST_APPLICATION"
	if len(rootEnv) > 0 {
		appRootEnv = rootEnv[0]
	}

	// Remove the data store
	var path = filepath.Join(os.Getenv(appRootEnv), "data", "stores")
	os.RemoveAll(path)

	root := os.Getenv(appRootEnv)
	var app application.Application
	var err error

	// if share.BUILDIN {

	// 	file, err := os.Executable()
	// 	if err != nil {
	// 		t.Fatal(err)
	// 	}

	// 	// Load from cache
	// 	app, err := application.OpenFromYazCache(file, pack.Cipher)

	// 	if err != nil {

	// 		// load from bin
	// 		reader, err := data.ReadApp()
	// 		if err != nil {
	// 			t.Fatal(err)
	// 		}

	// 		app, err = application.OpenFromYaz(reader, file, pack.Cipher) // Load app from Bin
	// 		if err != nil {
	// 			t.Fatal(err)
	// 		}
	// 	}

	// 	application.Load(app)
	// 	data.RemoveApp()
	// 	return
	// }

	app, err = application.OpenFromDisk(root) // Load app from Disk
	if err != nil {
		t.Fatal(err)
	}
	application.Load(app)

	cfg.DataRoot = filepath.Join(root, "data")

	// if cfg.DataRoot == "" {
	// 	cfg.DataRoot = filepath.Join(root, "data")
	// }

	var appData []byte
	var appFile string

	// Read app setting
	if has, _ := application.App.Exists("app.yao"); has {
		appFile = "app.yao"
		appData, err = application.App.Read("app.yao")
		if err != nil {
			t.Fatal(err)
		}

	} else if has, _ := application.App.Exists("app.jsonc"); has {
		appFile = "app.jsonc"
		appData, err = application.App.Read("app.jsonc")
		if err != nil {
			t.Fatal(err)
		}

	} else if has, _ := application.App.Exists("app.json"); has {
		appFile = "app.json"
		appData, err = application.App.Read("app.json")
		if err != nil {
			t.Fatal(err)
		}
	} else {
		t.Fatal(fmt.Errorf("app.yao or app.jsonc or app.json does not exists"))
	}

	// Replace $ENV with os.Getenv
	var envRe = regexp.MustCompile(`\$ENV\.([0-9a-zA-Z_-]+)`)
	appData = envRe.ReplaceAllFunc(appData, func(s []byte) []byte {
		key := string(s[5:])
		val := os.Getenv(key)
		if val == "" {
			return s
		}
		return []byte(val)
	})
	share.App = share.AppInfo{}
	err = application.Parse(appFile, appData, &share.App)
	if err != nil {
		t.Fatal(err)
	}

	// Set default prefix
	if share.App.Prefix == "" {
		share.App.Prefix = "yao_"
	}

	utils.Init()
	dbconnect(t, cfg)
	load(t, cfg)
	startRuntime(t, cfg)

}

// Clean the test environment
func Clean() {
	dbclose()
	runtime.Stop()

	// Remove the data store
	var path = filepath.Join(os.Getenv("YAO_TEST_APPLICATION"), "data", "stores")
	os.RemoveAll(path)
}

// Start the test server
func Start(t *testing.T, guards map[string]gin.HandlerFunc, cfg config.Config) {

	var err error
	option := http.Option{Port: 0, Root: "/", Timeout: 2 * time.Second}
	gin.SetMode(gin.ReleaseMode)

	router := gin.New()
	api.SetGuards(guards)
	api.SetRoutes(router, "api")

	testServer = http.New(router, option)
	go func() { err = testServer.Start() }()

	<-testServer.Event()
	if err != nil {
		t.Fatal(err)
	}
}

// Stop the test server
func Stop() {
	if testServer != nil {
		testServer.Stop()
		<-testServer.Event()
	}

	dbclose()
	runtime.Stop()
}

// Port Get the test server port
func Port(t *testing.T) int {
	if testServer == nil {
		t.Fatal(fmt.Errorf("server not started"))
	}
	port, err := testServer.Port()
	if err != nil {
		t.Fatal(err)
	}
	return port
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

func startRuntime(t *testing.T, cfg config.Config) {
	err := runtime.Start(cfg)
	if err != nil {
		t.Fatal(err)
	}
}

func load(t *testing.T, cfg config.Config) {
	loadFS(t, cfg)
	loadStore(t, cfg)
	loadScript(t, cfg)
	loadModel(t, cfg)
	loadConnector(t, cfg)
	loadQuery(t, cfg)
}

func loadFS(t *testing.T, cfg config.Config) {
	err := fs.Load(cfg)
	if err != nil {
		t.Fatal(err)
	}
}

func loadConnector(t *testing.T, cfg config.Config) {
	exts := []string{"*.yao", "*.json", "*.jsonc"}
	application.App.Walk("connectors", func(root, file string, isdir bool) error {
		if isdir {
			return nil
		}
		_, err := connector.Load(file, share.ID(root, file))
		return err
	}, exts...)
}

func loadScript(t *testing.T, cfg config.Config) {
	exts := []string{"*.js", "*.ts"}
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

	// Load system models
	err := loadSystemModels(t, cfg)
	if err != nil {
		t.Fatal(err)
	}

	exts := []string{"*.mod.yao", "*.mod.json", "*.mod.jsonc"}
	err = application.App.Walk("models", func(root, file string, isdir bool) error {
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

// loadStore load system stores for testing
func loadStore(t *testing.T, cfg config.Config) {
	err := loadSystemStores(t, cfg)
	if err != nil {
		t.Fatal(err)
	}

	exts := []string{"*.yao", "*.json", "*.jsonc"}
	err = application.App.Walk("stores", func(root, file string, isdir bool) error {
		if isdir {
			return nil
		}
		_, err := store.Load(file, share.ID(root, file))
		return err
	}, exts...)
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

// GuardBearerJWT test guard
func GuardBearerJWT(c *gin.Context) {
	tokenString := c.Request.Header.Get("Authorization")
	tokenString = strings.TrimSpace(strings.TrimPrefix(tokenString, "Bearer "))

	if tokenString == "" {
		c.JSON(403, gin.H{"code": 403, "message": "No permission"})
		c.Abort()
		return
	}

	claims := helper.JwtValidate(tokenString)
	c.Set("__sid", claims.SID)
}
