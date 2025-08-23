package user

import (
	"context"
	"fmt"

	"github.com/yaoapp/gou/model"
	"github.com/yaoapp/kun/maps"
)

// Type Resource

// GetType retrieves type information by type_id
func (u *DefaultUser) GetType(ctx context.Context, typeID string) (maps.MapStrAny, error) {
	m := model.Select(u.typeModel)
	types, err := m.Get(model.QueryParam{
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

// TypeExists checks if a type exists by type_id (lightweight query)
func (u *DefaultUser) TypeExists(ctx context.Context, typeID string) (bool, error) {
	m := model.Select(u.typeModel)
	types, err := m.Get(model.QueryParam{
		Select: []interface{}{"id"}, // Only select ID for existence check
		Wheres: []model.QueryWhere{
			{Column: "type_id", Value: typeID},
		},
		Limit: 1, // Only need to know if at least one exists
	})

	if err != nil {
		return false, fmt.Errorf(ErrFailedToGetType, err)
	}

	return len(types) > 0, nil
}

// CreateType creates a new user type
func (u *DefaultUser) CreateType(ctx context.Context, typeData maps.MapStrAny) (string, error) {
	// Validate required type_id field
	if _, exists := typeData["type_id"]; !exists {
		return "", fmt.Errorf("type_id is required in typeData")
	}

	// Set default values if not provided
	if _, exists := typeData["is_active"]; !exists {
		typeData["is_active"] = true
	}
	if _, exists := typeData["is_default"]; !exists {
		typeData["is_default"] = false
	}
	if _, exists := typeData["sort_order"]; !exists {
		typeData["sort_order"] = 0
	}
	if _, exists := typeData["max_sessions"]; !exists {
		typeData["max_sessions"] = nil // Allow unlimited sessions by default
	}
	if _, exists := typeData["session_timeout"]; !exists {
		typeData["session_timeout"] = 0 // No timeout by default
	}

	m := model.Select(u.typeModel)
	id, err := m.Create(typeData)
	if err != nil {
		return "", fmt.Errorf(ErrFailedToCreateType, err)
	}

	// Return the type_id as string (preferred approach)
	if typeID, ok := typeData["type_id"].(string); ok {
		return typeID, nil
	}

	// Fallback: convert the returned int id to string
	return fmt.Sprintf("%d", id), nil
}

// UpdateType updates an existing type
func (u *DefaultUser) UpdateType(ctx context.Context, typeID string, typeData maps.MapStrAny) error {
	// Remove sensitive fields that should not be updated directly
	sensitiveFields := []string{"id", "type_id", "created_at"}
	for _, field := range sensitiveFields {
		delete(typeData, field)
	}

	// Skip update if no valid fields remain
	if len(typeData) == 0 {
		return nil
	}

	m := model.Select(u.typeModel)
	affected, err := m.UpdateWhere(model.QueryParam{
		Wheres: []model.QueryWhere{
			{Column: "type_id", Value: typeID},
		},
		Limit: 1, // Safety: ensure only one record is updated
	}, typeData)

	if err != nil {
		return fmt.Errorf(ErrFailedToUpdateType, err)
	}

	if affected == 0 {
		return fmt.Errorf(ErrTypeNotFound)
	}

	return nil
}

// DeleteType soft deletes a type
func (u *DefaultUser) DeleteType(ctx context.Context, typeID string) error {
	// First check if type exists
	m := model.Select(u.typeModel)
	types, err := m.Get(model.QueryParam{
		Select: []interface{}{"id", "type_id"},
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

	// Proceed with soft delete
	affected, err := m.DeleteWhere(model.QueryParam{
		Wheres: []model.QueryWhere{
			{Column: "type_id", Value: typeID},
		},
		Limit: 1, // Safety: ensure only one record is deleted
	})

	if err != nil {
		return fmt.Errorf(ErrFailedToDeleteType, err)
	}

	if affected == 0 {
		return fmt.Errorf(ErrTypeNotFound)
	}

	return nil
}

// GetTypes retrieves types by query parameters
func (u *DefaultUser) GetTypes(ctx context.Context, param model.QueryParam) ([]maps.MapStr, error) {
	// Set default select fields if not provided
	if param.Select == nil {
		param.Select = u.typeFields
	}

	m := model.Select(u.typeModel)
	types, err := m.Get(param)
	if err != nil {
		return nil, fmt.Errorf(ErrFailedToGetType, err)
	}

	return types, nil
}

// PaginateTypes retrieves paginated list of types
func (u *DefaultUser) PaginateTypes(ctx context.Context, param model.QueryParam, page int, pagesize int) (maps.MapStr, error) {
	// Set default select fields if not provided
	if param.Select == nil {
		param.Select = u.typeFields
	}

	m := model.Select(u.typeModel)
	result, err := m.Paginate(param, page, pagesize)
	if err != nil {
		return nil, fmt.Errorf(ErrFailedToGetType, err)
	}

	return result, nil
}

// CountTypes returns total count of types with optional filters
func (u *DefaultUser) CountTypes(ctx context.Context, param model.QueryParam) (int64, error) {
	// Use Paginate with a small page size to get the total count
	// This is more reliable than manual COUNT(*) queries
	m := model.Select(u.typeModel)
	result, err := m.Paginate(param, 1, 1) // Get first page with 1 item to get total
	if err != nil {
		return 0, fmt.Errorf(ErrFailedToGetType, err)
	}

	// Extract total from pagination result
	if total, ok := result["total"].(int64); ok {
		return total, nil
	}

	// Handle different total types returned by Paginate
	if totalInterface, ok := result["total"]; ok {
		switch v := totalInterface.(type) {
		case int:
			return int64(v), nil
		case int32:
			return int64(v), nil
		case int64:
			return v, nil
		case uint:
			return int64(v), nil
		case uint32:
			return int64(v), nil
		case uint64:
			return int64(v), nil
		default:
			return 0, fmt.Errorf("unexpected total type: %T", totalInterface)
		}
	}

	return 0, fmt.Errorf("total not found in pagination result")
}

// GetTypeConfiguration retrieves configuration for a type (schema, features, limits, etc.)
func (u *DefaultUser) GetTypeConfiguration(ctx context.Context, typeID string) (maps.MapStrAny, error) {
	m := model.Select(u.typeModel)
	types, err := m.Get(model.QueryParam{
		Select: []interface{}{"type_id", "schema", "features", "limits", "password_policy", "metadata"},
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

	typeRecord := types[0]
	config := maps.MapStrAny{
		"type_id":         typeID,
		"schema":          typeRecord["schema"],
		"features":        typeRecord["features"],
		"limits":          typeRecord["limits"],
		"password_policy": typeRecord["password_policy"],
		"metadata":        typeRecord["metadata"],
	}

	return config, nil
}

// SetTypeConfiguration sets configuration for a type
func (u *DefaultUser) SetTypeConfiguration(ctx context.Context, typeID string, config maps.MapStrAny) error {
	// Prepare update data - only allow configuration-related fields
	updateData := maps.MapStrAny{}

	if schema, ok := config["schema"]; ok {
		updateData["schema"] = schema
	}

	if features, ok := config["features"]; ok {
		updateData["features"] = features
	}

	if limits, ok := config["limits"]; ok {
		updateData["limits"] = limits
	}

	if passwordPolicy, ok := config["password_policy"]; ok {
		updateData["password_policy"] = passwordPolicy
	}

	if metadata, ok := config["metadata"]; ok {
		updateData["metadata"] = metadata
	}

	// Skip update if no configuration fields provided
	if len(updateData) == 0 {
		return nil
	}

	m := model.Select(u.typeModel)
	affected, err := m.UpdateWhere(model.QueryParam{
		Wheres: []model.QueryWhere{
			{Column: "type_id", Value: typeID},
		},
		Limit: 1, // Safety: ensure only one record is updated
	}, updateData)

	if err != nil {
		return fmt.Errorf(ErrFailedToUpdateType, err)
	}

	if affected == 0 {
		return fmt.Errorf(ErrTypeNotFound)
	}

	return nil
}
