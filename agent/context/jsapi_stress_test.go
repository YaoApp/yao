package context_test

import (
	stdContext "context"
	"fmt"
	"os"
	"runtime"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	v8 "github.com/yaoapp/gou/runtime/v8"
	"github.com/yaoapp/yao/agent/context"
	"github.com/yaoapp/yao/config"
	"github.com/yaoapp/yao/test"
)

// newStressTestContext creates a test context for stress testing
func newStressTestContext(chatID string) *context.Context {
	ctx := context.New(stdContext.Background(), nil, chatID)
	ctx.AssistantID = "test-assistant"
	ctx.Referer = context.RefererAPI
	stack, _, _ := context.EnterStack(ctx, "test-assistant", &context.Options{})
	ctx.Stack = stack
	return ctx
}

// TestStressContextCreationAndRelease tests massive context creation and cleanup
func TestStressContextCreationAndRelease(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping stress test in short mode")
	}

	test.Prepare(t, config.Conf)
	defer test.Clean()

	iterations := 1000
	startMemory := getMemStats()

	for i := 0; i < iterations; i++ {
		cxt := newStressTestContext(fmt.Sprintf("chat-%d", i))

		_, err := v8.Call(v8.CallOptions{}, `
			function test(ctx) {
				// Use trace
				ctx.trace.Add({ type: "test" }, { label: "Test" })
				ctx.trace.Info("Processing")
				
				// Explicit release
				ctx.Release()
				
				return { iteration: true }
			}`, cxt)

		if err != nil {
			t.Fatalf("Iteration %d failed: %v", i, err)
		}

		// Force GC every 100 iterations to check for leaks
		if i%100 == 0 {
			runtime.GC()
			currentMemory := getMemStats()
			t.Logf("Iteration %d: Memory usage: %d MB", i, currentMemory/1024/1024)
		}
	}

	// Final GC and memory check
	runtime.GC()
	time.Sleep(100 * time.Millisecond)
	endMemory := getMemStats()

	t.Logf("Start memory: %d MB", startMemory/1024/1024)
	t.Logf("End memory: %d MB", endMemory/1024/1024)

	// Calculate memory growth (handle case where end < start)
	var memoryGrowth int64
	if endMemory > startMemory {
		memoryGrowth = int64(endMemory - startMemory)
		t.Logf("Memory growth: %d MB", memoryGrowth/1024/1024)
	} else {
		memoryGrowth = -int64(startMemory - endMemory)
		t.Logf("Memory decreased: %d MB", -memoryGrowth/1024/1024)
	}

	// Allow reasonable memory growth (not more than 50MB for 1000 iterations)
	// Memory can decrease due to GC, which is fine
	if memoryGrowth > 0 {
		assert.Less(t, memoryGrowth, int64(50*1024*1024), "Memory leak detected")
	}
}

// TestStressTraceOperations tests intensive trace operations
func TestStressTraceOperations(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping stress test in short mode")
	}

	test.Prepare(t, config.Conf)
	defer test.Clean()

	iterations := 500
	nodesPerIteration := 10

	startMemory := getMemStats()

	for i := 0; i < iterations; i++ {
		// Create new context for each iteration to avoid context cancellation issues
		cxt := newStressTestContext(fmt.Sprintf("stress-test-chat-%d", i))
		_, err := v8.Call(v8.CallOptions{}, fmt.Sprintf(`
			function test(ctx) {
				const trace = ctx.trace
				const nodes = []
				
				// Create multiple nodes
				for (let j = 0; j < %d; j++) {
					const node = trace.Add(
						{ type: "step", data: "data-" + j },
						{ label: "Step " + j }
					)
					nodes.push(node)
					
					// Add logs
					node.Info("Processing step " + j)
					node.Debug("Debug info " + j)
				}
				
				// Complete all nodes
				for (const node of nodes) {
					node.Complete({ result: "success" })
				}
				
				// Release resources
				ctx.Release()
				
				return { nodes: nodes.length }
			}`, nodesPerIteration), cxt)

		if err != nil {
			t.Fatalf("Iteration %d failed: %v", i, err)
		}

		if i%50 == 0 {
			runtime.GC()
			currentMemory := getMemStats()
			t.Logf("Iteration %d: Created %d nodes, Memory: %d MB",
				i, i*nodesPerIteration, currentMemory/1024/1024)
		}
	}

	runtime.GC()
	time.Sleep(100 * time.Millisecond)
	endMemory := getMemStats()

	t.Logf("Total nodes created: %d", iterations*nodesPerIteration)
	t.Logf("Start memory: %d MB", startMemory/1024/1024)
	t.Logf("End memory: %d MB", endMemory/1024/1024)

	if endMemory > startMemory {
		t.Logf("Memory growth: %d MB", (endMemory-startMemory)/1024/1024)
	} else {
		t.Logf("Memory decreased: %d MB", (startMemory-endMemory)/1024/1024)
	}
}

// TestStressMCPOperations tests intensive MCP operations
func TestStressMCPOperations(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping stress test in short mode")
	}

	test.Prepare(t, config.Conf)
	defer test.Clean()

	iterations := 500

	cxt := newStressTestContext("mcp-stress-test")

	startMemory := getMemStats()

	for i := 0; i < iterations; i++ {
		_, err := v8.Call(v8.CallOptions{}, `
			function test(ctx) {
				// List operations
				const tools = ctx.mcp.ListTools("echo", "")
				const resources = ctx.mcp.ListResources("echo", "")
				const prompts = ctx.mcp.ListPrompts("echo", "")
				
				// Call operations
				const result1 = ctx.mcp.CallTool("echo", "ping", { count: 1 })
				const result2 = ctx.mcp.CallTool("echo", "status", { verbose: false })
				
				// Read operations
				const info = ctx.mcp.ReadResource("echo", "echo://info")
				
				return {
					tools: tools.tools.length,
					resources: resources.resources.length,
					prompts: prompts.prompts.length
				}
			}`, cxt)

		if err != nil {
			t.Fatalf("Iteration %d failed: %v", i, err)
		}

		if i%50 == 0 {
			runtime.GC()
			currentMemory := getMemStats()
			t.Logf("Iteration %d: Memory: %d MB", i, currentMemory/1024/1024)
		}
	}

	runtime.GC()
	time.Sleep(100 * time.Millisecond)
	endMemory := getMemStats()

	t.Logf("MCP operations: %d iterations", iterations)
	t.Logf("Start memory: %d MB", startMemory/1024/1024)
	t.Logf("End memory: %d MB", endMemory/1024/1024)

	if endMemory > startMemory {
		t.Logf("Memory growth: %d MB", (endMemory-startMemory)/1024/1024)
	} else {
		t.Logf("Memory decreased: %d MB", (startMemory-endMemory)/1024/1024)
	}
}

// TestStressConcurrentContexts tests concurrent context creation and usage
func TestStressConcurrentContexts(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping stress test in short mode")
	}

	test.Prepare(t, config.Conf)
	defer test.Clean()

	goroutines := 50
	iterationsPerGoroutine := 20

	startMemory := getMemStats()

	var wg sync.WaitGroup
	errors := make(chan error, goroutines*iterationsPerGoroutine)

	for g := 0; g < goroutines; g++ {
		wg.Add(1)
		go func(goroutineID int) {
			defer wg.Done()

			for i := 0; i < iterationsPerGoroutine; i++ {
				cxt := newStressTestContext(fmt.Sprintf("chat-%d-%d", goroutineID, i))

				_, err := v8.Call(v8.CallOptions{}, `
					function test(ctx) {
						// Use trace
						const node = ctx.trace.Add({ type: "test" }, { label: "Concurrent Test" })
						ctx.trace.Info("Processing concurrent request")
						node.Complete({ result: "success" })
						
						// Use MCP
						const tools = ctx.mcp.ListTools("echo", "")
						
						// Release resources
						ctx.Release()
						
						return { success: true }
					}`, cxt)

				if err != nil {
					errors <- fmt.Errorf("goroutine %d iteration %d: %v", goroutineID, i, err)
					return
				}
			}
		}(g)
	}

	wg.Wait()
	close(errors)

	// Check for errors
	errorCount := 0
	for err := range errors {
		t.Error(err)
		errorCount++
	}

	assert.Equal(t, 0, errorCount, "No errors should occur in concurrent operations")

	runtime.GC()
	time.Sleep(100 * time.Millisecond)
	endMemory := getMemStats()

	totalOperations := goroutines * iterationsPerGoroutine
	t.Logf("Total operations: %d (goroutines: %d, iterations: %d)",
		totalOperations, goroutines, iterationsPerGoroutine)
	t.Logf("Start memory: %d MB", startMemory/1024/1024)
	t.Logf("End memory: %d MB", endMemory/1024/1024)

	if endMemory > startMemory {
		t.Logf("Memory growth: %d MB", (endMemory-startMemory)/1024/1024)
	} else {
		t.Logf("Memory decreased: %d MB", (startMemory-endMemory)/1024/1024)
	}
}

// TestStressNoOpTracePerformance tests no-op trace performance
func TestStressNoOpTracePerformance(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping stress test in short mode")
	}

	test.Prepare(t, config.Conf)
	defer test.Clean()

	iterations := 1000

	// Context without trace initialization (no-op trace)
	cxt := context.New(stdContext.Background(), nil, "noop-stress-test")
	cxt.AssistantID = "test-assistant"

	startMemory := getMemStats()
	startTime := time.Now()

	for i := 0; i < iterations; i++ {
		_, err := v8.Call(v8.CallOptions{}, `
			function test(ctx) {
				const trace = ctx.trace // no-op trace
				
				// All operations should be no-ops and fast
				trace.Info("No-op info")
				const node = trace.Add({ type: "test" }, { label: "No-op" })
				node.Info("No-op node info")
				node.Complete({ result: "done" })
				trace.Release()
				
				return { noop: true }
			}`, cxt)

		if err != nil {
			t.Fatalf("Iteration %d failed: %v", i, err)
		}
	}

	duration := time.Since(startTime)
	runtime.GC()
	endMemory := getMemStats()

	avgTimePerOp := duration / time.Duration(iterations)
	t.Logf("No-op trace operations: %d iterations", iterations)
	t.Logf("Total time: %v", duration)
	t.Logf("Average time per operation: %v", avgTimePerOp)
	t.Logf("Start memory: %d MB", startMemory/1024/1024)
	t.Logf("End memory: %d MB", endMemory/1024/1024)

	// No-op operations should be reasonably fast
	// Note: CI environments may be slower due to resource limits
	// Local: ~2ms, CI: ~10ms
	maxTimePerOp := 5 * time.Millisecond
	if os.Getenv("CI") != "" || os.Getenv("GITHUB_ACTIONS") != "" {
		maxTimePerOp = 15 * time.Millisecond // More lenient for CI
	}
	assert.Less(t, avgTimePerOp, maxTimePerOp, "No-op operations should be fast")

	// No-op operations should not leak memory (< 5MB growth)
	if endMemory > startMemory {
		memoryGrowth := int64(endMemory - startMemory)
		assert.Less(t, memoryGrowth, int64(5*1024*1024), "No-op operations should not leak memory")
		t.Logf("Memory growth: %d MB", memoryGrowth/1024/1024)
	} else {
		t.Logf("Memory decreased: %d MB", (startMemory-endMemory)/1024/1024)
	}
}

// TestStressReleasePatterns tests different release patterns
func TestStressReleasePatterns(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping stress test in short mode")
	}

	test.Prepare(t, config.Conf)
	defer test.Clean()

	iterations := 200

	t.Run("ManualRelease", func(t *testing.T) {
		startMemory := getMemStats()

		for i := 0; i < iterations; i++ {
			cxt := newStressTestContext(fmt.Sprintf("manual-%d", i))

			_, err := v8.Call(v8.CallOptions{}, `
				function test(ctx) {
					try {
						ctx.trace.Add({ type: "test" }, { label: "Manual Release" })
						return { success: true }
					} finally {
						ctx.Release() // Manual release
					}
				}`, cxt)

			if err != nil {
				t.Fatalf("Manual release iteration %d failed: %v", i, err)
			}
		}

		runtime.GC()
		endMemory := getMemStats()
		if endMemory > startMemory {
			t.Logf("Manual release: Memory growth: %d MB", (endMemory-startMemory)/1024/1024)
		} else {
			t.Logf("Manual release: Memory decreased: %d MB", (startMemory-endMemory)/1024/1024)
		}
	})

	t.Run("NoRelease_RelyOnGC", func(t *testing.T) {
		startMemory := getMemStats()

		for i := 0; i < iterations; i++ {
			cxt := newStressTestContext(fmt.Sprintf("gc-%d", i))

			_, err := v8.Call(v8.CallOptions{}, `
				function test(ctx) {
					ctx.trace.Add({ type: "test" }, { label: "GC Release" })
					return { success: true }
					// No manual release - rely on GC
				}`, cxt)

			if err != nil {
				t.Fatalf("GC release iteration %d failed: %v", i, err)
			}
		}

		// Force GC multiple times
		for i := 0; i < 3; i++ {
			runtime.GC()
			time.Sleep(50 * time.Millisecond)
		}

		endMemory := getMemStats()
		if endMemory > startMemory {
			t.Logf("GC release: Memory growth: %d MB", (endMemory-startMemory)/1024/1024)
		} else {
			t.Logf("GC release: Memory decreased: %d MB", (startMemory-endMemory)/1024/1024)
		}
	})

	t.Run("SeparateTraceRelease", func(t *testing.T) {
		startMemory := getMemStats()

		for i := 0; i < iterations; i++ {
			cxt := newStressTestContext(fmt.Sprintf("separate-%d", i))

			_, err := v8.Call(v8.CallOptions{}, `
				function test(ctx) {
					try {
						ctx.trace.Add({ type: "test" }, { label: "Separate Release" })
						ctx.trace.Release() // Release trace separately
						return { success: true }
					} finally {
						ctx.Release() // Release context
					}
				}`, cxt)

			if err != nil {
				t.Fatalf("Separate release iteration %d failed: %v", i, err)
			}
		}

		runtime.GC()
		endMemory := getMemStats()
		if endMemory > startMemory {
			t.Logf("Separate release: Memory growth: %d MB", (endMemory-startMemory)/1024/1024)
		} else {
			t.Logf("Separate release: Memory decreased: %d MB", (startMemory-endMemory)/1024/1024)
		}
	})
}

// TestStressLongRunningTrace tests long-running trace with many operations
func TestStressLongRunningTrace(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping stress test in short mode")
	}

	test.Prepare(t, config.Conf)
	defer test.Clean()

	cxt := newStressTestContext("long-running-test")

	startMemory := getMemStats()
	operations := 100

	_, err := v8.Call(v8.CallOptions{}, fmt.Sprintf(`
		function test(ctx) {
			const trace = ctx.trace
			const allNodes = []
			
			// Create many nested nodes
			for (let i = 0; i < %d; i++) {
				const parentNode = trace.Add(
					{ type: "parent", index: i },
					{ label: "Parent " + i }
				)
				allNodes.push(parentNode)
				
				// Create child nodes
				for (let j = 0; j < 5; j++) {
					const childNode = parentNode.Add(
						{ type: "child", parent: i, index: j },
						{ label: "Child " + i + "-" + j }
					)
					allNodes.push(childNode)
					
					// Add logs
					childNode.Info("Processing child " + i + "-" + j)
					childNode.Complete({ result: "success" })
				}
				
				parentNode.Complete({ result: "all children completed" })
			}
			
			// Release at the end
			trace.Release()
			ctx.Release()
			
			return { 
				totalNodes: allNodes.length,
				operations: %d
			}
		}`, operations, operations), cxt)

	if err != nil {
		t.Fatalf("Long running trace failed: %v", err)
	}

	runtime.GC()
	endMemory := getMemStats()

	expectedNodes := operations * 6 // parent + 5 children
	t.Logf("Long-running trace: %d operations, %d nodes", operations, expectedNodes)
	t.Logf("Start memory: %d MB", startMemory/1024/1024)
	t.Logf("End memory: %d MB", endMemory/1024/1024)

	if endMemory > startMemory {
		t.Logf("Memory growth: %d MB", (endMemory-startMemory)/1024/1024)
	} else {
		t.Logf("Memory decreased: %d MB", (startMemory-endMemory)/1024/1024)
	}
}

// Helper function to get current memory usage
func getMemStats() uint64 {
	runtime.GC()
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	return m.Alloc
}
