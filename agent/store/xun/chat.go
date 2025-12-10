package xun

import (
	"fmt"
	"math"
	"time"

	"github.com/google/uuid"
	jsoniter "github.com/json-iterator/go"
	"github.com/yaoapp/yao/agent/store/types"
)

// =============================================================================
// Chat Management
// =============================================================================

// CreateChat creates a new chat session
func (store *Xun) CreateChat(chat *types.Chat) error {
	if chat == nil {
		return fmt.Errorf("chat cannot be nil")
	}

	// Validate required fields
	if chat.AssistantID == "" {
		return fmt.Errorf("assistant_id is required")
	}

	// Generate chat_id if not provided
	if chat.ChatID == "" {
		chat.ChatID = uuid.New().String()
	}

	// Check if chat already exists
	exists, err := store.newQueryChat().
		Where("chat_id", chat.ChatID).
		Exists()
	if err != nil {
		return err
	}
	if exists {
		return fmt.Errorf("chat %s already exists", chat.ChatID)
	}

	// Set defaults
	if chat.Status == "" {
		chat.Status = "active"
	}
	if chat.Share == "" {
		chat.Share = "private"
	}

	// Prepare data
	data := map[string]interface{}{
		"chat_id":      chat.ChatID,
		"assistant_id": chat.AssistantID,
		"status":       chat.Status,
		"public":       chat.Public,
		"share":        chat.Share,
		"sort":         chat.Sort,
		"created_at":   time.Now(),
		"updated_at":   time.Now(),
	}

	// Handle last_mode (nullable)
	if chat.LastMode != "" {
		data["last_mode"] = chat.LastMode
	}

	// Handle nullable fields
	if chat.Title != "" {
		data["title"] = chat.Title
	}
	if chat.LastConnector != "" {
		data["last_connector"] = chat.LastConnector
	}
	if chat.LastMode != "" {
		data["last_mode"] = chat.LastMode
	}
	if chat.LastMessageAt != nil {
		data["last_message_at"] = *chat.LastMessageAt
	}
	if chat.Metadata != nil {
		metadataJSON, err := jsoniter.MarshalToString(chat.Metadata)
		if err != nil {
			return fmt.Errorf("failed to marshal metadata: %w", err)
		}
		data["metadata"] = metadataJSON
	}

	// Handle permission fields (Yao framework permission: true)
	if chat.CreatedBy != "" {
		data["__yao_created_by"] = chat.CreatedBy
	}
	if chat.UpdatedBy != "" {
		data["__yao_updated_by"] = chat.UpdatedBy
	}
	if chat.TeamID != "" {
		data["__yao_team_id"] = chat.TeamID
	}
	if chat.TenantID != "" {
		data["__yao_tenant_id"] = chat.TenantID
	}

	// Insert
	return store.newQueryChat().Insert(data)
}

// GetChat retrieves a single chat by ID
func (store *Xun) GetChat(chatID string) (*types.Chat, error) {
	if chatID == "" {
		return nil, fmt.Errorf("chat_id is required")
	}

	row, err := store.newQueryChat().
		Where("chat_id", chatID).
		WhereNull("deleted_at").
		First()
	if err != nil {
		return nil, err
	}

	if row == nil {
		return nil, fmt.Errorf("chat %s not found", chatID)
	}

	data := row.ToMap()
	if len(data) == 0 || data["chat_id"] == nil {
		return nil, fmt.Errorf("chat %s not found", chatID)
	}

	return store.rowToChat(data)
}

// UpdateChat updates chat fields
func (store *Xun) UpdateChat(chatID string, updates map[string]interface{}) error {
	if chatID == "" {
		return fmt.Errorf("chat_id is required")
	}
	if len(updates) == 0 {
		return fmt.Errorf("no fields to update")
	}

	// Check if chat exists
	exists, err := store.newQueryChat().
		Where("chat_id", chatID).
		WhereNull("deleted_at").
		Exists()
	if err != nil {
		return err
	}
	if !exists {
		return fmt.Errorf("chat %s not found", chatID)
	}

	// Prepare update data
	data := make(map[string]interface{})

	// Process each update field
	for key, value := range updates {
		// Skip system fields
		if key == "chat_id" || key == "created_at" {
			continue
		}

		// Handle metadata specially
		if key == "metadata" {
			if value != nil {
				metadataJSON, err := jsoniter.MarshalToString(value)
				if err != nil {
					return fmt.Errorf("failed to marshal metadata: %w", err)
				}
				data["metadata"] = metadataJSON
			} else {
				data["metadata"] = nil
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

	_, err = store.newQueryChat().
		Where("chat_id", chatID).
		Update(data)

	return err
}

// DeleteChat deletes a chat and its associated messages (soft delete)
func (store *Xun) DeleteChat(chatID string) error {
	if chatID == "" {
		return fmt.Errorf("chat_id is required")
	}

	// Check if chat exists
	exists, err := store.newQueryChat().
		Where("chat_id", chatID).
		WhereNull("deleted_at").
		Exists()
	if err != nil {
		return err
	}
	if !exists {
		return fmt.Errorf("chat %s not found", chatID)
	}

	// Soft delete the chat
	_, err = store.newQueryChat().
		Where("chat_id", chatID).
		Update(map[string]interface{}{
			"deleted_at": time.Now(),
			"updated_at": time.Now(),
		})

	return err
}

// ListChats retrieves a paginated list of chats with optional grouping
func (store *Xun) ListChats(filter types.ChatFilter) (*types.ChatList, error) {
	// Set defaults
	if filter.Page <= 0 {
		filter.Page = 1
	}
	if filter.PageSize <= 0 {
		filter.PageSize = 20
	}
	if filter.OrderBy == "" {
		filter.OrderBy = "last_message_at"
	}
	if filter.Order == "" {
		filter.Order = "desc"
	}
	if filter.TimeField == "" {
		filter.TimeField = "last_message_at"
	}

	// Build base query
	qb := store.newQueryChat().WhereNull("deleted_at")

	// Apply permission filters (UserID and TeamID)
	if filter.UserID != "" {
		qb.Where("__yao_created_by", filter.UserID)
	}
	if filter.TeamID != "" {
		qb.Where("__yao_team_id", filter.TeamID)
	}

	// Apply business filters
	if filter.AssistantID != "" {
		qb.Where("assistant_id", filter.AssistantID)
	}
	if filter.Status != "" {
		qb.Where("status", filter.Status)
	}
	if filter.Keywords != "" {
		qb.Where("title", "like", fmt.Sprintf("%%%s%%", filter.Keywords))
	}

	// Apply time range filter
	if filter.StartTime != nil {
		qb.Where(filter.TimeField, ">=", *filter.StartTime)
	}
	if filter.EndTime != nil {
		qb.Where(filter.TimeField, "<=", *filter.EndTime)
	}

	// Apply custom query filter (for advanced permission filtering)
	// This allows flexible combinations like: (created_by = user OR team_id = team)
	if filter.QueryFilter != nil {
		qb.Where(filter.QueryFilter)
	}

	// Get total count
	total, err := qb.Clone().Count()
	if err != nil {
		return nil, err
	}

	// Calculate pagination
	pageCount := int(math.Ceil(float64(total) / float64(filter.PageSize)))
	if pageCount < 1 {
		pageCount = 1
	}
	offset := (filter.Page - 1) * filter.PageSize

	// Get paginated results
	rows, err := qb.OrderBy(filter.OrderBy, filter.Order).
		Offset(offset).
		Limit(filter.PageSize).
		Get()
	if err != nil {
		return nil, err
	}

	// Convert rows to Chat objects
	chats := make([]*types.Chat, 0, len(rows))
	for _, row := range rows {
		data := row.ToMap()
		if data == nil || data["chat_id"] == nil {
			continue
		}

		chat, err := store.rowToChat(data)
		if err != nil {
			continue
		}
		chats = append(chats, chat)
	}

	result := &types.ChatList{
		Data:      chats,
		Page:      filter.Page,
		PageSize:  filter.PageSize,
		PageCount: pageCount,
		Total:     int(total),
	}

	// Apply time-based grouping if requested
	if filter.GroupBy == "time" {
		result.Groups = store.groupChatsByTime(chats)
	}

	return result, nil
}

// =============================================================================
// Helper Functions
// =============================================================================

// rowToChat converts a database row to a Chat struct
func (store *Xun) rowToChat(data map[string]interface{}) (*types.Chat, error) {
	chat := &types.Chat{
		ChatID:        getString(data, "chat_id"),
		Title:         getString(data, "title"),
		AssistantID:   getString(data, "assistant_id"),
		LastConnector: getString(data, "last_connector"),
		LastMode:      getString(data, "last_mode"),
		Status:        getString(data, "status"),
		Public:        getBool(data, "public"),
		Share:         getString(data, "share"),
		Sort:          getInt(data, "sort"),
	}

	// Handle timestamps
	if createdAt := getTime(data, "created_at"); createdAt != nil {
		chat.CreatedAt = *createdAt
	}
	if updatedAt := getTime(data, "updated_at"); updatedAt != nil {
		chat.UpdatedAt = *updatedAt
	}
	if lastMsgAt := getTime(data, "last_message_at"); lastMsgAt != nil {
		chat.LastMessageAt = lastMsgAt
	}

	// Handle metadata
	if metadata := data["metadata"]; metadata != nil {
		if metaStr, ok := metadata.(string); ok && metaStr != "" {
			var meta map[string]interface{}
			if err := jsoniter.UnmarshalFromString(metaStr, &meta); err == nil {
				chat.Metadata = meta
			}
		} else if metaMap, ok := metadata.(map[string]interface{}); ok {
			chat.Metadata = metaMap
		}
	}

	// Handle permission fields
	chat.CreatedBy = getString(data, "__yao_created_by")
	chat.UpdatedBy = getString(data, "__yao_updated_by")
	chat.TeamID = getString(data, "__yao_team_id")
	chat.TenantID = getString(data, "__yao_tenant_id")

	return chat, nil
}

// groupChatsByTime groups chats by time periods
func (store *Xun) groupChatsByTime(chats []*types.Chat) []*types.ChatGroup {
	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	yesterday := today.AddDate(0, 0, -1)
	thisWeekStart := today.AddDate(0, 0, -int(today.Weekday()))
	thisMonthStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())

	groups := map[string]*types.ChatGroup{
		"today":      {Key: "today", Label: "Today", Chats: []*types.Chat{}},
		"yesterday":  {Key: "yesterday", Label: "Yesterday", Chats: []*types.Chat{}},
		"this_week":  {Key: "this_week", Label: "This Week", Chats: []*types.Chat{}},
		"this_month": {Key: "this_month", Label: "This Month", Chats: []*types.Chat{}},
		"earlier":    {Key: "earlier", Label: "Earlier", Chats: []*types.Chat{}},
	}

	for _, chat := range chats {
		// Use last_message_at if available, otherwise created_at
		var chatTime time.Time
		if chat.LastMessageAt != nil {
			chatTime = *chat.LastMessageAt
		} else {
			chatTime = chat.CreatedAt
		}

		chatDate := time.Date(chatTime.Year(), chatTime.Month(), chatTime.Day(), 0, 0, 0, 0, chatTime.Location())

		switch {
		case chatDate.Equal(today) || chatDate.After(today):
			groups["today"].Chats = append(groups["today"].Chats, chat)
		case chatDate.Equal(yesterday):
			groups["yesterday"].Chats = append(groups["yesterday"].Chats, chat)
		case chatDate.After(thisWeekStart) || chatDate.Equal(thisWeekStart):
			groups["this_week"].Chats = append(groups["this_week"].Chats, chat)
		case chatDate.After(thisMonthStart) || chatDate.Equal(thisMonthStart):
			groups["this_month"].Chats = append(groups["this_month"].Chats, chat)
		default:
			groups["earlier"].Chats = append(groups["earlier"].Chats, chat)
		}
	}

	// Update counts and filter empty groups
	result := make([]*types.ChatGroup, 0)
	for _, key := range []string{"today", "yesterday", "this_week", "this_month", "earlier"} {
		group := groups[key]
		group.Count = len(group.Chats)
		if group.Count > 0 {
			result = append(result, group)
		}
	}

	return result
}

// getTime helper function to convert database value to time.Time pointer
func getTime(data map[string]interface{}, key string) *time.Time {
	if v := data[key]; v != nil {
		switch t := v.(type) {
		case time.Time:
			return &t
		case *time.Time:
			return t
		case string:
			// Try parsing various formats
			formats := []string{
				time.RFC3339,
				"2006-01-02 15:04:05",
				"2006-01-02 15:04:05.999999-07:00",
				"2006-01-02T15:04:05Z",
			}
			for _, format := range formats {
				if parsed, err := time.Parse(format, t); err == nil {
					return &parsed
				}
			}
		}
	}
	return nil
}

// UpdateChatLastMessageAt updates the last_message_at timestamp for a chat
func (store *Xun) UpdateChatLastMessageAt(chatID string, timestamp time.Time) error {
	if chatID == "" {
		return fmt.Errorf("chat_id is required")
	}

	_, err := store.newQueryChat().
		Where("chat_id", chatID).
		Update(map[string]interface{}{
			"last_message_at": timestamp,
			"updated_at":      time.Now(),
		})

	return err
}
