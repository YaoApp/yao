package manager

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/yaoapp/kun/log"
	agentcontext "github.com/yaoapp/yao/agent/context"
	"github.com/yaoapp/yao/agent/output/message"
	robotevents "github.com/yaoapp/yao/agent/robot/events"
	"github.com/yaoapp/yao/agent/robot/executor/standard"
	"github.com/yaoapp/yao/agent/robot/pool"
	"github.com/yaoapp/yao/agent/robot/store"
	"github.com/yaoapp/yao/agent/robot/types"
	"github.com/yaoapp/yao/agent/robot/utils"
	"github.com/yaoapp/yao/event"
)

// executeResume resumes a suspended execution using the Manager's shared executor.
// This avoids creating orphan Executor instances with independent counters.
func (m *Manager) executeResume(ctx *types.Context, execID, reply string) error {
	return m.executor.Resume(types.NewContext(ctx.Context, ctx.Auth), execID, reply)
}

// InteractRequest represents a unified interaction with a robot (Manager layer).
type InteractRequest struct {
	ExecutionID string               `json:"execution_id,omitempty"`
	TaskID      string               `json:"task_id,omitempty"`
	Source      types.InteractSource `json:"source,omitempty"`
	Message     string               `json:"message"`
	Action      string               `json:"action,omitempty"`
}

// InteractResponse is the result of an interaction.
type InteractResponse struct {
	ExecutionID string `json:"execution_id,omitempty"`
	Status      string `json:"status"`
	Message     string `json:"message,omitempty"`
	ChatID      string `json:"chat_id,omitempty"`
	Reply       string `json:"reply,omitempty"`
	WaitForMore bool   `json:"wait_for_more,omitempty"`
}

// CancelExecution cancels a waiting/confirming execution.
func (m *Manager) CancelExecution(ctx *types.Context, execID string) error {
	m.mu.RLock()
	if !m.started {
		m.mu.RUnlock()
		return fmt.Errorf("manager not started")
	}
	m.mu.RUnlock()

	execStore := store.NewExecutionStore()
	record, err := execStore.Get(ctx.Context, execID)
	if err != nil {
		return fmt.Errorf("execution not found: %s", execID)
	}
	if record == nil {
		return fmt.Errorf("execution not found: %s", execID)
	}

	if record.Status != types.ExecWaiting && record.Status != types.ExecConfirming {
		return fmt.Errorf("execution %s is in status %s, only waiting/confirming can be cancelled", execID, record.Status)
	}

	if err := execStore.UpdateStatus(ctx.Context, execID, types.ExecCancelled, "cancelled by user"); err != nil {
		return fmt.Errorf("failed to cancel execution: %w", err)
	}

	m.execController.Untrack(execID)
	if robot := m.cache.Get(record.MemberID); robot != nil {
		robot.RemoveExecution(execID)
	}

	event.Push(ctx.Context, robotevents.ExecCancelled, robotevents.ExecPayload{
		ExecutionID: execID,
		MemberID:    record.MemberID,
		TeamID:      record.TeamID,
		Status:      string(types.ExecCancelled),
		ChatID:      record.ChatID,
	})

	return nil
}

// HandleInteract processes all human-robot interactions through a unified entry point.
//
// Routing logic (§16.37):
//   - No execution_id: new interaction → createConfirmingExecution → Host Agent (assign)
//   - execution_id with status=confirming: Host Agent (assign) → processHostAction
//   - execution_id with status=waiting: Host Agent (clarify) → processHostAction
//   - execution_id with status=running: Host Agent (guide) → processHostAction
func (m *Manager) HandleInteract(ctx *types.Context, memberID string, req *InteractRequest) (*InteractResponse, error) {
	m.mu.RLock()
	if !m.started {
		m.mu.RUnlock()
		return nil, fmt.Errorf("manager not started")
	}
	m.mu.RUnlock()

	if memberID == "" {
		return nil, fmt.Errorf("member_id is required")
	}
	if req == nil || req.Message == "" {
		return nil, fmt.Errorf("message is required")
	}

	robot, _, err := m.getOrLoadRobot(ctx, memberID)
	if err != nil {
		return nil, fmt.Errorf("robot not found: %w", err)
	}

	execStore := store.NewExecutionStore()

	// No execution_id → create a new confirming execution
	if req.ExecutionID == "" {
		return m.handleNewInteraction(ctx, robot, req, execStore)
	}

	// Existing execution_id → load and route by status
	record, err := execStore.Get(ctx.Context, req.ExecutionID)
	if err != nil {
		return nil, fmt.Errorf("execution not found: %s", req.ExecutionID)
	}

	switch record.Status {
	case types.ExecConfirming:
		return m.handleConfirmingInteraction(ctx, robot, record, req, execStore)
	case types.ExecWaiting:
		return m.handleWaitingInteraction(ctx, robot, record, req, execStore)
	case types.ExecRunning:
		if record.WaitingTaskID == "" {
			return &InteractResponse{
				ExecutionID: record.ExecutionID,
				Status:      "rejected",
				Message:     "Execution is running and not waiting for input",
			}, nil
		}
		return m.handleRunningInteraction(ctx, robot, record, req, execStore)
	default:
		return nil, fmt.Errorf("execution %s is in status %s, cannot interact", req.ExecutionID, record.Status)
	}
}

// handleNewInteraction creates a confirming execution and calls Host Agent with "assign" scenario.
func (m *Manager) handleNewInteraction(ctx *types.Context, robot *types.Robot, req *InteractRequest, execStore *store.ExecutionStore) (*InteractResponse, error) {
	exec, chatID, err := m.createConfirmingExecution(ctx, robot, req, execStore)
	if err != nil {
		return nil, fmt.Errorf("failed to create confirming execution: %w", err)
	}

	hostOutput, err := m.callHostAgentForScenario(ctx, robot, "assign", req.Message, nil, chatID)
	if err != nil {
		log.Warn("Host Agent call failed, using direct assign: %v", err)
		return m.directAssign(ctx, robot, exec, req, execStore)
	}

	resp, err := m.processHostAction(ctx, robot, exec, hostOutput, execStore)
	if err != nil {
		return nil, err
	}
	resp.ExecutionID = exec.ExecutionID
	resp.ChatID = chatID
	return resp, nil
}

// handleConfirmingInteraction continues a confirming flow with Host Agent.
func (m *Manager) handleConfirmingInteraction(ctx *types.Context, robot *types.Robot, record *store.ExecutionRecord, req *InteractRequest, execStore *store.ExecutionStore) (*InteractResponse, error) {
	hostCtx := m.buildHostContext(robot, record, nil)
	hostOutput, err := m.callHostAgentForScenario(ctx, robot, "assign", req.Message, hostCtx, record.ChatID)
	if err != nil {
		log.Warn("Host Agent call failed during confirming: %v", err)
		return &InteractResponse{
			ExecutionID: record.ExecutionID,
			Status:      "error",
			Message:     fmt.Sprintf("Host Agent failed: %v", err),
		}, nil
	}

	resp, err := m.processHostAction(ctx, robot, record, hostOutput, execStore)
	if err != nil {
		return nil, err
	}
	resp.ExecutionID = record.ExecutionID
	resp.ChatID = record.ChatID
	return resp, nil
}

// handleWaitingInteraction processes input for a waiting (suspended) execution.
func (m *Manager) handleWaitingInteraction(ctx *types.Context, robot *types.Robot, record *store.ExecutionRecord, req *InteractRequest, execStore *store.ExecutionStore) (*InteractResponse, error) {
	waitingTask := m.findWaitingTask(record)
	hostCtx := m.buildHostContext(robot, record, waitingTask)

	hostOutput, err := m.callHostAgentForScenario(ctx, robot, "clarify", req.Message, hostCtx, record.ChatID)
	if err != nil {
		log.Warn("Host Agent call failed during clarify, falling back to direct resume: %v", err)
		return m.directResume(ctx, record, req)
	}

	resp, err := m.processHostAction(ctx, robot, record, hostOutput, execStore)
	if err != nil {
		return nil, err
	}
	resp.ExecutionID = record.ExecutionID
	resp.ChatID = record.ChatID
	return resp, nil
}

// handleRunningInteraction allows guidance for a running execution.
func (m *Manager) handleRunningInteraction(ctx *types.Context, robot *types.Robot, record *store.ExecutionRecord, req *InteractRequest, execStore *store.ExecutionStore) (*InteractResponse, error) {
	hostCtx := m.buildHostContext(robot, record, nil)
	hostOutput, err := m.callHostAgentForScenario(ctx, robot, "guide", req.Message, hostCtx, record.ChatID)
	if err != nil {
		return &InteractResponse{
			ExecutionID: record.ExecutionID,
			Status:      "acknowledged",
			Message:     "Guidance noted (Host Agent unavailable)",
		}, nil
	}

	resp, err := m.processHostAction(ctx, robot, record, hostOutput, execStore)
	if err != nil {
		return nil, err
	}
	resp.ExecutionID = record.ExecutionID
	resp.ChatID = record.ChatID
	return resp, nil
}

// ==================== Helper Methods ====================

// createConfirmingExecution creates a new execution in "confirming" status.
func (m *Manager) createConfirmingExecution(ctx *types.Context, robot *types.Robot, req *InteractRequest, execStore *store.ExecutionStore) (*store.ExecutionRecord, string, error) {
	execID := pool.GenerateExecID()
	chatID := fmt.Sprintf("robot_%s_%s", robot.MemberID, execID)
	now := time.Now()

	record := &store.ExecutionRecord{
		ExecutionID: execID,
		MemberID:    robot.MemberID,
		TeamID:      robot.TeamID,
		TriggerType: types.TriggerHuman,
		Status:      types.ExecConfirming,
		Phase:       types.PhaseGoals,
		ChatID:      chatID,
		Input: &types.TriggerInput{
			Action:   types.ActionTaskAdd,
			Messages: []agentcontext.Message{{Role: "user", Content: req.Message}},
			UserID:   ctx.UserID(),
		},
		StartTime: &now,
	}

	if err := execStore.Save(ctx.Context, record); err != nil {
		return nil, "", fmt.Errorf("failed to save confirming execution: %w", err)
	}

	return record, chatID, nil
}

// buildHostContext builds the HostContext for Host Agent calls.
func (m *Manager) buildHostContext(robot *types.Robot, record *store.ExecutionRecord, waitingTask *types.Task) *types.HostContext {
	hostCtx := &types.HostContext{
		RobotStatus: m.buildRobotStatusSnapshot(robot),
	}
	if record.Goals != nil {
		hostCtx.Goals = record.Goals
	}
	if len(record.Tasks) > 0 {
		hostCtx.Tasks = record.Tasks
	}
	if waitingTask != nil {
		hostCtx.CurrentTask = waitingTask
	}
	if record.WaitingQuestion != "" {
		hostCtx.AgentReply = record.WaitingQuestion
	}
	return hostCtx
}

// buildRobotStatusSnapshot builds a status snapshot for the Host Agent.
func (m *Manager) buildRobotStatusSnapshot(robot *types.Robot) *types.RobotStatusSnapshot {
	if robot == nil {
		return nil
	}
	snapshot := &types.RobotStatusSnapshot{
		MemberID:     robot.MemberID,
		Status:       robot.Status,
		ActiveCount:  robot.ActiveCount(),
		WaitingCount: robot.WaitingCount(),
		MaxQuota:     robot.MaxQuota(),
		ActiveExecs:  robot.ListExecutionBriefs(),
	}
	if m.pool != nil {
		snapshot.QueuedCount = m.pool.QueueSize()
	}
	return snapshot
}

// findWaitingTask finds the task that is currently waiting for input.
func (m *Manager) findWaitingTask(record *store.ExecutionRecord) *types.Task {
	if record.WaitingTaskID == "" {
		return nil
	}
	for i := range record.Tasks {
		if record.Tasks[i].ID == record.WaitingTaskID {
			return &record.Tasks[i]
		}
	}
	return nil
}

// callHostAgentForScenario calls the Host Agent with a given scenario.
func (m *Manager) callHostAgentForScenario(ctx *types.Context, robot *types.Robot, scenario string, message string, hostCtx *types.HostContext, chatID string) (*types.HostOutput, error) {
	agentID := ""
	if robot.Config != nil && robot.Config.Resources != nil {
		agentID = robot.Config.Resources.GetPhaseAgent(types.PhaseHost)
	}
	if agentID == "" {
		return nil, fmt.Errorf("no Host Agent configured for robot %s", robot.MemberID)
	}

	return m.callHostAgent(ctx, agentID, &types.HostInput{
		Scenario: scenario,
		Messages: []agentcontext.Message{{Role: "user", Content: message}},
		Context:  hostCtx,
	}, chatID, robot)
}

// callHostAgent calls the Host Agent assistant and parses output.
func (m *Manager) callHostAgent(ctx *types.Context, agentID string, input *types.HostInput, chatID string, robot *types.Robot) (*types.HostOutput, error) {
	inputJSON, err := json.Marshal(input)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal host input: %w", err)
	}

	caller := standard.NewConversationCaller(chatID)
	caller.Connector = robot.LanguageModel
	caller.Workspace = robot.Workspace
	result, err := caller.CallWithMessages(ctx, agentID, string(inputJSON))
	if err != nil {
		return nil, fmt.Errorf("host agent (%s) call failed: %w", agentID, err)
	}

	return m.parseHostAgentResult(result)
}

// parseHostAgentResult inspects the agent result to determine if it is an action
// decision (JSON with "action" field) or a conversational reply (natural language).
func (m *Manager) parseHostAgentResult(result *standard.CallResult) (*types.HostOutput, error) {
	data, err := result.GetJSON()
	if err == nil {
		output := &types.HostOutput{}
		raw, _ := json.Marshal(data)
		if err := json.Unmarshal(raw, output); err == nil && output.Action != "" {
			return output, nil
		}
	}

	return &types.HostOutput{
		Reply:       result.GetText(),
		WaitForMore: true,
	}, nil
}

// processHostAction processes the output from Host Agent and takes the appropriate action.
func (m *Manager) processHostAction(ctx *types.Context, robot *types.Robot, record *store.ExecutionRecord, output *types.HostOutput, execStore *store.ExecutionStore) (*InteractResponse, error) {
	resp := &InteractResponse{
		Reply:       output.Reply,
		WaitForMore: output.WaitForMore,
	}

	if output.WaitForMore {
		resp.Status = "waiting_for_more"
		resp.Message = output.Reply
		return resp, nil
	}

	switch output.Action {
	case types.HostActionConfirm:
		if err := m.advanceExecution(ctx, robot, record, execStore); err != nil {
			return nil, fmt.Errorf("failed to advance execution: %w", err)
		}
		resp.Status = "confirmed"
		resp.Message = "Execution confirmed and started"

	case types.HostActionAdjust:
		if err := m.adjustExecution(ctx, record, output.ActionData, execStore); err != nil {
			return nil, fmt.Errorf("failed to adjust execution: %w", err)
		}
		resp.Status = "adjusted"
		resp.Message = "Execution plan adjusted"

	case types.HostActionAddTask:
		if err := m.injectTask(ctx, record, output.ActionData, execStore); err != nil {
			return nil, fmt.Errorf("failed to inject task: %w", err)
		}
		resp.Status = "task_added"
		resp.Message = "New task injected"

	case types.HostActionSkip:
		if err := m.skipWaitingTask(ctx, record, execStore); err != nil {
			return nil, fmt.Errorf("failed to skip task: %w", err)
		}
		resp.Status = "task_skipped"
		resp.Message = "Waiting task skipped"

	case types.HostActionInjectCtx:
		if err := m.resumeWithContext(ctx, record, output.ActionData, execStore); err != nil {
			if err == types.ErrExecutionSuspended {
				resp.Status = "waiting"
				resp.Message = "Execution suspended again"
				return resp, nil
			}
			return nil, fmt.Errorf("failed to resume with context: %w", err)
		}
		resp.Status = "resumed"
		resp.Message = "Execution resumed with additional context"

	case types.HostActionCancel:
		if err := m.CancelExecution(ctx, record.ExecutionID); err != nil {
			return nil, fmt.Errorf("failed to cancel execution: %w", err)
		}
		resp.Status = "cancelled"
		resp.Message = "Execution cancelled"

	default:
		resp.Status = "acknowledged"
		resp.Message = output.Reply
	}

	return resp, nil
}

// advanceExecution moves a confirming execution to running.
func (m *Manager) advanceExecution(ctx *types.Context, robot *types.Robot, record *store.ExecutionRecord, execStore *store.ExecutionStore) error {
	if err := execStore.UpdateStatus(ctx.Context, record.ExecutionID, types.ExecRunning, ""); err != nil {
		return err
	}

	ctrlExec := m.execController.Track(record.ExecutionID, record.MemberID, record.TeamID)
	execCtx := types.NewContext(ctrlExec.Context(), ctx.Auth)

	triggerInput := record.Input
	_, err := m.pool.SubmitWithID(execCtx, robot, types.TriggerHuman, triggerInput, record.ExecutionID, ctrlExec)
	if err != nil {
		m.execController.Untrack(record.ExecutionID)
		return fmt.Errorf("failed to submit execution to pool: %w", err)
	}

	return nil
}

// adjustExecution adjusts goals/tasks based on Host Agent output.
func (m *Manager) adjustExecution(ctx *types.Context, record *store.ExecutionRecord, actionData interface{}, execStore *store.ExecutionStore) error {
	if actionData == nil {
		return nil
	}

	data, ok := actionData.(map[string]interface{})
	if !ok {
		raw, err := json.Marshal(actionData)
		if err != nil {
			return nil
		}
		json.Unmarshal(raw, &data)
	}

	if goalsContent, ok := data["goals"].(string); ok && goalsContent != "" {
		record.Goals = &types.Goals{Content: goalsContent}
	}

	if tasksRaw, ok := data["tasks"]; ok {
		raw, _ := json.Marshal(tasksRaw)
		var tasks []types.Task
		if err := json.Unmarshal(raw, &tasks); err == nil {
			record.Tasks = tasks
		}
	}

	return execStore.Save(ctx.Context, record)
}

// injectTask adds a new task to the execution's task list.
func (m *Manager) injectTask(ctx *types.Context, record *store.ExecutionRecord, actionData interface{}, execStore *store.ExecutionStore) error {
	if actionData == nil {
		return fmt.Errorf("task data is required")
	}

	raw, err := json.Marshal(actionData)
	if err != nil {
		return fmt.Errorf("invalid task data: %w", err)
	}

	var newTask types.Task
	if err := json.Unmarshal(raw, &newTask); err != nil {
		return fmt.Errorf("failed to parse task: %w", err)
	}

	if newTask.ID == "" {
		newTask.ID = fmt.Sprintf("injected-%s", utils.NewID()[:8])
	}
	newTask.Status = types.TaskPending

	record.Tasks = append(record.Tasks, newTask)
	return execStore.Save(ctx.Context, record)
}

// skipWaitingTask skips the currently waiting task and resumes execution.
func (m *Manager) skipWaitingTask(ctx *types.Context, record *store.ExecutionRecord, execStore *store.ExecutionStore) error {
	if record.WaitingTaskID == "" {
		return fmt.Errorf("no task is waiting")
	}

	for i := range record.Tasks {
		if record.Tasks[i].ID == record.WaitingTaskID {
			record.Tasks[i].Status = types.TaskSkipped
			break
		}
	}

	err := m.executeResume(ctx, record.ExecutionID, "__skip__")
	if err != nil && err != types.ErrExecutionSuspended {
		return fmt.Errorf("failed to resume after skip: %w", err)
	}
	return nil
}

// resumeWithContext injects context and resumes the waiting execution.
func (m *Manager) resumeWithContext(ctx *types.Context, record *store.ExecutionRecord, actionData interface{}, execStore *store.ExecutionStore) error {
	reply := ""
	if actionData != nil {
		if s, ok := actionData.(string); ok {
			reply = s
		} else if data, ok := actionData.(map[string]interface{}); ok {
			if r, ok := data["reply"].(string); ok {
				reply = r
			} else {
				raw, _ := json.Marshal(data)
				reply = string(raw)
			}
		}
	}

	return m.executeResume(ctx, record.ExecutionID, reply)
}

// directAssign is the fallback when Host Agent is unavailable: directly start execution.
func (m *Manager) directAssign(ctx *types.Context, robot *types.Robot, record *store.ExecutionRecord, req *InteractRequest, execStore *store.ExecutionStore) (*InteractResponse, error) {
	if err := m.advanceExecution(ctx, robot, record, execStore); err != nil {
		return nil, fmt.Errorf("direct assign failed: %w", err)
	}
	return &InteractResponse{
		ExecutionID: record.ExecutionID,
		Status:      "confirmed",
		Message:     "Execution started (direct assign)",
		ChatID:      record.ChatID,
	}, nil
}

// directResume is the fallback when Host Agent is unavailable: directly resume.
func (m *Manager) directResume(ctx *types.Context, record *store.ExecutionRecord, req *InteractRequest) (*InteractResponse, error) {
	err := m.executeResume(ctx, record.ExecutionID, req.Message)
	if err != nil {
		if err == types.ErrExecutionSuspended {
			return &InteractResponse{
				ExecutionID: record.ExecutionID,
				Status:      "waiting",
				Message:     "Execution suspended again: needs more input",
				ChatID:      record.ChatID,
			}, nil
		}
		return nil, fmt.Errorf("failed to resume execution: %w", err)
	}
	return &InteractResponse{
		ExecutionID: record.ExecutionID,
		Status:      "resumed",
		Message:     "Execution resumed and completed successfully",
		ChatID:      record.ChatID,
	}, nil
}

// ==================== Streaming Interact ====================

// HandleInteractStream is the streaming version of HandleInteract.
// It streams Host Agent text tokens via streamFn while still returning the final InteractResponse.
func (m *Manager) HandleInteractStream(ctx *types.Context, memberID string, req *InteractRequest, streamFn standard.StreamCallback) (*InteractResponse, error) {
	m.mu.RLock()
	if !m.started {
		m.mu.RUnlock()
		return nil, fmt.Errorf("manager not started")
	}
	m.mu.RUnlock()

	if memberID == "" {
		return nil, fmt.Errorf("member_id is required")
	}
	if req == nil || req.Message == "" {
		return nil, fmt.Errorf("message is required")
	}

	robot, _, err := m.getOrLoadRobot(ctx, memberID)
	if err != nil {
		return nil, fmt.Errorf("robot not found: %w", err)
	}

	execStore := store.NewExecutionStore()

	if req.ExecutionID == "" {
		return m.handleNewInteractionStream(ctx, robot, req, execStore, streamFn)
	}

	record, err := execStore.Get(ctx.Context, req.ExecutionID)
	if err != nil {
		return nil, fmt.Errorf("execution not found: %s", req.ExecutionID)
	}

	switch record.Status {
	case types.ExecConfirming:
		return m.handleConfirmingInteractionStream(ctx, robot, record, req, execStore, streamFn)
	case types.ExecWaiting:
		return m.handleWaitingInteractionStream(ctx, robot, record, req, execStore, streamFn)
	case types.ExecRunning:
		if record.WaitingTaskID == "" {
			return &InteractResponse{
				ExecutionID: record.ExecutionID,
				Status:      "rejected",
				Message:     "Execution is running and not waiting for input",
			}, nil
		}
		return m.handleRunningInteractionStream(ctx, robot, record, req, execStore, streamFn)
	default:
		return nil, fmt.Errorf("execution %s is in status %s, cannot interact", req.ExecutionID, record.Status)
	}
}

func (m *Manager) handleNewInteractionStream(ctx *types.Context, robot *types.Robot, req *InteractRequest, execStore *store.ExecutionStore, streamFn standard.StreamCallback) (*InteractResponse, error) {
	exec, chatID, err := m.createConfirmingExecution(ctx, robot, req, execStore)
	if err != nil {
		return nil, fmt.Errorf("failed to create confirming execution: %w", err)
	}

	hostOutput, err := m.callHostAgentForScenarioStream(ctx, robot, "assign", req.Message, nil, chatID, streamFn)
	if err != nil {
		log.Warn("Host Agent call failed, using direct assign: %v", err)
		return m.directAssign(ctx, robot, exec, req, execStore)
	}

	resp, err := m.processHostAction(ctx, robot, exec, hostOutput, execStore)
	if err != nil {
		return nil, err
	}
	resp.ExecutionID = exec.ExecutionID
	resp.ChatID = chatID
	return resp, nil
}

func (m *Manager) handleConfirmingInteractionStream(ctx *types.Context, robot *types.Robot, record *store.ExecutionRecord, req *InteractRequest, execStore *store.ExecutionStore, streamFn standard.StreamCallback) (*InteractResponse, error) {
	hostCtx := m.buildHostContext(robot, record, nil)
	hostOutput, err := m.callHostAgentForScenarioStream(ctx, robot, "assign", req.Message, hostCtx, record.ChatID, streamFn)
	if err != nil {
		log.Warn("Host Agent call failed during confirming: %v", err)
		return &InteractResponse{
			ExecutionID: record.ExecutionID,
			Status:      "error",
			Message:     fmt.Sprintf("Host Agent failed: %v", err),
		}, nil
	}

	resp, err := m.processHostAction(ctx, robot, record, hostOutput, execStore)
	if err != nil {
		return nil, err
	}
	resp.ExecutionID = record.ExecutionID
	resp.ChatID = record.ChatID
	return resp, nil
}

func (m *Manager) handleWaitingInteractionStream(ctx *types.Context, robot *types.Robot, record *store.ExecutionRecord, req *InteractRequest, execStore *store.ExecutionStore, streamFn standard.StreamCallback) (*InteractResponse, error) {
	waitingTask := m.findWaitingTask(record)
	hostCtx := m.buildHostContext(robot, record, waitingTask)

	hostOutput, err := m.callHostAgentForScenarioStream(ctx, robot, "clarify", req.Message, hostCtx, record.ChatID, streamFn)
	if err != nil {
		log.Warn("Host Agent call failed during clarify, falling back to direct resume: %v", err)
		return m.directResume(ctx, record, req)
	}

	resp, err := m.processHostAction(ctx, robot, record, hostOutput, execStore)
	if err != nil {
		return nil, err
	}
	resp.ExecutionID = record.ExecutionID
	resp.ChatID = record.ChatID
	return resp, nil
}

func (m *Manager) handleRunningInteractionStream(ctx *types.Context, robot *types.Robot, record *store.ExecutionRecord, req *InteractRequest, execStore *store.ExecutionStore, streamFn standard.StreamCallback) (*InteractResponse, error) {
	hostCtx := m.buildHostContext(robot, record, nil)
	hostOutput, err := m.callHostAgentForScenarioStream(ctx, robot, "guide", req.Message, hostCtx, record.ChatID, streamFn)
	if err != nil {
		return &InteractResponse{
			ExecutionID: record.ExecutionID,
			Status:      "acknowledged",
			Message:     "Guidance noted (Host Agent unavailable)",
		}, nil
	}

	resp, err := m.processHostAction(ctx, robot, record, hostOutput, execStore)
	if err != nil {
		return nil, err
	}
	resp.ExecutionID = record.ExecutionID
	resp.ChatID = record.ChatID
	return resp, nil
}

func (m *Manager) callHostAgentForScenarioStream(ctx *types.Context, robot *types.Robot, scenario string, msg string, hostCtx *types.HostContext, chatID string, streamFn standard.StreamCallback) (*types.HostOutput, error) {
	agentID := ""
	if robot.Config != nil && robot.Config.Resources != nil {
		agentID = robot.Config.Resources.GetPhaseAgent(types.PhaseHost)
	}
	if agentID == "" {
		return nil, fmt.Errorf("no Host Agent configured for robot %s", robot.MemberID)
	}

	return m.callHostAgentStream(ctx, agentID, &types.HostInput{
		Scenario: scenario,
		Messages: []agentcontext.Message{{Role: "user", Content: msg}},
		Context:  hostCtx,
	}, chatID, robot, streamFn)
}

func (m *Manager) callHostAgentStream(ctx *types.Context, agentID string, input *types.HostInput, chatID string, robot *types.Robot, streamFn standard.StreamCallback) (*types.HostOutput, error) {
	inputJSON, err := json.Marshal(input)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal host input: %w", err)
	}

	caller := standard.NewConversationCaller(chatID)
	caller.Connector = robot.LanguageModel
	caller.Workspace = robot.Workspace
	result, err := caller.CallWithMessagesStream(ctx, agentID, string(inputJSON), streamFn)
	if err != nil {
		return nil, fmt.Errorf("host agent (%s) call failed: %w", agentID, err)
	}

	return m.parseHostAgentResult(result)
}

// ==================== Raw Message Streaming (CUI Protocol) ====================

// HandleInteractStreamRaw is the CUI-protocol-aligned streaming version of HandleInteract.
// It passes raw message.Message objects directly to the onMessage callback, preserving all
// CUI protocol fields for direct SSE passthrough to the frontend.
func (m *Manager) HandleInteractStreamRaw(ctx *types.Context, memberID string, req *InteractRequest, onMessage agentcontext.OnMessageFunc) (*InteractResponse, error) {
	m.mu.RLock()
	if !m.started {
		m.mu.RUnlock()
		return nil, fmt.Errorf("manager not started")
	}
	m.mu.RUnlock()

	if memberID == "" {
		return nil, fmt.Errorf("member_id is required")
	}
	if req == nil || req.Message == "" {
		return nil, fmt.Errorf("message is required")
	}

	robot, _, err := m.getOrLoadRobot(ctx, memberID)
	if err != nil {
		return nil, fmt.Errorf("robot not found: %w", err)
	}

	execStore := store.NewExecutionStore()

	if req.ExecutionID == "" {
		return m.handleNewInteractionStreamRaw(ctx, robot, req, execStore, onMessage)
	}

	record, err := execStore.Get(ctx.Context, req.ExecutionID)
	if err != nil {
		return nil, fmt.Errorf("execution not found: %s", req.ExecutionID)
	}

	switch record.Status {
	case types.ExecConfirming:
		return m.handleConfirmingInteractionStreamRaw(ctx, robot, record, req, execStore, onMessage)
	case types.ExecWaiting:
		return m.handleWaitingInteractionStreamRaw(ctx, robot, record, req, execStore, onMessage)
	case types.ExecRunning:
		if record.WaitingTaskID == "" {
			return &InteractResponse{
				ExecutionID: record.ExecutionID,
				Status:      "rejected",
				Message:     "Execution is running and not waiting for input",
			}, nil
		}
		return m.handleRunningInteractionStreamRaw(ctx, robot, record, req, execStore, onMessage)
	default:
		return nil, fmt.Errorf("execution %s is in status %s, cannot interact", req.ExecutionID, record.Status)
	}
}

func (m *Manager) handleNewInteractionStreamRaw(ctx *types.Context, robot *types.Robot, req *InteractRequest, execStore *store.ExecutionStore, onMessage agentcontext.OnMessageFunc) (*InteractResponse, error) {
	exec, chatID, err := m.createConfirmingExecution(ctx, robot, req, execStore)
	if err != nil {
		return nil, fmt.Errorf("failed to create confirming execution: %w", err)
	}

	hostOutput, err := m.callHostAgentForScenarioStreamRaw(ctx, robot, "assign", req.Message, nil, chatID, onMessage)
	if err != nil {
		log.Warn("Host Agent call failed, using direct assign: %v", err)
		return m.directAssign(ctx, robot, exec, req, execStore)
	}

	resp, err := m.processHostAction(ctx, robot, exec, hostOutput, execStore)
	if err != nil {
		return nil, err
	}
	resp.ExecutionID = exec.ExecutionID
	resp.ChatID = chatID
	return resp, nil
}

func (m *Manager) handleConfirmingInteractionStreamRaw(ctx *types.Context, robot *types.Robot, record *store.ExecutionRecord, req *InteractRequest, execStore *store.ExecutionStore, onMessage agentcontext.OnMessageFunc) (*InteractResponse, error) {
	hostCtx := m.buildHostContext(robot, record, nil)
	hostOutput, err := m.callHostAgentForScenarioStreamRaw(ctx, robot, "assign", req.Message, hostCtx, record.ChatID, onMessage)
	if err != nil {
		log.Warn("Host Agent call failed during confirming: %v", err)
		return &InteractResponse{
			ExecutionID: record.ExecutionID,
			Status:      "error",
			Message:     fmt.Sprintf("Host Agent failed: %v", err),
		}, nil
	}

	resp, err := m.processHostAction(ctx, robot, record, hostOutput, execStore)
	if err != nil {
		return nil, err
	}
	resp.ExecutionID = record.ExecutionID
	resp.ChatID = record.ChatID
	return resp, nil
}

func (m *Manager) handleWaitingInteractionStreamRaw(ctx *types.Context, robot *types.Robot, record *store.ExecutionRecord, req *InteractRequest, execStore *store.ExecutionStore, onMessage agentcontext.OnMessageFunc) (*InteractResponse, error) {
	waitingTask := m.findWaitingTask(record)
	hostCtx := m.buildHostContext(robot, record, waitingTask)

	hostOutput, err := m.callHostAgentForScenarioStreamRaw(ctx, robot, "clarify", req.Message, hostCtx, record.ChatID, onMessage)
	if err != nil {
		log.Warn("Host Agent call failed during clarify, falling back to direct resume: %v", err)
		return m.directResume(ctx, record, req)
	}

	resp, err := m.processHostAction(ctx, robot, record, hostOutput, execStore)
	if err != nil {
		return nil, err
	}
	resp.ExecutionID = record.ExecutionID
	resp.ChatID = record.ChatID
	return resp, nil
}

func (m *Manager) handleRunningInteractionStreamRaw(ctx *types.Context, robot *types.Robot, record *store.ExecutionRecord, req *InteractRequest, execStore *store.ExecutionStore, onMessage agentcontext.OnMessageFunc) (*InteractResponse, error) {
	hostCtx := m.buildHostContext(robot, record, nil)
	hostOutput, err := m.callHostAgentForScenarioStreamRaw(ctx, robot, "guide", req.Message, hostCtx, record.ChatID, onMessage)
	if err != nil {
		return &InteractResponse{
			ExecutionID: record.ExecutionID,
			Status:      "acknowledged",
			Message:     "Guidance noted (Host Agent unavailable)",
		}, nil
	}

	resp, err := m.processHostAction(ctx, robot, record, hostOutput, execStore)
	if err != nil {
		return nil, err
	}
	resp.ExecutionID = record.ExecutionID
	resp.ChatID = record.ChatID
	return resp, nil
}

func (m *Manager) callHostAgentForScenarioStreamRaw(ctx *types.Context, robot *types.Robot, scenario string, msg string, hostCtx *types.HostContext, chatID string, onMessage agentcontext.OnMessageFunc) (*types.HostOutput, error) {
	agentID := ""
	if robot.Config != nil && robot.Config.Resources != nil {
		agentID = robot.Config.Resources.GetPhaseAgent(types.PhaseHost)
	}
	if agentID == "" {
		return nil, fmt.Errorf("no Host Agent configured for robot %s", robot.MemberID)
	}

	return m.callHostAgentStreamRaw(ctx, agentID, &types.HostInput{
		Scenario: scenario,
		Messages: []agentcontext.Message{{Role: "user", Content: msg}},
		Context:  hostCtx,
	}, chatID, robot, onMessage)
}

// callHostAgentStreamRaw calls the Host Agent with CUI raw message streaming.
// It buffers text chunks that look like JSON output (starting with "{" or "```json")
// so the frontend never sees raw decision JSON. If the final result is a decision,
// the buffered chunks are discarded and a clean reply is sent instead. If the
// result is a normal conversation turn, buffered chunks are flushed through.
func (m *Manager) callHostAgentStreamRaw(ctx *types.Context, agentID string, input *types.HostInput, chatID string, robot *types.Robot, onMessage agentcontext.OnMessageFunc) (*types.HostOutput, error) {
	inputJSON, err := json.Marshal(input)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal host input: %w", err)
	}

	var (
		bufferedChunks  []*message.Message
		buffering       bool
		accumulatedText string
		lastTextMsgID   string
	)

	wrappedOnMessage := func(msg *message.Message) int {
		if msg == nil {
			return onMessage(msg)
		}

		// Only intercept text type messages with delta content
		if msg.Type != message.TypeText || !msg.Delta {
			return onMessage(msg)
		}

		if msg.MessageID != "" {
			lastTextMsgID = msg.MessageID
		}

		// Extract the text content from this chunk
		chunkText := ""
		if msg.Props != nil {
			if c, ok := msg.Props["content"].(string); ok {
				chunkText = c
			}
		}
		accumulatedText += chunkText

		// Decide whether to buffer: check accumulated text so far
		trimmed := strings.TrimSpace(accumulatedText)
		if !buffering && len(trimmed) > 0 {
			if trimmed[0] == '{' || strings.HasPrefix(trimmed, "```") {
				buffering = true
			}
		}

		if buffering {
			bufferedChunks = append(bufferedChunks, msg)
			return 0
		}

		return onMessage(msg)
	}

	caller := standard.NewConversationCaller(chatID)
	caller.Connector = robot.LanguageModel
	caller.Workspace = robot.Workspace
	result, err := caller.CallWithMessagesStreamRaw(ctx, agentID, string(inputJSON), wrappedOnMessage)
	if err != nil {
		return nil, fmt.Errorf("host agent (%s) call failed: %w", agentID, err)
	}

	output, err := m.parseHostAgentResult(result)
	if err != nil {
		return nil, err
	}

	if output.Action != "" && lastTextMsgID != "" {
		// Decision detected — discard buffered JSON chunks, send reply text
		onMessage(&message.Message{
			Type:      message.TypeText,
			MessageID: lastTextMsgID,
			Props:     map[string]interface{}{"content": output.Reply},
			Delta:     false,
		})
	} else if len(bufferedChunks) > 0 {
		// Not a decision — flush all buffered chunks to the frontend
		for _, chunk := range bufferedChunks {
			if onMessage(chunk) != 0 {
				break
			}
		}
	}

	return output, nil
}
