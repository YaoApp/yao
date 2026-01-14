package api

import (
	"time"

	"github.com/yaoapp/yao/agent/robot/types"
)

// ==================== CRUD ====================

// Get returns a robot by member ID
// Stub: returns error (will be implemented in Phase 10)
func Get(ctx *types.Context, memberID string) (*types.Robot, error) {
	return nil, types.ErrRobotNotFound
}

// List returns robots with pagination and filtering
// Stub: returns empty result (will be implemented in Phase 10)
func List(ctx *types.Context, query *ListQuery) (*ListResult, error) {
	return &ListResult{
		Data:     []*types.Robot{},
		Total:    0,
		Page:     query.Page,
		PageSize: query.PageSize,
	}, nil
}

// Create creates a new robot member
// Stub: returns error (will be implemented in Phase 10)
func Create(ctx *types.Context, teamID string, req *CreateRequest) (*types.Robot, error) {
	return nil, types.ErrRobotNotFound
}

// Update updates robot config
// Stub: returns error (will be implemented in Phase 10)
func Update(ctx *types.Context, memberID string, req *UpdateRequest) (*types.Robot, error) {
	return nil, types.ErrRobotNotFound
}

// Remove deletes a robot member
// Stub: returns error (will be implemented in Phase 10)
func Remove(ctx *types.Context, memberID string) error {
	return types.ErrRobotNotFound
}

// ==================== Status ====================

// Status returns current robot runtime state
// Stub: returns empty state (will be implemented in Phase 10)
func Status(ctx *types.Context, memberID string) (*RobotState, error) {
	return &RobotState{
		MemberID: memberID,
		Status:   types.RobotIdle,
		Running:  0,
	}, nil
}

// UpdateStatus updates robot status (idle, paused, etc.)
// Stub: returns nil (will be implemented in Phase 10)
func UpdateStatus(ctx *types.Context, memberID string, status types.RobotStatus) error {
	return nil
}

// ==================== Trigger ====================

// Trigger starts execution with specified trigger type and request
// Stub: returns empty result (will be implemented in Phase 10)
func Trigger(ctx *types.Context, memberID string, req *TriggerRequest) (*TriggerResult, error) {
	return &TriggerResult{
		Accepted: false,
		Message:  "not implemented",
	}, nil
}

// ==================== Execution ====================

// GetExecutions returns execution history
// Stub: returns empty result (will be implemented in Phase 10)
func GetExecutions(ctx *types.Context, memberID string, query *ExecutionQuery) (*ExecutionResult, error) {
	return &ExecutionResult{
		Data:     []*types.Execution{},
		Total:    0,
		Page:     query.Page,
		PageSize: query.PageSize,
	}, nil
}

// GetExecution returns a specific execution by ID
// Stub: returns nil (will be implemented in Phase 10)
func GetExecution(ctx *types.Context, execID string) (*types.Execution, error) {
	return nil, types.ErrRobotNotFound
}

// Pause pauses a running execution
// Stub: returns nil (will be implemented in Phase 10)
func Pause(ctx *types.Context, execID string) error {
	return nil
}

// Resume resumes a paused execution
// Stub: returns nil (will be implemented in Phase 10)
func Resume(ctx *types.Context, execID string) error {
	return nil
}

// Stop stops a running execution
// Stub: returns nil (will be implemented in Phase 10)
func Stop(ctx *types.Context, execID string) error {
	return nil
}

// ==================== API Types ====================

// CreateRequest - request for Create()
type CreateRequest struct {
	DisplayName  string        `json:"display_name"`
	SystemPrompt string        `json:"system_prompt,omitempty"`
	Config       *types.Config `json:"robot_config"`
}

// UpdateRequest - request for Update()
type UpdateRequest struct {
	DisplayName  *string       `json:"display_name,omitempty"`
	SystemPrompt *string       `json:"system_prompt,omitempty"`
	Config       *types.Config `json:"robot_config,omitempty"`
}

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

// TriggerRequest - request for Trigger()
type TriggerRequest struct {
	Type types.TriggerType `json:"type"` // human | event

	// Human intervention fields (when Type = human)
	Action         types.InterventionAction `json:"action,omitempty"`
	Messages       []interface{}            `json:"messages,omitempty"` // context.Message
	PlanAt         *time.Time               `json:"plan_at,omitempty"`
	InsertPosition types.InsertPosition     `json:"insert_at,omitempty"`
	AtIndex        int                      `json:"at_index,omitempty"`

	// Event fields (when Type = event)
	Source    types.EventSource      `json:"source,omitempty"`
	EventType string                 `json:"event_type,omitempty"`
	Data      map[string]interface{} `json:"data,omitempty"`
}

// TriggerResult - result of Trigger()
type TriggerResult struct {
	Accepted  bool             `json:"accepted"`
	Queued    bool             `json:"queued"`
	Execution *types.Execution `json:"execution,omitempty"`
	JobID     string           `json:"job_id,omitempty"`
	Message   string           `json:"message,omitempty"`
}

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
