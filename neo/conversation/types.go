package conversation

// Setting the conversation config
type Setting struct {
	Connector string `json:"connector,omitempty"`
	UserField string `json:"user_field,omitempty"` // the user id field name, default is user_id
	Table     string `json:"table,omitempty"`
	MaxSize   int    `json:"max_size,omitempty" yaml:"max_size,omitempty"`
	TTL       int    `json:"ttl,omitempty" yaml:"ttl,omitempty"`
}

// ChatInfo represents the chat information and its history
type ChatInfo struct {
	Chat    map[string]interface{}   `json:"chat"`
	History []map[string]interface{} `json:"history"`
}

// ChatFilter represents the filter parameters for GetChats
type ChatFilter struct {
	Keywords string `json:"keywords,omitempty"`
	Page     int    `json:"page,omitempty"`     // 页码，从1开始
	PageSize int    `json:"pagesize,omitempty"` // 每页数量
	Order    string `json:"order,omitempty"`    // desc/asc
}

// ChatGroup represents a group of chats by date
type ChatGroup struct {
	Label string                   `json:"label"`
	Chats []map[string]interface{} `json:"chats"`
}

// ChatGroupResponse represents paginated chat groups
type ChatGroupResponse struct {
	Groups   []ChatGroup `json:"groups"`
	Page     int         `json:"page"`      // 当前页码
	PageSize int         `json:"pagesize"`  // 每页数量
	Total    int64       `json:"total"`     // 总记录数
	LastPage int         `json:"last_page"` // 最后一页页码
}

// Conversation the store interface
type Conversation interface {
	GetChats(sid string, filter ChatFilter) (*ChatGroupResponse, error)
	GetChat(sid string, cid string) (*ChatInfo, error)
	GetHistory(sid string, cid string) ([]map[string]interface{}, error)
	SaveHistory(sid string, messages []map[string]interface{}, cid string) error
	GetRequest(sid string, rid string) ([]map[string]interface{}, error)
	SaveRequest(sid string, rid string, cid string, messages []map[string]interface{}) error
	DeleteChat(sid string, cid string) error
	DeleteAllChats(sid string) error
	UpdateChatTitle(sid string, cid string, title string) error
}
