package robot

import (
	"context"
	"fmt"

	"github.com/gin-gonic/gin"
	robotstore "github.com/yaoapp/yao/agent/robot/store"
	robottypes "github.com/yaoapp/yao/agent/robot/types"
	"github.com/yaoapp/yao/openapi/chat"
	"github.com/yaoapp/yao/openapi/response"
)

// resolveHostAssistantID resolves the host assistant ID from a robot member ID.
// It fetches the RobotRecord, parses its config, and returns the PhaseHost agent ID.
func resolveHostAssistantID(ctx context.Context, memberID string) (string, *robotstore.RobotRecord, error) {
	store := robotstore.NewRobotStore()
	record, err := store.Get(ctx, memberID)
	if err != nil {
		return "", nil, fmt.Errorf("failed to get robot: %w", err)
	}
	if record == nil {
		return "", nil, fmt.Errorf("robot not found: %s", memberID)
	}

	config, err := robottypes.ParseConfig(record.RobotConfig)
	if err != nil {
		return "", nil, fmt.Errorf("failed to parse robot config: %w", err)
	}

	hostID := robottypes.ResolvePhaseAgent(config, robottypes.PhaseHost)
	if hostID == "" {
		return "", nil, fmt.Errorf("no Host Agent configured for robot %s (set uses.host in agent.yml or resources.phases in robot config)", memberID)
	}

	return hostID, record, nil
}

// injectAssistantID sets the assistant_id query parameter on the gin request,
// so that downstream GetCompletionRequest can pick it up.
func injectAssistantID(c *gin.Context, assistantID string) {
	q := c.Request.URL.Query()
	q.Set("assistant_id", assistantID)
	c.Request.URL.RawQuery = q.Encode()
}

// RobotCompletions handles POST /v1/agent/robots/:id/completions
// Mirror API that resolves the robot's host assistant and delegates to standard chat completions.
func RobotCompletions(c *gin.Context) {
	robotID := c.Param("id")
	if robotID == "" {
		response.RespondWithError(c, response.StatusBadRequest, &response.ErrorResponse{
			Code:             response.ErrInvalidRequest.Code,
			ErrorDescription: "robot id is required",
		})
		return
	}

	hostID, _, err := resolveHostAssistantID(c.Request.Context(), robotID)
	if err != nil {
		response.RespondWithError(c, response.StatusInternalServerError, &response.ErrorResponse{
			Code:             response.ErrServerError.Code,
			ErrorDescription: err.Error(),
		})
		return
	}

	injectAssistantID(c, hostID)
	chat.GinCreateCompletions(c)
}

// RobotAppendMessages handles POST /v1/agent/robots/:id/completions/:context_id/append
// Mirror API that resolves the robot's host assistant and delegates to standard append.
func RobotAppendMessages(c *gin.Context) {
	robotID := c.Param("id")
	if robotID == "" {
		response.RespondWithError(c, response.StatusBadRequest, &response.ErrorResponse{
			Code:             response.ErrInvalidRequest.Code,
			ErrorDescription: "robot id is required",
		})
		return
	}

	hostID, _, err := resolveHostAssistantID(c.Request.Context(), robotID)
	if err != nil {
		response.RespondWithError(c, response.StatusInternalServerError, &response.ErrorResponse{
			Code:             response.ErrServerError.Code,
			ErrorDescription: err.Error(),
		})
		return
	}

	injectAssistantID(c, hostID)
	chat.GinAppendMessages(c)
}

// RobotHostID handles GET /v1/agent/robots/:id/host
// Returns the host assistant ID for a robot (used by frontend to know which assistant to chat with).
func RobotHostID(c *gin.Context) {
	robotID := c.Param("id")
	if robotID == "" {
		response.RespondWithError(c, response.StatusBadRequest, &response.ErrorResponse{
			Code:             response.ErrInvalidRequest.Code,
			ErrorDescription: "robot id is required",
		})
		return
	}

	hostID, _, err := resolveHostAssistantID(c.Request.Context(), robotID)
	if err != nil {
		response.RespondWithError(c, response.StatusInternalServerError, &response.ErrorResponse{
			Code:             response.ErrServerError.Code,
			ErrorDescription: err.Error(),
		})
		return
	}

	response.RespondWithSuccess(c, response.StatusOK, gin.H{
		"assistant_id": hostID,
		"robot_id":     robotID,
	})
}
