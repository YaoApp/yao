package client

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/yaoapp/yao/grpc/pb"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// Client wraps a gRPC connection to a Yao server.
type Client struct {
	conn  *grpc.ClientConn
	svc   pb.YaoClient
	token *TokenManager
}

// NewFromEnv reads YAO_GRPC_ADDR and token env vars, dials the
// gRPC server, and returns a connected Client.
func NewFromEnv() (*Client, error) {
	addr := os.Getenv("YAO_GRPC_ADDR")
	if addr == "" {
		return nil, fmt.Errorf("YAO_GRPC_ADDR is required")
	}

	tm, err := NewTokenManagerFromEnv()
	if err != nil {
		return nil, err
	}

	return Dial(addr, tm)
}

// Dial connects to a Yao gRPC server at addr with the given TokenManager.
func Dial(addr string, tm *TokenManager) (*Client, error) {
	opts := []grpc.DialOption{
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	}
	if tm != nil {
		opts = append(opts,
			grpc.WithUnaryInterceptor(tm.UnaryInterceptor()),
			grpc.WithStreamInterceptor(tm.StreamInterceptor()),
		)
	}

	target := addr
	if !strings.Contains(addr, "://") {
		target = "passthrough:///" + addr
	}

	conn, err := grpc.NewClient(target, opts...)
	if err != nil {
		return nil, fmt.Errorf("dial %s: %w", addr, err)
	}

	return &Client{
		conn:  conn,
		svc:   pb.NewYaoClient(conn),
		token: tm,
	}, nil
}

// Close releases the gRPC connection.
func (c *Client) Close() error {
	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
}

// Conn returns the underlying gRPC connection.
func (c *Client) Conn() *grpc.ClientConn { return c.conn }

// TokenManager returns the client's token manager.
func (c *Client) TokenManager() *TokenManager { return c.token }

// --- Base ---

// Run executes a Yao process and returns the JSON-encoded result.
func (c *Client) Run(ctx context.Context, process string, args []byte, timeout int32) ([]byte, error) {
	resp, err := c.svc.Run(ctx, &pb.RunRequest{
		Process: process,
		Args:    args,
		Timeout: timeout,
	})
	if err != nil {
		return nil, err
	}
	return resp.Data, nil
}

// Shell executes a system command and returns stdout, stderr, exit code.
func (c *Client) Shell(ctx context.Context, command string, args []string, env map[string]string, timeout int32) (*pb.ShellResponse, error) {
	return c.svc.Shell(ctx, &pb.ShellRequest{
		Command: command,
		Args:    args,
		Env:     env,
		Timeout: timeout,
	})
}

// --- API ---

// API proxies an HTTP request through the gRPC gateway.
func (c *Client) API(ctx context.Context, method, path string, headers map[string]string, body []byte) (*pb.APIResponse, error) {
	return c.svc.API(ctx, &pb.APIRequest{
		Method:  method,
		Path:    path,
		Headers: headers,
		Body:    body,
	})
}

// --- MCP ---

// MCPListTools lists available MCP tools for a session.
func (c *Client) MCPListTools(ctx context.Context, sessionID string) ([]byte, error) {
	resp, err := c.svc.MCPListTools(ctx, &pb.MCPListRequest{SessionId: sessionID})
	if err != nil {
		return nil, err
	}
	return resp.Tools, nil
}

// MCPCallTool calls an MCP tool and returns the JSON result.
func (c *Client) MCPCallTool(ctx context.Context, sessionID, tool string, arguments []byte) ([]byte, error) {
	resp, err := c.svc.MCPCallTool(ctx, &pb.MCPCallRequest{
		SessionId: sessionID,
		Tool:      tool,
		Arguments: arguments,
	})
	if err != nil {
		return nil, err
	}
	return resp.Result, nil
}

// MCPListResources lists available MCP resources for a session.
func (c *Client) MCPListResources(ctx context.Context, sessionID string) ([]byte, error) {
	resp, err := c.svc.MCPListResources(ctx, &pb.MCPListRequest{SessionId: sessionID})
	if err != nil {
		return nil, err
	}
	return resp.Resources, nil
}

// MCPReadResource reads an MCP resource by URI.
func (c *Client) MCPReadResource(ctx context.Context, sessionID, uri string) ([]byte, error) {
	resp, err := c.svc.MCPReadResource(ctx, &pb.MCPResourceRequest{
		SessionId: sessionID,
		Uri:       uri,
	})
	if err != nil {
		return nil, err
	}
	return resp.Contents, nil
}

// --- LLM ---

// ChatCompletions sends a chat completion request and returns the result.
func (c *Client) ChatCompletions(ctx context.Context, connector string, messages, options []byte) ([]byte, error) {
	resp, err := c.svc.ChatCompletions(ctx, &pb.ChatRequest{
		Connector: connector,
		Messages:  messages,
		Options:   options,
	})
	if err != nil {
		return nil, err
	}
	return resp.Data, nil
}

// ChatCompletionsStream sends a streaming chat completion request.
func (c *Client) ChatCompletionsStream(ctx context.Context, connector string, messages, options []byte, cb func(data []byte, done bool) error) error {
	stream, err := c.svc.ChatCompletionsStream(ctx, &pb.ChatRequest{
		Connector: connector,
		Messages:  messages,
		Options:   options,
	})
	if err != nil {
		return err
	}
	for {
		chunk, err := stream.Recv()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}
		if err := cb(chunk.Data, chunk.Done); err != nil {
			return err
		}
		if chunk.Done {
			return nil
		}
	}
}

// --- Agent ---

// AgentStream calls an agent with streaming response.
func (c *Client) AgentStream(ctx context.Context, assistantID string, messages, options []byte, cb func(data []byte, done bool) error) error {
	stream, err := c.svc.AgentStream(ctx, &pb.AgentRequest{
		AssistantId: assistantID,
		Messages:    messages,
		Options:     options,
	})
	if err != nil {
		return err
	}
	for {
		chunk, err := stream.Recv()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}
		if err := cb(chunk.Data, chunk.Done); err != nil {
			return err
		}
		if chunk.Done {
			return nil
		}
	}
}

// --- Sandbox ---

// Heartbeat sends a sandbox heartbeat to the Yao gRPC server.
func (c *Client) Heartbeat(ctx context.Context, sandboxID string, cpuPercent int32, memBytes int64, runningProcs int32) (string, error) {
	resp, err := c.svc.Heartbeat(ctx, &pb.HeartbeatRequest{
		SandboxId:    sandboxID,
		CpuPercent:   cpuPercent,
		MemBytes:     memBytes,
		RunningProcs: runningProcs,
	})
	if err != nil {
		return "", err
	}
	return resp.Action, nil
}

// --- Health ---

// Healthz checks the server health.
func (c *Client) Healthz(ctx context.Context) (string, error) {
	resp, err := c.svc.Healthz(ctx, &pb.Empty{})
	if err != nil {
		return "", err
	}
	return resp.Status, nil
}
