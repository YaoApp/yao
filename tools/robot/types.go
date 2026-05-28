package robot

// ==================== Tool-layer input structs (whitelist) ====================
// These structs define EXACTLY which fields LLMs are allowed to pass.
// Unknown fields are silently dropped during json.Unmarshal.

// CreateRequest defines the allowed fields for robot_create.
type CreateRequest struct {
	DisplayName    string           `json:"display_name"`
	Bio            string           `json:"bio,omitempty"`
	SystemPrompt   string           `json:"system_prompt,omitempty"`
	Agents         []string         `json:"agents,omitempty"`
	Workspace      string           `json:"workspace,omitempty"`
	AutonomousMode *bool            `json:"autonomous_mode,omitempty"`
	RobotConfig    *ToolRobotConfig `json:"robot_config,omitempty"`
}

// UpdateRequest defines the allowed fields for robot_update.
type UpdateRequest struct {
	DisplayName    *string          `json:"display_name,omitempty"`
	Bio            *string          `json:"bio,omitempty"`
	SystemPrompt   *string          `json:"system_prompt,omitempty"`
	Agents         []string         `json:"agents,omitempty"`
	Workspace      *string          `json:"workspace,omitempty"`
	AutonomousMode *bool            `json:"autonomous_mode,omitempty"`
	RobotConfig    *ToolRobotConfig `json:"robot_config,omitempty"`
}

// TriggerRequest defines the allowed fields for robot_execution_create.
type TriggerRequest struct {
	Type      string                 `json:"type"`
	Messages  []TriggerMessage       `json:"messages,omitempty"`
	Source    string                 `json:"source,omitempty"`
	EventType string                 `json:"event_type,omitempty"`
	Data      map[string]interface{} `json:"data,omitempty"`
}

// TriggerMessage is a single message in a trigger request.
type TriggerMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// ==================== robot_config whitelist ====================
// Only safe sub-configs are exposed. Dangerous fields are excluded:
//   integrations (bot tokens, secrets), kb, db, learn, resources,
//   delivery, events — all blocked at the struct level.

// ToolRobotConfig is the whitelist subset of agent/robot/types.Config.
type ToolRobotConfig struct {
	Identity      *ToolIdentity `json:"identity,omitempty"`
	Quota         *ToolQuota    `json:"quota,omitempty"`
	Clock         *ToolClock    `json:"clock,omitempty"`
	Triggers      *ToolTriggers `json:"triggers,omitempty"`
	Executor      *ToolExecutor `json:"executor,omitempty"`
	DefaultLocale string        `json:"default_locale,omitempty"`
}

// ToolIdentity maps to types.Identity — role, duties, rules.
type ToolIdentity struct {
	Role   string   `json:"role"`
	Duties []string `json:"duties,omitempty"`
	Rules  []string `json:"rules,omitempty"`
}

// ToolQuota maps to types.Quota — concurrency limits.
type ToolQuota struct {
	Max      int `json:"max,omitempty"`
	Queue    int `json:"queue,omitempty"`
	Priority int `json:"priority,omitempty"`
}

// ToolClock maps to types.Clock — scheduling.
type ToolClock struct {
	Mode    string   `json:"mode"`
	Times   []string `json:"times,omitempty"`
	Days    []string `json:"days,omitempty"`
	Every   string   `json:"every,omitempty"`
	TZ      string   `json:"tz,omitempty"`
	Timeout string   `json:"timeout,omitempty"`
}

// ToolTriggers maps to types.Triggers — enable/disable triggers.
type ToolTriggers struct {
	Clock     *ToolTriggerSwitch `json:"clock,omitempty"`
	Intervene *ToolTriggerSwitch `json:"intervene,omitempty"`
	Event     *ToolTriggerSwitch `json:"event,omitempty"`
}

// ToolTriggerSwitch maps to types.TriggerSwitch.
type ToolTriggerSwitch struct {
	Enabled bool     `json:"enabled"`
	Actions []string `json:"actions,omitempty"`
}

// ToolExecutor maps to types.ExecutorConfig — execution mode.
type ToolExecutor struct {
	Mode        string `json:"mode,omitempty"`
	MaxDuration string `json:"max_duration,omitempty"`
}

// ==================== Query types ====================

type ListQuery struct {
	TeamID   string
	Status   string
	Keywords string
	Page     int
	PageSize int
}

type ExecutionQuery struct {
	Status   string
	Page     int
	PageSize int
}

type ResultQuery struct {
	Page     int
	PageSize int
}

// ==================== Response types ====================

type RobotSummary struct {
	MemberID       string `json:"member_id"`
	DisplayName    string `json:"display_name"`
	Bio            string `json:"bio"`
	Status         string `json:"status"`
	AutonomousMode bool   `json:"autonomous_mode"`
	Running        int    `json:"running"`
}

type ListResult struct {
	Data     []RobotSummary `json:"data"`
	Total    int            `json:"total"`
	Page     int            `json:"page"`
	PageSize int            `json:"pagesize"`
}

type RobotResponse struct {
	Data         interface{} `json:"data"`
	YaoTeamID    string      `json:"-"`
	YaoCreatedBy string      `json:"-"`
}

type RobotState struct {
	MemberID     string   `json:"member_id"`
	TeamID       string   `json:"team_id"`
	DisplayName  string   `json:"display_name"`
	Bio          string   `json:"bio,omitempty"`
	Status       string   `json:"status"`
	Running      int      `json:"running"`
	MaxRunning   int      `json:"max_running"`
	RunningIDs   []string `json:"running_ids,omitempty"`
	LastRun      string   `json:"last_run,omitempty"`
	NextRun      string   `json:"next_run,omitempty"`
	YaoTeamID    string   `json:"-"`
	YaoCreatedBy string   `json:"-"`
}

type TriggerResult struct {
	ExecutionID string `json:"execution_id"`
	Accepted    bool   `json:"accepted"`
	Message     string `json:"message,omitempty"`
}

type ExecutionResult struct {
	Data     interface{} `json:"data"`
	Total    int         `json:"total"`
	Page     int         `json:"page"`
	PageSize int         `json:"pagesize"`
}

type ExecutionDetail struct {
	Data interface{} `json:"data"`
}

type ResultListResponse struct {
	Data     interface{} `json:"data"`
	Total    int         `json:"total"`
	Page     int         `json:"page"`
	PageSize int         `json:"pagesize"`
}
