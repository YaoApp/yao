package pdf

import (
	"fmt"

	"github.com/yaoapp/yao/agent/content/types"
	agentContext "github.com/yaoapp/yao/agent/context"
	searchTypes "github.com/yaoapp/yao/agent/search/types"
)

// PDF handlePDF content
type PDF struct {
	options *types.Options
}

// New creates a new image handler
func New(options *types.Options) *PDF {
	return &PDF{options: options}
}

// Parse parses pdf content
func (h *PDF) Parse(ctx *agentContext.Context, content agentContext.ContentPart) (agentContext.ContentPart, []*searchTypes.Reference, error) {
	return content, nil, nil
}

// asImages converts pdf content to images then parse as image content
func (h *PDF) asImages(ctx *agentContext.Context, content agentContext.ContentPart) (agentContext.ContentPart, []*searchTypes.Reference, error) {
	return agentContext.ContentPart{}, nil, nil
}

// base64 encodes image content to base64 ( for PDF support )
func (h *PDF) base64(ctx *agentContext.Context, content agentContext.ContentPart) (agentContext.ContentPart, []*searchTypes.Reference, error) {
	return agentContext.ContentPart{}, nil, nil
}

// read image content from file uploader __uploader://fileid return agentContext.ContentPart
func (h *PDF) read(ctx *agentContext.Context, content agentContext.ContentPart) (agentContext.ContentPart, []*searchTypes.Reference, error) {
	return agentContext.ContentPart{}, nil, fmt.Errorf("not implemented")
}

// read image content from file uploader __uploader://fileid return agentContext.ContentPart
func (h *PDF) readFromCache(ctx *agentContext.Context, content agentContext.ContentPart) (agentContext.ContentPart, []*searchTypes.Reference, error) {
	return agentContext.ContentPart{}, nil, fmt.Errorf("not implemented")
}

// read image content from file uploader __uploader://fileid return base64 encoded string
func (h *PDF) readFromUploader(ctx *agentContext.Context, content agentContext.ContentPart) (string, error) {
	return "", fmt.Errorf("not implemented")
}
