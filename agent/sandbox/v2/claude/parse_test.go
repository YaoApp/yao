package claude

import (
	"context"
	"encoding/json"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/yaoapp/yao/agent/output/message"
)

type chunkRecord struct {
	Type message.StreamChunkType
	Data json.RawMessage
}

func recordingHandler(out *[]chunkRecord) message.StreamFunc {
	return func(chunkType message.StreamChunkType, data []byte) int {
		cp := make([]byte, len(data))
		copy(cp, data)
		*out = append(*out, chunkRecord{Type: chunkType, Data: cp})
		return 0
	}
}

func stoppingHandler(stopAfter int) (message.StreamFunc, *[]chunkRecord) {
	var out []chunkRecord
	count := 0
	fn := func(chunkType message.StreamChunkType, data []byte) int {
		cp := make([]byte, len(data))
		copy(cp, data)
		out = append(out, chunkRecord{Type: chunkType, Data: cp})
		count++
		if count >= stopAfter {
			return 1
		}
		return 0
	}
	return fn, &out
}

func jsonLine(v interface{}) string {
	b, _ := json.Marshal(v)
	return string(b)
}

func pipeWithLines(lines ...string) io.ReadCloser {
	return io.NopCloser(strings.NewReader(strings.Join(lines, "\n") + "\n"))
}

// --- helper: extract message_id from ChunkMessageStart data ---
func extractMessageID(data json.RawMessage) string {
	var d map[string]any
	json.Unmarshal(data, &d)
	if id, ok := d["message_id"].(string); ok {
		return id
	}
	return ""
}

func TestParser_TextOnly(t *testing.T) {
	lines := []string{
		jsonLine(map[string]any{"type": "system", "session_id": "abc"}),
		jsonLine(map[string]any{
			"type": "stream_event",
			"event": map[string]any{
				"type":  "content_block_delta",
				"delta": map[string]any{"type": "text_delta", "text": "Hello "},
			},
		}),
		jsonLine(map[string]any{
			"type": "stream_event",
			"event": map[string]any{
				"type":  "content_block_delta",
				"delta": map[string]any{"type": "text_delta", "text": "world"},
			},
		}),
		jsonLine(map[string]any{
			"type": "assistant",
			"message": map[string]any{
				"stop_reason": "end_turn",
				"content":     []any{map[string]any{"type": "text", "text": "Hello world"}},
			},
		}),
		jsonLine(map[string]any{
			"type":           "result",
			"total_cost_usd": 0.001,
			"duration_ms":    1234,
			"num_turns":      1,
			"usage":          map[string]any{"input_tokens": 10, "output_tokens": 20},
		}),
	}

	var chunks []chunkRecord
	p := newStreamParser(recordingHandler(&chunks))
	err := p.parse(context.Background(), pipeWithLines(lines...))
	require.NoError(t, err)
	assert.True(t, p.completed)

	hasMessageStart := false
	hasText := false
	hasMessageEnd := false
	hasResultMeta := false
	for _, c := range chunks {
		switch c.Type {
		case message.ChunkMessageStart:
			hasMessageStart = true
		case message.ChunkText:
			hasText = true
		case message.ChunkMessageEnd:
			hasMessageEnd = true
		case message.ChunkMetadata:
			var meta map[string]any
			json.Unmarshal(c.Data, &meta)
			if _, ok := meta["result_summary"]; ok {
				hasResultMeta = true
			}
		}
	}
	assert.True(t, hasMessageStart, "should emit message_start")
	assert.True(t, hasText, "should emit text chunks")
	assert.True(t, hasMessageEnd, "should emit message_end")
	assert.True(t, hasResultMeta, "should emit result_summary metadata")
}

func TestParser_ToolUseAndResult(t *testing.T) {
	lines := []string{
		jsonLine(map[string]any{
			"type": "stream_event",
			"event": map[string]any{
				"type": "content_block_start",
				"content_block": map[string]any{
					"type": "tool_use",
					"name": "Bash",
					"id":   "tool_123",
				},
			},
		}),
		jsonLine(map[string]any{
			"type": "stream_event",
			"event": map[string]any{
				"type":  "content_block_delta",
				"delta": map[string]any{"type": "input_json_delta", "partial_json": `{"command":"ls`},
			},
		}),
		jsonLine(map[string]any{
			"type": "stream_event",
			"event": map[string]any{
				"type":  "content_block_delta",
				"delta": map[string]any{"type": "input_json_delta", "partial_json": `"}`},
			},
		}),
		jsonLine(map[string]any{
			"type":  "stream_event",
			"event": map[string]any{"type": "content_block_stop"},
		}),
		jsonLine(map[string]any{
			"type": "user",
			"message": map[string]any{
				"content": []any{
					map[string]any{
						"type":        "tool_result",
						"tool_use_id": "tool_123",
						"content":     "file1.txt\nfile2.txt",
						"is_error":    false,
					},
				},
			},
		}),
		jsonLine(map[string]any{
			"type":      "result",
			"num_turns": 1,
		}),
	}

	var chunks []chunkRecord
	p := newStreamParser(recordingHandler(&chunks))
	err := p.parse(context.Background(), pipeWithLines(lines...))
	require.NoError(t, err)
	assert.True(t, p.completed)

	var execChunks []map[string]any
	for _, c := range chunks {
		if c.Type == message.ChunkExecute {
			var data map[string]any
			json.Unmarshal(c.Data, &data)
			execChunks = append(execChunks, data)
		}
	}
	require.GreaterOrEqual(t, len(execChunks), 2, "should have at least 2 execute chunks (start + result)")

	assert.Equal(t, "Bash", execChunks[0]["tool"])
	assert.Equal(t, "tool_123", execChunks[0]["tool_id"])
	assert.Equal(t, "running", execChunks[0]["status"])

	lastExec := execChunks[len(execChunks)-1]
	assert.Equal(t, "tool_123", lastExec["tool_id"])
	assert.Equal(t, "completed", lastExec["status"])
	assert.Equal(t, "Bash", lastExec["tool"], "tool_result should carry tool name")
}

// TestParser_ToolIndependentMessages verifies that each tool call gets its own
// message_start/message_end pair, and tool_result reuses the same message_id.
func TestParser_ToolIndependentMessages(t *testing.T) {
	lines := []string{
		// Tool 1: Write
		jsonLine(map[string]any{
			"type": "stream_event",
			"event": map[string]any{
				"type":          "content_block_start",
				"content_block": map[string]any{"type": "tool_use", "name": "Write", "id": "t_write"},
			},
		}),
		jsonLine(map[string]any{
			"type": "stream_event",
			"event": map[string]any{
				"type":  "content_block_delta",
				"delta": map[string]any{"type": "input_json_delta", "partial_json": `{"file_path":"server.js"}`},
			},
		}),
		jsonLine(map[string]any{
			"type":  "stream_event",
			"event": map[string]any{"type": "content_block_stop"},
		}),
		// Tool 2: Bash
		jsonLine(map[string]any{
			"type": "stream_event",
			"event": map[string]any{
				"type":          "content_block_start",
				"content_block": map[string]any{"type": "tool_use", "name": "Bash", "id": "t_bash"},
			},
		}),
		jsonLine(map[string]any{
			"type":  "stream_event",
			"event": map[string]any{"type": "content_block_stop"},
		}),
		// Results
		jsonLine(map[string]any{
			"type": "user",
			"message": map[string]any{
				"content": []any{
					map[string]any{"type": "tool_result", "tool_use_id": "t_write", "content": "ok"},
					map[string]any{"type": "tool_result", "tool_use_id": "t_bash", "content": "done"},
				},
			},
		}),
		jsonLine(map[string]any{"type": "result", "num_turns": 1}),
	}

	var chunks []chunkRecord
	p := newStreamParser(recordingHandler(&chunks))
	err := p.parse(context.Background(), pipeWithLines(lines...))
	require.NoError(t, err)

	// Collect all message_start IDs and their order
	var msgStarts []string
	for _, c := range chunks {
		if c.Type == message.ChunkMessageStart {
			msgStarts = append(msgStarts, extractMessageID(c.Data))
		}
	}

	// Should have 4 message_starts: Write(running), Bash(running), Write(result), Bash(result)
	require.Equal(t, 4, len(msgStarts), "should have 4 message_start events")

	writeMsgID := msgStarts[0]
	bashMsgID := msgStarts[1]
	assert.NotEqual(t, writeMsgID, bashMsgID, "Write and Bash should have different message_ids")

	// tool_result should reuse the original message_id
	assert.Equal(t, writeMsgID, msgStarts[2], "Write tool_result should reuse Write message_id")
	assert.Equal(t, bashMsgID, msgStarts[3], "Bash tool_result should reuse Bash message_id")

	// Count message_end events (should match message_start)
	endCount := 0
	for _, c := range chunks {
		if c.Type == message.ChunkMessageEnd {
			endCount++
		}
	}
	assert.Equal(t, 4, endCount, "each message_start should have matching message_end")
}

// TestParser_ToolSummaryExtraction verifies that summary is extracted from tool input.
func TestParser_ToolSummaryExtraction(t *testing.T) {
	lines := []string{
		jsonLine(map[string]any{
			"type": "stream_event",
			"event": map[string]any{
				"type":          "content_block_start",
				"content_block": map[string]any{"type": "tool_use", "name": "Bash", "id": "t1"},
			},
		}),
		jsonLine(map[string]any{
			"type": "stream_event",
			"event": map[string]any{
				"type":  "content_block_delta",
				"delta": map[string]any{"type": "input_json_delta", "partial_json": `{"command":"ls -la /workspace"}`},
			},
		}),
		jsonLine(map[string]any{
			"type":  "stream_event",
			"event": map[string]any{"type": "content_block_stop"},
		}),
		jsonLine(map[string]any{"type": "result", "num_turns": 1}),
	}

	var chunks []chunkRecord
	p := newStreamParser(recordingHandler(&chunks))
	err := p.parse(context.Background(), pipeWithLines(lines...))
	require.NoError(t, err)

	// Look for summary in the last execute chunk before message_end
	var summaryFound bool
	for _, c := range chunks {
		if c.Type == message.ChunkExecute {
			var data map[string]any
			json.Unmarshal(c.Data, &data)
			if s, ok := data["summary"].(string); ok && s != "" {
				summaryFound = true
				assert.Equal(t, "ls -la /workspace", s)
			}
		}
	}
	assert.True(t, summaryFound, "should emit summary from tool input")
}

func TestParser_UsageMetadata(t *testing.T) {
	lines := []string{
		jsonLine(map[string]any{
			"type": "assistant",
			"message": map[string]any{
				"usage": map[string]any{
					"input_tokens":  100,
					"output_tokens": 50,
				},
				"stop_reason": "end_turn",
				"content":     []any{},
			},
		}),
		jsonLine(map[string]any{"type": "result", "num_turns": 1}),
	}

	var chunks []chunkRecord
	p := newStreamParser(recordingHandler(&chunks))
	err := p.parse(context.Background(), pipeWithLines(lines...))
	require.NoError(t, err)

	var usageMeta map[string]any
	for _, c := range chunks {
		if c.Type == message.ChunkMetadata {
			var meta map[string]any
			json.Unmarshal(c.Data, &meta)
			if u, ok := meta["usage"]; ok {
				usageMeta, _ = u.(map[string]any)
			}
		}
	}
	require.NotNil(t, usageMeta, "should emit usage metadata")
	assert.Equal(t, float64(100), usageMeta["input_tokens"])
	assert.Equal(t, float64(50), usageMeta["output_tokens"])
}

func TestParser_ResultSummary(t *testing.T) {
	lines := []string{
		jsonLine(map[string]any{
			"type":           "result",
			"total_cost_usd": 0.05,
			"duration_ms":    5000,
			"num_turns":      3,
			"usage":          map[string]any{"input_tokens": 500, "output_tokens": 200},
		}),
	}

	var chunks []chunkRecord
	p := newStreamParser(recordingHandler(&chunks))
	err := p.parse(context.Background(), pipeWithLines(lines...))
	require.NoError(t, err)
	assert.True(t, p.completed)

	var summary map[string]any
	for _, c := range chunks {
		if c.Type == message.ChunkMetadata {
			var meta map[string]any
			json.Unmarshal(c.Data, &meta)
			if s, ok := meta["result_summary"]; ok {
				summary, _ = s.(map[string]any)
			}
		}
	}
	require.NotNil(t, summary, "should emit result_summary")
	assert.Equal(t, float64(0.05), summary["total_cost_usd"])
	assert.Equal(t, float64(5000), summary["duration_ms"])
	assert.Equal(t, float64(3), summary["num_turns"])
}

func TestParser_ErrorMessage(t *testing.T) {
	lines := []string{
		jsonLine(map[string]any{
			"type":  "error",
			"error": map[string]any{"message": "rate limit exceeded"},
		}),
	}

	var chunks []chunkRecord
	p := newStreamParser(recordingHandler(&chunks))
	err := p.parse(context.Background(), pipeWithLines(lines...))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "rate limit exceeded")
	assert.False(t, p.completed)

	hasError := false
	for _, c := range chunks {
		if c.Type == message.ChunkError {
			hasError = true
			assert.Contains(t, string(c.Data), "rate limit exceeded")
		}
	}
	assert.True(t, hasError, "should emit error chunk")
}

func TestParser_ResultIsError(t *testing.T) {
	lines := []string{
		jsonLine(map[string]any{
			"type":     "result",
			"is_error": true,
			"result":   "authentication failed",
		}),
	}

	var chunks []chunkRecord
	p := newStreamParser(recordingHandler(&chunks))
	err := p.parse(context.Background(), pipeWithLines(lines...))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "authentication failed")
}

func TestParser_ContextCancel(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	r, w := io.Pipe()

	go func() {
		time.Sleep(50 * time.Millisecond)
		cancel()
	}()

	p := newStreamParser(nil)
	err := p.parse(ctx, r)
	_ = w.Close()

	require.Error(t, err)
	assert.ErrorIs(t, err, context.Canceled)
}

func TestParser_HandlerStopsStream(t *testing.T) {
	lines := []string{
		jsonLine(map[string]any{
			"type": "stream_event",
			"event": map[string]any{
				"type":  "content_block_delta",
				"delta": map[string]any{"type": "text_delta", "text": "first"},
			},
		}),
		jsonLine(map[string]any{
			"type": "stream_event",
			"event": map[string]any{
				"type":  "content_block_delta",
				"delta": map[string]any{"type": "text_delta", "text": "second"},
			},
		}),
	}

	handler, chunks := stoppingHandler(2)
	p := newStreamParser(handler)
	err := p.parse(context.Background(), pipeWithLines(lines...))
	require.NoError(t, err)

	assert.LessOrEqual(t, len(*chunks), 3, "should stop early")
}

func TestParser_EmptyAndInvalidLines(t *testing.T) {
	lines := []string{
		"",
		"not json at all",
		"   ",
		`{"type": "result", "num_turns": 1}`,
	}

	var chunks []chunkRecord
	p := newStreamParser(recordingHandler(&chunks))
	err := p.parse(context.Background(), pipeWithLines(lines...))
	require.NoError(t, err)
	assert.True(t, p.completed, "should complete despite invalid lines")
}

func TestParser_MultiTurnConversation(t *testing.T) {
	lines := []string{
		jsonLine(map[string]any{
			"type": "stream_event",
			"event": map[string]any{
				"type":          "content_block_start",
				"content_block": map[string]any{"type": "tool_use", "name": "Read", "id": "t1"},
			},
		}),
		jsonLine(map[string]any{
			"type":  "stream_event",
			"event": map[string]any{"type": "content_block_stop"},
		}),
		jsonLine(map[string]any{
			"type": "assistant",
			"message": map[string]any{
				"stop_reason": "tool_use",
				"content": []any{
					map[string]any{"type": "tool_use", "name": "Read", "id": "t1", "input": map[string]any{"path": "/tmp"}},
				},
			},
		}),
		jsonLine(map[string]any{
			"type": "user",
			"message": map[string]any{
				"content": []any{
					map[string]any{"type": "tool_result", "tool_use_id": "t1", "content": "ok"},
				},
			},
		}),
		jsonLine(map[string]any{
			"type": "stream_event",
			"event": map[string]any{
				"type":  "content_block_delta",
				"delta": map[string]any{"type": "text_delta", "text": "Done reading"},
			},
		}),
		jsonLine(map[string]any{
			"type": "assistant",
			"message": map[string]any{
				"stop_reason": "end_turn",
				"content":     []any{map[string]any{"type": "text", "text": "Done reading"}},
			},
		}),
		jsonLine(map[string]any{"type": "result", "num_turns": 2}),
	}

	var chunks []chunkRecord
	p := newStreamParser(recordingHandler(&chunks))
	err := p.parse(context.Background(), pipeWithLines(lines...))
	require.NoError(t, err)
	assert.True(t, p.completed)

	startCount := 0
	endCount := 0
	for _, c := range chunks {
		switch c.Type {
		case message.ChunkMessageStart:
			startCount++
		case message.ChunkMessageEnd:
			endCount++
		}
	}
	// streaming tool_use(start) + streaming tool_use(stop) + assistant tool_use + tool_result + text = 5 starts
	assert.GreaterOrEqual(t, startCount, 3, "should have multiple message starts for multi-turn")
	assert.Equal(t, startCount, endCount, "each message_start should have matching message_end")
}

func TestParser_ErrorStringFormat(t *testing.T) {
	lines := []string{
		jsonLine(map[string]any{
			"type":  "error",
			"error": "simple string error",
		}),
	}

	var chunks []chunkRecord
	p := newStreamParser(recordingHandler(&chunks))
	err := p.parse(context.Background(), pipeWithLines(lines...))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "simple string error")
}

func TestParser_InputDeltaAccumulation(t *testing.T) {
	lines := []string{
		jsonLine(map[string]any{
			"type": "stream_event",
			"event": map[string]any{
				"type":          "content_block_start",
				"content_block": map[string]any{"type": "tool_use", "name": "Bash", "id": "t1"},
			},
		}),
		jsonLine(map[string]any{
			"type": "stream_event",
			"event": map[string]any{
				"type":  "content_block_delta",
				"delta": map[string]any{"type": "input_json_delta", "partial_json": `{"com`},
			},
		}),
		jsonLine(map[string]any{
			"type": "stream_event",
			"event": map[string]any{
				"type":  "content_block_delta",
				"delta": map[string]any{"type": "input_json_delta", "partial_json": `mand":"ls"}`},
			},
		}),
		jsonLine(map[string]any{
			"type":  "stream_event",
			"event": map[string]any{"type": "content_block_stop"},
		}),
		jsonLine(map[string]any{"type": "result", "num_turns": 1}),
	}

	var chunks []chunkRecord
	p := newStreamParser(recordingHandler(&chunks))
	err := p.parse(context.Background(), pipeWithLines(lines...))
	require.NoError(t, err)

	// The last input_delta chunk should contain the full accumulated input
	var lastDelta string
	for _, c := range chunks {
		if c.Type == message.ChunkExecute {
			var data map[string]any
			json.Unmarshal(c.Data, &data)
			if d, ok := data["input_delta"].(string); ok {
				lastDelta = d
			}
		}
	}
	assert.Equal(t, `{"command":"ls"}`, lastDelta, "input_delta should accumulate all fragments")
}
