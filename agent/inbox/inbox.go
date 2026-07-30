package inbox

import (
	"context"
	"fmt"
	"time"

	"github.com/yaoapp/gou/process"
	"github.com/yaoapp/xun"
	"github.com/yaoapp/xun/capsule"
	"github.com/yaoapp/xun/dbal/query"
)

// List returns a paginated list of inbox items (task-based with latest mail)
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

	// Step 1: Count tasks
	countQB := capsule.Global.Query()
	countQB.Table(tableTask()+" as t").
		Where("t.__yao_created_by", "=", auth.UserID).
		Where("t.__yao_team_id", "=", auth.TeamID).
		WhereNull("t.deleted_at")
	applyTaskFilters(countQB, q)

	total, err := countQB.Count()
	if err != nil {
		return nil, fmt.Errorf("inbox.List count: %w", err)
	}

	// Step 1b: Fetch task rows
	taskQB := capsule.Global.Query()
	taskQB.Table(tableTask()+" as t").
		Select("t.chat_id", "t.bookmarked", "t.inbox_pinned", "t.has_unread", "t.inbox_read_at", "t.last_mail_type", "t.updated_at").
		Where("t.__yao_created_by", "=", auth.UserID).
		Where("t.__yao_team_id", "=", auth.TeamID).
		WhereNull("t.deleted_at")
	applyTaskFilters(taskQB, q)

	offset := (q.Page - 1) * q.Size
	tasks, err := taskQB.
		OrderBy("t.inbox_pinned", "desc").
		OrderBy("t.has_unread", "desc").
		OrderBy("t.updated_at", "desc").
		Offset(offset).Limit(q.Size).Get()
	if err != nil {
		return nil, fmt.Errorf("inbox.List query: %w", err)
	}

	if len(tasks) == 0 {
		return &ListResult{Mails: []*AgentMail{}, Total: int64(total), Page: q.Page, Size: q.Size}, nil
	}

	// Step 2: Batch fetch latest mail per task
	chatIDs := extractChatIDs(tasks)
	mailRows, err := capsule.Global.Query().Table(tableMail()).
		Select("*").
		WhereIn("chat_id", chatIDs).
		Where("__yao_created_by", "=", auth.UserID).
		Where("__yao_team_id", "=", auth.TeamID).
		WhereNull("deleted_at").
		OrderBy("created_at", "desc").Get()
	if err != nil {
		return nil, fmt.Errorf("inbox.List mails: %w", err)
	}

	// Dedup: keep only the latest mail per chat_id
	latestMailMap := make(map[string]xun.R, len(chatIDs))
	for _, row := range mailRows {
		cid := getString(row, "chat_id")
		if cid == "" {
			continue
		}
		if _, exists := latestMailMap[cid]; !exists {
			latestMailMap[cid] = row
		}
	}

	// Merge: build result in task order
	mails := make([]*AgentMail, 0, len(tasks))
	for _, t := range tasks {
		cid := getString(t, "chat_id")
		mailRow, hasMail := latestMailMap[cid]
		if !hasMail {
			continue
		}
		m := rowToMail(mailRow)
		// Merge task-level fields
		m.Bookmarked = getBool(t, "bookmarked")
		m.InboxPinned = getBool(t, "inbox_pinned")
		m.HasUnread = getBool(t, "has_unread")
		m.InboxReadAt = getTime(t, "inbox_read_at")
		mails = append(mails, m)
	}

	enrichChatTitles(mails)

	return &ListResult{
		Mails: mails,
		Total: int64(total),
		Page:  q.Page,
		Size:  q.Size,
	}, nil
}

// Stats returns unread task-group counts per category for sidebar display
func Stats(ctx context.Context, auth *process.AuthorizedInfo) (*InboxStats, error) {
	stats := &InboxStats{}

	// All non-archived with unread
	row, err := capsule.Global.Query().Table(tableTask()).
		SelectRaw("COUNT(*) as cnt").
		Where("__yao_created_by", "=", auth.UserID).
		Where("__yao_team_id", "=", auth.TeamID).
		Where("has_unread", "=", true).
		WhereNotNull("last_mail_type").
		WhereNull("archive_status").
		WhereNull("deleted_at").
		First()
	if err != nil {
		return nil, fmt.Errorf("inbox.Stats all: %w", err)
	}
	if row != nil {
		stats.All = getInt(row, "cnt")
	}

	// By type (non-archived with unread)
	rows, err := capsule.Global.Query().Table(tableTask()).
		Select("last_mail_type").
		SelectRaw("COUNT(*) as cnt").
		Where("__yao_created_by", "=", auth.UserID).
		Where("__yao_team_id", "=", auth.TeamID).
		Where("has_unread", "=", true).
		WhereNull("archive_status").
		WhereNull("deleted_at").
		GroupBy("last_mail_type").
		Get()
	if err != nil {
		return nil, fmt.Errorf("inbox.Stats by type: %w", err)
	}
	for _, r := range rows {
		cnt := getInt(r, "cnt")
		switch getString(r, "last_mail_type") {
		case "input":
			stats.Input = cnt
		case "completed":
			stats.Completed = cnt
		case "failed":
			stats.Failed = cnt
		}
	}

	// Bookmarked with unread (non-archived)
	bookmarkedRow, err := capsule.Global.Query().Table(tableTask()).
		SelectRaw("COUNT(*) as cnt").
		Where("__yao_created_by", "=", auth.UserID).
		Where("__yao_team_id", "=", auth.TeamID).
		Where("has_unread", "=", true).
		Where("bookmarked", "=", true).
		WhereNotNull("last_mail_type").
		WhereNull("archive_status").
		WhereNull("deleted_at").
		First()
	if err != nil {
		return nil, fmt.Errorf("inbox.Stats bookmarked: %w", err)
	}
	if bookmarkedRow != nil {
		stats.Bookmarked = getInt(bookmarkedRow, "cnt")
	}

	// Archived with unread
	archivedRow, err := capsule.Global.Query().Table(tableTask()).
		SelectRaw("COUNT(*) as cnt").
		Where("__yao_created_by", "=", auth.UserID).
		Where("__yao_team_id", "=", auth.TeamID).
		Where("has_unread", "=", true).
		WhereNotNull("last_mail_type").
		WhereNotNull("archive_status").
		WhereNull("deleted_at").
		First()
	if err != nil {
		return nil, fmt.Errorf("inbox.Stats archived: %w", err)
	}
	if archivedRow != nil {
		stats.Archived = getInt(archivedRow, "cnt")
	}

	return stats, nil
}

// UnreadCount returns total number of tasks with unread notifications
func UnreadCount(ctx context.Context, auth *process.AuthorizedInfo) (int, error) {
	row, err := capsule.Global.Query().Table(tableTask()).
		SelectRaw("COUNT(*) as cnt").
		Where("__yao_created_by", "=", auth.UserID).
		Where("__yao_team_id", "=", auth.TeamID).
		Where("has_unread", "=", true).
		WhereNotNull("last_mail_type").
		WhereNull("archive_status").
		WhereNull("deleted_at").
		First()
	if err != nil {
		return 0, fmt.Errorf("inbox.UnreadCount: %w", err)
	}
	if row == nil {
		return 0, nil
	}
	return getInt(row, "cnt"), nil
}

// View marks a task as viewed in inbox (clears unread, sets inbox_read_at)
func View(ctx context.Context, auth *process.AuthorizedInfo, chatID string) error {
	now := time.Now()
	affected, err := capsule.Global.Query().Table(tableTask()).
		Where("chat_id", "=", chatID).
		Where("__yao_created_by", "=", auth.UserID).
		Where("__yao_team_id", "=", auth.TeamID).
		WhereNull("deleted_at").
		Update(map[string]interface{}{
			"has_unread":    false,
			"inbox_read_at": now,
		})
	if err != nil {
		return fmt.Errorf("inbox.View: %w", err)
	}
	if affected == 0 {
		return fmt.Errorf("inbox.View: task with chat_id %s not found", chatID)
	}
	return nil
}

// ReadAll marks all tasks as read (batch clear has_unread)
func ReadAll(ctx context.Context, auth *process.AuthorizedInfo) (int64, error) {
	now := time.Now()
	affected, err := capsule.Global.Query().Table(tableTask()).
		Where("__yao_created_by", "=", auth.UserID).
		Where("__yao_team_id", "=", auth.TeamID).
		Where("has_unread", "=", true).
		WhereNotNull("last_mail_type").
		WhereNull("deleted_at").
		Update(map[string]interface{}{
			"has_unread":    false,
			"inbox_read_at": now,
		})
	if err != nil {
		return 0, fmt.Errorf("inbox.ReadAll: %w", err)
	}
	return int64(affected), nil
}

// Bookmark marks a task as bookmarked
func Bookmark(ctx context.Context, auth *process.AuthorizedInfo, chatID string) error {
	return updateTaskField(auth, chatID, "bookmarked", true)
}

// Unbookmark removes bookmark from a task
func Unbookmark(ctx context.Context, auth *process.AuthorizedInfo, chatID string) error {
	return updateTaskField(auth, chatID, "bookmarked", false)
}

// Pin marks a task as pinned in inbox
func Pin(ctx context.Context, auth *process.AuthorizedInfo, chatID string) error {
	return updateTaskField(auth, chatID, "inbox_pinned", true)
}

// Unpin removes inbox pin from a task
func Unpin(ctx context.Context, auth *process.AuthorizedInfo, chatID string) error {
	return updateTaskField(auth, chatID, "inbox_pinned", false)
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

func updateTaskField(auth *process.AuthorizedInfo, chatID, field string, value interface{}) error {
	affected, err := capsule.Global.Query().Table(tableTask()).
		Where("chat_id", "=", chatID).
		Where("__yao_created_by", "=", auth.UserID).
		Where("__yao_team_id", "=", auth.TeamID).
		WhereNull("deleted_at").
		Update(map[string]interface{}{field: value})
	if err != nil {
		return fmt.Errorf("inbox.%s: %w", field, err)
	}
	if affected == 0 {
		return fmt.Errorf("inbox.%s: task with chat_id %s not found", field, chatID)
	}
	return nil
}

func applyTaskFilters(qb query.Query, q *ListQuery) {
	switch q.Filter {
	case "all", "none", "":
		qb.WhereNotNull("t.last_mail_type").WhereNull("t.archive_status")
	case "unread":
		qb.Where("t.has_unread", "=", true).WhereNull("t.archive_status")
	case "bookmarked":
		qb.Where("t.bookmarked", "=", true).WhereNotNull("t.last_mail_type").WhereNull("t.archive_status")
	case "input":
		qb.Where("t.last_mail_type", "=", "input").WhereNull("t.archive_status")
	case "completed":
		qb.Where("t.last_mail_type", "=", "completed").WhereNull("t.archive_status")
	case "failed":
		qb.Where("t.last_mail_type", "=", "failed").WhereNull("t.archive_status")
	case "archived":
		qb.WhereNotNull("t.last_mail_type").WhereNotNull("t.archive_status")
	}
	if q.Keyword != "" {
		like := "%" + q.Keyword + "%"
		qb.LeftJoin(tableChat()+" as c", "c.chat_id", "=", "t.chat_id")
		qb.Where(func(sub query.Query) {
			sub.Where("c.title", "like", like).
				OrWhere("t.summary", "like", like)
		})
	}
	if q.ChatID != "" {
		qb.Where("t.chat_id", "=", q.ChatID)
	}
}

func extractChatIDs(tasks []xun.R) []interface{} {
	ids := make([]interface{}, 0, len(tasks))
	for _, t := range tasks {
		if cid := getString(t, "chat_id"); cid != "" {
			ids = append(ids, cid)
		}
	}
	return ids
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

func rowToMail(row xun.R) *AgentMail {
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
	}
	if v := getTime(row, "created_at"); v != nil {
		m.CreatedAt = v
	}
	if v := getTime(row, "updated_at"); v != nil {
		m.UpdatedAt = v
	}
	return m
}

func getString(row xun.R, key string) string {
	if v, ok := row[key]; ok && v != nil {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

func getInt(row xun.R, key string) int {
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

func getBool(row xun.R, key string) bool {
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

func getTime(row xun.R, key string) *time.Time {
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
