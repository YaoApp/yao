package conversation

// Setting the conversation config
type Setting struct {
	Connector string `json:"connector,omitempty"`
	Table     string `json:"table,omitempty"`
	MaxSize   int    `json:"max_size,omitempty" yaml:"max_size,omitempty"`
	TTL       int    `json:"ttl,omitempty" yaml:"ttl,omitempty"`
}

// ChatInfo represents the chat information and its history
type ChatInfo struct {
	Chat    map[string]interface{}   `json:"chat"`
	History []map[string]interface{} `json:"history"`
}

// Conversation the store interface
type Conversation interface {
	UpdateChatTitle(sid string, cid string, title string) error
	GetChats(sid string, keywords ...string) ([]map[string]interface{}, error)
	GetChat(sid string, cid string) (*ChatInfo, error)
	GetHistory(sid string, cid string) ([]map[string]interface{}, error)
	SaveHistory(sid string, messages []map[string]interface{}, cid string) error
	GetRequest(sid string, rid string) ([]map[string]interface{}, error)
	SaveRequest(sid string, rid string, cid string, messages []map[string]interface{}) error
}
