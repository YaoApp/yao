package types

import (
	"time"

	robottypes "github.com/yaoapp/yao/agent/robot/types"
)

// Executor defines the interface for robot phase execution
// Different implementations provide different execution strategies:
//   - Standard: Real Agent calls with full phase execution
//   - DryRun:   Plan-only mode, simulates execution without Agent calls
//   - Sandbox:  Isolated execution with resource limits and safety controls
type Executor interface {
	// Execute runs a robot through all applicable phases
	// ctx: Execution context with auth and logging
	// robot: Robot configuration and state
	// trigger: What triggered this execution (clock, human, event)
	// data: Trigger-specific data (human input, event payload, etc.)
	// Returns: Execution record with all phase outputs
	Execute(ctx *robottypes.Context, robot *robottypes.Robot, trigger robottypes.TriggerType, data interface{}) (*robottypes.Execution, error)

	// Metrics and control
	ExecCount() int    // Total execution count
	CurrentCount() int // Currently running execution count
	Reset()            // Reset counters (for testing)
}

// PhaseExecutor defines the interface for individual phase execution
// Used internally by Executor implementations
type PhaseExecutor interface {
	// RunInspiration executes P0: Inspiration phase
	RunInspiration(ctx *robottypes.Context, exec *robottypes.Execution, data interface{}) error

	// RunGoals executes P1: Goals phase
	RunGoals(ctx *robottypes.Context, exec *robottypes.Execution, data interface{}) error

	// RunTasks executes P2: Tasks phase
	RunTasks(ctx *robottypes.Context, exec *robottypes.Execution, data interface{}) error

	// RunExecution executes P3: Run phase (task execution)
	RunExecution(ctx *robottypes.Context, exec *robottypes.Execution, data interface{}) error

	// RunDelivery executes P4: Delivery phase
	RunDelivery(ctx *robottypes.Context, exec *robottypes.Execution, data interface{}) error

	// RunLearning executes P5: Learning phase
	RunLearning(ctx *robottypes.Context, exec *robottypes.Execution, data interface{}) error
}

// Config holds common executor configuration
type Config struct {
	// SkipJobIntegration skips job system integration (for testing)
	SkipJobIntegration bool

	// SkipPersistence skips execution record persistence (for testing)
	SkipPersistence bool

	// OnPhaseStart callback when a phase starts
	OnPhaseStart func(phase robottypes.Phase)

	// OnPhaseEnd callback when a phase ends
	OnPhaseEnd func(phase robottypes.Phase)
}

// DryRunConfig holds dry-run specific configuration
type DryRunConfig struct {
	Config

	// Delay simulates execution delay for each phase
	Delay time.Duration

	// OnStart callback on execution start
	OnStart func()

	// OnEnd callback on execution end
	OnEnd func()
}

// SandboxConfig holds sandbox specific configuration
//
// ⚠️ NOT IMPLEMENTED: These settings are placeholders for future
// container-based isolation. True sandbox requires infrastructure support
// (Docker/gVisor/Firecracker). Current implementation behaves like DryRun.
type SandboxConfig struct {
	Config

	// MaxDuration limits total execution time
	MaxDuration time.Duration

	// MaxMemory limits memory usage (bytes) - requires container runtime
	MaxMemory int64

	// AllowedAgents restricts which agents can be called
	AllowedAgents []string

	// AllowedTools restricts which tools can be used
	AllowedTools []string

	// NetworkAccess controls network access - requires container networking
	NetworkAccess bool

	// FileAccess controls file system access - requires container filesystem
	FileAccess bool
}

// Mode represents the executor mode
type Mode string

const (
	ModeStandard Mode = "standard" // Real Agent execution (production)
	ModeDryRun   Mode = "dryrun"   // Simulated execution (testing/demo)
	ModeSandbox  Mode = "sandbox"  // Container-isolated execution (NOT IMPLEMENTED)
)

// Setting holds executor settings from configuration
type Setting struct {
	Mode          Mode          `json:"mode,omitempty" yaml:"mode,omitempty"`                     // Executor mode
	MaxDuration   time.Duration `json:"max_duration,omitempty" yaml:"max_duration,omitempty"`     // Max execution time
	MaxMemory     int64         `json:"max_memory,omitempty" yaml:"max_memory,omitempty"`         // Max memory (bytes)
	AllowedAgents []string      `json:"allowed_agents,omitempty" yaml:"allowed_agents,omitempty"` // Allowed agent IDs
	NetworkAccess bool          `json:"network_access,omitempty" yaml:"network_access,omitempty"` // Allow network
	FileAccess    bool          `json:"file_access,omitempty" yaml:"file_access,omitempty"`       // Allow file system
}

// DefaultSetting returns default executor settings
func DefaultSetting() *Setting {
	return &Setting{
		Mode:          ModeStandard,
		MaxDuration:   30 * time.Minute,
		MaxMemory:     512 * 1024 * 1024, // 512MB
		NetworkAccess: true,
		FileAccess:    false,
	}
}
