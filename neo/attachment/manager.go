package attachment

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"io"
	"mime"
	"mime/multipart"
	"net/http"
	"net/textproto"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/yaoapp/yao/config"
	"github.com/yaoapp/yao/neo/attachment/local"
	"github.com/yaoapp/yao/neo/attachment/s3"
)

// Managers the managers
var Managers = map[string]*Manager{}
var uploadChunks = sync.Map{}

// UploadChunk is the chunk data
type UploadChunk struct {
	Last        int
	Total       int64
	Chunksize   int64
	TotalChunks int64
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
		break

	case "s3":
		storage, err := s3.New(option.Options)
		if err != nil {
			return nil, err
		}
		manager.storage = storage
		break

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
	if option.AllowedTypes != nil && len(option.AllowedTypes) > 0 {
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
		}

		// Apply gzip compression if requested
		if option.Gzip {
			compressed, err := GzipFromReader(reader)
			if err != nil {
				return nil, fmt.Errorf("failed to gzip chunk: %w", err)
			}
			reader = bytes.NewReader(compressed)

		}

		// Upload chunk
		err = manager.storage.UploadChunk(ctx, file.ID, chunkIndex, reader, file.ContentType)
		if err != nil {
			return nil, err
		}

		// Fix the file size, the file size is the sum of all chunks
		file.Bytes = chunkIndex * int(chunkdata.Chunksize)
		file.Status = "uploading"

		// If this is the last chunk, merge all chunks
		if fileheader.Complete() {
			err = manager.storage.MergeChunks(ctx, file.ID, int(chunkdata.TotalChunks))
			if err != nil {
				return nil, err
			}

			// Apply image compression if requested and it's the final file
			if option.CompressImage && strings.HasPrefix(file.ContentType, "image/") {
				err = manager.compressStoredImage(ctx, file, option)
				if err != nil {
					return nil, err
				}
			}

			// Remove the chunk data
			uploadChunks.Delete(file.ID)

			// Fix the file size
			file.Bytes = int(chunkdata.Total)
			file.Status = "uploaded"
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
			finalReader = bytes.NewReader(decompressed)
		}

		compressed, err := CompressImage(finalReader, file.ContentType, size)
		if err != nil {
			return nil, err
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

	// Upload the file to storage
	id, err := manager.storage.Upload(ctx, file.ID, finalReader, file.ContentType)
	if err != nil {
		return nil, err
	}

	// Update the file ID
	file.ID = id
	file.Status = "uploaded"
	return file, nil
}

// compressStoredImage compresses an already stored image
func (manager Manager) compressStoredImage(ctx context.Context, file *File, option UploadOption) error {
	// Download the stored file
	reader, err := manager.storage.Reader(ctx, file.ID)
	if err != nil {
		return err
	}
	defer reader.Close()

	size := option.CompressSize
	if size == 0 {
		size = 1920
	}

	// Compress the image
	compressed, err := CompressImage(reader, file.ContentType, size)
	if err != nil {
		return err
	}

	// Re-upload the compressed image
	_, err = manager.storage.Upload(ctx, file.ID, bytes.NewReader(compressed), file.ContentType)
	return err
}

// Download downloads a file
func (manager Manager) Download(ctx context.Context, fileID string) (*FileResponse, error) {
	reader, contentType, err := manager.storage.Download(ctx, fileID)
	if err != nil {
		return nil, err
	}

	extension := filepath.Ext(fileID)
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
	reader, err := manager.storage.Reader(ctx, fileID)
	if err != nil {
		return nil, err
	}
	defer reader.Close()

	return io.ReadAll(reader)
}

// ReadBase64 reads a file and returns the content as base64 encoded string
func (manager Manager) ReadBase64(ctx context.Context, fileID string) (string, error) {
	data, err := manager.Read(ctx, fileID)
	if err != nil {
		return "", err
	}

	return base64.StdEncoding.EncodeToString(data), nil
}

// validate validates the file and option
func (manager Manager) makeFile(file *FileHeader, option UploadOption) (*File, error) {

	// Validate max size
	if manager.maxsize > 0 && file.Size > manager.maxsize {
		return nil, fmt.Errorf("file size %d exceeds the maximum size of %d", file.Size, manager.maxsize)
	}

	// Get the content type
	contentType := file.Header.Get("Content-Type")

	// Use original filename if provided, otherwise use the file header filename
	filename := file.Filename
	if option.OriginalFilename != "" {
		filename = option.OriginalFilename
	}

	extension := filepath.Ext(filename)

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

	// Generate file ID
	id, err := manager.generateFileID(file, extension, option)
	if err != nil {
		return nil, err
	}

	return &File{
		ID:          id,
		Filename:    filename, // Use the correct filename (original or from header)
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

// generateFileID generates a file ID with proper namespace
func (manager Manager) generateFileID(file *FileHeader, extension string, option UploadOption) (string, error) {

	filename := file.Fingerprint()

	// If the fingerprint is not set, use the filename
	if filename == "" {
		filename = file.Filename
	}

	// Use original filename if provided for better file identification
	if option.OriginalFilename != "" {
		filename = option.OriginalFilename
	}

	if file.IsChunk() {
		filename = file.UID()
	}

	hash := fmt.Sprintf("%x", sha256.Sum256([]byte(filename)))[:8]
	date := time.Now().Format("20060102")
	path := filepath.Join("attachments", date)
	if option.UserID != "" {
		path = filepath.Join(path, option.UserID)
	}

	if option.ChatID != "" {
		path = filepath.Join(path, option.ChatID)
	}

	if option.AssistantID != "" {
		path = filepath.Join(path, option.AssistantID)
	}

	id := filepath.Join(path, hash[:2], hash[2:4], hash) + extension
	if option.Gzip {
		id = id + ".gz"
	}
	return id, nil
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
