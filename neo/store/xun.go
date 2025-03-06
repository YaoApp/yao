package store

import (
	"fmt"
	"math"
	"time"

	"github.com/google/uuid"
	jsoniter "github.com/json-iterator/go"
	"github.com/yaoapp/gou/connector"
	"github.com/yaoapp/gou/session"
	"github.com/yaoapp/kun/log"
	"github.com/yaoapp/xun/capsule"
	"github.com/yaoapp/xun/dbal/query"
	"github.com/yaoapp/xun/dbal/schema"
)

// Package conversation provides functionality for managing chat conversations and assistants.

// Xun implements the Conversation interface using a database backend.
// It provides functionality for:
// - Managing chat conversations and their message histories
// - Organizing chats with pagination and date-based grouping
// - Handling chat metadata like titles and creation dates
// - Managing AI assistants with their configurations and metadata
// - Supporting data expiration through TTL settings
type Xun struct {
	query   query.Query
	schema  schema.Schema
	setting Setting
}

// Public interface methods:
//
// NewXun creates a new conversation instance with the given settings
// UpdateChatTitle updates the title of a specific chat
// GetChats retrieves a paginated list of chats grouped by date
// GetChat retrieves a specific chat and its message history
// GetHistory retrieves the message history for a specific chat
// SaveHistory saves new messages to a chat's history
// DeleteChat deletes a specific chat and its history
// DeleteAllChats deletes all chats and their histories for a user
// SaveAssistant creates or updates an assistant
// DeleteAssistant deletes an assistant by assistant_id
// GetAssistants retrieves a paginated list of assistants with filtering
// GetAssistant retrieves a single assistant by assistant_id

// NewXun create a new xun store
func NewXun(setting Setting) (Store, error) {
	conv := &Xun{setting: setting}
	if setting.Connector == "default" {
		conv.query = capsule.Global.Query()
		conv.schema = capsule.Global.Schema()
	} else {
		conn, err := connector.Select(setting.Connector)
		if err != nil {
			return nil, err
		}

		conv.query, err = conn.Query()
		if err != nil {
			return nil, err
		}

		conv.schema, err = conn.Schema()
		if err != nil {
			return nil, err
		}
	}

	err := conv.initialize()
	if err != nil {
		return nil, err
	}

	return conv, nil
}

// Rename the following functions to start with lowercase letters to make them private:

func (conv *Xun) newQuery() query.Query {
	qb := conv.query.New()
	qb.Table(conv.getHistoryTable())
	return qb
}

func (conv *Xun) newQueryChat() query.Query {
	qb := conv.query.New()
	qb.Table(conv.getChatTable())
	return qb
}

func (conv *Xun) clean() {
	nums, err := conv.newQuery().Where("expired_at", "<=", time.Now()).Delete()
	if err != nil {
		log.Error("Clean the conversation table error: %s", err.Error())
		return
	}

	if nums > 0 {
		log.Trace("Clean the conversation table: %s %d", conv.setting.Prefix, nums)
	}
}

// Rename Init to initialize to avoid conflicts
func (conv *Xun) initialize() error {
	// Initialize history table
	if err := conv.initHistoryTable(); err != nil {
		return err
	}

	// Initialize chat table
	if err := conv.initChatTable(); err != nil {
		return err
	}

	// Initialize assistant table
	if err := conv.initAssistantTable(); err != nil {
		return err
	}

	return nil
}

func (conv *Xun) initHistoryTable() error {
	historyTable := conv.getHistoryTable()
	has, err := conv.schema.HasTable(historyTable)
	if err != nil {
		return err
	}

	// Create the history table
	if !has {
		err = conv.schema.CreateTable(historyTable, func(table schema.Blueprint) {
			table.ID("id")
			table.String("sid", 255).Index()
			table.String("cid", 200).Null().Index()
			table.String("uid", 255).Null().Index()
			table.String("role", 200).Null().Index()
			table.String("name", 200).Null().Index()
			table.Text("content").Null()
			table.JSON("context").Null()
			table.String("assistant_id", 200).Null().Index()
			table.String("assistant_name", 200).Null()
			table.String("assistant_avatar", 200).Null()
			table.JSON("mentions").Null()
			table.Boolean("silent").SetDefault(false).Index()
			table.TimestampTz("created_at").SetDefaultRaw("NOW()").Index()
			table.TimestampTz("updated_at").Null().Index()
			table.TimestampTz("expired_at").Null().Index()
		})

		if err != nil {
			return err
		}
		log.Trace("Create the conversation history table: %s", historyTable)
	}

	// Validate the table
	tab, err := conv.schema.GetTable(historyTable)
	if err != nil {
		return err
	}

	fields := []string{"id", "sid", "cid", "uid", "role", "name", "content", "context", "assistant_id", "assistant_name", "assistant_avatar", "mentions", "silent", "created_at", "updated_at", "expired_at"}
	for _, field := range fields {
		if !tab.HasColumn(field) {
			return fmt.Errorf("%s is required", field)
		}
	}

	return nil
}

func (conv *Xun) initChatTable() error {
	chatTable := conv.getChatTable()
	has, err := conv.schema.HasTable(chatTable)
	if err != nil {
		return err
	}

	// Create the chat table
	if !has {
		err = conv.schema.CreateTable(chatTable, func(table schema.Blueprint) {
			table.ID("id")
			table.String("chat_id", 200).Unique().Index()
			table.String("title", 200).Null()
			table.String("assistant_id", 200).Null().Index()
			table.String("sid", 255).Index()
			table.Boolean("silent").SetDefault(false).Index()
			table.TimestampTz("created_at").SetDefaultRaw("NOW()").Index()
			table.TimestampTz("updated_at").Null().Index()
		})

		if err != nil {
			return err
		}
		log.Trace("Create the chat table: %s", chatTable)
	}

	// Validate the table
	tab, err := conv.schema.GetTable(chatTable)
	if err != nil {
		return err
	}

	fields := []string{"id", "chat_id", "title", "assistant_id", "sid", "silent", "created_at", "updated_at"}
	for _, field := range fields {
		if !tab.HasColumn(field) {
			return fmt.Errorf("%s is required", field)
		}
	}

	return nil
}

func (conv *Xun) initAssistantTable() error {
	assistantTable := conv.getAssistantTable()
	has, err := conv.schema.HasTable(assistantTable)
	if err != nil {
		return err
	}

	// Create the assistant table
	if !has {
		err = conv.schema.CreateTable(assistantTable, func(table schema.Blueprint) {
			table.ID("id")
			table.String("assistant_id", 200).Unique().Index()
			table.String("type", 200).SetDefault("assistant").Index() // default is assistant
			table.String("name", 200).Null()                          // assistant name
			table.String("avatar", 200).Null()                        // assistant avatar
			table.String("connector", 200).NotNull()                  // assistant connector
			table.Text("description").Null()                          // assistant description
			table.String("path", 200).Null()                          // assistant storage path
			table.Integer("sort").SetDefault(9999).Index()            // assistant sort order
			table.Boolean("built_in").SetDefault(false).Index()       // whether this is a built-in assistant
			table.JSON("placeholder").Null()                          // assistant placeholder
			table.JSON("options").Null()                              // assistant options
			table.JSON("prompts").Null()                              // assistant prompts
			table.JSON("flows").Null()                                // assistant flows
			table.JSON("files").Null()                                // assistant files
			table.JSON("tools").Null()                                // assistant tools
			table.JSON("tags").Null()                                 // assistant tags
			table.Boolean("readonly").SetDefault(false).Index()       // assistant readonly
			table.JSON("permissions").Null()                          // assistant permissions
			table.Boolean("automated").SetDefault(true).Index()       // assistant autoable
			table.Boolean("mentionable").SetDefault(true).Index()     // Whether this assistant can appear in @ mention list
			table.TimestampTz("created_at").SetDefaultRaw("NOW()").Index()
			table.TimestampTz("updated_at").Null().Index()
		})

		if err != nil {
			return err
		}
		log.Trace("Create the assistant table: %s", assistantTable)
	}

	// Validate the table
	tab, err := conv.schema.GetTable(assistantTable)
	if err != nil {
		return err
	}

	fields := []string{"id", "assistant_id", "type", "name", "avatar", "connector", "description", "path", "sort", "built_in", "placeholder", "options", "prompts", "flows", "files", "tools", "tags", "mentionable", "created_at", "updated_at"}
	for _, field := range fields {
		if !tab.HasColumn(field) {
			return fmt.Errorf("%s is required", field)
		}
	}

	return nil
}

func (conv *Xun) getUserID(sid string) (string, error) {
	field := "user_id"
	if conv.setting.UserField != "" {
		field = conv.setting.UserField
	}

	id, err := session.Global().ID(sid).Get(field)
	if err != nil {
		return "", err
	}

	if id == nil || id == "" {
		return sid, nil
	}

	return fmt.Sprintf("%v", id), nil
}

func (conv *Xun) getHistoryTable() string {
	return conv.setting.Prefix + "history"
}

func (conv *Xun) getChatTable() string {
	return conv.setting.Prefix + "chat"
}

func (conv *Xun) getAssistantTable() string {
	return conv.setting.Prefix + "assistant"
}

// UpdateChatTitle update the chat title
func (conv *Xun) UpdateChatTitle(sid string, cid string, title string) error {
	userID, err := conv.getUserID(sid)
	if err != nil {
		return err
	}

	_, err = conv.newQueryChat().
		Where("sid", userID).
		Where("chat_id", cid).
		Update(map[string]interface{}{
			"title":      title,
			"updated_at": time.Now(),
		})
	return err
}

// GetChats get the chat list with grouping by date
func (conv *Xun) GetChats(sid string, filter ChatFilter) (*ChatGroupResponse, error) {
	// Default behavior: exclude silent chats
	if filter.Silent == nil {
		silentFalse := false
		filter.Silent = &silentFalse
	}

	return conv.getChatsWithFilter(sid, filter)
}

// getChatsWithFilter get the chats with filter options
func (conv *Xun) getChatsWithFilter(sid string, filter ChatFilter) (*ChatGroupResponse, error) {
	userID, err := conv.getUserID(sid)
	if err != nil {
		return nil, err
	}

	// Set default values
	if filter.Page <= 0 {
		filter.Page = 1
	}
	if filter.PageSize <= 0 {
		filter.PageSize = 20
	}
	if filter.Order == "" {
		filter.Order = "desc"
	}

	// Get total count
	qbCount := conv.newQueryChat().
		Where("sid", userID)

	// Apply silent filter if provided
	if filter.Silent != nil {
		if *filter.Silent {
			// Include all chats (both silent and non-silent)
		} else {
			// Only include non-silent chats
			qbCount.Where("silent", false)
		}
	}

	// Apply keyword filter if provided
	if filter.Keywords != "" {
		qbCount.Where("title", "like", fmt.Sprintf("%%%s%%", filter.Keywords))
	}

	total, err := qbCount.Count()
	if err != nil {
		return nil, err
	}

	// Calculate last page
	lastPage := int(math.Ceil(float64(total) / float64(filter.PageSize)))
	if lastPage < 1 {
		lastPage = 1
	}

	// Get chats with pagination
	qb := conv.newQueryChat().
		Select("chat_id", "title", "assistant_id", "silent", "created_at", "updated_at").
		Where("sid", userID)

	// Apply silent filter if provided
	if filter.Silent != nil {
		if *filter.Silent {
			// Include all chats (both silent and non-silent)
		} else {
			// Only include non-silent chats
			qb.Where("silent", false)
		}
	}

	// Apply keyword filter if provided
	if filter.Keywords != "" {
		qb.Where("title", "like", fmt.Sprintf("%%%s%%", filter.Keywords))
	}

	// Apply pagination
	offset := (filter.Page - 1) * filter.PageSize
	qb.OrderBy("updated_at", filter.Order).
		Offset(offset).
		Limit(filter.PageSize)

	rows, err := qb.Get()
	if err != nil {
		return nil, err
	}

	// Group chats by date
	today := time.Now().Truncate(24 * time.Hour)
	yesterday := today.AddDate(0, 0, -1)
	thisWeekStart := today.AddDate(0, 0, -int(today.Weekday()))
	lastWeekStart := thisWeekStart.AddDate(0, 0, -7)
	lastWeekEnd := thisWeekStart.AddDate(0, 0, -1)

	groups := map[string][]map[string]interface{}{
		"Today":        {},
		"Yesterday":    {},
		"This Week":    {},
		"Last Week":    {},
		"Even Earlier": {},
	}

	// Collect assistant IDs to fetch their details
	assistantIDs := []interface{}{}
	for _, row := range rows {
		if assistantID := row.Get("assistant_id"); assistantID != nil && assistantID != "" {
			assistantIDs = append(assistantIDs, assistantID)
		}
	}

	// Fetch assistant details
	assistantMap := map[string]map[string]interface{}{}
	if len(assistantIDs) > 0 {
		assistants, err := conv.query.New().
			Table(conv.getAssistantTable()).
			Select("assistant_id", "name", "avatar").
			WhereIn("assistant_id", assistantIDs).
			Get()
		if err != nil {
			return nil, err
		}

		for _, assistant := range assistants {
			if id := assistant.Get("assistant_id"); id != nil {
				assistantMap[fmt.Sprintf("%v", id)] = map[string]interface{}{
					"name":   assistant.Get("name"),
					"avatar": assistant.Get("avatar"),
				}
			}
		}
	}

	for _, row := range rows {
		chatID := row.Get("chat_id")
		if chatID == nil || chatID == "" {
			continue
		}

		chat := map[string]interface{}{
			"chat_id":      chatID,
			"title":        row.Get("title"),
			"assistant_id": row.Get("assistant_id"),
			"silent":       row.Get("silent"),
		}

		// Add assistant details if available
		if assistantID := row.Get("assistant_id"); assistantID != nil && assistantID != "" {
			if assistant, ok := assistantMap[fmt.Sprintf("%v", assistantID)]; ok {
				chat["assistant_name"] = assistant["name"]
				chat["assistant_avatar"] = assistant["avatar"]
			}
		}

		var dbDatetime = row.Get("updated_at")
		if dbDatetime == nil {
			dbDatetime = row.Get("created_at")
		}

		var createdAt time.Time
		switch v := dbDatetime.(type) {
		case time.Time:
			createdAt = v
		case string:
			parsed, err := time.Parse("2006-01-02 15:04:05.999999-07:00", v)
			if err != nil {
				// Try alternative format
				parsed, err = time.Parse(time.RFC3339, v)
				if err != nil {
					continue
				}
			}
			createdAt = parsed
		default:
			continue
		}

		createdDate := createdAt.Truncate(24 * time.Hour)

		switch {
		case createdDate.Equal(today):
			groups["Today"] = append(groups["Today"], chat)
		case createdDate.Equal(yesterday):
			groups["Yesterday"] = append(groups["Yesterday"], chat)
		case createdDate.After(thisWeekStart) && createdDate.Before(today):
			groups["This Week"] = append(groups["This Week"], chat)
		case createdDate.After(lastWeekStart) && createdDate.Before(lastWeekEnd.AddDate(0, 0, 1)):
			groups["Last Week"] = append(groups["Last Week"], chat)
		default:
			groups["Even Earlier"] = append(groups["Even Earlier"], chat)
		}
	}

	// Convert to ordered slice
	result := []ChatGroup{}
	for _, label := range []string{"Today", "Yesterday", "This Week", "Last Week", "Even Earlier"} {
		if len(groups[label]) > 0 {
			result = append(result, ChatGroup{
				Label: label,
				Chats: groups[label],
			})
		}
	}

	return &ChatGroupResponse{
		Groups:   result,
		Page:     filter.Page,
		PageSize: filter.PageSize,
		Total:    total,
		LastPage: lastPage,
	}, nil
}

// GetHistory get the history
func (conv *Xun) GetHistory(sid string, cid string) ([]map[string]interface{}, error) {
	userID, err := conv.getUserID(sid)
	if err != nil {
		return nil, err
	}

	qb := conv.newQuery().
		Select("role", "name", "content", "context", "assistant_id", "assistant_name", "assistant_avatar", "mentions", "uid", "silent", "created_at", "updated_at").
		Where("sid", userID).
		Where("cid", cid).
		OrderBy("id", "desc")

	// By default, exclude silent messages
	qb.Where("silent", false)

	if conv.setting.TTL > 0 {
		qb.Where("expired_at", ">", time.Now())
	}

	limit := 20
	if conv.setting.MaxSize > 0 {
		limit = conv.setting.MaxSize
	}

	rows, err := qb.Limit(limit).Get()
	if err != nil {
		return nil, err
	}

	res := []map[string]interface{}{}
	for _, row := range rows {
		message := map[string]interface{}{
			"role":             row.Get("role"),
			"name":             row.Get("name"),
			"content":          row.Get("content"),
			"context":          row.Get("context"),
			"assistant_id":     row.Get("assistant_id"),
			"assistant_name":   row.Get("assistant_name"),
			"assistant_avatar": row.Get("assistant_avatar"),
			"mentions":         row.Get("mentions"),
			"uid":              row.Get("uid"),
			"silent":           row.Get("silent"),
			"created_at":       row.Get("created_at"),
			"updated_at":       row.Get("updated_at"),
		}
		res = append([]map[string]interface{}{message}, res...)
	}

	return res, nil
}

// SaveHistory save the history
func (conv *Xun) SaveHistory(sid string, messages []map[string]interface{}, cid string, context map[string]interface{}) error {

	if cid == "" {
		cid = uuid.New().String() // Generate a new UUID if cid is empty
	}

	userID, err := conv.getUserID(sid)
	if err != nil {
		return err
	}

	// Get assistant_id from context
	var assistantID interface{} = nil
	if context != nil {
		if id, ok := context["assistant_id"].(string); ok && id != "" {
			assistantID = id
		}
	}

	// Get silent flag from context
	var silent bool = false
	if context != nil {
		if silentVal, ok := context["silent"]; ok {
			switch v := silentVal.(type) {
			case bool:
				silent = v
			case string:
				silent = v == "true" || v == "1" || v == "yes"
			case int:
				silent = v != 0
			case float64:
				silent = v != 0
			}
		}
	}

	// First ensure chat record exists
	exists, err := conv.newQueryChat().
		Where("chat_id", cid).
		Where("sid", userID).
		Exists()

	if err != nil {
		return err
	}

	if !exists {
		// Create new chat record
		err = conv.newQueryChat().
			Insert(map[string]interface{}{
				"chat_id":      cid,
				"sid":          userID,
				"assistant_id": assistantID,
				"silent":       silent,
				"created_at":   time.Now(),
			})

		if err != nil {
			return err
		}
	} else {
		// Update assistant_id and silent if needed
		_, err = conv.newQueryChat().
			Where("chat_id", cid).
			Where("sid", userID).
			Update(map[string]interface{}{
				"assistant_id": assistantID,
				"silent":       silent,
			})
		if err != nil {
			return err
		}
	}

	// Save message history
	defer conv.clean()
	var expiredAt interface{} = nil
	values := []map[string]interface{}{}
	if conv.setting.TTL > 0 {
		expiredAt = time.Now().Add(time.Duration(conv.setting.TTL) * time.Second)
	}

	now := time.Now()
	for _, message := range messages {
		// Type assertion safety checks
		role, ok := message["role"].(string)
		if !ok {
			return fmt.Errorf("invalid role type in message: %v", message["role"])
		}

		content, ok := message["content"].(string)
		if !ok {
			return fmt.Errorf("invalid content type in message: %v", message["content"])
		}

		var contextRaw interface{} = nil
		if context != nil {
			contextRaw, err = jsoniter.MarshalToString(context)
			if err != nil {
				return err
			}
		}

		// Process mentions if present
		var mentionsRaw interface{} = nil
		if mentions, ok := message["mentions"].([]interface{}); ok && len(mentions) > 0 {
			mentionsRaw, err = jsoniter.MarshalToString(mentions)
			if err != nil {
				return err
			}
		}

		value := map[string]interface{}{
			"role":             role,
			"name":             "",
			"content":          content,
			"sid":              userID,
			"cid":              cid,
			"uid":              userID,
			"context":          contextRaw,
			"mentions":         mentionsRaw,
			"assistant_id":     nil,
			"assistant_name":   nil,
			"assistant_avatar": nil,
			"silent":           silent,
			"created_at":       now,
			"updated_at":       nil,
			"expired_at":       expiredAt,
		}

		if name, ok := message["name"].(string); ok {
			value["name"] = name
		}

		// Add assistant fields if present
		if assistantID, ok := message["assistant_id"].(string); ok {
			value["assistant_id"] = assistantID
		}
		if assistantName, ok := message["assistant_name"].(string); ok {
			value["assistant_name"] = assistantName
		}
		if assistantAvatar, ok := message["assistant_avatar"].(string); ok {
			value["assistant_avatar"] = assistantAvatar
		}

		values = append(values, value)
	}

	err = conv.newQuery().Insert(values)
	if err != nil {
		return err
	}

	// Update Chat updated_at
	_, err = conv.newQueryChat().
		Where("chat_id", cid).
		Where("sid", userID).
		Update(map[string]interface{}{"updated_at": now})
	if err != nil {
		return err
	}

	return nil
}

// GetChat get the chat info and its history
func (conv *Xun) GetChat(sid string, cid string) (*ChatInfo, error) {
	userID, err := conv.getUserID(sid)
	if err != nil {
		return nil, err
	}

	// Get chat info
	qb := conv.newQueryChat().
		Select("chat_id", "title", "assistant_id").
		Where("sid", userID).
		Where("chat_id", cid)

	row, err := qb.First()
	if err != nil {
		return nil, err
	}

	// Return nil if chat_id is nil (means no chat found)
	if row.Get("chat_id") == nil {
		return nil, nil
	}

	chat := map[string]interface{}{
		"chat_id":      row.Get("chat_id"),
		"title":        row.Get("title"),
		"assistant_id": row.Get("assistant_id"),
	}

	// Get assistant details if assistant_id exists
	if assistantID := row.Get("assistant_id"); assistantID != nil && assistantID != "" {
		assistant, err := conv.query.New().
			Table(conv.getAssistantTable()).
			Select("name", "avatar").
			Where("assistant_id", assistantID).
			First()
		if err != nil {
			return nil, err
		}

		if assistant != nil {
			chat["assistant_name"] = assistant.Get("name")
			chat["assistant_avatar"] = assistant.Get("avatar")
		}
	}

	// Get chat history with default filter (silent=false)
	history, err := conv.GetHistory(sid, cid)
	if err != nil {
		return nil, err
	}

	return &ChatInfo{
		Chat:    chat,
		History: history,
	}, nil
}

// GetChatWithFilter get the chat info and its history with filter options
func (conv *Xun) GetChatWithFilter(sid string, cid string, filter ChatFilter) (*ChatInfo, error) {
	userID, err := conv.getUserID(sid)
	if err != nil {
		return nil, err
	}

	// Get chat info
	qb := conv.newQueryChat().
		Select("chat_id", "title", "assistant_id").
		Where("sid", userID).
		Where("chat_id", cid)

	row, err := qb.First()
	if err != nil {
		return nil, err
	}

	// Return nil if chat_id is nil (means no chat found)
	if row.Get("chat_id") == nil {
		return nil, nil
	}

	chat := map[string]interface{}{
		"chat_id":      row.Get("chat_id"),
		"title":        row.Get("title"),
		"assistant_id": row.Get("assistant_id"),
	}

	// Get assistant details if assistant_id exists
	if assistantID := row.Get("assistant_id"); assistantID != nil && assistantID != "" {
		assistant, err := conv.query.New().
			Table(conv.getAssistantTable()).
			Select("name", "avatar").
			Where("assistant_id", assistantID).
			First()
		if err != nil {
			return nil, err
		}

		if assistant != nil {
			chat["assistant_name"] = assistant.Get("name")
			chat["assistant_avatar"] = assistant.Get("avatar")
		}
	}

	// Get chat history with filter
	history, err := conv.GetHistoryWithFilter(sid, cid, filter)
	if err != nil {
		return nil, err
	}

	return &ChatInfo{
		Chat:    chat,
		History: history,
	}, nil
}

// DeleteChat deletes a specific chat and its history
func (conv *Xun) DeleteChat(sid string, cid string) error {
	userID, err := conv.getUserID(sid)
	if err != nil {
		return err
	}

	// Delete history records first
	_, err = conv.newQuery().
		Where("sid", userID).
		Where("cid", cid).
		Delete()
	if err != nil {
		return err
	}

	// Then delete the chat
	_, err = conv.newQueryChat().
		Where("sid", userID).
		Where("chat_id", cid).
		Limit(1).
		Delete()
	return err
}

// DeleteAllChats deletes all chats and their histories for a user
func (conv *Xun) DeleteAllChats(sid string) error {
	userID, err := conv.getUserID(sid)
	if err != nil {
		return err
	}

	// Delete history records first
	_, err = conv.newQuery().
		Where("sid", userID).
		Delete()
	if err != nil {
		return err
	}

	// Then delete all chats
	_, err = conv.newQueryChat().
		Where("sid", userID).
		Delete()
	return err
}

// parseJSONFields parses JSON string fields into their corresponding Go types
func (conv *Xun) parseJSONFields(data map[string]interface{}, fields []string) {
	for _, field := range fields {
		if val := data[field]; val != nil {
			if strVal, ok := val.(string); ok && strVal != "" {
				var parsed interface{}
				if err := jsoniter.UnmarshalFromString(strVal, &parsed); err == nil {
					data[field] = parsed
				}
			}
		}
	}
}

// SaveAssistant saves assistant information
func (conv *Xun) SaveAssistant(assistant map[string]interface{}) (interface{}, error) {
	// Validate required fields
	requiredFields := []string{"name", "type", "connector"}
	for _, field := range requiredFields {
		if _, ok := assistant[field]; !ok {
			return nil, fmt.Errorf("field %s is required", field)
		}
		if assistant[field] == nil || assistant[field] == "" {
			return nil, fmt.Errorf("field %s cannot be empty", field)
		}
	}

	// Create a copy of the assistant map to avoid modifying the original
	assistantCopy := make(map[string]interface{})
	for k, v := range assistant {
		assistantCopy[k] = v
	}

	// Process JSON fields
	jsonFields := []string{"tags", "options", "prompts", "flows", "files", "tools", "permissions", "placeholder"}
	for _, field := range jsonFields {
		if val, ok := assistantCopy[field]; ok && val != nil {
			// If it's a string, try to parse it first
			if strVal, ok := val.(string); ok && strVal != "" {
				var parsed interface{}
				if err := jsoniter.UnmarshalFromString(strVal, &parsed); err == nil {
					assistantCopy[field] = parsed
				}
			}
		}
	}

	// Generate assistant_id if not provided
	if _, ok := assistantCopy["assistant_id"]; !ok {
		assistantCopy["assistant_id"] = uuid.New().String()
	}

	// Check if assistant exists
	exists, err := conv.query.New().
		Table(conv.getAssistantTable()).
		Where("assistant_id", assistantCopy["assistant_id"]).
		Exists()
	if err != nil {
		return nil, err
	}

	// Convert JSON fields to strings for storage
	for _, field := range jsonFields {
		if val, ok := assistantCopy[field]; ok && val != nil {
			jsonStr, err := jsoniter.MarshalToString(val)
			if err != nil {
				return nil, fmt.Errorf("failed to marshal %s to JSON: %v", field, err)
			}
			assistantCopy[field] = jsonStr
		}
	}

	// Update or insert
	if exists {
		_, err := conv.query.New().
			Table(conv.getAssistantTable()).
			Where("assistant_id", assistantCopy["assistant_id"]).
			Update(assistantCopy)
		if err != nil {
			return nil, err
		}
		return assistantCopy["assistant_id"], nil
	}

	err = conv.query.New().
		Table(conv.getAssistantTable()).
		Insert(assistantCopy)
	if err != nil {
		return nil, err
	}
	return assistantCopy["assistant_id"], nil
}

// DeleteAssistant deletes an assistant by assistant_id
func (conv *Xun) DeleteAssistant(assistantID string) error {
	// Check if assistant exists
	exists, err := conv.query.New().
		Table(conv.getAssistantTable()).
		Where("assistant_id", assistantID).
		Exists()
	if err != nil {
		return err
	}

	if !exists {
		return fmt.Errorf("assistant %s not found", assistantID)
	}

	_, err = conv.query.New().
		Table(conv.getAssistantTable()).
		Where("assistant_id", assistantID).
		Delete()
	return err
}

// GetAssistants retrieves assistants with pagination and filtering
func (conv *Xun) GetAssistants(filter AssistantFilter) (*AssistantResponse, error) {
	qb := conv.query.New().
		Table(conv.getAssistantTable())

	// Apply tag filter if provided
	if filter.Tags != nil && len(filter.Tags) > 0 {
		qb.Where(func(qb query.Query) {
			for i, tag := range filter.Tags {
				// For each tag, we need to match it as part of a JSON array
				// This will match both single tag arrays ["tag1"] and multi-tag arrays ["tag1","tag2"]
				pattern := fmt.Sprintf("%%\"%s\"%%", tag)
				if i == 0 {
					qb.Where("tags", "like", pattern)
				} else {
					qb.OrWhere("tags", "like", pattern)
				}
			}
		})
	}

	// Apply keyword filter if provided
	if filter.Keywords != "" {
		qb.Where(func(qb query.Query) {
			qb.Where("name", "like", fmt.Sprintf("%%%s%%", filter.Keywords)).
				OrWhere("description", "like", fmt.Sprintf("%%%s%%", filter.Keywords))
		})
	}

	// Apply connector filter if provided
	if filter.Connector != "" {
		qb.Where("connector", filter.Connector)
	}

	// Apply assistant_id filter if provided
	if filter.AssistantID != "" {
		qb.Where("assistant_id", filter.AssistantID)
	}

	// Apply assistantIDs filter if provided
	if filter.AssistantIDs != nil && len(filter.AssistantIDs) > 0 {
		qb.WhereIn("assistant_id", filter.AssistantIDs)
	}

	// Apply mentionable filter if provided
	if filter.Mentionable != nil {
		qb.Where("mentionable", *filter.Mentionable)
	}

	// Apply automated filter if provided
	if filter.Automated != nil {
		qb.Where("automated", *filter.Automated)
	}

	// Apply built_in filter if provided
	if filter.BuiltIn != nil {
		qb.Where("built_in", *filter.BuiltIn)
	}

	// Set defaults for pagination
	if filter.PageSize <= 0 {
		filter.PageSize = 20
	}
	if filter.Page <= 0 {
		filter.Page = 1
	}

	// Get total count
	total, err := qb.Clone().Count()
	if err != nil {
		return nil, err
	}

	// Calculate pagination
	offset := (filter.Page - 1) * filter.PageSize
	totalPages := int(math.Ceil(float64(total) / float64(filter.PageSize)))
	nextPage := filter.Page + 1
	if nextPage > totalPages {
		nextPage = 0
	}
	prevPage := filter.Page - 1
	if prevPage < 1 {
		prevPage = 0
	}

	// Apply select fields if provided
	if filter.Select != nil && len(filter.Select) > 0 {
		selectFields := make([]interface{}, len(filter.Select))
		for i, field := range filter.Select {
			selectFields[i] = field
		}
		qb.Select(selectFields...)
	}

	// Get paginated results
	rows, err := qb.OrderBy("sort", "asc").
		OrderBy("updated_at", "desc").
		Offset(offset).
		Limit(filter.PageSize).
		Get()
	if err != nil {
		return nil, err
	}

	// Convert rows to map slice and parse JSON fields
	data := make([]map[string]interface{}, len(rows))
	jsonFields := []string{"tags", "options", "prompts", "flows", "files", "tools", "permissions", "placeholder"}
	for i, row := range rows {
		data[i] = row
		// Only parse JSON fields if they are selected or no select filter is provided
		if filter.Select == nil || len(filter.Select) == 0 {
			conv.parseJSONFields(data[i], jsonFields)
		} else {
			// Parse only selected JSON fields
			selectedJSONFields := []string{}
			for _, field := range jsonFields {
				for _, selected := range filter.Select {
					if selected == field {
						selectedJSONFields = append(selectedJSONFields, field)
						break
					}
				}
			}
			if len(selectedJSONFields) > 0 {
				conv.parseJSONFields(data[i], selectedJSONFields)
			}
		}
	}

	return &AssistantResponse{
		Data:     data,
		Page:     filter.Page,
		PageSize: filter.PageSize,
		PageCnt:  totalPages,
		Next:     nextPage,
		Prev:     prevPage,
		Total:    total,
	}, nil
}

// GetAssistant retrieves a single assistant by ID
func (conv *Xun) GetAssistant(assistantID string) (map[string]interface{}, error) {
	row, err := conv.query.New().
		Table(conv.getAssistantTable()).
		Where("assistant_id", assistantID).
		First()
	if err != nil {
		return nil, err
	}

	if row == nil {
		return nil, fmt.Errorf("assistant %s not found", assistantID)
	}

	data := row.ToMap()
	if data == nil || len(data) == 0 {
		return nil, fmt.Errorf("assistant %s not found", assistantID)
	}

	// Parse JSON fields
	jsonFields := []string{"tags", "options", "prompts", "flows", "files", "tools", "permissions", "placeholder"}
	conv.parseJSONFields(data, jsonFields)

	return data, nil
}

// DeleteAssistants deletes assistants based on filter conditions
func (conv *Xun) DeleteAssistants(filter AssistantFilter) (int64, error) {
	qb := conv.query.New().
		Table(conv.getAssistantTable())

	// Apply tag filter if provided
	if filter.Tags != nil && len(filter.Tags) > 0 {
		qb.Where(func(qb query.Query) {
			for i, tag := range filter.Tags {
				pattern := fmt.Sprintf("%%\"%s\"%%", tag)
				if i == 0 {
					qb.Where("tags", "like", pattern)
				} else {
					qb.OrWhere("tags", "like", pattern)
				}
			}
		})
	}

	// Apply keyword filter if provided
	if filter.Keywords != "" {
		qb.Where(func(qb query.Query) {
			qb.Where("name", "like", fmt.Sprintf("%%%s%%", filter.Keywords)).
				OrWhere("description", "like", fmt.Sprintf("%%%s%%", filter.Keywords))
		})
	}

	// Apply connector filter if provided
	if filter.Connector != "" {
		qb.Where("connector", filter.Connector)
	}

	// Apply assistant_id filter if provided
	if filter.AssistantID != "" {
		qb.Where("assistant_id", filter.AssistantID)
	}

	// Apply assistantIDs filter if provided
	if filter.AssistantIDs != nil && len(filter.AssistantIDs) > 0 {
		qb.WhereIn("assistant_id", filter.AssistantIDs)
	}

	// Apply mentionable filter if provided
	if filter.Mentionable != nil {
		qb.Where("mentionable", *filter.Mentionable)
	}

	// Apply automated filter if provided
	if filter.Automated != nil {
		qb.Where("automated", *filter.Automated)
	}

	// Apply built_in filter if provided
	if filter.BuiltIn != nil {
		qb.Where("built_in", *filter.BuiltIn)
	}

	// Execute delete and return number of deleted records
	return qb.Delete()
}

// GetAssistantTags retrieves all unique tags from assistants
func (conv *Xun) GetAssistantTags() ([]string, error) {
	q := conv.newQuery().Table(conv.getAssistantTable())
	rows, err := q.Select("tags").GroupBy("tags").Get()
	if err != nil {
		return nil, err
	}

	tagSet := map[string]bool{}
	for _, row := range rows {
		if tags, ok := row["tags"].(string); ok && tags != "" {
			var tagList []string
			if err := jsoniter.UnmarshalFromString(tags, &tagList); err == nil {
				for _, tag := range tagList {
					tagSet[tag] = true
				}
			}
		}
	}

	// Convert map keys to slice
	tags := make([]string, 0, len(tagSet))
	for tag := range tagSet {
		tags = append(tags, tag)
	}
	return tags, nil
}

// GetHistoryWithFilter get the history with filter options
func (conv *Xun) GetHistoryWithFilter(sid string, cid string, filter ChatFilter) ([]map[string]interface{}, error) {
	userID, err := conv.getUserID(sid)
	if err != nil {
		return nil, err
	}

	qb := conv.newQuery().
		Select("role", "name", "content", "context", "assistant_id", "assistant_name", "assistant_avatar", "mentions", "uid", "silent", "created_at", "updated_at").
		Where("sid", userID).
		Where("cid", cid).
		OrderBy("id", "desc")

	// Apply silent filter if provided, otherwise exclude silent messages by default
	if filter.Silent != nil {
		if *filter.Silent {
			// Include all messages (both silent and non-silent)
		} else {
			// Only include non-silent messages
			qb.Where("silent", false)
		}
	} else {
		// Default behavior: exclude silent messages
		qb.Where("silent", false)
	}

	if conv.setting.TTL > 0 {
		qb.Where("expired_at", ">", time.Now())
	}

	limit := 20
	if conv.setting.MaxSize > 0 {
		limit = conv.setting.MaxSize
	}
	if filter.PageSize > 0 {
		limit = filter.PageSize
	}

	// Apply pagination if provided
	if filter.Page > 0 {
		offset := (filter.Page - 1) * limit
		qb.Offset(offset)
	}

	rows, err := qb.Limit(limit).Get()
	if err != nil {
		return nil, err
	}

	res := []map[string]interface{}{}
	for _, row := range rows {
		message := map[string]interface{}{
			"role":             row.Get("role"),
			"name":             row.Get("name"),
			"content":          row.Get("content"),
			"context":          row.Get("context"),
			"assistant_id":     row.Get("assistant_id"),
			"assistant_name":   row.Get("assistant_name"),
			"assistant_avatar": row.Get("assistant_avatar"),
			"mentions":         row.Get("mentions"),
			"uid":              row.Get("uid"),
			"silent":           row.Get("silent"),
			"created_at":       row.Get("created_at"),
			"updated_at":       row.Get("updated_at"),
		}
		res = append([]map[string]interface{}{message}, res...)
	}

	return res, nil
}
