package claude

import (
	"encoding/json"
	"fmt"
	"strings"

	agentContext "github.com/yaoapp/yao/agent/context"
)

// sandboxEnvPrompt is the system prompt injected for sandbox environment
// This tells Claude CLI about the workspace and project structure
const sandboxEnvPrompt = `## Sandbox Environment

You are running in a sandboxed environment with the following setup:

- **Working Directory**: /workspace
- **Project Structure**: If this is a new project, create a dedicated project folder (e.g., /workspace/my-project/) and work inside it
- **File Access**: You have full read/write access to /workspace
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

## GitHub CLI (gh) Usage

When working with GitHub and a token is provided:
1. First authenticate gh CLI using the token: echo "TOKEN" | gh auth login --with-token
2. Then use gh commands normally (gh repo create, gh pr create, etc.)
3. Do NOT use curl to call GitHub API directly - always prefer gh CLI
`

// BuildCommand builds the Claude CLI command and environment variables
// Uses stdin with --input-format stream-json for unlimited prompt length
// isContinuation: if true, uses --continue to resume previous session (only sends last user message)
func BuildCommand(messages []agentContext.Message, opts *Options) ([]string, map[string]string, error) {
	return BuildCommandWithContinuation(messages, opts, false)
}

// BuildCommandWithContinuation builds the Claude CLI command with continuation support
// isContinuation: if true, uses --continue to resume previous session
func BuildCommandWithContinuation(messages []agentContext.Message, opts *Options, isContinuation bool) ([]string, map[string]string, error) {
	// Build system prompt from conversation history (only for first request)
	var systemPrompt string
	if !isContinuation {
		systemPrompt, _ = buildPrompts(messages)
		// Inject sandbox environment prompt
		if systemPrompt != "" {
			systemPrompt = systemPrompt + "\n\n" + sandboxEnvPrompt
		} else {
			systemPrompt = sandboxEnvPrompt
		}
	}

	// Build input JSONL for Claude CLI (stream-json format)
	// For continuation, only send the last user message
	var inputJSONL []byte
	var err error
	if isContinuation {
		inputJSONL, err = BuildLastUserMessageJSONL(messages)
	} else {
		inputJSONL, err = BuildFirstRequestJSONL(messages)
	}
	if err != nil {
		return nil, nil, fmt.Errorf("failed to build input JSONL: %w", err)
	}

	// Build Claude CLI arguments
	var claudeArgs []string

	// Add permission mode (required for MCP tools to work)
	permMode := "bypassPermissions" // default for sandbox
	if opts != nil && opts.Arguments != nil {
		if mode, ok := opts.Arguments["permission_mode"].(string); ok && mode != "" {
			permMode = mode
		}
	}
	claudeArgs = append(claudeArgs, "--dangerously-skip-permissions")
	claudeArgs = append(claudeArgs, "--permission-mode", permMode)

	// Add streaming format flags (required for proper streaming output)
	claudeArgs = append(claudeArgs, "--input-format", "stream-json")
	claudeArgs = append(claudeArgs, "--output-format", "stream-json")
	claudeArgs = append(claudeArgs, "--include-partial-messages") // Enable realtime streaming
	claudeArgs = append(claudeArgs, "--verbose")

	// For continuation, use --continue to resume the previous session
	// Claude CLI will read session data from $HOME/.claude/ (which is /workspace/.claude/)
	if isContinuation {
		claudeArgs = append(claudeArgs, "--continue")
	}

	// Add max_turns if specified
	if opts != nil && opts.Arguments != nil {
		if maxTurns, ok := opts.Arguments["max_turns"]; ok {
			claudeArgs = append(claudeArgs, "--max-turns", fmt.Sprintf("%v", maxTurns))
		}
	}

	// Add MCP config if available
	if opts != nil && len(opts.MCPConfig) > 0 {
		claudeArgs = append(claudeArgs, "--mcp-config", "/workspace/.mcp.json")
		// Allow all tools from the "yao" MCP server
		claudeArgs = append(claudeArgs, "--allowedTools", "mcp__yao__*")
	}

	// Build the full bash command
	// Use heredoc for both system prompt and input JSONL to avoid shell escaping issues
	// System prompt may contain quotes, newlines, special characters that break shell quoting
	var bashCmd strings.Builder

	// Ensure $HOME/.Xauthority exists for PyAutoGUI/Xlib (HOME=/workspace).
	// Xvfb runs without auth, but Xlib requires the file to exist.
	bashCmd.WriteString("touch /home/sandbox/.Xauthority 2>/dev/null; touch \"$HOME/.Xauthority\" 2>/dev/null\n")

	// If we have a system prompt (first request only), write it to a temp file via heredoc first
	// then use --append-system-prompt-file
	if systemPrompt != "" {
		bashCmd.WriteString("cat << 'PROMPTEOF' > /tmp/.system-prompt.txt\n")
		bashCmd.WriteString(systemPrompt)
		bashCmd.WriteString("\nPROMPTEOF\n")
		claudeArgs = append(claudeArgs, "--append-system-prompt-file", "/tmp/.system-prompt.txt")
	}

	// Build claude command with all arguments
	bashCmd.WriteString("cat << 'INPUTEOF' | claude -p")
	for _, arg := range claudeArgs {
		// Quote arguments that might contain special characters
		bashCmd.WriteString(fmt.Sprintf(" %q", arg))
	}
	bashCmd.WriteString("\n")
	bashCmd.WriteString(string(inputJSONL))
	bashCmd.WriteString("\nINPUTEOF")

	cmd := []string{"bash", "-c", bashCmd.String()}

	// Build environment variables
	env := buildEnvironment(opts, systemPrompt)

	return cmd, env, nil
}

// BuildInputJSONL converts messages to Claude CLI stream-json input format
// Deprecated: Use BuildFirstRequestJSONL or BuildLastUserMessageJSONL instead
func BuildInputJSONL(messages []agentContext.Message) ([]byte, error) {
	return BuildFirstRequestJSONL(messages)
}

// BuildFirstRequestJSONL builds JSONL for the first request (all messages)
// Sends all user and assistant messages to establish context
func BuildFirstRequestJSONL(messages []agentContext.Message) ([]byte, error) {
	var lines []string

	for _, msg := range messages {
		// Skip system messages (handled via --system-prompt)
		if msg.Role == "system" {
			continue
		}

		// Build the message content
		var content interface{}
		if msg.Content != nil {
			content = msg.Content
		} else {
			content = ""
		}

		// Create stream-json message
		streamMsg := map[string]interface{}{
			"type": string(msg.Role), // "user" or "assistant"
			"message": map[string]interface{}{
				"role":    string(msg.Role),
				"content": content,
			},
		}

		jsonBytes, err := json.Marshal(streamMsg)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal message: %w", err)
		}
		lines = append(lines, string(jsonBytes))
	}

	return []byte(strings.Join(lines, "\n")), nil
}

// BuildLastUserMessageJSONL builds JSONL with only the last user message
// Used for continuation requests where Claude CLI manages history via --continue
func BuildLastUserMessageJSONL(messages []agentContext.Message) ([]byte, error) {
	// Find the last user message
	var lastUserMessage *agentContext.Message
	for i := len(messages) - 1; i >= 0; i-- {
		if messages[i].Role == "user" {
			lastUserMessage = &messages[i]
			break
		}
	}

	if lastUserMessage == nil {
		return nil, fmt.Errorf("no user message found")
	}

	var content interface{}
	if lastUserMessage.Content != nil {
		content = lastUserMessage.Content
	} else {
		content = ""
	}

	userMsg := map[string]interface{}{
		"type": "user",
		"message": map[string]interface{}{
			"role":    "user",
			"content": content,
		},
	}

	jsonBytes, err := json.Marshal(userMsg)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal user message: %w", err)
	}

	return jsonBytes, nil
}

// buildPrompts extracts system prompt and user prompt from messages
func buildPrompts(messages []agentContext.Message) (systemPrompt string, userPrompt string) {
	var systemParts []string
	var conversationParts []string
	var lastUserMessage string

	for _, msg := range messages {
		switch msg.Role {
		case "system":
			systemParts = append(systemParts, getMessageContent(msg))
		case "user":
			lastUserMessage = getMessageContent(msg)
			conversationParts = append(conversationParts, fmt.Sprintf("User: %s", lastUserMessage))
		case "assistant":
			conversationParts = append(conversationParts, fmt.Sprintf("Assistant: %s", getMessageContent(msg)))
		}
	}

	// Build system prompt with conversation history
	systemPrompt = strings.Join(systemParts, "\n\n")

	// If there's conversation history, include it in the system prompt
	if len(conversationParts) > 1 {
		historySection := "\n\n## Conversation History\n\n" + strings.Join(conversationParts[:len(conversationParts)-1], "\n\n")
		systemPrompt += historySection
	}

	// The user prompt is the last user message
	userPrompt = lastUserMessage

	return systemPrompt, userPrompt
}

// getMessageContent extracts text content from a message
func getMessageContent(msg agentContext.Message) string {
	if msg.Content == nil {
		return ""
	}

	// Handle string content
	if str, ok := msg.Content.(string); ok {
		return str
	}

	// Handle content array (multimodal messages)
	if arr, ok := msg.Content.([]interface{}); ok {
		var parts []string
		for _, item := range arr {
			if m, ok := item.(map[string]interface{}); ok {
				if m["type"] == "text" {
					if text, ok := m["text"].(string); ok {
						parts = append(parts, text)
					}
				}
			}
		}
		return strings.Join(parts, "\n")
	}

	return ""
}

// buildEnvironment builds environment variables for Claude CLI
func buildEnvironment(opts *Options, systemPrompt string) map[string]string {
	env := make(map[string]string)

	if opts == nil {
		return env
	}

	// Set HOME to /workspace so Claude CLI stores session data in the workspace
	// This allows session persistence across requests for the same chat
	// Session data is stored in $HOME/.claude/ (i.e., /workspace/.claude/)
	env["HOME"] = "/workspace"

	// Fix Python user-site-packages: changing HOME from /home/sandbox to /workspace
	// breaks Python's ability to find packages installed via pip --user (e.g., playwright,
	// pyautogui, playwright-stealth) which live in /home/sandbox/.local/lib/pythonX.Y/site-packages/
	env["PYTHONPATH"] = "/home/sandbox/.local/lib/python3.12/site-packages"

	// Fix X11 auth: PyAutoGUI/Xlib looks for $HOME/.Xauthority, but HOME=/workspace
	// so it fails to find /home/sandbox/.Xauthority created during image build.
	// Explicitly set XAUTHORITY to the correct path.
	env["XAUTHORITY"] = "/home/sandbox/.Xauthority"

	// claude-proxy runs on localhost:3456, Claude CLI connects to it
	env["ANTHROPIC_BASE_URL"] = "http://127.0.0.1:3456"
	env["ANTHROPIC_API_KEY"] = "dummy" // Proxy doesn't verify this

	// Pass secrets as environment variables for Claude CLI to use
	// These are configured in package.yao sandbox.secrets (e.g., LLM_API_KEY, GITHUB_TOKEN)
	// start-claude-proxy also exports them for the proxy process, but Claude CLI
	// is launched via a separate docker exec, so it needs them passed explicitly here.
	if len(opts.Secrets) > 0 {
		for k, v := range opts.Secrets {
			env[k] = v
		}
	}

	// Note: System prompt and max_turns are passed via CLI flags in BuildCommand
	// CLAUDE_SYSTEM_PROMPT environment variable is NOT supported by Claude CLI
	// --append-system-prompt or --system-prompt flags must be used instead

	return env
}

// BuildProxyConfig builds the claude-proxy configuration JSON
// This config file is read by start-claude-proxy script in the container
// Config is written to /tmp/.yao/proxy.json (not /workspace/) for security
func BuildProxyConfig(opts *Options) ([]byte, error) {
	if opts == nil {
		return nil, fmt.Errorf("options is required")
	}

	// Build backend URL - ensure it ends with /chat/completions
	backendURL := opts.ConnectorHost
	if !strings.HasSuffix(backendURL, "/chat/completions") {
		backendURL = strings.TrimSuffix(backendURL, "/") + "/chat/completions"
	}

	config := map[string]interface{}{
		"backend": backendURL,
		"api_key": opts.ConnectorKey,
		"model":   opts.Model,
	}

	// Add extra connector options if present (e.g., thinking, max_tokens, temperature)
	// These will be passed to the proxy via CLAUDE_PROXY_OPTIONS environment variable
	if len(opts.ConnectorOptions) > 0 {
		config["options"] = opts.ConnectorOptions
	}

	// Add secrets if present (e.g., GITHUB_TOKEN, AWS_ACCESS_KEY)
	// These will be exported as environment variables for Claude CLI to use
	if len(opts.Secrets) > 0 {
		config["secrets"] = opts.Secrets
	}

	return json.MarshalIndent(config, "", "  ")
}

// BuildCCRConfig is deprecated, kept for backward compatibility
// Use BuildProxyConfig instead
func BuildCCRConfig(opts *Options) ([]byte, error) {
	return BuildProxyConfig(opts)
}
