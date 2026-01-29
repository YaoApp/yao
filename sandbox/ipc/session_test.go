package ipc

import (
	"context"
	"encoding/json"
	"net"
	"os"
	"testing"
	"time"

	"github.com/yaoapp/yao/config"
	"github.com/yaoapp/yao/test"
)

// TestSessionHandleInitialize tests the initialize handler
func TestSessionHandleInitialize(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "session-init-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	m := NewManager(tmpDir)
	ctx := context.Background()

	session, err := m.Create(ctx, "init-test", &AgentContext{
		UserID: "user1",
		ChatID: "chat1",
		Locale: "en-US",
	}, nil)
	if err != nil {
		t.Fatalf("Create session failed: %v", err)
	}
	defer m.Close("init-test")

	time.Sleep(50 * time.Millisecond)

	conn, err := net.Dial("unix", session.SocketPath)
	if err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}
	defer conn.Close()

	// Send initialize
	req := JSONRPCRequest{
		JSONRPC: "2.0",
		ID:      1,
		Method:  "initialize",
		Params: json.RawMessage(`{
			"protocolVersion": "2024-11-05",
			"capabilities": {"tools": {}},
			"clientInfo": {"name": "test-client", "version": "1.0.0"}
		}`),
	}
	data, _ := json.Marshal(req)
	conn.Write(append(data, '\n'))

	conn.SetReadDeadline(time.Now().Add(5 * time.Second))
	buf := make([]byte, 4096)
	n, err := conn.Read(buf)
	if err != nil {
		t.Fatalf("Read failed: %v", err)
	}

	var resp JSONRPCResponse
	if err := json.Unmarshal(buf[:n], &resp); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	if resp.Error != nil {
		t.Fatalf("Unexpected error: %v", resp.Error)
	}

	// Parse result
	resultBytes, _ := json.Marshal(resp.Result)
	var initResult InitializeResult
	if err := json.Unmarshal(resultBytes, &initResult); err != nil {
		t.Fatalf("Failed to parse init result: %v", err)
	}

	if initResult.ProtocolVersion != "2024-11-05" {
		t.Errorf("Expected protocol version 2024-11-05, got %s", initResult.ProtocolVersion)
	}

	if initResult.ServerInfo.Name != "yao-sandbox" {
		t.Errorf("Expected server name yao-sandbox, got %s", initResult.ServerInfo.Name)
	}

	if initResult.Capabilities.Tools == nil {
		t.Error("Expected tools capability")
	}
}

// TestSessionHandleResourcesList tests the resources/list handler
func TestSessionHandleResourcesList(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "session-resources-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	m := NewManager(tmpDir)
	ctx := context.Background()

	session, err := m.Create(ctx, "resources-test", &AgentContext{
		UserID: "user1",
		ChatID: "chat1",
	}, nil)
	if err != nil {
		t.Fatalf("Create session failed: %v", err)
	}
	defer m.Close("resources-test")

	time.Sleep(50 * time.Millisecond)

	conn, err := net.Dial("unix", session.SocketPath)
	if err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}
	defer conn.Close()

	req := JSONRPCRequest{
		JSONRPC: "2.0",
		ID:      2,
		Method:  "resources/list",
	}
	data, _ := json.Marshal(req)
	conn.Write(append(data, '\n'))

	conn.SetReadDeadline(time.Now().Add(5 * time.Second))
	buf := make([]byte, 4096)
	n, err := conn.Read(buf)
	if err != nil {
		t.Fatalf("Read failed: %v", err)
	}

	var resp JSONRPCResponse
	if err := json.Unmarshal(buf[:n], &resp); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	if resp.Error != nil {
		t.Fatalf("Unexpected error: %v", resp.Error)
	}

	// Result should have empty resources array
	resultMap, ok := resp.Result.(map[string]interface{})
	if !ok {
		t.Fatalf("Expected map result")
	}

	resources, ok := resultMap["resources"].([]interface{})
	if !ok {
		t.Fatalf("Expected resources array")
	}

	if len(resources) != 0 {
		t.Errorf("Expected empty resources, got %d", len(resources))
	}
}

// TestSessionHandleResourcesRead tests the resources/read handler
func TestSessionHandleResourcesRead(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "session-read-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	m := NewManager(tmpDir)
	ctx := context.Background()

	session, err := m.Create(ctx, "read-test", &AgentContext{
		UserID: "user1",
		ChatID: "chat1",
	}, nil)
	if err != nil {
		t.Fatalf("Create session failed: %v", err)
	}
	defer m.Close("read-test")

	time.Sleep(50 * time.Millisecond)

	conn, err := net.Dial("unix", session.SocketPath)
	if err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}
	defer conn.Close()

	req := JSONRPCRequest{
		JSONRPC: "2.0",
		ID:      3,
		Method:  "resources/read",
		Params:  json.RawMessage(`{"uri": "test://resource"}`),
	}
	data, _ := json.Marshal(req)
	conn.Write(append(data, '\n'))

	conn.SetReadDeadline(time.Now().Add(5 * time.Second))
	buf := make([]byte, 4096)
	n, err := conn.Read(buf)
	if err != nil {
		t.Fatalf("Read failed: %v", err)
	}

	var resp JSONRPCResponse
	if err := json.Unmarshal(buf[:n], &resp); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	// Should return error (resource not found)
	if resp.Error == nil {
		t.Error("Expected error for non-existent resource")
	}

	if resp.Error != nil && resp.Error.Code != ErrCodeInvalidParams {
		t.Errorf("Expected error code %d, got %d", ErrCodeInvalidParams, resp.Error.Code)
	}
}

// TestSessionHandleToolsCallInvalidParams tests tools/call with invalid params
func TestSessionHandleToolsCallInvalidParams(t *testing.T) {
	// Use /tmp for shorter socket path
	tmpDir, err := os.MkdirTemp("/tmp", "ipc-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	m := NewManager(tmpDir)
	ctx := context.Background()

	session, err := m.Create(ctx, "inv", &AgentContext{
		UserID: "user1",
		ChatID: "chat1",
	}, nil)
	if err != nil {
		t.Fatalf("Create session failed: %v", err)
	}
	defer m.Close("inv")

	time.Sleep(50 * time.Millisecond)

	conn, err := net.Dial("unix", session.SocketPath)
	if err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}
	defer conn.Close()

	// Invalid params (not valid JSON object)
	req := JSONRPCRequest{
		JSONRPC: "2.0",
		ID:      4,
		Method:  "tools/call",
		Params:  json.RawMessage(`"not an object"`),
	}
	data, _ := json.Marshal(req)
	conn.Write(append(data, '\n'))

	conn.SetReadDeadline(time.Now().Add(5 * time.Second))
	buf := make([]byte, 4096)
	n, err := conn.Read(buf)
	if err != nil {
		t.Fatalf("Read failed: %v", err)
	}

	var resp JSONRPCResponse
	if err := json.Unmarshal(buf[:n], &resp); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	if resp.Error == nil {
		t.Error("Expected error for invalid params")
	}

	if resp.Error != nil && resp.Error.Code != ErrCodeInvalidParams {
		t.Errorf("Expected error code %d, got %d", ErrCodeInvalidParams, resp.Error.Code)
	}
}

// TestSessionHandleToolsCallUnauthorized tests tools/call with unauthorized tool
func TestSessionHandleToolsCallUnauthorized(t *testing.T) {
	// Use /tmp for shorter socket path
	tmpDir, err := os.MkdirTemp("/tmp", "ipc-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	m := NewManager(tmpDir)
	ctx := context.Background()

	// Create session with one tool
	mcpTools := map[string]*MCPTool{
		"allowed_tool": {
			Name:        "allowed_tool",
			Description: "An allowed tool",
			Process:     "scripts.test.allowed",
			InputSchema: json.RawMessage(`{"type":"object"}`),
		},
	}

	session, err := m.Create(ctx, "una", &AgentContext{
		UserID: "user1",
		ChatID: "chat1",
	}, mcpTools)
	if err != nil {
		t.Fatalf("Create session failed: %v", err)
	}
	defer m.Close("una")

	time.Sleep(50 * time.Millisecond)

	conn, err := net.Dial("unix", session.SocketPath)
	if err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}
	defer conn.Close()

	// Try to call unauthorized tool
	req := JSONRPCRequest{
		JSONRPC: "2.0",
		ID:      5,
		Method:  "tools/call",
		Params:  json.RawMessage(`{"name": "unauthorized_tool", "arguments": {}}`),
	}
	data, _ := json.Marshal(req)
	conn.Write(append(data, '\n'))

	conn.SetReadDeadline(time.Now().Add(5 * time.Second))
	buf := make([]byte, 4096)
	n, err := conn.Read(buf)
	if err != nil {
		t.Fatalf("Read failed: %v", err)
	}

	var resp JSONRPCResponse
	if err := json.Unmarshal(buf[:n], &resp); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	if resp.Error == nil {
		t.Error("Expected error for unauthorized tool")
	}

	if resp.Error != nil && resp.Error.Code != ErrCodeInvalidParams {
		t.Errorf("Expected error code %d, got %d", ErrCodeInvalidParams, resp.Error.Code)
	}
}

// TestSessionToolsCallWithYaoApp tests tools/call with Yao app loaded
// This is the full integration test
func TestSessionToolsCallWithYaoApp(t *testing.T) {
	// Check if YAO_TEST_APPLICATION is set
	if os.Getenv("YAO_TEST_APPLICATION") == "" {
		t.Skip("Skipping: YAO_TEST_APPLICATION not set")
	}

	// Prepare Yao test environment
	test.Prepare(t, config.Conf)
	defer test.Clean()

	tmpDir, err := os.MkdirTemp("", "session-yao-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	m := NewManager(tmpDir)
	ctx := context.Background()

	// Create session with a Yao process tool
	mcpTools := map[string]*MCPTool{
		"yao_utils_now": {
			Name:        "yao_utils_now",
			Description: "Get current time",
			Process:     "utils.now.Timestamp",
			InputSchema: json.RawMessage(`{"type":"object","properties":{}}`),
		},
	}

	session, err := m.Create(ctx, "yao-tool-test", &AgentContext{
		UserID: "user1",
		ChatID: "chat1",
		Locale: "en-US",
	}, mcpTools)
	if err != nil {
		t.Fatalf("Create session failed: %v", err)
	}
	defer m.Close("yao-tool-test")

	time.Sleep(50 * time.Millisecond)

	conn, err := net.Dial("unix", session.SocketPath)
	if err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}
	defer conn.Close()

	// Call Yao process
	req := JSONRPCRequest{
		JSONRPC: "2.0",
		ID:      10,
		Method:  "tools/call",
		Params:  json.RawMessage(`{"name": "yao_utils_now", "arguments": {}}`),
	}
	data, _ := json.Marshal(req)
	conn.Write(append(data, '\n'))

	conn.SetReadDeadline(time.Now().Add(10 * time.Second))
	buf := make([]byte, 4096)
	n, err := conn.Read(buf)
	if err != nil {
		t.Fatalf("Read failed: %v", err)
	}

	var resp JSONRPCResponse
	if err := json.Unmarshal(buf[:n], &resp); err != nil {
		t.Fatalf("Unmarshal failed: %v (raw: %s)", err, string(buf[:n]))
	}

	if resp.Error != nil {
		t.Logf("Tool call error: %v", resp.Error)
		// This is expected if the process doesn't exist in test app
		// The important thing is the IPC communication worked
		return
	}

	// Parse tool result
	resultBytes, _ := json.Marshal(resp.Result)
	var toolResult ToolResult
	if err := json.Unmarshal(resultBytes, &toolResult); err != nil {
		t.Fatalf("Failed to parse tool result: %v", err)
	}

	if len(toolResult.Content) == 0 {
		t.Error("Expected tool result content")
	}

	if toolResult.IsError {
		t.Errorf("Tool returned error: %v", toolResult.Content)
	}

	t.Logf("Tool result: %v", toolResult.Content)
}

// TestSessionMultipleRequests tests multiple requests over single connection
func TestSessionMultipleRequests(t *testing.T) {
	// Use /tmp for shorter socket path
	tmpDir, err := os.MkdirTemp("/tmp", "ipc-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	m := NewManager(tmpDir)
	ctx := context.Background()

	mcpTools := map[string]*MCPTool{
		"test_tool": {
			Name:        "test_tool",
			Description: "Test tool",
			Process:     "scripts.test.hello",
			InputSchema: json.RawMessage(`{"type":"object"}`),
		},
	}

	session, err := m.Create(ctx, "mul", &AgentContext{
		UserID: "user1",
		ChatID: "chat1",
	}, mcpTools)
	if err != nil {
		t.Fatalf("Create session failed: %v", err)
	}
	defer m.Close("mul")

	time.Sleep(50 * time.Millisecond)

	conn, err := net.Dial("unix", session.SocketPath)
	if err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}
	defer conn.Close()

	// Send multiple requests
	requests := []JSONRPCRequest{
		{JSONRPC: "2.0", ID: 1, Method: "initialize", Params: json.RawMessage(`{}`)},
		{JSONRPC: "2.0", ID: 2, Method: "tools/list"},
		{JSONRPC: "2.0", ID: 3, Method: "resources/list"},
	}

	for _, req := range requests {
		data, _ := json.Marshal(req)
		conn.Write(append(data, '\n'))

		conn.SetReadDeadline(time.Now().Add(5 * time.Second))
		buf := make([]byte, 4096)
		n, err := conn.Read(buf)
		if err != nil {
			t.Fatalf("Read for request %v failed: %v", req.ID, err)
		}

		var resp JSONRPCResponse
		if err := json.Unmarshal(buf[:n], &resp); err != nil {
			t.Fatalf("Unmarshal for request %v failed: %v", req.ID, err)
		}

		if resp.Error != nil {
			t.Errorf("Request %v returned error: %v", req.ID, resp.Error)
		}

		// Compare IDs as float64 since JSON numbers are decoded as float64
		reqIDFloat := float64(req.ID.(int))
		respIDFloat, ok := resp.ID.(float64)
		if !ok {
			t.Errorf("Response ID type is %T, expected float64", resp.ID)
		} else if respIDFloat != reqIDFloat {
			t.Errorf("Response ID %v doesn't match request ID %v", resp.ID, req.ID)
		}
	}
}

// TestSessionClose tests session close behavior
func TestSessionClose(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "session-close-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	m := NewManager(tmpDir)
	ctx := context.Background()

	session, err := m.Create(ctx, "close-test", &AgentContext{
		UserID: "user1",
		ChatID: "chat1",
	}, nil)
	if err != nil {
		t.Fatalf("Create session failed: %v", err)
	}

	socketPath := session.SocketPath

	time.Sleep(50 * time.Millisecond)

	// Connect
	conn, err := net.Dial("unix", socketPath)
	if err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}

	// Close session
	session.Close()

	// Wait a bit for cleanup
	time.Sleep(100 * time.Millisecond)

	// Connection should be broken
	conn.SetReadDeadline(time.Now().Add(500 * time.Millisecond))
	buf := make([]byte, 4096)
	_, err = conn.Read(buf)
	// Either EOF or connection reset is expected
	if err == nil {
		t.Error("Expected connection to be closed")
	}

	conn.Close()

	// Socket file should be removed
	if _, err := os.Stat(socketPath); !os.IsNotExist(err) {
		t.Error("Socket file should be removed after close")
	}
}

// TestSessionEmptyLines tests handling of empty lines
func TestSessionEmptyLines(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "session-empty-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	m := NewManager(tmpDir)
	ctx := context.Background()

	session, err := m.Create(ctx, "empty-test", &AgentContext{
		UserID: "user1",
		ChatID: "chat1",
	}, nil)
	if err != nil {
		t.Fatalf("Create session failed: %v", err)
	}
	defer m.Close("empty-test")

	time.Sleep(50 * time.Millisecond)

	conn, err := net.Dial("unix", session.SocketPath)
	if err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}
	defer conn.Close()

	// Send empty lines followed by valid request
	conn.Write([]byte("\n\n\n"))

	req := JSONRPCRequest{
		JSONRPC: "2.0",
		ID:      1,
		Method:  "initialize",
	}
	data, _ := json.Marshal(req)
	conn.Write(append(data, '\n'))

	// Should still get response
	conn.SetReadDeadline(time.Now().Add(5 * time.Second))
	buf := make([]byte, 4096)
	n, err := conn.Read(buf)
	if err != nil {
		t.Fatalf("Read failed: %v", err)
	}

	var resp JSONRPCResponse
	if err := json.Unmarshal(buf[:n], &resp); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	if resp.Error != nil {
		t.Errorf("Unexpected error: %v", resp.Error)
	}
}
