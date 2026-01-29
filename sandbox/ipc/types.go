package ipc

import (
	"context"
	"encoding/json"
	"net"
)

// Session represents an IPC session for a sandbox container
type Session struct {
	ID         string              // Session ID (usually equals chatID)
	SocketPath string              // Unix socket path
	Listener   net.Listener        // Socket listener
	Conn       net.Conn            // Current connection
	Context    *AgentContext       // Agent context
	MCPTools   map[string]*MCPTool // Authorized MCP tools
	cancel     context.CancelFunc  // Cancel function for cleanup
}

// AgentContext holds context information for the agent
type AgentContext struct {
	UserID string // User identifier
	ChatID string // Chat/session identifier
	Locale string // Locale for i18n
}

// MCPTool represents an MCP tool that can be called
type MCPTool struct {
	Name        string          // Tool name
	Description string          // Tool description
	Process     string          // Yao process name to execute
	InputSchema json.RawMessage // JSON Schema for input validation
}

// JSONRPCRequest represents a JSON-RPC 2.0 request
type JSONRPCRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      interface{}     `json:"id,omitempty"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

// JSONRPCResponse represents a JSON-RPC 2.0 response
type JSONRPCResponse struct {
	JSONRPC string        `json:"jsonrpc"`
	ID      interface{}   `json:"id,omitempty"`
	Result  interface{}   `json:"result,omitempty"`
	Error   *JSONRPCError `json:"error,omitempty"`
}

// JSONRPCError represents a JSON-RPC 2.0 error
type JSONRPCError struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

// Standard JSON-RPC error codes
const (
	ErrCodeParse          = -32700 // Parse error
	ErrCodeInvalidRequest = -32600 // Invalid request
	ErrCodeMethodNotFound = -32601 // Method not found
	ErrCodeInvalidParams  = -32602 // Invalid params
	ErrCodeInternal       = -32603 // Internal error
)

// ToolCallParams represents parameters for tools/call
type ToolCallParams struct {
	Name      string                 `json:"name"`
	Arguments map[string]interface{} `json:"arguments"`
}

// ToolResult represents the result of a tool call
type ToolResult struct {
	Content []ToolContent `json:"content"`
	IsError bool          `json:"isError,omitempty"`
}

// ToolContent represents content in a tool result
type ToolContent struct {
	Type string `json:"type"` // "text" or "resource"
	Text string `json:"text,omitempty"`
}

// InitializeParams represents parameters for initialize method
type InitializeParams struct {
	ProtocolVersion string       `json:"protocolVersion"`
	Capabilities    Capabilities `json:"capabilities"`
	ClientInfo      ClientInfo   `json:"clientInfo"`
}

// Capabilities represents MCP capabilities
type Capabilities struct {
	Tools     *ToolsCapability     `json:"tools,omitempty"`
	Resources *ResourcesCapability `json:"resources,omitempty"`
}

// ToolsCapability represents tools capability
type ToolsCapability struct {
	ListChanged bool `json:"listChanged,omitempty"`
}

// ResourcesCapability represents resources capability
type ResourcesCapability struct {
	Subscribe   bool `json:"subscribe,omitempty"`
	ListChanged bool `json:"listChanged,omitempty"`
}

// ClientInfo represents client information
type ClientInfo struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

// ServerInfo represents server information
type ServerInfo struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

// InitializeResult represents the result of initialize
type InitializeResult struct {
	ProtocolVersion string       `json:"protocolVersion"`
	Capabilities    Capabilities `json:"capabilities"`
	ServerInfo      ServerInfo   `json:"serverInfo"`
}

// Tool represents a tool in tools/list response
type Tool struct {
	Name        string          `json:"name"`
	Description string          `json:"description,omitempty"`
	InputSchema json.RawMessage `json:"inputSchema"`
}

// ToolsListResult represents the result of tools/list
type ToolsListResult struct {
	Tools []Tool `json:"tools"`
}
