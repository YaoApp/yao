package trigger

import (
	"time"

	"github.com/yaoapp/yao/agent/robot/types"
)

// ClockMatcher provides clock trigger matching logic
// This is extracted from Manager for reuse and testing
type ClockMatcher struct{}

// NewClockMatcher creates a new clock matcher
func NewClockMatcher() *ClockMatcher {
	return &ClockMatcher{}
}

// ShouldTrigger checks if a robot should be triggered based on its clock config
func (cm *ClockMatcher) ShouldTrigger(robot *types.Robot, now time.Time) bool {
	if robot == nil || robot.Config == nil || robot.Config.Clock == nil {
		return false
	}

	clock := robot.Config.Clock

	// Get time in robot's timezone
	loc := clock.GetLocation()
	localNow := now.In(loc)

	switch clock.Mode {
	case types.ClockTimes:
		return cm.shouldTriggerTimes(robot, clock, localNow)
	case types.ClockInterval:
		return cm.shouldTriggerInterval(robot, clock, localNow)
	case types.ClockDaemon:
		return cm.shouldTriggerDaemon(robot, clock, localNow)
	default:
		return false
	}
}

// shouldTriggerTimes checks if current time matches any configured times
// times mode: run at specific times (e.g., ["09:00", "14:00", "17:00"])
func (cm *ClockMatcher) shouldTriggerTimes(robot *types.Robot, clock *types.Clock, now time.Time) bool {
	// Check day of week first
	if !cm.matchesDay(clock, now) {
		return false
	}

	// Check if current time matches any configured time
	currentTime := now.Format("15:04")
	for _, t := range clock.Times {
		if t == currentTime {
			// Check if already triggered in this minute
			if !robot.LastRun.IsZero() {
				lastRunInLoc := robot.LastRun.In(now.Location())
				if lastRunInLoc.Format("15:04") == currentTime && lastRunInLoc.Day() == now.Day() {
					return false // Already triggered this minute today
				}
			}
			return true
		}
	}
	return false
}

// shouldTriggerInterval checks if enough time has passed since last run
// interval mode: run every X duration (e.g., "30m", "2h")
func (cm *ClockMatcher) shouldTriggerInterval(robot *types.Robot, clock *types.Clock, now time.Time) bool {
	interval, err := time.ParseDuration(clock.Every)
	if err != nil {
		return false
	}

	// First run if never executed
	if robot.LastRun.IsZero() {
		return true
	}

	// Check if interval has passed
	return now.Sub(robot.LastRun) >= interval
}

// shouldTriggerDaemon checks if robot can restart immediately after last run
// daemon mode: restart immediately after each run completes
func (cm *ClockMatcher) shouldTriggerDaemon(robot *types.Robot, clock *types.Clock, now time.Time) bool {
	// Daemon mode: trigger if not currently running
	// CanRun() checks if robot has available execution slots
	return robot.CanRun()
}

// matchesDay checks if current day matches the configured days
func (cm *ClockMatcher) matchesDay(clock *types.Clock, now time.Time) bool {
	// Empty days or ["*"] means all days
	if len(clock.Days) == 0 {
		return true
	}

	for _, day := range clock.Days {
		if day == "*" {
			return true
		}
		// Match day name (Mon, Tue, Wed, Thu, Fri, Sat, Sun)
		// or full name (Monday, Tuesday, etc.)
		weekday := now.Weekday().String()
		shortDay := weekday[:3] // Mon, Tue, etc.
		if day == weekday || day == shortDay {
			return true
		}
	}
	return false
}

// ParseTime parses a time string in "HH:MM" format
func ParseTime(timeStr string) (hour, minute int, err error) {
	t, err := time.Parse("15:04", timeStr)
	if err != nil {
		return 0, 0, err
	}
	return t.Hour(), t.Minute(), nil
}

// FormatTime formats hour and minute to "HH:MM" string
func FormatTime(hour, minute int) string {
	return time.Date(0, 1, 1, hour, minute, 0, 0, time.UTC).Format("15:04")
}
