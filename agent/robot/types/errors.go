package types

import "errors"

var (
	// Config errors
	ErrMissingIdentity    = errors.New("identity.role is required")
	ErrClockTimesEmpty    = errors.New("clock.times is required for times mode")
	ErrClockIntervalEmpty = errors.New("clock.every is required for interval mode")
	ErrClockModeInvalid   = errors.New("clock.mode must be times, interval, or daemon")

	// Runtime errors
	ErrRobotNotFound      = errors.New("robot not found")
	ErrRobotPaused        = errors.New("robot is paused")
	ErrRobotBusy          = errors.New("robot has reached max concurrent executions")
	ErrTriggerDisabled    = errors.New("trigger type is disabled for this robot")
	ErrExecutionCancelled = errors.New("execution was cancelled")
	ErrExecutionTimeout   = errors.New("execution timed out")

	// Phase errors
	ErrPhaseAgentNotFound = errors.New("phase agent not found")
	ErrGoalGenFailed      = errors.New("goal generation failed")
	ErrTaskPlanFailed     = errors.New("task planning failed")
	ErrDeliveryFailed     = errors.New("delivery failed")
)
