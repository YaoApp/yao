package brain

// NewBehaviors Returns the behaviors based on input words
func NewBehaviors(words string) (*Behaviors, error) {
	resp, err := NPL(words)
	if err != nil {
		return nil, err
	}
	return resp.Behaviors, nil
}

// Run the behaviors
func (behaviors *Behaviors) Run() {}

// NPL the Natural language processing
func NPL(words string) (*Response, error) {
	return &Response{Behaviors: &Behaviors{}}, nil
}
