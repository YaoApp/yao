package claude

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/yaoapp/gou/connector"
	agentContext "github.com/yaoapp/yao/agent/context"
	"github.com/yaoapp/yao/agent/sandbox/v2/types"
	infra "github.com/yaoapp/yao/sandbox/v2"
)

const defaultA2OPort = 3099

type command struct {
	shell   []string
	env     map[string]string
	stdin   []byte
	workDir string
}

func (r *ClaudeRunner) buildCommand(ctx context.Context, req *types.StreamRequest, p platform) command {
	computer := req.Computer
	workDir := computer.GetWorkDir()
	assistantID := ""
	if req.Config != nil {
		assistantID = req.Config.ID
	}

	isContinuation := hasExistingSession(ctx, computer, p, assistantID)
	env := buildEnv(req, p)
	args := buildArgs(req, r, p, isContinuation, assistantID)
	inputJSONL := buildInput(req.Messages, isContinuation)

	var systemPrompt string
	envPrompt := buildSandboxEnvPrompt(p, workDir)
	if !isContinuation && req.SystemPrompt != "" {
		systemPrompt = req.SystemPrompt + "\n\n" + envPrompt
	} else if !isContinuation {
		systemPrompt = envPrompt
	}

	promptFile := p.PathJoin(workDir, ".yao", "assistants", assistantID, "system-prompt.txt")
	if assistantID == "" {
		promptFile = p.PathJoin(workDir, ".yao", ".system-prompt.txt")
	}

	script, stdin := p.BuildScript(scriptInput{
		args:         args,
		systemPrompt: systemPrompt,
		inputJSONL:   inputJSONL,
		workDir:      workDir,
		promptFile:   promptFile,
	})

	return command{
		shell:   p.ShellCmd(script),
		env:     env,
		stdin:   stdin,
		workDir: workDir,
	}
}

func buildEnv(req *types.StreamRequest, p platform) map[string]string {
	env := make(map[string]string)
	workDir := req.Computer.GetWorkDir()

	for k, v := range p.HomeEnv(workDir) {
		env[k] = v
	}

	assistantID := ""
	if req.Config != nil {
		assistantID = req.Config.ID
	}
	if assistantID != "" {
		configDir := p.PathJoin(workDir, ".yao", "assistants", assistantID)
		env["CLAUDE_CONFIG_DIR"] = configDir
	}

	if req.Connector != nil {
		setting := req.Connector.Setting()
		host, _ := setting["host"].(string)
		key, _ := setting["key"].(string)
		model, _ := setting["model"].(string)

		if req.Connector.Is(connector.ANTHROPIC) {
			env["ANTHROPIC_BASE_URL"] = host
			env["ANTHROPIC_API_KEY"] = key
			if model != "" {
				env["ANTHROPIC_MODEL"] = model
				env["ANTHROPIC_DEFAULT_OPUS_MODEL"] = model
				env["ANTHROPIC_DEFAULT_SONNET_MODEL"] = model
				env["ANTHROPIC_DEFAULT_HAIKU_MODEL"] = model
				env["CLAUDE_CODE_SUBAGENT_MODEL"] = model
			}
		} else {
			connectorID := req.Connector.ID()
			env["ANTHROPIC_BASE_URL"] = fmt.Sprintf("http://127.0.0.1:%d/c/%s", defaultA2OPort, connectorID)
			env["ANTHROPIC_API_KEY"] = "dummy"
			// Use a valid Anthropic model name to pass Claude CLI's local
			// validation. The a2o proxy ignores this and substitutes the
			// real backend model from its connector config.
			env["ANTHROPIC_MODEL"] = "claude-sonnet-4-6"
			env["ANTHROPIC_DEFAULT_OPUS_MODEL"] = "claude-sonnet-4-6"
			env["ANTHROPIC_DEFAULT_SONNET_MODEL"] = "claude-sonnet-4-6"
			env["ANTHROPIC_DEFAULT_HAIKU_MODEL"] = "claude-sonnet-4-6"
			env["CLAUDE_CODE_SUBAGENT_MODEL"] = "claude-sonnet-4-6"
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

	return env
}

func buildArgs(req *types.StreamRequest, r *ClaudeRunner, p platform, isContinuation bool, assistantID string) []string {
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
		workDir := req.Computer.GetWorkDir()
		mcpPath := p.PathJoin(workDir, ".yao", "assistants", assistantID, "mcp.json")
		if assistantID == "" {
			mcpPath = p.PathJoin(workDir, ".claude", "mcp.json")
		}
		args = append(args, "--mcp-config", mcpPath)
		if r.mcpToolPattern != "" {
			args = append(args, "--allowedTools", r.mcpToolPattern)
		}
	}

	return args
}

func buildInput(messages []agentContext.Message, isContinuation bool) string {
	if isContinuation {
		return buildLastUserMessageJSONL(messages)
	}
	return buildFirstRequestJSONL(messages)
}

func buildSandboxEnvPrompt(p platform, workDir string) string {
	osName := p.OS()
	if osName == "" {
		osName = "linux"
	}
	shell := p.Shell()
	if shell == "" {
		shell = "bash"
	}

	shellNote := p.EnvPromptNote()

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

func hasExistingSession(ctx context.Context, computer infra.Computer, p platform, assistantID string) bool {
	workDir := computer.GetWorkDir()
	var sessionDir string
	if assistantID != "" {
		configDir := p.PathJoin(workDir, ".yao", "assistants", assistantID)
		sessionDir = p.PathJoin(configDir, "projects")
	} else {
		sessionDir = p.PathJoin(workDir, ".claude", "projects")
	}
	result, err := computer.Exec(ctx, p.ListDirCmd(sessionDir))
	if err != nil || result.ExitCode != 0 {
		return false
	}
	return strings.TrimSpace(result.Stdout) != ""
}

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

// buildFirstRequestJSONL builds the input JSONL for a new (non-continuation)
// Claude CLI session. Per the stream-json input protocol, only user messages
// should be sent; Claude CLI manages its own assistant history internally.
func buildFirstRequestJSONL(messages []agentContext.Message) string {
	var lines []string
	for _, msg := range messages {
		if msg.Role != "user" {
			continue
		}
		content := msg.Content
		if content == nil {
			content = ""
		}
		streamMsg := map[string]any{
			"type": "user",
			"message": map[string]any{
				"role":    "user",
				"content": content,
			},
		}
		data, _ := json.Marshal(streamMsg)
		lines = append(lines, string(data))
	}
	return strings.Join(lines, "\n")
}

func buildLastUserMessageJSONL(messages []agentContext.Message) string {
	for i := len(messages) - 1; i >= 0; i-- {
		if messages[i].Role == "user" {
			content := messages[i].Content
			if content == nil {
				content = ""
			}
			msg := map[string]any{
				"type": "user",
				"message": map[string]any{
					"role":    "user",
					"content": content,
				},
			}
			data, _ := json.Marshal(msg)
			return string(data)
		}
	}
	return ""
}

var claudeArgWhitelist = map[string]string{
	"max_turns":        "--max-turns",
	"disallowed_tools": "--disallowed-tools",
	"allowed_tools":    "--allowedTools",
}
