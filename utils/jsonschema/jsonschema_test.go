package jsonschema

import (
	"strings"
	"testing"

	"github.com/yaoapp/gou/process"
)

// TestNew tests the New function
func TestNew(t *testing.T) {
	t.Run("ValidSimpleSchema", func(t *testing.T) {
		schema := map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"name": map[string]interface{}{
					"type": "string",
				},
			},
		}

		validator, err := New(schema)
		if err != nil {
			t.Fatalf("Expected valid schema to compile, got error: %v", err)
		}

		if validator == nil {
			t.Fatal("Expected non-nil validator")
		}

		if validator.schema == nil {
			t.Fatal("Expected validator to have compiled schema")
		}

		t.Log("✓ Valid simple schema compiled successfully")
	})

	t.Run("ValidComplexSchema", func(t *testing.T) {
		schema := map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"name": map[string]interface{}{
					"type":      "string",
					"minLength": 1,
					"maxLength": 100,
				},
				"age": map[string]interface{}{
					"type":    "integer",
					"minimum": 0,
					"maximum": 150,
				},
				"email": map[string]interface{}{
					"type":   "string",
					"format": "email",
				},
			},
			"required": []string{"name", "email"},
		}

		validator, err := New(schema)
		if err != nil {
			t.Fatalf("Expected valid complex schema to compile, got error: %v", err)
		}

		if validator == nil {
			t.Fatal("Expected non-nil validator")
		}

		t.Log("✓ Valid complex schema compiled successfully")
	})

	t.Run("InvalidSchema_BadMinimum", func(t *testing.T) {
		schema := map[string]interface{}{
			"type":    "integer",
			"minimum": "not a number",
		}

		_, err := New(schema)
		if err == nil {
			t.Fatal("Expected error for invalid minimum value, got nil")
		}

		t.Log("✓ Invalid schema (bad minimum value) rejected correctly")
	})

	t.Run("SchemaFromJSONString", func(t *testing.T) {
		schemaJSON := `{
			"type": "object",
			"properties": {
				"name": {"type": "string"},
				"age": {"type": "integer"}
			},
			"required": ["name"]
		}`

		validator, err := New(schemaJSON)
		if err != nil {
			t.Fatalf("Expected valid JSON string schema to compile, got error: %v", err)
		}

		if validator == nil {
			t.Fatal("Expected non-nil validator")
		}

		// Test validation with the validator
		data := map[string]interface{}{
			"name": "John",
			"age":  30,
		}

		err = validator.Validate(data)
		if err != nil {
			t.Fatalf("Expected valid data to pass validation, got error: %v", err)
		}

		t.Log("✓ Schema from JSON string compiled and validated successfully")
	})

	t.Run("SchemaFromJSONBytes", func(t *testing.T) {
		schemaJSON := []byte(`{
			"type": "object",
			"properties": {
				"email": {"type": "string", "format": "email"}
			},
			"required": ["email"]
		}`)

		validator, err := New(schemaJSON)
		if err != nil {
			t.Fatalf("Expected valid JSON bytes schema to compile, got error: %v", err)
		}

		if validator == nil {
			t.Fatal("Expected non-nil validator")
		}

		// Test validation with the validator
		data := map[string]interface{}{
			"email": "test@example.com",
		}

		err = validator.Validate(data)
		if err != nil {
			t.Fatalf("Expected valid data to pass validation, got error: %v", err)
		}

		t.Log("✓ Schema from JSON bytes compiled and validated successfully")
	})

	t.Run("InvalidJSONString", func(t *testing.T) {
		schemaJSON := `{invalid json}`

		_, err := New(schemaJSON)
		if err == nil {
			t.Fatal("Expected error for invalid JSON string, got nil")
		}

		t.Log("✓ Invalid JSON string rejected correctly")
	})
}

// TestValidator_Validate tests the Validate method
func TestValidator_Validate(t *testing.T) {
	schema := map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"name": map[string]interface{}{
				"type":      "string",
				"minLength": 1,
			},
			"age": map[string]interface{}{
				"type":    "integer",
				"minimum": 0,
			},
		},
		"required": []string{"name"},
	}

	validator, err := New(schema)
	if err != nil {
		t.Fatalf("Failed to compile schema for testing: %v", err)
	}

	t.Run("ValidData", func(t *testing.T) {
		data := map[string]interface{}{
			"name": "John Doe",
			"age":  30,
		}

		err := validator.Validate(data)
		if err != nil {
			t.Fatalf("Expected valid data to pass validation, got error: %v", err)
		}

		t.Log("✓ Valid data passed validation")
	})

	t.Run("InvalidData_MissingRequired", func(t *testing.T) {
		data := map[string]interface{}{
			"age": 25,
		}

		err := validator.Validate(data)
		if err == nil {
			t.Fatal("Expected validation error for missing required field, got nil")
		}

		if !strings.Contains(err.Error(), "validation failed") {
			t.Errorf("Expected error message to contain 'validation failed', got: %v", err)
		}

		t.Log("✓ Invalid data (missing required) rejected correctly")
	})

	t.Run("InvalidData_WrongType", func(t *testing.T) {
		data := map[string]interface{}{
			"name": "Alice",
			"age":  "not a number",
		}

		err := validator.Validate(data)
		if err == nil {
			t.Fatal("Expected validation error for wrong type, got nil")
		}

		t.Log("✓ Invalid data (wrong type) rejected correctly")
	})
}

// TestValidateSchema tests the ValidateSchema function
func TestValidateSchema(t *testing.T) {
	t.Run("ValidSchema", func(t *testing.T) {
		schema := map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"name": map[string]interface{}{"type": "string"},
			},
		}

		err := ValidateSchema(schema)
		if err != nil {
			t.Fatalf("Expected valid schema, got error: %v", err)
		}

		t.Log("✓ Valid schema validated successfully")
	})

	t.Run("InvalidSchema", func(t *testing.T) {
		schema := map[string]interface{}{
			"type":    "integer",
			"minimum": "invalid",
		}

		err := ValidateSchema(schema)
		if err == nil {
			t.Fatal("Expected error for invalid schema, got nil")
		}

		t.Log("✓ Invalid schema rejected correctly")
	})
}

// TestValidateData tests the ValidateData function
func TestValidateData(t *testing.T) {
	schema := map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"name": map[string]interface{}{
				"type":      "string",
				"minLength": 1,
			},
		},
		"required": []string{"name"},
	}

	t.Run("ValidData", func(t *testing.T) {
		data := map[string]interface{}{
			"name": "John",
		}

		err := ValidateData(schema, data)
		if err != nil {
			t.Fatalf("Expected valid data, got error: %v", err)
		}

		t.Log("✓ Valid data validated successfully")
	})

	t.Run("InvalidData", func(t *testing.T) {
		data := map[string]interface{}{
			"name": "",
		}

		err := ValidateData(schema, data)
		if err == nil {
			t.Fatal("Expected validation error, got nil")
		}

		t.Log("✓ Invalid data rejected correctly")
	})

	t.Run("InvalidSchema", func(t *testing.T) {
		invalidSchema := map[string]interface{}{
			"type":    "integer",
			"minimum": "invalid",
		}

		data := map[string]interface{}{"value": 10}

		err := ValidateData(invalidSchema, data)
		if err == nil {
			t.Fatal("Expected error for invalid schema, got nil")
		}

		t.Log("✓ Invalid schema rejected correctly")
	})
}

// TestArraySchema tests validation with array schema
func TestArraySchema(t *testing.T) {
	schema := map[string]interface{}{
		"type": "array",
		"items": map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"id": map[string]interface{}{
					"type": "integer",
				},
				"name": map[string]interface{}{
					"type": "string",
				},
			},
			"required": []string{"id"},
		},
		"minItems": 1,
	}

	validator, err := New(schema)
	if err != nil {
		t.Fatalf("Failed to compile array schema: %v", err)
	}

	t.Run("ValidArray", func(t *testing.T) {
		data := []interface{}{
			map[string]interface{}{"id": 1, "name": "Item 1"},
			map[string]interface{}{"id": 2, "name": "Item 2"},
		}

		err := validator.Validate(data)
		if err != nil {
			t.Fatalf("Expected valid array to pass validation, got error: %v", err)
		}

		t.Log("✓ Valid array data passed validation")
	})

	t.Run("InvalidArray_Empty", func(t *testing.T) {
		data := []interface{}{}

		err := validator.Validate(data)
		if err == nil {
			t.Fatal("Expected validation error for empty array (minItems: 1), got nil")
		}

		t.Log("✓ Invalid array (empty) rejected correctly")
	})

	t.Run("InvalidArray_MissingRequiredInItem", func(t *testing.T) {
		data := []interface{}{
			map[string]interface{}{"name": "Item without ID"},
		}

		err := validator.Validate(data)
		if err == nil {
			t.Fatal("Expected validation error for item missing required field, got nil")
		}

		t.Log("✓ Invalid array (item missing required field) rejected correctly")
	})
}

// TestNestedSchema tests validation with nested objects
func TestNestedSchema(t *testing.T) {
	schema := map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"user": map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"profile": map[string]interface{}{
						"type": "object",
						"properties": map[string]interface{}{
							"bio": map[string]interface{}{
								"type":      "string",
								"maxLength": 500,
							},
						},
					},
				},
			},
		},
	}

	validator, err := New(schema)
	if err != nil {
		t.Fatalf("Failed to compile nested schema: %v", err)
	}

	t.Run("ValidNestedData", func(t *testing.T) {
		data := map[string]interface{}{
			"user": map[string]interface{}{
				"profile": map[string]interface{}{
					"bio": "This is a short bio",
				},
			},
		}

		err := validator.Validate(data)
		if err != nil {
			t.Fatalf("Expected valid nested data to pass validation, got error: %v", err)
		}

		t.Log("✓ Valid nested data passed validation")
	})

	t.Run("InvalidNestedData_ViolatesConstraint", func(t *testing.T) {
		longBio := strings.Repeat("a", 501)
		data := map[string]interface{}{
			"user": map[string]interface{}{
				"profile": map[string]interface{}{
					"bio": longBio,
				},
			},
		}

		err := validator.Validate(data)
		if err == nil {
			t.Fatal("Expected validation error for bio exceeding maxLength, got nil")
		}

		t.Log("✓ Invalid nested data (violates constraint) rejected correctly")
	})
}

// TestProcessValidateSchema tests the ProcessValidateSchema handler
func TestProcessValidateSchema(t *testing.T) {
	t.Run("ValidSchema", func(t *testing.T) {
		schema := map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"name": map[string]interface{}{"type": "string"},
			},
		}

		p := process.New("test.process", nil)
		p.Args = []interface{}{schema}

		result := ProcessValidateSchema(p)
		if result != nil {
			t.Fatalf("Expected nil for valid schema, got: %v", result)
		}

		t.Log("✓ ProcessValidateSchema: valid schema passed")
	})

	t.Run("InvalidSchema", func(t *testing.T) {
		schema := map[string]interface{}{
			"type":    "integer",
			"minimum": "not a number",
		}

		p := process.New("test.process", nil)
		p.Args = []interface{}{schema}

		result := ProcessValidateSchema(p)
		if result == nil {
			t.Fatal("Expected error for invalid schema, got nil")
		}

		errMsg, ok := result.(string)
		if !ok {
			t.Fatalf("Expected error message string, got: %T", result)
		}

		if !strings.Contains(errMsg, "invalid JSON Schema") {
			t.Errorf("Expected error message to contain 'invalid JSON Schema', got: %s", errMsg)
		}

		t.Log("✓ ProcessValidateSchema: invalid schema rejected correctly")
	})

	t.Run("SchemaFromJSONString", func(t *testing.T) {
		schemaJSON := `{
			"type": "object",
			"properties": {
				"email": {"type": "string"}
			}
		}`

		p := process.New("test.process", nil)
		p.Args = []interface{}{schemaJSON}

		result := ProcessValidateSchema(p)
		if result != nil {
			t.Fatalf("Expected nil for valid JSON string schema, got: %v", result)
		}

		t.Log("✓ ProcessValidateSchema: JSON string schema passed")
	})
}

// TestProcessValidate tests the ProcessValidate handler
func TestProcessValidate(t *testing.T) {
	schema := map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"name": map[string]interface{}{
				"type":      "string",
				"minLength": 1,
			},
			"age": map[string]interface{}{
				"type":    "integer",
				"minimum": 0,
			},
		},
		"required": []string{"name"},
	}

	t.Run("ValidData", func(t *testing.T) {
		data := map[string]interface{}{
			"name": "John Doe",
			"age":  30,
		}

		p := process.New("test.process", nil)
		p.Args = []interface{}{schema, data}

		result := ProcessValidate(p)
		if result != nil {
			t.Fatalf("Expected nil for valid data, got: %v", result)
		}

		t.Log("✓ ProcessValidate: valid data passed")
	})

	t.Run("InvalidData_MissingRequired", func(t *testing.T) {
		data := map[string]interface{}{
			"age": 25,
		}

		p := process.New("test.process", nil)
		p.Args = []interface{}{schema, data}

		result := ProcessValidate(p)
		if result == nil {
			t.Fatal("Expected error for missing required field, got nil")
		}

		errMsg, ok := result.(string)
		if !ok {
			t.Fatalf("Expected error message string, got: %T", result)
		}

		if !strings.Contains(errMsg, "validation failed") {
			t.Errorf("Expected error message to contain 'validation failed', got: %s", errMsg)
		}

		t.Log("✓ ProcessValidate: missing required field rejected correctly")
	})

	t.Run("InvalidData_WrongType", func(t *testing.T) {
		data := map[string]interface{}{
			"name": "Alice",
			"age":  "not a number",
		}

		p := process.New("test.process", nil)
		p.Args = []interface{}{schema, data}

		result := ProcessValidate(p)
		if result == nil {
			t.Fatal("Expected error for wrong type, got nil")
		}

		_, ok := result.(string)
		if !ok {
			t.Fatalf("Expected error message string, got: %T", result)
		}

		t.Log("✓ ProcessValidate: wrong type rejected correctly")
	})

	t.Run("InvalidData_ViolatesConstraint", func(t *testing.T) {
		data := map[string]interface{}{
			"name": "",
		}

		p := process.New("test.process", nil)
		p.Args = []interface{}{schema, data}

		result := ProcessValidate(p)
		if result == nil {
			t.Fatal("Expected error for constraint violation, got nil")
		}

		t.Log("✓ ProcessValidate: constraint violation rejected correctly")
	})

	t.Run("InvalidSchema", func(t *testing.T) {
		invalidSchema := map[string]interface{}{
			"type":    "integer",
			"minimum": "not a number",
		}

		data := map[string]interface{}{"value": 10}

		p := process.New("test.process", nil)
		p.Args = []interface{}{invalidSchema, data}

		result := ProcessValidate(p)
		if result == nil {
			t.Fatal("Expected error for invalid schema, got nil")
		}

		t.Log("✓ ProcessValidate: invalid schema rejected correctly")
	})

	t.Run("SchemaFromJSONString", func(t *testing.T) {
		schemaJSON := `{
			"type": "object",
			"properties": {
				"username": {"type": "string", "minLength": 3}
			},
			"required": ["username"]
		}`

		data := map[string]interface{}{
			"username": "john",
		}

		p := process.New("test.process", nil)
		p.Args = []interface{}{schemaJSON, data}

		result := ProcessValidate(p)
		if result != nil {
			t.Fatalf("Expected nil for valid data with JSON string schema, got: %v", result)
		}

		t.Log("✓ ProcessValidate: JSON string schema with valid data passed")
	})

	t.Run("SchemaFromJSONBytes", func(t *testing.T) {
		schemaJSON := []byte(`{
			"type": "object",
			"properties": {
				"email": {"type": "string"}
			},
			"required": ["email"]
		}`)

		data := map[string]interface{}{
			"email": "test@example.com",
		}

		p := process.New("test.process", nil)
		p.Args = []interface{}{schemaJSON, data}

		result := ProcessValidate(p)
		if result != nil {
			t.Fatalf("Expected nil for valid data with JSON bytes schema, got: %v", result)
		}

		t.Log("✓ ProcessValidate: JSON bytes schema with valid data passed")
	})

	t.Run("ComplexNestedValidation", func(t *testing.T) {
		complexSchema := map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"user": map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"profile": map[string]interface{}{
							"type": "object",
							"properties": map[string]interface{}{
								"age": map[string]interface{}{
									"type":    "integer",
									"minimum": 18,
									"maximum": 100,
								},
							},
							"required": []string{"age"},
						},
					},
					"required": []string{"profile"},
				},
			},
		}

		data := map[string]interface{}{
			"user": map[string]interface{}{
				"profile": map[string]interface{}{
					"age": 25,
				},
			},
		}

		p := process.New("test.process", nil)
		p.Args = []interface{}{complexSchema, data}

		result := ProcessValidate(p)
		if result != nil {
			t.Fatalf("Expected nil for valid nested data, got: %v", result)
		}

		t.Log("✓ ProcessValidate: complex nested validation passed")
	})
}
