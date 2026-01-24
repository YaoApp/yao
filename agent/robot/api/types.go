package api

import (
	"time"

	agentcontext "github.com/yaoapp/yao/agent/context"
	"github.com/yaoapp/yao/agent/robot/types"
)

// ListQuery - query options for List()
type ListQuery struct {
	TeamID         string            `json:"team_id,omitempty"`
	Status         types.RobotStatus `json:"status,omitempty"`
	Keywords       string            `json:"keywords,omitempty"`
	ClockMode      types.ClockMode   `json:"clock_mode,omitempty"`
	AutonomousMode *bool             `json:"autonomous_mode,omitempty"` // nil=all, true=autonomous only, false=on-demand only
	Page           int               `json:"page,omitempty"`
	PageSize       int               `json:"pagesize,omitempty"`
	Order          string            `json:"order,omitempty"`
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
	MemberID     string            `json:"member_id"`
	TeamID       string            `json:"team_id"`
	DisplayName  string            `json:"display_name"`
	Bio          string            `json:"bio,omitempty"`
	Status       types.RobotStatus `json:"status"`
	Running      int               `json:"running"`
	MaxRunning   int               `json:"max_running"`
	LastRun      *time.Time        `json:"last_run,omitempty"`
	NextRun      *time.Time        `json:"next_run,omitempty"`
	RunningIDs   []string          `json:"running_ids,omitempty"`
	YaoCreatedBy string            `json:"__yao_created_by,omitempty"` // Creator user_id for permission check
	YaoTeamID    string            `json:"__yao_team_id,omitempty"`    // Team ID for permission check
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

	// i18n support
	Locale string `json:"locale,omitempty"` // Locale for UI messages (e.g., "en", "zh")
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
	Accepted    bool             `json:"accepted"`
	Queued      bool             `json:"queued"`
	Execution   *types.Execution `json:"execution,omitempty"`
	ExecutionID string           `json:"execution_id,omitempty"` // Execution ID
	Message     string           `json:"message,omitempty"`
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

// ==================== CRUD Types ====================

// AuthScope contains Yao permission fields for data scoping
// These fields are used by Yao's permission system (when model has permission: true)
type AuthScope struct {
	CreatedBy string `json:"__yao_created_by,omitempty"` // Creator user_id
	UpdatedBy string `json:"__yao_updated_by,omitempty"` // Updater user_id
	TeamID    string `json:"__yao_team_id,omitempty"`    // Permission team scope
	TenantID  string `json:"__yao_tenant_id,omitempty"`  // Permission tenant scope
}

// CreateRobotRequest - request for CreateRobot()
type CreateRobotRequest struct {
	// Identity (member_id is optional - auto-generated if not provided)
	MemberID string `json:"member_id,omitempty"` // Unique robot identifier (auto-generated if empty)
	TeamID   string `json:"team_id"`             // Team ID (required)

	// Profile
	DisplayName string `json:"display_name,omitempty"` // Display name
	Bio         string `json:"bio,omitempty"`          // Robot description
	Avatar      string `json:"avatar,omitempty"`       // Avatar URL

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

	// Auth scope (optional, used by OpenAPI layer via WithCreateScope)
	AuthScope *AuthScope `json:"auth_scope,omitempty"`
}

// UpdateRobotRequest - request for UpdateRobot()
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

	// Auth scope (optional, used by OpenAPI layer via WithUpdateScope)
	AuthScope *AuthScope `json:"auth_scope,omitempty"`
}

// RobotResponse - response containing robot details for API
type RobotResponse struct {
	// Basic
	ID             int64  `json:"id,omitempty"`
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
	InvitedBy    string     `json:"invited_by,omitempty"`
	JoinedAt     *time.Time `json:"joined_at,omitempty"`
	YaoCreatedBy string     `json:"__yao_created_by,omitempty"` // Creator user_id for permission check
	YaoTeamID    string     `json:"__yao_team_id,omitempty"`    // Team ID for permission check

	// Timestamps
	CreatedAt *time.Time `json:"created_at,omitempty"`
	UpdatedAt *time.Time `json:"updated_at,omitempty"`
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
