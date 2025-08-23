package dsl

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	jsoniter "github.com/json-iterator/go"
	"github.com/stretchr/testify/assert"
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
	root := os.Getenv("YAO_TEST_APPLICATION")
	if root == "" {
		log.Error("YAO_TEST_APPLICATION environment variable is not set")
		os.Exit(1)
	}
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

// cleanTestData cleans test data from database
func cleanTestData() error {
	m := model.Select("__yao.dsl")
	err := m.DropTable()
	if err != nil {
		return err
	}
	err = m.Migrate(false, model.WithDonotInsertValues(true))
	if err != nil {
		return err
	}
	return nil
}

// getTestID generates a unique test ID
func getTestID() string {
	return fmt.Sprintf("test_%d", time.Now().UnixNano())
}

// TestCase defines a unified test case for all DSL types
type TestCase struct {
	ID            string
	Source        string
	UpdatedSource string
	Tags          []string
	Label         string
	Description   string
	DSLType       types.Type
}

// NewModelTestCase creates a new model test case
func NewModelTestCase() *TestCase {
	id := getTestID()
	return &TestCase{
		ID:      id,
		DSLType: types.TypeModel,
		Source: fmt.Sprintf(`{
  "name": "%s",
  "table": { "name": "%s", "comment": "Test User" },
  "columns": [
    { "name": "id", "type": "ID" },
    { "name": "name", "type": "string", "length": 80, "comment": "User Name", "index": true },
    { "name": "status", "type": "enum", "option": ["active", "disabled"], "default": "active", "comment": "Status", "index": true }
  ],
  "tags": ["test_%s"],
  "label": "Test Model",
  "description": "Test Model Description",
  "option": { "timestamps": true, "soft_deletes": true }
}`, id, id, id),
		UpdatedSource: fmt.Sprintf(`{
  "name": "%s",
  "table": { "name": "%s", "comment": "Updated Test User" },
  "columns": [
    { "name": "id", "type": "ID" },
    { "name": "name", "type": "string", "length": 80, "comment": "User Name", "index": true },
    { "name": "status", "type": "enum", "option": ["active", "disabled", "pending"], "default": "active", "comment": "Status", "index": true }
  ],
  "tags": ["test_%s", "updated"],
  "label": "Updated Model",
  "description": "Updated Model Description",
  "option": { "timestamps": true, "soft_deletes": true }
}`, id, id, id),
		Tags:        []string{fmt.Sprintf("test_%s", id)},
		Label:       "Test Model",
		Description: "Test Model Description",
	}
}

// NewConnectorTestCase creates a new connector test case
func NewConnectorTestCase() *TestCase {
	id := getTestID()
	return &TestCase{
		ID:      id,
		DSLType: types.TypeConnector,
		Source: fmt.Sprintf(`{
  "label": "Test Connector",
  "description": "Test Connector Description",
  "tags": ["test_%s"],
  "type": "openai",
  "options": {
    "proxy": "https://api.openai.com/v1",
    "model": "gpt-4o-mini",
    "key": "sk-test-key"
  }
}`, id),
		UpdatedSource: fmt.Sprintf(`{
  "label": "Updated Connector",
  "description": "Updated Connector Description",
  "tags": ["test_%s", "updated"],
  "type": "openai",
  "options": {
    "proxy": "https://api.openai.com/v1",
    "model": "gpt-4o-mini",
    "key": "sk-test-key"
  }
}`, id),
		Tags:        []string{fmt.Sprintf("test_%s", id)},
		Label:       "Test Connector",
		Description: "Test Connector Description",
	}
}

// NewMCPTestCase creates a new MCP test case
func NewMCPTestCase() *TestCase {
	id := getTestID()
	return &TestCase{
		ID:      id,
		DSLType: types.TypeMCPClient,
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

// CreateOptions returns creation options
func (tc *TestCase) CreateOptions(store types.StoreType) *types.CreateOptions {
	return &types.CreateOptions{
		ID:     tc.ID,
		Source: tc.Source,
		Store:  store,
	}
}

// UpdateOptions returns update options
func (tc *TestCase) UpdateOptions() *types.UpdateOptions {
	return &types.UpdateOptions{
		ID:     tc.ID,
		Source: tc.UpdatedSource,
	}
}

// DeleteOptions returns delete options
func (tc *TestCase) DeleteOptions() *types.DeleteOptions {
	return &types.DeleteOptions{
		ID: tc.ID,
	}
}

// LoadOptions returns load options
func (tc *TestCase) LoadOptions(store types.StoreType) *types.LoadOptions {
	return &types.LoadOptions{
		ID:     tc.ID,
		Source: tc.Source,
		Store:  store,
	}
}

// UnloadOptions returns unload options
func (tc *TestCase) UnloadOptions(store types.StoreType) *types.UnloadOptions {
	return &types.UnloadOptions{
		ID:    tc.ID,
		Store: store,
	}
}

// ReloadOptions returns reload options
func (tc *TestCase) ReloadOptions(store types.StoreType) *types.ReloadOptions {
	return &types.ReloadOptions{
		ID:     tc.ID,
		Source: tc.UpdatedSource,
		Store:  store,
	}
}

// ListOptions returns list options
func (tc *TestCase) ListOptions(store types.StoreType) *types.ListOptions {
	return &types.ListOptions{
		Tags:  tc.Tags,
		Store: store,
	}
}

// AssertInfo verifies if the information is correct
func (tc *TestCase) AssertInfo(info *types.Info) bool {
	if info == nil {
		return false
	}

	return info.ID == tc.ID &&
		info.Type == tc.DSLType &&
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
	expectedLabel := ""
	switch tc.DSLType {
	case types.TypeModel:
		expectedLabel = "Updated Model"
	case types.TypeConnector:
		expectedLabel = "Updated Connector"
	case types.TypeMCPClient:
		expectedLabel = "Updated MCP Client"
	}
	expectedDescription := ""
	switch tc.DSLType {
	case types.TypeModel:
		expectedDescription = "Updated Model Description"
	case types.TypeConnector:
		expectedDescription = "Updated Connector Description"
	case types.TypeMCPClient:
		expectedDescription = "Updated MCP Client Description"
	}
	return info.ID == tc.ID &&
		info.Type == tc.DSLType &&
		info.Label == expectedLabel &&
		len(info.Tags) == 2 &&
		info.Description == expectedDescription &&
		!info.Readonly &&
		!info.Builtin &&
		!info.Mtime.IsZero() &&
		!info.Ctime.IsZero()
}

// Test DSL creation with different types and stores
func TestDSLCreate(t *testing.T) {
	ctx := context.Background()

	testCases := []struct {
		name    string
		tcFunc  func() *TestCase
		dslType types.Type
		stores  []types.StoreType
	}{
		{"Model", NewModelTestCase, types.TypeModel, []types.StoreType{types.StoreTypeDB, types.StoreTypeFile}},
		{"Connector", NewConnectorTestCase, types.TypeConnector, []types.StoreType{types.StoreTypeDB, types.StoreTypeFile}},
		{"MCP", NewMCPTestCase, types.TypeMCPClient, []types.StoreType{types.StoreTypeDB, types.StoreTypeFile}},
	}

	for _, tt := range testCases {
		for _, store := range tt.stores {
			t.Run(fmt.Sprintf("%s_%s", tt.name, store), func(t *testing.T) {
				// Clean test data before each test
				err := cleanTestData()
				if err != nil {
					t.Fatalf("Failed to clean test data: %v", err)
				}

				dsl, err := New(tt.dslType)
				if !assert.Nil(t, err) {
					return
				}

				tc := tt.tcFunc()

				// Create
				err = dsl.Create(ctx, tc.CreateOptions(store))
				if !assert.Nil(t, err) {
					return
				}

				// Verify exists
				exists, err := dsl.Exists(ctx, tc.ID)
				if !assert.Nil(t, err) {
					return
				}
				assert.True(t, exists)

				// Verify info
				info, err := dsl.Inspect(ctx, tc.ID)
				if !assert.Nil(t, err) {
					return
				}
				assert.True(t, tc.AssertInfo(info))

				// Cleanup
				err = dsl.Delete(ctx, tc.DeleteOptions())
				assert.Nil(t, err)
			})
		}
	}
}

// Test DSL inspection
func TestDSLInspect(t *testing.T) {
	ctx := context.Background()

	testCases := []struct {
		name    string
		tcFunc  func() *TestCase
		dslType types.Type
		stores  []types.StoreType
	}{
		{"Model", NewModelTestCase, types.TypeModel, []types.StoreType{types.StoreTypeDB, types.StoreTypeFile}},
		{"Connector", NewConnectorTestCase, types.TypeConnector, []types.StoreType{types.StoreTypeDB, types.StoreTypeFile}},
		{"MCP", NewMCPTestCase, types.TypeMCPClient, []types.StoreType{types.StoreTypeDB, types.StoreTypeFile}},
	}

	for _, tt := range testCases {
		for _, store := range tt.stores {
			t.Run(fmt.Sprintf("%s_%s", tt.name, store), func(t *testing.T) {
				// Clean test data before each test
				err := cleanTestData()
				if err != nil {
					t.Fatalf("Failed to clean test data: %v", err)
				}

				dsl, err := New(tt.dslType)
				if !assert.Nil(t, err) {
					return
				}

				tc := tt.tcFunc()

				// Create
				err = dsl.Create(ctx, tc.CreateOptions(store))
				if !assert.Nil(t, err) {
					return
				}

				// Inspect
				info, err := dsl.Inspect(ctx, tc.ID)
				if !assert.Nil(t, err) {
					return
				}
				assert.True(t, tc.AssertInfo(info))

				// Cleanup
				err = dsl.Delete(ctx, tc.DeleteOptions())
				assert.Nil(t, err)
			})
		}
	}
}

// Test DSL source retrieval
func TestDSLSource(t *testing.T) {
	ctx := context.Background()

	testCases := []struct {
		name    string
		tcFunc  func() *TestCase
		dslType types.Type
		stores  []types.StoreType
	}{
		{"Model", NewModelTestCase, types.TypeModel, []types.StoreType{types.StoreTypeDB, types.StoreTypeFile}},
		{"Connector", NewConnectorTestCase, types.TypeConnector, []types.StoreType{types.StoreTypeDB, types.StoreTypeFile}},
		{"MCP", NewMCPTestCase, types.TypeMCPClient, []types.StoreType{types.StoreTypeDB, types.StoreTypeFile}},
	}

	for _, tt := range testCases {
		for _, store := range tt.stores {
			t.Run(fmt.Sprintf("%s_%s", tt.name, store), func(t *testing.T) {
				// Clean test data before each test
				err := cleanTestData()
				if err != nil {
					t.Fatalf("Failed to clean test data: %v", err)
				}

				dsl, err := New(tt.dslType)
				if !assert.Nil(t, err) {
					return
				}

				tc := tt.tcFunc()

				// Create
				err = dsl.Create(ctx, tc.CreateOptions(store))
				if !assert.Nil(t, err) {
					return
				}

				// Get source
				source, err := dsl.Source(ctx, tc.ID)
				if !assert.Nil(t, err) {
					return
				}
				assert.Equal(t, tc.Source, source)

				// Cleanup
				err = dsl.Delete(ctx, tc.DeleteOptions())
				assert.Nil(t, err)
			})
		}
	}
}

// Test DSL listing
func TestDSLList(t *testing.T) {
	ctx := context.Background()

	testCases := []struct {
		name    string
		tcFunc  func() *TestCase
		dslType types.Type
		stores  []types.StoreType
	}{
		{"Model", NewModelTestCase, types.TypeModel, []types.StoreType{types.StoreTypeDB, types.StoreTypeFile}},
		{"Connector", NewConnectorTestCase, types.TypeConnector, []types.StoreType{types.StoreTypeDB, types.StoreTypeFile}},
		{"MCP", NewMCPTestCase, types.TypeMCPClient, []types.StoreType{types.StoreTypeDB, types.StoreTypeFile}},
	}

	for _, tt := range testCases {
		for _, store := range tt.stores {
			t.Run(fmt.Sprintf("%s_%s", tt.name, store), func(t *testing.T) {
				// Clean test data before each test
				err := cleanTestData()
				if err != nil {
					t.Fatalf("Failed to clean test data: %v", err)
				}

				dsl, err := New(tt.dslType)
				if !assert.Nil(t, err) {
					return
				}

				tc1 := tt.tcFunc()
				tc2 := tt.tcFunc()

				// Create test cases
				err = dsl.Create(ctx, tc1.CreateOptions(store))
				if !assert.Nil(t, err) {
					return
				}
				err = dsl.Create(ctx, tc2.CreateOptions(store))
				if !assert.Nil(t, err) {
					return
				}

				// List all
				list, err := dsl.List(ctx, &types.ListOptions{Store: store})
				if !assert.Nil(t, err) {
					return
				}
				assert.GreaterOrEqual(t, len(list), 2)

				// List with tags
				list, err = dsl.List(ctx, tc1.ListOptions(store))
				if !assert.Nil(t, err) {
					return
				}
				assert.GreaterOrEqual(t, len(list), 1)

				// Cleanup
				err = dsl.Delete(ctx, tc1.DeleteOptions())
				assert.Nil(t, err)
				err = dsl.Delete(ctx, tc2.DeleteOptions())
				assert.Nil(t, err)
			})
		}
	}
}

// Test DSL update
func TestDSLUpdate(t *testing.T) {
	ctx := context.Background()

	testCases := []struct {
		name    string
		tcFunc  func() *TestCase
		dslType types.Type
		stores  []types.StoreType
	}{
		{"Model", NewModelTestCase, types.TypeModel, []types.StoreType{types.StoreTypeDB, types.StoreTypeFile}},
		{"Connector", NewConnectorTestCase, types.TypeConnector, []types.StoreType{types.StoreTypeDB, types.StoreTypeFile}},
		{"MCP", NewMCPTestCase, types.TypeMCPClient, []types.StoreType{types.StoreTypeDB, types.StoreTypeFile}},
	}

	for _, tt := range testCases {
		for _, store := range tt.stores {
			t.Run(fmt.Sprintf("%s_%s", tt.name, store), func(t *testing.T) {
				// Clean test data before each test
				err := cleanTestData()
				if err != nil {
					t.Fatalf("Failed to clean test data: %v", err)
				}

				dsl, err := New(tt.dslType)
				if !assert.Nil(t, err) {
					return
				}

				tc := tt.tcFunc()

				// Create
				err = dsl.Create(ctx, tc.CreateOptions(store))
				if !assert.Nil(t, err) {
					return
				}

				// Update
				err = dsl.Update(ctx, tc.UpdateOptions())
				if !assert.Nil(t, err) {
					return
				}

				// Verify updated info
				info, err := dsl.Inspect(ctx, tc.ID)
				if !assert.Nil(t, err) {
					return
				}
				assert.True(t, tc.AssertUpdatedInfo(info))

				// Cleanup
				err = dsl.Delete(ctx, tc.DeleteOptions())
				assert.Nil(t, err)
			})
		}
	}
}

// Test DSL delete
func TestDSLDelete(t *testing.T) {
	ctx := context.Background()

	testCases := []struct {
		name    string
		tcFunc  func() *TestCase
		dslType types.Type
		stores  []types.StoreType
	}{
		{"Model", NewModelTestCase, types.TypeModel, []types.StoreType{types.StoreTypeDB, types.StoreTypeFile}},
		{"Connector", NewConnectorTestCase, types.TypeConnector, []types.StoreType{types.StoreTypeDB, types.StoreTypeFile}},
		{"MCP", NewMCPTestCase, types.TypeMCPClient, []types.StoreType{types.StoreTypeDB, types.StoreTypeFile}},
	}

	for _, tt := range testCases {
		for _, store := range tt.stores {
			t.Run(fmt.Sprintf("%s_%s", tt.name, store), func(t *testing.T) {
				// Clean test data before each test
				err := cleanTestData()
				if err != nil {
					t.Fatalf("Failed to clean test data: %v", err)
				}

				dsl, err := New(tt.dslType)
				if !assert.Nil(t, err) {
					return
				}

				tc := tt.tcFunc()

				// Create
				err = dsl.Create(ctx, tc.CreateOptions(store))
				if !assert.Nil(t, err) {
					return
				}

				// Delete
				err = dsl.Delete(ctx, tc.DeleteOptions())
				if !assert.Nil(t, err) {
					return
				}

				// Verify deleted
				exists, err := dsl.Exists(ctx, tc.ID)
				if !assert.Nil(t, err) {
					return
				}
				assert.False(t, exists)
			})
		}
	}
}

// Test DSL full flow (create, inspect, update, delete)
func TestDSLFlow(t *testing.T) {
	ctx := context.Background()

	testCases := []struct {
		name    string
		tcFunc  func() *TestCase
		dslType types.Type
		stores  []types.StoreType
	}{
		{"Model", NewModelTestCase, types.TypeModel, []types.StoreType{types.StoreTypeDB, types.StoreTypeFile}},
		{"Connector", NewConnectorTestCase, types.TypeConnector, []types.StoreType{types.StoreTypeDB, types.StoreTypeFile}},
		{"MCP", NewMCPTestCase, types.TypeMCPClient, []types.StoreType{types.StoreTypeDB, types.StoreTypeFile}},
	}

	for _, tt := range testCases {
		for _, store := range tt.stores {
			t.Run(fmt.Sprintf("%s_%s", tt.name, store), func(t *testing.T) {
				// Clean test data before each test
				err := cleanTestData()
				if err != nil {
					t.Fatalf("Failed to clean test data: %v", err)
				}

				dsl, err := New(tt.dslType)
				if !assert.Nil(t, err) {
					return
				}

				tc := tt.tcFunc()

				// Create
				err = dsl.Create(ctx, tc.CreateOptions(store))
				if !assert.Nil(t, err) {
					return
				}

				// Inspect
				info, err := dsl.Inspect(ctx, tc.ID)
				if !assert.Nil(t, err) {
					return
				}
				assert.True(t, tc.AssertInfo(info))

				// Get source
				source, err := dsl.Source(ctx, tc.ID)
				if !assert.Nil(t, err) {
					return
				}
				assert.Equal(t, tc.Source, source)

				// Update
				err = dsl.Update(ctx, tc.UpdateOptions())
				if !assert.Nil(t, err) {
					return
				}

				// Verify updated
				info, err = dsl.Inspect(ctx, tc.ID)
				if !assert.Nil(t, err) {
					return
				}
				assert.True(t, tc.AssertUpdatedInfo(info))

				// Delete
				err = dsl.Delete(ctx, tc.DeleteOptions())
				if !assert.Nil(t, err) {
					return
				}

				// Verify deleted
				exists, err := dsl.Exists(ctx, tc.ID)
				if !assert.Nil(t, err) {
					return
				}
				assert.False(t, exists)
			})
		}
	}
}
