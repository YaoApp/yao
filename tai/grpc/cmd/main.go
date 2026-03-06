package main

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/signal"
	"syscall"

	yaogrpc "github.com/yaoapp/yao/tai/grpc"
)

// Build-time variables set via -ldflags.
var (
	Version   = "dev"
	Commit    = "none"
	BuildTime = "unknown"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "Usage: yao-grpc <version|serve>")
		os.Exit(1)
	}

	switch os.Args[1] {
	case "version":
		fmt.Printf("yao-grpc %s (commit: %s, built: %s)\n", Version, Commit, BuildTime)
	case "serve":
		if err := serve(); err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}
	default:
		fmt.Fprintf(os.Stderr, "Unknown command: %s\nUsage: yao-grpc <version|serve>\n", os.Args[1])
		os.Exit(1)
	}
}

// jsonrpcRequest is a minimal JSON-RPC 2.0 request.
type jsonrpcRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id,omitempty"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

// jsonrpcResponse is a minimal JSON-RPC 2.0 response.
type jsonrpcResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id,omitempty"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   *jsonrpcError   `json:"error,omitempty"`
}

type jsonrpcError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

func serve() error {
	client, err := yaogrpc.NewFromEnv()
	if err != nil {
		return err
	}
	defer client.Close()

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	if sandboxID := os.Getenv("YAO_SANDBOX_ID"); sandboxID != "" {
		go yaogrpc.HeartbeatLoop(ctx, client, sandboxID)
	}

	scanner := bufio.NewScanner(os.Stdin)
	scanner.Buffer(make([]byte, 0, 4*1024*1024), 4*1024*1024)
	encoder := json.NewEncoder(os.Stdout)

	for scanner.Scan() {
		select {
		case <-ctx.Done():
			return nil
		default:
		}

		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}

		var req jsonrpcRequest
		if err := json.Unmarshal(line, &req); err != nil {
			encoder.Encode(jsonrpcResponse{
				JSONRPC: "2.0",
				Error:   &jsonrpcError{Code: -32700, Message: "parse error"},
			})
			continue
		}

		resp := dispatch(ctx, client, &req)
		encoder.Encode(resp)
	}

	if err := scanner.Err(); err != nil && err != io.EOF {
		return fmt.Errorf("stdin read: %w", err)
	}
	return nil
}

func dispatch(ctx context.Context, client *yaogrpc.Client, req *jsonrpcRequest) jsonrpcResponse {
	base := jsonrpcResponse{JSONRPC: "2.0", ID: req.ID}

	switch req.Method {
	case "run":
		return handleRun(ctx, client, req, base)
	case "shell":
		return handleShell(ctx, client, req, base)
	case "mcp/list_tools":
		return handleMCPListTools(ctx, client, req, base)
	case "mcp/call_tool":
		return handleMCPCallTool(ctx, client, req, base)
	case "mcp/list_resources":
		return handleMCPListResources(ctx, client, req, base)
	case "mcp/read_resource":
		return handleMCPReadResource(ctx, client, req, base)
	case "healthz":
		return handleHealthz(ctx, client, base)
	default:
		base.Error = &jsonrpcError{Code: -32601, Message: "method not found: " + req.Method}
		return base
	}
}

// --- handlers ---

type runParams struct {
	Process string          `json:"process"`
	Args    json.RawMessage `json:"args,omitempty"`
	Timeout int32           `json:"timeout,omitempty"`
}

func handleRun(ctx context.Context, c *yaogrpc.Client, req *jsonrpcRequest, base jsonrpcResponse) jsonrpcResponse {
	var p runParams
	if err := json.Unmarshal(req.Params, &p); err != nil {
		base.Error = &jsonrpcError{Code: -32602, Message: "invalid params: " + err.Error()}
		return base
	}
	data, err := c.Run(ctx, p.Process, p.Args, p.Timeout)
	if err != nil {
		base.Error = &jsonrpcError{Code: -32000, Message: err.Error()}
		return base
	}
	base.Result = data
	return base
}

type shellParams struct {
	Command string            `json:"command"`
	Args    []string          `json:"args,omitempty"`
	Env     map[string]string `json:"env,omitempty"`
	Timeout int32             `json:"timeout,omitempty"`
}

func handleShell(ctx context.Context, c *yaogrpc.Client, req *jsonrpcRequest, base jsonrpcResponse) jsonrpcResponse {
	var p shellParams
	if err := json.Unmarshal(req.Params, &p); err != nil {
		base.Error = &jsonrpcError{Code: -32602, Message: "invalid params: " + err.Error()}
		return base
	}
	resp, err := c.Shell(ctx, p.Command, p.Args, p.Env, p.Timeout)
	if err != nil {
		base.Error = &jsonrpcError{Code: -32000, Message: err.Error()}
		return base
	}
	data, _ := json.Marshal(resp)
	base.Result = data
	return base
}

type mcpSessionParams struct {
	SessionID string `json:"session_id"`
}

func handleMCPListTools(ctx context.Context, c *yaogrpc.Client, req *jsonrpcRequest, base jsonrpcResponse) jsonrpcResponse {
	var p mcpSessionParams
	if err := json.Unmarshal(req.Params, &p); err != nil {
		base.Error = &jsonrpcError{Code: -32602, Message: "invalid params: " + err.Error()}
		return base
	}
	data, err := c.MCPListTools(ctx, p.SessionID)
	if err != nil {
		base.Error = &jsonrpcError{Code: -32000, Message: err.Error()}
		return base
	}
	base.Result = data
	return base
}

type mcpCallParams struct {
	SessionID string          `json:"session_id"`
	Tool      string          `json:"tool"`
	Arguments json.RawMessage `json:"arguments,omitempty"`
}

func handleMCPCallTool(ctx context.Context, c *yaogrpc.Client, req *jsonrpcRequest, base jsonrpcResponse) jsonrpcResponse {
	var p mcpCallParams
	if err := json.Unmarshal(req.Params, &p); err != nil {
		base.Error = &jsonrpcError{Code: -32602, Message: "invalid params: " + err.Error()}
		return base
	}
	data, err := c.MCPCallTool(ctx, p.SessionID, p.Tool, p.Arguments)
	if err != nil {
		base.Error = &jsonrpcError{Code: -32000, Message: err.Error()}
		return base
	}
	base.Result = data
	return base
}

func handleMCPListResources(ctx context.Context, c *yaogrpc.Client, req *jsonrpcRequest, base jsonrpcResponse) jsonrpcResponse {
	var p mcpSessionParams
	if err := json.Unmarshal(req.Params, &p); err != nil {
		base.Error = &jsonrpcError{Code: -32602, Message: "invalid params: " + err.Error()}
		return base
	}
	data, err := c.MCPListResources(ctx, p.SessionID)
	if err != nil {
		base.Error = &jsonrpcError{Code: -32000, Message: err.Error()}
		return base
	}
	base.Result = data
	return base
}

type mcpReadParams struct {
	SessionID string `json:"session_id"`
	URI       string `json:"uri"`
}

func handleMCPReadResource(ctx context.Context, c *yaogrpc.Client, req *jsonrpcRequest, base jsonrpcResponse) jsonrpcResponse {
	var p mcpReadParams
	if err := json.Unmarshal(req.Params, &p); err != nil {
		base.Error = &jsonrpcError{Code: -32602, Message: "invalid params: " + err.Error()}
		return base
	}
	data, err := c.MCPReadResource(ctx, p.SessionID, p.URI)
	if err != nil {
		base.Error = &jsonrpcError{Code: -32000, Message: err.Error()}
		return base
	}
	base.Result = data
	return base
}

func handleHealthz(ctx context.Context, c *yaogrpc.Client, base jsonrpcResponse) jsonrpcResponse {
	status, err := c.Healthz(ctx)
	if err != nil {
		base.Error = &jsonrpcError{Code: -32000, Message: err.Error()}
		return base
	}
	data, _ := json.Marshal(map[string]string{"status": status})
	base.Result = data
	return base
}
