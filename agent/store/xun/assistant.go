package xun

import (
	"fmt"
	"math"
	"strings"
	"time"

	jsoniter "github.com/json-iterator/go"
	"github.com/yaoapp/kun/log"
	"github.com/yaoapp/xun/dbal/query"
	"github.com/yaoapp/yao/agent/i18n"
	"github.com/yaoapp/yao/agent/store/types"
)

// SaveAssistant saves assistant information
func (conv *Xun) SaveAssistant(assistant *types.AssistantModel) (string, error) {
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
	data["connector"] = assistant.Connector
	data["built_in"] = assistant.BuiltIn
	data["sort"] = assistant.Sort
	data["readonly"] = assistant.Readonly
	data["public"] = assistant.Public
	data["mentionable"] = assistant.Mentionable
	data["automated"] = assistant.Automated
	data["created_at"] = assistant.CreatedAt
	data["updated_at"] = assistant.UpdatedAt

	// Handle nullable string fields from assistant.mod.yao
	// Store as nil if empty string (this matches database nullable: true fields)
	if assistant.Name != "" {
		data["name"] = assistant.Name
	} else {
		data["name"] = nil
	}
	if assistant.Avatar != "" {
		data["avatar"] = assistant.Avatar
	} else {
		data["avatar"] = nil
	}
	if assistant.Description != "" {
		data["description"] = assistant.Description
	} else {
		data["description"] = nil
	}
	if assistant.Path != "" {
		data["path"] = assistant.Path
	} else {
		data["path"] = nil
	}

	// Share field: nullable: false with default "private"
	// Apply default if empty
	if assistant.Share != "" {
		data["share"] = assistant.Share
	} else {
		data["share"] = "private" // Apply default value
	}

	// Permission management fields - store as nil if empty
	if assistant.YaoCreatedBy != "" {
		data["__yao_created_by"] = assistant.YaoCreatedBy
	} else {
		data["__yao_created_by"] = nil
	}
	if assistant.YaoUpdatedBy != "" {
		data["__yao_updated_by"] = assistant.YaoUpdatedBy
	} else {
		data["__yao_updated_by"] = nil
	}
	if assistant.YaoTeamID != "" {
		data["__yao_team_id"] = assistant.YaoTeamID
	} else {
		data["__yao_team_id"] = nil
	}
	if assistant.YaoTenantID != "" {
		data["__yao_tenant_id"] = assistant.YaoTenantID
	} else {
		data["__yao_tenant_id"] = nil
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
func (conv *Xun) GetAssistants(filter types.AssistantFilter, locale ...string) (*types.AssistantList, error) {
	qb := conv.query.New().
		Table(conv.getAssistantTable())

	// Apply tag filter if provided
	if len(filter.Tags) > 0 {
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
	if len(filter.AssistantIDs) > 0 {
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
	if len(filter.Select) > 0 {
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

	// Convert rows to types.AssistantModel slice
	assistants := make([]*types.AssistantModel, 0, len(rows))
	jsonFields := []string{"tags", "options", "prompts", "workflow", "kb", "mcp", "tools", "placeholder", "locales"}

	for _, row := range rows {
		data := row.ToMap()
		if data == nil {
			continue
		}

		// Parse JSON fields
		conv.parseJSONFields(data, jsonFields)

		// Convert map to types.AssistantModel using existing helper function
		model, err := types.ToAssistantModel(data)
		if err != nil {
			log.Error("Failed to convert row to types.AssistantModel: %s", err.Error())
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

	return &types.AssistantList{
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
func (conv *Xun) GetAssistant(assistantID string, locale ...string) (*types.AssistantModel, error) {
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
	if len(data) == 0 {
		return nil, fmt.Errorf("the assistant %s is empty", assistantID)
	}

	// Parse JSON fields
	jsonFields := []string{"tags", "options", "prompts", "workflow", "kb", "mcp", "tools", "placeholder", "locales"}
	conv.parseJSONFields(data, jsonFields)

	// Convert map to types.AssistantModel
	model := &types.AssistantModel{
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
			var p []types.Prompt
			if err := jsoniter.Unmarshal(raw, &p); err == nil {
				model.Prompts = p
			}
		}
	}

	if kb, has := data["kb"]; has && kb != nil {
		kbConverted, err := types.ToKnowledgeBase(kb)
		if err == nil {
			model.KB = kbConverted
		}
	}

	if mcp, has := data["mcp"]; has && mcp != nil {
		mcpConverted, err := types.ToMCPServers(mcp)
		if err == nil {
			model.MCP = mcpConverted
		}
	}

	if workflow, has := data["workflow"]; has && workflow != nil {
		wf, err := types.ToWorkflow(workflow)
		if err == nil {
			model.Workflow = wf
		}
	}

	if tools, has := data["tools"]; has && tools != nil {
		raw, err := jsoniter.Marshal(tools)
		if err == nil {
			var tc types.ToolCalls
			if err := jsoniter.Unmarshal(raw, &tc); err == nil {
				model.Tools = &tc
			}
		}
	}

	if placeholder, has := data["placeholder"]; has && placeholder != nil {
		raw, err := jsoniter.Marshal(placeholder)
		if err == nil {
			var ph types.Placeholder
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

// DeleteAssistants deletes assistants based on filter conditions
func (conv *Xun) DeleteAssistants(filter types.AssistantFilter) (int64, error) {
	qb := conv.query.New().
		Table(conv.getAssistantTable())

	// Apply tag filter if provided
	if len(filter.Tags) > 0 {
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
	if len(filter.AssistantIDs) > 0 {
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
func (conv *Xun) GetAssistantTags(locale ...string) ([]types.Tag, error) {
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
	tags := make([]types.Tag, 0, len(tagSet))
	for tag := range tagSet {
		tags = append(tags, types.Tag{
			Value: tag,
			Label: i18n.TranslateGlobal(lang, tag).(string),
		})
	}
	return tags, nil
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
