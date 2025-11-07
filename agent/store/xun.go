package store

import (
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/google/uuid"
	jsoniter "github.com/json-iterator/go"
	"github.com/yaoapp/gou/connector"
	"github.com/yaoapp/gou/model"
	"github.com/yaoapp/kun/log"
	"github.com/yaoapp/xun/capsule"
	"github.com/yaoapp/xun/dbal/query"
	"github.com/yaoapp/xun/dbal/schema"
	"github.com/yaoapp/yao/agent/i18n"
)

// Package store provides functionality for managing chat conversations and assistants.

// Xun implements the Store interface using a database backend.
// It provides functionality for:
// - Managing chat conversations and their message histories
// - Organizing chats with pagination and date-based grouping
// - Handling chat metadata like titles and creation dates
// - Managing AI assistants with their configurations and metadata
// - Supporting data expiration through TTL settings
type Xun struct {
	query       query.Query
	schema      schema.Schema
	setting     Setting
	cleanTicker *time.Ticker
	cleanStop   chan bool
}

// Public interface methods:
//
// NewXun creates a new conversation instance with the given settings
// GetChats retrieves a paginated list of chats grouped by date
// GetChat retrieves a specific chat and its message history
// GetChatWithFilter retrieves a specific chat with filter options
// GetHistory retrieves the message history for a specific chat
// GetHistoryWithFilter retrieves the message history with filter options
// SaveHistory saves new messages to a chat's history
// DeleteChat deletes a specific chat and its history
// DeleteAllChats deletes all chats and their histories for a user
// UpdateChatTitle updates the title of a specific chat
// SaveAssistant creates or updates an assistant
// DeleteAssistant deletes an assistant by assistant_id
// GetAssistants retrieves a paginated list of assistants with filtering
// GetAssistant retrieves a single assistant by assistant_id
// DeleteAssistants deletes assistants based on filter conditions
// GetAssistantTags retrieves all unique tags from assistants
// Close closes the store and releases any resources

// NewXun create a new xun store
func NewXun(setting Setting) (Store, error) {
	conv := &Xun{setting: setting}
	if setting.Connector == "default" || setting.Connector == "" {
		conv.query = capsule.Global.Query()
		conv.schema = capsule.Global.Schema()
	} else {
		conn, err := connector.Select(setting.Connector)
		if err != nil {
			return nil, fmt.Errorf("select store connector %s error: %s", setting.Connector, err.Error())
		}

		conv.query, err = conn.Query()
		if err != nil {
			return nil, fmt.Errorf("query store connector %s error: %s", setting.Connector, err.Error())
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
		log.Trace("Clean the conversation table: %d", nums)
	}
}

// startAutoClean starts the automatic cleanup routine
func (conv *Xun) startAutoClean() {
	if conv.cleanTicker != nil {
		conv.stopAutoClean() // Stop existing ticker if any
	}

	conv.cleanTicker = time.NewTicker(1 * time.Hour) // Clean every hour
	conv.cleanStop = make(chan bool)

	go func() {
		for {
			select {
			case <-conv.cleanTicker.C:
				conv.clean()
			case <-conv.cleanStop:
				return
			}
		}
	}()

	log.Trace("Started automatic cleanup")
}

// stopAutoClean stops the automatic cleanup routine
func (conv *Xun) stopAutoClean() {
	if conv.cleanTicker != nil {
		conv.cleanTicker.Stop()
		conv.cleanTicker = nil
	}

	if conv.cleanStop != nil {
		close(conv.cleanStop)
		conv.cleanStop = nil
	}

	log.Trace("Stopped automatic cleanup")
}

// Close stops the automatic cleanup and closes resources
func (conv *Xun) Close() error {
	conv.stopAutoClean()
	return nil
}

// Rename Init to initialize to avoid conflicts
func (conv *Xun) initialize() error {

	// Start automatic cleanup if TTL is enabled
	if conv.setting.TTL > 0 {
		conv.startAutoClean()
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
			table.TimestampTz("created_at").SetDefaultRaw("CURRENT_TIMESTAMP").Index()
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
			table.TimestampTz("created_at").SetDefaultRaw("CURRENT_TIMESTAMP").Index()
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

func (conv *Xun) getUserID(sid string) (string, error) {
	// TODO: get the user id from the authentication system
	return "guest", nil
}

func (conv *Xun) getHistoryTable() string {
	m := model.Select("__yao.agent.history")
	if m != nil && m.MetaData.Table.Name != "" {
		return m.MetaData.Table.Name
	}
	return "__yao.agent.history"
}

func (conv *Xun) getChatTable() string {
	m := model.Select("__yao.agent.chat")
	if m != nil && m.MetaData.Table.Name != "" {
		return m.MetaData.Table.Name
	}
	return "__yao.agent.chat"
}

func (conv *Xun) getAssistantTable() string {
	m := model.Select("__yao.agent.assistant")
	if m != nil && m.MetaData.Table.Name != "" {
		return m.MetaData.Table.Name
	}
	return "__yao.agent.assistant"
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
func (conv *Xun) GetChats(sid string, filter ChatFilter, locale ...string) (*ChatGroupResponse, error) {
	// Default behavior: exclude silent chats
	if filter.Silent == nil {
		silentFalse := false
		filter.Silent = &silentFalse
	}

	return conv.getChatsWithFilter(sid, filter, locale...)
}

// getChatsWithFilter get the chats with filter options
func (conv *Xun) getChatsWithFilter(sid string, filter ChatFilter, locale ...string) (*ChatGroupResponse, error) {
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
				name := assistant.Get("name")
				if len(locale) > 0 {
					lang := strings.ToLower(locale[0])
					name = i18n.Translate(id.(string), lang, name).(string)
				}
				assistantMap[fmt.Sprintf("%v", id)] = map[string]interface{}{
					"name":   name,
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
				name := assistant["name"]
				if len(locale) > 0 {
					lang := strings.ToLower(locale[0])
					name = i18n.Translate(assistantID.(string), lang, name).(string)
				}
				chat["assistant_name"] = name
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

	// Convert to ordered slice and apply i18n
	result := []ChatGroup{}
	for _, label := range []string{"Today", "Yesterday", "This Week", "Last Week", "Even Earlier"} {
		if len(groups[label]) > 0 {
			translatedLabel := label
			if len(locale) > 0 {
				lang := strings.ToLower(locale[0])
				translatedLabel = i18n.TranslateGlobal(lang, label).(string)
			}
			result = append(result, ChatGroup{
				Label: translatedLabel,
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
func (conv *Xun) GetHistory(sid string, cid string, locale ...string) ([]map[string]interface{}, error) {
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
		assistantName := row.Get("assistant_name")
		assistantID := row.Get("assistant_id")
		if len(locale) > 0 && assistantID != nil {
			lang := strings.ToLower(locale[0])
			assistantName = i18n.Translate(assistantID.(string), lang, assistantName).(string)
		}

		message := map[string]interface{}{
			"role":             row.Get("role"),
			"name":             row.Get("name"),
			"content":          row.Get("content"),
			"context":          row.Get("context"),
			"assistant_id":     row.Get("assistant_id"),
			"assistant_name":   assistantName,
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
	var historyVisible bool = true
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

		// Get history visible from context
		if historyVisibleVal, ok := context["history_visible"]; ok {
			switch v := historyVisibleVal.(type) {
			case bool:
				historyVisible = v
			case string:
				historyVisible = v == "true" || v == "1" || v == "yes"
			case int:
				historyVisible = v != 0
			case float64:
				historyVisible = v != 0
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
				"silent":       silent || historyVisible == false,
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
				"silent":       silent || historyVisible == false,
			})
		if err != nil {
			return err
		}
	}

	// Save message history
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
func (conv *Xun) GetChat(sid string, cid string, locale ...string) (*ChatInfo, error) {
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

		name := assistant.Get("name")
		if len(locale) > 0 {
			lang := strings.ToLower(locale[0])
			name = i18n.Translate(assistantID.(string), lang, name).(string)
		}

		if assistant != nil {
			chat["assistant_name"] = name
			chat["assistant_avatar"] = assistant.Get("avatar")
		}
	}

	// Get chat history with default filter (silent=false)
	history, err := conv.GetHistory(sid, cid, locale...)
	if err != nil {
		return nil, err
	}

	return &ChatInfo{
		Chat:    chat,
		History: history,
	}, nil
}

// GetChatWithFilter get the chat info and its history with filter options
func (conv *Xun) GetChatWithFilter(sid string, cid string, filter ChatFilter, locale ...string) (*ChatInfo, error) {
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
	history, err := conv.GetHistoryWithFilter(sid, cid, filter, locale...)
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
func (conv *Xun) SaveAssistant(assistant *AssistantModel) (string, error) {
	if assistant == nil {
		return "", fmt.Errorf("assistant cannot be nil")
	}

	// Validate required fields
	if assistant.Name == "" {
		return "", fmt.Errorf("field name is required")
	}
	if assistant.Type == "" {
		return "", fmt.Errorf("field type is required")
	}
	if assistant.Connector == "" {
		return "", fmt.Errorf("field connector is required")
	}

	// Generate assistant_id if not provided
	if assistant.ID == "" {
		var err error
		assistant.ID, err = conv.GenerateAssistantID()
		if err != nil {
			return "", err
		}
	}

	// Check if assistant exists
	exists, err := conv.query.New().
		Table(conv.getAssistantTable()).
		Where("assistant_id", assistant.ID).
		Exists()
	if err != nil {
		return "", err
	}

	// Convert model to map for database storage
	data := make(map[string]interface{})
	data["assistant_id"] = assistant.ID
	data["type"] = assistant.Type
	data["name"] = assistant.Name
	data["avatar"] = assistant.Avatar
	data["connector"] = assistant.Connector
	data["path"] = assistant.Path
	data["built_in"] = assistant.BuiltIn
	data["sort"] = assistant.Sort
	data["description"] = assistant.Description
	data["readonly"] = assistant.Readonly
	data["public"] = assistant.Public
	data["share"] = assistant.Share
	data["mentionable"] = assistant.Mentionable
	data["automated"] = assistant.Automated
	data["created_at"] = assistant.CreatedAt
	data["updated_at"] = assistant.UpdatedAt

	// Permission management fields
	if assistant.YaoCreatedBy != "" {
		data["__yao_created_by"] = assistant.YaoCreatedBy
	}
	if assistant.YaoUpdatedBy != "" {
		data["__yao_updated_by"] = assistant.YaoUpdatedBy
	}
	if assistant.YaoTeamID != "" {
		data["__yao_team_id"] = assistant.YaoTeamID
	}
	if assistant.YaoTenantID != "" {
		data["__yao_tenant_id"] = assistant.YaoTenantID
	}

	// Handle simple types
	if assistant.Options != nil {
		jsonStr, err := jsoniter.MarshalToString(assistant.Options)
		if err != nil {
			return "", fmt.Errorf("failed to marshal options: %w", err)
		}
		data["options"] = jsonStr
	}

	if assistant.Tags != nil {
		jsonStr, err := jsoniter.MarshalToString(assistant.Tags)
		if err != nil {
			return "", fmt.Errorf("failed to marshal tags: %w", err)
		}
		data["tags"] = jsonStr
	}

	// Handle interface{} fields - they should already be in the correct format
	jsonFields := map[string]interface{}{
		"prompts":     assistant.Prompts,
		"kb":          assistant.KB,
		"mcp":         assistant.MCP,
		"workflow":    assistant.Workflow,
		"tools":       assistant.Tools,
		"placeholder": assistant.Placeholder,
		"locales":     assistant.Locales,
	}

	for field, value := range jsonFields {
		if value != nil {
			jsonStr, err := jsoniter.MarshalToString(value)
			if err != nil {
				return "", fmt.Errorf("failed to marshal %s: %w", field, err)
			}
			data[field] = jsonStr
		}
	}

	// Update or insert
	if exists {
		_, err := conv.query.New().
			Table(conv.getAssistantTable()).
			Where("assistant_id", assistant.ID).
			Update(data)
		if err != nil {
			return "", err
		}
		return assistant.ID, nil
	}

	err = conv.query.New().
		Table(conv.getAssistantTable()).
		Insert(data)
	if err != nil {
		return "", err
	}
	return assistant.ID, nil
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
func (conv *Xun) GetAssistants(filter AssistantFilter, locale ...string) (*AssistantList, error) {
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

	// Apply type filter if provided
	if filter.Type != "" {
		qb.Where("type", filter.Type)
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

	// Convert rows to AssistantModel slice
	assistants := make([]*AssistantModel, 0, len(rows))
	jsonFields := []string{"tags", "options", "prompts", "workflow", "kb", "mcp", "tools", "placeholder", "locales"}

	for _, row := range rows {
		data := row.ToMap()
		if data == nil {
			continue
		}

		// Parse JSON fields
		conv.parseJSONFields(data, jsonFields)

		// Convert map to AssistantModel using existing helper function
		model, err := ToAssistantModel(data)
		if err != nil {
			log.Error("Failed to convert row to AssistantModel: %s", err.Error())
			continue
		}

		// Apply i18n translations if locale is provided
		if len(locale) > 0 && model != nil {
			lang := strings.ToLower(locale[0])
			// Translate name if locales are available
			if model.Locales != nil {
				if localeData, ok := model.Locales[lang]; ok {
					if messages, ok := localeData.Messages["name"]; ok {
						if nameStr, ok := messages.(string); ok {
							model.Name = nameStr
						}
					}
					if messages, ok := localeData.Messages["description"]; ok {
						if descStr, ok := messages.(string); ok {
							model.Description = descStr
						}
					}
				}
			}
		}

		assistants = append(assistants, model)
	}

	return &AssistantList{
		Data:      assistants,
		Page:      filter.Page,
		PageSize:  filter.PageSize,
		PageCount: totalPages,
		Next:      nextPage,
		Prev:      prevPage,
		Total:     int(total),
	}, nil
}

// GetAssistant retrieves a single assistant by ID
func (conv *Xun) GetAssistant(assistantID string, locale ...string) (*AssistantModel, error) {
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
		return nil, fmt.Errorf("the assistant %s is empty", assistantID)
	}

	// Parse JSON fields
	jsonFields := []string{"tags", "options", "prompts", "workflow", "kb", "mcp", "tools", "placeholder", "locales"}
	conv.parseJSONFields(data, jsonFields)

	// Convert map to AssistantModel
	model := &AssistantModel{
		ID:           getString(data, "assistant_id"),
		Type:         getString(data, "type"),
		Name:         getString(data, "name"),
		Avatar:       getString(data, "avatar"),
		Connector:    getString(data, "connector"),
		Path:         getString(data, "path"),
		BuiltIn:      getBool(data, "built_in"),
		Sort:         getInt(data, "sort"),
		Description:  getString(data, "description"),
		Readonly:     getBool(data, "readonly"),
		Public:       getBool(data, "public"),
		Share:        getString(data, "share"),
		Mentionable:  getBool(data, "mentionable"),
		Automated:    getBool(data, "automated"),
		CreatedAt:    getInt64(data, "created_at"),
		UpdatedAt:    getInt64(data, "updated_at"),
		YaoCreatedBy: getString(data, "__yao_created_by"),
		YaoUpdatedBy: getString(data, "__yao_updated_by"),
		YaoTeamID:    getString(data, "__yao_team_id"),
		YaoTenantID:  getString(data, "__yao_tenant_id"),
	}

	// Handle Tags
	if tags, ok := data["tags"].([]interface{}); ok {
		model.Tags = make([]string, len(tags))
		for i, tag := range tags {
			if s, ok := tag.(string); ok {
				model.Tags[i] = s
			}
		}
	}

	// Handle Options
	if options, ok := data["options"].(map[string]interface{}); ok {
		model.Options = options
	}

	// Handle typed fields with conversion
	if prompts, has := data["prompts"]; has && prompts != nil {
		// Try to unmarshal to []Prompt
		raw, err := jsoniter.Marshal(prompts)
		if err == nil {
			var p []Prompt
			if err := jsoniter.Unmarshal(raw, &p); err == nil {
				model.Prompts = p
			}
		}
	}

	if kb, has := data["kb"]; has && kb != nil {
		kbConverted, err := ToKnowledgeBase(kb)
		if err == nil {
			model.KB = kbConverted
		}
	}

	if mcp, has := data["mcp"]; has && mcp != nil {
		mcpConverted, err := ToMCPServers(mcp)
		if err == nil {
			model.MCP = mcpConverted
		}
	}

	if workflow, has := data["workflow"]; has && workflow != nil {
		wf, err := ToWorkflow(workflow)
		if err == nil {
			model.Workflow = wf
		}
	}

	if tools, has := data["tools"]; has && tools != nil {
		raw, err := jsoniter.Marshal(tools)
		if err == nil {
			var tc ToolCalls
			if err := jsoniter.Unmarshal(raw, &tc); err == nil {
				model.Tools = &tc
			}
		}
	}

	if placeholder, has := data["placeholder"]; has && placeholder != nil {
		raw, err := jsoniter.Marshal(placeholder)
		if err == nil {
			var ph Placeholder
			if err := jsoniter.Unmarshal(raw, &ph); err == nil {
				model.Placeholder = &ph
			}
		}
	}

	if locales, has := data["locales"]; has && locales != nil {
		raw, err := jsoniter.Marshal(locales)
		if err == nil {
			var loc i18n.Map
			if err := jsoniter.Unmarshal(raw, &loc); err == nil {
				model.Locales = loc
			}
		}
	}

	return model, nil
}

// Helper functions for type conversion
func getString(data map[string]interface{}, key string) string {
	if v, ok := data[key].(string); ok {
		return v
	}
	return ""
}

func getBool(data map[string]interface{}, key string) bool {
	if v, ok := data[key].(bool); ok {
		return v
	}
	return false
}

func getInt(data map[string]interface{}, key string) int {
	switch v := data[key].(type) {
	case int:
		return v
	case int64:
		return int(v)
	case float64:
		return int(v)
	}
	return 0
}

func getInt64(data map[string]interface{}, key string) int64 {
	switch v := data[key].(type) {
	case int64:
		return v
	case int:
		return int64(v)
	case float64:
		return int64(v)
	}
	return 0
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
func (conv *Xun) GetAssistantTags(locale ...string) ([]Tag, error) {
	q := conv.newQuery().Table(conv.getAssistantTable())
	rows, err := q.Select("tags").Where("type", "assistant").GroupBy("tags").Get()
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

	lang := "en"
	if len(locale) > 0 {
		lang = locale[0]
	}

	// Convert map keys to slice
	tags := make([]Tag, 0, len(tagSet))
	for tag := range tagSet {
		tags = append(tags, Tag{
			Value: tag,
			Label: i18n.TranslateGlobal(lang, tag).(string),
		})
	}
	return tags, nil
}

// GetHistoryWithFilter get the history with filter options
func (conv *Xun) GetHistoryWithFilter(sid string, cid string, filter ChatFilter, locale ...string) ([]map[string]interface{}, error) {
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

// GenerateAssistantID generates a random-looking 6-digit ID
func (conv *Xun) GenerateAssistantID() (string, error) {
	maxAttempts := 10 // Maximum number of attempts to generate a unique ID
	for i := 0; i < maxAttempts; i++ {
		// Generate a random number using timestamp and some bit operations
		timestamp := time.Now().UnixNano()
		random := (timestamp ^ (timestamp >> 12)) % 1000000
		hash := fmt.Sprintf("%06d", random)

		// Check if this ID already exists
		exists, err := conv.query.New().
			Table(conv.getAssistantTable()).
			Where("assistant_id", hash).
			Exists()

		if err != nil {
			return "", err
		}

		if !exists {
			return hash, nil
		}

		// If ID exists, wait a bit and try again
		time.Sleep(time.Millisecond)
	}

	return "", fmt.Errorf("failed to generate unique ID after %d attempts", maxAttempts)
}
