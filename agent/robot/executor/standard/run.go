package standard

import (
	"fmt"
	"time"

	robottypes "github.com/yaoapp/yao/agent/robot/types"
)

// RunConfig configures P3 execution behavior
type RunConfig struct {
	// MaxRetries is the maximum number of retry attempts per task (default: 3)
	MaxRetries int

	// RetryOnValidationFailure enables retry when validation fails (default: true)
	RetryOnValidationFailure bool

	// ContinueOnFailure continues to next task even if current task fails (default: false)
	ContinueOnFailure bool

	// ValidationThreshold is the minimum score to pass validation (default: 0.6)
	ValidationThreshold float64

	// MaxTurnsPerTask is the maximum conversation turns for multi-turn agents (default: 10)
	MaxTurnsPerTask int
}

// DefaultRunConfig returns the default P3 configuration
func DefaultRunConfig() *RunConfig {
	return &RunConfig{
		MaxRetries:               3,
		RetryOnValidationFailure: true,
		ContinueOnFailure:        false,
		ValidationThreshold:      0.6,
		MaxTurnsPerTask:          10,
	}
}

// RunExecution executes P3: Run phase
// Executes each task using the appropriate executor (Assistant, MCP, Process)
// with validation and retry mechanism
//
// Input:
//   - Tasks (from P2)
//
// Output:
//   - TaskResult for each task with output and validation
//
// Features:
//  1. Sequential task execution with progress tracking
//  2. Validation after each task using Validation Agent
//  3. Retry mechanism with feedback loop to expert agent
//  4. Multi-turn conversation support for complex tasks
//  5. Previous task results passed as context to next task
func (e *Executor) RunExecution(ctx *robottypes.Context, exec *robottypes.Execution, _ interface{}) error {
	robot := exec.GetRobot()
	if robot == nil {
		return fmt.Errorf("robot not found in execution")
	}

	if len(exec.Tasks) == 0 {
		return fmt.Errorf("no tasks to execute")
	}

	// Get run configuration
	config := DefaultRunConfig()

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

		// Mark task as running
		task.Status = robottypes.TaskRunning
		now := time.Now()
		task.StartTime = &now

		// Build task context with previous results
		taskCtx := runner.BuildTaskContext(exec, i)

		// Execute task with retry
		result := runner.ExecuteWithRetry(task, taskCtx)

		// Update task status based on result
		endTime := time.Now()
		task.EndTime = &endTime
		if result.Success && (result.Validation == nil || result.Validation.Passed) {
			task.Status = robottypes.TaskCompleted
		} else {
			task.Status = robottypes.TaskFailed
		}

		// Store result
		exec.Results = append(exec.Results, *result)

		// Check if we should continue on failure
		if !result.Success && !config.ContinueOnFailure {
			// Mark remaining tasks as skipped
			for j := i + 1; j < len(exec.Tasks); j++ {
				exec.Tasks[j].Status = robottypes.TaskSkipped
			}
			return fmt.Errorf("task %s failed: %s", task.ID, result.Error)
		}
	}

	// Clear current state
	exec.Current = nil

	return nil
}
