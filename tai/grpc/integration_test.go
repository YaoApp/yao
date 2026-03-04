package grpc_test

import (
	"context"
	"encoding/json"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/yaoapp/yao/grpc/tests/testutils"
	yaogrpc "github.com/yaoapp/yao/tai/grpc"
)

// Integration tests that start a real Yao gRPC server and test the tai/grpc
// client through the full interceptor -> handler chain.

func setupClient(t *testing.T, scopes ...string) *yaogrpc.Client {
	t.Helper()

	conn := testutils.Prepare(t)
	t.Cleanup(func() {
		conn.Close()
		testutils.Clean()
	})

	addr := testutils.Addr()
	token := testutils.ObtainAccessToken(t, scopes...)
	refreshToken := testutils.ObtainRefreshToken(t, scopes...)

	tm := yaogrpc.NewTokenManager(token, refreshToken, "test-sandbox", "")
	client, err := yaogrpc.Dial(addr, tm)
	require.NoError(t, err)
	t.Cleanup(func() { client.Close() })

	return client
}

// ── Healthz ──────────────────────────────────────────────────────────────────

func TestIntegration_Healthz(t *testing.T) {
	conn := testutils.Prepare(t)
	defer func() {
		conn.Close()
		testutils.Clean()
	}()

	addr := testutils.Addr()
	tm := yaogrpc.NewTokenManager("", "", "", "")
	client, err := yaogrpc.Dial(addr, tm)
	require.NoError(t, err)
	defer client.Close()

	status, err := client.Healthz(context.Background())
	assert.NoError(t, err)
	assert.Equal(t, "ok", status)
}

// ── Run ──────────────────────────────────────────────────────────────────────

func TestIntegration_Run_Ping(t *testing.T) {
	client := setupClient(t, "grpc:run")

	data, err := client.Run(context.Background(), "utils.app.Ping", nil, 0)
	assert.NoError(t, err)
	assert.NotNil(t, data)
}

func TestIntegration_Run_InvalidProcess(t *testing.T) {
	client := setupClient(t, "grpc:run")

	_, err := client.Run(context.Background(), "nonexistent.process", nil, 0)
	assert.Error(t, err)
}

func TestIntegration_Run_WithArgs(t *testing.T) {
	client := setupClient(t, "grpc:run")

	args, _ := json.Marshal([]any{"hello", "world"})
	data, err := client.Run(context.Background(), "utils.app.Ping", args, 5)
	assert.NoError(t, err)
	assert.NotNil(t, data)
}

// ── Shell ────────────────────────────────────────────────────────────────────

func TestIntegration_Shell_Echo(t *testing.T) {
	client := setupClient(t, "grpc:shell")

	resp, err := client.Shell(context.Background(), "echo", []string{"hello"}, nil, 5)
	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.Equal(t, int32(0), resp.ExitCode)
	assert.Contains(t, string(resp.Stdout), "hello")
}

func TestIntegration_Shell_NotFound(t *testing.T) {
	client := setupClient(t, "grpc:shell")

	_, err := client.Shell(context.Background(), "nonexistent-command-xyz", nil, nil, 5)
	assert.Error(t, err)
}

// ── MCP ──────────────────────────────────────────────────────────────────────

func TestIntegration_MCPListTools(t *testing.T) {
	client := setupClient(t, "grpc:mcp")

	data, err := client.MCPListTools(context.Background(), "echo")
	assert.NoError(t, err)
	assert.NotNil(t, data)

	var tools []any
	assert.NoError(t, json.Unmarshal(data, &tools))
	assert.Greater(t, len(tools), 0)
}

func TestIntegration_MCPCallTool(t *testing.T) {
	client := setupClient(t, "grpc:mcp")

	args, _ := json.Marshal(map[string]string{"message": "hi"})
	data, err := client.MCPCallTool(context.Background(), "echo", "ping", args)
	assert.NoError(t, err)
	assert.NotNil(t, data)
}

func TestIntegration_MCPListResources(t *testing.T) {
	client := setupClient(t, "grpc:mcp")

	data, err := client.MCPListResources(context.Background(), "echo")
	assert.NoError(t, err)
	assert.NotNil(t, data)
}

func TestIntegration_MCPReadResource(t *testing.T) {
	client := setupClient(t, "grpc:mcp")

	data, err := client.MCPReadResource(context.Background(), "echo", "echo://info")
	assert.NoError(t, err)
	assert.NotNil(t, data)
}

// ── API ──────────────────────────────────────────────────────────────────────

func TestIntegration_API_Proxy(t *testing.T) {
	client := setupClient(t, "grpc:run", "grpc:mcp")

	resp, err := client.API(context.Background(), "GET", "/api/__yao/app/setting", nil, nil)
	require.NoError(t, err)
	require.NotNil(t, resp)
	t.Logf("API proxy status: %d", resp.Status)
}

// ── LLM ──────────────────────────────────────────────────────────────────────

func TestIntegration_ChatCompletions_InvalidConnector(t *testing.T) {
	client := setupClient(t, "grpc:llm")

	messages, _ := json.Marshal([]map[string]string{
		{"role": "user", "content": "test"},
	})

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	_, err := client.ChatCompletions(ctx, "nonexistent-connector", messages, nil)
	assert.Error(t, err)
}

func TestIntegration_ChatCompletionsStream_InvalidConnector(t *testing.T) {
	client := setupClient(t, "grpc:llm")

	messages, _ := json.Marshal([]map[string]string{
		{"role": "user", "content": "test"},
	})

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	err := client.ChatCompletionsStream(ctx, "nonexistent-connector", messages, nil,
		func(data []byte, done bool) error { return nil })
	assert.Error(t, err)
}

func TestIntegration_ChatCompletions_EmptyMessages(t *testing.T) {
	client := setupClient(t, "grpc:llm")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	_, err := client.ChatCompletions(ctx, "default", nil, nil)
	assert.Error(t, err)
}

// ── Agent ────────────────────────────────────────────────────────────────────

func TestIntegration_AgentStream_InvalidRobot(t *testing.T) {
	client := setupClient(t, "grpc:agent")

	messages, _ := json.Marshal([]map[string]string{
		{"role": "user", "content": "hello"},
	})

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	err := client.AgentStream(ctx, "nonexistent-robot-xyz", messages, nil,
		func(data []byte, done bool) error { return nil })
	assert.Error(t, err)
}

// ── Unauthenticated ──────────────────────────────────────────────────────────

func TestIntegration_Run_NoToken(t *testing.T) {
	conn := testutils.Prepare(t)
	defer func() {
		conn.Close()
		testutils.Clean()
	}()

	addr := testutils.Addr()
	tm := yaogrpc.NewTokenManager("", "", "", "")
	client, err := yaogrpc.Dial(addr, tm)
	require.NoError(t, err)
	defer client.Close()

	_, err = client.Run(context.Background(), "utils.app.Ping", nil, 0)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Unauthenticated")
}

// ── Token Refresh via interceptor ────────────────────────────────────────────

func TestIntegration_TokenRefresh(t *testing.T) {
	conn := testutils.Prepare(t)
	defer func() {
		conn.Close()
		testutils.Clean()
	}()

	addr := testutils.Addr()
	scopes := []string{"grpc:run"}

	expiredToken := testutils.ObtainExpiredAccessToken(t, scopes...)
	refreshToken := testutils.ObtainRefreshToken(t, scopes...)

	tm := yaogrpc.NewTokenManager(expiredToken, refreshToken, "sb-test", "")
	client, err := yaogrpc.Dial(addr, tm)
	require.NoError(t, err)
	defer client.Close()

	data, err := client.Run(context.Background(), "utils.app.Ping", nil, 0)
	assert.NoError(t, err)
	assert.NotNil(t, data)

	newToken := tm.AccessToken()
	if newToken != expiredToken {
		t.Logf("token was refreshed: old=%s... new=%s...", expiredToken[:20], newToken[:20])
	}
}

// ══════════════════════════════════════════════════════════════════════════════
// Relay mode tests — client → Tai (:9100) → x-grpc-upstream → Yao gRPC
// Requires TAI_TEST_GRPC env var (e.g. 127.0.0.1:9100) and a running Tai server.
// ══════════════════════════════════════════════════════════════════════════════

func setupRelayClient(t *testing.T, scopes ...string) *yaogrpc.Client {
	t.Helper()

	taiAddr := os.Getenv("TAI_TEST_GRPC")
	if taiAddr == "" {
		t.Skip("TAI_TEST_GRPC not set, skipping relay mode test")
	}

	conn := testutils.Prepare(t)
	t.Cleanup(func() {
		conn.Close()
		testutils.Clean()
	})

	yaoAddr := testutils.Addr()
	token := testutils.ObtainAccessToken(t, scopes...)
	refreshToken := testutils.ObtainRefreshToken(t, scopes...)

	// upstream = Yao gRPC address; taiMode = true
	tm := yaogrpc.NewTokenManager(token, refreshToken, "relay-sandbox", yaoAddr)
	client, err := yaogrpc.Dial(taiAddr, tm)
	require.NoError(t, err)
	t.Cleanup(func() { client.Close() })

	return client
}

func TestRelay_Healthz(t *testing.T) {
	taiAddr := os.Getenv("TAI_TEST_GRPC")
	if taiAddr == "" {
		t.Skip("TAI_TEST_GRPC not set")
	}

	conn := testutils.Prepare(t)
	defer func() {
		conn.Close()
		testutils.Clean()
	}()

	yaoAddr := testutils.Addr()
	tm := yaogrpc.NewTokenManager("", "", "", yaoAddr)
	client, err := yaogrpc.Dial(taiAddr, tm)
	require.NoError(t, err)
	defer client.Close()

	status, err := client.Healthz(context.Background())
	require.NoError(t, err)
	assert.Equal(t, "ok", status)
}

func TestRelay_Run_Ping(t *testing.T) {
	client := setupRelayClient(t, "grpc:run")

	data, err := client.Run(context.Background(), "utils.app.Ping", nil, 0)
	require.NoError(t, err)
	require.NotNil(t, data)
	t.Logf("relay Run result: %s", string(data))
}

func TestRelay_Run_InvalidProcess(t *testing.T) {
	client := setupRelayClient(t, "grpc:run")

	_, err := client.Run(context.Background(), "nonexistent.process", nil, 0)
	assert.Error(t, err)
}

func TestRelay_Shell_Echo(t *testing.T) {
	client := setupRelayClient(t, "grpc:shell")

	resp, err := client.Shell(context.Background(), "echo", []string{"relay-test"}, nil, 5)
	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.Equal(t, int32(0), resp.ExitCode)
	assert.Contains(t, string(resp.Stdout), "relay-test")
}

func TestRelay_MCPListTools(t *testing.T) {
	client := setupRelayClient(t, "grpc:mcp")

	data, err := client.MCPListTools(context.Background(), "echo")
	assert.NoError(t, err)
	assert.NotNil(t, data)

	var tools []any
	assert.NoError(t, json.Unmarshal(data, &tools))
	assert.Greater(t, len(tools), 0)
}

func TestRelay_MCPCallTool(t *testing.T) {
	client := setupRelayClient(t, "grpc:mcp")

	args, _ := json.Marshal(map[string]string{"message": "relay"})
	data, err := client.MCPCallTool(context.Background(), "echo", "ping", args)
	assert.NoError(t, err)
	assert.NotNil(t, data)
}

func TestRelay_Run_NoToken(t *testing.T) {
	taiAddr := os.Getenv("TAI_TEST_GRPC")
	if taiAddr == "" {
		t.Skip("TAI_TEST_GRPC not set")
	}

	conn := testutils.Prepare(t)
	defer func() {
		conn.Close()
		testutils.Clean()
	}()

	yaoAddr := testutils.Addr()
	tm := yaogrpc.NewTokenManager("", "", "", yaoAddr)
	client, err := yaogrpc.Dial(taiAddr, tm)
	require.NoError(t, err)
	defer client.Close()

	_, err = client.Run(context.Background(), "utils.app.Ping", nil, 0)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Unauthenticated")
}

func TestRelay_TokenRefresh(t *testing.T) {
	taiAddr := os.Getenv("TAI_TEST_GRPC")
	if taiAddr == "" {
		t.Skip("TAI_TEST_GRPC not set")
	}

	conn := testutils.Prepare(t)
	defer func() {
		conn.Close()
		testutils.Clean()
	}()

	yaoAddr := testutils.Addr()
	scopes := []string{"grpc:run"}

	expiredToken := testutils.ObtainExpiredAccessToken(t, scopes...)
	refreshToken := testutils.ObtainRefreshToken(t, scopes...)

	tm := yaogrpc.NewTokenManager(expiredToken, refreshToken, "relay-sb", yaoAddr)
	client, err := yaogrpc.Dial(taiAddr, tm)
	require.NoError(t, err)
	defer client.Close()

	data, err := client.Run(context.Background(), "utils.app.Ping", nil, 0)
	assert.NoError(t, err)
	assert.NotNil(t, data)

	newToken := tm.AccessToken()
	if newToken != expiredToken {
		t.Logf("relay token refreshed: old=%s... new=%s...", expiredToken[:20], newToken[:20])
	}
}
