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
	"github.com/yaoapp/yao/attachment"
	"github.com/yaoapp/yao/messenger"
	messengertypes "github.com/yaoapp/yao/messenger/types"
	"github.com/yaoapp/yao/openapi/oauth"
	"github.com/yaoapp/yao/openapi/oauth/authorized"
	"github.com/yaoapp/yao/openapi/response"
	"github.com/yaoapp/yao/openapi/utils"
	"github.com/yaoapp/yao/share"
)

// User Team Invitation Management Handlers

// GinTeamInvitationList handles GET /teams/:team_id/invitations - Get team invitations
func GinTeamInvitationList(c *gin.Context) {
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
	result, err := teamInvitationList(c.Request.Context(), authInfo.UserID, teamID, page, pagesize, c.Query("status"))
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
	response.RespondWithSuccess(c, http.StatusOK, result)
}

// GinTeamInvitationGetPublic handles GET /user/teams/invitations/:invitation_id - Get invitation details (public)
// This is a public endpoint that doesn't require authentication
// Supports ?locale=zh-CN query parameter for internationalization
func GinTeamInvitationGetPublic(c *gin.Context) {
	invitationID := c.Param("invitation_id")
	if invitationID == "" {
		errorResp := &response.ErrorResponse{
			Code:             response.ErrInvalidRequest.Code,
			ErrorDescription: "Invitation ID is required",
		}
		response.RespondWithError(c, response.StatusBadRequest, errorResp)
		return
	}

	// Get locale from query parameter or Accept-Language header
	locale := c.Query("locale")
	if locale == "" {
		// Try to get from Accept-Language header
		acceptLang := c.GetHeader("Accept-Language")
		if acceptLang != "" {
			// Parse Accept-Language header (e.g., "zh-CN,zh;q=0.9,en;q=0.8")
			parts := strings.Split(acceptLang, ",")
			if len(parts) > 0 {
				langParts := strings.Split(parts[0], ";")
				if len(langParts) > 0 {
					locale = strings.TrimSpace(langParts[0])
				}
			}
		}
	}
	// Default to "en" if no locale specified
	if locale == "" {
		locale = "en"
	}

	// Call business logic (no user authentication required for public access)
	publicInvitation, err := teamInvitationGetPublic(c.Request.Context(), invitationID, locale)
	if err != nil {
		log.Error("Failed to get invitation details: %v", err)
		// Check error type for appropriate response
		if strings.Contains(err.Error(), "not found") || strings.Contains(err.Error(), "expired") {
			errorResp := &response.ErrorResponse{
				Code:             response.ErrInvalidRequest.Code,
				ErrorDescription: "Invitation not found or expired",
			}
			response.RespondWithError(c, response.StatusNotFound, errorResp)
		} else {
			errorResp := &response.ErrorResponse{
				Code:             response.ErrServerError.Code,
				ErrorDescription: "Failed to retrieve invitation details",
			}
			response.RespondWithError(c, response.StatusInternalServerError, errorResp)
		}
		return
	}

	response.RespondWithSuccess(c, http.StatusOK, publicInvitation)
}

// GinTeamInvitationGet handles GET /teams/:team_id/invitations/:invitation_id - Get invitation details (admin only)
func GinTeamInvitationGet(c *gin.Context) {
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
	invitationID := c.Param("invitation_id")
	if teamID == "" || invitationID == "" {
		errorResp := &response.ErrorResponse{
			Code:             response.ErrInvalidRequest.Code,
			ErrorDescription: "Team ID and Invitation ID are required",
		}
		response.RespondWithError(c, response.StatusBadRequest, errorResp)
		return
	}

	// Extract base URL from request
	requestBaseURL := getRequestBaseURL(c)

	// Call business logic
	invitationData, err := teamInvitationGet(c.Request.Context(), authInfo.UserID, teamID, invitationID)
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

	// Convert to response format (with requestBaseURL for building full invitation link)
	invitation := mapToTeamInvitationDetailResponse(invitationData, requestBaseURL)
	response.RespondWithSuccess(c, http.StatusOK, invitation)
}

// GinTeamInvitationCreate handles POST /teams/:team_id/invitations - Send team invitation
func GinTeamInvitationCreate(c *gin.Context) {
	// Get authorized user info
	authInfo := authorized.GetInfo(c)
	if authInfo.Constraints.OwnerOnly || authInfo.Constraints.TeamOnly {
		if authInfo == nil || authInfo.UserID == "" {
			errorResp := &response.ErrorResponse{
				Code:             response.ErrInvalidClient.Code,
				ErrorDescription: "User not authenticated",
			}
			response.RespondWithError(c, response.StatusUnauthorized, errorResp)
			return
		}
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
	var req CreateInvitationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		errorResp := &response.ErrorResponse{
			Code:             response.ErrInvalidRequest.Code,
			ErrorDescription: "Invalid request body: " + err.Error(),
		}
		response.RespondWithError(c, response.StatusBadRequest, errorResp)
		return
	}

	// Extract base URL from request
	requestBaseURL := getRequestBaseURL(c)

	// Prepare invitation data
	invitationData := authInfo.WithCreateScope(maps.MapStrAny{
		"user_id":          req.UserID,
		"email":            req.Email,
		"member_type":      req.MemberType,
		"role_id":          req.RoleID,
		"message":          req.Message,
		"expiry":           req.Expiry,
		"request_base_url": requestBaseURL,
	})

	// Prepare settings
	settings := &InvitationSettings{}
	if req.Settings != nil {
		settings = req.Settings
	}

	// Add send_email from top-level field (for backward compatibility)
	if req.SendEmail != nil {
		settings.SendEmail = *req.SendEmail
	}

	// Add locale from top-level field (for backward compatibility)
	if req.Locale != "" {
		settings.Locale = req.Locale
	}

	// Add settings to invitation data
	if settings.SendEmail || settings.Locale != "" {
		invitationData["settings"] = settings
	}

	// Call business logic
	invitationID, err := teamInvitationCreate(c.Request.Context(), authInfo.UserID, teamID, invitationData)
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
		} else if strings.Contains(err.Error(), "email is required") || strings.Contains(err.Error(), "is required") {
			errorResp := &response.ErrorResponse{
				Code:             response.ErrInvalidRequest.Code,
				ErrorDescription: err.Error(),
			}
			response.RespondWithError(c, response.StatusBadRequest, errorResp)
		} else {
			errorResp := &response.ErrorResponse{
				Code:             response.ErrServerError.Code,
				ErrorDescription: "Failed to send invitation",
			}
			response.RespondWithError(c, response.StatusInternalServerError, errorResp)
		}
		return
	}

	// Get the created invitation to return complete data
	invitation, err := teamInvitationGet(c.Request.Context(), authInfo.UserID, teamID, invitationID)
	if err != nil {
		log.Error("Failed to retrieve created invitation: %v", err)
		// Fallback to returning just the ID if retrieval fails
		response.RespondWithSuccess(c, http.StatusCreated, gin.H{"invitation_id": invitationID})
		return
	}

	// Convert to InvitationResponse (with requestBaseURL for building full invitation link)
	invitationResp := convertToTeamInvitationResponse(invitation, requestBaseURL)

	// Return created invitation with full details (including token)
	response.RespondWithSuccess(c, http.StatusCreated, invitationResp)
}

// GinTeamInvitationResend handles PUT /teams/:team_id/invitations/:invitation_id/resend - Resend invitation
func GinTeamInvitationResend(c *gin.Context) {
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
	invitationID := c.Param("invitation_id")
	if teamID == "" || invitationID == "" {
		errorResp := &response.ErrorResponse{
			Code:             response.ErrInvalidRequest.Code,
			ErrorDescription: "Team ID and Invitation ID are required",
		}
		response.RespondWithError(c, response.StatusBadRequest, errorResp)
		return
	}

	// Parse request body for locale
	var requestBody struct {
		Locale string `json:"locale"`
	}
	if err := c.ShouldBindJSON(&requestBody); err != nil {
		// If no body or invalid JSON, try query parameter as fallback
		requestBody.Locale = c.Query("locale")
	}

	// Get locale from request body or query parameter, default to "en"
	locale := requestBody.Locale
	if locale == "" {
		locale = c.Query("locale")
	}
	if locale == "" {
		locale = "en"
	}

	// Get request base URL for invitation link generation
	requestBaseURL := getRequestBaseURL(c)

	// Call business logic
	err := teamInvitationResend(c.Request.Context(), authInfo.UserID, teamID, invitationID, requestBaseURL, locale)
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

	response.RespondWithSuccess(c, http.StatusOK, gin.H{"message": "Invitation resent successfully"})
}

// GinTeamInvitationDelete handles DELETE /teams/:team_id/invitations/:invitation_id - Cancel invitation
func GinTeamInvitationDelete(c *gin.Context) {
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
	err := teamInvitationDelete(c.Request.Context(), authInfo.UserID, teamID, invitationID)
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

	response.RespondWithSuccess(c, http.StatusOK, gin.H{"message": "Invitation cancelled successfully"})
}

// GinTeamInvitationAccept handles POST /user/teams/invitations/:invitation_id/accept - Accept invitation and login to team
func GinTeamInvitationAccept(c *gin.Context) {
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

	ctx := c.Request.Context()

	// Use authInfo.UserID directly - it might be OAuth subject, but LoginByTeamID will handle user creation
	userID := authInfo.UserID

	invitationID := c.Param("invitation_id")
	if invitationID == "" {
		errorResp := &response.ErrorResponse{
			Code:             response.ErrInvalidRequest.Code,
			ErrorDescription: "Invitation ID is required",
		}
		response.RespondWithError(c, response.StatusBadRequest, errorResp)
		return
	}

	// Parse request body to get token
	var req struct {
		Token string `json:"token" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		errorResp := &response.ErrorResponse{
			Code:             response.ErrInvalidRequest.Code,
			ErrorDescription: "Invalid request body: token is required",
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
			ErrorDescription: "Failed to process invitation",
		}
		response.RespondWithError(c, response.StatusInternalServerError, errorResp)
		return
	}

	// Get invitation details first to retrieve team_id
	invitationData, err := provider.GetMemberByInvitationID(ctx, invitationID)
	if err != nil {
		log.Error("Failed to get invitation: %v", err)
		errorResp := &response.ErrorResponse{
			Code:             response.ErrInvalidRequest.Code,
			ErrorDescription: "Invitation not found",
		}
		response.RespondWithError(c, response.StatusNotFound, errorResp)
		return
	}

	// Get team_id from invitation
	teamID := utils.ToString(invitationData["team_id"])
	if teamID == "" {
		log.Error("Invalid invitation: missing team_id")
		errorResp := &response.ErrorResponse{
			Code:             response.ErrServerError.Code,
			ErrorDescription: "Invalid invitation data",
		}
		response.RespondWithError(c, response.StatusInternalServerError, errorResp)
		return
	}

	// Accept the invitation (will update user_id if invitation doesn't have one)
	err = provider.AcceptInvitation(ctx, invitationID, req.Token, userID)
	if err != nil {
		log.Error("Failed to accept invitation: %v", err)
		// Check error type for appropriate response
		if strings.Contains(err.Error(), "not found") || strings.Contains(err.Error(), "already accepted") {
			errorResp := &response.ErrorResponse{
				Code:             response.ErrInvalidRequest.Code,
				ErrorDescription: "Invitation not found or already accepted",
			}
			response.RespondWithError(c, response.StatusNotFound, errorResp)
		} else if strings.Contains(err.Error(), "expired") {
			errorResp := &response.ErrorResponse{
				Code:             response.ErrInvalidRequest.Code,
				ErrorDescription: "Invitation has expired",
			}
			response.RespondWithError(c, response.StatusBadRequest, errorResp)
		} else {
			errorResp := &response.ErrorResponse{
				Code:             response.ErrServerError.Code,
				ErrorDescription: "Failed to accept invitation",
			}
			response.RespondWithError(c, response.StatusInternalServerError, errorResp)
		}
		return
	}

	// Prepare login context with full device/platform information
	loginCtx := makeLoginContext(c)

	// Login with the team that was just joined
	// Note: userID must exist in database (user table)
	loginResponse, err := LoginByTeamID(userID, teamID, loginCtx)
	if err != nil {
		log.Error("Failed to login with team: %v", err)
		errorResp := &response.ErrorResponse{
			Code:             response.ErrServerError.Code,
			ErrorDescription: "Invitation accepted but failed to login: " + err.Error(),
		}
		response.RespondWithError(c, response.StatusInternalServerError, errorResp)
		return
	}

	// Revoke the current token if it exists (similar to team selection)
	currentToken := oauth.OAuth.GetAccessToken(c)
	if currentToken != "" {
		if err := oauth.OAuth.Revoke(ctx, currentToken, "access_token"); err != nil {
			// Log the error but don't fail the request
			log.Warn("Failed to revoke previous token: %v", err)
		}
	}

	// Send secure cookies (access token, refresh token, and session ID)
	SendLoginCookies(c, loginResponse, "")

	// Return the new tokens in response body
	response.RespondWithSuccess(c, http.StatusOK, loginResponse)
}

// Yao Process Handlers (for Yao application calls)

// ProcessTeamInvitationList user.team.invitation.list Team invitation list processor
// Args[0] string: team_id
// Args[1] map: Query parameters {"status": "pending", "page": 1, "pagesize": 20}
// Return: map: Paginated invitation list
func ProcessTeamInvitationList(process *process.Process) interface{} {
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

	if p := int(utils.ToInt64(queryMap["page"])); p > 0 {
		page = p
	}

	if ps := int(utils.ToInt64(queryMap["pagesize"])); ps > 0 && ps <= 100 {
		pagesize = ps
	}

	// Get status filter
	status := utils.ToString(queryMap["status"])

	// Get context
	ctx := process.Context
	if ctx == nil {
		ctx = context.Background()
	}

	// Call business logic
	result, err := teamInvitationList(ctx, userIDStr, teamID, page, pagesize, status)
	if err != nil {
		exception.New("failed to list team invitations: %s", 500, err.Error()).Throw()
	}

	return result
}

// ProcessTeamInvitationGet user.team.invitation.get Team invitation get processor
// Args[0] string: team_id
// Args[1] string: invitation_id
// Return: map: Invitation details
func ProcessTeamInvitationGet(process *process.Process) interface{} {
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
	result, err := teamInvitationGet(ctx, userIDStr, teamID, invitationID)
	if err != nil {
		exception.New("failed to get team invitation: %s", 500, err.Error()).Throw()
	}

	return result
}

// ProcessTeamInvitationCreate user.team.invitation.create Team invitation create processor
// Args[0] string: team_id
// Args[1] map: Invitation data {"user_id": "user123", "member_type": "user", "role_id": "member", "message": "...", "settings": {...}}
// Return: map: {"invitation_id": "created_invitation_id"}
func ProcessTeamInvitationCreate(process *process.Process) interface{} {
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
	invitationID, err := teamInvitationCreate(ctx, userIDStr, teamID, invitationData)
	if err != nil {
		exception.New("failed to create team invitation: %s", 500, err.Error()).Throw()
	}

	return map[string]interface{}{
		"invitation_id": invitationID,
	}
}

// ProcessTeamInvitationResend user.team.invitation.resend Team invitation resend processor
// Args[0] string: team_id
// Args[1] string: invitation_id
// Return: map: {"message": "success"}
func ProcessTeamInvitationResend(process *process.Process) interface{} {
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

	// Get locale from Args[2] if provided, default to "en"
	locale := "en"
	if process.NumOfArgsIs(3) {
		locale = process.ArgsString(2)
	}

	// Call business logic (no requestBaseURL available in process context)
	err := teamInvitationResend(ctx, userIDStr, teamID, invitationID, "", locale)
	if err != nil {
		exception.New("failed to resend team invitation: %s", 500, err.Error()).Throw()
	}

	return map[string]interface{}{
		"message": "success",
	}
}

// ProcessTeamInvitationDelete user.team.invitation.delete Team invitation delete processor
// Args[0] string: team_id
// Args[1] string: invitation_id
// Return: map: {"message": "success"}
func ProcessTeamInvitationDelete(process *process.Process) interface{} {
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
	err := teamInvitationDelete(ctx, userIDStr, teamID, invitationID)
	if err != nil {
		exception.New("failed to delete team invitation: %s", 500, err.Error()).Throw()
	}

	return map[string]interface{}{
		"message": "success",
	}
}

// Private Business Logic Functions (internal use only)

// getAdminRoot returns the admin root path from share.App configuration
// Similar to service.setupAdminRoot but without caching to avoid circular dependencies
func getAdminRoot() string {
	adminRoot := "/yao/"
	if share.App.AdminRoot != "" {
		root := strings.TrimPrefix(share.App.AdminRoot, "/")
		root = strings.TrimSuffix(root, "/")
		adminRoot = fmt.Sprintf("/%s/", root)
	}
	return adminRoot
}

// getRequestBaseURL extracts the base URL from the gin context request
// Returns: scheme://host (e.g., "https://example.com" or "http://localhost:8000")
func getRequestBaseURL(c *gin.Context) string {
	if c == nil || c.Request == nil {
		return ""
	}

	scheme := "http"
	if c.Request.TLS != nil {
		scheme = "https"
	}
	// Check X-Forwarded-Proto header
	if proto := c.GetHeader("X-Forwarded-Proto"); proto != "" {
		scheme = proto
	}

	host := c.Request.Host
	if host == "" {
		return ""
	}

	return fmt.Sprintf("%s://%s", scheme, host)
}

// buildTeamInvitationLink constructs a full invitation link from invitation_id, token and team configuration
// This is a centralized function to ensure consistency across email sending and link generation
// Format:
//   - With team config baseURL: {base_url}/{invitation_id}/{token}
//   - With requestBaseURL (from HTTP request): {scheme}://{host}{AdminRoot}team/invite/{invitation_id}/{token}
//   - Without any baseURL (fallback): {AdminRoot}team/invite/{invitation_id}/{token}
func buildTeamInvitationLink(invitationID, token string, teamConfig *TeamConfig, requestBaseURL string) string {
	// Priority 1: Use team config baseURL if specified
	if teamConfig != nil && teamConfig.Invite != nil && teamConfig.Invite.BaseURL != "" {
		baseURL := teamConfig.Invite.BaseURL
		// Ensure baseURL ends with /
		if !strings.HasSuffix(baseURL, "/") {
			baseURL = baseURL + "/"
		}
		return fmt.Sprintf("%s%s/%s", baseURL, invitationID, token)
	}

	// Get admin root from configuration
	adminRoot := getAdminRoot()
	// Ensure adminRoot doesn't end with / for URL construction
	adminRoot = strings.TrimSuffix(adminRoot, "/")

	// Priority 2: Use request baseURL with AdminRoot
	if requestBaseURL != "" {
		// Ensure requestBaseURL doesn't end with /
		requestBaseURL = strings.TrimSuffix(requestBaseURL, "/")
		return fmt.Sprintf("%s%s/team/invite/%s/%s", requestBaseURL, adminRoot, invitationID, token)
	}

	// Priority 3: Fallback to relative path with AdminRoot
	return fmt.Sprintf("%s/team/invite/%s/%s", adminRoot, invitationID, token)
}

// teamInvitationList handles the business logic for listing team invitations
func teamInvitationList(ctx context.Context, userID, teamID string, page, pagesize int, status string) (maps.MapStr, error) {
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

// teamInvitationGetPublic handles the business logic for getting a specific team invitation (public access)
// This function doesn't require authentication and is used for invitation recipients
// locale parameter is used to get localized role labels
func teamInvitationGetPublic(ctx context.Context, invitationID, locale string) (*PublicInvitationResponse, error) {
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

	// Only return if it's a pending invitation
	if utils.ToString(invitationData["status"]) != "pending" {
		return nil, fmt.Errorf("invitation not found or no longer pending")
	}

	// Check if invitation has expired
	expiresAt := invitationData["invitation_expires_at"]
	if expiresAt != nil {
		var expiryTime time.Time
		switch v := expiresAt.(type) {
		case time.Time:
			expiryTime = v
		case string:
			parsed, err := time.Parse(time.RFC3339, v)
			if err == nil {
				expiryTime = parsed
			}
		}
		if !expiryTime.IsZero() && time.Now().After(expiryTime) {
			return nil, fmt.Errorf("invitation has expired")
		}
	}

	// Get team information
	teamID := utils.ToString(invitationData["team_id"])
	team, err := provider.GetTeam(ctx, teamID)
	if err != nil {
		log.Warn("Failed to get team information: %v", err)
		team = maps.MapStrAny{"name": "Team"}
	}

	// Get inviter information
	inviterID := utils.ToString(invitationData["invited_by"])
	var inviterInfo *InviterInfo
	if inviterID != "" {
		inviter, err := provider.GetUser(ctx, inviterID)
		if err == nil {
			inviterInfo = &InviterInfo{
				UserID:  inviterID,
				Name:    utils.ToString(inviter["name"]),
				Picture: utils.ToString(inviter["picture"]),
			}
			// Fallback to masked email if name is empty (for privacy protection)
			if inviterInfo.Name == "" {
				inviterInfo.Name = maskEmail(utils.ToString(inviter["email"]))
			}
		}
	}

	// Get role label from team config using provided locale
	roleID := utils.ToString(invitationData["role_id"])
	roleLabel := ""
	teamConfig := GetTeamConfig(locale)
	if teamConfig != nil && teamConfig.Roles != nil {
		for _, role := range teamConfig.Roles {
			if role.RoleID == roleID {
				roleLabel = role.Label
				break
			}
		}
	}

	// Process team_logo if it's a wrapper - use Data URI format for direct display in img src
	teamLogo := utils.ToString(team["logo"])
	if teamLogo != "" {
		teamLogo = attachment.Base64(ctx, teamLogo, true)
	}

	// Process inviter_info.picture if it's a wrapper - use Data URI format for direct display in img src
	if inviterInfo != nil && inviterInfo.Picture != "" {
		inviterInfo.Picture = attachment.Base64(ctx, inviterInfo.Picture, true)
	}

	// Build public response (exclude sensitive data like IDs)
	publicResponse := &PublicInvitationResponse{
		InvitationID:        utils.ToString(invitationData["invitation_id"]),
		TeamName:            utils.ToString(team["name"]),
		TeamLogo:            teamLogo,
		TeamDescription:     utils.ToString(team["description"]),
		RoleLabel:           roleLabel,
		Status:              utils.ToString(invitationData["status"]),
		InvitedAt:           utils.ToTimeString(invitationData["invited_at"]),
		InvitationExpiresAt: utils.ToTimeString(invitationData["invitation_expires_at"]),
		Message:             utils.ToString(invitationData["message"]),
		InviterInfo:         inviterInfo,
	}

	return publicResponse, nil
}

// teamInvitationGet handles the business logic for getting a specific team invitation (admin access)
func teamInvitationGet(ctx context.Context, userID, teamID, invitationID string) (maps.MapStrAny, error) {
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
	if utils.ToString(invitationData["team_id"]) != teamID {
		return nil, fmt.Errorf("invitation not found in this team")
	}

	// Only return if it's a pending invitation
	if utils.ToString(invitationData["status"]) != "pending" {
		return nil, fmt.Errorf("invitation not found or no longer pending")
	}

	return invitationData, nil
}

// teamInvitationCreate handles the business logic for creating a team invitation
// Supports two scenarios:
// 1. Email invitation: provide email and role, send invitation link via email
// 2. Link invitation: create invitation link for display in frontend, customizable expiry
func teamInvitationCreate(ctx context.Context, userID, teamID string, invitationData maps.MapStrAny) (string, error) {
	// Remove empty string fields (should not be inserted to database)
	for _, field := range []string{"user_id", "email", "message", "display_name", "bio"} {
		if invitationData[field] == "" {
			delete(invitationData, field)
		}
	}

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

	// Get team information for email template
	team, err := provider.GetTeam(ctx, teamID)
	if err != nil {
		return "", fmt.Errorf("failed to get team information: %w", err)
	}
	teamName := utils.ToString(team["name"])

	// Get inviter information for email template
	inviter, err := provider.GetUser(ctx, userID)
	if err != nil {
		log.Warn("Failed to get inviter information: %v", err)
		inviter = maps.MapStrAny{"name": "Team Admin"}
	}
	inviterName := utils.ToString(inviter["name"])
	if inviterName == "" {
		inviterName = utils.ToString(inviter["email"])
	}

	// Check if user is already a member or has pending invitation (if user_id is provided)
	var inviteeUserID string
	var inviteeEmail string

	// Get email from invitation data first
	inviteeEmail = utils.ToString(invitationData["email"])

	if invitationData["user_id"] != nil && invitationData["user_id"] != "" {
		inviteeUserID = utils.ToString(invitationData["user_id"])
		exists, err := provider.MemberExists(ctx, teamID, inviteeUserID)
		if err != nil {
			return "", fmt.Errorf("failed to check member existence: %w", err)
		}
		if exists {
			return "", fmt.Errorf("user is already a member or has a pending invitation")
		}

		// If email not provided, get it from user profile
		if inviteeEmail == "" {
			user, err := provider.GetUser(ctx, inviteeUserID)
			if err != nil {
				return "", fmt.Errorf("failed to get user information: %w", err)
			}
			inviteeEmail = utils.ToString(user["email"])

			// Update invitation data with email from user profile
			if inviteeEmail != "" {
				invitationData["email"] = inviteeEmail
			}
		}
	} else {
		// For invitations without user_id (general invitation link or unregistered users)
		// Set user_id to nil (NULL in database)
		invitationData["user_id"] = nil
	}

	// Check send_email requirement early
	shouldSendEmail := false
	if settings, ok := invitationData["settings"].(*InvitationSettings); ok && settings != nil {
		shouldSendEmail = settings.SendEmail
	} else if settingsMap, ok := invitationData["settings"].(map[string]interface{}); ok {
		// Fallback for map format (for backward compatibility)
		shouldSendEmail = utils.ToBool(settingsMap["send_email"])
	}

	// If send_email is true, email must be provided
	if shouldSendEmail && inviteeEmail == "" {
		return "", fmt.Errorf("email is required when send_email is true")
	}

	// Generate invitation token
	token, err := generateTeamInvitationToken()
	if err != nil {
		return "", fmt.Errorf("failed to generate invitation token: %w", err)
	}

	// Calculate expiry duration
	expiryDuration, err := getTeamInvitationExpiry(invitationData)
	if err != nil {
		return "", fmt.Errorf("failed to parse expiry duration: %w", err)
	}

	// Save request_base_url and settings before database operation (they will be lost in DB)
	requestBaseURL := utils.ToString(invitationData["request_base_url"])
	savedSettings := invitationData["settings"] // Save settings reference

	// Set invitation-specific fields
	invitationData["team_id"] = teamID
	if invitationData["member_type"] == nil || invitationData["member_type"] == "" {
		invitationData["member_type"] = "user"
	}
	invitationData["status"] = "pending"
	invitationData["invited_by"] = userID
	invitationData["invited_at"] = time.Now()
	invitationData["invitation_token"] = token
	invitationData["invitation_expires_at"] = time.Now().Add(expiryDuration)
	invitationData["created_at"] = time.Now()
	invitationData["updated_at"] = time.Now()

	// Create invitation (as a pending member)
	businessMemberID, err := provider.CreateMember(ctx, invitationData)
	if err != nil {
		return "", fmt.Errorf("failed to create invitation: %w", err)
	}

	// Get the created member to retrieve the generated invitation_id
	createdMember, err := provider.GetMemberByMemberID(ctx, businessMemberID)
	if err != nil {
		return "", fmt.Errorf("failed to retrieve created invitation: %w", err)
	}

	// Get the generated invitation_id
	invitationID := utils.ToString(createdMember["invitation_id"])

	// Send email if requested (shouldSendEmail was already determined earlier)
	if shouldSendEmail {
		// Use the saved requestBaseURL and settings (not from invitationData, as they were lost in DB operation)
		// Send email asynchronously to improve user experience
		go func() {
			// Use background context for async operation
			bgCtx := context.Background()

			// Use createdMember data (from database) for email sending
			// This ensures we have the actual stored values including properly formatted timestamps
			emailData := maps.MapStrAny{}
			for k, v := range createdMember {
				emailData[k] = v
			}
			emailData["request_base_url"] = requestBaseURL
			emailData["settings"] = savedSettings // Restore settings

			err := sendTeamInvitationEmail(bgCtx, inviteeEmail, inviterName, teamName, token, invitationID, emailData)
			if err != nil {
				log.Error("Failed to send invitation email: %v", err)
			} else {
				log.Info("Invitation email sent to %s for team %s (invitation_id: %s)", inviteeEmail, teamName, invitationID)
			}
		}()
	}

	return invitationID, nil
}

// teamInvitationResend handles the business logic for resending a team invitation
func teamInvitationResend(ctx context.Context, userID, teamID, invitationID, requestBaseURL, locale string) error {
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
	if utils.ToString(invitationData["team_id"]) != teamID {
		return fmt.Errorf("invitation not found in this team")
	}

	// Check if invitation is still pending
	if utils.ToString(invitationData["status"]) != "pending" {
		return fmt.Errorf("invitation is no longer pending and cannot be resent")
	}

	// Get email directly from member record's email field
	inviteeEmail := utils.ToString(invitationData["email"])
	if inviteeEmail == "" {
		return fmt.Errorf("invitation has no email address, cannot resend")
	}

	// Get team information for email template
	team, err := provider.GetTeam(ctx, teamID)
	if err != nil {
		return fmt.Errorf("failed to get team information: %w", err)
	}
	teamName := utils.ToString(team["name"])

	// Get inviter information for email template
	inviter, err := provider.GetUser(ctx, userID)
	if err != nil {
		log.Warn("Failed to get inviter information: %v", err)
		inviter = maps.MapStrAny{"name": "Team Admin"}
	}
	inviterName := utils.ToString(inviter["name"])
	if inviterName == "" {
		inviterName = utils.ToString(inviter["email"])
	}

	// Generate new invitation token
	newToken, err := generateTeamInvitationToken()
	if err != nil {
		return fmt.Errorf("failed to generate new invitation token: %w", err)
	}

	// Get team config for expiry duration
	teamConfig := GetTeamConfig(locale)
	if teamConfig == nil || teamConfig.Invite == nil {
		return fmt.Errorf("team configuration not found for locale: %s", locale)
	}

	// Calculate expiry duration from config or use default
	expiryDuration := 7 * 24 * time.Hour // Default 7 days
	if teamConfig.Invite.Expiry != "" {
		normalizedDuration, err := normalizeDuration(teamConfig.Invite.Expiry)
		if err != nil {
			log.Warn("Invalid expiry format in team config: %v, using default", err)
		} else {
			duration, err := time.ParseDuration(normalizedDuration)
			if err != nil {
				log.Warn("Failed to parse expiry duration %s: %v, using default", teamConfig.Invite.Expiry, err)
			} else {
				expiryDuration = duration
			}
		}
	}

	// Update invitation with new token and extended expiry
	newExpiryTime := time.Now().Add(expiryDuration)
	updateData := maps.MapStrAny{
		"invitation_token":      newToken,
		"invitation_expires_at": newExpiryTime,
		"invited_at":            time.Now(), // Update invitation time
		"updated_at":            time.Now(),
	}

	// Update invitation using invitation_id
	err = provider.UpdateMemberByInvitationID(ctx, invitationID, updateData)
	if err != nil {
		return fmt.Errorf("failed to update invitation: %w", err)
	}

	// Prepare invitation data for email sending
	invitationData["invitation_token"] = newToken
	invitationData["invitation_expires_at"] = newExpiryTime
	invitationData["request_base_url"] = requestBaseURL

	// Set locale in invitation settings for email template
	if settings, ok := invitationData["settings"].(*InvitationSettings); ok && settings != nil {
		settings.Locale = locale
	} else {
		// Create settings if not exists
		invitationData["settings"] = &InvitationSettings{
			Locale: locale,
		}
	}

	// Send new invitation email (asynchronously)
	go func() {
		// Use background context for async operation
		bgCtx := context.Background()
		err := sendTeamInvitationEmail(bgCtx, inviteeEmail, inviterName, teamName, newToken, invitationID, invitationData)
		if err != nil {
			log.Error("Failed to resend invitation email: %v", err)
		} else {
			log.Info("Invitation email resent to %s for team %s (invitation_id: %s)", inviteeEmail, teamName, invitationID)
		}
	}()

	return nil
}

// teamInvitationDelete handles the business logic for cancelling a team invitation
func teamInvitationDelete(ctx context.Context, userID, teamID, invitationID string) error {
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
	if utils.ToString(invitationData["team_id"]) != teamID {
		return fmt.Errorf("invitation not found in this team")
	}

	// Check if invitation is still pending
	if utils.ToString(invitationData["status"]) != "pending" {
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

// generateTeamInvitationToken generates a secure random token for invitations
func generateTeamInvitationToken() (string, error) {
	bytes := make([]byte, 32) // 32 bytes = 256 bits
	_, err := rand.Read(bytes)
	if err != nil {
		return "", err
	}
	// Use URL-safe base64 encoding and remove padding
	return strings.TrimRight(base64.URLEncoding.EncodeToString(bytes), "="), nil
}

// getTeamInvitationExpiry calculates the expiry duration for an invitation
// Priority: 1. Request expiry parameter, 2. Team config, 3. Default (7 days)
func getTeamInvitationExpiry(invitationData maps.MapStrAny) (time.Duration, error) {
	// Default expiry: 7 days
	defaultExpiry := 7 * 24 * time.Hour

	// Check if expiry is provided in request
	expiry := utils.ToString(invitationData["expiry"])
	if expiry != "" {
		normalizedDuration, err := normalizeDuration(expiry)
		if err != nil {
			return 0, fmt.Errorf("invalid expiry format: %w", err)
		}
		duration, err := time.ParseDuration(normalizedDuration)
		if err != nil {
			return 0, fmt.Errorf("failed to parse expiry duration: %w", err)
		}
		return duration, nil
	}

	// Get team config expiry (from global config)
	// Try to get locale from invitation data settings
	locale := "en"
	if settings, ok := invitationData["settings"].(*InvitationSettings); ok && settings != nil {
		if settings.Locale != "" {
			locale = settings.Locale
		}
	} else if settingsMap, ok := invitationData["settings"].(map[string]interface{}); ok {
		// Fallback for map format (for backward compatibility)
		if loc := utils.ToString(settingsMap["locale"]); loc != "" {
			locale = loc
		}
	}

	teamConfig := GetTeamConfig(locale)
	if teamConfig != nil && teamConfig.Invite != nil && teamConfig.Invite.Expiry != "" {
		normalizedDuration, err := normalizeDuration(teamConfig.Invite.Expiry)
		if err != nil {
			log.Warn("Invalid expiry format in team config: %v, using default", err)
			return defaultExpiry, nil
		}
		duration, err := time.ParseDuration(normalizedDuration)
		if err != nil {
			log.Warn("Failed to parse team config expiry: %v, using default", err)
			return defaultExpiry, nil
		}
		return duration, nil
	}

	return defaultExpiry, nil
}

// sendTeamInvitationEmail sends an invitation email using messenger service
func sendTeamInvitationEmail(ctx context.Context, email, inviterName, teamName, token, invitationID string, invitationData maps.MapStrAny) error {
	// Check if messenger is available
	if messenger.Instance == nil {
		return fmt.Errorf("messenger service not available")
	}

	// Get locale from invitation data settings
	locale := "en"
	if settings, ok := invitationData["settings"].(*InvitationSettings); ok && settings != nil {
		if settings.Locale != "" {
			locale = settings.Locale
		}
	} else if settingsMap, ok := invitationData["settings"].(map[string]interface{}); ok {
		// Fallback for map format (for backward compatibility)
		if loc := utils.ToString(settingsMap["locale"]); loc != "" {
			locale = loc
		}
	}

	// Get team config for email template and channel
	// Note: GetTeamConfig will normalize locale internally (trim, lowercase, etc.)
	teamConfig := GetTeamConfig(locale)
	if teamConfig == nil || teamConfig.Invite == nil {
		return fmt.Errorf("team configuration not found for locale: %s", locale)
	}

	// Get email template ID from team config
	emailTemplate := ""
	if teamConfig.Invite.Templates != nil {
		if tpl, ok := teamConfig.Invite.Templates["mail"]; ok {
			emailTemplate = tpl
		}
	}
	if emailTemplate == "" {
		return fmt.Errorf("email template not configured in team config")
	}

	// Get channel from team config (default to "default")
	channel := "default"
	if teamConfig.Invite.Channel != "" {
		channel = teamConfig.Invite.Channel
	}

	// Get custom message from invitation data
	customMessage := utils.ToString(invitationData["message"])

	// Get request base URL from invitation data (if provided)
	requestBaseURL := utils.ToString(invitationData["request_base_url"])

	// Build invitation link using centralized helper function
	invitationLink := buildTeamInvitationLink(invitationID, token, teamConfig, requestBaseURL)

	// Get time format based on locale
	timeFormat := utils.GetTimeFormat(locale)

	// Format expires_at with locale-specific format
	expiresAtFormatted := utils.FormatTimeWithLocale(invitationData["invitation_expires_at"], timeFormat)

	// Prepare template data for messenger
	templateData := messengertypes.TemplateData{
		"to":              email,
		"inviter_name":    inviterName,
		"team_name":       teamName,
		"invitation_id":   invitationID,
		"invitation_link": invitationLink, // Full invitation link
		"token":           token,          // Keep token for backward compatibility
		"message":         customMessage,
		"role_id":         utils.ToString(invitationData["role_id"]),
		"expires_at":      expiresAtFormatted,
	}

	// Send email using messenger template
	err := messenger.Instance.SendT(ctx, channel, emailTemplate, templateData, messengertypes.MessageTypeEmail)
	if err != nil {
		return fmt.Errorf("failed to send invitation email: %w", err)
	}

	return nil
}

// convertToTeamInvitationResponse converts a map to InvitationResponse (alias for mapToTeamInvitationResponse)
func convertToTeamInvitationResponse(data maps.MapStrAny, requestBaseURL string) InvitationResponse {
	return mapToTeamInvitationResponse(maps.MapStr(data), requestBaseURL)
}

// mapToTeamInvitationResponse converts a map to InvitationResponse
func mapToTeamInvitationResponse(data maps.MapStr, requestBaseURL string) InvitationResponse {
	invitation := InvitationResponse{
		ID:                  utils.ToInt64(data["id"]),
		InvitationID:        utils.ToString(data["invitation_id"]),
		TeamID:              utils.ToString(data["team_id"]),
		UserID:              utils.ToString(data["user_id"]),
		MemberType:          utils.ToString(data["member_type"]),
		RoleID:              utils.ToString(data["role_id"]),
		Status:              utils.ToString(data["status"]),
		InvitedBy:           utils.ToString(data["invited_by"]),
		InvitedAt:           utils.ToTimeString(data["invited_at"]),
		InvitationToken:     utils.ToString(data["invitation_token"]),
		InvitationExpiresAt: utils.ToTimeString(data["invitation_expires_at"]),
		Message:             utils.ToString(data["message"]),
		CreatedAt:           utils.ToTimeString(data["created_at"]),
		UpdatedAt:           utils.ToTimeString(data["updated_at"]),
	}

	// Add settings if available
	locale := "en" // Default locale
	if settings, ok := data["settings"]; ok {
		if invSettings, ok := settings.(*InvitationSettings); ok {
			invitation.Settings = invSettings
			if invSettings.Locale != "" {
				locale = invSettings.Locale
			}
		} else if settingsMap, ok := settings.(map[string]interface{}); ok {
			// Convert map to InvitationSettings
			invSettings := &InvitationSettings{
				SendEmail: utils.ToBool(settingsMap["send_email"]),
				Locale:    utils.ToString(settingsMap["locale"]),
			}
			invitation.Settings = invSettings
			if invSettings.Locale != "" {
				locale = invSettings.Locale
			}
		}
	}

	// Build invitation link if token is available
	if invitation.InvitationToken != "" && invitation.InvitationID != "" {
		teamConfig := GetTeamConfig(locale)
		invitation.InvitationLink = buildTeamInvitationLink(invitation.InvitationID, invitation.InvitationToken, teamConfig, requestBaseURL)
	}

	return invitation
}

// mapToTeamInvitationDetailResponse converts a map to InvitationDetailResponse
func mapToTeamInvitationDetailResponse(data maps.MapStr, requestBaseURL string) InvitationDetailResponse {
	invitation := InvitationDetailResponse{
		InvitationResponse: mapToTeamInvitationResponse(data, requestBaseURL),
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
