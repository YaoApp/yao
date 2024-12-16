package conversation

import (
	"fmt"
	"strings"
	"time"

	"github.com/yaoapp/gou/connector"
	"github.com/yaoapp/kun/log"
	"github.com/yaoapp/xun/capsule"
	"github.com/yaoapp/xun/dbal/query"
	"github.com/yaoapp/xun/dbal/schema"
)

// Xun Database conversation
type Xun struct {
	query   query.Query
	schema  schema.Schema
	setting Setting
}

type row struct {
	Role      string      `json:"role"`
	Title     string      `json:"title"` // Chat title
	Name      string      `json:"name"`  // User name
	Content   string      `json:"content"`
	Sid       string      `json:"sid"`
	Rid       string      `json:"rid"`
	Cid       string      `json:"cid"` // Chat ID from chat history
	ExpiredAt interface{} `json:"expired_at"`
}

// Public interface methods and constructor remain exported:
// - NewXun
// - UpdateChatTitle
// - GetChats
// - GetChat
// - GetHistory
// - SaveHistory
// - GetRequest
// - SaveRequest

// NewXun create a new conversation
func NewXun(setting Setting) (*Xun, error) {
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
		log.Trace("Clean the conversation table: %s %d", conv.setting.Table, nums)
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
			table.String("rid", 255).Null().Index()
			table.String("cid", 200).Null().Index()
			table.String("role", 200).Null().Index()
			table.String("name", 200).Null().Index()
			table.Text("content").Null()
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

	fields := []string{"id", "sid", "rid", "cid", "role", "name", "content", "created_at", "updated_at", "expired_at"}
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
			table.String("sid", 255).Index()
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

	fields := []string{"id", "chat_id", "title", "sid", "created_at", "updated_at"}
	for _, field := range fields {
		if !tab.HasColumn(field) {
			return fmt.Errorf("%s is required", field)
		}
	}

	return nil
}

func (conv *Xun) getHistoryTable() string {
	return conv.setting.Table
}

func (conv *Xun) getChatTable() string {
	return conv.setting.Table + "_chat"
}

// UpdateChatTitle update the chat title
func (conv *Xun) UpdateChatTitle(sid string, cid string, title string) error {
	_, err := conv.newQueryChat().
		Where("sid", sid).
		Where("chat_id", cid).
		Update(map[string]interface{}{
			"title":      title,
			"updated_at": time.Now(),
		})
	return err
}

// GetChats get the chat list
func (conv *Xun) GetChats(sid string, keywords ...string) ([]map[string]interface{}, error) {
	qb := conv.newQueryChat().
		Select("chat_id", "title").
		Where("sid", sid)

	// Add title search if keywords provided
	if len(keywords) > 0 && keywords[0] != "" {
		keyword := strings.TrimSpace(keywords[0]) // Trim whitespace from keyword
		if keyword != "" {
			qb.Where("title", "like", "%"+keyword+"%")
		}
	}

	rows, err := qb.Get()
	if err != nil {
		return nil, err
	}

	res := []map[string]interface{}{}
	for _, row := range rows {
		res = append(res, map[string]interface{}{
			"chat_id": row.Get("chat_id"),
			"title":   row.Get("title"),
		})
	}

	return res, nil
}

// GetHistory get the history
func (conv *Xun) GetHistory(sid string, cid string) ([]map[string]interface{}, error) {

	qb := conv.newQuery().
		Select("role", "name", "content").
		Where("sid", sid).
		Where("cid", cid).
		OrderBy("id", "desc")

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
		res = append([]map[string]interface{}{{
			"role":    row.Get("role"),
			"name":    row.Get("name"),
			"content": row.Get("content"),
		}}, res...)
	}

	return res, nil
}

// SaveHistory save the history
func (conv *Xun) SaveHistory(sid string, messages []map[string]interface{}, cid string) error {
	// First ensure chat record exists
	exists, err := conv.newQueryChat().
		Where("chat_id", cid).
		Where("sid", sid).
		Exists()

	if err != nil {
		return err
	}

	if !exists {
		// Create new chat record
		err = conv.newQueryChat().
			Insert(map[string]interface{}{
				"chat_id":    cid,
				"sid":        sid,
				"created_at": time.Now(),
			})

		if err != nil {
			return err
		}
	}

	// Save message history
	defer conv.clean()
	var expiredAt interface{} = nil
	values := []row{}
	if conv.setting.TTL > 0 {
		expiredAt = time.Now().Add(time.Duration(conv.setting.TTL) * time.Second)
	}

	for _, message := range messages {
		value := row{
			Role:      message["role"].(string),
			Name:      "",
			Content:   message["content"].(string),
			Sid:       sid,
			Cid:       cid,
			ExpiredAt: expiredAt,
		}

		if message["name"] != nil {
			value.Name = message["name"].(string)
		}
		values = append(values, value)
	}

	return conv.newQuery().Insert(values)
}

// GetRequest get the request history
func (conv *Xun) GetRequest(sid string, rid string) ([]map[string]interface{}, error) {

	qb := conv.newQuery().
		Select("role", "name", "content", "sid").
		Where("rid", rid).
		Where("sid", sid).
		OrderBy("id", "desc")

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
		res = append([]map[string]interface{}{{
			"role":    row.Get("role"),
			"name":    row.Get("name"),
			"content": row.Get("content"),
		}}, res...)
	}

	return res, nil
}

// SaveRequest save the request history
func (conv *Xun) SaveRequest(sid string, rid string, cid string, messages []map[string]interface{}) error {

	defer conv.clean()
	var expiredAt interface{} = nil
	values := []row{}
	if conv.setting.TTL > 0 {
		expiredAt = time.Now().Add(time.Duration(conv.setting.TTL) * time.Second)
	}

	for _, message := range messages {
		value := row{
			Role:      message["role"].(string),
			Name:      "",
			Content:   message["content"].(string),
			Sid:       sid,
			Cid:       cid,
			Rid:       rid,
			ExpiredAt: expiredAt,
		}

		if message["name"] != nil {
			value.Name = message["name"].(string)
		}
		values = append(values, value)
	}

	return conv.newQuery().Insert(values)
}

// GetChat get the chat info and its history
func (conv *Xun) GetChat(sid string, cid string) (*ChatInfo, error) {
	// Get chat info
	qb := conv.newQueryChat().
		Select("chat_id", "title").
		Where("sid", sid).
		Where("chat_id", cid)

	row, err := qb.First()
	if err != nil {
		return nil, err
	}

	chat := map[string]interface{}{
		"chat_id": row.Get("chat_id"),
		"title":   row.Get("title"),
	}

	// Get chat history
	history, err := conv.GetHistory(sid, cid)
	if err != nil {
		return nil, err
	}

	return &ChatInfo{
		Chat:    chat,
		History: history,
	}, nil
}
