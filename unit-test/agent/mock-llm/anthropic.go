package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"
)

func handleAnthropic(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	mode := parseMockMode(r)

	switch mode {
	case ModeError429:
		w.Header().Set("Retry-After", "1")
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusTooManyRequests)
		fmt.Fprint(w, `{"type":"error","error":{"type":"rate_limit_error","message":"Rate limit exceeded"}}`)
		return
	case ModeError500:
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprint(w, `{"type":"error","error":{"type":"api_error","message":"Internal server error"}}`)
		return
	case ModeFixture:
		key := fixtureKey(r)
		if key == "" {
			key = "anthropic/default"
		}
		if data, ok := GetFixture(key); ok {
			w.Header().Set("Content-Type", "application/json")
			w.Write(data)
			return
		}
		http.Error(w, fmt.Sprintf(`{"type":"error","error":{"type":"not_found","message":"fixture not found: %s"}}`, key), http.StatusNotFound)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, `{"type":"error","error":{"type":"invalid_request_error","message":"failed to read body"}}`, http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	var req anthropicRequest
	if err := json.Unmarshal(body, &req); err != nil {
		http.Error(w, `{"type":"error","error":{"type":"invalid_request_error","message":"invalid JSON"}}`, http.StatusBadRequest)
		return
	}

	if r.Header.Get(MockModeHeader) == "" {
		mode = detectModeFromModel(req.Model)
	}

	if req.Stream {
		anthropicStreamResponse(w, r, mode, &req)
	} else {
		anthropicNonStreamResponse(w, mode, &req)
	}
}

type anthropicRequest struct {
	Model     string          `json:"model"`
	Messages  json.RawMessage `json:"messages"`
	Stream    bool            `json:"stream"`
	MaxTokens int             `json:"max_tokens"`
	Tools     json.RawMessage `json:"tools,omitempty"`
}

type anthropicResponse struct {
	ID         string             `json:"id"`
	Type       string             `json:"type"`
	Role       string             `json:"role"`
	Content    []anthropicContent `json:"content"`
	Model      string             `json:"model"`
	StopReason string             `json:"stop_reason"`
	Usage      anthropicUsage     `json:"usage"`
}

type anthropicContent struct {
	Type     string          `json:"type"`
	Text     string          `json:"text,omitempty"`
	ID       string          `json:"id,omitempty"`
	Name     string          `json:"name,omitempty"`
	Input    json.RawMessage `json:"input,omitempty"`
	Thinking string          `json:"thinking,omitempty"`
}

type anthropicUsage struct {
	InputTokens  int `json:"input_tokens"`
	OutputTokens int `json:"output_tokens"`
}

func anthropicNonStreamResponse(w http.ResponseWriter, mode MockMode, req *anthropicRequest) {
	if mode == ModeSlow {
		time.Sleep(3 * time.Second)
	}
	content := buildAnthropicContent(mode, req)
	stopReason := "end_turn"

	blocks := []anthropicContent{}
	if mode == ModeReasoning {
		blocks = append(blocks, anthropicContent{
			Type:     "thinking",
			Thinking: "Let me think about this carefully...",
		})
	}
	if mode == ModeToolCall {
		stopReason = "tool_use"
		blocks = append(blocks, anthropicContent{
			Type:  "tool_use",
			ID:    "toolu_mock_001",
			Name:  "get_weather",
			Input: json.RawMessage(`{"location":"San Francisco"}`),
		})
	} else {
		blocks = append(blocks, anthropicContent{Type: "text", Text: content})
	}

	resp := anthropicResponse{
		ID:         "msg-mock-001",
		Type:       "message",
		Role:       "assistant",
		Content:    blocks,
		Model:      req.Model,
		StopReason: stopReason,
		Usage:      anthropicUsage{InputTokens: 10, OutputTokens: 20},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func anthropicStreamResponse(w http.ResponseWriter, r *http.Request, mode MockMode, req *anthropicRequest) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming not supported", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	sendEvent := func(event, data string) {
		fmt.Fprintf(w, "event: %s\ndata: %s\n\n", event, data)
		flusher.Flush()
	}

	// message_start
	sendEvent("message_start", fmt.Sprintf(
		`{"type":"message_start","message":{"id":"msg-mock-stream-001","type":"message","role":"assistant","content":[],"model":"%s","stop_reason":null,"usage":{"input_tokens":10,"output_tokens":0}}}`,
		req.Model,
	))

	blockIdx := 0

	if mode == ModeReasoning {
		sendEvent("content_block_start", fmt.Sprintf(
			`{"type":"content_block_start","index":%d,"content_block":{"type":"thinking","thinking":""}}`, blockIdx,
		))
		for _, part := range []string{"Let me ", "think ", "carefully..."} {
			sendEvent("content_block_delta", fmt.Sprintf(
				`{"type":"content_block_delta","index":%d,"delta":{"type":"thinking_delta","thinking":"%s"}}`, blockIdx, part,
			))
			time.Sleep(10 * time.Millisecond)
		}
		sendEvent("content_block_stop", fmt.Sprintf(`{"type":"content_block_stop","index":%d}`, blockIdx))
		blockIdx++
	}

	if mode == ModeToolCall {
		sendEvent("content_block_start", fmt.Sprintf(
			`{"type":"content_block_start","index":%d,"content_block":{"type":"tool_use","id":"toolu_mock_001","name":"get_weather","input":{}}}`, blockIdx,
		))
		sendEvent("content_block_delta", fmt.Sprintf(
			`{"type":"content_block_delta","index":%d,"delta":{"type":"input_json_delta","partial_json":"{\"location\":\"San Francisco\"}"}}`, blockIdx,
		))
		sendEvent("content_block_stop", fmt.Sprintf(`{"type":"content_block_stop","index":%d}`, blockIdx))

		sendEvent("message_delta", `{"type":"message_delta","delta":{"stop_reason":"tool_use"},"usage":{"output_tokens":20}}`)
	} else {
		sendEvent("content_block_start", fmt.Sprintf(
			`{"type":"content_block_start","index":%d,"content_block":{"type":"text","text":""}}`, blockIdx,
		))

		content := buildAnthropicContent(mode, req)
		if mode == ModeSlow {
			time.Sleep(3 * time.Second)
		}
		words := strings.Fields(content)
		for _, word := range words {
			sendEvent("content_block_delta", fmt.Sprintf(
				`{"type":"content_block_delta","index":%d,"delta":{"type":"text_delta","text":"%s "}}`, blockIdx, word,
			))
			time.Sleep(10 * time.Millisecond)
		}
		sendEvent("content_block_stop", fmt.Sprintf(`{"type":"content_block_stop","index":%d}`, blockIdx))

		sendEvent("message_delta", `{"type":"message_delta","delta":{"stop_reason":"end_turn"},"usage":{"output_tokens":20}}`)
	}

	sendEvent("message_stop", `{"type":"message_stop"}`)
}

func buildAnthropicContent(mode MockMode, req *anthropicRequest) string {
	switch mode {
	case ModeEcho:
		return extractLastAnthropicUserMessage(req.Messages)
	case ModeMultiTurn:
		return fmt.Sprintf("Turn response for model %s. I received your message.", req.Model)
	case ModeReasoning:
		return "After careful reasoning, the answer is 42."
	case ModeSlow:
		return "This is a slow response for timeout testing."
	case ModeValidator:
		return buildValidatorResponse(req.Messages)
	case ModeGenerator:
		return buildGeneratorResponse(req.Messages)
	default:
		return "Mock response from " + req.Model
	}
}

func extractLastAnthropicUserMessage(messages json.RawMessage) string {
	var msgs []struct {
		Role    string          `json:"role"`
		Content json.RawMessage `json:"content"`
	}
	if err := json.Unmarshal(messages, &msgs); err != nil {
		log.Printf("failed to parse messages for echo: %v", err)
		return "echo: (failed to parse messages)"
	}
	for i := len(msgs) - 1; i >= 0; i-- {
		if msgs[i].Role == "user" {
			var text string
			if err := json.Unmarshal(msgs[i].Content, &text); err == nil {
				return "echo: " + text
			}
			var blocks []struct {
				Type string `json:"type"`
				Text string `json:"text"`
			}
			if err := json.Unmarshal(msgs[i].Content, &blocks); err == nil {
				for _, b := range blocks {
					if b.Type == "text" {
						return "echo: " + b.Text
					}
				}
			}
		}
	}
	return "echo: (no user message found)"
}
