package executor

import (
	"fmt"
	"time"

	"github.com/yaoapp/yao/agent/robot/types"
)

// RunExecution executes P3: Run phase
//
// Iterates through tasks and executes each one using the specified executor.
// Supports three executor types: assistant, mcp, process.
//
// Implementation (TODO Phase 7):
// 1. Iterate tasks
// 2. For each task, call executor (assistant/mcp/process)
// 3. Validate results
// 4. Collect results
func (e *Executor) RunExecution(_ *types.Context, exec *types.Execution, _ interface{}) error {
	// TODO (Phase 7): Replace with real task execution
	// for i, task := range exec.Tasks {
	//     exec.Current = &types.CurrentState{Task: &task, TaskIndex: i}
	//     result, err := executeTask(ctx, task, robot)
	//     if err != nil {
	//         return err
	//     }
	//     exec.Results = append(exec.Results, result)
	// }

	// Handle empty tasks case
	if len(exec.Tasks) == 0 {
		exec.Current = &types.CurrentState{
			TaskIndex: 0,
			Progress:  "0/0 tasks",
		}
		exec.Results = []types.TaskResult{}
		return nil
	}

	// Set current state (will be updated as tasks complete)
	exec.Current = &types.CurrentState{
		TaskIndex: 0,
		Progress:  fmt.Sprintf("0/%d tasks", len(exec.Tasks)),
	}

	// Simulate execution of each task
	exec.Results = make([]types.TaskResult, len(exec.Tasks))
	for i := range exec.Tasks {
		// Update current state
		exec.Current.TaskIndex = i
		exec.Current.Task = &exec.Tasks[i]
		exec.Current.Progress = fmt.Sprintf("%d/%d tasks", i+1, len(exec.Tasks))

		// Mark task start time
		startTime := time.Now()
		exec.Tasks[i].StartTime = &startTime

		// Simulate Agent Stream delay for each task
		e.simulateStreamDelay()

		// Mark task as completed
		exec.Tasks[i].Status = types.TaskCompleted
		endTime := time.Now()
		exec.Tasks[i].EndTime = &endTime

		// Generate mock result with actual duration
		exec.Results[i] = types.TaskResult{
			TaskID:    exec.Tasks[i].ID,
			Success:   true,
			Output:    fmt.Sprintf("Mock output for %s: Task completed successfully", exec.Tasks[i].ID),
			Duration:  endTime.Sub(startTime).Milliseconds(),
			Validated: true,
		}
	}

	return nil
}
