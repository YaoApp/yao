package task

import (
	"time"

	"github.com/yaoapp/yao/agent/output/message"
)

// ListQuery parameters for listing tasks
type ListQuery struct {
	RunStatus     string `json:"run_status,omitempty"`
	ArchiveStatus string `json:"archive_status,omitempty"`
	AssistantID   string `json:"assistant_id,omitempty"`
	BoardID       string `json:"board_id,omitempty"`
	Page          int    `json:"page,omitempty"`
	PageSize      int    `json:"page_size,omitempty"`
	Locale        string `json:"locale,omitempty"`
}

// ListResult paginated task list response
type ListResult struct {
	Tasks    []*Task `json:"tasks"`
	Total    int64   `json:"total"`
	Page     int     `json:"page"`
	PageSize int     `json:"page_size"`
}

// Task represents a full task with derived fields from JOINs
type Task struct {
	ID            int64      `json:"id,omitempty"`
	ChatID        string     `json:"chat_id"`
	ColumnID      *string    `json:"column_id"`
	Position      int        `json:"position"`
	Pinned        bool       `json:"pinned"`
	Priority      string     `json:"priority"`
	Tags          []string   `json:"tags,omitempty"`
	RunStatus     string     `json:"run_status"`
	ArchiveStatus string     `json:"archive_status,omitempty"`
	Progress      int        `json:"progress"`
	CurrentStep   *string    `json:"current_step,omitempty"`
	ErrorMessage  *string    `json:"error_message,omitempty"`
	StartedAt     *time.Time `json:"started_at,omitempty"`
	CompletedAt   *time.Time `json:"completed_at,omitempty"`
	Duration      int        `json:"duration"`
	RunCount      int        `json:"run_count"`
	ComputerID    *string    `json:"computer_id,omitempty"`
	ComputerMode  *string    `json:"computer_mode,omitempty"`
	SandboxType   *string    `json:"sandbox_type,omitempty"`
	Instruction   string     `json:"instruction,omitempty"`
	Summary       string     `json:"summary,omitempty"`
	Outputs       any        `json:"outputs,omitempty"`
	Metadata      any        `json:"metadata,omitempty"`
	CreatedAt     *time.Time `json:"created_at,omitempty"`
	UpdatedAt     *time.Time `json:"updated_at,omitempty"`

	// Derived from JOINs
	Title         string  `json:"title,omitempty"`
	AssistantID   string  `json:"assistant_id,omitempty"`
	AssistantName string  `json:"assistant_name,omitempty"`
	LastWorkspace *string `json:"last_workspace,omitempty"`
	LastConnector *string `json:"last_connector,omitempty"`
	BoardID       *string `json:"board_id,omitempty"`

	// Resolved at query time
	WorkspaceName string `json:"workspace_name,omitempty"`
}

// CreateReq parameters for creating a task
type CreateReq struct {
	ChatID      string `json:"chat_id,omitempty"`
	Title       string `json:"title"`
	AssistantID string `json:"assistant_id"`
	ColumnID    string `json:"column_id"`
}

// UpdateReq parameters for partially updating a task
type UpdateReq struct {
	Title         *string  `json:"title,omitempty"`
	AssistantID   *string  `json:"assistant_id,omitempty"`
	ColumnID      *string  `json:"column_id,omitempty"`
	Pinned        *bool    `json:"pinned,omitempty"`
	Priority      *string  `json:"priority,omitempty"`
	Tags          []string `json:"tags,omitempty"`
	LastWorkspace *string  `json:"last_workspace,omitempty"`
	ComputerID    *string  `json:"computer_id,omitempty"`
	ComputerMode  *string  `json:"computer_mode,omitempty"`
	SandboxType   *string  `json:"sandbox_type,omitempty"`
	Instruction   *string  `json:"instruction,omitempty"`
	Summary       *string  `json:"summary,omitempty"`
	Outputs       any      `json:"outputs,omitempty"`
	Metadata      any      `json:"metadata,omitempty"`
}

// MoveReq parameters for moving a task between columns
type MoveReq struct {
	ColumnID string `json:"column_id"`
	Position int    `json:"position"`
}

// CreateFromWSReq for creating task from WS first message.
// Task parameters are passed via Metadata (consistent with Stream/Interact interface).
type CreateFromWSReq struct {
	ChatID   string         `json:"chat_id"`
	Title    string         `json:"title"`
	Metadata map[string]any `json:"metadata,omitempty"`
}

// --- Config types (kept for schedule system) ---

// ServiceDecl declares a service exposed by the task container
type ServiceDecl struct {
	Name     string `json:"name"`
	Port     int    `json:"port"`
	Protocol string `json:"protocol"`
	Public   bool   `json:"public"`
}

// ScheduleConfig defines task scheduling parameters
type ScheduleConfig struct {
	Enabled       bool     `json:"enabled"`
	Mode          string   `json:"mode"`
	Times         []string `json:"times,omitempty"`
	Days          []string `json:"days,omitempty"`
	IntervalValue int      `json:"interval_value,omitempty"`
	IntervalUnit  string   `json:"interval_unit,omitempty"`
	Timezone      string   `json:"timezone,omitempty"`
	StartDate     string   `json:"start_date,omitempty"`
	EndDate       string   `json:"end_date,omitempty"`
}

// --- Execution types (Plan 3) ---

// RunReq parameters for running a task
type RunReq struct {
	Messages    []InputMessage `json:"messages"`
	AssistantID string         `json:"assistant_id,omitempty"`
	Model       string         `json:"model,omitempty"` // connector ID from user selection (overrides config)
	Metadata    map[string]any `json:"metadata,omitempty"`
	Priority    int            `json:"priority,omitempty"`
	Source      string         `json:"source,omitempty"` // "run", "retry", "repeat"
	Fresh       bool           `json:"fresh,omitempty"`  // true = skip chat history loading (retry from scratch)
	Locale      string         `json:"locale,omitempty"` // e.g. "zh-cn" for i18n
}

// RunResult is the response from Run()
type RunResult struct {
	ChatID    string `json:"chat_id"`
	Status    string `json:"status"`
	RequestID string `json:"request_id,omitempty"`
	Position  int    `json:"position,omitempty"`
}

// InputReq parameters for providing user input to a waiting task
type InputReq struct {
	Messages []InputMessage `json:"messages"`
}

// WatchOpts configures the Watch stream (historical + live messages)
type WatchOpts struct {
	AfterSeq int64  `json:"after_seq,omitempty"` // Retrieve messages after this sequence number (0 = from start)
	BeforeID int64  `json:"before_id,omitempty"` // Load messages with id < BeforeID (for load-more)
	Limit    int    `json:"limit,omitempty"`     // Max number of historical messages to return (0 = all)
	Locale   string `json:"locale,omitempty"`    // Locale for assistant info (e.g. "zh-cn")
}

// WatchStream represents an active Watch stream
type WatchStream struct {
	Ch       <-chan *message.Message
	Cancel   func()
	LiveMode bool // true = subscribed to live daemon; false = DB-only replay
}

// --- Legacy SubscribeOpts kept for backward compatibility during transition ---

// SubscribeOpts configures how a subscription replays history
type SubscribeOpts struct {
	Replay    ReplayMode `json:"replay"`
	AfterSeq  int64      `json:"after_seq,omitempty"`
	RequestID string     `json:"request_id,omitempty"`
}

// ReplayMode defines message replay behavior
type ReplayMode int

const (
	ReplayAll   ReplayMode = iota // Replay all buffered messages then live
	ReplayNone                    // No replay, live only
	ReplayAfter                   // Replay from AfterSeq+1 then live
)

// Subscription represents an active message subscription
type Subscription struct {
	Ch     <-chan *message.Message
	Cancel func()
}

// WSCommand represents a client-to-server WebSocket command
type WSCommand struct {
	Type        string         `json:"type"`                   // "read", "history", "run", "retry", "repeat", "stop", "cancel"
	Messages    []InputMessage `json:"messages,omitempty"`     // For run/retry: user messages
	AssistantID string         `json:"assistant_id,omitempty"` // For run: assistant to use
	Model       string         `json:"model,omitempty"`        // For run: connector ID from user selection
	Metadata    map[string]any `json:"metadata,omitempty"`     // For run: task metadata
	Priority    int            `json:"priority,omitempty"`     // For run: priority
	Force       bool           `json:"force,omitempty"`        // Reserved
	Since       int64          `json:"since,omitempty"`        // For read: after_seq pagination cursor (legacy, kept for compat)
	Before      int64          `json:"before,omitempty"`       // For read: load messages with id < before (load-more cursor)
	Limit       int            `json:"limit,omitempty"`        // For read/history: max messages to return
	Locale      string         `json:"locale,omitempty"`       // For read: locale for assistant info
}

// InputMessage is the message type for task execution input.
// Content is interface{} to support both plain text (string) and multipart
// content ([]ContentPart) consistent with agentcontext.Message.
type InputMessage struct {
	Role    string      `json:"role"`
	Content interface{} `json:"content"`
}
