package context

import (
	"fmt"

	"github.com/yaoapp/gou/mcp/types"
	"github.com/yaoapp/gou/runtime/v8/bridge"
	"rogchap.com/v8go"
)

// Suppress unused import warning - types is used in other functions
var _ = types.ToolCall{}

// MCP JavaScript API methods
// These methods expose MCP functionality to JavaScript runtime

// mcpListResourcesMethod implements ctx.MCP.ListResources(mcp, cursor)
// Lists all available resources from an MCP client
func (ctx *Context) mcpListResourcesMethod(iso *v8go.Isolate) *v8go.FunctionTemplate {
	return v8go.NewFunctionTemplate(iso, func(info *v8go.FunctionCallbackInfo) *v8go.Value {
		v8ctx := info.Context()
		args := info.Args()

		if len(args) < 1 {
			return bridge.JsException(v8ctx, "ListResources requires mcp parameter")
		}

		mcpID := args[0].String()
		cursor := ""
		if len(args) >= 2 && !args[1].IsUndefined() {
			cursor = args[1].String()
		}

		result, err := ctx.ListResources(mcpID, cursor)
		if err != nil {
			return bridge.JsException(v8ctx, err.Error())
		}

		jsVal, err := bridge.JsValue(v8ctx, result)
		if err != nil {
			return bridge.JsException(v8ctx, err.Error())
		}

		return jsVal
	})
}

// mcpReadResourceMethod implements ctx.MCP.ReadResource(mcp, uri)
// Reads a specific resource from an MCP client
func (ctx *Context) mcpReadResourceMethod(iso *v8go.Isolate) *v8go.FunctionTemplate {
	return v8go.NewFunctionTemplate(iso, func(info *v8go.FunctionCallbackInfo) *v8go.Value {
		v8ctx := info.Context()
		args := info.Args()

		if len(args) < 2 {
			return bridge.JsException(v8ctx, "ReadResource requires mcp and uri parameters")
		}

		mcpID := args[0].String()
		uri := args[1].String()

		result, err := ctx.ReadResource(mcpID, uri)
		if err != nil {
			return bridge.JsException(v8ctx, err.Error())
		}

		jsVal, err := bridge.JsValue(v8ctx, result)
		if err != nil {
			return bridge.JsException(v8ctx, err.Error())
		}

		return jsVal
	})
}

// mcpListToolsMethod implements ctx.MCP.ListTools(mcp, cursor)
// Lists all available tools from an MCP client
func (ctx *Context) mcpListToolsMethod(iso *v8go.Isolate) *v8go.FunctionTemplate {
	return v8go.NewFunctionTemplate(iso, func(info *v8go.FunctionCallbackInfo) *v8go.Value {
		v8ctx := info.Context()
		args := info.Args()

		if len(args) < 1 {
			return bridge.JsException(v8ctx, "ListTools requires mcp parameter")
		}

		mcpID := args[0].String()
		cursor := ""
		if len(args) >= 2 && !args[1].IsUndefined() {
			cursor = args[1].String()
		}

		result, err := ctx.ListTools(mcpID, cursor)
		if err != nil {
			return bridge.JsException(v8ctx, err.Error())
		}

		jsVal, err := bridge.JsValue(v8ctx, result)
		if err != nil {
			return bridge.JsException(v8ctx, err.Error())
		}

		return jsVal
	})
}

// mcpCallToolMethod implements ctx.MCP.CallTool(mcp, name, args)
// Calls a specific tool from an MCP client
// Returns CallToolResult with parsed 'result' field for convenience
func (ctx *Context) mcpCallToolMethod(iso *v8go.Isolate) *v8go.FunctionTemplate {
	return v8go.NewFunctionTemplate(iso, func(info *v8go.FunctionCallbackInfo) *v8go.Value {
		v8ctx := info.Context()
		args := info.Args()

		if len(args) < 2 {
			return bridge.JsException(v8ctx, "CallTool requires mcp and name parameters")
		}

		mcpID := args[0].String()
		toolName := args[1].String()

		// Parse arguments (optional)
		var toolArgs map[string]interface{}
		if len(args) >= 3 && !args[2].IsUndefined() {
			goVal, err := bridge.GoValue(args[2], v8ctx)
			if err != nil {
				return bridge.JsException(v8ctx, "invalid tool arguments: "+err.Error())
			}
			if argsMap, ok := goVal.(map[string]interface{}); ok {
				toolArgs = argsMap
			}
		}

		response, err := ctx.CallTool(mcpID, toolName, toolArgs)
		if err != nil {
			return bridge.JsException(v8ctx, err.Error())
		}

		// Return parsed result directly
		result := parseCallToolResponse(response)

		jsVal, err := bridge.JsValue(v8ctx, result)
		if err != nil {
			return bridge.JsException(v8ctx, err.Error())
		}

		return jsVal
	})
}

// mcpCallToolsMethod implements ctx.MCP.CallTools(mcp, tools)
// Calls multiple tools sequentially from an MCP client
// Returns CallToolsResult with parsed 'result' field in each item
func (ctx *Context) mcpCallToolsMethod(iso *v8go.Isolate) *v8go.FunctionTemplate {
	return v8go.NewFunctionTemplate(iso, func(info *v8go.FunctionCallbackInfo) *v8go.Value {
		v8ctx := info.Context()
		args := info.Args()

		if len(args) < 2 {
			return bridge.JsException(v8ctx, "CallTools requires mcp and tools parameters")
		}

		mcpID := args[0].String()

		// Parse tools array
		goVal, err := bridge.GoValue(args[1], v8ctx)
		if err != nil {
			return bridge.JsException(v8ctx, "invalid tools parameter: "+err.Error())
		}

		toolsArray, ok := goVal.([]interface{})
		if !ok {
			return bridge.JsException(v8ctx, "tools parameter must be an array")
		}

		// Convert to ToolCall array
		tools := make([]types.ToolCall, 0, len(toolsArray))
		for i, item := range toolsArray {
			toolMap, ok := item.(map[string]interface{})
			if !ok {
				return bridge.JsException(v8ctx, "each tool must be an object")
			}

			name, ok := toolMap["name"].(string)
			if !ok {
				return bridge.JsException(v8ctx, "tool name is required")
			}

			toolCall := types.ToolCall{
				Name: name,
			}

			if argsVal, exists := toolMap["arguments"]; exists && argsVal != nil {
				if argsMap, ok := argsVal.(map[string]interface{}); ok {
					toolCall.Arguments = argsMap
				}
			}

			tools = append(tools, toolCall)

			// Suppress unused variable warning
			_ = i
		}

		response, err := ctx.CallTools(mcpID, tools)
		if err != nil {
			return bridge.JsException(v8ctx, err.Error())
		}

		// Return parsed results directly as array
		result := parseCallToolsResponse(response)

		jsVal, err := bridge.JsValue(v8ctx, result)
		if err != nil {
			return bridge.JsException(v8ctx, err.Error())
		}

		return jsVal
	})
}

// mcpCallToolsParallelMethod implements ctx.MCP.CallToolsParallel(mcp, tools)
// Calls multiple tools in parallel from an MCP client
// Returns CallToolsResult with parsed 'result' field in each item
func (ctx *Context) mcpCallToolsParallelMethod(iso *v8go.Isolate) *v8go.FunctionTemplate {
	return v8go.NewFunctionTemplate(iso, func(info *v8go.FunctionCallbackInfo) *v8go.Value {
		v8ctx := info.Context()
		args := info.Args()

		if len(args) < 2 {
			return bridge.JsException(v8ctx, "CallToolsParallel requires mcp and tools parameters")
		}

		mcpID := args[0].String()

		// Parse tools array
		goVal, err := bridge.GoValue(args[1], v8ctx)
		if err != nil {
			return bridge.JsException(v8ctx, "invalid tools parameter: "+err.Error())
		}

		toolsArray, ok := goVal.([]interface{})
		if !ok {
			return bridge.JsException(v8ctx, "tools parameter must be an array")
		}

		// Convert to ToolCall array
		tools := make([]types.ToolCall, 0, len(toolsArray))
		for i, item := range toolsArray {
			toolMap, ok := item.(map[string]interface{})
			if !ok {
				return bridge.JsException(v8ctx, "each tool must be an object")
			}

			name, ok := toolMap["name"].(string)
			if !ok {
				return bridge.JsException(v8ctx, "tool name is required")
			}

			toolCall := types.ToolCall{
				Name: name,
			}

			if argsVal, exists := toolMap["arguments"]; exists && argsVal != nil {
				if argsMap, ok := argsVal.(map[string]interface{}); ok {
					toolCall.Arguments = argsMap
				}
			}

			tools = append(tools, toolCall)

			// Suppress unused variable warning
			_ = i
		}

		response, err := ctx.CallToolsParallel(mcpID, tools)
		if err != nil {
			return bridge.JsException(v8ctx, err.Error())
		}

		// Return parsed results directly as array
		result := parseCallToolsResponse(response)

		jsVal, err := bridge.JsValue(v8ctx, result)
		if err != nil {
			return bridge.JsException(v8ctx, err.Error())
		}

		return jsVal
	})
}

// mcpListPromptsMethod implements ctx.MCP.ListPrompts(mcp, cursor)
// Lists all available prompts from an MCP client
func (ctx *Context) mcpListPromptsMethod(iso *v8go.Isolate) *v8go.FunctionTemplate {
	return v8go.NewFunctionTemplate(iso, func(info *v8go.FunctionCallbackInfo) *v8go.Value {
		v8ctx := info.Context()
		args := info.Args()

		if len(args) < 1 {
			return bridge.JsException(v8ctx, "ListPrompts requires mcp parameter")
		}

		mcpID := args[0].String()
		cursor := ""
		if len(args) >= 2 && !args[1].IsUndefined() {
			cursor = args[1].String()
		}

		result, err := ctx.ListPrompts(mcpID, cursor)
		if err != nil {
			return bridge.JsException(v8ctx, err.Error())
		}

		jsVal, err := bridge.JsValue(v8ctx, result)
		if err != nil {
			return bridge.JsException(v8ctx, err.Error())
		}

		return jsVal
	})
}

// mcpGetPromptMethod implements ctx.MCP.GetPrompt(mcp, name, args)
// Gets a specific prompt from an MCP client
func (ctx *Context) mcpGetPromptMethod(iso *v8go.Isolate) *v8go.FunctionTemplate {
	return v8go.NewFunctionTemplate(iso, func(info *v8go.FunctionCallbackInfo) *v8go.Value {
		v8ctx := info.Context()
		args := info.Args()

		if len(args) < 2 {
			return bridge.JsException(v8ctx, "GetPrompt requires mcp and name parameters")
		}

		mcpID := args[0].String()
		promptName := args[1].String()

		// Parse arguments (optional)
		var promptArgs map[string]interface{}
		if len(args) >= 3 && !args[2].IsUndefined() {
			goVal, err := bridge.GoValue(args[2], v8ctx)
			if err != nil {
				return bridge.JsException(v8ctx, "invalid prompt arguments: "+err.Error())
			}
			if argsMap, ok := goVal.(map[string]interface{}); ok {
				promptArgs = argsMap
			}
		}

		result, err := ctx.GetPrompt(mcpID, promptName, promptArgs)
		if err != nil {
			return bridge.JsException(v8ctx, err.Error())
		}

		jsVal, err := bridge.JsValue(v8ctx, result)
		if err != nil {
			return bridge.JsException(v8ctx, err.Error())
		}

		return jsVal
	})
}

// mcpListSamplesMethod implements ctx.MCP.ListSamples(mcp, type, name)
// Lists all available samples from an MCP client
func (ctx *Context) mcpListSamplesMethod(iso *v8go.Isolate) *v8go.FunctionTemplate {
	return v8go.NewFunctionTemplate(iso, func(info *v8go.FunctionCallbackInfo) *v8go.Value {
		v8ctx := info.Context()
		args := info.Args()

		if len(args) < 3 {
			return bridge.JsException(v8ctx, "ListSamples requires mcp, type, and name parameters")
		}

		mcpID := args[0].String()
		sampleType := types.SampleItemType(args[1].String())
		name := args[2].String()

		result, err := ctx.ListSamples(mcpID, sampleType, name)
		if err != nil {
			return bridge.JsException(v8ctx, err.Error())
		}

		jsVal, err := bridge.JsValue(v8ctx, result)
		if err != nil {
			return bridge.JsException(v8ctx, err.Error())
		}

		return jsVal
	})
}

// mcpGetSampleMethod implements ctx.MCP.GetSample(mcp, type, name, index)
// Gets a specific sample from an MCP client
func (ctx *Context) mcpGetSampleMethod(iso *v8go.Isolate) *v8go.FunctionTemplate {
	return v8go.NewFunctionTemplate(iso, func(info *v8go.FunctionCallbackInfo) *v8go.Value {
		v8ctx := info.Context()
		args := info.Args()

		if len(args) < 4 {
			return bridge.JsException(v8ctx, "GetSample requires mcp, type, name, and index parameters")
		}

		mcpID := args[0].String()
		sampleType := types.SampleItemType(args[1].String())
		name := args[2].String()

		// Parse index
		indexVal, err := bridge.GoValue(args[3], v8ctx)
		if err != nil {
			return bridge.JsException(v8ctx, "invalid index parameter: "+err.Error())
		}

		var index int
		switch v := indexVal.(type) {
		case int:
			index = v
		case int32:
			index = int(v)
		case int64:
			index = int(v)
		case float64:
			index = int(v)
		default:
			return bridge.JsException(v8ctx, "index must be a number")
		}

		result, err := ctx.GetSample(mcpID, sampleType, name, index)
		if err != nil {
			return bridge.JsException(v8ctx, err.Error())
		}

		jsVal, err := bridge.JsValue(v8ctx, result)
		if err != nil {
			return bridge.JsException(v8ctx, err.Error())
		}

		return jsVal
	})
}

// mcpAllMethod implements ctx.mcp.All(requests)
// Calls tools on multiple MCP servers concurrently and waits for all to complete (like Promise.all)
// Each request should have: { mcp: string, tool: string, arguments?: object }
func (ctx *Context) mcpAllMethod(iso *v8go.Isolate) *v8go.FunctionTemplate {
	return v8go.NewFunctionTemplate(iso, func(info *v8go.FunctionCallbackInfo) *v8go.Value {
		v8ctx := info.Context()
		args := info.Args()

		if len(args) < 1 {
			return bridge.JsException(v8ctx, "All requires requests parameter")
		}

		// Parse requests array
		requests, err := ctx.parseMCPToolRequests(args[0], v8ctx)
		if err != nil {
			return bridge.JsException(v8ctx, err.Error())
		}

		// Execute all requests
		results := ctx.CallToolAll(requests)

		// Convert results to JS value
		jsVal, err := bridge.JsValue(v8ctx, results)
		if err != nil {
			return bridge.JsException(v8ctx, "failed to convert results: "+err.Error())
		}

		return jsVal
	})
}

// mcpAnyMethod implements ctx.mcp.Any(requests)
// Calls tools on multiple MCP servers concurrently and returns when any succeeds (like Promise.any)
// Each request should have: { mcp: string, tool: string, arguments?: object }
func (ctx *Context) mcpAnyMethod(iso *v8go.Isolate) *v8go.FunctionTemplate {
	return v8go.NewFunctionTemplate(iso, func(info *v8go.FunctionCallbackInfo) *v8go.Value {
		v8ctx := info.Context()
		args := info.Args()

		if len(args) < 1 {
			return bridge.JsException(v8ctx, "Any requires requests parameter")
		}

		// Parse requests array
		requests, err := ctx.parseMCPToolRequests(args[0], v8ctx)
		if err != nil {
			return bridge.JsException(v8ctx, err.Error())
		}

		// Execute requests until any succeeds
		results := ctx.CallToolAny(requests)

		// Convert results to JS value
		jsVal, err := bridge.JsValue(v8ctx, results)
		if err != nil {
			return bridge.JsException(v8ctx, "failed to convert results: "+err.Error())
		}

		return jsVal
	})
}

// mcpRaceMethod implements ctx.mcp.Race(requests)
// Calls tools on multiple MCP servers concurrently and returns when any completes (like Promise.race)
// Each request should have: { mcp: string, tool: string, arguments?: object }
func (ctx *Context) mcpRaceMethod(iso *v8go.Isolate) *v8go.FunctionTemplate {
	return v8go.NewFunctionTemplate(iso, func(info *v8go.FunctionCallbackInfo) *v8go.Value {
		v8ctx := info.Context()
		args := info.Args()

		if len(args) < 1 {
			return bridge.JsException(v8ctx, "Race requires requests parameter")
		}

		// Parse requests array
		requests, err := ctx.parseMCPToolRequests(args[0], v8ctx)
		if err != nil {
			return bridge.JsException(v8ctx, err.Error())
		}

		// Execute requests and return first completion
		results := ctx.CallToolRace(requests)

		// Convert results to JS value
		jsVal, err := bridge.JsValue(v8ctx, results)
		if err != nil {
			return bridge.JsException(v8ctx, "failed to convert results: "+err.Error())
		}

		return jsVal
	})
}

// parseMCPToolRequests parses JS requests array into MCPToolRequest slice
func (ctx *Context) parseMCPToolRequests(arg *v8go.Value, v8ctx *v8go.Context) ([]*MCPToolRequest, error) {
	goVal, err := bridge.GoValue(arg, v8ctx)
	if err != nil {
		return nil, fmt.Errorf("invalid requests: %w", err)
	}

	requestsArray, ok := goVal.([]interface{})
	if !ok {
		return nil, fmt.Errorf("requests must be an array")
	}

	requests := make([]*MCPToolRequest, 0, len(requestsArray))
	for _, item := range requestsArray {
		reqMap, ok := item.(map[string]interface{})
		if !ok {
			return nil, fmt.Errorf("each request must be an object")
		}

		// Required: mcp
		mcpID, ok := reqMap["mcp"].(string)
		if !ok || mcpID == "" {
			return nil, fmt.Errorf("request.mcp is required and must be a string")
		}

		// Required: tool
		tool, ok := reqMap["tool"].(string)
		if !ok || tool == "" {
			return nil, fmt.Errorf("request.tool is required and must be a string")
		}

		req := &MCPToolRequest{
			MCP:  mcpID,
			Tool: tool,
		}

		// Optional: arguments
		if args, exists := reqMap["arguments"]; exists && args != nil {
			req.Arguments = args
		}

		requests = append(requests, req)
	}

	return requests, nil
}

// newMCPObject creates a new MCP object with all MCP methods
func (ctx *Context) newMCPObject(iso *v8go.Isolate) *v8go.ObjectTemplate {
	mcpObj := v8go.NewObjectTemplate(iso)

	// Resource operations
	mcpObj.Set("ListResources", ctx.mcpListResourcesMethod(iso))
	mcpObj.Set("ReadResource", ctx.mcpReadResourceMethod(iso))

	// Tool operations
	mcpObj.Set("ListTools", ctx.mcpListToolsMethod(iso))
	mcpObj.Set("CallTool", ctx.mcpCallToolMethod(iso))
	mcpObj.Set("CallTools", ctx.mcpCallToolsMethod(iso))
	mcpObj.Set("CallToolsParallel", ctx.mcpCallToolsParallelMethod(iso))

	// Cross-server parallel tool operations (Promise-like patterns)
	mcpObj.Set("All", ctx.mcpAllMethod(iso))
	mcpObj.Set("Any", ctx.mcpAnyMethod(iso))
	mcpObj.Set("Race", ctx.mcpRaceMethod(iso))

	// Prompt operations
	mcpObj.Set("ListPrompts", ctx.mcpListPromptsMethod(iso))
	mcpObj.Set("GetPrompt", ctx.mcpGetPromptMethod(iso))

	// Sample operations
	mcpObj.Set("ListSamples", ctx.mcpListSamplesMethod(iso))
	mcpObj.Set("GetSample", ctx.mcpGetSampleMethod(iso))

	return mcpObj
}
