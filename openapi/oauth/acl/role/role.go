package role

import (
	"context"
	"fmt"

	"github.com/yaoapp/gou/store"
	"github.com/yaoapp/yao/openapi/oauth/types"
)

// RoleManager is the global role manager
var RoleManager *Manager = nil

// NewManager creates a new role manager
func NewManager(cache store.Store, provider types.UserProvider) *Manager {
	return &Manager{
		cache:    cache,
		provider: provider,
	}
}

// GetClientRole gets the role for a client
func (m *Manager) GetClientRole(ctx context.Context, clientID string) (string, error) {

	// Get From Cache
	roleID, has := m.getClientRoleCache(clientID)
	if has {
		return roleID, nil
	}

	// Get From Database
	role, err := m.getClientRole(ctx, clientID)
	if err != nil {
		return "", err
	}

	// Set Cache
	err = m.setClientRoleCache(clientID, role)
	if err != nil {
		return "", err
	}

	return role, nil
}

// GetUserRole gets the role for a user
func (m *Manager) GetUserRole(ctx context.Context, userID string) (string, error) {
	// Get From Cache
	roleID, has := m.getUserRoleCache(userID)
	if has {
		return roleID, nil
	}

	// Get From Database using UserProvider
	role, err := m.getUserRole(ctx, userID)
	if err != nil {
		return "", err
	}

	// Set Cache
	err = m.setUserRoleCache(userID, role)
	if err != nil {
		return "", err
	}

	return role, nil
}

// GetTeamRole gets the role for a team
func (m *Manager) GetTeamRole(ctx context.Context, teamID string) (string, error) {
	// Get From Cache
	roleID, has := m.getTeamRoleCache(teamID)
	if has {
		return roleID, nil
	}

	// Get From Database using UserProvider
	role, err := m.getTeamRole(ctx, teamID)
	if err != nil {
		return "", err
	}

	// Set Cache
	err = m.setTeamRoleCache(teamID, role)
	if err != nil {
		return "", err
	}

	return role, nil
}

// GetMemberRole gets the role for a member
func (m *Manager) GetMemberRole(ctx context.Context, teamID, userID string) (string, error) {
	// Get From Cache
	roleID, has := m.getMemberRoleCache(teamID, userID)
	if has {
		return roleID, nil
	}

	// Get From Database using UserProvider
	role, err := m.getMemberRole(ctx, teamID, userID)
	if err != nil {
		return "", err
	}

	// Set Cache
	err = m.setMemberRoleCache(teamID, userID, role)
	if err != nil {
		return "", err
	}

	return role, nil
}

// ============================================================================
// Scope Resource
// ============================================================================

// GetScopes gets the scopes for a role
// Returns: (allowedScopes, restrictedScopes, error)
func (m *Manager) GetScopes(ctx context.Context, roleID string) ([]string, []string, error) {
	// Get From Cache
	allowed, restricted, has := m.getScopesCache(roleID)
	if has {
		return allowed, restricted, nil
	}

	// Get From Database using UserProvider
	allowedScopes, restrictedScopes, err := m.getScopes(ctx, roleID)
	if err != nil {
		return nil, nil, err
	}

	// Set Cache
	err = m.setScopesCache(roleID, allowedScopes, restrictedScopes)
	if err != nil {
		return nil, nil, err
	}

	return allowedScopes, restrictedScopes, nil
}

// ============================================================================
// Role Resource - Private Methods
// ============================================================================

// getClientRole gets the role for a client from database
func (m *Manager) getClientRole(ctx context.Context, clientID string) (string, error) {
	// TODO: Implement client role retrieval from ClientProvider
	// For now, return a default role
	return "system:root", nil
}

// getUserRole gets the role for a user from database
func (m *Manager) getUserRole(ctx context.Context, userID string) (string, error) {
	if m.provider == nil {
		return "", fmt.Errorf("user provider is not configured")
	}

	// Get user role information
	roleInfo, err := m.provider.GetUserRole(ctx, userID)
	if err != nil {
		return "", fmt.Errorf("failed to get user role: %w", err)
	}

	// Extract role_id from the role information
	roleID, ok := roleInfo["role_id"].(string)
	if !ok || roleID == "" {
		return "", fmt.Errorf("user %s has no role_id assigned", userID)
	}

	return roleID, nil
}

// getTeamRole gets the role for a team from database
func (m *Manager) getTeamRole(ctx context.Context, teamID string) (string, error) {
	if m.provider == nil {
		return "", fmt.Errorf("user provider is not configured")
	}

	// Get team information
	teamInfo, err := m.provider.GetTeam(ctx, teamID)
	if err != nil {
		return "", fmt.Errorf("failed to get team: %w", err)
	}

	// Extract role_id from the team information
	// Note: Teams might not have a role_id field, adjust based on your schema
	roleID, ok := teamInfo["role_id"].(string)
	if !ok || roleID == "" {
		return "", fmt.Errorf("team %s has no role_id assigned", teamID)
	}

	return roleID, nil
}

// getMemberRole gets the role for a team member from database
func (m *Manager) getMemberRole(ctx context.Context, teamID, userID string) (string, error) {
	if m.provider == nil {
		return "", fmt.Errorf("user provider is not configured")
	}

	// Get member information
	memberInfo, err := m.provider.GetMember(ctx, teamID, userID)
	if err != nil {
		return "", fmt.Errorf("failed to get member: %w", err)
	}

	// Extract role_id from the member information
	roleID, ok := memberInfo["role_id"].(string)
	if !ok || roleID == "" {
		return "", fmt.Errorf("member %s in team %s has no role_id assigned", userID, teamID)
	}

	return roleID, nil
}

// getScopes gets the scopes for a role from database
// Returns: (allowedScopes, restrictedScopes, error)
func (m *Manager) getScopes(ctx context.Context, roleID string) ([]string, []string, error) {
	if m.provider == nil {
		return nil, nil, fmt.Errorf("user provider is not configured")
	}

	// Get role permissions which should contain scopes
	permissionsData, err := m.provider.GetRolePermissions(ctx, roleID)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get role permissions: %w", err)
	}

	// Extract allowed scopes (positive permissions)
	allowedScopes := []string{}
	if permissionsInterface, ok := permissionsData["permissions"]; ok {
		allowed, err := formatPermissions(permissionsInterface)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to format permissions: %w", err)
		}
		allowedScopes = allowed
	}

	// Extract restricted scopes (negative permissions)
	restrictedScopes := []string{}
	if restrictedInterface, ok := permissionsData["restricted_permissions"]; ok {
		restricted, err := formatPermissions(restrictedInterface)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to format restricted_permissions: %w", err)
		}
		restrictedScopes = restricted
	}

	return allowedScopes, restrictedScopes, nil
}
