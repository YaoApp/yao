package xun

import (
	"fmt"
	"reflect"
	"time"

	jsoniter "github.com/json-iterator/go"
	"github.com/yaoapp/yao/openapi/utils"
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
	return utils.ToBool(data[key])
}

func getInt(data map[string]interface{}, key string) int {
	return utils.ToInt(data[key])
}

func getInt64(data map[string]interface{}, key string) int64 {
	v := data[key]
	if t, ok := v.(time.Time); ok {
		return t.UnixNano()
	}
	return utils.ToInt64(v)
}

// toDBTime converts UnixNano timestamp to database BIGINT format
func toDBTime(unixNano int64) int64 {
	if unixNano == 0 {
		return 0
	}
	return unixNano
}

// fromDBTime converts database BIGINT timestamp to UnixNano
func fromDBTime(dbTime int64) int64 {
	return dbTime
}
