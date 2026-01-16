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
	"github.com/yaoapp/yao/agent/robot/types"
	"github.com/yaoapp/yao/agent/testutils"
)

// ============================================================================
// P1 Goals Phase Tests
// ============================================================================

func TestRunGoalsBasic(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	testutils.Prepare(t)
	defer testutils.Clean(t)

	ctx := types.NewContext(context.Background(), testAuth())

	t.Run("generates goals from inspiration report (clock trigger)", func(t *testing.T) {
		// Create robot with goals agent configured
		robot := createGoalsTestRobot(t, "robot.goals")

		// Create execution with inspiration report (from P0)
		exec := createGoalsTestExecution(robot, types.TriggerClock)
		exec.Inspiration = &types.InspirationReport{
			Clock:   types.NewClockContext(time.Now(), ""),
			Content: "## Summary\nToday is Monday morning. Focus on weekly planning.\n\n## Highlights\n- New sales leads arrived\n- Weekly report due Friday",
		}

		// Run goals phase
		e := standard.New()
		err := e.RunGoals(ctx, exec, nil)

		require.NoError(t, err)
		require.NotNil(t, exec.Goals)
		assert.NotEmpty(t, exec.Goals.Content)
	})

	t.Run("includes priority markers in output", func(t *testing.T) {
		robot := createGoalsTestRobot(t, "robot.goals")
		exec := createGoalsTestExecution(robot, types.TriggerClock)
		exec.Inspiration = &types.InspirationReport{
			Clock:   types.NewClockContext(time.Now(), ""),
			Content: "## Summary\nUrgent: Customer complaint needs attention.\n\n## Highlights\n- Critical bug reported\n- Regular maintenance scheduled",
		}

		e := standard.New()
		err := e.RunGoals(ctx, exec, nil)

		require.NoError(t, err)
		content := exec.Goals.Content

		// Verify expected structure in markdown output
		// Note: LLM output is non-deterministic, so we check for likely patterns
		hasGoals := strings.Contains(content, "Goal") ||
			strings.Contains(content, "##") ||
			strings.Contains(content, "High") ||
			strings.Contains(content, "Normal") ||
			strings.Contains(content, "1.")

		assert.True(t, hasGoals, "should contain goals structure, got: %s", content)
	})
}

func TestRunGoalsHumanTrigger(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	testutils.Prepare(t)
	defer testutils.Clean(t)

	ctx := types.NewContext(context.Background(), testAuth())

	t.Run("generates goals from human intervention", func(t *testing.T) {
		robot := createGoalsTestRobot(t, "robot.goals")
		exec := createGoalsTestExecution(robot, types.TriggerHuman)

		// Set human intervention input
		exec.Input = &types.TriggerInput{
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

		// Goals should be related to the user request
		content := strings.ToLower(exec.Goals.Content)
		hasRelevantContent := strings.Contains(content, "sales") ||
			strings.Contains(content, "report") ||
			strings.Contains(content, "analysis") ||
			strings.Contains(content, "data") ||
			strings.Contains(content, "q4")

		assert.True(t, hasRelevantContent, "goals should relate to user request, got: %s", exec.Goals.Content)
	})

	t.Run("includes robot identity for human trigger", func(t *testing.T) {
		robot := &types.Robot{
			MemberID:    "test-robot-1",
			TeamID:      "test-team-1",
			DisplayName: "Sales Analyst",
			Config: &types.Config{
				Identity: &types.Identity{
					Role:   "Sales Analyst",
					Duties: []string{"Analyze sales data", "Generate reports"},
					Rules:  []string{"Focus on actionable insights"},
				},
				Resources: &types.Resources{
					Phases: map[types.Phase]string{
						types.PhaseGoals: "robot.goals",
					},
				},
			},
		}
		exec := createGoalsTestExecution(robot, types.TriggerHuman)
		exec.Input = &types.TriggerInput{
			Action: "instruct",
			Messages: []agentcontext.Message{
				{Role: agentcontext.RoleUser, Content: "What should I focus on today?"},
			},
		}

		e := standard.New()
		err := e.RunGoals(ctx, exec, nil)

		require.NoError(t, err)
		assert.NotEmpty(t, exec.Goals.Content)
	})
}

func TestRunGoalsEventTrigger(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	testutils.Prepare(t)
	defer testutils.Clean(t)

	ctx := types.NewContext(context.Background(), testAuth())

	t.Run("generates goals from event trigger", func(t *testing.T) {
		robot := createGoalsTestRobot(t, "robot.goals")
		exec := createGoalsTestExecution(robot, types.TriggerEvent)

		// Set event input
		exec.Input = &types.TriggerInput{
			Source:    "webhook",
			EventType: "lead.created",
			Data: map[string]interface{}{
				"lead_id":      "lead-456",
				"company":      "BigCorp Inc",
				"contact_name": "John Smith",
				"email":        "john@bigcorp.com",
				"interest":     "Enterprise plan",
			},
		}

		e := standard.New()
		err := e.RunGoals(ctx, exec, nil)

		require.NoError(t, err)
		require.NotNil(t, exec.Goals)
		assert.NotEmpty(t, exec.Goals.Content)

		// Goals should be related to the event
		content := strings.ToLower(exec.Goals.Content)
		hasRelevantContent := strings.Contains(content, "lead") ||
			strings.Contains(content, "bigcorp") ||
			strings.Contains(content, "contact") ||
			strings.Contains(content, "follow") ||
			strings.Contains(content, "qualify")

		assert.True(t, hasRelevantContent, "goals should relate to event, got: %s", exec.Goals.Content)
	})
}

func TestRunGoalsErrorHandling(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	testutils.Prepare(t)
	defer testutils.Clean(t)

	ctx := types.NewContext(context.Background(), testAuth())

	t.Run("returns error when robot is nil", func(t *testing.T) {
		exec := &types.Execution{
			ID:          "test-exec-1",
			TriggerType: types.TriggerClock,
		}
		// Don't set robot

		e := standard.New()
		err := e.RunGoals(ctx, exec, nil)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "robot not found")
	})

	t.Run("returns error when agent not found", func(t *testing.T) {
		robot := &types.Robot{
			MemberID: "test-robot-1",
			TeamID:   "test-team-1",
			Config: &types.Config{
				Identity: &types.Identity{Role: "Test"},
				Resources: &types.Resources{
					Phases: map[types.Phase]string{
						types.PhaseGoals: "non.existent.agent",
					},
				},
			},
		}
		exec := createGoalsTestExecution(robot, types.TriggerClock)
		exec.Inspiration = &types.InspirationReport{
			Content: "Test content",
		}

		e := standard.New()
		err := e.RunGoals(ctx, exec, nil)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "agent call failed")
	})

	t.Run("returns error when no input available and no identity", func(t *testing.T) {
		// Robot without identity - should fail when no input is provided
		robot := &types.Robot{
			MemberID: "test-robot-1",
			TeamID:   "test-team-1",
			Config: &types.Config{
				// No Identity - so no fallback content
				Resources: &types.Resources{
					Phases: map[types.Phase]string{
						types.PhaseGoals: "robot.goals",
					},
				},
			},
		}
		exec := createGoalsTestExecution(robot, types.TriggerHuman)
		exec.Input = nil // No input

		e := standard.New()
		err := e.RunGoals(ctx, exec, nil)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "no input available")
	})
}

func TestRunGoalsFallbackBehavior(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	testutils.Prepare(t)
	defer testutils.Clean(t)

	ctx := types.NewContext(context.Background(), testAuth())

	t.Run("falls back to clock context when no inspiration report", func(t *testing.T) {
		robot := createGoalsTestRobot(t, "robot.goals")
		exec := createGoalsTestExecution(robot, types.TriggerClock)
		exec.Inspiration = nil // No inspiration report

		e := standard.New()
		err := e.RunGoals(ctx, exec, nil)

		// Should still work with fallback clock context
		require.NoError(t, err)
		require.NotNil(t, exec.Goals)
		assert.NotEmpty(t, exec.Goals.Content)
	})
}

// ============================================================================
// Delivery Parsing Tests
// ============================================================================

func TestParseDeliveryFromGoalsResponse(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	testutils.Prepare(t)
	defer testutils.Clean(t)

	ctx := types.NewContext(context.Background(), testAuth())

	t.Run("parses delivery when agent returns it", func(t *testing.T) {
		robot := createGoalsTestRobot(t, "robot.goals")
		exec := createGoalsTestExecution(robot, types.TriggerHuman)

		// Request that explicitly asks for email delivery
		exec.Input = &types.TriggerInput{
			Action: "task.add",
			Messages: []agentcontext.Message{
				{Role: agentcontext.RoleUser, Content: "Prepare a sales report and send it to team@example.com via email"},
			},
		}

		e := standard.New()
		err := e.RunGoals(ctx, exec, nil)

		require.NoError(t, err)
		require.NotNil(t, exec.Goals)
		assert.NotEmpty(t, exec.Goals.Content)

		// Delivery may or may not be present depending on LLM response
		// If present, verify structure
		if exec.Goals.Delivery != nil {
			// Type should be valid if present
			if exec.Goals.Delivery.Type != "" {
				validTypes := []types.DeliveryType{
					types.DeliveryEmail, types.DeliveryWebhook,
					types.DeliveryFile, types.DeliveryNotify,
				}
				found := false
				for _, vt := range validTypes {
					if exec.Goals.Delivery.Type == vt {
						found = true
						break
					}
				}
				// Note: LLM might return non-standard types, we accept them but log
				t.Logf("Delivery type: %s (valid: %v)", exec.Goals.Delivery.Type, found)
			}
		}
	})
}

func TestDeliveryTypeValidation(t *testing.T) {
	t.Run("valid delivery types", func(t *testing.T) {
		validTypes := []types.DeliveryType{
			types.DeliveryEmail,
			types.DeliveryWebhook,
			types.DeliveryFile,
			types.DeliveryNotify,
		}

		for _, dt := range validTypes {
			assert.True(t, standard.IsValidDeliveryType(dt), "should be valid: %s", dt)
		}
	})

	t.Run("invalid delivery types", func(t *testing.T) {
		invalidTypes := []types.DeliveryType{
			"invalid",
			"sms",
			"",
		}

		for _, dt := range invalidTypes {
			assert.False(t, standard.IsValidDeliveryType(dt), "should be invalid: %s", dt)
		}
	})
}

// ============================================================================
// InputFormatter Tests for P1
// ============================================================================

func TestInputFormatterFormatRobotIdentity(t *testing.T) {
	formatter := standard.NewInputFormatter()

	t.Run("formats robot identity correctly", func(t *testing.T) {
		robot := &types.Robot{
			MemberID: "test-robot",
			Config: &types.Config{
				Identity: &types.Identity{
					Role:   "Sales Analyst",
					Duties: []string{"Analyze sales data", "Generate reports"},
					Rules:  []string{"Be accurate", "Be concise"},
				},
			},
		}

		content := formatter.FormatRobotIdentity(robot)

		assert.Contains(t, content, "## Robot Identity")
		assert.Contains(t, content, "Sales Analyst")
		assert.Contains(t, content, "Analyze sales data")
		assert.Contains(t, content, "Generate reports")
		assert.Contains(t, content, "Be accurate")
		assert.Contains(t, content, "Be concise")
	})

	t.Run("returns empty for nil robot", func(t *testing.T) {
		content := formatter.FormatRobotIdentity(nil)
		assert.Empty(t, content)
	})

	t.Run("returns empty for robot without config", func(t *testing.T) {
		robot := &types.Robot{MemberID: "test"}
		content := formatter.FormatRobotIdentity(robot)
		assert.Empty(t, content)
	})

	t.Run("returns empty for robot without identity", func(t *testing.T) {
		robot := &types.Robot{
			MemberID: "test",
			Config:   &types.Config{},
		}
		content := formatter.FormatRobotIdentity(robot)
		assert.Empty(t, content)
	})

	t.Run("handles identity with only role", func(t *testing.T) {
		robot := &types.Robot{
			MemberID: "test",
			Config: &types.Config{
				Identity: &types.Identity{
					Role: "Simple Bot",
				},
			},
		}

		content := formatter.FormatRobotIdentity(robot)

		assert.Contains(t, content, "## Robot Identity")
		assert.Contains(t, content, "Simple Bot")
		assert.NotContains(t, content, "Duties")
		assert.NotContains(t, content, "Rules")
	})
}

// ============================================================================
// Helper Functions
// ============================================================================

// createGoalsTestRobot creates a test robot with specified goals agent
func createGoalsTestRobot(t *testing.T, agentID string) *types.Robot {
	t.Helper()
	return &types.Robot{
		MemberID:    "test-robot-1",
		TeamID:      "test-team-1",
		DisplayName: "Test Robot",
		Config: &types.Config{
			Identity: &types.Identity{
				Role:   "Test Assistant",
				Duties: []string{"Testing", "Validation"},
			},
			Resources: &types.Resources{
				Phases: map[types.Phase]string{
					types.PhaseGoals: agentID,
				},
			},
		},
	}
}

// createGoalsTestExecution creates a test execution for goals phase
func createGoalsTestExecution(robot *types.Robot, trigger types.TriggerType) *types.Execution {
	exec := &types.Execution{
		ID:          "test-exec-goals-1",
		MemberID:    robot.MemberID,
		TeamID:      robot.TeamID,
		TriggerType: trigger,
		StartTime:   time.Now(),
		Status:      types.ExecRunning,
		Phase:       types.PhaseGoals,
	}
	exec.SetRobot(robot)
	return exec
}
