//go:build unit

package task_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/yaoapp/gou/process"
	"github.com/yaoapp/yao/agent/output/message"
	"github.com/yaoapp/yao/agent/task"
)

// nextTextMsg drains event messages and returns the next non-event message
func nextTextMsg(ch <-chan *message.Message, timeout time.Duration) *message.Message {
	timer := time.After(timeout)
	for {
		select {
		case msg, ok := <-ch:
			if !ok {
				return nil
			}
			if msg.Type == "event" {
				continue
			}
			return msg
		case <-timer:
			return nil
		}
	}
}

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

	// Drain read_complete first
	time.Sleep(10 * time.Millisecond)
	drainEvents(sub.Ch)

	w := task.ExportNewDaemonResponseWriter(dc)

	// Write partial data (no newline terminator yet) — should NOT produce a text message
	w.Write([]byte(`data: {"type":"text","props":{"content":"part1`))

	select {
	case msg := <-sub.Ch:
		if msg.Type != "event" {
			t.Fatal("should not receive text message before line is complete")
		}
	case <-time.After(50 * time.Millisecond):
		// expected: no message yet
	}

	// Complete the message with closing JSON + newlines
	w.Write([]byte(`"}}` + "\n\n"))

	msg := nextTextMsg(sub.Ch, time.Second)
	if msg == nil {
		t.Fatal("timeout waiting for partial write completion")
	}
	assert.Equal(t, "text", msg.Type)
	assert.Equal(t, "part1", msg.Props["content"])
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

	msg1 := nextTextMsg(sub.Ch, time.Second)
	assert.NotNil(t, msg1)
	assert.Equal(t, "msg1", msg1.Props["content"])

	msg2 := nextTextMsg(sub.Ch, time.Second)
	assert.NotNil(t, msg2)
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

	msg := nextTextMsg(sub.Ch, time.Second)
	if msg == nil {
		t.Fatal("timeout")
	}
	assert.Equal(t, "real", msg.Props["content"])
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

	msg := nextTextMsg(sub.Ch, time.Second)
	if msg == nil {
		t.Fatal("timeout")
	}
	assert.Equal(t, "ok", msg.Props["content"])
}

// drainEvents drains any pending event messages from the channel
func drainEvents(ch <-chan *message.Message) {
	for {
		select {
		case msg := <-ch:
			if msg.Type != "event" {
				return
			}
		case <-time.After(50 * time.Millisecond):
			return
		}
	}
}

// --- contentText tests ---

func TestContentText_String(t *testing.T) {
	assert.Equal(t, "hello", task.ExportContentText("hello"))
}

func TestContentText_EmptyString(t *testing.T) {
	assert.Equal(t, "", task.ExportContentText(""))
}

func TestContentText_Nil(t *testing.T) {
	assert.Equal(t, "", task.ExportContentText(nil))
}

func TestContentText_MultipartSingleText(t *testing.T) {
	content := []interface{}{
		map[string]interface{}{"type": "text", "text": "look at this"},
	}
	assert.Equal(t, "look at this", task.ExportContentText(content))
}

func TestContentText_MultipartMixed(t *testing.T) {
	content := []interface{}{
		map[string]interface{}{"type": "text", "text": "image:"},
		map[string]interface{}{"type": "image_url", "image_url": map[string]interface{}{"url": "file://x.png"}},
		map[string]interface{}{"type": "text", "text": "analyze it"},
	}
	assert.Equal(t, "image:\nanalyze it", task.ExportContentText(content))
}

func TestContentText_MultipartNoText(t *testing.T) {
	content := []interface{}{
		map[string]interface{}{"type": "image_url", "image_url": map[string]interface{}{"url": "file://x.png"}},
	}
	assert.Equal(t, "", task.ExportContentText(content))
}

// --- Multipart InputMessage tests ---

func TestInputToAgentMessages_MultipartContent(t *testing.T) {
	multipart := []interface{}{
		map[string]interface{}{"type": "text", "text": "hello"},
		map[string]interface{}{"type": "image_url", "image_url": map[string]interface{}{"url": "file://img.png"}},
	}
	msgs := []task.InputMessage{
		{Role: "user", Content: multipart},
	}
	result := task.ExportInputToAgentMessages(msgs)
	assert.Len(t, result, 1)
	assert.Equal(t, "user", string(result[0].Role))
	parts, ok := result[0].Content.([]interface{})
	assert.True(t, ok)
	assert.Len(t, parts, 2)
}

// --- extractContentFromProps tests (simulates real DB props JSON) ---

func TestExtractContentFromProps_PlainText(t *testing.T) {
	// This is what DB stores for a plain text message
	propsJSON := `{"content":"hello world","type":"text"}`
	result := task.ExportExtractContentFromProps(propsJSON)
	assert.Equal(t, "hello world", result)
}

func TestExtractContentFromProps_Multipart(t *testing.T) {
	// This is what DB stores for a multipart message with image attachment
	propsJSON := `{"content":[{"type":"text","text":"看这图"},{"type":"image_url","image_url":{"url":"__yao.attachment://file-abc"}}],"type":"text"}`
	result := task.ExportExtractContentFromProps(propsJSON)

	// Should be []interface{} not string
	parts, ok := result.([]interface{})
	assert.True(t, ok, "multipart content should be []interface{}, got %T", result)
	assert.Len(t, parts, 2)

	// Verify first part is text
	part0, ok := parts[0].(map[string]interface{})
	assert.True(t, ok)
	assert.Equal(t, "text", part0["type"])
	assert.Equal(t, "看这图", part0["text"])

	// Verify second part is image_url
	part1, ok := parts[1].(map[string]interface{})
	assert.True(t, ok)
	assert.Equal(t, "image_url", part1["type"])
}

func TestExtractContentFromProps_MultipartFlowToContentText(t *testing.T) {
	// End-to-end: DB props → extractContentFromProps → contentText → string
	propsJSON := `{"content":[{"type":"text","text":"analyze this image"},{"type":"image_url","image_url":{"url":"file://x.png"}}]}`
	content := task.ExportExtractContentFromProps(propsJSON)

	// contentText should extract only text parts
	text := task.ExportContentText(content)
	assert.Equal(t, "analyze this image", text)
}

func TestExtractContentFromProps_EmptyJSON(t *testing.T) {
	assert.Equal(t, "", task.ExportExtractContentFromProps(""))
}

func TestExtractContentFromProps_NoContent(t *testing.T) {
	assert.Equal(t, "", task.ExportExtractContentFromProps(`{"type":"text"}`))
}

func TestExtractContentFromProps_InvalidJSON(t *testing.T) {
	assert.Equal(t, "", task.ExportExtractContentFromProps(`{invalid`))
}

func TestExtractContentFromProps_RetryFlowSimulation(t *testing.T) {
	// Simulate full retry flow: GetOriginalPrompt returns multipart,
	// then it's used as InputMessage.Content, then flows to inputToAgentMessages
	propsJSON := `{"content":[{"type":"text","text":"check this"},{"type":"image_url","image_url":{"url":"file://img.png"}}]}`
	originalPrompt := task.ExportExtractContentFromProps(propsJSON)

	// Simulate: messages := []InputMessage{{Role: "user", Content: originalPrompt}}
	// Then append extra retry message
	messages := []task.InputMessage{
		{Role: "user", Content: originalPrompt},
		{Role: "user", Content: "also try a different approach"},
	}

	// Verify inputToAgentMessages preserves both
	result := task.ExportInputToAgentMessages(messages)
	assert.Len(t, result, 2)

	// First message should have multipart content
	parts, ok := result[0].Content.([]interface{})
	assert.True(t, ok, "original prompt should remain []interface{}, got %T", result[0].Content)
	assert.Len(t, parts, 2)

	// Second message should be plain string
	assert.Equal(t, "also try a different approach", result[1].Content)
}
