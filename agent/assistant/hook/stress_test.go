//go:build stress

package hook_test

import (
	stdContext "context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"runtime/pprof"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/yaoapp/yao/agent/assistant"
	agentContext "github.com/yaoapp/yao/agent/context"
	oauthTypes "github.com/yaoapp/yao/openapi/oauth/types"
	"github.com/yaoapp/yao/unit-test/agent/testprepare"
)

// ============================================================================
// Real World Stress Tests
// ============================================================================

func TestRealWorldSimpleScenario(t *testing.T) {
	testprepare.PrepareE2E(t)

	agent, err := assistant.Get("tests.realworld")
	require.NoError(t, err)
	require.NotNil(t, agent.HookScript)

	ctx := newRealWorldContext("test-simple", "tests.realworld")

	messages := []agentContext.Message{
		{Role: "user", Content: "simple"},
	}

	response, _, err := agent.HookScript.Create(ctx, messages)
	require.NoError(t, err)
	require.NotNil(t, response)
	assert.NotEmpty(t, response.Messages)
	assert.Equal(t, "simple", response.Metadata["scenario"])
}

func TestRealWorldMCPScenarios(t *testing.T) {
	testprepare.PrepareE2E(t)

	agent, err := assistant.Get("tests.realworld")
	require.NoError(t, err)
	require.NotNil(t, agent.HookScript)

	t.Run("MCPHealth", func(t *testing.T) {
		ctx := newRealWorldContext("test-mcp-health", "tests.realworld")

		messages := []agentContext.Message{
			{Role: "user", Content: "mcp_health"},
		}

		response, _, err := agent.HookScript.Create(ctx, messages)
		require.NoError(t, err)
		require.NotNil(t, response)
		assert.NotEmpty(t, response.Messages)

		if response.Metadata == nil {
			messageContent := ""
			for _, msg := range response.Messages {
				if content, ok := msg.Content.(string); ok {
					messageContent += content + "\n"
				}
			}
			assert.Contains(t, messageContent, "Health")
			assert.Contains(t, messageContent, "Tools")
		} else {
			assert.Equal(t, "mcp_health", response.Metadata["scenario"])
			if toolsCount, ok := response.Metadata["tools_count"]; ok {
				count := int(toolsCount.(float64))
				assert.Greater(t, count, 0, "Should have tools from MCP")
			}
		}

		ctx.Release()
	})

	t.Run("MCPTools", func(t *testing.T) {
		ctx := newRealWorldContext("test-mcp-tools", "tests.realworld")

		messages := []agentContext.Message{
			{Role: "user", Content: "mcp_tools"},
		}

		response, _, err := agent.HookScript.Create(ctx, messages)
		require.NoError(t, err)
		require.NotNil(t, response)
		assert.NotEmpty(t, response.Messages)

		if response.Metadata == nil {
			messageContent := ""
			for _, msg := range response.Messages {
				if content, ok := msg.Content.(string); ok {
					messageContent += content + "\n"
				}
			}
			assert.Contains(t, messageContent, "Tools")
			assert.Contains(t, messageContent, "Ping")
		} else {
			assert.Equal(t, "mcp_tools", response.Metadata["scenario"])
			if toolsCount, ok := response.Metadata["tools_count"]; ok {
				count := int(toolsCount.(float64))
				assert.Greater(t, count, 0)
				if operations, ok := response.Metadata["operations"].([]interface{}); ok {
					assert.Len(t, operations, 2)
				}
			}
		}

		ctx.Release()
	})

	t.Run("FullWorkflow", func(t *testing.T) {
		ctx := newRealWorldContext("test-full-workflow", "tests.realworld")

		stack, _, done := agentContext.EnterStack(ctx, "tests.realworld", &agentContext.Options{})
		defer done()
		ctx.Stack = stack

		messages := []agentContext.Message{
			{Role: "user", Content: "full_workflow"},
		}

		response, _, err := agent.HookScript.Create(ctx, messages)
		require.NoError(t, err)
		require.NotNil(t, response)
		assert.NotEmpty(t, response.Messages)

		if response.Metadata == nil {
			messageContent := ""
			for _, msg := range response.Messages {
				if content, ok := msg.Content.(string); ok {
					messageContent += content + "\n"
				}
			}
			assert.Contains(t, messageContent, "Workflow")
			assert.Contains(t, messageContent, "Tools")
			assert.Contains(t, messageContent, "Roles")
		} else {
			assert.Equal(t, "full_workflow", response.Metadata["scenario"])
			if phasesCompleted, ok := response.Metadata["phases_completed"]; ok {
				phases := int(phasesCompleted.(float64))
				assert.Equal(t, 4, phases)
			}
		}

		ctx.Release()
	})
}

func TestRealWorldTraceIntensive(t *testing.T) {
	testprepare.PrepareE2E(t)

	agent, err := assistant.Get("tests.realworld")
	require.NoError(t, err)
	require.NotNil(t, agent.HookScript)

	ctx := newRealWorldContext("test-trace-intensive", "tests.realworld")
	stack, _, done := agentContext.EnterStack(ctx, "tests.realworld", &agentContext.Options{})
	defer done()
	ctx.Stack = stack

	messages := []agentContext.Message{
		{Role: "user", Content: "trace_intensive"},
	}

	response, _, err := agent.HookScript.Create(ctx, messages)
	require.NoError(t, err)
	require.NotNil(t, response)
	assert.Equal(t, "trace_intensive", response.Metadata["scenario"])
	assert.NotZero(t, response.Metadata["nodes_created"])
}

func TestRealWorldStressSimple(t *testing.T) {
	testprepare.PrepareE2E(t)

	agent, err := assistant.Get("tests.realworld")
	require.NoError(t, err)
	require.NotNil(t, agent.HookScript)

	iterations := 100
	startMemory := stressGetMemStats()

	for i := 0; i < iterations; i++ {
		ctx := newRealWorldContext(fmt.Sprintf("stress-simple-%d", i), "tests.realworld")

		messages := []agentContext.Message{
			{Role: "user", Content: "simple"},
		}

		response, _, err := agent.HookScript.Create(ctx, messages)
		require.NoError(t, err, "Iteration %d failed", i)
		require.NotNil(t, response, "Iteration %d: response should not be nil", i)
		assert.NotEmpty(t, response.Messages, "Iteration %d: messages should not be empty", i)
		if response.Metadata != nil {
			assert.Equal(t, "simple", response.Metadata["scenario"], "Iteration %d: scenario mismatch", i)
		}

		ctx.Release()

		if i%20 == 0 {
			runtime.GC()
			currentMemory := stressGetMemStats()
			t.Logf("Iteration %d: Memory: %d MB", i, currentMemory/1024/1024)
		}
	}

	runtime.GC()
	endMemory := stressGetMemStats()

	t.Logf("Simple stress: %d iterations", iterations)
	t.Logf("Start memory: %d MB", startMemory/1024/1024)
	t.Logf("End memory: %d MB", endMemory/1024/1024)
	if endMemory > startMemory {
		t.Logf("Memory growth: %d MB", (endMemory-startMemory)/1024/1024)
	}
}

func TestRealWorldStressMCP(t *testing.T) {
	testprepare.PrepareE2E(t)

	agent, err := assistant.Get("tests.realworld")
	require.NoError(t, err)
	require.NotNil(t, agent.HookScript)

	iterations := 50
	scenarios := []string{"mcp_health", "mcp_tools"}
	startMemory := stressGetMemStats()

	for i := 0; i < iterations; i++ {
		scenario := scenarios[i%len(scenarios)]
		ctx := newRealWorldContext(fmt.Sprintf("stress-mcp-%d", i), "tests.realworld")

		stack, _, done := agentContext.EnterStack(ctx, "tests.realworld", &agentContext.Options{})
		ctx.Stack = stack

		messages := []agentContext.Message{
			{Role: "user", Content: scenario},
		}

		response, _, err := agent.HookScript.Create(ctx, messages)
		require.NoError(t, err, "Iteration %d (%s) failed", i, scenario)
		require.NotNil(t, response, "Iteration %d (%s): response should not be nil", i, scenario)
		assert.NotEmpty(t, response.Messages, "Iteration %d (%s): messages should not be empty", i, scenario)

		if response.Metadata != nil {
			assert.Equal(t, scenario, response.Metadata["scenario"], "Iteration %d: scenario mismatch", i)

			if scenario == "mcp_health" {
				assert.NotNil(t, response.Metadata["tools_count"], "Iteration %d: should have tools_count", i)
				if toolsCount, ok := response.Metadata["tools_count"].(float64); ok {
					assert.Greater(t, int(toolsCount), 0, "Iteration %d: should have at least 1 tool", i)
					assert.Equal(t, 3, int(toolsCount), "Iteration %d: echo should have 3 tools", i)
				}
				assert.NotNil(t, response.Metadata["health_data"], "Iteration %d: should have health_data", i)
			} else if scenario == "mcp_tools" {
				assert.NotNil(t, response.Metadata["tools_count"], "Iteration %d: should have tools_count", i)
				if toolsCount, ok := response.Metadata["tools_count"].(float64); ok {
					assert.Equal(t, 3, int(toolsCount), "Iteration %d: echo should have 3 tools", i)
				}
				assert.NotNil(t, response.Metadata["operations"], "Iteration %d: should have operations", i)
				if operations, ok := response.Metadata["operations"].([]interface{}); ok {
					assert.Len(t, operations, 2, "Iteration %d: should have 2 operations", i)
				}
			}
		} else {
			t.Errorf("Iteration %d (%s): metadata is nil", i, scenario)
		}

		done()
		ctx.Release()

		if i%10 == 0 {
			runtime.GC()
			currentMemory := stressGetMemStats()
			t.Logf("Iteration %d (%s): Memory: %d MB", i, scenario, currentMemory/1024/1024)
		}
	}

	runtime.GC()
	endMemory := stressGetMemStats()

	t.Logf("MCP stress: %d iterations", iterations)
	t.Logf("Start memory: %d MB", startMemory/1024/1024)
	t.Logf("End memory: %d MB", endMemory/1024/1024)
	if endMemory > startMemory {
		t.Logf("Memory growth: %d MB", (endMemory-startMemory)/1024/1024)
	}
}

func TestRealWorldStressFullWorkflow(t *testing.T) {
	testprepare.PrepareE2E(t)

	agent, err := assistant.Get("tests.realworld")
	require.NoError(t, err)
	require.NotNil(t, agent.HookScript)

	iterations := 30
	startMemory := stressGetMemStats()
	startTime := time.Now()

	for i := 0; i < iterations; i++ {
		ctx := newRealWorldContext(fmt.Sprintf("stress-workflow-%d", i), "tests.realworld")

		stack, _, done := agentContext.EnterStack(ctx, "tests.realworld", &agentContext.Options{})
		ctx.Stack = stack

		messages := []agentContext.Message{
			{Role: "user", Content: "full_workflow"},
		}

		response, _, err := agent.HookScript.Create(ctx, messages)
		require.NoError(t, err, "Iteration %d failed", i)
		require.NotNil(t, response, "Iteration %d: response should not be nil", i)
		assert.NotEmpty(t, response.Messages, "Iteration %d: messages should not be empty", i)
		if response.Metadata != nil {
			assert.Equal(t, "full_workflow", response.Metadata["scenario"], "Iteration %d: scenario mismatch", i)
			if phasesCompleted, ok := response.Metadata["phases_completed"]; ok {
				phases := int(phasesCompleted.(float64))
				assert.Equal(t, 4, phases, "Iteration %d: should complete 4 phases", i)
			}
			if mcpTools, ok := response.Metadata["mcp_tools"]; ok {
				tools := int(mcpTools.(float64))
				assert.Greater(t, tools, 0, "Iteration %d: should have MCP tools", i)
			}
		}

		done()
		ctx.Release()

		if i%10 == 0 {
			runtime.GC()
			currentMemory := stressGetMemStats()
			elapsed := time.Since(startTime)
			t.Logf("Iteration %d: Memory: %d MB, Elapsed: %v", i, currentMemory/1024/1024, elapsed)
		}
	}

	duration := time.Since(startTime)
	runtime.GC()
	endMemory := stressGetMemStats()

	avgTime := duration / time.Duration(iterations)
	t.Logf("Full workflow stress: %d iterations", iterations)
	t.Logf("Total time: %v, Average: %v", duration, avgTime)
	t.Logf("Start memory: %d MB, End memory: %d MB", startMemory/1024/1024, endMemory/1024/1024)
}

func TestRealWorldStressConcurrent(t *testing.T) {
	testprepare.PrepareE2E(t)

	agent, err := assistant.Get("tests.realworld")
	require.NoError(t, err)
	require.NotNil(t, agent.HookScript)

	goroutines := 100
	iterationsPerGoroutine := 10
	scenarios := []string{"simple", "mcp_health", "mcp_tools", "full_workflow"}

	startMemory := stressGetMemStats()
	startTime := time.Now()

	var wg sync.WaitGroup
	errors := make(chan error, goroutines*iterationsPerGoroutine)

	type Result struct {
		goroutineID int
		iteration   int
		scenario    string
		metadata    map[string]interface{}
	}
	results := make(chan Result, goroutines*iterationsPerGoroutine)

	for g := 0; g < goroutines; g++ {
		wg.Add(1)
		go func(goroutineID int) {
			defer wg.Done()

			for i := 0; i < iterationsPerGoroutine; i++ {
				scenario := scenarios[(goroutineID+i)%len(scenarios)]
				ctx := newRealWorldContext(
					fmt.Sprintf("concurrent-%d-%d", goroutineID, i),
					"tests.realworld",
				)

				stack, _, done := agentContext.EnterStack(ctx, "tests.realworld", &agentContext.Options{})
				ctx.Stack = stack

				messages := []agentContext.Message{
					{Role: "user", Content: scenario},
				}

				response, _, err := agent.HookScript.Create(ctx, messages)
				if err != nil {
					errors <- fmt.Errorf("goroutine %d iteration %d (%s): %v", goroutineID, i, scenario, err)
					done()
					ctx.Release()
					return
				}

				if response == nil {
					errors <- fmt.Errorf("goroutine %d iteration %d (%s): nil response", goroutineID, i, scenario)
					done()
					ctx.Release()
					return
				}

				if len(response.Messages) == 0 {
					errors <- fmt.Errorf("goroutine %d iteration %d (%s): empty messages", goroutineID, i, scenario)
					done()
					ctx.Release()
					return
				}

				results <- Result{
					goroutineID: goroutineID,
					iteration:   i,
					scenario:    scenario,
					metadata:    response.Metadata,
				}

				done()
				ctx.Release()
			}
		}(g)
	}

	wg.Wait()
	close(errors)
	close(results)

	duration := time.Since(startTime)
	runtime.GC()
	endMemory := stressGetMemStats()

	errorCount := 0
	for err := range errors {
		t.Error(err)
		errorCount++
	}
	assert.Equal(t, 0, errorCount, "No errors should occur in concurrent operations")

	scenarioCounts := make(map[string]int)
	validResults := 0
	for result := range results {
		validResults++
		scenarioCounts[result.scenario]++
		if result.metadata != nil {
			if scenario, ok := result.metadata["scenario"].(string); ok {
				if scenario != result.scenario {
					t.Errorf("Metadata mismatch: expected %s, got %s (goroutine %d, iteration %d)",
						result.scenario, scenario, result.goroutineID, result.iteration)
				}
			}
		}
	}

	totalOperations := goroutines * iterationsPerGoroutine
	assert.Equal(t, totalOperations, validResults, "All operations should return valid results")

	avgTime := duration / time.Duration(totalOperations)
	t.Logf("Concurrent stress: %d operations (%d goroutines x %d iterations)", totalOperations, goroutines, iterationsPerGoroutine)
	t.Logf("Valid results: %d/%d", validResults, totalOperations)
	for scenario, count := range scenarioCounts {
		t.Logf("  %s: %d operations", scenario, count)
	}
	t.Logf("Total time: %v, Average: %v", duration, avgTime)
	t.Logf("Start memory: %d MB, End memory: %d MB", startMemory/1024/1024, endMemory/1024/1024)
	if endMemory > startMemory {
		t.Logf("Memory growth: %d MB", (endMemory-startMemory)/1024/1024)
	}
}

func TestRealWorldStressResourceHeavy(t *testing.T) {
	testprepare.PrepareE2E(t)

	agent, err := assistant.Get("tests.realworld")
	require.NoError(t, err)
	require.NotNil(t, agent.HookScript)

	iterations := 20
	startMemory := stressGetMemStats()
	startTime := time.Now()

	for i := 0; i < iterations; i++ {
		ctx := newRealWorldContext(fmt.Sprintf("stress-heavy-%d", i), "tests.realworld")

		stack, _, done := agentContext.EnterStack(ctx, "tests.realworld", &agentContext.Options{})
		ctx.Stack = stack

		messages := []agentContext.Message{
			{Role: "user", Content: "resource_heavy"},
		}

		response, _, err := agent.HookScript.Create(ctx, messages)
		require.NoError(t, err, "Iteration %d failed", i)
		require.NotNil(t, response, "Iteration %d: response should not be nil", i)
		assert.NotEmpty(t, response.Messages, "Iteration %d: messages should not be empty", i)
		if response.Metadata != nil {
			assert.Equal(t, "resource_heavy", response.Metadata["scenario"], "Iteration %d: scenario mismatch", i)
			if mcpIterations, ok := response.Metadata["mcp_iterations"]; ok {
				iters := int(mcpIterations.(float64))
				assert.Equal(t, 5, iters, "Iteration %d: should have 5 MCP iterations", i)
			}
		}

		done()
		ctx.Release()

		if i%5 == 0 {
			runtime.GC()
			currentMemory := stressGetMemStats()
			elapsed := time.Since(startTime)
			t.Logf("Iteration %d: Memory: %d MB, Elapsed: %v", i, currentMemory/1024/1024, elapsed)
		}
	}

	duration := time.Since(startTime)
	runtime.GC()
	endMemory := stressGetMemStats()

	avgTime := duration / time.Duration(iterations)
	t.Logf("Resource heavy stress: %d iterations", iterations)
	t.Logf("Total time: %v, Average: %v", duration, avgTime)
	t.Logf("Start memory: %d MB, End memory: %d MB", startMemory/1024/1024, endMemory/1024/1024)

	if endMemory > startMemory {
		memoryGrowth := int64(endMemory - startMemory)
		t.Logf("Memory growth: %d MB", memoryGrowth/1024/1024)
		assert.Less(t, memoryGrowth, int64(100*1024*1024), "Memory growth should be reasonable")
	}
}

// ============================================================================
// Goroutine Leak Tests
// ============================================================================

func TestGoroutineLeakDetailed(t *testing.T) {
	testprepare.PrepareE2E(t)

	agent, err := assistant.Get("tests.create")
	require.NoError(t, err)
	require.NotNil(t, agent.HookScript)

	profileDir := filepath.Join(os.TempDir(), "goroutine_profiles")
	os.MkdirAll(profileDir, 0755)

	runtime.GC()
	time.Sleep(200 * time.Millisecond)
	initialGoroutines := runtime.NumGoroutine()

	stressSaveGoroutineProfile(filepath.Join(profileDir, "00_initial.txt"))
	t.Logf("Initial goroutines: %d", initialGoroutines)

	iterations := 10
	for i := 0; i < iterations; i++ {
		ctx := newLeakTestContext(fmt.Sprintf("leak-test-%d", i), "tests.create")

		_, _, err := agent.HookScript.Create(ctx, []agentContext.Message{
			{Role: "user", Content: "Hello"},
		})
		require.NoError(t, err, "Create failed at iteration %d", i)

		ctx.Release()

		current := runtime.NumGoroutine()
		growth := current - initialGoroutines
		t.Logf("After iteration %d: %d goroutines (growth: %d)", i+1, current, growth)

		if (i+1)%5 == 0 {
			stressSaveGoroutineProfile(filepath.Join(profileDir, fmt.Sprintf("%02d_after_iter_%d.txt", i+1, i+1)))
		}
	}

	runtime.GC()
	time.Sleep(500 * time.Millisecond)

	finalGoroutines := runtime.NumGoroutine()
	growth := finalGoroutines - initialGoroutines

	t.Logf("=== SUMMARY ===")
	t.Logf("Initial:  %d goroutines", initialGoroutines)
	t.Logf("Final:    %d goroutines", finalGoroutines)
	t.Logf("Growth:   %d goroutines (%.2f per iteration)", growth, float64(growth)/float64(iterations))

	stressSaveGoroutineProfile(filepath.Join(profileDir, "99_final.txt"))
	stressAnalyzeGoroutineProfiles(t, profileDir)
}

func TestGoroutineLeakByComponent(t *testing.T) {
	testprepare.PrepareE2E(t)

	agent, err := assistant.Get("tests.create")
	require.NoError(t, err)
	require.NotNil(t, agent.HookScript)

	compProfileDir := filepath.Join(os.TempDir(), "component_profiles")
	os.MkdirAll(compProfileDir, 0755)

	t.Run("ContextCreationOnly", func(t *testing.T) {
		runtime.GC()
		time.Sleep(100 * time.Millisecond)
		initial := runtime.NumGoroutine()

		for i := 0; i < 10; i++ {
			ctx := newLeakTestContext(fmt.Sprintf("test-%d", i), "tests.create")
			_ = ctx
			ctx.Release()
		}

		runtime.GC()
		time.Sleep(100 * time.Millisecond)
		final := runtime.NumGoroutine()

		t.Logf("Context creation: initial=%d, final=%d, growth=%d", initial, final, final-initial)
	})

	t.Run("ScriptExecutionOnly", func(t *testing.T) {
		runtime.GC()
		time.Sleep(100 * time.Millisecond)
		initial := runtime.NumGoroutine()

		for i := 0; i < 10; i++ {
			ctx := newLeakTestContext(fmt.Sprintf("test-%d", i), "tests.create")
			_, _, _ = agent.HookScript.Create(ctx, []agentContext.Message{
				{Role: "user", Content: "Hello"},
			})
			ctx.Release()
		}

		runtime.GC()
		time.Sleep(100 * time.Millisecond)
		final := runtime.NumGoroutine()

		t.Logf("Script execution: initial=%d, final=%d, growth=%d", initial, final, final-initial)
		stressSaveGoroutineProfile(filepath.Join(compProfileDir, "script_execution.txt"))
	})

	t.Run("TraceOperations", func(t *testing.T) {
		runtime.GC()
		time.Sleep(100 * time.Millisecond)
		initial := runtime.NumGoroutine()

		for i := 0; i < 10; i++ {
			ctx := newLeakTestContext(fmt.Sprintf("test-%d", i), "tests.create")

			trace, err := ctx.Trace()
			if err == nil && trace != nil {
				_ = trace
			}

			ctx.Release()
		}

		runtime.GC()
		time.Sleep(100 * time.Millisecond)
		final := runtime.NumGoroutine()

		t.Logf("Trace operations: initial=%d, final=%d, growth=%d", initial, final, final-initial)
		stressSaveGoroutineProfile(filepath.Join(compProfileDir, "trace_operations.txt"))
	})
}

func TestGoroutineLeakWithoutRelease(t *testing.T) {
	testprepare.PrepareE2E(t)

	agent, err := assistant.Get("tests.create")
	require.NoError(t, err)
	require.NotNil(t, agent.HookScript)

	t.Run("WithoutRelease", func(t *testing.T) {
		runtime.GC()
		time.Sleep(100 * time.Millisecond)
		initial := runtime.NumGoroutine()

		for i := 0; i < 10; i++ {
			ctx := newLeakTestContext(fmt.Sprintf("no-release-%d", i), "tests.create")
			_, _, _ = agent.HookScript.Create(ctx, []agentContext.Message{
				{Role: "user", Content: "Hello"},
			})
		}

		runtime.GC()
		time.Sleep(100 * time.Millisecond)
		final := runtime.NumGoroutine()

		t.Logf("WITHOUT Release: initial=%d, final=%d, growth=%d (%.1f per iter)",
			initial, final, final-initial, float64(final-initial)/10.0)
	})

	t.Run("WithRelease", func(t *testing.T) {
		runtime.GC()
		time.Sleep(100 * time.Millisecond)
		initial := runtime.NumGoroutine()

		for i := 0; i < 10; i++ {
			ctx := newLeakTestContext(fmt.Sprintf("with-release-%d", i), "tests.create")
			_, _, _ = agent.HookScript.Create(ctx, []agentContext.Message{
				{Role: "user", Content: "Hello"},
			})
			ctx.Release()
		}

		runtime.GC()
		time.Sleep(100 * time.Millisecond)
		final := runtime.NumGoroutine()

		t.Logf("WITH Release: initial=%d, final=%d, growth=%d (%.1f per iter)",
			initial, final, final-initial, float64(final-initial)/10.0)
	})
}

// ============================================================================
// Memory Leak Detection Tests
// ============================================================================

func TestMemoryLeakStandardMode(t *testing.T) {
	testprepare.PrepareE2E(t)

	agent, err := assistant.Get("tests.create")
	require.NoError(t, err)
	require.NotNil(t, agent.HookScript)

	for i := 0; i < 10; i++ {
		ctx := newMemTestContext("warmup", "tests.create")
		_, _, _ = agent.HookScript.Create(ctx, []agentContext.Message{
			{Role: "user", Content: "Hello"},
		})
		ctx.Release()
	}

	runtime.GC()
	time.Sleep(100 * time.Millisecond)
	var baseline runtime.MemStats
	runtime.ReadMemStats(&baseline)

	iterations := 1000
	for i := 0; i < iterations; i++ {
		ctx := newMemTestContext("mem-test-standard", "tests.create")
		_, _, err := agent.HookScript.Create(ctx, []agentContext.Message{
			{Role: "user", Content: "Hello"},
		})
		require.NoError(t, err, "Create failed at iteration %d", i)
		ctx.Release()

		if i%100 == 0 {
			runtime.GC()
		}
	}

	runtime.GC()
	time.Sleep(100 * time.Millisecond)
	var final runtime.MemStats
	runtime.ReadMemStats(&final)

	growth := int64(final.HeapAlloc) - int64(baseline.HeapAlloc)
	growthPerIteration := float64(growth) / float64(iterations)

	t.Logf("Memory Statistics (Standard Mode):")
	t.Logf("  Iterations: %d", iterations)
	t.Logf("  Baseline HeapAlloc: %d bytes (%.2f MB)", baseline.HeapAlloc, float64(baseline.HeapAlloc)/1024/1024)
	t.Logf("  Final HeapAlloc: %d bytes (%.2f MB)", final.HeapAlloc, float64(final.HeapAlloc)/1024/1024)
	t.Logf("  Growth per iteration: %.2f bytes", growthPerIteration)
	t.Logf("  GC Runs: %d", final.NumGC-baseline.NumGC)

	maxGrowthPerIteration := 20480.0
	if growthPerIteration > maxGrowthPerIteration {
		t.Errorf("Possible memory leak: %.2f bytes/iteration (threshold: %.2f)", growthPerIteration, maxGrowthPerIteration)
	}
}

func TestMemoryLeakPerformanceMode(t *testing.T) {
	testprepare.PrepareE2E(t)

	agent, err := assistant.Get("tests.create")
	require.NoError(t, err)
	require.NotNil(t, agent.HookScript)

	for i := 0; i < 20; i++ {
		ctx := newMemTestContext("warmup", "tests.create")
		_, _, _ = agent.HookScript.Create(ctx, []agentContext.Message{
			{Role: "user", Content: "Hello"},
		})
		ctx.Release()
	}

	runtime.GC()
	time.Sleep(100 * time.Millisecond)
	var baseline runtime.MemStats
	runtime.ReadMemStats(&baseline)

	iterations := 1000
	for i := 0; i < iterations; i++ {
		ctx := newMemTestContext("mem-test-performance", "tests.create")
		_, _, err := agent.HookScript.Create(ctx, []agentContext.Message{
			{Role: "user", Content: "Hello"},
		})
		require.NoError(t, err, "Create failed at iteration %d", i)
		ctx.Release()

		if i%100 == 0 {
			runtime.GC()
		}
	}

	runtime.GC()
	time.Sleep(100 * time.Millisecond)
	var final runtime.MemStats
	runtime.ReadMemStats(&final)

	growth := int64(final.HeapAlloc) - int64(baseline.HeapAlloc)
	growthPerIteration := float64(growth) / float64(iterations)

	t.Logf("Memory Statistics (Performance Mode):")
	t.Logf("  Iterations: %d", iterations)
	t.Logf("  Baseline HeapAlloc: %d bytes (%.2f MB)", baseline.HeapAlloc, float64(baseline.HeapAlloc)/1024/1024)
	t.Logf("  Final HeapAlloc: %d bytes (%.2f MB)", final.HeapAlloc, float64(final.HeapAlloc)/1024/1024)
	t.Logf("  Growth per iteration: %.2f bytes", growthPerIteration)
	t.Logf("  GC Runs: %d", final.NumGC-baseline.NumGC)

	maxGrowthPerIteration := 5120.0
	if growthPerIteration > maxGrowthPerIteration {
		t.Errorf("Possible memory leak: %.2f bytes/iteration (threshold: %.2f)", growthPerIteration, maxGrowthPerIteration)
	}
}

func TestMemoryLeakBusinessScenarios(t *testing.T) {
	testprepare.PrepareE2E(t)

	agent, err := assistant.Get("tests.create")
	require.NoError(t, err)
	require.NotNil(t, agent.HookScript)

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

	for i := 0; i < 10; i++ {
		ctx := newMemTestContext("warmup", "tests.create")
		_, _, _ = agent.HookScript.Create(ctx, []agentContext.Message{
			{Role: "user", Content: "return_full"},
		})
		ctx.Release()
	}

	for _, scenario := range scenarios {
		t.Run(scenario.name, func(t *testing.T) {
			runtime.GC()
			time.Sleep(50 * time.Millisecond)
			var baseline runtime.MemStats
			runtime.ReadMemStats(&baseline)

			iterations := 200
			for i := 0; i < iterations; i++ {
				ctx := newMemTestContext("mem-test-business", "tests.create")
				_, _, err := agent.HookScript.Create(ctx, []agentContext.Message{
					{Role: "user", Content: scenario.content},
				})
				require.NoError(t, err, "Create failed at iteration %d", i)
				ctx.Release()

				if i%50 == 0 {
					runtime.GC()
				}
			}

			runtime.GC()
			time.Sleep(50 * time.Millisecond)
			var final runtime.MemStats
			runtime.ReadMemStats(&final)

			growth := int64(final.HeapAlloc) - int64(baseline.HeapAlloc)
			growthPerIteration := float64(growth) / float64(iterations)

			t.Logf("  Baseline: %.2f MB, Final: %.2f MB, Growth/iter: %.2f bytes",
				float64(baseline.HeapAlloc)/1024/1024, float64(final.HeapAlloc)/1024/1024, growthPerIteration)

			maxGrowthPerIteration := 20480.0
			if growthPerIteration > maxGrowthPerIteration {
				t.Errorf("Possible memory leak: %.2f bytes/iteration (threshold: %.2f)", growthPerIteration, maxGrowthPerIteration)
			}
		})
	}
}

func TestMemoryLeakConcurrent(t *testing.T) {
	testprepare.PrepareE2E(t)

	agent, err := assistant.Get("tests.create")
	require.NoError(t, err)
	require.NotNil(t, agent.HookScript)

	for i := 0; i < 20; i++ {
		ctx := newMemTestContext("warmup", "tests.create")
		_, _, _ = agent.HookScript.Create(ctx, []agentContext.Message{
			{Role: "user", Content: "Hello"},
		})
		ctx.Release()
	}

	runtime.GC()
	time.Sleep(100 * time.Millisecond)
	var baseline runtime.MemStats
	runtime.ReadMemStats(&baseline)

	iterations := 1000
	concurrency := 10
	iterPerGoroutine := iterations / concurrency

	done := make(chan bool, concurrency)
	for g := 0; g < concurrency; g++ {
		go func(id int) {
			defer func() { done <- true }()
			for i := 0; i < iterPerGoroutine; i++ {
				ctx := newMemTestContext("mem-test-concurrent", "tests.create")
				_, _, err := agent.HookScript.Create(ctx, []agentContext.Message{
					{Role: "user", Content: "Hello"},
				})
				if err != nil {
					t.Errorf("Goroutine %d failed at iteration %d: %s", id, i, err.Error())
				}
				ctx.Release()
			}
		}(g)
	}

	for g := 0; g < concurrency; g++ {
		<-done
	}

	runtime.GC()
	time.Sleep(100 * time.Millisecond)
	var final runtime.MemStats
	runtime.ReadMemStats(&final)

	growth := int64(final.HeapAlloc) - int64(baseline.HeapAlloc)
	growthPerIteration := float64(growth) / float64(iterations)

	t.Logf("Memory Statistics (Concurrent Load):")
	t.Logf("  Iterations: %d, Concurrency: %d", iterations, concurrency)
	t.Logf("  Baseline: %.2f MB, Final: %.2f MB", float64(baseline.HeapAlloc)/1024/1024, float64(final.HeapAlloc)/1024/1024)
	t.Logf("  Growth per iteration: %.2f bytes", growthPerIteration)

	maxGrowthPerIteration := 10240.0
	if growthPerIteration > maxGrowthPerIteration {
		t.Errorf("Possible memory leak: %.2f bytes/iteration (threshold: %.2f)", growthPerIteration, maxGrowthPerIteration)
	}
}

func TestMemoryLeakNestedCalls(t *testing.T) {
	testprepare.PrepareE2E(t)

	agent, err := assistant.Get("tests.create")
	require.NoError(t, err)
	require.NotNil(t, agent.HookScript)

	for i := 0; i < 10; i++ {
		ctx := newMemTestContext("warmup", "tests.create")
		_, _, _ = agent.HookScript.Create(ctx, []agentContext.Message{
			{Role: "user", Content: "nested_script_call"},
		})
		ctx.Release()
	}

	runtime.GC()
	time.Sleep(100 * time.Millisecond)
	var baseline runtime.MemStats
	runtime.ReadMemStats(&baseline)

	iterations := 200
	for i := 0; i < iterations; i++ {
		ctx := newMemTestContext("mem-test-nested", "tests.create")
		_, _, err := agent.HookScript.Create(ctx, []agentContext.Message{
			{Role: "user", Content: "deep_nested_call"},
		})
		require.NoError(t, err, "Nested call failed at iteration %d", i)
		ctx.Release()

		if i%50 == 0 {
			runtime.GC()
		}
	}

	runtime.GC()
	time.Sleep(100 * time.Millisecond)
	var final runtime.MemStats
	runtime.ReadMemStats(&final)

	growth := int64(final.HeapAlloc) - int64(baseline.HeapAlloc)
	growthPerIteration := float64(growth) / float64(iterations)

	t.Logf("Memory Statistics (Nested Calls):")
	t.Logf("  Iterations: %d", iterations)
	t.Logf("  Baseline: %.2f MB, Final: %.2f MB", float64(baseline.HeapAlloc)/1024/1024, float64(final.HeapAlloc)/1024/1024)
	t.Logf("  Growth per iteration: %.2f bytes", growthPerIteration)

	maxGrowthPerIteration := 20480.0
	if growthPerIteration > maxGrowthPerIteration {
		t.Errorf("Possible memory leak: %.2f bytes/iteration (threshold: %.2f)", growthPerIteration, maxGrowthPerIteration)
	}
}

func TestMemoryLeakNestedConcurrent(t *testing.T) {
	testprepare.PrepareE2E(t)

	agent, err := assistant.Get("tests.create")
	require.NoError(t, err)
	require.NotNil(t, agent.HookScript)

	for i := 0; i < 20; i++ {
		ctx := newMemTestContext("warmup", "tests.create")
		_, _, _ = agent.HookScript.Create(ctx, []agentContext.Message{
			{Role: "user", Content: "nested_script_call"},
		})
		ctx.Release()
	}

	runtime.GC()
	time.Sleep(100 * time.Millisecond)
	var baseline runtime.MemStats
	runtime.ReadMemStats(&baseline)

	iterations := 500
	concurrency := 10
	iterPerGoroutine := iterations / concurrency

	done := make(chan bool, concurrency)
	for g := 0; g < concurrency; g++ {
		go func(id int) {
			defer func() { done <- true }()
			for i := 0; i < iterPerGoroutine; i++ {
				ctx := newMemTestContext("mem-test-nested-concurrent", "tests.create")
				_, _, err := agent.HookScript.Create(ctx, []agentContext.Message{
					{Role: "user", Content: "deep_nested_call"},
				})
				if err != nil {
					t.Errorf("Goroutine %d nested call failed at iteration %d: %s", id, i, err.Error())
				}
				ctx.Release()
			}
		}(g)
	}

	for g := 0; g < concurrency; g++ {
		<-done
	}

	runtime.GC()
	time.Sleep(100 * time.Millisecond)
	var final runtime.MemStats
	runtime.ReadMemStats(&final)

	growth := int64(final.HeapAlloc) - int64(baseline.HeapAlloc)
	growthPerIteration := float64(growth) / float64(iterations)

	t.Logf("Memory Statistics (Concurrent Nested Calls):")
	t.Logf("  Iterations: %d, Concurrency: %d", iterations, concurrency)
	t.Logf("  Baseline: %.2f MB, Final: %.2f MB", float64(baseline.HeapAlloc)/1024/1024, float64(final.HeapAlloc)/1024/1024)
	t.Logf("  Growth per iteration: %.2f bytes", growthPerIteration)

	maxGrowthPerIteration := 25600.0
	if growthPerIteration > maxGrowthPerIteration {
		t.Errorf("Possible memory leak: %.2f bytes/iteration (threshold: %.2f)", growthPerIteration, maxGrowthPerIteration)
	}
}

func TestIsolateDisposal(t *testing.T) {
	testprepare.PrepareE2E(t)

	agent, err := assistant.Get("tests.create")
	require.NoError(t, err)
	require.NotNil(t, agent.HookScript)

	initialGoroutines := runtime.NumGoroutine()

	iterations := 100
	for i := 0; i < iterations; i++ {
		ctx := newMemTestContext("disposal-test", "tests.create")
		_, _, err := agent.HookScript.Create(ctx, []agentContext.Message{
			{Role: "user", Content: "Hello"},
		})
		require.NoError(t, err, "Create failed at iteration %d", i)
		ctx.Release()
	}

	time.Sleep(200 * time.Millisecond)
	runtime.GC()
	time.Sleep(200 * time.Millisecond)

	finalGoroutines := runtime.NumGoroutine()
	goroutineGrowth := finalGoroutines - initialGoroutines

	t.Logf("Goroutine Statistics:")
	t.Logf("  Initial: %d, Final: %d, Growth: %d", initialGoroutines, finalGoroutines, goroutineGrowth)

	maxGoroutineGrowthPerIteration := 5.0
	growthPerIteration := float64(goroutineGrowth) / float64(iterations)

	if growthPerIteration > maxGoroutineGrowthPerIteration {
		t.Errorf("Goroutine leak detected: %.2f goroutines per iteration (threshold: %.2f)",
			growthPerIteration, maxGoroutineGrowthPerIteration)
	} else {
		t.Logf("Goroutine growth acceptable: %.2f per iteration", growthPerIteration)
	}
}

// ============================================================================
// Helper Functions (stress-specific)
// ============================================================================

func newRealWorldContext(chatID, assistantID string) *agentContext.Context {
	authorized := &oauthTypes.AuthorizedInfo{
		Subject:    "realworld-test-user",
		ClientID:   "realworld-test-client",
		Scope:      "openid profile email",
		SessionID:  "realworld-test-session",
		UserID:     "realworld-user-123",
		TeamID:     "realworld-team-456",
		TenantID:   "realworld-tenant-789",
		RememberMe: true,
		Constraints: oauthTypes.DataConstraints{
			OwnerOnly:   false,
			CreatorOnly: false,
			EditorOnly:  false,
			TeamOnly:    true,
			Extra: map[string]interface{}{
				"department": "engineering",
				"region":     "us-west",
				"project":    "yao-realworld-test",
			},
		},
	}

	ctx := agentContext.New(stdContext.Background(), authorized, chatID)
	ctx.AssistantID = assistantID
	ctx.Locale = "en-us"
	ctx.Theme = "light"
	ctx.Client = agentContext.Client{
		Type:      "web",
		UserAgent: "RealWorldTest/1.0",
		IP:        "127.0.0.1",
	}
	ctx.Referer = agentContext.RefererAPI
	ctx.Accept = agentContext.AcceptWebCUI
	ctx.Route = ""
	ctx.Metadata = make(map[string]interface{})
	return ctx
}

func newLeakTestContext(chatID, assistantID string) *agentContext.Context {
	authorized := &oauthTypes.AuthorizedInfo{
		Subject:  "leak-test-user",
		ClientID: "leak-test-client",
		UserID:   "leak-user-123",
		TeamID:   "leak-team-456",
		TenantID: "leak-tenant-789",
		Constraints: oauthTypes.DataConstraints{
			TeamOnly: true,
			Extra: map[string]interface{}{
				"department": "testing",
			},
		},
	}

	ctx := agentContext.New(stdContext.Background(), authorized, chatID)
	ctx.AssistantID = assistantID
	ctx.Locale = "en-us"
	ctx.Theme = "light"
	ctx.Client = agentContext.Client{
		Type:      "web",
		UserAgent: "LeakTestAgent/1.0",
		IP:        "127.0.0.1",
	}
	ctx.Referer = agentContext.RefererAPI
	ctx.Accept = agentContext.AcceptWebCUI
	ctx.Route = ""
	ctx.Metadata = make(map[string]interface{})
	return ctx
}

func newMemTestContext(chatID, assistantID string) *agentContext.Context {
	authorized := &oauthTypes.AuthorizedInfo{
		Subject:  "mem-test-user",
		ClientID: "mem-test-client",
		UserID:   "mem-user-123",
		TeamID:   "mem-team-456",
		TenantID: "mem-tenant-789",
		Constraints: oauthTypes.DataConstraints{
			TeamOnly: true,
			Extra: map[string]interface{}{
				"department": "engineering",
			},
		},
	}

	ctx := agentContext.New(stdContext.Background(), authorized, chatID)
	ctx.AssistantID = assistantID
	ctx.Locale = "en-us"
	ctx.Theme = "light"
	ctx.Client = agentContext.Client{
		Type:      "web",
		UserAgent: "MemTestAgent/1.0",
		IP:        "127.0.0.1",
	}
	ctx.Referer = agentContext.RefererAPI
	ctx.Accept = agentContext.AcceptWebCUI
	ctx.Route = ""
	ctx.Metadata = make(map[string]interface{})
	return ctx
}

func stressGetMemStats() uint64 {
	runtime.GC()
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	return m.Alloc
}

func stressSaveGoroutineProfile(filename string) {
	f, err := os.Create(filename)
	if err != nil {
		return
	}
	defer f.Close()
	pprof.Lookup("goroutine").WriteTo(f, 2)
}

func stressAnalyzeGoroutineProfiles(t *testing.T, dir string) {
	initialData, err := os.ReadFile(filepath.Join(dir, "00_initial.txt"))
	if err != nil {
		t.Logf("Could not read initial profile: %v", err)
		return
	}

	finalData, err := os.ReadFile(filepath.Join(dir, "99_final.txt"))
	if err != nil {
		t.Logf("Could not read final profile: %v", err)
		return
	}

	initialFuncs := stressCountGoroutinesByFunction(string(initialData))
	finalFuncs := stressCountGoroutinesByFunction(string(finalData))

	t.Logf("Goroutine growth by function:")
	t.Logf("%-60s %8s %8s %8s", "Function", "Initial", "Final", "Growth")
	t.Logf("%s", strings.Repeat("-", 90))

	for fn, finalCount := range finalFuncs {
		initialCount := initialFuncs[fn]
		growth := finalCount - initialCount
		if growth > 0 {
			t.Logf("%-60s %8d %8d %8d", stressTruncate(fn, 60), initialCount, finalCount, growth)
		}
	}

	t.Logf("Profiles saved to: %s", dir)
}

func stressCountGoroutinesByFunction(profile string) map[string]int {
	counts := make(map[string]int)
	lines := strings.Split(profile, "\n")

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.Contains(line, "(") && !strings.HasPrefix(line, "#") {
			if idx := strings.Index(line, "("); idx > 0 {
				fn := strings.TrimSpace(line[:idx])
				counts[fn]++
			}
		}
	}

	return counts
}

func stressTruncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max-3] + "..."
}
