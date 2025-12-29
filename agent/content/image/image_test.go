package image_test

import (
	stdContext "context"
	"encoding/base64"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/yaoapp/gou/connector/openai"
	"github.com/yaoapp/yao/agent/content/image"
	contentTypes "github.com/yaoapp/yao/agent/content/types"
	agentContext "github.com/yaoapp/yao/agent/context"
	"github.com/yaoapp/yao/agent/output/message"
	"github.com/yaoapp/yao/agent/testutils"
	oauthTypes "github.com/yaoapp/yao/openapi/oauth/types"
)

// newTestContext creates a Context for testing with commonly used fields pre-populated
func newTestContext(capabilities *openai.Capabilities) *agentContext.Context {
	authorized := &oauthTypes.AuthorizedInfo{
		Subject:  "test-user",
		ClientID: "test-client-id",
		UserID:   "test-user-123",
		TeamID:   "test-team-456",
		TenantID: "test-tenant-789",
	}

	ctx := agentContext.New(stdContext.Background(), authorized, "test-chat")
	ctx.AssistantID = "test-assistant"
	ctx.Locale = "en-us"
	ctx.Theme = "light"
	ctx.Client = agentContext.Client{
		Type:      "web",
		UserAgent: "TestAgent/1.0",
		IP:        "127.0.0.1",
	}
	ctx.Referer = agentContext.RefererAPI
	ctx.Accept = agentContext.AcceptWebCUI
	ctx.Route = ""
	ctx.Metadata = make(map[string]interface{})
	ctx.Capabilities = capabilities
	ctx.IDGenerator = message.NewIDGenerator()
	return ctx
}

// newTestOptions creates test options with the given capabilities
func newTestOptions(capabilities *openai.Capabilities, completionOptions *agentContext.CompletionOptions) *contentTypes.Options {
	return &contentTypes.Options{
		Capabilities:      capabilities,
		CompletionOptions: completionOptions,
	}
}

// TestParseWithVisionSupport tests parsing image when model supports vision
func TestParseWithVisionSupport(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	testutils.Prepare(t)
	defer testutils.Clean(t)

	// Create capabilities with vision support
	capabilities := &openai.Capabilities{
		Vision: "openai",
	}

	options := newTestOptions(capabilities, nil)
	ctx := newTestContext(capabilities)

	// Create test image content with data URI
	base64Data := "data:image/png;base64," + base64.StdEncoding.EncodeToString(createTestPNG())
	content := agentContext.ContentPart{
		Type: agentContext.ContentImageURL,
		ImageURL: &agentContext.ImageURL{
			URL:    base64Data,
			Detail: agentContext.DetailAuto,
		},
	}

	handler := image.New(options)
	result, refs, err := handler.Parse(ctx, content)

	assert.NoError(t, err)
	assert.Nil(t, refs)
	assert.Equal(t, agentContext.ContentImageURL, result.Type)
	assert.NotNil(t, result.ImageURL)
	assert.Equal(t, base64Data, result.ImageURL.URL) // Should pass through unchanged
}

// TestParseWithoutVisionSupport tests parsing image when model doesn't support vision
func TestParseWithoutVisionSupport(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	testutils.Prepare(t)
	defer testutils.Clean(t)

	// Create capabilities WITHOUT vision support
	capabilities := &openai.Capabilities{
		Vision: nil,
	}

	options := newTestOptions(capabilities, nil)
	ctx := newTestContext(capabilities)

	// Create test image content
	base64Data := "data:image/png;base64," + base64.StdEncoding.EncodeToString(createTestPNG())
	content := agentContext.ContentPart{
		Type: agentContext.ContentImageURL,
		ImageURL: &agentContext.ImageURL{
			URL:    base64Data,
			Detail: agentContext.DetailAuto,
		},
	}

	handler := image.New(options)
	_, _, err := handler.Parse(ctx, content)

	// Should return error because no vision support and no vision tool specified
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no vision tool specified")
}

// TestParseWithEmptyURL tests parsing image with empty URL
func TestParseWithEmptyURL(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	testutils.Prepare(t)
	defer testutils.Clean(t)

	capabilities := &openai.Capabilities{
		Vision: "openai",
	}

	options := newTestOptions(capabilities, nil)
	ctx := newTestContext(capabilities)

	// Create content with empty URL
	content := agentContext.ContentPart{
		Type: agentContext.ContentImageURL,
		ImageURL: &agentContext.ImageURL{
			URL: "",
		},
	}

	handler := image.New(options)
	_, _, err := handler.Parse(ctx, content)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "missing URL")
}

// TestParseWithNilImageURL tests parsing image with nil ImageURL
func TestParseWithNilImageURL(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	testutils.Prepare(t)
	defer testutils.Clean(t)

	capabilities := &openai.Capabilities{
		Vision: "openai",
	}

	options := newTestOptions(capabilities, nil)
	ctx := newTestContext(capabilities)

	// Create content with nil ImageURL
	content := agentContext.ContentPart{
		Type:     agentContext.ContentImageURL,
		ImageURL: nil,
	}

	handler := image.New(options)
	_, _, err := handler.Parse(ctx, content)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "missing URL")
}

// TestEncodeToBase64DataURI tests base64 encoding
func TestEncodeToBase64DataURI(t *testing.T) {
	tests := []struct {
		name        string
		data        []byte
		contentType string
		wantPrefix  string
	}{
		{
			name:        "PNG image",
			data:        []byte{0x89, 0x50, 0x4E, 0x47},
			contentType: "image/png",
			wantPrefix:  "data:image/png;base64,",
		},
		{
			name:        "JPEG image",
			data:        []byte{0xFF, 0xD8, 0xFF},
			contentType: "image/jpeg",
			wantPrefix:  "data:image/jpeg;base64,",
		},
		{
			name:        "Empty content type defaults to PNG",
			data:        []byte{0x01, 0x02, 0x03},
			contentType: "",
			wantPrefix:  "data:image/png;base64,",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := image.EncodeToBase64DataURI(tt.data, tt.contentType)

			// Check prefix
			assert.True(t, strings.HasPrefix(result, tt.wantPrefix))

			// Verify base64 encoding by decoding
			base64Part := result[len(tt.wantPrefix):]
			decoded, err := base64.StdEncoding.DecodeString(base64Part)
			assert.NoError(t, err)

			// Verify decoded data matches original
			assert.Equal(t, tt.data, decoded)
		})
	}
}

// TestParseDataURIPassthrough tests that data URI images pass through unchanged when vision is supported
func TestParseDataURIPassthrough(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	testutils.Prepare(t)
	defer testutils.Clean(t)

	capabilities := &openai.Capabilities{
		Vision: "openai",
	}

	options := newTestOptions(capabilities, nil)
	ctx := newTestContext(capabilities)

	// Create test image content with data URI
	originalURL := "data:image/png;base64,iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mNk+M9QDwADhgGAWjR9awAAAABJRU5ErkJggg=="
	content := agentContext.ContentPart{
		Type: agentContext.ContentImageURL,
		ImageURL: &agentContext.ImageURL{
			URL:    originalURL,
			Detail: agentContext.DetailHigh,
		},
	}

	handler := image.New(options)
	result, refs, err := handler.Parse(ctx, content)

	assert.NoError(t, err)
	assert.Nil(t, refs)
	assert.Equal(t, agentContext.ContentImageURL, result.Type)
	assert.NotNil(t, result.ImageURL)
	assert.Equal(t, originalURL, result.ImageURL.URL)
	assert.Equal(t, agentContext.DetailHigh, result.ImageURL.Detail)
}

// TestParseWithVisionAgent tests parsing image using a vision agent when model doesn't support vision
func TestParseWithVisionAgent(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	testutils.Prepare(t)
	defer testutils.Clean(t)

	// Create capabilities WITHOUT vision support
	capabilities := &openai.Capabilities{
		Vision: nil, // Model doesn't support vision
	}

	// Configure to use vision agent
	completionOptions := &agentContext.CompletionOptions{
		Uses: &agentContext.Uses{
			Vision: "tests.vision-test", // Use our test vision agent
		},
	}

	options := newTestOptions(capabilities, completionOptions)
	ctx := newTestContext(capabilities)

	// Create test image content with data URI (1x1 red PNG)
	base64Data := "data:image/png;base64,iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mP8z8DwHwAFBQIAX8jx0gAAAABJRU5ErkJggg=="
	content := agentContext.ContentPart{
		Type: agentContext.ContentImageURL,
		ImageURL: &agentContext.ImageURL{
			URL:    base64Data,
			Detail: agentContext.DetailAuto,
		},
	}

	handler := image.New(options)
	result, refs, err := handler.Parse(ctx, content)

	// Should succeed and return text content (image description from agent)
	assert.NoError(t, err)
	assert.Nil(t, refs)
	assert.Equal(t, agentContext.ContentText, result.Type)
	assert.NotEmpty(t, result.Text) // Agent should return some description
	t.Logf("Vision agent response: %s", result.Text)
}

// TestParseWithForceUsesVisionAgent tests forceUses flag with vision agent
func TestParseWithForceUsesVisionAgent(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	testutils.Prepare(t)
	defer testutils.Clean(t)

	// Create capabilities WITH vision support
	capabilities := &openai.Capabilities{
		Vision: "openai", // Model supports vision
	}

	// Configure to FORCE use vision agent (even though model supports vision)
	completionOptions := &agentContext.CompletionOptions{
		ForceUses: true, // Force using the vision tool
		Uses: &agentContext.Uses{
			Vision: "tests.vision-test", // Use our test vision agent
		},
	}

	options := newTestOptions(capabilities, completionOptions)
	ctx := newTestContext(capabilities)

	// Create test image content with data URI
	base64Data := "data:image/png;base64,iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mP8z8DwHwAFBQIAX8jx0gAAAABJRU5ErkJggg=="
	content := agentContext.ContentPart{
		Type: agentContext.ContentImageURL,
		ImageURL: &agentContext.ImageURL{
			URL:    base64Data,
			Detail: agentContext.DetailAuto,
		},
	}

	handler := image.New(options)
	result, refs, err := handler.Parse(ctx, content)

	// Should succeed and return text content (forced to use agent even though model supports vision)
	assert.NoError(t, err)
	assert.Nil(t, refs)
	assert.Equal(t, agentContext.ContentText, result.Type)
	assert.NotEmpty(t, result.Text) // Agent should return some description
	t.Logf("Vision agent (forced) response: %s", result.Text)
}

// createTestPNG creates a minimal valid PNG image (1x1 red pixel)
func createTestPNG() []byte {
	return []byte{
		0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A, // PNG signature
		0x00, 0x00, 0x00, 0x0D, 0x49, 0x48, 0x44, 0x52, // IHDR chunk
		0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x01, // 1x1 dimensions
		0x08, 0x02, 0x00, 0x00, 0x00, 0x90, 0x77, 0x53,
		0xDE, 0x00, 0x00, 0x00, 0x0C, 0x49, 0x44, 0x41, // IDAT chunk
		0x54, 0x08, 0xD7, 0x63, 0xF8, 0xCF, 0xC0, 0x00,
		0x00, 0x03, 0x01, 0x01, 0x00, 0x18, 0xDD, 0x8D,
		0xB4, 0x00, 0x00, 0x00, 0x00, 0x49, 0x45, 0x4E, // IEND chunk
		0x44, 0xAE, 0x42, 0x60, 0x82,
	}
}
