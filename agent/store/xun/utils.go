package xun

import (
	"fmt"
	"reflect"
	"time"

	jsoniter "github.com/json-iterator/go"
)

// isNil checks whether a value is truly nil, handling the Go typed-nil-in-interface pitfall.
// A nil map, slice, or pointer stored in an interface{} is not == nil in Go;
// this helper uses reflect to detect that case.
func isNil(v interface{}) bool {
	if v == nil {
		return true
	}
	rv := reflect.ValueOf(v)
	switch rv.Kind() {
	case reflect.Ptr, reflect.Map, reflect.Slice, reflect.Interface, reflect.Chan, reflect.Func:
		return rv.IsNil()
	}
	return false
}

// marshalJSONFields serialises each value in fields to a JSON string and writes
// it into data. Truly-nil values (including typed nils) are skipped so the
// database column keeps its SQL NULL / default.
func marshalJSONFields(data map[string]interface{}, fields map[string]interface{}) error {
	for field, value := range fields {
		if isNil(value) {
			continue
		}
		jsonStr, err := jsoniter.MarshalToString(value)
		if err != nil {
			return fmt.Errorf("failed to marshal %s: %w", field, err)
		}
		data[field] = jsonStr
	}
	return nil
}

// Helper functions for type conversion
func getString(data map[string]interface{}, key string) string {
	if v, ok := data[key].(string); ok {
		return v
	}
	return ""
}

func getBool(data map[string]interface{}, key string) bool {
	switch v := data[key].(type) {
	case bool:
		return v
	case int64:
		return v != 0
	case int:
		return v != 0
	case float64:
		return v != 0
	}
	return false
}

func getInt(data map[string]interface{}, key string) int {
	switch v := data[key].(type) {
	case int:
		return v
	case int64:
		return int(v)
	case float64:
		return int(v)
	}
	return 0
}

func getInt64(data map[string]interface{}, key string) int64 {
	switch v := data[key].(type) {
	case int64:
		return v
	case int:
		return int64(v)
	case float64:
		return int64(v)
	case string:
		// Handle string representation of numbers (common with MySQL BIGINT)
		var result int64
		if _, err := fmt.Sscanf(v, "%d", &result); err == nil {
			return result
		}
	case time.Time:
		// Handle time.Time from database
		return v.UnixNano()
	}
	return 0
}

// toMySQLTime converts UnixNano timestamp to MySQL BIGINT format
func toMySQLTime(unixNano int64) int64 {
	if unixNano == 0 {
		return 0
	}
	return unixNano
}

// fromMySQLTime converts MySQL BIGINT timestamp to UnixNano
func fromMySQLTime(mysqlTime int64) int64 {
	return mysqlTime
}
