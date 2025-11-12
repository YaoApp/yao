package context

import "net/http"

// StreamFunc the streaming function
type StreamFunc func(data []byte) int

// Writer is an alias for http.ResponseWriter interface used by an agent to construct a response.
// A Writer may not be used after the agent execution has completed.
type Writer = http.ResponseWriter

// Agent the agent interface
type Agent interface {

	// Stream stream the agent
	Stream(ctx *Context, messages []Message, handler StreamFunc) error

	// Run run the agent
	Run(ctx *Context, messages []Message) (*Response, error)
}
