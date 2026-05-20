//go:build unit

package standard_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/yaoapp/yao/agent/robot/executor/standard"
)

// ============================================================================
// CallResult — GetText
// ============================================================================

func TestCallResultGetTextUnit(t *testing.T) {
	t.Run("returns content when available", func(t *testing.T) {
		result := &standard.CallResult{Content: "Hello World"}
		assert.Equal(t, "Hello World", result.GetText())
	})

	t.Run("returns empty for empty result", func(t *testing.T) {
		result := &standard.CallResult{}
		assert.Equal(t, "", result.GetText())
	})

	t.Run("returns next string when content is empty", func(t *testing.T) {
		result := &standard.CallResult{Next: "fallback text"}
		assert.Equal(t, "fallback text", result.GetText())
	})

	t.Run("prefers content over next", func(t *testing.T) {
		result := &standard.CallResult{
			Content: "primary",
			Next:    "secondary",
		}
		assert.Equal(t, "primary", result.GetText())
	})

	t.Run("returns content from next map", func(t *testing.T) {
		result := &standard.CallResult{
			Next: map[string]interface{}{
				"content": "from map",
			},
		}
		assert.Equal(t, "from map", result.GetText())
	})

	t.Run("returns content from next map data wrapper", func(t *testing.T) {
		result := &standard.CallResult{
			Next: map[string]interface{}{
				"data": map[string]interface{}{
					"content": "from data wrapper",
				},
			},
		}
		assert.Equal(t, "from data wrapper", result.GetText())
	})
}

// ============================================================================
// CallResult — IsEmpty
// ============================================================================

func TestCallResultIsEmptyUnit(t *testing.T) {
	t.Run("empty when no content and no next", func(t *testing.T) {
		result := &standard.CallResult{}
		assert.True(t, result.IsEmpty())
	})

	t.Run("not empty when has content", func(t *testing.T) {
		result := &standard.CallResult{Content: "test"}
		assert.False(t, result.IsEmpty())
	})

	t.Run("not empty when has next", func(t *testing.T) {
		result := &standard.CallResult{Next: map[string]interface{}{"key": "value"}}
		assert.False(t, result.IsEmpty())
	})

	t.Run("not empty when has both", func(t *testing.T) {
		result := &standard.CallResult{
			Content: "text",
			Next:    "data",
		}
		assert.False(t, result.IsEmpty())
	})
}

// ============================================================================
// CallResult — GetJSON
// ============================================================================

func TestCallResultGetJSONUnit(t *testing.T) {
	t.Run("returns next map directly", func(t *testing.T) {
		result := &standard.CallResult{
			Next: map[string]interface{}{"key": "value"},
		}
		data, err := result.GetJSON()
		require.NoError(t, err)
		assert.Equal(t, "value", data["key"])
	})

	t.Run("unwraps data from next map", func(t *testing.T) {
		result := &standard.CallResult{
			Next: map[string]interface{}{
				"data": map[string]interface{}{"inner": "value"},
			},
		}
		data, err := result.GetJSON()
		require.NoError(t, err)
		assert.Equal(t, "value", data["inner"])
	})

	t.Run("returns error for empty result", func(t *testing.T) {
		result := &standard.CallResult{}
		_, err := result.GetJSON()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "no content")
	})

	t.Run("returns error for non-JSON content", func(t *testing.T) {
		result := &standard.CallResult{Content: "not json at all"}
		_, err := result.GetJSON()
		assert.Error(t, err)
	})
}

// ============================================================================
// CallResult — GetJSONArray
// ============================================================================

func TestCallResultGetJSONArrayUnit(t *testing.T) {
	t.Run("returns next array directly", func(t *testing.T) {
		result := &standard.CallResult{
			Next: []interface{}{"a", "b", "c"},
		}
		arr, err := result.GetJSONArray()
		require.NoError(t, err)
		assert.Len(t, arr, 3)
	})

	t.Run("unwraps data array from next map", func(t *testing.T) {
		result := &standard.CallResult{
			Next: map[string]interface{}{
				"data": []interface{}{1, 2, 3},
			},
		}
		arr, err := result.GetJSONArray()
		require.NoError(t, err)
		assert.Len(t, arr, 3)
	})

	t.Run("returns error for empty result", func(t *testing.T) {
		result := &standard.CallResult{}
		_, err := result.GetJSONArray()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "no content")
	})
}

// ============================================================================
// ExtractCodeBlock — pure function using gou/text
// ============================================================================

func TestExtractCodeBlockUnit(t *testing.T) {
	t.Run("extracts JSON code block", func(t *testing.T) {
		content := "Here is the result:\n```json\n{\"key\": \"value\"}\n```"
		block := standard.ExtractCodeBlock(content)

		require.NotNil(t, block)
		assert.Equal(t, "json", block.Type)
		assert.Contains(t, block.Content, "key")
	})

	t.Run("returns text type for plain text", func(t *testing.T) {
		content := "Just plain text"
		block := standard.ExtractCodeBlock(content)

		require.NotNil(t, block)
		assert.Equal(t, "text", block.Type)
	})

	t.Run("extracts YAML code block", func(t *testing.T) {
		content := "Config:\n```yaml\nname: test\nvalue: 42\n```"
		block := standard.ExtractCodeBlock(content)

		require.NotNil(t, block)
		assert.Equal(t, "yaml", block.Type)
		assert.Contains(t, block.Content, "name")
	})
}

// ============================================================================
// ExtractAllCodeBlocks — pure function using gou/text
// ============================================================================

func TestExtractAllCodeBlocksUnit(t *testing.T) {
	t.Run("extracts multiple code blocks", func(t *testing.T) {
		content := "```json\n{}\n```\n\n```python\nprint('hello')\n```"
		blocks := standard.ExtractAllCodeBlocks(content)

		assert.Len(t, blocks, 2)
	})

	t.Run("extracts single code block", func(t *testing.T) {
		content := "Some text\n```go\nfmt.Println(\"hi\")\n```\nMore text"
		blocks := standard.ExtractAllCodeBlocks(content)

		assert.GreaterOrEqual(t, len(blocks), 1)
	})
}

// ============================================================================
// detectNeedMoreInfo — pure logic (via export_test.go)
// ============================================================================

func TestDetectNeedMoreInfoUnit(t *testing.T) {
	t.Run("detects need_input status in next data", func(t *testing.T) {
		result := &standard.CallResult{
			Next: map[string]interface{}{
				"data": map[string]interface{}{
					"status":   "need_input",
					"question": "What format do you prefer?",
				},
			},
		}
		needInput, question := standard.DetectNeedMoreInfoFn(result)
		assert.True(t, needInput)
		assert.Equal(t, "What format do you prefer?", question)
	})

	t.Run("detects need_input without data wrapper", func(t *testing.T) {
		result := &standard.CallResult{
			Next: map[string]interface{}{
				"status":   "need_input",
				"question": "Which database?",
			},
		}
		needInput, question := standard.DetectNeedMoreInfoFn(result)
		assert.True(t, needInput)
		assert.Equal(t, "Which database?", question)
	})

	t.Run("returns false for nil result", func(t *testing.T) {
		needInput, _ := standard.DetectNeedMoreInfoFn(nil)
		assert.False(t, needInput)
	})

	t.Run("returns false for nil next", func(t *testing.T) {
		result := &standard.CallResult{Content: "hello"}
		needInput, _ := standard.DetectNeedMoreInfoFn(result)
		assert.False(t, needInput)
	})

	t.Run("returns false for non-need_input status", func(t *testing.T) {
		result := &standard.CallResult{
			Next: map[string]interface{}{
				"status": "completed",
			},
		}
		needInput, _ := standard.DetectNeedMoreInfoFn(result)
		assert.False(t, needInput)
	})

	t.Run("falls back to GetText when question is empty", func(t *testing.T) {
		result := &standard.CallResult{
			Content: "Please clarify the requirements.",
			Next: map[string]interface{}{
				"status": "need_input",
			},
		}
		needInput, question := standard.DetectNeedMoreInfoFn(result)
		assert.True(t, needInput)
		assert.Equal(t, "Please clarify the requirements.", question)
	})
}

// ============================================================================
// Conversation — TurnCount, Messages, LastResponse, Reset (pure struct logic)
// ============================================================================

func TestConversationPureLogicUnit(t *testing.T) {
	t.Run("initial turn count is zero", func(t *testing.T) {
		conv := standard.NewConversation("test-agent", "test-chat", 10)
		assert.Equal(t, 0, conv.TurnCount())
	})

	t.Run("initial messages is empty", func(t *testing.T) {
		conv := standard.NewConversation("test-agent", "test-chat", 10)
		assert.Empty(t, conv.Messages())
	})

	t.Run("initial last response is nil", func(t *testing.T) {
		conv := standard.NewConversation("test-agent", "test-chat", 10)
		assert.Nil(t, conv.LastResponse())
	})

	t.Run("system prompt is preserved in messages", func(t *testing.T) {
		conv := standard.NewConversation("test-agent", "test-chat", 5).
			WithSystemPrompt("You are a task planner.")

		msgs := conv.Messages()
		require.Len(t, msgs, 1)
		assert.Equal(t, "system", string(msgs[0].Role))
		assert.Equal(t, "You are a task planner.", msgs[0].Content)
	})

	t.Run("reset preserves system prompt", func(t *testing.T) {
		conv := standard.NewConversation("test-agent", "test-chat", 5).
			WithSystemPrompt("You are helpful.")

		conv.Reset()

		msgs := conv.Messages()
		require.Len(t, msgs, 1)
		assert.Equal(t, "system", string(msgs[0].Role))
	})

	t.Run("reset clears messages without system prompt", func(t *testing.T) {
		conv := standard.NewConversation("test-agent", "test-chat", 5)
		conv.Reset()
		assert.Empty(t, conv.Messages())
	})
}
