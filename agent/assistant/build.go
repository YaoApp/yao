package assistant

import (
	"fmt"

	"github.com/yaoapp/gou/json"
	"github.com/yaoapp/yao/agent/context"
)

// BuildRequest build the LLM request
func (ast *Assistant) BuildRequest(ctx *context.Context, messages []context.Message, createResponse *context.HookCreateResponse) ([]context.Message, *context.CompletionOptions, error) {
	// Build completion options from createResponse and ctx (includes MCP tools)
	options, mcpSamplesPrompt, err := ast.buildCompletionOptions(ctx, createResponse)
	if err != nil {
		return nil, nil, err
	}

	// Build final messages with proper priority (includes MCP samples if available)
	finalMessages, err := ast.buildMessages(ctx, messages, createResponse, mcpSamplesPrompt)
	if err != nil {
		return nil, nil, err
	}

	return finalMessages, options, nil
}

// buildMessages builds the final message list with proper priority
// Priority: Prompts > MCP Samples > createResponse.Messages > input messages
// If createResponse is nil or has no messages, use input messages
func (ast *Assistant) buildMessages(ctx *context.Context, messages []context.Message, createResponse *context.HookCreateResponse, mcpSamplesPrompt string) ([]context.Message, error) {
	var finalMessages []context.Message

	// If createResponse is nil or has no messages, use input messages
	if createResponse == nil || len(createResponse.Messages) == 0 {
		finalMessages = messages
	} else {
		// createResponse.Messages takes priority over input messages
		finalMessages = createResponse.Messages
	}

	// Add MCP samples prompt as a system message (if available)
	if mcpSamplesPrompt != "" {
		mcpSamplesMsg := context.Message{
			Role:    context.RoleSystem,
			Content: mcpSamplesPrompt,
		}
		// Prepend MCP samples before other messages
		finalMessages = append([]context.Message{mcpSamplesMsg}, finalMessages...)
	}

	// ⚠️ Just for testing, will remove later
	// If we have prompts, prepend them to the beginning
	if len(ast.Prompts) > 0 {
		promptMessages := make([]context.Message, 0, len(ast.Prompts))
		for _, prompt := range ast.Prompts {
			msg := context.Message{
				Role:    context.MessageRole(prompt.Role),
				Content: prompt.Content,
			}
			// Add name if provided
			if prompt.Name != "" {
				name := prompt.Name
				msg.Name = &name
			}
			promptMessages = append(promptMessages, msg)
		}
		// Prepend prompt messages to the beginning
		finalMessages = append(promptMessages, finalMessages...)
	}

	return finalMessages, nil
}

// buildCompletionOptions builds completion options from multiple sources
// Priority (lowest to highest, later overrides earlier): ast > ctx > createResponse
// The priority means: if createResponse has a value, use it; else use ctx; else use ast
// Returns (options, mcpSamplesPrompt, error)
func (ast *Assistant) buildCompletionOptions(ctx *context.Context, createResponse *context.HookCreateResponse) (*context.CompletionOptions, string, error) {
	options := &context.CompletionOptions{}

	// Layer 1 (base): Apply ast - Assistant configuration
	if err := ast.applyAssistantOptions(options); err != nil {
		return nil, "", err
	}

	// Layer 2 (middle): Apply ctx - Context configuration (overrides ast)
	ast.applyContextOptions(options, ctx)

	// Layer 3 (highest): Apply createResponse - Hook configuration (overrides all)
	if createResponse != nil {
		ast.applyCreateResponseOptions(options, createResponse)
	}

	// Add MCP tools if configured and get samples prompt
	mcpSamplesPrompt, err := ast.applyMCPTools(ctx, options, createResponse)
	if err != nil {
		return nil, "", fmt.Errorf("failed to apply MCP tools: %w", err)
	}

	return options, mcpSamplesPrompt, nil
}

// applyAssistantOptions applies options from ast.Options to CompletionOptions
// ast.Options can contain any OpenAI API parameters (temperature, top_p, stop, etc.)
// Returns error if any option validation fails (e.g., invalid JSON Schema)
func (ast *Assistant) applyAssistantOptions(options *context.CompletionOptions) error {
	if ast.Options == nil {
		return nil
	}

	// Temperature
	if v, ok := ast.Options["temperature"].(float64); ok {
		options.Temperature = &v
	}

	// MaxTokens
	if v, ok := ast.Options["max_tokens"].(float64); ok {
		intVal := int(v)
		options.MaxTokens = &intVal
	} else if v, ok := ast.Options["max_tokens"].(int); ok {
		options.MaxTokens = &v
	}

	// MaxCompletionTokens
	if v, ok := ast.Options["max_completion_tokens"].(float64); ok {
		intVal := int(v)
		options.MaxCompletionTokens = &intVal
	} else if v, ok := ast.Options["max_completion_tokens"].(int); ok {
		options.MaxCompletionTokens = &v
	}

	// TopP
	if v, ok := ast.Options["top_p"].(float64); ok {
		options.TopP = &v
	}

	// N (number of choices)
	if v, ok := ast.Options["n"].(float64); ok {
		intVal := int(v)
		options.N = &intVal
	} else if v, ok := ast.Options["n"].(int); ok {
		options.N = &v
	}

	// Stop sequences (can be string or []string)
	if v, ok := ast.Options["stop"]; ok {
		options.Stop = v
	}

	// PresencePenalty
	if v, ok := ast.Options["presence_penalty"].(float64); ok {
		options.PresencePenalty = &v
	}

	// FrequencyPenalty
	if v, ok := ast.Options["frequency_penalty"].(float64); ok {
		options.FrequencyPenalty = &v
	}

	// LogitBias
	if v, ok := ast.Options["logit_bias"].(map[string]interface{}); ok {
		logitBias := make(map[string]float64)
		for key, val := range v {
			if fval, ok := val.(float64); ok {
				logitBias[key] = fval
			}
		}
		if len(logitBias) > 0 {
			options.LogitBias = logitBias
		}
	}

	// User
	if v, ok := ast.Options["user"].(string); ok {
		options.User = v
	}

	// ResponseFormat
	// @todo: Assistant should have a default response format
	if v, ok := ast.Options["response_format"]; ok {
		// Try to convert to *context.ResponseFormat
		if rf, ok := v.(*context.ResponseFormat); ok {
			// Validate JSONSchema if present - reject if invalid
			if rf.JSONSchema != nil && rf.JSONSchema.Schema != nil {
				if err := json.ValidateSchema(rf.JSONSchema.Schema); err != nil {
					return fmt.Errorf("invalid JSON Schema in response_format: %w", err)
				}
			}
			options.ResponseFormat = rf
		} else if rfMap, ok := v.(map[string]interface{}); ok {
			// Handle legacy map[string]interface{} format
			// Try to parse into ResponseFormat struct
			rf := &context.ResponseFormat{}

			// Parse type
			if typeStr, ok := rfMap["type"].(string); ok {
				rf.Type = context.ResponseFormatType(typeStr)
			}

			// Parse json_schema if present
			if jsonSchemaMap, ok := rfMap["json_schema"].(map[string]interface{}); ok {
				jsonSchema := &context.JSONSchema{}

				if name, ok := jsonSchemaMap["name"].(string); ok {
					jsonSchema.Name = name
				}
				if desc, ok := jsonSchemaMap["description"].(string); ok {
					jsonSchema.Description = desc
				}
				if schema, ok := jsonSchemaMap["schema"]; ok {
					// Validate schema format - reject if invalid
					if err := json.ValidateSchema(schema); err != nil {
						return fmt.Errorf("invalid JSON Schema in response_format: %w", err)
					}
					jsonSchema.Schema = schema
				}
				if strict, ok := jsonSchemaMap["strict"].(bool); ok {
					jsonSchema.Strict = &strict
				}

				rf.JSONSchema = jsonSchema
			}

			options.ResponseFormat = rf
		}
	}

	// Seed
	if v, ok := ast.Options["seed"].(float64); ok {
		intVal := int(v)
		options.Seed = &intVal
	} else if v, ok := ast.Options["seed"].(int); ok {
		options.Seed = &v
	}

	// Tools
	if v, ok := ast.Options["tools"].([]interface{}); ok {
		tools := make([]map[string]interface{}, 0, len(v))
		for _, tool := range v {
			if toolMap, ok := tool.(map[string]interface{}); ok {
				tools = append(tools, toolMap)
			}
		}
		if len(tools) > 0 {
			options.Tools = tools
		}
	}

	// ToolChoice
	if v, ok := ast.Options["tool_choice"]; ok {
		options.ToolChoice = v
	}

	// Stream
	if v, ok := ast.Options["stream"].(bool); ok {
		options.Stream = &v
	}

	return nil
}

// applyContextOptions applies options from ctx to CompletionOptions
// ctx provides Route and Metadata for CUI context
func (ast *Assistant) applyContextOptions(options *context.CompletionOptions, ctx *context.Context) {
	// Set Route and Metadata from ctx
	options.Route = ctx.Route
	options.Metadata = ctx.Metadata

	// Set Uses configurations (assistant.Uses has priority over global settings)
	// These can be overridden by createResponse
	options.Uses = ast.getUses()
}

// applyCreateResponseOptions applies options from createResponse to CompletionOptions
// createResponse takes highest priority and overrides any previous settings
func (ast *Assistant) applyCreateResponseOptions(options *context.CompletionOptions, createResponse *context.HookCreateResponse) {
	// Audio configuration
	if createResponse.Audio != nil {
		options.Audio = createResponse.Audio
	}

	// Temperature
	if createResponse.Temperature != nil {
		options.Temperature = createResponse.Temperature
	}

	// MaxTokens
	if createResponse.MaxTokens != nil {
		options.MaxTokens = createResponse.MaxTokens
	}

	// MaxCompletionTokens
	if createResponse.MaxCompletionTokens != nil {
		options.MaxCompletionTokens = createResponse.MaxCompletionTokens
	}

	// Route
	if createResponse.Route != "" {
		options.Route = createResponse.Route
	}

	// Metadata (merge with existing)
	if createResponse.Metadata != nil {
		if options.Metadata == nil {
			options.Metadata = createResponse.Metadata
		} else {
			// Merge: createResponse.Metadata overrides existing
			for key, value := range createResponse.Metadata {
				options.Metadata[key] = value
			}
		}
	}
}

// getUses get the Uses configuration with priority: assistant.Uses > global settings
func (ast *Assistant) getUses() *context.Uses {
	// Priority 1: Assistant-specific Uses configuration
	if ast.Uses != nil {
		// Create a merged Uses by starting with global, then override with assistant-specific
		merged := &context.Uses{}

		// Start with global settings
		if globalUses != nil {
			merged.Vision = globalUses.Vision
			merged.Audio = globalUses.Audio
			merged.Search = globalUses.Search
			merged.Fetch = globalUses.Fetch
		}

		// Override with assistant-specific settings (only if not empty)
		if ast.Uses.Vision != "" {
			merged.Vision = ast.Uses.Vision
		}
		if ast.Uses.Audio != "" {
			merged.Audio = ast.Uses.Audio
		}
		if ast.Uses.Search != "" {
			merged.Search = ast.Uses.Search
		}
		if ast.Uses.Fetch != "" {
			merged.Fetch = ast.Uses.Fetch
		}

		return merged
	}

	// Priority 2: Global settings only
	return globalUses
}

// applyMCPTools adds MCP tools to completion options and returns samples prompt
// Returns (samplesPrompt, error)
func (ast *Assistant) applyMCPTools(ctx *context.Context, options *context.CompletionOptions, createResponse *context.HookCreateResponse) (string, error) {

	// Priority 1: Check if hook provides MCP servers
	if createResponse != nil && len(createResponse.MCPServers) > 0 {
		return ast.buildAndApplyMCPTools(ctx, options, createResponse)
	}

	// Priority 2: Check if assistant has MCP config
	if ast.MCP != nil && len(ast.MCP.Servers) > 0 {
		return ast.buildAndApplyMCPTools(ctx, options, nil)
	}

	// No MCP config
	return "", nil
}

// buildAndApplyMCPTools builds MCP tools and applies them to options
func (ast *Assistant) buildAndApplyMCPTools(ctx *context.Context, options *context.CompletionOptions, createResponse *context.HookCreateResponse) (string, error) {
	// Build MCP tools and get samples prompt
	mcpTools, samplesPrompt, err := ast.buildMCPTools(ctx, createResponse)
	if err != nil {
		return "", fmt.Errorf("failed to build MCP tools: %w", err)
	}

	// Convert mcpTools to map format for CompletionOptions.Tools
	if len(mcpTools) > 0 {
		toolMaps := make([]map[string]interface{}, len(mcpTools))
		for i, tool := range mcpTools {
			toolMaps[i] = map[string]interface{}{
				"type": "function",
				"function": map[string]interface{}{
					"name":        tool.Name,
					"description": tool.Description,
					"parameters":  tool.Parameters,
				},
			}
		}

		// Add MCP tools to existing tools (append to preserve existing tools)
		if options.Tools == nil {
			options.Tools = toolMaps
		} else {
			options.Tools = append(options.Tools, toolMaps...)
		}
	}

	return samplesPrompt, nil
}
