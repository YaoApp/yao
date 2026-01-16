package standard_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/yaoapp/yao/agent/robot/executor/standard"
	"github.com/yaoapp/yao/agent/robot/types"
	"github.com/yaoapp/yao/agent/testutils"
	oauthtypes "github.com/yaoapp/yao/openapi/oauth/types"
)

// testAuth returns a test auth info for agent calls
func testAuth() *oauthtypes.AuthorizedInfo {
	return &oauthtypes.AuthorizedInfo{
		UserID: "test-user-1",
		TeamID: "test-team-1",
	}
}

// ============================================================================
// AgentCaller Tests - Single Call Mode
// ============================================================================

func TestAgentCallerSingleCall(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	testutils.Prepare(t)
	defer testutils.Clean(t)

	caller := standard.NewAgentCaller()
	ctx := types.NewContext(context.Background(), testAuth())

	// Test basic call - verify assistant responds and returns parseable JSON
	// Note: LLM outputs are non-deterministic, so we test structure not exact values
	t.Run("basic call returns response", func(t *testing.T) {
		result, err := caller.CallWithMessages(ctx, "tests.robot-single", "Hello, test message")

		require.NoError(t, err)
		require.NotNil(t, result)
		assert.False(t, result.IsEmpty(), "result should not be empty")

		// Should be able to get text content
		text := result.GetText()
		assert.NotEmpty(t, text, "should have text content")
	})

	t.Run("call returns parseable JSON", func(t *testing.T) {
		result, err := caller.CallWithMessages(ctx, "tests.robot-single", "Generate inspiration report")

		require.NoError(t, err)
		require.NotNil(t, result)

		// Should return parseable JSON (content may vary)
		data, err := result.GetJSON()
		require.NoError(t, err)
		assert.NotNil(t, data)
		// Verify it has "type" field (all test responses should have this)
		assert.Contains(t, data, "type", "response should have type field")
	})
}

func TestAgentCallerNextHookData(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	testutils.Prepare(t)
	defer testutils.Clean(t)

	caller := standard.NewAgentCaller()
	ctx := types.NewContext(context.Background(), testAuth())

	t.Run("next_hook inspiration returns structured data", func(t *testing.T) {
		result, err := caller.CallWithMessages(ctx, "tests.robot-single", "next_hook inspiration test")

		require.NoError(t, err)
		require.NotNil(t, result)

		// Next hook should return structured data
		data, err := result.GetJSON()
		require.NoError(t, err)
		assert.Equal(t, "inspiration", data["type"])
		assert.Equal(t, "next_hook", data["source"])
	})

	t.Run("next_hook goals returns structured data", func(t *testing.T) {
		result, err := caller.CallWithMessages(ctx, "tests.robot-single", "next_hook goals test")

		require.NoError(t, err)
		require.NotNil(t, result)

		data, err := result.GetJSON()
		require.NoError(t, err)
		assert.Equal(t, "goals", data["type"])
		assert.Equal(t, "next_hook", data["source"])
	})

	t.Run("next_hook tasks returns structured data", func(t *testing.T) {
		result, err := caller.CallWithMessages(ctx, "tests.robot-single", "next_hook tasks test")

		require.NoError(t, err)
		require.NotNil(t, result)

		data, err := result.GetJSON()
		require.NoError(t, err)
		assert.Equal(t, "tasks", data["type"])
		assert.Equal(t, "next_hook", data["source"])
	})
}

func TestAgentCallerJSONArray(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	testutils.Prepare(t)
	defer testutils.Clean(t)

	caller := standard.NewAgentCaller()
	ctx := types.NewContext(context.Background(), testAuth())

	t.Run("array_test returns JSON array", func(t *testing.T) {
		result, err := caller.CallWithMessages(ctx, "tests.robot-single", "array_test")

		require.NoError(t, err)
		require.NotNil(t, result)

		arr, err := result.GetJSONArray()
		require.NoError(t, err)
		assert.Len(t, arr, 3)

		// Verify first item structure
		item1, ok := arr[0].(map[string]interface{})
		require.True(t, ok, "first item should be a map")
		assert.Equal(t, float64(1), item1["id"])
		assert.Equal(t, "Item 1", item1["name"])
	})
}

func TestAgentCallerEmptyResponse(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	testutils.Prepare(t)
	defer testutils.Clean(t)

	caller := standard.NewAgentCaller()
	ctx := types.NewContext(context.Background(), testAuth())

	t.Run("empty_test falls back to completion content", func(t *testing.T) {
		result, err := caller.CallWithMessages(ctx, "tests.robot-single", "empty_test")

		require.NoError(t, err)
		require.NotNil(t, result)
		// When Next hook returns null, should use Completion content
		assert.False(t, result.IsEmpty())
	})
}

func TestAgentCallerAssistantNotFound(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	testutils.Prepare(t)
	defer testutils.Clean(t)

	caller := standard.NewAgentCaller()
	ctx := types.NewContext(context.Background(), testAuth())

	t.Run("non-existent assistant returns error", func(t *testing.T) {
		result, err := caller.CallWithMessages(ctx, "non.existent.assistant", "hello")

		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "assistant not found")
	})
}

func TestAgentCallerWithSystemAndUser(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	testutils.Prepare(t)
	defer testutils.Clean(t)

	caller := standard.NewAgentCaller()
	ctx := types.NewContext(context.Background(), testAuth())

	t.Run("call with system and user messages", func(t *testing.T) {
		result, err := caller.CallWithSystemAndUser(
			ctx,
			"tests.robot-single",
			"You are a helpful assistant.",
			"Generate inspiration report",
		)

		require.NoError(t, err)
		require.NotNil(t, result)
		assert.False(t, result.IsEmpty())
	})
}

// ============================================================================
// Conversation Tests - Multi-Turn Mode
// ============================================================================

func TestConversationMultiTurn(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	testutils.Prepare(t)
	defer testutils.Clean(t)

	ctx := types.NewContext(context.Background(), testAuth())

	t.Run("multi-turn conversation maintains state", func(t *testing.T) {
		conv := standard.NewConversation("tests.robot-conversation", "test-conv-1", 10)

		// Turn 1: Start planning
		turn1, err := conv.Turn(ctx, "Plan tasks for sending weekly report")
		require.NoError(t, err)
		require.NotNil(t, turn1)
		assert.Equal(t, 1, turn1.Turn)

		data1, err := turn1.Result.GetJSON()
		require.NoError(t, err)
		// Verify basic structure - turn number and completed flag
		assert.Contains(t, data1, "turn")
		assert.Contains(t, data1, "status")
		assert.Contains(t, data1, "completed")

		// Turn 2: Continue conversation
		turn2, err := conv.Turn(ctx, "Send to managers, include sales data")
		require.NoError(t, err)
		require.NotNil(t, turn2)
		assert.Equal(t, 2, turn2.Turn)

		data2, err := turn2.Result.GetJSON()
		require.NoError(t, err)
		assert.Contains(t, data2, "turn")
		assert.Contains(t, data2, "status")

		// Turn 3: Complete with confirm/skip
		turn3, err := conv.Turn(ctx, "skip") // Use skip for deterministic completion
		require.NoError(t, err)
		require.NotNil(t, turn3)
		assert.Equal(t, 3, turn3.Turn)

		data3, err := turn3.Result.GetJSON()
		require.NoError(t, err)
		assert.Equal(t, "completed", data3["status"])
		assert.Equal(t, true, data3["completed"])
	})
}

func TestConversationTurnCount(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	testutils.Prepare(t)
	defer testutils.Clean(t)

	ctx := types.NewContext(context.Background(), testAuth())

	t.Run("turn count increments correctly", func(t *testing.T) {
		conv := standard.NewConversation("tests.robot-conversation", "test-conv-2", 10)

		assert.Equal(t, 0, conv.TurnCount())

		_, err := conv.Turn(ctx, "First message")
		require.NoError(t, err)
		assert.Equal(t, 1, conv.TurnCount())

		_, err = conv.Turn(ctx, "Second message")
		require.NoError(t, err)
		assert.Equal(t, 2, conv.TurnCount())
	})
}

func TestConversationMaxTurns(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	testutils.Prepare(t)
	defer testutils.Clean(t)

	ctx := types.NewContext(context.Background(), testAuth())

	t.Run("exceeding max turns returns error", func(t *testing.T) {
		conv := standard.NewConversation("tests.robot-conversation", "test-conv-3", 2)

		_, err := conv.Turn(ctx, "First")
		require.NoError(t, err)

		_, err = conv.Turn(ctx, "Second")
		require.NoError(t, err)

		// Third turn should fail
		_, err = conv.Turn(ctx, "Third")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "max turns (2) exceeded")
	})
}

func TestConversationMessages(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	testutils.Prepare(t)
	defer testutils.Clean(t)

	ctx := types.NewContext(context.Background(), testAuth())

	t.Run("messages history is maintained", func(t *testing.T) {
		conv := standard.NewConversation("tests.robot-conversation", "test-conv-4", 5)

		// Initially empty
		assert.Empty(t, conv.Messages())

		// After first turn
		_, err := conv.Turn(ctx, "Hello")
		require.NoError(t, err)

		msgs := conv.Messages()
		assert.GreaterOrEqual(t, len(msgs), 1) // At least user message
	})
}

func TestConversationLastResponse(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	testutils.Prepare(t)
	defer testutils.Clean(t)

	ctx := types.NewContext(context.Background(), testAuth())

	t.Run("last response returns assistant message", func(t *testing.T) {
		conv := standard.NewConversation("tests.robot-conversation", "test-conv-5", 5)

		// No response yet
		assert.Nil(t, conv.LastResponse())

		// After turn
		_, err := conv.Turn(ctx, "Start planning")
		require.NoError(t, err)

		lastResp := conv.LastResponse()
		assert.NotNil(t, lastResp)
		assert.Equal(t, "assistant", string(lastResp.Role))
	})
}

func TestConversationReset(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	testutils.Prepare(t)
	defer testutils.Clean(t)

	ctx := types.NewContext(context.Background(), testAuth())

	t.Run("reset clears conversation history", func(t *testing.T) {
		conv := standard.NewConversation("tests.robot-conversation", "test-conv-6", 5)

		_, err := conv.Turn(ctx, "First message")
		require.NoError(t, err)
		assert.Equal(t, 1, conv.TurnCount())

		conv.Reset()
		assert.Equal(t, 0, conv.TurnCount())
		assert.Empty(t, conv.Messages())
	})
}

func TestConversationWithSystemPrompt(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	testutils.Prepare(t)
	defer testutils.Clean(t)

	ctx := types.NewContext(context.Background(), testAuth())

	t.Run("system prompt is preserved after reset", func(t *testing.T) {
		conv := standard.NewConversation("tests.robot-conversation", "test-conv-7", 5).
			WithSystemPrompt("You are a task planner.")

		msgs := conv.Messages()
		require.Len(t, msgs, 1)
		assert.Equal(t, "system", string(msgs[0].Role))

		_, err := conv.Turn(ctx, "Hello")
		require.NoError(t, err)

		conv.Reset()

		// System prompt should be preserved
		msgs = conv.Messages()
		require.Len(t, msgs, 1)
		assert.Equal(t, "system", string(msgs[0].Role))
	})
}

func TestConversationSpecialCommands(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	testutils.Prepare(t)
	defer testutils.Clean(t)

	ctx := types.NewContext(context.Background(), testAuth())

	t.Run("skip command jumps to completed", func(t *testing.T) {
		conv := standard.NewConversation("tests.robot-conversation", "test-conv-8", 5)

		turn, err := conv.Turn(ctx, "skip")
		require.NoError(t, err)

		data, err := turn.Result.GetJSON()
		require.NoError(t, err)
		assert.Equal(t, "completed", data["status"])
		assert.Equal(t, true, data["completed"])
	})

	t.Run("abort command ends conversation", func(t *testing.T) {
		conv := standard.NewConversation("tests.robot-conversation", "test-conv-9", 5)

		turn, err := conv.Turn(ctx, "abort")
		require.NoError(t, err)

		data, err := turn.Result.GetJSON()
		require.NoError(t, err)
		assert.Equal(t, "aborted", data["status"])
		assert.Equal(t, true, data["completed"])
	})

	t.Run("reset command resets conversation state", func(t *testing.T) {
		conv := standard.NewConversation("tests.robot-conversation", "test-conv-10", 5)

		// First do a turn
		_, err := conv.Turn(ctx, "Start planning")
		require.NoError(t, err)

		// Then reset via command
		turn, err := conv.Turn(ctx, "reset")
		require.NoError(t, err)

		data, err := turn.Result.GetJSON()
		require.NoError(t, err)
		assert.Equal(t, "reset", data["status"])
	})
}

func TestConversationRunUntil(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	testutils.Prepare(t)
	defer testutils.Clean(t)

	ctx := types.NewContext(context.Background(), testAuth())

	t.Run("run until completion", func(t *testing.T) {
		conv := standard.NewConversation("tests.robot-conversation", "test-conv-11", 10)

		inputs := []string{
			"Plan weekly report tasks",
			"Send to team leads, include metrics",
			"confirm",
		}
		inputIdx := 0

		results, err := conv.RunUntil(
			ctx,
			func(turn int, lastResult *standard.CallResult) (string, error) {
				if inputIdx < len(inputs) {
					input := inputs[inputIdx]
					inputIdx++
					return input, nil
				}
				return "confirm", nil
			},
			func(turn int, result *standard.CallResult) (bool, error) {
				data, err := result.GetJSON()
				if err != nil {
					return false, nil
				}
				completed, ok := data["completed"].(bool)
				return ok && completed, nil
			},
		)

		require.NoError(t, err)
		require.Len(t, results, 3, "should complete in 3 turns")

		// Final result should be completed
		finalData, err := results[len(results)-1].Result.GetJSON()
		require.NoError(t, err)
		assert.Equal(t, true, finalData["completed"])
	})
}

// ============================================================================
// CallResult Tests
// ============================================================================

func TestCallResultGetText(t *testing.T) {
	t.Run("returns content when available", func(t *testing.T) {
		result := &standard.CallResult{Content: "Hello World"}
		assert.Equal(t, "Hello World", result.GetText())
	})

	t.Run("returns empty for empty result", func(t *testing.T) {
		result := &standard.CallResult{}
		assert.Equal(t, "", result.GetText())
	})
}

func TestCallResultIsEmpty(t *testing.T) {
	t.Run("empty when no content and no next", func(t *testing.T) {
		result := &standard.CallResult{}
		assert.True(t, result.IsEmpty())
	})

	t.Run("not empty when has content", func(t *testing.T) {
		result := &standard.CallResult{Content: "test"}
		assert.False(t, result.IsEmpty())
	})

	t.Run("not empty when has next", func(t *testing.T) {
		result := &standard.CallResult{Next: map[string]interface{}{"key": "value"}}
		assert.False(t, result.IsEmpty())
	})
}

// ============================================================================
// ExtractCodeBlock Tests
// ============================================================================

func TestExtractCodeBlock(t *testing.T) {
	t.Run("extracts JSON code block", func(t *testing.T) {
		content := "Here is the result:\n```json\n{\"key\": \"value\"}\n```"
		block := standard.ExtractCodeBlock(content)

		require.NotNil(t, block)
		assert.Equal(t, "json", block.Type)
		assert.Contains(t, block.Content, "key")
	})

	t.Run("returns nil for no code block", func(t *testing.T) {
		content := "Just plain text"
		block := standard.ExtractCodeBlock(content)

		// gou/text returns text type for plain text
		require.NotNil(t, block)
		assert.Equal(t, "text", block.Type)
	})
}

func TestExtractAllCodeBlocks(t *testing.T) {
	t.Run("extracts multiple code blocks", func(t *testing.T) {
		content := "```json\n{}\n```\n\n```python\nprint('hello')\n```"
		blocks := standard.ExtractAllCodeBlocks(content)

		assert.Len(t, blocks, 2)
	})
}
