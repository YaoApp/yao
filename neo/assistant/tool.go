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
	Type        string `json:"type,omitempty"`
	Description string `json:"description,omitempty"`
}

// Parameter represents the parameters field in function calling format
type Parameter struct {
	Type                 string                    `json:"type"`
	Properties           map[string]SchemaProperty `json:"properties,omitempty"`
	Required             []string                  `json:"required,omitempty"`
	AdditionalProperties bool                      `json:"additionalProperties"`
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
	default:
		return fmt.Sprintf("<%s>", name)
	}
}
