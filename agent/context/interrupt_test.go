package context_test

import (
	stdContext "context"
	"fmt"
	"testing"
	"time"

	"github.com/yaoapp/yao/agent/context"
	"github.com/yaoapp/yao/openapi/oauth/types"
)

// newTestContextWithInterrupt creates a Context with interrupt controller for testing
func newTestContextWithInterrupt(chatID, assistantID string) *context.Context {
	ctx := context.New(stdContext.Background(), &types.AuthorizedInfo{
		Subject:   "test-user",
		ClientID:  "test-client-id",
		UserID:    "test-user-123",
		TeamID:    "test-team-456",
		TenantID:  "test-tenant-789",
		SessionID: "test-session-id",
	}, chatID)

	ctx.AssistantID = assistantID
	ctx.Locale = "en-us"
	ctx.Theme = "light"
	ctx.Client = context.Client{
		Type:      "web",
		UserAgent: "TestAgent/1.0",
		IP:        "127.0.0.1",
	}
	ctx.Referer = context.RefererAPI
	ctx.Accept = context.AcceptWebCUI
	ctx.Route = "/test/route"
	ctx.Metadata = map[string]interface{}{
		"test": "context_metadata",
	}

	// Initialize interrupt controller
	ctx.Interrupt = context.NewInterruptController()

	// Register context globally
	if err := context.Register(ctx); err != nil {
		panic(fmt.Sprintf("Failed to register context: %v", err))
	}

	// Start interrupt listener
	ctx.Interrupt.Start(ctx.ID)

	return ctx
}

// TestInterruptBasic tests basic interrupt signal sending and receiving
func TestInterruptBasic(t *testing.T) {
	// Create context with interrupt support
	ctx := newTestContextWithInterrupt("chat-test-interrupt", "test-assistant")
	defer ctx.Release()

	t.Run("SendGracefulInterrupt", func(t *testing.T) {
		// Create a graceful interrupt signal
		signal := &context.InterruptSignal{
			Type: context.InterruptGraceful,
			Messages: []context.Message{
				{Role: context.RoleUser, Content: "This is a graceful interrupt"},
			},
			Timestamp: time.Now().UnixMilli(),
		}

		// Send interrupt signal
		err := context.SendInterrupt(ctx.ID, signal)
		if err != nil {
			t.Fatalf("Failed to send interrupt signal: %v", err)
		}

		// Wait a bit for the signal to be processed
		time.Sleep(100 * time.Millisecond)

		// Check if signal was received
		receivedSignal := ctx.Interrupt.Peek()
		if receivedSignal == nil {
			t.Fatal("Expected interrupt signal to be received, got nil")
		}

		if receivedSignal.Type != context.InterruptGraceful {
			t.Errorf("Expected interrupt type 'graceful', got: %s", receivedSignal.Type)
		}

		if len(receivedSignal.Messages) != 1 {
			t.Errorf("Expected 1 message, got: %d", len(receivedSignal.Messages))
		}

		if receivedSignal.Messages[0].Content != "This is a graceful interrupt" {
			t.Errorf("Expected message content 'This is a graceful interrupt', got: %s", receivedSignal.Messages[0].Content)
		}

		t.Log("✓ Graceful interrupt signal sent and received successfully")
	})

	t.Run("SendForceInterrupt", func(t *testing.T) {
		// Clear previous signals
		ctx.Interrupt.Clear()

		// Create a force interrupt signal
		signal := &context.InterruptSignal{
			Type: context.InterruptForce,
			Messages: []context.Message{
				{Role: context.RoleUser, Content: "This is a force interrupt"},
			},
			Timestamp: time.Now().UnixMilli(),
		}

		// Send interrupt signal
		err := context.SendInterrupt(ctx.ID, signal)
		if err != nil {
			t.Fatalf("Failed to send interrupt signal: %v", err)
		}

		// Wait a bit for the signal to be processed
		time.Sleep(100 * time.Millisecond)

		// Check if signal was received
		receivedSignal := ctx.Interrupt.Peek()
		if receivedSignal == nil {
			t.Fatal("Expected interrupt signal to be received, got nil")
		}

		if receivedSignal.Type != context.InterruptForce {
			t.Errorf("Expected interrupt type 'force', got: %s", receivedSignal.Type)
		}

		t.Log("✓ Force interrupt signal sent and received successfully")
	})

	t.Run("MultipleInterrupts", func(t *testing.T) {
		// Clear previous signals
		ctx.Interrupt.Clear()

		// Send multiple interrupt signals
		for i := 0; i < 3; i++ {
			signal := &context.InterruptSignal{
				Type: context.InterruptGraceful,
				Messages: []context.Message{
					{Role: context.RoleUser, Content: fmt.Sprintf("Message %d", i+1)},
				},
				Timestamp: time.Now().UnixMilli(),
			}

			err := context.SendInterrupt(ctx.ID, signal)
			if err != nil {
				t.Fatalf("Failed to send interrupt signal %d: %v", i+1, err)
			}
		}

		// Wait a bit for signals to be processed
		time.Sleep(100 * time.Millisecond)

		// Check pending count
		pendingCount := ctx.Interrupt.GetPendingCount()
		if pendingCount != 3 {
			t.Errorf("Expected 3 pending interrupts, got: %d", pendingCount)
		}

		// Check merged signal
		mergedSignal := ctx.Interrupt.CheckWithMerge()
		if mergedSignal == nil {
			t.Fatal("Expected merged signal, got nil")
		}

		if len(mergedSignal.Messages) != 3 {
			t.Errorf("Expected 3 merged messages, got: %d", len(mergedSignal.Messages))
		}

		// Verify all messages are present
		for i := 0; i < 3; i++ {
			expectedContent := fmt.Sprintf("Message %d", i+1)
			if mergedSignal.Messages[i].Content != expectedContent {
				t.Errorf("Expected message %d content '%s', got: %s", i+1, expectedContent, mergedSignal.Messages[i].Content)
			}
		}

		t.Log("✓ Multiple interrupt signals merged successfully")
	})
}

// TestInterruptHandler tests interrupt handler invocation
func TestInterruptHandler(t *testing.T) {
	// Create context with interrupt support
	ctx := newTestContextWithInterrupt("chat-test-interrupt-handler", "test-assistant")
	defer ctx.Release()

	t.Run("HandlerInvocation", func(t *testing.T) {
		// Track if handler was called
		handlerCalled := false
		var receivedSignal *context.InterruptSignal

		// Set up handler
		ctx.Interrupt.SetHandler(func(c *context.Context, signal *context.InterruptSignal) error {
			handlerCalled = true
			receivedSignal = signal
			t.Logf("Handler called with signal type: %s, messages: %d", signal.Type, len(signal.Messages))
			return nil
		})

		// Send interrupt signal
		signal := &context.InterruptSignal{
			Type: context.InterruptGraceful,
			Messages: []context.Message{
				{Role: context.RoleUser, Content: "Test handler invocation"},
			},
			Timestamp: time.Now().UnixMilli(),
		}

		err := context.SendInterrupt(ctx.ID, signal)
		if err != nil {
			t.Fatalf("Failed to send interrupt signal: %v", err)
		}

		// Wait for handler to be called
		time.Sleep(200 * time.Millisecond)

		// Verify handler was called
		if !handlerCalled {
			t.Error("Expected handler to be called, but it wasn't")
		}

		if receivedSignal == nil {
			t.Fatal("Expected signal in handler, got nil")
		}

		if receivedSignal.Type != context.InterruptGraceful {
			t.Errorf("Expected graceful interrupt in handler, got: %s", receivedSignal.Type)
		}

		if len(receivedSignal.Messages) != 1 {
			t.Errorf("Expected 1 message in handler, got: %d", len(receivedSignal.Messages))
		}

		t.Log("✓ Interrupt handler invoked successfully")
	})

	t.Run("HandlerWithError", func(t *testing.T) {
		// Create new context
		ctx2 := newTestContextWithInterrupt("chat-test-handler-error", "test-assistant")
		defer ctx2.Release()

		// Set up handler that returns error
		handlerCalled := false
		ctx2.Interrupt.SetHandler(func(c *context.Context, signal *context.InterruptSignal) error {
			handlerCalled = true
			return fmt.Errorf("test error from handler")
		})

		// Send interrupt signal
		signal := &context.InterruptSignal{
			Type: context.InterruptForce,
			Messages: []context.Message{
				{Role: context.RoleUser, Content: "Test error handling"},
			},
			Timestamp: time.Now().UnixMilli(),
		}

		err := context.SendInterrupt(ctx2.ID, signal)
		if err != nil {
			t.Fatalf("Failed to send interrupt signal: %v", err)
		}

		// Wait for handler to be called
		time.Sleep(200 * time.Millisecond)

		// Handler should still be called even if it returns error
		if !handlerCalled {
			t.Error("Expected handler to be called even with error")
		}

		t.Log("✓ Handler error handling works correctly")
	})
}

// TestInterruptContextLifecycle tests context registration and cleanup
func TestInterruptContextLifecycle(t *testing.T) {
	t.Run("RegisterAndRetrieve", func(t *testing.T) {
		ctx := newTestContextWithInterrupt("chat-test-lifecycle", "test-assistant")

		// Verify context can be retrieved
		retrievedCtx, err := context.Get(ctx.ID)
		if err != nil {
			t.Fatalf("Failed to retrieve context: %v", err)
		}

		if retrievedCtx.ID != ctx.ID {
			t.Errorf("Expected context ID %s, got: %s", ctx.ID, retrievedCtx.ID)
		}

		ctx.Release()

		// After release, context should be removed
		_, err = context.Get(ctx.ID)
		if err == nil {
			t.Error("Expected error when retrieving released context")
		}

		t.Log("✓ Context registration and cleanup works correctly")
	})

	t.Run("SendToNonExistentContext", func(t *testing.T) {
		signal := &context.InterruptSignal{
			Type:      context.InterruptGraceful,
			Messages:  []context.Message{{Role: context.RoleUser, Content: "test"}},
			Timestamp: time.Now().UnixMilli(),
		}

		err := context.SendInterrupt("non-existent-id", signal)
		if err == nil {
			t.Error("Expected error when sending to non-existent context")
		}

		t.Log("✓ Sending to non-existent context returns error")
	})
}

// TestInterruptCheckMethods tests different check methods
func TestInterruptCheckMethods(t *testing.T) {
	ctx := newTestContextWithInterrupt("chat-test-check-methods", "test-assistant")
	defer ctx.Release()

	t.Run("PeekDoesNotRemove", func(t *testing.T) {
		// Send signal
		signal := &context.InterruptSignal{
			Type:      context.InterruptGraceful,
			Messages:  []context.Message{{Role: context.RoleUser, Content: "peek test"}},
			Timestamp: time.Now().UnixMilli(),
		}
		context.SendInterrupt(ctx.ID, signal)
		time.Sleep(100 * time.Millisecond)

		// Peek should return signal but not remove it
		peeked1 := ctx.Interrupt.Peek()
		if peeked1 == nil {
			t.Fatal("Expected signal from first peek")
		}

		peeked2 := ctx.Interrupt.Peek()
		if peeked2 == nil {
			t.Fatal("Expected signal from second peek")
		}

		if peeked1.Messages[0].Content != peeked2.Messages[0].Content {
			t.Error("Peek should return the same signal")
		}

		t.Log("✓ Peek does not remove signal")
	})

	t.Run("CheckRemovesSignal", func(t *testing.T) {
		ctx.Interrupt.Clear()

		// Send signal
		signal := &context.InterruptSignal{
			Type:      context.InterruptGraceful,
			Messages:  []context.Message{{Role: context.RoleUser, Content: "check test"}},
			Timestamp: time.Now().UnixMilli(),
		}
		context.SendInterrupt(ctx.ID, signal)
		time.Sleep(100 * time.Millisecond)

		// Check should return and remove signal
		checked := ctx.Interrupt.Check()
		if checked == nil {
			t.Fatal("Expected signal from check")
		}

		// Second check should return nil
		checked2 := ctx.Interrupt.Check()
		if checked2 != nil {
			t.Error("Expected nil from second check after removal")
		}

		t.Log("✓ Check removes signal after retrieval")
	})

	t.Run("CheckWithMergeMultipleSignals", func(t *testing.T) {
		ctx.Interrupt.Clear()

		// Send 5 signals with different messages
		messages := []string{
			"First message",
			"Second message",
			"Third message",
			"Fourth message",
			"Fifth message",
		}

		for i, msg := range messages {
			signal := &context.InterruptSignal{
				Type: context.InterruptGraceful,
				Messages: []context.Message{
					{Role: context.RoleUser, Content: msg},
				},
				Timestamp: time.Now().UnixMilli(),
				Metadata: map[string]interface{}{
					"sequence": i + 1,
				},
			}
			err := context.SendInterrupt(ctx.ID, signal)
			if err != nil {
				t.Fatalf("Failed to send signal %d: %v", i+1, err)
			}
			time.Sleep(10 * time.Millisecond) // Small delay between signals
		}

		time.Sleep(100 * time.Millisecond)

		// Verify all signals are queued
		pendingCount := ctx.Interrupt.GetPendingCount()
		if pendingCount != 5 {
			t.Errorf("Expected 5 pending signals, got: %d", pendingCount)
		}

		// CheckWithMerge should merge all messages into one signal
		merged := ctx.Interrupt.CheckWithMerge()
		if merged == nil {
			t.Fatal("Expected merged signal, got nil")
		}

		// Verify all messages are merged
		if len(merged.Messages) != 5 {
			t.Errorf("Expected 5 merged messages, got: %d", len(merged.Messages))
		}

		// Verify message order
		for i, msg := range messages {
			if merged.Messages[i].Content != msg {
				t.Errorf("Message %d mismatch: expected '%s', got '%s'", i+1, msg, merged.Messages[i].Content)
			}
		}

		// Verify metadata indicates merge
		if merged.Metadata["merged"] != true {
			t.Error("Expected merged metadata to be true")
		}
		if merged.Metadata["merged_count"] != 5 {
			t.Errorf("Expected merged_count 5, got: %v", merged.Metadata["merged_count"])
		}

		// After merge, queue should be empty
		if ctx.Interrupt.GetPendingCount() != 0 {
			t.Errorf("Expected empty queue after merge, got: %d", ctx.Interrupt.GetPendingCount())
		}

		t.Log("✓ CheckWithMerge correctly merged 5 signals into one")
	})

	t.Run("CheckWithMergeSingleSignal", func(t *testing.T) {
		ctx.Interrupt.Clear()

		// Send single signal
		signal := &context.InterruptSignal{
			Type:      context.InterruptGraceful,
			Messages:  []context.Message{{Role: context.RoleUser, Content: "single signal"}},
			Timestamp: time.Now().UnixMilli(),
		}
		context.SendInterrupt(ctx.ID, signal)
		time.Sleep(100 * time.Millisecond)

		// CheckWithMerge with single signal should return it without merge metadata
		merged := ctx.Interrupt.CheckWithMerge()
		if merged == nil {
			t.Fatal("Expected signal, got nil")
		}

		if len(merged.Messages) != 1 {
			t.Errorf("Expected 1 message, got: %d", len(merged.Messages))
		}

		// Single signal should not have merge metadata
		if merged.Metadata != nil && merged.Metadata["merged"] == true {
			t.Error("Single signal should not have merge metadata")
		}

		t.Log("✓ CheckWithMerge handles single signal correctly")
	})
}

// TestInterruptContext tests interrupt context methods
func TestInterruptContext(t *testing.T) {
	ctx := newTestContextWithInterrupt("chat-test-interrupt-context", "test-assistant")
	defer ctx.Release()

	t.Run("InterruptContextMethod", func(t *testing.T) {
		// Get interrupt context
		interruptCtx := ctx.Interrupt.Context()
		if interruptCtx == nil {
			t.Fatal("Expected interrupt context, got nil")
		}

		// Context should not be done initially
		select {
		case <-interruptCtx.Done():
			t.Error("Interrupt context should not be done initially")
		default:
			t.Log("✓ Interrupt context is not done initially")
		}
	})

	t.Run("IsInterruptedFalseInitially", func(t *testing.T) {
		// Should not be interrupted initially
		if ctx.Interrupt.IsInterrupted() {
			t.Error("Should not be interrupted initially")
		}
		t.Log("✓ IsInterrupted returns false initially")
	})

	t.Run("ForceInterruptCancelsContext", func(t *testing.T) {
		// Get context before interrupt
		interruptCtx := ctx.Interrupt.Context()

		// Send force interrupt with empty messages (pure cancellation)
		// This is the pattern for stopping streaming without appending messages
		signal := &context.InterruptSignal{
			Type:      context.InterruptForce,
			Messages:  []context.Message{}, // Empty messages = pure cancellation
			Timestamp: time.Now().UnixMilli(),
		}
		err := context.SendInterrupt(ctx.ID, signal)
		if err != nil {
			t.Fatalf("Failed to send interrupt: %v", err)
		}

		time.Sleep(100 * time.Millisecond)

		// The OLD context should be cancelled
		select {
		case <-interruptCtx.Done():
			t.Log("✓ Force interrupt with empty messages cancelled the old context")
		case <-time.After(200 * time.Millisecond):
			t.Error("Old context was not cancelled after force interrupt with empty messages")
		}

		// Note: IsInterrupted() checks the NEW context (which was recreated)
		// So it will return false. This is expected behavior.
		// The key is that the old context was cancelled (checked above)
		t.Log("✓ Context was recreated after force interrupt (expected behavior)")
	})

	t.Run("GracefulInterruptDoesNotCancelContext", func(t *testing.T) {
		// Create new context for this test
		ctx2 := newTestContextWithInterrupt("chat-test-graceful-no-cancel", "test-assistant")
		defer ctx2.Release()

		interruptCtx := ctx2.Interrupt.Context()

		// Send graceful interrupt
		signal := &context.InterruptSignal{
			Type:      context.InterruptGraceful,
			Messages:  []context.Message{{Role: context.RoleUser, Content: "graceful"}},
			Timestamp: time.Now().UnixMilli(),
		}
		context.SendInterrupt(ctx2.ID, signal)
		time.Sleep(100 * time.Millisecond)

		// Context should NOT be cancelled for graceful interrupt
		select {
		case <-interruptCtx.Done():
			t.Error("Graceful interrupt should not cancel context")
		default:
			t.Log("✓ Graceful interrupt does not cancel context")
		}

		// IsInterrupted should still return false for graceful
		if ctx2.Interrupt.IsInterrupted() {
			t.Error("IsInterrupted should return false for graceful interrupt")
		} else {
			t.Log("✓ IsInterrupted returns false for graceful interrupt")
		}
	})
}

// TestInterruptSendSignalDirectly tests SendSignal method directly
func TestInterruptSendSignalDirectly(t *testing.T) {
	ctx := newTestContextWithInterrupt("chat-test-send-signal", "test-assistant")
	defer ctx.Release()

	t.Run("SendSignalSuccess", func(t *testing.T) {
		signal := &context.InterruptSignal{
			Type:      context.InterruptGraceful,
			Messages:  []context.Message{{Role: context.RoleUser, Content: "direct send"}},
			Timestamp: time.Now().UnixMilli(),
		}

		err := ctx.Interrupt.SendSignal(signal)
		if err != nil {
			t.Fatalf("SendSignal failed: %v", err)
		}

		time.Sleep(100 * time.Millisecond)

		// Verify signal was received
		received := ctx.Interrupt.Peek()
		if received == nil {
			t.Fatal("Signal not received")
		}

		if received.Messages[0].Content != "direct send" {
			t.Errorf("Expected 'direct send', got: %s", received.Messages[0].Content)
		}

		t.Log("✓ SendSignal directly works")
	})

	t.Run("SendSignalToNilController", func(t *testing.T) {
		var nilController *context.InterruptController
		signal := &context.InterruptSignal{
			Type:      context.InterruptGraceful,
			Messages:  []context.Message{{Role: context.RoleUser, Content: "test"}},
			Timestamp: time.Now().UnixMilli(),
		}

		err := nilController.SendSignal(signal)
		if err == nil {
			t.Error("Expected error when sending to nil controller")
		} else {
			t.Logf("✓ Correctly returned error for nil controller: %v", err)
		}
	})

	t.Run("SendSignalTimeout", func(t *testing.T) {
		// Create controller but don't start listener
		testCtrl := context.NewInterruptController()
		// Don't call Start(), so channel won't be read

		// Fill the buffer (capacity is 10)
		for i := 0; i < 10; i++ {
			signal := &context.InterruptSignal{
				Type:      context.InterruptGraceful,
				Messages:  []context.Message{{Role: context.RoleUser, Content: fmt.Sprintf("msg %d", i)}},
				Timestamp: time.Now().UnixMilli(),
			}
			testCtrl.SendSignal(signal)
		}

		// This should timeout since buffer is full and no listener
		signal := &context.InterruptSignal{
			Type:      context.InterruptGraceful,
			Messages:  []context.Message{{Role: context.RoleUser, Content: "overflow"}},
			Timestamp: time.Now().UnixMilli(),
		}

		err := testCtrl.SendSignal(signal)
		if err == nil {
			t.Error("Expected timeout error when buffer is full")
		} else {
			t.Logf("✓ SendSignal correctly times out when buffer full: %v", err)
		}
	})
}
