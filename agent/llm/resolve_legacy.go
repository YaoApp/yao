package llm

// Legacy connector resolution — remove this file after full migration to yao/llm DB.

import (
	"fmt"
	"strings"

	"github.com/yaoapp/gou/connector"
	goullm "github.com/yaoapp/gou/llm"
	yaollm "github.com/yaoapp/yao/llm"
	"github.com/yaoapp/yao/llmprovider"
)

// ResolveConnector resolves an LLM connector using the legacy llmprovider registry.
//
// connectorID may be:
//   - explicit connector ID (e.g. "t123.openai:gpt-4o") — resolved directly
//   - role reference with prefix (e.g. "use::light") — resolved via llmprovider roles
//   - empty string — falls back to the "default" role
//
// Priority for role-based resolution:
//  1. GetRoleBy(role, identity) — user/team scoped setting
//  2. GetRole(role) — system-level default for that role
//  3. GetRoleBy("default", identity) — fallback to "default" role (user/team)
//  4. GetRole("default") — fallback to "default" role (system)
//  5. error — caller decides whether to apply legacy fallback
func ResolveConnector(connectorID string, identity llmprovider.Identity) (connector.Connector, *goullm.Capabilities, error) {

	// Parse use:: prefix to extract role
	role := ""
	if strings.HasPrefix(connectorID, RolePrefix) {
		role = strings.TrimPrefix(connectorID, RolePrefix)
		connectorID = ""
	}

	// Explicit connector ID takes highest priority
	if connectorID != "" {
		return selectWithCapabilities(connectorID)
	}

	// Empty connector with no role → treat as "default"
	if role == "" {
		role = "default"
	}

	if llmprovider.Global == nil {
		return nil, nil, fmt.Errorf("llmprovider not initialized and no explicit connector specified")
	}

	// Resolve by the specified role (e.g. "light", "vision")
	if role != "default" {
		if identity != nil {
			if cid, err := llmprovider.Global.GetRoleBy(role, identity); err == nil && cid != "" {
				if conn, caps, err := selectWithCapabilities(cid); err == nil {
					return conn, caps, nil
				}
			}
		}
		if cid, err := llmprovider.Global.GetRole(role); err == nil && cid != "" {
			if conn, caps, err := selectWithCapabilities(cid); err == nil {
				return conn, caps, nil
			}
		}
	}

	// Fallback to "default" role
	if identity != nil {
		if cid, err := llmprovider.Global.GetRoleBy("default", identity); err == nil && cid != "" {
			if conn, caps, err := selectWithCapabilities(cid); err == nil {
				return conn, caps, nil
			}
		}
	}
	if cid, err := llmprovider.Global.GetRole("default"); err == nil && cid != "" {
		if conn, caps, err := selectWithCapabilities(cid); err == nil {
			return conn, caps, nil
		}
	}

	return nil, nil, fmt.Errorf("no connector resolved for role %q", role)
}

func selectWithCapabilities(connectorID string) (connector.Connector, *goullm.Capabilities, error) {
	conn, err := connector.Select(connectorID)
	if err != nil && llmprovider.Global != nil {
		conn, err = llmprovider.Global.GetModel(connectorID)
	}
	if err != nil {
		return nil, nil, err
	}
	caps := GetCapabilitiesFromConn(conn)
	return conn, caps, nil
}

// resolveByConnectorID finds a Provider+Model in yao/llm from a connector ID.
//
// Supported formats (tried in order):
//  1. Legacy "scopePrefix.providerKey:modelID" (e.g. "t123.deepseek_anthropic:deepseek-flash")
//  2. Simple "providerKey.modelID" (e.g. "openai.gpt-4o")
//
// Returns nil,nil on no match.
func resolveByConnectorID(connectorID string) (*yaollm.Provider, *yaollm.Model) {
	// Format 1: legacy connector ID with ":" model separator
	if colIdx := strings.IndexByte(connectorID, ':'); colIdx > 0 {
		providerCID := connectorID[:colIdx]
		modelID := connectorID[colIdx+1:]
		if p, m := findByProviderCID(providerCID, modelID); p != nil {
			return p, m
		}
	}

	// Format 2: simple "providerKey.modelID"
	if dotIdx := strings.LastIndexByte(connectorID, '.'); dotIdx > 0 {
		providerKey := yaollm.ProviderID(connectorID[:dotIdx])
		modelID := connectorID[dotIdx+1:]
		if p, ok := yaollm.Global.FindProviderByKey(providerKey); ok {
			if m, err := yaollm.FindModelByModelID(p.ID, modelID); err == nil && m != nil {
				return p, m
			}
		}
	}

	return nil, nil
}

// findByProviderCID resolves a legacy provider connector ID (e.g. "t123.deepseek_anthropic")
// to a Provider by stripping the scope prefix and matching on ProviderID or PresetKey,
// then looks up the model under that provider by model_id.
func findByProviderCID(providerCID, modelID string) (*yaollm.Provider, *yaollm.Model) {
	// Strip scope prefix: "t123.deepseek_anthropic" → "deepseek_anthropic"
	key := providerCID
	if dotIdx := strings.IndexByte(providerCID, '.'); dotIdx >= 0 {
		key = providerCID[dotIdx+1:]
	}

	// Locate the provider (carries API key + base URL).
	var provider *yaollm.Provider
	if p, ok := yaollm.Global.FindProviderByPresetKey(key); ok {
		provider = p
	} else if p, ok := yaollm.Global.FindProviderByKey(yaollm.ProviderID(key)); ok {
		provider = p
	}
	if provider == nil {
		return nil, nil
	}

	// Try model under this provider by exact model_id.
	if m, err := yaollm.FindModelByModelID(provider.ID, modelID); err == nil && m != nil {
		return provider, m
	}

	// Fallback: the model ID may be an old variant that was merged into a
	// base model during migration. Check variant_aliases in each model's
	// Metadata under this provider.
	if m := findModelByAlias(provider.ID, modelID); m != nil {
		return provider, m
	}

	return nil, nil
}

// findModelByAlias searches all models under a provider for one whose
// Metadata["variant_aliases"] contains the given alias.
func findModelByAlias(providerRef int, alias string) *yaollm.Model {
	models, err := yaollm.ListModels(providerRef)
	if err != nil {
		return nil
	}
	for i := range models {
		raw, ok := models[i].Metadata["variant_aliases"]
		if !ok {
			continue
		}
		switch aliases := raw.(type) {
		case []interface{}:
			for _, a := range aliases {
				if s, ok := a.(string); ok && s == alias {
					return &models[i]
				}
			}
		case []string:
			for _, s := range aliases {
				if s == alias {
					return &models[i]
				}
			}
		}
	}
	return nil
}
