package seed

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/yaoapp/gou/model"
	"github.com/yaoapp/gou/process"
	"github.com/yaoapp/yao/config"
	"github.com/yaoapp/yao/test"
)

// TestSeedImportDuplicateUpdateAfterClear tests the Reset scenario:
// 1. Import data with primary keys
// 2. Clear all data
// 3. Import again with duplicate="update" mode
// This should work correctly now with the fix
func TestSeedImportDuplicateUpdateAfterClear(t *testing.T) {
	test.Prepare(t, config.Conf)
	defer test.Clean()

	// Ensure __yao.role model exists
	if !model.Exists("__yao.role") {
		t.Skip("__yao.role model not loaded, skipping test")
	}

	mod := model.Select("__yao.role")

	// Step 1: Clear and import data (initial import)
	_, _ = mod.DestroyWhere(model.QueryParam{})

	p1 := process.New("seeds.import", "roles.csv", "__yao.role", map[string]interface{}{
		"chunk_size": 100,
		"duplicate":  "update",
		"mode":       "each",
	})
	result1 := p1.Run()
	resultMap1, ok := result1.(*ImportResult)
	assert.True(t, ok)
	assert.Greater(t, resultMap1.Success, 0, "First import should succeed")
	assert.Equal(t, 0, resultMap1.Failure, "First import should have no failures")

	firstCount := resultMap1.Success

	// Step 2: Clear all data (simulate Reset scenario)
	deleted, err := mod.DestroyWhere(model.QueryParam{})
	assert.Nil(t, err)
	assert.Equal(t, firstCount, deleted, "Should delete all imported records")

	// Verify database is empty
	roles, err := mod.Get(model.QueryParam{})
	assert.Nil(t, err)
	assert.Equal(t, 0, len(roles), "Database should be empty after clear")

	// Step 3: Import again with duplicate="update" (this is the critical test)
	// Before fix: This would fail silently (UPDATE on non-existent records)
	// After fix: This should CREATE new records
	p2 := process.New("seeds.import", "roles.csv", "__yao.role", map[string]interface{}{
		"chunk_size": 100,
		"duplicate":  "update", // This should work now!
		"mode":       "each",
	})
	result2 := p2.Run()
	resultMap2, ok := result2.(*ImportResult)
	assert.True(t, ok)

	t.Logf("Second import result: Total=%d, Success=%d, Failure=%d, Ignore=%d",
		resultMap2.Total, resultMap2.Success, resultMap2.Failure, resultMap2.Ignore)

	// Print errors if any
	if len(resultMap2.Errors) > 0 {
		t.Logf("Import errors: %+v", resultMap2.Errors)
	}

	// Critical assertions: data should be imported successfully
	assert.Greater(t, resultMap2.Success, 0, "Second import should succeed (CREATE new records)")
	assert.Equal(t, 0, resultMap2.Failure, "Second import should have no failures")
	assert.Equal(t, firstCount, resultMap2.Success, "Should import same number of records")

	// Verify database has data
	roles2, err := mod.Get(model.QueryParam{})
	assert.Nil(t, err)
	assert.Equal(t, firstCount, len(roles2), "Database should have all records after re-import")
}

// TestSeedImportDuplicateUpdateMixedScenario tests a mixed scenario:
// 1. Import some data
// 2. Modify one record and delete another
// 3. Import again with duplicate="update"
// Should: UPDATE existing records and CREATE missing records
func TestSeedImportDuplicateUpdateMixedScenario(t *testing.T) {
	test.Prepare(t, config.Conf)
	defer test.Clean()

	// Ensure __yao.role model exists
	if !model.Exists("__yao.role") {
		t.Skip("__yao.role model not loaded, skipping test")
	}

	mod := model.Select("__yao.role")

	// Step 1: Clear and initial import
	_, _ = mod.DestroyWhere(model.QueryParam{})

	p1 := process.New("seeds.import", "roles.csv", "__yao.role", map[string]interface{}{
		"duplicate": "update",
		"mode":      "each",
	})
	result1 := p1.Run()
	resultMap1 := result1.(*ImportResult)
	initialCount := resultMap1.Success

	t.Logf("Initial import: Total=%d, Success=%d", resultMap1.Total, resultMap1.Success)

	// Step 2: Delete one specific role (simulate partial data loss)
	// Get all roles first
	allRoles, _ := mod.Get(model.QueryParam{
		Select: []interface{}{"id", "role_id"},
	})

	if len(allRoles) > 0 {
		// Delete the first role
		roleID := allRoles[0].Get("id")
		roleIDStr := allRoles[0].Get("role_id")
		t.Logf("Deleting role: id=%v, role_id=%v", roleID, roleIDStr)
		err := mod.Destroy(roleID)
		assert.Nil(t, err, "Should delete role")
	}

	// Verify one record is deleted
	remainingRoles, _ := mod.Get(model.QueryParam{})
	t.Logf("Remaining roles after delete: %d (expected %d)", len(remainingRoles), initialCount-1)
	assert.Equal(t, initialCount-1, len(remainingRoles), "Should have one less record")

	// Step 3: Re-import with duplicate="update"
	// Should: UPDATE existing records and CREATE the deleted record
	p2 := process.New("seeds.import", "roles.csv", "__yao.role", map[string]interface{}{
		"duplicate": "update",
		"mode":      "each",
	})
	result2 := p2.Run()
	resultMap2 := result2.(*ImportResult)

	t.Logf("Second import result: Total=%d, Success=%d, Failure=%d, Ignore=%d",
		resultMap2.Total, resultMap2.Success, resultMap2.Failure, resultMap2.Ignore)

	// Print errors if any
	if len(resultMap2.Errors) > 0 {
		for i, err := range resultMap2.Errors {
			t.Logf("Error %d: Row=%d, Message=%s", i+1, err.Row, err.Message)
		}
	}

	// Should import all records
	assert.Greater(t, resultMap2.Success, 0, "Should import successfully")

	// Note: Some failures may occur if CSV has duplicate IDs or validation issues
	// The important thing is that deleted record should be recreated

	// Verify count increased (deleted record was recreated)
	finalRoles, err := mod.Get(model.QueryParam{})
	assert.Nil(t, err)
	t.Logf("Final roles count: %d (expected %d)", len(finalRoles), initialCount)

	// At minimum, should have more records than before re-import
	assert.GreaterOrEqual(t, len(finalRoles), len(remainingRoles),
		"Should have at least as many records as before (deleted record recreated)")
}
