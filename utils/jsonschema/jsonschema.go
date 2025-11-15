package jsonschema

import (
	"encoding/json"
	"fmt"

	"github.com/kaptinlin/jsonschema"
	"github.com/yaoapp/gou/process"
)

// Validator wraps a compiled JSON Schema for validation
type Validator struct {
	schema *jsonschema.Schema
}

// New compiles a JSON Schema and returns a validator
// Returns error if the schema is invalid
//
// Args:
//   - schema: can be map[string]interface{}, []byte, string, or any JSON-serializable type
//
// Usage:
//
//	// From map
//	schemaMap := map[string]interface{}{
//	    "type": "object",
//	    "properties": map[string]interface{}{
//	        "name": map[string]interface{}{"type": "string"},
//	    },
//	    "required": []string{"name"},
//	}
//	validator, err := jsonschema.New(schemaMap)
//
//	// From JSON string
//	validator, err := jsonschema.New(`{"type": "object", "properties": {...}}`)
//
//	// From JSON bytes
//	validator, err := jsonschema.New([]byte(`{"type": "object", ...}`))
func New(schema interface{}) (*Validator, error) {
	var schemaBytes []byte
	var err error

	// Handle different input types
	switch v := schema.(type) {
	case string:
		// Already a JSON string
		schemaBytes = []byte(v)
	case []byte:
		// Already JSON bytes
		schemaBytes = v
	default:
		// Marshal to JSON
		schemaBytes, err = json.Marshal(schema)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal schema: %w", err)
		}
	}

	// Compile the schema - this validates the schema structure
	compiler := jsonschema.NewCompiler()
	compiledSchema, err := compiler.Compile(schemaBytes)
	if err != nil {
		return nil, fmt.Errorf("invalid JSON Schema: %w", err)
	}

	return &Validator{
		schema: compiledSchema,
	}, nil
}

// Validate validates data against the compiled JSON Schema
// Returns nil if data is valid, error with validation details otherwise
//
// Usage:
//
//	validator, _ := jsonschema.New(schemaMap)
//	data := map[string]interface{}{"name": "John"}
//	if err := validator.Validate(data); err != nil {
//	    log.Printf("Validation failed: %v", err)
//	}
func (v *Validator) Validate(data interface{}) error {
	result := v.schema.Validate(data)
	if !result.IsValid() {
		// Collect all validation errors
		var errMsg string
		for field, err := range result.Errors {
			if errMsg != "" {
				errMsg += "; "
			}
			errMsg += fmt.Sprintf("%s: %s", field, err.Message)
		}
		return fmt.Errorf("validation failed: %s", errMsg)
	}

	return nil
}

// ValidateSchema validates a JSON Schema structure without compiling it
// Returns error if the schema is invalid
func ValidateSchema(schema interface{}) error {
	_, err := New(schema)
	return err
}

// ValidateData validates data against a JSON Schema (one-shot validation)
// Returns error if schema is invalid or data doesn't match the schema
//
// Usage:
//
//	err := jsonschema.ValidateData(schemaMap, data)
//	if err != nil {
//	    log.Printf("Validation failed: %v", err)
//	}
func ValidateData(schema interface{}, data interface{}) error {
	validator, err := New(schema)
	if err != nil {
		return err
	}
	return validator.Validate(data)
}

// ****************************************
// * Process Handlers for JS/DSL
// ****************************************

// ProcessValidateSchema utils.jsonschema.ValidateSchema
// Validates a JSON Schema structure
// Args: schema interface{} - The JSON Schema to validate
// Returns: nil if valid, error message string if invalid
func ProcessValidateSchema(process *process.Process) interface{} {
	process.ValidateArgNums(1)
	schema := process.Args[0]

	err := ValidateSchema(schema)
	if err != nil {
		return err.Error()
	}
	return nil
}

// ProcessValidate utils.jsonschema.Validate
// Validates data against a JSON Schema
// Args:
//   - schema interface{} - The JSON Schema (map, string, or []byte)
//   - data interface{} - The data to validate
//
// Returns: nil if valid, error message string if invalid
//
// Usage in JS/DSL:
//
//	// Validate with schema map
//	var result = Process("utils.jsonschema.Validate", schema, data)
//	if result != null {
//	    log.Error("Validation failed: " + result)
//	}
//
//	// Validate with JSON string
//	var schemaStr = '{"type": "object", "properties": {"name": {"type": "string"}}}'
//	var result = Process("utils.jsonschema.Validate", schemaStr, {"name": "John"})
func ProcessValidate(process *process.Process) interface{} {
	process.ValidateArgNums(2)
	schema := process.Args[0]
	data := process.Args[1]

	err := ValidateData(schema, data)
	if err != nil {
		return err.Error()
	}
	return nil
}
