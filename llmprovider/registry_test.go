package llmprovider_test

import (
	"fmt"
	"os"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/yaoapp/gou/connector"
	"github.com/yaoapp/gou/store"
	"github.com/yaoapp/yao/config"
	"github.com/yaoapp/yao/llmprovider"
	"github.com/yaoapp/yao/test"
)

func TestMain(m *testing.M) {
	test.Prepare(nil, config.Conf)
	defer test.Clean()
	os.Exit(m.Run())
}

func setupRegistry(t *testing.T) *llmprovider.Registry {
	t.Helper()
	test.Prepare(t, config.Conf)

	err := llmprovider.Init()
	require.NoError(t, err)

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

	return llmprovider.Global
}

var testProvider = llmprovider.Provider{
	Key:        "test-openai",
	Name:       "Test OpenAI",
	Type:       "openai",
	APIURL:     "https://api.openai.com",
	APIKey:     "sk-test-xxxxx",
	Models:     []llmprovider.ModelInfo{{ID: "gpt-4o", Name: "GPT-4o", Capabilities: []string{"vision", "tool_calls", "streaming"}, Enabled: true}},
	Enabled:    true,
	RequireKey: true,
	Owner:      llmprovider.ProviderOwner{Type: "system"},
}

func TestCreate(t *testing.T) {
	r := setupRegistry(t)

	p := testProvider
	created, err := r.Create(&p)
	require.NoError(t, err)
	assert.Equal(t, "test-openai", created.Key)
	assert.Equal(t, llmprovider.ProviderSourceDynamic, created.Source)
	assert.NotEmpty(t, created.ConnectorID)

	// Verify store persistence
	s, _ := store.Get("__yao.store")
	assert.True(t, s.Has("llmprovider:p:test-openai"))

	// Verify connector registered
	_, err = connector.Select(created.ConnectorID)
	assert.NoError(t, err)
}

func TestCreateDuplicate(t *testing.T) {
	r := setupRegistry(t)

	p := testProvider
	_, err := r.Create(&p)
	require.NoError(t, err)

	dup := testProvider
	_, err = r.Create(&dup)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "already exists")
}

func TestGet(t *testing.T) {
	r := setupRegistry(t)

	p := testProvider
	_, err := r.Create(&p)
	require.NoError(t, err)

	got, err := r.Get("test-openai")
	require.NoError(t, err)
	assert.Equal(t, "Test OpenAI", got.Name)
	assert.Equal(t, "openai", got.Type)
	assert.Equal(t, "https://api.openai.com", got.APIURL)
	assert.Len(t, got.Models, 1)
	assert.Equal(t, "gpt-4o", got.Models[0].ID)
}

func TestGetNotFound(t *testing.T) {
	r := setupRegistry(t)

	_, err := r.Get("nonexistent")
	assert.Error(t, err)
}

func TestGetMasked(t *testing.T) {
	r := setupRegistry(t)

	p := testProvider
	_, err := r.Create(&p)
	require.NoError(t, err)

	got, err := r.GetMasked("test-openai")
	require.NoError(t, err)
	assert.NotEqual(t, "sk-test-xxxxx", got.APIKey)
	assert.True(t, len(got.APIKey) > 0)
	// Last 4 chars should be visible
	assert.Contains(t, got.APIKey, "xxxx")
}

// ---------------------------------------------------------------------------
// withKey behavior
// ---------------------------------------------------------------------------

func TestGet_DefaultMasked(t *testing.T) {
	r := setupRegistry(t)
	p := testProvider
	_, err := r.Create(&p)
	require.NoError(t, err)

	got, err := r.Get("test-openai")
	require.NoError(t, err)
	assert.NotEqual(t, "sk-test-xxxxx", got.APIKey, "Get() default should mask APIKey")
	assert.Contains(t, got.APIKey, "*")
}

func TestGet_WithKeyTrue(t *testing.T) {
	r := setupRegistry(t)
	p := testProvider
	_, err := r.Create(&p)
	require.NoError(t, err)

	got, err := r.Get("test-openai", true)
	require.NoError(t, err)
	assert.Equal(t, "sk-test-xxxxx", got.APIKey, "Get(key, true) should return plain text APIKey")
}

func TestGetByConnectorID_DefaultMasked(t *testing.T) {
	r := setupRegistry(t)
	p := testProvider
	created, err := r.Create(&p)
	require.NoError(t, err)

	got, err := r.GetByConnectorID(created.ConnectorID)
	require.NoError(t, err)
	assert.NotEqual(t, "sk-test-xxxxx", got.APIKey, "GetByConnectorID() default should mask")
	assert.Contains(t, got.APIKey, "*")
}

func TestGetByConnectorID_WithKeyTrue(t *testing.T) {
	r := setupRegistry(t)
	p := testProvider
	created, err := r.Create(&p)
	require.NoError(t, err)

	got, err := r.GetByConnectorID(created.ConnectorID, true)
	require.NoError(t, err)
	assert.Equal(t, "sk-test-xxxxx", got.APIKey, "GetByConnectorID(cid, true) should return plain text")
}

func TestList_DefaultMasked(t *testing.T) {
	r := setupRegistry(t)
	p := testProvider
	_, err := r.Create(&p)
	require.NoError(t, err)

	list, err := r.List(&llmprovider.ProviderFilter{Source: llmprovider.ProviderSourceDynamic})
	require.NoError(t, err)
	require.True(t, len(list) > 0)

	for _, item := range list {
		assert.NotEqual(t, "sk-test-xxxxx", item.APIKey, "List() default should mask all APIKeys")
	}
}

func TestList_WithKeyTrue(t *testing.T) {
	r := setupRegistry(t)
	p := testProvider
	_, err := r.Create(&p)
	require.NoError(t, err)

	list, err := r.List(&llmprovider.ProviderFilter{Source: llmprovider.ProviderSourceDynamic}, true)
	require.NoError(t, err)

	found := false
	for _, item := range list {
		if item.Key == "test-openai" {
			found = true
			assert.Equal(t, "sk-test-xxxxx", item.APIKey, "List(filter, true) should return plain text")
		}
	}
	assert.True(t, found)
}

func TestGetMasked_EqualsGetDefault(t *testing.T) {
	r := setupRegistry(t)
	p := testProvider
	_, err := r.Create(&p)
	require.NoError(t, err)

	fromGet, err := r.Get("test-openai")
	require.NoError(t, err)

	fromGetMasked, err := r.GetMasked("test-openai")
	require.NoError(t, err)

	assert.Equal(t, fromGet.APIKey, fromGetMasked.APIKey, "GetMasked should equal Get (both masked by default)")
}

func TestListModels_ConnectorHasRealKey(t *testing.T) {
	r := setupRegistry(t)

	owner := llmprovider.ProviderOwner{Type: "user", UserID: "rk-user"}
	p := llmprovider.Provider{
		Key:    llmprovider.ScopedKey(&owner, "realkey-prov"),
		Name:   "RealKey Test",
		Type:   "openai",
		APIURL: "https://api.openai.com",
		APIKey: "sk-real-secret-key-12345",
		Models: []llmprovider.ModelInfo{
			{ID: "gpt-4o", Name: "GPT-4o", Capabilities: []string{"streaming"}, Enabled: true},
		},
		Enabled: true,
		Owner:   owner,
	}
	_, err := r.Create(&p)
	require.NoError(t, err)

	opts := r.ListModelsByUser("rk-user")
	require.True(t, len(opts) > 0, "should have at least one model option")

	var modelCID string
	for _, o := range opts {
		if o.Label == "RealKey Test / GPT-4o" {
			modelCID = o.Value
			break
		}
	}
	require.NotEmpty(t, modelCID, "should find the model option")

	conn, err := connector.Select(modelCID)
	require.NoError(t, err, "model connector should be registered")

	s := conn.Setting()
	key, _ := s["key"].(string)
	assert.Equal(t, "sk-real-secret-key-12345", key, "connector should have the real API key, not masked")
}

func TestGetLazy(t *testing.T) {
	r := setupRegistry(t)

	p := testProvider
	created, err := r.Create(&p)
	require.NoError(t, err)

	// Manually unregister the connector
	err = connector.Unregister(created.ConnectorID)
	require.NoError(t, err)

	// Verify it's gone
	_, err = connector.Select(created.ConnectorID)
	assert.Error(t, err)

	// Get should lazily re-register
	got, err := r.Get("test-openai")
	require.NoError(t, err)
	assert.Equal(t, "test-openai", got.Key)

	// Connector should be back
	_, err = connector.Select(got.ConnectorID)
	assert.NoError(t, err)
}

func TestList(t *testing.T) {
	r := setupRegistry(t)

	p2Owner := llmprovider.ProviderOwner{Type: "user", UserID: "123"}
	providers := []llmprovider.Provider{
		{Key: "p1", Name: "Provider 1", Type: "openai", Enabled: true,
			Models: []llmprovider.ModelInfo{{ID: "gpt-4o", Name: "GPT-4o", Capabilities: []string{"vision", "tool_calls"}, Enabled: true}},
			Owner:  llmprovider.ProviderOwner{Type: "system"}},
		{Key: llmprovider.ScopedKey(&p2Owner, "p2"), Name: "Provider 2", Type: "anthropic", Enabled: false,
			Models: []llmprovider.ModelInfo{{ID: "claude-3", Name: "Claude 3", Capabilities: []string{"tool_calls"}, Enabled: true}},
			Owner:  p2Owner},
		{Key: "p3", Name: "Provider 3", Type: "openai", Enabled: true,
			Models: []llmprovider.ModelInfo{{ID: "gpt-4o-mini", Name: "GPT-4o Mini", Capabilities: []string{"streaming"}, Enabled: true}},
			Owner:  llmprovider.ProviderOwner{Type: "system"}},
	}
	for i := range providers {
		_, err := r.Create(&providers[i])
		require.NoError(t, err)
	}

	t.Run("AllDynamic", func(t *testing.T) {
		list, err := r.List(&llmprovider.ProviderFilter{Source: llmprovider.ProviderSourceDynamic})
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(list), 3)
	})

	t.Run("FilterByType", func(t *testing.T) {
		typ := "openai"
		list, err := r.List(&llmprovider.ProviderFilter{
			Source: llmprovider.ProviderSourceDynamic,
			Type:   &typ,
		})
		require.NoError(t, err)
		for _, p := range list {
			assert.Equal(t, "openai", p.Type)
		}
	})

	t.Run("FilterByEnabled", func(t *testing.T) {
		enabled := true
		list, err := r.List(&llmprovider.ProviderFilter{
			Source:  llmprovider.ProviderSourceDynamic,
			Enabled: &enabled,
		})
		require.NoError(t, err)
		for _, p := range list {
			assert.True(t, p.Enabled)
		}
	})

	t.Run("FilterByOwner", func(t *testing.T) {
		list, err := r.List(&llmprovider.ProviderFilter{
			Source: llmprovider.ProviderSourceDynamic,
			Owner:  &llmprovider.ProviderOwner{Type: "user", UserID: "123"},
		})
		require.NoError(t, err)
		for _, p := range list {
			assert.Equal(t, "user", p.Owner.Type)
			assert.Equal(t, "123", p.Owner.UserID)
		}
	})

	t.Run("FilterByCapabilities", func(t *testing.T) {
		list, err := r.List(&llmprovider.ProviderFilter{
			Source:       llmprovider.ProviderSourceDynamic,
			Capabilities: []string{"vision", "tool_calls"},
		})
		require.NoError(t, err)
		for _, p := range list {
			found := false
			for _, m := range p.Models {
				capSet := map[string]bool{}
				for _, c := range m.Capabilities {
					capSet[c] = true
				}
				if capSet["vision"] && capSet["tool_calls"] {
					found = true
					break
				}
			}
			assert.True(t, found, "provider %s should have model matching vision+tool_calls", p.Key)
		}
	})

	t.Run("FilterByKeyword", func(t *testing.T) {
		list, err := r.List(&llmprovider.ProviderFilter{
			Source:  llmprovider.ProviderSourceDynamic,
			Keyword: "Provider 2",
		})
		require.NoError(t, err)
		found := false
		for _, p := range list {
			if p.Name == "Provider 2" {
				found = true
			}
		}
		assert.True(t, found)
	})
}

func TestUpdate(t *testing.T) {
	r := setupRegistry(t)

	p := testProvider
	created, err := r.Create(&p)
	require.NoError(t, err)

	updated := *created
	updated.APIURL = "https://custom.openai.com"
	updated.APIKey = "sk-new-key"

	result, err := r.Update("test-openai", &updated)
	require.NoError(t, err)
	assert.Equal(t, "https://custom.openai.com", result.APIURL)

	// Verify store updated
	got, err := r.Get("test-openai")
	require.NoError(t, err)
	assert.Equal(t, "https://custom.openai.com", got.APIURL)
}

func TestDelete(t *testing.T) {
	r := setupRegistry(t)

	p := testProvider
	created, err := r.Create(&p)
	require.NoError(t, err)
	cid := created.ConnectorID

	err = r.Delete("test-openai")
	require.NoError(t, err)

	// Verify removed from store
	_, err = r.Get("test-openai")
	assert.Error(t, err)

	// Verify connector unregistered
	_, err = connector.Select(cid)
	assert.Error(t, err)
}

func TestReload(t *testing.T) {
	r := setupRegistry(t)

	p := testProvider
	_, err := r.Create(&p)
	require.NoError(t, err)

	// Clear cache to simulate stale state
	c, _ := store.Get("__yao.cache")
	if c != nil {
		c.Del("llmprovider:*")
	}

	err = r.Reload()
	require.NoError(t, err)

	// Should still be able to get the provider
	got, err := r.Get("test-openai")
	require.NoError(t, err)
	assert.Equal(t, "Test OpenAI", got.Name)
}

func TestImportFromConnectors(t *testing.T) {
	r := setupRegistry(t)

	// After Init, builtin connectors should be imported
	list, err := r.List(&llmprovider.ProviderFilter{
		Source: llmprovider.ProviderSourceAll,
	})
	require.NoError(t, err)

	builtinCount := 0
	for _, p := range list {
		if p.Source == llmprovider.ProviderSourceBuiltIn {
			builtinCount++
		}
	}

	// Should have imported some from connector.AIConnectors (if test app has connectors)
	t.Logf("Imported %d builtin providers from connector.AIConnectors (total AIConnectors: %d)", builtinCount, len(connector.AIConnectors))
}

func TestGetPresets(t *testing.T) {
	presets := llmprovider.GetPresets()
	assert.Greater(t, len(presets), 0, "should have at least one preset")

	// Verify openai preset exists
	var openai *llmprovider.ProviderPreset
	for i := range presets {
		if presets[i].Key == "openai" {
			openai = &presets[i]
			break
		}
	}
	require.NotNil(t, openai, "openai preset should exist")
	assert.Equal(t, "OpenAI", openai.Name)
	assert.Equal(t, "openai", openai.Type)
	assert.True(t, openai.RequireKey)
	assert.Greater(t, len(openai.DefaultModels), 0)
}

func TestGetPreset(t *testing.T) {
	p := llmprovider.GetPreset("anthropic")
	require.NotNil(t, p)
	assert.Equal(t, "Anthropic", p.Name)

	none := llmprovider.GetPreset("nonexistent")
	assert.Nil(t, none)
}

func TestEncryptionRoundTrip(t *testing.T) {
	r := setupRegistry(t)
	r.SetEncryptionKey("my-super-secret-key-for-tests")

	p := testProvider
	p.Key = "test-encrypted"
	_, err := r.Create(&p)
	require.NoError(t, err)

	got, err := r.Get("test-encrypted", true)
	require.NoError(t, err)
	assert.Equal(t, "sk-test-xxxxx", got.APIKey, "APIKey should be decrypted on read with withKey=true")

	masked, err := r.Get("test-encrypted")
	require.NoError(t, err)
	assert.NotEqual(t, "sk-test-xxxxx", masked.APIKey, "Get without withKey should mask")
	assert.Contains(t, masked.APIKey, "xxxx")

	// Verify raw store value is encrypted
	s, _ := store.Get("__yao.store")
	raw, ok := s.Get("llmprovider:p:test-encrypted")
	require.True(t, ok)
	m := raw.(map[string]interface{})
	storedKey, _ := m["api_key"].(string)
	assert.True(t, len(storedKey) > 0)
	assert.NotEqual(t, "sk-test-xxxxx", storedKey, "raw stored value should be encrypted")
}

func TestGetConnector(t *testing.T) {
	r := setupRegistry(t)

	p := testProvider
	p.Key = "test-getconn"
	_, err := r.Create(&p)
	require.NoError(t, err)

	conn, err := r.GetConnector("test-getconn")
	require.NoError(t, err)
	assert.NotNil(t, conn)

	setting := conn.Setting()
	assert.NotNil(t, setting)
	host, _ := setting["host"].(string)
	assert.Equal(t, "https://api.openai.com", host)
}

func TestGetSetting(t *testing.T) {
	r := setupRegistry(t)

	p := testProvider
	p.Key = "test-getsetting"
	_, err := r.Create(&p)
	require.NoError(t, err)

	setting, err := r.GetSetting("test-getsetting")
	require.NoError(t, err)
	assert.NotNil(t, setting)
	host, _ := setting["host"].(string)
	assert.Equal(t, "https://api.openai.com", host)
}

func TestGetConnectorNotFound(t *testing.T) {
	r := setupRegistry(t)
	_, err := r.GetConnector("not-exist")
	assert.Error(t, err)
}

func TestGetSettingNotFound(t *testing.T) {
	r := setupRegistry(t)
	_, err := r.GetSetting("not-exist")
	assert.Error(t, err)
}

func TestCreateEmptyKey(t *testing.T) {
	r := setupRegistry(t)
	p := llmprovider.Provider{Name: "No Key"}
	_, err := r.Create(&p)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "key is required")
}

func TestCreateDisabled(t *testing.T) {
	r := setupRegistry(t)
	p := llmprovider.Provider{
		Key:     "test-disabled",
		Name:    "Disabled Provider",
		Type:    "openai",
		APIURL:  "https://api.openai.com",
		Enabled: false,
		Models:  []llmprovider.ModelInfo{{ID: "gpt-4o", Name: "GPT-4o", Capabilities: []string{"streaming"}, Enabled: true}},
		Owner:   llmprovider.ProviderOwner{Type: "system"},
	}
	created, err := r.Create(&p)
	require.NoError(t, err)
	assert.Equal(t, "unconfigured", created.Status)

	// Disabled provider should not have its connector registered
	_, err = connector.Select(created.ConnectorID)
	assert.Error(t, err, "disabled provider should not register connector")
}

func TestOwnerPrefixedIDs(t *testing.T) {
	r := setupRegistry(t)

	cases := []struct {
		key    string
		owner  llmprovider.ProviderOwner
		prefix string
	}{
		{"owner-sys", llmprovider.ProviderOwner{Type: "system"}, "s."},
		{"owner-user", llmprovider.ProviderOwner{Type: "user", UserID: "42"}, "u42."},
		{"owner-team", llmprovider.ProviderOwner{Type: "team", TeamID: "99"}, "t99."},
	}

	for _, tc := range cases {
		t.Run(tc.key, func(t *testing.T) {
			scopedKey := llmprovider.ScopedKey(&tc.owner, tc.key)
			p := llmprovider.Provider{
				Key:     scopedKey,
				Name:    tc.key,
				Type:    "openai",
				APIURL:  "https://api.openai.com",
				Enabled: true,
				Models:  []llmprovider.ModelInfo{{ID: "m1", Name: "M1", Capabilities: []string{"streaming"}, Enabled: true}},
				Owner:   tc.owner,
			}
			created, err := r.Create(&p)
			require.NoError(t, err)
			assert.Contains(t, created.ConnectorID, tc.prefix,
				"ConnectorID for %s owner should contain prefix %s", tc.owner.Type, tc.prefix)

			// Verify connector is registered with the prefixed ID
			_, err = connector.Select(created.ConnectorID)
			assert.NoError(t, err)
		})
	}
}

func TestListBuiltInFilter(t *testing.T) {
	r := setupRegistry(t)

	builtinList, err := r.List(&llmprovider.ProviderFilter{Source: llmprovider.ProviderSourceBuiltIn})
	require.NoError(t, err)
	for _, p := range builtinList {
		assert.Equal(t, llmprovider.ProviderSourceBuiltIn, p.Source)
	}
}

func TestListPresetKeyFilter(t *testing.T) {
	r := setupRegistry(t)

	p := llmprovider.Provider{
		Key:       "from-preset",
		Name:      "From Preset",
		Type:      "openai",
		PresetKey: "openai",
		Enabled:   true,
		Models:    []llmprovider.ModelInfo{{ID: "gpt-4o", Name: "GPT-4o", Capabilities: []string{"streaming"}, Enabled: true}},
		Owner:     llmprovider.ProviderOwner{Type: "system"},
	}
	_, err := r.Create(&p)
	require.NoError(t, err)

	pk := "openai"
	list, err := r.List(&llmprovider.ProviderFilter{
		Source:    llmprovider.ProviderSourceDynamic,
		PresetKey: &pk,
	})
	require.NoError(t, err)
	found := false
	for _, item := range list {
		if item.Key == "from-preset" {
			found = true
			assert.Equal(t, "openai", item.PresetKey)
		}
	}
	assert.True(t, found)
}

func TestDefaultModelFallback(t *testing.T) {
	r := setupRegistry(t)

	// Provider with no enabled models — should use first model ID as default
	p := llmprovider.Provider{
		Key:     "test-fallback",
		Name:    "Fallback",
		Type:    "openai",
		APIURL:  "https://api.openai.com",
		Enabled: true,
		Models:  []llmprovider.ModelInfo{{ID: "only-model", Name: "Only", Capabilities: []string{"streaming"}, Enabled: false}},
		Owner:   llmprovider.ProviderOwner{Type: "system"},
	}
	created, err := r.Create(&p)
	require.NoError(t, err)

	// Connector should still be registered using the fallback model
	conn, cerr := connector.Select(created.ConnectorID)
	require.NoError(t, cerr)
	setting := conn.Setting()
	model, _ := setting["model"].(string)
	assert.Equal(t, "only-model", model)
}

func TestMaskShortKey(t *testing.T) {
	r := setupRegistry(t)

	p := llmprovider.Provider{
		Key:     "test-shortkey",
		Name:    "Short",
		Type:    "openai",
		APIKey:  "ab",
		Enabled: true,
		Models:  []llmprovider.ModelInfo{{ID: "m", Name: "M", Capabilities: []string{"streaming"}, Enabled: true}},
		Owner:   llmprovider.ProviderOwner{Type: "system"},
	}
	_, err := r.Create(&p)
	require.NoError(t, err)

	masked, err := r.GetMasked("test-shortkey")
	require.NoError(t, err)
	// Short keys should be fully masked
	assert.Equal(t, "**", masked.APIKey)
}

func TestConcurrency(t *testing.T) {
	r := setupRegistry(t)

	var wg sync.WaitGroup
	errCh := make(chan error, 30)

	// Concurrent creates with unique keys
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			p := llmprovider.Provider{
				Key:     fmt.Sprintf("conc-%d", idx),
				Name:    fmt.Sprintf("Concurrent %d", idx),
				Type:    "openai",
				APIURL:  "https://api.openai.com",
				Enabled: true,
				Models:  []llmprovider.ModelInfo{{ID: "gpt-4o", Name: "GPT-4o", Capabilities: []string{"streaming"}, Enabled: true}},
				Owner:   llmprovider.ProviderOwner{Type: "system"},
			}
			if _, err := r.Create(&p); err != nil {
				errCh <- err
			}
		}(i)
	}

	wg.Wait()

	// Concurrent reads
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			_, err := r.Get(fmt.Sprintf("conc-%d", idx))
			if err != nil {
				errCh <- err
			}
		}(i)
	}

	wg.Wait()

	// Concurrent deletes
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			if err := r.Delete(fmt.Sprintf("conc-%d", idx)); err != nil {
				errCh <- err
			}
		}(i)
	}

	wg.Wait()
	close(errCh)

	for err := range errCh {
		t.Errorf("concurrent operation error: %v", err)
	}
}
