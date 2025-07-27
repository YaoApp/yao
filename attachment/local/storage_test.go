package local

import (
	"bytes"
	"context"
	"image"
	"image/png"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

// generateTestFileName generates a unique test filename with the given prefix and extension
func generateTestFileName(prefix, ext string) string {
	return prefix + "-" + uuid.New().String() + ext
}

func TestLocalStorage(t *testing.T) {
	// Create a temporary directory for testing
	tempDir, err := os.MkdirTemp("", "local_storage_test")
	assert.NoError(t, err)
	defer os.RemoveAll(tempDir)

	testPath := filepath.Join(tempDir, "test_storage")

	t.Run("Create Storage", func(t *testing.T) {
		storage, err := New(map[string]interface{}{
			"path":        testPath,
			"compression": true,
		})
		assert.NoError(t, err)
		assert.NotNil(t, storage)
		assert.Equal(t, testPath, storage.Path)
		assert.True(t, storage.Compression)
	})

	t.Run("Upload and Download", func(t *testing.T) {
		storage, err := New(map[string]interface{}{
			"path":        testPath,
			"compression": true,
		})
		assert.NoError(t, err)

		content := []byte("test content")
		reader := bytes.NewReader(content)
		fileID := generateTestFileName("upload-download", ".txt")
		_, err = storage.Upload(context.Background(), fileID, reader, "text/plain")
		assert.NoError(t, err)
		assert.NotEmpty(t, fileID)

		// Download
		reader2, contentType, err := storage.Download(context.Background(), fileID)
		assert.NoError(t, err)
		assert.Contains(t, contentType, "text/plain")

		downloaded, err := io.ReadAll(reader2)
		assert.NoError(t, err)
		assert.Equal(t, content, downloaded)
	})

	t.Run("Upload and Download Image with Compression", func(t *testing.T) {
		storage, err := New(map[string]interface{}{
			"path":        testPath,
			"compression": true,
		})
		assert.NoError(t, err)

		// Create a test image (100x100 pixels - smaller for faster testing)
		img := image.NewRGBA(image.Rect(0, 0, 100, 100))
		var buf bytes.Buffer
		err = png.Encode(&buf, img)
		assert.NoError(t, err)

		// Upload
		reader := bytes.NewReader(buf.Bytes())
		fileID := generateTestFileName("image-with-compression", ".png")
		_, err = storage.Upload(context.Background(), fileID, reader, "image/png")
		assert.NoError(t, err)
		assert.NotEmpty(t, fileID)

		// Download and verify
		reader2, contentType, err := storage.Download(context.Background(), fileID)
		assert.NoError(t, err)
		assert.Equal(t, "image/png", contentType)

		downloaded, err := io.ReadAll(reader2)
		assert.NoError(t, err)

		// Decode the downloaded image
		downloadedImg, _, err := image.Decode(bytes.NewReader(downloaded))
		assert.NoError(t, err)

		// Verify image was processed
		bounds := downloadedImg.Bounds()
		assert.True(t, bounds.Dx() > 0)
		assert.True(t, bounds.Dy() > 0)
	})

	t.Run("Upload Image without Compression", func(t *testing.T) {
		storage, err := New(map[string]interface{}{
			"path":        testPath,
			"compression": false,
		})
		assert.NoError(t, err)

		// Create a test image (100x100 pixels)
		img := image.NewRGBA(image.Rect(0, 0, 100, 100))
		var buf bytes.Buffer
		err = png.Encode(&buf, img)
		assert.NoError(t, err)

		// Upload
		reader := bytes.NewReader(buf.Bytes())
		fileID := generateTestFileName("image-without-compression", ".png")
		_, err = storage.Upload(context.Background(), fileID, reader, "image/png")
		assert.NoError(t, err)
		assert.NotEmpty(t, fileID)

		// Download and verify
		reader2, contentType, err := storage.Download(context.Background(), fileID)
		assert.NoError(t, err)
		assert.Equal(t, "image/png", contentType)

		downloaded, err := io.ReadAll(reader2)
		assert.NoError(t, err)

		// Decode the downloaded image
		downloadedImg, _, err := image.Decode(bytes.NewReader(downloaded))
		assert.NoError(t, err)

		// Verify dimensions are unchanged
		bounds := downloadedImg.Bounds()
		assert.Equal(t, 100, bounds.Dx())
		assert.Equal(t, 100, bounds.Dy())
	})

	t.Run("URL Generation", func(t *testing.T) {
		storage, err := New(map[string]interface{}{
			"path":        testPath,
			"compression": true,
		})
		assert.NoError(t, err)

		fileID := "20240101/test-12345678.txt"
		url := storage.URL(context.Background(), fileID)
		expected := filepath.Join(testPath, fileID)
		assert.Equal(t, expected, url)
	})

	t.Run("Download Non-existent File", func(t *testing.T) {
		storage, err := New(map[string]interface{}{
			"path":        testPath,
			"compression": true,
		})
		assert.NoError(t, err)

		_, _, err = storage.Download(context.Background(), "non-existent.txt")
		assert.Error(t, err)
	})

	t.Run("Chunked Upload", func(t *testing.T) {
		storage, err := New(map[string]interface{}{
			"path": testPath,
		})
		assert.NoError(t, err)

		fileID := "test-chunked.txt"
		content1 := []byte("chunk1")
		content2 := []byte("chunk2")

		// Upload chunks
		err = storage.UploadChunk(context.Background(), fileID, 0, bytes.NewReader(content1), "text/plain")
		assert.NoError(t, err)

		err = storage.UploadChunk(context.Background(), fileID, 1, bytes.NewReader(content2), "text/plain")
		assert.NoError(t, err)

		// Merge chunks
		err = storage.MergeChunks(context.Background(), fileID, 2)
		assert.NoError(t, err)

		// Download and verify
		reader, contentType, err := storage.Download(context.Background(), fileID)
		assert.NoError(t, err)
		assert.Equal(t, "text/plain", contentType)

		downloaded, err := io.ReadAll(reader)
		assert.NoError(t, err)
		assert.Equal(t, append(content1, content2...), downloaded)
	})

	t.Run("File Operations", func(t *testing.T) {
		storage, err := New(map[string]interface{}{
			"path": testPath,
		})
		assert.NoError(t, err)

		fileID := "test-ops.txt"
		content := []byte("test content")

		// Upload file
		_, err = storage.Upload(context.Background(), fileID, bytes.NewReader(content), "text/plain")
		assert.NoError(t, err)

		// Check if file exists
		exists := storage.Exists(context.Background(), fileID)
		assert.True(t, exists)

		// Read file
		reader, err := storage.Reader(context.Background(), fileID)
		assert.NoError(t, err)
		defer reader.Close()

		data, err := io.ReadAll(reader)
		assert.NoError(t, err)
		assert.Equal(t, content, data)

		// Get file content directly
		directContent, err := storage.GetContent(context.Background(), fileID)
		assert.NoError(t, err)
		assert.Equal(t, content, directContent)

		// Delete file
		err = storage.Delete(context.Background(), fileID)
		assert.NoError(t, err)

		// Check if file no longer exists
		exists = storage.Exists(context.Background(), fileID)
		assert.False(t, exists)
	})

	t.Run("LocalPath", func(t *testing.T) {
		storage, err := New(map[string]interface{}{
			"path": testPath,
		})
		assert.NoError(t, err)

		// Test different file types to verify content type detection
		testFiles := []struct {
			ext         string
			content     []byte
			contentType string
			expectedCT  string
		}{
			{".txt", []byte("Hello World"), "text/plain", "text/plain"},
			{".json", []byte(`{"key": "value"}`), "application/json", "application/json"},
			{".html", []byte("<html><body>Test</body></html>"), "text/html", "text/html"},
			{".csv", []byte("col1,col2\nval1,val2"), "text/csv", "text/csv"},
			{".md", []byte("# Markdown Content"), "text/markdown", "text/markdown"},
			{".yao", []byte("yao file content"), "application/yao", "application/yao"},
		}

		for _, tf := range testFiles {
			// Generate unique filename with UUID to avoid conflicts
			fileName := generateTestFileName("localpath-test", tf.ext)

			// Upload file
			_, err = storage.Upload(context.Background(), fileName, bytes.NewReader(tf.content), tf.contentType)
			assert.NoError(t, err, "Failed to upload %s", fileName)

			// Get local path and content type
			localPath, detectedCT, err := storage.LocalPath(context.Background(), fileName)
			assert.NoError(t, err, "Failed to get local path for %s", fileName)
			assert.NotEmpty(t, localPath, "Local path should not be empty for %s", fileName)
			assert.Equal(t, tf.expectedCT, detectedCT, "Content type mismatch for %s", fileName)

			// Verify the path is absolute
			assert.True(t, filepath.IsAbs(localPath), "Path should be absolute for %s", fileName)

			// Verify the file exists at the returned path
			_, err = os.Stat(localPath)
			assert.NoError(t, err, "File should exist at local path for %s", fileName)

			// Verify file content
			fileContent, err := os.ReadFile(localPath)
			assert.NoError(t, err, "Failed to read file at local path for %s", fileName)
			assert.Equal(t, tf.content, fileContent, "File content mismatch for %s", fileName)
		}
	})

	t.Run("LocalPath_NonExistentFile", func(t *testing.T) {
		storage, err := New(map[string]interface{}{
			"path": testPath,
		})
		assert.NoError(t, err)

		// Test with non-existent file
		_, _, err = storage.LocalPath(context.Background(), "non-existent.txt")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "file not found")
	})

	t.Run("LocalPath_ContentDetection", func(t *testing.T) {
		storage, err := New(map[string]interface{}{
			"path": testPath,
		})
		assert.NoError(t, err)

		// Upload a file without extension but with recognizable content
		htmlContent := []byte("<!DOCTYPE html><html><head><title>Test</title></head><body><h1>Hello</h1></body></html>")
		_, err = storage.Upload(context.Background(), "noext", bytes.NewReader(htmlContent), "application/octet-stream")
		assert.NoError(t, err)

		// Get local path - should detect HTML content type
		localPath, contentType, err := storage.LocalPath(context.Background(), "noext")
		assert.NoError(t, err)
		assert.NotEmpty(t, localPath)
		// Content detection should identify this as HTML
		assert.Equal(t, "text/html; charset=utf-8", contentType)
	})
}
