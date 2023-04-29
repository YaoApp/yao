package conversation

// Xun Database conversation
type Xun struct{}

// NewXun create a new conversation
func NewXun() *Xun {
	return &Xun{}
}

// GetHistory get the history
func (conv *Xun) GetHistory(sid string) ([]map[string]interface{}, error) {
	return []map[string]interface{}{}, nil
}

// SaveHistory save the history
func (conv *Xun) SaveHistory(sid string, messages []map[string]interface{}) error {
	return nil
}
