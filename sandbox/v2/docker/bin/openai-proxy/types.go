package proxy

import "encoding/json"

// ============================================
// Anthropic API Types
// ============================================

type AnthropicRequest struct {
	Model         string               `json:"model"`
	Messages      []AnthropicMsg       `json:"messages"`
	System        interface{}          `json:"system,omitempty"`
	MaxTokens     int                  `json:"max_tokens"`
	Stream        bool                 `json:"stream,omitempty"`
	Temperature   *float64             `json:"temperature,omitempty"`
	TopP          *float64             `json:"top_p,omitempty"`
	TopK          *int                 `json:"top_k,omitempty"`
	StopSequences []string             `json:"stop_sequences,omitempty"`
	Tools         []AnthropicTool      `json:"tools,omitempty"`
	ToolChoice    *AnthropicToolChoice `json:"tool_choice,omitempty"`
	Metadata      map[string]string    `json:"metadata,omitempty"`
}

type AnthropicMsg struct {
	Role    string      `json:"role"`
	Content interface{} `json:"content"`
}

type ContentBlock struct {
	Type      string       `json:"type"`
	Text      string       `json:"text,omitempty"`
	Source    *ImageSource `json:"source,omitempty"`
	ID        string       `json:"id,omitempty"`
	Name      string       `json:"name,omitempty"`
	Input     interface{}  `json:"input,omitempty"`
	ToolUseID string       `json:"tool_use_id,omitempty"`
	Content   interface{}  `json:"content,omitempty"`
	IsError   bool         `json:"is_error,omitempty"`
}

type ImageSource struct {
	Type      string `json:"type"`
	MediaType string `json:"media_type,omitempty"`
	Data      string `json:"data,omitempty"`
	URL       string `json:"url,omitempty"`
}

type SystemBlock struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

type AnthropicTool struct {
	Name        string      `json:"name"`
	Description string      `json:"description,omitempty"`
	InputSchema interface{} `json:"input_schema"`
}

type AnthropicToolChoice struct {
	Type string `json:"type"`
	Name string `json:"name,omitempty"`
}

type AnthropicResponse struct {
	ID           string         `json:"id"`
	Type         string         `json:"type"`
	Role         string         `json:"role"`
	Content      []ContentBlock `json:"content"`
	Model        string         `json:"model"`
	StopReason   *string        `json:"stop_reason"`
	StopSequence *string        `json:"stop_sequence,omitempty"`
	Usage        *Usage         `json:"usage"`
}

type Usage struct {
	InputTokens  int `json:"input_tokens"`
	OutputTokens int `json:"output_tokens"`
}

type AnthropicStreamEvent struct {
	Type         string             `json:"type"`
	Index        int                `json:"index,omitempty"`
	Message      *AnthropicResponse `json:"message,omitempty"`
	ContentBlock *ContentBlock      `json:"content_block,omitempty"`
	Delta        *DeltaContent      `json:"delta,omitempty"`
	Usage        *Usage             `json:"usage,omitempty"`
}

type DeltaContent struct {
	Type        string  `json:"type,omitempty"`
	Text        string  `json:"text,omitempty"`
	PartialJSON string  `json:"partial_json,omitempty"`
	StopReason  *string `json:"stop_reason,omitempty"`
}

// ============================================
// OpenAI API Types
// ============================================

type OpenAIRequest struct {
	Model         string                 `json:"model"`
	Messages      []OpenAIMsg            `json:"messages"`
	MaxTokens     int                    `json:"max_tokens,omitempty"`
	Stream        bool                   `json:"stream,omitempty"`
	StreamOptions *StreamOptions         `json:"stream_options,omitempty"`
	Temperature   *float64               `json:"temperature,omitempty"`
	TopP          *float64               `json:"top_p,omitempty"`
	Stop          []string               `json:"stop,omitempty"`
	Tools         []OpenAITool           `json:"tools,omitempty"`
	ToolChoice    interface{}            `json:"tool_choice,omitempty"`
	ExtraOptions  map[string]interface{} `json:"-"`
}

func (r OpenAIRequest) MarshalJSON() ([]byte, error) {
	m := map[string]interface{}{
		"model":    r.Model,
		"messages": r.Messages,
	}
	if r.MaxTokens > 0 {
		m["max_tokens"] = r.MaxTokens
	}
	if r.Stream {
		m["stream"] = r.Stream
	}
	if r.StreamOptions != nil {
		m["stream_options"] = r.StreamOptions
	}
	if r.Temperature != nil {
		m["temperature"] = *r.Temperature
	}
	if r.TopP != nil {
		m["top_p"] = *r.TopP
	}
	if len(r.Stop) > 0 {
		m["stop"] = r.Stop
	}
	if len(r.Tools) > 0 {
		m["tools"] = r.Tools
	}
	if r.ToolChoice != nil {
		m["tool_choice"] = r.ToolChoice
	}
	for k, v := range r.ExtraOptions {
		if _, exists := m[k]; !exists {
			m[k] = v
		}
	}
	return json.Marshal(m)
}

type StreamOptions struct {
	IncludeUsage bool `json:"include_usage"`
}

type OpenAIMsg struct {
	Role       string           `json:"role"`
	Content    interface{}      `json:"content,omitempty"`
	ToolCalls  []OpenAIToolCall `json:"tool_calls,omitempty"`
	ToolCallID string           `json:"tool_call_id,omitempty"`
	Name       string           `json:"name,omitempty"`
}

type OpenAIContent struct {
	Type     string          `json:"type"`
	Text     string          `json:"text,omitempty"`
	ImageURL *OpenAIImageURL `json:"image_url,omitempty"`
}

type OpenAIImageURL struct {
	URL    string `json:"url"`
	Detail string `json:"detail,omitempty"`
}

type OpenAITool struct {
	Type     string         `json:"type"`
	Function OpenAIFunction `json:"function"`
}

type OpenAIFunction struct {
	Name        string      `json:"name"`
	Description string      `json:"description,omitempty"`
	Parameters  interface{} `json:"parameters"`
}

type OpenAIToolCall struct {
	ID       string             `json:"id"`
	Type     string             `json:"type"`
	Function OpenAIFunctionCall `json:"function"`
	Index    *int               `json:"index,omitempty"`
}

type OpenAIFunctionCall struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

type OpenAIResponse struct {
	ID      string         `json:"id"`
	Object  string         `json:"object"`
	Created int64          `json:"created"`
	Model   string         `json:"model"`
	Choices []OpenAIChoice `json:"choices"`
	Usage   *OpenAIUsage   `json:"usage,omitempty"`
}

type OpenAIChoice struct {
	Index        int       `json:"index"`
	Message      OpenAIMsg `json:"message"`
	FinishReason string    `json:"finish_reason"`
}

type OpenAIUsage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

type OpenAIStreamChunk struct {
	ID      string               `json:"id"`
	Object  string               `json:"object"`
	Created int64                `json:"created"`
	Model   string               `json:"model"`
	Choices []OpenAIStreamChoice `json:"choices"`
	Usage   *OpenAIUsage         `json:"usage,omitempty"`
}

type OpenAIStreamChoice struct {
	Index        int               `json:"index"`
	Delta        OpenAIStreamDelta `json:"delta"`
	FinishReason string            `json:"finish_reason,omitempty"`
}

type OpenAIStreamDelta struct {
	Role      string           `json:"role,omitempty"`
	Content   string           `json:"content,omitempty"`
	ToolCalls []OpenAIToolCall `json:"tool_calls,omitempty"`
}

type ToolCallAccumulator struct {
	Index int
	ID    string
	Name  string
	Args  string
}
