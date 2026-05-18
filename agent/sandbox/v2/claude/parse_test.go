package claude_test

import (
	"context"
	"encoding/json"
	"io"
	"strings"
	"testing"

	"github.com/yaoapp/yao/agent/output/message"
	"github.com/yaoapp/yao/agent/sandbox/v2/claude"
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

func runParser(t *testing.T, jsonl string) []recordedEvent {
	t.Helper()
	var events []recordedEvent
	p := claude.ExportNewStreamParser(mockStreamFunc(&events))
	r, w := io.Pipe()
	go func() {
		defer w.Close()
		w.Write([]byte(jsonl))
	}()
	if err := p.Parse(context.Background(), r); err != nil {
		t.Fatalf("parse error: %v", err)
	}
	return events
}

type messageGroup struct {
	messageID string
	events    []recordedEvent
}

func extractMessageGroups(events []recordedEvent) []messageGroup {
	var groups []messageGroup
	var cur *messageGroup
	for _, ev := range events {
		if ev.chunkType == message.ChunkMessageStart {
			var sd message.EventMessageStartData
			json.Unmarshal(ev.data, &sd)
			cur = &messageGroup{messageID: sd.MessageID}
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

const parallelToolCallJSONL = `{"type":"stream_event","event":{"type":"message_start","message":{"id":"msg_mock","type":"message","role":"assistant","content":[],"model":"claude-sonnet-4-20250514","stop_reason":null,"usage":{"input_tokens":100,"output_tokens":1}}}}
{"type":"stream_event","event":{"type":"content_block_start","index":0,"content_block":{"type":"tool_use","id":"toolu_pip","name":"Bash"}}}
{"type":"stream_event","event":{"type":"content_block_delta","index":0,"delta":{"type":"input_json_delta","partial_json":"{\"command\""}}}
{"type":"stream_event","event":{"type":"content_block_delta","index":0,"delta":{"type":"input_json_delta","partial_json":":\"pip --version\"}"}}}
{"type":"assistant","message":{"id":"msg_mock","type":"message","role":"assistant","content":[{"type":"tool_use","id":"toolu_pip","name":"Bash","input":{"command":"pip --version"}}],"model":"claude-sonnet-4-20250514","stop_reason":null,"usage":{"input_tokens":100,"output_tokens":30}}}
{"type":"stream_event","event":{"type":"content_block_stop","index":0}}
{"type":"stream_event","event":{"type":"content_block_start","index":1,"content_block":{"type":"tool_use","id":"toolu_npm","name":"Bash"}}}
{"type":"stream_event","event":{"type":"content_block_delta","index":1,"delta":{"type":"input_json_delta","partial_json":"{\"command\""}}}
{"type":"user","message":{"id":"msg_user_pip","type":"message","role":"user","content":[{"type":"tool_result","tool_use_id":"toolu_pip","content":"pip 24.0 (python 3.12)"}]}}
{"type":"stream_event","event":{"type":"content_block_delta","index":1,"delta":{"type":"input_json_delta","partial_json":":\"npm --version\"}"}}}
{"type":"assistant","message":{"id":"msg_mock2","type":"message","role":"assistant","content":[{"type":"tool_use","id":"toolu_npm","name":"Bash","input":{"command":"npm --version"}}],"model":"claude-sonnet-4-20250514","stop_reason":null,"usage":{"input_tokens":200,"output_tokens":60}}}
{"type":"stream_event","event":{"type":"content_block_stop","index":1}}
{"type":"stream_event","event":{"type":"message_delta","delta":{"stop_reason":"end_turn"},"usage":{"output_tokens":60}}}
{"type":"stream_event","event":{"type":"message_stop"}}
{"type":"user","message":{"id":"msg_user_npm","type":"message","role":"user","content":[{"type":"tool_result","tool_use_id":"toolu_npm","content":"10.9.7"}]}}
{"type":"stream_event","event":{"type":"message_start","message":{"id":"msg_final","type":"message","role":"assistant","content":[],"model":"claude-sonnet-4-20250514","stop_reason":null,"usage":{"input_tokens":300,"output_tokens":1}}}}
{"type":"stream_event","event":{"type":"content_block_start","index":0,"content_block":{"type":"text","text":""}}}
{"type":"stream_event","event":{"type":"content_block_delta","index":0,"delta":{"type":"text_delta","text":"pip 和 npm 都可以正常使用。"}}}
{"type":"assistant","message":{"id":"msg_final","type":"message","role":"assistant","content":[{"type":"text","text":"pip 和 npm 都可以正常使用。"}],"model":"claude-sonnet-4-20250514","stop_reason":"end_turn","usage":{"input_tokens":300,"output_tokens":20}}}
{"type":"stream_event","event":{"type":"content_block_stop","index":0}}
{"type":"stream_event","event":{"type":"message_delta","delta":{"stop_reason":"end_turn"},"usage":{"output_tokens":20}}}
{"type":"stream_event","event":{"type":"message_stop"}}
{"type":"result","result":"pip 和 npm 都可以正常使用。","is_error":false}
`

const singleToolCallJSONL = `{"type":"stream_event","event":{"type":"message_start","message":{"id":"msg_s","type":"message","role":"assistant","content":[],"model":"claude-sonnet-4-20250514","stop_reason":null,"usage":{"input_tokens":50,"output_tokens":1}}}}
{"type":"stream_event","event":{"type":"content_block_start","index":0,"content_block":{"type":"tool_use","id":"toolu_single","name":"Bash"}}}
{"type":"stream_event","event":{"type":"content_block_delta","index":0,"delta":{"type":"input_json_delta","partial_json":"{\"command\":\"ls\"}"}}}
{"type":"assistant","message":{"id":"msg_s","type":"message","role":"assistant","content":[{"type":"tool_use","id":"toolu_single","name":"Bash","input":{"command":"ls"}}],"model":"claude-sonnet-4-20250514","stop_reason":null,"usage":{"input_tokens":50,"output_tokens":10}}}
{"type":"stream_event","event":{"type":"content_block_stop","index":0}}
{"type":"stream_event","event":{"type":"message_delta","delta":{"stop_reason":"end_turn"},"usage":{"output_tokens":10}}}
{"type":"stream_event","event":{"type":"message_stop"}}
{"type":"user","message":{"id":"msg_u","type":"message","role":"user","content":[{"type":"tool_result","tool_use_id":"toolu_single","content":"file1.txt\nfile2.txt"}]}}
{"type":"stream_event","event":{"type":"message_start","message":{"id":"msg_s2","type":"message","role":"assistant","content":[],"model":"claude-sonnet-4-20250514","stop_reason":null,"usage":{"input_tokens":80,"output_tokens":1}}}}
{"type":"stream_event","event":{"type":"content_block_start","index":0,"content_block":{"type":"text","text":""}}}
{"type":"stream_event","event":{"type":"content_block_delta","index":0,"delta":{"type":"text_delta","text":"Done."}}}
{"type":"stream_event","event":{"type":"content_block_stop","index":0}}
{"type":"stream_event","event":{"type":"message_delta","delta":{"stop_reason":"end_turn"},"usage":{"output_tokens":5}}}
{"type":"stream_event","event":{"type":"message_stop"}}
{"type":"result","result":"Done.","is_error":false}
`

func TestParseParallelToolCalls_MessagePairing(t *testing.T) {
	events := runParser(t, parallelToolCallJSONL)

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
	if startCount < 5 {
		t.Errorf("expected at least 5 groups, got %d", startCount)
	}
}

func TestParseParallelToolCalls_ToolLifecycle(t *testing.T) {
	events := runParser(t, parallelToolCallJSONL)
	groups := extractMessageGroups(events)

	type toolPhase struct {
		messageID, status, tool string
		output                  any
	}
	var phases []toolPhase
	for _, g := range groups {
		for _, ev := range g.events {
			if ev.chunkType == message.ChunkExecute {
				props := extractExecuteProps(ev)
				if status, _ := props["status"].(string); status != "" {
					phases = append(phases, toolPhase{g.messageID, status, strDefault(props["tool"]), props["output"]})
				}
			}
		}
	}

	var pipRunning, pipCompleted, npmRunning, npmCompleted *toolPhase
	for i := range phases {
		p := &phases[i]
		if p.tool == "Bash" && p.status == "running" && pipRunning == nil {
			pipRunning = p
		} else if p.tool == "Bash" && p.status == "running" && npmRunning == nil {
			npmRunning = p
		}
		out := strDefault(p.output)
		if p.status == "completed" && strings.Contains(out, "pip") {
			pipCompleted = p
		}
		if p.status == "completed" && strings.Contains(out, "10.9.7") {
			npmCompleted = p
		}
	}

	if pipRunning == nil || pipCompleted == nil || npmRunning == nil || npmCompleted == nil {
		t.Fatal("missing tool phases")
	}
	if pipRunning.messageID != pipCompleted.messageID {
		t.Errorf("pip running/completed msgID mismatch")
	}
	if npmRunning.messageID != npmCompleted.messageID {
		t.Errorf("npm running/completed msgID mismatch")
	}
	if pipRunning.messageID == npmRunning.messageID {
		t.Error("pip and npm should have different msgIDs")
	}
}

func TestParseParallelToolCalls_FinalText(t *testing.T) {
	events := runParser(t, parallelToolCallJSONL)
	var textChunks []string
	inTextGroup := false
	for _, ev := range events {
		if ev.chunkType == message.ChunkMessageStart {
			var sd message.EventMessageStartData
			json.Unmarshal(ev.data, &sd)
			inTextGroup = sd.Type == "text"
		}
		if ev.chunkType == message.ChunkText && inTextGroup {
			textChunks = append(textChunks, string(ev.data))
		}
		if ev.chunkType == message.ChunkMessageEnd {
			inTextGroup = false
		}
	}
	fullText := strings.Join(textChunks, "")
	if !strings.Contains(fullText, "pip") || !strings.Contains(fullText, "npm") {
		t.Errorf("final text = %q", fullText)
	}
}

func TestParseSingleToolCall(t *testing.T) {
	events := runParser(t, singleToolCallJSONL)
	groups := extractMessageGroups(events)
	if len(groups) < 3 {
		t.Fatalf("expected at least 3 groups, got %d", len(groups))
	}

	depth := 0
	for _, ev := range events {
		switch ev.chunkType {
		case message.ChunkMessageStart:
			depth++
			if depth > 1 {
				t.Fatal("nested message_start")
			}
		case message.ChunkMessageEnd:
			depth--
			if depth < 0 {
				t.Fatal("unmatched message_end")
			}
		}
	}
	if depth != 0 {
		t.Fatalf("unbalanced start/end (depth=%d)", depth)
	}

	var runningID, completedID string
	for _, g := range groups {
		for _, ev := range g.events {
			if ev.chunkType == message.ChunkExecute {
				props := extractExecuteProps(ev)
				switch props["status"] {
				case "running":
					runningID = g.messageID
				case "completed":
					completedID = g.messageID
				}
			}
		}
	}
	if runningID == "" || completedID == "" {
		t.Fatal("missing running/completed phases")
	}
	if runningID != completedID {
		t.Errorf("running msgID %q != completed %q", runningID, completedID)
	}
}

// --- extractSummary ---

func TestExtractSummary_Bash(t *testing.T) {
	got := claude.ExportExtractSummary("Bash", `{"command":"npm install"}`)
	if got != "npm install" {
		t.Errorf("got %q", got)
	}
}

func TestExtractSummary_Write(t *testing.T) {
	got := claude.ExportExtractSummary("Write", `{"file_path":"/workspace/main.go"}`)
	if got != "/workspace/main.go" {
		t.Errorf("got %q", got)
	}
}

func TestExtractSummary_Empty(t *testing.T) {
	if got := claude.ExportExtractSummary("Bash", ""); got != "" {
		t.Errorf("got %q", got)
	}
}

func TestExtractSummary_InvalidJSON(t *testing.T) {
	if got := claude.ExportExtractSummary("Bash", "{invalid"); got != "" {
		t.Errorf("got %q", got)
	}
}

// --- truncate ---

func TestTruncate_Short(t *testing.T) {
	if got := claude.ExportTruncate("hello", 80); got != "hello" {
		t.Errorf("got %q", got)
	}
}

func TestTruncate_Long(t *testing.T) {
	long := strings.Repeat("x", 100)
	got := claude.ExportTruncate(long, 80)
	if len(got) != 83 { // 80 + "..."
		t.Errorf("got len %d, expected 83", len(got))
	}
}

// --- error event ---

func TestParseErrorEvent(t *testing.T) {
	jsonl := `{"type":"error","error":{"message":"something went wrong"}}` + "\n"
	var events []recordedEvent
	p := claude.ExportNewStreamParser(mockStreamFunc(&events))
	r, w := io.Pipe()
	go func() { defer w.Close(); w.Write([]byte(jsonl)) }()
	err := p.Parse(context.Background(), r)
	if err == nil || !strings.Contains(err.Error(), "something went wrong") {
		t.Errorf("expected error, got %v", err)
	}
}

func TestParseResultError(t *testing.T) {
	jsonl := `{"type":"result","is_error":true,"result":"CLI crash"}` + "\n"
	var events []recordedEvent
	p := claude.ExportNewStreamParser(mockStreamFunc(&events))
	r, w := io.Pipe()
	go func() { defer w.Close(); w.Write([]byte(jsonl)) }()
	err := p.Parse(context.Background(), r)
	if err == nil || !strings.Contains(err.Error(), "CLI crash") {
		t.Errorf("expected error, got %v", err)
	}
}
