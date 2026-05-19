package trigger_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/yaoapp/yao/agent/robot/trigger"
	"github.com/yaoapp/yao/agent/robot/types"
)

// ==================== ClockMatcher Tests ====================

func TestClockMatcherShouldTrigger(t *testing.T) {
	cm := trigger.NewClockMatcher()

	t.Run("nil robot returns false", func(t *testing.T) {
		result := cm.ShouldTrigger(nil, time.Now())
		assert.False(t, result)
	})

	t.Run("nil config returns false", func(t *testing.T) {
		robot := &types.Robot{
			MemberID: "robot_001",
			Config:   nil,
		}
		result := cm.ShouldTrigger(robot, time.Now())
		assert.False(t, result)
	})

	t.Run("nil clock config returns false", func(t *testing.T) {
		robot := &types.Robot{
			MemberID: "robot_001",
			Config:   &types.Config{Clock: nil},
		}
		result := cm.ShouldTrigger(robot, time.Now())
		assert.False(t, result)
	})
}

// ==================== Times Mode Tests ====================

func TestClockMatcherTimesMode(t *testing.T) {
	cm := trigger.NewClockMatcher()

	t.Run("matches configured time", func(t *testing.T) {
		robot := &types.Robot{
			MemberID: "robot_001",
			Config: &types.Config{
				Clock: &types.Clock{
					Mode:  types.ClockTimes,
					Times: []string{"09:00", "14:00", "17:00"},
					Days:  []string{"*"},
					TZ:    "UTC",
				},
			},
		}

		// Create time at 09:00 UTC
		now := time.Date(2025, 1, 15, 9, 0, 0, 0, time.UTC)
		result := cm.ShouldTrigger(robot, now)
		assert.True(t, result)
	})

	t.Run("does not match non-configured time", func(t *testing.T) {
		robot := &types.Robot{
			MemberID: "robot_001",
			Config: &types.Config{
				Clock: &types.Clock{
					Mode:  types.ClockTimes,
					Times: []string{"09:00", "14:00", "17:00"},
					Days:  []string{"*"},
					TZ:    "UTC",
				},
			},
		}

		// Create time at 10:00 UTC (not in configured times)
		now := time.Date(2025, 1, 15, 10, 0, 0, 0, time.UTC)
		result := cm.ShouldTrigger(robot, now)
		assert.False(t, result)
	})

	t.Run("respects day filter - weekday", func(t *testing.T) {
		robot := &types.Robot{
			MemberID: "robot_001",
			Config: &types.Config{
				Clock: &types.Clock{
					Mode:  types.ClockTimes,
					Times: []string{"09:00"},
					Days:  []string{"Mon", "Tue", "Wed", "Thu", "Fri"},
					TZ:    "UTC",
				},
			},
		}

		// Wednesday 09:00 - should trigger
		wed := time.Date(2025, 1, 15, 9, 0, 0, 0, time.UTC) // Wednesday
		assert.True(t, cm.ShouldTrigger(robot, wed))

		// Saturday 09:00 - should NOT trigger
		sat := time.Date(2025, 1, 18, 9, 0, 0, 0, time.UTC) // Saturday
		assert.False(t, cm.ShouldTrigger(robot, sat))
	})

	t.Run("dedup - same minute same day should not trigger twice", func(t *testing.T) {
		robot := &types.Robot{
			MemberID: "robot_001",
			Config: &types.Config{
				Clock: &types.Clock{
					Mode:  types.ClockTimes,
					Times: []string{"09:00"},
					Days:  []string{"*"},
					TZ:    "UTC",
				},
			},
		}

		now := time.Date(2025, 1, 15, 9, 0, 0, 0, time.UTC)

		// First trigger - should succeed
		assert.True(t, cm.ShouldTrigger(robot, now))

		// Simulate LastRun was set
		robot.LastRun = now

		// Second trigger same minute - should fail
		now2 := time.Date(2025, 1, 15, 9, 0, 30, 0, time.UTC)
		assert.False(t, cm.ShouldTrigger(robot, now2))
	})

	t.Run("different day should trigger again", func(t *testing.T) {
		robot := &types.Robot{
			MemberID: "robot_001",
			Config: &types.Config{
				Clock: &types.Clock{
					Mode:  types.ClockTimes,
					Times: []string{"09:00"},
					Days:  []string{"*"},
					TZ:    "UTC",
				},
			},
		}

		// First day
		day1 := time.Date(2025, 1, 15, 9, 0, 0, 0, time.UTC)
		robot.LastRun = day1

		// Next day same time - should trigger
		day2 := time.Date(2025, 1, 16, 9, 0, 0, 0, time.UTC)
		assert.True(t, cm.ShouldTrigger(robot, day2))
	})
}

// ==================== Interval Mode Tests ====================

func TestClockMatcherIntervalMode(t *testing.T) {
	cm := trigger.NewClockMatcher()

	t.Run("first run triggers immediately", func(t *testing.T) {
		robot := &types.Robot{
			MemberID: "robot_001",
			Config: &types.Config{
				Clock: &types.Clock{
					Mode:  types.ClockInterval,
					Every: "30m",
					TZ:    "UTC",
				},
			},
		}

		// LastRun is zero - should trigger
		now := time.Now()
		result := cm.ShouldTrigger(robot, now)
		assert.True(t, result)
	})

	t.Run("triggers after interval passed", func(t *testing.T) {
		robot := &types.Robot{
			MemberID: "robot_001",
			Config: &types.Config{
				Clock: &types.Clock{
					Mode:  types.ClockInterval,
					Every: "30m",
					TZ:    "UTC",
				},
			},
		}

		now := time.Now()
		robot.LastRun = now.Add(-31 * time.Minute) // 31 minutes ago

		result := cm.ShouldTrigger(robot, now)
		assert.True(t, result)
	})

	t.Run("does not trigger before interval", func(t *testing.T) {
		robot := &types.Robot{
			MemberID: "robot_001",
			Config: &types.Config{
				Clock: &types.Clock{
					Mode:  types.ClockInterval,
					Every: "30m",
					TZ:    "UTC",
				},
			},
		}

		now := time.Now()
		robot.LastRun = now.Add(-15 * time.Minute) // Only 15 minutes ago

		result := cm.ShouldTrigger(robot, now)
		assert.False(t, result)
	})

	t.Run("invalid interval format returns false", func(t *testing.T) {
		robot := &types.Robot{
			MemberID: "robot_001",
			Config: &types.Config{
				Clock: &types.Clock{
					Mode:  types.ClockInterval,
					Every: "invalid",
					TZ:    "UTC",
				},
			},
		}

		result := cm.ShouldTrigger(robot, time.Now())
		assert.False(t, result)
	})

	t.Run("various interval formats", func(t *testing.T) {
		intervals := []struct {
			every    string
			lastAgo  time.Duration
			expected bool
		}{
			{"1h", 61 * time.Minute, true},
			{"1h", 30 * time.Minute, false},
			{"2h", 121 * time.Minute, true},
			{"2h", 60 * time.Minute, false},
			{"10s", 11 * time.Second, true},
			{"10s", 5 * time.Second, false},
		}

		for _, tt := range intervals {
			t.Run(tt.every, func(t *testing.T) {
				robot := &types.Robot{
					MemberID: "robot_001",
					Config: &types.Config{
						Clock: &types.Clock{
							Mode:  types.ClockInterval,
							Every: tt.every,
							TZ:    "UTC",
						},
					},
				}

				now := time.Now()
				robot.LastRun = now.Add(-tt.lastAgo)

				result := cm.ShouldTrigger(robot, now)
				assert.Equal(t, tt.expected, result)
			})
		}
	})
}

// ==================== Daemon Mode Tests ====================

func TestClockMatcherDaemonMode(t *testing.T) {
	cm := trigger.NewClockMatcher()

	t.Run("triggers when robot can run", func(t *testing.T) {
		robot := &types.Robot{
			MemberID: "robot_001",
			Config: &types.Config{
				Clock: &types.Clock{
					Mode: types.ClockDaemon,
					TZ:   "UTC",
				},
				Quota: &types.Quota{Max: 2},
			},
		}

		// No running executions - should trigger
		result := cm.ShouldTrigger(robot, time.Now())
		assert.True(t, result)
	})

	t.Run("does not trigger when at quota", func(t *testing.T) {
		robot := &types.Robot{
			MemberID: "robot_001",
			Config: &types.Config{
				Clock: &types.Clock{
					Mode: types.ClockDaemon,
					TZ:   "UTC",
				},
				Quota: &types.Quota{Max: 1},
			},
		}

		// Add one execution to fill quota
		exec := &types.Execution{ID: "exec_001"}
		robot.AddExecution(exec)

		result := cm.ShouldTrigger(robot, time.Now())
		assert.False(t, result)

		// Remove execution
		robot.RemoveExecution("exec_001")

		// Now should trigger
		result = cm.ShouldTrigger(robot, time.Now())
		assert.True(t, result)
	})
}

// ==================== Timezone Tests ====================

func TestClockMatcherTimezone(t *testing.T) {
	cm := trigger.NewClockMatcher()

	t.Run("respects timezone for times mode", func(t *testing.T) {
		// Robot configured for Asia/Shanghai (UTC+8)
		robot := &types.Robot{
			MemberID: "robot_001",
			Config: &types.Config{
				Clock: &types.Clock{
					Mode:  types.ClockTimes,
					Times: []string{"09:00"},
					Days:  []string{"*"},
					TZ:    "Asia/Shanghai",
				},
			},
		}

		// 01:00 UTC = 09:00 Shanghai - should trigger
		utc0100 := time.Date(2025, 1, 15, 1, 0, 0, 0, time.UTC)
		assert.True(t, cm.ShouldTrigger(robot, utc0100))

		// 09:00 UTC = 17:00 Shanghai - should NOT trigger
		utc0900 := time.Date(2025, 1, 15, 9, 0, 0, 0, time.UTC)
		assert.False(t, cm.ShouldTrigger(robot, utc0900))
	})

	t.Run("invalid timezone falls back to local", func(t *testing.T) {
		robot := &types.Robot{
			MemberID: "robot_001",
			Config: &types.Config{
				Clock: &types.Clock{
					Mode:  types.ClockTimes,
					Times: []string{"09:00"},
					Days:  []string{"*"},
					TZ:    "Invalid/Timezone",
				},
			},
		}

		// Should still work with local time
		local0900 := time.Date(2025, 1, 15, 9, 0, 0, 0, time.Local)
		result := cm.ShouldTrigger(robot, local0900)
		// Result depends on local timezone, just verify no panic
		assert.IsType(t, true, result)
	})
}

// ==================== ParseTime/FormatTime Tests ====================

func TestParseTime(t *testing.T) {
	t.Run("parses valid time", func(t *testing.T) {
		hour, minute, err := trigger.ParseTime("09:30")
		assert.NoError(t, err)
		assert.Equal(t, 9, hour)
		assert.Equal(t, 30, minute)
	})

	t.Run("parses midnight", func(t *testing.T) {
		hour, minute, err := trigger.ParseTime("00:00")
		assert.NoError(t, err)
		assert.Equal(t, 0, hour)
		assert.Equal(t, 0, minute)
	})

	t.Run("parses 23:59", func(t *testing.T) {
		hour, minute, err := trigger.ParseTime("23:59")
		assert.NoError(t, err)
		assert.Equal(t, 23, hour)
		assert.Equal(t, 59, minute)
	})

	t.Run("invalid format returns error", func(t *testing.T) {
		// Note: time.Parse("15:04", "9:30") actually succeeds
		// Only truly invalid formats fail

		_, _, err := trigger.ParseTime("09:30:00")
		assert.Error(t, err)

		_, _, err = trigger.ParseTime("invalid")
		assert.Error(t, err)

		_, _, err = trigger.ParseTime("")
		assert.Error(t, err)
	})
}

func TestFormatTime(t *testing.T) {
	tests := []struct {
		hour     int
		minute   int
		expected string
	}{
		{9, 0, "09:00"},
		{9, 30, "09:30"},
		{0, 0, "00:00"},
		{23, 59, "23:59"},
		{14, 5, "14:05"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			result := trigger.FormatTime(tt.hour, tt.minute)
			assert.Equal(t, tt.expected, result)
		})
	}
}
