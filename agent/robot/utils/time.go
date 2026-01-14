package utils

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

// ParseTime parses a time string in HH:MM format
func ParseTime(timeStr string) (hour, minute int, err error) {
	parts := strings.Split(timeStr, ":")
	if len(parts) != 2 {
		return 0, 0, fmt.Errorf("invalid time format: %s (expected HH:MM)", timeStr)
	}

	hour, err = strconv.Atoi(parts[0])
	if err != nil || hour < 0 || hour > 23 {
		return 0, 0, fmt.Errorf("invalid hour: %s", parts[0])
	}

	minute, err = strconv.Atoi(parts[1])
	if err != nil || minute < 0 || minute > 59 {
		return 0, 0, fmt.Errorf("invalid minute: %s", parts[1])
	}

	return hour, minute, nil
}

// FormatTime formats hour and minute into HH:MM format
func FormatTime(hour, minute int) string {
	return fmt.Sprintf("%02d:%02d", hour, minute)
}

// LoadLocation loads a timezone location, returns Local if empty or invalid
func LoadLocation(tz string) *time.Location {
	if tz == "" {
		return time.Local
	}
	loc, err := time.LoadLocation(tz)
	if err != nil {
		return time.Local
	}
	return loc
}

// ParseDuration parses a duration string with fallback default
func ParseDuration(durStr string, defaultDur time.Duration) time.Duration {
	if durStr == "" {
		return defaultDur
	}
	d, err := time.ParseDuration(durStr)
	if err != nil {
		return defaultDur
	}
	return d
}

// IsTimeMatch checks if current time matches the specified time (HH:MM)
func IsTimeMatch(now time.Time, timeStr string, loc *time.Location) bool {
	hour, minute, err := ParseTime(timeStr)
	if err != nil {
		return false
	}

	nowInLoc := now.In(loc)
	return nowInLoc.Hour() == hour && nowInLoc.Minute() == minute
}

// IsDayMatch checks if current day matches the specified day
// days can be: "Mon", "Tue", "Wed", "Thu", "Fri", "Sat", "Sun", or "*" for any day
func IsDayMatch(now time.Time, days []string) bool {
	if len(days) == 0 {
		return true
	}

	dayName := now.Weekday().String()[:3] // "Monday" -> "Mon"

	for _, day := range days {
		if day == "*" || day == dayName {
			return true
		}
	}
	return false
}

// NextScheduledTime calculates the next time a scheduled time will occur
func NextScheduledTime(now time.Time, timeStr string, days []string, loc *time.Location) (time.Time, error) {
	hour, minute, err := ParseTime(timeStr)
	if err != nil {
		return time.Time{}, err
	}

	nowInLoc := now.In(loc)

	// Start from today at the specified time
	next := time.Date(nowInLoc.Year(), nowInLoc.Month(), nowInLoc.Day(), hour, minute, 0, 0, loc)

	// If the time has passed today, start from tomorrow
	if next.Before(nowInLoc) || next.Equal(nowInLoc) {
		next = next.Add(24 * time.Hour)
	}

	// Find the next matching day (within 7 days)
	for i := 0; i < 7; i++ {
		if IsDayMatch(next, days) {
			return next, nil
		}
		next = next.Add(24 * time.Hour)
	}

	// If no matching day found (should not happen with valid days), return the calculated time
	return next, nil
}
