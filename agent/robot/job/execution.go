package job

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/yaoapp/gou/model"
	yaojob "github.com/yaoapp/yao/job"

	"github.com/yaoapp/yao/agent/robot/types"
)

// CreateOptions holds options for creating a new execution
type CreateOptions struct {
	Robot       *types.Robot        // Required: the robot to execute
	TriggerType types.TriggerType   // Required: clock | human | event
	Input       *types.TriggerInput // Optional: trigger input data

	// Optional fields for future extension
	Priority          int                    // Execution priority (higher = more important)
	TimeoutSeconds    *int                   // Execution timeout
	ParentExecutionID string                 // Parent execution ID for sub-tasks
	ScheduledAt       *time.Time             // Scheduled execution time (for delayed execution)
	Metadata          map[string]interface{} // Custom metadata
}

// Validate validates the CreateOptions
func (o *CreateOptions) Validate() error {
	if o.Robot == nil {
		return fmt.Errorf("robot is required")
	}
	if o.TriggerType == "" {
		return fmt.Errorf("trigger type is required")
	}
	return nil
}

// CreateExecution creates a new execution record in the job system
// This creates both the robot Execution and the corresponding job.Execution
func CreateExecution(ctx *types.Context, opts *CreateOptions) (*types.Execution, error) {
	if opts == nil {
		return nil, fmt.Errorf("options is nil")
	}
	if err := opts.Validate(); err != nil {
		return nil, err
	}

	robot := opts.Robot
	triggerType := opts.TriggerType

	// Create job for this execution (returns jobID and execID)
	jobID, execID, err := Create(ctx, &Options{
		Robot:       robot,
		TriggerType: triggerType,
		Priority:    opts.Priority,
		Metadata:    opts.Metadata,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create job: %w", err)
	}

	// Determine starting phase based on trigger type
	// Clock trigger starts from P0 (Inspiration)
	// Human/Event triggers skip P0 and start from P1 (Goals)
	startPhase := types.PhaseInspiration
	if triggerType == types.TriggerHuman || triggerType == types.TriggerEvent {
		startPhase = types.PhaseGoals
	}

	// Create robot execution
	exec := &types.Execution{
		ID:          execID,
		MemberID:    robot.MemberID,
		TeamID:      robot.TeamID,
		TriggerType: triggerType,
		StartTime:   time.Now(),
		Status:      types.ExecPending,
		Phase:       startPhase,
		Input:       opts.Input,
		JobID:       jobID,
	}

	// Build trigger context for job execution
	triggerContext, _ := json.Marshal(map[string]interface{}{
		"trigger_type": string(triggerType),
		"member_id":    robot.MemberID,
		"team_id":      robot.TeamID,
	})
	triggerContextRaw := json.RawMessage(triggerContext)

	// Map trigger type to trigger category
	// TriggerCategory ENUM: manual, scheduled, event, api, system, dependency
	triggerCategory := mapTriggerTypeToCategory(triggerType)
	triggerSource := string(triggerType) // Store original trigger type as source

	// Create job execution record
	jobExec := &yaojob.Execution{
		ExecutionID:     execID,
		JobID:           jobID,
		Status:          "queued",
		TriggerCategory: triggerCategory,
		TriggerSource:   &triggerSource,
		TriggerContext:  &triggerContextRaw,
		Progress:        0,
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
	}

	// Apply optional fields
	if opts.TimeoutSeconds != nil {
		jobExec.TimeoutSeconds = opts.TimeoutSeconds
	}
	if opts.ParentExecutionID != "" {
		jobExec.ParentExecutionID = &opts.ParentExecutionID
	}
	if opts.ScheduledAt != nil {
		jobExec.ScheduledAt = opts.ScheduledAt
	}
	if opts.Priority > 0 {
		jobExec.ExecutionOptions = &yaojob.ExecutionOptions{
			Priority: opts.Priority,
		}
	}

	if err := yaojob.SaveExecution(jobExec); err != nil {
		return nil, fmt.Errorf("failed to save job execution: %w", err)
	}

	// Note: Job status is automatically updated by yaojob.SaveExecution -> updateJobProgress
	// No need to manually set job status here

	return exec, nil
}

// UpdatePhase updates the execution phase in the job system
func UpdatePhase(ctx *types.Context, exec *types.Execution, phase types.Phase) error {
	if exec == nil || exec.ID == "" || exec.JobID == "" {
		return fmt.Errorf("invalid execution or missing execution/job ID")
	}

	exec.Phase = phase

	// Update job
	if err := Update(ctx, exec); err != nil {
		return fmt.Errorf("failed to update job: %w", err)
	}

	// Update job execution progress
	progress := phaseToProgress(phase)
	if err := updateExecutionProgress(exec.ID, progress, string(phase)); err != nil {
		return fmt.Errorf("failed to update execution progress: %w", err)
	}

	// Log phase transition (ignore error, non-critical)
	_ = LogPhaseStart(ctx, exec, phase)

	return nil
}

// UpdateStatus updates the execution status in the job system
func UpdateStatus(ctx *types.Context, exec *types.Execution, status types.ExecStatus) error {
	if exec == nil || exec.ID == "" || exec.JobID == "" {
		return fmt.Errorf("invalid execution or missing execution/job ID")
	}

	exec.Status = status

	// Update job
	if err := Update(ctx, exec); err != nil {
		return fmt.Errorf("failed to update job: %w", err)
	}

	// Update job execution status
	if err := updateExecutionStatus(exec.ID, status); err != nil {
		return fmt.Errorf("failed to update execution status: %w", err)
	}

	return nil
}

// CompleteExecution marks execution as completed
func CompleteExecution(ctx *types.Context, exec *types.Execution) error {
	if exec == nil || exec.ID == "" || exec.JobID == "" {
		return fmt.Errorf("invalid execution or missing execution/job ID")
	}

	now := time.Now()
	exec.EndTime = &now
	exec.Status = types.ExecCompleted

	// Complete the job
	if err := Complete(ctx, exec); err != nil {
		return fmt.Errorf("failed to complete job: %w", err)
	}

	// Update job execution
	if err := completeJobExecution(exec.ID, exec.StartTime); err != nil {
		return fmt.Errorf("failed to complete job execution: %w", err)
	}

	// Log completion (ignore error, non-critical)
	locale := getLocale(ctx)
	var msg string
	if isChineseLocale(locale) {
		msg = "执行完成"
	} else {
		msg = "Execution completed successfully"
	}
	_ = Log(ctx, exec, "info", msg, map[string]interface{}{
		"duration_ms": now.Sub(exec.StartTime).Milliseconds(),
	})

	return nil
}

// FailExecution marks execution as failed
func FailExecution(ctx *types.Context, exec *types.Execution, execErr error) error {
	if exec == nil || exec.ID == "" || exec.JobID == "" {
		return fmt.Errorf("invalid execution or missing execution/job ID")
	}

	now := time.Now()
	exec.EndTime = &now
	exec.Status = types.ExecFailed
	if execErr != nil {
		exec.Error = execErr.Error()
	}

	// Fail the job
	if err := Fail(ctx, exec, execErr); err != nil {
		return fmt.Errorf("failed to fail job: %w", err)
	}

	// Update job execution
	if err := failJobExecution(exec.ID, execErr, exec.StartTime); err != nil {
		return fmt.Errorf("failed to fail job execution: %w", err)
	}

	// Log failure (ignore error, non-critical)
	_ = LogError(ctx, exec, execErr)

	return nil
}

// CancelExecution marks execution as cancelled
func CancelExecution(ctx *types.Context, exec *types.Execution) error {
	if exec == nil || exec.ID == "" || exec.JobID == "" {
		return fmt.Errorf("invalid execution or missing execution/job ID")
	}

	now := time.Now()
	exec.EndTime = &now
	exec.Status = types.ExecCancelled

	// Cancel the job
	if err := Cancel(ctx, exec); err != nil {
		return fmt.Errorf("failed to cancel job: %w", err)
	}

	// Update job execution
	if err := cancelJobExecution(exec.ID, exec.StartTime); err != nil {
		return fmt.Errorf("failed to cancel job execution: %w", err)
	}

	// Log cancellation (ignore error, non-critical)
	locale := getLocale(ctx)
	var msg string
	if isChineseLocale(locale) {
		msg = "执行已取消"
	} else {
		msg = "Execution cancelled"
	}
	_ = Log(ctx, exec, "info", msg, nil)

	return nil
}

// GetExecution retrieves a job execution by ID
func GetExecution(executionID string) (*yaojob.Execution, error) {
	if executionID == "" {
		return nil, fmt.Errorf("execution ID is empty")
	}
	return yaojob.GetExecution(executionID, model.QueryParam{})
}

// ListExecutions lists executions for a job
func ListExecutions(jobID string) ([]*yaojob.Execution, error) {
	if jobID == "" {
		return nil, fmt.Errorf("job ID is empty")
	}
	return yaojob.GetExecutions(jobID)
}

// phaseToProgress maps phase to progress percentage
func phaseToProgress(phase types.Phase) int {
	switch phase {
	case types.PhaseInspiration:
		return 10
	case types.PhaseGoals:
		return 25
	case types.PhaseTasks:
		return 40
	case types.PhaseRun:
		return 60
	case types.PhaseDelivery:
		return 80
	case types.PhaseLearning:
		return 95
	default:
		return 0
	}
}

// updateExecutionProgress updates the job execution progress
// Note: step parameter is kept for future use if yaojob.Execution adds Step field
func updateExecutionProgress(executionID string, progress int, _ string) error {
	exec, err := yaojob.GetExecution(executionID, model.QueryParam{})
	if err != nil {
		return err
	}

	exec.Progress = progress
	exec.UpdatedAt = time.Now()

	return yaojob.SaveExecution(exec)
}

// updateExecutionStatus updates the job execution status
func updateExecutionStatus(executionID string, status types.ExecStatus) error {
	exec, err := yaojob.GetExecution(executionID, model.QueryParam{})
	if err != nil {
		return err
	}

	exec.Status = mapStatusToJobStatus(status)
	exec.UpdatedAt = time.Now()

	if status == types.ExecRunning && exec.StartedAt == nil {
		now := time.Now()
		exec.StartedAt = &now
	}

	return yaojob.SaveExecution(exec)
}

// completeJobExecution marks job execution as completed
func completeJobExecution(executionID string, startTime time.Time) error {
	exec, err := yaojob.GetExecution(executionID, model.QueryParam{})
	if err != nil {
		return err
	}

	now := time.Now()
	exec.Status = "completed"
	exec.Progress = 100
	exec.EndedAt = &now
	exec.UpdatedAt = now

	// Calculate duration (handle zero startTime)
	if !startTime.IsZero() {
		duration := int(now.Sub(startTime).Milliseconds())
		exec.Duration = &duration
	}

	return yaojob.SaveExecution(exec)
}

// failJobExecution marks job execution as failed
func failJobExecution(executionID string, execErr error, startTime time.Time) error {
	exec, err := yaojob.GetExecution(executionID, model.QueryParam{})
	if err != nil {
		return err
	}

	now := time.Now()
	exec.Status = "failed"
	exec.EndedAt = &now
	exec.UpdatedAt = now

	// Calculate duration (handle zero startTime)
	if !startTime.IsZero() {
		duration := int(now.Sub(startTime).Milliseconds())
		exec.Duration = &duration
	}

	// Store error info
	if execErr != nil {
		errorInfo, _ := json.Marshal(map[string]string{
			"message": execErr.Error(),
		})
		errorInfoRaw := json.RawMessage(errorInfo)
		exec.ErrorInfo = &errorInfoRaw
	}

	return yaojob.SaveExecution(exec)
}

// cancelJobExecution marks job execution as cancelled
func cancelJobExecution(executionID string, startTime time.Time) error {
	exec, err := yaojob.GetExecution(executionID, model.QueryParam{})
	if err != nil {
		return err
	}

	now := time.Now()
	exec.Status = "cancelled"
	exec.EndedAt = &now
	exec.UpdatedAt = now

	// Calculate duration (handle zero startTime)
	if !startTime.IsZero() {
		duration := int(now.Sub(startTime).Milliseconds())
		exec.Duration = &duration
	}

	return yaojob.SaveExecution(exec)
}

// mapTriggerTypeToCategory maps robot TriggerType to job execution TriggerCategory
// TriggerCategory ENUM values: manual, scheduled, event, api, system, dependency
func mapTriggerTypeToCategory(triggerType types.TriggerType) string {
	switch triggerType {
	case types.TriggerClock:
		return "scheduled"
	case types.TriggerHuman:
		return "manual"
	case types.TriggerEvent:
		return "event"
	default:
		return "system"
	}
}
