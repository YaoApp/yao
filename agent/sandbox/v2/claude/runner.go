package claude

import (
	"context"
	"encoding/json"
	"errors"
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
	streamCompleted bool // set when Stream received "result"; Cleanup skips kill
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
		if err := req.RunSteps(ctx, steps, req.Computer, req.Config.ID, req.ConfigHash, req.AssistantDir); err != nil {
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

	// Kill claude processes only when the context is cancelled externally
	// (upstream timeout, user interrupt) — NOT on normal return.
	go func() {
		<-streamCtx.Done()
		if ctx.Err() == nil {
			fmt.Fprintf(os.Stderr, "[claude] streamCtx done: normal return, skipping kill (ctx.Err=nil)\n")
			return
		}
		fmt.Fprintf(os.Stderr, "[claude] streamCtx done: context cancelled externally (ctx.Err=%v), killing processes\n", ctx.Err())
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
	fmt.Fprintf(os.Stderr, "[claude] parseStreamJSON returned: %v\n", parseErr)

	// Received "result" — Claude finished normally. Return immediately.
	if errors.Is(parseErr, errStreamCompleted) {
		r.streamCompleted = true
		fmt.Fprintf(os.Stderr, "[claude] stream completed normally, returning nil\n")
		return nil
	}

	// Parse failed or stream ended without "result" — wait for process.
	fmt.Fprintf(os.Stderr, "[claude] stream did NOT complete normally, waiting for process exit...\n")
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
		fmt.Fprintf(os.Stderr, "[claude] exit code=%d stderr=%q\n", exitCode, stderrStr)
		if stderrStr != "" {
			return fmt.Errorf("claude CLI exited with code %d: %s", exitCode, stderrStr)
		}
		return fmt.Errorf("claude CLI exited with code %d", exitCode)
	}
	return nil
}

// Cleanup kills any remaining claude processes. If the stream completed
// normally (received "result"), child processes are preserved — the user
// may have asked Claude to launch a browser, server, etc.
func (r *ClaudeRunner) Cleanup(ctx context.Context, computer infra.Computer) error {
	if computer == nil {
		return nil
	}

	if r.streamCompleted {
		fmt.Fprintf(os.Stderr, "[claude] cleanup: stream completed normally, skipping process kill (child processes preserved)\n")
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
	envPrompt := buildSandboxEnvPrompt(oe)
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

// buildSandboxEnvPrompt generates the sandbox environment prompt with system info and working directory.
func buildSandboxEnvPrompt(oe *osEnv) string {
	workDir := oe.WorkDir

	osName := oe.OS
	if osName == "" {
		osName = "linux"
	}
	shell := oe.Shell
	if shell == "" {
		shell = "bash"
	}

	shellNote := ""
	if oe.isWindows() {
		shellNote = `
- **Desktop Environment**: You have full access to the Windows desktop (GUI applications, browsers, etc.)
- **Important**: When you launch GUI applications (browsers, editors, etc.), do NOT close them unless explicitly asked — the user expects them to remain open`
	}

	return fmt.Sprintf(`## Sandbox Environment

- **Operating System**: %[2]s
- **Shell**: %[3]s
- **Working Directory**: %[1]s
- **File Access**: You have full read/write access to %[1]s%[4]s

## User Attachments

User-uploaded files (images, documents, code files, etc.) are placed in %[1]s/.attachments/{chatID}/
Each chat session has its own subdirectory to avoid conflicts.
When the user references an attached file, read it from this directory using the Read or Bash tool.
For image files, you can view them directly as Claude supports vision on local files.
`, workDir, osName, shell, shellNote)
}

var claudeArgWhitelist = map[string]string{
	"max_turns":        "--max-turns",
	"disallowed_tools": "--disallowed-tools",
	"allowed_tools":    "--allowedTools",
}
