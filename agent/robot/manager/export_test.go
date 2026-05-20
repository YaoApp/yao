package manager

import (
	"github.com/yaoapp/yao/agent/robot/executor/standard"
	"github.com/yaoapp/yao/agent/robot/store"
	"github.com/yaoapp/yao/agent/robot/types"
)

func ExportBuildRobotStatusSnapshot(m *Manager, robot *types.Robot) *types.RobotStatusSnapshot {
	return m.buildRobotStatusSnapshot(robot)
}

func ExportFindWaitingTask(m *Manager, record *store.ExecutionRecord) *types.Task {
	return m.findWaitingTask(record)
}

func ExportBuildHostContext(m *Manager, robot *types.Robot, record *store.ExecutionRecord, waitingTask *types.Task) *types.HostContext {
	return m.buildHostContext(robot, record, waitingTask)
}

func ExportProcessHostAction(m *Manager, ctx *types.Context, robot *types.Robot, record *store.ExecutionRecord, output *types.HostOutput, execStore *store.ExecutionStore) (*InteractResponse, error) {
	return m.processHostAction(ctx, robot, record, output, execStore)
}

func ExportParseHostAgentResult(m *Manager, result *standard.CallResult) (*types.HostOutput, error) {
	return m.parseHostAgentResult(result)
}
