//go:build unit

package task_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/yaoapp/yao/agent/task"
)

func TestExtractFirstUserMessage(t *testing.T) {
	msgs := []task.InputMessage{
		{Role: "system", Content: "system prompt"},
		{Role: "user", Content: "help me build a website"},
		{Role: "user", Content: "second message"},
	}
	assert.Equal(t, "help me build a website", task.ExtractFirstUserMessage(msgs))
}

func TestExtractFirstUserMessage_Empty(t *testing.T) {
	msgs := []task.InputMessage{
		{Role: "system", Content: "system prompt"},
	}
	assert.Equal(t, "", task.ExtractFirstUserMessage(msgs))
}

func TestExtractFirstUserMessage_Nil(t *testing.T) {
	assert.Equal(t, "", task.ExtractFirstUserMessage(nil))
}

func TestCleanMarkdownFences(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{`{"title": "test"}`, `{"title": "test"}`},
		{"```json\n{\"title\": \"test\"}\n```", `{"title": "test"}`},
		{"```\n{\"title\": \"test\"}\n```", `{"title": "test"}`},
		{"  ```json\n{\"title\": \"test\"}\n```  ", `{"title": "test"}`},
	}
	for _, tt := range tests {
		assert.Equal(t, tt.expected, task.ExportCleanMarkdownFences(tt.input))
	}
}

func TestIsValidPriority(t *testing.T) {
	assert.True(t, task.ExportIsValidPriority("high"))
	assert.True(t, task.ExportIsValidPriority("medium"))
	assert.True(t, task.ExportIsValidPriority("low"))
	assert.True(t, task.ExportIsValidPriority("none"))
	assert.False(t, task.ExportIsValidPriority("urgent"))
	assert.False(t, task.ExportIsValidPriority(""))
}

func TestIsValidMailPriority(t *testing.T) {
	assert.True(t, task.ExportIsValidMailPriority("high"))
	assert.True(t, task.ExportIsValidMailPriority("medium"))
	assert.True(t, task.ExportIsValidMailPriority("low"))
	assert.False(t, task.ExportIsValidMailPriority("none"))
	assert.False(t, task.ExportIsValidMailPriority(""))
}
