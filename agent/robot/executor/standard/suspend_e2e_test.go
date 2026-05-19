//go:build e2e

package standard_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	agentcontext "github.com/yaoapp/yao/agent/context"
	"github.com/yaoapp/yao/agent/robot/executor/standard"
	"github.com/yaoapp/yao/agent/robot/store"
	robottypes "github.com/yaoapp/yao/agent/robot/types"
	"github.com/yaoapp/yao/unit-test/agent/testprepare"
)

func TestResumeWithSkipE2E(t *testing.T) {
	identity := testprepare.PrepareE2E(t)
	ctx := e2eCtx(identity)

	t.Run("R6_skip_reply_marks_task_as_skipped", func(t *testing.T) {
		robot := &robottypes.Robot{
			MemberID:     "e2e-rr-skip",
			TeamID:       identity.BetaOpenAITeamID,
			DisplayName:  "E2E Resume Skip Robot",
			SystemPrompt: "You are a helpful assistant.",
			Config: &robottypes.Config{
				Identity: &robottypes.Identity{Role: "Test", Duties: []string{"Execute tasks"}},
				Quota:    &robottypes.Quota{Max: 5},
				Resources: &robottypes.Resources{
					Phases: map[robottypes.Phase]string{
						robottypes.PhaseDelivery: "tests.e2e-robot-delivery",
						robottypes.PhaseLearning: "tests.robot-learning",
					},
					Agents: []string{"experts.text-writer"},
				},
			},
		}

		exec := &robottypes.Execution{
			ID:          "e2e-exec-resume-skip-" + time.Now().Format("150405.000"),
			MemberID:    robot.MemberID,
			TeamID:      robot.TeamID,
			TriggerType: robottypes.TriggerClock,
			StartTime:   time.Now(),
			Status:      robottypes.ExecWaiting,
			Phase:       robottypes.PhaseRun,
			Goals:       &robottypes.Goals{Content: "## Goals\n\n1. Test resume with skip"},
			Tasks: []robottypes.Task{
				{
					ID: "task-001", ExecutorType: robottypes.ExecutorAssistant,
					ExecutorID: "experts.text-writer",
					Messages:   []agentcontext.Message{{Role: agentcontext.RoleUser, Content: "Write 'hello'"}},
					Order:      0, Status: robottypes.TaskWaitingInput,
				},
			},
			WaitingTaskID:   "task-001",
			WaitingQuestion: "What should we do?",
			ChatID:          "robot_" + robot.MemberID + "_e2e-exec-resume-skip",
		}
		now := time.Now()
		exec.WaitingSince = &now
		exec.ResumeContext = &robottypes.ResumeContext{TaskIndex: 0, PreviousResults: []robottypes.TaskResult{}}
		exec.SetRobot(robot)

		execStore := store.NewExecutionStore()
		robotStore := store.NewRobotStore()
		require.NoError(t, execStore.Save(ctx.Context, store.FromExecution(exec)))
		require.NoError(t, robotStore.Save(ctx.Context, store.FromRobot(robot)))

		e := standard.New()
		err := e.Resume(ctx, exec.ID, "__skip__")
		require.NoError(t, err)

		loaded, err := execStore.Get(ctx.Context, exec.ID)
		require.NoError(t, err)
		require.NotNil(t, loaded)
		require.Len(t, loaded.Tasks, 1)
		assert.Equal(t, robottypes.TaskSkipped, loaded.Tasks[0].Status)
	})
}
