package ipc

import (
	"encoding/json"
	"testing"
)

func TestJSONRPCRequestParsing(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected JSONRPCRequest
	}{
		{
			name:  "initialize request",
			input: `{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2024-11-05"}}`,
			expected: JSONRPCRequest{
				JSONRPC: "2.0",
				ID:      float64(1), // JSON numbers are float64
				Method:  "initialize",
			},
		},
		{
			name:  "tools/list request",
			input: `{"jsonrpc":"2.0","id":2,"method":"tools/list"}`,
			expected: JSONRPCRequest{
				JSONRPC: "2.0",
				ID:      float64(2),
				Method:  "tools/list",
			},
		},
		{
			name:  "tools/call request",
			input: `{"jsonrpc":"2.0","id":3,"method":"tools/call","params":{"name":"test","arguments":{}}}`,
			expected: JSONRPCRequest{
				JSONRPC: "2.0",
				ID:      float64(3),
				Method:  "tools/call",
			},
		},
		{
			name:  "notification (no id)",
			input: `{"jsonrpc":"2.0","method":"initialized"}`,
			expected: JSONRPCRequest{
				JSONRPC: "2.0",
				Method:  "initialized",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var req JSONRPCRequest
			if err := json.Unmarshal([]byte(tt.input), &req); err != nil {
				t.Fatalf("failed to parse: %v", err)
			}

			if req.JSONRPC != tt.expected.JSONRPC {
				t.Errorf("JSONRPC = %s, want %s", req.JSONRPC, tt.expected.JSONRPC)
			}
			if req.Method != tt.expected.Method {
				t.Errorf("Method = %s, want %s", req.Method, tt.expected.Method)
			}
			if tt.expected.ID != nil && req.ID != tt.expected.ID {
				t.Errorf("ID = %v, want %v", req.ID, tt.expected.ID)
			}
		})
	}
}

func TestJSONRPCResponseSerialization(t *testing.T) {
	// Success response
	resp := JSONRPCResponse{
		JSONRPC: "2.0",
		ID:      1,
		Result: map[string]interface{}{
			"protocolVersion": "2024-11-05",
		},
	}

	data, err := json.Marshal(resp)
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}

	// Verify it can be parsed back
	var parsed JSONRPCResponse
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	if parsed.JSONRPC != "2.0" {
		t.Errorf("JSONRPC = %s, want 2.0", parsed.JSONRPC)
	}
	if parsed.Error != nil {
		t.Errorf("Error should be nil")
	}
}

func TestJSONRPCErrorResponse(t *testing.T) {
	resp := JSONRPCResponse{
		JSONRPC: "2.0",
		ID:      1,
		Error: &JSONRPCError{
			Code:    ErrCodeMethodNotFound,
			Message: "Method not found",
		},
	}

	data, err := json.Marshal(resp)
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}

	var parsed JSONRPCResponse
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	if parsed.Error == nil {
		t.Fatal("Error should not be nil")
	}
	if parsed.Error.Code != ErrCodeMethodNotFound {
		t.Errorf("Error.Code = %d, want %d", parsed.Error.Code, ErrCodeMethodNotFound)
	}
	if parsed.Error.Message != "Method not found" {
		t.Errorf("Error.Message = %s, want 'Method not found'", parsed.Error.Message)
	}
}

func TestToolCallParams(t *testing.T) {
	input := `{"name":"my_tool","arguments":{"key":"value","num":42}}`

	var params ToolCallParams
	if err := json.Unmarshal([]byte(input), &params); err != nil {
		t.Fatalf("failed to parse: %v", err)
	}

	if params.Name != "my_tool" {
		t.Errorf("Name = %s, want my_tool", params.Name)
	}
	if params.Arguments["key"] != "value" {
		t.Errorf("Arguments[key] = %v, want value", params.Arguments["key"])
	}
	if params.Arguments["num"] != float64(42) {
		t.Errorf("Arguments[num] = %v, want 42", params.Arguments["num"])
	}
}

func TestToolResult(t *testing.T) {
	result := ToolResult{
		Content: []ToolContent{
			{Type: "text", Text: "Hello, world!"},
		},
	}

	data, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}

	var parsed ToolResult
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	if len(parsed.Content) != 1 {
		t.Fatalf("expected 1 content item, got %d", len(parsed.Content))
	}
	if parsed.Content[0].Type != "text" {
		t.Errorf("Content[0].Type = %s, want text", parsed.Content[0].Type)
	}
	if parsed.Content[0].Text != "Hello, world!" {
		t.Errorf("Content[0].Text = %s, want 'Hello, world!'", parsed.Content[0].Text)
	}
}

func TestToolsListResult(t *testing.T) {
	result := ToolsListResult{
		Tools: []Tool{
			{
				Name:        "tool1",
				Description: "Test tool",
				InputSchema: json.RawMessage(`{"type":"object"}`),
			},
		},
	}

	data, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}

	var parsed ToolsListResult
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	if len(parsed.Tools) != 1 {
		t.Fatalf("expected 1 tool, got %d", len(parsed.Tools))
	}
	if parsed.Tools[0].Name != "tool1" {
		t.Errorf("Tools[0].Name = %s, want tool1", parsed.Tools[0].Name)
	}
}

func TestInitializeResult(t *testing.T) {
	result := InitializeResult{
		ProtocolVersion: "2024-11-05",
		Capabilities: Capabilities{
			Tools: &ToolsCapability{},
		},
		ServerInfo: ServerInfo{
			Name:    "yao-sandbox",
			Version: "1.0.0",
		},
	}

	data, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}

	var parsed InitializeResult
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	if parsed.ProtocolVersion != "2024-11-05" {
		t.Errorf("ProtocolVersion = %s, want 2024-11-05", parsed.ProtocolVersion)
	}
	if parsed.ServerInfo.Name != "yao-sandbox" {
		t.Errorf("ServerInfo.Name = %s, want yao-sandbox", parsed.ServerInfo.Name)
	}
}
