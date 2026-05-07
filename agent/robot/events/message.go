package events

import (
	"context"
	"fmt"
	"strings"

	agent "github.com/yaoapp/yao/agent"
	"github.com/yaoapp/yao/agent/assistant"
	agentcontext "github.com/yaoapp/yao/agent/context"
	robotstore "github.com/yaoapp/yao/agent/robot/store"
	robottypes "github.com/yaoapp/yao/agent/robot/types"
	eventtypes "github.com/yaoapp/yao/event/types"
	oauthtypes "github.com/yaoapp/yao/openapi/oauth/types"
)

// handleMessage processes messages from external integrations (Telegram, etc.).
// It calls the Host Agent with the provided messages and returns a MessageResult.
// Action detection is done via the Host Agent's Next hook return value.
func (h *robotHandler) handleMessage(ctx context.Context, ev *eventtypes.Event, resp chan<- eventtypes.Result) {
	var payload MessagePayload
	if err := ev.Should(&payload); err != nil {
		log.Error("message handler: invalid payload: %v", err)
		if ev.IsCall {
			resp <- eventtypes.Result{Err: err}
		}
		return
	}

	log.Info("message handler: robot=%s channel=%s msg_id=%s",
		payload.RobotID, payload.Metadata.Channel, payload.Metadata.MessageID)

	result, err := callHostAgent(ctx, &payload)
	if err != nil {
		log.Error("message handler: host agent call failed robot=%s: %v", payload.RobotID, err)
		if ev.IsCall {
			resp <- eventtypes.Result{Err: err}
		}
		return
	}

	if reply := getReplyFunc(); reply != nil && result.Message != nil {
		if err := reply(ctx, result.Message, payload.Metadata); err != nil {
			log.Error("message handler: reply failed robot=%s channel=%s: %v",
				payload.RobotID, payload.Metadata.Channel, err)
		}
	}

	if ev.IsCall {
		resp <- eventtypes.Result{Data: result}
	}
}

// callHostAgent resolves the Host Agent for the robot and calls it with messages.
// This avoids importing executor/standard to prevent import cycles; instead it
// calls the assistant directly via assistant.Get + ast.Stream (same as AgentCaller.Call).
func callHostAgent(ctx context.Context, payload *MessagePayload) (*MessageResult, error) {
	hostID, record, err := resolveHostAssistantID(ctx, payload.RobotID)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve host agent: %w", err)
	}

	ast, err := assistant.Get(hostID)
	if err != nil {
		return nil, fmt.Errorf("assistant not found: %s: %w", hostID, err)
	}

	opts := &agentcontext.Options{
		Skip: &agentcontext.Skip{
			Search: false,
		},
	}

	authorized := &oauthtypes.AuthorizedInfo{
		UserID: payload.Metadata.SenderID,
		TeamID: record.TeamID,
	}
	chatID := fmt.Sprintf("%s:%s", payload.Metadata.Channel, payload.Metadata.ChatID)
	agentCtx := agentcontext.New(ctx, authorized, chatID)
	agentCtx.AssistantID = hostID
	agentCtx.Referer = "integration"
	agentCtx.Locale = payload.Metadata.Locale
	agentCtx.Metadata = map[string]interface{}{
		"robot_id": payload.RobotID,
		"channel":  payload.Metadata.Channel,
	}

	if dsl := agent.GetAgent(); dsl != nil {
		if cache, err := dsl.GetCacheStore(); err == nil {
			agentCtx.Cache = cache
		}
	}

	defer agentCtx.Release()

	response, err := ast.Stream(agentCtx, payload.Messages, opts)
	if err != nil {
		return nil, fmt.Errorf("host agent call failed: %w", err)
	}

	result := &MessageResult{
		Metadata: payload.Metadata,
	}

	if response.Completion != nil {
		result.Message = &agentcontext.Message{
			Role:    agentcontext.RoleAssistant,
			Content: response.Completion.Content,
		}
	}

	// Detect action from Next hook return value
	log.Debug("response.Next type=%T value=%+v", response.Next, response.Next)
	if action := detectAction(response.Next); action != nil {
		log.Info("action detected: name=%s payload=%+v", action.Name, action.Payload)
		result.Action = action
		if action.Name == "robot.execute" {
			if execID := executeAction(ctx, payload, record, action); execID != "" {
				result.ExecutionID = execID
				result.Message = &agentcontext.Message{
					Role:    agentcontext.RoleAssistant,
					Content: taskDeployedMessage(execID, payload.Metadata.Locale),
				}
			}
		}
	} else {
		log.Debug("no action detected from response.Next")
	}

	return result, nil
}

// executeAction triggers robot execution when the Host Agent returns a
// confirmed action. Uses the injected TriggerFunc to call robotapi.TriggerManual
// without creating a circular import.
func executeAction(ctx context.Context, payload *MessagePayload, record *robotstore.RobotRecord, action *ActionResult) string {
	trigger := getTriggerFunc()
	if trigger == nil {
		log.Warn("message handler: trigger func not registered, cannot execute action for robot=%s", payload.RobotID)
		return ""
	}

	data, _ := action.Payload.(map[string]interface{})
	goals, _ := data["goals"].(string)
	if goals == "" {
		log.Warn("message handler: confirmed action has no goals, robot=%s", payload.RobotID)
		return ""
	}

	triggerData := &robottypes.TriggerInput{
		Data: map[string]interface{}{
			"goals":   goals,
			"channel": payload.Metadata.Channel,
			"chat_id": payload.Metadata.ChatID,
			"extra":   payload.Metadata.Extra,
		},
	}

	authorized := &oauthtypes.AuthorizedInfo{
		UserID: record.MemberID,
		TeamID: record.TeamID,
	}
	rCtx := robottypes.NewContext(ctx, authorized)

	execID, accepted, err := trigger(rCtx, payload.RobotID, robottypes.TriggerHuman, triggerData)
	if err != nil {
		log.Error("message handler: execute action failed robot=%s: %v", payload.RobotID, err)
		return ""
	}
	if !accepted {
		log.Warn("message handler: execute action not accepted robot=%s", payload.RobotID)
		return ""
	}

	log.Info("message handler: execution triggered robot=%s exec_id=%s", payload.RobotID, execID)
	return execID
}

// detectAction checks the Next hook return value for a confirmed action.
// The Host Agent returns { data: { confirmed: true, robot_id: "...", goals: "..." } }
// when it detects a confirm_task tool call.
func detectAction(next interface{}) *ActionResult {
	if next == nil {
		return nil
	}

	m, ok := next.(map[string]interface{})
	if !ok {
		return nil
	}

	// Next hook may return { data: { confirmed, ... } } or flat { confirmed, ... }
	data, _ := m["data"].(map[string]interface{})
	if data == nil {
		data = m
	}

	confirmed, _ := data["confirmed"].(bool)
	if !confirmed {
		return nil
	}

	return &ActionResult{
		Name:    "robot.execute",
		Payload: data,
	}
}

// resolveHostAssistantID resolves the host assistant ID from a robot member ID.
// Mirrors the logic in openapi/agent/robot/completions.go.
func resolveHostAssistantID(ctx context.Context, memberID string) (string, *robotstore.RobotRecord, error) {
	store := robotstore.NewRobotStore()
	record, err := store.Get(ctx, memberID)
	if err != nil {
		return "", nil, fmt.Errorf("failed to get robot: %w", err)
	}
	if record == nil {
		return "", nil, fmt.Errorf("robot not found: %s", memberID)
	}

	config, err := robottypes.ParseConfig(record.RobotConfig)
	if err != nil {
		return "", nil, fmt.Errorf("failed to parse robot config: %w", err)
	}

	var hostID string
	if config != nil && config.Resources != nil {
		hostID = config.Resources.GetPhaseAgent(robottypes.PhaseHost)
	} else {
		hostID = "__yao." + string(robottypes.PhaseHost)
	}

	return hostID, record, nil
}

func taskDeployedMessage(execID string, locale string) string {
	if strings.HasPrefix(locale, "zh") {
		return fmt.Sprintf("任务已部署（执行编号: %s），完成后会将结果发送给你。", execID)
	}
	return fmt.Sprintf("Task deployed (execution: %s). You will receive results once completed.", execID)
}
