package nodes_test

import (
	"context"

	"github.com/gin-gonic/gin"
	oauthTypes "github.com/yaoapp/yao/openapi/oauth/types"
)

// authOAuth is a minimal OAuth stub for Attach() in unit tests.
// Guard injects auth from X-Test-User-ID / X-Test-Team-ID headers.
type authOAuth struct{}

var _ oauthTypes.OAuth = authOAuth{}

func (authOAuth) Guard(c *gin.Context) {
	c.Set("__subject", "test-subject")
	c.Set("__client_id", "test-client")
	c.Set("__scope", "openid profile")
	if uid := c.GetHeader("X-Test-User-ID"); uid != "" {
		c.Set("__user_id", uid)
	}
	if tid := c.GetHeader("X-Test-Team-ID"); tid != "" {
		c.Set("__team_id", tid)
	}
	c.Next()
}

func (authOAuth) AuthorizationServer(context.Context) string { return "" }
func (authOAuth) ProtectedResource(context.Context) string   { return "" }
func (authOAuth) Authorize(context.Context, *oauthTypes.AuthorizationRequest) (*oauthTypes.AuthorizationResponse, error) {
	return nil, nil
}
func (authOAuth) Token(context.Context, string, string, string, string) (*oauthTypes.Token, error) {
	return nil, nil
}
func (authOAuth) Revoke(context.Context, string, string) error { return nil }
func (authOAuth) Introspect(context.Context, string) (*oauthTypes.TokenIntrospectionResponse, error) {
	return nil, nil
}
func (authOAuth) Register(context.Context, *oauthTypes.ClientInfo) (*oauthTypes.ClientInfo, error) {
	return nil, nil
}
func (authOAuth) JWKS(context.Context) (*oauthTypes.JWKSResponse, error) { return nil, nil }
func (authOAuth) Endpoints(context.Context) (map[string]string, error)   { return nil, nil }
func (authOAuth) RefreshToken(context.Context, string, ...string) (*oauthTypes.RefreshTokenResponse, error) {
	return nil, nil
}
func (authOAuth) DeviceAuthorization(context.Context, string, string) (*oauthTypes.DeviceAuthorizationResponse, error) {
	return nil, nil
}
func (authOAuth) UserInfo(context.Context, string) (interface{}, error) { return nil, nil }
func (authOAuth) GenerateCodeChallenge(context.Context, string, string) (string, error) {
	return "", nil
}
func (authOAuth) ValidateCodeChallenge(context.Context, string, string, string) error { return nil }
func (authOAuth) PushAuthorizationRequest(context.Context, *oauthTypes.PushedAuthorizationRequest) (*oauthTypes.PushedAuthorizationResponse, error) {
	return nil, nil
}
func (authOAuth) TokenExchange(context.Context, string, string, string, string) (*oauthTypes.TokenExchangeResponse, error) {
	return nil, nil
}
func (authOAuth) UpdateClient(context.Context, string, *oauthTypes.ClientInfo) (*oauthTypes.ClientInfo, error) {
	return nil, nil
}
func (authOAuth) DeleteClient(context.Context, string) error { return nil }
func (authOAuth) ValidateScope(context.Context, []string, string) (*oauthTypes.ValidationResult, error) {
	return nil, nil
}
func (authOAuth) GetServerMetadata(context.Context) (*oauthTypes.AuthorizationServerMetadata, error) {
	return nil, nil
}
func (authOAuth) ValidateResourceParameter(context.Context, string) (*oauthTypes.ValidationResult, error) {
	return nil, nil
}
func (authOAuth) GetCanonicalResourceURI(context.Context, string) (string, error) { return "", nil }
func (authOAuth) GetProtectedResourceMetadata(context.Context) (*oauthTypes.ProtectedResourceMetadata, error) {
	return nil, nil
}
func (authOAuth) HandleWWWAuthenticate(context.Context, string) (*oauthTypes.WWWAuthenticateChallenge, error) {
	return nil, nil
}
func (authOAuth) DynamicClientRegistration(context.Context, *oauthTypes.DynamicClientRegistrationRequest) (*oauthTypes.DynamicClientRegistrationResponse, error) {
	return nil, nil
}
func (authOAuth) ValidateStateParameter(context.Context, string, string) (*oauthTypes.ValidationResult, error) {
	return nil, nil
}
func (authOAuth) GenerateStateParameter(context.Context, string) (*oauthTypes.StateParameter, error) {
	return nil, nil
}
func (authOAuth) ValidateTokenAudience(context.Context, string, string) (*oauthTypes.ValidationResult, error) {
	return nil, nil
}
func (authOAuth) ValidateRedirectURI(context.Context, string, []string) (*oauthTypes.ValidationResult, error) {
	return nil, nil
}
func (authOAuth) RotateRefreshToken(context.Context, string, ...string) (*oauthTypes.RefreshTokenResponse, error) {
	return nil, nil
}
func (authOAuth) ValidateTokenBinding(context.Context, string, *oauthTypes.TokenBinding) (*oauthTypes.ValidationResult, error) {
	return nil, nil
}
