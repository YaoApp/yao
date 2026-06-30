//go:build unit

package task_test

import (
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/yaoapp/yao/agent/output/message"
	task "github.com/yaoapp/yao/agent/task"
)

// =============================================================================
// GAP 1: RunReq.Fresh behavior — verifies Options skip logic
// =============================================================================

func TestRunReq_FreshField_Serialization(t *testing.T) {
	req := task.RunReq{
		Messages: []task.InputMessage{{Role: "user", Content: "hello"}},
		Source:   "retry",
		Fresh:    true,
	}
	data, err := json.Marshal(req)
	require.NoError(t, err)

	var decoded task.RunReq
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)
	assert.True(t, decoded.Fresh)
	assert.Equal(t, "retry", decoded.Source)
	assert.Equal(t, "user", decoded.Messages[0].Role)
}

func TestRunReq_NotFresh_DefaultBehavior(t *testing.T) {
	req := task.RunReq{
		Messages: []task.InputMessage{{Role: "user", Content: "hello"}},
		Source:   "run",
	}
	assert.False(t, req.Fresh)
}

// =============================================================================
// GAP 4: StopIdleTimer
// =============================================================================

func TestDaemonContext_StopIdleTimer(t *testing.T) {
	dc := task.ExportNewDaemonContext("test-idle-timer-stop")
	defer dc.Cancel()

	// StopIdleTimer should be safe to call multiple times
	dc.StopIdleTimer()
	dc.StopIdleTimer()
}

func TestDaemonContext_StopIdleTimer_PreventsFire(t *testing.T) {
	dc := task.ExportNewDaemonContext("test-idle-fire")
	task.ExportRegisterDaemon("test-idle-fire", dc)

	dc.StopIdleTimer()

	// Verify daemon still in registry (timer didn't fire and clean up)
	_, found := task.ExportGetDaemon("test-idle-fire")
	assert.True(t, found)

	// Cleanup
	task.ExportUnregisterDaemon("test-idle-fire")
	dc.Cancel()
}

// =============================================================================
// enrichTaskResult prompt generation
// =============================================================================

func TestBuildEnrichResultPrompt_FirstRun(t *testing.T) {
	msgs := []string{"user: hello", "assistant: hi there"}
	systemPrompt, userContent := task.ExportBuildEnrichResultPrompt(msgs, true, nil, "zh-CN")
	prompt := systemPrompt + "\n" + userContent

	assert.Contains(t, prompt, "title")
	assert.Contains(t, prompt, "tags")
	assert.Contains(t, prompt, "priority")
	assert.Contains(t, prompt, "instruction")
	assert.Contains(t, prompt, "summary")
	assert.Contains(t, prompt, "outputs")
	assert.Contains(t, prompt, "completed normally")
	assert.Contains(t, prompt, "hello")
}

func TestBuildEnrichResultPrompt_NotFirstRun(t *testing.T) {
	msgs := []string{"user: continue", "assistant: done"}
	systemPrompt, userContent := task.ExportBuildEnrichResultPrompt(msgs, false, nil, "en")
	prompt := systemPrompt + "\n" + userContent

	// Should NOT contain title/tags/priority fields for non-first run
	assert.NotContains(t, prompt, `"title": "concise task title`)
	assert.NotContains(t, prompt, `"tags"`)
	assert.NotContains(t, prompt, `"priority": "none|low|medium|high"`)
	assert.Contains(t, prompt, "instruction")
}

func TestBuildEnrichResultPrompt_WithError(t *testing.T) {
	msgs := []string{"user: do something", "assistant: trying..."}
	systemPrompt, userContent := task.ExportBuildEnrichResultPrompt(msgs, true, fmt.Errorf("timeout after 60m"), "zh-CN")
	prompt := systemPrompt + "\n" + userContent

	assert.Contains(t, prompt, "execution error")
	assert.Contains(t, prompt, "timeout after 60m")
}

// =============================================================================
// Watch: combined AfterSeq + Limit
// =============================================================================

func TestWatch_DaemonAlive_AfterSeq_WithLimit(t *testing.T) {
	dc := task.ExportNewDaemonContext("test-watch-afterseq-limit")
	defer dc.Cancel()
	task.ExportRegisterDaemon("test-watch-afterseq-limit", dc)
	defer task.ExportUnregisterDaemon("test-watch-afterseq-limit")

	// Broadcast 10 messages (seq 1-10)
	for i := 0; i < 10; i++ {
		dc.Broadcast(&message.Message{Type: "text", Props: map[string]interface{}{"i": i}})
	}

	// AfterSeq=3, Limit=2 → should get seq 4, 5 only
	stream, err := dc.Watch(&task.WatchOpts{AfterSeq: 3, Limit: 2})
	require.NoError(t, err)
	defer stream.Cancel()

	// Receive 2 messages
	for i := 0; i < 2; i++ {
		select {
		case msg := <-stream.Ch:
			assert.Equal(t, "text", msg.Type)
			assert.True(t, msg.Metadata.Sequence > 3, "seq should be >3, got %d", msg.Metadata.Sequence)
			assert.True(t, msg.Metadata.Sequence <= 5, "seq should be <=5, got %d", msg.Metadata.Sequence)
		case <-time.After(time.Second):
			t.Fatalf("timeout waiting for msg %d", i)
		}
	}

	// read_complete with has_more=true (since there are more after seq 5)
	select {
	case msg := <-stream.Ch:
		assert.Equal(t, "event", msg.Type)
		assert.Equal(t, "read_complete", msg.Props["event"])
		assert.Equal(t, true, msg.Props["has_more"])
		assert.Equal(t, false, msg.Props["live"])
	case <-time.After(time.Second):
		t.Fatal("timeout waiting for read_complete")
	}
}

func TestWatch_AfterSeq_BeyondBuffer(t *testing.T) {
	dc := task.ExportNewDaemonContext("test-watch-beyond")
	defer dc.Cancel()
	task.ExportRegisterDaemon("test-watch-beyond", dc)
	defer task.ExportUnregisterDaemon("test-watch-beyond")

	// Broadcast 3 messages (seq 1-3)
	for i := 0; i < 3; i++ {
		dc.Broadcast(&message.Message{Type: "text", Props: map[string]interface{}{"i": i}})
	}

	// AfterSeq=100 → no replay, should go straight to read_complete + live
	stream, err := dc.Watch(&task.WatchOpts{AfterSeq: 100, Limit: 0})
	require.NoError(t, err)
	defer stream.Cancel()

	select {
	case msg := <-stream.Ch:
		assert.Equal(t, "event", msg.Type)
		assert.Equal(t, "read_complete", msg.Props["event"])
		assert.Equal(t, false, msg.Props["has_more"])
		assert.Equal(t, true, msg.Props["live"])
	case <-time.After(time.Second):
		t.Fatal("timeout waiting for read_complete")
	}

	// Should still receive live messages
	dc.Broadcast(&message.Message{Type: "text", Props: map[string]interface{}{"live": true}})
	select {
	case msg := <-stream.Ch:
		assert.Equal(t, true, msg.Props["live"])
	case <-time.After(time.Second):
		t.Fatal("timeout on live message")
	}
}

// =============================================================================
// Watch: cancel behavior
// =============================================================================

func TestWatch_Cancel_ClosesChannel(t *testing.T) {
	dc := task.ExportNewDaemonContext("test-watch-cancel")
	defer dc.Cancel()
	task.ExportRegisterDaemon("test-watch-cancel", dc)
	defer task.ExportUnregisterDaemon("test-watch-cancel")

	stream, err := dc.Watch(&task.WatchOpts{})
	require.NoError(t, err)

	// Drain read_complete
	select {
	case <-stream.Ch:
	case <-time.After(time.Second):
		t.Fatal("timeout")
	}

	// Cancel from client side
	stream.Cancel()

	// Channel should eventually close
	select {
	case _, ok := <-stream.Ch:
		assert.False(t, ok)
	case <-time.After(time.Second):
		t.Fatal("channel should have closed after cancel")
	}
}

// =============================================================================
// Watch: multiple concurrent watches
// =============================================================================

func TestWatch_MultipleConcurrent(t *testing.T) {
	dc := task.ExportNewDaemonContext("test-watch-multi")
	defer dc.Cancel()
	task.ExportRegisterDaemon("test-watch-multi", dc)
	defer task.ExportUnregisterDaemon("test-watch-multi")

	dc.Broadcast(&message.Message{Type: "text", Props: map[string]interface{}{"content": "setup"}})

	// Create 3 watchers
	streams := make([]*task.WatchStream, 3)
	for i := 0; i < 3; i++ {
		s, err := dc.Watch(&task.WatchOpts{})
		require.NoError(t, err)
		streams[i] = s
	}
	defer func() {
		for _, s := range streams {
			s.Cancel()
		}
	}()

	// Broadcast a live message
	dc.Broadcast(&message.Message{Type: "text", Props: map[string]interface{}{"content": "live-data"}})

	// All 3 should receive it (after draining replay + read_complete)
	for i, s := range streams {
		received := false
		timeout := time.After(2 * time.Second)
	loop:
		for {
			select {
			case msg, ok := <-s.Ch:
				if !ok {
					break loop
				}
				if msg.Type == "text" && msg.Props["content"] == "live-data" {
					received = true
					break loop
				}
			case <-timeout:
				break loop
			}
		}
		assert.True(t, received, "watcher %d should have received live-data", i)
	}
}

// =============================================================================
// DaemonContext: Broadcast with sequence correctness
// =============================================================================

func TestDaemonContext_Broadcast_SequenceMonotonic(t *testing.T) {
	dc := task.ExportNewDaemonContext("test-seq-monotonic")
	defer dc.Cancel()
	task.ExportRegisterDaemon("test-seq-monotonic", dc)
	defer task.ExportUnregisterDaemon("test-seq-monotonic")

	// Broadcast 100 messages
	for i := 0; i < 100; i++ {
		dc.Broadcast(&message.Message{Type: "text", Props: map[string]interface{}{"i": i}})
	}

	// Watch from beginning
	stream, err := dc.Watch(&task.WatchOpts{AfterSeq: 0, Limit: 0})
	require.NoError(t, err)
	defer stream.Cancel()

	var lastSeq int
	received := 0
	timeout := time.After(5 * time.Second)
loop:
	for received < 100 {
		select {
		case msg, ok := <-stream.Ch:
			if !ok {
				break loop
			}
			if msg.Type == "event" {
				continue
			}
			assert.True(t, msg.Metadata.Sequence > lastSeq,
				"sequence should be monotonically increasing: last=%d, got=%d", lastSeq, msg.Metadata.Sequence)
			lastSeq = msg.Metadata.Sequence
			received++
		case <-timeout:
			break loop
		}
	}
	assert.Equal(t, 100, received)
}

// =============================================================================
// WSCommand: field validation
// =============================================================================

func TestWSCommand_RunWithMetadata(t *testing.T) {
	raw := `{"type":"run","messages":[{"role":"user","content":"deploy"}],"assistant_id":"ast-01","metadata":{"env":"prod"},"priority":5}`
	var cmd task.WSCommand
	err := json.Unmarshal([]byte(raw), &cmd)
	require.NoError(t, err)
	assert.Equal(t, "run", cmd.Type)
	assert.Len(t, cmd.Messages, 1)
	assert.Equal(t, "ast-01", cmd.AssistantID)
	assert.Equal(t, 5, cmd.Priority)
	assert.Equal(t, "prod", cmd.Metadata["env"])
}

func TestWSCommand_ReadWithPagination(t *testing.T) {
	raw := `{"type":"read","since":42,"limit":20}`
	var cmd task.WSCommand
	err := json.Unmarshal([]byte(raw), &cmd)
	require.NoError(t, err)
	assert.Equal(t, "read", cmd.Type)
	assert.Equal(t, int64(42), cmd.Since)
	assert.Equal(t, 20, cmd.Limit)
}

func TestWSCommand_UnknownType(t *testing.T) {
	raw := `{"type":"unknown_cmd"}`
	var cmd task.WSCommand
	err := json.Unmarshal([]byte(raw), &cmd)
	require.NoError(t, err)
	assert.Equal(t, "unknown_cmd", cmd.Type)
}

// =============================================================================
// Task struct: new fields serialization
// =============================================================================

func TestTask_NewFields_Serialization(t *testing.T) {
	tk := task.Task{
		ChatID: "chat-001",
		Instruction: &task.ScheduledInstruction{
			Prompt:        "Build and deploy the frontend",
			Locale:        "en",
			FirstQuestion: "Deploy to staging",
			FirstAnswer:   "Done",
			UpdatedAt:     "2025-01-01T00:00:00Z",
		},
		Summary: "Deployed frontend to staging",
		Outputs: []any{map[string]any{"type": "url", "name": "staging", "path": "https://staging.example.com"}},
	}
	data, err := json.Marshal(tk)
	require.NoError(t, err)

	var decoded task.Task
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)
	require.NotNil(t, decoded.Instruction)
	assert.Equal(t, "Build and deploy the frontend", decoded.Instruction.Prompt)
	assert.Equal(t, "en", decoded.Instruction.Locale)
	assert.Equal(t, "Deploy to staging", decoded.Instruction.FirstQuestion)
	assert.Equal(t, "Deployed frontend to staging", decoded.Summary)
	assert.NotNil(t, decoded.Outputs)
}

func TestTask_EmptyOptionalFields_OmitEmpty(t *testing.T) {
	tk := task.Task{ChatID: "chat-002"}
	data, err := json.Marshal(tk)
	require.NoError(t, err)

	raw := string(data)
	assert.NotContains(t, raw, "instruction")
	assert.NotContains(t, raw, "summary")
	assert.NotContains(t, raw, "outputs")
}

// =============================================================================
// CleanMarkdownFences helper
// =============================================================================

func TestCleanMarkdownFences_WithFences(t *testing.T) {
	input := "```json\n{\"title\":\"test\"}\n```"
	result := task.ExportCleanMarkdownFences(input)
	assert.Equal(t, `{"title":"test"}`, result)
}

func TestCleanMarkdownFences_WithoutFences(t *testing.T) {
	input := `{"title":"test"}`
	result := task.ExportCleanMarkdownFences(input)
	assert.Equal(t, `{"title":"test"}`, result)
}

func TestCleanMarkdownFences_MultiLine(t *testing.T) {
	input := "```json\n{\n  \"run_status\": \"completed\",\n  \"summary\": \"Done\"\n}\n```"
	result := task.ExportCleanMarkdownFences(input)
	assert.Contains(t, result, `"run_status"`)
	assert.NotContains(t, result, "```")
}

// =============================================================================
// Priority validation (extended cases)
// =============================================================================

func TestIsValidPriority_Extended(t *testing.T) {
	assert.False(t, task.ExportIsValidPriority("urgent"))
	assert.False(t, task.ExportIsValidPriority("HIGH"))
	assert.False(t, task.ExportIsValidPriority("  low"))
}

func TestIsValidMailPriority_Extended(t *testing.T) {
	assert.False(t, task.ExportIsValidMailPriority("urgent"))
	assert.False(t, task.ExportIsValidMailPriority("HIGH"))
	assert.False(t, task.ExportIsValidMailPriority("  medium"))
}

// =============================================================================
// DaemonContext: CloseSubscribers idempotent
// =============================================================================

func TestDaemonContext_CloseSubscribers_Idempotent(t *testing.T) {
	dc := task.ExportNewDaemonContext("test-close-idem")
	defer dc.Cancel()

	stream, err := dc.Watch(&task.WatchOpts{})
	require.NoError(t, err)

	// First close
	dc.CloseSubscribers()

	// Drain all buffered messages until channel closes
	timeout := time.After(2 * time.Second)
	closed := false
	for !closed {
		select {
		case _, ok := <-stream.Ch:
			if !ok {
				closed = true
			}
		case <-timeout:
			t.Fatal("channel not closed within timeout")
			return
		}
	}
	assert.True(t, closed)

	// Second close should not panic
	dc.CloseSubscribers()
}
