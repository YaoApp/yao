//go:build integration

package board_test

import (
	"context"
	"testing"

	"github.com/yaoapp/gou/process"
	"github.com/yaoapp/yao/agent/board"
	"github.com/yaoapp/yao/config"
	yaotest "github.com/yaoapp/yao/test"
)

func TestBoardCRUD(t *testing.T) {
	yaotest.Prepare(t, config.Conf)
	defer yaotest.Clean()

	ctx := context.Background()
	auth := &process.AuthorizedInfo{
		UserID: "test-user-001",
		TeamID: "test-team-001",
	}

	// Test Create
	created, err := board.Create(ctx, auth, &board.CreateReq{
		Name:  "Test Board",
		Icon:  "material-test",
		Color: "#3B82F6",
	})
	if err != nil {
		t.Fatalf("board.Create failed: %v", err)
	}
	if created.BoardID == "" {
		t.Fatal("expected non-empty board_id")
	}
	if created.Name != "Test Board" {
		t.Errorf("unexpected name: %s", created.Name)
	}
	// Should have 1 default column
	if len(created.Columns) != 1 {
		t.Errorf("expected 1 default column, got %d", len(created.Columns))
	}
	if created.Columns[0].Name != "待处理" {
		t.Errorf("unexpected default column name: %s", created.Columns[0].Name)
	}

	// Test Get
	got, err := board.Get(ctx, auth, created.BoardID)
	if err != nil {
		t.Fatalf("board.Get failed: %v", err)
	}
	if got.BoardID != created.BoardID {
		t.Error("Get returned wrong board")
	}

	// Test List
	result, err := board.List(ctx, auth, &board.ListQuery{})
	if err != nil {
		t.Fatalf("board.List failed: %v", err)
	}
	if len(result.Boards) == 0 {
		t.Error("expected at least 1 board")
	}

	// Test CreateColumn
	col, err := board.CreateColumn(ctx, auth, created.BoardID, &board.ColumnReq{
		Name: "进行中",
		Icon: "material-play",
	})
	if err != nil {
		t.Fatalf("board.CreateColumn failed: %v", err)
	}
	if col.Name != "进行中" {
		t.Errorf("unexpected column name: %s", col.Name)
	}

	// Test Update
	newName := "Updated Board"
	updated, err := board.Update(ctx, auth, created.BoardID, &board.UpdateReq{
		Name: &newName,
	})
	if err != nil {
		t.Fatalf("board.Update failed: %v", err)
	}
	if updated.Name != "Updated Board" {
		t.Errorf("board name not updated")
	}

	// Test Delete
	err = board.Delete(ctx, auth, created.BoardID)
	if err != nil {
		t.Fatalf("board.Delete failed: %v", err)
	}

	// Verify deleted
	_, err = board.Get(ctx, auth, created.BoardID)
	if err == nil {
		t.Error("expected error getting deleted board")
	}
}

func TestBoardFromTemplate(t *testing.T) {
	yaotest.Prepare(t, config.Conf)
	defer yaotest.Clean()

	ctx := context.Background()
	auth := &process.AuthorizedInfo{
		UserID: "test-user-001",
		TeamID: "test-team-001",
	}

	created, err := board.FromTemplate(ctx, auth, &board.FromTemplateReq{
		TemplateID: "dev-workflow",
	})
	if err != nil {
		t.Fatalf("board.FromTemplate failed: %v", err)
	}
	if created.Name != "开发工作流" {
		t.Errorf("unexpected board name: %s", created.Name)
	}
	if len(created.Columns) != 3 {
		t.Errorf("expected 3 columns from template, got %d", len(created.Columns))
	}
}
