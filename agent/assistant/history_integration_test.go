//go:build integration

package assistant_test

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/yaoapp/yao/agent/assistant"
	agentContext "github.com/yaoapp/yao/agent/context"
	storetypes "github.com/yaoapp/yao/agent/store/types"
	"github.com/yaoapp/yao/unit-test/agent/testprepare"
)

func TestWithHistoryBasic(t *testing.T) {
	testprepare.PrepareSandbox(t)

	ast, err := assistant.Get("tests.history-basic")
	require.NoError(t, err)

	chatID := "test-history-basic-" + uuid.New().String()
	ctx := newTestContext(chatID, "tests.history-basic")

	// Write some history messages
	chatStore := assistant.GetChatStore()
	require.NotNil(t, chatStore)

	// Create the chat first
	err = chatStore.CreateChat(&storetypes.Chat{
		ChatID:      chatID,
		AssistantID: "tests.history-basic",
		Status:      "active",
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	})
	require.NoError(t, err)

	// Save history messages
	historyMessages := []*storetypes.Message{
		{
			MessageID: uuid.New().String(),
			ChatID:    chatID,
			Role:      "user",
			Type:      "text",
			Props:     map[string]interface{}{"content": "What is Go?"},
			Sequence:  1,
			CreatedAt: time.Now().Add(-2 * time.Minute),
			UpdatedAt: time.Now().Add(-2 * time.Minute),
		},
		{
			MessageID: uuid.New().String(),
			ChatID:    chatID,
			Role:      "assistant",
			Type:      "text",
			Props:     map[string]interface{}{"text": "Go is a programming language."},
			Sequence:  2,
			CreatedAt: time.Now().Add(-1 * time.Minute),
			UpdatedAt: time.Now().Add(-1 * time.Minute),
		},
	}
	err = chatStore.SaveMessages(chatID, historyMessages)
	require.NoError(t, err)

	// Build input messages
	input := []agentContext.Message{
		{Role: agentContext.RoleUser, Content: "Tell me more about Go concurrency"},
	}

	// Call WithHistory
	result, err := ast.WithHistory(ctx, input, nil)
	require.NoError(t, err)
	require.NotNil(t, result)

	// FullMessages should contain history + new input
	assert.Greater(t, len(result.FullMessages), len(input),
		"FullMessages should contain history plus input")

	// The last message in FullMessages should be our input
	lastMsg := result.FullMessages[len(result.FullMessages)-1]
	assert.Equal(t, agentContext.RoleUser, lastMsg.Role)
	assert.Equal(t, "Tell me more about Go concurrency", lastMsg.Content)

	// InputMessages should be clean input
	assert.Equal(t, input, result.InputMessages)
}

func TestWithHistoryOverlap(t *testing.T) {
	testprepare.PrepareSandbox(t)

	ast, err := assistant.Get("tests.history-basic")
	require.NoError(t, err)

	chatID := "test-history-overlap-" + uuid.New().String()
	ctx := newTestContext(chatID, "tests.history-basic")

	chatStore := assistant.GetChatStore()
	require.NotNil(t, chatStore)

	// Create chat
	err = chatStore.CreateChat(&storetypes.Chat{
		ChatID:      chatID,
		AssistantID: "tests.history-basic",
		Status:      "active",
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	})
	require.NoError(t, err)

	// Write history: "Hello" and "Hi there"
	historyMessages := []*storetypes.Message{
		{
			MessageID: uuid.New().String(),
			ChatID:    chatID,
			Role:      "user",
			Type:      "text",
			Props:     map[string]interface{}{"content": "Hello"},
			Sequence:  1,
			CreatedAt: time.Now().Add(-2 * time.Minute),
			UpdatedAt: time.Now().Add(-2 * time.Minute),
		},
		{
			MessageID: uuid.New().String(),
			ChatID:    chatID,
			Role:      "assistant",
			Type:      "text",
			Props:     map[string]interface{}{"text": "Hi there"},
			Sequence:  2,
			CreatedAt: time.Now().Add(-1 * time.Minute),
			UpdatedAt: time.Now().Add(-1 * time.Minute),
		},
	}
	err = chatStore.SaveMessages(chatID, historyMessages)
	require.NoError(t, err)

	// Build input that overlaps with history: repeated "Hello" + "Hi there" + new message
	input := []agentContext.Message{
		{Role: agentContext.RoleUser, Content: "Hello"},
		{Role: agentContext.RoleAssistant, Content: "Hi there"},
		{Role: agentContext.RoleUser, Content: "New message"},
	}

	// Call WithHistory
	result, err := ast.WithHistory(ctx, input, nil)
	require.NoError(t, err)
	require.NotNil(t, result)

	// InputMessages should not contain the overlapping messages
	// Only "New message" should remain
	require.Len(t, result.InputMessages, 1)
	assert.Equal(t, "New message", result.InputMessages[0].Content)

	// FullMessages should be history + clean input (no duplicates)
	found := false
	newMsgCount := 0
	for _, msg := range result.FullMessages {
		if msg.Content == "New message" {
			found = true
			newMsgCount++
		}
	}
	assert.True(t, found, "FullMessages should contain 'New message'")
	assert.Equal(t, 1, newMsgCount, "'New message' should appear exactly once")
}

func TestWithHistoryEmpty(t *testing.T) {
	testprepare.PrepareSandbox(t)

	ast, err := assistant.Get("tests.history-basic")
	require.NoError(t, err)

	// Use a brand new chatID with no history
	chatID := "test-history-empty-" + uuid.New().String()
	ctx := newTestContext(chatID, "tests.history-basic")

	input := []agentContext.Message{
		{Role: agentContext.RoleUser, Content: "First message in a new chat"},
	}

	result, err := ast.WithHistory(ctx, input, nil)
	require.NoError(t, err)
	require.NotNil(t, result)

	// With no history, FullMessages should equal InputMessages
	assert.Equal(t, result.InputMessages, result.FullMessages)
	assert.Equal(t, input, result.InputMessages)
}

func TestWithHistorySkip(t *testing.T) {
	testprepare.PrepareSandbox(t)

	ast, err := assistant.Get("tests.history-basic")
	require.NoError(t, err)

	chatID := "test-history-skip-" + uuid.New().String()
	ctx := newTestContext(chatID, "tests.history-basic")

	chatStore := assistant.GetChatStore()
	require.NotNil(t, chatStore)

	// Create chat and save some history
	err = chatStore.CreateChat(&storetypes.Chat{
		ChatID:      chatID,
		AssistantID: "tests.history-basic",
		Status:      "active",
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	})
	require.NoError(t, err)

	historyMessages := []*storetypes.Message{
		{
			MessageID: uuid.New().String(),
			ChatID:    chatID,
			Role:      "user",
			Type:      "text",
			Props:     map[string]interface{}{"content": "Previous question"},
			Sequence:  1,
			CreatedAt: time.Now().Add(-1 * time.Minute),
			UpdatedAt: time.Now().Add(-1 * time.Minute),
		},
	}
	err = chatStore.SaveMessages(chatID, historyMessages)
	require.NoError(t, err)

	input := []agentContext.Message{
		{Role: agentContext.RoleUser, Content: "Current message"},
	}

	// Pass opts with Skip.History = true
	opts := &agentContext.Options{
		Skip: &agentContext.Skip{History: true},
	}

	result, err := ast.WithHistory(ctx, input, nil, opts)
	require.NoError(t, err)
	require.NotNil(t, result)

	// Even though there's history, FullMessages should only be input
	assert.Equal(t, input, result.FullMessages)
	assert.Equal(t, input, result.InputMessages)
}

func TestFindOverlapIndex(t *testing.T) {
	testprepare.PrepareSandbox(t)

	ast, err := assistant.Get("tests.history-basic")
	require.NoError(t, err)

	t.Run("NoOverlap", func(t *testing.T) {
		history := []agentContext.Message{
			{Role: agentContext.RoleUser, Content: "Hello"},
			{Role: agentContext.RoleAssistant, Content: "Hi"},
		}
		input := []agentContext.Message{
			{Role: agentContext.RoleUser, Content: "Completely different"},
		}

		index := assistant.ExportFindOverlapIndex(ast, history, input)
		assert.Equal(t, 0, index)
	})

	t.Run("PartialOverlap", func(t *testing.T) {
		history := []agentContext.Message{
			{Role: agentContext.RoleUser, Content: "First"},
			{Role: agentContext.RoleAssistant, Content: "Reply1"},
			{Role: agentContext.RoleUser, Content: "Second"},
			{Role: agentContext.RoleAssistant, Content: "Reply2"},
		}
		// Input starts with "Second" + "Reply2" (overlap of 2) then new message
		input := []agentContext.Message{
			{Role: agentContext.RoleUser, Content: "Second"},
			{Role: agentContext.RoleAssistant, Content: "Reply2"},
			{Role: agentContext.RoleUser, Content: "Third"},
		}

		index := assistant.ExportFindOverlapIndex(ast, history, input)
		assert.Equal(t, 2, index)
	})

	t.Run("FullOverlap", func(t *testing.T) {
		history := []agentContext.Message{
			{Role: agentContext.RoleUser, Content: "A"},
			{Role: agentContext.RoleAssistant, Content: "B"},
		}
		// Input is exactly the last 2 messages of history
		input := []agentContext.Message{
			{Role: agentContext.RoleUser, Content: "A"},
			{Role: agentContext.RoleAssistant, Content: "B"},
		}

		index := assistant.ExportFindOverlapIndex(ast, history, input)
		assert.Equal(t, 2, index)
	})
}

func TestGetHistorySize(t *testing.T) {
	t.Run("NilOpts", func(t *testing.T) {
		size := assistant.ExportGetHistorySize(nil)
		assert.Equal(t, 20, size)
	})

	t.Run("OptsWithHistorySize", func(t *testing.T) {
		opts := &agentContext.Options{HistorySize: 50}
		size := assistant.ExportGetHistorySize(opts)
		assert.Equal(t, 50, size)
	})

	t.Run("OptsWithZeroHistorySize", func(t *testing.T) {
		opts := &agentContext.Options{HistorySize: 0}
		size := assistant.ExportGetHistorySize(opts)
		// Should fall back to storeSetting or default (20)
		assert.True(t, size > 0, "should return a positive default size")
	})
}
