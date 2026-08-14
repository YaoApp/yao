package dsh

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
	"github.com/yaoapp/yao/agent/sandbox/v2/shared"
)

// streamParser parses DSH JSON-RPC NDJSON stdout into StreamFunc chunks.
//
// Tool lifecycle has two phases:
//
//	Phase 1 (LLM streaming): block-start(tool-call) → tool-call-delta → block-end
//	Phase 2 (execution):     tool/call → tool/result
//
// The same message_id is reused across phases for UI grouping.
type streamParser struct {
	handler   message.StreamFunc
	completed bool

	textActive      bool
	thinkActive     bool
	toolIndex       int
	activeToolID    string
	executingToolID string
	toolNames       map[string]string
	toolMsgIDs      map[string]string
	toolInputs      map[string]string
	toolSummaries   map[string]string
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

	reader := bufio.NewReaderSize(stdout, 64*1024)
	lineCount := 0
	startTime := time.Now()

	for {
		line, skipped, err := shared.ReadJSONLine(reader)
		if err == io.EOF {
			break
		}
		if err != nil {
			if ctx.Err() != nil {
				return ctx.Err()
			}
			return err
		}
		if skipped || len(line) == 0 {
			continue
		}
		lineCount++

		var msg jsonRPCMessage
		if err := json.Unmarshal(line, &msg); err != nil {
			log.Trace("[dsh-parse] JSON unmarshal error: %v (len=%d)", err, len(line))
			continue
		}

		stopped := p.dispatch(&msg)
		if stopped || p.completed {
			log.Trace("[dsh-parse] done: lines=%d elapsed=%v completed=%v stopped=%v", lineCount, time.Since(startTime).Round(time.Second), p.completed, stopped)
			return nil
		}
	}

	// Ensure any active message groups are closed before returning.
	p.closeTextMessage()
	p.closeThinkMessage()

	log.Trace("[dsh-parse] stream ended: lines=%d elapsed=%v completed=%v", lineCount, time.Since(startTime).Round(time.Second), p.completed)
	return nil
}

func (p *streamParser) dispatch(msg *jsonRPCMessage) (stopped bool) {
	if msg.isResponse() {
		return p.handleResponse(msg)
	}
	if !msg.isNotification() {
		return false
	}
	switch msg.Method {
	case "session.event":
		return p.handleSessionEvent(msg.Params)
	case "session.status":
		return p.handleSessionStatus(msg.Params)
	}
	return false
}

func (p *streamParser) handleResponse(msg *jsonRPCMessage) (stopped bool) {
	if msg.Error != nil {
		errText := msg.Error.Message
		if p.handler != nil {
			p.handler(message.ChunkError, []byte(errText))
		}
		return true
	}
	// Success responses are acknowledgments only — the plugin returns
	// session/prompt before the agent loop streams events.  Completion
	// is signaled by session.status "idle" or stdout EOF.
	return false
}

func (p *streamParser) handleSessionStatus(raw json.RawMessage) (stopped bool) {
	var params sessionStatusParams
	if err := json.Unmarshal(raw, &params); err != nil {
		return false
	}
	if params.Status == "idle" {
		p.closeTextMessage()
		p.closeThinkMessage()
		p.completed = true
	}
	return false
}

func (p *streamParser) handleSessionEvent(raw json.RawMessage) (stopped bool) {
	var params sessionEventParams
	if err := json.Unmarshal(raw, &params); err != nil {
		return false
	}
	event := params.Event

	switch event.Type {
	case "assistant/chunk":
		return p.handleAssistantChunk(event.Data)
	case "tool/call":
		return p.handleToolCall(event.Data)
	case "tool/result":
		return p.handleToolResult(event.Data)
	case "turn/end":
		return p.handleTurnEnd(event.Data)
	}
	return false
}

// --- assistant/chunk handlers ---

type chunkData struct {
	Turn  int             `json:"turn"`
	Step  int             `json:"step"`
	Chunk json.RawMessage `json:"chunk"`
}

type chunkInner struct {
	Type      string          `json:"type"`
	Index     int             `json:"index"`
	Text      string          `json:"text"`
	BlockType string          `json:"blockType"`
	Block     json.RawMessage `json:"block"`
	ID        string          `json:"id"`
	Name      string          `json:"name"`
	ArgsDelta string          `json:"argumentsDelta"`
	Usage     json.RawMessage `json:"usage"`
	Reason    json.RawMessage `json:"reason"`
}

func (p *streamParser) handleAssistantChunk(data json.RawMessage) (stopped bool) {
	var cd chunkData
	if err := json.Unmarshal(data, &cd); err != nil {
		return false
	}
	var chunk chunkInner
	if err := json.Unmarshal(cd.Chunk, &chunk); err != nil {
		return false
	}

	switch chunk.Type {
	case "text-delta":
		return p.onTextDelta(chunk.Text)
	case "reasoning-delta":
		return p.onReasoningDelta(chunk.Text)
	case "block-start":
		return p.onBlockStart(chunk)
	case "block-end":
		return p.onBlockEnd(chunk)
	case "tool-call-delta":
		return p.onToolCallDelta(chunk)
	case "usage":
		return p.onUsage(chunk.Usage)
	case "finish":
		return p.onFinish(chunk.Reason)
	}
	return false
}

func (p *streamParser) onTextDelta(text string) (stopped bool) {
	if text == "" {
		return false
	}
	p.closeThinkMessage()
	if !p.textActive {
		if p.beginMessage("text") {
			return true
		}
		p.textActive = true
	}
	text = strings.ReplaceAll(text, "\r\n", "\n")
	text = strings.ReplaceAll(text, "\r", "\n")
	return p.handler != nil && p.handler(message.ChunkText, []byte(text)) != 0
}

func (p *streamParser) onReasoningDelta(text string) (stopped bool) {
	if text == "" {
		return false
	}
	p.closeTextMessage()
	if !p.thinkActive {
		if p.beginMessage("thinking") {
			return true
		}
		p.thinkActive = true
	}
	return p.handler != nil && p.handler(message.ChunkThinking, []byte(text)) != 0
}

func (p *streamParser) onBlockStart(chunk chunkInner) (stopped bool) {
	switch chunk.BlockType {
	case "tool-call":
		p.closeTextMessage()
		p.closeThinkMessage()
		toolID := chunk.ID
		if toolID == "" {
			toolID = fmt.Sprintf("dsh-tool_%d_%d", p.toolIndex, time.Now().UnixNano())
		}
		p.toolIndex++
		p.activeToolID = toolID
		if chunk.Name != "" {
			p.toolNames[toolID] = chunk.Name
		}
	case "text":
		p.closeThinkMessage()
		if !p.textActive {
			if p.beginMessage("text") {
				return true
			}
			p.textActive = true
		}
	case "reasoning":
		p.closeTextMessage()
		if !p.thinkActive {
			if p.beginMessage("thinking") {
				return true
			}
			p.thinkActive = true
		}
	}
	return false
}

func (p *streamParser) onBlockEnd(chunk chunkInner) (stopped bool) {
	if p.activeToolID != "" {
		if input := p.toolInputs[p.activeToolID]; input != "" {
			name := p.toolNames[p.activeToolID]
			if summary := extractToolSummary(name, input); summary != "" {
				p.toolSummaries[p.activeToolID] = summary
			}
		}
		p.activeToolID = ""
		return false
	}
	if p.textActive {
		p.endMessage()
		p.textActive = false
	}
	if p.thinkActive {
		p.endMessage()
		p.thinkActive = false
	}
	return false
}

func (p *streamParser) onToolCallDelta(chunk chunkInner) (stopped bool) {
	if p.activeToolID == "" {
		return false
	}
	if chunk.ArgsDelta != "" {
		existing := p.toolInputs[p.activeToolID]
		p.toolInputs[p.activeToolID] = existing + chunk.ArgsDelta
	}
	return false
}

func (p *streamParser) onUsage(raw json.RawMessage) (stopped bool) {
	if p.handler == nil || len(raw) == 0 {
		return false
	}
	data, _ := json.Marshal(map[string]any{"usage": json.RawMessage(raw)})
	p.handler(message.ChunkMetadata, data)
	return false
}

func (p *streamParser) onFinish(raw json.RawMessage) (stopped bool) {
	if p.handler == nil || len(raw) == 0 {
		return false
	}
	var reason map[string]any
	json.Unmarshal(raw, &reason)
	data, _ := json.Marshal(map[string]any{"finish_reason": reason})
	p.handler(message.ChunkMetadata, data)
	return false
}

// --- tool/call and tool/result handlers ---

type toolCallData struct {
	Turn      int    `json:"turn"`
	Step      int    `json:"step"`
	CallID    string `json:"callId"`
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

type toolResultData struct {
	Turn    int             `json:"turn"`
	Step    int             `json:"step"`
	Message json.RawMessage `json:"message"`
}

type toolResultMessage struct {
	Source  map[string]any `json:"source"`
	Content []any          `json:"content"`
}

func (p *streamParser) handleToolCall(data json.RawMessage) (stopped bool) {
	var tc toolCallData
	if err := json.Unmarshal(data, &tc); err != nil {
		return false
	}
	p.closeTextMessage()
	p.closeThinkMessage()

	if p.executingToolID != "" {
		p.endMessage()
		p.executingToolID = ""
	}

	toolID := tc.CallID
	p.toolNames[toolID] = tc.Name
	if tc.Arguments != "" {
		p.toolInputs[toolID] = tc.Arguments
	}

	summary := p.toolSummaries[toolID]
	if tc.Arguments != "" {
		if s := extractToolSummary(tc.Name, tc.Arguments); s != "" {
			summary = s
			p.toolSummaries[toolID] = summary
		}
	}

	msgID := p.emitMessageStart("execute")
	p.toolMsgIDs[toolID] = msgID

	input := p.toolInputs[toolID]
	var inputJSON json.RawMessage
	if input != "" {
		inputJSON = json.RawMessage(input)
	}

	execProps := map[string]any{
		"tool":    tc.Name,
		"tool_id": toolID,
		"input":   inputJSON,
		"summary": summary,
		"status":  "running",
		"runner":  "dsh",
	}
	injectDSHSemanticType(execProps, tc.Name)
	if p.emitExecute(execProps) {
		p.endMessage()
		return true
	}
	p.executingToolID = toolID
	return false
}

func (p *streamParser) handleToolResult(data json.RawMessage) (stopped bool) {
	var tr toolResultData
	if err := json.Unmarshal(data, &tr); err != nil {
		return false
	}

	// Extract callId and content from the tool result message
	var trMsg toolResultMessage
	json.Unmarshal(tr.Message, &trMsg)

	callID := ""
	if source := trMsg.Source; source != nil {
		if id, ok := source["callId"].(string); ok {
			callID = id
		}
	}

	// Extract output text from content
	output := extractToolOutput(trMsg.Content)
	isError := false
	for _, item := range trMsg.Content {
		if m, ok := item.(map[string]any); ok {
			if e, ok := m["isError"].(bool); ok && e {
				isError = true
				break
			}
		}
	}

	status := "completed"
	if isError {
		status = "error"
	}

	groupAlreadyOpen := p.executingToolID == callID
	if !groupAlreadyOpen {
		if msgID, ok := p.toolMsgIDs[callID]; ok {
			p.beginMessageWithID(msgID, "execute")
		} else {
			p.emitMessageStart("execute")
		}
	}

	execProps := map[string]any{
		"tool_id":  callID,
		"output":   output,
		"status":   status,
		"is_error": isError,
	}
	if name, ok := p.toolNames[callID]; ok {
		execProps["tool"] = name
	}
	if input, ok := p.toolInputs[callID]; ok && input != "" {
		execProps["input"] = json.RawMessage(input)
	}
	if summary, ok := p.toolSummaries[callID]; ok {
		execProps["summary"] = summary
	}
	injectDSHSemanticType(execProps, p.toolNames[callID])
	if p.emitExecute(execProps) {
		p.endMessage()
		p.executingToolID = ""
		return true
	}
	p.endMessage()
	p.executingToolID = ""
	return false
}

// --- turn/end ---

type turnEndData struct {
	Turn   int            `json:"turn"`
	Reason map[string]any `json:"reason"`
}

func (p *streamParser) handleTurnEnd(data json.RawMessage) (stopped bool) {
	var te turnEndData
	if err := json.Unmarshal(data, &te); err != nil {
		return false
	}
	p.closeTextMessage()
	p.closeThinkMessage()

	if p.handler != nil {
		summary, _ := json.Marshal(map[string]any{
			"result_summary": map[string]any{
				"turn":   te.Turn,
				"reason": te.Reason,
			},
		})
		p.handler(message.ChunkMetadata, summary)
	}

	// Surface DSH errors with full detail from the reason payload
	if kind, ok := te.Reason["kind"].(string); ok && kind != "completed" && kind != "stop" {
		if p.handler != nil {
			errMsg := formatTurnError(te.Reason)
			p.handler(message.ChunkError, []byte(errMsg))
		}
	}

	return false
}

// --- message lifecycle helpers ---

func (p *streamParser) beginMessage(msgType string) (stopped bool) {
	startData := message.EventMessageStartData{
		MessageID: fmt.Sprintf("dsh-%s-%s", msgType, message.GenerateNanoID()),
		Type:      msgType,
		Timestamp: time.Now().UnixMilli(),
	}
	sd, _ := json.Marshal(startData)
	return p.handler != nil && p.handler(message.ChunkMessageStart, sd) != 0
}

func (p *streamParser) emitMessageStart(msgType string) string {
	id := fmt.Sprintf("dsh-%s-%s", msgType, message.GenerateNanoID())
	startData := message.EventMessageStartData{
		MessageID: id,
		Type:      msgType,
		Timestamp: time.Now().UnixMilli(),
	}
	sd, _ := json.Marshal(startData)
	if p.handler != nil {
		p.handler(message.ChunkMessageStart, sd)
	}
	return id
}

func (p *streamParser) beginMessageWithID(id, msgType string) {
	startData := message.EventMessageStartData{
		MessageID: id,
		Type:      msgType,
		Timestamp: time.Now().UnixMilli(),
	}
	sd, _ := json.Marshal(startData)
	if p.handler != nil {
		p.handler(message.ChunkMessageStart, sd)
	}
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

func (p *streamParser) closeThinkMessage() {
	if p.thinkActive {
		p.endMessage()
		p.thinkActive = false
	}
}

func (p *streamParser) emitExecute(props map[string]any) (stopped bool) {
	data, err := json.Marshal(props)
	if err != nil {
		return false
	}
	return p.handler != nil && p.handler(message.ChunkExecute, data) != 0
}

// --- helpers ---

func extractToolOutput(content []any) any {
	if len(content) == 0 {
		return nil
	}
	var texts []string
	for _, item := range content {
		if m, ok := item.(map[string]any); ok {
			if c, ok := m["content"].([]any); ok {
				for _, ci := range c {
					if cm, ok := ci.(map[string]any); ok {
						if text, ok := cm["text"].(string); ok {
							texts = append(texts, text)
						}
					}
				}
			}
		}
	}
	if len(texts) == 1 {
		return texts[0]
	}
	if len(texts) > 1 {
		return strings.Join(texts, "\n")
	}
	return content
}

func extractToolSummary(toolName, inputJSON string) string {
	if inputJSON == "" {
		return ""
	}
	var obj map[string]any
	if err := json.Unmarshal([]byte(inputJSON), &obj); err != nil {
		return ""
	}

	switch strings.ToLower(toolName) {
	case "bash":
		if cmd, ok := obj["command"].(string); ok {
			return truncateStr(cmd, 80)
		}
	case "write", "read", "edit", "str_replace_editor":
		if fp, ok := obj["file_path"].(string); ok {
			return fp
		}
		if fp, ok := obj["path"].(string); ok {
			return fp
		}
	}

	for _, key := range []string{"command", "path", "file_path", "url", "query", "description", "prompt", "task"} {
		if v, ok := obj[key].(string); ok {
			return truncateStr(v, 80)
		}
	}
	return ""
}

func truncateStr(s string, max int) string {
	s = strings.TrimSpace(s)
	s = strings.ReplaceAll(s, "\n", " ")
	if len(s) > max {
		return s[:max] + "..."
	}
	return s
}

func formatTurnError(reason map[string]any) string {
	kind, _ := reason["kind"].(string)
	msg, _ := reason["message"].(string)
	errType, _ := reason["error"].(string)
	if msg != "" {
		return msg
	}
	if errType != "" {
		return errType
	}
	detail, _ := json.Marshal(reason)
	return fmt.Sprintf("DSH turn ended: %s (%s)", kind, string(detail))
}

func injectDSHSemanticType(props map[string]any, toolName string) {
	switch strings.ToLower(toolName) {
	case "subagent":
		props["semantic_type"] = message.TypeAgent
	case "todo_write", "todo":
		props["semantic_type"] = message.TypeTodo
	case "ask-user":
		props["semantic_type"] = message.TypeQuestion
	}
}
