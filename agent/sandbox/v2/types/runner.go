package types

import (
	"context"

	"github.com/yaoapp/gou/connector"
	agentContext "github.com/yaoapp/yao/agent/context"
	"github.com/yaoapp/yao/agent/output/message"
	infra "github.com/yaoapp/yao/sandbox/v2"
)

// Runner is the interface that all sandbox runners must implement.
// A Runner replaces the LLM invocation layer (executeLLMStream) when a
// sandbox is configured.
type Runner interface {
	Name() string
	Prepare(ctx context.Context, req *PrepareRequest) error
	Stream(ctx context.Context, req *StreamRequest, handler message.StreamFunc) error
	Cleanup(ctx context.Context, computer infra.Computer) error
}

// MCPServer mirrors store/types.MCPServerConfig to avoid a cyclic import
// between this leaf package and agent/store/types.
type MCPServer struct {
	ServerID  string   `json:"server_id,omitempty"`
	Resources []string `json:"resources,omitempty"`
	Tools     []string `json:"tools,omitempty"`
}

// RunStepsFunc is the signature of RunPrepareSteps. Workspace is obtained
// internally via computer.Workplace(). assistantDir is the absolute path to
// the assistant source directory on the host; copy steps resolve relative src
// paths against it.
type RunStepsFunc func(ctx context.Context, steps []PrepareStep, computer infra.Computer, assistantID, configHash, assistantDir string) error

// PrepareRequest carries everything needed by Runner.Prepare.
type PrepareRequest struct {
	Computer     infra.Computer
	Config       *SandboxConfig
	Connector    connector.Connector
	SkillsDir    string
	AssistantDir string // absolute host path to the assistant source directory
	MCPServers   []MCPServer
	ConfigHash   string
	RunSteps     RunStepsFunc
}

// StreamRequest carries everything needed by Runner.Stream.
type StreamRequest struct {
	Computer     infra.Computer
	Config       *SandboxConfig
	Connector    connector.Connector
	Messages     []agentContext.Message
	SystemPrompt string
	ChatID       string
	Token        *SandboxToken               // current user's sandbox token for MCP callbacks
	Logger       *agentContext.RequestLogger // request-scoped logger propagated from agent context
}
