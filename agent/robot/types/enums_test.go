package types_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/yaoapp/yao/agent/robot/types"
)

func TestPhaseEnum(t *testing.T) {
	assert.Equal(t, types.Phase("inspiration"), types.PhaseInspiration)
	assert.Equal(t, types.Phase("goals"), types.PhaseGoals)
	assert.Equal(t, types.Phase("tasks"), types.PhaseTasks)
	assert.Equal(t, types.Phase("run"), types.PhaseRun)
	assert.Equal(t, types.Phase("delivery"), types.PhaseDelivery)
	assert.Equal(t, types.Phase("learning"), types.PhaseLearning)
}

func TestAllPhases(t *testing.T) {
	assert.Len(t, types.AllPhases, 6)
	assert.Equal(t, types.PhaseInspiration, types.AllPhases[0])
	assert.Equal(t, types.PhaseGoals, types.AllPhases[1])
	assert.Equal(t, types.PhaseTasks, types.AllPhases[2])
	assert.Equal(t, types.PhaseRun, types.AllPhases[3])
	assert.Equal(t, types.PhaseDelivery, types.AllPhases[4])
	assert.Equal(t, types.PhaseLearning, types.AllPhases[5])
}

func TestClockModeEnum(t *testing.T) {
	assert.Equal(t, types.ClockMode("times"), types.ClockTimes)
	assert.Equal(t, types.ClockMode("interval"), types.ClockInterval)
	assert.Equal(t, types.ClockMode("daemon"), types.ClockDaemon)
}

func TestTriggerTypeEnum(t *testing.T) {
	assert.Equal(t, types.TriggerType("clock"), types.TriggerClock)
	assert.Equal(t, types.TriggerType("human"), types.TriggerHuman)
	assert.Equal(t, types.TriggerType("event"), types.TriggerEvent)
}

func TestExecStatusEnum(t *testing.T) {
	assert.Equal(t, types.ExecStatus("pending"), types.ExecPending)
	assert.Equal(t, types.ExecStatus("running"), types.ExecRunning)
	assert.Equal(t, types.ExecStatus("completed"), types.ExecCompleted)
	assert.Equal(t, types.ExecStatus("failed"), types.ExecFailed)
	assert.Equal(t, types.ExecStatus("cancelled"), types.ExecCancelled)
}

func TestRobotStatusEnum(t *testing.T) {
	assert.Equal(t, types.RobotStatus("idle"), types.RobotIdle)
	assert.Equal(t, types.RobotStatus("working"), types.RobotWorking)
	assert.Equal(t, types.RobotStatus("paused"), types.RobotPaused)
	assert.Equal(t, types.RobotStatus("error"), types.RobotError)
	assert.Equal(t, types.RobotStatus("maintenance"), types.RobotMaintenance)
}

func TestInterventionActionEnum(t *testing.T) {
	// Task operations
	assert.Equal(t, types.InterventionAction("task.add"), types.ActionTaskAdd)
	assert.Equal(t, types.InterventionAction("task.cancel"), types.ActionTaskCancel)
	assert.Equal(t, types.InterventionAction("task.update"), types.ActionTaskUpdate)

	// Goal operations
	assert.Equal(t, types.InterventionAction("goal.adjust"), types.ActionGoalAdjust)
	assert.Equal(t, types.InterventionAction("goal.add"), types.ActionGoalAdd)
	assert.Equal(t, types.InterventionAction("goal.complete"), types.ActionGoalComplete)
	assert.Equal(t, types.InterventionAction("goal.cancel"), types.ActionGoalCancel)

	// Plan operations
	assert.Equal(t, types.InterventionAction("plan.add"), types.ActionPlanAdd)
	assert.Equal(t, types.InterventionAction("plan.remove"), types.ActionPlanRemove)
	assert.Equal(t, types.InterventionAction("plan.update"), types.ActionPlanUpdate)

	// Instruction
	assert.Equal(t, types.InterventionAction("instruct"), types.ActionInstruct)
}

func TestPriorityEnum(t *testing.T) {
	assert.Equal(t, types.Priority("high"), types.PriorityHigh)
	assert.Equal(t, types.Priority("normal"), types.PriorityNormal)
	assert.Equal(t, types.Priority("low"), types.PriorityLow)
}

func TestDeliveryTypeEnum(t *testing.T) {
	assert.Equal(t, types.DeliveryType("email"), types.DeliveryEmail)
	assert.Equal(t, types.DeliveryType("webhook"), types.DeliveryWebhook)
	assert.Equal(t, types.DeliveryType("process"), types.DeliveryProcess)
	assert.Equal(t, types.DeliveryType("notify"), types.DeliveryNotify)
}

func TestDedupResultEnum(t *testing.T) {
	assert.Equal(t, types.DedupResult("skip"), types.DedupSkip)
	assert.Equal(t, types.DedupResult("merge"), types.DedupMerge)
	assert.Equal(t, types.DedupResult("proceed"), types.DedupProceed)
}

func TestEventSourceEnum(t *testing.T) {
	assert.Equal(t, types.EventSource("webhook"), types.EventWebhook)
	assert.Equal(t, types.EventSource("database"), types.EventDatabase)
}

func TestLearningTypeEnum(t *testing.T) {
	assert.Equal(t, types.LearningType("execution"), types.LearnExecution)
	assert.Equal(t, types.LearningType("feedback"), types.LearnFeedback)
	assert.Equal(t, types.LearningType("insight"), types.LearnInsight)
}

func TestTaskSourceEnum(t *testing.T) {
	assert.Equal(t, types.TaskSource("auto"), types.TaskSourceAuto)
	assert.Equal(t, types.TaskSource("human"), types.TaskSourceHuman)
	assert.Equal(t, types.TaskSource("event"), types.TaskSourceEvent)
}

func TestExecutorTypeEnum(t *testing.T) {
	assert.Equal(t, types.ExecutorType("assistant"), types.ExecutorAssistant)
	assert.Equal(t, types.ExecutorType("mcp"), types.ExecutorMCP)
	assert.Equal(t, types.ExecutorType("process"), types.ExecutorProcess)
}

func TestTaskStatusEnum(t *testing.T) {
	assert.Equal(t, types.TaskStatus("pending"), types.TaskPending)
	assert.Equal(t, types.TaskStatus("running"), types.TaskRunning)
	assert.Equal(t, types.TaskStatus("completed"), types.TaskCompleted)
	assert.Equal(t, types.TaskStatus("failed"), types.TaskFailed)
	assert.Equal(t, types.TaskStatus("skipped"), types.TaskSkipped)
	assert.Equal(t, types.TaskStatus("cancelled"), types.TaskCancelled)
}

func TestInsertPositionEnum(t *testing.T) {
	assert.Equal(t, types.InsertPosition("first"), types.InsertFirst)
	assert.Equal(t, types.InsertPosition("last"), types.InsertLast)
	assert.Equal(t, types.InsertPosition("next"), types.InsertNext)
	assert.Equal(t, types.InsertPosition("at"), types.InsertAt)
}

func TestExecutorModeEnum(t *testing.T) {
	assert.Equal(t, types.ExecutorMode("standard"), types.ExecutorStandard)
	assert.Equal(t, types.ExecutorMode("dryrun"), types.ExecutorDryRun)
	assert.Equal(t, types.ExecutorMode("sandbox"), types.ExecutorSandbox)
}

func TestExecutorModeIsValid(t *testing.T) {
	tests := []struct {
		mode  types.ExecutorMode
		valid bool
	}{
		{types.ExecutorStandard, true},
		{types.ExecutorDryRun, true},
		{types.ExecutorSandbox, true},
		{"", true}, // empty is valid (defaults to standard)
		{types.ExecutorMode("invalid"), false},
		{types.ExecutorMode("unknown"), false},
	}

	for _, tt := range tests {
		t.Run(string(tt.mode), func(t *testing.T) {
			assert.Equal(t, tt.valid, tt.mode.IsValid())
		})
	}
}

func TestExecutorModeGetDefault(t *testing.T) {
	tests := []struct {
		mode     types.ExecutorMode
		expected types.ExecutorMode
	}{
		{"", types.ExecutorStandard},
		{types.ExecutorStandard, types.ExecutorStandard},
		{types.ExecutorDryRun, types.ExecutorDryRun},
		{types.ExecutorSandbox, types.ExecutorSandbox},
	}

	for _, tt := range tests {
		t.Run(string(tt.mode), func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.mode.GetDefault())
		})
	}
}
