//go:build integration

package hook_test

import (
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/yaoapp/yao/agent/assistant"
	agentContext "github.com/yaoapp/yao/agent/context"
	"github.com/yaoapp/yao/unit-test/agent/testprepare"
)

func TestNestedScriptCall(t *testing.T) {
	testprepare.PrepareSandbox(t)

	agent, err := assistant.Get("tests.hook-echo")
	require.NoError(t, err, "failed to get tests.hook-echo assistant")
	require.NotNil(t, agent.HookScript, "tests.hook-echo has no hook script")

	ctx := newTestContext("test-nested-call", "tests.hook-echo")

	res, _, err := agent.HookScript.Create(ctx, []agentContext.Message{
		{Role: "user", Content: "deep_nested_call"},
	})
	require.NoError(t, err, "nested call failed")
	require.NotNil(t, res, "expected non-nil response")
	assert.NotEmpty(t, res.Messages, "expected messages in response")

	t.Logf("Nested script call completed: %d messages", len(res.Messages))
	if res.Metadata != nil {
		t.Logf("Metadata: %+v", res.Metadata)
	}
}

func TestNestedScriptCallConcurrent(t *testing.T) {
	testprepare.PrepareSandbox(t)

	agent, err := assistant.Get("tests.hook-echo")
	require.NoError(t, err, "failed to get tests.hook-echo assistant")
	require.NotNil(t, agent.HookScript, "tests.hook-echo has no hook script")

	concurrency := 100
	iterations := 1

	var wg sync.WaitGroup
	errors := make(chan error, concurrency*iterations)

	t.Logf("Starting concurrent test: %d users x %d iterations = %d total calls",
		concurrency, iterations, concurrency*iterations)

	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func(userID int) {
			defer wg.Done()

			for j := 0; j < iterations; j++ {
				ctx := newTestContext("test-concurrent", "tests.hook-echo")

				_, _, err := agent.HookScript.Create(ctx, []agentContext.Message{
					{Role: "user", Content: "deep_nested_call"},
				})
				if err != nil {
					errors <- err
					return
				}
			}
		}(i)
	}

	wg.Wait()
	close(errors)

	errorCount := 0
	for err := range errors {
		errorCount++
		t.Errorf("Concurrent call failed: %s", err.Error())
	}

	require.Zero(t, errorCount, "failed with %d errors out of %d total calls", errorCount, concurrency*iterations)
	t.Logf("All %d concurrent nested calls completed successfully", concurrency*iterations)
}
