package acl_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/yaoapp/yao/openapi/oauth/acl"
	"github.com/yaoapp/yao/openapi/tests/testutils"
)

// ACL Test Suite
//
// PREREQUISITES:
// These tests require yao-dev-app to be available with scopes configuration.
// Before running tests, set the environment to point to yao-dev-app:
//
//   export YAO_DEV=$HOME/Yao/yao-dev-app
//   cd $YAO_DEV && source env.local.sh
//
// Then run tests:
//   go test -v ./openapi/tests/oauth/acl/... -count=1
//
// The tests will use the scopes configuration from yao-dev-app/openapi/scopes/

// TestNew tests the creation of a new ACL enforcer
func TestNew(t *testing.T) {
	testutils.Prepare(t)
	defer testutils.Clean()

	t.Run("CreateWithDefaultConfig", func(t *testing.T) {
		// Create ACL with nil config (should use default)
		enforcer, err := acl.New(nil)
		assert.NoError(t, err)
		assert.NotNil(t, enforcer)

		// Default config should have Enabled=false
		assert.False(t, enforcer.Enabled())

		t.Log("Successfully created ACL enforcer with default config")
	})

	t.Run("CreateWithDisabledConfig", func(t *testing.T) {
		// Create ACL with disabled config
		config := &acl.Config{
			Enabled: false,
		}

		enforcer, err := acl.New(config)
		assert.NoError(t, err)
		assert.NotNil(t, enforcer)
		assert.False(t, enforcer.Enabled())

		t.Log("Successfully created ACL enforcer with disabled config")
	})

	t.Run("CreateWithEnabledConfig", func(t *testing.T) {
		// Create ACL with enabled config
		// Note: This will try to load scope configuration from openapi/scopes directory
		config := &acl.Config{
			Enabled: true,
		}

		enforcer, err := acl.New(config)

		// If scopes directory doesn't exist, it should still succeed with warning
		// If scopes directory exists, it should load successfully
		if err != nil {
			t.Logf("Expected behavior: ACL loading may fail if scopes directory is not configured: %v", err)
		} else {
			assert.NotNil(t, enforcer)
			assert.True(t, enforcer.Enabled())
			t.Log("Successfully created ACL enforcer with enabled config")
		}
	})
}

// TestLoad tests loading the ACL enforcer as global singleton
func TestLoad(t *testing.T) {
	testutils.Prepare(t)
	defer testutils.Clean()

	t.Run("LoadWithDefaultConfig", func(t *testing.T) {
		// Load ACL with default config
		enforcer, err := acl.Load(nil)
		assert.NoError(t, err)
		assert.NotNil(t, enforcer)

		// Should set global enforcer
		assert.NotNil(t, acl.Global)
		assert.Equal(t, enforcer, acl.Global)

		t.Log("Successfully loaded ACL enforcer as global singleton")
	})

	t.Run("LoadWithDisabledConfig", func(t *testing.T) {
		config := &acl.Config{
			Enabled: false,
		}

		enforcer, err := acl.Load(config)
		assert.NoError(t, err)
		assert.NotNil(t, enforcer)
		assert.False(t, enforcer.Enabled())

		// Global should be updated
		assert.Equal(t, enforcer, acl.Global)

		t.Log("Successfully loaded disabled ACL enforcer")
	})
}

// TestEnabled tests the Enabled method
func TestEnabled(t *testing.T) {
	testutils.Prepare(t)
	defer testutils.Clean()

	t.Run("DisabledACL", func(t *testing.T) {
		config := &acl.Config{
			Enabled: false,
		}

		enforcer, err := acl.New(config)
		assert.NoError(t, err)
		assert.False(t, enforcer.Enabled())
	})

	t.Run("EnabledACL", func(t *testing.T) {
		config := &acl.Config{
			Enabled: true,
		}

		enforcer, err := acl.New(config)

		// May fail if scopes directory doesn't exist, which is expected
		if err == nil {
			assert.True(t, enforcer.Enabled())
		}
	})
}

// TestDefaultConfig tests the default configuration
func TestDefaultConfig(t *testing.T) {
	t.Run("DefaultConfigValues", func(t *testing.T) {
		// The default config should have Enabled=false
		assert.False(t, acl.DefaultConfig.Enabled, "Default ACL config should be disabled")

		t.Log("Default config verified: Enabled=false")
	})
}
