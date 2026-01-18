package assert

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	jsoniter "github.com/json-iterator/go"
	"github.com/yaoapp/gou/text"
)

// ValidateOutput compares two values for equality using JSON serialization
func ValidateOutput(actual, expected interface{}) bool {
	actualJSON, err1 := jsoniter.Marshal(actual)
	expectedJSON, err2 := jsoniter.Marshal(expected)

	if err1 != nil || err2 != nil {
		return false
	}

	return string(actualJSON) == string(expectedJSON)
}

// ToString converts a value to string for comparison
func ToString(v interface{}) string {
	if v == nil {
		return ""
	}

	switch val := v.(type) {
	case string:
		return val
	case []byte:
		return string(val)
	default:
		b, err := json.Marshal(v)
		if err != nil {
			return fmt.Sprintf("%v", v)
		}
		return string(b)
	}
}

// GetType returns the type name of a value
func GetType(v interface{}) string {
	if v == nil {
		return "null"
	}

	switch v.(type) {
	case string:
		return "string"
	case float64, float32, int, int64, int32:
		return "number"
	case bool:
		return "boolean"
	case []interface{}:
		return "array"
	case map[string]interface{}:
		return "object"
	default:
		return fmt.Sprintf("%T", v)
	}
}

// ExtractPath extracts a value from JSON using dot-notation path with array index support
// Supports: "field", "field.nested", "field[0]", "field[0].nested", "field.nested[0].value"
func ExtractPath(data interface{}, path string) interface{} {
	current := data

	segments := ParsePathSegments(path)

	for _, segment := range segments {
		if segment == "" {
			continue
		}

		// Check if this is an array index like "[0]"
		if strings.HasPrefix(segment, "[") && strings.HasSuffix(segment, "]") {
			indexStr := segment[1 : len(segment)-1]
			index, err := strconv.Atoi(indexStr)
			if err != nil {
				return nil
			}

			arr, ok := current.([]interface{})
			if !ok {
				return nil
			}

			if index < 0 || index >= len(arr) {
				return nil
			}
			current = arr[index]
		} else {
			// Regular field access
			switch v := current.(type) {
			case map[string]interface{}:
				current = v[segment]
			default:
				return nil
			}
		}
	}

	return current
}

// ParsePathSegments splits a path like "wheres[0].like" into ["wheres", "[0]", "like"]
func ParsePathSegments(path string) []string {
	var segments []string
	var current strings.Builder

	for i := 0; i < len(path); i++ {
		ch := path[i]
		switch ch {
		case '.':
			if current.Len() > 0 {
				segments = append(segments, current.String())
				current.Reset()
			}
		case '[':
			if current.Len() > 0 {
				segments = append(segments, current.String())
				current.Reset()
			}
			j := i + 1
			for j < len(path) && path[j] != ']' {
				j++
			}
			if j < len(path) {
				segments = append(segments, path[i:j+1])
				i = j
			}
		default:
			current.WriteByte(ch)
		}
	}

	if current.Len() > 0 {
		segments = append(segments, current.String())
	}

	return segments
}

// TruncateOutput truncates output for error messages
func TruncateOutput(output interface{}, maxLen int) string {
	var s string
	switch v := output.(type) {
	case string:
		s = v
	case nil:
		return "<nil>"
	default:
		bytes, err := jsoniter.Marshal(v)
		if err != nil {
			s = fmt.Sprintf("%v", v)
		} else {
			s = string(bytes)
		}
	}

	if len(s) > maxLen {
		return s[:maxLen] + "..."
	}
	return s
}

// ExtractJSON extracts JSON from text (handles markdown code blocks, etc.)
func ExtractJSON(content string) interface{} {
	return text.ExtractJSON(content)
}
