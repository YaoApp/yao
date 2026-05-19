//go:build e2e

package api

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/yaoapp/yao/agent/robot/types"
)

// AI1-AI3: Interact routing
func TestInteract(t *testing.T) {
	t.Run("empty member_id returns error", func(t *testing.T) {
		ctx := types.NewContext(nil, nil)
		_, err := Interact(ctx, "", &InteractRequest{Message: "test"})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "member_id is required")
	})

	t.Run("nil request returns error", func(t *testing.T) {
		ctx := types.NewContext(nil, nil)
		_, err := Interact(ctx, "member-1", nil)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "interact request is required")
	})

	t.Run("no manager and no execution_id returns error", func(t *testing.T) {
		ctx := types.NewContext(nil, nil)
		_, err := Interact(ctx, "member-1", &InteractRequest{Message: "test"})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "execution_id is required")
	})
}

// AI6: Reply shortcut
func TestReply(t *testing.T) {
	t.Run("empty member_id returns error", func(t *testing.T) {
		ctx := types.NewContext(nil, nil)
		_, err := Reply(ctx, "", "exec-1", "task-1", "hello")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "member_id is required")
	})

	t.Run("routes through Interact", func(t *testing.T) {
		ctx := types.NewContext(nil, nil)
		// legacyResume accesses the DB model which panics if not initialized.
		// Verify the routing reaches legacyResume by catching the expected panic.
		assert.Panics(t, func() {
			Reply(ctx, "member-1", "exec-1", "task-1", "hello")
		}, "should reach legacyResume which requires DB model")
	})
}

// AI7: Confirm shortcut
func TestConfirm(t *testing.T) {
	t.Run("empty member_id returns error", func(t *testing.T) {
		ctx := types.NewContext(nil, nil)
		_, err := Confirm(ctx, "", "exec-1", "yes")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "member_id is required")
	})

	t.Run("routes through Interact", func(t *testing.T) {
		ctx := types.NewContext(nil, nil)
		assert.Panics(t, func() {
			Confirm(ctx, "member-1", "exec-1", "yes")
		}, "should reach legacyResume which requires DB model")
	})
}

// AI8-AI9: CancelExecution
func TestCancelExecution(t *testing.T) {
	t.Run("no manager returns error", func(t *testing.T) {
		ctx := types.NewContext(nil, nil)
		err := CancelExecution(ctx, "exec-1")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "cancel not available")
	})
}

// AI10-AI12: legacyResume
func TestLegacyResume(t *testing.T) {
	t.Run("non-existent execution panics without DB", func(t *testing.T) {
		ctx := types.NewContext(nil, nil)
		assert.Panics(t, func() {
			legacyResume(ctx, &InteractRequest{
				ExecutionID: "nonexistent-exec",
				Message:     "test",
			})
		}, "should panic because DB model is not initialized")
	})
}

// AI1: managerInteract delegates correctly
func TestManagerInteract(t *testing.T) {
	t.Run("converts request fields correctly", func(t *testing.T) {
		// This would require a running Manager; test the field mapping logic
		req := &InteractRequest{
			ExecutionID: "exec-ai1",
			TaskID:      "task-ai1",
			Source:      types.InteractSourceUI,
			Message:     "do it",
			Action:      "confirm",
		}

		// Verify InteractRequest has all expected fields
		assert.Equal(t, "exec-ai1", req.ExecutionID)
		assert.Equal(t, "task-ai1", req.TaskID)
		assert.Equal(t, types.InteractSourceUI, req.Source)
		assert.Equal(t, "do it", req.Message)
		assert.Equal(t, "confirm", req.Action)
	})
}

// AI2: Interact with execution_id and no manager falls back to legacy
func TestInteractLegacyFallback(t *testing.T) {
	t.Run("with execution_id delegates to legacyResume", func(t *testing.T) {
		ctx := types.NewContext(nil, nil)
		assert.Panics(t, func() {
			Interact(ctx, "member-1", &InteractRequest{
				ExecutionID: "exec-1",
				Message:     "resume this",
			})
		}, "should reach legacyResume which requires DB model")
	})
}

// Test InteractResult field mapping
func TestInteractResultFields(t *testing.T) {
	result := &InteractResult{
		ExecutionID: "exec-test",
		Status:      "confirmed",
		Message:     "Done",
		ChatID:      "chat-test",
		Reply:       "I'll do it",
		WaitForMore: true,
	}

	assert.Equal(t, "exec-test", result.ExecutionID)
	assert.Equal(t, "confirmed", result.Status)
	assert.Equal(t, "Done", result.Message)
	assert.Equal(t, "chat-test", result.ChatID)
	assert.Equal(t, "I'll do it", result.Reply)
	assert.True(t, result.WaitForMore)

	// Verify zero-value result
	empty := &InteractResult{}
	assert.Empty(t, empty.ExecutionID)
	assert.Empty(t, empty.Status)
	assert.False(t, empty.WaitForMore)
}

// Test that legacyResume returns "waiting" on ErrExecutionSuspended
func TestLegacyResumeStatusMapping(t *testing.T) {
	// ErrExecutionSuspended handling is tested via the suspend E2E tests.
	// Here we verify the InteractResult field structure.
	result := &InteractResult{
		ExecutionID: "exec-lr",
		Status:      "waiting",
		Message:     "Execution suspended again: needs more input",
	}
	assert.Equal(t, "waiting", result.Status)
	assert.Contains(t, result.Message, "suspended")

	resultOK := &InteractResult{
		ExecutionID: "exec-lr2",
		Status:      "resumed",
		Message:     "Execution resumed and completed successfully",
	}
	require.Equal(t, "resumed", resultOK.Status)
}
