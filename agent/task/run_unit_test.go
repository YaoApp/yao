//go:build unit

package task_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/yaoapp/gou/process"
	"github.com/yaoapp/yao/agent/task"
)

func TestInputToAgentMessages(t *testing.T) {
	msgs := []task.InputMessage{
		{Role: "user", Content: "hello"},
		{Role: "system", Content: "you are helpful"},
		{Role: "assistant", Content: "hi there"},
	}

	result := task.ExportInputToAgentMessages(msgs)
	assert.Len(t, result, 3)
	assert.Equal(t, "user", string(result[0].Role))
	assert.Equal(t, "hello", result[0].Content)
	assert.Equal(t, "system", string(result[1].Role))
	assert.Equal(t, "you are helpful", result[1].Content)
	assert.Equal(t, "assistant", string(result[2].Role))
	assert.Equal(t, "hi there", result[2].Content)
}

func TestInputToAgentMessages_Empty(t *testing.T) {
	result := task.ExportInputToAgentMessages(nil)
	assert.Empty(t, result)

	result = task.ExportInputToAgentMessages([]task.InputMessage{})
	assert.Empty(t, result)
}

func TestMailTypeFromStatus(t *testing.T) {
	tests := []struct {
		status string
		want   string
	}{
		{"waiting", "input"},
		{"completed", "completed"},
		{"failed", "failed"},
		{"running", ""},
		{"pending", ""},
		{"queued", ""},
		{"", ""},
	}
	for _, tt := range tests {
		got := task.ExportMailTypeFromStatus(tt.status)
		assert.Equal(t, tt.want, got, "mailTypeFromStatus(%q)", tt.status)
	}
}

func TestGetStringVal(t *testing.T) {
	s := "hello"
	assert.Equal(t, "hello", task.ExportGetStringVal(&s))
	assert.Equal(t, "", task.ExportGetStringVal(nil))

	empty := ""
	assert.Equal(t, "", task.ExportGetStringVal(&empty))
}

func TestToOAuthInfo_Nil(t *testing.T) {
	result := task.ExportToOAuthInfo(nil)
	assert.Nil(t, result)
}

func TestToOAuthInfo_Full(t *testing.T) {
	auth := &process.AuthorizedInfo{
		Subject:    "sub-001",
		ClientID:   "client-abc",
		Scope:      "admin",
		SessionID:  "sess-xyz",
		UserID:     "user-123",
		TeamID:     "team-456",
		TenantID:   "tenant-789",
		RememberMe: true,
	}

	result := task.ExportToOAuthInfo(auth)
	assert.NotNil(t, result)
	assert.Equal(t, "sub-001", result.Subject)
	assert.Equal(t, "client-abc", result.ClientID)
	assert.Equal(t, "admin", result.Scope)
	assert.Equal(t, "sess-xyz", result.SessionID)
	assert.Equal(t, "user-123", result.UserID)
	assert.Equal(t, "team-456", result.TeamID)
	assert.Equal(t, "tenant-789", result.TenantID)
	assert.Equal(t, true, result.RememberMe)
}

func TestDaemonContext_StatusSetGet(t *testing.T) {
	dc := task.ExportNewDaemonContext("test-status")
	defer dc.Cancel()

	assert.Equal(t, task.DaemonRunning, dc.Status())

	dc.SetStatus(task.DaemonWaiting)
	assert.Equal(t, task.DaemonWaiting, dc.Status())

	dc.SetStatus(task.DaemonStopping)
	assert.Equal(t, task.DaemonStopping, dc.Status())

	dc.SetStatus(task.DaemonStopped)
	assert.Equal(t, task.DaemonStopped, dc.Status())
}

func TestDaemonContext_NextSequence(t *testing.T) {
	dc := task.ExportNewDaemonContext("test-seq")
	defer dc.Cancel()

	assert.Equal(t, int64(1), dc.NextSequence())
	assert.Equal(t, int64(2), dc.NextSequence())
	assert.Equal(t, int64(3), dc.NextSequence())
}

func TestDaemonRegistry_RegisterGetUnregister(t *testing.T) {
	chatID := "test-registry-chat"
	dc := task.ExportNewDaemonContext(chatID)
	defer dc.Cancel()

	// Not registered yet
	_, found := task.ExportGetDaemon(chatID)
	assert.False(t, found)

	// Register
	task.ExportRegisterDaemon(chatID, dc)

	// Now should be found
	got, found := task.ExportGetDaemon(chatID)
	assert.True(t, found)
	assert.Equal(t, dc, got)

	// Unregister
	task.ExportUnregisterDaemon(chatID)
	_, found = task.ExportGetDaemon(chatID)
	assert.False(t, found)
}

func TestDaemonResponseWriter_PartialBuffer(t *testing.T) {
	dc := task.ExportNewDaemonContext("test-writer-partial")
	defer dc.Cancel()

	sub, err := dc.Subscribe(&task.SubscribeOpts{Replay: task.ReplayNone})
	assert.NoError(t, err)
	defer sub.Cancel()

	w := task.ExportNewDaemonResponseWriter(dc)

	// Write partial data (no newline terminator yet) — should NOT produce a message
	w.Write([]byte(`data: {"type":"text","props":{"content":"part1`))

	select {
	case <-sub.Ch:
		t.Fatal("should not receive message before line is complete")
	default:
	}

	// Complete the message with closing JSON + newlines
	w.Write([]byte(`"}}` + "\n\n"))

	select {
	case msg := <-sub.Ch:
		assert.Equal(t, "text", msg.Type)
		assert.Equal(t, "part1", msg.Props["content"])
	case <-time.After(100 * time.Millisecond):
		t.Fatal("timeout waiting for partial write completion")
	}
}

func TestDaemonResponseWriter_MultipleMessages(t *testing.T) {
	dc := task.ExportNewDaemonContext("test-writer-multi")
	defer dc.Cancel()

	sub, err := dc.Subscribe(&task.SubscribeOpts{Replay: task.ReplayNone})
	assert.NoError(t, err)
	defer sub.Cancel()

	w := task.ExportNewDaemonResponseWriter(dc)

	// Write two messages in one write call
	batch := `data: {"type":"text","props":{"content":"msg1"}}` + "\n\n" +
		`data: {"type":"text","props":{"content":"msg2"}}` + "\n\n"
	n, err := w.Write([]byte(batch))
	assert.NoError(t, err)
	assert.Equal(t, len(batch), n)

	msg1 := <-sub.Ch
	assert.Equal(t, "msg1", msg1.Props["content"])

	msg2 := <-sub.Ch
	assert.Equal(t, "msg2", msg2.Props["content"])
}

func TestDaemonResponseWriter_SkipsDoneAndNonData(t *testing.T) {
	dc := task.ExportNewDaemonContext("test-writer-skip")
	defer dc.Cancel()

	sub, err := dc.Subscribe(&task.SubscribeOpts{Replay: task.ReplayNone})
	assert.NoError(t, err)
	defer sub.Cancel()

	w := task.ExportNewDaemonResponseWriter(dc)

	// Write [DONE] marker and non-data lines — should be skipped
	w.Write([]byte("data: [DONE]\n\n"))
	w.Write([]byte("event: ping\n\n"))
	w.Write([]byte(": keep-alive\n\n"))

	// Write a valid message after
	w.Write([]byte(`data: {"type":"text","props":{"content":"real"}}` + "\n\n"))

	select {
	case msg := <-sub.Ch:
		assert.Equal(t, "real", msg.Props["content"])
	case <-time.After(100 * time.Millisecond):
		t.Fatal("timeout")
	}
}

func TestDaemonResponseWriter_InvalidJSON(t *testing.T) {
	dc := task.ExportNewDaemonContext("test-writer-badjson")
	defer dc.Cancel()

	sub, err := dc.Subscribe(&task.SubscribeOpts{Replay: task.ReplayNone})
	assert.NoError(t, err)
	defer sub.Cancel()

	w := task.ExportNewDaemonResponseWriter(dc)

	// Invalid JSON — should be silently skipped
	w.Write([]byte("data: {invalid json}\n\n"))
	// Valid message follows
	w.Write([]byte(`data: {"type":"text","props":{"content":"ok"}}` + "\n\n"))

	select {
	case msg := <-sub.Ch:
		assert.Equal(t, "ok", msg.Props["content"])
	case <-time.After(100 * time.Millisecond):
		t.Fatal("timeout")
	}
}
