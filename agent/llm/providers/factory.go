package providers

import (
	"fmt"

	"github.com/yaoapp/gou/connector"
	"github.com/yaoapp/yao/agent/context"
	"github.com/yaoapp/yao/agent/llm/providers/legacy"
	"github.com/yaoapp/yao/agent/llm/providers/openai"
	"github.com/yaoapp/yao/agent/llm/providers/reasoning"
)

// LLM interface (copied to avoid import cycle)
type LLM interface {
	Stream(ctx *context.Context, messages []context.Message, options *context.CompletionOptions, handler context.StreamFunc) (*context.CompletionResponse, error)
	Post(ctx *context.Context, messages []context.Message, options *context.CompletionOptions) (*context.CompletionResponse, error)
}

// SelectProvider select the appropriate provider based on connector and capabilities
func SelectProvider(conn connector.Connector, options *context.CompletionOptions) (LLM, error) {

	if options == nil {
		return nil, fmt.Errorf("options are required")
	}

	if options.Capabilities == nil {
		return nil, fmt.Errorf("capabilities are required")
	}

	capabilities := options.Capabilities

	// return openai.New(conn, capabilities), nil

	// Priority 1: Reasoning models (special response format)
	if capabilities.Reasoning != nil && *capabilities.Reasoning {
		return reasoning.New(conn, capabilities), nil
	}

	// Priority 2: Check if model supports native tool calls
	if capabilities.ToolCalls != nil && *capabilities.ToolCalls {
		// Use OpenAI-compatible provider (supports tools, vision, streaming)
		return openai.New(conn, capabilities), nil
	}

	// Priority 3: Legacy models (no native tool support)
	// Will use prompt engineering for tool calls
	return legacy.New(conn, capabilities), nil
}

// DetectProvider detect provider type from connector
func DetectProvider(conn connector.Connector) string {
	// TODO: Implement provider detection
	// - Check connector type (Is(connector.OPENAI))
	// - Check connector settings
	// - Determine provider type (openai, claude, deepseek, etc.)

	if conn.Is(connector.OPENAI) {
		return "openai"
	}

	// Default to OpenAI-compatible
	return "openai"
}
