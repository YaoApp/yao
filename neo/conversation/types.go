package conversation

// Setting the conversation config
type Setting struct {
	Connector string `json:"connector,omitempty"`
	Table     string `json:"table,omitempty"`
	MaxSize   int    `json:"max_size,omitempty" yaml:"max_size,omitempty"`
	TTL       int    `json:"ttl,omitempty" yaml:"ttl,omitempty"`
}

// Conversation the store interface
type Conversation interface {
	UpdateChatTitle(sid string, cid string, title string) error
	GetChats(sid string) ([]map[string]interface{}, error)
	GetHistory(sid string, cid string) ([]map[string]interface{}, error)
	SaveHistory(sid string, messages []map[string]interface{}, cid string) error
	GetRequest(sid string, rid string) ([]map[string]interface{}, error)
	SaveRequest(sid string, rid string, cid string, messages []map[string]interface{}) error
}
