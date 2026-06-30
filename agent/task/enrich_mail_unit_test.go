//go:build unit

package task_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/yaoapp/yao/agent/output/message"
	"github.com/yaoapp/yao/agent/task"
)

func TestExtractRecentText(t *testing.T) {
	dc := task.ExportNewDaemonContext("test-extract-text")
	defer dc.Cancel()

	dc.Broadcast(&message.Message{Type: "text", Props: map[string]interface{}{"role": "user", "content": "hello"}})
	dc.Broadcast(&message.Message{Type: "text", Props: map[string]interface{}{"role": "assistant", "content": "hi there"}})
	dc.Broadcast(&message.Message{Type: "loading", Props: map[string]interface{}{"message": "thinking..."}})
	dc.Broadcast(&message.Message{Type: "text", Props: map[string]interface{}{"content": "final answer"}})
	dc.Broadcast(&message.Message{Type: "error", Props: map[string]interface{}{"content": "something went wrong"}})

	texts := task.ExportExtractRecentText(dc, 10)
	assert.Len(t, texts, 4)
	assert.Equal(t, "[user] hello", texts[0])
	assert.Equal(t, "[assistant] hi there", texts[1])
	assert.Equal(t, "[assistant] final answer", texts[2])
	assert.Equal(t, "[system] something went wrong", texts[3])
}

func TestExtractRecentText_MergeDeltas(t *testing.T) {
	dc := task.ExportNewDaemonContext("test-extract-merge")
	defer dc.Cancel()

	dc.Broadcast(&message.Message{Type: "text", MessageID: "M1", Props: map[string]interface{}{"content": "Hello "}})
	dc.Broadcast(&message.Message{Type: "text", MessageID: "M1", Props: map[string]interface{}{"content": "world"}})
	dc.Broadcast(&message.Message{Type: "text", MessageID: "M1", Props: map[string]interface{}{"content": "!"}})
	dc.Broadcast(&message.Message{Type: "text", MessageID: "M2", Props: map[string]interface{}{"content": "Second message"}})

	texts := task.ExportExtractRecentText(dc, 10)
	assert.Len(t, texts, 2)
	assert.Equal(t, "[assistant] Hello world!", texts[0])
	assert.Equal(t, "[assistant] Second message", texts[1])
}

func TestExtractRecentText_LimitN(t *testing.T) {
	dc := task.ExportNewDaemonContext("test-extract-limit")
	defer dc.Cancel()

	for i := 0; i < 20; i++ {
		dc.Broadcast(&message.Message{Type: "text", Props: map[string]interface{}{"content": "msg"}})
	}

	// Each message has no MessageID → treated as separate complete messages; take last 5
	texts := task.ExportExtractRecentText(dc, 5)
	assert.Len(t, texts, 5)
}

func TestExtractRecentText_LimitN_Merged(t *testing.T) {
	dc := task.ExportNewDaemonContext("test-extract-limit-merged")
	defer dc.Cancel()

	// 3 logical messages, each with multiple delta chunks
	for i := 0; i < 10; i++ {
		dc.Broadcast(&message.Message{Type: "text", MessageID: "M1", Props: map[string]interface{}{"content": "a"}})
	}
	for i := 0; i < 10; i++ {
		dc.Broadcast(&message.Message{Type: "text", MessageID: "M2", Props: map[string]interface{}{"content": "b"}})
	}
	for i := 0; i < 10; i++ {
		dc.Broadcast(&message.Message{Type: "text", MessageID: "M3", Props: map[string]interface{}{"content": "c"}})
	}

	// N=2: take last 2 complete messages
	texts := task.ExportExtractRecentText(dc, 2)
	assert.Len(t, texts, 2)
	assert.Equal(t, "[assistant] bbbbbbbbbb", texts[0])
	assert.Equal(t, "[assistant] cccccccccc", texts[1])
}
