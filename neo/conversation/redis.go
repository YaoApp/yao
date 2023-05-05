package conversation

// Redis  conversation
type Redis struct{}

// NewRedis create a new conversation
func NewRedis() *Redis {
	return &Redis{}
}

// GetHistory get the history
func (conv *Redis) GetHistory(sid string) ([]map[string]interface{}, error) {
	return []map[string]interface{}{}, nil
}

// SaveHistory save the history
func (conv *Redis) SaveHistory(sid string, messages []map[string]interface{}) error {
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
