package user

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"github.com/yaoapp/kun/maps"
	oauthtypes "github.com/yaoapp/yao/openapi/oauth/types"
)

func withTaoBillingMode(t *testing.T, mode string) {
	t.Helper()
	configMutex.Lock()
	old := yaoClientConfig
	yaoClientConfig = &YaoClientConfig{Tao: &TaoClientConfig{BillingKey: &TaoBillingKeyConfig{Mode: mode}}}
	configMutex.Unlock()
	t.Cleanup(func() {
		configMutex.Lock()
		yaoClientConfig = old
		configMutex.Unlock()
	})
}

func TestResolveTaoBillingIdentity(t *testing.T) {
	info := &oauthtypes.AuthorizedInfo{UserID: "member", TeamID: "team-1"}
	fake := &fakeTaoUserProvider{
		teams: map[string]maps.MapStrAny{"team-1": {"owner_id": "owner"}},
	}
	installFakeTaoProvider(t, fake, nil)

	withTaoBillingMode(t, TaoBillingKeyModeOwner)
	identity, err := ResolveTaoBillingIdentity(context.Background(), info)
	require.NoError(t, err)
	require.Equal(t, "owner", identity.UserID)
	require.Equal(t, TaoBillingKeyModeOwner, identity.Mode)
	require.False(t, identity.IsOwner)

	ownerInfo := &oauthtypes.AuthorizedInfo{UserID: "owner", TeamID: "team-1"}
	identity, err = ResolveTaoBillingIdentity(context.Background(), ownerInfo)
	require.NoError(t, err)
	require.True(t, identity.IsOwner)

	withTaoBillingMode(t, TaoBillingKeyModeSelf)
	identity, err = ResolveTaoBillingIdentity(context.Background(), info)
	require.NoError(t, err)
	require.Equal(t, "member", identity.UserID)
	require.Equal(t, TaoBillingKeyModeSelf, identity.Mode)
}

func TestResolveTaoBillingIdentityWithoutTeamUsesSelf(t *testing.T) {
	withTaoBillingMode(t, TaoBillingKeyModeOwner)
	installFakeTaoProvider(t, &fakeTaoUserProvider{}, nil)

	identity, err := ResolveTaoBillingIdentity(context.Background(), &oauthtypes.AuthorizedInfo{UserID: "solo"})
	require.NoError(t, err)
	require.Equal(t, "solo", identity.UserID)
	require.Equal(t, TaoBillingKeyModeSelf, identity.Mode)
}

func TestResolveTaoBillingKeyOwnerMissing(t *testing.T) {
	withTaoBillingMode(t, TaoBillingKeyModeOwner)
	installFakeTaoProvider(t, &fakeTaoUserProvider{
		teams:    map[string]maps.MapStrAny{"team-1": {"owner_id": "owner"}},
		authUser: maps.MapStrAny{"tao_api_key": ""},
	}, nil)

	_, err := ResolveTaoBillingKey(context.Background(), &oauthtypes.AuthorizedInfo{UserID: "member", TeamID: "team-1"})
	var billingErr *TaoBillingError
	require.True(t, errors.As(err, &billingErr))
	require.Equal(t, "owner_key_missing", billingErr.Code)
	require.Equal(t, 403, billingErr.Status)
}

func TestResolveTaoBillingKeySelfMissing(t *testing.T) {
	withTaoBillingMode(t, TaoBillingKeyModeSelf)
	installFakeTaoProvider(t, &fakeTaoUserProvider{authUser: maps.MapStrAny{"tao_api_key": ""}}, nil)

	_, err := ResolveTaoBillingKey(context.Background(), &oauthtypes.AuthorizedInfo{UserID: "member", TeamID: "team-1"})
	var billingErr *TaoBillingError
	require.True(t, errors.As(err, &billingErr))
	require.Equal(t, "tao_key_missing", billingErr.Code)
}

func TestNormalizeTaoBillingKeyConfig(t *testing.T) {
	cfg := &YaoClientConfig{}
	require.NoError(t, normalizeTaoBillingKeyConfig(cfg))
	require.Equal(t, TaoBillingKeyModeOwner, cfg.Tao.BillingKey.Mode)

	cfg = &YaoClientConfig{Tao: &TaoClientConfig{BillingKey: &TaoBillingKeyConfig{Mode: "self"}}}
	require.NoError(t, normalizeTaoBillingKeyConfig(cfg))
	require.Equal(t, TaoBillingKeyModeSelf, cfg.Tao.BillingKey.Mode)

	cfg = &YaoClientConfig{Tao: &TaoClientConfig{BillingKey: &TaoBillingKeyConfig{Mode: "bogus"}}}
	require.Error(t, normalizeTaoBillingKeyConfig(cfg))
}

func TestGinUsageOwnerBalanceHiddenForMember(t *testing.T) {
	old := providers["tao"]
	providers["tao"] = &Provider{ID: "tao", Issuer: "https://tao.example"}
	t.Cleanup(func() { providers["tao"] = old })
	withTaoBillingMode(t, TaoBillingKeyModeOwner)
	installFakeTaoProvider(t, &fakeTaoUserProvider{
		teams: map[string]maps.MapStrAny{"team-1": {"owner_id": "owner"}},
	}, nil)

	gin.SetMode(gin.TestMode)
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodGet, "/v1/user/usage", nil)
	c.Set("__user_id", "member")
	c.Set("__team_id", "team-1")

	GinUsage(c)
	require.Equal(t, http.StatusForbidden, rec.Code)
	require.Contains(t, rec.Body.String(), `"owner_balance_hidden"`)
}

func TestRequireTaoManagedKeyOwnerMissing(t *testing.T) {
	withTaoBillingMode(t, TaoBillingKeyModeOwner)
	installFakeTaoProvider(t, &fakeTaoUserProvider{
		teams:    map[string]maps.MapStrAny{"team-1": {"owner_id": "owner"}},
		authUser: maps.MapStrAny{"tao_api_key": ""},
	}, nil)

	gin.SetMode(gin.TestMode)
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodGet, "/", nil)
	c.Set("__user_id", "member")
	c.Set("__team_id", "team-1")
	RequireTaoManagedKey(c)
	require.True(t, c.IsAborted())
	require.Equal(t, http.StatusForbidden, rec.Code)
	require.Contains(t, rec.Body.String(), `"owner_key_missing"`)
}

func TestGinUsageSelfMode(t *testing.T) {
	t.Setenv("TAO_OAUTH_CLIENT_ID", "cid")
	t.Setenv("TAO_OAUTH_CLIENT_SECRET", "secret")
	oldProviders := providers["tao"]
	t.Cleanup(func() { providers["tao"] = oldProviders })
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/oauth/token":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"access_token":"at","refresh_token":"rt2","token_type":"Bearer","expires_in":3600}`))
		case "/api/v1/users/me/balance":
			require.Equal(t, "Bearer at", r.Header.Get("Authorization"))
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"balance":1000}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()
	providers["tao"] = &Provider{ID: "tao", Issuer: srv.URL}
	fake := &fakeTaoUserProvider{
		user:     maps.MapStrAny{"type_id": "free"},
		accounts: []maps.MapStrAny{{"provider": "tao", "refresh_token": "rt", "sub": "sub-1"}},
	}
	installFakeTaoProvider(t, fake, nil)
	withTaoBillingMode(t, TaoBillingKeyModeSelf)

	gin.SetMode(gin.TestMode)
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodGet, "/v1/user/usage", nil)
	c.Set("__user_id", "self-user")
	c.Set("__team_id", "team-1")
	GinUsage(c)
	require.Equal(t, http.StatusOK, rec.Code)
	require.Contains(t, rec.Body.String(), `"remaining":1000`)
}

func TestResolveTaoBillingKeyOwnerCurrentMissing(t *testing.T) {
	withTaoBillingMode(t, TaoBillingKeyModeOwner)
	installFakeTaoProvider(t, &fakeTaoUserProvider{
		teams:    map[string]maps.MapStrAny{"team-1": {"owner_id": "owner"}},
		authUser: maps.MapStrAny{"tao_api_key": ""},
	}, nil)

	_, err := ResolveTaoBillingKey(context.Background(), &oauthtypes.AuthorizedInfo{UserID: "owner", TeamID: "team-1"})
	var billingErr *TaoBillingError
	require.True(t, errors.As(err, &billingErr))
	require.Equal(t, "owner_key_missing", billingErr.Code)
}
