package user

import (
	"context"

	"github.com/yaoapp/gou/model"
	"github.com/yaoapp/kun/maps"
)

// User List and Search

// GetUsers retrieves users by query parameters (compatible with Model.Get)
func (u *DefaultUser) GetUsers(ctx context.Context, param model.QueryParam) ([]maps.MapStr, error) {
	// TODO: implement
	return nil, nil
}

// PaginateUsers retrieves paginated list of users (compatible with Model.Paginate)
func (u *DefaultUser) PaginateUsers(ctx context.Context, param model.QueryParam, page int, pagesize int) (maps.MapStr, error) {
	// TODO: implement
	return nil, nil
}

// CountUsers returns total count of users with optional filters
func (u *DefaultUser) CountUsers(ctx context.Context, param model.QueryParam) (int64, error) {
	// TODO: implement
	return 0, nil
}
