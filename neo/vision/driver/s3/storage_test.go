package s3

import (
	"bytes"
	"context"
	"io"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/yaoapp/yao/config"
	"github.com/yaoapp/yao/test"
)

func TestS3Storage(t *testing.T) {
	test.Prepare(t, config.Conf)
	defer test.Clean()

	t.Run("Create Storage", func(t *testing.T) {
		options := map[string]interface{}{
			"endpoint":   os.Getenv("S3_API"),
			"region":     "auto",
			"key":        os.Getenv("S3_ACCESS_KEY"),
			"secret":     os.Getenv("S3_SECRET_KEY"),
			"bucket":     os.Getenv("S3_BUCKET"),
			"prefix":     "vision-test",
			"expiration": 10 * time.Minute,
		}

		storage, err := New(options)
		if err != nil {
			t.Logf("Error creating storage: %v", err)
		}
		assert.NoError(t, err)
		assert.NotNil(t, storage)
		if storage != nil {
			assert.Equal(t, os.Getenv("S3_API"), storage.Endpoint)
			assert.Equal(t, "auto", storage.Region)
			assert.Equal(t, os.Getenv("S3_ACCESS_KEY"), storage.Key)
			assert.Equal(t, os.Getenv("S3_SECRET_KEY"), storage.Secret)
			assert.Equal(t, os.Getenv("S3_BUCKET"), storage.Bucket)
			assert.Equal(t, "vision-test", storage.prefix)
			assert.Equal(t, 10*time.Minute, storage.Expiration)
		}
	})

	t.Run("Upload and Download", func(t *testing.T) {
		storage, err := New(map[string]interface{}{
			"endpoint":   os.Getenv("S3_API"),
			"region":     "auto",
			"key":        os.Getenv("S3_ACCESS_KEY"),
			"secret":     os.Getenv("S3_SECRET_KEY"),
			"bucket":     os.Getenv("S3_BUCKET"),
			"prefix":     "vision-test",
			"expiration": 5 * time.Minute,
		})
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

		// Test with different expiration
		storage.Expiration = 1 * time.Hour
		url2 := storage.URL(context.Background(), fileID)
		assert.NotEmpty(t, url2)
		assert.Contains(t, url2, "X-Amz-Signature")
		assert.Contains(t, url2, "X-Amz-Expires=3600")

		// Download
		reader2, contentType, err := storage.Download(context.Background(), fileID)
		if err != nil {
			t.Logf("Download error: %v", err)
			t.FailNow()
		}
		assert.NoError(t, err)
		assert.Contains(t, contentType, "text/plain")

		if reader2 != nil {
			downloaded, err := io.ReadAll(reader2)
			assert.NoError(t, err)
			assert.Equal(t, content, downloaded)
			reader2.Close()
		}
	})

	t.Run("Upload with Custom Expiration", func(t *testing.T) {
		storage, err := New(map[string]interface{}{
			"endpoint":   os.Getenv("S3_API"),
			"region":     "auto",
			"key":        os.Getenv("S3_ACCESS_KEY"),
			"secret":     os.Getenv("S3_SECRET_KEY"),
			"bucket":     os.Getenv("S3_BUCKET"),
			"prefix":     "vision-test",
			"expiration": 5 * time.Minute,
		})
		assert.NoError(t, err)

		content := []byte("test content")
		reader := bytes.NewReader(content)
		fileID, err := storage.Upload(context.Background(), "test.txt", reader, "text/plain")
		assert.NoError(t, err)
		assert.NotEmpty(t, fileID)

		// Get URL with default expiration (5 minutes)
		url := storage.URL(context.Background(), fileID)
		assert.NotEmpty(t, url)
		assert.Contains(t, url, "X-Amz-Signature")
		assert.Contains(t, url, "X-Amz-Expires=300") // 5 minutes = 300 seconds

		// Change expiration and get new URL
		storage.Expiration = 2 * time.Hour
		url2 := storage.URL(context.Background(), fileID)
		assert.NotEmpty(t, url2)
		assert.Contains(t, url2, "X-Amz-Signature")
		assert.Contains(t, url2, "X-Amz-Expires=7200") // 2 hours = 7200 seconds
	})

	t.Run("Download Non-existent File", func(t *testing.T) {
		storage, err := New(map[string]interface{}{
			"endpoint":   os.Getenv("S3_API"),
			"region":     "auto",
			"key":        os.Getenv("S3_ACCESS_KEY"),
			"secret":     os.Getenv("S3_SECRET_KEY"),
			"bucket":     os.Getenv("S3_BUCKET"),
			"prefix":     "vision-test",
			"expiration": 5 * time.Minute,
		})
		assert.NoError(t, err)

		_, _, err = storage.Download(context.Background(), "non-existent.txt")
		assert.Error(t, err)
	})
}
