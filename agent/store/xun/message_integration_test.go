//go:build integration

package xun_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/yaoapp/yao/agent/store/types"
	"github.com/yaoapp/yao/agent/store/xun"
	"github.com/yaoapp/yao/unit-test/agent/testprepare"
)

func TestSaveMessages(t *testing.T) {
	testprepare.PrepareSandbox(t)

	store, err := xun.NewXun(types.Setting{Connector: "default"})
	require.NoError(t, err)

	chat := &types.Chat{AssistantID: "test_assistant", Title: "Message Test Chat"}
	err = store.CreateChat(chat)
	require.NoError(t, err)
	t.Cleanup(func() { store.DeleteChat(chat.ChatID) })

	t.Run("SaveSingleMessage", func(t *testing.T) {
		msgChat := &types.Chat{AssistantID: "test_assistant"}
		err := store.CreateChat(msgChat)
		require.NoError(t, err)
		t.Cleanup(func() { store.DeleteChat(msgChat.ChatID) })

		messages := []*types.Message{
			{Role: "user", Type: "text", Props: map[string]interface{}{"content": "Hello, world!"}, Sequence: 1},
		}

		err = store.SaveMessages(msgChat.ChatID, messages)
		require.NoError(t, err)

		retrieved, err := store.GetMessages(msgChat.ChatID, types.MessageFilter{})
		require.NoError(t, err)
		require.GreaterOrEqual(t, len(retrieved), 1)

		var found *types.Message
		for _, msg := range retrieved {
			if msg.Sequence == 1 && msg.Type == "text" {
				found = msg
				break
			}
		}
		require.NotNil(t, found)
		assert.Equal(t, "user", found.Role)
		assert.Equal(t, "Hello, world!", found.Props["content"])
	})

	t.Run("SaveBatchMessages", func(t *testing.T) {
		batchChat := &types.Chat{AssistantID: "test_assistant", Title: "Batch Message Test"}
		err := store.CreateChat(batchChat)
		require.NoError(t, err)
		t.Cleanup(func() { store.DeleteChat(batchChat.ChatID) })

		messages := []*types.Message{
			{Role: "user", Type: "user_input", Props: map[string]interface{}{"content": "What's the weather?"}, Sequence: 1, RequestID: "req_001", AssistantID: "weather_assistant"},
			{Role: "assistant", Type: "loading", Props: map[string]interface{}{"message": "Checking weather..."}, Sequence: 2, RequestID: "req_001", BlockID: "B1", AssistantID: "weather_assistant"},
			{Role: "assistant", Type: "text", Props: map[string]interface{}{"content": "The weather is sunny, 25°C."}, Sequence: 3, RequestID: "req_001", BlockID: "B1", AssistantID: "weather_assistant"},
		}

		err = store.SaveMessages(batchChat.ChatID, messages)
		require.NoError(t, err)

		retrieved, err := store.GetMessages(batchChat.ChatID, types.MessageFilter{})
		require.NoError(t, err)
		assert.Equal(t, 3, len(retrieved))

		if len(retrieved) >= 3 {
			assert.Equal(t, 1, retrieved[0].Sequence)
			assert.Equal(t, 3, retrieved[2].Sequence)
		}
	})

	t.Run("SaveMessageWithAllFields", func(t *testing.T) {
		fullChat := &types.Chat{AssistantID: "test_assistant"}
		err := store.CreateChat(fullChat)
		require.NoError(t, err)
		t.Cleanup(func() { store.DeleteChat(fullChat.ChatID) })

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
		require.NoError(t, err)

		retrieved, err := store.GetMessages(fullChat.ChatID, types.MessageFilter{})
		require.NoError(t, err)
		require.Equal(t, 1, len(retrieved))

		msg := retrieved[0]
		assert.Equal(t, "req_full", msg.RequestID)
		assert.Equal(t, "B1", msg.BlockID)
		assert.Equal(t, "T1", msg.ThreadID)
		assert.Equal(t, "weather_assistant", msg.AssistantID)
		require.NotNil(t, msg.Metadata)
		assert.Equal(t, "call_123", msg.Metadata["tool_call_id"])
	})

	t.Run("SaveMessageWithConnector", func(t *testing.T) {
		connChat := &types.Chat{AssistantID: "test_assistant"}
		err := store.CreateChat(connChat)
		require.NoError(t, err)
		t.Cleanup(func() { store.DeleteChat(connChat.ChatID) })

		messages := []*types.Message{
			{Role: "user", Type: "user_input", Props: map[string]interface{}{"content": "Hello"}, Sequence: 1, Connector: "openai", AssistantID: "test_assistant"},
			{Role: "assistant", Type: "text", Props: map[string]interface{}{"content": "Hi there!"}, Sequence: 2, Connector: "openai", AssistantID: "test_assistant"},
			{Role: "user", Type: "user_input", Props: map[string]interface{}{"content": "Switch to Claude"}, Sequence: 3, Connector: "anthropic", AssistantID: "test_assistant"},
			{Role: "assistant", Type: "text", Props: map[string]interface{}{"content": "Now using Claude!"}, Sequence: 4, Connector: "anthropic", AssistantID: "test_assistant"},
		}

		err = store.SaveMessages(connChat.ChatID, messages)
		require.NoError(t, err)

		retrieved, err := store.GetMessages(connChat.ChatID, types.MessageFilter{})
		require.NoError(t, err)
		require.Equal(t, 4, len(retrieved))

		for _, msg := range retrieved {
			if msg.Sequence <= 2 {
				assert.Equal(t, "openai", msg.Connector)
			} else {
				assert.Equal(t, "anthropic", msg.Connector)
			}
		}
	})

	t.Run("SaveMessageWithMode", func(t *testing.T) {
		modeChat := &types.Chat{AssistantID: "test_assistant"}
		err := store.CreateChat(modeChat)
		require.NoError(t, err)
		t.Cleanup(func() { store.DeleteChat(modeChat.ChatID) })

		messages := []*types.Message{
			{Role: "user", Type: "user_input", Props: map[string]interface{}{"content": "Hello in chat mode"}, Sequence: 1, Mode: "chat", Connector: "deepseek.v3", AssistantID: "test_assistant"},
			{Role: "assistant", Type: "text", Props: map[string]interface{}{"content": "Hi there in chat mode!"}, Sequence: 2, Mode: "chat", Connector: "deepseek.v3", AssistantID: "test_assistant"},
			{Role: "user", Type: "user_input", Props: map[string]interface{}{"content": "Now run a task"}, Sequence: 3, Mode: "task", Connector: "deepseek.v3", AssistantID: "test_assistant"},
			{Role: "assistant", Type: "text", Props: map[string]interface{}{"content": "Running task!"}, Sequence: 4, Mode: "task", Connector: "deepseek.v3", AssistantID: "test_assistant"},
		}

		err = store.SaveMessages(modeChat.ChatID, messages)
		require.NoError(t, err)

		retrieved, err := store.GetMessages(modeChat.ChatID, types.MessageFilter{})
		require.NoError(t, err)
		require.Equal(t, 4, len(retrieved))

		for _, msg := range retrieved {
			if msg.Sequence <= 2 {
				assert.Equal(t, "chat", msg.Mode)
			} else {
				assert.Equal(t, "task", msg.Mode)
			}
		}
	})

	t.Run("SaveMessageWithEmptyConnector", func(t *testing.T) {
		emptyConnChat := &types.Chat{AssistantID: "test_assistant"}
		err := store.CreateChat(emptyConnChat)
		require.NoError(t, err)
		t.Cleanup(func() { store.DeleteChat(emptyConnChat.ChatID) })

		messages := []*types.Message{
			{Role: "user", Type: "text", Props: map[string]interface{}{"content": "No connector"}, Sequence: 1},
		}

		err = store.SaveMessages(emptyConnChat.ChatID, messages)
		require.NoError(t, err)

		retrieved, err := store.GetMessages(emptyConnChat.ChatID, types.MessageFilter{})
		require.NoError(t, err)
		require.Equal(t, 1, len(retrieved))
		assert.Empty(t, retrieved[0].Connector)
	})

	t.Run("SaveEmptyMessages", func(t *testing.T) {
		err := store.SaveMessages(chat.ChatID, []*types.Message{})
		assert.NoError(t, err)
	})

	t.Run("SaveMessagesWithoutChatID", func(t *testing.T) {
		messages := []*types.Message{{Role: "user", Type: "text", Props: map[string]interface{}{"content": "test"}}}
		err := store.SaveMessages("", messages)
		assert.Error(t, err)
	})

	t.Run("SaveMessageWithoutRole", func(t *testing.T) {
		messages := []*types.Message{{Type: "text", Props: map[string]interface{}{"content": "test"}, Sequence: 1}}
		err := store.SaveMessages(chat.ChatID, messages)
		assert.Error(t, err)
	})

	t.Run("SaveMessageWithoutType", func(t *testing.T) {
		messages := []*types.Message{{Role: "user", Props: map[string]interface{}{"content": "test"}, Sequence: 1}}
		err := store.SaveMessages(chat.ChatID, messages)
		assert.Error(t, err)
	})

	t.Run("SaveMessageWithoutProps", func(t *testing.T) {
		messages := []*types.Message{{Role: "user", Type: "text", Sequence: 1}}
		err := store.SaveMessages(chat.ChatID, messages)
		assert.Error(t, err)
	})
}

func TestGetMessages(t *testing.T) {
	testprepare.PrepareSandbox(t)

	store, err := xun.NewXun(types.Setting{Connector: "default"})
	require.NoError(t, err)

	chat := &types.Chat{AssistantID: "test_assistant"}
	err = store.CreateChat(chat)
	require.NoError(t, err)
	t.Cleanup(func() { store.DeleteChat(chat.ChatID) })

	messages := []*types.Message{
		{Role: "user", Type: "user_input", Props: map[string]interface{}{"content": "Hello"}, Sequence: 1, RequestID: "req_001"},
		{Role: "assistant", Type: "text", Props: map[string]interface{}{"content": "Hi there!"}, Sequence: 2, RequestID: "req_001", BlockID: "B1"},
		{Role: "user", Type: "user_input", Props: map[string]interface{}{"content": "Weather?"}, Sequence: 3, RequestID: "req_002"},
		{Role: "assistant", Type: "loading", Props: map[string]interface{}{"message": "Checking..."}, Sequence: 4, RequestID: "req_002", BlockID: "B2"},
		{Role: "assistant", Type: "text", Props: map[string]interface{}{"content": "Sunny!"}, Sequence: 5, RequestID: "req_002", BlockID: "B2", ThreadID: "T1"},
	}
	err = store.SaveMessages(chat.ChatID, messages)
	require.NoError(t, err)

	t.Run("GetAllMessages", func(t *testing.T) {
		retrieved, err := store.GetMessages(chat.ChatID, types.MessageFilter{})
		require.NoError(t, err)
		assert.Equal(t, 5, len(retrieved))

		for i := 1; i < len(retrieved); i++ {
			assert.GreaterOrEqual(t, retrieved[i].Sequence, retrieved[i-1].Sequence)
		}
	})

	t.Run("FilterByRole", func(t *testing.T) {
		retrieved, err := store.GetMessages(chat.ChatID, types.MessageFilter{Role: "user"})
		require.NoError(t, err)
		assert.Equal(t, 2, len(retrieved))

		for _, msg := range retrieved {
			assert.Equal(t, "user", msg.Role)
		}
	})

	t.Run("FilterByRequestID", func(t *testing.T) {
		retrieved, err := store.GetMessages(chat.ChatID, types.MessageFilter{RequestID: "req_002"})
		require.NoError(t, err)
		assert.Equal(t, 3, len(retrieved))
	})

	t.Run("FilterByBlockID", func(t *testing.T) {
		retrieved, err := store.GetMessages(chat.ChatID, types.MessageFilter{BlockID: "B2"})
		require.NoError(t, err)
		assert.Equal(t, 2, len(retrieved))
	})

	t.Run("FilterByThreadID", func(t *testing.T) {
		retrieved, err := store.GetMessages(chat.ChatID, types.MessageFilter{ThreadID: "T1"})
		require.NoError(t, err)
		assert.Equal(t, 1, len(retrieved))
	})

	t.Run("FilterByType", func(t *testing.T) {
		retrieved, err := store.GetMessages(chat.ChatID, types.MessageFilter{Type: "loading"})
		require.NoError(t, err)
		assert.Equal(t, 1, len(retrieved))
	})

	t.Run("FilterWithLimit", func(t *testing.T) {
		retrieved, err := store.GetMessages(chat.ChatID, types.MessageFilter{Limit: 2})
		require.NoError(t, err)
		assert.Equal(t, 2, len(retrieved))
	})

	t.Run("FilterWithOffset", func(t *testing.T) {
		retrieved, err := store.GetMessages(chat.ChatID, types.MessageFilter{Offset: 3})
		require.NoError(t, err)
		assert.Equal(t, 2, len(retrieved))
	})

	t.Run("FilterWithLimitAndOffset", func(t *testing.T) {
		retrieved, err := store.GetMessages(chat.ChatID, types.MessageFilter{Limit: 2, Offset: 1})
		require.NoError(t, err)
		assert.Equal(t, 2, len(retrieved))

		if len(retrieved) >= 2 {
			assert.Equal(t, 2, retrieved[0].Sequence)
		}
	})

	t.Run("GetMessagesWithEmptyChatID", func(t *testing.T) {
		_, err := store.GetMessages("", types.MessageFilter{})
		assert.Error(t, err)
	})

	t.Run("GetMessagesFromNonExistentChat", func(t *testing.T) {
		retrieved, err := store.GetMessages("nonexistent_chat", types.MessageFilter{})
		require.NoError(t, err)
		assert.Equal(t, 0, len(retrieved))
	})

	t.Run("OrderByCreatedAtThenSequence", func(t *testing.T) {
		orderChat := &types.Chat{AssistantID: "test_assistant"}
		err := store.CreateChat(orderChat)
		require.NoError(t, err)
		t.Cleanup(func() { store.DeleteChat(orderChat.ChatID) })

		req1Messages := []*types.Message{
			{Role: "user", Type: "user_input", Props: map[string]interface{}{"content": "Request 1 - Message 1"}, Sequence: 1, RequestID: "order_req_001"},
			{Role: "assistant", Type: "text", Props: map[string]interface{}{"content": "Request 1 - Response 1"}, Sequence: 2, RequestID: "order_req_001"},
		}
		err = store.SaveMessages(orderChat.ChatID, req1Messages)
		require.NoError(t, err)

		time.Sleep(1100 * time.Millisecond)

		req2Messages := []*types.Message{
			{Role: "user", Type: "user_input", Props: map[string]interface{}{"content": "Request 2 - Message 1"}, Sequence: 1, RequestID: "order_req_002"},
			{Role: "assistant", Type: "text", Props: map[string]interface{}{"content": "Request 2 - Response 1"}, Sequence: 2, RequestID: "order_req_002"},
		}
		err = store.SaveMessages(orderChat.ChatID, req2Messages)
		require.NoError(t, err)

		retrieved, err := store.GetMessages(orderChat.ChatID, types.MessageFilter{})
		require.NoError(t, err)
		require.Equal(t, 4, len(retrieved))

		expectedOrder := []struct {
			requestID string
			sequence  int
		}{
			{"order_req_001", 1},
			{"order_req_001", 2},
			{"order_req_002", 1},
			{"order_req_002", 2},
		}

		for i, expected := range expectedOrder {
			assert.Equal(t, expected.requestID, retrieved[i].RequestID)
			assert.Equal(t, expected.sequence, retrieved[i].Sequence)
		}
	})
}

func TestUpdateMessage(t *testing.T) {
	testprepare.PrepareSandbox(t)

	store, err := xun.NewXun(types.Setting{Connector: "default"})
	require.NoError(t, err)

	chat := &types.Chat{AssistantID: "test_assistant"}
	err = store.CreateChat(chat)
	require.NoError(t, err)
	t.Cleanup(func() { store.DeleteChat(chat.ChatID) })

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
	require.NoError(t, err)
	messageID := messages[0].MessageID

	t.Run("UpdateProps", func(t *testing.T) {
		err := store.UpdateMessage(messageID, map[string]interface{}{
			"props": map[string]interface{}{"content": "Updated content"},
		})
		require.NoError(t, err)

		retrieved, err := store.GetMessages(chat.ChatID, types.MessageFilter{})
		require.NoError(t, err)

		var found *types.Message
		for _, msg := range retrieved {
			if msg.MessageID == messageID {
				found = msg
				break
			}
		}
		require.NotNil(t, found)
		assert.Equal(t, "Updated content", found.Props["content"])
	})

	t.Run("UpdateType", func(t *testing.T) {
		err := store.UpdateMessage(messageID, map[string]interface{}{"type": "text"})
		require.NoError(t, err)

		retrieved, err := store.GetMessages(chat.ChatID, types.MessageFilter{})
		require.NoError(t, err)

		var found *types.Message
		for _, msg := range retrieved {
			if msg.MessageID == messageID {
				found = msg
				break
			}
		}
		require.NotNil(t, found)
		assert.Equal(t, "text", found.Type)
	})

	t.Run("UpdateMetadata", func(t *testing.T) {
		err := store.UpdateMessage(messageID, map[string]interface{}{
			"metadata": map[string]interface{}{"updated": true},
		})
		require.NoError(t, err)

		retrieved, err := store.GetMessages(chat.ChatID, types.MessageFilter{})
		require.NoError(t, err)

		var found *types.Message
		for _, msg := range retrieved {
			if msg.MessageID == messageID {
				found = msg
				break
			}
		}
		require.NotNil(t, found)
		require.NotNil(t, found.Metadata)
		assert.Equal(t, true, found.Metadata["updated"])
	})

	t.Run("UpdateNonExistentMessage", func(t *testing.T) {
		err := store.UpdateMessage("nonexistent_msg", map[string]interface{}{"type": "text"})
		assert.Error(t, err)
	})

	t.Run("UpdateWithEmptyID", func(t *testing.T) {
		err := store.UpdateMessage("", map[string]interface{}{"type": "text"})
		assert.Error(t, err)
	})

	t.Run("UpdateWithEmptyFields", func(t *testing.T) {
		err := store.UpdateMessage(messageID, map[string]interface{}{})
		assert.Error(t, err)
	})
}

func TestDeleteMessages(t *testing.T) {
	testprepare.PrepareSandbox(t)

	store, err := xun.NewXun(types.Setting{Connector: "default"})
	require.NoError(t, err)

	t.Run("DeleteSingleMessage", func(t *testing.T) {
		chat := &types.Chat{AssistantID: "test_assistant"}
		err := store.CreateChat(chat)
		require.NoError(t, err)
		t.Cleanup(func() { store.DeleteChat(chat.ChatID) })

		msgID := fmt.Sprintf("msg_del_%d", time.Now().UnixNano())
		messages := []*types.Message{
			{MessageID: msgID, Role: "user", Type: "text", Props: map[string]interface{}{"content": "test"}, Sequence: 1},
		}
		err = store.SaveMessages(chat.ChatID, messages)
		require.NoError(t, err)

		err = store.DeleteMessages(chat.ChatID, []string{msgID})
		require.NoError(t, err)

		retrieved, err := store.GetMessages(chat.ChatID, types.MessageFilter{})
		require.NoError(t, err)
		for _, msg := range retrieved {
			assert.NotEqual(t, msgID, msg.MessageID)
		}
	})

	t.Run("DeleteMultipleMessages", func(t *testing.T) {
		chat := &types.Chat{AssistantID: "test_assistant"}
		err := store.CreateChat(chat)
		require.NoError(t, err)
		t.Cleanup(func() { store.DeleteChat(chat.ChatID) })

		msgID1 := fmt.Sprintf("msg_del1_%d", time.Now().UnixNano())
		msgID2 := fmt.Sprintf("msg_del2_%d", time.Now().UnixNano())
		msgID3 := fmt.Sprintf("msg_del3_%d", time.Now().UnixNano())

		messages := []*types.Message{
			{MessageID: msgID1, Role: "user", Type: "text", Props: map[string]interface{}{"content": "1"}, Sequence: 1},
			{MessageID: msgID2, Role: "assistant", Type: "text", Props: map[string]interface{}{"content": "2"}, Sequence: 2},
			{MessageID: msgID3, Role: "user", Type: "text", Props: map[string]interface{}{"content": "3"}, Sequence: 3},
		}
		err = store.SaveMessages(chat.ChatID, messages)
		require.NoError(t, err)

		err = store.DeleteMessages(chat.ChatID, []string{msgID1, msgID2})
		require.NoError(t, err)

		retrieved, err := store.GetMessages(chat.ChatID, types.MessageFilter{})
		require.NoError(t, err)
		assert.Equal(t, 1, len(retrieved))
		if len(retrieved) > 0 {
			assert.Equal(t, msgID3, retrieved[0].MessageID)
		}
	})

	t.Run("DeleteEmptyList", func(t *testing.T) {
		chat := &types.Chat{AssistantID: "test_assistant"}
		err := store.CreateChat(chat)
		require.NoError(t, err)
		t.Cleanup(func() { store.DeleteChat(chat.ChatID) })

		err = store.DeleteMessages(chat.ChatID, []string{})
		assert.NoError(t, err)
	})

	t.Run("DeleteWithEmptyChatID", func(t *testing.T) {
		err := store.DeleteMessages("", []string{"msg_123"})
		assert.Error(t, err)
	})
}

func TestMessageCompleteWorkflow(t *testing.T) {
	testprepare.PrepareSandbox(t)

	store, err := xun.NewXun(types.Setting{Connector: "default"})
	require.NoError(t, err)

	t.Run("CompleteWorkflow", func(t *testing.T) {
		chat := &types.Chat{AssistantID: "workflow_assistant", Title: "Message Workflow Test"}
		err := store.CreateChat(chat)
		require.NoError(t, err)
		t.Cleanup(func() { store.DeleteChat(chat.ChatID) })

		requestID := fmt.Sprintf("req_%d", time.Now().UnixNano())
		messages := []*types.Message{
			{Role: "user", Type: "user_input", Props: map[string]interface{}{"content": "What's the weather in SF?"}, Sequence: 1, RequestID: requestID, AssistantID: "workflow_assistant"},
			{Role: "assistant", Type: "loading", Props: map[string]interface{}{"message": "Checking weather..."}, Sequence: 2, RequestID: requestID, BlockID: "B1", AssistantID: "workflow_assistant"},
			{Role: "assistant", Type: "tool_call", Props: map[string]interface{}{"id": "call_weather", "name": "get_weather"}, Sequence: 3, RequestID: requestID, BlockID: "B1", AssistantID: "workflow_assistant"},
			{Role: "assistant", Type: "text", Props: map[string]interface{}{"content": "The weather in SF is 18°C and sunny."}, Sequence: 4, RequestID: requestID, BlockID: "B1", AssistantID: "workflow_assistant", Metadata: map[string]interface{}{"tool_call_id": "call_weather"}},
		}

		err = store.SaveMessages(chat.ChatID, messages)
		require.NoError(t, err)

		retrieved, err := store.GetMessages(chat.ChatID, types.MessageFilter{})
		require.NoError(t, err)
		assert.Equal(t, 4, len(retrieved))

		byRequest, err := store.GetMessages(chat.ChatID, types.MessageFilter{RequestID: requestID})
		require.NoError(t, err)
		assert.Equal(t, 4, len(byRequest))

		byBlock, err := store.GetMessages(chat.ChatID, types.MessageFilter{BlockID: "B1"})
		require.NoError(t, err)
		assert.Equal(t, 3, len(byBlock))

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
			require.NoError(t, err)
		}

		if len(retrieved) > 0 {
			err = store.DeleteMessages(chat.ChatID, []string{retrieved[0].MessageID})
			require.NoError(t, err)
		}

		final, err := store.GetMessages(chat.ChatID, types.MessageFilter{})
		require.NoError(t, err)
		assert.Equal(t, 3, len(final))
	})
}

func TestConcurrentMessages(t *testing.T) {
	testprepare.PrepareSandbox(t)

	store, err := xun.NewXun(types.Setting{Connector: "default"})
	require.NoError(t, err)

	t.Run("ConcurrentThreadMessages", func(t *testing.T) {
		chat := &types.Chat{AssistantID: "test_assistant"}
		err := store.CreateChat(chat)
		require.NoError(t, err)
		t.Cleanup(func() { store.DeleteChat(chat.ChatID) })

		messages := []*types.Message{
			{Role: "assistant", Type: "text", Props: map[string]interface{}{"content": "Weather result"}, Sequence: 1, BlockID: "B1", ThreadID: "T1"},
			{Role: "assistant", Type: "text", Props: map[string]interface{}{"content": "News result"}, Sequence: 2, BlockID: "B1", ThreadID: "T2"},
			{Role: "assistant", Type: "text", Props: map[string]interface{}{"content": "Stock result"}, Sequence: 3, BlockID: "B1", ThreadID: "T3"},
			{Role: "assistant", Type: "text", Props: map[string]interface{}{"content": "Summary"}, Sequence: 4, BlockID: "B2"},
		}

		err = store.SaveMessages(chat.ChatID, messages)
		require.NoError(t, err)

		all, err := store.GetMessages(chat.ChatID, types.MessageFilter{})
		require.NoError(t, err)
		assert.Equal(t, 4, len(all))

		t1Messages, err := store.GetMessages(chat.ChatID, types.MessageFilter{ThreadID: "T1"})
		require.NoError(t, err)
		assert.Equal(t, 1, len(t1Messages))

		b1Messages, err := store.GetMessages(chat.ChatID, types.MessageFilter{BlockID: "B1"})
		require.NoError(t, err)
		assert.Equal(t, 3, len(b1Messages))
	})

	t.Run("MessageHasID", func(t *testing.T) {
		chat := &types.Chat{AssistantID: "test_assistant"}
		err := store.CreateChat(chat)
		require.NoError(t, err)
		t.Cleanup(func() { store.DeleteChat(chat.ChatID) })

		messages := []*types.Message{
			{Role: "user", Type: "text", Props: map[string]interface{}{"content": "Hi"}, Sequence: 1},
			{Role: "assistant", Type: "text", Props: map[string]interface{}{"content": "Hello"}, Sequence: 2},
		}
		err = store.SaveMessages(chat.ChatID, messages)
		require.NoError(t, err)

		retrieved, err := store.GetMessages(chat.ChatID, types.MessageFilter{})
		require.NoError(t, err)
		require.Equal(t, 2, len(retrieved))
		for _, msg := range retrieved {
			assert.Greater(t, msg.ID, int64(0), "Message ID should be populated")
		}
	})

	t.Run("FilterWithBeforeID", func(t *testing.T) {
		chat := &types.Chat{AssistantID: "test_assistant"}
		err := store.CreateChat(chat)
		require.NoError(t, err)
		t.Cleanup(func() { store.DeleteChat(chat.ChatID) })

		for i := 1; i <= 5; i++ {
			err := store.SaveMessages(chat.ChatID, []*types.Message{
				{Role: "user", Type: "text", Props: map[string]interface{}{"i": i}, Sequence: i},
			})
			require.NoError(t, err)
		}

		all, err := store.GetMessages(chat.ChatID, types.MessageFilter{})
		require.NoError(t, err)
		require.Equal(t, 5, len(all))

		midID := all[2].ID
		require.Greater(t, midID, int64(0))

		retrieved, err := store.GetMessages(chat.ChatID, types.MessageFilter{BeforeID: midID, Limit: 10})
		require.NoError(t, err)
		assert.Equal(t, 2, len(retrieved))

		for i := 1; i < len(retrieved); i++ {
			assert.Less(t, retrieved[i-1].ID, retrieved[i].ID)
		}
		for _, msg := range retrieved {
			assert.Less(t, msg.ID, midID)
		}
	})
}
