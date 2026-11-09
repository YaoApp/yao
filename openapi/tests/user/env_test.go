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

	// Skip this test if environment variables are not set
	// This test is meant to verify the environment setup but doesn't affect functionality
	if signinClientID == "" || signinClientSecret == "" {
		t.Skip("Skipping environment variable test: SIGNIN_CLIENT_ID or SIGNIN_CLIENT_SECRET not set. " +
			"This is expected in some test environments.")
	}

	// Check if client ID is exactly 32 characters
	assert.Equal(t, 32, len(signinClientID), "SIGNIN_CLIENT_ID should be exactly 32 characters")
}
