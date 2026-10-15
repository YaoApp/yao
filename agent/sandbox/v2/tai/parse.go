package tai

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/pmezard/go-difflib/difflib"
	"github.com/yaoapp/yao/agent/i18n"
	"github.com/yaoapp/yao/agent/output/message"
	"github.com/yaoapp/yao/agent/sandbox/v2/shared"
)

// ---------------------------------------------------------------------------
// NDJSON event line (yao-side deserialization of tai's EventLine)
// ---------------------------------------------------------------------------

type eventLine struct {
	Type      string `json:"type"`
	EventType string `json:"event_type,omitempty"`

	Delta        string `json:"delta,omitempty"`
	Content      string `json:"content,omitempty"`
	ContentIndex *int   `json:"content_index,omitempty"`

	Tool         string          `json:"tool,omitempty"`
	ToolID       string          `json:"tool_id,omitempty"`
	ToolLabel    string          `json:"tool_label,omitempty"`
	Args         json.RawMessage `json:"args,omitempty"`
	ParentToolID string          `json:"parent_tool_id,omitempty"`
	Result       json.RawMessage `json:"result,omitempty"`
	IsError      bool            `json:"is_error,omitempty"`
	DurationMs   int64           `json:"duration_ms,omitempty"`

	PartialResult json.RawMessage `json:"partial_result,omitempty"`

	ToolCall *toolCallInfo `json:"tool_call,omitempty"`

	Turn      *int   `json:"turn,omitempty"`
	Timestamp *int64 `json:"timestamp,omitempty"`

	Reason          string          `json:"reason,omitempty"`
	Summary         string          `json:"summary,omitempty"`
	TokensBefore    int             `json:"tokens_before,omitempty"`
	TokensAfter     int             `json:"tokens_after,omitempty"`
	Aborted         bool            `json:"aborted,omitempty"`
	WillRetry       bool            `json:"will_retry,omitempty"`
	CompactionUsage json.RawMessage `json:"compaction_usage,omitempty"`

	Attempt     int    `json:"attempt,omitempty"`
	MaxAttempts int    `json:"max_attempts,omitempty"`
	DelayMs     int    `json:"delay_ms,omitempty"`
	Success     *bool  `json:"success,omitempty"`
	FinalError  string `json:"final_error,omitempty"`
	Source      string `json:"source,omitempty"`

	SessionID     string          `json:"session_id,omitempty"`
	SessionName   string          `json:"session_name,omitempty"`
	ThinkingLevel string          `json:"thinking_level,omitempty"`
	Steering      []string        `json:"steering,omitempty"`
	FollowUp      []string        `json:"follow_up,omitempty"`
	Entry         json.RawMessage `json:"entry,omitempty"`

	Usage      json.RawMessage `json:"usage,omitempty"`
	StopReason string          `json:"stop_reason,omitempty"`

	Error string `json:"error,omitempty"`
}

type toolCallInfo struct {
	ID        string          `json:"id"`
	Name      string          `json:"name"`
	Arguments json.RawMessage `json:"arguments"`
}

// ---------------------------------------------------------------------------
// Stream parser — state machine translating NDJSON events to Yao messages.
// Mirrors DSH parse.go architecture: textActive/thinkActive flags, tool
// tracking maps, orphan cleanup, and loading state management.
// ---------------------------------------------------------------------------

type streamParser struct {
	handler   message.StreamFunc
	completed bool
	locale    string

	// text/thinking message group state.
	textActive  bool
	thinkActive bool

	// LLM-phase tool call tracking (toolcall_start/delta/end).
	activeToolCallIdx    int
	toolCallSyntheticIDs map[int]string
	toolCallInputs       map[string]string

	// Tool execution tracking.
	toolIndex        int
	toolNames        map[string]string
	toolMsgIDs       map[string]string
	toolFinished     map[string]bool
	toolSummaries    map[string]string
	lastActiveToolID string

	// Loading state.
	prepareLoadingMsgID string
	activeLoadingID     string
	loadingSeq          int
	contentStarted      bool

	// Message counter.
	msgSeq int
}

func newStreamParser(handler message.StreamFunc, locale string, prepareLoadingMsgID string) *streamParser {
	return &streamParser{
		handler:              handler,
		locale:               locale,
		prepareLoadingMsgID:  prepareLoadingMsgID,
		toolCallSyntheticIDs: make(map[int]string),
		toolCallInputs:       make(map[string]string),
		toolNames:            make(map[string]string),
		toolMsgIDs:           make(map[string]string),
		toolFinished:         make(map[string]bool),
		toolSummaries:        make(map[string]string),
	}
}

// parse reads NDJSON lines from stdout and dispatches each event.
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

	reader := bufio.NewReaderSize(stdout, 256*1024)

	for {
		line, skipped, err := shared.ReadJSONLine(reader)
		if err == io.EOF {
			fmt.Println("[YAO-DEBUG] parse: EOF")
			break
		}
		if err != nil {
			fmt.Printf("[YAO-DEBUG] parse: read error: %v (ctx=%v)\n", err, ctx.Err())
			if ctx.Err() != nil {
				return ctx.Err()
			}
			return err
		}
		if skipped || len(line) == 0 {
			fmt.Printf("[YAO-DEBUG] parse: skipped=%v len=%d\n", skipped, len(line))
			continue
		}

		fmt.Printf("[YAO-DEBUG] parse: line=%s\n", string(line))

		var ev eventLine
		if err := json.Unmarshal(line, &ev); err != nil {
			continue
		}
		if p.dispatch(ev) {
			break
		}
	}

	p.closeOrphanedTools()
	return nil
}

// dispatch routes a single event to the appropriate handler.
// Returns true to stop parsing.
func (p *streamParser) dispatch(ev eventLine) (stopped bool) {
	fmt.Printf("[YAO-DEBUG] dispatch: type=%q event_type=%q\n", ev.Type, ev.EventType)
	switch ev.Type {
	case "session":
		p.emitMetadata(map[string]any{"tai_session_id": ev.SessionID})

	case "agent_start":
		p.emitLoading(i18n.T(p.locale, "tai.agent.starting"))

	case "agent_end":
		fmt.Println("[YAO-DEBUG] agent_end received, closing all")
		p.closeAll()
		p.closeOrphanedTools()
		p.closeActiveLoading()
		p.completed = true
		if p.handler != nil {
			p.handler(message.ChunkStreamEnd, nil)
		}
		return true

	case "agent_settled":
		p.closeActiveLoading()

	case "turn_start":
		p.emitLoading(i18n.T(p.locale, "tai.turn.processing"))
		meta := map[string]any{}
		if ev.Turn != nil {
			meta["turn"] = *ev.Turn
		}
		if ev.Timestamp != nil {
			meta["turn_timestamp"] = *ev.Timestamp
		}
		if len(meta) > 0 {
			p.emitMetadata(meta)
		}

	case "turn_end":
		p.closeAll()
		if ev.Turn != nil {
			p.emitMetadata(map[string]any{"turn_end": *ev.Turn})
		}

	case "message_start":
		// Internal state; no Yao output.

	case "message_update":
		return p.dispatchMessageUpdate(ev)

	case "message_end":
		p.closeTextMessage()
		p.closeThinkMessage()

	case "tool_start":
		return p.onToolExecutionStart(ev)

	case "tool_update":
		return p.onToolExecutionUpdate(ev)

	case "tool_end":
		return p.onToolExecutionEnd(ev)

	case "bash_delta":
		return p.onBashDelta(ev)

	case "compaction_start":
		p.emitLoading(i18n.T(p.locale, "tai.compaction.running"))

	case "compaction_end":
		if ev.Aborted {
			p.emitLoading(i18n.T(p.locale, "tai.compaction.aborted"))
		} else if ev.Error != "" {
			p.emitLoading(fmt.Sprintf(i18n.T(p.locale, "tai.compaction.error"), ev.Error))
		} else {
			p.closeActiveLoading()
		}
		meta := map[string]any{
			"compaction": map[string]any{
				"tokens_before": ev.TokensBefore,
				"tokens_after":  ev.TokensAfter,
				"aborted":       ev.Aborted,
			},
		}
		if ev.CompactionUsage != nil {
			meta["compaction_usage"] = json.RawMessage(ev.CompactionUsage)
		}
		p.emitMetadata(meta)

	case "auto_retry_start":
		msg := fmt.Sprintf(i18n.T(p.locale, "tai.retry.waiting"), ev.Attempt, ev.MaxAttempts, ev.DelayMs/1000)
		p.emitLoading(msg)

	case "auto_retry_end":
		p.closeActiveLoading()

	case "summarization_retry_scheduled":
		msg := fmt.Sprintf(i18n.T(p.locale, "tai.summarization.retrying"), ev.Attempt, ev.MaxAttempts)
		p.emitLoading(msg)

	case "summarization_retry_start":
		p.emitLoading(i18n.T(p.locale, "tai.summarization.running"))

	case "summarization_retry_finished":
		p.closeActiveLoading()

	case "queue_update":
		meta := map[string]any{}
		if len(ev.Steering) > 0 {
			meta["steering"] = ev.Steering
		}
		if len(ev.FollowUp) > 0 {
			meta["follow_up"] = ev.FollowUp
		}
		if len(meta) > 0 {
			p.emitMetadata(meta)
		}

	case "session_info":
		if ev.SessionName != "" {
			p.emitMetadata(map[string]any{"session_name": ev.SessionName})
		}

	case "thinking_level":
		if ev.ThinkingLevel != "" {
			p.emitMetadata(map[string]any{"thinking_level": ev.ThinkingLevel})
		}

	case "entry_appended":
		if ev.Entry != nil {
			p.emitMetadata(map[string]any{"entry": json.RawMessage(ev.Entry)})
		}

	case "error":
		p.closeAll()
		if p.handler != nil {
			p.handler(message.ChunkError, []byte(ev.Error))
		}
	}
	return false
}

// ---------------------------------------------------------------------------
// MessageUpdateEvent sub-dispatch (12 AssistantMessageEvent types)
// ---------------------------------------------------------------------------

func (p *streamParser) dispatchMessageUpdate(ev eventLine) (stopped bool) {
	switch ev.EventType {
	case "start":
		p.emitLoading(i18n.T(p.locale, "tai.message.thinking"))

	case "text_start":
		p.closeActiveLoading()
		p.contentStarted = true
		p.closeThinkMessage()
		if !p.textActive {
			if p.beginMessage("text") {
				return true
			}
			p.textActive = true
		}

	case "text_delta":
		return p.onTextDelta(ev.Delta)

	case "text_end":
		p.closeTextMessage()

	case "thinking_start":
		p.closeActiveLoading()
		p.contentStarted = true
		p.closeTextMessage()
		if !p.thinkActive {
			if p.beginMessage("thinking") {
				return true
			}
			p.thinkActive = true
		}

	case "thinking_delta":
		return p.onThinkingDelta(ev.Delta)

	case "thinking_end":
		p.closeThinkMessage()

	case "toolcall_start":
		return p.onToolCallStart(ev)

	case "toolcall_delta":
		p.onToolCallDelta(ev)

	case "toolcall_end":
		return p.onToolCallEnd(ev)

	case "done":
		meta := map[string]any{}
		if ev.StopReason != "" {
			meta["stop_reason"] = ev.StopReason
		}
		if ev.Usage != nil {
			meta["usage"] = json.RawMessage(ev.Usage)
		}
		if len(meta) > 0 {
			p.emitMetadata(meta)
		}

	case "error":
		p.closeAll()
		errMsg := ev.Error
		if errMsg == "" {
			errMsg = "LLM error"
			if ev.StopReason != "" {
				errMsg += ": " + ev.StopReason
			}
		}
		if p.handler != nil {
			p.handler(message.ChunkError, []byte(errMsg))
		}
	}
	return false
}

// ---------------------------------------------------------------------------
// Text / Thinking handlers
// ---------------------------------------------------------------------------

func (p *streamParser) onTextDelta(text string) (stopped bool) {
	p.closeActiveLoading()
	p.contentStarted = true
	p.closeThinkMessage()
	if !p.textActive {
		if p.beginMessage("text") {
			return true
		}
		p.textActive = true
	}
	text = strings.ReplaceAll(text, "\r\n", "\n")
	fmt.Printf("[YAO-DEBUG] onTextDelta: sending ChunkText len=%d text=%q\n", len(text), text[:min(len(text), 50)])
	return p.handler != nil && p.handler(message.ChunkText, []byte(text)) != 0
}

func (p *streamParser) onThinkingDelta(text string) (stopped bool) {
	p.closeActiveLoading()
	p.contentStarted = true
	p.closeTextMessage()
	if !p.thinkActive {
		if p.beginMessage("thinking") {
			return true
		}
		p.thinkActive = true
	}
	text = strings.ReplaceAll(text, "\r\n", "\n")
	return p.handler != nil && p.handler(message.ChunkThinking, []byte(text)) != 0
}

// ---------------------------------------------------------------------------
// LLM-phase tool call handlers (toolcall_start / toolcall_delta / toolcall_end)
// ---------------------------------------------------------------------------

func (p *streamParser) onToolCallStart(ev eventLine) (stopped bool) {
	p.closeActiveLoading()
	p.contentStarted = true
	p.closeTextMessage()
	p.closeThinkMessage()

	syntheticID := fmt.Sprintf("tai-tc_%d_%d", p.toolIndex, time.Now().UnixNano())
	p.toolIndex++

	idx := 0
	if ev.ContentIndex != nil {
		idx = *ev.ContentIndex
	}
	p.activeToolCallIdx = idx
	p.toolCallSyntheticIDs[idx] = syntheticID

	msgID := p.emitMessageStart("execute")
	p.toolMsgIDs[syntheticID] = msgID

	execProps := map[string]any{
		"tool_id": syntheticID,
		"status":  "running",
		"runner":  "tai",
	}
	if p.emitExecute(execProps) {
		p.endMessage()
		return true
	}
	p.endMessage()
	return false
}

func (p *streamParser) onToolCallDelta(ev eventLine) {
	idx := 0
	if ev.ContentIndex != nil {
		idx = *ev.ContentIndex
	}
	syntheticID, ok := p.toolCallSyntheticIDs[idx]
	if !ok {
		return
	}
	p.toolCallInputs[syntheticID] += ev.Delta
}

func (p *streamParser) onToolCallEnd(ev eventLine) (stopped bool) {
	idx := 0
	if ev.ContentIndex != nil {
		idx = *ev.ContentIndex
	}
	syntheticID, ok := p.toolCallSyntheticIDs[idx]
	if !ok {
		return false
	}

	// Map synthetic → real tool call ID so tool_start can reuse the msgID.
	if ev.ToolCall != nil && ev.ToolCall.ID != "" {
		if msgID, ok := p.toolMsgIDs[syntheticID]; ok {
			p.toolMsgIDs[ev.ToolCall.ID] = msgID
		}
		p.toolNames[ev.ToolCall.ID] = ev.ToolCall.Name

		if ev.ToolCall.Arguments != nil {
			p.toolCallInputs[ev.ToolCall.ID] = string(ev.ToolCall.Arguments)
		}
	}

	// Update the execute message with finalized tool name and summary.
	toolName := ""
	if ev.ToolCall != nil {
		toolName = ev.ToolCall.Name
	}
	if toolName != "" {
		p.toolNames[syntheticID] = toolName
	}

	inputJSON := p.toolCallInputs[syntheticID]
	summary := extractToolSummary(toolName, inputJSON)
	if summary != "" {
		p.toolSummaries[syntheticID] = summary
		if ev.ToolCall != nil && ev.ToolCall.ID != "" {
			p.toolSummaries[ev.ToolCall.ID] = summary
		}
	}

	if msgID, ok := p.toolMsgIDs[syntheticID]; ok {
		p.beginMessageWithID(msgID, "execute")
		execProps := map[string]any{
			"tool":    toolName,
			"tool_id": syntheticID,
			"status":  "running",
			"runner":  "tai",
		}
		if inputJSON != "" {
			execProps["input"] = json.RawMessage(inputJSON)
		}
		if summary != "" {
			execProps["summary"] = summary
		}
		injectTaiSemanticType(execProps, toolName)
		p.emitExecute(execProps)
		p.endMessage()
	}

	return false
}

// ---------------------------------------------------------------------------
// Tool execution handlers (tool_start / tool_update / tool_end)
// ---------------------------------------------------------------------------

func (p *streamParser) onToolExecutionStart(ev eventLine) (stopped bool) {
	p.closeAll()
	p.closeActiveLoading()
	p.contentStarted = true

	fmt.Printf("[YAO-DEBUG] tool_start: tool=%q toolID=%q label=%q args=%s\n", ev.Tool, ev.ToolID, ev.ToolLabel, string(ev.Args))

	toolID := ev.ToolID
	p.toolNames[toolID] = ev.Tool
	p.lastActiveToolID = toolID

	// Reuse msgID from LLM phase if available.
	if msgID, ok := p.toolMsgIDs[toolID]; ok {
		p.beginMessageWithID(msgID, "execute")
	} else {
		msgID := p.emitMessageStart("execute")
		p.toolMsgIDs[toolID] = msgID
	}

	var inputJSON string
	if ev.Args != nil {
		inputJSON = string(ev.Args)
		// Store args for file_patches generation in tool_end.
		if _, exists := p.toolCallInputs[toolID]; !exists {
			p.toolCallInputs[toolID] = inputJSON
		}
	}

	summary := p.toolSummaries[toolID]
	if summary == "" && ev.ToolLabel != "" {
		summary = truncateStr(ev.ToolLabel, 80)
		p.toolSummaries[toolID] = summary
	}
	if summary == "" && inputJSON != "" {
		summary = extractToolSummary(ev.Tool, inputJSON)
		if summary != "" {
			p.toolSummaries[toolID] = summary
		}
	}

	execProps := map[string]any{
		"tool":    ev.Tool,
		"tool_id": toolID,
		"status":  "running",
		"runner":  "tai",
	}
	if ev.ToolLabel != "" {
		execProps["tool_label"] = ev.ToolLabel
	}
	if inputJSON != "" {
		execProps["input"] = json.RawMessage(inputJSON)
	}
	if summary != "" {
		execProps["summary"] = summary
	}
	injectTaiSemanticType(execProps, ev.Tool)

	if p.emitExecute(execProps) {
		p.endMessage()
		return true
	}
	p.endMessage()
	return false
}

func (p *streamParser) onToolExecutionUpdate(ev eventLine) (stopped bool) {
	toolID := ev.ToolID
	if toolID == "" {
		return false
	}

	if msgID, ok := p.toolMsgIDs[toolID]; ok {
		p.beginMessageWithID(msgID, "execute")
	} else {
		msgID := p.emitMessageStart("execute")
		p.toolMsgIDs[toolID] = msgID
	}

	execProps := map[string]any{
		"tool":    ev.Tool,
		"tool_id": toolID,
		"status":  "running",
		"runner":  "tai",
	}
	if ev.PartialResult != nil {
		execProps["output"] = extractResultContent(ev.PartialResult)
	}
	if summary := p.toolSummaries[toolID]; summary != "" {
		execProps["summary"] = summary
	}
	injectTaiSemanticType(execProps, ev.Tool)

	if p.emitExecute(execProps) {
		p.endMessage()
		return true
	}
	p.endMessage()
	return false
}

func (p *streamParser) onToolExecutionEnd(ev eventLine) (stopped bool) {
	toolID := ev.ToolID

	if msgID, ok := p.toolMsgIDs[toolID]; ok {
		p.beginMessageWithID(msgID, "execute")
	} else {
		p.emitMessageStart("execute")
	}

	status := "completed"
	if ev.IsError {
		status = "error"
	}

	execProps := map[string]any{
		"tool":    ev.Tool,
		"tool_id": toolID,
		"status":  status,
		"runner":  "tai",
	}
	if ev.IsError {
		execProps["is_error"] = true
	}
	if ev.Result != nil {
		execProps["output"] = extractResultContent(ev.Result)
	}
	if summary := p.toolSummaries[toolID]; summary != "" {
		execProps["summary"] = summary
	}
	if ev.DurationMs > 0 {
		execProps["duration_ms"] = ev.DurationMs
	}
	injectTaiSemanticType(execProps, ev.Tool)

	// Generate file_patches for write/edit tools (aligned with DSH diff.go).
	if !ev.IsError {
		if inputJSON := p.toolCallInputs[toolID]; inputJSON != "" {
			if patches := computeFilePatchFromArgs(ev.Tool, inputJSON); len(patches) > 0 {
				execProps["file_patches"] = patches
			}
		}
	}

	p.toolFinished[toolID] = true
	if p.lastActiveToolID == toolID {
		p.lastActiveToolID = ""
	}

	if p.emitExecute(execProps) {
		p.endMessage()
		return true
	}
	p.endMessage()
	return false
}

func (p *streamParser) onBashDelta(ev eventLine) (stopped bool) {
	toolID := ev.ToolID
	if toolID == "" {
		// Fallback: use the most recent unfinished tool.
		if p.lastActiveToolID != "" && !p.toolFinished[p.lastActiveToolID] {
			toolID = p.lastActiveToolID
		}
	}
	if toolID == "" {
		return false
	}

	if msgID, ok := p.toolMsgIDs[toolID]; ok {
		p.beginMessageWithID(msgID, "execute")
	} else {
		return false
	}

	execProps := map[string]any{
		"tool_id": toolID,
		"status":  "running",
		"runner":  "tai",
		"output":  ev.Delta,
	}
	if name := p.toolNames[toolID]; name != "" {
		execProps["tool"] = name
	}

	if p.emitExecute(execProps) {
		p.endMessage()
		return true
	}
	p.endMessage()
	return false
}

// ---------------------------------------------------------------------------
// Orphaned tool cleanup (DSH parse.go lines 730-761)
// ---------------------------------------------------------------------------

func (p *streamParser) closeOrphanedTools() {
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
		p.emitExecute(map[string]any{
			"tool":     p.toolNames[toolID],
			"tool_id":  toolID,
			"status":   "error",
			"is_error": true,
			"output":   "session ended before tool completed",
			"runner":   "tai",
		})
		p.endMessage()
	}
}

// ---------------------------------------------------------------------------
// Message lifecycle helpers
// ---------------------------------------------------------------------------

func (p *streamParser) beginMessage(msgType string) (stopped bool) {
	p.msgSeq++
	id := fmt.Sprintf("tai-%s-%d-%d", msgType, p.msgSeq, time.Now().UnixNano())
	return p.beginMessageWithID(id, msgType)
}

func (p *streamParser) beginMessageWithID(id, msgType string) (stopped bool) {
	data, _ := json.Marshal(message.EventMessageStartData{
		MessageID: id,
		Type:      msgType,
		Timestamp: time.Now().UnixMilli(),
	})
	fmt.Printf("[YAO-DEBUG] beginMessageWithID: id=%s type=%s\n", id, msgType)
	return p.handler != nil && p.handler(message.ChunkMessageStart, data) != 0
}

// emitMessageStart emits a MessageStart and returns the generated message ID.
func (p *streamParser) emitMessageStart(msgType string) string {
	p.msgSeq++
	id := fmt.Sprintf("tai-%s-%d-%d", msgType, p.msgSeq, time.Now().UnixNano())
	data, _ := json.Marshal(message.EventMessageStartData{
		MessageID: id,
		Type:      msgType,
		Timestamp: time.Now().UnixMilli(),
	})
	if p.handler != nil {
		p.handler(message.ChunkMessageStart, data)
	}
	return id
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

func (p *streamParser) closeAll() {
	p.closeTextMessage()
	p.closeThinkMessage()
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

// ---------------------------------------------------------------------------
// Loading state management
// ---------------------------------------------------------------------------

func (p *streamParser) emitLoading(text string) {
	var msg *message.Message

	if !p.contentStarted && p.prepareLoadingMsgID != "" {
		msg = &message.Message{
			MessageID:   p.prepareLoadingMsgID,
			Delta:       true,
			DeltaAction: message.DeltaReplace,
			Type:        message.TypeLoading,
			Props:       map[string]any{"message": text},
		}
	} else if p.activeLoadingID != "" {
		msg = &message.Message{
			MessageID:   p.activeLoadingID,
			Delta:       true,
			DeltaAction: message.DeltaReplace,
			Type:        message.TypeLoading,
			Props:       map[string]any{"message": text},
		}
	} else {
		p.loadingSeq++
		p.activeLoadingID = fmt.Sprintf("tai-loading-%d", p.loadingSeq)
		msg = &message.Message{
			MessageID: p.activeLoadingID,
			Type:      message.TypeLoading,
			Props:     map[string]any{"message": text},
		}
	}

	data, _ := json.Marshal(msg)
	if p.handler != nil {
		p.handler(message.ChunkLoading, data)
	}
}

func (p *streamParser) closeActiveLoading() {
	if p.activeLoadingID == "" {
		return
	}
	msg := &message.Message{
		MessageID:   p.activeLoadingID,
		Delta:       true,
		DeltaAction: message.DeltaReplace,
		Type:        message.TypeLoading,
		Props:       map[string]any{"done": true, "message": ""},
	}
	data, _ := json.Marshal(msg)
	if p.handler != nil {
		p.handler(message.ChunkLoading, data)
	}
	p.activeLoadingID = ""
}

// extractResultContent extracts the display-ready output from a tai
// ResultInfo JSON. Returns the "content" string when present; falls back
// to "structured_content" as-is; returns the raw JSON otherwise.
// This matches DSH's extractToolOutput which returns a plain string.
func extractResultContent(raw json.RawMessage) any {
	var ri struct {
		Content           string          `json:"content"`
		StructuredContent json.RawMessage `json:"structured_content"`
	}
	if err := json.Unmarshal(raw, &ri); err != nil {
		return string(raw)
	}
	if ri.Content != "" {
		return ri.Content
	}
	if ri.StructuredContent != nil {
		return json.RawMessage(ri.StructuredContent)
	}
	return string(raw)
}

// injectTaiSemanticType sets semantic_type for PiG tools that need
// special UI rendering, mirroring DSH's injectDSHSemanticType.
func injectTaiSemanticType(props map[string]any, toolName string) {
	switch strings.ToLower(toolName) {
	case "task", "subagent":
		props["semantic_type"] = message.TypeAgent
	case "todo_write", "todo":
		props["semantic_type"] = message.TypeTodo
	case "ask_user", "ask-user":
		props["semantic_type"] = message.TypeQuestion
	}
}

// ---------------------------------------------------------------------------
// File diff support — generates file_patches from write/edit tool args.
// PiG's write tool carries {file_path, content} and edit carries
// {path, edits:[{oldText,newText}]}. We compute unified diffs from these
// inputs to populate file_patches for UI rendering.
// ---------------------------------------------------------------------------

// computeFilePatchFromArgs extracts file patches from tool input args.
// Returns nil when the tool is not a file-mutation tool or args lack data.
func computeFilePatchFromArgs(toolName, inputJSON string) []*message.FilePatch {
	if inputJSON == "" {
		return nil
	}
	var obj map[string]any
	if err := json.Unmarshal([]byte(inputJSON), &obj); err != nil {
		return nil
	}

	switch strings.ToLower(toolName) {
	case "write", "create":
		return computeWritePatch(obj)
	case "edit":
		return computeEditPatch(obj)
	}
	return nil
}

func computeWritePatch(obj map[string]any) []*message.FilePatch {
	path, _ := obj["file_path"].(string)
	if path == "" {
		path, _ = obj["path"].(string)
	}
	content, _ := obj["content"].(string)
	if path == "" || content == "" {
		return nil
	}

	newLines := splitLines(content)
	patch, err := difflib.GetUnifiedDiffString(difflib.UnifiedDiff{
		A:        nil,
		B:        newLines,
		FromFile: "/dev/null",
		ToFile:   "b/" + path,
		Context:  3,
	})
	if err != nil || patch == "" {
		return nil
	}

	adds, _ := countDiffLines(patch)
	return []*message.FilePatch{{
		Path:      path,
		Status:    "created",
		Patch:     patch,
		Additions: adds,
	}}
}

func computeEditPatch(obj map[string]any) []*message.FilePatch {
	path, _ := obj["file_path"].(string)
	if path == "" {
		path, _ = obj["path"].(string)
	}
	if path == "" {
		return nil
	}

	var edits []struct{ OldText, NewText string }

	if rawEdits, ok := obj["edits"].([]any); ok {
		for _, e := range rawEdits {
			em, ok := e.(map[string]any)
			if !ok {
				continue
			}
			old, _ := em["oldText"].(string)
			if old == "" {
				old, _ = em["old_text"].(string)
			}
			nw, _ := em["newText"].(string)
			if nw == "" {
				nw, _ = em["new_text"].(string)
			}
			if old != "" || nw != "" {
				edits = append(edits, struct{ OldText, NewText string }{old, nw})
			}
		}
	} else {
		old, _ := obj["oldText"].(string)
		if old == "" {
			old, _ = obj["old_text"].(string)
		}
		nw, _ := obj["newText"].(string)
		if nw == "" {
			nw, _ = obj["new_text"].(string)
		}
		if old != "" || nw != "" {
			edits = append(edits, struct{ OldText, NewText string }{old, nw})
		}
	}

	if len(edits) == 0 {
		return nil
	}

	var patches []*message.FilePatch
	for _, edit := range edits {
		oldLines := splitLines(edit.OldText)
		newLines := splitLines(edit.NewText)
		patch, err := difflib.GetUnifiedDiffString(difflib.UnifiedDiff{
			A:        oldLines,
			B:        newLines,
			FromFile: "a/" + path,
			ToFile:   "b/" + path,
			Context:  3,
		})
		if err != nil || patch == "" {
			continue
		}
		adds, dels := countDiffLines(patch)
		patches = append(patches, &message.FilePatch{
			Path:      path,
			Status:    "modified",
			Patch:     patch,
			Additions: adds,
			Deletions: dels,
		})
	}
	return patches
}

func countDiffLines(patch string) (adds, dels int) {
	for _, line := range strings.Split(patch, "\n") {
		if len(line) == 0 {
			continue
		}
		switch line[0] {
		case '+':
			if !strings.HasPrefix(line, "+++") {
				adds++
			}
		case '-':
			if !strings.HasPrefix(line, "---") {
				dels++
			}
		}
	}
	return
}

func splitLines(s string) []string {
	if s == "" {
		return []string{}
	}
	lines := strings.SplitAfter(s, "\n")
	if lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}
	return lines
}
