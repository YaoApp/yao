package tai

import grpcclient "github.com/yaoapp/yao/grpc/client"

// TokenManager is an alias for grpc/client.TokenManager.
// New code should use grpc/client.TokenManager directly.
type TokenManager = grpcclient.TokenManager

// NewTokenManagerFromEnv creates a TokenManager from environment variables.
func NewTokenManagerFromEnv() (*TokenManager, error) {
	return grpcclient.NewTokenManagerFromEnv()
}

// NewTokenManager creates a TokenManager with explicit values.
func NewTokenManager(accessToken, refreshToken, sandboxID string) *TokenManager {
	return grpcclient.NewTokenManager(accessToken, refreshToken, sandboxID)
}
