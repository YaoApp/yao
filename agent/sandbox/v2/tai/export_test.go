package tai

import (
	"github.com/yaoapp/yao/agent/output/message"
)

// ExportStreamParser exposes the parser for unit tests.
type ExportStreamParser struct {
	p *streamParser
}

// NewExportStreamParser creates a parser with the given handler.
func NewExportStreamParser(handler message.StreamFunc) *ExportStreamParser {
	return &ExportStreamParser{p: newStreamParser(handler, "", "")}
}

// Dispatch feeds a single event into the parser and returns true if parsing should stop.
func (e *ExportStreamParser) Dispatch(ev map[string]any) bool {
	return e.p.dispatch(mapToEventLine(ev))
}

// Completed returns whether the parser received an agent_end event.
func (e *ExportStreamParser) Completed() bool {
	return e.p.completed
}

// CloseOrphanedTools exposes orphan cleanup for testing.
func (e *ExportStreamParser) CloseOrphanedTools() {
	e.p.closeOrphanedTools()
}

// ExportExtractToolSummary exposes the tool summary helper.
func ExportExtractToolSummary(name, input string) string {
	return extractToolSummary(name, input)
}

// ExportTruncateStr exposes the truncation helper.
func ExportTruncateStr(s string, max int) string {
	return truncateStr(s, max)
}

// mapToEventLine converts a generic map to eventLine for test convenience.
func mapToEventLine(m map[string]any) eventLine {
	var ev eventLine

	if v, ok := m["type"].(string); ok {
		ev.Type = v
	}
	if v, ok := m["event_type"].(string); ok {
		ev.EventType = v
	}
	if v, ok := m["delta"].(string); ok {
		ev.Delta = v
	}
	if v, ok := m["content"].(string); ok {
		ev.Content = v
	}
	if v, ok := m["content_index"].(int); ok {
		ev.ContentIndex = &v
	}
	if v, ok := m["tool"].(string); ok {
		ev.Tool = v
	}
	if v, ok := m["tool_id"].(string); ok {
		ev.ToolID = v
	}
	if v, ok := m["tool_label"].(string); ok {
		ev.ToolLabel = v
	}
	if v, ok := m["is_error"].(bool); ok {
		ev.IsError = v
	}
	if v, ok := m["duration_ms"].(int64); ok {
		ev.DurationMs = v
	}
	if v, ok := m["session_id"].(string); ok {
		ev.SessionID = v
	}
	if v, ok := m["session_name"].(string); ok {
		ev.SessionName = v
	}
	if v, ok := m["thinking_level"].(string); ok {
		ev.ThinkingLevel = v
	}
	if v, ok := m["error"].(string); ok {
		ev.Error = v
	}
	if v, ok := m["stop_reason"].(string); ok {
		ev.StopReason = v
	}
	if v, ok := m["reason"].(string); ok {
		ev.Reason = v
	}
	if v, ok := m["tokens_before"].(int); ok {
		ev.TokensBefore = v
	}
	if v, ok := m["tokens_after"].(int); ok {
		ev.TokensAfter = v
	}
	if v, ok := m["aborted"].(bool); ok {
		ev.Aborted = v
	}
	if v, ok := m["will_retry"].(bool); ok {
		ev.WillRetry = v
	}
	if v, ok := m["attempt"].(int); ok {
		ev.Attempt = v
	}
	if v, ok := m["max_attempts"].(int); ok {
		ev.MaxAttempts = v
	}
	if v, ok := m["delay_ms"].(int); ok {
		ev.DelayMs = v
	}
	if v, ok := m["source"].(string); ok {
		ev.Source = v
	}
	if v, ok := m["summary"].(string); ok {
		ev.Summary = v
	}
	if v, ok := m["final_error"].(string); ok {
		ev.FinalError = v
	}

	// Pointer fields.
	if v, ok := m["turn"].(int); ok {
		ev.Turn = &v
	}
	if v, ok := m["timestamp"].(int64); ok {
		ev.Timestamp = &v
	}
	if v, ok := m["success"].(bool); ok {
		ev.Success = &v
	}

	// Slice fields.
	if v, ok := m["steering"].([]string); ok {
		ev.Steering = v
	}
	if v, ok := m["follow_up"].([]string); ok {
		ev.FollowUp = v
	}

	// tool_call sub-object
	if tc, ok := m["tool_call"].(map[string]any); ok {
		tci := &toolCallInfo{}
		if v, ok := tc["id"].(string); ok {
			tci.ID = v
		}
		if v, ok := tc["name"].(string); ok {
			tci.Name = v
		}
		if v, ok := tc["arguments"].(string); ok {
			tci.Arguments = []byte(v)
		}
		ev.ToolCall = tci
	}

	// JSON fields passed as strings in test maps.
	if v, ok := m["args"].(string); ok {
		ev.Args = []byte(v)
	}
	if v, ok := m["result"].(string); ok {
		ev.Result = []byte(v)
	}
	if v, ok := m["partial_result"].(string); ok {
		ev.PartialResult = []byte(v)
	}
	if v, ok := m["usage"].(string); ok {
		ev.Usage = []byte(v)
	}
	if v, ok := m["compaction_usage"].(string); ok {
		ev.CompactionUsage = []byte(v)
	}
	if v, ok := m["entry"].(string); ok {
		ev.Entry = []byte(v)
	}

	return ev
}
