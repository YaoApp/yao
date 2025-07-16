package connector

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/yaoapp/yao/dsl/io"
	"github.com/yaoapp/yao/dsl/types"
)

func TestConnectorLoad(t *testing.T) {
	testCase := NewTestCase()
	fsio := io.NewFS(types.TypeConnector)
	dbio := io.NewDB(types.TypeConnector)
	manager := New("", fsio, dbio)

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

	path := types.ToPath(types.TypeConnector, testCase.ID)
	err = manager.Load(context.Background(), &types.LoadOptions{
		ID:    testCase.ID,
		Path:  path,
		Store: "fs",
	})
	assert.NoError(t, err)

	// Test Load from database
	err = dbio.Create(testCase.CreateOptions())
	assert.NoError(t, err)

	err = manager.Load(context.Background(), &types.LoadOptions{
		ID:    testCase.ID,
		Store: "db",
	})
	assert.NoError(t, err)

	// Clean up
	err = fsio.Delete(testCase.ID)
	assert.NoError(t, err)
	err = dbio.Delete(testCase.ID)
	assert.NoError(t, err)
}

func TestConnectorUnload(t *testing.T) {
	testCase := NewTestCase()
	fsio := io.NewFS(types.TypeConnector)
	dbio := io.NewDB(types.TypeConnector)
	manager := New("", fsio, dbio)

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
		Store: "fs",
	})
	assert.NoError(t, err)

	err = manager.Unload(context.Background(), testCase.UnloadOptions())
	assert.NoError(t, err)

	// Clean up
	err = fsio.Delete(testCase.ID)
	assert.NoError(t, err)
}

func TestConnectorReload(t *testing.T) {
	testCase := NewTestCase()
	fsio := io.NewFS(types.TypeConnector)
	dbio := io.NewDB(types.TypeConnector)
	manager := New("", fsio, dbio)

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
		Store: "fs",
	})
	assert.NoError(t, err)

	err = manager.Reload(context.Background(), testCase.ReloadOptions())
	assert.NoError(t, err)

	// Clean up
	err = fsio.Delete(testCase.ID)
	assert.NoError(t, err)
}

func TestConnectorLoaded(t *testing.T) {
	testCase := NewTestCase()
	fsio := io.NewFS(types.TypeConnector)
	dbio := io.NewDB(types.TypeConnector)
	manager := New("", fsio, dbio)

	// Load from filesystem
	err := fsio.Create(testCase.CreateOptions())
	assert.NoError(t, err)

	err = manager.Load(context.Background(), &types.LoadOptions{
		ID:    testCase.ID,
		Store: "fs",
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
	assert.Equal(t, types.TypeConnector, fsInfo.Type)
	assert.Equal(t, testCase.Label, fsInfo.Label)
	assert.Equal(t, testCase.Description, fsInfo.Description)
	assert.ElementsMatch(t, testCase.Tags, fsInfo.Tags)
	assert.False(t, fsInfo.Readonly)
	assert.False(t, fsInfo.Builtin)

	// Clean up
	err = fsio.Delete(testCase.ID)
	assert.NoError(t, err)
}

func TestConnectorValidate(t *testing.T) {
	manager := New("", nil, nil)

	// Test Validate
	valid, messages := manager.Validate(context.Background(), "test source")
	assert.True(t, valid)
	assert.Empty(t, messages)
}

func TestConnectorExecute(t *testing.T) {
	manager := New("", nil, nil)

	// Test Execute
	result, err := manager.Execute(context.Background(), "test_id", "test_method")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Not implemented")
	assert.Nil(t, result)
}
