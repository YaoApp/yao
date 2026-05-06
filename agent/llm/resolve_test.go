package llm_test

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/yaoapp/gou/connector"
	"github.com/yaoapp/gou/store"
	"github.com/yaoapp/yao/agent/llm"
	"github.com/yaoapp/yao/config"
	"github.com/yaoapp/yao/llmprovider"
	"github.com/yaoapp/yao/setting"
	"github.com/yaoapp/yao/test"
)

func TestMain(m *testing.M) {
	test.Prepare(nil, config.Conf)
	defer test.Clean()
	os.Exit(m.Run())
}

type mockIdentity struct {
	UserID string
	TeamID string
}

func (m *mockIdentity) GetUserID() string { return m.UserID }
func (m *mockIdentity) GetTeamID() string { return m.TeamID }

func setupResolveTest(t *testing.T) string {
	t.Helper()
	test.Prepare(t, config.Conf)

	err := setting.Init()
	require.NoError(t, err)

	err = llmprovider.Init()
	require.NoError(t, err)

	connIDs := connector.AIConnectors
	if len(connIDs) == 0 {
		t.Skip("no AI connectors available in test env")
	}

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
		test.Clean()
	})

	return cid
}

// --- use:: prefix tests ---

func TestResolveConnector_UseLight(t *testing.T) {
	cid := setupResolveTest(t)

	err := llmprovider.Global.SetDefaults(map[string]string{
		"default": cid,
		"light":   cid,
	})
	require.NoError(t, err)

	conn, caps, err := llm.ResolveConnector("use::light", nil)
	require.NoError(t, err)
	assert.NotNil(t, conn)
	assert.NotNil(t, caps)
}

func TestResolveConnector_UseDefault(t *testing.T) {
	cid := setupResolveTest(t)

	err := llmprovider.Global.SetDefaults(map[string]string{
		"default": cid,
	})
	require.NoError(t, err)

	conn, caps, err := llm.ResolveConnector("use::default", nil)
	require.NoError(t, err)
	assert.NotNil(t, conn)
	assert.NotNil(t, caps)
}

func TestResolveConnector_UseLightWithIdentity(t *testing.T) {
	cid := setupResolveTest(t)

	err := llmprovider.Global.SetDefaults(map[string]string{
		"default": cid,
		"light":   cid,
	})
	require.NoError(t, err)

	conn, caps, err := llm.ResolveConnector("use::light", &mockIdentity{UserID: "u1", TeamID: "t1"})
	require.NoError(t, err)
	assert.NotNil(t, conn)
	assert.NotNil(t, caps)
}

func TestResolveConnector_UseLightNoProvider(t *testing.T) {
	test.Prepare(t, config.Conf)
	defer test.Clean()

	saved := llmprovider.Global
	llmprovider.Global = nil
	defer func() { llmprovider.Global = saved }()

	_, _, err := llm.ResolveConnector("use::light", nil)
	assert.Error(t, err)
}

// --- Explicit connector tests ---

func TestResolveConnector_ExplicitID(t *testing.T) {
	cid := setupResolveTest(t)

	conn, caps, err := llm.ResolveConnector(cid, nil)
	require.NoError(t, err)
	assert.NotNil(t, conn)
	assert.NotNil(t, caps)
}

func TestResolveConnector_ExplicitIDPriority(t *testing.T) {
	cid := setupResolveTest(t)

	err := llmprovider.Global.SetDefaults(map[string]string{
		"default": cid,
		"light":   cid,
	})
	require.NoError(t, err)

	// Explicit connector ID is NOT a use:: prefix, so it takes priority
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

// --- Empty connector fallback ---

func TestResolveConnector_EmptyFallbackDefault(t *testing.T) {
	cid := setupResolveTest(t)

	err := llmprovider.Global.SetDefaults(map[string]string{
		"default": cid,
	})
	require.NoError(t, err)

	// Empty string → treated as use::default
	conn, caps, err := llm.ResolveConnector("", nil)
	require.NoError(t, err)
	assert.NotNil(t, conn)
	assert.NotNil(t, caps)
}
