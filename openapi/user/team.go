package user

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/yaoapp/gou/model"
	"github.com/yaoapp/gou/process"
	"github.com/yaoapp/kun/exception"
	"github.com/yaoapp/kun/log"
	"github.com/yaoapp/kun/maps"
	"github.com/yaoapp/yao/openapi/oauth"
	"github.com/yaoapp/yao/openapi/oauth/providers/user"
	"github.com/yaoapp/yao/openapi/response"
)

// Team Management Handlers

// GinTeamList handles GET /teams - Get user teams
func GinTeamList(c *gin.Context) {
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

	// Parse pagination parameters
	page := 1
	pagesize := 20

	if p := c.Query("page"); p != "" {
		if parsed, err := strconv.Atoi(p); err == nil && parsed > 0 {
			page = parsed
		}
	}

	if ps := c.Query("pagesize"); ps != "" {
		if parsed, err := strconv.Atoi(ps); err == nil && parsed > 0 && parsed <= 100 {
			pagesize = parsed
		}
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

	// Build query parameters
	param := model.QueryParam{
		Wheres: []model.QueryWhere{
			{Column: "owner_id", Value: authInfo.UserID},
		},
		Orders: []model.QueryOrder{
			{Column: "created_at", Option: "desc"},
		},
	}

	// Add status filter if provided
	if status := c.Query("status"); status != "" {
		param.Wheres = append(param.Wheres, model.QueryWhere{
			Column: "status",
			Value:  status,
		})
	}

	// Add name search if provided
	if name := c.Query("name"); name != "" {
		param.Wheres = append(param.Wheres, model.QueryWhere{
			Column: "name",
			Value:  "%" + name + "%",
			OP:     "like",
		})
	}

	// Get paginated teams
	result, err := provider.PaginateTeams(c.Request.Context(), param, page, pagesize)
	if err != nil {
		log.Error("Failed to get user teams: %v", err)
		errorResp := &response.ErrorResponse{
			Code:             response.ErrServerError.Code,
			ErrorDescription: "Failed to retrieve teams",
		}
		response.RespondWithError(c, response.StatusInternalServerError, errorResp)
		return
	}

	// Return the paginated result directly (consistent with other modules)
	c.JSON(http.StatusOK, result)
}

// GinTeamGet handles GET /teams/:team_id - Get user team details
func GinTeamGet(c *gin.Context) {
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

	teamID := c.Param("team_id")
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

	// Check if user owns this team
	ownerID := toString(teamData["owner_id"])
	if ownerID != authInfo.UserID {
		errorResp := &response.ErrorResponse{
			Code:             response.ErrAccessDenied.Code,
			ErrorDescription: "Access denied: you don't own this team",
		}
		response.RespondWithError(c, response.StatusForbidden, errorResp)
		return
	}

	// Convert to response format
	team := mapToTeamDetailResponse(teamData)
	c.JSON(http.StatusOK, team)
}

// GinTeamCreate handles POST /teams - Create user team
func GinTeamCreate(c *gin.Context) {
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
	teamData := maps.MapStrAny{
		"name":        req.Name,
		"description": req.Description,
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
		c.JSON(http.StatusCreated, gin.H{"team_id": teamID})
		return
	}

	createdTeam, err := provider.GetTeamDetail(c.Request.Context(), teamID)
	if err != nil {
		log.Error("Failed to get created team details: %v", err)
		// Return basic response if we can't get details
		c.JSON(http.StatusCreated, gin.H{"team_id": teamID})
		return
	}

	// Convert to response format
	team := mapToTeamDetailResponse(createdTeam)
	c.JSON(http.StatusCreated, team)
}

// GinTeamUpdate handles PUT /teams/:team_id - Update user team
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

	teamID := c.Param("team_id")
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
		c.JSON(http.StatusOK, gin.H{"message": "Team updated successfully"})
		return
	}

	updatedTeam, err := provider.GetTeamDetail(c.Request.Context(), teamID)
	if err != nil {
		log.Error("Failed to get updated team details: %v", err)
		c.JSON(http.StatusOK, gin.H{"message": "Team updated successfully"})
		return
	}

	// Convert to response format
	team := mapToTeamDetailResponse(updatedTeam)
	c.JSON(http.StatusOK, team)
}

// GinTeamDelete handles DELETE /teams/:team_id - Delete user team
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

	teamID := c.Param("team_id")
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

	c.JSON(http.StatusOK, gin.H{"message": "Team deleted successfully"})
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

	// Check if user owns this team
	ownerID := toString(teamData["owner_id"])
	if ownerID != userID {
		return nil, fmt.Errorf("access denied: user does not own this team")
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

	// Create team
	teamID, err := provider.CreateTeam(ctx, teamData)
	if err != nil {
		return "", fmt.Errorf("failed to create team: %w", err)
	}

	// Add the creator as an owner member of the team
	ownerMemberData := maps.MapStrAny{
		"team_id":     teamID,
		"user_id":     userID,
		"member_type": "user",
		"role_id":     "owner",
		"status":      "active",
		"joined_at":   time.Now(),
		"created_at":  time.Now(),
		"updated_at":  time.Now(),
	}

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
	ownerID := toString(teamData["owner_id"])
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
	ownerID := toString(teamData["owner_id"])
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
		ID:          toInt64(data["id"]),
		TeamID:      toString(data["team_id"]),
		Name:        toString(data["name"]),
		Description: toString(data["description"]),
		OwnerID:     toString(data["owner_id"]),
		Status:      toString(data["status"]),
		IsVerified:  toBool(data["is_verified"]),
		VerifiedBy:  toString(data["verified_by"]),
		VerifiedAt:  toTimeString(data["verified_at"]),
		CreatedAt:   toTimeString(data["created_at"]),
		UpdatedAt:   toTimeString(data["updated_at"]),
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
		if settingsMap, ok := settings.(map[string]interface{}); ok {
			team.Settings = settingsMap
		}
	}

	return team
}
