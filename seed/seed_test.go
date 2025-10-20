package seed

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/yaoapp/gou/model"
	"github.com/yaoapp/gou/process"
	"github.com/yaoapp/yao/config"
	"github.com/yaoapp/yao/test"
)

// TestSeedImportCSV tests importing roles from CSV file
func TestSeedImportCSV(t *testing.T) {
	test.Prepare(t, config.Conf)
	defer test.Clean()

	// Ensure __yao.role model exists
	if !model.Exists("__yao.role") {
		t.Skip("__yao.role model not loaded, skipping test")
	}

	// Clear existing roles
	mod := model.Select("__yao.role")
	_, _ = mod.DestroyWhere(model.QueryParam{})

	// Import CSV
	p := process.New("seeds.import", "roles.csv", "__yao.role")
	result := p.Run()

	// Verify result
	assert.NotNil(t, result)
	resultMap, ok := result.(*ImportResult)
	assert.True(t, ok, "Result should be ImportResult")
	assert.Greater(t, resultMap.Total, 0, "Should import at least 1 record")
	assert.Greater(t, resultMap.Success, 0, "Should have successful imports")
	assert.Equal(t, resultMap.Total, resultMap.Success+resultMap.Failure+resultMap.Ignore,
		"Total should equal sum of success, failure, and ignore")

	// Verify data in database
	roles, err := mod.Get(model.QueryParam{})
	assert.Nil(t, err)
	assert.Greater(t, len(roles), 0, "Should have roles in database")

	// Check specific role
	adminRoles, _ := mod.Get(model.QueryParam{
		Wheres: []model.QueryWhere{
			{Column: "role_id", Value: "admin"},
		},
	})
	if len(adminRoles) > 0 {
		assert.Equal(t, "admin", adminRoles[0].Get("role_id"))
		assert.Equal(t, "Administrator", adminRoles[0].Get("name"))
	}
}

// TestSeedImportJSON tests importing roles from JSON file
func TestSeedImportJSON(t *testing.T) {
	test.Prepare(t, config.Conf)
	defer test.Clean()

	// Ensure __yao.role model exists
	if !model.Exists("__yao.role") {
		t.Skip("__yao.role model not loaded, skipping test")
	}

	// Clear existing roles
	mod := model.Select("__yao.role")
	_, _ = mod.DestroyWhere(model.QueryParam{})

	// Import JSON
	p := process.New("seeds.import", "roles.json", "__yao.role")
	result := p.Run()

	// Verify result
	assert.NotNil(t, result)
	resultMap, ok := result.(*ImportResult)
	assert.True(t, ok, "Result should be ImportResult")
	assert.Greater(t, resultMap.Total, 0, "Should import at least 1 record")
	assert.Greater(t, resultMap.Success, 0, "Should have successful imports")

	// Verify data in database
	roles, err := mod.Get(model.QueryParam{})
	assert.Nil(t, err)
	assert.Greater(t, len(roles), 0, "Should have roles in database")

	// Check that permissions JSON was imported correctly
	adminRoles, _ := mod.Get(model.QueryParam{
		Wheres: []model.QueryWhere{
			{Column: "role_id", Value: "admin"},
		},
	})
	if len(adminRoles) > 0 {
		assert.Equal(t, "admin", adminRoles[0].Get("role_id"))
		assert.NotNil(t, adminRoles[0].Get("permissions"), "Should have permissions")
	}
}

// TestSeedImportXLSX tests importing roles from XLSX file
func TestSeedImportXLSX(t *testing.T) {
	test.Prepare(t, config.Conf)
	defer test.Clean()

	// Ensure __yao.role model exists
	if !model.Exists("__yao.role") {
		t.Skip("__yao.role model not loaded, skipping test")
	}

	// Clear existing roles
	mod := model.Select("__yao.role")
	_, _ = mod.DestroyWhere(model.QueryParam{})

	// Import XLSX file
	p := process.New("seeds.import", "roles.xlsx", "__yao.role")
	result := p.Run()

	// Verify result
	assert.NotNil(t, result)
	resultMap, ok := result.(*ImportResult)
	assert.True(t, ok, "Result should be ImportResult")
	assert.Greater(t, resultMap.Total, 0, "Should import at least 1 record")
	assert.Greater(t, resultMap.Success, 0, "Should have successful imports")

	// Verify data in database
	roles, err := mod.Get(model.QueryParam{})
	assert.Nil(t, err)
	assert.Greater(t, len(roles), 0, "Should have roles in database")

	// Check specific role
	adminRoles, _ := mod.Get(model.QueryParam{
		Wheres: []model.QueryWhere{
			{Column: "role_id", Value: "admin"},
		},
	})
	if len(adminRoles) > 0 {
		assert.Equal(t, "admin", adminRoles[0].Get("role_id"))
		assert.Equal(t, "Administrator", adminRoles[0].Get("name"))
	}
}

// TestSeedImportYao tests importing roles from Yao file (JSONC)
func TestSeedImportYao(t *testing.T) {
	test.Prepare(t, config.Conf)
	defer test.Clean()

	// Ensure __yao.role model exists
	if !model.Exists("__yao.role") {
		t.Skip("__yao.role model not loaded, skipping test")
	}

	// Clear existing roles
	mod := model.Select("__yao.role")
	_, _ = mod.DestroyWhere(model.QueryParam{})

	// Import Yao file
	p := process.New("seeds.import", "roles.yao", "__yao.role")
	result := p.Run()

	// Verify result
	assert.NotNil(t, result)
	resultMap, ok := result.(*ImportResult)
	assert.True(t, ok, "Result should be ImportResult")
	assert.Greater(t, resultMap.Total, 0, "Should import at least 1 record")
	assert.Greater(t, resultMap.Success, 0, "Should have successful imports")

	// Verify data in database
	roles, err := mod.Get(model.QueryParam{})
	assert.Nil(t, err)
	assert.Greater(t, len(roles), 0, "Should have roles in database")
}

// TestSeedImportWithOptions tests importing with custom options
func TestSeedImportWithOptions(t *testing.T) {
	test.Prepare(t, config.Conf)
	defer test.Clean()

	// Ensure __yao.role model exists
	if !model.Exists("__yao.role") {
		t.Skip("__yao.role model not loaded, skipping test")
	}

	// Clear existing roles
	mod := model.Select("__yao.role")
	_, _ = mod.DestroyWhere(model.QueryParam{})

	// Import with batch mode
	p := process.New("seeds.import", "roles.json", "__yao.role", map[string]interface{}{
		"chunk_size": 2,
		"duplicate":  "ignore",
		"mode":       "batch",
	})
	result := p.Run()

	// Verify result
	assert.NotNil(t, result)
	resultMap, ok := result.(*ImportResult)
	assert.True(t, ok)
	assert.Greater(t, resultMap.Success, 0)

	// Try importing again with ignore strategy
	p2 := process.New("seeds.import", "roles.json", "__yao.role", map[string]interface{}{
		"duplicate": "ignore",
	})
	result2 := p2.Run()

	resultMap2, ok := result2.(*ImportResult)
	assert.True(t, ok)
	// With ignore strategy, duplicates should be ignored
	assert.Greater(t, resultMap2.Ignore, 0, "Should have ignored duplicates")
}

// TestSeedImportEachMode tests importing with each mode (single record)
func TestSeedImportEachMode(t *testing.T) {
	test.Prepare(t, config.Conf)
	defer test.Clean()

	// Ensure __yao.role model exists
	if !model.Exists("__yao.role") {
		t.Skip("__yao.role model not loaded, skipping test")
	}

	// Clear existing roles
	mod := model.Select("__yao.role")
	_, _ = mod.DestroyWhere(model.QueryParam{})

	// Import with each mode
	p := process.New("seeds.import", "roles.json", "__yao.role", map[string]interface{}{
		"mode":      "each",
		"duplicate": "ignore",
	})
	result := p.Run()

	// Verify result
	assert.NotNil(t, result)
	resultMap, ok := result.(*ImportResult)
	assert.True(t, ok)
	assert.Greater(t, resultMap.Success, 0)

	// Verify data in database
	roles, err := mod.Get(model.QueryParam{})
	assert.Nil(t, err)
	assert.Greater(t, len(roles), 0)
}

// TestSeedImportDuplicateIgnore tests importing with ignore duplicate strategy
func TestSeedImportDuplicateIgnore(t *testing.T) {
	test.Prepare(t, config.Conf)
	defer test.Clean()

	// Ensure __yao.role model exists
	if !model.Exists("__yao.role") {
		t.Skip("__yao.role model not loaded, skipping test")
	}

	// Clear existing roles
	mod := model.Select("__yao.role")
	_, _ = mod.DestroyWhere(model.QueryParam{})

	// First import
	p1 := process.New("seeds.import", "roles.json", "__yao.role")
	result1 := p1.Run()
	resultMap1, ok := result1.(*ImportResult)
	assert.True(t, ok)
	firstSuccess := resultMap1.Success

	// Get the IDs of imported records
	roles1, err := mod.Get(model.QueryParam{
		Select: []interface{}{"id", "role_id"},
	})
	assert.Nil(t, err)
	assert.Greater(t, len(roles1), 0)

	// Second import with ignore mode
	p2 := process.New("seeds.import", "roles.json", "__yao.role", map[string]interface{}{
		"mode":      "each",
		"duplicate": "ignore",
	})
	result2 := p2.Run()

	// Verify result
	assert.NotNil(t, result2)
	resultMap2, ok := result2.(*ImportResult)
	assert.True(t, ok)
	// With ignore strategy, duplicates should be ignored
	assert.Greater(t, resultMap2.Ignore, 0, "Should have ignored duplicates")

	// Verify count hasn't changed (ignored duplicates, not created new)
	roles2, err := mod.Get(model.QueryParam{})
	assert.Nil(t, err)
	assert.Equal(t, firstSuccess, len(roles2), "Should have same number of roles after re-import")
}

// TestSeedImportChunkSize tests chunk processing
func TestSeedImportChunkSize(t *testing.T) {
	test.Prepare(t, config.Conf)
	defer test.Clean()

	// Ensure __yao.role model exists
	if !model.Exists("__yao.role") {
		t.Skip("__yao.role model not loaded, skipping test")
	}

	// Clear existing roles
	mod := model.Select("__yao.role")
	_, _ = mod.DestroyWhere(model.QueryParam{})

	// Import with small chunk size
	p := process.New("seeds.import", "roles.json", "__yao.role", map[string]interface{}{
		"chunk_size": 1, // Process one record at a time
		"mode":       "batch",
	})
	result := p.Run()

	// Verify result
	assert.NotNil(t, result)
	resultMap, ok := result.(*ImportResult)
	assert.True(t, ok)
	assert.Greater(t, resultMap.Success, 0)

	// Verify all data imported correctly
	roles, err := mod.Get(model.QueryParam{})
	assert.Nil(t, err)
	assert.Greater(t, len(roles), 0)
}

// TestSeedImportJSONFields tests that JSON fields are correctly parsed from CSV
func TestSeedImportJSONFields(t *testing.T) {
	test.Prepare(t, config.Conf)
	defer test.Clean()

	// Ensure __yao.role model exists
	if !model.Exists("__yao.role") {
		t.Skip("__yao.role model not loaded, skipping test")
	}

	// Clear existing roles
	mod := model.Select("__yao.role")
	_, _ = mod.DestroyWhere(model.QueryParam{})

	// Import CSV file
	p := process.New("seeds.import", "roles.csv", "__yao.role")
	result := p.Run()

	// Verify result
	assert.NotNil(t, result)
	resultMap, ok := result.(*ImportResult)
	assert.True(t, ok, "Result should be ImportResult")
	assert.Greater(t, resultMap.Success, 0, "Should have successful imports")

	// Get imported data
	roles, err := mod.Get(model.QueryParam{
		Wheres: []model.QueryWhere{
			{Column: "role_id", Value: "admin"},
		},
	})
	assert.Nil(t, err)
	assert.Equal(t, 1, len(roles), "Should find admin role")

	// Verify JSON fields are parsed as objects, not strings
	adminRole := roles[0]

	// Check permissions field (should be a map, not a string)
	permissions := adminRole.Get("permissions")
	assert.NotNil(t, permissions, "Permissions should not be nil")
	permissionsMap, ok := permissions.(map[string]interface{})
	assert.True(t, ok, "Permissions should be parsed as map[string]interface{}, got %T", permissions)
	assert.NotNil(t, permissionsMap["users"], "Should have users permissions")

	// Check metadata field (should be a map, not a string)
	metadata := adminRole.Get("metadata")
	assert.NotNil(t, metadata, "Metadata should not be nil")
	metadataMap, ok := metadata.(map[string]interface{})
	assert.True(t, ok, "Metadata should be parsed as map[string]interface{}, got %T", metadata)

	// Verify nested values
	if usersPerms, ok := permissionsMap["users"].([]interface{}); ok {
		assert.Greater(t, len(usersPerms), 0, "Should have user permissions")
		assert.Contains(t, usersPerms, "create", "Should have create permission")
	}

	if maxUsers, ok := metadataMap["max_users"].(float64); ok {
		assert.Equal(t, float64(5), maxUsers, "Max users should be 5")
	}
}

// TestSeedImportXLSXJSONFields tests that JSON fields are correctly parsed from XLSX
func TestSeedImportXLSXJSONFields(t *testing.T) {
	test.Prepare(t, config.Conf)
	defer test.Clean()

	// Ensure __yao.role model exists
	if !model.Exists("__yao.role") {
		t.Skip("__yao.role model not loaded, skipping test")
	}

	// Clear existing roles
	mod := model.Select("__yao.role")
	_, _ = mod.DestroyWhere(model.QueryParam{})

	// Import XLSX file
	p := process.New("seeds.import", "roles.xlsx", "__yao.role")
	result := p.Run()

	// Verify result
	assert.NotNil(t, result)
	resultMap, ok := result.(*ImportResult)
	assert.True(t, ok, "Result should be ImportResult")
	assert.Greater(t, resultMap.Success, 0, "Should have successful imports")

	// Get imported data
	roles, err := mod.Get(model.QueryParam{
		Wheres: []model.QueryWhere{
			{Column: "role_id", Value: "admin"},
		},
	})
	assert.Nil(t, err)
	assert.Equal(t, 1, len(roles), "Should find admin role")

	// Verify JSON fields are parsed as objects, not strings
	adminRole := roles[0]

	// Check permissions field (should be a map, not a string)
	permissions := adminRole.Get("permissions")
	assert.NotNil(t, permissions, "Permissions should not be nil")
	permissionsMap, ok := permissions.(map[string]interface{})
	assert.True(t, ok, "Permissions should be parsed as map[string]interface{}, got %T", permissions)
	assert.NotNil(t, permissionsMap["users"], "Should have users permissions")

	// Check metadata field (should be a map, not a string)
	metadata := adminRole.Get("metadata")
	assert.NotNil(t, metadata, "Metadata should not be nil")
	metadataMap, ok := metadata.(map[string]interface{})
	assert.True(t, ok, "Metadata should be parsed as map[string]interface{}, got %T", metadata)

	// Verify nested values from XLSX
	if usersPerms, ok := permissionsMap["users"].([]interface{}); ok {
		assert.Greater(t, len(usersPerms), 0, "Should have user permissions")
		assert.Contains(t, usersPerms, "create", "Should have create permission")
	}

	if maxUsers, ok := metadataMap["max_users"].(float64); ok {
		assert.Equal(t, float64(5), maxUsers, "Max users should be 5")
	}

	// Also verify other roles to ensure all JSON fields are parsed
	allRoles, err := mod.Get(model.QueryParam{})
	assert.Nil(t, err)
	assert.Greater(t, len(allRoles), 1, "Should have multiple roles")

	// Check that all roles have properly parsed JSON fields
	for _, role := range allRoles {
		roleID := role.Get("role_id")
		permissions := role.Get("permissions")
		if permissions != nil {
			_, ok := permissions.(map[string]interface{})
			assert.True(t, ok, "Role %s permissions should be parsed as map, got %T", roleID, permissions)
		}

		metadata := role.Get("metadata")
		if metadata != nil {
			_, ok := metadata.(map[string]interface{})
			assert.True(t, ok, "Role %s metadata should be parsed as map, got %T", roleID, metadata)
		}
	}
}
