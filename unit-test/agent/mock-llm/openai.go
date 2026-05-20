package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

func handleOpenAI(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	mode := parseMockMode(r)

	switch mode {
	case ModeError429:
		w.Header().Set("Retry-After", "1")
		http.Error(w, `{"error":{"message":"Rate limit exceeded","type":"rate_limit_error","code":"rate_limit_exceeded"}}`, http.StatusTooManyRequests)
		return
	case ModeError500:
		http.Error(w, `{"error":{"message":"Internal server error","type":"server_error"}}`, http.StatusInternalServerError)
		return
	case ModeFixture:
		key := fixtureKey(r)
		if key == "" {
			key = "openai/default"
		}
		if data, ok := GetFixture(key); ok {
			w.Header().Set("Content-Type", "application/json")
			w.Write(data)
			return
		}
		http.Error(w, fmt.Sprintf(`{"error":{"message":"fixture not found: %s"}}`, key), http.StatusNotFound)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, `{"error":{"message":"failed to read body"}}`, http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	fmt.Printf("[TRACE-SANDBOX] === OpenAI request from %s ===\n", r.RemoteAddr)
	fmt.Printf("[TRACE-SANDBOX] URL: %s\n", r.URL.Path)
	fmt.Printf("[TRACE-SANDBOX] Body (%d bytes): %s\n", len(body), string(body))

	var req openAIRequest
	if err := json.Unmarshal(body, &req); err != nil {
		fmt.Printf("[TRACE-SANDBOX] JSON parse error: %v\n", err)
		http.Error(w, `{"error":{"message":"invalid JSON"}}`, http.StatusBadRequest)
		return
	}
	fmt.Printf("[TRACE-SANDBOX] Model: %s, Stream: %v\n", req.Model, req.Stream)

	if r.Header.Get(MockModeHeader) == "" {
		mode = detectModeFromModel(req.Model)
	}

	if req.Stream {
		openAIStreamResponse(w, r, mode, &req)
	} else {
		openAINonStreamResponse(w, mode, &req)
	}
}

type openAIRequest struct {
	Model    string          `json:"model"`
	Messages json.RawMessage `json:"messages"`
	Stream   bool            `json:"stream"`
	Tools    json.RawMessage `json:"tools,omitempty"`
}

type openAIChoice struct {
	Index        int            `json:"index"`
	Delta        *openAIDelta   `json:"delta,omitempty"`
	Message      *openAIMessage `json:"message,omitempty"`
	FinishReason *string        `json:"finish_reason"`
}

type openAIDelta struct {
	Role             string          `json:"role,omitempty"`
	Content          string          `json:"content,omitempty"`
	ReasoningContent string          `json:"reasoning_content,omitempty"`
	ToolCalls        json.RawMessage `json:"tool_calls,omitempty"`
}

type openAIMessage struct {
	Role             string          `json:"role"`
	Content          string          `json:"content"`
	ReasoningContent string          `json:"reasoning_content,omitempty"`
	ToolCalls        json.RawMessage `json:"tool_calls,omitempty"`
}

type openAIStreamChunk struct {
	ID      string         `json:"id"`
	Object  string         `json:"object"`
	Created int64          `json:"created"`
	Model   string         `json:"model"`
	Choices []openAIChoice `json:"choices"`
}

type openAIResponse struct {
	ID      string         `json:"id"`
	Object  string         `json:"object"`
	Created int64          `json:"created"`
	Model   string         `json:"model"`
	Choices []openAIChoice `json:"choices"`
	Usage   openAIUsage    `json:"usage"`
}

type openAIUsage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

func openAINonStreamResponse(w http.ResponseWriter, mode MockMode, req *openAIRequest) {
	if mode == ModeSlow {
		time.Sleep(3 * time.Second)
	}
	content := buildOpenAIContent(mode, req)
	finish := "stop"

	msg := &openAIMessage{Role: "assistant", Content: content}
	if mode == ModeReasoning {
		msg.ReasoningContent = "Let me think about this step by step..."
		msg.Content = content
	}
	if mode == ModeToolCall {
		finish = "tool_calls"
		msg.Content = ""
		msg.ToolCalls = json.RawMessage(`[{"id":"call_mock_001","type":"function","function":{"name":"get_weather","arguments":"{\"location\":\"San Francisco\"}"}}]`)
	}

	resp := openAIResponse{
		ID:      "chatcmpl-mock-001",
		Object:  "chat.completion",
		Created: time.Now().Unix(),
		Model:   req.Model,
		Choices: []openAIChoice{{Index: 0, Message: msg, FinishReason: &finish}},
		Usage:   openAIUsage{PromptTokens: 10, CompletionTokens: 20, TotalTokens: 30},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func openAIStreamResponse(w http.ResponseWriter, r *http.Request, mode MockMode, req *openAIRequest) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming not supported", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	id := "chatcmpl-mock-stream-001"
	model := req.Model
	now := time.Now().Unix()

	sendChunk := func(chunk openAIStreamChunk) {
		data, _ := json.Marshal(chunk)
		fmt.Fprintf(w, "data: %s\n\n", data)
		flusher.Flush()
	}

	// Role chunk
	sendChunk(openAIStreamChunk{
		ID: id, Object: "chat.completion.chunk", Created: now, Model: model,
		Choices: []openAIChoice{{Index: 0, Delta: &openAIDelta{Role: "assistant"}}},
	})

	if mode == ModeReasoning {
		for _, part := range []string{"Let me ", "think ", "step by step..."} {
			sendChunk(openAIStreamChunk{
				ID: id, Object: "chat.completion.chunk", Created: now, Model: model,
				Choices: []openAIChoice{{Index: 0, Delta: &openAIDelta{ReasoningContent: part}}},
			})
			time.Sleep(10 * time.Millisecond)
		}
	}

	if mode == ModeToolCall {
		tc := json.RawMessage(`[{"index":0,"id":"call_mock_001","type":"function","function":{"name":"get_weather","arguments":""}}]`)
		sendChunk(openAIStreamChunk{
			ID: id, Object: "chat.completion.chunk", Created: now, Model: model,
			Choices: []openAIChoice{{Index: 0, Delta: &openAIDelta{ToolCalls: tc}}},
		})
		tc2 := json.RawMessage(`[{"index":0,"function":{"arguments":"{\"location\":\"San Francisco\"}"}}]`)
		sendChunk(openAIStreamChunk{
			ID: id, Object: "chat.completion.chunk", Created: now, Model: model,
			Choices: []openAIChoice{{Index: 0, Delta: &openAIDelta{ToolCalls: tc2}}},
		})
		finish := "tool_calls"
		sendChunk(openAIStreamChunk{
			ID: id, Object: "chat.completion.chunk", Created: now, Model: model,
			Choices: []openAIChoice{{Index: 0, Delta: &openAIDelta{}, FinishReason: &finish}},
		})
	} else {
		content := buildOpenAIContent(mode, req)
		if mode == ModeSlow {
			time.Sleep(3 * time.Second)
		}
		words := strings.Fields(content)
		for _, word := range words {
			sendChunk(openAIStreamChunk{
				ID: id, Object: "chat.completion.chunk", Created: now, Model: model,
				Choices: []openAIChoice{{Index: 0, Delta: &openAIDelta{Content: word + " "}}},
			})
			time.Sleep(10 * time.Millisecond)
		}
		finish := "stop"
		sendChunk(openAIStreamChunk{
			ID: id, Object: "chat.completion.chunk", Created: now, Model: model,
			Choices: []openAIChoice{{Index: 0, Delta: &openAIDelta{}, FinishReason: &finish}},
		})
	}

	fmt.Fprint(w, "data: [DONE]\n\n")
	flusher.Flush()
}

func buildOpenAIContent(mode MockMode, req *openAIRequest) string {
	switch mode {
	case ModeEcho:
		return extractLastUserMessage(req.Messages)
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

func extractLastUserMessage(messages json.RawMessage) string {
	fmt.Printf("[TRACE-SANDBOX] --- extractLastUserMessage ---\n")
	fmt.Printf("[TRACE-SANDBOX] raw messages: %s\n", string(messages))

	var msgs []struct {
		Role    string          `json:"role"`
		Content json.RawMessage `json:"content"`
	}
	if err := json.Unmarshal(messages, &msgs); err != nil {
		fmt.Printf("[TRACE-SANDBOX] PARSE ERROR: %v\n", err)
		return "echo: (failed to parse messages)"
	}
	fmt.Printf("[TRACE-SANDBOX] parsed %d messages\n", len(msgs))
	for i, m := range msgs {
		fmt.Printf("[TRACE-SANDBOX] msg[%d] role=%s content=%s\n", i, m.Role, string(m.Content))
	}

	for i := len(msgs) - 1; i >= 0; i-- {
		if msgs[i].Role == "user" {
			fmt.Printf("[TRACE-SANDBOX] found last user msg at index %d\n", i)

			var text string
			if err := json.Unmarshal(msgs[i].Content, &text); err == nil {
				fmt.Printf("[TRACE-SANDBOX] content is string: %s\n", text)
				return "echo: " + text
			}
			fmt.Printf("[TRACE-SANDBOX] content is NOT string, trying array...\n")

			var blocks []struct {
				Type string `json:"type"`
				Text string `json:"text"`
			}
			if err := json.Unmarshal(msgs[i].Content, &blocks); err == nil {
				fmt.Printf("[TRACE-SANDBOX] content is array with %d blocks\n", len(blocks))
				last := ""
				for j, b := range blocks {
					fmt.Printf("[TRACE-SANDBOX] block[%d] type=%s text_len=%d text_preview=%.200s\n", j, b.Type, len(b.Text), b.Text)
					if b.Type == "text" {
						last = b.Text
					}
				}
				if last != "" {
					fmt.Printf("[TRACE-SANDBOX] returning last text block (len=%d)\n", len(last))
					return "echo: " + last
				}
			} else {
				fmt.Printf("[TRACE-SANDBOX] content is NOT array either: %v\n", err)
			}
		}
	}
	fmt.Printf("[TRACE-SANDBOX] no user message found!\n")
	return "echo: (no user message found)"
}
