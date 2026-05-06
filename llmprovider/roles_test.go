package llmprovider_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/yaoapp/gou/process"
	"github.com/yaoapp/gou/store"
	"github.com/yaoapp/yao/llmprovider"
	"github.com/yaoapp/yao/setting"
)

func setupRegistryWithSetting(t *testing.T) *llmprovider.Registry {
	t.Helper()
	r := setupRegistry(t)

	err := setting.Init()
	require.NoError(t, err)

	t.Cleanup(func() {
		s, _ := store.Get("__yao.store")
		if s != nil {
			s.Del("setting:*")
		}
		c, _ := store.Get("__yao.cache")
		if c != nil {
			c.Del("setting:*")
		}
	})
	return r
}

func createTestProviderForRole(t *testing.T, r *llmprovider.Registry, key string) *llmprovider.Provider {
	t.Helper()
	p := llmprovider.Provider{
		Key:     key,
		Name:    "Test " + key,
		Type:    "openai",
		APIURL:  "https://api.openai.com",
		APIKey:  "sk-test-role",
		Enabled: true,
		Models: []llmprovider.ModelInfo{
			{ID: "gpt-4o", Name: "GPT-4o", Capabilities: []string{"vision", "tool_calls", "streaming"}, Enabled: true},
		},
		Owner: llmprovider.ProviderOwner{Type: "system"},
	}
	created, err := r.Create(&p)
	require.NoError(t, err)
	return created
}

func TestSetDefaults(t *testing.T) {
	r := setupRegistryWithSetting(t)
	p := createTestProviderForRole(t, r, "sd-provider")

	err := r.SetDefaults(map[string]string{
		"default": p.Key,
	})
	require.NoError(t, err)

	merged, err := setting.Global.GetMerged("", "", "llm.roles")
	require.NoError(t, err)
	def, ok := merged["default"].(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, p.Key, def["provider"])
	assert.Equal(t, "gpt-4o", def["model"])
}

func TestGetRole(t *testing.T) {
	r := setupRegistryWithSetting(t)
	p := createTestProviderForRole(t, r, "role-provider")

	err := r.SetDefaults(map[string]string{"default": p.Key})
	require.NoError(t, err)

	cid, err := r.GetRole("default")
	require.NoError(t, err)
	assert.Equal(t, p.ConnectorID+":gpt-4o", cid)
}

func TestGetRoleByUser(t *testing.T) {
	r := setupRegistryWithSetting(t)

	sysP := createTestProviderForRole(t, r, "sys-prov")
	err := r.SetDefaults(map[string]string{"default": sysP.Key})
	require.NoError(t, err)

	userP := createTestProviderForRole(t, r, "user-prov")
	_, err = setting.Global.Set(
		setting.ScopeID{Scope: setting.ScopeUser, UserID: "u1"},
		"llm.roles",
		map[string]interface{}{
			"default": map[string]interface{}{
				"provider": userP.Key,
				"model":    "gpt-4o",
			},
		},
	)
	require.NoError(t, err)

	cid, err := r.GetRoleByUser("default", "u1")
	require.NoError(t, err)
	assert.Equal(t, userP.ConnectorID+":gpt-4o", cid, "user scope should override system")

	cidSys, err := r.GetRole("default")
	require.NoError(t, err)
	assert.Equal(t, sysP.ConnectorID+":gpt-4o", cidSys, "system scope should still return system provider")
}

func TestGetRoleByTeam(t *testing.T) {
	r := setupRegistryWithSetting(t)

	sysP := createTestProviderForRole(t, r, "sys-team-prov")
	err := r.SetDefaults(map[string]string{"default": sysP.Key})
	require.NoError(t, err)

	teamP := createTestProviderForRole(t, r, "team-prov")
	_, err = setting.Global.Set(
		setting.ScopeID{Scope: setting.ScopeTeam, TeamID: "t1"},
		"llm.roles",
		map[string]interface{}{
			"default": map[string]interface{}{
				"provider": teamP.Key,
				"model":    "gpt-4o",
			},
		},
	)
	require.NoError(t, err)

	cid, err := r.GetRoleByTeam("default", "t1")
	require.NoError(t, err)
	assert.Equal(t, teamP.ConnectorID+":gpt-4o", cid, "team scope should override system")
}

func TestGetRoleIncludesModel(t *testing.T) {
	r := setupRegistryWithSetting(t)
	p := createTestProviderForRole(t, r, "model-inc-prov")

	err := r.SetDefaults(map[string]string{"default": p.Key})
	require.NoError(t, err)

	cid, err := r.GetRole("default")
	require.NoError(t, err)
	assert.Contains(t, cid, ":", "connector ID should contain ':' separator for model")
	assert.Equal(t, p.ConnectorID+":gpt-4o", cid, "should include model suffix")
}

func TestGetRoleNoModel(t *testing.T) {
	r := setupRegistryWithSetting(t)

	noModelProvider := llmprovider.Provider{
		Key:     "nomodel-prov",
		Name:    "No Model Provider",
		Type:    "openai",
		APIURL:  "https://api.openai.com",
		APIKey:  "sk-test",
		Enabled: true,
		Models:  []llmprovider.ModelInfo{},
		Owner:   llmprovider.ProviderOwner{Type: "system"},
	}
	created, err := r.Create(&noModelProvider)
	require.NoError(t, err)

	err = r.SetDefaults(map[string]string{"default": created.Key})
	require.NoError(t, err)

	cid, err := r.GetRole("default")
	require.NoError(t, err)
	assert.Equal(t, created.ConnectorID, cid, "should return base connector ID without model when no models defined")
	assert.NotContains(t, cid, ":", "should not contain model separator")
}

func TestGetRoleNotConfigured(t *testing.T) {
	_ = setupRegistryWithSetting(t)

	_, err := llmprovider.Global.GetRole("nonexistent")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not configured")
}

func TestListRoles(t *testing.T) {
	r := setupRegistryWithSetting(t)
	p := createTestProviderForRole(t, r, "list-prov")

	err := r.SetDefaults(map[string]string{
		"default": p.Key,
		"vision":  p.Key,
	})
	require.NoError(t, err)

	roles, err := r.ListRoles()
	require.NoError(t, err)
	assert.Contains(t, roles, "default")
	assert.Contains(t, roles, "vision")
	assert.Equal(t, p.Key, roles["default"].Provider)
}

func TestListRolesByUser(t *testing.T) {
	r := setupRegistryWithSetting(t)

	sysP := createTestProviderForRole(t, r, "list-sys")
	userP := createTestProviderForRole(t, r, "list-user")

	err := r.SetDefaults(map[string]string{"default": sysP.Key, "vision": sysP.Key})
	require.NoError(t, err)

	_, err = setting.Global.Set(
		setting.ScopeID{Scope: setting.ScopeUser, UserID: "u2"},
		"llm.roles",
		map[string]interface{}{
			"default": map[string]interface{}{
				"provider": userP.Key,
				"model":    "gpt-4o",
			},
		},
	)
	require.NoError(t, err)

	roles, err := r.ListRolesByUser("u2")
	require.NoError(t, err)
	assert.Equal(t, userP.Key, roles["default"].Provider, "user override for default")
	assert.Equal(t, sysP.Key, roles["vision"].Provider, "system fallback for vision")
}

func TestProcessGetRole(t *testing.T) {
	r := setupRegistryWithSetting(t)
	p := createTestProviderForRole(t, r, "proc-role")

	err := r.SetDefaults(map[string]string{"default": p.Key})
	require.NoError(t, err)

	proc := process.New("llmprovider.getrole", "default")
	result, err := proc.Exec()
	require.NoError(t, err)
	assert.Equal(t, p.ConnectorID+":gpt-4o", result)
}

func TestProcessListRoles(t *testing.T) {
	r := setupRegistryWithSetting(t)
	p := createTestProviderForRole(t, r, "proc-list-role")

	err := r.SetDefaults(map[string]string{"default": p.Key})
	require.NoError(t, err)

	proc := process.New("llmprovider.listroles")
	result, err := proc.Exec()
	require.NoError(t, err)

	m, ok := result.(map[string]interface{})
	require.True(t, ok)
	def, ok := m["default"].(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, p.Key, def["provider"])
}
