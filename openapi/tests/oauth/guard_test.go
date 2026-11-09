package openapi_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/yaoapp/yao/openapi/oauth"
	"github.com/yaoapp/yao/openapi/oauth/authorized"
	"github.com/yaoapp/yao/openapi/response"
	"github.com/yaoapp/yao/openapi/tests/testutils"
)

// TestGuard_ValidToken verifies that a valid, non-expired access token passes through authentication.
func TestGuard_ValidToken(t *testing.T) {
	serverURL := testutils.Prepare(t)
	defer testutils.Clean()
	_ = serverURL

	oauthService := oauth.OAuth
	assert.NotNil(t, oauthService, "OAuth service should be initialized")

	client := testutils.RegisterTestClient(t, "Guard Valid Token Test", []string{"https://localhost/callback"})
	defer testutils.CleanupTestClient(t, client.ClientID)

	tokenInfo := testutils.ObtainAccessToken(t, serverURL, client.ClientID, client.ClientSecret, "https://localhost/callback", "openid profile")

	router := authenticateRouter(oauthService)

	accessCookieName := response.GetCookieName("access_token")
	req := httptest.NewRequest("GET", "/guarded", nil)
	req.AddCookie(&http.Cookie{Name: accessCookieName, Value: fmt.Sprintf("Bearer %s", tokenInfo.AccessToken)})

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code, "Valid token should pass authentication")
	assert.Contains(t, w.Body.String(), `"subject"`, "Response should contain authorized subject")
}

// TestGuard_NoToken verifies that a request without any token is rejected with 401.
func TestGuard_NoToken(t *testing.T) {
	serverURL := testutils.Prepare(t)
	defer testutils.Clean()
	_ = serverURL

	oauthService := oauth.OAuth
	assert.NotNil(t, oauthService, "OAuth service should be initialized")

	router := authenticateRouter(oauthService)

	req := httptest.NewRequest("GET", "/guarded", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code, "No token should return 401")
	assert.Contains(t, w.Body.String(), "token_missing", "Error should indicate missing token")
}

// TestGuard_InvalidSignature verifies that a token with an invalid signature is rejected with 401.
func TestGuard_InvalidSignature(t *testing.T) {
	serverURL := testutils.Prepare(t)
	defer testutils.Clean()
	_ = serverURL

	oauthService := oauth.OAuth
	assert.NotNil(t, oauthService, "OAuth service should be initialized")

	router := authenticateRouter(oauthService)

	accessCookieName := response.GetCookieName("access_token")
	req := httptest.NewRequest("GET", "/guarded", nil)
	req.AddCookie(&http.Cookie{Name: accessCookieName, Value: "Bearer eyJhbGciOiJSUzI1NiJ9.eyJzdWIiOiJmYWtlIn0.invalidsignature"})

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code, "Invalid signature should return 401")
}

// TestGuard_ExpiredToken_NoRefresh verifies that an expired access token without a refresh token returns 401.
func TestGuard_ExpiredToken_NoRefresh(t *testing.T) {
	serverURL := testutils.Prepare(t)
	defer testutils.Clean()
	_ = serverURL

	oauthService := oauth.OAuth
	assert.NotNil(t, oauthService, "OAuth service should be initialized")

	client := testutils.RegisterTestClient(t, "Guard Expired No Refresh Test", []string{"https://localhost/callback"})
	defer testutils.CleanupTestClient(t, client.ClientID)

	expiredToken, err := oauthService.MakeAccessToken(client.ClientID, "openid profile", "test-subject-expired", -1)
	assert.NoError(t, err, "Should be able to create expired token")

	router := authenticateRouter(oauthService)

	accessCookieName := response.GetCookieName("access_token")
	req := httptest.NewRequest("GET", "/guarded", nil)
	req.AddCookie(&http.Cookie{Name: accessCookieName, Value: fmt.Sprintf("Bearer %s", expiredToken)})

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code, "Expired token without refresh token should return 401")
}

// TestGuard_ExpiredToken_WithValidRefresh verifies that an expired access token with a valid refresh token
// triggers auto-refresh: the request succeeds and a new access_token cookie is set.
func TestGuard_ExpiredToken_WithValidRefresh(t *testing.T) {
	serverURL := testutils.Prepare(t)
	defer testutils.Clean()
	_ = serverURL

	oauthService := oauth.OAuth
	assert.NotNil(t, oauthService, "OAuth service should be initialized")

	client := testutils.RegisterTestClient(t, "Guard Auto Refresh Test", []string{"https://localhost/callback"})
	defer testutils.CleanupTestClient(t, client.ClientID)

	subject := "test-subject-auto-refresh"

	expiredToken, err := oauthService.MakeAccessToken(client.ClientID, "openid profile", subject, -1)
	assert.NoError(t, err, "Should create expired access token")

	// Create a JWT-format refresh token so VerifyToken can validate it directly.
	// The default opaque format requires store lookup which is separate from the signing path.
	refreshToken, err := oauthService.MakeRefreshToken(client.ClientID, "openid profile", subject, 86400)
	assert.NoError(t, err, "Should create valid refresh token")

	router := authenticateRouter(oauthService)

	accessCookieName := response.GetCookieName("access_token")
	refreshCookieName := response.GetCookieName("refresh_token")

	req := httptest.NewRequest("GET", "/guarded", nil)
	req.AddCookie(&http.Cookie{Name: accessCookieName, Value: fmt.Sprintf("Bearer %s", expiredToken)})
	req.AddCookie(&http.Cookie{Name: refreshCookieName, Value: fmt.Sprintf("Bearer %s", refreshToken)})

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code, "Expired token + valid refresh should auto-refresh and succeed")
	assert.Contains(t, w.Body.String(), `"subject"`, "Response should contain authorized subject")

	// Verify that both access_token and refresh_token cookies were rotated
	setCookieHeaders := w.Result().Cookies()
	foundNewAccessToken := false
	foundNewRefreshToken := false
	for _, c := range setCookieHeaders {
		if c.Name == accessCookieName {
			foundNewAccessToken = true
			assert.NotEmpty(t, c.Value, "New access token cookie should have a value")
			rawValue := strings.TrimPrefix(c.Value, "Bearer ")
			assert.NotEqual(t, expiredToken, rawValue, "New token should differ from the expired one")
			t.Logf("New access_token cookie set with MaxAge=%d", c.MaxAge)
		}
		if c.Name == refreshCookieName {
			foundNewRefreshToken = true
			assert.NotEmpty(t, c.Value, "New refresh token cookie should have a value")
			rawValue := strings.TrimPrefix(c.Value, "Bearer ")
			assert.NotEqual(t, refreshToken, rawValue, "New refresh token should differ from the old one")
			t.Logf("New refresh_token cookie set with MaxAge=%d", c.MaxAge)
		}
	}
	assert.True(t, foundNewAccessToken, "Guard should write a new access_token cookie after auto-refresh")
	assert.True(t, foundNewRefreshToken, "Guard should rotate refresh_token cookie after auto-refresh")
}

// TestGuard_ExpiredToken_WithExpiredRefresh verifies that an expired access token paired with an
// also-expired refresh token returns 401.
func TestGuard_ExpiredToken_WithExpiredRefresh(t *testing.T) {
	serverURL := testutils.Prepare(t)
	defer testutils.Clean()
	_ = serverURL

	oauthService := oauth.OAuth
	assert.NotNil(t, oauthService, "OAuth service should be initialized")

	client := testutils.RegisterTestClient(t, "Guard Expired Refresh Test", []string{"https://localhost/callback"})
	defer testutils.CleanupTestClient(t, client.ClientID)

	subject := "test-subject-both-expired"

	expiredAccess, err := oauthService.MakeAccessToken(client.ClientID, "openid profile", subject, -1)
	assert.NoError(t, err)

	// Opaque refresh tokens expire via store TTL, not a field in the data.
	// Use a 1-second TTL and wait for it to expire from the store.
	expiredRefresh, err := oauthService.MakeRefreshToken(client.ClientID, "openid profile", subject, 1)
	assert.NoError(t, err)

	time.Sleep(2 * time.Second)

	router := authenticateRouter(oauthService)

	accessCookieName := response.GetCookieName("access_token")
	refreshCookieName := response.GetCookieName("refresh_token")

	req := httptest.NewRequest("GET", "/guarded", nil)
	req.AddCookie(&http.Cookie{Name: accessCookieName, Value: fmt.Sprintf("Bearer %s", expiredAccess)})
	req.AddCookie(&http.Cookie{Name: refreshCookieName, Value: fmt.Sprintf("Bearer %s", expiredRefresh)})

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code, "Both tokens expired should return 401")
}

// TestGuard_AuthorizationHeader verifies that the Guard also works with the Authorization header.
func TestGuard_AuthorizationHeader(t *testing.T) {
	serverURL := testutils.Prepare(t)
	defer testutils.Clean()
	_ = serverURL

	oauthService := oauth.OAuth
	assert.NotNil(t, oauthService, "OAuth service should be initialized")

	client := testutils.RegisterTestClient(t, "Guard Header Auth Test", []string{"https://localhost/callback"})
	defer testutils.CleanupTestClient(t, client.ClientID)

	tokenInfo := testutils.ObtainAccessToken(t, serverURL, client.ClientID, client.ClientSecret, "https://localhost/callback", "openid profile")

	router := authenticateRouter(oauthService)

	req := httptest.NewRequest("GET", "/guarded", nil)
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", tokenInfo.AccessToken))

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code, "Valid Bearer token in Authorization header should pass authentication")
	assert.Contains(t, w.Body.String(), `"subject"`, "Response should contain authorized subject")
}

// authenticateRouter creates a Gin router with ONLY the Authenticate middleware (no ACL).
// This isolates the token verification and auto-refresh logic from permission checks.
func authenticateRouter(oauthService *oauth.Service) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()

	handler := func(c *gin.Context) {
		info := authorized.GetInfo(c)
		if info == nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "no authorized info"})
			return
		}
		c.JSON(http.StatusOK, gin.H{
			"subject":    info.Subject,
			"client_id":  info.ClientID,
			"scope":      info.Scope,
			"user_id":    info.UserID,
			"session_id": info.SessionID,
		})
	}

	// Use Authenticate (auth only) instead of Guard (auth + ACL)
	router.GET("/guarded", func(c *gin.Context) {
		if !oauthService.Authenticate(c) {
			return
		}
		handler(c)
	})

	return router
}
