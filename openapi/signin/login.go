package signin

import (
	"context"
	"strings"

	"github.com/yaoapp/kun/log"
	"github.com/yaoapp/yao/openapi/oauth"
	"github.com/yaoapp/yao/openapi/oauth/providers/user"
	oauthtypes "github.com/yaoapp/yao/openapi/oauth/types"
)

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

	var scopes []string
	if v, ok := user["scopes"].([]string); ok {
		scopes = v
	}

	// Get Config form app.yao config ()
	clientID := "1234567890"
	oidcExpiresIn := 3600
	accessTokenExpiresIn := 3600

	subject, err := oauth.OAuth.Subject(clientID, userid)
	if err != nil {
		log.Warn("Failed to store user fingerprint: %s", err.Error())
	}
	oidcUserInfo := oauthtypes.MakeOIDCUserInfo(user)
	oidcUserInfo.Sub = subject

	// OIDC Token
	oidcToken, err := oauth.OAuth.SignIDToken(clientID, strings.Join(scopes, " "), oidcExpiresIn, oidcUserInfo)
	if err != nil {
		return nil, err
	}

	// Access Token
	accessToken, err := oauth.OAuth.MakeAccessToken(clientID, strings.Join(scopes, " "), subject, accessTokenExpiresIn)
	if err != nil {
		return nil, err
	}

	// Refresh Token
	refreshToken, err := oauth.OAuth.MakeRefreshToken(clientID, strings.Join(scopes, " "), subject)
	if err != nil {
		return nil, err
	}
	return &LoginResponse{
		AccessToken:  accessToken,
		IDToken:      oidcToken,
		RefreshToken: refreshToken,
		ExpiresIn:    accessTokenExpiresIn,
		TokenType:    "Bearer",
		Scope:        strings.Join(scopes, " "),
	}, nil
}
