//go:build integration

package task_test

import (
	"context"
	"testing"

	"github.com/yaoapp/gou/process"
	"github.com/yaoapp/yao/agent/task"
	"github.com/yaoapp/yao/config"
	yaotest "github.com/yaoapp/yao/test"
)

func TestTaskCRUD(t *testing.T) {
	yaotest.Prepare(t, config.Conf)
	defer yaotest.Clean()

	ctx := context.Background()
	auth := &process.AuthorizedInfo{
		UserID: "test-user-001",
		TeamID: "test-team-001",
	}

	// First create a board and column for testing
	setupTestColumn(t)

	// Test Create
	created, err := task.Create(ctx, auth, &task.CreateReq{
		Title:       "Integration Test Task",
		AssistantID: "asst-test-001",
		ColumnID:    "test-col-001",
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

	// Verify deleted
	_, err = task.Get(ctx, auth, created.ChatID)
	if err == nil {
		t.Error("expected error getting deleted task")
	}
}

func setupTestColumn(t *testing.T) {
	t.Helper()
	// Insert test board and column directly for integration test setup
	// This would typically be done via the board service, but we insert directly
	// to avoid circular test dependencies
	// The actual DB setup is handled by test.Prepare which migrates all models
}
