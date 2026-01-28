package manager_test

// Integration tests for execution control (Pause/Resume/Stop)
// Tests Manager's execution control methods and ExecutionController

import (
	"context"
	"encoding/json"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/yaoapp/gou/model"
	"github.com/yaoapp/xun/capsule"
	agentcontext "github.com/yaoapp/yao/agent/context"
	"github.com/yaoapp/yao/agent/robot/manager"
	"github.com/yaoapp/yao/agent/robot/pool"
	"github.com/yaoapp/yao/agent/robot/types"
	"github.com/yaoapp/yao/agent/testutils"
)

// ==================== Pause/Resume Tests ====================

// TestIntegrationExecutionPauseResume tests pausing and resuming executions
func TestIntegrationExecutionPauseResume(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	testutils.Prepare(t)
	defer testutils.Clean(t)

	cleanupIntegrationRobots(t)
	defer cleanupIntegrationRobots(t)

	t.Run("pause and resume execution", func(t *testing.T) {
		setupControlTestRobot(t, "robot_integ_ctrl_pause", "team_integ_ctrl")

		// Use slow executor to have time to pause
		exec := &slowExecutor{delay: 500 * time.Millisecond}

		config := &manager.Config{
			TickInterval: 100 * time.Millisecond,
			PoolConfig:   &pool.Config{WorkerSize: 3, QueueSize: 20},
		}
		m := manager.NewWithConfig(config)
		m.Pool().SetExecutor(exec)

		err := m.Start()
		require.NoError(t, err)
		defer m.Stop()

		// Verify robot is loaded into cache
		robot := m.Cache().Get("robot_integ_ctrl_pause")
		require.NotNil(t, robot, "Robot should be loaded into cache")

		ctx := types.NewContext(context.Background(), nil)

		// Trigger execution
		req := &types.InterveneRequest{
			MemberID: "robot_integ_ctrl_pause",
			Action:   types.ActionTaskAdd,
			Messages: []agentcontext.Message{
				{Role: agentcontext.RoleUser, Content: "Test task"},
			},
		}
		result, err := m.Intervene(ctx, req)
		require.NoError(t, err)
		execID := result.ExecutionID

		// Wait for execution to be tracked
		time.Sleep(100 * time.Millisecond)

		// Pause execution
		err = m.PauseExecution(ctx, execID)
		assert.NoError(t, err)

		// Verify paused
		status, err := m.GetExecutionStatus(execID)
		assert.NoError(t, err)
		assert.True(t, status.IsPaused(), "Execution should be paused")

		// Resume execution
		err = m.ResumeExecution(ctx, execID)
		assert.NoError(t, err)

		// Verify resumed
		status, err = m.GetExecutionStatus(execID)
		assert.NoError(t, err)
		assert.False(t, status.IsPaused(), "Execution should be resumed")
	})

	t.Run("pause non-existent execution", func(t *testing.T) {
		m := manager.New()
		err := m.Start()
		require.NoError(t, err)
		defer m.Stop()

		ctx := types.NewContext(context.Background(), nil)

		err = m.PauseExecution(ctx, "nonexistent_exec")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not found")
	})

	t.Run("resume non-paused execution", func(t *testing.T) {
		setupControlTestRobot(t, "robot_integ_ctrl_resume", "team_integ_ctrl")

		exec := &slowExecutor{delay: 500 * time.Millisecond}

		config := &manager.Config{
			TickInterval: 100 * time.Millisecond,
			PoolConfig:   &pool.Config{WorkerSize: 3, QueueSize: 20},
		}
		m := manager.NewWithConfig(config)
		m.Pool().SetExecutor(exec)

		err := m.Start()
		require.NoError(t, err)
		defer m.Stop()

		ctx := types.NewContext(context.Background(), nil)

		// Trigger execution
		req := &types.InterveneRequest{
			MemberID: "robot_integ_ctrl_resume",
			Action:   types.ActionTaskAdd,
			Messages: []agentcontext.Message{
				{Role: agentcontext.RoleUser, Content: "Test task"},
			},
		}
		result, err := m.Intervene(ctx, req)
		require.NoError(t, err)
		execID := result.ExecutionID

		// Wait for execution to be tracked
		time.Sleep(100 * time.Millisecond)

		// Resume without pausing first - should be safe
		err = m.ResumeExecution(ctx, execID)
		// May or may not error depending on implementation
		// The important thing is it doesn't panic
	})
}

// ==================== Stop Tests ====================

// TestIntegrationExecutionStop tests stopping executions
func TestIntegrationExecutionStop(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	testutils.Prepare(t)
	defer testutils.Clean(t)

	cleanupIntegrationRobots(t)
	defer cleanupIntegrationRobots(t)

	t.Run("stop execution", func(t *testing.T) {
		setupControlTestRobot(t, "robot_integ_ctrl_stop", "team_integ_ctrl")

		exec := &slowExecutor{delay: 1 * time.Second}

		config := &manager.Config{
			TickInterval: 100 * time.Millisecond,
			PoolConfig:   &pool.Config{WorkerSize: 3, QueueSize: 20},
		}
		m := manager.NewWithConfig(config)
		m.Pool().SetExecutor(exec)

		err := m.Start()
		require.NoError(t, err)
		defer m.Stop()

		ctx := types.NewContext(context.Background(), nil)

		// Trigger execution
		req := &types.InterveneRequest{
			MemberID: "robot_integ_ctrl_stop",
			Action:   types.ActionTaskAdd,
			Messages: []agentcontext.Message{
				{Role: agentcontext.RoleUser, Content: "Test task"},
			},
		}
		result, err := m.Intervene(ctx, req)
		require.NoError(t, err)
		execID := result.ExecutionID

		// Wait for execution to be tracked
		time.Sleep(100 * time.Millisecond)

		// Stop execution
		err = m.StopExecution(ctx, execID)
		assert.NoError(t, err)

		// Execution should be removed from tracking
		_, err = m.GetExecutionStatus(execID)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not found")
	})

	t.Run("stop non-existent execution", func(t *testing.T) {
		m := manager.New()
		err := m.Start()
		require.NoError(t, err)
		defer m.Stop()

		ctx := types.NewContext(context.Background(), nil)

		err = m.StopExecution(ctx, "nonexistent_exec")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not found")
	})
}

// ==================== List Executions Tests ====================

// TestIntegrationListExecutions tests listing executions
func TestIntegrationListExecutions(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	testutils.Prepare(t)
	defer testutils.Clean(t)

	cleanupIntegrationRobots(t)
	defer cleanupIntegrationRobots(t)

	t.Run("list all executions", func(t *testing.T) {
		setupControlTestRobot(t, "robot_integ_ctrl_list1", "team_integ_ctrl")
		setupControlTestRobot(t, "robot_integ_ctrl_list2", "team_integ_ctrl")

		exec := &slowExecutor{delay: 500 * time.Millisecond}

		config := &manager.Config{
			TickInterval: 100 * time.Millisecond,
			PoolConfig:   &pool.Config{WorkerSize: 5, QueueSize: 20},
		}
		m := manager.NewWithConfig(config)
		m.Pool().SetExecutor(exec)

		err := m.Start()
		require.NoError(t, err)
		defer m.Stop()

		ctx := types.NewContext(context.Background(), nil)

		// Trigger multiple executions
		execIDs := make([]string, 0)
		for _, memberID := range []string{"robot_integ_ctrl_list1", "robot_integ_ctrl_list2"} {
			req := &types.InterveneRequest{
				MemberID: memberID,
				Action:   types.ActionTaskAdd,
				Messages: []agentcontext.Message{
					{Role: agentcontext.RoleUser, Content: "Test task"},
				},
			}
			result, err := m.Intervene(ctx, req)
			require.NoError(t, err)
			execIDs = append(execIDs, result.ExecutionID)
		}

		// Wait for executions to be tracked
		time.Sleep(100 * time.Millisecond)

		// List all executions
		execs := m.ListExecutions()
		assert.GreaterOrEqual(t, len(execs), 2, "Should have at least 2 executions")

		// Verify our executions are in the list
		foundCount := 0
		for _, e := range execs {
			for _, id := range execIDs {
				if e.ID == id {
					foundCount++
				}
			}
		}
		assert.Equal(t, 2, foundCount, "Both executions should be in list")
	})

	t.Run("list executions by member", func(t *testing.T) {
		setupControlTestRobot(t, "robot_integ_ctrl_member1", "team_integ_ctrl")
		setupControlTestRobot(t, "robot_integ_ctrl_member2", "team_integ_ctrl")

		exec := &slowExecutor{delay: 500 * time.Millisecond}

		config := &manager.Config{
			TickInterval: 100 * time.Millisecond,
			PoolConfig:   &pool.Config{WorkerSize: 5, QueueSize: 20},
		}
		m := manager.NewWithConfig(config)
		m.Pool().SetExecutor(exec)

		err := m.Start()
		require.NoError(t, err)
		defer m.Stop()

		ctx := types.NewContext(context.Background(), nil)

		// Trigger 3 executions for robot 1
		for i := 0; i < 3; i++ {
			req := &types.InterveneRequest{
				MemberID: "robot_integ_ctrl_member1",
				Action:   types.ActionTaskAdd,
				Messages: []agentcontext.Message{
					{Role: agentcontext.RoleUser, Content: "Test task"},
				},
			}
			_, err := m.Intervene(ctx, req)
			require.NoError(t, err)
		}

		// Trigger 2 executions for robot 2
		for i := 0; i < 2; i++ {
			req := &types.InterveneRequest{
				MemberID: "robot_integ_ctrl_member2",
				Action:   types.ActionTaskAdd,
				Messages: []agentcontext.Message{
					{Role: agentcontext.RoleUser, Content: "Test task"},
				},
			}
			_, err := m.Intervene(ctx, req)
			require.NoError(t, err)
		}

		// Wait for executions to be tracked
		time.Sleep(100 * time.Millisecond)

		// List executions for robot 1
		execs1 := m.ListExecutionsByMember("robot_integ_ctrl_member1")
		assert.GreaterOrEqual(t, len(execs1), 1, "Robot 1 should have executions")

		// List executions for robot 2
		execs2 := m.ListExecutionsByMember("robot_integ_ctrl_member2")
		assert.GreaterOrEqual(t, len(execs2), 1, "Robot 2 should have executions")

		// Verify member IDs
		for _, e := range execs1 {
			assert.Equal(t, "robot_integ_ctrl_member1", e.MemberID)
		}
		for _, e := range execs2 {
			assert.Equal(t, "robot_integ_ctrl_member2", e.MemberID)
		}
	})
}

// ==================== Multiple Control Operations Tests ====================

// TestIntegrationMultipleControlOperations tests sequences of control operations
func TestIntegrationMultipleControlOperations(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	testutils.Prepare(t)
	defer testutils.Clean(t)

	cleanupIntegrationRobots(t)
	defer cleanupIntegrationRobots(t)

	t.Run("pause-resume-pause-stop sequence", func(t *testing.T) {
		setupControlTestRobot(t, "robot_integ_ctrl_seq", "team_integ_ctrl")

		exec := &slowExecutor{delay: 2 * time.Second}

		config := &manager.Config{
			TickInterval: 100 * time.Millisecond,
			PoolConfig:   &pool.Config{WorkerSize: 3, QueueSize: 20},
		}
		m := manager.NewWithConfig(config)
		m.Pool().SetExecutor(exec)

		err := m.Start()
		require.NoError(t, err)
		defer m.Stop()

		ctx := types.NewContext(context.Background(), nil)

		// Trigger execution
		req := &types.InterveneRequest{
			MemberID: "robot_integ_ctrl_seq",
			Action:   types.ActionTaskAdd,
			Messages: []agentcontext.Message{
				{Role: agentcontext.RoleUser, Content: "Test task"},
			},
		}
		result, err := m.Intervene(ctx, req)
		require.NoError(t, err)
		execID := result.ExecutionID

		// Wait for tracking
		time.Sleep(100 * time.Millisecond)

		// Pause
		err = m.PauseExecution(ctx, execID)
		assert.NoError(t, err)
		status, _ := m.GetExecutionStatus(execID)
		assert.True(t, status.IsPaused())

		// Resume
		err = m.ResumeExecution(ctx, execID)
		assert.NoError(t, err)
		status, _ = m.GetExecutionStatus(execID)
		assert.False(t, status.IsPaused())

		// Pause again
		err = m.PauseExecution(ctx, execID)
		assert.NoError(t, err)
		status, _ = m.GetExecutionStatus(execID)
		assert.True(t, status.IsPaused())

		// Stop
		err = m.StopExecution(ctx, execID)
		assert.NoError(t, err)
		_, err = m.GetExecutionStatus(execID)
		assert.Error(t, err) // Should be removed
	})

	t.Run("concurrent control operations", func(t *testing.T) {
		setupControlTestRobot(t, "robot_integ_ctrl_conc", "team_integ_ctrl")

		exec := &slowExecutor{delay: 1 * time.Second}

		config := &manager.Config{
			TickInterval: 100 * time.Millisecond,
			PoolConfig:   &pool.Config{WorkerSize: 3, QueueSize: 20},
		}
		m := manager.NewWithConfig(config)
		m.Pool().SetExecutor(exec)

		err := m.Start()
		require.NoError(t, err)
		defer m.Stop()

		ctx := types.NewContext(context.Background(), nil)

		// Trigger execution
		req := &types.InterveneRequest{
			MemberID: "robot_integ_ctrl_conc",
			Action:   types.ActionTaskAdd,
			Messages: []agentcontext.Message{
				{Role: agentcontext.RoleUser, Content: "Test task"},
			},
		}
		result, err := m.Intervene(ctx, req)
		require.NoError(t, err)
		execID := result.ExecutionID

		// Wait for tracking
		time.Sleep(100 * time.Millisecond)

		// Concurrent pause/resume operations should not panic
		var wg sync.WaitGroup
		for i := 0; i < 10; i++ {
			wg.Add(2)
			go func() {
				defer wg.Done()
				m.PauseExecution(ctx, execID)
			}()
			go func() {
				defer wg.Done()
				m.ResumeExecution(ctx, execID)
			}()
		}

		// Wait with timeout
		done := make(chan struct{})
		go func() {
			wg.Wait()
			close(done)
		}()

		select {
		case <-done:
			// Success - no deadlock
		case <-time.After(5 * time.Second):
			t.Fatal("Concurrent control operations caused deadlock")
		}
	})
}

// ==================== Helper Types ====================

// slowExecutor is an executor with configurable delay
type slowExecutor struct {
	delay   time.Duration
	count   int32
	current int32
}

func (e *slowExecutor) Execute(ctx *types.Context, robot *types.Robot, trigger types.TriggerType, data interface{}) (*types.Execution, error) {
	return e.ExecuteWithControl(ctx, robot, trigger, data, "", nil)
}

func (e *slowExecutor) ExecuteWithID(ctx *types.Context, robot *types.Robot, trigger types.TriggerType, data interface{}, execID string) (*types.Execution, error) {
	return e.ExecuteWithControl(ctx, robot, trigger, data, execID, nil)
}

func (e *slowExecutor) ExecuteWithControl(ctx *types.Context, robot *types.Robot, trigger types.TriggerType, data interface{}, execID string, control types.ExecutionControl) (*types.Execution, error) {
	if robot == nil {
		return nil, types.ErrRobotNotFound
	}

	// Use provided execID or generate one
	if execID == "" {
		execID = "exec_slow_" + robot.MemberID
	}
	exec := &types.Execution{
		ID:          execID,
		MemberID:    robot.MemberID,
		TeamID:      robot.TeamID,
		TriggerType: trigger,
		StartTime:   time.Now(),
		Status:      types.ExecPending,
	}

	if !robot.TryAcquireSlot(exec) {
		return nil, types.ErrQuotaExceeded
	}
	defer robot.RemoveExecution(exec.ID)

	atomic.AddInt32(&e.current, 1)
	defer atomic.AddInt32(&e.current, -1)

	exec.Status = types.ExecRunning
	time.Sleep(e.delay)

	exec.Status = types.ExecCompleted
	now := time.Now()
	exec.EndTime = &now

	atomic.AddInt32(&e.count, 1)
	return exec, nil
}

func (e *slowExecutor) ExecCount() int {
	return int(atomic.LoadInt32(&e.count))
}

func (e *slowExecutor) CurrentCount() int {
	return int(atomic.LoadInt32(&e.current))
}

func (e *slowExecutor) Reset() {
	atomic.StoreInt32(&e.count, 0)
	atomic.StoreInt32(&e.current, 0)
}

// ==================== Test Data Setup Helpers ====================

// setupControlTestRobot creates a robot for control testing
func setupControlTestRobot(t *testing.T, memberID, teamID string) {
	m := model.Select("__yao.member")
	tableName := m.MetaData.Table.Name
	qb := capsule.Query()

	robotConfig := map[string]interface{}{
		"identity": map[string]interface{}{
			"role":   "Control Test Robot",
			"duties": []string{"Test execution control"},
		},
		"quota": map[string]interface{}{
			"max":      5,
			"queue":    20,
			"priority": 5,
		},
		"triggers": map[string]interface{}{
			"clock":     map[string]interface{}{"enabled": true},
			"intervene": map[string]interface{}{"enabled": true},
			"event":     map[string]interface{}{"enabled": true},
		},
	}
	configJSON, _ := json.Marshal(robotConfig)

	err := qb.Table(tableName).Insert([]map[string]interface{}{
		{
			"member_id":       memberID,
			"team_id":         teamID,
			"member_type":     "robot",
			"display_name":    "Control Test Robot " + memberID,
			"status":          "active",
			"role_id":         "member",
			"autonomous_mode": true,
			"robot_status":    "idle",
			"robot_config":    string(configJSON),
		},
	})
	if err != nil {
		t.Fatalf("Failed to insert %s: %v", memberID, err)
	}
}
