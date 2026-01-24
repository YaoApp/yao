package standard

import (
	"fmt"
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
	execCount    atomic.Int32
	currentCount atomic.Int32
	onStart      func()
	onEnd        func()
}

// New creates a new standard executor
func New() *Executor {
	return &Executor{
		store: store.NewExecutionStore(),
	}
}

// NewWithConfig creates a new standard executor with configuration
func NewWithConfig(config types.Config) *Executor {
	return &Executor{
		config: config,
		store:  store.NewExecutionStore(),
	}
}

// Execute runs a robot through all applicable phases with real Agent calls
func (e *Executor) Execute(ctx *robottypes.Context, robot *robottypes.Robot, trigger robottypes.TriggerType, data interface{}) (*robottypes.Execution, error) {
	if robot == nil {
		return nil, fmt.Errorf("robot cannot be nil")
	}

	// Determine starting phase based on trigger type
	startPhaseIndex := 0
	if trigger == robottypes.TriggerHuman || trigger == robottypes.TriggerEvent {
		startPhaseIndex = 1 // Skip P0 (Inspiration)
	}

	// Create execution (Job system removed, using ExecutionStore only)
	input := types.BuildTriggerInput(trigger, data)
	exec := &robottypes.Execution{
		ID:          utils.NewID(),
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
	defer robot.RemoveExecution(exec.ID)

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
		if err := e.runPhase(ctx, exec, phase, data); err != nil {
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
func (e *Executor) runPhase(ctx *robottypes.Context, exec *robottypes.Execution, phase robottypes.Phase, data interface{}) error {
	exec.Phase = phase

	log.With(log.F{
		"execution_id": exec.ID,
		"member_id":    exec.MemberID,
		"phase":        string(phase),
	}).Info("Phase started: %s", phase)

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
		"completed":           "Completed",
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
		"completed":           "已完成",
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

// extractGoalName extracts the execution name from goals output
func extractGoalName(goals *robottypes.Goals) string {
	if goals == nil || goals.Content == "" {
		return ""
	}

	// Extract first line or first sentence as the goal name
	content := goals.Content
	// Find first newline
	if idx := indexAny(content, "\n\r"); idx > 0 {
		content = content[:idx]
	}
	// Limit length
	if len(content) > 150 {
		content = content[:150] + "..."
	}
	return content
}

// indexAny returns the index of the first occurrence of any char in chars
func indexAny(s string, chars string) int {
	for i, c := range s {
		for _, ch := range chars {
			if c == ch {
				return i
			}
		}
	}
	return -1
}

// Verify Executor implements types.Executor
var _ types.Executor = (*Executor)(nil)
