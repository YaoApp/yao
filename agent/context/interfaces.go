package context

// StreamFunc the streaming function
type StreamFunc func(data []byte) int

// Agent the agent interface
type Agent interface {

	// Stream stream the agent
	Stream(ctx Context, messages []Message, handler StreamFunc) error

	// Run run the agent
	Run(ctx Context, messages []Message) (*Response, error)
}
