package utils_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/yaoapp/yao/agent/robot/utils"
)

// ==================== To<Type> Tests ====================

func TestToBool(t *testing.T) {
	t.Run("from_bool", func(t *testing.T) {
		assert.True(t, utils.ToBool(true))
		assert.False(t, utils.ToBool(false))
	})

	t.Run("from_int", func(t *testing.T) {
		assert.True(t, utils.ToBool(1))
		assert.True(t, utils.ToBool(42))
		assert.False(t, utils.ToBool(0))
	})

	t.Run("from_int64", func(t *testing.T) {
		assert.True(t, utils.ToBool(int64(1)))
		assert.False(t, utils.ToBool(int64(0)))
	})

	t.Run("from_float64", func(t *testing.T) {
		assert.True(t, utils.ToBool(1.0))
		assert.True(t, utils.ToBool(0.1))
		assert.False(t, utils.ToBool(0.0))
	})

	t.Run("from_string", func(t *testing.T) {
		assert.True(t, utils.ToBool("true"))
		assert.True(t, utils.ToBool("1"))
		assert.True(t, utils.ToBool("yes"))
		assert.True(t, utils.ToBool("on"))
		assert.False(t, utils.ToBool("false"))
		assert.False(t, utils.ToBool("0"))
		assert.False(t, utils.ToBool(""))
	})

	t.Run("from_nil", func(t *testing.T) {
		assert.False(t, utils.ToBool(nil))
	})

	t.Run("from_unsupported_type", func(t *testing.T) {
		assert.False(t, utils.ToBool([]int{1, 2, 3}))
	})
}

func TestToInt(t *testing.T) {
	t.Run("from_int", func(t *testing.T) {
		assert.Equal(t, 42, utils.ToInt(42))
		assert.Equal(t, -10, utils.ToInt(-10))
	})

	t.Run("from_int64", func(t *testing.T) {
		assert.Equal(t, 100, utils.ToInt(int64(100)))
	})

	t.Run("from_float64", func(t *testing.T) {
		assert.Equal(t, 42, utils.ToInt(42.9)) // truncates
		assert.Equal(t, -5, utils.ToInt(-5.7))
	})

	t.Run("from_string", func(t *testing.T) {
		assert.Equal(t, 123, utils.ToInt("123"))
		assert.Equal(t, -456, utils.ToInt("-456"))
		assert.Equal(t, 0, utils.ToInt("invalid"))
	})

	t.Run("from_bool", func(t *testing.T) {
		assert.Equal(t, 1, utils.ToInt(true))
		assert.Equal(t, 0, utils.ToInt(false))
	})

	t.Run("from_nil", func(t *testing.T) {
		assert.Equal(t, 0, utils.ToInt(nil))
	})
}

func TestToInt64(t *testing.T) {
	t.Run("from_int64", func(t *testing.T) {
		assert.Equal(t, int64(9223372036854775807), utils.ToInt64(int64(9223372036854775807)))
	})

	t.Run("from_int", func(t *testing.T) {
		assert.Equal(t, int64(42), utils.ToInt64(42))
	})

	t.Run("from_float64", func(t *testing.T) {
		assert.Equal(t, int64(42), utils.ToInt64(42.9))
	})

	t.Run("from_string", func(t *testing.T) {
		assert.Equal(t, int64(123456789), utils.ToInt64("123456789"))
	})

	t.Run("from_nil", func(t *testing.T) {
		assert.Equal(t, int64(0), utils.ToInt64(nil))
	})
}

func TestToFloat64(t *testing.T) {
	t.Run("from_float64", func(t *testing.T) {
		assert.Equal(t, 3.14159, utils.ToFloat64(3.14159))
	})

	t.Run("from_float32", func(t *testing.T) {
		assert.InDelta(t, 3.14, utils.ToFloat64(float32(3.14)), 0.001)
	})

	t.Run("from_int", func(t *testing.T) {
		assert.Equal(t, 42.0, utils.ToFloat64(42))
	})

	t.Run("from_int64", func(t *testing.T) {
		assert.Equal(t, 100.0, utils.ToFloat64(int64(100)))
	})

	t.Run("from_string", func(t *testing.T) {
		assert.InDelta(t, 3.14, utils.ToFloat64("3.14"), 0.001)
		assert.Equal(t, 0.0, utils.ToFloat64("invalid"))
	})

	t.Run("from_bool", func(t *testing.T) {
		assert.Equal(t, 1.0, utils.ToFloat64(true))
		assert.Equal(t, 0.0, utils.ToFloat64(false))
	})

	t.Run("from_nil", func(t *testing.T) {
		assert.Equal(t, 0.0, utils.ToFloat64(nil))
	})
}

func TestToTimestamp(t *testing.T) {
	t.Run("from_time_Time", func(t *testing.T) {
		now := time.Now()
		result := utils.ToTimestamp(now)
		assert.NotNil(t, result)
		assert.Equal(t, now.Unix(), result.Unix())
	})

	t.Run("from_time_Time_pointer", func(t *testing.T) {
		now := time.Now()
		result := utils.ToTimestamp(&now)
		assert.NotNil(t, result)
		assert.Equal(t, now.Unix(), result.Unix())
	})

	t.Run("from_RFC3339_string", func(t *testing.T) {
		result := utils.ToTimestamp("2024-01-15T14:30:00Z")
		assert.NotNil(t, result)
		assert.Equal(t, 2024, result.Year())
		assert.Equal(t, time.January, result.Month())
		assert.Equal(t, 15, result.Day())
		assert.Equal(t, 14, result.Hour())
		assert.Equal(t, 30, result.Minute())
	})

	t.Run("from_datetime_string", func(t *testing.T) {
		result := utils.ToTimestamp("2024-01-15 14:30:00")
		assert.NotNil(t, result)
		assert.Equal(t, 2024, result.Year())
	})

	t.Run("from_date_string", func(t *testing.T) {
		result := utils.ToTimestamp("2024-01-15")
		assert.NotNil(t, result)
		assert.Equal(t, 2024, result.Year())
		assert.Equal(t, 15, result.Day())
	})

	t.Run("from_unix_timestamp_int64", func(t *testing.T) {
		// 2024-01-15 00:00:00 UTC
		result := utils.ToTimestamp(int64(1705276800))
		assert.NotNil(t, result)
		assert.Equal(t, 2024, result.Year())
	})

	t.Run("from_unix_timestamp_float64", func(t *testing.T) {
		result := utils.ToTimestamp(float64(1705276800))
		assert.NotNil(t, result)
		assert.Equal(t, 2024, result.Year())
	})

	t.Run("from_empty_string", func(t *testing.T) {
		result := utils.ToTimestamp("")
		assert.Nil(t, result)
	})

	t.Run("from_invalid_string", func(t *testing.T) {
		result := utils.ToTimestamp("not a date")
		assert.Nil(t, result)
	})

	t.Run("from_nil", func(t *testing.T) {
		result := utils.ToTimestamp(nil)
		assert.Nil(t, result)
	})
}

func TestToJSONValue(t *testing.T) {
	t.Run("from_json_string_object", func(t *testing.T) {
		result := utils.ToJSONValue(`{"name":"test","age":30}`)
		assert.NotNil(t, result)
		m, ok := result.(map[string]interface{})
		assert.True(t, ok)
		assert.Equal(t, "test", m["name"])
		assert.Equal(t, float64(30), m["age"])
	})

	t.Run("from_json_string_array", func(t *testing.T) {
		result := utils.ToJSONValue(`["a","b","c"]`)
		assert.NotNil(t, result)
		arr, ok := result.([]interface{})
		assert.True(t, ok)
		assert.Len(t, arr, 3)
		assert.Equal(t, "a", arr[0])
	})

	t.Run("from_bytes", func(t *testing.T) {
		result := utils.ToJSONValue([]byte(`{"key":"value"}`))
		assert.NotNil(t, result)
		m, ok := result.(map[string]interface{})
		assert.True(t, ok)
		assert.Equal(t, "value", m["key"])
	})

	t.Run("from_already_parsed_map", func(t *testing.T) {
		input := map[string]interface{}{"foo": "bar"}
		result := utils.ToJSONValue(input)
		assert.Equal(t, input, result)
	})

	t.Run("from_already_parsed_array", func(t *testing.T) {
		input := []interface{}{"a", "b"}
		result := utils.ToJSONValue(input)
		assert.Equal(t, input, result)
	})

	t.Run("from_empty_string", func(t *testing.T) {
		result := utils.ToJSONValue("")
		assert.Nil(t, result)
	})

	t.Run("from_empty_bytes", func(t *testing.T) {
		result := utils.ToJSONValue([]byte{})
		assert.Nil(t, result)
	})

	t.Run("from_invalid_json", func(t *testing.T) {
		result := utils.ToJSONValue("not json")
		assert.Nil(t, result)
	})

	t.Run("from_nil", func(t *testing.T) {
		result := utils.ToJSONValue(nil)
		assert.Nil(t, result)
	})

	t.Run("from_other_type_passthrough", func(t *testing.T) {
		// Non-string, non-[]byte types are passed through
		result := utils.ToJSONValue(42)
		assert.Equal(t, 42, result)
	})
}

// ==================== Get<Type> Tests ====================

func TestGetString(t *testing.T) {
	m := map[string]interface{}{
		"name":   "test",
		"number": 42,
		"bool":   true,
		"nil":    nil,
	}

	t.Run("existing_string_key", func(t *testing.T) {
		assert.Equal(t, "test", utils.GetString(m, "name"))
	})

	t.Run("converts_number_to_string", func(t *testing.T) {
		assert.Equal(t, "42", utils.GetString(m, "number"))
	})

	t.Run("converts_bool_to_string", func(t *testing.T) {
		assert.Equal(t, "true", utils.GetString(m, "bool"))
	})

	t.Run("non_existent_key", func(t *testing.T) {
		assert.Equal(t, "", utils.GetString(m, "missing"))
	})

	t.Run("nil_map", func(t *testing.T) {
		assert.Equal(t, "", utils.GetString(nil, "key"))
	})

	t.Run("nil_value", func(t *testing.T) {
		assert.Equal(t, "", utils.GetString(m, "nil"))
	})
}

func TestGetBool(t *testing.T) {
	m := map[string]interface{}{
		"bool_true":   true,
		"bool_false":  false,
		"int_one":     1,
		"int_zero":    0,
		"string_true": "true",
	}

	t.Run("bool_true", func(t *testing.T) {
		assert.True(t, utils.GetBool(m, "bool_true"))
	})

	t.Run("bool_false", func(t *testing.T) {
		assert.False(t, utils.GetBool(m, "bool_false"))
	})

	t.Run("int_one", func(t *testing.T) {
		assert.True(t, utils.GetBool(m, "int_one"))
	})

	t.Run("int_zero", func(t *testing.T) {
		assert.False(t, utils.GetBool(m, "int_zero"))
	})

	t.Run("string_true", func(t *testing.T) {
		assert.True(t, utils.GetBool(m, "string_true"))
	})

	t.Run("non_existent_key", func(t *testing.T) {
		assert.False(t, utils.GetBool(m, "missing"))
	})

	t.Run("nil_map", func(t *testing.T) {
		assert.False(t, utils.GetBool(nil, "key"))
	})
}

func TestGetInt(t *testing.T) {
	m := map[string]interface{}{
		"int":     42,
		"int64":   int64(100),
		"float64": 3.14,
		"string":  "123",
	}

	t.Run("int", func(t *testing.T) {
		assert.Equal(t, 42, utils.GetInt(m, "int"))
	})

	t.Run("int64", func(t *testing.T) {
		assert.Equal(t, 100, utils.GetInt(m, "int64"))
	})

	t.Run("float64", func(t *testing.T) {
		assert.Equal(t, 3, utils.GetInt(m, "float64"))
	})

	t.Run("string", func(t *testing.T) {
		assert.Equal(t, 123, utils.GetInt(m, "string"))
	})

	t.Run("non_existent_key", func(t *testing.T) {
		assert.Equal(t, 0, utils.GetInt(m, "missing"))
	})

	t.Run("nil_map", func(t *testing.T) {
		assert.Equal(t, 0, utils.GetInt(nil, "key"))
	})
}

func TestGetInt64(t *testing.T) {
	m := map[string]interface{}{
		"int64":  int64(9223372036854775807),
		"int":    42,
		"string": "123456789",
	}

	t.Run("int64", func(t *testing.T) {
		assert.Equal(t, int64(9223372036854775807), utils.GetInt64(m, "int64"))
	})

	t.Run("int", func(t *testing.T) {
		assert.Equal(t, int64(42), utils.GetInt64(m, "int"))
	})

	t.Run("string", func(t *testing.T) {
		assert.Equal(t, int64(123456789), utils.GetInt64(m, "string"))
	})

	t.Run("nil_map", func(t *testing.T) {
		assert.Equal(t, int64(0), utils.GetInt64(nil, "key"))
	})
}

func TestGetFloat64(t *testing.T) {
	m := map[string]interface{}{
		"float64": 3.14159,
		"int":     42,
		"string":  "2.718",
	}

	t.Run("float64", func(t *testing.T) {
		assert.Equal(t, 3.14159, utils.GetFloat64(m, "float64"))
	})

	t.Run("int", func(t *testing.T) {
		assert.Equal(t, 42.0, utils.GetFloat64(m, "int"))
	})

	t.Run("string", func(t *testing.T) {
		assert.InDelta(t, 2.718, utils.GetFloat64(m, "string"), 0.001)
	})

	t.Run("nil_map", func(t *testing.T) {
		assert.Equal(t, 0.0, utils.GetFloat64(nil, "key"))
	})
}

func TestGetTimestamp(t *testing.T) {
	now := time.Now()
	m := map[string]interface{}{
		"time":      now,
		"time_ptr":  &now,
		"rfc3339":   "2024-01-15T14:30:00Z",
		"unix":      int64(1705276800),
		"empty":     "",
		"nil_value": nil,
	}

	t.Run("time_value", func(t *testing.T) {
		result := utils.GetTimestamp(m, "time")
		assert.NotNil(t, result)
		assert.Equal(t, now.Unix(), result.Unix())
	})

	t.Run("time_ptr", func(t *testing.T) {
		result := utils.GetTimestamp(m, "time_ptr")
		assert.NotNil(t, result)
	})

	t.Run("rfc3339_string", func(t *testing.T) {
		result := utils.GetTimestamp(m, "rfc3339")
		assert.NotNil(t, result)
		assert.Equal(t, 2024, result.Year())
	})

	t.Run("unix_timestamp", func(t *testing.T) {
		result := utils.GetTimestamp(m, "unix")
		assert.NotNil(t, result)
	})

	t.Run("empty_string", func(t *testing.T) {
		result := utils.GetTimestamp(m, "empty")
		assert.Nil(t, result)
	})

	t.Run("nil_value", func(t *testing.T) {
		result := utils.GetTimestamp(m, "nil_value")
		assert.Nil(t, result)
	})

	t.Run("non_existent_key", func(t *testing.T) {
		result := utils.GetTimestamp(m, "missing")
		assert.Nil(t, result)
	})

	t.Run("nil_map", func(t *testing.T) {
		result := utils.GetTimestamp(nil, "key")
		assert.Nil(t, result)
	})
}

func TestGetJSONValue(t *testing.T) {
	m := map[string]interface{}{
		"json_string": `{"nested":"value"}`,
		"json_array":  `[1,2,3]`,
		"parsed_map":  map[string]interface{}{"foo": "bar"},
		"empty":       "",
		"invalid":     "not json",
	}

	t.Run("json_string", func(t *testing.T) {
		result := utils.GetJSONValue(m, "json_string")
		assert.NotNil(t, result)
		nested, ok := result.(map[string]interface{})
		assert.True(t, ok)
		assert.Equal(t, "value", nested["nested"])
	})

	t.Run("json_array", func(t *testing.T) {
		result := utils.GetJSONValue(m, "json_array")
		assert.NotNil(t, result)
		arr, ok := result.([]interface{})
		assert.True(t, ok)
		assert.Len(t, arr, 3)
	})

	t.Run("parsed_map", func(t *testing.T) {
		result := utils.GetJSONValue(m, "parsed_map")
		assert.NotNil(t, result)
		parsed, ok := result.(map[string]interface{})
		assert.True(t, ok)
		assert.Equal(t, "bar", parsed["foo"])
	})

	t.Run("empty_string", func(t *testing.T) {
		result := utils.GetJSONValue(m, "empty")
		assert.Nil(t, result)
	})

	t.Run("invalid_json", func(t *testing.T) {
		result := utils.GetJSONValue(m, "invalid")
		assert.Nil(t, result)
	})

	t.Run("nil_map", func(t *testing.T) {
		result := utils.GetJSONValue(nil, "key")
		assert.Nil(t, result)
	})
}

// ==================== ToString Extended Tests ====================

func TestToStringExtended(t *testing.T) {
	t.Run("from_nil", func(t *testing.T) {
		assert.Equal(t, "", utils.ToString(nil))
	})

	t.Run("from_bytes", func(t *testing.T) {
		assert.Equal(t, "hello", utils.ToString([]byte("hello")))
	})

	t.Run("from_int_types", func(t *testing.T) {
		assert.Equal(t, "8", utils.ToString(int8(8)))
		assert.Equal(t, "16", utils.ToString(int16(16)))
		assert.Equal(t, "32", utils.ToString(int32(32)))
		assert.Equal(t, "64", utils.ToString(int64(64)))
	})

	t.Run("from_uint_types", func(t *testing.T) {
		assert.Equal(t, "8", utils.ToString(uint8(8)))
		assert.Equal(t, "16", utils.ToString(uint16(16)))
		assert.Equal(t, "32", utils.ToString(uint32(32)))
		assert.Equal(t, "64", utils.ToString(uint64(64)))
	})

	t.Run("from_float_formats_nicely", func(t *testing.T) {
		assert.Equal(t, "3.14", utils.ToString(3.14))
		assert.Equal(t, "1000", utils.ToString(1000.0)) // no trailing zeros
	})

	t.Run("from_struct_to_json", func(t *testing.T) {
		type TestStruct struct {
			Name string `json:"name"`
		}
		result := utils.ToString(TestStruct{Name: "test"})
		assert.Contains(t, result, "test")
	})
}
