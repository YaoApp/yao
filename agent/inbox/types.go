package inbox

import "time"

// ListQuery parameters for listing inbox messages
type ListQuery struct {
	Filter  string `json:"filter,omitempty"`  // all | unread | bookmarked | input | completed | failed | archived
	Keyword string `json:"keyword,omitempty"` // search title/body
	ChatID  string `json:"chat_id,omitempty"` // filter by task chat_id
	Page    int    `json:"page,omitempty"`
	Size    int    `json:"size,omitempty"`   // default 20
	Locale  string `json:"locale,omitempty"` // e.g. "zh-cn" for i18n assistant name
}

// ListResult paginated inbox list response
type ListResult struct {
	Mails []*AgentMail `json:"mails"`
	Total int64        `json:"total"`
	Page  int          `json:"page"`
	Size  int          `json:"size"`
}

// InboxStats represents category counts for sidebar display (task-based)
type InboxStats struct {
	All        int `json:"all"`
	Bookmarked int `json:"bookmarked"`
	Input      int `json:"input"`
	Completed  int `json:"completed"`
	Failed     int `json:"failed"`
	Archived   int `json:"archived"`
}

// AgentMail represents an inbox item (latest mail merged with task-level fields)
type AgentMail struct {
	ID          int64      `json:"id,omitempty"`
	MailID      string     `json:"mail_id"`
	Type        string     `json:"type"`
	Priority    string     `json:"priority"`
	Title       string     `json:"title"`
	Body        string     `json:"body,omitempty"`
	ChatID      string     `json:"chat_id"`
	ChatTitle   string     `json:"chat_title,omitempty"`
	AssistantID string     `json:"assistant_id,omitempty"`
	SourceType  string     `json:"source_type,omitempty"`
	SourceID    string     `json:"source_id,omitempty"`
	SourceName  string     `json:"source_name,omitempty"`
	Metadata    any        `json:"metadata,omitempty"`
	CreatedAt   *time.Time `json:"created_at,omitempty"`
	UpdatedAt   *time.Time `json:"updated_at,omitempty"`
	// Task-level fields (merged from task table)
	Bookmarked    bool       `json:"bookmarked"`
	InboxPinned   bool       `json:"inbox_pinned"`
	HasUnread     bool       `json:"has_unread"`
	InboxReadAt   *time.Time `json:"inbox_read_at,omitempty"`
	RunStatus     string     `json:"run_status,omitempty"`
	AssistantName string     `json:"assistant_name,omitempty"`
	WorkspaceID   string     `json:"workspace_id,omitempty"`
	WorkspaceName string     `json:"workspace_name,omitempty"`
}

// AgentTask minimal task info needed by trigger
type AgentTask struct {
	ChatID      string
	AssistantID string
	ColumnID    string
	CreatedBy   string
	TeamID      string
	DeletedAt   *time.Time
}
