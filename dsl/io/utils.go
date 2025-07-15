package io

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
