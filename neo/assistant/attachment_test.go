package assistant

import (
	"bytes"
	"context"
	"encoding/base64"
	"io"
	"mime/multipart"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/yaoapp/gou/fs"
	gourag "github.com/yaoapp/gou/rag"
	"github.com/yaoapp/gou/rag/driver"
	"github.com/yaoapp/yao/config"
	neovision "github.com/yaoapp/yao/neo/vision"
	vdriver "github.com/yaoapp/yao/neo/vision/driver"
	"github.com/yaoapp/yao/test"
)

var (
	// 1x1 transparent PNG for testing
	testImageBase64 = "iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mNk+M9QDwADhgGAWjR9awAAAABJRU5ErkJggg=="
)

func TestUpload(t *testing.T) {
	test.Prepare(t, config.Conf)
	defer test.Clean()

	ast := setupTestAssistant()
	ctx := context.Background()

	t.Run("Basic File Upload", func(t *testing.T) {
		content := []byte("test content")
		file := &multipart.FileHeader{
			Filename: "test.txt",
			Size:     int64(len(content)),
		}
		file.Header = make(map[string][]string)
		file.Header.Set("Content-Type", "text/plain")

		reader := bytes.NewReader(content)
		fileResp, err := ast.Upload(ctx, file, reader, map[string]interface{}{
			"sid":     "test-user",
			"chat_id": "test-chat",
		})

		assert.NoError(t, err)
		assert.NotNil(t, fileResp)
		assert.Contains(t, fileResp.ID, "test-assistant/test-user/test-chat")
		assert.Equal(t, len(content), fileResp.Bytes)
		assert.Equal(t, "text/plain", fileResp.ContentType)
	})

	t.Run("File Size Limit", func(t *testing.T) {
		content := make([]byte, MaxSize+1)
		file := &multipart.FileHeader{
			Filename: "large.txt",
			Size:     int64(len(content)),
		}
		file.Header = make(map[string][]string)
		file.Header.Set("Content-Type", "text/plain")

		reader := bytes.NewReader(content)
		_, err := ast.Upload(ctx, file, reader, nil)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "exceeds the maximum size")
	})

	t.Run("Invalid Content Type", func(t *testing.T) {
		content := []byte("test")
		file := &multipart.FileHeader{
			Filename: "test.invalid",
			Size:     int64(len(content)),
		}
		file.Header = make(map[string][]string)
		file.Header.Set("Content-Type", "invalid/type")

		reader := bytes.NewReader(content)
		_, err := ast.Upload(ctx, file, reader, nil)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not allowed")
	})
}

func TestUploadWithRAG(t *testing.T) {
	test.Prepare(t, config.Conf)
	defer test.Clean()

	ast := setupTestAssistant()
	ragEngine, ragUploader, ragVectorizer := setupTestRAG(t)
	SetRAG(ragEngine, ragUploader, ragVectorizer, RAGSetting{IndexPrefix: "test_"})
	defer func() {
		rag = nil
	}()
	ctx := context.Background()

	t.Run("Text File with RAG Enabled", func(t *testing.T) {
		content := []byte("This is a test document for RAG indexing")
		file := &multipart.FileHeader{
			Filename: "test.txt",
			Size:     int64(len(content)),
		}
		file.Header = make(map[string][]string)
		file.Header.Set("Content-Type", "text/plain")

		reader := bytes.NewReader(content)
		fileResp, err := ast.Upload(ctx, file, reader, map[string]interface{}{
			"sid":     "test-user",
			"chat_id": "test-chat",
			"rag":     true,
		})

		assert.NoError(t, err)
		assert.NotNil(t, fileResp)
		assert.NotEmpty(t, fileResp.DocIDs, "Document IDs should not be empty")

		// Wait for indexing to complete
		time.Sleep(500 * time.Millisecond)

		// Verify the file was indexed
		exists, err := ragEngine.HasDocument(ctx, "test_test-assistant-test-user-test-chat", fileResp.DocIDs[0])
		assert.NoError(t, err)
		assert.True(t, exists, "Document should exist in RAG index")
	})

	t.Run("Text File with RAG Disabled", func(t *testing.T) {
		content := []byte("This is a test document with RAG disabled")
		file := &multipart.FileHeader{
			Filename: "test.txt",
			Size:     int64(len(content)),
		}
		file.Header = make(map[string][]string)
		file.Header.Set("Content-Type", "text/plain")

		reader := bytes.NewReader(content)
		fileResp, err := ast.Upload(ctx, file, reader, map[string]interface{}{
			"sid":     "test-user",
			"chat_id": "test-chat",
			"rag":     false,
		})

		assert.NoError(t, err)
		assert.NotNil(t, fileResp)
		assert.Empty(t, fileResp.DocIDs, "Document IDs should be empty when RAG is disabled")
	})
}

func setupTestAssistant() *Assistant {
	ast := &Assistant{
		ID:        "test-assistant",
		Name:      "Test Assistant",
		Connector: "test-connector",
	}
	return ast
}

func setupTestRAG(t *testing.T) (driver.Engine, driver.FileUpload, driver.Vectorizer) {
	// Get test config
	openaiKey := os.Getenv("OPENAI_API_KEY")
	if openaiKey == "" {
		t.Skip("OPENAI_API_KEY not set")
	}

	vectorizeConfig := driver.VectorizeConfig{
		Model: os.Getenv("VECTORIZER_MODEL"),
		Options: map[string]string{
			"api_key": openaiKey,
		},
	}

	// Qdrant config
	host := os.Getenv("QDRANT_HOST")
	if host == "" {
		host = "localhost"
	}

	port := os.Getenv("QDRANT_PORT")
	if port == "" {
		port = "6334"
	}

	// Create vectorizer
	vectorizer, err := gourag.NewVectorizer(gourag.DriverOpenAI, vectorizeConfig)
	if err != nil {
		t.Fatal(err)
	}

	// Create engine
	engine, err := gourag.NewEngine(gourag.DriverQdrant, driver.IndexConfig{
		Options: map[string]string{
			"host":    host,
			"port":    port,
			"api_key": "",
		},
	}, vectorizer)
	if err != nil {
		t.Fatal(err)
	}

	// Create file upload
	fileUpload, err := gourag.NewFileUpload(gourag.DriverQdrant, engine, vectorizer)
	if err != nil {
		t.Fatal(err)
	}

	return engine, fileUpload, vectorizer
}

func setupTestVision(t *testing.T) *neovision.Vision {
	// Create test data directory
	data, err := fs.Get("data")
	assert.NoError(t, err)

	// Write test image data
	imgData, err := base64.StdEncoding.DecodeString(testImageBase64)
	assert.NoError(t, err)
	_, err = data.WriteFile("/test.png", imgData, 0644)
	assert.NoError(t, err)

	cfg := &vdriver.Config{
		Storage: vdriver.StorageConfig{
			Driver: "local",
			Options: map[string]interface{}{
				"path":        "/__vision_test",
				"compression": true,
			},
		},
		Model: vdriver.ModelConfig{
			Driver: "openai",
			Options: map[string]interface{}{
				"api_key": os.Getenv("OPENAI_API_KEY"),
				"model":   os.Getenv("VISION_MODEL"),
			},
		},
	}

	v, err := neovision.New(cfg)
	if err != nil {
		t.Fatal(err)
	}
	return v
}

func TestUploadWithVision(t *testing.T) {
	test.Prepare(t, config.Conf)
	defer test.Clean()

	ast := setupTestAssistant()
	vision := setupTestVision(t)
	SetVision(vision)
	defer func() {
		vision = nil
	}()
	ctx := context.Background()

	t.Run("Image File with Vision Enabled", func(t *testing.T) {
		imgData, _ := base64.StdEncoding.DecodeString(testImageBase64)
		file := &multipart.FileHeader{
			Filename: "test.png",
			Size:     int64(len(imgData)),
		}
		file.Header = make(map[string][]string)
		file.Header.Set("Content-Type", "image/png")

		reader := bytes.NewReader(imgData)
		fileResp, err := ast.Upload(ctx, file, reader, map[string]interface{}{
			"vision": true,
			"model":  "gpt-4-vision-preview",
		})

		assert.NoError(t, err)
		assert.NotNil(t, fileResp)
		if fileResp.URL == "" && fileResp.Description == "" {
			t.Error("Either URL or Description should be set when vision is enabled")
		}
	})

	t.Run("Image File with Vision Disabled", func(t *testing.T) {
		imgData, _ := base64.StdEncoding.DecodeString(testImageBase64)
		file := &multipart.FileHeader{
			Filename: "test.png",
			Size:     int64(len(imgData)),
		}
		file.Header = make(map[string][]string)
		file.Header.Set("Content-Type", "image/png")

		reader := bytes.NewReader(imgData)
		fileResp, err := ast.Upload(ctx, file, reader, map[string]interface{}{
			"vision": false,
		})

		assert.NoError(t, err)
		assert.NotNil(t, fileResp)
		assert.Empty(t, fileResp.URL, "Vision URL should be empty when vision is disabled")
		assert.Empty(t, fileResp.Description, "Vision Description should be empty when vision is disabled")
	})

	t.Run("Image File with Non-Vision Model", func(t *testing.T) {
		imgData, _ := base64.StdEncoding.DecodeString(testImageBase64)
		file := &multipart.FileHeader{
			Filename: "test.png",
			Size:     int64(len(imgData)),
		}
		file.Header = make(map[string][]string)
		file.Header.Set("Content-Type", "image/png")

		reader := bytes.NewReader(imgData)
		fileResp, err := ast.Upload(ctx, file, reader, map[string]interface{}{
			"vision": true,
			"model":  "gpt-4",
		})

		assert.NoError(t, err)
		assert.NotNil(t, fileResp)
		assert.Empty(t, fileResp.URL, "Vision URL should be empty for non-vision models")
		assert.NotEmpty(t, fileResp.Description, "Vision Description should be set for non-vision models")
	})
}

func TestDownload(t *testing.T) {
	test.Prepare(t, config.Conf)
	defer test.Clean()

	ast := setupTestAssistant()
	ctx := context.Background()

	t.Run("Download Existing File", func(t *testing.T) {
		// First upload a file
		content := []byte("test content")
		file := &multipart.FileHeader{
			Filename: "test.txt",
			Size:     int64(len(content)),
		}
		file.Header = make(map[string][]string)
		file.Header.Set("Content-Type", "text/plain")

		reader := bytes.NewReader(content)
		fileResp, err := ast.Upload(ctx, file, reader, nil)
		assert.NoError(t, err)

		// Then download it
		downloadResp, err := ast.Download(ctx, fileResp.ID)
		assert.NoError(t, err)
		assert.NotNil(t, downloadResp)
		assert.True(t, strings.HasPrefix(downloadResp.ContentType, "text/plain"), "Content-Type should start with text/plain")
		assert.Equal(t, ".txt", downloadResp.Extension)

		// Verify content
		downloaded, err := io.ReadAll(downloadResp.Reader)
		assert.NoError(t, err)
		assert.Equal(t, content, downloaded)
	})

	t.Run("Download Non-Existent File", func(t *testing.T) {
		_, err := ast.Download(ctx, "non-existent-file")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not found")
	})
}
