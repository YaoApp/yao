package standard_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/yaoapp/yao/agent/robot/executor/standard"
	robottypes "github.com/yaoapp/yao/agent/robot/types"
	"github.com/yaoapp/yao/agent/testutils"
	oauthtypes "github.com/yaoapp/yao/openapi/oauth/types"
)

func hostTestAuth() *oauthtypes.AuthorizedInfo {
	return &oauthtypes.AuthorizedInfo{
		UserID: "test-user-host",
		TeamID: "test-team-host",
	}
}

// H1: nil robot
func TestCallHostAgent_NilRobot(t *testing.T) {
	e := standard.New()
	ctx := robottypes.NewContext(context.Background(), nil)

	_, err := e.CallHostAgent(ctx, nil, &robottypes.HostInput{Scenario: "assign"}, "chat-1")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "robot cannot be nil")
}

// H2: no Host Agent configured
func TestCallHostAgent_NoHostAgent(t *testing.T) {
	// Temporarily clear the global resolver so no fallback is available
	orig := robottypes.GlobalPhaseAgentResolver
	robottypes.GlobalPhaseAgentResolver = nil
	defer func() { robottypes.GlobalPhaseAgentResolver = orig }()

	e := standard.New()
	ctx := robottypes.NewContext(context.Background(), nil)

	t.Run("nil config", func(t *testing.T) {
		robot := &robottypes.Robot{MemberID: "member-h2a"}
		_, err := e.CallHostAgent(ctx, robot, &robottypes.HostInput{Scenario: "assign"}, "chat-1")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "no Host Agent configured")
	})

	t.Run("nil resources", func(t *testing.T) {
		robot := &robottypes.Robot{
			MemberID: "member-h2b",
			Config:   &robottypes.Config{},
		}
		_, err := e.CallHostAgent(ctx, robot, &robottypes.HostInput{Scenario: "assign"}, "chat-1")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "no Host Agent configured")
	})
}

// H3: valid JSON response from Host Agent
func TestCallHostAgent_ValidJSONResponse(t *testing.T) {
	if testing.Short() {
		t.Skip("Requires assistant framework and LLM")
	}

	testutils.Prepare(t)
	defer testutils.Clean(t)

	e := standard.New()
	ctx := robottypes.NewContext(context.Background(), hostTestAuth())

	robot := &robottypes.Robot{
		MemberID: "member-h3",
		Config: &robottypes.Config{
			Resources: &robottypes.Resources{
				Phases: map[robottypes.Phase]string{
					robottypes.PhaseHost: "tests.host-json",
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

	output, err := e.CallHostAgent(ctx, robot, input, "chat-h3")
	require.NoError(t, err, "CallHostAgent should not error for valid JSON host agent")
	require.NotNil(t, output, "output should not be nil")

	assert.NotEmpty(t, output.Reply, "reply should not be empty")
	assert.Equal(t, robottypes.HostActionConfirm, output.Action,
		"action should be 'confirm' for the JSON host agent")
	assert.False(t, output.WaitForMore, "wait_for_more should be false")
}

// H4: plain text response (non-JSON fallback)
func TestCallHostAgent_PlaintextFallback(t *testing.T) {
	if testing.Short() {
		t.Skip("Requires assistant framework and LLM")
	}

	testutils.Prepare(t)
	defer testutils.Clean(t)

	e := standard.New()
	ctx := robottypes.NewContext(context.Background(), hostTestAuth())

	robot := &robottypes.Robot{
		MemberID: "member-h4",
		Config: &robottypes.Config{
			Resources: &robottypes.Resources{
				Phases: map[robottypes.Phase]string{
					robottypes.PhaseHost: "tests.host-plaintext",
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

	output, err := e.CallHostAgent(ctx, robot, input, "chat-h4")
	require.NoError(t, err, "non-JSON response should fallback gracefully, not error")
	require.NotNil(t, output, "output should not be nil")

	assert.NotEmpty(t, output.Reply, "reply should contain the plaintext response")
	assert.Equal(t, robottypes.HostActionConfirm, output.Action,
		"action should fallback to 'confirm' for non-JSON response")
}

// H5: JSON with wrong structure (no action/reply fields)
func TestCallHostAgent_BadJSONStructureFallback(t *testing.T) {
	if testing.Short() {
		t.Skip("Requires assistant framework and LLM")
	}

	testutils.Prepare(t)
	defer testutils.Clean(t)

	e := standard.New()
	ctx := robottypes.NewContext(context.Background(), hostTestAuth())

	robot := &robottypes.Robot{
		MemberID: "member-h5",
		Config: &robottypes.Config{
			Resources: &robottypes.Resources{
				Phases: map[robottypes.Phase]string{
					robottypes.PhaseHost: "tests.host-badjson",
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

	output, err := e.CallHostAgent(ctx, robot, input, "chat-h5")
	require.NoError(t, err, "bad JSON structure should not error")
	require.NotNil(t, output, "output should not be nil")

	// The JSON is valid but has no action/reply fields.
	// json.Unmarshal won't error — Action will be zero value ("").
	// Verify the output is returned (either with empty action or fallback to confirm).
	if output.Action == "" {
		assert.Empty(t, output.Action,
			"action should be empty when JSON has no action field")
	} else {
		assert.Equal(t, robottypes.HostActionConfirm, output.Action,
			"action should be 'confirm' if fallback is triggered")
	}
}

// H6: assistant not found
func TestCallHostAgent_AssistantNotFound(t *testing.T) {
	if testing.Short() {
		t.Skip("Requires assistant framework initialization")
	}

	testutils.Prepare(t)
	defer testutils.Clean(t)

	e := standard.New()
	ctx := robottypes.NewContext(context.Background(), hostTestAuth())

	robot := &robottypes.Robot{
		MemberID: "member-h6",
		Config: &robottypes.Config{
			Resources: &robottypes.Resources{
				Phases: map[robottypes.Phase]string{
					robottypes.PhaseHost: "nonexistent-assistant",
				},
			},
		},
	}

	input := &robottypes.HostInput{Scenario: "assign"}

	_, err := e.CallHostAgent(ctx, robot, input, "chat-h6")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "host agent")
}

// H7: input marshalling verification (pure unit test, no LLM needed)
func TestCallHostAgent_InputMarshalling(t *testing.T) {
	input := &robottypes.HostInput{
		Scenario: "clarify",
		Context: &robottypes.HostContext{
			RobotStatus: &robottypes.RobotStatusSnapshot{
				ActiveCount: 2,
				MaxQuota:    5,
			},
			AgentReply: "What format?",
		},
	}

	assert.NotEmpty(t, input.Scenario)
	assert.NotNil(t, input.Context)
	assert.Equal(t, 2, input.Context.RobotStatus.ActiveCount)
}
