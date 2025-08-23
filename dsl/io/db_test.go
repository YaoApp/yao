package io

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/yaoapp/yao/dsl/types"
)

func TestDBNew(t *testing.T) {
	db := NewDB(types.TypeModel)
	dbImpl, ok := db.(*DB)
	assert.True(t, ok)
	assert.Equal(t, types.TypeModel, dbImpl.Type)
}

func TestDBCreate(t *testing.T) {
	prepare(t)
	db := NewDB(types.TypeModel)
	tc := NewTestCase()

	err := db.Create(tc.CreateOptions())
	assert.Nil(t, err)

	// Check if exists
	exists, err := db.Exists(tc.ID)
	assert.Nil(t, err)
	assert.True(t, exists)

	// Create again should fail
	err = db.Create(tc.CreateOptions())
	assert.NotNil(t, err)
}

func TestDBInspect(t *testing.T) {
	prepare(t)
	db := NewDB(types.TypeModel)
	tc := NewTestCase()

	err := db.Create(tc.CreateOptions())
	assert.Nil(t, err)

	info, exists, err := db.Inspect(tc.ID)
	assert.Nil(t, err)
	assert.True(t, exists)
	assert.True(t, tc.AssertInfo(info))
}

func TestDBSource(t *testing.T) {
	prepare(t)
	db := NewDB(types.TypeModel)
	tc := NewTestCase()

	err := db.Create(tc.CreateOptions())
	assert.Nil(t, err)

	data, exists, err := db.Source(tc.ID)
	assert.Nil(t, err)
	assert.True(t, exists)
	assert.Equal(t, tc.Source, data)
}

func TestDBList(t *testing.T) {
	prepare(t)
	db := NewDB(types.TypeModel)
	tc1 := NewTestCase()
	tc2 := NewTestCase()

	// Create test files
	err := db.Create(tc1.CreateOptions())
	assert.Nil(t, err)

	err = db.Create(tc2.CreateOptions())
	assert.Nil(t, err)

	// List all
	list, err := db.List(&types.ListOptions{})
	assert.Nil(t, err)
	assert.Equal(t, 2, len(list))

	// List with tag
	list, err = db.List(tc1.ListOptions(false))
	assert.Nil(t, err)
	assert.Equal(t, 1, len(list))
	if assert.Greater(t, len(list), 0, "List should not be empty") {
		assert.Equal(t, tc1.ID, list[0].ID)
	}
}

func TestDBUpdate(t *testing.T) {
	prepare(t)
	db := NewDB(types.TypeModel)
	tc := NewTestCase()

	err := db.Create(tc.CreateOptions())
	assert.Nil(t, err)

	// Update source
	err = db.Update(tc.UpdateOptions())
	assert.Nil(t, err)

	info, exists, err := db.Inspect(tc.ID)
	assert.Nil(t, err)
	assert.True(t, exists)
	assert.True(t, tc.AssertUpdatedInfo(info))

	// Update info
	err = db.Update(tc.UpdateInfoOptions())
	assert.Nil(t, err)

	info, exists, err = db.Inspect(tc.ID)
	assert.Nil(t, err)
	assert.True(t, exists)
	assert.True(t, tc.AssertUpdatedInfoViaInfo(info))
}

func TestDBDelete(t *testing.T) {
	prepare(t)
	db := NewDB(types.TypeModel)
	tc := NewTestCase()

	err := db.Create(tc.CreateOptions())
	assert.Nil(t, err)

	err = db.Delete(tc.ID)
	assert.Nil(t, err)

	exists, err := db.Exists(tc.ID)
	assert.Nil(t, err)
	assert.False(t, exists)
}

func TestDBFlow(t *testing.T) {
	prepare(t)
	db := NewDB(types.TypeModel)
	tc := NewTestCase()

	// Create
	err := db.Create(tc.CreateOptions())
	assert.Nil(t, err)

	// Inspect
	info, exists, err := db.Inspect(tc.ID)
	assert.Nil(t, err)
	assert.True(t, exists)
	assert.True(t, tc.AssertInfo(info))

	// Update
	err = db.Update(tc.UpdateOptions())
	assert.Nil(t, err)

	info, exists, err = db.Inspect(tc.ID)
	assert.Nil(t, err)
	assert.True(t, exists)
	assert.True(t, tc.AssertUpdatedInfo(info))

	// Delete
	err = db.Delete(tc.ID)
	assert.Nil(t, err)

	exists, err = db.Exists(tc.ID)
	assert.Nil(t, err)
	assert.False(t, exists)
}
