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

func runParserWithSession(t *testing.T, ndjson, sessionID string) ([]recordedEvent, bool) {
	t.Helper()
	var events []recordedEvent
	p := dsh.ExportNewStreamParser(mockStreamFunc(&events), sessionID)
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

	// 4 groups: block-start(running) + block-end(running+summary) + tool/call(running) + tool/result(completed)
	if len(executeGroups) != 4 {
		t.Fatalf("expected 4 execute groups, got %d", len(executeGroups))
	}

	// All four groups share the same messageID
	msgID := executeGroups[0].messageID
	for i, g := range executeGroups {
		if g.messageID != msgID {
			t.Errorf("group %d messageID = %q, want %q", i, g.messageID, msgID)
		}
	}

	// All statuses are running except the last (completed)
	for i, g := range executeGroups {
		for _, ev := range g.events {
			if ev.chunkType == message.ChunkExecute {
				props := extractExecuteProps(ev)
				s := strDefault(props["status"])
				if i < len(executeGroups)-1 {
					if s != "running" {
						t.Errorf("group %d status = %q, want running", i, s)
					}
				} else {
					if s != "completed" {
						t.Errorf("last group status = %q, want completed", s)
					}
				}
			}
		}
	}

	// block-end group (index 1) should carry the summary
	for _, ev := range executeGroups[1].events {
		if ev.chunkType == message.ChunkExecute {
			props := extractExecuteProps(ev)
			if s := strDefault(props["summary"]); !strings.Contains(s, "echo hello") {
				t.Errorf("block-end group summary = %q, want contains 'echo hello'", s)
			}
		}
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

// --- Subagent idle fixture: main session = sess-1, subagent = sess-sub ---

const subagentIdleNDJSON = `{"jsonrpc":"2.0","id":1,"result":{"sessionId":"sess-1"}}
{"jsonrpc":"2.0","method":"session.event","params":{"sessionId":"sess-1","event":{"type":"assistant/chunk","seq":1,"time":1000,"data":{"turn":1,"step":1,"chunk":{"type":"text-delta","text":"Spawning sub..."}}}}}
{"jsonrpc":"2.0","method":"session.status","params":{"sessionId":"sess-sub","status":"idle"}}
{"jsonrpc":"2.0","method":"session.event","params":{"sessionId":"sess-1","event":{"type":"assistant/chunk","seq":2,"time":1002,"data":{"turn":1,"step":1,"chunk":{"type":"text-delta","text":" Done."}}}}}
{"jsonrpc":"2.0","method":"session.status","params":{"sessionId":"sess-1","status":"idle"}}
`

const subagentEventNDJSON = `{"jsonrpc":"2.0","id":1,"result":{"sessionId":"sess-1"}}
{"jsonrpc":"2.0","method":"session.event","params":{"sessionId":"sess-sub","event":{"type":"assistant/chunk","seq":1,"time":1000,"data":{"turn":1,"step":1,"chunk":{"type":"text-delta","text":"sub text"}}}}}
{"jsonrpc":"2.0","method":"session.event","params":{"sessionId":"sess-1","event":{"type":"assistant/chunk","seq":1,"time":1001,"data":{"turn":1,"step":1,"chunk":{"type":"text-delta","text":"main text"}}}}}
{"jsonrpc":"2.0","method":"session.status","params":{"sessionId":"sess-1","status":"idle"}}
`

// parallelToolCallNDJSON: two tool calls streamed sequentially in Phase 1,
// then both executed in parallel in Phase 2.
const parallelToolCallNDJSON = `{"jsonrpc":"2.0","id":1,"result":{"sessionId":"sess-1"}}
{"jsonrpc":"2.0","method":"session.event","params":{"sessionId":"sess-1","event":{"type":"assistant/chunk","seq":1,"time":1000,"data":{"turn":1,"step":1,"chunk":{"type":"block-start","blockType":"tool-call","index":0,"id":"call_1","name":"bash"}}}}}
{"jsonrpc":"2.0","method":"session.event","params":{"sessionId":"sess-1","event":{"type":"assistant/chunk","seq":2,"time":1001,"data":{"turn":1,"step":1,"chunk":{"type":"tool-call-delta","index":0,"argumentsDelta":"{\"command\":\"ls\"}"}}}}}
{"jsonrpc":"2.0","method":"session.event","params":{"sessionId":"sess-1","event":{"type":"assistant/chunk","seq":3,"time":1002,"data":{"turn":1,"step":1,"chunk":{"type":"block-end","index":0}}}}}
{"jsonrpc":"2.0","method":"session.event","params":{"sessionId":"sess-1","event":{"type":"assistant/chunk","seq":4,"time":1003,"data":{"turn":1,"step":1,"chunk":{"type":"block-start","blockType":"tool-call","index":1,"id":"call_2","name":"read"}}}}}
{"jsonrpc":"2.0","method":"session.event","params":{"sessionId":"sess-1","event":{"type":"assistant/chunk","seq":5,"time":1004,"data":{"turn":1,"step":1,"chunk":{"type":"tool-call-delta","index":1,"argumentsDelta":"{\"path\":\"/tmp/f.txt\"}"}}}}}
{"jsonrpc":"2.0","method":"session.event","params":{"sessionId":"sess-1","event":{"type":"assistant/chunk","seq":6,"time":1005,"data":{"turn":1,"step":1,"chunk":{"type":"block-end","index":1}}}}}
{"jsonrpc":"2.0","method":"session.event","params":{"sessionId":"sess-1","event":{"type":"tool/call","seq":7,"time":1006,"data":{"turn":1,"step":1,"callId":"call_1","name":"bash","arguments":"{\"command\":\"ls\"}"}}}}
{"jsonrpc":"2.0","method":"session.event","params":{"sessionId":"sess-1","event":{"type":"tool/call","seq":8,"time":1007,"data":{"turn":1,"step":1,"callId":"call_2","name":"read","arguments":"{\"path\":\"/tmp/f.txt\"}"}}}}
{"jsonrpc":"2.0","method":"session.event","params":{"sessionId":"sess-1","event":{"type":"tool/result","seq":9,"time":1008,"data":{"turn":1,"step":1,"message":{"source":{"callId":"call_1"},"content":[{"content":[{"type":"text","text":"file1.txt\n"}]}]}}}}}
{"jsonrpc":"2.0","method":"session.event","params":{"sessionId":"sess-1","event":{"type":"tool/result","seq":10,"time":1009,"data":{"turn":1,"step":1,"message":{"source":{"callId":"call_2"},"content":[{"content":[{"type":"text","text":"hello world"}]}]}}}}}
{"jsonrpc":"2.0","method":"session.status","params":{"sessionId":"sess-1","status":"idle"}}
`

// --- New tests ---

func TestParse_SubagentIdle_DoesNotComplete(t *testing.T) {
	events, completed := runParserWithSession(t, subagentIdleNDJSON, "sess-1")
	if !completed {
		t.Fatal("should complete when main session goes idle")
	}

	var fullText string
	for _, ev := range events {
		if ev.chunkType == message.ChunkText {
			fullText += string(ev.data)
		}
	}
	if fullText != "Spawning sub... Done." {
		t.Errorf("text = %q, want %q", fullText, "Spawning sub... Done.")
	}
}

func TestParse_SubagentEvent_Filtered(t *testing.T) {
	events, completed := runParserWithSession(t, subagentEventNDJSON, "sess-1")
	if !completed {
		t.Fatal("should complete")
	}

	var fullText string
	for _, ev := range events {
		if ev.chunkType == message.ChunkText {
			fullText += string(ev.data)
		}
	}
	if fullText != "main text" {
		t.Errorf("text = %q, want %q (subagent text should be filtered)", fullText, "main text")
	}
}

func TestParse_ToolCall_PendingPhase1(t *testing.T) {
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
	if len(executeGroups) < 2 {
		t.Fatal("expected at least 2 execute groups (block-start + block-end)")
	}

	// First execute group (block-start): running, tool name, runner
	first := executeGroups[0]
	for _, ev := range first.events {
		if ev.chunkType == message.ChunkExecute {
			props := extractExecuteProps(ev)
			if strDefault(props["status"]) != "running" {
				t.Errorf("first execute group status = %q, want running", strDefault(props["status"]))
			}
			if strDefault(props["tool"]) != "bash" {
				t.Errorf("first execute group tool = %q, want bash", strDefault(props["tool"]))
			}
			if strDefault(props["runner"]) != "dsh" {
				t.Errorf("first execute group runner = %q, want dsh", strDefault(props["runner"]))
			}
		}
	}

	// Second execute group (block-end): running with summary
	second := executeGroups[1]
	for _, ev := range second.events {
		if ev.chunkType == message.ChunkExecute {
			props := extractExecuteProps(ev)
			if strDefault(props["status"]) != "running" {
				t.Errorf("second execute group status = %q, want running", strDefault(props["status"]))
			}
			if s := strDefault(props["summary"]); !strings.Contains(s, "echo hello") {
				t.Errorf("second execute group summary = %q, want contains 'echo hello'", s)
			}
		}
	}
}

func TestParse_ToolCall_Phase2ReusesGroup(t *testing.T) {
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
	if len(executeGroups) < 4 {
		t.Fatalf("expected 4 execute groups, got %d", len(executeGroups))
	}

	// All groups share the same messageID
	firstID := executeGroups[0].messageID
	for i, g := range executeGroups {
		if g.messageID != firstID {
			t.Errorf("group %d msgID %q != first msgID %q", i, g.messageID, firstID)
		}
	}
}

func TestParse_Idle_ClosesExecuteGroup(t *testing.T) {
	// Tool call with no tool/result before idle — verify orphaned tool gets error status
	ndjson := `{"jsonrpc":"2.0","id":1,"result":{"sessionId":"sess-1"}}
{"jsonrpc":"2.0","method":"session.event","params":{"sessionId":"sess-1","event":{"type":"assistant/chunk","seq":1,"time":1000,"data":{"turn":1,"step":1,"chunk":{"type":"block-start","blockType":"tool-call","index":0,"id":"call_orphan","name":"bash"}}}}}
{"jsonrpc":"2.0","method":"session.event","params":{"sessionId":"sess-1","event":{"type":"assistant/chunk","seq":2,"time":1001,"data":{"turn":1,"step":1,"chunk":{"type":"block-end","index":0}}}}}
{"jsonrpc":"2.0","method":"session.status","params":{"sessionId":"sess-1","status":"idle"}}
`
	events, completed := runParser(t, ndjson)
	if !completed {
		t.Fatal("should complete on idle")
	}

	// The pending execute group should be properly paired (start + end)
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

	// Orphaned tool must receive error status from closeOrphanedTools
	var hasError bool
	for _, ev := range events {
		if ev.chunkType == message.ChunkExecute {
			props := extractExecuteProps(ev)
			if strDefault(props["status"]) == "error" {
				hasError = true
				if strDefault(props["tool"]) != "bash" {
					t.Errorf("error tool = %q, want bash", strDefault(props["tool"]))
				}
			}
		}
	}
	if !hasError {
		t.Error("expected error status for orphaned tool on idle")
	}
}

func TestParse_ParallelToolCalls(t *testing.T) {
	events, completed := runParser(t, parallelToolCallNDJSON)
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

	// 2 tools × 4 phases (block-start, block-end, tool/call, tool/result) = 8 execute groups
	if len(executeGroups) != 8 {
		t.Fatalf("expected 8 execute groups for 2 parallel tools, got %d", len(executeGroups))
	}

	// Collect messageIDs and statuses per tool
	toolGroupIDs := map[string][]string{} // tool_id -> list of messageIDs
	toolStatuses := map[string][]string{} // tool_id -> list of statuses
	for _, g := range executeGroups {
		for _, ev := range g.events {
			if ev.chunkType == message.ChunkExecute {
				props := extractExecuteProps(ev)
				toolID := strDefault(props["tool_id"])
				if toolID != "" {
					toolGroupIDs[toolID] = append(toolGroupIDs[toolID], g.messageID)
					toolStatuses[toolID] = append(toolStatuses[toolID], strDefault(props["status"]))
				}
			}
		}
	}

	// Each tool should have 4 groups with same messageID
	for _, tid := range []string{"call_1", "call_2"} {
		ids := toolGroupIDs[tid]
		statuses := toolStatuses[tid]
		if len(ids) != 4 {
			t.Fatalf("tool %s: expected 4 groups, got %d", tid, len(ids))
		}
		for i := 1; i < len(ids); i++ {
			if ids[i] != ids[0] {
				t.Errorf("tool %s: messageIDs not consistent: %v", tid, ids)
			}
		}
		want := []string{"running", "running", "running", "completed"}
		for i, s := range statuses {
			if s != want[i] {
				t.Errorf("tool %s status[%d] = %q, want %q", tid, i, s, want[i])
			}
		}
	}

	// call_1 and call_2 should have DIFFERENT messageIDs
	if len(toolGroupIDs["call_1"]) > 0 && len(toolGroupIDs["call_2"]) > 0 {
		if toolGroupIDs["call_1"][0] == toolGroupIDs["call_2"][0] {
			t.Error("parallel tools should have different messageIDs")
		}
	}

	// Message pairing: all starts == all ends
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

// --- Session ID filtering edge cases ---

func TestParse_NoSessionFilter_AcceptsAll(t *testing.T) {
	// When promptedSessionID is empty, all events pass through
	events, completed := runParser(t, subagentIdleNDJSON)
	if !completed {
		t.Fatal("should complete on first idle (no filtering)")
	}

	var fullText string
	for _, ev := range events {
		if ev.chunkType == message.ChunkText {
			fullText += string(ev.data)
		}
	}
	// Without filtering, parser completes on subagent's idle (sess-sub) before "Done."
	if fullText != "Spawning sub..." {
		t.Errorf("text = %q, want %q (no filter means first idle completes)", fullText, "Spawning sub...")
	}
}

func TestParse_ContextCancel(t *testing.T) {
	ndjson := `{"jsonrpc":"2.0","id":1,"result":{"sessionId":"sess-1"}}
{"jsonrpc":"2.0","method":"session.event","params":{"sessionId":"sess-1","event":{"type":"assistant/chunk","seq":1,"time":1000,"data":{"turn":1,"step":1,"chunk":{"type":"text-delta","text":"hello"}}}}}
`
	var events []recordedEvent
	p := dsh.ExportNewStreamParser(mockStreamFunc(&events))
	ctx, cancel := context.WithCancel(context.Background())

	r, w := io.Pipe()
	go func() {
		w.Write([]byte(ndjson))
		cancel()
		w.Close()
	}()
	_ = p.Parse(ctx, r)
	// Should not hang or panic
}

func TestParse_ToolCallDelta_IDMapping(t *testing.T) {
	// block-start with synthetic ID, then delta with real callId
	ndjson := `{"jsonrpc":"2.0","id":1,"result":{"sessionId":"sess-1"}}
{"jsonrpc":"2.0","method":"session.event","params":{"sessionId":"sess-1","event":{"type":"assistant/chunk","seq":1,"time":1000,"data":{"turn":1,"step":1,"chunk":{"type":"block-start","blockType":"tool-call","index":0,"id":"temp_0","name":"bash"}}}}}
{"jsonrpc":"2.0","method":"session.event","params":{"sessionId":"sess-1","event":{"type":"assistant/chunk","seq":2,"time":1001,"data":{"turn":1,"step":1,"chunk":{"type":"tool-call-delta","index":0,"id":"real_call_1","argumentsDelta":"{\"command\":\"pwd\"}"}}}}}
{"jsonrpc":"2.0","method":"session.event","params":{"sessionId":"sess-1","event":{"type":"assistant/chunk","seq":3,"time":1002,"data":{"turn":1,"step":1,"chunk":{"type":"block-end","index":0}}}}}
{"jsonrpc":"2.0","method":"session.event","params":{"sessionId":"sess-1","event":{"type":"tool/call","seq":4,"time":1003,"data":{"turn":1,"step":1,"callId":"real_call_1","name":"bash","arguments":"{\"command\":\"pwd\"}"}}}}
{"jsonrpc":"2.0","method":"session.event","params":{"sessionId":"sess-1","event":{"type":"tool/result","seq":5,"time":1004,"data":{"turn":1,"step":1,"message":{"source":{"callId":"real_call_1"},"content":[{"content":[{"type":"text","text":"/home\n"}]}]}}}}}
{"jsonrpc":"2.0","method":"session.status","params":{"sessionId":"sess-1","status":"idle"}}
`
	events, completed := runParser(t, ndjson)
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
	// block-start(running) + block-end(running+summary) + tool/call(running) + tool/result(completed)
	// all reuse same messageID via ID mapping (real_call_1 mapped to temp_0's msgID)
	if len(executeGroups) != 4 {
		t.Fatalf("expected 4 execute groups, got %d", len(executeGroups))
	}
	msgID := executeGroups[0].messageID
	for i, g := range executeGroups {
		if g.messageID != msgID {
			t.Errorf("group %d messageID = %q, want %q (ID mapping failed)", i, g.messageID, msgID)
		}
	}
}

func TestParse_ToolError(t *testing.T) {
	ndjson := `{"jsonrpc":"2.0","id":1,"result":{"sessionId":"sess-1"}}
{"jsonrpc":"2.0","method":"session.event","params":{"sessionId":"sess-1","event":{"type":"assistant/chunk","seq":1,"time":1000,"data":{"turn":1,"step":1,"chunk":{"type":"block-start","blockType":"tool-call","index":0,"id":"call_err","name":"bash"}}}}}
{"jsonrpc":"2.0","method":"session.event","params":{"sessionId":"sess-1","event":{"type":"assistant/chunk","seq":2,"time":1001,"data":{"turn":1,"step":1,"chunk":{"type":"tool-call-delta","index":0,"argumentsDelta":"{\"command\":\"exit 1\"}"}}}}}
{"jsonrpc":"2.0","method":"session.event","params":{"sessionId":"sess-1","event":{"type":"assistant/chunk","seq":3,"time":1002,"data":{"turn":1,"step":1,"chunk":{"type":"block-end","index":0}}}}}
{"jsonrpc":"2.0","method":"session.event","params":{"sessionId":"sess-1","event":{"type":"tool/call","seq":4,"time":1003,"data":{"turn":1,"step":1,"callId":"call_err","name":"bash","arguments":"{\"command\":\"exit 1\"}"}}}}
{"jsonrpc":"2.0","method":"session.event","params":{"sessionId":"sess-1","event":{"type":"tool/result","seq":5,"time":1004,"data":{"turn":1,"step":1,"message":{"source":{"callId":"call_err"},"content":[{"isError":true,"content":[{"type":"text","text":"command failed"}]}]}}}}}
{"jsonrpc":"2.0","method":"session.status","params":{"sessionId":"sess-1","status":"idle"}}
`
	events, completed := runParser(t, ndjson)
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

	// Last execute group should have status "error"
	lastGroup := executeGroups[len(executeGroups)-1]
	for _, ev := range lastGroup.events {
		if ev.chunkType == message.ChunkExecute {
			props := extractExecuteProps(ev)
			if strDefault(props["status"]) != "error" {
				t.Errorf("error tool result status = %q, want error", strDefault(props["status"]))
			}
		}
	}
}

func TestParse_MixedThinkTextTool(t *testing.T) {
	ndjson := `{"jsonrpc":"2.0","id":1,"result":{"sessionId":"sess-1"}}
{"jsonrpc":"2.0","method":"session.event","params":{"sessionId":"sess-1","event":{"type":"assistant/chunk","seq":1,"time":1000,"data":{"turn":1,"step":1,"chunk":{"type":"block-start","blockType":"reasoning","index":0}}}}}
{"jsonrpc":"2.0","method":"session.event","params":{"sessionId":"sess-1","event":{"type":"assistant/chunk","seq":2,"time":1001,"data":{"turn":1,"step":1,"chunk":{"type":"reasoning-delta","text":"I should run a command"}}}}}
{"jsonrpc":"2.0","method":"session.event","params":{"sessionId":"sess-1","event":{"type":"assistant/chunk","seq":3,"time":1002,"data":{"turn":1,"step":1,"chunk":{"type":"block-end","index":0}}}}}
{"jsonrpc":"2.0","method":"session.event","params":{"sessionId":"sess-1","event":{"type":"assistant/chunk","seq":4,"time":1003,"data":{"turn":1,"step":1,"chunk":{"type":"block-start","blockType":"tool-call","index":1,"id":"call_mix","name":"bash"}}}}}
{"jsonrpc":"2.0","method":"session.event","params":{"sessionId":"sess-1","event":{"type":"assistant/chunk","seq":5,"time":1004,"data":{"turn":1,"step":1,"chunk":{"type":"tool-call-delta","index":1,"argumentsDelta":"{\"command\":\"date\"}"}}}}}
{"jsonrpc":"2.0","method":"session.event","params":{"sessionId":"sess-1","event":{"type":"assistant/chunk","seq":6,"time":1005,"data":{"turn":1,"step":1,"chunk":{"type":"block-end","index":1}}}}}
{"jsonrpc":"2.0","method":"session.event","params":{"sessionId":"sess-1","event":{"type":"tool/call","seq":7,"time":1006,"data":{"turn":1,"step":1,"callId":"call_mix","name":"bash","arguments":"{\"command\":\"date\"}"}}}}
{"jsonrpc":"2.0","method":"session.event","params":{"sessionId":"sess-1","event":{"type":"tool/result","seq":8,"time":1007,"data":{"turn":1,"step":1,"message":{"source":{"callId":"call_mix"},"content":[{"content":[{"type":"text","text":"Mon Aug 17"}]}]}}}}}
{"jsonrpc":"2.0","method":"session.event","params":{"sessionId":"sess-1","event":{"type":"assistant/chunk","seq":9,"time":1008,"data":{"turn":1,"step":2,"chunk":{"type":"block-start","blockType":"text","index":0}}}}}
{"jsonrpc":"2.0","method":"session.event","params":{"sessionId":"sess-1","event":{"type":"assistant/chunk","seq":10,"time":1009,"data":{"turn":1,"step":2,"chunk":{"type":"text-delta","text":"Today is Monday."}}}}}
{"jsonrpc":"2.0","method":"session.event","params":{"sessionId":"sess-1","event":{"type":"assistant/chunk","seq":11,"time":1010,"data":{"turn":1,"step":2,"chunk":{"type":"block-end","index":0}}}}}
{"jsonrpc":"2.0","method":"session.status","params":{"sessionId":"sess-1","status":"idle"}}
`
	events, completed := runParser(t, ndjson)
	if !completed {
		t.Fatal("should complete")
	}

	groups := extractMessageGroups(events)
	var types []string
	for _, g := range groups {
		types = append(types, g.msgType)
	}
	// thinking, execute(block-start), execute(block-end+summary), execute(tool/call), execute(tool/result), text
	want := []string{"thinking", "execute", "execute", "execute", "execute", "text"}
	if len(types) != len(want) {
		t.Fatalf("group types = %v, want %v", types, want)
	}
	for i := range types {
		if types[i] != want[i] {
			t.Errorf("group[%d] type = %q, want %q", i, types[i], want[i])
		}
	}
}

func TestParse_ToolCallWithoutPhase1(t *testing.T) {
	// tool/call arrives without prior block-start — should still work
	ndjson := `{"jsonrpc":"2.0","id":1,"result":{"sessionId":"sess-1"}}
{"jsonrpc":"2.0","method":"session.event","params":{"sessionId":"sess-1","event":{"type":"tool/call","seq":1,"time":1000,"data":{"turn":1,"step":1,"callId":"direct_call","name":"read","arguments":"{\"path\":\"/tmp/x\"}"}}}}
{"jsonrpc":"2.0","method":"session.event","params":{"sessionId":"sess-1","event":{"type":"tool/result","seq":2,"time":1001,"data":{"turn":1,"step":1,"message":{"source":{"callId":"direct_call"},"content":[{"content":[{"type":"text","text":"content"}]}]}}}}}
{"jsonrpc":"2.0","method":"session.status","params":{"sessionId":"sess-1","status":"idle"}}
`
	events, completed := runParser(t, ndjson)
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
	// running + completed = 2 groups (no pending since no Phase 1)
	if len(executeGroups) != 2 {
		t.Fatalf("expected 2 execute groups (no Phase 1), got %d", len(executeGroups))
	}

	// Both should share the same messageID
	if executeGroups[0].messageID != executeGroups[1].messageID {
		t.Error("running and completed should share same messageID")
	}
}

func TestParse_FinishChunk(t *testing.T) {
	ndjson := `{"jsonrpc":"2.0","id":1,"result":{"sessionId":"sess-1"}}
{"jsonrpc":"2.0","method":"session.event","params":{"sessionId":"sess-1","event":{"type":"assistant/chunk","seq":1,"time":1000,"data":{"turn":1,"step":1,"chunk":{"type":"text-delta","text":"Hi"}}}}}
{"jsonrpc":"2.0","method":"session.event","params":{"sessionId":"sess-1","event":{"type":"assistant/chunk","seq":2,"time":1001,"data":{"turn":1,"step":1,"chunk":{"type":"finish","reason":{"kind":"completed"}}}}}}
{"jsonrpc":"2.0","method":"session.status","params":{"sessionId":"sess-1","status":"idle"}}
`
	events, completed := runParser(t, ndjson)
	if !completed {
		t.Fatal("should complete")
	}
	var hasMetadata bool
	for _, ev := range events {
		if ev.chunkType == message.ChunkMetadata {
			var m map[string]any
			json.Unmarshal(ev.data, &m)
			if _, ok := m["finish_reason"]; ok {
				hasMetadata = true
			}
		}
	}
	if !hasMetadata {
		t.Error("expected metadata event with finish_reason")
	}
}

func TestParse_ReasoningDelta_WithoutBlockStart(t *testing.T) {
	ndjson := `{"jsonrpc":"2.0","id":1,"result":{"sessionId":"sess-1"}}
{"jsonrpc":"2.0","method":"session.event","params":{"sessionId":"sess-1","event":{"type":"assistant/chunk","seq":1,"time":1000,"data":{"turn":1,"step":1,"chunk":{"type":"reasoning-delta","text":"thinking..."}}}}}
{"jsonrpc":"2.0","method":"session.event","params":{"sessionId":"sess-1","event":{"type":"assistant/chunk","seq":2,"time":1001,"data":{"turn":1,"step":1,"chunk":{"type":"reasoning-delta","text":" more"}}}}}
{"jsonrpc":"2.0","method":"session.status","params":{"sessionId":"sess-1","status":"idle"}}
`
	events, completed := runParser(t, ndjson)
	if !completed {
		t.Fatal("should complete")
	}
	var thinkText string
	for _, ev := range events {
		if ev.chunkType == message.ChunkThinking {
			thinkText += string(ev.data)
		}
	}
	if thinkText != "thinking... more" {
		t.Errorf("thinking text = %q, want 'thinking... more'", thinkText)
	}

	groups := extractMessageGroups(events)
	if len(groups) != 1 || groups[0].msgType != "thinking" {
		t.Errorf("expected 1 thinking group, got %d groups", len(groups))
	}
}

func TestParse_TextDelta_CRLFNormalization(t *testing.T) {
	ndjson := `{"jsonrpc":"2.0","id":1,"result":{"sessionId":"sess-1"}}
{"jsonrpc":"2.0","method":"session.event","params":{"sessionId":"sess-1","event":{"type":"assistant/chunk","seq":1,"time":1000,"data":{"turn":1,"step":1,"chunk":{"type":"text-delta","text":"line1\r\nline2\rline3"}}}}}
{"jsonrpc":"2.0","method":"session.status","params":{"sessionId":"sess-1","status":"idle"}}
`
	events, _ := runParser(t, ndjson)
	var text string
	for _, ev := range events {
		if ev.chunkType == message.ChunkText {
			text += string(ev.data)
		}
	}
	if text != "line1\nline2\nline3" {
		t.Errorf("text = %q, want CRLF/CR normalized to LF", text)
	}
}

func TestParse_MultiContentToolResult(t *testing.T) {
	ndjson := `{"jsonrpc":"2.0","id":1,"result":{"sessionId":"sess-1"}}
{"jsonrpc":"2.0","method":"session.event","params":{"sessionId":"sess-1","event":{"type":"tool/call","seq":1,"time":1000,"data":{"turn":1,"step":1,"callId":"multi_out","name":"bash","arguments":"{\"command\":\"echo a && echo b\"}"}}}}
{"jsonrpc":"2.0","method":"session.event","params":{"sessionId":"sess-1","event":{"type":"tool/result","seq":2,"time":1001,"data":{"turn":1,"step":1,"message":{"source":{"callId":"multi_out"},"content":[{"content":[{"type":"text","text":"line1"}]},{"content":[{"type":"text","text":"line2"}]}]}}}}}
{"jsonrpc":"2.0","method":"session.status","params":{"sessionId":"sess-1","status":"idle"}}
`
	events, completed := runParser(t, ndjson)
	if !completed {
		t.Fatal("should complete")
	}
	var output string
	for _, ev := range events {
		if ev.chunkType == message.ChunkExecute {
			props := extractExecuteProps(ev)
			if o := strDefault(props["output"]); o != "" {
				output = o
			}
		}
	}
	if output != "line1\nline2" {
		t.Errorf("multi-content output = %q, want 'line1\\nline2'", output)
	}
}

func TestParse_EmptyToolResult(t *testing.T) {
	ndjson := `{"jsonrpc":"2.0","id":1,"result":{"sessionId":"sess-1"}}
{"jsonrpc":"2.0","method":"session.event","params":{"sessionId":"sess-1","event":{"type":"tool/call","seq":1,"time":1000,"data":{"turn":1,"step":1,"callId":"empty_out","name":"bash","arguments":"{\"command\":\"true\"}"}}}}
{"jsonrpc":"2.0","method":"session.event","params":{"sessionId":"sess-1","event":{"type":"tool/result","seq":2,"time":1001,"data":{"turn":1,"step":1,"message":{"source":{"callId":"empty_out"},"content":[]}}}}}
{"jsonrpc":"2.0","method":"session.status","params":{"sessionId":"sess-1","status":"idle"}}
`
	events, completed := runParser(t, ndjson)
	if !completed {
		t.Fatal("should complete")
	}
	// Should have completed status
	var hasCompleted bool
	for _, ev := range events {
		if ev.chunkType == message.ChunkExecute {
			props := extractExecuteProps(ev)
			if strDefault(props["status"]) == "completed" {
				hasCompleted = true
			}
		}
	}
	if !hasCompleted {
		t.Error("expected completed status for empty tool result")
	}
}

func TestParse_TurnEnd_ErrorKind(t *testing.T) {
	ndjson := `{"jsonrpc":"2.0","id":1,"result":{"sessionId":"sess-1"}}
{"jsonrpc":"2.0","method":"session.event","params":{"sessionId":"sess-1","event":{"type":"turn/end","seq":1,"time":1000,"data":{"turn":1,"reason":{"kind":"api_error","message":"rate limited"}}}}}
{"jsonrpc":"2.0","method":"session.status","params":{"sessionId":"sess-1","status":"idle"}}
`
	events, _ := runParser(t, ndjson)
	var errorText string
	for _, ev := range events {
		if ev.chunkType == message.ChunkError {
			errorText = string(ev.data)
		}
	}
	if errorText != "rate limited" {
		t.Errorf("error = %q, want 'rate limited'", errorText)
	}
}

func TestParse_TurnEnd_ErrorType(t *testing.T) {
	ndjson := `{"jsonrpc":"2.0","id":1,"result":{"sessionId":"sess-1"}}
{"jsonrpc":"2.0","method":"session.event","params":{"sessionId":"sess-1","event":{"type":"turn/end","seq":1,"time":1000,"data":{"turn":1,"reason":{"kind":"error","error":"InternalError"}}}}}
{"jsonrpc":"2.0","method":"session.status","params":{"sessionId":"sess-1","status":"idle"}}
`
	events, _ := runParser(t, ndjson)
	var errorText string
	for _, ev := range events {
		if ev.chunkType == message.ChunkError {
			errorText = string(ev.data)
		}
	}
	if errorText != "InternalError" {
		t.Errorf("error = %q, want 'InternalError'", errorText)
	}
}

func TestParse_TurnEnd_FallbackError(t *testing.T) {
	ndjson := `{"jsonrpc":"2.0","id":1,"result":{"sessionId":"sess-1"}}
{"jsonrpc":"2.0","method":"session.event","params":{"sessionId":"sess-1","event":{"type":"turn/end","seq":1,"time":1000,"data":{"turn":1,"reason":{"kind":"unknown_reason"}}}}}
{"jsonrpc":"2.0","method":"session.status","params":{"sessionId":"sess-1","status":"idle"}}
`
	events, _ := runParser(t, ndjson)
	var errorText string
	for _, ev := range events {
		if ev.chunkType == message.ChunkError {
			errorText = string(ev.data)
		}
	}
	if !strings.Contains(errorText, "unknown_reason") {
		t.Errorf("error = %q, want contains 'unknown_reason'", errorText)
	}
}

func TestParse_UnknownChunkType(t *testing.T) {
	ndjson := `{"jsonrpc":"2.0","id":1,"result":{"sessionId":"sess-1"}}
{"jsonrpc":"2.0","method":"session.event","params":{"sessionId":"sess-1","event":{"type":"assistant/chunk","seq":1,"time":1000,"data":{"turn":1,"step":1,"chunk":{"type":"unknown-future-type","text":"data"}}}}}
{"jsonrpc":"2.0","method":"session.event","params":{"sessionId":"sess-1","event":{"type":"assistant/chunk","seq":2,"time":1001,"data":{"turn":1,"step":1,"chunk":{"type":"text-delta","text":"ok"}}}}}
{"jsonrpc":"2.0","method":"session.status","params":{"sessionId":"sess-1","status":"idle"}}
`
	events, completed := runParser(t, ndjson)
	if !completed {
		t.Fatal("should complete")
	}
	var text string
	for _, ev := range events {
		if ev.chunkType == message.ChunkText {
			text += string(ev.data)
		}
	}
	if text != "ok" {
		t.Errorf("text = %q, want 'ok' (unknown chunk type should be ignored)", text)
	}
}

func TestParse_UnknownEventType(t *testing.T) {
	ndjson := `{"jsonrpc":"2.0","id":1,"result":{"sessionId":"sess-1"}}
{"jsonrpc":"2.0","method":"session.event","params":{"sessionId":"sess-1","event":{"type":"future/event","seq":1,"time":1000,"data":{}}}}
{"jsonrpc":"2.0","method":"session.event","params":{"sessionId":"sess-1","event":{"type":"assistant/chunk","seq":2,"time":1001,"data":{"turn":1,"step":1,"chunk":{"type":"text-delta","text":"still works"}}}}}
{"jsonrpc":"2.0","method":"session.status","params":{"sessionId":"sess-1","status":"idle"}}
`
	events, completed := runParser(t, ndjson)
	if !completed {
		t.Fatal("should complete")
	}
	var text string
	for _, ev := range events {
		if ev.chunkType == message.ChunkText {
			text += string(ev.data)
		}
	}
	if text != "still works" {
		t.Errorf("text = %q", text)
	}
}

func TestParse_UnknownMethod(t *testing.T) {
	ndjson := `{"jsonrpc":"2.0","id":1,"result":{"sessionId":"sess-1"}}
{"jsonrpc":"2.0","method":"session.future","params":{"sessionId":"sess-1","data":"something"}}
{"jsonrpc":"2.0","method":"session.event","params":{"sessionId":"sess-1","event":{"type":"assistant/chunk","seq":1,"time":1000,"data":{"turn":1,"step":1,"chunk":{"type":"text-delta","text":"ok"}}}}}
{"jsonrpc":"2.0","method":"session.status","params":{"sessionId":"sess-1","status":"idle"}}
`
	events, completed := runParser(t, ndjson)
	if !completed {
		t.Fatal("should complete")
	}
	var text string
	for _, ev := range events {
		if ev.chunkType == message.ChunkText {
			text += string(ev.data)
		}
	}
	if text != "ok" {
		t.Errorf("text = %q", text)
	}
}

func TestParse_TextAfterReasoning(t *testing.T) {
	// text-delta arrives while thinkActive — should close thinking and start text
	ndjson := `{"jsonrpc":"2.0","id":1,"result":{"sessionId":"sess-1"}}
{"jsonrpc":"2.0","method":"session.event","params":{"sessionId":"sess-1","event":{"type":"assistant/chunk","seq":1,"time":1000,"data":{"turn":1,"step":1,"chunk":{"type":"reasoning-delta","text":"thinking"}}}}}
{"jsonrpc":"2.0","method":"session.event","params":{"sessionId":"sess-1","event":{"type":"assistant/chunk","seq":2,"time":1001,"data":{"turn":1,"step":1,"chunk":{"type":"text-delta","text":"response"}}}}}
{"jsonrpc":"2.0","method":"session.status","params":{"sessionId":"sess-1","status":"idle"}}
`
	events, completed := runParser(t, ndjson)
	if !completed {
		t.Fatal("should complete")
	}
	groups := extractMessageGroups(events)
	if len(groups) != 2 {
		t.Fatalf("expected 2 groups (thinking, text), got %d", len(groups))
	}
	if groups[0].msgType != "thinking" || groups[1].msgType != "text" {
		t.Errorf("types = [%s, %s], want [thinking, text]", groups[0].msgType, groups[1].msgType)
	}
}

func TestParse_ReasoningAfterText(t *testing.T) {
	// reasoning-delta arrives while textActive — should close text and start thinking
	ndjson := `{"jsonrpc":"2.0","id":1,"result":{"sessionId":"sess-1"}}
{"jsonrpc":"2.0","method":"session.event","params":{"sessionId":"sess-1","event":{"type":"assistant/chunk","seq":1,"time":1000,"data":{"turn":1,"step":1,"chunk":{"type":"text-delta","text":"hello"}}}}}
{"jsonrpc":"2.0","method":"session.event","params":{"sessionId":"sess-1","event":{"type":"assistant/chunk","seq":2,"time":1001,"data":{"turn":1,"step":1,"chunk":{"type":"reasoning-delta","text":"hmm"}}}}}
{"jsonrpc":"2.0","method":"session.status","params":{"sessionId":"sess-1","status":"idle"}}
`
	events, completed := runParser(t, ndjson)
	if !completed {
		t.Fatal("should complete")
	}
	groups := extractMessageGroups(events)
	if len(groups) != 2 {
		t.Fatalf("expected 2 groups, got %d", len(groups))
	}
	if groups[0].msgType != "text" || groups[1].msgType != "thinking" {
		t.Errorf("types = [%s, %s], want [text, thinking]", groups[0].msgType, groups[1].msgType)
	}
}

func TestExtractToolSummary_FallbackKey(t *testing.T) {
	// Uses the generic fallback path (not bash/write/read specific)
	got := dsh.ExportExtractToolSummary("custom_tool", `{"description":"do something"}`)
	if got != "do something" {
		t.Errorf("got %q, want 'do something'", got)
	}
}

func TestExtractToolSummary_URL(t *testing.T) {
	got := dsh.ExportExtractToolSummary("fetch", `{"url":"https://example.com"}`)
	if got != "https://example.com" {
		t.Errorf("got %q", got)
	}
}

func TestInjectDSHSemanticType_AskUser(t *testing.T) {
	props := map[string]any{}
	dsh.ExportInjectDSHSemanticType(props, "ask-user")
	if props["semantic_type"] != message.TypeQuestion {
		t.Errorf("semantic_type = %v, want TypeQuestion", props["semantic_type"])
	}
}

func TestParse_EmptyTextDelta(t *testing.T) {
	ndjson := `{"jsonrpc":"2.0","id":1,"result":{"sessionId":"sess-1"}}
{"jsonrpc":"2.0","method":"session.event","params":{"sessionId":"sess-1","event":{"type":"assistant/chunk","seq":1,"time":1000,"data":{"turn":1,"step":1,"chunk":{"type":"text-delta","text":""}}}}}
{"jsonrpc":"2.0","method":"session.event","params":{"sessionId":"sess-1","event":{"type":"assistant/chunk","seq":2,"time":1001,"data":{"turn":1,"step":1,"chunk":{"type":"text-delta","text":"real"}}}}}
{"jsonrpc":"2.0","method":"session.status","params":{"sessionId":"sess-1","status":"idle"}}
`
	events, _ := runParser(t, ndjson)
	var text string
	for _, ev := range events {
		if ev.chunkType == message.ChunkText {
			text += string(ev.data)
		}
	}
	if text != "real" {
		t.Errorf("text = %q, want 'real' (empty delta should be skipped)", text)
	}
}

func TestParse_EmptyReasoningDelta(t *testing.T) {
	ndjson := `{"jsonrpc":"2.0","id":1,"result":{"sessionId":"sess-1"}}
{"jsonrpc":"2.0","method":"session.event","params":{"sessionId":"sess-1","event":{"type":"assistant/chunk","seq":1,"time":1000,"data":{"turn":1,"step":1,"chunk":{"type":"reasoning-delta","text":""}}}}}
{"jsonrpc":"2.0","method":"session.event","params":{"sessionId":"sess-1","event":{"type":"assistant/chunk","seq":2,"time":1001,"data":{"turn":1,"step":1,"chunk":{"type":"reasoning-delta","text":"think"}}}}}
{"jsonrpc":"2.0","method":"session.status","params":{"sessionId":"sess-1","status":"idle"}}
`
	events, _ := runParser(t, ndjson)
	var text string
	for _, ev := range events {
		if ev.chunkType == message.ChunkThinking {
			text += string(ev.data)
		}
	}
	if text != "think" {
		t.Errorf("thinking = %q, want 'think' (empty delta should be skipped)", text)
	}
}

func TestParse_ToolCallDelta_NoActiveToolID(t *testing.T) {
	// tool-call-delta without prior block-start should be ignored
	ndjson := `{"jsonrpc":"2.0","id":1,"result":{"sessionId":"sess-1"}}
{"jsonrpc":"2.0","method":"session.event","params":{"sessionId":"sess-1","event":{"type":"assistant/chunk","seq":1,"time":1000,"data":{"turn":1,"step":1,"chunk":{"type":"tool-call-delta","index":0,"argumentsDelta":"{\"x\":1}"}}}}}
{"jsonrpc":"2.0","method":"session.event","params":{"sessionId":"sess-1","event":{"type":"assistant/chunk","seq":2,"time":1001,"data":{"turn":1,"step":1,"chunk":{"type":"text-delta","text":"ok"}}}}}
{"jsonrpc":"2.0","method":"session.status","params":{"sessionId":"sess-1","status":"idle"}}
`
	events, completed := runParser(t, ndjson)
	if !completed {
		t.Fatal("should complete")
	}
	var text string
	for _, ev := range events {
		if ev.chunkType == message.ChunkText {
			text += string(ev.data)
		}
	}
	if text != "ok" {
		t.Errorf("text = %q", text)
	}
}

func TestParse_BlockStartText_WhileTextActive(t *testing.T) {
	// block-start text when textActive is already true — should not nest
	ndjson := `{"jsonrpc":"2.0","id":1,"result":{"sessionId":"sess-1"}}
{"jsonrpc":"2.0","method":"session.event","params":{"sessionId":"sess-1","event":{"type":"assistant/chunk","seq":1,"time":1000,"data":{"turn":1,"step":1,"chunk":{"type":"text-delta","text":"A"}}}}}
{"jsonrpc":"2.0","method":"session.event","params":{"sessionId":"sess-1","event":{"type":"assistant/chunk","seq":2,"time":1001,"data":{"turn":1,"step":1,"chunk":{"type":"block-start","blockType":"text","index":1}}}}}
{"jsonrpc":"2.0","method":"session.event","params":{"sessionId":"sess-1","event":{"type":"assistant/chunk","seq":3,"time":1002,"data":{"turn":1,"step":1,"chunk":{"type":"text-delta","text":"B"}}}}}
{"jsonrpc":"2.0","method":"session.status","params":{"sessionId":"sess-1","status":"idle"}}
`
	events, _ := runParser(t, ndjson)
	var text string
	for _, ev := range events {
		if ev.chunkType == message.ChunkText {
			text += string(ev.data)
		}
	}
	if text != "AB" {
		t.Errorf("text = %q, want 'AB'", text)
	}
}

func TestParse_BlockStartReasoning_WhileThinkActive(t *testing.T) {
	ndjson := `{"jsonrpc":"2.0","id":1,"result":{"sessionId":"sess-1"}}
{"jsonrpc":"2.0","method":"session.event","params":{"sessionId":"sess-1","event":{"type":"assistant/chunk","seq":1,"time":1000,"data":{"turn":1,"step":1,"chunk":{"type":"reasoning-delta","text":"A"}}}}}
{"jsonrpc":"2.0","method":"session.event","params":{"sessionId":"sess-1","event":{"type":"assistant/chunk","seq":2,"time":1001,"data":{"turn":1,"step":1,"chunk":{"type":"block-start","blockType":"reasoning","index":1}}}}}
{"jsonrpc":"2.0","method":"session.event","params":{"sessionId":"sess-1","event":{"type":"assistant/chunk","seq":3,"time":1002,"data":{"turn":1,"step":1,"chunk":{"type":"reasoning-delta","text":"B"}}}}}
{"jsonrpc":"2.0","method":"session.status","params":{"sessionId":"sess-1","status":"idle"}}
`
	events, _ := runParser(t, ndjson)
	var text string
	for _, ev := range events {
		if ev.chunkType == message.ChunkThinking {
			text += string(ev.data)
		}
	}
	if text != "AB" {
		t.Errorf("thinking = %q, want 'AB'", text)
	}
}

func TestParse_StopOnBlockStartText(t *testing.T) {
	stopOnStart := func(chunkType message.StreamChunkType, data []byte) int {
		if chunkType == message.ChunkMessageStart {
			return 1
		}
		return 0
	}
	ndjson := `{"jsonrpc":"2.0","id":1,"result":{"sessionId":"sess-1"}}
{"jsonrpc":"2.0","method":"session.event","params":{"sessionId":"sess-1","event":{"type":"assistant/chunk","seq":1,"time":1000,"data":{"turn":1,"step":1,"chunk":{"type":"block-start","blockType":"text","index":0}}}}}
{"jsonrpc":"2.0","method":"session.event","params":{"sessionId":"sess-1","event":{"type":"assistant/chunk","seq":2,"time":1001,"data":{"turn":1,"step":1,"chunk":{"type":"text-delta","text":"should not see"}}}}}
{"jsonrpc":"2.0","method":"session.status","params":{"sessionId":"sess-1","status":"idle"}}
`
	p := dsh.ExportNewStreamParser(stopOnStart)
	r, w := io.Pipe()
	go func() { defer w.Close(); w.Write([]byte(ndjson)) }()
	p.Parse(context.Background(), r)
	if p.Completed() {
		t.Error("should have stopped before completion")
	}
}

func TestParse_StopOnBlockStartReasoning(t *testing.T) {
	stopOnStart := func(chunkType message.StreamChunkType, data []byte) int {
		if chunkType == message.ChunkMessageStart {
			return 1
		}
		return 0
	}
	ndjson := `{"jsonrpc":"2.0","id":1,"result":{"sessionId":"sess-1"}}
{"jsonrpc":"2.0","method":"session.event","params":{"sessionId":"sess-1","event":{"type":"assistant/chunk","seq":1,"time":1000,"data":{"turn":1,"step":1,"chunk":{"type":"block-start","blockType":"reasoning","index":0}}}}}
{"jsonrpc":"2.0","method":"session.status","params":{"sessionId":"sess-1","status":"idle"}}
`
	p := dsh.ExportNewStreamParser(stopOnStart)
	r, w := io.Pipe()
	go func() { defer w.Close(); w.Write([]byte(ndjson)) }()
	p.Parse(context.Background(), r)
	if p.Completed() {
		t.Error("should have stopped before completion")
	}
}

func TestParse_ToolCallDelta_NameEmitsUpdate(t *testing.T) {
	// Real DSH: block-start has NO name/id; first delta carries them.
	ndjson := `{"jsonrpc":"2.0","id":1,"result":{"sessionId":"sess-1"}}
{"jsonrpc":"2.0","method":"session.event","params":{"sessionId":"sess-1","event":{"type":"assistant/chunk","seq":1,"time":1000,"data":{"turn":1,"step":1,"chunk":{"type":"block-start","blockType":"tool-call","index":0}}}}}
{"jsonrpc":"2.0","method":"session.event","params":{"sessionId":"sess-1","event":{"type":"assistant/chunk","seq":2,"time":1001,"data":{"turn":1,"step":1,"chunk":{"type":"tool-call-delta","index":0,"id":"call_real","name":"write","argumentsDelta":"{\"file_path\":\"/tmp/hello.md\","}}}}}
{"jsonrpc":"2.0","method":"session.event","params":{"sessionId":"sess-1","event":{"type":"assistant/chunk","seq":3,"time":1002,"data":{"turn":1,"step":1,"chunk":{"type":"tool-call-delta","index":0,"argumentsDelta":"\"content\":\"hello\"}"}}}}}
{"jsonrpc":"2.0","method":"session.event","params":{"sessionId":"sess-1","event":{"type":"assistant/chunk","seq":4,"time":1003,"data":{"turn":1,"step":1,"chunk":{"type":"block-end","index":0}}}}}
{"jsonrpc":"2.0","method":"session.event","params":{"sessionId":"sess-1","event":{"type":"tool/call","seq":5,"time":1004,"data":{"turn":1,"step":1,"callId":"call_real","name":"write","arguments":"{\"file_path\":\"/tmp/hello.md\",\"content\":\"hello\"}"}}}}
{"jsonrpc":"2.0","method":"session.event","params":{"sessionId":"sess-1","event":{"type":"tool/result","seq":6,"time":1005,"data":{"turn":1,"step":1,"message":{"source":{"callId":"call_real"},"content":[{"content":[{"type":"text","text":"ok"}]}]}}}}}
{"jsonrpc":"2.0","method":"session.status","params":{"sessionId":"sess-1","status":"idle"}}
`
	events, completed := runParser(t, ndjson)
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

	// 4 groups: block-start(no name) + delta(name+summary) + tool/call + tool/result
	// (block-end is skipped because summary didn't change from delta phase)
	if len(executeGroups) != 4 {
		t.Fatalf("expected 4 execute groups, got %d", len(executeGroups))
	}

	// All share same messageID
	msgID := executeGroups[0].messageID
	for i, g := range executeGroups {
		if g.messageID != msgID {
			t.Errorf("group %d msgID %q != first %q", i, g.messageID, msgID)
		}
	}

	// Group 0 (block-start): tool is empty (DSH doesn't send name in block-start)
	for _, ev := range executeGroups[0].events {
		if ev.chunkType == message.ChunkExecute {
			props := extractExecuteProps(ev)
			if s := strDefault(props["tool"]); s != "" {
				t.Errorf("block-start group tool = %q, want empty", s)
			}
		}
	}

	// Group 1 (first delta): tool name + summary both available via incremental extraction
	for _, ev := range executeGroups[1].events {
		if ev.chunkType == message.ChunkExecute {
			props := extractExecuteProps(ev)
			if s := strDefault(props["tool"]); s != "write" {
				t.Errorf("delta group tool = %q, want write", s)
			}
			if s := strDefault(props["summary"]); !strings.Contains(s, "/tmp/hello.md") {
				t.Errorf("delta group summary = %q, want contains '/tmp/hello.md'", s)
			}
		}
	}

	// Group 3 (tool/result): completed
	for _, ev := range executeGroups[3].events {
		if ev.chunkType == message.ChunkExecute {
			props := extractExecuteProps(ev)
			if s := strDefault(props["status"]); s != "completed" {
				t.Errorf("last group status = %q, want completed", s)
			}
		}
	}
}

func TestParse_StopOnToolCallPendingEmit(t *testing.T) {
	stopOnExecute := func(chunkType message.StreamChunkType, data []byte) int {
		if chunkType == message.ChunkExecute {
			return 1
		}
		return 0
	}
	ndjson := `{"jsonrpc":"2.0","id":1,"result":{"sessionId":"sess-1"}}
{"jsonrpc":"2.0","method":"session.event","params":{"sessionId":"sess-1","event":{"type":"assistant/chunk","seq":1,"time":1000,"data":{"turn":1,"step":1,"chunk":{"type":"block-start","blockType":"tool-call","index":0,"id":"stop_call","name":"bash"}}}}}
{"jsonrpc":"2.0","method":"session.status","params":{"sessionId":"sess-1","status":"idle"}}
`
	p := dsh.ExportNewStreamParser(stopOnExecute)
	r, w := io.Pipe()
	go func() { defer w.Close(); w.Write([]byte(ndjson)) }()
	p.Parse(context.Background(), r)
	if p.Completed() {
		t.Error("should have stopped on pending execute emit")
	}
}

func TestParse_NonNotificationRequest(t *testing.T) {
	// A request message (with id but no result/error) — should be silently ignored
	ndjson := `{"jsonrpc":"2.0","id":1,"result":{"sessionId":"sess-1"}}
{"jsonrpc":"2.0","id":99,"method":"session.prompt","params":{"sessionId":"sess-1"}}
{"jsonrpc":"2.0","method":"session.event","params":{"sessionId":"sess-1","event":{"type":"assistant/chunk","seq":1,"time":1000,"data":{"turn":1,"step":1,"chunk":{"type":"text-delta","text":"ok"}}}}}
{"jsonrpc":"2.0","method":"session.status","params":{"sessionId":"sess-1","status":"idle"}}
`
	events, completed := runParser(t, ndjson)
	if !completed {
		t.Fatal("should complete")
	}
	var text string
	for _, ev := range events {
		if ev.chunkType == message.ChunkText {
			text += string(ev.data)
		}
	}
	if text != "ok" {
		t.Errorf("text = %q", text)
	}
}

func TestParse_FinishEmptyReason(t *testing.T) {
	ndjson := `{"jsonrpc":"2.0","id":1,"result":{"sessionId":"sess-1"}}
{"jsonrpc":"2.0","method":"session.event","params":{"sessionId":"sess-1","event":{"type":"assistant/chunk","seq":1,"time":1000,"data":{"turn":1,"step":1,"chunk":{"type":"finish"}}}}}
{"jsonrpc":"2.0","method":"session.status","params":{"sessionId":"sess-1","status":"idle"}}
`
	events, completed := runParser(t, ndjson)
	if !completed {
		t.Fatal("should complete")
	}
	for _, ev := range events {
		if ev.chunkType == message.ChunkMetadata {
			var m map[string]any
			json.Unmarshal(ev.data, &m)
			if _, ok := m["finish_reason"]; ok {
				t.Error("empty finish reason should not emit metadata")
			}
		}
	}
}

func TestParse_UsageEmpty(t *testing.T) {
	ndjson := `{"jsonrpc":"2.0","id":1,"result":{"sessionId":"sess-1"}}
{"jsonrpc":"2.0","method":"session.event","params":{"sessionId":"sess-1","event":{"type":"assistant/chunk","seq":1,"time":1000,"data":{"turn":1,"step":1,"chunk":{"type":"usage"}}}}}
{"jsonrpc":"2.0","method":"session.status","params":{"sessionId":"sess-1","status":"idle"}}
`
	events, completed := runParser(t, ndjson)
	if !completed {
		t.Fatal("should complete")
	}
	for _, ev := range events {
		if ev.chunkType == message.ChunkMetadata {
			t.Error("empty usage should not emit metadata")
		}
	}
}

func TestParse_SessionStatus_NonIdle(t *testing.T) {
	ndjson := `{"jsonrpc":"2.0","id":1,"result":{"sessionId":"sess-1"}}
{"jsonrpc":"2.0","method":"session.status","params":{"sessionId":"sess-1","status":"running"}}
{"jsonrpc":"2.0","method":"session.event","params":{"sessionId":"sess-1","event":{"type":"assistant/chunk","seq":1,"time":1000,"data":{"turn":1,"step":1,"chunk":{"type":"text-delta","text":"hello"}}}}}
{"jsonrpc":"2.0","method":"session.status","params":{"sessionId":"sess-1","status":"idle"}}
`
	events, completed := runParser(t, ndjson)
	if !completed {
		t.Fatal("should complete")
	}
	var text string
	for _, ev := range events {
		if ev.chunkType == message.ChunkText {
			text += string(ev.data)
		}
	}
	if text != "hello" {
		t.Errorf("non-idle status should not interfere, text = %q", text)
	}
}

func TestParse_EmptyStream(t *testing.T) {
	events, completed := runParser(t, "")
	if completed {
		t.Error("empty stream should not be completed")
	}
	if len(events) != 0 {
		t.Errorf("expected 0 events, got %d", len(events))
	}
}

func TestParse_MalformedJSON(t *testing.T) {
	ndjson := `not valid json
{"jsonrpc":"2.0","id":1,"result":{"sessionId":"sess-1"}}
also not valid {{{
{"jsonrpc":"2.0","method":"session.event","params":{"sessionId":"sess-1","event":{"type":"assistant/chunk","seq":1,"time":1000,"data":{"turn":1,"step":1,"chunk":{"type":"text-delta","text":"survived"}}}}}
{"jsonrpc":"2.0","method":"session.status","params":{"sessionId":"sess-1","status":"idle"}}
`
	events, completed := runParser(t, ndjson)
	if !completed {
		t.Fatal("should complete despite malformed lines")
	}
	var text string
	for _, ev := range events {
		if ev.chunkType == message.ChunkText {
			text += string(ev.data)
		}
	}
	if text != "survived" {
		t.Errorf("text = %q, want 'survived'", text)
	}
}

func TestParse_StopSignal(t *testing.T) {
	// handler returns non-zero to stop
	stopAt := 0
	var events []recordedEvent
	handler := func(chunkType message.StreamChunkType, data []byte) int {
		cp := make([]byte, len(data))
		copy(cp, data)
		events = append(events, recordedEvent{chunkType: chunkType, data: cp})
		if chunkType == message.ChunkText {
			stopAt++
			if stopAt >= 2 {
				return 1
			}
		}
		return 0
	}

	ndjson := `{"jsonrpc":"2.0","id":1,"result":{"sessionId":"sess-1"}}
{"jsonrpc":"2.0","method":"session.event","params":{"sessionId":"sess-1","event":{"type":"assistant/chunk","seq":1,"time":1000,"data":{"turn":1,"step":1,"chunk":{"type":"text-delta","text":"one"}}}}}
{"jsonrpc":"2.0","method":"session.event","params":{"sessionId":"sess-1","event":{"type":"assistant/chunk","seq":2,"time":1001,"data":{"turn":1,"step":1,"chunk":{"type":"text-delta","text":"two"}}}}}
{"jsonrpc":"2.0","method":"session.event","params":{"sessionId":"sess-1","event":{"type":"assistant/chunk","seq":3,"time":1002,"data":{"turn":1,"step":1,"chunk":{"type":"text-delta","text":"three"}}}}}
{"jsonrpc":"2.0","method":"session.status","params":{"sessionId":"sess-1","status":"idle"}}
`
	p := dsh.ExportNewStreamParser(handler)
	r, w := io.Pipe()
	go func() {
		defer w.Close()
		w.Write([]byte(ndjson))
	}()
	p.Parse(context.Background(), r)

	// Should have stopped after "two" (the second text chunk)
	var texts []string
	for _, ev := range events {
		if ev.chunkType == message.ChunkText {
			texts = append(texts, string(ev.data))
		}
	}
	if len(texts) != 2 || texts[0] != "one" || texts[1] != "two" {
		t.Errorf("texts = %v, want [one two]", texts)
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

// --- Orphaned tool cleanup tests ---

func TestParse_OrphanedToolCleanup_Idle(t *testing.T) {
	// Tool enters Phase 1 but session goes idle before tool/result.
	ndjson := `{"jsonrpc":"2.0","id":1,"result":{"sessionId":"sess-1"}}
{"jsonrpc":"2.0","method":"session.event","params":{"sessionId":"sess-1","event":{"type":"assistant/chunk","seq":1,"time":1000,"data":{"turn":1,"step":1,"chunk":{"type":"block-start","blockType":"tool-call","index":0,"id":"orphan_1","name":"bash"}}}}}
{"jsonrpc":"2.0","method":"session.event","params":{"sessionId":"sess-1","event":{"type":"assistant/chunk","seq":2,"time":1001,"data":{"turn":1,"step":1,"chunk":{"type":"tool-call-delta","index":0,"argumentsDelta":"{\"command\":\"sleep 99\"}"}}}}}
{"jsonrpc":"2.0","method":"session.event","params":{"sessionId":"sess-1","event":{"type":"assistant/chunk","seq":3,"time":1002,"data":{"turn":1,"step":1,"chunk":{"type":"block-end","index":0}}}}}
{"jsonrpc":"2.0","method":"session.status","params":{"sessionId":"sess-1","status":"idle"}}
`
	events, completed := runParser(t, ndjson)
	if !completed {
		t.Fatal("should complete on idle")
	}

	groups := extractMessageGroups(events)
	var executeGroups []messageGroup
	for _, g := range groups {
		if g.msgType == "execute" {
			executeGroups = append(executeGroups, g)
		}
	}

	if len(executeGroups) < 3 {
		t.Fatalf("expected at least 3 execute groups, got %d", len(executeGroups))
	}

	// Last execute group should be the cleanup error
	last := executeGroups[len(executeGroups)-1]
	for _, ev := range last.events {
		if ev.chunkType == message.ChunkExecute {
			props := extractExecuteProps(ev)
			if strDefault(props["status"]) != "error" {
				t.Errorf("cleanup status = %q, want error", strDefault(props["status"]))
			}
			if strDefault(props["tool"]) != "bash" {
				t.Errorf("cleanup tool = %q, want bash", strDefault(props["tool"]))
			}
		}
	}

	// All groups for the orphan should share the same messageID
	msgID := executeGroups[0].messageID
	for i, g := range executeGroups {
		if g.messageID != msgID {
			t.Errorf("group %d msgID %q != first %q", i, g.messageID, msgID)
		}
	}

	// Message pairing
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
		t.Errorf("starts (%d) != ends (%d)", starts, ends)
	}
}

func TestParse_OrphanedToolCleanup_EOF(t *testing.T) {
	// Tool enters Phase 1 but stream ends (EOF) before idle or tool/result.
	ndjson := `{"jsonrpc":"2.0","id":1,"result":{"sessionId":"sess-1"}}
{"jsonrpc":"2.0","method":"session.event","params":{"sessionId":"sess-1","event":{"type":"assistant/chunk","seq":1,"time":1000,"data":{"turn":1,"step":1,"chunk":{"type":"block-start","blockType":"tool-call","index":0,"id":"eof_tool","name":"read"}}}}}
{"jsonrpc":"2.0","method":"session.event","params":{"sessionId":"sess-1","event":{"type":"assistant/chunk","seq":2,"time":1001,"data":{"turn":1,"step":1,"chunk":{"type":"tool-call-delta","index":0,"argumentsDelta":"{\"path\":\"/tmp/x\"}"}}}}}
{"jsonrpc":"2.0","method":"session.event","params":{"sessionId":"sess-1","event":{"type":"assistant/chunk","seq":3,"time":1002,"data":{"turn":1,"step":1,"chunk":{"type":"block-end","index":0}}}}}
`
	events, completed := runParser(t, ndjson)
	if completed {
		t.Fatal("should NOT complete without idle")
	}

	var hasError bool
	for _, ev := range events {
		if ev.chunkType == message.ChunkExecute {
			props := extractExecuteProps(ev)
			if strDefault(props["status"]) == "error" {
				hasError = true
			}
		}
	}
	if !hasError {
		t.Error("expected error status for orphaned tool on EOF")
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
		t.Errorf("starts (%d) != ends (%d)", starts, ends)
	}
}

func TestParse_OrphanedToolWithAlias(t *testing.T) {
	// block-start with synthetic ID, delta maps real callId, then tool/result
	// completes — cleanup should NOT emit extra error for the synthetic alias.
	ndjson := `{"jsonrpc":"2.0","id":1,"result":{"sessionId":"sess-1"}}
{"jsonrpc":"2.0","method":"session.event","params":{"sessionId":"sess-1","event":{"type":"assistant/chunk","seq":1,"time":1000,"data":{"turn":1,"step":1,"chunk":{"type":"block-start","blockType":"tool-call","index":0,"id":"synth_0","name":"bash"}}}}}
{"jsonrpc":"2.0","method":"session.event","params":{"sessionId":"sess-1","event":{"type":"assistant/chunk","seq":2,"time":1001,"data":{"turn":1,"step":1,"chunk":{"type":"tool-call-delta","index":0,"id":"real_call","argumentsDelta":"{\"command\":\"ls\"}"}}}}}
{"jsonrpc":"2.0","method":"session.event","params":{"sessionId":"sess-1","event":{"type":"assistant/chunk","seq":3,"time":1002,"data":{"turn":1,"step":1,"chunk":{"type":"block-end","index":0}}}}}
{"jsonrpc":"2.0","method":"session.event","params":{"sessionId":"sess-1","event":{"type":"tool/call","seq":4,"time":1003,"data":{"turn":1,"step":1,"callId":"real_call","name":"bash","arguments":"{\"command\":\"ls\"}"}}}}
{"jsonrpc":"2.0","method":"session.event","params":{"sessionId":"sess-1","event":{"type":"tool/result","seq":5,"time":1004,"data":{"turn":1,"step":1,"message":{"source":{"callId":"real_call"},"content":[{"content":[{"type":"text","text":"a.txt\n"}]}]}}}}}
{"jsonrpc":"2.0","method":"session.status","params":{"sessionId":"sess-1","status":"idle"}}
`
	events, completed := runParser(t, ndjson)
	if !completed {
		t.Fatal("should complete")
	}

	var errorCount int
	for _, ev := range events {
		if ev.chunkType == message.ChunkExecute {
			props := extractExecuteProps(ev)
			if strDefault(props["status"]) == "error" {
				errorCount++
			}
		}
	}
	if errorCount != 0 {
		t.Errorf("expected 0 error groups (alias should not trigger cleanup), got %d", errorCount)
	}
}

func TestParse_MixedOrphanAndCompleted(t *testing.T) {
	// Two parallel tools: call_ok completes, call_orphan is orphaned.
	ndjson := `{"jsonrpc":"2.0","id":1,"result":{"sessionId":"sess-1"}}
{"jsonrpc":"2.0","method":"session.event","params":{"sessionId":"sess-1","event":{"type":"assistant/chunk","seq":1,"time":1000,"data":{"turn":1,"step":1,"chunk":{"type":"block-start","blockType":"tool-call","index":0,"id":"call_ok","name":"bash"}}}}}
{"jsonrpc":"2.0","method":"session.event","params":{"sessionId":"sess-1","event":{"type":"assistant/chunk","seq":2,"time":1001,"data":{"turn":1,"step":1,"chunk":{"type":"tool-call-delta","index":0,"argumentsDelta":"{\"command\":\"ls\"}"}}}}}
{"jsonrpc":"2.0","method":"session.event","params":{"sessionId":"sess-1","event":{"type":"assistant/chunk","seq":3,"time":1002,"data":{"turn":1,"step":1,"chunk":{"type":"block-end","index":0}}}}}
{"jsonrpc":"2.0","method":"session.event","params":{"sessionId":"sess-1","event":{"type":"assistant/chunk","seq":4,"time":1003,"data":{"turn":1,"step":1,"chunk":{"type":"block-start","blockType":"tool-call","index":1,"id":"call_orphan","name":"write"}}}}}
{"jsonrpc":"2.0","method":"session.event","params":{"sessionId":"sess-1","event":{"type":"assistant/chunk","seq":5,"time":1004,"data":{"turn":1,"step":1,"chunk":{"type":"tool-call-delta","index":1,"argumentsDelta":"{\"file_path\":\"/tmp/x.txt\"}"}}}}}
{"jsonrpc":"2.0","method":"session.event","params":{"sessionId":"sess-1","event":{"type":"assistant/chunk","seq":6,"time":1005,"data":{"turn":1,"step":1,"chunk":{"type":"block-end","index":1}}}}}
{"jsonrpc":"2.0","method":"session.event","params":{"sessionId":"sess-1","event":{"type":"tool/call","seq":7,"time":1006,"data":{"turn":1,"step":1,"callId":"call_ok","name":"bash","arguments":"{\"command\":\"ls\"}"}}}}
{"jsonrpc":"2.0","method":"session.event","params":{"sessionId":"sess-1","event":{"type":"tool/result","seq":8,"time":1007,"data":{"turn":1,"step":1,"message":{"source":{"callId":"call_ok"},"content":[{"content":[{"type":"text","text":"files\n"}]}]}}}}}
{"jsonrpc":"2.0","method":"session.status","params":{"sessionId":"sess-1","status":"idle"}}
`
	events, completed := runParser(t, ndjson)
	if !completed {
		t.Fatal("should complete")
	}

	var completedCount, errorCount int
	for _, ev := range events {
		if ev.chunkType == message.ChunkExecute {
			props := extractExecuteProps(ev)
			switch strDefault(props["status"]) {
			case "completed":
				completedCount++
			case "error":
				errorCount++
				if strDefault(props["tool"]) != "write" {
					t.Errorf("error tool = %q, want write", strDefault(props["tool"]))
				}
			}
		}
	}
	if completedCount != 1 {
		t.Errorf("completed count = %d, want 1", completedCount)
	}
	if errorCount != 1 {
		t.Errorf("error count = %d, want 1", errorCount)
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
		t.Errorf("starts (%d) != ends (%d)", starts, ends)
	}
}

// --- Skill tool summary tests ---

func TestExtractToolSummary_Skill(t *testing.T) {
	got := dsh.ExportExtractToolSummary("skill", `{"name":"my-skill","description":"does things"}`)
	if got != "my-skill" {
		t.Errorf("got %q, want 'my-skill'", got)
	}
}

func TestExtractToolSummary_SkillDescription(t *testing.T) {
	got := dsh.ExportExtractToolSummary("skill", `{"description":"does things"}`)
	if got != "does things" {
		t.Errorf("got %q, want 'does things'", got)
	}
}

func TestExtractToolSummary_FallbackName(t *testing.T) {
	got := dsh.ExportExtractToolSummary("unknown_tool", `{"name":"some-name"}`)
	if got != "some-name" {
		t.Errorf("got %q, want 'some-name'", got)
	}
}

// --- Partial summary extraction tests ---

func TestExtractToolSummaryPartial_CompleteJSON(t *testing.T) {
	got := dsh.ExportExtractToolSummaryPartial("bash", `{"command":"echo hello"}`)
	if got != "echo hello" {
		t.Errorf("got %q, want 'echo hello'", got)
	}
}

func TestExtractToolSummaryPartial_TruncatedJSON(t *testing.T) {
	got := dsh.ExportExtractToolSummaryPartial("write", `{"file_path":"/tmp/hello.md","content":"hel`)
	if got != "/tmp/hello.md" {
		t.Errorf("got %q, want '/tmp/hello.md'", got)
	}
}

func TestExtractToolSummaryPartial_VeryShortFragment(t *testing.T) {
	got := dsh.ExportExtractToolSummaryPartial("bash", `{"com`)
	if got != "" {
		t.Errorf("got %q, want empty (too short)", got)
	}
}

func TestExtractToolSummaryPartial_Empty(t *testing.T) {
	got := dsh.ExportExtractToolSummaryPartial("bash", "")
	if got != "" {
		t.Errorf("got %q, want empty", got)
	}
}

func TestExtractToolSummaryPartial_RegexFallback(t *testing.T) {
	got := dsh.ExportExtractToolSummaryPartial("read", `"path":"/workspace/go.mod"`)
	if got != "/workspace/go.mod" {
		t.Errorf("got %q, want '/workspace/go.mod'", got)
	}
}

func TestExtractToolSummaryPartial_Skill(t *testing.T) {
	got := dsh.ExportExtractToolSummaryPartial("skill", `{"name":"my-skill","desc`)
	if got != "my-skill" {
		t.Errorf("got %q, want 'my-skill'", got)
	}
}

// --- Incremental summary in onToolCallDelta integration test ---

func TestParse_IncrementalSummaryDuringDelta(t *testing.T) {
	ndjson := `{"jsonrpc":"2.0","id":1,"result":{"sessionId":"sess-1"}}
{"jsonrpc":"2.0","method":"session.event","params":{"sessionId":"sess-1","event":{"type":"assistant/chunk","seq":1,"time":1000,"data":{"turn":1,"step":1,"chunk":{"type":"block-start","blockType":"tool-call","index":0,"id":"incr_1","name":"write"}}}}}
{"jsonrpc":"2.0","method":"session.event","params":{"sessionId":"sess-1","event":{"type":"assistant/chunk","seq":2,"time":1001,"data":{"turn":1,"step":1,"chunk":{"type":"tool-call-delta","index":0,"argumentsDelta":"{\"file_path\":\"/tmp/test.go\","}}}}}
{"jsonrpc":"2.0","method":"session.event","params":{"sessionId":"sess-1","event":{"type":"assistant/chunk","seq":3,"time":1002,"data":{"turn":1,"step":1,"chunk":{"type":"tool-call-delta","index":0,"argumentsDelta":"\"content\":\"package main\"}"}}}}}
{"jsonrpc":"2.0","method":"session.event","params":{"sessionId":"sess-1","event":{"type":"assistant/chunk","seq":4,"time":1003,"data":{"turn":1,"step":1,"chunk":{"type":"block-end","index":0}}}}}
{"jsonrpc":"2.0","method":"session.event","params":{"sessionId":"sess-1","event":{"type":"tool/call","seq":5,"time":1004,"data":{"turn":1,"step":1,"callId":"incr_1","name":"write","arguments":"{\"file_path\":\"/tmp/test.go\",\"content\":\"package main\"}"}}}}
{"jsonrpc":"2.0","method":"session.event","params":{"sessionId":"sess-1","event":{"type":"tool/result","seq":6,"time":1005,"data":{"turn":1,"step":1,"message":{"source":{"callId":"incr_1"},"content":[{"content":[{"type":"text","text":"ok"}]}]}}}}}
{"jsonrpc":"2.0","method":"session.status","params":{"sessionId":"sess-1","status":"idle"}}
`
	events, completed := runParser(t, ndjson)
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

	// Find the first group that carries a summary with the file path
	var firstSummaryIdx int = -1
	for i, g := range executeGroups {
		for _, ev := range g.events {
			if ev.chunkType == message.ChunkExecute {
				props := extractExecuteProps(ev)
				if s := strDefault(props["summary"]); strings.Contains(s, "/tmp/test.go") {
					firstSummaryIdx = i
					break
				}
			}
		}
		if firstSummaryIdx >= 0 {
			break
		}
	}

	if firstSummaryIdx < 0 {
		t.Fatal("no group carried summary containing /tmp/test.go")
	}
	// Summary should appear during delta phase (group 0=block-start, 1=delta summary),
	// not waiting until block-end or tool/call.
	if firstSummaryIdx > 2 {
		t.Errorf("summary first appeared at group %d, expected <= 2 (delta/block-end phase)", firstSummaryIdx)
	}
}
