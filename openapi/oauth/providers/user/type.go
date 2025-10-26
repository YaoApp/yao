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
		// Check if type exists
		exists, checkErr := u.TypeExists(ctx, typeID)
		if checkErr != nil {
			return fmt.Errorf(ErrFailedToUpdateType, checkErr)
		}
		if !exists {
			return fmt.Errorf(ErrTypeNotFound)
		}
		// Type exists but no changes were made
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

	// Extract total from pagination result using utility function
	if totalInterface, ok := result["total"]; ok {
		return parseIntFromDB(totalInterface)
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
		// Check if type exists
		exists, checkErr := u.TypeExists(ctx, typeID)
		if checkErr != nil {
			return fmt.Errorf(ErrFailedToUpdateType, checkErr)
		}
		if !exists {
			return fmt.Errorf(ErrTypeNotFound)
		}
		// Type exists but no changes were made
	}

	return nil
}

// GetTypePricing retrieves pricing information for a type
func (u *DefaultUser) GetTypePricing(ctx context.Context, typeID string) (maps.MapStrAny, error) {
	m := model.Select(u.typeModel)

	types, err := m.Get(model.QueryParam{
		Select: []interface{}{
			"type_id", "name", "price_daily", "price_monthly", "price_yearly",
			"credits_monthly", "introduction", "sale_type", "sale_link",
			"sale_price_label", "sale_description", "status", "locale",
		},
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

// GetPublishedTypes retrieves all published types with pricing information, optionally filtered by locale
func (u *DefaultUser) GetPublishedTypes(ctx context.Context, param model.QueryParam, locale ...string) ([]maps.MapStr, error) {
	// Add published status filter
	param.Wheres = append(param.Wheres, model.QueryWhere{
		Column: "status",
		Value:  "published",
	})

	// Add active filter
	param.Wheres = append(param.Wheres, model.QueryWhere{
		Column: "is_active",
		Value:  true,
	})

	// Add locale filter if provided
	if len(locale) > 0 && locale[0] != "" {
		param.Wheres = append(param.Wheres, model.QueryWhere{
			Column: "locale",
			Value:  locale[0],
		})
	}

	// Set default select fields if not provided
	if param.Select == nil {
		param.Select = []interface{}{
			"type_id", "name", "description", "price_daily", "price_monthly", "price_yearly",
			"credits_monthly", "introduction", "sale_type", "sale_link",
			"sale_price_label", "sale_description", "sort_order", "status", "locale", "is_active", "features", "limits",
		}
	}

	// Default ordering by sort_order
	if param.Orders == nil {
		param.Orders = []model.QueryOrder{
			{Column: "sort_order", Option: "asc"},
		}
	}

	m := model.Select(u.typeModel)
	types, err := m.Get(param)
	if err != nil {
		return nil, fmt.Errorf(ErrFailedToGetType, err)
	}

	return types, nil
}

// SetTypePricing updates pricing information for a type
func (u *DefaultUser) SetTypePricing(ctx context.Context, typeID string, pricing maps.MapStrAny) error {
	// Prepare update data - only allow pricing-related fields
	updateData := maps.MapStrAny{}

	if priceDaily, ok := pricing["price_daily"]; ok {
		updateData["price_daily"] = priceDaily
	}

	if priceMonthly, ok := pricing["price_monthly"]; ok {
		updateData["price_monthly"] = priceMonthly
	}

	if priceYearly, ok := pricing["price_yearly"]; ok {
		updateData["price_yearly"] = priceYearly
	}

	if creditsMonthly, ok := pricing["credits_monthly"]; ok {
		updateData["credits_monthly"] = creditsMonthly
	}

	if introduction, ok := pricing["introduction"]; ok {
		updateData["introduction"] = introduction
	}

	if saleType, ok := pricing["sale_type"]; ok {
		updateData["sale_type"] = saleType
	}

	if saleLink, ok := pricing["sale_link"]; ok {
		updateData["sale_link"] = saleLink
	}

	if salePriceLabel, ok := pricing["sale_price_label"]; ok {
		updateData["sale_price_label"] = salePriceLabel
	}

	if saleDescription, ok := pricing["sale_description"]; ok {
		updateData["sale_description"] = saleDescription
	}

	// Skip update if no pricing fields provided
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
		// Check if type exists
		exists, checkErr := u.TypeExists(ctx, typeID)
		if checkErr != nil {
			return fmt.Errorf(ErrFailedToUpdateType, checkErr)
		}
		if !exists {
			return fmt.Errorf(ErrTypeNotFound)
		}
		// Type exists but no changes were made
	}

	return nil
}

// UpdateTypeStatus updates the status of a type (draft/published/archived)
func (u *DefaultUser) UpdateTypeStatus(ctx context.Context, typeID string, status string) error {
	// Validate status
	validStatuses := map[string]bool{
		"draft":     true,
		"published": true,
		"archived":  true,
	}

	if !validStatuses[status] {
		return fmt.Errorf("invalid status: %s, must be one of: draft, published, archived", status)
	}

	m := model.Select(u.typeModel)
	affected, err := m.UpdateWhere(model.QueryParam{
		Wheres: []model.QueryWhere{
			{Column: "type_id", Value: typeID},
		},
		Limit: 1,
	}, maps.MapStrAny{
		"status": status,
	})

	if err != nil {
		return fmt.Errorf(ErrFailedToUpdateType, err)
	}

	if affected == 0 {
		// Check if type exists
		exists, checkErr := u.TypeExists(ctx, typeID)
		if checkErr != nil {
			return fmt.Errorf(ErrFailedToUpdateType, checkErr)
		}
		if !exists {
			return fmt.Errorf(ErrTypeNotFound)
		}
		// Type exists but no changes were made (already has this status)
	}

	return nil
}
