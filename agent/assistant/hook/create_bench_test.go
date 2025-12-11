package hook_test

import (
	stdContext "context"
	"testing"

	"github.com/yaoapp/yao/agent/assistant"
	"github.com/yaoapp/yao/agent/context"
	"github.com/yaoapp/yao/agent/testutils"
	"github.com/yaoapp/yao/openapi/oauth/types"
	"github.com/yaoapp/yao/test"
)

// ============================================================================
// Simple Scenario Benchmarks
// ============================================================================

// BenchmarkSimpleStandardMode benchmarks simple scenario in standard V8 mode
// Run with: go test -bench=BenchmarkSimpleStandardMode -benchmem -benchtime=100x
func BenchmarkSimpleStandardMode(b *testing.B) {
	testutils.Prepare(&testing.T{})
	defer testutils.Clean(&testing.T{})

	agent, err := assistant.Get("tests.create")
	if err != nil {
		b.Fatalf("Failed to get assistant: %s", err.Error())
	}

	if agent.HookScript == nil {
		b.Fatalf("Assistant has no script")
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ctx := newBenchContext("bench-simple-standard", "tests.create")
		_, _, err := agent.HookScript.Create(ctx, []context.Message{
			{Role: "user", Content: "Hello"},
		})
		if err != nil {
			b.Fatalf("Create failed: %s", err.Error())
		}
	}
}

// BenchmarkSimplePerformanceMode benchmarks simple scenario in performance V8 mode
// Run with: go test -bench=BenchmarkSimplePerformanceMode -benchmem -benchtime=100x
func BenchmarkSimplePerformanceMode(b *testing.B) {
	testutils.Prepare(&testing.T{}, test.PrepareOption{V8Mode: "performance"})
	defer testutils.Clean(&testing.T{})

	agent, err := assistant.Get("tests.create")
	if err != nil {
		b.Fatalf("Failed to get assistant: %s", err.Error())
	}

	if agent.HookScript == nil {
		b.Fatalf("Assistant has no script")
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ctx := newBenchContext("bench-simple-performance", "tests.create")
		_, _, err := agent.HookScript.Create(ctx, []context.Message{
			{Role: "user", Content: "Hello"},
		})
		if err != nil {
			b.Fatalf("Create failed: %s", err.Error())
		}
	}
}

// ============================================================================
// Business Scenario Benchmarks (with Process calls, DB access, etc.)
// ============================================================================

// BenchmarkBusinessStandardMode benchmarks business scenarios in standard V8 mode
// Run with: go test -bench=BenchmarkBusinessStandardMode -benchmem -benchtime=100x
func BenchmarkBusinessStandardMode(b *testing.B) {
	testutils.Prepare(&testing.T{})
	defer testutils.Clean(&testing.T{})

	agent, err := assistant.Get("tests.create")
	if err != nil {
		b.Fatalf("Failed to get assistant: %s", err.Error())
	}

	if agent.HookScript == nil {
		b.Fatalf("Assistant has no script")
	}

	scenarios := getBusinessScenarios()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		scenario := scenarios[i%len(scenarios)]
		ctx := newBenchContext("bench-business-standard", "tests.create")
		_, _, err := agent.HookScript.Create(ctx, []context.Message{
			{Role: "user", Content: scenario.content},
		})
		if err != nil {
			b.Errorf("%s failed: %s", scenario.name, err.Error())
		}
	}
}

// BenchmarkBusinessPerformanceMode benchmarks business scenarios in performance V8 mode
// Run with: go test -bench=BenchmarkBusinessPerformanceMode -benchmem -benchtime=100x
func BenchmarkBusinessPerformanceMode(b *testing.B) {
	testutils.Prepare(&testing.T{}, test.PrepareOption{V8Mode: "performance"})
	defer testutils.Clean(&testing.T{})

	agent, err := assistant.Get("tests.create")
	if err != nil {
		b.Fatalf("Failed to get assistant: %s", err.Error())
	}

	if agent.HookScript == nil {
		b.Fatalf("Assistant has no script")
	}

	scenarios := getBusinessScenarios()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		scenario := scenarios[i%len(scenarios)]
		ctx := newBenchContext("bench-business-performance", "tests.create")
		_, _, err := agent.HookScript.Create(ctx, []context.Message{
			{Role: "user", Content: scenario.content},
		})
		if err != nil {
			b.Errorf("%s failed: %s", scenario.name, err.Error())
		}
	}
}

// ============================================================================
// Concurrent Benchmarks
// ============================================================================

// BenchmarkConcurrentSimpleStandardMode benchmarks simple concurrent scenario in standard V8 mode
// Simulates concurrent users with isolate creation/disposal per request
// Run with: go test -bench=BenchmarkConcurrentSimpleStandardMode -benchmem -benchtime=100x
func BenchmarkConcurrentSimpleStandardMode(b *testing.B) {
	testutils.Prepare(&testing.T{})
	defer testutils.Clean(&testing.T{})

	agent, err := assistant.Get("tests.create")
	if err != nil {
		b.Fatalf("Failed to get assistant: %s", err.Error())
	}

	if agent.HookScript == nil {
		b.Fatalf("Assistant has no script")
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			ctx := newBenchContext("bench-concurrent-simple-standard", "tests.create")
			_, _, err := agent.HookScript.Create(ctx, []context.Message{
				{Role: "user", Content: "Hello"},
			})
			if err != nil {
				b.Errorf("Create failed (iteration %d): %s", i, err.Error())
			}
			i++
		}
	})
}

// BenchmarkConcurrentSimplePerformanceMode benchmarks simple concurrent scenario in performance V8 mode
// Simulates 100 users simultaneously using the system with isolate pool
// Run with: go test -bench=BenchmarkConcurrentSimplePerformanceMode -benchmem -benchtime=100x
func BenchmarkConcurrentSimplePerformanceMode(b *testing.B) {
	testutils.Prepare(&testing.T{}, test.PrepareOption{V8Mode: "performance"})
	defer testutils.Clean(&testing.T{})

	agent, err := assistant.Get("tests.create")
	if err != nil {
		b.Fatalf("Failed to get assistant: %s", err.Error())
	}

	if agent.HookScript == nil {
		b.Fatalf("Assistant has no script")
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			ctx := newBenchContext("bench-concurrent-simple", "tests.create")
			_, _, err := agent.HookScript.Create(ctx, []context.Message{
				{Role: "user", Content: "Hello"},
			})
			if err != nil {
				b.Errorf("Create failed (iteration %d): %s", i, err.Error())
			}
			i++
		}
	})
}

// BenchmarkConcurrentBusinessStandardMode benchmarks concurrent business scenarios in standard V8 mode
// Tests various scenarios with concurrent users and isolate creation/disposal per request
// Run with: go test -bench=BenchmarkConcurrentBusinessStandardMode -benchmem -benchtime=100x
func BenchmarkConcurrentBusinessStandardMode(b *testing.B) {
	testutils.Prepare(&testing.T{})
	defer testutils.Clean(&testing.T{})

	agent, err := assistant.Get("tests.create")
	if err != nil {
		b.Fatalf("Failed to get assistant: %s", err.Error())
	}

	if agent.HookScript == nil {
		b.Fatalf("Assistant has no script")
	}

	scenarios := getBusinessScenarios()

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			scenario := scenarios[i%len(scenarios)]
			ctx := newBenchContext("bench-concurrent-business-standard", "tests.create")
			_, _, err := agent.HookScript.Create(ctx, []context.Message{
				{Role: "user", Content: scenario.content},
			})
			if err != nil {
				b.Errorf("%s failed (iteration %d): %s", scenario.name, i, err.Error())
			}
			i++
		}
	})
}

// BenchmarkConcurrentBusinessPerformanceMode benchmarks concurrent business scenarios in performance V8 mode
// Tests various scenarios with 100 concurrent users with isolate pool
// Run with: go test -bench=BenchmarkConcurrentBusinessPerformanceMode -benchmem -benchtime=100x
func BenchmarkConcurrentBusinessPerformanceMode(b *testing.B) {
	testutils.Prepare(&testing.T{}, test.PrepareOption{V8Mode: "performance"})
	defer testutils.Clean(&testing.T{})

	agent, err := assistant.Get("tests.create")
	if err != nil {
		b.Fatalf("Failed to get assistant: %s", err.Error())
	}

	if agent.HookScript == nil {
		b.Fatalf("Assistant has no script")
	}

	scenarios := getBusinessScenarios()

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			scenario := scenarios[i%len(scenarios)]
			ctx := newBenchContext("bench-concurrent-business", "tests.create")
			_, _, err := agent.HookScript.Create(ctx, []context.Message{
				{Role: "user", Content: scenario.content},
			})
			if err != nil {
				b.Errorf("%s failed (iteration %d): %s", scenario.name, i, err.Error())
			}
			i++
		}
	})
}

// ============================================================================
// Helper Functions
// ============================================================================

// getBusinessScenarios returns the business test scenarios
func getBusinessScenarios() []struct {
	name    string
	content string
} {
	return []struct {
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
}

// newBenchContext creates a minimal context for benchmarking
func newBenchContext(chatID, assistantID string) *context.Context {
	authorized := &types.AuthorizedInfo{
		Subject:  "bench-user",
		ClientID: "bench-client",
		UserID:   "bench-user-123",
		TeamID:   "bench-team-456",
		TenantID: "bench-tenant-789",
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
		UserAgent: "BenchAgent/1.0",
		IP:        "127.0.0.1",
	}
	ctx.Referer = context.RefererAPI
	ctx.Accept = context.AcceptWebCUI
	ctx.Route = ""
	ctx.Metadata = make(map[string]interface{})
	return ctx
}
