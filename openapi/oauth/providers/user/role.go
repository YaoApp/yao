package user

import (
	"context"
	"fmt"

	"github.com/yaoapp/gou/model"
	"github.com/yaoapp/kun/maps"
)

// Role Resource

// GetRole retrieves role information by role_id
func (u *DefaultUser) GetRole(ctx context.Context, roleID string) (maps.MapStrAny, error) {
	m := model.Select(u.roleModel)
	roles, err := m.Get(model.QueryParam{
		Select: u.roleFields,
		Wheres: []model.QueryWhere{
			{Column: "role_id", Value: roleID},
		},
		Limit: 1,
	})

	if err != nil {
		return nil, fmt.Errorf(ErrFailedToGetRole, err)
	}

	if len(roles) == 0 {
		return nil, fmt.Errorf(ErrRoleNotFound)
	}

	return roles[0], nil
}

// RoleExists checks if a role exists by role_id (lightweight query)
func (u *DefaultUser) RoleExists(ctx context.Context, roleID string) (bool, error) {
	m := model.Select(u.roleModel)
	roles, err := m.Get(model.QueryParam{
		Select: []interface{}{"id"}, // Only select ID for existence check
		Wheres: []model.QueryWhere{
			{Column: "role_id", Value: roleID},
		},
		Limit: 1, // Only need to know if at least one exists
	})

	if err != nil {
		return false, fmt.Errorf(ErrFailedToGetRole, err)
	}

	return len(roles) > 0, nil
}

// CreateRole creates a new user role
func (u *DefaultUser) CreateRole(ctx context.Context, roleData maps.MapStrAny) (string, error) {
	// Validate required role_id field
	if _, exists := roleData["role_id"]; !exists {
		return "", fmt.Errorf("role_id is required in roleData")
	}

	// Set default values if not provided
	if _, exists := roleData["is_active"]; !exists {
		roleData["is_active"] = true
	}
	if _, exists := roleData["is_default"]; !exists {
		roleData["is_default"] = false
	}
	if _, exists := roleData["is_system"]; !exists {
		roleData["is_system"] = false
	}
	if _, exists := roleData["level"]; !exists {
		roleData["level"] = 0
	}
	if _, exists := roleData["sort_order"]; !exists {
		roleData["sort_order"] = 0
	}

	m := model.Select(u.roleModel)
	id, err := m.Create(roleData)
	if err != nil {
		return "", fmt.Errorf(ErrFailedToCreateRole, err)
	}

	// Return the role_id as string (preferred approach)
	if roleID, ok := roleData["role_id"].(string); ok {
		return roleID, nil
	}

	// Fallback: convert the returned int id to string
	return fmt.Sprintf("%d", id), nil
}

// UpdateRole updates an existing role
func (u *DefaultUser) UpdateRole(ctx context.Context, roleID string, roleData maps.MapStrAny) error {
	// Remove sensitive fields that should not be updated directly
	sensitiveFields := []string{"id", "role_id", "created_at"}
	for _, field := range sensitiveFields {
		delete(roleData, field)
	}

	// Skip update if no valid fields remain
	if len(roleData) == 0 {
		return nil
	}

	m := model.Select(u.roleModel)
	affected, err := m.UpdateWhere(model.QueryParam{
		Wheres: []model.QueryWhere{
			{Column: "role_id", Value: roleID},
		},
		Limit: 1, // Safety: ensure only one record is updated
	}, roleData)

	if err != nil {
		return fmt.Errorf(ErrFailedToUpdateRole, err)
	}

	if affected == 0 {
		// Check if role exists
		exists, checkErr := u.RoleExists(ctx, roleID)
		if checkErr != nil {
			return fmt.Errorf(ErrFailedToUpdateRole, checkErr)
		}
		if !exists {
			return fmt.Errorf(ErrRoleNotFound)
		}
		// Role exists but no changes were made
	}

	return nil
}

// DeleteRole soft deletes a role (if not system role)
func (u *DefaultUser) DeleteRole(ctx context.Context, roleID string) error {
	// First check if role exists and is not a system role
	m := model.Select(u.roleModel)
	roles, err := m.Get(model.QueryParam{
		Select: []interface{}{"id", "role_id", "is_system"},
		Wheres: []model.QueryWhere{
			{Column: "role_id", Value: roleID},
		},
		Limit: 1,
	})

	if err != nil {
		return fmt.Errorf(ErrFailedToGetRole, err)
	}

	if len(roles) == 0 {
		return fmt.Errorf(ErrRoleNotFound)
	}

	role := roles[0]
	// Check if this is a system role
	if isSystem, ok := role["is_system"].(bool); ok && isSystem {
		return fmt.Errorf("cannot delete system role: %s", roleID)
	}
	// Handle different boolean types from database
	if isSystemInt, ok := role["is_system"].(int64); ok && isSystemInt != 0 {
		return fmt.Errorf("cannot delete system role: %s", roleID)
	}

	// Proceed with soft delete
	affected, err := m.DeleteWhere(model.QueryParam{
		Wheres: []model.QueryWhere{
			{Column: "role_id", Value: roleID},
		},
		Limit: 1, // Safety: ensure only one record is deleted
	})

	if err != nil {
		return fmt.Errorf(ErrFailedToDeleteRole, err)
	}

	if affected == 0 {
		return fmt.Errorf(ErrRoleNotFound)
	}

	return nil
}

// GetRoles retrieves roles by query parameters
func (u *DefaultUser) GetRoles(ctx context.Context, param model.QueryParam) ([]maps.MapStr, error) {
	// Set default select fields if not provided
	if param.Select == nil {
		param.Select = u.roleFields
	}

	m := model.Select(u.roleModel)
	roles, err := m.Get(param)
	if err != nil {
		return nil, fmt.Errorf(ErrFailedToGetRole, err)
	}

	return roles, nil
}

// PaginateRoles retrieves paginated list of roles
func (u *DefaultUser) PaginateRoles(ctx context.Context, param model.QueryParam, page int, pagesize int) (maps.MapStr, error) {
	// Set default select fields if not provided
	if param.Select == nil {
		param.Select = u.roleFields
	}

	m := model.Select(u.roleModel)
	result, err := m.Paginate(param, page, pagesize)
	if err != nil {
		return nil, fmt.Errorf(ErrFailedToGetRole, err)
	}

	return result, nil
}

// CountRoles returns total count of roles with optional filters
func (u *DefaultUser) CountRoles(ctx context.Context, param model.QueryParam) (int64, error) {
	// Use Paginate with a small page size to get the total count
	// This is more reliable than manual COUNT(*) queries
	m := model.Select(u.roleModel)
	result, err := m.Paginate(param, 1, 1) // Get first page with 1 item to get total
	if err != nil {
		return 0, fmt.Errorf(ErrFailedToGetRole, err)
	}

	// Extract total from pagination result using utility function
	if totalInterface, ok := result["total"]; ok {
		return parseIntFromDB(totalInterface)
	}

	return 0, fmt.Errorf("total not found in pagination result")
}

// GetRolePermissions retrieves permissions for a role
func (u *DefaultUser) GetRolePermissions(ctx context.Context, roleID string) (maps.MapStrAny, error) {
	m := model.Select(u.roleModel)
	roles, err := m.Get(model.QueryParam{
		Select: []interface{}{"role_id", "permissions", "restricted_permissions"},
		Wheres: []model.QueryWhere{
			{Column: "role_id", Value: roleID},
		},
		Limit: 1,
	})

	if err != nil {
		return nil, fmt.Errorf(ErrFailedToGetRole, err)
	}

	if len(roles) == 0 {
		return nil, fmt.Errorf(ErrRoleNotFound)
	}

	role := roles[0]
	permissions := maps.MapStrAny{
		"role_id":                roleID,
		"permissions":            role["permissions"],
		"restricted_permissions": role["restricted_permissions"],
	}

	return permissions, nil
}

// SetRolePermissions sets permissions for a role
func (u *DefaultUser) SetRolePermissions(ctx context.Context, roleID string, permissions maps.MapStrAny) error {
	// Prepare update data - only allow permission-related fields
	updateData := maps.MapStrAny{}

	if perms, ok := permissions["permissions"]; ok {
		updateData["permissions"] = perms
	}

	if restrictedPerms, ok := permissions["restricted_permissions"]; ok {
		updateData["restricted_permissions"] = restrictedPerms
	}

	// Skip update if no permission fields provided
	if len(updateData) == 0 {
		return nil
	}

	m := model.Select(u.roleModel)
	affected, err := m.UpdateWhere(model.QueryParam{
		Wheres: []model.QueryWhere{
			{Column: "role_id", Value: roleID},
		},
		Limit: 1, // Safety: ensure only one record is updated
	}, updateData)

	if err != nil {
		return fmt.Errorf(ErrFailedToUpdateRole, err)
	}

	if affected == 0 {
		// Check if role exists
		exists, checkErr := u.RoleExists(ctx, roleID)
		if checkErr != nil {
			return fmt.Errorf(ErrFailedToUpdateRole, checkErr)
		}
		if !exists {
			return fmt.Errorf(ErrRoleNotFound)
		}
		// Role exists but no changes were made (same permissions)
	}

	return nil
}

// ValidateRolePermissions validates if role has specific permissions
func (u *DefaultUser) ValidateRolePermissions(ctx context.Context, roleID string, requiredPermissions []string) (bool, error) {
	if len(requiredPermissions) == 0 {
		return true, nil // No permissions required
	}

	// Get role permissions
	rolePermissions, err := u.GetRolePermissions(ctx, roleID)
	if err != nil {
		return false, err
	}

	// Extract permissions and restricted permissions
	permissions, _ := rolePermissions["permissions"].(map[string]interface{})
	restrictedPermissions, _ := rolePermissions["restricted_permissions"].([]interface{})

	// Convert restricted permissions to map for faster lookup
	restrictedMap := make(map[string]bool)
	for _, perm := range restrictedPermissions {
		if permStr, ok := perm.(string); ok {
			restrictedMap[permStr] = true
		}
	}

	// Check each required permission
	for _, requiredPerm := range requiredPermissions {
		// First check if permission is explicitly restricted
		if restrictedMap[requiredPerm] {
			return false, nil // Permission is explicitly denied
		}

		// Check if permission exists in granted permissions
		if permissions == nil {
			return false, nil // No permissions granted
		}

		// Look for the permission in the permissions object
		// This is a simple implementation - in practice, you might want more sophisticated permission matching
		permValue, exists := permissions[requiredPerm]
		if !exists {
			return false, nil // Permission not found
		}

		// Check if permission is enabled (assuming boolean values)
		if permBool, ok := permValue.(bool); ok && !permBool {
			return false, nil // Permission exists but is disabled
		}
	}

	return true, nil // All required permissions are valid
}
