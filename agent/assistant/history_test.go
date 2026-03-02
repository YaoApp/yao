package assistant_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/yaoapp/yao/agent/assistant"
	agentcontext "github.com/yaoapp/yao/agent/context"
	"github.com/yaoapp/yao/agent/output/message"
	storetypes "github.com/yaoapp/yao/agent/store/types"
	"github.com/yaoapp/yao/agent/testutils"
	oauthtypes "github.com/yaoapp/yao/openapi/oauth/types"
)

// =============================================================================
// Helper Functions
// =============================================================================

// newHistoryTestContext creates a test context for history tests
func newHistoryTestContext(chatID string) *agentcontext.Context {
	authorized := &oauthtypes.AuthorizedInfo{
		Subject:  "test-user",
		UserID:   "history-test-user",
		TeamID:   "history-test-team",
		TenantID: "history-test-tenant",
	}

	ctx := agentcontext.New(context.Background(), authorized, chatID)
	ctx.AssistantID = "tests.history"
	ctx.Locale = "en-us"
	ctx.Client = agentcontext.Client{
		Type: "web",
		IP:   "127.0.0.1",
	}
	ctx.Referer = agentcontext.RefererAPI
	ctx.Accept = agentcontext.AcceptWebCUI
	ctx.IDGenerator = message.NewIDGenerator()
	ctx.Metadata = make(map[string]interface{})
	return ctx
}

// =============================================================================
// WithHistory Tests
// =============================================================================

func TestWithHistory(t *testing.T) {
	testutils.Prepare(t)
	defer testutils.Clean(t)

	// Get assistant
	ast, err := assistant.Get("tests.history")
	require.NoError(t, err)
	require.NotNil(t, ast)

	// Get chat store for setup/cleanup
	chatStore := assistant.GetChatStore()
	if chatStore == nil {
		t.Skip("Chat store not configured, skipping history tests")
	}

	t.Run("NoHistory", func(t *testing.T) {
		chatID := fmt.Sprintf("test_history_none_%s", uuid.New().String()[:8])
		ctx := newHistoryTestContext(chatID)

		// Create chat without any messages
		err := chatStore.CreateChat(&storetypes.Chat{
			ChatID:      chatID,
			AssistantID: ast.ID,
			Status:      "active",
			Share:       "private",
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		})
		require.NoError(t, err)
		defer chatStore.DeleteChat(chatID)

		input := []agentcontext.Message{
			{Role: agentcontext.RoleUser, Content: "Hello, this is my first message"},
		}

		result, err := ast.WithHistory(ctx, input, nil)
		require.NoError(t, err)
		require.NotNil(t, result)

		// With no history, InputMessages and FullMessages should be the same as input
		assert.Equal(t, input, result.InputMessages)
		assert.Equal(t, input, result.FullMessages)
		t.Log("✓ No history: input returned as is")
	})

	t.Run("WithExistingHistory", func(t *testing.T) {
		chatID := fmt.Sprintf("test_history_exist_%s", uuid.New().String()[:8])
		ctx := newHistoryTestContext(chatID)
		reqID := uuid.New().String()[:8]

		// Create chat
		err := chatStore.CreateChat(&storetypes.Chat{
			ChatID:      chatID,
			AssistantID: ast.ID,
			Status:      "active",
			Share:       "private",
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		})
		require.NoError(t, err)
		defer func() {
			chatStore.DeleteMessages(chatID, nil)
			chatStore.DeleteChat(chatID)
		}()

		// Add history messages
		historyMessages := []*storetypes.Message{
			{
				MessageID:   fmt.Sprintf("hist_msg_1_%s", reqID),
				ChatID:      chatID,
				RequestID:   fmt.Sprintf("req_%s", reqID),
				Role:        "user",
				Type:        "user_input",
				Props:       map[string]interface{}{"content": "Previous question"},
				Sequence:    1,
				AssistantID: ast.ID,
				CreatedAt:   time.Now().Add(-2 * time.Minute),
			},
			{
				MessageID:   fmt.Sprintf("hist_msg_2_%s", reqID),
				ChatID:      chatID,
				RequestID:   fmt.Sprintf("req_%s", reqID),
				Role:        "assistant",
				Type:        "text",
				Props:       map[string]interface{}{"text": "Previous answer"},
				Sequence:    2,
				AssistantID: ast.ID,
				CreatedAt:   time.Now().Add(-1 * time.Minute),
			},
		}

		err = chatStore.SaveMessages(chatID, historyMessages)
		require.NoError(t, err)

		// New input message
		input := []agentcontext.Message{
			{Role: agentcontext.RoleUser, Content: "New question"},
		}

		result, err := ast.WithHistory(ctx, input, nil)
		require.NoError(t, err)
		require.NotNil(t, result)

		// InputMessages should be unchanged (no overlap)
		assert.Equal(t, input, result.InputMessages)

		// FullMessages should have history + input
		assert.Len(t, result.FullMessages, 3) // 2 history + 1 new

		// Verify order: history first, then input
		assert.Equal(t, agentcontext.RoleUser, result.FullMessages[0].Role)
		assert.Equal(t, "Previous question", result.FullMessages[0].Content)
		assert.Equal(t, agentcontext.RoleAssistant, result.FullMessages[1].Role)
		assert.Equal(t, "Previous answer", result.FullMessages[1].Content)
		assert.Equal(t, agentcontext.RoleUser, result.FullMessages[2].Role)
		assert.Equal(t, "New question", result.FullMessages[2].Content)

		t.Log("✓ History merged correctly with new input")
	})

	t.Run("SkipHistoryOption", func(t *testing.T) {
		chatID := fmt.Sprintf("test_history_skip_%s", uuid.New().String()[:8])
		ctx := newHistoryTestContext(chatID)
		reqID := uuid.New().String()[:8]

		// Create chat with history
		err := chatStore.CreateChat(&storetypes.Chat{
			ChatID:      chatID,
			AssistantID: ast.ID,
			Status:      "active",
			Share:       "private",
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		})
		require.NoError(t, err)
		defer func() {
			chatStore.DeleteMessages(chatID, nil)
			chatStore.DeleteChat(chatID)
		}()

		// Add history message
		err = chatStore.SaveMessages(chatID, []*storetypes.Message{
			{
				MessageID:   fmt.Sprintf("skip_hist_1_%s", reqID),
				ChatID:      chatID,
				RequestID:   fmt.Sprintf("req_skip_%s", reqID),
				Role:        "user",
				Type:        "user_input",
				Props:       map[string]interface{}{"content": "Should be skipped"},
				Sequence:    1,
				AssistantID: ast.ID,
				CreatedAt:   time.Now(),
			},
		})
		require.NoError(t, err)

		input := []agentcontext.Message{
			{Role: agentcontext.RoleUser, Content: "Only this should appear"},
		}

		// Use Skip.History option
		opts := &agentcontext.Options{
			Skip: &agentcontext.Skip{
				History: true,
			},
		}

		result, err := ast.WithHistory(ctx, input, nil, opts)
		require.NoError(t, err)
		require.NotNil(t, result)

		// Both should be same as input (history skipped)
		assert.Equal(t, input, result.InputMessages)
		assert.Equal(t, input, result.FullMessages)
		assert.Len(t, result.FullMessages, 1)

		t.Log("✓ History skipped when Skip.History=true")
	})

	t.Run("OverlapDetection", func(t *testing.T) {
		chatID := fmt.Sprintf("test_history_overlap_%s", uuid.New().String()[:8])
		ctx := newHistoryTestContext(chatID)
		reqID := uuid.New().String()[:8]

		// Create chat
		err := chatStore.CreateChat(&storetypes.Chat{
			ChatID:      chatID,
			AssistantID: ast.ID,
			Status:      "active",
			Share:       "private",
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		})
		require.NoError(t, err)
		defer func() {
			chatStore.DeleteMessages(chatID, nil)
			chatStore.DeleteChat(chatID)
		}()

		// Add history messages
		err = chatStore.SaveMessages(chatID, []*storetypes.Message{
			{
				MessageID:   fmt.Sprintf("overlap_1_%s", reqID),
				ChatID:      chatID,
				RequestID:   fmt.Sprintf("req_overlap_%s", reqID),
				Role:        "user",
				Type:        "user_input",
				Props:       map[string]interface{}{"content": "Message one"},
				Sequence:    1,
				AssistantID: ast.ID,
				CreatedAt:   time.Now().Add(-3 * time.Minute),
			},
			{
				MessageID:   fmt.Sprintf("overlap_2_%s", reqID),
				ChatID:      chatID,
				RequestID:   fmt.Sprintf("req_overlap_%s", reqID),
				Role:        "assistant",
				Type:        "text",
				Props:       map[string]interface{}{"text": "Response one"},
				Sequence:    2,
				AssistantID: ast.ID,
				CreatedAt:   time.Now().Add(-2 * time.Minute),
			},
			{
				MessageID:   fmt.Sprintf("overlap_3_%s", reqID),
				ChatID:      chatID,
				RequestID:   fmt.Sprintf("req_overlap_2_%s", reqID),
				Role:        "user",
				Type:        "user_input",
				Props:       map[string]interface{}{"content": "Message two"},
				Sequence:    3,
				AssistantID: ast.ID,
				CreatedAt:   time.Now().Add(-1 * time.Minute),
			},
		})
		require.NoError(t, err)

		// Input that overlaps with history (includes last messages)
		// Some clients send full history + new message
		input := []agentcontext.Message{
			{Role: agentcontext.RoleAssistant, Content: "Response one"}, // Overlap
			{Role: agentcontext.RoleUser, Content: "Message two"},       // Overlap
			{Role: agentcontext.RoleUser, Content: "New message"},       // New
		}

		result, err := ast.WithHistory(ctx, input, nil)
		require.NoError(t, err)
		require.NotNil(t, result)

		// InputMessages should have overlap removed
		assert.Len(t, result.InputMessages, 1, "Should remove 2 overlapping messages")
		assert.Equal(t, "New message", result.InputMessages[0].Content)

		// FullMessages should be history + clean input
		assert.Len(t, result.FullMessages, 4) // 3 history + 1 new

		t.Log("✓ Overlap detected and removed from input")
	})

	t.Run("EmptyChatID", func(t *testing.T) {
		ctx := newHistoryTestContext("")

		input := []agentcontext.Message{
			{Role: agentcontext.RoleUser, Content: "No chat ID"},
		}

		result, err := ast.WithHistory(ctx, input, nil)
		require.NoError(t, err)
		require.NotNil(t, result)

		// With empty chat ID, should return input as is
		assert.Equal(t, input, result.InputMessages)
		assert.Equal(t, input, result.FullMessages)

		t.Log("✓ Empty chat ID handled gracefully")
	})

	t.Run("MultipleUserMessages", func(t *testing.T) {
		chatID := fmt.Sprintf("test_history_multi_%s", uuid.New().String()[:8])
		ctx := newHistoryTestContext(chatID)
		reqID := uuid.New().String()[:8]

		// Create chat with history
		err := chatStore.CreateChat(&storetypes.Chat{
			ChatID:      chatID,
			AssistantID: ast.ID,
			Status:      "active",
			Share:       "private",
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		})
		require.NoError(t, err)
		defer func() {
			chatStore.DeleteMessages(chatID, nil)
			chatStore.DeleteChat(chatID)
		}()

		// Add history
		err = chatStore.SaveMessages(chatID, []*storetypes.Message{
			{
				MessageID:   fmt.Sprintf("multi_1_%s", reqID),
				ChatID:      chatID,
				RequestID:   fmt.Sprintf("req_multi_%s", reqID),
				Role:        "user",
				Type:        "user_input",
				Props:       map[string]interface{}{"content": "First"},
				Sequence:    1,
				AssistantID: ast.ID,
				CreatedAt:   time.Now().Add(-1 * time.Minute),
			},
		})
		require.NoError(t, err)

		// Multiple input messages
		input := []agentcontext.Message{
			{Role: agentcontext.RoleUser, Content: "Second"},
			{Role: agentcontext.RoleUser, Content: "Third"},
		}

		result, err := ast.WithHistory(ctx, input, nil)
		require.NoError(t, err)
		require.NotNil(t, result)

		assert.Len(t, result.InputMessages, 2)
		assert.Len(t, result.FullMessages, 3) // 1 history + 2 new

		t.Log("✓ Multiple input messages handled correctly")
	})
}

// =============================================================================
// History Load Tests
// =============================================================================

func TestHistoryLoading(t *testing.T) {
	testutils.Prepare(t)
	defer testutils.Clean(t)

	ast, err := assistant.Get("tests.history")
	require.NoError(t, err)

	chatStore := assistant.GetChatStore()
	if chatStore == nil {
		t.Skip("Chat store not configured")
	}

	t.Run("FilterNonConversationTypes", func(t *testing.T) {
		chatID := fmt.Sprintf("test_filter_%s", uuid.New().String()[:8])
		ctx := newHistoryTestContext(chatID)
		reqID := uuid.New().String()[:8]

		// Create chat
		err := chatStore.CreateChat(&storetypes.Chat{
			ChatID:      chatID,
			AssistantID: ast.ID,
			Status:      "active",
			Share:       "private",
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		})
		require.NoError(t, err)
		defer func() {
			chatStore.DeleteMessages(chatID, nil)
			chatStore.DeleteChat(chatID)
		}()

		// Add various message types (only user/assistant roles allowed by DB constraint)
		// loadHistory filters by role (user/assistant only) and converts based on type:
		// - loading/event: skipped (no semantic value)
		// - tool_call/action: converted to historical summary text
		// - text/user_input/error: kept as-is
		err = chatStore.SaveMessages(chatID, []*storetypes.Message{
			{
				MessageID:   fmt.Sprintf("filter_1_%s", reqID),
				ChatID:      chatID,
				RequestID:   fmt.Sprintf("req_filter_%s", reqID),
				Role:        "user",
				Type:        "user_input",
				Props:       map[string]interface{}{"content": "User message"},
				Sequence:    1,
				AssistantID: ast.ID,
				CreatedAt:   time.Now().Add(-3 * time.Minute),
			},
			{
				MessageID:   fmt.Sprintf("filter_2_%s", reqID),
				ChatID:      chatID,
				RequestID:   fmt.Sprintf("req_filter_%s", reqID),
				Role:        "assistant",
				Type:        "loading",
				Props:       map[string]interface{}{"text": "Loading..."},
				Sequence:    2,
				AssistantID: ast.ID,
				CreatedAt:   time.Now().Add(-2 * time.Minute),
			},
			{
				MessageID:   fmt.Sprintf("filter_3_%s", reqID),
				ChatID:      chatID,
				RequestID:   fmt.Sprintf("req_filter_%s", reqID),
				Role:        "assistant",
				Type:        "text",
				Props:       map[string]interface{}{"text": "Assistant response"},
				Sequence:    3,
				AssistantID: ast.ID,
				CreatedAt:   time.Now().Add(-1 * time.Minute),
			},
		})
		require.NoError(t, err)

		input := []agentcontext.Message{
			{Role: agentcontext.RoleUser, Content: "New input"},
		}

		result, err := ast.WithHistory(ctx, input, nil)
		require.NoError(t, err)
		require.NotNil(t, result)

		// loading type is skipped (no semantic value)
		// History: user_input + text = 2 messages; plus 1 new input = 3 total
		assert.Len(t, result.FullMessages, 3)

		// Verify only user and assistant roles
		for _, msg := range result.FullMessages {
			assert.True(t, msg.Role == agentcontext.RoleUser || msg.Role == agentcontext.RoleAssistant,
				"Expected user or assistant role, got: %s", msg.Role)
		}

		t.Log("✓ Loading type filtered, user/assistant roles kept")
	})

	t.Run("ToolCallConvertedToSummary", func(t *testing.T) {
		chatID := fmt.Sprintf("test_toolcall_%s", uuid.New().String()[:8])
		ctx := newHistoryTestContext(chatID)
		reqID := uuid.New().String()[:8]

		err := chatStore.CreateChat(&storetypes.Chat{
			ChatID:      chatID,
			AssistantID: ast.ID,
			Status:      "active",
			Share:       "private",
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		})
		require.NoError(t, err)
		defer func() {
			chatStore.DeleteMessages(chatID, nil)
			chatStore.DeleteChat(chatID)
		}()

		// Add tool_call messages in both formats
		err = chatStore.SaveMessages(chatID, []*storetypes.Message{
			{
				MessageID:   fmt.Sprintf("tc_1_%s", reqID),
				ChatID:      chatID,
				RequestID:   fmt.Sprintf("req_tc_%s", reqID),
				Role:        "user",
				Type:        "user_input",
				Props:       map[string]interface{}{"content": "echo 3 ping 4"},
				Sequence:    1,
				AssistantID: ast.ID,
				CreatedAt:   time.Now().Add(-3 * time.Minute),
			},
			// Raw stream chunk format (actual DB format)
			{
				MessageID:   fmt.Sprintf("tc_2_%s", reqID),
				ChatID:      chatID,
				RequestID:   fmt.Sprintf("req_tc_%s", reqID),
				Role:        "assistant",
				Type:        "tool_call",
				Props:       map[string]interface{}{"content": `[{"index":0,"id":"call_abc","type":"function","function":{"name":"echo__ping"}}][{"index":0,"function":{"arguments":"{\"count\":3}"}}]`},
				Sequence:    2,
				AssistantID: ast.ID,
				CreatedAt:   time.Now().Add(-2 * time.Minute),
			},
			// Standard ToolCallProps format
			{
				MessageID:   fmt.Sprintf("tc_3_%s", reqID),
				ChatID:      chatID,
				RequestID:   fmt.Sprintf("req_tc_%s", reqID),
				Role:        "assistant",
				Type:        "tool_call",
				Props:       map[string]interface{}{"name": "echo__echo", "arguments": `{"message":"hello"}`},
				Sequence:    3,
				AssistantID: ast.ID,
				CreatedAt:   time.Now().Add(-1 * time.Minute),
			},
		})
		require.NoError(t, err)

		input := []agentcontext.Message{
			{Role: agentcontext.RoleUser, Content: "echo 5 ping 6"},
		}

		result, err := ast.WithHistory(ctx, input, nil)
		require.NoError(t, err)
		require.NotNil(t, result)

		// 1 user_input + 2 tool_call summaries + 1 new input = 4
		assert.Len(t, result.FullMessages, 4)

		// Verify tool_call messages are converted to summary text
		tcMsg1 := result.FullMessages[1]
		assert.Equal(t, agentcontext.RoleAssistant, tcMsg1.Role)
		assert.Contains(t, tcMsg1.Content, "[Historical Tool Call Summary]")
		assert.Contains(t, tcMsg1.Content, "echo__ping")
		assert.Contains(t, tcMsg1.Content, `{"count":3}`)

		tcMsg2 := result.FullMessages[2]
		assert.Equal(t, agentcontext.RoleAssistant, tcMsg2.Role)
		assert.Contains(t, tcMsg2.Content, "[Historical Tool Call Summary]")
		assert.Contains(t, tcMsg2.Content, "echo__echo")
		assert.Contains(t, tcMsg2.Content, `{"message":"hello"}`)

		t.Log("✓ Tool call messages converted to historical summaries")
	})

	t.Run("ActionConvertedToSummary", func(t *testing.T) {
		chatID := fmt.Sprintf("test_action_%s", uuid.New().String()[:8])
		ctx := newHistoryTestContext(chatID)
		reqID := uuid.New().String()[:8]

		err := chatStore.CreateChat(&storetypes.Chat{
			ChatID:      chatID,
			AssistantID: ast.ID,
			Status:      "active",
			Share:       "private",
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		})
		require.NoError(t, err)
		defer func() {
			chatStore.DeleteMessages(chatID, nil)
			chatStore.DeleteChat(chatID)
		}()

		err = chatStore.SaveMessages(chatID, []*storetypes.Message{
			{
				MessageID:   fmt.Sprintf("act_1_%s", reqID),
				ChatID:      chatID,
				RequestID:   fmt.Sprintf("req_act_%s", reqID),
				Role:        "user",
				Type:        "user_input",
				Props:       map[string]interface{}{"content": "Do something"},
				Sequence:    1,
				AssistantID: ast.ID,
				CreatedAt:   time.Now().Add(-2 * time.Minute),
			},
			// Action with payload
			{
				MessageID: fmt.Sprintf("act_2_%s", reqID),
				ChatID:    chatID,
				RequestID: fmt.Sprintf("req_act_%s", reqID),
				Role:      "assistant",
				Type:      "action",
				Props: map[string]interface{}{
					"name":    "robot.execute",
					"payload": map[string]interface{}{"goals": "test goal", "robot_id": "12345"},
				},
				Sequence:    2,
				AssistantID: ast.ID,
				CreatedAt:   time.Now().Add(-1 * time.Minute),
			},
		})
		require.NoError(t, err)

		input := []agentcontext.Message{
			{Role: agentcontext.RoleUser, Content: "What happened?"},
		}

		result, err := ast.WithHistory(ctx, input, nil)
		require.NoError(t, err)
		require.NotNil(t, result)

		// 1 user_input + 1 action summary + 1 new input = 3
		assert.Len(t, result.FullMessages, 3)

		actMsg := result.FullMessages[1]
		assert.Equal(t, agentcontext.RoleAssistant, actMsg.Role)
		assert.Contains(t, actMsg.Content, "[Historical Action Summary]")
		assert.Contains(t, actMsg.Content, "robot.execute")
		assert.Contains(t, actMsg.Content, "test goal")
		assert.Contains(t, actMsg.Content, "12345")

		t.Log("✓ Action messages converted to historical summaries with payload")
	})

	t.Run("ActionWithoutPayload", func(t *testing.T) {
		chatID := fmt.Sprintf("test_action_nopay_%s", uuid.New().String()[:8])
		ctx := newHistoryTestContext(chatID)
		reqID := uuid.New().String()[:8]

		err := chatStore.CreateChat(&storetypes.Chat{
			ChatID:      chatID,
			AssistantID: ast.ID,
			Status:      "active",
			Share:       "private",
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		})
		require.NoError(t, err)
		defer func() {
			chatStore.DeleteMessages(chatID, nil)
			chatStore.DeleteChat(chatID)
		}()

		err = chatStore.SaveMessages(chatID, []*storetypes.Message{
			{
				MessageID:   fmt.Sprintf("actnp_1_%s", reqID),
				ChatID:      chatID,
				RequestID:   fmt.Sprintf("req_actnp_%s", reqID),
				Role:        "assistant",
				Type:        "action",
				Props:       map[string]interface{}{"name": "navigate"},
				Sequence:    1,
				AssistantID: ast.ID,
				CreatedAt:   time.Now(),
			},
		})
		require.NoError(t, err)

		input := []agentcontext.Message{
			{Role: agentcontext.RoleUser, Content: "What happened?"},
		}

		result, err := ast.WithHistory(ctx, input, nil)
		require.NoError(t, err)
		require.NotNil(t, result)

		// 1 action summary + 1 new input = 2
		assert.Len(t, result.FullMessages, 2)

		actMsg := result.FullMessages[0]
		assert.Equal(t, agentcontext.RoleAssistant, actMsg.Role)
		assert.Contains(t, actMsg.Content, "[Historical Action Summary]")
		assert.Contains(t, actMsg.Content, "navigate")
		assert.NotContains(t, actMsg.Content, "payload")

		t.Log("✓ Action without payload handled correctly")
	})

	t.Run("ContentExtraction", func(t *testing.T) {
		chatID := fmt.Sprintf("test_extract_%s", uuid.New().String()[:8])
		ctx := newHistoryTestContext(chatID)
		reqID := uuid.New().String()[:8]

		// Create chat
		err := chatStore.CreateChat(&storetypes.Chat{
			ChatID:      chatID,
			AssistantID: ast.ID,
			Status:      "active",
			Share:       "private",
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		})
		require.NoError(t, err)
		defer func() {
			chatStore.DeleteMessages(chatID, nil)
			chatStore.DeleteChat(chatID)
		}()

		// Add messages with different content formats
		err = chatStore.SaveMessages(chatID, []*storetypes.Message{
			{
				MessageID:   fmt.Sprintf("extract_1_%s", reqID),
				ChatID:      chatID,
				RequestID:   fmt.Sprintf("req_extract_%s", reqID),
				Role:        "user",
				Type:        "user_input",
				Props:       map[string]interface{}{"content": "User content from props.content"},
				Sequence:    1,
				AssistantID: ast.ID,
				CreatedAt:   time.Now().Add(-2 * time.Minute),
			},
			{
				MessageID:   fmt.Sprintf("extract_2_%s", reqID),
				ChatID:      chatID,
				RequestID:   fmt.Sprintf("req_extract_%s", reqID),
				Role:        "assistant",
				Type:        "text",
				Props:       map[string]interface{}{"text": "Assistant content from props.text"},
				Sequence:    2,
				AssistantID: ast.ID,
				CreatedAt:   time.Now().Add(-1 * time.Minute),
			},
		})
		require.NoError(t, err)

		input := []agentcontext.Message{
			{Role: agentcontext.RoleUser, Content: "New message"},
		}

		result, err := ast.WithHistory(ctx, input, nil)
		require.NoError(t, err)
		require.NotNil(t, result)

		// Verify content was extracted correctly
		assert.Len(t, result.FullMessages, 3)
		assert.Equal(t, "User content from props.content", result.FullMessages[0].Content)
		assert.Equal(t, "Assistant content from props.text", result.FullMessages[1].Content)

		t.Log("✓ Content extracted correctly from different formats")
	})
}

// =============================================================================
// Edge Cases Tests
// =============================================================================

func TestHistoryEdgeCases(t *testing.T) {
	testutils.Prepare(t)
	defer testutils.Clean(t)

	ast, err := assistant.Get("tests.history")
	require.NoError(t, err)

	chatStore := assistant.GetChatStore()
	if chatStore == nil {
		t.Skip("Chat store not configured")
	}

	t.Run("EmptyInput", func(t *testing.T) {
		chatID := fmt.Sprintf("test_empty_input_%s", uuid.New().String()[:8])
		ctx := newHistoryTestContext(chatID)
		reqID := uuid.New().String()[:8]

		// Create chat with history
		err := chatStore.CreateChat(&storetypes.Chat{
			ChatID:      chatID,
			AssistantID: ast.ID,
			Status:      "active",
			Share:       "private",
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		})
		require.NoError(t, err)
		defer func() {
			chatStore.DeleteMessages(chatID, nil)
			chatStore.DeleteChat(chatID)
		}()

		err = chatStore.SaveMessages(chatID, []*storetypes.Message{
			{
				MessageID:   fmt.Sprintf("empty_input_1_%s", reqID),
				ChatID:      chatID,
				RequestID:   fmt.Sprintf("req_empty_%s", reqID),
				Role:        "user",
				Type:        "user_input",
				Props:       map[string]interface{}{"content": "Previous"},
				Sequence:    1,
				AssistantID: ast.ID,
				CreatedAt:   time.Now(),
			},
		})
		require.NoError(t, err)

		// Empty input
		input := []agentcontext.Message{}

		result, err := ast.WithHistory(ctx, input, nil)
		require.NoError(t, err)
		require.NotNil(t, result)

		// Should return history only
		assert.Empty(t, result.InputMessages)
		assert.Len(t, result.FullMessages, 1)

		t.Log("✓ Empty input handled correctly")
	})

	t.Run("FullOverlap", func(t *testing.T) {
		chatID := fmt.Sprintf("test_full_overlap_%s", uuid.New().String()[:8])
		ctx := newHistoryTestContext(chatID)
		reqID := uuid.New().String()[:8]

		// Create chat
		err := chatStore.CreateChat(&storetypes.Chat{
			ChatID:      chatID,
			AssistantID: ast.ID,
			Status:      "active",
			Share:       "private",
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		})
		require.NoError(t, err)
		defer func() {
			chatStore.DeleteMessages(chatID, nil)
			chatStore.DeleteChat(chatID)
		}()

		// Add history
		err = chatStore.SaveMessages(chatID, []*storetypes.Message{
			{
				MessageID:   fmt.Sprintf("full_overlap_1_%s", reqID),
				ChatID:      chatID,
				RequestID:   fmt.Sprintf("req_full_%s", reqID),
				Role:        "user",
				Type:        "user_input",
				Props:       map[string]interface{}{"content": "Exact same message"},
				Sequence:    1,
				AssistantID: ast.ID,
				CreatedAt:   time.Now(),
			},
		})
		require.NoError(t, err)

		// Input is exactly the same as history
		input := []agentcontext.Message{
			{Role: agentcontext.RoleUser, Content: "Exact same message"},
		}

		result, err := ast.WithHistory(ctx, input, nil)
		require.NoError(t, err)
		require.NotNil(t, result)

		// Full overlap: clean input should be empty
		assert.Empty(t, result.InputMessages)
		// FullMessages should be just history (no duplicates)
		assert.Len(t, result.FullMessages, 1)

		t.Log("✓ Full overlap handled correctly")
	})

	t.Run("NonExistentChat", func(t *testing.T) {
		chatID := "non_existent_chat_12345"
		ctx := newHistoryTestContext(chatID)

		input := []agentcontext.Message{
			{Role: agentcontext.RoleUser, Content: "Message to non-existent chat"},
		}

		result, err := ast.WithHistory(ctx, input, nil)
		require.NoError(t, err)
		require.NotNil(t, result)

		// Should return input as is (no history found)
		assert.Equal(t, input, result.InputMessages)
		assert.Equal(t, input, result.FullMessages)

		t.Log("✓ Non-existent chat handled gracefully")
	})

	t.Run("MessageWithName", func(t *testing.T) {
		chatID := fmt.Sprintf("test_name_%s", uuid.New().String()[:8])
		ctx := newHistoryTestContext(chatID)
		reqID := uuid.New().String()[:8]

		// Create chat
		err := chatStore.CreateChat(&storetypes.Chat{
			ChatID:      chatID,
			AssistantID: ast.ID,
			Status:      "active",
			Share:       "private",
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		})
		require.NoError(t, err)
		defer func() {
			chatStore.DeleteMessages(chatID, nil)
			chatStore.DeleteChat(chatID)
		}()

		// Add message with name
		err = chatStore.SaveMessages(chatID, []*storetypes.Message{
			{
				MessageID:   fmt.Sprintf("name_msg_1_%s", reqID),
				ChatID:      chatID,
				RequestID:   fmt.Sprintf("req_name_%s", reqID),
				Role:        "user",
				Type:        "user_input",
				Props:       map[string]interface{}{"content": "Message with name", "name": "John"},
				Sequence:    1,
				AssistantID: ast.ID,
				CreatedAt:   time.Now(),
			},
		})
		require.NoError(t, err)

		input := []agentcontext.Message{
			{Role: agentcontext.RoleUser, Content: "New message"},
		}

		result, err := ast.WithHistory(ctx, input, nil)
		require.NoError(t, err)
		require.NotNil(t, result)

		// First message should have name
		assert.Len(t, result.FullMessages, 2)
		assert.NotNil(t, result.FullMessages[0].Name)
		assert.Equal(t, "John", *result.FullMessages[0].Name)

		t.Log("✓ Message name field preserved")
	})

	t.Run("EmptyContent", func(t *testing.T) {
		chatID := fmt.Sprintf("test_empty_content_%s", uuid.New().String()[:8])
		ctx := newHistoryTestContext(chatID)
		reqID := uuid.New().String()[:8]

		// Create chat
		err := chatStore.CreateChat(&storetypes.Chat{
			ChatID:      chatID,
			AssistantID: ast.ID,
			Status:      "active",
			Share:       "private",
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		})
		require.NoError(t, err)
		defer func() {
			chatStore.DeleteMessages(chatID, nil)
			chatStore.DeleteChat(chatID)
		}()

		// Add message with empty content in props
		err = chatStore.SaveMessages(chatID, []*storetypes.Message{
			{
				MessageID:   fmt.Sprintf("empty_content_1_%s", reqID),
				ChatID:      chatID,
				RequestID:   fmt.Sprintf("req_empty_content_%s", reqID),
				Role:        "user",
				Type:        "user_input",
				Props:       map[string]interface{}{}, // empty props (no content)
				Sequence:    1,
				AssistantID: ast.ID,
				CreatedAt:   time.Now(),
			},
		})
		require.NoError(t, err)

		input := []agentcontext.Message{
			{Role: agentcontext.RoleUser, Content: "New message"},
		}

		result, err := ast.WithHistory(ctx, input, nil)
		require.NoError(t, err)
		require.NotNil(t, result)

		// Message with empty props should be skipped (no content extractable)
		// Only new input should be present
		assert.Len(t, result.FullMessages, 1)
		assert.Equal(t, "New message", result.FullMessages[0].Content)

		t.Log("✓ Empty content handled gracefully (message skipped)")
	})
}
