package types

import (
	"github.com/yaoapp/gou/model"
	"github.com/yaoapp/kun/maps"
)

// CreateAccessScope extracts access scope from the authorized info for creating records
func (info *AuthorizedInfo) CreateAccessScope() *model.AccessScope {
	if info == nil {
		return nil
	}

	scope := &model.AccessScope{}

	if info.UserID != "" {
		scope.CreatedBy = info.UserID
	}

	if info.TeamID != "" {
		scope.TeamID = info.TeamID
	}

	if info.TenantID != "" {
		scope.TenantID = info.TenantID
	}

	return scope
}

// UpdateAccessScope extracts access scope from the authorized info for updating records
func (info *AuthorizedInfo) UpdateAccessScope() *model.AccessScope {
	if info == nil {
		return nil
	}

	scope := &model.AccessScope{}

	if info.UserID != "" {
		scope.UpdatedBy = info.UserID
	}

	if info.TeamID != "" {
		scope.TeamID = info.TeamID
	}

	if info.TenantID != "" {
		scope.TenantID = info.TenantID
	}

	return scope
}

// AccessScope extracts access scope from the authorized info
// Returns an AccessScope with all available fields populated
// Use specific Wheres methods (WheresTeamOnly, WheresCreatorOnly, etc.) to control query logic
func (info *AuthorizedInfo) AccessScope() *model.AccessScope {
	if info == nil {
		return nil
	}

	scope := &model.AccessScope{}

	if info.UserID != "" {
		scope.CreatedBy = info.UserID
		scope.UpdatedBy = info.UserID
	}

	if info.TeamID != "" {
		scope.TeamID = info.TeamID
	}

	if info.TenantID != "" {
		scope.TenantID = info.TenantID
	}

	return scope
}

// WithCreateScope appends CreateAccessScope fields to data for insertion
// Returns map[string]interface{} with access scope fields added
func (info *AuthorizedInfo) WithCreateScope(data interface{}) map[string]interface{} {
	scope := info.CreateAccessScope()
	if scope == nil {
		// If no scope, just convert data to map[string]interface{}
		result := map[string]interface{}{}
		switch v := data.(type) {
		case map[string]interface{}:
			return v
		default:
			// Try to convert using type assertion for common map types
			if m, ok := data.(map[string]interface{}); ok {
				return m
			}
			return result
		}
	}
	return scope.Append(data)
}

// WithUpdateScope appends UpdateAccessScope fields to data for update
// Returns map[string]interface{} with access scope fields added
func (info *AuthorizedInfo) WithUpdateScope(data interface{}) map[string]interface{} {
	scope := info.UpdateAccessScope()
	if scope == nil {
		// If no scope, just convert data to map[string]interface{}
		result := map[string]interface{}{}
		switch v := data.(type) {
		case map[string]interface{}:
			return v
		default:
			// Try to convert using type assertion for common map types
			if m, ok := data.(map[string]interface{}); ok {
				return m
			}
			return result
		}
	}
	return scope.Append(data)
}

// CopyCreateScope copies CreateAccessScope fields from source to dest
// Extracts __yao_created_by, __yao_team_id, __yao_tenant_id from source
// Returns dest map[string]interface{} with access scope fields added
func CopyCreateScope(source, dest interface{}) map[string]interface{} {
	sourceMap := convertToMap(source)
	scope := &model.AccessScope{}

	// Extract fields from source
	if createdBy, ok := sourceMap["__yao_created_by"].(string); ok && createdBy != "" {
		scope.CreatedBy = createdBy
	}
	if teamID, ok := sourceMap["__yao_team_id"].(string); ok && teamID != "" {
		scope.TeamID = teamID
	}
	if tenantID, ok := sourceMap["__yao_tenant_id"].(string); ok && tenantID != "" {
		scope.TenantID = tenantID
	}

	return scope.Append(dest)
}

// CopyUpdateScope copies UpdateAccessScope fields from source to dest
// Extracts __yao_updated_by, __yao_team_id, __yao_tenant_id from source
// Returns dest map[string]interface{} with access scope fields added
func CopyUpdateScope(source, dest interface{}) map[string]interface{} {
	sourceMap := convertToMap(source)
	scope := &model.AccessScope{}

	if updatedBy, ok := sourceMap["__yao_updated_by"].(string); ok && updatedBy != "" {
		scope.UpdatedBy = updatedBy
	}
	if teamID, ok := sourceMap["__yao_team_id"].(string); ok && teamID != "" {
		scope.TeamID = teamID
	}
	if tenantID, ok := sourceMap["__yao_tenant_id"].(string); ok && tenantID != "" {
		scope.TenantID = tenantID
	}

	return scope.Append(dest)
}

// convertToMap converts interface{} to map[string]interface{}
func convertToMap(data interface{}) map[string]interface{} {
	if data == nil {
		return map[string]interface{}{}
	}

	switch v := data.(type) {
	case map[string]interface{}:
		return v
	case maps.MapStrAny:
		return map[string]interface{}(v)
	default:
		return map[string]interface{}{}
	}
}
