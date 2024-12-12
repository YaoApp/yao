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
func (conv *Weaviate) GetChats(sid string) ([]map[string]interface{}, error) {
	return []map[string]interface{}{}, nil
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
