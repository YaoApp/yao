package local

import (
	"bytes"
	"context"
	"crypto/sha256"
	"fmt"
	"image"
	"image/jpeg"
	"image/png"
	"io"
	"path/filepath"
	"strings"
	"time"

	"github.com/yaoapp/gou/fs"
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

	// Check if compression is enabled and if it's an image
	if storage.Compression && isImage(contentType) {
		// Read the entire image into memory
		content, err := io.ReadAll(reader)
		if err != nil {
			return "", fmt.Errorf("failed to read image: %w", err)
		}

		// Compress image
		compressed, err := compressImage(content, contentType)
		if err != nil {
			return "", fmt.Errorf("failed to compress image: %w", err)
		}

		// Write compressed image
		_, err = data.Write(path, bytes.NewReader(compressed), 0644)
		if err != nil {
			return "", err
		}
	} else {
		// Write file without compression
		_, err = data.Write(path, reader, 0644)
		if err != nil {
			return "", err
		}
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

// isImage checks if the content type is an image
func isImage(contentType string) bool {
	return strings.HasPrefix(contentType, "image/")
}

// compressImage compresses the image while maintaining aspect ratio
func compressImage(data []byte, contentType string) ([]byte, error) {
	// Decode image
	img, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("failed to decode image: %w", err)
	}

	// Calculate new dimensions
	bounds := img.Bounds()
	width := bounds.Dx()
	height := bounds.Dy()
	var newWidth, newHeight int

	if width > height {
		if width > MaxImageSize {
			newWidth = MaxImageSize
			newHeight = int(float64(height) * (float64(MaxImageSize) / float64(width)))
		} else {
			return data, nil // No need to resize
		}
	} else {
		if height > MaxImageSize {
			newHeight = MaxImageSize
			newWidth = int(float64(width) * (float64(MaxImageSize) / float64(height)))
		} else {
			return data, nil // No need to resize
		}
	}

	// Create new image with new dimensions
	newImg := image.NewRGBA(image.Rect(0, 0, newWidth, newHeight))

	// Scale the image using bilinear interpolation
	for y := 0; y < newHeight; y++ {
		for x := 0; x < newWidth; x++ {
			srcX := float64(x) * float64(width) / float64(newWidth)
			srcY := float64(y) * float64(height) / float64(newHeight)
			newImg.Set(x, y, img.At(int(srcX), int(srcY)))
		}
	}

	// Encode image
	var buf bytes.Buffer
	switch contentType {
	case "image/jpeg":
		err = jpeg.Encode(&buf, newImg, &jpeg.Options{Quality: 85})
	case "image/png":
		err = png.Encode(&buf, newImg)
	default:
		return data, nil // Unsupported format, return original
	}

	if err != nil {
		return nil, fmt.Errorf("failed to encode image: %w", err)
	}

	return buf.Bytes(), nil
}
