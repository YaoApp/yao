package types

import (
	"reflect"
	"testing"
)

func TestValidateAssistantFields(t *testing.T) {
	t.Run("EmptyInput_ReturnsEmptySlice", func(t *testing.T) {
		result := ValidateAssistantFields([]string{})
		if len(result) != 0 {
			t.Errorf("Expected empty slice, got %v", result)
		}
	})

	t.Run("NilInput_ReturnsEmptySlice", func(t *testing.T) {
		result := ValidateAssistantFields(nil)
		if len(result) != 0 {
			t.Errorf("Expected empty slice, got %v", result)
		}
	})

	t.Run("ValidFields_ReturnsFiltered", func(t *testing.T) {
		input := []string{"assistant_id", "name", "type"}
		result := ValidateAssistantFields(input)
		expected := []string{"assistant_id", "name", "type"}
		if !reflect.DeepEqual(result, expected) {
			t.Errorf("Expected %v, got %v", expected, result)
		}
	})

	t.Run("MixedValidInvalidFields_ReturnsOnlyValid", func(t *testing.T) {
		input := []string{"assistant_id", "invalid_field", "name", "malicious_column"}
		result := ValidateAssistantFields(input)
		expected := []string{"assistant_id", "name"}
		if !reflect.DeepEqual(result, expected) {
			t.Errorf("Expected %v, got %v", expected, result)
		}
	})

	t.Run("AllInvalidFields_ReturnsDefaultFields", func(t *testing.T) {
		input := []string{"invalid1", "invalid2", "malicious"}
		result := ValidateAssistantFields(input)
		if !reflect.DeepEqual(result, AssistantDefaultFields) {
			t.Errorf("Expected default fields when all invalid, got %v", result)
		}
	})

	t.Run("PermissionFields_AreAllowed", func(t *testing.T) {
		input := []string{"__yao_created_by", "__yao_team_id", "assistant_id"}
		result := ValidateAssistantFields(input)
		expected := []string{"__yao_created_by", "__yao_team_id", "assistant_id"}
		if !reflect.DeepEqual(result, expected) {
			t.Errorf("Expected %v, got %v", expected, result)
		}
	})

	t.Run("AllAllowedFields_AreInWhitelist", func(t *testing.T) {
		// Verify all default fields are in the allowed list
		for _, field := range AssistantDefaultFields {
			if !AssistantAllowedFields[field] {
				t.Errorf("Default field %s is not in allowed fields", field)
			}
		}
	})

	t.Run("SQLInjectionAttempt_IsFiltered", func(t *testing.T) {
		input := []string{"assistant_id", "name; DROP TABLE assistants;--", "type"}
		result := ValidateAssistantFields(input)
		expected := []string{"assistant_id", "type"}
		if !reflect.DeepEqual(result, expected) {
			t.Errorf("Expected SQL injection attempt to be filtered, got %v", result)
		}
	})

	t.Run("DuplicateFields_AreKept", func(t *testing.T) {
		input := []string{"assistant_id", "name", "assistant_id", "name"}
		result := ValidateAssistantFields(input)
		// Duplicates should be kept as-is (validation doesn't deduplicate)
		expected := []string{"assistant_id", "name", "assistant_id", "name"}
		if !reflect.DeepEqual(result, expected) {
			t.Errorf("Expected %v, got %v", expected, result)
		}
	})
}

func TestAssistantAllowedFields(t *testing.T) {
	t.Run("ContainsBasicFields", func(t *testing.T) {
		requiredFields := []string{
			"assistant_id",
			"name",
			"type",
			"connector",
			"description",
		}
		for _, field := range requiredFields {
			if !AssistantAllowedFields[field] {
				t.Errorf("Required field %s is missing from allowed fields", field)
			}
		}
	})

	t.Run("ContainsPermissionFields", func(t *testing.T) {
		permissionFields := []string{
			"__yao_created_by",
			"__yao_updated_by",
			"__yao_team_id",
			"__yao_tenant_id",
		}
		for _, field := range permissionFields {
			if !AssistantAllowedFields[field] {
				t.Errorf("Permission field %s is missing from allowed fields", field)
			}
		}
	})

	t.Run("ContainsComplexFields", func(t *testing.T) {
		complexFields := []string{
			"options",
			"prompts",
			"prompt_presets",
			"disable_global_prompts",
			"workflow",
			"kb",
			"db",
			"mcp",
			"placeholder",
			"locales",
			"uses",
			"connector_options",
			"source",
			"modes",
			"default_mode",
		}
		for _, field := range complexFields {
			if !AssistantAllowedFields[field] {
				t.Errorf("Complex field %s is missing from allowed fields", field)
			}
		}
	})
}

func TestAssistantDefaultFields(t *testing.T) {
	t.Run("ContainsEssentialFields", func(t *testing.T) {
		essentialFields := []string{
			"assistant_id",
			"name",
			"type",
			"kb",               // Knowledge base is essential for assistant functionality
			"db",               // Database is essential for assistant functionality
			"mcp",              // MCP servers are essential for assistant functionality
			"modes",            // Supported modes are essential for mode filtering
			"default_mode",     // Default mode is essential for mode selection
			"__yao_created_by", // Permission fields are essential for access control
			"__yao_updated_by",
			"__yao_team_id",
			"__yao_tenant_id",
		}

		defaultFieldsMap := make(map[string]bool)
		for _, field := range AssistantDefaultFields {
			defaultFieldsMap[field] = true
		}

		for _, field := range essentialFields {
			if !defaultFieldsMap[field] {
				t.Errorf("Essential field %s is missing from default fields", field)
			}
		}
	})

	t.Run("DoesNotContainSensitiveFields", func(t *testing.T) {
		// Default fields should not include complex/large fields by default
		// Note: kb, db, mcp, tags, modes, and default_mode are lightweight and included in defaults
		sensitiveFields := []string{
			"options",
			"prompts",
			"prompt_presets",
			"workflow",
			"placeholder",
			"locales",
			"uses",
			"connector_options",
			"source",
		}

		defaultFieldsMap := make(map[string]bool)
		for _, field := range AssistantDefaultFields {
			defaultFieldsMap[field] = true
		}

		for _, field := range sensitiveFields {
			if defaultFieldsMap[field] {
				t.Errorf("Large/complex field %s should not be in default fields", field)
			}
		}
	})
}

func TestAssistantFullFields(t *testing.T) {
	t.Run("ContainsAllAllowedFields", func(t *testing.T) {
		// Full fields should contain all fields from allowed fields
		fullFieldsMap := make(map[string]bool)
		for _, field := range AssistantFullFields {
			fullFieldsMap[field] = true
		}

		for field := range AssistantAllowedFields {
			if field == "id" {
				// "id" is an alias for "assistant_id", skip
				continue
			}
			if !fullFieldsMap[field] {
				t.Errorf("Allowed field %s is missing from full fields", field)
			}
		}
	})

	t.Run("AllFieldsAreAllowed", func(t *testing.T) {
		// All fields in full list should be in allowed fields
		for _, field := range AssistantFullFields {
			if !AssistantAllowedFields[field] {
				t.Errorf("Full field %s is not in allowed fields", field)
			}
		}
	})

	t.Run("ContainsComplexFields", func(t *testing.T) {
		// Full fields should include all complex/large fields
		complexFields := []string{
			"options",
			"prompts",
			"prompt_presets",
			"disable_global_prompts",
			"workflow",
			"kb",
			"db",
			"mcp",
			"placeholder",
			"locales",
			"uses",
			"connector_options",
			"source",
			"modes",
			"default_mode",
		}

		fullFieldsMap := make(map[string]bool)
		for _, field := range AssistantFullFields {
			fullFieldsMap[field] = true
		}

		for _, field := range complexFields {
			if !fullFieldsMap[field] {
				t.Errorf("Complex field %s is missing from full fields", field)
			}
		}
	})

	t.Run("ContainsPermissionFields", func(t *testing.T) {
		// Full fields should include permission fields
		permissionFields := []string{
			"__yao_created_by",
			"__yao_updated_by",
			"__yao_team_id",
			"__yao_tenant_id",
		}

		fullFieldsMap := make(map[string]bool)
		for _, field := range AssistantFullFields {
			fullFieldsMap[field] = true
		}

		for _, field := range permissionFields {
			if !fullFieldsMap[field] {
				t.Errorf("Permission field %s is missing from full fields", field)
			}
		}
	})
}
