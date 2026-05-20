//go:build integration

package pdf_test

import (
	"os/exec"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/yaoapp/gou/connector/openai"
	"github.com/yaoapp/yao/agent/content/pdf"
	contentTypes "github.com/yaoapp/yao/agent/content/types"
	agentContext "github.com/yaoapp/yao/agent/context"
	"github.com/yaoapp/yao/unit-test/agent/testprepare"
)

func requirePDFTool(t *testing.T) {
	t.Helper()
	for _, tool := range []string{"pdftoppm", "mutool"} {
		if _, err := exec.LookPath(tool); err == nil {
			return
		}
	}
	t.Skip("No PDF conversion tool available (pdftoppm or mutool)")
}

func TestParseWithMissingURL(t *testing.T) {
	testprepare.PrepareSandbox(t)

	capabilities := &openai.Capabilities{Vision: "openai"}
	options := newTestOptions(capabilities, nil)
	ctx := newTestContext(capabilities)

	content := agentContext.ContentPart{
		Type: agentContext.ContentFile,
		File: nil,
	}

	_, _, err := pdf.New(options).Parse(ctx, content)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "missing URL")
}

func TestParseWithEmptyURL(t *testing.T) {
	testprepare.PrepareSandbox(t)

	capabilities := &openai.Capabilities{Vision: "openai"}
	options := newTestOptions(capabilities, nil)
	ctx := newTestContext(capabilities)

	content := agentContext.ContentPart{
		Type: agentContext.ContentFile,
		File: &agentContext.FileAttachment{URL: "", Filename: "test.pdf"},
	}

	_, _, err := pdf.New(options).Parse(ctx, content)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "missing URL")
}

func TestParseWithLocalPDFAndVisionSupport(t *testing.T) {
	testprepare.PrepareSandbox(t)
	requirePDFTool(t)

	pdfPath := getTestFilePath("test.pdf")
	capabilities := &openai.Capabilities{Vision: "openai"}
	options := newTestOptions(capabilities, nil)
	ctx := newTestContext(capabilities)

	content := agentContext.ContentPart{
		Type: agentContext.ContentFile,
		File: &agentContext.FileAttachment{URL: pdfPath, Filename: "test.pdf"},
	}

	result, refs, err := pdf.New(options).Parse(ctx, content)
	require.NoError(t, err)
	assert.Nil(t, refs)
	assert.Equal(t, agentContext.ContentImageURL, result.Type)
	assert.NotNil(t, result.ImageURL)
	assert.NotEmpty(t, result.ImageURL.URL)
}

func TestParseWithLocalPDFAndVisionAgent(t *testing.T) {
	testprepare.PrepareSandbox(t)
	requirePDFTool(t)

	pdfPath := getTestFilePath("test.pdf")
	capabilities := &openai.Capabilities{Vision: nil}
	completionOptions := &agentContext.CompletionOptions{
		Uses: &agentContext.Uses{Vision: "tests.vision-test"},
	}
	options := newTestOptions(capabilities, completionOptions)
	ctx := newTestContext(capabilities)

	content := agentContext.ContentPart{
		Type: agentContext.ContentFile,
		File: &agentContext.FileAttachment{URL: pdfPath, Filename: "test.pdf"},
	}

	result, refs, err := pdf.New(options).Parse(ctx, content)
	require.NoError(t, err)
	assert.Nil(t, refs)
	assert.Equal(t, agentContext.ContentText, result.Type)
	assert.NotEmpty(t, result.Text)
}

func TestParseMultiWithLocalPDF(t *testing.T) {
	testprepare.PrepareSandbox(t)
	requirePDFTool(t)

	pdfPath := getTestFilePath("test.pdf")
	capabilities := &openai.Capabilities{Vision: "openai"}
	options := newTestOptions(capabilities, nil)
	ctx := newTestContext(capabilities)

	content := agentContext.ContentPart{
		Type: agentContext.ContentFile,
		File: &agentContext.FileAttachment{URL: pdfPath, Filename: "test.pdf"},
	}

	parts, refs, err := pdf.New(options).ParseMulti(ctx, content)
	require.NoError(t, err)
	assert.Nil(t, refs)
	require.NotEmpty(t, parts)

	for _, part := range parts {
		assert.Equal(t, agentContext.ContentImageURL, part.Type)
		assert.NotNil(t, part.ImageURL)
	}
}

func TestParseWithUnsupportedSource(t *testing.T) {
	testprepare.PrepareSandbox(t)

	capabilities := &openai.Capabilities{Vision: "openai"}
	options := newTestOptions(capabilities, nil)
	ctx := newTestContext(capabilities)

	content := agentContext.ContentPart{
		Type: agentContext.ContentFile,
		File: &agentContext.FileAttachment{URL: "https://example.com/test.pdf", Filename: "test.pdf"},
	}

	_, _, err := pdf.New(options).Parse(ctx, content)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "HTTP URL fetch not implemented")
}

func TestParseWithNonExistentFile(t *testing.T) {
	testprepare.PrepareSandbox(t)

	capabilities := &openai.Capabilities{Vision: "openai"}
	options := newTestOptions(capabilities, nil)
	ctx := newTestContext(capabilities)

	content := agentContext.ContentPart{
		Type: agentContext.ContentFile,
		File: &agentContext.FileAttachment{URL: "/non/existent/path/test.pdf", Filename: "test.pdf"},
	}

	_, _, err := pdf.New(options).Parse(ctx, content)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported PDF source")
}

func TestSilentLoadingOption(t *testing.T) {
	testprepare.PrepareSandbox(t)

	capabilities := &openai.Capabilities{Vision: "openai"}
	options := &contentTypes.Options{
		Capabilities:  capabilities,
		SilentLoading: true,
	}

	handler := pdf.New(options)
	assert.NotNil(t, handler)
	assert.True(t, options.SilentLoading)
}
