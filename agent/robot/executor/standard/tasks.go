package standard

import (
	robottypes "github.com/yaoapp/yao/agent/robot/types"
)

// RunTasks executes P2: Tasks phase
// Calls the Tasks Agent to break down goals into executable tasks
//
// Input:
//   - Goals (from P1)
//   - Available resources (Agents, MCP tools)
//
// Output:
//   - List of Task objects with executor assignments
//
// TODO: Implement real Agent call
func (e *Executor) RunTasks(ctx *robottypes.Context, exec *robottypes.Execution, _ interface{}) error {
	e.simulateStreamDelay()

	exec.Tasks = []robottypes.Task{
		{
			ID:           "task-1",
			GoalRef:      "Goal 1",
			Source:       robottypes.TaskSourceAuto,
			ExecutorType: robottypes.ExecutorAssistant,
			ExecutorID:   "default-assistant",
			Status:       robottypes.TaskPending,
		},
	}
	return nil
}
