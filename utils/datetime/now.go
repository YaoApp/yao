package datetime

import (
	"time"

	"github.com/yaoapp/gou"
)

// ProcessTimestamp utils.now.Timestamp
func ProcessTimestamp(process *gou.Process) interface{} {
	return time.Now().Unix()
}

// ProcessTimestampms utils.now.Timestampms
func ProcessTimestampms(process *gou.Process) interface{} {
	return time.Now().UnixMilli()
}

// ProcessDate utils.now.Date
func ProcessDate(process *gou.Process) interface{} {
	return time.Now().Local().Format("2006-01-02")
}

// ProcessTime utils.now.Time
func ProcessTime(process *gou.Process) interface{} {
	return time.Now().Local().Format("15:04:05")
}

// ProcessDateTime utils.now.DateTime
func ProcessDateTime(process *gou.Process) interface{} {
	return time.Now().Local().Format("2006-01-02 15:04:05")
}
