package ipc

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"
)

// TestNewManager tests IPC manager creation
func TestNewManager(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "ipc-manager-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	m := NewManager(tmpDir)
	if m == nil {
		t.Fatal("NewManager returned nil")
	}

	if m.sockDir != tmpDir {
		t.Errorf("Expected sockDir %s, got %s", tmpDir, m.sockDir)
	}
}

// TestCreateSession tests creating an IPC session
func TestCreateSession(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "ipc-session-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	m := NewManager(tmpDir)
	ctx := context.Background()

	sessionID := "test-session-1"
	agentCtx := &AgentContext{
		UserID: "user1",
		ChatID: "chat1",
		Locale: "en-US",
	}

	mcpTools := map[string]*MCPTool{
		"test_tool": {
			Name:        "test_tool",
			Description: "A test tool",
			Process:     "scripts.test.hello",
			InputSchema: json.RawMessage(`{"type":"object","properties":{"name":{"type":"string"}}}`),
		},
	}

	session, err := m.Create(ctx, sessionID, agentCtx, mcpTools)
	if err != nil {
		t.Fatalf("Create session failed: %v", err)
	}
	defer m.Close(sessionID)

	// Verify session properties
	if session.ID != sessionID {
		t.Errorf("Expected session ID %s, got %s", sessionID, session.ID)
	}

	expectedSocketPath := filepath.Join(tmpDir, sessionID+".sock")
	if session.SocketPath != expectedSocketPath {
		t.Errorf("Expected socket path %s, got %s", expectedSocketPath, session.SocketPath)
	}

	if session.Context.UserID != "user1" {
		t.Errorf("Expected UserID user1, got %s", session.Context.UserID)
	}

	if len(session.MCPTools) != 1 {
		t.Errorf("Expected 1 MCP tool, got %d", len(session.MCPTools))
	}

	// Verify socket file exists
	if _, err := os.Stat(session.SocketPath); os.IsNotExist(err) {
		t.Error("Socket file should exist")
	}
}

// TestGetSession tests retrieving a session
func TestGetSession(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "ipc-get-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	m := NewManager(tmpDir)
	ctx := context.Background()

	sessionID := "test-get-session"
	agentCtx := &AgentContext{UserID: "user1", ChatID: "chat1"}

	// Get non-existent session
	_, ok := m.Get(sessionID)
	if ok {
		t.Error("Get should return false for non-existent session")
	}

	// Create session
	_, err = m.Create(ctx, sessionID, agentCtx, nil)
	if err != nil {
		t.Fatalf("Create session failed: %v", err)
	}
	defer m.Close(sessionID)

	// Get existing session
	session, ok := m.Get(sessionID)
	if !ok {
		t.Error("Get should return true for existing session")
	}

	if session.ID != sessionID {
		t.Errorf("Expected session ID %s, got %s", sessionID, session.ID)
	}
}

// TestCloseSession tests closing a session
func TestCloseSession(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "ipc-close-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	m := NewManager(tmpDir)
	ctx := context.Background()

	sessionID := "test-close-session"
	agentCtx := &AgentContext{UserID: "user1", ChatID: "chat1"}

	session, err := m.Create(ctx, sessionID, agentCtx, nil)
	if err != nil {
		t.Fatalf("Create session failed: %v", err)
	}

	socketPath := session.SocketPath

	// Close session
	err = m.Close(sessionID)
	if err != nil {
		t.Fatalf("Close session failed: %v", err)
	}

	// Verify session is removed
	_, ok := m.Get(sessionID)
	if ok {
		t.Error("Session should be removed after close")
	}

	// Verify socket file is removed (give it a moment)
	time.Sleep(100 * time.Millisecond)
	if _, err := os.Stat(socketPath); !os.IsNotExist(err) {
		t.Error("Socket file should be removed after close")
	}
}

// TestCloseNonExistentSession tests closing a non-existent session
func TestCloseNonExistentSession(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "ipc-close-nonexist-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	m := NewManager(tmpDir)

	// Should not error
	err = m.Close("nonexistent-session")
	if err != nil {
		t.Errorf("Close non-existent session should not error: %v", err)
	}
}

// TestCloseAllSessions tests closing all sessions
func TestCloseAllSessions(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "ipc-closeall-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	m := NewManager(tmpDir)
	ctx := context.Background()

	// Create multiple sessions
	sessionIDs := []string{"session-1", "session-2", "session-3"}
	for _, id := range sessionIDs {
		_, err := m.Create(ctx, id, &AgentContext{UserID: "user", ChatID: id}, nil)
		if err != nil {
			t.Fatalf("Create session %s failed: %v", id, err)
		}
	}

	// Verify sessions exist
	for _, id := range sessionIDs {
		if _, ok := m.Get(id); !ok {
			t.Errorf("Session %s should exist", id)
		}
	}

	// Close all
	m.CloseAll()

	// Verify all sessions are removed
	time.Sleep(100 * time.Millisecond)
	for _, id := range sessionIDs {
		if _, ok := m.Get(id); ok {
			t.Errorf("Session %s should be removed after CloseAll", id)
		}
	}
}

// TestSessionReplace tests that creating a session with existing ID replaces it
func TestSessionReplace(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "ipc-replace-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	m := NewManager(tmpDir)
	ctx := context.Background()

	sessionID := "test-replace-session"

	// Create first session
	session1, err := m.Create(ctx, sessionID, &AgentContext{UserID: "user1", ChatID: "chat1"}, nil)
	if err != nil {
		t.Fatalf("Create first session failed: %v", err)
	}
	socketPath1 := session1.SocketPath

	// Create second session with same ID
	session2, err := m.Create(ctx, sessionID, &AgentContext{UserID: "user2", ChatID: "chat2"}, nil)
	if err != nil {
		t.Fatalf("Create second session failed: %v", err)
	}
	defer m.Close(sessionID)

	// Verify second session replaced first
	if session2.Context.UserID != "user2" {
		t.Errorf("Expected UserID user2, got %s", session2.Context.UserID)
	}

	// Get session should return second
	session, ok := m.Get(sessionID)
	if !ok {
		t.Error("Get should return session")
	}
	if session.Context.UserID != "user2" {
		t.Errorf("Expected UserID user2 from Get, got %s", session.Context.UserID)
	}

	// Same socket path should be reused
	if session2.SocketPath != socketPath1 {
		t.Errorf("Expected same socket path, got %s vs %s", socketPath1, session2.SocketPath)
	}
}

// TestConcurrentSessionAccess tests concurrent access to sessions
func TestConcurrentSessionAccess(t *testing.T) {
	// Use /tmp for shorter socket path (macOS has 104 char limit for Unix sockets)
	tmpDir, err := os.MkdirTemp("/tmp", "ipc-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	m := NewManager(tmpDir)
	ctx := context.Background()

	var wg sync.WaitGroup
	var mu sync.Mutex
	errors := make([]error, 0)
	numGoroutines := 5 // Reduced for stability

	// Concurrent creates
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			sessionID := fmt.Sprintf("s%d", idx) // Short session ID
			_, err := m.Create(ctx, sessionID, &AgentContext{UserID: "user", ChatID: sessionID}, nil)
			if err != nil {
				mu.Lock()
				errors = append(errors, fmt.Errorf("session %s: %v", sessionID, err))
				mu.Unlock()
			}
		}(i)
	}

	wg.Wait()

	// Check errors
	for _, err := range errors {
		t.Errorf("Concurrent create error: %v", err)
	}

	// Verify all sessions exist
	for i := 0; i < numGoroutines; i++ {
		sessionID := fmt.Sprintf("s%d", i)
		if _, ok := m.Get(sessionID); !ok {
			t.Errorf("Session %s should exist", sessionID)
		}
	}

	// Cleanup
	m.CloseAll()
}

// TestSessionConnection tests connecting to a session socket
func TestSessionConnection(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "ipc-connect-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	m := NewManager(tmpDir)
	ctx := context.Background()

	sessionID := "test-connect-session"
	agentCtx := &AgentContext{UserID: "user1", ChatID: "chat1"}
	mcpTools := map[string]*MCPTool{}

	session, err := m.Create(ctx, sessionID, agentCtx, mcpTools)
	if err != nil {
		t.Fatalf("Create session failed: %v", err)
	}
	defer m.Close(sessionID)

	// Give the listener time to start
	time.Sleep(50 * time.Millisecond)

	// Try to connect to the socket
	conn, err := net.Dial("unix", session.SocketPath)
	if err != nil {
		t.Fatalf("Failed to connect to socket: %v", err)
	}
	defer conn.Close()

	// Send initialize request
	initReq := JSONRPCRequest{
		JSONRPC: "2.0",
		ID:      1,
		Method:  "initialize",
		Params:  json.RawMessage(`{"protocolVersion":"2024-11-05"}`),
	}
	data, _ := json.Marshal(initReq)

	// Write with newline (NDJSON)
	_, err = conn.Write(append(data, '\n'))
	if err != nil {
		t.Fatalf("Failed to write to socket: %v", err)
	}

	// Set read deadline
	conn.SetReadDeadline(time.Now().Add(5 * time.Second))

	// Read response
	buf := make([]byte, 4096)
	n, err := conn.Read(buf)
	if err != nil {
		t.Fatalf("Failed to read from socket: %v", err)
	}

	// Parse response
	var resp JSONRPCResponse
	if err := json.Unmarshal(buf[:n], &resp); err != nil {
		t.Fatalf("Failed to parse response: %v (raw: %s)", err, string(buf[:n]))
	}

	if resp.JSONRPC != "2.0" {
		t.Errorf("Expected JSONRPC 2.0, got %s", resp.JSONRPC)
	}

	if resp.Error != nil {
		t.Errorf("Unexpected error: %v", resp.Error)
	}

	if resp.Result == nil {
		t.Error("Expected result, got nil")
	}
}

// TestToolsList tests the tools/list method
func TestToolsList(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "ipc-tools-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	m := NewManager(tmpDir)
	ctx := context.Background()

	sessionID := "test-tools-session"
	agentCtx := &AgentContext{UserID: "user1", ChatID: "chat1"}
	mcpTools := map[string]*MCPTool{
		"tool1": {
			Name:        "tool1",
			Description: "First test tool",
			Process:     "scripts.test.tool1",
			InputSchema: json.RawMessage(`{"type":"object"}`),
		},
		"tool2": {
			Name:        "tool2",
			Description: "Second test tool",
			Process:     "scripts.test.tool2",
			InputSchema: json.RawMessage(`{"type":"object","properties":{"arg":{"type":"string"}}}`),
		},
	}

	session, err := m.Create(ctx, sessionID, agentCtx, mcpTools)
	if err != nil {
		t.Fatalf("Create session failed: %v", err)
	}
	defer m.Close(sessionID)

	time.Sleep(50 * time.Millisecond)

	conn, err := net.Dial("unix", session.SocketPath)
	if err != nil {
		t.Fatalf("Failed to connect to socket: %v", err)
	}
	defer conn.Close()

	// Send tools/list request
	req := JSONRPCRequest{
		JSONRPC: "2.0",
		ID:      2,
		Method:  "tools/list",
	}
	data, _ := json.Marshal(req)
	conn.Write(append(data, '\n'))

	// Read response
	conn.SetReadDeadline(time.Now().Add(5 * time.Second))
	buf := make([]byte, 4096)
	n, err := conn.Read(buf)
	if err != nil {
		t.Fatalf("Failed to read from socket: %v", err)
	}

	var resp JSONRPCResponse
	if err := json.Unmarshal(buf[:n], &resp); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	if resp.Error != nil {
		t.Fatalf("Unexpected error: %v", resp.Error)
	}

	// Parse result as ToolsListResult
	resultBytes, _ := json.Marshal(resp.Result)
	var toolsResult ToolsListResult
	if err := json.Unmarshal(resultBytes, &toolsResult); err != nil {
		t.Fatalf("Failed to parse tools result: %v", err)
	}

	if len(toolsResult.Tools) != 2 {
		t.Errorf("Expected 2 tools, got %d", len(toolsResult.Tools))
	}

	// Verify tool names
	toolNames := make(map[string]bool)
	for _, tool := range toolsResult.Tools {
		toolNames[tool.Name] = true
	}

	if !toolNames["tool1"] {
		t.Error("Expected tool1 in tools list")
	}
	if !toolNames["tool2"] {
		t.Error("Expected tool2 in tools list")
	}
}

// TestMethodNotFound tests handling of unknown methods
func TestMethodNotFound(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "ipc-notfound-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	m := NewManager(tmpDir)
	ctx := context.Background()

	session, err := m.Create(ctx, "test-notfound", &AgentContext{UserID: "user", ChatID: "chat"}, nil)
	if err != nil {
		t.Fatalf("Create session failed: %v", err)
	}
	defer m.Close("test-notfound")

	time.Sleep(50 * time.Millisecond)

	conn, err := net.Dial("unix", session.SocketPath)
	if err != nil {
		t.Fatalf("Failed to connect to socket: %v", err)
	}
	defer conn.Close()

	// Send unknown method
	req := JSONRPCRequest{
		JSONRPC: "2.0",
		ID:      3,
		Method:  "unknown/method",
	}
	data, _ := json.Marshal(req)
	conn.Write(append(data, '\n'))

	// Read response
	conn.SetReadDeadline(time.Now().Add(5 * time.Second))
	buf := make([]byte, 4096)
	n, err := conn.Read(buf)
	if err != nil {
		t.Fatalf("Failed to read from socket: %v", err)
	}

	var resp JSONRPCResponse
	if err := json.Unmarshal(buf[:n], &resp); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	if resp.Error == nil {
		t.Error("Expected error for unknown method")
	}

	if resp.Error != nil && resp.Error.Code != ErrCodeMethodNotFound {
		t.Errorf("Expected error code %d, got %d", ErrCodeMethodNotFound, resp.Error.Code)
	}
}

// TestParseError tests handling of invalid JSON
func TestParseError(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "ipc-parse-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	m := NewManager(tmpDir)
	ctx := context.Background()

	session, err := m.Create(ctx, "test-parse", &AgentContext{UserID: "user", ChatID: "chat"}, nil)
	if err != nil {
		t.Fatalf("Create session failed: %v", err)
	}
	defer m.Close("test-parse")

	time.Sleep(50 * time.Millisecond)

	conn, err := net.Dial("unix", session.SocketPath)
	if err != nil {
		t.Fatalf("Failed to connect to socket: %v", err)
	}
	defer conn.Close()

	// Send invalid JSON
	conn.Write([]byte("not valid json\n"))

	// Read response
	conn.SetReadDeadline(time.Now().Add(5 * time.Second))
	buf := make([]byte, 4096)
	n, err := conn.Read(buf)
	if err != nil {
		t.Fatalf("Failed to read from socket: %v", err)
	}

	var resp JSONRPCResponse
	if err := json.Unmarshal(buf[:n], &resp); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	if resp.Error == nil {
		t.Error("Expected error for invalid JSON")
	}

	if resp.Error != nil && resp.Error.Code != ErrCodeParse {
		t.Errorf("Expected error code %d, got %d", ErrCodeParse, resp.Error.Code)
	}
}

// TestInitializedNotification tests that initialized notification doesn't return response
func TestInitializedNotification(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "ipc-initialized-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	m := NewManager(tmpDir)
	ctx := context.Background()

	session, err := m.Create(ctx, "test-initialized", &AgentContext{UserID: "user", ChatID: "chat"}, nil)
	if err != nil {
		t.Fatalf("Create session failed: %v", err)
	}
	defer m.Close("test-initialized")

	time.Sleep(50 * time.Millisecond)

	conn, err := net.Dial("unix", session.SocketPath)
	if err != nil {
		t.Fatalf("Failed to connect to socket: %v", err)
	}
	defer conn.Close()

	// Send initialized notification (no ID = notification)
	req := JSONRPCRequest{
		JSONRPC: "2.0",
		Method:  "initialized",
	}
	data, _ := json.Marshal(req)
	conn.Write(append(data, '\n'))

	// Set short read deadline - we expect timeout since no response
	conn.SetReadDeadline(time.Now().Add(500 * time.Millisecond))
	buf := make([]byte, 4096)
	_, err = conn.Read(buf)

	// Expect timeout (no response for notifications)
	if err == nil {
		t.Error("Expected no response for notification")
	}
}
