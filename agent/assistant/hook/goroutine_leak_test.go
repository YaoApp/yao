package hook_test

import (
	stdContext "context"
	"fmt"
	"os"
	"runtime"
	"runtime/pprof"
	"strings"
	"testing"
	"time"

	"github.com/yaoapp/yao/agent/assistant"
	"github.com/yaoapp/yao/agent/context"
	"github.com/yaoapp/yao/agent/testutils"
	"github.com/yaoapp/yao/openapi/oauth/types"
)

// TestGoroutineLeakDetailed performs detailed goroutine leak analysis
func TestGoroutineLeakDetailed(t *testing.T) {
	testutils.Prepare(t)
	defer testutils.Clean(t)

	agent, err := assistant.Get("tests.create")
	if err != nil {
		t.Fatalf("Failed to get assistant: %s", err.Error())
	}

	if agent.HookScript == nil {
		t.Fatalf("Assistant has no script")
	}

	// Create profile directory
	os.MkdirAll("/tmp/goroutine_profiles", 0755)

	// Take initial snapshot
	runtime.GC()
	time.Sleep(200 * time.Millisecond)
	initialGoroutines := runtime.NumGoroutine()

	// Save initial profile
	saveGoroutineProfile("/tmp/goroutine_profiles/00_initial.txt")
	t.Logf("Initial goroutines: %d", initialGoroutines)

	// Test with just 10 iterations to see the pattern
	iterations := 10
	for i := 0; i < iterations; i++ {
		ctx := newLeakTestContext(fmt.Sprintf("leak-test-%d", i), "tests.create")

		_, _, err := agent.HookScript.Create(ctx, []context.Message{
			{Role: "user", Content: "Hello"},
		})
		if err != nil {
			t.Errorf("Create failed at iteration %d: %s", i, err.Error())
		}

		// Release context
		ctx.Release()

		// Check goroutines after each iteration
		current := runtime.NumGoroutine()
		growth := current - initialGoroutines
		t.Logf("After iteration %d: %d goroutines (growth: %d)", i+1, current, growth)

		// Save profile every 5 iterations
		if (i+1)%5 == 0 {
			saveGoroutineProfile(fmt.Sprintf("/tmp/goroutine_profiles/%02d_after_iter_%d.txt", i+1, i+1))
		}
	}

	// Force cleanup
	runtime.GC()
	time.Sleep(500 * time.Millisecond)

	finalGoroutines := runtime.NumGoroutine()
	growth := finalGoroutines - initialGoroutines

	t.Logf("\n=== SUMMARY ===")
	t.Logf("Initial:  %d goroutines", initialGoroutines)
	t.Logf("Final:    %d goroutines", finalGoroutines)
	t.Logf("Growth:   %d goroutines (%.2f per iteration)", growth, float64(growth)/float64(iterations))

	// Save final profile
	saveGoroutineProfile("/tmp/goroutine_profiles/99_final.txt")

	// Analyze the leak
	t.Logf("\n=== ANALYSIS ===")
	analyzeGoroutineProfiles(t, "/tmp/goroutine_profiles")
}

// TestGoroutineLeakByComponent tests each component separately
func TestGoroutineLeakByComponent(t *testing.T) {
	testutils.Prepare(t)
	defer testutils.Clean(t)

	agent, err := assistant.Get("tests.create")
	if err != nil {
		t.Fatalf("Failed to get assistant: %s", err.Error())
	}

	os.MkdirAll("/tmp/component_profiles", 0755)

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
			_, _, _ = agent.HookScript.Create(ctx, []context.Message{
				{Role: "user", Content: "Hello"},
			})
			ctx.Release()
		}

		runtime.GC()
		time.Sleep(100 * time.Millisecond)
		final := runtime.NumGoroutine()

		t.Logf("Script execution: initial=%d, final=%d, growth=%d", initial, final, final-initial)
		saveGoroutineProfile("/tmp/component_profiles/script_execution.txt")
	})

	t.Run("TraceOperations", func(t *testing.T) {
		runtime.GC()
		time.Sleep(100 * time.Millisecond)
		initial := runtime.NumGoroutine()

		for i := 0; i < 10; i++ {
			ctx := newLeakTestContext(fmt.Sprintf("test-%d", i), "tests.create")

			// Create trace
			trace, err := ctx.Trace()
			if err == nil && trace != nil {
				// Trace operations
				_ = trace
			}

			ctx.Release()
		}

		runtime.GC()
		time.Sleep(100 * time.Millisecond)
		final := runtime.NumGoroutine()

		t.Logf("Trace operations: initial=%d, final=%d, growth=%d", initial, final, final-initial)
		saveGoroutineProfile("/tmp/component_profiles/trace_operations.txt")
	})
}

// TestGoroutineLeakWithoutRelease tests if Release() fixes the leak
func TestGoroutineLeakWithoutRelease(t *testing.T) {
	testutils.Prepare(t)
	defer testutils.Clean(t)

	agent, err := assistant.Get("tests.create")
	if err != nil {
		t.Fatalf("Failed to get assistant: %s", err.Error())
	}

	t.Run("WithoutRelease", func(t *testing.T) {
		runtime.GC()
		time.Sleep(100 * time.Millisecond)
		initial := runtime.NumGoroutine()

		for i := 0; i < 10; i++ {
			ctx := newLeakTestContext(fmt.Sprintf("no-release-%d", i), "tests.create")
			_, _, _ = agent.HookScript.Create(ctx, []context.Message{
				{Role: "user", Content: "Hello"},
			})
			// Intentionally NOT calling ctx.Release()
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
			_, _, _ = agent.HookScript.Create(ctx, []context.Message{
				{Role: "user", Content: "Hello"},
			})
			ctx.Release() // WITH Release
		}

		runtime.GC()
		time.Sleep(100 * time.Millisecond)
		final := runtime.NumGoroutine()

		t.Logf("WITH Release: initial=%d, final=%d, growth=%d (%.1f per iter)",
			initial, final, final-initial, float64(final-initial)/10.0)
	})
}

// Helper functions

func saveGoroutineProfile(filename string) {
	f, err := os.Create(filename)
	if err != nil {
		return
	}
	defer f.Close()

	pprof.Lookup("goroutine").WriteTo(f, 2) // detail level 2
}

func analyzeGoroutineProfiles(t *testing.T, dir string) {
	// Read initial and final profiles
	initialData, err := os.ReadFile(dir + "/00_initial.txt")
	if err != nil {
		t.Logf("Could not read initial profile: %v", err)
		return
	}

	finalData, err := os.ReadFile(dir + "/99_final.txt")
	if err != nil {
		t.Logf("Could not read final profile: %v", err)
		return
	}

	// Count goroutines by function
	initialFuncs := countGoroutinesByFunction(string(initialData))
	finalFuncs := countGoroutinesByFunction(string(finalData))

	t.Logf("\nGoroutine growth by function:")
	t.Logf("%-60s %8s %8s %8s", "Function", "Initial", "Final", "Growth")
	t.Logf("%s", strings.Repeat("-", 90))

	// Find functions that grew
	for fn, finalCount := range finalFuncs {
		initialCount := initialFuncs[fn]
		growth := finalCount - initialCount
		if growth > 0 {
			t.Logf("%-60s %8d %8d %8d", truncate(fn, 60), initialCount, finalCount, growth)
		}
	}

	t.Logf("\nProfiles saved to: %s", dir)
	t.Logf("To compare: diff %s/00_initial.txt %s/99_final.txt | grep '^>'", dir, dir)
}

func countGoroutinesByFunction(profile string) map[string]int {
	counts := make(map[string]int)
	lines := strings.Split(profile, "\n")

	for _, line := range lines {
		line = strings.TrimSpace(line)
		// Look for function names in goroutine stack traces
		if strings.Contains(line, "(") && !strings.HasPrefix(line, "#") {
			// Extract function name
			if idx := strings.Index(line, "("); idx > 0 {
				fn := strings.TrimSpace(line[:idx])
				counts[fn]++
			}
		}
	}

	return counts
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max-3] + "..."
}

func newLeakTestContext(chatID, assistantID string) *context.Context {
	authorized := &types.AuthorizedInfo{
		Subject:  "leak-test-user",
		ClientID: "leak-test-client",
		UserID:   "leak-user-123",
		TeamID:   "leak-team-456",
		TenantID: "leak-tenant-789",
		Constraints: types.DataConstraints{
			TeamOnly: true,
			Extra: map[string]interface{}{
				"department": "testing",
			},
		},
	}

	ctx := context.New(stdContext.Background(), authorized, chatID)
	ctx.AssistantID = assistantID
	ctx.Locale = "en-us"
	ctx.Theme = "light"
	ctx.Client = context.Client{
		Type:      "web",
		UserAgent: "LeakTestAgent/1.0",
		IP:        "127.0.0.1",
	}
	ctx.Referer = context.RefererAPI
	ctx.Accept = context.AcceptWebCUI
	ctx.Route = ""
	ctx.Metadata = make(map[string]interface{})
	return ctx
}
