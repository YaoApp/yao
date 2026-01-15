package job

import (
	"encoding/json"
	"fmt"
	"time"

	yaojob "github.com/yaoapp/yao/job"

	"github.com/yaoapp/yao/agent/robot/types"
)

// Log writes a log entry for the execution
func Log(ctx *types.Context, exec *types.Execution, level string, message string, data map[string]interface{}) error {
	if exec == nil || exec.ID == "" || exec.JobID == "" {
		return fmt.Errorf("invalid execution or missing execution/job ID")
	}

	// Build context JSON with execution_id included
	if data == nil {
		data = make(map[string]interface{})
	}
	data["execution_id"] = exec.ID

	var contextRaw *json.RawMessage
	contextBytes, err := json.Marshal(data)
	if err == nil {
		raw := json.RawMessage(contextBytes)
		contextRaw = &raw
	}

	// Extract step from data if available
	var step *string
	if s, ok := data["step"].(string); ok {
		step = &s
	}

	logEntry := &yaojob.Log{
		JobID:       exec.JobID,
		Level:       level,
		Message:     message,
		Context:     contextRaw,
		ExecutionID: &exec.ID,
		Step:        step,
		Timestamp:   time.Now(),
		Sequence:    0,
	}

	return yaojob.SaveLog(logEntry)
}

// LogPhaseStart logs the start of a phase
func LogPhaseStart(ctx *types.Context, exec *types.Execution, phase types.Phase) error {
	if exec == nil || exec.ID == "" || exec.JobID == "" {
		return fmt.Errorf("invalid execution or missing execution/job ID")
	}

	locale := getLocale(ctx)
	phaseName := getPhaseName(locale, phase)

	var message string
	if isChineseLocale(locale) {
		message = fmt.Sprintf("阶段开始: %s", phaseName)
	} else {
		message = fmt.Sprintf("Phase started: %s", phaseName)
	}

	return Log(ctx, exec, "info", message, map[string]interface{}{
		"phase":      string(phase),
		"phase_name": phaseName,
		"step":       fmt.Sprintf("phase_%s_start", phase),
		"event":      "phase_start",
	})
}

// LogPhaseEnd logs the end of a phase
func LogPhaseEnd(ctx *types.Context, exec *types.Execution, phase types.Phase, durationMs int64) error {
	if exec == nil || exec.ID == "" || exec.JobID == "" {
		return fmt.Errorf("invalid execution or missing execution/job ID")
	}

	locale := getLocale(ctx)
	phaseName := getPhaseName(locale, phase)

	var message string
	if isChineseLocale(locale) {
		message = fmt.Sprintf("阶段完成: %s", phaseName)
	} else {
		message = fmt.Sprintf("Phase completed: %s", phaseName)
	}

	return Log(ctx, exec, "info", message, map[string]interface{}{
		"phase":       string(phase),
		"phase_name":  phaseName,
		"step":        fmt.Sprintf("phase_%s_end", phase),
		"event":       "phase_end",
		"duration_ms": durationMs,
	})
}

// LogPhaseError logs a phase error
func LogPhaseError(ctx *types.Context, exec *types.Execution, phase types.Phase, err error) error {
	if exec == nil || exec.ID == "" || exec.JobID == "" {
		return fmt.Errorf("invalid execution or missing execution/job ID")
	}

	locale := getLocale(ctx)
	phaseName := getPhaseName(locale, phase)

	errMsg := "unknown error"
	if err != nil {
		errMsg = err.Error()
	}

	var message string
	if isChineseLocale(locale) {
		message = fmt.Sprintf("阶段失败: %s - %s", phaseName, errMsg)
	} else {
		message = fmt.Sprintf("Phase failed: %s - %s", phaseName, errMsg)
	}

	return Log(ctx, exec, "error", message, map[string]interface{}{
		"phase":      string(phase),
		"phase_name": phaseName,
		"step":       fmt.Sprintf("phase_%s_error", phase),
		"event":      "phase_error",
		"error":      errMsg,
	})
}

// LogError logs an error
func LogError(ctx *types.Context, exec *types.Execution, err error) error {
	if exec == nil || exec.ID == "" || exec.JobID == "" {
		return fmt.Errorf("invalid execution or missing execution/job ID")
	}

	locale := getLocale(ctx)
	errMsg := "unknown error"
	if err != nil {
		errMsg = err.Error()
	}

	var message string
	if isChineseLocale(locale) {
		message = fmt.Sprintf("错误: %s", errMsg)
	} else {
		message = errMsg
	}

	return Log(ctx, exec, "error", message, map[string]interface{}{
		"event": "error",
		"error": errMsg,
	})
}

// LogInfo logs an info message
func LogInfo(ctx *types.Context, exec *types.Execution, message string) error {
	if exec == nil || exec.ID == "" || exec.JobID == "" {
		return fmt.Errorf("invalid execution or missing execution/job ID")
	}
	return Log(ctx, exec, "info", message, nil)
}

// LogDebug logs a debug message
func LogDebug(ctx *types.Context, exec *types.Execution, message string) error {
	if exec == nil || exec.ID == "" || exec.JobID == "" {
		return fmt.Errorf("invalid execution or missing execution/job ID")
	}
	return Log(ctx, exec, "debug", message, nil)
}

// LogWarn logs a warning message
func LogWarn(ctx *types.Context, exec *types.Execution, message string) error {
	if exec == nil || exec.ID == "" || exec.JobID == "" {
		return fmt.Errorf("invalid execution or missing execution/job ID")
	}
	return Log(ctx, exec, "warning", message, nil)
}

// LogTaskStart logs the start of a task
func LogTaskStart(ctx *types.Context, exec *types.Execution, taskID string, taskOrder int) error {
	if exec == nil || exec.ID == "" || exec.JobID == "" {
		return fmt.Errorf("invalid execution or missing execution/job ID")
	}

	locale := getLocale(ctx)

	var message string
	if isChineseLocale(locale) {
		message = fmt.Sprintf("任务开始: %s", taskID)
	} else {
		message = fmt.Sprintf("Task started: %s", taskID)
	}

	return Log(ctx, exec, "info", message, map[string]interface{}{
		"task_id":    taskID,
		"task_order": taskOrder,
		"step":       fmt.Sprintf("task_%d_start", taskOrder),
		"event":      "task_start",
	})
}

// LogTaskEnd logs the end of a task
func LogTaskEnd(ctx *types.Context, exec *types.Execution, taskID string, taskOrder int, success bool, durationMs int64) error {
	if exec == nil || exec.ID == "" || exec.JobID == "" {
		return fmt.Errorf("invalid execution or missing execution/job ID")
	}

	locale := getLocale(ctx)
	level := "info"
	event := "task_success"

	var msg string
	if isChineseLocale(locale) {
		if success {
			msg = fmt.Sprintf("任务完成: %s", taskID)
		} else {
			level = "warning"
			event = "task_failed"
			msg = fmt.Sprintf("任务失败: %s", taskID)
		}
	} else {
		if success {
			msg = fmt.Sprintf("Task completed: %s", taskID)
		} else {
			level = "warning"
			event = "task_failed"
			msg = fmt.Sprintf("Task failed: %s", taskID)
		}
	}

	return Log(ctx, exec, level, msg, map[string]interface{}{
		"task_id":     taskID,
		"task_order":  taskOrder,
		"step":        fmt.Sprintf("task_%d_end", taskOrder),
		"event":       event,
		"success":     success,
		"duration_ms": durationMs,
	})
}

// LogDelivery logs delivery result
func LogDelivery(ctx *types.Context, exec *types.Execution, deliveryType string, success bool) error {
	if exec == nil || exec.ID == "" || exec.JobID == "" {
		return fmt.Errorf("invalid execution or missing execution/job ID")
	}

	locale := getLocale(ctx)
	level := "info"

	var msg string
	if isChineseLocale(locale) {
		if success {
			msg = fmt.Sprintf("交付完成: %s", deliveryType)
		} else {
			level = "warning"
			msg = fmt.Sprintf("交付失败: %s", deliveryType)
		}
	} else {
		if success {
			msg = fmt.Sprintf("Delivery completed: %s", deliveryType)
		} else {
			level = "warning"
			msg = fmt.Sprintf("Delivery failed: %s", deliveryType)
		}
	}

	return Log(ctx, exec, level, msg, map[string]interface{}{
		"delivery_type": deliveryType,
		"step":          "delivery",
		"event":         "delivery",
		"success":       success,
	})
}

// LogLearning logs learning result
func LogLearning(ctx *types.Context, exec *types.Execution, entriesCount int) error {
	if exec == nil || exec.ID == "" || exec.JobID == "" {
		return fmt.Errorf("invalid execution or missing execution/job ID")
	}

	locale := getLocale(ctx)

	var msg string
	if isChineseLocale(locale) {
		msg = fmt.Sprintf("学习保存: %d 条记录", entriesCount)
	} else {
		msg = fmt.Sprintf("Learning saved: %d entries", entriesCount)
	}

	return Log(ctx, exec, "info", msg, map[string]interface{}{
		"entries_count": entriesCount,
		"step":          "learning",
		"event":         "learning",
	})
}
