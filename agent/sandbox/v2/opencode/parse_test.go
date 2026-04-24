package opencode

import (
	"context"
	"encoding/json"
	"io"
	"strings"
	"sync"
	"testing"

	"github.com/yaoapp/yao/agent/output/message"
)

type chunkRecord struct {
	eventType message.StreamChunkType
	data      string
}

func collectHandler(records *[]chunkRecord, mu *sync.Mutex) message.StreamFunc {
	return func(chunkType message.StreamChunkType, data []byte) int {
		mu.Lock()
		defer mu.Unlock()
		*records = append(*records, chunkRecord{eventType: chunkType, data: string(data)})
		return 0
	}
}

func makeJSONL(events ...map[string]any) string {
	var lines []string
	for _, e := range events {
		data, _ := json.Marshal(e)
		lines = append(lines, string(data))
	}
	return strings.Join(lines, "\n") + "\n"
}

func TestParse_StepStartEmitsMetadata(t *testing.T) {
	input := makeJSONL(
		map[string]any{
			"type":      "step_start",
			"timestamp": 1000,
			"sessionID": "ses_123",
			"part": map[string]any{
				"id":        "prt_abc",
				"type":      "step-start",
				"messageID": "msg_xyz",
				"sessionID": "ses_123",
			},
		},
		map[string]any{
			"type":      "text",
			"timestamp": 2000,
			"sessionID": "ses_123",
			"part":      map[string]any{"text": "Hi!"},
		},
		map[string]any{
			"type":      "step_finish",
			"timestamp": 3000,
			"sessionID": "ses_123",
			"part":      map[string]any{"reason": "stop"},
		},
	)

	var records []chunkRecord
	var mu sync.Mutex
	handler := collectHandler(&records, &mu)

	parser := newStreamParser(handler)
	reader := io.NopCloser(strings.NewReader(input))
	err := parser.parse(context.Background(), reader)

	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	mu.Lock()
	defer mu.Unlock()

	// step_start should only emit metadata (no execute widget for pure text).
	hasMeta := false
	hasRunningExec := false
	for _, r := range records {
		if r.eventType == message.ChunkMetadata {
			var meta map[string]any
			json.Unmarshal([]byte(r.data), &meta)
			if _, ok := meta["opencode_session_id"]; ok {
				hasMeta = true
			}
		}
		if r.eventType == message.ChunkExecute {
			var props map[string]any
			json.Unmarshal([]byte(r.data), &props)
			if props["status"] == "running" {
				hasRunningExec = true
			}
		}
	}
	if !hasMeta {
		t.Error("step_start should emit metadata with session ID")
	}
	if hasRunningExec {
		t.Error("step_start should NOT emit a running execute for pure text steps")
	}
}

func TestParse_TextEvent(t *testing.T) {
	input := makeJSONL(
		map[string]any{
			"type":      "text",
			"timestamp": 1000,
			"sessionID": "ses_123",
			"part": map[string]any{
				"type":    "text",
				"content": "Hello, world!",
			},
		},
		map[string]any{
			"type":      "step_finish",
			"timestamp": 2000,
			"sessionID": "ses_123",
			"part": map[string]any{
				"type":         "step-finish",
				"finishReason": "stop",
			},
		},
	)

	var records []chunkRecord
	var mu sync.Mutex
	handler := collectHandler(&records, &mu)

	parser := newStreamParser(handler)
	reader := io.NopCloser(strings.NewReader(input))
	err := parser.parse(context.Background(), reader)

	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	if !parser.completed {
		t.Error("parser should be completed")
	}

	mu.Lock()
	defer mu.Unlock()

	hasText := false
	for _, r := range records {
		if r.eventType == message.ChunkText {
			hasText = true
			if r.data != "Hello, world!" {
				t.Errorf("text data = %q, want 'Hello, world!'", r.data)
			}
		}
	}
	if !hasText {
		t.Error("should have emitted a ChunkText event")
	}
}

func TestParse_ToolUseEvent(t *testing.T) {
	input := makeJSONL(
		map[string]any{
			"type":      "tool_use",
			"timestamp": 1000,
			"sessionID": "ses_123",
			"part": map[string]any{
				"type":       "tool",
				"toolName":   "bash",
				"toolCallId": "call_abc",
				"state": map[string]any{
					"status": "completed",
					"input":  `{"command":"ls -la"}`,
					"output": "file1.txt\nfile2.txt",
				},
			},
		},
		map[string]any{
			"type":      "step_finish",
			"timestamp": 2000,
			"sessionID": "ses_123",
			"part": map[string]any{
				"type":         "step-finish",
				"finishReason": "stop",
			},
		},
	)

	var records []chunkRecord
	var mu sync.Mutex
	handler := collectHandler(&records, &mu)

	parser := newStreamParser(handler)
	reader := io.NopCloser(strings.NewReader(input))
	err := parser.parse(context.Background(), reader)

	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	mu.Lock()
	defer mu.Unlock()

	hasCompletedExec := false
	for _, r := range records {
		if r.eventType == message.ChunkExecute {
			var props map[string]any
			json.Unmarshal([]byte(r.data), &props)
			if props["tool"] == "bash" && props["status"] == "completed" {
				hasCompletedExec = true
				if props["runner"] != "opencode-cli" {
					t.Errorf("runner = %v, want opencode-cli", props["runner"])
				}
			}
		}
	}
	if !hasCompletedExec {
		t.Error("should have emitted a ChunkExecute with tool=bash status=completed")
	}
}

func TestParse_ErrorEvent(t *testing.T) {
	input := makeJSONL(
		map[string]any{
			"type":      "error",
			"timestamp": 1000,
			"sessionID": "ses_123",
			"error":     "something went wrong",
		},
	)

	var records []chunkRecord
	var mu sync.Mutex
	handler := collectHandler(&records, &mu)

	parser := newStreamParser(handler)
	reader := io.NopCloser(strings.NewReader(input))
	err := parser.parse(context.Background(), reader)

	if err == nil {
		t.Fatal("expected error from parse")
	}
	if !strings.Contains(err.Error(), "something went wrong") {
		t.Errorf("error should contain message, got: %v", err)
	}

	mu.Lock()
	defer mu.Unlock()

	hasError := false
	for _, r := range records {
		if r.eventType == message.ChunkError {
			hasError = true
		}
	}
	if !hasError {
		t.Error("should have emitted a ChunkError event")
	}
}

func TestParse_ErrorEvent_Nested(t *testing.T) {
	input := makeJSONL(
		map[string]any{
			"type":      "error",
			"timestamp": 1000,
			"sessionID": "ses_123",
			"error": map[string]any{
				"name": "APIError",
				"data": map[string]any{
					"message":    "Authentication Fails",
					"statusCode": 401,
				},
			},
		},
	)

	var records []chunkRecord
	var mu sync.Mutex
	handler := collectHandler(&records, &mu)

	parser := newStreamParser(handler)
	reader := io.NopCloser(strings.NewReader(input))
	err := parser.parse(context.Background(), reader)

	if err == nil {
		t.Fatal("expected error from parse")
	}
	if !strings.Contains(err.Error(), "Authentication Fails") {
		t.Errorf("error should contain nested message, got: %v", err)
	}
}

func TestParse_StepFinishToolCalls(t *testing.T) {
	input := makeJSONL(
		map[string]any{
			"type":      "step_finish",
			"timestamp": 1000,
			"sessionID": "ses_123",
			"part": map[string]any{
				"type":         "step-finish",
				"finishReason": "tool-calls",
			},
		},
	)

	var records []chunkRecord
	var mu sync.Mutex
	handler := collectHandler(&records, &mu)

	parser := newStreamParser(handler)
	reader := io.NopCloser(strings.NewReader(input))
	err := parser.parse(context.Background(), reader)

	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	if parser.completed {
		t.Error("tool-calls finish reason should NOT mark as completed")
	}

	mu.Lock()
	defer mu.Unlock()

	hasTransition := false
	for _, r := range records {
		if r.eventType == message.ChunkMetadata {
			var meta map[string]any
			json.Unmarshal([]byte(r.data), &meta)
			if _, ok := meta["step_transition"]; ok {
				hasTransition = true
			}
		}
	}
	if !hasTransition {
		t.Error("tool-calls step_finish should emit step_transition metadata")
	}
}

func TestParse_MultiStepToolThenText(t *testing.T) {
	input := makeJSONL(
		map[string]any{"type": "step_start", "sessionID": "ses_1", "part": map[string]any{"id": "step-1"}},
		map[string]any{
			"type": "tool_use", "sessionID": "ses_1",
			"part": map[string]any{
				"toolName": "bash", "toolCallId": "call_1",
				"state": map[string]any{"status": "completed", "input": `{"command":"echo hi"}`, "output": "hi"},
			},
		},
		map[string]any{
			"type": "step_finish", "sessionID": "ses_1",
			"part": map[string]any{"finishReason": "tool-calls"},
		},
		map[string]any{"type": "step_start", "sessionID": "ses_1", "part": map[string]any{"id": "step-2"}},
		map[string]any{
			"type": "text", "sessionID": "ses_1",
			"part": map[string]any{"text": "The output was: hi"},
		},
		map[string]any{
			"type": "step_finish", "sessionID": "ses_1",
			"part": map[string]any{"reason": "stop"},
		},
	)

	var records []chunkRecord
	var mu sync.Mutex
	handler := collectHandler(&records, &mu)

	parser := newStreamParser(handler)
	reader := io.NopCloser(strings.NewReader(input))
	err := parser.parse(context.Background(), reader)

	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	if !parser.completed {
		t.Error("multi-step stream should complete on final stop")
	}

	mu.Lock()
	defer mu.Unlock()

	var (
		completedCount int
		hasText        bool
	)
	for _, r := range records {
		if r.eventType == message.ChunkExecute {
			var props map[string]any
			json.Unmarshal([]byte(r.data), &props)
			if props["status"] == "completed" {
				completedCount++
			}
		}
		if r.eventType == message.ChunkText && r.data == "The output was: hi" {
			hasText = true
		}
	}
	if completedCount < 1 {
		t.Error("should have at least 1 completed exec from tool_use")
	}
	if !hasText {
		t.Error("should have emitted ChunkText for final text reply")
	}
}

func TestParse_ReasoningEvent(t *testing.T) {
	input := makeJSONL(
		map[string]any{
			"type":      "reasoning",
			"timestamp": 1000,
			"sessionID": "ses_123",
			"part": map[string]any{
				"type": "reasoning",
				"text": "Let me think about this...",
			},
		},
		map[string]any{
			"type":      "step_finish",
			"timestamp": 2000,
			"sessionID": "ses_123",
			"part": map[string]any{
				"type":         "step-finish",
				"finishReason": "stop",
			},
		},
	)

	var records []chunkRecord
	var mu sync.Mutex
	handler := collectHandler(&records, &mu)

	parser := newStreamParser(handler)
	reader := io.NopCloser(strings.NewReader(input))
	err := parser.parse(context.Background(), reader)

	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	mu.Lock()
	defer mu.Unlock()

	hasReasoning := false
	for _, r := range records {
		if r.eventType == message.ChunkMetadata {
			var meta map[string]any
			json.Unmarshal([]byte(r.data), &meta)
			if _, ok := meta["reasoning"]; ok {
				hasReasoning = true
			}
		}
	}
	if !hasReasoning {
		t.Error("should have emitted reasoning metadata")
	}
}

func TestParse_UnknownEventType(t *testing.T) {
	input := makeJSONL(
		map[string]any{
			"type":      "future_event_type",
			"timestamp": 1000,
			"sessionID": "ses_123",
		},
		map[string]any{
			"type":      "step_finish",
			"timestamp": 2000,
			"sessionID": "ses_123",
			"part": map[string]any{
				"type":         "step-finish",
				"finishReason": "stop",
			},
		},
	)

	var records []chunkRecord
	var mu sync.Mutex
	handler := collectHandler(&records, &mu)

	parser := newStreamParser(handler)
	reader := io.NopCloser(strings.NewReader(input))
	err := parser.parse(context.Background(), reader)

	if err != nil {
		t.Fatalf("unknown event type should not cause error: %v", err)
	}
	if !parser.completed {
		t.Error("parser should still complete")
	}
}

func TestExtractSummary(t *testing.T) {
	cases := []struct {
		tool, input, want string
	}{
		{"bash", `{"command":"ls -la /tmp"}`, "ls -la /tmp"},
		{"read", `{"file_path":"main.go"}`, "main.go"},
		{"write", `{"file_path":"output.txt"}`, "output.txt"},
		{"unknown", `{"query":"select * from users"}`, "select * from users"},
		{"bash", `{"no_command_key": true}`, ""},
		{"bash", `invalid json`, ""},
		{"bash", "", ""},
	}
	for _, tc := range cases {
		got := extractSummary(tc.tool, tc.input)
		if got != tc.want {
			t.Errorf("extractSummary(%q, %q) = %q, want %q", tc.tool, tc.input, got, tc.want)
		}
	}
}

func TestTruncate(t *testing.T) {
	short := "hello"
	if truncate(short, 80) != "hello" {
		t.Error("short string should not be truncated")
	}

	long := strings.Repeat("a", 100)
	result := truncate(long, 80)
	if len(result) != 83 { // 80 + "..."
		t.Errorf("truncated len = %d, want 83", len(result))
	}

	withNewlines := "line1\nline2\nline3"
	result = truncate(withNewlines, 80)
	if strings.Contains(result, "\n") {
		t.Error("truncate should replace newlines with spaces")
	}
}
