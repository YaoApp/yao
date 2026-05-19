//go:build integration

package pptx_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/yaoapp/yao/agent/content/pptx"
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

	_, _, err := pptx.New(options).Parse(ctx, content)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "missing URL")
}

func TestParseWithLocalPptx(t *testing.T) {
	testprepare.PrepareSandbox(t)

	pptxPath := getTestFilePath("pptx.pptx")
	options := newTestOptions()
	ctx := newTestContext()

	content := agentContext.ContentPart{
		Type: agentContext.ContentFile,
		File: &agentContext.FileAttachment{URL: pptxPath, Filename: "pptx.pptx"},
	}

	result, refs, err := pptx.New(options).Parse(ctx, content)
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
		File: &agentContext.FileAttachment{URL: "/non/existent/path/test.pptx", Filename: "test.pptx"},
	}

	_, _, err := pptx.New(options).Parse(ctx, content)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported PPTX source")
}
