package robot

import (
	"fmt"
	"time"

	robotapi "github.com/yaoapp/yao/agent/robot/api"
	robottypes "github.com/yaoapp/yao/agent/robot/types"
)

// ==================== Request Types ====================

// CreateRobotRequest - HTTP request for creating a robot
type CreateRobotRequest struct {
	// Identity (member_id is optional - auto-generated if not provided)
	MemberID string `json:"member_id,omitempty"` // Unique robot identifier (optional, auto-generated if empty)
	TeamID   string `json:"team_id,omitempty"`   // Team ID (optional, defaults to auth team or user_id)

	// Profile
	DisplayName string `json:"display_name" binding:"required"` // Display name
	Bio         string `json:"bio,omitempty"`                   // Robot description
	Avatar      string `json:"avatar,omitempty"`                // Avatar URL

	// Identity & Role
	SystemPrompt string `json:"system_prompt,omitempty"` // System prompt
	RoleID       string `json:"role_id,omitempty"`       // Role within team
	ManagerID    string `json:"manager_id,omitempty"`    // Direct manager user_id

	// Status
	Status         string `json:"status,omitempty"`          // Member status: active | inactive | pending | suspended
	RobotStatus    string `json:"robot_status,omitempty"`    // Robot status: idle | working | paused | error | maintenance
	AutonomousMode *bool  `json:"autonomous_mode,omitempty"` // Whether autonomous mode is enabled

	// Communication
	RobotEmail        string      `json:"robot_email,omitempty"`        // Robot email address
	AuthorizedSenders interface{} `json:"authorized_senders,omitempty"` // Email whitelist (JSON array)
	EmailFilterRules  interface{} `json:"email_filter_rules,omitempty"` // Email filter rules (JSON array)

	// Capabilities
	RobotConfig   interface{} `json:"robot_config,omitempty"`   // Robot config JSON
	Agents        interface{} `json:"agents,omitempty"`         // Accessible agents (JSON array)
	MCPServers    interface{} `json:"mcp_servers,omitempty"`    // MCP servers (JSON array)
	LanguageModel string      `json:"language_model,omitempty"` // Language model name

	// Limits
	CostLimit float64 `json:"cost_limit,omitempty"` // Monthly cost limit USD
}

// UpdateRobotRequest - HTTP request for updating a robot
type UpdateRobotRequest struct {
	// Profile
	DisplayName *string `json:"display_name,omitempty"` // Display name
	Bio         *string `json:"bio,omitempty"`          // Robot description
	Avatar      *string `json:"avatar,omitempty"`       // Avatar URL

	// Identity & Role
	SystemPrompt *string `json:"system_prompt,omitempty"` // System prompt
	RoleID       *string `json:"role_id,omitempty"`       // Role within team
	ManagerID    *string `json:"manager_id,omitempty"`    // Direct manager user_id

	// Status
	Status         *string `json:"status,omitempty"`          // Member status
	RobotStatus    *string `json:"robot_status,omitempty"`    // Robot status
	AutonomousMode *bool   `json:"autonomous_mode,omitempty"` // Autonomous mode

	// Communication
	RobotEmail        *string     `json:"robot_email,omitempty"`        // Robot email address
	AuthorizedSenders interface{} `json:"authorized_senders,omitempty"` // Email whitelist
	EmailFilterRules  interface{} `json:"email_filter_rules,omitempty"` // Email filter rules

	// Capabilities
	RobotConfig   interface{} `json:"robot_config,omitempty"`   // Robot config JSON
	Agents        interface{} `json:"agents,omitempty"`         // Accessible agents
	MCPServers    interface{} `json:"mcp_servers,omitempty"`    // MCP servers
	LanguageModel *string     `json:"language_model,omitempty"` // Language model name

	// Limits
	CostLimit *float64 `json:"cost_limit,omitempty"` // Monthly cost limit USD
}

// ==================== Response Types ====================

// Response - HTTP response for a robot
// Maps to frontend expectations: name ← member_id, description ← bio
type Response struct {
	// Basic (mapped for frontend)
	ID          int64  `json:"id,omitempty"`
	Name        string `json:"name"`        // Frontend name ← member_id
	Description string `json:"description"` // Frontend description ← bio

	// Original fields
	MemberID       string `json:"member_id"`
	TeamID         string `json:"team_id"`
	Status         string `json:"status"`
	RobotStatus    string `json:"robot_status"`
	AutonomousMode bool   `json:"autonomous_mode"`

	// Profile
	DisplayName string `json:"display_name"`
	Bio         string `json:"bio,omitempty"`
	Avatar      string `json:"avatar,omitempty"`

	// Identity & Role
	SystemPrompt string `json:"system_prompt,omitempty"`
	RoleID       string `json:"role_id,omitempty"`
	ManagerID    string `json:"manager_id,omitempty"`

	// Communication
	RobotEmail        string      `json:"robot_email,omitempty"`
	AuthorizedSenders interface{} `json:"authorized_senders,omitempty"`
	EmailFilterRules  interface{} `json:"email_filter_rules,omitempty"`

	// Capabilities
	RobotConfig   interface{} `json:"robot_config,omitempty"`
	Agents        interface{} `json:"agents,omitempty"`
	MCPServers    interface{} `json:"mcp_servers,omitempty"`
	LanguageModel string      `json:"language_model,omitempty"`

	// Limits
	CostLimit float64 `json:"cost_limit,omitempty"`

	// Ownership & Audit
	InvitedBy string     `json:"invited_by,omitempty"`
	JoinedAt  *time.Time `json:"joined_at,omitempty"`

	// Timestamps
	CreatedAt *time.Time `json:"created_at,omitempty"`
	UpdatedAt *time.Time `json:"updated_at,omitempty"`

	// Runtime Status (populated in list view for dashboard)
	Running    int        `json:"running"`               // Current running executions count
	MaxRunning int        `json:"max_running,omitempty"` // Maximum concurrent executions
	LastRun    *time.Time `json:"last_run,omitempty"`    // Last execution time
	NextRun    *time.Time `json:"next_run,omitempty"`    // Next scheduled run time
}

// StatusResponse - runtime status response
type StatusResponse struct {
	MemberID    string     `json:"member_id"`
	TeamID      string     `json:"team_id"`
	DisplayName string     `json:"display_name"`
	Bio         string     `json:"bio,omitempty"`
	Status      string     `json:"status"`      // Robot runtime status
	Running     int        `json:"running"`     // Current running executions
	MaxRunning  int        `json:"max_running"` // Maximum concurrent executions
	LastRun     *time.Time `json:"last_run,omitempty"`
	NextRun     *time.Time `json:"next_run,omitempty"`
	RunningIDs  []string   `json:"running_ids,omitempty"` // IDs of running executions
}

// ListResponse - paginated list response
type ListResponse struct {
	Data     []*Response `json:"data"`
	Total    int         `json:"total"`
	Page     int         `json:"page"`
	PageSize int         `json:"pagesize"`
}

// ==================== Conversion Functions ====================

// NewResponse creates a Response from api.RobotResponse
func NewResponse(r *robotapi.RobotResponse) *Response {
	if r == nil {
		return nil
	}

	return &Response{
		ID:                r.ID,
		Name:              r.MemberID, // Frontend mapping: name ← member_id
		Description:       r.Bio,      // Frontend mapping: description ← bio
		MemberID:          r.MemberID,
		TeamID:            r.TeamID,
		Status:            r.Status,
		RobotStatus:       r.RobotStatus,
		AutonomousMode:    r.AutonomousMode,
		DisplayName:       r.DisplayName,
		Bio:               r.Bio,
		Avatar:            r.Avatar,
		SystemPrompt:      r.SystemPrompt,
		RoleID:            r.RoleID,
		ManagerID:         r.ManagerID,
		RobotEmail:        r.RobotEmail,
		AuthorizedSenders: r.AuthorizedSenders,
		EmailFilterRules:  r.EmailFilterRules,
		RobotConfig:       r.RobotConfig,
		Agents:            r.Agents,
		MCPServers:        r.MCPServers,
		LanguageModel:     r.LanguageModel,
		CostLimit:         r.CostLimit,
		InvitedBy:         r.InvitedBy,
		JoinedAt:          r.JoinedAt,
		CreatedAt:         r.CreatedAt,
		UpdatedAt:         r.UpdatedAt,
	}
}

// ToAPICreateRequest converts HTTP request to api.CreateRobotRequest
func (r *CreateRobotRequest) ToAPICreateRequest() *robotapi.CreateRobotRequest {
	return &robotapi.CreateRobotRequest{
		MemberID:          r.MemberID,
		TeamID:            r.TeamID,
		DisplayName:       r.DisplayName,
		Bio:               r.Bio,
		Avatar:            r.Avatar,
		SystemPrompt:      r.SystemPrompt,
		RoleID:            r.RoleID,
		ManagerID:         r.ManagerID,
		Status:            r.Status,
		RobotStatus:       r.RobotStatus,
		AutonomousMode:    r.AutonomousMode,
		RobotEmail:        r.RobotEmail,
		AuthorizedSenders: r.AuthorizedSenders,
		EmailFilterRules:  r.EmailFilterRules,
		RobotConfig:       r.RobotConfig,
		Agents:            r.Agents,
		MCPServers:        r.MCPServers,
		LanguageModel:     r.LanguageModel,
		CostLimit:         r.CostLimit,
	}
}

// ToAPIUpdateRequest converts HTTP request to api.UpdateRobotRequest
func (r *UpdateRobotRequest) ToAPIUpdateRequest() *robotapi.UpdateRobotRequest {
	return &robotapi.UpdateRobotRequest{
		DisplayName:       r.DisplayName,
		Bio:               r.Bio,
		Avatar:            r.Avatar,
		SystemPrompt:      r.SystemPrompt,
		RoleID:            r.RoleID,
		ManagerID:         r.ManagerID,
		Status:            r.Status,
		RobotStatus:       r.RobotStatus,
		AutonomousMode:    r.AutonomousMode,
		RobotEmail:        r.RobotEmail,
		AuthorizedSenders: r.AuthorizedSenders,
		EmailFilterRules:  r.EmailFilterRules,
		RobotConfig:       r.RobotConfig,
		Agents:            r.Agents,
		MCPServers:        r.MCPServers,
		LanguageModel:     r.LanguageModel,
		CostLimit:         r.CostLimit,
	}
}

// NewStatusResponse creates a StatusResponse from api.RobotState
func NewStatusResponse(s *robotapi.RobotState) *StatusResponse {
	if s == nil {
		return nil
	}

	return &StatusResponse{
		MemberID:    s.MemberID,
		TeamID:      s.TeamID,
		DisplayName: s.DisplayName,
		Bio:         s.Bio,
		Status:      string(s.Status),
		Running:     s.Running,
		MaxRunning:  s.MaxRunning,
		LastRun:     s.LastRun,
		NextRun:     s.NextRun,
		RunningIDs:  s.RunningIDs,
	}
}

// ==================== Execution Types ====================

// ExecutionFilter - query params for listing executions
type ExecutionFilter struct {
	Status      string `form:"status"`       // pending | running | paused | completed | failed | cancelled
	TriggerType string `form:"trigger_type"` // clock | human | event
	Keyword     string `form:"keyword"`      // search in execution details
	Page        int    `form:"page"`
	PageSize    int    `form:"pagesize"`
}

// ExecutionResponse - single execution response
type ExecutionResponse struct {
	ID          string     `json:"id"`
	MemberID    string     `json:"member_id"`
	TeamID      string     `json:"team_id"`
	TriggerType string     `json:"trigger_type"`
	Status      string     `json:"status"`
	Phase       string     `json:"phase"`
	StartTime   time.Time  `json:"start_time"`
	EndTime     *time.Time `json:"end_time,omitempty"`
	Error       string     `json:"error,omitempty"`

	// UI display fields (updated by executor at each phase)
	Name            string `json:"name,omitempty"`              // Execution title
	CurrentTaskName string `json:"current_task_name,omitempty"` // Current task description

	// Phase outputs (optional, included in detail view)
	Inspiration interface{} `json:"inspiration,omitempty"`
	Goals       interface{} `json:"goals,omitempty"`
	Tasks       interface{} `json:"tasks,omitempty"`
	Current     interface{} `json:"current,omitempty"`
	Results     interface{} `json:"results,omitempty"`
	Delivery    interface{} `json:"delivery,omitempty"`

	// Input (optional, included in detail view)
	Input interface{} `json:"input,omitempty"`
}

// ExecutionListResponse - paginated list response
type ExecutionListResponse struct {
	Data     []*ExecutionResponse `json:"data"`
	Total    int                  `json:"total"`
	Page     int                  `json:"page"`
	PageSize int                  `json:"pagesize"`
}

// ExecutionControlResponse - response for pause/resume/cancel
type ExecutionControlResponse struct {
	ExecutionID string `json:"execution_id"`
	Action      string `json:"action"` // paused | resumed | cancelled
	Success     bool   `json:"success"`
	Message     string `json:"message,omitempty"`
}

// ==================== Trigger Types ====================

// TriggerRequest - HTTP request to trigger robot execution
type TriggerRequest struct {
	// Trigger type: human | event | clock (defaults to human)
	TriggerType string `json:"trigger_type,omitempty"`

	// Human intervention fields
	Action   string        `json:"action,omitempty"`   // task.add, goal.adjust, etc.
	Messages []MessageItem `json:"messages,omitempty"` // user's input

	// Event fields
	Source    string                 `json:"source,omitempty"`     // webhook | database
	EventType string                 `json:"event_type,omitempty"` // lead.created, etc.
	Data      map[string]interface{} `json:"data,omitempty"`       // event payload

	// Executor mode (optional)
	ExecutorMode string `json:"executor_mode,omitempty"` // standard | fast | careful

	// i18n support
	Locale string `json:"locale,omitempty"` // Locale for UI messages (e.g., "en", "zh")
}

// MessageItem - a single message in trigger request
type MessageItem struct {
	Role    string `json:"role"`              // user | assistant | system
	Content string `json:"content"`           // message text
	Name    string `json:"name,omitempty"`    // optional name
	FileID  string `json:"file_id,omitempty"` // optional attachment
}

// TriggerResponse - response after triggering
type TriggerResponse struct {
	Accepted    bool   `json:"accepted"`
	ExecutionID string `json:"execution_id,omitempty"`
	Queued      bool   `json:"queued,omitempty"`
	Message     string `json:"message,omitempty"`
}

// InterveneRequest - HTTP request for human intervention
type InterveneRequest struct {
	Action   string        `json:"action"`             // task.add, goal.adjust, etc.
	Messages []MessageItem `json:"messages,omitempty"` // user's input
	PlanAt   *time.Time    `json:"plan_at,omitempty"`  // schedule for later
}

// InterveneResponse - response after intervention
type InterveneResponse struct {
	Accepted    bool   `json:"accepted"`
	ExecutionID string `json:"execution_id,omitempty"`
	Message     string `json:"message,omitempty"`
}

// ==================== Execution Conversion Functions ====================

// NewExecutionListResponse creates an ExecutionListResponse from api.ExecutionResult
func NewExecutionListResponse(e *robotapi.ExecutionResult) *ExecutionListResponse {
	if e == nil {
		return nil
	}

	data := make([]*ExecutionResponse, 0, len(e.Data))
	for _, exec := range e.Data {
		data = append(data, NewExecutionResponseFromExecution(exec))
	}

	return &ExecutionListResponse{
		Data:     data,
		Total:    e.Total,
		Page:     e.Page,
		PageSize: e.PageSize,
	}
}

// NewExecutionResponseFromExecution converts types.Execution to ExecutionResponse
func NewExecutionResponseFromExecution(exec *robottypes.Execution) *ExecutionResponse {
	if exec == nil {
		return nil
	}

	return &ExecutionResponse{
		ID:          exec.ID,
		MemberID:    exec.MemberID,
		TeamID:      exec.TeamID,
		TriggerType: string(exec.TriggerType),
		Status:      string(exec.Status),
		Phase:       string(exec.Phase),
		StartTime:   exec.StartTime,
		EndTime:     exec.EndTime,
		Error:       exec.Error,
		// UI display fields
		Name:            exec.Name,
		CurrentTaskName: exec.CurrentTaskName,
		// Phase outputs - include in detail view
		Inspiration: exec.Inspiration,
		Goals:       exec.Goals,
		Tasks:       exec.Tasks,
		Current:     exec.Current,
		Results:     exec.Results,
		Delivery:    exec.Delivery,
		Input:       exec.Input,
	}
}

// NewExecutionResponseBrief creates a brief ExecutionResponse (for list view)
func NewExecutionResponseBrief(exec *robottypes.Execution) *ExecutionResponse {
	if exec == nil {
		return nil
	}

	// Calculate current state for progress bar display
	// If exec.Current is nil but we have tasks, calculate progress from tasks
	var current interface{}
	if exec.Current != nil {
		current = exec.Current
	} else if len(exec.Tasks) > 0 {
		// Calculate completed count from tasks
		completedCount := 0
		for _, task := range exec.Tasks {
			if task.Status == robottypes.TaskCompleted ||
				task.Status == robottypes.TaskFailed ||
				task.Status == robottypes.TaskSkipped {
				completedCount++
			}
		}
		// Create a synthetic current state for progress display
		current = map[string]interface{}{
			"task_index": len(exec.Tasks),
			"progress":   fmt.Sprintf("%d/%d", completedCount, len(exec.Tasks)),
		}
	}

	return &ExecutionResponse{
		ID:          exec.ID,
		MemberID:    exec.MemberID,
		TeamID:      exec.TeamID,
		TriggerType: string(exec.TriggerType),
		Status:      string(exec.Status),
		Phase:       string(exec.Phase),
		StartTime:   exec.StartTime,
		EndTime:     exec.EndTime,
		Error:       exec.Error,
		// UI display fields - include in list view for display
		Name:            exec.Name,
		CurrentTaskName: exec.CurrentTaskName,
		// Include Current for progress bar display in list view
		Current: current,
		// Omit other phase outputs for list view (inspiration, goals, tasks, results, delivery, input)
	}
}

// ==================== Results Types ====================

// ResultFilter - query params for listing results
type ResultFilter struct {
	TriggerType string `form:"trigger_type"` // clock | human | event
	Keyword     string `form:"keyword"`      // search in name/summary
	Page        int    `form:"page"`
	PageSize    int    `form:"pagesize"`
}

// ResultResponse - result list item
type ResultResponse struct {
	ID             string     `json:"id"`
	MemberID       string     `json:"member_id"`
	TriggerType    string     `json:"trigger_type"`
	Status         string     `json:"status"`
	Name           string     `json:"name"`
	Summary        string     `json:"summary"`
	StartTime      time.Time  `json:"start_time"`
	EndTime        *time.Time `json:"end_time,omitempty"`
	HasAttachments bool       `json:"has_attachments"`
}

// ResultDetailResponse - full result with delivery content
type ResultDetailResponse struct {
	ID          string      `json:"id"`
	MemberID    string      `json:"member_id"`
	TriggerType string      `json:"trigger_type"`
	Status      string      `json:"status"`
	Name        string      `json:"name"`
	Delivery    interface{} `json:"delivery,omitempty"`
	StartTime   time.Time   `json:"start_time"`
	EndTime     *time.Time  `json:"end_time,omitempty"`
}

// ResultListResponse - paginated list response
type ResultListResponse struct {
	Data     []*ResultResponse `json:"data"`
	Total    int               `json:"total"`
	Page     int               `json:"page"`
	PageSize int               `json:"pagesize"`
}

// NewResultResponse creates a ResultResponse from api.ResultItem
func NewResultResponse(item *robotapi.ResultItem) *ResultResponse {
	if item == nil {
		return nil
	}

	return &ResultResponse{
		ID:             item.ID,
		MemberID:       item.MemberID,
		TriggerType:    string(item.TriggerType),
		Status:         string(item.Status),
		Name:           item.Name,
		Summary:        item.Summary,
		StartTime:      item.StartTime,
		EndTime:        item.EndTime,
		HasAttachments: item.HasAttachments,
	}
}

// NewResultDetailResponse creates a ResultDetailResponse from api.ResultDetail
func NewResultDetailResponse(detail *robotapi.ResultDetail) *ResultDetailResponse {
	if detail == nil {
		return nil
	}

	return &ResultDetailResponse{
		ID:          detail.ID,
		MemberID:    detail.MemberID,
		TriggerType: string(detail.TriggerType),
		Status:      string(detail.Status),
		Name:        detail.Name,
		Delivery:    detail.Delivery,
		StartTime:   detail.StartTime,
		EndTime:     detail.EndTime,
	}
}

// ==================== Activities Types ====================

// ActivityFilter - query params for listing activities
type ActivityFilter struct {
	Limit int    `form:"limit"` // max number of activities
	Since string `form:"since"` // ISO timestamp, only activities after this time
	Type  string `form:"type"`  // activity type filter: execution.started, execution.completed, execution.failed, execution.cancelled
}

// ActivityResponse - activity item
type ActivityResponse struct {
	Type        string    `json:"type"` // execution.started, execution.completed, etc.
	RobotID     string    `json:"robot_id"`
	RobotName   string    `json:"robot_name,omitempty"`
	ExecutionID string    `json:"execution_id"`
	Message     string    `json:"message"`
	Timestamp   time.Time `json:"timestamp"`
}

// ActivityListResponse - activity list response
type ActivityListResponse struct {
	Data []*ActivityResponse `json:"data"`
}

// NewActivityResponse creates an ActivityResponse from api.Activity
func NewActivityResponse(activity *robotapi.Activity) *ActivityResponse {
	if activity == nil {
		return nil
	}

	return &ActivityResponse{
		Type:        string(activity.Type),
		RobotID:     activity.RobotID,
		RobotName:   activity.RobotName,
		ExecutionID: activity.ExecutionID,
		Message:     activity.Message,
		Timestamp:   activity.Timestamp,
	}
}
