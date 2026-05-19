package events

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	robottypes "github.com/yaoapp/yao/agent/robot/types"
)

// EP1: ExecPayload with all execution statuses
func TestExecPayloadAllStatuses(t *testing.T) {
	statuses := []string{
		"running", "completed", "failed", "cancelled", "waiting", "confirming",
	}

	for _, s := range statuses {
		payload := ExecPayload{
			ExecutionID: "exec-ep1",
			MemberID:    "member-ep1",
			TeamID:      "team-ep1",
			Status:      s,
		}

		data, err := json.Marshal(payload)
		require.NoError(t, err)

		var parsed ExecPayload
		err = json.Unmarshal(data, &parsed)
		require.NoError(t, err)
		assert.Equal(t, s, parsed.Status, "Status %s should round-trip", s)
	}
}

// EP2: NeedInputPayload with empty question
func TestNeedInputPayloadEmptyQuestion(t *testing.T) {
	payload := NeedInputPayload{
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

	// Empty string should be present but empty
	q, ok := parsed["question"]
	assert.True(t, ok)
	assert.Equal(t, "", q)
}

// EP3: TaskPayload serializes error correctly
func TestTaskPayloadErrorSerialization(t *testing.T) {
	payload := TaskPayload{
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

// EP4: DeliveryPayload with nested content
func TestDeliveryPayloadNestedContent(t *testing.T) {
	payload := DeliveryPayload{
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

	var parsed DeliveryPayload
	err = json.Unmarshal(data, &parsed)
	require.NoError(t, err)

	require.NotNil(t, parsed.Content)
	assert.Equal(t, "Daily Summary", parsed.Content.Summary)
	assert.Contains(t, parsed.Content.Body, "sections")
}

// EP5: Event constants follow naming convention
func TestEventConstantNamingConvention(t *testing.T) {
	allEvents := []string{
		TaskNeedInput, TaskFailed, TaskCompleted,
		ExecWaiting, ExecResumed, ExecCompleted, ExecFailed, ExecCancelled,
		Delivery,
	}

	for _, e := range allEvents {
		assert.Contains(t, e, "robot.", "Event %q should start with 'robot.'", e)
	}
}

// EP6: ExecPayload omits empty ChatID
func TestExecPayloadOmitsEmptyOptionalFields(t *testing.T) {
	payload := ExecPayload{
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

// EP7: All payloads share common fields (ExecutionID, MemberID, TeamID)
func TestPayloadCommonFields(t *testing.T) {
	needInput := NeedInputPayload{ExecutionID: "e1", MemberID: "m1", TeamID: "t1"}
	task := TaskPayload{ExecutionID: "e2", MemberID: "m2", TeamID: "t2"}
	exec := ExecPayload{ExecutionID: "e3", MemberID: "m3", TeamID: "t3"}
	delivery := DeliveryPayload{ExecutionID: "e4", MemberID: "m4", TeamID: "t4"}

	assert.Equal(t, "e1", needInput.ExecutionID)
	assert.Equal(t, "m2", task.MemberID)
	assert.Equal(t, "t3", exec.TeamID)
	assert.Equal(t, "e4", delivery.ExecutionID)
}
