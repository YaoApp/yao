package llm

import (
	"strings"

	yaollm "github.com/yaoapp/yao/llm"
	"github.com/yaoapp/yao/llmprovider"
)

// RolePrefix marks a Connector field value as a role reference (e.g. "use::light").
const RolePrefix = "use::"

// ResolveProviderModel resolves a connector/role reference to a Provider+Model
// pair from the new yao/llm DB. Returns nil,nil when the new registry has
// no matching data (callers should then try ResolveConnector).
//
// connectorID semantics match ResolveConnector: "use::role", explicit ID, or empty.
func ResolveProviderModel(connectorID string, identity llmprovider.Identity) (*yaollm.Provider, *yaollm.Model) {
	if yaollm.Global == nil {
		return nil, nil
	}

	role := ""
	if strings.HasPrefix(connectorID, RolePrefix) {
		role = strings.TrimPrefix(connectorID, RolePrefix)
	} else if connectorID == "" {
		role = "default"
	}

	// Explicit connector ID: try to find matching provider+model
	if role == "" && connectorID != "" {
		return resolveByConnectorID(connectorID)
	}

	teamID, userID := extractIDs(identity)

	// Try the specified role first, then fall back to "default"
	if role != "default" {
		if p, m, err := yaollm.Global.ResolveRoleModel(role, teamID, userID); err == nil {
			return p, m
		}
	}
	if p, m, err := yaollm.Global.ResolveRoleModel("default", teamID, userID); err == nil {
		return p, m
	}
	return nil, nil
}

func extractIDs(identity llmprovider.Identity) (string, string) {
	if identity == nil {
		return "", ""
	}
	return identity.GetTeamID(), identity.GetUserID()
}
