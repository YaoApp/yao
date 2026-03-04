package mcp_test

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/yaoapp/yao/grpc/pb"
	"github.com/yaoapp/yao/grpc/tests/testutils"
)

const echoSession = "echo"

// --- MCPListTools ---

func TestMCPListTools_Success(t *testing.T) {
	conn := testutils.Prepare(t)
	defer testutils.Clean()

	client := testutils.NewClient(conn)
	token := testutils.ObtainAccessToken(t, "grpc:mcp")
	ctx := testutils.WithToken(context.Background(), token)

	resp, err := client.MCPListTools(ctx, &pb.MCPListRequest{
		SessionId: echoSession,
	})
	assert.NoError(t, err)
	if assert.NotNil(t, resp) {
		assert.NotEmpty(t, resp.Tools)

		var tools []map[string]interface{}
		err := json.Unmarshal(resp.Tools, &tools)
		assert.NoError(t, err)
		assert.GreaterOrEqual(t, len(tools), 3, "echo MCP defines ping, status, echo")

		names := make(map[string]bool)
		for _, tool := range tools {
			if n, ok := tool["name"].(string); ok {
				names[n] = true
			}
		}
		assert.True(t, names["ping"], "should contain ping tool")
		assert.True(t, names["status"], "should contain status tool")
		assert.True(t, names["echo"], "should contain echo tool")
	}
}

func TestMCPListTools_InvalidSession(t *testing.T) {
	conn := testutils.Prepare(t)
	defer testutils.Clean()

	client := testutils.NewClient(conn)
	token := testutils.ObtainAccessToken(t, "grpc:mcp")
	ctx := testutils.WithToken(context.Background(), token)

	_, err := client.MCPListTools(ctx, &pb.MCPListRequest{
		SessionId: "nonexistent-session",
	})
	assert.Error(t, err)
	st, _ := status.FromError(err)
	assert.Equal(t, codes.NotFound, st.Code())
}

// --- MCPCallTool ---

func TestMCPCallTool_Ping(t *testing.T) {
	conn := testutils.Prepare(t)
	defer testutils.Clean()

	client := testutils.NewClient(conn)
	token := testutils.ObtainAccessToken(t, "grpc:mcp")
	ctx := testutils.WithToken(context.Background(), token)

	args, _ := json.Marshal(map[string]interface{}{"count": 2, "message": "ping"})
	resp, err := client.MCPCallTool(ctx, &pb.MCPCallRequest{
		SessionId: echoSession,
		Tool:      "ping",
		Arguments: args,
	})
	assert.NoError(t, err)
	if assert.NotNil(t, resp) {
		assert.NotEmpty(t, resp.Result)
		var result map[string]interface{}
		err := json.Unmarshal(resp.Result, &result)
		assert.NoError(t, err)
	}
}

func TestMCPCallTool_Echo(t *testing.T) {
	conn := testutils.Prepare(t)
	defer testutils.Clean()

	client := testutils.NewClient(conn)
	token := testutils.ObtainAccessToken(t, "grpc:mcp")
	ctx := testutils.WithToken(context.Background(), token)

	args, _ := json.Marshal(map[string]interface{}{"message": "hello", "uppercase": true})
	resp, err := client.MCPCallTool(ctx, &pb.MCPCallRequest{
		SessionId: echoSession,
		Tool:      "echo",
		Arguments: args,
	})
	assert.NoError(t, err)
	if assert.NotNil(t, resp) {
		assert.NotEmpty(t, resp.Result)
	}
}

func TestMCPCallTool_NilArgs(t *testing.T) {
	conn := testutils.Prepare(t)
	defer testutils.Clean()

	client := testutils.NewClient(conn)
	token := testutils.ObtainAccessToken(t, "grpc:mcp")
	ctx := testutils.WithToken(context.Background(), token)

	resp, err := client.MCPCallTool(ctx, &pb.MCPCallRequest{
		SessionId: echoSession,
		Tool:      "ping",
		Arguments: nil,
	})
	assert.NoError(t, err)
	if assert.NotNil(t, resp) {
		assert.NotEmpty(t, resp.Result)
	}
}

func TestMCPCallTool_InvalidSession(t *testing.T) {
	conn := testutils.Prepare(t)
	defer testutils.Clean()

	client := testutils.NewClient(conn)
	token := testutils.ObtainAccessToken(t, "grpc:mcp")
	ctx := testutils.WithToken(context.Background(), token)

	_, err := client.MCPCallTool(ctx, &pb.MCPCallRequest{
		SessionId: "nonexistent-session",
		Tool:      "some-tool",
	})
	assert.Error(t, err)
	st, _ := status.FromError(err)
	assert.Equal(t, codes.NotFound, st.Code())
}

func TestMCPCallTool_BadArgs(t *testing.T) {
	conn := testutils.Prepare(t)
	defer testutils.Clean()

	client := testutils.NewClient(conn)
	token := testutils.ObtainAccessToken(t, "grpc:mcp")
	ctx := testutils.WithToken(context.Background(), token)

	_, err := client.MCPCallTool(ctx, &pb.MCPCallRequest{
		SessionId: echoSession,
		Tool:      "ping",
		Arguments: []byte("{not-json"),
	})
	assert.Error(t, err)
	st, _ := status.FromError(err)
	assert.Equal(t, codes.InvalidArgument, st.Code())
}

// --- MCPListResources ---

func TestMCPListResources_Success(t *testing.T) {
	conn := testutils.Prepare(t)
	defer testutils.Clean()

	client := testutils.NewClient(conn)
	token := testutils.ObtainAccessToken(t, "grpc:mcp")
	ctx := testutils.WithToken(context.Background(), token)

	resp, err := client.MCPListResources(ctx, &pb.MCPListRequest{
		SessionId: echoSession,
	})
	assert.NoError(t, err)
	if assert.NotNil(t, resp) {
		assert.NotEmpty(t, resp.Resources)

		var resources []map[string]interface{}
		err := json.Unmarshal(resp.Resources, &resources)
		assert.NoError(t, err)
		assert.GreaterOrEqual(t, len(resources), 2, "echo MCP defines info and health resources")
	}
}

func TestMCPListResources_InvalidSession(t *testing.T) {
	conn := testutils.Prepare(t)
	defer testutils.Clean()

	client := testutils.NewClient(conn)
	token := testutils.ObtainAccessToken(t, "grpc:mcp")
	ctx := testutils.WithToken(context.Background(), token)

	_, err := client.MCPListResources(ctx, &pb.MCPListRequest{
		SessionId: "nonexistent-session",
	})
	assert.Error(t, err)
	st, _ := status.FromError(err)
	assert.Equal(t, codes.NotFound, st.Code())
}

// --- MCPReadResource ---

func TestMCPReadResource_Success(t *testing.T) {
	conn := testutils.Prepare(t)
	defer testutils.Clean()

	client := testutils.NewClient(conn)
	token := testutils.ObtainAccessToken(t, "grpc:mcp")
	ctx := testutils.WithToken(context.Background(), token)

	resp, err := client.MCPReadResource(ctx, &pb.MCPResourceRequest{
		SessionId: echoSession,
		Uri:       "echo://info",
	})
	assert.NoError(t, err)
	if assert.NotNil(t, resp) {
		assert.NotEmpty(t, resp.Contents)

		var contents []map[string]interface{}
		err := json.Unmarshal(resp.Contents, &contents)
		assert.NoError(t, err)
		assert.Greater(t, len(contents), 0)
	}
}

func TestMCPReadResource_InvalidSession(t *testing.T) {
	conn := testutils.Prepare(t)
	defer testutils.Clean()

	client := testutils.NewClient(conn)
	token := testutils.ObtainAccessToken(t, "grpc:mcp")
	ctx := testutils.WithToken(context.Background(), token)

	_, err := client.MCPReadResource(ctx, &pb.MCPResourceRequest{
		SessionId: "nonexistent-session",
		Uri:       "echo://info",
	})
	assert.Error(t, err)
	st, _ := status.FromError(err)
	assert.Equal(t, codes.NotFound, st.Code())
}

func TestMCPReadResource_NotFoundURI(t *testing.T) {
	conn := testutils.Prepare(t)
	defer testutils.Clean()

	client := testutils.NewClient(conn)
	token := testutils.ObtainAccessToken(t, "grpc:mcp")
	ctx := testutils.WithToken(context.Background(), token)

	_, err := client.MCPReadResource(ctx, &pb.MCPResourceRequest{
		SessionId: echoSession,
		Uri:       "echo://nonexistent",
	})
	assert.Error(t, err)
	st, _ := status.FromError(err)
	assert.Equal(t, codes.Internal, st.Code())
}
