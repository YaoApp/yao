package user

import (
	"context"
	"fmt"

	"github.com/yaoapp/gou/model"
	"github.com/yaoapp/kun/maps"
)

// User List and Search

// GetUsers retrieves users by query parameters (compatible with Model.Get)
func (u *DefaultUser) GetUsers(ctx context.Context, param model.QueryParam) ([]maps.MapStr, error) {
	// Set default select fields if not provided
	if param.Select == nil {
		param.Select = u.basicUserFields
	}

	m := model.Select(u.model)
	users, err := m.Get(param)
	if err != nil {
		return nil, fmt.Errorf(ErrFailedToGetUser, err)
	}

	return users, nil
}

// PaginateUsers retrieves paginated list of users (compatible with Model.Paginate)
func (u *DefaultUser) PaginateUsers(ctx context.Context, param model.QueryParam, page int, pagesize int) (maps.MapStr, error) {
	// Set default select fields if not provided
	if param.Select == nil {
		param.Select = u.basicUserFields
	}

	m := model.Select(u.model)
	result, err := m.Paginate(param, page, pagesize)
	if err != nil {
		return nil, fmt.Errorf(ErrFailedToGetUser, err)
	}

	return result, nil
}

// CountUsers returns total count of users with optional filters
func (u *DefaultUser) CountUsers(ctx context.Context, param model.QueryParam) (int64, error) {
	// Use Paginate with a small page size to get the total count
	// This is more reliable than manual COUNT(*) queries
	m := model.Select(u.model)
	result, err := m.Paginate(param, 1, 1) // Get first page with 1 item to get total
	if err != nil {
		return 0, fmt.Errorf(ErrFailedToGetUser, err)
	}

	// Extract total from pagination result using utility function
	if totalInterface, ok := result["total"]; ok {
		return parseIntFromDB(totalInterface)
	}

	return 0, fmt.Errorf("total not found in pagination result")
}
