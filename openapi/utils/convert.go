package utils

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

// Type Conversion Utilities
// These functions provide safe type conversion from interface{} to common types

// ToBool converts various types to boolean
// Supports: bool, int, int64, float64, string
// String values: "true", "false", "1", "0", "enabled", "disabled", "yes", "no", "on", "off"
// Returns false for nil or unsupported types
func ToBool(v interface{}) bool {
	if v == nil {
		return false
	}

	switch val := v.(type) {
	case bool:
		return val
	case int:
		return val != 0
	case int64:
		return val != 0
	case float64:
		return val != 0
	case string:
		// Normalize string to lowercase for case-insensitive comparison
		normalized := strings.ToLower(strings.TrimSpace(val))
		switch normalized {
		case "true", "1", "enabled", "yes", "on":
			return true
		case "false", "0", "disabled", "no", "off", "":
			return false
		default:
			return false
		}
	default:
		return false
	}
}

// ToString converts various types to string
// Supports: string, int, int64, float64, bool, time.Time, *time.Time
// time.Time is formatted using the optional timeFormat parameter
// If timeFormat is not provided, defaults to "2006-01-02 15:04:05"
// Returns empty string for nil or unsupported types
func ToString(v interface{}, timeFormat ...string) string {
	if v == nil {
		return ""
	}

	// Get time format (default or provided)
	format := "2006-01-02 15:04:05"
	if len(timeFormat) > 0 && timeFormat[0] != "" {
		format = timeFormat[0]
	}

	switch val := v.(type) {
	case string:
		return val
	case int:
		return fmt.Sprintf("%d", val)
	case int64:
		return fmt.Sprintf("%d", val)
	case float64:
		return fmt.Sprintf("%.0f", val)
	case bool:
		if val {
			return "true"
		}
		return "false"
	case time.Time:
		return val.Format(format)
	case *time.Time:
		if val != nil {
			return val.Format(format)
		}
		return ""
	default:
		return ""
	}
}

// ToInt64 converts various types to int64
// Supports: int, int64, float64, string
// Returns 0 for nil or unsupported types
func ToInt64(v interface{}) int64 {
	if v == nil {
		return 0
	}

	switch val := v.(type) {
	case int64:
		return val
	case int:
		return int64(val)
	case float64:
		return int64(val)
	case string:
		if parsed, err := strconv.ParseInt(val, 10, 64); err == nil {
			return parsed
		}
		return 0
	default:
		return 0
	}
}

// ToInt converts various types to int
// Supports: int, int64, float64, string
// Returns 0 for nil or unsupported types
func ToInt(v interface{}) int {
	if v == nil {
		return 0
	}

	switch val := v.(type) {
	case int:
		return val
	case int64:
		return int(val)
	case float64:
		return int(val)
	case string:
		if parsed, err := strconv.Atoi(val); err == nil {
			return parsed
		}
		return 0
	default:
		return 0
	}
}

// ToFloat64 converts various types to float64
// Supports: float64, int, int64, string
// Returns 0.0 for nil or unsupported types
func ToFloat64(v interface{}) float64 {
	if v == nil {
		return 0.0
	}

	switch val := v.(type) {
	case float64:
		return val
	case float32:
		return float64(val)
	case int:
		return float64(val)
	case int64:
		return float64(val)
	case string:
		if parsed, err := strconv.ParseFloat(val, 64); err == nil {
			return parsed
		}
		return 0.0
	default:
		return 0.0
	}
}

// ToTimeString converts various time types to RFC3339 string
// Supports: time.Time, string, int64 (unix timestamp)
// Returns empty string for nil or unsupported types
func ToTimeString(v interface{}) string {
	if v == nil {
		return ""
	}

	switch val := v.(type) {
	case time.Time:
		if val.IsZero() {
			return ""
		}
		return val.Format(time.RFC3339)
	case string:
		// Try to parse as RFC3339 first
		if t, err := time.Parse(time.RFC3339, val); err == nil {
			return t.Format(time.RFC3339)
		}
		// Try to parse as other common formats
		formats := []string{
			"2006-01-02 15:04:05",
			"2006-01-02T15:04:05Z",
			"2006-01-02T15:04:05.000Z",
		}
		for _, format := range formats {
			if t, err := time.Parse(format, val); err == nil {
				return t.Format(time.RFC3339)
			}
		}
		return val // Return as-is if can't parse
	case int64:
		// Assume unix timestamp
		if val > 0 {
			return time.Unix(val, 0).Format(time.RFC3339)
		}
		return ""
	default:
		return ""
	}
}

// GetTimeFormat returns the appropriate time format string for the given locale
// Returns format suitable for time.Format()
func GetTimeFormat(locale string) string {
	// Normalize locale to lowercase
	locale = strings.ToLower(strings.TrimSpace(locale))

	switch locale {
	case "zh-cn", "zh":
		// Chinese format: 2025年10月30日 08:57:51
		return "2006年01月02日 15:04:05"
	case "en", "en-us", "":
		// English format: October 30, 2025 08:57:51
		return "January 02, 2006 15:04:05"
	default:
		// Default ISO format
		return "2006-01-02 15:04:05"
	}
}

// FormatTimeWithLocale formats a time value (time.Time, *time.Time, or string) using the specified format
// If the input is already a string, it will parse it first and then reformat it
// Returns empty string if the value cannot be parsed
func FormatTimeWithLocale(v interface{}, targetFormat string) string {
	if v == nil {
		return ""
	}

	var t time.Time
	var err error

	switch val := v.(type) {
	case time.Time:
		t = val
	case *time.Time:
		if val != nil {
			t = *val
		} else {
			return ""
		}
	case string:
		// Try parsing with common formats
		formats := []string{
			"2006-01-02 15:04:05",
			"2006-01-02T15:04:05Z",
			"2006-01-02T15:04:05",
			time.RFC3339,
		}
		for _, format := range formats {
			t, err = time.Parse(format, val)
			if err == nil {
				break
			}
		}
		if err != nil {
			// If all parsing attempts failed, return the original string
			return val
		}
	default:
		// For unsupported types, try ToString first
		str := ToString(v)
		if str == "" {
			return ""
		}
		// Try parsing the string
		return FormatTimeWithLocale(str, targetFormat)
	}

	// Format with target format
	return t.Format(targetFormat)
}
