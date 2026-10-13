package context

import (
	"fmt"

	jsoniter "github.com/json-iterator/go"
	"github.com/yaoapp/gou/mcp"
	"github.com/yaoapp/gou/mcp/types"
)

// MCP Client Operations with resource management

// Resource Operations
// ==================

// ListResources lists all available resources from an MCP client
// Handles client lifecycle
func (ctx *Context) ListResources(mcpID string, cursor string) (*types.ListResourcesResponse, error) {
	// Get MCP client
	client, err := mcp.Select(mcpID)
	if err != nil {
		return nil, fmt.Errorf("failed to select MCP client '%s': %w", mcpID, err)
	}

	// Call ListResources
	result, err := client.ListResources(ctx.Context, cursor)
	if err != nil {
		return nil, err
	}

	return result, nil
}

// ReadResource reads a specific resource from an MCP client
// Handles client lifecycle
func (ctx *Context) ReadResource(mcpID string, uri string) (*types.ReadResourceResponse, error) {
	// Get MCP client
	client, err := mcp.Select(mcpID)
	if err != nil {
		return nil, fmt.Errorf("failed to select MCP client '%s': %w", mcpID, err)
	}

	// Call ReadResource
	result, err := client.ReadResource(ctx.Context, uri)
	if err != nil {
		return nil, err
	}

	return result, nil
}

// Tool Operations
// ===============

// ListTools lists all available tools from an MCP client
// Handles client lifecycle
func (ctx *Context) ListTools(mcpID string, cursor string) (*types.ListToolsResponse, error) {
	// Get MCP client
	client, err := mcp.Select(mcpID)
	if err != nil {
		return nil, fmt.Errorf("failed to select MCP client '%s': %w", mcpID, err)
	}

	// Call ListTools
	result, err := client.ListTools(ctx.Context, cursor)
	if err != nil {
		return nil, err
	}

	return result, nil
}

// CallTool calls a single tool from an MCP client
// Handles client lifecycle
func (ctx *Context) CallTool(mcpID string, name string, arguments interface{}) (*types.CallToolResponse, error) {
	// Get MCP client
	client, err := mcp.Select(mcpID)
	if err != nil {
		return nil, fmt.Errorf("failed to select MCP client '%s': %w", mcpID, err)
	}

	// Call tool (pass ctx as extraArgs for Process transport to propagate Authorized())
	result, err := client.CallTool(ctx.Context, name, arguments, ctx)
	if err != nil {
		return nil, err
	}

	return result, nil
}

// CallTools calls multiple tools sequentially from an MCP client
// Handles client lifecycle
func (ctx *Context) CallTools(mcpID string, tools []types.ToolCall) (*types.CallToolsResponse, error) {
	// Get MCP client
	client, err := mcp.Select(mcpID)
	if err != nil {
		return nil, fmt.Errorf("failed to select MCP client '%s': %w", mcpID, err)
	}

	// Call tools sequentially (pass ctx as extraArgs for Process transport to propagate Authorized())
	result, err := client.CallTools(ctx.Context, tools, ctx)
	if err != nil {
		return nil, err
	}

	return result, nil
}

// CallToolsParallel calls multiple tools in parallel from an MCP client
// Handles client lifecycle
func (ctx *Context) CallToolsParallel(mcpID string, tools []types.ToolCall) (*types.CallToolsResponse, error) {
	// Get MCP client
	client, err := mcp.Select(mcpID)
	if err != nil {
		return nil, fmt.Errorf("failed to select MCP client '%s': %w", mcpID, err)
	}

	// Call tools in parallel (pass ctx as extraArgs for Process transport to propagate Authorized())
	result, err := client.CallToolsParallel(ctx.Context, tools, ctx)
	if err != nil {
		return nil, err
	}

	return result, nil
}

// Prompt Operations
// =================

// ListPrompts lists all available prompts from an MCP client
// Handles client lifecycle
func (ctx *Context) ListPrompts(mcpID string, cursor string) (*types.ListPromptsResponse, error) {
	// Get MCP client
	client, err := mcp.Select(mcpID)
	if err != nil {
		return nil, fmt.Errorf("failed to select MCP client '%s': %w", mcpID, err)
	}

	// Call ListPrompts
	result, err := client.ListPrompts(ctx.Context, cursor)
	if err != nil {
		return nil, err
	}

	return result, nil
}

// GetPrompt gets a prompt with arguments from an MCP client
// Handles client lifecycle
func (ctx *Context) GetPrompt(mcpID string, name string, arguments map[string]interface{}) (*types.GetPromptResponse, error) {
	// Get MCP client
	client, err := mcp.Select(mcpID)
	if err != nil {
		return nil, fmt.Errorf("failed to select MCP client '%s': %w", mcpID, err)
	}

	// Get prompt
	result, err := client.GetPrompt(ctx.Context, name, arguments)
	if err != nil {
		return nil, err
	}

	return result, nil
}

// Sample Operations
// =================

// ListSamples lists samples for a tool or resource from an MCP client
// Handles client lifecycle
func (ctx *Context) ListSamples(mcpID string, itemType types.SampleItemType, itemName string) (*types.ListSamplesResponse, error) {
	// Get MCP client
	client, err := mcp.Select(mcpID)
	if err != nil {
		return nil, fmt.Errorf("failed to select MCP client '%s': %w", mcpID, err)
	}

	// Call ListSamples
	result, err := client.ListSamples(ctx.Context, itemType, itemName)
	if err != nil {
		return nil, err
	}

	return result, nil
}

// GetSample gets a specific sample by index from an MCP client
// Handles client lifecycle
func (ctx *Context) GetSample(mcpID string, itemType types.SampleItemType, itemName string, index int) (*types.SampleData, error) {
	// Get MCP client
	client, err := mcp.Select(mcpID)
	if err != nil {
		return nil, fmt.Errorf("failed to select MCP client '%s': %w", mcpID, err)
	}

	// Get sample
	result, err := client.GetSample(ctx.Context, itemType, itemName, index)
	if err != nil {
		return nil, err
	}

	return result, nil
}

// Single-Server Tool Response Helpers
// ====================================

// parseCallToolResponse parses a CallToolResponse and returns the parsed content directly
func parseCallToolResponse(response *types.CallToolResponse) interface{} {
	if response == nil {
		return nil
	}
	return parseToolResponseContent(response)
}

// parseCallToolsResponse parses a CallToolsResponse and returns an array of parsed results
func parseCallToolsResponse(response *types.CallToolsResponse) []interface{} {
	if response == nil {
		return nil
	}
	results := make([]interface{}, len(response.Results))
	for i, r := range response.Results {
		results[i] = parseToolResponseContent(&r)
	}
	return results
}

// Cross-Server Tool Operations
// ============================

// MCPToolRequest represents a request to call a tool on a specific MCP server
type MCPToolRequest struct {
	MCP       string      `json:"mcp"`       // MCP server ID
	Tool      string      `json:"tool"`      // Tool name
	Arguments interface{} `json:"arguments"` // Tool arguments
}

// MCPToolResult represents the result of a cross-server tool call
// Returns parsed result directly, with error field for failures
type MCPToolResult struct {
	MCP    string      `json:"mcp"`              // MCP server ID
	Tool   string      `json:"tool"`             // Tool name
	Result interface{} `json:"result,omitempty"` // Parsed result content (directly usable)
	Error  string      `json:"error,omitempty"`  // Error message (on failure)
}

// callToolResult is used internally to pass results through channels
type callToolResult struct {
	idx    int
	result *MCPToolResult
}

// CallToolAll calls tools on multiple MCP servers concurrently and waits for all to complete
// Returns results in the same order as requests, regardless of completion order (like Promise.all)
func (ctx *Context) CallToolAll(requests []*MCPToolRequest) []*MCPToolResult {
	if len(requests) == 0 {
		return []*MCPToolResult{}
	}

	results := make([]*MCPToolResult, len(requests))
	done := make(chan struct{})
	remaining := len(requests)

	for i, req := range requests {
		go func(idx int, r *MCPToolRequest) {
			defer func() {
				if err := recover(); err != nil {
					results[idx] = &MCPToolResult{
						MCP:   r.MCP,
						Tool:  r.Tool,
						Error: fmt.Sprintf("panic: %v", err),
					}
				}
				done <- struct{}{}
			}()

			results[idx] = ctx.callToolSingle(r)
		}(i, req)
	}

	// Wait for all to complete
	for remaining > 0 {
		<-done
		remaining--
	}

	return results
}

// CallToolAny calls tools on multiple MCP servers concurrently and returns when any succeeds
// Returns all results received so far when first success is found (like Promise.any)
func (ctx *Context) CallToolAny(requests []*MCPToolRequest) []*MCPToolResult {
	if len(requests) == 0 {
		return []*MCPToolResult{}
	}

	resultChan := make(chan callToolResult, len(requests))
	remaining := len(requests)

	for i, req := range requests {
		go func(idx int, r *MCPToolRequest) {
			defer func() {
				if err := recover(); err != nil {
					resultChan <- callToolResult{
						idx: idx,
						result: &MCPToolResult{
							MCP:   r.MCP,
							Tool:  r.Tool,
							Error: fmt.Sprintf("panic: %v", err),
						},
					}
				}
			}()

			resultChan <- callToolResult{idx: idx, result: ctx.callToolSingle(r)}
		}(i, req)
	}

	// Collect results until we find a success or all fail
	results := make([]*MCPToolResult, len(requests))

	for remaining > 0 {
		cr := <-resultChan
		remaining--
		results[cr.idx] = cr.result

		// Check if this is a success (no error)
		if cr.result.Error == "" {
			break // Stop waiting, we have a success
		}
	}

	// Drain remaining results in background (don't block)
	if remaining > 0 {
		go func(count int) {
			for i := 0; i < count; i++ {
				<-resultChan
			}
		}(remaining)
	}

	return results
}

// CallToolRace calls tools on multiple MCP servers concurrently and returns when any completes
// Returns all results received so far when first completion (like Promise.race)
func (ctx *Context) CallToolRace(requests []*MCPToolRequest) []*MCPToolResult {
	if len(requests) == 0 {
		return []*MCPToolResult{}
	}

	resultChan := make(chan callToolResult, len(requests))
	remaining := len(requests)

	for i, req := range requests {
		go func(idx int, r *MCPToolRequest) {
			defer func() {
				if err := recover(); err != nil {
					resultChan <- callToolResult{
						idx: idx,
						result: &MCPToolResult{
							MCP:   r.MCP,
							Tool:  r.Tool,
							Error: fmt.Sprintf("panic: %v", err),
						},
					}
				}
			}()

			resultChan <- callToolResult{idx: idx, result: ctx.callToolSingle(r)}
		}(i, req)
	}

	// Get first result (success or failure)
	results := make([]*MCPToolResult, len(requests))
	cr := <-resultChan
	remaining--
	results[cr.idx] = cr.result

	// Drain remaining results in background (don't block)
	if remaining > 0 {
		go func(count int) {
			for i := 0; i < count; i++ {
				<-resultChan
			}
		}(remaining)
	}

	return results
}

// callToolSingle executes a single tool call on an MCP server
// This is a helper method for the parallel call methods
func (ctx *Context) callToolSingle(req *MCPToolRequest) *MCPToolResult {
	result := &MCPToolResult{
		MCP:  req.MCP,
		Tool: req.Tool,
	}

	// Call the tool using existing CallTool method
	response, err := ctx.CallTool(req.MCP, req.Tool, req.Arguments)
	if err != nil {
		result.Error = err.Error()
		return result
	}

	// Parse and return result directly
	result.Result = parseToolResponseContent(response)
	return result
}

// parseToolResponseContent extracts and parses the actual content from a CallToolResponse
// Similar to ToolCallResult.ParsedContent() in assistant/types.go
// - For "text" type, parses the Text field as JSON (or returns as string if not JSON)
// - For "image" type, returns the Data and MimeType
// - For "resource" type, returns the Resource object
// - If only one content item, returns it directly (not as array)
func parseToolResponseContent(response *types.CallToolResponse) interface{} {
	if response == nil || len(response.Content) == 0 {
		return nil
	}

	var results []interface{}
	for _, tc := range response.Content {
		switch tc.Type {
		case types.ToolContentTypeText:
			// For text type, try to parse as JSON
			if tc.Text != "" {
				var parsed interface{}
				if err := jsoniter.UnmarshalFromString(tc.Text, &parsed); err == nil {
					results = append(results, parsed)
				} else {
					// If not JSON, return as plain string
					results = append(results, tc.Text)
				}
			}
		case types.ToolContentTypeImage:
			// For image type, return data and mimeType
			results = append(results, map[string]interface{}{
				"type":     "image",
				"data":     tc.Data,
				"mimeType": tc.MimeType,
			})
		case types.ToolContentTypeResource:
			// For resource type, return the resource object
			if tc.Resource != nil {
				results = append(results, tc.Resource)
			}
		default:
			// Unknown type, include as-is with type info
			results = append(results, map[string]interface{}{
				"type": tc.Type,
				"text": tc.Text,
			})
		}
	}

	// If only one result, return it directly (not as array)
	if len(results) == 1 {
		return results[0]
	}

	return results
}
