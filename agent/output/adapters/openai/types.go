package openai

import "github.com/yaoapp/yao/agent/output/message"

// ConverterFunc converts a message to OpenAI format chunks
type ConverterFunc func(msg *message.Message, config *AdapterConfig) ([]interface{}, error)

// LinkTransformer transforms a URL to a secure link (with OTP, short URL, etc.)
// Returns the transformed link or error
type LinkTransformer func(url string, msgType string, msgID string) (string, error)

// AdapterConfig holds the configuration for OpenAI adapter
type AdapterConfig struct {
	// BaseURL is the base URL for generating view links
	// Example: "https://api.example.com"
	BaseURL string

	// LinkTemplates defines the Markdown template for each message type
	// %s will be replaced with the link
	// Example: "🖼️ [View Image](%s)"
	LinkTemplates map[string]string

	// LinkTransformer transforms URLs to secure links with OTP
	// If nil, URLs are used as-is
	LinkTransformer LinkTransformer

	// Model name to include in OpenAI responses
	Model string

	// Capabilities holds the model capabilities
	// Used to determine how to convert certain message types (e.g., stream_start)
	Capabilities *ModelCapabilities

	// Locale for internationalization (e.g., "en-US", "zh-CN")
	Locale string
}

// ModelCapabilities is a simplified version of openai.Capabilities
// We use a local type to avoid circular dependencies
type ModelCapabilities struct {
	Reasoning *bool // Supports reasoning/thinking mode (o1, DeepSeek R1)
}

// DefaultLinkTemplates provides default Markdown templates for non-text message types
var DefaultLinkTemplates = map[string]string{
	"image":  "![%s](%s)",           // Markdown image: ![alt](url) - displays inline
	"audio":  "🔊 [Play Audio](%s)",  // Link (audio can't display inline in Markdown)
	"video":  "🎬 [Watch Video](%s)", // Link (video can't display inline in Markdown)
	"file":   "📎 [Download File](%s)",
	"page":   "📄 [Open Page](%s)",
	"table":  "📊 [View Table](%s)",
	"chart":  "📈 [View Chart](%s)",
	"list":   "📋 [View List](%s)",
	"form":   "📝 [Fill Form](%s)",
	"button": "🔘 [%s](%s)", // Special: button needs two params (text, link)
}

// DefaultAdapterConfig returns a default adapter configuration
func DefaultAdapterConfig() *AdapterConfig {
	return &AdapterConfig{
		BaseURL:         "", // Will be set from environment or context
		LinkTemplates:   copyLinkTemplates(DefaultLinkTemplates),
		LinkTransformer: nil, // No transformation by default
		Model:           "yao-agent",
	}
}

// copyLinkTemplates creates a copy of link templates
func copyLinkTemplates(templates map[string]string) map[string]string {
	copy := make(map[string]string, len(templates))
	for k, v := range templates {
		copy[k] = v
	}
	return copy
}
