package local

import (
	"bytes"
	"context"
	"io"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/yaoapp/yao/config"
	"github.com/yaoapp/yao/test"
)

func TestLocalStorage(t *testing.T) {
	test.Prepare(t, config.Conf)
	defer test.Clean()

	t.Run("Create Storage", func(t *testing.T) {
		storage, err := New(map[string]interface{}{
			"path":        "/__vision_test",
			"compression": true,
		})
		assert.NoError(t, err)
		assert.NotNil(t, storage)
		assert.Equal(t, "/__vision_test", storage.Path)
		assert.True(t, storage.Compression)
	})

	t.Run("Upload and Download", func(t *testing.T) {
		storage, err := New(map[string]interface{}{
			"path":        "/__vision_test",
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

	t.Run("URL Generation", func(t *testing.T) {
		storage, err := New(map[string]interface{}{
			"path":        "/__vision_test",
			"compression": true,
		})
		assert.NoError(t, err)

		fileID := "20240101/test-12345678.txt"
		url := storage.URL(context.Background(), fileID)
		assert.Equal(t, "/__vision_test/20240101/test-12345678.txt", url)
	})

	t.Run("Download Non-existent File", func(t *testing.T) {
		storage, err := New(map[string]interface{}{
			"path":        "/__vision_test",
			"compression": true,
		})
		assert.NoError(t, err)

		_, _, err = storage.Download(context.Background(), "non-existent.txt")
		assert.Error(t, err)
	})
}
