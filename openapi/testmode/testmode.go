package testmode

import (
	"context"
	"fmt"
	"net/http"
	"regexp"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/yaoapp/gou/model"
	"github.com/yaoapp/gou/session"
	"github.com/yaoapp/kun/log"
	"github.com/yaoapp/yao/helper"
	"github.com/yaoapp/yao/openapi/oauth"
	userdefs "github.com/yaoapp/yao/openapi/oauth/providers/user"
	"github.com/yaoapp/yao/openapi/otp"
	"github.com/yaoapp/yao/openapi/user"
	"github.com/yaoapp/yao/openapi/utils"
	"github.com/yaoapp/yao/setting"
)

// Attach registers test mode endpoints on the given router group.
// These endpoints are public (no OAuth guard) and should only be
// registered when config.Conf.TestMode is true.
func Attach(group *gin.RouterGroup) {
	group.Use(testModeHeader)
	group.POST("/login/web", handleLoginWeb)
	group.POST("/login/token", handleLoginToken)
	group.POST("/server-key", handleServerKey)
	group.GET("/users", handleListUsers)
	group.GET("/teams", handleListTeams)
	group.GET("/otp", handleOTP)
	group.GET("/captcha", handleCaptcha)
}

// testModeHeader adds X-Test-Mode header to all responses.
func testModeHeader(c *gin.Context) {
	c.Header("X-Test-Mode", "true")
	c.Next()
}

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

// handleOTP verifies an existing OTP code and returns its payload.
// Query params: code (required)
func handleOTP(c *gin.Context) {
	code := c.Query("code")
	if code == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "code query parameter is required"})
		return
	}

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
