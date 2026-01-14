package cache

import (
	"fmt"

	"github.com/yaoapp/gou/model"
	"github.com/yaoapp/kun/maps"
	"github.com/yaoapp/yao/agent/robot/types"
)

// memberModel is the model name for member table
// Can be changed via SetMemberModel() during system initialization
var memberModel = "__yao.member"

// memberFields are the fields to select when loading robots
var memberFields = []interface{}{
	"id",
	"member_id",
	"team_id",
	"display_name",
	"system_prompt",
	"robot_status",
	"autonomous_mode",
	"robot_config",
}

// SetMemberModel sets the member model name
// Call this during system initialization to override the default
func SetMemberModel(model string) {
	if model != "" {
		memberModel = model
	}
}

// Load loads all active robots from database with pagination
// Query: member_type='robot' AND autonomous_mode=true AND status='active'
func (c *Cache) Load(ctx *types.Context) error {
	m := model.Select(memberModel)

	// Clear existing cache first
	c.mu.Lock()
	c.robots = make(map[string]*types.Robot)
	c.byTeam = make(map[string][]string)
	c.mu.Unlock()

	// Paginate to handle large number of robots
	page := 1
	pageSize := 100 // load 100 robots per page
	totalLoaded := 0

	for {
		// Query with pagination
		result, err := m.Paginate(model.QueryParam{
			Select: memberFields,
			Wheres: []model.QueryWhere{
				{Column: "member_type", Value: "robot"},
				{Column: "autonomous_mode", Value: true},
				{Column: "status", Value: "active"},
			},
		}, page, pageSize)
		if err != nil {
			return fmt.Errorf("failed to load robots (page %d): %w", page, err)
		}

		// Extract records from pagination result
		data, ok := result.Get("data").([]maps.MapStr)
		if !ok || len(data) == 0 {
			break
		}

		// Parse and add each robot
		for _, record := range data {
			robot, err := types.NewRobotFromMap(map[string]interface{}(record))
			if err != nil {
				// Log error but continue loading other robots
				continue
			}
			c.Add(robot)
			totalLoaded++
		}

		// Check if there are more pages
		total, _ := result.Get("total").(int)
		if totalLoaded >= total {
			break
		}

		page++
	}

	return nil
}

// LoadByID loads a single robot from database by member ID
func (c *Cache) LoadByID(ctx *types.Context, memberID string) (*types.Robot, error) {
	m := model.Select(memberModel)

	records, err := m.Get(model.QueryParam{
		Select: memberFields,
		Wheres: []model.QueryWhere{
			{Column: "member_id", Value: memberID},
			{Column: "member_type", Value: "robot"},
		},
		Limit: 1,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to load robot %s: %w", memberID, err)
	}

	if len(records) == 0 {
		return nil, types.ErrRobotNotFound
	}

	return types.NewRobotFromMap(map[string]interface{}(records[0]))
}
