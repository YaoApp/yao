package content

import (
	"fmt"

	agentContext "github.com/yaoapp/yao/agent/context"
	"github.com/yaoapp/yao/attachment"
)

// DefaultFetcher implements the Fetcher interface
type DefaultFetcher struct{}

// NewFetcher creates a new default fetcher
func NewFetcher() Fetcher {
	return &DefaultFetcher{}
}

// Fetch retrieves content from HTTP URL or uploader wrapper
func (f *DefaultFetcher) Fetch(ctx *agentContext.Context, source Source, url string) (*Info, error) {
	switch source {
	case SourceHTTP:
		return f.fetchHTTP(ctx, url)
	case SourceUploader:
		return f.fetchUploader(ctx, url)
	default:
		return nil, fmt.Errorf("unsupported source: %s", source)
	}
}

// fetchHTTP fetches content from an HTTP(S) URL
func (f *DefaultFetcher) fetchHTTP(ctx *agentContext.Context, url string) (*Info, error) {
	// TODO: Implement HTTP fetch logic
	// 1. Download file from URL
	// 2. Detect content type
	// 3. Detect file type based on content type and extension
	// 4. Return Info with data
	return nil, fmt.Errorf("not implemented")
}

// fetchUploader fetches content from uploader wrapper (__uploader://fileid)
func (f *DefaultFetcher) fetchUploader(ctx *agentContext.Context, wrapper string) (*Info, error) {
	// 1. Parse wrapper to get uploader name and file ID
	uploaderName, fileID, ok := attachment.Parse(wrapper)
	if !ok {
		return nil, fmt.Errorf("invalid uploader wrapper format: %s", wrapper)
	}

	// 2. Get attachment manager
	var manager attachment.FileManager
	var exists bool

	// Try to get manager by name
	manager, exists = attachment.Managers[uploaderName]
	if !exists {
		return nil, fmt.Errorf("uploader '%s' not found", uploaderName)
	}

	// 3. Get file info
	file, err := manager.Info(ctx.Context, fileID)
	if err != nil {
		return nil, fmt.Errorf("failed to get file info: %w", err)
	}

	// 4. Read file content
	data, err := manager.Read(ctx.Context, fileID)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	// 5. Return Info with data
	return &Info{
		Data:        data,
		ContentType: file.ContentType,
		FileType:    DetectFileType(file.ContentType, file.Filename),
	}, nil
}

// parseUploaderWrapper parses uploader wrapper format: __uploader://fileid
func parseUploaderWrapper(wrapper string) (uploaderName, fileID string, err error) {
	// TODO: Implement wrapper parsing
	// Format: __uploader://fileid
	return "", "", fmt.Errorf("not implemented")
}

// detectFileType detects file type from content type and data
func detectFileType(contentType string, data []byte) FileType {
	// TODO: Implement file type detection
	// Based on content type and magic bytes
	return FileTypeUnknown
}
