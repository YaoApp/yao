package user

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/yaoapp/gou/session"
	"github.com/yaoapp/kun/log"
	"github.com/yaoapp/yao/helper"
	"github.com/yaoapp/yao/openapi/oauth"
	"github.com/yaoapp/yao/openapi/oauth/providers/user"
	oauthtypes "github.com/yaoapp/yao/openapi/oauth/types"
	"github.com/yaoapp/yao/openapi/response"
	"github.com/yaoapp/yao/openapi/utils"
)

// getLoginConfig is the handler for get login configuration (mapped from /signin)
func getLoginConfig(c *gin.Context) {
	// Get locale from query parameter (optional)
	locale := c.Query("locale")

	// Get public configuration for the specified locale
	config := GetPublicConfig(locale)

	// Set session id if not exists
	sid := utils.GetSessionID(c)
	if sid == "" {
		sid = generateSessionID()
		response.SendSessionCookie(c, sid)
	}

	// If no configuration found, return error
	if config == nil {
		errorResp := &response.ErrorResponse{
			Code:             response.ErrInvalidRequest.Code,
			ErrorDescription: "No signin configuration found for the requested locale",
		}
		response.RespondWithError(c, response.StatusNotFound, errorResp)
		return
	}

	// Return the public configuration
	response.RespondWithSuccess(c, response.StatusOK, config)
}

// login is the handler for login (password login, mapped from /signin)
func login(c *gin.Context) {
	// This is a placeholder - the original signin function was empty
	// You may need to implement the actual login logic here
}

// getCaptcha is the handler for get captcha image for login
func getCaptcha(c *gin.Context) {
	var option helper.CaptchaOption = helper.NewCaptchaOption()

	err := c.ShouldBindQuery(&option)
	if err != nil {
		response.RespondWithError(c, http.StatusBadRequest, &response.ErrorResponse{
			Code:             response.ErrInvalidRequest.Code,
			ErrorDescription: err.Error(),
		})
		return
	}

	// Set the type to image
	option.Type = "image"
	id, content := helper.CaptchaMake(option)

	// Return in the format expected by the frontend
	response.RespondWithSuccess(c, http.StatusOK, gin.H{
		"captcha_id":    id,
		"captcha_image": content,
		"expires_in":    300, // 5 minutes
	})
}

// LoginThirdParty is the handler for third party login
func LoginThirdParty(providerID string, userinfo *oauthtypes.OIDCUserInfo, loginCtx *LoginContext) (*LoginResponse, error) {

	// Get provider
	provider, err := GetProvider(providerID)
	if err != nil {
		return nil, err
	}

	// Check if user exists
	userProvider, err := oauth.OAuth.GetUserProvider()
	if err != nil {
		return nil, err
	}

	// Auto register user if not exists
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var userID string

	// Auto register user if not exists
	if provider.Register != nil && provider.Register.Auto {
		userID, err = userProvider.GetOAuthUserID(ctx, providerID, userinfo.Sub)
		if err != nil && err.Error() == user.ErrOAuthAccountNotFound {

			userData := map[string]interface{}{
				"name":        userinfo.Name,
				"given_name":  userinfo.GivenName,
				"family_name": userinfo.FamilyName,
				"picture":     userinfo.Picture,
				"role_id":     provider.Register.Role,
				"type_id":     provider.Register.Type,
				"status":      "active",
			}

			// Auto register user
			userID, err = userProvider.CreateUser(ctx, userData)
			if err != nil {
				return nil, err
			}

			// Create OAuth account
			userData = userinfo.Map()
			userData["provider"] = providerID
			_, err = userProvider.CreateOAuthAccount(ctx, userID, userData)
			if err != nil {
				return nil, err
			}
		}
	}

	// Get User ID from OAuth account
	userID, err = userProvider.GetOAuthUserID(ctx, providerID, userinfo.Sub)
	if err != nil {
		return nil, err
	}

	return LoginByUserID(userID, loginCtx)
}

// LoginByUserID is the handler for login by user ID
func LoginByUserID(userid string, loginCtx *LoginContext) (*LoginResponse, error) {
	// Get User
	userProvider, err := oauth.OAuth.GetUserProvider()
	if err != nil {
		return nil, err
	}

	// Get User
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	user, err := userProvider.GetUserWithScopes(ctx, userid)
	if err != nil {
		return nil, err
	}

	yaoClientConfig := GetYaoClientConfig()
	var scopes []string = yaoClientConfig.Scopes
	if v, ok := user["scopes"].([]string); ok {
		scopes = v
	}

	subject, err := oauth.OAuth.Subject(yaoClientConfig.ClientID, userid)
	if err != nil {
		log.Warn("Failed to store user fingerprint: %s", err.Error())
	}

	// Get MFA enabled status from user data
	mfaEnabled := toBool(user["mfa_enabled"])

	// If MFA enabled, generate MFA token
	if mfaEnabled {
		// Sign temporary access token for MFA
		var mfaExpire int = 10 * 60 // 10 minutes
		accessToken, err := oauth.OAuth.MakeAccessToken(yaoClientConfig.ClientID, ScopeMFAVerification, subject, mfaExpire)
		if err != nil {
			return nil, err
		}

		return &LoginResponse{
			UserID:      userid,
			AccessToken: accessToken,
			ExpiresIn:   mfaExpire,
			MFAEnabled:  mfaEnabled,
			TokenType:   "Bearer",
			Scope:       ScopeMFAVerification,
			Status:      LoginStatusMFA,
		}, nil
	}

	// Update Last Login
	if loginCtx != nil {
		err = userProvider.UpdateUserLastLogin(ctx, userid, loginCtx)
		if err != nil {
			log.Warn("Failed to update last login: %s", err.Error())
		}
	}

	// Count User Teams
	numTeams, err := getUserTeamsCount(ctx, userid)
	if err != nil {
		return nil, err
	}

	// If user has teams, return team selection status with temporary access token
	if numTeams > 0 {
		// Sign temporary access token for Team Selection
		var teamSelectionExpire int = 10 * 60 // 10 minutes
		accessToken, err := oauth.OAuth.MakeAccessToken(yaoClientConfig.ClientID, ScopeTeamSelection, subject, teamSelectionExpire)
		if err != nil {
			return nil, err
		}

		return &LoginResponse{
			UserID:      userid,
			Subject:     subject,
			AccessToken: accessToken,
			ExpiresIn:   teamSelectionExpire,
			MFAEnabled:  mfaEnabled,
			TokenType:   "Bearer",
			Scope:       ScopeTeamSelection,
			Status:      LoginStatusTeamSelection,
		}, nil
	}

	// Issue tokens without team context
	return issueTokens(ctx, userid, "", nil, user, subject, scopes)
}

// LoginByTeamID is the handler for login by team ID (after team selection)
func LoginByTeamID(userid string, teamID string, loginCtx *LoginContext) (*LoginResponse, error) {
	// Get User
	userProvider, err := oauth.OAuth.GetUserProvider()
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Get user data with scopes
	user, err := userProvider.GetUserWithScopes(ctx, userid)
	if err != nil {
		return nil, err
	}

	yaoClientConfig := GetYaoClientConfig()
	var scopes []string = yaoClientConfig.Scopes
	if v, ok := user["scopes"].([]string); ok {
		scopes = v
	}

	// Get or create subject
	subject, err := oauth.OAuth.Subject(yaoClientConfig.ClientID, userid)
	if err != nil {
		log.Warn("Failed to store user fingerprint: %s", err.Error())
	}

	// Handle personal account (no team)
	if teamID == "" || teamID == "personal" {
		return issueTokens(ctx, userid, "", nil, user, subject, scopes)
	}

	// Verify user is a member of the team and get team details
	team, err := userProvider.GetTeamByMember(ctx, teamID, userid)
	if err != nil {
		return nil, fmt.Errorf("access denied: you are not a member of this team")
	}

	// Update Last Login
	if loginCtx != nil {
		err = userProvider.UpdateUserLastLogin(ctx, userid, loginCtx)
		if err != nil {
			log.Warn("Failed to update last login: %s", err.Error())
		}
	}

	// Issue tokens with team context
	return issueTokens(ctx, userid, teamID, team, user, subject, scopes)
}

// issueTokens is the core function that issues all necessary tokens (ID token, access token, refresh token)
func issueTokens(ctx context.Context, userid string, teamID string, team map[string]interface{}, user map[string]interface{}, subject string, scopes []string) (*LoginResponse, error) {
	yaoClientConfig := GetYaoClientConfig()

	// Prepare OIDC user info
	oidcUserInfo := oauthtypes.MakeOIDCUserInfo(user)
	oidcUserInfo.Sub = subject

	// Prepare extra claims for team context
	var extraClaims map[string]interface{}
	if teamID != "" && team != nil {
		extraClaims = map[string]interface{}{
			"team_id": teamID,
		}

		// Add tenant_id if available from the team
		if tenantID := toString(team["tenant_id"]); tenantID != "" {
			extraClaims["tenant_id"] = tenantID
			oidcUserInfo.YaoTenantID = tenantID
		}

		// Add team info to OIDC user info
		oidcUserInfo.YaoTeamID = teamID
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

		// Add owner_id if available from the team (only check once)
		if ownerID := toString(team["owner_id"]); ownerID != "" {
			extraClaims["owner_id"] = ownerID
			teamInfo.OwnerID = ownerID

			// Check if user is owner
			if ownerID == userid {
				isOwner := true
				oidcUserInfo.YaoIsOwner = &isOwner
			}
		}

		oidcUserInfo.YaoTeam = teamInfo
	}

	// Sign OIDC Token
	var oidcToken string
	var err error
	if extraClaims != nil {
		oidcToken, err = oauth.OAuth.SignIDToken(yaoClientConfig.ClientID, strings.Join(scopes, " "), yaoClientConfig.ExpiresIn, oidcUserInfo, extraClaims)
	} else {
		oidcToken, err = oauth.OAuth.SignIDToken(yaoClientConfig.ClientID, strings.Join(scopes, " "), yaoClientConfig.ExpiresIn, oidcUserInfo)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to sign OIDC token: %w", err)
	}

	// Sign Access Token
	var accessToken string
	if extraClaims != nil {
		accessToken, err = oauth.OAuth.MakeAccessToken(yaoClientConfig.ClientID, strings.Join(scopes, " "), subject, yaoClientConfig.ExpiresIn, extraClaims)
	} else {
		accessToken, err = oauth.OAuth.MakeAccessToken(yaoClientConfig.ClientID, strings.Join(scopes, " "), subject, yaoClientConfig.ExpiresIn)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to sign access token: %w", err)
	}

	// Sign Refresh Token
	var refreshToken string
	if extraClaims != nil {
		refreshToken, err = oauth.OAuth.MakeRefreshToken(yaoClientConfig.ClientID, strings.Join(scopes, " "), subject, yaoClientConfig.RefreshTokenExpiresIn, extraClaims)
	} else {
		refreshToken, err = oauth.OAuth.MakeRefreshToken(yaoClientConfig.ClientID, strings.Join(scopes, " "), subject, yaoClientConfig.RefreshTokenExpiresIn)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to sign refresh token: %w", err)
	}

	return &LoginResponse{
		UserID:                userid,
		Subject:               subject,
		AccessToken:           accessToken,
		IDToken:               oidcToken,
		RefreshToken:          refreshToken,
		ExpiresIn:             yaoClientConfig.ExpiresIn,
		RefreshTokenExpiresIn: yaoClientConfig.RefreshTokenExpiresIn,
		TokenType:             "Bearer",
		MFAEnabled:            toBool(user["mfa_enabled"]),
		Scope:                 strings.Join(scopes, " "),
		Status:                LoginStatusSuccess,
	}, nil
}

// generateSessionID generates a session ID
func generateSessionID() string {
	return session.ID()
}

// SendLoginCookies sends all necessary cookies for a successful login
// This includes access token, refresh token, and optionally session ID cookies with appropriate security settings
func SendLoginCookies(c *gin.Context, loginResponse *LoginResponse, sessionID string) {

	// Send session ID cookie only if sessionID is provided
	if sessionID != "" {
		expires := time.Now().Add(time.Duration(yaoClientConfig.ExpiresIn) * time.Second)
		options := response.NewSecureCookieOptions().
			WithExpires(expires).
			WithSameSite("Strict")
		response.SendSecureCookieWithOptions(c, "session_id", sessionID, options)
	}

	// MFA Temporary Access Token
	if loginResponse.Status == LoginStatusMFA {
		mfaToken := fmt.Sprintf("Bearer %s", loginResponse.AccessToken)
		expires := time.Now().Add(time.Duration(loginResponse.ExpiresIn) * time.Second)
		response.SendAccessTokenCookieWithExpiry(c, mfaToken, expires)
		return
	}

	// Normal Access Token
	accessToken := fmt.Sprintf("%s %s", loginResponse.TokenType, loginResponse.AccessToken)
	refreshToken := fmt.Sprintf("%s %s", loginResponse.TokenType, loginResponse.RefreshToken)

	// Calculate expiration times
	refreshExpires := time.Now().Add(time.Duration(loginResponse.RefreshTokenExpiresIn) * time.Second)

	// Send access token cookie
	response.SendAccessTokenCookieWithExpiry(c, accessToken, time.Now().Add(time.Duration(loginResponse.ExpiresIn)*time.Second))

	// Send refresh token cookie
	response.SendRefreshTokenCookieWithExpiry(c, refreshToken, refreshExpires)
}
