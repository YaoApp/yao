package job

import (
	"fmt"
	"strings"

	gonanoid "github.com/matoous/go-nanoid/v2"
	yaojob "github.com/yaoapp/yao/job"

	"github.com/yaoapp/yao/agent/robot/types"
)

// CategoryID is the job category for robot executions
const CategoryID = "autonomous_robot"

// JobIDPrefix is the prefix for robot job IDs
const JobIDPrefix = "robot_exec_"

// Options holds options for creating a new job
type Options struct {
	Robot       *types.Robot      // Required: the robot to execute
	TriggerType types.TriggerType // Required: clock | human | event

	// Optional fields for future extension
	Priority       int                    // Job priority (higher = more important)
	MaxRetryCount  int                    // Max retry count on failure
	DefaultTimeout *int                   // Default execution timeout in seconds
	Metadata       map[string]interface{} // Custom metadata stored in job config
}

// Validate validates the Options
func (o *Options) Validate() error {
	if o.Robot == nil {
		return fmt.Errorf("robot is required")
	}
	if o.TriggerType == "" {
		return fmt.Errorf("trigger type is required")
	}
	return nil
}

// Create creates a new job for robot execution
// Returns the job ID (format: robot_exec_{execID}) and the generated execution ID
func Create(ctx *types.Context, opts *Options) (jobID string, execID string, err error) {
	if opts == nil {
		return "", "", fmt.Errorf("options is nil")
	}
	if err := opts.Validate(); err != nil {
		return "", "", err
	}

	robot := opts.Robot
	triggerType := opts.TriggerType

	// Generate execution ID
	execID, err = gonanoid.New()
	if err != nil {
		return "", "", fmt.Errorf("failed to generate execution ID: %w", err)
	}

	// Create job ID: robot_exec_{execID}
	jobID = JobIDPrefix + execID

	// Get locale from context
	locale := getLocale(ctx)

	// Build job name based on locale
	// Use robot display name for better readability in Activity Monitor
	displayName := robot.DisplayName
	if displayName == "" {
		displayName = robot.MemberID
	}
	name := buildJobName(locale, triggerType, displayName)

	// Build job config
	jobConfig := map[string]interface{}{
		"member_id":    robot.MemberID,
		"team_id":      robot.TeamID,
		"trigger_type": string(triggerType),
		"exec_id":      execID,
		"display_name": displayName,
	}

	// Merge custom metadata into config
	if opts.Metadata != nil {
		for k, v := range opts.Metadata {
			jobConfig[k] = v
		}
	}

	// Build job params
	jobParams := map[string]interface{}{
		"job_id":        jobID,
		"category_name": getCategoryName(locale),
		"name":          name,
		"config":        jobConfig,
	}

	// Apply optional fields
	if opts.Priority > 0 {
		jobParams["priority"] = opts.Priority
	}
	if opts.MaxRetryCount > 0 {
		jobParams["max_retry_count"] = opts.MaxRetryCount
	}
	if opts.DefaultTimeout != nil {
		jobParams["default_timeout"] = *opts.DefaultTimeout
	}

	// Create job using yao/job package
	j, err := yaojob.Once(yaojob.GOROUTINE, jobParams)
	if err != nil {
		return "", "", fmt.Errorf("failed to create job: %w", err)
	}

	// Save job to database
	if err := yaojob.SaveJob(j); err != nil {
		return "", "", fmt.Errorf("failed to save job: %w", err)
	}

	return jobID, execID, nil
}

// Get retrieves a job by job ID
func Get(jobID string) (*yaojob.Job, error) {
	if jobID == "" {
		return nil, fmt.Errorf("job ID is empty")
	}
	return yaojob.GetJob(jobID)
}

// Update updates job status and phase
func Update(ctx *types.Context, exec *types.Execution) error {
	if exec == nil || exec.JobID == "" {
		return fmt.Errorf("invalid execution or missing job ID")
	}

	j, err := yaojob.GetJob(exec.JobID)
	if err != nil {
		return fmt.Errorf("failed to get job: %w", err)
	}

	// Map robot status to job status
	jobStatus := mapStatusToJobStatus(exec.Status)
	j.Status = jobStatus

	// Update config with current phase
	if j.Config == nil {
		j.Config = make(map[string]interface{})
	}
	j.Config["current_phase"] = string(exec.Phase)
	j.Config["current_status"] = string(exec.Status)

	if err := yaojob.SaveJob(j); err != nil {
		return fmt.Errorf("failed to update job: %w", err)
	}

	return nil
}

// Complete marks job as completed
func Complete(ctx *types.Context, exec *types.Execution) error {
	if exec == nil || exec.JobID == "" {
		return fmt.Errorf("invalid execution or missing job ID")
	}

	j, err := yaojob.GetJob(exec.JobID)
	if err != nil {
		return fmt.Errorf("failed to get job: %w", err)
	}

	j.Status = "completed"

	// Update config with final state
	if j.Config == nil {
		j.Config = make(map[string]interface{})
	}
	j.Config["current_phase"] = string(types.PhaseLearning)
	j.Config["current_status"] = string(types.ExecCompleted)

	if exec.Delivery != nil {
		j.Config["delivery_success"] = exec.Delivery.Success
	}

	if err := yaojob.SaveJob(j); err != nil {
		return fmt.Errorf("failed to complete job: %w", err)
	}

	return nil
}

// Fail marks job as failed
func Fail(ctx *types.Context, exec *types.Execution, execErr error) error {
	if exec == nil || exec.JobID == "" {
		return fmt.Errorf("invalid execution or missing job ID")
	}

	j, err := yaojob.GetJob(exec.JobID)
	if err != nil {
		return fmt.Errorf("failed to get job: %w", err)
	}

	j.Status = "failed"

	// Update config with error info
	if j.Config == nil {
		j.Config = make(map[string]interface{})
	}
	j.Config["current_phase"] = string(exec.Phase)
	j.Config["current_status"] = string(types.ExecFailed)
	if execErr != nil {
		j.Config["error"] = execErr.Error()
	}

	if err := yaojob.SaveJob(j); err != nil {
		return fmt.Errorf("failed to fail job: %w", err)
	}

	return nil
}

// Cancel marks job as cancelled
func Cancel(ctx *types.Context, exec *types.Execution) error {
	if exec == nil || exec.JobID == "" {
		return fmt.Errorf("invalid execution or missing job ID")
	}

	j, err := yaojob.GetJob(exec.JobID)
	if err != nil {
		return fmt.Errorf("failed to get job: %w", err)
	}

	j.Status = "cancelled"

	// Update config with cancelled state
	if j.Config == nil {
		j.Config = make(map[string]interface{})
	}
	j.Config["current_phase"] = string(exec.Phase)
	j.Config["current_status"] = string(types.ExecCancelled)

	if err := yaojob.SaveJob(j); err != nil {
		return fmt.Errorf("failed to cancel job: %w", err)
	}

	return nil
}

// mapStatusToJobStatus maps robot ExecStatus to job status string
// Job model ENUM values: draft, ready, queued, running, paused, completed, failed, cancelled, disabled
func mapStatusToJobStatus(status types.ExecStatus) string {
	switch status {
	case types.ExecPending:
		return "queued"
	case types.ExecRunning:
		return "running"
	case types.ExecCompleted:
		return "completed"
	case types.ExecFailed:
		return "failed"
	case types.ExecCancelled:
		return "cancelled"
	default:
		return "draft"
	}
}

// getLocale returns the locale from context, defaults to "en-US"
func getLocale(ctx *types.Context) string {
	if ctx == nil || ctx.Locale == "" {
		return "en-US"
	}
	return ctx.Locale
}

// isChineseLocale checks if the locale is Chinese
func isChineseLocale(locale string) bool {
	return strings.HasPrefix(strings.ToLower(locale), "zh")
}

// buildJobName builds the job name based on locale
func buildJobName(locale string, triggerType types.TriggerType, displayName string) string {
	var name string
	if isChineseLocale(locale) {
		name = fmt.Sprintf("机器人执行 - %s", getTriggerTypeName(locale, triggerType))
	} else {
		name = fmt.Sprintf("Robot Execution - %s", getTriggerTypeName(locale, triggerType))
	}
	if displayName != "" {
		name = fmt.Sprintf("%s (%s)", name, displayName)
	}
	return name
}

// getCategoryName returns the category name based on locale
func getCategoryName(locale string) string {
	if isChineseLocale(locale) {
		return "自主机器人"
	}
	return "Autonomous Robot"
}

// getTriggerTypeName returns the trigger type name based on locale
func getTriggerTypeName(locale string, triggerType types.TriggerType) string {
	if isChineseLocale(locale) {
		switch triggerType {
		case types.TriggerClock:
			return "定时触发"
		case types.TriggerHuman:
			return "人工触发"
		case types.TriggerEvent:
			return "事件触发"
		default:
			return string(triggerType)
		}
	}
	switch triggerType {
	case types.TriggerClock:
		return "Clock"
	case types.TriggerHuman:
		return "Human"
	case types.TriggerEvent:
		return "Event"
	default:
		return string(triggerType)
	}
}

// getPhaseName returns the phase name based on locale
func getPhaseName(locale string, phase types.Phase) string {
	if isChineseLocale(locale) {
		switch phase {
		case types.PhaseInspiration:
			return "灵感收集"
		case types.PhaseGoals:
			return "目标生成"
		case types.PhaseTasks:
			return "任务规划"
		case types.PhaseRun:
			return "任务执行"
		case types.PhaseDelivery:
			return "结果交付"
		case types.PhaseLearning:
			return "学习总结"
		default:
			return string(phase)
		}
	}
	switch phase {
	case types.PhaseInspiration:
		return "Inspiration"
	case types.PhaseGoals:
		return "Goals"
	case types.PhaseTasks:
		return "Tasks"
	case types.PhaseRun:
		return "Run"
	case types.PhaseDelivery:
		return "Delivery"
	case types.PhaseLearning:
		return "Learning"
	default:
		return string(phase)
	}
}
