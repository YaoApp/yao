package openai

// Message is the response from OpenAI
// {"id":"chatcmpl-7Atx502nGBuYcvoZfIaWU4FREI1mT","object":"chat.completion.chunk","created":1682832715,"model":"gpt-3.5-turbo-0301","choices":[{"delta":{"content":"Hello"},"index":0,"finish_reason":null}]}
type Message struct {
	ID      string `json:"id,omitempty"`
	Object  string `json:"object,omitempty"`
	Created int64  `json:"created,omitempty"`
	Model   string `json:"model,omitempty"`
	Choices []struct {
		Delta struct {
			Content string `json:"content,omitempty"`
		} `json:"delta,omitempty"`
		Index        int    `json:"index,omitempty"`
		FinishReason string `json:"finish_reason,omitempty"`
	} `json:"choices,omitempty"`
}

// MessageWithReasoningContent is the response from OpenAI
type MessageWithReasoningContent struct {
	ID      string `json:"id,omitempty"`
	Object  string `json:"object,omitempty"`
	Created int64  `json:"created,omitempty"`
	Model   string `json:"model,omitempty"`
	Choices []struct {
		Delta        map[string]interface{} `json:"delta,omitempty"`
		Index        int                    `json:"index,omitempty"`
		FinishReason string                 `json:"finish_reason,omitempty"`
	} `json:"choices,omitempty"`
}

// ChatCompletionChunk is the response from OpenAI
type ChatCompletionChunk struct {
	ID                string                      `json:"id"`
	Object            string                      `json:"object"`
	Created           int64                       `json:"created"`
	Model             string                      `json:"model"`
	SystemFingerprint string                      `json:"system_fingerprint,omitempty"`
	Choices           []ChatCompletionChunkChoice `json:"choices"`
}

// ChatCompletionChunkChoice represents a chunk choice in the response
type ChatCompletionChunkChoice struct {
	Index        int                      `json:"index"`
	Delta        ChatCompletionChunkDelta `json:"delta"`
	LogProbs     *LogProbs                `json:"logprobs,omitempty"`
	FinishReason string                   `json:"finish_reason,omitempty"`
}

// ChatCompletionChunkDelta represents the delta content in a chunk
type ChatCompletionChunkDelta struct {
	Role             string        `json:"role,omitempty"`
	Content          string        `json:"content,omitempty"`
	ReasoningContent string        `json:"reasoning_content,omitempty"`
	ToolCalls        []ToolCall    `json:"tool_calls,omitempty"`
	FunctionCall     *FunctionCall `json:"function_call,omitempty"`
}

// LogProbs represents the log probabilities in a response
type LogProbs struct {
	Content []ContentLogProb `json:"content,omitempty"`
}

// ContentLogProb represents a single token's log probability information
type ContentLogProb struct {
	Token       string    `json:"token"`
	LogProb     float64   `json:"logprob"`
	Bytes       []int     `json:"bytes,omitempty"`
	TopLogProbs []LogProb `json:"top_logprobs,omitempty"`
}

// LogProb represents a token and its log probability
type LogProb struct {
	Token   string  `json:"token"`
	LogProb float64 `json:"logprob"`
	Bytes   []int   `json:"bytes,omitempty"`
}

// ToolCall represents a tool call in the response
type ToolCall struct {
	Index    int      `json:"index"`
	ID       string   `json:"id"`
	Type     string   `json:"type"`
	Function Function `json:"function"`
}

// FunctionCall represents a function call in the response
type FunctionCall struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

// Function represents a function in a tool call
type Function struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

// ToolCalls is the response from OpenAI
type ToolCalls struct {
	ID      string `json:"id,omitempty"`
	Object  string `json:"object,omitempty"`
	Created int64  `json:"created,omitempty"`
	Model   string `json:"model,omitempty"`
	Choices []struct {
		Delta struct {
			ToolCalls []struct {
				ID       string `json:"id,omitempty"`
				Type     string `json:"type,omitempty"`
				Function struct {
					Name      string `json:"name,omitempty"`
					Arguments string `json:"arguments,omitempty"`
				} `json:"function,omitempty"`
			} `json:"tool_calls,omitempty"`
		} `json:"delta,omitempty"`
		Index        int    `json:"index,omitempty"`
		FinishReason string `json:"finish_reason,omitempty"`
	} `json:"choices,omitempty"`
}

// ErrorMessage is the error response from OpenAI
type ErrorMessage struct {
	Error Error `json:"error,omitempty"`
}

// Error is the error response from OpenAI
type Error struct {
	Message string      `json:"message,omitempty"`
	Type    string      `json:"type,omitempty"`
	Param   interface{} `json:"param,omitempty"`
	Code    any         `json:"code,omitempty"` // string or int
}
