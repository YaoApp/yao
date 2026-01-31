package claude

import (
	"encoding/json"
	"fmt"
	"strings"

	agentContext "github.com/yaoapp/yao/agent/context"
)

// BuildCommand builds the Claude CLI command and environment variables
func BuildCommand(messages []agentContext.Message, opts *Options) ([]string, map[string]string, error) {
	// Build system prompt from conversation history
	systemPrompt, userPrompt := buildPrompts(messages)

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

	// Add MCP config if available
	if opts != nil && len(opts.MCPConfig) > 0 {
		claudeArgs = append(claudeArgs, "--mcp-config", "/workspace/.mcp.json")
		// Allow all tools from the "yao" MCP server
		claudeArgs = append(claudeArgs, "--allowedTools", "mcp__yao__*")
	}

	// Build the full bash command
	// claude-proxy is already started by prepareEnvironment
	bashCmd := "claude -p"
	for _, arg := range claudeArgs {
		// Quote arguments that might contain special characters
		bashCmd += fmt.Sprintf(" %q", arg)
	}
	if userPrompt != "" {
		bashCmd += fmt.Sprintf(" %q", userPrompt)
	}

	cmd := []string{"bash", "-c", bashCmd}

	// Build environment variables
	env := buildEnvironment(opts, systemPrompt)

	return cmd, env, nil
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

	// claude-proxy runs on localhost:3456, Claude CLI connects to it
	env["ANTHROPIC_BASE_URL"] = "http://127.0.0.1:3456"
	env["ANTHROPIC_API_KEY"] = "dummy" // Proxy doesn't verify this

	// Set system prompt via environment (Claude CLI supports this)
	if systemPrompt != "" {
		env["CLAUDE_SYSTEM_PROMPT"] = systemPrompt
	}

	// Additional Claude CLI options from Arguments
	if opts.Arguments != nil {
		// max_turns
		if maxTurns, ok := opts.Arguments["max_turns"]; ok {
			env["CLAUDE_MAX_TURNS"] = fmt.Sprintf("%v", maxTurns)
		}

		// output_format (default to stream-json for streaming)
		if outputFormat, ok := opts.Arguments["output_format"].(string); ok {
			env["CLAUDE_OUTPUT_FORMAT"] = outputFormat
		} else {
			env["CLAUDE_OUTPUT_FORMAT"] = "stream-json"
		}
	} else {
		env["CLAUDE_OUTPUT_FORMAT"] = "stream-json"
	}

	return env
}

// BuildProxyConfig builds the claude-proxy configuration JSON
// This config file is read by start-claude-proxy script in the container
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

	return json.MarshalIndent(config, "", "  ")
}

// BuildCCRConfig is deprecated, kept for backward compatibility
// Use BuildProxyConfig instead
func BuildCCRConfig(opts *Options) ([]byte, error) {
	return BuildProxyConfig(opts)
}
