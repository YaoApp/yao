package content

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/yaoapp/gou/connector/openai"
	agentContext "github.com/yaoapp/yao/agent/context"
)

// TextHandler handles plain text, code files, CSV, JSON, XML, Markdown, etc.
type TextHandler struct{}

// CanHandle checks if this handler can handle the content type
func (h *TextHandler) CanHandle(contentType string, fileType FileType) bool {
	// Handle explicit text file types
	if fileType == FileTypeText || fileType == FileTypeCSV || fileType == FileTypeJSON {
		return true
	}

	// Handle text-based MIME types
	if strings.HasPrefix(contentType, "text/") {
		return true
	}

	// Handle common text-based content types
	textContentTypes := []string{
		"application/json",
		"application/xml",
		"application/javascript",
		"application/typescript",
		"application/x-yaml",
		"application/yaml",
		"application/toml",
		"application/x-sh",
		"application/x-python",
		"application/x-ruby",
		"application/x-perl",
	}

	for _, ct := range textContentTypes {
		if contentType == ct || strings.Contains(contentType, ct) {
			return true
		}
	}

	return false
}

// Handle processes text content
func (h *TextHandler) Handle(ctx *agentContext.Context, info *Info, capabilities *openai.Capabilities, uses *agentContext.Uses, forceUses bool) (*Result, error) {
	if len(info.Data) == 0 {
		return nil, fmt.Errorf("no data to process")
	}

	var text string
	var err error

	// Handle different text formats
	switch {
	case info.FileType == FileTypeCSV || strings.Contains(info.ContentType, "csv"):
		// Format CSV as readable text (for now, just return as-is, can enhance later)
		text = string(info.Data)

	case info.FileType == FileTypeJSON ||
		info.ContentType == "application/json" ||
		strings.Contains(info.ContentType, "json"):
		// Pretty print JSON
		text, err = formatJSONAsText(info.Data)
		if err != nil {
			// If JSON parsing fails, return raw text
			text = string(info.Data)
		}

	case info.ContentType == "application/xml" ||
		strings.Contains(info.ContentType, "xml"):
		// For now, return XML as-is (can enhance formatting later)
		text = string(info.Data)

	default:
		// Plain text, code files, markdown, etc.
		text, err = readTextContent(info.Data, info.ContentType)
		if err != nil {
			return nil, fmt.Errorf("failed to read text content: %w", err)
		}
	}

	return &Result{
		Text: text,
	}, nil
}

// readTextContent reads text content from data
func readTextContent(data []byte, contentType string) (string, error) {
	// For now, assume UTF-8 encoding
	// TODO: Add encoding detection if needed (e.g., using golang.org/x/text/encoding)
	return string(data), nil
}

// formatCSVAsText formats CSV data as readable text
func formatCSVAsText(data []byte) (string, error) {
	// TODO: Parse CSV and format as readable table
	// Consider using encoding/csv package
	// For now, just return as-is
	return string(data), nil
}

// formatJSONAsText formats JSON data as readable text
func formatJSONAsText(data []byte) (string, error) {
	// Pretty print JSON with indentation
	var obj interface{}
	if err := json.Unmarshal(data, &obj); err != nil {
		return "", err
	}

	pretty, err := json.MarshalIndent(obj, "", "  ")
	if err != nil {
		return "", err
	}

	return string(pretty), nil
}
