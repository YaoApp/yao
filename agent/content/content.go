package content

import (
	"fmt"
	"strings"

	"github.com/yaoapp/gou/connector/openai"
	agentContext "github.com/yaoapp/yao/agent/context"
	"github.com/yaoapp/yao/attachment"
)

// Vision transforms extended content types to LLM-compatible formats
// This is the main entry point for content preprocessing before sending to LLM
//
// IMPORTANT: This function is called BEFORE sending messages to LLM (in agent.executeLLMStream)
// It must convert all extended content types to standard LLM-compatible types.
//
// Input Content Types (Extended):
//   - type="text"        -> Pass through (already standard)
//   - type="image_url"   -> Process based on model capability (may need base64 conversion or vision tool)
//   - type="input_audio" -> Process based on model capability (may need transcription)
//   - type="file"        -> Convert to text or image_url (MUST be converted)
//   - type="data"        -> Convert to text (MUST be converted)
//
// Output Content Types (LLM-compatible only):
//   - type="text"        -> Text content
//   - type="image_url"   -> Image (only if model supports vision)
//   - type="input_audio" -> Audio (only if model supports audio)
//
// Processing Logic:
// 1. For images (image_url):
//   - If model supports vision -> keep as image_url (may convert URL to base64)
//   - If model doesn't support -> use vision agent/MCP to extract text -> convert to type="text"
//
// 2. For audio (input_audio):
//   - If model supports audio -> keep as input_audio
//   - If model doesn't support -> use audio agent/MCP to transcribe -> convert to type="text"
//
// 3. For files (type="file"):
//   - Parse uploader wrapper (__uploader://fileid) or fetch HTTP URL
//   - Detect file type (PDF, Word, Excel, Image, etc.)
//   - Process based on file type:
//   - Images: same as image processing above
//   - PDF: use vision tool if available, otherwise extract text -> type="text"
//   - Word/Excel/PPT/CSV: extract text -> type="text"
//   - MUST convert to type="text" or type="image_url" (if image and model supports)
//
// 4. For data (type="data"):
//   - Fetch data from sources (models, KB, MCP resources, etc.)
//   - Format as readable text
//   - MUST convert to type="text"
//
// Return: Messages with only standard LLM-compatible content types (text, image_url, input_audio)
func Vision(ctx *agentContext.Context, capabilities *openai.Capabilities, messages []agentContext.Message, uses *agentContext.Uses, forceUses ...bool) ([]agentContext.Message, error) {
	// Determine if we should force using Uses tools even when model has native capabilities
	shouldForceUses := false
	if len(forceUses) > 0 {
		shouldForceUses = forceUses[0]
	}
	// Initialize handlers and fetcher
	registry := NewRegistry()
	fetcher := NewFetcher()

	// Cache for processed files (uploader wrapper -> extracted text)
	// Ensures each file is only processed once
	processedFiles := make(map[string]string)

	// Process each message
	processedMessages := make([]agentContext.Message, 0, len(messages))

	for _, msg := range messages {
		processedMsg, err := processMessage(ctx, &msg, capabilities, uses, shouldForceUses, registry, fetcher, processedFiles)
		if err != nil {
			// Log error but continue processing other messages
			// TODO: Add proper logging
			fmt.Printf("Warning: failed to process message: %v\n", err)
			processedMessages = append(processedMessages, msg) // Keep original on error
			continue
		}
		processedMessages = append(processedMessages, processedMsg)
	}

	return processedMessages, nil
}

// processMessage processes a single message and its content parts
func processMessage(
	ctx *agentContext.Context,
	msg *agentContext.Message,
	capabilities *openai.Capabilities,
	uses *agentContext.Uses,
	forceUses bool,
	registry *Registry,
	fetcher Fetcher,
	processedFiles map[string]string,
) (agentContext.Message, error) {
	// If content is simple string, no processing needed
	if _, ok := msg.GetContentAsString(); ok {
		return *msg, nil
	}

	// Get content parts
	parts, ok := msg.GetContentAsParts()
	if !ok {
		return *msg, nil
	}

	// Note: File information will be collected and stored in Space by CallAgentWithFileInfo
	// when calling vision agents, using the agent ID as namespace prefix

	// Process each content part
	processedParts := make([]agentContext.ContentPart, 0, len(parts))
	for _, part := range parts {
		processedPart, err := processContentPart(ctx, &part, capabilities, uses, forceUses, registry, fetcher, processedFiles)
		if err != nil {
			// Log error and handle gracefully
			fmt.Printf("Warning: failed to process content part: %v\n", err)

			// For image_url that failed to process, convert to text description
			// This prevents sending unsupported multimodal content to non-vision models
			if part.Type == agentContext.ContentImageURL {
				processedParts = append(processedParts, agentContext.ContentPart{
					Type: agentContext.ContentText,
					Text: fmt.Sprintf("[Image processing failed: %s]", part.ImageURL.URL),
				})
			} else {
				// For other types, keep original
				processedParts = append(processedParts, part)
			}
			continue
		}

		// If handling returned text, convert to text part
		if processedPart.Text != "" {
			processedParts = append(processedParts, agentContext.ContentPart{
				Type: agentContext.ContentText,
				Text: processedPart.Text,
			})
		} else if processedPart.ContentPart != nil {
			// Use the processed content part (e.g., base64 image)
			processedParts = append(processedParts, *processedPart.ContentPart)
		} else {
			// Keep original if no handling result
			processedParts = append(processedParts, part)
		}
	}

	// Return new message with processed content
	return agentContext.Message{
		Role:       msg.Role,
		Content:    processedParts,
		Name:       msg.Name,
		ToolCallID: msg.ToolCallID,
		ToolCalls:  msg.ToolCalls,
		Refusal:    msg.Refusal,
	}, nil
}

// processContentPart processes a single content part
// IMPORTANT: Must convert extended types (file, data) to standard types (text, image_url, input_audio)
func processContentPart(
	ctx *agentContext.Context,
	part *agentContext.ContentPart,
	capabilities *openai.Capabilities,
	uses *agentContext.Uses,
	forceUses bool,
	registry *Registry,
	fetcher Fetcher,
	processedFiles map[string]string,
) (*Result, error) {
	// 1. Handle standard types - pass through
	switch part.Type {
	case agentContext.ContentText:
		// Text is already standard, pass through
		return &Result{
			ContentPart: part,
		}, nil

	case agentContext.ContentImageURL:
		// Image URL - check if it needs processing
		return processImageURLContent(ctx, part, capabilities, uses, forceUses, registry, fetcher, processedFiles)

	case agentContext.ContentInputAudio:
		// Audio - check if it needs processing
		return processAudioContent(ctx, part, capabilities, uses, registry, fetcher, processedFiles)
	}

	// 2. Handle extended types - MUST convert to standard types
	switch part.Type {
	case agentContext.ContentFile:
		return processFileContent(ctx, part, capabilities, uses, forceUses, registry, fetcher, processedFiles)

	case agentContext.ContentData:
		return processDataContent(ctx, part)

	default:
		// Unknown type, return error
		return nil, fmt.Errorf("unsupported content type: %s", part.Type)
	}
}

// processFileContent processes file content with caching
func processFileContent(
	ctx *agentContext.Context,
	part *agentContext.ContentPart,
	capabilities *openai.Capabilities,
	uses *agentContext.Uses,
	forceUses bool,
	registry *Registry,
	fetcher Fetcher,
	processedFiles map[string]string,
) (*Result, error) {
	if part.File == nil || part.File.URL == "" {
		return nil, fmt.Errorf("file content part missing URL")
	}

	url := part.File.URL

	// Step 1: Try to get cached text (three-tier cache)
	cachedText, found, err := tryGetCachedText(ctx, url, processedFiles)
	if err != nil {
		return nil, fmt.Errorf("failed to check cache: %w", err)
	}
	if found {
		// Cache hit! Return as text
		return &Result{
			Text: cachedText,
		}, nil
	}

	// Step 2: No cache, need to process the file
	// Determine content source
	source, sourceURL, err := determineContentSource(part)
	if err != nil {
		return nil, fmt.Errorf("failed to determine content source: %w", err)
	}

	// Fetch content
	info, err := fetcher.Fetch(ctx, source, sourceURL)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch content: %w", err)
	}

	// Set filename from part if not already set
	if info.Filename == "" && part.File.Filename != "" {
		info.Filename = part.File.Filename
	}

	// Detect file type if not already set
	if info.FileType == FileTypeUnknown {
		info.FileType = DetectFileType(info.ContentType, part.File.Filename)
	}

	// Process with appropriate handler
	result, err := registry.Handle(ctx, info, capabilities, uses, forceUses)
	if err != nil {
		return nil, fmt.Errorf("failed to handle content: %w", err)
	}

	// Step 3: Cache the result if it's text
	if result.Text != "" {
		if cacheErr := cacheProcessedText(ctx, url, result.Text, processedFiles); cacheErr != nil {
			// Log error but don't fail the request
			fmt.Printf("Warning: failed to cache processed text: %v\n", cacheErr)
		}
	}

	return result, nil
}

// processImageURLContent processes image_url content
// If URL is uploader wrapper or HTTP, fetch and process it
func processImageURLContent(
	ctx *agentContext.Context,
	part *agentContext.ContentPart,
	capabilities *openai.Capabilities,
	uses *agentContext.Uses,
	forceUses bool,
	registry *Registry,
	fetcher Fetcher,
	processedFiles map[string]string,
) (*Result, error) {
	if part.ImageURL == nil || part.ImageURL.URL == "" {
		return nil, fmt.Errorf("image_url content missing URL")
	}

	url := part.ImageURL.URL

	// If it's a data URI (base64), pass through
	if strings.HasPrefix(url, "data:") {
		return &Result{
			ContentPart: part,
		}, nil
	}

	// If it's uploader wrapper or HTTP URL, need to process
	// Check cache first
	cachedText, found, err := tryGetCachedText(ctx, url, processedFiles)
	if err != nil {
		return nil, fmt.Errorf("failed to check cache: %w", err)
	}
	if found {
		// Cache hit! Return as text
		return &Result{
			Text: cachedText,
		}, nil
	}

	// Determine source
	source, sourceURL, err := determineContentSource(part)
	if err != nil {
		return nil, fmt.Errorf("failed to determine content source: %w", err)
	}

	// Fetch content
	info, err := fetcher.Fetch(ctx, source, sourceURL)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch image: %w", err)
	}

	// Set file type as image
	info.FileType = FileTypeImage

	// Process with image handler
	result, err := registry.Handle(ctx, info, capabilities, uses, forceUses)
	if err != nil {
		return nil, fmt.Errorf("failed to handle image: %w", err)
	}

	// Cache if result is text
	if result.Text != "" {
		if cacheErr := cacheProcessedText(ctx, url, result.Text, processedFiles); cacheErr != nil {
			fmt.Printf("Warning: failed to cache processed text: %v\n", cacheErr)
		}
	}

	return result, nil
}

// processAudioContent processes input_audio content
func processAudioContent(
	ctx *agentContext.Context,
	part *agentContext.ContentPart,
	capabilities *openai.Capabilities,
	uses *agentContext.Uses,
	registry *Registry,
	fetcher Fetcher,
	processedFiles map[string]string,
) (*Result, error) {
	if part.InputAudio == nil || part.InputAudio.Data == "" {
		return nil, fmt.Errorf("input_audio content missing data")
	}

	// For now, pass through audio as-is
	// TODO: Implement audio processing (transcription, etc.)
	return &Result{
		ContentPart: part,
	}, nil
}

// processDataContent processes data content (converts to text)
func processDataContent(ctx *agentContext.Context, part *agentContext.ContentPart) (*Result, error) {
	if part.Data == nil {
		return nil, fmt.Errorf("data content part missing data")
	}

	// TODO: Implement data processing
	// For now, just return error
	return nil, fmt.Errorf("data content processing not implemented yet")
}

// determineContentSource determines where the content comes from
func determineContentSource(part *agentContext.ContentPart) (Source, string, error) {
	var url string

	// Extract URL based on content type
	switch part.Type {
	case agentContext.ContentFile:
		if part.File == nil || part.File.URL == "" {
			return "", "", fmt.Errorf("file content missing URL")
		}
		url = part.File.URL

	case agentContext.ContentImageURL:
		if part.ImageURL == nil || part.ImageURL.URL == "" {
			return "", "", fmt.Errorf("image_url content missing URL")
		}
		url = part.ImageURL.URL

	case agentContext.ContentInputAudio:
		if part.InputAudio == nil || part.InputAudio.Data == "" {
			return "", "", fmt.Errorf("input_audio content missing data")
		}
		// Audio data is base64, treat as base64 source
		return SourceBase64, part.InputAudio.Data, nil

	default:
		return "", "", fmt.Errorf("unsupported content type for source detection: %s", part.Type)
	}

	// Determine source type based on URL format
	if strings.HasPrefix(url, "http://") || strings.HasPrefix(url, "https://") {
		return SourceHTTP, url, nil
	}

	if strings.HasPrefix(url, "__") {
		// Uploader wrapper format: __uploader://fileid
		return SourceUploader, url, nil
	}

	if strings.HasPrefix(url, "data:") {
		// Data URI (base64)
		return SourceBase64, url, nil
	}

	// Default to treating as uploader if no prefix matches
	return SourceUploader, url, nil
}

// shouldProcessWithModel checks if content should be processed by the model directly
func shouldProcessWithModel(capabilities *openai.Capabilities, fileType FileType) (bool, agentContext.VisionFormat) {
	// TODO: Implement model capability check
	// For images: check if model supports vision
	// For audio: check if model supports audio input
	// Return whether to use model and the format to use
	return false, agentContext.VisionFormatNone
}

// getToolForProcessing gets the agent/MCP tool to use for processing
func getToolForProcessing(uses *agentContext.Uses, fileType FileType) string {
	// TODO: Implement tool selection
	// Based on file type, return the appropriate tool from uses
	// - Images -> uses.Vision
	// - Audio -> uses.Audio
	// - PDF (if vision available) -> uses.Vision
	return ""
}

// tryGetCachedText checks if the URL is an uploader wrapper and tries to get cached text
// Returns (text, found, error)
func tryGetCachedText(ctx *agentContext.Context, url string, processedFiles map[string]string) (string, bool, error) {
	// Parse URL to check if it's an uploader wrapper
	uploaderName, fileID, isWrapper := attachment.Parse(url)
	if !isWrapper {
		return "", false, nil // Not an uploader wrapper, no cache
	}

	// 1. Check in-memory cache for this Vision call
	if text, ok := processedFiles[fileID]; ok {
		return text, true, nil
	}

	// 2. Try attachment manager's content_preview (cross-call cache)
	manager, exists := attachment.Managers[uploaderName]
	if exists {
		// GetText with fullContent=false to get preview (default)
		text, err := manager.GetText(ctx.Context, fileID, false)
		if err == nil && text != "" {
			// Cache in-memory for this Vision call
			processedFiles[fileID] = text
			return text, true, nil
		}
	}

	// No cache found
	return "", false, nil
}

// cacheProcessedText caches the processed text for an uploader wrapper
func cacheProcessedText(ctx *agentContext.Context, url string, text string, processedFiles map[string]string) error {
	// Parse URL to get uploader name and file ID
	uploaderName, fileID, isWrapper := attachment.Parse(url)
	if !isWrapper {
		return nil // Not an uploader wrapper, nothing to cache
	}

	// 1. Cache in-memory for this Vision call
	processedFiles[fileID] = text

	// 2. Save to attachment manager for future Vision calls
	manager, exists := attachment.Managers[uploaderName]
	if exists {
		return manager.SaveText(ctx.Context, fileID, text)
	}

	return nil
}
