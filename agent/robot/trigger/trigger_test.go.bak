package trigger_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	agentcontext "github.com/yaoapp/yao/agent/context"
	"github.com/yaoapp/yao/agent/robot/trigger"
	"github.com/yaoapp/yao/agent/robot/types"
)

// ==================== ValidateIntervention Tests ====================

func TestValidateIntervention(t *testing.T) {
	t.Run("nil request returns error", func(t *testing.T) {
		err := trigger.ValidateIntervention(nil)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "request is nil")
	})

	t.Run("empty member_id returns error", func(t *testing.T) {
		req := &types.InterveneRequest{
			MemberID: "",
			Action:   types.ActionTaskAdd,
		}
		err := trigger.ValidateIntervention(req)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "member_id is required")
	})

	t.Run("invalid action returns error", func(t *testing.T) {
		req := &types.InterveneRequest{
			MemberID: "robot_001",
			Action:   types.InterventionAction("invalid.action"),
		}
		err := trigger.ValidateIntervention(req)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid action")
	})

	t.Run("task.add without messages returns error", func(t *testing.T) {
		req := &types.InterveneRequest{
			MemberID: "robot_001",
			Action:   types.ActionTaskAdd,
			Messages: nil,
		}
		err := trigger.ValidateIntervention(req)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "messages required")
	})

	t.Run("goal.add without messages returns error", func(t *testing.T) {
		req := &types.InterveneRequest{
			MemberID: "robot_001",
			Action:   types.ActionGoalAdd,
			Messages: nil,
		}
		err := trigger.ValidateIntervention(req)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "messages required")
	})

	t.Run("instruct without messages returns error", func(t *testing.T) {
		req := &types.InterveneRequest{
			MemberID: "robot_001",
			Action:   types.ActionInstruct,
			Messages: nil,
		}
		err := trigger.ValidateIntervention(req)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "messages required")
	})

	t.Run("plan.add without plan_time returns error", func(t *testing.T) {
		req := &types.InterveneRequest{
			MemberID: "robot_001",
			Action:   types.ActionPlanAdd,
			PlanTime: nil,
		}
		err := trigger.ValidateIntervention(req)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "plan_time required")
	})

	t.Run("valid task.add request passes", func(t *testing.T) {
		req := &types.InterveneRequest{
			MemberID: "robot_001",
			Action:   types.ActionTaskAdd,
			Messages: []agentcontext.Message{
				{Role: agentcontext.RoleUser, Content: "Add a new task"},
			},
		}
		err := trigger.ValidateIntervention(req)
		assert.NoError(t, err)
	})

	t.Run("valid plan.add request passes", func(t *testing.T) {
		planTime := time.Now().Add(time.Hour)
		req := &types.InterveneRequest{
			MemberID: "robot_001",
			Action:   types.ActionPlanAdd,
			PlanTime: &planTime,
		}
		err := trigger.ValidateIntervention(req)
		assert.NoError(t, err)
	})

	t.Run("task.cancel without messages passes", func(t *testing.T) {
		req := &types.InterveneRequest{
			MemberID: "robot_001",
			Action:   types.ActionTaskCancel,
		}
		err := trigger.ValidateIntervention(req)
		assert.NoError(t, err)
	})

	t.Run("goal.adjust without messages passes", func(t *testing.T) {
		req := &types.InterveneRequest{
			MemberID: "robot_001",
			Action:   types.ActionGoalAdjust,
		}
		err := trigger.ValidateIntervention(req)
		assert.NoError(t, err)
	})
}

// ==================== ValidateEvent Tests ====================

func TestValidateEvent(t *testing.T) {
	t.Run("nil request returns error", func(t *testing.T) {
		err := trigger.ValidateEvent(nil)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "request is nil")
	})

	t.Run("empty member_id returns error", func(t *testing.T) {
		req := &types.EventRequest{
			MemberID:  "",
			Source:    "webhook",
			EventType: "lead.created",
		}
		err := trigger.ValidateEvent(req)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "member_id is required")
	})

	t.Run("empty source returns error", func(t *testing.T) {
		req := &types.EventRequest{
			MemberID:  "robot_001",
			Source:    "",
			EventType: "lead.created",
		}
		err := trigger.ValidateEvent(req)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "source is required")
	})

	t.Run("empty event_type returns error", func(t *testing.T) {
		req := &types.EventRequest{
			MemberID:  "robot_001",
			Source:    "webhook",
			EventType: "",
		}
		err := trigger.ValidateEvent(req)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "event_type is required")
	})

	t.Run("valid request passes", func(t *testing.T) {
		req := &types.EventRequest{
			MemberID:  "robot_001",
			Source:    "webhook",
			EventType: "lead.created",
			Data:      map[string]interface{}{"name": "John"},
		}
		err := trigger.ValidateEvent(req)
		assert.NoError(t, err)
	})
}

// ==================== BuildEventInput Tests ====================

func TestBuildEventInput(t *testing.T) {
	t.Run("builds correct TriggerInput", func(t *testing.T) {
		req := &types.EventRequest{
			MemberID:  "robot_001",
			Source:    "webhook",
			EventType: "lead.created",
			Data:      map[string]interface{}{"name": "John", "email": "john@example.com"},
		}

		input := trigger.BuildEventInput(req)

		assert.NotNil(t, input)
		assert.Equal(t, types.EventSource("webhook"), input.Source)
		assert.Equal(t, "lead.created", input.EventType)
		assert.Equal(t, "John", input.Data["name"])
		assert.Equal(t, "john@example.com", input.Data["email"])
	})

	t.Run("handles nil data", func(t *testing.T) {
		req := &types.EventRequest{
			MemberID:  "robot_001",
			Source:    "database",
			EventType: "order.paid",
			Data:      nil,
		}

		input := trigger.BuildEventInput(req)

		assert.NotNil(t, input)
		assert.Equal(t, types.EventSource("database"), input.Source)
		assert.Equal(t, "order.paid", input.EventType)
		assert.Nil(t, input.Data)
	})
}

// ==================== GetActionCategory Tests ====================

func TestGetActionCategory(t *testing.T) {
	tests := []struct {
		action   types.InterventionAction
		expected string
	}{
		{types.ActionTaskAdd, "task"},
		{types.ActionTaskCancel, "task"},
		{types.ActionTaskUpdate, "task"},
		{types.ActionGoalAdjust, "goal"},
		{types.ActionGoalAdd, "goal"},
		{types.ActionGoalComplete, "goal"},
		{types.ActionGoalCancel, "goal"},
		{types.ActionPlanAdd, "plan"},
		{types.ActionPlanRemove, "plan"},
		{types.ActionPlanUpdate, "plan"},
		{types.ActionInstruct, "instruct"},
		{types.InterventionAction("unknown"), "unknown"},
	}

	for _, tt := range tests {
		t.Run(string(tt.action), func(t *testing.T) {
			result := trigger.GetActionCategory(tt.action)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// ==================== GetActionDescription Tests ====================

func TestGetActionDescription(t *testing.T) {
	tests := []struct {
		action   types.InterventionAction
		contains string
	}{
		{types.ActionTaskAdd, "Add"},
		{types.ActionTaskCancel, "Cancel"},
		{types.ActionTaskUpdate, "Update"},
		{types.ActionGoalAdjust, "Adjust"},
		{types.ActionGoalAdd, "Add"},
		{types.ActionGoalComplete, "complete"},
		{types.ActionGoalCancel, "Cancel"},
		{types.ActionPlanAdd, "plan"},
		{types.ActionPlanRemove, "Remove"},
		{types.ActionPlanUpdate, "Update"},
		{types.ActionInstruct, "instruction"},
		{types.InterventionAction("unknown"), "Unknown"},
	}

	for _, tt := range tests {
		t.Run(string(tt.action), func(t *testing.T) {
			result := trigger.GetActionDescription(tt.action)
			assert.NotEmpty(t, result)
			assert.Contains(t, result, tt.contains)
		})
	}
}
