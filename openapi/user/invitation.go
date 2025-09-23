package user

import (
	"context"
	"crypto/rand"
	"encoding/base64"
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
	"github.com/yaoapp/yao/openapi/response"
)

// Team Invitation Management Handlers

// GinInvitationList handles GET /teams/:team_id/invitations - Get team invitations
func GinInvitationList(c *gin.Context) {
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
	result, err := invitationList(c.Request.Context(), authInfo.UserID, teamID, page, pagesize, c.Query("status"))
	if err != nil {
		log.Error("Failed to get team invitations: %v", err)
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
				ErrorDescription: "Failed to retrieve team invitations",
			}
			response.RespondWithError(c, response.StatusInternalServerError, errorResp)
		}
		return
	}

	// Return the paginated result
	c.JSON(http.StatusOK, result)
}

// GinInvitationGet handles GET /teams/:team_id/invitations/:invitation_id - Get invitation details
func GinInvitationGet(c *gin.Context) {
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
	invitationID := c.Param("invitation_id")
	if teamID == "" || invitationID == "" {
		errorResp := &response.ErrorResponse{
			Code:             response.ErrInvalidRequest.Code,
			ErrorDescription: "Team ID and Invitation ID are required",
		}
		response.RespondWithError(c, response.StatusBadRequest, errorResp)
		return
	}

	// Call business logic
	invitationData, err := invitationGet(c.Request.Context(), authInfo.UserID, teamID, invitationID)
	if err != nil {
		log.Error("Failed to get invitation details: %v", err)
		// Check error type for appropriate response
		if strings.Contains(err.Error(), "not found") {
			errorResp := &response.ErrorResponse{
				Code:             response.ErrInvalidRequest.Code,
				ErrorDescription: "Invitation not found",
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
				ErrorDescription: "Failed to retrieve invitation details",
			}
			response.RespondWithError(c, response.StatusInternalServerError, errorResp)
		}
		return
	}

	// Convert to response format
	invitation := mapToInvitationDetailResponse(invitationData)
	c.JSON(http.StatusOK, invitation)
}

// GinInvitationCreate handles POST /teams/:team_id/invitations - Send team invitation
func GinInvitationCreate(c *gin.Context) {
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
	var req CreateInvitationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		errorResp := &response.ErrorResponse{
			Code:             response.ErrInvalidRequest.Code,
			ErrorDescription: "Invalid request body: " + err.Error(),
		}
		response.RespondWithError(c, response.StatusBadRequest, errorResp)
		return
	}

	// Prepare invitation data
	invitationData := maps.MapStrAny{
		"user_id":     req.UserID,
		"member_type": req.MemberType,
		"role_id":     req.RoleID,
		"message":     req.Message,
	}

	// Add settings if provided
	if req.Settings != nil {
		invitationData["settings"] = req.Settings
	}

	// Call business logic
	invitationID, err := invitationCreate(c.Request.Context(), authInfo.UserID, teamID, invitationData)
	if err != nil {
		log.Error("Failed to create invitation: %v", err)
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
		} else if strings.Contains(err.Error(), "already exists") || strings.Contains(err.Error(), "already invited") {
			errorResp := &response.ErrorResponse{
				Code:             response.ErrInvalidRequest.Code,
				ErrorDescription: err.Error(),
			}
			response.RespondWithError(c, response.StatusConflict, errorResp)
		} else {
			errorResp := &response.ErrorResponse{
				Code:             response.ErrServerError.Code,
				ErrorDescription: "Failed to send invitation",
			}
			response.RespondWithError(c, response.StatusInternalServerError, errorResp)
		}
		return
	}

	// Return created invitation ID
	c.JSON(http.StatusCreated, gin.H{"invitation_id": invitationID})
}

// GinInvitationResend handles PUT /teams/:team_id/invitations/:invitation_id/resend - Resend invitation
func GinInvitationResend(c *gin.Context) {
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
	invitationID := c.Param("invitation_id")
	if teamID == "" || invitationID == "" {
		errorResp := &response.ErrorResponse{
			Code:             response.ErrInvalidRequest.Code,
			ErrorDescription: "Team ID and Invitation ID are required",
		}
		response.RespondWithError(c, response.StatusBadRequest, errorResp)
		return
	}

	// Call business logic
	err := invitationResend(c.Request.Context(), authInfo.UserID, teamID, invitationID)
	if err != nil {
		log.Error("Failed to resend invitation: %v", err)
		// Check error type for appropriate response
		if strings.Contains(err.Error(), "not found") {
			errorResp := &response.ErrorResponse{
				Code:             response.ErrInvalidRequest.Code,
				ErrorDescription: "Invitation not found",
			}
			response.RespondWithError(c, response.StatusNotFound, errorResp)
		} else if strings.Contains(err.Error(), "access denied") {
			errorResp := &response.ErrorResponse{
				Code:             response.ErrAccessDenied.Code,
				ErrorDescription: err.Error(),
			}
			response.RespondWithError(c, response.StatusForbidden, errorResp)
		} else if strings.Contains(err.Error(), "already accepted") || strings.Contains(err.Error(), "invalid status") {
			errorResp := &response.ErrorResponse{
				Code:             response.ErrInvalidRequest.Code,
				ErrorDescription: err.Error(),
			}
			response.RespondWithError(c, response.StatusBadRequest, errorResp)
		} else {
			errorResp := &response.ErrorResponse{
				Code:             response.ErrServerError.Code,
				ErrorDescription: "Failed to resend invitation",
			}
			response.RespondWithError(c, response.StatusInternalServerError, errorResp)
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Invitation resent successfully"})
}

// GinInvitationDelete handles DELETE /teams/:team_id/invitations/:invitation_id - Cancel invitation
func GinInvitationDelete(c *gin.Context) {
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
	invitationID := c.Param("invitation_id")
	if teamID == "" || invitationID == "" {
		errorResp := &response.ErrorResponse{
			Code:             response.ErrInvalidRequest.Code,
			ErrorDescription: "Team ID and Invitation ID are required",
		}
		response.RespondWithError(c, response.StatusBadRequest, errorResp)
		return
	}

	// Call business logic
	err := invitationDelete(c.Request.Context(), authInfo.UserID, teamID, invitationID)
	if err != nil {
		log.Error("Failed to cancel invitation: %v", err)
		// Check error type for appropriate response
		if strings.Contains(err.Error(), "not found") {
			errorResp := &response.ErrorResponse{
				Code:             response.ErrInvalidRequest.Code,
				ErrorDescription: "Invitation not found",
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
				ErrorDescription: "Failed to cancel invitation",
			}
			response.RespondWithError(c, response.StatusInternalServerError, errorResp)
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Invitation cancelled successfully"})
}

// Yao Process Handlers (for Yao application calls)

// ProcessInvitationList user.invitation.list Invitation list processor
// Args[0] string: team_id
// Args[1] map: Query parameters {"status": "pending", "page": 1, "pagesize": 20}
// Return: map: Paginated invitation list
func ProcessInvitationList(process *process.Process) interface{} {
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
	result, err := invitationList(ctx, userIDStr, teamID, page, pagesize, status)
	if err != nil {
		exception.New("failed to list invitations: %s", 500, err.Error()).Throw()
	}

	return result
}

// ProcessInvitationGet user.invitation.get Invitation get processor
// Args[0] string: team_id
// Args[1] string: invitation_id
// Return: map: Invitation details
func ProcessInvitationGet(process *process.Process) interface{} {
	process.ValidateArgNums(2)

	// Get user_id from session
	userIDStr := GetUserIDFromSession(process)

	teamID := process.ArgsString(0)
	invitationID := process.ArgsString(1)

	if teamID == "" || invitationID == "" {
		exception.New("team_id and invitation_id are required", 400).Throw()
	}

	// Get context
	ctx := process.Context
	if ctx == nil {
		ctx = context.Background()
	}

	// Call business logic
	result, err := invitationGet(ctx, userIDStr, teamID, invitationID)
	if err != nil {
		exception.New("failed to get invitation: %s", 500, err.Error()).Throw()
	}

	return result
}

// ProcessInvitationCreate user.invitation.create Invitation create processor
// Args[0] string: team_id
// Args[1] map: Invitation data {"user_id": "user123", "member_type": "user", "role_id": "member", "message": "...", "settings": {...}}
// Return: map: {"invitation_id": "created_invitation_id"}
func ProcessInvitationCreate(process *process.Process) interface{} {
	process.ValidateArgNums(2)

	// Get user_id from session
	userIDStr := GetUserIDFromSession(process)

	teamID := process.ArgsString(0)
	invitationData := maps.MapStrAny(process.ArgsMap(1))

	if teamID == "" {
		exception.New("team_id is required", 400).Throw()
	}

	// Validate required fields
	if _, ok := invitationData["user_id"]; !ok {
		exception.New("user_id is required", 400).Throw()
	}
	if _, ok := invitationData["role_id"]; !ok {
		exception.New("role_id is required", 400).Throw()
	}

	// Get context
	ctx := process.Context
	if ctx == nil {
		ctx = context.Background()
	}

	// Call business logic
	invitationID, err := invitationCreate(ctx, userIDStr, teamID, invitationData)
	if err != nil {
		exception.New("failed to create invitation: %s", 500, err.Error()).Throw()
	}

	return map[string]interface{}{
		"invitation_id": invitationID,
	}
}

// ProcessInvitationResend user.invitation.resend Invitation resend processor
// Args[0] string: team_id
// Args[1] string: invitation_id
// Return: map: {"message": "success"}
func ProcessInvitationResend(process *process.Process) interface{} {
	process.ValidateArgNums(2)

	// Get user_id from session
	userIDStr := GetUserIDFromSession(process)

	teamID := process.ArgsString(0)
	invitationID := process.ArgsString(1)

	if teamID == "" || invitationID == "" {
		exception.New("team_id and invitation_id are required", 400).Throw()
	}

	// Get context
	ctx := process.Context
	if ctx == nil {
		ctx = context.Background()
	}

	// Call business logic
	err := invitationResend(ctx, userIDStr, teamID, invitationID)
	if err != nil {
		exception.New("failed to resend invitation: %s", 500, err.Error()).Throw()
	}

	return map[string]interface{}{
		"message": "success",
	}
}

// ProcessInvitationDelete user.invitation.delete Invitation delete processor
// Args[0] string: team_id
// Args[1] string: invitation_id
// Return: map: {"message": "success"}
func ProcessInvitationDelete(process *process.Process) interface{} {
	process.ValidateArgNums(2)

	// Get user_id from session
	userIDStr := GetUserIDFromSession(process)

	teamID := process.ArgsString(0)
	invitationID := process.ArgsString(1)

	if teamID == "" || invitationID == "" {
		exception.New("team_id and invitation_id are required", 400).Throw()
	}

	// Get context
	ctx := process.Context
	if ctx == nil {
		ctx = context.Background()
	}

	// Call business logic
	err := invitationDelete(ctx, userIDStr, teamID, invitationID)
	if err != nil {
		exception.New("failed to delete invitation: %s", 500, err.Error()).Throw()
	}

	return map[string]interface{}{
		"message": "success",
	}
}

// Private Business Logic Functions (internal use only)

// invitationList handles the business logic for listing team invitations
func invitationList(ctx context.Context, userID, teamID string, page, pagesize int, status string) (maps.MapStr, error) {
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

	// Build query parameters for pending invitations
	param := model.QueryParam{
		Wheres: []model.QueryWhere{
			{Column: "team_id", Value: teamID},
			{Column: "status", Value: "pending"}, // Only show pending invitations
		},
		Orders: []model.QueryOrder{
			{Column: "invited_at", Option: "desc"},
			{Column: "created_at", Option: "desc"},
		},
	}

	// Add additional status filter if provided
	if status != "" && status != "pending" {
		// Replace the default pending status filter
		param.Wheres[1] = model.QueryWhere{
			Column: "status",
			Value:  status,
		}
	}

	// Get paginated invitations (pending members)
	result, err := provider.PaginateMembers(ctx, param, page, pagesize)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve invitations: %w", err)
	}

	return result, nil
}

// invitationGet handles the business logic for getting a specific team invitation
func invitationGet(ctx context.Context, userID, teamID, invitationID string) (maps.MapStrAny, error) {
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

	// Get invitation details using invitation_id (business key)
	invitationData, err := provider.GetMemberByInvitationID(ctx, invitationID)
	if err != nil {
		return nil, fmt.Errorf("invitation not found: %w", err)
	}

	// Verify invitation belongs to this team
	if toString(invitationData["team_id"]) != teamID {
		return nil, fmt.Errorf("invitation not found in this team")
	}

	// Only return if it's a pending invitation
	if toString(invitationData["status"]) != "pending" {
		return nil, fmt.Errorf("invitation not found or no longer pending")
	}

	return invitationData, nil
}

// invitationCreate handles the business logic for creating a team invitation
func invitationCreate(ctx context.Context, userID, teamID string, invitationData maps.MapStrAny) (string, error) {
	// Check if user has access to the team (write permission: owner only)
	isOwner, _, err := checkTeamAccess(ctx, teamID, userID)
	if err != nil {
		return "", err
	}

	// Only allow access if user is owner
	if !isOwner {
		return "", fmt.Errorf("access denied: only team owner can send invitations")
	}

	// Get user provider instance
	provider, err := getUserProvider()
	if err != nil {
		return "", fmt.Errorf("failed to get user provider: %w", err)
	}

	// Check if user is already a member or has pending invitation (if user_id is provided)
	var inviteeUserID string
	if invitationData["user_id"] != nil && invitationData["user_id"] != "" {
		inviteeUserID = toString(invitationData["user_id"])
		exists, err := provider.MemberExists(ctx, teamID, inviteeUserID)
		if err != nil {
			return "", fmt.Errorf("failed to check member existence: %w", err)
		}
		if exists {
			return "", fmt.Errorf("user is already a member or has a pending invitation")
		}
	} else {
		// For unregistered users, set user_id to nil (NULL in database)
		invitationData["user_id"] = nil
	}

	// Generate invitation token
	token, err := generateInvitationToken()
	if err != nil {
		return "", fmt.Errorf("failed to generate invitation token: %w", err)
	}

	// Set invitation-specific fields
	invitationData["team_id"] = teamID
	if invitationData["member_type"] == nil || invitationData["member_type"] == "" {
		invitationData["member_type"] = "user"
	}
	invitationData["status"] = "pending"
	invitationData["invited_by"] = userID
	invitationData["invited_at"] = time.Now()
	invitationData["invitation_token"] = token
	invitationData["invitation_expires_at"] = time.Now().Add(7 * 24 * time.Hour) // 7 days expiry
	invitationData["created_at"] = time.Now()
	invitationData["updated_at"] = time.Now()

	// Create invitation (as a pending member)
	memberID, err := provider.CreateMember(ctx, invitationData)
	if err != nil {
		return "", fmt.Errorf("failed to create invitation: %w", err)
	}

	// Get the created member to retrieve the generated invitation_id
	createdMember, err := provider.GetMemberByID(ctx, memberID)
	if err != nil {
		return "", fmt.Errorf("failed to retrieve created invitation: %w", err)
	}

	// Get the generated invitation_id
	invitationID := toString(createdMember["invitation_id"])

	// TODO: Send invitation email/notification here
	// This would typically involve calling an email service or notification system

	return invitationID, nil
}

// invitationResend handles the business logic for resending a team invitation
func invitationResend(ctx context.Context, userID, teamID, invitationID string) error {
	// Check if user has access to the team (write permission: owner only)
	isOwner, _, err := checkTeamAccess(ctx, teamID, userID)
	if err != nil {
		return err
	}

	// Only allow access if user is owner
	if !isOwner {
		return fmt.Errorf("access denied: only team owner can resend invitations")
	}

	// Get user provider instance
	provider, err := getUserProvider()
	if err != nil {
		return fmt.Errorf("failed to get user provider: %w", err)
	}

	// Get existing invitation using invitation_id (business key)
	invitationData, err := provider.GetMemberByInvitationID(ctx, invitationID)
	if err != nil {
		return fmt.Errorf("invitation not found: %w", err)
	}

	// Verify invitation belongs to this team
	if toString(invitationData["team_id"]) != teamID {
		return fmt.Errorf("invitation not found in this team")
	}

	// Check if invitation is still pending
	if toString(invitationData["status"]) != "pending" {
		return fmt.Errorf("invitation is no longer pending and cannot be resent")
	}

	// Generate new invitation token
	newToken, err := generateInvitationToken()
	if err != nil {
		return fmt.Errorf("failed to generate new invitation token: %w", err)
	}

	// Update invitation with new token and extended expiry
	updateData := maps.MapStrAny{
		"invitation_token":      newToken,
		"invitation_expires_at": time.Now().Add(7 * 24 * time.Hour), // Extend for another 7 days
		"invited_at":            time.Now(),                         // Update invitation time
		"updated_at":            time.Now(),
	}

	// Update invitation using invitation_id
	err = provider.UpdateMemberByInvitationID(ctx, invitationID, updateData)
	if err != nil {
		return fmt.Errorf("failed to update invitation: %w", err)
	}

	// TODO: Send new invitation email/notification here
	// This would typically involve calling an email service or notification system

	return nil
}

// invitationDelete handles the business logic for cancelling a team invitation
func invitationDelete(ctx context.Context, userID, teamID, invitationID string) error {
	// Check if user has access to the team (write permission: owner only)
	isOwner, _, err := checkTeamAccess(ctx, teamID, userID)
	if err != nil {
		return err
	}

	// Only allow access if user is owner
	if !isOwner {
		return fmt.Errorf("access denied: only team owner can cancel invitations")
	}

	// Get user provider instance
	provider, err := getUserProvider()
	if err != nil {
		return fmt.Errorf("failed to get user provider: %w", err)
	}

	// Get existing invitation using invitation_id (business key)
	invitationData, err := provider.GetMemberByInvitationID(ctx, invitationID)
	if err != nil {
		return fmt.Errorf("invitation not found: %w", err)
	}

	// Verify invitation belongs to this team
	if toString(invitationData["team_id"]) != teamID {
		return fmt.Errorf("invitation not found in this team")
	}

	// Check if invitation is still pending
	if toString(invitationData["status"]) != "pending" {
		return fmt.Errorf("invitation is no longer pending and cannot be cancelled")
	}

	// Remove the pending invitation (delete the member record)
	err = provider.RemoveMemberByInvitationID(ctx, invitationID)
	if err != nil {
		return fmt.Errorf("failed to cancel invitation: %w", err)
	}

	return nil
}

// Private Helper Functions (internal use only)

// generateInvitationToken generates a secure random token for invitations
func generateInvitationToken() (string, error) {
	bytes := make([]byte, 32) // 32 bytes = 256 bits
	_, err := rand.Read(bytes)
	if err != nil {
		return "", err
	}
	// Use URL-safe base64 encoding and remove padding
	return strings.TrimRight(base64.URLEncoding.EncodeToString(bytes), "="), nil
}

// mapToInvitationResponse converts a map to InvitationResponse
func mapToInvitationResponse(data maps.MapStr) InvitationResponse {
	invitation := InvitationResponse{
		ID:                  toInt64(data["id"]),
		TeamID:              toString(data["team_id"]),
		UserID:              toString(data["user_id"]),
		MemberType:          toString(data["member_type"]),
		RoleID:              toString(data["role_id"]),
		Status:              toString(data["status"]),
		InvitedBy:           toString(data["invited_by"]),
		InvitedAt:           toTimeString(data["invited_at"]),
		InvitationToken:     toString(data["invitation_token"]),
		InvitationExpiresAt: toTimeString(data["invitation_expires_at"]),
		Message:             toString(data["message"]),
		CreatedAt:           toTimeString(data["created_at"]),
		UpdatedAt:           toTimeString(data["updated_at"]),
	}

	// Add settings if available
	if settings, ok := data["settings"]; ok {
		if settingsMap, ok := settings.(map[string]interface{}); ok {
			invitation.Settings = settingsMap
		}
	}

	return invitation
}

// mapToInvitationDetailResponse converts a map to InvitationDetailResponse
func mapToInvitationDetailResponse(data maps.MapStr) InvitationDetailResponse {
	invitation := InvitationDetailResponse{
		InvitationResponse: mapToInvitationResponse(data),
	}

	// Add user info if available (could be joined from user table)
	if userInfo, ok := data["user_info"]; ok {
		if userInfoMap, ok := userInfo.(map[string]interface{}); ok {
			invitation.UserInfo = userInfoMap
		}
	}

	// Add team info if available (could be joined from team table)
	if teamInfo, ok := data["team_info"]; ok {
		if teamInfoMap, ok := teamInfo.(map[string]interface{}); ok {
			invitation.TeamInfo = teamInfoMap
		}
	}

	return invitation
}
