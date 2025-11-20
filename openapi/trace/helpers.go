package trace

import (
	"encoding/json"
	"fmt"

	"github.com/gin-gonic/gin"
	"github.com/yaoapp/gou/model"
	"github.com/yaoapp/yao/config"
	"github.com/yaoapp/yao/openapi/oauth/authorized"
	oauthtypes "github.com/yaoapp/yao/openapi/oauth/types"
	"github.com/yaoapp/yao/openapi/response"
	"github.com/yaoapp/yao/trace"
	"github.com/yaoapp/yao/trace/types"
)

// loadTraceManager loads trace manager and info with permission checking
func loadTraceManager(c *gin.Context, traceID string) (manager types.Manager, info *types.TraceInfo, shouldRelease bool, err error) {
	// Get authorized info for permission checking
	authInfo := authorized.GetInfo(c)

	// Get trace info from application configuration
	ctx := c.Request.Context()

	// Get configured driver
	driverType, driverOptions, err := getTraceDriver()
	if err != nil {
		return nil, nil, false, err
	}

	// Get trace info
	info, err = trace.GetInfo(ctx, driverType, traceID, driverOptions...)
	if err != nil {
		return nil, nil, false, fmt.Errorf("trace not found: %w", err)
	}

	// Check read permission
	hasPermission, err := checkTracePermission(authInfo, info)
	if err != nil {
		return nil, nil, false, fmt.Errorf("permission check failed: %w", err)
	}

	if !hasPermission {
		return nil, nil, false, fmt.Errorf("no permission to access trace")
	}

	// Load or get trace manager
	if trace.IsLoaded(traceID) {
		// Get from registry
		manager, err = trace.Load(traceID)
		if err != nil {
			return nil, nil, false, fmt.Errorf("failed to load trace from registry: %w", err)
		}
		return manager, info, false, nil
	}

	// Load from storage
	_, manager, err = trace.LoadFromStorage(ctx, driverType, traceID, driverOptions...)
	if err != nil {
		return nil, nil, false, fmt.Errorf("failed to load trace from storage: %w", err)
	}

	// Return true for shouldRelease since we loaded it temporarily
	return manager, info, true, nil
}

// respondWithLoadError responds with appropriate error based on load error
func respondWithLoadError(c *gin.Context, err error) {
	var statusCode int
	errMsg := err.Error()

	if errMsg == "trace not found" || containsString(errMsg, "trace not found:") {
		statusCode = response.StatusNotFound
	} else if errMsg == "no permission to access trace" || containsString(errMsg, "permission") {
		statusCode = response.StatusForbidden
	} else {
		statusCode = response.StatusInternalServerError
	}

	errorResp := &response.ErrorResponse{
		Code:             response.ErrInvalidRequest.Code,
		ErrorDescription: errMsg,
	}
	response.RespondWithError(c, statusCode, errorResp)
}

// checkTracePermission checks if the user has permission to access the trace
func checkTracePermission(authInfo *oauthtypes.AuthorizedInfo, info *types.TraceInfo) (bool, error) {
	// If no auth info, deny access
	if authInfo == nil {
		return false, nil
	}

	// No constraints, allow access (root/admin)
	if !authInfo.Constraints.TeamOnly && !authInfo.Constraints.OwnerOnly {
		return true, nil
	}

	// Combined Team and Owner permission validation
	if authInfo.Constraints.TeamOnly && authInfo.Constraints.OwnerOnly {
		if info.CreatedBy == authInfo.UserID && info.TeamID == authInfo.TeamID {
			return true, nil
		}
	}

	// Owner only permission validation
	if authInfo.Constraints.OwnerOnly && info.CreatedBy == authInfo.UserID {
		return true, nil
	}

	// Team only permission validation
	if authInfo.Constraints.TeamOnly && info.TeamID == authInfo.TeamID {
		return true, nil
	}

	return false, fmt.Errorf("no permission to access trace: %s", info.ID)
}

// getTraceDriver returns the configured trace driver type and options from global config
func getTraceDriver() (driverType string, driverOptions []any, err error) {
	cfg := config.Conf

	switch cfg.Trace.Driver {
	case "store":
		if cfg.Trace.Store == "" {
			return "", nil, fmt.Errorf("trace store ID not configured")
		}
		return trace.Store, []any{cfg.Trace.Store, cfg.Trace.Prefix}, nil

	case "local", "":
		return trace.Local, []any{cfg.Trace.Path}, nil

	default:
		return "", nil, fmt.Errorf("unsupported trace driver: %s", cfg.Trace.Driver)
	}
}

// formatUpdateData formats trace update data as JSON string
func formatUpdateData(update types.TraceUpdate) string {
	// Use proper JSON marshaling
	data, err := json.Marshal(update)
	if err != nil {
		// Fallback to basic JSON if marshaling fails
		return fmt.Sprintf(`{"traceId":"%s","type":"%s","timestamp":%d,"error":"failed to marshal data"}`,
			update.TraceID, update.Type, update.Timestamp)
	}
	return string(data)
}

// containsString checks if a string contains a substring
func containsString(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && findSubstring(s, substr))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// AuthFilter returns query filters based on authorization info
// This can be used when listing traces with permission filtering
func AuthFilter(c *gin.Context, authInfo *oauthtypes.AuthorizedInfo) []model.QueryWhere {
	var wheres []model.QueryWhere

	if authInfo == nil {
		return wheres
	}

	// No constraints, no filters needed
	if !authInfo.Constraints.TeamOnly && !authInfo.Constraints.OwnerOnly {
		return wheres
	}

	// Combined Team and Owner constraint
	if authInfo.Constraints.TeamOnly && authInfo.Constraints.OwnerOnly {
		wheres = append(wheres, model.QueryWhere{
			Column: "__yao_created_by",
			Value:  authInfo.UserID,
		})
		wheres = append(wheres, model.QueryWhere{
			Column: "__yao_team_id",
			Value:  authInfo.TeamID,
		})
		return wheres
	}

	// Owner only constraint
	if authInfo.Constraints.OwnerOnly {
		wheres = append(wheres, model.QueryWhere{
			Column: "__yao_created_by",
			Value:  authInfo.UserID,
		})
		return wheres
	}

	// Team only constraint
	if authInfo.Constraints.TeamOnly {
		wheres = append(wheres, model.QueryWhere{
			Column: "__yao_team_id",
			Value:  authInfo.TeamID,
		})
		return wheres
	}

	return wheres
}
