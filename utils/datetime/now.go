package datetime

import (
	"time"

	"github.com/yaoapp/gou/process"
)

// ProcessTimestamp utils.now.Timestamp
func ProcessTimestamp(process *process.Process) interface{} {
	return time.Now().Unix()
}

// ProcessTimestampms utils.now.Timestampms
func ProcessTimestampms(process *process.Process) interface{} {
	return time.Now().UnixMilli()
}

// ProcessDate utils.now.Date
func ProcessDate(process *process.Process) interface{} {
	return time.Now().Local().Format("2006-01-02")
}

// ProcessTime utils.now.Time
func ProcessTime(process *process.Process) interface{} {
	return time.Now().Local().Format("15:04:05")
}

// ProcessDateTime utils.now.DateTime
func ProcessDateTime(process *process.Process) interface{} {
	return time.Now().Local().Format("2006-01-02 15:04:05")
}
