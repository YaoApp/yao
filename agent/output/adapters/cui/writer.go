package cui

import (
	"encoding/json"

	"github.com/yaoapp/yao/agent/context"
	"github.com/yaoapp/yao/agent/output/message"
)

// Writer implements the message.Writer interface for CUI clients
type Writer struct {
	ctx     *context.Context
	adapter *Adapter
}

// NewWriter creates a new CUI writer
func NewWriter(ctx *context.Context) (*Writer, error) {
	return &Writer{
		ctx:     ctx,
		adapter: NewAdapter(),
	}, nil
}

// Write writes a single message to the output stream
func (w *Writer) Write(msg *message.Message) error {
	// CUI adapter passes messages through as-is
	chunks, err := w.adapter.Adapt(msg)
	if err != nil {
		return err
	}

	// Send each chunk
	for _, chunk := range chunks {
		if err := w.sendChunk(chunk); err != nil {
			return err
		}
	}

	return nil
}

// WriteGroup writes a message group to the output stream
func (w *Writer) WriteGroup(group *message.MessageGroup) error {
	// For CUI, we send a group start message, all messages, then a group end message
	// The group structure itself is also sent for clients that want it

	// Send the group
	if err := w.sendChunk(group); err != nil {
		return err
	}

	return nil
}

// Flush flushes any buffered data to the output stream
func (w *Writer) Flush() error {
	// For SSE, we don't need explicit flushing
	// The underlying connection handles it
	return nil
}

// Close closes the writer and cleans up resources
func (w *Writer) Close() error {
	// Nothing to clean up for CUI writer
	return nil
}

// sendChunk sends a chunk to the output stream
func (w *Writer) sendChunk(chunk interface{}) error {
	// Convert chunk to JSON
	data, err := json.Marshal(chunk)
	if err != nil {
		return err
	}

	// Send via context's writer
	// The context knows how to send data based on the connection type (SSE, WebSocket, etc.)
	return w.ctx.Send(data)
}
