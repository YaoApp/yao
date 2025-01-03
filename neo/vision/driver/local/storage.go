package local

import (
	"context"
	"crypto/sha256"
	"fmt"
	"io"
	"path/filepath"
	"strings"
	"time"

	"github.com/yaoapp/gou/fs"
)

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

	return storage, nil
}

// Upload upload file to local storage
func (storage *Storage) Upload(ctx context.Context, filename string, reader io.Reader, contentType string) (string, error) {
	data, err := fs.Get("data")
	if err != nil {
		return "", err
	}

	ext := filepath.Ext(filename)
	id := storage.makeID(filename, ext)
	path := filepath.Join(storage.Path, id)

	// Create directory if not exists
	dir := filepath.Dir(path)
	if err := data.MkdirAll(dir, 0755); err != nil {
		return "", err
	}

	// Write file
	_, err = data.Write(path, reader, 0644)
	if err != nil {
		return "", err
	}

	return id, nil
}

// Download download file from local storage
func (storage *Storage) Download(ctx context.Context, fileID string) (io.ReadCloser, string, error) {
	data, err := fs.Get("data")
	if err != nil {
		return nil, "", err
	}

	path := filepath.Join(storage.Path, fileID)
	reader, err := data.ReadCloser(path)
	if err != nil {
		return nil, "", err
	}

	contentType := "application/octet-stream"
	if v, err := data.MimeType(path); err == nil {
		contentType = v
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

func (storage *Storage) makeID(filename string, ext string) string {
	date := time.Now().Format("20060102")
	hash := fmt.Sprintf("%x", sha256.Sum256([]byte(filename)))[:8]
	name := strings.TrimSuffix(filepath.Base(filename), ext)
	return fmt.Sprintf("%s/%s-%s%s", date, name, hash, ext)
}
