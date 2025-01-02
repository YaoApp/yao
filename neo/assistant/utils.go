package assistant

import (
	"fmt"
	"strconv"
	"time"
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

	}
	return 0, fmt.Errorf("invalid timestamp type")
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
