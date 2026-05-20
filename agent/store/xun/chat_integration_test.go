//go:build integration

package xun_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/yaoapp/xun/dbal/query"
	"github.com/yaoapp/yao/agent/store/types"
	"github.com/yaoapp/yao/agent/store/xun"
	"github.com/yaoapp/yao/unit-test/agent/testprepare"
)

func TestCreateChat(t *testing.T) {
	testprepare.PrepareSandbox(t)

	store, err := xun.NewXun(types.Setting{Connector: "default"})
	require.NoError(t, err)

	t.Run("CreateNewChat", func(t *testing.T) {
		chat := &types.Chat{
			AssistantID: "test_assistant",
			Title:       "Test Chat",
			Status:      "active",
			Share:       "private",
		}

		err := store.CreateChat(chat)
		require.NoError(t, err)
		assert.NotEmpty(t, chat.ChatID)

		t.Cleanup(func() { store.DeleteChat(chat.ChatID) })
	})

	t.Run("CreateChatWithAllFields", func(t *testing.T) {
		now := time.Now()
		chat := &types.Chat{
			AssistantID:   "test_assistant",
			LastConnector: "openai",
			Title:         "Full Chat",
			LastMode:      "task",
			Status:        "active",
			Public:        true,
			Share:         "team",
			Sort:          100,
			LastMessageAt: &now,
			Metadata: map[string]interface{}{
				"source": "test",
				"tags":   []string{"test", "chat"},
			},
		}

		err := store.CreateChat(chat)
		require.NoError(t, err)
		t.Cleanup(func() { store.DeleteChat(chat.ChatID) })

		retrieved, err := store.GetChat(chat.ChatID)
		require.NoError(t, err)

		assert.Equal(t, "Full Chat", retrieved.Title)
		assert.Equal(t, "openai", retrieved.LastConnector)
		assert.Equal(t, "task", retrieved.LastMode)
		assert.True(t, retrieved.Public)
		assert.Equal(t, "team", retrieved.Share)
		assert.Equal(t, 100, retrieved.Sort)
		assert.NotNil(t, retrieved.Metadata)
	})

	t.Run("CreateChatWithCustomID", func(t *testing.T) {
		customID := fmt.Sprintf("custom_chat_%d", time.Now().UnixNano())
		chat := &types.Chat{
			ChatID:      customID,
			AssistantID: "test_assistant",
		}

		err := store.CreateChat(chat)
		require.NoError(t, err)
		t.Cleanup(func() { store.DeleteChat(chat.ChatID) })

		assert.Equal(t, customID, chat.ChatID)
	})

	t.Run("CreateDuplicateChatFails", func(t *testing.T) {
		chat := &types.Chat{AssistantID: "test_assistant"}
		err := store.CreateChat(chat)
		require.NoError(t, err)
		t.Cleanup(func() { store.DeleteChat(chat.ChatID) })

		duplicate := &types.Chat{ChatID: chat.ChatID, AssistantID: "test_assistant"}
		err = store.CreateChat(duplicate)
		assert.Error(t, err)
	})

	t.Run("CreateChatWithoutAssistantIDFails", func(t *testing.T) {
		chat := &types.Chat{Title: "No Assistant"}
		err := store.CreateChat(chat)
		assert.Error(t, err)
	})

	t.Run("CreateNilChatFails", func(t *testing.T) {
		err := store.CreateChat(nil)
		assert.Error(t, err)
	})

	t.Run("CreateChatWithDefaults", func(t *testing.T) {
		chat := &types.Chat{AssistantID: "test_assistant"}
		err := store.CreateChat(chat)
		require.NoError(t, err)
		t.Cleanup(func() { store.DeleteChat(chat.ChatID) })

		retrieved, err := store.GetChat(chat.ChatID)
		require.NoError(t, err)

		assert.Empty(t, retrieved.LastMode)
		assert.Equal(t, "active", retrieved.Status)
		assert.Equal(t, "private", retrieved.Share)
	})
}

func TestGetChat(t *testing.T) {
	testprepare.PrepareSandbox(t)

	store, err := xun.NewXun(types.Setting{Connector: "default"})
	require.NoError(t, err)

	t.Run("GetExistingChat", func(t *testing.T) {
		chat := &types.Chat{AssistantID: "test_assistant", Title: "Get Test Chat"}
		err := store.CreateChat(chat)
		require.NoError(t, err)
		t.Cleanup(func() { store.DeleteChat(chat.ChatID) })

		retrieved, err := store.GetChat(chat.ChatID)
		require.NoError(t, err)

		assert.Equal(t, chat.ChatID, retrieved.ChatID)
		assert.Equal(t, "Get Test Chat", retrieved.Title)
	})

	t.Run("GetNonExistentChat", func(t *testing.T) {
		_, err := store.GetChat("nonexistent_chat_id")
		assert.Error(t, err)
	})

	t.Run("GetChatWithEmptyID", func(t *testing.T) {
		_, err := store.GetChat("")
		assert.Error(t, err)
	})

	t.Run("GetDeletedChatFails", func(t *testing.T) {
		chat := &types.Chat{AssistantID: "test_assistant"}
		err := store.CreateChat(chat)
		require.NoError(t, err)

		err = store.DeleteChat(chat.ChatID)
		require.NoError(t, err)

		_, err = store.GetChat(chat.ChatID)
		assert.Error(t, err)
	})
}

func TestUpdateChat(t *testing.T) {
	testprepare.PrepareSandbox(t)

	store, err := xun.NewXun(types.Setting{Connector: "default"})
	require.NoError(t, err)

	t.Run("UpdateTitle", func(t *testing.T) {
		chat := &types.Chat{AssistantID: "test_assistant", Title: "Original Title"}
		err := store.CreateChat(chat)
		require.NoError(t, err)
		t.Cleanup(func() { store.DeleteChat(chat.ChatID) })

		err = store.UpdateChat(chat.ChatID, map[string]interface{}{"title": "Updated Title"})
		require.NoError(t, err)

		retrieved, err := store.GetChat(chat.ChatID)
		require.NoError(t, err)
		assert.Equal(t, "Updated Title", retrieved.Title)
	})

	t.Run("UpdateLastConnector", func(t *testing.T) {
		chat := &types.Chat{AssistantID: "test_assistant", LastConnector: "openai"}
		err := store.CreateChat(chat)
		require.NoError(t, err)
		t.Cleanup(func() { store.DeleteChat(chat.ChatID) })

		err = store.UpdateChat(chat.ChatID, map[string]interface{}{"last_connector": "anthropic"})
		require.NoError(t, err)

		retrieved, err := store.GetChat(chat.ChatID)
		require.NoError(t, err)
		assert.Equal(t, "anthropic", retrieved.LastConnector)
	})

	t.Run("UpdateLastConnectorAndLastMessageAt", func(t *testing.T) {
		chat := &types.Chat{AssistantID: "test_assistant", LastConnector: "openai"}
		err := store.CreateChat(chat)
		require.NoError(t, err)
		t.Cleanup(func() { store.DeleteChat(chat.ChatID) })

		now := time.Now()
		err = store.UpdateChat(chat.ChatID, map[string]interface{}{
			"last_message_at": now,
			"last_connector":  "claude",
		})
		require.NoError(t, err)

		retrieved, err := store.GetChat(chat.ChatID)
		require.NoError(t, err)
		assert.Equal(t, "claude", retrieved.LastConnector)
		assert.NotNil(t, retrieved.LastMessageAt)
	})

	t.Run("UpdateMultipleFields", func(t *testing.T) {
		chat := &types.Chat{AssistantID: "test_assistant", Title: "Original", Status: "active", Share: "private"}
		err := store.CreateChat(chat)
		require.NoError(t, err)
		t.Cleanup(func() { store.DeleteChat(chat.ChatID) })

		err = store.UpdateChat(chat.ChatID, map[string]interface{}{
			"title":  "Updated",
			"status": "archived",
			"share":  "team",
			"public": true,
			"sort":   50,
		})
		require.NoError(t, err)

		retrieved, err := store.GetChat(chat.ChatID)
		require.NoError(t, err)
		assert.Equal(t, "Updated", retrieved.Title)
		assert.Equal(t, "archived", retrieved.Status)
		assert.Equal(t, "team", retrieved.Share)
		assert.True(t, retrieved.Public)
		assert.Equal(t, 50, retrieved.Sort)
	})

	t.Run("UpdateMetadata", func(t *testing.T) {
		chat := &types.Chat{AssistantID: "test_assistant"}
		err := store.CreateChat(chat)
		require.NoError(t, err)
		t.Cleanup(func() { store.DeleteChat(chat.ChatID) })

		err = store.UpdateChat(chat.ChatID, map[string]interface{}{
			"metadata": map[string]interface{}{"key1": "value1", "key2": 123},
		})
		require.NoError(t, err)

		retrieved, err := store.GetChat(chat.ChatID)
		require.NoError(t, err)
		require.NotNil(t, retrieved.Metadata)
		assert.Equal(t, "value1", retrieved.Metadata["key1"])
	})

	t.Run("UpdateNonExistentChatFails", func(t *testing.T) {
		err := store.UpdateChat("nonexistent_chat", map[string]interface{}{"title": "Test"})
		assert.Error(t, err)
	})

	t.Run("UpdateWithEmptyIDFails", func(t *testing.T) {
		err := store.UpdateChat("", map[string]interface{}{"title": "Test"})
		assert.Error(t, err)
	})

	t.Run("UpdateWithEmptyFieldsFails", func(t *testing.T) {
		chat := &types.Chat{AssistantID: "test_assistant"}
		err := store.CreateChat(chat)
		require.NoError(t, err)
		t.Cleanup(func() { store.DeleteChat(chat.ChatID) })

		err = store.UpdateChat(chat.ChatID, map[string]interface{}{})
		assert.Error(t, err)
	})

	t.Run("UpdateSkipsSystemFields", func(t *testing.T) {
		chat := &types.Chat{AssistantID: "test_assistant"}
		err := store.CreateChat(chat)
		require.NoError(t, err)
		t.Cleanup(func() { store.DeleteChat(chat.ChatID) })

		originalID := chat.ChatID
		err = store.UpdateChat(chat.ChatID, map[string]interface{}{
			"chat_id": "new_id",
			"title":   "Valid Update",
		})
		require.NoError(t, err)

		retrieved, err := store.GetChat(originalID)
		require.NoError(t, err)
		assert.Equal(t, originalID, retrieved.ChatID)
		assert.Equal(t, "Valid Update", retrieved.Title)
	})
}

func TestDeleteChat(t *testing.T) {
	testprepare.PrepareSandbox(t)

	store, err := xun.NewXun(types.Setting{Connector: "default"})
	require.NoError(t, err)

	t.Run("DeleteExistingChat", func(t *testing.T) {
		chat := &types.Chat{AssistantID: "test_assistant"}
		err := store.CreateChat(chat)
		require.NoError(t, err)

		err = store.DeleteChat(chat.ChatID)
		require.NoError(t, err)

		_, err = store.GetChat(chat.ChatID)
		assert.Error(t, err)
	})

	t.Run("DeleteNonExistentChatFails", func(t *testing.T) {
		err := store.DeleteChat("nonexistent_chat")
		assert.Error(t, err)
	})

	t.Run("DeleteWithEmptyIDFails", func(t *testing.T) {
		err := store.DeleteChat("")
		assert.Error(t, err)
	})

	t.Run("DeleteAlreadyDeletedChatFails", func(t *testing.T) {
		chat := &types.Chat{AssistantID: "test_assistant"}
		err := store.CreateChat(chat)
		require.NoError(t, err)

		err = store.DeleteChat(chat.ChatID)
		require.NoError(t, err)

		err = store.DeleteChat(chat.ChatID)
		assert.Error(t, err)
	})
}

func TestListChats(t *testing.T) {
	testprepare.PrepareSandbox(t)

	store, err := xun.NewXun(types.Setting{Connector: "default"})
	require.NoError(t, err)

	listAssistantID := fmt.Sprintf("test_list_%d", time.Now().UnixNano())
	chatIDs := []string{}
	for i := 0; i < 5; i++ {
		chat := &types.Chat{
			AssistantID: listAssistantID,
			Title:       fmt.Sprintf("Chat %d", i),
			Status:      "active",
		}
		if i >= 3 {
			chat.Status = "archived"
		}
		err := store.CreateChat(chat)
		require.NoError(t, err)
		chatIDs = append(chatIDs, chat.ChatID)

	}
	t.Cleanup(func() {
		for _, id := range chatIDs {
			store.DeleteChat(id)
		}
	})

	t.Run("ListAllChats", func(t *testing.T) {
		result, err := store.ListChats(types.ChatFilter{AssistantID: listAssistantID, Page: 1, PageSize: 20})
		require.NoError(t, err)
		assert.Equal(t, 5, len(result.Data))
	})

	t.Run("ListChatsByStatus", func(t *testing.T) {
		result, err := store.ListChats(types.ChatFilter{AssistantID: listAssistantID, Status: "active", Page: 1, PageSize: 20})
		require.NoError(t, err)
		assert.Equal(t, 3, len(result.Data))
		for _, chat := range result.Data {
			assert.Equal(t, "active", chat.Status)
		}
	})

	t.Run("ListChatsByAssistant", func(t *testing.T) {
		result, err := store.ListChats(types.ChatFilter{AssistantID: listAssistantID, Page: 1, PageSize: 20})
		require.NoError(t, err)
		for _, chat := range result.Data {
			assert.Equal(t, listAssistantID, chat.AssistantID)
		}
	})

	t.Run("ListChatsByKeywords", func(t *testing.T) {
		result, err := store.ListChats(types.ChatFilter{AssistantID: listAssistantID, Keywords: "Chat 1", Page: 1, PageSize: 20})
		require.NoError(t, err)
		require.GreaterOrEqual(t, len(result.Data), 1)
		assert.Equal(t, "Chat 1", result.Data[0].Title)
	})

	t.Run("ListChatsPagination", func(t *testing.T) {
		result1, err := store.ListChats(types.ChatFilter{AssistantID: listAssistantID, Page: 1, PageSize: 2})
		require.NoError(t, err)
		assert.Equal(t, 2, len(result1.Data))
		assert.Equal(t, 1, result1.Page)
		assert.Equal(t, 2, result1.PageSize)
		assert.Equal(t, 5, result1.Total)

		result2, err := store.ListChats(types.ChatFilter{AssistantID: listAssistantID, Page: 2, PageSize: 2})
		require.NoError(t, err)
		assert.Equal(t, 2, result2.Page)
		assert.Equal(t, 2, len(result2.Data))
	})

	t.Run("ListChatsWithGrouping", func(t *testing.T) {
		result, err := store.ListChats(types.ChatFilter{AssistantID: listAssistantID, GroupBy: "time", Page: 1, PageSize: 20})
		require.NoError(t, err)
		require.NotNil(t, result.Groups)
		for _, group := range result.Groups {
			assert.NotEmpty(t, group.Key)
			assert.NotEmpty(t, group.Label)
			assert.Equal(t, group.Count, len(group.Chats))
		}
	})

	t.Run("ListChatsWithTimeRange", func(t *testing.T) {
		// Use DB-stored time as reference to avoid timezone mismatch
		// (PG timestamp without timezone stores local time but reads back as UTC)
		refChat, err := store.GetChat(chatIDs[0])
		require.NoError(t, err)
		start := refChat.CreatedAt.Add(-24 * time.Hour)
		end := refChat.CreatedAt.Add(time.Minute)
		result, err := store.ListChats(types.ChatFilter{
			AssistantID: listAssistantID,
			StartTime:   &start,
			EndTime:     &end,
			TimeField:   "created_at",
			Page:        1,
			PageSize:    20,
		})
		require.NoError(t, err)
		assert.Equal(t, 5, len(result.Data))
	})

	t.Run("ListChatsWithSorting", func(t *testing.T) {
		resultAsc, err := store.ListChats(types.ChatFilter{AssistantID: listAssistantID, OrderBy: "created_at", Order: "asc", Page: 1, PageSize: 20})
		require.NoError(t, err)

		resultDesc, err := store.ListChats(types.ChatFilter{AssistantID: listAssistantID, OrderBy: "created_at", Order: "desc", Page: 1, PageSize: 20})
		require.NoError(t, err)

		require.Equal(t, 5, len(resultAsc.Data))
		require.Equal(t, 5, len(resultDesc.Data))

		// Verify asc and desc are mirror images of each other
		for i := 0; i < 5; i++ {
			assert.Equal(t, resultAsc.Data[i].ChatID, resultDesc.Data[4-i].ChatID,
				"asc[%d] should equal desc[%d]", i, 4-i)
		}
	})

	t.Run("ListChatsWithQueryFilter", func(t *testing.T) {
		result, err := store.ListChats(types.ChatFilter{
			Page:     1,
			PageSize: 20,
			QueryFilter: func(qb query.Query) {
				qb.Where("status", "active")
			},
		})
		require.NoError(t, err)

		for _, chat := range result.Data {
			assert.Equal(t, "active", chat.Status)
		}
	})
}

func TestChatCompleteWorkflow(t *testing.T) {
	testprepare.PrepareSandbox(t)

	store, err := xun.NewXun(types.Setting{Connector: "default"})
	require.NoError(t, err)

	t.Run("CompleteWorkflow", func(t *testing.T) {
		chat := &types.Chat{
			AssistantID: "workflow_assistant",
			Title:       "Workflow Test Chat",
			Status:      "active",
		}

		err := store.CreateChat(chat)
		require.NoError(t, err)
		t.Cleanup(func() { store.DeleteChat(chat.ChatID) })

		retrieved, err := store.GetChat(chat.ChatID)
		require.NoError(t, err)
		assert.Equal(t, "Workflow Test Chat", retrieved.Title)

		err = store.UpdateChat(chat.ChatID, map[string]interface{}{
			"title":  "Updated Workflow Chat",
			"status": "archived",
		})
		require.NoError(t, err)

		updated, err := store.GetChat(chat.ChatID)
		require.NoError(t, err)
		assert.Equal(t, "Updated Workflow Chat", updated.Title)
		assert.Equal(t, "archived", updated.Status)

		result, err := store.ListChats(types.ChatFilter{AssistantID: "workflow_assistant", Page: 1, PageSize: 20})
		require.NoError(t, err)

		found := false
		for _, c := range result.Data {
			if c.ChatID == chat.ChatID {
				found = true
				break
			}
		}
		assert.True(t, found)

		err = store.DeleteChat(chat.ChatID)
		require.NoError(t, err)

		_, err = store.GetChat(chat.ChatID)
		assert.Error(t, err)
	})
}
