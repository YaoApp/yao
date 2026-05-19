//go:build integration

package context_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/yaoapp/gou/store"
	agentctx "github.com/yaoapp/yao/agent/context"
	"github.com/yaoapp/yao/unit-test/agent/testprepare"
)

func getTestCache(t *testing.T) store.Store {
	t.Helper()
	cache, err := store.Get("__yao.agent.cache")
	require.NoError(t, err)
	cache.Clear()
	return cache
}

func TestGetChatIDByMessages_NewConversation(t *testing.T) {
	testprepare.PrepareSandbox(t)

	cache := getTestCache(t)

	messages := []agentctx.Message{
		{
			Role:    agentctx.RoleUser,
			Content: "Hello, how are you?",
		},
	}

	chatID1, err := agentctx.GetChatIDByMessages(cache, messages)
	require.NoError(t, err)
	assert.NotEmpty(t, chatID1)

	chatID2, err := agentctx.GetChatIDByMessages(cache, messages)
	require.NoError(t, err)
	assert.NotEmpty(t, chatID2)

	assert.NotEqual(t, chatID1, chatID2, "Single user message should generate different chat IDs each time")
}

func TestGetChatIDByMessages_ContinuousConversation(t *testing.T) {
	testprepare.PrepareSandbox(t)

	cache := getTestCache(t)

	messages1 := []agentctx.Message{
		{Role: agentctx.RoleUser, Content: "First message"},
	}
	chatID1, err := agentctx.GetChatIDByMessages(cache, messages1)
	require.NoError(t, err)

	messages2 := []agentctx.Message{
		{Role: agentctx.RoleUser, Content: "First message"},
		{Role: agentctx.RoleUser, Content: "Second message"},
	}
	chatID2, err := agentctx.GetChatIDByMessages(cache, messages2)
	require.NoError(t, err)
	assert.Equal(t, chatID1, chatID2)

	messages3 := []agentctx.Message{
		{Role: agentctx.RoleUser, Content: "First message"},
		{Role: agentctx.RoleUser, Content: "Second message"},
		{Role: agentctx.RoleUser, Content: "Third message"},
	}
	chatID3, err := agentctx.GetChatIDByMessages(cache, messages3)
	require.NoError(t, err)
	assert.Equal(t, chatID2, chatID3)

	messages4 := []agentctx.Message{
		{Role: agentctx.RoleUser, Content: "First message"},
		{Role: agentctx.RoleUser, Content: "Second message"},
		{Role: agentctx.RoleUser, Content: "Third message"},
		{Role: agentctx.RoleUser, Content: "Fourth message"},
	}
	chatID4, err := agentctx.GetChatIDByMessages(cache, messages4)
	require.NoError(t, err)
	assert.Equal(t, chatID3, chatID4)

	assert.Equal(t, chatID1, chatID4, "All chat IDs should be the same conversation")
}

func TestGetChatIDByMessages_DifferentConversations(t *testing.T) {
	testprepare.PrepareSandbox(t)

	cache := getTestCache(t)

	messages1 := []agentctx.Message{
		{Role: agentctx.RoleUser, Content: "Hello"},
	}
	chatID1, err := agentctx.GetChatIDByMessages(cache, messages1)
	require.NoError(t, err)

	err = agentctx.CacheChatID(cache, messages1, chatID1)
	require.NoError(t, err)

	messages2 := []agentctx.Message{
		{Role: agentctx.RoleUser, Content: "Goodbye"},
	}
	chatID2, err := agentctx.GetChatIDByMessages(cache, messages2)
	require.NoError(t, err)

	assert.NotEqual(t, chatID1, chatID2, "Different conversations should have different chat IDs")
}

func TestGetChatIDByMessages_MultiModalContent(t *testing.T) {
	testprepare.PrepareSandbox(t)

	cache := getTestCache(t)

	messages1 := []agentctx.Message{
		{
			Role: agentctx.RoleUser,
			Content: []agentctx.ContentPart{
				{
					Type: agentctx.ContentText,
					Text: "What's in this image?",
				},
				{
					Type: agentctx.ContentImageURL,
					ImageURL: &agentctx.ImageURL{
						URL:    "https://example.com/image.jpg",
						Detail: agentctx.DetailHigh,
					},
				},
			},
		},
	}

	chatID1, err := agentctx.GetChatIDByMessages(cache, messages1)
	require.NoError(t, err)

	messages2 := append(messages1, agentctx.Message{
		Role:    agentctx.RoleUser,
		Content: "Tell me more details",
	})

	chatID2, err := agentctx.GetChatIDByMessages(cache, messages2)
	require.NoError(t, err)

	assert.Equal(t, chatID1, chatID2, "Multimodal continuation should have same chat ID")
}

func TestGetChatIDByMessages_WithToolCalls(t *testing.T) {
	testprepare.PrepareSandbox(t)

	cache := getTestCache(t)

	messages1 := []agentctx.Message{
		{Role: agentctx.RoleUser, Content: "What's the weather in Tokyo?"},
	}

	chatID1, err := agentctx.GetChatIDByMessages(cache, messages1)
	require.NoError(t, err)

	messages2 := []agentctx.Message{
		{Role: agentctx.RoleUser, Content: "What's the weather in Tokyo?"},
		{
			Role:    agentctx.RoleAssistant,
			Content: nil,
			ToolCalls: []agentctx.ToolCall{
				{
					ID:   "call_123",
					Type: agentctx.ToolTypeFunction,
					Function: agentctx.Function{
						Name:      "get_weather",
						Arguments: `{"location":"Tokyo"}`,
					},
				},
			},
		},
		{Role: agentctx.RoleUser, Content: "How about tomorrow?"},
	}

	chatID2, err := agentctx.GetChatIDByMessages(cache, messages2)
	require.NoError(t, err)

	assert.Equal(t, chatID1, chatID2, "Messages with tool calls should maintain same chat ID")
}

func TestCacheChatID_EmptyMessages(t *testing.T) {
	testprepare.PrepareSandbox(t)

	cache := getTestCache(t)

	err := agentctx.CacheChatID(cache, []agentctx.Message{}, "chat_123")
	assert.Error(t, err)
}

func TestCacheChatID_EmptyChatID(t *testing.T) {
	testprepare.PrepareSandbox(t)

	cache := getTestCache(t)

	messages := []agentctx.Message{
		{Role: agentctx.RoleUser, Content: "Hello"},
	}

	err := agentctx.CacheChatID(cache, messages, "")
	assert.Error(t, err)
}

func TestGetChatIDByMessages_EmptyMessages(t *testing.T) {
	testprepare.PrepareSandbox(t)

	cache := getTestCache(t)

	_, err := agentctx.GetChatIDByMessages(cache, []agentctx.Message{})
	assert.Error(t, err)
}

func TestGenChatID(t *testing.T) {
	testprepare.PrepareSandbox(t)

	id1 := agentctx.GenChatID()
	assert.NotEmpty(t, id1)
	assert.GreaterOrEqual(t, len(id1), 10)
}
