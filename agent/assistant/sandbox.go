package assistant

import (
	"fmt"
	"path/filepath"
	"sync"
	"time"

	"github.com/yaoapp/yao/agent/context"
	"github.com/yaoapp/yao/agent/output/message"
	agentsandbox "github.com/yaoapp/yao/agent/sandbox"
	"github.com/yaoapp/yao/config"
	infraSandbox "github.com/yaoapp/yao/sandbox"
	traceTypes "github.com/yaoapp/yao/trace/types"
)

var (
	sandboxManager     *infraSandbox.Manager
	sandboxManagerOnce sync.Once
	sandboxManagerErr  error
)

// GetSandboxManager returns the sandbox manager singleton
// Returns nil and error if sandbox is not configured or Docker is unavailable
func GetSandboxManager() (*infraSandbox.Manager, error) {
	sandboxManagerOnce.Do(func() {
		// Create sandbox config from Yao config
		cfg := &infraSandbox.Config{}

		// Use YAO_DATA_ROOT for workspace and IPC paths
		dataRoot := config.Conf.DataRoot
		if dataRoot != "" {
			cfg.Init(dataRoot)
		}

		// Create manager (will fail if Docker is not available)
		sandboxManager, sandboxManagerErr = infraSandbox.NewManager(cfg)
	})

	return sandboxManager, sandboxManagerErr
}

// HasSandbox returns true if the assistant has sandbox configuration
func (ast *Assistant) HasSandbox() bool {
	return ast.Sandbox != nil && ast.Sandbox.Command != ""
}

// initSandbox initializes the sandbox executor
// Returns the full Executor (for LLM calls), cleanup function, and any error
// This is called BEFORE hooks so that hooks can access ctx.sandbox
// The executor implements both agentsandbox.Executor and context.SandboxExecutor interfaces
func (ast *Assistant) initSandbox(ctx *context.Context, opts *context.Options) (agentsandbox.Executor, func(), error) {
	// Get sandbox manager (singleton)
	manager, err := GetSandboxManager()
	if err != nil {
		ctx.Logger.Error("Sandbox manager initialization failed: %v", err)
		return nil, nil, fmt.Errorf("sandbox manager not available: %w", err)
	}
	if manager == nil {
		return nil, nil, fmt.Errorf("sandbox manager not initialized")
	}

	// Build executor options from assistant config
	execOpts, err := ast.buildSandboxOptions(ctx, opts)
	if err != nil {
		ctx.Logger.Error("Failed to build sandbox options: %v", err)
		return nil, nil, fmt.Errorf("failed to build sandbox options: %w", err)
	}

	// Log sandbox creation
	ctx.Logger.Info("Creating sandbox container for command: %s", ast.Sandbox.Command)

	// Add trace for sandbox creation
	trace, traceErr := ctx.Trace()
	if traceErr == nil && trace != nil {
		trace.Info("Creating sandbox container...")
	}

	// Send loading message to user
	loadingMsg := &message.Message{
		Type: message.TypeLoading,
		Props: map[string]interface{}{
			"message": "Preparing sandbox environment...",
		},
	}
	loadingMsgID, _ := ctx.SendStream(loadingMsg)

	// Create executor (container starts here)
	executor, err := agentsandbox.New(manager, execOpts)
	if err != nil {
		ctx.Logger.Error("Sandbox creation failed: %v", err)
		if traceErr == nil && trace != nil {
			trace.Error("Sandbox creation failed: %v", err)
		}
		// End loading message
		if loadingMsgID != "" {
			ctx.End(loadingMsgID)
		}
		return nil, nil, fmt.Errorf("failed to create sandbox executor: %w", err)
	}

	// Log sandbox ready
	ctx.Logger.Info("Sandbox container ready")
	if traceErr == nil && trace != nil {
		trace.Info("Sandbox container ready")
	}

	// End loading message
	if loadingMsgID != "" {
		ctx.End(loadingMsgID)
	}

	// Return cleanup function
	cleanup := func() {
		if err := executor.Close(); err != nil {
			ctx.Logger.Error("Failed to close sandbox executor: %v", err)
		}
	}

	return executor, cleanup, nil
}

// executeSandboxStream executes the request using sandbox (Claude CLI, etc.)
// This is called when ast.Sandbox is configured
// NOTE: The executor is passed directly from initSandbox, no type assertion needed
func (ast *Assistant) executeSandboxStream(
	ctx *context.Context,
	completionMessages []context.Message,
	agentNode traceTypes.Node,
	streamHandler message.StreamFunc,
	executor agentsandbox.Executor,
) (*context.CompletionResponse, error) {

	// Mark the agentNode as used to avoid unused variable error
	_ = agentNode

	if executor == nil {
		return nil, fmt.Errorf("sandbox executor not initialized (call initSandbox first)")
	}

	// Log sandbox execution
	ctx.Logger.Info("Executing via sandbox (command: %s)", ast.Sandbox.Command)

	// Execute LLM call via sandbox
	resp, err := executor.Stream(ctx, completionMessages, streamHandler)
	if err != nil {
		return nil, fmt.Errorf("sandbox execution failed: %w", err)
	}

	return resp, nil
}

// buildSandboxOptions builds executor options from assistant config
func (ast *Assistant) buildSandboxOptions(ctx *context.Context, opts *context.Options) (*agentsandbox.Options, error) {
	if ast.Sandbox == nil {
		return nil, fmt.Errorf("sandbox configuration is required")
	}

	execOpts := &agentsandbox.Options{
		Command:   ast.Sandbox.Command,
		Image:     ast.Sandbox.Image,
		MaxMemory: ast.Sandbox.MaxMemory,
		MaxCPU:    ast.Sandbox.MaxCPU,
		Arguments: ast.Sandbox.Arguments,
	}

	// Parse timeout string (e.g., "10m") to duration
	if ast.Sandbox.Timeout != "" {
		timeout, err := time.ParseDuration(ast.Sandbox.Timeout)
		if err != nil {
			return nil, fmt.Errorf("invalid timeout format: %w", err)
		}
		execOpts.Timeout = timeout
	}

	// Set user and chat IDs for workspace isolation
	if ctx.Authorized != nil && ctx.Authorized.UserID != "" {
		execOpts.UserID = ctx.Authorized.UserID
	} else {
		execOpts.UserID = "anonymous"
	}
	execOpts.ChatID = ctx.ChatID

	// Set skills directory (auto-resolved from assistant path)
	if ast.Path != "" {
		execOpts.SkillsDir = filepath.Join(ast.Path, "skills")
	}

	// Resolve connector settings
	conn, _, err := ast.GetConnector(ctx, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to get connector: %w", err)
	}

	setting := conn.Setting()
	if host, ok := setting["host"].(string); ok {
		execOpts.ConnectorHost = host
	}
	if key, ok := setting["key"].(string); ok {
		execOpts.ConnectorKey = key
	}
	if model, ok := setting["model"].(string); ok {
		execOpts.Model = model
	}

	// Build MCP config if needed
	// TODO: implement MCP config building for sandbox

	return execOpts, nil
}
