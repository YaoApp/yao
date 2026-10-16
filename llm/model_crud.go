package llm

import (
	"fmt"

	goumodel "github.com/yaoapp/gou/model"
	"github.com/yaoapp/kun/maps"
)

const modelModelID = "__yao.llm.model"

func modelModel() *goumodel.Model {
	m, err := goumodel.Get(modelModelID)
	if err != nil {
		panic("llm: model model not loaded: " + err.Error())
	}
	return m
}

// CreateModel inserts a new Model and returns its auto-increment ID.
func CreateModel(m *Model) (int, error) {
	row := modelToRow(m)
	id, err := modelModel().Create(row)
	if err != nil {
		return 0, fmt.Errorf("llm: create model: %w", err)
	}
	m.ID = id
	return id, nil
}

// GetModel retrieves a Model by its auto-increment ID.
func GetModel(id int) (*Model, error) {
	row, err := modelModel().Find(id, goumodel.QueryParam{})
	if err != nil {
		return nil, fmt.Errorf("llm: get model %d: %w", id, err)
	}
	return rowToModel(row), nil
}

// UpdateModel updates an existing Model by ID.
func UpdateModel(id int, m *Model) error {
	row := modelToRow(m)
	row.Del("id")
	return modelModel().Update(id, row)
}

// DeleteModel removes a Model by ID.
func DeleteModel(id int) error {
	return modelModel().Delete(id)
}

// ListModels returns all models for a given provider.
func ListModels(providerRef int) ([]Model, error) {
	param := goumodel.QueryParam{
		Wheres: []goumodel.QueryWhere{
			{Column: "provider_ref", Value: providerRef},
		},
		Orders: []goumodel.QueryOrder{{Column: "priority", Option: "asc"}, {Column: "id", Option: "asc"}},
	}
	rows, err := modelModel().Get(param)
	if err != nil {
		return nil, fmt.Errorf("llm: list models: %w", err)
	}
	out := make([]Model, len(rows))
	for i, row := range rows {
		out[i] = *rowToModel(row)
	}
	return out, nil
}

// ListEnabledModels returns all enabled models for a given provider.
func ListEnabledModels(providerRef int) ([]Model, error) {
	param := goumodel.QueryParam{
		Wheres: []goumodel.QueryWhere{
			{Column: "provider_ref", Value: providerRef},
			{Column: "enabled", Value: true},
		},
		Orders: []goumodel.QueryOrder{{Column: "priority", Option: "asc"}, {Column: "id", Option: "asc"}},
	}
	rows, err := modelModel().Get(param)
	if err != nil {
		return nil, fmt.Errorf("llm: list enabled models: %w", err)
	}
	out := make([]Model, len(rows))
	for i, row := range rows {
		out[i] = *rowToModel(row)
	}
	return out, nil
}

// FindModelByModelID finds a model by its model_id within a provider.
func FindModelByModelID(providerRef int, modelID string) (*Model, error) {
	param := goumodel.QueryParam{
		Wheres: []goumodel.QueryWhere{
			{Column: "provider_ref", Value: providerRef},
			{Column: "model_id", Value: modelID},
		},
		Limit: 1,
	}
	rows, err := modelModel().Get(param)
	if err != nil {
		return nil, fmt.Errorf("llm: find model by model_id: %w", err)
	}
	if len(rows) == 0 {
		return nil, nil
	}
	return rowToModel(rows[0]), nil
}

// ---------------------------------------------------------------------------
// Row conversion helpers
// ---------------------------------------------------------------------------

func modelToRow(m *Model) maps.MapStrAny {
	row := maps.MapStrAny{
		"provider_ref":            m.ProviderRef,
		"model_id":                m.ModelID,
		"name":                    m.Name,
		"type":                    string(m.Type),
		"family":                  m.Family,
		"enabled":                 m.Enabled,
		"context_window":          m.ContextWindow,
		"max_output_tokens":       m.MaxOutputTokens,
		"supports_images":         m.SupportsImages,
		"supports_tool_use":       m.SupportsToolUse,
		"reasoning":               m.Reasoning,
		"audio":                   m.Audio,
		"stt":                     m.STT,
		"streaming":               m.Streaming,
		"json_mode":               m.JSON,
		"embedding":               m.Embedding,
		"image_generation":        m.ImageGeneration,
		"image_editing":           m.ImageEditing,
		"ocr":                     m.OCR,
		"temperature_adjustable":  m.TemperatureAdjustable,
		"input_cost_per_1m":       m.InputCostPer1M,
		"output_cost_per_1m":      m.OutputCostPer1M,
		"cache_read_cost_per_1m":  m.CacheReadCostPer1M,
		"cache_write_cost_per_1m": m.CacheWriteCostPer1M,
		"priority":                m.Priority,
	}
	if m.ID > 0 {
		row["id"] = m.ID
	}
	if m.ModelAPI != "" {
		row["model_api"] = m.ModelAPI
	}
	if m.MaxThinking != "" {
		row["max_thinking"] = string(m.MaxThinking)
	}
	if m.Input != nil {
		row["input"] = m.Input
	}
	if m.Output != nil {
		row["output"] = m.Output
	}
	if m.ThinkingLevelMap != nil {
		row["thinking_level_map"] = m.ThinkingLevelMap
	}
	if m.SamplingParams != nil {
		row["sampling_params"] = m.SamplingParams
	}
	if m.SamplingParamsByThinkingLevel != nil {
		row["sampling_params_by_level"] = m.SamplingParamsByThinkingLevel
	}
	if m.Capabilities != nil {
		row["capabilities"] = m.Capabilities
	}
	if m.Compat != nil {
		row["compat"] = m.Compat
	}
	if m.CostTiers != nil {
		row["cost_tiers"] = m.CostTiers
	}
	if m.InputLimits != nil {
		row["input_limits"] = m.InputLimits
	}
	if m.PromptCache != nil {
		row["prompt_cache"] = m.PromptCache
	}
	if m.Lab != "" {
		row["lab"] = m.Lab
	}
	if m.Metadata != nil {
		row["metadata"] = m.Metadata
	}
	return row
}

func rowToModel(row maps.MapStr) *Model {
	m := &Model{
		ID:                    getInt(row, "id"),
		ProviderRef:           getInt(row, "provider_ref"),
		ModelID:               getString(row, "model_id"),
		Name:                  getString(row, "name"),
		Type:                  ModelType(getString(row, "type")),
		ModelAPI:              getString(row, "model_api"),
		Family:                getString(row, "family"),
		Enabled:               getBool(row, "enabled"),
		ContextWindow:         getInt(row, "context_window"),
		MaxOutputTokens:       getInt(row, "max_output_tokens"),
		SupportsImages:        getBool(row, "supports_images"),
		SupportsToolUse:       getBool(row, "supports_tool_use"),
		Reasoning:             getBool(row, "reasoning"),
		Audio:                 getBool(row, "audio"),
		STT:                   getBool(row, "stt"),
		Streaming:             getBool(row, "streaming"),
		JSON:                  getBool(row, "json_mode"),
		Embedding:             getBool(row, "embedding"),
		ImageGeneration:       getBool(row, "image_generation"),
		ImageEditing:          getString(row, "image_editing"),
		OCR:                   getBool(row, "ocr"),
		TemperatureAdjustable: getBool(row, "temperature_adjustable"),
		MaxThinking:           ThinkingLevel(getString(row, "max_thinking")),
		InputCostPer1M:        getFloat(row, "input_cost_per_1m"),
		OutputCostPer1M:       getFloat(row, "output_cost_per_1m"),
		CacheReadCostPer1M:    getFloat(row, "cache_read_cost_per_1m"),
		CacheWriteCostPer1M:   getFloat(row, "cache_write_cost_per_1m"),
		Lab:                   getString(row, "lab"),
		Priority:              getInt(row, "priority"),
	}
	if v, ok := row["input"]; ok && v != nil {
		m.Input = toStringSliceFromAny(v)
	}
	if v, ok := row["output"]; ok && v != nil {
		m.Output = toStringSliceFromAny(v)
	}
	if v, ok := row["capabilities"]; ok && v != nil {
		m.Capabilities = toStringSliceFromAny(v)
	}
	if v, ok := row["thinking_level_map"]; ok && v != nil {
		m.ThinkingLevelMap = toThinkingLevelMap(v)
	}
	if v, ok := row["sampling_params"]; ok && v != nil {
		m.SamplingParams = toMapStringAny(v)
	}
	if v, ok := row["sampling_params_by_level"]; ok && v != nil {
		m.SamplingParamsByThinkingLevel = toSamplingParamsByLevel(v)
	}
	if v, ok := row["compat"]; ok && v != nil {
		m.Compat = toJSONMap(v)
	}
	if v, ok := row["cost_tiers"]; ok && v != nil {
		m.CostTiers = toCostTiers(v)
	}
	if v, ok := row["input_limits"]; ok && v != nil {
		m.InputLimits = toInputLimits(v)
	}
	if v, ok := row["prompt_cache"]; ok && v != nil {
		m.PromptCache = toPromptCache(v)
	}
	if v, ok := row["metadata"]; ok && v != nil {
		m.Metadata = toJSONMap(v)
	}
	m.CreatedAt = getTime(row, "created_at")
	m.UpdatedAt = getTime(row, "updated_at")
	return m
}
