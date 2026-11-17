package oauth

import (
	"context"

	"github.com/yaoapp/yao/openapi/oauth/types"
)

// UserInfo returns user information for a given access token
func (s *Service) UserInfo(ctx context.Context, accessToken string) (interface{}, error) {
	return s.userProvider.GetUser(ctx, accessToken)
}

// BuildUserInfo builds the OAuth/OIDC userinfo response from verified access
// token claims.
//
// The response exposes the standard subject and the Yao identity extensions
// auth_source / identity_source. Team/Tenant claims are intentionally never
// returned: Team is a Yao application concept, not part of OAuth identity.
func (s *Service) BuildUserInfo(ctx context.Context, claims *types.TokenClaims) *types.OIDCUserInfo {
	if claims == nil {
		return nil
	}

	userID := ""
	if claims.ClientID != "" && claims.Subject != "" {
		if resolved, err := s.UserID(claims.ClientID, claims.Subject); err == nil {
			userID = resolved
		}
	}

	authSource := extraClaimString(claims, types.ClaimAuthSource)
	identitySource := extraClaimString(claims, types.ClaimIdentitySource)
	if identitySource == "" {
		identitySource = s.resolveIdentitySource(ctx, userID, authSource)
	}

	info := &types.OIDCUserInfo{
		Sub:            claims.Subject,
		AuthSource:     authSource,
		IdentitySource: identitySource,
	}

	// Enrich with the user's OIDC profile when the user record is available.
	if userID != "" && s.userProvider != nil {
		if user, err := s.userProvider.GetUser(ctx, userID); err == nil && user != nil {
			userData := map[string]interface{}(user)
			info.PreferredUsername = userInfoString(userData, "preferred_username")
			info.Email = userInfoString(userData, "email")
			info.Name = userInfoString(userData, "name")
			info.GivenName = userInfoString(userData, "given_name")
			info.FamilyName = userInfoString(userData, "family_name")
			info.Picture = userInfoString(userData, "picture")
			info.Locale = userInfoString(userData, "locale")

			if emailVerified, ok := userData["email_verified"].(bool); ok {
				info.EmailVerified = &emailVerified
			}
			if source, ok := userData[types.ClaimIdentitySource].(string); ok && source != "" {
				info.IdentitySource = source
			}
		}
	}

	return info
}

// userInfoString reads a string field from a user record.
func userInfoString(user map[string]interface{}, key string) string {
	if user == nil {
		return ""
	}
	value, _ := user[key].(string)
	return value
}
