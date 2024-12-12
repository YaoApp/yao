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
func (conv *Redis) GetChats(sid string) ([]map[string]interface{}, error) {
	return []map[string]interface{}{}, nil
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
