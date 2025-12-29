package image

import (
	"encoding/base64"
	"fmt"
	"strings"

	"github.com/yaoapp/yao/agent/content/tools"
	"github.com/yaoapp/yao/agent/content/types"
	agentContext "github.com/yaoapp/yao/agent/context"
	"github.com/yaoapp/yao/agent/i18n"
	"github.com/yaoapp/yao/agent/output/message"
	searchTypes "github.com/yaoapp/yao/agent/search/types"
	"github.com/yaoapp/yao/attachment"
)

// Image handles image content
type Image struct {
	options *types.Options
}

// New creates a new image handler
func New(options *types.Options) *Image {
	return &Image{options: options}
}

// Parse parses image content
// Logic:
// 1. Check model capabilities first
// 2. If forceUses is true and uses.Vision is specified -> use vision tool regardless of model capability
// 3. If model supports vision -> pass through or convert to base64 format
// 4. If model doesn't support vision -> use vision agent/MCP to extract text
func (h *Image) Parse(ctx *agentContext.Context, content agentContext.ContentPart) (agentContext.ContentPart, []*searchTypes.Reference, error) {
	if content.ImageURL == nil || content.ImageURL.URL == "" {
		return content, nil, fmt.Errorf("image_url content missing URL")
	}

	// Check model capabilities first
	supportsVision, visionFormat := agentContext.GetVisionSupport(h.options.Capabilities)

	// Check if we should force using Uses tools
	forceUses := h.options.CompletionOptions != nil && h.options.CompletionOptions.ForceUses

	// If forceUses is true and uses.Vision is specified, use vision tool regardless of model capability
	if forceUses && h.options.CompletionOptions != nil && h.options.CompletionOptions.Uses != nil && h.options.CompletionOptions.Uses.Vision != "" {
		// Check cache first before calling agent
		cachedText, found, err := h.readFromCache(ctx, content.ImageURL.URL)
		if err == nil && found {
			return agentContext.ContentPart{
				Type: agentContext.ContentText,
				Text: cachedText,
			}, nil, nil
		}
		return h.agent(ctx, content)
	}

	// If model supports vision
	if supportsVision {
		url := content.ImageURL.URL
		// If it's already a data URI (base64), pass through directly
		if strings.HasPrefix(url, "data:") {
			return content, nil, nil
		}
		// Convert to base64 format
		return h.base64(ctx, content, visionFormat)
	}

	// Model doesn't support vision - check cache first, then use vision agent/MCP
	// Try to get cached text (from attachment's content_preview)
	cachedText, found, err := h.readFromCache(ctx, content.ImageURL.URL)
	if err == nil && found {
		// Cache hit! Return as text content
		return agentContext.ContentPart{
			Type: agentContext.ContentText,
			Text: cachedText,
		}, nil, nil
	}

	// No cache, try to use vision agent/MCP
	if h.options.CompletionOptions != nil && h.options.CompletionOptions.Uses != nil && h.options.CompletionOptions.Uses.Vision != "" {
		return h.agent(ctx, content)
	}

	// No vision support and no vision tool specified, return error
	return content, nil, fmt.Errorf("model doesn't support vision and no vision tool specified in uses.Vision")
}

// base64 encodes image content to base64 (for vision support)
func (h *Image) base64(ctx *agentContext.Context, content agentContext.ContentPart, format agentContext.VisionFormat) (agentContext.ContentPart, []*searchTypes.Reference, error) {
	if content.ImageURL == nil || content.ImageURL.URL == "" {
		return content, nil, fmt.Errorf("image_url content missing URL")
	}

	url := content.ImageURL.URL

	// Read image data from source
	data, contentType, err := h.read(ctx, url)
	if err != nil {
		return content, nil, fmt.Errorf("failed to read image: %w", err)
	}

	// Encode to base64 data URI
	base64Data := EncodeToBase64DataURI(data, contentType)

	// Return as image_url ContentPart
	return agentContext.ContentPart{
		Type: agentContext.ContentImageURL,
		ImageURL: &agentContext.ImageURL{
			URL:    base64Data,
			Detail: content.ImageURL.Detail,
		},
	}, nil, nil
}

// read reads image content from various sources
func (h *Image) read(ctx *agentContext.Context, url string) ([]byte, string, error) {
	// Determine source type and read accordingly
	if strings.HasPrefix(url, "data:") {
		// Data URI format: data:image/png;base64,xxxxx
		return h.readFromDataURI(url)
	}

	if strings.HasPrefix(url, "__") {
		// Uploader wrapper format: __uploader://fileid
		return h.readFromUploader(ctx, url)
	}

	if strings.HasPrefix(url, "http://") || strings.HasPrefix(url, "https://") {
		// HTTP URL - for now return error, can be implemented later
		return nil, "", fmt.Errorf("HTTP URL fetch not implemented yet: %s", url)
	}

	// Unknown source
	return nil, "", fmt.Errorf("unsupported image source: %s", url)
}

// readFromDataURI reads image content from a data URI
func (h *Image) readFromDataURI(dataURI string) ([]byte, string, error) {
	// Parse data URI: data:image/png;base64,xxxxx
	if !strings.HasPrefix(dataURI, "data:") {
		return nil, "", fmt.Errorf("invalid data URI format")
	}

	// Find the comma separator
	commaIndex := strings.Index(dataURI, ",")
	if commaIndex == -1 {
		return nil, "", fmt.Errorf("invalid data URI: missing comma separator")
	}

	// Extract metadata part (e.g., "image/png;base64")
	metadata := dataURI[5:commaIndex] // Skip "data:"
	base64Data := dataURI[commaIndex+1:]

	// Parse content type
	contentType := "image/png" // default
	if strings.Contains(metadata, ";") {
		parts := strings.Split(metadata, ";")
		if len(parts) > 0 && parts[0] != "" {
			contentType = parts[0]
		}
	} else if metadata != "" && metadata != "base64" {
		contentType = metadata
	}

	// Decode base64 data
	data, err := base64.StdEncoding.DecodeString(base64Data)
	if err != nil {
		return nil, "", fmt.Errorf("failed to decode base64 data: %w", err)
	}

	return data, contentType, nil
}

// readFromUploader reads image content from file uploader __uploader://fileid
func (h *Image) readFromUploader(ctx *agentContext.Context, wrapper string) ([]byte, string, error) {
	// Parse wrapper to get uploader name and file ID
	uploaderName, fileID, ok := attachment.Parse(wrapper)
	if !ok {
		return nil, "", fmt.Errorf("invalid uploader wrapper format: %s", wrapper)
	}

	// Get attachment manager
	manager, exists := attachment.Managers[uploaderName]
	if !exists {
		return nil, "", fmt.Errorf("uploader '%s' not found", uploaderName)
	}

	// Get file info
	file, err := manager.Info(ctx.Context, fileID)
	if err != nil {
		return nil, "", fmt.Errorf("failed to get file info: %w", err)
	}

	// Read file content
	data, err := manager.Read(ctx.Context, fileID)
	if err != nil {
		return nil, "", fmt.Errorf("failed to read file: %w", err)
	}

	return data, file.ContentType, nil
}

// readFromCache reads cached text content for an image
func (h *Image) readFromCache(ctx *agentContext.Context, url string) (string, bool, error) {
	// Parse URL to check if it's an uploader wrapper
	uploaderName, fileID, isWrapper := attachment.Parse(url)
	if !isWrapper {
		return "", false, nil // Not an uploader wrapper, no cache
	}

	// Try attachment manager's content_preview (cross-call cache)
	manager, exists := attachment.Managers[uploaderName]
	if !exists {
		return "", false, nil
	}

	// GetText with fullContent=false to get preview (default)
	text, err := manager.GetText(ctx.Context, fileID, false)
	if err == nil && text != "" {
		return text, true, nil
	}

	// No cache found
	return "", false, nil
}

// saveToCache saves processed text to cache
func (h *Image) saveToCache(ctx *agentContext.Context, url string, text string) error {
	// Parse URL to get uploader name and file ID
	uploaderName, fileID, isWrapper := attachment.Parse(url)
	if !isWrapper {
		return nil // Not an uploader wrapper, nothing to cache
	}

	// Save to attachment manager for future calls
	manager, exists := attachment.Managers[uploaderName]
	if !exists {
		return nil
	}

	return manager.SaveText(ctx.Context, fileID, text)
}

// agent calls image agent to parse image content
// Note: Cache check is done in Parse() before calling this method
func (h *Image) agent(ctx *agentContext.Context, content agentContext.ContentPart) (agentContext.ContentPart, []*searchTypes.Reference, error) {
	if content.ImageURL == nil || content.ImageURL.URL == "" {
		return content, nil, fmt.Errorf("image_url content missing URL")
	}

	url := content.ImageURL.URL

	// Get vision tool from options
	visionTool := ""
	if h.options.CompletionOptions != nil && h.options.CompletionOptions.Uses != nil {
		visionTool = h.options.CompletionOptions.Uses.Vision
	}

	if visionTool == "" {
		return content, nil, fmt.Errorf("no vision tool specified in uses.Vision")
	}

	// Parse vision tool format
	// Format can be:
	// - "agent_id" (call agent)
	// - "mcp:server_id" (call MCP tool)
	var text string
	var err error
	if strings.HasPrefix(visionTool, "mcp:") {
		// MCP tool
		serverID := strings.TrimPrefix(visionTool, "mcp:")
		text, err = h.callMCPVisionTool(ctx, serverID, content)
	} else {
		// Agent call
		text, err = h.callVisionAgent(ctx, visionTool, content)
	}

	if err != nil {
		return content, nil, fmt.Errorf("failed to process image with vision tool: %w", err)
	}

	// Cache the result
	if cacheErr := h.saveToCache(ctx, url, text); cacheErr != nil {
		// Log error but don't fail the request
		fmt.Printf("Warning: failed to cache processed text: %v\n", cacheErr)
	}

	// Return as text content
	return agentContext.ContentPart{
		Type: agentContext.ContentText,
		Text: text,
	}, nil, nil
}

// callVisionAgent calls a vision agent to describe the image
func (h *Image) callVisionAgent(ctx *agentContext.Context, agentID string, content agentContext.ContentPart) (string, error) {
	// Read image data and convert to base64
	data, contentType, err := h.read(ctx, content.ImageURL.URL)
	if err != nil {
		return "", fmt.Errorf("failed to read image: %w", err)
	}

	base64Data := EncodeToBase64DataURI(data, contentType)

	// Prepare message with image
	message := agentContext.Message{
		Role: agentContext.RoleUser,
		Content: []agentContext.ContentPart{
			{
				Type: agentContext.ContentText,
				Text: "Please analyze this image.",
			},
			{
				Type: agentContext.ContentImageURL,
				ImageURL: &agentContext.ImageURL{
					URL:    base64Data,
					Detail: agentContext.DetailAuto,
				},
			},
		},
	}

	// Send loading message
	loadingID := h.sendLoading(ctx, i18n.T(ctx.Locale, "content.image.analyzing"))

	// Call agent using the tools package
	result, err := tools.CallAgent(ctx, agentID, message)

	// Send done message
	h.sendLoadingDone(ctx, loadingID)

	return result, err
}

// callMCPVisionTool calls an MCP vision tool to describe the image
func (h *Image) callMCPVisionTool(ctx *agentContext.Context, serverID string, content agentContext.ContentPart) (string, error) {
	// Read image data and convert to base64
	data, contentType, err := h.read(ctx, content.ImageURL.URL)
	if err != nil {
		return "", fmt.Errorf("failed to read image: %w", err)
	}

	base64Data := EncodeToBase64DataURI(data, contentType)

	// Prepare arguments for MCP tool
	arguments := map[string]interface{}{
		"image":        base64Data,
		"content_type": contentType,
	}

	// Send loading message
	loadingID := h.sendLoading(ctx, i18n.T(ctx.Locale, "content.image.analyzing"))

	// Call MCP tool (typically "describe_image" or similar)
	result, err := tools.CallMCPTool(ctx, serverID, "describe_image", arguments)

	// Send done message
	h.sendLoadingDone(ctx, loadingID)

	return result, err
}

// sendLoading sends a loading message and returns the message ID
// Returns empty string if SilentLoading is enabled
func (h *Image) sendLoading(ctx *agentContext.Context, msg string) string {
	// Skip loading message if SilentLoading is enabled (called from parent handler like PDF)
	if h.options != nil && h.options.SilentLoading {
		return ""
	}

	loadingMsg := &message.Message{
		Type: message.TypeLoading,
		Props: map[string]interface{}{
			"message": msg,
		},
	}

	msgID, err := ctx.SendStream(loadingMsg)
	if err != nil {
		return ""
	}
	return msgID
}

// sendLoadingDone marks the loading message as done
func (h *Image) sendLoadingDone(ctx *agentContext.Context, loadingID string) {
	if loadingID == "" {
		return
	}

	doneMsg := &message.Message{
		MessageID:   loadingID,
		Delta:       true,
		DeltaAction: message.DeltaReplace,
		Type:        message.TypeLoading,
		Props: map[string]interface{}{
			"done": true,
		},
	}

	ctx.Send(doneMsg)
}

// EncodeToBase64DataURI encodes data to base64 with data URI prefix
func EncodeToBase64DataURI(data []byte, contentType string) string {
	if contentType == "" {
		contentType = "image/png" // default for images
	}

	encoded := base64.StdEncoding.EncodeToString(data)
	return fmt.Sprintf("data:%s;base64,%s", contentType, encoded)
}
