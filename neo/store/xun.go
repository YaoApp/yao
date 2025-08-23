package store

import (
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/google/uuid"
	jsoniter "github.com/json-iterator/go"
	"github.com/yaoapp/gou/connector"
	"github.com/yaoapp/gou/session"
	"github.com/yaoapp/kun/log"
	"github.com/yaoapp/xun/capsule"
	"github.com/yaoapp/xun/dbal/query"
	"github.com/yaoapp/xun/dbal/schema"
	"github.com/yaoapp/yao/neo/i18n"
)

// Package conversation provides functionality for managing chat conversations and assistants.

// Xun implements the Conversation interface using a database backend.
// It provides functionality for:
// - Managing chat conversations and their message histories
// - Organizing chats with pagination and date-based grouping
// - Handling chat metadata like titles and creation dates
// - Managing AI assistants with their configurations and metadata
// - Managing file attachments with metadata and access control
// - Managing knowledge collections for AI assistants
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
// SaveAttachment creates or updates an attachment
// DeleteAttachment deletes an attachment by file_id
// GetAttachments retrieves a paginated list of attachments with filtering
// GetAttachment retrieves a single attachment by file_id
// SaveKnowledge creates or updates a knowledge collection
// DeleteKnowledge deletes a knowledge collection by collection_id
// GetKnowledges retrieves a paginated list of knowledge collections with filtering
// GetKnowledge retrieves a single knowledge collection by collection_id

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

	log.Trace("Started automatic cleanup for: %s", conv.setting.Prefix)
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

	log.Trace("Stopped automatic cleanup for: %s", conv.setting.Prefix)
}

// Close stops the automatic cleanup and closes resources
func (conv *Xun) Close() error {
	conv.stopAutoClean()
	return nil
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

	// Initialize attachment table
	if err := conv.initAttachmentTable(); err != nil {
		return err
	}

	// Initialize knowledge table
	if err := conv.initKnowledgeTable(); err != nil {
		return err
	}

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
			table.String("description", 600).Null().Index()           // assistant description
			table.String("path", 200).Null()                          // assistant storage path
			table.Integer("sort").SetDefault(9999).Index()            // assistant sort order
			table.Boolean("built_in").SetDefault(false).Index()       // whether this is a built-in assistant
			table.JSON("placeholder").Null()                          // assistant placeholder
			table.JSON("options").Null()                              // assistant options
			table.JSON("prompts").Null()                              // assistant prompts
			table.JSON("workflow").Null()                             // assistant workflow
			table.JSON("knowledge").Null()                            // assistant knowledge
			table.JSON("tools").Null()                                // assistant tools
			table.JSON("tags").Null()                                 // assistant tags
			table.Boolean("readonly").SetDefault(false).Index()       // assistant readonly
			table.JSON("permissions").Null()                          // assistant permissions
			table.JSON("locales").Null()                              // assistant i18n
			table.Boolean("automated").SetDefault(true).Index()       // assistant autoable
			table.Boolean("mentionable").SetDefault(true).Index()     // Whether this assistant can appear in @ mention list
			table.TimestampTz("created_at").SetDefaultRaw("CURRENT_TIMESTAMP").Index()
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

	fields := []string{"id", "assistant_id", "type", "name", "avatar", "connector", "description", "path", "sort", "built_in", "placeholder", "options", "prompts", "workflow", "knowledge", "tools", "tags", "readonly", "permissions", "locales", "automated", "mentionable", "created_at", "updated_at"}
	for _, field := range fields {
		if !tab.HasColumn(field) {
			return fmt.Errorf("%s is required", field)
		}
	}

	return nil
}

func (conv *Xun) initAttachmentTable() error {
	attachmentTable := conv.getAttachmentTable()
	has, err := conv.schema.HasTable(attachmentTable)
	if err != nil {
		return err
	}

	// Create the attachment table
	if !has {
		err = conv.schema.CreateTable(attachmentTable, func(table schema.Blueprint) {
			table.ID("id")
			table.String("file_id", 255).Unique().Index()
			table.String("uid", 255).Index()
			table.Boolean("guest").SetDefault(false).Index()
			table.String("manager", 200).Index()
			table.String("content_type", 200).Index()
			table.String("name", 500).Index()
			table.Boolean("public").SetDefault(false).Index()
			table.JSON("scope").Null()
			table.Boolean("gzip").SetDefault(false).Index()
			table.BigInteger("bytes").Index()
			table.String("collection_id", 200).Null().Index()
			table.Enum("status", []string{"uploading", "uploaded", "indexing", "indexed", "upload_failed", "index_failed"}).SetDefault("uploading").Index() // Status field enum
			table.String("progress", 200).Null()                                                                                                            // Progress information
			table.String("error", 600).Null()                                                                                                               // Error information
			table.TimestampTz("created_at").SetDefaultRaw("CURRENT_TIMESTAMP").Index()
			table.TimestampTz("updated_at").Null().Index()
		})

		if err != nil {
			return err
		}
		log.Trace("Create the attachment table: %s", attachmentTable)
	}

	// Validate the table
	tab, err := conv.schema.GetTable(attachmentTable)
	if err != nil {
		return err
	}

	fields := []string{"id", "file_id", "uid", "guest", "manager", "content_type", "name", "public", "scope", "gzip", "bytes", "collection_id", "status", "progress", "error", "created_at", "updated_at"}
	for _, field := range fields {
		if !tab.HasColumn(field) {
			return fmt.Errorf("%s is required", field)
		}
	}

	return nil
}

func (conv *Xun) initKnowledgeTable() error {
	knowledgeTable := conv.getKnowledgeTable()
	has, err := conv.schema.HasTable(knowledgeTable)
	if err != nil {
		return err
	}

	// Create the knowledge table
	if !has {
		err = conv.schema.CreateTable(knowledgeTable, func(table schema.Blueprint) {
			table.ID("id")
			table.String("collection_id", 200).Unique().Index()
			table.String("name", 200).Index()
			table.String("description", 600).Null().Index() // knowledge description
			table.String("uid", 255).Index()
			table.Boolean("public").SetDefault(false).Index()
			table.JSON("scope").Null()
			table.Boolean("readonly").SetDefault(false).Index()
			table.JSON("option").Null()
			table.Boolean("system").SetDefault(false).Index()
			table.Integer("sort").SetDefault(9999).Index() // knowledge sort order
			table.String("cover", 500).Null()
			table.TimestampTz("created_at").SetDefaultRaw("CURRENT_TIMESTAMP").Index()
			table.TimestampTz("updated_at").Null().Index()
		})

		if err != nil {
			return err
		}
		log.Trace("Create the knowledge table: %s", knowledgeTable)
	}

	// Validate the table
	tab, err := conv.schema.GetTable(knowledgeTable)
	if err != nil {
		return err
	}

	fields := []string{"id", "collection_id", "name", "description", "uid", "public", "scope", "readonly", "option", "system", "sort", "cover", "created_at", "updated_at"}
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

func (conv *Xun) getAttachmentTable() string {
	return conv.setting.Prefix + "attachment"
}

func (conv *Xun) getKnowledgeTable() string {
	return conv.setting.Prefix + "knowledge"
}

func (conv *Xun) newQueryAttachment() query.Query {
	qb := conv.query.New()
	qb.Table(conv.getAttachmentTable())
	return qb
}

func (conv *Xun) newQueryKnowledge() query.Query {
	qb := conv.query.New()
	qb.Table(conv.getKnowledgeTable())
	return qb
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
	jsonFields := []string{"tags", "options", "prompts", "workflow", "knowledge", "tools", "permissions", "placeholder", "locales"}
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
		var err error
		assistantCopy["assistant_id"], err = conv.GenerateAssistantID()
		if err != nil {
			return nil, err
		}
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
func (conv *Xun) GetAssistants(filter AssistantFilter, locale ...string) (*AssistantResponse, error) {
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

	// Convert rows to map slice and parse JSON fields
	data := make([]map[string]interface{}, len(rows))
	jsonFields := []string{"tags", "options", "prompts", "workflow", "knowledge", "tools", "permissions", "placeholder"}
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

	// Translate Data
	if len(locale) > 0 {
		lang := strings.ToLower(locale[0])
		for i, row := range data {
			assistantID := row["assistant_id"].(string)
			data[i] = i18n.Translate(assistantID, lang, row).(map[string]interface{})
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
func (conv *Xun) GetAssistant(assistantID string, locale ...string) (map[string]interface{}, error) {
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
	jsonFields := []string{"tags", "options", "prompts", "workflow", "knowledge", "tools", "permissions", "placeholder"}
	conv.parseJSONFields(data, jsonFields)
	if len(locale) > 0 {
		lang := strings.ToLower(locale[0])
		return i18n.Translate(assistantID, lang, data).(map[string]interface{}), nil
	}
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

// SaveAttachment saves attachment information
func (conv *Xun) SaveAttachment(attachment map[string]interface{}) (interface{}, error) {
	// Validate required fields
	requiredFields := []string{"file_id", "uid", "manager", "content_type", "name"}
	for _, field := range requiredFields {
		if _, ok := attachment[field]; !ok {
			return nil, fmt.Errorf("field %s is required", field)
		}
		if attachment[field] == nil || attachment[field] == "" {
			return nil, fmt.Errorf("field %s cannot be empty", field)
		}
	}

	// Create a copy of the attachment map to avoid modifying the original
	attachmentCopy := make(map[string]interface{})
	for k, v := range attachment {
		attachmentCopy[k] = v
	}

	// Process JSON fields
	jsonFields := []string{"scope"}
	for _, field := range jsonFields {
		if val, ok := attachmentCopy[field]; ok && val != nil {
			// If it's a string, try to parse it first
			if strVal, ok := val.(string); ok && strVal != "" {
				var parsed interface{}
				if err := jsoniter.UnmarshalFromString(strVal, &parsed); err == nil {
					attachmentCopy[field] = parsed
				}
			}
		}
	}

	// Check if attachment exists
	exists, err := conv.query.New().
		Table(conv.getAttachmentTable()).
		Where("file_id", attachmentCopy["file_id"]).
		Exists()
	if err != nil {
		return nil, err
	}

	// Convert JSON fields to strings for storage
	for _, field := range jsonFields {
		if val, ok := attachmentCopy[field]; ok && val != nil {
			jsonStr, err := jsoniter.MarshalToString(val)
			if err != nil {
				return nil, fmt.Errorf("failed to marshal %s to JSON: %v", field, err)
			}
			attachmentCopy[field] = jsonStr
		}
	}

	// Update or insert
	if exists {
		attachmentCopy["updated_at"] = time.Now()
		_, err := conv.query.New().
			Table(conv.getAttachmentTable()).
			Where("file_id", attachmentCopy["file_id"]).
			Update(attachmentCopy)
		if err != nil {
			return nil, err
		}
		return attachmentCopy["file_id"], nil
	}

	attachmentCopy["created_at"] = time.Now()
	err = conv.query.New().
		Table(conv.getAttachmentTable()).
		Insert(attachmentCopy)
	if err != nil {
		return nil, err
	}
	return attachmentCopy["file_id"], nil
}

// DeleteAttachment deletes an attachment by file_id
func (conv *Xun) DeleteAttachment(fileID string) error {
	// Check if attachment exists
	exists, err := conv.query.New().
		Table(conv.getAttachmentTable()).
		Where("file_id", fileID).
		Exists()
	if err != nil {
		return err
	}

	if !exists {
		return fmt.Errorf("attachment %s not found", fileID)
	}

	_, err = conv.query.New().
		Table(conv.getAttachmentTable()).
		Where("file_id", fileID).
		Delete()
	return err
}

// GetAttachments retrieves attachments with pagination and filtering
func (conv *Xun) GetAttachments(filter AttachmentFilter, locale ...string) (*AttachmentResponse, error) {
	qb := conv.query.New().
		Table(conv.getAttachmentTable())

	// Apply UID filter if provided
	if filter.UID != "" {
		qb.Where("uid", filter.UID)
	}

	// Apply guest filter if provided
	if filter.Guest != nil {
		qb.Where("guest", *filter.Guest)
	}

	// Apply manager filter if provided
	if filter.Manager != "" {
		qb.Where("manager", filter.Manager)
	}

	// Apply content_type filter if provided
	if filter.ContentType != "" {
		qb.Where("content_type", filter.ContentType)
	}

	// Apply name filter if provided
	if filter.Name != "" {
		qb.Where("name", "like", fmt.Sprintf("%%%s%%", filter.Name))
	}

	// Apply public filter if provided
	if filter.Public != nil {
		qb.Where("public", *filter.Public)
	}

	// Apply gzip filter if provided
	if filter.Gzip != nil {
		qb.Where("gzip", *filter.Gzip)
	}

	// Apply collection_id filter if provided
	if filter.CollectionID != "" {
		qb.Where("collection_id", filter.CollectionID)
	}

	// Apply status filter if provided
	if filter.Status != "" {
		qb.Where("status", filter.Status)
	}

	// Apply keyword filter if provided
	if filter.Keywords != "" {
		qb.Where("name", "like", fmt.Sprintf("%%%s%%", filter.Keywords))
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
	rows, err := qb.OrderBy("created_at", "desc").
		Offset(offset).
		Limit(filter.PageSize).
		Get()
	if err != nil {
		return nil, err
	}

	// Convert rows to map slice and parse JSON fields
	data := make([]map[string]interface{}, len(rows))
	jsonFields := []string{"scope"}
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

	return &AttachmentResponse{
		Data:     data,
		Page:     filter.Page,
		PageSize: filter.PageSize,
		PageCnt:  totalPages,
		Next:     nextPage,
		Prev:     prevPage,
		Total:    total,
	}, nil
}

// GetAttachment retrieves a single attachment by file_id
func (conv *Xun) GetAttachment(fileID string, locale ...string) (map[string]interface{}, error) {
	row, err := conv.query.New().
		Table(conv.getAttachmentTable()).
		Where("file_id", fileID).
		First()
	if err != nil {
		return nil, err
	}

	if row == nil {
		return nil, fmt.Errorf("attachment %s not found", fileID)
	}

	data := row.ToMap()
	if data == nil || len(data) == 0 {
		return nil, fmt.Errorf("the attachment %s is empty", fileID)
	}

	// Parse JSON fields
	jsonFields := []string{"scope"}
	conv.parseJSONFields(data, jsonFields)

	return data, nil
}

// DeleteAttachments deletes attachments based on filter conditions
func (conv *Xun) DeleteAttachments(filter AttachmentFilter) (int64, error) {
	qb := conv.query.New().
		Table(conv.getAttachmentTable())

	// Apply UID filter if provided
	if filter.UID != "" {
		qb.Where("uid", filter.UID)
	}

	// Apply guest filter if provided
	if filter.Guest != nil {
		qb.Where("guest", *filter.Guest)
	}

	// Apply manager filter if provided
	if filter.Manager != "" {
		qb.Where("manager", filter.Manager)
	}

	// Apply content_type filter if provided
	if filter.ContentType != "" {
		qb.Where("content_type", filter.ContentType)
	}

	// Apply name filter if provided
	if filter.Name != "" {
		qb.Where("name", "like", fmt.Sprintf("%%%s%%", filter.Name))
	}

	// Apply public filter if provided
	if filter.Public != nil {
		qb.Where("public", *filter.Public)
	}

	// Apply gzip filter if provided
	if filter.Gzip != nil {
		qb.Where("gzip", *filter.Gzip)
	}

	// Apply collection_id filter if provided
	if filter.CollectionID != "" {
		qb.Where("collection_id", filter.CollectionID)
	}

	// Apply status filter if provided
	if filter.Status != "" {
		qb.Where("status", filter.Status)
	}

	// Apply keyword filter if provided
	if filter.Keywords != "" {
		qb.Where("name", "like", fmt.Sprintf("%%%s%%", filter.Keywords))
	}

	// Execute delete and return number of deleted records
	return qb.Delete()
}

// SaveKnowledge saves knowledge collection information
func (conv *Xun) SaveKnowledge(knowledge map[string]interface{}) (interface{}, error) {
	// Validate required fields
	requiredFields := []string{"collection_id", "name", "uid"}
	for _, field := range requiredFields {
		if _, ok := knowledge[field]; !ok {
			return nil, fmt.Errorf("field %s is required", field)
		}
		if knowledge[field] == nil || knowledge[field] == "" {
			return nil, fmt.Errorf("field %s cannot be empty", field)
		}
	}

	// Create a copy of the knowledge map to avoid modifying the original
	knowledgeCopy := make(map[string]interface{})
	for k, v := range knowledge {
		knowledgeCopy[k] = v
	}

	// Process JSON fields
	jsonFields := []string{"scope", "option"}
	for _, field := range jsonFields {
		if val, ok := knowledgeCopy[field]; ok && val != nil {
			// If it's a string, try to parse it first
			if strVal, ok := val.(string); ok && strVal != "" {
				var parsed interface{}
				if err := jsoniter.UnmarshalFromString(strVal, &parsed); err == nil {
					knowledgeCopy[field] = parsed
				}
			}
		}
	}

	// Check if knowledge exists
	exists, err := conv.query.New().
		Table(conv.getKnowledgeTable()).
		Where("collection_id", knowledgeCopy["collection_id"]).
		Exists()
	if err != nil {
		return nil, err
	}

	// Convert JSON fields to strings for storage
	for _, field := range jsonFields {
		if val, ok := knowledgeCopy[field]; ok && val != nil {
			jsonStr, err := jsoniter.MarshalToString(val)
			if err != nil {
				return nil, fmt.Errorf("failed to marshal %s to JSON: %v", field, err)
			}
			knowledgeCopy[field] = jsonStr
		}
	}

	// Update or insert
	if exists {
		knowledgeCopy["updated_at"] = time.Now()
		_, err := conv.query.New().
			Table(conv.getKnowledgeTable()).
			Where("collection_id", knowledgeCopy["collection_id"]).
			Update(knowledgeCopy)
		if err != nil {
			return nil, err
		}
		return knowledgeCopy["collection_id"], nil
	}

	knowledgeCopy["created_at"] = time.Now()
	err = conv.query.New().
		Table(conv.getKnowledgeTable()).
		Insert(knowledgeCopy)
	if err != nil {
		return nil, err
	}
	return knowledgeCopy["collection_id"], nil
}

// DeleteKnowledge deletes a knowledge collection by collection_id
func (conv *Xun) DeleteKnowledge(collectionID string) error {
	// Check if knowledge exists
	exists, err := conv.query.New().
		Table(conv.getKnowledgeTable()).
		Where("collection_id", collectionID).
		Exists()
	if err != nil {
		return err
	}

	if !exists {
		return fmt.Errorf("knowledge collection %s not found", collectionID)
	}

	_, err = conv.query.New().
		Table(conv.getKnowledgeTable()).
		Where("collection_id", collectionID).
		Delete()
	return err
}

// GetKnowledges retrieves knowledge collections with pagination and filtering
func (conv *Xun) GetKnowledges(filter KnowledgeFilter, locale ...string) (*KnowledgeResponse, error) {
	qb := conv.query.New().
		Table(conv.getKnowledgeTable())

	// Apply UID filter if provided
	if filter.UID != "" {
		qb.Where("uid", filter.UID)
	}

	// Apply name filter if provided
	if filter.Name != "" {
		qb.Where("name", "like", fmt.Sprintf("%%%s%%", filter.Name))
	}

	// Apply keyword filter if provided
	if filter.Keywords != "" {
		qb.Where(func(qb query.Query) {
			qb.Where("name", "like", fmt.Sprintf("%%%s%%", filter.Keywords)).
				OrWhere("description", "like", fmt.Sprintf("%%%s%%", filter.Keywords))
		})
	}

	// Apply public filter if provided
	if filter.Public != nil {
		qb.Where("public", *filter.Public)
	}

	// Apply readonly filter if provided
	if filter.Readonly != nil {
		qb.Where("readonly", *filter.Readonly)
	}

	// Apply system filter if provided
	if filter.System != nil {
		qb.Where("system", *filter.System)
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
		OrderBy("created_at", "desc").
		Offset(offset).
		Limit(filter.PageSize).
		Get()
	if err != nil {
		return nil, err
	}

	// Convert rows to map slice and parse JSON fields
	data := make([]map[string]interface{}, len(rows))
	jsonFields := []string{"scope", "option"}
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

	return &KnowledgeResponse{
		Data:     data,
		Page:     filter.Page,
		PageSize: filter.PageSize,
		PageCnt:  totalPages,
		Next:     nextPage,
		Prev:     prevPage,
		Total:    total,
	}, nil
}

// GetKnowledge retrieves a single knowledge collection by collection_id
func (conv *Xun) GetKnowledge(collectionID string, locale ...string) (map[string]interface{}, error) {
	row, err := conv.query.New().
		Table(conv.getKnowledgeTable()).
		Where("collection_id", collectionID).
		First()
	if err != nil {
		return nil, err
	}

	if row == nil {
		return nil, fmt.Errorf("knowledge collection %s not found", collectionID)
	}

	data := row.ToMap()
	if data == nil || len(data) == 0 {
		return nil, fmt.Errorf("the knowledge collection %s is empty", collectionID)
	}

	// Parse JSON fields
	jsonFields := []string{"scope", "option"}
	conv.parseJSONFields(data, jsonFields)

	return data, nil
}

// DeleteKnowledges deletes knowledge collections based on filter conditions
func (conv *Xun) DeleteKnowledges(filter KnowledgeFilter) (int64, error) {
	qb := conv.query.New().
		Table(conv.getKnowledgeTable())

	// Apply UID filter if provided
	if filter.UID != "" {
		qb.Where("uid", filter.UID)
	}

	// Apply name filter if provided
	if filter.Name != "" {
		qb.Where("name", "like", fmt.Sprintf("%%%s%%", filter.Name))
	}

	// Apply keyword filter if provided
	if filter.Keywords != "" {
		qb.Where(func(qb query.Query) {
			qb.Where("name", "like", fmt.Sprintf("%%%s%%", filter.Keywords)).
				OrWhere("description", "like", fmt.Sprintf("%%%s%%", filter.Keywords))
		})
	}

	// Apply public filter if provided
	if filter.Public != nil {
		qb.Where("public", *filter.Public)
	}

	// Apply readonly filter if provided
	if filter.Readonly != nil {
		qb.Where("readonly", *filter.Readonly)
	}

	// Apply system filter if provided
	if filter.System != nil {
		qb.Where("system", *filter.System)
	}

	// Execute delete and return number of deleted records
	return qb.Delete()
}
