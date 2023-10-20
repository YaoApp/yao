package message

// Message the message
type Message struct {
	Text    string                 `json:"text,omitempty"`
	Error   string                 `json:"error,omitempty"`
	Done    bool                   `json:"done,omitempty"`
	Confirm bool                   `json:"confirm,omitempty"`
	Command *Command               `json:"command,omitempty"`
	Actions []Action               `json:"actions,omitempty"`
	Data    map[string]interface{} `json:"-,omitempty"`
}

// Action the action
type Action struct {
	Name    string      `json:"name,omitempty"`
	Type    string      `json:"type"`
	Payload interface{} `json:"payload,omitempty"`
	Next    string      `json:"next,omitempty"`
}

// Command the command
type Command struct {
	ID      string `json:"id,omitempty"`
	Name    string `json:"name,omitempty"`
	Reqeust string `json:"request,omitempty"`
}
