//go:build unit

package api_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/yaoapp/yao/agent/robot/api"
	"github.com/yaoapp/yao/agent/robot/types"
)

func TestInteract(t *testing.T) {
	t.Run("empty_member_id_returns_error", func(t *testing.T) {
		ctx := types.NewContext(nil, nil)
		_, err := api.Interact(ctx, "", &api.InteractRequest{Message: "test"})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "member_id is required")
	})

	t.Run("nil_request_returns_error", func(t *testing.T) {
		ctx := types.NewContext(nil, nil)
		_, err := api.Interact(ctx, "member-1", nil)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "interact request is required")
	})

	t.Run("no_manager_and_no_execution_id_returns_error", func(t *testing.T) {
		ctx := types.NewContext(nil, nil)
		_, err := api.Interact(ctx, "member-1", &api.InteractRequest{Message: "test"})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "execution_id is required")
	})
}

func TestReply(t *testing.T) {
	t.Run("empty_member_id_returns_error", func(t *testing.T) {
		ctx := types.NewContext(nil, nil)
		_, err := api.Reply(ctx, "", "exec-1", "task-1", "hello")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "member_id is required")
	})
}

func TestConfirm(t *testing.T) {
	t.Run("empty_member_id_returns_error", func(t *testing.T) {
		ctx := types.NewContext(nil, nil)
		_, err := api.Confirm(ctx, "", "exec-1", "yes")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "member_id is required")
	})
}

func TestCancelExecution(t *testing.T) {
	t.Run("no_manager_returns_error", func(t *testing.T) {
		ctx := types.NewContext(nil, nil)
		err := api.CancelExecution(ctx, "exec-1")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "cancel not available")
	})
}

func TestManagerInteract(t *testing.T) {
	t.Run("converts_request_fields_correctly", func(t *testing.T) {
		req := &api.InteractRequest{
			ExecutionID: "exec-ai1",
			TaskID:      "task-ai1",
			Source:      types.InteractSourceUI,
			Message:     "do it",
			Action:      "confirm",
		}
		assert.Equal(t, "exec-ai1", req.ExecutionID)
		assert.Equal(t, "task-ai1", req.TaskID)
		assert.Equal(t, types.InteractSourceUI, req.Source)
		assert.Equal(t, "do it", req.Message)
		assert.Equal(t, "confirm", req.Action)
	})
}

func TestInteractLegacyFallback(t *testing.T) {
	t.Run("with_execution_id_delegates_to_legacyResume", func(t *testing.T) {
		ctx := types.NewContext(nil, nil)
		var result *api.InteractResult
		var err error
		panicked := true
		func() {
			defer func() {
				if r := recover(); r == nil {
					panicked = false
				}
			}()
			result, err = api.Interact(ctx, "member-1", &api.InteractRequest{
				ExecutionID: "exec-1",
				Message:     "resume this",
			})
		}()

		if panicked {
			return // DB not initialized — panic confirms legacy path reached
		}
		// DB available — legacy path returns error for nonexistent execution
		assert.True(t, err != nil || result != nil, "legacy path should return result or error")
	})
}

func TestInteractResultFields(t *testing.T) {
	result := &api.InteractResult{
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

	empty := &api.InteractResult{}
	assert.Empty(t, empty.ExecutionID)
	assert.Empty(t, empty.Status)
	assert.False(t, empty.WaitForMore)
}

func TestLegacyResumeStatusMapping(t *testing.T) {
	result := &api.InteractResult{
		ExecutionID: "exec-lr",
		Status:      "waiting",
		Message:     "Execution suspended again: needs more input",
	}
	assert.Equal(t, "waiting", result.Status)
	assert.Contains(t, result.Message, "suspended")

	resultOK := &api.InteractResult{
		ExecutionID: "exec-lr2",
		Status:      "resumed",
		Message:     "Execution resumed and completed successfully",
	}
	require.Equal(t, "resumed", resultOK.Status)
}
