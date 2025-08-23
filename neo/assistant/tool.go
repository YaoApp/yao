package assistant

import (
	"fmt"

	jsoniter "github.com/json-iterator/go"
)

// Tool represents a tool
type Tool struct {
	Type     string `json:"type"`
	Function struct {
		Name        string    `json:"name"`
		Description string    `json:"description"`
		Parameters  Parameter `json:"parameters"`
		Strict      bool      `json:"strict,omitempty"`
	} `json:"function"`
}

// SchemaProperty represents a JSON Schema property
type SchemaProperty struct {
	Type        string           `json:"type,omitempty"`
	Description string           `json:"description,omitempty"`
	Items       *Parameter       `json:"items,omitempty"`
	OneOf       []SchemaProperty `json:"oneOf,omitempty"`
	Enum        []interface{}    `json:"enum,omitempty"`
}

// Parameter represents the parameters field in function calling format
type Parameter struct {
	Type                 string                    `json:"type,omitempty"`
	Properties           map[string]SchemaProperty `json:"properties,omitempty"`
	Description          string                    `json:"description,omitempty"`
	Required             []string                  `json:"required,omitempty"`
	AdditionalProperties bool                      `json:"additionalProperties,omitempty"`
	Strict               bool                      `json:"strict,omitempty"`
	OneOf                []SchemaProperty          `json:"oneOf,omitempty"`
	Enum                 []interface{}             `json:"enum,omitempty"`
}

// Example returns a formatted example of how to use this tool
func (tool Tool) Example() string {
	return fmt.Sprintf("<tool>\n{\"function\":\"%s\",\"arguments\":%s}\n</tool>",
		tool.Function.Name,
		jsoniter.Wrap(tool.ExampleArguments()).ToString())
}

// ExampleArguments generates example arguments for the tool based on parameter types
func (tool Tool) ExampleArguments() map[string]interface{} {

	args := map[string]interface{}{}

	// Handle the root parameter object
	if tool.Function.Parameters.Type == "object" && tool.Function.Parameters.Properties != nil {
		for name, prop := range tool.Function.Parameters.Properties {
			args[name] = generateExampleValue(name, prop)
		}
	}
	return args
}

// generateExampleValue creates an example value for a parameter
func generateExampleValue(name string, prop SchemaProperty) interface{} {
	if len(prop.OneOf) > 0 {
		// Return the first non-null type example value from oneOf
		for _, subProp := range prop.OneOf {
			if subProp.Type != "null" {
				return generateExampleValue(name, subProp)
			}
		}
		return nil
	}

	// If enum is defined, return the first enum value
	if len(prop.Enum) > 0 {
		return prop.Enum[0]
	}

	switch prop.Type {
	case "string":
		return fmt.Sprintf("<%s:string>", name)
	case "number":
		return fmt.Sprintf("<%s:number>", name)
	case "integer":
		return fmt.Sprintf("<%s:integer>", name)
	case "boolean":
		return fmt.Sprintf("<%s:boolean>", name)
	case "object":
		return fmt.Sprintf("<%s:object>", name)
	case "array":
		return fmt.Sprintf("<%s:array>", name)
	case "null":
		return nil
	default:
		return fmt.Sprintf("<%s>", name)
	}
}
