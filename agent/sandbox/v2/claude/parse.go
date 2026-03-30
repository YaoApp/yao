package claude

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

// streamParser is an explicit state machine for Claude CLI stream-json output.
//
// Each tool call gets its own message lifecycle:
//
//	content_block_start  -> message_start(id=exec-N-xxx)  + ChunkExecute{tool, status:running}
//	input_json_delta     -> ChunkExecute{input_delta:...}  (same message group)
//	content_block_stop   -> message_end(exec-N-xxx)
//	...later...
//	user/tool_result     -> message_start(id=exec-N-xxx, reuse!) + ChunkExecute{status:completed, output:...} + message_end
type streamParser struct {
	handler   message.StreamFunc
	completed bool

	textActive bool
	toolIndex  int
	curTool    *toolState

	toolNames     map[string]string // tool_id -> tool_name
	toolMsgIDs    map[string]string // tool_id -> message_id (for result reuse)
	toolInputs    map[string]string // tool_id -> full input JSON (for result replay)
	toolSummaries map[string]string // tool_id -> summary (for result replay)
}

type toolState struct {
	id        string
	name      string
	msgID     string
	index     int
	inputJSON strings.Builder
}

func newStreamParser(handler message.StreamFunc) *streamParser {
	return &streamParser{
		handler:       handler,
		toolNames:     make(map[string]string),
		toolMsgIDs:    make(map[string]string),
		toolInputs:    make(map[string]string),
		toolSummaries: make(map[string]string),
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
	lastEventType := ""

	log.Trace("[claude-parse] stream started")

	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}
		lineCount++

		if time.Since(lastHeartbeat) > 30*time.Second {
			builderLen := 0
			if p.curTool != nil {
				builderLen = p.curTool.inputJSON.Len()
			}
			log.Trace("[claude-parse] heartbeat: lines=%d elapsed=%v lastEvent=%s toolBuilderLen=%d",
				lineCount, time.Since(startTime).Round(time.Second), lastEventType, builderLen)
			lastHeartbeat = time.Now()
		}

		var msg map[string]any
		if err := json.Unmarshal([]byte(line), &msg); err != nil {
			if len(line) > 200 {
				log.Trace("[claude-parse] JSON unmarshal error: %v (line len=%d, prefix=%q)", err, len(line), line[:200])
			} else {
				log.Trace("[claude-parse] JSON unmarshal error: %v (line=%q)", err, line)
			}
			continue
		}

		msgType, _ := msg["type"].(string)
		lastEventType = msgType
		var stopped bool

		switch msgType {
		case "system":
			stopped = p.handleSystem(msg)
		case "stream_event":
			stopped = p.handleStreamEvent(msg)
		case "assistant":
			stopped = p.handleAssistant(msg)
		case "user":
			stopped = p.handleUser(msg)
		case "result":
			log.Trace("[claude-parse] stream ended: lines=%d elapsed=%v completed=true", lineCount, time.Since(startTime).Round(time.Second))
			return p.handleResult(msg)
		case "error":
			log.Trace("[claude-parse] stream ended with error: lines=%d elapsed=%v", lineCount, time.Since(startTime).Round(time.Second))
			return p.handleError(msg)
		}

		if stopped {
			log.Trace("[claude-parse] stream stopped by handler: lines=%d elapsed=%v", lineCount, time.Since(startTime).Round(time.Second))
			return nil
		}
	}

	log.Trace("[claude-parse] stream ended: lines=%d elapsed=%v completed=%v scanErr=%v",
		lineCount, time.Since(startTime).Round(time.Second), p.completed, scanner.Err())

	if err := scanner.Err(); err != nil {
		log.Trace("[claude-parse] scanner error: %v (ctx.Err=%v)", err, ctx.Err())
		if ctx.Err() != nil {
			return ctx.Err()
		}
		return err
	}
	return nil
}

// --- Message lifecycle helpers ---

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
	}
}

// closeCurrentTool closes the in-flight streaming tool message (if any),
// flushing its accumulated input and emitting message_end. This must be
// called before opening a new message group so that the downstream handler
// never sees interleaved message_start/message_end pairs.
func (p *streamParser) closeCurrentTool() {
	if p.curTool == nil {
		return
	}
	toolID := p.curTool.id
	inputStr := p.curTool.inputJSON.String()
	if inputStr != "" {
		p.toolInputs[toolID] = inputStr
		summary := extractSummary(p.curTool.name, inputStr)
		if summary != "" {
			p.toolSummaries[toolID] = summary
			p.emitExecute(map[string]any{
				"summary": summary,
			})
		}
	}
	p.endMessage()
	p.curTool = nil
}

func (p *streamParser) ensureTextMessage() (stopped bool) {
	if !p.textActive {
		_, stopped = p.beginMessage("text")
		if stopped {
			return true
		}
		p.textActive = true
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

// extractSummary builds a short human-readable summary from the tool input JSON.
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

	// Fallback: try common field names
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

// --- Event handlers ---

func (p *streamParser) handleSystem(msg map[string]any) (stopped bool) {
	if p.handler != nil {
		data, _ := json.Marshal(msg)
		return p.handler(message.ChunkMetadata, data) != 0
	}
	return false
}

func (p *streamParser) handleStreamEvent(msg map[string]any) (stopped bool) {
	event, _ := msg["event"].(map[string]any)
	if event == nil {
		return false
	}
	eventType, _ := event["type"].(string)

	switch eventType {
	case "content_block_start":
		return p.onContentBlockStart(event)
	case "content_block_delta":
		return p.onContentBlockDelta(event)
	case "content_block_stop":
		return p.onContentBlockStop()
	}
	return false
}

func (p *streamParser) onContentBlockStart(event map[string]any) (stopped bool) {
	cb, ok := event["content_block"].(map[string]any)
	if !ok {
		return false
	}
	blockType, _ := cb["type"].(string)
	if blockType != "tool_use" {
		return false
	}

	p.closeTextMessage()

	toolName, _ := cb["name"].(string)
	toolID, _ := cb["id"].(string)
	if toolID == "" {
		toolID = fmt.Sprintf("tool_%d_%d", p.toolIndex, time.Now().UnixNano())
	}

	msgID, stopped := p.beginMessage("execute")
	if stopped {
		return true
	}

	p.curTool = &toolState{id: toolID, name: toolName, msgID: msgID, index: p.toolIndex}
	p.toolIndex++
	p.toolNames[toolID] = toolName
	p.toolMsgIDs[toolID] = msgID

	if p.handler == nil {
		return false
	}

	return p.emitExecute(map[string]any{
		"tool":    toolName,
		"tool_id": toolID,
		"status":  "running",
		"runner":  "claude-cli",
	})
}

func (p *streamParser) onContentBlockStop() (stopped bool) {
	p.closeCurrentTool()
	return false
}

func (p *streamParser) onContentBlockDelta(event map[string]any) (stopped bool) {
	delta, ok := event["delta"].(map[string]any)
	if !ok {
		return false
	}

	deltaType, _ := delta["type"].(string)

	switch deltaType {
	case "text_delta":
		text, _ := delta["text"].(string)
		if text == "" {
			return false
		}
		// If there is no active text message and this delta is only
		// whitespace, buffer it instead of opening a brand-new message
		// group just for spaces/indentation between tool calls.
		if !p.textActive && strings.TrimSpace(text) == "" {
			return false
		}
		if p.ensureTextMessage() {
			return true
		}
		return p.emitText(text)

	case "input_json_delta":
		if p.curTool == nil {
			return false
		}
		partial, _ := delta["partial_json"].(string)
		if partial == "" {
			return false
		}
		p.curTool.inputJSON.WriteString(partial)
		builderLen := p.curTool.inputJSON.Len()
		if builderLen > 0 && builderLen%100000 < len(partial) {
			log.Trace("[claude-parse] WARN: tool %s inputJSON growing: %d bytes", p.curTool.name, builderLen)
		}
		if p.handler != nil {
			return p.emitExecute(map[string]any{
				"input_delta": p.curTool.inputJSON.String(),
			})
		}
	}
	return false
}

func (p *streamParser) handleAssistant(msg map[string]any) (stopped bool) {
	msgData, _ := msg["message"].(map[string]any)
	if msgData == nil {
		return false
	}

	if usage, ok := msgData["usage"].(map[string]any); ok {
		p.emitMetadata(map[string]any{
			"usage": usage,
		})
	}

	stopReason, _ := msgData["stop_reason"].(string)
	if stopReason == "" {
		return false
	}

	contentArr, _ := msgData["content"].([]any)
	for _, item := range contentArr {
		ci, ok := item.(map[string]any)
		if !ok {
			continue
		}
		itemType, _ := ci["type"].(string)

		if itemType == "tool_use" && p.handler != nil {
			toolID, _ := ci["id"].(string)

			if _, alreadyStreamed := p.toolNames[toolID]; alreadyStreamed && toolID != "" {
				continue
			}

			p.closeTextMessage()
			p.closeCurrentTool()

			toolName, _ := ci["name"].(string)
			if toolID == "" {
				toolID = fmt.Sprintf("tool_%d_%d", p.toolIndex, time.Now().UnixNano())
			}

			msgID, stopped := p.beginMessage("execute")
			if stopped {
				return true
			}

			p.toolIndex++
			p.toolNames[toolID] = toolName
			p.toolMsgIDs[toolID] = msgID

			inputRaw, _ := json.Marshal(ci["input"])
			inputStr := string(inputRaw)
			p.toolInputs[toolID] = inputStr
			summary := extractSummary(toolName, inputStr)
			p.toolSummaries[toolID] = summary

			if p.emitExecute(map[string]any{
				"tool":    toolName,
				"tool_id": toolID,
				"input":   json.RawMessage(inputRaw),
				"summary": summary,
				"status":  "running",
				"runner":  "claude-cli",
			}) {
				p.endMessage()
				return true
			}
			p.endMessage()
		}

		if itemType == "text" {
			text, _ := ci["text"].(string)
			if text == "" || p.handler == nil {
				continue
			}
			if p.ensureTextMessage() {
				return true
			}
			if p.emitText(text) {
				return true
			}
		}
	}

	p.closeTextMessage()
	return false
}

func (p *streamParser) handleUser(msg map[string]any) (stopped bool) {
	msgData, _ := msg["message"].(map[string]any)
	if msgData == nil {
		return false
	}

	contentArr, _ := msgData["content"].([]interface{})
	for _, item := range contentArr {
		ci, ok := item.(map[string]any)
		if !ok {
			continue
		}

		ciType, _ := ci["type"].(string)
		if ciType != "tool_result" {
			continue
		}

		// Close any open text message before opening an execute message.
		p.closeTextMessage()

		// When Claude CLI executes tools in parallel, tool_result messages
		// can arrive while a new tool_use is still streaming. The downstream
		// handler (stream.go) tracks only a single currentGroupID, so we
		// must close the in-flight streaming tool message before opening
		// the result message — otherwise the message_start/message_end
		// pairs become interleaved and chunks lose their message_id.
		p.closeCurrentTool()

		toolUseID, _ := ci["tool_use_id"].(string)
		content := ci["content"]
		isError, _ := ci["is_error"].(bool)

		status := "completed"
		if isError {
			status = "error"
		}

		execProps := map[string]any{
			"tool_id":  toolUseID,
			"output":   content,
			"status":   status,
			"is_error": isError,
		}
		if name, ok := p.toolNames[toolUseID]; ok {
			execProps["tool"] = name
		}
		if input, ok := p.toolInputs[toolUseID]; ok {
			execProps["input"] = json.RawMessage(input)
		}
		if summary, ok := p.toolSummaries[toolUseID]; ok {
			execProps["summary"] = summary
		}

		if reuseMsgID, ok := p.toolMsgIDs[toolUseID]; ok {
			if p.beginMessageWithID(reuseMsgID, "execute") {
				return true
			}
		} else {
			if _, stopped := p.beginMessage("execute"); stopped {
				return true
			}
		}

		if p.emitExecute(execProps) {
			p.endMessage()
			return true
		}
		p.endMessage()
	}
	return false
}

func (p *streamParser) handleResult(msg map[string]any) error {
	isError, _ := msg["is_error"].(bool)
	if isError {
		if result, ok := msg["result"].(string); ok {
			if p.handler != nil {
				p.handler(message.ChunkError, []byte(result))
			}
			return fmt.Errorf("Claude CLI error: %s", result)
		}
	}

	p.closeTextMessage()

	if p.handler != nil {
		p.emitMetadata(map[string]any{
			"result_summary": map[string]any{
				"total_cost_usd": msg["total_cost_usd"],
				"duration_ms":    msg["duration_ms"],
				"num_turns":      msg["num_turns"],
				"usage":          msg["usage"],
			},
		})
	}

	p.completed = true
	return nil
}

func (p *streamParser) handleError(msg map[string]any) error {
	var errMsg string
	switch e := msg["error"].(type) {
	case string:
		errMsg = e
	case map[string]any:
		errMsg, _ = e["message"].(string)
	}
	if errMsg != "" {
		if p.handler != nil {
			p.handler(message.ChunkError, []byte(errMsg))
		}
		return fmt.Errorf("Claude CLI error: %s", errMsg)
	}
	return nil
}
