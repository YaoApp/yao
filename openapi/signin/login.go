package signin

import (
	"context"

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

	return &LoginResponse{
		AccessToken:  "mock_access_token",
		IDToken:      "mock_id_token",
		RefreshToken: "mock_refresh_token",
		ExpiresIn:    3600,
		TokenType:    "Bearer",
		Scope:        "openid profile email",
		User:         user,
	}, nil
}
