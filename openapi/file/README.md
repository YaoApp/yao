# File Management API

This document describes the RESTful API for managing file uploads, downloads, and file operations in Yao applications.

## Base URL

All endpoints are prefixed with the configured base URL followed by `/file` (e.g., `/v1/file`).

## Authentication

All endpoints require OAuth authentication via the configured OAuth provider.

## File Operations

The File Management API provides comprehensive file handling capabilities including:

- **File Upload** - Single and chunked file uploads with compression support
- **File Listing** - Paginated file listing with filtering and sorting
- **File Retrieval** - Get file metadata and download file content with accurate headers
- **File Management** - Check existence and delete files
- **Storage Flexibility** - Support for local and cloud storage backends
- **Optimized Content Delivery** - Direct content reading with database-driven metadata headers

## Endpoints

### File Upload

Upload files with support for chunked uploads, compression, and metadata.

```
POST /file/{uploaderID}
```

**Parameters:**

- `uploaderID` (path): Uploader/manager identifier

**Form Data:**

- `file` (required): The file to upload
- `original_filename` (optional): Original filename (defaults to uploaded filename)
- `path` (optional): User-specified file path (defaults to original_filename)
- `groups` (optional): Comma-separated list of groups for directory organization
- `client_id` (optional): Client identifier
- `openid` (optional): OpenID identifier
- `gzip` (optional): Enable gzip compression ("true"/"false")
- `compress_image` (optional): Enable image compression ("true"/"false")
- `compress_size` (optional): Target compression size in bytes

**Chunked Upload Headers:**

- `Content-Range`: Byte range for chunk (e.g., "bytes 0-1023/2048")
- `Content-Sync`: Synchronization header for chunks
- `Content-Uid`: Unique identifier for chunked upload session

**Example:**

```bash
# Simple file upload
curl -X POST "/v1/file/default" \
  -H "Authorization: Bearer {token}" \
  -F "file=@document.pdf" \
  -F "path=documents/reports/quarterly-report.pdf" \
  -F "groups=documents,reports" \
  -F "client_id=app123" \
  -F "gzip=true"

# Chunked upload (first chunk)
curl -X POST "/v1/file/default" \
  -H "Authorization: Bearer {token}" \
  -H "Content-Range: bytes 0-1023/2048" \
  -H "Content-Sync: chunk-upload" \
  -H "Content-Uid: unique-upload-id" \
  -F "file=@chunk1.bin"
```

**Response:**

```json
{
  "file_id": "a1b2c3d4e5f6789012345678901234567890abcd",
  "user_path": "documents/reports/quarterly-report.pdf",
  "path": "documents/reports/quarterly-report.pdf",
  "filename": "quarterly-report.pdf",
  "content_type": "application/pdf",
  "bytes": 2048576,
  "gzip": true,
  "status": "completed",
  "created_at": 1640995200
}
```

### List Files

List files with pagination, filtering, and sorting capabilities.

```
GET /file/{uploaderID}?page={page}&page_size={page_size}&status={status}&content_type={content_type}&name={name}&order_by={order_by}&select={select}
```

**Parameters:**

- `uploaderID` (path): Uploader/manager identifier

**Query Parameters:**

- `page` (optional): Page number (default: 1)
- `page_size` (optional): Items per page (default: 20, max: 100)
- `status` (optional): Filter by file status
- `content_type` (optional): Filter by content type
- `name` (optional): Filter by filename (supports wildcard matching)
- `order_by` (optional): Sort field and direction (default: "created_at desc")
- `select` (optional): Comma-separated list of fields to return

**Example:**

```bash
# List files with pagination
curl -X GET "/v1/file/default?page=1&page_size=10" \
  -H "Authorization: Bearer {token}"

# List files with filters
curl -X GET "/v1/file/default?status=completed&content_type=image/jpeg&name=photo*" \
  -H "Authorization: Bearer {token}"

# List with custom ordering and field selection
curl -X GET "/v1/file/default?order_by=bytes desc&select=file_id,filename,bytes" \
  -H "Authorization: Bearer {token}"
```

**Response:**

```json
{
  "files": [
    {
      "file_id": "a1b2c3d4e5f6789012345678901234567890abcd",
      "user_path": "documents/reports/quarterly-report.pdf",
      "path": "documents/reports/quarterly-report.pdf",
      "filename": "quarterly-report.pdf",
      "content_type": "application/pdf",
      "bytes": 2048576,
      "gzip": true,
      "status": "completed",
      "created_at": 1640995200
    }
  ],
  "total": 150,
  "page": 1,
  "page_size": 20,
  "total_pages": 8
}
```

### Retrieve File Information

Get detailed metadata for a specific file.

```
GET /file/{uploaderID}/{fileID}
```

**Parameters:**

- `uploaderID` (path): Uploader/manager identifier
- `fileID` (path): File identifier (URL-encoded)

**Example:**

```bash
curl -X GET "/v1/file/default/a1b2c3d4e5f6789012345678901234567890abcd" \
  -H "Authorization: Bearer {token}"
```

**Response:**

```json
{
  "file_id": "a1b2c3d4e5f6789012345678901234567890abcd",
  "user_path": "documents/reports/quarterly-report.pdf",
  "path": "documents/reports/quarterly-report.pdf",
  "filename": "quarterly-report.pdf",
  "content_type": "application/pdf",
  "bytes": 2048576,
  "gzip": true,
  "status": "completed",
  "created_at": 1640995200,
  "uploader": "default",
  "client_id": "app123",
  "openid": "user456",
  "groups": ["documents", "reports"]
}
```

### Download File Content

Download the actual file content directly from storage.

```
GET /file/{uploaderID}/{fileID}/content
```

**Parameters:**

- `uploaderID` (path): Uploader/manager identifier
- `fileID` (path): File identifier (URL-encoded)

**Example:**

```bash
curl -X GET "/v1/file/default/a1b2c3d4e5f6789012345678901234567890abcd/content" \
  -H "Authorization: Bearer {token}" \
  --output downloaded-file.pdf
```

**Response:**

Returns the raw file content with metadata-driven headers:

```
Content-Type: application/pdf
Content-Disposition: attachment; filename="quarterly-report.pdf"
Content-Length: 2048576
```

**Implementation Details:**

- File metadata is retrieved from the database to set accurate response headers
- Content is read directly using the storage manager's Read method
- Headers include the actual filename, precise content type, and content length
- Automatic decompression is handled transparently for gzipped files

### Check File Existence

Check if a file exists without downloading it.

```
GET /file/{uploaderID}/{fileID}/exists
```

**Parameters:**

- `uploaderID` (path): Uploader/manager identifier
- `fileID` (path): File identifier (URL-encoded)

**Example:**

```bash
curl -X GET "/v1/file/default/a1b2c3d4e5f6789012345678901234567890abcd/exists" \
  -H "Authorization: Bearer {token}"
```

**Response:**

```json
{
  "exists": true,
  "file_id": "a1b2c3d4e5f6789012345678901234567890abcd"
}
```

### Delete File

Delete a file and its metadata.

```
DELETE /file/{uploaderID}/{fileID}
```

**Parameters:**

- `uploaderID` (path): Uploader/manager identifier
- `fileID` (path): File identifier (URL-encoded)

**Example:**

```bash
curl -X DELETE "/v1/file/default/a1b2c3d4e5f6789012345678901234567890abcd" \
  -H "Authorization: Bearer {token}"
```

**Response:**

```json
{
  "message": "File deleted successfully",
  "file_id": "a1b2c3d4e5f6789012345678901234567890abcd"
}
```

## File ID System

The File Management API uses a secure file ID system:

- **File ID**: A URL-safe MD5 hash that serves as a public alias for the file
- **Storage Path**: The actual file system path where the file is stored
- **User Path**: The original path specified by the user for organization

This system provides security by hiding internal storage paths while maintaining a consistent public API.

## Storage Backends

The API supports multiple storage backends:

- **Local Storage**: Files stored on the local file system
- **S3 Storage**: Files stored in Amazon S3 or S3-compatible services
- **Custom Storage**: Extensible storage interface for custom implementations

## Compression Support

### Gzip Compression

Files can be automatically compressed using gzip:

- Set `gzip=true` in upload form data
- Compressed files are automatically decompressed when downloaded
- Storage path includes `.gz` extension for compressed files
- File ID remains unchanged (hash of uncompressed path)

### Image Compression

Images can be compressed for storage optimization:

- Set `compress_image=true` in upload form data
- Optionally specify `compress_size` for target size in bytes
- Maintains image quality while reducing file size

## Chunked Upload

For large files, use chunked upload for better reliability:

1. **Split file into chunks** (typically 1MB each)
2. **Upload each chunk** with appropriate headers:
   - `Content-Range`: Byte range of the chunk
   - `Content-Sync`: Set to "chunk-upload"
   - `Content-Uid`: Unique identifier for the upload session
3. **Final chunk** triggers automatic merge and file completion

**Example Chunked Upload:**

```bash
# Upload chunk 1
curl -X POST "/v1/file/default" \
  -H "Authorization: Bearer {token}" \
  -H "Content-Range: bytes 0-1048575/3145728" \
  -H "Content-Sync: chunk-upload" \
  -H "Content-Uid: upload-session-123" \
  -F "file=@chunk1.bin"

# Upload chunk 2
curl -X POST "/v1/file/default" \
  -H "Authorization: Bearer {token}" \
  -H "Content-Range: bytes 1048576-2097151/3145728" \
  -H "Content-Sync: chunk-upload" \
  -H "Content-Uid: upload-session-123" \
  -F "file=@chunk2.bin"

# Upload final chunk (triggers merge)
curl -X POST "/v1/file/default" \
  -H "Authorization: Bearer {token}" \
  -H "Content-Range: bytes 2097152-3145727/3145728" \
  -H "Content-Sync: chunk-upload" \
  -H "Content-Uid: upload-session-123" \
  -F "file=@chunk3.bin"
```

## Error Responses

All endpoints return standardized error responses:

```json
{
  "error": "invalid_request",
  "error_description": "File ID is required"
}
```

**Common HTTP Status Codes:**

- `200` - Success
- `400` - Bad Request (invalid parameters, missing file)
- `401` - Unauthorized (authentication required)
- `404` - Not Found (uploader or file not found)
- `500` - Internal Server Error (upload/storage failure)

**Common Error Scenarios:**

- `Uploader not found` - Invalid uploader ID
- `File is required` - No file provided in upload
- `File not found` - File ID does not exist
- `Failed to upload file` - Storage or processing error

## Example Workflows

### Simple File Upload and Download

1. **Upload a file:**

```bash
curl -X POST "/v1/file/default" \
  -H "Authorization: Bearer {token}" \
  -F "file=@document.pdf" \
  -F "path=documents/important-doc.pdf" \
  -F "groups=documents"
```

2. **List files to find the uploaded file:**

```bash
curl -X GET "/v1/file/default?name=important-doc*" \
  -H "Authorization: Bearer {token}"
```

3. **Download the file:**

```bash
curl -X GET "/v1/file/default/{file_id}/content" \
  -H "Authorization: Bearer {token}" \
  --output downloaded-document.pdf
```

### Large File Chunked Upload

1. **Split large file into chunks:**

```bash
split -b 1048576 largefile.zip chunk_
```

2. **Upload chunks sequentially:**

```bash
#!/bin/bash
TOTAL_SIZE=$(stat -c%s largefile.zip)
CHUNK_SIZE=1048576
UPLOAD_ID="upload-$(date +%s)"

for i in chunk_*; do
  START=$((CHUNK_SIZE * (${i#chunk_} - 1)))
  END=$((START + $(stat -c%s $i) - 1))

  curl -X POST "/v1/file/default" \
    -H "Authorization: Bearer {token}" \
    -H "Content-Range: bytes ${START}-${END}/${TOTAL_SIZE}" \
    -H "Content-Sync: chunk-upload" \
    -H "Content-Uid: ${UPLOAD_ID}" \
    -F "file=@${i}"
done
```

### File Management with Metadata

1. **Upload with comprehensive metadata:**

```bash
curl -X POST "/v1/file/default" \
  -H "Authorization: Bearer {token}" \
  -F "file=@report.pdf" \
  -F "path=reports/2024/quarterly-report.pdf" \
  -F "groups=reports,2024,quarterly" \
  -F "client_id=dashboard-app" \
  -F "openid=user123" \
  -F "gzip=true"
```

2. **List files with filters:**

```bash
curl -X GET "/v1/file/default?status=completed&content_type=application/pdf&order_by=created_at desc" \
  -H "Authorization: Bearer {token}"
```

3. **Get detailed file information:**

```bash
curl -X GET "/v1/file/default/{file_id}" \
  -H "Authorization: Bearer {token}"
```

4. **Clean up old files:**

```bash
curl -X DELETE "/v1/file/default/{file_id}" \
  -H "Authorization: Bearer {token}"
```

## Performance Optimizations

### Content Delivery Optimization

The File Management API implements several performance optimizations for efficient content delivery:

- **Direct Content Reading**: The `/content` endpoint uses direct file reading instead of streaming, reducing overhead
- **Database-Driven Headers**: Response headers are generated from accurate database metadata rather than file system inspection
- **Optimized Header Information**: Includes precise content length, actual filename, and accurate MIME types
- **Transparent Decompression**: Gzipped files are automatically decompressed without additional processing overhead

### Implementation Benefits

- **Reduced Latency**: Direct content reading eliminates streaming overhead
- **Accurate Metadata**: Headers reflect database-stored information for consistency
- **Better Caching**: Content-Length headers improve browser and proxy caching behavior
- **Resource Efficiency**: Single database query for metadata followed by direct file access

## Security Considerations

### Access Control

- All endpoints require valid OAuth authentication
- File access is scoped to the uploader/manager level
- File IDs are cryptographically secure (MD5 hash)

### File Validation

- Content type validation based on file headers
- File size limits enforced by uploader configuration
- Allowed file type restrictions per uploader

### Path Security

- User paths are normalized and validated
- Internal storage paths are hidden from public API
- Directory traversal attacks prevented

This File Management API provides a robust, secure, and scalable solution for handling file operations in Yao applications.
