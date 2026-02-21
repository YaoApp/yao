package otp_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/yaoapp/yao/openapi"
	"github.com/yaoapp/yao/openapi/oauth"
	"github.com/yaoapp/yao/openapi/otp"
	"github.com/yaoapp/yao/openapi/tests/testutils"
)

type otpTestContext struct {
	UserID   string
	TeamID   string
	MemberID string
	Token    string // access token with team context
	ClientID string
	Scope    string
}

// setupTestData creates a real user, team type, team, and member for OTP tests.
// Returns an otpTestContext and a cleanup function.
func setupTestData(t *testing.T, serverURL string) (*otpTestContext, func()) {
	t.Helper()
	provider := testutils.GetUserProvider(t)
	ctx := context.Background()

	client := testutils.RegisterTestClient(t, "OTP Test Client", []string{"https://localhost/callback"})
	tokenInfo := testutils.ObtainAccessTokenWithRootPermission(t, serverURL, client.ClientID, client.ClientSecret, "https://localhost/callback", "openid profile")

	// Step 1: Create a team type
	teamTypeID := fmt.Sprintf("otp_team_type_%d", time.Now().UnixNano())
	_, err := provider.CreateType(ctx, map[string]interface{}{
		"type_id":     teamTypeID,
		"name":        "OTP Test Team Type",
		"locale":      "en-US",
		"description": "Team type for OTP tests",
		"is_active":   true,
		"created_at":  time.Now(),
		"updated_at":  time.Now(),
	})
	require.NoError(t, err, "Failed to create team type")

	// Step 2: Create a team with role_id (required for ACL)
	teamID := fmt.Sprintf("otp_test_team_%d", time.Now().UnixNano())
	_, err = provider.CreateTeam(ctx, map[string]interface{}{
		"team_id":     teamID,
		"name":        "OTP Test Team",
		"description": "Team for OTP integration tests",
		"owner_id":    tokenInfo.UserID,
		"type_id":     teamTypeID,
		"role_id":     "system:root",
		"status":      "active",
		"is_verified": true,
		"created_at":  time.Now(),
		"updated_at":  time.Now(),
	})
	require.NoError(t, err, "Failed to create test team")

	// Step 3: Add user as team member with system:root role
	memberID, err := provider.CreateMember(ctx, map[string]interface{}{
		"team_id":     teamID,
		"user_id":     tokenInfo.UserID,
		"member_type": "user",
		"role_id":     "system:root",
		"is_owner":    true,
		"status":      "active",
		"joined_at":   time.Now(),
		"created_at":  time.Now(),
		"updated_at":  time.Now(),
	})
	require.NoError(t, err, "Failed to create team member")

	// Step 4: Get team details for extra claims
	team, err := provider.GetTeamByMember(ctx, teamID, tokenInfo.UserID)
	require.NoError(t, err)

	// Step 5: Create access token with team context
	oauthService := oauth.OAuth
	require.NotNil(t, oauthService, "OAuth service not initialized")

	subject, err := oauthService.Subject(client.ClientID, tokenInfo.UserID)
	require.NoError(t, err)

	extraClaims := map[string]interface{}{
		"user_id": tokenInfo.UserID,
		"team_id": teamID,
	}
	if tenantID, ok := team["tenant_id"].(string); ok && tenantID != "" {
		extraClaims["tenant_id"] = tenantID
	}
	if ownerID, ok := team["owner_id"].(string); ok && ownerID != "" {
		extraClaims["owner_id"] = ownerID
	}
	if typeID, ok := team["type_id"].(string); ok && typeID != "" {
		extraClaims["type_id"] = typeID
	}

	scope := "openid profile email system:root"
	accessToken, err := oauthService.MakeAccessToken(client.ClientID, scope, subject, 3600, extraClaims)
	require.NoError(t, err)

	cleanup := func() {
		provider.DeleteTeam(ctx, teamID)
		provider.DeleteType(ctx, teamTypeID)
		testutils.CleanupTestClient(t, client.ClientID)
	}

	return &otpTestContext{
		UserID:   tokenInfo.UserID,
		TeamID:   teamID,
		MemberID: memberID,
		Token:    accessToken,
		ClientID: client.ClientID,
		Scope:    scope,
	}, cleanup
}

// ---------- POST /otp/login (public) ----------

func TestOTPLoginInvalidCode(t *testing.T) {
	serverURL := testutils.Prepare(t)
	defer testutils.Clean()

	baseURL := ""
	if openapi.Server != nil && openapi.Server.Config != nil {
		baseURL = openapi.Server.Config.BaseURL
	}

	body, _ := json.Marshal(map[string]string{"code": "nonexistent_code"})
	resp, err := http.Post(serverURL+baseURL+"/otp/login", "application/json", bytes.NewBuffer(body))
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)
	assert.Equal(t, "invalid_otp", result["error"])
}

func TestOTPLoginMissingCode(t *testing.T) {
	serverURL := testutils.Prepare(t)
	defer testutils.Clean()

	baseURL := ""
	if openapi.Server != nil && openapi.Server.Config != nil {
		baseURL = openapi.Server.Config.BaseURL
	}

	body, _ := json.Marshal(map[string]string{})
	resp, err := http.Post(serverURL+baseURL+"/otp/login", "application/json", bytes.NewBuffer(body))
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

func TestOTPLoginInvalidJSON(t *testing.T) {
	serverURL := testutils.Prepare(t)
	defer testutils.Clean()

	baseURL := ""
	if openapi.Server != nil && openapi.Server.Config != nil {
		baseURL = openapi.Server.Config.BaseURL
	}

	resp, err := http.Post(serverURL+baseURL+"/otp/login", "application/json", bytes.NewBufferString("not json"))
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

// ---------- POST /otp/create (disabled — HTTP endpoint removed for security) ----------
// Validation tests now covered by TestOTPServiceCreateValidation.

// ---------- Full flow: create (service) -> login (HTTP) ----------

func TestOTPCreateAndLogin(t *testing.T) {
	serverURL := testutils.Prepare(t)
	defer testutils.Clean()

	baseURL := ""
	if openapi.Server != nil && openapi.Server.Config != nil {
		baseURL = openapi.Server.Config.BaseURL
	}

	tc, cleanup := setupTestData(t, serverURL)
	defer cleanup()

	code, err := otp.OTP.Create(&otp.GenerateParams{
		UserID:   tc.UserID,
		TeamID:   tc.TeamID,
		Redirect: "/test/dashboard",
		Consume:  true,
	})
	require.NoError(t, err)
	assert.Len(t, code, 12)

	// Login with the OTP code (public endpoint)
	loginBody, _ := json.Marshal(map[string]string{"code": code, "locale": "en-US"})
	loginResp, err := http.Post(serverURL+baseURL+"/otp/login", "application/json", bytes.NewBuffer(loginBody))
	require.NoError(t, err)
	defer loginResp.Body.Close()

	loginRaw, _ := io.ReadAll(loginResp.Body)
	t.Logf("Login response: %d, body: %s", loginResp.StatusCode, string(loginRaw))
	require.Equal(t, http.StatusOK, loginResp.StatusCode, "body: %s", string(loginRaw))

	var loginResult map[string]interface{}
	json.Unmarshal(loginRaw, &loginResult)
	assert.Equal(t, "success", loginResult["status"])
	assert.Equal(t, "/test/dashboard", loginResult["redirect"])

	// Verify the code is consumed (default Consume=true)
	loginBody2, _ := json.Marshal(map[string]string{"code": code})
	loginResp2, err := http.Post(serverURL+baseURL+"/otp/login", "application/json", bytes.NewBuffer(loginBody2))
	require.NoError(t, err)
	defer loginResp2.Body.Close()
	assert.Equal(t, http.StatusUnauthorized, loginResp2.StatusCode, "Code should be consumed after first login")
}

func TestOTPCreateAndLoginWithMemberID(t *testing.T) {
	serverURL := testutils.Prepare(t)
	defer testutils.Clean()

	baseURL := ""
	if openapi.Server != nil && openapi.Server.Config != nil {
		baseURL = openapi.Server.Config.BaseURL
	}

	tc, cleanup := setupTestData(t, serverURL)
	defer cleanup()

	code, err := otp.OTP.Create(&otp.GenerateParams{
		TeamID:   tc.TeamID,
		MemberID: tc.MemberID,
		Redirect: "/member-login-test",
		Consume:  true,
	})
	require.NoError(t, err)

	payload, err := otp.OTP.Verify(code)
	require.NoError(t, err)
	assert.Equal(t, tc.MemberID, payload.MemberID)
	assert.Equal(t, tc.TeamID, payload.TeamID)
	assert.Equal(t, "", payload.UserID)

	loginBody, _ := json.Marshal(map[string]string{"code": code, "locale": "en-US"})
	loginResp, err := http.Post(serverURL+baseURL+"/otp/login", "application/json", bytes.NewBuffer(loginBody))
	require.NoError(t, err)
	defer loginResp.Body.Close()

	loginRaw, _ := io.ReadAll(loginResp.Body)
	t.Logf("Login response: %d, body: %s", loginResp.StatusCode, string(loginRaw))
	require.Equal(t, http.StatusOK, loginResp.StatusCode, "body: %s", string(loginRaw))

	var loginResult map[string]interface{}
	json.Unmarshal(loginRaw, &loginResult)
	assert.Equal(t, "success", loginResult["status"])
	assert.Equal(t, "/member-login-test", loginResult["redirect"])
}

func TestOTPCreateWithConsumeDisabled(t *testing.T) {
	serverURL := testutils.Prepare(t)
	defer testutils.Clean()

	baseURL := ""
	if openapi.Server != nil && openapi.Server.Config != nil {
		baseURL = openapi.Server.Config.BaseURL
	}

	tc, cleanup := setupTestData(t, serverURL)
	defer cleanup()

	code, err := otp.OTP.Create(&otp.GenerateParams{
		UserID:   tc.UserID,
		TeamID:   tc.TeamID,
		Redirect: "/reusable",
		Consume:  false,
	})
	require.NoError(t, err)

	// First login
	loginBody, _ := json.Marshal(map[string]string{"code": code, "locale": "en-US"})
	loginResp, err := http.Post(serverURL+baseURL+"/otp/login", "application/json", bytes.NewBuffer(loginBody))
	require.NoError(t, err)
	defer loginResp.Body.Close()
	require.Equal(t, http.StatusOK, loginResp.StatusCode)

	// Second login should also work (Consume=false)
	loginBody2, _ := json.Marshal(map[string]string{"code": code, "locale": "en-US"})
	loginResp2, err := http.Post(serverURL+baseURL+"/otp/login", "application/json", bytes.NewBuffer(loginBody2))
	require.NoError(t, err)
	defer loginResp2.Body.Close()
	assert.Equal(t, http.StatusOK, loginResp2.StatusCode, "Reusable OTP code should allow multiple logins")
}

func TestOTPCreateWithTokenExpiresIn(t *testing.T) {
	serverURL := testutils.Prepare(t)
	defer testutils.Clean()
	_ = serverURL

	code, err := otp.OTP.Create(&otp.GenerateParams{
		UserID:         "token_ttl_user",
		Redirect:       "/custom-ttl",
		TokenExpiresIn: 600,
		Consume:        true,
	})
	require.NoError(t, err)

	payload, err := otp.OTP.Verify(code)
	require.NoError(t, err)
	assert.Equal(t, 600, payload.TokenExpiresIn)
	assert.Equal(t, "/custom-ttl", payload.Redirect)
	assert.True(t, payload.Consume)
}

// ---------- Service-level tests (direct API) ----------

func TestOTPServiceCreateAndVerify(t *testing.T) {
	serverURL := testutils.Prepare(t)
	defer testutils.Clean()
	_ = serverURL

	code, err := otp.OTP.Create(&otp.GenerateParams{
		UserID:   "test_user_direct",
		TeamID:   "test_team_direct",
		Redirect: "/direct-test",
		Consume:  true,
	})
	require.NoError(t, err)
	assert.Len(t, code, 12)

	payload, err := otp.OTP.Verify(code)
	require.NoError(t, err)
	assert.Equal(t, "test_user_direct", payload.UserID)
	assert.Equal(t, "test_team_direct", payload.TeamID)
	assert.Equal(t, "/direct-test", payload.Redirect)
	assert.True(t, payload.Consume)
	assert.Equal(t, 0, payload.TokenExpiresIn)
}

func TestOTPServiceCreateAndVerifyWithMemberID(t *testing.T) {
	serverURL := testutils.Prepare(t)
	defer testutils.Clean()
	_ = serverURL

	code, err := otp.OTP.Create(&otp.GenerateParams{
		TeamID:         "team_member_test",
		MemberID:       "member_12345",
		Redirect:       "/member-redirect",
		Scope:          "read:data",
		TokenExpiresIn: 900,
		Consume:        false,
	})
	require.NoError(t, err)
	assert.Len(t, code, 12)

	payload, err := otp.OTP.Verify(code)
	require.NoError(t, err)
	assert.Equal(t, "", payload.UserID)
	assert.Equal(t, "team_member_test", payload.TeamID)
	assert.Equal(t, "member_12345", payload.MemberID)
	assert.Equal(t, "/member-redirect", payload.Redirect)
	assert.Equal(t, "read:data", payload.Scope)
	assert.Equal(t, 900, payload.TokenExpiresIn)
	assert.False(t, payload.Consume)
}

func TestOTPServiceCreateStoresMapPayload(t *testing.T) {
	serverURL := testutils.Prepare(t)
	defer testutils.Clean()
	_ = serverURL

	code, err := otp.OTP.Create(&otp.GenerateParams{
		TeamID:         "map_team",
		MemberID:       "map_member",
		Redirect:       "$dashboard/assistants",
		Scope:          "openid profile",
		TokenExpiresIn: 600,
		Consume:        true,
	})
	require.NoError(t, err)

	payload, err := otp.OTP.Verify(code)
	require.NoError(t, err)
	assert.Equal(t, "map_team", payload.TeamID)
	assert.Equal(t, "map_member", payload.MemberID)
	assert.Equal(t, "$dashboard/assistants", payload.Redirect)
	assert.Equal(t, "openid profile", payload.Scope)
	assert.Equal(t, 600, payload.TokenExpiresIn)
	assert.True(t, payload.Consume)
}

func TestOTPServiceVerifyEmptyCode(t *testing.T) {
	serverURL := testutils.Prepare(t)
	defer testutils.Clean()
	_ = serverURL

	_, err := otp.OTP.Verify("")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "code is required")
}

func TestOTPServiceVerifyNonexistent(t *testing.T) {
	serverURL := testutils.Prepare(t)
	defer testutils.Clean()
	_ = serverURL

	_, err := otp.OTP.Verify("doesnotexist1")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid or expired")
}

func TestOTPServiceRevoke(t *testing.T) {
	serverURL := testutils.Prepare(t)
	defer testutils.Clean()
	_ = serverURL

	code, err := otp.OTP.Create(&otp.GenerateParams{
		UserID:   "revoke_user",
		Redirect: "/revoke-test",
		Consume:  true,
	})
	require.NoError(t, err)

	_, err = otp.OTP.Verify(code)
	require.NoError(t, err)

	err = otp.OTP.Revoke(code)
	require.NoError(t, err)

	_, err = otp.OTP.Verify(code)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid or expired")
}

func TestOTPServiceRevokeEmpty(t *testing.T) {
	serverURL := testutils.Prepare(t)
	defer testutils.Clean()
	_ = serverURL

	err := otp.OTP.Revoke("")
	assert.NoError(t, err, "Revoking empty code should be silent")
}

func TestOTPServiceCreateValidation(t *testing.T) {
	serverURL := testutils.Prepare(t)
	defer testutils.Clean()
	_ = serverURL

	tests := []struct {
		name   string
		params *otp.GenerateParams
		errMsg string
	}{
		{"nil params", nil, "params is required"},
		{"missing user and member", &otp.GenerateParams{Redirect: "/test"}, "user_id or member_id is required"},
		{"missing redirect", &otp.GenerateParams{UserID: "u1"}, "redirect is required"},
		{"member_id without team_id", &otp.GenerateParams{MemberID: "m1", Redirect: "/test"}, "team_id is required"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := otp.OTP.Create(tt.params)
			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.errMsg)
		})
	}
}

func TestOTPServiceCreateWithScope(t *testing.T) {
	serverURL := testutils.Prepare(t)
	defer testutils.Clean()
	_ = serverURL

	code, err := otp.OTP.Create(&otp.GenerateParams{
		UserID:   "scope_user",
		TeamID:   "scope_team",
		Redirect: "/scoped",
		Scope:    "read:data write:data",
		Consume:  false,
	})
	require.NoError(t, err)

	payload, err := otp.OTP.Verify(code)
	require.NoError(t, err)
	assert.Equal(t, "read:data write:data", payload.Scope)
	assert.False(t, payload.Consume)
}

func TestOTPServiceCreateWithCustomExpiry(t *testing.T) {
	serverURL := testutils.Prepare(t)
	defer testutils.Clean()
	_ = serverURL

	code, err := otp.OTP.Create(&otp.GenerateParams{
		UserID:         "expiry_user",
		Redirect:       "/custom-expiry",
		ExpiresIn:      60,
		TokenExpiresIn: 300,
		Consume:        true,
	})
	require.NoError(t, err)

	payload, err := otp.OTP.Verify(code)
	require.NoError(t, err)
	assert.Equal(t, 300, payload.TokenExpiresIn)
}

// ---------- Cross-team validation ----------
// NOTE: Cross-team HTTP endpoint test removed — /otp/create is disabled.
// Server-side Process callers are trusted and should validate team membership themselves.

// ---------- OTP Login sets cookies (no refresh token) ----------

func TestOTPLoginSetsAccessTokenCookieOnly(t *testing.T) {
	serverURL := testutils.Prepare(t)
	defer testutils.Clean()

	baseURL := ""
	if openapi.Server != nil && openapi.Server.Config != nil {
		baseURL = openapi.Server.Config.BaseURL
	}

	tc, cleanup := setupTestData(t, serverURL)
	defer cleanup()

	code, err := otp.OTP.Create(&otp.GenerateParams{
		UserID:   tc.UserID,
		TeamID:   tc.TeamID,
		Redirect: "/cookie-test",
		Consume:  true,
	})
	require.NoError(t, err)

	loginBody, _ := json.Marshal(map[string]string{"code": code})
	loginResp, err := http.Post(serverURL+baseURL+"/otp/login", "application/json", bytes.NewBuffer(loginBody))
	require.NoError(t, err)
	defer loginResp.Body.Close()
	require.Equal(t, http.StatusOK, loginResp.StatusCode)

	cookies := loginResp.Cookies()
	hasAccessToken := false
	hasRefreshToken := false
	for _, c := range cookies {
		t.Logf("Cookie: %s", c.Name)
		if strings.HasSuffix(c.Name, "access_token") {
			hasAccessToken = true
		}
		if strings.HasSuffix(c.Name, "refresh_token") {
			hasRefreshToken = true
		}
	}
	assert.True(t, hasAccessToken, "OTP login should set access_token cookie")
	assert.False(t, hasRefreshToken, "OTP login should NOT set refresh_token cookie (SkipRefreshToken)")
}

// ---------- Code uniqueness ----------

func TestOTPCodeUniqueness(t *testing.T) {
	serverURL := testutils.Prepare(t)
	defer testutils.Clean()
	_ = serverURL

	codes := make(map[string]bool)
	for i := 0; i < 50; i++ {
		code, err := otp.OTP.Create(&otp.GenerateParams{
			UserID:   fmt.Sprintf("unique_user_%d", i),
			Redirect: "/unique",
			Consume:  true,
		})
		require.NoError(t, err)
		assert.False(t, codes[code], "Duplicate OTP code generated: %s", code)
		codes[code] = true
	}
	assert.Len(t, codes, 50, "All 50 codes should be unique")
}
