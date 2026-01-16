package types

import (
	"time"

	agentcontext "github.com/yaoapp/yao/agent/context"
)

// InterveneRequest - human intervention request
type InterveneRequest struct {
	TeamID       string                 `json:"team_id"`
	MemberID     string                 `json:"member_id"`
	Action       InterventionAction     `json:"action"`
	Messages     []agentcontext.Message `json:"messages"`                // user input (text, images, files)
	PlanTime     *time.Time             `json:"plan_time,omitempty"`     // for action=plan
	ExecutorMode ExecutorMode           `json:"executor_mode,omitempty"` // optional: override robot config
}

// EventRequest - event trigger request
type EventRequest struct {
	MemberID     string                 `json:"member_id"`
	Source       string                 `json:"source"`     // webhook path or table name
	EventType    string                 `json:"event_type"` // lead.created, etc.
	Data         map[string]interface{} `json:"data"`
	ExecutorMode ExecutorMode           `json:"executor_mode,omitempty"` // optional: override robot config
}

// ExecutionResult - trigger result
type ExecutionResult struct {
	ExecutionID string     `json:"execution_id"`
	Status      ExecStatus `json:"status"`
	Message     string     `json:"message,omitempty"`
}

// RobotState - robot status query result
type RobotState struct {
	MemberID    string      `json:"member_id"`
	TeamID      string      `json:"team_id"`
	DisplayName string      `json:"display_name"`
	Status      RobotStatus `json:"status"`
	Running     int         `json:"running"`     // current running execution count
	MaxRunning  int         `json:"max_running"` // max concurrent allowed
	LastRun     *time.Time  `json:"last_run,omitempty"`
	NextRun     *time.Time  `json:"next_run,omitempty"`
	RunningIDs  []string    `json:"running_ids,omitempty"` // list of running execution IDs
}
