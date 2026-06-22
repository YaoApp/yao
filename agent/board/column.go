package board

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/yaoapp/gou/process"
	"github.com/yaoapp/xun/capsule"
)

// CreateColumn creates a new column in a board at the end position
func CreateColumn(ctx context.Context, auth *process.AuthorizedInfo, boardID string, req *ColumnReq) (*Column, error) {
	// Verify board exists
	_, err := Get(ctx, auth, boardID)
	if err != nil {
		return nil, err
	}

	colID := uuid.New().String()
	now := time.Now()

	// Get max position in board
	maxPos := 0
	posResult, _ := capsule.Global.Query().Table(tableBoardColumn()).
		Where("board_id", "=", boardID).
		WhereNull("deleted_at").
		Max("position")
	if posResult.Number != nil {
		if v, ok := posResult.Number.(float64); ok {
			maxPos = int(v)
		}
	}

	data := map[string]interface{}{
		"column_id":        colID,
		"board_id":         boardID,
		"name":             req.Name,
		"icon":             req.Icon,
		"color":            req.Color,
		"position":         maxPos + 1,
		"collapsed":        false,
		"__yao_created_by": auth.UserID,
		"__yao_team_id":    auth.TeamID,
		"created_at":       now,
		"updated_at":       now,
	}
	if req.Collapsed != nil {
		data["collapsed"] = *req.Collapsed
	}

	err = capsule.Global.Query().Table(tableBoardColumn()).Insert(data)
	if err != nil {
		return nil, fmt.Errorf("board.CreateColumn: %w", err)
	}

	return &Column{
		ColumnID:  colID,
		BoardID:   boardID,
		Name:      req.Name,
		Icon:      req.Icon,
		Color:     req.Color,
		Position:  maxPos + 1,
		Collapsed: data["collapsed"].(bool),
		CreatedAt: now,
	}, nil
}

// ReorderColumns reorders columns by the given column_id array
func ReorderColumns(ctx context.Context, auth *process.AuthorizedInfo, boardID string, ids []string) error {
	_, err := Get(ctx, auth, boardID)
	if err != nil {
		return err
	}

	for i, colID := range ids {
		_, err := capsule.Global.Query().Table(tableBoardColumn()).
			Where("column_id", "=", colID).
			Where("board_id", "=", boardID).
			Update(map[string]interface{}{
				"position":   i + 1,
				"updated_at": time.Now(),
			})
		if err != nil {
			return fmt.Errorf("board.ReorderColumns: %w", err)
		}
	}

	return nil
}

// UpdateColumn updates a column's properties
func UpdateColumn(ctx context.Context, auth *process.AuthorizedInfo, boardID, colID string, req *ColumnReq) (*Column, error) {
	_, err := Get(ctx, auth, boardID)
	if err != nil {
		return nil, err
	}

	updates := map[string]interface{}{"updated_at": time.Now()}
	if req.Name != "" {
		updates["name"] = req.Name
	}
	if req.Icon != "" {
		updates["icon"] = req.Icon
	}
	if req.Color != "" {
		updates["color"] = req.Color
	}
	if req.Collapsed != nil {
		updates["collapsed"] = *req.Collapsed
	}

	_, err = capsule.Global.Query().Table(tableBoardColumn()).
		Where("column_id", "=", colID).
		Where("board_id", "=", boardID).
		WhereNull("deleted_at").
		Update(updates)
	if err != nil {
		return nil, fmt.Errorf("board.UpdateColumn: %w", err)
	}

	// Return updated column
	row, err := capsule.Global.Query().Table(tableBoardColumn()).
		Where("column_id", "=", colID).
		First()
	if err != nil || row == nil {
		return nil, fmt.Errorf("board.UpdateColumn fetch: %w", err)
	}

	return rowToColumn(row), nil
}

// DeleteColumn deletes a column, moving its tasks to an adjacent column.
// The last column in a board cannot be deleted.
func DeleteColumn(ctx context.Context, auth *process.AuthorizedInfo, boardID, colID string) error {
	b, err := Get(ctx, auth, boardID)
	if err != nil {
		return err
	}

	if len(b.Columns) <= 1 {
		return fmt.Errorf("board.DeleteColumn: cannot delete the last column")
	}

	// Find target column (adjacent)
	var targetColID string
	for i, col := range b.Columns {
		if col.ColumnID == colID {
			if i > 0 {
				targetColID = b.Columns[i-1].ColumnID
			} else if i < len(b.Columns)-1 {
				targetColID = b.Columns[i+1].ColumnID
			}
			break
		}
	}
	if targetColID == "" {
		return fmt.Errorf("board.DeleteColumn: column %s not found in board", colID)
	}

	now := time.Now()

	// Move tasks to adjacent column (append at end)
	maxPos := 0
	posResult, _ := capsule.Global.Query().Table(tableTask()).
		Where("column_id", "=", targetColID).
		WhereNull("deleted_at").
		Max("position")
	if posResult.Number != nil {
		if v, ok := posResult.Number.(float64); ok {
			maxPos = int(v)
		}
	}

	// Get tasks in the column to be deleted
	tasks, err := capsule.Global.Query().Table(tableTask()).
		Where("column_id", "=", colID).
		WhereNull("deleted_at").
		OrderBy("position", "asc").
		Get()
	if err != nil {
		return fmt.Errorf("board.DeleteColumn get tasks: %w", err)
	}

	for i, t := range tasks {
		chatID := ""
		if v, ok := t["chat_id"].(string); ok {
			chatID = v
		}
		if chatID != "" {
			_, err = capsule.Global.Query().Table(tableTask()).
				Where("chat_id", "=", chatID).
				Update(map[string]interface{}{
					"column_id":  targetColID,
					"position":   maxPos + i + 1,
					"updated_at": now,
				})
			if err != nil {
				return fmt.Errorf("board.DeleteColumn move task: %w", err)
			}
		}
	}

	// Soft delete column
	_, err = capsule.Global.Query().Table(tableBoardColumn()).
		Where("column_id", "=", colID).
		Update(map[string]interface{}{
			"deleted_at": now,
			"updated_at": now,
		})
	if err != nil {
		return fmt.Errorf("board.DeleteColumn: %w", err)
	}

	// Reorder remaining columns
	remainingCols, _ := getColumns(boardID)
	ids := make([]string, 0, len(remainingCols))
	for _, c := range remainingCols {
		ids = append(ids, c.ColumnID)
	}
	if len(ids) > 0 {
		return ReorderColumns(ctx, auth, boardID, ids)
	}

	return nil
}
