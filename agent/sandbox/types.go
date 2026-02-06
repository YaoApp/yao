package sandbox

import (
	"context"
	"time"

	agentContext "github.com/yaoapp/yao/agent/context"
	"github.com/yaoapp/yao/agent/output/message"
	infraSandbox "github.com/yaoapp/yao/sandbox"
	"github.com/yaoapp/yao/sandbox/ipc"
)

// Executor executes LLM requests in sandbox
type Executor interface {
	// Execute runs the request and returns response (uses options set at creation time)
	Execute(ctx *agentContext.Context, messages []agentContext.Message) (*agentContext.CompletionResponse, error)

	// Stream runs the request with streaming output (uses options set at creation time)
	Stream(ctx *agentContext.Context, messages []agentContext.Message, handler message.StreamFunc) (*agentContext.CompletionResponse, error)

	// SetLoadingMsgID sets the loading message ID for tool execution status updates
	SetLoadingMsgID(id string)

	// Filesystem operations (for Hooks)
	ReadFile(ctx context.Context, path string) ([]byte, error)
	WriteFile(ctx context.Context, path string, content []byte) error
	ListDir(ctx context.Context, path string) ([]infraSandbox.FileInfo, error)

	// Command execution (for Hooks)
	Exec(ctx context.Context, cmd []string) (string, error)

	// GetWorkDir returns the container workspace directory
	GetWorkDir() string

	// GetSandboxID returns the sandbox ID (userID-chatID)
	GetSandboxID() string

	// GetVNCUrl returns the VNC preview URL path (e.g., /api/__yao/vnc/{sandboxID}/)
	// Returns empty string if VNC is not enabled for this sandbox image
	GetVNCUrl() string

	// Close releases container resources
	Close() error
}

// FileInfo is an alias to infrastructure sandbox FileInfo for convenience
type FileInfo = infraSandbox.FileInfo

// Options for sandbox execution
type Options struct {
	// Command type (claude, cursor)
	Command string `json:"command"`

	// Docker image (optional, auto-selected by command)
	Image string `json:"image,omitempty"`

	// Resource limits
	MaxMemory string  `json:"max_memory,omitempty"`
	MaxCPU    float64 `json:"max_cpu,omitempty"`

	// Execution timeout
	Timeout time.Duration `json:"timeout,omitempty"`

	// Command-specific arguments (passed to CLI)
	Arguments map[string]interface{} `json:"arguments,omitempty"`

	// ========================================
	// Internal fields (auto-resolved by Yao)
	// Do NOT set these in package.yao config
	// ========================================

	// UserID for workspace isolation
	UserID string `json:"-"`

	// ChatID for session isolation
	ChatID string `json:"-"`

	// MCP configuration - auto-loaded from assistants/{name}/mcps/
	MCPConfig []byte `json:"-"`

	// MCPTools - MCP tools to expose via IPC (tool name → tool definition)
	MCPTools map[string]*ipc.MCPTool `json:"-"`

	// Skills directory - auto-resolved to assistants/{name}/skills/
	SkillsDir string `json:"-"`

	// SystemPrompt - extracted from assistant prompts.yml
	// Used to determine if Claude CLI should be called
	SystemPrompt string `json:"-"`

	// Connector settings - auto-resolved from connector config file
	// e.g., connectors/deepseek/v3.conn.yao → host, key, model
	ConnectorHost string `json:"-"`
	ConnectorKey  string `json:"-"`
	Model         string `json:"-"`

	// ConnectorOptions - extra options from connector config (e.g., thinking, max_tokens, temperature)
	// These are backend-specific parameters passed to the proxy
	ConnectorOptions map[string]interface{} `json:"-"`

	// Secrets - sensitive values from sandbox.secrets config (e.g., GITHUB_TOKEN)
	// Resolved from $ENV.XXX references, exported as env vars in container
	Secrets map[string]string `json:"-"`
}

// SandboxConfig represents the sandbox configuration in assistant package.yao
type SandboxConfig struct {
	// Command type (claude, cursor)
	Command string `json:"command" yaml:"command"`

	// Docker image (optional, auto-selected by command)
	Image string `json:"image,omitempty" yaml:"image,omitempty"`

	// Resource limits
	MaxMemory string  `json:"max_memory,omitempty" yaml:"max_memory,omitempty"`
	MaxCPU    float64 `json:"max_cpu,omitempty" yaml:"max_cpu,omitempty"`

	// Execution timeout
	Timeout string `json:"timeout,omitempty" yaml:"timeout,omitempty"`

	// Command-specific arguments (passed to CLI)
	Arguments map[string]interface{} `json:"arguments,omitempty" yaml:"arguments,omitempty"`
}

// DefaultImage returns the default Docker image for a command type
func DefaultImage(command string) string {
	switch command {
	case "claude":
		return "yaoapp/sandbox-claude:latest"
	case "cursor":
		return "yaoapp/sandbox-cursor:latest"
	default:
		return ""
	}
}

// CommandTypes is the list of supported command types
var CommandTypes = []string{"claude", "cursor"}

// IsValidCommand checks if a command type is valid
func IsValidCommand(command string) bool {
	for _, c := range CommandTypes {
		if c == command {
			return true
		}
	}
	return false
}
