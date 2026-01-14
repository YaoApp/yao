package types_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/yaoapp/yao/agent/robot/types"
)

func TestConfigValidate(t *testing.T) {
	t.Run("valid config", func(t *testing.T) {
		config := &types.Config{
			Identity: &types.Identity{
				Role: "Sales Manager",
			},
		}
		err := config.Validate()
		assert.NoError(t, err)
	})

	t.Run("missing identity", func(t *testing.T) {
		config := &types.Config{}
		err := config.Validate()
		assert.Error(t, err)
		assert.Equal(t, types.ErrMissingIdentity, err)
	})

	t.Run("missing identity role", func(t *testing.T) {
		config := &types.Config{
			Identity: &types.Identity{},
		}
		err := config.Validate()
		assert.Error(t, err)
		assert.Equal(t, types.ErrMissingIdentity, err)
	})

	t.Run("invalid clock config", func(t *testing.T) {
		config := &types.Config{
			Identity: &types.Identity{Role: "Test"},
			Clock: &types.Clock{
				Mode: types.ClockTimes,
				// Times is empty - should fail
			},
		}
		err := config.Validate()
		assert.Error(t, err)
		assert.Equal(t, types.ErrClockTimesEmpty, err)
	})
}

func TestClockValidate(t *testing.T) {
	t.Run("valid times mode", func(t *testing.T) {
		clock := &types.Clock{
			Mode:  types.ClockTimes,
			Times: []string{"09:00", "14:00"},
		}
		err := clock.Validate()
		assert.NoError(t, err)
	})

	t.Run("times mode without times", func(t *testing.T) {
		clock := &types.Clock{
			Mode: types.ClockTimes,
		}
		err := clock.Validate()
		assert.Error(t, err)
		assert.Equal(t, types.ErrClockTimesEmpty, err)
	})

	t.Run("valid interval mode", func(t *testing.T) {
		clock := &types.Clock{
			Mode:  types.ClockInterval,
			Every: "30m",
		}
		err := clock.Validate()
		assert.NoError(t, err)
	})

	t.Run("interval mode without every", func(t *testing.T) {
		clock := &types.Clock{
			Mode: types.ClockInterval,
		}
		err := clock.Validate()
		assert.Error(t, err)
		assert.Equal(t, types.ErrClockIntervalEmpty, err)
	})

	t.Run("valid daemon mode", func(t *testing.T) {
		clock := &types.Clock{
			Mode: types.ClockDaemon,
		}
		err := clock.Validate()
		assert.NoError(t, err)
	})

	t.Run("invalid mode", func(t *testing.T) {
		clock := &types.Clock{
			Mode: types.ClockMode("invalid"),
		}
		err := clock.Validate()
		assert.Error(t, err)
		assert.Equal(t, types.ErrClockModeInvalid, err)
	})
}

func TestClockGetTimeout(t *testing.T) {
	t.Run("default timeout", func(t *testing.T) {
		clock := &types.Clock{}
		timeout := clock.GetTimeout()
		assert.Equal(t, 30*time.Minute, timeout)
	})

	t.Run("custom timeout", func(t *testing.T) {
		clock := &types.Clock{
			Timeout: "10m",
		}
		timeout := clock.GetTimeout()
		assert.Equal(t, 10*time.Minute, timeout)
	})

	t.Run("invalid timeout returns default", func(t *testing.T) {
		clock := &types.Clock{
			Timeout: "invalid",
		}
		timeout := clock.GetTimeout()
		assert.Equal(t, 30*time.Minute, timeout)
	})
}

func TestClockGetLocation(t *testing.T) {
	t.Run("default location", func(t *testing.T) {
		clock := &types.Clock{}
		loc := clock.GetLocation()
		assert.Equal(t, time.Local, loc)
	})

	t.Run("valid timezone", func(t *testing.T) {
		clock := &types.Clock{
			TZ: "Asia/Shanghai",
		}
		loc := clock.GetLocation()
		assert.NotNil(t, loc)
		assert.Equal(t, "Asia/Shanghai", loc.String())
	})

	t.Run("invalid timezone returns local", func(t *testing.T) {
		clock := &types.Clock{
			TZ: "Invalid/Timezone",
		}
		loc := clock.GetLocation()
		assert.Equal(t, time.Local, loc)
	})
}

func TestTriggersIsEnabled(t *testing.T) {
	t.Run("nil triggers - all enabled by default", func(t *testing.T) {
		var triggers *types.Triggers
		assert.True(t, triggers.IsEnabled(types.TriggerClock))
		assert.True(t, triggers.IsEnabled(types.TriggerHuman))
		assert.True(t, triggers.IsEnabled(types.TriggerEvent))
	})

	t.Run("clock enabled", func(t *testing.T) {
		triggers := &types.Triggers{
			Clock: &types.TriggerSwitch{Enabled: true},
		}
		assert.True(t, triggers.IsEnabled(types.TriggerClock))
	})

	t.Run("clock disabled", func(t *testing.T) {
		triggers := &types.Triggers{
			Clock: &types.TriggerSwitch{Enabled: false},
		}
		assert.False(t, triggers.IsEnabled(types.TriggerClock))
	})

	t.Run("intervene enabled by default", func(t *testing.T) {
		triggers := &types.Triggers{}
		assert.True(t, triggers.IsEnabled(types.TriggerHuman))
	})

	t.Run("event disabled", func(t *testing.T) {
		triggers := &types.Triggers{
			Event: &types.TriggerSwitch{Enabled: false},
		}
		assert.False(t, triggers.IsEnabled(types.TriggerEvent))
	})
}

func TestQuotaDefaults(t *testing.T) {
	t.Run("nil quota", func(t *testing.T) {
		var quota *types.Quota
		assert.Equal(t, 2, quota.GetMax())
		assert.Equal(t, 10, quota.GetQueue())
		assert.Equal(t, 5, quota.GetPriority())
	})

	t.Run("zero values", func(t *testing.T) {
		quota := &types.Quota{}
		assert.Equal(t, 2, quota.GetMax())
		assert.Equal(t, 10, quota.GetQueue())
		assert.Equal(t, 5, quota.GetPriority())
	})

	t.Run("custom values", func(t *testing.T) {
		quota := &types.Quota{
			Max:      5,
			Queue:    20,
			Priority: 8,
		}
		assert.Equal(t, 5, quota.GetMax())
		assert.Equal(t, 20, quota.GetQueue())
		assert.Equal(t, 8, quota.GetPriority())
	})
}

func TestResourcesGetPhaseAgent(t *testing.T) {
	t.Run("nil resources - returns default", func(t *testing.T) {
		var resources *types.Resources
		agent := resources.GetPhaseAgent(types.PhaseGoals)
		assert.Equal(t, "__yao.goals", agent)
	})

	t.Run("phase not configured - returns default", func(t *testing.T) {
		resources := &types.Resources{
			Phases: map[types.Phase]string{},
		}
		agent := resources.GetPhaseAgent(types.PhaseGoals)
		assert.Equal(t, "__yao.goals", agent)
	})

	t.Run("custom phase agent", func(t *testing.T) {
		resources := &types.Resources{
			Phases: map[types.Phase]string{
				types.PhaseGoals: "custom.goals.agent",
			},
		}
		agent := resources.GetPhaseAgent(types.PhaseGoals)
		assert.Equal(t, "custom.goals.agent", agent)
	})

	t.Run("all phases default names", func(t *testing.T) {
		resources := &types.Resources{}
		assert.Equal(t, "__yao.inspiration", resources.GetPhaseAgent(types.PhaseInspiration))
		assert.Equal(t, "__yao.goals", resources.GetPhaseAgent(types.PhaseGoals))
		assert.Equal(t, "__yao.tasks", resources.GetPhaseAgent(types.PhaseTasks))
		assert.Equal(t, "__yao.run", resources.GetPhaseAgent(types.PhaseRun))
		assert.Equal(t, "__yao.delivery", resources.GetPhaseAgent(types.PhaseDelivery))
		assert.Equal(t, "__yao.learning", resources.GetPhaseAgent(types.PhaseLearning))
	})
}
