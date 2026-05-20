//go:build integration

package standard_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/yaoapp/yao/agent/robot/executor/standard"
	robottypes "github.com/yaoapp/yao/agent/robot/types"
	"github.com/yaoapp/yao/unit-test/agent/testprepare"
)

// ============================================================================
// CallHostAgent Tests
// ============================================================================

func TestCallHostAgent_NilRobot(t *testing.T) {
	_ = testprepare.PrepareSandbox(t)
	e := standard.New()
	ctx := robottypes.NewContext(context.Background(), nil)

	_, err := e.CallHostAgent(ctx, nil, &robottypes.HostInput{Scenario: "assign"}, "chat-1")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "robot cannot be nil")
}

func TestCallHostAgent_NoHostAgent(t *testing.T) {
	_ = testprepare.PrepareSandbox(t)

	orig := robottypes.GlobalPhaseAgentResolver
	robottypes.GlobalPhaseAgentResolver = nil
	defer func() { robottypes.GlobalPhaseAgentResolver = orig }()

	e := standard.New()
	ctx := robottypes.NewContext(context.Background(), nil)

	t.Run("nil_config", func(t *testing.T) {
		robot := &robottypes.Robot{MemberID: "member-h2a"}
		_, err := e.CallHostAgent(ctx, robot, &robottypes.HostInput{Scenario: "assign"}, "chat-1")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "no Host Agent configured")
	})

	t.Run("nil_resources", func(t *testing.T) {
		robot := &robottypes.Robot{MemberID: "member-h2b", Config: &robottypes.Config{}}
		_, err := e.CallHostAgent(ctx, robot, &robottypes.HostInput{Scenario: "assign"}, "chat-1")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "no Host Agent configured")
	})
}

func TestCallHostAgent_AssistantNotFound(t *testing.T) {
	identity := testprepare.PrepareSandbox(t)
	e := standard.New()
	ctx := testCtx(identity)

	robot := &robottypes.Robot{
		MemberID: "member-h6",
		TeamID:   identity.AlphaTeamID,
		Config: &robottypes.Config{
			Resources: &robottypes.Resources{
				Phases: map[robottypes.Phase]string{
					robottypes.PhaseHost: "nonexistent.assistant.xyz",
				},
			},
		},
	}

	_, err := e.CallHostAgent(ctx, robot, &robottypes.HostInput{Scenario: "assign"}, "chat-h6")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "host agent")
}

func TestCallHostAgent_InputMarshalling(t *testing.T) {
	_ = testprepare.PrepareSandbox(t)

	input := &robottypes.HostInput{
		Scenario: "clarify",
		Context: &robottypes.HostContext{
			RobotStatus: &robottypes.RobotStatusSnapshot{ActiveCount: 2, MaxQuota: 5},
			AgentReply:  "What format?",
		},
	}

	assert.NotEmpty(t, input.Scenario)
	assert.NotNil(t, input.Context)
	assert.Equal(t, 2, input.Context.RobotStatus.ActiveCount)
}
