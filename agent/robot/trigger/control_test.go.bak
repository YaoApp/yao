package trigger_test

import (
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/yaoapp/yao/agent/robot/trigger"
	"github.com/yaoapp/yao/agent/robot/types"
)

// ==================== ExecutionController Tests ====================

func TestExecutionControllerTrack(t *testing.T) {
	t.Run("tracks new execution", func(t *testing.T) {
		ctrl := trigger.NewExecutionController()

		exec := ctrl.Track("exec_001", "robot_001", "team_001")

		assert.NotNil(t, exec)
		assert.Equal(t, "exec_001", exec.ID)
		assert.Equal(t, "robot_001", exec.MemberID)
		assert.Equal(t, "team_001", exec.TeamID)
		assert.Equal(t, types.ExecRunning, exec.Status)
		assert.False(t, exec.IsPaused())
		assert.False(t, exec.IsCancelled())
	})

	t.Run("get tracked execution", func(t *testing.T) {
		ctrl := trigger.NewExecutionController()
		ctrl.Track("exec_001", "robot_001", "team_001")

		exec := ctrl.Get("exec_001")
		assert.NotNil(t, exec)
		assert.Equal(t, "exec_001", exec.ID)
	})

	t.Run("get non-existent execution returns nil", func(t *testing.T) {
		ctrl := trigger.NewExecutionController()

		exec := ctrl.Get("non_existent")
		assert.Nil(t, exec)
	})
}

func TestExecutionControllerList(t *testing.T) {
	t.Run("list all executions", func(t *testing.T) {
		ctrl := trigger.NewExecutionController()
		ctrl.Track("exec_001", "robot_001", "team_001")
		ctrl.Track("exec_002", "robot_002", "team_001")
		ctrl.Track("exec_003", "robot_001", "team_002")

		list := ctrl.List()
		assert.Len(t, list, 3)
	})

	t.Run("list by member", func(t *testing.T) {
		ctrl := trigger.NewExecutionController()
		ctrl.Track("exec_001", "robot_001", "team_001")
		ctrl.Track("exec_002", "robot_002", "team_001")
		ctrl.Track("exec_003", "robot_001", "team_002")

		list := ctrl.ListByMember("robot_001")
		assert.Len(t, list, 2)

		list = ctrl.ListByMember("robot_002")
		assert.Len(t, list, 1)

		list = ctrl.ListByMember("robot_003")
		assert.Len(t, list, 0)
	})
}

func TestExecutionControllerUntrack(t *testing.T) {
	t.Run("untrack removes execution", func(t *testing.T) {
		ctrl := trigger.NewExecutionController()
		ctrl.Track("exec_001", "robot_001", "team_001")

		assert.NotNil(t, ctrl.Get("exec_001"))

		ctrl.Untrack("exec_001")

		assert.Nil(t, ctrl.Get("exec_001"))
	})

	t.Run("untrack non-existent does not panic", func(t *testing.T) {
		ctrl := trigger.NewExecutionController()

		assert.NotPanics(t, func() {
			ctrl.Untrack("non_existent")
		})
	})
}

// ==================== Pause/Resume Tests ====================

func TestExecutionControllerPause(t *testing.T) {
	t.Run("pause execution", func(t *testing.T) {
		ctrl := trigger.NewExecutionController()
		exec := ctrl.Track("exec_001", "robot_001", "team_001")

		err := ctrl.Pause("exec_001")
		assert.NoError(t, err)
		assert.True(t, exec.IsPaused())
		assert.NotNil(t, exec.PausedAt)
	})

	t.Run("pause non-existent returns error", func(t *testing.T) {
		ctrl := trigger.NewExecutionController()

		err := ctrl.Pause("non_existent")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not found")
	})

	t.Run("pause already paused returns error", func(t *testing.T) {
		ctrl := trigger.NewExecutionController()
		ctrl.Track("exec_001", "robot_001", "team_001")

		err := ctrl.Pause("exec_001")
		assert.NoError(t, err)

		err = ctrl.Pause("exec_001")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "already paused")
	})
}

func TestExecutionControllerResume(t *testing.T) {
	t.Run("resume paused execution", func(t *testing.T) {
		ctrl := trigger.NewExecutionController()
		exec := ctrl.Track("exec_001", "robot_001", "team_001")

		ctrl.Pause("exec_001")
		assert.True(t, exec.IsPaused())

		err := ctrl.Resume("exec_001")
		assert.NoError(t, err)
		assert.False(t, exec.IsPaused())
		assert.Nil(t, exec.PausedAt)
	})

	t.Run("resume non-existent returns error", func(t *testing.T) {
		ctrl := trigger.NewExecutionController()

		err := ctrl.Resume("non_existent")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not found")
	})

	t.Run("resume not paused returns error", func(t *testing.T) {
		ctrl := trigger.NewExecutionController()
		ctrl.Track("exec_001", "robot_001", "team_001")

		err := ctrl.Resume("exec_001")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not paused")
	})
}

// ==================== Stop Tests ====================

func TestExecutionControllerStop(t *testing.T) {
	t.Run("stop execution", func(t *testing.T) {
		ctrl := trigger.NewExecutionController()
		exec := ctrl.Track("exec_001", "robot_001", "team_001")

		err := ctrl.Stop("exec_001")
		assert.NoError(t, err)
		assert.True(t, exec.IsCancelled())
		assert.Equal(t, types.ExecCancelled, exec.Status)

		// Should be removed from tracking
		assert.Nil(t, ctrl.Get("exec_001"))
	})

	t.Run("stop non-existent returns error", func(t *testing.T) {
		ctrl := trigger.NewExecutionController()

		err := ctrl.Stop("non_existent")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not found")
	})
}

// ==================== ControlledExecution Methods Tests ====================

func TestControlledExecutionContext(t *testing.T) {
	t.Run("context is valid", func(t *testing.T) {
		ctrl := trigger.NewExecutionController()
		exec := ctrl.Track("exec_001", "robot_001", "team_001")

		ctx := exec.Context()
		assert.NotNil(t, ctx)

		// Context should not be done yet
		select {
		case <-ctx.Done():
			t.Fatal("context should not be done")
		default:
			// OK
		}
	})

	t.Run("context done after stop", func(t *testing.T) {
		ctrl := trigger.NewExecutionController()
		exec := ctrl.Track("exec_001", "robot_001", "team_001")

		ctx := exec.Context()
		ctrl.Stop("exec_001")

		select {
		case <-ctx.Done():
			// OK
		case <-time.After(100 * time.Millisecond):
			t.Fatal("context should be done after stop")
		}
	})
}

func TestControlledExecutionCheckCancelled(t *testing.T) {
	t.Run("not cancelled returns nil", func(t *testing.T) {
		ctrl := trigger.NewExecutionController()
		exec := ctrl.Track("exec_001", "robot_001", "team_001")

		err := exec.CheckCancelled()
		assert.NoError(t, err)
	})

	t.Run("cancelled returns error", func(t *testing.T) {
		ctrl := trigger.NewExecutionController()
		exec := ctrl.Track("exec_001", "robot_001", "team_001")

		ctrl.Stop("exec_001")

		err := exec.CheckCancelled()
		assert.Error(t, err)
		assert.Equal(t, types.ErrExecutionCancelled, err)
	})
}

func TestControlledExecutionUpdatePhase(t *testing.T) {
	ctrl := trigger.NewExecutionController()
	exec := ctrl.Track("exec_001", "robot_001", "team_001")

	assert.Equal(t, types.PhaseInspiration, exec.Phase)

	exec.UpdatePhase(types.PhaseGoals)
	assert.Equal(t, types.PhaseGoals, exec.Phase)

	exec.UpdatePhase(types.PhaseTasks)
	assert.Equal(t, types.PhaseTasks, exec.Phase)
}

func TestControlledExecutionUpdateStatus(t *testing.T) {
	ctrl := trigger.NewExecutionController()
	exec := ctrl.Track("exec_001", "robot_001", "team_001")

	assert.Equal(t, types.ExecRunning, exec.Status)

	exec.UpdateStatus(types.ExecCompleted)
	assert.Equal(t, types.ExecCompleted, exec.Status)

	exec.UpdateStatus(types.ExecFailed)
	assert.Equal(t, types.ExecFailed, exec.Status)
}

// ==================== WaitIfPaused Tests ====================

func TestControlledExecutionWaitIfPaused(t *testing.T) {
	t.Run("returns immediately if not paused", func(t *testing.T) {
		ctrl := trigger.NewExecutionController()
		exec := ctrl.Track("exec_001", "robot_001", "team_001")

		done := make(chan error)
		go func() {
			done <- exec.WaitIfPaused()
		}()

		select {
		case err := <-done:
			assert.NoError(t, err)
		case <-time.After(100 * time.Millisecond):
			t.Fatal("WaitIfPaused should return immediately when not paused")
		}
	})

	t.Run("does not infinite loop when paused without resume", func(t *testing.T) {
		// This test verifies the fix for the infinite loop bug
		// where WaitIfPaused would spin if pauseCh was closed but paused remained true
		ctrl := trigger.NewExecutionController()
		exec := ctrl.Track("exec_001", "robot_001", "team_001")

		ctrl.Pause("exec_001")

		// Start WaitIfPaused in a goroutine
		done := make(chan error)
		go func() {
			done <- exec.WaitIfPaused()
		}()

		// Wait a bit - if there's an infinite loop, CPU would spike
		// The goroutine should be blocked, not spinning
		time.Sleep(100 * time.Millisecond)

		// Now stop the execution - this should unblock WaitIfPaused
		ctrl.Stop("exec_001")

		select {
		case err := <-done:
			// Should get cancellation error
			assert.Error(t, err)
		case <-time.After(200 * time.Millisecond):
			t.Fatal("WaitIfPaused should unblock after stop")
		}
	})

	t.Run("rapid pause-resume-pause does not cause issues", func(t *testing.T) {
		// Test TOCTOU race condition handling
		ctrl := trigger.NewExecutionController()
		exec := ctrl.Track("exec_001", "robot_001", "team_001")

		// Pause first
		ctrl.Pause("exec_001")

		done := make(chan error)
		go func() {
			done <- exec.WaitIfPaused()
		}()

		// Rapid resume then pause again
		time.Sleep(10 * time.Millisecond)
		ctrl.Resume("exec_001")

		// WaitIfPaused should return (the original resumeCh was closed)
		select {
		case err := <-done:
			assert.NoError(t, err)
		case <-time.After(200 * time.Millisecond):
			t.Fatal("WaitIfPaused should return after resume")
		}
	})

	t.Run("blocks when paused, resumes after resume", func(t *testing.T) {
		ctrl := trigger.NewExecutionController()
		exec := ctrl.Track("exec_001", "robot_001", "team_001")

		ctrl.Pause("exec_001")

		done := make(chan error)
		go func() {
			done <- exec.WaitIfPaused()
		}()

		// Should be blocked
		select {
		case <-done:
			t.Fatal("WaitIfPaused should block when paused")
		case <-time.After(50 * time.Millisecond):
			// OK, still blocked
		}

		// Resume
		ctrl.Resume("exec_001")

		// Should unblock
		select {
		case err := <-done:
			assert.NoError(t, err)
		case <-time.After(100 * time.Millisecond):
			t.Fatal("WaitIfPaused should unblock after resume")
		}
	})

	t.Run("returns error when cancelled while paused", func(t *testing.T) {
		ctrl := trigger.NewExecutionController()
		exec := ctrl.Track("exec_001", "robot_001", "team_001")

		ctrl.Pause("exec_001")

		done := make(chan error)
		go func() {
			done <- exec.WaitIfPaused()
		}()

		// Should be blocked
		select {
		case <-done:
			t.Fatal("WaitIfPaused should block when paused")
		case <-time.After(50 * time.Millisecond):
			// OK, still blocked
		}

		// Stop instead of resume
		ctrl.Stop("exec_001")

		// Should unblock with error
		select {
		case err := <-done:
			assert.Error(t, err)
			assert.Equal(t, types.ErrExecutionCancelled, err)
		case <-time.After(100 * time.Millisecond):
			t.Fatal("WaitIfPaused should unblock after stop")
		}
	})
}

// ==================== Concurrent Access Tests ====================

func TestExecutionControllerConcurrency(t *testing.T) {
	t.Run("concurrent track and list", func(t *testing.T) {
		ctrl := trigger.NewExecutionController()
		var wg sync.WaitGroup

		// Concurrent tracking
		for i := 0; i < 100; i++ {
			wg.Add(1)
			go func(id int) {
				defer wg.Done()
				ctrl.Track(
					"exec_"+string(rune('0'+id%10)),
					"robot_"+string(rune('0'+id%5)),
					"team_001",
				)
			}(i)
		}

		// Concurrent listing
		for i := 0; i < 50; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				_ = ctrl.List()
			}()
		}

		wg.Wait()
		// No race conditions or panics
	})

	t.Run("concurrent pause/resume", func(t *testing.T) {
		ctrl := trigger.NewExecutionController()
		ctrl.Track("exec_001", "robot_001", "team_001")

		var wg sync.WaitGroup

		// Concurrent pause/resume attempts
		for i := 0; i < 50; i++ {
			wg.Add(2)
			go func() {
				defer wg.Done()
				_ = ctrl.Pause("exec_001")
			}()
			go func() {
				defer wg.Done()
				_ = ctrl.Resume("exec_001")
			}()
		}

		wg.Wait()
		// No race conditions or panics
	})
}
