package opencode_test

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/yaoapp/yao/agent/caller"
	agentcontext "github.com/yaoapp/yao/agent/context"
	oauthtypes "github.com/yaoapp/yao/openapi/oauth/types"
	"github.com/yaoapp/yao/unit-test/agent/testprepare"
)

const defaultTimeout = 3 * time.Minute

type protocolCase struct {
	Name   string
	TeamID string
	UserID string
}

// ---------------------------------------------------------------------------
// Scenario 1: Oneshot — new container per request, no session persistence
// ---------------------------------------------------------------------------

func TestOpenCode_Oneshot(t *testing.T) {
	t.Skip("opencode runner is WIP — skip until stabilized")
	identity := testprepare.PrepareE2E(t)
	require.NotNil(t, caller.AgentGetterFunc)

	protocols := []protocolCase{
		{"openai", identity.BetaOpenAITeamID, identity.BetaOpenAIOwnerUserID},
		{"anthropic", identity.BetaAnthropicTeamID, identity.BetaAnthropicOwnerUserID},
	}

	for _, proto := range protocols {
		proto := proto
		t.Run(proto.Name, func(t *testing.T) {
			const assistantID = "tests.sandbox-v2.opencode-oneshot-cli"
			agent, err := caller.AgentGetterFunc(assistantID)
			require.NoError(t, err)

			chatID := fmt.Sprintf("e2e-oneshot-%s-%d", proto.Name, time.Now().UnixMilli())
			ctx := agentcontext.New(
				context.Background(),
				&oauthtypes.AuthorizedInfo{TeamID: proto.TeamID, UserID: proto.UserID},
				chatID,
			)

			resp := streamAndWait(t, agent, ctx, "Reply exactly with: hello opencode sandbox", defaultTimeout)

			require.NotNil(t, resp.Completion)
			assert.Equal(t, "assistant", resp.Completion.Role)
			content := contentString(t, resp)
			t.Logf("Oneshot response: %s", content)
			assert.Contains(t, strings.ToLower(content), "hello opencode sandbox")
		})
	}
}

// ---------------------------------------------------------------------------
// Scenario 2 & 3: Session — first turn (new conversation) + continuation
// ---------------------------------------------------------------------------

func TestOpenCode_Session(t *testing.T) {
	t.Skip("opencode runner is WIP — skip until stabilized")
	identity := testprepare.PrepareE2E(t)
	require.NotNil(t, caller.AgentGetterFunc)

	protocols := []protocolCase{
		{"openai", identity.BetaOpenAITeamID, identity.BetaOpenAIOwnerUserID},
		{"anthropic", identity.BetaAnthropicTeamID, identity.BetaAnthropicOwnerUserID},
	}

	for _, proto := range protocols {
		proto := proto
		t.Run(proto.Name, func(t *testing.T) {
			const assistantID = "tests.sandbox-v2.opencode-session-cli"
			agent, err := caller.AgentGetterFunc(assistantID)
			require.NoError(t, err)

			chatID := fmt.Sprintf("e2e-session-%s-%d", proto.Name, time.Now().UnixMilli())

			t.Run("turn1_new_session", func(t *testing.T) {
				ctx := agentcontext.New(
					context.Background(),
					&oauthtypes.AuthorizedInfo{TeamID: proto.TeamID, UserID: proto.UserID},
					chatID,
				)

				resp := streamAndWait(t, agent, ctx,
					"Remember this secret code: PINEAPPLE-42. Reply with: understood",
					defaultTimeout,
				)

				require.NotNil(t, resp.Completion)
				assert.Equal(t, "assistant", resp.Completion.Role)
				content := contentString(t, resp)
				t.Logf("Turn 1 response: %s", content)
				assert.Contains(t, strings.ToLower(content), "understood")
			})

			t.Run("turn2_continue_session", func(t *testing.T) {
				ctx := agentcontext.New(
					context.Background(),
					&oauthtypes.AuthorizedInfo{TeamID: proto.TeamID, UserID: proto.UserID},
					chatID,
				)

				resp := streamAndWait(t, agent, ctx,
					"What was the secret code I told you? Reply with just the code.",
					defaultTimeout,
				)

				require.NotNil(t, resp.Completion)
				assert.Equal(t, "assistant", resp.Completion.Role)
				content := contentString(t, resp)
				t.Logf("Turn 2 response: %s", content)
				assert.Contains(t, strings.ToLower(content), "pineapple-42",
					"continuation should recall secret from turn 1")
			})
		})
	}
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func streamAndWait(
	t *testing.T,
	agent caller.AgentCaller,
	ctx *agentcontext.Context,
	prompt string,
	timeout time.Duration,
) *agentcontext.Response {
	t.Helper()

	messages := []agentcontext.Message{{Role: "user", Content: prompt}}

	done := make(chan struct{})
	var resp *agentcontext.Response
	var streamErr error

	go func() {
		defer close(done)
		resp, streamErr = agent.Stream(ctx, messages)
	}()

	select {
	case <-done:
	case <-time.After(timeout):
		t.Fatalf("timeout after %v", timeout)
	}

	if streamErr != nil {
		t.Logf("Stream error: %v", streamErr)
	}
	require.NoError(t, streamErr, "Stream should not return error")
	require.NotNil(t, resp, "response should not be nil")

	if resp.Completion != nil {
		t.Logf("Completion: role=%s content=%v", resp.Completion.Role, resp.Completion.Content)
	}

	require.NotNil(t, ctx.Buffer, "ctx.Buffer should not be nil")
	msgs := ctx.Buffer.GetMessages()
	t.Logf("buffer message count: %d", len(msgs))
	for _, m := range msgs {
		t.Logf("  seq=%d role=%s type=%s streaming=%v",
			m.Sequence, m.Role, m.Type, m.IsStreaming)
	}

	return resp
}

func contentString(t *testing.T, resp *agentcontext.Response) string {
	t.Helper()
	s, ok := resp.Completion.Content.(string)
	require.True(t, ok, "Content should be string, got %T", resp.Completion.Content)
	return s
}
