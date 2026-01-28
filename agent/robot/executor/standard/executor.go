package standard

import (
	"fmt"
	"strings"
	"sync/atomic"
	"time"

	"github.com/yaoapp/kun/log"
	"github.com/yaoapp/yao/agent/robot/executor/types"
	"github.com/yaoapp/yao/agent/robot/store"
	robottypes "github.com/yaoapp/yao/agent/robot/types"
	"github.com/yaoapp/yao/agent/robot/utils"
)

// Executor implements the standard executor with real Agent calls
// This is the production executor that:
// - Persists execution history to database
// - Calls real Agents via Assistant.Stream()
// - Logs phase transitions and errors using kun/log
type Executor struct {
	config       types.Config
	store        *store.ExecutionStore
	robotStore   *store.RobotStore
	execCount    atomic.Int32
	currentCount atomic.Int32
	onStart      func()
	onEnd        func()
}

// New creates a new standard executor
func New() *Executor {
	return &Executor{
		store:      store.NewExecutionStore(),
		robotStore: store.NewRobotStore(),
	}
}

// NewWithConfig creates a new standard executor with configuration
func NewWithConfig(config types.Config) *Executor {
	return &Executor{
		config:     config,
		store:      store.NewExecutionStore(),
		robotStore: store.NewRobotStore(),
	}
}

// Execute runs a robot through all applicable phases with real Agent calls (auto-generates ID)
func (e *Executor) Execute(ctx *robottypes.Context, robot *robottypes.Robot, trigger robottypes.TriggerType, data interface{}) (*robottypes.Execution, error) {
	return e.ExecuteWithControl(ctx, robot, trigger, data, "", nil)
}

// ExecuteWithID runs a robot through all applicable phases with a pre-generated execution ID (no control)
func (e *Executor) ExecuteWithID(ctx *robottypes.Context, robot *robottypes.Robot, trigger robottypes.TriggerType, data interface{}, execID string) (*robottypes.Execution, error) {
	return e.ExecuteWithControl(ctx, robot, trigger, data, execID, nil)
}

// ExecuteWithControl runs a robot through all applicable phases with execution control
// control: optional, allows pause/resume functionality during execution
func (e *Executor) ExecuteWithControl(ctx *robottypes.Context, robot *robottypes.Robot, trigger robottypes.TriggerType, data interface{}, execID string, control robottypes.ExecutionControl) (*robottypes.Execution, error) {
	if robot == nil {
		return nil, fmt.Errorf("robot cannot be nil")
	}

	// Determine starting phase based on trigger type
	startPhaseIndex := 0
	if trigger == robottypes.TriggerHuman || trigger == robottypes.TriggerEvent {
		startPhaseIndex = 1 // Skip P0 (Inspiration)
	}

	// Use provided execID or generate new one
	if execID == "" {
		execID = utils.NewID()
	}

	// Create execution (Job system removed, using ExecutionStore only)
	input := types.BuildTriggerInput(trigger, data)
	exec := &robottypes.Execution{
		ID:          execID,
		MemberID:    robot.MemberID,
		TeamID:      robot.TeamID,
		TriggerType: trigger,
		StartTime:   time.Now(),
		Status:      robottypes.ExecPending,
		Phase:       robottypes.AllPhases[startPhaseIndex],
		Input:       input,
	}

	// Initialize UI display fields (with i18n support)
	exec.Name, exec.CurrentTaskName = e.initUIFields(trigger, input, robot)

	// Set robot reference for phase methods
	exec.SetRobot(robot)

	// Persist execution record to database
	// Robot is identified by member_id (globally unique in __yao.member table)
	if !e.config.SkipPersistence && e.store != nil {
		record := store.FromExecution(exec)
		if err := e.store.Save(ctx.Context, record); err != nil {
			// Log warning but don't fail execution
			log.With(log.F{
				"execution_id": exec.ID,
				"member_id":    exec.MemberID,
				"error":        err,
			}).Warn("Failed to persist execution record: %v", err)
		}
	}

	// Acquire execution slot
	if !robot.TryAcquireSlot(exec) {
		log.With(log.F{
			"execution_id": exec.ID,
			"member_id":    exec.MemberID,
		}).Warn("Execution quota exceeded")
		return nil, robottypes.ErrQuotaExceeded
	}
	// Defer: remove execution from robot's tracking and update robot status if no more executions
	defer func() {
		robot.RemoveExecution(exec.ID)
		// Update robot status to idle if no more running executions
		if robot.RunningCount() == 0 && !e.config.SkipPersistence && e.robotStore != nil {
			if err := e.robotStore.UpdateStatus(ctx.Context, robot.MemberID, robottypes.RobotIdle); err != nil {
				log.With(log.F{
					"member_id": robot.MemberID,
					"error":     err,
				}).Warn("Failed to update robot status to idle: %v", err)
			}
		}
	}()

	// Track execution count
	e.execCount.Add(1)
	e.currentCount.Add(1)
	defer e.currentCount.Add(-1)

	// Callbacks
	if e.onStart != nil {
		e.onStart()
	}
	if e.onEnd != nil {
		defer e.onEnd()
	}

	// Update status to running
	exec.Status = robottypes.ExecRunning
	log.With(log.F{
		"execution_id": exec.ID,
		"member_id":    exec.MemberID,
		"trigger_type": string(exec.TriggerType),
	}).Info("Execution started")

	// Persist running status
	if !e.config.SkipPersistence && e.store != nil {
		if err := e.store.UpdateStatus(ctx.Context, exec.ID, robottypes.ExecRunning, ""); err != nil {
			log.With(log.F{
				"execution_id": exec.ID,
				"error":        err,
			}).Warn("Failed to persist running status: %v", err)
		}
	}

	// Update robot status to working (when execution starts)
	if !e.config.SkipPersistence && e.robotStore != nil {
		if err := e.robotStore.UpdateStatus(ctx.Context, robot.MemberID, robottypes.RobotWorking); err != nil {
			log.With(log.F{
				"member_id": robot.MemberID,
				"error":     err,
			}).Warn("Failed to update robot status to working: %v", err)
		}
	}

	// Check for simulated failure (for testing)
	if dataStr, ok := data.(string); ok && dataStr == "simulate_failure" {
		exec.Status = robottypes.ExecFailed
		exec.Error = "simulated failure"
		log.With(log.F{
			"execution_id": exec.ID,
			"member_id":    exec.MemberID,
		}).Warn("Simulated failure triggered")
		// Persist failed status
		if !e.config.SkipPersistence && e.store != nil {
			_ = e.store.UpdateStatus(ctx.Context, exec.ID, robottypes.ExecFailed, "simulated failure")
		}
		return exec, nil
	}

	// Determine locale for UI messages
	locale := getEffectiveLocale(robot, exec.Input)

	// Execute phases
	phases := robottypes.AllPhases[startPhaseIndex:]
	for _, phase := range phases {
		if err := e.runPhase(ctx, exec, phase, data, control); err != nil {
			// Check if execution was cancelled
			if err == robottypes.ErrExecutionCancelled {
				exec.Status = robottypes.ExecCancelled
				exec.Error = "execution cancelled by user"
				now := time.Now()
				exec.EndTime = &now

				// Update UI field for cancellation with i18n
				e.updateUIFields(ctx, exec, "", getLocalizedMessage(locale, "cancelled"))

				log.With(log.F{
					"execution_id": exec.ID,
					"member_id":    exec.MemberID,
					"phase":        string(phase),
				}).Info("Execution cancelled by user")

				// Persist cancelled status
				if !e.config.SkipPersistence && e.store != nil {
					_ = e.store.UpdateStatus(ctx.Context, exec.ID, robottypes.ExecCancelled, "execution cancelled by user")
				}
				return exec, nil
			}

			// Normal failure case
			exec.Status = robottypes.ExecFailed
			exec.Error = err.Error()

			// Update UI field for failure with i18n
			// Use concise phase name, NOT the full error message (error is in exec.Error)
			failedPrefix := getLocalizedMessage(locale, "failed_prefix")
			phaseName := getLocalizedMessage(locale, "phase_"+string(phase))
			failureMsg := failedPrefix + phaseName
			e.updateUIFields(ctx, exec, "", failureMsg)

			log.With(log.F{
				"execution_id": exec.ID,
				"member_id":    exec.MemberID,
				"phase":        string(phase),
				"error":        err.Error(),
			}).Error("Phase execution failed: %v", err)
			// Persist failed status
			if !e.config.SkipPersistence && e.store != nil {
				_ = e.store.UpdateStatus(ctx.Context, exec.ID, robottypes.ExecFailed, err.Error())
			}
			return exec, nil
		}
	}

	// Mark completed
	exec.Status = robottypes.ExecCompleted
	now := time.Now()
	exec.EndTime = &now

	// Update UI field for completion with i18n
	e.updateUIFields(ctx, exec, "", getLocalizedMessage(locale, "completed"))

	duration := now.Sub(exec.StartTime)
	log.With(log.F{
		"execution_id": exec.ID,
		"member_id":    exec.MemberID,
		"duration_ms":  duration.Milliseconds(),
	}).Info("Execution completed successfully")

	// Persist completed status
	if !e.config.SkipPersistence && e.store != nil {
		if err := e.store.UpdateStatus(ctx.Context, exec.ID, robottypes.ExecCompleted, ""); err != nil {
			log.With(log.F{
				"execution_id": exec.ID,
				"error":        err,
			}).Warn("Failed to persist completed status: %v", err)
		}
	}

	return exec, nil
}

// runPhase executes a single phase
func (e *Executor) runPhase(ctx *robottypes.Context, exec *robottypes.Execution, phase robottypes.Phase, data interface{}, control robottypes.ExecutionControl) error {
	// Check if context is cancelled before starting this phase
	select {
	case <-ctx.Context.Done():
		return robottypes.ErrExecutionCancelled
	default:
	}

	// Wait if execution is paused (blocks until resumed or cancelled)
	if control != nil {
		if err := control.WaitIfPaused(); err != nil {
			return err // Returns ErrExecutionCancelled if cancelled while paused
		}
	}

	exec.Phase = phase

	log.With(log.F{
		"execution_id": exec.ID,
		"member_id":    exec.MemberID,
		"phase":        string(phase),
	}).Info("Phase started: %s", phase)

	// Persist phase change immediately (so frontend sees current phase)
	if !e.config.SkipPersistence && e.store != nil {
		if err := e.store.UpdatePhase(ctx.Context, exec.ID, phase, nil); err != nil {
			log.With(log.F{
				"execution_id": exec.ID,
				"phase":        string(phase),
				"error":        err,
			}).Warn("Failed to persist phase start: %v", err)
		}
	}

	if e.config.OnPhaseStart != nil {
		e.config.OnPhaseStart(phase)
	}

	phaseStart := time.Now()

	// Execute phase-specific logic
	var err error
	switch phase {
	case robottypes.PhaseInspiration:
		err = e.RunInspiration(ctx, exec, data)
	case robottypes.PhaseGoals:
		err = e.RunGoals(ctx, exec, data)
	case robottypes.PhaseTasks:
		err = e.RunTasks(ctx, exec, data)
	case robottypes.PhaseRun:
		err = e.RunExecution(ctx, exec, data)
	case robottypes.PhaseDelivery:
		err = e.RunDelivery(ctx, exec, data)
	case robottypes.PhaseLearning:
		err = e.RunLearning(ctx, exec, data)
	}

	if err != nil {
		log.With(log.F{
			"execution_id": exec.ID,
			"member_id":    exec.MemberID,
			"phase":        string(phase),
			"error":        err.Error(),
		}).Error("Phase failed: %s - %v", phase, err)
		return err
	}

	// Persist phase output to database
	if !e.config.SkipPersistence && e.store != nil {
		phaseData := e.getPhaseData(exec, phase)
		if phaseData != nil {
			if err := e.store.UpdatePhase(ctx.Context, exec.ID, phase, phaseData); err != nil {
				// Log warning but don't fail execution
				log.With(log.F{
					"execution_id": exec.ID,
					"phase":        string(phase),
					"error":        err,
				}).Warn("Failed to persist phase %s data: %v", phase, err)
			}
		}
	}

	if e.config.OnPhaseEnd != nil {
		e.config.OnPhaseEnd(phase)
	}

	phaseDuration := time.Since(phaseStart).Milliseconds()
	log.With(log.F{
		"execution_id": exec.ID,
		"member_id":    exec.MemberID,
		"phase":        string(phase),
		"duration_ms":  phaseDuration,
	}).Info("Phase completed: %s (took %dms)", phase, phaseDuration)

	return nil
}

// getPhaseData extracts the output data for a specific phase from execution
func (e *Executor) getPhaseData(exec *robottypes.Execution, phase robottypes.Phase) interface{} {
	switch phase {
	case robottypes.PhaseInspiration:
		return exec.Inspiration
	case robottypes.PhaseGoals:
		return exec.Goals
	case robottypes.PhaseTasks:
		return exec.Tasks
	case robottypes.PhaseRun:
		return exec.Results
	case robottypes.PhaseDelivery:
		return exec.Delivery
	case robottypes.PhaseLearning:
		return exec.Learning
	default:
		return nil
	}
}

// ExecCount returns total execution count
func (e *Executor) ExecCount() int {
	return int(e.execCount.Load())
}

// CurrentCount returns currently running execution count
func (e *Executor) CurrentCount() int {
	return int(e.currentCount.Load())
}

// Reset resets the executor counters
func (e *Executor) Reset() {
	e.execCount.Store(0)
	e.currentCount.Store(0)
}

// DefaultStreamDelay is the simulated delay for Agent Stream calls
// This will be removed when real Agent calls are implemented
const DefaultStreamDelay = 50 * time.Millisecond

// simulateStreamDelay simulates the delay of an Agent Stream call
func (e *Executor) simulateStreamDelay() {
	time.Sleep(DefaultStreamDelay)
}

// initUIFields initializes UI display fields based on trigger type with i18n support
// Returns (name, currentTaskName)
func (e *Executor) initUIFields(trigger robottypes.TriggerType, input *robottypes.TriggerInput, robot *robottypes.Robot) (string, string) {
	// Determine locale for UI messages
	locale := getEffectiveLocale(robot, input)

	// Get localized default messages
	name := getLocalizedMessage(locale, "preparing")
	currentTaskName := getLocalizedMessage(locale, "starting")

	switch trigger {
	case robottypes.TriggerHuman:
		// For human trigger, extract name from first message
		if input != nil && len(input.Messages) > 0 {
			if content, ok := input.Messages[0].GetContentAsString(); ok && content != "" {
				// Use first 100 chars of message as name
				name = content
				if len(name) > 100 {
					name = name[:100] + "..."
				}
			}
		}
	case robottypes.TriggerClock:
		name = getLocalizedMessage(locale, "scheduled_execution")
	case robottypes.TriggerEvent:
		if input != nil && input.EventType != "" {
			name = getLocalizedMessage(locale, "event_prefix") + input.EventType
		} else {
			name = getLocalizedMessage(locale, "event_triggered")
		}
	}

	return name, currentTaskName
}

// getEffectiveLocale determines the locale for UI display
// Priority: input.Locale > robot.Config.DefaultLocale > "en"
func getEffectiveLocale(robot *robottypes.Robot, input *robottypes.TriggerInput) string {
	// 1. Human trigger with explicit locale
	if input != nil && input.Locale != "" {
		return input.Locale
	}
	// 2. Robot configured default
	if robot != nil && robot.Config != nil {
		return robot.Config.GetDefaultLocale()
	}
	// 3. System default
	return "en"
}

// i18n message maps for UI display fields
// Use simple locale codes (en, zh) as keys
var uiMessages = map[string]map[string]string{
	"en": {
		"preparing":           "Preparing...",
		"starting":            "Starting...",
		"scheduled_execution": "Scheduled execution",
		"event_prefix":        "Event: ",
		"event_triggered":     "Event triggered",
		"analyzing_context":   "Analyzing context...",
		"planning_goals":      "Planning goals...",
		"breaking_down_tasks": "Breaking down tasks...",
		"generating_delivery": "Generating delivery content...",
		"sending_delivery":    "Sending delivery...",
		"learning_from_exec":  "Learning from execution...",
		"completed":           "Completed",
		"cancelled":           "Cancelled",
		"failed_prefix":       "Failed at ",
		"task_prefix":         "Task",
		// Phase names for failure messages
		"phase_inspiration": "inspiration",
		"phase_goals":       "goals",
		"phase_tasks":       "tasks",
		"phase_run":         "execution",
		"phase_delivery":    "delivery",
		"phase_learning":    "learning",
	},
	"zh": {
		"preparing":           "准备中...",
		"starting":            "启动中...",
		"scheduled_execution": "定时执行",
		"event_prefix":        "事件: ",
		"event_triggered":     "事件触发",
		"analyzing_context":   "分析上下文...",
		"planning_goals":      "规划目标...",
		"breaking_down_tasks": "分解任务...",
		"generating_delivery": "生成交付内容...",
		"sending_delivery":    "正在发送...",
		"learning_from_exec":  "学习执行经验...",
		"completed":           "已完成",
		"cancelled":           "已取消",
		"failed_prefix":       "失败于",
		"task_prefix":         "任务",
		// Phase names for failure messages
		"phase_inspiration": "灵感阶段",
		"phase_goals":       "目标阶段",
		"phase_tasks":       "任务阶段",
		"phase_run":         "执行阶段",
		"phase_delivery":    "交付阶段",
		"phase_learning":    "学习阶段",
	},
}

// getLocalizedMessage returns a localized message for the given key
func getLocalizedMessage(locale string, key string) string {
	if messages, ok := uiMessages[locale]; ok {
		if msg, ok := messages[key]; ok {
			return msg
		}
	}
	// Fallback to English
	if messages, ok := uiMessages["en"]; ok {
		if msg, ok := messages[key]; ok {
			return msg
		}
	}
	return key // Return key as fallback
}

// updateUIFields updates UI display fields and persists to database
func (e *Executor) updateUIFields(ctx *robottypes.Context, exec *robottypes.Execution, name string, currentTaskName string) {
	// Update in-memory execution
	if name != "" {
		exec.Name = name
	}
	if currentTaskName != "" {
		exec.CurrentTaskName = currentTaskName
	}

	// Persist to database
	if !e.config.SkipPersistence && e.store != nil {
		if err := e.store.UpdateUIFields(ctx.Context, exec.ID, name, currentTaskName); err != nil {
			log.With(log.F{
				"execution_id": exec.ID,
				"error":        err,
			}).Warn("Failed to update UI fields: %v", err)
		}
	}
}

// updateTasksState persists the current tasks array with status to database
// This should be called after each task status change for real-time UI updates
func (e *Executor) updateTasksState(ctx *robottypes.Context, exec *robottypes.Execution) {
	if e.config.SkipPersistence || e.store == nil {
		return
	}

	// Convert Current to store.CurrentState
	var current *store.CurrentState
	if exec.Current != nil {
		current = &store.CurrentState{
			TaskIndex: exec.Current.TaskIndex,
			Progress:  exec.Current.Progress,
		}
	}

	if err := e.store.UpdateTasks(ctx.Context, exec.ID, exec.Tasks, current); err != nil {
		log.With(log.F{
			"execution_id": exec.ID,
			"error":        err,
		}).Warn("Failed to update tasks state: %v", err)
	}
}

// extractGoalName extracts the execution name from goals output
func extractGoalName(goals *robottypes.Goals) string {
	if goals == nil || goals.Content == "" {
		return ""
	}

	// Extract first non-empty, non-markdown-header line as the goal name
	content := goals.Content
	lines := strings.Split(content, "\n")

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		// Skip markdown headers (# ## ### etc.)
		if strings.HasPrefix(line, "#") {
			continue
		}
		// Skip markdown horizontal rules (--- or ***)
		if strings.HasPrefix(line, "---") || strings.HasPrefix(line, "***") {
			continue
		}
		// Found a content line - strip markdown formatting
		line = stripMarkdownFormatting(line)
		// Limit length
		if len(line) > 150 {
			line = line[:150] + "..."
		}
		return line
	}

	// Fallback: if all lines are headers, use first header without # prefix
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		// Strip leading # symbols
		line = strings.TrimLeft(line, "#")
		line = strings.TrimSpace(line)
		line = stripMarkdownFormatting(line)
		if line != "" {
			if len(line) > 150 {
				line = line[:150] + "..."
			}
			return line
		}
	}

	return ""
}

// stripMarkdownFormatting removes common markdown formatting from text
func stripMarkdownFormatting(s string) string {
	// Remove bold/italic markers
	s = strings.ReplaceAll(s, "**", "")
	s = strings.ReplaceAll(s, "__", "")
	s = strings.ReplaceAll(s, "*", "")
	s = strings.ReplaceAll(s, "_", "")
	// Remove inline code
	s = strings.ReplaceAll(s, "`", "")
	// Remove link syntax [text](url) -> text
	// Simple approach: just remove brackets and parentheses content
	for {
		start := strings.Index(s, "[")
		if start == -1 {
			break
		}
		end := strings.Index(s[start:], "]")
		if end == -1 {
			break
		}
		linkEnd := start + end
		// Check if followed by (url)
		if linkEnd+1 < len(s) && s[linkEnd+1] == '(' {
			parenEnd := strings.Index(s[linkEnd+1:], ")")
			if parenEnd != -1 {
				// Extract just the link text
				linkText := s[start+1 : linkEnd]
				s = s[:start] + linkText + s[linkEnd+1+parenEnd+1:]
				continue
			}
		}
		// Just remove brackets
		s = s[:start] + s[start+1:linkEnd] + s[linkEnd+1:]
	}
	return strings.TrimSpace(s)
}

// Verify Executor implements types.Executor
var _ types.Executor = (*Executor)(nil)
