package providers

import (
	"fmt"

	"github.com/yaoapp/gou/connector"
	gouAnthropicConn "github.com/yaoapp/gou/connector/anthropic"
	"github.com/yaoapp/yao/agent/context"
	"github.com/yaoapp/yao/agent/llm/providers/anthropic"
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

	case "anthropic":
		// Anthropic Messages API (Claude, Kimi Code, etc.)
		// Check if connector has native Anthropic capabilities
		settings := conn.Setting()
		if caps, ok := settings["capabilities"].(*gouAnthropicConn.Capabilities); ok {
			return anthropic.NewFromAnthropicCaps(conn, caps), nil
		}
		// Fallback: use OpenAI capabilities (converted from connector settings)
		return anthropic.New(conn, options.Capabilities), nil

	default:
		// Default to OpenAI-compatible provider
		return openai.New(conn, options.Capabilities), nil
	}
}

// DetectAPIFormat detects the API format from connector
func DetectAPIFormat(conn connector.Connector) string {
	// Check connector type directly
	if conn.Is(connector.ANTHROPIC) {
		return "anthropic"
	}

	if conn.Is(connector.OPENAI) {
		return "openai"
	}

	// Check connector settings for host URL patterns as fallback
	settings := conn.Setting()
	if settings != nil {
		if host, ok := settings["host"].(string); ok {
			if contains(host, "anthropic.com") || contains(host, "api.kimi.com/coding") {
				return "anthropic"
			}
			if contains(host, "deepseek.com") {
				return "openai"
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
