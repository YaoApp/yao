package attachment

import (
	"context"
	"io"
	"mime/multipart"
)

// File the file
type File struct {
	ID          string `json:"file_id"`
	Bytes       int    `json:"bytes"`
	CreatedAt   int    `json:"created_at"`
	Filename    string `json:"filename"`
	ContentType string `json:"content_type"`
	Status      string `json:"status"` // uploading, uploaded, indexing, indexed, upload_failed, index_failed
}

// FileResponse represents a file download response
type FileResponse struct {
	Reader      io.ReadCloser
	ContentType string
	Extension   string
}

// Attachment represents a file attachment
type Attachment struct {
	Name        string `json:"name,omitempty"`
	URL         string `json:"url,omitempty"`
	Description string `json:"description,omitempty"`
	Type        string `json:"type,omitempty"`
	ContentType string `json:"content_type,omitempty"`
	Bytes       int64  `json:"bytes,omitempty"`
	CreatedAt   int64  `json:"created_at,omitempty"`
	FileID      string `json:"file_id,omitempty"`
	ChatID      string `json:"chat_id,omitempty"`
	AssistantID string `json:"assistant_id,omitempty"`
	Gzip        bool   `json:"gzip,omitempty"` // Gzip the file, Optional, default is false
}

// Manager the manager struct
type Manager struct {
	ManagerOption
	storage      Storage
	maxsize      int64
	chunsize     int64
	allowedTypes allowedType
}

// Storage the storage interface
type Storage interface {
	Upload(ctx context.Context, fileID string, reader io.Reader, contentType string) (string, error)
	UploadChunk(ctx context.Context, fileID string, chunkIndex int, reader io.Reader, contentType string) error
	MergeChunks(ctx context.Context, fileID string, totalChunks int) error
	Download(ctx context.Context, fileID string) (io.ReadCloser, string, error)
	Reader(ctx context.Context, fileID string) (io.ReadCloser, error)
	URL(ctx context.Context, fileID string) string
	Exists(ctx context.Context, fileID string) bool
	Delete(ctx context.Context, fileID string) error
}

// ManagerOption the manager option
type ManagerOption struct {
	MaxSize      string                 `json:"max_size,omitempty" yaml:"max_size,omitempty"`           // Max size of the file, Optional, default is 20M
	ChunkSize    string                 `json:"chunk_size,omitempty" yaml:"chunk_size,omitempty"`       // Chunk size of the file, Optional, default is 2M
	AllowedTypes []string               `json:"allowed_types,omitempty" yaml:"allowed_types,omitempty"` // Allowed types of the file, Optional, default is all
	Driver       string                 `json:"driver,omitempty" yaml:"driver,omitempty"`               // Driver, Optional, default is local
	Options      map[string]interface{} `json:"options,omitempty" yaml:"options,omitempty"`             // Options, Optional
}

type allowedType struct {
	mapping   map[string]bool
	wildcards []string // Wildcard patterns for file types (e.g., "image/*", "text/*")
}

// UploadOption the upload option
type UploadOption struct {
	CompressImage    bool   `json:"compress_image,omitempty" form:"compress_image"`       // Compress the file, Optional, default is true
	CompressSize     int    `json:"compress_size,omitempty" form:"compress_size"`         // Compress the file size, Optional, default is 1920, if compress_image is true, the file size will be compressed to the compress_size
	Gzip             bool   `json:"gzip,omitempty" form:"gzip"`                           // Gzip the file, Optional, default is false
	Knowledge        bool   `json:"knowledge,omitempty" form:"knowledge"`                 // Push to knowledge base, Optional, default is false
	ChatID           string `json:"chat_id,omitempty" form:"chat_id"`                     // Chat ID, Optional
	AssistantID      string `json:"assistant_id,omitempty" form:"assistant_id"`           // Assistant ID, Optional
	UserID           string `json:"user_id,omitempty"`                                    // User ID, Optional
	OriginalFilename string `json:"original_filename,omitempty" form:"original_filename"` // Original filename sent separately to avoid encoding issues
}

// FileHeader the file header
type FileHeader struct {
	*multipart.FileHeader
}
