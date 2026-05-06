package llmprovider

import (
	"fmt"
	"strings"
	"sync"

	"github.com/yaoapp/gou/connector"
	"github.com/yaoapp/gou/store"
)

// Global is the singleton LLM Provider Registry.
var Global *Registry

// Registry manages LLM providers with CRUD, persistence, cache and runtime sync.
type Registry struct {
	store  store.Store
	cache  store.Store
	encKey string
	mu     sync.RWMutex
}

// Init initializes the global Registry.
// Must be called after store.Load (so __yao.store and __yao.cache are available).
func Init() error {
	s, err := store.Get("__yao.store")
	if err != nil {
		return fmt.Errorf("llmprovider.Init: %w", err)
	}
	c, _ := store.Get("__yao.cache")

	r := &Registry{store: s, cache: c}
	Global = r

	if err := importFromConnectors(r); err != nil {
		return fmt.Errorf("llmprovider.Init importFromConnectors: %w", err)
	}

	return nil
}

// SetEncryptionKey sets the key used for API key encryption at rest.
// Should be called right after Init if encryption is desired.
func (r *Registry) SetEncryptionKey(key string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.encKey = key
}

// shouldExposeKey returns true when the caller explicitly requests plain-text APIKey.
func shouldExposeKey(withKey []bool) bool {
	return len(withKey) > 0 && withKey[0]
}

// Get retrieves a provider by key. Lazily ensures its connector is registered.
// By default the APIKey is masked; pass withKey=true to get the plain-text key
// (only for internal LLM-request paths).
func (r *Registry) Get(key string, withKey ...bool) (*Provider, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	p, err := storeGet(r.store, r.cache, key, r.encKey)
	if err != nil {
		return nil, err
	}

	_ = ensureConnector(p)

	if !shouldExposeKey(withKey) {
		cp := *p
		cp.APIKey = maskAPIKey(cp.APIKey)
		return &cp, nil
	}
	return p, nil
}

// GetByConnectorID finds a provider by its ConnectorID field (linear scan).
// Use when the caller has a ConnectorID but not the store Key.
// By default the APIKey is masked; pass withKey=true to get the plain-text key.
func (r *Registry) GetByConnectorID(cid string, withKey ...bool) (*Provider, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	keys, err := indexGet(r.store, r.cache)
	if err != nil {
		return nil, err
	}

	for _, key := range keys {
		p, err := storeGet(r.store, r.cache, key, r.encKey)
		if err != nil {
			continue
		}
		if p.ConnectorID == cid {
			_ = ensureConnector(p)
			if !shouldExposeKey(withKey) {
				cp := *p
				cp.APIKey = maskAPIKey(cp.APIKey)
				return &cp, nil
			}
			return p, nil
		}
	}
	return nil, fmt.Errorf("provider with connector_id %q not found", cid)
}

// Deprecated: GetMasked is equivalent to Get(key) since Get now masks by default.
func (r *Registry) GetMasked(key string) (*Provider, error) {
	return r.Get(key)
}

// Create adds a new provider. Persists, caches, registers connector, and updates index.
func (r *Registry) Create(p *Provider) (*Provider, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if p.Key == "" {
		return nil, fmt.Errorf("provider key is required")
	}
	if r.store.Has(storeKey(p.Key)) {
		return nil, fmt.Errorf("provider %s already exists", p.Key)
	}

	if p.Source == "" {
		p.Source = ProviderSourceDynamic
	}
	if p.ConnectorID == "" {
		p.ConnectorID = connectorID(p)
	}
	if p.Status == "" {
		p.Status = "unconfigured"
	}

	if err := storeSet(r.store, r.cache, p, r.encKey); err != nil {
		return nil, err
	}
	if err := indexAdd(r.store, r.cache, p.Key); err != nil {
		return nil, err
	}

	if p.Enabled {
		_ = ensureConnector(p)
	}

	return p, nil
}

// Update modifies an existing provider. Hot-replaces the connector if needed.
func (r *Registry) Update(key string, p *Provider) (*Provider, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	old, err := storeGet(r.store, r.cache, key, r.encKey)
	if err != nil {
		return nil, err
	}

	p.Key = key
	if p.Source == "" {
		p.Source = old.Source
	}
	if p.ConnectorID == "" {
		p.ConnectorID = old.ConnectorID
	}
	if p.Owner == (ProviderOwner{}) {
		p.Owner = old.Owner
	}

	_ = unregisterConnector(old)

	if err := storeSet(r.store, r.cache, p, r.encKey); err != nil {
		return nil, err
	}

	if p.Enabled {
		_ = ensureConnector(p)
	}

	return p, nil
}

// Delete removes a provider by key. Unregisters connector, deletes store/cache/index.
func (r *Registry) Delete(key string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	p, err := storeGet(r.store, r.cache, key, r.encKey)
	if err != nil {
		return err
	}

	_ = unregisterConnector(p)

	if err := storeDel(r.store, r.cache, key); err != nil {
		return err
	}
	return indexRemove(r.store, r.cache, key)
}

// List returns providers matching the filter.
// By default the APIKey is masked; pass withKey=true to get plain-text keys.
func (r *Registry) List(filter *ProviderFilter, withKey ...bool) ([]Provider, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	keys, err := indexGet(r.store, r.cache)
	if err != nil {
		return nil, err
	}

	expose := shouldExposeKey(withKey)
	var result []Provider
	for _, key := range keys {
		p, err := storeGet(r.store, r.cache, key, r.encKey)
		if err != nil {
			continue
		}
		if filter != nil && !matchFilter(p, filter) {
			continue
		}
		cp := *p
		if !expose {
			cp.APIKey = maskAPIKey(cp.APIKey)
		}
		result = append(result, cp)
	}
	return result, nil
}

// Reload re-reads all providers from persistent store and rebuilds cache + connectors.
func (r *Registry) Reload() error {
	r.mu.Lock()
	defer r.mu.Unlock()

	keys, err := indexGet(r.store, nil)
	if err != nil {
		return err
	}

	for _, key := range keys {
		p, err := storeGet(r.store, nil, key, r.encKey)
		if err != nil {
			continue
		}
		m, err := providerToMap(p, r.encKey)
		if err != nil {
			continue
		}
		if r.cache != nil {
			r.cache.Set(storeKey(key), m, 0)
		}
		if p.Source == ProviderSourceDynamic && p.Enabled {
			_ = ensureConnector(p)
		}
	}
	return nil
}

// GetConnector returns the runtime connector for a given provider key.
func (r *Registry) GetConnector(key string) (connector.Connector, error) {
	p, err := r.Get(key, true)
	if err != nil {
		return nil, err
	}
	cid := p.ConnectorID
	if cid == "" {
		cid = connectorID(p)
	}
	return connector.Select(cid)
}

// GetSetting returns the runtime connector setting map for a given provider key.
func (r *Registry) GetSetting(key string) (map[string]interface{}, error) {
	conn, err := r.GetConnector(key)
	if err != nil {
		return nil, err
	}
	return conn.Setting(), nil
}

// matchFilter checks if a provider matches the given filter.
func matchFilter(p *Provider, f *ProviderFilter) bool {
	src := f.Source
	if src == "" {
		src = ProviderSourceDynamic
	}
	if src != ProviderSourceAll && p.Source != src {
		return false
	}

	if f.Owner != nil {
		if f.Owner.Type != "" && p.Owner.Type != f.Owner.Type {
			return false
		}
		if f.Owner.UserID != "" && p.Owner.UserID != f.Owner.UserID {
			return false
		}
		if f.Owner.TeamID != "" && p.Owner.TeamID != f.Owner.TeamID {
			return false
		}
	}

	if f.Enabled != nil && p.Enabled != *f.Enabled {
		return false
	}

	if f.Type != nil && p.Type != *f.Type {
		return false
	}

	if f.PresetKey != nil && p.PresetKey != *f.PresetKey {
		return false
	}

	if len(f.Capabilities) > 0 && !matchCapabilities(p, f.Capabilities) {
		return false
	}

	if f.Keyword != "" {
		kw := strings.ToLower(f.Keyword)
		if !strings.Contains(strings.ToLower(p.Name), kw) &&
			!strings.Contains(strings.ToLower(p.Key), kw) {
			return false
		}
	}

	return true
}

// matchCapabilities returns true if at least one model in the provider
// satisfies ALL of the required capabilities (AND logic).
func matchCapabilities(p *Provider, required []string) bool {
	for _, m := range p.Models {
		if !m.Enabled {
			continue
		}
		capSet := make(map[string]bool, len(m.Capabilities))
		for _, c := range m.Capabilities {
			capSet[c] = true
		}
		allMatch := true
		for _, req := range required {
			if !capSet[req] {
				allMatch = false
				break
			}
		}
		if allMatch {
			return true
		}
	}
	return false
}
