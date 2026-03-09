package claude

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"strings"
	"time"

	goujson "github.com/yaoapp/gou/json"
	agentContext "github.com/yaoapp/yao/agent/context"
	"github.com/yaoapp/yao/agent/output/message"
)

// parseStreamJSON reads stream-json lines from Claude CLI stdout and
// pushes them through handler as standard StreamChunkType events.
func parseStreamJSON(_ context.Context, stdout io.ReadCloser, handler message.StreamFunc) error {
	scanner := bufio.NewScanner(stdout)
	buf := make([]byte, 0, 64*1024)
	scanner.Buffer(buf, 1024*1024)

	messageStarted := false

	type toolState struct {
		name      string
		inputJSON strings.Builder
	}
	var currentTool *toolState

	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}

		var msg map[string]any
		if err := json.Unmarshal([]byte(line), &msg); err != nil {
			continue
		}

		msgType, _ := msg["type"].(string)
		stopped := false

		switch msgType {
		case "system":
			if handler != nil {
				data, _ := json.Marshal(msg)
				if handler(message.ChunkMetadata, data) != 0 {
					stopped = true
				}
			}

		case "stream_event":
			event, _ := msg["event"].(map[string]any)
			if event == nil {
				continue
			}
			eventType, _ := event["type"].(string)

			switch eventType {
			case "content_block_start":
				if cb, ok := event["content_block"].(map[string]any); ok {
					blockType, _ := cb["type"].(string)
					if blockType == "tool_use" {
						toolName, _ := cb["name"].(string)
						currentTool = &toolState{name: toolName}
						if handler != nil {
							data, _ := json.Marshal(map[string]any{"tool": toolName})
							if handler(message.ChunkToolCall, data) != 0 {
								stopped = true
							}
						}
					}
				}

			case "content_block_delta":
				if delta, ok := event["delta"].(map[string]any); ok {
					deltaType, _ := delta["type"].(string)
					switch deltaType {
					case "text_delta":
						if text, ok := delta["text"].(string); ok && text != "" {
							if handler != nil {
								if !messageStarted {
									startData := message.EventMessageStartData{
										MessageID: fmt.Sprintf("sandbox-%d", time.Now().UnixNano()),
										Type:      "text",
										Timestamp: time.Now().UnixMilli(),
									}
									sd, _ := json.Marshal(startData)
									if handler(message.ChunkMessageStart, sd) != 0 {
										stopped = true
										break
									}
									messageStarted = true
								}
								if handler(message.ChunkText, []byte(text)) != 0 {
									stopped = true
								}
							}
						}
					case "input_json_delta":
						if currentTool != nil {
							if partial, ok := delta["partial_json"].(string); ok {
								currentTool.inputJSON.WriteString(partial)
							}
						}
					}
				}

			case "content_block_stop":
				currentTool = nil
			}

		case "assistant":
			if msgData, ok := msg["message"].(map[string]any); ok {
				stopReason, _ := msgData["stop_reason"].(string)
				if stopReason != "" {
					if contentArr, ok := msgData["content"].([]any); ok {
						for _, item := range contentArr {
							ci, ok := item.(map[string]any)
							if !ok {
								continue
							}
							itemType, _ := ci["type"].(string)
							if itemType == "text" {
								if text, ok := ci["text"].(string); ok && text != "" && handler != nil && !messageStarted {
									startData := message.EventMessageStartData{
										MessageID: fmt.Sprintf("sandbox-%d", time.Now().UnixNano()),
										Type:      "text",
										Timestamp: time.Now().UnixMilli(),
									}
									sd, _ := json.Marshal(startData)
									if handler(message.ChunkMessageStart, sd) != 0 {
										stopped = true
										break
									}
									if handler(message.ChunkText, []byte(text)) != 0 {
										stopped = true
										break
									}
									messageStarted = true
								}
							}
						}
					}
				}
			}

		case "result":
			isError, _ := msg["is_error"].(bool)
			if isError {
				if result, ok := msg["result"].(string); ok {
					if handler != nil {
						handler(message.ChunkError, []byte(result))
					}
					return fmt.Errorf("Claude CLI error: %s", result)
				}
			}
			if handler != nil && messageStarted {
				handler(message.ChunkMessageEnd, nil)
			}

		case "error":
			var errMsg string
			switch e := msg["error"].(type) {
			case string:
				errMsg = e
			case map[string]any:
				errMsg, _ = e["message"].(string)
			}
			if errMsg != "" {
				if handler != nil {
					handler(message.ChunkError, []byte(errMsg))
				}
				return fmt.Errorf("Claude CLI error: %s", errMsg)
			}
		}

		if stopped {
			break
		}
	}

	return scanner.Err()
}

// buildFirstRequestJSONL builds JSONL with all messages for the first request.
func buildFirstRequestJSONL(messages []agentContext.Message) string {
	var lines []string
	for _, msg := range messages {
		if msg.Role == "system" {
			continue
		}
		content := msg.Content
		if content == nil {
			content = ""
		}
		streamMsg := map[string]any{
			"type": string(msg.Role),
			"message": map[string]any{
				"role":    string(msg.Role),
				"content": content,
			},
		}
		data, _ := json.Marshal(streamMsg)
		lines = append(lines, string(data))
	}
	return strings.Join(lines, "\n")
}

// buildLastUserMessageJSONL builds JSONL with only the last user message.
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

// Suppress unused import warnings — goujson.Parse is used for tool description
// parsing in V1 and will be used for detailed tool descriptions in future.
var _ = goujson.Parse
var _ = log.Printf
