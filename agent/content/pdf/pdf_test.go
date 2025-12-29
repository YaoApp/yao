package pdf_test

import (
	stdContext "context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/yaoapp/gou/connector/openai"
	"github.com/yaoapp/yao/agent/content/pdf"
	contentTypes "github.com/yaoapp/yao/agent/content/types"
	agentContext "github.com/yaoapp/yao/agent/context"
	"github.com/yaoapp/yao/agent/output/message"
	"github.com/yaoapp/yao/agent/testutils"
	oauthTypes "github.com/yaoapp/yao/openapi/oauth/types"
)

// Test files directory (relative to yao-dev-app)
const testFilesDir = "assistants/tests/vision-helper/tests"

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

// getTestFilePath returns the full path to a test file
func getTestFilePath(filename string) string {
	yaoRoot := os.Getenv("YAO_TEST_APPLICATION")
	if yaoRoot == "" {
		yaoRoot = os.Getenv("YAO_ROOT")
	}
	return filepath.Join(yaoRoot, testFilesDir, filename)
}

// TestParseWithMissingURL tests parsing PDF with missing URL
func TestParseWithMissingURL(t *testing.T) {
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

	// Create content with nil File
	content := agentContext.ContentPart{
		Type: agentContext.ContentFile,
		File: nil,
	}

	handler := pdf.New(options)
	_, _, err := handler.Parse(ctx, content)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "missing URL")
}

// TestParseWithEmptyURL tests parsing PDF with empty URL
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
		Type: agentContext.ContentFile,
		File: &agentContext.FileAttachment{
			URL:      "",
			Filename: "test.pdf",
		},
	}

	handler := pdf.New(options)
	_, _, err := handler.Parse(ctx, content)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "missing URL")
}

// TestParseWithLocalPDFAndVisionSupport tests parsing a local PDF file when model supports vision
func TestParseWithLocalPDFAndVisionSupport(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	testutils.Prepare(t)
	defer testutils.Clean(t)

	// Check if test file exists
	pdfPath := getTestFilePath("test.pdf")
	if _, err := os.Stat(pdfPath); os.IsNotExist(err) {
		t.Skipf("Test PDF file not found: %s", pdfPath)
	}

	// Create capabilities with vision support
	capabilities := &openai.Capabilities{
		Vision: "openai",
	}

	options := newTestOptions(capabilities, nil)
	ctx := newTestContext(capabilities)

	// Create content with local file path
	content := agentContext.ContentPart{
		Type: agentContext.ContentFile,
		File: &agentContext.FileAttachment{
			URL:      pdfPath,
			Filename: "test.pdf",
		},
	}

	handler := pdf.New(options)
	result, refs, err := handler.Parse(ctx, content)

	// Should succeed - PDF converted to images
	assert.NoError(t, err)
	assert.Nil(t, refs)

	// When model supports vision, Parse returns the first image_url part
	// Use ParseMulti to get all pages as separate image_url parts
	assert.Equal(t, agentContext.ContentImageURL, result.Type)
	assert.NotNil(t, result.ImageURL)
	assert.NotEmpty(t, result.ImageURL.URL)
	t.Logf("PDF parse result type: %s, URL prefix: %s...", result.Type, result.ImageURL.URL[:50])
}

// TestParseWithLocalPDFAndVisionAgent tests parsing a local PDF file using vision agent
func TestParseWithLocalPDFAndVisionAgent(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	testutils.Prepare(t)
	defer testutils.Clean(t)

	// Check if test file exists
	pdfPath := getTestFilePath("test.pdf")
	if _, err := os.Stat(pdfPath); os.IsNotExist(err) {
		t.Skipf("Test PDF file not found: %s", pdfPath)
	}

	// Create capabilities WITHOUT vision support
	capabilities := &openai.Capabilities{
		Vision: nil,
	}

	// Configure to use vision agent
	completionOptions := &agentContext.CompletionOptions{
		Uses: &agentContext.Uses{
			Vision: "tests.vision-test", // Use our test vision agent
		},
	}

	options := newTestOptions(capabilities, completionOptions)
	ctx := newTestContext(capabilities)

	// Create content with local file path
	content := agentContext.ContentPart{
		Type: agentContext.ContentFile,
		File: &agentContext.FileAttachment{
			URL:      pdfPath,
			Filename: "test.pdf",
		},
	}

	handler := pdf.New(options)
	result, refs, err := handler.Parse(ctx, content)

	// Should succeed - PDF converted to images and processed by vision agent
	assert.NoError(t, err)
	assert.Nil(t, refs)
	assert.Equal(t, agentContext.ContentText, result.Type)
	assert.NotEmpty(t, result.Text)
	t.Logf("PDF parse result (via vision agent): %s", result.Text)
}

// TestParseMultiWithLocalPDF tests ParseMulti which returns separate parts for each page
func TestParseMultiWithLocalPDF(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	testutils.Prepare(t)
	defer testutils.Clean(t)

	// Check if test file exists
	pdfPath := getTestFilePath("test.pdf")
	if _, err := os.Stat(pdfPath); os.IsNotExist(err) {
		t.Skipf("Test PDF file not found: %s", pdfPath)
	}

	// Create capabilities with vision support
	capabilities := &openai.Capabilities{
		Vision: "openai",
	}

	options := newTestOptions(capabilities, nil)
	ctx := newTestContext(capabilities)

	// Create content with local file path
	content := agentContext.ContentPart{
		Type: agentContext.ContentFile,
		File: &agentContext.FileAttachment{
			URL:      pdfPath,
			Filename: "test.pdf",
		},
	}

	handler := pdf.New(options)
	parts, refs, err := handler.ParseMulti(ctx, content)

	// Should succeed and return at least one part (one per page)
	assert.NoError(t, err)
	assert.Nil(t, refs)
	assert.NotEmpty(t, parts)
	t.Logf("PDF ParseMulti returned %d parts", len(parts))

	// When model supports vision, each part should be image_url type
	for i, part := range parts {
		assert.Equal(t, agentContext.ContentImageURL, part.Type)
		assert.NotNil(t, part.ImageURL)
		t.Logf("  Part %d: type=%s, has URL=%v", i+1, part.Type, part.ImageURL != nil && part.ImageURL.URL != "")
	}
}

// TestParseWithUnsupportedSource tests parsing PDF with unsupported source
func TestParseWithUnsupportedSource(t *testing.T) {
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

	// Create content with HTTP URL (not implemented)
	content := agentContext.ContentPart{
		Type: agentContext.ContentFile,
		File: &agentContext.FileAttachment{
			URL:      "https://example.com/test.pdf",
			Filename: "test.pdf",
		},
	}

	handler := pdf.New(options)
	_, _, err := handler.Parse(ctx, content)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "HTTP URL fetch not implemented")
}

// TestParseWithNonExistentFile tests parsing PDF with non-existent file
func TestParseWithNonExistentFile(t *testing.T) {
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

	// Create content with non-existent file
	content := agentContext.ContentPart{
		Type: agentContext.ContentFile,
		File: &agentContext.FileAttachment{
			URL:      "/non/existent/path/test.pdf",
			Filename: "test.pdf",
		},
	}

	handler := pdf.New(options)
	_, _, err := handler.Parse(ctx, content)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported PDF source")
}

// TestSilentLoadingOption tests that SilentLoading option is respected
func TestSilentLoadingOption(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	testutils.Prepare(t)
	defer testutils.Clean(t)

	capabilities := &openai.Capabilities{
		Vision: "openai",
	}

	// Create options with SilentLoading enabled
	options := &contentTypes.Options{
		Capabilities:  capabilities,
		SilentLoading: true,
	}

	// This test just verifies the option can be set
	// The actual behavior is tested in the image handler tests
	handler := pdf.New(options)
	assert.NotNil(t, handler)
	assert.True(t, options.SilentLoading)
}
