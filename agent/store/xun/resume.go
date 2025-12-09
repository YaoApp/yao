package xun

import (
	"fmt"
	"time"

	"github.com/google/uuid"
	jsoniter "github.com/json-iterator/go"
	"github.com/yaoapp/yao/agent/store/types"
)

// =============================================================================
// Resume Management (only called on failure/interrupt)
// =============================================================================

// SaveResume batch saves resume records using a single database call
// Only called when request is interrupted or failed
func (store *Xun) SaveResume(records []*types.Resume) error {
	if len(records) == 0 {
		return nil // Nothing to save
	}

	// Prepare batch insert data
	now := time.Now()
	rows := make([]map[string]interface{}, 0, len(records))

	for _, record := range records {
		if record == nil {
			continue
		}

		// Generate resume_id if not provided
		resumeID := record.ResumeID
		if resumeID == "" {
			resumeID = uuid.New().String()
		}

		// Validate required fields
		if record.ChatID == "" {
			return fmt.Errorf("chat_id is required")
		}
		if record.RequestID == "" {
			return fmt.Errorf("request_id is required")
		}
		if record.AssistantID == "" {
			return fmt.Errorf("assistant_id is required")
		}
		if record.StackID == "" {
			return fmt.Errorf("stack_id is required")
		}
		if record.Type == "" {
			return fmt.Errorf("type is required")
		}
		if record.Status == "" {
			return fmt.Errorf("status is required")
		}

		// Build row with all fields (including nullable ones for consistent batch insert)
		row := map[string]interface{}{
			"resume_id":       resumeID,
			"chat_id":         record.ChatID,
			"request_id":      record.RequestID,
			"assistant_id":    record.AssistantID,
			"stack_id":        record.StackID,
			"stack_parent_id": nil,
			"stack_depth":     record.StackDepth,
			"type":            record.Type,
			"status":          record.Status,
			"input":           nil,
			"output":          nil,
			"space_snapshot":  nil,
			"error":           nil,
			"sequence":        record.Sequence,
			"metadata":        nil,
			"created_at":      now,
			"updated_at":      now,
		}

		// Set nullable fields if they have values
		if record.StackParentID != "" {
			row["stack_parent_id"] = record.StackParentID
		}
		if record.Input != nil {
			inputJSON, err := jsoniter.MarshalToString(record.Input)
			if err != nil {
				return fmt.Errorf("failed to marshal input: %w", err)
			}
			row["input"] = inputJSON
		}
		if record.Output != nil {
			outputJSON, err := jsoniter.MarshalToString(record.Output)
			if err != nil {
				return fmt.Errorf("failed to marshal output: %w", err)
			}
			row["output"] = outputJSON
		}
		if record.SpaceSnapshot != nil {
			snapshotJSON, err := jsoniter.MarshalToString(record.SpaceSnapshot)
			if err != nil {
				return fmt.Errorf("failed to marshal space_snapshot: %w", err)
			}
			row["space_snapshot"] = snapshotJSON
		}
		if record.Error != "" {
			row["error"] = record.Error
		}
		if record.Metadata != nil {
			metadataJSON, err := jsoniter.MarshalToString(record.Metadata)
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

	// Single batch insert - one database call for all records
	return store.newQueryResume().Insert(rows)
}

// GetResume retrieves all resume records for a chat
func (store *Xun) GetResume(chatID string) ([]*types.Resume, error) {
	if chatID == "" {
		return nil, fmt.Errorf("chat_id is required")
	}

	rows, err := store.newQueryResume().
		Where("chat_id", chatID).
		WhereNull("deleted_at").
		OrderBy("sequence", "asc").
		Get()
	if err != nil {
		return nil, err
	}

	records := make([]*types.Resume, 0, len(rows))
	for _, row := range rows {
		data := row.ToMap()
		if data == nil || data["resume_id"] == nil {
			continue
		}

		record, err := store.rowToResume(data)
		if err != nil {
			continue
		}
		records = append(records, record)
	}

	return records, nil
}

// GetLastResume retrieves the last (most recent) resume record for a chat
func (store *Xun) GetLastResume(chatID string) (*types.Resume, error) {
	if chatID == "" {
		return nil, fmt.Errorf("chat_id is required")
	}

	row, err := store.newQueryResume().
		Where("chat_id", chatID).
		WhereNull("deleted_at").
		OrderBy("sequence", "desc").
		First()
	if err != nil {
		return nil, err
	}

	if row == nil {
		return nil, nil // No resume records found
	}

	data := row.ToMap()
	if len(data) == 0 || data["resume_id"] == nil {
		return nil, nil
	}

	return store.rowToResume(data)
}

// GetResumeByStackID retrieves resume records for a specific stack
func (store *Xun) GetResumeByStackID(stackID string) ([]*types.Resume, error) {
	if stackID == "" {
		return nil, fmt.Errorf("stack_id is required")
	}

	rows, err := store.newQueryResume().
		Where("stack_id", stackID).
		WhereNull("deleted_at").
		OrderBy("sequence", "asc").
		Get()
	if err != nil {
		return nil, err
	}

	records := make([]*types.Resume, 0, len(rows))
	for _, row := range rows {
		data := row.ToMap()
		if data == nil || data["resume_id"] == nil {
			continue
		}

		record, err := store.rowToResume(data)
		if err != nil {
			continue
		}
		records = append(records, record)
	}

	return records, nil
}

// GetStackPath returns the stack path from root to the given stack
// Returns: [root_stack_id, ..., current_stack_id]
func (store *Xun) GetStackPath(stackID string) ([]string, error) {
	if stackID == "" {
		return nil, fmt.Errorf("stack_id is required")
	}

	path := []string{stackID}
	currentStackID := stackID

	// Walk up the stack tree by following stack_parent_id
	for {
		row, err := store.newQueryResume().
			Where("stack_id", currentStackID).
			WhereNull("deleted_at").
			First()
		if err != nil {
			return nil, err
		}

		if row == nil {
			break
		}

		data := row.ToMap()
		parentID := getString(data, "stack_parent_id")
		if parentID == "" {
			break // Reached root
		}

		// Prepend parent to path
		path = append([]string{parentID}, path...)
		currentStackID = parentID
	}

	return path, nil
}

// DeleteResume soft deletes all resume records for a chat
// Called after successful resume to clean up
func (store *Xun) DeleteResume(chatID string) error {
	if chatID == "" {
		return fmt.Errorf("chat_id is required")
	}

	_, err := store.newQueryResume().
		Where("chat_id", chatID).
		WhereNull("deleted_at").
		Update(map[string]interface{}{
			"deleted_at": time.Now(),
			"updated_at": time.Now(),
		})

	return err
}

// GetResumeByRequestID retrieves resume records for a specific request
func (store *Xun) GetResumeByRequestID(requestID string) ([]*types.Resume, error) {
	if requestID == "" {
		return nil, fmt.Errorf("request_id is required")
	}

	rows, err := store.newQueryResume().
		Where("request_id", requestID).
		WhereNull("deleted_at").
		OrderBy("sequence", "asc").
		Get()
	if err != nil {
		return nil, err
	}

	records := make([]*types.Resume, 0, len(rows))
	for _, row := range rows {
		data := row.ToMap()
		if data == nil || data["resume_id"] == nil {
			continue
		}

		record, err := store.rowToResume(data)
		if err != nil {
			continue
		}
		records = append(records, record)
	}

	return records, nil
}

// =============================================================================
// Helper Functions
// =============================================================================

// rowToResume converts a database row to a Resume struct
func (store *Xun) rowToResume(data map[string]interface{}) (*types.Resume, error) {
	record := &types.Resume{
		ResumeID:      getString(data, "resume_id"),
		ChatID:        getString(data, "chat_id"),
		RequestID:     getString(data, "request_id"),
		AssistantID:   getString(data, "assistant_id"),
		StackID:       getString(data, "stack_id"),
		StackParentID: getString(data, "stack_parent_id"),
		StackDepth:    getInt(data, "stack_depth"),
		Type:          getString(data, "type"),
		Status:        getString(data, "status"),
		Error:         getString(data, "error"),
		Sequence:      getInt(data, "sequence"),
	}

	// Handle timestamps
	if createdAt := getTime(data, "created_at"); createdAt != nil {
		record.CreatedAt = *createdAt
	}
	if updatedAt := getTime(data, "updated_at"); updatedAt != nil {
		record.UpdatedAt = *updatedAt
	}

	// Handle JSON fields
	if input := data["input"]; input != nil {
		if inputStr, ok := input.(string); ok && inputStr != "" {
			var inputMap map[string]interface{}
			if err := jsoniter.UnmarshalFromString(inputStr, &inputMap); err == nil {
				record.Input = inputMap
			}
		} else if inputMap, ok := input.(map[string]interface{}); ok {
			record.Input = inputMap
		}
	}

	if output := data["output"]; output != nil {
		if outputStr, ok := output.(string); ok && outputStr != "" {
			var outputMap map[string]interface{}
			if err := jsoniter.UnmarshalFromString(outputStr, &outputMap); err == nil {
				record.Output = outputMap
			}
		} else if outputMap, ok := output.(map[string]interface{}); ok {
			record.Output = outputMap
		}
	}

	if snapshot := data["space_snapshot"]; snapshot != nil {
		if snapshotStr, ok := snapshot.(string); ok && snapshotStr != "" {
			var snapshotMap map[string]interface{}
			if err := jsoniter.UnmarshalFromString(snapshotStr, &snapshotMap); err == nil {
				record.SpaceSnapshot = snapshotMap
			}
		} else if snapshotMap, ok := snapshot.(map[string]interface{}); ok {
			record.SpaceSnapshot = snapshotMap
		}
	}

	if metadata := data["metadata"]; metadata != nil {
		if metaStr, ok := metadata.(string); ok && metaStr != "" {
			var metaMap map[string]interface{}
			if err := jsoniter.UnmarshalFromString(metaStr, &metaMap); err == nil {
				record.Metadata = metaMap
			}
		} else if metaMap, ok := metadata.(map[string]interface{}); ok {
			record.Metadata = metaMap
		}
	}

	return record, nil
}
