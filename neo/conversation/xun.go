package conversation

import (
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/yaoapp/gou/connector"
	"github.com/yaoapp/gou/session"
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
	Name      string      `json:"name"` // User name
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
	return conv.setting.Table
}

func (conv *Xun) getChatTable() string {
	return conv.setting.Table + "_chat"
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
	userID, err := conv.getUserID(sid)
	if err != nil {
		return nil, err
	}

	// Set defaults
	if filter.PageSize <= 0 {
		filter.PageSize = 100
	}
	if filter.Page <= 0 {
		filter.Page = 1
	}
	if filter.Order == "" {
		filter.Order = "desc"
	}

	// Build base query
	qb := conv.newQueryChat().
		Select("chat_id", "title", "created_at").
		Where("sid", userID).
		Where("chat_id", "!=", "")

	// Add keyword filter
	if filter.Keywords != "" {
		keyword := strings.TrimSpace(filter.Keywords)
		if keyword != "" {
			qb.Where("title", "like", "%"+keyword+"%")
		}
	}

	// Get total count
	total, err := qb.Clone().Count()
	if err != nil {
		return nil, err
	}

	// Calculate pagination
	offset := (filter.Page - 1) * filter.PageSize
	lastPage := int(math.Ceil(float64(total) / float64(filter.PageSize)))

	// Get paginated results
	rows, err := qb.OrderBy("created_at", filter.Order).
		Offset(offset).
		Limit(filter.PageSize).
		Get()
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

	for _, row := range rows {
		chatID := row.Get("chat_id")
		if chatID == nil || chatID == "" {
			continue
		}

		chat := map[string]interface{}{
			"chat_id": chatID,
			"title":   row.Get("title"),
		}

		var createdAt time.Time
		switch v := row.Get("created_at").(type) {
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
		Select("role", "name", "content").
		Where("sid", userID).
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

	if cid == "" {
		cid = uuid.New().String() // Generate a new UUID if cid is empty
	}

	userID, err := conv.getUserID(sid)
	if err != nil {
		return err
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
				"chat_id":    cid,
				"sid":        userID,
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
			Sid:       userID,
			Cid:       cid,
			ExpiredAt: expiredAt,
		}

		if message["name"] != nil {
			value.Name = message["name"].(string)
		}
		values = append(values, value)
	}

	err = conv.newQuery().Insert(values)
	if err != nil {
		return err
	}

	return nil
}

// GetRequest get the request history
func (conv *Xun) GetRequest(sid string, rid string) ([]map[string]interface{}, error) {
	userID, err := conv.getUserID(sid)
	if err != nil {
		return nil, err
	}

	qb := conv.newQuery().
		Select("role", "name", "content", "sid").
		Where("rid", rid).
		Where("sid", userID).
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
	userID, err := conv.getUserID(sid)
	if err != nil {
		return err
	}

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
			Sid:       userID,
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
	userID, err := conv.getUserID(sid)
	if err != nil {
		return nil, err
	}

	// Get chat info
	qb := conv.newQueryChat().
		Select("chat_id", "title").
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
