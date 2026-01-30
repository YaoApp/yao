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

	// Start with ccr-run if available, otherwise fall back to claude directly
	cmd := []string{"ccr-run"}

	// Add the prompt
	if userPrompt != "" {
		cmd = append(cmd, userPrompt)
	}

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

	// CCR configuration via environment
	// CCR (Claude Code Router) transforms OpenAI-compatible API to Anthropic API format
	if opts.ConnectorHost != "" {
		// CCR expects ANTHROPIC_BASE_URL but will proxy through its own router
		env["CCR_API_BASE"] = opts.ConnectorHost
	}

	if opts.ConnectorKey != "" {
		env["CCR_API_KEY"] = opts.ConnectorKey
	}

	if opts.Model != "" {
		env["CCR_MODEL"] = opts.Model
	}

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

		// permission_mode
		if permMode, ok := opts.Arguments["permission_mode"].(string); ok {
			env["CLAUDE_PERMISSION_MODE"] = permMode
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

// BuildCCRConfig builds the CCR (Claude Code Router) configuration JSON
// CCR requires a specific format with Providers array and Router configuration
func BuildCCRConfig(opts *Options) ([]byte, error) {
	if opts == nil {
		return nil, fmt.Errorf("options is required")
	}

	// Determine provider name based on host
	providerName := "custom"
	apiBaseURL := opts.ConnectorHost
	needsTransformer := false

	if strings.Contains(opts.ConnectorHost, "volces.com") || strings.Contains(opts.ConnectorHost, "volcengine") {
		providerName = "volcengine"
		needsTransformer = true
		// Ensure URL ends with chat/completions
		if !strings.HasSuffix(apiBaseURL, "/chat/completions") {
			apiBaseURL = strings.TrimSuffix(apiBaseURL, "/") + "/chat/completions"
		}
	} else if strings.Contains(opts.ConnectorHost, "deepseek") {
		providerName = "deepseek"
		needsTransformer = true
		if !strings.HasSuffix(apiBaseURL, "/chat/completions") {
			apiBaseURL = strings.TrimSuffix(apiBaseURL, "/") + "/chat/completions"
		}
	} else if strings.Contains(opts.ConnectorHost, "openai.com") {
		providerName = "openai"
		if !strings.HasSuffix(apiBaseURL, "/chat/completions") {
			apiBaseURL = strings.TrimSuffix(apiBaseURL, "/") + "/v1/chat/completions"
		}
	} else if strings.Contains(opts.ConnectorHost, "anthropic.com") {
		providerName = "claude"
	}

	// Build provider configuration
	provider := map[string]interface{}{
		"name":         providerName,
		"api_base_url": apiBaseURL,
		"api_key":      opts.ConnectorKey,
		"models":       []string{opts.Model},
	}

	// Add transformer for providers that need it (DeepSeek, Volcengine)
	if needsTransformer {
		provider["transformer"] = map[string]interface{}{
			"use": []interface{}{
				[]interface{}{"maxtoken", map[string]interface{}{"max_tokens": 16384}},
			},
		}
	}

	// Build router configuration
	routerKey := fmt.Sprintf("%s,%s", providerName, opts.Model)
	router := map[string]interface{}{
		"default":    routerKey,
		"background": routerKey,
		"think":      routerKey,
	}

	// Build full config
	config := map[string]interface{}{
		"LOG":                  true,
		"API_TIMEOUT_MS":       600000,
		"NON_INTERACTIVE_MODE": true,
		"Providers":            []interface{}{provider},
		"Router":               router,
	}

	return json.MarshalIndent(config, "", "  ")
}
