package role

import "fmt"

// ============================================================================
// Type Conversion Utilities
// ============================================================================

// toString converts the value to a string
func toString(value interface{}) string {
	switch v := value.(type) {
	case string:
		return v
	case []byte:
		return string(v)
	default:
		return fmt.Sprintf("%v", v)
	}
}

// toStringArray converts various types to a string slice
func toStringArray(value interface{}) []string {
	switch v := value.(type) {
	case []string:
		return v
	case []interface{}:
		result := []string{}
		for _, v := range v {
			result = append(result, toString(v))
		}
		return result
	default:
		return []string{}
	}
}

// ============================================================================
// Permission Format Utilities
// ============================================================================

// formatPermissions converts various permission formats to a string slice
// Supports: []string, []interface{}, map[string]interface{}, map[string]bool, string
func formatPermissions(value interface{}) ([]string, error) {
	if value == nil {
		return []string{}, nil
	}

	switch v := value.(type) {
	case []string:
		// Direct string slice
		return v, nil

	case []interface{}:
		// Interface slice - convert each element
		result := make([]string, 0, len(v))
		for i, item := range v {
			switch itemVal := item.(type) {
			case string:
				result = append(result, itemVal)
			case []byte:
				result = append(result, string(itemVal))
			default:
				return nil, fmt.Errorf("item at index %d has unsupported type %T", i, item)
			}
		}
		return result, nil

	case map[string]interface{}:
		// Map with interface{} values - extract keys where value is truthy
		result := make([]string, 0, len(v))
		for key, val := range v {
			// Include if value is truthy
			if isTrue(val) {
				result = append(result, key)
			}
		}
		return result, nil

	case map[string]bool:
		// Map with bool values - extract keys where value is true
		result := make([]string, 0, len(v))
		for key, enabled := range v {
			if enabled {
				result = append(result, key)
			}
		}
		return result, nil

	case string:
		// Single string - return as single-element slice
		if v == "" {
			return []string{}, nil
		}
		return []string{v}, nil

	case []byte:
		// Byte slice - convert to string
		str := string(v)
		if str == "" {
			return []string{}, nil
		}
		return []string{str}, nil

	default:
		return nil, fmt.Errorf("unsupported permissions type: %T", value)
	}
}

// isTrue checks if a value is truthy
func isTrue(value interface{}) bool {
	if value == nil {
		return false
	}

	switch v := value.(type) {
	case bool:
		return v
	case int, int8, int16, int32, int64:
		return v != 0
	case uint, uint8, uint16, uint32, uint64:
		return v != 0
	case float32, float64:
		return v != 0
	case string:
		return v != "" && v != "false" && v != "0"
	default:
		return true // Non-nil, non-false values are considered truthy
	}
}
