package content

import (
	"fmt"
	"strings"

	"github.com/yaoapp/gou/connector/openai"
	agentContext "github.com/yaoapp/yao/agent/context"
)

// WordHandler handles Microsoft Word documents
type WordHandler struct{}

// CanHandle checks if this handler can handle the content type
func (h *WordHandler) CanHandle(contentType string, fileType FileType) bool {
	return fileType == FileTypeWord ||
		contentType == "application/vnd.openxmlformats-officedocument.wordprocessingml.document" ||
		contentType == "application/msword" ||
		strings.Contains(contentType, "word")
}

// Handle processes Word document content
func (h *WordHandler) Handle(ctx *agentContext.Context, info *Info, capabilities *openai.Capabilities, uses *agentContext.Uses) (*Result, error) {
	// TODO: Implement Word document handling
	// 1. Extract text from .docx or .doc file
	// 2. Preserve formatting information if needed
	// 3. Return Result with extracted text
	return nil, fmt.Errorf("not implemented")
}

// extractWordText extracts text from Word document
func extractWordText(data []byte, contentType string) (string, error) {
	// TODO: Implement Word text extraction
	// Handle both .doc (old format) and .docx (new format)
	// Consider using libraries like:
	// - github.com/unidoc/unioffice for .docx
	// - Other libraries for .doc
	return "", fmt.Errorf("not implemented")
}
