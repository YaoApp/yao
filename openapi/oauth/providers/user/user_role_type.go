package user

import (
	"context"
	"fmt"

	"github.com/yaoapp/gou/model"
	"github.com/yaoapp/kun/maps"
)

// User Role and Type Management

// GetUserRole retrieves user's role information
func (u *DefaultUser) GetUserRole(ctx context.Context, userID string) (maps.MapStrAny, error) {
	// First get the user's role_id
	userModel := model.Select(u.model)
	users, err := userModel.Get(model.QueryParam{
		Select: []interface{}{"user_id", "role_id"},
		Wheres: []model.QueryWhere{
			{Column: "user_id", Value: userID},
		},
		Limit: 1,
	})

	if err != nil {
		return nil, fmt.Errorf(ErrFailedToGetUser, err)
	}

	if len(users) == 0 {
		return nil, fmt.Errorf(ErrUserNotFound)
	}

	user := users[0]
	roleID, ok := user["role_id"].(string)
	if !ok || roleID == "" {
		return nil, fmt.Errorf("user %s has no role assigned", userID)
	}

	// Now get the full role information
	roleModel := model.Select(u.roleModel)
	roles, err := roleModel.Get(model.QueryParam{
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

// SetUserRole assigns a role to a user
func (u *DefaultUser) SetUserRole(ctx context.Context, userID string, roleID string) error {
	// First validate that the role exists
	roleModel := model.Select(u.roleModel)
	roles, err := roleModel.Get(model.QueryParam{
		Select: []interface{}{"role_id", "is_active"},
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

	// Check if role is active
	role := roles[0]
	if isActive, ok := role["is_active"].(bool); ok && !isActive {
		return fmt.Errorf("cannot assign inactive role: %s", roleID)
	}
	// Handle different boolean types from database
	if isActiveInt, ok := role["is_active"].(int64); ok && isActiveInt == 0 {
		return fmt.Errorf("cannot assign inactive role: %s", roleID)
	}

	// Update user's role_id
	updateData := maps.MapStrAny{
		"role_id": roleID,
	}

	userModel := model.Select(u.model)
	affected, err := userModel.UpdateWhere(model.QueryParam{
		Wheres: []model.QueryWhere{
			{Column: "user_id", Value: userID},
		},
		Limit: 1, // Safety: ensure only one record is updated
	}, updateData)

	if err != nil {
		return fmt.Errorf(ErrFailedToUpdateUser, err)
	}

	if affected == 0 {
		// Check if user exists
		exists, checkErr := u.UserExists(ctx, userID)
		if checkErr != nil {
			return fmt.Errorf(ErrFailedToUpdateUser, checkErr)
		}
		if !exists {
			return fmt.Errorf(ErrUserNotFound)
		}
		// User exists but no changes were made (already has this role)
	}

	return nil
}

// ClearUserRole removes role assignment from a user (sets role_id to null)
func (u *DefaultUser) ClearUserRole(ctx context.Context, userID string) error {
	// First check if user exists
	userModel := model.Select(u.model)
	users, err := userModel.Get(model.QueryParam{
		Select: []interface{}{"user_id"},
		Wheres: []model.QueryWhere{
			{Column: "user_id", Value: userID},
		},
		Limit: 1,
	})

	if err != nil {
		return fmt.Errorf(ErrFailedToGetUser, err)
	}

	if len(users) == 0 {
		return fmt.Errorf(ErrUserNotFound)
	}

	// Update role_id to null (even if it's already null, this should succeed)
	updateData := maps.MapStrAny{
		"role_id": nil, // Set role_id to null to clear role assignment
	}

	_, err = userModel.UpdateWhere(model.QueryParam{
		Wheres: []model.QueryWhere{
			{Column: "user_id", Value: userID},
		},
		Limit: 1, // Safety: ensure only one record is updated
	}, updateData)

	if err != nil {
		return fmt.Errorf(ErrFailedToUpdateUser, err)
	}

	// Don't check affected rows - setting null to null is still a successful operation
	return nil
}

// UserHasRole checks if a user has a role assigned (lightweight query)
func (u *DefaultUser) UserHasRole(ctx context.Context, userID string) (bool, error) {
	userModel := model.Select(u.model)
	users, err := userModel.Get(model.QueryParam{
		Select: []interface{}{"role_id"}, // Only select role_id field
		Wheres: []model.QueryWhere{
			{Column: "user_id", Value: userID},
		},
		Limit: 1,
	})

	if err != nil {
		return false, fmt.Errorf(ErrFailedToGetUser, err)
	}

	if len(users) == 0 {
		return false, fmt.Errorf(ErrUserNotFound)
	}

	user := users[0]
	roleID, ok := user["role_id"].(string)
	return ok && roleID != "", nil
}

// GetUserType retrieves user's type information
func (u *DefaultUser) GetUserType(ctx context.Context, userID string) (maps.MapStrAny, error) {
	// First get the user's type_id
	userModel := model.Select(u.model)
	users, err := userModel.Get(model.QueryParam{
		Select: []interface{}{"user_id", "type_id"},
		Wheres: []model.QueryWhere{
			{Column: "user_id", Value: userID},
		},
		Limit: 1,
	})

	if err != nil {
		return nil, fmt.Errorf(ErrFailedToGetUser, err)
	}

	if len(users) == 0 {
		return nil, fmt.Errorf(ErrUserNotFound)
	}

	user := users[0]
	typeID, ok := user["type_id"].(string)
	if !ok || typeID == "" {
		return nil, fmt.Errorf("user %s has no type assigned", userID)
	}

	// Now get the full type information
	typeModel := model.Select(u.typeModel)
	types, err := typeModel.Get(model.QueryParam{
		Select: u.typeFields,
		Wheres: []model.QueryWhere{
			{Column: "type_id", Value: typeID},
		},
		Limit: 1,
	})

	if err != nil {
		return nil, fmt.Errorf(ErrFailedToGetType, err)
	}

	if len(types) == 0 {
		return nil, fmt.Errorf(ErrTypeNotFound)
	}

	return types[0], nil
}

// SetUserType assigns a type to a user
func (u *DefaultUser) SetUserType(ctx context.Context, userID string, typeID string) error {
	// First validate that the type exists
	typeModel := model.Select(u.typeModel)
	types, err := typeModel.Get(model.QueryParam{
		Select: []interface{}{"type_id", "is_active"},
		Wheres: []model.QueryWhere{
			{Column: "type_id", Value: typeID},
		},
		Limit: 1,
	})

	if err != nil {
		return fmt.Errorf(ErrFailedToGetType, err)
	}

	if len(types) == 0 {
		return fmt.Errorf(ErrTypeNotFound)
	}

	// Check if type is active
	typeRecord := types[0]
	if isActive, ok := typeRecord["is_active"].(bool); ok && !isActive {
		return fmt.Errorf("cannot assign inactive type: %s", typeID)
	}
	// Handle different boolean types from database
	if isActiveInt, ok := typeRecord["is_active"].(int64); ok && isActiveInt == 0 {
		return fmt.Errorf("cannot assign inactive type: %s", typeID)
	}

	// Update user's type_id
	updateData := maps.MapStrAny{
		"type_id": typeID,
	}

	userModel := model.Select(u.model)
	affected, err := userModel.UpdateWhere(model.QueryParam{
		Wheres: []model.QueryWhere{
			{Column: "user_id", Value: userID},
		},
		Limit: 1, // Safety: ensure only one record is updated
	}, updateData)

	if err != nil {
		return fmt.Errorf(ErrFailedToUpdateUser, err)
	}

	if affected == 0 {
		// Check if user exists
		exists, checkErr := u.UserExists(ctx, userID)
		if checkErr != nil {
			return fmt.Errorf(ErrFailedToUpdateUser, checkErr)
		}
		if !exists {
			return fmt.Errorf(ErrUserNotFound)
		}
		// User exists but no changes were made (already has this type)
	}

	return nil
}

// ClearUserType removes type assignment from a user (sets type_id to null)
func (u *DefaultUser) ClearUserType(ctx context.Context, userID string) error {
	// First check if user exists
	userModel := model.Select(u.model)
	users, err := userModel.Get(model.QueryParam{
		Select: []interface{}{"user_id"},
		Wheres: []model.QueryWhere{
			{Column: "user_id", Value: userID},
		},
		Limit: 1,
	})

	if err != nil {
		return fmt.Errorf(ErrFailedToGetUser, err)
	}

	if len(users) == 0 {
		return fmt.Errorf(ErrUserNotFound)
	}

	// Update type_id to null (even if it's already null, this should succeed)
	updateData := maps.MapStrAny{
		"type_id": nil, // Set type_id to null to clear type assignment
	}

	_, err = userModel.UpdateWhere(model.QueryParam{
		Wheres: []model.QueryWhere{
			{Column: "user_id", Value: userID},
		},
		Limit: 1, // Safety: ensure only one record is updated
	}, updateData)

	if err != nil {
		return fmt.Errorf(ErrFailedToUpdateUser, err)
	}

	// Don't check affected rows - setting null to null is still a successful operation
	return nil
}

// UserHasType checks if a user has a type assigned (lightweight query)
func (u *DefaultUser) UserHasType(ctx context.Context, userID string) (bool, error) {
	userModel := model.Select(u.model)
	users, err := userModel.Get(model.QueryParam{
		Select: []interface{}{"type_id"}, // Only select type_id field
		Wheres: []model.QueryWhere{
			{Column: "user_id", Value: userID},
		},
		Limit: 1,
	})

	if err != nil {
		return false, fmt.Errorf(ErrFailedToGetUser, err)
	}

	if len(users) == 0 {
		return false, fmt.Errorf(ErrUserNotFound)
	}

	user := users[0]
	typeID, ok := user["type_id"].(string)
	return ok && typeID != "", nil
}

// ValidateUserScope validates if a user has access to requested scopes based on role and type
func (u *DefaultUser) ValidateUserScope(ctx context.Context, userID string, scopes []string) (bool, error) {
	if len(scopes) == 0 {
		return true, nil // No scopes required
	}

	// Get user's role
	userRole, err := u.GetUserRole(ctx, userID)
	if err != nil {
		// If user has no role, check if scopes are required
		if err.Error() == fmt.Sprintf("user %s has no role assigned", userID) {
			// Users without roles have minimal access (empty scopes only)
			return len(scopes) == 0, nil
		}
		return false, err
	}

	// Extract role_id for permission validation
	roleID, ok := userRole["role_id"].(string)
	if !ok {
		return false, fmt.Errorf("invalid role_id format")
	}

	// Use role-based permission validation
	valid, err := u.ValidateRolePermissions(ctx, roleID, scopes)
	if err != nil {
		return false, err
	}

	// If role validation passes, check type-specific restrictions if applicable
	if valid {
		// Get user's type for additional validation
		userType, err := u.GetUserType(ctx, userID)
		if err != nil {
			// If user has no type, role validation is sufficient
			if err.Error() == fmt.Sprintf("user %s has no type assigned", userID) {
				return valid, nil
			}
			return false, err
		}

		// Get type configuration to check for additional scope restrictions
		typeID, ok := userType["type_id"].(string)
		if !ok {
			return false, fmt.Errorf("invalid type_id format")
		}

		typeConfig, err := u.GetTypeConfiguration(ctx, typeID)
		if err != nil {
			return false, err
		}

		// Check if type has specific scope limitations
		if features, ok := typeConfig["features"].(map[string]interface{}); ok {
			if scopeLimits, exists := features["scope_limits"]; exists {
				if limitList, ok := scopeLimits.([]interface{}); ok {
					// If type has scope limits, ensure all requested scopes are allowed
					allowedScopes := make(map[string]bool)
					for _, scope := range limitList {
						if scopeStr, ok := scope.(string); ok {
							allowedScopes[scopeStr] = true
						}
					}

					// Check each requested scope against type limits
					for _, scope := range scopes {
						if !allowedScopes[scope] {
							return false, nil // Scope not allowed by type
						}
					}
				}
			}
		}
	}

	return valid, nil
}
