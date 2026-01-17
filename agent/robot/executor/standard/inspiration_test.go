package standard_test

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/yaoapp/yao/agent/robot/executor/standard"
	"github.com/yaoapp/yao/agent/robot/types"
	"github.com/yaoapp/yao/agent/testutils"
)

// ============================================================================
// P0 Inspiration Phase Tests
// ============================================================================

func TestRunInspirationBasic(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	testutils.Prepare(t)
	defer testutils.Clean(t)

	ctx := types.NewContext(context.Background(), testAuth())

	t.Run("generates inspiration report with clock context", func(t *testing.T) {
		// Create robot with inspiration agent configured
		robot := createTestRobot(t, "robot.inspiration")

		// Create executor and execution
		exec := createTestExecution(robot, types.TriggerClock)

		// Run inspiration phase
		e := standard.New()
		err := e.RunInspiration(ctx, exec, nil)

		require.NoError(t, err)
		require.NotNil(t, exec.Inspiration)
		assert.NotEmpty(t, exec.Inspiration.Content)
		assert.NotNil(t, exec.Inspiration.Clock)
	})

	t.Run("includes expected markdown sections", func(t *testing.T) {
		robot := createTestRobot(t, "robot.inspiration")
		exec := createTestExecution(robot, types.TriggerClock)

		e := standard.New()
		err := e.RunInspiration(ctx, exec, nil)

		require.NoError(t, err)
		content := exec.Inspiration.Content

		// Verify expected sections in markdown output
		// Note: LLM output is non-deterministic, so we check for likely sections
		hasSection := strings.Contains(content, "##") ||
			strings.Contains(content, "Summary") ||
			strings.Contains(content, "Highlight") ||
			strings.Contains(content, "Recommend")

		assert.True(t, hasSection, "should contain markdown sections, got: %s", content)
	})
}

func TestRunInspirationClockContext(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	testutils.Prepare(t)
	defer testutils.Clean(t)

	ctx := types.NewContext(context.Background(), testAuth())

	t.Run("uses clock from trigger input", func(t *testing.T) {
		robot := createTestRobot(t, "robot.inspiration")
		exec := createTestExecution(robot, types.TriggerClock)

		// Set specific clock context
		specificTime := time.Date(2024, 12, 31, 17, 0, 0, 0, time.UTC)
		exec.Input = &types.TriggerInput{
			Clock: types.NewClockContext(specificTime, "UTC"),
		}

		e := standard.New()
		err := e.RunInspiration(ctx, exec, nil)

		require.NoError(t, err)
		require.NotNil(t, exec.Inspiration.Clock)

		// Clock should match input
		assert.Equal(t, specificTime.Year(), exec.Inspiration.Clock.Year)
		assert.Equal(t, int(specificTime.Month()), exec.Inspiration.Clock.Month)
		assert.Equal(t, specificTime.Day(), exec.Inspiration.Clock.DayOfMonth)
	})

	t.Run("creates clock context when not provided", func(t *testing.T) {
		robot := createTestRobot(t, "robot.inspiration")
		exec := createTestExecution(robot, types.TriggerClock)
		exec.Input = nil // No input

		e := standard.New()
		err := e.RunInspiration(ctx, exec, nil)

		require.NoError(t, err)
		require.NotNil(t, exec.Inspiration.Clock)

		// Clock should be current time (approximately)
		now := time.Now()
		assert.Equal(t, now.Year(), exec.Inspiration.Clock.Year)
	})
}

func TestRunInspirationRobotIdentity(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	testutils.Prepare(t)
	defer testutils.Clean(t)

	ctx := types.NewContext(context.Background(), testAuth())

	t.Run("robot identity influences output", func(t *testing.T) {
		// Create robot with specific identity
		robot := &types.Robot{
			MemberID:    "test-robot-1",
			TeamID:      "test-team-1",
			DisplayName: "Sales Assistant",
			Config: &types.Config{
				Identity: &types.Identity{
					Role:   "Sales Assistant",
					Duties: []string{"Track sales metrics", "Prepare weekly reports"},
					Rules:  []string{"Focus on actionable insights"},
				},
				Resources: &types.Resources{
					Phases: map[types.Phase]string{
						types.PhaseInspiration: "robot.inspiration",
					},
				},
			},
		}

		exec := createTestExecution(robot, types.TriggerClock)

		e := standard.New()
		err := e.RunInspiration(ctx, exec, nil)

		require.NoError(t, err)
		assert.NotEmpty(t, exec.Inspiration.Content)

		// The content should be influenced by robot identity
		// (exact content varies due to LLM non-determinism)
	})
}

func TestRunInspirationErrorHandling(t *testing.T) {
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
		err := e.RunInspiration(ctx, exec, nil)

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
						types.PhaseInspiration: "non.existent.agent",
					},
				},
			},
		}
		exec := createTestExecution(robot, types.TriggerClock)

		e := standard.New()
		err := e.RunInspiration(ctx, exec, nil)

		// Real AgentCaller returns error for non-existent agent
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "call failed")
	})
}

func TestRunInspirationWithDefaultAgent(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	testutils.Prepare(t)
	defer testutils.Clean(t)

	ctx := types.NewContext(context.Background(), testAuth())

	t.Run("uses default agent when not configured", func(t *testing.T) {
		robot := &types.Robot{
			MemberID: "test-robot-1",
			TeamID:   "test-team-1",
			Config: &types.Config{
				Identity: &types.Identity{Role: "Test Robot"},
				// No Resources configured - should use default __yao.inspiration
			},
		}
		exec := createTestExecution(robot, types.TriggerClock)

		e := standard.New()
		err := e.RunInspiration(ctx, exec, nil)

		// This will fail if __yao.inspiration doesn't exist
		// In test environment, we expect it to fail with "agent not found"
		// In production, it would use the default agent
		if err != nil {
			assert.Contains(t, err.Error(), "call failed")
		}
	})
}

// ============================================================================
// InputFormatter Tests for P0
// ============================================================================

func TestInputFormatterClockContext(t *testing.T) {
	t.Run("formats clock context correctly", func(t *testing.T) {
		formatter := standard.NewInputFormatter()

		// Create a specific clock context
		clock := types.NewClockContext(
			time.Date(2024, 12, 31, 17, 30, 0, 0, time.UTC),
			"UTC",
		)

		robot := &types.Robot{
			Config: &types.Config{
				Identity: &types.Identity{
					Role:   "Sales Assistant",
					Duties: []string{"Track metrics", "Send reports"},
				},
			},
		}

		content := formatter.FormatClockContext(clock, robot)

		// Verify time context
		assert.Contains(t, content, "Current Time Context")
		assert.Contains(t, content, "2024")
		assert.Contains(t, content, "12")
		assert.Contains(t, content, "31")
		assert.Contains(t, content, "Tuesday") // Dec 31, 2024 is Tuesday

		// Verify robot identity
		assert.Contains(t, content, "Robot Identity")
		assert.Contains(t, content, "Sales Assistant")
		assert.Contains(t, content, "Track metrics")
	})

	t.Run("handles nil clock", func(t *testing.T) {
		formatter := standard.NewInputFormatter()
		content := formatter.FormatClockContext(nil, nil)
		assert.Empty(t, content)
	})

	t.Run("handles nil robot", func(t *testing.T) {
		formatter := standard.NewInputFormatter()
		clock := types.NewClockContext(time.Now(), "")
		content := formatter.FormatClockContext(clock, nil)

		// Should have time context but no robot identity
		assert.Contains(t, content, "Current Time Context")
		assert.NotContains(t, content, "Robot Identity")
	})

	t.Run("includes time markers", func(t *testing.T) {
		formatter := standard.NewInputFormatter()

		// Create a weekend + month start clock context
		// Jan 1, 2028 is Saturday (weekend + month start)
		clock := types.NewClockContext(
			time.Date(2028, 1, 1, 10, 0, 0, 0, time.UTC),
			"UTC",
		)

		content := formatter.FormatClockContext(clock, nil)

		assert.Contains(t, content, "Weekend")
		assert.Contains(t, content, "Month Start")
	})
}

// ============================================================================
// Helper Functions
// ============================================================================

// createTestRobot creates a test robot with specified inspiration agent
// Includes available expert agents so the Inspiration Agent knows what resources are available
//
// Note: The agent IDs listed in Resources.Agents must exist in yao-dev-app/assistants/experts/
// Current available experts: data-analyst, summarizer, text-writer, web-reader
func createTestRobot(t *testing.T, agentID string) *types.Robot {
	t.Helper()
	return &types.Robot{
		MemberID:    "test-robot-1",
		TeamID:      "test-team-1",
		DisplayName: "Test Robot",
		Config: &types.Config{
			Identity: &types.Identity{
				Role:   "Test Assistant",
				Duties: []string{"Testing", "Data Analysis", "Report Generation"},
			},
			Resources: &types.Resources{
				Phases: map[types.Phase]string{
					types.PhaseInspiration: agentID,
				},
				// Available expert agents that can be delegated to
				// These IDs correspond to assistants in yao-dev-app/assistants/experts/
				Agents: []string{
					"experts.data-analyst", // Data analysis and insights
					"experts.summarizer",   // Content summarization
					"experts.text-writer",  // Report and document generation
					"experts.web-reader",   // Web content extraction
				},
			},
			// Knowledge base collections (if any)
			KB: &types.KB{
				Collections: []string{"test-knowledge"},
			},
		},
	}
}

// createTestExecution creates a test execution for a robot
func createTestExecution(robot *types.Robot, trigger types.TriggerType) *types.Execution {
	exec := &types.Execution{
		ID:          "test-exec-1",
		MemberID:    robot.MemberID,
		TeamID:      robot.TeamID,
		TriggerType: trigger,
		StartTime:   time.Now(),
		Status:      types.ExecRunning,
		Phase:       types.PhaseInspiration,
		Input: &types.TriggerInput{
			Clock: types.NewClockContext(time.Now(), ""),
		},
	}
	exec.SetRobot(robot)
	return exec
}
