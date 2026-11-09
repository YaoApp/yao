package openapi_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/yaoapp/yao/openapi"
	"github.com/yaoapp/yao/openapi/tests/testutils"
)

func TestLoad(t *testing.T) {
	serverURL := testutils.Prepare(t)
	defer testutils.Clean()

	assert.NotNil(t, openapi.Server)
	assert.NotEmpty(t, serverURL)
	assert.Contains(t, serverURL, "http://127.0.0.1:")
}

func TestObtainAccessToken(t *testing.T) {
	serverURL := testutils.Prepare(t)
	defer testutils.Clean()

	// Register a test client
	client := testutils.RegisterTestClient(t, "Token Utility Test Client", []string{"https://localhost/callback"})
	defer testutils.CleanupTestClient(t, client.ClientID)

	// Test the ObtainAccessToken utility function
	tokenInfo := testutils.ObtainAccessToken(t, serverURL, client.ClientID, client.ClientSecret, "https://localhost/callback", "openid profile email")

	// Verify token information
	assert.NotEmpty(t, tokenInfo.AccessToken, "Access token should not be empty")
	assert.NotEmpty(t, tokenInfo.RefreshToken, "Refresh token should not be empty")
	assert.Equal(t, "Bearer", tokenInfo.TokenType, "Token type should be Bearer")
	assert.Greater(t, tokenInfo.ExpiresIn, 0, "ExpiresIn should be greater than 0")
	assert.Equal(t, client.ClientID, tokenInfo.ClientID, "Client ID should match")
	// Note: Scope might be empty in token response, which is valid

	t.Logf("Successfully obtained token: AccessToken=%s, TokenType=%s, ExpiresIn=%d, Scope=%s",
		tokenInfo.AccessToken, tokenInfo.TokenType, tokenInfo.ExpiresIn, tokenInfo.Scope)
}
