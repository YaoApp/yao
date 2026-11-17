package user

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v4"
	"github.com/stretchr/testify/require"
	"github.com/yaoapp/gou/store"
	"github.com/yaoapp/gou/store/lru"
	"github.com/yaoapp/kun/maps"
	"github.com/yaoapp/yao/config"
	"github.com/yaoapp/yao/openapi/oauth"
	oauthtypes "github.com/yaoapp/yao/openapi/oauth/types"
)

type fakeTaoUserProvider struct {
	oauthtypes.UserProvider
	accounts    []maps.MapStrAny
	accountsErr error
	user        maps.MapStrAny
	userErr     error
	authUser    maps.MapStrAny
	authErr     error
	updateErr   error
}

func TestTaoPKCE(t *testing.T) {
	verifier, challenge, err := taoPKCE()
	require.NoError(t, err)
	require.NotEmpty(t, verifier)
	sum := sha256.Sum256([]byte(verifier))
	require.Equal(t, base64.RawURLEncoding.EncodeToString(sum[:]), challenge)
}

func TestTaoValidateClaims(t *testing.T) {
	claims := map[string]interface{}{"iss": "https://tao.example", "aud": "client", "nonce": "n"}
	require.NoError(t, taoValidateClaims(claims, "https://tao.example", "n", "client"))
	require.Error(t, taoValidateClaims(claims, "https://other", "n", "client"))
	require.Error(t, taoValidateClaims(claims, "https://tao.example", "bad", "client"))
	require.Error(t, taoValidateClaims(claims, "https://tao.example", "n", "other"))
	claims["aud"] = []interface{}{"other", "client"}
	require.NoError(t, taoValidateClaims(claims, "https://tao.example", "n", "client"))
	claims["aud"] = []interface{}{"other"}
	require.Error(t, taoValidateClaims(claims, "https://tao.example", "n", "client"))
}

func TestTaoValidateActivationRedirects(t *testing.T) {
	activation := "https://yao.example/v1/user/oauth/tao/activate/callback"
	login := "https://yao.example/v1/user/oauth/tao/callback"
	require.NoError(t, taoValidateActivationRedirects([]string{activation, login}, activation, []string{login}))
	require.Error(t, taoValidateActivationRedirects([]string{activation}, activation, []string{login}))
	require.Error(t, taoValidateActivationRedirects([]string{login}, activation, []string{login}))
}

func TestTaoDeploymentEnv(t *testing.T) {
	t.Setenv("TAO_DEPLOYMENT_ID", "dep-1")
	t.Setenv("TAO_DEPLOYMENT_PUBLIC_KEY", "PEM")
	d, err := loadTaoDeployment()
	require.NoError(t, err)
	require.Equal(t, "dep-1", d.ID)
	require.Equal(t, "PEM", d.PublicKey)
}

func TestTaoDeploymentFile(t *testing.T) {
	oldRoot := config.Conf.Root
	config.Conf.Root = t.TempDir()
	t.Cleanup(func() { config.Conf.Root = oldRoot })
	t.Setenv("TAO_DEPLOYMENT_ID", "")
	t.Setenv("TAO_DEPLOYMENT_PUBLIC_KEY", "")
	d, err := loadTaoDeployment()
	require.NoError(t, err)
	require.NotEmpty(t, d.ID)
	require.Contains(t, d.PublicKey, "BEGIN PUBLIC KEY")
	d2, err := loadTaoDeployment()
	require.NoError(t, err)
	require.Equal(t, d.ID, d2.ID)
	require.Equal(t, d.PublicKey, d2.PublicKey)
}

func TestTaoClientCredentials(t *testing.T) {
	t.Setenv("TAO_OAUTH_CLIENT_ID", "cid")
	t.Setenv("TAO_OAUTH_CLIENT_SECRET", "secret")
	c, err := loadTaoClientCredentials()
	require.NoError(t, err)
	require.True(t, c.complete())
	require.Equal(t, "cid", c.ClientID)
	t.Setenv("TAO_OAUTH_CLIENT_SECRET", "")
	_, err = loadTaoClientCredentials()
	require.Error(t, err)
}

func TestTaoClientStore(t *testing.T) {
	oldRoot := config.Conf.Root
	config.Conf.Root = t.TempDir()
	t.Cleanup(func() { config.Conf.Root = oldRoot })
	t.Setenv("TAO_OAUTH_CLIENT_ID", "")
	t.Setenv("TAO_OAUTH_CLIENT_SECRET", "")
	c, err := loadTaoClientCredentials()
	require.NoError(t, err)
	require.False(t, c.complete())
	require.NoError(t, saveTaoClientCredentials("cid", "secret"))
	c, err = loadTaoClientCredentials()
	require.NoError(t, err)
	require.True(t, c.complete())
	info, err := os.Stat(filepath.Join(config.Conf.Root, ".tao", "client.json"))
	require.NoError(t, err)
	if runtime.GOOS != "windows" {
		require.Equal(t, os.FileMode(0600), info.Mode().Perm())
	}
}

func TestTaoLoginRedirectURIs(t *testing.T) {
	gin.SetMode(gin.TestMode)
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodGet, "http://yao.example/v1/user/oauth/tao/authorize", nil)
	t.Setenv("YAO_SERVER_URL", "https://yao.example")
	t.Setenv("TAO_OAUTH_LOGIN_REDIRECT_URIS", "https://web.example/auth/tao/callback, https://yao.example/v1/user/oauth/tao/callback")
	got := taoLoginRedirectURIs(c)
	require.Len(t, got, 2)
	require.Equal(t, "https://yao.example/v1/user/oauth/tao/callback", got[0])
	require.Equal(t, "https://web.example/auth/tao/callback", got[1])
}

func TestTaoHTTPActivationAndToken(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/oauth/activate/exchange", func(w http.ResponseWriter, r *http.Request) {
		var body map[string]string
		_ = json.NewDecoder(r.Body).Decode(&body)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"client_id":"cid","client_secret":"secret","redirect_uris":["https://yao.example/v1/user/oauth/tao/activate/callback","https://yao.example/v1/user/oauth/tao/callback"]}`))
	})
	mux.HandleFunc("/oauth/token", func(w http.ResponseWriter, r *http.Request) {
		if _, _, ok := r.BasicAuth(); !ok {
			http.Error(w, "missing basic auth", http.StatusUnauthorized)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"access_token":"at","token_type":"Bearer","expires_in":3600,"refresh_token":"rt"}`))
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()
	old := providers["tao"]
	providers["tao"] = &Provider{ID: "tao", Issuer: srv.URL}
	t.Cleanup(func() { providers["tao"] = old })
	act, err := taoActivationExchange(context.Background(), "code", "verifier", &taoDeployment{ID: "dep", PublicKey: "pub"})
	require.NoError(t, err)
	require.Equal(t, "cid", act.ClientID)
	require.Len(t, act.RedirectURIs, 2)
	t.Setenv("TAO_OAUTH_CLIENT_ID", "cid")
	t.Setenv("TAO_OAUTH_CLIENT_SECRET", "secret")
	token, err := taoExchangeCode(context.Background(), "code", "https://yao.example/cb", "verifier")
	require.NoError(t, err)
	require.Equal(t, "at", token.AccessToken)
	require.Equal(t, "rt", token.RefreshToken)
	refreshed, err := taoRefreshAccessToken(context.Background(), "rt")
	require.NoError(t, err)
	require.Equal(t, "at", refreshed.AccessToken)
	_, err = taoRefreshAccessToken(context.Background(), "")
	require.Error(t, err)
}

func TestLocalLoginDisabledAndPublicConfig(t *testing.T) {
	t.Setenv("YAO_USER_LOCAL_LOGIN_ENABLED", "")
	old := yaoClientConfig
	t.Cleanup(func() { yaoClientConfig = old })
	yaoClientConfig = &YaoClientConfig{User: &UserSystemConfig{LocalLogin: &LocalLoginConfig{Enabled: false}}}
	require.False(t, LocalLoginEnabled())
	gin.SetMode(gin.TestMode)
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	require.True(t, localLoginDisabled(c))
	require.Equal(t, http.StatusForbidden, rec.Code)
	require.Contains(t, rec.Body.String(), "local_login_disabled")

	cfg := &EntryConfig{Title: "Login"}
	public := createPublicEntryConfig(cfg)
	require.False(t, public.LocalLoginEnabled)
	yaoClientConfig.User.LocalLogin.Enabled = true
	public = createPublicEntryConfig(cfg)
	require.True(t, public.LocalLoginEnabled)
}

func TestTaoIdentityGateNotConfigured(t *testing.T) {
	old := providers["tao"]
	providers["tao"] = &Provider{ID: "tao"}
	t.Cleanup(func() { providers["tao"] = old })
	gin.SetMode(gin.TestMode)
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	RequireTaoIdentity(c)
	require.Equal(t, http.StatusServiceUnavailable, rec.Code)
	require.Contains(t, rec.Body.String(), "tao_not_configured")

	providers["tao"] = &Provider{ID: "tao", Issuer: "https://tao.example"}
	installFakeTaoProvider(t, &fakeTaoUserProvider{accounts: []maps.MapStrAny{{"provider": "tao", "sub": "sub", "refresh_token": "rt"}}}, nil)
	rec = httptest.NewRecorder()
	c, _ = gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodGet, "/", nil)
	c.Set("__user_id", "user_1")
	c.Set("__auth_source", "tao")
	RequireTaoIdentity(c)
	require.Equal(t, http.StatusOK, rec.Code)
}

func TestRegisterBuiltinTaoProvider(t *testing.T) {
	oldProviders := providers
	providers = make(map[string]*Provider)
	t.Cleanup(func() { providers = oldProviders })
	t.Setenv("TAO_OAUTH_ISSUER", "https://tao.example")
	t.Setenv("TAO_OAUTH_CLIENT_ID", "cid")
	t.Setenv("TAO_OAUTH_CLIENT_SECRET", "secret")
	require.NoError(t, registerBuiltinTaoProvider())
	p, err := GetProvider("tao")
	require.NoError(t, err)
	require.True(t, p.Available)
	require.Equal(t, "https://tao.example/oauth/authorize", p.Endpoints.Authorization)
	require.Equal(t, "cid", p.ClientID)
}

func TestPublicProvidersSanitize(t *testing.T) {
	oldProviders := providers
	providers = map[string]*Provider{
		"tao":   {ID: "tao", Issuer: "https://tao.example", ClientSecret: "secret", Available: true, Endpoints: &Endpoints{Authorization: "https://tao.example/oauth/authorize"}},
		"empty": {ID: "empty", ClientID: "", ClientSecret: "secret"},
	}
	t.Cleanup(func() { providers = oldProviders })
	list := PublicProviders()
	require.Len(t, list, 1)
	require.Equal(t, "tao", list[0].ID)
	require.Empty(t, list[0].ClientSecret)
}

func TestTaoCallbackFailure(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/cb", func(c *gin.Context) { taoCallbackFailure(c, "access_denied") })
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/cb", nil))
	require.Equal(t, http.StatusFound, rec.Code)
	require.Contains(t, rec.Header().Get("Location"), "access_denied")
}

func TestTaoBaseURLs(t *testing.T) {
	gin.SetMode(gin.TestMode)
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodGet, "https://yao.example/v1/user/oauth/tao/authorize", nil)
	t.Setenv("YAO_SERVER_URL", "")
	require.Equal(t, "https://yao.example", taoBaseURL(c))
	require.Contains(t, taoActivationCallbackURL(c), "/v1/user/oauth/tao/activate/callback")
	require.Contains(t, taoLoginCallbackURL(c), "/v1/user/oauth/tao/callback")
}

func TestTaoUserInfoFromAccessToken(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"sub":"tao-user","name":"Alice","email":"alice@example.com","email_verified":true}`))
	}))
	defer srv.Close()
	p := &Provider{ID: "tao", Endpoints: &Endpoints{UserInfo: srv.URL}}
	info, err := taoUserInfoFromToken(p, &OAuthTokenResponse{AccessToken: "at", TokenType: "Bearer"}, "")
	require.NoError(t, err)
	require.Equal(t, "tao-user", info.Sub)
	_, err = taoUserInfoFromToken(p, nil, "")
	require.Error(t, err)
}

func TestGetAvailableProvidersHandler(t *testing.T) {
	oldProviders := providers
	providers = map[string]*Provider{"tao": {ID: "tao", Issuer: "https://tao.example", Available: true}}
	t.Cleanup(func() { providers = oldProviders })
	gin.SetMode(gin.TestMode)
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	getAvailableProviders(c)
	require.Equal(t, http.StatusOK, rec.Code)
	require.Contains(t, rec.Body.String(), "tao")
}

func TestTaoProviderErrors(t *testing.T) {
	old := providers["tao"]
	providers["tao"] = nil
	t.Cleanup(func() { providers["tao"] = old })
	_, err := taoProvider()
	require.Error(t, err)
	providers["tao"] = &Provider{ID: "tao"}
	_, err = taoProvider()
	require.Error(t, err)
}

func TestTaoActivateAndLoginInvalidState(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/activate", taoActivateCallback)
	r.GET("/login", taoLoginCallback)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/activate?state=missing", nil))
	require.Equal(t, http.StatusFound, rec.Code)
	require.Contains(t, rec.Header().Get("Location"), "invalid_state")
	rec = httptest.NewRecorder()
	r.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/login?state=missing", nil))
	require.Equal(t, http.StatusFound, rec.Code)
	require.Contains(t, rec.Header().Get("Location"), "invalid_state")
}

func TestTaoLogoutWithoutOAuth(t *testing.T) {
	old := oauth.OAuth
	oauth.OAuth = nil
	t.Cleanup(func() { oauth.OAuth = old })
	// taoLogoutByUserID is intentionally best effort; nil OAuth must not panic.
	require.Error(t, taoLogoutByUserID(context.Background(), "user"))
}

func TestTaoUserInfoInvalidToken(t *testing.T) {
	p := &Provider{ID: "tao"}
	_, err := taoUserInfoFromToken(p, &OAuthTokenResponse{IDToken: "not-a-jwt"}, "")
	require.Error(t, err)
	_, err = taoUserInfoFromToken(p, &OAuthTokenResponse{}, "")
	require.Error(t, err)
}

func TestTaoHTTPErrorResponses(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, `{"error":"invalid_grant","error_description":"bad code"}`, http.StatusBadRequest)
	}))
	defer srv.Close()
	old := providers["tao"]
	providers["tao"] = &Provider{ID: "tao", Issuer: srv.URL}
	t.Cleanup(func() { providers["tao"] = old })
	t.Setenv("TAO_OAUTH_CLIENT_ID", "cid")
	t.Setenv("TAO_OAUTH_CLIENT_SECRET", "secret")
	_, err := taoExchangeCode(context.Background(), "code", "https://yao.example/cb", "verifier")
	require.Error(t, err)
	_, err = taoActivationExchange(context.Background(), "code", "verifier", &taoDeployment{ID: "d", PublicKey: "p"})
	require.Error(t, err)
}

func TestTaoManagedKeyValidation(t *testing.T) {
	ctx := context.Background()
	require.Error(t, taoEnsureManagedKey(ctx, "", "token"))
	require.Error(t, taoEnsureManagedKey(ctx, "user", ""))
	old := oauth.OAuth
	oauth.OAuth = nil
	t.Cleanup(func() { oauth.OAuth = old })
	require.Error(t, taoEnsureManagedKey(ctx, "user", "token"))
}

func TestTaoStateNilOAuth(t *testing.T) {
	old := oauth.OAuth
	oauth.OAuth = nil
	t.Cleanup(func() { oauth.OAuth = old })
	require.Error(t, saveTaoTxn(&taoLoginTxn{State: "s"}))
	_, err := loadTaoTxn("s")
	require.Error(t, err)
	deleteTaoTxn("s")
}

func TestTaoDeploymentCorruptAndDecode(t *testing.T) {
	oldRoot := config.Conf.Root
	config.Conf.Root = t.TempDir()
	t.Cleanup(func() { config.Conf.Root = oldRoot })
	t.Setenv("TAO_DEPLOYMENT_ID", "")
	t.Setenv("TAO_DEPLOYMENT_PUBLIC_KEY", "")
	d, err := loadTaoDeployment()
	require.NoError(t, err)
	// Save a valid private key and decode it.
	privPath := filepath.Join(config.Conf.Root, ".tao", "deployment.json")
	raw, err := os.ReadFile(privPath)
	require.NoError(t, err)
	var stored struct {
		ID         string `json:"deployment_id"`
		PublicKey  string `json:"deployment_public_key"`
		PrivateKey string `json:"deployment_private_key"`
	}
	require.NoError(t, json.Unmarshal(raw, &stored))
	_, err = decodeTaoDeploymentPrivateKey(stored.PrivateKey)
	require.NoError(t, err)
	_, err = decodeTaoDeploymentPrivateKey("bad")
	require.Error(t, err)
	require.NotEmpty(t, d.ID)
	require.NoError(t, os.WriteFile(privPath, []byte("{bad"), 0600))
	_, err = loadTaoDeployment()
	require.Error(t, err)
}

func TestTaoClientStoreInvalid(t *testing.T) {
	oldRoot := config.Conf.Root
	config.Conf.Root = t.TempDir()
	t.Cleanup(func() { config.Conf.Root = oldRoot })
	t.Setenv("TAO_OAUTH_CLIENT_ID", "")
	t.Setenv("TAO_OAUTH_CLIENT_SECRET", "")
	path := filepath.Join(config.Conf.Root, ".tao", "client.json")
	require.NoError(t, os.MkdirAll(filepath.Dir(path), 0700))
	require.NoError(t, os.WriteFile(path, []byte("{bad"), 0600))
	_, err := loadTaoClientCredentials()
	require.Error(t, err)
	require.Error(t, saveTaoClientCredentials("", "secret"))
	require.Error(t, saveTaoClientCredentials("id", ""))
}

func TestTaoStartAuthorizationErrorHandler(t *testing.T) {
	old := providers["tao"]
	providers["tao"] = nil
	t.Cleanup(func() { providers["tao"] = old })
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/authorize", getTaoAuthorizationURL)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/authorize", nil))
	require.Equal(t, http.StatusServiceUnavailable, rec.Code)
	require.Contains(t, rec.Body.String(), "tao_not_configured")
}

func testTaoCache(t *testing.T) store.Store {
	t.Helper()
	cache, err := lru.New(100)
	require.NoError(t, err)
	old := taoCache
	taoCache = func() store.Store { return cache }
	t.Cleanup(func() { taoCache = old })
	return cache
}

func TestTaoStateWithCache(t *testing.T) {
	testTaoCache(t)
	txn := &taoLoginTxn{State: "state-1", Nonce: "n", Verifier: "v", Mode: taoModeLogin}
	require.NoError(t, saveTaoTxn(txn))
	got, err := loadTaoTxn("state-1")
	require.NoError(t, err)
	require.Equal(t, "n", got.Nonce)
	require.NoError(t, taoCache().Set(taoTxnKey("bad"), "not-json", taoTxnTTL))
	_, err = loadTaoTxn("bad")
	require.Error(t, err)
	deleteTaoTxn("state-1")
	_, err = loadTaoTxn("state-1")
	require.Error(t, err)
	require.Error(t, saveTaoTxn(nil))
	require.Error(t, saveTaoTxn(&taoLoginTxn{}))
}

func TestTaoStartLoginBranches(t *testing.T) {
	testTaoCache(t)
	oldProviders := providers["tao"]
	providers["tao"] = &Provider{ID: "tao", Issuer: "https://tao.example", Scopes: []string{"openid"}}
	t.Cleanup(func() { providers["tao"] = oldProviders })
	oldRoot := config.Conf.Root
	config.Conf.Root = t.TempDir()
	t.Cleanup(func() { config.Conf.Root = oldRoot })
	t.Setenv("TAO_DEPLOYMENT_ID", "")
	t.Setenv("TAO_DEPLOYMENT_PUBLIC_KEY", "")
	t.Setenv("TAO_OAUTH_CLIENT_ID", "")
	t.Setenv("TAO_OAUTH_CLIENT_SECRET", "")
	t.Setenv("YAO_SERVER_URL", "https://yao.example")
	gin.SetMode(gin.TestMode)
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodGet, "https://yao.example/v1/user/oauth/tao/authorize", nil)
	authURL, state, err := taoStartLogin(c)
	require.NoError(t, err)
	require.Contains(t, authURL, "/oauth/activate?")
	require.NotEmpty(t, state)
	txn, err := loadTaoTxn(state)
	require.NoError(t, err)
	require.Equal(t, taoModeActivate, txn.Mode)

	t.Setenv("TAO_OAUTH_CLIENT_ID", "cid")
	t.Setenv("TAO_OAUTH_CLIENT_SECRET", "secret")
	rec = httptest.NewRecorder()
	c, _ = gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodGet, "https://yao.example/v1/user/oauth/tao/authorize", nil)
	authURL, state, err = taoStartLogin(c)
	require.NoError(t, err)
	require.Contains(t, authURL, "/oauth/authorize?")
	txn, err = loadTaoTxn(state)
	require.NoError(t, err)
	require.Equal(t, taoModeLogin, txn.Mode)
}

func TestTaoActivateCallbackHandler(t *testing.T) {
	cache := testTaoCache(t)
	_ = cache
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"client_id":"activated","client_secret":"secret","redirect_uris":["https://yao.example/v1/user/oauth/tao/activate/callback","https://yao.example/v1/user/oauth/tao/callback"]}`))
	}))
	defer srv.Close()
	oldProviders := providers["tao"]
	providers["tao"] = &Provider{ID: "tao", Issuer: srv.URL, Scopes: []string{"openid"}}
	t.Cleanup(func() { providers["tao"] = oldProviders })
	oldRoot := config.Conf.Root
	config.Conf.Root = t.TempDir()
	t.Cleanup(func() { config.Conf.Root = oldRoot })
	t.Setenv("TAO_DEPLOYMENT_ID", "dep")
	t.Setenv("TAO_DEPLOYMENT_PUBLIC_KEY", "pub")
	t.Setenv("TAO_OAUTH_CLIENT_ID", "")
	t.Setenv("TAO_OAUTH_CLIENT_SECRET", "")
	t.Setenv("YAO_SERVER_URL", "https://yao.example")
	require.NoError(t, saveTaoTxn(&taoLoginTxn{State: "state", SID: "sid", Mode: taoModeActivate, Verifier: "v"}))
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/cb", taoActivateCallback)
	req := httptest.NewRequest(http.MethodGet, "/cb?state=state&activation_code=code", nil)
	req.AddCookie(&http.Cookie{Name: "session_id", Value: "sid"})
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	require.Equal(t, http.StatusFound, rec.Code)
	require.Contains(t, rec.Header().Get("Location"), "/oauth/authorize?")

	// Missing activation code still consumes the state and fails.
	require.NoError(t, saveTaoTxn(&taoLoginTxn{State: "state2", SID: "sid", Mode: taoModeActivate, Verifier: "v"}))
	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/cb?state=state2", nil)
	req.AddCookie(&http.Cookie{Name: "session_id", Value: "sid"})
	r.ServeHTTP(rec, req)
	require.Equal(t, http.StatusFound, rec.Code)
	require.Contains(t, rec.Header().Get("Location"), "tao_activation_failed")
}

func TestTaoLoginCallbackHandlerErrors(t *testing.T) {
	testTaoCache(t)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, `{"error":"invalid_grant"}`, http.StatusBadRequest)
	}))
	defer srv.Close()
	oldProviders := providers["tao"]
	providers["tao"] = &Provider{ID: "tao", Issuer: srv.URL}
	t.Cleanup(func() { providers["tao"] = oldProviders })
	t.Setenv("TAO_OAUTH_CLIENT_ID", "cid")
	t.Setenv("TAO_OAUTH_CLIENT_SECRET", "secret")
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/cb", taoLoginCallback)
	require.NoError(t, saveTaoTxn(&taoLoginTxn{State: "s1", SID: "sid", Mode: taoModeLogin, RedirectURI: "https://yao.example/cb"}))
	req := httptest.NewRequest(http.MethodGet, "/cb?state=s1&code=code", nil)
	req.AddCookie(&http.Cookie{Name: "session_id", Value: "sid"})
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	require.Equal(t, http.StatusFound, rec.Code)
	require.Contains(t, rec.Header().Get("Location"), "server_error")

	require.NoError(t, saveTaoTxn(&taoLoginTxn{State: "s2", SID: "sid", Mode: taoModeLogin}))
	req = httptest.NewRequest(http.MethodGet, "/cb?state=s2", nil)
	req.AddCookie(&http.Cookie{Name: "session_id", Value: "sid"})
	rec = httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	require.Equal(t, http.StatusFound, rec.Code)
	require.Contains(t, rec.Header().Get("Location"), "access_denied")
}

func TestTaoStartLoginNoCache(t *testing.T) {
	old := taoCache
	taoCache = func() store.Store { return nil }
	t.Cleanup(func() { taoCache = old })
	oldProviders := providers["tao"]
	providers["tao"] = &Provider{ID: "tao", Issuer: "https://tao.example"}
	t.Cleanup(func() { providers["tao"] = oldProviders })
	t.Setenv("TAO_OAUTH_CLIENT_ID", "cid")
	t.Setenv("TAO_OAUTH_CLIENT_SECRET", "secret")
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest(http.MethodGet, "https://yao.example/", nil)
	_, _, err := taoStartLogin(c)
	require.Error(t, err)
}

var _ = url.QueryEscape

func TestTaoSaveHelpersWithoutOAuth(t *testing.T) {
	old := oauth.OAuth
	oauth.OAuth = nil
	t.Cleanup(func() { oauth.OAuth = old })
	_, err := taoUserID(context.Background(), "sub")
	require.Error(t, err)
	require.NoError(t, taoSaveRefreshToken(context.Background(), "sub", &OAuthTokenResponse{}))
	require.Error(t, taoSaveRefreshToken(context.Background(), "sub", &OAuthTokenResponse{RefreshToken: "rt"}))
}

func TestTaoDeploymentMissingEnvPublicKey(t *testing.T) {
	t.Setenv("TAO_DEPLOYMENT_ID", "id")
	t.Setenv("TAO_DEPLOYMENT_PUBLIC_KEY", "")
	_, err := loadTaoDeployment()
	require.Error(t, err)
}

func TestTaoSecretStoreReadDirectory(t *testing.T) {
	oldRoot := config.Conf.Root
	config.Conf.Root = t.TempDir()
	t.Cleanup(func() { config.Conf.Root = oldRoot })
	t.Setenv("TAO_OAUTH_CLIENT_ID", "")
	t.Setenv("TAO_OAUTH_CLIENT_SECRET", "")
	path := filepath.Join(config.Conf.Root, ".tao", "client.json")
	require.NoError(t, os.MkdirAll(path, 0700))
	_, err := loadTaoClientCredentials()
	require.Error(t, err)
}

func TestTaoStateInvalidValue(t *testing.T) {
	testTaoCache(t)
	require.NoError(t, taoCache().Set(taoTxnKey("bad-type"), 42, taoTxnTTL))
	_, err := loadTaoTxn("bad-type")
	require.Error(t, err)
	_, err = loadTaoTxn("")
	require.Error(t, err)
}

func TestTaoActivationFailureBranches(t *testing.T) {
	gin.SetMode(gin.TestMode)
	oldRoot := config.Conf.Root
	config.Conf.Root = t.TempDir()
	t.Cleanup(func() { config.Conf.Root = oldRoot })
	t.Setenv("TAO_DEPLOYMENT_ID", "dep")
	t.Setenv("TAO_DEPLOYMENT_PUBLIC_KEY", "pub")
	t.Setenv("YAO_SERVER_URL", "https://yao.example")
	testTaoCache(t)

	cases := []struct {
		name     string
		response string
		status   int
		code     string
		want     string
	}{
		{"missing credentials", `{}`, http.StatusOK, "code", "tao_activation_failed"},
		{"invalid redirects", `{"client_id":"cid","client_secret":"secret","redirect_uris":[]}`, http.StatusOK, "code", "tao_activation_failed"},
		{"exchange error", ``, http.StatusInternalServerError, "code", "tao_activation_failed"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if tc.status != http.StatusOK {
					http.Error(w, "boom", tc.status)
					return
				}
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(tc.response))
			}))
			defer srv.Close()
			oldProvider := providers["tao"]
			providers["tao"] = &Provider{ID: "tao", Issuer: srv.URL}
			defer func() { providers["tao"] = oldProvider }()
			state := "state-" + tc.name
			require.NoError(t, saveTaoTxn(&taoLoginTxn{State: state, SID: "sid", Mode: taoModeActivate, Verifier: "v"}))
			r := gin.New()
			r.GET("/cb", taoActivateCallback)
			req := httptest.NewRequest(http.MethodGet, "/cb?state="+url.QueryEscape(state)+"&activation_code="+tc.code, nil)
			req.AddCookie(&http.Cookie{Name: "session_id", Value: "sid"})
			rec := httptest.NewRecorder()
			r.ServeHTTP(rec, req)
			require.Equal(t, http.StatusFound, rec.Code)
			require.Contains(t, rec.Header().Get("Location"), tc.want)
		})
	}
}

func TestTaoLoginCallbackProviderMissing(t *testing.T) {
	testTaoCache(t)
	old := providers["tao"]
	providers["tao"] = &Provider{ID: "tao"}
	t.Cleanup(func() { providers["tao"] = old })
	require.NoError(t, saveTaoTxn(&taoLoginTxn{State: "s", SID: "sid", Mode: taoModeLogin}))
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/cb", taoLoginCallback)
	req := httptest.NewRequest(http.MethodGet, "/cb?state=s&code=code", nil)
	req.AddCookie(&http.Cookie{Name: "session_id", Value: "sid"})
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	require.Equal(t, http.StatusFound, rec.Code)
	require.Contains(t, rec.Header().Get("Location"), "tao_not_configured")
}

func TestTaoRefreshErrors(t *testing.T) {
	t.Setenv("TAO_OAUTH_CLIENT_ID", "")
	t.Setenv("TAO_OAUTH_CLIENT_SECRET", "")
	old := providers["tao"]
	providers["tao"] = &Provider{ID: "tao", Issuer: "https://tao.example"}
	t.Cleanup(func() { providers["tao"] = old })
	_, err := taoRefreshAccessToken(context.Background(), "")
	require.Error(t, err)
	_, err = taoRefreshAccessToken(context.Background(), "rt")
	require.Error(t, err) // no client credentials and no server
}

func TestTaoStartLoginErrors(t *testing.T) {
	testTaoCache(t)
	oldProviders := providers["tao"]
	providers["tao"] = &Provider{ID: "tao", Issuer: "https://tao.example"}
	t.Cleanup(func() { providers["tao"] = oldProviders })
	gin.SetMode(gin.TestMode)
	makeCtx := func() *gin.Context {
		c, _ := gin.CreateTestContext(httptest.NewRecorder())
		c.Request = httptest.NewRequest(http.MethodGet, "https://yao.example/v1/user/oauth/tao/authorize", nil)
		return c
	}
	t.Setenv("TAO_OAUTH_CLIENT_ID", "cid")
	t.Setenv("TAO_OAUTH_CLIENT_SECRET", "")
	_, _, err := taoStartLogin(makeCtx())
	require.Error(t, err)

	t.Setenv("TAO_OAUTH_CLIENT_ID", "")
	t.Setenv("TAO_OAUTH_CLIENT_SECRET", "")
	oldRoot := config.Conf.Root
	rootFile := filepath.Join(t.TempDir(), "not-a-dir")
	require.NoError(t, os.WriteFile(rootFile, []byte("x"), 0600))
	config.Conf.Root = rootFile
	t.Cleanup(func() { config.Conf.Root = oldRoot })
	_, _, err = taoStartLogin(makeCtx())
	require.Error(t, err)
}

func TestTaoExchangeErrors(t *testing.T) {
	oldProviders := providers["tao"]
	providers["tao"] = nil
	t.Cleanup(func() { providers["tao"] = oldProviders })
	_, err := taoExchangeCode(context.Background(), "code", "cb", "v")
	require.Error(t, err)
	_, err = taoActivationExchange(context.Background(), "code", "v", &taoDeployment{ID: "d", PublicKey: "p"})
	require.Error(t, err)
	_, err = taoRefreshAccessToken(context.Background(), "rt")
	require.Error(t, err)
}

func TestTaoCallbackSIDMismatch(t *testing.T) {
	testTaoCache(t)
	oldProviders := providers["tao"]
	providers["tao"] = &Provider{ID: "tao", Issuer: "https://tao.example"}
	t.Cleanup(func() { providers["tao"] = oldProviders })
	gin.SetMode(gin.TestMode)
	for _, path := range []string{"/activate", "/login"} {
		state := "sid-" + path
		mode := taoModeActivate
		if path == "/login" {
			mode = taoModeLogin
		}
		require.NoError(t, saveTaoTxn(&taoLoginTxn{State: state, SID: "expected", Mode: mode}))
		r := gin.New()
		if path == "/activate" {
			r.GET(path, taoActivateCallback)
		} else {
			r.GET(path, taoLoginCallback)
		}
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, path+"?state="+state+"&activation_code=c&code=c", nil)
		req.AddCookie(&http.Cookie{Name: "session_id", Value: "other"})
		r.ServeHTTP(rec, req)
		require.Equal(t, http.StatusFound, rec.Code)
		require.Contains(t, rec.Header().Get("Location"), "invalid_state")
	}
}

func TestTaoLoginCallbackUserInfoError(t *testing.T) {
	testTaoCache(t)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"access_token":"at","token_type":"Bearer","expires_in":3600,"id_token":"bad"}`))
	}))
	defer srv.Close()
	oldProviders := providers["tao"]
	providers["tao"] = &Provider{ID: "tao", Issuer: srv.URL}
	t.Cleanup(func() { providers["tao"] = oldProviders })
	t.Setenv("TAO_OAUTH_CLIENT_ID", "cid")
	t.Setenv("TAO_OAUTH_CLIENT_SECRET", "secret")
	require.NoError(t, saveTaoTxn(&taoLoginTxn{State: "u", SID: "sid", Mode: taoModeLogin, RedirectURI: "cb"}))
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/cb", taoLoginCallback)
	req := httptest.NewRequest(http.MethodGet, "/cb?state=u&code=c", nil)
	req.AddCookie(&http.Cookie{Name: "session_id", Value: "sid"})
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	require.Equal(t, http.StatusFound, rec.Code)
	require.Contains(t, rec.Header().Get("Location"), "server_error")
}

func TestTaoRefreshHTTPBranches(t *testing.T) {
	oldProviders := providers["tao"]
	t.Cleanup(func() { providers["tao"] = oldProviders })
	t.Setenv("TAO_OAUTH_CLIENT_ID", "cid")
	t.Setenv("TAO_OAUTH_CLIENT_SECRET", "secret")

	// Success and missing refresh token.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"access_token":"new","refresh_token":"rotated","expires_in":3600}`))
	}))
	token, err := func() (*OAuthTokenResponse, error) {
		providers["tao"] = &Provider{ID: "tao", Issuer: srv.URL}
		return taoRefreshAccessToken(context.Background(), "rt")
	}()
	require.NoError(t, err)
	require.Equal(t, "rotated", token.RefreshToken)
	srv.Close()

	// Invalid JSON and non-200 with error description.
	srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, `not-json`, http.StatusBadRequest)
	}))
	providers["tao"] = &Provider{ID: "tao", Issuer: srv.URL}
	_, err = taoRefreshAccessToken(context.Background(), "rt")
	require.Error(t, err)
	srv.Close()
	srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"error":"invalid_grant","error_description":"expired"}`))
	}))
	providers["tao"] = &Provider{ID: "tao", Issuer: srv.URL}
	_, err = taoRefreshAccessToken(context.Background(), "rt")
	require.Error(t, err)
	srv.Close()
}

func TestTaoRefreshUserTokenNilOAuth(t *testing.T) {
	old := oauth.OAuth
	oauth.OAuth = nil
	t.Cleanup(func() { oauth.OAuth = old })
	_, err := taoRefreshUserToken(context.Background(), "user")
	require.Error(t, err)
}

func TestTaoClaimsAudienceType(t *testing.T) {
	require.Error(t, taoValidateClaims(map[string]interface{}{"iss": "https://tao.example", "aud": 42}, "https://tao.example", "", "cid"))
}

func TestTaoActivationExchangeHTTPErrors(t *testing.T) {
	oldProviders := providers["tao"]
	providers["tao"] = &Provider{ID: "tao", Issuer: "http://127.0.0.1:1"}
	t.Cleanup(func() { providers["tao"] = oldProviders })
	_, err := taoActivationExchange(context.Background(), "c", "v", &taoDeployment{ID: "d", PublicKey: "p"})
	require.Error(t, err)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`not-json`))
	}))
	defer srv.Close()
	providers["tao"] = &Provider{ID: "tao", Issuer: srv.URL}
	_, err = taoActivationExchange(context.Background(), "c", "v", &taoDeployment{ID: "d", PublicKey: "p"})
	require.Error(t, err)
}

func TestTaoExchangeMissingClientAndBadResponse(t *testing.T) {
	oldProviders := providers["tao"]
	t.Cleanup(func() { providers["tao"] = oldProviders })
	providers["tao"] = &Provider{ID: "tao", Issuer: "https://tao.example"}
	t.Setenv("TAO_OAUTH_CLIENT_ID", "")
	t.Setenv("TAO_OAUTH_CLIENT_SECRET", "")
	_, err := taoExchangeCode(context.Background(), "c", "cb", "v")
	require.Error(t, err)
}

func TestTaoLoginCallbackWithoutSession(t *testing.T) {
	testTaoCache(t)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, `bad`, http.StatusBadRequest)
	}))
	defer srv.Close()
	oldProviders := providers["tao"]
	providers["tao"] = &Provider{ID: "tao", Issuer: srv.URL}
	t.Cleanup(func() { providers["tao"] = oldProviders })
	t.Setenv("TAO_OAUTH_CLIENT_ID", "cid")
	t.Setenv("TAO_OAUTH_CLIENT_SECRET", "secret")
	require.NoError(t, saveTaoTxn(&taoLoginTxn{State: "nosid", Mode: taoModeLogin, SID: ""}))
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/cb", taoLoginCallback)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/cb?state=nosid&code=c", nil))
	require.Equal(t, http.StatusFound, rec.Code)
	require.Contains(t, rec.Header().Get("Location"), "server_error")
}

func TestTaoUserInfoRequiresAccessToken(t *testing.T) {
	_, err := taoUserInfoFromToken(&Provider{ID: "tao"}, &OAuthTokenResponse{}, "")
	require.Error(t, err)
}

func TestBuiltinTaoConfigErrors(t *testing.T) {
	old := builtinTaoConfig
	t.Cleanup(func() { builtinTaoConfig = old })
	builtinTaoConfig = nil
	require.Error(t, registerBuiltinTaoProvider())
	builtinTaoConfig = []byte("{bad")
	require.Error(t, registerBuiltinTaoProvider())
}

func TestTaoDeploymentReadAndWriteErrors(t *testing.T) {
	oldRoot := config.Conf.Root
	t.Cleanup(func() { config.Conf.Root = oldRoot })
	t.Setenv("TAO_DEPLOYMENT_ID", "")
	t.Setenv("TAO_DEPLOYMENT_PUBLIC_KEY", "")
	rootFile := filepath.Join(t.TempDir(), "root-file")
	require.NoError(t, os.WriteFile(rootFile, []byte("x"), 0600))
	config.Conf.Root = rootFile
	_, err := loadTaoDeployment()
	require.Error(t, err)

	dir := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(dir, ".tao"), 0700))
	config.Conf.Root = dir
	// `.tao/deployment.json` is a directory, so WriteFile fails.
	require.NoError(t, os.MkdirAll(filepath.Join(dir, ".tao", "deployment.json"), 0700))
	_, err = loadTaoDeployment()
	require.Error(t, err)
}

func TestDecodeTaoDeploymentUnsupportedKey(t *testing.T) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)
	der, err := x509.MarshalPKCS8PrivateKey(key)
	require.NoError(t, err)
	pemText := string(pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: der}))
	_, err = decodeTaoDeploymentPrivateKey(pemText)
	require.Error(t, err)
}

func TestTaoExchangeCodeErrorBranches(t *testing.T) {
	oldProviders := providers["tao"]
	t.Cleanup(func() { providers["tao"] = oldProviders })
	// Missing client credentials.
	providers["tao"] = &Provider{ID: "tao", Issuer: "https://tao.example"}
	t.Setenv("TAO_OAUTH_CLIENT_ID", "")
	t.Setenv("TAO_OAUTH_CLIENT_SECRET", "")
	_, err := taoExchangeCode(context.Background(), "c", "cb", "v")
	require.Error(t, err)

	// Bad issuer URL -> request construction error.
	t.Setenv("TAO_OAUTH_CLIENT_ID", "cid")
	t.Setenv("TAO_OAUTH_CLIENT_SECRET", "secret")
	providers["tao"] = &Provider{ID: "tao", Issuer: "://bad"}
	_, err = taoExchangeCode(context.Background(), "c", "cb", "v")
	require.Error(t, err)

	// Network error.
	providers["tao"] = &Provider{ID: "tao", Issuer: "http://127.0.0.1:1"}
	_, err = taoExchangeCode(context.Background(), "c", "cb", "v")
	require.Error(t, err)

	// Malformed response.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`not-json`))
	}))
	defer srv.Close()
	providers["tao"] = &Provider{ID: "tao", Issuer: srv.URL}
	_, err = taoExchangeCode(context.Background(), "c", "cb", "v")
	require.Error(t, err)

	// Error description on non-200.
	srv2 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"error":"invalid_grant","error_description":"bad"}`))
	}))
	defer srv2.Close()
	providers["tao"] = &Provider{ID: "tao", Issuer: srv2.URL}
	_, err = taoExchangeCode(context.Background(), "c", "cb", "v")
	require.Error(t, err)
}

func TestTaoActivationExchangeErrorBranches(t *testing.T) {
	oldProviders := providers["tao"]
	t.Cleanup(func() { providers["tao"] = oldProviders })
	providers["tao"] = &Provider{ID: "tao", Issuer: "http://127.0.0.1:1"}
	_, err := taoActivationExchange(context.Background(), "c", "v", &taoDeployment{ID: "d", PublicKey: "p"})
	require.Error(t, err)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`not-json`))
	}))
	defer srv.Close()
	providers["tao"] = &Provider{ID: "tao", Issuer: srv.URL}
	_, err = taoActivationExchange(context.Background(), "c", "v", &taoDeployment{ID: "d", PublicKey: "p"})
	require.Error(t, err)

	srv2 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "no", http.StatusBadGateway)
	}))
	defer srv2.Close()
	providers["tao"] = &Provider{ID: "tao", Issuer: srv2.URL}
	_, err = taoActivationExchange(context.Background(), "c", "v", &taoDeployment{ID: "d", PublicKey: "p"})
	require.Error(t, err)
}

func TestTaoRefreshAccessTokenErrorBranches(t *testing.T) {
	oldProviders := providers["tao"]
	t.Cleanup(func() { providers["tao"] = oldProviders })
	t.Setenv("TAO_OAUTH_CLIENT_ID", "cid")
	t.Setenv("TAO_OAUTH_CLIENT_SECRET", "secret")
	providers["tao"] = &Provider{ID: "tao", Issuer: "http://127.0.0.1:1"}
	_, err := taoRefreshAccessToken(context.Background(), "rt")
	require.Error(t, err)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`not-json`))
	}))
	defer srv.Close()
	providers["tao"] = &Provider{ID: "tao", Issuer: srv.URL}
	_, err = taoRefreshAccessToken(context.Background(), "rt")
	require.Error(t, err)

	srv2 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"error_description":"expired"}`))
	}))
	defer srv2.Close()
	providers["tao"] = &Provider{ID: "tao", Issuer: srv2.URL}
	_, err = taoRefreshAccessToken(context.Background(), "rt")
	require.Error(t, err)

	srv3 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
	}))
	defer srv3.Close()
	providers["tao"] = &Provider{ID: "tao", Issuer: srv3.URL}
	_, err = taoRefreshAccessToken(context.Background(), "rt")
	require.Error(t, err)
}

func TestTaoClaimsAndUserInfoErrorBranches(t *testing.T) {
	require.Error(t, taoValidateClaims(map[string]interface{}{}, "", "", ""))
	_, err := taoUserInfoFromToken(&Provider{ID: "tao", Issuer: ""}, &OAuthTokenResponse{IDToken: "bad"}, "")
	require.Error(t, err)
}

func (f *fakeTaoUserProvider) GetUserOAuthAccounts(ctx context.Context, userID string) ([]maps.MapStrAny, error) {
	return f.accounts, f.accountsErr
}
func (f *fakeTaoUserProvider) GetUser(ctx context.Context, userID string) (maps.MapStrAny, error) {
	return f.user, f.userErr
}
func (f *fakeTaoUserProvider) GetUserForAuth(ctx context.Context, identifier, identifierType string) (maps.MapStrAny, error) {
	return f.authUser, f.authErr
}
func (f *fakeTaoUserProvider) UpdateOAuthAccount(ctx context.Context, provider, subject string, data maps.MapStrAny) error {
	if f.authUser == nil {
		f.authUser = maps.MapStrAny{}
	}
	f.authUser["updated_subject"] = subject
	return f.updateErr
}

func installFakeTaoProvider(t *testing.T, p oauthtypes.UserProvider, err error) {
	t.Helper()
	old := taoGetUserProvider
	taoGetUserProvider = func() (oauthtypes.UserProvider, error) { return p, err }
	t.Cleanup(func() { taoGetUserProvider = old })
}

func TestTaoLogoutBranches(t *testing.T) {
	oldProviders := providers["tao"]
	providers["tao"] = &Provider{ID: "tao", Issuer: "https://tao.example"}
	t.Cleanup(func() { providers["tao"] = oldProviders })
	t.Setenv("TAO_OAUTH_CLIENT_ID", "cid")
	t.Setenv("TAO_OAUTH_CLIENT_SECRET", "secret")

	installFakeTaoProvider(t, &fakeTaoUserProvider{}, nil)
	require.NoError(t, taoLogoutByUserID(context.Background(), "u"))

	installFakeTaoProvider(t, &fakeTaoUserProvider{accountsErr: errors.New("db")}, nil)
	require.Error(t, taoLogoutByUserID(context.Background(), "u"))

	installFakeTaoProvider(t, &fakeTaoUserProvider{accounts: []maps.MapStrAny{{"provider": "google"}}}, nil)
	require.NoError(t, taoLogoutByUserID(context.Background(), "u"))

	t.Setenv("TAO_OAUTH_CLIENT_ID", "")
	t.Setenv("TAO_OAUTH_CLIENT_SECRET", "")
	installFakeTaoProvider(t, &fakeTaoUserProvider{accounts: []maps.MapStrAny{{"provider": "tao", "refresh_token": "rt"}}}, nil)
	require.NoError(t, taoLogoutByUserID(context.Background(), "u"))

	t.Setenv("TAO_OAUTH_CLIENT_ID", "cid")
	t.Setenv("TAO_OAUTH_CLIENT_SECRET", "secret")
	installFakeTaoProvider(t, &fakeTaoUserProvider{accounts: []maps.MapStrAny{{"provider": "tao", "refresh_token": "rt"}}}, nil)
	providers["tao"] = &Provider{ID: "tao", Issuer: "http://127.0.0.1:1"}
	require.NoError(t, taoLogoutByUserID(context.Background(), "u"))
}

func TestTaoManagedKeyBranches(t *testing.T) {
	oldProviders := providers["tao"]
	t.Cleanup(func() { providers["tao"] = oldProviders })
	installFakeTaoProvider(t, &fakeTaoUserProvider{authUser: maps.MapStrAny{"tao_api_key": "sk-tao-existing"}}, nil)
	require.NoError(t, taoEnsureManagedKey(context.Background(), "u", "at"))

	installFakeTaoProvider(t, &fakeTaoUserProvider{authErr: errors.New("db")}, nil)
	providers["tao"] = nil
	require.Error(t, taoEnsureManagedKey(context.Background(), "u", "at"))

	cases := []struct {
		name string
		body string
		code int
	}{
		{"status", `{}`, http.StatusInternalServerError},
		{"json", `not-json`, http.StatusOK},
		{"empty", `{}`, http.StatusOK},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tc.code)
				_, _ = w.Write([]byte(tc.body))
			}))
			defer srv.Close()
			providers["tao"] = &Provider{ID: "tao", Issuer: srv.URL}
			installFakeTaoProvider(t, &fakeTaoUserProvider{}, nil)
			require.Error(t, taoEnsureManagedKey(context.Background(), "u", "at"))
		})
	}
}

func TestTaoGateProviderBranches(t *testing.T) {
	oldProviders := providers["tao"]
	providers["tao"] = &Provider{ID: "tao", Issuer: "https://tao.example"}
	t.Cleanup(func() { providers["tao"] = oldProviders })
	gin.SetMode(gin.TestMode)
	installFakeTaoProvider(t, &fakeTaoUserProvider{user: maps.MapStrAny{"identity_source": "tao"}}, nil)
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Set("__user_id", "u")
	c.Set("__auth_source", "password")
	RequireTaoIdentity(c)
	require.Equal(t, http.StatusOK, rec.Code)

	rec = httptest.NewRecorder()
	c, _ = gin.CreateTestContext(rec)
	c.Set("__user_id", "u")
	c.Set("__auth_source", "password")
	installFakeTaoProvider(t, &fakeTaoUserProvider{userErr: errors.New("db")}, nil)
	RequireTaoIdentity(c)
	require.Equal(t, http.StatusForbidden, rec.Code)
}

func TestTaoStateAndSecretEasyBranches(t *testing.T) {
	testTaoCache(t)
	require.NoError(t, taoCache().Set(taoTxnKey("mismatch"), `{"state":"other"}`, taoTxnTTL))
	_, err := loadTaoTxn("mismatch")
	require.Error(t, err)
	deleteTaoTxn("")

	oldRoot := config.Conf.Root
	config.Conf.Root = t.TempDir()
	t.Cleanup(func() { config.Conf.Root = oldRoot })
	rootFile := filepath.Join(config.Conf.Root, ".tao")
	require.NoError(t, os.WriteFile(rootFile, []byte("x"), 0600))
	require.Error(t, saveTaoClientCredentials("id", "secret"))

	pemText := string(pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: []byte("bad")}))
	_, err = decodeTaoDeploymentPrivateKey(pemText)
	require.Error(t, err)
}

func TestTaoActivationErrorMapping(t *testing.T) {
	cases := []struct {
		code string
		want string
	}{
		{"deployment_conflict", "tao_activation_conflict"},
		{"client_unavailable", "tao_activation_conflict"},
		{"rate_limited", "tao_activation_rate_limited"},
		{"client_quota_exceeded", "tao_activation_rate_limited"},
		{"invalid_activation", "tao_activation_failed"},
	}
	for _, tc := range cases {
		require.Equal(t, tc.want, taoActivationErrorCode(&taoAPIError{Code: tc.code}))
	}
	require.Equal(t, "tao_activation_failed", taoActivationErrorCode(errors.New("other")))
}

func TestTaoGateAuthRequiredBranches(t *testing.T) {
	old := providers["tao"]
	providers["tao"] = &Provider{ID: "tao", Issuer: "https://tao.example"}
	t.Cleanup(func() { providers["tao"] = old })
	gin.SetMode(gin.TestMode)
	ctx := func(authSource string) *gin.Context {
		rec := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(rec)
		c.Request = httptest.NewRequest(http.MethodGet, "/", nil)
		c.Set("__user_id", "user")
		c.Set("__auth_source", authSource)
		return c
	}

	installFakeTaoProvider(t, &fakeTaoUserProvider{accounts: []maps.MapStrAny{{"provider": "tao", "sub": "s", "refresh_token": "rt"}}}, nil)
	c := ctx("tao")
	RequireTaoIdentity(c)
	require.False(t, c.IsAborted())

	installFakeTaoProvider(t, &fakeTaoUserProvider{accounts: []maps.MapStrAny{{"provider": "google"}}}, nil)
	c = ctx("tao")
	RequireTaoIdentity(c)
	require.True(t, c.IsAborted())

	installFakeTaoProvider(t, nil, errors.New("provider down"))
	c = ctx("tao")
	RequireTaoIdentity(c)
	require.True(t, c.IsAborted())
}

func TestTaoManagedKeyGuardBranches(t *testing.T) {
	old := providers["tao"]
	providers["tao"] = &Provider{ID: "tao", Issuer: "https://tao.example"}
	t.Cleanup(func() { providers["tao"] = old })
	gin.SetMode(gin.TestMode)
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodGet, "/", nil)
	RequireTaoManagedKey(c)
	require.True(t, c.IsAborted())

	installFakeTaoProvider(t, &fakeTaoUserProvider{authUser: maps.MapStrAny{"tao_api_key": "sk-tao-key"}}, nil)
	rec = httptest.NewRecorder()
	c, _ = gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodGet, "/", nil)
	c.Set("__user_id", "user")
	RequireTaoManagedKey(c)
	require.False(t, c.IsAborted())

	installFakeTaoProvider(t, &fakeTaoUserProvider{authUser: maps.MapStrAny{}}, nil)
	rec = httptest.NewRecorder()
	c, _ = gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodGet, "/", nil)
	c.Set("__user_id", "user")
	RequireTaoManagedKey(c)
	require.True(t, c.IsAborted())
}

func TestTaoEnsureManagedKeyForUserBranches(t *testing.T) {
	require.Error(t, EnsureManagedKeyForUser(context.Background(), ""))
	installFakeTaoProvider(t, nil, errors.New("down"))
	require.Error(t, EnsureManagedKeyForUser(context.Background(), "user"))
	installFakeTaoProvider(t, &fakeTaoUserProvider{authUser: maps.MapStrAny{"tao_api_key": "sk-tao-key"}}, nil)
	require.NoError(t, EnsureManagedKeyForUser(context.Background(), "user"))
	installFakeTaoProvider(t, &fakeTaoUserProvider{authUser: maps.MapStrAny{}}, nil)
	require.Error(t, EnsureManagedKeyForUser(context.Background(), "user"))
}

func TestTaoClearRefreshTokenBranches(t *testing.T) {
	installFakeTaoProvider(t, nil, errors.New("down"))
	clearTaoRefreshTokenForUser(context.Background(), "user")
	installFakeTaoProvider(t, &fakeTaoUserProvider{accountsErr: errors.New("db")}, nil)
	clearTaoRefreshTokenForUser(context.Background(), "user")

	fake := &fakeTaoUserProvider{accounts: []maps.MapStrAny{
		{"provider": "google"},
		{"provider": "tao", "sub": ""},
		{"provider": "tao", "sub": "tao-sub", "refresh_token": "rt"},
	}}
	installFakeTaoProvider(t, fake, nil)
	clearTaoRefreshTokenForUser(context.Background(), "user")
	require.Equal(t, "tao-sub", fake.authUser["updated_subject"])
}

func TestBuiltinTaoEndpointFallback(t *testing.T) {
	oldCfg := builtinTaoConfig
	oldProviders := providers
	t.Cleanup(func() { builtinTaoConfig = oldCfg; providers = oldProviders })
	builtinTaoConfig = []byte(`{"id":"tao","issuer":"$ENV.TAO_OAUTH_ISSUER","client_id":"cid","mapping":"tao"}`)
	providers = make(map[string]*Provider)
	t.Setenv("TAO_OAUTH_ISSUER", "https://tao.example")
	t.Setenv("TAO_OAUTH_CLIENT_ID", "")
	t.Setenv("TAO_OAUTH_CLIENT_SECRET", "")
	require.NoError(t, registerBuiltinTaoProvider())
	p, err := GetProvider("tao")
	require.NoError(t, err)
	require.Equal(t, "https://tao.example/oauth/authorize", p.Endpoints.Authorization)
	require.Equal(t, "https://tao.example/oauth/token", p.Endpoints.Token)
	require.Equal(t, "https://tao.example/oauth/userinfo", p.Endpoints.UserInfo)
	require.Equal(t, "https://tao.example/.well-known/jwks.json", p.Endpoints.JWKS)
}

func TestTaoConfiguredAndLocalLoginHelpers(t *testing.T) {
	t.Setenv("YAO_USER_LOCAL_LOGIN_ENABLED", "")
	oldProviders := providers
	oldCfg := yaoClientConfig
	t.Cleanup(func() { providers = oldProviders; yaoClientConfig = oldCfg })
	providers = map[string]*Provider{"tao": nil}
	require.False(t, TaoConfigured())
	providers["tao"] = &Provider{ID: "tao", Issuer: "https://tao.example"}
	require.True(t, TaoConfigured())
	yaoClientConfig = nil
	require.False(t, LocalLoginEnabled())
	yaoClientConfig = &YaoClientConfig{}
	require.False(t, LocalLoginEnabled())
	t.Setenv("YAO_USER_LOCAL_LOGIN_ENABLED", "true")
	require.True(t, LocalLoginEnabled())
	t.Setenv("YAO_USER_LOCAL_LOGIN_ENABLED", "false")
	require.False(t, LocalLoginEnabled())
}

func TestTaoValidateAccessTokenAllowsClockSkew(t *testing.T) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)
	pub := key.PublicKey
	jwks := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]interface{}{"keys": []map[string]interface{}{{
			"kid": "k", "kty": "RSA", "use": "sig", "alg": "RS256",
			"n": base64.RawURLEncoding.EncodeToString(pub.N.Bytes()),
			"e": base64.RawURLEncoding.EncodeToString([]byte{byte(pub.E >> 16), byte(pub.E >> 8), byte(pub.E)}),
		}}})
	}))
	defer jwks.Close()
	provider := &Provider{Issuer: "https://tao.example", Endpoints: &Endpoints{JWKS: jwks.URL}}
	claims := jwt.MapClaims{
		"iss": "https://tao.example", "aud": "aud", "sub": "sub", "client_id": "client",
		"exp": time.Now().Add(time.Hour).Unix(), "iat": time.Now().Add(2 * time.Minute).Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	token.Header["kid"] = "k"
	signed, err := token.SignedString(key)
	require.NoError(t, err)
	require.NoError(t, taoValidateAccessToken(provider, signed, "client"))
}
