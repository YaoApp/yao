package content

import (
	"fmt"
	"strings"

	"github.com/yaoapp/gou/connector/openai"
	agentContext "github.com/yaoapp/yao/agent/context"
)

// ImageHandler handles image content
type ImageHandler struct{}

// CanHandle checks if this handler can handle the content type
func (h *ImageHandler) CanHandle(contentType string, fileType FileType) bool {
	return fileType == FileTypeImage || strings.HasPrefix(contentType, "image/")
}

// Handle processes image content
// Logic:
// 1. If forceUses is true and uses.Vision is specified -> use vision tool regardless of model capability
// 2. If model supports vision and forceUses is false -> convert to base64 or image_url format
// 3. If model doesn't support vision -> use agent/MCP specified in uses.Vision
func (h *ImageHandler) Handle(ctx *agentContext.Context, info *Info, capabilities *openai.Capabilities, uses *agentContext.Uses, forceUses bool) (*Result, error) {
	if len(info.Data) == 0 {
		return nil, fmt.Errorf("no image data to process")
	}

	if capabilities == nil {
		return nil, fmt.Errorf("no capabilities provided")
	}

	// Check if model supports vision
	supportsVision, visionFormat := agentContext.GetVisionSupport(capabilities)

	// If forceUses is true and uses.Vision is specified, use vision tool regardless of model capability
	if forceUses && uses != nil && uses.Vision != "" {
		text, err := h.handleWithVisionAgent(ctx, info, uses.Vision)
		if err != nil {
			return nil, fmt.Errorf("failed to handle image with vision agent/MCP (forced): %w", err)
		}
		return &Result{
			Text: text,
		}, nil
	}

	if supportsVision {
		// Model supports vision - return as image_url ContentPart
		contentPart, err := h.handleWithVisionModel(ctx, info, visionFormat)
		if err != nil {
			return nil, fmt.Errorf("failed to handle image with vision model: %w", err)
		}
		return &Result{
			ContentPart: contentPart,
		}, nil
	}

	// Model doesn't support vision - use vision agent/MCP
	visionTool := ""
	if uses != nil && uses.Vision != "" {
		visionTool = uses.Vision
	}

	if visionTool == "" {
		return nil, fmt.Errorf("model doesn't support vision and no vision tool specified in uses.Vision")
	}

	// Call vision agent/MCP to extract text
	text, err := h.handleWithVisionAgent(ctx, info, visionTool)
	if err != nil {
		return nil, fmt.Errorf("failed to handle image with vision agent/MCP: %w", err)
	}

	return &Result{
		Text: text,
	}, nil
}

// handleWithVisionModel processes image using model's vision capability
func (h *ImageHandler) handleWithVisionModel(ctx *agentContext.Context, info *Info, format agentContext.VisionFormat) (*agentContext.ContentPart, error) {
	// Encode image to base64
	base64Data := encodeImageBase64(info.Data, info.ContentType)

	// Format according to model's vision format
	switch format {
	case agentContext.VisionFormatOpenAI:
		// OpenAI format: image_url with data URI
		return &agentContext.ContentPart{
			Type: agentContext.ContentImageURL,
			ImageURL: &agentContext.ImageURL{
				URL:    base64Data,
				Detail: agentContext.DetailAuto,
			},
		}, nil

	case agentContext.VisionFormatClaude:
		// Claude format: also uses image_url but may have different handling
		// For now, use the same format as OpenAI
		return &agentContext.ContentPart{
			Type: agentContext.ContentImageURL,
			ImageURL: &agentContext.ImageURL{
				URL:    base64Data,
				Detail: agentContext.DetailAuto,
			},
		}, nil

	case agentContext.VisionFormatDefault, "":
		// Default format (when Vision: true) - use OpenAI format
		return &agentContext.ContentPart{
			Type: agentContext.ContentImageURL,
			ImageURL: &agentContext.ImageURL{
				URL:    base64Data,
				Detail: agentContext.DetailAuto,
			},
		}, nil

	default:
		return nil, fmt.Errorf("unsupported vision format: %s", format)
	}
}

// handleWithVisionAgent processes image using vision agent or MCP
func (h *ImageHandler) handleWithVisionAgent(ctx *agentContext.Context, info *Info, visionTool string) (string, error) {
	// Parse vision tool format
	// Format can be:
	// - "agent_id" (call agent)
	// - "mcp:server_id" (call MCP tool)
	if strings.HasPrefix(visionTool, "mcp:") {
		// MCP tool
		serverID := strings.TrimPrefix(visionTool, "mcp:")
		return h.callMCPVisionTool(ctx, serverID, info)
	}

	// Agent call
	return h.callVisionAgent(ctx, visionTool, info)
}

// callVisionAgent calls a vision agent to describe the image
func (h *ImageHandler) callVisionAgent(ctx *agentContext.Context, agentID string, info *Info) (string, error) {
	// Prepare message with image
	base64Data := EncodeToBase64DataURI(info.Data, info.ContentType)

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

	// Call agent with file metadata in context
	// File info (filename, file_id, etc.) will be available in ctx.Metadata["file_info"]
	// This allows hooks (especially Next hook) to access and format file information
	return CallAgentWithFileInfo(ctx, agentID, message, info)
}

// callMCPVisionTool calls an MCP vision tool to describe the image
func (h *ImageHandler) callMCPVisionTool(ctx *agentContext.Context, serverID string, info *Info) (string, error) {
	// Prepare base64 encoded image for MCP tool
	base64Data := EncodeToBase64DataURI(info.Data, info.ContentType)

	// Prepare arguments for MCP tool
	arguments := map[string]interface{}{
		"image":        base64Data,
		"content_type": info.ContentType,
	}

	// Call MCP tool (typically "describe_image" or similar)
	return CallMCPTool(ctx, serverID, "describe_image", arguments)
}

// encodeImageBase64 encodes image data to base64 with data URI prefix
func encodeImageBase64(data []byte, contentType string) string {
	// Use the common function
	if contentType == "" {
		contentType = "image/png" // default for images
	}
	return EncodeToBase64DataURI(data, contentType)
}
