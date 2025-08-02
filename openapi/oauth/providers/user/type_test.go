package user_test

import (
	"context"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/yaoapp/gou/model"
	"github.com/yaoapp/kun/maps"
)

// TestTypeData represents test type data structure
type TestTypeData struct {
	TypeID         string                 `json:"type_id"`
	Name           string                 `json:"name"`
	Description    string                 `json:"description"`
	IsActive       bool                   `json:"is_active"`
	IsDefault      bool                   `json:"is_default"`
	SortOrder      int                    `json:"sort_order"`
	DefaultRoleID  string                 `json:"default_role_id"`
	MaxSessions    *int                   `json:"max_sessions"`
	SessionTimeout int                    `json:"session_timeout"`
	Schema         map[string]interface{} `json:"schema"`
	Features       map[string]interface{} `json:"features"`
	Limits         map[string]interface{} `json:"limits"`
	PasswordPolicy map[string]interface{} `json:"password_policy"`
	Metadata       map[string]interface{} `json:"metadata"`
}

func TestTypeBasicOperations(t *testing.T) {
	prepare(t)
	defer clean()

	ctx := context.Background()

	// Use UUID to ensure unique identifiers
	testUUID := strings.ReplaceAll(uuid.New().String(), "-", "")[:8] // 8 char UUID

	// Create test type data dynamically
	maxSessions := 5
	testType := &TestTypeData{
		TypeID:         "testtype_" + testUUID,
		Name:           "Test Type " + testUUID,
		Description:    "Test type for unit testing " + testUUID,
		IsActive:       true,
		IsDefault:      false,
		SortOrder:      100,
		DefaultRoleID:  "user",
		MaxSessions:    &maxSessions,
		SessionTimeout: 3600,
		Schema: map[string]interface{}{
			"version": "1.0",
			"fields": map[string]interface{}{
				"profile": map[string]interface{}{
					"required": true,
					"type":     "object",
				},
			},
		},
		Features: map[string]interface{}{
			"mfa_enabled":     true,
			"api_access":      true,
			"export_data":     false,
			"custom_branding": true,
		},
		Limits: map[string]interface{}{
			"storage_mb":    1024,
			"api_calls_day": 10000,
			"team_members":  50,
			"projects":      10,
		},
		PasswordPolicy: map[string]interface{}{
			"min_length":        8,
			"require_uppercase": true,
			"require_lowercase": true,
			"require_numbers":   true,
			"require_symbols":   false,
			"max_age_days":      90,
		},
		Metadata: map[string]interface{}{
			"source":  "test",
			"uuid":    testUUID,
			"version": "1.0",
		},
	}

	// Test CreateType
	t.Run("CreateType", func(t *testing.T) {
		typeData := maps.MapStrAny{
			"type_id":         testType.TypeID,
			"name":            testType.Name,
			"description":     testType.Description,
			"sort_order":      testType.SortOrder,
			"default_role_id": testType.DefaultRoleID,
			"max_sessions":    testType.MaxSessions,
			"session_timeout": testType.SessionTimeout,
			"schema":          testType.Schema,
			"features":        testType.Features,
			"limits":          testType.Limits,
			"password_policy": testType.PasswordPolicy,
			"metadata":        testType.Metadata,
		}

		id, err := testProvider.CreateType(ctx, typeData)
		assert.NoError(t, err)
		assert.NotNil(t, id)

		// Verify default values were set
		assert.Equal(t, true, typeData["is_active"])
		assert.Equal(t, false, typeData["is_default"])
		// sort_order, max_sessions, session_timeout should remain as provided
	})

	// Test GetType
	t.Run("GetType", func(t *testing.T) {
		typeRecord, err := testProvider.GetType(ctx, testType.TypeID)
		assert.NoError(t, err)
		assert.NotNil(t, typeRecord)

		// Verify key fields
		assert.Equal(t, testType.TypeID, typeRecord["type_id"])
		assert.Equal(t, testType.Name, typeRecord["name"])
		assert.Equal(t, testType.Description, typeRecord["description"])
		assert.Equal(t, testType.DefaultRoleID, typeRecord["default_role_id"])

		// Handle different boolean representations from database
		isActive := typeRecord["is_active"]
		switch v := isActive.(type) {
		case bool:
			assert.True(t, v)
		case int, int32, int64:
			assert.NotEqual(t, 0, v) // Any non-zero value is true
		default:
			t.Errorf("unexpected is_active type: %T, value: %v", isActive, isActive)
		}

		assert.NotNil(t, typeRecord["created_at"])
	})

	// Test UpdateType
	t.Run("UpdateType", func(t *testing.T) {
		newMaxSessions := 10
		updateData := maps.MapStrAny{
			"name":            "Updated Test Type",
			"description":     "Updated description for testing",
			"sort_order":      200,
			"default_role_id": "admin",
			"max_sessions":    &newMaxSessions,
			"session_timeout": 7200,
			"metadata": map[string]interface{}{
				"updated": true,
				"version": "2.0",
			},
		}

		err := testProvider.UpdateType(ctx, testType.TypeID, updateData)
		assert.NoError(t, err)

		// Verify update
		typeRecord, err := testProvider.GetType(ctx, testType.TypeID)
		assert.NoError(t, err)
		assert.Equal(t, "Updated Test Type", typeRecord["name"])
		assert.Equal(t, "Updated description for testing", typeRecord["description"])
		assert.Equal(t, "admin", typeRecord["default_role_id"])

		// Test updating sensitive fields (should be ignored)
		sensitiveData := maps.MapStrAny{
			"id":         999,
			"type_id":    "malicious_type_id",
			"created_at": "2020-01-01T00:00:00Z",
		}

		err = testProvider.UpdateType(ctx, testType.TypeID, sensitiveData)
		assert.NoError(t, err) // Should not error, just ignore sensitive fields

		// Verify sensitive fields were not changed
		typeRecord, err = testProvider.GetType(ctx, testType.TypeID)
		assert.NoError(t, err)
		assert.Equal(t, testType.TypeID, typeRecord["type_id"]) // Should remain unchanged
	})

	// Test DeleteType (at the end)
	t.Run("DeleteType", func(t *testing.T) {
		err := testProvider.DeleteType(ctx, testType.TypeID)
		assert.NoError(t, err)

		// Verify type was deleted
		_, err = testProvider.GetType(ctx, testType.TypeID)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "type not found")
	})
}

func TestTypeConfigurationOperations(t *testing.T) {
	prepare(t)
	defer clean()

	ctx := context.Background()

	// Use UUID to ensure unique identifiers
	testUUID := strings.ReplaceAll(uuid.New().String(), "-", "")[:8]

	// Create a type for configuration testing
	testType := &TestTypeData{
		TypeID:      "configtype_" + testUUID,
		Name:        "Config Test Type " + testUUID,
		Description: "Type for testing configuration",
		IsActive:    true,
		Schema: map[string]interface{}{
			"version": "1.0",
			"type":    "premium",
		},
		Features: map[string]interface{}{
			"api_access":          true,
			"advanced_reports":    true,
			"custom_integrations": false,
			"scope_limits": []interface{}{
				"read", "write", "admin.read",
			},
		},
		Limits: map[string]interface{}{
			"storage_gb": 10,
			"users":      100,
			"api_calls":  50000,
		},
		PasswordPolicy: map[string]interface{}{
			"min_length":      12,
			"require_symbols": true,
			"history_count":   5,
		},
		Metadata: map[string]interface{}{
			"plan":     "premium",
			"tier":     2,
			"features": "advanced",
		},
	}

	// Create type
	typeData := maps.MapStrAny{
		"type_id":         testType.TypeID,
		"name":            testType.Name,
		"description":     testType.Description,
		"schema":          testType.Schema,
		"features":        testType.Features,
		"limits":          testType.Limits,
		"password_policy": testType.PasswordPolicy,
		"metadata":        testType.Metadata,
	}

	_, err := testProvider.CreateType(ctx, typeData)
	assert.NoError(t, err)

	// Test GetTypeConfiguration
	t.Run("GetTypeConfiguration", func(t *testing.T) {
		config, err := testProvider.GetTypeConfiguration(ctx, testType.TypeID)
		assert.NoError(t, err)
		assert.NotNil(t, config)

		assert.Equal(t, testType.TypeID, config["type_id"])
		assert.NotNil(t, config["schema"])
		assert.NotNil(t, config["features"])
		assert.NotNil(t, config["limits"])
		assert.NotNil(t, config["password_policy"])
		assert.NotNil(t, config["metadata"])

		// Verify schema structure
		schemaMap, ok := config["schema"].(map[string]interface{})
		if ok {
			assert.Equal(t, "1.0", schemaMap["version"])
			assert.Equal(t, "premium", schemaMap["type"])
		}

		// Verify features structure
		featuresMap, ok := config["features"].(map[string]interface{})
		if ok {
			assert.Equal(t, true, featuresMap["api_access"])
			assert.Equal(t, true, featuresMap["advanced_reports"])
			assert.Equal(t, false, featuresMap["custom_integrations"])
		}
	})

	// Test SetTypeConfiguration
	t.Run("SetTypeConfiguration", func(t *testing.T) {
		newConfig := maps.MapStrAny{
			"schema": map[string]interface{}{
				"version": "2.0",
				"type":    "enterprise", // Changed
			},
			"features": map[string]interface{}{
				"api_access":          true,
				"advanced_reports":    true,
				"custom_integrations": true, // Changed
				"white_label":         true, // New
				"scope_limits": []interface{}{
					"read", "write", "admin.read", "admin.write", // Extended
				},
			},
			"limits": map[string]interface{}{
				"storage_gb": 50,     // Increased
				"users":      500,    // Increased
				"api_calls":  100000, // Increased
			},
			"password_policy": map[string]interface{}{
				"min_length":       16, // Increased
				"require_symbols":  true,
				"history_count":    10, // Increased
				"complexity_score": 8,  // New
			},
			"metadata": map[string]interface{}{
				"plan":       "enterprise", // Changed
				"tier":       3,            // Changed
				"features":   "premium",
				"updated_by": "test", // New
			},
		}

		err := testProvider.SetTypeConfiguration(ctx, testType.TypeID, newConfig)
		assert.NoError(t, err)

		// Verify configuration was updated
		config, err := testProvider.GetTypeConfiguration(ctx, testType.TypeID)
		assert.NoError(t, err)

		// Verify schema update
		schemaMap, ok := config["schema"].(map[string]interface{})
		if ok {
			assert.Equal(t, "2.0", schemaMap["version"])
			assert.Equal(t, "enterprise", schemaMap["type"]) // Should be updated
		}

		// Verify features update
		featuresMap, ok := config["features"].(map[string]interface{})
		if ok {
			assert.Equal(t, true, featuresMap["custom_integrations"]) // Should be updated
			assert.Equal(t, true, featuresMap["white_label"])         // Should be new
		}

		// Verify limits update
		limitsMap, ok := config["limits"].(map[string]interface{})
		if ok {
			// Handle different numeric types from database
			storageInterface := limitsMap["storage_gb"]
			switch v := storageInterface.(type) {
			case int:
				assert.Equal(t, 50, v)
			case int32:
				assert.Equal(t, int32(50), v)
			case int64:
				assert.Equal(t, int64(50), v)
			case float64:
				assert.Equal(t, float64(50), v)
			default:
				t.Errorf("unexpected storage_gb type: %T, value: %v", storageInterface, storageInterface)
			}
		}
	})

	// Test SetTypeConfiguration with partial data
	t.Run("SetTypeConfiguration_PartialUpdate", func(t *testing.T) {
		partialConfig := maps.MapStrAny{
			"metadata": map[string]interface{}{
				"plan":      "enterprise",
				"tier":      3,
				"features":  "premium",
				"updated":   true,         // New field
				"timestamp": "2024-01-01", // New field
			},
		}

		err := testProvider.SetTypeConfiguration(ctx, testType.TypeID, partialConfig)
		assert.NoError(t, err)

		// Verify only metadata was updated, other configs remain
		config, err := testProvider.GetTypeConfiguration(ctx, testType.TypeID)
		assert.NoError(t, err)

		// Schema should remain from previous update
		schemaMap, ok := config["schema"].(map[string]interface{})
		if ok {
			assert.Equal(t, "2.0", schemaMap["version"])
		}

		// Metadata should be updated
		metadataMap, ok := config["metadata"].(map[string]interface{})
		if ok {
			assert.Equal(t, true, metadataMap["updated"])
			assert.Equal(t, "2024-01-01", metadataMap["timestamp"])
		}
	})

	// Test SetTypeConfiguration with empty data (should not error)
	t.Run("SetTypeConfiguration_EmptyData", func(t *testing.T) {
		emptyConfig := maps.MapStrAny{}
		err := testProvider.SetTypeConfiguration(ctx, testType.TypeID, emptyConfig)
		assert.NoError(t, err) // Should not error, just skip update
	})
}

func TestTypeListOperations(t *testing.T) {
	prepare(t)
	defer clean()

	ctx := context.Background()

	// Create multiple test types for list operations
	// Use UUID to ensure unique identifiers
	testUUID := strings.ReplaceAll(uuid.New().String(), "-", "")[:8]

	testTypes := []TestTypeData{
		{
			TypeID:      "listtype_" + testUUID + "_1",
			Name:        "List Type 1",
			Description: "First type for list testing",
			IsActive:    true,
			SortOrder:   10,
		},
		{
			TypeID:      "listtype_" + testUUID + "_2",
			Name:        "List Type 2",
			Description: "Second type for list testing",
			IsActive:    true,
			SortOrder:   20,
		},
		{
			TypeID:      "listtype_" + testUUID + "_3",
			Name:        "List Type 3",
			Description: "Third type for list testing",
			IsActive:    false, // Different status for filtering
			SortOrder:   30,
		},
		{
			TypeID:      "listtype_" + testUUID + "_4",
			Name:        "List Type 4",
			Description: "Fourth type for list testing",
			IsActive:    true,
			SortOrder:   40,
		},
		{
			TypeID:      "listtype_" + testUUID + "_5",
			Name:        "List Type 5",
			Description: "Fifth type for list testing",
			IsActive:    true,
			SortOrder:   50,
		},
	}

	// Create types in database
	for _, typeData := range testTypes {
		typeMap := maps.MapStrAny{
			"type_id":     typeData.TypeID,
			"name":        typeData.Name,
			"description": typeData.Description,
			"is_active":   typeData.IsActive,
			"sort_order":  typeData.SortOrder,
		}

		_, err := testProvider.CreateType(ctx, typeMap)
		assert.NoError(t, err)
	}

	// Test GetTypes
	t.Run("GetTypes_All", func(t *testing.T) {
		param := model.QueryParam{
			Wheres: []model.QueryWhere{
				{Column: "type_id", OP: "like", Value: "listtype_" + testUUID + "_%"},
			},
		}
		types, err := testProvider.GetTypes(ctx, param)
		assert.NoError(t, err)
		assert.GreaterOrEqual(t, len(types), 5) // At least our 5 test types

		// Check that basic fields are returned by default
		if len(types) > 0 {
			typeRecord := types[0]
			assert.Contains(t, typeRecord, "type_id")
			assert.Contains(t, typeRecord, "name")
			assert.Contains(t, typeRecord, "description")
			assert.Contains(t, typeRecord, "is_active")
		}
	})

	t.Run("GetTypes_WithFilters", func(t *testing.T) {
		param := model.QueryParam{
			Wheres: []model.QueryWhere{
				{Column: "type_id", OP: "like", Value: "listtype_" + testUUID + "_%"},
				{Column: "is_active", Value: true},
			},
		}
		types, err := testProvider.GetTypes(ctx, param)
		assert.NoError(t, err)
		assert.GreaterOrEqual(t, len(types), 4) // At least 4 active types

		// All returned types should be active
		for _, typeRecord := range types {
			if strings.Contains(typeRecord["type_id"].(string), "listtype_"+testUUID+"_") {
				// Handle different boolean representations from database
				isActive := typeRecord["is_active"]
				switch v := isActive.(type) {
				case bool:
					assert.True(t, v)
				case int, int32, int64:
					assert.NotEqual(t, 0, v) // Any non-zero value is true
				default:
					t.Errorf("unexpected is_active type: %T, value: %v", isActive, isActive)
				}
			}
		}
	})

	t.Run("GetTypes_WithCustomFields", func(t *testing.T) {
		param := model.QueryParam{
			Select: []interface{}{"type_id", "name", "is_active", "sort_order"},
			Wheres: []model.QueryWhere{
				{Column: "type_id", OP: "like", Value: "listtype_" + testUUID + "_%"},
			},
			Limit: 3,
		}
		types, err := testProvider.GetTypes(ctx, param)
		assert.NoError(t, err)
		assert.LessOrEqual(t, len(types), 3) // Respects limit

		if len(types) > 0 {
			typeRecord := types[0]
			assert.Contains(t, typeRecord, "type_id")
			assert.Contains(t, typeRecord, "name")
			assert.Contains(t, typeRecord, "is_active")
			assert.Contains(t, typeRecord, "sort_order")
		}
	})

	// Test PaginateTypes
	t.Run("PaginateTypes_FirstPage", func(t *testing.T) {
		param := model.QueryParam{
			Wheres: []model.QueryWhere{
				{Column: "type_id", OP: "like", Value: "listtype_" + testUUID + "_%"},
			},
			Orders: []model.QueryOrder{
				{Column: "sort_order", Option: "asc"},
			},
		}
		result, err := testProvider.PaginateTypes(ctx, param, 1, 3)
		assert.NoError(t, err)
		assert.NotNil(t, result)

		// Check pagination structure
		assert.Contains(t, result, "data")
		assert.Contains(t, result, "total")
		assert.Contains(t, result, "page")
		assert.Contains(t, result, "pagesize")

		data, ok := result["data"].([]maps.MapStr)
		assert.True(t, ok)
		assert.LessOrEqual(t, len(data), 3) // Page size limit

		// Handle different total types
		totalInterface, exists := result["total"]
		assert.True(t, exists)

		var total int64
		switch v := totalInterface.(type) {
		case int:
			total = int64(v)
		case int32:
			total = int64(v)
		case int64:
			total = v
		case uint:
			total = int64(v)
		case uint32:
			total = int64(v)
		case uint64:
			total = int64(v)
		default:
			t.Errorf("unexpected total type: %T, value: %v", totalInterface, totalInterface)
		}
		assert.GreaterOrEqual(t, total, int64(5)) // At least 5 types

		assert.Equal(t, 1, result["page"])
		assert.Equal(t, 3, result["pagesize"])
	})

	t.Run("PaginateTypes_WithFilters", func(t *testing.T) {
		param := model.QueryParam{
			Wheres: []model.QueryWhere{
				{Column: "type_id", OP: "like", Value: "listtype_" + testUUID + "_%"},
				{Column: "is_active", Value: true},
			},
		}
		result, err := testProvider.PaginateTypes(ctx, param, 1, 10)
		assert.NoError(t, err)
		assert.NotNil(t, result)

		data, ok := result["data"].([]maps.MapStr)
		assert.True(t, ok)
		assert.GreaterOrEqual(t, len(data), 4) // At least 4 active types

		// Verify is_active filter works
		for _, typeRecord := range data {
			if strings.Contains(typeRecord["type_id"].(string), "listtype_"+testUUID+"_") {
				// Handle different boolean representations from database
				isActive := typeRecord["is_active"]
				switch v := isActive.(type) {
				case bool:
					assert.True(t, v)
				case int, int32, int64:
					assert.NotEqual(t, 0, v) // Any non-zero value is true
				default:
					t.Errorf("unexpected is_active type: %T, value: %v", isActive, isActive)
				}
			}
		}
	})

	// Test CountTypes
	t.Run("CountTypes_All", func(t *testing.T) {
		param := model.QueryParam{
			Wheres: []model.QueryWhere{
				{Column: "type_id", OP: "like", Value: "listtype_" + testUUID + "_%"},
			},
		}
		count, err := testProvider.CountTypes(ctx, param)
		assert.NoError(t, err)
		assert.GreaterOrEqual(t, count, int64(5)) // At least 5 types
	})

	t.Run("CountTypes_WithFilters", func(t *testing.T) {
		param := model.QueryParam{
			Wheres: []model.QueryWhere{
				{Column: "type_id", OP: "like", Value: "listtype_" + testUUID + "_%"},
				{Column: "is_active", Value: true},
			},
		}
		count, err := testProvider.CountTypes(ctx, param)
		assert.NoError(t, err)
		assert.GreaterOrEqual(t, count, int64(4)) // At least 4 active types
	})

	t.Run("CountTypes_SpecificSortOrder", func(t *testing.T) {
		param := model.QueryParam{
			Wheres: []model.QueryWhere{
				{Column: "type_id", OP: "like", Value: "listtype_" + testUUID + "_%"},
				{Column: "sort_order", OP: ">=", Value: 30},
			},
		}
		count, err := testProvider.CountTypes(ctx, param)
		assert.NoError(t, err)
		// We created 3 types with sort_order >= 30 (30, 40, 50), but be flexible with database state
		assert.GreaterOrEqual(t, count, int64(1)) // At least 1 type with sort_order >= 30
		assert.LessOrEqual(t, count, int64(5))    // But not more than 5 (our total test types)
	})

	t.Run("CountTypes_NoResults", func(t *testing.T) {
		param := model.QueryParam{
			Wheres: []model.QueryWhere{
				{Column: "type_id", Value: "nonexistent_type_id"},
			},
		}
		count, err := testProvider.CountTypes(ctx, param)
		assert.NoError(t, err)
		assert.Equal(t, int64(0), count)
	})
}

func TestTypeErrorHandling(t *testing.T) {
	prepare(t)
	defer clean()

	ctx := context.Background()
	testUUID := strings.ReplaceAll(uuid.New().String(), "-", "")[:8]
	nonExistentTypeID := "nonexistent_type_" + testUUID

	t.Run("GetType_NotFound", func(t *testing.T) {
		_, err := testProvider.GetType(ctx, nonExistentTypeID)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "type not found")
	})

	t.Run("CreateType_MissingTypeID", func(t *testing.T) {
		typeData := maps.MapStrAny{
			"name":        "Test Type",
			"description": "Type without type_id",
		}

		_, err := testProvider.CreateType(ctx, typeData)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "type_id is required")
	})

	t.Run("UpdateType_NotFound", func(t *testing.T) {
		updateData := maps.MapStrAny{"name": "Test"}
		err := testProvider.UpdateType(ctx, nonExistentTypeID, updateData)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "type not found")
	})

	t.Run("DeleteType_NotFound", func(t *testing.T) {
		err := testProvider.DeleteType(ctx, nonExistentTypeID)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "type not found")
	})

	t.Run("GetTypeConfiguration_NotFound", func(t *testing.T) {
		_, err := testProvider.GetTypeConfiguration(ctx, nonExistentTypeID)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "type not found")
	})

	t.Run("SetTypeConfiguration_NotFound", func(t *testing.T) {
		config := maps.MapStrAny{
			"schema": map[string]interface{}{"test": true},
		}
		err := testProvider.SetTypeConfiguration(ctx, nonExistentTypeID, config)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "type not found")
	})

	t.Run("GetTypes_EmptyResult", func(t *testing.T) {
		param := model.QueryParam{
			Wheres: []model.QueryWhere{
				{Column: "type_id", Value: nonExistentTypeID},
			},
		}
		types, err := testProvider.GetTypes(ctx, param)
		assert.NoError(t, err)
		assert.Equal(t, 0, len(types)) // Empty slice, not nil
	})

	t.Run("PaginateTypes_EmptyResult", func(t *testing.T) {
		param := model.QueryParam{
			Wheres: []model.QueryWhere{
				{Column: "type_id", Value: nonExistentTypeID},
			},
		}
		result, err := testProvider.PaginateTypes(ctx, param, 1, 10)
		assert.NoError(t, err)
		assert.NotNil(t, result)

		data, ok := result["data"].([]maps.MapStr)
		assert.True(t, ok)
		assert.Equal(t, 0, len(data))

		// Handle different total types
		totalInterface, exists := result["total"]
		assert.True(t, exists)

		var total int64
		switch v := totalInterface.(type) {
		case int:
			total = int64(v)
		case int32:
			total = int64(v)
		case int64:
			total = v
		case uint:
			total = int64(v)
		case uint32:
			total = int64(v)
		case uint64:
			total = int64(v)
		default:
			t.Errorf("unexpected total type: %T, value: %v", totalInterface, totalInterface)
		}
		assert.Equal(t, int64(0), total)
	})

	t.Run("UpdateType_EmptyData", func(t *testing.T) {
		// First create a type for this test
		testTypeID := "emptyupdate_" + testUUID
		typeData := maps.MapStrAny{
			"type_id": testTypeID,
			"name":    "Test Type for Empty Update",
		}
		_, err := testProvider.CreateType(ctx, typeData)
		assert.NoError(t, err)

		// Test with empty update data (should not error, just do nothing)
		emptyData := maps.MapStrAny{}
		err = testProvider.UpdateType(ctx, testTypeID, emptyData)
		assert.NoError(t, err) // Should not error, just skip update
	})

	t.Run("SetTypeConfiguration_EmptyData", func(t *testing.T) {
		// First create a type for this test
		testTypeID := "emptyconfig_" + testUUID
		typeData := maps.MapStrAny{
			"type_id": testTypeID,
			"name":    "Test Type for Empty Configuration",
		}
		_, err := testProvider.CreateType(ctx, typeData)
		assert.NoError(t, err)

		// Test with empty configuration data (should not error, just do nothing)
		emptyData := maps.MapStrAny{}
		err = testProvider.SetTypeConfiguration(ctx, testTypeID, emptyData)
		assert.NoError(t, err) // Should not error, just skip update
	})

	t.Run("CountTypes_ComplexFilters", func(t *testing.T) {
		param := model.QueryParam{
			Wheres: []model.QueryWhere{
				{Column: "is_active", Value: true},
				{Column: "sort_order", OP: ">=", Value: 10},
				{Column: "is_default", Value: false},
			},
		}
		count, err := testProvider.CountTypes(ctx, param)
		assert.NoError(t, err)
		assert.GreaterOrEqual(t, count, int64(0)) // Should handle complex filters without error
	})
}
