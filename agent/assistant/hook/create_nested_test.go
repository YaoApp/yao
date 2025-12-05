package hook_test

import (
	"sync"
	"testing"

	"github.com/yaoapp/yao/agent/assistant"
	"github.com/yaoapp/yao/agent/context"
	"github.com/yaoapp/yao/agent/testutils"
)

// TestNestedScriptCall tests nested script calls with V8 context sharing
// This test calls: hook -> scripts.tests.create.NestedCall -> GetRoles/GetRole -> models
func TestNestedScriptCall(t *testing.T) {
	testutils.Prepare(t)
	defer testutils.Clean(t)

	agent, err := assistant.Get("tests.create")
	if err != nil {
		t.Fatalf("Failed to get assistant: %s", err.Error())
	}

	if agent.HookScript == nil {
		t.Fatalf("Assistant has no script")
	}

	// Create context
	ctx := newTestContext("test-nested-call", "tests.create")

	// Call with deep_nested_call scenario
	// This will: hook -> scripts.tests.create.NestedCall -> GetRoles -> model
	res, _, err := agent.HookScript.Create(ctx, []context.Message{
		{Role: "user", Content: "deep_nested_call"},
	})

	if err != nil {
		t.Fatalf("Nested call failed: %s", err.Error())
	}

	if res == nil {
		t.Fatal("Expected non-nil response")
	}

	// Verify messages
	if len(res.Messages) == 0 {
		t.Fatal("Expected messages in response")
	}

	t.Logf("✓ Nested script call completed successfully")
	t.Logf("  Messages count: %d", len(res.Messages))
	if res.Metadata != nil {
		t.Logf("  Metadata: %+v", res.Metadata)
	}
}

// TestNestedScriptCallConcurrent tests nested script calls under high concurrency
// Simulates 100 concurrent users making nested script calls
func TestNestedScriptCallConcurrent(t *testing.T) {
	testutils.Prepare(t)
	defer testutils.Clean(t)

	agent, err := assistant.Get("tests.create")
	if err != nil {
		t.Fatalf("Failed to get assistant: %s", err.Error())
	}

	if agent.HookScript == nil {
		t.Fatalf("Assistant has no script")
	}

	// High concurrency test: 100 concurrent users (testing race condition)
	concurrency := 100
	iterations := 1 // Each user makes 1 call

	var wg sync.WaitGroup
	errors := make(chan error, concurrency*iterations)

	t.Logf("Starting concurrent test: %d users × %d iterations = %d total calls",
		concurrency, iterations, concurrency*iterations)

	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func(userID int) {
			defer wg.Done()

			for j := 0; j < iterations; j++ {
				ctx := newTestContext("test-concurrent", "tests.create")

				_, _, err := agent.HookScript.Create(ctx, []context.Message{
					{Role: "user", Content: "deep_nested_call"},
				})

				if err != nil {
					errors <- err
					return
				}
			}
		}(i)
	}

	// Wait for all goroutines to complete
	wg.Wait()
	close(errors)

	// Check for errors
	errorCount := 0
	for err := range errors {
		errorCount++
		t.Errorf("Concurrent call failed: %s", err.Error())
	}

	if errorCount > 0 {
		t.Fatalf("Failed with %d errors out of %d total calls", errorCount, concurrency*iterations)
	}

	t.Logf("✓ All %d concurrent nested calls completed successfully", concurrency*iterations)
}
