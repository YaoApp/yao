//go:build unit

package task_test

import (
	"encoding/json"
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

func TestParseScheduleConfig_StringInput(t *testing.T) {
	input := `{"enabled":true,"mode":"interval","interval_value":30,"interval_unit":"minutes"}`
	cfg := task.ExportParseScheduleConfig(input)
	assert.True(t, cfg.Enabled)
	assert.Equal(t, "interval", cfg.Mode)
	assert.Equal(t, 30, cfg.IntervalValue)
	assert.Equal(t, "minutes", cfg.IntervalUnit)
}

func TestParseScheduleConfig_MapInput(t *testing.T) {
	input := map[string]interface{}{
		"enabled":        true,
		"mode":           "times",
		"times":          []interface{}{"09:00", "18:00"},
		"interval_value": float64(0),
	}
	cfg := task.ExportParseScheduleConfig(input)
	assert.True(t, cfg.Enabled)
	assert.Equal(t, "times", cfg.Mode)
	assert.Equal(t, []string{"09:00", "18:00"}, cfg.Times)
}

func TestParseScheduleConfig_NilInput(t *testing.T) {
	cfg := task.ExportParseScheduleConfig(nil)
	assert.False(t, cfg.Enabled)
	assert.Empty(t, cfg.Mode)
}

func TestParseScheduleJSON_StringInput(t *testing.T) {
	input := `{"enabled":true,"mode":"times","times":["08:00"],"timezone":"Asia/Shanghai"}`
	cfg := task.ExportParseScheduleJSON(input)
	assert.NotNil(t, cfg)
	assert.True(t, cfg.Enabled)
	assert.Equal(t, "times", cfg.Mode)
	assert.Equal(t, []string{"08:00"}, cfg.Times)
	assert.Equal(t, "Asia/Shanghai", cfg.Timezone)
}

func TestParseScheduleJSON_NilInput(t *testing.T) {
	cfg := task.ExportParseScheduleJSON(nil)
	assert.Nil(t, cfg)
}

func TestParseScheduleJSON_EmptyString(t *testing.T) {
	cfg := task.ExportParseScheduleJSON("")
	assert.Nil(t, cfg)
}

func TestTask_Schedule_Serialization(t *testing.T) {
	tk := task.Task{
		ChatID: "chat-sched-001",
		Schedule: &task.ScheduleConfig{
			Enabled:       true,
			Mode:          "interval",
			IntervalValue: 30,
			IntervalUnit:  "minutes",
			Timezone:      "Asia/Shanghai",
		},
		Instruction: &task.ScheduledInstruction{
			Prompt: "Run daily report",
			Locale: "zh-cn",
		},
	}

	data, err := json.Marshal(tk)
	assert.NoError(t, err)

	var decoded task.Task
	err = json.Unmarshal(data, &decoded)
	assert.NoError(t, err)
	assert.NotNil(t, decoded.Schedule)
	assert.True(t, decoded.Schedule.Enabled)
	assert.Equal(t, "interval", decoded.Schedule.Mode)
	assert.Equal(t, 30, decoded.Schedule.IntervalValue)
	assert.Equal(t, "minutes", decoded.Schedule.IntervalUnit)
}

func TestTask_NoSchedule_OmitEmpty(t *testing.T) {
	tk := task.Task{ChatID: "chat-no-sched"}
	data, err := json.Marshal(tk)
	assert.NoError(t, err)
	assert.NotContains(t, string(data), "schedule")
	assert.NotContains(t, string(data), "next_run")
}
