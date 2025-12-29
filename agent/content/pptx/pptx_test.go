package pptx_test

import (
	stdContext "context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/yaoapp/gou/connector/openai"
	"github.com/yaoapp/yao/agent/content/pptx"
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

// TestParseWithMissingURL tests parsing PPTX with missing URL
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

	handler := pptx.New(options)
	_, _, err := handler.Parse(ctx, content)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "missing URL")
}

// TestParseWithLocalPptx tests parsing a local PPTX file
func TestParseWithLocalPptx(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	testutils.Prepare(t)
	defer testutils.Clean(t)

	pptxPath := getTestFilePath("pptx.pptx")
	if _, err := os.Stat(pptxPath); os.IsNotExist(err) {
		t.Skipf("Test PPTX file not found: %s", pptxPath)
	}

	options := newTestOptions()
	ctx := newTestContext()

	content := agentContext.ContentPart{
		Type: agentContext.ContentFile,
		File: &agentContext.FileAttachment{
			URL:      pptxPath,
			Filename: "pptx.pptx",
		},
	}

	handler := pptx.New(options)
	result, refs, err := handler.Parse(ctx, content)

	assert.NoError(t, err)
	assert.Nil(t, refs)
	assert.Equal(t, agentContext.ContentText, result.Type)
	assert.NotEmpty(t, result.Text)
	t.Logf("PPTX parse result (first 500 chars): %.500s...", result.Text)
}

// TestParseWithNonExistentFile tests parsing PPTX with non-existent file
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
			URL:      "/non/existent/path/test.pptx",
			Filename: "test.pptx",
		},
	}

	handler := pptx.New(options)
	_, _, err := handler.Parse(ctx, content)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported PPTX source")
}
