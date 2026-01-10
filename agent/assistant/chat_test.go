package assistant_test

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/yaoapp/yao/agent/assistant"
	agentcontext "github.com/yaoapp/yao/agent/context"
	storetypes "github.com/yaoapp/yao/agent/store/types"
	"github.com/yaoapp/yao/agent/testutils"
	oauthtypes "github.com/yaoapp/yao/openapi/oauth/types"
)

func TestGetChatKBID(t *testing.T) {
	t.Run("WithTeamAndUser", func(t *testing.T) {
		teamID := "5659-5504-2879"
		userID := "4287-9400-2030-0504"

		collectionID := assistant.GetChatKBID(teamID, userID)

		// Should sanitize dashes to underscores
		expected := "chat_5659_5504_2879_4287_9400_2030_0504"
		assert.Equal(t, expected, collectionID)
		t.Logf("✓ Collection ID with team: %s", collectionID)
	})

	t.Run("WithoutTeam", func(t *testing.T) {
		teamID := ""
		userID := "4287-9400-2030-0504"

		collectionID := assistant.GetChatKBID(teamID, userID)

		// Should use chat_user_ prefix
		expected := "chat_user_4287_9400_2030_0504"
		assert.Equal(t, expected, collectionID)
		t.Logf("✓ Collection ID without team: %s", collectionID)
	})

	t.Run("Idempotent", func(t *testing.T) {
		teamID := "test-team-123"
		userID := "test-user-456"

		id1 := assistant.GetChatKBID(teamID, userID)
		id2 := assistant.GetChatKBID(teamID, userID)
		id3 := assistant.GetChatKBID(teamID, userID)

		// Same input should always produce same output
		assert.Equal(t, id1, id2)
		assert.Equal(t, id2, id3)
		t.Logf("✓ Idempotent: %s", id1)
	})

	t.Run("SanitizeSpecialChars", func(t *testing.T) {
		teamID := "team-with-dashes@123"
		userID := "user.with.dots!"

		collectionID := assistant.GetChatKBID(teamID, userID)

		// Should only contain alphanumeric and underscores
		assert.Regexp(t, "^[a-zA-Z0-9_]+$", collectionID)
		t.Logf("✓ Sanitized ID: %s", collectionID)
	})

	t.Run("EmptyUserID", func(t *testing.T) {
		teamID := "test-team"
		userID := ""

		collectionID := assistant.GetChatKBID(teamID, userID)

		// Should handle empty user ID gracefully
		expected := "chat_test_team_"
		assert.Equal(t, expected, collectionID)
		t.Logf("✓ Empty user ID handled: %s", collectionID)
	})
}

func TestPrepareKBCollection(t *testing.T) {
	testutils.Prepare(t)
	defer testutils.Clean(t)

	// Get assistant
	ast, err := assistant.Get("mohe")
	require.NoError(t, err)
	require.NotNil(t, ast)

	// Note: KB collection is now created during user login (see openapi/user/login.go)
	// These tests verify that InitializeConversation handles various scenarios gracefully

	t.Run("InitializeWithAuthorizedInfo", func(t *testing.T) {
		// Use unique IDs based on timestamp to avoid conflicts
		timestamp := fmt.Sprintf("%d", time.Now().UnixNano())
		teamID := fmt.Sprintf("test_team_%s", timestamp)
		userID := fmt.Sprintf("test_user_%s", timestamp)

		ctx := agentcontext.New(context.Background(), &oauthtypes.AuthorizedInfo{
			TeamID: teamID,
			UserID: userID,
		}, "test_chat_prepare_001")

		opts := &agentcontext.Options{}

		// InitializeConversation should succeed (KB collection created at login time)
		err := ast.InitializeConversation(ctx, opts)
		assert.NoError(t, err)
		t.Logf("✓ InitializeConversation completed successfully")
	})

	t.Run("IdempotentInitialization", func(t *testing.T) {
		// Use unique IDs based on timestamp to avoid conflicts
		timestamp := fmt.Sprintf("%d", time.Now().UnixNano())
		teamID := fmt.Sprintf("idem_team_%s", timestamp)
		userID := fmt.Sprintf("idem_user_%s", timestamp)

		ctx := agentcontext.New(context.Background(), &oauthtypes.AuthorizedInfo{
			TeamID: teamID,
			UserID: userID,
		}, "test_chat_idempotent")

		opts := &agentcontext.Options{}

		// Multiple calls should all succeed
		err1 := ast.InitializeConversation(ctx, opts)
		assert.NoError(t, err1)

		err2 := ast.InitializeConversation(ctx, opts)
		assert.NoError(t, err2)

		err3 := ast.InitializeConversation(ctx, opts)
		assert.NoError(t, err3)

		t.Logf("✓ Idempotent initialization works correctly")
	})

	t.Run("HandleMissingAuthorizedInfo", func(t *testing.T) {
		ctx := agentcontext.New(context.Background(), nil, "test_chat_no_auth") // Missing authorized info

		opts := &agentcontext.Options{}

		// Should not error, just return nil
		err := ast.InitializeConversation(ctx, opts)
		assert.NoError(t, err)
		t.Logf("✓ Correctly handled missing authorized info")
	})

	t.Run("ConcurrentInitialization", func(t *testing.T) {
		// Use unique IDs based on timestamp to avoid conflicts
		timestamp := fmt.Sprintf("%d", time.Now().UnixNano())
		teamID := fmt.Sprintf("concurrent_team_%s", timestamp)
		userID := fmt.Sprintf("concurrent_user_%s", timestamp)

		ctx := agentcontext.New(context.Background(), &oauthtypes.AuthorizedInfo{
			TeamID: teamID,
			UserID: userID,
		}, "test_chat_concurrent")

		opts := &agentcontext.Options{}

		// Launch 5 concurrent calls
		var wg sync.WaitGroup
		errors := make([]error, 5)
		for i := 0; i < 5; i++ {
			wg.Add(1)
			go func(idx int) {
				defer wg.Done()
				errors[idx] = ast.InitializeConversation(ctx, opts)
			}(i)
		}

		// Wait for all goroutines to complete
		wg.Wait()

		// All calls should succeed
		for i, err := range errors {
			assert.NoError(t, err, "Goroutine %d should not error", i)
		}

		t.Logf("✓ Concurrent initialization handled correctly")
	})
}

func TestInitializeConversation(t *testing.T) {
	testutils.Prepare(t)
	defer testutils.Clean(t)

	ast, err := assistant.Get("mohe")
	require.NoError(t, err)
	require.NotNil(t, ast)

	t.Run("FullInitialization", func(t *testing.T) {
		// Use unique IDs based on timestamp to avoid conflicts
		timestamp := fmt.Sprintf("%d", time.Now().UnixNano())
		teamID := fmt.Sprintf("init_team_%s", timestamp)
		userID := fmt.Sprintf("init_user_%s", timestamp)

		ctx := agentcontext.New(context.Background(), &oauthtypes.AuthorizedInfo{
			TeamID: teamID,
			UserID: userID,
		}, "test_init_chat_001")

		opts := &agentcontext.Options{}

		// Should initialize conversation without error
		// Note: KB collection is now created during user login, not here
		err := ast.InitializeConversation(ctx, opts)
		assert.NoError(t, err)
		t.Logf("✓ Conversation initialized successfully (KB collection created at login time)")
	})

	t.Run("SkipHistoryFlag", func(t *testing.T) {
		ctx := agentcontext.New(context.Background(), &oauthtypes.AuthorizedInfo{
			TeamID: "skip_team",
			UserID: "skip_user",
		}, "test_skip_history")

		opts := &agentcontext.Options{
			Skip: &agentcontext.Skip{
				History: true,
			},
		}

		// Should skip initialization when history flag is set
		err := ast.InitializeConversation(ctx, opts)
		assert.NoError(t, err)
		t.Logf("✓ Correctly skipped with history flag")
	})
}

// =============================================================================
// Buffer Integration Tests
// =============================================================================

func TestBufferInitialization(t *testing.T) {
	testutils.Prepare(t)
	defer testutils.Clean(t)

	ast, err := assistant.Get("mohe")
	require.NoError(t, err)
	require.NotNil(t, ast)

	t.Run("InitBufferForRootStack", func(t *testing.T) {
		ctx := agentcontext.New(context.Background(), nil, "test_chat_buffer_001")

		// Enter stack to simulate root stack
		_, _, done := agentcontext.EnterStack(ctx, ast.ID, nil)
		defer done()

		// Initialize buffer
		ast.InitBuffer(ctx)

		// Verify buffer was created
		assert.NotNil(t, ctx.Buffer, "Buffer should be initialized for root stack")
		assert.Equal(t, "test_chat_buffer_001", ctx.Buffer.ChatID())
		assert.Equal(t, ast.ID, ctx.Buffer.AssistantID())
		t.Logf("✓ Buffer initialized: chatID=%s, assistantID=%s", ctx.Buffer.ChatID(), ctx.Buffer.AssistantID())
	})

	t.Run("SkipBufferForNestedStack", func(t *testing.T) {
		ctx := agentcontext.New(context.Background(), nil, "test_chat_buffer_nested")

		// Enter root stack
		_, _, doneRoot := agentcontext.EnterStack(ctx, "root_assistant", nil)
		defer doneRoot()

		// Enter nested stack
		_, _, doneNested := agentcontext.EnterStack(ctx, "nested_assistant", nil)
		defer doneNested()

		// Try to initialize buffer (should be skipped for nested stack)
		ast.InitBuffer(ctx)

		// Buffer should be nil because we're not at root
		assert.Nil(t, ctx.Buffer, "Buffer should not be initialized for nested stack")
		t.Logf("✓ Buffer correctly skipped for nested stack")
	})

	t.Run("IdempotentBufferInit", func(t *testing.T) {
		ctx := agentcontext.New(context.Background(), nil, "test_chat_buffer_idem")

		// Enter stack
		_, _, done := agentcontext.EnterStack(ctx, ast.ID, nil)
		defer done()

		// Initialize buffer twice
		ast.InitBuffer(ctx)
		firstBuffer := ctx.Buffer

		ast.InitBuffer(ctx)
		secondBuffer := ctx.Buffer

		// Should be the same buffer instance
		assert.Same(t, firstBuffer, secondBuffer, "Buffer should be idempotent")
		t.Logf("✓ Buffer initialization is idempotent")
	})
}

func TestBufferUserInput(t *testing.T) {
	testutils.Prepare(t)
	defer testutils.Clean(t)

	ast, err := assistant.Get("mohe")
	require.NoError(t, err)

	t.Run("BufferSimpleTextInput", func(t *testing.T) {
		ctx := agentcontext.New(context.Background(), nil, "test_chat_input_001")

		// Enter stack and init buffer
		_, _, done := agentcontext.EnterStack(ctx, ast.ID, nil)
		defer done()
		ast.InitBuffer(ctx)

		// Create input messages
		inputMessages := []agentcontext.Message{
			{
				Role:    agentcontext.RoleUser,
				Content: "Hello, how are you?",
			},
		}

		// Buffer user input
		ast.BufferUserInput(ctx, inputMessages)

		// Verify buffer contains the message
		messages := ctx.Buffer.GetMessages()
		assert.Len(t, messages, 1, "Should have 1 buffered message")
		assert.Equal(t, "user", messages[0].Role)
		assert.Equal(t, "user_input", messages[0].Type)
		assert.Equal(t, "Hello, how are you?", messages[0].Props["content"])
		t.Logf("✓ User input buffered: %v", messages[0].Props)
	})

	t.Run("BufferMultipleMessages", func(t *testing.T) {
		ctx := agentcontext.New(context.Background(), nil, "test_chat_input_multi")

		// Enter stack and init buffer
		_, _, done := agentcontext.EnterStack(ctx, ast.ID, nil)
		defer done()
		ast.InitBuffer(ctx)

		// Create multiple input messages
		inputMessages := []agentcontext.Message{
			{Role: agentcontext.RoleUser, Content: "First message"},
			{Role: agentcontext.RoleUser, Content: "Second message"},
		}

		// Buffer user input
		ast.BufferUserInput(ctx, inputMessages)

		// Verify buffer contains all messages
		messages := ctx.Buffer.GetMessages()
		assert.Len(t, messages, 2, "Should have 2 buffered messages")
		assert.Equal(t, 1, messages[0].Sequence)
		assert.Equal(t, 2, messages[1].Sequence)
		t.Logf("✓ Multiple messages buffered with correct sequence")
	})

	t.Run("BufferWithNilBuffer", func(t *testing.T) {
		ctx := agentcontext.New(context.Background(), nil, "test_chat_input_nil")

		// Don't initialize buffer
		inputMessages := []agentcontext.Message{
			{Role: agentcontext.RoleUser, Content: "Test"},
		}

		// Should not panic
		ast.BufferUserInput(ctx, inputMessages)
		t.Logf("✓ BufferUserInput handles nil buffer gracefully")
	})
}

func TestBufferStepTracking(t *testing.T) {
	testutils.Prepare(t)
	defer testutils.Clean(t)

	ast, err := assistant.Get("mohe")
	require.NoError(t, err)

	t.Run("BeginAndCompleteStep", func(t *testing.T) {
		ctx := agentcontext.New(context.Background(), nil, "test_chat_step_001")

		// Enter stack and init buffer
		_, _, done := agentcontext.EnterStack(ctx, ast.ID, nil)
		defer done()
		ast.InitBuffer(ctx)

		// Set some context memory data
		if ctx.Memory != nil && ctx.Memory.Context != nil {
			ctx.Memory.Context.Set("test_key", "test_value", 0)
		}

		// Begin a step
		step := ast.BeginStep(ctx, agentcontext.StepTypeLLM, map[string]interface{}{
			"messages": []string{"Hello"},
		})

		assert.NotNil(t, step, "Step should be created")
		assert.Equal(t, agentcontext.StepTypeLLM, step.Type)
		assert.Equal(t, agentcontext.StepStatusRunning, step.Status)
		assert.NotEmpty(t, step.StackID)

		// Complete the step
		ast.CompleteStep(ctx, map[string]interface{}{
			"content": "Response",
		})

		// Verify step is completed
		steps := ctx.Buffer.GetAllSteps()
		assert.Len(t, steps, 1)
		assert.Equal(t, agentcontext.StepStatusCompleted, steps[0].Status)
		assert.Equal(t, "Response", steps[0].Output["content"])
		t.Logf("✓ Step tracking works correctly")
	})

	t.Run("ContextMemorySnapshotCapture", func(t *testing.T) {
		ctx := agentcontext.New(context.Background(), nil, "test_chat_memory_001")

		// Enter stack and init buffer
		_, _, done := agentcontext.EnterStack(ctx, ast.ID, nil)
		defer done()
		ast.InitBuffer(ctx)

		// Set context memory data before step
		require.NotNil(t, ctx.Memory)
		require.NotNil(t, ctx.Memory.Context)
		ctx.Memory.Context.Set("key1", "value1", 0)
		ctx.Memory.Context.Set("key2", 123, 0)

		// Begin step (should capture context memory snapshot)
		ast.BeginStep(ctx, agentcontext.StepTypeHookCreate, nil)

		// Verify context memory snapshot was captured
		steps := ctx.Buffer.GetAllSteps()
		require.Len(t, steps, 1)
		assert.NotNil(t, steps[0].SpaceSnapshot)
		assert.Equal(t, "value1", steps[0].SpaceSnapshot["key1"])
		assert.Equal(t, 123, steps[0].SpaceSnapshot["key2"])
		t.Logf("✓ Context memory snapshot captured: %v", steps[0].SpaceSnapshot)
	})

	t.Run("MultipleSteps", func(t *testing.T) {
		ctx := agentcontext.New(context.Background(), nil, "test_chat_multi_step")

		// Enter stack and init buffer
		_, _, done := agentcontext.EnterStack(ctx, ast.ID, nil)
		defer done()
		ast.InitBuffer(ctx)

		// Step 1: hook_create
		ast.BeginStep(ctx, agentcontext.StepTypeHookCreate, map[string]interface{}{"phase": "create"})
		ast.CompleteStep(ctx, map[string]interface{}{"result": "created"})

		// Step 2: llm
		ast.BeginStep(ctx, agentcontext.StepTypeLLM, map[string]interface{}{"phase": "llm"})
		ast.CompleteStep(ctx, map[string]interface{}{"result": "completed"})

		// Step 3: hook_next
		ast.BeginStep(ctx, agentcontext.StepTypeHookNext, map[string]interface{}{"phase": "next"})
		ast.CompleteStep(ctx, map[string]interface{}{"result": "done"})

		// Verify all steps
		steps := ctx.Buffer.GetAllSteps()
		assert.Len(t, steps, 3)
		assert.Equal(t, agentcontext.StepTypeHookCreate, steps[0].Type)
		assert.Equal(t, agentcontext.StepTypeLLM, steps[1].Type)
		assert.Equal(t, agentcontext.StepTypeHookNext, steps[2].Type)
		t.Logf("✓ Multiple steps tracked correctly")
	})
}

func TestFlushBuffer(t *testing.T) {
	testutils.Prepare(t)
	defer testutils.Clean(t)

	ast, err := assistant.Get("mohe")
	require.NoError(t, err)

	// Skip if chat store not available
	chatStore := assistant.GetChatStore()
	if chatStore == nil {
		t.Skip("Chat store not configured, skipping flush tests")
	}

	t.Run("FlushOnSuccess", func(t *testing.T) {
		chatID := fmt.Sprintf("test_flush_success_%s", uuid.New().String()[:8])
		ctx := agentcontext.New(context.Background(), nil, chatID)

		// Enter stack and init buffer
		_, _, done := agentcontext.EnterStack(ctx, ast.ID, nil)
		defer done()
		ast.InitBuffer(ctx)

		// Ensure chat exists
		err := chatStore.CreateChat(&storetypes.Chat{
			ChatID:      chatID,
			AssistantID: ast.ID,
			Status:      "active",
			Share:       "private",
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		})
		require.NoError(t, err)

		// Add some messages to buffer
		require.NotNil(t, ctx.Buffer, "Buffer should be initialized")
		ctx.Buffer.AddUserInput("Test question", "")
		ctx.Buffer.AddAssistantMessage("M1", "text", map[string]interface{}{"content": "Test answer"}, "", "", ast.ID, nil)

		// Add a step
		ast.BeginStep(ctx, agentcontext.StepTypeLLM, nil)
		ast.CompleteStep(ctx, nil)

		// Flush buffer (success case)
		ast.FlushBuffer(ctx, agentcontext.StepStatusCompleted, nil)

		// Verify messages were saved
		messages, err := chatStore.GetMessages(chatID, storetypes.MessageFilter{})
		assert.NoError(t, err)
		assert.Len(t, messages, 2, "Should have 2 messages saved")

		// Verify no resume records (success case)
		resumes, err := chatStore.GetResume(chatID)
		assert.NoError(t, err)
		assert.Len(t, resumes, 0, "Should have no resume records on success")

		// Cleanup
		chatStore.DeleteChat(chatID)
		t.Logf("✓ Buffer flushed on success: %d messages saved, no resume records", len(messages))
	})

	t.Run("FlushOnFailure", func(t *testing.T) {
		chatID := fmt.Sprintf("test_flush_fail_%s", uuid.New().String()[:8])
		ctx := agentcontext.New(context.Background(), nil, chatID)

		// Enter stack and init buffer
		_, _, done := agentcontext.EnterStack(ctx, ast.ID, nil)
		defer done()
		ast.InitBuffer(ctx)

		// Ensure chat exists
		err := chatStore.CreateChat(&storetypes.Chat{
			ChatID:      chatID,
			AssistantID: ast.ID,
			Status:      "active",
			Share:       "private",
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		})
		require.NoError(t, err)

		// Add messages
		ctx.Buffer.AddUserInput("Test question", "")

		// Add a step that will "fail"
		ast.BeginStep(ctx, agentcontext.StepTypeLLM, map[string]interface{}{"test": "data"})
		// Don't complete - simulate failure

		// Flush buffer (failure case)
		testErr := fmt.Errorf("simulated error")
		ast.FlushBuffer(ctx, agentcontext.ResumeStatusFailed, testErr)

		// Verify messages were saved
		messages, err := chatStore.GetMessages(chatID, storetypes.MessageFilter{})
		assert.NoError(t, err)
		assert.Len(t, messages, 1, "Should have 1 message saved")

		// Verify resume records were saved
		resumes, err := chatStore.GetResume(chatID)
		assert.NoError(t, err)
		assert.Len(t, resumes, 1, "Should have 1 resume record on failure")
		assert.Equal(t, agentcontext.ResumeStatusFailed, resumes[0].Status)

		// Cleanup
		chatStore.DeleteResume(chatID)
		chatStore.DeleteChat(chatID)
		t.Logf("✓ Buffer flushed on failure: messages and resume records saved")
	})

	t.Run("FlushOnInterrupt", func(t *testing.T) {
		chatID := fmt.Sprintf("test_flush_interrupt_%s", uuid.New().String()[:8])
		ctx := agentcontext.New(context.Background(), nil, chatID)

		// Enter stack and init buffer
		_, _, done := agentcontext.EnterStack(ctx, ast.ID, nil)
		defer done()
		ast.InitBuffer(ctx)

		// Ensure chat exists
		err := chatStore.CreateChat(&storetypes.Chat{
			ChatID:      chatID,
			AssistantID: ast.ID,
			Status:      "active",
			Share:       "private",
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		})
		require.NoError(t, err)

		// Add messages and steps
		ctx.Buffer.AddUserInput("Test question", "")
		ast.BeginStep(ctx, agentcontext.StepTypeLLM, nil)

		// Flush buffer (interrupt case)
		ast.FlushBuffer(ctx, agentcontext.ResumeStatusInterrupted, nil)

		// Verify resume records were saved with interrupted status
		resumes, err := chatStore.GetResume(chatID)
		assert.NoError(t, err)
		assert.Len(t, resumes, 1, "Should have 1 resume record on interrupt")
		assert.Equal(t, agentcontext.ResumeStatusInterrupted, resumes[0].Status)

		// Cleanup
		chatStore.DeleteResume(chatID)
		chatStore.DeleteChat(chatID)
		t.Logf("✓ Buffer flushed on interrupt: resume records saved with interrupted status")
	})

	t.Run("FlushWithModeAndConnector", func(t *testing.T) {
		chatID := fmt.Sprintf("test_flush_mode_%s", uuid.New().String()[:8])
		ctx := agentcontext.New(context.Background(), nil, chatID)

		// Enter stack with connector and mode options
		opts := &agentcontext.Options{
			Connector: "deepseek.v3",
			Mode:      "task",
		}
		_, _, done := agentcontext.EnterStack(ctx, ast.ID, opts)
		defer done()
		ast.InitBuffer(ctx)

		// Verify buffer has correct connector and mode
		require.NotNil(t, ctx.Buffer, "Buffer should be initialized")
		assert.Equal(t, "deepseek.v3", ctx.Buffer.Connector(), "Buffer should have connector set")
		assert.Equal(t, "task", ctx.Buffer.Mode(), "Buffer should have mode set")

		// Ensure chat exists
		err := chatStore.CreateChat(&storetypes.Chat{
			ChatID:      chatID,
			AssistantID: ast.ID,
			Status:      "active",
			Share:       "private",
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		})
		require.NoError(t, err)

		// Add some messages to buffer
		ctx.Buffer.AddUserInput("Test question for mode", "")
		ctx.Buffer.AddAssistantMessage("M1", "text", map[string]interface{}{"content": "Test answer with mode"}, "", "", ast.ID, nil)

		// Flush buffer
		ast.FlushBuffer(ctx, agentcontext.StepStatusCompleted, nil)

		// Verify messages were saved with connector and mode
		messages, err := chatStore.GetMessages(chatID, storetypes.MessageFilter{})
		assert.NoError(t, err)
		assert.Len(t, messages, 2, "Should have 2 messages saved")

		// Assistant message should have connector and mode
		var assistantMsg *storetypes.Message
		for _, msg := range messages {
			if msg.Role == "assistant" {
				assistantMsg = msg
				break
			}
		}
		require.NotNil(t, assistantMsg, "Should find assistant message")
		assert.Equal(t, "deepseek.v3", assistantMsg.Connector, "Message should have connector")
		assert.Equal(t, "task", assistantMsg.Mode, "Message should have mode")

		// Verify chat was updated with last_connector and last_mode
		chat, err := chatStore.GetChat(chatID)
		assert.NoError(t, err)
		assert.Equal(t, "deepseek.v3", chat.LastConnector, "Chat should have last_connector updated")
		assert.Equal(t, "task", chat.LastMode, "Chat should have last_mode updated")

		// Cleanup
		chatStore.DeleteChat(chatID)
		t.Logf("✓ Buffer flushed with mode and connector: connector=%s, mode=%s", chat.LastConnector, chat.LastMode)
	})
}

func TestEnsureChat(t *testing.T) {
	testutils.Prepare(t)
	defer testutils.Clean(t)

	ast, err := assistant.Get("mohe")
	require.NoError(t, err)

	// Skip if chat store not available
	chatStore := assistant.GetChatStore()
	if chatStore == nil {
		t.Skip("Chat store not configured, skipping EnsureChat tests")
	}

	t.Run("CreateNewChat", func(t *testing.T) {
		chatID := fmt.Sprintf("test_ensure_new_%s", uuid.New().String()[:8])
		ctx := agentcontext.New(context.Background(), nil, chatID)

		// Ensure chat creates it
		err := ast.EnsureChat(ctx)
		assert.NoError(t, err)

		// Verify chat was created
		chat, err := chatStore.GetChat(chatID)
		assert.NoError(t, err)
		assert.NotNil(t, chat)
		assert.Equal(t, chatID, chat.ChatID)
		assert.Equal(t, ast.ID, chat.AssistantID)
		assert.Equal(t, "active", chat.Status)

		// Cleanup
		chatStore.DeleteChat(chatID)
		t.Logf("✓ New chat created: %s", chatID)
	})

	t.Run("SkipExistingChat", func(t *testing.T) {
		chatID := fmt.Sprintf("test_ensure_exist_%s", uuid.New().String()[:8])
		ctx := agentcontext.New(context.Background(), nil, chatID)

		// Create chat first
		err := chatStore.CreateChat(&storetypes.Chat{
			ChatID:      chatID,
			AssistantID: ast.ID,
			Title:       "Existing Chat",
			Status:      "active",
			Share:       "private",
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		})
		require.NoError(t, err)

		// EnsureChat should not error
		err = ast.EnsureChat(ctx)
		assert.NoError(t, err)

		// Verify chat still has original title
		chat, err := chatStore.GetChat(chatID)
		assert.NoError(t, err)
		assert.Equal(t, "Existing Chat", chat.Title)

		// Cleanup
		chatStore.DeleteChat(chatID)
		t.Logf("✓ Existing chat preserved")
	})

	t.Run("SkipEmptyChatID", func(t *testing.T) {
		ctx := agentcontext.New(context.Background(), nil, "")

		// Should not error with empty chat ID
		err := ast.EnsureChat(ctx)
		assert.NoError(t, err)
		t.Logf("✓ Empty chat ID handled gracefully")
	})

	t.Run("CreateChatWithPermissions", func(t *testing.T) {
		chatID := fmt.Sprintf("test_ensure_perm_%s", uuid.New().String()[:8])

		// Create context with authorized info
		ctx := agentcontext.New(context.Background(), &oauthtypes.AuthorizedInfo{
			UserID:   "test_user_001",
			TeamID:   "test_team_001",
			TenantID: "test_tenant_001",
		}, chatID)

		// EnsureChat should create with permission fields
		err := ast.EnsureChat(ctx)
		assert.NoError(t, err)

		// Verify permission fields were saved
		chat, err := chatStore.GetChat(chatID)
		assert.NoError(t, err)
		assert.NotNil(t, chat)
		assert.Equal(t, "test_user_001", chat.CreatedBy, "CreatedBy should be set")
		assert.Equal(t, "test_user_001", chat.UpdatedBy, "UpdatedBy should be set")
		assert.Equal(t, "test_team_001", chat.TeamID, "TeamID should be set")
		assert.Equal(t, "test_tenant_001", chat.TenantID, "TenantID should be set")

		// Cleanup
		chatStore.DeleteChat(chatID)
		t.Logf("✓ Chat created with permission fields: user=%s, team=%s, tenant=%s",
			chat.CreatedBy, chat.TeamID, chat.TenantID)
	})

	t.Run("SkipHistoryEnabled", func(t *testing.T) {
		chatID := fmt.Sprintf("test_ensure_skip_%s", uuid.New().String()[:8])

		// Create context
		ctx := agentcontext.New(context.Background(), nil, chatID)

		// Set up stack with Skip.History = true
		ctx.Stack = &agentcontext.Stack{
			ID:          "test_stack",
			AssistantID: ast.ID,
			Depth:       0,
			Options: &agentcontext.Options{
				Skip: &agentcontext.Skip{
					History: true,
				},
			},
		}

		// EnsureChat should NOT create chat when Skip.History is true
		err := ast.EnsureChat(ctx)
		assert.NoError(t, err)

		// Verify chat was NOT created
		_, err = chatStore.GetChat(chatID)
		assert.Error(t, err, "Chat should not be created when Skip.History is true")
		t.Logf("✓ Chat not created when Skip.History is true")
	})
}

func TestConvertBufferedTypes(t *testing.T) {
	t.Run("ConvertBufferedMessages", func(t *testing.T) {
		// Create buffered messages
		buffered := []*agentcontext.BufferedMessage{
			{
				MessageID: "msg_001",
				ChatID:    "chat_001",
				RequestID: "req_001",
				Role:      "user",
				Type:      "user_input",
				Props:     map[string]interface{}{"content": "Hello"},
				Sequence:  1,
				CreatedAt: time.Now(),
			},
			{
				MessageID:   "msg_002",
				ChatID:      "chat_001",
				RequestID:   "req_001",
				Role:        "assistant",
				Type:        "text",
				Props:       map[string]interface{}{"content": "Hi there!"},
				BlockID:     "block_001",
				AssistantID: "test_assistant",
				Sequence:    2,
				CreatedAt:   time.Now(),
			},
		}

		// Verify structure matches store types
		assert.Len(t, buffered, 2)
		assert.Equal(t, "user", buffered[0].Role)
		assert.Equal(t, "assistant", buffered[1].Role)
		assert.Equal(t, "block_001", buffered[1].BlockID)
		t.Logf("✓ Buffered messages have correct structure")
	})

	t.Run("ConvertBufferedSteps", func(t *testing.T) {
		// Create buffered steps
		buffered := []*agentcontext.BufferedStep{
			{
				ResumeID:      "resume_001",
				ChatID:        "chat_001",
				RequestID:     "req_001",
				AssistantID:   "test_assistant",
				StackID:       "stack_001",
				StackDepth:    0,
				Type:          agentcontext.StepTypeLLM,
				Status:        agentcontext.ResumeStatusFailed,
				Input:         map[string]interface{}{"messages": []string{"Hello"}},
				SpaceSnapshot: map[string]interface{}{"key": "value"},
				Error:         "Test error",
				Sequence:      1,
				CreatedAt:     time.Now(),
			},
		}

		// Verify structure
		assert.Len(t, buffered, 1)
		assert.Equal(t, agentcontext.StepTypeLLM, buffered[0].Type)
		assert.Equal(t, agentcontext.ResumeStatusFailed, buffered[0].Status)
		assert.Equal(t, "Test error", buffered[0].Error)
		assert.Equal(t, "value", buffered[0].SpaceSnapshot["key"])
		t.Logf("✓ Buffered steps have correct structure")
	})
}
