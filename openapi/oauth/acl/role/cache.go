package role

import (
	"fmt"
	"time"
)

// PRE prefix for the role cache
const PRE = "acl:role:"

// TTL time for the role cache
const TTL = 1 * time.Hour

// keyUserRole returns the key for the user role cache
func (m *Manager) keyUserRole(userID string) string {
	return fmt.Sprintf("%suser:%s", PRE, userID)
}

// keyClientRole returns the key for the client role cache
func (m *Manager) keyClientRole(clientID string) string {
	return fmt.Sprintf("%sclient:%s", PRE, clientID)
}

// keyTeamRole returns the key for the team role cache
func (m *Manager) keyTeamRole(teamID string) string {
	return fmt.Sprintf("%steam:%s", PRE, teamID)
}

// keyMemberRole returns the key for the member role cache
func (m *Manager) keyMemberRole(teamID, userID string) string {
	return fmt.Sprintf("%smember:%s:%s", PRE, teamID, userID)
}

// keyScopes returns the key for the allowed scopes cache
func (m *Manager) keyScopes(roleID string) string {
	return fmt.Sprintf("%sscopes:%s", PRE, roleID)
}

// keyScopesRestricted returns the key for the restricted scopes cache
func (m *Manager) keyScopesRestricted(roleID string) string {
	return fmt.Sprintf("%sscopes:restricted:%s", PRE, roleID)
}

// ============================================================================
// Cache Get Operations
// ============================================================================

// getUserRoleCache gets the user role from the cache
func (m *Manager) getUserRoleCache(userID string) (string, bool) {
	if m.cache == nil {
		return "", false
	}
	value, has := m.cache.Get(m.keyUserRole(userID))
	if !has {
		return "", false
	}
	return toString(value), true
}

// getClientRoleCache gets the client role from the cache
func (m *Manager) getClientRoleCache(clientID string) (string, bool) {
	if m.cache == nil {
		return "", false
	}
	value, has := m.cache.Get(m.keyClientRole(clientID))
	if !has {
		return "", false
	}
	return toString(value), true
}

// getTeamRoleCache gets the team role from the cache
func (m *Manager) getTeamRoleCache(teamID string) (string, bool) {
	if m.cache == nil {
		return "", false
	}
	value, has := m.cache.Get(m.keyTeamRole(teamID))
	if !has {
		return "", false
	}
	return toString(value), true
}

// getMemberRoleCache gets the member role from the cache
func (m *Manager) getMemberRoleCache(teamID, userID string) (string, bool) {
	if m.cache == nil {
		return "", false
	}
	value, has := m.cache.Get(m.keyMemberRole(teamID, userID))
	if !has {
		return "", false
	}
	return toString(value), true
}

// getScopesCache gets the scopes from the cache
// Returns: (allowedScopes, restrictedScopes, found)
func (m *Manager) getScopesCache(roleID string) ([]string, []string, bool) {
	if m.cache == nil {
		return nil, nil, false
	}

	// Get allowed scopes
	allowedValue, hasAllowed := m.cache.Get(m.keyScopes(roleID))
	if !hasAllowed {
		return nil, nil, false
	}

	// Get restricted scopes
	restrictedValue, hasRestricted := m.cache.Get(m.keyScopesRestricted(roleID))
	// Note: restricted scopes might not exist, which is OK

	allowedScopes := toStringArray(allowedValue)
	restrictedScopes := []string{}
	if hasRestricted {
		restrictedScopes = toStringArray(restrictedValue)
	}

	return allowedScopes, restrictedScopes, true
}

// ============================================================================
// Cache Set Operations
// ============================================================================

// setUserRoleCache sets the user role in the cache
func (m *Manager) setUserRoleCache(userID, roleID string) error {
	if m.cache == nil {
		return nil // Silently skip if cache is not configured
	}
	return m.cache.Set(m.keyUserRole(userID), roleID, TTL)
}

// setClientRoleCache sets the client role in the cache
func (m *Manager) setClientRoleCache(clientID, roleID string) error {
	if m.cache == nil {
		return nil // Silently skip if cache is not configured
	}
	return m.cache.Set(m.keyClientRole(clientID), roleID, TTL)
}

// setTeamRoleCache sets the team role in the cache
func (m *Manager) setTeamRoleCache(teamID, roleID string) error {
	if m.cache == nil {
		return nil // Silently skip if cache is not configured
	}
	return m.cache.Set(m.keyTeamRole(teamID), roleID, TTL)
}

// setMemberRoleCache sets the member role in the cache
func (m *Manager) setMemberRoleCache(teamID, userID, roleID string) error {
	if m.cache == nil {
		return nil // Silently skip if cache is not configured
	}
	return m.cache.Set(m.keyMemberRole(teamID, userID), roleID, TTL)
}

// setScopesCache sets the scopes in the cache
func (m *Manager) setScopesCache(roleID string, allowedScopes []string, restrictedScopes []string) error {
	if m.cache == nil {
		return nil // Silently skip if cache is not configured
	}

	// Set allowed scopes
	err := m.cache.Set(m.keyScopes(roleID), allowedScopes, TTL)
	if err != nil {
		return err
	}

	// Set restricted scopes
	err = m.cache.Set(m.keyScopesRestricted(roleID), restrictedScopes, TTL)
	if err != nil {
		// If setting restricted scopes fails, delete the allowed scopes to maintain consistency
		_ = m.cache.Del(m.keyScopes(roleID))
		return err
	}

	return nil
}

// ============================================================================
// Cache Delete Operations
// ============================================================================

// delUserRoleCache deletes the user role from the cache
func (m *Manager) delUserRoleCache(userID string) error {
	if m.cache == nil {
		return nil // Silently skip if cache is not configured
	}
	return m.cache.Del(m.keyUserRole(userID))
}

// delClientRoleCache deletes the client role from the cache
func (m *Manager) delClientRoleCache(clientID string) error {
	if m.cache == nil {
		return nil // Silently skip if cache is not configured
	}
	return m.cache.Del(m.keyClientRole(clientID))
}

// delTeamRoleCache deletes the team role from the cache
func (m *Manager) delTeamRoleCache(teamID string) error {
	if m.cache == nil {
		return nil // Silently skip if cache is not configured
	}
	return m.cache.Del(m.keyTeamRole(teamID))
}

// delMemberRoleCache deletes the member role from the cache
func (m *Manager) delMemberRoleCache(teamID, userID string) error {
	if m.cache == nil {
		return nil // Silently skip if cache is not configured
	}
	return m.cache.Del(m.keyMemberRole(teamID, userID))
}

// delScopesCache deletes the scopes from the cache
func (m *Manager) delScopesCache(roleID string) error {
	if m.cache == nil {
		return nil // Silently skip if cache is not configured
	}

	// Delete allowed scopes
	err := m.cache.Del(m.keyScopes(roleID))
	if err != nil {
		return err
	}

	// Delete restricted scopes
	err = m.cache.Del(m.keyScopesRestricted(roleID))
	if err != nil {
		return err
	}

	return nil
}

// ClearCache clears the role cache
func (m *Manager) ClearCache() error {
	if m.cache == nil {
		return nil // Silently skip if cache is not configured
	}
	return m.cache.Del(fmt.Sprintf("%s*", PRE))
}
