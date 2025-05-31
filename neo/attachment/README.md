# Attachment Package

A comprehensive file upload package for Go that supports chunked uploads, file format validation, compression, and multiple storage backends.

## Features

- **Multiple Storage Backends**: Local filesystem and S3-compatible storage
- **Chunked Upload Support**: Handle large files with standard HTTP Content-Range headers
- **File Compression**:
  - Gzip compression for any file type
  - Image compression with configurable size limits
- **File Validation**:
  - File size limits
  - MIME type and extension validation
  - Wildcard pattern support (e.g., `image/*`, `text/*`)
- **Flexible File Organization**: Hierarchical storage with user/chat/assistant organization
- **Multiple Read Methods**: Stream, bytes, and base64 encoding
- **Global Manager Registry**: Support for registering and accessing managers globally

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
    // Create a manager
    manager, err := attachment.New(attachment.ManagerOption{
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
        UserID: "user123",
        ChatID: "chat456",
    }

    file, err := manager.Upload(context.Background(), fileHeader, strings.NewReader(content), option)
    if err != nil {
        panic(err)
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
// Register managers
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
```

## File Organization

Files are organized in a hierarchical structure:

```
attachments/
├── 20240101/           # Date (YYYYMMDD)
│   └── user123/        # User ID (optional)
│       └── chat456/    # Chat ID (optional)
│           └── assistant789/  # Assistant ID (optional)
│               └── ab/        # First 2 chars of hash
│                   └── cd/    # Next 2 chars of hash
│                       └── abcdef12.txt  # Hash + extension
```

The file ID generation includes:

- Date prefix for organization
- User/Chat/Assistant IDs for access control
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
- `UserID`, `ChatID`, `AssistantID`: Organization IDs

#### `File`

Uploaded file information:

- `ID`: Unique file identifier
- `Filename`: Original filename
- `ContentType`: MIME type
- `Bytes`: File size
- `CreatedAt`: Upload timestamp

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
- **Access Control**: User/Chat/Assistant-based file organization

## License

This package is part of the Yao project and follows the same license terms.
