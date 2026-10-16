package llm

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/yaoapp/kun/maps"
)

// ---------------------------------------------------------------------------
// Row value extraction helpers — safely extract typed values from maps.MapStr
// returned by gou model queries.
// ---------------------------------------------------------------------------

func getString(row maps.MapStr, key string) string {
	v, ok := row[key]
	if !ok || v == nil {
		return ""
	}
	if s, ok := v.(string); ok {
		return s
	}
	return fmt.Sprintf("%v", v)
}

func getInt(row maps.MapStr, key string) int {
	v, ok := row[key]
	if !ok || v == nil {
		return 0
	}
	return toIntFromAny(v)
}

func getFloat(row maps.MapStr, key string) float64 {
	v, ok := row[key]
	if !ok || v == nil {
		return 0
	}
	switch n := v.(type) {
	case float64:
		return n
	case float32:
		return float64(n)
	case int:
		return float64(n)
	case int64:
		return float64(n)
	case json.Number:
		f, _ := n.Float64()
		return f
	}
	return 0
}

func getBool(row maps.MapStr, key string) bool {
	v, ok := row[key]
	if !ok || v == nil {
		return false
	}
	switch b := v.(type) {
	case bool:
		return b
	case int:
		return b != 0
	case int64:
		return b != 0
	case float64:
		return b != 0
	}
	return false
}

func getTime(row maps.MapStr, key string) time.Time {
	v, ok := row[key]
	if !ok || v == nil {
		return time.Time{}
	}
	switch t := v.(type) {
	case time.Time:
		return t
	case string:
		formats := []string{time.RFC3339, "2006-01-02 15:04:05", "2006-01-02T15:04:05Z"}
		for _, f := range formats {
			if parsed, err := time.Parse(f, t); err == nil {
				return parsed
			}
		}
	}
	return time.Time{}
}

func toIntFromAny(v interface{}) int {
	switch n := v.(type) {
	case int:
		return n
	case int64:
		return int(n)
	case float64:
		return int(n)
	case json.Number:
		i, _ := n.Int64()
		return int(i)
	}
	return 0
}

func toJSONMap(v interface{}) JSONMap {
	switch m := v.(type) {
	case map[string]interface{}:
		return m
	case string:
		var out JSONMap
		if err := json.Unmarshal([]byte(m), &out); err == nil {
			return out
		}
	}
	return nil
}

func toAPISlice(v interface{}) []API {
	switch s := v.(type) {
	case []API:
		return s
	case []interface{}:
		out := make([]API, len(s))
		for i, item := range s {
			out[i] = API(fmt.Sprintf("%v", item))
		}
		return out
	case []string:
		out := make([]API, len(s))
		for i, item := range s {
			out[i] = API(item)
		}
		return out
	case string:
		var arr []string
		if err := json.Unmarshal([]byte(s), &arr); err == nil {
			out := make([]API, len(arr))
			for i, item := range arr {
				out[i] = API(item)
			}
			return out
		}
	}
	return nil
}

func toStringSliceFromAny(v interface{}) []string {
	switch s := v.(type) {
	case []interface{}:
		out := make([]string, len(s))
		for i, item := range s {
			out[i] = fmt.Sprintf("%v", item)
		}
		return out
	case []string:
		return s
	case string:
		var arr []string
		if err := json.Unmarshal([]byte(s), &arr); err == nil {
			return arr
		}
	}
	return nil
}

func toMapStringAny(v interface{}) map[string]any {
	switch m := v.(type) {
	case map[string]interface{}:
		return m
	case string:
		var out map[string]any
		if err := json.Unmarshal([]byte(m), &out); err == nil {
			return out
		}
	}
	return nil
}

func toThinkingLevelMap(v interface{}) ThinkingLevelMap {
	m := toMapStringAny(v)
	if m == nil {
		return nil
	}
	out := make(ThinkingLevelMap, len(m))
	for k, val := range m {
		if val == nil {
			out[ThinkingLevel(k)] = nil
		} else if s, ok := val.(string); ok {
			out[ThinkingLevel(k)] = &s
		}
	}
	return out
}

func toSamplingParamsByLevel(v interface{}) SamplingParamsByThinkingLevel {
	m := toMapStringAny(v)
	if m == nil {
		return nil
	}
	out := make(SamplingParamsByThinkingLevel, len(m))
	for k, val := range m {
		if params, ok := val.(map[string]interface{}); ok {
			out[ThinkingLevel(k)] = params
		}
	}
	return out
}

func toCostTiers(v interface{}) []CostTier {
	switch s := v.(type) {
	case []interface{}:
		out := make([]CostTier, 0, len(s))
		for _, item := range s {
			if m, ok := item.(map[string]interface{}); ok {
				ct := CostTier{
					InputTokensAbove:    toIntFromAny(m["inputTokensAbove"]),
					InputCostPer1M:      toFloatFromAny(m["input"]),
					OutputCostPer1M:     toFloatFromAny(m["output"]),
					CacheReadCostPer1M:  toFloatFromAny(m["cacheRead"]),
					CacheWriteCostPer1M: toFloatFromAny(m["cacheWrite"]),
				}
				out = append(out, ct)
			}
		}
		return out
	case string:
		var arr []CostTier
		if err := json.Unmarshal([]byte(s), &arr); err == nil {
			return arr
		}
	}
	return nil
}

func toInputLimits(v interface{}) *ModelInputLimits {
	m := toMapStringAny(v)
	if m == nil {
		if s, ok := v.(string); ok {
			var out ModelInputLimits
			if err := json.Unmarshal([]byte(s), &out); err == nil {
				return &out
			}
		}
		return nil
	}
	data, _ := json.Marshal(m)
	var out ModelInputLimits
	if err := json.Unmarshal(data, &out); err == nil {
		return &out
	}
	return nil
}

func toPromptCache(v interface{}) ModelPromptCache {
	m := toMapStringAny(v)
	if m == nil {
		return nil
	}
	out := make(ModelPromptCache, len(m))
	for k, val := range m {
		out[k] = toIntFromAny(val)
	}
	return out
}

func toFloatFromAny(v interface{}) float64 {
	switch n := v.(type) {
	case float64:
		return n
	case float32:
		return float64(n)
	case int:
		return float64(n)
	case int64:
		return float64(n)
	case json.Number:
		f, _ := n.Float64()
		return f
	}
	return 0
}
