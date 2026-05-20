//go:build unit

package manager_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/yaoapp/yao/agent/robot/executor/standard"
	"github.com/yaoapp/yao/agent/robot/manager"
	"github.com/yaoapp/yao/agent/robot/store"
	"github.com/yaoapp/yao/agent/robot/types"
)

func TestBuildRobotStatusSnapshot(t *testing.T) {
	m := manager.New()

	t.Run("nil_robot_returns_nil", func(t *testing.T) {
		snap := manager.ExportBuildRobotStatusSnapshot(m, nil)
		assert.Nil(t, snap)
	})

	t.Run("robot_with_quota", func(t *testing.T) {
		robot := &types.Robot{
			MemberID: "test-member",
			Config: &types.Config{
				Quota: &types.Quota{Max: 5},
			},
		}
		snap := manager.ExportBuildRobotStatusSnapshot(m, robot)
		require.NotNil(t, snap)
		assert.Equal(t, 5, snap.MaxQuota)
	})

	t.Run("robot_without_quota_uses_default", func(t *testing.T) {
		robot := &types.Robot{
			MemberID: "test-member",
		}
		snap := manager.ExportBuildRobotStatusSnapshot(m, robot)
		require.NotNil(t, snap)
		assert.Equal(t, 2, snap.MaxQuota)
	})
}

func TestFindWaitingTask(t *testing.T) {
	m := manager.New()

	t.Run("returns_nil_when_no_waiting_task_id", func(t *testing.T) {
		record := &store.ExecutionRecord{
			Tasks: []types.Task{
				{ID: "task-1"},
			},
		}
		task := manager.ExportFindWaitingTask(m, record)
		assert.Nil(t, task)
	})

	t.Run("finds_matching_task", func(t *testing.T) {
		record := &store.ExecutionRecord{
			WaitingTaskID: "task-2",
			Tasks: []types.Task{
				{ID: "task-1"},
				{ID: "task-2", Status: types.TaskWaitingInput},
				{ID: "task-3"},
			},
		}
		task := manager.ExportFindWaitingTask(m, record)
		require.NotNil(t, task)
		assert.Equal(t, "task-2", task.ID)
	})

	t.Run("returns_nil_when_task_not_found", func(t *testing.T) {
		record := &store.ExecutionRecord{
			WaitingTaskID: "nonexistent",
			Tasks: []types.Task{
				{ID: "task-1"},
			},
		}
		task := manager.ExportFindWaitingTask(m, record)
		assert.Nil(t, task)
	})
}

func TestBuildHostContext(t *testing.T) {
	m := manager.New()

	t.Run("builds_context_with_goals_and_tasks", func(t *testing.T) {
		robot := &types.Robot{MemberID: "test"}
		record := &store.ExecutionRecord{
			Goals: &types.Goals{Content: "test goals"},
			Tasks: []types.Task{
				{ID: "task-1"},
			},
			WaitingQuestion: "What is the answer?",
		}
		waitingTask := &types.Task{ID: "task-1", Status: types.TaskWaitingInput}

		hostCtx := manager.ExportBuildHostContext(m, robot, record, waitingTask)
		require.NotNil(t, hostCtx)
		assert.NotNil(t, hostCtx.Goals)
		assert.Equal(t, "test goals", hostCtx.Goals.Content)
		assert.Len(t, hostCtx.Tasks, 1)
		assert.NotNil(t, hostCtx.CurrentTask)
		assert.Equal(t, "What is the answer?", hostCtx.AgentReply)
	})

	t.Run("builds_context_without_optional_fields", func(t *testing.T) {
		robot := &types.Robot{MemberID: "test"}
		record := &store.ExecutionRecord{}

		hostCtx := manager.ExportBuildHostContext(m, robot, record, nil)
		require.NotNil(t, hostCtx)
		assert.Nil(t, hostCtx.Goals)
		assert.Nil(t, hostCtx.Tasks)
		assert.Nil(t, hostCtx.CurrentTask)
		assert.Empty(t, hostCtx.AgentReply)
	})
}

func TestProcessHostAction(t *testing.T) {
	m := manager.New()

	t.Run("wait_for_more_returns_waiting_status", func(t *testing.T) {
		output := &types.HostOutput{
			Reply:       "Please provide more details",
			WaitForMore: true,
		}
		record := &store.ExecutionRecord{}
		robot := &types.Robot{}
		execStore := store.NewExecutionStore()

		resp, err := manager.ExportProcessHostAction(m, types.NewContext(nil, nil), robot, record, output, execStore)
		require.NoError(t, err)
		assert.Equal(t, "waiting_for_more", resp.Status)
		assert.Equal(t, "Please provide more details", resp.Reply)
		assert.True(t, resp.WaitForMore)
	})

	t.Run("unknown_action_returns_acknowledged", func(t *testing.T) {
		output := &types.HostOutput{
			Reply:  "Got it",
			Action: "unknown_action",
		}
		record := &store.ExecutionRecord{}
		robot := &types.Robot{}
		execStore := store.NewExecutionStore()

		resp, err := manager.ExportProcessHostAction(m, types.NewContext(nil, nil), robot, record, output, execStore)
		require.NoError(t, err)
		assert.Equal(t, "acknowledged", resp.Status)
	})
}

func TestParseHostAgentResult(t *testing.T) {
	m := manager.New()

	t.Run("plain_text_returns_WaitForMore", func(t *testing.T) {
		result := &standard.CallResult{Content: "I understand your request. Shall I proceed?"}
		output, err := manager.ExportParseHostAgentResult(m, result)
		require.NoError(t, err)
		assert.True(t, output.WaitForMore)
		assert.Equal(t, "I understand your request. Shall I proceed?", output.Reply)
		assert.Empty(t, string(output.Action))
	})

	t.Run("JSON_with_action_returns_action", func(t *testing.T) {
		result := &standard.CallResult{
			Content: `{"reply":"Task confirmed","action":"confirm","wait_for_more":false}`,
		}
		output, err := manager.ExportParseHostAgentResult(m, result)
		require.NoError(t, err)
		assert.False(t, output.WaitForMore)
		assert.Equal(t, types.HostActionConfirm, output.Action)
		assert.Equal(t, "Task confirmed", output.Reply)
	})

	t.Run("JSON_without_action_returns_WaitForMore", func(t *testing.T) {
		result := &standard.CallResult{
			Content: `{"reply":"Let me think about this","some_field":"value"}`,
		}
		output, err := manager.ExportParseHostAgentResult(m, result)
		require.NoError(t, err)
		assert.True(t, output.WaitForMore)
		assert.NotEmpty(t, output.Reply)
	})

	t.Run("JSON_with_adjust_action_and_action_data", func(t *testing.T) {
		result := &standard.CallResult{
			Content: `{"reply":"Plan adjusted","action":"adjust","action_data":{"goals":"new goals"}}`,
		}
		output, err := manager.ExportParseHostAgentResult(m, result)
		require.NoError(t, err)
		assert.False(t, output.WaitForMore)
		assert.Equal(t, types.HostActionAdjust, output.Action)
		assert.NotNil(t, output.ActionData)
	})

	t.Run("malformed_JSON_returns_WaitForMore", func(t *testing.T) {
		result := &standard.CallResult{Content: `{invalid json`}
		output, err := manager.ExportParseHostAgentResult(m, result)
		require.NoError(t, err)
		assert.True(t, output.WaitForMore)
		assert.Equal(t, `{invalid json`, output.Reply)
	})

	t.Run("empty_content_returns_WaitForMore", func(t *testing.T) {
		result := &standard.CallResult{Content: ""}
		output, err := manager.ExportParseHostAgentResult(m, result)
		require.NoError(t, err)
		assert.True(t, output.WaitForMore)
	})
}
