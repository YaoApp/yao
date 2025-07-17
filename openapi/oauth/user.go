package oauth

import (
	"context"
)

// DefaultUserProvider provides a default implementation of UserProvider
type DefaultUserProvider struct {
	getUserByAccessTokenFunc func(ctx context.Context, accessToken string) (interface{}, error)
	getUserBySubjectFunc     func(ctx context.Context, subject string) (interface{}, error)
	validateUserScopeFunc    func(ctx context.Context, userID string, scopes []string) (bool, error)
}

// NewDefaultUserProvider creates a new DefaultUserProvider with the given functions
func NewDefaultUserProvider(
	getUserByAccessTokenFunc func(ctx context.Context, accessToken string) (interface{}, error),
	getUserBySubjectFunc func(ctx context.Context, subject string) (interface{}, error),
	validateUserScopeFunc func(ctx context.Context, userID string, scopes []string) (bool, error),
) *DefaultUserProvider {
	return &DefaultUserProvider{
		getUserByAccessTokenFunc: getUserByAccessTokenFunc,
		getUserBySubjectFunc:     getUserBySubjectFunc,
		validateUserScopeFunc:    validateUserScopeFunc,
	}
}

// GetUserByAccessToken retrieves user information using an access token
func (p *DefaultUserProvider) GetUserByAccessToken(ctx context.Context, accessToken string) (interface{}, error) {
	if p.getUserByAccessTokenFunc == nil {
		return nil, &ErrorResponse{Code: "not_implemented", ErrorDescription: "GetUserByAccessToken is not implemented"}
	}
	return p.getUserByAccessTokenFunc(ctx, accessToken)
}

// GetUserBySubject retrieves user information using a subject identifier
func (p *DefaultUserProvider) GetUserBySubject(ctx context.Context, subject string) (interface{}, error) {
	if p.getUserBySubjectFunc == nil {
		return nil, &ErrorResponse{Code: "not_implemented", ErrorDescription: "GetUserBySubject is not implemented"}
	}
	return p.getUserBySubjectFunc(ctx, subject)
}

// ValidateUserScope validates if a user has access to requested scopes
func (p *DefaultUserProvider) ValidateUserScope(ctx context.Context, userID string, scopes []string) (bool, error) {
	if p.validateUserScopeFunc == nil {
		// Default implementation: allow all scopes
		return true, nil
	}
	return p.validateUserScopeFunc(ctx, userID, scopes)
}
