package attachment

import (
	"context"
	"io"
	"mime/multipart"

	"github.com/yaoapp/gou/model"
	"github.com/yaoapp/gou/types"
)

// FileManager defines the interface for file management operations.
// This interface provides abstraction for file operations, making it easier to:
// - Write unit tests with mock implementations
// - Switch between different storage backends
// - Maintain consistent API across different implementations
//
// Example usage:
//
//	var fileManager FileManager = manager // Manager implements FileManager
//	file, err := fileManager.Upload(ctx, header, reader, options)
//	data, err := fileManager.Read(ctx, file.ID)
type FileManager interface {
	// Upload uploads a file with optional chunked upload support
	Upload(ctx context.Context, fileheader *FileHeader, reader io.Reader, option UploadOption) (*File, error)

	// Download downloads a file by its ID
	Download(ctx context.Context, fileID string) (*FileResponse, error)

	// Read reads a file content as bytes
	Read(ctx context.Context, fileID string) ([]byte, error)

	// ReadBase64 reads a file content as base64 encoded string
	ReadBase64(ctx context.Context, fileID string) (string, error)

	// Info retrieves complete file information from database by file ID
	Info(ctx context.Context, fileID string) (*File, error)

	// List retrieves files from database with pagination and filtering
	List(ctx context.Context, option ListOption) (*ListResult, error)

	// Exists checks if a file exists
	Exists(ctx context.Context, fileID string) bool

	// Delete deletes a file
	Delete(ctx context.Context, fileID string) error

	// LocalPath gets the local path of the file
	LocalPath(ctx context.Context, fileID string) (string, string, error)

	// GetText retrieves the parsed text content for a file
	// By default returns preview (first 2000 chars), set fullContent=true for complete text
	GetText(ctx context.Context, fileID string, fullContent ...bool) (string, error)

	// SaveText saves the parsed text content for a file
	// Automatically saves both full content and preview
	SaveText(ctx context.Context, fileID string, text string) error
}

// File the file
type File struct {
	ID          string `json:"file_id"`
	UserPath    string `json:"user_path"` // User-specified complete file path
	Path        string `json:"path"`      // Actual storage path
	Bytes       int    `json:"bytes"`
	CreatedAt   int    `json:"created_at"`
	Filename    string `json:"filename"`
	ContentType string `json:"content_type"`
	Status      string `json:"status"` // uploading, uploaded, indexing, indexed, upload_failed, index_failed

	// Permission fields
	Public       bool   `json:"public,omitempty"` // Whether this attachment is shared across all teams
	Share        string `json:"share,omitempty"`  // Attachment sharing scope: "private" or "team"
	YaoCreatedBy string `json:"-"`                // User who created the attachment (not exposed in JSON)
	YaoTeamID    string `json:"-"`                // Team ID for team-based access control (not exposed in JSON)
	YaoTenantID  string `json:"-"`                // Tenant ID for multi-tenancy support (not exposed in JSON)
}

// FileResponse represents a file download response
type FileResponse struct {
	Reader      io.ReadCloser
	ContentType string
	Extension   string
}

// Attachment represents a file attachment
type Attachment struct {
	Name        string   `json:"name,omitempty"`
	URL         string   `json:"url,omitempty"`
	Description string   `json:"description,omitempty"`
	Type        string   `json:"type,omitempty"`
	ContentType string   `json:"content_type,omitempty"`
	Bytes       int64    `json:"bytes,omitempty"`
	CreatedAt   int64    `json:"created_at,omitempty"`
	FileID      string   `json:"file_id,omitempty"`
	UserPath    string   `json:"user_path,omitempty"` // User-specified complete file path
	Path        string   `json:"path,omitempty"`      // Actual storage path
	Groups      []string `json:"groups,omitempty"`
	Gzip        bool     `json:"gzip,omitempty"`   // Gzip the file, Optional, default is false
	Public      bool     `json:"public,omitempty"` // Whether this attachment is shared across all teams in the platform
	Share       string   `json:"share,omitempty"`  // Attachment sharing scope: "private" or "team"

	// Yao custom fields for permission control
	YaoCreatedBy string `json:"__yao_created_by,omitempty"` // User who created the attachment
	YaoUpdatedBy string `json:"__yao_updated_by,omitempty"` // User who last updated the attachment
	YaoTeamID    string `json:"__yao_team_id,omitempty"`    // Team ID for team-based access control
	YaoTenantID  string `json:"__yao_tenant_id,omitempty"`  // Tenant ID for multi-tenancy support
}

// Manager the manager struct
type Manager struct {
	ManagerOption
	Name         string // Manager name for identification
	storage      Storage
	maxsize      int64
	chunsize     int64
	allowedTypes allowedType
}

// Storage the storage interface
type Storage interface {
	Upload(ctx context.Context, path string, reader io.Reader, contentType string) (string, error)
	UploadChunk(ctx context.Context, path string, chunkIndex int, reader io.Reader, contentType string) error
	MergeChunks(ctx context.Context, path string, totalChunks int) error
	Download(ctx context.Context, path string) (io.ReadCloser, string, error)
	Reader(ctx context.Context, path string) (io.ReadCloser, error)
	GetContent(ctx context.Context, path string) ([]byte, error)
	URL(ctx context.Context, path string) string
	Exists(ctx context.Context, path string) bool
	Delete(ctx context.Context, path string) error
	LocalPath(ctx context.Context, path string) (string, string, error) // Returns absolute path and content type
}

// ManagerOption the manager option
type ManagerOption struct {
	types.MetaInfo
	MaxSize      string                 `json:"max_size,omitempty" yaml:"max_size,omitempty"`           // Max size of the file, Optional, default is 20M
	ChunkSize    string                 `json:"chunk_size,omitempty" yaml:"chunk_size,omitempty"`       // Chunk size of the file, Optional, default is 2M
	AllowedTypes []string               `json:"allowed_types,omitempty" yaml:"allowed_types,omitempty"` // Allowed types of the file, Optional, default is all
	Gzip         bool                   `json:"gzip,omitempty" yaml:"gzip,omitempty"`                   // Gzip the file, Optional, default is false
	Driver       string                 `json:"driver,omitempty" yaml:"driver,omitempty"`               // Driver, Optional, default is local
	Options      map[string]interface{} `json:"options,omitempty" yaml:"options,omitempty"`             // Options, Optional
}

type allowedType struct {
	mapping   map[string]bool
	wildcards []string // Wildcard patterns for file types (e.g., "image/*", "text/*")
}

// UploadOption the upload option
type UploadOption struct {
	CompressImage    bool     `json:"compress_image,omitempty" form:"compress_image"`       // Compress the file, Optional, default is true
	CompressSize     int      `json:"compress_size,omitempty" form:"compress_size"`         // Compress the file size, Optional, default is 1920, if compress_image is true, the file size will be compressed to the compress_size
	Gzip             bool     `json:"gzip,omitempty" form:"gzip"`                           // Gzip the file, Optional, default is false
	OriginalFilename string   `json:"original_filename,omitempty" form:"original_filename"` // Original filename sent separately to avoid encoding issues
	Groups           []string `json:"groups,omitempty" form:"groups"`                       // Groups, Optional, default is empty, Multi-level groups like ["user", "user123", "chat", "chat456"]
	Public           bool     `json:"public,omitempty" form:"public"`                       // Whether this attachment is shared across all teams in the platform
	Share            string   `json:"share,omitempty" form:"share"`                         // Attachment sharing scope: "private" or "team"

	// Yao custom fields for permission control
	YaoCreatedBy string `json:"__yao_created_by,omitempty" form:"__yao_created_by"` // User who created the attachment
	YaoUpdatedBy string `json:"__yao_updated_by,omitempty" form:"__yao_updated_by"` // User who last updated the attachment
	YaoTeamID    string `json:"__yao_team_id,omitempty" form:"__yao_team_id"`       // Team ID for team-based access control
	YaoTenantID  string `json:"__yao_tenant_id,omitempty" form:"__yao_tenant_id"`   // Tenant ID for multi-tenancy support
}

// ListOption defines options for listing files
type ListOption struct {
	Page     int                    `json:"page,omitempty"`      // Page number (1-based), default is 1
	PageSize int                    `json:"page_size,omitempty"` // Page size, default is 20
	Filters  map[string]interface{} `json:"filters,omitempty"`   // Filter conditions, e.g., {"status": "uploaded", "content_type": "image/*"}
	Wheres   []model.QueryWhere     `json:"wheres,omitempty"`    // Advanced where clauses for permission filtering
	OrderBy  string                 `json:"order_by,omitempty"`  // Order by field, e.g., "created_at desc", "name asc"
	Select   []string               `json:"select,omitempty"`    // Fields to select, empty means select all
}

// ListResult contains the paginated list result
type ListResult struct {
	Files      []*File `json:"files"`       // List of files
	Total      int64   `json:"total"`       // Total count
	Page       int     `json:"page"`        // Current page
	PageSize   int     `json:"page_size"`   // Page size
	TotalPages int     `json:"total_pages"` // Total pages
}

// FileHeader the file header
type FileHeader struct {
	*multipart.FileHeader
}
