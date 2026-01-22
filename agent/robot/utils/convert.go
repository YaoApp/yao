package utils

import (
	"encoding/json"
	"fmt"
	"time"
)

// ==================== To<Type> Functions ====================
// Convert any value to specified type (safe, returns zero value on failure)

// ToString converts any value to string
func ToString(v interface{}) string {
	if v == nil {
		return ""
	}
	switch val := v.(type) {
	case string:
		return val
	case []byte:
		return string(val)
	case int:
		return fmt.Sprintf("%d", val)
	case int8:
		return fmt.Sprintf("%d", val)
	case int16:
		return fmt.Sprintf("%d", val)
	case int32:
		return fmt.Sprintf("%d", val)
	case int64:
		return fmt.Sprintf("%d", val)
	case uint:
		return fmt.Sprintf("%d", val)
	case uint8:
		return fmt.Sprintf("%d", val)
	case uint16:
		return fmt.Sprintf("%d", val)
	case uint32:
		return fmt.Sprintf("%d", val)
	case uint64:
		return fmt.Sprintf("%d", val)
	case float32:
		return fmt.Sprintf("%g", val)
	case float64:
		return fmt.Sprintf("%g", val)
	case bool:
		if val {
			return "true"
		}
		return "false"
	default:
		if str, err := json.Marshal(v); err == nil {
			return string(str)
		}
		return fmt.Sprintf("%v", v)
	}
}

// ToBool converts any value to bool
func ToBool(v interface{}) bool {
	if v == nil {
		return false
	}
	switch b := v.(type) {
	case bool:
		return b
	case int:
		return b != 0
	case int8:
		return b != 0
	case int16:
		return b != 0
	case int32:
		return b != 0
	case int64:
		return b != 0
	case uint:
		return b != 0
	case uint8:
		return b != 0
	case uint16:
		return b != 0
	case uint32:
		return b != 0
	case uint64:
		return b != 0
	case float32:
		return b != 0
	case float64:
		return b != 0
	case string:
		return b == "true" || b == "1" || b == "yes" || b == "on"
	}
	return false
}

// ToInt converts any value to int
func ToInt(v interface{}) int {
	if v == nil {
		return 0
	}
	switch n := v.(type) {
	case int:
		return n
	case int8:
		return int(n)
	case int16:
		return int(n)
	case int32:
		return int(n)
	case int64:
		return int(n)
	case uint:
		return int(n)
	case uint8:
		return int(n)
	case uint16:
		return int(n)
	case uint32:
		return int(n)
	case uint64:
		return int(n)
	case float32:
		return int(n)
	case float64:
		return int(n)
	case string:
		var i int
		fmt.Sscanf(n, "%d", &i)
		return i
	case bool:
		if n {
			return 1
		}
		return 0
	}
	return 0
}

// ToInt64 converts any value to int64
func ToInt64(v interface{}) int64 {
	if v == nil {
		return 0
	}
	switch n := v.(type) {
	case int64:
		return n
	case int:
		return int64(n)
	case int8:
		return int64(n)
	case int16:
		return int64(n)
	case int32:
		return int64(n)
	case uint:
		return int64(n)
	case uint8:
		return int64(n)
	case uint16:
		return int64(n)
	case uint32:
		return int64(n)
	case uint64:
		return int64(n)
	case float32:
		return int64(n)
	case float64:
		return int64(n)
	case string:
		var i int64
		fmt.Sscanf(n, "%d", &i)
		return i
	case bool:
		if n {
			return 1
		}
		return 0
	}
	return 0
}

// ToFloat64 converts any value to float64
func ToFloat64(v interface{}) float64 {
	if v == nil {
		return 0
	}
	switch f := v.(type) {
	case float64:
		return f
	case float32:
		return float64(f)
	case int:
		return float64(f)
	case int8:
		return float64(f)
	case int16:
		return float64(f)
	case int32:
		return float64(f)
	case int64:
		return float64(f)
	case uint:
		return float64(f)
	case uint8:
		return float64(f)
	case uint16:
		return float64(f)
	case uint32:
		return float64(f)
	case uint64:
		return float64(f)
	case string:
		var result float64
		fmt.Sscanf(f, "%f", &result)
		return result
	case bool:
		if f {
			return 1
		}
		return 0
	}
	return 0
}

// ToTimestamp converts any value to *time.Time
// Handles: time.Time, *time.Time, string (various formats), int64/float64 (unix timestamp)
func ToTimestamp(v interface{}) *time.Time {
	if v == nil {
		return nil
	}
	switch t := v.(type) {
	case time.Time:
		return &t
	case *time.Time:
		return t
	case string:
		if t == "" {
			return nil
		}
		// Try common time formats
		formats := []string{
			time.RFC3339,
			time.RFC3339Nano,
			"2006-01-02 15:04:05",
			"2006-01-02T15:04:05Z",
			"2006-01-02T15:04:05",
			"2006-01-02",
		}
		for _, format := range formats {
			if parsed, err := time.Parse(format, t); err == nil {
				return &parsed
			}
		}
	case int64:
		// Unix timestamp (seconds)
		parsed := time.Unix(t, 0)
		return &parsed
	case int:
		parsed := time.Unix(int64(t), 0)
		return &parsed
	case float64:
		// Unix timestamp (seconds as float)
		parsed := time.Unix(int64(t), 0)
		return &parsed
	}
	return nil
}

// ToJSONValue parses JSON from string/[]byte or returns already-parsed value
func ToJSONValue(v interface{}) interface{} {
	if v == nil {
		return nil
	}
	switch data := v.(type) {
	case string:
		if data == "" {
			return nil
		}
		var result interface{}
		if err := json.Unmarshal([]byte(data), &result); err != nil {
			return nil
		}
		return result
	case []byte:
		if len(data) == 0 {
			return nil
		}
		var result interface{}
		if err := json.Unmarshal(data, &result); err != nil {
			return nil
		}
		return result
	case map[string]interface{}, []interface{}:
		// Already parsed
		return data
	default:
		return v
	}
}

// ==================== Get<Type> Functions ====================
// Safely get typed value from map[string]interface{}

// GetString safely gets a string value from map
func GetString(m map[string]interface{}, key string) string {
	if m == nil {
		return ""
	}
	if v, ok := m[key]; ok {
		return ToString(v)
	}
	return ""
}

// GetBool safely gets a bool value from map
func GetBool(m map[string]interface{}, key string) bool {
	if m == nil {
		return false
	}
	if v, ok := m[key]; ok {
		return ToBool(v)
	}
	return false
}

// GetInt safely gets an int value from map
func GetInt(m map[string]interface{}, key string) int {
	if m == nil {
		return 0
	}
	if v, ok := m[key]; ok {
		return ToInt(v)
	}
	return 0
}

// GetInt64 safely gets an int64 value from map
func GetInt64(m map[string]interface{}, key string) int64 {
	if m == nil {
		return 0
	}
	if v, ok := m[key]; ok {
		return ToInt64(v)
	}
	return 0
}

// GetFloat64 safely gets a float64 value from map
func GetFloat64(m map[string]interface{}, key string) float64 {
	if m == nil {
		return 0
	}
	if v, ok := m[key]; ok {
		return ToFloat64(v)
	}
	return 0
}

// GetTimestamp safely gets a *time.Time value from map
func GetTimestamp(m map[string]interface{}, key string) *time.Time {
	if m == nil {
		return nil
	}
	if v, ok := m[key]; ok {
		return ToTimestamp(v)
	}
	return nil
}

// GetJSONValue safely gets a parsed JSON value from map
func GetJSONValue(m map[string]interface{}, key string) interface{} {
	if m == nil {
		return nil
	}
	if v, ok := m[key]; ok {
		return ToJSONValue(v)
	}
	return nil
}

// ==================== JSON/Map Conversion ====================

// ToJSON converts any value to JSON string
func ToJSON(v interface{}) (string, error) {
	data, err := json.Marshal(v)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// FromJSON parses JSON string to target struct
func FromJSON(jsonStr string, target interface{}) error {
	return json.Unmarshal([]byte(jsonStr), target)
}

// ToMap converts struct to map[string]interface{}
func ToMap(v interface{}) (map[string]interface{}, error) {
	data, err := json.Marshal(v)
	if err != nil {
		return nil, err
	}
	var result map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, err
	}
	return result, nil
}

// FromMap converts map to struct
func FromMap(m map[string]interface{}, target interface{}) error {
	data, err := json.Marshal(m)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, target)
}

// ==================== Map Utilities ====================

// MergeMap merges source map into target map (shallow copy)
func MergeMap(target, source map[string]interface{}) map[string]interface{} {
	if target == nil {
		target = make(map[string]interface{})
	}
	for k, v := range source {
		target[k] = v
	}
	return target
}

// CloneMap creates a shallow copy of a map
func CloneMap(m map[string]interface{}) map[string]interface{} {
	if m == nil {
		return nil
	}
	result := make(map[string]interface{}, len(m))
	for k, v := range m {
		result[k] = v
	}
	return result
}
