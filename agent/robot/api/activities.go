package api

import (
	"context"
	"fmt"
	"time"

	"github.com/yaoapp/yao/agent/robot/store"
	"github.com/yaoapp/yao/agent/robot/types"
)

// ==================== Activity Types ====================

// ActivityQuery - query parameters for listing activities
type ActivityQuery struct {
	TeamID string     `json:"team_id,omitempty"` // Filter by team ID
	Limit  int        `json:"limit,omitempty"`
	Since  *time.Time `json:"since,omitempty"` // Only activities after this time
	Type   string     `json:"type,omitempty"`  // Filter by activity type: execution.started, execution.completed, execution.failed, execution.cancelled
}

// Activity - activity item for feed
type Activity struct {
	Type        store.ActivityType `json:"type"`
	RobotID     string             `json:"robot_id"`
	RobotName   string             `json:"robot_name,omitempty"` // Display name from robot
	ExecutionID string             `json:"execution_id"`
	Message     string             `json:"message"`
	Timestamp   time.Time          `json:"timestamp"`
}

// ActivityListResponse - response with activities
type ActivityListResponse struct {
	Data []*Activity `json:"data"`
}

// ==================== Activity API Functions ====================

// ListActivities returns recent activities for a team
// Activities are derived from execution status changes
func ListActivities(ctx *types.Context, query *ActivityQuery) (*ActivityListResponse, error) {
	if query == nil {
		query = &ActivityQuery{}
	}
	query.applyDefaults()

	// Build store options
	opts := &store.ActivityListOptions{
		Limit: query.Limit,
		Since: query.Since,
	}

	if query.TeamID != "" {
		opts.TeamID = query.TeamID
	}

	// Pass type filter if provided
	if query.Type != "" {
		opts.Type = store.ActivityType(query.Type)
	}

	// Query from store
	storeActivities, err := getExecutionStore().ListActivities(context.Background(), opts)
	if err != nil {
		return nil, fmt.Errorf("failed to list activities: %w", err)
	}

	// Transform to Activity slice
	// Also enrich with robot display names
	activities := make([]*Activity, 0, len(storeActivities))
	robotNames := make(map[string]string) // Cache robot names

	for _, sa := range storeActivities {
		activity := &Activity{
			Type:        sa.Type,
			RobotID:     sa.RobotID,
			ExecutionID: sa.ExecutionID,
			Message:     sa.Message,
			Timestamp:   sa.Timestamp,
		}

		// Try to get robot name (with caching)
		if name, ok := robotNames[sa.RobotID]; ok {
			activity.RobotName = name
		} else {
			// Try to get robot display name
			robotResp, err := GetRobotResponse(ctx, sa.RobotID)
			if err == nil && robotResp != nil {
				activity.RobotName = robotResp.DisplayName
				robotNames[sa.RobotID] = robotResp.DisplayName
			}
		}

		activities = append(activities, activity)
	}

	return &ActivityListResponse{
		Data: activities,
	}, nil
}

// ==================== Helper Functions ====================

// applyDefaults applies default values to ActivityQuery
func (q *ActivityQuery) applyDefaults() {
	if q.Limit <= 0 {
		q.Limit = 20
	}
	if q.Limit > 100 {
		q.Limit = 100
	}
}
