package app

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/yaoapp/gou/process"
	"github.com/yaoapp/yao/openapi/oauth/authorized"
	"github.com/yaoapp/yao/openapi/oauth/types"
	"github.com/yaoapp/yao/openapi/response"
)

// Attach attaches the app handlers to the router
func Attach(group *gin.RouterGroup, oauth types.OAuth) {
	// Menu endpoint - requires authentication
	group.GET("/menu", oauth.Guard, getMenu)
}

// MenuRequest represents the menu request parameters
type MenuRequest struct {
	Locale string `form:"locale" json:"locale"`
}

// getMenu handles GET /app/menu
// Returns the application menu based on user permissions and locale
func getMenu(c *gin.Context) {
	var req MenuRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		response.RespondWithError(c, http.StatusBadRequest, &response.ErrorResponse{
			Code:             response.ErrInvalidRequest.Code,
			ErrorDescription: err.Error(),
		})
		return
	}

	// Get authorized info from context (set by oauth.Guard)
	authInfo := authorized.GetInfo(c)
	if authInfo == nil {
		response.RespondWithError(c, http.StatusUnauthorized, &response.ErrorResponse{
			Code:             response.ErrInvalidToken.Code,
			ErrorDescription: "Authorization required",
		})
		return
	}

	// Call yao.app.Menu process with locale parameter
	handle, err := process.Of("yao.app.Menu", req.Locale)
	if err != nil {
		response.RespondWithError(c, http.StatusBadRequest, &response.ErrorResponse{
			Code:             response.ErrInvalidRequest.Code,
			ErrorDescription: err.Error(),
		})
		return
	}

	// Set process context
	handle.WithSID(authInfo.SessionID)

	// Set authorized info for process
	handle.WithAuthorized(map[string]interface{}{
		"subject":     authInfo.Subject,
		"client_id":   authInfo.ClientID,
		"user_id":     authInfo.UserID,
		"scope":       authInfo.Scope,
		"team_id":     authInfo.TeamID,
		"tenant_id":   authInfo.TenantID,
		"session_id":  authInfo.SessionID,
		"remember_me": authInfo.RememberMe,
		"constraints": map[string]interface{}{
			"owner_only":   authInfo.Constraints.OwnerOnly,
			"creator_only": authInfo.Constraints.CreatorOnly,
			"editor_only":  authInfo.Constraints.EditorOnly,
			"team_only":    authInfo.Constraints.TeamOnly,
			"extra":        authInfo.Constraints.Extra,
		},
	})

	// Execute the process
	err = handle.Execute()
	if err != nil {
		response.RespondWithError(c, http.StatusInternalServerError, &response.ErrorResponse{
			Code:             response.ErrServerError.Code,
			ErrorDescription: err.Error(),
		})
		return
	}
	defer handle.Dispose()

	// Return the menu data
	response.RespondWithSuccess(c, http.StatusOK, handle.Value())
}
