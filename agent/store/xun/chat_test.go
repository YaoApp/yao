package xun_test

import (
	"fmt"
	"testing"
	"time"

	goumodel "github.com/yaoapp/gou/model"
	"github.com/yaoapp/xun/dbal/query"
	"github.com/yaoapp/yao/agent/store/types"
	"github.com/yaoapp/yao/agent/store/xun"
	"github.com/yaoapp/yao/config"
	"github.com/yaoapp/yao/test"
)

// TestCreateChat tests creating chat sessions
func TestCreateChat(t *testing.T) {
	test.Prepare(t, config.Conf)
	defer test.Clean()

	store, err := xun.NewXun(types.Setting{
		Connector: "default",
	})
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}

	t.Run("CreateNewChat", func(t *testing.T) {
		chat := &types.Chat{
			AssistantID: "test_assistant",
			Title:       "Test Chat",
			Status:      "active",
			Share:       "private",
		}

		err := store.CreateChat(chat)
		if err != nil {
			t.Fatalf("Failed to create chat: %v", err)
		}

		if chat.ChatID == "" {
			t.Error("Expected chat_id to be generated")
		}

		t.Logf("Created chat with ID: %s", chat.ChatID)

		// Clean up
		_ = store.DeleteChat(chat.ChatID)
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
		if err != nil {
			t.Fatalf("Failed to create chat: %v", err)
		}

		// Retrieve and verify
		retrieved, err := store.GetChat(chat.ChatID)
		if err != nil {
			t.Fatalf("Failed to retrieve chat: %v", err)
		}

		if retrieved.Title != "Full Chat" {
			t.Errorf("Expected title 'Full Chat', got '%s'", retrieved.Title)
		}
		if retrieved.LastConnector != "openai" {
			t.Errorf("Expected last_connector 'openai', got '%s'", retrieved.LastConnector)
		}
		if retrieved.LastMode != "task" {
			t.Errorf("Expected last_mode 'task', got '%s'", retrieved.LastMode)
		}
		if !retrieved.Public {
			t.Error("Expected public to be true")
		}
		if retrieved.Share != "team" {
			t.Errorf("Expected share 'team', got '%s'", retrieved.Share)
		}
		if retrieved.Sort != 100 {
			t.Errorf("Expected sort 100, got %d", retrieved.Sort)
		}
		if retrieved.Metadata == nil {
			t.Error("Expected metadata to be set")
		}

		// Clean up
		_ = store.DeleteChat(chat.ChatID)
	})

	t.Run("CreateChatWithCustomID", func(t *testing.T) {
		customID := fmt.Sprintf("custom_chat_%d", time.Now().UnixNano())
		chat := &types.Chat{
			ChatID:      customID,
			AssistantID: "test_assistant",
		}

		err := store.CreateChat(chat)
		if err != nil {
			t.Fatalf("Failed to create chat: %v", err)
		}

		if chat.ChatID != customID {
			t.Errorf("Expected chat_id '%s', got '%s'", customID, chat.ChatID)
		}

		// Clean up
		_ = store.DeleteChat(chat.ChatID)
	})

	t.Run("CreateDuplicateChatFails", func(t *testing.T) {
		chat := &types.Chat{
			AssistantID: "test_assistant",
		}

		err := store.CreateChat(chat)
		if err != nil {
			t.Fatalf("Failed to create first chat: %v", err)
		}

		// Try to create with same ID
		duplicateChat := &types.Chat{
			ChatID:      chat.ChatID,
			AssistantID: "test_assistant",
		}

		err = store.CreateChat(duplicateChat)
		if err == nil {
			t.Error("Expected error when creating duplicate chat")
		}

		// Clean up
		_ = store.DeleteChat(chat.ChatID)
	})

	t.Run("CreateChatWithoutAssistantIDFails", func(t *testing.T) {
		chat := &types.Chat{
			Title: "No Assistant",
		}

		err := store.CreateChat(chat)
		if err == nil {
			t.Error("Expected error when creating chat without assistant_id")
		}
	})

	t.Run("CreateNilChatFails", func(t *testing.T) {
		err := store.CreateChat(nil)
		if err == nil {
			t.Error("Expected error when creating nil chat")
		}
	})

	t.Run("CreateChatWithDefaults", func(t *testing.T) {
		chat := &types.Chat{
			AssistantID: "test_assistant",
		}

		err := store.CreateChat(chat)
		if err != nil {
			t.Fatalf("Failed to create chat: %v", err)
		}

		// Retrieve and verify defaults
		retrieved, err := store.GetChat(chat.ChatID)
		if err != nil {
			t.Fatalf("Failed to retrieve chat: %v", err)
		}

		// last_mode is nullable, so it should be empty by default
		if retrieved.LastMode != "" {
			t.Errorf("Expected default last_mode to be empty, got '%s'", retrieved.LastMode)
		}
		if retrieved.Status != "active" {
			t.Errorf("Expected default status 'active', got '%s'", retrieved.Status)
		}
		if retrieved.Share != "private" {
			t.Errorf("Expected default share 'private', got '%s'", retrieved.Share)
		}

		// Clean up
		_ = store.DeleteChat(chat.ChatID)
	})
}

// TestGetChat tests retrieving chat sessions
func TestGetChat(t *testing.T) {
	test.Prepare(t, config.Conf)
	defer test.Clean()

	store, err := xun.NewXun(types.Setting{
		Connector: "default",
	})
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}

	t.Run("GetExistingChat", func(t *testing.T) {
		// Create chat first
		chat := &types.Chat{
			AssistantID: "test_assistant",
			Title:       "Get Test Chat",
		}
		err := store.CreateChat(chat)
		if err != nil {
			t.Fatalf("Failed to create chat: %v", err)
		}

		// Get it
		retrieved, err := store.GetChat(chat.ChatID)
		if err != nil {
			t.Fatalf("Failed to get chat: %v", err)
		}

		if retrieved.ChatID != chat.ChatID {
			t.Errorf("Expected chat_id '%s', got '%s'", chat.ChatID, retrieved.ChatID)
		}
		if retrieved.Title != "Get Test Chat" {
			t.Errorf("Expected title 'Get Test Chat', got '%s'", retrieved.Title)
		}

		// Clean up
		_ = store.DeleteChat(chat.ChatID)
	})

	t.Run("GetNonExistentChat", func(t *testing.T) {
		_, err := store.GetChat("nonexistent_chat_id")
		if err == nil {
			t.Error("Expected error when getting non-existent chat")
		}
	})

	t.Run("GetChatWithEmptyID", func(t *testing.T) {
		_, err := store.GetChat("")
		if err == nil {
			t.Error("Expected error when getting chat with empty ID")
		}
	})

	t.Run("GetDeletedChatFails", func(t *testing.T) {
		// Create and delete chat
		chat := &types.Chat{
			AssistantID: "test_assistant",
		}
		err := store.CreateChat(chat)
		if err != nil {
			t.Fatalf("Failed to create chat: %v", err)
		}

		err = store.DeleteChat(chat.ChatID)
		if err != nil {
			t.Fatalf("Failed to delete chat: %v", err)
		}

		// Try to get deleted chat
		_, err = store.GetChat(chat.ChatID)
		if err == nil {
			t.Error("Expected error when getting deleted chat")
		}
	})
}

// TestUpdateChat tests updating chat sessions
func TestUpdateChat(t *testing.T) {
	test.Prepare(t, config.Conf)
	defer test.Clean()

	store, err := xun.NewXun(types.Setting{
		Connector: "default",
	})
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}

	t.Run("UpdateTitle", func(t *testing.T) {
		chat := &types.Chat{
			AssistantID: "test_assistant",
			Title:       "Original Title",
		}
		err := store.CreateChat(chat)
		if err != nil {
			t.Fatalf("Failed to create chat: %v", err)
		}

		err = store.UpdateChat(chat.ChatID, map[string]interface{}{
			"title": "Updated Title",
		})
		if err != nil {
			t.Fatalf("Failed to update chat: %v", err)
		}

		retrieved, err := store.GetChat(chat.ChatID)
		if err != nil {
			t.Fatalf("Failed to retrieve chat: %v", err)
		}

		if retrieved.Title != "Updated Title" {
			t.Errorf("Expected title 'Updated Title', got '%s'", retrieved.Title)
		}

		// Clean up
		_ = store.DeleteChat(chat.ChatID)
	})

	t.Run("UpdateLastConnector", func(t *testing.T) {
		chat := &types.Chat{
			AssistantID:   "test_assistant",
			LastConnector: "openai",
		}
		err := store.CreateChat(chat)
		if err != nil {
			t.Fatalf("Failed to create chat: %v", err)
		}

		// Verify initial connector
		retrieved, err := store.GetChat(chat.ChatID)
		if err != nil {
			t.Fatalf("Failed to retrieve chat: %v", err)
		}
		if retrieved.LastConnector != "openai" {
			t.Errorf("Expected last_connector 'openai', got '%s'", retrieved.LastConnector)
		}

		// Update to different connector (simulating user switching connector)
		err = store.UpdateChat(chat.ChatID, map[string]interface{}{
			"last_connector": "anthropic",
		})
		if err != nil {
			t.Fatalf("Failed to update chat: %v", err)
		}

		// Verify updated connector
		retrieved, err = store.GetChat(chat.ChatID)
		if err != nil {
			t.Fatalf("Failed to retrieve chat: %v", err)
		}
		if retrieved.LastConnector != "anthropic" {
			t.Errorf("Expected last_connector 'anthropic', got '%s'", retrieved.LastConnector)
		}

		// Clean up
		_ = store.DeleteChat(chat.ChatID)
	})

	t.Run("UpdateLastConnectorAndLastMessageAt", func(t *testing.T) {
		// This simulates what FlushBuffer does
		chat := &types.Chat{
			AssistantID:   "test_assistant",
			LastConnector: "openai",
		}
		err := store.CreateChat(chat)
		if err != nil {
			t.Fatalf("Failed to create chat: %v", err)
		}

		// Update both fields together (like FlushBuffer does)
		now := time.Now()
		err = store.UpdateChat(chat.ChatID, map[string]interface{}{
			"last_message_at": now,
			"last_connector":  "claude",
		})
		if err != nil {
			t.Fatalf("Failed to update chat: %v", err)
		}

		retrieved, err := store.GetChat(chat.ChatID)
		if err != nil {
			t.Fatalf("Failed to retrieve chat: %v", err)
		}

		if retrieved.LastConnector != "claude" {
			t.Errorf("Expected last_connector 'claude', got '%s'", retrieved.LastConnector)
		}
		if retrieved.LastMessageAt == nil {
			t.Error("Expected last_message_at to be set")
		}

		// Clean up
		_ = store.DeleteChat(chat.ChatID)
	})

	t.Run("UpdateMultipleFields", func(t *testing.T) {
		chat := &types.Chat{
			AssistantID: "test_assistant",
			Title:       "Original",
			Status:      "active",
			Share:       "private",
		}
		err := store.CreateChat(chat)
		if err != nil {
			t.Fatalf("Failed to create chat: %v", err)
		}

		err = store.UpdateChat(chat.ChatID, map[string]interface{}{
			"title":  "Updated",
			"status": "archived",
			"share":  "team",
			"public": true,
			"sort":   50,
		})
		if err != nil {
			t.Fatalf("Failed to update chat: %v", err)
		}

		retrieved, err := store.GetChat(chat.ChatID)
		if err != nil {
			t.Fatalf("Failed to retrieve chat: %v", err)
		}

		if retrieved.Title != "Updated" {
			t.Errorf("Expected title 'Updated', got '%s'", retrieved.Title)
		}
		if retrieved.Status != "archived" {
			t.Errorf("Expected status 'archived', got '%s'", retrieved.Status)
		}
		if retrieved.Share != "team" {
			t.Errorf("Expected share 'team', got '%s'", retrieved.Share)
		}
		if !retrieved.Public {
			t.Error("Expected public to be true")
		}
		if retrieved.Sort != 50 {
			t.Errorf("Expected sort 50, got %d", retrieved.Sort)
		}

		// Clean up
		_ = store.DeleteChat(chat.ChatID)
	})

	t.Run("UpdateMetadata", func(t *testing.T) {
		chat := &types.Chat{
			AssistantID: "test_assistant",
		}
		err := store.CreateChat(chat)
		if err != nil {
			t.Fatalf("Failed to create chat: %v", err)
		}

		err = store.UpdateChat(chat.ChatID, map[string]interface{}{
			"metadata": map[string]interface{}{
				"key1": "value1",
				"key2": 123,
			},
		})
		if err != nil {
			t.Fatalf("Failed to update metadata: %v", err)
		}

		retrieved, err := store.GetChat(chat.ChatID)
		if err != nil {
			t.Fatalf("Failed to retrieve chat: %v", err)
		}

		if retrieved.Metadata == nil {
			t.Fatal("Expected metadata to be set")
		}
		if retrieved.Metadata["key1"] != "value1" {
			t.Errorf("Expected metadata key1 'value1', got '%v'", retrieved.Metadata["key1"])
		}

		// Clean up
		_ = store.DeleteChat(chat.ChatID)
	})

	t.Run("UpdateNonExistentChatFails", func(t *testing.T) {
		err := store.UpdateChat("nonexistent_chat", map[string]interface{}{
			"title": "Test",
		})
		if err == nil {
			t.Error("Expected error when updating non-existent chat")
		}
	})

	t.Run("UpdateWithEmptyIDFails", func(t *testing.T) {
		err := store.UpdateChat("", map[string]interface{}{
			"title": "Test",
		})
		if err == nil {
			t.Error("Expected error when updating with empty ID")
		}
	})

	t.Run("UpdateWithEmptyFieldsFails", func(t *testing.T) {
		chat := &types.Chat{
			AssistantID: "test_assistant",
		}
		err := store.CreateChat(chat)
		if err != nil {
			t.Fatalf("Failed to create chat: %v", err)
		}

		err = store.UpdateChat(chat.ChatID, map[string]interface{}{})
		if err == nil {
			t.Error("Expected error when updating with empty fields")
		}

		// Clean up
		_ = store.DeleteChat(chat.ChatID)
	})

	t.Run("UpdateSkipsSystemFields", func(t *testing.T) {
		chat := &types.Chat{
			AssistantID: "test_assistant",
		}
		err := store.CreateChat(chat)
		if err != nil {
			t.Fatalf("Failed to create chat: %v", err)
		}

		originalID := chat.ChatID

		// Try to update system fields
		err = store.UpdateChat(chat.ChatID, map[string]interface{}{
			"chat_id": "new_id",
			"title":   "Valid Update",
		})
		if err != nil {
			t.Fatalf("Failed to update chat: %v", err)
		}

		// Verify chat_id unchanged
		retrieved, err := store.GetChat(originalID)
		if err != nil {
			t.Fatalf("Failed to retrieve chat: %v", err)
		}

		if retrieved.ChatID != originalID {
			t.Errorf("Expected chat_id to remain '%s', got '%s'", originalID, retrieved.ChatID)
		}
		if retrieved.Title != "Valid Update" {
			t.Errorf("Expected title 'Valid Update', got '%s'", retrieved.Title)
		}

		// Clean up
		_ = store.DeleteChat(chat.ChatID)
	})
}

// TestDeleteChat tests deleting chat sessions
func TestDeleteChat(t *testing.T) {
	test.Prepare(t, config.Conf)
	defer test.Clean()

	store, err := xun.NewXun(types.Setting{
		Connector: "default",
	})
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}

	t.Run("DeleteExistingChat", func(t *testing.T) {
		chat := &types.Chat{
			AssistantID: "test_assistant",
		}
		err := store.CreateChat(chat)
		if err != nil {
			t.Fatalf("Failed to create chat: %v", err)
		}

		err = store.DeleteChat(chat.ChatID)
		if err != nil {
			t.Fatalf("Failed to delete chat: %v", err)
		}

		// Verify deleted
		_, err = store.GetChat(chat.ChatID)
		if err == nil {
			t.Error("Expected error when getting deleted chat")
		}
	})

	t.Run("DeleteNonExistentChatFails", func(t *testing.T) {
		err := store.DeleteChat("nonexistent_chat")
		if err == nil {
			t.Error("Expected error when deleting non-existent chat")
		}
	})

	t.Run("DeleteWithEmptyIDFails", func(t *testing.T) {
		err := store.DeleteChat("")
		if err == nil {
			t.Error("Expected error when deleting with empty ID")
		}
	})

	t.Run("DeleteAlreadyDeletedChatFails", func(t *testing.T) {
		chat := &types.Chat{
			AssistantID: "test_assistant",
		}
		err := store.CreateChat(chat)
		if err != nil {
			t.Fatalf("Failed to create chat: %v", err)
		}

		// Delete first time
		err = store.DeleteChat(chat.ChatID)
		if err != nil {
			t.Fatalf("Failed to delete chat: %v", err)
		}

		// Try to delete again
		err = store.DeleteChat(chat.ChatID)
		if err == nil {
			t.Error("Expected error when deleting already deleted chat")
		}
	})
}

// TestListChats tests listing chat sessions
func TestListChats(t *testing.T) {
	test.Prepare(t, config.Conf)
	defer test.Clean()

	store, err := xun.NewXun(types.Setting{
		Connector: "default",
	})
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}

	// Create test chats
	chatIDs := []string{}
	for i := 0; i < 5; i++ {
		chat := &types.Chat{
			AssistantID: "test_assistant",
			Title:       fmt.Sprintf("Chat %d", i),
			Status:      "active",
		}
		if i >= 3 {
			chat.Status = "archived"
		}
		err := store.CreateChat(chat)
		if err != nil {
			t.Fatalf("Failed to create chat: %v", err)
		}
		chatIDs = append(chatIDs, chat.ChatID)

		// Add small delay to ensure different timestamps
		time.Sleep(10 * time.Millisecond)
	}

	// Clean up at the end
	defer func() {
		for _, id := range chatIDs {
			_ = store.DeleteChat(id)
		}
	}()

	t.Run("ListAllChats", func(t *testing.T) {
		result, err := store.ListChats(types.ChatFilter{
			Page:     1,
			PageSize: 20,
		})
		if err != nil {
			t.Fatalf("Failed to list chats: %v", err)
		}

		if len(result.Data) < 5 {
			t.Errorf("Expected at least 5 chats, got %d", len(result.Data))
		}
	})

	t.Run("ListChatsByStatus", func(t *testing.T) {
		result, err := store.ListChats(types.ChatFilter{
			Status:   "active",
			Page:     1,
			PageSize: 20,
		})
		if err != nil {
			t.Fatalf("Failed to list chats: %v", err)
		}

		for _, chat := range result.Data {
			if chat.Status != "active" {
				t.Errorf("Expected status 'active', got '%s'", chat.Status)
			}
		}
	})

	t.Run("ListChatsByAssistant", func(t *testing.T) {
		result, err := store.ListChats(types.ChatFilter{
			AssistantID: "test_assistant",
			Page:        1,
			PageSize:    20,
		})
		if err != nil {
			t.Fatalf("Failed to list chats: %v", err)
		}

		for _, chat := range result.Data {
			if chat.AssistantID != "test_assistant" {
				t.Errorf("Expected assistant_id 'test_assistant', got '%s'", chat.AssistantID)
			}
		}
	})

	t.Run("ListChatsByKeywords", func(t *testing.T) {
		result, err := store.ListChats(types.ChatFilter{
			Keywords: "Chat 1",
			Page:     1,
			PageSize: 20,
		})
		if err != nil {
			t.Fatalf("Failed to list chats: %v", err)
		}

		found := false
		for _, chat := range result.Data {
			if chat.Title == "Chat 1" {
				found = true
				break
			}
		}
		if !found {
			t.Error("Expected to find chat with title 'Chat 1'")
		}
	})

	t.Run("ListChatsPagination", func(t *testing.T) {
		// First page
		result1, err := store.ListChats(types.ChatFilter{
			Page:     1,
			PageSize: 2,
		})
		if err != nil {
			t.Fatalf("Failed to list first page: %v", err)
		}

		if len(result1.Data) > 2 {
			t.Errorf("Expected max 2 chats, got %d", len(result1.Data))
		}
		if result1.Page != 1 {
			t.Errorf("Expected page 1, got %d", result1.Page)
		}
		if result1.PageSize != 2 {
			t.Errorf("Expected pagesize 2, got %d", result1.PageSize)
		}

		// Second page
		if result1.Total > 2 {
			result2, err := store.ListChats(types.ChatFilter{
				Page:     2,
				PageSize: 2,
			})
			if err != nil {
				t.Fatalf("Failed to list second page: %v", err)
			}
			if result2.Page != 2 {
				t.Errorf("Expected page 2, got %d", result2.Page)
			}
		}
	})

	t.Run("ListChatsWithGrouping", func(t *testing.T) {
		result, err := store.ListChats(types.ChatFilter{
			GroupBy:  "time",
			Page:     1,
			PageSize: 20,
		})
		if err != nil {
			t.Fatalf("Failed to list chats with grouping: %v", err)
		}

		// Should have groups when GroupBy is "time"
		if result.Groups == nil {
			t.Error("Expected groups to be set when GroupBy='time'")
		}

		// Verify group structure
		for _, group := range result.Groups {
			if group.Key == "" {
				t.Error("Expected group key to be set")
			}
			if group.Label == "" {
				t.Error("Expected group label to be set")
			}
			if group.Count != len(group.Chats) {
				t.Errorf("Expected count %d to match chats length %d", group.Count, len(group.Chats))
			}
		}
	})

	t.Run("ListChatsWithTimeRange", func(t *testing.T) {
		now := time.Now()
		yesterday := now.AddDate(0, 0, -1)

		result, err := store.ListChats(types.ChatFilter{
			StartTime: &yesterday,
			EndTime:   &now,
			TimeField: "created_at",
			Page:      1,
			PageSize:  20,
		})
		if err != nil {
			t.Fatalf("Failed to list chats with time range: %v", err)
		}

		// Should return chats created within the time range
		t.Logf("Found %d chats in time range", len(result.Data))
	})

	t.Run("ListChatsWithSorting", func(t *testing.T) {
		// Ascending order
		resultAsc, err := store.ListChats(types.ChatFilter{
			OrderBy:  "created_at",
			Order:    "asc",
			Page:     1,
			PageSize: 20,
		})
		if err != nil {
			t.Fatalf("Failed to list chats ascending: %v", err)
		}

		// Descending order
		resultDesc, err := store.ListChats(types.ChatFilter{
			OrderBy:  "created_at",
			Order:    "desc",
			Page:     1,
			PageSize: 20,
		})
		if err != nil {
			t.Fatalf("Failed to list chats descending: %v", err)
		}

		// Verify different order
		if len(resultAsc.Data) > 1 && len(resultDesc.Data) > 1 {
			if resultAsc.Data[0].ChatID == resultDesc.Data[0].ChatID {
				// This is fine if there's only one chat, but otherwise order should differ
				if len(resultAsc.Data) > 1 {
					t.Logf("First chat in asc: %s, first in desc: %s", resultAsc.Data[0].ChatID, resultDesc.Data[0].ChatID)
				}
			}
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
		if err != nil {
			t.Fatalf("Failed to list chats with query filter: %v", err)
		}

		for _, chat := range result.Data {
			if chat.Status != "active" {
				t.Errorf("Expected status 'active', got '%s'", chat.Status)
			}
		}
	})
}

// TestListChatsByUserAndTeam tests filtering chats by UserID and TeamID
func TestListChatsByUserAndTeam(t *testing.T) {
	test.Prepare(t, config.Conf)
	defer test.Clean()

	store, err := xun.NewXun(types.Setting{
		Connector: "default",
	})
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}

	// Create chats with different user/team combinations
	// Note: __yao_created_by and __yao_team_id are managed by Yao's permission system
	// For testing, we'll create chats and then update these fields directly via raw query

	chat1 := &types.Chat{AssistantID: "test_assistant", Title: "User1 Team1 Chat"}
	chat2 := &types.Chat{AssistantID: "test_assistant", Title: "User1 Team2 Chat"}
	chat3 := &types.Chat{AssistantID: "test_assistant", Title: "User2 Team1 Chat"}
	chat4 := &types.Chat{AssistantID: "test_assistant", Title: "User2 Team2 Chat"}

	for _, chat := range []*types.Chat{chat1, chat2, chat3, chat4} {
		err := store.CreateChat(chat)
		if err != nil {
			t.Fatalf("Failed to create chat: %v", err)
		}
	}
	defer func() {
		store.DeleteChat(chat1.ChatID)
		store.DeleteChat(chat2.ChatID)
		store.DeleteChat(chat3.ChatID)
		store.DeleteChat(chat4.ChatID)
	}()

	// Update permission fields directly for testing
	// In production, these would be set by Yao's permission middleware
	updatePermissionFields := func(chatID, userID, teamID string) error {
		// Use Yao model to update permission fields
		m := goumodel.Select("__yao.agent.chat")
		if m == nil {
			return fmt.Errorf("model __yao.agent.chat not found")
		}
		_, err := m.UpdateWhere(
			goumodel.QueryParam{Wheres: []goumodel.QueryWhere{{Column: "chat_id", Value: chatID}}},
			map[string]interface{}{
				"__yao_created_by": userID,
				"__yao_team_id":    teamID,
			},
		)
		return err
	}

	// Set up permission fields
	updatePermissionFields(chat1.ChatID, "user1", "team1")
	updatePermissionFields(chat2.ChatID, "user1", "team2")
	updatePermissionFields(chat3.ChatID, "user2", "team1")
	updatePermissionFields(chat4.ChatID, "user2", "team2")

	t.Run("FilterByUserID", func(t *testing.T) {
		result, err := store.ListChats(types.ChatFilter{
			UserID:   "user1",
			Page:     1,
			PageSize: 20,
		})
		if err != nil {
			t.Fatalf("Failed to list chats by user: %v", err)
		}

		if len(result.Data) != 2 {
			t.Errorf("Expected 2 chats for user1, got %d", len(result.Data))
		}

		// Verify all returned chats belong to user1
		for _, chat := range result.Data {
			if chat.Title != "User1 Team1 Chat" && chat.Title != "User1 Team2 Chat" {
				t.Errorf("Unexpected chat title: %s", chat.Title)
			}
		}
	})

	t.Run("FilterByTeamID", func(t *testing.T) {
		result, err := store.ListChats(types.ChatFilter{
			TeamID:   "team1",
			Page:     1,
			PageSize: 20,
		})
		if err != nil {
			t.Fatalf("Failed to list chats by team: %v", err)
		}

		if len(result.Data) != 2 {
			t.Errorf("Expected 2 chats for team1, got %d", len(result.Data))
		}

		// Verify all returned chats belong to team1
		for _, chat := range result.Data {
			if chat.Title != "User1 Team1 Chat" && chat.Title != "User2 Team1 Chat" {
				t.Errorf("Unexpected chat title: %s", chat.Title)
			}
		}
	})

	t.Run("FilterByUserIDAndTeamID", func(t *testing.T) {
		result, err := store.ListChats(types.ChatFilter{
			UserID:   "user1",
			TeamID:   "team1",
			Page:     1,
			PageSize: 20,
		})
		if err != nil {
			t.Fatalf("Failed to list chats by user and team: %v", err)
		}

		if len(result.Data) != 1 {
			t.Errorf("Expected 1 chat for user1+team1, got %d", len(result.Data))
		}

		if len(result.Data) > 0 && result.Data[0].Title != "User1 Team1 Chat" {
			t.Errorf("Expected 'User1 Team1 Chat', got '%s'", result.Data[0].Title)
		}
	})

	t.Run("FilterByUserIDWithOtherFilters", func(t *testing.T) {
		// Combine UserID with Status filter
		result, err := store.ListChats(types.ChatFilter{
			UserID:   "user1",
			Status:   "active",
			Page:     1,
			PageSize: 20,
		})
		if err != nil {
			t.Fatalf("Failed to list chats: %v", err)
		}

		// All user1's chats should be active (default status)
		if len(result.Data) != 2 {
			t.Errorf("Expected 2 active chats for user1, got %d", len(result.Data))
		}
	})

	t.Run("FilterByTeamIDWithQueryFilter", func(t *testing.T) {
		// Combine TeamID with custom QueryFilter
		result, err := store.ListChats(types.ChatFilter{
			TeamID:   "team2",
			Page:     1,
			PageSize: 20,
			QueryFilter: func(qb query.Query) {
				// Additional filter: only chats with "User1" in title
				qb.Where("title", "like", "%User1%")
			},
		})
		if err != nil {
			t.Fatalf("Failed to list chats: %v", err)
		}

		if len(result.Data) != 1 {
			t.Errorf("Expected 1 chat (User1 in team2), got %d", len(result.Data))
		}

		if len(result.Data) > 0 && result.Data[0].Title != "User1 Team2 Chat" {
			t.Errorf("Expected 'User1 Team2 Chat', got '%s'", result.Data[0].Title)
		}
	})

	t.Run("FilterByNonExistentUser", func(t *testing.T) {
		result, err := store.ListChats(types.ChatFilter{
			UserID:   "nonexistent_user",
			Page:     1,
			PageSize: 20,
		})
		if err != nil {
			t.Fatalf("Failed to list chats: %v", err)
		}

		if len(result.Data) != 0 {
			t.Errorf("Expected 0 chats for nonexistent user, got %d", len(result.Data))
		}
	})

	t.Run("FilterByNonExistentTeam", func(t *testing.T) {
		result, err := store.ListChats(types.ChatFilter{
			TeamID:   "nonexistent_team",
			Page:     1,
			PageSize: 20,
		})
		if err != nil {
			t.Fatalf("Failed to list chats: %v", err)
		}

		if len(result.Data) != 0 {
			t.Errorf("Expected 0 chats for nonexistent team, got %d", len(result.Data))
		}
	})

	t.Run("QueryFilterForOrCondition", func(t *testing.T) {
		// Use QueryFilter for complex OR condition:
		// Get chats where user is user1 OR team is team2
		result, err := store.ListChats(types.ChatFilter{
			Page:     1,
			PageSize: 20,
			QueryFilter: func(qb query.Query) {
				qb.Where(func(sub query.Query) {
					sub.Where("__yao_created_by", "user1").
						OrWhere("__yao_team_id", "team2")
				})
			},
		})
		if err != nil {
			t.Fatalf("Failed to list chats with OR condition: %v", err)
		}

		// Should return: user1+team1, user1+team2, user2+team2 = 3 chats
		if len(result.Data) != 3 {
			t.Errorf("Expected 3 chats (user1 OR team2), got %d", len(result.Data))
		}
	})
}

// TestChatCompleteWorkflow tests a complete chat workflow
func TestChatCompleteWorkflow(t *testing.T) {
	test.Prepare(t, config.Conf)
	defer test.Clean()

	store, err := xun.NewXun(types.Setting{
		Connector: "default",
	})
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}

	t.Run("CompleteWorkflow", func(t *testing.T) {
		// 1. Create chat
		chat := &types.Chat{
			AssistantID: "workflow_assistant",
			Title:       "Workflow Test Chat",
			Status:      "active",
		}

		err := store.CreateChat(chat)
		if err != nil {
			t.Fatalf("Failed to create chat: %v", err)
		}
		t.Logf("Created chat: %s", chat.ChatID)

		// 2. Get chat
		retrieved, err := store.GetChat(chat.ChatID)
		if err != nil {
			t.Fatalf("Failed to get chat: %v", err)
		}
		if retrieved.Title != "Workflow Test Chat" {
			t.Errorf("Expected title 'Workflow Test Chat', got '%s'", retrieved.Title)
		}

		// 3. Update chat
		err = store.UpdateChat(chat.ChatID, map[string]interface{}{
			"title":  "Updated Workflow Chat",
			"status": "archived",
		})
		if err != nil {
			t.Fatalf("Failed to update chat: %v", err)
		}

		// 4. Verify update
		updated, err := store.GetChat(chat.ChatID)
		if err != nil {
			t.Fatalf("Failed to get updated chat: %v", err)
		}
		if updated.Title != "Updated Workflow Chat" {
			t.Errorf("Expected title 'Updated Workflow Chat', got '%s'", updated.Title)
		}
		if updated.Status != "archived" {
			t.Errorf("Expected status 'archived', got '%s'", updated.Status)
		}

		// 5. List chats
		result, err := store.ListChats(types.ChatFilter{
			AssistantID: "workflow_assistant",
			Page:        1,
			PageSize:    20,
		})
		if err != nil {
			t.Fatalf("Failed to list chats: %v", err)
		}

		found := false
		for _, c := range result.Data {
			if c.ChatID == chat.ChatID {
				found = true
				break
			}
		}
		if !found {
			t.Error("Expected to find chat in list")
		}

		// 6. Delete chat
		err = store.DeleteChat(chat.ChatID)
		if err != nil {
			t.Fatalf("Failed to delete chat: %v", err)
		}

		// 7. Verify deletion
		_, err = store.GetChat(chat.ChatID)
		if err == nil {
			t.Error("Expected error when getting deleted chat")
		}

		t.Log("Complete workflow passed!")
	})
}
