package assistant

import (
	stdContext "context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	gouMCP "github.com/yaoapp/gou/mcp"
	mcpProcess "github.com/yaoapp/gou/mcp/process"
	"github.com/yaoapp/yao/agent/context"
	"github.com/yaoapp/yao/agent/output/message"
	agentsandbox "github.com/yaoapp/yao/agent/sandbox"
	"github.com/yaoapp/yao/config"
	infraSandbox "github.com/yaoapp/yao/sandbox"
	"github.com/yaoapp/yao/sandbox/ipc"
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
	// Only set if the directory actually exists
	if ast.Path != "" {
		appRoot := config.Conf.AppSource
		skillsDir := filepath.Join(appRoot, ast.Path, "skills")
		if info, err := os.Stat(skillsDir); err == nil && info.IsDir() {
			execOpts.SkillsDir = skillsDir
			ctx.Logger.Debug("Skills directory found: %s", skillsDir)
		}
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

	// Build MCP config and load tools if the assistant has MCP servers configured
	if ast.MCP != nil && len(ast.MCP.Servers) > 0 {
		// Build MCP config for Claude CLI
		mcpConfig, err := ast.BuildMCPConfigForSandbox(ctx)
		if err != nil {
			ctx.Logger.Warn("Failed to build MCP config for sandbox: %v", err)
			// Non-fatal: sandbox can work without MCP
		} else {
			execOpts.MCPConfig = mcpConfig
			ctx.Logger.Debug("MCP config built for sandbox (%d bytes)", len(mcpConfig))
		}

		// Load MCP tools for IPC session
		mcpTools, err := ast.loadMCPToolsForIPC(ctx)
		if err != nil {
			ctx.Logger.Warn("Failed to load MCP tools for IPC: %v", err)
			// Non-fatal: IPC will have no tools
		} else if len(mcpTools) > 0 {
			execOpts.MCPTools = mcpTools
			ctx.Logger.Debug("Loaded %d MCP tools for IPC", len(mcpTools))
		}
	}

	return execOpts, nil
}

// loadMCPToolsForIPC loads MCP tools from configured servers and converts them to IPC format
func (ast *Assistant) loadMCPToolsForIPC(ctx *context.Context) (map[string]*ipc.MCPTool, error) {
	if ast.MCP == nil || len(ast.MCP.Servers) == 0 {
		return nil, nil
	}

	tools := make(map[string]*ipc.MCPTool)
	stdCtx := ctx.Context
	if stdCtx == nil {
		stdCtx = stdContext.Background()
	}

	for _, serverConfig := range ast.MCP.Servers {
		if serverConfig.ServerID == "" {
			continue
		}

		// Get MCP client
		client, err := gouMCP.Select(serverConfig.ServerID)
		if err != nil {
			ctx.Logger.Warn("MCP server '%s' not found: %v", serverConfig.ServerID, err)
			continue
		}

		// List tools from the MCP client
		toolsResp, err := client.ListTools(stdCtx, "")
		if err != nil {
			ctx.Logger.Warn("Failed to list tools from MCP server '%s': %v", serverConfig.ServerID, err)
			continue
		}

		// Get tool mapping for process names
		mapping, ok := mcpProcess.GetMapping(serverConfig.ServerID)
		if !ok {
			ctx.Logger.Warn("No mapping found for MCP server '%s'", serverConfig.ServerID)
			continue
		}

		// Filter tools if specified in config
		toolFilter := make(map[string]bool)
		if len(serverConfig.Tools) > 0 {
			for _, t := range serverConfig.Tools {
				toolFilter[t] = true
			}
		}

		// Convert tools to IPC format
		// Tool names are prefixed with server ID to avoid conflicts
		// e.g., "echo" server's "ping" tool becomes "echo__ping"
		for _, tool := range toolsResp.Tools {
			// Apply tool filter if specified
			if len(toolFilter) > 0 && !toolFilter[tool.Name] {
				continue
			}

			// Find the process name from mapping
			processName := ""
			if toolSchema, ok := mapping.Tools[tool.Name]; ok {
				processName = toolSchema.Process
			}
			if processName == "" {
				ctx.Logger.Warn("No process mapping for tool '%s' in server '%s'", tool.Name, serverConfig.ServerID)
				continue
			}

			// Prefixed tool name: serverID__toolName
			// This matches Claude's MCP naming: mcp__yao__serverID__toolName
			prefixedName := serverConfig.ServerID + "__" + tool.Name

			// Create IPC tool entry with prefixed name
			ipcTool := &ipc.MCPTool{
				Name:        prefixedName,
				Description: tool.Description,
				Process:     processName,
				InputSchema: tool.InputSchema,
			}

			tools[prefixedName] = ipcTool
		}
	}

	return tools, nil
}

// BuildMCPConfigForSandbox builds the MCP configuration JSON for sandbox
// This creates a .mcp.json format that Claude CLI can understand
// Exported for testing
func (ast *Assistant) BuildMCPConfigForSandbox(ctx *context.Context) ([]byte, error) {
	if ast.MCP == nil || len(ast.MCP.Servers) == 0 {
		return nil, nil
	}

	// Build MCP config in Claude CLI format
	// Claude CLI expects: { "mcpServers": { "server_id": { "command": "...", "args": [...] } } }
	//
	// For Yao's MCP servers, we use yao-bridge to connect to the IPC socket.
	// yao-bridge bridges stdio to Unix socket, allowing Claude CLI to communicate
	// with Yao's IPC server running on the host.
	//
	// Architecture:
	//   Claude CLI → yao-bridge → Unix Socket → IPC Session → Yao Process
	config := map[string]interface{}{
		"mcpServers": map[string]interface{}{
			// Single "yao" server that handles all MCP tools via IPC
			"yao": map[string]interface{}{
				"command": "yao-bridge",
				"args":    []string{"/tmp/yao.sock"}, // ContainerIPCSocket from sandbox config
			},
		},
	}

	return json.Marshal(config)
}
