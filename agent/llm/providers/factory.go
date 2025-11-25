package providers

import (
	"fmt"

	"github.com/yaoapp/gou/connector"
	"github.com/yaoapp/yao/agent/context"
	"github.com/yaoapp/yao/agent/llm/providers/openai"
	"github.com/yaoapp/yao/agent/output/message"
)

// LLM interface (copied to avoid import cycle)
type LLM interface {
	Stream(ctx *context.Context, messages []context.Message, options *context.CompletionOptions, handler message.StreamFunc) (*context.CompletionResponse, error)
	Post(ctx *context.Context, messages []context.Message, options *context.CompletionOptions) (*context.CompletionResponse, error)
}

// SelectProvider selects the appropriate provider based on API format and capabilities
// The new architecture uses capability adapters to handle different model features
func SelectProvider(conn connector.Connector, options *context.CompletionOptions) (LLM, error) {
	if options == nil {
		return nil, fmt.Errorf("options are required")
	}

	if options.Capabilities == nil {
		return nil, fmt.Errorf("capabilities are required")
	}

	// Detect API format
	apiFormat := DetectAPIFormat(conn)

	// Select provider based on API format
	switch apiFormat {
	case "openai":
		// OpenAI-compatible API
		// Capability adapters will handle:
		// - Tool calling (native or prompt engineering)
		// - Vision (native or removal)
		// - Audio (native or removal)
		// - Reasoning (o1, GPT-4o thinking, etc.)
		return openai.New(conn, options.Capabilities), nil

	case "claude":
		// TODO: Implement Claude provider
		// For now, use OpenAI provider (may have compatibility issues)
		return openai.New(conn, options.Capabilities), nil

	default:
		// Default to OpenAI-compatible provider
		return openai.New(conn, options.Capabilities), nil
	}
}

// DetectAPIFormat detects the API format from connector
func DetectAPIFormat(conn connector.Connector) string {
	// Check connector type
	if conn.Is(connector.OPENAI) {
		return "openai"
	}

	// Check connector settings for host URL
	settings := conn.Setting()
	if settings != nil {
		if host, ok := settings["host"].(string); ok {
			// Detect by host URL patterns
			if contains(host, "anthropic.com") || contains(host, "claude") {
				return "claude"
			}
			if contains(host, "deepseek.com") {
				return "openai" // DeepSeek uses OpenAI-compatible API
			}
		}
	}

	// Default to OpenAI-compatible
	return "openai"
}

// contains checks if a string contains a substring (case-insensitive helper)
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) &&
		(s[:len(substr)] == substr || s[len(s)-len(substr):] == substr ||
			findSubstring(s, substr)))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
