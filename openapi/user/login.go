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
func LoginThirdParty(providerID string, userinfo *oauthtypes.OIDCUserInfo, ip string) (*LoginResponse, error) {

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

	return LoginByUserID(userID, ip)
}

// LoginByUserID is the handler for login
func LoginByUserID(userid string, ip string) (*LoginResponse, error) {

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

	// Update Last Login
	err = userProvider.UpdateUserLastLogin(ctx, userid, ip)
	if err != nil {
		log.Warn("Failed to update last login: %s", err.Error())
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
	oidcUserInfo := oauthtypes.MakeOIDCUserInfo(user)
	oidcUserInfo.Sub = subject

	// OIDC Token
	oidcToken, err := oauth.OAuth.SignIDToken(yaoClientConfig.ClientID, strings.Join(scopes, " "), yaoClientConfig.ExpiresIn, oidcUserInfo)
	if err != nil {
		return nil, err
	}

	// Access Token
	accessToken, err := oauth.OAuth.MakeAccessToken(yaoClientConfig.ClientID, strings.Join(scopes, " "), subject, yaoClientConfig.ExpiresIn)
	if err != nil {
		return nil, err
	}

	// Refresh Token
	refreshToken, err := oauth.OAuth.MakeRefreshToken(yaoClientConfig.ClientID, strings.Join(scopes, " "), subject, yaoClientConfig.RefreshTokenExpiresIn)
	if err != nil {
		return nil, err
	}

	// Get MFA enabled status from user data
	mfaEnabled := toBool(user["mfa_enabled"])

	return &LoginResponse{
		AccessToken:           accessToken,
		IDToken:               oidcToken,
		RefreshToken:          refreshToken,
		ExpiresIn:             yaoClientConfig.ExpiresIn,
		RefreshTokenExpiresIn: yaoClientConfig.RefreshTokenExpiresIn,
		TokenType:             "Bearer",
		MFAEnabled:            mfaEnabled,
		Scope:                 strings.Join(scopes, " "),
	}, nil
}

// generateSessionID generates a session ID
func generateSessionID() string {
	return session.ID()
}

// SendLoginCookies sends all necessary cookies for a successful login
// This includes access token, refresh token, and session ID cookies with appropriate security settings
func SendLoginCookies(c *gin.Context, loginResponse *LoginResponse, sessionID string) {
	// Format tokens with Bearer prefix
	accessToken := fmt.Sprintf("%s %s", loginResponse.TokenType, loginResponse.AccessToken)
	refreshToken := fmt.Sprintf("%s %s", loginResponse.TokenType, loginResponse.RefreshToken)

	// Calculate expiration times
	expires := time.Now().Add(time.Duration(loginResponse.ExpiresIn) * time.Second)
	refreshExpires := time.Now().Add(time.Duration(loginResponse.RefreshTokenExpiresIn) * time.Second)

	// Send access token cookie
	response.SendAccessTokenCookieWithExpiry(c, accessToken, expires)

	// Send refresh token cookie
	response.SendRefreshTokenCookieWithExpiry(c, refreshToken, refreshExpires)

	// Send session ID cookie with the same expiration as access token
	// Using HTTP-only flag for security
	options := response.NewSecureCookieOptions().
		WithExpires(expires).
		WithSameSite("Strict")
	response.SendSecureCookieWithOptions(c, "session_id", sessionID, options)
}
