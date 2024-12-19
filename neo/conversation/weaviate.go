package conversation

// Weaviate Database conversation
type Weaviate struct{}

// NewWeaviate create a new conversation
func NewWeaviate() *Weaviate {
	return &Weaviate{}
}

// UpdateChatTitle update the chat title
func (conv *Weaviate) UpdateChatTitle(sid string, cid string, title string) error {
	return nil
}

// GetChats get the chat list
func (conv *Weaviate) GetChats(sid string, filter ChatFilter) (*ChatGroupResponse, error) {
	return &ChatGroupResponse{
		Groups:   []ChatGroup{},
		Page:     filter.Page,
		PageSize: filter.PageSize,
		Total:    0,
		LastPage: 1,
	}, nil
}

// GetHistory get the history
func (conv *Weaviate) GetHistory(sid string, cid string) ([]map[string]interface{}, error) {
	return []map[string]interface{}{}, nil
}

// SaveHistory save the history
func (conv *Weaviate) SaveHistory(sid string, messages []map[string]interface{}, cid string) error {
	return nil
}

// GetRequest get the request
func (conv *Weaviate) GetRequest(sid string, rid string) ([]map[string]interface{}, error) {
	return nil, nil
}

// SaveRequest save the request
func (conv *Weaviate) SaveRequest(sid string, rid string, cid string, messages []map[string]interface{}) error {
	return nil
}

// GetChat get the chat info and its history
func (conv *Weaviate) GetChat(sid string, cid string) (*ChatInfo, error) {
	return nil, nil
}

// DeleteChat deletes a specific chat and its history
func (conv *Weaviate) DeleteChat(sid string, cid string) error {
	return nil
}

// DeleteAllChats deletes all chats and their histories for a user
func (conv *Weaviate) DeleteAllChats(sid string) error {
	return nil
}
