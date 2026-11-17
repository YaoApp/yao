package oauth

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"math/big"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/yaoapp/gou/store/lru"
	clientprovider "github.com/yaoapp/yao/openapi/oauth/providers/client"
	"github.com/yaoapp/yao/openapi/oauth/types"
	"github.com/yaoapp/yao/openapi/response"
)

// =============================================================================
// Unit test helpers (Phase 2: Yao OAuth Server client authorization)
//
// These helpers build an isolated OAuth service with an in-memory store and an
// in-memory signing key pair. No database or application environment is needed.
// =============================================================================

// newUnitTestService creates an isolated OAuth service for pure unit tests.
func newUnitTestService(t *testing.T) *Service {
	t.Helper()

	st, err := lru.New(4096)
	require.NoError(t, err)

	clientProvider, err := clientprovider.NewDefaultClient(&clientprovider.DefaultClientOptions{
		Prefix: "__unit:",
		Store:  st,
		Cache:  st,
	})
	require.NoError(t, err)

	return &Service{
		config: &Config{
			Store:     st,
			Cache:     st,
			IssuerURL: "https://oauth.unit.test",
			BaseURL:   "/v1",
			Token: types.TokenConfig{
				AccessTokenLifetime:       time.Hour,
				RefreshTokenLifetime:      24 * time.Hour,
				AuthorizationCodeLifetime: 10 * time.Minute,
				DeviceCodeLifetime:        15 * time.Minute,
				AccessTokenFormat:         "jwt",
				RefreshTokenFormat:        "opaque",
				AccessTokenSigningAlg:     "RS256",
			},
			Security: types.SecurityConfig{
				PKCECodeChallengeMethod: []string{"S256"},
				PKCECodeVerifierLength:  64,
			},
			Client: types.ClientConfig{
				DefaultClientType:              types.ClientTypeConfidential,
				DefaultTokenEndpointAuthMethod: types.TokenEndpointAuthBasic,
				DefaultGrantTypes:              []string{types.GrantTypeAuthorizationCode, types.GrantTypeRefreshToken},
				DefaultResponseTypes:           []string{types.ResponseTypeCode},
				ClientIDLength:                 32,
				ClientSecretLength:             64,
				DynamicRegistrationEnabled:     true,
				AllowedRedirectURISchemes:      []string{"https", "http"},
				AllowedRedirectURIHosts:        []string{"localhost", "127.0.0.1", "client.example.com"},
			},
			Features: FeatureFlags{
				OAuth21Enabled:                   true,
				PKCEEnforced:                     true,
				RefreshTokenRotationEnabled:      true,
				DynamicClientRegistrationEnabled: true,
			},
		},
		store:          st,
		cache:          st,
		clientProvider: clientProvider,
		prefix:         "__unit:",
		signingCerts:   newTestSigningCertificates(t),
	}
}

// newTestSigningCertificates generates an in-memory RSA signing key pair.
func newTestSigningCertificates(t *testing.T) *SigningCertificates {
	t.Helper()

	key, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)

	template := x509.Certificate{
		SerialNumber:          big.NewInt(1),
		Subject:               pkix.Name{CommonName: "oauth-unit-test"},
		NotBefore:             time.Now().Add(-time.Hour),
		NotAfter:              time.Now().Add(24 * time.Hour),
		KeyUsage:              x509.KeyUsageDigitalSignature,
		BasicConstraintsValid: true,
	}

	der, err := x509.CreateCertificate(rand.Reader, &template, &template, &key.PublicKey, key)
	require.NoError(t, err)

	cert, err := x509.ParseCertificate(der)
	require.NoError(t, err)

	return &SigningCertificates{
		SigningCert: cert,
		SigningKey:  key,
		Algorithm:   "RS256",
	}
}

// createUnitTestClient registers a confidential client for unit tests.
func createUnitTestClient(t *testing.T, service *Service, clientID string, redirectURIs []string) *types.ClientInfo {
	t.Helper()

	if len(redirectURIs) == 0 {
		redirectURIs = []string{"https://localhost/callback"}
	}

	created, err := service.clientProvider.CreateClient(context.Background(), &types.ClientInfo{
		ClientID:                clientID,
		ClientSecret:            "unit-test-secret",
		ClientName:              "Unit Test Client",
		ClientType:              types.ClientTypeConfidential,
		RedirectURIs:            redirectURIs,
		GrantTypes:              []string{types.GrantTypeAuthorizationCode, types.GrantTypeRefreshToken},
		ResponseTypes:           []string{types.ResponseTypeCode},
		Scope:                   "openid profile email",
		ApplicationType:         types.ApplicationTypeWeb,
		TokenEndpointAuthMethod: types.TokenEndpointAuthBasic,
	})
	require.NoError(t, err)
	return created
}

// newGinTestContext builds a gin context backed by an httptest recorder.
func newGinTestContext(method, target string, headers map[string]string, cookies []*http.Cookie) (*gin.Context, *httptest.ResponseRecorder) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(method, target, nil)
	c.Request.Host = c.Request.URL.Host
	for key, value := range headers {
		c.Request.Header.Set(key, value)
	}
	for _, cookie := range cookies {
		c.Request.AddCookie(cookie)
	}
	return c, recorder
}

// accessTokenCookie builds the access token cookie used by Yao sessions.
func accessTokenCookie(token string) *http.Cookie {
	return &http.Cookie{
		Name:  response.GetCookieName("access_token"),
		Value: "Bearer " + token,
	}
}

// =============================================================================
// Login redirect / return_to helpers
// =============================================================================

func TestBuildLoginRedirectURL(t *testing.T) {
	returnTo := "https://server.test/v1/oauth/authorize?client_id=abc&redirect_uri=https%3A%2F%2Fclient.example.com%2Fcb&state=s1"

	t.Run("default entry path", func(t *testing.T) {
		loginURL := BuildLoginRedirectURL(DefaultLoginEntryPath, returnTo)
		parsed, err := url.Parse(loginURL)
		require.NoError(t, err)
		assert.Equal(t, DefaultLoginEntryPath, parsed.Path)
		assert.Equal(t, returnTo, parsed.Query().Get("return_to"), "return_to must round-trip unchanged")
	})

	t.Run("empty entry path falls back to default", func(t *testing.T) {
		loginURL := BuildLoginRedirectURL("", returnTo)
		assert.True(t, strings.HasPrefix(loginURL, DefaultLoginEntryPath+"?return_to="))
	})

	t.Run("entry path with existing query", func(t *testing.T) {
		loginURL := BuildLoginRedirectURL("/login?locale=en", returnTo)
		parsed, err := url.Parse(loginURL)
		require.NoError(t, err)
		assert.Equal(t, "/login", parsed.Path)
		assert.Equal(t, "en", parsed.Query().Get("locale"))
		assert.Equal(t, returnTo, parsed.Query().Get("return_to"))
	})

	t.Run("special characters are encoded", func(t *testing.T) {
		raw := "https://server.test/v1/oauth/authorize?scope=openid+profile&state=a%26b"
		loginURL := BuildLoginRedirectURL(DefaultLoginEntryPath, raw)
		assert.NotContains(t, loginURL, "&state=", "return_to value must be encoded")
		assert.Contains(t, loginURL, "state%3Da%2526b")
		parsed, err := url.Parse(loginURL)
		require.NoError(t, err)
		assert.Equal(t, raw, parsed.Query().Get("return_to"))
	})
}

func TestBuildAuthorizeReturnTo(t *testing.T) {
	t.Run("http request preserves path and query", func(t *testing.T) {
		c, _ := newGinTestContext(http.MethodGet, "http://server.test/v1/oauth/authorize?client_id=abc&state=s1", nil, nil)
		assert.Equal(t, "http://server.test/v1/oauth/authorize?client_id=abc&state=s1", BuildAuthorizeReturnTo(c))
	})

	t.Run("forwarded headers are honoured", func(t *testing.T) {
		headers := map[string]string{
			"X-Forwarded-Proto": "https, http",
			"X-Forwarded-Host":  "auth.example.com",
		}
		c, _ := newGinTestContext(http.MethodGet, "http://internal:5098/v1/oauth/authorize?state=s2", headers, nil)
		assert.Equal(t, "https://auth.example.com/v1/oauth/authorize?state=s2", BuildAuthorizeReturnTo(c))
	})

	t.Run("TLS request uses https", func(t *testing.T) {
		c, _ := newGinTestContext(http.MethodGet, "http://server.test/v1/oauth/authorize", nil, nil)
		c.Request.TLS = &tls.ConnectionState{}
		assert.Equal(t, "https://server.test/v1/oauth/authorize", BuildAuthorizeReturnTo(c))
	})

	t.Run("no request returns empty string", func(t *testing.T) {
		assert.Equal(t, "", BuildAuthorizeReturnTo(nil))
	})
}

func TestShouldRedirectToLogin(t *testing.T) {
	assert.True(t, ShouldRedirectToLogin(""), "default authorize requests may go through interactive login")
	assert.True(t, ShouldRedirectToLogin("login"))
	assert.True(t, ShouldRedirectToLogin("select_account"))
	assert.False(t, ShouldRedirectToLogin(PromptNone))
	assert.False(t, ShouldRedirectToLogin("NONE"), "prompt comparison must be case-insensitive")
	assert.Equal(t, "none", PromptNone)
}

// =============================================================================
// Yao session resolution
// =============================================================================

func TestResolveAuthorizeSession(t *testing.T) {
	service := newUnitTestService(t)
	sessionClient := createUnitTestClient(t, service, "session-client", nil)
	downstreamClient := createUnitTestClient(t, service, "downstream-client", []string{"https://client.example.com/callback"})

	userID := "user-42"
	sessionSubject, err := service.Subject(sessionClient.ClientID, userID)
	require.NoError(t, err)

	newSessionRequest := func() (*gin.Context, *httptest.ResponseRecorder) {
		return newGinTestContext(http.MethodGet, "http://server.test/v1/oauth/authorize?client_id=downstream-client", nil, nil)
	}

	t.Run("no session returns nil", func(t *testing.T) {
		c, _ := newSessionRequest()
		session, err := service.ResolveAuthorizeSession(c, downstreamClient.ClientID)
		require.NoError(t, err)
		assert.Nil(t, session)
	})

	t.Run("invalid token is treated as anonymous", func(t *testing.T) {
		c, _ := newGinTestContext(http.MethodGet, "http://server.test/v1/oauth/authorize", nil,
			[]*http.Cookie{accessTokenCookie("not-a-valid-token")})
		session, err := service.ResolveAuthorizeSession(c, downstreamClient.ClientID)
		require.NoError(t, err)
		assert.Nil(t, session)
	})

	t.Run("valid Tao session maps to client-scoped opaque subject", func(t *testing.T) {
		accessToken, err := service.MakeAccessToken(sessionClient.ClientID, "openid profile", sessionSubject, 3600, map[string]interface{}{
			types.ClaimAuthSource: types.AuthSourceTao,
		})
		require.NoError(t, err)

		c, _ := newGinTestContext(http.MethodGet, "http://server.test/v1/oauth/authorize", nil,
			[]*http.Cookie{accessTokenCookie(accessToken)})
		session, err := service.ResolveAuthorizeSession(c, downstreamClient.ClientID)
		require.NoError(t, err)
		require.NotNil(t, session)

		assert.Equal(t, userID, session.UserID)
		assert.Equal(t, sessionClient.ClientID, session.Claims.ClientID)

		expectedSubject, err := service.Subject(downstreamClient.ClientID, userID)
		require.NoError(t, err)
		assert.Equal(t, expectedSubject, session.Subject)
		assert.NotEqual(t, sessionSubject, session.Subject, "subject must be scoped to the target client")

		resolvedUserID, err := service.UserID(downstreamClient.ClientID, session.Subject)
		require.NoError(t, err)
		assert.Equal(t, userID, resolvedUserID, "subject must reverse-map to the Yao user ID")

		assert.Equal(t, types.AuthSourceTao, session.AuthSource)
		assert.Equal(t, types.IdentitySourceTao, session.IdentitySource)
	})

	t.Run("local password session maps to local identity", func(t *testing.T) {
		accessToken, err := service.MakeAccessToken(sessionClient.ClientID, "openid profile", sessionSubject, 3600, map[string]interface{}{
			types.ClaimAuthSource: types.AuthSourcePassword,
		})
		require.NoError(t, err)

		c, _ := newGinTestContext(http.MethodGet, "http://server.test/v1/oauth/authorize", nil,
			[]*http.Cookie{accessTokenCookie(accessToken)})
		session, err := service.ResolveAuthorizeSession(c, downstreamClient.ClientID)
		require.NoError(t, err)
		require.NotNil(t, session)
		assert.Equal(t, types.AuthSourcePassword, session.AuthSource)
		assert.Equal(t, types.IdentitySourceLocal, session.IdentitySource)
	})

	t.Run("authorization header session is accepted", func(t *testing.T) {
		accessToken, err := service.MakeAccessToken(sessionClient.ClientID, "openid profile", sessionSubject, 3600)
		require.NoError(t, err)

		c, _ := newGinTestContext(http.MethodGet, "http://server.test/v1/oauth/authorize",
			map[string]string{"Authorization": "Bearer " + accessToken}, nil)
		session, err := service.ResolveAuthorizeSession(c, downstreamClient.ClientID)
		require.NoError(t, err)
		require.NotNil(t, session)
		assert.Equal(t, userID, session.UserID)
		assert.Empty(t, session.AuthSource)
		assert.Equal(t, types.IdentitySourceLocal, session.IdentitySource)
	})

	t.Run("unknown subject is treated as anonymous", func(t *testing.T) {
		accessToken, err := service.MakeAccessToken(sessionClient.ClientID, "openid", "unknown-subject", 3600)
		require.NoError(t, err)

		c, _ := newGinTestContext(http.MethodGet, "http://server.test/v1/oauth/authorize", nil,
			[]*http.Cookie{accessTokenCookie(accessToken)})
		session, err := service.ResolveAuthorizeSession(c, downstreamClient.ClientID)
		require.NoError(t, err)
		assert.Nil(t, session)
	})
}

// =============================================================================
// Authorize endpoint login flow (validation -> session -> code)
// =============================================================================

func TestAuthorizeLoginFlow(t *testing.T) {
	service := newUnitTestService(t)
	client := createUnitTestClient(t, service, "authorize-client", []string{"https://client.example.com/callback"})
	ctx := context.Background()

	validRequest := func() *types.AuthorizationRequest {
		return &types.AuthorizationRequest{
			ClientID:     client.ClientID,
			ResponseType: types.ResponseTypeCode,
			RedirectURI:  "https://client.example.com/callback",
			Scope:        "openid profile",
			State:        "state-1",
		}
	}

	t.Run("invalid client is rejected before any login redirect", func(t *testing.T) {
		request := validRequest()
		request.ClientID = "does-not-exist"
		errResp := service.ValidateAuthorizationRequest(ctx, request)
		require.NotNil(t, errResp)
		assert.Equal(t, types.ErrorInvalidClient, errResp.Error)
	})

	t.Run("invalid redirect URI is rejected before any login redirect", func(t *testing.T) {
		request := validRequest()
		request.RedirectURI = "https://evil.example.com/callback"
		errResp := service.ValidateAuthorizationRequest(ctx, request)
		require.NotNil(t, errResp)
		assert.Equal(t, types.ErrorInvalidRequest, errResp.Error)
		assert.Equal(t, "Invalid redirect URI", errResp.ErrorDescription)
	})

	t.Run("invalid scope is rejected", func(t *testing.T) {
		request := validRequest()
		request.Scope = "admin"
		errResp := service.ValidateAuthorizationRequest(ctx, request)
		require.NotNil(t, errResp)
		assert.Equal(t, types.ErrorInvalidScope, errResp.Error)
	})

	t.Run("no session redirects to login page preserving the authorize URL", func(t *testing.T) {
		request := validRequest()
		require.Nil(t, service.ValidateAuthorizationRequest(ctx, request))

		authorizeURL := "http://server.test/v1/oauth/authorize?client_id=" + client.ClientID +
			"&response_type=code&redirect_uri=https%3A%2F%2Fclient.example.com%2Fcallback&state=state-1"
		c, _ := newGinTestContext(http.MethodGet, authorizeURL, nil, nil)

		session, err := service.ResolveAuthorizeSession(c, request.ClientID)
		require.NoError(t, err)
		assert.Nil(t, session, "no cookies means no Yao session")

		require.True(t, ShouldRedirectToLogin(""))
		loginURL := BuildLoginRedirectURL(DefaultLoginEntryPath, BuildAuthorizeReturnTo(c))
		parsedLogin, err := url.Parse(loginURL)
		require.NoError(t, err)
		assert.Equal(t, DefaultLoginEntryPath, parsedLogin.Path)

		returnTo, err := url.Parse(parsedLogin.Query().Get("return_to"))
		require.NoError(t, err)
		assert.Equal(t, "server.test", returnTo.Host, "return_to must be same-origin")
		assert.Equal(t, "/v1/oauth/authorize", returnTo.Path)
		assert.Equal(t, client.ClientID, returnTo.Query().Get("client_id"))
		assert.Equal(t, "state-1", returnTo.Query().Get("state"))
	})

	t.Run("prompt none without session returns login_required", func(t *testing.T) {
		assert.False(t, ShouldRedirectToLogin(PromptNone))
		assert.Equal(t, "login_required", types.ErrorLoginRequired)
		assert.Equal(t, types.ErrorLoginRequired, types.ErrLoginRequired.Code)
	})

	t.Run("session resolves subject and Authorize stores it on the code", func(t *testing.T) {
		firstParty := createUnitTestClient(t, service, "first-party-client", nil)
		userID := "user-7"
		sessionSubject, err := service.Subject(firstParty.ClientID, userID)
		require.NoError(t, err)

		accessToken, err := service.MakeAccessToken(firstParty.ClientID, "openid profile", sessionSubject, 3600, map[string]interface{}{
			types.ClaimAuthSource: types.AuthSourceTao,
		})
		require.NoError(t, err)

		c, _ := newGinTestContext(http.MethodGet, "http://server.test/v1/oauth/authorize", nil,
			[]*http.Cookie{accessTokenCookie(accessToken)})

		request := validRequest()
		session, err := service.ResolveAuthorizeSession(c, request.ClientID)
		require.NoError(t, err)
		require.NotNil(t, session)

		request.Subject = session.Subject
		request.AuthSource = session.AuthSource
		request.IdentitySource = session.IdentitySource

		resp, err := service.Authorize(ctx, request)
		require.NoError(t, err)
		require.NotNil(t, resp)
		assert.Empty(t, resp.Error)
		require.NotEmpty(t, resp.Code)
		assert.Equal(t, "state-1", resp.State)

		codeData, err := service.getAuthorizationCodeData(resp.Code)
		require.NoError(t, err)
		assert.Equal(t, session.Subject, codeData["subject"])
		assert.Equal(t, types.AuthSourceTao, codeData[types.ClaimAuthSource])
		assert.Equal(t, types.IdentitySourceTao, codeData[types.ClaimIdentitySource])
	})
}
