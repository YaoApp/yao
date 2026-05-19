//go:build e2e

package standard_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/yaoapp/yao/agent/robot/executor/standard"
	robottypes "github.com/yaoapp/yao/agent/robot/types"
	"github.com/yaoapp/yao/unit-test/agent/testprepare"
)

func TestCallHostAgentJSONE2E(t *testing.T) {
	identity := testprepare.PrepareE2E(t)
	ctx := e2eCtx(identity)

	e := standard.New()

	robot := &robottypes.Robot{
		MemberID: "e2e-host-json",
		TeamID:   identity.BetaOpenAITeamID,
		Config: &robottypes.Config{
			Resources: &robottypes.Resources{
				Phases: map[robottypes.Phase]string{
					robottypes.PhaseHost: "tests.e2e-robot-host",
				},
			},
		},
	}

	input := &robottypes.HostInput{
		Scenario: "assign",
		Context: &robottypes.HostContext{
			RobotStatus: &robottypes.RobotStatusSnapshot{ActiveCount: 0, MaxQuota: 10},
		},
	}

	output, err := e.CallHostAgent(ctx, robot, input, "e2e-chat-host-json")
	require.NoError(t, err, "CallHostAgent should not error for valid host agent")
	require.NotNil(t, output, "output should not be nil")

	assert.NotEmpty(t, output.Reply, "reply should not be empty")

	validActions := []robottypes.HostAction{
		robottypes.HostActionConfirm,
		robottypes.HostActionAdjust,
		robottypes.HostActionAddTask,
		robottypes.HostActionSkip,
		robottypes.HostActionInjectCtx,
		robottypes.HostActionCancel,
	}
	actionValid := false
	for _, a := range validActions {
		if output.Action == a {
			actionValid = true
			break
		}
	}
	assert.True(t, actionValid || output.Action == "",
		"action should be a valid host action or empty fallback, got: %s", output.Action)
}

func TestCallHostAgentPlaintextE2E(t *testing.T) {
	identity := testprepare.PrepareE2E(t)
	ctx := e2eCtx(identity)

	e := standard.New()

	robot := &robottypes.Robot{
		MemberID: "e2e-host-plaintext",
		TeamID:   identity.BetaOpenAITeamID,
		Config: &robottypes.Config{
			Resources: &robottypes.Resources{
				Phases: map[robottypes.Phase]string{
					robottypes.PhaseHost: "tests.e2e-robot-host",
				},
			},
		},
	}

	input := &robottypes.HostInput{
		Scenario: "clarify",
		Context: &robottypes.HostContext{
			RobotStatus: &robottypes.RobotStatusSnapshot{ActiveCount: 3, MaxQuota: 5},
			AgentReply:  "I need more details about what data to analyze.",
		},
	}

	output, err := e.CallHostAgent(ctx, robot, input, "e2e-chat-host-plain")
	require.NoError(t, err, "host agent should handle response gracefully")
	require.NotNil(t, output, "output should not be nil")
	assert.NotEmpty(t, output.Reply, "reply should contain a response")
}
