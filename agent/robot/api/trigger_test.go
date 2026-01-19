package api_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/yaoapp/yao/agent/robot/api"
	"github.com/yaoapp/yao/agent/robot/types"
	"github.com/yaoapp/yao/agent/testutils"
)

// TestTriggerValidation tests parameter validation for Trigger
func TestTriggerValidation(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	testutils.Prepare(t)
	defer testutils.Clean(t)

	ctx := types.NewContext(context.Background(), nil)

	t.Run("returns error for empty member_id", func(t *testing.T) {
		result, err := api.Trigger(ctx, "", &api.TriggerRequest{
			Type: types.TriggerHuman,
		})
		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "member_id is required")
	})

	t.Run("returns error for nil request", func(t *testing.T) {
		result, err := api.Trigger(ctx, "test_member", nil)
		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "trigger request is required")
	})

	t.Run("returns error when manager not started", func(t *testing.T) {
		result, err := api.Trigger(ctx, "test_member", &api.TriggerRequest{
			Type: types.TriggerHuman,
		})
		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "not started")
	})
}

// TestTriggerManualValidation tests parameter validation for TriggerManual
func TestTriggerManualValidation(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	testutils.Prepare(t)
	defer testutils.Clean(t)

	ctx := types.NewContext(context.Background(), nil)

	t.Run("returns error for empty member_id", func(t *testing.T) {
		result, err := api.TriggerManual(ctx, "", types.TriggerClock, nil)
		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "member_id is required")
	})

	t.Run("returns error when manager not started", func(t *testing.T) {
		result, err := api.TriggerManual(ctx, "test_member", types.TriggerClock, nil)
		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "not started")
	})
}

// TestInterveneValidation tests parameter validation for Intervene
func TestInterveneValidation(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	testutils.Prepare(t)
	defer testutils.Clean(t)

	ctx := types.NewContext(context.Background(), nil)

	t.Run("returns error for empty member_id", func(t *testing.T) {
		result, err := api.Intervene(ctx, "", &api.TriggerRequest{
			Type:   types.TriggerHuman,
			Action: types.ActionTaskAdd,
		})
		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "member_id is required")
	})

	t.Run("returns error for nil request", func(t *testing.T) {
		result, err := api.Intervene(ctx, "test_member", nil)
		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "intervention request is required")
	})
}

// TestHandleEventValidation tests parameter validation for HandleEvent
func TestHandleEventValidation(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	testutils.Prepare(t)
	defer testutils.Clean(t)

	ctx := types.NewContext(context.Background(), nil)

	t.Run("returns error for empty member_id", func(t *testing.T) {
		result, err := api.HandleEvent(ctx, "", &api.TriggerRequest{
			Type:      types.TriggerEvent,
			Source:    types.EventWebhook,
			EventType: "test.event",
		})
		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "member_id is required")
	})

	t.Run("returns error for nil request", func(t *testing.T) {
		result, err := api.HandleEvent(ctx, "test_member", nil)
		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "event request is required")
	})
}

// TestTriggerWithManagerStarted tests trigger APIs when manager is running
func TestTriggerWithManagerStarted(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	testutils.Prepare(t)
	defer testutils.Clean(t)

	// Start manager
	err := api.Start()
	require.NoError(t, err)
	defer api.Stop()

	ctx := types.NewContext(context.Background(), nil)

	t.Run("returns not accepted for non-existent robot", func(t *testing.T) {
		result, err := api.Trigger(ctx, "non_existent_robot_xyz", &api.TriggerRequest{
			Type:   types.TriggerHuman,
			Action: types.ActionTaskAdd,
		})
		// Should not error, but return not accepted
		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.False(t, result.Accepted)
	})

	t.Run("returns error for invalid trigger type", func(t *testing.T) {
		result, err := api.Trigger(ctx, "test_member", &api.TriggerRequest{
			Type: "invalid_type",
		})
		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "invalid trigger type")
	})
}
