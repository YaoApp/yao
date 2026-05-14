package yao_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/yaoapp/yao/agent/caller"
	agentcontext "github.com/yaoapp/yao/agent/context"
	sandboxtestutils "github.com/yaoapp/yao/agent/sandbox/v2/testutils"
	oauthtypes "github.com/yaoapp/yao/openapi/oauth/types"
)

func TestSandboxV2_Yao_JSAPI(t *testing.T) {
	sandboxtestutils.Prepare(t)
	defer sandboxtestutils.Clean(t)

	require.NotNil(t, caller.AgentGetterFunc, "AgentGetterFunc should be registered after Prepare")

	agent, err := caller.AgentGetterFunc("tests.sandbox-v2.jsapi-v2")
	require.NoError(t, err, "should load assistant tests.sandbox-v2.jsapi-v2")

	chatID := fmt.Sprintf("e2e-jsapi-%d", time.Now().UnixMilli())
	ctx := agentcontext.New(
		context.Background(),
		&oauthtypes.AuthorizedInfo{
			TeamID: "test-team-jsapi",
			UserID: "test-user-jsapi",
		},
		chatID,
	)

	messages := []agentcontext.Message{
		{Role: "user", Content: "test jsapi"},
	}

	done := make(chan struct{})
	var resp *agentcontext.Response
	var streamErr error

	go func() {
		defer close(done)
		resp, streamErr = agent.Stream(ctx, messages)
	}()

	select {
	case <-done:
	case <-time.After(3 * time.Minute):
		t.Fatalf("timeout after 3m")
	}

	require.NoError(t, streamErr, "Stream should not return error")
	require.NotNil(t, resp, "response should not be nil")

	// runner=yao goes through executeLLMStream, then Next hook returns { data: results }
	// The Next hook result should appear in resp.Next
	require.NotNil(t, resp.Next, "resp.Next should not be nil (Next hook returned data)")
	t.Logf("resp.Next: %+v", resp.Next)

	nextData, ok := resp.Next.(map[string]interface{})
	if !ok {
		t.Fatalf("resp.Next should be a map, got %T: %+v", resp.Next, resp.Next)
	}

	// The Next hook returns { data: results }, the framework unwraps .data
	data, hasData := nextData["data"]
	if hasData {
		nextData, ok = data.(map[string]interface{})
		require.True(t, ok, "data should be a map")
	}

	t.Logf("JSAPI test results: %+v", nextData)

	// ── Verify ctx.computer was available ──
	assert.Equal(t, true, nextData["has_computer"], "ctx.computer should be available")
	assert.Equal(t, true, nextData["has_workspace"], "ctx.workspace should be available")

	// ── Verify ctx.computer.Info() ──
	if infoRaw, ok := nextData["computer_info"]; ok {
		info, ok := infoRaw.(map[string]interface{})
		require.True(t, ok, "computer_info should be a map")
		assert.NotEmpty(t, info["kind"], "computer_info.kind should not be empty")
		t.Logf("computer info: kind=%v os=%v", info["kind"], info["os"])
	} else {
		assert.Nil(t, nextData["computer_info_error"], "computer.Info() should not error")
	}

	// ── Verify ctx.computer.Exec() ──
	assert.Equal(t, "jsapi-v2-test", nextData["exec_stdout"], "Exec should return expected stdout")
	assert.Nil(t, nextData["exec_error"], "Exec should not error")
	if exitCode, ok := nextData["exec_exit_code"]; ok {
		// JS numbers come back as float64 through JSON
		switch v := exitCode.(type) {
		case float64:
			assert.Equal(t, float64(0), v, "exit_code should be 0")
		case int:
			assert.Equal(t, 0, v, "exit_code should be 0")
		}
	}

	// ── Verify ctx.workspace write/read ──
	assert.Equal(t, true, nextData["write_read_ok"], "workspace WriteFile+ReadFile round-trip should work")
	assert.Equal(t, "hello from jsapi v2", nextData["read_content"], "read content should match")
	assert.Nil(t, nextData["write_read_error"], "write/read should not error")

	// ── Verify ctx.workspace MkdirAll + Exists ──
	assert.Equal(t, true, nextData["mkdir_exists_ok"], "MkdirAll + Exists should work")
	assert.Nil(t, nextData["mkdir_exists_error"], "mkdir/exists should not error")

	// ── Verify ctx.workspace ReadDir ──
	assert.Nil(t, nextData["readdir_error"], "ReadDir should not error")
	if count, ok := nextData["readdir_count"]; ok {
		switch v := count.(type) {
		case float64:
			assert.Greater(t, v, float64(0), "ReadDir should return entries")
		}
	}

	// ── Verify ctx.workspace Stat ──
	assert.Equal(t, true, nextData["stat_ok"], "Stat should return correct info")
	assert.Nil(t, nextData["stat_error"], "Stat should not error")

	// ── Verify ctx.workspace Copy ──
	assert.Equal(t, true, nextData["copy_ok"], "Copy should work")
	assert.Nil(t, nextData["copy_error"], "Copy should not error")

	// ── Verify ctx.workspace Rename ──
	assert.Equal(t, true, nextData["rename_ok"], "Rename should work")
	assert.Nil(t, nextData["rename_error"], "Rename should not error")

	// ── Verify ctx.workspace Remove ──
	assert.Equal(t, true, nextData["remove_ok"], "Remove should work")
	assert.Nil(t, nextData["remove_error"], "Remove should not error")
}
