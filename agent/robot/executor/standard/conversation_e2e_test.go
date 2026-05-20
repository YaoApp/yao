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

func TestConversationMultiTurnE2E(t *testing.T) {
	identity := testprepare.PrepareE2E(t)
	ctx := testCtx(identity)

	t.Run("multi_turn_maintains_state", func(t *testing.T) {
		conv := standard.NewConversation("tests.robot-conversation", "e2e-conv-1", 10)

		turn1, err := conv.Turn(ctx, "Plan tasks for sending weekly report")
		require.NoError(t, err)
		require.NotNil(t, turn1)
		assert.Equal(t, 1, turn1.Turn)
		assert.False(t, turn1.Result.IsEmpty())

		turn2, err := conv.Turn(ctx, "Send to managers, include sales data")
		require.NoError(t, err)
		require.NotNil(t, turn2)
		assert.Equal(t, 2, turn2.Turn)
	})

	t.Run("turn_count_increments", func(t *testing.T) {
		conv := standard.NewConversation("tests.robot-conversation", "e2e-conv-2", 5)
		assert.Equal(t, 0, conv.TurnCount())

		_, err := conv.Turn(ctx, "Hello")
		require.NoError(t, err)
		assert.Equal(t, 1, conv.TurnCount())
	})

	t.Run("max_turns_enforced", func(t *testing.T) {
		conv := standard.NewConversation("tests.robot-conversation", "e2e-conv-3", 2)

		_, err := conv.Turn(ctx, "Turn 1")
		require.NoError(t, err)

		_, err = conv.Turn(ctx, "Turn 2")
		require.NoError(t, err)

		_, err = conv.Turn(ctx, "Turn 3 should fail")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "max turns")
	})

	t.Run("reset_clears_state", func(t *testing.T) {
		conv := standard.NewConversation("tests.robot-conversation", "e2e-conv-4", 5)

		_, err := conv.Turn(ctx, "Hello")
		require.NoError(t, err)
		assert.Equal(t, 1, conv.TurnCount())

		conv.Reset()
		assert.Equal(t, 0, conv.TurnCount())
	})

	t.Run("messages_track_history", func(t *testing.T) {
		conv := standard.NewConversation("tests.robot-conversation", "e2e-conv-5", 10)

		_, err := conv.Turn(ctx, "First message")
		require.NoError(t, err)

		msgs := conv.Messages()
		assert.GreaterOrEqual(t, len(msgs), 1)
	})
}

func TestAgentCallerJSONArrayE2E(t *testing.T) {
	identity := testprepare.PrepareE2E(t)
	caller := standard.NewAgentCaller()
	ctx := robottypes.NewContext(testCtx(identity).Context, testAuth(identity))

	t.Run("array_test_returns_JSON_array", func(t *testing.T) {
		result, err := caller.CallWithMessages(ctx, "tests.robot-single", "array_test")

		require.NoError(t, err)
		require.NotNil(t, result)

		arr, err := result.GetJSONArray()
		require.NoError(t, err)
		assert.Len(t, arr, 3)

		item1, ok := arr[0].(map[string]interface{})
		require.True(t, ok)
		assert.Equal(t, float64(1), item1["id"])
		assert.Equal(t, "Item 1", item1["name"])
	})
}

func TestAgentCallerEmptyResponseE2E(t *testing.T) {
	identity := testprepare.PrepareE2E(t)
	caller := standard.NewAgentCaller()
	ctx := robottypes.NewContext(testCtx(identity).Context, testAuth(identity))

	t.Run("empty_test_falls_back_to_completion_content", func(t *testing.T) {
		result, err := caller.CallWithMessages(ctx, "tests.robot-single", "empty_test")

		require.NoError(t, err)
		require.NotNil(t, result)
		assert.False(t, result.IsEmpty())
	})
}
