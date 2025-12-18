package assistant

import (
	jsoniter "github.com/json-iterator/go"
	v8 "github.com/yaoapp/gou/runtime/v8"
	"github.com/yaoapp/yao/agent/assistant/hook"
	chatctx "github.com/yaoapp/yao/agent/context"

	outputMessage "github.com/yaoapp/yao/agent/output/message"
	store "github.com/yaoapp/yao/agent/store/types"
	api "github.com/yaoapp/yao/openai"
)

const (
	// HookErrorMethodNotFound is the error message for method not found
	HookErrorMethodNotFound = "method not found"
)

// API the assistant API interface
type API interface {
	GetPlaceholder(locale string) *store.Placeholder
}

// Script the script scripts except hook script
type Script struct {
	*v8.Script
}

// Assistant the assistant
type Assistant struct {
	store.AssistantModel
	HookScript *hook.Script       `json:"-" yaml:"-"` // Hook Script (index.ts)
	Scripts    map[string]*Script `json:"-" yaml:"-"` // Other scripts

	// Internal
	// ===============================
	openai *api.OpenAI // OpenAI API
	vision bool        // Whether this assistant supports vision
}

// MCPTool represents a simplified MCP tool for building LLM requests
// This is an internal representation used when collecting tools from MCP servers
// and preparing them for the LLM's tool calling interface
type MCPTool struct {
	Name        string      // Formatted tool name with server prefix (e.g., "server_id__tool_name")
	Description string      // Tool description from MCP server
	Parameters  interface{} // JSON Schema for tool parameters (from MCP InputSchema)
}

// ToolCallResult represents the result of a tool call execution
// Used to track the outcome of MCP tool invocations during agent execution
type ToolCallResult struct {
	ToolCallID       string // Tool call ID from the LLM (matches the ID in the LLM's tool_calls response)
	Name             string // Tool name (formatted with server prefix, e.g., "server_id__tool_name")
	Content          string // Result content (JSON string of the tool's output or error message)
	Error            error  // Error if the call failed (nil if successful)
	IsRetryableError bool   // Whether the error should be sent to LLM for retry
	// true: parameter/validation errors that LLM can fix (e.g., "missing required field")
	// false: MCP internal errors that LLM cannot fix (e.g., "network error", "service unavailable")
}

// Server extracts the MCP server ID from the formatted tool name
// Example: "echo__ping" -> "echo"
func (r *ToolCallResult) Server() string {
	serverID, _, _ := ParseMCPToolName(r.Name)
	return serverID
}

// Tool extracts the original tool name without server prefix
// Example: "echo__ping" -> "ping"
func (r *ToolCallResult) Tool() string {
	_, toolName, _ := ParseMCPToolName(r.Name)
	return toolName
}

// NextProcessContext encapsulates all the context needed to process Next hook responses
// This simplifies function signatures and makes it easier to add new fields in the future
type NextProcessContext struct {
	Context            *chatctx.Context            // Agent context
	NextResponse       *chatctx.NextHookResponse   // Response from Next hook (already converted from JS)
	CompletionResponse *chatctx.CompletionResponse // LLM completion response
	FullMessages       []chatctx.Message           // Full conversation history
	ToolCallResponses  []chatctx.ToolCallResponse  // Tool call results (if any)
	StreamHandler      outputMessage.StreamFunc    // Stream handler for output
	CreateResponse     *chatctx.HookCreateResponse // Create hook response
}

// SearchIntent is an alias for context.SearchIntent
// Used for search intent detection from __yao.needsearch agent
type SearchIntent = chatctx.SearchIntent

// ParsedContent extracts the actual tool return value from MCP ToolContent array
// According to MCP protocol:
// - Content is []ToolContent array
// - For "text" type, the actual value is in Text field (usually JSON string)
// - For "image" type, returns the Data field
// - For "resource" type, returns the Resource object
// If there are multiple content items, returns an array of parsed values
func (r *ToolCallResult) ParsedContent() (interface{}, error) {
	if r.Content == "" {
		return nil, nil
	}

	// Parse Content as []ToolContent
	var toolContents []map[string]interface{}
	if err := jsoniter.UnmarshalFromString(r.Content, &toolContents); err != nil {
		// If parsing fails, return the string content directly (error message)
		return r.Content, nil
	}

	// Extract actual values from ToolContent items
	var results []interface{}
	for _, tc := range toolContents {
		contentType, _ := tc["type"].(string)

		switch contentType {
		case "text":
			// For text type, parse the Text field (usually JSON)
			if textStr, ok := tc["text"].(string); ok {
				// Try to parse as JSON
				var parsed interface{}
				if err := jsoniter.UnmarshalFromString(textStr, &parsed); err == nil {
					results = append(results, parsed)
				} else {
					// If not JSON, return as plain string
					results = append(results, textStr)
				}
			}
		case "image":
			// For image type, return the data and mimeType
			results = append(results, map[string]interface{}{
				"type":     "image",
				"data":     tc["data"],
				"mimeType": tc["mimeType"],
			})
		case "resource":
			// For resource type, return the resource object
			results = append(results, tc["resource"])
		default:
			// Unknown type, return as-is
			results = append(results, tc)
		}
	}

	// If only one result, return it directly (not as array)
	if len(results) == 1 {
		return results[0], nil
	}

	return results, nil
}
