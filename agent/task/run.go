package task

import (
	"context"
	"fmt"
	"runtime/debug"
	"time"

	"github.com/google/uuid"
	"github.com/yaoapp/gou/process"
	"github.com/yaoapp/kun/log"
	"github.com/yaoapp/xun/capsule"
	agentcontext "github.com/yaoapp/yao/agent/context"
	"github.com/yaoapp/yao/agent/inbox"
	"github.com/yaoapp/yao/agent/output/message"
	"github.com/yaoapp/yao/event"
)

// AssistantStreamFn is the function pointer for assistant.Stream() to avoid circular imports.
// Injected via tools_bridge or init. Signature mirrors assistant.Assistant.Stream().
var AssistantStreamFn func(assistantID string, ctx *agentcontext.Context, msgs []agentcontext.Message, opts ...*agentcontext.Options) (*agentcontext.Response, error)

// Run starts or queues a task execution. Atomic operation (no LLM enrichment).
func Run(ctx context.Context, auth *process.AuthorizedInfo, chatID string, req *RunReq) (*RunResult, error) {
	task, err := Get(ctx, auth, chatID)
	if err != nil {
		return nil, err
	}

	cfg, err := GetConfig(ctx, auth, chatID)
	if err != nil {
		return nil, fmt.Errorf("task.Run: get config: %w", err)
	}

	dc := newDaemonContext(chatID)
	if _, loaded := daemonRegistry.LoadOrStore(chatID, dc); loaded {
		return nil, fmt.Errorf("task %s is already running or queued", chatID)
	}

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
				runDaemon(dc, auth, cfg, req, task)
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
		runDaemon(dc, auth, cfg, req, task)
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

// Input provides user input to a waiting task
func Input(ctx context.Context, auth *process.AuthorizedInfo, chatID string, req *InputReq) error {
	dc, exists := GetDaemon(chatID)
	if !exists {
		return fmt.Errorf("task %s is not running", chatID)
	}
	if dc.Status() != DaemonWaiting {
		return fmt.Errorf("task %s is not waiting for input", chatID)
	}

	// Broadcast user input to subscribers for display
	for _, m := range req.Messages {
		dc.Broadcast(&message.Message{
			Type: "text",
			Props: map[string]interface{}{
				"role":    m.Role,
				"content": m.Content,
			},
		})
	}

	select {
	case dc.inputCh <- req.Messages:
		return nil
	default:
		return fmt.Errorf("task %s input channel full", chatID)
	}
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

// runDaemon is the core execution loop: delegates to assistant.Stream()
func runDaemon(dc *DaemonContext, auth *process.AuthorizedInfo, cfg *Config, req *RunReq, task *Task) {
	defer UnregisterDaemon(dc.ChatID)
	defer dc.CloseSubscribers()
	defer func() { updateFinalStatus(dc, auth) }()

	if AssistantStreamFn == nil {
		markFailed(dc, auth, fmt.Errorf("AssistantStreamFn not initialized"))
		return
	}

	dur, _ := time.ParseDuration(cfg.Setting.Timeout)
	if dur == 0 {
		dur = 60 * time.Minute
	}
	stdCtx, cancel := context.WithTimeout(dc.Context, dur)
	defer cancel()

	agentCtx := agentcontext.New(stdCtx, toOAuthInfo(auth), dc.ChatID)
	defer agentCtx.Release()
	agentCtx.AssistantID = task.AssistantID
	agentCtx.Writer = NewDaemonResponseWriter(dc)
	agentCtx.Accept = agentcontext.AcceptWebCUI
	agentCtx.Referer = "task"

	opts := &agentcontext.Options{
		Connector: cfg.Setting.Model,
		Metadata:  map[string]any{"max_turns": cfg.Setting.MaxTurns},
	}

	maxTurns := cfg.Setting.MaxTurns
	if maxTurns <= 0 {
		maxTurns = 1
	}

	messages := inputToAgentMessages(req.Messages)
	var lastErr error

	for turn := 0; turn < maxTurns; turn++ {
		dc.resetIdleTimer()

		_, lastErr = AssistantStreamFn(task.AssistantID, agentCtx, messages, opts)
		if lastErr != nil {
			break
		}
		if stdCtx.Err() != nil {
			break
		}

		if dc.Status() == DaemonWaiting {
			setStatus(dc.ChatID, "waiting", auth, nil)
			select {
			case inputMsgs := <-dc.inputCh:
				messages = inputToAgentMessages(inputMsgs)
				dc.SetStatus(DaemonRunning)
				setStatus(dc.ChatID, "running", auth, nil)
			case <-stdCtx.Done():
				break
			}
		} else {
			break
		}
	}

	if stdCtx.Err() == context.DeadlineExceeded {
		markFailed(dc, auth, fmt.Errorf("timeout after %s", cfg.Setting.Timeout))
	} else if lastErr != nil {
		markFailed(dc, auth, lastErr)
	} else {
		markCompleted(dc, auth)
	}
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

	if status == "waiting" || status == "completed" || status == "failed" {
		inboxTask := &inbox.AgentTask{
			ChatID:    chatID,
			CreatedBy: auth.UserID,
			TeamID:    auth.TeamID,
		}
		if task, _ := getTaskBasicInfo(chatID); task != nil {
			inboxTask.AssistantID = task.AssistantID
			inboxTask.ColumnID = getStringVal(task.ColumnID)
		}
		mailID, _ := inbox.OnStatusChange(context.Background(), inboxTask, status)
		if mailID != "" {
			mailType := mailTypeFromStatus(status)
			enrichMailContent(mailID, chatID, mailType, auth)
		}
	}
}

func mailTypeFromStatus(status string) string {
	switch status {
	case "waiting":
		return "input"
	case "completed":
		return "completed"
	case "failed":
		return "failed"
	}
	return ""
}

func markCompleted(dc *DaemonContext, auth *process.AuthorizedInfo) {
	dc.SetStatus(DaemonStopped)
	setStatus(dc.ChatID, "completed", auth, map[string]any{"completed_at": time.Now()})
}

func markFailed(dc *DaemonContext, auth *process.AuthorizedInfo, err error) {
	dc.SetStatus(DaemonStopped)
	errMsg := err.Error()
	setStatus(dc.ChatID, "failed", auth, map[string]any{
		"error_message": errMsg,
		"completed_at":  time.Now(),
	})
}

func updateFinalStatus(dc *DaemonContext, auth *process.AuthorizedInfo) {
	if dc.Status() == DaemonRunning || dc.Status() == DaemonWaiting {
		markCompleted(dc, auth)
	}
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
