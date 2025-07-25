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

	"github.com/stretchr/testify/assert"
)

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
		fileID, err := storage.Upload(context.Background(), "test.txt", reader, "text/plain")
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
		fileID, err := storage.Upload(context.Background(), "test.png", reader, "image/png")
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
		fileID, err := storage.Upload(context.Background(), "test.png", reader, "image/png")
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

		// Delete file
		err = storage.Delete(context.Background(), fileID)
		assert.NoError(t, err)

		// Check if file no longer exists
		exists = storage.Exists(context.Background(), fileID)
		assert.False(t, exists)
	})
}
