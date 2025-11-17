package assistant

import (
	"fmt"

	"github.com/yaoapp/gou/connector"
	"github.com/yaoapp/yao/agent/context"
	"github.com/yaoapp/yao/agent/llm"
	"github.com/yaoapp/yao/agent/output"
	"github.com/yaoapp/yao/utils/jsonschema"
)

// Stream stream the agent
// handler is optional, if not provided, a default handler will be used
func (ast *Assistant) Stream(ctx *context.Context, inputMessages []context.Message, handler ...context.StreamFunc) (*context.Response, error) {

	var err error

	// Initialize stack and auto-handle completion/failure/restore
	_, traceID, done := context.EnterStack(ctx, ast.ID, ctx.Referer)
	defer done()

	_ = traceID // traceID is available for trace logging

	// Full input messages with chat history
	fullMessages, err := ast.WithHistory(ctx, inputMessages)
	if err != nil {
		return nil, err
	}

	// Request Create hook ( Optional )
	var createResponse *context.HookCreateResponse
	if ast.Script != nil {
		var err error
		createResponse, err = ast.Script.Create(ctx, fullMessages)
		if err != nil {
			return nil, err
		}
	}

	var completionOptions *context.CompletionOptions // default is nil

	// LLM Call Stream ( Optional )
	var completionMessages []context.Message
	var completionResponse *context.CompletionResponse
	if ast.Prompts != nil || ast.MCP != nil {
		// Build the LLM request first
		completionMessages, completionOptions, err = ast.BuildRequest(ctx, inputMessages, createResponse)
		if err != nil {
			return nil, err
		}

		// Get connector object and capabilities
		conn, capabilities, err := ast.GetConnector(ctx)
		if err != nil {
			return nil, err
		}

		// Set capabilities in options if not already set
		if completionOptions.Capabilities == nil && capabilities != nil {
			completionOptions.Capabilities = capabilities
		}

		// Create LLM instance with connector and options
		llmInstance, err := llm.New(conn, completionOptions)
		if err != nil {
			return nil, err
		}

		// Use provided handler or default handler
		streamHandler := llm.DefaultStreamHandler(ctx)
		if len(handler) > 0 && handler[0] != nil {
			streamHandler = handler[0]
		}

		// Call the LLM Completion Stream
		completionResponse, err = llmInstance.Stream(ctx, completionMessages, completionOptions, streamHandler)
		if err != nil {
			return nil, err
		}

	}

	// Request MCP hook ( Optional )
	var mcpResponse *context.ResponseHookMCP
	if ast.MCP != nil {
		_ = mcpResponse // mcpResponse is available for further processing

		// MCP Execution Loop
	}

	// Request Done hook ( Optional )
	var doneResponse *context.ResponseHookDone
	if ast.Script != nil {
		var err error
		doneResponse, err = ast.Script.Done(ctx, fullMessages, completionResponse, mcpResponse)
		if err != nil {
			return nil, err
		}
	}

	_ = doneResponse // doneResponse is available for further processing

	// Close the output writer to send [DONE] marker and flush data
	if err := output.Close(ctx); err != nil {
		// Log error but don't fail the request
		fmt.Printf("Warning: Failed to close output writer: %v\n", err)
	}

	// Flush any remaining data to the client
	if err := output.Flush(ctx); err != nil {
		fmt.Printf("Warning: Failed to flush output: %v\n", err)
	}

	return &context.Response{Create: createResponse, Done: doneResponse, Completion: completionResponse}, nil
}

// GetConnector get the connector object, capabilities, and error with priority: createResponse > ctx > ast
// Note: createResponse.Connector is already applied to ctx.Connector by applyContextAdjustments in create.go
// Returns: (connector, capabilities, error)
func (ast *Assistant) GetConnector(ctx *context.Context) (connector.Connector, *context.ModelCapabilities, error) {
	// Determine connector ID with priority
	connectorID := ast.Connector
	if ctx.Connector != "" {
		connectorID = ctx.Connector
	}

	// If empty, return error
	if connectorID == "" {
		return nil, nil, fmt.Errorf("connector not specified")
	}

	// Load gou connector
	conn, err := connector.Select(connectorID)
	if err != nil {
		return nil, nil, err
	}

	// Get connector capabilities from settings
	capabilities := ast.getConnectorCapabilities(connectorID)

	return conn, capabilities, nil
}

// getConnectorCapabilities get the capabilities of a connector from settings
func (ast *Assistant) getConnectorCapabilities(connectorID string) *context.ModelCapabilities {
	// Initialize with default capabilities (all disabled)
	falseVal := false
	capabilities := &context.ModelCapabilities{
		Vision:    falseVal,
		ToolCalls: &falseVal,
		Audio:     &falseVal,
		Reasoning: &falseVal,
		Streaming: &falseVal,
	}

	// Get model capabilities from global configuration
	modelCaps, exists := modelCapabilities[connectorID]
	if !exists {
		// Return default capabilities if model not found in configuration
		return capabilities
	}

	// Update capabilities based on model configuration
	// Vision can be bool or string (VisionFormat)
	if modelCaps.Vision != nil {
		capabilities.Vision = modelCaps.Vision
	}

	// Handle both Tools (deprecated) and ToolCalls
	if modelCaps.ToolCalls || modelCaps.Tools {
		v := true
		capabilities.ToolCalls = &v
	}

	if modelCaps.Audio {
		v := true
		capabilities.Audio = &v
	}

	if modelCaps.Reasoning {
		v := true
		capabilities.Reasoning = &v
	}

	if modelCaps.Streaming {
		v := true
		capabilities.Streaming = &v
	}

	if modelCaps.JSON {
		v := true
		capabilities.JSON = &v
	}

	if modelCaps.Multimodal {
		v := true
		capabilities.Multimodal = &v
	}

	return capabilities
}

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
// Priority: createResponse.Messages > input messages
// If createResponse is nil or has no messages, use input messages
func (ast *Assistant) buildMessages(ctx *context.Context, messages []context.Message, createResponse *context.HookCreateResponse) ([]context.Message, error) {
	// If createResponse is nil or has no messages, return input messages as-is
	if createResponse == nil || len(createResponse.Messages) == 0 {
		return messages, nil
	}

	// createResponse.Messages takes highest priority
	// Return them directly as they override everything
	return createResponse.Messages, nil
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
				if _, err := jsonschema.New(rf.JSONSchema.Schema); err != nil {
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
					if _, err := jsonschema.New(schema); err != nil {
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

// WithHistory with the history messages
func (ast *Assistant) WithHistory(ctx *context.Context, messages []context.Message) ([]context.Message, error) {
	return messages, nil
}
