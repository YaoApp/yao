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

func TestBuildEnrichPrompt_Input(t *testing.T) {
	msgs := []string{"[assistant] Please provide the file path"}
	prompt := task.ExportBuildEnrichPrompt("input", msgs)
	assert.Contains(t, prompt, "等待用户输入")
	assert.Contains(t, prompt, "[assistant] Please provide the file path")
	assert.Contains(t, prompt, "JSON")
}

func TestBuildEnrichPrompt_Completed(t *testing.T) {
	msgs := []string{"[assistant] Task completed successfully"}
	prompt := task.ExportBuildEnrichPrompt("completed", msgs)
	assert.Contains(t, prompt, "任务已完成")
}

func TestBuildEnrichPrompt_Failed(t *testing.T) {
	msgs := []string{"[assistant] Error: connection refused"}
	prompt := task.ExportBuildEnrichPrompt("failed", msgs)
	assert.Contains(t, prompt, "任务执行失败")
	assert.Contains(t, prompt, "error_type")
}

func TestBuildEnrichPrompt_Unknown(t *testing.T) {
	prompt := task.ExportBuildEnrichPrompt("unknown", []string{"msg"})
	assert.Equal(t, "", prompt)
}
