//go:build unit

package types_test

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	types "github.com/yaoapp/yao/agent/robot/types"
)

func TestHostInputJSON(t *testing.T) {
	input := &types.HostInput{
		Scenario: "assign",
		Context: &types.HostContext{
			RobotStatus: &types.RobotStatusSnapshot{
				ActiveCount: 1,
				MaxQuota:    5,
			},
			Goals: &types.Goals{Content: "test goals"},
		},
	}

	data, err := json.Marshal(input)
	require.NoError(t, err)

	var parsed types.HostInput
	err = json.Unmarshal(data, &parsed)
	require.NoError(t, err)
	assert.Equal(t, "assign", parsed.Scenario)
	assert.NotNil(t, parsed.Context)
	assert.Equal(t, 1, parsed.Context.RobotStatus.ActiveCount)
}

func TestHostOutputJSON(t *testing.T) {
	output := &types.HostOutput{
		Reply:       "Task confirmed",
		Action:      types.HostActionConfirm,
		WaitForMore: false,
	}

	data, err := json.Marshal(output)
	require.NoError(t, err)

	var parsed types.HostOutput
	err = json.Unmarshal(data, &parsed)
	require.NoError(t, err)
	assert.Equal(t, "Task confirmed", parsed.Reply)
	assert.Equal(t, types.HostActionConfirm, parsed.Action)
	assert.False(t, parsed.WaitForMore)
}

func TestHostOutputWithActionData(t *testing.T) {
	output := &types.HostOutput{
		Reply:      "I'll adjust the plan",
		Action:     types.HostActionAdjust,
		ActionData: map[string]interface{}{"goals": "adjusted goals"},
	}

	data, err := json.Marshal(output)
	require.NoError(t, err)

	var parsed types.HostOutput
	err = json.Unmarshal(data, &parsed)
	require.NoError(t, err)
	assert.Equal(t, types.HostActionAdjust, parsed.Action)
	assert.NotNil(t, parsed.ActionData)
}
