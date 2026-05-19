//go:build e2e

package standard_test

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	agentcontext "github.com/yaoapp/yao/agent/context"
	"github.com/yaoapp/yao/agent/robot/executor/standard"
	robottypes "github.com/yaoapp/yao/agent/robot/types"
	oauthtypes "github.com/yaoapp/yao/openapi/oauth/types"
	"github.com/yaoapp/yao/unit-test/agent/testprepare"
)

func e2eCtx(identity *testprepare.TestIdentity) *robottypes.Context {
	return robottypes.NewContext(context.Background(), &oauthtypes.AuthorizedInfo{
		UserID: identity.BetaOpenAIOwnerUserID,
		TeamID: identity.BetaOpenAITeamID,
	})
}

func e2eGoalsRobot(identity *testprepare.TestIdentity) *robottypes.Robot {
	return &robottypes.Robot{
		MemberID:    "e2e-goals-robot",
		TeamID:      identity.BetaOpenAITeamID,
		DisplayName: "E2E Goals Robot",
		Config: &robottypes.Config{
			Identity: &robottypes.Identity{
				Role:   "Sales Analyst",
				Duties: []string{"Analyze sales data", "Generate reports", "Identify trends"},
			},
			Resources: &robottypes.Resources{
				Phases: map[robottypes.Phase]string{
					robottypes.PhaseGoals: "tests.e2e-robot-goals",
				},
				Agents: []string{"experts.data-analyst", "experts.summarizer"},
			},
		},
	}
}

func e2eGoalsExecution(robot *robottypes.Robot, trigger robottypes.TriggerType) *robottypes.Execution {
	exec := &robottypes.Execution{
		ID:          "e2e-exec-goals-" + time.Now().Format("150405.000"),
		MemberID:    robot.MemberID,
		TeamID:      robot.TeamID,
		TriggerType: trigger,
		StartTime:   time.Now(),
		Status:      robottypes.ExecRunning,
		Phase:       robottypes.PhaseGoals,
	}
	exec.SetRobot(robot)
	return exec
}

func TestRunGoalsBasicE2E(t *testing.T) {
	identity := testprepare.PrepareE2E(t)
	ctx := e2eCtx(identity)

	t.Run("generates_goals_from_inspiration_report", func(t *testing.T) {
		robot := e2eGoalsRobot(identity)
		exec := e2eGoalsExecution(robot, robottypes.TriggerClock)
		exec.Inspiration = &robottypes.InspirationReport{
			Clock:   robottypes.NewClockContext(time.Now(), ""),
			Content: "## Summary\nToday is Monday morning. Focus on weekly planning.\n\n## Highlights\n- New sales leads arrived\n- Weekly report due Friday",
		}

		e := standard.New()
		err := e.RunGoals(ctx, exec, nil)

		require.NoError(t, err)
		require.NotNil(t, exec.Goals)
		assert.NotEmpty(t, exec.Goals.Content, "goals content should not be empty")
	})

	t.Run("output_contains_goal_structure", func(t *testing.T) {
		robot := e2eGoalsRobot(identity)
		exec := e2eGoalsExecution(robot, robottypes.TriggerClock)
		exec.Inspiration = &robottypes.InspirationReport{
			Clock:   robottypes.NewClockContext(time.Now(), ""),
			Content: "## Summary\nUrgent: Customer complaint needs attention.\n\n## Highlights\n- Critical bug reported\n- Regular maintenance scheduled",
		}

		e := standard.New()
		err := e.RunGoals(ctx, exec, nil)

		require.NoError(t, err)
		content := exec.Goals.Content
		hasGoals := strings.Contains(content, "Goal") ||
			strings.Contains(content, "##") ||
			strings.Contains(content, "High") ||
			strings.Contains(content, "Normal") ||
			strings.Contains(content, "1.")

		assert.True(t, hasGoals, "should contain goals structure, got: %s", content)
	})
}

func TestRunGoalsHumanTriggerE2E(t *testing.T) {
	identity := testprepare.PrepareE2E(t)
	ctx := e2eCtx(identity)

	t.Run("generates_goals_from_human_intervention", func(t *testing.T) {
		robot := e2eGoalsRobot(identity)
		exec := e2eGoalsExecution(robot, robottypes.TriggerHuman)
		exec.Input = &robottypes.TriggerInput{
			Action: "task.add",
			UserID: "user-123",
			Messages: []agentcontext.Message{
				{Role: agentcontext.RoleUser, Content: "Please analyze the Q4 sales data and prepare a summary report for the management meeting tomorrow."},
			},
		}

		e := standard.New()
		err := e.RunGoals(ctx, exec, nil)

		require.NoError(t, err)
		require.NotNil(t, exec.Goals)
		assert.NotEmpty(t, exec.Goals.Content)

		content := strings.ToLower(exec.Goals.Content)
		hasRelevantContent := strings.Contains(content, "sales") ||
			strings.Contains(content, "report") ||
			strings.Contains(content, "analysis") ||
			strings.Contains(content, "data") ||
			strings.Contains(content, "q4")

		assert.True(t, hasRelevantContent, "goals should relate to user request, got: %s", exec.Goals.Content)
	})
}

func TestRunGoalsFallbackE2E(t *testing.T) {
	identity := testprepare.PrepareE2E(t)
	ctx := e2eCtx(identity)

	t.Run("falls_back_to_clock_context_when_no_inspiration", func(t *testing.T) {
		robot := e2eGoalsRobot(identity)
		exec := e2eGoalsExecution(robot, robottypes.TriggerClock)
		exec.Inspiration = nil

		e := standard.New()
		err := e.RunGoals(ctx, exec, nil)

		require.NoError(t, err)
		require.NotNil(t, exec.Goals)
		assert.NotEmpty(t, exec.Goals.Content)
	})
}
