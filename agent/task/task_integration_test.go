//go:build integration

package task_test

import (
	"context"
	"testing"

	"github.com/yaoapp/gou/process"
	"github.com/yaoapp/yao/agent/board"
	"github.com/yaoapp/yao/agent/task"
	"github.com/yaoapp/yao/unit-test/agent/testprepare"
)

func TestTaskCRUD(t *testing.T) {
	identity := testprepare.PrepareSandbox(t)

	ctx := context.Background()
	auth := &process.AuthorizedInfo{
		UserID: identity.AlphaOwnerUserID,
		TeamID: identity.AlphaTeamID,
	}

	// Create a real board+column
	b, err := board.Create(ctx, auth, &board.CreateReq{
		Name: "CRUD Test Board", Icon: "material-test", Color: "#3B82F6",
	})
	if err != nil {
		t.Fatalf("board.Create failed: %v", err)
	}
	colID := b.Columns[0].ColumnID

	// Test Create
	created, err := task.Create(ctx, auth, &task.CreateReq{
		Title:       "Integration Test Task",
		AssistantID: "asst-test-001",
		ColumnID:    colID,
	})
	if err != nil {
		t.Fatalf("task.Create failed: %v", err)
	}
	if created.ChatID == "" {
		t.Fatal("expected non-empty chat_id")
	}
	if created.Title != "Integration Test Task" {
		t.Errorf("unexpected title: %s", created.Title)
	}
	if created.RunStatus != "pending" {
		t.Errorf("expected pending status, got %s", created.RunStatus)
	}

	// Test Get
	got, err := task.Get(ctx, auth, created.ChatID)
	if err != nil {
		t.Fatalf("task.Get failed: %v", err)
	}
	if got.ChatID != created.ChatID {
		t.Errorf("Get returned wrong task")
	}

	// Test List
	result, err := task.List(ctx, auth, &task.ListQuery{PageSize: 10})
	if err != nil {
		t.Fatalf("task.List failed: %v", err)
	}
	if result.Total == 0 {
		t.Error("expected at least 1 task in list")
	}

	// Test Update
	newTitle := "Updated Title"
	updated, err := task.Update(ctx, auth, created.ChatID, &task.UpdateReq{
		Title: &newTitle,
	})
	if err != nil {
		t.Fatalf("task.Update failed: %v", err)
	}
	if updated.Title != "Updated Title" {
		t.Errorf("title not updated, got: %s", updated.Title)
	}

	// Test Delete
	err = task.Delete(ctx, auth, created.ChatID)
	if err != nil {
		t.Fatalf("task.Delete failed: %v", err)
	}

	// Verify deleted (soft-delete)
	// After delete, List should not include the task
	listResult, err := task.List(ctx, auth, &task.ListQuery{PageSize: 100})
	if err != nil {
		t.Fatalf("task.List after delete: %v", err)
	}
	for _, item := range listResult.Tasks {
		if item.ChatID == created.ChatID {
			t.Error("deleted task should not appear in list")
		}
	}
}
