package openapi_test

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/yaoapp/yao/openapi"
	"github.com/yaoapp/yao/openapi/oauth"
	"github.com/yaoapp/yao/openapi/oauth/types"
	"github.com/yaoapp/yao/openapi/tests/testutils"
)

func TestOAuthToken_JWTBearer_ValidToken(t *testing.T) {
	serverURL := testutils.Prepare(t)
	defer testutils.Clean()

	baseURL := ""
	if openapi.Server != nil && openapi.Server.Config != nil {
		baseURL = openapi.Server.Config.BaseURL
	}

	oauthService := oauth.OAuth
	assert.NotNil(t, oauthService, "OAuth service should be initialized")

	client := testutils.RegisterTestClient(t, "JWT Bearer Valid Test", []string{"https://localhost/callback"})
	defer testutils.CleanupTestClient(t, client.ClientID)

	// Create a valid (non-expired) access token
	accessToken, err := oauthService.MakeAccessToken(client.ClientID, "openid profile", "test-subject-valid", 3600)
	assert.NoError(t, err)
	assert.NotEmpty(t, accessToken)

	// Exchange it via JWT Bearer grant
	data := url.Values{}
	data.Set("grant_type", types.GrantTypeJWTBearer)
	data.Set("assertion", accessToken)

	endpoint := serverURL + baseURL + "/oauth/token"
	req, err := http.NewRequest("POST", endpoint, bytes.NewBufferString(data.Encode()))
	assert.NoError(t, err)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := http.DefaultClient.Do(req)
	assert.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	// Verify security headers
	assert.Equal(t, "no-store", resp.Header.Get("Cache-Control"))
	assert.Equal(t, "no-cache", resp.Header.Get("Pragma"))

	var tokenResp types.RefreshTokenResponse
	err = json.NewDecoder(resp.Body).Decode(&tokenResp)
	assert.NoError(t, err)

	assert.NotEmpty(t, tokenResp.AccessToken, "Should return a new access token")
	assert.Equal(t, "Bearer", tokenResp.TokenType)
	assert.Greater(t, tokenResp.ExpiresIn, 0)
	assert.NotEqual(t, accessToken, tokenResp.AccessToken, "New token should differ from the original")
	assert.Empty(t, tokenResp.RefreshToken, "JWT Bearer grant should not return a refresh token")

	t.Logf("JWT Bearer exchange succeeded: ExpiresIn=%d", tokenResp.ExpiresIn)
}

func TestOAuthToken_JWTBearer_ExpiredToken(t *testing.T) {
	serverURL := testutils.Prepare(t)
	defer testutils.Clean()

	baseURL := ""
	if openapi.Server != nil && openapi.Server.Config != nil {
		baseURL = openapi.Server.Config.BaseURL
	}

	oauthService := oauth.OAuth
	assert.NotNil(t, oauthService, "OAuth service should be initialized")

	client := testutils.RegisterTestClient(t, "JWT Bearer Expired Test", []string{"https://localhost/callback"})
	defer testutils.CleanupTestClient(t, client.ClientID)

	// Create an expired access token (expiresIn = -1 means already expired)
	expiredToken, err := oauthService.MakeAccessToken(client.ClientID, "openid profile", "test-subject-expired", -1)
	assert.NoError(t, err)
	assert.NotEmpty(t, expiredToken)

	// Exchange the expired JWT — should succeed (core Android use case)
	data := url.Values{}
	data.Set("grant_type", types.GrantTypeJWTBearer)
	data.Set("assertion", expiredToken)

	endpoint := serverURL + baseURL + "/oauth/token"
	req, err := http.NewRequest("POST", endpoint, bytes.NewBufferString(data.Encode()))
	assert.NoError(t, err)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := http.DefaultClient.Do(req)
	assert.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode, "Expired but signature-valid JWT should be accepted")

	var tokenResp types.RefreshTokenResponse
	err = json.NewDecoder(resp.Body).Decode(&tokenResp)
	assert.NoError(t, err)

	assert.NotEmpty(t, tokenResp.AccessToken)
	assert.Equal(t, "Bearer", tokenResp.TokenType)
	assert.Greater(t, tokenResp.ExpiresIn, 0, "New token should have a positive TTL")

	t.Logf("Expired JWT exchange succeeded: ExpiresIn=%d", tokenResp.ExpiresIn)
}

func TestOAuthToken_JWTBearer_ExtraClaimsPreserved(t *testing.T) {
	serverURL := testutils.Prepare(t)
	defer testutils.Clean()

	baseURL := ""
	if openapi.Server != nil && openapi.Server.Config != nil {
		baseURL = openapi.Server.Config.BaseURL
	}

	oauthService := oauth.OAuth
	assert.NotNil(t, oauthService, "OAuth service should be initialized")

	client := testutils.RegisterTestClient(t, "JWT Bearer Extra Claims Test", []string{"https://localhost/callback"})
	defer testutils.CleanupTestClient(t, client.ClientID)

	// Create a token with extra claims (team_id, tenant_id)
	extraClaims := map[string]interface{}{
		"team_id":   "team-123",
		"tenant_id": "tenant-456",
	}
	accessToken, err := oauthService.MakeAccessToken(client.ClientID, "openid profile", "test-subject-extra", 3600, extraClaims)
	assert.NoError(t, err)

	// Exchange via JWT Bearer
	data := url.Values{}
	data.Set("grant_type", types.GrantTypeJWTBearer)
	data.Set("assertion", accessToken)

	endpoint := serverURL + baseURL + "/oauth/token"
	req, err := http.NewRequest("POST", endpoint, bytes.NewBufferString(data.Encode()))
	assert.NoError(t, err)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := http.DefaultClient.Do(req)
	assert.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var tokenResp types.RefreshTokenResponse
	err = json.NewDecoder(resp.Body).Decode(&tokenResp)
	assert.NoError(t, err)
	assert.NotEmpty(t, tokenResp.AccessToken)

	// Verify the new token contains the extra claims by verifying it
	newClaims, err := oauthService.VerifyToken(tokenResp.AccessToken)
	assert.NoError(t, err)
	assert.Equal(t, "team-123", newClaims.TeamID, "team_id should be preserved")
	assert.Equal(t, "tenant-456", newClaims.TenantID, "tenant_id should be preserved")
	assert.Equal(t, "test-subject-extra", newClaims.Subject, "subject should be preserved")
	assert.Equal(t, client.ClientID, newClaims.ClientID, "client_id should be preserved")

	t.Logf("Extra claims preserved: team_id=%s, tenant_id=%s", newClaims.TeamID, newClaims.TenantID)
}

func TestOAuthToken_JWTBearer_InvalidSignature(t *testing.T) {
	serverURL := testutils.Prepare(t)
	defer testutils.Clean()

	baseURL := ""
	if openapi.Server != nil && openapi.Server.Config != nil {
		baseURL = openapi.Server.Config.BaseURL
	}

	// Send a JWT with tampered signature
	data := url.Values{}
	data.Set("grant_type", types.GrantTypeJWTBearer)
	data.Set("assertion", "eyJhbGciOiJSUzI1NiJ9.eyJzdWIiOiJmYWtlIn0.invalidsignature")

	endpoint := serverURL + baseURL + "/oauth/token"
	req, err := http.NewRequest("POST", endpoint, bytes.NewBufferString(data.Encode()))
	assert.NoError(t, err)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := http.DefaultClient.Do(req)
	assert.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusBadRequest, resp.StatusCode, "Invalid signature should be rejected")

	body, _ := io.ReadAll(resp.Body)
	assert.Contains(t, string(body), "invalid_grant", "Error should be invalid_grant")
}

func TestOAuthToken_JWTBearer_NonJWTInput(t *testing.T) {
	serverURL := testutils.Prepare(t)
	defer testutils.Clean()

	baseURL := ""
	if openapi.Server != nil && openapi.Server.Config != nil {
		baseURL = openapi.Server.Config.BaseURL
	}

	// Send a non-JWT string
	data := url.Values{}
	data.Set("grant_type", types.GrantTypeJWTBearer)
	data.Set("assertion", "not-a-jwt-token")

	endpoint := serverURL + baseURL + "/oauth/token"
	req, err := http.NewRequest("POST", endpoint, bytes.NewBufferString(data.Encode()))
	assert.NoError(t, err)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := http.DefaultClient.Do(req)
	assert.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusBadRequest, resp.StatusCode, "Non-JWT input should be rejected")

	body, _ := io.ReadAll(resp.Body)
	assert.Contains(t, string(body), "invalid_grant", "Error should be invalid_grant")
}

func TestOAuthToken_JWTBearer_MissingAssertion(t *testing.T) {
	serverURL := testutils.Prepare(t)
	defer testutils.Clean()

	baseURL := ""
	if openapi.Server != nil && openapi.Server.Config != nil {
		baseURL = openapi.Server.Config.BaseURL
	}

	// Send without assertion parameter
	data := url.Values{}
	data.Set("grant_type", types.GrantTypeJWTBearer)

	endpoint := serverURL + baseURL + "/oauth/token"
	req, err := http.NewRequest("POST", endpoint, bytes.NewBufferString(data.Encode()))
	assert.NoError(t, err)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := http.DefaultClient.Do(req)
	assert.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusBadRequest, resp.StatusCode, "Missing assertion should be rejected")

	body, _ := io.ReadAll(resp.Body)
	assert.Contains(t, string(body), "invalid_request", "Error should be invalid_request")
}

func TestOAuthToken_JWTBearer_ExpiredBeyondMaxAge(t *testing.T) {
	serverURL := testutils.Prepare(t)
	defer testutils.Clean()

	baseURL := ""
	if openapi.Server != nil && openapi.Server.Config != nil {
		baseURL = openapi.Server.Config.BaseURL
	}

	oauthService := oauth.OAuth
	assert.NotNil(t, oauthService, "OAuth service should be initialized")

	client := testutils.RegisterTestClient(t, "JWT Bearer Max Age Test", []string{"https://localhost/callback"})
	defer testutils.CleanupTestClient(t, client.ClientID)

	// Create a token that expired 25 hours ago (exceeds default 24h RefreshTokenLifetime)
	expiredToken, err := oauthService.MakeAccessToken(client.ClientID, "openid profile", "test-subject-old", -90000)
	assert.NoError(t, err)
	assert.NotEmpty(t, expiredToken)

	data := url.Values{}
	data.Set("grant_type", types.GrantTypeJWTBearer)
	data.Set("assertion", expiredToken)

	endpoint := serverURL + baseURL + "/oauth/token"
	req, err := http.NewRequest("POST", endpoint, bytes.NewBufferString(data.Encode()))
	assert.NoError(t, err)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := http.DefaultClient.Do(req)
	assert.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusBadRequest, resp.StatusCode, "Token expired beyond max age should be rejected")

	body, _ := io.ReadAll(resp.Body)
	assert.Contains(t, string(body), "invalid_grant", "Error should be invalid_grant")
	assert.Contains(t, string(body), "beyond maximum allowed age", "Should mention max age exceeded")
}

func TestOAuthToken_JWTBearer_FeatureFlagDisabled(t *testing.T) {
	serverURL := testutils.Prepare(t)
	defer testutils.Clean()

	baseURL := ""
	if openapi.Server != nil && openapi.Server.Config != nil {
		baseURL = openapi.Server.Config.BaseURL
	}

	oauthService := oauth.OAuth
	assert.NotNil(t, oauthService, "OAuth service should be initialized")

	// Disable JWT Bearer feature flag, restore on cleanup
	cfg := oauthService.GetConfig()
	cfg.Features.JWTBearerEnabled = false
	defer func() { cfg.Features.JWTBearerEnabled = true }()

	client := testutils.RegisterTestClient(t, "JWT Bearer Disabled Test", []string{"https://localhost/callback"})
	defer testutils.CleanupTestClient(t, client.ClientID)

	accessToken, err := oauthService.MakeAccessToken(client.ClientID, "openid profile", "test-subject-disabled", 3600)
	assert.NoError(t, err)

	data := url.Values{}
	data.Set("grant_type", types.GrantTypeJWTBearer)
	data.Set("assertion", accessToken)

	endpoint := serverURL + baseURL + "/oauth/token"
	req, err := http.NewRequest("POST", endpoint, bytes.NewBufferString(data.Encode()))
	assert.NoError(t, err)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := http.DefaultClient.Do(req)
	assert.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusBadRequest, resp.StatusCode, "Disabled feature should reject the grant")

	body, _ := io.ReadAll(resp.Body)
	assert.Contains(t, string(body), "unsupported_grant_type", "Error should be unsupported_grant_type")
}

func TestOAuthToken_JWTBearer_UnsupportedGrantType(t *testing.T) {
	serverURL := testutils.Prepare(t)
	defer testutils.Clean()

	baseURL := ""
	if openapi.Server != nil && openapi.Server.Config != nil {
		baseURL = openapi.Server.Config.BaseURL
	}

	// Send a completely unknown grant_type to verify the default branch still works
	data := url.Values{}
	data.Set("grant_type", "urn:unknown:grant-type")

	endpoint := serverURL + baseURL + "/oauth/token"
	req, err := http.NewRequest("POST", endpoint, bytes.NewBufferString(data.Encode()))
	assert.NoError(t, err)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := http.DefaultClient.Do(req)
	assert.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusBadRequest, resp.StatusCode, "Unknown grant type should be rejected")

	body, _ := io.ReadAll(resp.Body)
	assert.Contains(t, string(body), "unsupported_grant_type", "Error should be unsupported_grant_type")
}
