package store

import (
	"context"
	"fmt"
	"time"

	"github.com/yaoapp/gou/model"
	"github.com/yaoapp/kun/maps"
	"github.com/yaoapp/yao/agent/robot/types"
	"github.com/yaoapp/yao/agent/robot/utils"
)

// RobotRecord - persistent storage for robot member
// Maps to __yao.member model
type RobotRecord struct {
	ID             int64  `json:"id,omitempty"`    // Auto-increment primary key
	MemberID       string `json:"member_id"`       // Unique robot identifier
	TeamID         string `json:"team_id"`         // Team ID
	MemberType     string `json:"member_type"`     // Always "robot" for robots
	Status         string `json:"status"`          // Member status: active | inactive | pending | suspended
	RobotStatus    string `json:"robot_status"`    // Robot status: idle | working | paused | error | maintenance
	AutonomousMode bool   `json:"autonomous_mode"` // Whether autonomous mode is enabled

	// Profile
	DisplayName string `json:"display_name"`  // Display name
	Bio         string `json:"bio,omitempty"` // Robot description
	Avatar      string `json:"avatar,omitempty"`

	// Identity & Role
	SystemPrompt string `json:"system_prompt"` // System prompt
	RoleID       string `json:"role_id"`       // Role within team
	ManagerID    string `json:"manager_id"`    // Direct manager user_id (who manages this robot)

	// Communication
	RobotEmail        string      `json:"robot_email"`                  // Robot email address
	AuthorizedSenders interface{} `json:"authorized_senders,omitempty"` // Email whitelist (JSON array)
	EmailFilterRules  interface{} `json:"email_filter_rules,omitempty"` // Email filter rules (JSON array)

	// Capabilities
	RobotConfig   interface{} `json:"robot_config"`             // Robot config JSON
	Agents        interface{} `json:"agents,omitempty"`         // Accessible agents (JSON array)
	MCPServers    interface{} `json:"mcp_servers,omitempty"`    // MCP servers (JSON array)
	LanguageModel string      `json:"language_model,omitempty"` // Language model name

	// Limits
	CostLimit float64 `json:"cost_limit,omitempty"` // Monthly cost limit USD

	// Ownership & Audit
	InvitedBy string     `json:"invited_by,omitempty"` // Who created/added this robot
	JoinedAt  *time.Time `json:"joined_at,omitempty"`  // When robot was created

	// Timestamps
	CreatedAt *time.Time `json:"created_at,omitempty"`
	UpdatedAt *time.Time `json:"updated_at,omitempty"`

	// Yao Permission Fields (automatically handled by Yao model when permission:true)
	// These fields are passed through to the model layer for permission control
	YaoCreatedBy string `json:"__yao_created_by,omitempty"` // Creator user_id (set on create)
	YaoUpdatedBy string `json:"__yao_updated_by,omitempty"` // Updater user_id (set on update)
	YaoTeamID    string `json:"__yao_team_id,omitempty"`    // Permission team scope
	YaoTenantID  string `json:"__yao_tenant_id,omitempty"`  // Permission tenant scope
}

// RobotListOptions - options for listing robot records
type RobotListOptions struct {
	TeamID   string            `json:"team_id,omitempty"`
	Status   types.RobotStatus `json:"status,omitempty"`
	Keywords string            `json:"keywords,omitempty"` // Search in display_name
	Limit    int               `json:"limit,omitempty"`
	Offset   int               `json:"offset,omitempty"`
	Page     int               `json:"page,omitempty"`
	PageSize int               `json:"pagesize,omitempty"`
	OrderBy  string            `json:"order_by,omitempty"`
}

// RobotStore - persistent storage for robot members
type RobotStore struct {
	modelID string
}

// NewRobotStore creates a new robot store instance
func NewRobotStore() *RobotStore {
	return &RobotStore{
		modelID: "__yao.member",
	}
}

// robotFields are the fields to select when loading robots
var robotFields = []interface{}{
	// Basic
	"id",
	"member_id",
	"team_id",
	"member_type",
	"status",
	"robot_status",
	"autonomous_mode",

	// Profile
	"display_name",
	"bio",
	"avatar",

	// Identity & Role
	"system_prompt",
	"role_id",
	"manager_id",

	// Communication
	"robot_email",
	"authorized_senders",
	"email_filter_rules",

	// Capabilities
	"robot_config",
	"agents",
	"mcp_servers",
	"language_model",

	// Limits
	"cost_limit",

	// Ownership & Audit
	"invited_by",
	"joined_at",

	// Timestamps
	"created_at",
	"updated_at",

	// Yao Permission Fields (for access control)
	"__yao_created_by",
	"__yao_updated_by",
	"__yao_team_id",
	"__yao_tenant_id",
}

// Save creates or updates a robot member record
func (s *RobotStore) Save(ctx context.Context, record *RobotRecord) error {
	mod := model.Select(s.modelID)
	if mod == nil {
		return fmt.Errorf("model %s not found", s.modelID)
	}

	// Ensure member_type is robot
	record.MemberType = "robot"

	data := s.recordToMap(record)

	// Check if record exists by member_id
	existing, err := s.Get(ctx, record.MemberID)
	if err == nil && existing != nil {
		// Update existing record
		_, err = mod.UpdateWhere(
			model.QueryParam{
				Wheres: []model.QueryWhere{
					{Column: "member_id", Value: record.MemberID},
				},
			},
			data,
		)
		if err != nil {
			return fmt.Errorf("failed to update robot record: %w", err)
		}
		return nil
	}

	// Create new record
	_, err = mod.Create(data)
	if err != nil {
		return fmt.Errorf("failed to create robot record: %w", err)
	}
	return nil
}

// Get retrieves a robot record by member_id
func (s *RobotStore) Get(ctx context.Context, memberID string) (*RobotRecord, error) {
	mod := model.Select(s.modelID)
	if mod == nil {
		return nil, fmt.Errorf("model %s not found", s.modelID)
	}

	rows, err := mod.Get(model.QueryParam{
		Select: robotFields,
		Wheres: []model.QueryWhere{
			{Column: "member_id", Value: memberID},
			{Column: "member_type", Value: "robot"},
		},
		Limit: 1,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get robot record: %w", err)
	}

	if len(rows) == 0 {
		return nil, nil
	}

	return s.mapToRecord(rows[0])
}

// List retrieves robot records with filters
func (s *RobotStore) List(ctx context.Context, opts *RobotListOptions) ([]*RobotRecord, int, error) {
	mod := model.Select(s.modelID)
	if mod == nil {
		return nil, 0, fmt.Errorf("model %s not found", s.modelID)
	}

	// Build where conditions - only require member_type=robot
	wheres := []model.QueryWhere{
		{Column: "member_type", Value: "robot"},
	}

	if opts != nil {
		if opts.TeamID != "" {
			wheres = append(wheres, model.QueryWhere{Column: "team_id", Value: opts.TeamID})
		}
		if opts.Status != "" {
			wheres = append(wheres, model.QueryWhere{Column: "robot_status", Value: string(opts.Status)})
		}
		if opts.Keywords != "" {
			wheres = append(wheres, model.QueryWhere{
				Column: "display_name",
				OP:     "like",
				Value:  "%" + opts.Keywords + "%",
			})
		}
	}

	// Build order
	orders := []model.QueryOrder{}
	if opts != nil && opts.OrderBy != "" {
		orders = append(orders, model.QueryOrder{Column: opts.OrderBy})
	} else {
		orders = append(orders, model.QueryOrder{Column: "created_at", Option: "desc"})
	}

	// Determine pagination
	page := 1
	pageSize := 100
	if opts != nil {
		if opts.Page > 0 {
			page = opts.Page
		}
		if opts.PageSize > 0 {
			pageSize = opts.PageSize
		}
		// Limit overrides PageSize for simple limit queries
		if opts.Limit > 0 {
			pageSize = opts.Limit
		}
	}

	// Execute paginated query
	result, err := mod.Paginate(model.QueryParam{
		Select: robotFields,
		Wheres: wheres,
		Orders: orders,
	}, page, pageSize)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list robots: %w", err)
	}

	// Get total count
	total := 0
	if t, ok := result.Get("total").(int); ok {
		total = t
	}

	// Parse records
	records := []*RobotRecord{}
	data := result.Get("data")
	switch rows := data.(type) {
	case []maps.MapStr:
		for _, row := range rows {
			record, err := s.mapToRecord(map[string]interface{}(row))
			if err != nil {
				continue // skip invalid records
			}
			records = append(records, record)
		}
	case []map[string]interface{}:
		for _, row := range rows {
			record, err := s.mapToRecord(row)
			if err != nil {
				continue // skip invalid records
			}
			records = append(records, record)
		}
	}

	return records, total, nil
}

// Delete removes a robot member by member_id
func (s *RobotStore) Delete(ctx context.Context, memberID string) error {
	mod := model.Select(s.modelID)
	if mod == nil {
		return fmt.Errorf("model %s not found", s.modelID)
	}

	_, err := mod.DeleteWhere(model.QueryParam{
		Wheres: []model.QueryWhere{
			{Column: "member_id", Value: memberID},
			{Column: "member_type", Value: "robot"},
		},
	})
	if err != nil {
		return fmt.Errorf("failed to delete robot record: %w", err)
	}

	return nil
}

// UpdateConfig updates only the robot_config field
func (s *RobotStore) UpdateConfig(ctx context.Context, memberID string, config interface{}) error {
	mod := model.Select(s.modelID)
	if mod == nil {
		return fmt.Errorf("model %s not found", s.modelID)
	}

	data := map[string]interface{}{
		"robot_config": config,
	}

	_, err := mod.UpdateWhere(
		model.QueryParam{
			Wheres: []model.QueryWhere{
				{Column: "member_id", Value: memberID},
				{Column: "member_type", Value: "robot"},
			},
		},
		data,
	)
	if err != nil {
		return fmt.Errorf("failed to update robot config: %w", err)
	}

	return nil
}

// UpdateStatus updates the robot_status field
func (s *RobotStore) UpdateStatus(ctx context.Context, memberID string, status types.RobotStatus) error {
	mod := model.Select(s.modelID)
	if mod == nil {
		return fmt.Errorf("model %s not found", s.modelID)
	}

	data := map[string]interface{}{
		"robot_status": string(status),
	}

	_, err := mod.UpdateWhere(
		model.QueryParam{
			Wheres: []model.QueryWhere{
				{Column: "member_id", Value: memberID},
				{Column: "member_type", Value: "robot"},
			},
		},
		data,
	)
	if err != nil {
		return fmt.Errorf("failed to update robot status: %w", err)
	}

	return nil
}

// recordToMap converts RobotRecord to map for model operations
func (s *RobotStore) recordToMap(record *RobotRecord) map[string]interface{} {
	data := map[string]interface{}{
		// Required fields
		"member_id":       record.MemberID,
		"team_id":         record.TeamID,
		"member_type":     "robot",
		"autonomous_mode": record.AutonomousMode,
	}

	// Status
	if record.Status != "" {
		data["status"] = record.Status
	} else {
		data["status"] = "active"
	}
	if record.RobotStatus != "" {
		data["robot_status"] = record.RobotStatus
	} else {
		data["robot_status"] = "idle"
	}

	// Profile
	if record.DisplayName != "" {
		data["display_name"] = record.DisplayName
	}
	if record.Bio != "" {
		data["bio"] = record.Bio
	}
	if record.Avatar != "" {
		data["avatar"] = record.Avatar
	}

	// Identity & Role
	if record.SystemPrompt != "" {
		data["system_prompt"] = record.SystemPrompt
	}
	if record.RoleID != "" {
		data["role_id"] = record.RoleID
	}
	if record.ManagerID != "" {
		data["manager_id"] = record.ManagerID
	}

	// Communication
	if record.RobotEmail != "" {
		data["robot_email"] = record.RobotEmail
	}
	if record.AuthorizedSenders != nil {
		data["authorized_senders"] = record.AuthorizedSenders
	}
	if record.EmailFilterRules != nil {
		data["email_filter_rules"] = record.EmailFilterRules
	}

	// Capabilities
	if record.RobotConfig != nil {
		data["robot_config"] = record.RobotConfig
	}
	if record.Agents != nil {
		data["agents"] = record.Agents
	}
	if record.MCPServers != nil {
		data["mcp_servers"] = record.MCPServers
	}
	if record.LanguageModel != "" {
		data["language_model"] = record.LanguageModel
	}

	// Limits
	if record.CostLimit > 0 {
		data["cost_limit"] = record.CostLimit
	}

	// Ownership & Audit
	if record.InvitedBy != "" {
		data["invited_by"] = record.InvitedBy
	}
	if record.JoinedAt != nil {
		// Format time for Gou model (expects string format)
		data["joined_at"] = record.JoinedAt.Format("2006-01-02 15:04:05")
	}

	// Yao Permission Fields - pass through for model layer
	if record.YaoCreatedBy != "" {
		data["__yao_created_by"] = record.YaoCreatedBy
	}
	if record.YaoUpdatedBy != "" {
		data["__yao_updated_by"] = record.YaoUpdatedBy
	}
	if record.YaoTeamID != "" {
		data["__yao_team_id"] = record.YaoTeamID
	}
	if record.YaoTenantID != "" {
		data["__yao_tenant_id"] = record.YaoTenantID
	}

	return data
}

// mapToRecord converts a model row to RobotRecord
func (s *RobotStore) mapToRecord(row map[string]interface{}) (*RobotRecord, error) {
	record := &RobotRecord{}

	// Basic fields
	if v, ok := row["id"]; ok {
		switch id := v.(type) {
		case float64:
			record.ID = int64(id)
		case int64:
			record.ID = id
		case int:
			record.ID = int64(id)
		}
	}
	if v, ok := row["member_id"].(string); ok {
		record.MemberID = v
	}
	if v, ok := row["team_id"].(string); ok {
		record.TeamID = v
	}
	if v, ok := row["member_type"].(string); ok {
		record.MemberType = v
	}
	if v, ok := row["status"].(string); ok {
		record.Status = v
	}
	if v, ok := row["robot_status"].(string); ok {
		record.RobotStatus = v
	}
	if v, ok := row["autonomous_mode"]; ok {
		record.AutonomousMode = utils.ToBool(v)
	}

	// Profile
	if v, ok := row["display_name"].(string); ok {
		record.DisplayName = v
	}
	if v, ok := row["bio"].(string); ok {
		record.Bio = v
	}
	if v, ok := row["avatar"].(string); ok {
		record.Avatar = v
	}

	// Identity & Role
	if v, ok := row["system_prompt"].(string); ok {
		record.SystemPrompt = v
	}
	if v, ok := row["role_id"].(string); ok {
		record.RoleID = v
	}
	if v, ok := row["manager_id"].(string); ok {
		record.ManagerID = v
	}

	// Communication
	if v, ok := row["robot_email"].(string); ok {
		record.RobotEmail = v
	}
	if v := row["authorized_senders"]; v != nil {
		record.AuthorizedSenders = utils.ToJSONValue(v)
	}
	if v := row["email_filter_rules"]; v != nil {
		record.EmailFilterRules = utils.ToJSONValue(v)
	}

	// Capabilities
	if v := row["robot_config"]; v != nil {
		record.RobotConfig = utils.ToJSONValue(v)
	}
	if v := row["agents"]; v != nil {
		record.Agents = utils.ToJSONValue(v)
	}
	if v := row["mcp_servers"]; v != nil {
		record.MCPServers = utils.ToJSONValue(v)
	}
	if v, ok := row["language_model"].(string); ok {
		record.LanguageModel = v
	}

	// Limits
	if v := row["cost_limit"]; v != nil {
		record.CostLimit = utils.ToFloat64(v)
	}

	// Ownership & Audit
	if v, ok := row["invited_by"].(string); ok {
		record.InvitedBy = v
	}
	if v := row["joined_at"]; v != nil {
		record.JoinedAt = utils.ToTimestamp(v)
	}

	// Timestamps
	if v := row["created_at"]; v != nil {
		record.CreatedAt = utils.ToTimestamp(v)
	}
	if v := row["updated_at"]; v != nil {
		record.UpdatedAt = utils.ToTimestamp(v)
	}

	// Yao Permission Fields
	if v, ok := row["__yao_created_by"].(string); ok {
		record.YaoCreatedBy = v
	}
	if v, ok := row["__yao_updated_by"].(string); ok {
		record.YaoUpdatedBy = v
	}
	if v, ok := row["__yao_team_id"].(string); ok {
		record.YaoTeamID = v
	}
	if v, ok := row["__yao_tenant_id"].(string); ok {
		record.YaoTenantID = v
	}

	return record, nil
}

// ToRobot converts a RobotRecord to types.Robot
func (r *RobotRecord) ToRobot() (*types.Robot, error) {
	robot := &types.Robot{
		MemberID:       r.MemberID,
		TeamID:         r.TeamID,
		DisplayName:    r.DisplayName,
		Bio:            r.Bio,
		SystemPrompt:   r.SystemPrompt,
		AutonomousMode: r.AutonomousMode,
		RobotEmail:     r.RobotEmail,
	}

	// Parse robot_status
	if r.RobotStatus != "" {
		robot.Status = types.RobotStatus(r.RobotStatus)
	} else {
		robot.Status = types.RobotIdle
	}

	// Parse robot_config
	if r.RobotConfig != nil {
		config, err := types.ParseConfig(r.RobotConfig)
		if err != nil {
			return nil, fmt.Errorf("failed to parse robot_config: %w", err)
		}
		robot.Config = config
	}

	// Ensure Config exists for merging agents/mcp_servers
	if robot.Config == nil {
		robot.Config = &types.Config{}
	}
	if robot.Config.Resources == nil {
		robot.Config.Resources = &types.Resources{}
	}

	// Merge agents from member table into Config.Resources.Agents
	if r.Agents != nil {
		agents := parseStringSlice(r.Agents)
		if len(agents) > 0 {
			robot.Config.Resources.Agents = agents
		}
	}

	// Merge mcp_servers from member table into Config.Resources.MCP
	if r.MCPServers != nil {
		mcpServers := parseStringSlice(r.MCPServers)
		if len(mcpServers) > 0 {
			// Convert string slice to MCPConfig slice (each server ID becomes an MCPConfig)
			for _, serverID := range mcpServers {
				robot.Config.Resources.MCP = append(robot.Config.Resources.MCP, types.MCPConfig{
					ID: serverID,
					// Tools empty means all tools available
				})
			}
		}
	}

	return robot, nil
}

// parseStringSlice converts interface{} to []string
func parseStringSlice(v interface{}) []string {
	if v == nil {
		return nil
	}
	switch val := v.(type) {
	case []string:
		return val
	case []interface{}:
		result := make([]string, 0, len(val))
		for _, item := range val {
			if s, ok := item.(string); ok {
				result = append(result, s)
			}
		}
		return result
	}
	return nil
}

// FromRobot creates a RobotRecord from types.Robot
func FromRobot(robot *types.Robot) *RobotRecord {
	record := &RobotRecord{
		MemberID:       robot.MemberID,
		TeamID:         robot.TeamID,
		DisplayName:    robot.DisplayName,
		Bio:            robot.Bio,
		SystemPrompt:   robot.SystemPrompt,
		RobotStatus:    string(robot.Status),
		AutonomousMode: robot.AutonomousMode,
		RobotEmail:     robot.RobotEmail,
		MemberType:     "robot",
		Status:         "active",
	}

	if robot.Config != nil {
		record.RobotConfig = robot.Config
	}

	return record
}
