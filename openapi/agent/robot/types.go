package robot

import (
	"time"

	robotapi "github.com/yaoapp/yao/agent/robot/api"
)

// ==================== Request Types ====================

// CreateRobotRequest - HTTP request for creating a robot
type CreateRobotRequest struct {
	// Required fields
	MemberID string `json:"member_id" binding:"required"` // Unique robot identifier
	TeamID   string `json:"team_id" binding:"required"`   // Team ID

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
