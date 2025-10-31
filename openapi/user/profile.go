package user

import (
	"context"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/yaoapp/gou/process"
	"github.com/yaoapp/gou/session"
	"github.com/yaoapp/kun/exception"
	"github.com/yaoapp/kun/log"
	"github.com/yaoapp/kun/maps"
	"github.com/yaoapp/yao/openapi/oauth"
	"github.com/yaoapp/yao/openapi/oauth/authorized"
	"github.com/yaoapp/yao/openapi/oauth/providers/user"
	oauthtypes "github.com/yaoapp/yao/openapi/oauth/types"
	"github.com/yaoapp/yao/openapi/response"
	"github.com/yaoapp/yao/openapi/utils"
)

// User Profile Management Handlers

// GinProfileGet handles GET /profile - Get current user profile
func GinProfileGet(c *gin.Context) {
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

	// Parse query parameters
	var req ProfileGetRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		// Ignore binding errors for optional parameters
		req = ProfileGetRequest{}
	}

	// Call business logic to get profile
	profile, err := profileGet(c.Request.Context(), authInfo.UserID, authInfo.TeamID, req)
	if err != nil {
		log.Error("Failed to get user profile: %v", err)
		errorResp := &response.ErrorResponse{
			Code:             response.ErrServerError.Code,
			ErrorDescription: "Failed to retrieve user profile",
		}
		response.RespondWithError(c, response.StatusInternalServerError, errorResp)
		return
	}

	// Return user profile
	response.RespondWithSuccess(c, http.StatusOK, profile)
}

// GinProfileUpdate handles PUT /profile - Update current user profile
func GinProfileUpdate(c *gin.Context) {
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

	// Parse request body
	var req ProfileUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		errorResp := &response.ErrorResponse{
			Code:             response.ErrInvalidRequest.Code,
			ErrorDescription: fmt.Sprintf("Invalid request body: %s", err.Error()),
		}
		response.RespondWithError(c, response.StatusBadRequest, errorResp)
		return
	}

	// Call business logic to update profile
	result, err := profileUpdate(c.Request.Context(), authInfo.UserID, req)
	if err != nil {
		log.Error("Failed to update user profile: %v", err)
		errorResp := &response.ErrorResponse{
			Code:             response.ErrServerError.Code,
			ErrorDescription: "Failed to update user profile",
		}
		response.RespondWithError(c, response.StatusInternalServerError, errorResp)
		return
	}

	// Return response with user_id and message
	response.RespondWithSuccess(c, http.StatusOK, result)
}

// Yao Process Handlers (for Yao application calls)

// ProcessProfileGet user.profile.get Profile get processor
// Args[0] (optional) map: {"team": true, "member": true, "type": true}
// Return: map: User profile data
func ProcessProfileGet(process *process.Process) interface{} {
	// Get user_id from session
	userIDStr := GetUserIDFromSession(process)

	// Get context
	ctx := process.Context
	if ctx == nil {
		ctx = context.Background()
	}

	// Get team_id from session if available
	teamID := ""
	if teamIDVal, err := session.Global().ID(process.Sid).Get("__team_id"); err == nil && teamIDVal != nil {
		if teamIDStr, ok := teamIDVal.(string); ok {
			teamID = teamIDStr
		}
	}

	// Parse options
	req := ProfileGetRequest{}
	if process.NumOfArgs() > 0 {
		opts := process.ArgsMap(0)
		if v, ok := opts["team"].(bool); ok {
			req.Team = v
		}
		if v, ok := opts["member"].(bool); ok {
			req.Member = v
		}
		if v, ok := opts["type"].(bool); ok {
			req.Type = v
		}
	}

	// Call business logic
	result, err := profileGet(ctx, userIDStr, teamID, req)
	if err != nil {
		exception.New("failed to get profile: %s", 500, err.Error()).Throw()
	}

	return result
}

// ProcessProfileUpdate user.profile.update Profile update processor
// Args[0] map: Profile update data (only profile fields allowed)
// Return: map: Updated user profile data
func ProcessProfileUpdate(process *process.Process) interface{} {
	// Get user_id from session
	userIDStr := GetUserIDFromSession(process)

	// Get context
	ctx := process.Context
	if ctx == nil {
		ctx = context.Background()
	}

	// Parse update data from first argument
	if process.NumOfArgs() == 0 {
		exception.New("profile update data is required", 400).Throw()
	}

	updateData := process.ArgsMap(0)
	req := buildProfileUpdateRequest(updateData)

	// Call business logic
	result, err := profileUpdate(ctx, userIDStr, req)
	if err != nil {
		exception.New("failed to update profile: %s", 500, err.Error()).Throw()
	}

	return result
}

// Private Business Logic Functions (internal use only)

// profileGet handles the business logic for getting user profile
func profileGet(ctx context.Context, userID, teamID string, req ProfileGetRequest) (maps.MapStrAny, error) {
	// Get user provider instance
	provider, err := getUserProvider()
	if err != nil {
		return nil, fmt.Errorf("failed to get user provider: %w", err)
	}

	// Get user data
	userData, err := provider.GetUser(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve user data: %w", err)
	}

	// Get Yao client config
	yaoClientConfig := GetYaoClientConfig()

	// Get or create subject
	subject, err := oauth.OAuth.Subject(yaoClientConfig.ClientID, userID)
	if err != nil {
		log.Warn("Failed to get user subject: %s", err.Error())
		subject = userID // Fallback to user ID
	}

	// Prepare OIDC user info (same format as login response)
	oidcUserInfo := oauthtypes.MakeOIDCUserInfo(userData)
	oidcUserInfo.Sub = subject
	oidcUserInfo.YaoUserID = userID

	// Add team and member info if requested and team_id is provided
	if teamID != "" && (req.Team || req.Member) {
		addTeamInfo(ctx, provider, oidcUserInfo, teamID, userID, req.Team, req.Member)
	}

	// Add type information if requested
	if req.Type {
		addTypeInfo(ctx, provider, oidcUserInfo, userData, teamID, userID)
	}

	// Convert to map for response (this will include all Yao fields)
	profileData := oidcUserInfo.Map()

	// Add member as separate object if requested (not part of OIDCUserInfo structure)
	if req.Member && teamID != "" {
		member, err := provider.GetMember(ctx, teamID, userID)
		if err == nil && member != nil {
			profileData["member"] = member
		}
	}

	return profileData, nil
}

// addTeamInfo adds team information to the profile
func addTeamInfo(ctx context.Context, provider *user.DefaultUser, oidcUserInfo *oauthtypes.OIDCUserInfo, teamID, userID string, withTeam, withMember bool) {
	// Only fetch team if either team or member info is requested
	if !withTeam && !withMember {
		return
	}

	// Get team details
	team, err := provider.GetTeamByMember(ctx, teamID, userID)
	if err != nil {
		log.Warn("Failed to get team: %v", err)
		return
	}

	// Add team info to OIDCUserInfo if requested
	if withTeam {
		oidcUserInfo.YaoTeamID = teamID
		oidcUserInfo.YaoTeam = &oauthtypes.OIDCTeamInfo{
			TeamID:      utils.ToString(team["team_id"]),
			Name:        utils.ToString(team["name"]),
			Description: utils.ToString(team["description"]),
			Logo:        utils.ToString(team["logo"]),
			OwnerID:     utils.ToString(team["owner_id"]),
		}

		// Check if user is owner
		if oidcUserInfo.YaoTeam.OwnerID == userID {
			isOwner := true
			oidcUserInfo.YaoIsOwner = &isOwner
		}

		// Add tenant_id if available
		if tenantID := utils.ToString(team["tenant_id"]); tenantID != "" {
			oidcUserInfo.YaoTenantID = tenantID
		}
	}
}

// addTypeInfo adds type information to the profile
func addTypeInfo(ctx context.Context, provider *user.DefaultUser, oidcUserInfo *oauthtypes.OIDCUserInfo, userData maps.MapStr, teamID, userID string) {
	var typeID string

	// Team context - try to get team's type first
	if teamID != "" {
		team, err := provider.GetTeamByMember(ctx, teamID, userID)
		if err == nil && team != nil {
			typeID = utils.ToString(team["type_id"])
		}
	}

	// Fallback to user's type
	if typeID == "" {
		typeID = utils.ToString(userData["type_id"])
	}

	if typeID == "" {
		return
	}

	oidcUserInfo.YaoTypeID = typeID

	// Get type details
	typeInfo, err := provider.GetType(ctx, typeID)
	if err != nil {
		log.Warn("Failed to get type: %v", err)
		return
	}

	oidcUserInfo.YaoType = &oauthtypes.OIDCTypeInfo{
		TypeID: utils.ToString(typeInfo["type_id"]),
		Name:   utils.ToString(typeInfo["name"]),
		Locale: utils.ToString(typeInfo["locale"]),
	}
}

// profileUpdate handles the business logic for updating user profile
func profileUpdate(ctx context.Context, userID string, req ProfileUpdateRequest) (ProfileUpdateResponse, error) {
	// Get user provider instance
	provider, err := getUserProvider()
	if err != nil {
		return ProfileUpdateResponse{}, fmt.Errorf("failed to get user provider: %w", err)
	}

	// Build update data map (only include non-nil fields)
	updateData := make(map[string]interface{})

	if req.Name != nil {
		updateData["name"] = *req.Name
	}
	if req.GivenName != nil {
		updateData["given_name"] = *req.GivenName
	}
	if req.FamilyName != nil {
		updateData["family_name"] = *req.FamilyName
	}
	if req.MiddleName != nil {
		updateData["middle_name"] = *req.MiddleName
	}
	if req.Nickname != nil {
		updateData["nickname"] = *req.Nickname
	}
	if req.Profile != nil {
		updateData["profile"] = *req.Profile
	}
	if req.Picture != nil {
		updateData["picture"] = *req.Picture
	}
	if req.Website != nil {
		updateData["website"] = *req.Website
	}
	if req.Gender != nil {
		updateData["gender"] = *req.Gender
	}
	if req.Birthdate != nil {
		updateData["birthdate"] = *req.Birthdate
	}
	if req.Zoneinfo != nil {
		updateData["zoneinfo"] = *req.Zoneinfo
	}
	if req.Locale != nil {
		updateData["locale"] = *req.Locale
	}
	if req.Address != nil {
		updateData["address"] = req.Address
	}
	if req.Theme != nil {
		updateData["theme"] = *req.Theme
	}
	if req.Metadata != nil {
		updateData["metadata"] = req.Metadata
	}

	// If no fields to update, return error
	if len(updateData) == 0 {
		return ProfileUpdateResponse{}, fmt.Errorf("no fields to update")
	}

	// Update user profile
	if err := provider.UpdateUser(ctx, userID, updateData); err != nil {
		return ProfileUpdateResponse{}, fmt.Errorf("failed to update user profile: %w", err)
	}

	// Return response with user_id and message
	return ProfileUpdateResponse{
		UserID:  userID,
		Message: "Profile updated successfully",
	}, nil
}

// buildProfileUpdateRequest converts a map to ProfileUpdateRequest
func buildProfileUpdateRequest(data map[string]interface{}) ProfileUpdateRequest {
	req := ProfileUpdateRequest{}

	if v, ok := data["name"].(string); ok {
		req.Name = &v
	}
	if v, ok := data["given_name"].(string); ok {
		req.GivenName = &v
	}
	if v, ok := data["family_name"].(string); ok {
		req.FamilyName = &v
	}
	if v, ok := data["middle_name"].(string); ok {
		req.MiddleName = &v
	}
	if v, ok := data["nickname"].(string); ok {
		req.Nickname = &v
	}
	if v, ok := data["profile"].(string); ok {
		req.Profile = &v
	}
	if v, ok := data["picture"].(string); ok {
		req.Picture = &v
	}
	if v, ok := data["website"].(string); ok {
		req.Website = &v
	}
	if v, ok := data["gender"].(string); ok {
		req.Gender = &v
	}
	if v, ok := data["birthdate"].(string); ok {
		req.Birthdate = &v
	}
	if v, ok := data["zoneinfo"].(string); ok {
		req.Zoneinfo = &v
	}
	if v, ok := data["locale"].(string); ok {
		req.Locale = &v
	}
	if v, ok := data["address"].(map[string]interface{}); ok {
		req.Address = v
	}
	if v, ok := data["theme"].(string); ok {
		req.Theme = &v
	}
	if v, ok := data["metadata"].(map[string]interface{}); ok {
		req.Metadata = v
	}

	return req
}
