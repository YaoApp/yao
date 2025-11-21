package trace

import (
	"github.com/gin-gonic/gin"
	"github.com/yaoapp/yao/openapi/response"
	"github.com/yaoapp/yao/trace"
)

// GetNodes retrieves all nodes in the trace
// GET /api/__yao/openapi/v1/trace/traces/:traceID/nodes
func GetNodes(c *gin.Context) {
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

	// Get all nodes from manager (reads from storage)
	nodes, err := manager.GetAllNodes()
	if err != nil {
		errorResp := &response.ErrorResponse{
			Code:             response.ErrServerError.Code,
			ErrorDescription: "Failed to get nodes: " + err.Error(),
		}
		response.RespondWithError(c, response.StatusInternalServerError, errorResp)
		return
	}

	// Prepare response - return flat list of nodes with basic info
	nodeList := make([]gin.H, 0, len(nodes))
	for _, node := range nodes {
		nodeInfo := gin.H{
			"id":          node.ID,
			"parent_ids":  node.ParentIDs,
			"label":       node.Label,
			"type":        node.Type,
			"icon":        node.Icon,
			"description": node.Description,
			"status":      node.Status,
			"created_at":  node.CreatedAt,
			"start_time":  node.StartTime,
			"end_time":    node.EndTime,
			"updated_at":  node.UpdatedAt,
		}

		if node.Metadata != nil {
			nodeInfo["metadata"] = node.Metadata
		}

		nodeList = append(nodeList, nodeInfo)
	}

	response.RespondWithSuccess(c, response.StatusOK, gin.H{
		"trace_id": traceID,
		"nodes":    nodeList,
		"count":    len(nodeList),
	})
}

// GetNode retrieves a single node by ID
// GET /api/__yao/openapi/v1/trace/traces/:traceID/nodes/:nodeID
func GetNode(c *gin.Context) {
	// Get trace ID and node ID from URL parameters
	traceID := c.Param("traceID")
	nodeID := c.Param("nodeID")

	if traceID == "" || nodeID == "" {
		errorResp := &response.ErrorResponse{
			Code:             response.ErrInvalidRequest.Code,
			ErrorDescription: "Trace ID and Node ID are required",
		}
		response.RespondWithError(c, response.StatusBadRequest, errorResp)
		return
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

	// Get node by ID from manager (reads from storage)
	node, err := manager.GetNodeByID(nodeID)
	if err != nil {
		errorResp := &response.ErrorResponse{
			Code:             response.ErrServerError.Code,
			ErrorDescription: "Failed to get node: " + err.Error(),
		}
		response.RespondWithError(c, response.StatusInternalServerError, errorResp)
		return
	}

	if node == nil {
		errorResp := &response.ErrorResponse{
			Code:             response.ErrInvalidRequest.Code,
			ErrorDescription: "Node not found",
		}
		response.RespondWithError(c, response.StatusNotFound, errorResp)
		return
	}

	// Prepare detailed node response
	nodeData := gin.H{
		"id":          node.ID,
		"parent_ids":  node.ParentIDs,
		"label":       node.Label,
		"type":        node.Type,
		"icon":        node.Icon,
		"description": node.Description,
		"status":      node.Status,
		"input":       node.Input,
		"output":      node.Output,
		"created_at":  node.CreatedAt,
		"start_time":  node.StartTime,
		"end_time":    node.EndTime,
		"updated_at":  node.UpdatedAt,
	}

	if node.Metadata != nil {
		nodeData["metadata"] = node.Metadata
	}

	// Add children IDs (not full children objects to avoid deep nesting)
	if len(node.Children) > 0 {
		childrenIDs := make([]string, 0, len(node.Children))
		for _, child := range node.Children {
			childrenIDs = append(childrenIDs, child.ID)
		}
		nodeData["children_ids"] = childrenIDs
	}

	response.RespondWithSuccess(c, response.StatusOK, nodeData)
}
