//go:build integration

package context_test

import (
	stdContext "context"
	"fmt"
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
	stack, _ := agentctx.EnterStack(ctx, "test-assistant", &agentctx.Options{})
	ctx.Stack = stack
	return ctx
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

func getMemStats() uint64 {
	runtime.GC()
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	return m.Alloc
}
