//go:build unit

package context_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/yaoapp/yao/agent/context"
)

// =============================================================================
// InterruptController Creation Tests
// =============================================================================

func TestInterruptNewController(t *testing.T) {
	ctrl := context.NewInterruptController()
	require.NotNil(t, ctrl)

	assert.Equal(t, 0, ctrl.GetPendingCount())
	assert.Nil(t, ctrl.Check())
	assert.Nil(t, ctrl.Peek())
	assert.False(t, ctrl.IsInterrupted())
}

// =============================================================================
// SendSignal and Check Tests
// =============================================================================

func TestInterruptSendSignalAndCheck(t *testing.T) {
	t.Run("SendAndCheckGracefulSignal", func(t *testing.T) {
		ctrl := context.NewInterruptController()
		ctrl.Start("test-ctx-1")
		defer ctrl.Stop()

		signal := &context.InterruptSignal{
			Type: context.InterruptGraceful,
			Messages: []context.Message{
				{Role: context.RoleUser, Content: "Hello from interrupt"},
			},
			Timestamp: time.Now().UnixMilli(),
		}

		err := ctrl.SendSignal(signal)
		require.NoError(t, err)

		time.Sleep(100 * time.Millisecond)

		received := ctrl.Check()
		require.NotNil(t, received)
		assert.Equal(t, context.InterruptGraceful, received.Type)
		require.Len(t, received.Messages, 1)
		assert.Equal(t, "Hello from interrupt", received.Messages[0].Content)
	})

	t.Run("CheckRemovesSignal", func(t *testing.T) {
		ctrl := context.NewInterruptController()
		ctrl.Start("test-ctx-2")
		defer ctrl.Stop()

		signal := &context.InterruptSignal{
			Type:      context.InterruptGraceful,
			Messages:  []context.Message{{Role: context.RoleUser, Content: "check test"}},
			Timestamp: time.Now().UnixMilli(),
		}

		err := ctrl.SendSignal(signal)
		require.NoError(t, err)
		time.Sleep(100 * time.Millisecond)

		checked := ctrl.Check()
		require.NotNil(t, checked)

		checked2 := ctrl.Check()
		assert.Nil(t, checked2)
	})

	t.Run("CheckWithNoSignal", func(t *testing.T) {
		ctrl := context.NewInterruptController()
		checked := ctrl.Check()
		assert.Nil(t, checked)
	})

	t.Run("SendForceSignal", func(t *testing.T) {
		ctrl := context.NewInterruptController()
		ctrl.Start("test-ctx-3")
		defer ctrl.Stop()

		signal := &context.InterruptSignal{
			Type: context.InterruptForce,
			Messages: []context.Message{
				{Role: context.RoleUser, Content: "Force interrupt"},
			},
			Timestamp: time.Now().UnixMilli(),
		}

		err := ctrl.SendSignal(signal)
		require.NoError(t, err)
		time.Sleep(100 * time.Millisecond)

		received := ctrl.Check()
		require.NotNil(t, received)
		assert.Equal(t, context.InterruptForce, received.Type)
	})
}

// =============================================================================
// Peek Tests
// =============================================================================

func TestInterruptPeek(t *testing.T) {
	t.Run("PeekDoesNotRemove", func(t *testing.T) {
		ctrl := context.NewInterruptController()
		ctrl.Start("test-ctx-peek")
		defer ctrl.Stop()

		signal := &context.InterruptSignal{
			Type:      context.InterruptGraceful,
			Messages:  []context.Message{{Role: context.RoleUser, Content: "peek test"}},
			Timestamp: time.Now().UnixMilli(),
		}

		err := ctrl.SendSignal(signal)
		require.NoError(t, err)
		time.Sleep(100 * time.Millisecond)

		peeked1 := ctrl.Peek()
		require.NotNil(t, peeked1)

		peeked2 := ctrl.Peek()
		require.NotNil(t, peeked2)

		assert.Equal(t, peeked1.Messages[0].Content, peeked2.Messages[0].Content)
	})

	t.Run("PeekWithNoSignal", func(t *testing.T) {
		ctrl := context.NewInterruptController()
		assert.Nil(t, ctrl.Peek())
	})
}

// =============================================================================
// CheckWithMerge Tests
// =============================================================================

func TestInterruptCheckWithMerge(t *testing.T) {
	t.Run("MergeMultipleSignals", func(t *testing.T) {
		ctrl := context.NewInterruptController()
		ctrl.Start("test-ctx-merge")
		defer ctrl.Stop()

		messages := []string{"First message", "Second message", "Third message"}
		for _, msg := range messages {
			signal := &context.InterruptSignal{
				Type:      context.InterruptGraceful,
				Messages:  []context.Message{{Role: context.RoleUser, Content: msg}},
				Timestamp: time.Now().UnixMilli(),
			}
			err := ctrl.SendSignal(signal)
			require.NoError(t, err)
			time.Sleep(10 * time.Millisecond)
		}

		time.Sleep(100 * time.Millisecond)

		assert.Equal(t, 3, ctrl.GetPendingCount())

		merged := ctrl.CheckWithMerge()
		require.NotNil(t, merged)
		require.Len(t, merged.Messages, 3)

		for i, msg := range messages {
			assert.Equal(t, msg, merged.Messages[i].Content)
		}

		assert.Equal(t, true, merged.Metadata["merged"])
		assert.Equal(t, 3, merged.Metadata["merged_count"])

		assert.Equal(t, 0, ctrl.GetPendingCount())
	})

	t.Run("MergeSingleSignal", func(t *testing.T) {
		ctrl := context.NewInterruptController()
		ctrl.Start("test-ctx-merge-single")
		defer ctrl.Stop()

		signal := &context.InterruptSignal{
			Type:      context.InterruptGraceful,
			Messages:  []context.Message{{Role: context.RoleUser, Content: "single signal"}},
			Timestamp: time.Now().UnixMilli(),
		}

		err := ctrl.SendSignal(signal)
		require.NoError(t, err)
		time.Sleep(100 * time.Millisecond)

		merged := ctrl.CheckWithMerge()
		require.NotNil(t, merged)
		require.Len(t, merged.Messages, 1)

		if merged.Metadata != nil {
			assert.NotEqual(t, true, merged.Metadata["merged"])
		}
	})

	t.Run("MergeWithNoSignals", func(t *testing.T) {
		ctrl := context.NewInterruptController()
		merged := ctrl.CheckWithMerge()
		assert.Nil(t, merged)
	})

	t.Run("MergeFiveSignals", func(t *testing.T) {
		ctrl := context.NewInterruptController()
		ctrl.Start("test-ctx-merge-5")
		defer ctrl.Stop()

		msgs := []string{
			"First message",
			"Second message",
			"Third message",
			"Fourth message",
			"Fifth message",
		}

		for _, msg := range msgs {
			signal := &context.InterruptSignal{
				Type:      context.InterruptGraceful,
				Messages:  []context.Message{{Role: context.RoleUser, Content: msg}},
				Timestamp: time.Now().UnixMilli(),
			}
			err := ctrl.SendSignal(signal)
			require.NoError(t, err)
			time.Sleep(10 * time.Millisecond)
		}

		time.Sleep(100 * time.Millisecond)

		assert.Equal(t, 5, ctrl.GetPendingCount())

		merged := ctrl.CheckWithMerge()
		require.NotNil(t, merged)
		require.Len(t, merged.Messages, 5)

		for i, msg := range msgs {
			assert.Equal(t, msg, merged.Messages[i].Content)
		}

		assert.Equal(t, true, merged.Metadata["merged"])
		assert.Equal(t, 5, merged.Metadata["merged_count"])

		assert.Equal(t, 0, ctrl.GetPendingCount())
	})
}

// =============================================================================
// Clear Tests
// =============================================================================

func TestInterruptClear(t *testing.T) {
	t.Run("ClearRemovesAllSignals", func(t *testing.T) {
		ctrl := context.NewInterruptController()
		ctrl.Start("test-ctx-clear")
		defer ctrl.Stop()

		for i := 0; i < 3; i++ {
			signal := &context.InterruptSignal{
				Type:      context.InterruptGraceful,
				Messages:  []context.Message{{Role: context.RoleUser, Content: "test"}},
				Timestamp: time.Now().UnixMilli(),
			}
			ctrl.SendSignal(signal)
		}

		time.Sleep(100 * time.Millisecond)
		assert.True(t, ctrl.GetPendingCount() > 0)

		ctrl.Clear()

		assert.Equal(t, 0, ctrl.GetPendingCount())
		assert.Nil(t, ctrl.Check())
		assert.Nil(t, ctrl.Peek())
	})

	t.Run("ClearOnEmptyController", func(t *testing.T) {
		ctrl := context.NewInterruptController()
		ctrl.Clear()
		assert.Equal(t, 0, ctrl.GetPendingCount())
	})
}

// =============================================================================
// GetPendingCount Tests
// =============================================================================

func TestInterruptGetPendingCount(t *testing.T) {
	t.Run("CountIncludesCurrent", func(t *testing.T) {
		ctrl := context.NewInterruptController()
		ctrl.Start("test-ctx-count")
		defer ctrl.Stop()

		signal := &context.InterruptSignal{
			Type:      context.InterruptGraceful,
			Messages:  []context.Message{{Role: context.RoleUser, Content: "test"}},
			Timestamp: time.Now().UnixMilli(),
		}

		ctrl.SendSignal(signal)
		time.Sleep(100 * time.Millisecond)

		assert.Equal(t, 1, ctrl.GetPendingCount())
	})

	t.Run("ZeroOnEmpty", func(t *testing.T) {
		ctrl := context.NewInterruptController()
		assert.Equal(t, 0, ctrl.GetPendingCount())
	})

	t.Run("MultipleSignals", func(t *testing.T) {
		ctrl := context.NewInterruptController()
		ctrl.Start("test-ctx-count-multi")
		defer ctrl.Stop()

		for i := 0; i < 5; i++ {
			signal := &context.InterruptSignal{
				Type:      context.InterruptGraceful,
				Messages:  []context.Message{{Role: context.RoleUser, Content: "test"}},
				Timestamp: time.Now().UnixMilli(),
			}
			ctrl.SendSignal(signal)
			time.Sleep(10 * time.Millisecond)
		}

		time.Sleep(100 * time.Millisecond)
		assert.Equal(t, 5, ctrl.GetPendingCount())
	})
}

// =============================================================================
// IsInterrupted Tests
// =============================================================================

func TestInterruptIsInterrupted(t *testing.T) {
	t.Run("NotInterruptedInitially", func(t *testing.T) {
		ctrl := context.NewInterruptController()
		assert.False(t, ctrl.IsInterrupted())
	})

	t.Run("GracefulDoesNotSetInterrupted", func(t *testing.T) {
		ctrl := context.NewInterruptController()
		ctrl.Start("test-ctx-graceful-no-interrupted")
		defer ctrl.Stop()

		signal := &context.InterruptSignal{
			Type:      context.InterruptGraceful,
			Messages:  []context.Message{{Role: context.RoleUser, Content: "graceful"}},
			Timestamp: time.Now().UnixMilli(),
		}

		ctrl.SendSignal(signal)
		time.Sleep(100 * time.Millisecond)

		assert.False(t, ctrl.IsInterrupted())
	})

	t.Run("ForceWithEmptyMessagesCancelsContext", func(t *testing.T) {
		ctrl := context.NewInterruptController()
		ctrl.Start("test-ctx-force-cancel")
		defer ctrl.Stop()

		interruptCtx := ctrl.Context()

		signal := &context.InterruptSignal{
			Type:      context.InterruptForce,
			Messages:  []context.Message{},
			Timestamp: time.Now().UnixMilli(),
		}

		ctrl.SendSignal(signal)
		time.Sleep(100 * time.Millisecond)

		select {
		case <-interruptCtx.Done():
			// The old context was cancelled as expected
		case <-time.After(200 * time.Millisecond):
			t.Error("Old context was not cancelled after force interrupt with empty messages")
		}
	})

	t.Run("ForceWithMessagesDoesNotCancelContext", func(t *testing.T) {
		ctrl := context.NewInterruptController()
		ctrl.Start("test-ctx-force-msgs")
		defer ctrl.Stop()

		interruptCtx := ctrl.Context()

		signal := &context.InterruptSignal{
			Type:      context.InterruptForce,
			Messages:  []context.Message{{Role: context.RoleUser, Content: "has message"}},
			Timestamp: time.Now().UnixMilli(),
		}

		ctrl.SendSignal(signal)
		time.Sleep(100 * time.Millisecond)

		select {
		case <-interruptCtx.Done():
			t.Error("Context should not be cancelled for force interrupt with messages")
		default:
			// expected
		}
	})
}

// =============================================================================
// Context() Tests
// =============================================================================

func TestInterruptContext(t *testing.T) {
	t.Run("ReturnsValidContext", func(t *testing.T) {
		ctrl := context.NewInterruptController()
		ctx := ctrl.Context()
		require.NotNil(t, ctx)

		select {
		case <-ctx.Done():
			t.Error("Context should not be done initially")
		default:
			// expected
		}
	})

	t.Run("ContextRecreatedAfterForceCancel", func(t *testing.T) {
		ctrl := context.NewInterruptController()
		ctrl.Start("test-ctx-recreate")
		defer ctrl.Stop()

		oldCtx := ctrl.Context()

		signal := &context.InterruptSignal{
			Type:      context.InterruptForce,
			Messages:  []context.Message{},
			Timestamp: time.Now().UnixMilli(),
		}
		ctrl.SendSignal(signal)
		time.Sleep(100 * time.Millisecond)

		select {
		case <-oldCtx.Done():
			// old context cancelled as expected
		case <-time.After(200 * time.Millisecond):
			t.Error("Old context should be cancelled")
		}

		newCtx := ctrl.Context()
		require.NotNil(t, newCtx)

		select {
		case <-newCtx.Done():
			t.Error("New context should not be done")
		default:
			// expected: new context is fresh
		}
	})
}

// =============================================================================
// Handler Tests
// =============================================================================

func TestInterruptHandler(t *testing.T) {
	t.Run("SetHandlerNilSafe", func(t *testing.T) {
		ctrl := context.NewInterruptController()
		ctrl.SetHandler(nil)
		// Should not panic
	})

	t.Run("NilControllerSetHandler", func(t *testing.T) {
		var ctrl *context.InterruptController
		ctrl.SetHandler(func(c *context.Context, s *context.InterruptSignal) error {
			return nil
		})
		// Should not panic
	})
}

// =============================================================================
// Multiple Signals Queuing Tests
// =============================================================================

func TestInterruptMultipleSignalsQueuing(t *testing.T) {
	t.Run("QueuePreservesOrder", func(t *testing.T) {
		ctrl := context.NewInterruptController()
		ctrl.Start("test-ctx-queue-order")
		defer ctrl.Stop()

		for i := 0; i < 5; i++ {
			signal := &context.InterruptSignal{
				Type:      context.InterruptGraceful,
				Messages:  []context.Message{{Role: context.RoleUser, Content: "test"}},
				Timestamp: time.Now().UnixMilli(),
			}
			ctrl.SendSignal(signal)
			time.Sleep(10 * time.Millisecond)
		}

		time.Sleep(100 * time.Millisecond)

		for i := 0; i < 5; i++ {
			signal := ctrl.Check()
			require.NotNil(t, signal, "Signal %d should not be nil", i)
		}

		assert.Nil(t, ctrl.Check())
	})

	t.Run("CheckTransitionsToNextSignal", func(t *testing.T) {
		ctrl := context.NewInterruptController()
		ctrl.Start("test-ctx-transition")
		defer ctrl.Stop()

		for i := 0; i < 3; i++ {
			signal := &context.InterruptSignal{
				Type: context.InterruptGraceful,
				Messages: []context.Message{
					{Role: context.RoleUser, Content: "test"},
				},
				Timestamp: time.Now().UnixMilli(),
				Metadata:  map[string]interface{}{"index": i},
			}
			ctrl.SendSignal(signal)
			time.Sleep(10 * time.Millisecond)
		}

		time.Sleep(100 * time.Millisecond)

		s1 := ctrl.Check()
		require.NotNil(t, s1)

		s2 := ctrl.Check()
		require.NotNil(t, s2)

		s3 := ctrl.Check()
		require.NotNil(t, s3)

		s4 := ctrl.Check()
		assert.Nil(t, s4)
	})
}

// =============================================================================
// Nil Controller Safety Tests
// =============================================================================

func TestInterruptNilControllerSafety(t *testing.T) {
	var ctrl *context.InterruptController

	t.Run("NilCheck", func(t *testing.T) {
		assert.Nil(t, ctrl.Check())
	})

	t.Run("NilPeek", func(t *testing.T) {
		assert.Nil(t, ctrl.Peek())
	})

	t.Run("NilCheckWithMerge", func(t *testing.T) {
		assert.Nil(t, ctrl.CheckWithMerge())
	})

	t.Run("NilIsInterrupted", func(t *testing.T) {
		assert.False(t, ctrl.IsInterrupted())
	})

	t.Run("NilContext", func(t *testing.T) {
		ctx := ctrl.Context()
		assert.NotNil(t, ctx)
	})

	t.Run("NilGetPendingCount", func(t *testing.T) {
		assert.Equal(t, 0, ctrl.GetPendingCount())
	})

	t.Run("NilClear", func(t *testing.T) {
		ctrl.Clear()
	})

	t.Run("NilStop", func(t *testing.T) {
		ctrl.Stop()
	})

	t.Run("NilSendSignal", func(t *testing.T) {
		err := ctrl.SendSignal(&context.InterruptSignal{
			Type:      context.InterruptGraceful,
			Messages:  []context.Message{{Role: context.RoleUser, Content: "test"}},
			Timestamp: time.Now().UnixMilli(),
		})
		assert.Error(t, err)
	})
}

// =============================================================================
// SendSignal Edge Cases
// =============================================================================

func TestInterruptSendSignalEdgeCases(t *testing.T) {
	t.Run("SendSignalTimeout", func(t *testing.T) {
		ctrl := context.NewInterruptController()

		for i := 0; i < 10; i++ {
			signal := &context.InterruptSignal{
				Type:      context.InterruptGraceful,
				Messages:  []context.Message{{Role: context.RoleUser, Content: "fill"}},
				Timestamp: time.Now().UnixMilli(),
			}
			ctrl.SendSignal(signal)
		}

		signal := &context.InterruptSignal{
			Type:      context.InterruptGraceful,
			Messages:  []context.Message{{Role: context.RoleUser, Content: "overflow"}},
			Timestamp: time.Now().UnixMilli(),
		}

		err := ctrl.SendSignal(signal)
		assert.Error(t, err)
	})
}

// =============================================================================
// Interrupt Type Constants Tests
// =============================================================================

func TestInterruptTypeConstants(t *testing.T) {
	assert.Equal(t, context.InterruptType("graceful"), context.InterruptGraceful)
	assert.Equal(t, context.InterruptType("force"), context.InterruptForce)
}

func TestInterruptActionConstants(t *testing.T) {
	assert.Equal(t, context.InterruptAction("continue"), context.InterruptActionContinue)
	assert.Equal(t, context.InterruptAction("restart"), context.InterruptActionRestart)
	assert.Equal(t, context.InterruptAction("abort"), context.InterruptActionAbort)
}
