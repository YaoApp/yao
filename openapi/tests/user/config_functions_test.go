package user_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/yaoapp/yao/openapi/user"
)

// TestGetTeamConfigFunction tests the GetTeamConfig function
func TestGetTeamConfigFunction(t *testing.T) {
	// Test with empty locale
	teamConfig := user.GetTeamConfig("")
	assert.Nil(t, teamConfig, "Should return nil when no config is loaded")

	// Test with specific locale
	teamConfig = user.GetTeamConfig("en")
	assert.Nil(t, teamConfig, "Should return nil when no config is loaded")

	// Test with invalid locale
	teamConfig = user.GetTeamConfig("invalid")
	assert.Nil(t, teamConfig, "Should return nil when no config is loaded")
}

// TestGetEntryConfigFunction tests the GetEntryConfig function
func TestGetEntryConfigFunction(t *testing.T) {
	// Test with empty locale
	entryConfig := user.GetEntryConfig("")
	assert.Nil(t, entryConfig, "Should return nil when no config is loaded")

	// Test with specific locale
	entryConfig = user.GetEntryConfig("en")
	assert.Nil(t, entryConfig, "Should return nil when no config is loaded")

	// Test with invalid locale
	entryConfig = user.GetEntryConfig("invalid")
	assert.Nil(t, entryConfig, "Should return nil when no config is loaded")
}

// TestGetYaoClientConfigFunction tests the GetYaoClientConfig function
func TestGetYaoClientConfigFunction(t *testing.T) {
	// Test when no client config is loaded
	clientConfig := user.GetYaoClientConfig()
	assert.Nil(t, clientConfig, "Should return nil when no client config is loaded")
}

// TestGetProviderFunction tests the GetProvider function
func TestGetProviderFunction(t *testing.T) {
	// Test with non-existent provider
	provider, err := user.GetProvider("non-existent")
	assert.Error(t, err, "Should return error for non-existent provider")
	assert.Nil(t, provider, "Should return nil provider for non-existent provider")
	assert.Contains(t, err.Error(), "not found", "Error should contain 'not found'")

	// Test with empty provider ID
	provider, err = user.GetProvider("")
	assert.Error(t, err, "Should return error for empty provider ID")
	assert.Nil(t, provider, "Should return nil provider for empty provider ID")
}
