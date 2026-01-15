package job_test

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/yaoapp/gou/model"
	"github.com/yaoapp/kun/maps"
	yaojob "github.com/yaoapp/yao/job"

	"github.com/yaoapp/yao/agent/robot/job"
	"github.com/yaoapp/yao/agent/robot/types"
	"github.com/yaoapp/yao/agent/testutils"
)

// TestLog tests writing log entries
func TestLog(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	testutils.Prepare(t)
	defer testutils.Clean(t)

	ctx := types.NewContext(context.Background(), nil)

	t.Run("write info log", func(t *testing.T) {
		robot := createTestRobot("test_log_001")
		exec, err := job.CreateExecution(ctx, &job.CreateOptions{
			Robot:       robot,
			TriggerType: types.TriggerClock,
		})
		require.NoError(t, err)

		err = job.Log(ctx, exec, "info", "Test message", map[string]interface{}{
			"key": "value",
		})
		require.NoError(t, err)

		// Verify log was written
		logs, err := getJobLogs(exec.JobID)
		require.NoError(t, err)
		assert.NotEmpty(t, logs)

		found := false
		for _, log := range logs {
			if log.Message == "Test message" && log.Level == "info" {
				found = true
				break
			}
		}
		assert.True(t, found, "Log entry should be found")
	})

	t.Run("write error log", func(t *testing.T) {
		robot := createTestRobot("test_log_002")
		exec, err := job.CreateExecution(ctx, &job.CreateOptions{
			Robot:       robot,
			TriggerType: types.TriggerClock,
		})
		require.NoError(t, err)

		err = job.Log(ctx, exec, "error", "Error occurred", nil)
		require.NoError(t, err)

		logs, err := getJobLogs(exec.JobID)
		require.NoError(t, err)

		found := false
		for _, log := range logs {
			if log.Message == "Error occurred" && log.Level == "error" {
				found = true
				break
			}
		}
		assert.True(t, found, "Error log entry should be found")
	})

	t.Run("log with nil execution returns error", func(t *testing.T) {
		err := job.Log(ctx, nil, "info", "Test", nil)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid execution")
	})

	t.Run("log with empty job ID returns error", func(t *testing.T) {
		exec := &types.Execution{
			ID:    "some_id",
			JobID: "",
		}
		err := job.Log(ctx, exec, "info", "Test", nil)
		assert.Error(t, err)
	})
}

// TestLogPhaseStart tests logging phase start
func TestLogPhaseStart(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	testutils.Prepare(t)
	defer testutils.Clean(t)

	ctx := types.NewContext(context.Background(), nil)

	t.Run("log phase start in english", func(t *testing.T) {
		ctx := &types.Context{
			Context: context.Background(),
			Locale:  "en-US",
		}
		robot := createTestRobot("test_phase_log_001")
		exec, err := job.CreateExecution(ctx, &job.CreateOptions{
			Robot:       robot,
			TriggerType: types.TriggerClock,
		})
		require.NoError(t, err)

		err = job.LogPhaseStart(ctx, exec, types.PhaseGoals)
		require.NoError(t, err)

		logs, err := getJobLogs(exec.JobID)
		require.NoError(t, err)

		found := false
		for _, log := range logs {
			if log.Level == "info" && containsString(log.Message, "Phase started") && containsString(log.Message, "Goals") {
				found = true
				break
			}
		}
		assert.True(t, found, "Phase start log should be found")
	})

	t.Run("log phase start in chinese", func(t *testing.T) {
		ctx := &types.Context{
			Context: context.Background(),
			Locale:  "zh-CN",
		}
		robot := createTestRobot("test_phase_log_002")
		exec, err := job.CreateExecution(ctx, &job.CreateOptions{
			Robot:       robot,
			TriggerType: types.TriggerClock,
		})
		require.NoError(t, err)

		err = job.LogPhaseStart(ctx, exec, types.PhaseGoals)
		require.NoError(t, err)

		logs, err := getJobLogs(exec.JobID)
		require.NoError(t, err)

		found := false
		for _, log := range logs {
			if log.Level == "info" && containsString(log.Message, "阶段开始") {
				found = true
				break
			}
		}
		assert.True(t, found, "Chinese phase start log should be found")
	})

	t.Run("log phase start with nil execution returns error", func(t *testing.T) {
		err := job.LogPhaseStart(ctx, nil, types.PhaseGoals)
		assert.Error(t, err)
	})
}

// TestLogPhaseEnd tests logging phase end
func TestLogPhaseEnd(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	testutils.Prepare(t)
	defer testutils.Clean(t)

	ctx := types.NewContext(context.Background(), nil)

	t.Run("log phase end with duration", func(t *testing.T) {
		robot := createTestRobot("test_phase_end_001")
		exec, err := job.CreateExecution(ctx, &job.CreateOptions{
			Robot:       robot,
			TriggerType: types.TriggerClock,
		})
		require.NoError(t, err)

		err = job.LogPhaseEnd(ctx, exec, types.PhaseInspiration, 1500)
		require.NoError(t, err)

		logs, err := getJobLogs(exec.JobID)
		require.NoError(t, err)

		found := false
		for _, log := range logs {
			if log.Level == "info" && (containsString(log.Message, "Phase completed") || containsString(log.Message, "阶段完成")) {
				found = true
				break
			}
		}
		assert.True(t, found, "Phase end log should be found")
	})
}

// TestLogPhaseError tests logging phase error
func TestLogPhaseError(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	testutils.Prepare(t)
	defer testutils.Clean(t)

	ctx := types.NewContext(context.Background(), nil)

	t.Run("log phase error", func(t *testing.T) {
		robot := createTestRobot("test_phase_err_001")
		exec, err := job.CreateExecution(ctx, &job.CreateOptions{
			Robot:       robot,
			TriggerType: types.TriggerClock,
		})
		require.NoError(t, err)

		testErr := errors.New("goal generation failed")
		err = job.LogPhaseError(ctx, exec, types.PhaseGoals, testErr)
		require.NoError(t, err)

		logs, err := getJobLogs(exec.JobID)
		require.NoError(t, err)

		found := false
		for _, log := range logs {
			if log.Level == "error" && containsString(log.Message, "goal generation failed") {
				found = true
				break
			}
		}
		assert.True(t, found, "Phase error log should be found")
	})

	t.Run("log phase error with nil error", func(t *testing.T) {
		robot := createTestRobot("test_phase_err_002")
		exec, err := job.CreateExecution(ctx, &job.CreateOptions{
			Robot:       robot,
			TriggerType: types.TriggerClock,
		})
		require.NoError(t, err)

		err = job.LogPhaseError(ctx, exec, types.PhaseGoals, nil)
		require.NoError(t, err)

		logs, err := getJobLogs(exec.JobID)
		require.NoError(t, err)

		found := false
		for _, log := range logs {
			if log.Level == "error" && containsString(log.Message, "unknown error") {
				found = true
				break
			}
		}
		assert.True(t, found, "Phase error log with unknown error should be found")
	})
}

// TestLogError tests logging errors
func TestLogError(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	testutils.Prepare(t)
	defer testutils.Clean(t)

	ctx := types.NewContext(context.Background(), nil)

	t.Run("log error", func(t *testing.T) {
		robot := createTestRobot("test_error_001")
		exec, err := job.CreateExecution(ctx, &job.CreateOptions{
			Robot:       robot,
			TriggerType: types.TriggerClock,
		})
		require.NoError(t, err)

		testErr := errors.New("connection timeout")
		err = job.LogError(ctx, exec, testErr)
		require.NoError(t, err)

		logs, err := getJobLogs(exec.JobID)
		require.NoError(t, err)

		found := false
		for _, log := range logs {
			if log.Level == "error" && containsString(log.Message, "connection timeout") {
				found = true
				break
			}
		}
		assert.True(t, found, "Error log should be found")
	})
}

// TestLogInfo tests logging info messages
func TestLogInfo(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	testutils.Prepare(t)
	defer testutils.Clean(t)

	ctx := types.NewContext(context.Background(), nil)

	t.Run("log info message", func(t *testing.T) {
		robot := createTestRobot("test_info_001")
		exec, err := job.CreateExecution(ctx, &job.CreateOptions{
			Robot:       robot,
			TriggerType: types.TriggerClock,
		})
		require.NoError(t, err)

		err = job.LogInfo(ctx, exec, "Processing started")
		require.NoError(t, err)

		logs, err := getJobLogs(exec.JobID)
		require.NoError(t, err)

		found := false
		for _, log := range logs {
			if log.Level == "info" && log.Message == "Processing started" {
				found = true
				break
			}
		}
		assert.True(t, found, "Info log should be found")
	})
}

// TestLogDebug tests logging debug messages
func TestLogDebug(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	testutils.Prepare(t)
	defer testutils.Clean(t)

	ctx := types.NewContext(context.Background(), nil)

	t.Run("log debug message", func(t *testing.T) {
		robot := createTestRobot("test_debug_001")
		exec, err := job.CreateExecution(ctx, &job.CreateOptions{
			Robot:       robot,
			TriggerType: types.TriggerClock,
		})
		require.NoError(t, err)

		err = job.LogDebug(ctx, exec, "Debug info")
		require.NoError(t, err)

		logs, err := getJobLogs(exec.JobID)
		require.NoError(t, err)

		found := false
		for _, log := range logs {
			if log.Level == "debug" && log.Message == "Debug info" {
				found = true
				break
			}
		}
		assert.True(t, found, "Debug log should be found")
	})
}

// TestLogWarn tests logging warning messages
func TestLogWarn(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	testutils.Prepare(t)
	defer testutils.Clean(t)

	ctx := types.NewContext(context.Background(), nil)

	t.Run("log warning message", func(t *testing.T) {
		robot := createTestRobot("test_warn_001")
		exec, err := job.CreateExecution(ctx, &job.CreateOptions{
			Robot:       robot,
			TriggerType: types.TriggerClock,
		})
		require.NoError(t, err)

		err = job.LogWarn(ctx, exec, "Resource running low")
		require.NoError(t, err)

		logs, err := getJobLogs(exec.JobID)
		require.NoError(t, err)

		found := false
		for _, log := range logs {
			if log.Level == "warning" && log.Message == "Resource running low" {
				found = true
				break
			}
		}
		assert.True(t, found, "Warning log should be found")
	})
}

// TestLogTaskStart tests logging task start
func TestLogTaskStart(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	testutils.Prepare(t)
	defer testutils.Clean(t)

	ctx := types.NewContext(context.Background(), nil)

	t.Run("log task start", func(t *testing.T) {
		robot := createTestRobot("test_task_start_001")
		exec, err := job.CreateExecution(ctx, &job.CreateOptions{
			Robot:       robot,
			TriggerType: types.TriggerClock,
		})
		require.NoError(t, err)

		err = job.LogTaskStart(ctx, exec, "task_001", 1)
		require.NoError(t, err)

		logs, err := getJobLogs(exec.JobID)
		require.NoError(t, err)

		found := false
		for _, log := range logs {
			if log.Level == "info" && containsString(log.Message, "task_001") {
				found = true
				break
			}
		}
		assert.True(t, found, "Task start log should be found")
	})
}

// TestLogTaskEnd tests logging task end
func TestLogTaskEnd(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	testutils.Prepare(t)
	defer testutils.Clean(t)

	ctx := types.NewContext(context.Background(), nil)

	t.Run("log task end success", func(t *testing.T) {
		robot := createTestRobot("test_task_end_001")
		exec, err := job.CreateExecution(ctx, &job.CreateOptions{
			Robot:       robot,
			TriggerType: types.TriggerClock,
		})
		require.NoError(t, err)

		err = job.LogTaskEnd(ctx, exec, "task_001", 1, true, 500)
		require.NoError(t, err)

		logs, err := getJobLogs(exec.JobID)
		require.NoError(t, err)

		found := false
		for _, log := range logs {
			if log.Level == "info" && containsString(log.Message, "task_001") {
				found = true
				break
			}
		}
		assert.True(t, found, "Task end success log should be found")
	})

	t.Run("log task end failure", func(t *testing.T) {
		robot := createTestRobot("test_task_end_002")
		exec, err := job.CreateExecution(ctx, &job.CreateOptions{
			Robot:       robot,
			TriggerType: types.TriggerClock,
		})
		require.NoError(t, err)

		err = job.LogTaskEnd(ctx, exec, "task_002", 2, false, 300)
		require.NoError(t, err)

		logs, err := getJobLogs(exec.JobID)
		require.NoError(t, err)

		found := false
		for _, log := range logs {
			if log.Level == "warning" && containsString(log.Message, "task_002") {
				found = true
				break
			}
		}
		assert.True(t, found, "Task end failure log should be found")
	})
}

// TestLogDelivery tests logging delivery
func TestLogDelivery(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	testutils.Prepare(t)
	defer testutils.Clean(t)

	ctx := types.NewContext(context.Background(), nil)

	t.Run("log delivery success", func(t *testing.T) {
		robot := createTestRobot("test_delivery_001")
		exec, err := job.CreateExecution(ctx, &job.CreateOptions{
			Robot:       robot,
			TriggerType: types.TriggerClock,
		})
		require.NoError(t, err)

		err = job.LogDelivery(ctx, exec, "email", true)
		require.NoError(t, err)

		logs, err := getJobLogs(exec.JobID)
		require.NoError(t, err)

		found := false
		for _, log := range logs {
			if log.Level == "info" && containsString(log.Message, "email") {
				found = true
				break
			}
		}
		assert.True(t, found, "Delivery success log should be found")
	})

	t.Run("log delivery failure", func(t *testing.T) {
		robot := createTestRobot("test_delivery_002")
		exec, err := job.CreateExecution(ctx, &job.CreateOptions{
			Robot:       robot,
			TriggerType: types.TriggerClock,
		})
		require.NoError(t, err)

		err = job.LogDelivery(ctx, exec, "webhook", false)
		require.NoError(t, err)

		logs, err := getJobLogs(exec.JobID)
		require.NoError(t, err)

		found := false
		for _, log := range logs {
			if log.Level == "warning" && containsString(log.Message, "webhook") {
				found = true
				break
			}
		}
		assert.True(t, found, "Delivery failure log should be found")
	})
}

// TestLogLearning tests logging learning
func TestLogLearning(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	testutils.Prepare(t)
	defer testutils.Clean(t)

	ctx := types.NewContext(context.Background(), nil)

	t.Run("log learning entries", func(t *testing.T) {
		robot := createTestRobot("test_learning_001")
		exec, err := job.CreateExecution(ctx, &job.CreateOptions{
			Robot:       robot,
			TriggerType: types.TriggerClock,
		})
		require.NoError(t, err)

		err = job.LogLearning(ctx, exec, 5)
		require.NoError(t, err)

		logs, err := getJobLogs(exec.JobID)
		require.NoError(t, err)

		found := false
		for _, log := range logs {
			if log.Level == "info" && (containsString(log.Message, "5") || containsString(log.Message, "Learning")) {
				found = true
				break
			}
		}
		assert.True(t, found, "Learning log should be found")
	})
}

// TestLogLocalization tests log message localization
func TestLogLocalization(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	testutils.Prepare(t)
	defer testutils.Clean(t)

	t.Run("english locale messages", func(t *testing.T) {
		ctx := &types.Context{
			Context: context.Background(),
			Locale:  "en-US",
		}
		robot := createTestRobot("test_locale_en_001")
		exec, err := job.CreateExecution(ctx, &job.CreateOptions{
			Robot:       robot,
			TriggerType: types.TriggerClock,
		})
		require.NoError(t, err)

		err = job.LogPhaseStart(ctx, exec, types.PhaseRun)
		require.NoError(t, err)

		logs, err := getJobLogs(exec.JobID)
		require.NoError(t, err)

		found := false
		for _, log := range logs {
			if containsString(log.Message, "Phase started") && containsString(log.Message, "Run") {
				found = true
				break
			}
		}
		assert.True(t, found, "English phase start message should be found")
	})

	t.Run("chinese locale messages", func(t *testing.T) {
		ctx := &types.Context{
			Context: context.Background(),
			Locale:  "zh-CN",
		}
		robot := createTestRobot("test_locale_zh_001")
		exec, err := job.CreateExecution(ctx, &job.CreateOptions{
			Robot:       robot,
			TriggerType: types.TriggerClock,
		})
		require.NoError(t, err)

		err = job.LogPhaseStart(ctx, exec, types.PhaseRun)
		require.NoError(t, err)

		logs, err := getJobLogs(exec.JobID)
		require.NoError(t, err)

		found := false
		for _, log := range logs {
			if containsString(log.Message, "阶段开始") && containsString(log.Message, "任务执行") {
				found = true
				break
			}
		}
		assert.True(t, found, "Chinese phase start message should be found")
	})
}

// getJobLogs retrieves logs for a job
func getJobLogs(jobID string) ([]*yaojob.Log, error) {
	result, err := yaojob.ListLogs(jobID, model.QueryParam{}, 1, 100)
	if err != nil {
		return nil, err
	}

	data, exists := result["data"]
	if !exists {
		return nil, fmt.Errorf("ListLogs result missing 'data' field")
	}

	// Handle nil data
	if data == nil {
		return []*yaojob.Log{}, nil
	}

	// Handle different data types from ListLogs
	var logs []*yaojob.Log

	switch typedData := data.(type) {
	case []maps.MapStrAny:
		for _, item := range typedData {
			log := &yaojob.Log{}
			if msg, ok := item["message"].(string); ok {
				log.Message = msg
			}
			if level, ok := item["level"].(string); ok {
				log.Level = level
			}
			if jid, ok := item["job_id"].(string); ok {
				log.JobID = jid
			}
			logs = append(logs, log)
		}
	case []map[string]interface{}:
		for _, item := range typedData {
			log := &yaojob.Log{}
			if msg, ok := item["message"].(string); ok {
				log.Message = msg
			}
			if level, ok := item["level"].(string); ok {
				log.Level = level
			}
			if jid, ok := item["job_id"].(string); ok {
				log.JobID = jid
			}
			logs = append(logs, log)
		}
	case []interface{}:
		// Handle generic []interface{} which may contain map types
		for _, rawItem := range typedData {
			log := &yaojob.Log{}
			switch item := rawItem.(type) {
			case maps.MapStrAny:
				if msg, ok := item["message"].(string); ok {
					log.Message = msg
				}
				if level, ok := item["level"].(string); ok {
					log.Level = level
				}
				if jid, ok := item["job_id"].(string); ok {
					log.JobID = jid
				}
			case map[string]interface{}:
				if msg, ok := item["message"].(string); ok {
					log.Message = msg
				}
				if level, ok := item["level"].(string); ok {
					log.Level = level
				}
				if jid, ok := item["job_id"].(string); ok {
					log.JobID = jid
				}
			default:
				return nil, fmt.Errorf("unexpected item type in data array: %T", rawItem)
			}
			logs = append(logs, log)
		}
	default:
		return nil, fmt.Errorf("unexpected data type from ListLogs: %T (value: %v)", data, data)
	}

	return logs, nil
}

// containsString checks if a string contains a substring
func containsString(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		(len(s) > 0 && len(substr) > 0 && findSubstring(s, substr)))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
