package seed

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/yaoapp/gou/model"
	"github.com/yaoapp/gou/process"
	"github.com/yaoapp/yao/config"
	"github.com/yaoapp/yao/test"
)

func TestProcessSeedImportCSV(t *testing.T) {
	test.Prepare(t, config.Conf)
	defer test.Clean()

	// Ensure __yao.role model exists
	if !model.Exists("__yao.role") {
		t.Skip("__yao.role model not loaded, skipping test")
	}

	// Clear existing roles
	mod := model.Select("__yao.role")
	_, _ = mod.DestroyWhere(model.QueryParam{})

	// Test importing CSV file using process
	p, err := process.Of("seeds.import", "roles.csv", "__yao.role")
	if err != nil {
		t.Fatal(err)
	}

	result, err := p.Exec()
	if err != nil {
		t.Fatal(err)
	}

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

func TestProcessSeedImportJSON(t *testing.T) {
	test.Prepare(t, config.Conf)
	defer test.Clean()

	// Ensure __yao.role model exists
	if !model.Exists("__yao.role") {
		t.Skip("__yao.role model not loaded, skipping test")
	}

	// Clear existing roles
	mod := model.Select("__yao.role")
	_, _ = mod.DestroyWhere(model.QueryParam{})

	// Test importing JSON file using process
	p, err := process.Of("seeds.import", "roles.json", "__yao.role")
	if err != nil {
		t.Fatal(err)
	}

	result, err := p.Exec()
	if err != nil {
		t.Fatal(err)
	}

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

	// Check that JSON data was imported correctly
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

func TestProcessSeedImportXLSX(t *testing.T) {
	test.Prepare(t, config.Conf)
	defer test.Clean()

	// Ensure __yao.role model exists
	if !model.Exists("__yao.role") {
		t.Skip("__yao.role model not loaded, skipping test")
	}

	// Clear existing roles
	mod := model.Select("__yao.role")
	_, _ = mod.DestroyWhere(model.QueryParam{})

	// Test importing XLSX file using process
	p, err := process.Of("seeds.import", "roles.xlsx", "__yao.role")
	if err != nil {
		t.Fatal(err)
	}

	result, err := p.Exec()
	if err != nil {
		t.Fatal(err)
	}

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

func TestProcessSeedImportYao(t *testing.T) {
	test.Prepare(t, config.Conf)
	defer test.Clean()

	// Ensure __yao.role model exists
	if !model.Exists("__yao.role") {
		t.Skip("__yao.role model not loaded, skipping test")
	}

	// Clear existing roles
	mod := model.Select("__yao.role")
	_, _ = mod.DestroyWhere(model.QueryParam{})

	// Test importing JSONC file using process
	p, err := process.Of("seeds.import", "roles.yao", "__yao.role")
	if err != nil {
		t.Fatal(err)
	}

	result, err := p.Exec()
	if err != nil {
		t.Fatal(err)
	}

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

func TestProcessSeedImportWithBatchMode(t *testing.T) {
	test.Prepare(t, config.Conf)
	defer test.Clean()

	// Ensure __yao.role model exists
	if !model.Exists("__yao.role") {
		t.Skip("__yao.role model not loaded, skipping test")
	}

	// Clear existing roles
	mod := model.Select("__yao.role")
	_, _ = mod.DestroyWhere(model.QueryParam{})

	// Test importing with batch mode and custom chunk size
	options := map[string]interface{}{
		"chunk_size": 2,
		"duplicate":  "ignore",
		"mode":       "batch",
	}

	p, err := process.Of("seeds.import", "roles.json", "__yao.role", options)
	if err != nil {
		t.Fatal(err)
	}

	result, err := p.Exec()
	if err != nil {
		t.Fatal(err)
	}

	// Verify result
	assert.NotNil(t, result)
	resultMap, ok := result.(*ImportResult)
	assert.True(t, ok, "Result should be ImportResult")
	assert.Greater(t, resultMap.Success, 0, "Should have successful imports")

	// Verify data in database
	roles, err := mod.Get(model.QueryParam{})
	assert.Nil(t, err)
	assert.Greater(t, len(roles), 0, "Should have roles in database")
}

func TestProcessSeedImportWithEachMode(t *testing.T) {
	test.Prepare(t, config.Conf)
	defer test.Clean()

	// Ensure __yao.role model exists
	if !model.Exists("__yao.role") {
		t.Skip("__yao.role model not loaded, skipping test")
	}

	// Clear existing roles
	mod := model.Select("__yao.role")
	_, _ = mod.DestroyWhere(model.QueryParam{})

	// Test importing with each mode
	options := map[string]interface{}{
		"mode":      "each",
		"duplicate": "ignore",
	}

	p, err := process.Of("seeds.import", "roles.json", "__yao.role", options)
	if err != nil {
		t.Fatal(err)
	}

	result, err := p.Exec()
	if err != nil {
		t.Fatal(err)
	}

	// Verify result
	assert.NotNil(t, result)
	resultMap, ok := result.(*ImportResult)
	assert.True(t, ok, "Result should be ImportResult")
	assert.Greater(t, resultMap.Success, 0, "Should have successful imports")

	// Verify data in database
	roles, err := mod.Get(model.QueryParam{})
	assert.Nil(t, err)
	assert.Greater(t, len(roles), 0, "Should have roles in database")
}

func TestProcessSeedImportDuplicateStrategies(t *testing.T) {
	test.Prepare(t, config.Conf)
	defer test.Clean()

	// Ensure __yao.role model exists
	if !model.Exists("__yao.role") {
		t.Skip("__yao.role model not loaded, skipping test")
	}

	// Test ignore strategy
	t.Run("DuplicateIgnore", func(t *testing.T) {
		mod := model.Select("__yao.role")
		_, _ = mod.DestroyWhere(model.QueryParam{})

		// First import
		p, err := process.Of("seeds.import", "roles.json", "__yao.role")
		assert.NoError(t, err)
		result1, err := p.Exec()
		assert.NoError(t, err)
		resultMap1 := result1.(*ImportResult)
		firstSuccess := resultMap1.Success

		// Second import with ignore
		p, err = process.Of("seeds.import", "roles.json", "__yao.role", map[string]interface{}{
			"mode":      "each",
			"duplicate": "ignore",
		})
		assert.NoError(t, err)
		result2, err := p.Exec()
		assert.NoError(t, err)

		resultMap2 := result2.(*ImportResult)
		assert.Greater(t, resultMap2.Ignore, 0, "Should have ignored duplicates")

		// Verify count hasn't changed
		roles, err := mod.Get(model.QueryParam{})
		assert.Nil(t, err)
		assert.Equal(t, firstSuccess, len(roles), "Should have same number of roles")
	})

	// Test error strategy
	t.Run("DuplicateError", func(t *testing.T) {
		mod := model.Select("__yao.role")
		_, _ = mod.DestroyWhere(model.QueryParam{})

		// First import
		p, err := process.Of("seeds.import", "roles.json", "__yao.role")
		assert.NoError(t, err)
		_, err = p.Exec()
		assert.NoError(t, err)

		// Second import with error strategy
		p, err = process.Of("seeds.import", "roles.json", "__yao.role", map[string]interface{}{
			"mode":      "each",
			"duplicate": "error",
		})
		assert.NoError(t, err)
		result, err := p.Exec()
		assert.NoError(t, err)

		resultMap := result.(*ImportResult)
		assert.Greater(t, resultMap.Failure, 0, "Should have failures for duplicates")
	})
}

func TestProcessSeedImportInvalidArguments(t *testing.T) {
	test.Prepare(t, config.Conf)
	defer test.Clean()

	// Test with missing arguments
	t.Run("MissingArguments", func(t *testing.T) {
		p, err := process.Of("seeds.import", "roles.csv")
		assert.NoError(t, err)
		_, err = p.Exec()
		assert.Error(t, err, "Should fail with missing model argument")
	})

	// Test with invalid file
	t.Run("InvalidFile", func(t *testing.T) {
		p, err := process.Of("seeds.import", "nonexistent.csv", "__yao.role")
		assert.NoError(t, err)
		_, err = p.Exec()
		assert.Error(t, err, "Should fail with non-existent file")
	})

	// Test with invalid model
	t.Run("InvalidModel", func(t *testing.T) {
		p, err := process.Of("seeds.import", "roles.csv", "nonexistent.model")
		assert.NoError(t, err)
		_, err = p.Exec()
		assert.Error(t, err, "Should fail with non-existent model")
	})

	// Test with unsupported file format
	t.Run("UnsupportedFormat", func(t *testing.T) {
		p, err := process.Of("seeds.import", "roles.txt", "__yao.role")
		assert.NoError(t, err)
		_, err = p.Exec()
		assert.Error(t, err, "Should fail with unsupported file format")
	})
}

func TestProcessSeedImportOptions(t *testing.T) {
	test.Prepare(t, config.Conf)
	defer test.Clean()

	// Ensure __yao.role model exists
	if !model.Exists("__yao.role") {
		t.Skip("__yao.role model not loaded, skipping test")
	}

	// Test with various option types
	t.Run("OptionsAsMap", func(t *testing.T) {
		mod := model.Select("__yao.role")
		_, _ = mod.DestroyWhere(model.QueryParam{})

		options := map[string]interface{}{
			"chunk_size": 100,
			"duplicate":  "ignore",
			"mode":       "batch",
		}

		p, err := process.Of("seeds.import", "roles.csv", "__yao.role", options)
		assert.NoError(t, err)
		result, err := p.Exec()
		assert.NoError(t, err)
		assert.NotNil(t, result)
	})

	// Test with float64 chunk_size (JSON numbers)
	t.Run("OptionsWithFloat64", func(t *testing.T) {
		mod := model.Select("__yao.role")
		_, _ = mod.DestroyWhere(model.QueryParam{})

		options := map[string]interface{}{
			"chunk_size": float64(200),
		}

		p, err := process.Of("seeds.import", "roles.csv", "__yao.role", options)
		assert.NoError(t, err)
		result, err := p.Exec()
		assert.NoError(t, err)
		assert.NotNil(t, result)
	})

	// Test with partial options
	t.Run("PartialOptions", func(t *testing.T) {
		mod := model.Select("__yao.role")
		_, _ = mod.DestroyWhere(model.QueryParam{})

		options := map[string]interface{}{
			"mode": "each",
		}

		p, err := process.Of("seeds.import", "roles.csv", "__yao.role", options)
		assert.NoError(t, err)
		result, err := p.Exec()
		assert.NoError(t, err)
		assert.NotNil(t, result)
	})

	// Test with empty options
	t.Run("EmptyOptions", func(t *testing.T) {
		mod := model.Select("__yao.role")
		_, _ = mod.DestroyWhere(model.QueryParam{})

		options := map[string]interface{}{}

		p, err := process.Of("seeds.import", "roles.csv", "__yao.role", options)
		assert.NoError(t, err)
		result, err := p.Exec()
		assert.NoError(t, err)
		assert.NotNil(t, result)
	})
}

func TestProcessSeedImportResultStructure(t *testing.T) {
	test.Prepare(t, config.Conf)
	defer test.Clean()

	// Ensure __yao.role model exists
	if !model.Exists("__yao.role") {
		t.Skip("__yao.role model not loaded, skipping test")
	}

	// Clear existing roles
	mod := model.Select("__yao.role")
	_, _ = mod.DestroyWhere(model.QueryParam{})

	// Import data
	p, err := process.Of("seeds.import", "roles.csv", "__yao.role")
	if err != nil {
		t.Fatal(err)
	}

	result, err := p.Exec()
	if err != nil {
		t.Fatal(err)
	}

	// Verify result structure
	resultMap, ok := result.(*ImportResult)
	assert.True(t, ok, "Result should be ImportResult type")

	// Check all fields exist and have proper types
	assert.GreaterOrEqual(t, resultMap.Total, 0, "Total should be non-negative")
	assert.GreaterOrEqual(t, resultMap.Success, 0, "Success should be non-negative")
	assert.GreaterOrEqual(t, resultMap.Failure, 0, "Failure should be non-negative")
	assert.GreaterOrEqual(t, resultMap.Ignore, 0, "Ignore should be non-negative")
	assert.NotNil(t, resultMap.Errors, "Errors should not be nil")

	// Verify total = success + failure + ignore
	assert.Equal(t, resultMap.Total, resultMap.Success+resultMap.Failure+resultMap.Ignore,
		"Total should equal sum of success, failure, and ignore")
}

func TestProcessSeedImportMultipleFiles(t *testing.T) {
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
	t.Run("ImportCSV", func(t *testing.T) {
		p, err := process.Of("seeds.import", "roles.csv", "__yao.role")
		assert.NoError(t, err)
		result, err := p.Exec()
		assert.NoError(t, err)
		resultMap := result.(*ImportResult)
		assert.Greater(t, resultMap.Success, 0)
	})

	// Clear and import JSON
	t.Run("ImportJSON", func(t *testing.T) {
		_, _ = mod.DestroyWhere(model.QueryParam{})
		p, err := process.Of("seeds.import", "roles.json", "__yao.role")
		assert.NoError(t, err)
		result, err := p.Exec()
		assert.NoError(t, err)
		resultMap := result.(*ImportResult)
		assert.Greater(t, resultMap.Success, 0)
	})

	// Clear and import XLSX
	t.Run("ImportXLSX", func(t *testing.T) {
		_, _ = mod.DestroyWhere(model.QueryParam{})
		p, err := process.Of("seeds.import", "roles.xlsx", "__yao.role")
		assert.NoError(t, err)
		result, err := p.Exec()
		assert.NoError(t, err)
		resultMap := result.(*ImportResult)
		assert.Greater(t, resultMap.Success, 0)
	})

	// Clear and import Yao
	t.Run("ImportYao", func(t *testing.T) {
		_, _ = mod.DestroyWhere(model.QueryParam{})
		p, err := process.Of("seeds.import", "roles.yao", "__yao.role")
		assert.NoError(t, err)
		result, err := p.Exec()
		assert.NoError(t, err)
		resultMap := result.(*ImportResult)
		assert.Greater(t, resultMap.Success, 0)
	})
}
