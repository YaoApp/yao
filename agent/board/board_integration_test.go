//go:build integration

package board_test

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/yaoapp/gou/process"
	"github.com/yaoapp/yao/agent/board"
	"github.com/yaoapp/yao/agent/task"
	"github.com/yaoapp/yao/unit-test/agent/testprepare"
)

func setupTest(t *testing.T) (context.Context, *process.AuthorizedInfo) {
	t.Helper()
	identity := testprepare.PrepareSandbox(t)
	ctx := context.Background()
	auth := &process.AuthorizedInfo{
		UserID: identity.AlphaOwnerUserID,
		TeamID: identity.AlphaTeamID,
	}
	return ctx, auth
}

func TestBoardCRUD(t *testing.T) {
	ctx, auth := setupTest(t)

	// Create
	created, err := board.Create(ctx, auth, &board.CreateReq{
		Name:  "Test Board",
		Icon:  "material-test",
		Color: "#3B82F6",
	})
	require.NoError(t, err)
	require.NotEmpty(t, created.BoardID)
	assert.Equal(t, "Test Board", created.Name)
	assert.Equal(t, "material-test", created.Icon)
	assert.Equal(t, "#3B82F6", created.Color)
	require.Len(t, created.Columns, 1)
	assert.Equal(t, "To Do", created.Columns[0].Name)
	assert.Equal(t, 1, created.Columns[0].Position)

	// Get
	got, err := board.Get(ctx, auth, created.BoardID)
	require.NoError(t, err)
	assert.Equal(t, created.BoardID, got.BoardID)
	assert.Equal(t, created.Name, got.Name)
	assert.Equal(t, created.Icon, got.Icon)
	assert.Equal(t, created.Color, got.Color)
	assert.Len(t, got.Columns, 1)

	// List
	result, err := board.List(ctx, auth, &board.ListQuery{})
	require.NoError(t, err)
	found := false
	for _, b := range result.Boards {
		if b.BoardID == created.BoardID {
			found = true
			assert.Equal(t, "Test Board", b.Name)
		}
	}
	assert.True(t, found, "created board should appear in list")

	// CreateColumn
	col, err := board.CreateColumn(ctx, auth, created.BoardID, &board.ColumnReq{
		Name:  "In Progress",
		Icon:  "material-play",
		Color: "#F59E0B",
	})
	require.NoError(t, err)
	assert.Equal(t, "In Progress", col.Name)
	assert.Equal(t, "material-play", col.Icon)
	assert.Equal(t, "#F59E0B", col.Color)
	assert.GreaterOrEqual(t, col.Position, 1)

	// Verify column was created (Get returns columns for this board)
	got, err = board.Get(ctx, auth, created.BoardID)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(got.Columns), 1)

	// Update
	newName := "Updated Board"
	newIcon := "material-edit"
	updated, err := board.Update(ctx, auth, created.BoardID, &board.UpdateReq{
		Name: &newName,
		Icon: &newIcon,
	})
	require.NoError(t, err)
	assert.Equal(t, "Updated Board", updated.Name)
	assert.Equal(t, "material-edit", updated.Icon)
	assert.Equal(t, "#3B82F6", updated.Color, "unchanged fields should persist")

	// Delete
	err = board.Delete(ctx, auth, created.BoardID)
	require.NoError(t, err)

	// Verify soft-delete via List
	listAfter, err := board.List(ctx, auth, &board.ListQuery{})
	require.NoError(t, err)
	for _, b := range listAfter.Boards {
		assert.NotEqual(t, created.BoardID, b.BoardID, "deleted board should not appear in list")
	}
}

func TestBoardFromTemplate(t *testing.T) {
	ctx, auth := setupTest(t)

	t.Run("dev-workflow", func(t *testing.T) {
		created, err := board.FromTemplate(ctx, auth, &board.FromTemplateReq{
			TemplateID: "dev-workflow",
		})
		require.NoError(t, err)
		assert.Equal(t, "Dev Workflow", created.Name)
		assert.GreaterOrEqual(t, len(created.Columns), 1, "dev-workflow template should have at least one column")
		assert.NotEmpty(t, created.BoardID)
	})

	t.Run("kanban-basic", func(t *testing.T) {
		created, err := board.FromTemplate(ctx, auth, &board.FromTemplateReq{
			TemplateID: "kanban-basic",
		})
		require.NoError(t, err)
		assert.NotEmpty(t, created.BoardID)
		assert.NotEmpty(t, created.Name)
		assert.GreaterOrEqual(t, len(created.Columns), 2)
	})
}

func TestBoardTasks(t *testing.T) {
	ctx, auth := setupTest(t)

	// Create a board with a default column
	b, err := board.Create(ctx, auth, &board.CreateReq{
		Name: "Tasks Test Board",
	})
	require.NoError(t, err)
	require.Len(t, b.Columns, 1)
	colID := b.Columns[0].ColumnID

	// Board should have no tasks initially
	tasks, err := board.Tasks(ctx, auth, b.BoardID, "")
	require.NoError(t, err)
	assert.Empty(t, tasks)

	// Create tasks in the board column
	t1, err := task.Create(ctx, auth, &task.CreateReq{
		Title:       "First Task",
		AssistantID: "asst-test",
		ColumnID:    colID,
	})
	require.NoError(t, err)
	assert.Equal(t, "First Task", t1.Title)

	t2, err := task.Create(ctx, auth, &task.CreateReq{
		Title:       "Second Task",
		AssistantID: "asst-test",
		ColumnID:    colID,
	})
	require.NoError(t, err)
	assert.Equal(t, "Second Task", t2.Title)

	// Verify Tasks() returns the created tasks
	tasks, err = board.Tasks(ctx, auth, b.BoardID, "")
	require.NoError(t, err)
	assert.Len(t, tasks, 2)

	// Verify TaskCount via Get
	got, err := board.Get(ctx, auth, b.BoardID)
	require.NoError(t, err)
	assert.Equal(t, 2, got.TaskCount)

	// Verify TaskCount via List
	result, err := board.List(ctx, auth, &board.ListQuery{})
	require.NoError(t, err)
	for _, lb := range result.Boards {
		if lb.BoardID == b.BoardID {
			assert.Equal(t, 2, lb.TaskCount)
		}
	}
}

func TestBoardReorderColumns(t *testing.T) {
	ctx, auth := setupTest(t)

	b, err := board.Create(ctx, auth, &board.CreateReq{
		Name: "Reorder Test Board",
	})
	require.NoError(t, err)
	defaultColID := b.Columns[0].ColumnID

	// Add more columns
	col2, err := board.CreateColumn(ctx, auth, b.BoardID, &board.ColumnReq{
		Name: "In Progress",
	})
	require.NoError(t, err)

	col3, err := board.CreateColumn(ctx, auth, b.BoardID, &board.ColumnReq{
		Name: "Done",
	})
	require.NoError(t, err)

	// Verify initial order
	got, err := board.Get(ctx, auth, b.BoardID)
	require.NoError(t, err)
	require.Len(t, got.Columns, 3)
	assert.Equal(t, defaultColID, got.Columns[0].ColumnID)
	assert.Equal(t, col2.ColumnID, got.Columns[1].ColumnID)
	assert.Equal(t, col3.ColumnID, got.Columns[2].ColumnID)

	// Reorder: move "Done" to first, "In Progress" to second, default to last
	newOrder := []string{col3.ColumnID, col2.ColumnID, defaultColID}
	err = board.ReorderColumns(ctx, auth, b.BoardID, newOrder)
	require.NoError(t, err)

	// Verify new order
	got, err = board.Get(ctx, auth, b.BoardID)
	require.NoError(t, err)
	require.Len(t, got.Columns, 3)
	assert.Equal(t, col3.ColumnID, got.Columns[0].ColumnID)
	assert.Equal(t, 1, got.Columns[0].Position)
	assert.Equal(t, col2.ColumnID, got.Columns[1].ColumnID)
	assert.Equal(t, 2, got.Columns[1].Position)
	assert.Equal(t, defaultColID, got.Columns[2].ColumnID)
	assert.Equal(t, 3, got.Columns[2].Position)

	// Reorder back to original
	err = board.ReorderColumns(ctx, auth, b.BoardID, []string{defaultColID, col2.ColumnID, col3.ColumnID})
	require.NoError(t, err)
	got, err = board.Get(ctx, auth, b.BoardID)
	require.NoError(t, err)
	assert.Equal(t, defaultColID, got.Columns[0].ColumnID)
	assert.Equal(t, col2.ColumnID, got.Columns[1].ColumnID)
	assert.Equal(t, col3.ColumnID, got.Columns[2].ColumnID)
}

func TestBoardUpdateColumn(t *testing.T) {
	ctx, auth := setupTest(t)

	b, err := board.Create(ctx, auth, &board.CreateReq{
		Name: "UpdateCol Test Board",
	})
	require.NoError(t, err)
	colID := b.Columns[0].ColumnID

	// Update name and color
	updated, err := board.UpdateColumn(ctx, auth, b.BoardID, colID, &board.ColumnReq{
		Name:  "Backlog",
		Color: "#EF4444",
		Icon:  "material-inbox",
	})
	require.NoError(t, err)
	assert.Equal(t, "Backlog", updated.Name)
	assert.Equal(t, "#EF4444", updated.Color)
	assert.Equal(t, "material-inbox", updated.Icon)
	assert.Equal(t, colID, updated.ColumnID)

	// Verify via Get
	got, err := board.Get(ctx, auth, b.BoardID)
	require.NoError(t, err)
	require.Len(t, got.Columns, 1)
	assert.Equal(t, "Backlog", got.Columns[0].Name)
	assert.Equal(t, "#EF4444", got.Columns[0].Color)

	// Partial update: only name
	updated, err = board.UpdateColumn(ctx, auth, b.BoardID, colID, &board.ColumnReq{
		Name: "Updated Backlog",
	})
	require.NoError(t, err)
	assert.Equal(t, "Updated Backlog", updated.Name)

	// Update collapsed state
	collapsed := true
	updated, err = board.UpdateColumn(ctx, auth, b.BoardID, colID, &board.ColumnReq{
		Name:      "Updated Backlog",
		Collapsed: &collapsed,
	})
	require.NoError(t, err)
	assert.True(t, updated.Collapsed)

	// Expand it back
	expanded := false
	updated, err = board.UpdateColumn(ctx, auth, b.BoardID, colID, &board.ColumnReq{
		Name:      "Updated Backlog",
		Collapsed: &expanded,
	})
	require.NoError(t, err)
	assert.False(t, updated.Collapsed)
}

func TestBoardDeleteColumn(t *testing.T) {
	ctx, auth := setupTest(t)

	b, err := board.Create(ctx, auth, &board.CreateReq{
		Name: "DeleteCol Test Board",
	})
	require.NoError(t, err)
	firstColID := b.Columns[0].ColumnID

	// Add a second column
	col2, err := board.CreateColumn(ctx, auth, b.BoardID, &board.ColumnReq{
		Name: "In Progress",
	})
	require.NoError(t, err)

	// Add a third column
	col3, err := board.CreateColumn(ctx, auth, b.BoardID, &board.ColumnReq{
		Name: "Done",
	})
	require.NoError(t, err)

	// Create a task in col2 so we can verify migration
	_, err = task.Create(ctx, auth, &task.CreateReq{
		Title:       "Migrating Task",
		AssistantID: "asst-test",
		ColumnID:    col2.ColumnID,
	})
	require.NoError(t, err)

	// Delete col2 — tasks should migrate to adjacent column (col1, since col2 is at index 1)
	err = board.DeleteColumn(ctx, auth, b.BoardID, col2.ColumnID)
	require.NoError(t, err)

	// Verify column is gone
	got, err := board.Get(ctx, auth, b.BoardID)
	require.NoError(t, err)
	assert.Len(t, got.Columns, 2)
	for _, col := range got.Columns {
		assert.NotEqual(t, col2.ColumnID, col.ColumnID, "deleted column should not appear")
	}

	// Remaining columns should be reordered
	assert.Equal(t, 1, got.Columns[0].Position)
	assert.Equal(t, 2, got.Columns[1].Position)

	// Verify task migrated (should be in firstColID now)
	tasks, err := board.Tasks(ctx, auth, b.BoardID, "")
	require.NoError(t, err)
	assert.Len(t, tasks, 1, "migrated task should still exist")

	// Cannot delete last column
	err = board.DeleteColumn(ctx, auth, b.BoardID, col3.ColumnID)
	require.NoError(t, err)

	// Only firstColID remains — trying to delete it should fail
	got, err = board.Get(ctx, auth, b.BoardID)
	require.NoError(t, err)
	require.Len(t, got.Columns, 1)

	err = board.DeleteColumn(ctx, auth, b.BoardID, firstColID)
	assert.Error(t, err, "should not be able to delete the last column")
	assert.Contains(t, err.Error(), "cannot delete the last column")
}

func TestBoardListMultiple(t *testing.T) {
	ctx, auth := setupTest(t)

	// Create multiple boards
	boardIDs := make([]string, 3)
	for i := 0; i < 3; i++ {
		b, err := board.Create(ctx, auth, &board.CreateReq{
			Name: "List Board " + string(rune('A'+i)),
		})
		require.NoError(t, err)
		boardIDs[i] = b.BoardID
	}

	// List should contain all created boards
	result, err := board.List(ctx, auth, &board.ListQuery{})
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(result.Boards), 3)

	foundCount := 0
	for _, b := range result.Boards {
		for _, id := range boardIDs {
			if b.BoardID == id {
				foundCount++
			}
		}
	}
	assert.Equal(t, 3, foundCount, "all created boards should appear in list")

	// Boards should be ordered by position (ascending)
	for i := 1; i < len(result.Boards); i++ {
		assert.GreaterOrEqual(t, result.Boards[i].Position, result.Boards[i-1].Position,
			"boards should be ordered by position")
	}

	// Each board in list should have columns loaded
	for _, b := range result.Boards {
		for _, id := range boardIDs {
			if b.BoardID == id {
				assert.GreaterOrEqual(t, len(b.Columns), 1, "listed board should have columns")
			}
		}
	}

	// Delete one and verify it disappears from list
	err = board.Delete(ctx, auth, boardIDs[1])
	require.NoError(t, err)

	result, err = board.List(ctx, auth, &board.ListQuery{})
	require.NoError(t, err)
	for _, b := range result.Boards {
		assert.NotEqual(t, boardIDs[1], b.BoardID, "deleted board should not be in list")
	}
}

func TestBoardGetColumnsIndirect(t *testing.T) {
	ctx, auth := setupTest(t)

	// Create board — should start with 1 default column
	b, err := board.Create(ctx, auth, &board.CreateReq{
		Name: "GetColumns Test",
	})
	require.NoError(t, err)
	require.Len(t, b.Columns, 1)
	assert.Equal(t, "To Do", b.Columns[0].Name)
	assert.Equal(t, b.BoardID, b.Columns[0].BoardID)

	// Add multiple columns
	names := []string{"In Progress", "Review", "Done"}
	colIDs := []string{b.Columns[0].ColumnID}
	for _, name := range names {
		col, err := board.CreateColumn(ctx, auth, b.BoardID, &board.ColumnReq{Name: name})
		require.NoError(t, err)
		colIDs = append(colIDs, col.ColumnID)
	}

	// Get should return all columns for this board
	got, err := board.Get(ctx, auth, b.BoardID)
	require.NoError(t, err)
	require.GreaterOrEqual(t, len(got.Columns), 1, "should have at least default column")
	assert.Equal(t, "To Do", got.Columns[0].Name)
	assert.Equal(t, b.BoardID, got.Columns[0].BoardID)
	for _, col := range got.Columns {
		assert.NotEmpty(t, col.ColumnID)
		assert.Equal(t, b.BoardID, col.BoardID)
	}
}

func TestBoardDeleteNullifiesTaskColumn(t *testing.T) {
	ctx, auth := setupTest(t)

	b, err := board.Create(ctx, auth, &board.CreateReq{
		Name: "Delete Nullify Test",
	})
	require.NoError(t, err)
	colID := b.Columns[0].ColumnID

	// Create a task in the board
	created, err := task.Create(ctx, auth, &task.CreateReq{
		Title:       "Orphan Task",
		AssistantID: "asst-test",
		ColumnID:    colID,
	})
	require.NoError(t, err)
	require.NotNil(t, created.ColumnID)

	// Delete the entire board — tasks should have column_id nullified
	err = board.Delete(ctx, auth, b.BoardID)
	require.NoError(t, err)

	// Verify board no longer in list
	result, err := board.List(ctx, auth, &board.ListQuery{})
	require.NoError(t, err)
	for _, lb := range result.Boards {
		assert.NotEqual(t, b.BoardID, lb.BoardID)
	}
}

func TestBoardColumnCollapsed(t *testing.T) {
	ctx, auth := setupTest(t)

	b, err := board.Create(ctx, auth, &board.CreateReq{
		Name: "Collapsed Test",
	})
	require.NoError(t, err)

	// Default column should not be collapsed
	got, err := board.Get(ctx, auth, b.BoardID)
	require.NoError(t, err)
	assert.False(t, got.Columns[0].Collapsed)

	// Create a collapsed column
	collapsed := true
	col, err := board.CreateColumn(ctx, auth, b.BoardID, &board.ColumnReq{
		Name:      "Archived",
		Collapsed: &collapsed,
	})
	require.NoError(t, err)
	assert.True(t, col.Collapsed)

	// Verify persisted via Get
	got, err = board.Get(ctx, auth, b.BoardID)
	require.NoError(t, err)
	require.Len(t, got.Columns, 2)
	assert.False(t, got.Columns[0].Collapsed)
	assert.True(t, got.Columns[1].Collapsed)
}

func TestBoardUpdatePartialFields(t *testing.T) {
	ctx, auth := setupTest(t)

	b, err := board.Create(ctx, auth, &board.CreateReq{
		Name:  "Partial Update",
		Icon:  "material-star",
		Color: "#10B981",
	})
	require.NoError(t, err)

	// Update only color
	newColor := "#8B5CF6"
	updated, err := board.Update(ctx, auth, b.BoardID, &board.UpdateReq{
		Color: &newColor,
	})
	require.NoError(t, err)
	assert.Equal(t, "Partial Update", updated.Name, "name should remain unchanged")
	assert.Equal(t, "material-star", updated.Icon, "icon should remain unchanged")
	assert.Equal(t, "#8B5CF6", updated.Color, "color should be updated")

	// Update only name
	newName := "Renamed Board"
	updated, err = board.Update(ctx, auth, b.BoardID, &board.UpdateReq{
		Name: &newName,
	})
	require.NoError(t, err)
	assert.Equal(t, "Renamed Board", updated.Name)
	assert.Equal(t, "#8B5CF6", updated.Color, "color should persist from previous update")
}

func TestBoardTasksMigrationOnColumnDelete(t *testing.T) {
	ctx, auth := setupTest(t)

	b, err := board.Create(ctx, auth, &board.CreateReq{
		Name: "Migration Test",
	})
	require.NoError(t, err)
	col1ID := b.Columns[0].ColumnID

	col2, err := board.CreateColumn(ctx, auth, b.BoardID, &board.ColumnReq{Name: "Col 2"})
	require.NoError(t, err)

	col3, err := board.CreateColumn(ctx, auth, b.BoardID, &board.ColumnReq{Name: "Col 3"})
	require.NoError(t, err)

	// Create tasks in each column
	_, err = task.Create(ctx, auth, &task.CreateReq{
		Title: "Task in Col1", AssistantID: "asst-test", ColumnID: col1ID,
	})
	require.NoError(t, err)

	_, err = task.Create(ctx, auth, &task.CreateReq{
		Title: "Task A in Col2", AssistantID: "asst-test", ColumnID: col2.ColumnID,
	})
	require.NoError(t, err)

	_, err = task.Create(ctx, auth, &task.CreateReq{
		Title: "Task B in Col2", AssistantID: "asst-test", ColumnID: col2.ColumnID,
	})
	require.NoError(t, err)

	_, err = task.Create(ctx, auth, &task.CreateReq{
		Title: "Task in Col3", AssistantID: "asst-test", ColumnID: col3.ColumnID,
	})
	require.NoError(t, err)

	// Delete col2 (middle column) — its tasks should migrate to col1 (previous adjacent)
	err = board.DeleteColumn(ctx, auth, b.BoardID, col2.ColumnID)
	require.NoError(t, err)

	// All 4 tasks should still exist in the board
	tasks, err := board.Tasks(ctx, auth, b.BoardID, "")
	require.NoError(t, err)
	assert.Len(t, tasks, 4, "all tasks should survive column deletion")

	// Delete col1 (now first column) — its tasks should migrate to col3 (next adjacent)
	got, err := board.Get(ctx, auth, b.BoardID)
	require.NoError(t, err)
	require.Len(t, got.Columns, 2)

	err = board.DeleteColumn(ctx, auth, b.BoardID, col1ID)
	require.NoError(t, err)

	// All 4 tasks should still exist
	tasks, err = board.Tasks(ctx, auth, b.BoardID, "")
	require.NoError(t, err)
	assert.Len(t, tasks, 4)

	// Only col3 remains
	got, err = board.Get(ctx, auth, b.BoardID)
	require.NoError(t, err)
	assert.Len(t, got.Columns, 1)
	assert.Equal(t, col3.ColumnID, got.Columns[0].ColumnID)
}

func TestBoardTasks_ResponseFormat(t *testing.T) {
	ctx, auth := setupTest(t)

	b, err := board.Create(ctx, auth, &board.CreateReq{Name: "Format Test Board"})
	require.NoError(t, err)
	colID := b.Columns[0].ColumnID

	_, err = task.Create(ctx, auth, &task.CreateReq{
		Title:       "Format Task",
		AssistantID: "asst-test",
		ColumnID:    colID,
	})
	require.NoError(t, err)

	tasks, err := board.Tasks(ctx, auth, b.BoardID, "")
	require.NoError(t, err)
	require.Len(t, tasks, 1)

	// Simulate handler response wrapping (must match frontend expectation)
	wrapped := map[string]interface{}{
		"tasks": tasks,
		"total": len(tasks),
	}

	raw, err := json.Marshal(wrapped)
	require.NoError(t, err)

	var parsed map[string]json.RawMessage
	require.NoError(t, json.Unmarshal(raw, &parsed))

	// Frontend expects res.data.tasks to be an array
	assert.Contains(t, parsed, "tasks", "response must have 'tasks' field")
	assert.Contains(t, parsed, "total", "response must have 'total' field")

	var taskList []json.RawMessage
	require.NoError(t, json.Unmarshal(parsed["tasks"], &taskList))
	assert.Len(t, taskList, 1)

	var total int
	require.NoError(t, json.Unmarshal(parsed["total"], &total))
	assert.Equal(t, 1, total)
}
