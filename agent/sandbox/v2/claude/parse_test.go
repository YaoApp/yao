//go:build unit

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

// --- Agent adoption + sub-agent tool registration ---

// agentWithSubToolJSONL simulates:
// 1. Agent tool streamed via content_block_start (id=stream_agent)
// 2. Agent tool finalised in assistant message (id=final_agent, different ID — a2o re-keying)
// 3. Sub-agent Bash tool via assistant message (parent_tool_use_id=final_agent, stop_reason=null)
// 4. Sub-agent Bash tool_result via user message
// 5. Agent tool_result via user message
const agentWithSubToolJSONL = `{"type":"stream_event","event":{"type":"message_start","message":{"id":"msg_a1","type":"message","role":"assistant","content":[],"model":"claude-sonnet-4-20250514","stop_reason":null,"usage":{"input_tokens":100,"output_tokens":1}}}}
{"type":"stream_event","event":{"type":"content_block_start","index":0,"content_block":{"type":"tool_use","id":"stream_agent","name":"Agent"}}}
{"type":"stream_event","event":{"type":"content_block_delta","index":0,"delta":{"type":"input_json_delta","partial_json":"{\"prompt\""}}}
{"type":"stream_event","event":{"type":"content_block_delta","index":0,"delta":{"type":"input_json_delta","partial_json":":\"do something\"}"}}}
{"type":"stream_event","event":{"type":"content_block_stop","index":0}}
{"type":"assistant","message":{"id":"msg_a1","type":"message","role":"assistant","content":[{"type":"tool_use","id":"final_agent","name":"Agent","input":{"prompt":"do something"}}],"model":"claude-sonnet-4-20250514","stop_reason":"end_turn","usage":{"input_tokens":100,"output_tokens":30}}}
{"type":"assistant","parent_tool_use_id":"final_agent","message":{"id":"msg_sub1","type":"message","role":"assistant","content":[{"type":"tool_use","id":"sub_bash","name":"Bash","input":{"command":"echo hello"}}],"model":"claude-sonnet-4-20250514","stop_reason":null,"usage":{"input_tokens":50,"output_tokens":10}}}
{"type":"user","parent_tool_use_id":"final_agent","message":{"id":"msg_sub1_r","type":"message","role":"user","content":[{"type":"tool_result","tool_use_id":"sub_bash","content":"hello"}]}}
{"type":"user","message":{"id":"msg_a1_r","type":"message","role":"user","content":[{"type":"tool_result","tool_use_id":"final_agent","content":"Agent completed"}]}}
{"type":"result","result":"done","is_error":false}
`

func TestAdoption_NoPhantomMessages(t *testing.T) {
	events := runParser(t, agentWithSubToolJSONL)
	groups := extractMessageGroups(events)

	var agentGroups []messageGroup
	for _, g := range groups {
		for _, ev := range g.events {
			if ev.chunkType == message.ChunkExecute {
				props := extractExecuteProps(ev)
				if strDefault(props["tool"]) == "Agent" {
					agentGroups = append(agentGroups, g)
					break
				}
			}
		}
	}

	// Streaming phase creates one group (running), tool_result reuses the
	// same msgID (completed). Adoption ensures no phantom from final_agent.
	if len(agentGroups) != 2 {
		t.Fatalf("expected 2 Agent message groups (running + completed), got %d", len(agentGroups))
	}
	if agentGroups[0].messageID != agentGroups[1].messageID {
		t.Errorf("Agent running/completed msgID mismatch: %q vs %q",
			agentGroups[0].messageID, agentGroups[1].messageID)
	}

	// The streaming-phase group should use stream_agent as tool_id (it IS
	// emitted now — this is the real-time feedback). The adoption remaps
	// final_agent → same msgID, so no phantom group is created for final_agent.
	distinctMsgIDs := map[string]bool{}
	for _, g := range agentGroups {
		distinctMsgIDs[g.messageID] = true
	}
	if len(distinctMsgIDs) != 1 {
		t.Errorf("expected 1 distinct Agent msgID, got %d", len(distinctMsgIDs))
	}
}

func TestAdoption_StreamingEmitsInput(t *testing.T) {
	events := runParser(t, agentWithSubToolJSONL)

	// Under adoption, input_delta IS emitted during the streaming phase
	// (unlike the old deferred strategy that suppressed it).
	var hasInputDelta bool
	for _, ev := range events {
		if ev.chunkType == message.ChunkExecute {
			props := extractExecuteProps(ev)
			if _, ok := props["input_delta"]; ok {
				hasInputDelta = true
				break
			}
		}
	}
	if !hasInputDelta {
		t.Error("adoption strategy should emit input_delta during streaming phase")
	}
}

func TestSubAgentTool_HasToolName(t *testing.T) {
	events := runParser(t, agentWithSubToolJSONL)
	groups := extractMessageGroups(events)

	// Find the sub-agent Bash tool_result (completed phase).
	var bashCompletedProps map[string]any
	for _, g := range groups {
		for _, ev := range g.events {
			if ev.chunkType == message.ChunkExecute {
				props := extractExecuteProps(ev)
				if strDefault(props["status"]) == "completed" && strDefault(props["tool"]) == "Bash" {
					bashCompletedProps = props
				}
			}
		}
	}
	if bashCompletedProps == nil {
		t.Fatal("missing completed Bash sub-tool event")
	}
	if strDefault(bashCompletedProps["tool"]) != "Bash" {
		t.Errorf("sub-tool name = %q, want Bash", strDefault(bashCompletedProps["tool"]))
	}
	if strDefault(bashCompletedProps["summary"]) != "echo hello" {
		t.Errorf("sub-tool summary = %q, want 'echo hello'", strDefault(bashCompletedProps["summary"]))
	}
}

func TestSubAgentTool_HasParentIDs(t *testing.T) {
	events := runParser(t, agentWithSubToolJSONL)

	// Collect all execute chunks with parent info.
	var subToolChunks []map[string]any
	for _, ev := range events {
		if ev.chunkType == message.ChunkExecute {
			props := extractExecuteProps(ev)
			if _, has := props["parent_tool_use_id"]; has {
				subToolChunks = append(subToolChunks, props)
			}
		}
	}

	if len(subToolChunks) == 0 {
		t.Fatal("no sub-tool chunks with parent_tool_use_id found")
	}

	for _, props := range subToolChunks {
		ptid := strDefault(props["parent_tool_use_id"])
		pmid := strDefault(props["parent_message_id"])
		if ptid != "final_agent" {
			t.Errorf("parent_tool_use_id = %q, want final_agent", ptid)
		}
		if pmid == "" {
			t.Error("parent_message_id should be set for sub-agent tools")
		}
	}
}

func TestMainThreadTool_NoParentID(t *testing.T) {
	events := runParser(t, agentWithSubToolJSONL)
	groups := extractMessageGroups(events)

	// Find the Agent tool's own execute events — they should NOT carry parent IDs.
	for _, g := range groups {
		for _, ev := range g.events {
			if ev.chunkType == message.ChunkExecute {
				props := extractExecuteProps(ev)
				if strDefault(props["tool"]) == "Agent" {
					if _, has := props["parent_tool_use_id"]; has {
						t.Error("main-thread Agent tool should not have parent_tool_use_id")
					}
				}
			}
		}
	}
}

// agentSameIDJSONL tests the case where streaming and final Agent IDs are
// identical (direct Anthropic connection, no a2o re-keying).
const agentSameIDJSONL = `{"type":"stream_event","event":{"type":"message_start","message":{"id":"msg_x","type":"message","role":"assistant","content":[],"model":"claude-sonnet-4-20250514","stop_reason":null,"usage":{"input_tokens":100,"output_tokens":1}}}}
{"type":"stream_event","event":{"type":"content_block_start","index":0,"content_block":{"type":"tool_use","id":"agent_same","name":"Agent"}}}
{"type":"stream_event","event":{"type":"content_block_delta","index":0,"delta":{"type":"input_json_delta","partial_json":"{\"prompt\":\"test\"}"}}}
{"type":"stream_event","event":{"type":"content_block_stop","index":0}}
{"type":"assistant","message":{"id":"msg_x","type":"message","role":"assistant","content":[{"type":"tool_use","id":"agent_same","name":"Agent","input":{"prompt":"test"}}],"model":"claude-sonnet-4-20250514","stop_reason":"end_turn","usage":{"input_tokens":100,"output_tokens":20}}}
{"type":"user","message":{"id":"msg_x_r","type":"message","role":"user","content":[{"type":"tool_result","tool_use_id":"agent_same","content":"done"}]}}
{"type":"result","result":"done","is_error":false}
`

func TestAdoption_SameID_Passthrough(t *testing.T) {
	events := runParser(t, agentSameIDJSONL)
	groups := extractMessageGroups(events)

	var agentGroups []messageGroup
	for _, g := range groups {
		for _, ev := range g.events {
			if ev.chunkType == message.ChunkExecute {
				props := extractExecuteProps(ev)
				if strDefault(props["tool"]) == "Agent" {
					agentGroups = append(agentGroups, g)
					break
				}
			}
		}
	}

	// Same ID: alreadyStreamed triggers, no adoption needed. Still 2 groups
	// (streaming running + tool_result completed), sharing the same msgID.
	if len(agentGroups) != 2 {
		t.Fatalf("expected 2 Agent groups (running + completed), got %d", len(agentGroups))
	}
	if agentGroups[0].messageID != agentGroups[1].messageID {
		t.Errorf("Agent msgID mismatch when IDs are the same: %q vs %q",
			agentGroups[0].messageID, agentGroups[1].messageID)
	}
}

// agentMultiJSONL tests two consecutive Agent calls with different streaming/final IDs.
// FIFO adoption: final_1 adopts stream_1, final_2 adopts stream_2.
const agentMultiJSONL = `{"type":"stream_event","event":{"type":"message_start","message":{"id":"msg_m","type":"message","role":"assistant","content":[],"model":"claude-sonnet-4-20250514","stop_reason":null,"usage":{"input_tokens":100,"output_tokens":1}}}}
{"type":"stream_event","event":{"type":"content_block_start","index":0,"content_block":{"type":"tool_use","id":"stream_1","name":"Agent"}}}
{"type":"stream_event","event":{"type":"content_block_delta","index":0,"delta":{"type":"input_json_delta","partial_json":"{\"prompt\":\"task1\"}"}}}
{"type":"stream_event","event":{"type":"content_block_stop","index":0}}
{"type":"stream_event","event":{"type":"content_block_start","index":1,"content_block":{"type":"tool_use","id":"stream_2","name":"Agent"}}}
{"type":"stream_event","event":{"type":"content_block_delta","index":1,"delta":{"type":"input_json_delta","partial_json":"{\"prompt\":\"task2\"}"}}}
{"type":"stream_event","event":{"type":"content_block_stop","index":1}}
{"type":"assistant","message":{"id":"msg_m","type":"message","role":"assistant","content":[{"type":"tool_use","id":"final_1","name":"Agent","input":{"prompt":"task1"}},{"type":"tool_use","id":"final_2","name":"Agent","input":{"prompt":"task2"}}],"model":"claude-sonnet-4-20250514","stop_reason":"end_turn","usage":{"input_tokens":100,"output_tokens":40}}}
{"type":"user","message":{"id":"msg_m_r1","type":"message","role":"user","content":[{"type":"tool_result","tool_use_id":"final_1","content":"done1"}]}}
{"type":"user","message":{"id":"msg_m_r2","type":"message","role":"user","content":[{"type":"tool_result","tool_use_id":"final_2","content":"done2"}]}}
{"type":"result","result":"done","is_error":false}
`

func TestAdoption_MultiAgent_FIFOOrder(t *testing.T) {
	events := runParser(t, agentMultiJSONL)
	groups := extractMessageGroups(events)

	var agentGroups []messageGroup
	for _, g := range groups {
		for _, ev := range g.events {
			if ev.chunkType == message.ChunkExecute {
				props := extractExecuteProps(ev)
				if strDefault(props["tool"]) == "Agent" {
					agentGroups = append(agentGroups, g)
					break
				}
			}
		}
	}

	// 2 streaming Agents + 2 completed Agents = 4 groups, but only 2 distinct msgIDs.
	if len(agentGroups) != 4 {
		t.Fatalf("expected 4 Agent groups (2 running + 2 completed), got %d", len(agentGroups))
	}

	distinctMsgIDs := map[string]bool{}
	for _, g := range agentGroups {
		distinctMsgIDs[g.messageID] = true
	}
	if len(distinctMsgIDs) != 2 {
		t.Errorf("expected 2 distinct Agent msgIDs, got %d", len(distinctMsgIDs))
	}

	// Running and completed phases for each Agent should share the same msgID.
	if agentGroups[0].messageID != agentGroups[2].messageID {
		t.Errorf("Agent 1 running/completed msgID mismatch: %q vs %q",
			agentGroups[0].messageID, agentGroups[2].messageID)
	}
	if agentGroups[1].messageID != agentGroups[3].messageID {
		t.Errorf("Agent 2 running/completed msgID mismatch: %q vs %q",
			agentGroups[1].messageID, agentGroups[3].messageID)
	}
}

// agentMixedIDJSONL tests Scenario D: one Agent's streaming ID matches its
// final ID (direct Anthropic), while another's differs (a2o re-keying).
// Validates BUG 1 fix: alreadyStreamed must clean pendingAgents to prevent
// the next Agent from incorrectly adopting it.
const agentMixedIDJSONL = `{"type":"stream_event","event":{"type":"message_start","message":{"id":"msg_d","type":"message","role":"assistant","content":[],"model":"claude-sonnet-4-20250514","stop_reason":null,"usage":{"input_tokens":100,"output_tokens":1}}}}
{"type":"stream_event","event":{"type":"content_block_start","index":0,"content_block":{"type":"tool_use","id":"A","name":"Agent"}}}
{"type":"stream_event","event":{"type":"content_block_delta","index":0,"delta":{"type":"input_json_delta","partial_json":"{\"prompt\":\"taskA\"}"}}}
{"type":"stream_event","event":{"type":"content_block_stop","index":0}}
{"type":"stream_event","event":{"type":"content_block_start","index":1,"content_block":{"type":"tool_use","id":"B","name":"Agent"}}}
{"type":"stream_event","event":{"type":"content_block_delta","index":1,"delta":{"type":"input_json_delta","partial_json":"{\"prompt\":\"taskB\"}"}}}
{"type":"stream_event","event":{"type":"content_block_stop","index":1}}
{"type":"assistant","message":{"id":"msg_d","type":"message","role":"assistant","content":[{"type":"tool_use","id":"A","name":"Agent","input":{"prompt":"taskA"}},{"type":"tool_use","id":"C","name":"Agent","input":{"prompt":"taskB"}}],"model":"claude-sonnet-4-20250514","stop_reason":"end_turn","usage":{"input_tokens":100,"output_tokens":40}}}
{"type":"user","message":{"id":"msg_d_r1","type":"message","role":"user","content":[{"type":"tool_result","tool_use_id":"A","content":"doneA"}]}}
{"type":"user","message":{"id":"msg_d_r2","type":"message","role":"user","content":[{"type":"tool_result","tool_use_id":"C","content":"doneB"}]}}
{"type":"result","result":"done","is_error":false}
`

func TestAdoption_MixedID_NoLeak(t *testing.T) {
	events := runParser(t, agentMixedIDJSONL)
	groups := extractMessageGroups(events)

	var agentGroups []messageGroup
	for _, g := range groups {
		for _, ev := range g.events {
			if ev.chunkType == message.ChunkExecute {
				props := extractExecuteProps(ev)
				if strDefault(props["tool"]) == "Agent" {
					agentGroups = append(agentGroups, g)
					break
				}
			}
		}
	}

	// Stream A + Stream B = 2 running groups.
	// A matches (alreadyStreamed, cleans pendingAgents).
	// C adopts B (pop from pendingAgents).
	// tool_result A + tool_result C = 2 completed groups.
	// Total: 4 Agent groups, 2 distinct msgIDs.
	if len(agentGroups) != 4 {
		t.Fatalf("expected 4 Agent groups, got %d", len(agentGroups))
	}

	distinctMsgIDs := map[string]bool{}
	for _, g := range agentGroups {
		distinctMsgIDs[g.messageID] = true
	}
	if len(distinctMsgIDs) != 2 {
		t.Errorf("expected 2 distinct Agent msgIDs, got %d", len(distinctMsgIDs))
	}

	// Agent A: streaming (group 0) and completed (group 2) share the same msgID.
	if agentGroups[0].messageID != agentGroups[2].messageID {
		t.Errorf("Agent A running/completed msgID mismatch: %q vs %q",
			agentGroups[0].messageID, agentGroups[2].messageID)
	}
	// Agent C adopted B: streaming B (group 1) and completed C (group 3) share the same msgID.
	if agentGroups[1].messageID != agentGroups[3].messageID {
		t.Errorf("Agent B→C running/completed msgID mismatch: %q vs %q",
			agentGroups[1].messageID, agentGroups[3].messageID)
	}
}

// agentMultiWithSubToolsJSONL simulates two concurrent Agents, each with a
// sub-tool call. Streaming IDs differ from final IDs (a2o re-keying).
// Validates: adoption FIFO order, parent_message_id propagation per Agent,
// summary/input population on both Agent completions, and main-thread
// tool_results having no parent_message_id.
const agentMultiWithSubToolsJSONL = `{"type":"stream_event","event":{"type":"message_start","message":{"id":"msg_mt","type":"message","role":"assistant","content":[],"model":"claude-sonnet-4-20250514","stop_reason":null,"usage":{"input_tokens":100,"output_tokens":1}}}}
{"type":"stream_event","event":{"type":"content_block_start","index":0,"content_block":{"type":"tool_use","id":"stream_1","name":"Agent"}}}
{"type":"stream_event","event":{"type":"content_block_delta","index":0,"delta":{"type":"input_json_delta","partial_json":"{\"prompt\":\"research AI\"}"}}}
{"type":"stream_event","event":{"type":"content_block_stop","index":0}}
{"type":"stream_event","event":{"type":"content_block_start","index":1,"content_block":{"type":"tool_use","id":"stream_2","name":"Agent"}}}
{"type":"stream_event","event":{"type":"content_block_delta","index":1,"delta":{"type":"input_json_delta","partial_json":"{\"prompt\":\"write report\"}"}}}
{"type":"stream_event","event":{"type":"content_block_stop","index":1}}
{"type":"assistant","message":{"id":"msg_mt","type":"message","role":"assistant","content":[{"type":"tool_use","id":"final_1","name":"Agent","input":{"prompt":"research AI"}},{"type":"tool_use","id":"final_2","name":"Agent","input":{"prompt":"write report"}}],"model":"claude-sonnet-4-20250514","stop_reason":"end_turn","usage":{"input_tokens":100,"output_tokens":40}}}
{"type":"assistant","parent_tool_use_id":"final_1","message":{"id":"msg_s1a","type":"message","role":"assistant","content":[{"type":"tool_use","id":"sub_bash_1","name":"Bash","input":{"command":"curl https://ai.news"}}],"model":"claude-sonnet-4-20250514","stop_reason":null,"usage":{"input_tokens":50,"output_tokens":10}}}
{"type":"user","parent_tool_use_id":"final_1","message":{"id":"msg_s1r","type":"message","role":"user","content":[{"type":"tool_result","tool_use_id":"sub_bash_1","content":"AI news content"}]}}
{"type":"assistant","parent_tool_use_id":"final_2","message":{"id":"msg_s2a","type":"message","role":"assistant","content":[{"type":"tool_use","id":"sub_write_1","name":"Write","input":{"file_path":"/tmp/report.md","content":"# Report"}}],"model":"claude-sonnet-4-20250514","stop_reason":null,"usage":{"input_tokens":50,"output_tokens":15}}}
{"type":"user","parent_tool_use_id":"final_2","message":{"id":"msg_s2r","type":"message","role":"user","content":[{"type":"tool_result","tool_use_id":"sub_write_1","content":"File written"}]}}
{"type":"user","message":{"id":"msg_mt_r","type":"message","role":"user","content":[{"type":"tool_result","tool_use_id":"final_1","content":"Agent 1 done"},{"type":"tool_result","tool_use_id":"final_2","content":"Agent 2 done"}]}}
{"type":"result","result":"done","is_error":false}
`

func TestAdoption_MultiAgent_WithSubTools(t *testing.T) {
	events := runParser(t, agentMultiWithSubToolsJSONL)
	groups := extractMessageGroups(events)

	// --- Verify Agent groups ---
	var agentGroups []messageGroup
	for _, g := range groups {
		for _, ev := range g.events {
			if ev.chunkType == message.ChunkExecute {
				props := extractExecuteProps(ev)
				if strDefault(props["tool"]) == "Agent" {
					agentGroups = append(agentGroups, g)
					break
				}
			}
		}
	}

	// 2 streaming (running) + 2 completed = 4 Agent groups, 2 distinct msgIDs.
	if len(agentGroups) != 4 {
		t.Fatalf("expected 4 Agent groups (2 running + 2 completed), got %d", len(agentGroups))
	}

	distinctMsgIDs := map[string]bool{}
	for _, g := range agentGroups {
		distinctMsgIDs[g.messageID] = true
	}
	if len(distinctMsgIDs) != 2 {
		t.Errorf("expected 2 distinct Agent msgIDs, got %d", len(distinctMsgIDs))
	}

	// Running and completed phases share the same msgID per Agent.
	if agentGroups[0].messageID != agentGroups[2].messageID {
		t.Errorf("Agent 1 running/completed msgID mismatch: %q vs %q",
			agentGroups[0].messageID, agentGroups[2].messageID)
	}
	if agentGroups[1].messageID != agentGroups[3].messageID {
		t.Errorf("Agent 2 running/completed msgID mismatch: %q vs %q",
			agentGroups[1].messageID, agentGroups[3].messageID)
	}
	agent1MsgID := agentGroups[0].messageID
	agent2MsgID := agentGroups[1].messageID

	// --- Verify Agent completions have input and summary ---
	for _, idx := range []int{2, 3} {
		g := agentGroups[idx]
		for _, ev := range g.events {
			if ev.chunkType == message.ChunkExecute {
				props := extractExecuteProps(ev)
				if strDefault(props["status"]) == "completed" {
					if props["input"] == nil {
						t.Errorf("Agent group %d completion missing input", idx)
					}
					if strDefault(props["summary"]) == "" {
						t.Errorf("Agent group %d completion missing summary", idx)
					}
				}
			}
		}
	}

	// --- Verify sub-tool parent_message_id ---
	var subToolChunks []map[string]any
	for _, ev := range events {
		if ev.chunkType == message.ChunkExecute {
			props := extractExecuteProps(ev)
			if _, has := props["parent_message_id"]; has {
				subToolChunks = append(subToolChunks, props)
			}
		}
	}

	// Sub-tools from both Agents should have parent_message_id.
	parentMsgIDs := map[string]bool{}
	for _, props := range subToolChunks {
		pmid := strDefault(props["parent_message_id"])
		parentMsgIDs[pmid] = true
	}
	if !parentMsgIDs[agent1MsgID] {
		t.Errorf("no sub-tool found with parent_message_id=%q (Agent 1)", agent1MsgID)
	}
	if !parentMsgIDs[agent2MsgID] {
		t.Errorf("no sub-tool found with parent_message_id=%q (Agent 2)", agent2MsgID)
	}

	// Verify parent_tool_use_id correctness for each sub-tool.
	for _, props := range subToolChunks {
		ptid := strDefault(props["parent_tool_use_id"])
		pmid := strDefault(props["parent_message_id"])
		if ptid == "final_1" && pmid != agent1MsgID {
			t.Errorf("sub-tool with parent_tool_use_id=final_1: parent_message_id=%q, want %q", pmid, agent1MsgID)
		}
		if ptid == "final_2" && pmid != agent2MsgID {
			t.Errorf("sub-tool with parent_tool_use_id=final_2: parent_message_id=%q, want %q", pmid, agent2MsgID)
		}
	}

	// --- Verify main-thread tool_results have NO parent_message_id ---
	for _, g := range agentGroups {
		for _, ev := range g.events {
			if ev.chunkType == message.ChunkExecute {
				props := extractExecuteProps(ev)
				if strDefault(props["tool"]) == "Agent" {
					if _, has := props["parent_message_id"]; has {
						t.Error("main-thread Agent tool_result should not have parent_message_id")
					}
				}
			}
		}
	}
}
