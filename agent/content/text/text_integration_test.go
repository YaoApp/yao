//go:build integration

package text_test

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/yaoapp/yao/agent/content/text"
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

	_, _, err := text.New(options).Parse(ctx, content)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "missing URL")
}

func TestParseWithLocalTextFile(t *testing.T) {
	testprepare.PrepareSandbox(t)

	txtPath := getTestFilePath("text.txt")
	options := newTestOptions()
	ctx := newTestContext()

	content := agentContext.ContentPart{
		Type: agentContext.ContentFile,
		File: &agentContext.FileAttachment{URL: txtPath, Filename: "text.txt"},
	}

	result, refs, err := text.New(options).Parse(ctx, content)
	require.NoError(t, err)
	assert.Nil(t, refs)
	assert.Equal(t, agentContext.ContentText, result.Type)
	assert.NotEmpty(t, result.Text)
}

func TestParseWithLocalMarkdownFile(t *testing.T) {
	testprepare.PrepareSandbox(t)

	mdPath := getTestFilePath("test.md")
	options := newTestOptions()
	ctx := newTestContext()

	content := agentContext.ContentPart{
		Type: agentContext.ContentFile,
		File: &agentContext.FileAttachment{URL: mdPath, Filename: "test.md"},
	}

	result, refs, err := text.New(options).Parse(ctx, content)
	require.NoError(t, err)
	assert.Nil(t, refs)
	assert.Equal(t, agentContext.ContentText, result.Type)
	assert.NotEmpty(t, result.Text)
}

func TestParseWithLocalCodeFile(t *testing.T) {
	testprepare.PrepareSandbox(t)

	tsPath := getTestFilePath("code.ts")
	options := newTestOptions()
	ctx := newTestContext()

	content := agentContext.ContentPart{
		Type: agentContext.ContentFile,
		File: &agentContext.FileAttachment{URL: tsPath, Filename: "code.ts"},
	}

	result, refs, err := text.New(options).Parse(ctx, content)
	require.NoError(t, err)
	assert.Nil(t, refs)
	assert.Equal(t, agentContext.ContentText, result.Type)
	assert.NotEmpty(t, result.Text)
	assert.True(t, strings.HasPrefix(result.Text, "```typescript"))
	assert.True(t, strings.HasSuffix(strings.TrimSpace(result.Text), "```"))
}

func TestParseWithLocalYaoFile(t *testing.T) {
	testprepare.PrepareSandbox(t)

	yaoPath := getTestFilePath("hero.mod.yao")
	options := newTestOptions()
	ctx := newTestContext()

	content := agentContext.ContentPart{
		Type: agentContext.ContentFile,
		File: &agentContext.FileAttachment{URL: yaoPath, Filename: "hero.mod.yao"},
	}

	result, refs, err := text.New(options).Parse(ctx, content)
	require.NoError(t, err)
	assert.Nil(t, refs)
	assert.Equal(t, agentContext.ContentText, result.Type)
	assert.NotEmpty(t, result.Text)
}

func TestParseWithLocalJsonFile(t *testing.T) {
	testprepare.PrepareSandbox(t)

	jsonPath := getTestFilePath("test.json")
	options := newTestOptions()
	ctx := newTestContext()

	content := agentContext.ContentPart{
		Type: agentContext.ContentFile,
		File: &agentContext.FileAttachment{URL: jsonPath, Filename: "test.json"},
	}

	result, refs, err := text.New(options).Parse(ctx, content)
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
		File: &agentContext.FileAttachment{URL: "/non/existent/path/test.txt", Filename: "test.txt"},
	}

	_, _, err := text.New(options).Parse(ctx, content)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported text file source")
}

func TestParseRawWithLocalFile(t *testing.T) {
	testprepare.PrepareSandbox(t)

	txtPath := getTestFilePath("text.txt")
	options := newTestOptions()
	ctx := newTestContext()

	content := agentContext.ContentPart{
		Type: agentContext.ContentFile,
		File: &agentContext.FileAttachment{URL: txtPath, Filename: "text.txt"},
	}

	result, refs, err := text.New(options).ParseRaw(ctx, content)
	require.NoError(t, err)
	assert.Nil(t, refs)
	assert.Equal(t, agentContext.ContentText, result.Type)
	assert.NotEmpty(t, result.Text)
	assert.True(t, strings.HasPrefix(result.Text, "File: text.txt"))
}
