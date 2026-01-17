// Package executor provides robot execution strategies
//
// Architecture:
//
//	executor/
//	├── types/
//	│   ├── types.go          # Interface definitions and common types
//	│   └── helpers.go        # Shared helper functions
//	├── standard/
//	│   ├── executor.go       # Real Agent execution (production)
//	│   ├── agent.go          # AgentCaller for LLM calls
//	│   ├── input.go          # InputFormatter for prompts
//	│   ├── inspiration.go    # P0: Inspiration phase
//	│   ├── goals.go          # P1: Goals phase
//	│   ├── tasks.go          # P2: Tasks phase
//	│   ├── run.go            # P3: Run phase
//	│   ├── delivery.go       # P4: Delivery phase
//	│   └── learning.go       # P5: Learning phase
//	├── dryrun/
//	│   └── executor.go       # Simulated execution (testing/demo)
//	├── sandbox/
//	│   └── executor.go       # Container-isolated execution (NOT IMPLEMENTED)
//	└── executor.go           # Factory functions (this file)
//
// Usage:
//
//	// Production - real Agent calls
//	exec := executor.New()
//
//	// Testing - simulated execution
//	exec := executor.NewDryRun()
//
//	// Sandbox - NOT IMPLEMENTED (requires container infrastructure)
//	// exec := executor.NewSandbox() // placeholder only
//
//	// With mode selection
//	exec := executor.NewWithMode(executor.ModeDryRun)
package executor

import (
	"time"

	"github.com/yaoapp/yao/agent/robot/executor/dryrun"
	"github.com/yaoapp/yao/agent/robot/executor/sandbox"
	"github.com/yaoapp/yao/agent/robot/executor/standard"
	"github.com/yaoapp/yao/agent/robot/executor/types"
	robottypes "github.com/yaoapp/yao/agent/robot/types"
)

// Re-export types for convenience
type (
	Executor      = types.Executor
	Config        = types.Config
	DryRunConfig  = types.DryRunConfig
	SandboxConfig = types.SandboxConfig
	Mode          = types.Mode
	Setting       = types.Setting
)

// Re-export mode constants
const (
	ModeStandard = types.ModeStandard
	ModeDryRun   = types.ModeDryRun
	ModeSandbox  = types.ModeSandbox
)

// ==================== Factory Functions ====================

// New creates a new standard executor (production mode)
// Uses real Agent calls for phase execution
func New() Executor {
	return standard.New()
}

// NewWithConfig creates a standard executor with configuration
func NewWithConfig(config Config) Executor {
	return standard.NewWithConfig(config)
}

// NewDryRun creates a dry-run executor (testing/demo mode)
// Simulates execution without real Agent calls
func NewDryRun() Executor {
	return dryrun.New()
}

// NewDryRunWithDelay creates a dry-run executor with specified delay
func NewDryRunWithDelay(delay time.Duration) *DryRunExecutor {
	return dryrun.NewWithDelay(delay)
}

// NewDryRunWithConfig creates a dry-run executor with full configuration
func NewDryRunWithConfig(config DryRunConfig) *DryRunExecutor {
	return dryrun.NewWithConfig(config)
}

// NewDryRunWithCallbacks creates a dry-run executor with start/end callbacks
func NewDryRunWithCallbacks(delay time.Duration, onStart, onEnd func()) *DryRunExecutor {
	return dryrun.NewWithConfig(DryRunConfig{
		Delay:   delay,
		OnStart: onStart,
		OnEnd:   onEnd,
	})
}

// NewSandbox creates a sandbox executor placeholder
//
// ⚠️ NOT IMPLEMENTED: True sandbox requires container-level isolation
// (Docker/gVisor/Firecracker). Current implementation behaves like DryRun.
func NewSandbox() Executor {
	return sandbox.New()
}

// NewSandboxWithConfig creates a sandbox executor placeholder with configuration
//
// ⚠️ NOT IMPLEMENTED: Config options are placeholders. Current implementation
// behaves like DryRun and does NOT provide real security isolation.
func NewSandboxWithConfig(config SandboxConfig) Executor {
	return sandbox.NewWithConfig(config)
}

// NewWithMode creates an executor based on the specified mode
func NewWithMode(mode Mode) Executor {
	switch mode {
	case ModeDryRun:
		return NewDryRun()
	case ModeSandbox:
		return NewSandbox()
	default:
		return New()
	}
}

// NewWithSetting creates an executor based on configuration settings
func NewWithSetting(setting *Setting) Executor {
	if setting == nil {
		return New()
	}

	switch setting.Mode {
	case ModeDryRun:
		return NewDryRun()
	case ModeSandbox:
		return NewSandboxWithConfig(SandboxConfig{
			MaxDuration:   setting.MaxDuration,
			MaxMemory:     setting.MaxMemory,
			AllowedAgents: setting.AllowedAgents,
			NetworkAccess: setting.NetworkAccess,
			FileAccess:    setting.FileAccess,
		})
	default:
		return New()
	}
}

// ==================== Concrete Types ====================
// Export concrete executor types for direct access when needed

// DryRunExecutor is the concrete dry-run executor type
type DryRunExecutor = dryrun.Executor

// StandardExecutor is the concrete standard executor type
type StandardExecutor = standard.Executor

// SandboxExecutor is the concrete sandbox executor type
type SandboxExecutor = sandbox.Executor

// ==================== Interface Verification ====================

// Verify all executors implement the Executor interface
var (
	_ Executor = (*standard.Executor)(nil)
	_ Executor = (*dryrun.Executor)(nil)
	_ Executor = (*sandbox.Executor)(nil)
)

// Verify standard executor implements PhaseExecutor
var _ types.PhaseExecutor = (*standard.Executor)(nil)

// ==================== Helper Types ====================

// DefaultSetting returns default executor settings
func DefaultSetting() *Setting {
	return types.DefaultSetting()
}

// PhaseExecutor is the interface for phase execution
type PhaseExecutor = types.PhaseExecutor

// ==================== Context Helpers ====================

// These are re-exported from robot types for convenience
type (
	Context   = robottypes.Context
	Robot     = robottypes.Robot
	Execution = robottypes.Execution
	Phase     = robottypes.Phase
)
