# Robot Agent Go API

Go API for managing autonomous robot agents.

## Quick Start

```go
import "github.com/yaoapp/yao/agent/robot/api"

// Start system
api.Start()
defer api.Stop()

// Trigger execution
result, _ := api.Trigger(ctx, "member_123", &api.TriggerRequest{
    Type:     types.TriggerHuman,
    Action:   types.ActionTaskAdd,
    Messages: []agentcontext.Message{{Role: "user", Content: "Analyze sales"}},
})

// Check status
exec, _ := api.GetExecution(ctx, result.ExecutionID)
```

## Lifecycle

```go
api.Start()                    // Start with defaults
api.StartWithConfig(config)    // Start with custom config
api.Stop()                     // Graceful shutdown
api.IsRunning()                // Check if running
```

## Robot Query

```go
// Get single robot
robot, err := api.GetRobot(ctx, "member_123")

// List robots with filters
result, err := api.ListRobots(ctx, &api.ListQuery{
    TeamID:    "team_1",
    Status:    types.RobotIdle,
    Keywords:  "sales",
    ClockMode: types.ClockInterval,
    Page:      1,
    PageSize:  20,
    Order:     "created_at desc",
})

// Get runtime status
state, err := api.GetRobotStatus(ctx, "member_123")
// state.Running, state.MaxRunning, state.RunningIDs, state.LastRun, state.NextRun
```

## Triggers

### Human Intervention

```go
result, err := api.Trigger(ctx, "member_123", &api.TriggerRequest{
    Type:     types.TriggerHuman,
    Action:   types.ActionTaskAdd,
    Messages: []agentcontext.Message{
        {Role: "user", Content: "Generate weekly report"},
    },
})
// Or use shorthand:
result, err := api.Intervene(ctx, "member_123", req)
```

### Event Trigger

```go
result, err := api.Trigger(ctx, "member_123", &api.TriggerRequest{
    Type:      types.TriggerEvent,
    Source:    types.EventWebhook,
    EventType: "order.created",
    Data:      map[string]interface{}{"order_id": "12345"},
})
// Or use shorthand:
result, err := api.HandleEvent(ctx, "member_123", req)
```

### Manual Trigger (Testing)

```go
result, err := api.TriggerManual(ctx, "member_123", types.TriggerClock, nil)
```

## Execution Management

```go
// Get execution by ID
exec, err := api.GetExecution(ctx, "exec_abc123")

// List executions with filters
result, err := api.ListExecutions(ctx, "member_123", &api.ExecutionQuery{
    Status:   types.ExecRunning,
    Trigger:  types.TriggerClock,
    Page:     1,
    PageSize: 10,
})

// Get execution with runtime status
exec, err := api.GetExecutionStatus(ctx, "exec_abc123")

// Control execution
api.PauseExecution(ctx, "exec_abc123")
api.ResumeExecution(ctx, "exec_abc123")
api.StopExecution(ctx, "exec_abc123")
```

## Types

### ListQuery

```go
type ListQuery struct {
    TeamID    string            // Filter by team
    Status    types.RobotStatus // Filter by status (idle|working|paused|error)
    Keywords  string            // Search in display_name
    ClockMode types.ClockMode   // Filter by clock mode (times|interval|daemon)
    Page      int               // Page number (default: 1)
    PageSize  int               // Page size (default: 20, max: 100)
    Order     string            // Order by column (default: "created_at desc")
}
```

### TriggerRequest

```go
type TriggerRequest struct {
    Type           types.TriggerType        // human | event | clock
    Action         types.InterventionAction // task.add, goal.adjust, etc.
    Messages       []agentcontext.Message   // User input
    PlanAt         *time.Time               // Schedule for later
    InsertPosition InsertPosition           // first | last | next | at
    AtIndex        int                      // When InsertPosition = "at"
    Source         types.EventSource        // webhook | database
    EventType      string                   // Event name
    Data           map[string]interface{}   // Event payload
    ExecutorMode   types.ExecutorMode       // standard | dryrun | sandbox
}
```

### TriggerResult

```go
type TriggerResult struct {
    Accepted    bool             // Whether trigger was accepted
    Queued      bool             // Whether queued (vs immediate)
    Execution   *types.Execution // Execution details
    ExecutionID string           // Execution ID for tracking
    Message     string           // Status message
}
```

### ExecutionQuery

```go
type ExecutionQuery struct {
    Status   types.ExecStatus  // Filter by status
    Trigger  types.TriggerType // Filter by trigger type
    Page     int               // Page number (default: 1)
    PageSize int               // Page size (default: 20, max: 100)
}
```

### RobotState

```go
type RobotState struct {
    MemberID    string            // Robot member ID
    TeamID      string            // Team ID
    DisplayName string            // Display name
    Status      types.RobotStatus // idle | working | paused | error
    Running     int               // Current running count
    MaxRunning  int               // Max concurrent limit
    LastRun     *time.Time        // Last execution time
    NextRun     *time.Time        // Next scheduled time
    RunningIDs  []string          // IDs of running executions
}
```

## Files

| File | Functions |
|------|-----------|
| `lifecycle.go` | `Start`, `StartWithConfig`, `Stop`, `IsRunning` |
| `robot.go` | `GetRobot`, `ListRobots`, `GetRobotStatus` |
| `trigger.go` | `Trigger`, `TriggerManual`, `Intervene`, `HandleEvent` |
| `execution.go` | `GetExecution`, `ListExecutions`, `GetExecutionStatus`, `PauseExecution`, `ResumeExecution`, `StopExecution` |
| `types.go` | Type definitions |
