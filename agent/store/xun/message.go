package xun

import (
	"fmt"
	"time"

	"github.com/google/uuid"
	jsoniter "github.com/json-iterator/go"
	"github.com/yaoapp/yao/agent/store/types"
)

// =============================================================================
// Message Management
// =============================================================================

// SaveMessages batch saves messages for a chat using a single database call
// This is the primary write method - messages are buffered during execution
// and batch-written at the end of a request
func (store *Xun) SaveMessages(chatID string, messages []*types.Message) error {
	if chatID == "" {
		return fmt.Errorf("chat_id is required")
	}
	if len(messages) == 0 {
		return nil // Nothing to save
	}

	// Prepare batch insert data
	now := time.Now()
	rows := make([]map[string]interface{}, 0, len(messages))

	for _, msg := range messages {
		if msg == nil {
			continue
		}

		// Generate message_id if not provided
		messageID := msg.MessageID
		if messageID == "" {
			messageID = uuid.New().String()
		}

		// Validate required fields
		if msg.Role == "" {
			return fmt.Errorf("message role is required")
		}
		if msg.Type == "" {
			return fmt.Errorf("message type is required")
		}
		if msg.Props == nil {
			return fmt.Errorf("message props is required")
		}

		// Serialize JSON fields
		propsJSON, err := jsoniter.MarshalToString(msg.Props)
		if err != nil {
			return fmt.Errorf("failed to marshal props: %w", err)
		}

		// Build row with all fields (including nullable ones for consistent batch insert)
		row := map[string]interface{}{
			"message_id":   messageID,
			"chat_id":      chatID,
			"role":         msg.Role,
			"type":         msg.Type,
			"props":        propsJSON,
			"sequence":     msg.Sequence,
			"request_id":   nil,
			"block_id":     nil,
			"thread_id":    nil,
			"assistant_id": nil,
			"connector":    nil,
			"mode":         nil,
			"metadata":     nil,
			"created_at":   now,
			"updated_at":   now,
		}

		// Set nullable fields if they have values
		if msg.RequestID != "" {
			row["request_id"] = msg.RequestID
		}
		if msg.BlockID != "" {
			row["block_id"] = msg.BlockID
		}
		if msg.ThreadID != "" {
			row["thread_id"] = msg.ThreadID
		}
		if msg.AssistantID != "" {
			row["assistant_id"] = msg.AssistantID
		}
		if msg.Connector != "" {
			row["connector"] = msg.Connector
		}
		if msg.Mode != "" {
			row["mode"] = msg.Mode
		}
		if msg.Metadata != nil {
			metadataJSON, err := jsoniter.MarshalToString(msg.Metadata)
			if err != nil {
				return fmt.Errorf("failed to marshal metadata: %w", err)
			}
			row["metadata"] = metadataJSON
		}

		rows = append(rows, row)
	}

	if len(rows) == 0 {
		return nil
	}

	// Single batch insert - one database call for all messages
	return store.newQueryMessage().Insert(rows)
}

// GetMessages retrieves messages for a chat with filtering
func (store *Xun) GetMessages(chatID string, filter types.MessageFilter) ([]*types.Message, error) {
	if chatID == "" {
		return nil, fmt.Errorf("chat_id is required")
	}

	qb := store.newQueryMessage().
		Where("chat_id", chatID).
		WhereNull("deleted_at")

	// Apply filters
	if filter.RequestID != "" {
		qb.Where("request_id", filter.RequestID)
	}
	if filter.Role != "" {
		qb.Where("role", filter.Role)
	}
	if filter.BlockID != "" {
		qb.Where("block_id", filter.BlockID)
	}
	if filter.ThreadID != "" {
		qb.Where("thread_id", filter.ThreadID)
	}
	if filter.Type != "" {
		qb.Where("type", filter.Type)
	}

	// Apply pagination (MySQL requires LIMIT when using OFFSET)
	if filter.Limit > 0 {
		qb.Limit(filter.Limit)
		if filter.Offset > 0 {
			qb.Offset(filter.Offset)
		}
	} else if filter.Offset > 0 {
		// If only offset is specified, use a large limit
		qb.Limit(1000000).Offset(filter.Offset)
	}

	// Order by created_at first, then by sequence within the same request
	qb.OrderBy("created_at", "asc").OrderBy("sequence", "asc")

	rows, err := qb.Get()
	if err != nil {
		return nil, err
	}

	messages := make([]*types.Message, 0, len(rows))
	for _, row := range rows {
		data := row.ToMap()
		if data == nil || data["message_id"] == nil {
			continue
		}

		msg, err := store.rowToMessage(data)
		if err != nil {
			continue
		}
		messages = append(messages, msg)
	}

	return messages, nil
}

// UpdateMessage updates a single message
func (store *Xun) UpdateMessage(messageID string, updates map[string]interface{}) error {
	if messageID == "" {
		return fmt.Errorf("message_id is required")
	}
	if len(updates) == 0 {
		return fmt.Errorf("no fields to update")
	}

	// Check if message exists
	exists, err := store.newQueryMessage().
		Where("message_id", messageID).
		WhereNull("deleted_at").
		Exists()
	if err != nil {
		return err
	}
	if !exists {
		return fmt.Errorf("message %s not found", messageID)
	}

	// Prepare update data
	data := make(map[string]interface{})

	for key, value := range updates {
		// Skip system fields
		if key == "message_id" || key == "chat_id" || key == "created_at" {
			continue
		}

		// Handle JSON fields
		if key == "props" || key == "metadata" {
			if value != nil {
				jsonStr, err := jsoniter.MarshalToString(value)
				if err != nil {
					return fmt.Errorf("failed to marshal %s: %w", key, err)
				}
				data[key] = jsonStr
			} else {
				data[key] = nil
			}
			continue
		}

		data[key] = value
	}

	// Always update updated_at
	data["updated_at"] = time.Now()

	if len(data) == 0 {
		return fmt.Errorf("no valid fields to update")
	}

	_, err = store.newQueryMessage().
		Where("message_id", messageID).
		Update(data)

	return err
}

// DeleteMessages soft deletes specific messages from a chat
func (store *Xun) DeleteMessages(chatID string, messageIDs []string) error {
	if chatID == "" {
		return fmt.Errorf("chat_id is required")
	}
	if len(messageIDs) == 0 {
		return nil // Nothing to delete
	}

	// Soft delete all specified messages in one query
	_, err := store.newQueryMessage().
		Where("chat_id", chatID).
		WhereIn("message_id", messageIDs).
		WhereNull("deleted_at").
		Update(map[string]interface{}{
			"deleted_at": time.Now(),
			"updated_at": time.Now(),
		})

	return err
}

// GetMessageByID retrieves a single message by ID
func (store *Xun) GetMessageByID(messageID string) (*types.Message, error) {
	if messageID == "" {
		return nil, fmt.Errorf("message_id is required")
	}

	row, err := store.newQueryMessage().
		Where("message_id", messageID).
		WhereNull("deleted_at").
		First()
	if err != nil {
		return nil, err
	}

	if row == nil {
		return nil, fmt.Errorf("message %s not found", messageID)
	}

	data := row.ToMap()
	if len(data) == 0 || data["message_id"] == nil {
		return nil, fmt.Errorf("message %s not found", messageID)
	}

	return store.rowToMessage(data)
}

// GetMessageCount returns the count of messages for a chat
func (store *Xun) GetMessageCount(chatID string) (int64, error) {
	if chatID == "" {
		return 0, fmt.Errorf("chat_id is required")
	}

	return store.newQueryMessage().
		Where("chat_id", chatID).
		WhereNull("deleted_at").
		Count()
}

// GetLastSequence returns the last sequence number for a chat
func (store *Xun) GetLastSequence(chatID string) (int, error) {
	if chatID == "" {
		return 0, fmt.Errorf("chat_id is required")
	}

	row, err := store.newQueryMessage().
		Where("chat_id", chatID).
		WhereNull("deleted_at").
		OrderBy("sequence", "desc").
		First()
	if err != nil {
		return 0, err
	}

	if row == nil {
		return 0, nil
	}

	data := row.ToMap()
	return getInt(data, "sequence"), nil
}

// =============================================================================
// Helper Functions
// =============================================================================

// rowToMessage converts a database row to a Message struct
func (store *Xun) rowToMessage(data map[string]interface{}) (*types.Message, error) {
	msg := &types.Message{
		MessageID:   getString(data, "message_id"),
		ChatID:      getString(data, "chat_id"),
		RequestID:   getString(data, "request_id"),
		Role:        getString(data, "role"),
		Type:        getString(data, "type"),
		BlockID:     getString(data, "block_id"),
		ThreadID:    getString(data, "thread_id"),
		AssistantID: getString(data, "assistant_id"),
		Connector:   getString(data, "connector"),
		Mode:        getString(data, "mode"),
		Sequence:    getInt(data, "sequence"),
	}

	// Handle timestamps
	if createdAt := getTime(data, "created_at"); createdAt != nil {
		msg.CreatedAt = *createdAt
	}
	if updatedAt := getTime(data, "updated_at"); updatedAt != nil {
		msg.UpdatedAt = *updatedAt
	}

	// Handle props (required)
	if props := data["props"]; props != nil {
		if propsStr, ok := props.(string); ok && propsStr != "" {
			var propsMap map[string]interface{}
			if err := jsoniter.UnmarshalFromString(propsStr, &propsMap); err == nil {
				msg.Props = propsMap
			}
		} else if propsMap, ok := props.(map[string]interface{}); ok {
			msg.Props = propsMap
		}
	}

	// Handle metadata (optional)
	if metadata := data["metadata"]; metadata != nil {
		if metaStr, ok := metadata.(string); ok && metaStr != "" {
			var metaMap map[string]interface{}
			if err := jsoniter.UnmarshalFromString(metaStr, &metaMap); err == nil {
				msg.Metadata = metaMap
			}
		} else if metaMap, ok := metadata.(map[string]interface{}); ok {
			msg.Metadata = metaMap
		}
	}

	return msg, nil
}
