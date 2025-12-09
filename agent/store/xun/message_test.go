package xun_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/yaoapp/yao/agent/store/types"
	"github.com/yaoapp/yao/agent/store/xun"
	"github.com/yaoapp/yao/config"
	"github.com/yaoapp/yao/test"
)

// TestSaveMessages tests batch saving messages
func TestSaveMessages(t *testing.T) {
	test.Prepare(t, config.Conf)
	defer test.Clean()

	store, err := xun.NewXun(types.Setting{
		Connector: "default",
	})
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}

	// Create a chat first
	chat := &types.Chat{
		AssistantID: "test_assistant",
		Title:       "Message Test Chat",
	}
	err = store.CreateChat(chat)
	if err != nil {
		t.Fatalf("Failed to create chat: %v", err)
	}
	defer store.DeleteChat(chat.ChatID)

	t.Run("SaveSingleMessage", func(t *testing.T) {
		messages := []*types.Message{
			{
				Role:     "user",
				Type:     "text",
				Props:    map[string]interface{}{"content": "Hello, world!"},
				Sequence: 1,
			},
		}

		err := store.SaveMessages(chat.ChatID, messages)
		if err != nil {
			t.Fatalf("Failed to save message: %v", err)
		}

		// Verify
		retrieved, err := store.GetMessages(chat.ChatID, types.MessageFilter{})
		if err != nil {
			t.Fatalf("Failed to get messages: %v", err)
		}

		if len(retrieved) < 1 {
			t.Fatal("Expected at least 1 message")
		}

		// Find the message we just saved
		var found *types.Message
		for _, msg := range retrieved {
			if msg.Sequence == 1 && msg.Type == "text" {
				found = msg
				break
			}
		}

		if found == nil {
			t.Fatal("Could not find saved message")
		}

		if found.Role != "user" {
			t.Errorf("Expected role 'user', got '%s'", found.Role)
		}
		if found.Props["content"] != "Hello, world!" {
			t.Errorf("Expected content 'Hello, world!', got '%v'", found.Props["content"])
		}
	})

	t.Run("SaveBatchMessages", func(t *testing.T) {
		// Create a new chat for this test
		batchChat := &types.Chat{
			AssistantID: "test_assistant",
			Title:       "Batch Message Test",
		}
		err := store.CreateChat(batchChat)
		if err != nil {
			t.Fatalf("Failed to create chat: %v", err)
		}
		defer store.DeleteChat(batchChat.ChatID)

		// Save multiple messages in one batch
		messages := []*types.Message{
			{
				Role:        "user",
				Type:        "user_input",
				Props:       map[string]interface{}{"content": "What's the weather?"},
				Sequence:    1,
				RequestID:   "req_001",
				AssistantID: "weather_assistant",
			},
			{
				Role:        "assistant",
				Type:        "loading",
				Props:       map[string]interface{}{"message": "Checking weather..."},
				Sequence:    2,
				RequestID:   "req_001",
				BlockID:     "B1",
				AssistantID: "weather_assistant",
			},
			{
				Role:        "assistant",
				Type:        "text",
				Props:       map[string]interface{}{"content": "The weather is sunny, 25°C."},
				Sequence:    3,
				RequestID:   "req_001",
				BlockID:     "B1",
				AssistantID: "weather_assistant",
			},
		}

		err = store.SaveMessages(batchChat.ChatID, messages)
		if err != nil {
			t.Fatalf("Failed to save batch messages: %v", err)
		}

		// Verify all messages saved
		retrieved, err := store.GetMessages(batchChat.ChatID, types.MessageFilter{})
		if err != nil {
			t.Fatalf("Failed to get messages: %v", err)
		}

		if len(retrieved) != 3 {
			t.Errorf("Expected 3 messages, got %d", len(retrieved))
		}

		// Verify order (should be by sequence)
		if len(retrieved) >= 3 {
			if retrieved[0].Sequence != 1 {
				t.Errorf("Expected first message sequence 1, got %d", retrieved[0].Sequence)
			}
			if retrieved[2].Sequence != 3 {
				t.Errorf("Expected last message sequence 3, got %d", retrieved[2].Sequence)
			}
		}

		t.Logf("Saved %d messages in single batch call", len(messages))
	})

	t.Run("SaveMessageWithAllFields", func(t *testing.T) {
		fullChat := &types.Chat{
			AssistantID: "test_assistant",
		}
		err := store.CreateChat(fullChat)
		if err != nil {
			t.Fatalf("Failed to create chat: %v", err)
		}
		defer store.DeleteChat(fullChat.ChatID)

		messages := []*types.Message{
			{
				Role:        "assistant",
				Type:        "tool_call",
				Props:       map[string]interface{}{"id": "call_123", "name": "get_weather", "arguments": `{"location":"SF"}`},
				Sequence:    1,
				RequestID:   "req_full",
				BlockID:     "B1",
				ThreadID:    "T1",
				AssistantID: "weather_assistant",
				Metadata:    map[string]interface{}{"tool_call_id": "call_123", "is_tool_result": false},
			},
		}

		err = store.SaveMessages(fullChat.ChatID, messages)
		if err != nil {
			t.Fatalf("Failed to save message: %v", err)
		}

		retrieved, err := store.GetMessages(fullChat.ChatID, types.MessageFilter{})
		if err != nil {
			t.Fatalf("Failed to get messages: %v", err)
		}

		if len(retrieved) != 1 {
			t.Fatalf("Expected 1 message, got %d", len(retrieved))
		}

		msg := retrieved[0]
		if msg.RequestID != "req_full" {
			t.Errorf("Expected request_id 'req_full', got '%s'", msg.RequestID)
		}
		if msg.BlockID != "B1" {
			t.Errorf("Expected block_id 'B1', got '%s'", msg.BlockID)
		}
		if msg.ThreadID != "T1" {
			t.Errorf("Expected thread_id 'T1', got '%s'", msg.ThreadID)
		}
		if msg.AssistantID != "weather_assistant" {
			t.Errorf("Expected assistant_id 'weather_assistant', got '%s'", msg.AssistantID)
		}
		if msg.Metadata == nil {
			t.Error("Expected metadata to be set")
		} else if msg.Metadata["tool_call_id"] != "call_123" {
			t.Errorf("Expected metadata tool_call_id 'call_123', got '%v'", msg.Metadata["tool_call_id"])
		}
	})

	t.Run("SaveMessageWithConnector", func(t *testing.T) {
		connChat := &types.Chat{
			AssistantID: "test_assistant",
		}
		err := store.CreateChat(connChat)
		if err != nil {
			t.Fatalf("Failed to create chat: %v", err)
		}
		defer store.DeleteChat(connChat.ChatID)

		// Save messages with different connectors
		messages := []*types.Message{
			{
				Role:        "user",
				Type:        "user_input",
				Props:       map[string]interface{}{"content": "Hello"},
				Sequence:    1,
				Connector:   "openai",
				AssistantID: "test_assistant",
			},
			{
				Role:        "assistant",
				Type:        "text",
				Props:       map[string]interface{}{"content": "Hi there!"},
				Sequence:    2,
				Connector:   "openai",
				AssistantID: "test_assistant",
			},
			{
				Role:        "user",
				Type:        "user_input",
				Props:       map[string]interface{}{"content": "Switch to Claude"},
				Sequence:    3,
				Connector:   "anthropic",
				AssistantID: "test_assistant",
			},
			{
				Role:        "assistant",
				Type:        "text",
				Props:       map[string]interface{}{"content": "Now using Claude!"},
				Sequence:    4,
				Connector:   "anthropic",
				AssistantID: "test_assistant",
			},
		}

		err = store.SaveMessages(connChat.ChatID, messages)
		if err != nil {
			t.Fatalf("Failed to save messages: %v", err)
		}

		// Retrieve and verify connectors
		retrieved, err := store.GetMessages(connChat.ChatID, types.MessageFilter{})
		if err != nil {
			t.Fatalf("Failed to get messages: %v", err)
		}

		if len(retrieved) != 4 {
			t.Fatalf("Expected 4 messages, got %d", len(retrieved))
		}

		// Verify each message has correct connector
		for _, msg := range retrieved {
			if msg.Sequence <= 2 && msg.Connector != "openai" {
				t.Errorf("Expected connector 'openai' for sequence %d, got '%s'", msg.Sequence, msg.Connector)
			}
			if msg.Sequence > 2 && msg.Connector != "anthropic" {
				t.Errorf("Expected connector 'anthropic' for sequence %d, got '%s'", msg.Sequence, msg.Connector)
			}
		}

		t.Logf("Successfully saved and retrieved messages with different connectors")
	})

	t.Run("SaveMessageWithEmptyConnector", func(t *testing.T) {
		emptyConnChat := &types.Chat{
			AssistantID: "test_assistant",
		}
		err := store.CreateChat(emptyConnChat)
		if err != nil {
			t.Fatalf("Failed to create chat: %v", err)
		}
		defer store.DeleteChat(emptyConnChat.ChatID)

		// Save message without connector
		messages := []*types.Message{
			{
				Role:     "user",
				Type:     "text",
				Props:    map[string]interface{}{"content": "No connector"},
				Sequence: 1,
				// Connector is empty
			},
		}

		err = store.SaveMessages(emptyConnChat.ChatID, messages)
		if err != nil {
			t.Fatalf("Failed to save message: %v", err)
		}

		retrieved, err := store.GetMessages(emptyConnChat.ChatID, types.MessageFilter{})
		if err != nil {
			t.Fatalf("Failed to get messages: %v", err)
		}

		if len(retrieved) != 1 {
			t.Fatalf("Expected 1 message, got %d", len(retrieved))
		}

		// Empty connector should be stored as empty string
		if retrieved[0].Connector != "" {
			t.Errorf("Expected empty connector, got '%s'", retrieved[0].Connector)
		}
	})

	t.Run("SaveEmptyMessages", func(t *testing.T) {
		err := store.SaveMessages(chat.ChatID, []*types.Message{})
		if err != nil {
			t.Errorf("Expected no error for empty messages, got: %v", err)
		}
	})

	t.Run("SaveMessagesWithoutChatID", func(t *testing.T) {
		messages := []*types.Message{{Role: "user", Type: "text", Props: map[string]interface{}{"content": "test"}}}
		err := store.SaveMessages("", messages)
		if err == nil {
			t.Error("Expected error when saving without chat_id")
		}
	})

	t.Run("SaveMessageWithoutRole", func(t *testing.T) {
		messages := []*types.Message{{Type: "text", Props: map[string]interface{}{"content": "test"}, Sequence: 1}}
		err := store.SaveMessages(chat.ChatID, messages)
		if err == nil {
			t.Error("Expected error when saving message without role")
		}
	})

	t.Run("SaveMessageWithoutType", func(t *testing.T) {
		messages := []*types.Message{{Role: "user", Props: map[string]interface{}{"content": "test"}, Sequence: 1}}
		err := store.SaveMessages(chat.ChatID, messages)
		if err == nil {
			t.Error("Expected error when saving message without type")
		}
	})

	t.Run("SaveMessageWithoutProps", func(t *testing.T) {
		messages := []*types.Message{{Role: "user", Type: "text", Sequence: 1}}
		err := store.SaveMessages(chat.ChatID, messages)
		if err == nil {
			t.Error("Expected error when saving message without props")
		}
	})
}

// TestGetMessages tests retrieving messages with filters
func TestGetMessages(t *testing.T) {
	test.Prepare(t, config.Conf)
	defer test.Clean()

	store, err := xun.NewXun(types.Setting{
		Connector: "default",
	})
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}

	// Create chat and messages
	chat := &types.Chat{
		AssistantID: "test_assistant",
	}
	err = store.CreateChat(chat)
	if err != nil {
		t.Fatalf("Failed to create chat: %v", err)
	}
	defer store.DeleteChat(chat.ChatID)

	// Save test messages
	messages := []*types.Message{
		{Role: "user", Type: "user_input", Props: map[string]interface{}{"content": "Hello"}, Sequence: 1, RequestID: "req_001"},
		{Role: "assistant", Type: "text", Props: map[string]interface{}{"content": "Hi there!"}, Sequence: 2, RequestID: "req_001", BlockID: "B1"},
		{Role: "user", Type: "user_input", Props: map[string]interface{}{"content": "Weather?"}, Sequence: 3, RequestID: "req_002"},
		{Role: "assistant", Type: "loading", Props: map[string]interface{}{"message": "Checking..."}, Sequence: 4, RequestID: "req_002", BlockID: "B2"},
		{Role: "assistant", Type: "text", Props: map[string]interface{}{"content": "Sunny!"}, Sequence: 5, RequestID: "req_002", BlockID: "B2", ThreadID: "T1"},
	}
	err = store.SaveMessages(chat.ChatID, messages)
	if err != nil {
		t.Fatalf("Failed to save messages: %v", err)
	}

	t.Run("GetAllMessages", func(t *testing.T) {
		retrieved, err := store.GetMessages(chat.ChatID, types.MessageFilter{})
		if err != nil {
			t.Fatalf("Failed to get messages: %v", err)
		}

		if len(retrieved) != 5 {
			t.Errorf("Expected 5 messages, got %d", len(retrieved))
		}

		// Verify order by sequence
		for i := 1; i < len(retrieved); i++ {
			if retrieved[i].Sequence < retrieved[i-1].Sequence {
				t.Error("Messages not ordered by sequence")
			}
		}
	})

	t.Run("FilterByRole", func(t *testing.T) {
		retrieved, err := store.GetMessages(chat.ChatID, types.MessageFilter{Role: "user"})
		if err != nil {
			t.Fatalf("Failed to get messages: %v", err)
		}

		if len(retrieved) != 2 {
			t.Errorf("Expected 2 user messages, got %d", len(retrieved))
		}

		for _, msg := range retrieved {
			if msg.Role != "user" {
				t.Errorf("Expected role 'user', got '%s'", msg.Role)
			}
		}
	})

	t.Run("FilterByRequestID", func(t *testing.T) {
		retrieved, err := store.GetMessages(chat.ChatID, types.MessageFilter{RequestID: "req_002"})
		if err != nil {
			t.Fatalf("Failed to get messages: %v", err)
		}

		if len(retrieved) != 3 {
			t.Errorf("Expected 3 messages for req_002, got %d", len(retrieved))
		}
	})

	t.Run("FilterByBlockID", func(t *testing.T) {
		retrieved, err := store.GetMessages(chat.ChatID, types.MessageFilter{BlockID: "B2"})
		if err != nil {
			t.Fatalf("Failed to get messages: %v", err)
		}

		if len(retrieved) != 2 {
			t.Errorf("Expected 2 messages in block B2, got %d", len(retrieved))
		}
	})

	t.Run("FilterByThreadID", func(t *testing.T) {
		retrieved, err := store.GetMessages(chat.ChatID, types.MessageFilter{ThreadID: "T1"})
		if err != nil {
			t.Fatalf("Failed to get messages: %v", err)
		}

		if len(retrieved) != 1 {
			t.Errorf("Expected 1 message in thread T1, got %d", len(retrieved))
		}
	})

	t.Run("FilterByType", func(t *testing.T) {
		retrieved, err := store.GetMessages(chat.ChatID, types.MessageFilter{Type: "loading"})
		if err != nil {
			t.Fatalf("Failed to get messages: %v", err)
		}

		if len(retrieved) != 1 {
			t.Errorf("Expected 1 loading message, got %d", len(retrieved))
		}
	})

	t.Run("FilterWithLimit", func(t *testing.T) {
		retrieved, err := store.GetMessages(chat.ChatID, types.MessageFilter{Limit: 2})
		if err != nil {
			t.Fatalf("Failed to get messages: %v", err)
		}

		if len(retrieved) != 2 {
			t.Errorf("Expected 2 messages with limit, got %d", len(retrieved))
		}
	})

	t.Run("FilterWithOffset", func(t *testing.T) {
		retrieved, err := store.GetMessages(chat.ChatID, types.MessageFilter{Offset: 3})
		if err != nil {
			t.Fatalf("Failed to get messages: %v", err)
		}

		if len(retrieved) != 2 {
			t.Errorf("Expected 2 messages with offset 3, got %d", len(retrieved))
		}
	})

	t.Run("FilterWithLimitAndOffset", func(t *testing.T) {
		retrieved, err := store.GetMessages(chat.ChatID, types.MessageFilter{Limit: 2, Offset: 1})
		if err != nil {
			t.Fatalf("Failed to get messages: %v", err)
		}

		if len(retrieved) != 2 {
			t.Errorf("Expected 2 messages, got %d", len(retrieved))
		}

		// Should be sequence 2 and 3
		if len(retrieved) >= 2 {
			if retrieved[0].Sequence != 2 {
				t.Errorf("Expected first message sequence 2, got %d", retrieved[0].Sequence)
			}
		}
	})

	t.Run("GetMessagesWithEmptyChatID", func(t *testing.T) {
		_, err := store.GetMessages("", types.MessageFilter{})
		if err == nil {
			t.Error("Expected error when getting messages without chat_id")
		}
	})

	t.Run("GetMessagesFromNonExistentChat", func(t *testing.T) {
		retrieved, err := store.GetMessages("nonexistent_chat", types.MessageFilter{})
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}
		if len(retrieved) != 0 {
			t.Errorf("Expected 0 messages from non-existent chat, got %d", len(retrieved))
		}
	})
}

// TestUpdateMessage tests updating messages
func TestUpdateMessage(t *testing.T) {
	test.Prepare(t, config.Conf)
	defer test.Clean()

	store, err := xun.NewXun(types.Setting{
		Connector: "default",
	})
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}

	// Create chat and message
	chat := &types.Chat{
		AssistantID: "test_assistant",
	}
	err = store.CreateChat(chat)
	if err != nil {
		t.Fatalf("Failed to create chat: %v", err)
	}
	defer store.DeleteChat(chat.ChatID)

	messages := []*types.Message{
		{
			MessageID: fmt.Sprintf("msg_%d", time.Now().UnixNano()),
			Role:      "assistant",
			Type:      "loading",
			Props:     map[string]interface{}{"message": "Loading..."},
			Sequence:  1,
		},
	}
	err = store.SaveMessages(chat.ChatID, messages)
	if err != nil {
		t.Fatalf("Failed to save message: %v", err)
	}

	messageID := messages[0].MessageID

	t.Run("UpdateProps", func(t *testing.T) {
		err := store.UpdateMessage(messageID, map[string]interface{}{
			"props": map[string]interface{}{"content": "Updated content"},
		})
		if err != nil {
			t.Fatalf("Failed to update message: %v", err)
		}

		retrieved, err := store.GetMessages(chat.ChatID, types.MessageFilter{})
		if err != nil {
			t.Fatalf("Failed to get messages: %v", err)
		}

		var found *types.Message
		for _, msg := range retrieved {
			if msg.MessageID == messageID {
				found = msg
				break
			}
		}

		if found == nil {
			t.Fatal("Could not find updated message")
		}

		if found.Props["content"] != "Updated content" {
			t.Errorf("Expected props content 'Updated content', got '%v'", found.Props["content"])
		}
	})

	t.Run("UpdateType", func(t *testing.T) {
		err := store.UpdateMessage(messageID, map[string]interface{}{
			"type": "text",
		})
		if err != nil {
			t.Fatalf("Failed to update message: %v", err)
		}

		retrieved, err := store.GetMessages(chat.ChatID, types.MessageFilter{})
		if err != nil {
			t.Fatalf("Failed to get messages: %v", err)
		}

		var found *types.Message
		for _, msg := range retrieved {
			if msg.MessageID == messageID {
				found = msg
				break
			}
		}

		if found == nil {
			t.Fatal("Could not find updated message")
		}

		if found.Type != "text" {
			t.Errorf("Expected type 'text', got '%s'", found.Type)
		}
	})

	t.Run("UpdateMetadata", func(t *testing.T) {
		err := store.UpdateMessage(messageID, map[string]interface{}{
			"metadata": map[string]interface{}{"updated": true},
		})
		if err != nil {
			t.Fatalf("Failed to update metadata: %v", err)
		}

		retrieved, err := store.GetMessages(chat.ChatID, types.MessageFilter{})
		if err != nil {
			t.Fatalf("Failed to get messages: %v", err)
		}

		var found *types.Message
		for _, msg := range retrieved {
			if msg.MessageID == messageID {
				found = msg
				break
			}
		}

		if found == nil {
			t.Fatal("Could not find updated message")
		}

		if found.Metadata == nil || found.Metadata["updated"] != true {
			t.Errorf("Expected metadata updated=true, got %v", found.Metadata)
		}
	})

	t.Run("UpdateNonExistentMessage", func(t *testing.T) {
		err := store.UpdateMessage("nonexistent_msg", map[string]interface{}{
			"type": "text",
		})
		if err == nil {
			t.Error("Expected error when updating non-existent message")
		}
	})

	t.Run("UpdateWithEmptyID", func(t *testing.T) {
		err := store.UpdateMessage("", map[string]interface{}{
			"type": "text",
		})
		if err == nil {
			t.Error("Expected error when updating with empty ID")
		}
	})

	t.Run("UpdateWithEmptyFields", func(t *testing.T) {
		err := store.UpdateMessage(messageID, map[string]interface{}{})
		if err == nil {
			t.Error("Expected error when updating with empty fields")
		}
	})
}

// TestDeleteMessages tests deleting messages
func TestDeleteMessages(t *testing.T) {
	test.Prepare(t, config.Conf)
	defer test.Clean()

	store, err := xun.NewXun(types.Setting{
		Connector: "default",
	})
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}

	t.Run("DeleteSingleMessage", func(t *testing.T) {
		chat := &types.Chat{AssistantID: "test_assistant"}
		err := store.CreateChat(chat)
		if err != nil {
			t.Fatalf("Failed to create chat: %v", err)
		}
		defer store.DeleteChat(chat.ChatID)

		msgID := fmt.Sprintf("msg_del_%d", time.Now().UnixNano())
		messages := []*types.Message{
			{MessageID: msgID, Role: "user", Type: "text", Props: map[string]interface{}{"content": "test"}, Sequence: 1},
		}
		err = store.SaveMessages(chat.ChatID, messages)
		if err != nil {
			t.Fatalf("Failed to save message: %v", err)
		}

		err = store.DeleteMessages(chat.ChatID, []string{msgID})
		if err != nil {
			t.Fatalf("Failed to delete message: %v", err)
		}

		// Verify deleted
		retrieved, err := store.GetMessages(chat.ChatID, types.MessageFilter{})
		if err != nil {
			t.Fatalf("Failed to get messages: %v", err)
		}

		for _, msg := range retrieved {
			if msg.MessageID == msgID {
				t.Error("Message should have been deleted")
			}
		}
	})

	t.Run("DeleteMultipleMessages", func(t *testing.T) {
		chat := &types.Chat{AssistantID: "test_assistant"}
		err := store.CreateChat(chat)
		if err != nil {
			t.Fatalf("Failed to create chat: %v", err)
		}
		defer store.DeleteChat(chat.ChatID)

		msgID1 := fmt.Sprintf("msg_del1_%d", time.Now().UnixNano())
		msgID2 := fmt.Sprintf("msg_del2_%d", time.Now().UnixNano())
		msgID3 := fmt.Sprintf("msg_del3_%d", time.Now().UnixNano())

		messages := []*types.Message{
			{MessageID: msgID1, Role: "user", Type: "text", Props: map[string]interface{}{"content": "1"}, Sequence: 1},
			{MessageID: msgID2, Role: "assistant", Type: "text", Props: map[string]interface{}{"content": "2"}, Sequence: 2},
			{MessageID: msgID3, Role: "user", Type: "text", Props: map[string]interface{}{"content": "3"}, Sequence: 3},
		}
		err = store.SaveMessages(chat.ChatID, messages)
		if err != nil {
			t.Fatalf("Failed to save messages: %v", err)
		}

		// Delete first two
		err = store.DeleteMessages(chat.ChatID, []string{msgID1, msgID2})
		if err != nil {
			t.Fatalf("Failed to delete messages: %v", err)
		}

		// Verify
		retrieved, err := store.GetMessages(chat.ChatID, types.MessageFilter{})
		if err != nil {
			t.Fatalf("Failed to get messages: %v", err)
		}

		if len(retrieved) != 1 {
			t.Errorf("Expected 1 remaining message, got %d", len(retrieved))
		}

		if len(retrieved) > 0 && retrieved[0].MessageID != msgID3 {
			t.Errorf("Expected remaining message to be %s, got %s", msgID3, retrieved[0].MessageID)
		}
	})

	t.Run("DeleteEmptyList", func(t *testing.T) {
		chat := &types.Chat{AssistantID: "test_assistant"}
		err := store.CreateChat(chat)
		if err != nil {
			t.Fatalf("Failed to create chat: %v", err)
		}
		defer store.DeleteChat(chat.ChatID)

		err = store.DeleteMessages(chat.ChatID, []string{})
		if err != nil {
			t.Errorf("Expected no error for empty delete list, got: %v", err)
		}
	})

	t.Run("DeleteWithEmptyChatID", func(t *testing.T) {
		err := store.DeleteMessages("", []string{"msg_123"})
		if err == nil {
			t.Error("Expected error when deleting with empty chat_id")
		}
	})
}

// TestMessageCompleteWorkflow tests a complete message workflow
func TestMessageCompleteWorkflow(t *testing.T) {
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
			Title:       "Message Workflow Test",
		}
		err := store.CreateChat(chat)
		if err != nil {
			t.Fatalf("Failed to create chat: %v", err)
		}
		defer store.DeleteChat(chat.ChatID)

		// 2. Save batch messages (simulating a request)
		requestID := fmt.Sprintf("req_%d", time.Now().UnixNano())
		messages := []*types.Message{
			{
				Role:        "user",
				Type:        "user_input",
				Props:       map[string]interface{}{"content": "What's the weather in SF?"},
				Sequence:    1,
				RequestID:   requestID,
				AssistantID: "workflow_assistant",
			},
			{
				Role:        "assistant",
				Type:        "loading",
				Props:       map[string]interface{}{"message": "Checking weather..."},
				Sequence:    2,
				RequestID:   requestID,
				BlockID:     "B1",
				AssistantID: "workflow_assistant",
			},
			{
				Role:        "assistant",
				Type:        "tool_call",
				Props:       map[string]interface{}{"id": "call_weather", "name": "get_weather", "arguments": `{"location":"SF"}`},
				Sequence:    3,
				RequestID:   requestID,
				BlockID:     "B1",
				AssistantID: "workflow_assistant",
			},
			{
				Role:        "assistant",
				Type:        "text",
				Props:       map[string]interface{}{"content": "The weather in San Francisco is 18°C and sunny."},
				Sequence:    4,
				RequestID:   requestID,
				BlockID:     "B1",
				AssistantID: "workflow_assistant",
				Metadata:    map[string]interface{}{"tool_call_id": "call_weather"},
			},
		}

		err = store.SaveMessages(chat.ChatID, messages)
		if err != nil {
			t.Fatalf("Failed to save messages: %v", err)
		}
		t.Logf("Saved %d messages in single batch", len(messages))

		// 3. Get all messages
		retrieved, err := store.GetMessages(chat.ChatID, types.MessageFilter{})
		if err != nil {
			t.Fatalf("Failed to get messages: %v", err)
		}

		if len(retrieved) != 4 {
			t.Errorf("Expected 4 messages, got %d", len(retrieved))
		}

		// 4. Filter by request
		byRequest, err := store.GetMessages(chat.ChatID, types.MessageFilter{RequestID: requestID})
		if err != nil {
			t.Fatalf("Failed to filter by request: %v", err)
		}

		if len(byRequest) != 4 {
			t.Errorf("Expected 4 messages for request, got %d", len(byRequest))
		}

		// 5. Filter by block
		byBlock, err := store.GetMessages(chat.ChatID, types.MessageFilter{BlockID: "B1"})
		if err != nil {
			t.Fatalf("Failed to filter by block: %v", err)
		}

		if len(byBlock) != 3 {
			t.Errorf("Expected 3 messages in block B1, got %d", len(byBlock))
		}

		// 6. Update loading message to text (simulating stream completion)
		var loadingMsgID string
		for _, msg := range retrieved {
			if msg.Type == "loading" {
				loadingMsgID = msg.MessageID
				break
			}
		}

		if loadingMsgID != "" {
			err = store.UpdateMessage(loadingMsgID, map[string]interface{}{
				"type":  "text",
				"props": map[string]interface{}{"content": "Weather check complete."},
			})
			if err != nil {
				t.Fatalf("Failed to update message: %v", err)
			}
		}

		// 7. Delete a message
		if len(retrieved) > 0 {
			err = store.DeleteMessages(chat.ChatID, []string{retrieved[0].MessageID})
			if err != nil {
				t.Fatalf("Failed to delete message: %v", err)
			}
		}

		// 8. Verify final state
		final, err := store.GetMessages(chat.ChatID, types.MessageFilter{})
		if err != nil {
			t.Fatalf("Failed to get final messages: %v", err)
		}

		if len(final) != 3 {
			t.Errorf("Expected 3 messages after delete, got %d", len(final))
		}

		t.Log("Complete message workflow passed!")
	})
}

// TestConcurrentMessages tests concurrent message storage
func TestConcurrentMessages(t *testing.T) {
	test.Prepare(t, config.Conf)
	defer test.Clean()

	store, err := xun.NewXun(types.Setting{
		Connector: "default",
	})
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}

	t.Run("ConcurrentThreadMessages", func(t *testing.T) {
		chat := &types.Chat{AssistantID: "test_assistant"}
		err := store.CreateChat(chat)
		if err != nil {
			t.Fatalf("Failed to create chat: %v", err)
		}
		defer store.DeleteChat(chat.ChatID)

		// Simulate concurrent operations with different threads
		messages := []*types.Message{
			{Role: "assistant", Type: "text", Props: map[string]interface{}{"content": "Weather result"}, Sequence: 1, BlockID: "B1", ThreadID: "T1"},
			{Role: "assistant", Type: "text", Props: map[string]interface{}{"content": "News result"}, Sequence: 2, BlockID: "B1", ThreadID: "T2"},
			{Role: "assistant", Type: "text", Props: map[string]interface{}{"content": "Stock result"}, Sequence: 3, BlockID: "B1", ThreadID: "T3"},
			{Role: "assistant", Type: "text", Props: map[string]interface{}{"content": "Summary"}, Sequence: 4, BlockID: "B2"},
		}

		err = store.SaveMessages(chat.ChatID, messages)
		if err != nil {
			t.Fatalf("Failed to save concurrent messages: %v", err)
		}

		// Verify all saved
		all, err := store.GetMessages(chat.ChatID, types.MessageFilter{})
		if err != nil {
			t.Fatalf("Failed to get messages: %v", err)
		}

		if len(all) != 4 {
			t.Errorf("Expected 4 messages, got %d", len(all))
		}

		// Filter by thread
		t1Messages, err := store.GetMessages(chat.ChatID, types.MessageFilter{ThreadID: "T1"})
		if err != nil {
			t.Fatalf("Failed to filter by thread: %v", err)
		}

		if len(t1Messages) != 1 {
			t.Errorf("Expected 1 message in thread T1, got %d", len(t1Messages))
		}

		// Filter by block
		b1Messages, err := store.GetMessages(chat.ChatID, types.MessageFilter{BlockID: "B1"})
		if err != nil {
			t.Fatalf("Failed to filter by block: %v", err)
		}

		if len(b1Messages) != 3 {
			t.Errorf("Expected 3 messages in block B1, got %d", len(b1Messages))
		}

		t.Log("Concurrent thread messages test passed!")
	})
}
