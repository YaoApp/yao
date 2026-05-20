//go:build integration

package assistant_test

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/yaoapp/yao/agent/assistant"
	agentContext "github.com/yaoapp/yao/agent/context"
	"github.com/yaoapp/yao/unit-test/agent/testprepare"
)

func TestEnsureChatCreate(t *testing.T) {
	testprepare.PrepareSandbox(t)

	ast, err := assistant.Get("tests.history-basic")
	require.NoError(t, err)

	chatID := "test-ensure-chat-create-" + uuid.New().String()
	ctx := newTestContext(chatID, "tests.history-basic")
	ctx.Stack = agentContext.NewStack("", ast.ID, "api", nil)

	err = ast.EnsureChat(ctx)
	require.NoError(t, err)

	// Verify chat was created
	chatStore := assistant.GetChatStore()
	require.NotNil(t, chatStore)

	chat, err := chatStore.GetChat(chatID)
	require.NoError(t, err)
	assert.Equal(t, chatID, chat.ChatID)
	assert.Equal(t, "active", chat.Status)
}

func TestEnsureChatExisting(t *testing.T) {
	testprepare.PrepareSandbox(t)

	ast, err := assistant.Get("tests.history-basic")
	require.NoError(t, err)

	chatID := "test-ensure-chat-existing-" + uuid.New().String()
	ctx := newTestContext(chatID, "tests.history-basic")
	ctx.Stack = agentContext.NewStack("", ast.ID, "api", nil)

	// Create chat the first time
	err = ast.EnsureChat(ctx)
	require.NoError(t, err)

	// Call again — should be idempotent (no error)
	err = ast.EnsureChat(ctx)
	assert.NoError(t, err)

	// Verify only one chat exists
	chatStore := assistant.GetChatStore()
	require.NotNil(t, chatStore)

	chat, err := chatStore.GetChat(chatID)
	require.NoError(t, err)
	assert.Equal(t, chatID, chat.ChatID)
}

func TestEnsureChatSkipHistory(t *testing.T) {
	testprepare.PrepareSandbox(t)

	ast, err := assistant.Get("tests.history-basic")
	require.NoError(t, err)

	chatID := "test-ensure-chat-skip-" + uuid.New().String()
	ctx := newTestContext(chatID, "tests.history-basic")

	// Set Skip.History = true via stack options
	opts := &agentContext.Options{
		Skip: &agentContext.Skip{History: true},
	}
	ctx.Stack = agentContext.NewStack("", ast.ID, "api", opts)

	err = ast.EnsureChat(ctx)
	require.NoError(t, err)

	// Chat should NOT have been created
	chatStore := assistant.GetChatStore()
	require.NotNil(t, chatStore)

	_, err = chatStore.GetChat(chatID)
	assert.Error(t, err, "chat should not exist when Skip.History is true")
}

func TestGetChatKBID(t *testing.T) {
	t.Run("WithTeamAndUser", func(t *testing.T) {
		result := assistant.GetChatKBID("team123", "user456")
		assert.Equal(t, "chat_team123_user456", result)
	})

	t.Run("OnlyUser", func(t *testing.T) {
		result := assistant.GetChatKBID("", "user456")
		assert.Equal(t, "chat_user_user456", result)
	})

	t.Run("SpecialCharacters", func(t *testing.T) {
		result := assistant.GetChatKBID("team-abc", "user@xyz")
		assert.Equal(t, "chat_team_abc_user_xyz", result)
	})
}

func TestSanitizeCollectionID(t *testing.T) {
	t.Run("NormalID", func(t *testing.T) {
		result := assistant.ExportSanitizeCollectionID("abc123_XYZ")
		assert.Equal(t, "abc123_XYZ", result)
	})

	t.Run("SpecialChars", func(t *testing.T) {
		result := assistant.ExportSanitizeCollectionID("hello-world@test.com")
		assert.Equal(t, "hello_world_test_com", result)
	})

	t.Run("EmptyString", func(t *testing.T) {
		result := assistant.ExportSanitizeCollectionID("")
		assert.Equal(t, "", result)
	})
}

func TestInitBuffer(t *testing.T) {
	testprepare.PrepareSandbox(t)

	ast, err := assistant.Get("tests.history-basic")
	require.NoError(t, err)

	chatID := "test-init-buffer-" + uuid.New().String()
	ctx := newTestContext(chatID, "tests.history-basic")

	// Create root stack context
	ctx.Stack = agentContext.NewStack("", "tests.history-basic", "api", nil)

	// Buffer should be nil initially
	assert.Nil(t, ctx.Buffer)

	// Call InitBuffer
	ast.InitBuffer(ctx)

	// Buffer should now be initialized
	assert.NotNil(t, ctx.Buffer)
}

func TestBufferUserInput(t *testing.T) {
	testprepare.PrepareSandbox(t)

	ast, err := assistant.Get("tests.history-basic")
	require.NoError(t, err)

	chatID := "test-buffer-user-input-" + uuid.New().String()
	ctx := newTestContext(chatID, "tests.history-basic")
	ctx.Stack = agentContext.NewStack("", "tests.history-basic", "api", nil)

	// Initialize buffer
	ast.InitBuffer(ctx)
	require.NotNil(t, ctx.Buffer)

	// Add user input messages
	inputMessages := []agentContext.Message{
		{Role: agentContext.RoleUser, Content: "Hello from user"},
		{Role: agentContext.RoleUser, Content: "Another message"},
	}

	ast.BufferUserInput(ctx, inputMessages)

	// Verify buffer has messages
	messages := ctx.Buffer.GetMessages()
	assert.NotEmpty(t, messages, "buffer should contain user input messages")
}
