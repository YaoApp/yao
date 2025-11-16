package openai

import (
	"encoding/json"
	"fmt"

	"github.com/yaoapp/yao/agent/context"
	"github.com/yaoapp/yao/agent/output/message"
)

// Writer implements the message.Writer interface for OpenAI-compatible clients
type Writer struct {
	ctx        *context.Context
	adapter    *Adapter
	firstChunk bool // Track if this is the first chunk to add role
}

// NewWriter creates a new OpenAI writer
func NewWriter(ctx *context.Context) (*Writer, error) {
	// Create adapter with default config
	adapter := NewAdapter()

	return &Writer{
		ctx:        ctx,
		adapter:    adapter,
		firstChunk: true, // First chunk should include role
	}, nil
}

// Write writes a single message to the output stream
func (w *Writer) Write(msg *message.Message) error {
	// Convert message to OpenAI format using adapter
	chunks, err := w.adapter.Adapt(msg)
	if err != nil {
		return err
	}

	// Send each chunk
	for _, chunk := range chunks {
		// Add role to first text chunk
		if w.firstChunk && (msg.Type == message.TypeText || msg.Type == message.TypeThinking) {
			if chunkMap, ok := chunk.(map[string]interface{}); ok {
				if choices, ok := chunkMap["choices"].([]map[string]interface{}); ok && len(choices) > 0 {
					if delta, ok := choices[0]["delta"].(map[string]interface{}); ok {
						delta["role"] = "assistant"
						w.firstChunk = false
					}
				}
			}
		}

		if err := w.sendChunk(chunk); err != nil {
			return err
		}
	}

	return nil
}

// WriteGroup writes a message group to the output stream
func (w *Writer) WriteGroup(group *message.MessageGroup) error {
	// For OpenAI, we don't send group markers
	// Just send each message individually
	for _, msg := range group.Messages {
		if err := w.Write(msg); err != nil {
			return err
		}
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
	// Send final [DONE] message for OpenAI compatibility
	return w.sendDone()
}

// sendChunk sends a chunk to the output stream in SSE format
func (w *Writer) sendChunk(chunk interface{}) error {
	// Convert chunk to JSON
	data, err := json.Marshal(chunk)
	if err != nil {
		return err
	}

	// Debug: print the chunk being sent
	fmt.Println("-----------------------------------------------")
	fmt.Println("Sending SSE chunk: ", string(data))
	fmt.Println("-----------------------------------------------")

	// Format as SSE: "data: {json}\n\n"
	sseData := append([]byte("data: "), data...)
	sseData = append(sseData, []byte("\n\n")...)

	// Send via context's writer
	return w.ctx.Send(sseData)
}

// sendDone sends the final [DONE] message
func (w *Writer) sendDone() error {
	// OpenAI SSE format uses "data: [DONE]\n\n" to signal completion
	doneData := []byte("data: [DONE]\n\n")
	return w.ctx.Send(doneData)
}
