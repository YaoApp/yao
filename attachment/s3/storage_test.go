package s3

import (
	"bytes"
	"context"
	"io"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func getS3Config() map[string]interface{} {
	return map[string]interface{}{
		"endpoint":    os.Getenv("S3_API"),
		"region":      "auto",
		"key":         os.Getenv("S3_ACCESS_KEY"),
		"secret":      os.Getenv("S3_SECRET_KEY"),
		"bucket":      os.Getenv("S3_BUCKET"),
		"prefix":      "attachment-test",
		"expiration":  5 * time.Minute,
		"compression": true,
	}
}

func skipIfNoS3Config(t *testing.T) {
	if os.Getenv("S3_ACCESS_KEY") == "" || os.Getenv("S3_SECRET_KEY") == "" || os.Getenv("S3_BUCKET") == "" {
		t.Skip("S3 configuration not available (set S3_ACCESS_KEY, S3_SECRET_KEY, S3_BUCKET environment variables)")
	}
}

func TestS3Storage(t *testing.T) {
	t.Run("Create Storage", func(t *testing.T) {
		options := getS3Config()

		storage, err := New(options)
		if os.Getenv("S3_ACCESS_KEY") == "" || os.Getenv("S3_SECRET_KEY") == "" || os.Getenv("S3_BUCKET") == "" {
			// Should fail without credentials
			assert.Error(t, err)
			return
		}

		assert.NoError(t, err)
		assert.NotNil(t, storage)
		if storage != nil {
			assert.Equal(t, os.Getenv("S3_API"), storage.Endpoint)
			assert.Equal(t, "auto", storage.Region)
			assert.Equal(t, os.Getenv("S3_ACCESS_KEY"), storage.Key)
			assert.Equal(t, os.Getenv("S3_SECRET_KEY"), storage.Secret)
			assert.Equal(t, os.Getenv("S3_BUCKET"), storage.Bucket)
			assert.Equal(t, "attachment-test", storage.prefix)
			assert.Equal(t, 5*time.Minute, storage.Expiration)
			assert.True(t, storage.compression)
		}
	})

	t.Run("Upload and Download Text File", func(t *testing.T) {
		skipIfNoS3Config(t)

		storage, err := New(getS3Config())
		assert.NoError(t, err)

		content := []byte("test content")
		reader := bytes.NewReader(content)
		fileID, err := storage.Upload(context.Background(), "test.txt", reader, "text/plain")
		assert.NoError(t, err)
		assert.NotEmpty(t, fileID)

		// Get presigned URL
		url := storage.URL(context.Background(), fileID)
		assert.NotEmpty(t, url)
		assert.Contains(t, url, "X-Amz-Signature")
		assert.Contains(t, url, "X-Amz-Expires")

		// Download
		reader2, contentType, err := storage.Download(context.Background(), fileID)
		assert.NoError(t, err)
		assert.Contains(t, contentType, "text/plain")

		downloaded, err := io.ReadAll(reader2)
		assert.NoError(t, err)
		assert.Equal(t, content, downloaded)
		reader2.Close()

		// Clean up
		storage.Delete(context.Background(), fileID)
	})

	t.Run("Chunked Upload", func(t *testing.T) {
		skipIfNoS3Config(t)

		storage, err := New(getS3Config())
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
		reader.Close()

		// Clean up
		storage.Delete(context.Background(), fileID)
	})

	t.Run("File Operations", func(t *testing.T) {
		skipIfNoS3Config(t)

		storage, err := New(getS3Config())
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

	t.Run("Download Non-existent File", func(t *testing.T) {
		skipIfNoS3Config(t)

		storage, err := New(getS3Config())
		assert.NoError(t, err)

		_, _, err = storage.Download(context.Background(), "non-existent.txt")
		assert.Error(t, err)
	})

	t.Run("Invalid Configuration", func(t *testing.T) {
		// Test with missing required fields
		_, err := New(map[string]interface{}{
			"endpoint": "https://s3.amazonaws.com",
			"region":   "us-east-1",
			// Missing key and secret
		})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "key and secret are required")

		// Test with missing bucket
		_, err = New(map[string]interface{}{
			"endpoint": "https://s3.amazonaws.com",
			"region":   "us-east-1",
			"key":      "test-key",
			"secret":   "test-secret",
			// Missing bucket
		})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "bucket is required")
	})
}
