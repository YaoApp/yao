package oauth

import (
	"context"

	"github.com/yaoapp/yao/openapi/oauth/types"
)

// Register registers a new OAuth client with the authorization server
func (s *Service) Register(ctx context.Context, clientInfo *types.ClientInfo) (*types.ClientInfo, error) {
	return s.clientProvider.CreateClient(ctx, clientInfo)
}

// UpdateClient updates an existing OAuth client configuration
func (s *Service) UpdateClient(ctx context.Context, clientID string, clientInfo *types.ClientInfo) (*types.ClientInfo, error) {
	return s.clientProvider.UpdateClient(ctx, clientID, clientInfo)
}

// DeleteClient removes an OAuth client from the authorization server
func (s *Service) DeleteClient(ctx context.Context, clientID string) error {
	return s.clientProvider.DeleteClient(ctx, clientID)
}

// ValidateScope validates requested scopes against available scopes
func (s *Service) ValidateScope(ctx context.Context, requestedScopes []string, clientID string) (*types.ValidationResult, error) {
	return s.clientProvider.ValidateScope(ctx, clientID, requestedScopes)
}

// DynamicClientRegistration handles dynamic client registration
// This implements RFC 7591 for automatic client registration
func (s *Service) DynamicClientRegistration(ctx context.Context, request *types.DynamicClientRegistrationRequest) (*types.DynamicClientRegistrationResponse, error) {
	// TODO: Implement dynamic client registration
	return nil, nil
}
