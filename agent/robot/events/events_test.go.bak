package events

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	robottypes "github.com/yaoapp/yao/agent/robot/types"
)

func TestEventConstants(t *testing.T) {
	expected := map[string]string{
		"TaskNeedInput": "robot.task.need_input",
		"TaskFailed":    "robot.task.failed",
		"TaskCompleted": "robot.task.completed",
		"ExecWaiting":   "robot.exec.waiting",
		"ExecResumed":   "robot.exec.resumed",
		"ExecCompleted": "robot.exec.completed",
		"ExecFailed":    "robot.exec.failed",
		"ExecCancelled": "robot.exec.cancelled",
		"Delivery":      "robot.delivery",
	}

	actual := map[string]string{
		"TaskNeedInput": TaskNeedInput,
		"TaskFailed":    TaskFailed,
		"TaskCompleted": TaskCompleted,
		"ExecWaiting":   ExecWaiting,
		"ExecResumed":   ExecResumed,
		"ExecCompleted": ExecCompleted,
		"ExecFailed":    ExecFailed,
		"ExecCancelled": ExecCancelled,
		"Delivery":      Delivery,
	}

	for name, exp := range expected {
		assert.Equal(t, exp, actual[name], "Event constant %s mismatch", name)
	}
	assert.Len(t, actual, 9, "Expected exactly 9 event constants")
}

func TestNeedInputPayloadMarshalling(t *testing.T) {
	payload := NeedInputPayload{
		ExecutionID: "exec-123",
		MemberID:    "member-1",
		TeamID:      "team-1",
		TaskID:      "task-5",
		Question:    "What date range?",
		ChatID:      "chat-abc",
	}

	data, err := json.Marshal(payload)
	require.NoError(t, err)

	var parsed NeedInputPayload
	err = json.Unmarshal(data, &parsed)
	require.NoError(t, err)
	assert.Equal(t, payload, parsed)
}

func TestTaskPayloadMarshalling(t *testing.T) {
	t.Run("with error", func(t *testing.T) {
		payload := TaskPayload{
			ExecutionID: "exec-1",
			MemberID:    "member-1",
			TeamID:      "team-1",
			TaskID:      "task-1",
			Error:       "timeout",
			ChatID:      "chat-1",
		}

		data, err := json.Marshal(payload)
		require.NoError(t, err)

		var parsed TaskPayload
		err = json.Unmarshal(data, &parsed)
		require.NoError(t, err)
		assert.Equal(t, payload, parsed)
	})

	t.Run("without error", func(t *testing.T) {
		payload := TaskPayload{
			ExecutionID: "exec-2",
			MemberID:    "member-2",
			TeamID:      "team-2",
			TaskID:      "task-2",
		}

		data, err := json.Marshal(payload)
		require.NoError(t, err)

		var parsed TaskPayload
		err = json.Unmarshal(data, &parsed)
		require.NoError(t, err)
		assert.Equal(t, payload, parsed)
		assert.Empty(t, parsed.Error)
	})
}

func TestExecPayloadMarshalling(t *testing.T) {
	payload := ExecPayload{
		ExecutionID: "exec-100",
		MemberID:    "member-10",
		TeamID:      "team-10",
		Status:      "completed",
		ChatID:      "chat-100",
	}

	data, err := json.Marshal(payload)
	require.NoError(t, err)

	var parsed ExecPayload
	err = json.Unmarshal(data, &parsed)
	require.NoError(t, err)
	assert.Equal(t, payload, parsed)
}

func TestDeliveryPayloadMarshalling(t *testing.T) {
	payload := DeliveryPayload{
		ExecutionID: "exec-d1",
		MemberID:    "member-d1",
		TeamID:      "team-d1",
		ChatID:      "chat-d1",
		Content: &robottypes.DeliveryContent{
			Summary: "done",
			Body:    "full report",
		},
		Preferences: &robottypes.DeliveryPreferences{
			Email: &robottypes.EmailPreference{
				Enabled: true,
				Targets: []robottypes.EmailTarget{{To: []string{"a@b.com"}}},
			},
		},
	}

	data, err := json.Marshal(payload)
	require.NoError(t, err)

	var parsed DeliveryPayload
	err = json.Unmarshal(data, &parsed)
	require.NoError(t, err)
	assert.Equal(t, "exec-d1", parsed.ExecutionID)
	assert.Equal(t, "member-d1", parsed.MemberID)
	assert.NotNil(t, parsed.Content)
	assert.Equal(t, "done", parsed.Content.Summary)
	assert.NotNil(t, parsed.Preferences)
	assert.NotNil(t, parsed.Preferences.Email)
}
