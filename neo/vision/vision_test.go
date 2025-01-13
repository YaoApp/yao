package vision

import (
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"image"
	"image/png"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/yaoapp/gou/fs"
	"github.com/yaoapp/yao/config"
	"github.com/yaoapp/yao/neo/vision/driver"
	"github.com/yaoapp/yao/neo/vision/driver/local"
	"github.com/yaoapp/yao/test"
)

var (
	// 1x1 transparent PNG
	testImageBase64 = "iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mNk+M9QDwADhgGAWjR9awAAAABJRU5ErkJggg=="
)

// MaxImageSize maximum image size (1920x1080)
const MaxImageSize = local.MaxImageSize

func TestVision(t *testing.T) {
	test.Prepare(t, config.Conf)
	defer test.Clean()

	// Setup test server for image hosting
	imgServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Log request for debugging
		t.Logf("Received request for: %s", r.URL.Path)

		// Always return the test image
		imgData, _ := base64.StdEncoding.DecodeString(testImageBase64)
		w.Header().Set("Content-Type", "image/png")
		w.Write(imgData)
	}))
	defer imgServer.Close()

	t.Logf("Test server running at: %s", imgServer.URL)

	t.Run("Create Vision Service", func(t *testing.T) {
		vision, err := createTestVision(imgServer.URL)
		assert.NoError(t, err)
		assert.NotNil(t, vision)
	})

	t.Run("Upload and Download with Local Storage", func(t *testing.T) {
		vision, err := createTestVision(imgServer.URL)
		assert.NoError(t, err)

		// Test with text file
		content := []byte("test content")
		reader := bytes.NewReader(content)
		resp, err := vision.Upload(context.Background(), "test.txt", reader, "text/plain")
		assert.NoError(t, err)
		assert.NotEmpty(t, resp.FileID)
		assert.NotEmpty(t, resp.URL)

		// Download
		reader2, contentType, err := vision.Download(context.Background(), resp.FileID)
		assert.NoError(t, err)
		assert.Contains(t, contentType, "text/plain")

		if reader2 != nil {
			downloaded, err := io.ReadAll(reader2)
			assert.NoError(t, err)
			assert.Equal(t, content, downloaded)
			reader2.Close()
		}
	})

	t.Run("Upload and Download with S3 Storage", func(t *testing.T) {
		vision, err := createTestVisionWithS3()
		if err != nil {
			t.Skip("S3 configuration not available")
		}

		// Test with text file
		content := []byte("test content")
		reader := bytes.NewReader(content)
		resp, err := vision.Upload(context.Background(), "test.txt", reader, "text/plain")
		assert.NoError(t, err)
		assert.NotEmpty(t, resp.FileID)
		assert.NotEmpty(t, resp.URL)

		// Download
		reader2, contentType, err := vision.Download(context.Background(), resp.FileID)
		assert.NoError(t, err)
		assert.Contains(t, contentType, "text/plain")

		if reader2 != nil {
			downloaded, err := io.ReadAll(reader2)
			assert.NoError(t, err)
			assert.Equal(t, content, downloaded)
			reader2.Close()
		}
	})

	t.Run("Analyze Image with Base64", func(t *testing.T) {
		// Create vision service
		cfg := &driver.Config{
			Storage: driver.StorageConfig{
				Driver: "local",
				Options: map[string]interface{}{
					"path":        "/__vision_test",
					"compression": true,
				},
			},
			Model: driver.ModelConfig{
				Driver: "openai",
				Options: map[string]interface{}{
					"api_key": os.Getenv("OPENAI_API_KEY"),
					"model":   os.Getenv("VISION_MODEL"),
				},
			},
		}

		vision, err := New(cfg)
		assert.NoError(t, err)

		// Use base64 data directly
		result, err := vision.Analyze(context.Background(), "data:image/png;base64,"+testImageBase64, "Describe this image in detail")
		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.NotEmpty(t, result.Description)
	})

	t.Run("Analyze Image with File", func(t *testing.T) {
		// Create vision service
		cfg := &driver.Config{
			Storage: driver.StorageConfig{
				Driver: "local",
				Options: map[string]interface{}{
					"path":        "/__vision_test",
					"compression": true,
				},
			},
			Model: driver.ModelConfig{
				Driver: "openai",
				Options: map[string]interface{}{
					"api_key": os.Getenv("OPENAI_API_KEY"),
					"model":   os.Getenv("VISION_MODEL"),
				},
			},
		}

		vision, err := New(cfg)
		assert.NoError(t, err)

		// Create test file
		data, err := fs.Get("data")
		assert.NoError(t, err)

		// Write test image data
		imgData, err := base64.StdEncoding.DecodeString(testImageBase64)
		assert.NoError(t, err)
		_, err = data.WriteFile("/test.png", imgData, 0644)
		assert.NoError(t, err)

		// Analyze using file path
		result, err := vision.Analyze(context.Background(), "/test.png", "Describe this image in detail")
		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.NotEmpty(t, result.Description)
	})

	t.Run("Analyze Image with S3 URL", func(t *testing.T) {
		if os.Getenv("S3_API") == "" || os.Getenv("S3_ACCESS_KEY") == "" ||
			os.Getenv("S3_SECRET_KEY") == "" || os.Getenv("S3_BUCKET") == "" {
			t.Skip("S3 environment variables not set")
		}

		// Create vision service
		cfg := &driver.Config{
			Storage: driver.StorageConfig{
				Driver: "s3",
				Options: map[string]interface{}{
					"endpoint":   os.Getenv("S3_API"),
					"region":     "auto",
					"key":        os.Getenv("S3_ACCESS_KEY"),
					"secret":     os.Getenv("S3_SECRET_KEY"),
					"bucket":     os.Getenv("S3_BUCKET"),
					"prefix":     "vision-test",
					"expiration": "5m",
				},
			},
			Model: driver.ModelConfig{
				Driver: "openai",
				Options: map[string]interface{}{
					"api_key": os.Getenv("OPENAI_API_KEY"),
					"model":   os.Getenv("VISION_MODEL"),
				},
			},
		}

		vision, err := New(cfg)
		assert.NoError(t, err)

		// Upload test image
		imgData, err := base64.StdEncoding.DecodeString(testImageBase64)
		assert.NoError(t, err)
		reader := bytes.NewReader(imgData)
		resp, err := vision.Upload(context.Background(), "test.png", reader, "image/png")
		assert.NoError(t, err)
		assert.NotEmpty(t, resp.FileID)
		assert.NotEmpty(t, resp.URL)

		// Analyze using S3 URL
		result, err := vision.Analyze(context.Background(), resp.URL, "Describe this image in detail")
		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.NotEmpty(t, result.Description)
	})

	t.Run("Invalid Model", func(t *testing.T) {
		cfg := &driver.Config{
			Storage: driver.StorageConfig{
				Driver: "local",
				Options: map[string]interface{}{
					"path":        "/__vision_test",
					"compression": true,
				},
			},
			Model: driver.ModelConfig{
				Driver:  "invalid",
				Options: map[string]interface{}{},
			},
		}

		_, err := New(cfg)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "model driver invalid not supported")
	})

	t.Run("Invalid Storage", func(t *testing.T) {
		cfg := &driver.Config{
			Storage: driver.StorageConfig{
				Driver:  "invalid",
				Options: map[string]interface{}{},
			},
			Model: driver.ModelConfig{
				Driver: "openai",
				Options: map[string]interface{}{
					"api_key": "test",
				},
			},
		}

		_, err := New(cfg)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "storage driver invalid not supported")
	})

	t.Run("Upload and Download Image with Local Storage", func(t *testing.T) {
		vision, err := createTestVision(imgServer.URL)
		assert.NoError(t, err)

		// Create test image (2000x2000 pixels)
		img := image.NewRGBA(image.Rect(0, 0, 2000, 2000))
		var buf bytes.Buffer
		err = png.Encode(&buf, img)
		assert.NoError(t, err)

		// Upload
		reader := bytes.NewReader(buf.Bytes())
		resp, err := vision.Upload(context.Background(), "test.png", reader, "image/png")
		assert.NoError(t, err)
		assert.NotEmpty(t, resp.FileID)
		assert.NotEmpty(t, resp.URL)

		// Download and verify size
		reader2, contentType, err := vision.Download(context.Background(), resp.FileID)
		assert.NoError(t, err)
		assert.Equal(t, "image/png", contentType)

		downloaded, err := io.ReadAll(reader2)
		assert.NoError(t, err)

		// Decode the downloaded image
		downloadedImg, _, err := image.Decode(bytes.NewReader(downloaded))
		assert.NoError(t, err)

		// Verify dimensions
		bounds := downloadedImg.Bounds()
		assert.LessOrEqual(t, bounds.Dx(), MaxImageSize)
		assert.LessOrEqual(t, bounds.Dy(), MaxImageSize)
	})

	t.Run("Upload and Download Image with S3 Storage", func(t *testing.T) {
		vision, err := createTestVisionWithS3()
		if err != nil {
			t.Skip("S3 configuration not available")
		}

		// Create test image (2000x2000 pixels)
		img := image.NewRGBA(image.Rect(0, 0, 2000, 2000))
		var buf bytes.Buffer
		err = png.Encode(&buf, img)
		assert.NoError(t, err)

		// Upload
		reader := bytes.NewReader(buf.Bytes())
		resp, err := vision.Upload(context.Background(), "test.png", reader, "image/png")
		assert.NoError(t, err)
		assert.NotEmpty(t, resp.FileID)
		assert.NotEmpty(t, resp.URL)

		// Download and verify size
		reader2, contentType, err := vision.Download(context.Background(), resp.FileID)
		assert.NoError(t, err)
		assert.Equal(t, "image/png", contentType)

		downloaded, err := io.ReadAll(reader2)
		assert.NoError(t, err)

		// Decode the downloaded image
		downloadedImg, _, err := image.Decode(bytes.NewReader(downloaded))
		assert.NoError(t, err)

		// Verify dimensions
		bounds := downloadedImg.Bounds()
		assert.LessOrEqual(t, bounds.Dx(), MaxImageSize)
		assert.LessOrEqual(t, bounds.Dy(), MaxImageSize)
	})

	t.Run("Analyze Image with Default Prompt", func(t *testing.T) {
		// Create vision service with default prompt
		cfg := &driver.Config{
			Storage: driver.StorageConfig{
				Driver: "local",
				Options: map[string]interface{}{
					"path":        "/__vision_test",
					"compression": true,
				},
			},
			Model: driver.ModelConfig{
				Driver: "openai",
				Options: map[string]interface{}{
					"api_key": os.Getenv("OPENAI_API_KEY"),
					"model":   os.Getenv("VISION_MODEL"),
					"prompt":  "Default test prompt",
				},
			},
		}

		vision, err := New(cfg)
		assert.NoError(t, err)

		// Use base64 data without providing a prompt
		result, err := vision.Analyze(context.Background(), "data:image/png;base64,"+testImageBase64)
		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.NotEmpty(t, result.Description)
	})

	t.Run("Analyze Image with Custom Prompt", func(t *testing.T) {
		// Create vision service with default prompt
		cfg := &driver.Config{
			Storage: driver.StorageConfig{
				Driver: "local",
				Options: map[string]interface{}{
					"path":        "/__vision_test",
					"compression": true,
				},
			},
			Model: driver.ModelConfig{
				Driver: "openai",
				Options: map[string]interface{}{
					"api_key": os.Getenv("OPENAI_API_KEY"),
					"model":   os.Getenv("VISION_MODEL"),
					"prompt":  "Default test prompt",
				},
			},
		}

		vision, err := New(cfg)
		assert.NoError(t, err)

		// Use base64 data with custom prompt
		result, err := vision.Analyze(context.Background(), "data:image/png;base64,"+testImageBase64, "Custom test prompt")
		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.NotEmpty(t, result.Description)
	})

	t.Run("Analyze Image with Empty Custom Prompt", func(t *testing.T) {
		// Create vision service with default prompt
		cfg := &driver.Config{
			Storage: driver.StorageConfig{
				Driver: "local",
				Options: map[string]interface{}{
					"path":        "/__vision_test",
					"compression": true,
				},
			},
			Model: driver.ModelConfig{
				Driver: "openai",
				Options: map[string]interface{}{
					"api_key": os.Getenv("OPENAI_API_KEY"),
					"model":   os.Getenv("VISION_MODEL"),
					"prompt":  "Default test prompt",
				},
			},
		}

		vision, err := New(cfg)
		assert.NoError(t, err)

		// Use base64 data with empty prompt (should use default)
		result, err := vision.Analyze(context.Background(), "data:image/png;base64,"+testImageBase64, "")
		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.NotEmpty(t, result.Description)
	})
}

func createTestVision(baseURL string) (*Vision, error) {
	cfg := &driver.Config{
		Storage: driver.StorageConfig{
			Driver: "local",
			Options: map[string]interface{}{
				"path":        "/__vision_test",
				"compression": true,
				"base_url":    baseURL,
			},
		},
		Model: driver.ModelConfig{
			Driver: "openai",
			Options: map[string]interface{}{
				"api_key": os.Getenv("OPENAI_API_KEY"),
				"model":   os.Getenv("VISION_MODEL"),
				"prompt": `# Objective 
					You are a vision assistant, you can help the user to understand the image and describe it.
					
					## Task Execution Steps
					1. Understand the image/video and describe it.
					2. Describe the image/video in detail.
					
					## Result Format
					{
						"description": "The description of the image/video",
						"content": "The content of the image/video"
					}`,
			},
		},
	}

	return New(cfg)
}

func createTestVisionWithS3() (*Vision, error) {
	// Check required S3 environment variables
	if os.Getenv("S3_API") == "" || os.Getenv("S3_ACCESS_KEY") == "" ||
		os.Getenv("S3_SECRET_KEY") == "" || os.Getenv("S3_BUCKET") == "" {
		return nil, fmt.Errorf("S3 environment variables not set")
	}

	cfg := &driver.Config{
		Storage: driver.StorageConfig{
			Driver: "s3",
			Options: map[string]interface{}{
				"endpoint":   os.Getenv("S3_API"),
				"region":     "auto",
				"key":        os.Getenv("S3_ACCESS_KEY"),
				"secret":     os.Getenv("S3_SECRET_KEY"),
				"bucket":     os.Getenv("S3_BUCKET"),
				"prefix":     "vision-test",
				"expiration": "5m",
			},
		},
		Model: driver.ModelConfig{
			Driver: "openai",
			Options: map[string]interface{}{
				"api_key": os.Getenv("OPENAI_API_KEY"),
				"model":   os.Getenv("VISION_MODEL"),
				"prompt": `# Objective 
				You are a vision assistant, you can help the user to understand the image and describe it.
				
				## Task Execution Steps
				1. Understand the image/video and describe it.
				2. Describe the image/video in detail.
				
				## Result Format
				{
					"description": "The description of the image/video",
					"content": "The content of the image/video"
				}`,
			},
		},
	}

	return New(cfg)
}
