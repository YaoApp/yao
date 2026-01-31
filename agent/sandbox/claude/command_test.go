package claude

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	agentContext "github.com/yaoapp/yao/agent/context"
)

func TestBuildCommand(t *testing.T) {
	messages := []agentContext.Message{
		{Role: "system", Content: "You are a helpful assistant"},
		{Role: "user", Content: "Hello"},
	}

	opts := &Options{
		ConnectorHost: "https://api.example.com",
		ConnectorKey:  "key123",
		Model:         "test-model",
	}

	cmd, env, err := BuildCommand(messages, opts)
	require.NoError(t, err)

	// Verify command structure
	// Command is now: ["bash", "-c", "cat << 'INPUTEOF' | claude -p ... INPUTEOF"]
	assert.Equal(t, "bash", cmd[0])
	assert.Equal(t, "-c", cmd[1])
	// User message should be in bash command (as JSONL via stdin)
	assert.Contains(t, cmd[2], "Hello")
	// Should have stream-json flags
	assert.Contains(t, cmd[2], "--input-format")
	assert.Contains(t, cmd[2], "--output-format")
	assert.Contains(t, cmd[2], "--include-partial-messages")
	assert.Contains(t, cmd[2], "--verbose")
	assert.Contains(t, cmd[2], "stream-json")

	// Verify environment variables (claude-proxy)
	assert.Equal(t, "http://127.0.0.1:3456", env["ANTHROPIC_BASE_URL"])
	assert.Equal(t, "dummy", env["ANTHROPIC_API_KEY"])
}

func TestBuildCommandWithSystemPrompt(t *testing.T) {
	messages := []agentContext.Message{
		{Role: "system", Content: "You are a code reviewer"},
		{Role: "user", Content: "Review this code"},
		{Role: "assistant", Content: "Sure, I'll review it"},
		{Role: "user", Content: "Here is the code"},
	}

	opts := &Options{}

	cmd, _, err := BuildCommand(messages, opts)
	require.NoError(t, err)

	// System prompt should be written to file via heredoc, then passed via --append-system-prompt-file
	bashCmd := cmd[2] // The bash -c command string
	assert.Contains(t, bashCmd, "cat << 'PROMPTEOF' > /tmp/.system-prompt.txt")
	assert.Contains(t, bashCmd, "You are a code reviewer")
	assert.Contains(t, bashCmd, "PROMPTEOF")
	assert.Contains(t, bashCmd, "--append-system-prompt-file")
	assert.Contains(t, bashCmd, "/tmp/.system-prompt.txt")
}

func TestBuildCommandWithSpecialCharsInPrompt(t *testing.T) {
	// Test that special characters in prompts are handled correctly
	messages := []agentContext.Message{
		{Role: "system", Content: "You are a helper.\n\n## Rules\n- Rule 1: Don't use \"quotes\" wrongly\n- Rule 2: Handle 'single quotes' too\n- Rule 3: Special chars like $VAR and `backticks`"},
		{Role: "user", Content: "Hello"},
	}

	opts := &Options{}

	cmd, _, err := BuildCommand(messages, opts)
	require.NoError(t, err)

	bashCmd := cmd[2]
	// The heredoc approach should preserve all special characters
	assert.Contains(t, bashCmd, "## Rules")
	assert.Contains(t, bashCmd, `Don't use "quotes" wrongly`)
	assert.Contains(t, bashCmd, "'single quotes'")
}

func TestBuildCommandWithArguments(t *testing.T) {
	messages := []agentContext.Message{
		{Role: "user", Content: "Hello"},
	}

	opts := &Options{
		Arguments: map[string]interface{}{
			"max_turns":       20,
			"permission_mode": "acceptEdits",
		},
	}

	cmd, _, err := BuildCommand(messages, opts)
	require.NoError(t, err)

	bashCmd := cmd[2] // The bash -c command string
	// max_turns should be in command args via --max-turns
	assert.Contains(t, bashCmd, "--max-turns")
	assert.Contains(t, bashCmd, "20")
	// permission_mode should be in command args
	assert.Contains(t, bashCmd, "acceptEdits")
}

func TestBuildProxyConfig(t *testing.T) {
	opts := &Options{
		ConnectorHost: "https://api.example.com",
		ConnectorKey:  "key123",
		Model:         "test-model",
	}

	configJSON, err := BuildProxyConfig(opts)
	require.NoError(t, err)

	configStr := string(configJSON)
	// Proxy config uses simple format
	assert.Contains(t, configStr, "backend")
	assert.Contains(t, configStr, "https://api.example.com/chat/completions")
	assert.Contains(t, configStr, "api_key")
	assert.Contains(t, configStr, "key123")
	assert.Contains(t, configStr, "model")
	assert.Contains(t, configStr, "test-model")
}

func TestBuildProxyConfigVolcengine(t *testing.T) {
	opts := &Options{
		ConnectorHost: "https://ark.cn-beijing.volces.com/api/v3/",
		ConnectorKey:  "test-key",
		Model:         "ep-xxx",
	}

	configJSON, err := BuildProxyConfig(opts)
	require.NoError(t, err)

	configStr := string(configJSON)
	// URL should end with /chat/completions
	assert.Contains(t, configStr, "/chat/completions")
	assert.Contains(t, configStr, "ep-xxx")
}

func TestBuildInputJSONL(t *testing.T) {
	messages := []agentContext.Message{
		{Role: "system", Content: "You are helpful"},
		{Role: "user", Content: "Hello"},
		{Role: "assistant", Content: "Hi there!"},
		{Role: "user", Content: "How are you?"},
	}

	jsonl, err := BuildInputJSONL(messages)
	require.NoError(t, err)

	// Should not contain system messages (handled separately)
	assert.NotContains(t, string(jsonl), "You are helpful")

	// Should contain user and assistant messages
	assert.Contains(t, string(jsonl), "Hello")
	assert.Contains(t, string(jsonl), "Hi there!")
	assert.Contains(t, string(jsonl), "How are you?")

	// Verify JSONL format (each line is valid JSON)
	lines := splitLines(string(jsonl))
	for _, line := range lines {
		if line == "" {
			continue
		}
		var msg map[string]interface{}
		err := json.Unmarshal([]byte(line), &msg)
		assert.NoError(t, err, "Line should be valid JSON: %s", line)
		assert.Contains(t, msg, "type")
		assert.Contains(t, msg, "message")
	}
}

func TestBuildInputJSONLMultimodal(t *testing.T) {
	// Test with multimodal content (image)
	messages := []agentContext.Message{
		{
			Role: "user",
			Content: []interface{}{
				map[string]interface{}{"type": "text", "text": "What's in this image?"},
				map[string]interface{}{
					"type": "image",
					"source": map[string]interface{}{
						"type":       "base64",
						"media_type": "image/png",
						"data":       "iVBORw0KGgo=",
					},
				},
			},
		},
	}

	jsonl, err := BuildInputJSONL(messages)
	require.NoError(t, err)

	// Should contain the multimodal content
	assert.Contains(t, string(jsonl), "What's in this image?")
	assert.Contains(t, string(jsonl), "image")
	assert.Contains(t, string(jsonl), "base64")
}

func TestGetMessageContent(t *testing.T) {
	// String content
	msg1 := agentContext.Message{Content: "Hello World"}
	assert.Equal(t, "Hello World", getMessageContent(msg1))

	// Nil content
	msg2 := agentContext.Message{Content: nil}
	assert.Equal(t, "", getMessageContent(msg2))

	// Array content (multimodal)
	msg3 := agentContext.Message{
		Content: []interface{}{
			map[string]interface{}{"type": "text", "text": "Part 1"},
			map[string]interface{}{"type": "text", "text": "Part 2"},
		},
	}
	assert.Contains(t, getMessageContent(msg3), "Part 1")
	assert.Contains(t, getMessageContent(msg3), "Part 2")
}

// Helper to split lines
func splitLines(s string) []string {
	var lines []string
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == '\n' {
			lines = append(lines, s[start:i])
			start = i + 1
		}
	}
	if start < len(s) {
		lines = append(lines, s[start:])
	}
	return lines
}
