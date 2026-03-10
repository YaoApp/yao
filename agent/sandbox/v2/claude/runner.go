package claude

import (
	"context"
	"encoding/json"
	"fmt"
	"path"
	"strings"
	"time"

	"github.com/yaoapp/gou/connector"
	"github.com/yaoapp/yao/agent/output/message"
	"github.com/yaoapp/yao/agent/sandbox/v2/types"
	infra "github.com/yaoapp/yao/sandbox/v2"
)

const (
	defaultWorkDir   = "/workspace"
	defaultUser      = "sandbox"
	defaultUserHome  = "/home/sandbox"
	defaultProxyPort = 3456
)

// ClaudeRunner implements the Runner interface for Claude CLI (mode=cli).
type ClaudeRunner struct {
	mode            string
	hasMCP          bool
	mcpToolPattern  string // e.g. "mcp__yao__*,mcp__github__*"
	servicePort     int
	servicePath     string
	serviceProtocol string
}

// New creates a new ClaudeRunner.
func New() *ClaudeRunner {
	return &ClaudeRunner{mode: "cli"}
}

func (r *ClaudeRunner) Name() string { return "claude" }

// Prepare executes user-defined and runner-specific prepare steps.
func (r *ClaudeRunner) Prepare(ctx context.Context, req *types.PrepareRequest) error {
	r.mode = req.Config.Runner.Mode
	if r.mode == "" {
		r.mode = "cli"
	}

	workDir := resolveWorkDir(req.Config)

	// Merge user-defined steps with runner-specific steps.
	steps := append([]types.PrepareStep{}, req.Config.Prepare...)

	// Runner-specific: ensure .claude directory in workDir.
	if req.SkillsDir != "" {
		steps = append(steps, types.PrepareStep{
			Action: "exec",
			Cmd:    fmt.Sprintf("mkdir -p %s/.claude", workDir),
			Once:   true,
		})
	}

	// Runner-specific: write MCP config.
	if len(req.MCPServers) > 0 {
		r.hasMCP = true
		r.mcpToolPattern = buildMCPAllowedTools(req.MCPServers)
		mcpJSON := buildMCPConfig(req.MCPServers)
		steps = append(steps, types.PrepareStep{
			Action:  "file",
			Path:    path.Join(workDir, ".mcp.json"),
			Content: mcpJSON,
		})
	}

	// Execute all steps via the injected callback.
	if req.RunSteps != nil && len(steps) > 0 {
		if err := req.RunSteps(ctx, steps, req.Computer, req.Config.ID, req.ConfigHash); err != nil {
			return fmt.Errorf("claude prepare steps: %w", err)
		}
	}

	return nil
}

// Stream executes the Claude CLI and streams output to handler.
func (r *ClaudeRunner) Stream(ctx context.Context, req *types.StreamRequest, handler message.StreamFunc) error {
	computer := req.Computer
	if computer == nil {
		return fmt.Errorf("computer is nil")
	}

	workDir := resolveWorkDir(req.Config)

	// Prepare attachments: resolve __yao.attachment:// URLs, copy files to workspace.
	if req.ChatID != "" {
		ws := computer.Workplace()
		if ws != nil {
			processed, err := prepareAttachments(ctx, req.Messages, req.ChatID, ws)
			if err != nil {
				return fmt.Errorf("prepareAttachments: %w", err)
			}
			req.Messages = processed
		}
	}

	// Detect continuation (existing .claude/projects/ directory).
	isContinuation := hasExistingSession(ctx, computer, workDir)

	// Build CLI command and env.
	cmd, env := r.buildCLICommand(req, isContinuation)

	// Create stream.
	execStream, err := computer.Stream(ctx, cmd, infra.WithWorkDir(workDir), infra.WithEnv(env))
	if err != nil {
		return fmt.Errorf("computer.Stream: %w", err)
	}

	// Monitor for context cancellation — kill the process.
	done := make(chan struct{})
	defer func() {
		close(done)
	}()

	go func() {
		select {
		case <-ctx.Done():
			killCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			computer.Exec(killCtx, []string{"pkill", "-f", "claude"})
			execStream.Cancel()
		case <-done:
		}
	}()

	// Parse streaming output.
	parseErr := parseStreamJSON(ctx, execStream.Stdout, handler)

	// Wait for process exit.
	exitCode, waitErr := execStream.Wait()
	if parseErr != nil {
		return parseErr
	}
	if waitErr != nil {
		return waitErr
	}
	if exitCode != 0 {
		return fmt.Errorf("claude CLI exited with code %d", exitCode)
	}
	return nil
}

// Cleanup kills any remaining claude processes.
// mode=cli: kill all claude CLI processes.
func (r *ClaudeRunner) Cleanup(ctx context.Context, computer infra.Computer) error {
	if computer == nil {
		return nil
	}

	if r.mode != "service" {
		computer.Exec(ctx, []string{"sh", "-c", "pkill -f 'claude' || true"})
	}

	return nil
}

// hasExistingSession checks if a Claude CLI session exists in the workspace.
func hasExistingSession(ctx context.Context, computer infra.Computer, workDir string) bool {
	sessionDir := path.Join(workDir, ".claude/projects")
	result, err := computer.Exec(ctx, []string{"ls", sessionDir})
	if err != nil || result.ExitCode != 0 {
		return false
	}
	return strings.TrimSpace(result.Stdout) != ""
}

// buildCLICommand constructs the Claude CLI command and environment variables.
func (r *ClaudeRunner) buildCLICommand(req *types.StreamRequest, isContinuation bool) ([]string, map[string]string) {
	workDir := resolveWorkDir(req.Config)
	userHome := resolveUserHome(req.Config)

	env := make(map[string]string)
	env["HOME"] = workDir

	// User-specific paths (only set when running as non-root user inside container).
	if userHome != "" {
		env["XAUTHORITY"] = path.Join(userHome, ".Xauthority")
	}

	// Connector environment.
	if req.Connector != nil {
		setting := req.Connector.Setting()
		host, _ := setting["host"].(string)
		key, _ := setting["key"].(string)
		model, _ := setting["model"].(string)

		if req.Connector.Is(connector.ANTHROPIC) {
			env["ANTHROPIC_BASE_URL"] = host
			env["ANTHROPIC_API_KEY"] = key
		} else {
			env["ANTHROPIC_BASE_URL"] = fmt.Sprintf("http://127.0.0.1:%d", defaultProxyPort)
			env["ANTHROPIC_API_KEY"] = "dummy"
		}

		if model != "" {
			env["ANTHROPIC_MODEL"] = model
			env["ANTHROPIC_DEFAULT_OPUS_MODEL"] = model
			env["ANTHROPIC_DEFAULT_SONNET_MODEL"] = model
			env["ANTHROPIC_DEFAULT_HAIKU_MODEL"] = model
			env["CLAUDE_CODE_SUBAGENT_MODEL"] = model
		}
	}

	// Secrets from config.
	if req.Config != nil && len(req.Config.Secrets) > 0 {
		for k, v := range req.Config.Secrets {
			env[k] = v
		}
	}

	// Build system prompt.
	var systemPrompt string
	envPrompt := buildSandboxEnvPrompt(workDir)
	if !isContinuation && req.SystemPrompt != "" {
		systemPrompt = req.SystemPrompt + "\n\n" + envPrompt
	} else if !isContinuation {
		systemPrompt = envPrompt
	}

	// Build input JSONL.
	var inputJSONL string
	if isContinuation {
		inputJSONL = buildLastUserMessageJSONL(req.Messages)
	} else {
		inputJSONL = buildFirstRequestJSONL(req.Messages)
	}

	// CLI args.
	var args []string
	args = append(args, "--dangerously-skip-permissions")
	args = append(args, "--permission-mode", "bypassPermissions")
	args = append(args, "--input-format", "stream-json")
	args = append(args, "--output-format", "stream-json")
	args = append(args, "--include-partial-messages")
	args = append(args, "--verbose")

	if isContinuation {
		args = append(args, "--continue")
	}

	// Runner options pass-through.
	if req.Config != nil && req.Config.Runner.Options != nil {
		for key, val := range req.Config.Runner.Options {
			if flag, ok := claudeArgWhitelist[key]; ok {
				args = append(args, flag, fmt.Sprintf("%v", val))
			}
		}
	}

	// MCP config (set by Prepare if MCPServers were present).
	if r.hasMCP {
		args = append(args, "--mcp-config", path.Join(workDir, ".mcp.json"))
		if r.mcpToolPattern != "" {
			args = append(args, "--allowedTools", r.mcpToolPattern)
		}
	}

	// Build bash command with heredoc.
	var bash strings.Builder
	if userHome != "" {
		bash.WriteString(fmt.Sprintf("touch %s/.Xauthority 2>/dev/null; ", userHome))
	}
	bash.WriteString("touch \"$HOME/.Xauthority\" 2>/dev/null\n")

	if systemPrompt != "" {
		promptFile := path.Join(workDir, ".yao/.system-prompt.txt")
		bash.WriteString(fmt.Sprintf("mkdir -p %s/.yao\n", workDir))
		bash.WriteString(fmt.Sprintf("cat << 'PROMPTEOF' > %s\n", promptFile))
		bash.WriteString(systemPrompt)
		bash.WriteString("\nPROMPTEOF\n")
		args = append(args, "--append-system-prompt-file", promptFile)
	}

	bash.WriteString("cat << 'INPUTEOF' | claude -p")
	for _, arg := range args {
		bash.WriteString(fmt.Sprintf(" %q", arg))
	}
	bash.WriteString(" 2>&1\n")
	bash.WriteString(inputJSONL)
	bash.WriteString("\nINPUTEOF")

	return []string{"bash", "-c", bash.String()}, env
}

// buildMCPConfig creates the .mcp.json for Claude CLI based on declared servers.
// Each server delegates to "tai mcp" which implements the standard MCP protocol
// over stdio and bridges to Yao gRPC with authentication.
// Connection is configured via env vars (YAO_GRPC_ADDR, YAO_TOKEN, etc.)
// injected by the sandbox infrastructure at container start.
func buildMCPConfig(servers []types.MCPServer) []byte {
	mcpServers := make(map[string]any, len(servers))
	for _, s := range servers {
		name := s.ServerID
		if name == "" {
			continue
		}
		mcpServers[name] = map[string]any{
			"command": "tai",
			"args":    []string{"mcp"},
		}
	}
	if len(mcpServers) == 0 {
		mcpServers["yao"] = map[string]any{
			"command": "tai",
			"args":    []string{"mcp"},
		}
	}
	config := map[string]any{"mcpServers": mcpServers}
	data, _ := json.Marshal(config)
	return data
}

// buildMCPAllowedTools generates the --allowedTools pattern from server IDs.
func buildMCPAllowedTools(servers []types.MCPServer) string {
	patterns := make([]string, 0, len(servers))
	for _, s := range servers {
		if s.ServerID != "" {
			patterns = append(patterns, fmt.Sprintf("mcp__%s__*", s.ServerID))
		}
	}
	if len(patterns) == 0 {
		return "mcp__yao__*"
	}
	return strings.Join(patterns, ",")
}

// buildSandboxEnvPrompt generates the sandbox environment prompt with the actual working directory.
func buildSandboxEnvPrompt(workDir string) string {
	return fmt.Sprintf(`## Sandbox Environment

You are running in a sandboxed environment with the following setup:

- **Working Directory**: %[1]s
- **Project Structure**: If this is a new project, create a dedicated project folder (e.g., %[1]s/my-project/) and work inside it
- **File Access**: You have full read/write access to %[1]s
- **Output Files**: Save all output files to the working directory

When creating new projects:
1. Create a project directory with a descriptive name
2. Initialize the project structure inside that directory
3. Keep all related files organized within the project folder

## IMPORTANT: Restricted Tools

The following tools are NOT available in this environment and you must NOT use them:
- EnterPlanMode, ExitPlanMode (use regular text to explain plans instead)
- Task, TaskOutput, TaskStop (complete tasks directly without delegation)
- AskUserQuestion (make reasonable assumptions instead of asking)
- Skill, ToolSearch (not supported)

Focus on using the core tools: Bash, Read, Write, Edit, Glob, Grep, WebSearch, WebFetch.

## User Attachments

User-uploaded files (images, documents, code files, etc.) are placed in %[1]s/.attachments/{chatID}/
Each chat session has its own subdirectory to avoid conflicts.
When the user references an attached file, read it from this directory using the Read or Bash tool.
For image files, you can view them directly as Claude supports vision on local files.

## GitHub CLI (gh) Usage

When working with GitHub and a token is provided:
1. First authenticate gh CLI using the token: echo "TOKEN" | gh auth login --with-token
2. Then use gh commands normally (gh repo create, gh pr create, etc.)
3. Do NOT use curl to call GitHub API directly - always prefer gh CLI
`, workDir)
}

// resolveWorkDir returns the configured working directory, falling back to default.
func resolveWorkDir(cfg *types.SandboxConfig) string {
	if cfg != nil && cfg.Computer.WorkDir != "" {
		return cfg.Computer.WorkDir
	}
	return defaultWorkDir
}

// resolveUserHome returns the home directory for the container user.
// Returns empty string if no user is configured (root or unspecified).
func resolveUserHome(cfg *types.SandboxConfig) string {
	if cfg == nil {
		return defaultUserHome
	}
	user := cfg.Computer.User
	if user == "" {
		user = defaultUser
	}
	if user == "root" {
		return "/root"
	}
	return fmt.Sprintf("/home/%s", user)
}

var claudeArgWhitelist = map[string]string{
	"max_turns":        "--max-turns",
	"disallowed_tools": "--disallowed-tools",
	"allowed_tools":    "--allowedTools",
}
