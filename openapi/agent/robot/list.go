package robot

import (
	"strconv"
	"strings"
	"sync"

	"github.com/gin-gonic/gin"
	"github.com/yaoapp/kun/log"
	robotapi "github.com/yaoapp/yao/agent/robot/api"
	robottypes "github.com/yaoapp/yao/agent/robot/types"
	"github.com/yaoapp/yao/openapi/oauth/authorized"
	"github.com/yaoapp/yao/openapi/response"
)

// ListRobots lists robots with pagination and filtering
// GET /v1/agent/robots
func ListRobots(c *gin.Context) {
	// Get authorized information
	authInfo := authorized.GetInfo(c)

	// Parse pagination parameters
	page := 1
	if pageStr := c.Query("page"); pageStr != "" {
		if p, err := strconv.Atoi(pageStr); err == nil && p > 0 {
			page = p
		}
	}

	pageSize := 20
	if pageSizeStr := c.Query("pagesize"); pageSizeStr != "" {
		if ps, err := strconv.Atoi(pageSizeStr); err == nil && ps > 0 && ps <= 100 {
			pageSize = ps
		}
	}

	// Parse filter parameters
	requestedTeamID := strings.TrimSpace(c.Query("team_id"))
	status := strings.TrimSpace(c.Query("status"))
	keywords := strings.TrimSpace(c.Query("keywords"))
	autonomousModeStr := strings.TrimSpace(c.Query("autonomous_mode"))

	// Apply permission-based filtering
	// This ensures users only see robots they have access to:
	// - No constraints: use requested team_id or no filter
	// - TeamOnly: force filter to user's team
	// - OwnerOnly: filter by user_id (personal resources)
	effectiveTeamID := BuildListFilter(c, authInfo, requestedTeamID)

	// Build query
	query := &robotapi.ListQuery{
		TeamID:   effectiveTeamID,
		Keywords: keywords,
		Page:     page,
		PageSize: pageSize,
	}
	if status != "" {
		query.Status = robottypes.RobotStatus(status)
	}
	// Parse autonomous_mode filter: "true" or "false" to filter, empty/other to show all
	if autonomousModeStr == "true" {
		autonomousMode := true
		query.AutonomousMode = &autonomousMode
	} else if autonomousModeStr == "false" {
		autonomousMode := false
		query.AutonomousMode = &autonomousMode
	}

	// Create robot context
	ctx := &robottypes.Context{}

	// Call API layer
	result, err := robotapi.ListRobots(ctx, query)
	if err != nil {
		log.Error("Failed to list robots: %v", err)
		errorResp := &response.ErrorResponse{
			Code:             response.ErrServerError.Code,
			ErrorDescription: "Failed to list robots: " + err.Error(),
		}
		response.RespondWithError(c, response.StatusInternalServerError, errorResp)
		return
	}

	// Convert to HTTP response format with runtime status
	robots := make([]*Response, len(result.Data))
	var wg sync.WaitGroup

	for i, r := range result.Data {
		wg.Add(1)
		go func(idx int, robot *robottypes.Robot) {
			defer wg.Done()
			resp := newResponseFromRobot(robot)

			// Fetch runtime status for each robot
			if status, err := robotapi.GetRobotStatus(ctx, robot.MemberID); err == nil && status != nil {
				resp.Running = status.Running
				resp.MaxRunning = status.MaxRunning
				resp.LastRun = status.LastRun
				resp.NextRun = status.NextRun
				// Use runtime status instead of stored status
				resp.RobotStatus = string(status.Status)
			}

			robots[idx] = resp
		}(i, r)
	}
	wg.Wait()

	resp := &ListResponse{
		Data:     robots,
		Total:    result.Total,
		Page:     result.Page,
		PageSize: result.PageSize,
	}

	response.RespondWithSuccess(c, response.StatusOK, resp)
}

// newResponseFromRobot converts types.Robot to Response
func newResponseFromRobot(r *robottypes.Robot) *Response {
	if r == nil {
		return nil
	}

	return &Response{
		Name:           r.MemberID, // Frontend mapping: name ← member_id
		Description:    r.Bio,      // Frontend mapping: description ← bio
		MemberID:       r.MemberID,
		TeamID:         r.TeamID,
		RobotStatus:    string(r.Status),
		AutonomousMode: r.AutonomousMode,
		DisplayName:    r.DisplayName,
		Bio:            r.Bio,
		SystemPrompt:   r.SystemPrompt,
		RobotEmail:     r.RobotEmail,
	}
}
