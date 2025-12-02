package openai

import (
	"encoding/json"
	"net/http"

	"github.com/yaoapp/yao/agent/i18n"
	"github.com/yaoapp/yao/agent/output/message"
	traceTypes "github.com/yaoapp/yao/trace/types"
)

// Writer implements the message.Writer interface for OpenAI-compatible clients
type Writer struct {
	Writer     http.ResponseWriter
	Trace      traceTypes.Manager
	Locale     string
	adapter    *Adapter
	firstChunk bool // Track if this is the first chunk to add role
}

// NewWriter creates a new OpenAI writer
func NewWriter(options message.Options) (*Writer, error) {
	// Get model capabilities from context (set by assistant)
	var capabilities *ModelCapabilities
	if options.Capabilities != nil && options.Capabilities.Reasoning {
		v := true
		capabilities = &ModelCapabilities{
			Reasoning: &v,
		}
	}

	// Create adapter with capabilities, base URL, and locale
	adapter := NewAdapter(
		WithCapabilities(capabilities),
		WithBaseURL(getBaseURL(options.BaseURL)),
		WithLocale(options.Locale),
	)

	return &Writer{
		adapter:    adapter,
		Writer:     options.Writer,
		Locale:     options.Locale,
		firstChunk: true, // First chunk should include role
	}, nil
}

// getBaseURL gets the base URL from context or environment
func getBaseURL(baseURL string) string {
	// @todo: get from context metadata
	return "http://localhost:8000/__yao_admin_root"

	// // Try to get from context metadata
	// if ctx.Metadata != nil {
	// 	if baseURL, ok := ctx.Metadata["base_url"].(string); ok && baseURL != "" {
	// 		return baseURL
	// 	}
	// }

	// // TODO: Get from environment variable or config
	// return ""
}

// Write writes a single message to the output stream
func (w *Writer) Write(msg *message.Message) error {
	// Convert message to OpenAI format using adapter
	chunks, err := w.adapter.Adapt(msg)
	if err != nil {
		if w.Trace != nil {
			w.Trace.Error(i18n.T(w.Locale, "output.openai.writer.adapt_error"), map[string]any{ // "OpenAI Writer: Failed to adapt message"
				"error":        err.Error(),
				"message_type": msg.Type,
			})
		}
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
			if w.Trace != nil {
				w.Trace.Error(i18n.T(w.Locale, "output.openai.writer.chunk_error"), map[string]any{"error": err.Error()}) // "OpenAI Writer: Failed to send chunk"
			}
			return err
		}
	}

	return nil
}

// WriteGroup writes a message group to the output stream
func (w *Writer) WriteGroup(group *message.Group) error {
	// For OpenAI, we don't send group markers
	// Just send each message individually
	for _, msg := range group.Messages {
		if err := w.Write(msg); err != nil {
			if w.Trace != nil {
				w.Trace.Error(i18n.T(w.Locale, "output.openai.writer.group_error"), map[string]any{ // "OpenAI Writer: Failed to write message in group"
					"error":        err.Error(),
					"group_id":     group.ID,
					"message_type": msg.Type,
				})
			}
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

func (w *Writer) sendData(data []byte) error {
	if w.Writer == nil {
		return nil // No writer, silently ignore
	}
	_, err := w.Writer.Write(data)
	return err
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

// sendChunk sends a chunk to the output stream in SSE format
func (w *Writer) sendChunk(chunk interface{}) error {
	// Convert chunk to JSON
	data, err := json.Marshal(chunk)
	if err != nil {
		if w.Trace != nil {
			w.Trace.Error(i18n.T(w.Locale, "output.openai.writer.marshal_error"), map[string]any{"error": err.Error()}) // "OpenAI Writer: Failed to marshal chunk"
		}
		return err
	}

	// Format as SSE: "data: {json}\n\n"
	sseData := append([]byte("data: "), data...)
	sseData = append(sseData, []byte("\n\n")...)

	// Log outgoing data to trace for debugging
	if w.Trace != nil {
		w.Trace.Debug("OpenAI Writer: Sending chunk to client", map[string]any{
			"data": string(data),
		})
	}

	// Send via context's writer
	if err := w.sendData(sseData); err != nil {
		if w.Trace != nil {
			w.Trace.Error(i18n.T(w.Locale, "output.openai.writer.send_error"), map[string]any{"error": err.Error()}) // "OpenAI Writer: Failed to send data to client"
		}
		return err
	}

	// Flush immediately to ensure real-time streaming
	// Cast to http.ResponseWriter and call Flush if available
	w.flush()

	return nil
}

// sendDone sends the final [DONE] message
func (w *Writer) sendDone() error {
	// Log completion to trace
	if w.Trace != nil {
		w.Trace.Debug("OpenAI Writer: Sending [DONE] to client")
	}

	// OpenAI SSE format uses "data: [DONE]\n\n" to signal completion
	doneData := []byte("data: [DONE]\n\n")
	if err := w.sendData(doneData); err != nil {
		if w.Trace != nil {
			w.Trace.Error(i18n.T(w.Locale, "output.openai.writer.done_error"), map[string]any{"error": err.Error()}) // "OpenAI Writer: Failed to send [DONE] to client"
		}
		return err
	}

	// Flush the final [DONE] message
	w.flush()
	return nil
}
