// Package proxy provides a lightweight API proxy that translates
// Anthropic Messages API to OpenAI Chat Completions API.
package proxy

import (
	"bufio"
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

// Config holds the proxy server configuration
type Config struct {
	Port    int
	Backend string
	Model   string
	APIKey  string
	Timeout int
	Verbose bool
	LogFile string
	Options map[string]interface{}
}

// Server is the API proxy server
type Server struct {
	config *Config
	client *http.Client
}

// Main is the entry point for the proxy server
func Main() {
	config := parseFlags()
	if err := config.Validate(); err != nil {
		log.Fatalf("Configuration error: %v", err)
	}

	if config.LogFile != "" {
		f, err := os.OpenFile(config.LogFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			log.Fatalf("Failed to open log file: %v", err)
		}
		mw := io.MultiWriter(os.Stdout, f)
		log.SetOutput(mw)
	}

	server := NewServer(config)
	addr := fmt.Sprintf(":%d", config.Port)

	log.Printf("OpenAI Proxy starting on %s", addr)
	log.Printf("Backend: %s", config.Backend)
	log.Printf("Model: %s", config.Model)
	if len(config.Options) > 0 {
		optBytes, _ := json.Marshal(config.Options)
		log.Printf("Options: %s", string(optBytes))
	}

	http.HandleFunc("/v1/messages", server.handleMessages)
	http.HandleFunc("/health", server.handleHealth)

	if err := http.ListenAndServe(addr, nil); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}

func parseFlags() *Config {
	config := &Config{}

	flag.IntVar(&config.Port, "p", 0, "Listen port")
	flag.IntVar(&config.Port, "port", 0, "Listen port")
	flag.StringVar(&config.Backend, "b", "", "Backend API URL")
	flag.StringVar(&config.Backend, "backend", "", "Backend API URL")
	flag.StringVar(&config.Model, "m", "", "Backend model name")
	flag.StringVar(&config.Model, "model", "", "Backend model name")
	flag.StringVar(&config.APIKey, "k", "", "Backend API key")
	flag.StringVar(&config.APIKey, "api-key", "", "Backend API key")
	flag.IntVar(&config.Timeout, "t", 0, "Request timeout in seconds")
	flag.IntVar(&config.Timeout, "timeout", 0, "Request timeout in seconds")
	flag.BoolVar(&config.Verbose, "v", false, "Verbose logging")
	flag.BoolVar(&config.Verbose, "verbose", false, "Verbose logging")
	flag.StringVar(&config.LogFile, "l", "", "Log file path")
	flag.StringVar(&config.LogFile, "log", "", "Log file path")

	flag.Parse()

	if config.Port == 0 {
		if v := os.Getenv("OPENAI_PROXY_PORT"); v != "" {
			config.Port, _ = strconv.Atoi(v)
		}
	}
	if config.Port == 0 {
		config.Port = 3456
	}

	if config.Backend == "" {
		config.Backend = os.Getenv("OPENAI_PROXY_BACKEND")
	}

	if config.Model == "" {
		config.Model = os.Getenv("OPENAI_PROXY_MODEL")
	}

	if config.APIKey == "" {
		config.APIKey = os.Getenv("OPENAI_PROXY_API_KEY")
	}

	if config.Timeout == 0 {
		if v := os.Getenv("OPENAI_PROXY_TIMEOUT"); v != "" {
			config.Timeout, _ = strconv.Atoi(v)
		}
	}
	if config.Timeout == 0 {
		config.Timeout = 300
	}

	if optionsStr := os.Getenv("OPENAI_PROXY_OPTIONS"); optionsStr != "" {
		var options map[string]interface{}
		if err := json.Unmarshal([]byte(optionsStr), &options); err != nil {
			log.Printf("Warning: failed to parse OPENAI_PROXY_OPTIONS: %v", err)
		} else {
			config.Options = options
		}
	}

	return config
}

// Validate checks if the configuration is valid
func (c *Config) Validate() error {
	if c.Backend == "" {
		return fmt.Errorf("backend URL is required (-b or OPENAI_PROXY_BACKEND)")
	}
	if c.Model == "" {
		return fmt.Errorf("model name is required (-m or OPENAI_PROXY_MODEL)")
	}
	if c.APIKey == "" {
		return fmt.Errorf("API key is required (-k or OPENAI_PROXY_API_KEY)")
	}
	return nil
}

// NewServer creates a new proxy server
func NewServer(config *Config) *Server {
	return &Server{
		config: config,
		client: &http.Client{
			Timeout: time.Duration(config.Timeout) * time.Second,
		},
	}
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

func (s *Server) handleMessages(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		s.errorResponse(w, http.StatusBadRequest, "invalid_request", "Failed to read request body")
		return
	}
	defer r.Body.Close()

	if s.config.Verbose {
		log.Printf("Received request: %s", string(body))
	}

	var anthropicReq AnthropicRequest
	if err := json.Unmarshal(body, &anthropicReq); err != nil {
		s.errorResponse(w, http.StatusBadRequest, "invalid_request", "Invalid JSON")
		return
	}

	openaiReq := s.convertRequest(&anthropicReq)

	if anthropicReq.Stream {
		s.handleStreamingRequest(w, openaiReq)
	} else {
		s.handleNonStreamingRequest(w, openaiReq)
	}
}

func (s *Server) handleNonStreamingRequest(w http.ResponseWriter, openaiReq *OpenAIRequest) {
	openaiReq.Stream = false

	resp, err := s.forwardRequest(openaiReq)
	if err != nil {
		s.errorResponse(w, http.StatusBadGateway, "backend_error", err.Error())
		return
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		s.errorResponse(w, http.StatusBadGateway, "backend_error", "Failed to read backend response")
		return
	}

	if s.config.Verbose {
		log.Printf("Backend response: %s", string(body))
	}

	if resp.StatusCode != http.StatusOK {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(resp.StatusCode)
		w.Write(body)
		return
	}

	var openaiResp OpenAIResponse
	if err := json.Unmarshal(body, &openaiResp); err != nil {
		s.errorResponse(w, http.StatusBadGateway, "backend_error", "Invalid backend response")
		return
	}

	anthropicResp := s.convertResponse(&openaiResp)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(anthropicResp)
}

func (s *Server) handleStreamingRequest(w http.ResponseWriter, openaiReq *OpenAIRequest) {
	openaiReq.Stream = true
	openaiReq.StreamOptions = &StreamOptions{IncludeUsage: true}

	resp, err := s.forwardRequest(openaiReq)
	if err != nil {
		s.errorResponse(w, http.StatusBadGateway, "backend_error", err.Error())
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(resp.StatusCode)
		w.Write(body)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	flusher, ok := w.(http.Flusher)
	if !ok {
		s.errorResponse(w, http.StatusInternalServerError, "server_error", "Streaming not supported")
		return
	}

	msgID := generateID("msg_")
	startEvent := AnthropicStreamEvent{
		Type: "message_start",
		Message: &AnthropicResponse{
			ID:           msgID,
			Type:         "message",
			Role:         "assistant",
			Content:      []ContentBlock{},
			Model:        s.config.Model,
			StopReason:   nil,
			StopSequence: nil,
			Usage:        &Usage{InputTokens: 0, OutputTokens: 0},
		},
	}
	s.writeSSE(w, flusher, startEvent)

	s.processStream(w, flusher, resp.Body, msgID)
}

func (s *Server) processStream(w http.ResponseWriter, flusher http.Flusher, body io.Reader, msgID string) {
	scanner := bufio.NewScanner(body)
	scanner.Buffer(make([]byte, 64*1024), 1024*1024)

	var contentBlockStarted bool
	var currentToolCall *ToolCallAccumulator
	var toolCalls []*ToolCallAccumulator
	var contentIndex int
	var finishReason string
	var lastUsage *Usage

	for scanner.Scan() {
		line := scanner.Text()

		if !strings.HasPrefix(line, "data: ") {
			continue
		}

		data := strings.TrimPrefix(line, "data: ")
		if data == "[DONE]" {
			break
		}

		var chunk OpenAIStreamChunk
		if err := json.Unmarshal([]byte(data), &chunk); err != nil {
			if s.config.Verbose {
				log.Printf("Failed to parse chunk: %s", data)
			}
			continue
		}

		if len(chunk.Choices) == 0 {
			if chunk.Usage != nil {
				lastUsage = &Usage{
					InputTokens:  chunk.Usage.PromptTokens,
					OutputTokens: chunk.Usage.CompletionTokens,
				}
			}
			continue
		}

		choice := chunk.Choices[0]

		if choice.FinishReason != "" {
			finishReason = mapFinishReason(choice.FinishReason)
		}

		if len(choice.Delta.ToolCalls) > 0 {
			for _, tc := range choice.Delta.ToolCalls {
				if tc.Index != nil {
					idx := *tc.Index
					if idx >= len(toolCalls) {
						if contentBlockStarted && currentToolCall == nil {
							stopEvent := AnthropicStreamEvent{
								Type:  "content_block_stop",
								Index: contentIndex - 1,
							}
							s.writeSSE(w, flusher, stopEvent)
						}

						currentToolCall = &ToolCallAccumulator{
							Index: idx,
							ID:    tc.ID,
							Name:  tc.Function.Name,
							Args:  "",
						}
						toolCalls = append(toolCalls, currentToolCall)

						startEvent := AnthropicStreamEvent{
							Type:  "content_block_start",
							Index: contentIndex,
							ContentBlock: &ContentBlock{
								Type:  "tool_use",
								ID:    tc.ID,
								Name:  tc.Function.Name,
								Input: map[string]interface{}{},
							},
						}
						s.writeSSE(w, flusher, startEvent)
						contentIndex++
					}

					if tc.Function.Arguments != "" {
						currentToolCall.Args += tc.Function.Arguments
						deltaEvent := AnthropicStreamEvent{
							Type:  "content_block_delta",
							Index: contentIndex - 1,
							Delta: &DeltaContent{
								Type:        "input_json_delta",
								PartialJSON: tc.Function.Arguments,
							},
						}
						s.writeSSE(w, flusher, deltaEvent)
					}
				}
			}
			continue
		}

		if choice.Delta.Content != "" {
			if !contentBlockStarted {
				startEvent := AnthropicStreamEvent{
					Type:  "content_block_start",
					Index: contentIndex,
					ContentBlock: &ContentBlock{
						Type: "text",
						Text: "",
					},
				}
				s.writeSSE(w, flusher, startEvent)
				contentBlockStarted = true
				contentIndex++
			}

			deltaEvent := AnthropicStreamEvent{
				Type:  "content_block_delta",
				Index: contentIndex - 1,
				Delta: &DeltaContent{
					Type: "text_delta",
					Text: choice.Delta.Content,
				},
			}
			s.writeSSE(w, flusher, deltaEvent)
		}
	}

	if contentBlockStarted || len(toolCalls) > 0 {
		stopEvent := AnthropicStreamEvent{
			Type:  "content_block_stop",
			Index: contentIndex - 1,
		}
		s.writeSSE(w, flusher, stopEvent)
	}

	if finishReason == "" {
		finishReason = "end_turn"
	}
	if lastUsage == nil {
		lastUsage = &Usage{InputTokens: 0, OutputTokens: 0}
	}
	deltaEvent := AnthropicStreamEvent{
		Type: "message_delta",
		Delta: &DeltaContent{
			StopReason: &finishReason,
		},
		Usage: lastUsage,
	}
	s.writeSSE(w, flusher, deltaEvent)

	stopEvent := AnthropicStreamEvent{
		Type: "message_stop",
	}
	s.writeSSE(w, flusher, stopEvent)
}

func (s *Server) writeSSE(w http.ResponseWriter, flusher http.Flusher, event interface{}) {
	data, err := json.Marshal(event)
	if err != nil {
		return
	}

	eventType := ""
	if e, ok := event.(AnthropicStreamEvent); ok {
		eventType = e.Type
	}

	if eventType != "" {
		fmt.Fprintf(w, "event: %s\n", eventType)
	}
	fmt.Fprintf(w, "data: %s\n\n", data)
	flusher.Flush()

	if s.config.Verbose {
		log.Printf("SSE event: %s", string(data))
	}
}

func (s *Server) forwardRequest(openaiReq *OpenAIRequest) (*http.Response, error) {
	body, err := json.Marshal(openaiReq)
	if err != nil {
		return nil, err
	}

	if s.config.Verbose {
		log.Printf("Forwarding to backend: %s", string(body))
	}

	req, err := http.NewRequest(http.MethodPost, s.config.Backend, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+s.config.APIKey)

	return s.client.Do(req)
}

func (s *Server) errorResponse(w http.ResponseWriter, status int, errType, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"type": "error",
		"error": map[string]string{
			"type":    errType,
			"message": message,
		},
	})
}

func generateID(prefix string) string {
	return fmt.Sprintf("%s%d", prefix, time.Now().UnixNano())
}

func mapFinishReason(reason string) string {
	switch reason {
	case "stop":
		return "end_turn"
	case "length":
		return "max_tokens"
	case "tool_calls", "function_call":
		return "tool_use"
	case "content_filter":
		return "end_turn"
	default:
		return "end_turn"
	}
}
