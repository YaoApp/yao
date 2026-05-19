//go:build integration

package docx_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/yaoapp/yao/agent/content/docx"
	agentContext "github.com/yaoapp/yao/agent/context"
	"github.com/yaoapp/yao/unit-test/agent/testprepare"
)

func TestParseWithMissingURL(t *testing.T) {
	testprepare.PrepareSandbox(t)

	options := newTestOptions()
	ctx := newTestContext()

	content := agentContext.ContentPart{
		Type: agentContext.ContentFile,
		File: nil,
	}

	_, _, err := docx.New(options).Parse(ctx, content)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "missing URL")
}

func TestParseWithLocalDocx(t *testing.T) {
	testprepare.PrepareSandbox(t)

	docxPath := getTestFilePath("docx.docx")
	options := newTestOptions()
	ctx := newTestContext()

	content := agentContext.ContentPart{
		Type: agentContext.ContentFile,
		File: &agentContext.FileAttachment{URL: docxPath, Filename: "docx.docx"},
	}

	result, refs, err := docx.New(options).Parse(ctx, content)
	require.NoError(t, err)
	assert.Nil(t, refs)
	assert.Equal(t, agentContext.ContentText, result.Type)
	assert.NotEmpty(t, result.Text)
}

func TestParseWithNonExistentFile(t *testing.T) {
	testprepare.PrepareSandbox(t)

	options := newTestOptions()
	ctx := newTestContext()

	content := agentContext.ContentPart{
		Type: agentContext.ContentFile,
		File: &agentContext.FileAttachment{URL: "/non/existent/path/test.docx", Filename: "test.docx"},
	}

	_, _, err := docx.New(options).Parse(ctx, content)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported DOCX source")
}
