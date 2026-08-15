//go:build unit

package dsh_test

import (
	"context"
	"encoding/json"
	"io"
	"strings"
	"testing"

	"github.com/yaoapp/yao/agent/output/message"
	"github.com/yaoapp/yao/agent/sandbox/v2/dsh"
)

type recordedEvent struct {
	chunkType message.StreamChunkType
	data      []byte
}

func mockStreamFunc(events *[]recordedEvent) message.StreamFunc {
	return func(chunkType message.StreamChunkType, data []byte) int {
		cp := make([]byte, len(data))
		copy(cp, data)
		*events = append(*events, recordedEvent{chunkType: chunkType, data: cp})
		return 0
	}
}

func runParser(t *testing.T, ndjson string) ([]recordedEvent, bool) {
	t.Helper()
	var events []recordedEvent
	p := dsh.ExportNewStreamParser(mockStreamFunc(&events))
	r, w := io.Pipe()
	go func() {
		defer w.Close()
		w.Write([]byte(ndjson))
	}()
	if err := p.Parse(context.Background(), r); err != nil {
		t.Fatalf("parse error: %v", err)
	}
	return events, p.Completed()
}

type messageGroup struct {
	messageID string
	msgType   string
	events    []recordedEvent
}

func extractMessageGroups(events []recordedEvent) []messageGroup {
	var groups []messageGroup
	var cur *messageGroup
	for _, ev := range events {
		if ev.chunkType == message.ChunkMessageStart {
			var sd message.EventMessageStartData
			json.Unmarshal(ev.data, &sd)
			cur = &messageGroup{messageID: sd.MessageID, msgType: sd.Type}
		}
		if cur != nil {
			cur.events = append(cur.events, ev)
		}
		if ev.chunkType == message.ChunkMessageEnd {
			if cur != nil {
				groups = append(groups, *cur)
				cur = nil
			}
		}
	}
	return groups
}

func extractExecuteProps(ev recordedEvent) map[string]any {
	var m map[string]any
	json.Unmarshal(ev.data, &m)
	return m
}

func strDefault(v any) string {
	if v == nil {
		return ""
	}
	s, _ := v.(string)
	return s
}

// --- DSH NDJSON fixtures ---

const textOnlyNDJSON = `{"jsonrpc":"2.0","id":1,"result":{"sessionId":"sess-1"}}
{"jsonrpc":"2.0","method":"session.event","params":{"sessionId":"sess-1","event":{"type":"assistant/chunk","seq":1,"time":1000,"data":{"turn":1,"step":1,"chunk":{"type":"block-start","blockType":"text","index":0}}}}}
{"jsonrpc":"2.0","method":"session.event","params":{"sessionId":"sess-1","event":{"type":"assistant/chunk","seq":2,"time":1001,"data":{"turn":1,"step":1,"chunk":{"type":"text-delta","text":"Hello "}}}}}
{"jsonrpc":"2.0","method":"session.event","params":{"sessionId":"sess-1","event":{"type":"assistant/chunk","seq":3,"time":1002,"data":{"turn":1,"step":1,"chunk":{"type":"text-delta","text":"world!"}}}}}
{"jsonrpc":"2.0","method":"session.event","params":{"sessionId":"sess-1","event":{"type":"assistant/chunk","seq":4,"time":1003,"data":{"turn":1,"step":1,"chunk":{"type":"block-end","index":0}}}}}
{"jsonrpc":"2.0","method":"session.status","params":{"sessionId":"sess-1","status":"idle"}}
`

const thinkingNDJSON = `{"jsonrpc":"2.0","id":1,"result":{"sessionId":"sess-1"}}
{"jsonrpc":"2.0","method":"session.event","params":{"sessionId":"sess-1","event":{"type":"assistant/chunk","seq":1,"time":1000,"data":{"turn":1,"step":1,"chunk":{"type":"block-start","blockType":"reasoning","index":0}}}}}
{"jsonrpc":"2.0","method":"session.event","params":{"sessionId":"sess-1","event":{"type":"assistant/chunk","seq":2,"time":1001,"data":{"turn":1,"step":1,"chunk":{"type":"reasoning-delta","text":"Let me think..."}}}}}
{"jsonrpc":"2.0","method":"session.event","params":{"sessionId":"sess-1","event":{"type":"assistant/chunk","seq":3,"time":1002,"data":{"turn":1,"step":1,"chunk":{"type":"block-end","index":0}}}}}
{"jsonrpc":"2.0","method":"session.event","params":{"sessionId":"sess-1","event":{"type":"assistant/chunk","seq":4,"time":1003,"data":{"turn":1,"step":1,"chunk":{"type":"block-start","blockType":"text","index":1}}}}}
{"jsonrpc":"2.0","method":"session.event","params":{"sessionId":"sess-1","event":{"type":"assistant/chunk","seq":5,"time":1004,"data":{"turn":1,"step":1,"chunk":{"type":"text-delta","text":"The answer is 42."}}}}}
{"jsonrpc":"2.0","method":"session.event","params":{"sessionId":"sess-1","event":{"type":"assistant/chunk","seq":6,"time":1005,"data":{"turn":1,"step":1,"chunk":{"type":"block-end","index":1}}}}}
{"jsonrpc":"2.0","method":"session.status","params":{"sessionId":"sess-1","status":"idle"}}
`

const toolCallNDJSON = `{"jsonrpc":"2.0","id":1,"result":{"sessionId":"sess-1"}}
{"jsonrpc":"2.0","method":"session.event","params":{"sessionId":"sess-1","event":{"type":"assistant/chunk","seq":1,"time":1000,"data":{"turn":1,"step":1,"chunk":{"type":"block-start","blockType":"tool-call","index":0,"id":"call_1","name":"bash"}}}}}
{"jsonrpc":"2.0","method":"session.event","params":{"sessionId":"sess-1","event":{"type":"assistant/chunk","seq":2,"time":1001,"data":{"turn":1,"step":1,"chunk":{"type":"tool-call-delta","index":0,"argumentsDelta":"{\"command\""}}}}}
{"jsonrpc":"2.0","method":"session.event","params":{"sessionId":"sess-1","event":{"type":"assistant/chunk","seq":3,"time":1002,"data":{"turn":1,"step":1,"chunk":{"type":"tool-call-delta","index":0,"argumentsDelta":":\"echo hello\"}"}}}}}
{"jsonrpc":"2.0","method":"session.event","params":{"sessionId":"sess-1","event":{"type":"assistant/chunk","seq":4,"time":1003,"data":{"turn":1,"step":1,"chunk":{"type":"block-end","index":0}}}}}
{"jsonrpc":"2.0","method":"session.event","params":{"sessionId":"sess-1","event":{"type":"tool/call","seq":5,"time":1004,"data":{"turn":1,"step":1,"callId":"call_1","name":"bash","arguments":"{\"command\":\"echo hello\"}"}}}}
{"jsonrpc":"2.0","method":"session.event","params":{"sessionId":"sess-1","event":{"type":"tool/result","seq":6,"time":1005,"data":{"turn":1,"step":1,"message":{"source":{"callId":"call_1"},"content":[{"content":[{"type":"text","text":"hello\n"}]}]}}}}}
{"jsonrpc":"2.0","method":"session.event","params":{"sessionId":"sess-1","event":{"type":"assistant/chunk","seq":7,"time":1006,"data":{"turn":1,"step":2,"chunk":{"type":"block-start","blockType":"text","index":0}}}}}
{"jsonrpc":"2.0","method":"session.event","params":{"sessionId":"sess-1","event":{"type":"assistant/chunk","seq":8,"time":1007,"data":{"turn":1,"step":2,"chunk":{"type":"text-delta","text":"Done."}}}}}
{"jsonrpc":"2.0","method":"session.event","params":{"sessionId":"sess-1","event":{"type":"assistant/chunk","seq":9,"time":1008,"data":{"turn":1,"step":2,"chunk":{"type":"block-end","index":0}}}}}
{"jsonrpc":"2.0","method":"session.event","params":{"sessionId":"sess-1","event":{"type":"turn/end","seq":10,"time":1009,"data":{"turn":1,"reason":{"kind":"completed"}}}}}
{"jsonrpc":"2.0","method":"session.status","params":{"sessionId":"sess-1","status":"idle"}}
`

const errorResponseNDJSON = `{"jsonrpc":"2.0","id":1,"error":{"code":-32600,"message":"Invalid API key"}}
`

const usageNDJSON = `{"jsonrpc":"2.0","id":1,"result":{"sessionId":"sess-1"}}
{"jsonrpc":"2.0","method":"session.event","params":{"sessionId":"sess-1","event":{"type":"assistant/chunk","seq":1,"time":1000,"data":{"turn":1,"step":1,"chunk":{"type":"text-delta","text":"Hi"}}}}}
{"jsonrpc":"2.0","method":"session.event","params":{"sessionId":"sess-1","event":{"type":"assistant/chunk","seq":2,"time":1001,"data":{"turn":1,"step":1,"chunk":{"type":"usage","usage":{"input_tokens":50,"output_tokens":10}}}}}}
{"jsonrpc":"2.0","method":"session.status","params":{"sessionId":"sess-1","status":"idle"}}
`

// --- Tests ---

func TestParse_TextOnly_MessagePairing(t *testing.T) {
	events, completed := runParser(t, textOnlyNDJSON)

	if !completed {
		t.Fatal("parser should report completed on idle")
	}

	var startCount, endCount int
	depth := 0
	for _, ev := range events {
		switch ev.chunkType {
		case message.ChunkMessageStart:
			startCount++
			depth++
			if depth > 1 {
				t.Fatalf("nested message_start (depth %d)", depth)
			}
		case message.ChunkMessageEnd:
			endCount++
			depth--
			if depth < 0 {
				t.Fatal("message_end without start")
			}
		}
	}
	if startCount != endCount {
		t.Fatalf("start(%d) != end(%d)", startCount, endCount)
	}
	if startCount != 1 {
		t.Errorf("expected 1 text group, got %d", startCount)
	}
}

func TestParse_TextOnly_Content(t *testing.T) {
	events, _ := runParser(t, textOnlyNDJSON)

	var textChunks []string
	for _, ev := range events {
		if ev.chunkType == message.ChunkText {
			textChunks = append(textChunks, string(ev.data))
		}
	}
	fullText := strings.Join(textChunks, "")
	if fullText != "Hello world!" {
		t.Errorf("text = %q, want %q", fullText, "Hello world!")
	}
}

func TestParse_Thinking_ThenText(t *testing.T) {
	events, completed := runParser(t, thinkingNDJSON)
	if !completed {
		t.Fatal("should complete")
	}

	groups := extractMessageGroups(events)
	if len(groups) != 2 {
		t.Fatalf("expected 2 groups (thinking + text), got %d", len(groups))
	}
	if groups[0].msgType != "thinking" {
		t.Errorf("group 0 type = %q, want thinking", groups[0].msgType)
	}
	if groups[1].msgType != "text" {
		t.Errorf("group 1 type = %q, want text", groups[1].msgType)
	}

	var thinkText, respText string
	for _, ev := range events {
		if ev.chunkType == message.ChunkThinking {
			thinkText += string(ev.data)
		}
		if ev.chunkType == message.ChunkText {
			respText += string(ev.data)
		}
	}
	if thinkText != "Let me think..." {
		t.Errorf("thinking = %q", thinkText)
	}
	if respText != "The answer is 42." {
		t.Errorf("text = %q", respText)
	}
}

func TestParse_ToolCall_TwoPhaseLifecycle(t *testing.T) {
	events, completed := runParser(t, toolCallNDJSON)
	if !completed {
		t.Fatal("should complete")
	}

	groups := extractMessageGroups(events)

	var executeGroups []messageGroup
	for _, g := range groups {
		if g.msgType == "execute" {
			executeGroups = append(executeGroups, g)
		}
	}

	if len(executeGroups) != 1 {
		t.Fatalf("expected exactly 1 execute group (single lifecycle), got %d", len(executeGroups))
	}

	var statuses []string
	for _, ev := range executeGroups[0].events {
		if ev.chunkType == message.ChunkExecute {
			props := extractExecuteProps(ev)
			if s := strDefault(props["status"]); s != "" {
				statuses = append(statuses, s)
			}
		}
	}
	if len(statuses) != 2 || statuses[0] != "running" || statuses[1] != "completed" {
		t.Errorf("status progression: %v (want [running completed])", statuses)
	}
}

func TestParse_ToolCall_Summary(t *testing.T) {
	events, _ := runParser(t, toolCallNDJSON)

	var summaries []string
	for _, ev := range events {
		if ev.chunkType == message.ChunkExecute {
			props := extractExecuteProps(ev)
			if s := strDefault(props["summary"]); s != "" {
				summaries = append(summaries, s)
			}
		}
	}
	found := false
	for _, s := range summaries {
		if strings.Contains(s, "echo hello") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected summary containing 'echo hello', got %v", summaries)
	}
}

func TestParse_ToolCall_Output(t *testing.T) {
	events, _ := runParser(t, toolCallNDJSON)

	var output string
	for _, ev := range events {
		if ev.chunkType == message.ChunkExecute {
			props := extractExecuteProps(ev)
			if o := strDefault(props["output"]); o != "" {
				output = o
			}
		}
	}
	if !strings.Contains(output, "hello") {
		t.Errorf("output = %q, want contains 'hello'", output)
	}
}

func TestParse_ToolCall_FinalText(t *testing.T) {
	events, _ := runParser(t, toolCallNDJSON)
	groups := extractMessageGroups(events)

	var textGroups []messageGroup
	for _, g := range groups {
		if g.msgType == "text" {
			textGroups = append(textGroups, g)
		}
	}
	if len(textGroups) == 0 {
		t.Fatal("expected at least 1 text group after tool execution")
	}

	var texts []string
	for _, ev := range events {
		if ev.chunkType == message.ChunkText {
			texts = append(texts, string(ev.data))
		}
	}
	fullText := strings.Join(texts, "")
	if !strings.Contains(fullText, "Done.") {
		t.Errorf("final text = %q", fullText)
	}
}

func TestParse_ErrorResponse(t *testing.T) {
	events, _ := runParser(t, errorResponseNDJSON)

	var errorText string
	for _, ev := range events {
		if ev.chunkType == message.ChunkError {
			errorText = string(ev.data)
		}
	}
	if !strings.Contains(errorText, "Invalid API key") {
		t.Errorf("error = %q, want contains 'Invalid API key'", errorText)
	}
}

func TestParse_Usage_Metadata(t *testing.T) {
	events, completed := runParser(t, usageNDJSON)
	if !completed {
		t.Fatal("should complete")
	}

	var usageData []byte
	for _, ev := range events {
		if ev.chunkType == message.ChunkMetadata {
			usageData = ev.data
		}
	}
	if usageData == nil {
		t.Fatal("expected metadata event with usage")
	}
	var meta map[string]any
	json.Unmarshal(usageData, &meta)
	usage, ok := meta["usage"]
	if !ok {
		t.Fatal("metadata missing 'usage' key")
	}
	usageMap, ok := usage.(map[string]any)
	if !ok {
		t.Fatalf("usage is %T, want map", usage)
	}
	if usageMap["input_tokens"] != float64(50) {
		t.Errorf("input_tokens = %v", usageMap["input_tokens"])
	}
}

func TestParse_TurnEnd_NonCompleted(t *testing.T) {
	ndjson := `{"jsonrpc":"2.0","id":1,"result":{"sessionId":"sess-1"}}
{"jsonrpc":"2.0","method":"session.event","params":{"sessionId":"sess-1","event":{"type":"turn/end","seq":1,"time":1000,"data":{"turn":1,"reason":{"kind":"max_tokens"}}}}}
{"jsonrpc":"2.0","method":"session.status","params":{"sessionId":"sess-1","status":"idle"}}
`
	events, _ := runParser(t, ndjson)

	var errorText string
	for _, ev := range events {
		if ev.chunkType == message.ChunkError {
			errorText = string(ev.data)
		}
	}
	if !strings.Contains(errorText, "max_tokens") {
		t.Errorf("expected error about max_tokens, got %q", errorText)
	}
}

func TestParse_TurnEnd_Completed_NoError(t *testing.T) {
	ndjson := `{"jsonrpc":"2.0","id":1,"result":{"sessionId":"sess-1"}}
{"jsonrpc":"2.0","method":"session.event","params":{"sessionId":"sess-1","event":{"type":"turn/end","seq":1,"time":1000,"data":{"turn":1,"reason":{"kind":"completed"}}}}}
{"jsonrpc":"2.0","method":"session.status","params":{"sessionId":"sess-1","status":"idle"}}
`
	events, _ := runParser(t, ndjson)

	for _, ev := range events {
		if ev.chunkType == message.ChunkError {
			t.Fatalf("unexpected error event: %s", ev.data)
		}
	}
}

// --- extractToolSummary ---

func TestExtractToolSummary_Bash(t *testing.T) {
	got := dsh.ExportExtractToolSummary("bash", `{"command":"npm install"}`)
	if got != "npm install" {
		t.Errorf("got %q", got)
	}
}

func TestExtractToolSummary_Write(t *testing.T) {
	got := dsh.ExportExtractToolSummary("write", `{"file_path":"/workspace/main.go"}`)
	if got != "/workspace/main.go" {
		t.Errorf("got %q", got)
	}
}

func TestExtractToolSummary_Read(t *testing.T) {
	got := dsh.ExportExtractToolSummary("read", `{"path":"/workspace/go.mod"}`)
	if got != "/workspace/go.mod" {
		t.Errorf("got %q", got)
	}
}

func TestExtractToolSummary_Empty(t *testing.T) {
	if got := dsh.ExportExtractToolSummary("bash", ""); got != "" {
		t.Errorf("got %q", got)
	}
}

func TestExtractToolSummary_InvalidJSON(t *testing.T) {
	if got := dsh.ExportExtractToolSummary("bash", "{invalid"); got != "" {
		t.Errorf("got %q", got)
	}
}

// --- truncateStr ---

func TestTruncateStr_Short(t *testing.T) {
	if got := dsh.ExportTruncateStr("hello", 80); got != "hello" {
		t.Errorf("got %q", got)
	}
}

func TestTruncateStr_Long(t *testing.T) {
	long := strings.Repeat("x", 100)
	got := dsh.ExportTruncateStr(long, 80)
	if len(got) != 83 {
		t.Errorf("got len %d, expected 83", len(got))
	}
}

func TestTruncateStr_Newlines(t *testing.T) {
	got := dsh.ExportTruncateStr("line1\nline2\nline3", 80)
	if strings.Contains(got, "\n") {
		t.Errorf("got %q, should not contain newlines", got)
	}
}

// --- injectDSHSemanticType ---

func TestInjectDSHSemanticType_Subagent(t *testing.T) {
	props := map[string]any{}
	dsh.ExportInjectDSHSemanticType(props, "subagent")
	if props["semantic_type"] != message.TypeAgent {
		t.Errorf("semantic_type = %v", props["semantic_type"])
	}
}

func TestInjectDSHSemanticType_Todo(t *testing.T) {
	props := map[string]any{}
	dsh.ExportInjectDSHSemanticType(props, "todo_write")
	if props["semantic_type"] != message.TypeTodo {
		t.Errorf("semantic_type = %v", props["semantic_type"])
	}
}

func TestInjectDSHSemanticType_Unknown(t *testing.T) {
	props := map[string]any{}
	dsh.ExportInjectDSHSemanticType(props, "bash")
	if _, ok := props["semantic_type"]; ok {
		t.Error("should not set semantic_type for unknown tool")
	}
}

// TestParse_IdleAfterResponse verifies that session.status "idle" (not the
// prompt response) marks the stream as completed. The prompt response is an
// asynchronous acknowledgment sent before the agent loop streams events.
func TestParse_IdleAfterResponse(t *testing.T) {
	ndjson := `{"jsonrpc":"2.0","id":1,"result":{"sessionId":"sess-1"}}
{"jsonrpc":"2.0","id":2,"result":{"ok":true}}
{"jsonrpc":"2.0","method":"session.event","params":{"sessionId":"sess-1","event":{"type":"assistant/chunk","seq":1,"time":1000,"data":{"turn":1,"step":1,"chunk":{"type":"text-delta","text":"你好！"}}}}}
{"jsonrpc":"2.0","method":"session.event","params":{"sessionId":"sess-1","event":{"type":"turn/end","seq":2,"time":1001,"data":{"turn":1,"reason":{"kind":"completed"}}}}}
{"jsonrpc":"2.0","method":"session.status","params":{"sessionId":"sess-1","status":"idle"}}
`
	events, completed := runParser(t, ndjson)

	if !completed {
		t.Fatal("parser should report completed on session.status idle")
	}

	var hasText bool
	for _, ev := range events {
		if ev.chunkType == message.ChunkText && string(ev.data) == "你好！" {
			hasText = true
		}
	}
	if !hasText {
		t.Error("expected text chunk '你好！'")
	}
}

// TestParse_ResponseOnly_NoSessionStatus verifies that content is delivered
// even when DSH sends only the response without session.status. Completion
// comes from stdout EOF; the parser returns completed=false but the content
// was still emitted to the handler.
func TestParse_ResponseOnly_NoSessionStatus(t *testing.T) {
	ndjson := `{"jsonrpc":"2.0","id":1,"result":{"sessionId":"sess-1"}}
{"jsonrpc":"2.0","id":2,"result":{"ok":true}}
{"jsonrpc":"2.0","method":"session.event","params":{"sessionId":"sess-1","event":{"type":"assistant/chunk","seq":1,"time":1000,"data":{"turn":1,"step":1,"chunk":{"type":"block-start","blockType":"text","index":0}}}}}
{"jsonrpc":"2.0","method":"session.event","params":{"sessionId":"sess-1","event":{"type":"assistant/chunk","seq":2,"time":1001,"data":{"turn":1,"step":1,"chunk":{"type":"text-delta","text":"Hello"}}}}}
{"jsonrpc":"2.0","method":"session.event","params":{"sessionId":"sess-1","event":{"type":"assistant/chunk","seq":3,"time":1002,"data":{"turn":1,"step":1,"chunk":{"type":"block-end","index":0}}}}}
`
	events, completed := runParser(t, ndjson)

	if completed {
		t.Fatal("parser should not report completed without session.status idle")
	}

	var starts, ends int
	for _, ev := range events {
		switch ev.chunkType {
		case message.ChunkMessageStart:
			starts++
		case message.ChunkMessageEnd:
			ends++
		}
	}
	if starts != ends {
		t.Errorf("message starts (%d) != ends (%d)", starts, ends)
	}
}
