package ipc

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"net"
	"os"

	"github.com/yaoapp/gou/process"
)

// Close closes the session and cleans up resources
func (s *Session) Close() error {
	if s.cancel != nil {
		s.cancel()
	}
	if s.Conn != nil {
		s.Conn.Close()
	}
	if s.Listener != nil {
		s.Listener.Close()
	}
	// Remove socket file
	os.Remove(s.SocketPath)
	return nil
}

// serve handles incoming connections
func (s *Session) serve(ctx context.Context) {
	defer s.cleanup()

	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		// Accept connection with deadline to allow context cancellation check
		conn, err := s.Listener.Accept()
		if err != nil {
			// Check if context was cancelled
			select {
			case <-ctx.Done():
				return
			default:
				continue
			}
		}

		s.Conn = conn
		s.handleConnection(ctx, conn)
	}
}

// cleanup cleans up session resources
func (s *Session) cleanup() {
	if s.Conn != nil {
		s.Conn.Close()
	}
	if s.Listener != nil {
		s.Listener.Close()
	}
	os.Remove(s.SocketPath)
}

// handleConnection handles a single connection
func (s *Session) handleConnection(ctx context.Context, conn net.Conn) {
	defer conn.Close()

	scanner := bufio.NewScanner(conn)
	// Increase buffer size for large messages
	scanner.Buffer(make([]byte, 64*1024), 1024*1024)

	for scanner.Scan() {
		select {
		case <-ctx.Done():
			return
		default:
		}

		line := scanner.Text()
		if line == "" {
			continue
		}

		response := s.handleMessage(line)
		if response != "" {
			if _, err := conn.Write([]byte(response + "\n")); err != nil {
				// Connection error, stop processing
				return
			}
		}
	}

	// Check for scanner errors (excluding EOF which is normal)
	if err := scanner.Err(); err != nil {
		// Log error but don't return it since this is a goroutine
		// In production, consider adding structured logging
		_ = err
	}
}

// handleMessage processes a single JSON-RPC message
func (s *Session) handleMessage(line string) string {
	var req JSONRPCRequest
	if err := json.Unmarshal([]byte(line), &req); err != nil {
		return s.errorResponse(nil, ErrCodeParse, "Parse error")
	}

	switch req.Method {
	case "initialize":
		return s.handleInitialize(req)
	case "initialized":
		return "" // notification, no response
	case "tools/list":
		return s.handleListTools(req)
	case "tools/call":
		return s.handleCallTool(req)
	case "resources/list":
		return s.handleListResources(req)
	case "resources/read":
		return s.handleReadResource(req)
	default:
		return s.errorResponse(req.ID, ErrCodeMethodNotFound, "Method not found: "+req.Method)
	}
}

// handleInitialize handles the initialize method
func (s *Session) handleInitialize(req JSONRPCRequest) string {
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

	return s.successResponse(req.ID, result)
}

// handleListTools handles the tools/list method
func (s *Session) handleListTools(req JSONRPCRequest) string {
	tools := make([]Tool, 0, len(s.MCPTools))
	for _, mcpTool := range s.MCPTools {
		tools = append(tools, Tool{
			Name:        mcpTool.Name,
			Description: mcpTool.Description,
			InputSchema: mcpTool.InputSchema,
		})
	}

	return s.successResponse(req.ID, ToolsListResult{Tools: tools})
}

// handleCallTool handles the tools/call method
func (s *Session) handleCallTool(req JSONRPCRequest) string {
	var params ToolCallParams
	if err := json.Unmarshal(req.Params, &params); err != nil {
		return s.errorResponse(req.ID, ErrCodeInvalidParams, "Invalid params")
	}

	// Check authorization
	tool, ok := s.MCPTools[params.Name]
	if !ok {
		return s.errorResponse(req.ID, ErrCodeInvalidParams, "Tool not found or not authorized: "+params.Name)
	}

	// Execute Yao Process
	proc := process.New(tool.Process, params.Arguments)

	// Set context if available
	if s.Context != nil {
		// TODO: Set process context with user info
	}

	result, err := proc.Exec()
	if err != nil {
		return s.toolErrorResponse(req.ID, params.Name, err)
	}

	return s.toolSuccessResponse(req.ID, result)
}

// handleListResources handles the resources/list method
func (s *Session) handleListResources(req JSONRPCRequest) string {
	// Return empty resources list for now
	return s.successResponse(req.ID, map[string]interface{}{
		"resources": []interface{}{},
	})
}

// handleReadResource handles the resources/read method
func (s *Session) handleReadResource(req JSONRPCRequest) string {
	return s.errorResponse(req.ID, ErrCodeInvalidParams, "Resource not found")
}

// successResponse creates a JSON-RPC success response
func (s *Session) successResponse(id interface{}, result interface{}) string {
	resp := JSONRPCResponse{
		JSONRPC: "2.0",
		ID:      id,
		Result:  result,
	}
	data, err := json.Marshal(resp)
	if err != nil {
		// Fallback to error response if marshaling fails
		return s.errorResponse(id, ErrCodeInternal, "Failed to marshal response")
	}
	return string(data)
}

// errorResponse creates a JSON-RPC error response
func (s *Session) errorResponse(id interface{}, code int, message string) string {
	resp := JSONRPCResponse{
		JSONRPC: "2.0",
		ID:      id,
		Error: &JSONRPCError{
			Code:    code,
			Message: message,
		},
	}
	data, err := json.Marshal(resp)
	if err != nil {
		// Absolute fallback - manually construct JSON
		return fmt.Sprintf(`{"jsonrpc":"2.0","id":null,"error":{"code":%d,"message":"Internal error"}}`, ErrCodeInternal)
	}
	return string(data)
}

// toolSuccessResponse creates a tool success response
func (s *Session) toolSuccessResponse(id interface{}, result interface{}) string {
	// Convert result to string
	var text string
	switch v := result.(type) {
	case string:
		text = v
	case []byte:
		text = string(v)
	case nil:
		text = "null"
	default:
		data, err := json.Marshal(result)
		if err != nil {
			text = fmt.Sprintf("%v", result)
		} else {
			text = string(data)
		}
	}

	toolResult := ToolResult{
		Content: []ToolContent{
			{Type: "text", Text: text},
		},
	}

	return s.successResponse(id, toolResult)
}

// toolErrorResponse creates a tool error response
func (s *Session) toolErrorResponse(id interface{}, toolName string, err error) string {
	toolResult := ToolResult{
		Content: []ToolContent{
			{Type: "text", Text: fmt.Sprintf("Error executing %s: %v", toolName, err)},
		},
		IsError: true,
	}

	return s.successResponse(id, toolResult)
}
