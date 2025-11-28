package context

import (
	"fmt"

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
