package model

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/yaoapp/yao/dsl/io"
	"github.com/yaoapp/yao/dsl/types"
)

func TestModelLoad(t *testing.T) {
	testCase := NewTestCase()
	fsio := io.NewFS(types.TypeModel)
	dbio := io.NewDB(types.TypeModel)
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
	err = manager.Load(context.Background(), &types.LoadOptions{
		ID:     testCase.ID,
		Source: testCase.Source,
	})
	assert.NoError(t, err)

	// Test Load from filesystem
	err = fsio.Create(&types.CreateOptions{
		ID:     testCase.ID + "_fs",
		Source: testCase.Source,
	})
	assert.NoError(t, err)

	path := types.ToPath(types.TypeModel, testCase.ID+"_fs")
	err = manager.Load(context.Background(), &types.LoadOptions{
		ID:    testCase.ID + "_fs",
		Path:  path,
		Store: "fs",
	})
	assert.NoError(t, err)

	// Test Load from database
	err = dbio.Create(&types.CreateOptions{
		ID:     testCase.ID + "_db",
		Source: testCase.Source,
	})
	assert.NoError(t, err)

	err = manager.Load(context.Background(), &types.LoadOptions{
		ID:    testCase.ID + "_db",
		Store: "db",
	})
	assert.NoError(t, err)

	// Test Load with default path (should use filesystem)
	err = manager.Load(context.Background(), &types.LoadOptions{
		ID: testCase.ID + "_fs",
	})
	assert.NoError(t, err)

	// Test Load with migration
	err = manager.Load(context.Background(), &types.LoadOptions{
		ID:      testCase.ID + "_fs",
		Options: map[string]interface{}{"migration": true},
	})
	assert.NoError(t, err)

	// Test Load with reset
	err = manager.Load(context.Background(), &types.LoadOptions{
		ID:      testCase.ID + "_fs",
		Options: map[string]interface{}{"reset": true},
	})
	assert.NoError(t, err)

	// Clean up
	err = fsio.Delete(testCase.ID + "_fs")
	assert.NoError(t, err)
	err = dbio.Delete(testCase.ID + "_db")
	assert.NoError(t, err)
	err = cleanTestData()
	assert.NoError(t, err)
}

func TestModelLoadWithDB(t *testing.T) {
	testCase := NewTestCase()
	dbio := io.NewDB(types.TypeModel)
	manager := New("", nil, dbio)

	// Create model in DB first
	err := dbio.Create(&types.CreateOptions{
		ID:     testCase.ID,
		Source: testCase.Source,
	})
	assert.NoError(t, err)

	// Test Load with Store=db
	err = manager.Load(context.Background(), &types.LoadOptions{
		ID:    testCase.ID,
		Store: "db",
	})
	assert.NoError(t, err)

	// Test Load non-existent model from DB
	err = manager.Load(context.Background(), &types.LoadOptions{
		ID:    "non-existent",
		Store: "db",
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found in database")

	// Clean up
	err = dbio.Delete(testCase.ID)
	assert.NoError(t, err)
	err = cleanTestData()
	assert.NoError(t, err)
}

func TestModelUnload(t *testing.T) {
	testCase := NewTestCase()
	fsio := io.NewFS(types.TypeModel)
	dbio := io.NewDB(types.TypeModel)
	manager := New("", fsio, dbio)

	// Test Unload with nil options
	err := manager.Unload(context.Background(), nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unload options is required")

	// Test Unload with empty ID
	err = manager.Unload(context.Background(), &types.UnloadOptions{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unload options id is required")

	// Test Unload non-existent model
	err = manager.Unload(context.Background(), &types.UnloadOptions{
		ID: "non-existent",
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "model non-existent not found")

	// Test Unload from filesystem
	err = fsio.Create(&types.CreateOptions{
		ID:     testCase.ID + "_fs",
		Source: testCase.Source,
	})
	assert.NoError(t, err)

	err = manager.Load(context.Background(), &types.LoadOptions{
		ID:    testCase.ID + "_fs",
		Store: "fs",
	})
	assert.NoError(t, err)

	err = manager.Unload(context.Background(), &types.UnloadOptions{
		ID:      testCase.ID + "_fs",
		Options: map[string]interface{}{"dropTable": true},
	})
	assert.NoError(t, err)

	// Test Unload from database
	err = dbio.Create(&types.CreateOptions{
		ID:     testCase.ID + "_db",
		Source: testCase.Source,
	})
	assert.NoError(t, err)

	err = manager.Load(context.Background(), &types.LoadOptions{
		ID:    testCase.ID + "_db",
		Store: "db",
	})
	assert.NoError(t, err)

	err = manager.Unload(context.Background(), &types.UnloadOptions{
		ID:      testCase.ID + "_db",
		Options: map[string]interface{}{"dropTable": true},
	})
	assert.NoError(t, err)

	// Clean up
	err = fsio.Delete(testCase.ID + "_fs")
	assert.NoError(t, err)
	err = dbio.Delete(testCase.ID + "_db")
	assert.NoError(t, err)
	err = cleanTestData()
	assert.NoError(t, err)
}

func TestModelReload(t *testing.T) {
	testCase := NewTestCase()
	fsio := io.NewFS(types.TypeModel)
	dbio := io.NewDB(types.TypeModel)
	manager := New("", fsio, dbio)

	// Test Reload with nil options
	err := manager.Reload(context.Background(), nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "reload options is required")

	// Test Reload with empty ID
	err = manager.Reload(context.Background(), &types.ReloadOptions{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "reload options id is required")

	// Test Reload from filesystem
	err = fsio.Create(&types.CreateOptions{
		ID:     testCase.ID + "_fs",
		Source: testCase.Source,
	})
	assert.NoError(t, err)

	err = manager.Load(context.Background(), &types.LoadOptions{
		ID:    testCase.ID + "_fs",
		Store: "fs",
	})
	assert.NoError(t, err)

	err = manager.Reload(context.Background(), &types.ReloadOptions{
		ID:      testCase.ID + "_fs",
		Store:   "fs",
		Options: map[string]interface{}{"migrate": true},
	})
	assert.NoError(t, err)

	// Test Reload from database
	err = dbio.Create(&types.CreateOptions{
		ID:     testCase.ID + "_db",
		Source: testCase.Source,
	})
	assert.NoError(t, err)

	err = manager.Load(context.Background(), &types.LoadOptions{
		ID:    testCase.ID + "_db",
		Store: "db",
	})
	assert.NoError(t, err)

	err = manager.Reload(context.Background(), &types.ReloadOptions{
		ID:      testCase.ID + "_db",
		Store:   "db",
		Options: map[string]interface{}{"migrate": true},
	})
	assert.NoError(t, err)

	// Clean up
	err = fsio.Delete(testCase.ID + "_fs")
	assert.NoError(t, err)
	err = dbio.Delete(testCase.ID + "_db")
	assert.NoError(t, err)
	err = cleanTestData()
	assert.NoError(t, err)
}

func TestModelLoaded(t *testing.T) {
	testCase := NewTestCase()
	fsio := io.NewFS(types.TypeModel)
	dbio := io.NewDB(types.TypeModel)
	manager := New("", fsio, dbio)

	// Test Load from filesystem
	err := fsio.Create(&types.CreateOptions{
		ID:     testCase.ID + "_fs",
		Source: testCase.Source,
	})
	assert.NoError(t, err)

	err = manager.Load(context.Background(), &types.LoadOptions{
		ID:    testCase.ID + "_fs",
		Store: "fs",
	})
	assert.NoError(t, err)

	// Test Load from database
	err = dbio.Create(&types.CreateOptions{
		ID:     testCase.ID + "_db",
		Source: testCase.Source,
	})
	assert.NoError(t, err)

	err = manager.Load(context.Background(), &types.LoadOptions{
		ID:    testCase.ID + "_db",
		Store: "db",
	})
	assert.NoError(t, err)

	// Test Loaded
	infos, err := manager.Loaded(context.Background())
	assert.NoError(t, err)
	assert.NotNil(t, infos)
	assert.Contains(t, infos, testCase.ID+"_fs")
	assert.Contains(t, infos, testCase.ID+"_db")

	// Verify metadata fields for filesystem model
	fsInfo := infos[testCase.ID+"_fs"]
	assert.Equal(t, testCase.ID+"_fs", fsInfo.ID)
	assert.Equal(t, types.TypeModel, fsInfo.Type)
	assert.Equal(t, testCase.Label, fsInfo.Label)
	assert.Equal(t, testCase.Description, fsInfo.Description)
	assert.ElementsMatch(t, testCase.Tags, fsInfo.Tags)
	assert.False(t, fsInfo.Readonly)
	assert.False(t, fsInfo.Builtin)
	// assert.False(t, fsInfo.Mtime.IsZero())
	// assert.False(t, fsInfo.Ctime.IsZero())

	// Verify metadata fields for database model
	dbInfo := infos[testCase.ID+"_db"]
	assert.Equal(t, testCase.ID+"_db", dbInfo.ID)
	assert.Equal(t, types.TypeModel, dbInfo.Type)
	assert.Equal(t, testCase.Label, dbInfo.Label)
	assert.Equal(t, testCase.Description, dbInfo.Description)
	assert.ElementsMatch(t, testCase.Tags, dbInfo.Tags)
	assert.False(t, dbInfo.Readonly)
	assert.False(t, dbInfo.Builtin)
	// assert.False(t, dbInfo.Mtime.IsZero())
	// assert.False(t, dbInfo.Ctime.IsZero())

	// Clean up
	err = fsio.Delete(testCase.ID + "_fs")
	assert.NoError(t, err)
	err = dbio.Delete(testCase.ID + "_db")
	assert.NoError(t, err)
	err = cleanTestData()
	assert.NoError(t, err)
}

func TestModelValidate(t *testing.T) {
	manager := New("", nil, nil)

	// Test Validate
	valid, messages := manager.Validate(context.Background(), "test source")
	assert.True(t, valid)
	assert.Empty(t, messages)
}

func TestModelExecute(t *testing.T) {
	manager := New("", nil, nil)

	// Test Execute
	result, err := manager.Execute(context.Background(), "test_id", "test_method")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Not implemented")
	assert.Nil(t, result)
}
