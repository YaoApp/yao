package handlers

import (
	"fmt"
	"time"

	jsoniter "github.com/json-iterator/go"
	"github.com/yaoapp/kun/log"
	"github.com/yaoapp/yao/agent/context"
	"github.com/yaoapp/yao/agent/i18n"
	"github.com/yaoapp/yao/agent/output"
	"github.com/yaoapp/yao/agent/output/message"
)

// DefaultStreamHandler creates a default stream handler that sends messages via context
// This handler is used when no custom handler is provided
func DefaultStreamHandler(ctx *context.Context) message.StreamFunc {

	// Create stream state manager
	state := &streamState{
		ctx:            ctx,
		inGroup:        false,
		currentGroupID: "",
		messageSeq:     0,
	}

	return func(chunkType message.StreamChunkType, data []byte) int {
		trace, _ := ctx.Trace()
		if trace != nil {
			trace.Info(i18n.T(ctx.Locale, "llm.handlers.stream.info"), map[string]any{"data": string(data)})
		}

		// Handle different chunk types
		switch chunkType {
		case message.ChunkStreamStart:
			return state.handleStreamStart(data)

		case message.ChunkGroupStart:
			return state.handleGroupStart(data)

		case message.ChunkText:
			return state.handleText(data)

		case message.ChunkThinking:
			return state.handleThinking(data)

		case message.ChunkToolCall:
			return state.handleToolCall(data)

		case message.ChunkMetadata:
			return state.handleMetadata(data)

		case message.ChunkError:
			return state.handleError(data)

		case message.ChunkGroupEnd:
			return state.handleGroupEnd(data)

		case message.ChunkStreamEnd:
			return state.handleStreamEnd(data)

		default:
			// Unknown chunk type, continue
			return 0
		}
	}
}

// streamState manages the state of the streaming process
type streamState struct {
	ctx            *context.Context
	inGroup        bool
	currentGroupID string // Current group ID (shared by all chunks in the group)
	currentType    string // Track the current message type (text, thinking, tool_call)
	buffer         []byte
	chunkCount     int       // Track number of chunks in current group
	messageSeq     int       // Message sequence number (for generating readable IDs)
	groupStartTime time.Time // Track when group started
}

// handleStreamStart handles stream start event
func (s *streamState) handleStreamStart(data []byte) int {
	// Send event message to indicate stream has started
	// This is a lifecycle event, CUI clients can show it, OpenAI clients will ignore it
	var startData message.StreamStartData
	err := jsoniter.Unmarshal(data, &startData)
	if err != nil {
		log.Error("Failed to unmarshal stream start data: %v", err)
	}
	msg := output.NewEventMessage("stream_start", "Stream started", startData)
	s.ctx.Send(msg)
	return 0
}

// handleGroupStart handles group start event
func (s *streamState) handleGroupStart(data []byte) int {
	// Parse group start data first to get the group ID
	var startData message.GroupStartData
	if err := jsoniter.Unmarshal(data, &startData); err != nil {
		log.Error("Failed to unmarshal group start data: %v", err)
		return 0
	}

	// Use the group ID from the start data, or generate one if not provided
	groupID := startData.GroupID
	if groupID == "" {
		groupID = generateMessageID()
		startData.GroupID = groupID
	}

	// Initialize group state with the correct group ID
	s.inGroup = true
	s.currentGroupID = groupID
	s.buffer = []byte{}
	s.chunkCount = 0
	s.messageSeq = 0 // Reset message sequence for each group
	s.groupStartTime = time.Now()

	// Send group_start event
	msg := output.NewEventMessage(message.EventGroupStart, "Group started", startData)
	s.ctx.Send(msg)

	return 0 // Continue
}

// handleText handles text content chunks
func (s *streamState) handleText(data []byte) int {
	if len(data) == 0 {
		return 0
	}

	// Track current message type
	s.currentType = message.TypeText

	// Append to buffer
	s.buffer = append(s.buffer, data...)
	s.chunkCount++
	s.messageSeq++

	// Send delta message
	// - ID: Sequential message ID (e.g., msg_001, msg_002) for readability
	// - GroupID: Same for all chunks of this logical message (frontend merges by group_id)
	msg := &message.Message{
		ID:      s.generateSequentialID(), // Sequential ID for this chunk
		GroupID: s.currentGroupID,         // Group ID for merging (all chunks share this)
		Type:    message.TypeText,
		Delta:   true,
		Props: map[string]interface{}{
			"content": string(data),
		},
	}

	if err := s.ctx.Send(msg); err != nil {
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

	// Track current message type
	s.currentType = message.TypeThinking

	// Append to buffer
	s.buffer = append(s.buffer, data...)
	s.chunkCount++
	s.messageSeq++

	// Send delta message
	// - ID: Sequential message ID (e.g., msg_001, msg_002) for readability
	// - GroupID: Same for all chunks of this logical message (frontend merges by group_id)
	msg := &message.Message{
		ID:      s.generateSequentialID(), // Sequential ID for this chunk
		GroupID: s.currentGroupID,         // Group ID for merging (all chunks share this)
		Type:    message.TypeThinking,
		Delta:   true,
		Props: map[string]interface{}{
			"content": string(data),
		},
	}

	if err := s.ctx.Send(msg); err != nil {
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

	s.ctx.Send(msg)
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
	s.ctx.Send(msg)

	return 1 // Stop streaming on error
}

// handleGroupEnd handles group end event
func (s *streamState) handleGroupEnd(data []byte) int {
	if !s.inGroup {
		return 0
	}

	// Calculate duration
	durationMs := time.Since(s.groupStartTime).Milliseconds()

	// Use the tracked message type (thinking, text, tool_call, etc.)
	msgType := s.currentType
	if msgType == "" {
		msgType = message.TypeText // Fallback to text if type not set
	}

	// Build GroupEndData with complete content
	endData := message.GroupEndData{
		GroupID:    s.currentGroupID, // Use the group ID, not message ID
		Type:       msgType,
		Timestamp:  time.Now().UnixMilli(),
		DurationMs: durationMs,
		ChunkCount: s.chunkCount,
		Status:     "completed",
		Extra: map[string]interface{}{
			"content": string(s.buffer), // Include complete content in the event
		},
	}

	// Send group_end event
	msg := output.NewEventMessage(message.EventGroupEnd, "Group completed", endData)
	s.ctx.Send(msg)

	// Reset state
	s.inGroup = false
	s.currentGroupID = ""
	s.currentType = ""
	s.buffer = []byte{}
	s.chunkCount = 0

	return 0 // Continue
}

// handleStreamEnd handles stream end event
func (s *streamState) handleStreamEnd(data []byte) int {
	// Parse the stream end data
	var endData message.StreamEndData
	if err := jsoniter.Unmarshal(data, &endData); err != nil {
		log.Error("Failed to parse stream_end data: %v", err)
		s.ctx.Flush()
		return 0
	}

	// Send stream_end event as a message to frontend
	msg := output.NewEventMessage("stream_end", "Stream completed", endData)
	s.ctx.Send(msg)

	// Flush any remaining data
	s.ctx.Flush()
	return 0 // Continue (stream will end naturally)
}

// generateSequentialID generates a sequential message ID for better readability
func (s *streamState) generateSequentialID() string {
	// Format: 1, 2, 3, etc.
	// This makes it easier for developers to track message order in logs
	return fmt.Sprintf("%d", s.messageSeq)
}

// generateMessageID generates a unique message ID
func generateMessageID() string {
	// TODO: Implement proper ID generation
	// For now, use a simple approach
	return output.GenerateID()
}
