package user_test

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestEnvironmentVariables(t *testing.T) {
	// Test that environment variables are available
	signinClientID := os.Getenv("SIGNIN_CLIENT_ID")
	signinClientSecret := os.Getenv("SIGNIN_CLIENT_SECRET")

	t.Logf("SIGNIN_CLIENT_ID: %s (length: %d)", signinClientID, len(signinClientID))
	t.Logf("SIGNIN_CLIENT_SECRET: %s (length: %d)", signinClientSecret, len(signinClientSecret))

	// Check if environment variables are set
	assert.NotEmpty(t, signinClientID, "SIGNIN_CLIENT_ID should be set")
	assert.NotEmpty(t, signinClientSecret, "SIGNIN_CLIENT_SECRET should be set")

	// Check if client ID is exactly 32 characters
	assert.Equal(t, 32, len(signinClientID), "SIGNIN_CLIENT_ID should be exactly 32 characters")
}
