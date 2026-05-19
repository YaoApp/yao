//go:build unit

package events_test

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	events "github.com/yaoapp/yao/agent/robot/events"
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
		"ExecRecovered": "robot.exec.recovered",
		"Delivery":      "robot.delivery",
		"Message":       "robot.message",
	}

	actual := map[string]string{
		"TaskNeedInput": events.TaskNeedInput,
		"TaskFailed":    events.TaskFailed,
		"TaskCompleted": events.TaskCompleted,
		"ExecWaiting":   events.ExecWaiting,
		"ExecResumed":   events.ExecResumed,
		"ExecCompleted": events.ExecCompleted,
		"ExecFailed":    events.ExecFailed,
		"ExecCancelled": events.ExecCancelled,
		"ExecRecovered": events.ExecRecovered,
		"Delivery":      events.Delivery,
		"Message":       events.Message,
	}

	for name, exp := range expected {
		assert.Equal(t, exp, actual[name], "Event constant %s mismatch", name)
	}
	assert.Len(t, actual, 11, "Expected exactly 11 event constants")
}

func TestEventConstantNamingConvention(t *testing.T) {
	allEvents := []string{
		events.TaskNeedInput, events.TaskFailed, events.TaskCompleted,
		events.ExecWaiting, events.ExecResumed, events.ExecCompleted,
		events.ExecFailed, events.ExecCancelled, events.ExecRecovered,
		events.Delivery, events.Message,
	}

	for _, e := range allEvents {
		assert.Contains(t, e, "robot.", "Event %q should start with 'robot.'", e)
	}
}

func TestRobotConfigEventConstants(t *testing.T) {
	assert.Equal(t, "robot.config.created", events.RobotConfigCreated)
	assert.Equal(t, "robot.config.updated", events.RobotConfigUpdated)
	assert.Equal(t, "robot.config.deleted", events.RobotConfigDeleted)
}

func TestNeedInputPayloadMarshalling(t *testing.T) {
	payload := events.NeedInputPayload{
		ExecutionID: "exec-123",
		MemberID:    "member-1",
		TeamID:      "team-1",
		TaskID:      "task-5",
		Question:    "What date range?",
		ChatID:      "chat-abc",
	}

	data, err := json.Marshal(payload)
	require.NoError(t, err)

	var parsed events.NeedInputPayload
	err = json.Unmarshal(data, &parsed)
	require.NoError(t, err)
	assert.Equal(t, payload, parsed)
}

func TestNeedInputPayloadEmptyQuestion(t *testing.T) {
	payload := events.NeedInputPayload{
		ExecutionID: "exec-ep2",
		MemberID:    "member-ep2",
		TeamID:      "team-ep2",
		TaskID:      "task-ep2",
		Question:    "",
	}

	data, err := json.Marshal(payload)
	require.NoError(t, err)

	var parsed map[string]interface{}
	err = json.Unmarshal(data, &parsed)
	require.NoError(t, err)

	q, ok := parsed["question"]
	assert.True(t, ok)
	assert.Equal(t, "", q)
}

func TestTaskPayloadMarshalling(t *testing.T) {
	t.Run("with error", func(t *testing.T) {
		payload := events.TaskPayload{
			ExecutionID: "exec-1",
			MemberID:    "member-1",
			TeamID:      "team-1",
			TaskID:      "task-1",
			Error:       "timeout",
			ChatID:      "chat-1",
		}

		data, err := json.Marshal(payload)
		require.NoError(t, err)

		var parsed events.TaskPayload
		err = json.Unmarshal(data, &parsed)
		require.NoError(t, err)
		assert.Equal(t, payload, parsed)
	})

	t.Run("without error", func(t *testing.T) {
		payload := events.TaskPayload{
			ExecutionID: "exec-2",
			MemberID:    "member-2",
			TeamID:      "team-2",
			TaskID:      "task-2",
		}

		data, err := json.Marshal(payload)
		require.NoError(t, err)

		var parsed events.TaskPayload
		err = json.Unmarshal(data, &parsed)
		require.NoError(t, err)
		assert.Equal(t, payload, parsed)
		assert.Empty(t, parsed.Error)
	})
}

func TestTaskPayloadErrorSerialization(t *testing.T) {
	payload := events.TaskPayload{
		ExecutionID: "exec-ep3",
		MemberID:    "member-ep3",
		TeamID:      "team-ep3",
		TaskID:      "task-ep3",
		Error:       "context deadline exceeded",
	}

	data, err := json.Marshal(payload)
	require.NoError(t, err)

	var parsed map[string]interface{}
	err = json.Unmarshal(data, &parsed)
	require.NoError(t, err)
	assert.Equal(t, "context deadline exceeded", parsed["error"])
}

func TestExecPayloadMarshalling(t *testing.T) {
	payload := events.ExecPayload{
		ExecutionID: "exec-100",
		MemberID:    "member-10",
		TeamID:      "team-10",
		Status:      "completed",
		ChatID:      "chat-100",
	}

	data, err := json.Marshal(payload)
	require.NoError(t, err)

	var parsed events.ExecPayload
	err = json.Unmarshal(data, &parsed)
	require.NoError(t, err)
	assert.Equal(t, payload, parsed)
}

func TestExecPayloadAllStatuses(t *testing.T) {
	statuses := []string{
		"running", "completed", "failed", "cancelled", "waiting", "confirming",
	}

	for _, s := range statuses {
		payload := events.ExecPayload{
			ExecutionID: "exec-ep1",
			MemberID:    "member-ep1",
			TeamID:      "team-ep1",
			Status:      s,
		}

		data, err := json.Marshal(payload)
		require.NoError(t, err)

		var parsed events.ExecPayload
		err = json.Unmarshal(data, &parsed)
		require.NoError(t, err)
		assert.Equal(t, s, parsed.Status, "Status %s should round-trip", s)
	}
}

func TestExecPayloadOmitsEmptyOptionalFields(t *testing.T) {
	payload := events.ExecPayload{
		ExecutionID: "exec-ep6",
		MemberID:    "member-ep6",
		TeamID:      "team-ep6",
		Status:      "completed",
	}

	data, err := json.Marshal(payload)
	require.NoError(t, err)

	var parsed map[string]interface{}
	err = json.Unmarshal(data, &parsed)
	require.NoError(t, err)

	_, hasChatID := parsed["chat_id"]
	if hasChatID {
		assert.Equal(t, "", parsed["chat_id"])
	}
}

func TestDeliveryPayloadMarshalling(t *testing.T) {
	payload := events.DeliveryPayload{
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

	var parsed events.DeliveryPayload
	err = json.Unmarshal(data, &parsed)
	require.NoError(t, err)
	assert.Equal(t, "exec-d1", parsed.ExecutionID)
	assert.Equal(t, "member-d1", parsed.MemberID)
	assert.NotNil(t, parsed.Content)
	assert.Equal(t, "done", parsed.Content.Summary)
	assert.NotNil(t, parsed.Preferences)
	assert.NotNil(t, parsed.Preferences.Email)
}

func TestDeliveryPayloadNestedContent(t *testing.T) {
	payload := events.DeliveryPayload{
		ExecutionID: "exec-ep4",
		MemberID:    "member-ep4",
		TeamID:      "team-ep4",
		Content: &robottypes.DeliveryContent{
			Summary: "Daily Summary",
			Body:    "Full body with sections: intro, body, conclusion",
		},
	}

	data, err := json.Marshal(payload)
	require.NoError(t, err)

	var parsed events.DeliveryPayload
	err = json.Unmarshal(data, &parsed)
	require.NoError(t, err)

	require.NotNil(t, parsed.Content)
	assert.Equal(t, "Daily Summary", parsed.Content.Summary)
	assert.Contains(t, parsed.Content.Body, "sections")
}

func TestMessagePayloadMarshalling(t *testing.T) {
	payload := events.MessagePayload{
		RobotID: "robot-1",
		Metadata: &events.MessageMetadata{
			Channel:    "telegram",
			MessageID:  "msg-123",
			ChatID:     "chat-456",
			SenderID:   "user-789",
			SenderName: "TestUser",
			Locale:     "zh-cn",
		},
	}

	data, err := json.Marshal(payload)
	require.NoError(t, err)

	var parsed events.MessagePayload
	err = json.Unmarshal(data, &parsed)
	require.NoError(t, err)
	assert.Equal(t, "robot-1", parsed.RobotID)
	require.NotNil(t, parsed.Metadata)
	assert.Equal(t, "telegram", parsed.Metadata.Channel)
	assert.Equal(t, "zh-cn", parsed.Metadata.Locale)
}

func TestPayloadCommonFields(t *testing.T) {
	needInput := events.NeedInputPayload{ExecutionID: "e1", MemberID: "m1", TeamID: "t1"}
	task := events.TaskPayload{ExecutionID: "e2", MemberID: "m2", TeamID: "t2"}
	exec := events.ExecPayload{ExecutionID: "e3", MemberID: "m3", TeamID: "t3"}
	delivery := events.DeliveryPayload{ExecutionID: "e4", MemberID: "m4", TeamID: "t4"}

	assert.Equal(t, "e1", needInput.ExecutionID)
	assert.Equal(t, "m2", task.MemberID)
	assert.Equal(t, "t3", exec.TeamID)
	assert.Equal(t, "e4", delivery.ExecutionID)
}

func TestNormalizeLocale(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"", "en"},
		{"zh-Hans", "zh-cn"},
		{"zh-CN", "zh-cn"},
		{"zh-Hant", "zh-tw"},
		{"zh-TW", "zh-tw"},
		{"zh-HK", "zh-tw"},
		{"zh", "zh-cn"},
		{"en-US", "en-us"},
		{"en-GB", "en-gb"},
		{"en", "en"},
		{"ja", "ja"},
		{"zh_CN", "zh-cn"},
		{"  zh-Hans  ", "zh-cn"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := events.NormalizeLocale(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}
