package handlers

import (
	"fmt"

	"github.com/yaoapp/yao/agent/context"
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
		fmt.Println("-----------------------------------------------")
		fmt.Println("Chunk Type: ", string(chunkType))
		fmt.Println("Data: ", string(data))
		fmt.Println("-----------------------------------------------")
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
	ctx       *context.Context
	inGroup   bool
	currentID string
	buffer    []byte
}

// handleStreamStart handles stream start event
func (s *streamState) handleStreamStart(data []byte) int {
	// Send event message to indicate stream has started
	// This is a lifecycle event, CUI clients can show it, OpenAI clients will ignore it
	msg := output.NewEventMessage("stream_start", "Connecting...", nil)
	output.Send(s.ctx, msg)
	return 0 // Continue
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
		msg := &message.Message{
			ID:   s.currentID,
			Type: message.TypeText, // Default to text
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
	s.buffer = []byte{}

	return 0 // Continue
}

// handleStreamEnd handles stream end event
func (s *streamState) handleStreamEnd(data []byte) int {
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
