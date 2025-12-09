package handlers

import (
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

		case message.ChunkMessageStart:
			return state.handleMessageStart(data)

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

		case message.ChunkMessageEnd:
			return state.handleMessageEnd(data)

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
	var startData message.EventStreamStartData
	err := jsoniter.Unmarshal(data, &startData)
	if err != nil {
		log.Error("Failed to unmarshal stream start data: %v", err)
	}
	msg := output.NewEventMessage("stream_start", "Stream started", startData)
	s.ctx.Send(msg)
	return 0
}

// handleMessageStart handles message start event
func (s *streamState) handleMessageStart(data []byte) int {
	// Parse message start data first to get the message ID
	var startData message.EventMessageStartData
	if err := jsoniter.Unmarshal(data, &startData); err != nil {
		log.Error("Failed to unmarshal message start data: %v", err)
		return 0
	}

	// Use the message ID from the start data, or generate one if not provided
	messageID := startData.MessageID
	if messageID == "" {
		messageID = s.ctx.IDGenerator.GenerateMessageID()
		startData.MessageID = messageID
	}

	// Auto-set ThreadID from Stack for nested agent calls
	if startData.ThreadID == "" && s.ctx.Stack != nil && !s.ctx.Stack.IsRoot() {
		startData.ThreadID = s.ctx.Stack.ID
	}

	// Initialize message state with the correct message ID
	s.inGroup = true
	s.currentGroupID = messageID
	s.buffer = []byte{}
	s.chunkCount = 0
	s.messageSeq = 0 // Reset message sequence for each message
	s.groupStartTime = time.Now()

	// Send message_start event
	msg := output.NewEventMessage(message.EventMessageStart, "Message started", startData)
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
	// - ChunkID: Unique chunk ID (C1, C2, C3...) for this fragment
	// - MessageID: Same for all chunks of this logical message (frontend merges by message_id)
	msg := &message.Message{
		ChunkID:   s.ctx.IDGenerator.GenerateChunkID(), // Unique chunk ID
		MessageID: s.currentGroupID,                    // Message ID for merging (all chunks share this)
		Type:      message.TypeText,
		Delta:     true,
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
	// - ChunkID: Unique chunk ID (C1, C2, C3...) for this fragment
	// - MessageID: Same for all chunks of this logical message (frontend merges by message_id)
	msg := &message.Message{
		ChunkID:   s.ctx.IDGenerator.GenerateChunkID(), // Unique chunk ID
		MessageID: s.currentGroupID,                    // Message ID for merging (all chunks share this)
		Type:      message.TypeThinking,
		Delta:     true,
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
	if len(data) == 0 {
		return 0
	}

	// Track current message type
	s.currentType = message.TypeToolCall

	// Append to buffer for message_end event
	s.buffer = append(s.buffer, data...)
	s.chunkCount++
	s.messageSeq++

	// Parse the tool call delta data (JSON array from OpenAI)
	var toolCallArray []map[string]interface{}
	if err := jsoniter.Unmarshal(data, &toolCallArray); err != nil {
		// If parse fails, log and skip this chunk
		return 0
	}

	// Extract tool call fields from delta
	// OpenAI delta typically has one element, but we handle arrays safely
	var props map[string]interface{}
	var deltaAction string
	var deltaPath string

	if len(toolCallArray) == 1 {
		// Single tool call - flatten to props root level
		tc := toolCallArray[0]
		props = map[string]interface{}{}

		// Static fields (only in first chunk): use merge
		if id, ok := tc["id"].(string); ok {
			props["id"] = id
		}
		if typ, ok := tc["type"].(string); ok {
			props["type"] = typ
		}
		if index, ok := tc["index"].(float64); ok {
			props["index"] = int(index)
		}
		if fn, ok := tc["function"].(map[string]interface{}); ok {
			if name, ok := fn["name"].(string); ok {
				props["name"] = name
			}
			// Arguments field: use append
			if args, ok := fn["arguments"].(string); ok {
				props["arguments"] = args
				// If this chunk has arguments, use append action for arguments field
				deltaAction = "append"
				deltaPath = "arguments"
			}
		}

		// If no arguments in this chunk, use merge for other fields
		if deltaAction == "" {
			deltaAction = "merge"
		}
	} else {
		// Multiple tool calls in delta (rare) - keep as array
		props = map[string]interface{}{
			"calls": toolCallArray,
		}
		deltaAction = "merge"
	}

	// Send delta message
	// - ChunkID: Unique chunk ID (C1, C2, C3...) for this fragment
	// - MessageID: Same for all chunks of this logical message (frontend merges by message_id)
	// - DeltaAction: "append" for arguments chunks, "merge" for id/type/name chunks
	// - DeltaPath: "arguments" when appending arguments field
	//   OpenAI sends: first chunk has id/type/name, subsequent chunks only have arguments fragments
	msg := &message.Message{
		ChunkID:     s.ctx.IDGenerator.GenerateChunkID(), // Unique chunk ID
		MessageID:   s.currentGroupID,                    // Message ID for merging (all chunks share this)
		Type:        message.TypeToolCall,
		Delta:       true,
		DeltaAction: deltaAction, // "append" for arguments, "merge" for static fields
		DeltaPath:   deltaPath,   // "arguments" when appending
		Props:       props,       // Flattened tool call fields
	}

	if err := s.ctx.Send(msg); err != nil {
		return 0
	}

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

// handleMessageEnd handles message end event
func (s *streamState) handleMessageEnd(data []byte) int {
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

	// Get ThreadID from Stack for nested agent calls
	var threadID string
	if s.ctx.Stack != nil && !s.ctx.Stack.IsRoot() {
		threadID = s.ctx.Stack.ID
	}

	// Get BlockID from metadata if available
	var blockID string
	if s.ctx != nil {
		if metadata := s.ctx.GetMessageMetadata(s.currentGroupID); metadata != nil {
			blockID = metadata.BlockID
		}
	}

	// Buffer the complete LLM message for storage
	// Delta chunks are not stored, but we need to save the final complete content
	// Skip if History is disabled in options
	shouldSkipHistory := s.ctx.Stack != nil && s.ctx.Stack.Options != nil &&
		s.ctx.Stack.Options.Skip != nil && s.ctx.Stack.Options.Skip.History

	if s.ctx.Buffer != nil && len(s.buffer) > 0 && !shouldSkipHistory {
		assistantID := ""
		if s.ctx.Stack != nil {
			assistantID = s.ctx.Stack.AssistantID
		}

		// Build props based on message type
		var props map[string]interface{}
		if msgType == message.TypeToolCall {
			// For tool calls, try to parse the accumulated buffer as JSON
			var toolCallData interface{}
			if err := jsoniter.Unmarshal(s.buffer, &toolCallData); err == nil {
				props = map[string]interface{}{
					"calls": toolCallData,
				}
			} else {
				props = map[string]interface{}{
					"content": string(s.buffer),
				}
			}
		} else {
			// For text/thinking, content is the accumulated text
			props = map[string]interface{}{
				"content": string(s.buffer),
			}
		}

		s.ctx.Buffer.AddAssistantMessage(
			s.currentGroupID, // Use the message ID
			msgType,
			props,
			blockID,
			threadID,
			assistantID,
			nil,
		)
	}

	// Build EventMessageEndData with complete content
	endData := message.EventMessageEndData{
		MessageID:  s.currentGroupID, // Use the message ID
		Type:       msgType,
		Timestamp:  time.Now().UnixMilli(),
		ThreadID:   threadID, // Include ThreadID for concurrent stream identification
		DurationMs: durationMs,
		ChunkCount: s.chunkCount,
		Status:     "completed",
		Extra: map[string]interface{}{
			"content": string(s.buffer), // Include complete content in the event
		},
	}

	// Send message_end event
	msg := output.NewEventMessage(message.EventMessageEnd, "Message completed", endData)
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
	var endData message.EventStreamEndData
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
