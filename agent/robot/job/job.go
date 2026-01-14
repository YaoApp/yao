package job

import "github.com/yaoapp/yao/agent/robot/types"

// Create creates a new job for robot execution
// Stub: returns empty job ID (will be implemented in Phase 3)
func Create(ctx *types.Context, exec *types.Execution) (string, error) {
	return "", nil
}

// Update updates job status
// Stub: returns nil (will be implemented in Phase 3)
func Update(ctx *types.Context, jobID string, status types.ExecStatus, phase types.Phase) error {
	return nil
}

// Log writes a log entry for the execution
// Stub: returns nil (will be implemented in Phase 3)
func Log(ctx *types.Context, jobID string, level string, message string, data map[string]interface{}) error {
	return nil
}

// Complete marks job as completed
// Stub: returns nil (will be implemented in Phase 3)
func Complete(ctx *types.Context, jobID string, exec *types.Execution) error {
	return nil
}

// Fail marks job as failed
// Stub: returns nil (will be implemented in Phase 3)
func Fail(ctx *types.Context, jobID string, err error) error {
	return nil
}
