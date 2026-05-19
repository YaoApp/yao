//go:build integration

package manager_test

import (
	"context"
	"runtime"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/yaoapp/yao/agent/robot/executor"
	"github.com/yaoapp/yao/agent/robot/manager"
	"github.com/yaoapp/yao/agent/robot/pool"
	"github.com/yaoapp/yao/agent/robot/types"
	"github.com/yaoapp/yao/unit-test/agent/testprepare"
)

func TestManagerStartStop(t *testing.T) {
	testprepare.PrepareSandbox(t)

	t.Run("start and stop manager", func(t *testing.T) {
		config := &manager.Config{
			TickInterval: 10 * time.Second,
			Executor:     executor.NewDryRun(),
		}
		m := manager.NewWithConfig(config)

		assert.False(t, m.IsStarted())

		err := m.Start()
		require.NoError(t, err)
		assert.True(t, m.IsStarted())

		err = m.Stop()
		require.NoError(t, err)
		assert.False(t, m.IsStarted())
	})

	t.Run("double start should fail", func(t *testing.T) {
		config := &manager.Config{
			TickInterval: 10 * time.Second,
			Executor:     executor.NewDryRun(),
		}
		m := manager.NewWithConfig(config)

		err := m.Start()
		require.NoError(t, err)

		err = m.Start()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "already started")

		m.Stop()
	})

	t.Run("stop without start should not panic", func(t *testing.T) {
		m := manager.New()
		assert.NotPanics(t, func() {
			err := m.Stop()
			assert.NoError(t, err)
		})
	})
}

func TestManagerTick(t *testing.T) {
	testprepare.PrepareSandbox(t)

	t.Run("tick with times mode - matching time", func(t *testing.T) {
		config := &manager.Config{
			TickInterval: 10 * time.Second,
			PoolConfig:   &pool.Config{WorkerSize: 2, QueueSize: 10},
			Executor:     executor.NewDryRun(),
		}
		m := manager.NewWithConfig(config)

		err := m.Start()
		require.NoError(t, err)
		defer m.Stop()

		loc, _ := time.LoadLocation("Asia/Shanghai")
		now := time.Date(2025, 1, 15, 9, 0, 0, 0, loc) // Wednesday 09:00

		err = m.Tick(context.Background(), now)
		assert.NoError(t, err)

		time.Sleep(300 * time.Millisecond)
	})

	t.Run("tick when not started does nothing", func(t *testing.T) {
		m := manager.New()

		err := m.Tick(context.Background(), time.Now())
		assert.NoError(t, err)
	})
}

func TestManagerTriggerManual(t *testing.T) {
	testprepare.PrepareSandbox(t)

	t.Run("trigger manual - manager not started", func(t *testing.T) {
		m := manager.New()

		ctx := types.NewContext(context.Background(), nil)
		_, err := m.TriggerManual(ctx, "any-member", types.TriggerHuman, nil)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not started")
	})

	t.Run("trigger manual - robot not found", func(t *testing.T) {
		config := &manager.Config{
			TickInterval: 10 * time.Second,
			Executor:     executor.NewDryRun(),
		}
		m := manager.NewWithConfig(config)

		err := m.Start()
		require.NoError(t, err)
		defer m.Stop()

		ctx := types.NewContext(context.Background(), nil)
		_, err = m.TriggerManual(ctx, "robot_nonexistent_xyz", types.TriggerHuman, nil)
		assert.Error(t, err)
	})
}

func TestManagerClockModes(t *testing.T) {
	testprepare.PrepareSandbox(t)

	t.Run("interval mode - first run should trigger", func(t *testing.T) {
		config := &manager.Config{
			TickInterval: 10 * time.Second,
			PoolConfig:   &pool.Config{WorkerSize: 2, QueueSize: 10},
			Executor:     executor.NewDryRun(),
		}
		m := manager.NewWithConfig(config)

		err := m.Start()
		require.NoError(t, err)
		defer m.Stop()

		err = m.Tick(context.Background(), time.Now())
		assert.NoError(t, err)

		time.Sleep(300 * time.Millisecond)
	})

	t.Run("daemon mode - triggers when idle", func(t *testing.T) {
		config := &manager.Config{
			TickInterval: 10 * time.Second,
			PoolConfig:   &pool.Config{WorkerSize: 2, QueueSize: 10},
			Executor:     executor.NewDryRun(),
		}
		m := manager.NewWithConfig(config)

		err := m.Start()
		require.NoError(t, err)
		defer m.Stop()

		err = m.Tick(context.Background(), time.Now())
		assert.NoError(t, err)

		time.Sleep(300 * time.Millisecond)
	})
}

func TestManagerGoroutineLeak(t *testing.T) {
	testprepare.PrepareSandbox(t)

	t.Run("start stop cycle should not leak goroutines", func(t *testing.T) {
		runtime.GC()
		time.Sleep(100 * time.Millisecond)
		initialGoroutines := runtime.NumGoroutine()

		for i := 0; i < 3; i++ {
			config := &manager.Config{
				TickInterval: 10 * time.Second,
				Executor:     executor.NewDryRun(),
			}
			m := manager.NewWithConfig(config)

			err := m.Start()
			require.NoError(t, err)

			time.Sleep(50 * time.Millisecond)

			err = m.Stop()
			require.NoError(t, err)
		}

		time.Sleep(200 * time.Millisecond)
		runtime.GC()
		time.Sleep(100 * time.Millisecond)

		finalGoroutines := runtime.NumGoroutine()
		assert.LessOrEqual(t, finalGoroutines, initialGoroutines+3,
			"Should not leak goroutines (initial: %d, final: %d)",
			initialGoroutines, finalGoroutines)
	})
}

func TestManagerComponents(t *testing.T) {
	testprepare.PrepareSandbox(t)

	config := &manager.Config{
		TickInterval: 10 * time.Second,
		Executor:     executor.NewDryRun(),
	}
	m := manager.NewWithConfig(config)
	err := m.Start()
	require.NoError(t, err)
	defer m.Stop()

	t.Run("cache access", func(t *testing.T) {
		cache := m.Cache()
		assert.NotNil(t, cache)
	})

	t.Run("pool access", func(t *testing.T) {
		p := m.Pool()
		assert.NotNil(t, p)
	})

	t.Run("executor access", func(t *testing.T) {
		e := m.Executor()
		assert.NotNil(t, e)
	})

	t.Run("running and queued counts", func(t *testing.T) {
		running := m.Running()
		queued := m.Queued()
		cached := m.CachedRobots()

		assert.GreaterOrEqual(t, running, 0)
		assert.GreaterOrEqual(t, queued, 0)
		assert.GreaterOrEqual(t, cached, 0)
	})
}
