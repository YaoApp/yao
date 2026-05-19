//go:build unit

package opencode_test

import (
	"context"
	"encoding/json"
	"io"
	"strings"
	"sync"
	"testing"

	"github.com/yaoapp/yao/agent/output/message"
	opencode "github.com/yaoapp/yao/agent/sandbox/v2/opencode"
)

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

	var records []opencode.ChunkRecord
	var mu sync.Mutex
	handler := opencode.CollectHandler(&records, &mu)

	parser := opencode.NewStreamParserForTest(handler)
	reader := io.NopCloser(strings.NewReader(input))
	err := opencode.StreamParserParse(parser, context.Background(), reader)

	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	mu.Lock()
	defer mu.Unlock()

	hasMeta := false
	hasRunningExec := false
	for _, r := range records {
		if r.EventType == message.ChunkMetadata {
			var meta map[string]any
			json.Unmarshal([]byte(r.Data), &meta)
			if _, ok := meta["opencode_session_id"]; ok {
				hasMeta = true
			}
		}
		if r.EventType == message.ChunkExecute {
			var props map[string]any
			json.Unmarshal([]byte(r.Data), &props)
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

	var records []opencode.ChunkRecord
	var mu sync.Mutex
	handler := opencode.CollectHandler(&records, &mu)

	parser := opencode.NewStreamParserForTest(handler)
	reader := io.NopCloser(strings.NewReader(input))
	err := opencode.StreamParserParse(parser, context.Background(), reader)

	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	if !opencode.StreamParserCompleted(parser) {
		t.Error("parser should be completed")
	}

	mu.Lock()
	defer mu.Unlock()

	hasText := false
	for _, r := range records {
		if r.EventType == message.ChunkText {
			hasText = true
			if r.Data != "Hello, world!" {
				t.Errorf("text data = %q, want 'Hello, world!'", r.Data)
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

	var records []opencode.ChunkRecord
	var mu sync.Mutex
	handler := opencode.CollectHandler(&records, &mu)

	parser := opencode.NewStreamParserForTest(handler)
	reader := io.NopCloser(strings.NewReader(input))
	err := opencode.StreamParserParse(parser, context.Background(), reader)

	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	mu.Lock()
	defer mu.Unlock()

	hasCompletedExec := false
	for _, r := range records {
		if r.EventType == message.ChunkExecute {
			var props map[string]any
			json.Unmarshal([]byte(r.Data), &props)
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

	var records []opencode.ChunkRecord
	var mu sync.Mutex
	handler := opencode.CollectHandler(&records, &mu)

	parser := opencode.NewStreamParserForTest(handler)
	reader := io.NopCloser(strings.NewReader(input))
	err := opencode.StreamParserParse(parser, context.Background(), reader)

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
		if r.EventType == message.ChunkError {
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

	var records []opencode.ChunkRecord
	var mu sync.Mutex
	handler := opencode.CollectHandler(&records, &mu)

	parser := opencode.NewStreamParserForTest(handler)
	reader := io.NopCloser(strings.NewReader(input))
	err := opencode.StreamParserParse(parser, context.Background(), reader)

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

	var records []opencode.ChunkRecord
	var mu sync.Mutex
	handler := opencode.CollectHandler(&records, &mu)

	parser := opencode.NewStreamParserForTest(handler)
	reader := io.NopCloser(strings.NewReader(input))
	err := opencode.StreamParserParse(parser, context.Background(), reader)

	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	if opencode.StreamParserCompleted(parser) {
		t.Error("tool-calls finish reason should NOT mark as completed")
	}

	mu.Lock()
	defer mu.Unlock()

	hasTransition := false
	for _, r := range records {
		if r.EventType == message.ChunkMetadata {
			var meta map[string]any
			json.Unmarshal([]byte(r.Data), &meta)
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

	var records []opencode.ChunkRecord
	var mu sync.Mutex
	handler := opencode.CollectHandler(&records, &mu)

	parser := opencode.NewStreamParserForTest(handler)
	reader := io.NopCloser(strings.NewReader(input))
	err := opencode.StreamParserParse(parser, context.Background(), reader)

	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	if !opencode.StreamParserCompleted(parser) {
		t.Error("multi-step stream should complete on final stop")
	}

	mu.Lock()
	defer mu.Unlock()

	var (
		completedCount int
		hasText        bool
	)
	for _, r := range records {
		if r.EventType == message.ChunkExecute {
			var props map[string]any
			json.Unmarshal([]byte(r.Data), &props)
			if props["status"] == "completed" {
				completedCount++
			}
		}
		if r.EventType == message.ChunkText && r.Data == "The output was: hi" {
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

	var records []opencode.ChunkRecord
	var mu sync.Mutex
	handler := opencode.CollectHandler(&records, &mu)

	parser := opencode.NewStreamParserForTest(handler)
	reader := io.NopCloser(strings.NewReader(input))
	err := opencode.StreamParserParse(parser, context.Background(), reader)

	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	mu.Lock()
	defer mu.Unlock()

	hasReasoning := false
	for _, r := range records {
		if r.EventType == message.ChunkMetadata {
			var meta map[string]any
			json.Unmarshal([]byte(r.Data), &meta)
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

	var records []opencode.ChunkRecord
	var mu sync.Mutex
	handler := opencode.CollectHandler(&records, &mu)

	parser := opencode.NewStreamParserForTest(handler)
	reader := io.NopCloser(strings.NewReader(input))
	err := opencode.StreamParserParse(parser, context.Background(), reader)

	if err != nil {
		t.Fatalf("unknown event type should not cause error: %v", err)
	}
	if !opencode.StreamParserCompleted(parser) {
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
		got := opencode.ExtractSummary(tc.tool, tc.input)
		if got != tc.want {
			t.Errorf("ExtractSummary(%q, %q) = %q, want %q", tc.tool, tc.input, got, tc.want)
		}
	}
}

func TestTruncate(t *testing.T) {
	short := "hello"
	if opencode.Truncate(short, 80) != "hello" {
		t.Error("short string should not be truncated")
	}

	long := strings.Repeat("a", 100)
	result := opencode.Truncate(long, 80)
	if len(result) != 83 { // 80 + "..."
		t.Errorf("truncated len = %d, want 83", len(result))
	}

	withNewlines := "line1\nline2\nline3"
	result = opencode.Truncate(withNewlines, 80)
	if strings.Contains(result, "\n") {
		t.Error("truncate should replace newlines with spaces")
	}
}
