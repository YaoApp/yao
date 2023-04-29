package conversation

// Mongo conversation
type Mongo struct{}

// NewMongo create a new conversation
func NewMongo() *Mongo {
	return &Mongo{}
}

// GetHistory get the history
func (conv *Mongo) GetHistory(sid string) ([]map[string]interface{}, error) {
	return []map[string]interface{}{}, nil
}

// SaveHistory save the history
func (conv *Mongo) SaveHistory(sid string, messages []map[string]interface{}) error {
	return nil
}
