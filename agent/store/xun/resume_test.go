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

// TestSaveResume tests batch saving resume records
func TestSaveResume(t *testing.T) {
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
		Title:       "Resume Test Chat",
	}
	err = store.CreateChat(chat)
	if err != nil {
		t.Fatalf("Failed to create chat: %v", err)
	}
	defer store.DeleteChat(chat.ChatID)

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
		if err != nil {
			t.Fatalf("Failed to save resume record: %v", err)
		}

		// Verify
		retrieved, err := store.GetResume(chat.ChatID)
		if err != nil {
			t.Fatalf("Failed to get resume records: %v", err)
		}

		found := false
		for _, r := range retrieved {
			if r.RequestID == requestID {
				found = true
				if r.Type != types.ResumeTypeLLM {
					t.Errorf("Expected type '%s', got '%s'", types.ResumeTypeLLM, r.Type)
				}
				if r.Status != types.ResumeStatusInterrupted {
					t.Errorf("Expected status '%s', got '%s'", types.ResumeStatusInterrupted, r.Status)
				}
				break
			}
		}

		if !found {
			t.Error("Could not find saved resume record")
		}

		// Clean up
		store.DeleteResume(chat.ChatID)
	})

	t.Run("SaveBatchRecords", func(t *testing.T) {
		// Create a new chat for this test
		batchChat := &types.Chat{
			AssistantID: "test_assistant",
		}
		err := store.CreateChat(batchChat)
		if err != nil {
			t.Fatalf("Failed to create chat: %v", err)
		}
		defer store.DeleteChat(batchChat.ChatID)

		requestID := fmt.Sprintf("req_%d", time.Now().UnixNano())
		records := []*types.Resume{
			{
				ChatID:      batchChat.ChatID,
				RequestID:   requestID,
				AssistantID: "test_assistant",
				StackID:     "stack_001",
				StackDepth:  0,
				Type:        types.ResumeTypeInput,
				Status:      types.ResumeStatusInterrupted,
				Sequence:    1,
			},
			{
				ChatID:      batchChat.ChatID,
				RequestID:   requestID,
				AssistantID: "test_assistant",
				StackID:     "stack_001",
				StackDepth:  0,
				Type:        types.ResumeTypeHookCreate,
				Status:      types.ResumeStatusInterrupted,
				Sequence:    2,
			},
			{
				ChatID:      batchChat.ChatID,
				RequestID:   requestID,
				AssistantID: "test_assistant",
				StackID:     "stack_001",
				StackDepth:  0,
				Type:        types.ResumeTypeLLM,
				Status:      types.ResumeStatusFailed,
				Sequence:    3,
				Error:       "Connection timeout",
			},
		}

		err = store.SaveResume(records)
		if err != nil {
			t.Fatalf("Failed to save batch resume records: %v", err)
		}

		// Verify all records saved
		retrieved, err := store.GetResume(batchChat.ChatID)
		if err != nil {
			t.Fatalf("Failed to get resume records: %v", err)
		}

		if len(retrieved) != 3 {
			t.Errorf("Expected 3 records, got %d", len(retrieved))
		}

		// Verify order (should be by sequence)
		if len(retrieved) >= 3 {
			if retrieved[0].Sequence != 1 {
				t.Errorf("Expected first record sequence 1, got %d", retrieved[0].Sequence)
			}
			if retrieved[2].Sequence != 3 {
				t.Errorf("Expected last record sequence 3, got %d", retrieved[2].Sequence)
			}
			if retrieved[2].Error != "Connection timeout" {
				t.Errorf("Expected error 'Connection timeout', got '%s'", retrieved[2].Error)
			}
		}

		t.Logf("Saved %d resume records in single batch call", len(records))
	})

	t.Run("SaveRecordWithAllFields", func(t *testing.T) {
		fullChat := &types.Chat{
			AssistantID: "test_assistant",
		}
		err := store.CreateChat(fullChat)
		if err != nil {
			t.Fatalf("Failed to create chat: %v", err)
		}
		defer store.DeleteChat(fullChat.ChatID)

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
		if err != nil {
			t.Fatalf("Failed to save record: %v", err)
		}

		retrieved, err := store.GetResume(fullChat.ChatID)
		if err != nil {
			t.Fatalf("Failed to get records: %v", err)
		}

		if len(retrieved) != 1 {
			t.Fatalf("Expected 1 record, got %d", len(retrieved))
		}

		r := retrieved[0]
		if r.StackParentID != "stack_000" {
			t.Errorf("Expected stack_parent_id 'stack_000', got '%s'", r.StackParentID)
		}
		if r.StackDepth != 1 {
			t.Errorf("Expected stack_depth 1, got %d", r.StackDepth)
		}
		if r.Input == nil {
			t.Error("Expected input to be set")
		}
		if r.Output == nil {
			t.Error("Expected output to be set")
		}
		if r.SpaceSnapshot == nil {
			t.Error("Expected space_snapshot to be set")
		} else if r.SpaceSnapshot["key1"] != "value1" {
			t.Errorf("Expected space_snapshot key1='value1', got '%v'", r.SpaceSnapshot["key1"])
		}
		if r.Metadata == nil {
			t.Error("Expected metadata to be set")
		}
	})

	t.Run("SaveEmptyRecords", func(t *testing.T) {
		err := store.SaveResume([]*types.Resume{})
		if err != nil {
			t.Errorf("Expected no error for empty records, got: %v", err)
		}
	})

	t.Run("SaveRecordWithoutChatID", func(t *testing.T) {
		records := []*types.Resume{{RequestID: "req", AssistantID: "ast", StackID: "stk", Type: "llm", Status: "failed", Sequence: 1}}
		err := store.SaveResume(records)
		if err == nil {
			t.Error("Expected error when saving without chat_id")
		}
	})

	t.Run("SaveRecordWithoutRequestID", func(t *testing.T) {
		records := []*types.Resume{{ChatID: chat.ChatID, AssistantID: "ast", StackID: "stk", Type: "llm", Status: "failed", Sequence: 1}}
		err := store.SaveResume(records)
		if err == nil {
			t.Error("Expected error when saving without request_id")
		}
	})

	t.Run("SaveRecordWithoutAssistantID", func(t *testing.T) {
		records := []*types.Resume{{ChatID: chat.ChatID, RequestID: "req", StackID: "stk", Type: "llm", Status: "failed", Sequence: 1}}
		err := store.SaveResume(records)
		if err == nil {
			t.Error("Expected error when saving without assistant_id")
		}
	})

	t.Run("SaveRecordWithoutStackID", func(t *testing.T) {
		records := []*types.Resume{{ChatID: chat.ChatID, RequestID: "req", AssistantID: "ast", Type: "llm", Status: "failed", Sequence: 1}}
		err := store.SaveResume(records)
		if err == nil {
			t.Error("Expected error when saving without stack_id")
		}
	})

	t.Run("SaveRecordWithoutType", func(t *testing.T) {
		records := []*types.Resume{{ChatID: chat.ChatID, RequestID: "req", AssistantID: "ast", StackID: "stk", Status: "failed", Sequence: 1}}
		err := store.SaveResume(records)
		if err == nil {
			t.Error("Expected error when saving without type")
		}
	})

	t.Run("SaveRecordWithoutStatus", func(t *testing.T) {
		records := []*types.Resume{{ChatID: chat.ChatID, RequestID: "req", AssistantID: "ast", StackID: "stk", Type: "llm", Sequence: 1}}
		err := store.SaveResume(records)
		if err == nil {
			t.Error("Expected error when saving without status")
		}
	})
}

// TestGetResume tests retrieving resume records
func TestGetResume(t *testing.T) {
	test.Prepare(t, config.Conf)
	defer test.Clean()

	store, err := xun.NewXun(types.Setting{
		Connector: "default",
	})
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}

	// Create chat and resume records
	chat := &types.Chat{
		AssistantID: "test_assistant",
	}
	err = store.CreateChat(chat)
	if err != nil {
		t.Fatalf("Failed to create chat: %v", err)
	}
	defer store.DeleteChat(chat.ChatID)

	requestID := fmt.Sprintf("req_%d", time.Now().UnixNano())
	records := []*types.Resume{
		{ChatID: chat.ChatID, RequestID: requestID, AssistantID: "ast1", StackID: "stk1", Type: types.ResumeTypeInput, Status: types.ResumeStatusInterrupted, Sequence: 1},
		{ChatID: chat.ChatID, RequestID: requestID, AssistantID: "ast1", StackID: "stk1", Type: types.ResumeTypeHookCreate, Status: types.ResumeStatusInterrupted, Sequence: 2},
		{ChatID: chat.ChatID, RequestID: requestID, AssistantID: "ast1", StackID: "stk1", Type: types.ResumeTypeLLM, Status: types.ResumeStatusFailed, Sequence: 3},
	}
	err = store.SaveResume(records)
	if err != nil {
		t.Fatalf("Failed to save records: %v", err)
	}
	defer store.DeleteResume(chat.ChatID)

	t.Run("GetAllRecords", func(t *testing.T) {
		retrieved, err := store.GetResume(chat.ChatID)
		if err != nil {
			t.Fatalf("Failed to get records: %v", err)
		}

		if len(retrieved) != 3 {
			t.Errorf("Expected 3 records, got %d", len(retrieved))
		}

		// Verify order by sequence
		for i := 1; i < len(retrieved); i++ {
			if retrieved[i].Sequence < retrieved[i-1].Sequence {
				t.Error("Records not ordered by sequence")
			}
		}
	})

	t.Run("GetRecordsWithEmptyChatID", func(t *testing.T) {
		_, err := store.GetResume("")
		if err == nil {
			t.Error("Expected error when getting records without chat_id")
		}
	})

	t.Run("GetRecordsFromNonExistentChat", func(t *testing.T) {
		retrieved, err := store.GetResume("nonexistent_chat")
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}
		if len(retrieved) != 0 {
			t.Errorf("Expected 0 records from non-existent chat, got %d", len(retrieved))
		}
	})
}

// TestGetLastResume tests retrieving the last resume record
func TestGetLastResume(t *testing.T) {
	test.Prepare(t, config.Conf)
	defer test.Clean()

	store, err := xun.NewXun(types.Setting{
		Connector: "default",
	})
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}

	chat := &types.Chat{
		AssistantID: "test_assistant",
	}
	err = store.CreateChat(chat)
	if err != nil {
		t.Fatalf("Failed to create chat: %v", err)
	}
	defer store.DeleteChat(chat.ChatID)

	t.Run("GetLastRecordFromMultiple", func(t *testing.T) {
		requestID := fmt.Sprintf("req_%d", time.Now().UnixNano())
		records := []*types.Resume{
			{ChatID: chat.ChatID, RequestID: requestID, AssistantID: "ast", StackID: "stk", Type: types.ResumeTypeInput, Status: types.ResumeStatusInterrupted, Sequence: 1},
			{ChatID: chat.ChatID, RequestID: requestID, AssistantID: "ast", StackID: "stk", Type: types.ResumeTypeHookCreate, Status: types.ResumeStatusInterrupted, Sequence: 2},
			{ChatID: chat.ChatID, RequestID: requestID, AssistantID: "ast", StackID: "stk", Type: types.ResumeTypeLLM, Status: types.ResumeStatusFailed, Sequence: 3, Error: "Last error"},
		}
		err := store.SaveResume(records)
		if err != nil {
			t.Fatalf("Failed to save records: %v", err)
		}
		defer store.DeleteResume(chat.ChatID)

		last, err := store.GetLastResume(chat.ChatID)
		if err != nil {
			t.Fatalf("Failed to get last record: %v", err)
		}

		if last == nil {
			t.Fatal("Expected last record, got nil")
		}

		if last.Sequence != 3 {
			t.Errorf("Expected sequence 3, got %d", last.Sequence)
		}
		if last.Type != types.ResumeTypeLLM {
			t.Errorf("Expected type '%s', got '%s'", types.ResumeTypeLLM, last.Type)
		}
		if last.Error != "Last error" {
			t.Errorf("Expected error 'Last error', got '%s'", last.Error)
		}
	})

	t.Run("GetLastRecordFromEmpty", func(t *testing.T) {
		emptyChat := &types.Chat{AssistantID: "test_assistant"}
		err := store.CreateChat(emptyChat)
		if err != nil {
			t.Fatalf("Failed to create chat: %v", err)
		}
		defer store.DeleteChat(emptyChat.ChatID)

		last, err := store.GetLastResume(emptyChat.ChatID)
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}

		if last != nil {
			t.Error("Expected nil for empty chat, got record")
		}
	})

	t.Run("GetLastRecordWithEmptyChatID", func(t *testing.T) {
		_, err := store.GetLastResume("")
		if err == nil {
			t.Error("Expected error when getting last record without chat_id")
		}
	})
}

// TestGetResumeByStackID tests retrieving records by stack ID
func TestGetResumeByStackID(t *testing.T) {
	test.Prepare(t, config.Conf)
	defer test.Clean()

	store, err := xun.NewXun(types.Setting{
		Connector: "default",
	})
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}

	chat := &types.Chat{
		AssistantID: "test_assistant",
	}
	err = store.CreateChat(chat)
	if err != nil {
		t.Fatalf("Failed to create chat: %v", err)
	}
	defer store.DeleteChat(chat.ChatID)

	requestID := fmt.Sprintf("req_%d", time.Now().UnixNano())
	records := []*types.Resume{
		{ChatID: chat.ChatID, RequestID: requestID, AssistantID: "ast1", StackID: "stack_A", Type: types.ResumeTypeInput, Status: types.ResumeStatusInterrupted, Sequence: 1},
		{ChatID: chat.ChatID, RequestID: requestID, AssistantID: "ast1", StackID: "stack_A", Type: types.ResumeTypeLLM, Status: types.ResumeStatusInterrupted, Sequence: 2},
		{ChatID: chat.ChatID, RequestID: requestID, AssistantID: "ast2", StackID: "stack_B", StackParentID: "stack_A", StackDepth: 1, Type: types.ResumeTypeDelegate, Status: types.ResumeStatusFailed, Sequence: 3},
	}
	err = store.SaveResume(records)
	if err != nil {
		t.Fatalf("Failed to save records: %v", err)
	}
	defer store.DeleteResume(chat.ChatID)

	t.Run("GetRecordsByStackA", func(t *testing.T) {
		retrieved, err := store.GetResumeByStackID("stack_A")
		if err != nil {
			t.Fatalf("Failed to get records: %v", err)
		}

		if len(retrieved) != 2 {
			t.Errorf("Expected 2 records for stack_A, got %d", len(retrieved))
		}
	})

	t.Run("GetRecordsByStackB", func(t *testing.T) {
		retrieved, err := store.GetResumeByStackID("stack_B")
		if err != nil {
			t.Fatalf("Failed to get records: %v", err)
		}

		if len(retrieved) != 1 {
			t.Errorf("Expected 1 record for stack_B, got %d", len(retrieved))
		}

		if len(retrieved) > 0 {
			if retrieved[0].StackParentID != "stack_A" {
				t.Errorf("Expected stack_parent_id 'stack_A', got '%s'", retrieved[0].StackParentID)
			}
			if retrieved[0].StackDepth != 1 {
				t.Errorf("Expected stack_depth 1, got %d", retrieved[0].StackDepth)
			}
		}
	})

	t.Run("GetRecordsByNonExistentStack", func(t *testing.T) {
		retrieved, err := store.GetResumeByStackID("nonexistent_stack")
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}
		if len(retrieved) != 0 {
			t.Errorf("Expected 0 records, got %d", len(retrieved))
		}
	})

	t.Run("GetRecordsByEmptyStackID", func(t *testing.T) {
		_, err := store.GetResumeByStackID("")
		if err == nil {
			t.Error("Expected error when getting records without stack_id")
		}
	})
}

// TestGetStackPath tests retrieving the stack path
func TestGetStackPath(t *testing.T) {
	test.Prepare(t, config.Conf)
	defer test.Clean()

	store, err := xun.NewXun(types.Setting{
		Connector: "default",
	})
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}

	chat := &types.Chat{
		AssistantID: "test_assistant",
	}
	err = store.CreateChat(chat)
	if err != nil {
		t.Fatalf("Failed to create chat: %v", err)
	}
	defer store.DeleteChat(chat.ChatID)

	// Create a nested stack structure: root -> child -> grandchild
	requestID := fmt.Sprintf("req_%d", time.Now().UnixNano())
	records := []*types.Resume{
		{ChatID: chat.ChatID, RequestID: requestID, AssistantID: "ast1", StackID: "root_stack", Type: types.ResumeTypeInput, Status: types.ResumeStatusInterrupted, Sequence: 1},
		{ChatID: chat.ChatID, RequestID: requestID, AssistantID: "ast2", StackID: "child_stack", StackParentID: "root_stack", StackDepth: 1, Type: types.ResumeTypeDelegate, Status: types.ResumeStatusInterrupted, Sequence: 2},
		{ChatID: chat.ChatID, RequestID: requestID, AssistantID: "ast3", StackID: "grandchild_stack", StackParentID: "child_stack", StackDepth: 2, Type: types.ResumeTypeLLM, Status: types.ResumeStatusFailed, Sequence: 3},
	}
	err = store.SaveResume(records)
	if err != nil {
		t.Fatalf("Failed to save records: %v", err)
	}
	defer store.DeleteResume(chat.ChatID)

	t.Run("GetPathFromGrandchild", func(t *testing.T) {
		path, err := store.GetStackPath("grandchild_stack")
		if err != nil {
			t.Fatalf("Failed to get stack path: %v", err)
		}

		if len(path) != 3 {
			t.Errorf("Expected path length 3, got %d", len(path))
		}

		if len(path) >= 3 {
			if path[0] != "root_stack" {
				t.Errorf("Expected first element 'root_stack', got '%s'", path[0])
			}
			if path[1] != "child_stack" {
				t.Errorf("Expected second element 'child_stack', got '%s'", path[1])
			}
			if path[2] != "grandchild_stack" {
				t.Errorf("Expected third element 'grandchild_stack', got '%s'", path[2])
			}
		}

		t.Logf("Stack path: %v", path)
	})

	t.Run("GetPathFromChild", func(t *testing.T) {
		path, err := store.GetStackPath("child_stack")
		if err != nil {
			t.Fatalf("Failed to get stack path: %v", err)
		}

		if len(path) != 2 {
			t.Errorf("Expected path length 2, got %d", len(path))
		}

		if len(path) >= 2 {
			if path[0] != "root_stack" {
				t.Errorf("Expected first element 'root_stack', got '%s'", path[0])
			}
			if path[1] != "child_stack" {
				t.Errorf("Expected second element 'child_stack', got '%s'", path[1])
			}
		}
	})

	t.Run("GetPathFromRoot", func(t *testing.T) {
		path, err := store.GetStackPath("root_stack")
		if err != nil {
			t.Fatalf("Failed to get stack path: %v", err)
		}

		if len(path) != 1 {
			t.Errorf("Expected path length 1, got %d", len(path))
		}

		if len(path) >= 1 && path[0] != "root_stack" {
			t.Errorf("Expected 'root_stack', got '%s'", path[0])
		}
	})

	t.Run("GetPathWithEmptyStackID", func(t *testing.T) {
		_, err := store.GetStackPath("")
		if err == nil {
			t.Error("Expected error when getting path without stack_id")
		}
	})
}

// TestDeleteResume tests deleting resume records
func TestDeleteResume(t *testing.T) {
	test.Prepare(t, config.Conf)
	defer test.Clean()

	store, err := xun.NewXun(types.Setting{
		Connector: "default",
	})
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}

	t.Run("DeleteExistingRecords", func(t *testing.T) {
		chat := &types.Chat{AssistantID: "test_assistant"}
		err := store.CreateChat(chat)
		if err != nil {
			t.Fatalf("Failed to create chat: %v", err)
		}
		defer store.DeleteChat(chat.ChatID)

		requestID := fmt.Sprintf("req_%d", time.Now().UnixNano())
		records := []*types.Resume{
			{ChatID: chat.ChatID, RequestID: requestID, AssistantID: "ast", StackID: "stk", Type: types.ResumeTypeLLM, Status: types.ResumeStatusFailed, Sequence: 1},
			{ChatID: chat.ChatID, RequestID: requestID, AssistantID: "ast", StackID: "stk", Type: types.ResumeTypeTool, Status: types.ResumeStatusFailed, Sequence: 2},
		}
		err = store.SaveResume(records)
		if err != nil {
			t.Fatalf("Failed to save records: %v", err)
		}

		// Delete
		err = store.DeleteResume(chat.ChatID)
		if err != nil {
			t.Fatalf("Failed to delete records: %v", err)
		}

		// Verify deleted
		retrieved, err := store.GetResume(chat.ChatID)
		if err != nil {
			t.Fatalf("Failed to get records: %v", err)
		}

		if len(retrieved) != 0 {
			t.Errorf("Expected 0 records after delete, got %d", len(retrieved))
		}
	})

	t.Run("DeleteFromEmptyChat", func(t *testing.T) {
		chat := &types.Chat{AssistantID: "test_assistant"}
		err := store.CreateChat(chat)
		if err != nil {
			t.Fatalf("Failed to create chat: %v", err)
		}
		defer store.DeleteChat(chat.ChatID)

		// Delete from chat with no records - should not error
		err = store.DeleteResume(chat.ChatID)
		if err != nil {
			t.Errorf("Expected no error when deleting from empty chat, got: %v", err)
		}
	})

	t.Run("DeleteWithEmptyChatID", func(t *testing.T) {
		err := store.DeleteResume("")
		if err == nil {
			t.Error("Expected error when deleting with empty chat_id")
		}
	})
}

// TestResumeCompleteWorkflow tests a complete resume/retry workflow
func TestResumeCompleteWorkflow(t *testing.T) {
	test.Prepare(t, config.Conf)
	defer test.Clean()

	store, err := xun.NewXun(types.Setting{
		Connector: "default",
	})
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}

	t.Run("CompleteA2AWorkflow", func(t *testing.T) {
		// Create chat
		chat := &types.Chat{
			AssistantID: "main_assistant",
			Title:       "A2A Workflow Test",
		}
		err := store.CreateChat(chat)
		if err != nil {
			t.Fatalf("Failed to create chat: %v", err)
		}
		defer store.DeleteChat(chat.ChatID)

		// Simulate A2A call that gets interrupted
		// Main assistant -> Sub assistant (interrupted during LLM call)
		requestID := fmt.Sprintf("req_%d", time.Now().UnixNano())
		records := []*types.Resume{
			// Main assistant steps
			{
				ChatID:      chat.ChatID,
				RequestID:   requestID,
				AssistantID: "main_assistant",
				StackID:     "main_stack",
				StackDepth:  0,
				Type:        types.ResumeTypeInput,
				Status:      types.ResumeStatusInterrupted,
				Input:       map[string]interface{}{"messages": []interface{}{map[string]interface{}{"role": "user", "content": "Analyze this"}}},
				Sequence:    1,
			},
			{
				ChatID:        chat.ChatID,
				RequestID:     requestID,
				AssistantID:   "main_assistant",
				StackID:       "main_stack",
				StackDepth:    0,
				Type:          types.ResumeTypeDelegate,
				Status:        types.ResumeStatusInterrupted,
				SpaceSnapshot: map[string]interface{}{"task": "analyze", "data_id": "123"},
				Sequence:      2,
			},
			// Sub assistant steps
			{
				ChatID:        chat.ChatID,
				RequestID:     requestID,
				AssistantID:   "sub_assistant",
				StackID:       "sub_stack",
				StackParentID: "main_stack",
				StackDepth:    1,
				Type:          types.ResumeTypeInput,
				Status:        types.ResumeStatusInterrupted,
				Sequence:      3,
			},
			{
				ChatID:        chat.ChatID,
				RequestID:     requestID,
				AssistantID:   "sub_assistant",
				StackID:       "sub_stack",
				StackParentID: "main_stack",
				StackDepth:    1,
				Type:          types.ResumeTypeLLM,
				Status:        types.ResumeStatusInterrupted,
				Input:         map[string]interface{}{"messages": []interface{}{}},
				Output:        map[string]interface{}{"partial_content": "The analysis shows..."},
				SpaceSnapshot: map[string]interface{}{"task": "analyze", "data_id": "123"},
				Sequence:      4,
			},
		}

		err = store.SaveResume(records)
		if err != nil {
			t.Fatalf("Failed to save resume records: %v", err)
		}
		t.Logf("Saved %d resume records for A2A workflow", len(records))

		// 1. Get last resume record (should be the interrupted LLM call)
		last, err := store.GetLastResume(chat.ChatID)
		if err != nil {
			t.Fatalf("Failed to get last resume: %v", err)
		}

		if last == nil {
			t.Fatal("Expected last resume record")
		}

		if last.Type != types.ResumeTypeLLM {
			t.Errorf("Expected type '%s', got '%s'", types.ResumeTypeLLM, last.Type)
		}
		if last.StackDepth != 1 {
			t.Errorf("Expected stack_depth 1, got %d", last.StackDepth)
		}

		// 2. Get stack path to understand the call hierarchy
		path, err := store.GetStackPath(last.StackID)
		if err != nil {
			t.Fatalf("Failed to get stack path: %v", err)
		}

		if len(path) != 2 {
			t.Errorf("Expected path length 2, got %d", len(path))
		}
		t.Logf("Stack path: %v", path)

		// 3. Get all records for the sub stack
		subRecords, err := store.GetResumeByStackID("sub_stack")
		if err != nil {
			t.Fatalf("Failed to get sub stack records: %v", err)
		}

		if len(subRecords) != 2 {
			t.Errorf("Expected 2 records for sub_stack, got %d", len(subRecords))
		}

		// 4. Verify space snapshot is preserved
		if last.SpaceSnapshot == nil {
			t.Error("Expected space_snapshot to be set")
		} else {
			if last.SpaceSnapshot["task"] != "analyze" {
				t.Errorf("Expected task='analyze', got '%v'", last.SpaceSnapshot["task"])
			}
		}

		// 5. Clean up after successful resume
		err = store.DeleteResume(chat.ChatID)
		if err != nil {
			t.Fatalf("Failed to delete resume records: %v", err)
		}

		// 6. Verify cleanup
		remaining, err := store.GetResume(chat.ChatID)
		if err != nil {
			t.Fatalf("Failed to get remaining records: %v", err)
		}

		if len(remaining) != 0 {
			t.Errorf("Expected 0 records after cleanup, got %d", len(remaining))
		}

		t.Log("Complete A2A workflow test passed!")
	})
}
