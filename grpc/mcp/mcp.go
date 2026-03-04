package mcp

import (
	"context"
	"encoding/json"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	goumcp "github.com/yaoapp/gou/mcp"
	"github.com/yaoapp/yao/grpc/pb"
)

// Handler implements the MCP gRPC methods.
type Handler struct{}

// MCPListTools lists all available MCP tools for a given session.
func (h *Handler) MCPListTools(ctx context.Context, req *pb.MCPListRequest) (*pb.MCPListResponse, error) {
	client, err := goumcp.Select(req.SessionId)
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "MCP client not found: %v", err)
	}

	resp, err := client.ListTools(ctx, "")
	if err != nil {
		return nil, status.Errorf(codes.Internal, "ListTools failed: %v", err)
	}

	data, err := json.Marshal(resp.Tools)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to marshal tools: %v", err)
	}

	return &pb.MCPListResponse{Tools: data}, nil
}

// MCPCallTool calls an MCP tool by name with the provided arguments.
func (h *Handler) MCPCallTool(ctx context.Context, req *pb.MCPCallRequest) (*pb.MCPCallResponse, error) {
	client, err := goumcp.Select(req.SessionId)
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "MCP client not found: %v", err)
	}

	var args interface{}
	if len(req.Arguments) > 0 {
		if err := json.Unmarshal(req.Arguments, &args); err != nil {
			return nil, status.Errorf(codes.InvalidArgument, "invalid arguments JSON: %v", err)
		}
	}

	resp, err := client.CallTool(ctx, req.Tool, args)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "CallTool failed: %v", err)
	}

	data, err := json.Marshal(resp)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to marshal result: %v", err)
	}

	return &pb.MCPCallResponse{Result: data}, nil
}

// MCPListResources lists all available MCP resources for a given session.
func (h *Handler) MCPListResources(ctx context.Context, req *pb.MCPListRequest) (*pb.MCPResourcesResponse, error) {
	client, err := goumcp.Select(req.SessionId)
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "MCP client not found: %v", err)
	}

	resp, err := client.ListResources(ctx, "")
	if err != nil {
		return nil, status.Errorf(codes.Internal, "ListResources failed: %v", err)
	}

	data, err := json.Marshal(resp.Resources)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to marshal resources: %v", err)
	}

	return &pb.MCPResourcesResponse{Resources: data}, nil
}

// MCPReadResource reads a specific MCP resource by URI.
func (h *Handler) MCPReadResource(ctx context.Context, req *pb.MCPResourceRequest) (*pb.MCPResourceResponse, error) {
	client, err := goumcp.Select(req.SessionId)
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "MCP client not found: %v", err)
	}

	resp, err := client.ReadResource(ctx, req.Uri)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "ReadResource failed: %v", err)
	}

	data, err := json.Marshal(resp.Contents)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to marshal contents: %v", err)
	}

	return &pb.MCPResourceResponse{Contents: data}, nil
}
