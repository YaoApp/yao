package local

import (
	"compress/gzip"
	"context"
	"crypto/sha256"
	"fmt"
	"io"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// MaxImageSize maximum image size (1920x1080)
const MaxImageSize = 1920

// Storage the local storage driver
type Storage struct {
	Path        string                     `json:"path" yaml:"path"`
	Compression bool                       `json:"compression" yaml:"compression"`
	BaseURL     string                     `json:"base_url" yaml:"base_url"`
	PreviewURL  func(fileID string) string `json:"-" yaml:"-"`
}

// New create a new local storage
func New(options map[string]interface{}) (*Storage, error) {
	storage := &Storage{
		Compression: true,
	}

	if path, ok := options["path"].(string); ok {
		storage.Path = path
	}

	if compression, ok := options["compression"].(bool); ok {
		storage.Compression = compression
	}

	if baseURL, ok := options["base_url"].(string); ok {
		storage.BaseURL = baseURL
	}

	if previewURL, ok := options["preview_url"].(func(string) string); ok {
		storage.PreviewURL = previewURL
	}

	if storage.Path == "" {
		return nil, fmt.Errorf("path is required")
	}

	// Ensure the base path exists
	if err := os.MkdirAll(storage.Path, 0755); err != nil {
		return nil, fmt.Errorf("failed to create base path: %w", err)
	}

	return storage, nil
}

// Upload upload file to local storage
func (storage *Storage) Upload(ctx context.Context, path string, reader io.Reader, contentType string) (string, error) {
	fullPath := filepath.Join(storage.Path, path)

	// Create directory if not exists
	dir := filepath.Dir(fullPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", err
	}

	// Create and write file
	file, err := os.Create(fullPath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	_, err = io.Copy(file, reader)
	if err != nil {
		return "", err
	}

	return path, nil
}

// UploadChunk uploads a chunk of a file
func (storage *Storage) UploadChunk(ctx context.Context, path string, chunkIndex int, reader io.Reader, contentType string) error {
	// Create chunks directory
	chunksDir := filepath.Join(storage.Path, ".chunks", path)
	if err := os.MkdirAll(chunksDir, 0755); err != nil {
		return err
	}

	// Write chunk file
	chunkPath := filepath.Join(chunksDir, fmt.Sprintf("chunk_%d", chunkIndex))
	file, err := os.Create(chunkPath)
	if err != nil {
		return err
	}
	defer file.Close()

	_, err = io.Copy(file, reader)
	return err
}

// MergeChunks merges all chunks into the final file
func (storage *Storage) MergeChunks(ctx context.Context, path string, totalChunks int) error {
	chunksDir := filepath.Join(storage.Path, ".chunks", path)
	finalPath := filepath.Join(storage.Path, path)

	// Create directory for final file
	dir := filepath.Dir(finalPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	// Create final file
	finalFile, err := os.Create(finalPath)
	if err != nil {
		return err
	}
	defer finalFile.Close()

	// Read and merge chunks in order
	for i := 0; i < totalChunks; i++ {
		chunkPath := filepath.Join(chunksDir, fmt.Sprintf("chunk_%d", i))
		chunkFile, err := os.Open(chunkPath)
		if err != nil {
			return fmt.Errorf("failed to read chunk %d: %w", i, err)
		}

		_, err = io.Copy(finalFile, chunkFile)
		chunkFile.Close()
		if err != nil {
			return fmt.Errorf("failed to copy chunk %d: %w", i, err)
		}
	}

	// Clean up chunks directory
	os.RemoveAll(chunksDir)
	return nil
}

// Reader read file from local storage
func (storage *Storage) Reader(ctx context.Context, path string) (io.ReadCloser, error) {
	fullpath := filepath.Join(storage.Path, path)

	reader, err := os.Open(fullpath)
	if err != nil {
		return nil, err
	}

	// If the file is a gzip file, decompress it
	if strings.HasSuffix(path, ".gz") {
		reader, err := gzip.NewReader(reader)
		if err != nil {
			return nil, err
		}
		return reader, nil
	}

	return reader, nil
}

// Download download file from local storage
func (storage *Storage) Download(ctx context.Context, path string) (io.ReadCloser, string, error) {
	fullPath := filepath.Join(storage.Path, path)
	reader, err := os.Open(fullPath)
	if err != nil {
		return nil, "", err
	}

	// Try to detect content type from file extension
	contentType := "application/octet-stream"
	ext := filepath.Ext(strings.TrimSuffix(path, ".gz"))
	switch strings.ToLower(ext) {
	case ".txt":
		contentType = "text/plain"
	case ".html":
		contentType = "text/html"
	case ".css":
		contentType = "text/css"
	case ".js":
		contentType = "application/javascript"
	case ".json":
		contentType = "application/json"
	case ".jpg", ".jpeg":
		contentType = "image/jpeg"
	case ".png":
		contentType = "image/png"
	case ".gif":
		contentType = "image/gif"
	case ".pdf":
		contentType = "application/pdf"
	case ".mp4":
		contentType = "video/mp4"
	case ".mp3":
		contentType = "audio/mpeg"
	case ".wav":
		contentType = "audio/wav"
	case ".ogg":
		contentType = "audio/ogg"
	case ".webm":
		contentType = "video/webm"
	case ".webp":
		contentType = "image/webp"
	case ".zip":
	}

	// If the file is a gzip file, decompress it
	if strings.HasSuffix(path, ".gz") {
		reader, err := gzip.NewReader(reader)
		if err != nil {
			return nil, "", err
		}
		return reader, contentType, nil
	}

	return reader, contentType, nil
}

// URL get file url
func (storage *Storage) URL(ctx context.Context, path string) string {
	if storage.PreviewURL != nil {
		return storage.PreviewURL(path)
	}
	if storage.BaseURL != "" {
		return fmt.Sprintf("%s/%s", strings.TrimRight(storage.BaseURL, "/"), path)
	}
	return fmt.Sprintf("%s/%s", storage.Path, path)
}

// GetContent gets file content as bytes
func (storage *Storage) GetContent(ctx context.Context, path string) ([]byte, error) {
	reader, err := storage.Reader(ctx, path)
	if err != nil {
		return nil, err
	}
	defer reader.Close()

	return io.ReadAll(reader)
}

// Exists checks if a file exists
func (storage *Storage) Exists(ctx context.Context, path string) bool {
	fullpath := filepath.Join(storage.Path, path)
	_, err := os.Stat(fullpath)
	return err == nil
}

// Delete deletes a file
func (storage *Storage) Delete(ctx context.Context, path string) error {
	fullpath := filepath.Join(storage.Path, path)
	return os.Remove(fullpath)
}

func (storage *Storage) makeID(filename string, ext string) string {
	date := time.Now().Format("20060102")
	hash := fmt.Sprintf("%x", sha256.Sum256([]byte(filename)))[:8]
	name := strings.TrimSuffix(filepath.Base(filename), ext)
	return fmt.Sprintf("%s/%s-%s%s", date, name, hash, ext)
}

// LocalPath returns the absolute path of the file and its content type
func (storage *Storage) LocalPath(ctx context.Context, path string) (string, string, error) {
	fullPath := filepath.Join(storage.Path, path)

	// Check if file exists
	if _, err := os.Stat(fullPath); os.IsNotExist(err) {
		return "", "", fmt.Errorf("file not found: %s", path)
	}

	// For gzipped files, we need to detect the original content type, not the gzip wrapper
	var contentType string
	var err error

	if strings.HasSuffix(path, ".gz") {
		// For gzipped files, detect content type of the decompressed content
		originalPath := strings.TrimSuffix(path, ".gz")
		ext := filepath.Ext(originalPath)

		// First try to detect by original file extension
		contentType, err = detectContentTypeFromExtension(ext)
		if err != nil || contentType == "application/octet-stream" {
			// Fallback: decompress and detect from content
			contentType, err = detectContentTypeFromGzippedFile(fullPath)
			if err != nil {
				return "", "", fmt.Errorf("failed to detect content type from gzipped file: %w", err)
			}
		}
	} else {
		// Regular file content type detection
		contentType, err = detectContentType(fullPath)
		if err != nil {
			return "", "", fmt.Errorf("failed to detect content type: %w", err)
		}
	}

	// Return absolute path
	absPath, err := filepath.Abs(fullPath)
	if err != nil {
		return "", "", fmt.Errorf("failed to get absolute path: %w", err)
	}

	return absPath, contentType, nil
}

// detectContentType detects content type based on file extension and content
func detectContentType(filePath string) (string, error) {
	// First try to detect by file extension
	ext := strings.ToLower(filepath.Ext(filePath))

	// Common file extensions mapping
	switch ext {
	case ".txt":
		return "text/plain", nil
	case ".html", ".htm":
		return "text/html", nil
	case ".css":
		return "text/css", nil
	case ".js":
		return "application/javascript", nil
	case ".json":
		return "application/json", nil
	case ".xml":
		return "application/xml", nil
	case ".jpg", ".jpeg":
		return "image/jpeg", nil
	case ".png":
		return "image/png", nil
	case ".gif":
		return "image/gif", nil
	case ".webp":
		return "image/webp", nil
	case ".svg":
		return "image/svg+xml", nil
	case ".pdf":
		return "application/pdf", nil
	case ".doc":
		return "application/msword", nil
	case ".docx":
		return "application/vnd.openxmlformats-officedocument.wordprocessingml.document", nil
	case ".xls":
		return "application/vnd.ms-excel", nil
	case ".xlsx":
		return "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet", nil
	case ".ppt":
		return "application/vnd.ms-powerpoint", nil
	case ".pptx":
		return "application/vnd.openxmlformats-officedocument.presentationml.presentation", nil
	case ".zip":
		return "application/zip", nil
	case ".tar":
		return "application/x-tar", nil
	case ".gz":
		return "application/gzip", nil
	case ".mp3":
		return "audio/mpeg", nil
	case ".wav":
		return "audio/wav", nil
	case ".m4a":
		return "audio/mp4", nil
	case ".ogg":
		return "audio/ogg", nil
	case ".mp4":
		return "video/mp4", nil
	case ".avi":
		return "video/x-msvideo", nil
	case ".mov":
		return "video/quicktime", nil
	case ".webm":
		return "video/webm", nil
	case ".md", ".mdx":
		return "text/markdown", nil
	case ".yao":
		return "application/yao", nil
	case ".csv":
		return "text/csv", nil
	}

	// Try to detect by MIME package
	if contentType := mime.TypeByExtension(ext); contentType != "" {
		return contentType, nil
	}

	// Fallback: detect by reading file content
	file, err := os.Open(filePath)
	if err != nil {
		return "application/octet-stream", nil // Default fallback
	}
	defer file.Close()

	// Read first 512 bytes for content detection
	buffer := make([]byte, 512)
	n, err := file.Read(buffer)
	if err != nil && err != io.EOF {
		return "application/octet-stream", nil
	}

	// Use http.DetectContentType to detect based on content
	contentType := http.DetectContentType(buffer[:n])
	return contentType, nil
}

// detectContentTypeFromExtension detects content type based only on file extension
func detectContentTypeFromExtension(ext string) (string, error) {
	ext = strings.ToLower(ext)

	// Common file extensions mapping
	switch ext {
	case ".txt":
		return "text/plain", nil
	case ".html", ".htm":
		return "text/html", nil
	case ".css":
		return "text/css", nil
	case ".js":
		return "application/javascript", nil
	case ".json":
		return "application/json", nil
	case ".xml":
		return "application/xml", nil
	case ".jpg", ".jpeg":
		return "image/jpeg", nil
	case ".png":
		return "image/png", nil
	case ".gif":
		return "image/gif", nil
	case ".webp":
		return "image/webp", nil
	case ".svg":
		return "image/svg+xml", nil
	case ".pdf":
		return "application/pdf", nil
	case ".doc":
		return "application/msword", nil
	case ".docx":
		return "application/vnd.openxmlformats-officedocument.wordprocessingml.document", nil
	case ".xls":
		return "application/vnd.ms-excel", nil
	case ".xlsx":
		return "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet", nil
	case ".ppt":
		return "application/vnd.ms-powerpoint", nil
	case ".pptx":
		return "application/vnd.openxmlformats-officedocument.presentationml.presentation", nil
	case ".zip":
		return "application/zip", nil
	case ".tar":
		return "application/x-tar", nil
	case ".mp3":
		return "audio/mpeg", nil
	case ".wav":
		return "audio/wav", nil
	case ".m4a":
		return "audio/mp4", nil
	case ".ogg":
		return "audio/ogg", nil
	case ".mp4":
		return "video/mp4", nil
	case ".avi":
		return "video/x-msvideo", nil
	case ".mov":
		return "video/quicktime", nil
	case ".webm":
		return "video/webm", nil
	case ".md", ".mdx":
		return "text/markdown", nil
	case ".yao":
		return "application/yao", nil
	case ".csv":
		return "text/csv", nil
	}

	// Try to detect by MIME package
	if contentType := mime.TypeByExtension(ext); contentType != "" {
		return contentType, nil
	}

	// Return default if not found
	return "application/octet-stream", nil
}

// detectContentTypeFromGzippedFile detects content type by decompressing and reading gzipped file
func detectContentTypeFromGzippedFile(gzippedFilePath string) (string, error) {
	file, err := os.Open(gzippedFilePath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	// Create gzip reader
	gzipReader, err := gzip.NewReader(file)
	if err != nil {
		return "", err
	}
	defer gzipReader.Close()

	// Read first 512 bytes of decompressed content
	buffer := make([]byte, 512)
	n, err := gzipReader.Read(buffer)
	if err != nil && err != io.EOF {
		return "", err
	}

	// Use http.DetectContentType to detect based on decompressed content
	contentType := http.DetectContentType(buffer[:n])
	return contentType, nil
}
