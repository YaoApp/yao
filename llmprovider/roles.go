package llmprovider

import (
	"fmt"

	"github.com/yaoapp/yao/setting"
)

// RolesNamespace is the setting namespace for LLM role assignments.
const RolesNamespace = "llm.roles"

// SetDefaults writes agent.yml system-level role defaults into setting.Global
// under ScopeSystem. roles maps role names (e.g. "default", "vision") to connectorIDs.
func (r *Registry) SetDefaults(roles map[string]string) error {
	if setting.Global == nil {
		return fmt.Errorf("setting registry not initialized")
	}

	data := make(map[string]interface{})
	for role, cid := range roles {
		p, err := r.Get(cid, true)
		if err != nil {
			// Builtin providers: Key == ConnectorID, use connectorID directly
			data[role] = map[string]interface{}{
				"provider": cid,
				"model":    "",
			}
			continue
		}
		data[role] = map[string]interface{}{
			"provider": p.Key,
			"model":    defaultModel(p),
		}
	}

	_, err := setting.Global.Set(
		setting.ScopeID{Scope: setting.ScopeSystem},
		RolesNamespace,
		data,
	)
	return err
}

// GetRole returns the connectorID for a role at system scope.
func (r *Registry) GetRole(role string) (string, error) {
	return r.resolveRole(role, "", "")
}

// GetRoleByUser returns the connectorID for a role, merged user > system.
func (r *Registry) GetRoleByUser(role, userID string) (string, error) {
	return r.resolveRole(role, userID, "")
}

// GetRoleByTeam returns the connectorID for a role, merged team > system.
func (r *Registry) GetRoleByTeam(role, teamID string) (string, error) {
	return r.resolveRole(role, "", teamID)
}

// GetRoleBy returns the connectorID for a role, scoped by identity (team > user).
func (r *Registry) GetRoleBy(role string, id Identity) (string, error) {
	if id.GetTeamID() != "" {
		return r.GetRoleByTeam(role, id.GetTeamID())
	}
	return r.GetRoleByUser(role, id.GetUserID())
}

// ListRoles returns all role assignments at system scope.
func (r *Registry) ListRoles() (map[string]RoleTarget, error) {
	return r.listRoles("", "")
}

// ListRolesByUser returns all role assignments, merged user > system.
func (r *Registry) ListRolesByUser(userID string) (map[string]RoleTarget, error) {
	return r.listRoles(userID, "")
}

// ListRolesByTeam returns all role assignments, merged team > system.
func (r *Registry) ListRolesByTeam(teamID string) (map[string]RoleTarget, error) {
	return r.listRoles("", teamID)
}

// ListRolesBy returns all role assignments, scoped by identity (team > user).
func (r *Registry) ListRolesBy(id Identity) (map[string]RoleTarget, error) {
	if id.GetTeamID() != "" {
		return r.ListRolesByTeam(id.GetTeamID())
	}
	return r.ListRolesByUser(id.GetUserID())
}

// ---------------------------------------------------------------------------
// internal
// ---------------------------------------------------------------------------

func (r *Registry) resolveRole(role, userID, teamID string) (string, error) {
	if setting.Global == nil {
		return "", fmt.Errorf("setting registry not initialized")
	}

	merged, err := setting.Global.GetMerged(userID, teamID, RolesNamespace)
	if err != nil {
		return "", fmt.Errorf("role %q not configured: %w", role, err)
	}

	target, ok := merged[role]
	if !ok {
		return "", fmt.Errorf("role %q not configured", role)
	}

	cid := r.extractConnectorID(target)
	if cid == "" {
		return "", fmt.Errorf("role %q has invalid target", role)
	}
	return cid, nil
}

func (r *Registry) listRoles(userID, teamID string) (map[string]RoleTarget, error) {
	if setting.Global == nil {
		return nil, fmt.Errorf("setting registry not initialized")
	}

	merged, err := setting.Global.GetMerged(userID, teamID, RolesNamespace)
	if err != nil {
		return nil, fmt.Errorf("failed to load roles: %w", err)
	}

	result := make(map[string]RoleTarget)
	for role, target := range merged {
		rt := parseRoleTarget(target)
		if rt.Provider != "" {
			result[role] = rt
		}
	}
	return result, nil
}

func (r *Registry) extractConnectorID(target interface{}) string {
	rt := parseRoleTarget(target)
	if rt.Provider == "" {
		return ""
	}

	p, err := r.Get(rt.Provider, true)
	if err != nil {
		if rt.Model != "" {
			return rt.Provider + ":" + rt.Model
		}
		return rt.Provider
	}

	// Builtin providers have model baked into the connector itself;
	// appending :model would create a non-existent composite ID.
	if p.Source == ProviderSourceBuiltIn {
		return p.ConnectorID
	}

	if rt.Model != "" {
		return p.ConnectorID + ":" + rt.Model
	}
	return p.ConnectorID
}

func parseRoleTarget(v interface{}) RoleTarget {
	switch t := v.(type) {
	case map[string]interface{}:
		provider, _ := t["provider"].(string)
		model, _ := t["model"].(string)
		return RoleTarget{Provider: provider, Model: model}
	case RoleTarget:
		return t
	default:
		return RoleTarget{}
	}
}
