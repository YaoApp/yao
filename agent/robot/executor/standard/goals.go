package standard

import (
	robottypes "github.com/yaoapp/yao/agent/robot/types"
)

// RunGoals executes P1: Goals phase
// Calls the Goals Agent to plan daily objectives
//
// Input:
//   - InspirationReport (from P0) for clock trigger
//   - TriggerInput for human/event trigger
//
// Output:
//   - Goals with markdown content listing prioritized objectives
//
// TODO: Implement real Agent call
func (e *Executor) RunGoals(ctx *robottypes.Context, exec *robottypes.Execution, _ interface{}) error {
	e.simulateStreamDelay()

	exec.Goals = &robottypes.Goals{
		Content: "## Today's Goals\n\n1. [High] Review pending tasks\n2. [Medium] Process new requests\n3. [Low] Organize knowledge base",
	}
	return nil
}
