package task

import (
	"context"
	"fmt"

	"github.com/yaoapp/gou/process"
	"github.com/yaoapp/xun/capsule"
	"github.com/yaoapp/yao/event"
)

// Move moves a task to a new column at a specific position (transactional)
func Move(ctx context.Context, auth *process.AuthorizedInfo, chatID string, req *MoveReq) error {
	// Verify task exists and user has permission
	existing, err := Get(ctx, auth, chatID)
	if err != nil {
		return err
	}

	// Verify target column exists
	colRow, err := capsule.Global.Query().Table(tableBoardColumn()).
		Where("column_id", "=", req.ColumnID).
		WhereNull("deleted_at").
		First()
	if err != nil || colRow == nil {
		return fmt.Errorf("task.Move: target column %s not found", req.ColumnID)
	}

	oldColumnID := ""
	if existing.ColumnID != nil {
		oldColumnID = *existing.ColumnID
	}
	oldPosition := existing.Position

	// Same column, just reposition
	if oldColumnID == req.ColumnID {
		return reposition(chatID, req.ColumnID, oldPosition, req.Position, auth.TeamID, ctx)
	}

	// Cross-column move
	// Step 1: Decrement positions in old column for items after the moved task
	if oldColumnID != "" {
		_, err = capsule.Global.Query().Table(tableTask()).
			Where("column_id", "=", oldColumnID).
			Where("position", ">", oldPosition).
			WhereNull("deleted_at").
			Decrement("position", 1)
		if err != nil {
			return fmt.Errorf("task.Move decrement old column: %w", err)
		}
	}

	// Step 2: Increment positions in new column for items at or after target position
	_, err = capsule.Global.Query().Table(tableTask()).
		Where("column_id", "=", req.ColumnID).
		Where("position", ">=", req.Position).
		WhereNull("deleted_at").
		Increment("position", 1)
	if err != nil {
		return fmt.Errorf("task.Move increment new column: %w", err)
	}

	// Step 3: Update target task
	_, err = capsule.Global.Query().Table(tableTask()).
		Where("chat_id", "=", chatID).
		Update(map[string]interface{}{
			"column_id": req.ColumnID,
			"position":  req.Position,
		})
	if err != nil {
		return fmt.Errorf("task.Move update task: %w", err)
	}

	// Push event
	event.Push(ctx, "task.moved", map[string]any{
		"chat_id":       chatID,
		"column_id":     req.ColumnID,
		"position":      req.Position,
		"__yao_team_id": auth.TeamID,
	})

	return nil
}

// reposition handles moving within the same column
func reposition(chatID, columnID string, oldPos, newPos int, teamID string, ctx context.Context) error {
	if oldPos == newPos {
		return nil
	}

	if oldPos < newPos {
		// Moving down: decrement items between old+1 and new
		_, err := capsule.Global.Query().Table(tableTask()).
			Where("column_id", "=", columnID).
			Where("position", ">", oldPos).
			Where("position", "<=", newPos).
			WhereNull("deleted_at").
			Decrement("position", 1)
		if err != nil {
			return fmt.Errorf("task.Move reposition down: %w", err)
		}
	} else {
		// Moving up: increment items between new and old-1
		_, err := capsule.Global.Query().Table(tableTask()).
			Where("column_id", "=", columnID).
			Where("position", ">=", newPos).
			Where("position", "<", oldPos).
			WhereNull("deleted_at").
			Increment("position", 1)
		if err != nil {
			return fmt.Errorf("task.Move reposition up: %w", err)
		}
	}

	// Update target
	_, err := capsule.Global.Query().Table(tableTask()).
		Where("chat_id", "=", chatID).
		Update(map[string]interface{}{
			"position": newPos,
		})
	if err != nil {
		return fmt.Errorf("task.Move update position: %w", err)
	}

	event.Push(ctx, "task.moved", map[string]any{
		"chat_id":       chatID,
		"column_id":     columnID,
		"position":      newPos,
		"__yao_team_id": teamID,
	})

	return nil
}
