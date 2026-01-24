package standard

import (
	"fmt"
	"time"

	robottypes "github.com/yaoapp/yao/agent/robot/types"
)

// RunInspiration executes P0: Inspiration phase
// Calls the Inspiration Agent to generate daily briefing
//
// Input:
//   - ClockContext from trigger input or current time
//   - Robot identity and resources
//
// Output:
//   - InspirationReport with markdown content
func (e *Executor) RunInspiration(ctx *robottypes.Context, exec *robottypes.Execution, _ interface{}) error {
	// Get robot for identity and resources
	robot := exec.GetRobot()
	if robot == nil {
		return fmt.Errorf("robot not found in execution")
	}

	// Update UI field with i18n
	locale := getEffectiveLocale(robot, exec.Input)
	e.updateUIFields(ctx, exec, "", getLocalizedMessage(locale, "analyzing_context"))

	// Build clock context from trigger input or current time
	var clock *robottypes.ClockContext
	if exec.Input != nil && exec.Input.Clock != nil {
		clock = exec.Input.Clock
	} else {
		clock = robottypes.NewClockContext(time.Now(), "")
	}

	// Get agent ID for inspiration phase
	agentID := "__yao.inspiration" // default
	if robot.Config != nil && robot.Config.Resources != nil {
		agentID = robot.Config.Resources.GetPhaseAgent(robottypes.PhaseInspiration)
	}

	// Build prompt using InputFormatter
	formatter := NewInputFormatter()
	userContent := formatter.FormatClockContext(clock, robot)

	// Add available resources - critical for generating achievable insights
	resourcesContent := formatter.FormatAvailableResources(robot)
	if resourcesContent != "" {
		userContent += "\n\n" + resourcesContent
	}

	// Call agent
	caller := NewAgentCaller()
	result, err := caller.CallWithMessages(ctx, agentID, userContent)
	if err != nil {
		return fmt.Errorf("inspiration agent (%s) call failed: %w", agentID, err)
	}

	// Parse response - get markdown content
	content := result.GetText()
	if content == "" {
		return fmt.Errorf("inspiration agent (%s) returned empty response", agentID)
	}

	// Build InspirationReport
	exec.Inspiration = &robottypes.InspirationReport{
		Clock:   clock,
		Content: content,
	}

	return nil
}
