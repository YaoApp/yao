//go:build integration

package standard_test

import (
	"strings"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	agentcontext "github.com/yaoapp/yao/agent/context"
	"github.com/yaoapp/yao/agent/robot/executor/standard"
	robottypes "github.com/yaoapp/yao/agent/robot/types"
	"github.com/yaoapp/yao/unit-test/agent/testprepare"
)

// ============================================================================
// AgentCaller Tests - Single Call Mode
// ============================================================================

func TestAgentCallerSingleCall(t *testing.T) {
	identity := testprepare.PrepareSandbox(t)
	caller := standard.NewAgentCaller()
	ctx := robottypes.NewContext(testCtx(identity).Context, testAuth(identity))

	t.Run("basic_call_returns_response", func(t *testing.T) {
		result, err := caller.CallWithMessages(ctx, "tests.robot-single", "Hello, test message")

		require.NoError(t, err)
		require.NotNil(t, result)
		assert.False(t, result.IsEmpty())

		text := result.GetText()
		assert.NotEmpty(t, text)
	})

	t.Run("call_returns_text_content", func(t *testing.T) {
		result, err := caller.CallWithMessages(ctx, "tests.robot-single", "Generate inspiration report")

		require.NoError(t, err)
		require.NotNil(t, result)

		text := result.GetText()
		assert.NotEmpty(t, text)
		assert.Contains(t, text, "echo:")
	})
}

func TestAgentCallerNextHookData(t *testing.T) {
	identity := testprepare.PrepareSandbox(t)
	caller := standard.NewAgentCaller()
	ctx := robottypes.NewContext(testCtx(identity).Context, testAuth(identity))

	// tests.robot-single uses mock echo (no Next hook), so Next is nil.
	// We verify Content is populated with the echo and Next is absent.

	t.Run("mock_echo_returns_content_without_next_hook", func(t *testing.T) {
		result, err := caller.CallWithMessages(ctx, "tests.robot-single", "inspiration test")

		require.NoError(t, err)
		require.NotNil(t, result)

		assert.NotEmpty(t, result.Content, "Content should be populated by LLM completion")
		assert.Nil(t, result.Next, "Next should be nil when assistant has no Next hook")
	})

	t.Run("GetText_prefers_Content_over_Next", func(t *testing.T) {
		result, err := caller.CallWithMessages(ctx, "tests.robot-single", "goals test")

		require.NoError(t, err)
		require.NotNil(t, result)

		text := result.GetText()
		assert.NotEmpty(t, text)
		assert.Contains(t, text, "echo:")
	})

	t.Run("call_with_different_input_returns_echo", func(t *testing.T) {
		result, err := caller.CallWithMessages(ctx, "tests.robot-single", "tasks test")

		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Contains(t, result.GetText(), "echo:")
	})
}

func TestAgentCallerJSONParsing(t *testing.T) {
	_ = testprepare.PrepareSandbox(t)

	// Mock echo returns plain text, so GetJSON/GetJSONArray will fail.
	// Test the parsing methods with synthetic CallResult data instead.

	t.Run("GetJSONArray_parses_Next_hook_array", func(t *testing.T) {
		result := &standard.CallResult{
			Next: []interface{}{
				map[string]interface{}{"id": float64(1), "name": "Item 1"},
				map[string]interface{}{"id": float64(2), "name": "Item 2"},
				map[string]interface{}{"id": float64(3), "name": "Item 3"},
			},
		}

		arr, err := result.GetJSONArray()
		require.NoError(t, err)
		assert.Len(t, arr, 3)

		item1, ok := arr[0].(map[string]interface{})
		require.True(t, ok)
		assert.Equal(t, float64(1), item1["id"])
		assert.Equal(t, "Item 1", item1["name"])
	})

	t.Run("GetJSON_parses_Next_hook_map", func(t *testing.T) {
		result := &standard.CallResult{
			Next: map[string]interface{}{"type": "goals", "source": "next_hook"},
		}

		data, err := result.GetJSON()
		require.NoError(t, err)
		assert.Equal(t, "goals", data["type"])
		assert.Equal(t, "next_hook", data["source"])
	})

	t.Run("GetJSON_extracts_JSON_from_markdown_content", func(t *testing.T) {
		result := &standard.CallResult{
			Content: "Here is the result:\n```json\n{\"key\": \"value\"}\n```",
		}

		data, err := result.GetJSON()
		require.NoError(t, err)
		assert.Equal(t, "value", data["key"])
	})

	t.Run("GetJSONArray_returns_error_for_plain_text", func(t *testing.T) {
		result := &standard.CallResult{Content: "echo: array_test"}
		_, err := result.GetJSONArray()
		assert.Error(t, err)
	})
}

func TestAgentCallerAssistantNotFound(t *testing.T) {
	identity := testprepare.PrepareSandbox(t)
	caller := standard.NewAgentCaller()
	ctx := robottypes.NewContext(testCtx(identity).Context, testAuth(identity))

	t.Run("non_existent_assistant_returns_error", func(t *testing.T) {
		result, err := caller.CallWithMessages(ctx, "non.existent.assistant", "hello")

		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "assistant not found")
	})
}

func TestAgentCallerWithSystemAndUser(t *testing.T) {
	identity := testprepare.PrepareSandbox(t)
	caller := standard.NewAgentCaller()
	ctx := robottypes.NewContext(testCtx(identity).Context, testAuth(identity))

	t.Run("call_with_system_and_user_messages", func(t *testing.T) {
		result, err := caller.CallWithSystemAndUser(
			ctx, "tests.robot-single",
			"You are a helpful assistant.",
			"Generate inspiration report",
		)

		require.NoError(t, err)
		require.NotNil(t, result)
		assert.False(t, result.IsEmpty())
	})
}

// ============================================================================
// Stream Tests
// ============================================================================

func TestAgentCallerCallStream(t *testing.T) {
	identity := testprepare.PrepareSandbox(t)
	caller := standard.NewAgentCaller()
	ctx := robottypes.NewContext(testCtx(identity).Context, testAuth(identity))

	t.Run("streams_text_chunks_and_returns_result", func(t *testing.T) {
		var mu sync.Mutex
		var chunks []string

		streamFn := func(chunk *standard.StreamChunk) int {
			mu.Lock()
			defer mu.Unlock()
			if chunk.Type == "text" && chunk.Delta {
				chunks = append(chunks, chunk.Content)
			}
			return 0
		}

		result, err := caller.CallWithMessagesStream(ctx, "tests.robot-single", "Hello, test message", streamFn)

		require.NoError(t, err)
		require.NotNil(t, result)
		assert.False(t, result.IsEmpty())

		mu.Lock()
		combined := strings.Join(chunks, "")
		chunkCount := len(chunks)
		mu.Unlock()

		t.Logf("Received %d text chunks, total length: %d", chunkCount, len(combined))
		assert.Greater(t, chunkCount, 0)
		assert.NotEmpty(t, combined)
	})

	t.Run("nil_callback_works_like_non_stream_call", func(t *testing.T) {
		result, err := caller.CallStream(ctx, "tests.robot-single",
			[]agentcontext.Message{{Role: "user", Content: "Hello"}}, nil)
		require.NoError(t, err)
		require.NotNil(t, result)
		assert.False(t, result.IsEmpty())
	})

	t.Run("stream_assistant_not_found_returns_error", func(t *testing.T) {
		result, err := caller.CallWithMessagesStream(ctx, "non.existent", "hello", nil)
		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "assistant not found")
	})
}

// ============================================================================
// Conversation Tests - Multi-Turn Mode
// ============================================================================

func TestConversationMultiTurn(t *testing.T) {
	identity := testprepare.PrepareSandbox(t)
	ctx := robottypes.NewContext(testCtx(identity).Context, testAuth(identity))

	t.Run("multi_turn_conversation_maintains_state", func(t *testing.T) {
		conv := standard.NewConversation("tests.robot-conversation", "test-conv-1", 10)

		turn1, err := conv.Turn(ctx, "Plan tasks for sending weekly report")
		require.NoError(t, err)
		require.NotNil(t, turn1)
		assert.Equal(t, 1, turn1.Turn)
		assert.False(t, turn1.Result.IsEmpty(), "first turn should produce a response")

		turn2, err := conv.Turn(ctx, "Send to managers, include sales data")
		require.NoError(t, err)
		assert.Equal(t, 2, turn2.Turn)
		assert.False(t, turn2.Result.IsEmpty(), "second turn should produce a response")
	})

	t.Run("turn_count_increments_correctly", func(t *testing.T) {
		conv := standard.NewConversation("tests.robot-conversation", "test-conv-2", 10)

		assert.Equal(t, 0, conv.TurnCount())

		_, err := conv.Turn(ctx, "First message")
		require.NoError(t, err)
		assert.Equal(t, 1, conv.TurnCount())

		_, err = conv.Turn(ctx, "Second message")
		require.NoError(t, err)
		assert.Equal(t, 2, conv.TurnCount())
	})

	t.Run("exceeding_max_turns_returns_error", func(t *testing.T) {
		conv := standard.NewConversation("tests.robot-conversation", "test-conv-3", 2)

		_, err := conv.Turn(ctx, "First")
		require.NoError(t, err)
		_, err = conv.Turn(ctx, "Second")
		require.NoError(t, err)

		_, err = conv.Turn(ctx, "Third")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "max turns (2) exceeded")
	})

	t.Run("reset_clears_conversation_history", func(t *testing.T) {
		conv := standard.NewConversation("tests.robot-conversation", "test-conv-4", 5)

		_, err := conv.Turn(ctx, "First message")
		require.NoError(t, err)
		assert.Equal(t, 1, conv.TurnCount())

		conv.Reset()
		assert.Equal(t, 0, conv.TurnCount())
		assert.Empty(t, conv.Messages())
	})

	t.Run("system_prompt_preserved_after_reset", func(t *testing.T) {
		conv := standard.NewConversation("tests.robot-conversation", "test-conv-5", 5).
			WithSystemPrompt("You are a task planner.")

		msgs := conv.Messages()
		require.Len(t, msgs, 1)
		assert.Equal(t, "system", string(msgs[0].Role))

		_, err := conv.Turn(ctx, "Hello")
		require.NoError(t, err)

		conv.Reset()
		msgs = conv.Messages()
		require.Len(t, msgs, 1)
		assert.Equal(t, "system", string(msgs[0].Role))
	})
}

// ============================================================================
// CallResult Tests
// ============================================================================

func TestCallResultGetText(t *testing.T) {
	_ = testprepare.PrepareSandbox(t)

	t.Run("returns_content_when_available", func(t *testing.T) {
		result := &standard.CallResult{Content: "Hello World"}
		assert.Equal(t, "Hello World", result.GetText())
	})

	t.Run("returns_empty_for_empty_result", func(t *testing.T) {
		result := &standard.CallResult{}
		assert.Empty(t, result.GetText())
	})
}

func TestCallResultIsEmpty(t *testing.T) {
	_ = testprepare.PrepareSandbox(t)

	assert.True(t, (&standard.CallResult{}).IsEmpty())
	assert.False(t, (&standard.CallResult{Content: "test"}).IsEmpty())
	assert.False(t, (&standard.CallResult{Next: map[string]interface{}{"key": "value"}}).IsEmpty())
}

func TestExtractCodeBlock(t *testing.T) {
	_ = testprepare.PrepareSandbox(t)

	t.Run("extracts_JSON_code_block", func(t *testing.T) {
		content := "Here is the result:\n```json\n{\"key\": \"value\"}\n```"
		block := standard.ExtractCodeBlock(content)

		require.NotNil(t, block)
		assert.Equal(t, "json", block.Type)
		assert.Contains(t, block.Content, "key")
	})
}
