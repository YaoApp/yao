package hook_test

import (
	stdContext "context"
	"fmt"
	"runtime"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/yaoapp/yao/agent/assistant"
	"github.com/yaoapp/yao/agent/context"
	"github.com/yaoapp/yao/agent/testutils"
	"github.com/yaoapp/yao/openapi/oauth/types"
)

// ============================================================================
// Real World Stress Tests
// These tests simulate actual production usage patterns with Stream() flow
// ============================================================================

// TestRealWorldSimpleScenario tests basic Stream() flow with simple Create hook
func TestRealWorldSimpleScenario(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping real world test in short mode")
	}

	testutils.Prepare(t)
	defer testutils.Clean(t)

	agent, err := assistant.Get("tests.realworld")
	if err != nil {
		t.Fatalf("Failed to get assistant: %v", err)
	}

	ctx := newRealWorldContext("test-simple", "tests.realworld")

	// Test Create hook with simple scenario
	messages := []context.Message{
		{Role: "user", Content: "simple"},
	}

	response, _, err := agent.HookScript.Create(ctx, messages)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	assert.NotNil(t, response)
	assert.NotEmpty(t, response.Messages)
	assert.Equal(t, "simple", response.Metadata["scenario"])
}

// TestRealWorldMCPScenarios tests MCP integration scenarios
func TestRealWorldMCPScenarios(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping real world test in short mode")
	}

	testutils.Prepare(t)
	defer testutils.Clean(t)

	agent, err := assistant.Get("tests.realworld")
	if err != nil {
		t.Fatalf("Failed to get assistant: %v", err)
	}

	t.Run("MCP Health", func(t *testing.T) {
		ctx := newRealWorldContext("test-mcp-health", "tests.realworld")

		messages := []context.Message{
			{Role: "user", Content: "mcp_health"},
		}

		response, _, err := agent.HookScript.Create(ctx, messages)
		if err != nil {
			t.Fatalf("Create failed: %v", err)
		}

		// Detailed validation
		assert.NotNil(t, response)
		assert.NotEmpty(t, response.Messages)

		// Check if metadata exists
		if response.Metadata == nil {
			t.Logf("⚠ Metadata is nil - checking messages content")
			// Verify messages contain expected content
			messageContent := ""
			for _, msg := range response.Messages {
				if content, ok := msg.Content.(string); ok {
					messageContent += content + "\n"
				}
			}
			assert.Contains(t, messageContent, "Health", "Message should mention health")
			assert.Contains(t, messageContent, "Tools", "Message should mention tools")
			t.Logf("✓ MCP Health executed (verified via message content)")
		} else {
			assert.Equal(t, "mcp_health", response.Metadata["scenario"])

			// Verify metadata contains MCP results
			if toolsCount, ok := response.Metadata["tools_count"]; ok {
				count := int(toolsCount.(float64))
				assert.Greater(t, count, 0, "Should have tools from MCP")
				t.Logf("✓ MCP Health: %d tools, health data: %v",
					count, response.Metadata["health_data"])
			}
		}

		ctx.Release()
	})

	t.Run("MCP Tools", func(t *testing.T) {
		ctx := newRealWorldContext("test-mcp-tools", "tests.realworld")

		messages := []context.Message{
			{Role: "user", Content: "mcp_tools"},
		}

		response, _, err := agent.HookScript.Create(ctx, messages)
		if err != nil {
			t.Fatalf("Create failed: %v", err)
		}

		// Detailed validation
		assert.NotNil(t, response)
		assert.NotEmpty(t, response.Messages)

		// Check if metadata exists
		if response.Metadata == nil {
			t.Logf("⚠ Metadata is nil - checking messages content")
			// Verify messages contain expected content
			messageContent := ""
			for _, msg := range response.Messages {
				if content, ok := msg.Content.(string); ok {
					messageContent += content + "\n"
				}
			}
			assert.Contains(t, messageContent, "Tools", "Message should mention tools")
			assert.Contains(t, messageContent, "Ping", "Message should mention ping")
			t.Logf("✓ MCP Tools executed (verified via message content)")
		} else {
			assert.Equal(t, "mcp_tools", response.Metadata["scenario"])

			// Verify tools were called
			if toolsCount, ok := response.Metadata["tools_count"]; ok {
				count := int(toolsCount.(float64))
				assert.Greater(t, count, 0, "Should have tools from MCP")

				// Verify operations list
				if operations, ok := response.Metadata["operations"].([]interface{}); ok {
					assert.Len(t, operations, 2, "Should execute 2 operations: ping, status")
					t.Logf("✓ MCP Tools: %d tools, operations: %v", count, operations)
				}
			}
		}

		ctx.Release()
	})

	t.Run("Full Workflow", func(t *testing.T) {
		ctx := newRealWorldContext("test-full-workflow", "tests.realworld")

		// Initialize stack for trace
		stack, _, done := context.EnterStack(ctx, "tests.realworld", &context.Options{})
		defer done()
		ctx.Stack = stack

		messages := []context.Message{
			{Role: "user", Content: "full_workflow"},
		}

		response, _, err := agent.HookScript.Create(ctx, messages)
		if err != nil {
			t.Fatalf("Create failed: %v", err)
		}

		// Detailed validation
		assert.NotNil(t, response)
		assert.NotEmpty(t, response.Messages)

		// Check if metadata exists
		if response.Metadata == nil {
			t.Logf("⚠ Metadata is nil - checking messages content")
			// Verify messages contain expected content
			messageContent := ""
			for _, msg := range response.Messages {
				if content, ok := msg.Content.(string); ok {
					messageContent += content + "\n"
				}
			}
			assert.Contains(t, messageContent, "Workflow", "Message should mention workflow")
			assert.Contains(t, messageContent, "Tools", "Message should mention tools")
			assert.Contains(t, messageContent, "Roles", "Message should mention database roles")
			t.Logf("✓ Full Workflow executed (verified via message content)")
		} else {
			assert.Equal(t, "full_workflow", response.Metadata["scenario"])

			// Verify all phases completed
			if phasesCompleted, ok := response.Metadata["phases_completed"]; ok {
				phases := int(phasesCompleted.(float64))
				assert.Equal(t, 4, phases, "Should complete 4 phases")

				// Verify MCP tools
				if mcpTools, ok := response.Metadata["mcp_tools"]; ok {
					tools := int(mcpTools.(float64))
					assert.Greater(t, tools, 0, "Should have MCP tools")

					// Verify DB records
					if dbRecords, ok := response.Metadata["db_records"]; ok {
						records := int(dbRecords.(float64))
						assert.GreaterOrEqual(t, records, 0, "Should have DB query result")

						t.Logf("✓ Full Workflow: %d phases, %d MCP tools, %d DB records",
							phases, tools, records)
					}
				}
			}
		}

		ctx.Release()
	})
}

// TestRealWorldTraceIntensive tests trace-heavy scenarios
func TestRealWorldTraceIntensive(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping real world test in short mode")
	}

	testutils.Prepare(t)
	defer testutils.Clean(t)

	agent, err := assistant.Get("tests.realworld")
	if err != nil {
		t.Fatalf("Failed to get assistant: %v", err)
	}

	ctx := newRealWorldContext("test-trace-intensive", "tests.realworld")
	stack, _, done := context.EnterStack(ctx, "tests.realworld", &context.Options{})
	defer done()
	ctx.Stack = stack

	messages := []context.Message{
		{Role: "user", Content: "trace_intensive"},
	}

	response, _, err := agent.HookScript.Create(ctx, messages)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	assert.NotNil(t, response)
	assert.Equal(t, "trace_intensive", response.Metadata["scenario"])
	assert.NotZero(t, response.Metadata["nodes_created"])
}

// TestRealWorldStressSimple tests simple scenario under stress
func TestRealWorldStressSimple(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping stress test in short mode")
	}

	testutils.Prepare(t)
	defer testutils.Clean(t)

	agent, err := assistant.Get("tests.realworld")
	if err != nil {
		t.Fatalf("Failed to get assistant: %v", err)
	}

	iterations := 100
	startMemory := getMemStats()

	for i := 0; i < iterations; i++ {
		ctx := newRealWorldContext(fmt.Sprintf("stress-simple-%d", i), "tests.realworld")

		messages := []context.Message{
			{Role: "user", Content: "simple"},
		}

		response, _, err := agent.HookScript.Create(ctx, messages)
		if err != nil {
			t.Fatalf("Iteration %d failed: %v", i, err)
		}

		// Validate response
		assert.NotNil(t, response, "Iteration %d: response should not be nil", i)
		assert.NotEmpty(t, response.Messages, "Iteration %d: messages should not be empty", i)
		if response.Metadata != nil {
			assert.Equal(t, "simple", response.Metadata["scenario"], "Iteration %d: scenario mismatch", i)
		}

		// Explicit cleanup
		ctx.Release()

		if i%20 == 0 {
			runtime.GC()
			currentMemory := getMemStats()
			t.Logf("Iteration %d: Memory: %d MB", i, currentMemory/1024/1024)
		}
	}

	runtime.GC()
	endMemory := getMemStats()

	t.Logf("Simple stress: %d iterations", iterations)
	t.Logf("Start memory: %d MB", startMemory/1024/1024)
	t.Logf("End memory: %d MB", endMemory/1024/1024)

	if endMemory > startMemory {
		t.Logf("Memory growth: %d MB", (endMemory-startMemory)/1024/1024)
	} else {
		t.Logf("Memory decreased: %d MB", (startMemory-endMemory)/1024/1024)
	}
}

// TestRealWorldStressMCP tests MCP scenarios under stress
func TestRealWorldStressMCP(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping stress test in short mode")
	}

	testutils.Prepare(t)
	defer testutils.Clean(t)

	agent, err := assistant.Get("tests.realworld")
	if err != nil {
		t.Fatalf("Failed to get assistant: %v", err)
	}

	iterations := 50
	scenarios := []string{"mcp_health", "mcp_tools"}
	startMemory := getMemStats()

	for i := 0; i < iterations; i++ {
		scenario := scenarios[i%len(scenarios)]
		ctx := newRealWorldContext(fmt.Sprintf("stress-mcp-%d", i), "tests.realworld")

		// Initialize stack for trace
		stack, _, done := context.EnterStack(ctx, "tests.realworld", &context.Options{})
		ctx.Stack = stack

		messages := []context.Message{
			{Role: "user", Content: scenario},
		}

		response, _, err := agent.HookScript.Create(ctx, messages)
		if err != nil {
			t.Fatalf("Iteration %d (%s) failed: %v", i, scenario, err)
		}

		// Validate response
		assert.NotNil(t, response, "Iteration %d (%s): response should not be nil", i, scenario)
		assert.NotEmpty(t, response.Messages, "Iteration %d (%s): messages should not be empty", i, scenario)

		// Validate metadata
		if response.Metadata != nil {
			assert.Equal(t, scenario, response.Metadata["scenario"], "Iteration %d: scenario mismatch", i)

			// Verify MCP-specific data
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
					assert.Len(t, operations, 2, "Iteration %d: should have 2 operations (ping, status)", i)
				}
			}
		} else {
			t.Errorf("Iteration %d (%s): metadata is nil", i, scenario)
		}

		// Cleanup
		done()
		ctx.Release()

		if i%10 == 0 {
			runtime.GC()
			currentMemory := getMemStats()
			t.Logf("Iteration %d (%s): Memory: %d MB", i, scenario, currentMemory/1024/1024)
		}
	}

	runtime.GC()
	endMemory := getMemStats()

	t.Logf("MCP stress: %d iterations", iterations)
	t.Logf("Start memory: %d MB", startMemory/1024/1024)
	t.Logf("End memory: %d MB", endMemory/1024/1024)

	if endMemory > startMemory {
		t.Logf("Memory growth: %d MB", (endMemory-startMemory)/1024/1024)
	} else {
		t.Logf("Memory decreased: %d MB", (startMemory-endMemory)/1024/1024)
	}
}

// TestRealWorldStressFullWorkflow tests complete workflow under stress
func TestRealWorldStressFullWorkflow(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping stress test in short mode")
	}

	testutils.Prepare(t)
	defer testutils.Clean(t)

	agent, err := assistant.Get("tests.realworld")
	if err != nil {
		t.Fatalf("Failed to get assistant: %v", err)
	}

	iterations := 30
	startMemory := getMemStats()
	startTime := time.Now()

	for i := 0; i < iterations; i++ {
		ctx := newRealWorldContext(fmt.Sprintf("stress-workflow-%d", i), "tests.realworld")

		// Initialize stack for trace
		stack, _, done := context.EnterStack(ctx, "tests.realworld", &context.Options{})
		ctx.Stack = stack

		messages := []context.Message{
			{Role: "user", Content: "full_workflow"},
		}

		response, _, err := agent.HookScript.Create(ctx, messages)
		if err != nil {
			t.Fatalf("Iteration %d failed: %v", i, err)
		}

		// Verify response
		assert.NotNil(t, response, "Iteration %d: response should not be nil", i)
		assert.NotEmpty(t, response.Messages, "Iteration %d: messages should not be empty", i)
		if response.Metadata != nil {
			assert.Equal(t, "full_workflow", response.Metadata["scenario"], "Iteration %d: scenario mismatch", i)
			// Verify workflow-specific metadata
			if phasesCompleted, ok := response.Metadata["phases_completed"]; ok {
				phases := int(phasesCompleted.(float64))
				assert.Equal(t, 4, phases, "Iteration %d: should complete 4 phases", i)
			}
			if mcpTools, ok := response.Metadata["mcp_tools"]; ok {
				tools := int(mcpTools.(float64))
				assert.Greater(t, tools, 0, "Iteration %d: should have MCP tools", i)
			}
		}

		// Cleanup
		done()
		ctx.Release()

		if i%10 == 0 {
			runtime.GC()
			currentMemory := getMemStats()
			elapsed := time.Since(startTime)
			t.Logf("Iteration %d: Memory: %d MB, Elapsed: %v", i, currentMemory/1024/1024, elapsed)
		}
	}

	duration := time.Since(startTime)
	runtime.GC()
	endMemory := getMemStats()

	avgTime := duration / time.Duration(iterations)
	t.Logf("Full workflow stress: %d iterations", iterations)
	t.Logf("Total time: %v", duration)
	t.Logf("Average time per iteration: %v", avgTime)
	t.Logf("Start memory: %d MB", startMemory/1024/1024)
	t.Logf("End memory: %d MB", endMemory/1024/1024)

	if endMemory > startMemory {
		t.Logf("Memory growth: %d MB", (endMemory-startMemory)/1024/1024)
	} else {
		t.Logf("Memory decreased: %d MB", (startMemory-endMemory)/1024/1024)
	}
}

// TestRealWorldStressConcurrent tests concurrent real-world usage
func TestRealWorldStressConcurrent(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping stress test in short mode")
	}

	testutils.Prepare(t)
	defer testutils.Clean(t)

	agent, err := assistant.Get("tests.realworld")
	if err != nil {
		t.Fatalf("Failed to get assistant: %v", err)
	}

	goroutines := 100
	iterationsPerGoroutine := 10
	scenarios := []string{"simple", "mcp_health", "mcp_tools", "full_workflow"}

	startMemory := getMemStats()
	startTime := time.Now()

	var wg sync.WaitGroup
	errors := make(chan error, goroutines*iterationsPerGoroutine)

	// Track results for validation
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

				// Initialize stack for trace
				stack, _, done := context.EnterStack(ctx, "tests.realworld", &context.Options{})
				ctx.Stack = stack

				messages := []context.Message{
					{Role: "user", Content: scenario},
				}

				response, _, err := agent.HookScript.Create(ctx, messages)
				if err != nil {
					errors <- fmt.Errorf("goroutine %d iteration %d (%s): %v", goroutineID, i, scenario, err)
					done()
					ctx.Release()
					return
				}

				// Validate response
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

				// Collect result
				results <- Result{
					goroutineID: goroutineID,
					iteration:   i,
					scenario:    scenario,
					metadata:    response.Metadata,
				}

				// Cleanup
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
	endMemory := getMemStats()

	// Check for errors
	errorCount := 0
	for err := range errors {
		t.Error(err)
		errorCount++
	}

	assert.Equal(t, 0, errorCount, "No errors should occur in concurrent operations")

	// Validate results
	scenarioCounts := make(map[string]int)
	validResults := 0

	for result := range results {
		validResults++
		scenarioCounts[result.scenario]++

		// Validate metadata exists and has expected scenario
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

	t.Logf("✓ Concurrent stress: %d operations (goroutines: %d, iterations: %d)",
		totalOperations, goroutines, iterationsPerGoroutine)
	t.Logf("✓ Valid results: %d/%d (100%%)", validResults, totalOperations)
	t.Logf("✓ Scenario distribution:")
	for scenario, count := range scenarioCounts {
		t.Logf("  - %s: %d operations", scenario, count)
	}
	t.Logf("✓ Total time: %v", duration)
	t.Logf("✓ Average time per operation: %v", avgTime)
	t.Logf("✓ Start memory: %d MB", startMemory/1024/1024)
	t.Logf("✓ End memory: %d MB", endMemory/1024/1024)

	if endMemory > startMemory {
		t.Logf("✓ Memory growth: %d MB", (endMemory-startMemory)/1024/1024)
	} else {
		t.Logf("✓ Memory decreased: %d MB", (startMemory-endMemory)/1024/1024)
	}
}

// TestRealWorldStressResourceHeavy tests resource-intensive scenarios
func TestRealWorldStressResourceHeavy(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping stress test in short mode")
	}

	testutils.Prepare(t)
	defer testutils.Clean(t)

	agent, err := assistant.Get("tests.realworld")
	if err != nil {
		t.Fatalf("Failed to get assistant: %v", err)
	}

	iterations := 20
	startMemory := getMemStats()
	startTime := time.Now()

	for i := 0; i < iterations; i++ {
		ctx := newRealWorldContext(fmt.Sprintf("stress-heavy-%d", i), "tests.realworld")

		// Initialize stack for trace
		stack, _, done := context.EnterStack(ctx, "tests.realworld", &context.Options{})
		ctx.Stack = stack

		messages := []context.Message{
			{Role: "user", Content: "resource_heavy"},
		}

		response, _, err := agent.HookScript.Create(ctx, messages)
		if err != nil {
			t.Fatalf("Iteration %d failed: %v", i, err)
		}

		// Validate response
		assert.NotNil(t, response, "Iteration %d: response should not be nil", i)
		assert.NotEmpty(t, response.Messages, "Iteration %d: messages should not be empty", i)
		if response.Metadata != nil {
			assert.Equal(t, "resource_heavy", response.Metadata["scenario"], "Iteration %d: scenario mismatch", i)
			// Verify resource-heavy metadata
			if mcpIterations, ok := response.Metadata["mcp_iterations"]; ok {
				iterations := int(mcpIterations.(float64))
				assert.Equal(t, 5, iterations, "Iteration %d: should have 5 MCP iterations", i)
			}
		}

		// Cleanup
		done()
		ctx.Release()

		if i%5 == 0 {
			runtime.GC()
			currentMemory := getMemStats()
			elapsed := time.Since(startTime)
			t.Logf("Iteration %d: Memory: %d MB, Elapsed: %v", i, currentMemory/1024/1024, elapsed)
		}
	}

	duration := time.Since(startTime)
	runtime.GC()
	endMemory := getMemStats()

	avgTime := duration / time.Duration(iterations)
	t.Logf("Resource heavy stress: %d iterations", iterations)
	t.Logf("Total time: %v", duration)
	t.Logf("Average time per iteration: %v", avgTime)
	t.Logf("Start memory: %d MB", startMemory/1024/1024)
	t.Logf("End memory: %d MB", endMemory/1024/1024)

	if endMemory > startMemory {
		memoryGrowth := int64(endMemory - startMemory)
		t.Logf("Memory growth: %d MB", memoryGrowth/1024/1024)
		// Allow up to 100MB growth for resource-heavy operations
		assert.Less(t, memoryGrowth, int64(100*1024*1024), "Memory growth should be reasonable")
	} else {
		t.Logf("Memory decreased: %d MB", (startMemory-endMemory)/1024/1024)
	}
}

// ============================================================================
// Helper Functions
// ============================================================================

// newRealWorldContext creates a Context for real-world testing
func newRealWorldContext(chatID, assistantID string) *context.Context {
	authorized := &types.AuthorizedInfo{
		Subject:    "realworld-test-user",
		ClientID:   "realworld-test-client",
		Scope:      "openid profile email",
		SessionID:  "realworld-test-session",
		UserID:     "realworld-user-123",
		TeamID:     "realworld-team-456",
		TenantID:   "realworld-tenant-789",
		RememberMe: true,
		Constraints: types.DataConstraints{
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

	ctx := context.New(stdContext.Background(), authorized, chatID)
	ctx.AssistantID = assistantID
	ctx.Locale = "en-us"
	ctx.Theme = "light"
	ctx.Client = context.Client{
		Type:      "web",
		UserAgent: "RealWorldTest/1.0",
		IP:        "127.0.0.1",
	}
	ctx.Referer = context.RefererAPI
	ctx.Accept = context.AcceptWebCUI
	ctx.Route = ""
	ctx.Metadata = make(map[string]interface{})
	return ctx
}

// getMemStats returns current memory allocation in bytes
func getMemStats() uint64 {
	runtime.GC()
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	return m.Alloc
}
