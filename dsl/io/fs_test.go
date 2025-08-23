package io

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/yaoapp/gou/application"
	"github.com/yaoapp/yao/dsl/types"
)

func prepare(t *testing.T) {
	root := os.Getenv("YAO_TEST_APPLICATION")
	if root == "" {
		t.Fatal("YAO_TEST_APPLICATION environment variable is not set")
	}

	// Create models directory if it doesn't exist
	modelsDir := filepath.Join(root, "models")
	if err := os.MkdirAll(modelsDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Clean test files
	files, err := os.ReadDir(modelsDir)
	if err != nil {
		t.Fatal(err)
	}

	// Remove test files
	for _, file := range files {
		if !file.IsDir() && strings.HasPrefix(file.Name(), "test_") && strings.HasSuffix(file.Name(), ".mod.yao") {
			path := filepath.Join(modelsDir, file.Name())
			if err := os.Remove(path); err != nil {
				t.Fatal(err)
			}
		}
	}

	// Clean test data from database
	err = cleanTestData()
	if err != nil {
		t.Fatal(err)
	}

	app, err := application.OpenFromDisk(root)
	if err != nil {
		t.Fatal(err)
	}
	application.App = app
}

func TestFSNew(t *testing.T) {
	fs := NewFS(types.TypeModel)
	fsImpl, ok := fs.(*FS)
	assert.True(t, ok)
	assert.Equal(t, types.TypeModel, fsImpl.Type)
}

func TestFSCreate(t *testing.T) {
	prepare(t)
	fs := NewFS(types.TypeModel)
	tc := NewTestCase()

	err := fs.Create(tc.CreateOptions())
	assert.Nil(t, err)

	// Check if exists
	exists, err := fs.Exists(tc.ID)
	assert.Nil(t, err)
	assert.True(t, exists)

	// Create again should fail
	err = fs.Create(tc.CreateOptions())
	assert.NotNil(t, err)
}

func TestFSInspect(t *testing.T) {
	prepare(t)
	fs := NewFS(types.TypeModel)
	tc := NewTestCase()

	err := fs.Create(tc.CreateOptions())
	assert.Nil(t, err)

	info, exists, err := fs.Inspect(tc.ID)
	assert.Nil(t, err)
	assert.True(t, exists)
	assert.True(t, tc.AssertInfo(info))
}

func TestFSSource(t *testing.T) {
	prepare(t)
	fs := NewFS(types.TypeModel)
	tc := NewTestCase()

	err := fs.Create(tc.CreateOptions())
	assert.Nil(t, err)

	data, exists, err := fs.Source(tc.ID)
	assert.Nil(t, err)
	assert.True(t, exists)
	assert.Equal(t, tc.Source, data)
}

func TestFSList(t *testing.T) {
	prepare(t)
	fs := NewFS(types.TypeModel)

	// Get initial count
	initialList, err := fs.List(&types.ListOptions{})
	assert.Nil(t, err)
	initialCount := len(initialList)

	tc1 := NewTestCase()
	tc2 := NewTestCase()

	// Create test files
	err = fs.Create(tc1.CreateOptions())
	assert.Nil(t, err)

	err = fs.Create(tc2.CreateOptions())
	assert.Nil(t, err)

	// List all
	list, err := fs.List(&types.ListOptions{})
	assert.Nil(t, err)
	assert.Equal(t, initialCount+2, len(list))

	// List with tag - should return both files since tags are OR relationship
	list, err = fs.List(tc1.ListOptions(false))
	assert.Nil(t, err)
	assert.Equal(t, 2, len(list))
	// Verify both files are in the results
	found := false
	for _, info := range list {
		if info.ID == tc1.ID {
			found = true
			break
		}
	}
	assert.True(t, found, "Should find tc1's file in results")
}

func TestFSUpdate(t *testing.T) {
	prepare(t)
	fs := NewFS(types.TypeModel)
	tc := NewTestCase()

	err := fs.Create(tc.CreateOptions())
	assert.Nil(t, err)

	// Update source
	err = fs.Update(tc.UpdateOptions())
	assert.Nil(t, err)

	info, exists, err := fs.Inspect(tc.ID)
	assert.Nil(t, err)
	assert.True(t, exists)
	assert.True(t, tc.AssertUpdatedInfo(info))

	// Update info
	err = fs.Update(tc.UpdateInfoOptions())
	assert.Nil(t, err)

	info, exists, err = fs.Inspect(tc.ID)
	assert.Nil(t, err)
	assert.True(t, exists)
	assert.True(t, tc.AssertUpdatedInfoViaInfo(info))
}

func TestFSDelete(t *testing.T) {
	prepare(t)
	fs := NewFS(types.TypeModel)
	tc := NewTestCase()

	err := fs.Create(tc.CreateOptions())
	assert.Nil(t, err)

	err = fs.Delete(tc.ID)
	assert.Nil(t, err)

	exists, err := fs.Exists(tc.ID)
	assert.Nil(t, err)
	assert.False(t, exists)
}

func TestFSFlow(t *testing.T) {
	prepare(t)
	fs := NewFS(types.TypeModel)
	tc := NewTestCase()

	// Create
	err := fs.Create(tc.CreateOptions())
	assert.Nil(t, err)

	// Inspect
	info, exists, err := fs.Inspect(tc.ID)
	assert.Nil(t, err)
	assert.True(t, exists)
	assert.True(t, tc.AssertInfo(info))

	// Update
	err = fs.Update(tc.UpdateOptions())
	assert.Nil(t, err)

	info, exists, err = fs.Inspect(tc.ID)
	assert.Nil(t, err)
	assert.True(t, exists)
	assert.True(t, tc.AssertUpdatedInfo(info))

	// Delete
	err = fs.Delete(tc.ID)
	assert.Nil(t, err)

	exists, err = fs.Exists(tc.ID)
	assert.Nil(t, err)
	assert.False(t, exists)
}
