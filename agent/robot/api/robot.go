package api

import (
	"fmt"

	"github.com/yaoapp/gou/model"
	"github.com/yaoapp/kun/maps"
	"github.com/yaoapp/yao/agent/robot/types"
)

// ==================== Robot Query API ====================
// These functions query robot information

// memberModel is the model name for member table
const memberModel = "__yao.member"

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

	// If only teamID specified (no other filters), use cache for faster lookup
	// Note: Cache only contains autonomous_mode=true robots, so this is safe
	if query.TeamID != "" && query.Status == "" && query.Keywords == "" && query.ClockMode == "" {
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

	state := &RobotState{
		MemberID:    robot.MemberID,
		TeamID:      robot.TeamID,
		DisplayName: robot.DisplayName,
		Status:      robot.Status,
		Running:     robot.RunningCount(),
		MaxRunning:  2, // default
	}

	if robot.Config != nil && robot.Config.Quota != nil {
		state.MaxRunning = robot.Config.Quota.GetMax()
	}

	// Get running execution IDs
	executions := robot.GetExecutions()
	state.RunningIDs = make([]string, 0, len(executions))
	for _, exec := range executions {
		state.RunningIDs = append(state.RunningIDs, exec.ID)
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
			"id", "member_id", "team_id", "display_name",
			"system_prompt", "robot_status", "autonomous_mode",
			"robot_config", "robot_email",
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
		{Column: "autonomous_mode", Value: true},
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
			"id", "member_id", "team_id", "display_name",
			"system_prompt", "robot_status", "autonomous_mode",
			"robot_config", "robot_email",
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
