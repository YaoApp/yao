package assistant

import (
	"fmt"

	jsoniter "github.com/json-iterator/go"
	store "github.com/yaoapp/yao/agent/store/types"
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

// ToRuntimeTool converts store.Tool to assistant.Tool (OpenAI format)
func ToRuntimeTool(storeTool store.Tool) (Tool, error) {
	var tool Tool

	// Marshal and unmarshal to convert between formats
	raw, err := jsoniter.Marshal(storeTool)
	if err != nil {
		return tool, fmt.Errorf("failed to marshal store tool: %w", err)
	}

	// Try to unmarshal as OpenAI format first
	err = jsoniter.Unmarshal(raw, &tool)
	if err == nil && tool.Function.Name != "" {
		return tool, nil
	}

	// If it's a simple format, convert it
	tool.Type = "function"
	if storeTool.Type != "" {
		tool.Type = storeTool.Type
	}
	tool.Function.Name = storeTool.Name
	tool.Function.Description = storeTool.Description

	// Convert parameters
	if storeTool.Parameters != nil {
		raw, err := jsoniter.Marshal(storeTool.Parameters)
		if err != nil {
			return tool, fmt.Errorf("failed to marshal parameters: %w", err)
		}
		var params Parameter
		err = jsoniter.Unmarshal(raw, &params)
		if err != nil {
			return tool, fmt.Errorf("failed to unmarshal parameters: %w", err)
		}
		tool.Function.Parameters = params
	}

	return tool, nil
}

// ToRuntimeTools converts []store.Tool to []assistant.Tool
func ToRuntimeTools(storeTools []store.Tool) ([]Tool, error) {
	if storeTools == nil {
		return nil, nil
	}

	tools := make([]Tool, 0, len(storeTools))
	for _, storeTool := range storeTools {
		tool, err := ToRuntimeTool(storeTool)
		if err != nil {
			return nil, err
		}
		tools = append(tools, tool)
	}
	return tools, nil
}
