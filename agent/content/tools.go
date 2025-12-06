package content

import (
	"context"
	"encoding/base64"
	"fmt"
	"sync"

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

// fileInfoMutex protects concurrent access to files_info list in Space
var fileInfoMutex sync.Mutex

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

// CallAgentWithFileInfo calls an agent to process content with file metadata
// The file metadata is passed via ctx.Space for access by hooks (especially Next hook)
// Uses Space instead of Metadata to avoid creating context copies and ensure proper cleanup
//
// Space Keys (with agent ID as namespace prefix to avoid conflicts between different agents):
//   - {agentID}:files_info      - List of all files being processed by this agent (array)
//   - {agentID}:current_file    - Currently processing file (single object)
func CallAgentWithFileInfo(ctx *agentContext.Context, agentID string, message agentContext.Message, info *Info) (string, error) {
	// Store file information in Space if available
	if info != nil && ctx.Space != nil {
		fileInfo := map[string]interface{}{
			"url":          info.URL,
			"filename":     info.Filename,
			"content_type": info.ContentType,
			"file_type":    string(info.FileType),
			"source":       string(info.Source),
		}

		// Add uploader-specific information if available
		if info.UploaderName != "" {
			fileInfo["uploader_name"] = info.UploaderName
		}
		if info.FileID != "" {
			fileInfo["file_id"] = info.FileID
		}

		// Use agent ID as namespace prefix for Space keys
		filesListKey := agentID + ":files_info"
		currentFileKey := agentID + ":current_file"

		// Thread-safe: append current file to files list
		fileInfoMutex.Lock()
		var filesList []map[string]interface{}
		if existing, err := ctx.Space.Get(filesListKey); err == nil {
			// Convert existing data to []map[string]interface{}
			if existingList, ok := existing.([]interface{}); ok {
				for _, item := range existingList {
					if itemMap, ok := item.(map[string]interface{}); ok {
						filesList = append(filesList, itemMap)
					}
				}
			} else if existingList, ok := existing.([]map[string]interface{}); ok {
				filesList = existingList
			}
		}
		// Append current file to list
		filesList = append(filesList, fileInfo)
		ctx.Space.Set(filesListKey, filesList)
		fileInfoMutex.Unlock()

		// Store current file in Space
		if err := ctx.Space.Set(currentFileKey, fileInfo); err != nil {
			log.Trace("[Content] Failed to set current file info in Space: %v", err)
		}

		// Ensure cleanup after agent call completes
		defer func() {
			// Clean up current file
			if err := ctx.Space.Delete(currentFileKey); err != nil {
				log.Trace("[Content] Failed to delete current file info from Space: %v", err)
			}
			// Clean up files list (reset for next call)
			if err := ctx.Space.Delete(filesListKey); err != nil {
				log.Trace("[Content] Failed to delete files list from Space: %v", err)
			}
		}()
	}

	// Call the agent with the original context
	return CallAgent(ctx, agentID, message)
}

// extractTextFromAgentResponse extracts text from agent response
// Handles two main response formats from agent.Stream():
//
//  1. Standard Response (No Next Hook or Next Hook returns nil):
//     Structure: { completion: { content: "text" | [...ContentPart] } }
//     Action: Extract text from completion.content field
//
//  2. Next Hook Response with Custom Data:
//     Structure: { next: <any data from Next hook> }
//     Action:
//     - If next is string → return directly
//     - If next is map/object → JSON stringify and return
//     - This preserves the complete custom data structure from the hook
//
// Priority:
//  1. Check for "next" field (custom hook data) → return complete data
//  2. Check for "completion" field (standard LLM response) → extract text only
//  3. Fallback to direct string or JSON stringify
func extractTextFromAgentResponse(response interface{}) (string, error) {
	if response == nil {
		return "", fmt.Errorf("agent returned nil response")
	}

	// First, try to convert to map if it's a struct
	// agent.Stream() may return *agentContext.Response which needs to be converted
	var responseMap map[string]interface{}

	// Check if it's already a map
	if rm, ok := response.(map[string]interface{}); ok {
		responseMap = rm
	} else {
		// Try to marshal and unmarshal to convert struct to map
		jsonBytes, err := jsoniter.Marshal(response)
		if err != nil {
			// If it's a plain string, return directly
			if responseStr, ok := response.(string); ok {
				return responseStr, nil
			}
			return "", fmt.Errorf("failed to serialize agent response: %w", err)
		}

		// Unmarshal to map
		if err := jsoniter.Unmarshal(jsonBytes, &responseMap); err != nil {
			// If unmarshal fails, return the JSON string
			return string(jsonBytes), nil
		}
	}

	// Priority 1: Check for "next" field (custom hook data)
	// If Next hook returns custom data, it's stored in the "next" field
	// Return the complete custom data structure (preserve hook's intent)
	if next, hasNext := responseMap["next"]; hasNext && next != nil {
		// If next is a string, return directly
		if nextStr, ok := next.(string); ok {
			return nextStr, nil
		}
		// Otherwise, JSON stringify to preserve complete structure
		jsonBytes, err := jsoniter.Marshal(next)
		if err != nil {
			return "", fmt.Errorf("failed to serialize next hook data: %w", err)
		}
		return string(jsonBytes), nil
	}

	// Priority 2: Check for "completion" field (standard LLM response)
	// Extract text content from the LLM completion
	if completion, hasCompletion := responseMap["completion"]; hasCompletion && completion != nil {
		if completionMap, ok := completion.(map[string]interface{}); ok {
			// Extract content from completion
			if content, hasContent := completionMap["content"]; hasContent {
				// Content can be string or []ContentPart (multimodal)
				switch v := content.(type) {
				case string:
					// Simple text content
					return v, nil
				case []interface{}:
					// Multimodal content array - extract all text parts
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
					// No text found in content parts
					return "", fmt.Errorf("no text content found in completion content parts")
				}
			}
		}
	}

	// Fallback: Try to find a "content" field directly (shouldn't happen normally)
	if content, hasContent := responseMap["content"]; hasContent {
		if contentStr, ok := content.(string); ok {
			return contentStr, nil
		}
	}

	// Last resort: JSON stringify the entire response
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
