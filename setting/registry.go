package setting

import (
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/yaoapp/gou/store"
)

// Global is the singleton Setting Registry.
var Global *Registry

// Registry manages namespaced settings with three-level scope cascade.
type Registry struct {
	store store.Store
	cache store.Store
	mu    sync.RWMutex
}

// Init initializes the global Registry.
// Must be called after store.Load (so __yao.store and __yao.cache are available).
func Init() error {
	s, err := store.Get("__yao.store")
	if err != nil {
		return fmt.Errorf("setting.Init: %w", err)
	}
	c, _ := store.Get("__yao.cache")
	Global = &Registry{store: s, cache: c}
	return nil
}

// Get reads the raw data for a single scope+namespace.
// If one or more dest pointers are provided, the data is also unmarshalled
// into dest[0] (like json.Unmarshal).
func (r *Registry) Get(scope ScopeID, ns string, dest ...interface{}) (map[string]interface{}, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	data, err := storeGet(r.store, r.cache, scope, ns)
	if err != nil {
		return nil, err
	}

	if len(dest) > 0 && dest[0] != nil {
		if err := bindDest(data, dest[0]); err != nil {
			return data, fmt.Errorf("setting bind: %w", err)
		}
	}
	return data, nil
}

// GetMerged reads a namespace across all three scopes and returns a shallow-merged
// result: system <- team <- user (later wins).
// If one or more dest pointers are provided, the merged data is also unmarshalled
// into dest[0].
func (r *Registry) GetMerged(userID, teamID, ns string, dest ...interface{}) (map[string]interface{}, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	merged := make(map[string]interface{})

	if sys, err := storeGet(r.store, r.cache, ScopeID{Scope: ScopeSystem}, ns); err == nil {
		shallowMerge(merged, sys)
	}

	if teamID != "" {
		if team, err := storeGet(r.store, r.cache, ScopeID{Scope: ScopeTeam, TeamID: teamID}, ns); err == nil {
			shallowMerge(merged, team)
		}
	}

	if userID != "" {
		if user, err := storeGet(r.store, r.cache, ScopeID{Scope: ScopeUser, UserID: userID}, ns); err == nil {
			shallowMerge(merged, user)
		}
	}

	if len(merged) == 0 {
		return nil, fmt.Errorf("setting %s: no data found at any scope", ns)
	}

	if len(dest) > 0 && dest[0] != nil {
		if err := bindDest(merged, dest[0]); err != nil {
			return merged, fmt.Errorf("setting bind: %w", err)
		}
	}
	return merged, nil
}

// GetMergedBatch reads multiple namespaces across all three scopes in one pass,
// returning a map of namespace → shallow-merged data. Namespaces with no data
// at any scope are omitted from the result.
func (r *Registry) GetMergedBatch(userID, teamID string, namespaces []string) map[string]map[string]interface{} {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make(map[string]map[string]interface{}, len(namespaces))
	for _, ns := range namespaces {
		merged := make(map[string]interface{})
		if sys, err := storeGet(r.store, r.cache, ScopeID{Scope: ScopeSystem}, ns); err == nil {
			shallowMerge(merged, sys)
		}
		if teamID != "" {
			if team, err := storeGet(r.store, r.cache, ScopeID{Scope: ScopeTeam, TeamID: teamID}, ns); err == nil {
				shallowMerge(merged, team)
			}
		}
		if userID != "" {
			if user, err := storeGet(r.store, r.cache, ScopeID{Scope: ScopeUser, UserID: userID}, ns); err == nil {
				shallowMerge(merged, user)
			}
		}
		if len(merged) > 0 {
			result[ns] = merged
		}
	}
	return result
}

// Set writes (or overwrites) a namespace entry for the given scope.
func (r *Registry) Set(scope ScopeID, ns string, data map[string]interface{}) (*Entry, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if ns == "" {
		return nil, fmt.Errorf("namespace is required")
	}

	if err := storeSet(r.store, r.cache, scope, ns, data); err != nil {
		return nil, err
	}
	if err := indexAdd(r.store, r.cache, scope, ns); err != nil {
		return nil, err
	}

	return &Entry{
		Namespace: ns,
		Scope:     scope,
		Data:      data,
		UpdatedAt: time.Now().UTC().Format(time.RFC3339),
	}, nil
}

// Delete removes a namespace entry from a given scope.
func (r *Registry) Delete(scope ScopeID, ns string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	sk := storeKey(scope, ns)
	if !r.store.Has(sk) {
		return fmt.Errorf("setting %s/%s not found", scopePrefix(scope), ns)
	}

	if err := storeDel(r.store, r.cache, scope, ns); err != nil {
		return err
	}
	return indexRemove(r.store, r.cache, scope, ns)
}

// ListNamespaces returns all namespace names stored under the given scope.
func (r *Registry) ListNamespaces(scope ScopeID) ([]string, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return indexGet(r.store, r.cache, scope)
}

// Reload clears the cache and re-populates it from the persistent store.
func (r *Registry) Reload() error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.cache != nil {
		_ = r.cache.Del(keyPrefix + "*")
	}

	for _, scope := range []ScopeID{
		{Scope: ScopeSystem},
	} {
		keys, err := indexGet(r.store, nil, scope)
		if err != nil {
			continue
		}
		ik := indexKey(scope)
		raw, ok := r.store.Get(ik)
		if ok && r.cache != nil {
			r.cache.Set(ik, raw, 0)
		}
		for _, ns := range keys {
			if data, err := storeGet(r.store, nil, scope, ns); err == nil && r.cache != nil {
				r.cache.Set(storeKey(scope, ns), data, 0)
			}
		}
	}
	return nil
}

// shallowMerge copies all keys from src into dst (overwrites existing keys).
func shallowMerge(dst, src map[string]interface{}) {
	for k, v := range src {
		dst[k] = v
	}
}

// bindDest marshals data to JSON and then unmarshals into the dest pointer.
func bindDest(data map[string]interface{}, dest interface{}) error {
	raw, err := json.Marshal(data)
	if err != nil {
		return err
	}
	return json.Unmarshal(raw, dest)
}
