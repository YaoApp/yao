package trace

import (
	"fmt"
	"io"

	"github.com/gin-gonic/gin"
	"github.com/yaoapp/yao/openapi/response"
	"github.com/yaoapp/yao/trace"
	"github.com/yaoapp/yao/trace/types"
)

// GetEvents retrieves all trace events
// GET /api/__yao/openapi/v1/trace/traces/:traceID/events?stream=true
func GetEvents(c *gin.Context) {
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

	// Load trace manager and info with permission checking
	manager, info, shouldRelease, err := loadTraceManager(c, traceID)
	if err != nil {
		respondWithLoadError(c, err)
		return
	}

	// Release after use if we loaded it temporarily
	if shouldRelease {
		defer trace.Release(traceID)
	}

	// Check if stream mode is requested
	streamMode := c.Query("stream") == "true"

	// Handle streaming mode
	if streamMode {
		handleStreamMode(c, manager, info)
		return
	}

	// Handle normal mode - return all events
	handleNormalMode(c, manager, info)
}

// handleStreamMode handles streaming mode for trace events (SSE)
func handleStreamMode(c *gin.Context, manager types.Manager, info *types.TraceInfo) {
	// Set SSE headers
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("X-Accel-Buffering", "no")

	// Subscribe to trace updates
	updates, err := manager.Subscribe()
	if err != nil {
		// Send error as SSE event
		fmt.Fprintf(c.Writer, "event: error\ndata: {\"error\":\"Failed to subscribe: %s\"}\n\n", err.Error())
		c.Writer.Flush()
		return
	}

	// Stream events
	ctx := c.Request.Context()
	clientGone := ctx.Done()

	for {
		select {
		case <-clientGone:
			// Client disconnected
			return

		case update, ok := <-updates:
			if !ok {
				// Channel closed
				return
			}

			// Format and send SSE event
			err := sendSSEEvent(c.Writer, *update)
			if err != nil {
				return
			}

			// Check if trace is complete
			if update.Type == types.UpdateTypeComplete {
				return
			}
		}
	}
}

// handleNormalMode handles normal mode for trace events (JSON array)
func handleNormalMode(c *gin.Context, manager types.Manager, info *types.TraceInfo) {
	// Get all events from the beginning (timestamp 0 = all)
	events, err := manager.GetEvents(0)
	if err != nil {
		errorResp := &response.ErrorResponse{
			Code:             response.ErrServerError.Code,
			ErrorDescription: "Failed to get events: " + err.Error(),
		}
		response.RespondWithError(c, response.StatusInternalServerError, errorResp)
		return
	}

	// Determine trace status
	var traceStatus types.TraceStatus
	if manager.IsComplete() {
		// Check last event for actual completion status
		traceStatus = types.TraceStatusCompleted
		for i := len(events) - 1; i >= 0; i-- {
			if events[i].Type == types.UpdateTypeComplete {
				if data, ok := events[i].Data.(*types.TraceCompleteData); ok {
					traceStatus = data.Status
				}
				break
			}
		}
	} else {
		traceStatus = types.TraceStatusRunning
		// Check if there are any events yet
		if len(events) == 0 || (len(events) == 1 && events[0].Type == types.UpdateTypeInit) {
			traceStatus = types.TraceStatusPending
		}
	}

	// Override with stored status if it indicates failure or cancellation
	switch info.Status {
	case types.TraceStatusFailed:
		traceStatus = types.TraceStatusFailed
	case types.TraceStatusCancelled:
		traceStatus = types.TraceStatusCancelled
	}

	// Prepare response data
	eventsData := gin.H{
		"id":         info.ID,
		"status":     traceStatus,
		"created_at": info.CreatedAt,
		"updated_at": info.UpdatedAt,
		"archived":   info.Archived,
		"events":     events,
	}

	if info.ArchivedAt != nil {
		eventsData["archived_at"] = *info.ArchivedAt
	}

	response.RespondWithSuccess(c, response.StatusOK, eventsData)
}

// sendSSEEvent sends a trace update as an SSE event
func sendSSEEvent(w io.Writer, update types.TraceUpdate) error {
	// Write event type
	_, err := fmt.Fprintf(w, "event: %s\n", update.Type)
	if err != nil {
		return err
	}

	// Write data (JSON format)
	dataJSON := formatUpdateData(update)
	_, err = fmt.Fprintf(w, "data: %s\n\n", dataJSON)
	if err != nil {
		return err
	}

	// Flush to client
	if flusher, ok := w.(gin.ResponseWriter); ok {
		flusher.Flush()
	}

	return nil
}
