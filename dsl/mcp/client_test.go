package mcp

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/yaoapp/yao/dsl/io"
	"github.com/yaoapp/yao/dsl/types"
)

func TestMCPClientLoad(t *testing.T) {
	testCase := NewTestCase()
	fsio := io.NewFS(types.TypeMCPClient)
	dbio := io.NewDB(types.TypeMCPClient)
	manager := NewClient("mcps", fsio, dbio)

	// Test Load with nil options
	err := manager.Load(context.Background(), nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "load options is required")

	// Test Load with empty ID
	err = manager.Load(context.Background(), &types.LoadOptions{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "load options id is required")

	// Test Load with Source
	err = manager.Load(context.Background(), testCase.LoadOptions())
	assert.NoError(t, err)

	// Test Load from filesystem
	err = fsio.Create(testCase.CreateOptions())
	assert.NoError(t, err)

	path := types.ToPath(types.TypeMCPClient, testCase.ID)
	err = manager.Load(context.Background(), &types.LoadOptions{
		ID:    testCase.ID,
		Path:  path,
		Store: types.StoreTypeFile,
	})
	assert.NoError(t, err)

	// Test Load from database
	err = dbio.Create(testCase.CreateOptions())
	assert.NoError(t, err)

	err = manager.Load(context.Background(), &types.LoadOptions{
		ID:    testCase.ID,
		Store: types.StoreTypeDB,
	})
	assert.NoError(t, err)

	// Clean up
	err = fsio.Delete(testCase.ID)
	assert.NoError(t, err)
	err = dbio.Delete(testCase.ID)
	assert.NoError(t, err)
}

func TestMCPClientUnload(t *testing.T) {
	testCase := NewTestCase()
	fsio := io.NewFS(types.TypeMCPClient)
	dbio := io.NewDB(types.TypeMCPClient)
	manager := NewClient("mcps", fsio, dbio)

	// Test Unload with nil options
	err := manager.Unload(context.Background(), nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unload options is required")

	// Test Unload with empty ID
	err = manager.Unload(context.Background(), &types.UnloadOptions{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unload options id is required")

	// Load and then unload from filesystem
	err = fsio.Create(testCase.CreateOptions())
	assert.NoError(t, err)

	err = manager.Load(context.Background(), &types.LoadOptions{
		ID:    testCase.ID,
		Store: types.StoreTypeFile,
	})
	assert.NoError(t, err)

	err = manager.Unload(context.Background(), testCase.UnloadOptions())
	assert.NoError(t, err)

	// Clean up
	err = fsio.Delete(testCase.ID)
	assert.NoError(t, err)
}

func TestMCPClientReload(t *testing.T) {
	testCase := NewTestCase()
	fsio := io.NewFS(types.TypeMCPClient)
	dbio := io.NewDB(types.TypeMCPClient)
	manager := NewClient("mcps", fsio, dbio)

	// Test Reload with nil options
	err := manager.Reload(context.Background(), nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "reload options is required")

	// Test Reload with empty ID
	err = manager.Reload(context.Background(), &types.ReloadOptions{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "reload options id is required")

	// Load and then reload from filesystem
	err = fsio.Create(testCase.CreateOptions())
	assert.NoError(t, err)

	err = manager.Load(context.Background(), &types.LoadOptions{
		ID:    testCase.ID,
		Store: types.StoreTypeFile,
	})
	assert.NoError(t, err)

	err = manager.Reload(context.Background(), testCase.ReloadOptions())
	assert.NoError(t, err)

	// Clean up
	err = fsio.Delete(testCase.ID)
	assert.NoError(t, err)
}

func TestMCPClientLoaded(t *testing.T) {
	testCase := NewTestCase()
	fsio := io.NewFS(types.TypeMCPClient)
	dbio := io.NewDB(types.TypeMCPClient)
	manager := NewClient("mcps", fsio, dbio)

	// Load from filesystem
	err := fsio.Create(testCase.CreateOptions())
	assert.NoError(t, err)

	err = manager.Load(context.Background(), &types.LoadOptions{
		ID:    testCase.ID,
		Store: types.StoreTypeFile,
	})
	assert.NoError(t, err)

	// Test Loaded
	infos, err := manager.Loaded(context.Background())
	assert.NoError(t, err)
	assert.NotNil(t, infos)
	assert.Contains(t, infos, testCase.ID)

	// Verify metadata fields
	fsInfo := infos[testCase.ID]
	assert.Equal(t, testCase.ID, fsInfo.ID)
	assert.Equal(t, types.TypeMCPClient, fsInfo.Type)
	assert.Equal(t, testCase.Label, fsInfo.Label)
	assert.Equal(t, testCase.Description, fsInfo.Description)
	assert.ElementsMatch(t, testCase.Tags, fsInfo.Tags)
	assert.False(t, fsInfo.Readonly)
	assert.False(t, fsInfo.Builtin)

	// Clean up
	err = fsio.Delete(testCase.ID)
	assert.NoError(t, err)
}

func TestMCPClientValidate(t *testing.T) {
	manager := NewClient("mcps", nil, nil)

	// Test Validate
	valid, messages := manager.Validate(context.Background(), "test source")
	assert.True(t, valid)
	assert.Empty(t, messages)
}

func TestMCPClientExecute(t *testing.T) {
	manager := NewClient("mcps", nil, nil)

	// Test Execute
	result, err := manager.Execute(context.Background(), "test_id", "test_method")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Not implemented")
	assert.Nil(t, result)
}

func TestMCPClientHTTPLoad(t *testing.T) {
	testCase := NewHTTPTestCase()
	fsio := io.NewFS(types.TypeMCPClient)
	dbio := io.NewDB(types.TypeMCPClient)
	manager := NewClient("mcps", fsio, dbio)

	// Test Load with HTTP Source
	err := manager.Load(context.Background(), testCase.LoadOptions())
	assert.NoError(t, err)

	// Test Loaded
	infos, err := manager.Loaded(context.Background())
	assert.NoError(t, err)
	assert.NotNil(t, infos)
	assert.Contains(t, infos, testCase.ID)

	// Verify HTTP metadata fields
	httpInfo := infos[testCase.ID]
	assert.Equal(t, testCase.ID, httpInfo.ID)
	assert.Equal(t, types.TypeMCPClient, httpInfo.Type)
	assert.Equal(t, "Test HTTP MCP Client", httpInfo.Label)
	assert.Equal(t, "Test HTTP MCP Client Description", httpInfo.Description)
	assert.Contains(t, httpInfo.Tags, "http")
	assert.False(t, httpInfo.Readonly)
	assert.False(t, httpInfo.Builtin)

	// Test Unload
	err = manager.Unload(context.Background(), testCase.UnloadOptions())
	assert.NoError(t, err)
}

func TestMCPClientSSELoad(t *testing.T) {
	testCase := NewSSETestCase()
	fsio := io.NewFS(types.TypeMCPClient)
	dbio := io.NewDB(types.TypeMCPClient)
	manager := NewClient("mcps", fsio, dbio)

	// Test Load with SSE Source
	err := manager.Load(context.Background(), testCase.LoadOptions())
	assert.NoError(t, err)

	// Test Loaded
	infos, err := manager.Loaded(context.Background())
	assert.NoError(t, err)
	assert.NotNil(t, infos)
	assert.Contains(t, infos, testCase.ID)

	// Verify SSE metadata fields
	sseInfo := infos[testCase.ID]
	assert.Equal(t, testCase.ID, sseInfo.ID)
	assert.Equal(t, types.TypeMCPClient, sseInfo.Type)
	assert.Equal(t, "Test SSE MCP Client", sseInfo.Label)
	assert.Equal(t, "Test SSE MCP Client Description", sseInfo.Description)
	assert.Contains(t, sseInfo.Tags, "sse")
	assert.False(t, sseInfo.Readonly)
	assert.False(t, sseInfo.Builtin)

	// Test Unload
	err = manager.Unload(context.Background(), testCase.UnloadOptions())
	assert.NoError(t, err)
}

func TestMCPClientLoadWithDatabaseStore(t *testing.T) {
	testCase := NewTestCase()
	fsio := io.NewFS(types.TypeMCPClient)
	dbio := io.NewDB(types.TypeMCPClient)
	manager := NewClient("mcps", fsio, dbio)

	// Create in database first
	err := dbio.Create(testCase.CreateOptions())
	assert.NoError(t, err)

	// Test Load from database
	err = manager.Load(context.Background(), &types.LoadOptions{
		ID:    testCase.ID,
		Store: types.StoreTypeDB,
	})
	assert.NoError(t, err)

	// Test Loaded
	infos, err := manager.Loaded(context.Background())
	assert.NoError(t, err)
	assert.NotNil(t, infos)
	assert.Contains(t, infos, testCase.ID)

	// Verify metadata fields
	dbInfo := infos[testCase.ID]
	assert.Equal(t, testCase.ID, dbInfo.ID)
	assert.Equal(t, types.TypeMCPClient, dbInfo.Type)
	assert.Equal(t, testCase.Label, dbInfo.Label)
	assert.Equal(t, testCase.Description, dbInfo.Description)
	assert.ElementsMatch(t, testCase.Tags, dbInfo.Tags)
	assert.False(t, dbInfo.Readonly)
	assert.False(t, dbInfo.Builtin)

	// Test Reload from database
	err = manager.Reload(context.Background(), &types.ReloadOptions{
		ID:     testCase.ID,
		Source: testCase.UpdatedSource,
		Store:  types.StoreTypeDB,
	})
	assert.NoError(t, err)

	// Clean up
	err = dbio.Delete(testCase.ID)
	assert.NoError(t, err)
}

func TestMCPClientLoadWithFileStore(t *testing.T) {
	testCase := NewTestCase()
	fsio := io.NewFS(types.TypeMCPClient)
	dbio := io.NewDB(types.TypeMCPClient)
	manager := NewClient("mcps", fsio, dbio)

	// Create in file system first
	err := fsio.Create(testCase.CreateOptions())
	assert.NoError(t, err)

	// Test Load from file system with explicit path
	path := types.ToPath(types.TypeMCPClient, testCase.ID)
	err = manager.Load(context.Background(), &types.LoadOptions{
		ID:    testCase.ID,
		Path:  path,
		Store: types.StoreTypeFile,
	})
	assert.NoError(t, err)

	// Test Loaded
	infos, err := manager.Loaded(context.Background())
	assert.NoError(t, err)
	assert.NotNil(t, infos)
	assert.Contains(t, infos, testCase.ID)

	// Test Reload from file system
	err = manager.Reload(context.Background(), &types.ReloadOptions{
		ID:     testCase.ID,
		Path:   path,
		Source: testCase.UpdatedSource,
		Store:  types.StoreTypeFile,
	})
	assert.NoError(t, err)

	// Clean up
	err = fsio.Delete(testCase.ID)
	assert.NoError(t, err)
}
