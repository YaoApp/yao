package auth

import (
	"fmt"
	"strings"

	"github.com/yaoapp/yao/grpc/pb"
)

// VirtualEndpoint maps a gRPC full method + request to a virtual HTTP endpoint for ACL.
// Returns the HTTP method and path used for scope-based access control.
func VirtualEndpoint(fullMethod string, req interface{}) (method string, path string) {
	switch fullMethod {
	case "/yao.Yao/Run":
		if r, ok := req.(*pb.RunRequest); ok && r.Process != "" {
			return "POST", "/grpc/run/" + r.Process
		}
		return "POST", "/grpc/run/"

	case "/yao.Yao/Stream":
		if r, ok := req.(*pb.RunRequest); ok && r.Process != "" {
			return "POST", "/grpc/stream/" + r.Process
		}
		return "POST", "/grpc/stream/"

	case "/yao.Yao/Shell", "/yao.Yao/ShellStream":
		return "POST", "/grpc/shell"

	case "/yao.Yao/API":
		if r, ok := req.(*pb.APIRequest); ok && r.Path != "" {
			m := strings.ToUpper(r.Method)
			if m == "" {
				m = "POST"
			}
			return m, r.Path
		}
		return "POST", "/"

	case "/yao.Yao/MCPListTools":
		return "GET", "/grpc/mcp/tools"

	case "/yao.Yao/MCPCallTool":
		if r, ok := req.(*pb.MCPCallRequest); ok && r.Tool != "" {
			return "POST", "/grpc/mcp/call/" + r.Tool
		}
		return "POST", "/grpc/mcp/call/"

	case "/yao.Yao/MCPListResources":
		return "GET", "/grpc/mcp/resources"

	case "/yao.Yao/MCPReadResource":
		return "GET", "/grpc/mcp/resources/read"

	case "/yao.Yao/ChatCompletions", "/yao.Yao/ChatCompletionsStream":
		return "POST", "/grpc/llm/completions"

	case "/yao.Yao/AgentStream":
		if r, ok := req.(*pb.AgentRequest); ok && r.AssistantId != "" {
			return "POST", fmt.Sprintf("/grpc/agent/%s", r.AssistantId)
		}
		return "POST", "/grpc/agent/"

	case "/yao.Yao/Heartbeat":
		return "POST", "/grpc/heartbeat"

	case "/tai.tunnel.TaiTunnel/Register":
		return "POST", "/grpc/tai/register"

	case "/tai.tunnel.TaiTunnel/Forward":
		return "POST", "/grpc/tai/forward"

	default:
		return "POST", "/grpc/unknown"
	}
}
