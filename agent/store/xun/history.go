package xun

import (
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	jsoniter "github.com/json-iterator/go"
	"github.com/yaoapp/yao/agent/i18n"
	"github.com/yaoapp/yao/agent/store/types"
)

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

// GetHistoryWithFilter get the history with filter options
func (conv *Xun) GetHistoryWithFilter(sid string, cid string, filter types.ChatFilter, locale ...string) ([]map[string]interface{}, error) {
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
