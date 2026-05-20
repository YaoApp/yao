//go:build e2e

package standard_test

import (
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/yaoapp/yao/agent/robot/executor/standard"
	robottypes "github.com/yaoapp/yao/agent/robot/types"
	"github.com/yaoapp/yao/unit-test/agent/testprepare"
)

func e2eDeliveryRobot(identity *testprepare.TestIdentity) *robottypes.Robot {
	return &robottypes.Robot{
		MemberID:    "e2e-delivery-robot",
		TeamID:      identity.BetaOpenAITeamID,
		DisplayName: "E2E Delivery Robot",
		Config: &robottypes.Config{
			Identity: &robottypes.Identity{
				Role:   "Report Generator",
				Duties: []string{"Compile results", "Generate summaries"},
			},
			Resources: &robottypes.Resources{
				Phases: map[robottypes.Phase]string{
					robottypes.PhaseDelivery: "tests.e2e-robot-delivery",
				},
			},
		},
	}
}

func e2eDeliveryExecution(robot *robottypes.Robot) *robottypes.Execution {
	exec := &robottypes.Execution{
		ID:          "e2e-exec-delivery-" + time.Now().Format("150405.000"),
		MemberID:    robot.MemberID,
		TeamID:      robot.TeamID,
		TriggerType: robottypes.TriggerClock,
		StartTime:   time.Now(),
		Status:      robottypes.ExecRunning,
		Phase:       robottypes.PhaseDelivery,
	}
	exec.SetRobot(robot)
	return exec
}

func TestRunDeliveryBasicE2E(t *testing.T) {
	identity := testprepare.PrepareE2E(t)
	ctx := e2eCtx(identity)

	t.Run("generates_delivery_content_from_execution_results", func(t *testing.T) {
		robot := e2eDeliveryRobot(identity)
		exec := e2eDeliveryExecution(robot)

		exec.Inspiration = &robottypes.InspirationReport{
			Content: "Morning analysis suggests focus on Q4 review.",
		}
		exec.Goals = &robottypes.Goals{
			Content: "## Goals\n1. Review Q4 data\n2. Generate summary report",
		}
		exec.Tasks = []robottypes.Task{
			{ID: "task-001", ExecutorID: "experts.data-analyst", ExecutorType: robottypes.ExecutorAssistant, Status: robottypes.TaskCompleted},
			{ID: "task-002", ExecutorID: "experts.summarizer", ExecutorType: robottypes.ExecutorAssistant, Status: robottypes.TaskCompleted},
		}
		exec.Results = []robottypes.TaskResult{
			{TaskID: "task-001", Success: true, Duration: 1500, Output: map[string]interface{}{"total_sales": 1500000}},
			{TaskID: "task-002", Success: true, Duration: 800, Output: "Q4 sales exceeded expectations by 15%."},
		}

		e := standard.New()
		err := e.RunDelivery(ctx, exec, nil)

		require.NoError(t, err)
		require.NotNil(t, exec.Delivery)
		require.NotNil(t, exec.Delivery.Content)
		assert.NotEmpty(t, exec.Delivery.Content.Summary, "delivery summary should not be empty")
		assert.NotEmpty(t, exec.Delivery.Content.Body, "delivery body should not be empty")
		assert.True(t, exec.Delivery.Success)
	})

	t.Run("handles_partial_failure_in_results", func(t *testing.T) {
		robot := e2eDeliveryRobot(identity)
		exec := e2eDeliveryExecution(robot)

		exec.Goals = &robottypes.Goals{
			Content: "## Goals\n1. Analyze data\n2. Generate report",
		}
		exec.Tasks = []robottypes.Task{
			{ID: "task-001", ExecutorID: "experts.data-analyst", ExecutorType: robottypes.ExecutorAssistant, Status: robottypes.TaskCompleted},
			{ID: "task-002", ExecutorID: "experts.summarizer", ExecutorType: robottypes.ExecutorAssistant, Status: robottypes.TaskFailed},
		}
		exec.Results = []robottypes.TaskResult{
			{TaskID: "task-001", Success: true, Duration: 1500, Output: map[string]interface{}{"data": "analyzed"}},
			{TaskID: "task-002", Success: false, Duration: 500, Error: "Summarization failed: timeout"},
		}

		e := standard.New()
		err := e.RunDelivery(ctx, exec, nil)

		require.NoError(t, err)
		require.NotNil(t, exec.Delivery)
		require.NotNil(t, exec.Delivery.Content)

		body := strings.ToLower(exec.Delivery.Content.Body)
		hasFailureInfo := strings.Contains(body, "fail") ||
			strings.Contains(body, "error") ||
			strings.Contains(body, "partial") ||
			strings.Contains(body, "timeout")

		assert.True(t, hasFailureInfo || exec.Delivery.Content.Summary != "",
			"should mention failure or have valid summary")
	})
}
