package inbox

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/yaoapp/xun/capsule"
	"github.com/yaoapp/yao/event"
)

// OnStatusChange generates inbox messages when task status changes.
// Called by the execution engine in Plan 3; implemented now but not invoked in Plan 1.
func OnStatusChange(ctx context.Context, task *AgentTask, newStatus string) error {
	if task.DeletedAt != nil {
		return nil
	}

	var mailType, priority string
	switch newStatus {
	case "waiting":
		mailType, priority = "input", "high"
	case "completed":
		mailType, priority = "completed", "low"
	case "failed":
		mailType, priority = "failed", "medium"
	default:
		return nil
	}

	return createMail(ctx, task, mailType, priority)
}

func createMail(ctx context.Context, task *AgentTask, mailType, priority string) error {
	boardID := getBoardIDFromColumn(ctx, task.ColumnID)
	boardName := getBoardName(ctx, boardID)

	title := generateTitle(task.ChatID, mailType)
	mailID := uuid.New().String()
	now := time.Now()

	err := capsule.Global.Query().Table("agent_mail").Insert(map[string]interface{}{
		"mail_id":          mailID,
		"type":             mailType,
		"priority":         priority,
		"title":            title,
		"body":             "",
		"chat_id":          task.ChatID,
		"assistant_id":     task.AssistantID,
		"source_type":      "kanban",
		"source_id":        boardID,
		"source_name":      boardName,
		"read":             false,
		"archived":         false,
		"starred":          false,
		"pinned":           false,
		"__yao_created_by": task.CreatedBy,
		"__yao_team_id":    task.TeamID,
		"created_at":       now,
		"updated_at":       now,
	})
	if err != nil {
		return fmt.Errorf("inbox.createMail: %w", err)
	}

	event.Push(ctx, "mail.new", map[string]any{
		"mail_id":          mailID,
		"type":             mailType,
		"title":            title,
		"chat_id":          task.ChatID,
		"__yao_created_by": task.CreatedBy,
	})

	return nil
}

func generateTitle(chatID, mailType string) string {
	// Get chat title
	title := chatID
	row, err := capsule.Global.Query().Table("agent_chat").
		Select("title").
		Where("chat_id", "=", chatID).
		First()
	if err == nil && row != nil {
		if t := getString(row, "title"); t != "" {
			title = t
		}
	}

	switch mailType {
	case "input":
		return fmt.Sprintf("「%s」需要你的输入", title)
	case "completed":
		return fmt.Sprintf("「%s」已完成", title)
	case "failed":
		return fmt.Sprintf("「%s」执行失败", title)
	default:
		return fmt.Sprintf("「%s」状态更新", title)
	}
}

func getBoardIDFromColumn(ctx context.Context, columnID string) string {
	if columnID == "" {
		return ""
	}
	row, err := capsule.Global.Query().Table("agent_board_column").
		Select("board_id").
		Where("column_id", "=", columnID).
		First()
	if err != nil || row == nil {
		return ""
	}
	return getString(row, "board_id")
}

func getBoardName(ctx context.Context, boardID string) string {
	if boardID == "" {
		return ""
	}
	row, err := capsule.Global.Query().Table("agent_board").
		Select("name").
		Where("board_id", "=", boardID).
		First()
	if err != nil || row == nil {
		return ""
	}
	return getString(row, "name")
}
