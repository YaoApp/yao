# Robot Executor

Robot Executor provides pluggable execution strategies for robot phase execution.

## Architecture

```
executor/
├── types/
│   ├── types.go          # Interface definitions and common types
│   └── helpers.go        # Shared helper functions
├── standard/
│   ├── executor.go       # Real Agent execution (production)
│   └── phases.go         # Phase implementations
├── dryrun/
│   └── executor.go       # Simulated execution (testing/demo)
├── sandbox/
│   └── executor.go       # Container-isolated execution (NOT IMPLEMENTED)
└── executor.go           # Factory functions and unified entry
```

## Execution Modes

### Standard Mode (Production)

Real Agent calls with full phase execution:

```go
exec := executor.New()
// or
exec := executor.NewWithConfig(executor.Config{
    OnPhaseStart: func(phase types.Phase) { ... },
    OnPhaseEnd:   func(phase types.Phase) { ... },
})
```

### DryRun Mode (Testing/Demo)

Simulates execution without real Agent calls:

```go
// Simple dry-run
exec := executor.NewDryRun()

// With delay simulation
exec := executor.NewDryRunWithDelay(100 * time.Millisecond)

// With full configuration
exec := executor.NewDryRunWithConfig(executor.DryRunConfig{
    Delay:        100 * time.Millisecond,
    OnStart:      func() { ... },
    OnEnd:        func() { ... },
    Config: executor.Config{
        OnPhaseStart: func(phase types.Phase) { ... },
    },
})
```

### Sandbox Mode (NOT IMPLEMENTED)

> **⚠️ Not Implemented:** Sandbox mode requires container-level isolation (Docker/gVisor/Firecracker) for true security isolation. This is a future feature that depends on infrastructure support.

**Intended Design:**

Sandbox mode is designed for executing untrusted robot configurations in a fully isolated environment:

- **Container Isolation:** Each execution runs in a separate container
- **Resource Limits:** CPU, memory, disk, network quotas enforced by container runtime
- **Network Isolation:** Restricted network access via container networking
- **File System Isolation:** Read-only root filesystem, limited writable paths
- **Process Isolation:** Separate PID namespace, no access to host processes

**Future Implementation:**

```go
// Future API (not yet implemented)
exec := executor.NewSandbox(executor.SandboxConfig{
    Image:         "yao-executor:latest",
    MaxDuration:   30 * time.Minute,
    MaxMemory:     512 * 1024 * 1024, // 512MB
    MaxCPU:        1.0,               // 1 CPU core
    NetworkPolicy: "restricted",      // restricted | none | full
    AllowedAgents: []string{"agent1", "agent2"},
})
```

**Current Placeholder:**

The current `sandbox/executor.go` is a placeholder that behaves like DryRun mode. It does NOT provide real security isolation.

## Mode Selection

Select mode dynamically:

```go
// By mode constant
exec := executor.NewWithMode(executor.ModeDryRun)

// From settings
setting := &executor.Setting{
    Mode: executor.ModeStandard,
}
exec := executor.NewWithSetting(setting)
```

## Interface

All executors implement the `Executor` interface:

```go
type Executor interface {
    Execute(ctx *Context, robot *Robot, trigger TriggerType, data interface{}) (*Execution, error)
    ExecCount() int
    CurrentCount() int
    Reset()
}
```

## Use Cases

| Mode     | Use Case                                            | Status             |
| -------- | --------------------------------------------------- | ------------------ |
| Standard | Production environment with real Agent calls        | ✅ Implemented     |
| DryRun   | Unit tests, integration tests, demos, previews      | ✅ Implemented     |
| Sandbox  | Untrusted code execution, multi-tenant environments | ⬜ Not Implemented |

## Testing

Tests use DryRun mode by default:

```go
func TestSomething(t *testing.T) {
    exec := executor.NewDryRunWithDelay(50 * time.Millisecond)
    // ... test with simulated execution
}
```

## Manager Integration

Inject executor into Manager:

```go
exec := executor.NewDryRun()
config := &manager.Config{
    Executor: exec,
}
m := manager.NewWithConfig(config)
```
