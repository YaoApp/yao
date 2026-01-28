package api

import (
	"context"
	"fmt"
	"time"

	gonanoid "github.com/matoous/go-nanoid/v2"
	"github.com/yaoapp/gou/model"
	"github.com/yaoapp/kun/maps"
	"github.com/yaoapp/yao/agent/robot/store"
	"github.com/yaoapp/yao/agent/robot/types"
)

// ==================== Robot Query API ====================
// These functions query robot information

// memberModel is the model name for member table
const memberModel = "__yao.member"

// robotStore is the shared robot store instance
var robotStore = store.NewRobotStore()

// executionStore is the shared execution store instance
var executionStore = store.NewExecutionStore()

// GetRobot returns a robot by member ID
// Returns the robot from cache if available, otherwise loads from database
func GetRobot(ctx *types.Context, memberID string) (*types.Robot, error) {
	if memberID == "" {
		return nil, fmt.Errorf("member_id is required")
	}

	mgr, err := getManager()
	if err != nil {
		// Manager not started, try to load directly from database
		return loadRobotFromDB(memberID)
	}

	// Try cache first
	robot := mgr.Cache().Get(memberID)
	if robot != nil {
		return robot, nil
	}

	// Not in cache, try to load from database
	robot, err = mgr.Cache().LoadByID(ctx, memberID)
	if err != nil {
		return nil, err
	}

	return robot, nil
}

// ListRobots returns robots with pagination and filtering
func ListRobots(ctx *types.Context, query *ListQuery) (*ListResult, error) {
	if query == nil {
		query = &ListQuery{}
	}
	query.applyDefaults()

	mgr, err := getManager()
	if err != nil {
		// Manager not started, load directly from database
		return listRobotsFromDB(query)
	}

	// If only teamID specified AND explicitly filtering for autonomous_mode=true, use cache
	// Cache only contains autonomous_mode=true robots
	// When autonomous_mode is not specified or false, must query database to include all robots
	if query.TeamID != "" && query.Status == "" && query.Keywords == "" && query.ClockMode == "" &&
		query.AutonomousMode != nil && *query.AutonomousMode == true {
		robots := mgr.Cache().List(query.TeamID)
		return paginateRobots(robots, query), nil
	}

	// For complex queries, load from database
	return listRobotsFromDB(query)
}

// GetRobotStatus returns the runtime status of a robot
func GetRobotStatus(ctx *types.Context, memberID string) (*RobotState, error) {
	if memberID == "" {
		return nil, fmt.Errorf("member_id is required")
	}

	robot, err := GetRobot(ctx, memberID)
	if err != nil {
		return nil, err
	}

	// Get permission fields from store (for access control)
	record, _ := robotStore.Get(context.Background(), memberID)

	state := &RobotState{
		MemberID:    robot.MemberID,
		TeamID:      robot.TeamID,
		DisplayName: robot.DisplayName,
		Bio:         robot.Bio,
		Status:      robot.Status,
		MaxRunning:  2, // default
	}

	// Add permission fields if available
	if record != nil {
		state.YaoCreatedBy = record.YaoCreatedBy
		state.YaoTeamID = record.YaoTeamID
	}

	if robot.Config != nil && robot.Config.Quota != nil {
		state.MaxRunning = robot.Config.Quota.GetMax()
	}

	// Get running execution IDs from ExecutionStore (more reliable than in-memory)
	// This ensures we get accurate status even when robot is loaded from database
	runningExecs, err := executionStore.List(context.Background(), &store.ListOptions{
		MemberID: memberID,
		Status:   types.ExecRunning,
		Limit:    100,
	})
	if err == nil && len(runningExecs) > 0 {
		state.Running = len(runningExecs)
		state.RunningIDs = make([]string, 0, len(runningExecs))
		for _, exec := range runningExecs {
			state.RunningIDs = append(state.RunningIDs, exec.ExecutionID)
		}
		// Update status based on running count
		state.Status = types.RobotWorking
	} else {
		// No running executions from store, check in-memory
		executions := robot.GetExecutions()
		state.Running = len(executions)
		state.RunningIDs = make([]string, 0, len(executions))
		for _, exec := range executions {
			state.RunningIDs = append(state.RunningIDs, exec.ID)
		}
		// If there are running executions in memory, update status
		if state.Running > 0 {
			state.Status = types.RobotWorking
		}
	}

	// Set last run time
	if !robot.LastRun.IsZero() {
		state.LastRun = &robot.LastRun
	}

	// Set next run time
	if !robot.NextRun.IsZero() {
		state.NextRun = &robot.NextRun
	}

	return state, nil
}

// ==================== Helper Functions ====================

// loadRobotFromDB loads a robot directly from database
func loadRobotFromDB(memberID string) (*types.Robot, error) {
	m := model.Select(memberModel)
	if m == nil {
		return nil, fmt.Errorf("model %s not found", memberModel)
	}

	records, err := m.Get(model.QueryParam{
		Select: []interface{}{
			"id", "member_id", "team_id", "display_name", "bio",
			"system_prompt", "robot_status", "autonomous_mode",
			"robot_config", "robot_email", "agents", "mcp_servers",
			"manager_id",
		},
		Wheres: []model.QueryWhere{
			{Column: "member_id", Value: memberID},
			{Column: "member_type", Value: "robot"},
		},
		Limit: 1,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to load robot: %w", err)
	}

	if len(records) == 0 {
		return nil, types.ErrRobotNotFound
	}

	return types.NewRobotFromMap(map[string]interface{}(records[0]))
}

// listRobotsFromDB loads robots from database with filtering
func listRobotsFromDB(query *ListQuery) (*ListResult, error) {
	m := model.Select(memberModel)
	if m == nil {
		return nil, fmt.Errorf("model %s not found", memberModel)
	}

	// Build where conditions
	wheres := []model.QueryWhere{
		{Column: "member_type", Value: "robot"},
		{Column: "status", Value: "active"},
	}

	if query.TeamID != "" {
		wheres = append(wheres, model.QueryWhere{Column: "team_id", Value: query.TeamID})
	}
	if query.Status != "" {
		wheres = append(wheres, model.QueryWhere{Column: "robot_status", Value: string(query.Status)})
	}
	if query.Keywords != "" {
		wheres = append(wheres, model.QueryWhere{
			Column: "display_name",
			OP:     "like",
			Value:  "%" + query.Keywords + "%",
		})
	}
	if query.AutonomousMode != nil {
		wheres = append(wheres, model.QueryWhere{Column: "autonomous_mode", Value: *query.AutonomousMode})
	}

	// Build order
	orders := []model.QueryOrder{}
	if query.Order != "" {
		orders = append(orders, model.QueryOrder{Column: query.Order})
	} else {
		orders = append(orders, model.QueryOrder{Column: "created_at", Option: "desc"})
	}

	// Execute paginated query
	result, err := m.Paginate(model.QueryParam{
		Select: []interface{}{
			"id", "member_id", "team_id", "display_name", "bio",
			"system_prompt", "robot_status", "autonomous_mode",
			"robot_config", "robot_email", "agents", "mcp_servers",
		},
		Wheres: wheres,
		Orders: orders,
	}, query.Page, query.PageSize)
	if err != nil {
		return nil, fmt.Errorf("failed to list robots: %w", err)
	}

	// Parse result
	listResult := &ListResult{
		Data:     []*types.Robot{},
		Page:     query.Page,
		PageSize: query.PageSize,
	}

	// Get total count
	if total, ok := result.Get("total").(int); ok {
		listResult.Total = total
	}

	// Parse robot records - handle both []maps.MapStr and []map[string]interface{}
	data := result.Get("data")
	switch records := data.(type) {
	case []maps.MapStr:
		for _, record := range records {
			robot, err := types.NewRobotFromMap(map[string]interface{}(record))
			if err != nil {
				continue // skip invalid records
			}
			listResult.Data = append(listResult.Data, robot)
		}
	case []map[string]interface{}:
		for _, record := range records {
			robot, err := types.NewRobotFromMap(record)
			if err != nil {
				continue // skip invalid records
			}
			listResult.Data = append(listResult.Data, robot)
		}
	}

	return listResult, nil
}

// paginateRobots applies pagination to a slice of robots
func paginateRobots(robots []*types.Robot, query *ListQuery) *ListResult {
	total := len(robots)

	// Calculate offset
	offset := (query.Page - 1) * query.PageSize
	if offset >= total {
		return &ListResult{
			Data:     []*types.Robot{},
			Total:    total,
			Page:     query.Page,
			PageSize: query.PageSize,
		}
	}

	// Calculate end index
	end := offset + query.PageSize
	if end > total {
		end = total
	}

	return &ListResult{
		Data:     robots[offset:end],
		Total:    total,
		Page:     query.Page,
		PageSize: query.PageSize,
	}
}

// ==================== Robot CRUD API ====================
// These functions create, update, and delete robots
// They call store layer for persistence and manage cache
// Request/Response types are defined in types.go

// CreateRobot creates a new robot member
// Calls store.RobotStore.Save() and refreshes cache
// If member_id is not provided, it will be auto-generated
func CreateRobot(ctx *types.Context, req *CreateRobotRequest) (*RobotResponse, error) {
	// Validate required fields
	if req.TeamID == "" {
		return nil, fmt.Errorf("team_id is required")
	}
	if req.DisplayName == "" {
		return nil, fmt.Errorf("display_name is required")
	}

	// Generate member_id if not provided
	if req.MemberID == "" {
		generatedID, err := generateMemberID(context.Background())
		if err != nil {
			return nil, fmt.Errorf("failed to generate member_id: %w", err)
		}
		req.MemberID = generatedID
	}

	// Check if robot already exists
	existing, err := robotStore.Get(context.Background(), req.MemberID)
	if err != nil {
		return nil, fmt.Errorf("failed to check existing robot: %w", err)
	}
	if existing != nil {
		return nil, fmt.Errorf("robot with member_id '%s' already exists", req.MemberID)
	}

	// Determine autonomous_mode value
	autonomousMode := false
	if req.AutonomousMode != nil {
		autonomousMode = *req.AutonomousMode
	}

	// Determine status values
	status := "active"
	if req.Status != "" {
		status = req.Status
	}
	robotStatus := "idle"
	if req.RobotStatus != "" {
		robotStatus = req.RobotStatus
	}

	// Create store record with all fields
	now := time.Now()
	record := &store.RobotRecord{
		// Required
		MemberID:       req.MemberID,
		TeamID:         req.TeamID,
		MemberType:     "robot",
		Status:         status,
		RobotStatus:    robotStatus,
		AutonomousMode: autonomousMode,

		// Profile
		DisplayName: req.DisplayName,
		Bio:         req.Bio,
		Avatar:      req.Avatar,

		// Identity & Role
		SystemPrompt: req.SystemPrompt,
		RoleID:       req.RoleID,
		ManagerID:    req.ManagerID,

		// Communication
		RobotEmail:        req.RobotEmail,
		AuthorizedSenders: req.AuthorizedSenders,
		EmailFilterRules:  req.EmailFilterRules,

		// Capabilities
		RobotConfig:   req.RobotConfig,
		Agents:        req.Agents,
		MCPServers:    req.MCPServers,
		LanguageModel: req.LanguageModel,

		// Limits
		CostLimit: req.CostLimit,

		// Timestamps
		JoinedAt: &now,
	}

	// Apply Yao permission fields if provided
	if req.AuthScope != nil {
		record.YaoCreatedBy = req.AuthScope.CreatedBy
		record.YaoTeamID = req.AuthScope.TeamID
		record.YaoTenantID = req.AuthScope.TenantID
		// Set invited_by from CreatedBy if not explicitly set
		if record.InvitedBy == "" && req.AuthScope.CreatedBy != "" {
			record.InvitedBy = req.AuthScope.CreatedBy
		}
	}

	// Save to database
	err = robotStore.Save(context.Background(), record)
	if err != nil {
		return nil, fmt.Errorf("failed to create robot: %w", err)
	}

	// Refresh cache if manager is running
	// Use Refresh() which handles autonomous_mode correctly:
	// - If autonomous_mode=true: adds to cache for scheduling
	// - If autonomous_mode=false: does not add to cache
	mgr, err := getManager()
	if err == nil && mgr != nil {
		_ = mgr.Cache().Refresh(ctx, req.MemberID)
	}

	// Return the created robot as response
	return GetRobotResponse(ctx, req.MemberID)
}

// UpdateRobot updates an existing robot member
// Calls store.RobotStore.Save() and refreshes cache
func UpdateRobot(ctx *types.Context, memberID string, req *UpdateRobotRequest) (*RobotResponse, error) {
	if memberID == "" {
		return nil, fmt.Errorf("member_id is required")
	}

	// Get existing record
	existing, err := robotStore.Get(context.Background(), memberID)
	if err != nil {
		return nil, fmt.Errorf("failed to get robot: %w", err)
	}
	if existing == nil {
		return nil, types.ErrRobotNotFound
	}

	// Apply updates - only non-nil fields are updated
	// Profile
	if req.DisplayName != nil {
		existing.DisplayName = *req.DisplayName
	}
	if req.Bio != nil {
		existing.Bio = *req.Bio
	}
	if req.Avatar != nil {
		existing.Avatar = *req.Avatar
	}

	// Identity & Role
	if req.SystemPrompt != nil {
		existing.SystemPrompt = *req.SystemPrompt
	}
	if req.RoleID != nil {
		existing.RoleID = *req.RoleID
	}
	if req.ManagerID != nil {
		existing.ManagerID = *req.ManagerID
	}

	// Status
	if req.Status != nil {
		existing.Status = *req.Status
	}
	if req.RobotStatus != nil {
		existing.RobotStatus = *req.RobotStatus
	}
	if req.AutonomousMode != nil {
		existing.AutonomousMode = *req.AutonomousMode
	}

	// Communication
	if req.RobotEmail != nil {
		existing.RobotEmail = *req.RobotEmail
	}
	if req.AuthorizedSenders != nil {
		existing.AuthorizedSenders = req.AuthorizedSenders
	}
	if req.EmailFilterRules != nil {
		existing.EmailFilterRules = req.EmailFilterRules
	}

	// Capabilities
	if req.RobotConfig != nil {
		existing.RobotConfig = req.RobotConfig
	}
	if req.Agents != nil {
		existing.Agents = req.Agents
	}
	if req.MCPServers != nil {
		existing.MCPServers = req.MCPServers
	}
	if req.LanguageModel != nil {
		existing.LanguageModel = *req.LanguageModel
	}

	// Limits
	if req.CostLimit != nil {
		existing.CostLimit = *req.CostLimit
	}

	// Apply Yao permission fields if provided (update scope)
	if req.AuthScope != nil {
		existing.YaoUpdatedBy = req.AuthScope.UpdatedBy
		// Team and Tenant are typically set on create, not update
		// But allow override if explicitly provided
		if req.AuthScope.TeamID != "" {
			existing.YaoTeamID = req.AuthScope.TeamID
		}
		if req.AuthScope.TenantID != "" {
			existing.YaoTenantID = req.AuthScope.TenantID
		}
	}

	// Save to database
	err = robotStore.Save(context.Background(), existing)
	if err != nil {
		return nil, fmt.Errorf("failed to update robot: %w", err)
	}

	// Refresh cache if manager is running
	// Use Refresh() which handles autonomous_mode correctly:
	// - If autonomous_mode=true: adds to cache for scheduling
	// - If autonomous_mode=false: removes from cache
	mgr, err := getManager()
	if err == nil && mgr != nil {
		_ = mgr.Cache().Refresh(ctx, memberID) // Ignore error, database is already saved
	}

	// Return the updated robot as response
	return GetRobotResponse(ctx, memberID)
}

// RemoveRobot deletes a robot member
// Calls store.RobotStore.Delete() and invalidates cache
func RemoveRobot(ctx *types.Context, memberID string) error {
	if memberID == "" {
		return fmt.Errorf("member_id is required")
	}

	// Check if robot exists
	existing, err := robotStore.Get(context.Background(), memberID)
	if err != nil {
		return fmt.Errorf("failed to get robot: %w", err)
	}
	if existing == nil {
		return types.ErrRobotNotFound
	}

	// Check if robot has running executions
	mgr, err := getManager()
	if err == nil && mgr != nil {
		robot := mgr.Cache().Get(memberID)
		if robot != nil && robot.RunningCount() > 0 {
			return fmt.Errorf("cannot delete robot with running executions")
		}
	}

	// Delete from database
	err = robotStore.Delete(context.Background(), memberID)
	if err != nil {
		return fmt.Errorf("failed to delete robot: %w", err)
	}

	// Invalidate cache if manager is running
	if mgr != nil {
		mgr.Cache().Remove(memberID)
	}

	return nil
}

// GetRobotResponse retrieves a robot and converts to API response format
func GetRobotResponse(ctx *types.Context, memberID string) (*RobotResponse, error) {
	record, err := robotStore.Get(context.Background(), memberID)
	if err != nil {
		return nil, fmt.Errorf("failed to get robot: %w", err)
	}
	if record == nil {
		return nil, types.ErrRobotNotFound
	}

	return recordToResponse(record), nil
}

// recordToResponse converts a store.RobotRecord to API RobotResponse
func recordToResponse(record *store.RobotRecord) *RobotResponse {
	return &RobotResponse{
		ID:             record.ID,
		MemberID:       record.MemberID,
		TeamID:         record.TeamID,
		Status:         record.Status,
		RobotStatus:    record.RobotStatus,
		AutonomousMode: record.AutonomousMode,

		DisplayName: record.DisplayName,
		Bio:         record.Bio,
		Avatar:      record.Avatar,

		SystemPrompt: record.SystemPrompt,
		RoleID:       record.RoleID,
		ManagerID:    record.ManagerID,

		RobotEmail:        record.RobotEmail,
		AuthorizedSenders: record.AuthorizedSenders,
		EmailFilterRules:  record.EmailFilterRules,

		RobotConfig:   record.RobotConfig,
		Agents:        record.Agents,
		MCPServers:    record.MCPServers,
		LanguageModel: record.LanguageModel,

		CostLimit:    record.CostLimit,
		InvitedBy:    record.InvitedBy,
		JoinedAt:     record.JoinedAt,
		YaoCreatedBy: record.YaoCreatedBy,
		YaoTeamID:    record.YaoTeamID,
		CreatedAt:    record.CreatedAt,
		UpdatedAt:    record.UpdatedAt,
	}
}

// ==================== Member ID Generation ====================

// generateMemberID generates a unique member_id with collision detection
// Uses 12-digit numeric ID to match existing pattern in openapi/oauth/providers/user
func generateMemberID(ctx context.Context) (string, error) {
	const maxRetries = 10

	for i := 0; i < maxRetries; i++ {
		// Generate 12-digit numeric ID
		id, err := gonanoid.Generate("0123456789", 12)
		if err != nil {
			return "", fmt.Errorf("failed to generate member_id: %w", err)
		}

		// Check if ID already exists
		exists, err := memberIDExists(ctx, id)
		if err != nil {
			return "", fmt.Errorf("failed to check member_id existence: %w", err)
		}

		if !exists {
			return id, nil
		}
		// ID exists, retry
	}

	return "", fmt.Errorf("failed to generate unique member_id after %d retries", maxRetries)
}

// memberIDExists checks if a member_id already exists in the database
func memberIDExists(ctx context.Context, memberID string) (bool, error) {
	m := model.Select(memberModel)
	if m == nil {
		return false, fmt.Errorf("model %s not found", memberModel)
	}

	members, err := m.Get(model.QueryParam{
		Select: []interface{}{"id"},
		Wheres: []model.QueryWhere{
			{Column: "member_id", Value: memberID},
		},
		Limit: 1,
	})

	if err != nil {
		return false, err
	}

	return len(members) > 0, nil
}
