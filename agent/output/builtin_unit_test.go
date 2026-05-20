//go:build unit

package output_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/yaoapp/yao/agent/output"
	"github.com/yaoapp/yao/agent/output/message"
)

func TestNewUserInputMessage(t *testing.T) {
	msg := output.NewUserInputMessage("hello world", "user", "Alice")
	require.NotNil(t, msg)
	assert.Equal(t, message.TypeUserInput, msg.Type)
	assert.Equal(t, "hello world", msg.Props["content"])
	assert.Equal(t, "user", msg.Props["role"])
	assert.Equal(t, "Alice", msg.Props["name"])
}

func TestNewUserInputMessage_EmptyRoleAndName(t *testing.T) {
	msg := output.NewUserInputMessage("test", "", "")
	require.NotNil(t, msg)
	assert.Equal(t, message.TypeUserInput, msg.Type)
	assert.Equal(t, "test", msg.Props["content"])
	_, hasRole := msg.Props["role"]
	_, hasName := msg.Props["name"]
	assert.False(t, hasRole)
	assert.False(t, hasName)
}

func TestNewTextMessage(t *testing.T) {
	msg := output.NewTextMessage("hello")
	require.NotNil(t, msg)
	assert.Equal(t, message.TypeText, msg.Type)
	assert.Equal(t, "hello", msg.Props["content"])
}

func TestNewThinkingMessage(t *testing.T) {
	msg := output.NewThinkingMessage("reasoning about the problem")
	require.NotNil(t, msg)
	assert.Equal(t, message.TypeThinking, msg.Type)
	assert.Equal(t, "reasoning about the problem", msg.Props["content"])
}

func TestNewLoadingMessage(t *testing.T) {
	msg := output.NewLoadingMessage("Searching knowledge base...")
	require.NotNil(t, msg)
	assert.Equal(t, message.TypeLoading, msg.Type)
	assert.Equal(t, "Searching knowledge base...", msg.Props["message"])
}

func TestNewToolCallMessage(t *testing.T) {
	msg := output.NewToolCallMessage("call_123", "get_weather", `{"city":"Beijing"}`)
	require.NotNil(t, msg)
	assert.Equal(t, message.TypeToolCall, msg.Type)
	assert.Equal(t, "call_123", msg.Props["id"])
	assert.Equal(t, "get_weather", msg.Props["name"])
	assert.Equal(t, `{"city":"Beijing"}`, msg.Props["arguments"])
}

func TestNewExecuteMessage_WithInput(t *testing.T) {
	input := map[string]interface{}{"command": "ls -la"}
	msg := output.NewExecuteMessage("Bash", "toolu_abc", input, "running")
	require.NotNil(t, msg)
	assert.Equal(t, message.TypeExecute, msg.Type)
	assert.Equal(t, "Bash", msg.Props["tool"])
	assert.Equal(t, "toolu_abc", msg.Props["tool_id"])
	assert.Equal(t, "running", msg.Props["status"])
	assert.Equal(t, input, msg.Props["input"])
}

func TestNewExecuteMessage_WithoutInput(t *testing.T) {
	msg := output.NewExecuteMessage("Read", "toolu_xyz", nil, "completed")
	require.NotNil(t, msg)
	assert.Equal(t, message.TypeExecute, msg.Type)
	assert.Equal(t, "Read", msg.Props["tool"])
	assert.Equal(t, "toolu_xyz", msg.Props["tool_id"])
	assert.Equal(t, "completed", msg.Props["status"])
	_, hasInput := msg.Props["input"]
	assert.False(t, hasInput)
}

func TestNewErrorMessage(t *testing.T) {
	msg := output.NewErrorMessage("something went wrong", "internal_error")
	require.NotNil(t, msg)
	assert.Equal(t, message.TypeError, msg.Type)
	assert.Equal(t, "something went wrong", msg.Props["message"])
	assert.Equal(t, "internal_error", msg.Props["code"])
}

func TestNewActionMessage(t *testing.T) {
	payload := map[string]interface{}{"panel": "settings", "tab": "general"}
	msg := output.NewActionMessage("open_panel", payload)
	require.NotNil(t, msg)
	assert.Equal(t, message.TypeAction, msg.Type)
	assert.Equal(t, "open_panel", msg.Props["name"])
	assert.Equal(t, payload, msg.Props["payload"])
}

func TestNewEventMessage(t *testing.T) {
	data := map[string]interface{}{"request_id": "req_001"}
	msg := output.NewEventMessage("stream_start", "Stream started", data)
	require.NotNil(t, msg)
	assert.Equal(t, message.TypeEvent, msg.Type)
	assert.Equal(t, "stream_start", msg.Props["event"])
	assert.Equal(t, "Stream started", msg.Props["message"])
	assert.Equal(t, data, msg.Props["data"])
}

func TestNewImageMessage(t *testing.T) {
	msg := output.NewImageMessage("https://example.com/photo.png", "A sunset")
	require.NotNil(t, msg)
	assert.Equal(t, message.TypeImage, msg.Type)
	assert.Equal(t, "https://example.com/photo.png", msg.Props["url"])
	assert.Equal(t, "A sunset", msg.Props["alt"])
}

func TestNewAudioMessage(t *testing.T) {
	msg := output.NewAudioMessage("https://example.com/audio.mp3", "mp3")
	require.NotNil(t, msg)
	assert.Equal(t, message.TypeAudio, msg.Type)
	assert.Equal(t, "https://example.com/audio.mp3", msg.Props["url"])
	assert.Equal(t, "mp3", msg.Props["format"])
}

func TestNewVideoMessage(t *testing.T) {
	msg := output.NewVideoMessage("https://example.com/video.mp4")
	require.NotNil(t, msg)
	assert.Equal(t, message.TypeVideo, msg.Type)
	assert.Equal(t, "https://example.com/video.mp4", msg.Props["url"])
}

func TestIsBuiltinType(t *testing.T) {
	builtinTypes := []string{
		message.TypeUserInput,
		message.TypeText,
		message.TypeThinking,
		message.TypeLoading,
		message.TypeToolCall,
		message.TypeExecute,
		message.TypeError,
		message.TypeAction,
		message.TypeEvent,
		message.TypeImage,
		message.TypeAudio,
		message.TypeVideo,
	}

	for _, typ := range builtinTypes {
		assert.True(t, output.IsBuiltinType(typ), "expected %q to be builtin", typ)
	}

	assert.False(t, output.IsBuiltinType("custom_widget"))
	assert.False(t, output.IsBuiltinType(""))
	assert.False(t, output.IsBuiltinType("unknown"))
}

func TestGenerateID(t *testing.T) {
	id1 := output.GenerateID()
	assert.NotEmpty(t, id1)

	id2 := output.GenerateID()
	assert.NotEmpty(t, id2)
	assert.NotEqual(t, id1, id2)
}
