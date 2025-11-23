package handlers

import (
	jsoniter "github.com/json-iterator/go"
	"github.com/yaoapp/kun/log"
	"github.com/yaoapp/yao/agent/context"
	"github.com/yaoapp/yao/agent/i18n"
	"github.com/yaoapp/yao/agent/output"
	"github.com/yaoapp/yao/agent/output/message"
)

// DefaultStreamHandler creates a default stream handler that sends messages via context
// This handler is used when no custom handler is provided
func DefaultStreamHandler(ctx *context.Context) context.StreamFunc {

	// Create stream state manager
	state := &streamState{
		ctx:       ctx,
		inGroup:   false,
		currentID: "",
	}

	return func(chunkType context.StreamChunkType, data []byte) int {
		trace, _ := ctx.Trace()
		if trace != nil {
			trace.Info(i18n.T(ctx.Locale, "llm.handlers.stream.info"), map[string]any{"data": string(data)})
		}

		// Handle different chunk types
		switch chunkType {
		case context.ChunkStreamStart:
			return state.handleStreamStart(data)

		case context.ChunkGroupStart:
			return state.handleGroupStart(data)

		case context.ChunkText:
			return state.handleText(data)

		case context.ChunkThinking:
			return state.handleThinking(data)

		case context.ChunkToolCall:
			return state.handleToolCall(data)

		case context.ChunkMetadata:
			return state.handleMetadata(data)

		case context.ChunkError:
			return state.handleError(data)

		case context.ChunkGroupEnd:
			return state.handleGroupEnd(data)

		case context.ChunkStreamEnd:
			return state.handleStreamEnd(data)

		default:
			// Unknown chunk type, continue
			return 0
		}
	}
}

// streamState manages the state of the streaming process
type streamState struct {
	ctx         *context.Context
	inGroup     bool
	currentID   string
	currentType string // Track the current message type (text, thinking, tool_call)
	buffer      []byte
}

// handleStreamStart handles stream start event
func (s *streamState) handleStreamStart(data []byte) int {
	// Send event message to indicate stream has started
	// This is a lifecycle event, CUI clients can show it, OpenAI clients will ignore it
	var startData context.StreamStartData
	err := jsoniter.Unmarshal(data, &startData)
	if err != nil {
		log.Error("Failed to unmarshal stream start data: %v", err)
	}
	msg := output.NewEventMessage("stream_start", "Stream started", startData)
	output.Send(s.ctx, msg)
	return 0
}

// handleGroupStart handles group start event
func (s *streamState) handleGroupStart(data []byte) int {
	s.inGroup = true
	s.currentID = generateMessageID()
	s.buffer = []byte{}
	return 0 // Continue
}

// handleText handles text content chunks
func (s *streamState) handleText(data []byte) int {
	if len(data) == 0 {
		return 0
	}

	// Ensure we have a message ID
	if s.currentID == "" {
		s.currentID = generateMessageID()
	}

	// Track current message type
	s.currentType = message.TypeText

	// Append to buffer
	s.buffer = append(s.buffer, data...)

	// Send delta message
	msg := &message.Message{
		ID:    s.currentID,
		Type:  message.TypeText,
		Delta: true,
		Props: map[string]interface{}{
			"content": string(data),
		},
	}

	if err := output.Send(s.ctx, msg); err != nil {
		// Log error but continue streaming
		return 0
	}

	return 0 // Continue
}

// handleThinking handles thinking/reasoning chunks
func (s *streamState) handleThinking(data []byte) int {
	if len(data) == 0 {
		return 0
	}

	// Ensure we have a message ID
	if s.currentID == "" {
		s.currentID = generateMessageID()
	}

	// Track current message type
	s.currentType = message.TypeThinking

	// Append to buffer
	s.buffer = append(s.buffer, data...)

	// Send delta message
	msg := &message.Message{
		ID:    s.currentID,
		Type:  message.TypeThinking,
		Delta: true,
		Props: map[string]interface{}{
			"content": string(data),
		},
	}

	if err := output.Send(s.ctx, msg); err != nil {
		return 0
	}

	return 0 // Continue
}

// handleToolCall handles tool call chunks
func (s *streamState) handleToolCall(data []byte) int {
	// Tool calls are usually complete JSON objects
	// Parse and send as tool_call message
	msg := &message.Message{
		ID:    generateMessageID(),
		Type:  message.TypeToolCall,
		Delta: true,
		Props: map[string]interface{}{
			// TODO: Parse tool call data
			"raw": string(data),
		},
	}

	output.Send(s.ctx, msg)
	return 0 // Continue
}

// handleMetadata handles metadata chunks (usage, finish_reason, etc.)
func (s *streamState) handleMetadata(data []byte) int {
	// Metadata is usually not displayed to users
	// Could be logged or stored for analytics
	return 0 // Continue
}

// handleError handles error chunks
func (s *streamState) handleError(data []byte) int {
	// Send error message
	msg := output.NewErrorMessage(string(data), "stream_error")
	output.Send(s.ctx, msg)

	return 1 // Stop streaming on error
}

// handleGroupEnd handles group end event
func (s *streamState) handleGroupEnd(data []byte) int {
	if !s.inGroup {
		return 0
	}

	// Send done message with complete content
	if s.currentID != "" && len(s.buffer) > 0 {
		// Use the tracked message type (thinking, text, tool_call, etc.)
		msgType := s.currentType
		if msgType == "" {
			msgType = message.TypeText // Fallback to text if type not set
		}

		msg := &message.Message{
			ID:   s.currentID,
			Type: msgType, // Use the actual message type from the group
			Done: true,
			Props: map[string]interface{}{
				"content": string(s.buffer),
			},
		}
		output.Send(s.ctx, msg)
	}

	// Reset state
	s.inGroup = false
	s.currentID = ""
	s.currentType = ""
	s.buffer = []byte{}

	return 0 // Continue
}

// handleStreamEnd handles stream end event
func (s *streamState) handleStreamEnd(data []byte) int {
	// Parse the stream end data
	var endData context.StreamEndData
	if err := jsoniter.Unmarshal(data, &endData); err != nil {
		log.Error("Failed to parse stream_end data: %v", err)
		output.Flush(s.ctx)
		return 0
	}

	// Send stream_end event as a message to frontend
	msg := output.NewEventMessage("stream_end", "Stream completed", endData)
	output.Send(s.ctx, msg)

	// Flush any remaining data
	output.Flush(s.ctx)
	return 0 // Continue (stream will end naturally)
}

// generateMessageID generates a unique message ID
func generateMessageID() string {
	// TODO: Implement proper ID generation
	// For now, use a simple approach
	return output.GenerateID()
}
