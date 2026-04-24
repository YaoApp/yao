package opencode

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/yaoapp/kun/log"
	"github.com/yaoapp/yao/agent/output/message"
)

// streamParser handles OpenCode's JSONL output (--format json).
//
// OpenCode emits one JSON object per line with a "type" field. Unlike Claude's
// streaming events which provide incremental deltas, OpenCode events arrive at
// completion boundaries:
//
//	step_start   – agent step begins (contains sessionID, part metadata)
//	text         – completed text block (full text, only emitted when part.time.end is set)
//	tool_use     – completed tool call (status: completed|error, contains full input/output)
//	step_finish  – step ended (reason: stop|tool-calls; includes token/cost info)
//	reasoning    – thinking/reasoning output
//	error        – error event
//
// Because tool_use events only arrive after the tool finishes, the parser
// emits a "running" execute chunk at step_start so the frontend has immediate
// feedback that work is in progress. It also tracks intermediate tool_use
// events (status=completed) so they are shown as soon as they arrive, even
// before step_finish.
type streamParser struct {
	handler   message.StreamFunc
	completed bool
	toolIndex int

	// textActive tracks whether a text message group is currently open.
	textActive bool
	textMsgID  string

	// pendingExec tracks tools that were announced at step_start but haven't
	// received their completed tool_use event yet. Key = step ID.
	pendingExec map[string]string // stepID -> msgID
}

func newStreamParser(handler message.StreamFunc) *streamParser {
	return &streamParser{
		handler:     handler,
		pendingExec: make(map[string]string),
	}
}

func (p *streamParser) parse(ctx context.Context, stdout io.ReadCloser) error {
	doneParsing := make(chan struct{})
	defer close(doneParsing)
	go func() {
		select {
		case <-ctx.Done():
			stdout.Close()
		case <-doneParsing:
		}
	}()

	scanner := bufio.NewScanner(stdout)
	scanner.Buffer(make([]byte, 0, 64*1024), 10*1024*1024)

	startTime := time.Now()
	lineCount := 0
	lastHeartbeat := time.Now()

	log.Trace("[opencode-parse] stream started")

	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}
		lineCount++

		if time.Since(lastHeartbeat) > 30*time.Second {
			log.Trace("[opencode-parse] heartbeat: lines=%d elapsed=%v",
				lineCount, time.Since(startTime).Round(time.Second))
			lastHeartbeat = time.Now()
		}

		var msg map[string]any
		if err := json.Unmarshal([]byte(line), &msg); err != nil {
			if len(line) > 200 {
				log.Trace("[opencode-parse] JSON unmarshal error: %v (line len=%d)", err, len(line))
			} else {
				log.Trace("[opencode-parse] JSON unmarshal error: %v (line=%q)", err, line)
			}
			continue
		}

		msgType, _ := msg["type"].(string)

		switch msgType {
		case "step_start":
			if p.handleStepStart(msg) {
				return nil
			}
		case "text":
			if p.handleText(msg) {
				return nil
			}
		case "tool_use":
			if p.handleToolUse(msg) {
				return nil
			}
		case "step_finish":
			if err := p.handleStepFinish(msg); err != nil {
				return err
			}
			if p.completed {
				log.Trace("[opencode-parse] stream completed: lines=%d elapsed=%v",
					lineCount, time.Since(startTime).Round(time.Second))
				return nil
			}
			log.Trace("[opencode-parse] step_finish (intermediate, reason!=stop): continuing parse loop")
		case "reasoning":
			p.handleReasoning(msg)
		case "error":
			log.Trace("[opencode-parse] error event: lines=%d elapsed=%v",
				lineCount, time.Since(startTime).Round(time.Second))
			return p.handleError(msg)
		default:
			log.Trace("[opencode-parse] unknown event type: %s", msgType)
		}
	}

	log.Trace("[opencode-parse] stream ended: lines=%d elapsed=%v completed=%v scanErr=%v",
		lineCount, time.Since(startTime).Round(time.Second), p.completed, scanner.Err())

	if err := scanner.Err(); err != nil {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		return err
	}
	return nil
}

// --- Message lifecycle helpers (mirroring Claude parser) ---

func (p *streamParser) beginMessageWithID(id, msgType string) (stopped bool) {
	startData := message.EventMessageStartData{
		MessageID: id,
		Type:      msgType,
		Timestamp: time.Now().UnixMilli(),
	}
	sd, _ := json.Marshal(startData)
	return p.handler != nil && p.handler(message.ChunkMessageStart, sd) != 0
}

func (p *streamParser) beginMessage(msgType string) (messageID string, stopped bool) {
	id := fmt.Sprintf("sandbox-%s-%s", msgType, message.GenerateNanoID())
	return id, p.beginMessageWithID(id, msgType)
}

func (p *streamParser) endMessage() {
	if p.handler != nil {
		p.handler(message.ChunkMessageEnd, nil)
	}
}

func (p *streamParser) closeTextMessage() {
	if p.textActive {
		p.endMessage()
		p.textActive = false
		p.textMsgID = ""
	}
}

func (p *streamParser) ensureTextMessage() (stopped bool) {
	if !p.textActive {
		id, stopped := p.beginMessage("text")
		if stopped {
			return true
		}
		p.textActive = true
		p.textMsgID = id
	}
	return false
}

func (p *streamParser) emitText(text string) (stopped bool) {
	text = strings.ReplaceAll(text, "\r\n", "\n")
	text = strings.ReplaceAll(text, "\r", "\n")
	return p.handler != nil && p.handler(message.ChunkText, []byte(text)) != 0
}

func (p *streamParser) emitExecute(props map[string]any) (stopped bool) {
	data, _ := json.Marshal(props)
	return p.handler != nil && p.handler(message.ChunkExecute, data) != 0
}

func (p *streamParser) emitMetadata(data map[string]any) {
	if p.handler == nil {
		return
	}
	encoded, _ := json.Marshal(data)
	p.handler(message.ChunkMetadata, encoded)
}

// --- Event handlers ---

func (p *streamParser) handleStepStart(msg map[string]any) (stopped bool) {
	if p.handler == nil {
		return false
	}

	sessionID, _ := msg["sessionID"].(string)
	meta := map[string]any{"opencode_session_id": sessionID}

	part, _ := msg["part"].(map[string]any)
	if part != nil {
		if id, ok := part["id"].(string); ok {
			meta["step_id"] = id
		}
	}

	p.emitMetadata(meta)

	// Record the step ID so that a subsequent tool_use can be correlated.
	// We do NOT emit a "running" execute here because we can't tell yet
	// whether the step will involve tool calls or just text output. For
	// pure text responses an empty execute widget would look wrong. The
	// "running" indicator is instead emitted lazily in handleToolUse when
	// the first tool event actually arrives.
	if part != nil {
		if stepID, ok := part["id"].(string); ok && stepID != "" {
			p.pendingExec[stepID] = "" // placeholder — msgID assigned later
		}
	}
	return false
}

func (p *streamParser) handleText(msg map[string]any) (stopped bool) {
	if p.handler == nil {
		return false
	}

	part, _ := msg["part"].(map[string]any)
	if part == nil {
		return false
	}
	content, _ := part["text"].(string)
	if content == "" {
		content, _ = part["content"].(string)
	}
	if content == "" {
		return false
	}

	// Close any pending execute message before text output.
	p.closePendingExec(msg)

	if p.ensureTextMessage() {
		return true
	}
	if p.emitText(content) {
		p.closeTextMessage()
		return true
	}
	p.closeTextMessage()

	return false
}

func (p *streamParser) handleToolUse(msg map[string]any) (stopped bool) {
	if p.handler == nil {
		return false
	}

	part, _ := msg["part"].(map[string]any)
	if part == nil {
		return false
	}

	state, _ := part["state"].(map[string]any)
	if state == nil {
		return false
	}

	toolName, _ := part["toolName"].(string)
	if toolName == "" {
		toolName, _ = part["tool"].(string)
	}
	toolID, _ := part["toolCallId"].(string)
	if toolID == "" {
		toolID = fmt.Sprintf("oc-tool_%d_%d", p.toolIndex, time.Now().UnixNano())
	}
	p.toolIndex++

	status, _ := state["status"].(string)
	isError := status == "error"

	p.closeTextMessage()
	// Clear any pending step placeholder (no msgID to reuse since we didn't
	// emit anything at step_start).
	p.closePendingExec(msg)

	// Build execute properties.
	execProps := map[string]any{
		"tool":    toolName,
		"tool_id": toolID,
		"status":  status,
		"runner":  "opencode-cli",
	}
	if isError {
		execProps["is_error"] = true
	}

	var inputStr string
	if input, ok := state["input"].(string); ok && input != "" {
		inputStr = input
	} else if inputObj, ok := state["input"].(map[string]any); ok {
		inputJSON, _ := json.Marshal(inputObj)
		inputStr = string(inputJSON)
	}
	if inputStr != "" {
		execProps["input"] = json.RawMessage(inputStr)
		summary := extractSummary(toolName, inputStr)
		if summary != "" {
			execProps["summary"] = summary
		}
	}

	if output, ok := state["output"].(string); ok && output != "" {
		execProps["output"] = output
	} else if outputObj := state["output"]; outputObj != nil {
		execProps["output"] = outputObj
	}

	// Single message group with the complete tool result.
	if _, stopped := p.beginMessage("execute"); stopped {
		return true
	}
	if p.emitExecute(execProps) {
		p.endMessage()
		return true
	}
	p.endMessage()
	return false
}

func (p *streamParser) handleStepFinish(msg map[string]any) error {
	p.closeTextMessage()
	p.closePendingExec(msg)

	part, _ := msg["part"].(map[string]any)
	reason := ""
	if part != nil {
		reason, _ = part["reason"].(string)
		if reason == "" {
			reason, _ = part["finishReason"].(string)
		}
	}

	if reason == "stop" || reason == "end_turn" {
		p.completed = true

		if p.handler != nil {
			finishMeta := map[string]any{
				"result_summary": map[string]any{
					"finish_reason": reason,
				},
			}
			// Include token/cost info if available.
			if part != nil {
				if tokens, ok := part["tokens"].(map[string]any); ok {
					finishMeta["result_summary"].(map[string]any)["tokens"] = tokens
				}
				if cost, ok := part["cost"]; ok {
					finishMeta["result_summary"].(map[string]any)["cost"] = cost
				}
			}
			p.emitMetadata(finishMeta)
		}
		return nil
	}

	// reason == "tool-calls" or other intermediate reasons: not final.
	// Emit metadata so the frontend knows a new round is starting.
	if p.handler != nil && reason != "" {
		p.emitMetadata(map[string]any{
			"step_transition": map[string]any{
				"reason": reason,
			},
		})
	}
	return nil
}

func (p *streamParser) handleReasoning(msg map[string]any) {
	if p.handler == nil {
		return
	}

	part, _ := msg["part"].(map[string]any)
	if part == nil {
		return
	}
	content, _ := part["text"].(string)
	if content == "" {
		content, _ = part["content"].(string)
	}
	if content == "" {
		return
	}

	p.emitMetadata(map[string]any{
		"reasoning": content,
	})
}

func (p *streamParser) handleError(msg map[string]any) error {
	p.closePendingExec(msg)

	var errMsg string

	if part, ok := msg["part"].(map[string]any); ok {
		errMsg, _ = part["error"].(string)
		if errMsg == "" {
			errMsg, _ = part["message"].(string)
		}
	}
	if errMsg == "" {
		switch e := msg["error"].(type) {
		case string:
			errMsg = e
		case map[string]any:
			errMsg, _ = e["message"].(string)
			if errMsg == "" {
				if data, ok := e["data"].(map[string]any); ok {
					errMsg, _ = data["message"].(string)
				}
			}
			if errMsg == "" {
				name, _ := e["name"].(string)
				if name != "" {
					errMsg = name
				}
			}
		}
	}
	if errMsg == "" {
		errMsg = "unknown OpenCode error"
	}

	if p.handler != nil {
		p.handler(message.ChunkError, []byte(errMsg))
	}
	return fmt.Errorf("OpenCode CLI error: %s", errMsg)
}

// --- Pending exec helpers ---

// closePendingExec clears step placeholders recorded at step_start.
func (p *streamParser) closePendingExec(msg map[string]any) {
	if len(p.pendingExec) == 0 {
		return
	}
	// Best-effort: clear matching step or all if we can't match.
	part, _ := msg["part"].(map[string]any)
	if part != nil {
		for _, key := range []string{"id", "messageID"} {
			if id, _ := part[key].(string); id != "" {
				if _, ok := p.pendingExec[id]; ok {
					delete(p.pendingExec, id)
					return
				}
			}
		}
	}
	// Fallback: clear the single pending entry (most common).
	if len(p.pendingExec) == 1 {
		for k := range p.pendingExec {
			delete(p.pendingExec, k)
		}
	}
}

// --- Utility ---

// extractSummary builds a short human-readable summary from the tool input.
func extractSummary(toolName string, inputJSON string) string {
	if inputJSON == "" {
		return ""
	}
	var obj map[string]any
	if err := json.Unmarshal([]byte(inputJSON), &obj); err != nil {
		return ""
	}

	switch strings.ToLower(toolName) {
	case "bash", "execute":
		if cmd, ok := obj["command"].(string); ok {
			return truncate(cmd, 80)
		}
	case "write", "create":
		if fp, ok := obj["file_path"].(string); ok {
			return fp
		}
	case "read":
		if fp, ok := obj["file_path"].(string); ok {
			return fp
		}
	case "edit":
		if fp, ok := obj["file_path"].(string); ok {
			return fp
		}
	}

	for _, key := range []string{"path", "file_path", "command", "url", "query"} {
		if v, ok := obj[key].(string); ok {
			return truncate(v, 80)
		}
	}
	return ""
}

func truncate(s string, max int) string {
	s = strings.TrimSpace(s)
	s = strings.ReplaceAll(s, "\n", " ")
	if len(s) > max {
		return s[:max] + "..."
	}
	return s
}
