package local

import (
	"compress/gzip"
	"context"
	"crypto/sha256"
	"fmt"
	"io"
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
func (storage *Storage) Upload(ctx context.Context, fileID string, reader io.Reader, contentType string) (string, error) {
	path := filepath.Join(storage.Path, fileID)

	// Create directory if not exists
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", err
	}

	// Create and write file
	file, err := os.Create(path)
	if err != nil {
		return "", err
	}
	defer file.Close()

	_, err = io.Copy(file, reader)
	if err != nil {
		return "", err
	}

	return fileID, nil
}

// UploadChunk uploads a chunk of a file
func (storage *Storage) UploadChunk(ctx context.Context, fileID string, chunkIndex int, reader io.Reader, contentType string) error {
	// Create chunks directory
	chunksDir := filepath.Join(storage.Path, ".chunks", fileID)
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
func (storage *Storage) MergeChunks(ctx context.Context, fileID string, totalChunks int) error {
	chunksDir := filepath.Join(storage.Path, ".chunks", fileID)
	finalPath := filepath.Join(storage.Path, fileID)

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
func (storage *Storage) Reader(ctx context.Context, fileID string) (io.ReadCloser, error) {
	fullpath := filepath.Join(storage.Path, fileID)

	reader, err := os.Open(fullpath)
	if err != nil {
		return nil, err
	}

	// If the file is a gzip file, decompress it
	if strings.HasSuffix(fileID, ".gz") {
		reader, err := gzip.NewReader(reader)
		if err != nil {
			return nil, err
		}
		return reader, nil
	}

	return reader, nil
}

// Download download file from local storage
func (storage *Storage) Download(ctx context.Context, fileID string) (io.ReadCloser, string, error) {
	path := filepath.Join(storage.Path, fileID)
	reader, err := os.Open(path)
	if err != nil {
		return nil, "", err
	}

	// Try to detect content type from file extension
	contentType := "application/octet-stream"
	ext := filepath.Ext(strings.TrimSuffix(fileID, ".gz"))
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
	if strings.HasSuffix(fileID, ".gz") {
		reader, err := gzip.NewReader(reader)
		if err != nil {
			return nil, "", err
		}
		return reader, contentType, nil
	}

	return reader, contentType, nil
}

// URL get file url
func (storage *Storage) URL(ctx context.Context, fileID string) string {
	if storage.PreviewURL != nil {
		return storage.PreviewURL(fileID)
	}
	if storage.BaseURL != "" {
		return fmt.Sprintf("%s/%s", strings.TrimRight(storage.BaseURL, "/"), fileID)
	}
	return fmt.Sprintf("%s/%s", storage.Path, fileID)
}

// Exists checks if a file exists
func (storage *Storage) Exists(ctx context.Context, fileID string) bool {
	fullpath := filepath.Join(storage.Path, fileID)
	_, err := os.Stat(fullpath)
	return err == nil
}

// Delete deletes a file
func (storage *Storage) Delete(ctx context.Context, fileID string) error {
	fullpath := filepath.Join(storage.Path, fileID)
	return os.Remove(fullpath)
}

func (storage *Storage) makeID(filename string, ext string) string {
	date := time.Now().Format("20060102")
	hash := fmt.Sprintf("%x", sha256.Sum256([]byte(filename)))[:8]
	name := strings.TrimSuffix(filepath.Base(filename), ext)
	return fmt.Sprintf("%s/%s-%s%s", date, name, hash, ext)
}
