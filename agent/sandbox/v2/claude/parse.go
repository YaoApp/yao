package claude

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"strings"
	"time"

	goujson "github.com/yaoapp/gou/json"
	agentContext "github.com/yaoapp/yao/agent/context"
	"github.com/yaoapp/yao/agent/output/message"
)

// errStreamCompleted is a sentinel indicating the parser received the terminal
// "result" message. It is NOT a real error — callers should treat it as
// successful completion of the stream.
var errStreamCompleted = errors.New("claude stream completed")

// parseStreamJSON reads stream-json lines from Claude CLI stdout and
// pushes them through handler as standard StreamChunkType events.
func parseStreamJSON(ctx context.Context, stdout io.ReadCloser, handler message.StreamFunc) error {
	// When the context is cancelled (upstream timeout / interrupt), close
	// stdout so that scanner.Scan() unblocks immediately. Without this,
	// a failed TerminateProcess (Access is denied) would leave us stuck
	// forever on the read.
	doneParsing := make(chan struct{})
	defer close(doneParsing)
	go func() {
		select {
		case <-ctx.Done():
			stdout.Close()
		case <-doneParsing:
		}
	}()

	scanner := bufio.NewScanner(stdout)
	buf := make([]byte, 0, 64*1024)
	scanner.Buffer(buf, 1024*1024)

	messageStarted := false
	toolBlockActive := false
	toolIndex := 0

	type toolState struct {
		id        string
		name      string
		index     int
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
						toolID, _ := cb["id"].(string)
						if toolID == "" {
							toolID = fmt.Sprintf("tool_%d_%d", toolIndex, time.Now().UnixNano())
						}
						currentTool = &toolState{id: toolID, name: toolName, index: toolIndex}
						toolIndex++

						if handler != nil {
							if messageStarted {
								handler(message.ChunkMessageEnd, nil)
								messageStarted = false
							}
							if !toolBlockActive {
								startData := message.EventMessageStartData{
									MessageID: fmt.Sprintf("sandbox-tool-%d", time.Now().UnixNano()),
									Type:      "tool_call",
									Timestamp: time.Now().UnixMilli(),
								}
								sd, _ := json.Marshal(startData)
								if handler(message.ChunkMessageStart, sd) != 0 {
									stopped = true
									break
								}
								toolBlockActive = true
							}
							tcData, _ := json.Marshal([]map[string]any{{
								"index": currentTool.index,
								"id":    currentTool.id,
								"type":  "function",
								"function": map[string]any{
									"name":      toolName,
									"arguments": "",
								},
							}})
							if handler(message.ChunkToolCall, tcData) != 0 {
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
							text = strings.ReplaceAll(text, "\r\n", "\n")
							text = strings.ReplaceAll(text, "\r", "\n")
							if handler != nil {
								if toolBlockActive {
									handler(message.ChunkMessageEnd, nil)
									toolBlockActive = false
									messageStarted = false
								}
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
								if handler != nil {
									tcData, _ := json.Marshal([]map[string]any{{
										"index": currentTool.index,
										"function": map[string]any{
											"arguments": partial,
										},
									}})
									if handler(message.ChunkToolCall, tcData) != 0 {
										stopped = true
									}
								}
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

							if itemType == "tool_use" && handler != nil {
								toolName, _ := ci["name"].(string)
								toolID, _ := ci["id"].(string)
								if toolID == "" {
									toolID = fmt.Sprintf("tool_%d_%d", toolIndex, time.Now().UnixNano())
								}
								inputRaw, _ := json.Marshal(ci["input"])
								idx := toolIndex
								toolIndex++

								if !toolBlockActive {
									startData := message.EventMessageStartData{
										MessageID: fmt.Sprintf("sandbox-tool-%d", time.Now().UnixNano()),
										Type:      "tool_call",
										Timestamp: time.Now().UnixMilli(),
									}
									sd, _ := json.Marshal(startData)
									if handler(message.ChunkMessageStart, sd) != 0 {
										stopped = true
										break
									}
									toolBlockActive = true
								}
								tcData, _ := json.Marshal([]map[string]any{{
									"index": idx,
									"id":    toolID,
									"type":  "function",
									"function": map[string]any{
										"name":      toolName,
										"arguments": string(inputRaw),
									},
								}})
								if handler(message.ChunkToolCall, tcData) != 0 {
									stopped = true
									break
								}
							}

							if itemType == "text" {
								if text, ok := ci["text"].(string); ok && text != "" && handler != nil && !messageStarted {
									text = strings.ReplaceAll(text, "\r\n", "\n")
									text = strings.ReplaceAll(text, "\r", "\n")
									if toolBlockActive {
										handler(message.ChunkMessageEnd, nil)
										toolBlockActive = false
									}
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

					// Close any open message from the streaming phase.
					// stream_event text_deltas set messageStarted=true but
					// nothing resets it when the turn ends — the assistant
					// message marks the turn boundary, so we must close
					// the message here to keep state in sync with the
					// stream handler (which already sent message_end).
					if handler != nil {
						if toolBlockActive {
							handler(message.ChunkMessageEnd, nil)
							toolBlockActive = false
						}
						if messageStarted {
							handler(message.ChunkMessageEnd, nil)
							messageStarted = false
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
			if handler != nil {
				if toolBlockActive {
					handler(message.ChunkMessageEnd, nil)
					toolBlockActive = false
				}
				if messageStarted {
					handler(message.ChunkMessageEnd, nil)
				}
			}
			// "result" is the terminal message in Claude CLI's stream-json
			// protocol. Return immediately instead of continuing to
			// scanner.Scan(), which would block forever if the process
			// stays alive (e.g. child processes like chrome.exe keep the
			// stdout pipe open).
			return errStreamCompleted

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

	if err := scanner.Err(); err != nil {
		// If the context was cancelled (upstream timeout / interrupt), the
		// stdout pipe was closed by the goroutine above. The resulting
		// read error is expected — surface it as context.Canceled so the
		// caller can handle it uniformly.
		if ctx.Err() != nil {
			return ctx.Err()
		}
		return err
	}
	return nil
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
