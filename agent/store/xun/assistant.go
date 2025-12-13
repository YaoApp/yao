package xun

import (
	"fmt"
	"math"
	"time"

	jsoniter "github.com/json-iterator/go"
	"github.com/yaoapp/kun/log"
	"github.com/yaoapp/xun/dbal/query"
	"github.com/yaoapp/yao/agent/context"
	"github.com/yaoapp/yao/agent/i18n"
	searchTypes "github.com/yaoapp/yao/agent/search/types"
	"github.com/yaoapp/yao/agent/store/types"
)

// SaveAssistant saves assistant information
func (store *Xun) SaveAssistant(assistant *types.AssistantModel) (string, error) {
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
		assistant.ID, err = store.GenerateAssistantID()
		if err != nil {
			return "", err
		}
	}

	// Check if assistant exists
	exists, err := store.query.New().
		Table(store.getAssistantTable()).
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
	data["disable_global_prompts"] = assistant.DisableGlobalPrompts

	// Set timestamps
	now := time.Now().UnixNano()
	if exists {
		// Update: set updated_at, keep created_at unchanged
		if assistant.UpdatedAt == 0 {
			data["updated_at"] = now
		} else {
			data["updated_at"] = assistant.UpdatedAt
		}
		// Don't modify created_at on update
	} else {
		// Create: set created_at, updated_at is null
		if assistant.CreatedAt == 0 {
			data["created_at"] = now
		} else {
			data["created_at"] = assistant.CreatedAt
		}
		data["updated_at"] = nil
	}

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
	if assistant.Source != "" {
		data["source"] = assistant.Source
	} else {
		data["source"] = nil
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

	if assistant.Modes != nil {
		jsonStr, err := jsoniter.MarshalToString(assistant.Modes)
		if err != nil {
			return "", fmt.Errorf("failed to marshal modes: %w", err)
		}
		data["modes"] = jsonStr
	}

	// DefaultMode is a simple string field
	if assistant.DefaultMode != "" {
		data["default_mode"] = assistant.DefaultMode
	} else {
		data["default_mode"] = nil
	}

	// Handle interface{} fields - they should already be in the correct format
	jsonFields := map[string]interface{}{
		"prompts":           assistant.Prompts,
		"prompt_presets":    assistant.PromptPresets,
		"connector_options": assistant.ConnectorOptions,
		"kb":                assistant.KB,
		"db":                assistant.DB,
		"mcp":               assistant.MCP,
		"workflow":          assistant.Workflow,
		"placeholder":       assistant.Placeholder,
		"locales":           assistant.Locales,
		"uses":              assistant.Uses,
		"search":            assistant.Search,
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
		_, err := store.query.New().
			Table(store.getAssistantTable()).
			Where("assistant_id", assistant.ID).
			Update(data)
		if err != nil {
			return "", err
		}
		return assistant.ID, nil
	}

	err = store.query.New().
		Table(store.getAssistantTable()).
		Insert(data)
	if err != nil {
		return "", err
	}
	return assistant.ID, nil
}

// UpdateAssistant updates specific fields of an assistant
func (store *Xun) UpdateAssistant(assistantID string, updates map[string]interface{}) error {
	if assistantID == "" {
		return fmt.Errorf("assistant_id is required")
	}
	if len(updates) == 0 {
		return fmt.Errorf("no fields to update")
	}

	// Check if assistant exists
	exists, err := store.query.New().
		Table(store.getAssistantTable()).
		Where("assistant_id", assistantID).
		Exists()
	if err != nil {
		return err
	}
	if !exists {
		return fmt.Errorf("assistant %s not found", assistantID)
	}

	// Prepare update data
	data := make(map[string]interface{})

	// List of fields that need JSON marshaling
	jsonFields := []string{"options", "tags", "modes", "prompts", "prompt_presets", "connector_options", "kb", "db", "mcp", "workflow", "placeholder", "locales", "uses", "search"}
	jsonFieldSet := make(map[string]bool)
	for _, field := range jsonFields {
		jsonFieldSet[field] = true
	}

	// List of nullable string fields
	nullableStringFields := []string{"name", "avatar", "description", "path", "source", "default_mode", "__yao_created_by", "__yao_updated_by", "__yao_team_id", "__yao_tenant_id"}
	nullableFieldSet := make(map[string]bool)
	for _, field := range nullableStringFields {
		nullableFieldSet[field] = true
	}

	// Process each update field
	for key, value := range updates {
		// Skip system fields that shouldn't be updated directly
		if key == "assistant_id" || key == "created_at" {
			continue
		}

		// Handle JSON fields
		if jsonFieldSet[key] {
			if value != nil {
				jsonStr, err := jsoniter.MarshalToString(value)
				if err != nil {
					return fmt.Errorf("failed to marshal %s: %w", key, err)
				}
				data[key] = jsonStr
			} else {
				data[key] = nil
			}
		} else {
			// Handle regular fields
			// Convert empty strings to nil for nullable fields
			if strVal, ok := value.(string); ok && strVal == "" && nullableFieldSet[key] {
				data[key] = nil
				continue
			}
			data[key] = value
		}
	}

	// Always update updated_at timestamp
	data["updated_at"] = types.ToMySQLTime(time.Now().UnixNano())

	if len(data) == 0 {
		return fmt.Errorf("no valid fields to update")
	}

	// Perform update
	_, err = store.query.New().
		Table(store.getAssistantTable()).
		Where("assistant_id", assistantID).
		Update(data)

	return err
}

// DeleteAssistant deletes an assistant by assistant_id
func (store *Xun) DeleteAssistant(assistantID string) error {
	// Check if assistant exists
	exists, err := store.query.New().
		Table(store.getAssistantTable()).
		Where("assistant_id", assistantID).
		Exists()
	if err != nil {
		return err
	}

	if !exists {
		return fmt.Errorf("assistant %s not found", assistantID)
	}

	_, err = store.query.New().
		Table(store.getAssistantTable()).
		Where("assistant_id", assistantID).
		Delete()
	return err
}

// GetAssistants retrieves assistants with pagination and filtering
func (store *Xun) GetAssistants(filter types.AssistantFilter, locale ...string) (*types.AssistantList, error) {
	qb := store.query.New().
		Table(store.getAssistantTable())

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
				OrWhere("description", "like", fmt.Sprintf("%%%s%%", filter.Keywords)).
				OrWhere("locales", "like", fmt.Sprintf("%%%s%%", filter.Keywords))
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

	// Apply custom query filter function (for permission filtering)
	if filter.QueryFilter != nil {
		qb.Where(filter.QueryFilter)
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

	// Apply select fields with security validation (only if fields are explicitly specified)
	if len(filter.Select) > 0 {
		// ValidateAssistantFields will validate fields against whitelist
		sanitized := types.ValidateAssistantFields(filter.Select)
		selectFields := make([]interface{}, len(sanitized))
		for i, field := range sanitized {
			selectFields[i] = field
		}
		qb.Select(selectFields...)
	}
	// If no select fields specified, query will return all fields (SELECT *)

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
	jsonFields := []string{"tags", "options", "prompts", "prompt_presets", "connector_options", "workflow", "kb", "mcp", "placeholder", "locales", "uses", "search"}

	for _, row := range rows {
		data := row.ToMap()
		if data == nil {
			continue
		}

		// Parse JSON fields
		store.parseJSONFields(data, jsonFields)

		// Convert map to types.AssistantModel using existing helper function
		model, err := types.ToAssistantModel(data)
		if err != nil {
			log.Error("Failed to convert row to types.AssistantModel: %s", err.Error())
			continue
		}

		// Apply i18n translations if locale is provided
		if len(locale) > 0 && locale[0] != "" && model != nil {
			store.translate(model, model.ID, locale[0])
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
func (store *Xun) GetAssistant(assistantID string, fields []string, locale ...string) (*types.AssistantModel, error) {
	qb := store.query.New().
		Table(store.getAssistantTable()).
		Where("assistant_id", assistantID)

	// Apply select fields with security validation
	// If no fields specified, use default fields
	fieldsToSelect := fields
	if len(fieldsToSelect) == 0 {
		fieldsToSelect = types.AssistantDefaultFields
	}

	// ValidateAssistantFields will validate fields against whitelist
	sanitized := types.ValidateAssistantFields(fieldsToSelect)
	selectFields := make([]interface{}, len(sanitized))
	for i, field := range sanitized {
		selectFields[i] = field
	}
	qb.Select(selectFields...)

	row, err := qb.First()
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
	jsonFields := []string{"tags", "modes", "options", "prompts", "prompt_presets", "connector_options", "workflow", "kb", "db", "mcp", "placeholder", "locales", "uses", "search"}
	store.parseJSONFields(data, jsonFields)

	// Convert map to types.AssistantModel
	model := &types.AssistantModel{
		ID:                   getString(data, "assistant_id"),
		Type:                 getString(data, "type"),
		Name:                 getString(data, "name"),
		Avatar:               getString(data, "avatar"),
		Connector:            getString(data, "connector"),
		Path:                 getString(data, "path"),
		Source:               getString(data, "source"),
		BuiltIn:              getBool(data, "built_in"),
		Sort:                 getInt(data, "sort"),
		Description:          getString(data, "description"),
		DefaultMode:          getString(data, "default_mode"),
		Readonly:             getBool(data, "readonly"),
		Public:               getBool(data, "public"),
		Share:                getString(data, "share"),
		Mentionable:          getBool(data, "mentionable"),
		Automated:            getBool(data, "automated"),
		DisableGlobalPrompts: getBool(data, "disable_global_prompts"),
		CreatedAt:            getInt64(data, "created_at"),
		UpdatedAt:            getInt64(data, "updated_at"),
		YaoCreatedBy:         getString(data, "__yao_created_by"),
		YaoUpdatedBy:         getString(data, "__yao_updated_by"),
		YaoTeamID:            getString(data, "__yao_team_id"),
		YaoTenantID:          getString(data, "__yao_tenant_id"),
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

	// Handle Modes
	if modes, ok := data["modes"].([]interface{}); ok {
		model.Modes = make([]string, len(modes))
		for i, mode := range modes {
			if s, ok := mode.(string); ok {
				model.Modes[i] = s
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

	if promptPresets, has := data["prompt_presets"]; has && promptPresets != nil {
		raw, err := jsoniter.Marshal(promptPresets)
		if err == nil {
			var pp map[string][]types.Prompt
			if err := jsoniter.Unmarshal(raw, &pp); err == nil {
				model.PromptPresets = pp
			}
		}
	}

	if connectorOptions, has := data["connector_options"]; has && connectorOptions != nil {
		raw, err := jsoniter.Marshal(connectorOptions)
		if err == nil {
			var co types.ConnectorOptions
			if err := jsoniter.Unmarshal(raw, &co); err == nil {
				model.ConnectorOptions = &co
			}
		}
	}

	if kb, has := data["kb"]; has && kb != nil {
		kbConverted, err := types.ToKnowledgeBase(kb)
		if err == nil {
			model.KB = kbConverted
		}
	}

	if db, has := data["db"]; has && db != nil {
		dbConverted, err := types.ToDatabase(db)
		if err == nil {
			model.DB = dbConverted
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

	if uses, has := data["uses"]; has && uses != nil {
		raw, err := jsoniter.Marshal(uses)
		if err == nil {
			var u context.Uses
			if err := jsoniter.Unmarshal(raw, &u); err == nil {
				model.Uses = &u
			}
		}
	}

	if search, has := data["search"]; has && search != nil {
		raw, err := jsoniter.Marshal(search)
		if err == nil {
			var s searchTypes.Config
			if err := jsoniter.Unmarshal(raw, &s); err == nil {
				model.Search = &s
			}
		}
	}

	// Apply i18n translation if locale is provided
	if len(locale) > 0 && locale[0] != "" {
		store.translate(model, assistantID, locale[0])
	}

	return model, nil
}

// DeleteAssistants deletes assistants based on filter conditions
func (store *Xun) DeleteAssistants(filter types.AssistantFilter) (int64, error) {
	qb := store.query.New().
		Table(store.getAssistantTable())

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

// GetAssistantTags retrieves all unique tags from assistants with filtering
func (store *Xun) GetAssistantTags(filter types.AssistantFilter, locale ...string) ([]types.Tag, error) {
	qb := store.query.New().Table(store.getAssistantTable())

	// Apply type filter (default to "assistant")
	typeFilter := "assistant"
	if filter.Type != "" {
		typeFilter = filter.Type
	}
	qb.Where("type", typeFilter)

	// Apply custom query filter function (for permission filtering)
	if filter.QueryFilter != nil {
		qb.Where(filter.QueryFilter)
	}

	// Apply other filters if provided
	if filter.Connector != "" {
		qb.Where("connector", filter.Connector)
	}

	if filter.BuiltIn != nil {
		qb.Where("built_in", *filter.BuiltIn)
	}

	if filter.Mentionable != nil {
		qb.Where("mentionable", *filter.Mentionable)
	}

	if filter.Automated != nil {
		qb.Where("automated", *filter.Automated)
	}

	// Apply keyword filter if provided
	if filter.Keywords != "" {
		qb.Where(func(qb query.Query) {
			qb.Where("name", "like", fmt.Sprintf("%%%s%%", filter.Keywords)).
				OrWhere("description", "like", fmt.Sprintf("%%%s%%", filter.Keywords))
		})
	}

	rows, err := qb.Select("tags").GroupBy("tags").Get()
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

// translate applies i18n translation to assistant model fields
func (store *Xun) translate(model *types.AssistantModel, assistantID string, locale string) {
	if model == nil {
		return
	}

	// Translate name
	if translated := i18n.Translate(assistantID, locale, model.Name); translated != nil {
		if s, ok := translated.(string); ok {
			model.Name = s
		}
	}

	// Translate description
	if translated := i18n.Translate(assistantID, locale, model.Description); translated != nil {
		if s, ok := translated.(string); ok {
			model.Description = s
		}
	}

	// Translate prompts
	if model.Prompts != nil {
		for i := range model.Prompts {
			if translated := i18n.Translate(assistantID, locale, model.Prompts[i].Name); translated != nil {
				if s, ok := translated.(string); ok {
					model.Prompts[i].Name = s
				}
			}
			if translated := i18n.Translate(assistantID, locale, model.Prompts[i].Content); translated != nil {
				if s, ok := translated.(string); ok {
					model.Prompts[i].Content = s
				}
			}
		}
	}

	// Translate placeholder
	if model.Placeholder != nil {
		if translated := i18n.Translate(assistantID, locale, model.Placeholder.Title); translated != nil {
			if s, ok := translated.(string); ok {
				model.Placeholder.Title = s
			}
		}
		if translated := i18n.Translate(assistantID, locale, model.Placeholder.Description); translated != nil {
			if s, ok := translated.(string); ok {
				model.Placeholder.Description = s
			}
		}
		if translated := i18n.Translate(assistantID, locale, model.Placeholder.Prompts); translated != nil {
			if prompts, ok := translated.([]string); ok {
				model.Placeholder.Prompts = prompts
			}
		}
	}

	// Translate tags
	if translated := i18n.Translate(assistantID, locale, model.Tags); translated != nil {
		if tags, ok := translated.([]string); ok {
			model.Tags = tags
		}
	}
}
