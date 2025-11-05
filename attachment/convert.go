package attachment

import (
	"fmt"
	"strings"
)

// toBool converts various types to boolean
func toBool(v interface{}) bool {
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
	case uint8: // MySQL tinyint(1)
		return val != 0
	case float64:
		return val != 0
	case string:
		normalized := strings.ToLower(strings.TrimSpace(val))
		switch normalized {
		case "true", "1", "enabled", "yes", "on":
			return true
		default:
			return false
		}
	default:
		return false
	}
}

// toString converts various types to string
func toString(v interface{}) string {
	if v == nil {
		return ""
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
	default:
		return fmt.Sprintf("%v", val)
	}
}
