package pdf

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	goupdf "github.com/yaoapp/gou/pdf"
	"github.com/yaoapp/yao/agent/content/image"
	"github.com/yaoapp/yao/agent/content/types"
	agentContext "github.com/yaoapp/yao/agent/context"
	"github.com/yaoapp/yao/agent/i18n"
	"github.com/yaoapp/yao/agent/output/message"
	searchTypes "github.com/yaoapp/yao/agent/search/types"
	"github.com/yaoapp/yao/attachment"
	kbTypes "github.com/yaoapp/yao/kb/types"
)

// PDF handles PDF content
type PDF struct {
	options *types.Options
}

// New creates a new PDF handler
func New(options *types.Options) *PDF {
	return &PDF{options: options}
}

// Parse parses PDF content by converting to images and processing each page
// Returns multiple ContentPart (one text part per page) combined into a single text part
func (h *PDF) Parse(ctx *agentContext.Context, content agentContext.ContentPart) (agentContext.ContentPart, []*searchTypes.Reference, error) {
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

	// Convert PDF to images and process each page
	return h.asImages(ctx, content)
}

// ParseMulti parses PDF content and returns multiple ContentParts (one per page)
// This is useful when you need separate parts for each page
func (h *PDF) ParseMulti(ctx *agentContext.Context, content agentContext.ContentPart) ([]agentContext.ContentPart, []*searchTypes.Reference, error) {
	if content.File == nil || content.File.URL == "" {
		return nil, nil, fmt.Errorf("file content missing URL")
	}

	url := content.File.URL

	// Check cache first - if cached, return as single text part
	cachedText, found, err := h.readFromCache(ctx, url)
	if err == nil && found {
		return []agentContext.ContentPart{
			{
				Type: agentContext.ContentText,
				Text: cachedText,
			},
		}, nil, nil
	}

	// Convert PDF to images and process each page
	return h.asImagesMulti(ctx, content)
}

// asImages converts PDF to images and processes each page, returning combined result
func (h *PDF) asImages(ctx *agentContext.Context, content agentContext.ContentPart) (agentContext.ContentPart, []*searchTypes.Reference, error) {
	parts, refs, err := h.asImagesMulti(ctx, content)
	if err != nil {
		return content, nil, err
	}

	if len(parts) == 0 {
		return content, nil, fmt.Errorf("no pages extracted from PDF")
	}

	// Check if any parts are text (vision agent was used) or image_url (model supports vision)
	hasTextParts := false
	hasImageParts := false
	for _, part := range parts {
		if part.Type == agentContext.ContentText {
			hasTextParts = true
		} else if part.Type == agentContext.ContentImageURL {
			hasImageParts = true
		}
	}

	// If all parts are image_url (model supports vision), return the first image
	// The caller should use ParseMulti to get all images
	if hasImageParts && !hasTextParts {
		return parts[0], refs, nil
	}

	// Combine all text parts into one
	var combinedText strings.Builder
	pageNum := 0
	for _, part := range parts {
		if part.Type == agentContext.ContentText && part.Text != "" {
			pageNum++
			if pageNum > 1 {
				combinedText.WriteString("\n\n---\n\n") // Page separator
			}
			combinedText.WriteString(fmt.Sprintf("## Page %d\n\n", pageNum))
			combinedText.WriteString(part.Text)
		}
	}

	result := agentContext.ContentPart{
		Type: agentContext.ContentText,
		Text: combinedText.String(),
	}

	// Cache the combined result
	if content.File != nil && content.File.URL != "" && combinedText.Len() > 0 {
		h.saveToCache(ctx, content.File.URL, combinedText.String())
	}

	return result, refs, nil
}

// asImagesMulti converts PDF to images and processes each page separately
func (h *PDF) asImagesMulti(ctx *agentContext.Context, content agentContext.ContentPart) ([]agentContext.ContentPart, []*searchTypes.Reference, error) {
	if content.File == nil || content.File.URL == "" {
		return nil, nil, fmt.Errorf("file content missing URL")
	}

	url := content.File.URL

	// Read PDF file
	pdfData, err := h.readPDF(ctx, url)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to read PDF: %w", err)
	}

	// Create temporary file for PDF
	tempDir := os.TempDir()
	pdfPath := filepath.Join(tempDir, fmt.Sprintf("pdf_%d.pdf", time.Now().UnixNano()))
	if err := os.WriteFile(pdfPath, pdfData, 0644); err != nil {
		return nil, nil, fmt.Errorf("failed to write temp PDF: %w", err)
	}
	defer os.Remove(pdfPath)

	// Get PDF processor with global config
	processor, err := h.getPDFProcessor()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create PDF processor: %w", err)
	}

	// Create output directory for images
	imagesDir := filepath.Join(tempDir, fmt.Sprintf("pdf_images_%d", time.Now().UnixNano()))
	if err := os.MkdirAll(imagesDir, 0755); err != nil {
		return nil, nil, fmt.Errorf("failed to create images directory: %w", err)
	}
	defer os.RemoveAll(imagesDir)

	// Convert PDF to images
	convertConfig := goupdf.ConvertConfig{
		OutputDir:    imagesDir,
		OutputPrefix: "page",
		Format:       "png",
		DPI:          150,
		Quality:      90,
		PageRange:    "all",
	}

	imageFiles, err := processor.Convert(ctx.Context, pdfPath, convertConfig)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to convert PDF to images: %w", err)
	}

	if len(imageFiles) == 0 {
		return nil, nil, fmt.Errorf("no pages extracted from PDF")
	}

	// Process each image using the image handler (with SilentLoading to suppress image loading messages)
	imageOptions := *h.options // Copy options
	imageOptions.SilentLoading = true
	imageHandler := image.New(&imageOptions)
	var parts []agentContext.ContentPart
	var allRefs []*searchTypes.Reference

	for i, imageFile := range imageFiles {
		// Send loading message for this page
		loadingMsg := fmt.Sprintf(i18n.T(ctx.Locale, "content.pdf.analyzing_page"), i+1, len(imageFiles))
		loadingID := h.sendLoading(ctx, loadingMsg)

		// Read image file
		imageData, err := os.ReadFile(imageFile)
		if err != nil {
			h.sendLoadingDone(ctx, loadingID)
			continue
		}

		// Convert to base64 data URI
		base64Data := image.EncodeToBase64DataURI(imageData, "image/png")

		// Create image content part
		imagePart := agentContext.ContentPart{
			Type: agentContext.ContentImageURL,
			ImageURL: &agentContext.ImageURL{
				URL:    base64Data,
				Detail: agentContext.DetailAuto,
			},
		}

		// Parse image using image handler
		parsedPart, refs, err := imageHandler.Parse(ctx, imagePart)

		// Mark loading as done
		h.sendLoadingDone(ctx, loadingID)

		if err != nil {
			// If parsing fails, skip this page
			continue
		}

		parts = append(parts, parsedPart)
		if refs != nil {
			allRefs = append(allRefs, refs...)
		}
	}

	if len(parts) == 0 {
		return nil, nil, fmt.Errorf("failed to process any PDF pages")
	}

	return parts, allRefs, nil
}

// readPDF reads PDF content from various sources
func (h *PDF) readPDF(ctx *agentContext.Context, url string) ([]byte, error) {
	if strings.HasPrefix(url, "__") {
		// Uploader wrapper format: __uploader://fileid
		return h.readFromUploader(ctx, url)
	}

	if strings.HasPrefix(url, "http://") || strings.HasPrefix(url, "https://") {
		return nil, fmt.Errorf("HTTP URL fetch not implemented yet: %s", url)
	}

	// Try to read as local file path
	if _, err := os.Stat(url); err == nil {
		return os.ReadFile(url)
	}

	return nil, fmt.Errorf("unsupported PDF source: %s", url)
}

// readFromUploader reads PDF content from file uploader
func (h *PDF) readFromUploader(ctx *agentContext.Context, wrapper string) ([]byte, error) {
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

// readFromCache reads cached text content for a PDF
func (h *PDF) readFromCache(ctx *agentContext.Context, url string) (string, bool, error) {
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
func (h *PDF) saveToCache(ctx *agentContext.Context, url string, text string) error {
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

// getPDFProcessor creates a PDF processor using global KB config
func (h *PDF) getPDFProcessor() (*goupdf.PDF, error) {
	globalPDF := kbTypes.GetGlobalPDF()

	opts := goupdf.Options{
		ConvertTool: goupdf.ToolPdftoppm, // default
		ToolPath:    "",
	}

	if globalPDF != nil {
		if globalPDF.ConvertTool != "" {
			switch globalPDF.ConvertTool {
			case "pdftoppm":
				opts.ConvertTool = goupdf.ToolPdftoppm
			case "mutool":
				opts.ConvertTool = goupdf.ToolMutool
			case "imagemagick", "convert":
				opts.ConvertTool = goupdf.ToolImageMagick
			}
		}
		if globalPDF.ToolPath != "" {
			opts.ToolPath = globalPDF.ToolPath
		}
	}

	return goupdf.New(opts), nil
}

// sendLoading sends a loading message and returns the message ID
func (h *PDF) sendLoading(ctx *agentContext.Context, msg string) string {
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
func (h *PDF) sendLoadingDone(ctx *agentContext.Context, loadingID string) {
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
