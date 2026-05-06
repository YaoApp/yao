package llmprovider_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/yaoapp/gou/connector"
	"github.com/yaoapp/gou/process"
	"github.com/yaoapp/yao/setting"
)

func TestGetModel(t *testing.T) {
	r := setupRegistryWithSetting(t)
	p := createTestProviderForRole(t, r, "model-get")

	conn, err := r.GetModel(p.ConnectorID)
	require.NoError(t, err)
	assert.NotNil(t, conn)

	s := conn.Setting()
	host, _ := s["host"].(string)
	assert.Equal(t, "https://api.openai.com", host)
}

func TestGetModelNotFound(t *testing.T) {
	r := setupRegistryWithSetting(t)
	_, err := r.GetModel("nonexistent-connector")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestGetModelByProviderKey(t *testing.T) {
	r := setupRegistryWithSetting(t)
	p := createTestProviderForRole(t, r, "model-key")

	conn, err := r.GetModel(p.Key)
	require.NoError(t, err)
	assert.NotNil(t, conn)
}

func TestGetRoleModel(t *testing.T) {
	r := setupRegistryWithSetting(t)
	p := createTestProviderForRole(t, r, "rolemodel-prov")

	err := r.SetDefaults(map[string]string{"default": p.Key})
	require.NoError(t, err)

	conn, err := r.GetRoleModel("default")
	require.NoError(t, err)
	assert.NotNil(t, conn)

	s := conn.Setting()
	host, _ := s["host"].(string)
	assert.Equal(t, "https://api.openai.com", host)
}

func TestGetDefaultModel(t *testing.T) {
	r := setupRegistryWithSetting(t)
	p := createTestProviderForRole(t, r, "default-model")

	err := r.SetDefaults(map[string]string{"default": p.Key})
	require.NoError(t, err)

	conn, err := r.GetDefaultModel()
	require.NoError(t, err)
	assert.NotNil(t, conn)
}

func TestGetDefaultModelByUser(t *testing.T) {
	r := setupRegistryWithSetting(t)

	sysP := createTestProviderForRole(t, r, "dm-sys")
	userP := createTestProviderForRole(t, r, "dm-user")

	err := r.SetDefaults(map[string]string{"default": sysP.Key})
	require.NoError(t, err)

	_, err = setting.Global.Set(
		setting.ScopeID{Scope: setting.ScopeUser, UserID: "dm-u1"},
		"llm.roles",
		map[string]interface{}{
			"default": map[string]interface{}{
				"provider": userP.Key,
				"model":    "gpt-4o",
			},
		},
	)
	require.NoError(t, err)

	conn, err := r.GetDefaultModelByUser("dm-u1")
	require.NoError(t, err)
	assert.NotNil(t, conn)

	s := conn.Setting()
	model, _ := s["model"].(string)
	assert.Equal(t, "gpt-4o", model)
}

func TestGetCapabilities(t *testing.T) {
	r := setupRegistryWithSetting(t)
	p := createTestProviderForRole(t, r, "caps-prov")

	caps, err := r.GetCapabilities(p.ConnectorID)
	require.NoError(t, err)
	assert.NotNil(t, caps)
}

func TestGetRoleCapabilities(t *testing.T) {
	r := setupRegistryWithSetting(t)
	p := createTestProviderForRole(t, r, "rolecaps-prov")

	err := r.SetDefaults(map[string]string{"default": p.Key})
	require.NoError(t, err)

	caps, err := r.GetRoleCapabilities("default")
	require.NoError(t, err)
	assert.NotNil(t, caps)
}

func TestListModels(t *testing.T) {
	r := setupRegistryWithSetting(t)
	_ = createTestProviderForRole(t, r, "listm-prov")

	opts := r.ListModels()
	assert.NotEmpty(t, opts, "should have at least the created provider")

	found := false
	for _, o := range opts {
		if o.Label == "Test listm-prov / GPT-4o" {
			found = true
			break
		}
	}
	assert.True(t, found, "should contain the test provider's model")
}

func TestListModelsByUser(t *testing.T) {
	r := setupRegistryWithSetting(t)
	_ = createTestProviderForRole(t, r, "listmu-prov")

	opts := r.ListModelsByUser("some-user")
	assert.NotEmpty(t, opts)
}

func TestListModelsReturnsConnectorOption(t *testing.T) {
	r := setupRegistryWithSetting(t)
	p := createTestProviderForRole(t, r, "opt-prov")

	opts := r.ListModels()
	modelCID := p.ConnectorID + ":gpt-4o"
	found := false
	for _, o := range opts {
		if o.Value == modelCID {
			found = true
			assert.Equal(t, "Test opt-prov / GPT-4o", o.Label)
		}
	}
	assert.True(t, found, "should contain model-level option with colon-separated CID")
}

func TestListModelsIncludesBuiltin(t *testing.T) {
	r := setupRegistryWithSetting(t)

	opts := r.ListModels()
	builtinCount := 0
	for _, o := range opts {
		for _, ai := range connector.AIConnectors {
			if o.Value == ai.Value {
				builtinCount++
				break
			}
		}
	}
	t.Logf("ListModels returned %d options, %d matching builtin AIConnectors (total: %d)",
		len(opts), builtinCount, len(connector.AIConnectors))
}

func TestProcessGetModel(t *testing.T) {
	r := setupRegistryWithSetting(t)
	p := createTestProviderForRole(t, r, "proc-model")

	proc := process.New("llmprovider.getmodel", p.ConnectorID)
	result, err := proc.Exec()
	require.NoError(t, err)

	m, ok := result.(map[string]interface{})
	require.True(t, ok)
	host, _ := m["host"].(string)
	assert.Equal(t, "https://api.openai.com", host)
}

func TestProcessListModels(t *testing.T) {
	r := setupRegistryWithSetting(t)
	_ = createTestProviderForRole(t, r, "proc-listm")

	proc := process.New("llmprovider.listmodels")
	result, err := proc.Exec()
	require.NoError(t, err)

	list, ok := result.([]interface{})
	require.True(t, ok)
	assert.NotEmpty(t, list)

	item := list[0].(map[string]interface{})
	assert.Contains(t, item, "label")
	assert.Contains(t, item, "value")
}

func TestProcessGetCapabilities(t *testing.T) {
	r := setupRegistryWithSetting(t)
	p := createTestProviderForRole(t, r, "proc-caps")

	proc := process.New("llmprovider.getcapabilities", p.ConnectorID)
	result, err := proc.Exec()
	require.NoError(t, err)

	m, ok := result.(map[string]interface{})
	require.True(t, ok)
	assert.Contains(t, m, "streaming")
	assert.Contains(t, m, "tool_calls")
}

func TestProcessGetRoleModel(t *testing.T) {
	r := setupRegistryWithSetting(t)
	p := createTestProviderForRole(t, r, "proc-rm")

	err := r.SetDefaults(map[string]string{"default": p.Key})
	require.NoError(t, err)

	proc := process.New("llmprovider.getrolemodel", "default")
	result, err := proc.Exec()
	require.NoError(t, err)

	m, ok := result.(map[string]interface{})
	require.True(t, ok)
	host, _ := m["host"].(string)
	assert.Equal(t, "https://api.openai.com", host)
}
