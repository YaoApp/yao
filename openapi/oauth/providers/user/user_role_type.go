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
		return fmt.Errorf(ErrUserNotFound)
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

// GetUserType retrieves user's type information
func (u *DefaultUser) GetUserType(ctx context.Context, userID string) (maps.MapStrAny, error) {
	// TODO: implement
	return nil, nil
}

// SetUserType assigns a type to a user
func (u *DefaultUser) SetUserType(ctx context.Context, userID string, typeID string) error {
	// TODO: implement
	return nil
}

// ValidateUserScope validates if a user has access to requested scopes based on role and type
func (u *DefaultUser) ValidateUserScope(ctx context.Context, userID string, scopes []string) (bool, error) {
	// TODO: implement
	return false, nil
}
