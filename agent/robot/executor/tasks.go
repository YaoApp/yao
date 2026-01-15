package executor

import (
	"github.com/yaoapp/yao/agent/robot/types"
)

// RunTasks executes P2: Tasks phase
//
// Reads Goals markdown and breaks into executable tasks.
// Each task specifies executor type (assistant/mcp/process) and arguments.
//
// Implementation (TODO Phase 6):
// 1. Build prompt with Goals
// 2. Call Task Planning Agent via Assistant.Stream()
// 3. Parse response to []Task (structured)
func (e *Executor) RunTasks(_ *types.Context, exec *types.Execution, _ interface{}) error {
	// TODO (Phase 6): Replace with real Agent call
	// agentID := robot.Config.Resources.GetPhaseAgent(types.PhaseTasks)
	// messages := buildTasksMessages(exec.Goals, robot)
	// response, err := callAgentStream(ctx, agentID, messages)
	// if err != nil {
	//     return err
	// }
	// exec.Tasks = parseTasks(response)

	// Simulate Agent Stream delay
	e.simulateStreamDelay()

	// Generate mock tasks
	exec.Tasks = []types.Task{
		{
			ID:           "task_1",
			GoalRef:      "Goal 1",
			Source:       types.TaskSourceAuto,
			ExecutorType: types.ExecutorAssistant,
			ExecutorID:   "data-analyst",
			Status:       types.TaskPending,
			Order:        0,
		},
		{
			ID:           "task_2",
			GoalRef:      "Goal 2",
			Source:       types.TaskSourceAuto,
			ExecutorType: types.ExecutorAssistant,
			ExecutorID:   "report-writer",
			Status:       types.TaskPending,
			Order:        1,
		},
	}

	return nil
}
