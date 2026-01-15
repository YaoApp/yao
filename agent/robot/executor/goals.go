package executor

import (
	"github.com/yaoapp/yao/agent/robot/types"
)

// RunGoals executes P1: Goals phase
//
// For Clock trigger: Uses InspirationReport to generate goals
// For Human/Event: Uses TriggerInput directly as goals or to generate goals
//
// Implementation (TODO Phase 5):
// 1. Build prompt with InspirationReport (or TriggerInput for Human/Event)
// 2. Call Goal Generation Agent via Assistant.Stream()
// 3. Parse response to Goals (markdown)
func (e *Executor) RunGoals(_ *types.Context, exec *types.Execution, _ interface{}) error {
	// TODO (Phase 5): Replace with real Agent call
	// agentID := robot.Config.Resources.GetPhaseAgent(types.PhaseGoals)
	// messages := buildGoalsMessages(exec.Inspiration, exec.Input, robot)
	// response, err := callAgentStream(ctx, agentID, messages)
	// if err != nil {
	//     return err
	// }
	// exec.Goals = parseGoals(response)

	// Simulate Agent Stream delay
	e.simulateStreamDelay()

	// Generate mock goals
	exec.Goals = &types.Goals{
		Content: `## Goals

1. [High] Complete primary objective
   - Reason: Critical for business success
   - Expected outcome: Measurable improvement

2. [Normal] Review and validate results
   - Reason: Quality assurance required
   - Expected outcome: Verified deliverables

3. [Low] Document learnings
   - Reason: Future reference and improvement
   - Expected outcome: Knowledge base update`,
	}

	return nil
}
