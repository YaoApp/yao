package assistant

import (
	"fmt"

	"github.com/spf13/cast"
	"github.com/yaoapp/gou/json"
	"github.com/yaoapp/yao/agent/context"
	store "github.com/yaoapp/yao/agent/store/types"
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

	// Build and prepend system prompts (global + assistant prompts)
	promptMessages := ast.buildSystemPrompts(ctx, createResponse)
	if len(promptMessages) > 0 {
		finalMessages = append(promptMessages, finalMessages...)
	}

	return finalMessages, nil
}

// buildSystemPrompts builds system prompt messages from global prompts and assistant prompts
// Order: Global prompts (if not disabled) -> Assistant prompts (or preset)
// Variables are parsed with context information
//
// Priority for prompt preset selection:
//  1. createResponse.PromptPreset (highest)
//  2. ctx.Metadata["__prompt_preset"]
//  3. ast.Prompts (default)
//
// Priority for disable global prompts:
//  1. createResponse.DisableGlobalPrompts (highest)
//  2. ctx.Metadata["__disable_global_prompts"]
//  3. ast.DisableGlobalPrompts (default)
func (ast *Assistant) buildSystemPrompts(ctx *context.Context, createResponse *context.HookCreateResponse) []context.Message {
	// Build context variables from ctx and ast
	ctxVars := ast.buildContextVariables(ctx)

	// Determine if global prompts should be disabled
	disableGlobal := ast.shouldDisableGlobalPrompts(ctx, createResponse)

	// Get assistant prompts (default or preset)
	assistantPrompts := ast.getAssistantPrompts(ctx, createResponse)

	var allPrompts []store.Prompt

	// 1. Add global prompts (if not disabled)
	if !disableGlobal && len(globalPrompts) > 0 {
		// Parse global prompts with context variables
		parsedGlobal := store.Prompts(globalPrompts).Parse(ctxVars)
		allPrompts = append(allPrompts, parsedGlobal...)
	}

	// 2. Add assistant prompts (default or preset)
	if len(assistantPrompts) > 0 {
		// Parse assistant prompts with context variables
		parsedAssistant := store.Prompts(assistantPrompts).Parse(ctxVars)
		allPrompts = append(allPrompts, parsedAssistant...)
	}

	// Convert to context.Message slice
	if len(allPrompts) == 0 {
		return nil
	}

	messages := make([]context.Message, 0, len(allPrompts))
	for _, prompt := range allPrompts {
		msg := context.Message{
			Role:    context.MessageRole(prompt.Role),
			Content: prompt.Content,
		}
		if prompt.Name != "" {
			name := prompt.Name
			msg.Name = &name
		}
		messages = append(messages, msg)
	}

	return messages
}

// shouldDisableGlobalPrompts determines if global prompts should be disabled
// Priority: createResponse > ctx.Metadata > ast.DisableGlobalPrompts
func (ast *Assistant) shouldDisableGlobalPrompts(ctx *context.Context, createResponse *context.HookCreateResponse) bool {
	// Priority 1: Hook response (highest)
	if createResponse != nil && createResponse.DisableGlobalPrompts != nil {
		return *createResponse.DisableGlobalPrompts
	}

	// Priority 2: ctx.Metadata["__disable_global_prompts"]
	if ctx != nil && ctx.Metadata != nil {
		if disable, ok := ctx.Metadata["__disable_global_prompts"].(bool); ok {
			return disable
		}
	}

	// Priority 3: Assistant configuration (default)
	return ast.DisableGlobalPrompts
}

// getAssistantPrompts returns the assistant prompts based on preset selection
// Priority: createResponse.PromptPreset > ctx.Metadata["__prompt_preset"] > ast.Prompts
func (ast *Assistant) getAssistantPrompts(ctx *context.Context, createResponse *context.HookCreateResponse) []store.Prompt {
	// Get preset key
	presetKey := ast.getPromptPresetKey(ctx, createResponse)

	// If preset key is specified and exists, use it
	if presetKey != "" && ast.PromptPresets != nil {
		if presets, ok := ast.PromptPresets[presetKey]; ok && len(presets) > 0 {
			return presets
		}
	}

	// Fallback to default prompts
	return ast.Prompts
}

// getPromptPresetKey returns the prompt preset key
// Priority: createResponse.PromptPreset > ctx.Metadata["__prompt_preset"]
func (ast *Assistant) getPromptPresetKey(ctx *context.Context, createResponse *context.HookCreateResponse) string {
	// Priority 1: Hook response (highest)
	if createResponse != nil && createResponse.PromptPreset != "" {
		return createResponse.PromptPreset
	}

	// Priority 2: ctx.Metadata["__prompt_preset"]
	if ctx != nil && ctx.Metadata != nil {
		if preset, ok := ctx.Metadata["__prompt_preset"].(string); ok && preset != "" {
			return preset
		}
	}

	// No preset specified
	return ""
}

// buildContextVariables extracts context variables from Context and Assistant for prompt parsing
func (ast *Assistant) buildContextVariables(ctx *context.Context) map[string]string {
	vars := make(map[string]string)

	// Get locale from ctx (default to empty)
	locale := ""
	if ctx != nil && ctx.Locale != "" {
		locale = ctx.Locale
	}

	// Assistant info (with locale support)
	if ast != nil {
		if ast.ID != "" {
			vars["ASSISTANT_ID"] = ast.ID
		}
		// Use localized name and description
		name := ast.GetName(locale)
		if name != "" {
			vars["ASSISTANT_NAME"] = name
		}
		description := ast.GetDescription(locale)
		if description != "" {
			vars["ASSISTANT_DESCRIPTION"] = description
		}
		if ast.Type != "" {
			vars["ASSISTANT_TYPE"] = ast.Type
		}
	}

	if ctx == nil {
		return vars
	}

	// Basic context info
	if ctx.ChatID != "" {
		vars["CHAT_ID"] = ctx.ChatID
	}
	if ctx.Locale != "" {
		vars["LOCALE"] = ctx.Locale
	}
	if ctx.Theme != "" {
		vars["THEME"] = ctx.Theme
	}
	if ctx.Route != "" {
		vars["ROUTE"] = ctx.Route
	}
	if ctx.Referer != "" {
		vars["REFERER"] = ctx.Referer
	}

	// Client info (only non-sensitive fields)
	if ctx.Client.Type != "" {
		vars["CLIENT_TYPE"] = ctx.Client.Type
	}

	// Authorized info (only internal IDs, no PII)
	// Note: USER_SUBJECT and CLIENT_IP are excluded for privacy/GDPR compliance
	if ctx.Authorized != nil {
		if ctx.Authorized.UserID != "" {
			vars["USER_ID"] = ctx.Authorized.UserID
		}
		if ctx.Authorized.TeamID != "" {
			vars["TEAM_ID"] = ctx.Authorized.TeamID
		}
		if ctx.Authorized.TenantID != "" {
			vars["TENANT_ID"] = ctx.Authorized.TenantID
		}
	}

	// Metadata - custom variables from ctx.Metadata
	// All metadata keys are exposed as $CTX.{KEY}
	// Supports string, int, uint, float, bool types
	if ctx.Metadata != nil {
		for key, value := range ctx.Metadata {
			if value == nil {
				continue
			}
			strVal := cast.ToString(value)
			if strVal != "" {
				vars[key] = strVal
			}
		}
	}

	return vars
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

	// Uses configuration (merge with existing)
	// createResponse.Uses has highest priority and overrides existing Uses
	if createResponse.Uses != nil {
		if options.Uses == nil {
			options.Uses = createResponse.Uses
		} else {
			// Merge: createResponse.Uses overrides existing (only non-empty fields)
			if createResponse.Uses.Vision != "" {
				options.Uses.Vision = createResponse.Uses.Vision
			}
			if createResponse.Uses.Audio != "" {
				options.Uses.Audio = createResponse.Uses.Audio
			}
			if createResponse.Uses.Search != "" {
				options.Uses.Search = createResponse.Uses.Search
			}
			if createResponse.Uses.Fetch != "" {
				options.Uses.Fetch = createResponse.Uses.Fetch
			}
		}
	}

	// ForceUses configuration
	// If hook specifies ForceUses, it takes priority
	if createResponse.ForceUses != nil {
		options.ForceUses = *createResponse.ForceUses
	}
}

// getUses get the Uses configuration with priority: assistant.Uses > global settings
// Note: createResponse.Uses (applied in applyCreateResponseOptions) has even higher priority
// getUses returns the Uses config for this assistant
// Note: The config is already merged with global config during loading (loadMap)
func (ast *Assistant) getUses() *context.Uses {
	return ast.Uses
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
