package conversation

// Weaviate Database conversation
type Weaviate struct{}

// NewWeaviate create a new conversation
func NewWeaviate() *Weaviate {
	return &Weaviate{}
}

// GetHistory get the history
func (conv *Weaviate) GetHistory(sid string) ([]map[string]interface{}, error) {
	return []map[string]interface{}{}, nil
}

// SaveHistory save the history
func (conv *Weaviate) SaveHistory(sid string, messages []map[string]interface{}) error {
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
