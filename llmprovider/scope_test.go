package llmprovider_test

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/yaoapp/gou/connector"
	"github.com/yaoapp/yao/llmprovider"
	oauthTypes "github.com/yaoapp/yao/openapi/oauth/types"
	"github.com/yaoapp/yao/setting"
)

// ---------------------------------------------------------------------------
// ScopedKey
// ---------------------------------------------------------------------------

func TestScopedKeyFormats(t *testing.T) {
	assert.Equal(t, "ualice.deepseek", llmprovider.ScopedKey(
		&llmprovider.ProviderOwner{Type: "user", UserID: "alice"}, "deepseek"))
	assert.Equal(t, "t9253.deepseek", llmprovider.ScopedKey(
		&llmprovider.ProviderOwner{Type: "team", TeamID: "9253"}, "deepseek"))
	assert.Equal(t, "deepseek", llmprovider.ScopedKey(
		&llmprovider.ProviderOwner{Type: "system"}, "deepseek"))
	assert.Equal(t, "deepseek", llmprovider.ScopedKey(
		&llmprovider.ProviderOwner{}, "deepseek"))
}

func TestDifferentOwnerSameBaseKey(t *testing.T) {
	r := setupRegistryWithSetting(t)

	ownerA := llmprovider.ProviderOwner{Type: "team", TeamID: "teamA"}
	ownerB := llmprovider.ProviderOwner{Type: "team", TeamID: "teamB"}

	pA := createOwnedProvider(t, r, "deepseek", ownerA)
	pB := createOwnedProvider(t, r, "deepseek", ownerB)

	assert.Equal(t, "tteamA.deepseek", pA.Key)
	assert.Equal(t, "tteamB.deepseek", pB.Key)

	gotA, err := r.Get(pA.Key)
	require.NoError(t, err)
	assert.Equal(t, pA.Key, gotA.Key)

	gotB, err := r.Get(pB.Key)
	require.NoError(t, err)
	assert.Equal(t, pB.Key, gotB.Key)
}

func TestSameOwnerDuplicateKey(t *testing.T) {
	r := setupRegistryWithSetting(t)

	owner := llmprovider.ProviderOwner{Type: "user", UserID: "u1"}
	_ = createOwnedProvider(t, r, "openai", owner)

	dup := llmprovider.Provider{
		Key:     llmprovider.ScopedKey(&owner, "openai"),
		Name:    "Dup",
		Type:    "openai",
		APIURL:  "https://api.openai.com",
		APIKey:  "sk-dup",
		Enabled: true,
		Models:  []llmprovider.ModelInfo{{ID: "gpt-4o", Name: "GPT-4o", Enabled: true}},
		Owner:   owner,
	}
	_, err := r.Create(&dup)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "already exists")
}

// ---------------------------------------------------------------------------
// Identity interface
// ---------------------------------------------------------------------------

func TestAuthorizedInfoSatisfiesIdentity(t *testing.T) {
	info := &oauthTypes.AuthorizedInfo{UserID: "u1", TeamID: "t1"}
	var id llmprovider.Identity = info
	assert.Equal(t, "u1", id.GetUserID())
	assert.Equal(t, "t1", id.GetTeamID())
}

func TestAuthorizedInfoNilSafe(t *testing.T) {
	var info *oauthTypes.AuthorizedInfo
	assert.Equal(t, "", info.GetUserID())
	assert.Equal(t, "", info.GetTeamID())
}

// ---------------------------------------------------------------------------
// ListModels owner filtering
// ---------------------------------------------------------------------------

func TestListModelsByUserFiltersOwner(t *testing.T) {
	r := setupRegistryWithSetting(t)

	createOwnedProvider(t, r, "user-alice-prov", llmprovider.ProviderOwner{Type: "user", UserID: "alice"})
	createOwnedProvider(t, r, "user-bob-prov", llmprovider.ProviderOwner{Type: "user", UserID: "bob"})
	createOwnedProvider(t, r, "team-x-prov", llmprovider.ProviderOwner{Type: "team", TeamID: "x"})

	opts := r.ListModelsByUser("alice")
	labels := optLabels(opts)
	assert.Contains(t, labels, "Test user-alice-prov / GPT-4o", "should include alice's model")
	assert.NotContains(t, labels, "Test user-bob-prov / GPT-4o", "should exclude bob's model")
	assert.NotContains(t, labels, "Test team-x-prov / GPT-4o", "should exclude team model")
}

func TestListModelsByTeamFiltersOwner(t *testing.T) {
	r := setupRegistryWithSetting(t)

	createOwnedProvider(t, r, "team-alpha-prov", llmprovider.ProviderOwner{Type: "team", TeamID: "alpha"})
	createOwnedProvider(t, r, "team-beta-prov", llmprovider.ProviderOwner{Type: "team", TeamID: "beta"})
	createOwnedProvider(t, r, "user-u1-prov", llmprovider.ProviderOwner{Type: "user", UserID: "u1"})

	opts := r.ListModelsByTeam("alpha")
	labels := optLabels(opts)
	assert.Contains(t, labels, "Test team-alpha-prov / GPT-4o", "should include alpha's model")
	assert.NotContains(t, labels, "Test team-beta-prov / GPT-4o", "should exclude beta's model")
	assert.NotContains(t, labels, "Test user-u1-prov / GPT-4o", "should exclude user model")
}

func TestListModelsByIncludesBuiltin(t *testing.T) {
	r := setupRegistryWithSetting(t)

	createOwnedProvider(t, r, "user-x-prov", llmprovider.ProviderOwner{Type: "user", UserID: "x"})

	all := r.ListModels()
	byUser := r.ListModelsByUser("x")

	builtinAll := countBuiltin(all)
	builtinScoped := countBuiltin(byUser)
	assert.Equal(t, builtinAll, builtinScoped, "ByUser should include all builtin providers")
}

func TestListModelsBy_TeamRouting(t *testing.T) {
	r := setupRegistryWithSetting(t)

	createOwnedProvider(t, r, "team-rt-prov", llmprovider.ProviderOwner{Type: "team", TeamID: "rt"})
	createOwnedProvider(t, r, "user-rt-prov", llmprovider.ProviderOwner{Type: "user", UserID: "rt"})

	info := &oauthTypes.AuthorizedInfo{UserID: "rt", TeamID: "rt"}
	opts := r.ListModelsBy(info)
	labels := optLabels(opts)

	assert.Contains(t, labels, "Test team-rt-prov / GPT-4o", "team takes priority when TeamID is set")
	assert.NotContains(t, labels, "Test user-rt-prov / GPT-4o", "user model should be excluded when TeamID is set")
}

func TestListModelsBy_UserFallback(t *testing.T) {
	r := setupRegistryWithSetting(t)

	createOwnedProvider(t, r, "user-fb-prov", llmprovider.ProviderOwner{Type: "user", UserID: "fb"})

	info := &oauthTypes.AuthorizedInfo{UserID: "fb"}
	opts := r.ListModelsBy(info)
	labels := optLabels(opts)

	assert.Contains(t, labels, "Test user-fb-prov / GPT-4o", "should include user model when no TeamID")
}

// ---------------------------------------------------------------------------
// ListModels per-model expansion
// ---------------------------------------------------------------------------

func TestListModelsExpandsMultipleModels(t *testing.T) {
	r := setupRegistryWithSetting(t)

	owner := llmprovider.ProviderOwner{Type: "user", UserID: "multi-u"}
	p := llmprovider.Provider{
		Key:    llmprovider.ScopedKey(&owner, "multi-model-prov"),
		Name:   "MultiModel",
		Type:   "openai",
		APIURL: "https://api.openai.com",
		APIKey: "sk-test",
		Models: []llmprovider.ModelInfo{
			{ID: "gpt-4o", Name: "GPT-4o", Enabled: true},
			{ID: "gpt-4o-mini", Name: "GPT-4o Mini", Enabled: true},
			{ID: "gpt-disabled", Name: "Disabled", Enabled: false},
		},
		Enabled: true,
		Owner:   owner,
	}
	_, err := r.Create(&p)
	require.NoError(t, err)

	opts := r.ListModelsByUser("multi-u")
	labels := optLabels(opts)
	values := optValues(opts)

	assert.Contains(t, labels, "MultiModel / GPT-4o")
	assert.Contains(t, labels, "MultiModel / GPT-4o Mini")
	assert.NotContains(t, labels, "MultiModel / Disabled", "disabled model should not appear")

	// Values should be "providerCID:modelID" format
	for _, v := range values {
		if strings.Contains(v, "multi-model-prov") {
			assert.Contains(t, v, ":", "dynamic model option should use colon-separated format")
		}
	}
}

func TestGetModelWithModelLevelCID(t *testing.T) {
	r := setupRegistryWithSetting(t)

	owner := llmprovider.ProviderOwner{Type: "team", TeamID: "mlcid-t1"}
	p := llmprovider.Provider{
		Key:    llmprovider.ScopedKey(&owner, "mlcid-prov"),
		Name:   "MLTest",
		Type:   "openai",
		APIURL: "https://api.openai.com",
		APIKey: "sk-test",
		Models: []llmprovider.ModelInfo{
			{ID: "gpt-4o", Name: "GPT-4o", Enabled: true},
			{ID: "gpt-4o-mini", Name: "GPT-4o Mini", Enabled: true},
		},
		Enabled: true,
		Owner:   owner,
	}
	created, err := r.Create(&p)
	require.NoError(t, err)

	modelCID := created.ConnectorID + ":gpt-4o"
	conn, err := r.GetModel(modelCID)
	require.NoError(t, err)
	assert.NotNil(t, conn)

	s := conn.Setting()
	model, _ := s["model"].(string)
	assert.Equal(t, "gpt-4o", model, "model-level connector should have the correct model")

	modelCID2 := created.ConnectorID + ":gpt-4o-mini"
	conn2, err := r.GetModel(modelCID2)
	require.NoError(t, err)

	s2 := conn2.Setting()
	model2, _ := s2["model"].(string)
	assert.Equal(t, "gpt-4o-mini", model2, "second model should have its own connector")
}

// ---------------------------------------------------------------------------
// GetModel ConnectorID reverse lookup
// ---------------------------------------------------------------------------

func TestGetModelByConnectorIDReverseLookup(t *testing.T) {
	r := setupRegistryWithSetting(t)

	p := createOwnedProvider(t, r, "rev-prov", llmprovider.ProviderOwner{Type: "user", UserID: "u99"})
	cid := p.ConnectorID
	assert.Equal(t, p.Key, cid, "dynamic provider ConnectorID should equal scoped Key")

	_ = connector.Unregister(cid)

	conn, err := r.GetModel(cid)
	require.NoError(t, err, "GetModel should find provider via ConnectorID reverse lookup")
	assert.NotNil(t, conn)

	s := conn.Setting()
	host, _ := s["host"].(string)
	assert.Equal(t, "https://api.openai.com", host)
}

func TestGetByConnectorID(t *testing.T) {
	r := setupRegistryWithSetting(t)

	p := createOwnedProvider(t, r, "bycid-prov", llmprovider.ProviderOwner{Type: "team", TeamID: "t55"})

	found, err := r.GetByConnectorID(p.ConnectorID)
	require.NoError(t, err)
	assert.Equal(t, p.Key, found.Key)
}

func TestGetByConnectorIDNotFound(t *testing.T) {
	r := setupRegistryWithSetting(t)

	_, err := r.GetByConnectorID("nonexistent-cid")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

// ---------------------------------------------------------------------------
// GetRoleBy / ListRolesBy / GetRoleModelBy — Identity routing
// ---------------------------------------------------------------------------

func TestGetRoleBy_TeamPriority(t *testing.T) {
	r := setupRegistryWithSetting(t)

	sysP := createTestProviderForRole(t, r, "grb-sys")
	teamP := createTestProviderForRole(t, r, "grb-team")

	err := r.SetDefaults(map[string]string{"default": sysP.Key})
	require.NoError(t, err)

	_, err = setting.Global.Set(
		setting.ScopeID{Scope: setting.ScopeTeam, TeamID: "grb-t1"},
		"llm.roles",
		map[string]interface{}{
			"default": map[string]interface{}{
				"provider": teamP.Key,
				"model":    "gpt-4o",
			},
		},
	)
	require.NoError(t, err)

	info := &oauthTypes.AuthorizedInfo{UserID: "u1", TeamID: "grb-t1"}
	cid, err := r.GetRoleBy("default", info)
	require.NoError(t, err)
	assert.Equal(t, teamP.ConnectorID+":gpt-4o", cid, "should resolve via team scope when TeamID is set")
}

func TestGetRoleBy_UserFallback(t *testing.T) {
	r := setupRegistryWithSetting(t)

	sysP := createTestProviderForRole(t, r, "grbu-sys")
	userP := createTestProviderForRole(t, r, "grbu-user")

	err := r.SetDefaults(map[string]string{"default": sysP.Key})
	require.NoError(t, err)

	_, err = setting.Global.Set(
		setting.ScopeID{Scope: setting.ScopeUser, UserID: "grbu-u1"},
		"llm.roles",
		map[string]interface{}{
			"default": map[string]interface{}{
				"provider": userP.Key,
				"model":    "gpt-4o",
			},
		},
	)
	require.NoError(t, err)

	info := &oauthTypes.AuthorizedInfo{UserID: "grbu-u1"}
	cid, err := r.GetRoleBy("default", info)
	require.NoError(t, err)
	assert.Equal(t, userP.ConnectorID+":gpt-4o", cid, "should resolve via user scope when no TeamID")
}

func TestListRolesBy(t *testing.T) {
	r := setupRegistryWithSetting(t)

	sysP := createTestProviderForRole(t, r, "lrb-sys")
	teamP := createTestProviderForRole(t, r, "lrb-team")

	err := r.SetDefaults(map[string]string{"default": sysP.Key, "vision": sysP.Key})
	require.NoError(t, err)

	_, err = setting.Global.Set(
		setting.ScopeID{Scope: setting.ScopeTeam, TeamID: "lrb-t1"},
		"llm.roles",
		map[string]interface{}{
			"default": map[string]interface{}{
				"provider": teamP.Key,
				"model":    "gpt-4o",
			},
		},
	)
	require.NoError(t, err)

	info := &oauthTypes.AuthorizedInfo{TeamID: "lrb-t1"}
	roles, err := r.ListRolesBy(info)
	require.NoError(t, err)

	assert.Equal(t, teamP.Key, roles["default"].Provider, "team override for default")
	assert.Equal(t, sysP.Key, roles["vision"].Provider, "system fallback for vision")
}

func TestGetRoleModelBy(t *testing.T) {
	r := setupRegistryWithSetting(t)

	sysP := createTestProviderForRole(t, r, "grmb-sys")
	userP := createTestProviderForRole(t, r, "grmb-user")

	err := r.SetDefaults(map[string]string{"default": sysP.Key})
	require.NoError(t, err)

	_, err = setting.Global.Set(
		setting.ScopeID{Scope: setting.ScopeUser, UserID: "grmb-u1"},
		"llm.roles",
		map[string]interface{}{
			"default": map[string]interface{}{
				"provider": userP.Key,
				"model":    "gpt-4o",
			},
		},
	)
	require.NoError(t, err)

	info := &oauthTypes.AuthorizedInfo{UserID: "grmb-u1"}
	conn, err := r.GetRoleModelBy("default", info)
	require.NoError(t, err)
	assert.NotNil(t, conn)

	s := conn.Setting()
	model, _ := s["model"].(string)
	assert.Equal(t, "gpt-4o", model)
}

func TestGetDefaultModelBy(t *testing.T) {
	r := setupRegistryWithSetting(t)

	p := createTestProviderForRole(t, r, "gdmb-prov")
	err := r.SetDefaults(map[string]string{"default": p.Key})
	require.NoError(t, err)

	info := &oauthTypes.AuthorizedInfo{UserID: "gdmb-u1"}
	conn, err := r.GetDefaultModelBy(info)
	require.NoError(t, err)
	assert.NotNil(t, conn)
}

func TestGetRoleCapabilitiesBy(t *testing.T) {
	r := setupRegistryWithSetting(t)

	p := createTestProviderForRole(t, r, "grcb-prov")
	err := r.SetDefaults(map[string]string{"default": p.Key})
	require.NoError(t, err)

	info := &oauthTypes.AuthorizedInfo{UserID: "grcb-u1"}
	caps, err := r.GetRoleCapabilitiesBy("default", info)
	require.NoError(t, err)
	assert.NotNil(t, caps)
}

// ---------------------------------------------------------------------------
// helpers
// ---------------------------------------------------------------------------

func createOwnedProvider(t *testing.T, r *llmprovider.Registry, baseKey string, owner llmprovider.ProviderOwner) *llmprovider.Provider {
	t.Helper()
	p := llmprovider.Provider{
		Key:     llmprovider.ScopedKey(&owner, baseKey),
		Name:    "Test " + baseKey,
		Type:    "openai",
		APIURL:  "https://api.openai.com",
		APIKey:  "sk-test-owned",
		Enabled: true,
		Models: []llmprovider.ModelInfo{
			{ID: "gpt-4o", Name: "GPT-4o", Capabilities: []string{"vision", "tool_calls", "streaming"}, Enabled: true},
		},
		Owner: owner,
	}
	created, err := r.Create(&p)
	require.NoError(t, err)
	return created
}

func optLabels(opts []connector.Option) []string {
	labels := make([]string, len(opts))
	for i, o := range opts {
		labels[i] = o.Label
	}
	return labels
}

func optValues(opts []connector.Option) []string {
	values := make([]string, len(opts))
	for i, o := range opts {
		values[i] = o.Value
	}
	return values
}

func countBuiltin(opts []connector.Option) int {
	n := 0
	for _, o := range opts {
		for _, ai := range connector.AIConnectors {
			if o.Value == ai.Value {
				n++
				break
			}
		}
	}
	return n
}
