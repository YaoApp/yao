package task

import (
	"context"
	"errors"
	"fmt"
	"runtime/debug"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/yaoapp/gou/process"
	"github.com/yaoapp/kun/log"
	"github.com/yaoapp/xun/capsule"
	agentconfig "github.com/yaoapp/yao/agent/config"
	agentcontext "github.com/yaoapp/yao/agent/context"
	"github.com/yaoapp/yao/agent/output/message"
	"github.com/yaoapp/yao/event"
)

// AssistantStreamFn is the function pointer for assistant.Stream() to avoid circular imports.
// Injected via tools_bridge or init. Signature mirrors assistant.Assistant.Stream().
var AssistantStreamFn func(assistantID string, ctx *agentcontext.Context, msgs []agentcontext.Message, opts ...*agentcontext.Options) (*agentcontext.Response, error)

var errUserCancelled = fmt.Errorf("user_cancelled")

// Run starts or queues a task execution. Atomic operation (no LLM enrichment).
func Run(ctx context.Context, auth *process.AuthorizedInfo, chatID string, req *RunReq) (*RunResult, error) {
	task, err := Get(ctx, auth, chatID)
	if err != nil {
		return nil, err
	}

	dc := newDaemonContext(chatID)
	if _, loaded := daemonRegistry.LoadOrStore(chatID, dc); loaded {
		return nil, fmt.Errorf("task %s is already running or queued", chatID)
	}
	fmt.Printf("  • [task.run] daemon REGISTERED chatID=%s\n", chatID)

	now := time.Now()
	requestID := uuid.New().String()

	if !GlobalQuota.TryAcquire(auth.TeamID, "") {
		setStatus(chatID, "queued", auth, map[string]any{
			"queued_at":      now,
			"queue_priority": req.Priority,
		})
		entry := GlobalQuota.Enqueue(auth.TeamID, chatID, req.Priority)
		daemonWg.Add(1)
		go func() {
			defer daemonWg.Done()
			select {
			case <-entry.ready:
				defer GlobalQuota.Release(auth.TeamID)
				defer recoverDaemonPanic(dc)
				setStatus(chatID, "running", auth, nil)
				runDaemon(dc, auth, req, task)
			case <-dc.Context.Done():
				GlobalQuota.Dequeue(auth.TeamID, chatID)
				setStatus(chatID, "cancelled", auth, map[string]any{
					"cancelled_at":  time.Now(),
					"cancel_reason": "cancelled_while_queued",
				})
				dc.CloseSubscribers()
				UnregisterDaemon(chatID)
			}
		}()
		pos := GlobalQuota.QueuePosition(auth.TeamID, chatID)
		return &RunResult{ChatID: chatID, Status: "queued", RequestID: requestID, Position: pos}, nil
	}

	setStatus(chatID, "running", auth, map[string]any{"started_at": now})
	incrementRunCount(chatID)
	daemonWg.Add(1)
	go func() {
		defer daemonWg.Done()
		defer GlobalQuota.Release(auth.TeamID)
		defer recoverDaemonPanic(dc)
		runDaemon(dc, auth, req, task)
	}()
	return &RunResult{ChatID: chatID, Status: "running", RequestID: requestID}, nil
}

// Stop halts a running or queued task
func Stop(ctx context.Context, auth *process.AuthorizedInfo, chatID string, force bool) error {
	dc, exists := GetDaemon(chatID)
	if !exists {
		return fmt.Errorf("task %s is not running", chatID)
	}
	if force {
		dc.ForceCancel()
	} else {
		dc.Cancel()
	}
	return nil
}

// Input is deprecated. With single-round daemon, user input is provided via
// a new "run" command on the WS channel (which starts a fresh daemon).
func Input(ctx context.Context, auth *process.AuthorizedInfo, chatID string, req *InputReq) error {
	return fmt.Errorf("task %s: Input() is deprecated, use run command instead", chatID)
}

// SetPriority updates queue priority for a queued task
func SetPriority(ctx context.Context, auth *process.AuthorizedInfo, chatID string, priority int) error {
	_, err := Get(ctx, auth, chatID)
	if err != nil {
		return err
	}
	_, err = capsule.Global.Query().Table(tableTask()).
		Where("chat_id", "=", chatID).
		Update(map[string]interface{}{"queue_priority": priority, "updated_at": time.Now()})
	return err
}

// runDaemon is the single-round execution: delegates to assistant.Stream(), then triggers enrichTaskResult.
func runDaemon(dc *DaemonContext, auth *process.AuthorizedInfo, req *RunReq, task *Task) {
	defer UnregisterDaemon(dc.ChatID)
	defer dc.StopIdleTimer()

	daemonStart := time.Now()
	columnID := ""
	if task.ColumnID != nil {
		columnID = *task.ColumnID
	}

	if AssistantStreamFn == nil {
		markFailed(dc, auth, fmt.Errorf("AssistantStreamFn not initialized"))
		return
	}

	// Build agent context first so config.Get can read AssistantID/ChatID/Authorized.
	agentCtx := agentcontext.New(dc.Context, toOAuthInfo(auth), dc.ChatID)
	defer agentCtx.Release()
	agentCtx.AssistantID = task.AssistantID
	agentCtx.Writer = NewDaemonResponseWriter(dc)
	agentCtx.Accept = agentcontext.AcceptWebCUI
	agentCtx.Referer = "task"
	agentCtx.Locale = req.Locale

	// Load unified config via agent/config.
	resolved, cfgErr := agentconfig.Get(agentCtx)
	if cfgErr != nil {
		markFailed(dc, auth, fmt.Errorf("config.Get: %w", cfgErr))
		return
	}

	dur, _ := time.ParseDuration(resolved.Timeout)
	if dur == 0 {
		dur = 60 * time.Minute
	}
	stdCtx, cancel := context.WithTimeout(dc.Context, dur)
	agentCtx.Context = stdCtx
	defer func() {
		dc.CloseSubscribers()
		cancel()
	}()

	opts := &agentcontext.Options{
		Connector: resolved.Model,
		Metadata:  map[string]any{"max_turns": resolved.MaxTurns},
	}
	if req.Model != "" {
		opts.Connector = req.Model
	}
	// Only propagate workspace_id for sandbox binding — do NOT merge all WS
	// metadata keys (column_id, assistant_id, etc.) into opts.Metadata as they
	// pollute ctx.Metadata and can change sandbox identifier / behavior.
	if ws := metaString(req.Metadata, "workspace_id"); ws != "" {
		opts.Metadata["workspace_id"] = ws
	} else if task.LastWorkspace != nil && *task.LastWorkspace != "" {
		opts.Metadata["workspace_id"] = *task.LastWorkspace
	}

	// Fresh=true (retry): skip history loading so agent starts clean with only req.Messages
	if req.Fresh {
		opts.Skip = &agentcontext.Skip{History: true}
	}

	// Broadcast user messages to live subscribers (preserves multipart content)
	userMsgID := metaString(req.Metadata, "user_msg_id")
	for i, m := range req.Messages {
		if m.Role != "user" {
			continue
		}
		msgID := userMsgID
		if msgID == "" {
			msgID = fmt.Sprintf("user-%d", time.Now().UnixMilli())
		}
		if i > 0 {
			msgID = fmt.Sprintf("%s-%d", msgID, i)
		}
		userMsg := &message.Message{
			Type:      "user_input",
			MessageID: msgID,
			Props: map[string]interface{}{
				"content": m.Content,
				"role":    "user",
			},
		}
		dc.Broadcast(userMsg)
	}

	_, err := AssistantStreamFn(task.AssistantID, agentCtx, inputToAgentMessages(req.Messages), opts)

	var finalErr error
	status := "completed"

	if stdCtx.Err() == context.DeadlineExceeded {
		finalErr = fmt.Errorf("timeout after %s", resolved.Timeout)
		status = "failed"
		markFailed(dc, auth, finalErr)
	} else if dc.Context.Err() != nil && isContextCanceled(err) {
		status = "cancelled"
		finalErr = errUserCancelled
		dc.Broadcast(&message.Message{
			Type:  "text",
			Props: map[string]interface{}{"content": "\n\n---\n*[任务已被用户取消]*"},
		})
		setStatus(dc.ChatID, "cancelled", auth, map[string]any{
			"cancelled_at":  time.Now(),
			"cancel_reason": "user_stopped",
		})
	} else if err != nil {
		finalErr = err
		status = "failed"
		markFailed(dc, auth, finalErr)
	}

	dc.SetStatus(DaemonStopped)
	logTaskCompleted(dc.ChatID, columnID, task.AssistantID, status, time.Since(daemonStart), finalErr)

	isFirstRun := (task.RunCount <= 1)
	recentTexts := collectConversationText(req.Messages, dc)
	go enrichTaskResult(dc.ChatID, auth, isFirstRun, finalErr, recentTexts, req.Locale)
}

func collectConversationText(msgs []InputMessage, dc *DaemonContext) []string {
	var texts []string
	for _, m := range msgs {
		if m.Role == "user" {
			if s := contentText(m.Content); s != "" {
				texts = append(texts, fmt.Sprintf("[user] %s", s))
			}
		}
	}
	texts = append(texts, extractRecentText(dc, 20)...)
	return texts
}

func inputToAgentMessages(msgs []InputMessage) []agentcontext.Message {
	out := make([]agentcontext.Message, 0, len(msgs))
	for _, m := range msgs {
		out = append(out, agentcontext.Message{
			Role:    agentcontext.MessageRole(m.Role),
			Content: m.Content,
		})
	}
	return out
}

// setStatus updates the task run_status in DB and pushes a minimal event.
// Mail creation is now handled by enrichTaskResult after daemon exit.
func setStatus(chatID, status string, auth *process.AuthorizedInfo, extra map[string]any) {
	updates := map[string]interface{}{
		"run_status": status,
		"updated_at": time.Now(),
	}
	for k, v := range extra {
		updates[k] = v
	}
	capsule.Global.Query().Table(tableTask()).
		Where("chat_id", "=", chatID).
		Update(updates)

	eventData := map[string]any{
		"chat_id":    chatID,
		"run_status": status,
	}
	if auth != nil {
		eventData["__yao_team_id"] = auth.TeamID
	}
	event.Push(context.Background(), "task.updated", eventData)
}

func markFailed(dc *DaemonContext, auth *process.AuthorizedInfo, err error) {
	errMsg := err.Error()
	setStatus(dc.ChatID, "failed", auth, map[string]any{
		"error_message": errMsg,
		"completed_at":  time.Now(),
	})
}

// isContextCanceled checks if an error is caused by context cancellation
func isContextCanceled(err error) bool {
	if err == nil {
		return true // no error but context was canceled — still a cancel
	}
	if errors.Is(err, context.Canceled) {
		return true
	}
	errMsg := err.Error()
	return strings.Contains(errMsg, "context canceled") || strings.Contains(errMsg, "context deadline exceeded")
}

func recoverDaemonPanic(dc *DaemonContext) {
	if r := recover(); r != nil {
		log.Error("task %s daemon panic: %v\n%s", dc.ChatID, r, debug.Stack())
	}
}

func incrementRunCount(chatID string) {
	capsule.Global.Query().Table(tableTask()).
		Where("chat_id", "=", chatID).
		Increment("run_count", 1)
}

func getTaskBasicInfo(chatID string) (*Task, error) {
	row, err := capsule.Global.Query().Table(tableTask()+" as t").
		Select("t.column_id", "c.assistant_id").
		LeftJoin(tableChat()+" as c", "t.chat_id", "=", "c.chat_id").
		Where("t.chat_id", "=", chatID).
		First()
	if err != nil || row == nil {
		return nil, err
	}
	return &Task{
		ChatID:      chatID,
		AssistantID: getString(row, "assistant_id"),
		ColumnID:    getStringPtr(row, "column_id"),
	}, nil
}

func getStringVal(p *string) string {
	if p == nil {
		return ""
	}
	return *p
}
