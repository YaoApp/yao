package claude

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	agentContext "github.com/yaoapp/yao/agent/context"
	"github.com/yaoapp/yao/config"
	"github.com/yaoapp/yao/test"
)

func TestExtensionFromContentType(t *testing.T) {
	tests := []struct {
		contentType string
		expected    string
	}{
		{"image/png", ".png"},
		{"image/jpeg", ".jpg"},
		{"image/gif", ".gif"},
		{"image/webp", ".webp"},
		{"image/svg+xml", ".svg"},
		{"application/pdf", ".pdf"},
		{"text/plain", ".txt"},
		{"text/html", ".html"},
		{"text/css", ".css"},
		{"text/javascript", ".js"},
		{"application/javascript", ".js"},
		{"application/json", ".json"},
		{"application/zip", ".zip"},
		{"application/octet-stream", ""},
		{"unknown/type", ""},
	}

	for _, tt := range tests {
		t.Run(tt.contentType, func(t *testing.T) {
			assert.Equal(t, tt.expected, extensionFromContentType(tt.contentType))
		})
	}
}

func TestFormatFileSize(t *testing.T) {
	tests := []struct {
		bytes    int
		expected string
	}{
		{0, "0B"},
		{100, "100B"},
		{1023, "1023B"},
		{1024, "1.0KB"},
		{1536, "1.5KB"},
		{10240, "10.0KB"},
		{1048576, "1.0MB"},
		{1572864, "1.5MB"},
		{10485760, "10.0MB"},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("%d", tt.bytes), func(t *testing.T) {
			assert.Equal(t, tt.expected, formatFileSize(tt.bytes))
		})
	}
}

func TestPrepareAttachmentsPlainText(t *testing.T) {
	test.Prepare(t, config.Conf)
	defer test.Clean()

	manager := createTestManager(t)
	if manager == nil {
		return
	}
	defer manager.Close()

	opts := &Options{
		Command: "claude",
		Image:   "alpine:latest",
		UserID:  "test-user",
		ChatID:  fmt.Sprintf("test-chat-att-plain-%d", time.Now().UnixNano()),
	}

	exec, err := NewExecutor(manager, opts)
	require.NoError(t, err)
	defer exec.Close()

	ctx := context.Background()

	// Plain text messages should pass through unchanged
	messages := []agentContext.Message{
		{Role: "system", Content: "You are a helpful assistant"},
		{Role: "user", Content: "Hello, world!"},
		{Role: "assistant", Content: "Hi there!"},
		{Role: "user", Content: "What is 1+1?"},
	}

	result, err := exec.prepareAttachments(ctx, messages)
	require.NoError(t, err)
	require.Len(t, result, 4)

	// Verify messages are unchanged
	assert.Equal(t, "system", string(result[0].Role))
	assert.Equal(t, "You are a helpful assistant", result[0].Content)
	assert.Equal(t, "Hello, world!", result[1].Content)
	assert.Equal(t, "Hi there!", result[2].Content)
	assert.Equal(t, "What is 1+1?", result[3].Content)
}

func TestPrepareAttachmentsMultimodalNoWrapper(t *testing.T) {
	test.Prepare(t, config.Conf)
	defer test.Clean()

	manager := createTestManager(t)
	if manager == nil {
		return
	}
	defer manager.Close()

	opts := &Options{
		Command: "claude",
		Image:   "alpine:latest",
		UserID:  "test-user",
		ChatID:  fmt.Sprintf("test-chat-att-nowrap-%d", time.Now().UnixNano()),
	}

	exec, err := NewExecutor(manager, opts)
	require.NoError(t, err)
	defer exec.Close()

	ctx := context.Background()

	// Multimodal message with a non-wrapper URL (e.g. regular http URL)
	// Should convert to text description but not try to resolve attachment
	messages := []agentContext.Message{
		{
			Role: "user",
			Content: []interface{}{
				map[string]interface{}{"type": "text", "text": "Look at this"},
				map[string]interface{}{
					"type": "image_url",
					"image_url": map[string]interface{}{
						"url":    "https://example.com/image.png",
						"detail": "auto",
					},
				},
			},
		},
	}

	result, err := exec.prepareAttachments(ctx, messages)
	require.NoError(t, err)
	require.Len(t, result, 1)

	// Content should be converted to text with URL reference
	content, ok := result[0].Content.(string)
	require.True(t, ok, "Content should be converted to string")
	assert.Contains(t, content, "Look at this")
	assert.Contains(t, content, "[Image: https://example.com/image.png]")
}

func TestPrepareAttachmentsTextOnlyMultimodal(t *testing.T) {
	test.Prepare(t, config.Conf)
	defer test.Clean()

	manager := createTestManager(t)
	if manager == nil {
		return
	}
	defer manager.Close()

	opts := &Options{
		Command: "claude",
		Image:   "alpine:latest",
		UserID:  "test-user",
		ChatID:  fmt.Sprintf("test-chat-att-textonly-%d", time.Now().UnixNano()),
	}

	exec, err := NewExecutor(manager, opts)
	require.NoError(t, err)
	defer exec.Close()

	ctx := context.Background()

	// Multimodal message with only text parts
	messages := []agentContext.Message{
		{
			Role: "user",
			Content: []interface{}{
				map[string]interface{}{"type": "text", "text": "Hello"},
				map[string]interface{}{"type": "text", "text": "World"},
			},
		},
	}

	result, err := exec.prepareAttachments(ctx, messages)
	require.NoError(t, err)
	require.Len(t, result, 1)

	// Should combine text parts
	content, ok := result[0].Content.(string)
	require.True(t, ok, "Content should be converted to string")
	assert.Contains(t, content, "Hello")
	assert.Contains(t, content, "World")
}

func TestPrepareAttachmentsInvalidWrapperURL(t *testing.T) {
	test.Prepare(t, config.Conf)
	defer test.Clean()

	manager := createTestManager(t)
	if manager == nil {
		return
	}
	defer manager.Close()

	opts := &Options{
		Command: "claude",
		Image:   "alpine:latest",
		UserID:  "test-user",
		ChatID:  fmt.Sprintf("test-chat-att-invalid-%d", time.Now().UnixNano()),
	}

	exec, err := NewExecutor(manager, opts)
	require.NoError(t, err)
	defer exec.Close()

	ctx := context.Background()

	// Message with an attachment URL pointing to a non-existent manager
	messages := []agentContext.Message{
		{
			Role: "user",
			Content: []interface{}{
				map[string]interface{}{"type": "text", "text": "See this image"},
				map[string]interface{}{
					"type": "image_url",
					"image_url": map[string]interface{}{
						"url":    "__nonexistent.uploader://fakefile123",
						"detail": "auto",
					},
				},
			},
		},
	}

	result, err := exec.prepareAttachments(ctx, messages)
	require.NoError(t, err)
	require.Len(t, result, 1)

	// Should gracefully fallback to error text
	content, ok := result[0].Content.(string)
	require.True(t, ok, "Content should be converted to string")
	assert.Contains(t, content, "See this image")
	assert.Contains(t, content, "failed to load")
}

func TestPrepareAttachmentsMixedRoles(t *testing.T) {
	test.Prepare(t, config.Conf)
	defer test.Clean()

	manager := createTestManager(t)
	if manager == nil {
		return
	}
	defer manager.Close()

	opts := &Options{
		Command: "claude",
		Image:   "alpine:latest",
		UserID:  "test-user",
		ChatID:  fmt.Sprintf("test-chat-att-mixed-%d", time.Now().UnixNano()),
	}

	exec, err := NewExecutor(manager, opts)
	require.NoError(t, err)
	defer exec.Close()

	ctx := context.Background()

	// Only user messages should be processed; system and assistant messages pass through
	messages := []agentContext.Message{
		{Role: "system", Content: "System prompt"},
		{
			Role: "user",
			Content: []interface{}{
				map[string]interface{}{"type": "text", "text": "User message with image"},
				map[string]interface{}{
					"type": "image_url",
					"image_url": map[string]interface{}{
						"url":    "https://example.com/photo.jpg",
						"detail": "auto",
					},
				},
			},
		},
		{Role: "assistant", Content: "I can see the photo"},
		{Role: "user", Content: "Thanks!"},
	}

	result, err := exec.prepareAttachments(ctx, messages)
	require.NoError(t, err)
	require.Len(t, result, 4)

	// System and assistant messages unchanged
	assert.Equal(t, "System prompt", result[0].Content)
	assert.Equal(t, "I can see the photo", result[2].Content)
	assert.Equal(t, "Thanks!", result[3].Content)

	// User multimodal message converted
	content, ok := result[1].Content.(string)
	require.True(t, ok, "User multimodal content should be converted to string")
	assert.Contains(t, content, "User message with image")
	assert.Contains(t, content, "[Image: https://example.com/photo.jpg]")
}
