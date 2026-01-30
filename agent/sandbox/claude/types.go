package claude

// StreamMessage represents a parsed stream message from Claude CLI
type StreamMessage struct {
	Type    string      `json:"type"`
	Subtype string      `json:"subtype,omitempty"`
	Content interface{} `json:"content,omitempty"`
	Error   string      `json:"error,omitempty"`
}

// ToolCall represents a tool invocation from the agent
type ToolCall struct {
	ID        string                 `json:"id"`
	Name      string                 `json:"name"`
	Arguments map[string]interface{} `json:"arguments"`
}

// ToolResult represents a tool execution result
type ToolResult struct {
	ID      string `json:"id"`
	Content string `json:"content"`
	IsError bool   `json:"is_error,omitempty"`
}

// CLIResponse represents the parsed response from Claude CLI
type CLIResponse struct {
	Text      string     `json:"text,omitempty"`
	ToolCalls []ToolCall `json:"tool_calls,omitempty"`
	Usage     *Usage     `json:"usage,omitempty"`
	Model     string     `json:"model,omitempty"`
}

// Usage represents token usage statistics
type Usage struct {
	InputTokens  int `json:"input_tokens,omitempty"`
	OutputTokens int `json:"output_tokens,omitempty"`
}
