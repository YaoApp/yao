package xun

import (
	"fmt"
	"time"

	jsoniter "github.com/json-iterator/go"
	"github.com/yaoapp/gou/connector"
	"github.com/yaoapp/gou/model"
	"github.com/yaoapp/kun/log"
	"github.com/yaoapp/xun/capsule"
	"github.com/yaoapp/xun/dbal/query"
	"github.com/yaoapp/xun/dbal/schema"
	"github.com/yaoapp/yao/agent/store/types"
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
	setting     types.Setting
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
// SaveHistory saves new messages to a chat's historys
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
func NewXun(setting types.Setting) (types.Store, error) {
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
