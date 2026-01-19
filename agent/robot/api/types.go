package api

import (
	"time"

	agentcontext "github.com/yaoapp/yao/agent/context"
	"github.com/yaoapp/yao/agent/robot/types"
)

// ListQuery - query options for List()
type ListQuery struct {
	TeamID    string            `json:"team_id,omitempty"`
	Status    types.RobotStatus `json:"status,omitempty"`
	Keywords  string            `json:"keywords,omitempty"`
	ClockMode types.ClockMode   `json:"clock_mode,omitempty"`
	Page      int               `json:"page,omitempty"`
	PageSize  int               `json:"pagesize,omitempty"`
	Order     string            `json:"order,omitempty"`
}

// ListResult - result of List()
type ListResult struct {
	Data     []*types.Robot `json:"data"`
	Total    int            `json:"total"`
	Page     int            `json:"page"`
	PageSize int            `json:"pagesize"`
}

// RobotState - runtime state from Status()
type RobotState struct {
	MemberID    string            `json:"member_id"`
	TeamID      string            `json:"team_id"`
	DisplayName string            `json:"display_name"`
	Status      types.RobotStatus `json:"status"`
	Running     int               `json:"running"`
	MaxRunning  int               `json:"max_running"`
	LastRun     *time.Time        `json:"last_run,omitempty"`
	NextRun     *time.Time        `json:"next_run,omitempty"`
	RunningIDs  []string          `json:"running_ids,omitempty"`
}

// ==================== Trigger Types ====================

// TriggerRequest - request for Trigger()
// Input uses []context.Message to support rich content (text, images, files, audio)
type TriggerRequest struct {
	Type types.TriggerType `json:"type"` // human | event | clock

	// Human intervention fields (when Type = human)
	Action         types.InterventionAction `json:"action,omitempty"`
	Messages       []agentcontext.Message   `json:"messages,omitempty"` // user's input (supports text, images, files)
	PlanAt         *time.Time               `json:"plan_at,omitempty"`
	InsertPosition InsertPosition           `json:"insert_at,omitempty"`
	AtIndex        int                      `json:"at_index,omitempty"`

	// Event fields (when Type = event)
	Source    types.EventSource      `json:"source,omitempty"`
	EventType string                 `json:"event_type,omitempty"`
	Data      map[string]interface{} `json:"data,omitempty"`

	// Executor mode (optional, overrides robot config)
	ExecutorMode types.ExecutorMode `json:"executor_mode,omitempty"`
}

// InsertPosition - where to insert task in queue
type InsertPosition string

const (
	// InsertFirst inserts at beginning (highest priority)
	InsertFirst InsertPosition = "first"
	// InsertLast appends at end (default)
	InsertLast InsertPosition = "last"
	// InsertNext inserts after current task
	InsertNext InsertPosition = "next"
	// InsertAt inserts at specific index (use AtIndex)
	InsertAt InsertPosition = "at"
)

// TriggerResult - result of Trigger()
type TriggerResult struct {
	Accepted  bool             `json:"accepted"`
	Queued    bool             `json:"queued"`
	Execution *types.Execution `json:"execution,omitempty"`
	JobID     string           `json:"job_id,omitempty"`
	Message   string           `json:"message,omitempty"`
}

// ==================== Execution Types ====================

// ExecutionQuery - query options for GetExecutions()
type ExecutionQuery struct {
	Status   types.ExecStatus  `json:"status,omitempty"`
	Trigger  types.TriggerType `json:"trigger,omitempty"`
	Page     int               `json:"page,omitempty"`
	PageSize int               `json:"pagesize,omitempty"`
}

// ExecutionResult - result of GetExecutions()
type ExecutionResult struct {
	Data     []*types.Execution `json:"data"`
	Total    int                `json:"total"`
	Page     int                `json:"page"`
	PageSize int                `json:"pagesize"`
}

// ==================== Helper Functions ====================

// applyDefaults applies default values to ListQuery
func (q *ListQuery) applyDefaults() {
	if q.Page <= 0 {
		q.Page = 1
	}
	if q.PageSize <= 0 {
		q.PageSize = 20
	}
	if q.PageSize > 100 {
		q.PageSize = 100
	}
}

// applyDefaults applies default values to ExecutionQuery
func (q *ExecutionQuery) applyDefaults() {
	if q.Page <= 0 {
		q.Page = 1
	}
	if q.PageSize <= 0 {
		q.PageSize = 20
	}
	if q.PageSize > 100 {
		q.PageSize = 100
	}
}
