package standard

import (
	"fmt"
	"strings"

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
//   - Goals with markdown content and delivery info
func (e *Executor) RunGoals(ctx *robottypes.Context, exec *robottypes.Execution, _ interface{}) error {
	// Get robot for identity and resources
	robot := exec.GetRobot()
	if robot == nil {
		return fmt.Errorf("robot not found in execution")
	}

	// Get agent ID for goals phase
	agentID := "__yao.goals" // default
	if robot.Config != nil && robot.Config.Resources != nil {
		agentID = robot.Config.Resources.GetPhaseAgent(robottypes.PhaseGoals)
	}

	// Build prompt based on trigger type
	formatter := NewInputFormatter()
	var userContent string

	switch exec.TriggerType {
	case robottypes.TriggerClock:
		// For clock trigger: use InspirationReport from P0
		if exec.Inspiration != nil {
			userContent = formatter.FormatInspirationReport(exec.Inspiration)
		} else {
			// Fallback: if no inspiration report, create minimal context
			userContent = formatter.FormatClockContext(
				robottypes.NewClockContext(exec.StartTime, ""),
				robot,
			)
		}

	case robottypes.TriggerHuman, robottypes.TriggerEvent:
		// For human/event trigger: use TriggerInput directly
		if exec.Input != nil {
			userContent = formatter.FormatTriggerInput(exec.Input)
		}
	}

	// Add robot identity context if not already included
	// For clock trigger with inspiration report, identity is not in the report
	// For human/event trigger, identity provides context
	if robot.Config != nil && robot.Config.Identity != nil {
		if !strings.Contains(userContent, "## Robot Identity") {
			userContent = formatter.FormatRobotIdentity(robot) + "\n\n" + userContent
		}
	}

	if userContent == "" {
		return fmt.Errorf("no input available for goals generation")
	}

	// Call agent
	caller := NewAgentCaller()
	result, err := caller.CallWithMessages(ctx, agentID, userContent)
	if err != nil {
		return fmt.Errorf("goals agent call failed: %w", err)
	}

	// Parse response as JSON
	// Goals Agent returns: { "content": "...", "delivery": {...} }
	data, err := result.GetJSON()
	if err != nil {
		// Fallback: if not JSON, use raw text as content
		content := result.GetText()
		if content == "" {
			return fmt.Errorf("goals agent returned empty response")
		}
		exec.Goals = &robottypes.Goals{
			Content: content,
		}
		return nil
	}

	// Build Goals from JSON
	exec.Goals = &robottypes.Goals{}

	// Extract content (markdown)
	if content, ok := data["content"].(string); ok {
		exec.Goals.Content = content
	}

	// Extract delivery
	if delivery, ok := data["delivery"].(map[string]interface{}); ok {
		exec.Goals.Delivery = parseDelivery(delivery)
	}

	// Validate: content is required
	if exec.Goals.Content == "" {
		return fmt.Errorf("goals agent returned empty content")
	}

	return nil
}

// parseDelivery converts map to DeliveryTarget struct
func parseDelivery(data map[string]interface{}) *robottypes.DeliveryTarget {
	if data == nil {
		return nil
	}

	target := &robottypes.DeliveryTarget{}

	// Parse and validate type
	if t, ok := data["type"].(string); ok {
		deliveryType := robottypes.DeliveryType(t)
		switch deliveryType {
		case robottypes.DeliveryEmail, robottypes.DeliveryWebhook,
			robottypes.DeliveryFile, robottypes.DeliveryNotify:
			target.Type = deliveryType
		default:
			// Invalid type - still set it but caller should validate
			target.Type = deliveryType
		}
	}

	// Parse recipients
	if recipients, ok := data["recipients"].([]interface{}); ok {
		for _, r := range recipients {
			if s, ok := r.(string); ok {
				target.Recipients = append(target.Recipients, s)
			}
		}
	}

	// Parse format
	if format, ok := data["format"].(string); ok {
		target.Format = format
	}

	// Parse template
	if template, ok := data["template"].(string); ok {
		target.Template = template
	}

	// Parse options
	if options, ok := data["options"].(map[string]interface{}); ok {
		target.Options = options
	}

	return target
}

// IsValidDeliveryType checks if the delivery type is valid
func IsValidDeliveryType(t robottypes.DeliveryType) bool {
	switch t {
	case robottypes.DeliveryEmail, robottypes.DeliveryWebhook,
		robottypes.DeliveryFile, robottypes.DeliveryNotify:
		return true
	default:
		return false
	}
}
