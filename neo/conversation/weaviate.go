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
