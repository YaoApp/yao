// Package test provides comprehensive testing utilities for the Yao application framework.
//
// AI ASSISTANT INSTRUCTIONS:
// All tests in the Yao ecosystem MUST use these utility functions for proper test environment setup.
// These are preparation and cleanup utility functions, NOT actual test cases.
//
// TESTING GUIDELINES FOR AI ASSISTANTS:
// 1. DO NOT modify configuration files (app.yao, models/*.mod.yao, etc.) to make tests pass
// 2. DO NOT bypass validation or security checks to make tests pass
// 3. If tests fail, investigate the root cause - it may be a real program bug that needs fixing
// 4. Tests should verify actual functionality, not just pass assertions
// 5. Use realistic test data that represents real-world usage scenarios
// 6. When tests fail, check: environment setup, missing dependencies, configuration issues, actual code bugs
// 7. Fix the underlying issue in the code, not the test or configuration
// 8. ALWAYS use test.Prepare() and test.Clean() for consistent test environment
//
// STANDARD USAGE PATTERN FOR ALL YAO TESTS:
//
//	func TestYourFunction(t *testing.T) {
//	    // Step 1: Prepare test environment
//	    test.Prepare(t, config.Conf)
//	    defer test.Clean()
//
//	    // Step 2: Your actual test code here...
//	    // The test environment will have:
//	    // - Database connections established
//	    // - All models migrated and ready
//	    // - Scripts, connectors, stores loaded
//	    // - Messenger providers configured
//	    // - File systems mounted
//	    // - V8 runtime started
//	}
//
// ADVANCED USAGE WITH HTTP SERVER:
//
//	func TestAPIEndpoint(t *testing.T) {
//	    test.Prepare(t, config.Conf)
//	    defer test.Stop() // Use Stop() instead of Clean() for server tests
//
//	    // Start HTTP server for API testing
//	    test.Start(t, map[string]gin.HandlerFunc{
//	        "bearer-jwt": test.GuardBearerJWT,
//	    }, config.Conf)
//
//	    port := test.Port(t)
//	    // Make HTTP requests to http://localhost:{port}/api/...
//	}
//
// PREREQUISITES:
// Before running any tests, you MUST execute in your terminal:
//
//	source $YAO_SOURCE_ROOT/env.local.sh
//
// This loads required environment variables including:
// - YAO_TEST_APPLICATION: Path to test application directory
// - Database connection parameters
// - Other configuration needed for testing
//
// WHAT test.Prepare() DOES:
// 1. Loads application from YAO_TEST_APPLICATION directory
// 2. Parses app.yao/app.json configuration with environment variable substitution
// 3. Establishes database connections (SQLite3 or MySQL based on config)
// 4. Loads and migrates all system models (users, roles, attachments, etc.)
// 5. Loads file systems, stores, connectors, scripts
// 6. Loads messenger providers and validates configurations
// 7. Starts V8 JavaScript runtime
// 8. Registers query engines for database operations
// 9. Creates temporary data directories for test isolation
//
// WHAT test.Clean() DOES:
// 1. Stops V8 runtime and releases resources
// 2. Closes all database connections
// 3. Removes temporary test data stores
// 4. Resets global state to prevent test interference
//
// WHAT test.Start() DOES:
// 1. Creates Gin HTTP server with API routes
// 2. Applies authentication guards (optional)
// 3. Starts server on random available port
// 4. Returns immediately, server runs in background
//
// WHAT test.Stop() DOES:
// 1. Gracefully shuts down HTTP server
// 2. Performs same cleanup as test.Clean()
//
// TESTING DIFFERENT MODULES:
//
// For Model Testing:
//
//	func TestUserModel(t *testing.T) {
//	    test.Prepare(t, config.Conf)
//	    defer test.Clean()
//
//	    // Models are auto-migrated and ready to use
//	    user := model.New("user")
//	    id, err := user.Create(map[string]interface{}{
//	        "name": "Test User",
//	        "email": "test@example.com",
//	    })
//	    // ... test model operations
//	}
//
// For Script Testing:
//
//	func TestJavaScript(t *testing.T) {
//	    test.Prepare(t, config.Conf)
//	    defer test.Clean()
//
//	    // Scripts are loaded and V8 runtime is ready
//	    result, err := process.New("scripts.myfunction").Exec()
//	    // ... test script execution
//	}
//
// For Connector Testing:
//
//	func TestDatabaseConnector(t *testing.T) {
//	    test.Prepare(t, config.Conf)
//	    defer test.Clean()
//
//	    // Connectors are loaded and ready
//	    conn := connector.Select("mysql")
//	    // ... test connector operations
//	}
//
// For Messenger Testing:
//
//	func TestEmailSending(t *testing.T) {
//	    test.Prepare(t, config.Conf)
//	    defer test.Clean()
//
//	    // Messenger providers are loaded and validated
//	    // Test messenger functionality here
//	    // Note: Actual messenger service creation is handled by the messenger package
//	}
//
// ERROR HANDLING:
// If any step in test.Prepare() fails, the test will fail immediately with a descriptive error.
// This ensures tests only run in a properly configured environment.
//
// TEST ISOLATION:
// Each test gets:
// - Fresh database connections
// - Isolated temporary directories
// - Clean global state
// - Independent data stores
//
// PERFORMANCE CONSIDERATIONS:
// - test.Prepare() is relatively expensive (database setup, migrations, etc.)
// - Consider using subtests or table-driven tests to amortize setup costs
// - For integration tests, prefer fewer, more comprehensive tests over many small ones
//
// DEBUGGING FAILED TESTS:
// 1. Check environment variables are set correctly
// 2. Verify test application directory exists and is readable
// 3. Check database connectivity and permissions
// 4. Look for configuration file syntax errors
// 5. Examine log output for detailed error messages
// 6. Ensure all required dependencies are available
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
	"github.com/yaoapp/gou/mcp"
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
	"__yao.agent.assistant":    "yao/models/agent/assistant.mod.yao",
	"__yao.agent.chat":         "yao/models/agent/chat.mod.yao",
	"__yao.agent.message":      "yao/models/agent/message.mod.yao",
	"__yao.agent.resume":       "yao/models/agent/resume.mod.yao",
	"__yao.agent.search":       "yao/models/agent/search.mod.yao",
	"__yao.attachment":         "yao/models/attachment.mod.yao",
	"__yao.audit":              "yao/models/audit.mod.yao",
	"__yao.config":             "yao/models/config.mod.yao",
	"__yao.dsl":                "yao/models/dsl.mod.yao",
	"__yao.invitation":         "yao/models/invitation.mod.yao",
	"__yao.job.category":       "yao/models/job/category.mod.yao",
	"__yao.job":                "yao/models/job/job.mod.yao",
	"__yao.job.execution":      "yao/models/job/execution.mod.yao",
	"__yao.job.log":            "yao/models/job/log.mod.yao",
	"__yao.kb.collection":      "yao/models/kb/collection.mod.yao",
	"__yao.kb.document":        "yao/models/kb/document.mod.yao",
	"__yao.team":               "yao/models/team.mod.yao",
	"__yao.member":             "yao/models/member.mod.yao",
	"__yao.user":               "yao/models/user.mod.yao",
	"__yao.role":               "yao/models/role.mod.yao",
	"__yao.user.type":          "yao/models/user/type.mod.yao",
	"__yao.user.oauth_account": "yao/models/user/oauth_account.mod.yao",
}

var testSystemStores = map[string]string{
	"__yao.store":                "yao/stores/store.xun.yao",
	"__yao.cache":                "yao/stores/cache.lru.yao",
	"__yao.oauth.store":          "yao/stores/oauth/store.xun.yao",
	"__yao.oauth.client":         "yao/stores/oauth/client.xun.yao",
	"__yao.oauth.cache":          "yao/stores/oauth/cache.lru.yao",
	"__yao.agent.memory.user":    "yao/stores/agent/memory/user.xun.yao",
	"__yao.agent.memory.team":    "yao/stores/agent/memory/team.xun.yao",
	"__yao.agent.memory.chat":    "yao/stores/agent/memory/chat.xun.yao",
	"__yao.agent.memory.context": "yao/stores/agent/memory/context.xun.yao",
	"__yao.agent.cache":          "yao/stores/agent/cache.lru.yao",
	"__yao.kb.store":             "yao/stores/kb/store.xun.yao",
	"__yao.kb.cache":             "yao/stores/kb/cache.lru.yao",
}

func loadSystemStores(t *testing.T, cfg config.Config) error {
	for id, path := range testSystemStores {
		// Check if store already exists, skip if already loaded
		if _, err := store.Get(id); err == nil {
			continue
		}

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
		// Check if model already exists, skip if already loaded
		if _, exists := model.Models[id]; exists {
			continue
		}

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

// PrepareOption options for test preparation
type PrepareOption struct {
	// V8Mode sets the V8 runtime mode: "standard" (default) or "performance"
	// - standard: Lower memory usage, creates/disposes isolates for each execution
	// - performance: Higher memory usage, maintains isolate pool for better performance
	// Use "performance" mode for benchmarks and stress tests
	V8Mode string
}

// Prepare test environment with optional configuration
// Usage:
//
//	test.Prepare(t, config.Conf)                                    // standard mode (default)
//	test.Prepare(t, config.Conf, test.PrepareOption{V8Mode: "performance"}) // performance mode
func Prepare(t *testing.T, cfg config.Config, opts ...interface{}) {

	appRootEnv := "YAO_TEST_APPLICATION"
	v8Mode := "standard" // default to standard mode

	// Parse options
	for _, opt := range opts {
		switch v := opt.(type) {
		case string:
			// Legacy: string parameter for appRootEnv
			appRootEnv = v
		case PrepareOption:
			// New: structured options
			if v.V8Mode != "" {
				v8Mode = v.V8Mode
			}
		}
	}

	// Override with environment variable if set
	if envMode := os.Getenv("YAO_RUNTIME_MODE"); envMode != "" {
		v8Mode = envMode
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

	// Apply V8 mode to config
	cfg.Runtime.Mode = v8Mode

	// Ensure MinSize and MaxSize are set for performance mode
	if v8Mode == "performance" {
		if cfg.Runtime.MinSize == 0 {
			cfg.Runtime.MinSize = 3
		}
		if cfg.Runtime.MaxSize == 0 {
			cfg.Runtime.MaxSize = 10
		}
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
	default:
		capsule.AddConn("primary", "mysql", cfg.DB.Primary[0]).SetAsGlobal()
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
	loadMCP(t, cfg)
	loadMessenger(t, cfg)
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

func loadMCP(t *testing.T, cfg config.Config) {
	// Check if mcps directory exists
	exists, err := application.App.Exists("mcps")
	if err != nil || !exists {
		return
	}

	exts := []string{"*.mcp.yao", "*.mcp.json", "*.mcp.jsonc"}
	err = application.App.Walk("mcps", func(root, file string, isdir bool) error {
		if isdir {
			return nil
		}
		_, err := mcp.LoadClient(file, share.ID(root, file))
		return err
	}, exts...)

	if err != nil {
		t.Fatal(err)
	}
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

// loadMessenger validates messenger configurations for testing without creating circular imports.
//
// AI ASSISTANT INSTRUCTIONS:
// This function is called automatically by test.Prepare() and should NOT be called directly.
// It validates messenger provider configurations to ensure they are syntactically correct.
//
// WHAT THIS FUNCTION DOES:
// 1. Checks if messengers/ directory exists (optional, skips if not found)
// 2. Validates messengers/providers/ directory and all provider files
// 3. Parses each provider configuration file to ensure valid JSON/YAML syntax
// 4. Does NOT create actual messenger service instances (avoids circular imports)
// 5. Allows messenger package tests to use test.Prepare() safely
//
// CIRCULAR IMPORT PREVENTION:
// This function intentionally does NOT import the messenger package or create messenger instances.
// Instead, it only validates that configuration files are parseable.
// The actual messenger service creation is handled by the messenger package itself.
//
// SUPPORTED PROVIDER FILE FORMATS:
// - *.yao (YAML with .yao extension)
// - *.json (Standard JSON)
// - *.jsonc (JSON with comments)
//
// VALIDATION PERFORMED:
// - File readability and accessibility
// - JSON/YAML syntax validation
// - Basic structure verification
// - Environment variable substitution compatibility
//
// ERROR HANDLING:
// If any provider file cannot be read or parsed, the test fails immediately.
// This ensures messenger configurations are valid before tests run.
func loadMessenger(t *testing.T, cfg config.Config) {
	// Check if messengers directory exists
	exists, err := application.App.Exists("messengers")
	if err != nil {
		t.Fatal(err)
	}
	if !exists {
		// Skip loading messenger if directory doesn't exist
		// This is normal for applications that don't use messaging features
		return
	}

	// For testing purposes, we just need to ensure the messenger directory
	// and provider files exist and can be parsed. We don't need to create
	// the full messenger service instance since that would require importing
	// the messenger package (which would cause circular imports).

	// Load provider configurations for validation
	providersPath := "messengers/providers"
	providerExists, err := application.App.Exists(providersPath)
	if err != nil {
		t.Fatal(err)
	}
	if !providerExists {
		// No providers directory is acceptable - messenger might not be configured
		return
	}

	// Walk through provider files to validate they can be parsed
	exts := []string{"*.yao", "*.json", "*.jsonc"}
	err = application.App.Walk(providersPath, func(root, file string, isdir bool) error {
		if isdir {
			return nil
		}

		raw, err := application.App.Read(file)
		if err != nil {
			return fmt.Errorf("failed to read messenger provider %s: %w", file, err)
		}

		// Try to parse the provider config to ensure it's valid
		var config map[string]interface{}
		err = application.Parse(file, raw, &config)
		if err != nil {
			return fmt.Errorf("failed to parse messenger provider %s: %w", file, err)
		}

		// Basic validation - ensure required fields are present
		if config["connector"] == nil {
			return fmt.Errorf("messenger provider %s missing required 'connector' field", file)
		}

		return nil
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

// LoadAgentTestScripts loads all *_test.ts/js scripts from an agent's src directory.
// This is useful for testing agent hooks (before/after scripts) and other agent-specific test scripts.
//
// Usage:
//
//	test.Prepare(t, config.Conf)
//	defer test.Clean()
//	scripts := test.LoadAgentTestScripts(t, "assistants/tests/hooks-test")
//
// Parameters:
//   - t: testing.T instance
//   - agentRelPath: relative path to agent directory from app root (e.g., "assistants/tests/hooks-test")
//
// Returns:
//   - []string: list of loaded script IDs (e.g., ["hook.env_test"])
func LoadAgentTestScripts(t *testing.T, agentRelPath string) []string {
	srcDir := filepath.Join(agentRelPath, "src")

	// Check if src directory exists
	exists, err := application.App.Exists(srcDir)
	if err != nil {
		t.Fatalf("Failed to check src directory: %v", err)
	}
	if !exists {
		t.Logf("No src directory found at %s, skipping", srcDir)
		return nil
	}

	var loadedScripts []string
	exts := []string{"*_test.ts", "*_test.js"}

	err = application.App.Walk(srcDir, func(root, file string, isdir bool) error {
		if isdir {
			return nil
		}

		// Only load *_test.ts/js files
		base := filepath.Base(file)
		if !strings.HasSuffix(base, "_test.ts") && !strings.HasSuffix(base, "_test.js") {
			return nil
		}

		// Generate script ID: hook.{relative_path_without_ext}
		// e.g., assistants/tests/hooks-test/src/env_test.ts -> hook.env_test
		relPath := strings.TrimPrefix(file, srcDir+"/")
		relPath = strings.TrimPrefix(relPath, "/")
		relPath = strings.TrimSuffix(relPath, filepath.Ext(relPath))
		scriptID := "hook." + strings.ReplaceAll(relPath, "/", ".")

		// Load the script
		_, err := v8.Load(file, scriptID)
		if err != nil {
			t.Logf("Warning: Failed to load hook script %s: %v", base, err)
			return nil // Continue loading other scripts
		}

		loadedScripts = append(loadedScripts, scriptID)
		return nil
	}, exts...)

	if err != nil {
		t.Fatalf("Failed to walk src directory: %v", err)
	}

	return loadedScripts
}
