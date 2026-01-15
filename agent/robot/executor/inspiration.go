package executor

import (
	"time"

	"github.com/yaoapp/yao/agent/robot/types"
)

// RunInspiration executes P0: Inspiration phase (Clock trigger only)
//
// This phase gathers information to help make good goals.
// ClockContext is the key input - Agent knows what time it is and can decide
// what to do (e.g., 5pm Friday → write weekly report).
//
// Implementation (TODO Phase 4):
// 1. Build prompt with ClockContext + data sources (KB, DB, web search)
// 2. Call Inspiration Agent via Assistant.Stream()
// 3. Parse response to InspirationReport (markdown)
func (e *Executor) RunInspiration(_ *types.Context, exec *types.Execution, _ interface{}) error {
	// TODO (Phase 4): Replace with real Agent call
	// agentID := robot.Config.Resources.GetPhaseAgent(types.PhaseInspiration)
	// messages := buildInspirationMessages(exec.Input.Clock, robot)
	// response, err := callAgentStream(ctx, agentID, messages)
	// if err != nil {
	//     return err
	// }
	// exec.Inspiration = parseInspirationReport(response)

	// Simulate Agent Stream delay
	e.simulateStreamDelay()

	// Generate mock inspiration report
	exec.Inspiration = &types.InspirationReport{
		Clock: types.NewClockContext(time.Now(), ""),
		Content: `## Summary
Mock inspiration report for testing.

## Highlights
- [High] Test item 1 - Critical business metric changed
- [Normal] Test item 2 - Regular update available

## Opportunities
- Market growth potential identified
- New customer segment emerging

## Risks
- None identified in current period

## Pending
- 2 tasks from previous execution
- 1 scheduled report due`,
	}

	return nil
}
