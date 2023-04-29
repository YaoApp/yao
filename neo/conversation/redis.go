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
