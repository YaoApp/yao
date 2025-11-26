package context

import (
	"fmt"

	"github.com/yaoapp/gou/runtime/v8/bridge"
	"github.com/yaoapp/yao/agent/output/message"
	"rogchap.com/v8go"
)

// parseMessage parses a JavaScript value into a message.Message
func parseMessage(v8ctx *v8go.Context, jsValue *v8go.Value) (*message.Message, error) {
	// Handle string shorthand: convert to text message
	if jsValue.IsString() {
		return &message.Message{
			Type: message.TypeText,
			Props: map[string]interface{}{
				"content": jsValue.String(),
			},
		}, nil
	}

	// Handle object
	if !jsValue.IsObject() {
		return nil, fmt.Errorf("message must be a string or object")
	}

	// Convert to Go map
	goValue, err := bridge.GoValue(jsValue, v8ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to convert message: %w", err)
	}

	msgMap, ok := goValue.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("message must be an object")
	}

	// Build message
	msg := &message.Message{}

	// Type field (required)
	if msgType, ok := msgMap["type"].(string); ok {
		msg.Type = msgType
	} else {
		return nil, fmt.Errorf("message.type is required and must be a string")
	}

	// Props field (optional)
	if props, ok := msgMap["props"].(map[string]interface{}); ok {
		msg.Props = props
	}

	// Optional fields - Streaming control
	if chunkID, ok := msgMap["chunk_id"].(string); ok {
		msg.ChunkID = chunkID
	}
	if messageID, ok := msgMap["message_id"].(string); ok {
		msg.MessageID = messageID
	}
	if blockID, ok := msgMap["block_id"].(string); ok {
		msg.BlockID = blockID
	}
	if threadID, ok := msgMap["thread_id"].(string); ok {
		msg.ThreadID = threadID
	}

	// Delta control
	if delta, ok := msgMap["delta"].(bool); ok {
		msg.Delta = delta
	}
	if deltaPath, ok := msgMap["delta_path"].(string); ok {
		msg.DeltaPath = deltaPath
	}
	if deltaAction, ok := msgMap["delta_action"].(string); ok {
		msg.DeltaAction = deltaAction
	}
	if typeChange, ok := msgMap["type_change"].(bool); ok {
		msg.TypeChange = typeChange
	}

	// Metadata (optional)
	if metadataMap, ok := msgMap["metadata"].(map[string]interface{}); ok {
		metadata := &message.Metadata{}
		if timestamp, ok := metadataMap["timestamp"].(float64); ok {
			metadata.Timestamp = int64(timestamp)
		}
		if sequence, ok := metadataMap["sequence"].(float64); ok {
			metadata.Sequence = int(sequence)
		}
		if traceID, ok := metadataMap["trace_id"].(string); ok {
			metadata.TraceID = traceID
		}
		msg.Metadata = metadata
	}

	return msg, nil
}

// parseGroup parses a JavaScript value into a message.Group
func parseGroup(v8ctx *v8go.Context, jsValue *v8go.Value) (*message.Group, error) {
	// Must be an object
	if !jsValue.IsObject() {
		return nil, fmt.Errorf("group must be an object")
	}

	// Convert to Go map
	goValue, err := bridge.GoValue(jsValue, v8ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to convert group: %w", err)
	}

	groupMap, ok := goValue.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("group must be an object")
	}

	// Build group
	group := &message.Group{}

	// ID field (required)
	if id, ok := groupMap["id"].(string); ok {
		group.ID = id
	} else {
		return nil, fmt.Errorf("group.id is required and must be a string")
	}

	// Messages field (required)
	if messagesArray, ok := groupMap["messages"].([]interface{}); ok {
		group.Messages = make([]*message.Message, 0, len(messagesArray))
		for i, msgInterface := range messagesArray {
			// Convert to map
			msgMap, ok := msgInterface.(map[string]interface{})
			if !ok {
				return nil, fmt.Errorf("group.messages[%d] must be an object", i)
			}

			// Convert map to Message
			msg := &message.Message{}

			// Type field (required)
			if msgType, ok := msgMap["type"].(string); ok {
				msg.Type = msgType
			} else {
				return nil, fmt.Errorf("group.messages[%d].type is required", i)
			}

			// Props field (optional)
			if props, ok := msgMap["props"].(map[string]interface{}); ok {
				msg.Props = props
			}

			// Optional fields - Streaming control
			if chunkID, ok := msgMap["chunk_id"].(string); ok {
				msg.ChunkID = chunkID
			}
			if messageID, ok := msgMap["message_id"].(string); ok {
				msg.MessageID = messageID
			}
			if blockID, ok := msgMap["block_id"].(string); ok {
				msg.BlockID = blockID
			}
			if threadID, ok := msgMap["thread_id"].(string); ok {
				msg.ThreadID = threadID
			}

			// Delta control
			if delta, ok := msgMap["delta"].(bool); ok {
				msg.Delta = delta
			}
			if deltaPath, ok := msgMap["delta_path"].(string); ok {
				msg.DeltaPath = deltaPath
			}
			if deltaAction, ok := msgMap["delta_action"].(string); ok {
				msg.DeltaAction = deltaAction
			}
			if typeChange, ok := msgMap["type_change"].(bool); ok {
				msg.TypeChange = typeChange
			}

			// Metadata (optional)
			if metadataMap, ok := msgMap["metadata"].(map[string]interface{}); ok {
				metadata := &message.Metadata{}
				if timestamp, ok := metadataMap["timestamp"].(float64); ok {
					metadata.Timestamp = int64(timestamp)
				}
				if sequence, ok := metadataMap["sequence"].(float64); ok {
					metadata.Sequence = int(sequence)
				}
				if traceID, ok := metadataMap["trace_id"].(string); ok {
					metadata.TraceID = traceID
				}
				msg.Metadata = metadata
			}

			group.Messages = append(group.Messages, msg)
		}
	} else {
		return nil, fmt.Errorf("group.messages is required and must be an array")
	}

	// Metadata (optional)
	if metadataMap, ok := groupMap["metadata"].(map[string]interface{}); ok {
		metadata := &message.Metadata{}
		if timestamp, ok := metadataMap["timestamp"].(float64); ok {
			metadata.Timestamp = int64(timestamp)
		}
		if sequence, ok := metadataMap["sequence"].(float64); ok {
			metadata.Sequence = int(sequence)
		}
		if traceID, ok := metadataMap["trace_id"].(string); ok {
			metadata.TraceID = traceID
		}
		group.Metadata = metadata
	}

	return group, nil
}
