package dsh

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"regexp"
	"strings"
	"time"

	gouJSON "github.com/yaoapp/gou/json"
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
	handler           message.StreamFunc
	promptedSessionID string
	completed         bool

	textActive    bool
	thinkActive   bool
	toolIndex     int
	activeToolID  string
	toolNames     map[string]string
	toolMsgIDs    map[string]string
	toolInputs    map[string]string
	toolSummaries map[string]string
	toolFinished  map[string]bool
}

func newStreamParser(handler message.StreamFunc, promptedSessionID string) *streamParser {
	return &streamParser{
		handler:           handler,
		promptedSessionID: promptedSessionID,
		toolNames:         make(map[string]string),
		toolMsgIDs:        make(map[string]string),
		toolInputs:        make(map[string]string),
		toolSummaries:     make(map[string]string),
		toolFinished:      make(map[string]bool),
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
				p.closeTextMessage()
				p.closeThinkMessage()
				p.closeOrphanedTools()
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
	p.closeOrphanedTools()

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
		if p.promptedSessionID != "" && params.SessionID != p.promptedSessionID {
			return false
		}
		p.closeTextMessage()
		p.closeThinkMessage()
		p.closeOrphanedTools()
		p.completed = true
	}
	return false
}

func (p *streamParser) handleSessionEvent(raw json.RawMessage) (stopped bool) {
	var params sessionEventParams
	if err := json.Unmarshal(raw, &params); err != nil {
		return false
	}
	if p.promptedSessionID != "" && params.SessionID != p.promptedSessionID {
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
		msgID := p.emitMessageStart("execute")
		p.toolMsgIDs[toolID] = msgID
		execProps := map[string]any{
			"tool":    chunk.Name,
			"tool_id": toolID,
			"status":  "running",
			"runner":  "dsh",
		}
		injectDSHSemanticType(execProps, chunk.Name)
		if p.emitExecute(execProps) {
			p.endMessage()
			return true
		}
		p.endMessage()
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
		toolID := p.activeToolID
		oldSummary := p.toolSummaries[toolID]
		if input := p.toolInputs[toolID]; input != "" {
			name := p.toolNames[toolID]
			if summary := extractToolSummary(name, input); summary != "" {
				p.toolSummaries[toolID] = summary
			}
		}
		p.activeToolID = ""

		// Only emit update if summary actually changed (avoid duplicate with
		// delta-phase update), or if name became available but no summary was
		// sent yet.
		name := p.toolNames[toolID]
		newSummary := p.toolSummaries[toolID]
		summaryChanged := newSummary != oldSummary
		if summaryChanged || (name != "" && oldSummary == "") {
			if p.emitToolUpdate(toolID) {
				return true
			}
		}
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
	if chunk.ID != "" && chunk.ID != p.activeToolID {
		if msgID, ok := p.toolMsgIDs[p.activeToolID]; ok {
			p.toolMsgIDs[chunk.ID] = msgID
		}
	}
	nameJustSet := false
	if chunk.Name != "" && p.toolNames[p.activeToolID] == "" {
		p.toolNames[p.activeToolID] = chunk.Name
		nameJustSet = true
	}
	summaryEmitted := false
	if chunk.ArgsDelta != "" {
		existing := p.toolInputs[p.activeToolID]
		p.toolInputs[p.activeToolID] = existing + chunk.ArgsDelta

		toolName := p.toolNames[p.activeToolID]
		if s := extractToolSummaryPartial(toolName, p.toolInputs[p.activeToolID]); s != "" {
			if s != p.toolSummaries[p.activeToolID] {
				p.toolSummaries[p.activeToolID] = s
				if p.emitToolUpdate(p.activeToolID) {
					return true
				}
				summaryEmitted = true
			}
		}
	}

	// Emit update as soon as tool name arrives (first delta carries it).
	// Skip if a summary update already carried the name in this same delta.
	if nameJustSet && !summaryEmitted {
		toolID := p.activeToolID
		name := p.toolNames[toolID]
		if msgID, ok := p.toolMsgIDs[toolID]; ok {
			p.beginMessageWithID(msgID, "execute")
		} else {
			p.emitMessageStart("execute")
		}
		execProps := map[string]any{
			"tool":    name,
			"tool_id": toolID,
			"status":  "running",
			"runner":  "dsh",
		}
		injectDSHSemanticType(execProps, name)
		if p.emitExecute(execProps) {
			p.endMessage()
			return true
		}
		p.endMessage()
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

	if msgID, ok := p.toolMsgIDs[toolID]; ok {
		p.beginMessageWithID(msgID, "execute")
	} else {
		msgID := p.emitMessageStart("execute")
		p.toolMsgIDs[toolID] = msgID
	}

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
	p.endMessage()
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

	if msgID, ok := p.toolMsgIDs[callID]; ok {
		p.beginMessageWithID(msgID, "execute")
	} else {
		p.emitMessageStart("execute")
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
		p.toolFinished[callID] = true
		p.endMessage()
		return true
	}
	p.toolFinished[callID] = true
	p.endMessage()
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
		errMsg := formatTurnError(te.Reason)
		log.Warn("[dsh-parse] turn ended with error: kind=%s msg=%s", kind, errMsg)
		if p.handler != nil {
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

func (p *streamParser) closeOrphanedTools() {
	// Collect msgIDs of already-finished tools to handle aliases
	// (synthetic block-start ID and real callId both map to same msgID).
	finishedMsgIDs := make(map[string]bool)
	for toolID := range p.toolFinished {
		if msgID, ok := p.toolMsgIDs[toolID]; ok {
			finishedMsgIDs[msgID] = true
		}
	}

	for toolID, msgID := range p.toolMsgIDs {
		if p.toolFinished[toolID] || finishedMsgIDs[msgID] {
			continue
		}
		p.beginMessageWithID(msgID, "execute")
		execProps := map[string]any{
			"tool_id":  toolID,
			"status":   "error",
			"is_error": true,
			"output":   "session ended before tool completed",
		}
		if name, ok := p.toolNames[toolID]; ok {
			execProps["tool"] = name
		}
		if summary, ok := p.toolSummaries[toolID]; ok {
			execProps["summary"] = summary
		}
		injectDSHSemanticType(execProps, p.toolNames[toolID])
		p.emitExecute(execProps)
		p.endMessage()
		finishedMsgIDs[msgID] = true
	}
}

func (p *streamParser) emitToolUpdate(toolID string) (stopped bool) {
	name := p.toolNames[toolID]
	summary := p.toolSummaries[toolID]
	if msgID, ok := p.toolMsgIDs[toolID]; ok {
		p.beginMessageWithID(msgID, "execute")
	} else {
		p.emitMessageStart("execute")
	}
	execProps := map[string]any{
		"tool":    name,
		"tool_id": toolID,
		"status":  "running",
		"runner":  "dsh",
	}
	if summary != "" {
		execProps["summary"] = summary
	}
	injectDSHSemanticType(execProps, name)
	if p.emitExecute(execProps) {
		p.endMessage()
		return true
	}
	p.endMessage()
	return false
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

func extractSummaryFromObj(toolName string, obj map[string]any) string {
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
	case "skill":
		if n, ok := obj["name"].(string); ok {
			return truncateStr(n, 80)
		}
		if d, ok := obj["description"].(string); ok {
			return truncateStr(d, 80)
		}
	}

	for _, key := range []string{"command", "path", "file_path", "url", "query", "description", "prompt", "task", "name"} {
		if v, ok := obj[key].(string); ok {
			return truncateStr(v, 80)
		}
	}
	return ""
}

func extractToolSummary(toolName, inputJSON string) string {
	if inputJSON == "" {
		return ""
	}
	var obj map[string]any
	if err := json.Unmarshal([]byte(inputJSON), &obj); err != nil {
		return ""
	}
	return extractSummaryFromObj(toolName, obj)
}

func extractToolSummaryPartial(toolName, partialInput string) string {
	if partialInput == "" {
		return ""
	}
	v, err := gouJSON.Parse(partialInput)
	if err == nil {
		if obj, ok := v.(map[string]any); ok {
			return extractSummaryFromObj(toolName, obj)
		}
	}
	return extractSummaryByRegex(toolName, partialInput)
}

var summaryKeyRe = regexp.MustCompile(`"(file_path|path|command|name|description|url|query|prompt|task)"\s*:\s*"([^"]*)"`)

func extractSummaryByRegex(toolName string, input string) string {
	matches := summaryKeyRe.FindAllStringSubmatch(input, -1)
	if len(matches) == 0 {
		return ""
	}
	preferred := preferredKeyForTool(toolName)
	for _, m := range matches {
		if m[1] == preferred && m[2] != "" {
			return truncateStr(m[2], 80)
		}
	}
	if matches[0][2] != "" {
		return truncateStr(matches[0][2], 80)
	}
	return ""
}

func preferredKeyForTool(toolName string) string {
	switch strings.ToLower(toolName) {
	case "bash":
		return "command"
	case "write", "read", "edit", "str_replace_editor":
		return "file_path"
	case "skill":
		return "name"
	default:
		return ""
	}
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
	if msg != "" {
		return msg
	}

	if errStr, ok := reason["error"].(string); ok && errStr != "" {
		return errStr
	}

	if errMap, ok := reason["error"].(map[string]any); ok {
		if errMsg, ok := errMap["message"].(string); ok && errMsg != "" {
			return errMsg
		}
		if errCode, ok := errMap["code"].(string); ok && errCode != "" {
			return errCode
		}
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
