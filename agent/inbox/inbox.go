package inbox

import (
	"context"
	"fmt"
	"time"

	"github.com/yaoapp/gou/process"
	"github.com/yaoapp/xun/capsule"
	"github.com/yaoapp/xun/dbal/query"
)

// List returns a paginated list of inbox messages
func List(ctx context.Context, auth *process.AuthorizedInfo, q *ListQuery) (*ListResult, error) {
	if q.Size <= 0 {
		q.Size = 20
	}
	if q.Page <= 0 {
		q.Page = 1
	}
	if q.Filter == "" {
		q.Filter = "all"
	}

	// Count query (separate builder to avoid state pollution)
	countQB := capsule.Global.Query()
	countQB.Table("agent_mail").
		Where("__yao_created_by", "=", auth.UserID).
		WhereNull("deleted_at")
	applyInboxFilters(countQB, q)

	total, err := countQB.Count()
	if err != nil {
		return nil, fmt.Errorf("inbox.List count: %w", err)
	}

	// Data query (fresh builder)
	qb := capsule.Global.Query()
	qb.Table("agent_mail").
		Where("__yao_created_by", "=", auth.UserID).
		WhereNull("deleted_at")
	applyInboxFilters(qb, q)

	offset := (q.Page - 1) * q.Size
	rows, err := qb.
		OrderBy("pinned", "desc").
		OrderBy("read", "asc").
		OrderBy("created_at", "desc").
		Offset(offset).
		Limit(q.Size).
		Get()
	if err != nil {
		return nil, fmt.Errorf("inbox.List query: %w", err)
	}

	mails := make([]*AgentMail, 0, len(rows))
	for _, row := range rows {
		mails = append(mails, rowToMail(row))
	}

	return &ListResult{
		Mails: mails,
		Total: int64(total),
		Page:  q.Page,
		Size:  q.Size,
	}, nil
}

// UnreadCount returns unread counts grouped by type
func UnreadCount(ctx context.Context, auth *process.AuthorizedInfo) (*Counts, error) {
	rows, err := capsule.Global.Query().Table("agent_mail").
		Select("type").
		SelectRaw("COUNT(*) as cnt").
		Where("__yao_created_by", "=", auth.UserID).
		Where("read", "=", false).
		Where("archived", "=", false).
		WhereNull("deleted_at").
		GroupBy("type").
		Get()
	if err != nil {
		return nil, fmt.Errorf("inbox.UnreadCount: %w", err)
	}

	counts := &Counts{}
	for _, row := range rows {
		typ := getString(row, "type")
		cnt := getInt(row, "cnt")
		counts.Total += cnt
		switch typ {
		case "input":
			counts.Input = cnt
		case "completed":
			counts.Completed = cnt
		case "failed":
			counts.Failed = cnt
		}
	}

	return counts, nil
}

// Read marks a single mail as read
func Read(ctx context.Context, auth *process.AuthorizedInfo, mailID string) error {
	now := time.Now()
	affected, err := capsule.Global.Query().Table("agent_mail").
		Where("mail_id", "=", mailID).
		Where("__yao_created_by", "=", auth.UserID).
		Update(map[string]interface{}{
			"read":       true,
			"read_at":    now,
			"updated_at": now,
		})
	if err != nil {
		return fmt.Errorf("inbox.Read: %w", err)
	}
	if affected == 0 {
		return fmt.Errorf("inbox.Read: mail %s not found", mailID)
	}
	return nil
}

// ReadAll marks all unread mails as read, optionally filtered by type
func ReadAll(ctx context.Context, auth *process.AuthorizedInfo, mailType string) (int64, error) {
	now := time.Now()
	qb := capsule.Global.Query().Table("agent_mail").
		Where("__yao_created_by", "=", auth.UserID).
		Where("read", "=", false).
		Where("archived", "=", false).
		WhereNull("deleted_at")

	if mailType != "" {
		qb.Where("type", "=", mailType)
	}

	affected, err := qb.Update(map[string]interface{}{
		"read":       true,
		"read_at":    now,
		"updated_at": now,
	})
	if err != nil {
		return 0, fmt.Errorf("inbox.ReadAll: %w", err)
	}
	return int64(affected), nil
}

// Archive marks a mail as archived
func Archive(ctx context.Context, auth *process.AuthorizedInfo, mailID string) error {
	return updateMailField(auth, mailID, "archived", true)
}

// Star marks a mail as starred
func Star(ctx context.Context, auth *process.AuthorizedInfo, mailID string) error {
	return updateMailField(auth, mailID, "starred", true)
}

// Unstar removes star from a mail
func Unstar(ctx context.Context, auth *process.AuthorizedInfo, mailID string) error {
	return updateMailField(auth, mailID, "starred", false)
}

// Pin marks a mail as pinned
func Pin(ctx context.Context, auth *process.AuthorizedInfo, mailID string) error {
	return updateMailField(auth, mailID, "pinned", true)
}

// Unpin removes pin from a mail
func Unpin(ctx context.Context, auth *process.AuthorizedInfo, mailID string) error {
	return updateMailField(auth, mailID, "pinned", false)
}

func updateMailField(auth *process.AuthorizedInfo, mailID, field string, value interface{}) error {
	affected, err := capsule.Global.Query().Table("agent_mail").
		Where("mail_id", "=", mailID).
		Where("__yao_created_by", "=", auth.UserID).
		WhereNull("deleted_at").
		Update(map[string]interface{}{
			field:        value,
			"updated_at": time.Now(),
		})
	if err != nil {
		return fmt.Errorf("inbox.%s: %w", field, err)
	}
	if affected == 0 {
		return fmt.Errorf("inbox.%s: mail %s not found", field, mailID)
	}
	return nil
}

func applyInboxFilters(qb query.Query, q *ListQuery) {
	switch q.Filter {
	case "all":
		qb.Where("archived", "=", false)
	case "unread":
		qb.Where("read", "=", false).Where("archived", "=", false)
	case "starred":
		qb.Where("starred", "=", true).Where("archived", "=", false)
	case "input":
		qb.Where("type", "=", "input").Where("archived", "=", false)
	case "completed":
		qb.Where("type", "=", "completed").Where("archived", "=", false)
	case "failed":
		qb.Where("type", "=", "failed").Where("archived", "=", false)
	case "archived":
		qb.Where("archived", "=", true)
	}
	if q.Keyword != "" {
		like := "%" + q.Keyword + "%"
		qb.Where(func(sub query.Query) {
			sub.Where("title", "like", like).
				OrWhere("body", "like", like)
		})
	}
}

func rowToMail(row map[string]interface{}) *AgentMail {
	m := &AgentMail{
		MailID:      getString(row, "mail_id"),
		Type:        getString(row, "type"),
		Priority:    getString(row, "priority"),
		Title:       getString(row, "title"),
		Body:        getString(row, "body"),
		ChatID:      getString(row, "chat_id"),
		AssistantID: getString(row, "assistant_id"),
		SourceType:  getString(row, "source_type"),
		SourceID:    getString(row, "source_id"),
		SourceName:  getString(row, "source_name"),
		Read:        getBool(row, "read"),
		Archived:    getBool(row, "archived"),
		Starred:     getBool(row, "starred"),
		Pinned:      getBool(row, "pinned"),
	}
	if v := getTime(row, "read_at"); v != nil {
		m.ReadAt = v
	}
	if v := getTime(row, "created_at"); v != nil {
		m.CreatedAt = v
	}
	if v := getTime(row, "updated_at"); v != nil {
		m.UpdatedAt = v
	}
	return m
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
