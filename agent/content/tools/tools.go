package tools

import (
	"context"
	"fmt"

	jsoniter "github.com/json-iterator/go"
	"github.com/yaoapp/gou/mcp"
	"github.com/yaoapp/kun/log"
	"github.com/yaoapp/yao/agent/caller"
	agentContext "github.com/yaoapp/yao/agent/context"
)

// CallAgent calls an agent to process content (vision, audio, etc.)
func CallAgent(ctx *agentContext.Context, agentID string, message agentContext.Message) (string, error) {
	if caller.AgentGetterFunc == nil {
		return "", fmt.Errorf("AgentGetterFunc not initialized")
	}

	// Load the agent by ID using the injected function
	agent, err := caller.AgentGetterFunc(agentID)
	if err != nil {
		return "", fmt.Errorf("failed to load agent %s: %w", agentID, err)
	}

	// Call the agent with the message
	messages := []agentContext.Message{message}

	// For A2A calls, skip history and output (we only need the response data)
	opts := &agentContext.Options{Skip: &agentContext.Skip{History: true, Output: true}}
	response, err := agent.Stream(ctx, messages, opts)
	if err != nil {
		return "", fmt.Errorf("failed to call agent %s: %w", agentID, err)
	}

	// Extract text from agent response
	return ExtractTextFromResponse(response)
}

// ExtractTextFromResponse extracts text from agent response
func ExtractTextFromResponse(response *agentContext.Response) (string, error) {
	if response == nil {
		return "", fmt.Errorf("agent returned nil response")
	}

	// Priority 1: Check Next field (custom hook data)
	if response.Next != nil {
		if nextStr, ok := response.Next.(string); ok {
			return nextStr, nil
		}
		// Otherwise, JSON stringify to preserve complete structure
		jsonBytes, err := jsoniter.Marshal(response.Next)
		if err != nil {
			return "", fmt.Errorf("failed to serialize next hook data: %w", err)
		}
		return string(jsonBytes), nil
	}

	// Priority 2: Check Completion field (standard LLM response)
	if response.Completion != nil {
		switch v := response.Completion.Content.(type) {
		case string:
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
			return "", fmt.Errorf("no text content found in completion content parts")
		}
	}

	return "", fmt.Errorf("no content found in agent response")
}

// CallMCPTool calls an MCP tool to process content
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
	var text string
	for _, content := range callResult.Content {
		if content.Type == "text" {
			text += content.Text
		}
	}

	if text == "" {
		return "", fmt.Errorf("MCP tool returned no text content")
	}

	return text, nil
}
