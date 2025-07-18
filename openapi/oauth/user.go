package oauth

import (
	"context"
)

// UserInfo returns user information for a given access token
func (s *Service) UserInfo(ctx context.Context, accessToken string) (interface{}, error) {
	return s.userProvider.GetUserByAccessToken(ctx, accessToken)
}
