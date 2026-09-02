package image

import (
	"encoding/json"
	"strconv"
)

// extractExtra returns the "extra" field from allArgs as a map.
// It handles both the LLM path (where extra is already map[string]interface{})
// and the Tai CLI path (where extra is a JSON string).
func extractExtra(allArgs map[string]interface{}) map[string]interface{} {
	v := allArgs["extra"]
	if v == nil {
		return nil
	}
	switch typed := v.(type) {
	case map[string]interface{}:
		return typed
	case string:
		if typed == "" {
			return nil
		}
		var m map[string]interface{}
		if json.Unmarshal([]byte(typed), &m) == nil {
			return m
		}
	}
	return nil
}

// extractIntArg reads an integer argument from allArgs by key.
// It handles float64 (from JSON unmarshal) and string (from CLI flags).
// Returns fallback when the key is absent or not parseable.
func extractIntArg(allArgs map[string]interface{}, key string, fallback int) int {
	v := allArgs[key]
	if v == nil {
		return fallback
	}
	switch typed := v.(type) {
	case float64:
		return int(typed)
	case int:
		return typed
	case string:
		if n, err := strconv.Atoi(typed); err == nil {
			return n
		}
	}
	return fallback
}
