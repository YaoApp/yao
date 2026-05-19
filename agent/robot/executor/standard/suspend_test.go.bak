//go:build e2e

package standard

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// ============================================================================
// detectNeedMoreInfo unit tests (internal — tests unexported function)
// ============================================================================

func TestDetectNeedMoreInfo(t *testing.T) {
	t.Run("returns false for nil result", func(t *testing.T) {
		needInput, question := detectNeedMoreInfo(nil)
		assert.False(t, needInput)
		assert.Empty(t, question)
	})

	t.Run("returns false for nil Next", func(t *testing.T) {
		result := &CallResult{Content: "some text"}
		needInput, question := detectNeedMoreInfo(result)
		assert.False(t, needInput)
		assert.Empty(t, question)
	})

	t.Run("returns false for non-map Next", func(t *testing.T) {
		result := &CallResult{Next: "just a string"}
		needInput, question := detectNeedMoreInfo(result)
		assert.False(t, needInput)
		assert.Empty(t, question)
	})

	t.Run("returns false when status is not need_input", func(t *testing.T) {
		result := &CallResult{
			Next: map[string]interface{}{
				"status":  "ok",
				"content": "everything is fine",
			},
		}
		needInput, question := detectNeedMoreInfo(result)
		assert.False(t, needInput)
		assert.Empty(t, question)
	})

	t.Run("returns true with question from Next", func(t *testing.T) {
		result := &CallResult{
			Next: map[string]interface{}{
				"status":   "need_input",
				"question": "What time range should I use?",
			},
		}
		needInput, question := detectNeedMoreInfo(result)
		assert.True(t, needInput)
		assert.Equal(t, "What time range should I use?", question)
	})

	t.Run("falls back to GetText when question is empty", func(t *testing.T) {
		result := &CallResult{
			Content: "I need more information about the time range.",
			Next: map[string]interface{}{
				"status": "need_input",
			},
		}
		needInput, question := detectNeedMoreInfo(result)
		assert.True(t, needInput)
		assert.Equal(t, "I need more information about the time range.", question)
	})

	t.Run("returns true with empty question when both are empty", func(t *testing.T) {
		result := &CallResult{
			Next: map[string]interface{}{
				"status": "need_input",
			},
		}
		needInput, question := detectNeedMoreInfo(result)
		assert.True(t, needInput)
		assert.Empty(t, question)
	})

	t.Run("unwraps data envelope from Next hook", func(t *testing.T) {
		result := &CallResult{
			Next: map[string]interface{}{
				"data": map[string]interface{}{
					"status":   "need_input",
					"question": "Which database should I query?",
				},
			},
		}
		needInput, question := detectNeedMoreInfo(result)
		assert.True(t, needInput)
		assert.Equal(t, "Which database should I query?", question)
	})
}
