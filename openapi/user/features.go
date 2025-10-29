package user

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/yaoapp/kun/log"
	"github.com/yaoapp/yao/openapi/oauth/acl"
	"github.com/yaoapp/yao/openapi/oauth/authorized"
	"github.com/yaoapp/yao/openapi/response"
)

// GinFeatures handles GET /user/features - Get features for the current user
// Supports optional domain filtering via query parameter: ?domain=user/team
func GinFeatures(c *gin.Context) {
	// Get authorized user info
	authInfo := authorized.GetInfo(c)
	if authInfo == nil || authInfo.UserID == "" {
		errorResp := &response.ErrorResponse{
			Code:             response.ErrInvalidClient.Code,
			ErrorDescription: "User not authenticated",
		}
		response.RespondWithError(c, response.StatusUnauthorized, errorResp)
		return
	}

	// Parse query parameters for optional domain filtering
	domain := c.Query("domain")

	// Get features from ACL
	var features map[string]bool
	var err error

	if domain != "" {
		// Get features filtered by domain
		features, err = acl.GetFeaturesByDomain(c, domain)
	} else {
		// Get all features
		features, err = acl.GetFeatures(c)
	}

	if err != nil {
		log.Error("Failed to get user features: %v", err)
		errorResp := &response.ErrorResponse{
			Code:             response.ErrServerError.Code,
			ErrorDescription: "Failed to retrieve user features",
		}
		response.RespondWithError(c, response.StatusInternalServerError, errorResp)
		return
	}

	// Return features map directly for O(1) frontend lookups
	// Frontend can check: if (features["profile:write"]) { ... }
	response.RespondWithSuccess(c, http.StatusOK, features)
}
