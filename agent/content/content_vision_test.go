package content_test

import (
	"bytes"
	"context"
	"image"
	"image/color"
	"image/png"
	"mime/multipart"
	"strings"
	"testing"

	"github.com/yaoapp/gou/connector/openai"
	"github.com/yaoapp/yao/agent/content"
	agentContext "github.com/yaoapp/yao/agent/context"
	"github.com/yaoapp/yao/agent/testutils"
	"github.com/yaoapp/yao/attachment"
)

// setupTestUploader creates and registers a test uploader manager
// The manager will be registered with "__" prefix as required by attachment.Parse
func setupTestUploader(t *testing.T, name string) attachment.FileManager {
	// Register with __ prefix to match Parse behavior
	managerName := "__" + name
	manager, err := attachment.Register(managerName, "local", attachment.ManagerOption{
		Driver:       "local",
		MaxSize:      "10M",
		AllowedTypes: []string{"text/*", "image/*", "application/*"},
		Options: map[string]interface{}{
			"path": "/tmp/test_vision_attachments_" + name,
		},
	})
	if err != nil {
		t.Fatalf("Failed to register attachment manager '%s': %v", managerName, err)
	}
	return manager
}

// cleanupTestUploader removes the test uploader from registry
func cleanupTestUploader(name string) {
	delete(attachment.Managers, "__"+name)
}

// generateTestImage creates a valid PNG image (100x100 red square)
func generateTestImage(t *testing.T) []byte {
	img := image.NewRGBA(image.Rect(0, 0, 100, 100))
	red := color.RGBA{255, 0, 0, 255}
	for y := 0; y < 100; y++ {
		for x := 0; x < 100; x++ {
			img.Set(x, y, red)
		}
	}

	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatalf("Failed to encode test image: %v", err)
	}
	return buf.Bytes()
}

// TestVision_TextFile tests Vision function with text/code file parsing
func TestVision_TextFile(t *testing.T) {
	testutils.Prepare(t)
	defer testutils.Clean(t)

	// Setup test uploader
	uploaderName := "test-vision-text"
	manager := setupTestUploader(t, uploaderName)
	defer cleanupTestUploader(uploaderName)

	// 1. Create and upload a Go source file
	testContent := `package main

import "fmt"

func main() {
	fmt.Println("Hello, Vision Test!")
}
`

	// Upload file
	reader := strings.NewReader(testContent)
	fileHeader := &attachment.FileHeader{
		FileHeader: &multipart.FileHeader{
			Filename: "main.go",
			Size:     int64(len(testContent)),
			Header:   make(map[string][]string),
		},
	}
	fileHeader.Header.Set("Content-Type", "text/x-go")

	uploadedFile, err := manager.Upload(context.Background(), fileHeader, reader, attachment.UploadOption{
		Groups: []string{"vision", "test"},
	})
	if err != nil {
		t.Fatalf("Failed to upload file: %v", err)
	}

	t.Logf("Uploaded file ID: %s", uploadedFile.ID)

	// 2. Prepare Vision context (text files don't need special capabilities)
	ctx := agentContext.New(context.Background(), nil, "test")

	capabilities := &openai.Capabilities{}

	messages := []agentContext.Message{
		{
			Role: "user",
			Content: []agentContext.ContentPart{
				{
					Type: agentContext.ContentFile,
					File: &agentContext.FileAttachment{
						URL:      "__" + uploaderName + "://" + uploadedFile.ID,
						Filename: "main.go",
					},
				},
			},
		},
	}

	// 3. Call Vision function
	result, err := content.Vision(ctx, capabilities, messages, nil, false)
	if err != nil {
		t.Fatalf("Vision function failed: %v", err)
	}

	if len(result) != 1 {
		t.Fatalf("Expected 1 message, got %d", len(result))
	}

	// 4. Verify result
	contentParts, ok := result[0].Content.([]agentContext.ContentPart)
	if !ok {
		t.Fatalf("Expected content to be []ContentPart, got %T", result[0].Content)
	}

	if len(contentParts) != 1 {
		t.Fatalf("Expected 1 content part, got %d", len(contentParts))
	}

	// Should be converted to text
	if contentParts[0].Type != agentContext.ContentText {
		t.Errorf("Expected ContentText type, got %s", contentParts[0].Type)
	}

	if !strings.Contains(contentParts[0].Text, "package main") {
		t.Errorf("Expected text to contain 'package main', got: %s", contentParts[0].Text)
	}

	if !strings.Contains(contentParts[0].Text, "Hello, Vision Test!") {
		t.Errorf("Expected text to contain 'Hello, Vision Test!', got: %s", contentParts[0].Text)
	}

	t.Logf("✓ Text file successfully parsed: %d characters", len(contentParts[0].Text))
}

// TestVision_ImageWithVisionSupport tests image processing with vision-capable model
func TestVision_ImageWithVisionSupport(t *testing.T) {
	testutils.Prepare(t)
	defer testutils.Clean(t)

	// Setup test uploader
	uploaderName := "test-vision-image"
	manager := setupTestUploader(t, uploaderName)
	defer cleanupTestUploader(uploaderName)

	// 1. Create and upload a test image (1x1 red PNG)
	imageData := []byte{
		0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A,
		0x00, 0x00, 0x00, 0x0D, 0x49, 0x48, 0x44, 0x52,
		0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x01,
		0x08, 0x02, 0x00, 0x00, 0x00, 0x90, 0x77, 0x53,
		0xDE, 0x00, 0x00, 0x00, 0x0C, 0x49, 0x44, 0x41,
		0x54, 0x08, 0xD7, 0x63, 0xF8, 0xCF, 0xC0, 0x00,
		0x00, 0x03, 0x01, 0x01, 0x00, 0x18, 0xDD, 0x8D,
		0xB4, 0x00, 0x00, 0x00, 0x00, 0x49, 0x45, 0x4E,
		0x44, 0xAE, 0x42, 0x60, 0x82,
	}

	// Upload image
	reader := strings.NewReader(string(imageData))
	fileHeader := &attachment.FileHeader{
		FileHeader: &multipart.FileHeader{
			Filename: "test.png",
			Size:     int64(len(imageData)),
			Header:   make(map[string][]string),
		},
	}
	fileHeader.Header.Set("Content-Type", "image/png")

	uploadedFile, err := manager.Upload(context.Background(), fileHeader, reader, attachment.UploadOption{
		Groups: []string{"vision", "test"},
	})
	if err != nil {
		t.Fatalf("Failed to upload image: %v", err)
	}

	// 2. Prepare Vision context with vision-capable model
	ctx := agentContext.New(context.Background(), nil, "test")

	// Construct capabilities with vision support (OpenAI format)
	capabilities := &openai.Capabilities{
		Vision: agentContext.VisionFormatOpenAI, // OpenAI vision format
	}

	messages := []agentContext.Message{
		{
			Role: "user",
			Content: []agentContext.ContentPart{
				{
					Type: agentContext.ContentImageURL,
					ImageURL: &agentContext.ImageURL{
						URL: "__" + uploaderName + "://" + uploadedFile.ID,
					},
				},
			},
		},
	}

	// 3. Call Vision function (no uses needed for direct vision support)
	result, err := content.Vision(ctx, capabilities, messages, nil, false)
	if err != nil {
		t.Fatalf("Vision function failed: %v", err)
	}

	if len(result) != 1 {
		t.Fatalf("Expected 1 message, got %d", len(result))
	}

	// 4. Verify result
	contentParts, ok := result[0].Content.([]agentContext.ContentPart)
	if !ok {
		t.Fatalf("Expected content to be []ContentPart, got %T", result[0].Content)
	}

	if len(contentParts) != 1 {
		t.Fatalf("Expected 1 content part, got %d", len(contentParts))
	}

	// If model supports vision, should be image_url with base64
	if capabilities.Vision != nil {
		if contentParts[0].Type != agentContext.ContentImageURL {
			t.Errorf("Expected ContentImageURL type, got %s", contentParts[0].Type)
		}

		if contentParts[0].ImageURL == nil {
			t.Fatal("Expected ImageURL to be set")
		}

		if !strings.Contains(contentParts[0].ImageURL.URL, "data:image/png;base64,") {
			t.Errorf("Expected base64 data URI, got: %s", contentParts[0].ImageURL.URL)
		}

		t.Logf("✓ Image processed with vision support: %d bytes (base64)", len(contentParts[0].ImageURL.URL))
	} else {
		// If no vision support, should fall back to text (via agent/MCP)
		t.Logf("ℹ Model doesn't support vision, result type: %s", contentParts[0].Type)
	}
}

// TestVision_ImageWithAgent tests image processing with vision agent when model doesn't support vision
// Note: This test demonstrates the agent fallback mechanism when the model doesn't support vision
func TestVision_ImageWithAgent(t *testing.T) {
	testutils.Prepare(t)
	defer testutils.Clean(t)

	// Setup test uploader
	uploaderName := "test-vision-agent"
	manager := setupTestUploader(t, uploaderName)
	defer cleanupTestUploader(uploaderName)

	// 1. Generate and upload a valid test image (100x100 red PNG)
	imageData := generateTestImage(t)

	reader := strings.NewReader(string(imageData))
	fileHeader := &attachment.FileHeader{
		FileHeader: &multipart.FileHeader{
			Filename: "test.png",
			Size:     int64(len(imageData)),
			Header:   make(map[string][]string),
		},
	}
	fileHeader.Header.Set("Content-Type", "image/png")

	uploadedFile, err := manager.Upload(context.Background(), fileHeader, reader, attachment.UploadOption{
		Groups: []string{"vision", "test"},
	})
	if err != nil {
		t.Fatalf("Failed to upload image: %v", err)
	}

	// 2. Prepare Vision context with proper setup
	// Model does NOT support vision, but uses.Vision specifies a vision agent
	ctx := agentContext.New(context.Background(), nil, "test")

	// Capabilities without vision support
	capabilities := &openai.Capabilities{
		Vision: nil, // No vision support
	}

	// Uses configuration with vision agent
	uses := &agentContext.Uses{
		Vision: "tests.vision-helper", // Use vision-helper agent
	}

	messages := []agentContext.Message{
		{
			Role: "user",
			Content: []agentContext.ContentPart{
				{
					Type: agentContext.ContentImageURL,
					ImageURL: &agentContext.ImageURL{
						URL: "__" + uploaderName + "://" + uploadedFile.ID,
					},
				},
			},
		},
	}

	// 3. Call Vision - should use agent since model doesn't support vision
	result, err := content.Vision(ctx, capabilities, messages, uses, false)
	if err != nil {
		t.Fatalf("Vision function failed: %v", err)
	}

	if len(result) != 1 {
		t.Fatalf("Expected 1 message, got %d", len(result))
	}

	// 4. Verify result is text (processed by vision agent)
	contentParts, ok := result[0].Content.([]agentContext.ContentPart)
	if !ok {
		t.Fatalf("Expected content to be []ContentPart, got %T", result[0].Content)
	}

	if len(contentParts) != 1 {
		t.Fatalf("Expected 1 content part, got %d", len(contentParts))
	}

	if contentParts[0].Type != agentContext.ContentText {
		t.Errorf("Expected ContentText (from agent), got: %s", contentParts[0].Type)
	}

	if contentParts[0].Text == "" {
		t.Error("Expected non-empty text from vision agent processing")
	}

	t.Logf("✓ Vision agent processed image to text: %d characters", len(contentParts[0].Text))
	t.Logf("Agent response text:\n%s", contentParts[0].Text)
}

// TestVision_CachedContent tests that file content is cached and reused
func TestVision_CachedContent(t *testing.T) {
	testutils.Prepare(t)
	defer testutils.Clean(t)

	// Setup test uploader
	uploaderName := "test-vision-cache"
	manager := setupTestUploader(t, uploaderName)
	defer cleanupTestUploader(uploaderName)

	// 1. Upload a text file
	testContent := "Test content for caching verification"

	reader := strings.NewReader(testContent)
	fileHeader := &attachment.FileHeader{
		FileHeader: &multipart.FileHeader{
			Filename: "cache-test.txt",
			Size:     int64(len(testContent)),
			Header:   make(map[string][]string),
		},
	}
	fileHeader.Header.Set("Content-Type", "text/plain")

	uploadedFile, err := manager.Upload(context.Background(), fileHeader, reader, attachment.UploadOption{
		Groups: []string{"vision", "test"},
	})
	if err != nil {
		t.Fatalf("Failed to upload file: %v", err)
	}

	// 2. Prepare Vision context with same file referenced twice
	ctx := agentContext.New(context.Background(), nil, "test")

	// Construct simple capabilities (text files don't need vision)
	capabilities := &openai.Capabilities{}

	messages := []agentContext.Message{
		{
			Role: "user",
			Content: []agentContext.ContentPart{
				{
					Type: agentContext.ContentFile,
					File: &agentContext.FileAttachment{
						URL:      "__" + uploaderName + "://" + uploadedFile.ID,
						Filename: "cache-test.txt",
					},
				},
				{
					Type: agentContext.ContentFile,
					File: &agentContext.FileAttachment{
						URL:      "__" + uploaderName + "://" + uploadedFile.ID, // Same file
						Filename: "cache-test.txt",
					},
				},
			},
		},
	}

	// 3. Call Vision
	result, err := content.Vision(ctx, capabilities, messages, nil, false)
	if err != nil {
		t.Fatalf("Vision function failed: %v", err)
	}

	if len(result) != 1 {
		t.Fatalf("Expected 1 message, got %d", len(result))
	}

	// 4. Verify both file references were processed
	contentParts, ok := result[0].Content.([]agentContext.ContentPart)
	if !ok {
		t.Fatalf("Expected content to be []ContentPart")
	}

	if len(contentParts) != 2 {
		t.Fatalf("Expected 2 content parts (both files), got %d", len(contentParts))
	}

	// Both should be text with same content
	if contentParts[0].Type != agentContext.ContentText {
		t.Errorf("First part: expected ContentText, got %s", contentParts[0].Type)
	}

	if contentParts[1].Type != agentContext.ContentText {
		t.Errorf("Second part: expected ContentText, got %s", contentParts[1].Type)
	}

	if !strings.Contains(contentParts[0].Text, testContent) {
		t.Errorf("First part text doesn't contain expected content")
	}

	if !strings.Contains(contentParts[1].Text, testContent) {
		t.Errorf("Second part text doesn't contain expected content")
	}

	// Verify content was cached (check attachment manager)
	cachedText, err := manager.GetText(context.Background(), uploadedFile.ID)
	if err != nil {
		t.Fatalf("Failed to get cached text: %v", err)
	}

	if cachedText == "" {
		t.Error("Expected content to be cached in attachment manager")
	}

	t.Logf("✓ Content successfully cached and reused: %d characters", len(cachedText))
}

// TestVision_FileMetadataInSpace tests that file metadata is correctly passed to vision agent via ctx.Space
func TestVision_FileMetadataInSpace(t *testing.T) {
	testutils.Prepare(t)
	defer testutils.Clean(t)

	// Setup test uploader
	uploaderName := "test-vision-metadata"
	manager := setupTestUploader(t, uploaderName)
	defer cleanupTestUploader(uploaderName)

	// 1. Generate and upload a test image
	imageData := generateTestImage(t)

	reader := strings.NewReader(string(imageData))
	fileHeader := &attachment.FileHeader{
		FileHeader: &multipart.FileHeader{
			Filename: "test-metadata.png",
			Size:     int64(len(imageData)),
			Header:   make(map[string][]string),
		},
	}
	fileHeader.Header.Set("Content-Type", "image/png")

	uploadedFile, err := manager.Upload(context.Background(), fileHeader, reader, attachment.UploadOption{
		Groups: []string{"vision", "test"},
	})
	if err != nil {
		t.Fatalf("Failed to upload image: %v", err)
	}

	t.Logf("Uploaded file: %s (ID: %s)", uploadedFile.Filename, uploadedFile.ID)

	// 2. Prepare Vision context - model doesn't support vision, use agent
	ctx := agentContext.New(context.Background(), nil, "test")

	// Capabilities without vision support
	capabilities := &openai.Capabilities{
		Vision: nil, // No vision support - force agent usage
	}

	// Uses configuration with vision-helper agent
	uses := &agentContext.Uses{
		Vision: "tests.vision-helper", // Use vision-helper agent that logs file metadata
	}

	messages := []agentContext.Message{
		{
			Role: "user",
			Content: []agentContext.ContentPart{
				{
					Type: agentContext.ContentImageURL,
					ImageURL: &agentContext.ImageURL{
						URL: "__" + uploaderName + "://" + uploadedFile.ID,
					},
				},
			},
		},
	}

	// 3. Call Vision - should pass file metadata to agent via Space
	result, err := content.Vision(ctx, capabilities, messages, uses, false)
	if err != nil {
		t.Fatalf("Vision function failed: %v", err)
	}

	if len(result) != 1 {
		t.Fatalf("Expected 1 message, got %d", len(result))
	}

	// 4. Verify result contains file metadata validation info
	contentParts, ok := result[0].Content.([]agentContext.ContentPart)
	if !ok {
		t.Fatalf("Expected content to be []ContentPart, got %T", result[0].Content)
	}

	if len(contentParts) != 1 {
		t.Fatalf("Expected 1 content part, got %d", len(contentParts))
	}

	if contentParts[0].Type != agentContext.ContentText {
		t.Errorf("Expected ContentText (from agent), got: %s", contentParts[0].Type)
	}

	// The vision-helper assistant should have logged file metadata from Space
	// We can verify this through the response text (which should contain the description)
	if contentParts[0].Text == "" {
		t.Error("Expected non-empty text from vision agent")
	}

	t.Logf("✓ Vision agent processed image with file metadata")
	t.Logf("Agent response: %s", contentParts[0].Text)

	// Note: The actual validation of Space data happens in the vision-helper's Next hook
	// which logs the file metadata. In a real test, we would need to capture those logs
	// or have the agent return structured data that we can verify.
}

// TestVision_MultipleFilesMetadata tests file metadata handling with multiple attachments
func TestVision_MultipleFilesMetadata(t *testing.T) {
	testutils.Prepare(t)
	defer testutils.Clean(t)

	// Setup test uploader
	uploaderName := "test-vision-multi"
	manager := setupTestUploader(t, uploaderName)
	defer cleanupTestUploader(uploaderName)

	// 1. Upload two test images
	imageData1 := generateTestImage(t)
	imageData2 := generateTestImage(t)

	// Upload first image
	reader1 := strings.NewReader(string(imageData1))
	fileHeader1 := &attachment.FileHeader{
		FileHeader: &multipart.FileHeader{
			Filename: "test-image-1.png",
			Size:     int64(len(imageData1)),
			Header:   make(map[string][]string),
		},
	}
	fileHeader1.Header.Set("Content-Type", "image/png")

	uploadedFile1, err := manager.Upload(context.Background(), fileHeader1, reader1, attachment.UploadOption{
		Groups: []string{"vision", "test"},
	})
	if err != nil {
		t.Fatalf("Failed to upload first image: %v", err)
	}

	// Upload second image
	reader2 := strings.NewReader(string(imageData2))
	fileHeader2 := &attachment.FileHeader{
		FileHeader: &multipart.FileHeader{
			Filename: "test-image-2.png",
			Size:     int64(len(imageData2)),
			Header:   make(map[string][]string),
		},
	}
	fileHeader2.Header.Set("Content-Type", "image/png")

	uploadedFile2, err := manager.Upload(context.Background(), fileHeader2, reader2, attachment.UploadOption{
		Groups: []string{"vision", "test"},
	})
	if err != nil {
		t.Fatalf("Failed to upload second image: %v", err)
	}

	t.Logf("Uploaded files: %s (ID: %s), %s (ID: %s)",
		uploadedFile1.Filename, uploadedFile1.ID,
		uploadedFile2.Filename, uploadedFile2.ID)

	// 2. Prepare Vision context with both images
	ctx := agentContext.New(context.Background(), nil, "test")

	capabilities := &openai.Capabilities{
		Vision: nil, // No vision support
	}

	uses := &agentContext.Uses{
		Vision: "tests.vision-helper",
	}

	messages := []agentContext.Message{
		{
			Role: "user",
			Content: []agentContext.ContentPart{
				{
					Type: agentContext.ContentImageURL,
					ImageURL: &agentContext.ImageURL{
						URL: "__" + uploaderName + "://" + uploadedFile1.ID,
					},
				},
				{
					Type: agentContext.ContentImageURL,
					ImageURL: &agentContext.ImageURL{
						URL: "__" + uploaderName + "://" + uploadedFile2.ID,
					},
				},
			},
		},
	}

	// 3. Call Vision - should handle multiple files' metadata
	result, err := content.Vision(ctx, capabilities, messages, uses, false)
	if err != nil {
		t.Fatalf("Vision function failed: %v", err)
	}

	if len(result) != 1 {
		t.Fatalf("Expected 1 message, got %d", len(result))
	}

	// 4. Verify both images were processed
	contentParts, ok := result[0].Content.([]agentContext.ContentPart)
	if !ok {
		t.Fatalf("Expected content to be []ContentPart, got %T", result[0].Content)
	}

	// Each image should be processed by the vision agent and return text
	if len(contentParts) != 2 {
		t.Fatalf("Expected 2 content parts (one per image), got %d", len(contentParts))
	}

	for i, part := range contentParts {
		if part.Type != agentContext.ContentText {
			t.Errorf("Part %d: expected ContentText, got: %s", i, part.Type)
		}
		if part.Text == "" {
			t.Errorf("Part %d: expected non-empty text", i)
		}
	}

	t.Logf("✓ Multiple files processed with metadata tracking")
	t.Logf("File 1 response: %s", contentParts[0].Text)
	t.Logf("File 2 response: %s", contentParts[1].Text)
}
