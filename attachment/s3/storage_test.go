package s3

import (
	"bytes"
	"context"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"compress/gzip"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

// generateTestFileName generates a unique test filename with the given prefix and extension
func generateTestFileName(prefix, ext string) string {
	return prefix + "-" + uuid.New().String() + ext
}

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
		fileID := generateTestFileName("upload-test", ".txt")
		_, err = storage.Upload(context.Background(), fileID, reader, "text/plain")
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

		fileID := generateTestFileName("test-chunked", ".txt")
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

		fileID := generateTestFileName("test-ops", ".txt")
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

	t.Run("Download Non-existent File", func(t *testing.T) {
		skipIfNoS3Config(t)

		storage, err := New(getS3Config())
		assert.NoError(t, err)

		// Use UUID for non-existent file to avoid any potential conflicts
		nonExistentFileID := generateTestFileName("non-existent", ".txt")
		_, _, err = storage.Download(context.Background(), nonExistentFileID)
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

	t.Run("LocalPath", func(t *testing.T) {
		skipIfNoS3Config(t)

		// Create storage with custom cache directory
		tempCacheDir, err := os.MkdirTemp("", "s3_cache_test")
		assert.NoError(t, err)
		defer os.RemoveAll(tempCacheDir)

		config := getS3Config()
		config["cache_dir"] = tempCacheDir

		storage, err := New(config)
		assert.NoError(t, err)

		// Test different file types
		testFiles := []struct {
			name        string
			content     []byte
			contentType string
			expectedCT  string
		}{
			{"localpath-test.txt", []byte("Hello S3 World"), "text/plain", "text/plain"},
			{"localpath-test.json", []byte(`{"s3": "test"}`), "application/json", "application/json"},
			{"localpath-test.html", []byte("<html><body>S3 Test</body></html>"), "text/html", "text/html"},
			{"localpath-test.csv", []byte("s3,test\nval1,val2"), "text/csv", "text/csv"},
			{"localpath-test.md", []byte("# S3 Markdown"), "text/markdown", "text/markdown"},
			{"localpath-test.yao", []byte("s3 yao content"), "application/yao", "application/yao"},
		}

		for _, tf := range testFiles {
			// Upload file to S3
			fileID := generateTestFileName("s3-localpath", "-"+tf.name)
			_, err = storage.Upload(context.Background(), fileID, bytes.NewReader(tf.content), tf.contentType)
			assert.NoError(t, err, "Failed to upload %s", tf.name)

			// Get local path - first call should download to cache
			localPath1, detectedCT1, err := storage.LocalPath(context.Background(), fileID)
			assert.NoError(t, err, "Failed to get local path for %s", tf.name)
			assert.NotEmpty(t, localPath1, "Local path should not be empty for %s", tf.name)
			assert.Equal(t, tf.expectedCT, detectedCT1, "Content type mismatch for %s", tf.name)

			// Verify the path is absolute
			assert.True(t, filepath.IsAbs(localPath1), "Path should be absolute for %s", tf.name)

			// Verify the file exists at the returned path
			_, err = os.Stat(localPath1)
			assert.NoError(t, err, "File should exist at local path for %s", tf.name)

			// Verify file content
			fileContent, err := os.ReadFile(localPath1)
			assert.NoError(t, err, "Failed to read file at local path for %s", tf.name)
			assert.Equal(t, tf.content, fileContent, "File content mismatch for %s", tf.name)

			// Get local path again - should use cached version
			localPath2, detectedCT2, err := storage.LocalPath(context.Background(), fileID)
			assert.NoError(t, err, "Failed to get cached local path for %s", tf.name)
			assert.Equal(t, localPath1, localPath2, "Cached path should be same as first call for %s", tf.name)
			assert.Equal(t, detectedCT1, detectedCT2, "Cached content type should be same as first call for %s", tf.name)

			// Clean up from S3
			storage.Delete(context.Background(), fileID)
		}
	})

	t.Run("LocalPath_GzippedFile", func(t *testing.T) {
		skipIfNoS3Config(t)

		// Create storage with custom cache directory
		tempCacheDir, err := os.MkdirTemp("", "s3_cache_gzip_test")
		assert.NoError(t, err)
		defer os.RemoveAll(tempCacheDir)

		config := getS3Config()
		config["cache_dir"] = tempCacheDir

		storage, err := New(config)
		assert.NoError(t, err)

		// Create gzipped content
		originalContent := []byte("This content will be gzipped and stored in S3")
		var gzipBuf bytes.Buffer
		gzipWriter := gzip.NewWriter(&gzipBuf)
		_, err = gzipWriter.Write(originalContent)
		assert.NoError(t, err)
		gzipWriter.Close()

		// Upload gzipped file
		fileID := generateTestFileName("gzipped", ".txt.gz")
		_, err = storage.Upload(context.Background(), fileID, bytes.NewReader(gzipBuf.Bytes()), "text/plain")
		assert.NoError(t, err)

		// Get local path - should decompress during download
		localPath, contentType, err := storage.LocalPath(context.Background(), fileID)
		assert.NoError(t, err)
		assert.NotEmpty(t, localPath)

		// Verify the file is decompressed in cache (path should not end with .gz)
		assert.False(t, strings.HasSuffix(localPath, ".gz"), "Cached file should be decompressed")

		// Verify content is decompressed
		cachedContent, err := os.ReadFile(localPath)
		assert.NoError(t, err)
		assert.Equal(t, originalContent, cachedContent, "Cached file should contain decompressed content")

		// Verify content type
		assert.Equal(t, "text/plain", contentType)

		// Clean up
		storage.Delete(context.Background(), fileID)
	})

	t.Run("LocalPath_NonExistentFile", func(t *testing.T) {
		skipIfNoS3Config(t)

		storage, err := New(getS3Config())
		assert.NoError(t, err)

		// Test with non-existent file
		nonExistentFileID := generateTestFileName("non-existent-localpath", ".txt")
		_, _, err = storage.LocalPath(context.Background(), nonExistentFileID)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to download file")
	})

	t.Run("LocalPath_CustomCacheDir", func(t *testing.T) {
		skipIfNoS3Config(t)

		// Create custom cache directory
		customCacheDir, err := os.MkdirTemp("", "custom_s3_cache")
		assert.NoError(t, err)
		defer os.RemoveAll(customCacheDir)

		config := getS3Config()
		config["cache_dir"] = customCacheDir

		storage, err := New(config)
		assert.NoError(t, err)

		// Verify cache directory is set correctly
		assert.Equal(t, customCacheDir, storage.CacheDir)

		// Upload a test file
		content := []byte("Custom cache directory test")
		fileID := generateTestFileName("custom-cache", ".txt")
		_, err = storage.Upload(context.Background(), fileID, bytes.NewReader(content), "text/plain")
		assert.NoError(t, err)

		// Get local path
		localPath, contentType, err := storage.LocalPath(context.Background(), fileID)
		assert.NoError(t, err)
		assert.NotEmpty(t, localPath)
		assert.Equal(t, "text/plain", contentType)

		// Verify the file is cached in the custom directory
		assert.True(t, strings.HasPrefix(localPath, customCacheDir), "File should be cached in custom directory")

		// Clean up
		storage.Delete(context.Background(), fileID)
	})
}
