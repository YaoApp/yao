# Attachment Package

A comprehensive file upload package for Go that supports chunked uploads, file format validation, compression, and multiple storage backends.

## Features

- **Multiple Storage Backends**: Local filesystem and S3-compatible storage
- **Chunked Upload Support**: Handle large files with standard HTTP Content-Range headers
- **File Deduplication**: Content-based fingerprinting to avoid duplicate uploads
- **File Compression**:
  - Gzip compression for any file type
  - Image compression with configurable size limits
- **File Validation**:
  - File size limits
  - MIME type and extension validation
  - Wildcard pattern support (e.g., `image/*`, `text/*`)
- **Flexible File Organization**: Hierarchical storage with multi-level group organization
- **Multiple Read Methods**: Stream, bytes, and base64 encoding
- **Global Manager Registry**: Support for registering and accessing managers globally
- **Upload Status Tracking**: Track upload progress with status field
- **Content Synchronization**: Support for synchronized uploads with Content-Sync header

## Installation

```bash
go get github.com/yaoapp/yao/neo/attachment
```

## Quick Start

### Basic Usage

```go
package main

import (
    "context"
    "strings"
    "mime/multipart"
    "github.com/yaoapp/yao/neo/attachment"
)

func main() {
    // Create a manager with default settings
    manager, err := attachment.RegisterDefault("uploads")
    if err != nil {
        panic(err)
    }

    // Or create a custom manager
    customManager, err := attachment.New(attachment.ManagerOption{
        Driver:       "local",
        MaxSize:      "20M",
        ChunkSize:    "2M",
        AllowedTypes: []string{"text/*", "image/*", ".pdf"},
        Options: map[string]interface{}{
            "path": "/var/uploads",
        },
    })
    if err != nil {
        panic(err)
    }

    // Upload a file
    content := "Hello, World!"
    fileHeader := &attachment.FileHeader{
        FileHeader: &multipart.FileHeader{
            Filename: "hello.txt",
            Size:     int64(len(content)),
            Header:   make(map[string][]string),
        },
    }
    fileHeader.Header.Set("Content-Type", "text/plain")

    option := attachment.UploadOption{
        Groups:           []string{"user123", "chat456"}, // Multi-level groups (e.g., user, chat, knowledge, etc.)
        OriginalFilename: "my_document.txt", // Preserve original filename
    }

    file, err := manager.Upload(context.Background(), fileHeader, strings.NewReader(content), option)
    if err != nil {
        panic(err)
    }

    // Check upload status
    if file.Status == "uploaded" {
        fmt.Printf("File uploaded successfully: %s\n", file.ID)
    }

    // Read the file back
    data, err := manager.Read(context.Background(), file.ID)
    if err != nil {
        panic(err)
    }

    println(string(data)) // Output: Hello, World!
}
```

### Storage Backends

#### Local Storage

```go
manager, err := attachment.New(attachment.ManagerOption{
    Driver:  "local",
    MaxSize: "20M",
    Options: map[string]interface{}{
        "path":     "/var/uploads",
        "base_url": "https://example.com/files",
    },
})
```

#### S3 Storage

```go
manager, err := attachment.New(attachment.ManagerOption{
    Driver:  "s3",
    MaxSize: "100M",
    Options: map[string]interface{}{
        "endpoint": "https://s3.amazonaws.com",
        "region":   "us-east-1",
        "key":      "your-access-key",
        "secret":   "your-secret-key",
        "bucket":   "your-bucket-name",
        "prefix":   "attachments/",
    },
})
```

### Chunked Upload

For large files, you can upload in chunks using standard HTTP Content-Range headers:

```go
// Upload chunks
totalSize := int64(1024000) // 1MB file
chunkSize := int64(1024)    // 1KB chunks
uid := "unique-file-id-123"

for start := int64(0); start < totalSize; start += chunkSize {
    end := start + chunkSize - 1
    if end >= totalSize {
        end = totalSize - 1
    }

    chunkData := make([]byte, end-start+1)
    // ... fill chunkData with actual data ...

    chunkHeader := &attachment.FileHeader{
        FileHeader: &multipart.FileHeader{
            Filename: "large_file.zip",
            Size:     end - start + 1,
            Header:   make(map[string][]string),
        },
    }
    chunkHeader.Header.Set("Content-Type", "application/zip")
    chunkHeader.Header.Set("Content-Range", fmt.Sprintf("bytes %d-%d/%d", start, end, totalSize))
    chunkHeader.Header.Set("Content-Uid", uid)

    file, err := manager.Upload(ctx, chunkHeader, bytes.NewReader(chunkData), option)
    if err != nil {
        return err
    }

    // File is complete when the last chunk is uploaded
    if chunkHeader.Complete() {
        fmt.Printf("Upload complete: %s\n", file.ID)
        break
    }
}
```

### Compression

#### Gzip Compression

```go
option := attachment.UploadOption{
    Gzip: true, // Enable gzip compression
}

file, err := manager.Upload(ctx, fileHeader, reader, option)
```

#### Image Compression

```go
option := attachment.UploadOption{
    CompressImage: true,
    CompressSize:  1920, // Max dimension in pixels (default: 1920)
}

file, err := manager.Upload(ctx, imageHeader, imageReader, option)
```

### Multi-level Groups

The `Groups` field supports hierarchical file organization:

```go
// Single level grouping
option := attachment.UploadOption{
    Groups: []string{"users"},
}

// Multi-level grouping
option := attachment.UploadOption{
    Groups: []string{"users", "user123", "chats", "chat456"},
}

// Knowledge base organization
option := attachment.UploadOption{
    Groups: []string{"knowledge", "documents", "technical"},
}
```

This creates nested directory structures for better organization and access control.

### File Validation

#### Size Limits

```go
manager, err := attachment.New(attachment.ManagerOption{
    MaxSize: "20M", // Maximum file size
    // Supports: B, K, M, G (e.g., "1024B", "2K", "10M", "1G")
})
```

#### Type Validation

```go
manager, err := attachment.New(attachment.ManagerOption{
    AllowedTypes: []string{
        "text/*",           // All text types
        "image/*",          // All image types
        "application/pdf",  // Specific MIME type
        ".txt",            // File extension
        ".jpg",            // File extension
    },
})
```

### Reading Files

#### Stream Reading

```go
response, err := manager.Download(ctx, fileID)
if err != nil {
    return err
}
defer response.Reader.Close()

// Use response.Reader as io.ReadCloser
// response.ContentType contains the MIME type
// response.Extension contains the file extension
```

#### Read as Bytes

```go
data, err := manager.Read(ctx, fileID)
if err != nil {
    return err
}
// data is []byte
```

#### Read as Base64

```go
base64Data, err := manager.ReadBase64(ctx, fileID)
if err != nil {
    return err
}
// base64Data is string
```

### Global Managers

You can register managers globally for easy access:

```go
// Register default manager with sensible defaults
attachment.RegisterDefault("main")

// Register custom managers
attachment.Register("local", "local", attachment.ManagerOption{
    Driver: "local",
    Options: map[string]interface{}{
        "path": "/var/uploads",
    },
})

attachment.Register("s3", "s3", attachment.ManagerOption{
    Driver: "s3",
    Options: map[string]interface{}{
        "bucket": "my-bucket",
        "key":    "access-key",
        "secret": "secret-key",
    },
})

// Use global managers
localManager := attachment.Managers["local"]
s3Manager := attachment.Managers["s3"]
defaultManager := attachment.Managers["main"]
```

## File Organization

Files are organized in a hierarchical structure:

```
attachments/
├── 20240101/           # Date (YYYYMMDD)
│   └── user123/        # First level group (optional)
│       └── chat456/    # Second level group (optional)
│           └── knowledge/  # Additional group levels (optional)
│               └── ab/     # First 2 chars of hash
│                   └── cd/ # Next 2 chars of hash
│                       └── abcdef12.txt  # Hash + extension
```

The file ID generation includes:

- Date prefix for organization
- Multi-level groups for access control and organization
- Content hash for deduplication
- Original file extension

## API Reference

### Manager

#### `New(option ManagerOption) (*Manager, error)`

Creates a new attachment manager.

#### `Register(name string, driver string, option ManagerOption) (*Manager, error)`

Registers a global attachment manager.

#### `Upload(ctx context.Context, fileheader *FileHeader, reader io.Reader, option UploadOption) (*File, error)`

Uploads a file (supports chunked upload).

#### `Download(ctx context.Context, fileID string) (*FileResponse, error)`

Downloads a file as a stream.

#### `Read(ctx context.Context, fileID string) ([]byte, error)`

Reads a file as bytes.

#### `ReadBase64(ctx context.Context, fileID string) (string, error)`

Reads a file as base64 encoded string.

### Storage Interface

All storage backends implement the following interface:

```go
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
```

### Types

#### `ManagerOption`

Configuration for creating a manager:

- `Driver`: "local" or "s3"
- `MaxSize`: Maximum file size (e.g., "20M")
- `ChunkSize`: Chunk size for uploads (e.g., "2M")
- `AllowedTypes`: Array of allowed MIME types/extensions
- `Options`: Driver-specific options

#### `UploadOption`

Options for file upload:

- `CompressImage`: Enable image compression
- `CompressSize`: Maximum image dimension (default: 1920)
- `Gzip`: Enable gzip compression
- `Groups`: Multi-level group identifiers for hierarchical file organization (e.g., []string{"user123", "chat456", "knowledge"})
- `OriginalFilename`: Original filename to preserve (avoids encoding issues)

#### `File`

Uploaded file information:

- `ID`: Unique file identifier
- `Filename`: Original filename
- `ContentType`: MIME type
- `Bytes`: File size
- `CreatedAt`: Upload timestamp
- `Status`: Upload status ("uploading", "uploaded", "indexing", "indexed", "upload_failed", "index_failed")

#### `FileResponse`

Download response:

- `Reader`: io.ReadCloser for file content
- `ContentType`: MIME type
- `Extension`: File extension

## Chunked Upload Details

The package supports chunked uploads using standard HTTP headers:

- `Content-Range`: Specifies byte range (e.g., "bytes 0-1023/2048")
- `Content-Uid`: Unique identifier for the file being uploaded

### Chunk Index Calculation

The package uses a standard chunk size (1024 bytes by default) to calculate chunk indices consistently. This ensures proper chunk ordering during merge operations.

### Content Type Preservation

For chunked uploads, the content type is preserved from the first chunk and applied to the final merged file, ensuring proper MIME type handling across all storage backends.

## Error Handling

The package returns descriptive errors for common issues:

- File size exceeds limit
- Unsupported file type
- Storage backend errors
- Invalid chunk information
- Missing required configuration

## Testing

Run the tests:

```bash
# Run all tests
go test ./...

# Run with S3 credentials (optional)
export S3_ACCESS_KEY="your-key"
export S3_SECRET_KEY="your-secret"
export S3_BUCKET="your-bucket"
export S3_API="https://your-s3-endpoint"
go test ./...
```

The package includes comprehensive tests for:

- Basic file upload/download
- Chunked uploads with content type preservation
- Compression (gzip and image)
- File validation (size, type, wildcards)
- Multiple storage backends (local and S3)
- Error handling and edge cases

### Test Coverage

- **Manager Tests**: Upload, download, validation, compression
- **Local Storage Tests**: File operations, chunked uploads, directory management
- **S3 Storage Tests**: S3 operations, chunked uploads, presigned URLs (requires credentials)

## Performance Considerations

- **Chunked Uploads**: Use appropriate chunk sizes (1-5MB) for optimal performance
- **Image Compression**: Automatically resizes large images to reduce storage costs
- **Gzip Compression**: Reduces storage size for text-based files
- **Content Type Detection**: Efficient MIME type detection and preservation

## Security Features

- **File Type Validation**: Prevents upload of unauthorized file types
- **Size Limits**: Configurable file size restrictions
- **Path Sanitization**: Secure file path generation
- **Access Control**: Multi-level hierarchical file organization

## License

This package is part of the Yao project and follows the same license terms.

### File Deduplication with Fingerprints

The package supports file deduplication using content fingerprints:

```go
// Set a content fingerprint to enable deduplication
fileHeader := &attachment.FileHeader{
    FileHeader: &multipart.FileHeader{
        Filename: "document.pdf",
        Size:     fileSize,
        Header:   make(map[string][]string),
    },
}
fileHeader.Header.Set("Content-Type", "application/pdf")
fileHeader.Header.Set("Content-Fingerprint", "sha256:abcdef123456") // Content-based hash

file, err := manager.Upload(ctx, fileHeader, reader, option)
```

### Content Synchronization

For synchronized uploads across multiple clients:

```go
// Enable content synchronization
fileHeader.Header.Set("Content-Sync", "true")

// Each client can upload the same content with the same fingerprint
// The system will deduplicate based on the content fingerprint
```

### Chunked Upload with Enhanced Headers

For large files, you can upload in chunks using standard HTTP Content-Range headers with additional metadata:

```go
// Upload chunks with unique identifier and fingerprint
totalSize := int64(1024000) // 1MB file
chunkSize := int64(1024)    // 1KB chunks
uid := "unique-file-id-123"
fingerprint := "sha256:content-hash-here"

for start := int64(0); start < totalSize; start += chunkSize {
    end := start + chunkSize - 1
    if end >= totalSize {
        end = totalSize - 1
    }

    chunkData := make([]byte, end-start+1)
    // ... fill chunkData with actual data ...

    chunkHeader := &attachment.FileHeader{
        FileHeader: &multipart.FileHeader{
            Filename: "large_file.zip",
            Size:     end - start + 1,
            Header:   make(map[string][]string),
        },
    }
    chunkHeader.Header.Set("Content-Type", "application/zip")
    chunkHeader.Header.Set("Content-Range", fmt.Sprintf("bytes %d-%d/%d", start, end, totalSize))
    chunkHeader.Header.Set("Content-Uid", uid)
    chunkHeader.Header.Set("Content-Fingerprint", fingerprint)
    chunkHeader.Header.Set("Content-Sync", "true") // Enable synchronization

    option := attachment.UploadOption{
        Groups:           []string{"user123", "chat456"}, // Multi-level groups
        OriginalFilename: "my_large_file.zip", // Preserve original name
    }

    file, err := manager.Upload(ctx, chunkHeader, bytes.NewReader(chunkData), option)
    if err != nil {
        return err
    }

    // Check if upload is complete
    if file.Status == "uploaded" {
        fmt.Printf("Upload complete: %s\n", file.ID)
        break
    } else if file.Status == "uploading" {
        fmt.Printf("Chunk uploaded, progress: %d/%d\n", chunkHeader.GetChunkSize(), chunkHeader.GetTotalSize())
    }
}
```

### FileHeader Methods

The `FileHeader` type provides several utility methods:

```go
// Get unique identifier for chunked uploads
uid := fileHeader.UID()

// Get content fingerprint for deduplication
fingerprint := fileHeader.Fingerprint()

// Get byte range for chunked uploads
rangeHeader := fileHeader.Range()

// Check if synchronization is enabled
isSync := fileHeader.Sync()

// Check if this is a chunked upload
isChunk := fileHeader.IsChunk()

// Check if upload is complete (for chunked uploads)
isComplete := fileHeader.Complete()

// Get detailed chunk information
start, end, total, err := fileHeader.GetChunkInfo()

// Get total file size (for chunked uploads)
totalSize := fileHeader.GetTotalSize()

// Get current chunk size
chunkSize := fileHeader.GetChunkSize()
```

## File Headers and Metadata

The package supports several HTTP headers for enhanced functionality:

- `Content-Range`: Standard HTTP range header for chunked uploads (e.g., "bytes 0-1023/2048")
- `Content-Uid`: Unique identifier for file uploads (for deduplication and tracking)
- `Content-Fingerprint`: Content-based hash for deduplication (e.g., "sha256:abc123")
- `Content-Sync`: Enable synchronized uploads across multiple clients ("true"/"false")

### Header Processing

When processing uploads, headers can be extracted from both HTTP request headers and multipart file headers:

```go
// Extract headers from HTTP request and file headers
header := attachment.GetHeader(requestHeader, fileHeader, fileSize)

// The resulting FileHeader will contain merged headers from both sources
uid := header.UID()
fingerprint := header.Fingerprint()
isSync := header.Sync()
```

## Upload Status Tracking

Files have a status field that tracks the upload lifecycle:

- `"uploading"`: File upload is in progress (for chunked uploads)
- `"uploaded"`: File has been successfully uploaded
- `"indexing"`: File is being processed for search indexing
- `"indexed"`: File has been indexed and is fully processed
- `"upload_failed"`: Upload failed due to an error
- `"index_failed"`: Indexing failed but file is still accessible

```go
file, err := manager.Upload(ctx, fileHeader, reader, option)
if err != nil {
    return err
}

switch file.Status {
case "uploading":
    fmt.Println("Upload in progress...")
case "uploaded":
    fmt.Println("Upload completed successfully")
case "upload_failed":
    fmt.Println("Upload failed")
}
```

## Text Content Storage

The attachment package supports storing parsed text content extracted from files (e.g., from PDFs, Word documents, or image OCR). This is useful for building search indexes or providing text-based previews.

The system automatically maintains two versions of the text content:
- **Full content** (`content`): Complete text, stored as longText (up to 4GB)
- **Preview** (`content_preview`): First 2000 characters, stored as text for quick access

### Saving Parsed Text Content

Use `SaveText` to store the extracted text content. It automatically saves both full content and preview:

```go
// Upload a PDF file
file, err := manager.Upload(ctx, fileHeader, reader, option)
if err != nil {
    return err
}

// Extract text from the PDF (using your preferred library)
parsedText := extractTextFromPDF(file.ID)

// Save the parsed text (automatically saves both full and preview)
err = manager.SaveText(ctx, file.ID, parsedText)
if err != nil {
    return fmt.Errorf("failed to save text content: %w", err)
}
```

### Retrieving Parsed Text Content

Use `GetText` to retrieve text content. By default, it returns the preview for better performance:

```go
// Get preview (first 2000 characters) - Fast, suitable for UI display
preview, err := manager.GetText(ctx, file.ID)
if err != nil {
    return fmt.Errorf("failed to get preview: %w", err)
}

if preview == "" {
    fmt.Println("No text content available for this file")
} else {
    fmt.Printf("Preview (%d characters): %s\n", len(preview), preview)
}

// Get full content - Use only when complete text is needed (e.g., for indexing)
fullText, err := manager.GetText(ctx, file.ID, true)
if err != nil {
    return fmt.Errorf("failed to get full text: %w", err)
}

fmt.Printf("Full content (%d characters)\n", len(fullText))
```

### Performance Optimization

The text content fields are optimized for different use cases:

| Field | Size Limit | Use Case | Performance |
|-------|------------|----------|-------------|
| `content_preview` | 2000 chars | Quick preview, UI display, snippets | ⚡ Very Fast |
| `content` | 4GB | Full text search, complete content | 🐌 Slow for large files |

**Best Practices:**
1. Use preview by default: `GetText(ctx, fileID)`
2. Only request full content when necessary: `GetText(ctx, fileID, true)`
3. Both fields are excluded from `List()` by default for optimal performance
4. Preview uses character (rune) count, not bytes, for proper UTF-8 handling

### Example: Complete Text Processing Workflow

```go
// 1. Upload file
file, err := manager.Upload(ctx, fileHeader, reader, option)
if err != nil {
    return err
}

// 2. Process file based on content type
var parsedText string
switch {
case strings.HasPrefix(file.ContentType, "image/"):
    // Use OCR to extract text from image
    parsedText, err = performOCR(file.ID)
    
case file.ContentType == "application/pdf":
    // Extract text from PDF
    parsedText, err = extractPDFText(file.ID)
    
case strings.Contains(file.ContentType, "wordprocessingml"):
    // Extract text from Word document
    parsedText, err = extractWordText(file.ID)
}

if err != nil {
    return fmt.Errorf("failed to extract text: %w", err)
}

// 3. Save the extracted text
if parsedText != "" {
    err = manager.SaveText(ctx, file.ID, parsedText)
    if err != nil {
        return fmt.Errorf("failed to save text: %w", err)
    }
}

// 4. Later, retrieve the text for search or display
savedText, err := manager.GetText(ctx, file.ID)
if err != nil {
    return err
}

fmt.Printf("Retrieved text: %s\n", savedText)
```

### Text Content Features

- **Dual Storage**: Automatically maintains both full content and preview (2000 chars)
- **Size Limits**: 
  - Preview: 2000 characters (text type)
  - Full content: Up to 4GB (longText type)
- **Smart Retrieval**: Returns preview by default, full content on demand
- **Update**: Text content can be updated at any time using `SaveText`
- **Clear**: Set text to empty string to clear both fields
- **UTF-8 Safe**: Preview uses character (rune) count, not bytes, ensuring proper multi-byte character handling
- **Performance**: Both `content` and `content_preview` fields are excluded by default in `List()` and `Info()` operations to avoid loading text data. Use `GetText()` to explicitly retrieve text content when needed

#### `RegisterDefault(name string) (*Manager, error)`

Registers a default attachment manager with sensible defaults for common file types.
