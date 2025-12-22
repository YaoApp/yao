package hook_test

import (
	stdContext "context"
	"runtime"
	"testing"
	"time"

	"github.com/yaoapp/yao/agent/assistant"
	"github.com/yaoapp/yao/agent/context"
	"github.com/yaoapp/yao/agent/testutils"
	"github.com/yaoapp/yao/openapi/oauth/types"
	"github.com/yaoapp/yao/test"
)

// ============================================================================
// Memory Leak Detection Tests
// ============================================================================

// TestMemoryLeakStandardMode checks for memory leaks in standard V8 mode
// Run with: go test -run=TestMemoryLeakStandardMode -v
func TestMemoryLeakStandardMode(t *testing.T) {
	testutils.Prepare(t)
	defer testutils.Clean(t)

	agent, err := assistant.Get("tests.create")
	if err != nil {
		t.Fatalf("Failed to get assistant: %s", err.Error())
	}

	if agent.HookScript == nil {
		t.Fatalf("Assistant has no script")
	}

	// Warm up - execute a few times to stabilize memory
	for i := 0; i < 10; i++ {
		ctx := newMemTestContext("warmup", "tests.create")
		_, _, _ = agent.HookScript.Create(ctx, []context.Message{
			{Role: "user", Content: "Hello"},
		})
		ctx.Release()
	}

	// Force GC and get baseline memory
	runtime.GC()
	time.Sleep(100 * time.Millisecond)
	var baseline runtime.MemStats
	runtime.ReadMemStats(&baseline)

	// Execute many iterations
	iterations := 1000
	for i := 0; i < iterations; i++ {
		ctx := newMemTestContext("mem-test-standard", "tests.create")
		_, _, err := agent.HookScript.Create(ctx, []context.Message{
			{Role: "user", Content: "Hello"},
		})
		if err != nil {
			t.Errorf("Create failed at iteration %d: %s", i, err.Error())
		}

		// Release context resources
		ctx.Release()

		// Periodic GC to help detect leaks faster
		if i%100 == 0 {
			runtime.GC()
		}
	}

	// Force GC and check final memory
	runtime.GC()
	time.Sleep(100 * time.Millisecond)
	var final runtime.MemStats
	runtime.ReadMemStats(&final)

	// Calculate memory growth
	baselineHeap := baseline.HeapAlloc
	finalHeap := final.HeapAlloc
	growth := int64(finalHeap) - int64(baselineHeap)
	growthPerIteration := float64(growth) / float64(iterations)

	t.Logf("Memory Statistics (Standard Mode):")
	t.Logf("  Iterations:              %d", iterations)
	t.Logf("  Baseline HeapAlloc:      %d bytes (%.2f MB)", baselineHeap, float64(baselineHeap)/1024/1024)
	t.Logf("  Final HeapAlloc:         %d bytes (%.2f MB)", finalHeap, float64(finalHeap)/1024/1024)
	t.Logf("  Total Growth:            %d bytes (%.2f MB)", growth, float64(growth)/1024/1024)
	t.Logf("  Growth per iteration:    %.2f bytes", growthPerIteration)
	t.Logf("  Total Alloc:             %d bytes (%.2f MB)", final.TotalAlloc, float64(final.TotalAlloc)/1024/1024)
	t.Logf("  Mallocs:                 %d", final.Mallocs)
	t.Logf("  Frees:                   %d", final.Frees)
	t.Logf("  Live Objects:            %d", final.Mallocs-final.Frees)
	t.Logf("  GC Runs:                 %d", final.NumGC-baseline.NumGC)

	// Check for memory leak
	// Standard mode creates/disposes isolates per request, so some overhead is expected
	// Allow up to 20KB growth per iteration as threshold
	// This accounts for V8 isolate creation/disposal overhead and bridge management
	// Significant leaks would show much higher growth rates (50KB+)
	maxGrowthPerIteration := 20480.0 // 20 KB
	if growthPerIteration > maxGrowthPerIteration {
		t.Errorf("Possible memory leak detected: %.2f bytes/iteration (threshold: %.2f bytes/iteration)",
			growthPerIteration, maxGrowthPerIteration)
	} else {
		t.Logf("✓ Memory growth is within acceptable range (%.2f bytes/iteration)", growthPerIteration)
	}
}

// TestMemoryLeakPerformanceMode checks for memory leaks in performance V8 mode
// Run with: go test -run=TestMemoryLeakPerformanceMode -v
func TestMemoryLeakPerformanceMode(t *testing.T) {
	testutils.Prepare(t, test.PrepareOption{V8Mode: "performance"})
	defer testutils.Clean(t)

	agent, err := assistant.Get("tests.create")
	if err != nil {
		t.Fatalf("Failed to get assistant: %s", err.Error())
	}

	if agent.HookScript == nil {
		t.Fatalf("Assistant has no script")
	}

	// Warm up - execute a few times to stabilize memory and fill isolate pool
	for i := 0; i < 20; i++ {
		ctx := newMemTestContext("warmup", "tests.create")
		_, _, _ = agent.HookScript.Create(ctx, []context.Message{
			{Role: "user", Content: "Hello"},
		})
		ctx.Release()
	}

	// Force GC and get baseline memory
	runtime.GC()
	time.Sleep(100 * time.Millisecond)
	var baseline runtime.MemStats
	runtime.ReadMemStats(&baseline)

	// Execute many iterations
	iterations := 1000
	for i := 0; i < iterations; i++ {
		ctx := newMemTestContext("mem-test-performance", "tests.create")
		_, _, err := agent.HookScript.Create(ctx, []context.Message{
			{Role: "user", Content: "Hello"},
		})
		if err != nil {
			t.Errorf("Create failed at iteration %d: %s", i, err.Error())
		}

		// Release context resources
		ctx.Release()

		// Periodic GC
		if i%100 == 0 {
			runtime.GC()
		}
	}

	// Force GC and check final memory
	runtime.GC()
	time.Sleep(100 * time.Millisecond)
	var final runtime.MemStats
	runtime.ReadMemStats(&final)

	// Calculate memory growth
	baselineHeap := baseline.HeapAlloc
	finalHeap := final.HeapAlloc
	growth := int64(finalHeap) - int64(baselineHeap)
	growthPerIteration := float64(growth) / float64(iterations)

	t.Logf("Memory Statistics (Performance Mode):")
	t.Logf("  Iterations:              %d", iterations)
	t.Logf("  Baseline HeapAlloc:      %d bytes (%.2f MB)", baselineHeap, float64(baselineHeap)/1024/1024)
	t.Logf("  Final HeapAlloc:         %d bytes (%.2f MB)", finalHeap, float64(finalHeap)/1024/1024)
	t.Logf("  Total Growth:            %d bytes (%.2f MB)", growth, float64(growth)/1024/1024)
	t.Logf("  Growth per iteration:    %.2f bytes", growthPerIteration)
	t.Logf("  Total Alloc:             %d bytes (%.2f MB)", final.TotalAlloc, float64(final.TotalAlloc)/1024/1024)
	t.Logf("  Mallocs:                 %d", final.Mallocs)
	t.Logf("  Frees:                   %d", final.Frees)
	t.Logf("  Live Objects:            %d", final.Mallocs-final.Frees)
	t.Logf("  GC Runs:                 %d", final.NumGC-baseline.NumGC)

	// Performance mode should have less growth due to isolate reuse
	// Allow up to 5KB per iteration as threshold
	maxGrowthPerIteration := 5120.0
	if growthPerIteration > maxGrowthPerIteration {
		t.Errorf("Possible memory leak detected: %.2f bytes/iteration (threshold: %.2f bytes/iteration)",
			growthPerIteration, maxGrowthPerIteration)
	} else {
		t.Logf("✓ Memory growth is within acceptable range")
	}
}

// TestMemoryLeakBusinessScenarios checks for memory leaks with business logic
// Run with: go test -run=TestMemoryLeakBusinessScenarios -v
func TestMemoryLeakBusinessScenarios(t *testing.T) {
	testutils.Prepare(t)
	defer testutils.Clean(t)

	agent, err := assistant.Get("tests.create")
	if err != nil {
		t.Fatalf("Failed to get assistant: %s", err.Error())
	}

	if agent.HookScript == nil {
		t.Fatalf("Assistant has no script")
	}

	scenarios := []struct {
		name    string
		content string
	}{
		{name: "FullResponse", content: "return_full"},
		{name: "PartialResponse", content: "return_partial"},
		{name: "ProcessCall", content: "return_process"},
		{name: "ContextAdjustment", content: "adjust_context"},
		{name: "NestedScriptCall", content: "nested_script_call"},
		{name: "DeepNestedCall", content: "deep_nested_call"},
	}

	// Warm up
	for i := 0; i < 10; i++ {
		ctx := newMemTestContext("warmup", "tests.create")
		_, _, _ = agent.HookScript.Create(ctx, []context.Message{
			{Role: "user", Content: "return_full"},
		})
		ctx.Release()
	}

	// Test each scenario
	for _, scenario := range scenarios {
		t.Run(scenario.name, func(t *testing.T) {
			// Get baseline
			runtime.GC()
			time.Sleep(50 * time.Millisecond)
			var baseline runtime.MemStats
			runtime.ReadMemStats(&baseline)

			// Execute iterations (reduced to avoid V8 OOM)
			iterations := 200
			for i := 0; i < iterations; i++ {
				ctx := newMemTestContext("mem-test-business", "tests.create")
				_, _, err := agent.HookScript.Create(ctx, []context.Message{
					{Role: "user", Content: scenario.content},
				})
				if err != nil {
					t.Errorf("Create failed at iteration %d: %s", i, err.Error())
				}
				ctx.Release()

				if i%50 == 0 {
					runtime.GC()
				}
			}

			// Check final memory
			runtime.GC()
			time.Sleep(50 * time.Millisecond)
			var final runtime.MemStats
			runtime.ReadMemStats(&final)

			growth := int64(final.HeapAlloc) - int64(baseline.HeapAlloc)
			growthPerIteration := float64(growth) / float64(iterations)

			t.Logf("  Baseline HeapAlloc: %d bytes (%.2f MB)", baseline.HeapAlloc, float64(baseline.HeapAlloc)/1024/1024)
			t.Logf("  Final HeapAlloc:    %d bytes (%.2f MB)", final.HeapAlloc, float64(final.HeapAlloc)/1024/1024)
			t.Logf("  Growth:             %d bytes (%.2f MB)", growth, float64(growth)/1024/1024)
			t.Logf("  Growth/iteration:   %.2f bytes", growthPerIteration)

			// Business scenarios may have more memory usage due to complex operations
			// Allow up to 20KB per iteration as threshold
			// Note: Some scenarios like ContextAdjustment generate dynamic timestamps,
			// causing slightly higher memory usage. Real leaks would show 50KB+ growth.
			maxGrowthPerIteration := 20480.0
			if growthPerIteration > maxGrowthPerIteration {
				t.Errorf("Possible memory leak: %.2f bytes/iteration (threshold: %.2f)",
					growthPerIteration, maxGrowthPerIteration)
			} else {
				t.Logf("  ✓ Memory growth is within acceptable range")
			}
		})
	}
}

// TestMemoryLeakConcurrent checks for memory leaks under concurrent load
// Run with: go test -run=TestMemoryLeakConcurrent -v
func TestMemoryLeakConcurrent(t *testing.T) {
	testutils.Prepare(t, test.PrepareOption{V8Mode: "performance"})
	defer testutils.Clean(t)

	agent, err := assistant.Get("tests.create")
	if err != nil {
		t.Fatalf("Failed to get assistant: %s", err.Error())
	}

	if agent.HookScript == nil {
		t.Fatalf("Assistant has no script")
	}

	// Warm up
	for i := 0; i < 20; i++ {
		ctx := newMemTestContext("warmup", "tests.create")
		_, _, _ = agent.HookScript.Create(ctx, []context.Message{
			{Role: "user", Content: "Hello"},
		})
		ctx.Release()
	}

	// Get baseline
	runtime.GC()
	time.Sleep(100 * time.Millisecond)
	var baseline runtime.MemStats
	runtime.ReadMemStats(&baseline)

	// Run concurrent load
	iterations := 1000
	concurrency := 10
	iterPerGoroutine := iterations / concurrency

	done := make(chan bool, concurrency)
	for g := 0; g < concurrency; g++ {
		go func(id int) {
			defer func() { done <- true }()
			for i := 0; i < iterPerGoroutine; i++ {
				ctx := newMemTestContext("mem-test-concurrent", "tests.create")
				_, _, err := agent.HookScript.Create(ctx, []context.Message{
					{Role: "user", Content: "Hello"},
				})
				if err != nil {
					t.Errorf("Goroutine %d failed at iteration %d: %s", id, i, err.Error())
				}
				ctx.Release()
			}
		}(g)
	}

	// Wait for all goroutines
	for g := 0; g < concurrency; g++ {
		<-done
	}

	// Check final memory
	runtime.GC()
	time.Sleep(100 * time.Millisecond)
	var final runtime.MemStats
	runtime.ReadMemStats(&final)

	growth := int64(final.HeapAlloc) - int64(baseline.HeapAlloc)
	growthPerIteration := float64(growth) / float64(iterations)

	t.Logf("Memory Statistics (Concurrent Load):")
	t.Logf("  Iterations:           %d", iterations)
	t.Logf("  Concurrency:          %d", concurrency)
	t.Logf("  Baseline HeapAlloc:   %d bytes (%.2f MB)", baseline.HeapAlloc, float64(baseline.HeapAlloc)/1024/1024)
	t.Logf("  Final HeapAlloc:      %d bytes (%.2f MB)", final.HeapAlloc, float64(final.HeapAlloc)/1024/1024)
	t.Logf("  Growth:               %d bytes (%.2f MB)", growth, float64(growth)/1024/1024)
	t.Logf("  Growth/iteration:     %.2f bytes", growthPerIteration)
	t.Logf("  GC Runs:              %d", final.NumGC-baseline.NumGC)

	// Concurrent scenarios may have slightly more overhead
	maxGrowthPerIteration := 10240.0
	if growthPerIteration > maxGrowthPerIteration {
		t.Errorf("Possible memory leak: %.2f bytes/iteration (threshold: %.2f)",
			growthPerIteration, maxGrowthPerIteration)
	} else {
		t.Logf("✓ Memory growth is within acceptable range")
	}
}

// TestMemoryLeakNestedCalls checks for memory leaks with nested script calls
// Run with: go test -run=TestMemoryLeakNestedCalls -v
func TestMemoryLeakNestedCalls(t *testing.T) {
	testutils.Prepare(t)
	defer testutils.Clean(t)

	agent, err := assistant.Get("tests.create")
	if err != nil {
		t.Fatalf("Failed to get assistant: %s", err.Error())
	}

	if agent.HookScript == nil {
		t.Fatalf("Assistant has no script")
	}

	// Warm up
	for i := 0; i < 10; i++ {
		ctx := newMemTestContext("warmup", "tests.create")
		_, _, _ = agent.HookScript.Create(ctx, []context.Message{
			{Role: "user", Content: "nested_script_call"},
		})
		ctx.Release()
	}

	// Get baseline
	runtime.GC()
	time.Sleep(100 * time.Millisecond)
	var baseline runtime.MemStats
	runtime.ReadMemStats(&baseline)

	// Execute iterations with nested calls
	// Nested calls: hook -> scripts.tests.create.NestedCall -> GetRoles/GetRole -> models
	iterations := 200
	for i := 0; i < iterations; i++ {
		ctx := newMemTestContext("mem-test-nested", "tests.create")
		_, _, err := agent.HookScript.Create(ctx, []context.Message{
			{Role: "user", Content: "deep_nested_call"},
		})
		if err != nil {
			t.Errorf("Nested call failed at iteration %d: %s", i, err.Error())
		}
		ctx.Release()

		if i%50 == 0 {
			runtime.GC()
		}
	}

	// Check final memory
	runtime.GC()
	time.Sleep(100 * time.Millisecond)
	var final runtime.MemStats
	runtime.ReadMemStats(&final)

	growth := int64(final.HeapAlloc) - int64(baseline.HeapAlloc)
	growthPerIteration := float64(growth) / float64(iterations)

	t.Logf("Memory Statistics (Nested Calls):")
	t.Logf("  Iterations:           %d", iterations)
	t.Logf("  Baseline HeapAlloc:   %d bytes (%.2f MB)", baseline.HeapAlloc, float64(baseline.HeapAlloc)/1024/1024)
	t.Logf("  Final HeapAlloc:      %d bytes (%.2f MB)", final.HeapAlloc, float64(final.HeapAlloc)/1024/1024)
	t.Logf("  Growth:               %d bytes (%.2f MB)", growth, float64(growth)/1024/1024)
	t.Logf("  Growth/iteration:     %.2f bytes", growthPerIteration)
	t.Logf("  GC Runs:              %d", final.NumGC-baseline.NumGC)

	// Nested calls involve database operations, so allow more overhead
	// Allow up to 20KB per iteration as threshold
	maxGrowthPerIteration := 20480.0
	if growthPerIteration > maxGrowthPerIteration {
		t.Errorf("Possible memory leak: %.2f bytes/iteration (threshold: %.2f)",
			growthPerIteration, maxGrowthPerIteration)
	} else {
		t.Logf("✓ Memory growth is within acceptable range")
	}
}

// TestMemoryLeakNestedConcurrent checks for memory leaks with concurrent nested calls
// Run with: go test -run=TestMemoryLeakNestedConcurrent -v
func TestMemoryLeakNestedConcurrent(t *testing.T) {
	testutils.Prepare(t, test.PrepareOption{V8Mode: "performance"})
	defer testutils.Clean(t)

	agent, err := assistant.Get("tests.create")
	if err != nil {
		t.Fatalf("Failed to get assistant: %s", err.Error())
	}

	if agent.HookScript == nil {
		t.Fatalf("Assistant has no script")
	}

	// Warm up
	for i := 0; i < 20; i++ {
		ctx := newMemTestContext("warmup", "tests.create")
		_, _, _ = agent.HookScript.Create(ctx, []context.Message{
			{Role: "user", Content: "nested_script_call"},
		})
		ctx.Release()
	}

	// Get baseline
	runtime.GC()
	time.Sleep(100 * time.Millisecond)
	var baseline runtime.MemStats
	runtime.ReadMemStats(&baseline)

	// Run concurrent nested calls
	iterations := 500
	concurrency := 10
	iterPerGoroutine := iterations / concurrency

	done := make(chan bool, concurrency)
	for g := 0; g < concurrency; g++ {
		go func(id int) {
			defer func() { done <- true }()
			for i := 0; i < iterPerGoroutine; i++ {
				ctx := newMemTestContext("mem-test-nested-concurrent", "tests.create")
				_, _, err := agent.HookScript.Create(ctx, []context.Message{
					{Role: "user", Content: "deep_nested_call"},
				})
				if err != nil {
					t.Errorf("Goroutine %d nested call failed at iteration %d: %s", id, i, err.Error())
				}
				ctx.Release()
			}
		}(g)
	}

	// Wait for all goroutines
	for g := 0; g < concurrency; g++ {
		<-done
	}

	// Check final memory
	runtime.GC()
	time.Sleep(100 * time.Millisecond)
	var final runtime.MemStats
	runtime.ReadMemStats(&final)

	growth := int64(final.HeapAlloc) - int64(baseline.HeapAlloc)
	growthPerIteration := float64(growth) / float64(iterations)

	t.Logf("Memory Statistics (Concurrent Nested Calls):")
	t.Logf("  Iterations:           %d", iterations)
	t.Logf("  Concurrency:          %d", concurrency)
	t.Logf("  Baseline HeapAlloc:   %d bytes (%.2f MB)", baseline.HeapAlloc, float64(baseline.HeapAlloc)/1024/1024)
	t.Logf("  Final HeapAlloc:      %d bytes (%.2f MB)", final.HeapAlloc, float64(final.HeapAlloc)/1024/1024)
	t.Logf("  Growth:               %d bytes (%.2f MB)", growth, float64(growth)/1024/1024)
	t.Logf("  Growth/iteration:     %.2f bytes", growthPerIteration)
	t.Logf("  GC Runs:              %d", final.NumGC-baseline.NumGC)

	// Concurrent nested calls with database operations
	// Allow up to 25KB per iteration as threshold
	maxGrowthPerIteration := 25600.0
	if growthPerIteration > maxGrowthPerIteration {
		t.Errorf("Possible memory leak: %.2f bytes/iteration (threshold: %.2f)",
			growthPerIteration, maxGrowthPerIteration)
	} else {
		t.Logf("✓ Memory growth is within acceptable range")
	}
}

// TestIsolateDisposal verifies that isolates are properly disposed in standard mode
// Run with: go test -run=TestIsolateDisposal -v
func TestIsolateDisposal(t *testing.T) {
	testutils.Prepare(t)
	defer testutils.Clean(t)

	agent, err := assistant.Get("tests.create")
	if err != nil {
		t.Fatalf("Failed to get assistant: %s", err.Error())
	}

	if agent.HookScript == nil {
		t.Fatalf("Assistant has no script")
	}

	// Track goroutine count to detect goroutine leaks
	initialGoroutines := runtime.NumGoroutine()

	// Execute multiple iterations
	iterations := 100
	for i := 0; i < iterations; i++ {
		ctx := newMemTestContext("disposal-test", "tests.create")
		_, _, err := agent.HookScript.Create(ctx, []context.Message{
			{Role: "user", Content: "Hello"},
		})
		if err != nil {
			t.Errorf("Create failed at iteration %d: %s", i, err.Error())
		}
		ctx.Release()
	}

	// Give time for cleanup
	time.Sleep(200 * time.Millisecond)
	runtime.GC()
	time.Sleep(200 * time.Millisecond)

	finalGoroutines := runtime.NumGoroutine()
	goroutineGrowth := finalGoroutines - initialGoroutines

	t.Logf("Goroutine Statistics:")
	t.Logf("  Initial:  %d", initialGoroutines)
	t.Logf("  Final:    %d", finalGoroutines)
	t.Logf("  Growth:   %d", goroutineGrowth)

	// Allow some goroutine growth for runtime internals
	//
	// ROOT CAUSE ANALYSIS:
	// Each Create() call creates a Trace, which starts 2 goroutines:
	// 1. trace/pubsub.(*PubSub).forward() - PubSub event forwarding
	// 2. trace.(*manager).startStateWorker() - State machine worker
	//
	// These goroutines exit when Release() closes their channels, but:
	// - Exit is ASYNCHRONOUS (goroutine needs to reach select statement)
	// - Go runtime needs time to schedule and cleanup
	// - In rapid iterations, new goroutines are created before old ones fully exit
	//
	// This is NOT a true leak:
	// ✓ Goroutines eventually exit (channels are closed)
	// ✓ No unbounded growth (they will be GC'd)
	// ✓ Typical pattern for async cleanup in Go
	//
	// Acceptable: ~2 goroutines per iteration (trace pubsub + state worker)
	// Concerning: >5 goroutines per iteration (indicates goroutines NOT exiting)
	maxGoroutineGrowthPerIteration := 5.0
	growthPerIteration := float64(goroutineGrowth) / float64(iterations)

	if growthPerIteration > maxGoroutineGrowthPerIteration {
		t.Errorf("Goroutine leak detected: %.2f goroutines per iteration (threshold: %.2f)",
			growthPerIteration, maxGoroutineGrowthPerIteration)
		t.Errorf("This indicates goroutines are NOT being cleaned up properly")
	} else {
		t.Logf("✓ Goroutine growth is acceptable: %.2f per iteration", growthPerIteration)
		t.Logf("  (Trace creates 2 goroutines per call: pubsub.forward + stateWorker)")
		t.Logf("  (These exit asynchronously after Release(), causing temporary accumulation)")
	}
}

// ============================================================================
// Helper Functions
// ============================================================================

// newMemTestContext creates a context for memory leak testing
func newMemTestContext(chatID, assistantID string) *context.Context {
	authorized := &types.AuthorizedInfo{
		Subject:  "mem-test-user",
		ClientID: "mem-test-client",
		UserID:   "mem-user-123",
		TeamID:   "mem-team-456",
		TenantID: "mem-tenant-789",
		Constraints: types.DataConstraints{
			TeamOnly: true,
			Extra: map[string]interface{}{
				"department": "engineering",
			},
		},
	}

	ctx := context.New(stdContext.Background(), authorized, chatID)
	ctx.AssistantID = assistantID
	ctx.Locale = "en-us"
	ctx.Theme = "light"
	ctx.Client = context.Client{
		Type:      "web",
		UserAgent: "MemTestAgent/1.0",
		IP:        "127.0.0.1",
	}
	ctx.Referer = context.RefererAPI
	ctx.Accept = context.AcceptWebCUI
	ctx.Route = ""
	ctx.Metadata = make(map[string]interface{})
	return ctx
}
