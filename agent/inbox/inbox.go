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
		q.Filter = "none"
	}

	// Count query (JOIN task for archive_status filtering)
	countQB := capsule.Global.Query()
	countQB.Table(tableMail()+" as m").
		LeftJoin(tableTask()+" as t", "m.chat_id", "=", "t.chat_id").
		Where("m.__yao_created_by", "=", auth.UserID).
		Where("m.__yao_team_id", "=", auth.TeamID).
		WhereNull("m.deleted_at")
	applyInboxFilters(countQB, q)

	total, err := countQB.Count()
	if err != nil {
		return nil, fmt.Errorf("inbox.List count: %w", err)
	}

	// Data query (fresh builder with JOIN)
	qb := capsule.Global.Query()
	qb.Table(tableMail()+" as m").
		Select("m.*").
		LeftJoin(tableTask()+" as t", "m.chat_id", "=", "t.chat_id").
		Where("m.__yao_created_by", "=", auth.UserID).
		Where("m.__yao_team_id", "=", auth.TeamID).
		WhereNull("m.deleted_at")
	applyInboxFilters(qb, q)

	offset := (q.Page - 1) * q.Size
	rows, err := qb.
		OrderBy("m.pinned", "desc").
		OrderBy("m.read", "asc").
		OrderBy("m.created_at", "desc").
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

	enrichChatTitles(mails)

	return &ListResult{
		Mails: mails,
		Total: int64(total),
		Page:  q.Page,
		Size:  q.Size,
	}, nil
}

// Stats returns unread chat-group counts per category for sidebar display
func Stats(ctx context.Context, auth *process.AuthorizedInfo) (*InboxStats, error) {
	// Count chat groups that have at least one unread mail, grouped by type (non-archived)
	rows, err := capsule.Global.Query().Table(tableMail()+" as m").
		Select("m.type").
		SelectRaw("COUNT(DISTINCT m.chat_id) as cnt").
		LeftJoin(tableTask()+" as t", "m.chat_id", "=", "t.chat_id").
		Where("m.__yao_created_by", "=", auth.UserID).
		Where("m.__yao_team_id", "=", auth.TeamID).
		Where("m.read", "=", false).
		WhereNull("t.archive_status").
		WhereNull("m.deleted_at").
		GroupBy("m.type").
		Get()
	if err != nil {
		return nil, fmt.Errorf("inbox.Stats: %w", err)
	}

	stats := &InboxStats{}
	for _, row := range rows {
		cnt := getInt(row, "cnt")
		stats.All += cnt
		switch getString(row, "type") {
		case "input":
			stats.Input = cnt
		case "completed":
			stats.Completed = cnt
		case "failed":
			stats.Failed = cnt
		}
	}

	// Starred: chat groups with at least one unread+starred mail (non-archived)
	starredRow, err := capsule.Global.Query().Table(tableMail()+" as m").
		SelectRaw("COUNT(DISTINCT m.chat_id) as cnt").
		LeftJoin(tableTask()+" as t", "m.chat_id", "=", "t.chat_id").
		Where("m.__yao_created_by", "=", auth.UserID).
		Where("m.__yao_team_id", "=", auth.TeamID).
		Where("m.read", "=", false).
		Where("m.starred", "=", true).
		WhereNull("t.archive_status").
		WhereNull("m.deleted_at").
		First()
	if err != nil {
		return nil, fmt.Errorf("inbox.Stats starred count: %w", err)
	}
	if starredRow != nil {
		stats.Starred = getInt(starredRow, "cnt")
	}

	// Archived: chat groups with at least one unread mail belonging to archived tasks
	archivedRow, err := capsule.Global.Query().Table(tableMail()+" as m").
		SelectRaw("COUNT(DISTINCT m.chat_id) as cnt").
		LeftJoin(tableTask()+" as t", "m.chat_id", "=", "t.chat_id").
		Where("m.__yao_created_by", "=", auth.UserID).
		Where("m.__yao_team_id", "=", auth.TeamID).
		Where("m.read", "=", false).
		WhereNotNull("t.archive_status").
		WhereNull("m.deleted_at").
		First()
	if err != nil {
		return nil, fmt.Errorf("inbox.Stats archived count: %w", err)
	}
	if archivedRow != nil {
		stats.Archived = getInt(archivedRow, "cnt")
	}

	return stats, nil
}

// UnreadCount returns unread counts grouped by type (excludes mails from archived tasks)
func UnreadCount(ctx context.Context, auth *process.AuthorizedInfo) (*Counts, error) {
	rows, err := capsule.Global.Query().Table(tableMail()+" as m").
		Select("m.type").
		SelectRaw("COUNT(*) as cnt").
		LeftJoin(tableTask()+" as t", "m.chat_id", "=", "t.chat_id").
		Where("m.__yao_created_by", "=", auth.UserID).
		Where("m.__yao_team_id", "=", auth.TeamID).
		Where("m.read", "=", false).
		WhereNull("t.archive_status").
		WhereNull("m.deleted_at").
		GroupBy("m.type").
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
	affected, err := capsule.Global.Query().Table(tableMail()).
		Where("mail_id", "=", mailID).
		Where("__yao_created_by", "=", auth.UserID).
		Where("__yao_team_id", "=", auth.TeamID).
		WhereNull("deleted_at").
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
	qb := capsule.Global.Query().Table(tableMail()).
		Where("__yao_created_by", "=", auth.UserID).
		Where("__yao_team_id", "=", auth.TeamID).
		Where("read", "=", false).
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

// DeleteByChatID soft-deletes all inbox mails belonging to the given chat_id
func DeleteByChatID(ctx context.Context, auth *process.AuthorizedInfo, chatID string) (int64, error) {
	affected, err := capsule.Global.Query().Table(tableMail()).
		Where("chat_id", "=", chatID).
		Where("__yao_created_by", "=", auth.UserID).
		Where("__yao_team_id", "=", auth.TeamID).
		WhereNull("deleted_at").
		Update(map[string]interface{}{
			"deleted_at": time.Now(),
		})
	if err != nil {
		return 0, fmt.Errorf("inbox.DeleteByChatID: %w", err)
	}
	return int64(affected), nil
}

func updateMailField(auth *process.AuthorizedInfo, mailID, field string, value interface{}) error {
	affected, err := capsule.Global.Query().Table(tableMail()).
		Where("mail_id", "=", mailID).
		Where("__yao_created_by", "=", auth.UserID).
		Where("__yao_team_id", "=", auth.TeamID).
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

func enrichChatTitles(mails []*AgentMail) {
	if len(mails) == 0 {
		return
	}

	chatIDs := make([]interface{}, 0)
	seen := make(map[string]bool)
	for _, m := range mails {
		if m.ChatID != "" && !seen[m.ChatID] {
			seen[m.ChatID] = true
			chatIDs = append(chatIDs, m.ChatID)
		}
	}
	if len(chatIDs) == 0 {
		return
	}

	rows, err := capsule.Global.Query().Table(tableChat()).
		Select("chat_id", "title").
		WhereIn("chat_id", chatIDs).
		Get()
	if err != nil || len(rows) == 0 {
		return
	}

	titleMap := make(map[string]string, len(rows))
	for _, row := range rows {
		cid := getString(row, "chat_id")
		title := getString(row, "title")
		if cid != "" && title != "" {
			titleMap[cid] = title
		}
	}

	for _, m := range mails {
		if t, ok := titleMap[m.ChatID]; ok {
			m.ChatTitle = t
		}
	}
}

func applyInboxFilters(qb query.Query, q *ListQuery) {
	switch q.Filter {
	case "all":
		qb.WhereNull("t.archive_status")
	case "unread":
		qb.Where("m.read", "=", false).WhereNull("t.archive_status")
	case "starred":
		qb.Where("m.starred", "=", true).WhereNull("t.archive_status")
	case "input":
		qb.Where("m.type", "=", "input").WhereNull("t.archive_status")
	case "completed":
		qb.Where("m.type", "=", "completed").WhereNull("t.archive_status")
	case "failed":
		qb.Where("m.type", "=", "failed").WhereNull("t.archive_status")
	case "archived":
		qb.WhereNotNull("t.archive_status")
	}
	if q.Keyword != "" {
		like := "%" + q.Keyword + "%"
		qb.Where(func(sub query.Query) {
			sub.Where("m.title", "like", like).
				OrWhere("m.body", "like", like)
		})
	}
	if q.ChatID != "" {
		qb.Where("m.chat_id", "=", q.ChatID)
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
			for _, layout := range []string{
				time.RFC3339,
				time.RFC3339Nano,
				"2006-01-02 15:04:05.999999999-07:00",
				"2006-01-02 15:04:05-07:00",
				"2006-01-02 15:04:05.999999999",
				"2006-01-02 15:04:05",
			} {
				if parsed, err := time.Parse(layout, t); err == nil {
					return &parsed
				}
			}
		}
	}
	return nil
}
