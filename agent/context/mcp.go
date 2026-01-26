package context

import (
	"fmt"

	jsoniter "github.com/json-iterator/go"
	"github.com/yaoapp/gou/mcp"
	"github.com/yaoapp/gou/mcp/types"
	"github.com/yaoapp/yao/agent/i18n"
	traceTypes "github.com/yaoapp/yao/trace/types"
)

// MCP Client Operations with automatic trace logging and resource management

// Resource Operations
// ==================

// ListResources lists all available resources from an MCP client
// Automatically creates trace node and handles client lifecycle
func (ctx *Context) ListResources(mcpID string, cursor string) (*types.ListResourcesResponse, error) {
	// Get MCP client
	client, err := mcp.Select(mcpID)
	if err != nil {
		return nil, fmt.Errorf("failed to select MCP client '%s': %w", mcpID, err)
	}

	// Get client label for display
	clientLabel := client.GetMetaInfo().Label
	if clientLabel == "" {
		clientLabel = mcpID
	}

	// Get trace manager
	trace, _ := ctx.Trace()

	// Create trace node
	var node traceTypes.Node
	if trace != nil {
		node, _ = trace.Add(
			map[string]any{
				"mcp":    mcpID,
				"cursor": cursor,
			},
			traceTypes.TraceNodeOption{
				Label:       i18n.T(ctx.Locale, "mcp.list_resources.label"), // "MCP: List Resources"
				Type:        "mcp",
				Icon:        "list",
				Description: fmt.Sprintf(i18n.T(ctx.Locale, "mcp.list_resources.description"), clientLabel), // "List resources from MCP client '%s'"
			},
		)
	}

	// Call ListResources
	result, err := client.ListResources(ctx.Context, cursor)
	if err != nil {
		if node != nil {
			node.Fail(err)
		}
		return nil, err
	}

	// Complete trace node with result
	if node != nil {
		node.Complete(map[string]any{
			"resources":  len(result.Resources),
			"nextCursor": result.NextCursor,
		})
	}

	return result, nil
}

// ReadResource reads a specific resource from an MCP client
// Automatically creates trace node and handles client lifecycle
func (ctx *Context) ReadResource(mcpID string, uri string) (*types.ReadResourceResponse, error) {
	// Get MCP client
	client, err := mcp.Select(mcpID)
	if err != nil {
		return nil, fmt.Errorf("failed to select MCP client '%s': %w", mcpID, err)
	}

	// Get client label for display
	clientLabel := client.GetMetaInfo().Label
	if clientLabel == "" {
		clientLabel = mcpID
	}

	// Get trace manager
	trace, _ := ctx.Trace()

	// Create trace node
	var node traceTypes.Node
	if trace != nil {
		node, _ = trace.Add(
			map[string]any{
				"mcp": mcpID,
				"uri": uri,
			},
			traceTypes.TraceNodeOption{
				Label:       i18n.T(ctx.Locale, "mcp.read_resource.label"), // "MCP: Read Resource"
				Type:        "mcp",
				Icon:        "description",
				Description: fmt.Sprintf(i18n.T(ctx.Locale, "mcp.read_resource.description"), uri, clientLabel), // "Read resource '%s' from MCP client '%s'"
			},
		)
	}

	// Call ReadResource
	result, err := client.ReadResource(ctx.Context, uri)
	if err != nil {
		if node != nil {
			node.Fail(err)
		}
		return nil, err
	}

	// Complete trace node with result
	if node != nil {
		node.Complete(map[string]any{
			"contents": len(result.Contents),
		})
	}

	return result, nil
}

// Tool Operations
// ===============

// ListTools lists all available tools from an MCP client
// Automatically creates trace node and handles client lifecycle
func (ctx *Context) ListTools(mcpID string, cursor string) (*types.ListToolsResponse, error) {
	// Get MCP client
	client, err := mcp.Select(mcpID)
	if err != nil {
		return nil, fmt.Errorf("failed to select MCP client '%s': %w", mcpID, err)
	}

	// Get client label for display
	clientLabel := client.GetMetaInfo().Label
	if clientLabel == "" {
		clientLabel = mcpID
	}

	// Get trace manager
	trace, _ := ctx.Trace()

	// Create trace node
	var node traceTypes.Node
	if trace != nil {
		node, _ = trace.Add(
			map[string]any{
				"mcp":    mcpID,
				"cursor": cursor,
			},
			traceTypes.TraceNodeOption{
				Label:       i18n.T(ctx.Locale, "mcp.list_tools.label"), // "MCP: List Tools"
				Type:        "mcp",
				Icon:        "build",
				Description: fmt.Sprintf(i18n.T(ctx.Locale, "mcp.list_tools.description"), clientLabel), // "List tools from MCP client '%s'"
			},
		)
	}

	// Call ListTools
	result, err := client.ListTools(ctx.Context, cursor)
	if err != nil {
		if node != nil {
			node.Fail(err)
		}
		return nil, err
	}

	// Complete trace node with result
	if node != nil {
		node.Complete(map[string]any{
			"tools":      len(result.Tools),
			"nextCursor": result.NextCursor,
		})
	}

	return result, nil
}

// CallTool calls a single tool from an MCP client
// Automatically creates trace node and handles client lifecycle
func (ctx *Context) CallTool(mcpID string, name string, arguments interface{}) (*types.CallToolResponse, error) {
	// Get MCP client
	client, err := mcp.Select(mcpID)
	if err != nil {
		return nil, fmt.Errorf("failed to select MCP client '%s': %w", mcpID, err)
	}

	// Get client label for display
	clientLabel := client.GetMetaInfo().Label
	if clientLabel == "" {
		clientLabel = mcpID
	}

	// Get trace manager
	trace, _ := ctx.Trace()

	// Create trace node
	var node traceTypes.Node
	if trace != nil {
		node, _ = trace.Add(
			map[string]any{
				"mcp":       mcpID,
				"tool":      name,
				"arguments": arguments,
			},
			traceTypes.TraceNodeOption{
				Label:       i18n.T(ctx.Locale, "mcp.call_tool.label"), // "MCP: Call Tool"
				Type:        "mcp",
				Icon:        "settings",
				Description: fmt.Sprintf(i18n.T(ctx.Locale, "mcp.call_tool.description"), name, clientLabel), // "Call tool '%s' from MCP client '%s'"
			},
		)
	}

	// Call tool
	result, err := client.CallTool(ctx.Context, name, arguments)
	if err != nil {
		if node != nil {
			node.Fail(err)
		}
		return nil, err
	}

	// Complete trace node with result
	if node != nil {
		node.Complete(map[string]any{
			"contents": len(result.Content),
		})
	}

	return result, nil
}

// CallTools calls multiple tools sequentially from an MCP client
// Automatically creates trace node and handles client lifecycle
func (ctx *Context) CallTools(mcpID string, tools []types.ToolCall) (*types.CallToolsResponse, error) {
	// Get MCP client
	client, err := mcp.Select(mcpID)
	if err != nil {
		return nil, fmt.Errorf("failed to select MCP client '%s': %w", mcpID, err)
	}

	// Get client label for display
	clientLabel := client.GetMetaInfo().Label
	if clientLabel == "" {
		clientLabel = mcpID
	}

	// Get trace manager
	trace, _ := ctx.Trace()

	// Create trace node
	var node traceTypes.Node
	if trace != nil {
		node, _ = trace.Add(
			map[string]any{
				"mcp":   mcpID,
				"tools": tools,
				"count": len(tools),
			},
			traceTypes.TraceNodeOption{
				Label:       i18n.T(ctx.Locale, "mcp.call_tools.label"), // "MCP: Call Tools"
				Type:        "mcp",
				Icon:        "settings",
				Description: fmt.Sprintf(i18n.T(ctx.Locale, "mcp.call_tools.description"), len(tools), clientLabel), // "Call %d tools sequentially from MCP client '%s'"
			},
		)
	}

	// Call tools sequentially
	result, err := client.CallTools(ctx.Context, tools)
	if err != nil {
		if node != nil {
			node.Fail(err)
		}
		return nil, err
	}

	// Complete trace node with result
	if node != nil {
		node.Complete(map[string]any{
			"results": len(result.Results),
		})
	}

	return result, nil
}

// CallToolsParallel calls multiple tools in parallel from an MCP client
// Automatically creates trace node and handles client lifecycle
func (ctx *Context) CallToolsParallel(mcpID string, tools []types.ToolCall) (*types.CallToolsResponse, error) {
	// Get MCP client
	client, err := mcp.Select(mcpID)
	if err != nil {
		return nil, fmt.Errorf("failed to select MCP client '%s': %w", mcpID, err)
	}

	// Get client label for display
	clientLabel := client.GetMetaInfo().Label
	if clientLabel == "" {
		clientLabel = mcpID
	}

	// Get trace manager
	trace, _ := ctx.Trace()

	// Create trace node
	var node traceTypes.Node
	if trace != nil {
		node, _ = trace.Add(
			map[string]any{
				"mcp":   mcpID,
				"tools": tools,
				"count": len(tools),
			},
			traceTypes.TraceNodeOption{
				Label:       i18n.T(ctx.Locale, "mcp.call_tools_parallel.label"), // "MCP: Call Tools (Parallel)"
				Type:        "mcp",
				Icon:        "settings",
				Description: fmt.Sprintf(i18n.T(ctx.Locale, "mcp.call_tools_parallel.description"), len(tools), clientLabel), // "Call %d tools in parallel from MCP client '%s'"
			},
		)
	}

	// Call tools in parallel
	result, err := client.CallToolsParallel(ctx.Context, tools)
	if err != nil {
		if node != nil {
			node.Fail(err)
		}
		return nil, err
	}

	// Complete trace node with result
	if node != nil {
		node.Complete(map[string]any{
			"results": len(result.Results),
		})
	}

	return result, nil
}

// Prompt Operations
// =================

// ListPrompts lists all available prompts from an MCP client
// Automatically creates trace node and handles client lifecycle
func (ctx *Context) ListPrompts(mcpID string, cursor string) (*types.ListPromptsResponse, error) {
	// Get MCP client
	client, err := mcp.Select(mcpID)
	if err != nil {
		return nil, fmt.Errorf("failed to select MCP client '%s': %w", mcpID, err)
	}

	// Get client label for display
	clientLabel := client.GetMetaInfo().Label
	if clientLabel == "" {
		clientLabel = mcpID
	}

	// Get trace manager
	trace, _ := ctx.Trace()

	// Create trace node
	var node traceTypes.Node
	if trace != nil {
		node, _ = trace.Add(
			map[string]any{
				"mcp":    mcpID,
				"cursor": cursor,
			},
			traceTypes.TraceNodeOption{
				Label:       i18n.T(ctx.Locale, "mcp.list_prompts.label"), // "MCP: List Prompts"
				Type:        "mcp",
				Icon:        "chat",
				Description: fmt.Sprintf(i18n.T(ctx.Locale, "mcp.list_prompts.description"), clientLabel), // "List prompts from MCP client '%s'"
			},
		)
	}

	// Call ListPrompts
	result, err := client.ListPrompts(ctx.Context, cursor)
	if err != nil {
		if node != nil {
			node.Fail(err)
		}
		return nil, err
	}

	// Complete trace node with result
	if node != nil {
		node.Complete(map[string]any{
			"prompts":    len(result.Prompts),
			"nextCursor": result.NextCursor,
		})
	}

	return result, nil
}

// GetPrompt gets a prompt with arguments from an MCP client
// Automatically creates trace node and handles client lifecycle
func (ctx *Context) GetPrompt(mcpID string, name string, arguments map[string]interface{}) (*types.GetPromptResponse, error) {
	// Get MCP client
	client, err := mcp.Select(mcpID)
	if err != nil {
		return nil, fmt.Errorf("failed to select MCP client '%s': %w", mcpID, err)
	}

	// Get client label for display
	clientLabel := client.GetMetaInfo().Label
	if clientLabel == "" {
		clientLabel = mcpID
	}

	// Get trace manager
	trace, _ := ctx.Trace()

	// Create trace node
	var node traceTypes.Node
	if trace != nil {
		node, _ = trace.Add(
			map[string]any{
				"mcp":       mcpID,
				"prompt":    name,
				"arguments": arguments,
			},
			traceTypes.TraceNodeOption{
				Label:       i18n.T(ctx.Locale, "mcp.get_prompt.label"), // "MCP: Get Prompt"
				Type:        "mcp",
				Icon:        "chat",
				Description: fmt.Sprintf(i18n.T(ctx.Locale, "mcp.get_prompt.description"), name, clientLabel), // "Get prompt '%s' from MCP client '%s'"
			},
		)
	}

	// Get prompt
	result, err := client.GetPrompt(ctx.Context, name, arguments)
	if err != nil {
		if node != nil {
			node.Fail(err)
		}
		return nil, err
	}

	// Complete trace node with result
	if node != nil {
		node.Complete(map[string]any{
			"messages": len(result.Messages),
		})
	}

	return result, nil
}

// Sample Operations
// =================

// ListSamples lists samples for a tool or resource from an MCP client
// Automatically creates trace node and handles client lifecycle
func (ctx *Context) ListSamples(mcpID string, itemType types.SampleItemType, itemName string) (*types.ListSamplesResponse, error) {
	// Get MCP client
	client, err := mcp.Select(mcpID)
	if err != nil {
		return nil, fmt.Errorf("failed to select MCP client '%s': %w", mcpID, err)
	}

	// Get client label for display
	clientLabel := client.GetMetaInfo().Label
	if clientLabel == "" {
		clientLabel = mcpID
	}

	// Get trace manager
	trace, _ := ctx.Trace()

	// Create trace node
	var node traceTypes.Node
	if trace != nil {
		node, _ = trace.Add(
			map[string]any{
				"mcp":      mcpID,
				"itemType": itemType,
				"itemName": itemName,
			},
			traceTypes.TraceNodeOption{
				Label:       i18n.T(ctx.Locale, "mcp.list_samples.label"), // "MCP: List Samples"
				Type:        "mcp",
				Icon:        "library_books",
				Description: fmt.Sprintf(i18n.T(ctx.Locale, "mcp.list_samples.description"), itemName, clientLabel), // "List samples for '%s' from MCP client '%s'"
			},
		)
	}

	// Call ListSamples
	result, err := client.ListSamples(ctx.Context, itemType, itemName)
	if err != nil {
		if node != nil {
			node.Fail(err)
		}
		return nil, err
	}

	// Complete trace node with result
	if node != nil {
		node.Complete(map[string]any{
			"samples": len(result.Samples),
		})
	}

	return result, nil
}

// GetSample gets a specific sample by index from an MCP client
// Automatically creates trace node and handles client lifecycle
func (ctx *Context) GetSample(mcpID string, itemType types.SampleItemType, itemName string, index int) (*types.SampleData, error) {
	// Get MCP client
	client, err := mcp.Select(mcpID)
	if err != nil {
		return nil, fmt.Errorf("failed to select MCP client '%s': %w", mcpID, err)
	}

	// Get client label for display
	clientLabel := client.GetMetaInfo().Label
	if clientLabel == "" {
		clientLabel = mcpID
	}

	// Get trace manager
	trace, _ := ctx.Trace()

	// Create trace node
	var node traceTypes.Node
	if trace != nil {
		node, _ = trace.Add(
			map[string]any{
				"mcp":      mcpID,
				"itemType": itemType,
				"itemName": itemName,
				"index":    index,
			},
			traceTypes.TraceNodeOption{
				Label:       i18n.T(ctx.Locale, "mcp.get_sample.label"), // "MCP: Get Sample"
				Type:        "mcp",
				Icon:        "library_books",
				Description: fmt.Sprintf(i18n.T(ctx.Locale, "mcp.get_sample.description"), index, itemName, clientLabel), // "Get sample #%d for '%s' from MCP client '%s'"
			},
		)
	}

	// Get sample
	result, err := client.GetSample(ctx.Context, itemType, itemName, index)
	if err != nil {
		if node != nil {
			node.Fail(err)
		}
		return nil, err
	}

	// Complete trace node with result
	if node != nil {
		node.Complete(result)
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
