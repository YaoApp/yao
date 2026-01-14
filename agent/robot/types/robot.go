package types

import (
	"context"
	"sync"
	"time"

	agentcontext "github.com/yaoapp/yao/agent/context"
)

// Robot - runtime representation of an autonomous robot (from __yao.member)
// Relationship: 1 Robot : N Executions (concurrent)
// Each trigger creates a new Execution (mapped to job.Job)
type Robot struct {
	// From __yao.member
	MemberID       string      `json:"member_id"`
	TeamID         string      `json:"team_id"`
	DisplayName    string      `json:"display_name"`
	SystemPrompt   string      `json:"system_prompt"`
	Status         RobotStatus `json:"robot_status"`
	AutonomousMode bool        `json:"autonomous_mode"`

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
func (r *Robot) CanRun() bool {
	r.execMu.RLock()
	defer r.execMu.RUnlock()
	return len(r.executions) < r.Config.Quota.GetMax()
}

// RunningCount returns current running execution count
func (r *Robot) RunningCount() int {
	r.execMu.RLock()
	defer r.execMu.RUnlock()
	return len(r.executions)
}

// AddExecution adds an execution to tracking
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
// Each trigger creates a new Execution, mapped to a job.Job for monitoring
// Relationship: 1 Execution = 1 job.Job
type Execution struct {
	ID          string      `json:"id"`           // unique execution ID
	MemberID    string      `json:"member_id"`    // robot member ID
	TeamID      string      `json:"team_id"`
	TriggerType TriggerType `json:"trigger_type"` // clock | human | event
	StartTime   time.Time   `json:"start_time"`
	EndTime     *time.Time  `json:"end_time,omitempty"`
	Status      ExecStatus  `json:"status"`
	Phase       Phase       `json:"phase"`
	Error       string      `json:"error,omitempty"`

	// Job integration (each Execution = 1 job.Job)
	JobID string `json:"job_id"` // corresponding job.Job ID

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

// TriggerInput - stored trigger input for traceability
type TriggerInput struct {
	// For human intervention
	Action   InterventionAction      `json:"action,omitempty"`   // task.add, goal.adjust, etc.
	Messages []agentcontext.Message  `json:"messages,omitempty"` // user's input (text, images, files)
	UserID   string                  `json:"user_id,omitempty"`  // who triggered

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

// Goals - P1 output (markdown for LLM)
// P1 Agent reads InspirationReport and generates goals as markdown
// Example:
// ## Goals
// 1. [High] Analyze sales data and identify trends
//    - Reason: Sales up 50%, need to understand why
// 2. [Normal] Prepare weekly report for manager
//    - Reason: Friday 5pm, weekly report due
// 3. [Low] Update CRM with new leads
//    - Reason: 3 pending leads from yesterday
type Goals struct {
	Content string `json:"content"` // markdown text
}

// Task - planned task (structured, for execution)
type Task struct {
	ID       string                 `json:"id"`
	Messages []agentcontext.Message `json:"messages"`           // original input (text, images, files)
	GoalRef  string                 `json:"goal_ref,omitempty"` // reference to goal (e.g., "Goal 1")
	Source   TaskSource             `json:"source"`             // auto | human | event

	// Executor
	ExecutorType ExecutorType `json:"executor_type"`
	ExecutorID   string       `json:"executor_id"`
	Args         []any        `json:"args,omitempty"`

	// Runtime
	Status    TaskStatus `json:"status"`
	Order     int        `json:"order"` // execution order (0-based)
	StartTime *time.Time `json:"start_time,omitempty"`
	EndTime   *time.Time `json:"end_time,omitempty"`
}

// TaskResult - task execution result
type TaskResult struct {
	TaskID    string      `json:"task_id"`
	Success   bool        `json:"success"`
	Output    interface{} `json:"output,omitempty"`
	Error     string      `json:"error,omitempty"`
	Duration  int64       `json:"duration_ms"`
	Validated bool        `json:"validated"`
}

// DeliveryResult - delivery output
type DeliveryResult struct {
	Type    DeliveryType `json:"type"`
	Success bool         `json:"success"`
	Details interface{}  `json:"details,omitempty"`
	Error   string       `json:"error,omitempty"`
}

// LearningEntry - knowledge to save
type LearningEntry struct {
	Type    LearningType `json:"type"` // execution | feedback | insight
	Content string       `json:"content"`
	Tags    []string     `json:"tags,omitempty"`
	Meta    interface{}  `json:"meta,omitempty"`
}
