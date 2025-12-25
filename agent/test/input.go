package test

import (
	"encoding/base64"
	"fmt"
	"mime"
	"os"
	"path/filepath"
	"strings"

	jsoniter "github.com/json-iterator/go"
	"github.com/yaoapp/yao/agent/context"
)

// FileProtocol is the protocol prefix for local file references
const FileProtocol = "file://"

// SupportedImageExtensions lists supported image file extensions
var SupportedImageExtensions = map[string]string{
	".jpg":  "image/jpeg",
	".jpeg": "image/jpeg",
	".png":  "image/png",
	".gif":  "image/gif",
	".webp": "image/webp",
	".bmp":  "image/bmp",
}

// SupportedAudioExtensions lists supported audio file extensions
var SupportedAudioExtensions = map[string]string{
	".wav":  "wav",
	".mp3":  "mp3",
	".flac": "flac",
	".ogg":  "ogg",
	".m4a":  "m4a",
}

// SupportedFileExtensions lists supported document file extensions
var SupportedFileExtensions = map[string]string{
	// Documents
	".pdf":  "application/pdf",
	".doc":  "application/msword",
	".docx": "application/vnd.openxmlformats-officedocument.wordprocessingml.document",
	".xls":  "application/vnd.ms-excel",
	".xlsx": "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
	".txt":  "text/plain",
	".csv":  "text/csv",
	".json": "application/json",
	".xml":  "application/xml",
	".html": "text/html",
	".htm":  "text/html",
	".md":   "text/markdown",

	// Source code
	".yao":    "application/json",   // Yao DSL files (JSON-based)
	".ts":     "text/typescript",    // TypeScript
	".tsx":    "text/typescript",    // TypeScript JSX
	".js":     "text/javascript",    // JavaScript
	".jsx":    "text/javascript",    // JavaScript JSX
	".go":     "text/x-go",          // Go
	".py":     "text/x-python",      // Python
	".rs":     "text/x-rust",        // Rust
	".java":   "text/x-java",        // Java
	".c":      "text/x-c",           // C
	".cpp":    "text/x-c++",         // C++
	".h":      "text/x-c",           // C header
	".hpp":    "text/x-c++",         // C++ header
	".rb":     "text/x-ruby",        // Ruby
	".php":    "text/x-php",         // PHP
	".sh":     "text/x-shellscript", // Shell script
	".bash":   "text/x-shellscript", // Bash script
	".zsh":    "text/x-shellscript", // Zsh script
	".sql":    "text/x-sql",         // SQL
	".yaml":   "text/yaml",          // YAML
	".yml":    "text/yaml",          // YAML
	".toml":   "text/x-toml",        // TOML
	".ini":    "text/x-ini",         // INI
	".conf":   "text/plain",         // Config files
	".css":    "text/css",           // CSS
	".scss":   "text/x-scss",        // SCSS
	".less":   "text/x-less",        // LESS
	".vue":    "text/x-vue",         // Vue
	".svelte": "text/x-svelte",      // Svelte
}

// InputOptions configures how input is parsed
type InputOptions struct {
	// BaseDir is the base directory for resolving relative file paths
	// If empty, the current working directory is used
	BaseDir string
}

// ParseInput converts various input formats to []context.Message
// Supported formats:
//   - string: converted to single user message
//   - map (Message): single message with role and content
//   - []interface{} ([]Message): array of messages (conversation history)
func ParseInput(input interface{}) ([]context.Message, error) {
	return ParseInputWithOptions(input, nil)
}

// ParseInputWithOptions converts various input formats to []context.Message with options
// Supported formats:
//   - string: converted to single user message
//   - map (Message): single message with role and content
//   - []interface{} ([]Message): array of messages (conversation history)
//
// File references in content parts (type="image", "file", "audio") with "source" field
// starting with "file://" will be loaded and converted to appropriate format:
//   - Images: converted to base64 data URL in image_url field
//   - Audio: converted to base64 in input_audio field
//   - Files: converted to base64 data URL in file field
func ParseInputWithOptions(input interface{}, opts *InputOptions) ([]context.Message, error) {
	if input == nil {
		return nil, fmt.Errorf("input is nil")
	}

	if opts == nil {
		opts = &InputOptions{}
	}

	switch v := input.(type) {
	case string:
		// Simple string input -> single user message
		return []context.Message{
			{
				Role:    context.RoleUser,
				Content: v,
			},
		}, nil

	case map[string]interface{}:
		// Single message object
		msg, err := parseMessageMap(v, opts)
		if err != nil {
			return nil, fmt.Errorf("failed to parse message: %w", err)
		}
		return []context.Message{*msg}, nil

	case []interface{}:
		// Array of messages (conversation history)
		messages := make([]context.Message, 0, len(v))
		for i, item := range v {
			switch m := item.(type) {
			case map[string]interface{}:
				msg, err := parseMessageMap(m, opts)
				if err != nil {
					return nil, fmt.Errorf("failed to parse message at index %d: %w", i, err)
				}
				messages = append(messages, *msg)
			default:
				return nil, fmt.Errorf("invalid message type at index %d: expected object, got %T", i, item)
			}
		}
		return messages, nil

	default:
		return nil, fmt.Errorf("unsupported input type: %T", input)
	}
}

// parseMessageMap converts a map to context.Message
func parseMessageMap(m map[string]interface{}, opts *InputOptions) (*context.Message, error) {
	msg := &context.Message{}

	// Parse role (required)
	if role, ok := m["role"].(string); ok {
		msg.Role = context.MessageRole(role)
	} else {
		// Default to user role if not specified
		msg.Role = context.RoleUser
	}

	// Parse content (required)
	if content, ok := m["content"]; ok {
		// Process content to handle file:// references
		processedContent, err := processContent(content, opts)
		if err != nil {
			return nil, fmt.Errorf("failed to process content: %w", err)
		}
		msg.Content = processedContent
	} else {
		return nil, fmt.Errorf("message missing 'content' field")
	}

	// Parse optional name
	if name, ok := m["name"].(string); ok {
		msg.Name = &name
	}

	// Parse optional tool_call_id (for tool messages)
	if toolCallID, ok := m["tool_call_id"].(string); ok {
		msg.ToolCallID = &toolCallID
	}

	// Parse optional tool_calls (for assistant messages)
	if toolCalls, ok := m["tool_calls"].([]interface{}); ok {
		msg.ToolCalls = make([]context.ToolCall, 0, len(toolCalls))
		for _, tc := range toolCalls {
			if tcMap, ok := tc.(map[string]interface{}); ok {
				toolCall, err := parseToolCall(tcMap)
				if err != nil {
					return nil, fmt.Errorf("failed to parse tool_call: %w", err)
				}
				msg.ToolCalls = append(msg.ToolCalls, *toolCall)
			}
		}
	}

	// Parse optional refusal (for assistant messages)
	if refusal, ok := m["refusal"].(string); ok {
		msg.Refusal = &refusal
	}

	return msg, nil
}

// processContent processes content to handle file:// references
// Returns the processed content with files loaded and converted
func processContent(content interface{}, opts *InputOptions) (interface{}, error) {
	switch v := content.(type) {
	case string:
		// Simple string content, no processing needed
		return v, nil

	case []interface{}:
		// Array of content parts
		processedParts := make([]context.ContentPart, 0, len(v))
		for i, part := range v {
			if partMap, ok := part.(map[string]interface{}); ok {
				processedPart, err := processContentPart(partMap, opts)
				if err != nil {
					return nil, fmt.Errorf("failed to process content part at index %d: %w", i, err)
				}
				processedParts = append(processedParts, *processedPart)
			} else {
				return nil, fmt.Errorf("invalid content part type at index %d: expected object, got %T", i, part)
			}
		}
		return processedParts, nil

	case map[string]interface{}:
		// Single content part
		processedPart, err := processContentPart(v, opts)
		if err != nil {
			return nil, fmt.Errorf("failed to process content part: %w", err)
		}
		return []context.ContentPart{*processedPart}, nil

	default:
		return content, nil
	}
}

// processContentPart processes a single content part map
// Handles file:// references and converts them to appropriate format
func processContentPart(partMap map[string]interface{}, opts *InputOptions) (*context.ContentPart, error) {
	partType, _ := partMap["type"].(string)

	switch partType {
	case "text":
		text, _ := partMap["text"].(string)
		return &context.ContentPart{
			Type: context.ContentText,
			Text: text,
		}, nil

	case "image":
		return processImagePart(partMap, opts)

	case "image_url":
		// Already in correct format, just parse it
		return parseImageURLPart(partMap)

	case "audio", "input_audio":
		return processAudioPart(partMap, opts)

	case "file":
		return processFilePart(partMap, opts)

	case "data":
		return parseDataPart(partMap)

	default:
		// Unknown type, try to preserve as-is
		return parseGenericPart(partMap)
	}
}

// processImagePart processes an image content part
// Supports: source="file://path" for local files
func processImagePart(partMap map[string]interface{}, opts *InputOptions) (*context.ContentPart, error) {
	source, hasSource := partMap["source"].(string)

	// Check for file:// protocol
	if hasSource && strings.HasPrefix(source, FileProtocol) {
		filePath := strings.TrimPrefix(source, FileProtocol)
		return loadImageFile(filePath, opts)
	}

	// Check for url field (already a URL or base64)
	if url, ok := partMap["url"].(string); ok {
		detail := context.DetailAuto
		if d, ok := partMap["detail"].(string); ok {
			detail = context.ImageDetailLevel(d)
		}
		return &context.ContentPart{
			Type: context.ContentImageURL,
			ImageURL: &context.ImageURL{
				URL:    url,
				Detail: detail,
			},
		}, nil
	}

	return nil, fmt.Errorf("image part requires 'source' (file://...) or 'url' field")
}

// parseImageURLPart parses an image_url content part
func parseImageURLPart(partMap map[string]interface{}) (*context.ContentPart, error) {
	imageURL, ok := partMap["image_url"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("image_url part requires 'image_url' object")
	}

	url, _ := imageURL["url"].(string)
	detail := context.DetailAuto
	if d, ok := imageURL["detail"].(string); ok {
		detail = context.ImageDetailLevel(d)
	}

	return &context.ContentPart{
		Type: context.ContentImageURL,
		ImageURL: &context.ImageURL{
			URL:    url,
			Detail: detail,
		},
	}, nil
}

// processAudioPart processes an audio content part
// Supports: source="file://path" for local files
func processAudioPart(partMap map[string]interface{}, opts *InputOptions) (*context.ContentPart, error) {
	source, hasSource := partMap["source"].(string)

	// Check for file:// protocol
	if hasSource && strings.HasPrefix(source, FileProtocol) {
		filePath := strings.TrimPrefix(source, FileProtocol)
		return loadAudioFile(filePath, opts)
	}

	// Check for data field (already base64)
	if data, ok := partMap["data"].(string); ok {
		format, _ := partMap["format"].(string)
		return &context.ContentPart{
			Type: context.ContentInputAudio,
			InputAudio: &context.InputAudio{
				Data:   data,
				Format: format,
			},
		}, nil
	}

	// Check for input_audio field
	if inputAudio, ok := partMap["input_audio"].(map[string]interface{}); ok {
		data, _ := inputAudio["data"].(string)
		format, _ := inputAudio["format"].(string)
		return &context.ContentPart{
			Type: context.ContentInputAudio,
			InputAudio: &context.InputAudio{
				Data:   data,
				Format: format,
			},
		}, nil
	}

	return nil, fmt.Errorf("audio part requires 'source' (file://...) or 'data'/'input_audio' field")
}

// processFilePart processes a file content part
// Supports: source="file://path" for local files
func processFilePart(partMap map[string]interface{}, opts *InputOptions) (*context.ContentPart, error) {
	source, hasSource := partMap["source"].(string)

	// Check for file:// protocol
	if hasSource && strings.HasPrefix(source, FileProtocol) {
		filePath := strings.TrimPrefix(source, FileProtocol)
		name, _ := partMap["name"].(string)
		return loadFile(filePath, name, opts)
	}

	// Check for url field (already a URL)
	if url, ok := partMap["url"].(string); ok {
		filename, _ := partMap["filename"].(string)
		if filename == "" {
			filename, _ = partMap["name"].(string)
		}
		return &context.ContentPart{
			Type: context.ContentFile,
			File: &context.FileAttachment{
				URL:      url,
				Filename: filename,
			},
		}, nil
	}

	// Check for file field
	if file, ok := partMap["file"].(map[string]interface{}); ok {
		url, _ := file["url"].(string)
		filename, _ := file["filename"].(string)
		return &context.ContentPart{
			Type: context.ContentFile,
			File: &context.FileAttachment{
				URL:      url,
				Filename: filename,
			},
		}, nil
	}

	return nil, fmt.Errorf("file part requires 'source' (file://...), 'url', or 'file' field")
}

// parseDataPart parses a data content part
func parseDataPart(partMap map[string]interface{}) (*context.ContentPart, error) {
	data, ok := partMap["data"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("data part requires 'data' object")
	}

	// Convert to DataContent
	dataContent := &context.DataContent{}

	if sources, ok := data["sources"].([]interface{}); ok {
		dataContent.Sources = make([]context.DataSource, 0, len(sources))
		for _, src := range sources {
			if srcMap, ok := src.(map[string]interface{}); ok {
				ds := context.DataSource{}
				if t, ok := srcMap["type"].(string); ok {
					ds.Type = context.DataSourceType(t)
				}
				if id, ok := srcMap["id"].(string); ok {
					ds.ID = id
				}
				if name, ok := srcMap["name"].(string); ok {
					ds.Name = name
				}
				if filters, ok := srcMap["filters"].(map[string]interface{}); ok {
					ds.Filters = filters
				}
				if metadata, ok := srcMap["metadata"].(map[string]interface{}); ok {
					ds.Metadata = metadata
				}
				dataContent.Sources = append(dataContent.Sources, ds)
			}
		}
	}

	return &context.ContentPart{
		Type: context.ContentData,
		Data: dataContent,
	}, nil
}

// parseGenericPart tries to parse an unknown content part type
func parseGenericPart(partMap map[string]interface{}) (*context.ContentPart, error) {
	partType, _ := partMap["type"].(string)

	// Try to create a basic ContentPart
	part := &context.ContentPart{
		Type: context.ContentPartType(partType),
	}

	// Try to extract text if present
	if text, ok := partMap["text"].(string); ok {
		part.Text = text
	}

	return part, nil
}

// loadImageFile loads an image file and converts it to a ContentPart
func loadImageFile(filePath string, opts *InputOptions) (*context.ContentPart, error) {
	absPath := resolveFilePath(filePath, opts)

	// Read file
	data, err := os.ReadFile(absPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read image file %s: %w", filePath, err)
	}

	// Determine MIME type
	ext := strings.ToLower(filepath.Ext(absPath))
	mimeType, ok := SupportedImageExtensions[ext]
	if !ok {
		// Try to detect from extension
		mimeType = mime.TypeByExtension(ext)
		if mimeType == "" {
			mimeType = "application/octet-stream"
		}
	}

	// Encode to base64 data URL
	b64Data := base64.StdEncoding.EncodeToString(data)
	dataURL := fmt.Sprintf("data:%s;base64,%s", mimeType, b64Data)

	return &context.ContentPart{
		Type: context.ContentImageURL,
		ImageURL: &context.ImageURL{
			URL:    dataURL,
			Detail: context.DetailAuto,
		},
	}, nil
}

// loadAudioFile loads an audio file and converts it to a ContentPart
func loadAudioFile(filePath string, opts *InputOptions) (*context.ContentPart, error) {
	absPath := resolveFilePath(filePath, opts)

	// Read file
	data, err := os.ReadFile(absPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read audio file %s: %w", filePath, err)
	}

	// Determine format from extension
	ext := strings.ToLower(filepath.Ext(absPath))
	format, ok := SupportedAudioExtensions[ext]
	if !ok {
		format = strings.TrimPrefix(ext, ".")
	}

	// Encode to base64
	b64Data := base64.StdEncoding.EncodeToString(data)

	return &context.ContentPart{
		Type: context.ContentInputAudio,
		InputAudio: &context.InputAudio{
			Data:   b64Data,
			Format: format,
		},
	}, nil
}

// loadFile loads a file and converts it to a ContentPart
func loadFile(filePath string, name string, opts *InputOptions) (*context.ContentPart, error) {
	absPath := resolveFilePath(filePath, opts)

	// Read file
	data, err := os.ReadFile(absPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file %s: %w", filePath, err)
	}

	// Determine filename
	filename := name
	if filename == "" {
		filename = filepath.Base(absPath)
	}

	// Determine MIME type
	ext := strings.ToLower(filepath.Ext(absPath))
	mimeType, ok := SupportedFileExtensions[ext]
	if !ok {
		mimeType = mime.TypeByExtension(ext)
		if mimeType == "" {
			mimeType = "application/octet-stream"
		}
	}

	// Encode to base64 data URL
	b64Data := base64.StdEncoding.EncodeToString(data)
	dataURL := fmt.Sprintf("data:%s;base64,%s", mimeType, b64Data)

	return &context.ContentPart{
		Type: context.ContentFile,
		File: &context.FileAttachment{
			URL:      dataURL,
			Filename: filename,
		},
	}, nil
}

// resolveFilePath resolves a file path relative to the base directory
// If the path is absolute, it's returned as-is
// If BaseDir is empty, the current working directory is used
func resolveFilePath(filePath string, opts *InputOptions) string {
	// If path is absolute, return as-is
	if filepath.IsAbs(filePath) {
		return filePath
	}

	// If BaseDir is set, resolve relative to it
	if opts != nil && opts.BaseDir != "" {
		return filepath.Join(opts.BaseDir, filePath)
	}

	// Otherwise, resolve relative to current working directory
	return filePath
}

// parseToolCall converts a map to context.ToolCall
func parseToolCall(m map[string]interface{}) (*context.ToolCall, error) {
	tc := &context.ToolCall{}

	if id, ok := m["id"].(string); ok {
		tc.ID = id
	}

	if typ, ok := m["type"].(string); ok {
		tc.Type = context.ToolCallType(typ)
	} else {
		tc.Type = context.ToolTypeFunction
	}

	if fn, ok := m["function"].(map[string]interface{}); ok {
		if name, ok := fn["name"].(string); ok {
			tc.Function.Name = name
		}
		if args, ok := fn["arguments"].(string); ok {
			tc.Function.Arguments = args
		} else if args, ok := fn["arguments"].(map[string]interface{}); ok {
			// Convert map to JSON string
			argsBytes, err := jsoniter.Marshal(args)
			if err != nil {
				return nil, fmt.Errorf("failed to marshal arguments: %w", err)
			}
			tc.Function.Arguments = string(argsBytes)
		}
	}

	return tc, nil
}

// ExtractTextContent extracts text content from various content formats
// Used for display in reports
func ExtractTextContent(content interface{}) string {
	if content == nil {
		return ""
	}

	switch v := content.(type) {
	case string:
		return v

	case []interface{}:
		// ContentPart array
		var texts []string
		for _, part := range v {
			if partMap, ok := part.(map[string]interface{}); ok {
				if partMap["type"] == "text" {
					if text, ok := partMap["text"].(string); ok {
						texts = append(texts, text)
					}
				}
			}
		}
		if len(texts) > 0 {
			result := texts[0]
			for i := 1; i < len(texts); i++ {
				result += "\n" + texts[i]
			}
			return result
		}
		return fmt.Sprintf("[%d content parts]", len(v))

	case map[string]interface{}:
		// Single ContentPart or Message
		if v["type"] == "text" {
			if text, ok := v["text"].(string); ok {
				return text
			}
		}
		if content, ok := v["content"]; ok {
			return ExtractTextContent(content)
		}
		return fmt.Sprintf("%v", v)

	default:
		return fmt.Sprintf("%v", v)
	}
}

// SummarizeInput creates a short summary of the input for display
func SummarizeInput(input interface{}, maxLen int) string {
	text := ""

	switch v := input.(type) {
	case string:
		text = v

	case map[string]interface{}:
		if content, ok := v["content"]; ok {
			text = ExtractTextContent(content)
		}

	case []interface{}:
		// Get the last user message for summary
		for i := len(v) - 1; i >= 0; i-- {
			if msg, ok := v[i].(map[string]interface{}); ok {
				if msg["role"] == "user" {
					if content, ok := msg["content"]; ok {
						text = ExtractTextContent(content)
						break
					}
				}
			}
		}
		if text == "" && len(v) > 0 {
			text = fmt.Sprintf("[%d messages]", len(v))
		}

	default:
		text = fmt.Sprintf("%v", v)
	}

	if maxLen > 0 && len(text) > maxLen {
		return text[:maxLen-3] + "..."
	}
	return text
}
