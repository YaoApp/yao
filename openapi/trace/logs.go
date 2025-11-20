package trace

import (
	"github.com/gin-gonic/gin"
	"github.com/yaoapp/yao/openapi/response"
	"github.com/yaoapp/yao/trace"
	"github.com/yaoapp/yao/trace/types"
)

// GetLogs retrieves logs for a trace or specific node
// GET /api/__yao/openapi/v1/trace/traces/:traceID/logs?node_id=xxx
func GetLogs(c *gin.Context) {
	// Get trace ID from URL parameter
	traceID := c.Param("traceID")
	if traceID == "" {
		errorResp := &response.ErrorResponse{
			Code:             response.ErrInvalidRequest.Code,
			ErrorDescription: "Trace ID is required",
		}
		response.RespondWithError(c, response.StatusBadRequest, errorResp)
		return
	}

	// Get optional node_id from URL parameter or query parameter
	nodeID := c.Param("nodeID")
	if nodeID == "" {
		nodeID = c.Query("node_id")
	}

	// Load trace manager with permission checking
	manager, _, shouldRelease, err := loadTraceManager(c, traceID)
	if err != nil {
		respondWithLoadError(c, err)
		return
	}

	// Release after use if we loaded it temporarily
	if shouldRelease {
		defer trace.Release(traceID)
	}

	// Get logs from manager (reads from storage)
	var logs []*types.TraceLog
	if nodeID != "" {
		// Get logs for specific node
		logs, err = manager.GetLogsByNode(nodeID)
		if err != nil {
			errorResp := &response.ErrorResponse{
				Code:             response.ErrServerError.Code,
				ErrorDescription: "Failed to get logs for node: " + err.Error(),
			}
			response.RespondWithError(c, response.StatusInternalServerError, errorResp)
			return
		}
	} else {
		// Get all logs
		logs, err = manager.GetAllLogs()
		if err != nil {
			errorResp := &response.ErrorResponse{
				Code:             response.ErrServerError.Code,
				ErrorDescription: "Failed to get logs: " + err.Error(),
			}
			response.RespondWithError(c, response.StatusInternalServerError, errorResp)
			return
		}
	}

	// Prepare response
	logList := make([]gin.H, 0, len(logs))
	for _, log := range logs {
		logInfo := gin.H{
			"timestamp": log.Timestamp,
			"level":     log.Level,
			"message":   log.Message,
			"node_id":   log.NodeID,
		}

		if len(log.Data) > 0 {
			logInfo["data"] = log.Data
		}

		logList = append(logList, logInfo)
	}

	responseData := gin.H{
		"trace_id": traceID,
		"logs":     logList,
		"count":    len(logList),
	}

	if nodeID != "" {
		responseData["node_id"] = nodeID
	}

	response.RespondWithSuccess(c, response.StatusOK, responseData)
}
