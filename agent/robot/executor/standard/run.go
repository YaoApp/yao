package standard

import (
	"fmt"
	"time"

	robottypes "github.com/yaoapp/yao/agent/robot/types"
)

// RunConfig configures P3 execution behavior
type RunConfig struct {
	// ContinueOnFailure continues to next task even if current task fails (default: false)
	ContinueOnFailure bool

	// ValidationThreshold is the minimum score to pass validation (default: 0.6)
	ValidationThreshold float64

	// MaxTurnsPerTask is the maximum conversation turns for multi-turn tasks (default: 10)
	// This controls how many times the assistant can be called for a single task
	// (including retries with validation feedback)
	MaxTurnsPerTask int
}

// DefaultRunConfig returns the default P3 configuration
func DefaultRunConfig() *RunConfig {
	return &RunConfig{
		ContinueOnFailure:   false,
		ValidationThreshold: 0.6,
		MaxTurnsPerTask:     10,
	}
}

// RunExecution executes P3: Run phase
// Executes each task using the appropriate executor (Assistant, MCP, Process)
// with multi-turn conversation and validation
//
// Input:
//   - Tasks (from P2)
//
// Output:
//   - TaskResult for each task with output and validation
//
// Execution Flow (per task):
//  1. Call assistant/MCP/process and get result
//  2. Validate result using two-layer validation (rule-based + semantic)
//  3. If validation.NeedReply, continue conversation with validation.ReplyContent
//  4. Repeat until validation.Complete or max turns exceeded
//  5. Pass previous task results as context to next task
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

	// Determine locale for UI messages
	locale := getEffectiveLocale(robot, exec.Input)

	// Initialize results slice
	exec.Results = make([]robottypes.TaskResult, 0, len(exec.Tasks))

	// Create task runner
	runner := NewRunner(ctx, robot, config)

	// Execute tasks sequentially
	for i := range exec.Tasks {
		task := &exec.Tasks[i]

		// Update current state for tracking
		exec.Current = &robottypes.CurrentState{
			Task:      task,
			TaskIndex: i,
			Progress:  fmt.Sprintf("%d/%d tasks", i+1, len(exec.Tasks)),
		}

		// Update UI field with current task description (i18n)
		taskName := formatTaskProgressName(task, i, len(exec.Tasks), locale)
		e.updateUIFields(ctx, exec, "", taskName)

		// Mark task as running
		task.Status = robottypes.TaskRunning
		now := time.Now()
		task.StartTime = &now

		// Persist running state to database
		e.updateTasksState(ctx, exec)

		// Build task context with previous results
		taskCtx := runner.BuildTaskContext(exec, i)

		// Execute task with multi-turn conversation support
		result := runner.ExecuteWithRetry(task, taskCtx)

		// Update task status based on result
		endTime := time.Now()
		task.EndTime = &endTime

		// Determine task status from result
		// Note: result.Success is already set to (validation.Complete && validation.Passed) in runner
		if result.Success {
			task.Status = robottypes.TaskCompleted
		} else {
			task.Status = robottypes.TaskFailed
		}

		// Store result
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

	// Clear current state
	exec.Current = nil

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
