package claude

import (
	"context"
	"encoding/json"
	"io"
	"strings"
	"testing"

	"github.com/yaoapp/yao/agent/output/message"
)

// recordedEvent captures a single StreamFunc callback invocation.
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

// runParser feeds JSONL lines to a streamParser and returns all recorded events.
func runParser(t *testing.T, jsonl string) []recordedEvent {
	t.Helper()
	var events []recordedEvent
	p := newStreamParser(mockStreamFunc(&events))
	r, w := io.Pipe()
	go func() {
		defer w.Close()
		w.Write([]byte(jsonl))
	}()
	if err := p.parse(context.Background(), r); err != nil {
		t.Fatalf("parse error: %v", err)
	}
	return events
}

// --- helpers to inspect recorded events ---

type messageGroup struct {
	messageID string
	events    []recordedEvent
}

// extractMessageGroups splits the flat event list into message_start..message_end
// groups, each carrying the messageID from the start event.
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

// --- Mock JSONL data based on real terminal logs ---

// parallelToolCallJSONL reproduces the exact interleaved event sequence from
// the production log: two Bash tool calls (pip --version + npm --version) where
// tool0's tool_result arrives while tool1 is still streaming content_block_delta.
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

// ---------- Test 1: message_start / message_end pairing ----------

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
				t.Fatalf("nested message_start detected (depth %d) — message_start/message_end not strictly paired", depth)
			}
		case message.ChunkMessageEnd:
			endCount++
			depth--
			if depth < 0 {
				t.Fatal("message_end without preceding message_start")
			}
		}
	}

	if startCount != endCount {
		t.Fatalf("message_start count (%d) != message_end count (%d)", startCount, endCount)
	}
	// 4 execute phases (pip running, npm running, pip completed, npm completed) + 1 text = 5
	if startCount < 5 {
		t.Errorf("expected at least 5 message groups, got %d", startCount)
	}
}

// ---------- Test 2: each tool has running + completed lifecycle ----------

func TestParseParallelToolCalls_ToolLifecycle(t *testing.T) {
	events := runParser(t, parallelToolCallJSONL)
	groups := extractMessageGroups(events)

	// Collect execute groups by message_id
	type toolPhase struct {
		messageID string
		status    string
		tool      string
		output    any
	}
	var phases []toolPhase
	for _, g := range groups {
		for _, ev := range g.events {
			if ev.chunkType == message.ChunkExecute {
				props := extractExecuteProps(ev)
				status, _ := props["status"].(string)
				if status != "" {
					phases = append(phases, toolPhase{
						messageID: g.messageID,
						status:    status,
						tool:      strDefault(props["tool"]),
						output:    props["output"],
					})
				}
			}
		}
	}

	// Find pip and npm phases
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

	if pipRunning == nil {
		t.Fatal("pip running phase not found")
	}
	if pipCompleted == nil {
		t.Fatal("pip completed phase not found")
	}
	if npmRunning == nil {
		t.Fatal("npm running phase not found")
	}
	if npmCompleted == nil {
		t.Fatal("npm completed phase not found")
	}

	// running and completed phases must share message_id (reuse)
	if pipRunning.messageID != pipCompleted.messageID {
		t.Errorf("pip: running message_id %q != completed message_id %q", pipRunning.messageID, pipCompleted.messageID)
	}
	if npmRunning.messageID != npmCompleted.messageID {
		t.Errorf("npm: running message_id %q != completed message_id %q", npmRunning.messageID, npmCompleted.messageID)
	}

	// two tools should have different message_ids
	if pipRunning.messageID == npmRunning.messageID {
		t.Error("pip and npm should have different message_ids")
	}
}

// ---------- Test 3: interleaved result does not corrupt message_id ----------

func TestParseParallelToolCalls_InterleavedResult(t *testing.T) {
	events := runParser(t, parallelToolCallJSONL)
	groups := extractMessageGroups(events)

	// Find the npm running group (the one with tool_id toolu_npm or second Bash running)
	var npmRunningGroup *messageGroup
	bashRunningCount := 0
	for i := range groups {
		for _, ev := range groups[i].events {
			if ev.chunkType == message.ChunkExecute {
				props := extractExecuteProps(ev)
				if props["status"] == "running" && props["tool"] == "Bash" {
					bashRunningCount++
					if bashRunningCount == 2 {
						npmRunningGroup = &groups[i]
					}
				}
			}
		}
	}

	if npmRunningGroup == nil {
		t.Fatal("npm running message group not found")
	}

	// The npm running group must have a non-empty message_id
	if npmRunningGroup.messageID == "" {
		t.Fatal("npm running group has empty message_id")
	}

	// All execute chunks in the npm running group must have input_delta data
	// (the streaming deltas must be present, proving they weren't lost)
	var hasDelta bool
	for _, ev := range npmRunningGroup.events {
		if ev.chunkType == message.ChunkExecute {
			props := extractExecuteProps(ev)
			if _, ok := props["input_delta"]; ok {
				hasDelta = true
			}
		}
	}
	if !hasDelta {
		t.Error("npm running group has no input_delta chunks — streaming was interrupted")
	}

	// pip completed group must not contain npm delta events:
	// find pip completed group
	var pipCompletedGroup *messageGroup
	for i := range groups {
		for _, ev := range groups[i].events {
			if ev.chunkType == message.ChunkExecute {
				props := extractExecuteProps(ev)
				out := strDefault(props["output"])
				if props["status"] == "completed" && strings.Contains(out, "pip") {
					pipCompletedGroup = &groups[i]
				}
			}
		}
	}

	if pipCompletedGroup == nil {
		t.Fatal("pip completed group not found")
	}

	// pip completed group should only contain its own events, not npm deltas
	for _, ev := range pipCompletedGroup.events {
		if ev.chunkType == message.ChunkExecute {
			props := extractExecuteProps(ev)
			if _, ok := props["input_delta"]; ok {
				t.Error("pip completed group contains input_delta — npm delta leaked into pip group")
			}
		}
	}
}

// ---------- Test 4: final text after all tools complete ----------

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
		t.Errorf("final text should mention pip and npm, got %q", fullText)
	}
}

// ---------- Test 5: single tool call regression ----------

func TestParseSingleToolCall(t *testing.T) {
	events := runParser(t, singleToolCallJSONL)
	groups := extractMessageGroups(events)

	// Should have: 1 running group + 1 completed group + 1 text group = 3
	if len(groups) < 3 {
		t.Fatalf("expected at least 3 message groups for single tool call, got %d", len(groups))
	}

	// Verify message_start / message_end pairing
	depth := 0
	for _, ev := range events {
		switch ev.chunkType {
		case message.ChunkMessageStart:
			depth++
			if depth > 1 {
				t.Fatal("nested message_start in single tool call")
			}
		case message.ChunkMessageEnd:
			depth--
			if depth < 0 {
				t.Fatal("unmatched message_end in single tool call")
			}
		}
	}
	if depth != 0 {
		t.Fatalf("unbalanced message_start/message_end (depth=%d)", depth)
	}

	// Verify tool lifecycle: running then completed with same message_id
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

	if runningID == "" {
		t.Fatal("no running phase found")
	}
	if completedID == "" {
		t.Fatal("no completed phase found")
	}
	if runningID != completedID {
		t.Errorf("running message_id %q != completed message_id %q", runningID, completedID)
	}

	// Verify text output
	var hasText bool
	for _, ev := range events {
		if ev.chunkType == message.ChunkText && strings.Contains(string(ev.data), "Done") {
			hasText = true
		}
	}
	if !hasText {
		t.Error("final text 'Done.' not found")
	}
}

func strDefault(v any) string {
	if v == nil {
		return ""
	}
	s, _ := v.(string)
	return s
}
