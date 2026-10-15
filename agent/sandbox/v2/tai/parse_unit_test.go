package tai_test

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/yaoapp/yao/agent/output/message"
	"github.com/yaoapp/yao/agent/sandbox/v2/tai"
)

// chunk records a single StreamFunc call for assertion.
type chunk struct {
	Type message.StreamChunkType
	Data string
}

// collector gathers all StreamFunc calls.
type collector struct {
	chunks []chunk
}

func (c *collector) handler(chunkType message.StreamChunkType, data []byte) int {
	c.chunks = append(c.chunks, chunk{Type: chunkType, Data: string(data)})
	return 0
}

func (c *collector) hasType(t message.StreamChunkType) bool {
	for _, ch := range c.chunks {
		if ch.Type == t {
			return true
		}
	}
	return false
}

func (c *collector) countType(t message.StreamChunkType) int {
	n := 0
	for _, ch := range c.chunks {
		if ch.Type == t {
			n++
		}
	}
	return n
}

func (c *collector) dataForType(t message.StreamChunkType) []string {
	var out []string
	for _, ch := range c.chunks {
		if ch.Type == t {
			out = append(out, ch.Data)
		}
	}
	return out
}

func (c *collector) lastExecute() map[string]any {
	for i := len(c.chunks) - 1; i >= 0; i-- {
		if c.chunks[i].Type == message.ChunkExecute {
			var m map[string]any
			json.Unmarshal([]byte(c.chunks[i].Data), &m)
			return m
		}
	}
	return nil
}

// ---------------------------------------------------------------------------
// Core flow tests
// ---------------------------------------------------------------------------

func TestTextDeltaSequence(t *testing.T) {
	c := &collector{}
	p := tai.NewExportStreamParser(c.handler)

	p.Dispatch(map[string]any{"type": "message_update", "event_type": "text_start", "content_index": 0})
	p.Dispatch(map[string]any{"type": "message_update", "event_type": "text_delta", "delta": "Hello "})
	p.Dispatch(map[string]any{"type": "message_update", "event_type": "text_delta", "delta": "world"})
	p.Dispatch(map[string]any{"type": "message_update", "event_type": "text_end", "content_index": 0})

	if !c.hasType(message.ChunkMessageStart) {
		t.Fatal("expected MessageStart")
	}
	texts := c.dataForType(message.ChunkText)
	if len(texts) != 2 {
		t.Fatalf("expected 2 text chunks, got %d", len(texts))
	}
	if texts[0] != "Hello " || texts[1] != "world" {
		t.Fatalf("unexpected text: %v", texts)
	}
	if !c.hasType(message.ChunkMessageEnd) {
		t.Fatal("expected MessageEnd")
	}
}

func TestThinkingDeltaSequence(t *testing.T) {
	c := &collector{}
	p := tai.NewExportStreamParser(c.handler)

	p.Dispatch(map[string]any{"type": "message_update", "event_type": "thinking_start", "content_index": 0})
	p.Dispatch(map[string]any{"type": "message_update", "event_type": "thinking_delta", "delta": "Let me think..."})
	p.Dispatch(map[string]any{"type": "message_update", "event_type": "thinking_end", "content_index": 0})

	thinks := c.dataForType(message.ChunkThinking)
	if len(thinks) != 1 || thinks[0] != "Let me think..." {
		t.Fatalf("unexpected thinking: %v", thinks)
	}
}

func TestThinkingToTextSwitch(t *testing.T) {
	c := &collector{}
	p := tai.NewExportStreamParser(c.handler)

	// Thinking phase.
	p.Dispatch(map[string]any{"type": "message_update", "event_type": "thinking_delta", "delta": "hmm"})
	// Switch to text: should close thinking, open text.
	p.Dispatch(map[string]any{"type": "message_update", "event_type": "text_delta", "delta": "Answer"})

	// Expect: MessageStart(thinking) + thinking + MessageEnd + MessageStart(text) + text
	starts := c.countType(message.ChunkMessageStart)
	ends := c.countType(message.ChunkMessageEnd)
	if starts != 2 {
		t.Fatalf("expected 2 MessageStarts, got %d", starts)
	}
	// thinking_end not sent, but text_delta auto-closes thinking.
	if ends < 1 {
		t.Fatal("expected at least 1 MessageEnd for thinking close")
	}
}

func TestTextToToolCallSwitch(t *testing.T) {
	c := &collector{}
	p := tai.NewExportStreamParser(c.handler)

	p.Dispatch(map[string]any{"type": "message_update", "event_type": "text_delta", "delta": "I'll read"})
	// toolcall_start should close text.
	p.Dispatch(map[string]any{"type": "message_update", "event_type": "toolcall_start", "content_index": 0})

	texts := c.dataForType(message.ChunkText)
	if len(texts) != 1 {
		t.Fatalf("expected 1 text chunk, got %d", len(texts))
	}
	// Should have execute chunk from toolcall_start.
	if !c.hasType(message.ChunkExecute) {
		t.Fatal("expected execute chunk from toolcall_start")
	}
}

func TestToolCallLLMPhase(t *testing.T) {
	c := &collector{}
	p := tai.NewExportStreamParser(c.handler)

	p.Dispatch(map[string]any{"type": "message_update", "event_type": "toolcall_start", "content_index": 0})
	p.Dispatch(map[string]any{"type": "message_update", "event_type": "toolcall_delta", "content_index": 0, "delta": `{"command":`})
	p.Dispatch(map[string]any{"type": "message_update", "event_type": "toolcall_delta", "content_index": 0, "delta": `"ls -la"}`})
	p.Dispatch(map[string]any{
		"type": "message_update", "event_type": "toolcall_end", "content_index": 0,
		"tool_call": map[string]any{"id": "call_abc", "name": "bash", "arguments": `{"command":"ls -la"}`},
	})

	// Should have execute chunks.
	if !c.hasType(message.ChunkExecute) {
		t.Fatal("expected execute chunk")
	}
}

func TestToolExecutionPhase(t *testing.T) {
	c := &collector{}
	p := tai.NewExportStreamParser(c.handler)

	p.Dispatch(map[string]any{
		"type": "tool_start", "tool_id": "call_1", "tool": "read",
		"args": `{"file_path":"main.go"}`,
	})
	p.Dispatch(map[string]any{
		"type": "tool_end", "tool_id": "call_1", "tool": "read",
		"result": `{"content":"package main"}`, "duration_ms": int64(42),
	})

	execs := c.dataForType(message.ChunkExecute)
	if len(execs) < 2 {
		t.Fatalf("expected at least 2 execute chunks (start+end), got %d", len(execs))
	}

	last := c.lastExecute()
	if last == nil {
		t.Fatal("no execute chunk found")
	}
	if last["status"] != "completed" {
		t.Fatalf("expected status=completed, got %v", last["status"])
	}
	if last["output"] != "package main" {
		t.Fatalf("expected output='package main', got %v (%T)", last["output"], last["output"])
	}
}

func TestToolExecutionWithUpdate(t *testing.T) {
	c := &collector{}
	p := tai.NewExportStreamParser(c.handler)

	p.Dispatch(map[string]any{
		"type": "tool_start", "tool_id": "call_2", "tool": "bash",
		"args": `{"command":"make build"}`,
	})
	p.Dispatch(map[string]any{
		"type": "tool_update", "tool_id": "call_2", "tool": "bash",
		"partial_result": `{"content":"compiling..."}`,
	})
	p.Dispatch(map[string]any{
		"type": "tool_end", "tool_id": "call_2", "tool": "bash",
		"result": `{"content":"build complete"}`,
	})

	execs := c.dataForType(message.ChunkExecute)
	if len(execs) < 3 {
		t.Fatalf("expected at least 3 execute chunks (start+update+end), got %d", len(execs))
	}

	// Verify tool_update extracts content string from PartialResult.
	var updateExec map[string]any
	json.Unmarshal([]byte(execs[1]), &updateExec)
	if updateExec["output"] != "compiling..." {
		t.Fatalf("expected update output='compiling...', got %v (%T)", updateExec["output"], updateExec["output"])
	}

	// Verify tool_end extracts content string from Result.
	var endExec map[string]any
	json.Unmarshal([]byte(execs[2]), &endExec)
	if endExec["output"] != "build complete" {
		t.Fatalf("expected end output='build complete', got %v (%T)", endExec["output"], endExec["output"])
	}
}

func TestTwoPhaseToolIDReuse(t *testing.T) {
	c := &collector{}
	p := tai.NewExportStreamParser(c.handler)

	// LLM phase: toolcall_start → toolcall_end with real ID.
	p.Dispatch(map[string]any{"type": "message_update", "event_type": "toolcall_start", "content_index": 0})
	p.Dispatch(map[string]any{
		"type": "message_update", "event_type": "toolcall_end", "content_index": 0,
		"tool_call": map[string]any{"id": "real_call_1", "name": "read", "arguments": `{"file_path":"x.go"}`},
	})

	// Execution phase: tool_start with real ID should reuse msgID.
	p.Dispatch(map[string]any{
		"type": "tool_start", "tool_id": "real_call_1", "tool": "read",
		"args": `{"file_path":"x.go"}`,
	})
	p.Dispatch(map[string]any{
		"type": "tool_end", "tool_id": "real_call_1", "tool": "read",
		"result": `{"content":"file contents"}`,
	})

	// All execute events for this tool should share the same message group.
	// Count distinct MessageStart IDs — the tool_start should reuse the LLM phase's msgID.
	var msgIDs []string
	for _, ch := range c.chunks {
		if ch.Type == message.ChunkMessageStart {
			var sd message.EventMessageStartData
			json.Unmarshal([]byte(ch.Data), &sd)
			if sd.Type == "execute" {
				msgIDs = append(msgIDs, sd.MessageID)
			}
		}
	}

	if len(msgIDs) < 2 {
		t.Fatalf("expected at least 2 execute MessageStarts, got %d", len(msgIDs))
	}
	// The first (from toolcall_start) and second (from toolcall_end update) may differ,
	// but tool_start (third) should reuse one of the earlier IDs.
	lastID := msgIDs[len(msgIDs)-1]
	reused := false
	for i := 0; i < len(msgIDs)-1; i++ {
		if msgIDs[i] == lastID {
			reused = true
			break
		}
	}
	if !reused {
		t.Fatalf("tool_start did not reuse LLM phase msgID: %v", msgIDs)
	}
}

func TestMixedFlow(t *testing.T) {
	c := &collector{}
	p := tai.NewExportStreamParser(c.handler)

	// thinking → text → tool → text → agent_end
	p.Dispatch(map[string]any{"type": "message_update", "event_type": "thinking_delta", "delta": "hmm"})
	p.Dispatch(map[string]any{"type": "message_update", "event_type": "thinking_end", "content_index": 0})
	p.Dispatch(map[string]any{"type": "message_update", "event_type": "text_delta", "delta": "I'll"})
	p.Dispatch(map[string]any{"type": "tool_start", "tool_id": "c1", "tool": "bash", "args": `{"command":"ls"}`})
	p.Dispatch(map[string]any{"type": "tool_end", "tool_id": "c1", "tool": "bash", "result": `{"content":"."}`})
	p.Dispatch(map[string]any{"type": "message_update", "event_type": "text_delta", "delta": " done"})
	stopped := p.Dispatch(map[string]any{"type": "agent_end"})

	if !stopped {
		t.Fatal("expected agent_end to stop parsing")
	}
	if !p.Completed() {
		t.Fatal("expected completed=true")
	}
	if !c.hasType(message.ChunkStreamEnd) {
		t.Fatal("expected ChunkStreamEnd")
	}
}

func TestAgentEnd(t *testing.T) {
	c := &collector{}
	p := tai.NewExportStreamParser(c.handler)

	stopped := p.Dispatch(map[string]any{"type": "agent_end"})

	if !stopped {
		t.Fatal("expected agent_end to stop")
	}
	if !p.Completed() {
		t.Fatal("expected completed=true")
	}
	if !c.hasType(message.ChunkStreamEnd) {
		t.Fatal("expected ChunkStreamEnd")
	}
}

func TestOrphanedToolCleanup(t *testing.T) {
	c := &collector{}
	p := tai.NewExportStreamParser(c.handler)

	// Start a tool but never end it.
	p.Dispatch(map[string]any{
		"type": "tool_start", "tool_id": "orphan_1", "tool": "bash",
		"args": `{"command":"sleep 100"}`,
	})
	// agent_end triggers orphan cleanup.
	p.Dispatch(map[string]any{"type": "agent_end"})

	// The orphaned tool should get an error execute chunk.
	found := false
	for _, ch := range c.chunks {
		if ch.Type == message.ChunkExecute && strings.Contains(ch.Data, "session ended before tool completed") {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("expected orphaned tool cleanup error execute chunk")
	}
}

func TestErrorEvent(t *testing.T) {
	c := &collector{}
	p := tai.NewExportStreamParser(c.handler)

	p.Dispatch(map[string]any{"type": "error", "error": "connection failed"})

	if !c.hasType(message.ChunkError) {
		t.Fatal("expected ChunkError")
	}
	errors := c.dataForType(message.ChunkError)
	if len(errors) != 1 || errors[0] != "connection failed" {
		t.Fatalf("unexpected error: %v", errors)
	}
}

func TestMessageUpdateError(t *testing.T) {
	c := &collector{}
	p := tai.NewExportStreamParser(c.handler)

	p.Dispatch(map[string]any{
		"type": "message_update", "event_type": "error",
		"stop_reason": "overloaded", "error": "rate limit exceeded",
	})

	if !c.hasType(message.ChunkError) {
		t.Fatal("expected ChunkError from message_update error")
	}
}

// ---------------------------------------------------------------------------
// Operational flow tests
// ---------------------------------------------------------------------------

func TestCompactionLifecycle(t *testing.T) {
	c := &collector{}
	p := tai.NewExportStreamParser(c.handler)

	p.Dispatch(map[string]any{"type": "compaction_start", "reason": "token_limit"})
	p.Dispatch(map[string]any{
		"type": "compaction_end", "reason": "token_limit",
		"tokens_before": 50000, "tokens_after": 10000,
	})

	if !c.hasType(message.ChunkLoading) {
		t.Fatal("expected ChunkLoading from compaction")
	}
	if !c.hasType(message.ChunkMetadata) {
		t.Fatal("expected ChunkMetadata from compaction_end")
	}
}

func TestAutoRetryLifecycle(t *testing.T) {
	c := &collector{}
	p := tai.NewExportStreamParser(c.handler)

	p.Dispatch(map[string]any{
		"type": "auto_retry_start", "attempt": 1, "max_attempts": 3, "delay_ms": 2000,
	})
	p.Dispatch(map[string]any{"type": "auto_retry_end"})

	loadings := c.countType(message.ChunkLoading)
	if loadings < 2 {
		t.Fatalf("expected at least 2 loading chunks (start+end), got %d", loadings)
	}
}

func TestToolEndWithError(t *testing.T) {
	c := &collector{}
	p := tai.NewExportStreamParser(c.handler)

	p.Dispatch(map[string]any{
		"type": "tool_start", "tool_id": "err_1", "tool": "bash",
		"args": `{"command":"fail"}`,
	})
	p.Dispatch(map[string]any{
		"type": "tool_end", "tool_id": "err_1", "tool": "bash",
		"result": `{"content":"permission denied"}`, "is_error": true,
	})

	last := c.lastExecute()
	if last == nil {
		t.Fatal("no execute chunk")
	}
	if last["status"] != "error" {
		t.Fatalf("expected status=error, got %v", last["status"])
	}
	if last["is_error"] != true {
		t.Fatalf("expected is_error=true, got %v", last["is_error"])
	}
	if last["output"] != "permission denied" {
		t.Fatalf("expected output='permission denied', got %v (%T)", last["output"], last["output"])
	}
}

func TestCompactionAfterContent(t *testing.T) {
	c := &collector{}
	p := tai.NewExportStreamParser(c.handler)

	// Content already started.
	p.Dispatch(map[string]any{"type": "message_update", "event_type": "text_delta", "delta": "hi"})
	p.Dispatch(map[string]any{"type": "message_update", "event_type": "text_end", "content_index": 0})

	// Compaction loading MUST still emit after content has started.
	p.Dispatch(map[string]any{"type": "compaction_start", "reason": "token_limit"})

	if !c.hasType(message.ChunkLoading) {
		t.Fatal("compaction loading should NOT be dropped after contentStarted")
	}
}

func TestTurnStartEnd(t *testing.T) {
	c := &collector{}
	p := tai.NewExportStreamParser(c.handler)

	p.Dispatch(map[string]any{"type": "turn_start"})
	p.Dispatch(map[string]any{"type": "message_update", "event_type": "text_delta", "delta": "hi"})
	p.Dispatch(map[string]any{"type": "turn_end"})

	// turn_start/turn_end emit loading or metadata only when Turn is set.
	// Verify that text flows through correctly despite turn boundaries.
	texts := c.dataForType(message.ChunkText)
	if len(texts) != 1 || texts[0] != "hi" {
		t.Fatalf("expected text 'hi', got %v", texts)
	}
	// turn_start always emits a loading chunk.
	if !c.hasType(message.ChunkLoading) {
		t.Fatal("expected ChunkLoading from turn_start")
	}
}

func TestDoneEventUsage(t *testing.T) {
	c := &collector{}
	p := tai.NewExportStreamParser(c.handler)

	p.Dispatch(map[string]any{
		"type": "message_update", "event_type": "done",
		"stop_reason": "end_turn",
		"usage":       `{"input":100,"output":50,"total_tokens":150}`,
	})

	metadatas := c.dataForType(message.ChunkMetadata)
	found := false
	for _, m := range metadatas {
		if strings.Contains(m, "stop_reason") && strings.Contains(m, "end_turn") {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected metadata with stop_reason, got: %v", metadatas)
	}
}

// ---------------------------------------------------------------------------
// Info flow tests
// ---------------------------------------------------------------------------

func TestSessionInfo(t *testing.T) {
	c := &collector{}
	p := tai.NewExportStreamParser(c.handler)

	p.Dispatch(map[string]any{"type": "session_info", "session_name": "my task"})

	metadatas := c.dataForType(message.ChunkMetadata)
	found := false
	for _, m := range metadatas {
		if strings.Contains(m, "my task") {
			found = true
		}
	}
	if !found {
		t.Fatal("expected session_name in metadata")
	}
}

func TestThinkingLevelChanged(t *testing.T) {
	c := &collector{}
	p := tai.NewExportStreamParser(c.handler)

	p.Dispatch(map[string]any{"type": "thinking_level", "thinking_level": "high"})

	metadatas := c.dataForType(message.ChunkMetadata)
	found := false
	for _, m := range metadatas {
		if strings.Contains(m, "high") {
			found = true
		}
	}
	if !found {
		t.Fatal("expected thinking_level in metadata")
	}
}

func TestSessionMetadata(t *testing.T) {
	c := &collector{}
	p := tai.NewExportStreamParser(c.handler)

	p.Dispatch(map[string]any{"type": "session", "session_id": "sess-123"})

	metadatas := c.dataForType(message.ChunkMetadata)
	found := false
	for _, m := range metadatas {
		if strings.Contains(m, "sess-123") {
			found = true
		}
	}
	if !found {
		t.Fatal("expected tai_session_id in metadata")
	}
}

func TestBashDeltaFallback(t *testing.T) {
	c := &collector{}
	p := tai.NewExportStreamParser(c.handler)

	// Start a tool, then send bash_delta without tool_id.
	p.Dispatch(map[string]any{
		"type": "tool_start", "tool_id": "call_x", "tool": "bash",
		"args": `{"command":"make"}`,
	})
	p.Dispatch(map[string]any{"type": "bash_delta", "delta": "compiling..."})

	execs := c.dataForType(message.ChunkExecute)
	found := false
	for _, e := range execs {
		if strings.Contains(e, "compiling...") {
			found = true
		}
	}
	if !found {
		t.Fatal("expected bash_delta output in execute chunk")
	}
}

// ---------------------------------------------------------------------------
// Summary helper tests
// ---------------------------------------------------------------------------

func TestExtractToolSummary(t *testing.T) {
	tests := []struct {
		name     string
		tool     string
		input    string
		expected string
	}{
		{"bash command", "bash", `{"command":"ls -la"}`, "ls -la"},
		{"read path", "read", `{"file_path":"main.go"}`, "main.go"},
		{"write path", "write", `{"file_path":"out.txt"}`, "out.txt"},
		{"edit path", "edit", `{"file_path":"x.go"}`, "x.go"},
		{"generic path", "unknown", `{"path":"/tmp/log"}`, "/tmp/log"},
		{"generic url", "fetch", `{"url":"https://example.com"}`, "https://example.com"},
		{"empty", "bash", `{}`, ""},
		{"invalid json", "bash", `not json`, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tai.ExportExtractToolSummary(tt.tool, tt.input)
			if got != tt.expected {
				t.Fatalf("extractToolSummary(%q, %q) = %q, want %q", tt.tool, tt.input, got, tt.expected)
			}
		})
	}
}

func TestTruncateStr(t *testing.T) {
	long := strings.Repeat("a", 100)
	got := tai.ExportTruncateStr(long, 80)
	if len(got) != 83 { // 80 + "..."
		t.Fatalf("expected length 83, got %d", len(got))
	}
	if !strings.HasSuffix(got, "...") {
		t.Fatal("expected ... suffix")
	}

	short := "hello"
	if tai.ExportTruncateStr(short, 80) != "hello" {
		t.Fatal("short string should not be truncated")
	}
}

// ---------------------------------------------------------------------------
// Subagent summary + semantic_type tests
// ---------------------------------------------------------------------------

func TestExtractToolSummary_SubagentRun(t *testing.T) {
	got := tai.ExportExtractToolSummary("subagent_run", `{"name":"doc-researcher","prompt":"summarize","description":"Summarize docs"}`)
	if got != "doc-researcher: Summarize docs" {
		t.Fatalf("got %q", got)
	}
}

func TestExtractToolSummary_SubagentCreate(t *testing.T) {
	got := tai.ExportExtractToolSummary("subagent_create", `{"name":"reviewer","description":"Code reviewer"}`)
	if got != "Create reviewer: Code reviewer" {
		t.Fatalf("got %q", got)
	}
}

func TestExtractToolSummary_SubagentList(t *testing.T) {
	// subagent_list has no special branch; the fallback should return empty for no known keys.
	got := tai.ExportExtractToolSummary("subagent_list", `{}`)
	if got != "" {
		t.Fatalf("got %q, want empty", got)
	}
}

func TestInjectTaiSemanticType_SubagentRun(t *testing.T) {
	var datas []json.RawMessage
	handler := func(_ message.StreamChunkType, data []byte) int {
		datas = append(datas, json.RawMessage(data))
		return 0
	}

	parser := tai.NewExportStreamParser(handler)
	parser.Dispatch(map[string]any{
		"type":    "tool_start",
		"tool":    "subagent_run",
		"tool_id": "tc_1",
		"args":    `{"name":"test","prompt":"hello","description":"Test task"}`,
	})

	// Find the ChunkExecute and verify semantic_type.
	found := false
	for _, data := range datas {
		if data == nil {
			continue
		}
		var props map[string]any
		if err := json.Unmarshal(data, &props); err != nil {
			continue
		}
		if st, ok := props["semantic_type"].(string); ok && st == string(message.TypeAgent) {
			found = true
			break
		}
	}
	if !found {
		t.Error("subagent_run should have semantic_type=agent")
	}
}
