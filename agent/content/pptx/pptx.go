package pptx

import (
	"fmt"
	"os"
	"strings"

	"github.com/yaoapp/gou/office"
	"github.com/yaoapp/yao/agent/content/types"
	agentContext "github.com/yaoapp/yao/agent/context"
	searchTypes "github.com/yaoapp/yao/agent/search/types"
	"github.com/yaoapp/yao/attachment"
)

// Pptx handles PPTX content
type Pptx struct {
	options *types.Options
}

// New creates a new PPTX handler
func New(options *types.Options) *Pptx {
	return &Pptx{options: options}
}

// Parse parses PPTX content and returns text
func (h *Pptx) Parse(ctx *agentContext.Context, content agentContext.ContentPart) (agentContext.ContentPart, []*searchTypes.Reference, error) {
	if content.File == nil || content.File.URL == "" {
		return content, nil, fmt.Errorf("file content missing URL")
	}

	url := content.File.URL

	// Check cache first
	cachedText, found, err := h.readFromCache(ctx, url)
	if err == nil && found {
		return agentContext.ContentPart{
			Type: agentContext.ContentText,
			Text: cachedText,
		}, nil, nil
	}

	// Read PPTX file
	data, err := h.readFile(ctx, url)
	if err != nil {
		return content, nil, fmt.Errorf("failed to read PPTX: %w", err)
	}

	// Parse PPTX using gou/office
	parser := office.NewParser()
	result, err := parser.Parse(data)
	if err != nil {
		return content, nil, fmt.Errorf("failed to parse PPTX: %w", err)
	}

	text := result.Markdown
	if text == "" {
		return content, nil, fmt.Errorf("no text content extracted from PPTX")
	}

	// Cache the result
	if err := h.saveToCache(ctx, url, text); err != nil {
		// Log warning but don't fail
		fmt.Printf("Warning: failed to cache PPTX text: %v\n", err)
	}

	return agentContext.ContentPart{
		Type: agentContext.ContentText,
		Text: text,
	}, nil, nil
}

// readFile reads PPTX content from various sources
func (h *Pptx) readFile(ctx *agentContext.Context, url string) ([]byte, error) {
	if strings.HasPrefix(url, "__") {
		return h.readFromUploader(ctx, url)
	}

	if strings.HasPrefix(url, "http://") || strings.HasPrefix(url, "https://") {
		return nil, fmt.Errorf("HTTP URL fetch not implemented yet: %s", url)
	}

	// Try to read as local file path
	if _, err := os.Stat(url); err == nil {
		return os.ReadFile(url)
	}

	return nil, fmt.Errorf("unsupported PPTX source: %s", url)
}

// readFromUploader reads PPTX content from file uploader
func (h *Pptx) readFromUploader(ctx *agentContext.Context, wrapper string) ([]byte, error) {
	uploaderName, fileID, ok := attachment.Parse(wrapper)
	if !ok {
		return nil, fmt.Errorf("invalid uploader wrapper format: %s", wrapper)
	}

	manager, exists := attachment.Managers[uploaderName]
	if !exists {
		return nil, fmt.Errorf("uploader '%s' not found", uploaderName)
	}

	data, err := manager.Read(ctx.Context, fileID)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	return data, nil
}

// readFromCache reads cached text content for a PPTX
func (h *Pptx) readFromCache(ctx *agentContext.Context, url string) (string, bool, error) {
	uploaderName, fileID, isWrapper := attachment.Parse(url)
	if !isWrapper {
		return "", false, nil
	}

	manager, exists := attachment.Managers[uploaderName]
	if !exists {
		return "", false, nil
	}

	text, err := manager.GetText(ctx.Context, fileID, false)
	if err == nil && text != "" {
		return text, true, nil
	}

	return "", false, nil
}

// saveToCache saves processed text to cache
func (h *Pptx) saveToCache(ctx *agentContext.Context, url string, text string) error {
	uploaderName, fileID, isWrapper := attachment.Parse(url)
	if !isWrapper {
		return nil
	}

	manager, exists := attachment.Managers[uploaderName]
	if !exists {
		return nil
	}

	return manager.SaveText(ctx.Context, fileID, text)
}
