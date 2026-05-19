//go:build integration

package llm_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/yaoapp/gou/connector"
	"github.com/yaoapp/gou/store"
	"github.com/yaoapp/yao/agent/llm"
	"github.com/yaoapp/yao/llmprovider"
	"github.com/yaoapp/yao/setting"
	"github.com/yaoapp/yao/unit-test/agent/testprepare"
)

type mockIdentity struct {
	UserID string
	TeamID string
}

func (m *mockIdentity) GetUserID() string { return m.UserID }
func (m *mockIdentity) GetTeamID() string { return m.TeamID }

func setupResolveTest(t *testing.T) string {
	t.Helper()
	testprepare.PrepareSandbox(t)

	err := setting.Init()
	require.NoError(t, err)

	err = llmprovider.Init()
	require.NoError(t, err)

	connIDs := connector.AIConnectors
	require.NotEmpty(t, connIDs, "no AI connectors available in test env")

	cid := connIDs[0].Value

	t.Cleanup(func() {
		s, _ := store.Get("__yao.store")
		if s != nil {
			s.Del("llmprovider:*")
		}
		c, _ := store.Get("__yao.cache")
		if c != nil {
			c.Del("llmprovider:*")
		}
	})

	return cid
}

// --- Resolve tests ---

func TestResolveConnector_UseLight(t *testing.T) {
	cid := setupResolveTest(t)

	err := llmprovider.Global.SetDefaults(map[string]string{"default": cid, "light": cid})
	require.NoError(t, err)

	conn, caps, err := llm.ResolveConnector("use::light", nil)
	require.NoError(t, err)
	assert.NotNil(t, conn)
	assert.NotNil(t, caps)
}

func TestResolveConnector_UseDefault(t *testing.T) {
	cid := setupResolveTest(t)

	err := llmprovider.Global.SetDefaults(map[string]string{"default": cid})
	require.NoError(t, err)

	conn, caps, err := llm.ResolveConnector("use::default", nil)
	require.NoError(t, err)
	assert.NotNil(t, conn)
	assert.NotNil(t, caps)
}

func TestResolveConnector_UseLightWithIdentity(t *testing.T) {
	cid := setupResolveTest(t)

	err := llmprovider.Global.SetDefaults(map[string]string{"default": cid, "light": cid})
	require.NoError(t, err)

	conn, caps, err := llm.ResolveConnector("use::light", &mockIdentity{UserID: "u1", TeamID: "t1"})
	require.NoError(t, err)
	assert.NotNil(t, conn)
	assert.NotNil(t, caps)
}

func TestResolveConnector_UseLightNoProvider(t *testing.T) {
	testprepare.PrepareSandbox(t)

	saved := llmprovider.Global
	llmprovider.Global = nil
	t.Cleanup(func() { llmprovider.Global = saved })

	_, _, err := llm.ResolveConnector("use::light", nil)
	assert.Error(t, err)
}

func TestResolveConnector_ExplicitID(t *testing.T) {
	cid := setupResolveTest(t)

	conn, caps, err := llm.ResolveConnector(cid, nil)
	require.NoError(t, err)
	assert.NotNil(t, conn)
	assert.NotNil(t, caps)
}

func TestResolveConnector_ExplicitIDPriority(t *testing.T) {
	cid := setupResolveTest(t)

	err := llmprovider.Global.SetDefaults(map[string]string{"default": cid})
	require.NoError(t, err)

	conn, caps, err := llm.ResolveConnector(cid, &mockIdentity{UserID: "u1"})
	require.NoError(t, err)
	assert.NotNil(t, conn)
	assert.NotNil(t, caps)
}

func TestResolveConnector_InvalidID(t *testing.T) {
	setupResolveTest(t)

	_, _, err := llm.ResolveConnector("nonexistent-connector-xyz", nil)
	assert.Error(t, err)
}

func TestResolveConnector_EmptyFallbackDefault(t *testing.T) {
	cid := setupResolveTest(t)

	err := llmprovider.Global.SetDefaults(map[string]string{"default": cid})
	require.NoError(t, err)

	conn, caps, err := llm.ResolveConnector("", nil)
	require.NoError(t, err)
	assert.NotNil(t, conn)
	assert.NotNil(t, caps)
}

// --- Capabilities tests ---

func TestCapabilitiesFromMap(t *testing.T) {
	caps := llm.ExportCapabilitiesFromMap(map[string]interface{}{
		"streaming":  true,
		"tool_calls": true,
		"vision":     true,
		"reasoning":  true,
		"json":       true,
	})
	require.NotNil(t, caps)
	assert.True(t, caps.Streaming)
	assert.True(t, caps.ToolCalls)
	assert.Equal(t, true, caps.Vision)
	assert.True(t, caps.Reasoning)
	assert.True(t, caps.JSON)

	defaults := llm.ExportCapabilitiesFromMap(map[string]interface{}{})
	require.NotNil(t, defaults)
	assert.False(t, defaults.Streaming)
	assert.False(t, defaults.ToolCalls)
	assert.False(t, defaults.Reasoning)
	assert.True(t, defaults.TemperatureAdjustable)
}

func TestGetCapabilities(t *testing.T) {
	cid := setupResolveTest(t)

	caps := llm.GetCapabilities(cid)
	assert.NotNil(t, caps)

	defaults := llm.GetCapabilities("")
	assert.NotNil(t, defaults)

	nonexistent := llm.GetCapabilities("nonexistent")
	assert.NotNil(t, nonexistent)
	assert.Equal(t, defaults, nonexistent)
}

func TestGetCapabilitiesFromConn(t *testing.T) {
	cid := setupResolveTest(t)

	conn, err := connector.Select(cid)
	require.NoError(t, err)

	caps := llm.GetCapabilitiesFromConn(conn)
	assert.NotNil(t, caps)

	nilCaps := llm.GetCapabilitiesFromConn(nil)
	assert.NotNil(t, nilCaps)
}

func TestGetCapabilitiesMap(t *testing.T) {
	cid := setupResolveTest(t)

	m := llm.GetCapabilitiesMap(cid)
	assert.NotNil(t, m)
	assert.Contains(t, m, "streaming")
	assert.Contains(t, m, "tool_calls")
	assert.Contains(t, m, "reasoning")
	assert.Contains(t, m, "json")

	caps := llm.GetCapabilities(cid)
	roundTrip := llm.ToMap(caps)
	assert.NotNil(t, roundTrip)
	assert.Equal(t, m, roundTrip)
}
