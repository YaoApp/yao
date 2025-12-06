package content

import (
	stdContext "context"
	"encoding/base64"
	"os"
	"strings"
	"testing"

	"github.com/yaoapp/gou/connector/openai"
	"github.com/yaoapp/gou/plan"
	agentContext "github.com/yaoapp/yao/agent/context"
	"github.com/yaoapp/yao/config"
	"github.com/yaoapp/yao/openapi/oauth/types"
	"github.com/yaoapp/yao/test"
)

func TestMain(m *testing.M) {
	// Setup test environment
	test.Prepare(nil, config.Conf)
	defer test.Clean()

	// Run tests
	code := m.Run()
	os.Exit(code)
}

// newTestContext creates a Context for testing with commonly used fields pre-populated
func newTestContext(capabilities *openai.Capabilities) *agentContext.Context {
	return &agentContext.Context{
		Context:     stdContext.Background(),
		Space:       plan.NewMemorySharedSpace(),
		ChatID:      "test-chat",
		AssistantID: "test-assistant",
		Locale:      "en-us",
		Theme:       "light",
		Client: agentContext.Client{
			Type:      "web",
			UserAgent: "TestAgent/1.0",
			IP:        "127.0.0.1",
		},
		Referer:      agentContext.RefererAPI,
		Accept:       agentContext.AcceptWebCUI,
		Route:        "",
		Metadata:     make(map[string]interface{}),
		Capabilities: capabilities,
		Authorized: &types.AuthorizedInfo{
			Subject:  "test-user",
			ClientID: "test-client-id",
			UserID:   "test-user-123",
			TeamID:   "test-team-456",
			TenantID: "test-tenant-789",
		},
	}
}

func TestImageHandler_CanHandle(t *testing.T) {
	handler := &ImageHandler{}

	tests := []struct {
		name        string
		contentType string
		fileType    FileType
		want        bool
	}{
		{"PNG image", "image/png", FileTypeImage, true},
		{"JPEG image", "image/jpeg", FileTypeImage, true},
		{"GIF image", "image/gif", FileTypeImage, true},
		{"WebP image", "image/webp", FileTypeImage, true},
		{"Text (should not handle)", "text/plain", FileTypeText, false},
		{"PDF (should not handle)", "application/pdf", FileTypePDF, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := handler.CanHandle(tt.contentType, tt.fileType)
			if got != tt.want {
				t.Errorf("CanHandle(%q, %q) = %v, want %v", tt.contentType, tt.fileType, got, tt.want)
			}
		})
	}
}

func TestImageHandler_Handle_WithVisionSupport(t *testing.T) {
	test.Prepare(t, config.Conf)
	defer test.Clean()

	handler := &ImageHandler{}

	// Create a simple test image (1x1 red PNG)
	pngData := createTestPNG()

	// Create capabilities with vision support
	capabilities := &openai.Capabilities{
		Vision: "openai", // Vision is enabled with OpenAI format
	}

	// Create test context
	ctx := newTestContext(capabilities)

	info := &Info{
		FileType:    FileTypeImage,
		ContentType: "image/png",
		Data:        pngData,
	}

	result, err := handler.Handle(ctx, info, capabilities, nil, false)
	if err != nil {
		t.Fatalf("Handle() error = %v", err)
	}

	if result == nil {
		t.Fatal("Expected non-nil result")
	}

	if result.ContentPart == nil {
		t.Fatal("Expected ContentPart for vision-supported model")
	}

	if result.ContentPart.Type != agentContext.ContentImageURL {
		t.Errorf("Expected ContentPart type = %v, got %v", agentContext.ContentImageURL, result.ContentPart.Type)
	}

	if result.ContentPart.ImageURL == nil {
		t.Fatal("Expected ImageURL to be set")
	}

	// Verify base64 encoding
	if result.ContentPart.ImageURL.URL == "" {
		t.Error("Expected non-empty URL")
	}

	// Should be data URI format
	if len(result.ContentPart.ImageURL.URL) < 20 {
		t.Error("Expected data URI to be longer")
	}
}

func TestImageHandler_Handle_WithoutVisionSupport(t *testing.T) {
	test.Prepare(t, config.Conf)
	defer test.Clean()

	handler := &ImageHandler{}

	// Create a simple test image
	pngData := createTestPNG()

	// Create capabilities WITHOUT vision support
	capabilities := &openai.Capabilities{
		Vision: nil, // No vision support
	}

	// Create test context
	ctx := newTestContext(capabilities)

	info := &Info{
		FileType:    FileTypeImage,
		ContentType: "image/png",
		Data:        pngData,
	}

	// Should return error because no vision support and no tool
	_, err := handler.Handle(ctx, info, capabilities, nil, false)
	if err == nil {
		t.Error("Expected error when no vision support and no tool specified")
	}
}

func TestImageHandler_Handle_EmptyData(t *testing.T) {
	test.Prepare(t, config.Conf)
	defer test.Clean()

	handler := &ImageHandler{}

	capabilities := &openai.Capabilities{
		Vision: "openai",
	}

	// Create test context
	ctx := newTestContext(capabilities)

	info := &Info{
		FileType:    FileTypeImage,
		ContentType: "image/png",
		Data:        []byte{}, // Empty data
	}

	_, err := handler.Handle(ctx, info, capabilities, nil, false)
	if err == nil {
		t.Error("Expected error for empty image data")
	}
}

func TestEncodeImageBase64(t *testing.T) {
	tests := []struct {
		name        string
		data        []byte
		contentType string
		wantPrefix  string
	}{
		{
			name:        "PNG image",
			data:        []byte{0x89, 0x50, 0x4E, 0x47}, // PNG magic number
			contentType: "image/png",
			wantPrefix:  "data:image/png;base64,",
		},
		{
			name:        "JPEG image",
			data:        []byte{0xFF, 0xD8, 0xFF}, // JPEG magic number
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
			result := encodeImageBase64(tt.data, tt.contentType)

			// Check prefix
			if !strings.HasPrefix(result, tt.wantPrefix) {
				t.Errorf("Expected prefix %q, got %q", tt.wantPrefix, result[:len(tt.wantPrefix)])
			}

			// Verify base64 encoding by decoding
			base64Part := result[len(tt.wantPrefix):]
			decoded, err := base64.StdEncoding.DecodeString(base64Part)
			if err != nil {
				t.Errorf("Failed to decode base64: %v", err)
			}

			// Verify decoded data matches original
			if len(decoded) != len(tt.data) {
				t.Errorf("Decoded length = %d, want %d", len(decoded), len(tt.data))
			}
			for i := range decoded {
				if decoded[i] != tt.data[i] {
					t.Errorf("Decoded byte[%d] = %x, want %x", i, decoded[i], tt.data[i])
				}
			}
		})
	}
}

// createTestPNG creates a minimal valid PNG image (1x1 red pixel)
func createTestPNG() []byte {
	// This is a minimal valid 1x1 red PNG image
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
