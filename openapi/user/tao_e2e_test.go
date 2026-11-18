//go:build e2e

package user_test

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
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
	mux.HandleFunc("/v1/key", func(w http.ResponseWriter, r *http.Request) {
		require.True(t, strings.HasPrefix(r.Header.Get("Authorization"), "Bearer sk-tao-"))
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"object":"key","valid":true,"balance":4052,"balance_available":true}`))
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
	require.Contains(t, string(body), "end_session_endpoint")
	require.Contains(t, string(body), "prompt_values_supported")
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

	// The account menu reads the current plan and Tao balance through Yao.
	usageReq, usageReqErr := http.NewRequest(http.MethodGet, environment.BaseURL+"/v1/user/usage", nil)
	require.NoError(t, usageReqErr)
	usageResp, err := authClient.Do(usageReq)
	require.NoError(t, err)
	usageBody, _ := io.ReadAll(usageResp.Body)
	_ = usageResp.Body.Close()
	require.Equal(t, http.StatusOK, usageResp.StatusCode, string(usageBody))
	var usage struct {
		Plan             string `json:"plan"`
		Remaining        *int64 `json:"remaining"`
		Unit             string `json:"unit"`
		BalanceAvailable bool   `json:"balance_available"`
	}
	require.NoError(t, json.Unmarshal(usageBody, &usage))
	require.Equal(t, "free", usage.Plan)
	require.Equal(t, "credits", usage.Unit)
	require.True(t, usage.BalanceAvailable)
	require.NotNil(t, usage.Remaining)
	require.Equal(t, int64(4052), *usage.Remaining)

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

	// prompt=login must be forwarded to Tao so Tao can force reauthentication;
	// login_hint / max_age are forwarded too.
	reauthClient := &http.Client{
		Timeout:       20 * time.Second,
		CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse },
	}
	reauthURL := environment.BaseURL + "/v1/oauth/authorize?client_id=yao_e2e_desktop_tao" +
		"&redirect_uri=" + url.QueryEscape("http://127.0.0.1:49152/callback") +
		"&response_type=code&state=reauth-state&prompt=login" +
		"&login_hint=" + url.QueryEscape("another@example.com") + "&max_age=0" +
		"&code_challenge=reauth&code_challenge_method=S256"
	reauthResp, reauthErr := reauthClient.Get(reauthURL)
	require.NoError(t, reauthErr)
	_ = reauthResp.Body.Close()
	require.Equal(t, http.StatusFound, reauthResp.StatusCode)
	taoStartURL := reauthResp.Header.Get("Location")
	require.Contains(t, taoStartURL, "/v1/user/oauth/tao/authorize")

	taoResp, taoErr := reauthClient.Get(environment.BaseURL + taoStartURL)
	require.NoError(t, taoErr)
	_ = taoResp.Body.Close()
	require.Equal(t, http.StatusFound, taoResp.StatusCode)
	taoParsedURL, taoParseErr := url.Parse(taoResp.Header.Get("Location"))
	require.NoError(t, taoParseErr)
	require.Equal(t, "login", taoParsedURL.Query().Get("prompt"))
	require.Equal(t, "another@example.com", taoParsedURL.Query().Get("login_hint"))
	require.Equal(t, "0", taoParsedURL.Query().Get("max_age"))

	// RFC 8252 loopback ignores the port, and the celadon custom scheme is
	// accepted by the registered mobile client.
	noRedirectProbe := &http.Client{
		Timeout:       20 * time.Second,
		CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse },
	}
	loopbackProbe := environment.BaseURL + "/v1/oauth/authorize?client_id=yao_e2e_desktop_tao" +
		"&redirect_uri=" + url.QueryEscape("http://127.0.0.1:52341/callback") +
		"&response_type=code&state=loopback-probe&code_challenge=probe&code_challenge_method=S256"
	loopbackProbeResp, loopbackProbeErr := noRedirectProbe.Get(loopbackProbe)
	require.NoError(t, loopbackProbeErr)
	_ = loopbackProbeResp.Body.Close()
	require.Equal(t, http.StatusFound, loopbackProbeResp.StatusCode)

	customSchemeProbe := environment.BaseURL + "/v1/oauth/authorize?client_id=yao_e2e_mobile_client_0123456789" +
		"&redirect_uri=" + url.QueryEscape("celadon://callback") +
		"&response_type=code&state=scheme-probe&code_challenge=probe&code_challenge_method=S256"
	customSchemeResp, customSchemeErr := noRedirectProbe.Get(customSchemeProbe)
	require.NoError(t, customSchemeErr)
	_ = customSchemeResp.Body.Close()
	require.Equal(t, http.StatusFound, customSchemeResp.StatusCode)

	// Tao-only native flow: no login page, cookie-less handoff.
	nativeNoRedirect := &http.Client{
		Timeout:       20 * time.Second,
		CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse },
	}
	nativeVerifier := "test-verifier"
	nativeSum := sha256.Sum256([]byte(nativeVerifier))
	nativeChallenge := base64.RawURLEncoding.EncodeToString(nativeSum[:])
	nativeAuthorizeURL := environment.BaseURL + "/v1/oauth/authorize?client_id=yao_e2e_desktop_tao" +
		"&redirect_uri=" + url.QueryEscape("http://127.0.0.1:49152/callback") +
		"&response_type=code&state=native-direct-state" +
		"&code_challenge=" + url.QueryEscape(nativeChallenge) + "&code_challenge_method=S256"
	nativeResp, nativeErr := nativeNoRedirect.Get(nativeAuthorizeURL)
	require.NoError(t, nativeErr)
	_ = nativeResp.Body.Close()
	require.Equal(t, http.StatusFound, nativeResp.StatusCode)
	nativeTaoLoginURL := nativeResp.Header.Get("Location")
	require.Contains(t, nativeTaoLoginURL, "/v1/user/oauth/tao/authorize?redirect=1")

	nativeResp, nativeErr = nativeNoRedirect.Get(environment.BaseURL + nativeTaoLoginURL)
	require.NoError(t, nativeErr)
	_ = nativeResp.Body.Close()
	require.Equal(t, http.StatusFound, nativeResp.StatusCode)
	nativeTaoURL := nativeResp.Header.Get("Location")
	nativeParsed, nativeErr := url.Parse(nativeTaoURL)
	require.NoError(t, nativeErr)
	nativeState := nativeParsed.Query().Get("state")
	require.NotEmpty(t, nativeState)
	fake.nonce = nativeParsed.Query().Get("nonce")

	nativeCallback := environment.BaseURL + "/v1/user/oauth/tao/callback?code=native-code&state=" + url.QueryEscape(nativeState)
	nativeResp, nativeErr = nativeNoRedirect.Get(nativeCallback)
	require.NoError(t, nativeErr)
	_ = nativeResp.Body.Close()
	require.Equal(t, http.StatusFound, nativeResp.StatusCode)
	nativeResumeURL := nativeResp.Header.Get("Location")
	require.Contains(t, nativeResumeURL, "login_handoff=")

	// Deliberately use a cookie-less client: the server-side handoff must
	// establish the session for this one authorize request. The request must
	// now stop at the consent page before issuing a code.
	nativeResp, nativeErr = nativeNoRedirect.Get(nativeResumeURL)
	require.NoError(t, nativeErr)
	_ = nativeResp.Body.Close()
	require.Equal(t, http.StatusFound, nativeResp.StatusCode)
	nativeConsentURL := nativeResp.Header.Get("Location")
	require.Contains(t, nativeConsentURL, "/app/consent")
	nativeParsedConsent, nativeErr := url.Parse(nativeConsentURL)
	require.NoError(t, nativeErr)
	nativeConsentToken := nativeParsedConsent.Query().Get("consent_token")
	require.NotEmpty(t, nativeConsentToken)

	nativeConfirmForm := url.Values{
		"client_id":             {"yao_e2e_desktop_tao"},
		"redirect_uri":          {"http://127.0.0.1:49152/callback"},
		"response_type":         {"code"},
		"state":                 {"native-direct-state"},
		"code_challenge":        {nativeChallenge},
		"code_challenge_method": {"S256"},
		"consent_token":         {nativeConsentToken},
	}
	nativeConfirmReq, nativeErr := http.NewRequest(http.MethodPost, environment.BaseURL+"/v1/oauth/authorize", strings.NewReader(nativeConfirmForm.Encode()))
	require.NoError(t, nativeErr)
	nativeConfirmReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	nativeConfirmResp, nativeErr := nativeNoRedirect.Do(nativeConfirmReq)
	require.NoError(t, nativeErr)
	_ = nativeConfirmResp.Body.Close()
	require.Equal(t, http.StatusFound, nativeConfirmResp.StatusCode)
	nativeLoopback := nativeConfirmResp.Header.Get("Location")
	require.Contains(t, nativeLoopback, "127.0.0.1:49152/callback")
	require.Contains(t, nativeLoopback, "code=")
	nativeParsedLoopback, nativeErr := url.Parse(nativeLoopback)
	require.NoError(t, nativeErr)
	nativeCode := nativeParsedLoopback.Query().Get("code")
	require.NotEmpty(t, nativeCode)
	require.Equal(t, "native-direct-state", nativeParsedLoopback.Query().Get("state"))

	// The official desktop client scope must be applied when the authorize
	// request omits scope, and the public client must exchange the code without
	// a secret.
	nativeTokenResp, nativeErr := http.PostForm(environment.BaseURL+"/v1/oauth/token", url.Values{
		"grant_type":    {"authorization_code"},
		"code":          {nativeCode},
		"redirect_uri":  {"http://127.0.0.1:49152/callback"},
		"client_id":     {"yao_e2e_desktop_tao"},
		"code_verifier": {nativeVerifier},
	})
	require.NoError(t, nativeErr)
	var nativeTokenBody struct {
		AccessToken  string `json:"access_token"`
		IDToken      string `json:"id_token"`
		RefreshToken string `json:"refresh_token"`
	}
	require.NoError(t, json.NewDecoder(nativeTokenResp.Body).Decode(&nativeTokenBody))
	_ = nativeTokenResp.Body.Close()
	require.Equal(t, http.StatusOK, nativeTokenResp.StatusCode)
	require.NotEmpty(t, nativeTokenBody.AccessToken)
	require.NotEmpty(t, nativeTokenBody.IDToken)
	require.NotEmpty(t, nativeTokenBody.RefreshToken)

	nativeProfileReq, nativeErr := http.NewRequest(http.MethodGet, environment.BaseURL+"/v1/user/profile", nil)
	require.NoError(t, nativeErr)
	nativeProfileReq.Header.Set("Authorization", "Bearer "+nativeTokenBody.AccessToken)
	nativeProfileResp, nativeErr := nativeNoRedirect.Do(nativeProfileReq)
	require.NoError(t, nativeErr)
	nativeProfileBody, _ := io.ReadAll(nativeProfileResp.Body)
	_ = nativeProfileResp.Body.Close()
	require.Equal(t, http.StatusOK, nativeProfileResp.StatusCode, string(nativeProfileBody))

	// Team selection through the authorize extension + consent page.
	teamsReq, teamsErr := http.NewRequest(http.MethodGet, environment.BaseURL+"/v1/user/teams", nil)
	require.NoError(t, teamsErr)
	teamsReq.Header.Set("Authorization", "Bearer "+nativeTokenBody.AccessToken)
	teamsResp, teamsErr := nativeNoRedirect.Do(teamsReq)
	require.NoError(t, teamsErr)
	teamsBody, _ := io.ReadAll(teamsResp.Body)
	_ = teamsResp.Body.Close()
	require.Equal(t, http.StatusOK, teamsResp.StatusCode, string(teamsBody))
	var nativeTeams []map[string]interface{}
	require.NoError(t, json.Unmarshal(teamsBody, &nativeTeams))
	require.NotEmpty(t, nativeTeams)
	nativeTeamID, _ := nativeTeams[0]["team_id"].(string)
	require.NotEmpty(t, nativeTeamID)

	teamVerifier := "team-verifier"
	teamSum := sha256.Sum256([]byte(teamVerifier))
	teamChallenge := base64.RawURLEncoding.EncodeToString(teamSum[:])
	teamAuthorizeURL := environment.BaseURL + "/v1/oauth/authorize?client_id=yao_e2e_desktop_tao" +
		"&redirect_uri=" + url.QueryEscape("http://127.0.0.1:49152/callback") +
		"&response_type=code&state=team-state" +
		"&code_challenge=" + url.QueryEscape(teamChallenge) + "&code_challenge_method=S256"
	teamAuthResp, teamAuthErr := authClient.Get(teamAuthorizeURL)
	require.NoError(t, teamAuthErr)
	_ = teamAuthResp.Body.Close()
	require.Equal(t, http.StatusFound, teamAuthResp.StatusCode)
	teamConsentURL := teamAuthResp.Header.Get("Location")
	require.Contains(t, teamConsentURL, "/app/consent")
	teamParsedConsent, teamParseErr := url.Parse(teamConsentURL)
	require.NoError(t, teamParseErr)
	teamConsentToken := teamParsedConsent.Query().Get("consent_token")
	require.NotEmpty(t, teamConsentToken)

	teamConfirmForm := url.Values{
		"client_id":             {"yao_e2e_desktop_tao"},
		"redirect_uri":          {"http://127.0.0.1:49152/callback"},
		"response_type":         {"code"},
		"state":                 {"team-state"},
		"team_id":               {nativeTeamID},
		"code_challenge":        {teamChallenge},
		"code_challenge_method": {"S256"},
		"consent_token":         {teamConsentToken},
	}
	teamConfirmReq, teamConfirmErr := http.NewRequest(http.MethodPost, environment.BaseURL+"/v1/oauth/authorize", strings.NewReader(teamConfirmForm.Encode()))
	require.NoError(t, teamConfirmErr)
	teamConfirmReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	teamConfirmResp, teamConfirmErr := authClient.Do(teamConfirmReq)
	require.NoError(t, teamConfirmErr)
	_ = teamConfirmResp.Body.Close()
	require.Equal(t, http.StatusFound, teamConfirmResp.StatusCode)
	teamLoopback := teamConfirmResp.Header.Get("Location")
	teamParsedLoopback, teamLoopbackErr := url.Parse(teamLoopback)
	require.NoError(t, teamLoopbackErr)
	teamCode := teamParsedLoopback.Query().Get("code")
	require.NotEmpty(t, teamCode)

	teamTokenResp, teamTokenErr := http.PostForm(environment.BaseURL+"/v1/oauth/token", url.Values{
		"grant_type":    {"authorization_code"},
		"code":          {teamCode},
		"redirect_uri":  {"http://127.0.0.1:49152/callback"},
		"client_id":     {"yao_e2e_desktop_tao"},
		"code_verifier": {teamVerifier},
	})
	require.NoError(t, teamTokenErr)
	var teamTokenBody struct {
		AccessToken string `json:"access_token"`
	}
	require.NoError(t, json.NewDecoder(teamTokenResp.Body).Decode(&teamTokenBody))
	_ = teamTokenResp.Body.Close()
	require.Equal(t, http.StatusOK, teamTokenResp.StatusCode)
	require.NotEmpty(t, teamTokenBody.AccessToken)
	teamParts := strings.Split(teamTokenBody.AccessToken, ".")
	require.Len(t, teamParts, 3)
	teamPayload, teamPayloadErr := base64.RawURLEncoding.DecodeString(teamParts[1])
	require.NoError(t, teamPayloadErr)
	var teamClaims map[string]interface{}
	require.NoError(t, json.Unmarshal(teamPayload, &teamClaims))
	require.Equal(t, nativeTeamID, teamClaims["team_id"])

	teamProfileReq, teamProfileErr := http.NewRequest(http.MethodGet, environment.BaseURL+"/v1/user/profile?team=true", nil)
	require.NoError(t, teamProfileErr)
	teamProfileReq.Header.Set("Authorization", "Bearer "+teamTokenBody.AccessToken)
	teamProfileResp, teamProfileErr := nativeNoRedirect.Do(teamProfileReq)
	require.NoError(t, teamProfileErr)
	teamProfileBody, _ := io.ReadAll(teamProfileResp.Body)
	_ = teamProfileResp.Body.Close()
	require.Equal(t, http.StatusOK, teamProfileResp.StatusCode, string(teamProfileBody))
	require.Contains(t, string(teamProfileBody), "yao:team")

	// A POSTed team_id that does not belong to the user must fail with
	// access_denied before any code is issued.
	badTeamVerifier := "bad-team-verifier"
	badTeamSum := sha256.Sum256([]byte(badTeamVerifier))
	badTeamChallenge := base64.RawURLEncoding.EncodeToString(badTeamSum[:])
	badTeamAuthorizeURL := environment.BaseURL + "/v1/oauth/authorize?client_id=yao_e2e_desktop_tao" +
		"&redirect_uri=" + url.QueryEscape("http://127.0.0.1:49152/callback") +
		"&response_type=code&state=bad-team-state" +
		"&code_challenge=" + url.QueryEscape(badTeamChallenge) + "&code_challenge_method=S256"
	badTeamAuthResp, badTeamAuthErr := authClient.Get(badTeamAuthorizeURL)
	require.NoError(t, badTeamAuthErr)
	_ = badTeamAuthResp.Body.Close()
	require.Equal(t, http.StatusFound, badTeamAuthResp.StatusCode)
	badTeamConsentURL := badTeamAuthResp.Header.Get("Location")
	badTeamParsedConsent, badTeamParseErr := url.Parse(badTeamConsentURL)
	require.NoError(t, badTeamParseErr)
	badTeamConsentToken := badTeamParsedConsent.Query().Get("consent_token")
	require.NotEmpty(t, badTeamConsentToken)

	badTeamConfirmForm := url.Values{
		"client_id":             {"yao_e2e_desktop_tao"},
		"redirect_uri":          {"http://127.0.0.1:49152/callback"},
		"response_type":         {"code"},
		"state":                 {"bad-team-state"},
		"team_id":               {"999999999999"},
		"code_challenge":        {badTeamChallenge},
		"code_challenge_method": {"S256"},
		"consent_token":         {badTeamConsentToken},
	}
	badTeamConfirmReq, badTeamConfirmErr := http.NewRequest(http.MethodPost, environment.BaseURL+"/v1/oauth/authorize", strings.NewReader(badTeamConfirmForm.Encode()))
	require.NoError(t, badTeamConfirmErr)
	badTeamConfirmReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	badTeamConfirmResp, badTeamConfirmErr := authClient.Do(badTeamConfirmReq)
	require.NoError(t, badTeamConfirmErr)
	_ = badTeamConfirmResp.Body.Close()
	require.Equal(t, http.StatusFound, badTeamConfirmResp.StatusCode)
	badTeamLocation := badTeamConfirmResp.Header.Get("Location")
	require.Contains(t, badTeamLocation, "error=access_denied")
	require.NotContains(t, badTeamLocation, "code=")

	teamSelectBody := `{"team_id":"` + nativeTeamID + `"}`
	teamSelectReq, teamSelectErr := http.NewRequest(http.MethodPost, environment.BaseURL+"/v1/user/teams/select", strings.NewReader(teamSelectBody))
	require.NoError(t, teamSelectErr)
	teamSelectReq.Header.Set("Authorization", "Bearer "+teamTokenBody.AccessToken)
	teamSelectReq.Header.Set("Content-Type", "application/json")
	teamSelectResp, teamSelectErr := nativeNoRedirect.Do(teamSelectReq)
	require.NoError(t, teamSelectErr)
	teamSelectBodyResp, _ := io.ReadAll(teamSelectResp.Body)
	_ = teamSelectResp.Body.Close()
	require.Equal(t, http.StatusOK, teamSelectResp.StatusCode, string(teamSelectBodyResp))

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
	// Logout must be callable with the scope issued to the first-party client.
	// The login session must be able to call logout. The ACL maps
	// POST /user/logout to user:entry:write:own, so that scope must be
	// present in the token issued for the first-party client.
	baseURL, err := url.Parse(environment.BaseURL)
	require.NoError(t, err)
	var accessToken string
	for _, cookie := range jar.Cookies(baseURL) {
		if cookie.Name == "access_token" {
			accessToken = strings.TrimPrefix(cookie.Value, "Bearer ")
			break
		}
	}
	require.NotEmpty(t, accessToken)
	parts := strings.Split(accessToken, ".")
	require.Len(t, parts, 3)
	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	require.NoError(t, err)
	var tokenClaims map[string]interface{}
	require.NoError(t, json.Unmarshal(payload, &tokenClaims))
	scope, _ := tokenClaims["scope"].(string)
	require.Contains(t, scope, "user:entry:write:own")

	// A session cookie with a team claim must be able to list teams for the
	// consent team picker; the team-selection endpoints bypass team-role scope
	// enforcement after the token scope check.
	teamsCookieResp, teamsCookieErr := authClient.Get(environment.BaseURL + "/v1/user/teams")
	require.NoError(t, teamsCookieErr)
	teamsCookieBody, _ := io.ReadAll(teamsCookieResp.Body)
	_ = teamsCookieResp.Body.Close()
	require.Equal(t, http.StatusOK, teamsCookieResp.StatusCode, string(teamsCookieBody))

	logoutReq, err := http.NewRequest(http.MethodPost, environment.BaseURL+"/v1/user/logout", strings.NewReader("{}"))
	require.NoError(t, err)
	logoutReq.Header.Set("Content-Type", "application/json")
	resp, err = authClient.Do(logoutReq)
	require.NoError(t, err)
	_, _ = io.Copy(io.Discard, resp.Body)
	_ = resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode)

	resp, err = authClient.Get(environment.BaseURL + "/v1/user/profile")
	require.NoError(t, err)
	_ = resp.Body.Close()
	require.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	require.True(t, strings.HasPrefix(fmt.Sprint(authUser["tao_api_key"]), "sk-tao-"))

	// OIDC RP-Initiated Logout must redirect to the registered post-logout URI.
	endSessionClient := &http.Client{
		Timeout:       20 * time.Second,
		CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse },
	}
	endSessionURL := environment.BaseURL + "/v1/oauth/logout?client_id=yao_e2e_desktop_tao" +
		"&post_logout_redirect_uri=" + url.QueryEscape("http://127.0.0.1:49152/callback") +
		"&state=end-session-state"
	endSessionResp, endSessionErr := endSessionClient.Get(endSessionURL)
	require.NoError(t, endSessionErr)
	_ = endSessionResp.Body.Close()
	require.Equal(t, http.StatusFound, endSessionResp.StatusCode)
	require.Contains(t, endSessionResp.Header.Get("Location"), "127.0.0.1:49152/callback")
	require.Contains(t, endSessionResp.Header.Get("Location"), "state=end-session-state")

}
