//go:build integration

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
	"github.com/stretchr/testify/require"
	v8 "github.com/yaoapp/gou/runtime/v8"
	agentctx "github.com/yaoapp/yao/agent/context"
	"github.com/yaoapp/yao/unit-test/agent/testprepare"
)

func newStressTestContext(chatID string) *agentctx.Context {
	ctx := agentctx.New(stdContext.Background(), nil, chatID)
	ctx.AssistantID = "test-assistant"
	ctx.Referer = agentctx.RefererAPI
	stack, _, _ := agentctx.EnterStack(ctx, "test-assistant", &agentctx.Options{})
	ctx.Stack = stack
	return ctx
}

func TestStressContextCreationAndRelease(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping stress test in short mode")
	}

	testprepare.PrepareSandbox(t)

	iterations := 1000
	startMemory := getMemStats()

	for i := 0; i < iterations; i++ {
		cxt := newStressTestContext(fmt.Sprintf("chat-%d", i))

		_, err := v8.Call(v8.CallOptions{}, `
			function test(ctx) {
				ctx.trace.Add({ type: "test" }, { label: "Test" })
				ctx.trace.Info("Processing")
				
				ctx.Release()
				
				return { iteration: true }
			}`, cxt)

		require.NoError(t, err, "Iteration %d failed", i)

		if i%100 == 0 {
			runtime.GC()
			currentMemory := getMemStats()
			t.Logf("Iteration %d: Memory usage: %d MB", i, currentMemory/1024/1024)
		}
	}

	runtime.GC()
	time.Sleep(100 * time.Millisecond)
	endMemory := getMemStats()

	t.Logf("Start memory: %d MB", startMemory/1024/1024)
	t.Logf("End memory: %d MB", endMemory/1024/1024)

	var memoryGrowth int64
	if endMemory > startMemory {
		memoryGrowth = int64(endMemory - startMemory)
		t.Logf("Memory growth: %d MB", memoryGrowth/1024/1024)
	} else {
		memoryGrowth = -int64(startMemory - endMemory)
		t.Logf("Memory decreased: %d MB", -memoryGrowth/1024/1024)
	}

	if memoryGrowth > 0 {
		assert.Less(t, memoryGrowth, int64(50*1024*1024), "Memory leak detected")
	}
}

func TestStressTraceOperations(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping stress test in short mode")
	}

	testprepare.PrepareSandbox(t)

	iterations := 500
	nodesPerIteration := 10

	startMemory := getMemStats()

	for i := 0; i < iterations; i++ {
		cxt := newStressTestContext(fmt.Sprintf("stress-test-chat-%d", i))
		_, err := v8.Call(v8.CallOptions{}, fmt.Sprintf(`
			function test(ctx) {
				const trace = ctx.trace
				const nodes = []
				
				for (let j = 0; j < %d; j++) {
					const node = trace.Add(
						{ type: "step", data: "data-" + j },
						{ label: "Step " + j }
					)
					nodes.push(node)
					
					node.Info("Processing step " + j)
					node.Debug("Debug info " + j)
				}
				
				for (const node of nodes) {
					node.Complete({ result: "success" })
				}
				
				ctx.Release()
				
				return { nodes: nodes.length }
			}`, nodesPerIteration), cxt)

		require.NoError(t, err, "Iteration %d failed", i)

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

func TestStressMCPOperations(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping stress test in short mode")
	}

	testprepare.PrepareSandbox(t)

	iterations := 500

	cxt := newStressTestContext("mcp-stress-test")

	startMemory := getMemStats()

	for i := 0; i < iterations; i++ {
		_, err := v8.Call(v8.CallOptions{}, `
			function test(ctx) {
				const tools = ctx.mcp.ListTools("echo", "")
				const resources = ctx.mcp.ListResources("echo", "")
				const prompts = ctx.mcp.ListPrompts("echo", "")
				
				const result1 = ctx.mcp.CallTool("echo", "ping", { count: 1 })
				const result2 = ctx.mcp.CallTool("echo", "status", { verbose: false })
				
				const info = ctx.mcp.ReadResource("echo", "echo://info")
				
				return {
					tools: tools.tools.length,
					resources: resources.resources.length,
					prompts: prompts.prompts.length
				}
			}`, cxt)

		require.NoError(t, err, "Iteration %d failed", i)

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

func TestStressConcurrentContexts(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping stress test in short mode")
	}

	testprepare.PrepareSandbox(t)

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
						const node = ctx.trace.Add({ type: "test" }, { label: "Concurrent Test" })
						ctx.trace.Info("Processing concurrent request")
						node.Complete({ result: "success" })
						
						const tools = ctx.mcp.ListTools("echo", "")
						
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

func TestStressNoOpTracePerformance(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping stress test in short mode")
	}

	testprepare.PrepareSandbox(t)

	iterations := 1000

	cxt := agentctx.New(stdContext.Background(), nil, "noop-stress-test")
	cxt.AssistantID = "test-assistant"

	startMemory := getMemStats()
	startTime := time.Now()

	for i := 0; i < iterations; i++ {
		_, err := v8.Call(v8.CallOptions{}, `
			function test(ctx) {
				const trace = ctx.trace
				
				trace.Info("No-op info")
				const node = trace.Add({ type: "test" }, { label: "No-op" })
				node.Info("No-op node info")
				node.Complete({ result: "done" })
				trace.Release()
				
				return { noop: true }
			}`, cxt)

		require.NoError(t, err, "Iteration %d failed", i)
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

	maxTimePerOp := 5 * time.Millisecond
	if os.Getenv("CI") != "" || os.Getenv("GITHUB_ACTIONS") != "" {
		maxTimePerOp = 15 * time.Millisecond
	}
	assert.Less(t, avgTimePerOp, maxTimePerOp, "No-op operations should be fast")

	if endMemory > startMemory {
		memoryGrowth := int64(endMemory - startMemory)
		assert.Less(t, memoryGrowth, int64(5*1024*1024), "No-op operations should not leak memory")
		t.Logf("Memory growth: %d MB", memoryGrowth/1024/1024)
	} else {
		t.Logf("Memory decreased: %d MB", (startMemory-endMemory)/1024/1024)
	}
}

func TestStressReleasePatterns(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping stress test in short mode")
	}

	testprepare.PrepareSandbox(t)

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
						ctx.Release()
					}
				}`, cxt)

			require.NoError(t, err, "Manual release iteration %d failed", i)
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
				}`, cxt)

			require.NoError(t, err, "GC release iteration %d failed", i)
		}

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
						ctx.trace.Release()
						return { success: true }
					} finally {
						ctx.Release()
					}
				}`, cxt)

			require.NoError(t, err, "Separate release iteration %d failed", i)
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

func TestStressLongRunningTrace(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping stress test in short mode")
	}

	testprepare.PrepareSandbox(t)

	cxt := newStressTestContext("long-running-test")

	startMemory := getMemStats()
	operations := 100

	_, err := v8.Call(v8.CallOptions{}, fmt.Sprintf(`
		function test(ctx) {
			const trace = ctx.trace
			const allNodes = []
			
			for (let i = 0; i < %d; i++) {
				const parentNode = trace.Add(
					{ type: "parent", index: i },
					{ label: "Parent " + i }
				)
				allNodes.push(parentNode)
				
				for (let j = 0; j < 5; j++) {
					const childNode = parentNode.Add(
						{ type: "child", parent: i, index: j },
						{ label: "Child " + i + "-" + j }
					)
					allNodes.push(childNode)
					
					childNode.Info("Processing child " + i + "-" + j)
					childNode.Complete({ result: "success" })
				}
				
				parentNode.Complete({ result: "all children completed" })
			}
			
			trace.Release()
			ctx.Release()
			
			return { 
				totalNodes: allNodes.length,
				operations: %d
			}
		}`, operations, operations), cxt)

	require.NoError(t, err, "Long running trace failed")

	runtime.GC()
	endMemory := getMemStats()

	expectedNodes := operations * 6
	t.Logf("Long-running trace: %d operations, %d nodes", operations, expectedNodes)
	t.Logf("Start memory: %d MB", startMemory/1024/1024)
	t.Logf("End memory: %d MB", endMemory/1024/1024)

	if endMemory > startMemory {
		t.Logf("Memory growth: %d MB", (endMemory-startMemory)/1024/1024)
	} else {
		t.Logf("Memory decreased: %d MB", (startMemory-endMemory)/1024/1024)
	}
}

func getMemStats() uint64 {
	runtime.GC()
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	return m.Alloc
}
