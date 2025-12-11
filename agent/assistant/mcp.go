package assistant

import (
	"context"
	"fmt"
	"strings"

	jsoniter "github.com/json-iterator/go"
	gouJson "github.com/yaoapp/gou/json"
	"github.com/yaoapp/gou/mcp"
	mcpTypes "github.com/yaoapp/gou/mcp/types"
	agentContext "github.com/yaoapp/yao/agent/context"
	storeTypes "github.com/yaoapp/yao/agent/store/types"
	"github.com/yaoapp/yao/trace/types"
)

const (
	// MaxMCPTools maximum number of MCP tools to include (to avoid overwhelming the LLM)
	MaxMCPTools = 20
)

// MCPToolName formats a tool name with MCP server prefix
// Format: server_id__tool_name (double underscore separator)
// Dots in server_id are replaced with single underscores
// Examples:
//   - ("echo", "ping") → "echo__ping"
//   - ("github.enterprise", "search") → "github_enterprise__search"
//
// Naming constraint: MCP server_id MUST NOT contain underscores (_)
// Only dots (.), letters, numbers, and hyphens (-) are allowed in server_id
func MCPToolName(serverID, toolName string) string {
	if serverID == "" || toolName == "" {
		return ""
	}
	// Replace dots with single underscores in server_id
	cleanServerID := strings.ReplaceAll(serverID, ".", "_")
	// Use double underscore as separator
	return fmt.Sprintf("%s__%s", cleanServerID, toolName)
}

// ParseMCPToolName parses a formatted MCP tool name into server ID and tool name
// Splits by double underscore (__), then restores dots in server_id
// Examples:
//   - "echo__ping" → ("echo", "ping")
//   - "github_enterprise__search" → ("github.enterprise", "search")
//
// Returns (serverID, toolName, true) if valid format, ("", "", false) otherwise
func ParseMCPToolName(formattedName string) (string, string, bool) {
	if formattedName == "" {
		return "", "", false
	}

	// Split by double underscore
	parts := strings.Split(formattedName, "__")
	if len(parts) != 2 {
		return "", "", false
	}

	cleanServerID := parts[0]
	toolName := parts[1]

	// Validate that both parts are non-empty
	if cleanServerID == "" || toolName == "" {
		return "", "", false
	}

	// Restore dots in server_id (replace single underscores back to dots)
	serverID := strings.ReplaceAll(cleanServerID, "_", ".")

	return serverID, toolName, true
}

// buildMCPTools builds tool definitions and samples system prompt from MCP servers
// Returns (tools, samplesPrompt, error)
func (ast *Assistant) buildMCPTools(ctx *agentContext.Context, createResponse *agentContext.HookCreateResponse) ([]MCPTool, string, error) {
	// Determine which MCP servers to use: hook's or assistant's (hook takes precedence)
	var servers []storeTypes.MCPServerConfig

	// If hook provides MCP servers, use those (override)
	if createResponse != nil && len(createResponse.MCPServers) > 0 {
		servers = make([]storeTypes.MCPServerConfig, len(createResponse.MCPServers))
		for i, hookServer := range createResponse.MCPServers {
			// Convert context.MCPServerConfig to storeTypes.MCPServerConfig
			servers[i] = storeTypes.MCPServerConfig{
				ServerID:  hookServer.ServerID,
				Tools:     hookServer.Tools,
				Resources: hookServer.Resources,
			}
		}
	} else if ast.MCP != nil && len(ast.MCP.Servers) > 0 {
		// Otherwise, use assistant's configured servers
		servers = ast.MCP.Servers
	} else {
		// No servers configured
		return nil, "", nil
	}

	// Use the agent context for cancellation and timeout control
	mcpCtx := ctx.Context
	if mcpCtx == nil {
		mcpCtx = context.Background()
	}

	allTools := make([]MCPTool, 0)
	samplesBuilder := strings.Builder{}
	hasSamples := false

	// Process each MCP server in order
	for _, serverConfig := range servers {
		if len(allTools) >= MaxMCPTools {
			ctx.Logger.Warn("Reached maximum tool limit (%d), skipping remaining servers", MaxMCPTools)
			break
		}

		// Get MCP client
		client, err := mcp.Select(serverConfig.ServerID)
		if err != nil {
			ctx.Logger.Warn("Failed to select MCP client '%s': %v", serverConfig.ServerID, err)
			continue
		}

		// Get tools list (filter by serverConfig.Tools if specified)
		toolsResponse, err := client.ListTools(mcpCtx, "")
		if err != nil {
			ctx.Logger.Warn("Failed to list tools for '%s': %v", serverConfig.ServerID, err)
			continue
		}

		// Build tool filter map if specified
		toolFilter := make(map[string]bool)
		if len(serverConfig.Tools) > 0 {
			for _, toolName := range serverConfig.Tools {
				toolFilter[toolName] = true
			}
		}

		// Process each tool
		for _, tool := range toolsResponse.Tools {
			// Check tool limit
			if len(allTools) >= MaxMCPTools {
				break
			}

			// Apply tool filter if specified
			if len(toolFilter) > 0 && !toolFilter[tool.Name] {
				continue
			}

			// Format tool name with server prefix
			formattedName := MCPToolName(serverConfig.ServerID, tool.Name)

			// Convert MCP tool to MCPTool format
			mcpTool := MCPTool{
				Name:        formattedName,
				Description: tool.Description,
				Parameters:  tool.InputSchema,
			}

			allTools = append(allTools, mcpTool)

			// Try to get samples for this tool
			samples, err := client.ListSamples(mcpCtx, mcpTypes.SampleTool, tool.Name)
			if err == nil && len(samples.Samples) > 0 {
				if !hasSamples {
					samplesBuilder.WriteString("\n\n## MCP Tool Usage Examples\n\n")
					samplesBuilder.WriteString("The following examples demonstrate how to use MCP tools correctly:\n\n")
					hasSamples = true
				}

				samplesBuilder.WriteString(fmt.Sprintf("### %s\n\n", formattedName))
				if tool.Description != "" {
					samplesBuilder.WriteString(fmt.Sprintf("**Description**: %s\n\n", tool.Description))
				}

				for i, sample := range samples.Samples {
					if i >= 3 { // Limit to 3 examples per tool
						break
					}

					samplesBuilder.WriteString(fmt.Sprintf("**Example %d", i+1))
					if sample.Name != "" {
						samplesBuilder.WriteString(fmt.Sprintf(" - %s", sample.Name))
					}
					samplesBuilder.WriteString("**:\n")

					// Check metadata for description
					if sample.Metadata != nil {
						if desc, ok := sample.Metadata["description"].(string); ok && desc != "" {
							samplesBuilder.WriteString(fmt.Sprintf("- Description: %s\n", desc))
						}
					}

					if sample.Input != nil {
						samplesBuilder.WriteString(fmt.Sprintf("- Input: `%v`\n", sample.Input))
					}

					if sample.Output != nil {
						samplesBuilder.WriteString(fmt.Sprintf("- Output: `%v`\n", sample.Output))
					}

					samplesBuilder.WriteString("\n")
				}
			}
		}

		ctx.Logger.Debug("Loaded %d tools from server '%s'", len(toolsResponse.Tools), serverConfig.ServerID)
	}

	samplesPrompt := ""
	if hasSamples {
		samplesPrompt = samplesBuilder.String()
	}

	ctx.Logger.Debug("Total MCP tools loaded: %d", len(allTools))
	return allTools, samplesPrompt, nil
}

// ToolCallResult represents the result of a tool call execution
// executeToolCalls executes tool calls with intelligent strategy and trace logging:
// - Single tool: use CallTool, single trace node
// - Multiple tools: use CallToolsParallel with parallel trace nodes, fallback to sequential on certain errors
// Returns (results, hasErrors)
func (ast *Assistant) executeToolCalls(ctx *agentContext.Context, toolCalls []agentContext.ToolCall, attempt int) ([]ToolCallResult, bool) {
	if len(toolCalls) == 0 {
		return nil, false
	}

	ctx.Logger.Debug("Executing %d tool calls (attempt %d)", len(toolCalls), attempt)

	// Single tool call
	if len(toolCalls) == 1 {
		return ast.executeSingleToolCall(ctx, toolCalls[0])
	}

	// Multiple tool calls - try parallel first
	return ast.executeMultipleToolCallsParallel(ctx, toolCalls)
}

// executeSingleToolCall executes a single tool call with trace logging
func (ast *Assistant) executeSingleToolCall(ctx *agentContext.Context, toolCall agentContext.ToolCall) ([]ToolCallResult, bool) {
	ctx.Logger.ToolStart(toolCall.Function.Name)

	trace, _ := ctx.Trace()

	// Use the agent context for cancellation and timeout control
	mcpCtx := ctx.Context
	if mcpCtx == nil {
		mcpCtx = context.Background()
	}

	result := ToolCallResult{
		ToolCallID: toolCall.ID,
		Name:       toolCall.Function.Name,
	}

	// Parse tool name
	serverID, toolName, ok := ParseMCPToolName(toolCall.Function.Name)
	if !ok {
		result.Error = fmt.Errorf("invalid MCP tool name format: %s", toolCall.Function.Name)
		result.Content = result.Error.Error()
		ctx.Logger.Error("Invalid MCP tool name format: %s", toolCall.Function.Name)
		ctx.Logger.ToolComplete(toolCall.Function.Name, false)
		return []ToolCallResult{result}, true
	}

	// Get MCP client
	client, err := mcp.Select(serverID)
	if err != nil {
		result.Error = fmt.Errorf("failed to select MCP client '%s': %w", serverID, err)
		result.Content = result.Error.Error()
		result.IsRetryableError = false // MCP client selection error is not retryable
		ctx.Logger.Error("Failed to select MCP client '%s': %v", serverID, err)
		ctx.Logger.ToolComplete(toolCall.Function.Name, false)
		return []ToolCallResult{result}, true
	}

	// Get tool info for description and schema
	toolsResponse, err := client.ListTools(mcpCtx, "")
	var toolDescription string
	var toolSchema interface{}
	if err == nil {
		for _, t := range toolsResponse.Tools {
			if t.Name == toolName {
				toolDescription = t.Description
				toolSchema = t.InputSchema
				break
			}
		}
	}
	if toolDescription == "" {
		toolDescription = fmt.Sprintf("MCP tool '%s'", toolName)
	}

	// Add trace node for this tool call
	var toolNode types.Node
	if trace != nil {
		toolNode, _ = trace.Add(
			map[string]any{
				"tool_call_id": toolCall.ID,
				"server":       serverID,
				"tool":         toolName,
				"arguments":    toolCall.Function.Arguments,
			},
			types.TraceNodeOption{
				Label:       toolDescription,
				Type:        "mcp_tool",
				Icon:        "build",
				Description: fmt.Sprintf("Calling '%s' on server '%s'", toolName, serverID),
			},
		)
	}

	// Parse arguments with repair support for better tolerance
	var args map[string]interface{}
	if toolCall.Function.Arguments != "" {
		parsed, err := gouJson.Parse(toolCall.Function.Arguments)
		if err != nil {
			result.Error = fmt.Errorf("failed to parse arguments: %w", err)
			result.Content = result.Error.Error()
			result.IsRetryableError = true // Argument parsing error is retryable by LLM
			ctx.Logger.Error("Failed to parse arguments: %v", err)
			if toolNode != nil {
				toolNode.Fail(result.Error)
			}
			return []ToolCallResult{result}, true
		}

		// Convert to map
		if argsMap, ok := parsed.(map[string]interface{}); ok {
			args = argsMap
		} else {
			result.Error = fmt.Errorf("arguments must be an object, got %T", parsed)
			result.Content = result.Error.Error()
			result.IsRetryableError = true // Type error is retryable by LLM
			ctx.Logger.Error("Arguments must be an object, got %T", parsed)
			if toolNode != nil {
				toolNode.Fail(result.Error)
			}
			return []ToolCallResult{result}, true
		}

		// Validate arguments against tool schema if available
		if toolSchema != nil {
			if err := gouJson.Validate(args, toolSchema); err != nil {
				result.Error = fmt.Errorf("argument validation failed: %w", err)
				result.Content = result.Error.Error()
				result.IsRetryableError = true // Validation error is retryable by LLM
				ctx.Logger.Error("Argument validation failed: %v", err)
				if toolNode != nil {
					toolNode.Fail(result.Error)
				}
				return []ToolCallResult{result}, true
			}
		}
	}

	// Call the tool with agent context as extra argument
	ctx.Logger.Debug("Calling tool: %s (server: %s)", toolName, serverID)

	// Pass agent context as extra argument (only used for Process transport)
	callResult, err := client.CallTool(mcpCtx, toolName, args, ctx)
	if err != nil {
		result.Error = fmt.Errorf("tool call failed: %w", err)
		result.Content = result.Error.Error()
		// Check if error is retryable (parameter/validation errors)
		result.IsRetryableError = isRetryableToolError(err)
		ctx.Logger.Error("Tool call failed: %v (retryable: %v)", err, result.IsRetryableError)
		ctx.Logger.ToolComplete(toolCall.Function.Name, false)
		if toolNode != nil {
			toolNode.Fail(result.Error)
		}
		return []ToolCallResult{result}, true
	}

	// Check if result is an error
	if callResult.IsError {
		result.Error = fmt.Errorf("MCP tool error")
		result.IsRetryableError = false // MCP internal error is not retryable
	}

	// Serialize the Content field only ([]ToolContent)
	contentBytes, err := jsoniter.Marshal(callResult.Content)
	if err != nil {
		result.Error = fmt.Errorf("failed to serialize result: %w", err)
		result.Content = result.Error.Error()
		result.IsRetryableError = false
		ctx.Logger.Error("Failed to serialize result: %v", err)
		ctx.Logger.ToolComplete(toolCall.Function.Name, false)
		if toolNode != nil {
			toolNode.Fail(result.Error)
		}
		return []ToolCallResult{result}, true
	}

	result.Content = string(contentBytes)
	ctx.Logger.ToolComplete(toolName, true)

	if toolNode != nil {
		toolNode.Complete(map[string]any{
			"result": callResult,
		})
	}

	return []ToolCallResult{result}, false
}

// executeMultipleToolCallsParallel executes multiple tool calls in parallel with trace logging
func (ast *Assistant) executeMultipleToolCallsParallel(ctx *agentContext.Context, toolCalls []agentContext.ToolCall) ([]ToolCallResult, bool) {
	trace, _ := ctx.Trace()

	// Use the agent context for cancellation and timeout control
	mcpCtx := ctx.Context
	if mcpCtx == nil {
		mcpCtx = context.Background()
	}

	// Group tool calls by server
	serverGroups := make(map[string][]agentContext.ToolCall)
	for _, tc := range toolCalls {
		serverID, _, ok := ParseMCPToolName(tc.Function.Name)
		if !ok {
			ctx.Logger.Warn("Invalid tool name format: %s", tc.Function.Name)
			continue
		}
		serverGroups[serverID] = append(serverGroups[serverID], tc)
	}

	results := make([]ToolCallResult, 0, len(toolCalls))
	hasErrors := false

	// Process each server's tools
	for serverID, calls := range serverGroups {
		client, err := mcp.Select(serverID)
		if err != nil {
			ctx.Logger.Error("Failed to select MCP client '%s': %v", serverID, err)
			// Add error results for all calls to this server
			for _, tc := range calls {
				results = append(results, ToolCallResult{
					ToolCallID: tc.ID,
					Name:       tc.Function.Name,
					Content:    fmt.Sprintf("Failed to select MCP client: %v", err),
					Error:      err,
				})
			}
			hasErrors = true
			continue
		}

		// Try parallel execution
		serverResults, serverHasErrors := ast.executeServerToolsParallelWithTrace(
			mcpCtx, ctx, trace, client, serverID, calls,
		)

		// If parallel execution failed with retryable error, try sequential
		if serverHasErrors && ast.shouldRetrySequential(serverResults) {
			ctx.Logger.Warn("Parallel execution had parameter errors for server '%s', retrying sequentially", serverID)
			serverResults, serverHasErrors = ast.executeServerToolsSequentialWithTrace(
				mcpCtx, ctx, trace, client, serverID, calls,
			)
		}

		results = append(results, serverResults...)
		if serverHasErrors {
			hasErrors = true
		}
	}

	return results, hasErrors
}

// isRetryableToolError checks if an error is retryable by LLM (parameter/validation errors)
// Returns true for errors that LLM can potentially fix by adjusting parameters
// Returns false for MCP internal errors (network, auth, service unavailable, etc.)
func isRetryableToolError(err error) bool {
	if err == nil {
		return false
	}

	errMsg := strings.ToLower(err.Error())

	// These are NOT retryable (MCP internal issues)
	nonRetryablePatterns := []string{
		"network",
		"timeout",
		"connection",
		"unauthorized",
		"forbidden",
		"unavailable",
		"failed to select",
		"context canceled",
		"context deadline",
		"server error",
		"internal error",
	}

	for _, pattern := range nonRetryablePatterns {
		if strings.Contains(errMsg, pattern) {
			return false
		}
	}

	// These ARE retryable (parameter/validation issues LLM can fix)
	retryablePatterns := []string{
		"invalid",
		"required",
		"missing",
		"validation",
		"schema",
		"type",
		"format",
		"parse",
		"argument",
		"parameter",
	}

	for _, pattern := range retryablePatterns {
		if strings.Contains(errMsg, pattern) {
			return true
		}
	}

	// Default: assume it's retryable unless proven otherwise
	// This allows LLM to attempt fixes for unknown error types
	return true
}

// shouldRetrySequential checks if errors are retryable (parameter issues, not network/service issues)
func (ast *Assistant) shouldRetrySequential(results []ToolCallResult) bool {
	// Check if any result has a retryable error
	hasRetryable := false
	for _, result := range results {
		if result.Error != nil && result.IsRetryableError {
			hasRetryable = true
			break
		}
	}
	return hasRetryable
}

// executeServerToolsParallelWithTrace executes tools for a single server in parallel with trace
func (ast *Assistant) executeServerToolsParallelWithTrace(mcpCtx context.Context, ctx *agentContext.Context, trace types.Manager, client mcp.Client, serverID string, toolCalls []agentContext.ToolCall) ([]ToolCallResult, bool) {
	// Prepare parallel trace inputs
	var parallelInputs []types.TraceParallelInput
	mcpCalls := make([]mcpTypes.ToolCall, 0, len(toolCalls))
	callMap := make(map[string]agentContext.ToolCall)

	for _, tc := range toolCalls {
		_, toolName, ok := ParseMCPToolName(tc.Function.Name)
		if !ok {
			continue
		}

		var args map[string]interface{}
		if tc.Function.Arguments != "" {
			if err := jsoniter.UnmarshalFromString(tc.Function.Arguments, &args); err != nil {
				ctx.Logger.Error("Failed to parse arguments for %s: %v", toolName, err)
				continue
			}
		}

		mcpCalls = append(mcpCalls, mcpTypes.ToolCall{
			Name:      toolName,
			Arguments: args,
		})
		callMap[toolName] = tc

		// Add trace input for this tool
		parallelInputs = append(parallelInputs, types.TraceParallelInput{
			Input: map[string]any{
				"tool_call_id": tc.ID,
				"server":       serverID,
				"tool":         toolName,
				"arguments":    tc.Function.Arguments,
			},
			Option: types.TraceNodeOption{
				Label:       fmt.Sprintf("Tool: %s", toolName),
				Type:        "mcp_tool",
				Icon:        "build",
				Description: fmt.Sprintf("Calling MCP tool '%s' on server '%s'", toolName, serverID),
			},
		})
	}

	// Create parallel trace nodes
	var toolNodes []types.Node
	if trace != nil && len(parallelInputs) > 0 {
		var err error
		toolNodes, err = trace.Parallel(parallelInputs)
		if err != nil {
			ctx.Logger.Debug("trace.Parallel() failed: %v", err)
		}
	}

	// Call tools in parallel with agent context as extra argument
	ctx.Logger.Debug("Calling %d tools in parallel on server '%s'", len(mcpCalls), serverID)

	// Pass agent context as extra argument (only used for Process transport)
	mcpResponse, err := client.CallToolsParallel(mcpCtx, mcpCalls, ctx)
	if err != nil {
		ctx.Logger.Error("Parallel call failed: %v", err)
		// Mark all trace nodes as failed
		for _, node := range toolNodes {
			if node != nil {
				node.Fail(err)
			}
		}
		return nil, true
	}

	// Process results
	results := make([]ToolCallResult, 0, len(mcpResponse.Results))
	hasErrors := false

	for i, mcpResult := range mcpResponse.Results {
		toolName := mcpCalls[i].Name
		originalCall := callMap[toolName]
		var toolNode types.Node
		if i < len(toolNodes) {
			toolNode = toolNodes[i]
		}

		result := ToolCallResult{
			ToolCallID: originalCall.ID,
			Name:       originalCall.Function.Name,
		}

		// Serialize content
		contentBytes, err := jsoniter.Marshal(mcpResult.Content)
		if err != nil {
			result.Error = fmt.Errorf("failed to serialize result: %w", err)
			result.Content = result.Error.Error()
			result.IsRetryableError = false // Serialization error is not retryable
			hasErrors = true
			if toolNode != nil {
				toolNode.Fail(result.Error)
			}
		} else {
			result.Content = string(contentBytes)

			// Check if it's an error result
			if mcpResult.IsError {
				result.Error = fmt.Errorf("tool call error: %s", result.Content)
				result.IsRetryableError = isRetryableToolError(result.Error)
				hasErrors = true
				ctx.Logger.Error("Tool call failed: %s - %s (retryable: %v)", toolName, result.Content, result.IsRetryableError)
				if toolNode != nil {
					toolNode.Fail(result.Error)
				}
			} else {
				// Success
				if toolNode != nil {
					toolNode.Complete(map[string]any{
						"result": mcpResult.Content,
					})
				}
			}
		}

		results = append(results, result)
	}

	return results, hasErrors
}

// executeServerToolsSequentialWithTrace executes tools for a single server sequentially with trace
func (ast *Assistant) executeServerToolsSequentialWithTrace(mcpCtx context.Context, ctx *agentContext.Context, trace types.Manager, client mcp.Client, serverID string, toolCalls []agentContext.ToolCall) ([]ToolCallResult, bool) {
	results := make([]ToolCallResult, 0, len(toolCalls))
	hasErrors := false

	ctx.Logger.Debug("Calling %d tools sequentially on server '%s'", len(toolCalls), serverID)

	for _, tc := range toolCalls {
		_, toolName, ok := ParseMCPToolName(tc.Function.Name)
		if !ok {
			results = append(results, ToolCallResult{
				ToolCallID: tc.ID,
				Name:       tc.Function.Name,
				Content:    fmt.Sprintf("Invalid tool name format: %s", tc.Function.Name),
				Error:      fmt.Errorf("invalid tool name format"),
			})
			hasErrors = true
			continue
		}

		// Get tool schema for validation
		toolsResponse, err := client.ListTools(mcpCtx, "")
		var toolSchema interface{}
		if err == nil {
			for _, t := range toolsResponse.Tools {
				if t.Name == toolName {
					toolSchema = t.InputSchema
					break
				}
			}
		}

		// Add trace node for this tool call
		var toolNode types.Node
		if trace != nil {
			toolNode, _ = trace.Add(
				map[string]any{
					"tool_call_id": tc.ID,
					"server":       serverID,
					"tool":         toolName,
					"arguments":    tc.Function.Arguments,
				},
				types.TraceNodeOption{
					Label:       fmt.Sprintf("Tool: %s (sequential retry)", toolName),
					Type:        "mcp_tool",
					Icon:        "build",
					Description: fmt.Sprintf("Retrying MCP tool '%s' on server '%s' sequentially", toolName, serverID),
				},
			)
		}

		// Parse arguments with repair support
		var args map[string]interface{}
		if tc.Function.Arguments != "" {
			parsed, err := gouJson.Parse(tc.Function.Arguments)
			if err != nil {
				result := ToolCallResult{
					ToolCallID:       tc.ID,
					Name:             tc.Function.Name,
					Content:          fmt.Sprintf("Failed to parse arguments: %v", err),
					Error:            err,
					IsRetryableError: true, // Parsing error is retryable
				}
				results = append(results, result)
				hasErrors = true
				if toolNode != nil {
					toolNode.Fail(err)
				}
				continue
			}

			// Convert to map
			if argsMap, ok := parsed.(map[string]interface{}); ok {
				args = argsMap
			} else {
				err := fmt.Errorf("arguments must be an object, got %T", parsed)
				result := ToolCallResult{
					ToolCallID:       tc.ID,
					Name:             tc.Function.Name,
					Content:          err.Error(),
					Error:            err,
					IsRetryableError: true, // Type error is retryable
				}
				results = append(results, result)
				hasErrors = true
				if toolNode != nil {
					toolNode.Fail(err)
				}
				continue
			}

			// Validate arguments against tool schema if available
			if toolSchema != nil {
				if err := gouJson.Validate(args, toolSchema); err != nil {
					result := ToolCallResult{
						ToolCallID:       tc.ID,
						Name:             tc.Function.Name,
						Content:          fmt.Sprintf("Argument validation failed: %v", err),
						Error:            err,
						IsRetryableError: true, // Validation error is retryable
					}
					results = append(results, result)
					hasErrors = true
					if toolNode != nil {
						toolNode.Fail(err)
					}
					continue
				}
			}
		}

		// Call single tool with agent context as extra argument
		ctx.Logger.Debug("Calling tool: %s", toolName)
		mcpResult, err := client.CallTool(mcpCtx, toolName, args, ctx)

		result := ToolCallResult{
			ToolCallID: tc.ID,
			Name:       tc.Function.Name,
		}

		if err != nil {
			result.Error = err
			result.Content = fmt.Sprintf("Tool call failed: %v", err)
			result.IsRetryableError = isRetryableToolError(err)
			hasErrors = true
			ctx.Logger.Error("Tool call failed: %s - %v (retryable: %v)", toolName, err, result.IsRetryableError)
			if toolNode != nil {
				toolNode.Fail(err)
			}
		} else {
			// Check if result is an error
			if mcpResult.IsError {
				result.Error = fmt.Errorf("MCP tool error")
				result.IsRetryableError = false // MCP internal error is not retryable
				hasErrors = true
			}

			// Serialize the Content field only ([]ToolContent)
			contentBytes, err := jsoniter.Marshal(mcpResult.Content)
			if err != nil {
				result.Error = err
				result.Content = fmt.Sprintf("Failed to serialize result: %v", err)
				result.IsRetryableError = false // Serialization error is not retryable
				hasErrors = true
				if toolNode != nil {
					toolNode.Fail(err)
				}
			} else {
				result.Content = string(contentBytes)
				if toolNode != nil {
					toolNode.Complete(map[string]any{
						"result": mcpResult.Content,
					})
				}
			}
		}

		results = append(results, result)
	}

	return results, hasErrors
}
