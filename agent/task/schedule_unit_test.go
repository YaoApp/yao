//go:build unit

package task_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/yaoapp/yao/agent/task"
)

func TestShouldTrigger_Once(t *testing.T) {
	entry := &task.ScheduleEntry{Config: task.ScheduleConfig{Mode: "once"}}

	assert.True(t, task.ExportShouldTrigger(entry, time.Now()))

	entry.LastRun = time.Now()
	assert.False(t, task.ExportShouldTrigger(entry, time.Now()))
}

func TestShouldTrigger_Times(t *testing.T) {
	now := time.Date(2026, 6, 15, 9, 0, 0, 0, time.Local)
	entry := &task.ScheduleEntry{Config: task.ScheduleConfig{Mode: "times", Times: []string{"09:00", "18:00"}}}

	assert.True(t, task.ExportShouldTrigger(entry, now))

	entry.LastRun = now
	assert.False(t, task.ExportShouldTrigger(entry, now))

	entry.LastRun = time.Time{}
	assert.False(t, task.ExportShouldTrigger(entry, now.Add(30*time.Minute)))
}

func TestShouldTrigger_Interval(t *testing.T) {
	entry := &task.ScheduleEntry{
		Config:  task.ScheduleConfig{Mode: "interval", IntervalValue: 30, IntervalUnit: "minutes"},
		LastRun: time.Now().Add(-31 * time.Minute),
	}

	assert.True(t, task.ExportShouldTrigger(entry, time.Now()))

	entry.LastRun = time.Now().Add(-10 * time.Minute)
	assert.False(t, task.ExportShouldTrigger(entry, time.Now()))
}

func TestShouldTrigger_Daemon(t *testing.T) {
	entry := &task.ScheduleEntry{
		Config:    task.ScheduleConfig{Mode: "daemon"},
		LastRun:   time.Now().Add(-20 * time.Second),
		FailCount: 0,
	}

	assert.True(t, task.ExportShouldTrigger(entry, time.Now()))

	entry.FailCount = 5
	entry.LastRun = time.Now().Add(-30 * time.Second)
	assert.False(t, task.ExportShouldTrigger(entry, time.Now()))
}

func TestCalcBackoff(t *testing.T) {
	assert.Equal(t, 10*time.Second, task.ExportCalcBackoff(0))
	assert.Equal(t, 10*time.Second, task.ExportCalcBackoff(1))
	assert.Equal(t, 40*time.Second, task.ExportCalcBackoff(2))
	assert.Equal(t, 5*time.Minute, task.ExportCalcBackoff(10))
}

func TestIntervalDuration(t *testing.T) {
	assert.Equal(t, 30*time.Minute, task.ExportIntervalDuration(30, "minutes"))
	assert.Equal(t, 2*time.Hour, task.ExportIntervalDuration(2, "hours"))
	assert.Equal(t, 24*time.Hour, task.ExportIntervalDuration(1, "days"))
	assert.Equal(t, time.Duration(0), task.ExportIntervalDuration(5, "unknown"))
}

func TestShouldTrigger_UnknownMode(t *testing.T) {
	entry := &task.ScheduleEntry{Config: task.ScheduleConfig{Mode: "unknown-mode"}}
	assert.False(t, task.ExportShouldTrigger(entry, time.Now()))
}

func TestShouldTrigger_IntervalZeroDuration(t *testing.T) {
	entry := &task.ScheduleEntry{
		Config:  task.ScheduleConfig{Mode: "interval", IntervalValue: 5, IntervalUnit: "invalid"},
		LastRun: time.Now().Add(-10 * time.Hour),
	}
	assert.False(t, task.ExportShouldTrigger(entry, time.Now()))
}
