package api_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/yaoapp/yao/agent/robot/api"
	"github.com/yaoapp/yao/agent/testutils"
)

// TestLifecycle tests the Start/Stop lifecycle APIs
func TestLifecycle(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	testutils.Prepare(t)
	defer testutils.Clean(t)

	t.Run("start and stop cycle", func(t *testing.T) {
		// Initially not running
		assert.False(t, api.IsRunning())

		// Start
		err := api.Start()
		require.NoError(t, err)
		assert.True(t, api.IsRunning())

		// Start again should fail
		err = api.Start()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "already started")

		// Stop
		err = api.Stop()
		require.NoError(t, err)
		assert.False(t, api.IsRunning())

		// Stop again should be no-op (not error)
		err = api.Stop()
		assert.NoError(t, err)
	})

	t.Run("can restart after stop", func(t *testing.T) {
		// Start
		err := api.Start()
		require.NoError(t, err)
		assert.True(t, api.IsRunning())

		// Stop
		err = api.Stop()
		require.NoError(t, err)
		assert.False(t, api.IsRunning())

		// Start again should work
		err = api.Start()
		require.NoError(t, err)
		assert.True(t, api.IsRunning())

		// Cleanup
		api.Stop()
	})
}
