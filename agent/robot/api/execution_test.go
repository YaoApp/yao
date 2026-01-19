package api_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/yaoapp/yao/agent/robot/api"
	"github.com/yaoapp/yao/agent/robot/types"
	"github.com/yaoapp/yao/agent/testutils"
)

// TestGetExecutionValidation tests parameter validation for GetExecution
func TestGetExecutionValidation(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	testutils.Prepare(t)
	defer testutils.Clean(t)

	ctx := types.NewContext(context.Background(), nil)

	t.Run("returns error for empty execution_id", func(t *testing.T) {
		exec, err := api.GetExecution(ctx, "")
		assert.Error(t, err)
		assert.Nil(t, exec)
		assert.Contains(t, err.Error(), "execution_id is required")
	})

	t.Run("returns error for non-existent execution", func(t *testing.T) {
		exec, err := api.GetExecution(ctx, "non_existent_exec_id_xyz")
		assert.Error(t, err)
		assert.Nil(t, exec)
	})
}

// TestListExecutionsValidation tests parameter validation for ListExecutions
func TestListExecutionsValidation(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	testutils.Prepare(t)
	defer testutils.Clean(t)

	ctx := types.NewContext(context.Background(), nil)

	t.Run("returns error for empty member_id", func(t *testing.T) {
		result, err := api.ListExecutions(ctx, "", nil)
		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "member_id is required")
	})

	t.Run("applies default pagination when query is nil", func(t *testing.T) {
		result, err := api.ListExecutions(ctx, "test_member", nil)
		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, 1, result.Page)
		assert.Equal(t, 20, result.PageSize)
	})

	t.Run("caps pagesize at 100", func(t *testing.T) {
		result, err := api.ListExecutions(ctx, "test_member", &api.ExecutionQuery{
			Page:     1,
			PageSize: 200,
		})
		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, 100, result.PageSize)
	})
}

// TestPauseExecutionValidation tests parameter validation for PauseExecution
func TestPauseExecutionValidation(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	testutils.Prepare(t)
	defer testutils.Clean(t)

	ctx := types.NewContext(context.Background(), nil)

	t.Run("returns error for empty execution_id", func(t *testing.T) {
		err := api.PauseExecution(ctx, "")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "execution_id is required")
	})

	t.Run("returns error when manager not started", func(t *testing.T) {
		err := api.PauseExecution(ctx, "test_exec_id")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not started")
	})
}

// TestResumeExecutionValidation tests parameter validation for ResumeExecution
func TestResumeExecutionValidation(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	testutils.Prepare(t)
	defer testutils.Clean(t)

	ctx := types.NewContext(context.Background(), nil)

	t.Run("returns error for empty execution_id", func(t *testing.T) {
		err := api.ResumeExecution(ctx, "")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "execution_id is required")
	})

	t.Run("returns error when manager not started", func(t *testing.T) {
		err := api.ResumeExecution(ctx, "test_exec_id")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not started")
	})
}

// TestStopExecutionValidation tests parameter validation for StopExecution
func TestStopExecutionValidation(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	testutils.Prepare(t)
	defer testutils.Clean(t)

	ctx := types.NewContext(context.Background(), nil)

	t.Run("returns error for empty execution_id", func(t *testing.T) {
		err := api.StopExecution(ctx, "")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "execution_id is required")
	})

	t.Run("returns error when manager not started", func(t *testing.T) {
		err := api.StopExecution(ctx, "test_exec_id")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not started")
	})
}

// TestGetExecutionStatusValidation tests parameter validation for GetExecutionStatus
func TestGetExecutionStatusValidation(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	testutils.Prepare(t)
	defer testutils.Clean(t)

	ctx := types.NewContext(context.Background(), nil)

	t.Run("returns error for empty execution_id", func(t *testing.T) {
		exec, err := api.GetExecutionStatus(ctx, "")
		assert.Error(t, err)
		assert.Nil(t, exec)
		assert.Contains(t, err.Error(), "execution_id is required")
	})

	t.Run("returns error for non-existent execution", func(t *testing.T) {
		exec, err := api.GetExecutionStatus(ctx, "non_existent_exec_id_xyz")
		assert.Error(t, err)
		assert.Nil(t, exec)
	})
}

// TestExecutionControlWithManagerStarted tests execution control APIs when manager is running
func TestExecutionControlWithManagerStarted(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	testutils.Prepare(t)
	defer testutils.Clean(t)

	// Start manager
	err := api.Start()
	require.NoError(t, err)
	defer api.Stop()

	ctx := types.NewContext(context.Background(), nil)

	t.Run("pause returns error for non-existent execution", func(t *testing.T) {
		err := api.PauseExecution(ctx, "non_existent_exec_id_xyz")
		assert.Error(t, err)
	})

	t.Run("resume returns error for non-existent execution", func(t *testing.T) {
		err := api.ResumeExecution(ctx, "non_existent_exec_id_xyz")
		assert.Error(t, err)
	})

	t.Run("stop returns error for non-existent execution", func(t *testing.T) {
		err := api.StopExecution(ctx, "non_existent_exec_id_xyz")
		assert.Error(t, err)
	})
}
