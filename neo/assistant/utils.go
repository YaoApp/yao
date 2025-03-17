package assistant

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strconv"
	"time"

	jsoniter "github.com/json-iterator/go"
	"github.com/watchfultele/jsonrepair"
)

func getTimestamp(v interface{}) (int64, error) {
	switch v := v.(type) {
	case int64:
		return v, nil
	case int:
		return int64(v), nil

	case string:
		if ts, err := time.Parse(time.RFC3339, v); err == nil {
			return ts.UnixNano(), nil
		}

		// MySQL format
		if ts, err := time.Parse("2006-01-02 15:04:05", v); err == nil {
			return ts.UnixNano(), nil
		}

		// UnixNano format
		if ts, err := strconv.ParseInt(v, 10, 64); err == nil {
			return ts, nil
		}

	case nil:
		return 0, nil
	}

	return 0, fmt.Errorf("invalid timestamp type %T", v)
}

func stringToTimestamp(v string) (int64, error) {
	return strconv.ParseInt(v, 10, 64)
}

func timeToMySQLFormat(ts int64) string {
	if ts == 0 {
		return "0000-00-00 00:00:00"
	}
	return time.Unix(ts/1e9, ts%1e9).Format("2006-01-02 15:04:05")
}

// stringHash returns the sha256 hash of the string
func stringHash(v string) string {
	h := sha256.New()
	h.Write([]byte(v))
	return hex.EncodeToString(h.Sum(nil))
}

// ParseJSON attempts to parse a potentially malformed JSON string
func ParseJSON(jsonStr string, v interface{}) error {
	// Try parsing as-is first
	err := jsoniter.UnmarshalFromString(jsonStr, v)
	if err == nil {
		return nil
	}
	originalErr := err

	// Try adding a closing brace
	if err := jsoniter.UnmarshalFromString(jsonStr+"}", v); err == nil {
		return nil
	}

	// Try repairing the JSON
	repaired, err := jsonrepair.JSONRepair(jsonStr)
	if err != nil {
		return originalErr
	}

	// Try parsing the repaired JSON
	if err := jsoniter.UnmarshalFromString(repaired, v); err == nil {
		return nil
	}

	// If all attempts fail, return the original error
	return originalErr
}
