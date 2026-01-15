// Package trigger provides trigger-related utilities and execution control
// The main trigger logic is in the manager package.
// This package provides:
// - Validation functions for intervention and event requests
// - Builder helpers for TriggerInput
// - ExecutionController for pause/resume/stop
// - ClockMatcher for clock trigger matching (reusable)
package trigger

import (
	"fmt"

	"github.com/yaoapp/yao/agent/robot/types"
)

// ValidateIntervention validates a human intervention request
func ValidateIntervention(req *types.InterveneRequest) error {
	if req == nil {
		return fmt.Errorf("request is nil")
	}

	if req.MemberID == "" {
		return fmt.Errorf("member_id is required")
	}

	if !isValidAction(req.Action) {
		return fmt.Errorf("invalid action: %s", req.Action)
	}

	// Validate action-specific requirements
	switch req.Action {
	case types.ActionTaskAdd, types.ActionGoalAdd, types.ActionInstruct:
		// These actions require messages
		if len(req.Messages) == 0 {
			return fmt.Errorf("messages required for action: %s", req.Action)
		}

	case types.ActionPlanAdd:
		// Plan add requires plan_time
		if req.PlanTime == nil {
			return fmt.Errorf("plan_time required for action: plan.add")
		}
	}

	return nil
}

// ValidateEvent validates an event trigger request
func ValidateEvent(req *types.EventRequest) error {
	if req == nil {
		return fmt.Errorf("request is nil")
	}

	if req.MemberID == "" {
		return fmt.Errorf("member_id is required")
	}

	if req.Source == "" {
		return fmt.Errorf("source is required")
	}

	if req.EventType == "" {
		return fmt.Errorf("event_type is required")
	}

	return nil
}

// BuildEventInput creates a TriggerInput from an event request
func BuildEventInput(req *types.EventRequest) *types.TriggerInput {
	return &types.TriggerInput{
		Source:    types.EventSource(req.Source),
		EventType: req.EventType,
		Data:      req.Data,
	}
}

// isValidAction checks if the intervention action is valid
func isValidAction(action types.InterventionAction) bool {
	switch action {
	case types.ActionTaskAdd,
		types.ActionTaskCancel,
		types.ActionTaskUpdate,
		types.ActionGoalAdjust,
		types.ActionGoalAdd,
		types.ActionGoalComplete,
		types.ActionGoalCancel,
		types.ActionPlanAdd,
		types.ActionPlanRemove,
		types.ActionPlanUpdate,
		types.ActionInstruct:
		return true
	default:
		return false
	}
}

// GetActionCategory returns the category of an intervention action
func GetActionCategory(action types.InterventionAction) string {
	switch action {
	case types.ActionTaskAdd, types.ActionTaskCancel, types.ActionTaskUpdate:
		return "task"
	case types.ActionGoalAdjust, types.ActionGoalAdd, types.ActionGoalComplete, types.ActionGoalCancel:
		return "goal"
	case types.ActionPlanAdd, types.ActionPlanRemove, types.ActionPlanUpdate:
		return "plan"
	case types.ActionInstruct:
		return "instruct"
	default:
		return "unknown"
	}
}

// GetActionDescription returns a human-readable description of an action
func GetActionDescription(action types.InterventionAction) string {
	switch action {
	case types.ActionTaskAdd:
		return "Add a new task"
	case types.ActionTaskCancel:
		return "Cancel a task"
	case types.ActionTaskUpdate:
		return "Update task details"
	case types.ActionGoalAdjust:
		return "Adjust current goal"
	case types.ActionGoalAdd:
		return "Add a new goal"
	case types.ActionGoalComplete:
		return "Mark goal as complete"
	case types.ActionGoalCancel:
		return "Cancel a goal"
	case types.ActionPlanAdd:
		return "Add to plan queue"
	case types.ActionPlanRemove:
		return "Remove from plan queue"
	case types.ActionPlanUpdate:
		return "Update planned item"
	case types.ActionInstruct:
		return "Direct instruction to robot"
	default:
		return "Unknown action"
	}
}
