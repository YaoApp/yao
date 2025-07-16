package mcp

import (
	"fmt"
	"os"
	"testing"
	"time"

	jsoniter "github.com/json-iterator/go"
	"github.com/yaoapp/gou/application"
	"github.com/yaoapp/gou/model"
	"github.com/yaoapp/kun/log"
	"github.com/yaoapp/yao/config"
	"github.com/yaoapp/yao/data"
	"github.com/yaoapp/yao/dsl/types"
	"github.com/yaoapp/yao/test"
)

// systemModels system models
var systemModels = map[string]string{
	"__yao.dsl": "yao/models/dsl.mod.yao",
}

func TestMain(m *testing.M) {
	// Setup
	test.Prepare(&testing.T{}, config.Conf)
	defer test.Clean()

	// Load system models
	model.WithCrypt([]byte(fmt.Sprintf(`{"key":"%s"}`, config.Conf.DB.AESKey)), "AES")
	model.WithCrypt([]byte(`{}`), "PASSWORD")
	err := loadSystemModels()
	if err != nil {
		log.Error("Load system models error: %s", err.Error())
		os.Exit(1)
	}

	// Load application
	root := os.Getenv("GOU_TEST_APPLICATION")
	app, err := application.OpenFromDisk(root) // Load app
	if err != nil {
		log.Error("Load application error: %s", err.Error())
		os.Exit(1)
	}
	application.Load(app)

	// Run tests
	code := m.Run()
	os.Exit(code)
}

// loadSystemModels load system models
func loadSystemModels() error {
	for id, path := range systemModels {
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
				table["name"] = "__yao_" + name
				content, err = jsoniter.Marshal(data)
				if err != nil {
					log.Error("failed to marshal model data: %v", err)
					return fmt.Errorf("failed to marshal model data: %v", err)
				}
			}
		}

		// Load Model
		mod, err := model.LoadSource(content, id, path)
		if err != nil {
			log.Error("load system model %s error: %s", id, err.Error())
			return err
		}

		// Drop table first
		err = mod.DropTable()
		if err != nil {
			log.Error("drop table error: %s", err.Error())
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

// TestCase defines a single test case
type TestCase struct {
	ID            string
	Source        string
	UpdatedSource string
	Tags          []string
	Label         string
	Description   string
}

// NewTestCase creates a new test case
func NewTestCase() *TestCase {
	id := getTestID()
	return &TestCase{
		ID: id,
		Source: fmt.Sprintf(`{
  "name": "Test MCP Client %s",
  "label": "Test MCP Client",
  "description": "Test MCP Client Description",
  "tags": ["test_%s"],
  "transport": "stdio",
  "command": "echo",
  "arguments": ["hello", "world"],
  "env": {
    "MCP_TEST": "true"
  },
  "enable_sampling": true,
  "enable_roots": false,
  "timeout": "30s"
}`, id, id),
		UpdatedSource: fmt.Sprintf(`{
  "name": "Updated MCP Client %s",
  "label": "Updated MCP Client",
  "description": "Updated MCP Client Description",
  "tags": ["test_%s", "updated"],
  "transport": "stdio",
  "command": "echo",
  "arguments": ["hello", "updated"],
  "env": {
    "MCP_TEST": "true",
    "MCP_UPDATED": "true"
  },
  "enable_sampling": false,
  "enable_roots": true,
  "timeout": "60s"
}`, id, id),
		Tags:        []string{fmt.Sprintf("test_%s", id)},
		Label:       "Test MCP Client",
		Description: "Test MCP Client Description",
	}
}

// NewHTTPTestCase creates a new HTTP test case
func NewHTTPTestCase() *TestCase {
	id := getTestID()
	return &TestCase{
		ID: id,
		Source: fmt.Sprintf(`{
  "name": "Test HTTP MCP Client %s",
  "label": "Test HTTP MCP Client",
  "description": "Test HTTP MCP Client Description",
  "tags": ["test_%s", "http"],
  "transport": "http",
  "url": "http://localhost:8080/mcp",
  "authorization_token": "Bearer test-token",
  "enable_sampling": true,
  "enable_roots": true,
  "timeout": "30s"
}`, id, id),
		UpdatedSource: fmt.Sprintf(`{
  "name": "Updated HTTP MCP Client %s",
  "label": "Updated HTTP MCP Client",
  "description": "Updated HTTP MCP Client Description",
  "tags": ["test_%s", "http", "updated"],
  "transport": "http",
  "url": "http://localhost:8080/mcp/v2",
  "authorization_token": "Bearer updated-token",
  "enable_sampling": false,
  "enable_roots": false,
  "timeout": "60s"
}`, id, id),
		Tags:        []string{fmt.Sprintf("test_%s", id), "http"},
		Label:       "Test HTTP MCP Client",
		Description: "Test HTTP MCP Client Description",
	}
}

// NewSSETestCase creates a new SSE test case
func NewSSETestCase() *TestCase {
	id := getTestID()
	return &TestCase{
		ID: id,
		Source: fmt.Sprintf(`{
  "name": "Test SSE MCP Client %s",
  "label": "Test SSE MCP Client",
  "description": "Test SSE MCP Client Description",
  "tags": ["test_%s", "sse"],
  "transport": "sse",
  "url": "http://localhost:8080/sse",
  "authorization_token": "Bearer sse-token",
  "enable_sampling": true,
  "enable_elicitation": true,
  "timeout": "45s"
}`, id, id),
		UpdatedSource: fmt.Sprintf(`{
  "name": "Updated SSE MCP Client %s",
  "label": "Updated SSE MCP Client",
  "description": "Updated SSE MCP Client Description",
  "tags": ["test_%s", "sse", "updated"],
  "transport": "sse",
  "url": "http://localhost:8080/sse/v2",
  "authorization_token": "Bearer updated-sse-token",
  "enable_sampling": false,
  "enable_elicitation": false,
  "timeout": "90s"
}`, id, id),
		Tags:        []string{fmt.Sprintf("test_%s", id), "sse"},
		Label:       "Test SSE MCP Client",
		Description: "Test SSE MCP Client Description",
	}
}

// getTestID generates a unique test ID
func getTestID() string {
	return fmt.Sprintf("test_%d", time.Now().UnixNano())
}

// CreateOptions returns creation options
func (tc *TestCase) CreateOptions() *types.CreateOptions {
	return &types.CreateOptions{
		ID:     tc.ID,
		Source: tc.Source,
	}
}

// LoadOptions returns load options
func (tc *TestCase) LoadOptions() *types.LoadOptions {
	return &types.LoadOptions{
		ID:     tc.ID,
		Source: tc.Source,
	}
}

// UnloadOptions returns unload options
func (tc *TestCase) UnloadOptions() *types.UnloadOptions {
	return &types.UnloadOptions{
		ID: tc.ID,
	}
}

// ReloadOptions returns reload options
func (tc *TestCase) ReloadOptions() *types.ReloadOptions {
	return &types.ReloadOptions{
		ID:     tc.ID,
		Source: tc.UpdatedSource,
	}
}

// AssertInfo verifies if the information is correct
func (tc *TestCase) AssertInfo(info *types.Info) bool {
	if info == nil {
		return false
	}
	return info.ID == tc.ID &&
		info.Type == types.TypeMCPClient &&
		info.Label == tc.Label &&
		len(info.Tags) == len(tc.Tags) &&
		info.Description == tc.Description &&
		!info.Readonly &&
		!info.Builtin &&
		!info.Mtime.IsZero() &&
		!info.Ctime.IsZero()
}

// AssertUpdatedInfo verifies if the updated information is correct
func (tc *TestCase) AssertUpdatedInfo(info *types.Info) bool {
	if info == nil {
		return false
	}
	expectedTags := append(tc.Tags, "updated")
	return info.ID == tc.ID &&
		info.Type == types.TypeMCPClient &&
		info.Label == "Updated MCP Client" &&
		len(info.Tags) == len(expectedTags) &&
		info.Description == "Updated MCP Client Description" &&
		!info.Readonly &&
		!info.Builtin &&
		!info.Mtime.IsZero() &&
		!info.Ctime.IsZero()
}

// AssertHTTPInfo verifies if the HTTP client information is correct
func (tc *TestCase) AssertHTTPInfo(info *types.Info) bool {
	if info == nil {
		return false
	}
	return info.ID == tc.ID &&
		info.Type == types.TypeMCPClient &&
		info.Label == "Test HTTP MCP Client" &&
		len(info.Tags) == 2 && // test_xxx and http
		info.Description == "Test HTTP MCP Client Description" &&
		!info.Readonly &&
		!info.Builtin &&
		!info.Mtime.IsZero() &&
		!info.Ctime.IsZero()
}

// AssertSSEInfo verifies if the SSE client information is correct
func (tc *TestCase) AssertSSEInfo(info *types.Info) bool {
	if info == nil {
		return false
	}
	return info.ID == tc.ID &&
		info.Type == types.TypeMCPClient &&
		info.Label == "Test SSE MCP Client" &&
		len(info.Tags) == 2 && // test_xxx and sse
		info.Description == "Test SSE MCP Client Description" &&
		!info.Readonly &&
		!info.Builtin &&
		!info.Mtime.IsZero() &&
		!info.Ctime.IsZero()
}
