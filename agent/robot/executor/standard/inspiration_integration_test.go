//go:build integration

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

// ============================================================================
// P0 Inspiration Phase Tests
// ============================================================================

func TestRunInspirationBasic(t *testing.T) {
	identity := testprepare.PrepareSandbox(t)
	ctx := testCtx(identity)

	t.Run("generates_inspiration_report_with_clock_context", func(t *testing.T) {
		robot := newTestRobot(t, identity)
		exec := createTestExecution(robot, robottypes.TriggerClock)

		e := standard.New()
		err := e.RunInspiration(ctx, exec, nil)

		require.NoError(t, err)
		require.NotNil(t, exec.Inspiration)
		assert.NotEmpty(t, exec.Inspiration.Content)
		assert.NotNil(t, exec.Inspiration.Clock)
	})

	t.Run("includes_markdown_structure", func(t *testing.T) {
		robot := newTestRobot(t, identity)
		exec := createTestExecution(robot, robottypes.TriggerClock)

		e := standard.New()
		err := e.RunInspiration(ctx, exec, nil)

		require.NoError(t, err)
		content := exec.Inspiration.Content

		hasSection := strings.Contains(content, "##") ||
			strings.Contains(content, "Summary") ||
			strings.Contains(content, "Highlight") ||
			strings.Contains(content, "Recommend")

		assert.True(t, hasSection, "should contain markdown sections, got: %s", content[:min(200, len(content))])
	})
}

func TestRunInspirationClockContext(t *testing.T) {
	identity := testprepare.PrepareSandbox(t)
	ctx := testCtx(identity)

	t.Run("uses_clock_from_trigger_input", func(t *testing.T) {
		robot := newTestRobot(t, identity)
		exec := createTestExecution(robot, robottypes.TriggerClock)

		specificTime := time.Date(2024, 12, 31, 17, 0, 0, 0, time.UTC)
		exec.Input = &robottypes.TriggerInput{
			Clock: robottypes.NewClockContext(specificTime, "UTC"),
		}

		e := standard.New()
		err := e.RunInspiration(ctx, exec, nil)

		require.NoError(t, err)
		require.NotNil(t, exec.Inspiration.Clock)
		assert.Equal(t, specificTime.Year(), exec.Inspiration.Clock.Year)
		assert.Equal(t, int(specificTime.Month()), exec.Inspiration.Clock.Month)
		assert.Equal(t, specificTime.Day(), exec.Inspiration.Clock.DayOfMonth)
	})

	t.Run("creates_clock_context_when_not_provided", func(t *testing.T) {
		robot := newTestRobot(t, identity)
		exec := createTestExecution(robot, robottypes.TriggerClock)
		exec.Input = nil

		e := standard.New()
		err := e.RunInspiration(ctx, exec, nil)

		require.NoError(t, err)
		require.NotNil(t, exec.Inspiration.Clock)
		assert.Equal(t, time.Now().Year(), exec.Inspiration.Clock.Year)
	})
}

func TestRunInspirationErrorHandling(t *testing.T) {
	identity := testprepare.PrepareSandbox(t)
	ctx := testCtx(identity)

	t.Run("returns_error_when_robot_is_nil", func(t *testing.T) {
		exec := &robottypes.Execution{
			ID:          "test-exec-no-robot",
			TriggerType: robottypes.TriggerClock,
		}

		e := standard.New()
		err := e.RunInspiration(ctx, exec, nil)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "robot not found")
	})

	t.Run("returns_error_when_agent_not_found", func(t *testing.T) {
		robot := &robottypes.Robot{
			MemberID: "test-robot-bad-agent",
			TeamID:   identity.AlphaTeamID,
			Config: &robottypes.Config{
				Identity: &robottypes.Identity{Role: "Test"},
				Resources: &robottypes.Resources{
					Phases: map[robottypes.Phase]string{
						robottypes.PhaseInspiration: "non.existent.agent",
					},
				},
			},
		}
		exec := createTestExecution(robot, robottypes.TriggerClock)

		e := standard.New()
		err := e.RunInspiration(ctx, exec, nil)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "call failed")
	})
}

// ============================================================================
// InputFormatter Tests for P0
// ============================================================================

func TestInputFormatterClockContext(t *testing.T) {
	_ = testprepare.PrepareSandbox(t)

	t.Run("formats_clock_context_correctly", func(t *testing.T) {
		formatter := standard.NewInputFormatter()
		clock := robottypes.NewClockContext(
			time.Date(2024, 12, 31, 17, 30, 0, 0, time.UTC),
			"UTC",
		)
		robot := &robottypes.Robot{
			Config: &robottypes.Config{
				Identity: &robottypes.Identity{
					Role:   "Sales Assistant",
					Duties: []string{"Track metrics", "Send reports"},
				},
			},
		}

		content := formatter.FormatClockContext(clock, robot)

		assert.Contains(t, content, "Current Time Context")
		assert.Contains(t, content, "2024")
		assert.Contains(t, content, "12")
		assert.Contains(t, content, "31")
		assert.Contains(t, content, "Robot Identity")
		assert.Contains(t, content, "Sales Assistant")
		assert.Contains(t, content, "Track metrics")
	})

	t.Run("handles_nil_clock", func(t *testing.T) {
		formatter := standard.NewInputFormatter()
		content := formatter.FormatClockContext(nil, nil)
		assert.Empty(t, content)
	})

	t.Run("handles_nil_robot", func(t *testing.T) {
		formatter := standard.NewInputFormatter()
		clock := robottypes.NewClockContext(time.Now(), "")
		content := formatter.FormatClockContext(clock, nil)

		assert.Contains(t, content, "Current Time Context")
		assert.NotContains(t, content, "Robot Identity")
	})
}
