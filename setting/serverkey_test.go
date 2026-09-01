package setting_test

import (
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/yaoapp/gou/store"
	"github.com/yaoapp/yao/setting"
)

func TestIsServerKeyFormat(t *testing.T) {
	tests := []struct {
		token string
		want  bool
	}{
		{"yao-sk:abc123def456", true},
		{"yao-sk:x", true},
		{"yao-sk:", false}, // prefix only, no content
		{"yao-", false},    // API key prefix, not server key
		{"yao-abc", false}, // API key
		{"bearer-token", false},
		{"", false},
	}
	for _, tt := range tests {
		if got := setting.IsServerKeyFormat(tt.token); got != tt.want {
			t.Errorf("IsServerKeyFormat(%q) = %v, want %v", tt.token, got, tt.want)
		}
	}
}

func TestCreateServerKey(t *testing.T) {
	setupRegistry(t)

	plainKey, keyID, err := setting.CreateServerKey("test-node")
	if err != nil {
		t.Fatalf("CreateServerKey: %v", err)
	}

	if !strings.HasPrefix(plainKey, "yao-sk:") {
		t.Errorf("plainKey should start with yao-sk:, got %q", plainKey)
	}
	if !strings.HasPrefix(keyID, "sk-") {
		t.Errorf("keyID should start with sk-, got %q", keyID)
	}
}

func TestValidateServerKey(t *testing.T) {
	setupRegistry(t)

	plainKey, _, err := setting.CreateServerKey("validate-test")
	if err != nil {
		t.Fatalf("CreateServerKey: %v", err)
	}

	// Valid key
	keyID, err := setting.ValidateServerKey(plainKey)
	if err != nil {
		t.Fatalf("ValidateServerKey should succeed: %v", err)
	}
	if keyID == "" {
		t.Error("keyID should not be empty")
	}

	// Invalid key
	_, err = setting.ValidateServerKey("yao-sk:invalid-key")
	if err == nil {
		t.Error("ValidateServerKey should fail for invalid key")
	}

	// Revoked key
	if err := setting.RevokeServerKey(keyID); err != nil {
		t.Fatalf("RevokeServerKey: %v", err)
	}
	_, err = setting.ValidateServerKey(plainKey)
	if err == nil {
		t.Error("ValidateServerKey should fail for revoked key")
	}
}

// TestValidateServerKey_CrossProcessCreate simulates a key created by a
// separate CLI process: the key exists in __yao.store (persistent) but the
// __yao.cache (in-memory LRU) has a stale snapshot without it.
func TestValidateServerKey_CrossProcessCreate(t *testing.T) {
	setupRegistry(t)

	// Step 1: create key1 normally — this populates ALL cache layers
	_, _, err := setting.CreateServerKey("existing-node")
	require.NoError(t, err)
	setting.Global.Flush()

	// Step 2: create key2 normally (it goes into both store and cache)
	plainKey2, _, err := setting.CreateServerKey("new-cli-node")
	require.NoError(t, err)
	setting.Global.Flush()

	// Step 3: overwrite __yao.cache with a stale index that only has key1's
	// hash, simulating a server whose LRU cache was populated before key2
	// was created by a CLI process.
	scope := setting.ScopeID{Scope: setting.ScopeSystem}
	indexData, err := setting.Global.Get(scope, "server_keys_index")
	require.NoError(t, err)

	key2Hash := setting.HashServerKey(plainKey2)
	staleIndex := make(map[string]interface{})
	for k, v := range indexData {
		if k != key2Hash {
			staleIndex[k] = v
		}
	}

	cache, _ := store.Get("__yao.cache")
	require.NotNil(t, cache)
	err = cache.Set("setting:s:server_keys_index", staleIndex, 0)
	require.NoError(t, err)

	// Step 4: validate key2 — cache has stale data, fallback should read DB
	keyID, err := setting.ValidateServerKey(plainKey2)
	assert.NoError(t, err, "key created by CLI should be found via DB fallback")
	assert.NotEmpty(t, keyID)
}

// TestValidateServerKey_CrossProcessRevoke simulates a key revoked by a
// separate CLI process: the persistent store has revoked=true but the
// in-memory cache still has the old non-revoked entry.
func TestValidateServerKey_CrossProcessRevoke(t *testing.T) {
	setupRegistry(t)

	// Step 1: create and validate a key (populates all caches)
	plainKey, keyID, err := setting.CreateServerKey("revoke-test")
	require.NoError(t, err)
	setting.Global.Flush()

	_, err = setting.ValidateServerKey(plainKey)
	require.NoError(t, err)

	// Step 2: revoke via normal path (updates store + cache in this process)
	err = setting.RevokeServerKey(keyID)
	require.NoError(t, err)
	setting.Global.Flush()

	// Step 3: put stale (non-revoked) data back into __yao.cache,
	// simulating a server whose cache was populated before CLI revocation.
	cache, _ := store.Get("__yao.cache")
	require.NotNil(t, cache)

	scope := setting.ScopeID{Scope: setting.ScopeSystem}
	freshKeys, err := setting.Global.GetDirect(scope, "server_keys")
	require.NoError(t, err)

	staleKeys := make(map[string]interface{})
	for k, v := range freshKeys {
		staleKeys[k] = v
	}
	if entry, ok := staleKeys[keyID].(map[string]interface{}); ok {
		entry["revoked"] = false
	}
	err = cache.Set("setting:s:server_keys", staleKeys, 0)
	require.NoError(t, err)

	// Also restore hash in cache index (revoke removes it)
	keyHash := setting.HashServerKey(plainKey)
	staleIndex := map[string]interface{}{keyHash: keyID}
	err = cache.Set("setting:s:server_keys_index", staleIndex, 0)
	require.NoError(t, err)

	// Step 4: validate — cache says valid, but GetDirect reads DB (revoked)
	_, err = setting.ValidateServerKey(plainKey)
	assert.Error(t, err, "revoked key should be rejected via DB fallback")
	assert.Contains(t, err.Error(), "revoked")
}

// TestValidateServerKey_CrossProcessExpiry simulates a key that expired
// between cache population and validation.
func TestValidateServerKey_CrossProcessExpiry(t *testing.T) {
	setupRegistry(t)

	// Create a key with very short TTL
	plainKey, _, err := setting.CreateServerKey("expiry-test", 1*time.Millisecond)
	require.NoError(t, err)
	setting.Global.Flush()

	// Wait for expiry
	time.Sleep(10 * time.Millisecond)

	_, err = setting.ValidateServerKey(plainKey)
	assert.Error(t, err, "expired key should be rejected")
	assert.Contains(t, err.Error(), "expired")
}

// TestGetDirect verifies that GetDirect bypasses the LRU cache layer.
func TestGetDirect(t *testing.T) {
	r := setupRegistry(t)
	scope := setting.ScopeID{Scope: setting.ScopeSystem}

	// Write data normally
	_, err := r.Set(scope, "direct-test", map[string]interface{}{"v": "original"})
	require.NoError(t, err)
	r.Flush()

	// Overwrite __yao.cache with stale data
	cache, _ := store.Get("__yao.cache")
	require.NotNil(t, cache)
	err = cache.Set("setting:s:direct-test", map[string]interface{}{"v": "stale"}, 0)
	require.NoError(t, err)

	// Regular Get returns stale cache
	stale, err := r.Get(scope, "direct-test")
	require.NoError(t, err)
	assert.Equal(t, "stale", stale["v"])

	// GetDirect bypasses cache
	fresh, err := r.GetDirect(scope, "direct-test")
	require.NoError(t, err)
	assert.Equal(t, "original", fresh["v"])

	// After GetDirect, cache is refreshed
	cached, err := r.Get(scope, "direct-test")
	require.NoError(t, err)
	assert.Equal(t, "original", cached["v"])
}
