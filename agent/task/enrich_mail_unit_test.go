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
	assert.Equal(t, "[assistant] something went wrong", texts[3])
}

func TestExtractRecentText_LimitN(t *testing.T) {
	dc := task.ExportNewDaemonContext("test-extract-limit")
	defer dc.Cancel()

	for i := 0; i < 20; i++ {
		dc.Broadcast(&message.Message{Type: "text", Props: map[string]interface{}{"content": "msg"}})
	}

	texts := task.ExportExtractRecentText(dc, 5)
	assert.Len(t, texts, 5)
}
