//go:build e2e

package user_test

import (
	"crypto/rand"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v4"
	"github.com/stretchr/testify/require"
	"github.com/yaoapp/yao/e2e/client"
	"github.com/yaoapp/yao/e2e/env"
	"github.com/yaoapp/yao/openapi"
	"github.com/yaoapp/yao/openapi/oauth"
	userpkg "github.com/yaoapp/yao/openapi/user"
)

type taoFake struct {
	server          *httptest.Server
	key             *rsa.PrivateKey
	clientID        string
	nonce           string
	activationURL   string
	activationError bool
	tokenError      bool
	badToken        bool
	keyError        bool
	keyBadJSON      bool
	keyEmpty        bool
}

func newTaoFake(t *testing.T, clientID string) *taoFake {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)
	f := &taoFake{key: key, clientID: clientID}
	mux := http.NewServeMux()
	mux.HandleFunc("/.well-known/jwks.json", func(w http.ResponseWriter, r *http.Request) {
		pub := f.key.PublicKey
		e := big.NewInt(int64(pub.E)).Bytes()
		body := map[string]interface{}{"keys": []map[string]interface{}{{
			"kid": "e2e-key", "kty": "RSA", "use": "sig", "alg": "RS256",
			"n": base64.RawURLEncoding.EncodeToString(pub.N.Bytes()),
			"e": base64.RawURLEncoding.EncodeToString(e),
		}}}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(body)
	})
	mux.HandleFunc("/oauth/activate/exchange", func(w http.ResponseWriter, r *http.Request) {
		if f.activationError {
			http.Error(w, "activation failed", http.StatusInternalServerError)
			return
		}
		var body map[string]string
		_ = json.NewDecoder(r.Body).Decode(&body)
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"client_id": "activated-client", "client_secret": "activated-secret",
			"redirect_uris": []string{
				f.activationURL + "/v1/user/oauth/tao/activate/callback",
				f.activationURL + "/v1/user/oauth/tao/callback",
			},
		})
	})
	mux.HandleFunc("/oauth/token", func(w http.ResponseWriter, r *http.Request) {
		if f.tokenError {
			http.Error(w, `{"error":"server_error"}`, http.StatusInternalServerError)
			return
		}
		require.NoError(t, r.ParseForm())
		require.Equal(t, "authorization_code", r.PostForm.Get("grant_type"))
		claims := jwt.MapClaims{
			"iss": f.server.URL, "aud": f.clientID, "sub": "tao-e2e-sub",
			"nonce": f.nonce, "exp": time.Now().Add(time.Hour).Unix(),
			"iat": time.Now().Unix(), "name": "Tao E2E User",
			"email": "tao-e2e@example.com", "email_verified": true,
		}
		token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
		token.Header["kid"] = "e2e-key"
		idToken, err := token.SignedString(f.key)
		require.NoError(t, err)
		if f.badToken {
			idToken = "not-a-valid-token"
		}
		accessClaims := jwt.MapClaims{
			"iss": f.server.URL, "aud": f.server.URL, "sub": "tao-e2e-sub",
			"client_id": f.clientID, "azp": f.clientID,
			"scope": "openid profile email offline_access keys:manage",
			"exp":   time.Now().Add(time.Hour).Unix(), "iat": time.Now().Unix(),
		}
		accessJWT := jwt.NewWithClaims(jwt.SigningMethodRS256, accessClaims)
		accessJWT.Header["kid"] = "e2e-key"
		accessToken, err := accessJWT.SignedString(f.key)
		require.NoError(t, err)
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"access_token": accessToken, "refresh_token": "tao-refresh",
			"token_type": "Bearer", "expires_in": 3600,
			"scope": "openid profile email offline_access", "id_token": idToken,
		})
	})
	mux.HandleFunc("/oauth/revoke", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	mux.HandleFunc("/api/v1/keys", func(w http.ResponseWriter, r *http.Request) {
		require.True(t, strings.HasPrefix(r.Header.Get("Authorization"), "Bearer "))
		if f.keyError {
			http.Error(w, "key unavailable", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		if f.keyBadJSON {
			_, _ = w.Write([]byte(`not-json`))
			return
		}
		if f.keyEmpty {
			_, _ = w.Write([]byte(`{}`))
			return
		}
		_, _ = w.Write([]byte(`{"key":"sk-tao-e2e-managed","item":{"id":"key_1"}}`))
	})
	f.server = httptest.NewServer(mux)
	t.Cleanup(f.server.Close)
	return f
}

func loginAgain(t *testing.T, fake *taoFake, hc *http.Client, baseURL string) {
	t.Helper()
	resp, err := hc.Get(baseURL + "/v1/user/oauth/tao/authorize")
	require.NoError(t, err)
	var auth struct {
		URL string `json:"authorization_url"`
	}
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&auth))
	_ = resp.Body.Close()
	parsed, err := url.Parse(auth.URL)
	require.NoError(t, err)
	fake.nonce = parsed.Query().Get("nonce")
	callback := baseURL + "/v1/user/oauth/tao/callback?code=again&state=" + url.QueryEscape(parsed.Query().Get("state"))
	resp, err = hc.Get(callback)
	require.NoError(t, err)
	body, _ := io.ReadAll(resp.Body)
	_ = resp.Body.Close()
	require.Equal(t, http.StatusFound, resp.StatusCode, "location=%s body=%s", resp.Header.Get("Location"), string(body))
}

func TestE2ETaoLoginFlow(t *testing.T) {
	fake := newTaoFake(t, "activated-client")
	t.Setenv("TAO_OAUTH_ISSUER", fake.server.URL)
	t.Setenv("TAO_OAUTH_CLIENT_ID", "")
	t.Setenv("TAO_OAUTH_CLIENT_SECRET", "")
	environment, err := env.New(t)
	require.NoError(t, err)
	fake.activationURL = environment.BaseURL

	// Entry exposes local login (test app) and the Tao provider.
	httpClient := client.New(environment.BaseURL)
	resp, body, err := httpClient.Get("/v1/user/entry?locale=en")
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode, string(body))
	var entry struct {
		LocalLoginEnabled bool `json:"local_login_enabled"`
	}
	require.NoError(t, json.Unmarshal(body, &entry))
	require.True(t, entry.LocalLoginEnabled)
	resp, body, err = httpClient.Get("/v1/user/oauth/providers/available")
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode, string(body))
	require.Contains(t, string(body), `"tao"`)
	resp, body, err = httpClient.Get("/.well-known/openid-configuration")
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode, string(body))
	require.NotContains(t, string(body), "end_session_endpoint")
	require.Contains(t, string(body), "authorization_endpoint")

	// First-party preset client: PKCE authorize without a session goes to login.
	presetURL := environment.BaseURL + "/v1/oauth/authorize?client_id=yao_e2e_web_client_0123456789ab" +
		"&redirect_uri=" + url.QueryEscape("https://app.e2e.local/callback") +
		"&response_type=code&state=preset-state" +
		"&code_challenge=test-challenge&code_challenge_method=S256"
	presetClient := &http.Client{Timeout: 10 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	presetResp, err := presetClient.Get(presetURL)
	require.NoError(t, err)
	presetBody, _ := io.ReadAll(presetResp.Body)
	_ = presetResp.Body.Close()
	require.Equal(t, http.StatusFound, presetResp.StatusCode, "location=%s body=%s", presetResp.Header.Get("Location"), string(presetBody))
	require.Contains(t, presetResp.Header.Get("Location"), "/auth/entry")
	require.Contains(t, presetResp.Header.Get("Location"), "return_to=")

	jar, err := cookiejar.New(nil)
	require.NoError(t, err)
	authClient := &http.Client{Jar: jar, Timeout: 20 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	resp, err = authClient.Get(environment.BaseURL + "/v1/user/oauth/tao/authorize")
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode)
	var activation struct {
		URL string `json:"authorization_url"`
	}
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&activation))
	_ = resp.Body.Close()
	activationParsed, err := url.Parse(activation.URL)
	require.NoError(t, err)
	require.Contains(t, activation.URL, "/oauth/activate?")
	require.Equal(t, "S256", activationParsed.Query().Get("code_challenge_method"))
	require.NotEmpty(t, activationParsed.Query().Get("deployment_public_key"))
	require.Contains(t, activationParsed.Query()["login_redirect_uris"], environment.BaseURL+"/v1/user/oauth/tao/callback")
	activationState := activationParsed.Query().Get("state")

	activationCallback := environment.BaseURL + "/v1/user/oauth/tao/activate/callback?activation_code=activation-code&state=" + url.QueryEscape(activationState)
	resp, err = authClient.Get(activationCallback)
	require.NoError(t, err)
	body, _ = io.ReadAll(resp.Body)
	_ = resp.Body.Close()
	require.Equal(t, http.StatusFound, resp.StatusCode, "location=%s body=%s", resp.Header.Get("Location"), string(body))
	loginURL := resp.Header.Get("Location")
	require.Contains(t, loginURL, fake.server.URL+"/oauth/authorize")
	require.Contains(t, loginURL, "client_id=activated-client")
	loginParsed, err := url.Parse(loginURL)
	require.NoError(t, err)
	state := loginParsed.Query().Get("state")
	fake.nonce = loginParsed.Query().Get("nonce")
	require.NotEmpty(t, state)
	require.NotEmpty(t, fake.nonce)

	callback := environment.BaseURL + "/v1/user/oauth/tao/callback?code=e2e-code&state=" + url.QueryEscape(state)
	resp, err = authClient.Get(callback)
	require.NoError(t, err)
	body, _ = io.ReadAll(resp.Body)
	_ = resp.Body.Close()
	require.Equal(t, http.StatusFound, resp.StatusCode, "location=%s body=%s", resp.Header.Get("Location"), string(body))
	require.Equal(t, "/auth/entry/success", resp.Header.Get("Location"))

	// Replaying a callback must fail without mutating state.
	resp, err = authClient.Get(callback)
	require.NoError(t, err)
	_ = resp.Body.Close()
	require.Equal(t, http.StatusFound, resp.StatusCode)
	require.Contains(t, resp.Header.Get("Location"), "invalid_state")

	service, ok := openapi.Server.OAuth.(*oauth.Service)
	require.True(t, ok)
	provider, err := service.GetUserProvider()
	require.NoError(t, err)
	userID, err := provider.GetOAuthUserID(t.Context(), "tao", "tao-e2e-sub")
	require.NoError(t, err)
	user, err := provider.GetUser(t.Context(), userID)
	require.NoError(t, err)
	require.Equal(t, "tao", fmt.Sprint(user["identity_source"]))
	authUser, err := provider.GetUserForAuth(t.Context(), userID, "user_id")
	require.NoError(t, err)
	require.NotEmpty(t, fmt.Sprint(authUser["tao_api_key"]))
	account, err := provider.GetOAuthAccount(t.Context(), "tao", "tao-e2e-sub")
	require.NoError(t, err)
	require.Equal(t, "tao", fmt.Sprint(account["provider"]))
	require.NotEmpty(t, fmt.Sprint(account["refresh_token"]))
	require.True(t, strings.HasPrefix(fmt.Sprint(authUser["tao_api_key"]), "sk-tao-"))

	// RequireTaoIdentity allows Tao identity and rejects local identity.
	recorder := httptest.NewRecorder()
	gateCtx, _ := gin.CreateTestContext(recorder)
	gateCtx.Request = httptest.NewRequest(http.MethodGet, "/", nil)
	gateCtx.Set("__user_id", userID)
	gateCtx.Set("__auth_source", "tao")
	userpkg.RequireTaoIdentity(gateCtx)
	require.False(t, gateCtx.IsAborted())

	// Token endpoint and ID-token failures are surfaced as callback failures.
	fake.tokenError = true
	resp, err = authClient.Get(environment.BaseURL + "/v1/user/oauth/tao/authorize")
	require.NoError(t, err)
	var retryAuth struct {
		URL string `json:"authorization_url"`
	}
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&retryAuth))
	_ = resp.Body.Close()
	retryURL, err := url.Parse(retryAuth.URL)
	require.NoError(t, err)
	resp, err = authClient.Get(environment.BaseURL + "/v1/user/oauth/tao/callback?code=bad&state=" + url.QueryEscape(retryURL.Query().Get("state")))
	require.NoError(t, err)
	_ = resp.Body.Close()
	require.Contains(t, resp.Header.Get("Location"), "server_error")
	fake.tokenError = false

	fake.badToken = true
	resp, err = authClient.Get(environment.BaseURL + "/v1/user/oauth/tao/authorize")
	require.NoError(t, err)
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&retryAuth))
	_ = resp.Body.Close()
	retryURL, err = url.Parse(retryAuth.URL)
	require.NoError(t, err)
	resp, err = authClient.Get(environment.BaseURL + "/v1/user/oauth/tao/callback?code=bad-token&state=" + url.QueryEscape(retryURL.Query().Get("state")))
	require.NoError(t, err)
	_ = resp.Body.Close()
	require.Contains(t, resp.Header.Get("Location"), "server_error")
	fake.badToken = false

	// Activation exchange failure is surfaced without leaving a partial client.
	require.NoError(t, os.Remove(filepath.Join(environment.Root, "app", ".tao", "client.json")))
	fake.activationError = true
	resp, err = authClient.Get(environment.BaseURL + "/v1/user/oauth/tao/authorize")
	require.NoError(t, err)
	var retryActivation struct {
		URL string `json:"authorization_url"`
	}
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&retryActivation))
	_ = resp.Body.Close()
	retryParsed, err := url.Parse(retryActivation.URL)
	require.NoError(t, err)
	resp, err = authClient.Get(environment.BaseURL + "/v1/user/oauth/tao/activate/callback?activation_code=bad&state=" + url.QueryEscape(retryParsed.Query().Get("state")))
	require.NoError(t, err)
	_ = resp.Body.Close()
	require.Contains(t, resp.Header.Get("Location"), "tao_activation_failed")
	fake.activationError = false

	// Managed-key failure modes keep login successful but log the provisioning error.
	require.NoError(t, provider.UpdateUser(t.Context(), userID, map[string]interface{}{"tao_api_key": ""}))
	for _, tc := range []struct {
		name string
		set  func(bool)
	}{
		{"key-error", func(v bool) { fake.keyError = v }},
		{"key-bad-json", func(v bool) { fake.keyBadJSON = v }},
		{"key-empty", func(v bool) { fake.keyEmpty = v }},
	} {
		tc.set(true)
		loginAgain(t, fake, authClient, environment.BaseURL)
		tc.set(false)
	}
	// Existing key path: second successful login must not request a new plaintext key.
	loginAgain(t, fake, authClient, environment.BaseURL)

	admin, err := provider.GetUserByEmail(t.Context(), "admin@e2e.local")
	require.NoError(t, err)
	recorder = httptest.NewRecorder()
	gateCtx, _ = gin.CreateTestContext(recorder)
	gateCtx.Request = httptest.NewRequest(http.MethodGet, "/", nil)
	gateCtx.Set("__user_id", fmt.Sprint(admin["user_id"]))
	gateCtx.Set("__auth_source", "password")
	userpkg.RequireTaoIdentity(gateCtx)
	require.Equal(t, http.StatusForbidden, recorder.Code)
	require.Contains(t, recorder.Body.String(), "tao_identity_required")

	// Missing identity is unauthorized; a missing OAuth service is unavailable.
	recorder = httptest.NewRecorder()
	gateCtx, _ = gin.CreateTestContext(recorder)
	gateCtx.Request = httptest.NewRequest(http.MethodGet, "/", nil)
	userpkg.RequireTaoIdentity(gateCtx)
	require.Equal(t, http.StatusUnauthorized, recorder.Code)
	oldOAuth := oauth.OAuth
	oauth.OAuth = nil
	recorder = httptest.NewRecorder()
	gateCtx, _ = gin.CreateTestContext(recorder)
	gateCtx.Request = httptest.NewRequest(http.MethodGet, "/", nil)
	gateCtx.Set("__user_id", userID)
	gateCtx.Set("__auth_source", "password")
	userpkg.RequireTaoIdentity(gateCtx)
	require.Equal(t, http.StatusServiceUnavailable, recorder.Code)
	oauth.OAuth = oldOAuth

	adminLogin, err := userpkg.LoginByUserID(fmt.Sprint(admin["user_id"]), nil)
	require.NoError(t, err)
	profileClient := client.New(environment.BaseURL)
	profileClient.Token = adminLogin.AccessToken
	resp, profileBody, err := profileClient.Get("/v1/user/profile")
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode, string(profileBody))
	require.Contains(t, string(profileBody), "identity_source")
}
