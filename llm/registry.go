package llm

import (
	"fmt"
	"sync"
)

// Registry is the global runtime manager for LLM providers and models.
// It caches provider+model data from the database for fast lookups.
type Registry struct {
	mu        sync.RWMutex
	providers []Provider
	loaded    bool
}

// Global is the package-level Registry singleton.
var Global = &Registry{}

// Init initializes the Registry by loading all providers from the database.
// Called from engine.Load after model tables are created.
func Init() error {
	return Global.Reload()
}

// Reload refreshes the cached provider list from the database.
func (r *Registry) Reload() error {
	r.mu.Lock()
	defer r.mu.Unlock()

	providers, err := ListProviders("", "", "")
	if err != nil {
		return fmt.Errorf("llm: reload providers: %w", err)
	}
	r.providers = providers
	r.loaded = true
	return nil
}

// GetProviders returns a copy of all cached providers.
func (r *Registry) GetProviders() []Provider {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]Provider, len(r.providers))
	copy(out, r.providers)
	return out
}

// GetEnabledProviders returns cached providers that are enabled.
func (r *Registry) GetEnabledProviders() []Provider {
	r.mu.RLock()
	defer r.mu.RUnlock()
	var out []Provider
	for _, p := range r.providers {
		if p.Enabled {
			out = append(out, p)
		}
	}
	return out
}

// FindProvider finds a cached provider by auto-increment ID.
func (r *Registry) FindProvider(id int) (*Provider, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	for _, p := range r.providers {
		if p.ID == id {
			cp := p
			return &cp, true
		}
	}
	return nil, false
}

// FindProviderByKey finds a cached provider by ProviderID.
func (r *Registry) FindProviderByKey(providerID ProviderID) (*Provider, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	for _, p := range r.providers {
		if p.ProviderID == providerID {
			cp := p
			return &cp, true
		}
	}
	return nil, false
}

// FindProviderByPresetKey finds a cached provider by PresetKey (the old connector Key).
func (r *Registry) FindProviderByPresetKey(presetKey string) (*Provider, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	for _, p := range r.providers {
		if p.PresetKey == presetKey && presetKey != "" {
			cp := p
			return &cp, true
		}
	}
	return nil, false
}

// refreshRegistry reloads the Global registry cache after a provider
// write operation. Errors are silently ignored because a stale cache
// is acceptable — the next explicit Reload() will correct it.
func refreshRegistry() {
	if Global != nil {
		_ = Global.Reload()
	}
}

// ResolveRoleModel resolves a role to a Provider+Model pair using the
// three-level scope cascade. Returns an error if the role is not bound
// or the provider/model is not found.
func (r *Registry) ResolveRoleModel(role, teamID, userID string) (*Provider, *Model, error) {
	rb, err := ResolveRole(role, teamID, userID)
	if err != nil {
		return nil, nil, err
	}
	if rb == nil {
		return nil, nil, fmt.Errorf("llm: role %q is not bound", role)
	}

	p, err := GetProvider(rb.ProviderRef)
	if err != nil {
		return nil, nil, fmt.Errorf("llm: provider %d for role %q: %w", rb.ProviderRef, role, err)
	}

	m, err := FindModelByModelID(rb.ProviderRef, rb.ModelID)
	if err != nil {
		return nil, nil, fmt.Errorf("llm: model %q for role %q: %w", rb.ModelID, role, err)
	}
	if m == nil {
		return nil, nil, fmt.Errorf("llm: model %q not found under provider %d for role %q", rb.ModelID, rb.ProviderRef, role)
	}

	return p, m, nil
}
