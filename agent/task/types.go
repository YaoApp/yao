package task

import (
	"time"

	"github.com/yaoapp/yao/agent/output/message"
)

// ListQuery parameters for listing tasks
type ListQuery struct {
	RunStatus   string `json:"run_status,omitempty"`
	AssistantID string `json:"assistant_id,omitempty"`
	BoardID     string `json:"board_id,omitempty"`
	Page        int    `json:"page,omitempty"`
	PageSize    int    `json:"page_size,omitempty"`
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
	ID           int64      `json:"id,omitempty"`
	ChatID       string     `json:"chat_id"`
	ColumnID     *string    `json:"column_id"`
	Position     int        `json:"position"`
	Pinned       bool       `json:"pinned"`
	Priority     string     `json:"priority"`
	Tags         []string   `json:"tags,omitempty"`
	RunStatus    string     `json:"run_status"`
	Progress     int        `json:"progress"`
	CurrentStep  *string    `json:"current_step,omitempty"`
	ErrorMessage *string    `json:"error_message,omitempty"`
	StartedAt    *time.Time `json:"started_at,omitempty"`
	CompletedAt  *time.Time `json:"completed_at,omitempty"`
	Duration     int        `json:"duration"`
	RunCount     int        `json:"run_count"`
	ComputerID   *string    `json:"computer_id,omitempty"`
	ComputerMode *string    `json:"computer_mode,omitempty"`
	SandboxType  *string    `json:"sandbox_type,omitempty"`
	Metadata     any        `json:"metadata,omitempty"`
	CreatedAt    *time.Time `json:"created_at,omitempty"`
	UpdatedAt    *time.Time `json:"updated_at,omitempty"`

	// Derived from JOINs
	Title         string  `json:"title,omitempty"`
	AssistantID   string  `json:"assistant_id,omitempty"`
	AssistantName string  `json:"assistant_name,omitempty"`
	LastWorkspace *string `json:"last_workspace,omitempty"`
	LastConnector *string `json:"last_connector,omitempty"`
	BoardID       *string `json:"board_id,omitempty"`
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

// --- Config types (Plan 2) ---

// Config is the response from GetConfig, containing merged settings and metadata
type Config struct {
	Setting        *TaskSetting      `json:"setting"`
	ResolvedFrom   map[string]string `json:"_resolved_from,omitempty"`
	ScheduleStatus *ScheduleStatus   `json:"_schedule_status,omitempty"`
}

// ScheduleStatus provides runtime schedule info (populated by ScheduleEngine in Plan 3)
type ScheduleStatus struct {
	LastRun   *time.Time `json:"last_run,omitempty"`
	NextRun   *time.Time `json:"next_run,omitempty"`
	TotalRuns int        `json:"total_runs"`
}

// ConfigReq is the request body for SetConfig (partial update)
type ConfigReq struct {
	Runner   *string            `json:"runner,omitempty"`
	Model    *string            `json:"model,omitempty"`
	Image    *string            `json:"image,omitempty"`
	Timeout  *string            `json:"timeout,omitempty"`
	MaxTurns *int               `json:"max_turns,omitempty"`
	Secrets  map[string]*string `json:"secrets,omitempty"`
	Services []ServiceDecl      `json:"services,omitempty"`
	Skills   []string           `json:"skills,omitempty"`
	Schedule *ScheduleConfig    `json:"schedule,omitempty"`
}

// TaskSetting represents the merged task configuration across all layers
type TaskSetting struct {
	Runner   string            `json:"runner,omitempty"`
	Model    string            `json:"model,omitempty"`
	Image    string            `json:"image,omitempty"`
	Timeout  string            `json:"timeout,omitempty"`
	MaxTurns int               `json:"max_turns,omitempty"`
	Secrets  map[string]string `json:"secrets,omitempty"`
	Services []ServiceDecl     `json:"services,omitempty"`
	Skills   []string          `json:"skills,omitempty"`
	Schedule *ScheduleConfig   `json:"schedule,omitempty"`
}

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
	Metadata    map[string]any `json:"metadata,omitempty"`
	Priority    int            `json:"priority,omitempty"`
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
	Type        string         `json:"type"`
	Messages    []InputMessage `json:"messages,omitempty"`
	AssistantID string         `json:"assistant_id,omitempty"`
	Metadata    map[string]any `json:"metadata,omitempty"`
	Priority    int            `json:"priority,omitempty"`
	Force       bool           `json:"force,omitempty"`
}

// InputMessage is the message type for task execution input
type InputMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}
