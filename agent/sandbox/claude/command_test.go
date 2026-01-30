package claude

import (
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
	assert.Equal(t, "ccr-run", cmd[0])
	assert.Contains(t, cmd, "Hello") // User prompt should be in command

	// Verify environment variables
	assert.Equal(t, "https://api.example.com", env["CCR_API_BASE"])
	assert.Equal(t, "key123", env["CCR_API_KEY"])
	assert.Equal(t, "test-model", env["CCR_MODEL"])
	assert.Equal(t, "stream-json", env["CLAUDE_OUTPUT_FORMAT"])
}

func TestBuildCommandWithSystemPrompt(t *testing.T) {
	messages := []agentContext.Message{
		{Role: "system", Content: "You are a code reviewer"},
		{Role: "user", Content: "Review this code"},
		{Role: "assistant", Content: "Sure, I'll review it"},
		{Role: "user", Content: "Here is the code"},
	}

	opts := &Options{}

	_, env, err := BuildCommand(messages, opts)
	require.NoError(t, err)

	// System prompt should include conversation history
	assert.Contains(t, env["CLAUDE_SYSTEM_PROMPT"], "You are a code reviewer")
	assert.Contains(t, env["CLAUDE_SYSTEM_PROMPT"], "Conversation History")
}

func TestBuildCommandWithArguments(t *testing.T) {
	messages := []agentContext.Message{
		{Role: "user", Content: "Hello"},
	}

	opts := &Options{
		Arguments: map[string]interface{}{
			"max_turns":       20,
			"permission_mode": "acceptEdits",
			"output_format":   "json",
		},
	}

	_, env, err := BuildCommand(messages, opts)
	require.NoError(t, err)

	assert.Equal(t, "20", env["CLAUDE_MAX_TURNS"])
	assert.Equal(t, "acceptEdits", env["CLAUDE_PERMISSION_MODE"])
	assert.Equal(t, "json", env["CLAUDE_OUTPUT_FORMAT"])
}

func TestBuildCCRConfig(t *testing.T) {
	opts := &Options{
		ConnectorHost: "https://api.example.com",
		ConnectorKey:  "key123",
		Model:         "test-model",
	}

	configJSON, err := BuildCCRConfig(opts)
	require.NoError(t, err)

	configStr := string(configJSON)
	// CCR config uses snake_case for fields
	assert.Contains(t, configStr, "api_base_url")
	assert.Contains(t, configStr, "https://api.example.com")
	assert.Contains(t, configStr, "api_key")
	assert.Contains(t, configStr, "key123")
	assert.Contains(t, configStr, "models")
	assert.Contains(t, configStr, "test-model")
	// Verify new CCR format fields
	assert.Contains(t, configStr, "Providers")
	assert.Contains(t, configStr, "Router")
	assert.Contains(t, configStr, "NON_INTERACTIVE_MODE")
}

func TestBuildCCRConfigVolcengine(t *testing.T) {
	opts := &Options{
		ConnectorHost: "https://ark.cn-beijing.volces.com/api/v3/",
		ConnectorKey:  "test-key",
		Model:         "ep-xxx",
	}

	configJSON, err := BuildCCRConfig(opts)
	require.NoError(t, err)

	configStr := string(configJSON)
	// Verify volcengine-specific configuration
	assert.Contains(t, configStr, "volcengine")
	assert.Contains(t, configStr, "transformer")
	assert.Contains(t, configStr, "maxtoken")
	// URL should end with /chat/completions
	assert.Contains(t, configStr, "/chat/completions")
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
