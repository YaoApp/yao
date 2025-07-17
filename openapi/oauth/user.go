package oauth

import (
	"context"
)

// UserInfo returns user information for a given access token
func (s *Service) UserInfo(ctx context.Context, accessToken string) (interface{}, error) {
	return s.userProvider.GetUserByAccessToken(ctx, accessToken)
}

// Additional user-related helper methods can be added here as needed
// For example:
// - User profile management
// - User consent handling
// - User authentication verification
// - User scope validation
// etc.
