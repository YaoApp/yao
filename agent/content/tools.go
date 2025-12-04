package content

import (
	"context"
	"encoding/base64"
	"fmt"

	jsoniter "github.com/json-iterator/go"
	"github.com/yaoapp/gou/mcp"
	"github.com/yaoapp/kun/log"
	agentContext "github.com/yaoapp/yao/agent/context"
)

// AgentCaller interface for calling agents (to avoid circular dependency)
type AgentCaller interface {
	Stream(ctx *agentContext.Context, messages []agentContext.Message, options ...*agentContext.Options) (interface{}, error)
}

// AgentGetterFunc is a function type that gets an agent by ID
var AgentGetterFunc func(agentID string) (AgentCaller, error)

// CallAgent calls an agent to process content (vision, audio, etc.)
// This is a generic function that can be used by any handler
func CallAgent(ctx *agentContext.Context, agentID string, message agentContext.Message) (string, error) {
	if AgentGetterFunc == nil {
		return "", fmt.Errorf("AgentGetterFunc not initialized")
	}

	// Load the agent by ID using the injected function
	agent, err := AgentGetterFunc(agentID)
	if err != nil {
		return "", fmt.Errorf("failed to load agent %s: %w", agentID, err)
	}

	// Call the agent with the message
	messages := []agentContext.Message{message}

	// Note: Connector is now in Options (call-level parameter), not Context
	// For A2A calls, skip history and output (we only need the response data)
	opts := &agentContext.Options{Skip: &agentContext.Skip{History: true, Output: true}} // Skip history and output
	response, err := agent.Stream(ctx, messages, opts)
	if err != nil {
		return "", fmt.Errorf("failed to call agent %s: %w", agentID, err)
	}

	// Extract text from agent response
	// Two formats are supported:
	// 1. Custom Hook response (from Next hook)
	// 2. Standard Agent Stream response (LLM completion)

	return extractTextFromAgentResponse(response)
}

// extractTextFromAgentResponse extracts text from agent response
// Handles two response formats:
// 1. Custom Hook response: if it's a string, return directly; otherwise JSON stringify
// 2. Standard response: extract from completion.content
func extractTextFromAgentResponse(response interface{}) (string, error) {
	if response == nil {
		return "", fmt.Errorf("agent returned nil response")
	}

	// Try to parse as standard response format (has "completion" field with LLM result)
	if responseMap, ok := response.(map[string]interface{}); ok {
		// Check for completion field (standard LLM response)
		if completion, hasCompletion := responseMap["completion"]; hasCompletion {
			if completionMap, ok := completion.(map[string]interface{}); ok {
				// Extract content from completion
				if content, hasContent := completionMap["content"]; hasContent {
					// Content can be string or structured
					switch v := content.(type) {
					case string:
						return v, nil
					case []interface{}:
						// Handle multimodal content array
						var text string
						for _, part := range v {
							if partMap, ok := part.(map[string]interface{}); ok {
								if partType, _ := partMap["type"].(string); partType == "text" {
									if textContent, ok := partMap["text"].(string); ok {
										text += textContent
									}
								}
							}
						}
						if text != "" {
							return text, nil
						}
					}
				}
			}
		}

		// Check for data field (custom hook response with data wrapper)
		if data, hasData := responseMap["data"]; hasData {
			// If data is a string, return directly
			if dataStr, ok := data.(string); ok {
				return dataStr, nil
			}
			// Otherwise, JSON stringify
			jsonBytes, err := jsoniter.Marshal(data)
			if err != nil {
				return "", fmt.Errorf("failed to serialize hook data response: %w", err)
			}
			return string(jsonBytes), nil
		}

		// If the map itself looks like content, try to extract
		// This handles cases where the response is the content directly
		if content, hasContent := responseMap["content"]; hasContent {
			if contentStr, ok := content.(string); ok {
				return contentStr, nil
			}
		}
	}

	// Custom Hook response: if it's a plain string, return directly
	if responseStr, ok := response.(string); ok {
		return responseStr, nil
	}

	// Otherwise, JSON stringify the response
	jsonBytes, err := jsoniter.Marshal(response)
	if err != nil {
		return "", fmt.Errorf("failed to serialize agent response: %w", err)
	}
	return string(jsonBytes), nil
}

// CallMCPTool calls an MCP tool to process content
// This is a generic function that can be used by any handler
func CallMCPTool(ctx *agentContext.Context, serverID string, toolName string, arguments map[string]interface{}) (string, error) {
	// Get MCP context for cancellation/timeout control
	mcpCtx := ctx.Context
	if mcpCtx == nil {
		mcpCtx = context.Background()
	}

	// Get MCP client
	client, err := mcp.Select(serverID)
	if err != nil {
		return "", fmt.Errorf("failed to select MCP client '%s': %w", serverID, err)
	}

	// Call the tool
	log.Trace("[Content] Calling MCP tool: %s (server: %s)", toolName, serverID)
	callResult, err := client.CallTool(mcpCtx, toolName, arguments)
	if err != nil {
		return "", fmt.Errorf("MCP tool call failed: %w", err)
	}

	// Check if result is an error
	if callResult.IsError {
		return "", fmt.Errorf("MCP tool returned error: %v", callResult.Content)
	}

	// Extract text content from result
	// callResult.Content is []ToolContent
	var text string
	for _, content := range callResult.Content {
		if content.Type == "text" {
			text += content.Text
		}
		// Can also handle other types like image, resource if needed
	}

	if text == "" {
		// If no text content found, return error
		return "", fmt.Errorf("MCP tool returned no text content")
	}

	return text, nil
}

// EncodeToBase64DataURI encodes data to base64 with data URI prefix
// This is useful for encoding images, audio, or other binary data
func EncodeToBase64DataURI(data []byte, contentType string) string {
	// Ensure we have a valid content type
	if contentType == "" {
		contentType = "application/octet-stream" // default
	}

	// Encode to base64
	encoded := base64.StdEncoding.EncodeToString(data)

	// Return data URI format
	return fmt.Sprintf("data:%s;base64,%s", contentType, encoded)
}
