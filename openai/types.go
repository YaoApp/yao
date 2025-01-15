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
	Code    string      `json:"code,omitempty"`
}
