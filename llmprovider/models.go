package llmprovider

import (
	"fmt"
	"strings"

	"github.com/yaoapp/gou/connector"
	goullm "github.com/yaoapp/gou/llm"
)

// ---------------------------------------------------------------------------
// GetModel — by connectorID
// ---------------------------------------------------------------------------

// GetModel returns the runtime connector for a given connectorID.
// Lookup order:
//  1. connector.Select (already registered in runtime)
//  2. Model-level ID with ":" separator (e.g. "t123.openai:gpt-4o")
//  3. r.Get by store Key (works when connectorID == Key, e.g. builtin)
//  4. r.GetByConnectorID (linear scan by ConnectorID field, for dynamic providers)
func (r *Registry) GetModel(connectorID string) (connector.Connector, error) {
	if conn, err := connector.Select(connectorID); err == nil {
		return conn, nil
	}

	// Path 2: model-level ID "providerCID:modelID"
	if parts := strings.SplitN(connectorID, ":", 2); len(parts) == 2 {
		return r.getModelConnector(parts[0], parts[1])
	}

	// Path 3: try by Key (fast, works for builtin where Key == ConnectorID)
	if p, err := r.Get(connectorID, true); err == nil {
		if eerr := ensureConnector(p); eerr != nil {
			return nil, fmt.Errorf("model %q ensure connector: %w", connectorID, eerr)
		}
		cid := p.ConnectorID
		if cid == "" {
			cid = connectorID
		}
		return connector.Select(cid)
	}

	// Path 4: reverse lookup by ConnectorID field (dynamic providers where Key != ConnectorID)
	p, err := r.GetByConnectorID(connectorID, true)
	if err != nil {
		return nil, fmt.Errorf("model %q not found", connectorID)
	}

	cid := p.ConnectorID
	if cid == "" {
		cid = connectorID
	}
	return connector.Select(cid)
}

// getModelConnector finds a provider by connectorID, locates the model, and
// ensures a per-model connector is registered in the runtime.
func (r *Registry) getModelConnector(providerCID, modelID string) (connector.Connector, error) {
	p, err := r.GetByConnectorID(providerCID, true)
	if err != nil {
		if p2, err2 := r.Get(providerCID, true); err2 == nil {
			p = p2
		} else {
			return nil, fmt.Errorf("provider %q not found for model %q", providerCID, modelID)
		}
	}

	var model *ModelInfo
	for i, m := range p.Models {
		if m.ID == modelID {
			model = &p.Models[i]
			break
		}
	}
	if model == nil {
		return nil, fmt.Errorf("model %q not found in provider %q", modelID, providerCID)
	}

	if err := ensureModelConnector(p, model); err != nil {
		return nil, err
	}

	cid := providerCID + ":" + modelID
	return connector.Select(cid)
}

// ---------------------------------------------------------------------------
// GetRoleModel — role → connector
// ---------------------------------------------------------------------------

// GetRoleModel returns the connector for a role at system scope.
func (r *Registry) GetRoleModel(role string) (connector.Connector, error) {
	cid, err := r.GetRole(role)
	if err != nil {
		return nil, err
	}
	return r.GetModel(cid)
}

// GetRoleModelByUser returns the connector for a role, merged user > system.
func (r *Registry) GetRoleModelByUser(role, userID string) (connector.Connector, error) {
	cid, err := r.GetRoleByUser(role, userID)
	if err != nil {
		return nil, err
	}
	return r.GetModel(cid)
}

// GetRoleModelByTeam returns the connector for a role, merged team > system.
func (r *Registry) GetRoleModelByTeam(role, teamID string) (connector.Connector, error) {
	cid, err := r.GetRoleByTeam(role, teamID)
	if err != nil {
		return nil, err
	}
	return r.GetModel(cid)
}

// ---------------------------------------------------------------------------
// Built-in role shortcuts
// ---------------------------------------------------------------------------

func (r *Registry) GetDefaultModel() (connector.Connector, error) { return r.GetRoleModel("default") }
func (r *Registry) GetDefaultModelByUser(userID string) (connector.Connector, error) {
	return r.GetRoleModelByUser("default", userID)
}
func (r *Registry) GetDefaultModelByTeam(teamID string) (connector.Connector, error) {
	return r.GetRoleModelByTeam("default", teamID)
}
func (r *Registry) GetVisionModel() (connector.Connector, error) { return r.GetRoleModel("vision") }
func (r *Registry) GetVisionModelByUser(userID string) (connector.Connector, error) {
	return r.GetRoleModelByUser("vision", userID)
}
func (r *Registry) GetVisionModelByTeam(teamID string) (connector.Connector, error) {
	return r.GetRoleModelByTeam("vision", teamID)
}
func (r *Registry) GetAudioModel() (connector.Connector, error) { return r.GetRoleModel("audio") }
func (r *Registry) GetAudioModelByUser(userID string) (connector.Connector, error) {
	return r.GetRoleModelByUser("audio", userID)
}
func (r *Registry) GetAudioModelByTeam(teamID string) (connector.Connector, error) {
	return r.GetRoleModelByTeam("audio", teamID)
}
func (r *Registry) GetEmbeddingModel() (connector.Connector, error) {
	return r.GetRoleModel("embedding")
}
func (r *Registry) GetEmbeddingModelByUser(userID string) (connector.Connector, error) {
	return r.GetRoleModelByUser("embedding", userID)
}
func (r *Registry) GetEmbeddingModelByTeam(teamID string) (connector.Connector, error) {
	return r.GetRoleModelByTeam("embedding", teamID)
}

// ---------------------------------------------------------------------------
// Capabilities
// ---------------------------------------------------------------------------

// GetCapabilities returns capabilities for a connector by connectorID.
func (r *Registry) GetCapabilities(connectorID string) (*goullm.Capabilities, error) {
	conn, err := r.GetModel(connectorID)
	if err != nil {
		return nil, err
	}
	return capabilitiesFromConn(conn), nil
}

// GetRoleCapabilities returns capabilities for a role at system scope.
func (r *Registry) GetRoleCapabilities(role string) (*goullm.Capabilities, error) {
	conn, err := r.GetRoleModel(role)
	if err != nil {
		return nil, err
	}
	return capabilitiesFromConn(conn), nil
}

// GetRoleCapabilitiesByUser returns capabilities for a role, merged user > system.
func (r *Registry) GetRoleCapabilitiesByUser(role, userID string) (*goullm.Capabilities, error) {
	conn, err := r.GetRoleModelByUser(role, userID)
	if err != nil {
		return nil, err
	}
	return capabilitiesFromConn(conn), nil
}

// GetRoleCapabilitiesByTeam returns capabilities for a role, merged team > system.
func (r *Registry) GetRoleCapabilitiesByTeam(role, teamID string) (*goullm.Capabilities, error) {
	conn, err := r.GetRoleModelByTeam(role, teamID)
	if err != nil {
		return nil, err
	}
	return capabilitiesFromConn(conn), nil
}

// ---------------------------------------------------------------------------
// ListModels
// ---------------------------------------------------------------------------

// ListModels returns all enabled models as []connector.Option (system scope, no owner filter).
func (r *Registry) ListModels() []connector.Option {
	return r.listModels(nil)
}

// ListModelsByUser returns builtin + user-owned dynamic models.
func (r *Registry) ListModelsByUser(userID string) []connector.Option {
	return r.listModels(&ProviderOwner{Type: "user", UserID: userID})
}

// ListModelsByTeam returns builtin + team-owned dynamic models.
func (r *Registry) ListModelsByTeam(teamID string) []connector.Option {
	return r.listModels(&ProviderOwner{Type: "team", TeamID: teamID})
}

// ListModelsBy returns models scoped to the caller's identity (team > user).
func (r *Registry) ListModelsBy(id Identity) []connector.Option {
	if id.GetTeamID() != "" {
		return r.ListModelsByTeam(id.GetTeamID())
	}
	return r.ListModelsByUser(id.GetUserID())
}

// ---------------------------------------------------------------------------
// By — Identity-scoped convenience methods
// ---------------------------------------------------------------------------

// GetRoleBy returns the connectorID for a role, scoped by identity.
func (r *Registry) GetRoleModelBy(role string, id Identity) (connector.Connector, error) {
	if id.GetTeamID() != "" {
		return r.GetRoleModelByTeam(role, id.GetTeamID())
	}
	return r.GetRoleModelByUser(role, id.GetUserID())
}

func (r *Registry) GetDefaultModelBy(id Identity) (connector.Connector, error) {
	return r.GetRoleModelBy("default", id)
}
func (r *Registry) GetVisionModelBy(id Identity) (connector.Connector, error) {
	return r.GetRoleModelBy("vision", id)
}
func (r *Registry) GetAudioModelBy(id Identity) (connector.Connector, error) {
	return r.GetRoleModelBy("audio", id)
}
func (r *Registry) GetEmbeddingModelBy(id Identity) (connector.Connector, error) {
	return r.GetRoleModelBy("embedding", id)
}

func (r *Registry) GetRoleCapabilitiesBy(role string, id Identity) (*goullm.Capabilities, error) {
	conn, err := r.GetRoleModelBy(role, id)
	if err != nil {
		return nil, err
	}
	return capabilitiesFromConn(conn), nil
}

// ---------------------------------------------------------------------------
// internal
// ---------------------------------------------------------------------------

// listModels returns enabled models. When owner is non-nil, returns builtin
// providers plus dynamic providers belonging to that owner.
// Builtin providers are one-connector-per-model; dynamic providers are expanded
// to per-model options here.
func (r *Registry) listModels(owner *ProviderOwner) []connector.Option {
	enabled := true
	providers, err := r.List(&ProviderFilter{
		Source:  ProviderSourceAll,
		Enabled: &enabled,
	}, true)
	if err != nil {
		return nil
	}

	var result []connector.Option
	for _, p := range providers {
		if owner != nil && p.Source == ProviderSourceDynamic {
			if !ownerMatch(&p.Owner, owner) {
				continue
			}
		}

		if p.Source == ProviderSourceDynamic && len(p.Models) > 0 {
			for _, m := range p.Models {
				if !m.Enabled {
					continue
				}
				_ = ensureModelConnector(&p, &m)
				cid := p.ConnectorID + ":" + m.ID
				label := p.Name + " / " + m.Name
				if m.Name == "" {
					label = p.Name + " / " + m.ID
				}
				result = append(result, connector.Option{
					Label: label,
					Value: cid,
				})
			}
		} else {
			result = append(result, connector.Option{
				Label: p.Name,
				Value: p.ConnectorID,
			})
		}
	}
	return result
}

// ownerMatch returns true if the provider owner matches the requested scope.
func ownerMatch(po, want *ProviderOwner) bool {
	if want.Type == "team" {
		return po.Type == "team" && po.TeamID == want.TeamID
	}
	return po.Type == "user" && po.UserID == want.UserID
}

// capabilitiesFromConn extracts *llm.Capabilities from a connector.
// Prefers LLMConnector.GetCapabilities() when available.
func capabilitiesFromConn(conn connector.Connector) *goullm.Capabilities {
	if conn == nil {
		return defaultCaps()
	}

	if lc, ok := conn.(goullm.LLMConnector); ok {
		if caps := lc.GetCapabilities(); caps != nil {
			return caps
		}
	}

	settings := conn.Setting()
	if settings != nil {
		if caps, ok := settings["capabilities"]; ok {
			if c, ok := caps.(*goullm.Capabilities); ok {
				return c
			}
			if c, ok := caps.(goullm.Capabilities); ok {
				return &c
			}
		}
	}
	return defaultCaps()
}

// capsToMap converts Capabilities to map[string]interface{} for process handlers.
// Delegates to the canonical Capabilities.ToMap() method in gou/llm.
func capsToMap(caps *goullm.Capabilities) map[string]interface{} {
	return caps.ToMap()
}

func defaultCaps() *goullm.Capabilities {
	return &goullm.Capabilities{
		Vision:                false,
		ToolCalls:             false,
		Audio:                 false,
		Reasoning:             false,
		Streaming:             false,
		JSON:                  false,
		Multimodal:            false,
		TemperatureAdjustable: true,
	}
}
