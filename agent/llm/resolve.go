package llm

import (
	"fmt"
	"strings"

	"github.com/yaoapp/gou/connector"
	goullm "github.com/yaoapp/gou/llm"
	"github.com/yaoapp/yao/llmprovider"
)

// RolePrefix marks a Connector field value as a role reference (e.g. "use::light").
const RolePrefix = "use::"

// ResolveConnector resolves an LLM connector using a unified priority chain.
//
// connectorID may be:
//   - explicit connector ID (e.g. "openai.gpt-4o") — resolved directly
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
