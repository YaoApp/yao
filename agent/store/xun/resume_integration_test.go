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

func TestSaveResume(t *testing.T) {
	testprepare.PrepareSandbox(t)

	store, err := xun.NewXun(types.Setting{Connector: "default"})
	require.NoError(t, err)

	chat := &types.Chat{AssistantID: "test_assistant", Title: "Resume Test Chat"}
	err = store.CreateChat(chat)
	require.NoError(t, err)
	t.Cleanup(func() { store.DeleteChat(chat.ChatID) })

	t.Run("SaveSingleRecord", func(t *testing.T) {
		requestID := fmt.Sprintf("req_%d", time.Now().UnixNano())
		records := []*types.Resume{
			{
				ChatID:      chat.ChatID,
				RequestID:   requestID,
				AssistantID: "test_assistant",
				StackID:     "stack_001",
				StackDepth:  0,
				Type:        types.ResumeTypeLLM,
				Status:      types.ResumeStatusInterrupted,
				Sequence:    1,
			},
		}

		err := store.SaveResume(records)
		require.NoError(t, err)
		t.Cleanup(func() { store.DeleteResume(chat.ChatID) })

		retrieved, err := store.GetResume(chat.ChatID)
		require.NoError(t, err)

		found := false
		for _, r := range retrieved {
			if r.RequestID == requestID {
				found = true
				assert.Equal(t, types.ResumeTypeLLM, r.Type)
				assert.Equal(t, types.ResumeStatusInterrupted, r.Status)
				break
			}
		}
		assert.True(t, found)
	})

	t.Run("SaveBatchRecords", func(t *testing.T) {
		batchChat := &types.Chat{AssistantID: "test_assistant"}
		err := store.CreateChat(batchChat)
		require.NoError(t, err)
		t.Cleanup(func() { store.DeleteChat(batchChat.ChatID) })

		requestID := fmt.Sprintf("req_%d", time.Now().UnixNano())
		records := []*types.Resume{
			{ChatID: batchChat.ChatID, RequestID: requestID, AssistantID: "test_assistant", StackID: "stack_001", StackDepth: 0, Type: types.ResumeTypeInput, Status: types.ResumeStatusInterrupted, Sequence: 1},
			{ChatID: batchChat.ChatID, RequestID: requestID, AssistantID: "test_assistant", StackID: "stack_001", StackDepth: 0, Type: types.ResumeTypeHookCreate, Status: types.ResumeStatusInterrupted, Sequence: 2},
			{ChatID: batchChat.ChatID, RequestID: requestID, AssistantID: "test_assistant", StackID: "stack_001", StackDepth: 0, Type: types.ResumeTypeLLM, Status: types.ResumeStatusFailed, Sequence: 3, Error: "Connection timeout"},
		}

		err = store.SaveResume(records)
		require.NoError(t, err)
		t.Cleanup(func() { store.DeleteResume(batchChat.ChatID) })

		retrieved, err := store.GetResume(batchChat.ChatID)
		require.NoError(t, err)
		assert.Equal(t, 3, len(retrieved))

		if len(retrieved) >= 3 {
			assert.Equal(t, 1, retrieved[0].Sequence)
			assert.Equal(t, 3, retrieved[2].Sequence)
			assert.Equal(t, "Connection timeout", retrieved[2].Error)
		}
	})

	t.Run("SaveRecordWithAllFields", func(t *testing.T) {
		fullChat := &types.Chat{AssistantID: "test_assistant"}
		err := store.CreateChat(fullChat)
		require.NoError(t, err)
		t.Cleanup(func() { store.DeleteChat(fullChat.ChatID) })

		requestID := fmt.Sprintf("req_%d", time.Now().UnixNano())
		records := []*types.Resume{
			{
				ChatID:        fullChat.ChatID,
				RequestID:     requestID,
				AssistantID:   "test_assistant",
				StackID:       "stack_001",
				StackParentID: "stack_000",
				StackDepth:    1,
				Type:          types.ResumeTypeDelegate,
				Status:        types.ResumeStatusInterrupted,
				Input:         map[string]interface{}{"agent_id": "sub_agent", "messages": []interface{}{}},
				Output:        map[string]interface{}{"partial": true},
				SpaceSnapshot: map[string]interface{}{"key1": "value1", "key2": 123},
				Error:         "User cancelled",
				Sequence:      1,
				Metadata:      map[string]interface{}{"retry_count": 0},
			},
		}

		err = store.SaveResume(records)
		require.NoError(t, err)
		t.Cleanup(func() { store.DeleteResume(fullChat.ChatID) })

		retrieved, err := store.GetResume(fullChat.ChatID)
		require.NoError(t, err)
		require.Equal(t, 1, len(retrieved))

		r := retrieved[0]
		assert.Equal(t, "stack_000", r.StackParentID)
		assert.Equal(t, 1, r.StackDepth)
		assert.NotNil(t, r.Input)
		assert.NotNil(t, r.Output)
		require.NotNil(t, r.SpaceSnapshot)
		assert.Equal(t, "value1", r.SpaceSnapshot["key1"])
		assert.NotNil(t, r.Metadata)
	})

	t.Run("SaveEmptyRecords", func(t *testing.T) {
		err := store.SaveResume([]*types.Resume{})
		assert.NoError(t, err)
	})

	t.Run("SaveRecordWithoutChatID", func(t *testing.T) {
		records := []*types.Resume{{RequestID: "req", AssistantID: "ast", StackID: "stk", Type: "llm", Status: "failed", Sequence: 1}}
		err := store.SaveResume(records)
		assert.Error(t, err)
	})

	t.Run("SaveRecordWithoutRequestID", func(t *testing.T) {
		records := []*types.Resume{{ChatID: chat.ChatID, AssistantID: "ast", StackID: "stk", Type: "llm", Status: "failed", Sequence: 1}}
		err := store.SaveResume(records)
		assert.Error(t, err)
	})

	t.Run("SaveRecordWithoutAssistantID", func(t *testing.T) {
		records := []*types.Resume{{ChatID: chat.ChatID, RequestID: "req", StackID: "stk", Type: "llm", Status: "failed", Sequence: 1}}
		err := store.SaveResume(records)
		assert.Error(t, err)
	})

	t.Run("SaveRecordWithoutStackID", func(t *testing.T) {
		records := []*types.Resume{{ChatID: chat.ChatID, RequestID: "req", AssistantID: "ast", Type: "llm", Status: "failed", Sequence: 1}}
		err := store.SaveResume(records)
		assert.Error(t, err)
	})

	t.Run("SaveRecordWithoutType", func(t *testing.T) {
		records := []*types.Resume{{ChatID: chat.ChatID, RequestID: "req", AssistantID: "ast", StackID: "stk", Status: "failed", Sequence: 1}}
		err := store.SaveResume(records)
		assert.Error(t, err)
	})

	t.Run("SaveRecordWithoutStatus", func(t *testing.T) {
		records := []*types.Resume{{ChatID: chat.ChatID, RequestID: "req", AssistantID: "ast", StackID: "stk", Type: "llm", Sequence: 1}}
		err := store.SaveResume(records)
		assert.Error(t, err)
	})
}

func TestGetResume(t *testing.T) {
	testprepare.PrepareSandbox(t)

	store, err := xun.NewXun(types.Setting{Connector: "default"})
	require.NoError(t, err)

	chat := &types.Chat{AssistantID: "test_assistant"}
	err = store.CreateChat(chat)
	require.NoError(t, err)
	t.Cleanup(func() { store.DeleteChat(chat.ChatID) })

	requestID := fmt.Sprintf("req_%d", time.Now().UnixNano())
	records := []*types.Resume{
		{ChatID: chat.ChatID, RequestID: requestID, AssistantID: "ast1", StackID: "stk1", Type: types.ResumeTypeInput, Status: types.ResumeStatusInterrupted, Sequence: 1},
		{ChatID: chat.ChatID, RequestID: requestID, AssistantID: "ast1", StackID: "stk1", Type: types.ResumeTypeHookCreate, Status: types.ResumeStatusInterrupted, Sequence: 2},
		{ChatID: chat.ChatID, RequestID: requestID, AssistantID: "ast1", StackID: "stk1", Type: types.ResumeTypeLLM, Status: types.ResumeStatusFailed, Sequence: 3},
	}
	err = store.SaveResume(records)
	require.NoError(t, err)
	t.Cleanup(func() { store.DeleteResume(chat.ChatID) })

	t.Run("GetAllRecords", func(t *testing.T) {
		retrieved, err := store.GetResume(chat.ChatID)
		require.NoError(t, err)
		assert.Equal(t, 3, len(retrieved))

		for i := 1; i < len(retrieved); i++ {
			assert.GreaterOrEqual(t, retrieved[i].Sequence, retrieved[i-1].Sequence)
		}
	})

	t.Run("GetRecordsWithEmptyChatID", func(t *testing.T) {
		_, err := store.GetResume("")
		assert.Error(t, err)
	})

	t.Run("GetRecordsFromNonExistentChat", func(t *testing.T) {
		retrieved, err := store.GetResume("nonexistent_chat")
		require.NoError(t, err)
		assert.Equal(t, 0, len(retrieved))
	})
}

func TestGetLastResume(t *testing.T) {
	testprepare.PrepareSandbox(t)

	store, err := xun.NewXun(types.Setting{Connector: "default"})
	require.NoError(t, err)

	chat := &types.Chat{AssistantID: "test_assistant"}
	err = store.CreateChat(chat)
	require.NoError(t, err)
	t.Cleanup(func() { store.DeleteChat(chat.ChatID) })

	t.Run("GetLastRecordFromMultiple", func(t *testing.T) {
		requestID := fmt.Sprintf("req_%d", time.Now().UnixNano())
		records := []*types.Resume{
			{ChatID: chat.ChatID, RequestID: requestID, AssistantID: "ast", StackID: "stk", Type: types.ResumeTypeInput, Status: types.ResumeStatusInterrupted, Sequence: 1},
			{ChatID: chat.ChatID, RequestID: requestID, AssistantID: "ast", StackID: "stk", Type: types.ResumeTypeHookCreate, Status: types.ResumeStatusInterrupted, Sequence: 2},
			{ChatID: chat.ChatID, RequestID: requestID, AssistantID: "ast", StackID: "stk", Type: types.ResumeTypeLLM, Status: types.ResumeStatusFailed, Sequence: 3, Error: "Last error"},
		}
		err := store.SaveResume(records)
		require.NoError(t, err)
		t.Cleanup(func() { store.DeleteResume(chat.ChatID) })

		last, err := store.GetLastResume(chat.ChatID)
		require.NoError(t, err)
		require.NotNil(t, last)

		assert.Equal(t, 3, last.Sequence)
		assert.Equal(t, types.ResumeTypeLLM, last.Type)
		assert.Equal(t, "Last error", last.Error)
	})

	t.Run("GetLastRecordFromEmpty", func(t *testing.T) {
		emptyChat := &types.Chat{AssistantID: "test_assistant"}
		err := store.CreateChat(emptyChat)
		require.NoError(t, err)
		t.Cleanup(func() { store.DeleteChat(emptyChat.ChatID) })

		last, err := store.GetLastResume(emptyChat.ChatID)
		require.NoError(t, err)
		assert.Nil(t, last)
	})

	t.Run("GetLastRecordWithEmptyChatID", func(t *testing.T) {
		_, err := store.GetLastResume("")
		assert.Error(t, err)
	})
}

func TestGetResumeByStackID(t *testing.T) {
	testprepare.PrepareSandbox(t)

	store, err := xun.NewXun(types.Setting{Connector: "default"})
	require.NoError(t, err)

	chat := &types.Chat{AssistantID: "test_assistant"}
	err = store.CreateChat(chat)
	require.NoError(t, err)
	t.Cleanup(func() { store.DeleteChat(chat.ChatID) })

	requestID := fmt.Sprintf("req_%d", time.Now().UnixNano())
	records := []*types.Resume{
		{ChatID: chat.ChatID, RequestID: requestID, AssistantID: "ast1", StackID: "stack_A", Type: types.ResumeTypeInput, Status: types.ResumeStatusInterrupted, Sequence: 1},
		{ChatID: chat.ChatID, RequestID: requestID, AssistantID: "ast1", StackID: "stack_A", Type: types.ResumeTypeLLM, Status: types.ResumeStatusInterrupted, Sequence: 2},
		{ChatID: chat.ChatID, RequestID: requestID, AssistantID: "ast2", StackID: "stack_B", StackParentID: "stack_A", StackDepth: 1, Type: types.ResumeTypeDelegate, Status: types.ResumeStatusFailed, Sequence: 3},
	}
	err = store.SaveResume(records)
	require.NoError(t, err)
	t.Cleanup(func() { store.DeleteResume(chat.ChatID) })

	t.Run("GetRecordsByStackA", func(t *testing.T) {
		retrieved, err := store.GetResumeByStackID("stack_A")
		require.NoError(t, err)
		assert.Equal(t, 2, len(retrieved))
	})

	t.Run("GetRecordsByStackB", func(t *testing.T) {
		retrieved, err := store.GetResumeByStackID("stack_B")
		require.NoError(t, err)
		assert.Equal(t, 1, len(retrieved))

		if len(retrieved) > 0 {
			assert.Equal(t, "stack_A", retrieved[0].StackParentID)
			assert.Equal(t, 1, retrieved[0].StackDepth)
		}
	})

	t.Run("GetRecordsByNonExistentStack", func(t *testing.T) {
		retrieved, err := store.GetResumeByStackID("nonexistent_stack")
		require.NoError(t, err)
		assert.Equal(t, 0, len(retrieved))
	})

	t.Run("GetRecordsByEmptyStackID", func(t *testing.T) {
		_, err := store.GetResumeByStackID("")
		assert.Error(t, err)
	})
}

func TestGetStackPath(t *testing.T) {
	testprepare.PrepareSandbox(t)

	store, err := xun.NewXun(types.Setting{Connector: "default"})
	require.NoError(t, err)

	chat := &types.Chat{AssistantID: "test_assistant"}
	err = store.CreateChat(chat)
	require.NoError(t, err)
	t.Cleanup(func() { store.DeleteChat(chat.ChatID) })

	requestID := fmt.Sprintf("req_%d", time.Now().UnixNano())
	records := []*types.Resume{
		{ChatID: chat.ChatID, RequestID: requestID, AssistantID: "ast1", StackID: "root_stack", Type: types.ResumeTypeInput, Status: types.ResumeStatusInterrupted, Sequence: 1},
		{ChatID: chat.ChatID, RequestID: requestID, AssistantID: "ast2", StackID: "child_stack", StackParentID: "root_stack", StackDepth: 1, Type: types.ResumeTypeDelegate, Status: types.ResumeStatusInterrupted, Sequence: 2},
		{ChatID: chat.ChatID, RequestID: requestID, AssistantID: "ast3", StackID: "grandchild_stack", StackParentID: "child_stack", StackDepth: 2, Type: types.ResumeTypeLLM, Status: types.ResumeStatusFailed, Sequence: 3},
	}
	err = store.SaveResume(records)
	require.NoError(t, err)
	t.Cleanup(func() { store.DeleteResume(chat.ChatID) })

	t.Run("GetPathFromGrandchild", func(t *testing.T) {
		path, err := store.GetStackPath("grandchild_stack")
		require.NoError(t, err)
		assert.Equal(t, 3, len(path))

		if len(path) >= 3 {
			assert.Equal(t, "root_stack", path[0])
			assert.Equal(t, "child_stack", path[1])
			assert.Equal(t, "grandchild_stack", path[2])
		}
	})

	t.Run("GetPathFromChild", func(t *testing.T) {
		path, err := store.GetStackPath("child_stack")
		require.NoError(t, err)
		assert.Equal(t, 2, len(path))

		if len(path) >= 2 {
			assert.Equal(t, "root_stack", path[0])
			assert.Equal(t, "child_stack", path[1])
		}
	})

	t.Run("GetPathFromRoot", func(t *testing.T) {
		path, err := store.GetStackPath("root_stack")
		require.NoError(t, err)
		assert.Equal(t, 1, len(path))

		if len(path) >= 1 {
			assert.Equal(t, "root_stack", path[0])
		}
	})

	t.Run("GetPathWithEmptyStackID", func(t *testing.T) {
		_, err := store.GetStackPath("")
		assert.Error(t, err)
	})
}

func TestDeleteResume(t *testing.T) {
	testprepare.PrepareSandbox(t)

	store, err := xun.NewXun(types.Setting{Connector: "default"})
	require.NoError(t, err)

	t.Run("DeleteExistingRecords", func(t *testing.T) {
		chat := &types.Chat{AssistantID: "test_assistant"}
		err := store.CreateChat(chat)
		require.NoError(t, err)
		t.Cleanup(func() { store.DeleteChat(chat.ChatID) })

		requestID := fmt.Sprintf("req_%d", time.Now().UnixNano())
		records := []*types.Resume{
			{ChatID: chat.ChatID, RequestID: requestID, AssistantID: "ast", StackID: "stk", Type: types.ResumeTypeLLM, Status: types.ResumeStatusFailed, Sequence: 1},
			{ChatID: chat.ChatID, RequestID: requestID, AssistantID: "ast", StackID: "stk", Type: types.ResumeTypeTool, Status: types.ResumeStatusFailed, Sequence: 2},
		}
		err = store.SaveResume(records)
		require.NoError(t, err)

		err = store.DeleteResume(chat.ChatID)
		require.NoError(t, err)

		retrieved, err := store.GetResume(chat.ChatID)
		require.NoError(t, err)
		assert.Equal(t, 0, len(retrieved))
	})

	t.Run("DeleteFromEmptyChat", func(t *testing.T) {
		chat := &types.Chat{AssistantID: "test_assistant"}
		err := store.CreateChat(chat)
		require.NoError(t, err)
		t.Cleanup(func() { store.DeleteChat(chat.ChatID) })

		err = store.DeleteResume(chat.ChatID)
		assert.NoError(t, err)
	})

	t.Run("DeleteWithEmptyChatID", func(t *testing.T) {
		err := store.DeleteResume("")
		assert.Error(t, err)
	})
}

func TestResumeCompleteWorkflow(t *testing.T) {
	testprepare.PrepareSandbox(t)

	store, err := xun.NewXun(types.Setting{Connector: "default"})
	require.NoError(t, err)

	t.Run("CompleteA2AWorkflow", func(t *testing.T) {
		chat := &types.Chat{AssistantID: "main_assistant", Title: "A2A Workflow Test"}
		err := store.CreateChat(chat)
		require.NoError(t, err)
		t.Cleanup(func() { store.DeleteChat(chat.ChatID) })

		requestID := fmt.Sprintf("req_%d", time.Now().UnixNano())
		records := []*types.Resume{
			{
				ChatID: chat.ChatID, RequestID: requestID, AssistantID: "main_assistant",
				StackID: "main_stack", StackDepth: 0,
				Type: types.ResumeTypeInput, Status: types.ResumeStatusInterrupted,
				Input:    map[string]interface{}{"messages": []interface{}{map[string]interface{}{"role": "user", "content": "Analyze this"}}},
				Sequence: 1,
			},
			{
				ChatID: chat.ChatID, RequestID: requestID, AssistantID: "main_assistant",
				StackID: "main_stack", StackDepth: 0,
				Type: types.ResumeTypeDelegate, Status: types.ResumeStatusInterrupted,
				SpaceSnapshot: map[string]interface{}{"task": "analyze", "data_id": "123"},
				Sequence:      2,
			},
			{
				ChatID: chat.ChatID, RequestID: requestID, AssistantID: "sub_assistant",
				StackID: "sub_stack", StackParentID: "main_stack", StackDepth: 1,
				Type: types.ResumeTypeInput, Status: types.ResumeStatusInterrupted,
				Sequence: 3,
			},
			{
				ChatID: chat.ChatID, RequestID: requestID, AssistantID: "sub_assistant",
				StackID: "sub_stack", StackParentID: "main_stack", StackDepth: 1,
				Type: types.ResumeTypeLLM, Status: types.ResumeStatusInterrupted,
				Input:         map[string]interface{}{"messages": []interface{}{}},
				Output:        map[string]interface{}{"partial_content": "The analysis shows..."},
				SpaceSnapshot: map[string]interface{}{"task": "analyze", "data_id": "123"},
				Sequence:      4,
			},
		}

		err = store.SaveResume(records)
		require.NoError(t, err)

		last, err := store.GetLastResume(chat.ChatID)
		require.NoError(t, err)
		require.NotNil(t, last)
		assert.Equal(t, types.ResumeTypeLLM, last.Type)
		assert.Equal(t, 1, last.StackDepth)

		path, err := store.GetStackPath(last.StackID)
		require.NoError(t, err)
		assert.Equal(t, 2, len(path))

		subRecords, err := store.GetResumeByStackID("sub_stack")
		require.NoError(t, err)
		assert.Equal(t, 2, len(subRecords))

		require.NotNil(t, last.SpaceSnapshot)
		assert.Equal(t, "analyze", last.SpaceSnapshot["task"])

		err = store.DeleteResume(chat.ChatID)
		require.NoError(t, err)

		remaining, err := store.GetResume(chat.ChatID)
		require.NoError(t, err)
		assert.Equal(t, 0, len(remaining))
	})
}
