package auth_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/yaoapp/yao/grpc/auth"
	"github.com/yaoapp/yao/grpc/pb"
)

func TestVirtualEndpoint_Run(t *testing.T) {
	method, path := auth.VirtualEndpoint("/yao.Yao/Run", &pb.RunRequest{Process: "models.user.Find"})
	assert.Equal(t, "POST", method)
	assert.Equal(t, "/grpc/run/models.user.Find", path)
}

func TestVirtualEndpoint_Stream(t *testing.T) {
	method, path := auth.VirtualEndpoint("/yao.Yao/Stream", &pb.RunRequest{Process: "flows.report"})
	assert.Equal(t, "POST", method)
	assert.Equal(t, "/grpc/stream/flows.report", path)
}

func TestVirtualEndpoint_Shell(t *testing.T) {
	method, path := auth.VirtualEndpoint("/yao.Yao/Shell", &pb.ShellRequest{Command: "ls"})
	assert.Equal(t, "POST", method)
	assert.Equal(t, "/grpc/shell", path)
}

func TestVirtualEndpoint_ShellStream(t *testing.T) {
	method, path := auth.VirtualEndpoint("/yao.Yao/ShellStream", &pb.ShellRequest{Command: "ls"})
	assert.Equal(t, "POST", method)
	assert.Equal(t, "/grpc/shell", path)
}

func TestVirtualEndpoint_API(t *testing.T) {
	method, path := auth.VirtualEndpoint("/yao.Yao/API", &pb.APIRequest{Method: "GET", Path: "/kb/collections"})
	assert.Equal(t, "GET", method)
	assert.Equal(t, "/kb/collections", path)
}

func TestVirtualEndpoint_MCPListTools(t *testing.T) {
	method, path := auth.VirtualEndpoint("/yao.Yao/MCPListTools", &pb.MCPListRequest{SessionId: "abc"})
	assert.Equal(t, "GET", method)
	assert.Equal(t, "/grpc/mcp/tools", path)
}

func TestVirtualEndpoint_MCPCallTool(t *testing.T) {
	method, path := auth.VirtualEndpoint("/yao.Yao/MCPCallTool", &pb.MCPCallRequest{Tool: "search"})
	assert.Equal(t, "POST", method)
	assert.Equal(t, "/grpc/mcp/call/search", path)
}

func TestVirtualEndpoint_MCPListResources(t *testing.T) {
	method, path := auth.VirtualEndpoint("/yao.Yao/MCPListResources", &pb.MCPListRequest{})
	assert.Equal(t, "GET", method)
	assert.Equal(t, "/grpc/mcp/resources", path)
}

func TestVirtualEndpoint_MCPReadResource(t *testing.T) {
	method, path := auth.VirtualEndpoint("/yao.Yao/MCPReadResource", &pb.MCPResourceRequest{Uri: "file://test"})
	assert.Equal(t, "GET", method)
	assert.Equal(t, "/grpc/mcp/resources/read", path)
}

func TestVirtualEndpoint_ChatCompletions(t *testing.T) {
	method, path := auth.VirtualEndpoint("/yao.Yao/ChatCompletions", &pb.ChatRequest{Connector: "openai"})
	assert.Equal(t, "POST", method)
	assert.Equal(t, "/grpc/llm/completions", path)
}

func TestVirtualEndpoint_ChatCompletionsStream(t *testing.T) {
	method, path := auth.VirtualEndpoint("/yao.Yao/ChatCompletionsStream", &pb.ChatRequest{})
	assert.Equal(t, "POST", method)
	assert.Equal(t, "/grpc/llm/completions", path)
}

func TestVirtualEndpoint_AgentStream(t *testing.T) {
	method, path := auth.VirtualEndpoint("/yao.Yao/AgentStream", &pb.AgentRequest{AssistantId: "my-robot"})
	assert.Equal(t, "POST", method)
	assert.Equal(t, "/grpc/agent/my-robot", path)
}

func TestVirtualEndpoint_Unknown(t *testing.T) {
	method, path := auth.VirtualEndpoint("/yao.Yao/NonExistent", nil)
	assert.Equal(t, "POST", method)
	assert.Equal(t, "/grpc/unknown", path)
}

func TestVirtualEndpoint_RunNilReq(t *testing.T) {
	method, path := auth.VirtualEndpoint("/yao.Yao/Run", nil)
	assert.Equal(t, "POST", method)
	assert.Equal(t, "/grpc/run/", path)
}

func TestVirtualEndpoint_RunEmptyProcess(t *testing.T) {
	method, path := auth.VirtualEndpoint("/yao.Yao/Run", &pb.RunRequest{Process: ""})
	assert.Equal(t, "POST", method)
	assert.Equal(t, "/grpc/run/", path)
}

func TestVirtualEndpoint_StreamNilReq(t *testing.T) {
	method, path := auth.VirtualEndpoint("/yao.Yao/Stream", nil)
	assert.Equal(t, "POST", method)
	assert.Equal(t, "/grpc/stream/", path)
}

func TestVirtualEndpoint_APINilReq(t *testing.T) {
	method, path := auth.VirtualEndpoint("/yao.Yao/API", nil)
	assert.Equal(t, "POST", method)
	assert.Equal(t, "/", path)
}

func TestVirtualEndpoint_APIEmptyMethod(t *testing.T) {
	method, path := auth.VirtualEndpoint("/yao.Yao/API", &pb.APIRequest{Method: "", Path: "/test"})
	assert.Equal(t, "POST", method)
	assert.Equal(t, "/test", path)
}

func TestVirtualEndpoint_MCPCallToolNilReq(t *testing.T) {
	method, path := auth.VirtualEndpoint("/yao.Yao/MCPCallTool", nil)
	assert.Equal(t, "POST", method)
	assert.Equal(t, "/grpc/mcp/call/", path)
}

func TestVirtualEndpoint_AgentStreamNilReq(t *testing.T) {
	method, path := auth.VirtualEndpoint("/yao.Yao/AgentStream", nil)
	assert.Equal(t, "POST", method)
	assert.Equal(t, "/grpc/agent/", path)
}

func TestVirtualEndpoint_AgentStreamEmptyID(t *testing.T) {
	method, path := auth.VirtualEndpoint("/yao.Yao/AgentStream", &pb.AgentRequest{AssistantId: ""})
	assert.Equal(t, "POST", method)
	assert.Equal(t, "/grpc/agent/", path)
}

func TestVirtualEndpoint_Heartbeat(t *testing.T) {
	method, path := auth.VirtualEndpoint("/yao.Yao/Heartbeat", &pb.HeartbeatRequest{SandboxId: "sb-1"})
	assert.Equal(t, "POST", method)
	assert.Equal(t, "/grpc/heartbeat", path)
}

func TestVirtualEndpoint_HeartbeatNilReq(t *testing.T) {
	method, path := auth.VirtualEndpoint("/yao.Yao/Heartbeat", nil)
	assert.Equal(t, "POST", method)
	assert.Equal(t, "/grpc/heartbeat", path)
}
