package user

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/yaoapp/kun/log"
	"github.com/yaoapp/yao/openapi/oauth"
	oauthtypes "github.com/yaoapp/yao/openapi/oauth/types"
	"github.com/yaoapp/yao/openapi/response"
)

// User Profile Management Handlers

// GinProfileGet handles GET /profile - Get current user profile
func GinProfileGet(c *gin.Context) {
	// Get authorized user info
	authInfo := oauth.GetAuthorizedInfo(c)
	if authInfo == nil || authInfo.UserID == "" {
		errorResp := &response.ErrorResponse{
			Code:             response.ErrInvalidClient.Code,
			ErrorDescription: "User not authenticated",
		}
		response.RespondWithError(c, response.StatusUnauthorized, errorResp)
		return
	}

	// Get user provider
	userProvider, err := oauth.OAuth.GetUserProvider()
	if err != nil {
		log.Error("Failed to get user provider: %v", err)
		errorResp := &response.ErrorResponse{
			Code:             response.ErrServerError.Code,
			ErrorDescription: "Failed to get user provider",
		}
		response.RespondWithError(c, response.StatusInternalServerError, errorResp)
		return
	}

	// Get user data with scopes
	ctx := c.Request.Context()
	if ctx == nil {
		ctx = context.Background()
	}

	user, err := userProvider.GetUserWithScopes(ctx, authInfo.UserID)
	if err != nil {
		log.Error("Failed to get user profile: %v", err)
		errorResp := &response.ErrorResponse{
			Code:             response.ErrServerError.Code,
			ErrorDescription: "Failed to retrieve user profile",
		}
		response.RespondWithError(c, response.StatusInternalServerError, errorResp)
		return
	}

	// Get Yao client config
	yaoClientConfig := GetYaoClientConfig()

	// Get or create subject
	subject, err := oauth.OAuth.Subject(yaoClientConfig.ClientID, authInfo.UserID)
	if err != nil {
		log.Warn("Failed to get user subject: %s", err.Error())
		subject = authInfo.UserID // Fallback to user ID
	}

	// Prepare OIDC user info (same format as login response)
	oidcUserInfo := oauthtypes.MakeOIDCUserInfo(user)
	oidcUserInfo.Sub = subject
	oidcUserInfo.YaoUserID = authInfo.UserID

	// Add team context if available from token
	if authInfo.TeamID != "" {
		// Get team details
		team, err := userProvider.GetTeamByMember(ctx, authInfo.TeamID, authInfo.UserID)
		if err == nil && team != nil {
			// Add team info to OIDC user info
			oidcUserInfo.YaoTeamID = authInfo.TeamID

			teamInfo := &oauthtypes.OIDCTeamInfo{}
			if teamIDVal := toString(team["team_id"]); teamIDVal != "" {
				teamInfo.TeamID = teamIDVal
			}
			if logo := toString(team["logo"]); logo != "" {
				teamInfo.Logo = logo
			}
			if name := toString(team["name"]); name != "" {
				teamInfo.Name = name
			}
			if description := toString(team["description"]); description != "" {
				teamInfo.Description = description
			}
			if ownerID := toString(team["owner_id"]); ownerID != "" {
				teamInfo.OwnerID = ownerID

				// Check if user is owner
				if ownerID == authInfo.UserID {
					isOwner := true
					oidcUserInfo.YaoIsOwner = &isOwner
				}
			}
			oidcUserInfo.YaoTeam = teamInfo

			// Add tenant_id if available from the team
			if tenantID := toString(team["tenant_id"]); tenantID != "" {
				oidcUserInfo.YaoTenantID = tenantID
			}
		}
	}

	// Add type information
	var typeID string
	if authInfo.TeamID != "" {
		// Team context - try to get team's type first
		team, err := userProvider.GetTeamByMember(ctx, authInfo.TeamID, authInfo.UserID)
		if err == nil && team != nil {
			typeID = toString(team["type_id"])
		}
	}

	// Fallback to user's type if no team type
	if typeID == "" {
		typeID = toString(user["type_id"])
	}

	if typeID != "" {
		oidcUserInfo.YaoTypeID = typeID

		// Get type details
		typeInfo, err := userProvider.GetType(ctx, typeID)
		if err == nil && typeInfo != nil {
			typeDetails := &oauthtypes.OIDCTypeInfo{}
			if typeIDVal := toString(typeInfo["type_id"]); typeIDVal != "" {
				typeDetails.TypeID = typeIDVal
			}
			if name := toString(typeInfo["name"]); name != "" {
				typeDetails.Name = name
			}
			if locale := toString(typeInfo["locale"]); locale != "" {
				typeDetails.Locale = locale
			}
			oidcUserInfo.YaoType = typeDetails
		}
	}

	// Convert to map for response
	profileData := oidcUserInfo.Map()

	// Return user profile
	response.RespondWithSuccess(c, http.StatusOK, profileData)
}
