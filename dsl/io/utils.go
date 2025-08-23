package io

import "time"

// toBool converts various types to boolean
func toBool(v interface{}) bool {
	if v == nil {
		return false
	}

	switch val := v.(type) {
	case bool:
		return val
	case int:
		return val == 1
	case int64:
		return val == 1
	case float64:
		return val == 1
	case string:
		return val == "1" || val == "true"
	default:
		return false
	}
}

// toTime converts various time formats to RFC3339 string
func toTime(v interface{}) string {
	if v == nil {
		return ""
	}

	switch val := v.(type) {
	case string:
		// Try common formats
		formats := []string{
			"2006-01-02 15:04:05",       // SQLite format
			"2006-01-02T15:04:05Z07:00", // RFC3339 format
			"2006-01-02T15:04:05Z",      // RFC3339 without timezone
			time.RFC3339,
		}
		for _, format := range formats {
			if t, err := time.Parse(format, val); err == nil {
				return t.UTC().Format(time.RFC3339) // Convert to UTC and format as RFC3339
			}
		}
	case time.Time:
		return val.UTC().Format(time.RFC3339) // Convert to UTC and format as RFC3339
	}
	return ""
}
