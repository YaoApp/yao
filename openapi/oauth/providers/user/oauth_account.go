package user

import (
	"context"
	"fmt"
	"time"

	"github.com/yaoapp/gou/model"
	"github.com/yaoapp/kun/maps"
)

// OAuth Account Resource

// CreateOAuthAccount creates a new OAuth account association
func (u *DefaultUser) CreateOAuthAccount(ctx context.Context, userID string, oauthData maps.MapStrAny) (interface{}, error) {
	// Set required fields
	oauthData["user_id"] = userID

	// Set default status if not provided
	if _, exists := oauthData["is_active"]; !exists {
		oauthData["is_active"] = true
	}

	// Set last login time if not provided
	if _, exists := oauthData["last_login_at"]; !exists {
		oauthData["last_login_at"] = time.Now()
	}

	m := model.Select(u.oauthAccountModel)
	id, err := m.Create(oauthData)
	if err != nil {
		return nil, fmt.Errorf(ErrFailedToCreateOAuth, err)
	}

	return id, nil
}

// GetOAuthAccount retrieves OAuth account by provider and subject
func (u *DefaultUser) GetOAuthAccount(ctx context.Context, provider string, subject string) (maps.MapStrAny, error) {
	m := model.Select(u.oauthAccountModel)
	accounts, err := m.Get(model.QueryParam{
		Select: u.oauthAccountFields,
		Wheres: []model.QueryWhere{
			{Column: "provider", Value: provider},
			{Column: "sub", Value: subject},
		},
		Limit: 1,
	})

	if err != nil {
		return nil, fmt.Errorf(ErrFailedToGetOAuthAccount, err)
	}

	if len(accounts) == 0 {
		return nil, fmt.Errorf("oauth account not found for provider %s with subject %s", provider, subject)
	}

	return accounts[0], nil
}

// OAuthAccountExists checks if an OAuth account exists by provider and subject (lightweight query)
func (u *DefaultUser) OAuthAccountExists(ctx context.Context, provider string, subject string) (bool, error) {
	m := model.Select(u.oauthAccountModel)
	accounts, err := m.Get(model.QueryParam{
		Select: []interface{}{"id"}, // Only select ID for existence check
		Wheres: []model.QueryWhere{
			{Column: "provider", Value: provider},
			{Column: "sub", Value: subject},
		},
		Limit: 1, // Only need to know if at least one exists
	})

	if err != nil {
		return false, fmt.Errorf(ErrFailedToGetOAuthAccount, err)
	}

	return len(accounts) > 0, nil
}

// GetUserOAuthAccounts retrieves all OAuth accounts for a user
func (u *DefaultUser) GetUserOAuthAccounts(ctx context.Context, userID string) ([]maps.MapStrAny, error) {
	m := model.Select(u.oauthAccountModel)
	accounts, err := m.Get(model.QueryParam{
		Select: u.oauthAccountFields,
		Wheres: []model.QueryWhere{
			{Column: "user_id", Value: userID},
		},
		Orders: []model.QueryOrder{
			{Column: "last_login_at", Option: "desc"},
		},
	})

	if err != nil {
		return nil, fmt.Errorf(ErrFailedToGetOAuthAccount, err)
	}

	return accounts, nil
}

// UpdateOAuthAccount updates OAuth account information
func (u *DefaultUser) UpdateOAuthAccount(ctx context.Context, provider string, subject string, oauthData maps.MapStrAny) error {
	// Remove sensitive fields that should not be updated directly
	sensitiveFields := []string{"id", "user_id", "provider", "sub", "created_at"}
	for _, field := range sensitiveFields {
		delete(oauthData, field)
	}

	// Skip update if no valid fields remain
	if len(oauthData) == 0 {
		return nil
	}

	m := model.Select(u.oauthAccountModel)
	affected, err := m.UpdateWhere(model.QueryParam{
		Wheres: []model.QueryWhere{
			{Column: "provider", Value: provider},
			{Column: "sub", Value: subject},
		},
		Limit: 1, // Safety: ensure only one record is updated
	}, oauthData)

	if err != nil {
		return fmt.Errorf(ErrFailedToUpdateOAuth, err)
	}

	if affected == 0 {
		return fmt.Errorf("oauth account not found for provider %s with subject %s", provider, subject)
	}

	return nil
}

// DeleteOAuthAccount removes an OAuth account association
func (u *DefaultUser) DeleteOAuthAccount(ctx context.Context, provider string, subject string) error {
	m := model.Select(u.oauthAccountModel)
	affected, err := m.DeleteWhere(model.QueryParam{
		Wheres: []model.QueryWhere{
			{Column: "provider", Value: provider},
			{Column: "sub", Value: subject},
		},
		Limit: 1, // Safety: ensure only one record is deleted
	})

	if err != nil {
		return fmt.Errorf(ErrFailedToDeleteOAuth, err)
	}

	if affected == 0 {
		return fmt.Errorf("oauth account not found for provider %s with subject %s", provider, subject)
	}

	return nil
}

// DeleteUserOAuthAccounts removes all OAuth accounts for a specific user
func (u *DefaultUser) DeleteUserOAuthAccounts(ctx context.Context, userID string) error {
	m := model.Select(u.oauthAccountModel)

	// Use batch soft delete (the Gou library bug has been fixed)
	_, err := m.DeleteWhere(model.QueryParam{
		Wheres: []model.QueryWhere{
			{Column: "user_id", Value: userID},
		},
	})

	if err != nil {
		return fmt.Errorf(ErrFailedToDeleteOAuth, err)
	}

	// Note: We don't check affected count here because it's valid for a user to have no OAuth accounts
	// This method is typically called during user deletion as a cleanup operation

	return nil
}

// GetOAuthAccounts retrieves OAuth accounts by query parameters
func (u *DefaultUser) GetOAuthAccounts(ctx context.Context, param model.QueryParam) ([]maps.MapStr, error) {
	// Set default select fields if not provided
	if param.Select == nil {
		param.Select = u.oauthAccountFields
	}

	m := model.Select(u.oauthAccountModel)
	accounts, err := m.Get(param)
	if err != nil {
		return nil, fmt.Errorf(ErrFailedToGetOAuthAccount, err)
	}

	return accounts, nil
}

// PaginateOAuthAccounts retrieves paginated list of OAuth accounts
func (u *DefaultUser) PaginateOAuthAccounts(ctx context.Context, param model.QueryParam, page int, pagesize int) (maps.MapStr, error) {
	// Set default select fields if not provided
	if param.Select == nil {
		param.Select = u.oauthAccountFields
	}

	m := model.Select(u.oauthAccountModel)
	result, err := m.Paginate(param, page, pagesize)
	if err != nil {
		return nil, fmt.Errorf(ErrFailedToGetOAuthAccount, err)
	}

	return result, nil
}

// CountOAuthAccounts returns total count of OAuth accounts with optional filters
func (u *DefaultUser) CountOAuthAccounts(ctx context.Context, param model.QueryParam) (int64, error) {
	// Use Paginate with a small page size to get the total count
	// This is more reliable than manual COUNT(*) queries
	m := model.Select(u.oauthAccountModel)
	result, err := m.Paginate(param, 1, 1) // Get first page with 1 item to get total
	if err != nil {
		return 0, fmt.Errorf(ErrFailedToGetOAuthAccount, err)
	}

	// Extract total from pagination result using utility function
	if totalInterface, ok := result["total"]; ok {
		return parseIntFromDB(totalInterface)
	}

	return 0, fmt.Errorf("total not found in pagination result")
}
