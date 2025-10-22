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
	"github.com/yaoapp/yao/openapi/oauth/authorized"
	"github.com/yaoapp/yao/openapi/response"
)

// Member Management Handlers

// GinMemberList handles GET /teams/:team_id/members - Get team members
func GinMemberList(c *gin.Context) {

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

	// Call business logic
	result, err := memberList(c.Request.Context(), authInfo.UserID, teamID, page, pagesize, c.Query("status"))
	if err != nil {
		log.Error("Failed to get team members: %v", err)
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
				ErrorDescription: "Failed to retrieve team members",
			}
			response.RespondWithError(c, response.StatusInternalServerError, errorResp)
		}
		return
	}

	// Return the paginated result
	response.RespondWithSuccess(c, http.StatusOK, result)
}

// GinMemberGet handles GET /teams/:team_id/members/:member_id - Get team member details
func GinMemberGet(c *gin.Context) {
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
	memberID := c.Param("member_id")
	if teamID == "" || memberID == "" {
		errorResp := &response.ErrorResponse{
			Code:             response.ErrInvalidRequest.Code,
			ErrorDescription: "Team ID and Member ID are required",
		}
		response.RespondWithError(c, response.StatusBadRequest, errorResp)
		return
	}

	// Call business logic
	memberData, err := memberGet(c.Request.Context(), authInfo.UserID, teamID, memberID)
	if err != nil {
		log.Error("Failed to get member details: %v", err)
		// Check error type for appropriate response
		if strings.Contains(err.Error(), "not found") {
			errorResp := &response.ErrorResponse{
				Code:             response.ErrInvalidRequest.Code,
				ErrorDescription: "Member not found",
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
				ErrorDescription: "Failed to retrieve member details",
			}
			response.RespondWithError(c, response.StatusInternalServerError, errorResp)
		}
		return
	}

	// Convert to response format
	member := mapToMemberDetailResponse(memberData)
	response.RespondWithSuccess(c, http.StatusOK, member)
}

// GinMemberCreateDirect handles POST /teams/:team_id/members - Add member directly to team
func GinMemberCreateDirect(c *gin.Context) {
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
	var req CreateMemberRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		errorResp := &response.ErrorResponse{
			Code:             response.ErrInvalidRequest.Code,
			ErrorDescription: "Invalid request body: " + err.Error(),
		}
		response.RespondWithError(c, response.StatusBadRequest, errorResp)
		return
	}

	// Prepare member data
	memberData := maps.MapStrAny{
		"user_id":     req.UserID,
		"member_type": req.MemberType,
		"role_id":     req.RoleID,
	}

	// Add settings if provided
	if req.Settings != nil {
		memberData["settings"] = req.Settings
	}

	// Call business logic
	memberID, err := memberCreateDirect(c.Request.Context(), authInfo.UserID, teamID, memberData)
	if err != nil {
		log.Error("Failed to create member: %v", err)
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
		} else if strings.Contains(err.Error(), "already exists") {
			errorResp := &response.ErrorResponse{
				Code:             response.ErrInvalidRequest.Code,
				ErrorDescription: err.Error(),
			}
			response.RespondWithError(c, response.StatusConflict, errorResp)
		} else {
			errorResp := &response.ErrorResponse{
				Code:             response.ErrServerError.Code,
				ErrorDescription: "Failed to create member",
			}
			response.RespondWithError(c, response.StatusInternalServerError, errorResp)
		}
		return
	}

	// Return created member ID
	response.RespondWithSuccess(c, http.StatusCreated, gin.H{"member_id": memberID})
}

// GinMemberUpdate handles PUT /teams/:team_id/members/:member_id - Update team member
func GinMemberUpdate(c *gin.Context) {
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
	memberID := c.Param("member_id")
	if teamID == "" || memberID == "" {
		errorResp := &response.ErrorResponse{
			Code:             response.ErrInvalidRequest.Code,
			ErrorDescription: "Team ID and Member ID are required",
		}
		response.RespondWithError(c, response.StatusBadRequest, errorResp)
		return
	}

	// Parse request body
	var req UpdateMemberRequest
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

	if req.RoleID != "" {
		updateData["role_id"] = req.RoleID
	}
	if req.Status != "" {
		updateData["status"] = req.Status
	}
	if req.Settings != nil {
		updateData["settings"] = req.Settings
	}
	if req.LastActivity != "" {
		updateData["last_activity"] = req.LastActivity
	}

	// Call business logic
	err := memberUpdate(c.Request.Context(), authInfo.UserID, teamID, memberID, updateData)
	if err != nil {
		log.Error("Failed to update member: %v", err)
		// Check error type for appropriate response
		if strings.Contains(err.Error(), "not found") {
			errorResp := &response.ErrorResponse{
				Code:             response.ErrInvalidRequest.Code,
				ErrorDescription: "Member not found",
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
				ErrorDescription: "Failed to update member",
			}
			response.RespondWithError(c, response.StatusInternalServerError, errorResp)
		}
		return
	}

	response.RespondWithSuccess(c, http.StatusOK, gin.H{"message": "Member updated successfully"})
}

// GinMemberDelete handles DELETE /teams/:team_id/members/:member_id - Remove team member
func GinMemberDelete(c *gin.Context) {
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
	memberID := c.Param("member_id")
	if teamID == "" || memberID == "" {
		errorResp := &response.ErrorResponse{
			Code:             response.ErrInvalidRequest.Code,
			ErrorDescription: "Team ID and Member ID are required",
		}
		response.RespondWithError(c, response.StatusBadRequest, errorResp)
		return
	}

	// Call business logic
	err := memberDelete(c.Request.Context(), authInfo.UserID, teamID, memberID)
	if err != nil {
		log.Error("Failed to delete member: %v", err)
		// Check error type for appropriate response
		if strings.Contains(err.Error(), "not found") {
			errorResp := &response.ErrorResponse{
				Code:             response.ErrInvalidRequest.Code,
				ErrorDescription: "Member not found",
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
				ErrorDescription: "Failed to delete member",
			}
			response.RespondWithError(c, response.StatusInternalServerError, errorResp)
		}
		return
	}

	response.RespondWithSuccess(c, http.StatusOK, gin.H{"message": "Member removed successfully"})
}

// Yao Process Handlers (for Yao application calls)

// ProcessMemberList user.member.list Member list processor
// Args[0] string: team_id
// Args[1] map: Query parameters {"status": "active", "page": 1, "pagesize": 20}
// Return: map: Paginated member list
func ProcessMemberList(process *process.Process) interface{} {
	process.ValidateArgNums(2)

	// Get user_id from session
	userIDStr := GetUserIDFromSession(process)

	teamID := process.ArgsString(0)
	if teamID == "" {
		exception.New("team_id is required", 400).Throw()
	}

	// Parse query parameters
	queryMap := process.ArgsMap(1)

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

	// Get status filter
	status := ""
	if s, ok := queryMap["status"].(string); ok {
		status = s
	}

	// Get context
	ctx := process.Context
	if ctx == nil {
		ctx = context.Background()
	}

	// Call business logic
	result, err := memberList(ctx, userIDStr, teamID, page, pagesize, status)
	if err != nil {
		exception.New("failed to list members: %s", 500, err.Error()).Throw()
	}

	return result
}

// ProcessMemberGet user.member.get Member get processor
// Args[0] string: team_id
// Args[1] string: member_id
// Return: map: Member details
func ProcessMemberGet(process *process.Process) interface{} {
	process.ValidateArgNums(2)

	// Get user_id from session
	userIDStr := GetUserIDFromSession(process)

	teamID := process.ArgsString(0)
	memberID := process.ArgsString(1)

	if teamID == "" || memberID == "" {
		exception.New("team_id and member_id are required", 400).Throw()
	}

	// Get context
	ctx := process.Context
	if ctx == nil {
		ctx = context.Background()
	}

	// Call business logic
	result, err := memberGet(ctx, userIDStr, teamID, memberID)
	if err != nil {
		exception.New("failed to get member: %s", 500, err.Error()).Throw()
	}

	return result
}

// ProcessMemberCreateDirect user.member.create Member create processor
// Args[0] string: team_id
// Args[1] map: Member data {"user_id": "user123", "member_type": "user", "role_id": "member", "settings": {...}}
// Return: map: {"member_id": "created_member_id"}
func ProcessMemberCreateDirect(process *process.Process) interface{} {
	process.ValidateArgNums(2)

	// Get user_id from session
	userIDStr := GetUserIDFromSession(process)

	teamID := process.ArgsString(0)
	memberData := maps.MapStrAny(process.ArgsMap(1))

	if teamID == "" {
		exception.New("team_id is required", 400).Throw()
	}

	// Validate required fields
	if _, ok := memberData["user_id"]; !ok {
		exception.New("user_id is required", 400).Throw()
	}
	if _, ok := memberData["role_id"]; !ok {
		exception.New("role_id is required", 400).Throw()
	}

	// Get context
	ctx := process.Context
	if ctx == nil {
		ctx = context.Background()
	}

	// Call business logic
	memberID, err := memberCreateDirect(ctx, userIDStr, teamID, memberData)
	if err != nil {
		exception.New("failed to create member: %s", 500, err.Error()).Throw()
	}

	return map[string]interface{}{
		"member_id": memberID,
	}
}

// ProcessMemberUpdate user.member.update Member update processor
// Args[0] string: team_id
// Args[1] string: member_id
// Args[2] map: Update data {"role_id": "admin", "status": "active", "settings": {...}}
// Return: map: {"message": "success"}
func ProcessMemberUpdate(process *process.Process) interface{} {
	process.ValidateArgNums(3)

	// Get user_id from session
	userIDStr := GetUserIDFromSession(process)

	teamID := process.ArgsString(0)
	memberID := process.ArgsString(1)
	updateData := maps.MapStrAny(process.ArgsMap(2))

	if teamID == "" || memberID == "" {
		exception.New("team_id and member_id are required", 400).Throw()
	}

	// Get context
	ctx := process.Context
	if ctx == nil {
		ctx = context.Background()
	}

	// Call business logic
	err := memberUpdate(ctx, userIDStr, teamID, memberID, updateData)
	if err != nil {
		exception.New("failed to update member: %s", 500, err.Error()).Throw()
	}

	return map[string]interface{}{
		"message": "success",
	}
}

// ProcessMemberDelete user.member.delete Member delete processor
// Args[0] string: team_id
// Args[1] string: member_id
// Return: map: {"message": "success"}
func ProcessMemberDelete(process *process.Process) interface{} {
	process.ValidateArgNums(2)

	// Get user_id from session
	userIDStr := GetUserIDFromSession(process)

	teamID := process.ArgsString(0)
	memberID := process.ArgsString(1)

	if teamID == "" || memberID == "" {
		exception.New("team_id and member_id are required", 400).Throw()
	}

	// Get context
	ctx := process.Context
	if ctx == nil {
		ctx = context.Background()
	}

	// Call business logic
	err := memberDelete(ctx, userIDStr, teamID, memberID)
	if err != nil {
		exception.New("failed to delete member: %s", 500, err.Error()).Throw()
	}

	return map[string]interface{}{
		"message": "success",
	}
}

// Private Business Logic Functions (internal use only)

// memberList handles the business logic for listing team members
func memberList(ctx context.Context, userID, teamID string, page, pagesize int, status string) (maps.MapStr, error) {
	// Check if user has access to the team (read permission: owner or member)
	isOwner, isMember, err := checkTeamAccess(ctx, teamID, userID)
	if err != nil {
		return nil, err
	}

	// Allow access if user is owner or member
	if !isOwner && !isMember {
		return nil, fmt.Errorf("access denied: user is not a member of this team")
	}

	// Get user provider instance
	provider, err := getUserProvider()
	if err != nil {
		return nil, fmt.Errorf("failed to get user provider: %w", err)
	}

	// Build query parameters
	param := model.QueryParam{
		Wheres: []model.QueryWhere{
			{Column: "team_id", Value: teamID},
		},
		Orders: []model.QueryOrder{
			{Column: "joined_at", Option: "desc"},
			{Column: "created_at", Option: "desc"},
		},
	}

	// Add status filter if provided
	if status != "" {
		param.Wheres = append(param.Wheres, model.QueryWhere{
			Column: "status",
			Value:  status,
		})
	}

	// Get paginated members
	result, err := provider.PaginateMembers(ctx, param, page, pagesize)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve members: %w", err)
	}

	return result, nil
}

// memberGet handles the business logic for getting a specific team member
func memberGet(ctx context.Context, userID, teamID, memberID string) (maps.MapStrAny, error) {
	// Check if user has access to the team (read permission: owner or member)
	isOwner, isMember, err := checkTeamAccess(ctx, teamID, userID)
	if err != nil {
		return nil, err
	}

	// Allow access if user is owner or member
	if !isOwner && !isMember {
		return nil, fmt.Errorf("access denied: user is not a member of this team")
	}

	// Get user provider instance
	provider, err := getUserProvider()
	if err != nil {
		return nil, fmt.Errorf("failed to get user provider: %w", err)
	}

	// Get member details using team_id + user_id (business keys)
	// memberID parameter is actually user_id in the context of team_id
	memberData, err := provider.GetMember(ctx, teamID, memberID)
	if err != nil {
		return nil, fmt.Errorf("member not found: %w", err)
	}

	return memberData, nil
}

// memberCreateDirect handles the business logic for creating a team member directly
func memberCreateDirect(ctx context.Context, userID, teamID string, memberData maps.MapStrAny) (int64, error) {
	// Check if user has access to the team (write permission: owner only)
	isOwner, _, err := checkTeamAccess(ctx, teamID, userID)
	if err != nil {
		return 0, err
	}

	// Only allow access if user is owner
	if !isOwner {
		return 0, fmt.Errorf("access denied: only team owner can add members")
	}

	// Get user provider instance
	provider, err := getUserProvider()
	if err != nil {
		return 0, fmt.Errorf("failed to get user provider: %w", err)
	}

	// Check if member already exists
	memberUserID := toString(memberData["user_id"])
	exists, err := provider.MemberExists(ctx, teamID, memberUserID)
	if err != nil {
		return 0, fmt.Errorf("failed to check member existence: %w", err)
	}
	if exists {
		return 0, fmt.Errorf("member already exists in this team")
	}

	// Set team ID and default values
	memberData["team_id"] = teamID
	if memberData["member_type"] == nil || memberData["member_type"] == "" {
		memberData["member_type"] = "user"
	}
	memberData["status"] = "active"
	memberData["joined_at"] = time.Now()
	memberData["created_at"] = time.Now()
	memberData["updated_at"] = time.Now()

	// Create member
	memberID, err := provider.CreateMember(ctx, memberData)
	if err != nil {
		return 0, fmt.Errorf("failed to create member: %w", err)
	}

	return memberID, nil
}

// memberUpdate handles the business logic for updating a team member
func memberUpdate(ctx context.Context, userID, teamID, memberUserID string, updateData maps.MapStrAny) error {
	// Check if user has access to the team (write permission: owner only)
	isOwner, _, err := checkTeamAccess(ctx, teamID, userID)
	if err != nil {
		return err
	}

	// Only allow access if user is owner
	if !isOwner {
		return fmt.Errorf("access denied: only team owner can update members")
	}

	// Get user provider instance
	provider, err := getUserProvider()
	if err != nil {
		return fmt.Errorf("failed to get user provider: %w", err)
	}

	// Check if member exists using team_id + user_id (business keys)
	_, err = provider.GetMember(ctx, teamID, memberUserID)
	if err != nil {
		return fmt.Errorf("member not found: %w", err)
	}

	// Add updated_at timestamp
	updateData["updated_at"] = time.Now()

	// Update member using team_id + user_id
	err = provider.UpdateMember(ctx, teamID, memberUserID, updateData)
	if err != nil {
		return fmt.Errorf("failed to update member: %w", err)
	}

	return nil
}

// memberDelete handles the business logic for deleting a team member
func memberDelete(ctx context.Context, userID, teamID, memberID string) error {
	// Check if user has access to the team (write permission: owner only)
	isOwner, _, err := checkTeamAccess(ctx, teamID, userID)
	if err != nil {
		return err
	}

	// Only allow access if user is owner
	if !isOwner {
		return fmt.Errorf("access denied: only team owner can remove members")
	}

	// Get user provider instance
	provider, err := getUserProvider()
	if err != nil {
		return fmt.Errorf("failed to get user provider: %w", err)
	}

	// Check if member exists using team_id + user_id (business keys)
	// memberID parameter is actually user_id in the context of team_id
	_, err = provider.GetMember(ctx, teamID, memberID)
	if err != nil {
		return fmt.Errorf("member not found: %w", err)
	}

	// Remove member using team_id + user_id
	err = provider.RemoveMember(ctx, teamID, memberID)
	if err != nil {
		return fmt.Errorf("failed to delete member: %w", err)
	}

	return nil
}

// Private Helper Functions (internal use only)

// checkTeamAccess checks if user has access to the team
// Returns: (isOwner bool, isMember bool, error)
func checkTeamAccess(ctx context.Context, teamID, userID string) (bool, bool, error) {
	// Get user provider instance
	provider, err := getUserProvider()
	if err != nil {
		return false, false, fmt.Errorf("failed to get user provider: %w", err)
	}

	// Use UserProvider's CheckTeamAccess method - note parameter order: (ctx, teamID, userID)
	return provider.CheckTeamAccess(ctx, teamID, userID)
}

// mapToMemberResponse converts a map to MemberResponse
func mapToMemberResponse(data maps.MapStr) MemberResponse {
	member := MemberResponse{
		ID:           toInt64(data["id"]),
		TeamID:       toString(data["team_id"]),
		UserID:       toString(data["user_id"]),
		MemberType:   toString(data["member_type"]),
		RoleID:       toString(data["role_id"]),
		Status:       toString(data["status"]),
		InvitedBy:    toString(data["invited_by"]),
		InvitedAt:    toTimeString(data["invited_at"]),
		JoinedAt:     toTimeString(data["joined_at"]),
		LastActivity: toTimeString(data["last_activity"]),
		CreatedAt:    toTimeString(data["created_at"]),
		UpdatedAt:    toTimeString(data["updated_at"]),
	}

	// Add settings if available
	if settings, ok := data["settings"]; ok {
		if memSettings, ok := settings.(*MemberSettings); ok {
			member.Settings = memSettings
		} else if settingsMap, ok := settings.(map[string]interface{}); ok {
			// Convert map to MemberSettings (for backward compatibility)
			memSettings := &MemberSettings{
				Notifications: toBool(settingsMap["notifications"]),
			}
			// Handle permissions array
			if perms, ok := settingsMap["permissions"]; ok {
				if permsSlice, ok := perms.([]interface{}); ok {
					permissions := make([]string, 0, len(permsSlice))
					for _, p := range permsSlice {
						if permStr, ok := p.(string); ok {
							permissions = append(permissions, permStr)
						}
					}
					memSettings.Permissions = permissions
				} else if permsStrSlice, ok := perms.([]string); ok {
					memSettings.Permissions = permsStrSlice
				}
			}
			member.Settings = memSettings
		}
	}

	return member
}

// mapToMemberDetailResponse converts a map to MemberDetailResponse
func mapToMemberDetailResponse(data maps.MapStr) MemberDetailResponse {
	member := MemberDetailResponse{
		MemberResponse: mapToMemberResponse(data),
	}

	// Add user info if available (could be joined from user table)
	if userInfo, ok := data["user_info"]; ok {
		if userInfoMap, ok := userInfo.(map[string]interface{}); ok {
			member.UserInfo = userInfoMap
		}
	}

	return member
}
