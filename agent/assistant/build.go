package assistant

import (
	"fmt"

	"github.com/yaoapp/gou/json"
	"github.com/yaoapp/yao/agent/context"
)

// BuildRequest build the LLM request
func (ast *Assistant) BuildRequest(ctx *context.Context, messages []context.Message, createResponse *context.HookCreateResponse) ([]context.Message, *context.CompletionOptions, error) {
	// Build final messages with proper priority
	finalMessages, err := ast.buildMessages(ctx, messages, createResponse)
	if err != nil {
		return nil, nil, err
	}

	// Build completion options from createResponse and ctx
	options, err := ast.buildCompletionOptions(ctx, createResponse)
	if err != nil {
		return nil, nil, err
	}

	return finalMessages, options, nil
}

// buildMessages builds the final message list with proper priority
// Priority: Prompts > createResponse.Messages > input messages
// If createResponse is nil or has no messages, use input messages
func (ast *Assistant) buildMessages(ctx *context.Context, messages []context.Message, createResponse *context.HookCreateResponse) ([]context.Message, error) {
	var finalMessages []context.Message

	// If createResponse is nil or has no messages, use input messages
	if createResponse == nil || len(createResponse.Messages) == 0 {
		finalMessages = messages
	} else {
		// createResponse.Messages takes priority over input messages
		finalMessages = createResponse.Messages
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
func (ast *Assistant) buildCompletionOptions(ctx *context.Context, createResponse *context.HookCreateResponse) (*context.CompletionOptions, error) {
	options := &context.CompletionOptions{}

	// Layer 1 (base): Apply ast - Assistant configuration
	if err := ast.applyAssistantOptions(options); err != nil {
		return nil, err
	}

	// Layer 2 (middle): Apply ctx - Context configuration (overrides ast)
	ast.applyContextOptions(options, ctx)

	// Layer 3 (highest): Apply createResponse - Hook configuration (overrides all)
	if createResponse != nil {
		ast.applyCreateResponseOptions(options, createResponse)
	}

	return options, nil
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
