package text_test

import (
	stdContext "context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/yaoapp/gou/connector/openai"
	"github.com/yaoapp/yao/agent/content/text"
	contentTypes "github.com/yaoapp/yao/agent/content/types"
	agentContext "github.com/yaoapp/yao/agent/context"
	"github.com/yaoapp/yao/agent/output/message"
	"github.com/yaoapp/yao/agent/testutils"
	oauthTypes "github.com/yaoapp/yao/openapi/oauth/types"
)

const testFilesDir = "assistants/tests/vision-helper/tests"

func newTestContext() *agentContext.Context {
	authorized := &oauthTypes.AuthorizedInfo{
		Subject:  "test-user",
		ClientID: "test-client-id",
		UserID:   "test-user-123",
	}
	ctx := agentContext.New(stdContext.Background(), authorized, "test-chat")
	ctx.AssistantID = "test-assistant"
	ctx.Locale = "en-us"
	ctx.IDGenerator = message.NewIDGenerator()
	return ctx
}

func newTestOptions() *contentTypes.Options {
	return &contentTypes.Options{
		Capabilities: &openai.Capabilities{},
	}
}

func getTestFilePath(filename string) string {
	yaoRoot := os.Getenv("YAO_TEST_APPLICATION")
	if yaoRoot == "" {
		yaoRoot = os.Getenv("YAO_ROOT")
	}
	return filepath.Join(yaoRoot, testFilesDir, filename)
}

// TestIsSupportedExtension tests the IsSupportedExtension function
func TestIsSupportedExtension(t *testing.T) {
	// Supported extensions
	assert.True(t, text.IsSupportedExtension("test.md"))
	assert.True(t, text.IsSupportedExtension("test.txt"))
	assert.True(t, text.IsSupportedExtension("test.go"))
	assert.True(t, text.IsSupportedExtension("test.ts"))
	assert.True(t, text.IsSupportedExtension("test.json"))
	assert.True(t, text.IsSupportedExtension("test.jsonc"))
	assert.True(t, text.IsSupportedExtension("test.yao"))
	assert.True(t, text.IsSupportedExtension("test.yaml"))
	assert.True(t, text.IsSupportedExtension("test.yml"))
	assert.True(t, text.IsSupportedExtension("test.py"))
	assert.True(t, text.IsSupportedExtension("test.js"))
	assert.True(t, text.IsSupportedExtension("test.css"))
	assert.True(t, text.IsSupportedExtension("test.html"))

	// Unsupported extensions
	assert.False(t, text.IsSupportedExtension("test.docx"))
	assert.False(t, text.IsSupportedExtension("test.pptx"))
	assert.False(t, text.IsSupportedExtension("test.pdf"))
	assert.False(t, text.IsSupportedExtension("test.png"))
	assert.False(t, text.IsSupportedExtension("test.jpg"))
	assert.False(t, text.IsSupportedExtension("test.exe"))
	assert.False(t, text.IsSupportedExtension("test.zip"))
}

// TestParseWithMissingURL tests parsing text with missing URL
func TestParseWithMissingURL(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	testutils.Prepare(t)
	defer testutils.Clean(t)

	options := newTestOptions()
	ctx := newTestContext()

	content := agentContext.ContentPart{
		Type: agentContext.ContentFile,
		File: nil,
	}

	handler := text.New(options)
	_, _, err := handler.Parse(ctx, content)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "missing URL")
}

// TestParseWithLocalTextFile tests parsing a local text file
func TestParseWithLocalTextFile(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	testutils.Prepare(t)
	defer testutils.Clean(t)

	txtPath := getTestFilePath("text.txt")
	if _, err := os.Stat(txtPath); os.IsNotExist(err) {
		t.Skipf("Test text file not found: %s", txtPath)
	}

	options := newTestOptions()
	ctx := newTestContext()

	content := agentContext.ContentPart{
		Type: agentContext.ContentFile,
		File: &agentContext.FileAttachment{
			URL:      txtPath,
			Filename: "text.txt",
		},
	}

	handler := text.New(options)
	result, refs, err := handler.Parse(ctx, content)

	assert.NoError(t, err)
	assert.Nil(t, refs)
	assert.Equal(t, agentContext.ContentText, result.Type)
	assert.NotEmpty(t, result.Text)
	t.Logf("Text parse result: %s", result.Text)
}

// TestParseWithLocalMarkdownFile tests parsing a local markdown file
func TestParseWithLocalMarkdownFile(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	testutils.Prepare(t)
	defer testutils.Clean(t)

	mdPath := getTestFilePath("test.md")
	if _, err := os.Stat(mdPath); os.IsNotExist(err) {
		t.Skipf("Test markdown file not found: %s", mdPath)
	}

	options := newTestOptions()
	ctx := newTestContext()

	content := agentContext.ContentPart{
		Type: agentContext.ContentFile,
		File: &agentContext.FileAttachment{
			URL:      mdPath,
			Filename: "test.md",
		},
	}

	handler := text.New(options)
	result, refs, err := handler.Parse(ctx, content)

	assert.NoError(t, err)
	assert.Nil(t, refs)
	assert.Equal(t, agentContext.ContentText, result.Type)
	assert.NotEmpty(t, result.Text)
	t.Logf("Markdown parse result: %s", result.Text)
}

// TestParseWithLocalCodeFile tests parsing a local code file (TypeScript)
func TestParseWithLocalCodeFile(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	testutils.Prepare(t)
	defer testutils.Clean(t)

	tsPath := getTestFilePath("code.ts")
	if _, err := os.Stat(tsPath); os.IsNotExist(err) {
		t.Skipf("Test TypeScript file not found: %s", tsPath)
	}

	options := newTestOptions()
	ctx := newTestContext()

	content := agentContext.ContentPart{
		Type: agentContext.ContentFile,
		File: &agentContext.FileAttachment{
			URL:      tsPath,
			Filename: "code.ts",
		},
	}

	handler := text.New(options)
	result, refs, err := handler.Parse(ctx, content)

	assert.NoError(t, err)
	assert.Nil(t, refs)
	assert.Equal(t, agentContext.ContentText, result.Type)
	assert.NotEmpty(t, result.Text)
	// Code files should be wrapped in markdown code blocks
	assert.True(t, strings.HasPrefix(result.Text, "```typescript"))
	assert.True(t, strings.HasSuffix(strings.TrimSpace(result.Text), "```"))
	t.Logf("Code parse result (first 500 chars): %.500s...", result.Text)
}

// TestParseWithLocalYaoFile tests parsing a local .yao file
func TestParseWithLocalYaoFile(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	testutils.Prepare(t)
	defer testutils.Clean(t)

	yaoPath := getTestFilePath("hero.mod.yao")
	if _, err := os.Stat(yaoPath); os.IsNotExist(err) {
		t.Skipf("Test .yao file not found: %s", yaoPath)
	}

	options := newTestOptions()
	ctx := newTestContext()

	content := agentContext.ContentPart{
		Type: agentContext.ContentFile,
		File: &agentContext.FileAttachment{
			URL:      yaoPath,
			Filename: "hero.mod.yao",
		},
	}

	handler := text.New(options)
	result, refs, err := handler.Parse(ctx, content)

	assert.NoError(t, err)
	assert.Nil(t, refs)
	assert.Equal(t, agentContext.ContentText, result.Type)
	assert.NotEmpty(t, result.Text)
	t.Logf("Yao file parse result: %s", result.Text)
}

// TestParseWithLocalJsonFile tests parsing a local JSON file
func TestParseWithLocalJsonFile(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	testutils.Prepare(t)
	defer testutils.Clean(t)

	jsonPath := getTestFilePath("test.json")
	if _, err := os.Stat(jsonPath); os.IsNotExist(err) {
		t.Skipf("Test JSON file not found: %s", jsonPath)
	}

	options := newTestOptions()
	ctx := newTestContext()

	content := agentContext.ContentPart{
		Type: agentContext.ContentFile,
		File: &agentContext.FileAttachment{
			URL:      jsonPath,
			Filename: "test.json",
		},
	}

	handler := text.New(options)
	result, refs, err := handler.Parse(ctx, content)

	assert.NoError(t, err)
	assert.Nil(t, refs)
	assert.Equal(t, agentContext.ContentText, result.Type)
	assert.NotEmpty(t, result.Text)
	t.Logf("JSON parse result: %s", result.Text)
}

// TestParseWithNonExistentFile tests parsing text with non-existent file
func TestParseWithNonExistentFile(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	testutils.Prepare(t)
	defer testutils.Clean(t)

	options := newTestOptions()
	ctx := newTestContext()

	content := agentContext.ContentPart{
		Type: agentContext.ContentFile,
		File: &agentContext.FileAttachment{
			URL:      "/non/existent/path/test.txt",
			Filename: "test.txt",
		},
	}

	handler := text.New(options)
	_, _, err := handler.Parse(ctx, content)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported text file source")
}

// TestParseRawWithLocalFile tests ParseRaw with a local file
func TestParseRawWithLocalFile(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	testutils.Prepare(t)
	defer testutils.Clean(t)

	txtPath := getTestFilePath("text.txt")
	if _, err := os.Stat(txtPath); os.IsNotExist(err) {
		t.Skipf("Test text file not found: %s", txtPath)
	}

	options := newTestOptions()
	ctx := newTestContext()

	content := agentContext.ContentPart{
		Type: agentContext.ContentFile,
		File: &agentContext.FileAttachment{
			URL:      txtPath,
			Filename: "text.txt",
		},
	}

	handler := text.New(options)
	result, refs, err := handler.ParseRaw(ctx, content)

	assert.NoError(t, err)
	assert.Nil(t, refs)
	assert.Equal(t, agentContext.ContentText, result.Type)
	assert.NotEmpty(t, result.Text)
	// ParseRaw should include filename as context
	assert.True(t, strings.HasPrefix(result.Text, "File: text.txt"))
	t.Logf("ParseRaw result: %s", result.Text)
}
