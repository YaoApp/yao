package types

import (
	"context"
	"fmt"
	"sync"
	"time"

	agentcontext "github.com/yaoapp/yao/agent/context"
)

// Robot - runtime representation of an autonomous robot (from __yao.member)
// Relationship: 1 Robot : N Executions (concurrent)
// Each trigger creates a new Execution (stored in __yao.agent_execution)
type Robot struct {
	// From __yao.member
	MemberID       string      `json:"member_id"`
	TeamID         string      `json:"team_id"`
	DisplayName    string      `json:"display_name"`
	Bio            string      `json:"bio"` // Robot's description (from __yao.member.bio)
	SystemPrompt   string      `json:"system_prompt"`
	Status         RobotStatus `json:"robot_status"`
	AutonomousMode bool        `json:"autonomous_mode"`
	RobotEmail     string      `json:"robot_email"` // Robot's email address for sending emails

	// Manager info (from __yao.member)
	ManagerID    string `json:"manager_id"`    // Direct manager user_id (who manages this robot)
	ManagerEmail string `json:"manager_email"` // Manager's email address (for default delivery)

	// Parsed config (from robot_config JSON field)
	Config *Config `json:"-"`

	// Runtime state
	LastRun time.Time `json:"-"` // last execution start time
	NextRun time.Time `json:"-"` // next scheduled execution (for clock trigger)

	// Concurrency control
	// Each Robot can run multiple Executions concurrently (up to Quota.Max)
	executions map[string]*Execution // execID -> Execution
	execMu     sync.RWMutex
}

// CanRun checks if robot can accept new execution
// Note: This is a read-only check. For atomic check-and-acquire, use TryAcquireSlot()
func (r *Robot) CanRun() bool {
	r.execMu.RLock()
	defer r.execMu.RUnlock()
	if r.Config == nil {
		return len(r.executions) < 2 // default max
	}
	return len(r.executions) < r.Config.Quota.GetMax()
}

// TryAcquireSlot atomically checks if robot can run and reserves a slot
// Returns true if slot was acquired, false if quota is full
// This prevents race conditions between CanRun() check and AddExecution()
func (r *Robot) TryAcquireSlot(exec *Execution) bool {
	r.execMu.Lock()
	defer r.execMu.Unlock()

	// Get max quota
	maxQuota := 2 // default
	if r.Config != nil {
		maxQuota = r.Config.Quota.GetMax()
	}

	// Check if we can add
	if len(r.executions) >= maxQuota {
		return false // quota full
	}

	// Reserve slot by adding execution
	if r.executions == nil {
		r.executions = make(map[string]*Execution)
	}
	r.executions[exec.ID] = exec
	return true
}

// RunningCount returns current running execution count
func (r *Robot) RunningCount() int {
	r.execMu.RLock()
	defer r.execMu.RUnlock()
	return len(r.executions)
}

// AddExecution adds an execution to tracking
// Note: Prefer TryAcquireSlot() for atomic check-and-add
func (r *Robot) AddExecution(exec *Execution) {
	r.execMu.Lock()
	defer r.execMu.Unlock()
	if r.executions == nil {
		r.executions = make(map[string]*Execution)
	}
	r.executions[exec.ID] = exec
}

// RemoveExecution removes an execution from tracking
func (r *Robot) RemoveExecution(execID string) {
	r.execMu.Lock()
	defer r.execMu.Unlock()
	delete(r.executions, execID)
}

// GetExecution returns an execution by ID
func (r *Robot) GetExecution(execID string) *Execution {
	r.execMu.RLock()
	defer r.execMu.RUnlock()
	return r.executions[execID]
}

// GetExecutions returns all running executions
func (r *Robot) GetExecutions() []*Execution {
	r.execMu.RLock()
	defer r.execMu.RUnlock()
	execs := make([]*Execution, 0, len(r.executions))
	for _, exec := range r.executions {
		execs = append(execs, exec)
	}
	return execs
}

// Execution - single execution instance
// Each trigger creates a new Execution, stored in ExecutionStore
type Execution struct {
	ID          string      `json:"id"`        // unique execution ID
	MemberID    string      `json:"member_id"` // robot member ID
	TeamID      string      `json:"team_id"`
	TriggerType TriggerType `json:"trigger_type"` // clock | human | event
	StartTime   time.Time   `json:"start_time"`
	EndTime     *time.Time  `json:"end_time,omitempty"`
	Status      ExecStatus  `json:"status"`
	Phase       Phase       `json:"phase"`
	Error       string      `json:"error,omitempty"`

	// UI display fields (updated by executor at each phase)
	// These provide human-readable status for frontend display
	Name            string `json:"name,omitempty"`              // Execution title (updated when goals complete)
	CurrentTaskName string `json:"current_task_name,omitempty"` // Current task description (updated during run phase)

	// Trigger input (stored for traceability)
	Input *TriggerInput `json:"input,omitempty"` // original trigger input

	// Phase outputs
	Inspiration *InspirationReport `json:"inspiration,omitempty"` // P0: markdown
	Goals       *Goals             `json:"goals,omitempty"`       // P1: markdown
	Tasks       []Task             `json:"tasks,omitempty"`       // P2: structured tasks
	Current     *CurrentState      `json:"current,omitempty"`     // current executing state
	Results     []TaskResult       `json:"results,omitempty"`     // P3: task results
	Delivery    *DeliveryResult    `json:"delivery,omitempty"`
	Learning    []LearningEntry    `json:"learning,omitempty"`

	// Runtime (internal, not serialized)
	ctx    context.Context    `json:"-"`
	cancel context.CancelFunc `json:"-"`
	robot  *Robot             `json:"-"`
}

// GetRobot returns the robot associated with this execution
func (e *Execution) GetRobot() *Robot {
	return e.robot
}

// SetRobot sets the robot associated with this execution
func (e *Execution) SetRobot(robot *Robot) {
	e.robot = robot
}

// TriggerInput - stored trigger input for traceability
type TriggerInput struct {
	// For human intervention
	Action   InterventionAction     `json:"action,omitempty"`   // task.add, goal.adjust, etc.
	Messages []agentcontext.Message `json:"messages,omitempty"` // user's input (text, images, files)
	UserID   string                 `json:"user_id,omitempty"`  // who triggered
	Locale   string                 `json:"locale,omitempty"`   // language for UI display (e.g., "en-US", "zh-CN")

	// For event trigger
	Source    EventSource            `json:"source,omitempty"`     // webhook | database
	EventType string                 `json:"event_type,omitempty"` // lead.created, etc.
	Data      map[string]interface{} `json:"data,omitempty"`       // event payload

	// For clock trigger
	Clock *ClockContext `json:"clock,omitempty"` // time context when triggered
}

// CurrentState - current executing goal and task
type CurrentState struct {
	Task      *Task  `json:"task,omitempty"`     // current task being executed
	TaskIndex int    `json:"task_index"`         // index in Tasks slice
	Progress  string `json:"progress,omitempty"` // human-readable progress (e.g., "2/5 tasks")
}

// Goals - P1 output (markdown for LLM + structured metadata)
// P1 Agent reads InspirationReport and generates goals as markdown
// Example:
// ## Goals
// 1. [High] Analyze sales data and identify trends
//   - Reason: Sales up 50%, need to understand why
//
// 2. [Normal] Prepare weekly report for manager
//   - Reason: Friday 5pm, weekly report due
//
// 3. [Low] Update CRM with new leads
//   - Reason: 3 pending leads from yesterday
type Goals struct {
	Content string `json:"content"` // markdown text

	// Delivery for P4 (where to send results)
	Delivery *DeliveryTarget `json:"delivery,omitempty"`
}

// DeliveryTarget - where to deliver results (defined in P1, used in P4)
type DeliveryTarget struct {
	Type       DeliveryType           `json:"type"`                 // email | webhook | report | notification
	Recipients []string               `json:"recipients,omitempty"` // email addresses, webhook URLs, user IDs
	Format     string                 `json:"format,omitempty"`     // markdown | html | json | text
	Template   string                 `json:"template,omitempty"`   // template name or inline template
	Options    map[string]interface{} `json:"options,omitempty"`    // channel-specific options
}

// Task - planned task (structured, for execution)
type Task struct {
	ID          string                 `json:"id"`
	Description string                 `json:"description,omitempty"` // human-readable task description (for UI display)
	Messages    []agentcontext.Message `json:"messages"`              // original input (text, images, files)
	GoalRef     string                 `json:"goal_ref,omitempty"`    // reference to goal (e.g., "Goal 1")
	Source      TaskSource             `json:"source"`                // auto | human | event

	// Executor
	ExecutorType ExecutorType `json:"executor_type"`
	ExecutorID   string       `json:"executor_id"` // unified ID: agent/assistant/process ID, or "mcp_server.mcp_tool" for MCP
	Args         []any        `json:"args,omitempty"`

	// MCP-specific fields (required when executor_type is "mcp")
	MCPServer string `json:"mcp_server,omitempty"` // MCP server/client ID (e.g., "ark.image.text2img")
	MCPTool   string `json:"mcp_tool,omitempty"`   // MCP tool name (e.g., "generate")

	// Validation (defined in P2, used in P3)
	// ExpectedOutput describes what the task should produce (for LLM semantic validation)
	ExpectedOutput string `json:"expected_output,omitempty"` // e.g., "JSON with sales_total, growth_rate fields"
	// ValidationRules are specific checks to perform (can be semantic or structural)
	ValidationRules []string `json:"validation_rules,omitempty"` // e.g., ["output must be valid JSON", "sales_total > 0"]

	// Runtime
	Status    TaskStatus `json:"status"`
	Order     int        `json:"order"` // execution order (0-based)
	StartTime *time.Time `json:"start_time,omitempty"`
	EndTime   *time.Time `json:"end_time,omitempty"`
}

// TaskResult - task execution result
type TaskResult struct {
	TaskID   string      `json:"task_id"`
	Success  bool        `json:"success"`
	Output   interface{} `json:"output,omitempty"`
	Error    string      `json:"error,omitempty"`
	Duration int64       `json:"duration_ms"`

	// Validation result (populated by P3)
	Validation *ValidationResult `json:"validation,omitempty"`
}

// ValidationResult - P3 semantic validation result
type ValidationResult struct {
	// Basic validation result
	Passed      bool     `json:"passed"`                // overall validation passed
	Score       float64  `json:"score,omitempty"`       // 0-1 confidence score
	Issues      []string `json:"issues,omitempty"`      // what failed
	Suggestions []string `json:"suggestions,omitempty"` // how to improve
	Details     string   `json:"details,omitempty"`     // detailed validation report (markdown)

	// Execution state (for multi-turn conversation control)
	Complete     bool   `json:"complete"`                // whether expected result is obtained
	NeedReply    bool   `json:"need_reply,omitempty"`    // whether to continue conversation
	ReplyContent string `json:"reply_content,omitempty"` // content for next turn (if NeedReply)
}

// DeliveryResult - P4 delivery output (new architecture)
type DeliveryResult struct {
	RequestID string           `json:"request_id"`        // Delivery request ID
	Content   *DeliveryContent `json:"content"`           // Agent-generated content
	Results   []ChannelResult  `json:"results,omitempty"` // Results per channel
	Success   bool             `json:"success"`           // Overall success
	Error     string           `json:"error,omitempty"`   // Error if failed
	SentAt    *time.Time       `json:"sent_at,omitempty"` // When delivery completed
}

// DeliveryContent - Content generated by Delivery Agent (only content, no channels)
type DeliveryContent struct {
	Summary     string               `json:"summary"`               // Brief 1-2 sentence summary
	Body        string               `json:"body"`                  // Full markdown report
	Attachments []DeliveryAttachment `json:"attachments,omitempty"` // Output artifacts from P3
}

// DeliveryAttachment - Task output attachment with metadata
type DeliveryAttachment struct {
	Title       string `json:"title"`                 // Human-readable title
	Description string `json:"description,omitempty"` // What this artifact is
	TaskID      string `json:"task_id,omitempty"`     // Which task produced this
	File        string `json:"file"`                  // Wrapper: __<uploader>://<fileID>
}

// DeliveryRequest - pushed to Delivery Center (no channels - center decides based on preferences)
type DeliveryRequest struct {
	Content *DeliveryContent `json:"content"` // Agent-generated content
	Context *DeliveryContext `json:"context"` // Tracking info
}

// DeliveryContext - tracking and audit info
type DeliveryContext struct {
	MemberID    string      `json:"member_id"`    // Robot member ID (globally unique)
	ExecutionID string      `json:"execution_id"` // Execution ID
	TriggerType TriggerType `json:"trigger_type"` // clock | human | event
	TeamID      string      `json:"team_id"`      // Team ID
}

// DeliveryPreferences - Robot/User delivery preferences (from Config)
type DeliveryPreferences struct {
	Email   *EmailPreference   `json:"email,omitempty"`   // Email delivery settings
	Webhook *WebhookPreference `json:"webhook,omitempty"` // Webhook delivery settings
	Process *ProcessPreference `json:"process,omitempty"` // Process delivery settings
}

// EmailPreference - Email delivery configuration
type EmailPreference struct {
	Enabled bool          `json:"enabled"`           // Whether email delivery is enabled
	Targets []EmailTarget `json:"targets,omitempty"` // Multiple email targets
}

// EmailTarget - Single email target
type EmailTarget struct {
	To       []string `json:"to"`                 // Recipient addresses
	Template string   `json:"template,omitempty"` // Email template ID
	Subject  string   `json:"subject,omitempty"`  // Subject template
}

// WebhookPreference - Webhook delivery configuration
type WebhookPreference struct {
	Enabled bool            `json:"enabled"`           // Whether webhook delivery is enabled
	Targets []WebhookTarget `json:"targets,omitempty"` // Multiple webhook targets
}

// WebhookTarget - Single webhook target
type WebhookTarget struct {
	URL     string            `json:"url"`               // Webhook URL
	Method  string            `json:"method,omitempty"`  // HTTP method (default: POST)
	Headers map[string]string `json:"headers,omitempty"` // Custom headers
	Secret  string            `json:"secret,omitempty"`  // Signing secret
}

// ProcessPreference - Process delivery configuration
type ProcessPreference struct {
	Enabled bool            `json:"enabled"`           // Whether process delivery is enabled
	Targets []ProcessTarget `json:"targets,omitempty"` // Multiple process targets
}

// ProcessTarget - Single process target
type ProcessTarget struct {
	Process string `json:"process"`        // Yao Process name
	Args    []any  `json:"args,omitempty"` // Process arguments
}

// ChannelResult - Result of delivery to a single channel target
type ChannelResult struct {
	Type       DeliveryType `json:"type"`                 // email | webhook | process
	Target     string       `json:"target"`               // Target identifier (email, URL, process name)
	Success    bool         `json:"success"`              // Whether delivery succeeded
	Recipients []string     `json:"recipients,omitempty"` // Who received (for email)
	Details    interface{}  `json:"details,omitempty"`    // Channel-specific response
	Error      string       `json:"error,omitempty"`      // Error message if failed
	SentAt     *time.Time   `json:"sent_at,omitempty"`    // When this target was delivered
}

// LearningEntry - knowledge to save
type LearningEntry struct {
	Type    LearningType `json:"type"` // execution | feedback | insight
	Content string       `json:"content"`
	Tags    []string     `json:"tags,omitempty"`
	Meta    interface{}  `json:"meta,omitempty"`
}

// NewRobotFromMap creates a Robot from a map (typically from DB record)
func NewRobotFromMap(m map[string]interface{}) (*Robot, error) {
	memberID := getString(m, "member_id")
	teamID := getString(m, "team_id")

	// Validate required fields
	if memberID == "" || teamID == "" {
		return nil, fmt.Errorf("missing required fields: member_id or team_id")
	}

	robot := &Robot{
		MemberID:       memberID,
		TeamID:         teamID,
		DisplayName:    getString(m, "display_name"),
		Bio:            getString(m, "bio"),
		SystemPrompt:   getString(m, "system_prompt"),
		AutonomousMode: getBool(m, "autonomous_mode"),
		RobotEmail:     getString(m, "robot_email"),
		ManagerID:      getString(m, "manager_id"),
		ManagerEmail:   getString(m, "manager_email"),
	}

	// Parse robot_status
	if status := getString(m, "robot_status"); status != "" {
		robot.Status = RobotStatus(status)
	} else {
		robot.Status = RobotIdle
	}

	// Parse robot_config JSON
	if configData, ok := m["robot_config"]; ok && configData != nil {
		config, err := ParseConfig(configData)
		if err != nil {
			return nil, fmt.Errorf("failed to parse robot_config: %w", err)
		}
		robot.Config = config
	}

	// Ensure Config exists for merging agents/mcp_servers
	if robot.Config == nil {
		robot.Config = &Config{}
	}
	if robot.Config.Resources == nil {
		robot.Config.Resources = &Resources{}
	}

	// Merge agents from member table into Config.Resources.Agents
	if agentsData, ok := m["agents"]; ok && agentsData != nil {
		agents := getStringSlice(agentsData)
		if len(agents) > 0 {
			robot.Config.Resources.Agents = agents
		}
	}

	// Merge mcp_servers from member table into Config.Resources.MCP
	if mcpData, ok := m["mcp_servers"]; ok && mcpData != nil {
		mcpServers := getStringSlice(mcpData)
		if len(mcpServers) > 0 {
			for _, serverID := range mcpServers {
				robot.Config.Resources.MCP = append(robot.Config.Resources.MCP, MCPConfig{
					ID: serverID,
				})
			}
		}
	}

	return robot, nil
}

// getStringSlice converts interface{} to []string
func getStringSlice(v interface{}) []string {
	if v == nil {
		return nil
	}
	switch val := v.(type) {
	case []string:
		return val
	case []interface{}:
		result := make([]string, 0, len(val))
		for _, item := range val {
			if s, ok := item.(string); ok {
				result = append(result, s)
			}
		}
		return result
	}
	return nil
}

// getString safely gets a string value from map
func getString(m map[string]interface{}, key string) string {
	if m == nil {
		return ""
	}
	if v, ok := m[key]; ok && v != nil {
		if s, ok := v.(string); ok {
			return s
		}
		return fmt.Sprintf("%v", v)
	}
	return ""
}

// getBool safely gets a bool value from map
func getBool(m map[string]interface{}, key string) bool {
	if m == nil {
		return false
	}
	if v, ok := m[key]; ok && v != nil {
		switch b := v.(type) {
		case bool:
			return b
		case int:
			return b != 0
		case int64:
			return b != 0
		case float64:
			return b != 0
		case string:
			return b == "true" || b == "1"
		}
	}
	return false
}
