package content

import (
	"github.com/yaoapp/gou/connector/openai"
	agentContext "github.com/yaoapp/yao/agent/context"
)

// Handler defines the interface for handling different content types
// Converts content (images, documents, etc.) to text or standard formats
type Handler interface {
	// CanHandle checks if this handler can handle the given content type
	CanHandle(contentType string, fileType FileType) bool

	// Handle converts the content and returns processed result
	// ctx: agent context (passed from Vision function)
	// capabilities: model capabilities (for vision/audio support detection)
	// uses: configuration for external tools (agents/MCP servers)
	// forceUses: if true, force using Uses tools even when model has native capabilities
	Handle(ctx *agentContext.Context, info *Info, capabilities *openai.Capabilities, uses *agentContext.Uses, forceUses bool) (*Result, error)
}

// Fetcher defines the interface for fetching content from different sources
type Fetcher interface {
	// Fetch retrieves content from a URL or file ID
	Fetch(ctx *agentContext.Context, source Source, url string) (*Info, error)
}
