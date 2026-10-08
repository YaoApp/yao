package testmode

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net/http"
	"regexp"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/yaoapp/gou/model"
	"github.com/yaoapp/gou/session"
	"github.com/yaoapp/kun/log"
	"github.com/yaoapp/kun/maps"
	"github.com/yaoapp/yao/helper"
	"github.com/yaoapp/yao/openapi/oauth"
	userdefs "github.com/yaoapp/yao/openapi/oauth/providers/user"
	"github.com/yaoapp/yao/openapi/oauth/types"
	"github.com/yaoapp/yao/openapi/otp"
	"github.com/yaoapp/yao/openapi/user"
	"github.com/yaoapp/yao/openapi/utils"
	"github.com/yaoapp/yao/setting"
	utilsotp "github.com/yaoapp/yao/utils/otp"
)

// validUserStatuses lists all accepted values for user status endpoints.
var validUserStatuses = map[string]bool{
	types.UserStatusPending:         true,
	types.UserStatusActive:          true,
	types.UserStatusDisabled:        true,
	types.UserStatusSuspended:       true,
	types.UserStatusLocked:          true,
	types.UserStatusPasswordExpired: true,
	types.UserStatusEmailUnverified: true,
	types.UserStatusArchived:        true,
	"pending_invite":                true,
}

// Attach registers test mode endpoints on the given router group.
// These endpoints are public (no OAuth guard) and should only be
// registered when config.Conf.TestMode is true.
func Attach(group *gin.RouterGroup) {
	group.Use(testModeHeader)

	// Existing endpoints
	group.POST("/login/web", handleLoginWeb)
	group.POST("/login/token", handleLoginToken)
	group.POST("/server-key", handleServerKey)
	group.GET("/users", handleListUsers)
	group.GET("/teams", handleListTeams)
	group.GET("/otp", handleOTP)
	group.GET("/captcha", handleCaptcha)

	// New endpoints
	group.POST("/users", handleCreateUser)
	group.DELETE("/users/:id", handleDeleteUser)
	group.POST("/users/:id/status", handleUpdateUserStatus)
	group.POST("/invite", handleCreateInvite)
	group.POST("/oauth/device/approve", handleDeviceApprove)
	group.POST("/entry/error", handleEntryError)
}

// testModeHeader adds X-Test-Mode header to all responses.
func testModeHeader(c *gin.Context) {
	c.Header("X-Test-Mode", "true")
	c.Next()
}

// ===================== Login endpoints =====================

// loginRequest is the request body for login endpoints.
type loginRequest struct {
	User   string `json:"user" binding:"required"` // email, phone, or user_id
	TeamID string `json:"team_id,omitempty"`       // optional team ID
}

// handleLoginWeb issues a full web login (cookies + JSON) for the given user.
func handleLoginWeb(c *gin.Context) {
	var req loginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	userID, err := resolveUserID(c.Request.Context(), req.User)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "user not found"})
		return
	}

	log.Info("[TestMode] login/web user_id=%s team_id=%s remote=%s", userID, req.TeamID, c.Request.RemoteAddr)

	loginResp, err := doLogin(userID, req.TeamID)
	if err != nil {
		log.Error("[TestMode] login/web failed: %s", err.Error())
		c.JSON(http.StatusInternalServerError, gin.H{"error": "login failed"})
		return
	}

	sid := session.ID()
	user.SendLoginCookies(c, loginResp, sid)

	c.JSON(http.StatusOK, gin.H{
		"session_id":               sid,
		"id_token":                 loginResp.IDToken,
		"access_token":             loginResp.AccessToken,
		"refresh_token":            loginResp.RefreshToken,
		"expires_in":               loginResp.ExpiresIn,
		"refresh_token_expires_in": loginResp.RefreshTokenExpiresIn,
		"status":                   loginResp.Status,
	})
}

// handleLoginToken issues tokens for mobile/client apps (no cookies).
func handleLoginToken(c *gin.Context) {
	var req loginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	userID, err := resolveUserID(c.Request.Context(), req.User)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "user not found"})
		return
	}

	log.Info("[TestMode] login/token user_id=%s team_id=%s remote=%s", userID, req.TeamID, c.Request.RemoteAddr)

	loginResp, err := doLogin(userID, req.TeamID)
	if err != nil {
		log.Error("[TestMode] login/token failed: %s", err.Error())
		c.JSON(http.StatusInternalServerError, gin.H{"error": "login failed"})
		return
	}

	sid := session.ID()

	c.JSON(http.StatusOK, gin.H{
		"id_token":                 loginResp.IDToken,
		"access_token":             loginResp.AccessToken,
		"refresh_token":            loginResp.RefreshToken,
		"expires_in":               loginResp.ExpiresIn,
		"refresh_token_expires_in": loginResp.RefreshTokenExpiresIn,
		"session_id":               sid,
		"status":                   loginResp.Status,
	})
}

// ===================== Server key endpoint =====================

// serverKeyRequest is the request body for server key creation.
type serverKeyRequest struct {
	Name string `json:"name,omitempty"`
	TTL  string `json:"ttl,omitempty"`
}

// handleServerKey creates a new server key.
func handleServerKey(c *gin.Context) {
	var req serverKeyRequest
	c.ShouldBindJSON(&req)

	if req.Name == "" {
		req.Name = "test-node"
	}

	var ttlDuration time.Duration
	var opts []time.Duration
	if req.TTL != "" {
		d, err := time.ParseDuration(req.TTL)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid ttl format"})
			return
		}
		ttlDuration = d
		opts = append(opts, d)
	}

	log.Info("[TestMode] server-key create name=%s ttl=%s remote=%s", req.Name, req.TTL, c.Request.RemoteAddr)

	plainKey, keyID, err := setting.CreateServerKey(req.Name, opts...)
	if err != nil {
		log.Error("[TestMode] server-key create failed: %s", err.Error())
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create server key"})
		return
	}

	if setting.Global != nil {
		setting.Global.Flush()
	}

	resp := gin.H{
		"key":    plainKey,
		"key_id": keyID,
		"name":   req.Name,
	}
	if req.TTL != "" {
		resp["expires_at"] = time.Now().UTC().Add(ttlDuration).Format(time.RFC3339)
	}

	c.JSON(http.StatusOK, resp)
}

// ===================== List endpoints =====================

// handleListUsers returns a paginated list of users.
func handleListUsers(c *gin.Context) {
	userProvider, err := oauth.OAuth.GetUserProvider()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "user provider not available"})
		return
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pagesize, _ := strconv.Atoi(c.DefaultQuery("pagesize", "20"))
	if page < 1 {
		page = 1
	}
	if pagesize < 1 || pagesize > 100 {
		pagesize = 20
	}

	param := model.QueryParam{
		Select: userdefs.DefaultBasicUserFields,
	}

	if status := c.Query("status"); status != "" {
		param.Wheres = append(param.Wheres, model.QueryWhere{
			Column: "status",
			Value:  status,
		})
	}

	result, err := userProvider.PaginateUsers(c.Request.Context(), param, page, pagesize)
	if err != nil {
		log.Error("[TestMode] list users failed: %s", err.Error())
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list users"})
		return
	}

	c.JSON(http.StatusOK, result)
}

// handleListTeams returns a list of teams.
func handleListTeams(c *gin.Context) {
	userProvider, err := oauth.OAuth.GetUserProvider()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "user provider not available"})
		return
	}

	ctx := c.Request.Context()
	userID := c.Query("user_id")

	if userID != "" {
		teams, err := userProvider.GetUserTeams(ctx, userID)
		if err != nil {
			log.Error("[TestMode] list teams for user failed: %s", err.Error())
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list teams"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"data": teams})
		return
	}

	param := model.QueryParam{
		Select: userdefs.DefaultTeamFields,
	}
	teams, err := userProvider.GetTeams(ctx, param)
	if err != nil {
		log.Error("[TestMode] list teams failed: %s", err.Error())
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list teams"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": teams})
}

// ===================== OTP & Captcha query endpoints =====================

// handleOTP queries OTP by entry otp_id or by magic-link code.
// Query params: id (entry OTP) or code (magic-link OTP); id takes priority.
func handleOTP(c *gin.Context) {
	// Entry OTP: utils/otp in-memory store, keyed by UUID otp_id
	if id := c.Query("id"); id != "" {
		code := utilsotp.Get(id)
		if code == "" {
			c.JSON(http.StatusNotFound, gin.H{"error": "OTP not found or expired"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"id": id, "code": code})
		return
	}

	// Magic-link OTP: openapi/otp persistent store, keyed by code
	if code := c.Query("code"); code != "" {
		if otp.OTP == nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "OTP service not initialized"})
			return
		}
		payload, err := otp.OTP.Verify(code)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "OTP code not found or expired"})
			return
		}
		c.JSON(http.StatusOK, gin.H{
			"code":      code,
			"user_id":   payload.UserID,
			"team_id":   payload.TeamID,
			"member_id": payload.MemberID,
			"redirect":  payload.Redirect,
			"scope":     payload.Scope,
			"consume":   payload.Consume,
		})
		return
	}

	c.JSON(http.StatusBadRequest, gin.H{"error": "id or code query parameter is required"})
}

// handleCaptcha returns the answer for an existing captcha by ID.
// Query params: id (required)
func handleCaptcha(c *gin.Context) {
	id := c.Query("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "id query parameter is required"})
		return
	}

	answer := helper.CaptchaGet(id)
	if answer == "" {
		c.JSON(http.StatusNotFound, gin.H{"error": "captcha not found or expired"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"id":     id,
		"answer": answer,
	})
}

// ===================== User CRUD endpoints =====================

// createUserRequest is the request body for creating a test user.
type createUserRequest struct {
	Email       string `json:"email" binding:"required"`
	Password    string `json:"password" binding:"required"`
	Name        string `json:"name,omitempty"`
	PhoneNumber string `json:"phone_number,omitempty"`
	Status      string `json:"status,omitempty"` // default: "active"
}

// handleCreateUser creates a test user via the user provider.
func handleCreateUser(c *gin.Context) {
	var req createUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	userProvider, err := oauth.OAuth.GetUserProvider()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "user provider not available"})
		return
	}

	status := req.Status
	if status == "" {
		status = types.UserStatusActive
	}
	if !validUserStatuses[status] {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("invalid status %q", status)})
		return
	}

	ctx := c.Request.Context()

	userData := maps.MapStrAny{
		"email":    utils.NormalizeEmail(req.Email),
		"password": req.Password,
		"status":   status,
	}
	if req.Name != "" {
		userData["name"] = req.Name
	}
	if req.PhoneNumber != "" {
		userData["phone_number"] = req.PhoneNumber
	}

	log.Info("[TestMode] create user email=%s status=%s remote=%s", req.Email, status, c.Request.RemoteAddr)

	userID, err := userProvider.CreateUser(ctx, userData)
	if err != nil {
		log.Error("[TestMode] create user failed: %s", err.Error())
		c.JSON(http.StatusConflict, gin.H{"error": "failed to create user"})
		return
	}

	// Return the created user
	u, err := userProvider.GetUser(ctx, userID)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"user_id": userID})
		return
	}

	c.JSON(http.StatusOK, u)
}

// handleDeleteUser deletes a test user.
// Query params: force=true for hard delete (permanent), default is soft delete.
func handleDeleteUser(c *gin.Context) {
	userID := c.Param("id")
	if userID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "user id is required"})
		return
	}

	force := c.Query("force") == "true"

	if force {
		log.Info("[TestMode] hard delete user user_id=%s remote=%s", userID, c.Request.RemoteAddr)
		m := model.Select("__yao.user")
		affected, err := m.DestroyWhere(model.QueryParam{
			Wheres: []model.QueryWhere{
				{Column: "user_id", Value: userID},
			},
			Limit: 1,
		})
		if err != nil {
			log.Error("[TestMode] hard delete user failed: %s", err.Error())
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete user"})
			return
		}
		if affected == 0 {
			c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"deleted": true, "user_id": userID, "hard": true})
		return
	}

	userProvider, err := oauth.OAuth.GetUserProvider()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "user provider not available"})
		return
	}

	log.Info("[TestMode] delete user user_id=%s remote=%s", userID, c.Request.RemoteAddr)

	if err := userProvider.DeleteUser(c.Request.Context(), userID); err != nil {
		log.Error("[TestMode] delete user failed: %s", err.Error())
		c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"deleted": true, "user_id": userID})
}

// ===================== User status endpoint =====================

// updateStatusRequest is the request body for updating user status.
type updateStatusRequest struct {
	Status string `json:"status" binding:"required"`
}

// handleUpdateUserStatus updates a user's status.
func handleUpdateUserStatus(c *gin.Context) {
	userID := c.Param("id")
	if userID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "user id is required"})
		return
	}

	var req updateStatusRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	if !validUserStatuses[req.Status] {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("invalid status %q", req.Status)})
		return
	}

	userProvider, err := oauth.OAuth.GetUserProvider()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "user provider not available"})
		return
	}

	ctx := c.Request.Context()

	log.Info("[TestMode] update status user_id=%s status=%s remote=%s", userID, req.Status, c.Request.RemoteAddr)

	if err := userProvider.UpdateUserStatus(ctx, userID, req.Status); err != nil {
		log.Error("[TestMode] update status failed: %s", err.Error())
		c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
		return
	}

	u, err := userProvider.GetUser(ctx, userID)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"user_id": userID, "status": req.Status})
		return
	}

	c.JSON(http.StatusOK, u)
}

// ===================== Invitation code endpoint =====================

// handleCreateInvite creates a usable invitation code.
func handleCreateInvite(c *gin.Context) {
	userProvider, err := oauth.OAuth.GetUserProvider()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "user provider not available"})
		return
	}

	// Generate a random invitation code
	b := make([]byte, 12)
	if _, err := rand.Read(b); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate code"})
		return
	}
	code := "test-" + hex.EncodeToString(b)

	ctx := c.Request.Context()

	var expiresAt string
	var req struct {
		ExpiresIn int `json:"expires_in,omitempty"` // seconds
	}
	c.ShouldBindJSON(&req)
	if req.ExpiresIn > 0 {
		expiresAt = time.Now().UTC().Add(time.Duration(req.ExpiresIn) * time.Second).Format(time.RFC3339)
	}

	codeData := maps.MapStrAny{
		"code":         code,
		"status":       "active",
		"is_published": true,
		"code_type":    "official",
		"source":       "test-mode",
		"description":  "Test mode invitation code",
	}
	if expiresAt != "" {
		codeData["expires_at"] = expiresAt
	}

	log.Info("[TestMode] create invite code=%s remote=%s", code, c.Request.RemoteAddr)

	_, err = userProvider.CreateInvitationCodes(ctx, []maps.MapStrAny{codeData})
	if err != nil {
		log.Error("[TestMode] create invite failed: %s", err.Error())
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create invitation code"})
		return
	}

	resp := gin.H{"code": code}
	if expiresAt != "" {
		resp["expires_at"] = expiresAt
	}
	c.JSON(http.StatusOK, resp)
}

// ===================== Device code approval endpoint =====================

// deviceApproveRequest is the request body for device code approval.
type deviceApproveRequest struct {
	UserCode string `json:"user_code" binding:"required"`
	User     string `json:"user" binding:"required"` // email, phone, or user_id
}

// handleDeviceApprove approves a pending device authorization code.
func handleDeviceApprove(c *gin.Context) {
	var req deviceApproveRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	userID, err := resolveUserID(c.Request.Context(), req.User)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "user not found"})
		return
	}

	if oauth.OAuth == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "OAuth service not available"})
		return
	}

	log.Info("[TestMode] device/approve user_code=%s user_id=%s remote=%s", req.UserCode, userID, c.Request.RemoteAddr)

	if err := oauth.OAuth.AuthorizeDevice(c.Request.Context(), req.UserCode, userID); err != nil {
		log.Error("[TestMode] device/approve failed: %s", err.Error())
		c.JSON(http.StatusBadRequest, gin.H{"error": "failed to approve device code"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"approved": true, "user_code": req.UserCode})
}

// ===================== Entry error echo endpoint =====================

// handleEntryError echoes back the given error as an HTTP response.
// CUI tests call this endpoint to verify client-side error handling UI
// without modifying production entry handlers.
func handleEntryError(c *gin.Context) {
	var req struct {
		StatusCode       int    `json:"status_code" binding:"required"`
		Error            string `json:"error" binding:"required"`
		ErrorDescription string `json:"error_description,omitempty"`
		Reason           string `json:"reason,omitempty"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	resp := gin.H{"error": req.Error}
	if req.ErrorDescription != "" {
		resp["error_description"] = req.ErrorDescription
	}
	if req.Reason != "" {
		resp["reason"] = req.Reason
	}
	c.JSON(req.StatusCode, resp)
}

// ===================== Shared helpers =====================

// resolveUserID resolves an email, phone number, or user_id to a user_id.
func resolveUserID(ctx context.Context, input string) (string, error) {
	userProvider, err := oauth.OAuth.GetUserProvider()
	if err != nil {
		return "", fmt.Errorf("user provider not available: %w", err)
	}

	// Email — uses publicUserFields (safe, no password_hash)
	if isEmail(input) {
		input = utils.NormalizeEmail(input)
		u, err := userProvider.GetUserByEmail(ctx, input)
		if err != nil {
			return "", fmt.Errorf("user not found")
		}
		if id, ok := u["user_id"].(string); ok && id != "" {
			return id, nil
		}
		return "", fmt.Errorf("user not found")
	}

	// Phone number — query with explicit safe fields to avoid loading password_hash
	if isPhone(input) {
		users, err := userProvider.GetUsers(ctx, model.QueryParam{
			Select: []interface{}{"user_id"},
			Wheres: []model.QueryWhere{
				{Column: "phone_number", Value: input},
			},
			Limit: 1,
		})
		if err != nil || len(users) == 0 {
			return "", fmt.Errorf("user not found")
		}
		if id, ok := users[0]["user_id"].(string); ok && id != "" {
			return id, nil
		}
		return "", fmt.Errorf("user not found")
	}

	// Treat as user_id directly
	return input, nil
}

// doLogin calls the standard login flow.
func doLogin(userID, teamID string) (*user.LoginResponse, error) {
	if teamID != "" {
		return user.LoginByTeamID(userID, teamID, nil)
	}
	return user.LoginByUserID(userID, nil)
}

var emailRegex = regexp.MustCompile(`^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}$`)
var phoneRegex = regexp.MustCompile(`^\+?[0-9]{10,15}$`)

func isEmail(s string) bool {
	return emailRegex.MatchString(s)
}

func isPhone(s string) bool {
	return phoneRegex.MatchString(s)
}
