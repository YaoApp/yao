package openai

import "github.com/yaoapp/yao/agent/context"

// StreamChunk represents a chunk from OpenAI's streaming response
type StreamChunk struct {
	ID      string  `json:"id"`
	Object  string  `json:"object"`
	Created int64   `json:"created"`
	Model   string  `json:"model"`
	Choices []Delta `json:"choices"`
	Usage   *struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
		TotalTokens      int `json:"total_tokens"`
	} `json:"usage,omitempty"`
}

// Delta represents the delta in a streaming chunk
type Delta struct {
	Index        int          `json:"index"`
	Delta        DeltaContent `json:"delta"`
	FinishReason *string      `json:"finish_reason"`
}

// DeltaContent represents the content in a delta
type DeltaContent struct {
	Role      string          `json:"role,omitempty"`
	Content   string          `json:"content,omitempty"`
	ToolCalls []ToolCallDelta `json:"tool_calls,omitempty"`
	Refusal   string          `json:"refusal,omitempty"`
}

// ToolCallDelta represents a tool call delta in streaming
type ToolCallDelta struct {
	Index    int               `json:"index"`
	ID       string            `json:"id,omitempty"`
	Type     string            `json:"type,omitempty"`
	Function FunctionCallDelta `json:"function,omitempty"`
}

// FunctionCallDelta represents a function call delta
type FunctionCallDelta struct {
	Name      string `json:"name,omitempty"`
	Arguments string `json:"arguments,omitempty"`
}

// CompletionResponseFull represents the full non-streaming response
type CompletionResponseFull struct {
	ID      string `json:"id"`
	Object  string `json:"object"`
	Created int64  `json:"created"`
	Model   string `json:"model"`
	Choices []struct {
		Index        int             `json:"index"`
		Message      context.Message `json:"message"`
		FinishReason string          `json:"finish_reason"`
	} `json:"choices"`
	Usage             *context.UsageInfo `json:"usage,omitempty"`
	SystemFingerprint string             `json:"system_fingerprint,omitempty"`
}

// streamAccumulator accumulates streaming response data
type streamAccumulator struct {
	id           string
	model        string
	created      int64
	role         string
	content      string
	refusal      string
	toolCalls    map[int]*accumulatedToolCall
	finishReason string
	usage        *context.UsageInfo
}

// accumulatedToolCall accumulates a single tool call
type accumulatedToolCall struct {
	id           string
	typ          string
	functionName string
	functionArgs string
}
