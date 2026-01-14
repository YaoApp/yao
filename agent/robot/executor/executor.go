package executor

import "github.com/yaoapp/yao/agent/robot/types"

// Executor implements types.Executor interface
// This is a stub implementation for Phase 2
type Executor struct{}

// New creates a new executor instance
func New() *Executor {
	return &Executor{}
}

// Execute executes a robot through all phases
// Stub: returns empty execution (will be implemented in Phase 3+)
func (e *Executor) Execute(ctx *types.Context, robot *types.Robot, trigger types.TriggerType, data interface{}) (*types.Execution, error) {
	// Create a basic execution instance
	exec := &types.Execution{
		MemberID:    robot.MemberID,
		TeamID:      robot.TeamID,
		TriggerType: trigger,
		Status:      types.ExecCompleted,
		Phase:       types.PhaseLearning,
	}
	return exec, nil
}
