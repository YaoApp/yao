package types

import "errors"

// ErrMissingIdentity indicates identity.role is required
var ErrMissingIdentity = errors.New("identity.role is required")

// ErrClockTimesEmpty indicates clock.times is required for times mode
var ErrClockTimesEmpty = errors.New("clock.times is required for times mode")

// ErrClockIntervalEmpty indicates clock.every is required for interval mode
var ErrClockIntervalEmpty = errors.New("clock.every is required for interval mode")

// ErrClockModeInvalid indicates clock.mode must be times, interval, or daemon
var ErrClockModeInvalid = errors.New("clock.mode must be times, interval, or daemon")

// ErrRobotNotFound indicates robot not found
var ErrRobotNotFound = errors.New("robot not found")

// ErrRobotPaused indicates robot is paused
var ErrRobotPaused = errors.New("robot is paused")

// ErrRobotBusy indicates robot has reached max concurrent executions
var ErrRobotBusy = errors.New("robot has reached max concurrent executions")

// ErrQuotaExceeded indicates robot quota was exceeded (atomic check failed)
var ErrQuotaExceeded = errors.New("robot quota exceeded")

// ErrTriggerDisabled indicates trigger type is disabled for this robot
var ErrTriggerDisabled = errors.New("trigger type is disabled for this robot")

// ErrExecutionCancelled indicates execution was cancelled
var ErrExecutionCancelled = errors.New("execution was cancelled")

// ErrExecutionTimeout indicates execution timed out
var ErrExecutionTimeout = errors.New("execution timed out")

// ErrPhaseAgentNotFound indicates phase agent not found
var ErrPhaseAgentNotFound = errors.New("phase agent not found")

// ErrGoalGenFailed indicates goal generation failed
var ErrGoalGenFailed = errors.New("goal generation failed")

// ErrTaskPlanFailed indicates task planning failed
var ErrTaskPlanFailed = errors.New("task planning failed")

// ErrDeliveryFailed indicates delivery failed
var ErrDeliveryFailed = errors.New("delivery failed")
