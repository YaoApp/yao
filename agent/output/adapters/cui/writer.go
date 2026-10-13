package cui

import (
	"encoding/json"
	"net/http"

	"github.com/yaoapp/yao/agent/output/message"
)

// Writer implements the message.Writer interface for CUI clients
type Writer struct {
	Writer  http.ResponseWriter
	Locale  string
	adapter *Adapter
}

// NewWriter creates a new CUI writer
// The Writer should already be wrapped in SafeWriter by the context layer
// to ensure thread-safe concurrent writes for SSE streaming.
func NewWriter(options message.Options) (*Writer, error) {
	return &Writer{
		Writer:  options.Writer,
		Locale:  options.Locale,
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
func (w *Writer) WriteGroup(group *message.Group) error {
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
	// SafeWriter cleanup is handled by the context layer
	return nil
}

// sendChunk sends a chunk to the output stream
func (w *Writer) sendChunk(chunk interface{}) error {
	// Convert chunk to JSON
	data, err := json.Marshal(chunk)
	if err != nil {
		return err
	}

	// Format as SSE (Server-Sent Events) format: "data: {json}\n\n"
	sseData := []byte("data: ")
	sseData = append(sseData, data...)
	sseData = append(sseData, '\n', '\n')

	// Send via context's writer
	// The context knows how to send data based on the connection type (SSE, WebSocket, etc.)
	if err := w.sendData(sseData); err != nil {
		return err
	}

	w.flush()
	return nil
}

func (w *Writer) flush() error {
	if w.Writer == nil {
		return nil // No writer, silently ignore
	}
	if flusher, ok := w.Writer.(interface{ Flush() }); ok {
		flusher.Flush()
	}
	return nil
}

func (w *Writer) sendData(data []byte) error {
	if w.Writer == nil {
		return nil // No writer, silently ignore
	}
	_, err := w.Writer.Write(data)
	return err
}
