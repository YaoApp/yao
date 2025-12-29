package docx_test

import (
	stdContext "context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/yaoapp/gou/connector/openai"
	"github.com/yaoapp/yao/agent/content/docx"
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

// TestParseWithMissingURL tests parsing DOCX with missing URL
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

	handler := docx.New(options)
	_, _, err := handler.Parse(ctx, content)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "missing URL")
}

// TestParseWithLocalDocx tests parsing a local DOCX file
func TestParseWithLocalDocx(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	testutils.Prepare(t)
	defer testutils.Clean(t)

	docxPath := getTestFilePath("docx.docx")
	if _, err := os.Stat(docxPath); os.IsNotExist(err) {
		t.Skipf("Test DOCX file not found: %s", docxPath)
	}

	options := newTestOptions()
	ctx := newTestContext()

	content := agentContext.ContentPart{
		Type: agentContext.ContentFile,
		File: &agentContext.FileAttachment{
			URL:      docxPath,
			Filename: "docx.docx",
		},
	}

	handler := docx.New(options)
	result, refs, err := handler.Parse(ctx, content)

	assert.NoError(t, err)
	assert.Nil(t, refs)
	assert.Equal(t, agentContext.ContentText, result.Type)
	assert.NotEmpty(t, result.Text)
	t.Logf("DOCX parse result (first 500 chars): %.500s...", result.Text)
}

// TestParseWithNonExistentFile tests parsing DOCX with non-existent file
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
			URL:      "/non/existent/path/test.docx",
			Filename: "test.docx",
		},
	}

	handler := docx.New(options)
	_, _, err := handler.Parse(ctx, content)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported DOCX source")
}
