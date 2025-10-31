package user

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/yaoapp/gou/model"
	"github.com/yaoapp/gou/process"
	"github.com/yaoapp/kun/exception"
	"github.com/yaoapp/kun/log"
	"github.com/yaoapp/kun/maps"
	"github.com/yaoapp/yao/openapi/oauth"
	"github.com/yaoapp/yao/openapi/oauth/authorized"
	"github.com/yaoapp/yao/openapi/oauth/providers/user"
	"github.com/yaoapp/yao/openapi/oauth/types"
	"github.com/yaoapp/yao/openapi/response"
	"github.com/yaoapp/yao/openapi/utils"
)

// Team Management Handlers

// GinTeamConfig handles GET /teams/config - Get team configuration (public)
func GinTeamConfig(c *gin.Context) {
	locale := c.Query("locale")
	if locale == "" {
		locale = "en" // default locale
	}

	// Clean locale: remove whitespace and special characters
	locale = strings.TrimSpace(locale)
	locale = strings.Trim(locale, "?&=")

	config := GetTeamConfigPublic(locale)
	if config == nil {
		errorResp := &response.ErrorResponse{
			Code:             response.ErrInvalidRequest.Code,
			ErrorDescription: "Team configuration not found",
		}
		response.RespondWithError(c, response.StatusNotFound, errorResp)
		return
	}

	response.RespondWithSuccess(c, http.StatusOK, config)
}

// GinTeamList handles GET /teams - Get user teams (all teams where user is a member)
func GinTeamList(c *gin.Context) {
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

	// Call business logic to get user teams with roles
	teams, err := getUserTeams(c.Request.Context(), authInfo.UserID)
	if err != nil {
		log.Error("Failed to get user teams: %v", err)
		errorResp := &response.ErrorResponse{
			Code:             response.ErrServerError.Code,
			ErrorDescription: "Failed to retrieve teams",
		}
		response.RespondWithError(c, response.StatusInternalServerError, errorResp)
		return
	}

	// Return teams list directly (no pagination)
	response.RespondWithSuccess(c, http.StatusOK, teams)
}

// GinTeamGet handles GET /teams/:id - Get user team details
func GinTeamGet(c *gin.Context) {
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

	teamID := c.Param("id")
	if teamID == "" {
		errorResp := &response.ErrorResponse{
			Code:             response.ErrInvalidRequest.Code,
			ErrorDescription: "Team ID is required",
		}
		response.RespondWithError(c, response.StatusBadRequest, errorResp)
		return
	}

	// Get user provider instance
	provider, err := getUserProvider()
	if err != nil {
		log.Error("Failed to get user provider: %v", err)
		errorResp := &response.ErrorResponse{
			Code:             response.ErrServerError.Code,
			ErrorDescription: "Failed to initialize user provider",
		}
		response.RespondWithError(c, response.StatusInternalServerError, errorResp)
		return
	}

	// Get team details
	teamData, err := provider.GetTeamDetail(c.Request.Context(), teamID)
	if err != nil {
		log.Error("Failed to get team details: %v", err)
		// Check if it's a "team not found" error
		if err.Error() == "team not found" {
			errorResp := &response.ErrorResponse{
				Code:             response.ErrInvalidRequest.Code,
				ErrorDescription: "Team not found",
			}
			response.RespondWithError(c, response.StatusNotFound, errorResp)
		} else {
			errorResp := &response.ErrorResponse{
				Code:             response.ErrServerError.Code,
				ErrorDescription: "Failed to retrieve team details",
			}
			response.RespondWithError(c, response.StatusInternalServerError, errorResp)
		}
		return
	}

	// // Check if user owns this team
	// ownerID := utils.ToString(teamData["owner_id"])
	// if ownerID != authInfo.UserID {
	// 	errorResp := &response.ErrorResponse{
	// 		Code:             response.ErrAccessDenied.Code,
	// 		ErrorDescription: "Access denied: you don't own this team",
	// 	}
	// 	response.RespondWithError(c, response.StatusForbidden, errorResp)
	// 	return
	// }

	// Convert to response format
	team := mapToTeamDetailResponse(teamData)
	response.RespondWithSuccess(c, http.StatusOK, team)
}

// GinTeamCreate handles POST /teams - Create user team
func GinTeamCreate(c *gin.Context) {
	// Get authorized user info
	authInfo := authorized.GetInfo(c)
	if authInfo.Constraints.OwnerOnly {
		if authInfo == nil || authInfo.UserID == "" {
			errorResp := &response.ErrorResponse{
				Code:             response.ErrInvalidClient.Code,
				ErrorDescription: "User not authenticated",
			}
			response.RespondWithError(c, response.StatusUnauthorized, errorResp)
			return
		}
	}

	// Parse request body
	var req CreateTeamRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		errorResp := &response.ErrorResponse{
			Code:             response.ErrInvalidRequest.Code,
			ErrorDescription: "Invalid request body: " + err.Error(),
		}
		response.RespondWithError(c, response.StatusBadRequest, errorResp)
		return
	}

	// Prepare team data
	teamData := authInfo.WithCreateScope(maps.MapStrAny{
		"name":        req.Name,
		"description": req.Description,
	})

	// Add logo if provided
	if req.Logo != "" {
		teamData["logo"] = req.Logo
	}

	// Add settings if provided
	if req.Settings != nil {
		teamData["settings"] = req.Settings
	}

	// Call business logic
	teamID, err := teamCreate(c.Request.Context(), authInfo.UserID, teamData)
	if err != nil {
		log.Error("Failed to create team: %v", err)
		errorResp := &response.ErrorResponse{
			Code:             response.ErrServerError.Code,
			ErrorDescription: "Failed to create team",
		}
		response.RespondWithError(c, response.StatusInternalServerError, errorResp)
		return
	}

	// Get the created team details
	provider, err := getUserProvider()
	if err != nil {
		log.Error("Failed to get user provider: %v", err)
		// Return basic response if we can't get details
		response.RespondWithSuccess(c, http.StatusCreated, gin.H{"team_id": teamID})
		return
	}

	createdTeam, err := provider.GetTeamDetail(c.Request.Context(), teamID)
	if err != nil {
		log.Error("Failed to get created team details: %v", err)
		// Return basic response if we can't get details
		response.RespondWithSuccess(c, http.StatusCreated, gin.H{"team_id": teamID})
		return
	}

	// Convert to response format
	team := mapToTeamDetailResponse(createdTeam)
	response.RespondWithSuccess(c, http.StatusCreated, team)
}

// GinTeamUpdate handles PUT /teams/:id - Update user team
func GinTeamUpdate(c *gin.Context) {
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

	teamID := c.Param("id")
	if teamID == "" {
		errorResp := &response.ErrorResponse{
			Code:             response.ErrInvalidRequest.Code,
			ErrorDescription: "Team ID is required",
		}
		response.RespondWithError(c, response.StatusBadRequest, errorResp)
		return
	}

	// Parse request body
	var req UpdateTeamRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		errorResp := &response.ErrorResponse{
			Code:             response.ErrInvalidRequest.Code,
			ErrorDescription: "Invalid request body: " + err.Error(),
		}
		response.RespondWithError(c, response.StatusBadRequest, errorResp)
		return
	}

	// Prepare update data
	updateData := maps.MapStrAny{}

	if req.Name != "" {
		updateData["name"] = req.Name
	}
	if req.Description != "" {
		updateData["description"] = req.Description
	}
	if req.Logo != "" {
		updateData["logo"] = req.Logo
	}
	if req.Settings != nil {
		updateData["settings"] = req.Settings
	}

	// Call business logic
	err := teamUpdate(c.Request.Context(), authInfo.UserID, teamID, updateData)
	if err != nil {
		log.Error("Failed to update team: %v", err)
		// Check error type for appropriate response
		if strings.Contains(err.Error(), "not found") {
			errorResp := &response.ErrorResponse{
				Code:             response.ErrInvalidRequest.Code,
				ErrorDescription: "Team not found",
			}
			response.RespondWithError(c, response.StatusNotFound, errorResp)
		} else if strings.Contains(err.Error(), "access denied") {
			errorResp := &response.ErrorResponse{
				Code:             response.ErrAccessDenied.Code,
				ErrorDescription: err.Error(),
			}
			response.RespondWithError(c, response.StatusForbidden, errorResp)
		} else {
			errorResp := &response.ErrorResponse{
				Code:             response.ErrServerError.Code,
				ErrorDescription: "Failed to update team",
			}
			response.RespondWithError(c, response.StatusInternalServerError, errorResp)
		}
		return
	}

	// Get updated team details
	provider, err := getUserProvider()
	if err != nil {
		log.Error("Failed to get user provider: %v", err)
		response.RespondWithSuccess(c, http.StatusOK, gin.H{"message": "Team updated successfully"})
		return
	}

	updatedTeam, err := provider.GetTeamDetail(c.Request.Context(), teamID)
	if err != nil {
		log.Error("Failed to get updated team details: %v", err)
		response.RespondWithSuccess(c, http.StatusOK, gin.H{"message": "Team updated successfully"})
		return
	}

	// Convert to response format
	team := mapToTeamDetailResponse(updatedTeam)
	response.RespondWithSuccess(c, http.StatusOK, team)
}

// GinTeamCurrent handles GET /teams/current - Get current team
func GinTeamCurrent(c *gin.Context) {
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

	// Get current team ID (from token or first owner team)
	teamID, err := getCurrentTeamID(c.Request.Context(), authInfo.TeamID, authInfo.UserID)
	if err != nil {
		log.Error("Failed to get current team ID: %v", err)

		// Return 404 if user has no team
		if err.Error() == "no owner team found for user" {
			errorResp := &response.ErrorResponse{
				Code:             response.ErrInvalidRequest.Code,
				ErrorDescription: "No owner team found for user",
			}
			response.RespondWithError(c, response.StatusNotFound, errorResp)
			return
		}

		// Return 500 for other errors
		errorResp := &response.ErrorResponse{
			Code:             response.ErrServerError.Code,
			ErrorDescription: "Failed to get current team ID",
		}
		response.RespondWithError(c, response.StatusInternalServerError, errorResp)
		return
	}

	// Get team details
	team, err := teamGet(c.Request.Context(), authInfo.UserID, teamID)
	if err != nil {
		log.Error("Failed to get team details: %v", err)
		errorResp := &response.ErrorResponse{
			Code:             response.ErrServerError.Code,
			ErrorDescription: "Failed to get team details",
		}
		response.RespondWithError(c, response.StatusInternalServerError, errorResp)
		return
	}

	response.RespondWithSuccess(c, http.StatusOK, team)
}

// GinTeamSelection handles POST /teams/select - Select a team and issue tokens with team_id
func GinTeamSelection(c *gin.Context) {
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

	// Verify the current token has team_selection scope
	if authInfo.Scope != ScopeTeamSelection {
		errorResp := &response.ErrorResponse{
			Code:             response.ErrAccessDenied.Code,
			ErrorDescription: "Invalid scope: team_selection scope required",
		}
		response.RespondWithError(c, response.StatusForbidden, errorResp)
		return
	}

	// Parse request body
	var req TeamSelectionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		errorResp := &response.ErrorResponse{
			Code:             response.ErrInvalidRequest.Code,
			ErrorDescription: "Invalid request body: " + err.Error(),
		}
		response.RespondWithError(c, response.StatusBadRequest, errorResp)
		return
	}

	ctx := c.Request.Context()

	// Prepare login context with full device/platform information
	loginCtx := makeLoginContext(c)

	// Preserve Remember Me state from temporary token
	loginCtx.RememberMe = authInfo.RememberMe

	// Login with selected team
	loginResponse, err := LoginByTeamID(authInfo.UserID, req.TeamID, loginCtx)
	if err != nil {
		log.Error("Failed to login with team: %v", err)
		errorResp := &response.ErrorResponse{
			Code:             response.ErrServerError.Code,
			ErrorDescription: "Failed to login with team: " + err.Error(),
		}
		response.RespondWithError(c, response.StatusInternalServerError, errorResp)
		return
	}

	// Revoke the current temporary token (read from header)
	currentToken := oauth.OAuth.GetAccessToken(c)
	if currentToken != "" {
		if err := oauth.OAuth.Revoke(ctx, currentToken, "access_token"); err != nil {
			// Log the error but don't fail the request
			log.Warn("Failed to revoke temporary token: %v", err)
		}
	}

	// Send secure cookies (access token, refresh token, and session ID)
	SendLoginCookies(c, loginResponse, "")

	// Return the new tokens in response body
	response.RespondWithSuccess(c, http.StatusOK, loginResponse)
}

// GinTeamDelete handles DELETE /teams/:id - Delete user team
func GinTeamDelete(c *gin.Context) {
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

	teamID := c.Param("id")
	if teamID == "" {
		errorResp := &response.ErrorResponse{
			Code:             response.ErrInvalidRequest.Code,
			ErrorDescription: "Team ID is required",
		}
		response.RespondWithError(c, response.StatusBadRequest, errorResp)
		return
	}

	// Call business logic
	err := teamDelete(c.Request.Context(), authInfo.UserID, teamID)
	if err != nil {
		log.Error("Failed to delete team: %v", err)
		// Check error type for appropriate response
		if strings.Contains(err.Error(), "not found") {
			errorResp := &response.ErrorResponse{
				Code:             response.ErrInvalidRequest.Code,
				ErrorDescription: "Team not found",
			}
			response.RespondWithError(c, response.StatusNotFound, errorResp)
		} else if strings.Contains(err.Error(), "access denied") {
			errorResp := &response.ErrorResponse{
				Code:             response.ErrAccessDenied.Code,
				ErrorDescription: err.Error(),
			}
			response.RespondWithError(c, response.StatusForbidden, errorResp)
		} else {
			errorResp := &response.ErrorResponse{
				Code:             response.ErrServerError.Code,
				ErrorDescription: "Failed to delete team",
			}
			response.RespondWithError(c, response.StatusInternalServerError, errorResp)
		}
		return
	}

	response.RespondWithSuccess(c, http.StatusOK, gin.H{"message": "Team deleted successfully"})
}

// Yao Process Handlers (for Yao application calls)

// ProcessTeamList user.team.list Team list processor
// Args[0] map: Query parameters {"status": "active", "name": "search", "page": 1, "pagesize": 20}
// Return: map: Paginated team list
func ProcessTeamList(process *process.Process) interface{} {
	process.ValidateArgNums(1)

	// Get user_id from session
	userIDStr := GetUserIDFromSession(process)

	// Parse query parameters
	queryMap := process.ArgsMap(0)

	// Build query parameters
	param := model.QueryParam{}

	// Add filters
	if status, ok := queryMap["status"].(string); ok && status != "" {
		param.Wheres = append(param.Wheres, model.QueryWhere{
			Column: "status",
			Value:  status,
		})
	}

	if name, ok := queryMap["name"].(string); ok && name != "" {
		param.Wheres = append(param.Wheres, model.QueryWhere{
			Column: "name",
			Value:  "%" + name + "%",
			OP:     "like",
		})
	}

	// Parse pagination
	page := 1
	pagesize := 20

	if p, ok := queryMap["page"]; ok {
		if pageInt, ok := p.(int); ok && pageInt > 0 {
			page = pageInt
		}
	}

	if ps, ok := queryMap["pagesize"]; ok {
		if pagesizeInt, ok := ps.(int); ok && pagesizeInt > 0 && pagesizeInt <= 100 {
			pagesize = pagesizeInt
		}
	}

	// Get context
	ctx := process.Context
	if ctx == nil {
		ctx = context.Background()
	}

	// Call business logic
	result, err := teamList(ctx, userIDStr, param, page, pagesize)
	if err != nil {
		exception.New("failed to list teams: %s", 500, err.Error()).Throw()
	}

	return result
}

// ProcessTeamGet user.team.get Team get processor
// Args[0] string: team_id
// Return: map: Team details
func ProcessTeamGet(process *process.Process) interface{} {
	process.ValidateArgNums(1)

	// Get user_id from session
	userIDStr := GetUserIDFromSession(process)

	teamID := process.ArgsString(0)
	if teamID == "" {
		exception.New("team_id is required", 400).Throw()
	}

	// Get context
	ctx := process.Context
	if ctx == nil {
		ctx = context.Background()
	}

	// Call business logic
	result, err := teamGet(ctx, userIDStr, teamID)
	if err != nil {
		exception.New("failed to get team: %s", 500, err.Error()).Throw()
	}

	return result
}

// ProcessTeamCreate user.team.create Team create processor
// Args[0] map: Team data {"name": "Team Name", "description": "Description", "settings": {...}}
// Return: map: {"team_id": "created_team_id"}
func ProcessTeamCreate(process *process.Process) interface{} {
	process.ValidateArgNums(1)

	// Get user_id from session
	userIDStr := GetUserIDFromSession(process)

	teamData := maps.MapStrAny(process.ArgsMap(0))

	// Validate required fields
	if _, ok := teamData["name"]; !ok {
		exception.New("name is required", 400).Throw()
	}

	// Get context
	ctx := process.Context
	if ctx == nil {
		ctx = context.Background()
	}

	// Call business logic
	teamID, err := teamCreate(ctx, userIDStr, teamData)
	if err != nil {
		exception.New("failed to create team: %s", 500, err.Error()).Throw()
	}

	return map[string]interface{}{
		"team_id": teamID,
	}
}

// ProcessTeamUpdate user.team.update Team update processor
// Args[0] string: team_id
// Args[1] map: Update data {"name": "New Name", "description": "New Description", "settings": {...}}
// Return: map: {"message": "success"}
func ProcessTeamUpdate(process *process.Process) interface{} {
	process.ValidateArgNums(2)

	// Get user_id from session
	userIDStr := GetUserIDFromSession(process)

	teamID := process.ArgsString(0)
	updateData := maps.MapStrAny(process.ArgsMap(1))

	if teamID == "" {
		exception.New("team_id is required", 400).Throw()
	}

	// Get context
	ctx := process.Context
	if ctx == nil {
		ctx = context.Background()
	}

	// Call business logic
	err := teamUpdate(ctx, userIDStr, teamID, updateData)
	if err != nil {
		exception.New("failed to update team: %s", 500, err.Error()).Throw()
	}

	return map[string]interface{}{
		"message": "success",
	}
}

// ProcessTeamDelete user.team.delete Team delete processor
// Args[0] string: team_id
// Return: map: {"message": "success"}
func ProcessTeamDelete(process *process.Process) interface{} {
	process.ValidateArgNums(1)

	// Get user_id from session
	userIDStr := GetUserIDFromSession(process)

	teamID := process.ArgsString(0)
	if teamID == "" {
		exception.New("team_id is required", 400).Throw()
	}

	// Get context
	ctx := process.Context
	if ctx == nil {
		ctx = context.Background()
	}

	// Call business logic
	err := teamDelete(ctx, userIDStr, teamID)
	if err != nil {
		exception.New("failed to delete team: %s", 500, err.Error()).Throw()
	}

	return map[string]interface{}{
		"message": "success",
	}
}

// Private Business Logic Functions (internal use only)

// teamList handles the business logic for listing user teams
func teamList(ctx context.Context, userID string, param model.QueryParam, page, pagesize int) (maps.MapStr, error) {
	// Get user provider instance
	provider, err := getUserProvider()
	if err != nil {
		return nil, fmt.Errorf("failed to get user provider: %w", err)
	}

	// Add owner filter to query parameters
	param.Wheres = append(param.Wheres, model.QueryWhere{
		Column: "owner_id",
		Value:  userID,
	})

	// Set default ordering if not provided
	if len(param.Orders) == 0 {
		param.Orders = []model.QueryOrder{
			{Column: "created_at", Option: "desc"},
		}
	}

	// Get paginated teams
	result, err := provider.PaginateTeams(ctx, param, page, pagesize)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve teams: %w", err)
	}

	return result, nil
}

// getCurrentTeamID resolves the current team ID for a user
// If teamID is provided, it returns it directly
// If teamID is empty, it gets the first owner team for the user
func getCurrentTeamID(ctx context.Context, teamID, userID string) (string, error) {
	// If team ID is already provided, return it
	if teamID != "" {
		return teamID, nil
	}

	// Get owner teams for the user
	teams, err := getOwnerTeams(ctx, userID)
	if err != nil {
		return "", fmt.Errorf("failed to get owner teams: %w", err)
	}

	// Check if user has any owner teams
	if len(teams) == 0 {
		return "", fmt.Errorf("no owner team found for user")
	}

	// Return the first team ID
	if teamIDVal, ok := teams[0]["team_id"].(string); ok {
		return teamIDVal, nil
	}

	return "", fmt.Errorf("invalid team_id format")
}

// teamGet handles the business logic for getting a specific user team
func teamGet(ctx context.Context, userID, teamID string) (maps.MapStrAny, error) {
	// Get user provider instance
	provider, err := getUserProvider()
	if err != nil {
		return nil, fmt.Errorf("failed to get user provider: %w", err)
	}

	// Get team details
	teamData, err := provider.GetTeamDetail(ctx, teamID)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve team details: %w", err)
	}

	// Validate if user is a member of the team
	exists, err := provider.MemberExists(ctx, teamID, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve team member record: %w", err)
	}

	if !exists {
		return nil, fmt.Errorf("user is not a member of this team")
	}

	return teamData, nil
}

// teamCreate handles the business logic for creating a user team
func teamCreate(ctx context.Context, userID string, teamData maps.MapStrAny) (string, error) {
	// Get user provider instance
	provider, err := getUserProvider()
	if err != nil {
		return "", fmt.Errorf("failed to get user provider: %w", err)
	}

	// Set owner and default values
	teamData["owner_id"] = userID
	teamData["status"] = "active"
	teamData["is_verified"] = false
	teamData["created_at"] = time.Now()
	teamData["updated_at"] = time.Now()

	// Get team config for setting defaults
	locale := ""
	if localeVal, ok := teamData["locale"].(string); ok && localeVal != "" {
		locale = strings.TrimSpace(strings.ToLower(localeVal))
	}

	// Fallback: try common locale variations or use "en" as final fallback
	// This ensures we always get a valid config even if locale is invalid
	teamConfig := GetTeamConfig(locale)
	if teamConfig == nil {
		// Try fallback locales in order
		fallbackLocales := []string{"en", "zh-cn"}
		for _, fallback := range fallbackLocales {
			teamConfig = GetTeamConfig(fallback)
			if teamConfig != nil {
				break
			}
		}
	}

	// Set default type_id from team config if not provided
	if _, hasType := teamData["type_id"]; !hasType {
		// Apply default type from config if available
		if teamConfig != nil && teamConfig.Type != "" {
			teamData["type_id"] = teamConfig.Type
		}
	}

	// Set default role_id from team config if not provided
	if _, hasRole := teamData["role_id"]; !hasRole {
		// Apply default role from config if available
		if teamConfig != nil && teamConfig.Role != "" {
			teamData["role_id"] = teamConfig.Role
		}
	}

	// Clean up: remove locale from team data as it's not stored in database
	delete(teamData, "locale")

	// Create team
	teamID, err := provider.CreateTeam(ctx, teamData)
	if err != nil {
		return "", fmt.Errorf("failed to create team: %w", err)
	}

	// Determine owner member role_id from team config
	ownerRoleID := "owner" // fallback default
	if teamConfig != nil && teamConfig.Role != "" {
		ownerRoleID = teamConfig.Role
	}

	// Add the creator as an owner member of the team
	ownerMemberData := types.CopyCreateScope(teamData, maps.MapStrAny{
		"team_id":     teamID,
		"user_id":     userID,
		"member_type": "user",
		"role_id":     ownerRoleID,
		"is_owner":    true,
		"status":      "active",
		"joined_at":   time.Now(),
		"created_at":  time.Now(),
		"updated_at":  time.Now(),
	})

	_, err = provider.CreateMember(ctx, ownerMemberData)
	if err != nil {
		// Log the error but don't fail the team creation
		log.Error("Failed to add owner as team member: %v", err)
		// Consider whether to rollback team creation or continue
		// For now, we'll continue as the team is already created
	}

	return teamID, nil
}

// teamUpdate handles the business logic for updating a user team
func teamUpdate(ctx context.Context, userID, teamID string, updateData maps.MapStrAny) error {
	// Get user provider instance
	provider, err := getUserProvider()
	if err != nil {
		return fmt.Errorf("failed to get user provider: %w", err)
	}

	// Check if team exists and user owns it
	teamData, err := provider.GetTeam(ctx, teamID)
	if err != nil {
		return fmt.Errorf("team not found or access denied: %w", err)
	}

	// Check ownership
	ownerID := utils.ToString(teamData["owner_id"])
	if ownerID != userID {
		return fmt.Errorf("access denied: user does not own this team")
	}

	// Add updated_at timestamp
	updateData["updated_at"] = time.Now()

	// Update team
	err = provider.UpdateTeam(ctx, teamID, updateData)
	if err != nil {
		return fmt.Errorf("failed to update team: %w", err)
	}

	return nil
}

// teamDelete handles the business logic for deleting a user team
func teamDelete(ctx context.Context, userID, teamID string) error {
	// Get user provider instance
	provider, err := getUserProvider()
	if err != nil {
		return fmt.Errorf("failed to get user provider: %w", err)
	}

	// Check if team exists and user owns it
	teamData, err := provider.GetTeam(ctx, teamID)
	if err != nil {
		return fmt.Errorf("team not found or access denied: %w", err)
	}

	// Check ownership
	ownerID := utils.ToString(teamData["owner_id"])
	if ownerID != userID {
		return fmt.Errorf("access denied: user does not own this team")
	}

	// First, remove all team members
	err = provider.RemoveAllTeamMembers(ctx, teamID)
	if err != nil {
		// Log error but don't fail team deletion - members might not exist
		log.Error("Failed to remove team members during team deletion: %v", err)
	}

	// Then delete the team
	err = provider.DeleteTeam(ctx, teamID)
	if err != nil {
		return fmt.Errorf("failed to delete team: %w", err)
	}

	return nil
}

// Private Helper Functions (internal use only)

// getUserProvider gets the user provider from the global OAuth service
func getUserProvider() (*user.DefaultUser, error) {
	// Check if global OAuth service is initialized
	if oauth.OAuth == nil {
		return nil, fmt.Errorf("OAuth service not initialized")
	}

	// Get user provider from OAuth service
	userProvider, err := oauth.OAuth.GetUserProvider()
	if err != nil {
		return nil, fmt.Errorf("failed to get user provider: %w", err)
	}

	// Type assert to DefaultUser (this should be safe based on the OAuth service implementation)
	if defaultUser, ok := userProvider.(*user.DefaultUser); ok {
		return defaultUser, nil
	}

	return nil, fmt.Errorf("user provider is not of type DefaultUser")
}

// mapToTeamResponse converts a map to TeamResponse
func mapToTeamResponse(data maps.MapStr) TeamResponse {
	team := TeamResponse{
		ID:          utils.ToInt64(data["id"]),
		TeamID:      utils.ToString(data["team_id"]),
		Name:        utils.ToString(data["name"]),
		Description: utils.ToString(data["description"]),
		Logo:        utils.ToString(data["logo"]),
		OwnerID:     utils.ToString(data["owner_id"]),
		Status:      utils.ToString(data["status"]),
		IsVerified:  utils.ToBool(data["is_verified"]),
		VerifiedBy:  utils.ToString(data["verified_by"]),
		VerifiedAt:  utils.ToTimeString(data["verified_at"]),
		CreatedAt:   utils.ToTimeString(data["created_at"]),
		UpdatedAt:   utils.ToTimeString(data["updated_at"]),
	}

	return team
}

// mapToTeamDetailResponse converts a map to TeamDetailResponse
func mapToTeamDetailResponse(data maps.MapStr) TeamDetailResponse {
	team := TeamDetailResponse{
		TeamResponse: mapToTeamResponse(data),
	}

	// Add settings if available
	if settings, ok := data["settings"]; ok {
		if teamSettings, ok := settings.(*TeamSettings); ok {
			team.Settings = teamSettings
		} else if settingsMap, ok := settings.(map[string]interface{}); ok {
			// Convert map to TeamSettings (for backward compatibility)
			teamSettings := &TeamSettings{
				Theme:      utils.ToString(settingsMap["theme"]),
				Visibility: utils.ToString(settingsMap["visibility"]),
			}
			team.Settings = teamSettings
		}
	}

	return team
}

// Business Logic Functions for Team Membership

// getUserTeams gets all teams where the user is a member (includes role information)
func getUserTeams(ctx context.Context, userID string) ([]maps.MapStr, error) {
	// Get user provider instance
	provider, err := getUserProvider()
	if err != nil {
		return nil, fmt.Errorf("failed to get user provider: %w", err)
	}

	// Get teams with role information
	teams, err := provider.GetTeamsByMember(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve user teams: %w", err)
	}

	return teams, nil
}

// getOwnerTeams gets all teams where the user is the owner
func getOwnerTeams(ctx context.Context, userID string) ([]maps.MapStr, error) {
	// Get user provider instance
	provider, err := getUserProvider()
	if err != nil {
		return nil, fmt.Errorf("failed to get user provider: %w", err)
	}

	// Get owner team
	teams, err := provider.GetTeamsByOwner(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve owner team: %w", err)
	}

	// Return the first team as the owner team
	return teams, nil
}

// getUserTeamsCount counts the number of teams a user is a member of
func getUserTeamsCount(ctx context.Context, userID string) (int64, error) {
	// Get user provider instance
	provider, err := getUserProvider()
	if err != nil {
		return 0, fmt.Errorf("failed to get user provider: %w", err)
	}

	// Count teams
	count, err := provider.CountTeamsByMember(ctx, userID)
	if err != nil {
		return 0, fmt.Errorf("failed to count user teams: %w", err)
	}

	return count, nil
}
