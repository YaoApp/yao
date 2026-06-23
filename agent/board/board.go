package board

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/yaoapp/gou/process"
	"github.com/yaoapp/xun/capsule"
	"github.com/yaoapp/yao/agent/task"
	"github.com/yaoapp/yao/event"
)

// List returns all boards for the authenticated user/team
func List(ctx context.Context, auth *process.AuthorizedInfo, q *ListQuery) (*ListResult, error) {
	qb := capsule.Global.Query()

	qb.Table(tableBoard()).WhereNull("deleted_at")

	if auth.Constraints.TeamOnly {
		qb.Where("__yao_team_id", "=", auth.TeamID)
	}
	if auth.Constraints.CreatorOnly {
		qb.Where("__yao_created_by", "=", auth.UserID)
	}

	rows, err := qb.OrderBy("position", "asc").Get()
	if err != nil {
		return nil, fmt.Errorf("board.List: %w", err)
	}

	boards := make([]*Board, 0, len(rows))
	for _, row := range rows {
		b := rowToBoard(row)

		// Load columns
		cols, err := getColumns(b.BoardID)
		if err == nil {
			b.Columns = cols
		}

		// Count tasks
		colIDs := getColumnIDs(b.Columns)
		if len(colIDs) > 0 {
			count, _ := capsule.Global.Query().Table(tableTask()).
				WhereIn("column_id", colIDs).
				WhereNull("deleted_at").
				Count()
			b.TaskCount = int(count)
		}

		boards = append(boards, b)
	}

	return &ListResult{Boards: boards}, nil
}

// Get retrieves a single board by board_id
func Get(ctx context.Context, auth *process.AuthorizedInfo, boardID string) (*Board, error) {
	row, err := capsule.Global.Query().Table(tableBoard()).
		Where("board_id", "=", boardID).
		WhereNull("deleted_at").
		First()
	if err != nil {
		return nil, fmt.Errorf("board.Get: %w", err)
	}
	if row == nil {
		return nil, fmt.Errorf("board.Get: board %s not found", boardID)
	}

	if auth.Constraints.TeamOnly {
		if getString(row, "__yao_team_id") != auth.TeamID {
			return nil, fmt.Errorf("board.Get: permission denied")
		}
	}

	b := rowToBoard(row)
	cols, _ := getColumns(b.BoardID)
	b.Columns = cols

	colIDs := getColumnIDs(b.Columns)
	if len(colIDs) > 0 {
		count, _ := capsule.Global.Query().Table(tableTask()).
			WhereIn("column_id", colIDs).
			WhereNull("deleted_at").
			Count()
		b.TaskCount = int(count)
	}

	return b, nil
}

// Create creates a new board with one default column
func Create(ctx context.Context, auth *process.AuthorizedInfo, req *CreateReq) (*Board, error) {
	boardID := uuid.New().String()
	now := time.Now()

	// Get max position
	maxPos := 0
	posResult, _ := capsule.Global.Query().Table(tableBoard()).
		WhereNull("deleted_at").
		Max("position")
	if posResult.Number != nil {
		switch v := posResult.Number.(type) {
		case float64:
			maxPos = int(v)
		case int64:
			maxPos = int(v)
		case int:
			maxPos = v
		}
	}

	err := capsule.Global.Query().Table(tableBoard()).Insert(map[string]interface{}{
		"board_id":         boardID,
		"name":             req.Name,
		"icon":             req.Icon,
		"color":            req.Color,
		"position":         maxPos + 1,
		"__yao_created_by": auth.UserID,
		"__yao_team_id":    auth.TeamID,
		"created_at":       now,
		"updated_at":       now,
	})
	if err != nil {
		return nil, fmt.Errorf("board.Create: %w", err)
	}

	// Create default column
	colID := uuid.New().String()
	err = capsule.Global.Query().Table(tableBoardColumn()).Insert(map[string]interface{}{
		"column_id":        colID,
		"board_id":         boardID,
		"name":             "To Do",
		"position":         1,
		"collapsed":        false,
		"__yao_created_by": auth.UserID,
		"__yao_team_id":    auth.TeamID,
		"created_at":       now,
		"updated_at":       now,
	})
	if err != nil {
		return nil, fmt.Errorf("board.Create default column: %w", err)
	}

	return Get(ctx, auth, boardID)
}

// Update partially updates a board
func Update(ctx context.Context, auth *process.AuthorizedInfo, boardID string, req *UpdateReq) (*Board, error) {
	_, err := Get(ctx, auth, boardID)
	if err != nil {
		return nil, err
	}

	updates := map[string]interface{}{"updated_at": time.Now()}
	if req.Name != nil {
		updates["name"] = *req.Name
	}
	if req.Icon != nil {
		updates["icon"] = *req.Icon
	}
	if req.Color != nil {
		updates["color"] = *req.Color
	}
	if req.Position != nil {
		updates["position"] = *req.Position
	}

	if len(updates) > 1 {
		_, err = capsule.Global.Query().Table(tableBoard()).
			Where("board_id", "=", boardID).
			Update(updates)
		if err != nil {
			return nil, fmt.Errorf("board.Update: %w", err)
		}
	}

	return Get(ctx, auth, boardID)
}

// Delete soft-deletes a board, nullifies tasks' column_id, soft-deletes columns
func Delete(ctx context.Context, auth *process.AuthorizedInfo, boardID string) error {
	b, err := Get(ctx, auth, boardID)
	if err != nil {
		return err
	}

	now := time.Now()
	colIDs := getColumnIDs(b.Columns)

	// Nullify task column references
	if len(colIDs) > 0 {
		_, err = capsule.Global.Query().Table(tableTask()).
			WhereIn("column_id", colIDs).
			WhereNull("deleted_at").
			Update(map[string]interface{}{
				"column_id":  nil,
				"updated_at": now,
			})
		if err != nil {
			return fmt.Errorf("board.Delete nullify tasks: %w", err)
		}
	}

	// Soft delete columns
	_, err = capsule.Global.Query().Table(tableBoardColumn()).
		Where("board_id", "=", boardID).
		WhereNull("deleted_at").
		Update(map[string]interface{}{
			"deleted_at": now,
			"updated_at": now,
		})
	if err != nil {
		return fmt.Errorf("board.Delete columns: %w", err)
	}

	// Soft delete board
	_, err = capsule.Global.Query().Table(tableBoard()).
		Where("board_id", "=", boardID).
		Update(map[string]interface{}{
			"deleted_at": now,
			"updated_at": now,
		})
	if err != nil {
		return fmt.Errorf("board.Delete board: %w", err)
	}

	event.Push(ctx, "board.deleted", map[string]any{
		"board_id":      boardID,
		"__yao_team_id": auth.TeamID,
	})

	return nil
}

// Tasks returns all tasks in a board ordered by column position then task position
func Tasks(ctx context.Context, auth *process.AuthorizedInfo, boardID string, locale string) ([]*task.Task, error) {
	_, err := Get(ctx, auth, boardID)
	if err != nil {
		return nil, err
	}

	q := &task.ListQuery{
		BoardID:  boardID,
		PageSize: 1000,
		Page:     1,
		Locale:   locale,
	}
	result, err := task.List(ctx, auth, q)
	if err != nil {
		return nil, err
	}
	return result.Tasks, nil
}

// helpers

func getColumns(boardID string) ([]*Column, error) {
	rows, err := capsule.Global.Query().Table(tableBoardColumn()).
		Where("board_id", "=", boardID).
		WhereNull("deleted_at").
		OrderBy("position", "asc").
		Get()
	if err != nil {
		return nil, err
	}

	cols := make([]*Column, 0, len(rows))
	for _, row := range rows {
		cols = append(cols, rowToColumn(row))
	}
	return cols, nil
}

func getColumnIDs(cols []*Column) []interface{} {
	ids := make([]interface{}, 0, len(cols))
	for _, c := range cols {
		ids = append(ids, c.ColumnID)
	}
	return ids
}

func rowToBoard(row map[string]interface{}) *Board {
	b := &Board{
		BoardID:  getString(row, "board_id"),
		Name:     getString(row, "name"),
		Icon:     getString(row, "icon"),
		Color:    getString(row, "color"),
		Position: getInt(row, "position"),
	}
	if v := getTime(row, "created_at"); v != nil {
		b.CreatedAt = *v
	}
	if v := getTime(row, "updated_at"); v != nil {
		b.UpdatedAt = *v
	}
	return b
}

func rowToColumn(row map[string]interface{}) *Column {
	c := &Column{
		ColumnID:  getString(row, "column_id"),
		BoardID:   getString(row, "board_id"),
		Name:      getString(row, "name"),
		Icon:      getString(row, "icon"),
		Color:     getString(row, "color"),
		Position:  getInt(row, "position"),
		Collapsed: getBool(row, "collapsed"),
	}
	if v := getTime(row, "created_at"); v != nil {
		c.CreatedAt = *v
	}
	return c
}

func getString(row map[string]interface{}, key string) string {
	if v, ok := row[key]; ok && v != nil {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

func getInt(row map[string]interface{}, key string) int {
	if v, ok := row[key]; ok && v != nil {
		switch n := v.(type) {
		case float64:
			return int(n)
		case int64:
			return int(n)
		case int:
			return n
		}
	}
	return 0
}

func getBool(row map[string]interface{}, key string) bool {
	if v, ok := row[key]; ok && v != nil {
		switch b := v.(type) {
		case bool:
			return b
		case float64:
			return b != 0
		case int64:
			return b != 0
		}
	}
	return false
}

func getTime(row map[string]interface{}, key string) *time.Time {
	if v, ok := row[key]; ok && v != nil {
		switch t := v.(type) {
		case time.Time:
			return &t
		case string:
			if parsed, err := time.Parse(time.RFC3339, t); err == nil {
				return &parsed
			}
			if parsed, err := time.Parse("2006-01-02 15:04:05", t); err == nil {
				return &parsed
			}
		}
	}
	return nil
}
