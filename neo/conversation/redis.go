package conversation

// Redis  conversation
type Redis struct{}

// NewRedis create a new conversation
func NewRedis() *Redis {
	return &Redis{}
}

// UpdateChatTitle update the chat title
func (conv *Redis) UpdateChatTitle(sid string, cid string, title string) error {
	return nil
}

// GetChats get the chat list
func (conv *Redis) GetChats(sid string, filter ChatFilter) (*ChatGroupResponse, error) {
	return &ChatGroupResponse{
		Groups:   []ChatGroup{},
		Page:     filter.Page,
		PageSize: filter.PageSize,
		Total:    0,
		LastPage: 1,
	}, nil
}

// GetHistory get the history
func (conv *Redis) GetHistory(sid string, cid string) ([]map[string]interface{}, error) {
	return []map[string]interface{}{}, nil
}

// SaveHistory save the history
func (conv *Redis) SaveHistory(sid string, messages []map[string]interface{}, cid string) error {
	return nil
}

// GetRequest get the request
func (conv *Redis) GetRequest(sid string, rid string) ([]map[string]interface{}, error) {
	return nil, nil
}

// SaveRequest save the request
func (conv *Redis) SaveRequest(sid string, rid string, cid string, messages []map[string]interface{}) error {
	return nil
}

// GetChat get the chat info and its history
func (conv *Redis) GetChat(sid string, cid string) (*ChatInfo, error) {
	return nil, nil
}

// DeleteChat deletes a specific chat and its history
func (conv *Redis) DeleteChat(sid string, cid string) error {
	return nil
}

// DeleteAllChats deletes all chats and their histories for a user
func (conv *Redis) DeleteAllChats(sid string) error {
	return nil
}
