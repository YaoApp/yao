package assistant_test

import (
	stdContext "context"
	"fmt"
	"testing"
	"time"

	"github.com/yaoapp/yao/agent/assistant"
	"github.com/yaoapp/yao/agent/context"
	"github.com/yaoapp/yao/agent/output/message"
	"github.com/yaoapp/yao/agent/testutils"
	"github.com/yaoapp/yao/openapi/oauth/types"
)

// newTestContextWithInterrupt creates a Context with interrupt controller for testing
// Returns the context and a cancel function that should be called before Release()
func newTestContextWithInterrupt(chatID, assistantID string) (*context.Context, stdContext.CancelFunc) {
	authorized := &types.AuthorizedInfo{
		Subject:   "test-user",
		ClientID:  "test-client-id",
		UserID:    "test-user-123",
		TeamID:    "test-team-456",
		TenantID:  "test-tenant-789",
		SessionID: "test-session-id",
	}

	// Use cancellable context to properly stop goroutines on timeout
	parentCtx, cancel := stdContext.WithCancel(stdContext.Background())

	ctx := context.New(parentCtx, authorized, chatID)
	ctx.ID = fmt.Sprintf("test_ctx_%d", time.Now().UnixNano())
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
	ctx.IDGenerator = message.NewIDGenerator() // Initialize context-scoped ID generator
	ctx.Metadata = map[string]interface{}{
		"test": "interrupt_test",
	}

	// Initialize interrupt controller
	ctx.Interrupt = context.NewInterruptController()

	// Register context globally
	if err := context.Register(ctx); err != nil {
		panic(fmt.Sprintf("Failed to register context: %v", err))
	}

	// Start interrupt listener
	ctx.Interrupt.Start(ctx.ID)

	return ctx, cancel
}

// TestAgentInterruptGraceful tests graceful interrupt during agent stream
func TestAgentInterruptGraceful(t *testing.T) {
	testutils.Prepare(t)
	defer testutils.Clean(t)

	agent, err := assistant.Get("tests.interrupt")
	if err != nil {
		t.Skipf("Skipping test: assistant 'tests.interrupt' not found: %v", err)
		return
	}

	t.Run("GracefulInterruptDuringStream", func(t *testing.T) {
		// Create context with interrupt support
		ctx, cancel := newTestContextWithInterrupt("chat-interrupt-graceful", "tests.interrupt")
		defer func() {
			cancel()                           // Cancel context first to stop goroutines
			time.Sleep(100 * time.Millisecond) // Wait for goroutines to exit
			ctx.Release()
		}()

		// Track handler invocations
		handlerInvoked := false
		var receivedSignal *context.InterruptSignal

		// Override the handler to track invocations
		originalHandler := ctx.Interrupt
		ctx.Interrupt.SetHandler(func(c *context.Context, signal *context.InterruptSignal) error {
			handlerInvoked = true
			receivedSignal = signal
			t.Logf("✓ Interrupt handler invoked: type=%s, messages=%d", signal.Type, len(signal.Messages))
			return nil
		})

		inputMessages := []context.Message{
			{Role: context.RoleUser, Content: "Tell me a long story about artificial intelligence"},
		}

		// Start streaming in a goroutine
		streamDone := make(chan error, 1)
		go func() {
			_, err := agent.Stream(ctx, inputMessages)
			streamDone <- err
		}()

		// Wait a bit to ensure stream has started
		time.Sleep(300 * time.Millisecond)

		// Send graceful interrupt signal
		signal := &context.InterruptSignal{
			Type: context.InterruptGraceful,
			Messages: []context.Message{
				{Role: context.RoleUser, Content: "Actually, can you make it shorter?"},
			},
			Timestamp: time.Now().UnixMilli(),
		}

		err = context.SendInterrupt(ctx.ID, signal)
		if err != nil {
			t.Logf("Warning: Failed to send interrupt (stream may have completed): %v", err)
		} else {
			t.Log("✓ Graceful interrupt signal sent")
		}

		// Wait for stream to complete (with timeout)
		select {
		case err := <-streamDone:
			if err != nil {
				t.Logf("Stream completed with error: %v", err)
			} else {
				t.Log("✓ Stream completed successfully")
			}
		case <-time.After(10 * time.Second):
			t.Log("Stream timeout (expected for real LLM calls)")
			cancel()     // Cancel to stop the stream goroutine
			<-streamDone // Wait for goroutine to exit
		}

		// Verify handler was invoked if signal was sent
		if originalHandler != nil {
			time.Sleep(200 * time.Millisecond) // Wait for async handler
			if handlerInvoked {
				t.Log("✓ Interrupt handler was invoked")
				if receivedSignal != nil && receivedSignal.Type == context.InterruptGraceful {
					t.Log("✓ Received graceful interrupt signal")
				}
			}
		}
	})
}

// TestAgentInterruptForce tests force interrupt during agent stream
func TestAgentInterruptForce(t *testing.T) {
	testutils.Prepare(t)
	defer testutils.Clean(t)

	agent, err := assistant.Get("tests.interrupt")
	if err != nil {
		t.Skipf("Skipping test: assistant 'tests.interrupt' not found: %v", err)
		return
	}

	t.Run("ForceInterruptDuringStream", func(t *testing.T) {
		// Create context with interrupt support
		ctx, cancel := newTestContextWithInterrupt("chat-interrupt-force", "tests.interrupt")
		defer func() {
			cancel()                           // Cancel context first to stop goroutines
			time.Sleep(100 * time.Millisecond) // Wait for goroutines to exit
			ctx.Release()
		}()

		// Track handler invocations
		handlerInvoked := false
		streamInterrupted := false

		ctx.Interrupt.SetHandler(func(c *context.Context, signal *context.InterruptSignal) error {
			handlerInvoked = true
			t.Logf("✓ Interrupt handler invoked: type=%s", signal.Type)
			return nil
		})

		inputMessages := []context.Message{
			{Role: context.RoleUser, Content: "Write a very detailed essay about machine learning"},
		}

		// Start streaming in a goroutine
		streamDone := make(chan error, 1)
		go func() {
			_, err := agent.Stream(ctx, inputMessages)
			streamDone <- err
		}()

		// Wait a bit to ensure stream has started
		time.Sleep(300 * time.Millisecond)

		// Send force interrupt signal
		signal := &context.InterruptSignal{
			Type: context.InterruptForce,
			Messages: []context.Message{
				{Role: context.RoleUser, Content: "Stop! I need something else now."},
			},
			Timestamp: time.Now().UnixMilli(),
		}

		err = context.SendInterrupt(ctx.ID, signal)
		if err != nil {
			t.Logf("Warning: Failed to send interrupt: %v", err)
		} else {
			t.Log("✓ Force interrupt signal sent")
		}

		// Wait for stream to complete or be interrupted
		select {
		case err := <-streamDone:
			if err != nil {
				// Check if error is due to interrupt
				if err.Error() == "force interrupted by user" ||
					err.Error() == "interrupted by user" ||
					err.Error() == "interrupted by user before stream start" {
					streamInterrupted = true
					t.Logf("✓ Stream was interrupted: %v", err)
				} else {
					t.Logf("Stream completed with error: %v", err)
				}
			} else {
				t.Log("Stream completed without error")
			}
		case <-time.After(10 * time.Second):
			t.Log("Stream timeout")
			cancel()     // Cancel to stop the stream goroutine
			<-streamDone // Wait for goroutine to exit
		}

		// Verify interrupt behavior
		time.Sleep(200 * time.Millisecond)
		if handlerInvoked {
			t.Log("✓ Force interrupt handler was invoked")
		}
		if streamInterrupted {
			t.Log("✓ Stream was interrupted by force signal")
		}
	})
}

// TestAgentMultipleInterrupts tests multiple interrupts during stream
func TestAgentMultipleInterrupts(t *testing.T) {
	testutils.Prepare(t)
	defer testutils.Clean(t)

	agent, err := assistant.Get("tests.interrupt")
	if err != nil {
		t.Skipf("Skipping test: assistant 'tests.interrupt' not found: %v", err)
		return
	}

	t.Run("MultipleGracefulInterrupts", func(t *testing.T) {
		// Create context with interrupt support
		ctx, cancel := newTestContextWithInterrupt("chat-interrupt-multiple", "tests.interrupt")
		defer func() {
			cancel()                           // Cancel context first to stop goroutines
			time.Sleep(100 * time.Millisecond) // Wait for goroutines to exit
			ctx.Release()
		}()

		handlerCallCount := 0
		ctx.Interrupt.SetHandler(func(c *context.Context, signal *context.InterruptSignal) error {
			handlerCallCount++
			t.Logf("✓ Interrupt handler invoked (call %d): %d messages", handlerCallCount, len(signal.Messages))
			return nil
		})

		inputMessages := []context.Message{
			{Role: context.RoleUser, Content: "Explain quantum computing in detail"},
		}

		// Start streaming
		streamDone := make(chan error, 1)
		go func() {
			_, err := agent.Stream(ctx, inputMessages)
			streamDone <- err
		}()

		// Wait for stream to start
		time.Sleep(300 * time.Millisecond)

		// Send multiple graceful interrupts
		for i := 1; i <= 3; i++ {
			signal := &context.InterruptSignal{
				Type: context.InterruptGraceful,
				Messages: []context.Message{
					{Role: context.RoleUser, Content: fmt.Sprintf("Additional question %d", i)},
				},
				Timestamp: time.Now().UnixMilli(),
			}

			err = context.SendInterrupt(ctx.ID, signal)
			if err != nil {
				t.Logf("Warning: Failed to send interrupt %d: %v", i, err)
			} else {
				t.Logf("✓ Sent interrupt %d", i)
			}

			time.Sleep(100 * time.Millisecond)
		}

		// Wait for stream to complete
		select {
		case err := <-streamDone:
			if err != nil {
				t.Logf("Stream completed with error: %v", err)
			}
		case <-time.After(10 * time.Second):
			t.Log("Stream timeout")
			cancel()     // Cancel to stop the stream goroutine
			<-streamDone // Wait for goroutine to exit
		}

		// Check if interrupts were received
		time.Sleep(300 * time.Millisecond)
		pendingCount := ctx.Interrupt.GetPendingCount()
		t.Logf("Handler was called %d times, pending count: %d", handlerCallCount, pendingCount)

		if handlerCallCount > 0 {
			t.Log("✓ Multiple interrupts were processed")
		}
	})
}

// TestAgentInterruptWithoutStream tests interrupt behavior when no stream is active
func TestAgentInterruptWithoutStream(t *testing.T) {
	t.Run("InterruptBeforeStream", func(t *testing.T) {
		// Create context with interrupt support
		ctx, cancel := newTestContextWithInterrupt("chat-interrupt-before", "test-assistant")
		defer func() {
			cancel()
			ctx.Release()
		}()

		// Send interrupt before starting stream
		signal := &context.InterruptSignal{
			Type: context.InterruptGraceful,
			Messages: []context.Message{
				{Role: context.RoleUser, Content: "Early interrupt"},
			},
			Timestamp: time.Now().UnixMilli(),
		}

		err := context.SendInterrupt(ctx.ID, signal)
		if err != nil {
			t.Fatalf("Failed to send interrupt: %v", err)
		}

		// Wait for signal to be processed
		time.Sleep(100 * time.Millisecond)

		// Check if signal is in queue
		receivedSignal := ctx.Interrupt.Peek()
		if receivedSignal == nil {
			t.Fatal("Expected interrupt signal to be queued")
		}

		if receivedSignal.Type != context.InterruptGraceful {
			t.Errorf("Expected graceful interrupt, got: %s", receivedSignal.Type)
		}

		t.Log("✓ Interrupt queued before stream starts")
	})
}

// TestAgentInterruptContextCleanup tests cleanup after interrupt
func TestAgentInterruptContextCleanup(t *testing.T) {
	t.Run("CleanupAfterInterrupt", func(t *testing.T) {
		ctx, cancel := newTestContextWithInterrupt("chat-interrupt-cleanup", "test-assistant")

		// Send interrupt
		signal := &context.InterruptSignal{
			Type:      context.InterruptGraceful,
			Messages:  []context.Message{{Role: context.RoleUser, Content: "test"}},
			Timestamp: time.Now().UnixMilli(),
		}
		context.SendInterrupt(ctx.ID, signal)

		time.Sleep(100 * time.Millisecond)

		// Cancel and release context
		cancel()
		ctx.Release()

		// Try to send interrupt to released context
		err := context.SendInterrupt(ctx.ID, signal)
		if err == nil {
			t.Error("Expected error when sending to released context")
		} else {
			t.Logf("✓ Correctly rejected interrupt to released context: %v", err)
		}
	})
}
