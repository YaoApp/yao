//go:build unit

package manager_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/yaoapp/yao/agent/robot/manager"
	"github.com/yaoapp/yao/agent/robot/types"
)

func TestInteractRequestStructFields(t *testing.T) {
	req := &manager.InteractRequest{
		ExecutionID: "exec-1",
		TaskID:      "task-1",
		Source:      types.InteractSourceUI,
		Message:     "do something",
		Action:      "confirm",
	}
	assert.Equal(t, "exec-1", req.ExecutionID)
	assert.Equal(t, "task-1", req.TaskID)
	assert.Equal(t, types.InteractSourceUI, req.Source)
	assert.Equal(t, "do something", req.Message)
	assert.Equal(t, "confirm", req.Action)
}

func TestInteractResponseStructFields(t *testing.T) {
	resp := &manager.InteractResponse{
		ExecutionID: "exec-1",
		Status:      "confirmed",
		Message:     "Done",
		ChatID:      "chat-1",
		Reply:       "I'll do it",
		WaitForMore: true,
	}
	assert.Equal(t, "exec-1", resp.ExecutionID)
	assert.Equal(t, "confirmed", resp.Status)
	assert.Equal(t, "Done", resp.Message)
	assert.Equal(t, "chat-1", resp.ChatID)
	assert.Equal(t, "I'll do it", resp.Reply)
	assert.True(t, resp.WaitForMore)
}

func TestManagerNew(t *testing.T) {
	t.Run("creates manager with default config", func(t *testing.T) {
		m := manager.New()
		require.NotNil(t, m)
		assert.False(t, m.IsStarted())
		assert.Equal(t, 0, m.Running())
		assert.Equal(t, 0, m.Queued())
	})

	t.Run("creates manager with custom config", func(t *testing.T) {
		config := &manager.Config{
			TickInterval: 5000000000, // 5 seconds
		}
		m := manager.NewWithConfig(config)
		require.NotNil(t, m)
		assert.False(t, m.IsStarted())
	})

	t.Run("nil config uses defaults", func(t *testing.T) {
		m := manager.NewWithConfig(nil)
		require.NotNil(t, m)
		assert.False(t, m.IsStarted())
	})
}

func TestManagerNotStartedErrors(t *testing.T) {
	m := manager.New()

	t.Run("TriggerManual returns error when not started", func(t *testing.T) {
		ctx := types.NewContext(context.Background(), nil)
		_, err := m.TriggerManual(ctx, "some-member", types.TriggerHuman, nil)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not started")
	})

	t.Run("Intervene returns error when not started", func(t *testing.T) {
		ctx := types.NewContext(context.Background(), nil)
		_, err := m.Intervene(ctx, &types.InterveneRequest{
			MemberID: "some-member",
			Action:   types.ActionTaskAdd,
		})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not started")
	})

	t.Run("HandleEvent returns error when not started", func(t *testing.T) {
		ctx := types.NewContext(context.Background(), nil)
		_, err := m.HandleEvent(ctx, &types.EventRequest{
			MemberID:  "some-member",
			Source:    "webhook",
			EventType: "test.event",
		})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not started")
	})

	t.Run("HandleInteract returns error when not started", func(t *testing.T) {
		ctx := types.NewContext(context.Background(), nil)
		_, err := m.HandleInteract(ctx, "some-member", &manager.InteractRequest{Message: "test"})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not started")
	})
}

func TestManagerStopWithoutStartNoPanic(t *testing.T) {
	m := manager.New()
	assert.NotPanics(t, func() {
		err := m.Stop()
		assert.NoError(t, err)
	})
}

func TestDefaultConfig(t *testing.T) {
	config := manager.DefaultConfig()
	require.NotNil(t, config)
	assert.Equal(t, manager.DefaultTickInterval, config.TickInterval)
	assert.NotNil(t, config.PoolConfig)
	assert.Nil(t, config.Executor)
}
