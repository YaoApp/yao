package claude

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path"
	"strings"
	"time"

	"github.com/yaoapp/gou/connector"
	"github.com/yaoapp/yao/agent/output/message"
	"github.com/yaoapp/yao/agent/sandbox/v2/types"
	infra "github.com/yaoapp/yao/sandbox/v2"
)

const defaultProxyPort = 3456

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

	steps := append([]types.PrepareStep{}, req.Config.Prepare...)

	if req.SkillsDir != "" {
		ws := req.Computer.Workplace()
		if ws != nil {
			src := "local:///" + req.SkillsDir
			dst := ".claude/skills"
			if _, err := ws.Copy(src, dst); err != nil {
				fmt.Fprintf(os.Stderr, "[claude] warn: copy skills %s -> %s: %v\n", src, dst, err)
			}
		}
	}

	if len(req.MCPServers) > 0 {
		r.hasMCP = true
		r.mcpToolPattern = buildMCPAllowedTools(req.MCPServers)
		mcpJSON := buildMCPConfig(req.MCPServers)
		steps = append(steps, types.PrepareStep{
			Action:  "file",
			Path:    ".claude/mcp.json",
			Content: mcpJSON,
		})
	}

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

	oe := resolveOSEnv(computer, req.Config)

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

	isContinuation := hasExistingSession(ctx, computer, oe)

	cmd, env, stdin := r.buildCLICommand(req, oe, isContinuation)

	streamOpts := []infra.ExecOption{infra.WithWorkDir(oe.WorkDir), infra.WithEnv(env)}
	if len(stdin) > 0 {
		streamOpts = append(streamOpts, infra.WithStdin(stdin))
	}

	fmt.Fprintf(os.Stderr, "[claude] Stream cmd=%v hasMCP=%v isContinuation=%v stdinLen=%d workDir=%q\n", cmd, r.hasMCP, isContinuation, len(stdin), oe.WorkDir)

	execStream, err := computer.Stream(ctx, cmd, streamOpts...)
	if err != nil {
		return fmt.Errorf("computer.Stream: %w", err)
	}

	streamCtx, streamCancel := context.WithCancel(ctx)
	defer streamCancel()

	go func() {
		<-streamCtx.Done()
		killCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		computer.Exec(killCtx, oe.killProcessCmd("claude"))
		execStream.Cancel()
	}()

	var stderrBuf strings.Builder
	go func() {
		buf := make([]byte, 4096)
		for {
			n, err := execStream.Stderr.Read(buf)
			if n > 0 {
				stderrBuf.Write(buf[:n])
				chunk := string(buf[:n])
				if strings.Contains(strings.ToLower(chunk), "error") {
					streamCancel()
					io.Copy(&stderrBuf, execStream.Stderr)
					return
				}
			}
			if err != nil {
				return
			}
		}
	}()

	parseErr := parseStreamJSON(streamCtx, execStream.Stdout, handler)

	exitCode, waitErr := execStream.Wait()
	stderrStr := strings.TrimSpace(stderrBuf.String())

	if parseErr != nil {
		if stderrStr != "" {
			return fmt.Errorf("%w (stderr: %s)", parseErr, stderrStr)
		}
		return parseErr
	}
	if waitErr != nil {
		if stderrStr != "" {
			return fmt.Errorf("%w (stderr: %s)", waitErr, stderrStr)
		}
		return waitErr
	}
	if exitCode != 0 {
		fmt.Fprintf(os.Stderr, "[claude] exit code=%d parseErr=%v waitErr=%v stderr=%q\n", exitCode, parseErr, waitErr, stderrStr)
		if stderrStr != "" {
			return fmt.Errorf("claude CLI exited with code %d: %s", exitCode, stderrStr)
		}
		return fmt.Errorf("claude CLI exited with code %d", exitCode)
	}
	return nil
}

// Cleanup kills any remaining claude processes.
func (r *ClaudeRunner) Cleanup(ctx context.Context, computer infra.Computer) error {
	if computer == nil {
		return nil
	}

	if r.mode != "service" {
		oe := resolveOSEnv(computer, nil)
		computer.Exec(ctx, oe.killProcessCmd("claude"))
	}

	return nil
}

// hasExistingSession checks if a Claude CLI session exists in the workspace.
func hasExistingSession(ctx context.Context, computer infra.Computer, oe *osEnv) bool {
	sessionDir := oe.pathJoin(oe.WorkDir, ".claude", "projects")
	result, err := computer.Exec(ctx, oe.listDirCmd(sessionDir))
	if err != nil || result.ExitCode != 0 {
		return false
	}
	return strings.TrimSpace(result.Stdout) != ""
}

// buildCLICommand constructs the Claude CLI command, environment variables, and optional stdin bytes.
func (r *ClaudeRunner) buildCLICommand(req *types.StreamRequest, oe *osEnv, isContinuation bool) ([]string, map[string]string, []byte) {
	env := make(map[string]string)

	if oe.isWindows() {
		env["USERPROFILE"] = oe.WorkDir
		if len(oe.WorkDir) >= 2 && oe.WorkDir[1] == ':' {
			env["HOMEDRIVE"] = oe.WorkDir[:2]
			env["HOMEPATH"] = oe.WorkDir[2:]
		}
	} else {
		env["HOME"] = oe.WorkDir
		if oe.UserHome != "" {
			env["XAUTHORITY"] = path.Join(oe.UserHome, ".Xauthority")
		}
	}

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

		if thinking, ok := setting["thinking"].(map[string]interface{}); ok {
			thinkType, _ := thinking["type"].(string)
			switch thinkType {
			case "disabled":
				env["MAX_THINKING_TOKENS"] = "0"
			case "enabled":
				if budget, ok := thinking["budget_tokens"].(float64); ok && budget > 0 {
					env["MAX_THINKING_TOKENS"] = fmt.Sprintf("%d", int(budget))
				}
			}
		}
	}

	if req.Config != nil && len(req.Config.Secrets) > 0 {
		for k, v := range req.Config.Secrets {
			env[k] = v
		}
	}

	if req.Token != nil {
		if req.Token.Token != "" {
			env["YAO_TOKEN"] = req.Token.Token
		}
		if req.Token.RefreshToken != "" {
			env["YAO_REFRESH_TOKEN"] = req.Token.RefreshToken
		}
	}

	var systemPrompt string
	envPrompt := buildSandboxEnvPrompt(oe.WorkDir)
	if !isContinuation && req.SystemPrompt != "" {
		systemPrompt = req.SystemPrompt + "\n\n" + envPrompt
	} else if !isContinuation {
		systemPrompt = envPrompt
	}

	var inputJSONL string
	if isContinuation {
		inputJSONL = buildLastUserMessageJSONL(req.Messages)
	} else {
		inputJSONL = buildFirstRequestJSONL(req.Messages)
	}

	var args []string

	permMode := ""
	if req.Config != nil && req.Config.Runner.Options != nil {
		if v, ok := req.Config.Runner.Options["permission_mode"]; ok {
			permMode = fmt.Sprintf("%v", v)
		}
	}
	if permMode == "bypassPermissions" {
		args = append(args, "--dangerously-skip-permissions")
		args = append(args, "--permission-mode", permMode)
	}

	args = append(args, "--input-format", "stream-json")
	args = append(args, "--output-format", "stream-json")
	args = append(args, "--include-partial-messages")
	args = append(args, "--verbose")

	if isContinuation {
		args = append(args, "--continue")
	}

	if req.Config != nil && req.Config.Runner.Options != nil {
		for key, val := range req.Config.Runner.Options {
			if flag, ok := claudeArgWhitelist[key]; ok {
				args = append(args, flag, fmt.Sprintf("%v", val))
			}
		}
	}

	if r.hasMCP {
		mcpPath := oe.pathJoin(oe.WorkDir, ".claude", "mcp.json")
		args = append(args, "--mcp-config", mcpPath)
		if r.mcpToolPattern != "" {
			args = append(args, "--allowedTools", r.mcpToolPattern)
		}
	}

	script, stdin := oe.buildCLIScript(args, systemPrompt, inputJSONL)
	return oe.shellCmd(script), env, stdin
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
			"args":    []string{"mcp", name},
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

var claudeArgWhitelist = map[string]string{
	"max_turns":        "--max-turns",
	"disallowed_tools": "--disallowed-tools",
	"allowed_tools":    "--allowedTools",
}
