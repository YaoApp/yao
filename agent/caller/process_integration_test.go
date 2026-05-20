//go:build integration

package caller_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/yaoapp/gou/process"
	"github.com/yaoapp/yao/agent/caller"
	"github.com/yaoapp/yao/unit-test/agent/testprepare"
)

func TestProcessCall_BasicLLM(t *testing.T) {
	testprepare.PrepareSandbox(t)

	result := process.New("agent.Call", map[string]interface{}{
		"assistant_id": "tests.caller-target",
		"messages": []map[string]interface{}{
			{"role": "user", "content": "hello"},
		},
	}).Run()

	r, ok := result.(*caller.Result)
	require.True(t, ok, "result should be *caller.Result, got %T", result)
	assert.Empty(t, r.Error)
	assert.NotNil(t, r.Response)
}

func TestProcessCall_MissingAssistantID(t *testing.T) {
	testprepare.PrepareSandbox(t)

	assert.Panics(t, func() {
		process.New("agent.Call", map[string]interface{}{
			"messages": []map[string]interface{}{
				{"role": "user", "content": "hello"},
			},
		}).Run()
	}, "should panic when assistant_id is missing")
}

func TestProcessCall_EmptyMessages(t *testing.T) {
	testprepare.PrepareSandbox(t)

	assert.Panics(t, func() {
		process.New("agent.Call", map[string]interface{}{
			"assistant_id": "tests.caller-target",
			"messages":     []map[string]interface{}{},
		}).Run()
	}, "should panic when messages is empty")
}

func TestProcessCall_NoArgument(t *testing.T) {
	testprepare.PrepareSandbox(t)

	assert.Panics(t, func() {
		process.New("agent.Call").Run()
	}, "should panic when no argument provided")
}

func TestProcessCall_NonexistentAgent(t *testing.T) {
	testprepare.PrepareSandbox(t)

	result := process.New("agent.Call", map[string]interface{}{
		"assistant_id": "nonexistent.agent.id.xyz",
		"messages": []map[string]interface{}{
			{"role": "user", "content": "hello"},
		},
	}).Run()

	r, ok := result.(*caller.Result)
	require.True(t, ok, "result should be *caller.Result, got %T", result)
	assert.NotEmpty(t, r.Error, "should have error for nonexistent agent")
}
