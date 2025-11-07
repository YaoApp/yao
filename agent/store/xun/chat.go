package xun

import (
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/yaoapp/yao/agent/i18n"
	"github.com/yaoapp/yao/agent/store/types"
)

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

// GetChat get the chat info and its history
func (conv *Xun) GetChat(sid string, cid string, locale ...string) (*types.ChatInfo, error) {
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

	return &types.ChatInfo{
		Chat:    chat,
		History: history,
	}, nil
}

// GetChatWithFilter get the chat info and its history with filter options
func (conv *Xun) GetChatWithFilter(sid string, cid string, filter types.ChatFilter, locale ...string) (*types.ChatInfo, error) {
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

	return &types.ChatInfo{
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

// GetChats get the chat list with grouping by date
func (conv *Xun) GetChats(sid string, filter types.ChatFilter, locale ...string) (*types.ChatGroupResponse, error) {
	// Default behavior: exclude silent chats
	if filter.Silent == nil {
		silentFalse := false
		filter.Silent = &silentFalse
	}

	return conv.getChatsWithFilter(sid, filter, locale...)
}

// getChatsWithFilter get the chats with filter options
func (conv *Xun) getChatsWithFilter(sid string, filter types.ChatFilter, locale ...string) (*types.ChatGroupResponse, error) {
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
	result := []types.ChatGroup{}
	for _, label := range []string{"Today", "Yesterday", "This Week", "Last Week", "Even Earlier"} {
		if len(groups[label]) > 0 {
			translatedLabel := label
			if len(locale) > 0 {
				lang := strings.ToLower(locale[0])
				translatedLabel = i18n.TranslateGlobal(lang, label).(string)
			}
			result = append(result, types.ChatGroup{
				Label: translatedLabel,
				Chats: groups[label],
			})
		}
	}

	return &types.ChatGroupResponse{
		Groups:   result,
		Page:     filter.Page,
		PageSize: filter.PageSize,
		Total:    total,
		LastPage: lastPage,
	}, nil
}
