package openai

import (
	"bytes"
	"context"
	"encoding/base64"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/yaoapp/gou/fs"
	"github.com/yaoapp/yao/config"
	"github.com/yaoapp/yao/neo/vision/driver/s3"
	"github.com/yaoapp/yao/test"
)

var (
	// 1x1 transparent PNG
	testImageBase64 = "iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mNk+M9QDwADhgGAWjR9awAAAABJRU5ErkJggg=="
)

func TestOpenAIModel(t *testing.T) {
	test.Prepare(t, config.Conf)
	defer test.Clean()

	t.Run("Create Model", func(t *testing.T) {
		model, err := New(map[string]interface{}{
			"api_key": os.Getenv("OPENAI_API_KEY"),
			"model":   os.Getenv("VISION_MODEL"),
		})
		assert.NoError(t, err)
		assert.NotNil(t, model)
		if model != nil {
			assert.Equal(t, os.Getenv("OPENAI_API_KEY"), model.APIKey)
			assert.Equal(t, os.Getenv("VISION_MODEL"), model.Model)
			assert.True(t, model.Compression)
		}
	})

	t.Run("Create Model with Invalid API Key", func(t *testing.T) {
		_, err := New(map[string]interface{}{})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "api_key is required")
	})

	t.Run("Analyze with Base64 Image", func(t *testing.T) {
		model, err := New(map[string]interface{}{
			"api_key": os.Getenv("OPENAI_API_KEY"),
			"model":   os.Getenv("VISION_MODEL"),
		})
		assert.NoError(t, err)

		// Use base64 image data
		result, err := model.Analyze(context.Background(), "data:image/png;base64,"+testImageBase64, "Describe this image in detail")
		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.NotEmpty(t, result["description"])
	})

	t.Run("Analyze with URL", func(t *testing.T) {
		if os.Getenv("S3_API") == "" || os.Getenv("S3_ACCESS_KEY") == "" ||
			os.Getenv("S3_SECRET_KEY") == "" || os.Getenv("S3_BUCKET") == "" {
			t.Skip("S3 environment variables not set")
		}

		model, err := New(map[string]interface{}{
			"api_key": os.Getenv("OPENAI_API_KEY"),
			"model":   os.Getenv("VISION_MODEL"),
		})
		assert.NoError(t, err)

		// Create S3 client and upload test image
		s3Client, err := s3.New(map[string]interface{}{
			"endpoint":   os.Getenv("S3_API"),
			"region":     "auto",
			"key":        os.Getenv("S3_ACCESS_KEY"),
			"secret":     os.Getenv("S3_SECRET_KEY"),
			"bucket":     os.Getenv("S3_BUCKET"),
			"prefix":     "vision-test",
			"expiration": "5m",
		})
		assert.NoError(t, err)

		// Upload test image
		imgData, err := base64.StdEncoding.DecodeString(testImageBase64)
		assert.NoError(t, err)
		reader := bytes.NewReader(imgData)
		fileID, err := s3Client.Upload(context.Background(), "test.png", reader, "image/png")
		assert.NoError(t, err)

		// Get URL from S3
		url := s3Client.URL(context.Background(), fileID)
		assert.NotEmpty(t, url)

		// Use S3 URL for analysis
		result, err := model.Analyze(context.Background(), url, "Describe this image in detail")
		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.NotEmpty(t, result["description"])
	})

	t.Run("Analyze with File ID", func(t *testing.T) {
		model, err := New(map[string]interface{}{
			"api_key": os.Getenv("OPENAI_API_KEY"),
			"model":   os.Getenv("VISION_MODEL"),
		})
		assert.NoError(t, err)

		// Create test file
		data, err := fs.Get("data")
		assert.NoError(t, err)

		// Write test image data
		imgData, err := base64.StdEncoding.DecodeString(testImageBase64)
		assert.NoError(t, err)
		_, err = data.WriteFile("/__vision_test/test.png", imgData, 0644)
		assert.NoError(t, err)

		// Analyze using file ID
		result, err := model.Analyze(context.Background(), "/__vision_test/test.png", "Describe this image in detail")
		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.NotEmpty(t, result["description"])
	})

	t.Run("Analyze with Invalid File ID", func(t *testing.T) {
		model, err := New(map[string]interface{}{
			"api_key": os.Getenv("OPENAI_API_KEY"),
			"model":   os.Getenv("VISION_MODEL"),
		})
		assert.NoError(t, err)

		_, err = model.Analyze(context.Background(), "/non-existent.png", "Describe this image in detail")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to read file")
	})

	t.Run("Analyze with Invalid API Key", func(t *testing.T) {
		model, err := New(map[string]interface{}{
			"api_key": "invalid-key",
			"model":   os.Getenv("VISION_MODEL"),
		})
		assert.NoError(t, err)

		_, err = model.Analyze(context.Background(), "data:image/png;base64,"+testImageBase64, "Describe this image in detail")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "OpenAI API error")
	})

	t.Run("Analyze with Default Prompt", func(t *testing.T) {
		model, err := New(map[string]interface{}{
			"api_key": os.Getenv("OPENAI_API_KEY"),
			"model":   os.Getenv("VISION_MODEL"),
			"prompt":  "Default test prompt",
		})
		assert.NoError(t, err)

		// Use base64 image data without providing a prompt
		result, err := model.Analyze(context.Background(), "data:image/png;base64,"+testImageBase64)
		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.NotEmpty(t, result["description"])
	})

	t.Run("Analyze with Custom Prompt Overriding Default", func(t *testing.T) {
		model, err := New(map[string]interface{}{
			"api_key": os.Getenv("OPENAI_API_KEY"),
			"model":   os.Getenv("VISION_MODEL"),
			"prompt":  "Default test prompt",
		})
		assert.NoError(t, err)

		// Use base64 image data with custom prompt
		result, err := model.Analyze(context.Background(), "data:image/png;base64,"+testImageBase64, "Custom test prompt")
		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.NotEmpty(t, result["description"])
	})

	t.Run("Analyze with Empty Custom Prompt", func(t *testing.T) {
		model, err := New(map[string]interface{}{
			"api_key": os.Getenv("OPENAI_API_KEY"),
			"model":   os.Getenv("VISION_MODEL"),
			"prompt":  "Default test prompt",
		})
		assert.NoError(t, err)

		// Use base64 image data with empty prompt (should use default)
		result, err := model.Analyze(context.Background(), "data:image/png;base64,"+testImageBase64, "")
		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.NotEmpty(t, result["description"])
	})
}
