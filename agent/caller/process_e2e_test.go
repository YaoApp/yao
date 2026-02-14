package caller_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/yaoapp/gou/process"
	"github.com/yaoapp/yao/agent/caller"
	"github.com/yaoapp/yao/agent/testutils"
)

// newLLMProcess creates a process.Process with a 120s outer timeout for LLM calls.
// agent.Call has its own internal default timeout (DefaultProcessTimeout = 600s),
// but the outer context (120s) takes precedence via context.WithTimeout chaining.
func newLLMProcess(t *testing.T, name string, args ...interface{}) *process.Process {
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	t.Cleanup(cancel)
	return process.NewWithContext(ctx, name, args...)
}

// ============================================================================
// A. Pure LLM scenarios (tests.simple-greeting — no hooks)
// ============================================================================

func TestProcessCall_LLM_Basic(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping: requires real LLM (source env.local.sh)")
	}
	testutils.Prepare(t)
	defer testutils.Clean(t)

	proc := newLLMProcess(t, "agent.call", map[string]interface{}{
		"assistant_id": "tests.simple-greeting",
		"messages": []interface{}{
			map[string]interface{}{"role": "user", "content": "Hello!"},
		},
	})

	err := proc.Execute()
	require.NoError(t, err)

	val := proc.Value()
	require.NotNil(t, val, "process should return a value")

	result, ok := val.(*caller.Result)
	require.True(t, ok, "value should be *caller.Result, got %T", val)

	assert.Equal(t, "tests.simple-greeting", result.AgentID)
	assert.Empty(t, result.Error, "should not have error")
	assert.NotEmpty(t, result.Content, "should have LLM content")
	assert.NotNil(t, result.Response, "should have full response")
	t.Logf("LLM response: %s", result.Content)
}

func TestProcessCall_LLM_MultipleMessages(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping: requires real LLM")
	}
	testutils.Prepare(t)
	defer testutils.Clean(t)

	proc := newLLMProcess(t, "agent.call", map[string]interface{}{
		"assistant_id": "tests.simple-greeting",
		"messages": []interface{}{
			map[string]interface{}{"role": "system", "content": "Always reply in JSON format."},
			map[string]interface{}{"role": "user", "content": "Say hello"},
		},
	})

	err := proc.Execute()
	require.NoError(t, err)

	result, ok := proc.Value().(*caller.Result)
	require.True(t, ok)
	assert.Empty(t, result.Error)
	assert.NotEmpty(t, result.Content)
	t.Logf("Multi-message response: %s", result.Content)
}

func TestProcessCall_LLM_WithMetadata(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping: requires real LLM")
	}
	testutils.Prepare(t)
	defer testutils.Clean(t)

	proc := newLLMProcess(t, "agent.call", map[string]interface{}{
		"assistant_id": "tests.simple-greeting",
		"messages": []interface{}{
			map[string]interface{}{"role": "user", "content": "Hi there!"},
		},
		"metadata": map[string]interface{}{"source": "e2e-test", "mode": "task"},
		"locale":   "zh-CN",
		"route":    "/test/e2e",
		"chat_id":  "e2e-test-chat-001",
	})

	err := proc.Execute()
	require.NoError(t, err)

	result, ok := proc.Value().(*caller.Result)
	require.True(t, ok)
	assert.Empty(t, result.Error)
	assert.NotEmpty(t, result.Content)
	t.Logf("With-metadata response: %s", result.Content)
}

func TestProcessCall_LLM_SkipOutputForced(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping: requires real LLM")
	}
	testutils.Prepare(t)
	defer testutils.Clean(t)

	// Explicitly pass skip.output=false — headless context MUST force it to true
	// If the force logic fails, this would panic (nil Writer).
	proc := newLLMProcess(t, "agent.call", map[string]interface{}{
		"assistant_id": "tests.simple-greeting",
		"messages": []interface{}{
			map[string]interface{}{"role": "user", "content": "Hello!"},
		},
		"skip": map[string]interface{}{
			"output":  false,
			"history": false,
		},
	})

	err := proc.Execute()
	require.NoError(t, err, "should NOT panic even with skip.output=false — headless forces true")

	result, ok := proc.Value().(*caller.Result)
	require.True(t, ok)
	assert.Empty(t, result.Error)
	assert.NotEmpty(t, result.Content)
}

// ============================================================================
// B. Create Hook scenarios (tests.create)
// ============================================================================

func TestProcessCall_CreateHook_Default(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping: requires real LLM")
	}
	testutils.Prepare(t)
	defer testutils.Clean(t)

	// Send a generic message — Create Hook routes to scenarioDefault
	proc := newLLMProcess(t, "agent.call", map[string]interface{}{
		"assistant_id": "tests.create",
		"messages": []interface{}{
			map[string]interface{}{"role": "user", "content": "hello world"},
		},
	})

	err := proc.Execute()
	require.NoError(t, err)

	result, ok := proc.Value().(*caller.Result)
	require.True(t, ok)
	assert.Equal(t, "tests.create", result.AgentID)
	assert.Empty(t, result.Error)
	assert.NotEmpty(t, result.Content, "Create Hook should still produce LLM response")
	t.Logf("CreateHook default response: %s", result.Content)
}

func TestProcessCall_CreateHook_ReturnFull(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping: requires real LLM")
	}
	testutils.Prepare(t)
	defer testutils.Clean(t)

	// Send "return_full" — Create Hook returns full HookCreateResponse with custom messages
	proc := newLLMProcess(t, "agent.call", map[string]interface{}{
		"assistant_id": "tests.create",
		"messages": []interface{}{
			map[string]interface{}{"role": "user", "content": "return_full"},
		},
	})

	err := proc.Execute()
	require.NoError(t, err)

	result, ok := proc.Value().(*caller.Result)
	require.True(t, ok)
	assert.Equal(t, "tests.create", result.AgentID)
	assert.Empty(t, result.Error)
	// The Create Hook overrides messages with system + user, then LLM responds
	assert.NotEmpty(t, result.Content)
	t.Logf("CreateHook return_full response: %s", result.Content)
}

// ============================================================================
// C. Next Hook scenarios (tests.next)
// ============================================================================

func TestProcessCall_NextHook_Standard(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping: requires real LLM")
	}
	testutils.Prepare(t)
	defer testutils.Clean(t)

	// Send "standard" — Next Hook returns null, standard LLM response is used
	proc := newLLMProcess(t, "agent.call", map[string]interface{}{
		"assistant_id": "tests.next",
		"messages": []interface{}{
			map[string]interface{}{"role": "user", "content": "standard"},
		},
	})

	err := proc.Execute()
	require.NoError(t, err)

	result, ok := proc.Value().(*caller.Result)
	require.True(t, ok)
	assert.Equal(t, "tests.next", result.AgentID)
	assert.Empty(t, result.Error)
	assert.NotEmpty(t, result.Content, "standard scenario should return LLM content")
	t.Logf("NextHook standard response: %s", result.Content)
}

func TestProcessCall_NextHook_CustomData(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping: requires real LLM")
	}
	testutils.Prepare(t)
	defer testutils.Clean(t)

	// Send "return_custom_data" — Next Hook returns custom data
	proc := newLLMProcess(t, "agent.call", map[string]interface{}{
		"assistant_id": "tests.next",
		"messages": []interface{}{
			map[string]interface{}{"role": "user", "content": "return_custom_data"},
		},
	})

	err := proc.Execute()
	require.NoError(t, err)

	result, ok := proc.Value().(*caller.Result)
	require.True(t, ok)
	assert.Equal(t, "tests.next", result.AgentID)
	assert.Empty(t, result.Error)
	assert.NotNil(t, result.Response, "should have response")

	// Next Hook custom data is available in response.Next
	if result.Response != nil && result.Response.Next != nil {
		t.Logf("NextHook custom data: %+v", result.Response.Next)
		nextMap, ok := result.Response.Next.(map[string]interface{})
		if ok {
			// The Next Hook returns { data: { message, test, timestamp } }
			if dataMap, ok := nextMap["data"].(map[string]interface{}); ok {
				assert.Equal(t, "Custom response from Next Hook", dataMap["message"])
				assert.Equal(t, true, dataMap["test"])
			}
		}
	}
}

// ============================================================================
// D. Timeout scenarios
// ============================================================================

func TestProcessCall_Timeout_Short(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping: requires real LLM (source env.local.sh)")
	}
	testutils.Prepare(t)
	defer testutils.Clean(t)

	// Set timeout=2 seconds — LLM round-trip will certainly exceed this.
	// Verifies that the timeout parameter is respected and produces an error.
	proc := newLLMProcess(t, "agent.call", map[string]interface{}{
		"assistant_id": "tests.simple-greeting",
		"messages": []interface{}{
			map[string]interface{}{"role": "user", "content": "Tell me a very long story about the history of computing."},
		},
		"timeout": 2,
	})

	err := proc.Execute()
	if err != nil {
		// Timeout may surface as a process-level error (context deadline exceeded)
		t.Logf("Process error (expected timeout): %s", err.Error())
		assert.Contains(t, err.Error(), "deadline exceeded",
			"error should indicate context deadline exceeded")
		return
	}

	// Or the agent.Stream may catch the timeout and return it in Result.Error
	val := proc.Value()
	require.NotNil(t, val, "process should return a value")
	result, ok := val.(*caller.Result)
	require.True(t, ok, "value should be *caller.Result, got %T", val)
	assert.NotEmpty(t, result.Error, "should have timeout error in result")
	t.Logf("Timeout error in result: %s", result.Error)
}

// ============================================================================
// E. Error / validation scenarios
// ============================================================================

func TestProcessCall_Error_MissingAssistantID(t *testing.T) {
	testutils.Prepare(t)
	defer testutils.Clean(t)

	proc := process.New("agent.call", map[string]interface{}{
		"messages": []interface{}{
			map[string]interface{}{"role": "user", "content": "Hello"},
		},
	})

	err := proc.Execute()
	require.Error(t, err, "should fail: assistant_id is required")
	t.Logf("Expected error: %s", err.Error())
	assert.Contains(t, err.Error(), "assistant_id")
}

func TestProcessCall_Error_EmptyMessages(t *testing.T) {
	testutils.Prepare(t)
	defer testutils.Clean(t)

	proc := process.New("agent.call", map[string]interface{}{
		"assistant_id": "tests.simple-greeting",
		"messages":     []interface{}{},
	})

	err := proc.Execute()
	require.Error(t, err, "should fail: messages is required")
	t.Logf("Expected error: %s", err.Error())
	assert.Contains(t, err.Error(), "messages")
}

func TestProcessCall_Error_InvalidArgument(t *testing.T) {
	testutils.Prepare(t)
	defer testutils.Clean(t)

	// Pass a string instead of map — json.Marshal will succeed but Unmarshal will fail
	proc := process.New("agent.call", "not-a-map")

	err := proc.Execute()
	require.Error(t, err, "should fail: argument must be a map")
	t.Logf("Expected error: %s", err.Error())
}

func TestProcessCall_Error_NoArgument(t *testing.T) {
	testutils.Prepare(t)
	defer testutils.Clean(t)

	proc := process.New("agent.call")

	err := proc.Execute()
	require.Error(t, err, "should fail: argument is required")
	t.Logf("Expected error: %s", err.Error())
}

func TestProcessCall_Error_NonexistentAgent(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping: requires environment")
	}
	testutils.Prepare(t)
	defer testutils.Clean(t)

	proc := process.New("agent.call", map[string]interface{}{
		"assistant_id": "does.not.exist.agent",
		"messages": []interface{}{
			map[string]interface{}{"role": "user", "content": "Hello"},
		},
	})

	err := proc.Execute()
	require.NoError(t, err, "process should not error — error is in Result")

	result, ok := proc.Value().(*caller.Result)
	require.True(t, ok)
	assert.Equal(t, "does.not.exist.agent", result.AgentID)
	assert.NotEmpty(t, result.Error, "should have error for nonexistent agent")
	assert.Contains(t, result.Error, "failed to get agent")
	t.Logf("Expected error in result: %s", result.Error)
}
