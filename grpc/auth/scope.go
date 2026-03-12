package auth

import "github.com/yaoapp/yao/openapi/oauth/acl"

func init() {
	acl.Register(
		&acl.ScopeDefinition{Name: "grpc:run", Endpoints: []string{"POST /grpc/run/*", "POST /grpc/run/"}},
		&acl.ScopeDefinition{Name: "grpc:stream", Endpoints: []string{"POST /grpc/stream/*", "POST /grpc/stream/"}},
		&acl.ScopeDefinition{Name: "grpc:shell", Endpoints: []string{"POST /grpc/shell"}},
		&acl.ScopeDefinition{Name: "grpc:mcp", Endpoints: []string{"GET /grpc/mcp/tools", "POST /grpc/mcp/call/*", "POST /grpc/mcp/call/", "GET /grpc/mcp/resources", "GET /grpc/mcp/resources/read", "POST /grpc/heartbeat"}},
		&acl.ScopeDefinition{Name: "grpc:llm", Endpoints: []string{"POST /grpc/llm/completions"}},
		&acl.ScopeDefinition{Name: "grpc:agent", Endpoints: []string{"POST /grpc/agent/*", "POST /grpc/agent/"}},
		&acl.ScopeDefinition{Name: "tai:connect", Endpoints: []string{"POST /grpc/tai/register", "POST /grpc/tai/forward"}},
	)
}
