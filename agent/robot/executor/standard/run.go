package standard

import (
	"fmt"
	"path"
	"time"

	kunlog "github.com/yaoapp/kun/log"
	robotevents "github.com/yaoapp/yao/agent/robot/events"
	robottypes "github.com/yaoapp/yao/agent/robot/types"
	"github.com/yaoapp/yao/event"
)

// RunConfig configures P3 execution behavior
type RunConfig struct {
	// ContinueOnFailure continues to next task even if current task fails.
	// V2 default: true — the Robot is an orchestrator, not a judge.
	// Failed tasks are recorded and evaluated by the Delivery Agent.
	ContinueOnFailure bool
}

// DefaultRunConfig returns the default P3 configuration
func DefaultRunConfig() *RunConfig {
	return &RunConfig{
		ContinueOnFailure: true,
	}
}

// RunExecution executes P3: Run phase
// Executes each task using the appropriate executor (Assistant, MCP, Process).
//
// V2 simplified flow: single call per task, no validation loop.
// Success is determined by whether the call itself succeeds (no error).
// The Delivery Agent (P4) evaluates overall quality using expected_output.
//
// Supports resume: if exec.ResumeContext is set, execution starts from the
// suspended task index with previously completed results restored.
//
// Returns ErrExecutionSuspended if a task signals it needs human input.
func (e *Executor) RunExecution(ctx *robottypes.Context, exec *robottypes.Execution, data interface{}) error {
	robot := exec.GetRobot()
	if robot == nil {
		return fmt.Errorf("robot not found in execution")
	}

	if len(exec.Tasks) == 0 {
		return fmt.Errorf("no tasks to execute")
	}

	// Get run configuration from data or use default
	var config *RunConfig
	if cfg, ok := data.(*RunConfig); ok && cfg != nil {
		config = cfg
	} else {
		config = DefaultRunConfig()
	}

	// Determine start index and restore results from resume context
	startIndex := 0
	if exec.ResumeContext != nil {
		startIndex = exec.ResumeContext.TaskIndex
		exec.Results = exec.ResumeContext.PreviousResults
	} else {
		exec.Results = make([]robottypes.TaskResult, 0, len(exec.Tasks))
	}

	// Create task runner with execution-level chatID (§8.4)
	runner := NewRunner(ctx, robot, config, exec.ChatID, exec.ID)
	if ctx.Locale != "" {
		runner.locale = ctx.Locale
	} else {
		runner.locale = getEffectiveLocale(robot, exec.Input)
	}

	// Initialize workspace for file-based context
	wsFS, err := ensureRobotWorkspace(ctx, robot)
	if err != nil {
		kunlog.Warn("[robot-run] workspace unavailable, falling back to in-memory context: %v", err)
	} else {
		execDir := path.Join("robots", robot.MemberID, exec.ID)
		if mkErr := wsFS.MkdirAll(execDir, 0755); mkErr != nil {
			kunlog.Warn("[robot-run] mkdir %s: %v", execDir, mkErr)
		} else {
			runner.wsFS = wsFS
			runner.execDir = execDir
			runner.initManifest(exec)
		}
	}

	// Execute tasks sequentially from startIndex
	for i := startIndex; i < len(exec.Tasks); i++ {
		task := &exec.Tasks[i]

		// Update current state for tracking
		exec.Current = &robottypes.CurrentState{
			Task:      task,
			TaskIndex: i,
			Progress:  fmt.Sprintf("%d/%d tasks", i+1, len(exec.Tasks)),
		}

		// Update UI field with current task description (i18n)
		taskName := formatTaskProgressName(task, i, len(exec.Tasks), runner.locale)
		e.updateUIFields(ctx, exec, "", taskName)

		// Mark task as running
		task.Status = robottypes.TaskRunning
		now := time.Now()
		task.StartTime = &now

		// Persist running state to database
		e.updateTasksState(ctx, exec)

		// Build task context with previous results
		taskCtx := runner.BuildTaskContext(exec, i)

		// Execute task (single call, no validation loop)
		result := runner.ExecuteTask(task, taskCtx)

		// Task needs human input — suspend execution without recording a half-result
		if result.NeedInput {
			return e.Suspend(ctx, exec, i, result.InputQuestion)
		}

		// Update task status based on result
		endTime := time.Now()
		task.EndTime = &endTime

		if result.Success {
			task.Status = robottypes.TaskCompleted
			event.Push(ctx.Context, robotevents.TaskCompleted, robotevents.TaskPayload{
				ExecutionID: exec.ID,
				MemberID:    exec.MemberID,
				TeamID:      exec.TeamID,
				TaskID:      task.ID,
				ChatID:      exec.ChatID,
			})
		} else {
			task.Status = robottypes.TaskFailed
			event.Push(ctx.Context, robotevents.TaskFailed, robotevents.TaskPayload{
				ExecutionID: exec.ID,
				MemberID:    exec.MemberID,
				TeamID:      exec.TeamID,
				TaskID:      task.ID,
				Error:       result.Error,
				ChatID:      exec.ChatID,
			})
		}

		// Write task files to workspace (non-blocking, errors logged)
		runner.writeTaskOutput(task, result, runner.lastPromptSnapshot)

		// Store result (in-memory, for persistence + resume)
		exec.Results = append(exec.Results, *result)

		// Persist completed/failed state to database
		e.updateTasksState(ctx, exec)

		// Check if we should continue on failure
		if !result.Success && !config.ContinueOnFailure {
			// Mark remaining tasks as skipped
			for j := i + 1; j < len(exec.Tasks); j++ {
				exec.Tasks[j].Status = robottypes.TaskSkipped
			}
			// Persist skipped state to database
			e.updateTasksState(ctx, exec)
			return fmt.Errorf("task %s failed: %s", task.ID, result.Error)
		}
	}

	// Clear current state and resume context after successful completion
	exec.Current = nil
	exec.ResumeContext = nil

	return nil
}

// formatTaskProgressName formats a progress name for the current task (used for UI with i18n)
func formatTaskProgressName(task *robottypes.Task, index int, total int, locale string) string {
	taskPrefix := getLocalizedMessage(locale, "task_prefix")
	prefix := fmt.Sprintf("%s %d/%d: ", taskPrefix, index+1, total)

	// Priority 1: Use Description field if available
	if task.Description != "" {
		desc := task.Description
		if len(desc) > 80 {
			desc = desc[:80] + "..."
		}
		return prefix + desc
	}

	// Priority 2: Try to get description from first message
	if len(task.Messages) > 0 {
		if content, ok := task.Messages[0].GetContentAsString(); ok && content != "" {
			// Truncate if too long
			if len(content) > 80 {
				content = content[:80] + "..."
			}
			return prefix + content
		}
	}

	// Fallback to executor info
	return prefix + string(task.ExecutorType) + ":" + task.ExecutorID
}
