//go:build integration

package image_test

import (
	"encoding/base64"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/yaoapp/gou/connector/openai"
	"github.com/yaoapp/yao/agent/content/image"
	agentContext "github.com/yaoapp/yao/agent/context"
	"github.com/yaoapp/yao/unit-test/agent/testprepare"
)

func TestParseWithVisionSupport(t *testing.T) {
	testprepare.PrepareSandbox(t)

	capabilities := &openai.Capabilities{Vision: "openai"}
	options := newTestOptions(capabilities, nil)
	ctx := newTestContext(capabilities)

	base64Data := "data:image/png;base64," + base64.StdEncoding.EncodeToString(createTestPNG())
	content := agentContext.ContentPart{
		Type:     agentContext.ContentImageURL,
		ImageURL: &agentContext.ImageURL{URL: base64Data, Detail: agentContext.DetailAuto},
	}

	result, refs, err := image.New(options).Parse(ctx, content)
	require.NoError(t, err)
	assert.Nil(t, refs)
	assert.Equal(t, agentContext.ContentImageURL, result.Type)
	assert.NotNil(t, result.ImageURL)
	assert.Equal(t, base64Data, result.ImageURL.URL)
}

func TestParseWithoutVisionSupport(t *testing.T) {
	testprepare.PrepareSandbox(t)

	capabilities := &openai.Capabilities{Vision: nil}
	options := newTestOptions(capabilities, nil)
	ctx := newTestContext(capabilities)

	base64Data := "data:image/png;base64," + base64.StdEncoding.EncodeToString(createTestPNG())
	content := agentContext.ContentPart{
		Type:     agentContext.ContentImageURL,
		ImageURL: &agentContext.ImageURL{URL: base64Data, Detail: agentContext.DetailAuto},
	}

	result, _, err := image.New(options).Parse(ctx, content)
	require.NoError(t, err)
	assert.Equal(t, agentContext.ContentText, result.Type)
	assert.NotEmpty(t, result.Text)
}

func TestParseWithEmptyURL(t *testing.T) {
	testprepare.PrepareSandbox(t)

	capabilities := &openai.Capabilities{Vision: "openai"}
	options := newTestOptions(capabilities, nil)
	ctx := newTestContext(capabilities)

	content := agentContext.ContentPart{
		Type:     agentContext.ContentImageURL,
		ImageURL: &agentContext.ImageURL{URL: ""},
	}

	_, _, err := image.New(options).Parse(ctx, content)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "missing URL")
}

func TestParseWithNilImageURL(t *testing.T) {
	testprepare.PrepareSandbox(t)

	capabilities := &openai.Capabilities{Vision: "openai"}
	options := newTestOptions(capabilities, nil)
	ctx := newTestContext(capabilities)

	content := agentContext.ContentPart{
		Type:     agentContext.ContentImageURL,
		ImageURL: nil,
	}

	_, _, err := image.New(options).Parse(ctx, content)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "missing URL")
}

func TestParseDataURIPassthrough(t *testing.T) {
	testprepare.PrepareSandbox(t)

	capabilities := &openai.Capabilities{Vision: "openai"}
	options := newTestOptions(capabilities, nil)
	ctx := newTestContext(capabilities)

	originalURL := "data:image/png;base64,iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mNk+M9QDwADhgGAWjR9awAAAABJRU5ErkJggg=="
	content := agentContext.ContentPart{
		Type:     agentContext.ContentImageURL,
		ImageURL: &agentContext.ImageURL{URL: originalURL, Detail: agentContext.DetailHigh},
	}

	result, refs, err := image.New(options).Parse(ctx, content)
	require.NoError(t, err)
	assert.Nil(t, refs)
	assert.Equal(t, agentContext.ContentImageURL, result.Type)
	assert.Equal(t, originalURL, result.ImageURL.URL)
	assert.Equal(t, agentContext.DetailHigh, result.ImageURL.Detail)
}

func TestParseWithVisionAgent(t *testing.T) {
	testprepare.PrepareSandbox(t)

	capabilities := &openai.Capabilities{Vision: nil}
	completionOptions := &agentContext.CompletionOptions{
		Uses: &agentContext.Uses{Vision: "tests.vision-test"},
	}
	options := newTestOptions(capabilities, completionOptions)
	ctx := newTestContext(capabilities)

	base64Data := "data:image/png;base64,iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mP8z8DwHwAFBQIAX8jx0gAAAABJRU5ErkJggg=="
	content := agentContext.ContentPart{
		Type:     agentContext.ContentImageURL,
		ImageURL: &agentContext.ImageURL{URL: base64Data, Detail: agentContext.DetailAuto},
	}

	result, refs, err := image.New(options).Parse(ctx, content)
	require.NoError(t, err)
	assert.Nil(t, refs)
	assert.Equal(t, agentContext.ContentText, result.Type)
	assert.NotEmpty(t, result.Text)
}

func TestParseWithForceUsesVisionAgent(t *testing.T) {
	testprepare.PrepareSandbox(t)

	capabilities := &openai.Capabilities{Vision: "openai"}
	completionOptions := &agentContext.CompletionOptions{
		ForceUses: true,
		Uses:      &agentContext.Uses{Vision: "tests.vision-test"},
	}
	options := newTestOptions(capabilities, completionOptions)
	ctx := newTestContext(capabilities)

	base64Data := "data:image/png;base64,iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mP8z8DwHwAFBQIAX8jx0gAAAABJRU5ErkJggg=="
	content := agentContext.ContentPart{
		Type:     agentContext.ContentImageURL,
		ImageURL: &agentContext.ImageURL{URL: base64Data, Detail: agentContext.DetailAuto},
	}

	result, refs, err := image.New(options).Parse(ctx, content)
	require.NoError(t, err)
	assert.Nil(t, refs)
	assert.Equal(t, agentContext.ContentText, result.Type)
	assert.NotEmpty(t, result.Text)
}
