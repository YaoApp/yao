package inbox

import "time"

// ListQuery parameters for listing inbox messages
type ListQuery struct {
	Filter  string `json:"filter,omitempty"`  // all | unread | starred | input | completed | failed | archived
	Keyword string `json:"keyword,omitempty"` // search title/body
	Page    int    `json:"page,omitempty"`
	Size    int    `json:"size,omitempty"` // default 20
}

// ListResult paginated inbox list response
type ListResult struct {
	Mails []*AgentMail `json:"mails"`
	Total int64        `json:"total"`
	Page  int          `json:"page"`
	Size  int          `json:"size"`
}

// Counts unread counts grouped by type
type Counts struct {
	Total     int `json:"total"`
	Input     int `json:"input"`
	Completed int `json:"completed"`
	Failed    int `json:"failed"`
}

// AgentMail represents an inbox message
type AgentMail struct {
	ID          int64      `json:"id,omitempty"`
	MailID      string     `json:"mail_id"`
	Type        string     `json:"type"`
	Priority    string     `json:"priority"`
	Title       string     `json:"title"`
	Body        string     `json:"body,omitempty"`
	ChatID      string     `json:"chat_id"`
	AssistantID string     `json:"assistant_id,omitempty"`
	SourceType  string     `json:"source_type,omitempty"`
	SourceID    string     `json:"source_id,omitempty"`
	SourceName  string     `json:"source_name,omitempty"`
	Read        bool       `json:"read"`
	Archived    bool       `json:"archived"`
	Starred     bool       `json:"starred"`
	Pinned      bool       `json:"pinned"`
	ReadAt      *time.Time `json:"read_at,omitempty"`
	Metadata    any        `json:"metadata,omitempty"`
	CreatedAt   *time.Time `json:"created_at,omitempty"`
	UpdatedAt   *time.Time `json:"updated_at,omitempty"`
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
