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
	sandboxtestutils "github.com/yaoapp/yao/agent/sandbox/v2/testutils"
	oauthtypes "github.com/yaoapp/yao/openapi/oauth/types"
)

const defaultTimeout = 3 * time.Minute

// ---------------------------------------------------------------------------
// Scenario 1: Oneshot — new container per request, no session persistence
// ---------------------------------------------------------------------------

func TestOpenCode_Oneshot(t *testing.T) {
	sandboxtestutils.Prepare(t)
	defer sandboxtestutils.Clean(t)
	require.NotNil(t, caller.AgentGetterFunc)

	const assistantID = "tests.sandbox-v2.opencode-oneshot-cli"
	agent, err := caller.AgentGetterFunc(assistantID)
	require.NoError(t, err)

	chatID := fmt.Sprintf("e2e-oneshot-%d", time.Now().UnixMilli())
	ctx := agentcontext.New(
		context.Background(),
		&oauthtypes.AuthorizedInfo{TeamID: "test-team-e2e", UserID: "test-user-e2e"},
		chatID,
	)

	resp := streamAndWait(t, agent, ctx, "Reply exactly with: hello opencode sandbox", defaultTimeout)

	require.NotNil(t, resp.Completion)
	assert.Equal(t, "assistant", resp.Completion.Role)
	content := contentString(t, resp)
	t.Logf("Oneshot response: %s", content)
	assert.Contains(t, strings.ToLower(content), "hello opencode sandbox")
}

// ---------------------------------------------------------------------------
// Scenario 2 & 3: Session — first turn (new conversation) + continuation
// ---------------------------------------------------------------------------

func TestOpenCode_Session(t *testing.T) {
	sandboxtestutils.Prepare(t)
	defer sandboxtestutils.Clean(t)
	require.NotNil(t, caller.AgentGetterFunc)

	const assistantID = "tests.sandbox-v2.opencode-session-cli"
	agent, err := caller.AgentGetterFunc(assistantID)
	require.NoError(t, err)

	chatID := fmt.Sprintf("e2e-session-%d", time.Now().UnixMilli())

	// ── Turn 1: first message — creates a new session ──────────────────
	t.Run("turn1_new_session", func(t *testing.T) {
		ctx := agentcontext.New(
			context.Background(),
			&oauthtypes.AuthorizedInfo{TeamID: "test-team-e2e", UserID: "test-user-e2e"},
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

	// ── Turn 2: continuation — reuses the session ──────────────────────
	t.Run("turn2_continue_session", func(t *testing.T) {
		ctx := agentcontext.New(
			context.Background(),
			&oauthtypes.AuthorizedInfo{TeamID: "test-team-e2e", UserID: "test-user-e2e"},
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
}

// ---------------------------------------------------------------------------
// Scenario 4: No vision connector — read.ts should NOT be copied
// ---------------------------------------------------------------------------

func TestOpenCode_NoVision_ReadToolNotCopied(t *testing.T) {
	sandboxtestutils.Prepare(t)
	defer sandboxtestutils.Clean(t)
	require.NotNil(t, caller.AgentGetterFunc)

	const assistantID = "tests.sandbox-v2.opencode-oneshot-cli"
	agent, err := caller.AgentGetterFunc(assistantID)
	require.NoError(t, err)

	chatID := fmt.Sprintf("e2e-novision-%d", time.Now().UnixMilli())
	ctx := agentcontext.New(
		context.Background(),
		&oauthtypes.AuthorizedInfo{TeamID: "test-team-e2e", UserID: "test-user-e2e"},
		chatID,
	)

	resp := streamAndWait(t, agent, ctx,
		`Check if the file $HOME/.config/opencode/tools/read.ts exists. `+
			`Reply with exactly "READ_EXISTS" if it does, or "READ_MISSING" if it does not. Nothing else.`,
		defaultTimeout,
	)

	require.NotNil(t, resp.Completion)
	content := strings.ToLower(contentString(t, resp))
	t.Logf("NoVision check: %s", content)
	assert.Contains(t, content, "read_missing",
		"without vision connector, read.ts should NOT be copied")
}

// ---------------------------------------------------------------------------
// Scenario 5: With vision connector — read.ts SHOULD be copied
// ---------------------------------------------------------------------------

func TestOpenCode_Vision_ReadToolCopied(t *testing.T) {
	sandboxtestutils.Prepare(t)
	defer sandboxtestutils.Clean(t)
	require.NotNil(t, caller.AgentGetterFunc)

	const assistantID = "tests.sandbox-v2.opencode-vision-cli"
	agent, err := caller.AgentGetterFunc(assistantID)
	require.NoError(t, err)

	chatID := fmt.Sprintf("e2e-vision-%d", time.Now().UnixMilli())
	ctx := agentcontext.New(
		context.Background(),
		&oauthtypes.AuthorizedInfo{TeamID: "test-team-e2e", UserID: "test-user-e2e"},
		chatID,
	)

	resp := streamAndWait(t, agent, ctx,
		`Check if the file $HOME/.config/opencode/tools/read.ts exists. `+
			`Reply with exactly "READ_EXISTS" if it does, or "READ_MISSING" if it does not. Nothing else.`,
		defaultTimeout,
	)

	require.NotNil(t, resp.Completion)
	content := strings.ToLower(contentString(t, resp))
	t.Logf("Vision check: %s", content)
	assert.Contains(t, content, "read_exists",
		"with vision connector, read.ts SHOULD be copied")
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
