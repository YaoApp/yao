package attachment

import (
	"bytes"
	"context"
	"crypto/md5"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"io"
	"mime"
	"mime/multipart"
	"net/http"
	"net/textproto"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/yaoapp/gou/fs"
	"github.com/yaoapp/gou/model"
	"github.com/yaoapp/kun/log"
	"github.com/yaoapp/yao/attachment/local"
	"github.com/yaoapp/yao/attachment/s3"
	"github.com/yaoapp/yao/config"
)

// Ensure Manager implements FileManager interface
var _ FileManager = (*Manager)(nil)

// Managers the managers
var Managers = map[string]*Manager{}
var uploadChunks = sync.Map{}

// UploadChunk is the chunk data
type UploadChunk struct {
	Last        int
	Total       int64
	Chunksize   int64
	TotalChunks int64
	// Cache metadata from first chunk to avoid inconsistencies
	ContentType   string
	Filename      string
	UserPath      string
	CompressImage bool
	CompressSize  int
}

// Parse parses an attachment wrapper string and returns uploader name and file ID
// Format: __<uploader>://<fileID>
// Example: __yao.attachment://ccd472d11feb96e03a3fc468f494045c
// Returns (uploader, fileID, isWrapper)
func Parse(value string) (string, string, bool) {
	if !strings.HasPrefix(value, "__") {
		return "", value, false
	}

	// Exclude common protocols (ftp, http, https, etc.)
	excludedProtocols := []string{"__ftp://", "__http://", "__https://", "__ws://", "__wss://"}
	for _, protocol := range excludedProtocols {
		if strings.HasPrefix(value, protocol) {
			return "", value, false
		}
	}

	// Split by ://
	parts := strings.SplitN(value, "://", 2)
	if len(parts) != 2 {
		return "", value, false
	}

	uploader := parts[0] // Keep the __ prefix as it's part of the manager name
	fileID := parts[1]

	return uploader, fileID, true
}

// Base64 processes a wrapper value and converts it to Base64 if it's an attachment wrapper
// If the value is not a wrapper, it returns the original value
// Special case: if value looks like a file path, it will try to read from fs data
// Optional parameter dataURI: if true, returns data URI format (data:image/png;base64,...)
func Base64(ctx context.Context, value string, dataURI ...bool) string {
	useDataURI := false
	if len(dataURI) > 0 {
		useDataURI = dataURI[0]
	}

	uploader, fileID, isWrapper := Parse(value)
	if !isWrapper {
		// Try to read as file path from fs data
		if base64Data := readFilePathAsBase64(value, useDataURI); base64Data != "" {
			return base64Data
		}
		return value
	}

	// Get the manager
	manager, exists := Managers[uploader]
	if !exists {
		return value
	}

	// Get file info to determine content type
	var contentType string
	if useDataURI {
		fileInfo, err := manager.Info(ctx, fileID)
		if err == nil && fileInfo != nil {
			contentType = fileInfo.ContentType
		}
	}

	// Read the file as Base64
	base64Data, err := manager.ReadBase64(ctx, fileID)
	if err != nil {
		return value
	}

	// Return with data URI prefix if requested
	if useDataURI && contentType != "" {
		return fmt.Sprintf("data:%s;base64,%s", contentType, base64Data)
	}

	return base64Data
}

// readFilePathAsBase64 reads a file from fs data and returns Base64 encoded content
// Returns empty string if file doesn't exist or can't be read
// If dataURI is true, returns data URI format with mime type detection
func readFilePathAsBase64(path string, dataURI bool) string {
	// Check if path looks like a file path (contains / or \)
	if !strings.Contains(path, "/") && !strings.Contains(path, "\\") {
		return ""
	}

	// Try to get fs data
	dataFS, err := fs.Get("data")
	if err != nil || dataFS == nil {
		return ""
	}

	// Check if file exists
	exists, err := dataFS.Exists(path)
	if err != nil || !exists {
		return ""
	}

	// Read file content
	content, err := dataFS.ReadFile(path)
	if err != nil {
		return ""
	}

	// Encode to Base64
	base64Str := base64.StdEncoding.EncodeToString(content)

	// Return with data URI prefix if requested
	if dataURI {
		// Detect content type from file extension or content
		contentType := detectContentType(path, content)
		if contentType != "" {
			return fmt.Sprintf("data:%s;base64,%s", contentType, base64Str)
		}
	}

	return base64Str
}

// detectContentType detects the MIME type from file path and content
func detectContentType(path string, content []byte) string {
	// First try to get from file extension
	ext := filepath.Ext(path)
	if ext != "" {
		mimeType := mime.TypeByExtension(ext)
		if mimeType != "" {
			return mimeType
		}
	}

	// Fallback to detecting from content (first 512 bytes)
	if len(content) > 0 {
		detectSize := len(content)
		if detectSize > 512 {
			detectSize = 512
		}
		return http.DetectContentType(content[:detectSize])
	}

	return ""
}

// GetHeader gets the header from the file header and request header
func GetHeader(requestHeader http.Header, fileHeader textproto.MIMEHeader, size int64) *FileHeader {

	// Convert the header to a FileHeader
	header := &FileHeader{FileHeader: &multipart.FileHeader{Header: make(map[string][]string), Size: size}}

	for key, values := range fileHeader {
		for _, value := range values {
			header.Header.Set(key, value)
		}
	}

	// Set Content-Sync, Content-Uid, Content-Range
	if requestHeader.Get("Content-Sync") != "" {
		header.Header.Set("Content-Sync", requestHeader.Get("Content-Sync"))
	}

	// Set Content-Uid
	if requestHeader.Get("Content-Uid") != "" {
		header.Header.Set("Content-Uid", requestHeader.Get("Content-Uid"))
	}

	// Set Content-Range
	if requestHeader.Get("Content-Range") != "" {
		header.Header.Set("Content-Range", requestHeader.Get("Content-Range"))
	}

	return header
}

// Register registers a global attachment manager
func Register(name string, driver string, option ManagerOption) (*Manager, error) {

	// Create a new manager
	manager, err := New(option)
	if err != nil {
		return nil, err
	}

	// Set the manager name
	manager.Name = name

	// Register the manager
	Managers[name] = manager
	return manager, nil
}

// RegisterDefault registers a default attachment manager
func RegisterDefault(name string) (*Manager, error) {

	option := ManagerOption{
		Driver:    "local",
		Options:   map[string]interface{}{"path": filepath.Join(config.Conf.DataRoot, name)},
		MaxSize:   "50M",
		ChunkSize: "2M",
		AllowedTypes: []string{
			"text/*",
			"image/*",
			"video/*",
			"audio/*",
			"application/x-zip-compressed",
			"application/x-tar",
			"application/x-gzip",
			"application/yao",
			"application/zip",
			"application/pdf",
			"application/json",
			"application/vnd.openxmlformats-officedocument.wordprocessingml.document",
			"application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
			"application/vnd.openxmlformats-officedocument.presentationml.presentation",
			"application/vnd.openxmlformats-officedocument.presentationml.slideshow",
			".md",
			".txt",
			".csv",
			".xls",
			".xlsx",
			".ppt",
			".pptx",
			".doc",
			".docx",
			".mdx",
			".m4a",
			".mp3",
			".mp4",
			".wav",
			".webm",
			".yao",
		},
	}
	return Register(name, option.Driver, option)
}

// ReplaceEnv replaces the environment variables in the options
func (option *ManagerOption) ReplaceEnv(root string) {
	if option.Options != nil {
		// Replace the environment variables in the options
		for k, v := range option.Options {
			if iv, ok := v.(string); ok {
				if strings.HasPrefix(iv, "$ENV.") {
					iv = os.ExpandEnv(fmt.Sprintf("${%s}", strings.TrimPrefix(iv, "$ENV.")))
					option.Options[k] = iv
				}

				// Path
				if k == "path" {
					iv = strings.TrimPrefix(iv, "/")
					option.Options[k] = filepath.Join(root, iv)
				}

			}
		}
	}
}

// New creates a new attachment manager
func New(option ManagerOption) (*Manager, error) {
	manager := &Manager{
		ManagerOption: option,
		allowedTypes: allowedType{mapping: make(map[string]bool),
			wildcards: []string{},
		}}

	switch strings.ToLower(option.Driver) {
	case "local":
		storage, err := local.New(option.Options)
		if err != nil {
			return nil, err
		}
		manager.storage = storage

	case "s3":
		storage, err := s3.New(option.Options)
		if err != nil {
			return nil, err
		}
		manager.storage = storage

	default:
		return nil, fmt.Errorf("driver %s does not support", option.Driver)
	}

	// Max size
	if option.MaxSize != "" {
		maxsize, err := getSize(option.MaxSize)
		if err != nil {
			return nil, err
		}
		manager.maxsize = maxsize
	}

	// Chunk size
	if option.ChunkSize != "" {
		chunsize, err := getSize(option.ChunkSize)
		if err != nil {
			return nil, err
		}
		manager.chunsize = chunsize
	}

	// init allowedTypes
	if len(option.AllowedTypes) > 0 {
		for _, t := range option.AllowedTypes {
			t = strings.TrimSpace(t)
			if strings.HasSuffix(t, "*") {
				manager.allowedTypes.wildcards = append(manager.allowedTypes.wildcards, t)
				continue
			}
			manager.allowedTypes.mapping[t] = true
		}
	}

	return manager, nil
}

// LocalPath gets the local path of the file
func (manager Manager) LocalPath(ctx context.Context, fileID string) (string, string, error) {
	// Get the real storage path from database
	storagePath, err := manager.getStoragePathFromDatabase(ctx, fileID)
	if err != nil {
		return "", "", err
	}

	// Call the storage implementation
	return manager.storage.LocalPath(ctx, storagePath)
}

// Upload uploads a file, Content-Sync must be true for chunked upload
func (manager Manager) Upload(ctx context.Context, fileheader *FileHeader, reader io.Reader, option UploadOption) (*File, error) {

	file, err := manager.makeFile(fileheader, option)
	if err != nil {
		return nil, err
	}

	// Handle chunked upload
	if fileheader.IsChunk() {
		start, end, total, err := fileheader.GetChunkInfo()
		if err != nil {
			return nil, fmt.Errorf("invalid chunk info: %w", err)
		}

		// Store the chunk info
		chunkIndex := 0
		if start == 0 {
			chunksize := end - start + 1
			totalChunks := (total + chunksize - 1) / chunksize
			uploadChunks.LoadOrStore(file.ID, &UploadChunk{
				Last:        chunkIndex,
				Total:       total,
				Chunksize:   chunksize,
				TotalChunks: totalChunks,
				// Cache metadata from first chunk
				ContentType:   file.ContentType,
				Filename:      file.Filename,
				UserPath:      file.UserPath,
				CompressImage: option.CompressImage,
				CompressSize:  option.CompressSize,
			})
		}

		// Update the chunk index
		v, ok := uploadChunks.Load(file.ID)
		if !ok {
			return nil, fmt.Errorf("chunk data not found")
		}

		chunkdata := v.(*UploadChunk)

		// Update the chunk index
		if start != 0 {
			chunkIndex = chunkdata.Last + 1
			chunkdata.Last = chunkIndex
			uploadChunks.Store(file.ID, chunkdata)

			// For non-first chunks, use cached metadata from first chunk
			file.ContentType = chunkdata.ContentType
			file.Filename = chunkdata.Filename
			file.UserPath = chunkdata.UserPath
		}

		// Apply gzip compression if requested
		if option.Gzip {
			compressed, err := GzipFromReader(reader)
			if err != nil {
				return nil, fmt.Errorf("failed to gzip chunk: %w", err)
			}
			reader = bytes.NewReader(compressed)

		}

		// Upload chunk using the storage path
		err = manager.storage.UploadChunk(ctx, file.Path, chunkIndex, reader, file.ContentType)
		if err != nil {
			return nil, err
		}

		// Save to database on first chunk only
		if start == 0 {
			file.Status = "uploading"
			err = manager.saveFileToDatabase(ctx, file, file.Path, option)
			if err != nil {
				return nil, fmt.Errorf("failed to create database record for chunked upload: %w", err)
			}
		}

		// Fix the file size, the file size is the sum of all chunks
		file.Bytes = chunkIndex * int(chunkdata.Chunksize)
		file.Status = "uploading"

		// If this is the last chunk, merge all chunks
		if fileheader.Complete() {
			err = manager.storage.MergeChunks(ctx, file.Path, int(chunkdata.TotalChunks))
			if err != nil {
				return nil, err
			}

			// Set initial file size from chunks
			file.Bytes = int(chunkdata.Total)

			// Apply image compression if requested and it's the final file
			// Use cached compress options from first chunk
			if chunkdata.CompressImage && strings.HasPrefix(file.ContentType, "image/") {
				// Create a temporary option with cached compress size
				compressOption := UploadOption{
					CompressSize: chunkdata.CompressSize,
				}
				compressedBytes, err := manager.compressStoredImageAndGetSize(ctx, file, compressOption)
				if err != nil {
					return nil, err
				}
				// Update file size to compressed size
				file.Bytes = compressedBytes
			}

			// Remove the chunk data
			uploadChunks.Delete(file.ID)

			// Update status to uploaded
			file.Status = "uploaded"

			// Update only bytes and status for the last chunk
			err = manager.saveFileToDatabase(ctx, file, file.Path, option)
			if err != nil {
				return nil, fmt.Errorf("failed to update chunked file status: %w", err)
			}
		}

		return file, nil
	}

	// Handle single file upload
	var finalReader io.Reader = reader

	// Apply gzip compression if requested
	if option.Gzip {
		compressed, err := GzipFromReader(reader)
		if err != nil {
			return nil, fmt.Errorf("failed to gzip file: %w", err)
		}

		finalReader = bytes.NewReader(compressed)
	}

	// Apply image compression if requested
	if option.CompressImage && strings.HasPrefix(file.ContentType, "image/") {
		size := option.CompressSize
		if size == 0 {
			size = 1920
		}

		// Read original data for fallback
		var originalData []byte
		var err error

		// If gzip was applied, we need to decompress first
		if option.Gzip {
			data, err := io.ReadAll(finalReader)
			if err != nil {
				return nil, err
			}
			decompressed, err := Gunzip(data)
			if err != nil {
				return nil, err
			}
			originalData = decompressed
			finalReader = bytes.NewReader(decompressed)
		} else {
			originalData, err = io.ReadAll(finalReader)
			if err != nil {
				return nil, err
			}
			finalReader = bytes.NewReader(originalData)
		}

		// Try to compress the image with failback mechanism
		compressed, err := CompressImage(finalReader, file.ContentType, size)
		if err != nil {
			// Log the error and use original file as fallback
			log.Warn("Failed to compress image (content-type: %s, file: %s): %v. Using original file.",
				file.ContentType, file.Filename, err)
			// Use original data
			compressed = originalData
		}

		// Re-apply gzip if it was requested
		if option.Gzip {
			gzipped, err := Gzip(compressed)
			if err != nil {
				return nil, err
			}
			finalReader = bytes.NewReader(gzipped)
		} else {
			finalReader = bytes.NewReader(compressed)
		}
	}

	// Upload the file to storage using the generated storage path
	actualStoragePath, err := manager.storage.Upload(ctx, file.Path, finalReader, file.ContentType)
	if err != nil {
		return nil, err
	}

	// Update the actual storage path if storage returns a different path
	if actualStoragePath != "" && actualStoragePath != file.Path {
		file.Path = actualStoragePath
	}

	// Update the file status
	file.Status = "uploaded"

	// Save file information to database
	err = manager.saveFileToDatabase(ctx, file, file.Path, option)
	if err != nil {
		return nil, fmt.Errorf("failed to save file to database: %w", err)
	}

	return file, nil
}

// compressStoredImageAndGetSize compresses the stored image and returns the compressed size
func (manager Manager) compressStoredImageAndGetSize(ctx context.Context, file *File, option UploadOption) (int, error) {
	// Download the stored file using storage path
	reader, err := manager.storage.Reader(ctx, file.Path)
	if err != nil {
		return 0, err
	}
	defer reader.Close()

	size := option.CompressSize
	if size == 0 {
		size = 1920
	}

	// Read original data for fallback
	originalData, err := io.ReadAll(reader)
	if err != nil {
		return 0, err
	}

	// Try to compress the image with failback mechanism
	compressed, err := CompressImage(bytes.NewReader(originalData), file.ContentType, size)
	if err != nil {
		// Log the error and keep original file
		log.Warn("Failed to compress stored image (content-type: %s, file: %s): %v. Keeping original file.",
			file.ContentType, file.Filename, err)
		// File is already stored (merged chunks), just return original size
		return len(originalData), nil
	}

	// Re-upload the compressed image using storage path
	_, err = manager.storage.Upload(ctx, file.Path, bytes.NewReader(compressed), file.ContentType)
	if err != nil {
		return 0, err
	}

	// Return the compressed size
	return len(compressed), nil
}

// Download downloads a file
func (manager Manager) Download(ctx context.Context, fileID string) (*FileResponse, error) {
	// Get real storage path from database
	storagePath, err := manager.getStoragePathFromDatabase(ctx, fileID)
	if err != nil {
		return nil, err
	}

	reader, contentType, err := manager.storage.Download(ctx, storagePath)
	if err != nil {
		return nil, err
	}

	extension := filepath.Ext(storagePath)
	if extension == "" {
		// Try to get extension from content type
		extensions, err := mime.ExtensionsByType(contentType)
		if err == nil && len(extensions) > 0 {
			extension = extensions[0]
		}
	}

	return &FileResponse{
		Reader:      reader,
		ContentType: contentType,
		Extension:   extension,
	}, nil
}

// Read reads a file and returns the content as bytes
func (manager Manager) Read(ctx context.Context, fileID string) ([]byte, error) {
	// Get file info from database to check if it's gzipped
	file, err := manager.getFileFromDatabase(ctx, fileID)
	if err != nil {
		return nil, err
	}

	reader, err := manager.storage.Reader(ctx, file.Path)
	if err != nil {
		return nil, err
	}
	defer reader.Close()

	data, err := io.ReadAll(reader)
	if err != nil {
		return nil, err
	}

	// Storage layer already handles gzip decompression for .gz files
	// No need to decompress again at Manager level

	return data, nil
}

// ReadBase64 reads a file and returns the content as base64 encoded string
func (manager Manager) ReadBase64(ctx context.Context, fileID string) (string, error) {
	data, err := manager.Read(ctx, fileID)
	if err != nil {
		return "", err
	}

	return base64.StdEncoding.EncodeToString(data), nil
}

// Info retrieves complete file information from database by file ID
func (manager Manager) Info(ctx context.Context, fileID string) (*File, error) {
	return manager.getFileFromDatabase(ctx, fileID)
}

// List retrieves files from database with pagination and filtering
func (manager Manager) List(ctx context.Context, option ListOption) (*ListResult, error) {
	m := model.Select("__yao.attachment")

	// Set default values
	page := option.Page
	if page <= 0 {
		page = 1
	}

	pageSize := option.PageSize
	if pageSize <= 0 {
		pageSize = 20
	}

	// Build query parameters
	queryParam := model.QueryParam{}

	// Add select fields
	if len(option.Select) > 0 {
		queryParam.Select = make([]interface{}, 0, len(option.Select))
		for _, field := range option.Select {
			queryParam.Select = append(queryParam.Select, field)
		}
	} else {
		// Default: exclude the 'content' field (which may contain large text data)
		// Only include it if explicitly requested in Select
		queryParam.Select = []interface{}{
			"id", "file_id", "uploader", "content_type", "name", "url", "description",
			"type", "user_path", "path", "groups", "gzip", "bytes", "status",
			"progress", "error", "preset", "public", "share",
			"created_at", "updated_at", "deleted_at",
			"__yao_created_by", "__yao_updated_by", "__yao_team_id", "__yao_tenant_id",
		}
	}

	// Add filters
	if len(option.Filters) > 0 {
		queryParam.Wheres = make([]model.QueryWhere, 0, len(option.Filters))
		for field, value := range option.Filters {
			where := model.QueryWhere{
				Column: field,
				Value:  value,
			}

			// Handle special operators for wildcard matching
			if strValue, ok := value.(string); ok {
				if strings.Contains(strValue, "*") {
					// Wildcard matching for LIKE queries
					where.OP = "like"
					where.Value = strings.ReplaceAll(strValue, "*", "%")
				}
			}

			queryParam.Wheres = append(queryParam.Wheres, where)
		}
	}

	// Add advanced where clauses (for permission filtering, etc.)
	if len(option.Wheres) > 0 {
		if queryParam.Wheres == nil {
			queryParam.Wheres = make([]model.QueryWhere, 0, len(option.Wheres))
		}
		queryParam.Wheres = append(queryParam.Wheres, option.Wheres...)
	}

	// Add ordering
	if option.OrderBy != "" {
		// Parse order by string like "created_at desc" or "name asc"
		parts := strings.Fields(option.OrderBy)
		if len(parts) >= 1 {
			orderField := parts[0]
			orderDirection := "asc"
			if len(parts) >= 2 {
				orderDirection = strings.ToLower(parts[1])
			}

			queryParam.Orders = []model.QueryOrder{
				{
					Column: orderField,
					Option: orderDirection,
				},
			}
		}
	} else {
		// Default order by created_at desc
		queryParam.Orders = []model.QueryOrder{
			{
				Column: "created_at",
				Option: "desc",
			},
		}
	}

	// Use model's built-in Paginate method
	result, err := m.Paginate(queryParam, page, pageSize)
	if err != nil {
		return nil, fmt.Errorf("failed to paginate files: %w", err)
	}

	// Extract pagination info from result
	total := int64(0)
	if totalInterface, ok := result["total"]; ok {
		if totalInt, ok := totalInterface.(int); ok {
			total = int64(totalInt)
		} else if totalInt64, ok := totalInterface.(int64); ok {
			total = totalInt64
		}
	}

	// Extract data from result - handle maps.MapStrAny type
	var records []map[string]interface{}
	if dataInterface, ok := result["data"]; ok {
		// The data is of type []maps.MapStrAny, need to convert
		if dataSlice, ok := dataInterface.([]interface{}); ok {
			records = make([]map[string]interface{}, len(dataSlice))
			for i, item := range dataSlice {
				if record, ok := item.(map[string]interface{}); ok {
					records[i] = record
				}
			}
		} else {
			// Try to handle it as the actual type returned by gou using reflection
			dataValue := reflect.ValueOf(dataInterface)
			if dataValue.Kind() == reflect.Slice {
				length := dataValue.Len()
				records = make([]map[string]interface{}, length)
				for i := 0; i < length; i++ {
					item := dataValue.Index(i).Interface()
					// Convert the item to map[string]interface{} using reflection
					if itemValue := reflect.ValueOf(item); itemValue.Kind() == reflect.Map {
						record := make(map[string]interface{})
						for _, key := range itemValue.MapKeys() {
							if keyStr := key.String(); keyStr != "" {
								record[keyStr] = itemValue.MapIndex(key).Interface()
							}
						}
						records[i] = record
					}
				}
			}
		}
	}

	// Convert records to File structs
	files := make([]*File, 0, len(records))
	for _, record := range records {
		file := &File{}

		// Map required fields
		if fileID, ok := record["file_id"].(string); ok {
			file.ID = fileID
		}
		if name, ok := record["name"].(string); ok {
			file.Filename = name
		}
		if contentType, ok := record["content_type"].(string); ok {
			file.ContentType = contentType
		}
		if status, ok := record["status"].(string); ok {
			file.Status = status
		}

		// Map optional fields
		if userPath, ok := record["user_path"].(string); ok {
			file.UserPath = userPath
		}
		if path, ok := record["path"].(string); ok {
			file.Path = path
		}
		if bytes, ok := record["bytes"].(int64); ok {
			file.Bytes = int(bytes)
		} else if bytesInt, ok := record["bytes"].(int); ok {
			file.Bytes = bytesInt
		}
		if createdAt, ok := record["created_at"].(int64); ok {
			file.CreatedAt = int(createdAt)
		} else if createdAtInt, ok := record["created_at"].(int); ok {
			file.CreatedAt = createdAtInt
		} else {
			// Fallback to current time if not available
			file.CreatedAt = int(time.Now().Unix())
		}

		files = append(files, file)
	}

	// Calculate total pages
	totalPages := int((total + int64(pageSize) - 1) / int64(pageSize))

	return &ListResult{
		Files:      files,
		Total:      total,
		Page:       page,
		PageSize:   pageSize,
		TotalPages: totalPages,
	}, nil
}

// validate validates the file and option
func (manager Manager) makeFile(file *FileHeader, option UploadOption) (*File, error) {

	// Validate max size
	if manager.maxsize > 0 && file.Size > manager.maxsize {
		return nil, fmt.Errorf("file size %d exceeds the maximum size of %d", file.Size, manager.maxsize)
	}

	// Use original filename if provided, otherwise use the file header filename
	filename := file.Filename
	userPath := option.OriginalFilename
	if userPath != "" {
		// If user provided a path, extract just the filename for the filename field
		filename = filepath.Base(userPath)
	}

	extension := filepath.Ext(filename)

	// Get the content type
	// For chunked uploads, file.Header may have incorrect content-type (e.g., application/octet-stream for Blob)
	// Try to detect from filename extension first, then fallback to header
	contentType := file.Header.Get("Content-Type")
	if extension != "" {
		// Try to get content type from extension
		detectedType := mime.TypeByExtension(extension)
		if detectedType != "" {
			// If detected type is not the generic octet-stream, use it
			// This handles chunked uploads where the header has incorrect type
			if detectedType != "application/octet-stream" || contentType == "application/octet-stream" {
				contentType = detectedType
			}
		}
	}

	// Get the extension from the content type if not available from filename
	if extension == "" {
		// Special handling for common types
		switch contentType {
		case "text/plain":
			extension = ".txt"
		case "image/jpeg":
			extension = ".jpg"
		case "image/png":
			extension = ".png"
		case "application/pdf":
			extension = ".pdf"
		default:
			extensions, err := mime.ExtensionsByType(contentType)
			if err == nil && len(extensions) > 0 {
				// For text/plain, prefer .txt over .conf
				if contentType == "text/plain" {
					for _, ext := range extensions {
						if ext == ".txt" {
							extension = ext
							break
						}
					}
					if extension == "" {
						extension = ".txt"
					}
				} else {
					extension = extensions[0]
				}
			}
		}
	}

	// Validate allowed types
	if !manager.allowed(contentType, extension) {
		return nil, fmt.Errorf("%s type %s is not allowed", filename, contentType)
	}

	// Generate file ID and storage path using the new approach
	id, storagePath, err := manager.generateFilePaths(file, extension, option)
	if err != nil {
		return nil, err
	}

	// Set the path: use userPath if provided, otherwise use filename
	filePath := userPath
	if filePath == "" {
		filePath = filename
	}

	return &File{
		ID:          id,
		UserPath:    userPath,    // Keep user's original input exactly as provided
		Path:        storagePath, // Complete storage path: Groups + filename
		Filename:    filename,    // Use just the filename (extracted from path or header)
		ContentType: contentType,
		Bytes:       int(file.Size),
		CreatedAt:   int(time.Now().Unix()),
		Status:      "uploading",
	}, nil
}

func (manager Manager) allowed(contentType string, extension string) bool {

	// text/*, image/*, audio/*, video/*, application/yao-*, ...
	for _, t := range manager.allowedTypes.wildcards {
		prefix := strings.TrimSuffix(t, "*")
		if strings.HasPrefix(contentType, prefix) {
			return true
		}
	}

	// Accepted types
	if _, ok := manager.allowedTypes.mapping[contentType]; ok {
		return true
	}

	// Accepted extensions
	if _, ok := manager.allowedTypes.mapping[extension]; ok {
		return true
	}

	// Not allowed
	return false
}

// generateFileID generates file ID and storage path based on Groups and filename
func (manager Manager) generateFilePaths(file *FileHeader, extension string, option UploadOption) (fileID string, storagePath string, err error) {

	// 1. Get the filename
	var filename string
	if file.Fingerprint() != "" {
		filename = file.Fingerprint()
	} else if file.IsChunk() {
		filename = file.UID()
	} else {
		// Generate unique filename to avoid conflicts
		var originalName string
		if option.OriginalFilename != "" {
			originalName = filepath.Base(option.OriginalFilename)
		} else {
			originalName = file.Filename
		}

		// Extract extension from original filename
		ext := filepath.Ext(originalName)
		if ext == "" && extension != "" {
			ext = extension
		}

		// Generate unique filename: MD5 hash of original name + timestamp + extension
		nameHash := generateID(originalName + fmt.Sprintf("%d", time.Now().UnixNano()))
		filename = nameHash[:16] + ext // Use first 16 chars of hash + extension
	}

	// 2. Build complete storage path: Groups + filename
	pathParts := []string{}

	// Add groups to path
	if len(option.Groups) > 0 {
		pathParts = append(pathParts, option.Groups...)
	}

	// Add filename
	pathParts = append(pathParts, filename)

	// Join to create complete storage path
	storagePath = strings.Join(pathParts, "/")

	// 3. Validate the storage path
	if !isValidPath(storagePath) {
		return "", "", fmt.Errorf("invalid storage path: %s", storagePath)
	}

	// 4. Generate ID as alias of the storage path (for security)
	fileID = generateID(storagePath)

	// 5. Add gzip extension to storage path if needed (not to fileID)
	if option.Gzip {
		storagePath = storagePath + ".gz"
	}

	return fileID, storagePath, nil
}

// generateID generates a URL-safe ID based on the storage path
func generateID(storagePath string) string {
	hash := md5.Sum([]byte(storagePath))
	return hex.EncodeToString(hash[:])
}

// isValidPath checks if a file path is valid
func isValidPath(path string) bool {
	if path == "" {
		return false
	}

	// Check for invalid characters that could cause issues
	invalidChars := []string{"../", "..\\", "\\", "//"}
	for _, invalid := range invalidChars {
		if strings.Contains(path, invalid) {
			return false
		}
	}

	return true
}

// getSize converts the size to bytes
func getSize(size string) (int64, error) {
	if size == "" || size == "0" {
		return 0, fmt.Errorf("size is empty")
	}

	unit := strings.ToUpper(size[len(size)-1:])
	str := size[:len(size)-1]
	if unit != "B" && unit != "K" && unit != "M" && unit != "G" {
		unit = "B"
		str = size
	}

	value, err := strconv.ParseInt(str, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid size: %s %s", size, err)
	}

	switch unit {
	case "B":
		return value, nil
	case "K":
		return value * 1024, nil
	case "M":
		return value * 1024 * 1024, nil
	case "G":
		return value * 1024 * 1024 * 1024, nil
	}

	return 0, fmt.Errorf("invalid size: %s", size)
}

// Exists checks if a file exists in storage
func (manager Manager) Exists(ctx context.Context, fileID string) bool {
	// Check if file exists in database first
	storagePath, err := manager.getStoragePathFromDatabase(ctx, fileID)
	if err != nil {
		return false
	}

	// Then check if it exists in storage
	return manager.storage.Exists(ctx, storagePath)
}

// Delete deletes a file from storage
func (manager Manager) Delete(ctx context.Context, fileID string) error {
	// Get real storage path from database
	storagePath, err := manager.getStoragePathFromDatabase(ctx, fileID)
	if err != nil {
		return err
	}

	// Delete from storage
	err = manager.storage.Delete(ctx, storagePath)
	if err != nil {
		return err
	}

	// Delete from database
	m := model.Select("__yao.attachment")
	_, err = m.DeleteWhere(model.QueryParam{
		Wheres: []model.QueryWhere{
			{Column: "file_id", Value: fileID},
		},
	})

	if err != nil {
		return fmt.Errorf("failed to delete from database: %w", err)
	}

	return nil
}

// saveFileToDatabase saves file information to the database
// For chunked uploads, it only updates bytes/status/progress if record exists
func (manager Manager) saveFileToDatabase(ctx context.Context, file *File, storagePath string, option UploadOption) error {

	m := model.Select("__yao.attachment")

	// Check if record exists first
	records, err := m.Get(model.QueryParam{
		Select: []interface{}{"file_id"},
		Wheres: []model.QueryWhere{
			{Column: "file_id", Value: file.ID},
		},
	})

	if err != nil {
		return fmt.Errorf("failed to check existing record: %w", err)
	}

	if len(records) > 0 {
		// Record exists - this is a chunked upload update
		// Only update bytes, status, and progress (don't overwrite metadata)
		updateData := map[string]interface{}{
			"bytes":  int64(file.Bytes),
			"status": file.Status,
		}

		_, err = m.UpdateWhere(model.QueryParam{
			Wheres: []model.QueryWhere{
				{Column: "file_id", Value: file.ID},
			},
		}, updateData)

		return err
	}

	// Record doesn't exist - create new record with full metadata
	// Set default value for share if empty
	share := option.Share
	if share == "" {
		share = "private"
	}

	// Prepare data for database
	data := map[string]interface{}{
		"file_id":      file.ID,
		"uploader":     manager.Name,
		"content_type": file.ContentType,
		"name":         file.Filename,
		"user_path":    option.OriginalFilename,
		"path":         storagePath,
		"bytes":        int64(file.Bytes),
		"status":       file.Status,
		"gzip":         option.Gzip,
		"groups":       option.Groups,
		"public":       option.Public,
		"share":        share,
	}

	// Add Yao permission fields if provided
	if option.YaoCreatedBy != "" {
		data["__yao_created_by"] = option.YaoCreatedBy
	}
	if option.YaoUpdatedBy != "" {
		data["__yao_updated_by"] = option.YaoUpdatedBy
	}
	if option.YaoTeamID != "" {
		data["__yao_team_id"] = option.YaoTeamID
	}
	if option.YaoTenantID != "" {
		data["__yao_tenant_id"] = option.YaoTenantID
	}

	// Create new record
	_, err = m.Create(data)
	return err
}

// getFileFromDatabase retrieves file information from database by file_id
func (manager Manager) getFileFromDatabase(ctx context.Context, fileID string) (*File, error) {
	m := model.Select("__yao.attachment")

	records, err := m.Get(model.QueryParam{
		Select: []interface{}{
			"file_id", "name", "content_type", "status", "user_path", "path", "bytes",
			"public", "share", "__yao_created_by", "__yao_team_id", "__yao_tenant_id",
		},
		Wheres: []model.QueryWhere{
			{Column: "file_id", Value: fileID},
		},
		Limit: 1,
	})

	if err != nil {
		return nil, fmt.Errorf("failed to query file: %w", err)
	}

	if len(records) == 0 {
		return nil, fmt.Errorf("file not found")
	}

	record := records[0]

	// Convert database record to File struct
	file := &File{
		ID:          record["file_id"].(string),
		Filename:    record["name"].(string),
		ContentType: record["content_type"].(string),
		Status:      record["status"].(string),
		CreatedAt:   int(time.Now().Unix()), // TODO: get from database
	}

	// Handle optional fields
	if userPath, ok := record["user_path"].(string); ok {
		file.UserPath = userPath
	}

	if path, ok := record["path"].(string); ok {
		file.Path = path
	}

	if bytes, ok := record["bytes"].(int64); ok {
		file.Bytes = int(bytes)
	}

	// Handle permission fields with safe conversion
	file.Public = toBool(record["public"])
	file.Share = toString(record["share"])
	file.YaoCreatedBy = toString(record["__yao_created_by"])
	file.YaoTeamID = toString(record["__yao_team_id"])
	file.YaoTenantID = toString(record["__yao_tenant_id"])

	return file, nil
}

// getStoragePathFromDatabase retrieves the real storage path for a file_id
func (manager Manager) getStoragePathFromDatabase(ctx context.Context, fileID string) (string, error) {
	m := model.Select("__yao.attachment")

	records, err := m.Get(model.QueryParam{
		Select: []interface{}{"path"},
		Wheres: []model.QueryWhere{
			{Column: "file_id", Value: fileID},
		},
	})

	if err != nil {
		return "", fmt.Errorf("failed to query database: %w", err)
	}

	if len(records) == 0 {
		return "", fmt.Errorf("file not found: %s", fileID)
	}

	if path, ok := records[0]["path"].(string); ok && path != "" {
		return path, nil
	}

	return "", fmt.Errorf("invalid storage path for file ID: %s", fileID)
}

// GetText retrieves the parsed text content for a file by its ID
// By default, returns the preview (first 2000 characters) from 'content_preview' field
// Set fullContent to true to retrieve the complete text from 'content' field
func (manager Manager) GetText(ctx context.Context, fileID string, fullContent ...bool) (string, error) {
	m := model.Select("__yao.attachment")

	// Determine which field to query
	wantFullContent := false
	if len(fullContent) > 0 {
		wantFullContent = fullContent[0]
	}

	fieldName := "content_preview"
	if wantFullContent {
		fieldName = "content"
	}

	records, err := m.Get(model.QueryParam{
		Select: []interface{}{fieldName},
		Wheres: []model.QueryWhere{
			{Column: "file_id", Value: fileID},
		},
		Limit: 1,
	})

	if err != nil {
		return "", fmt.Errorf("failed to query text content: %w", err)
	}

	if len(records) == 0 {
		return "", fmt.Errorf("file not found: %s", fileID)
	}

	// Handle content field - it may be nil, string, or other types
	if content, ok := records[0][fieldName].(string); ok {
		return content, nil
	}

	// If content is nil or not a string, return empty string
	return "", nil
}

// SaveText saves the parsed text content for a file by its ID
// Automatically saves both full content and preview (first 2000 characters)
// Updates both 'content' and 'content_preview' fields in the attachment record
func (manager Manager) SaveText(ctx context.Context, fileID string, text string) error {
	m := model.Select("__yao.attachment")

	// Check if record exists first
	records, err := m.Get(model.QueryParam{
		Select: []interface{}{"file_id"},
		Wheres: []model.QueryWhere{
			{Column: "file_id", Value: fileID},
		},
		Limit: 1,
	})

	if err != nil {
		return fmt.Errorf("failed to check file existence: %w", err)
	}

	if len(records) == 0 {
		return fmt.Errorf("file not found: %s", fileID)
	}

	// Create preview: first 2000 characters (or runes for proper UTF-8 handling)
	preview := text
	const maxPreviewLength = 2000
	if len([]rune(text)) > maxPreviewLength {
		preview = string([]rune(text)[:maxPreviewLength])
	}

	// Update both content and content_preview fields
	updateData := map[string]interface{}{
		"content":         text,
		"content_preview": preview,
	}

	_, err = m.UpdateWhere(model.QueryParam{
		Wheres: []model.QueryWhere{
			{Column: "file_id", Value: fileID},
		},
	}, updateData)

	if err != nil {
		return fmt.Errorf("failed to save text content: %w", err)
	}

	return nil
}
