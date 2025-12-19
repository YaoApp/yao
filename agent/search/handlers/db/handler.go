package db

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/yaoapp/gou/model"
	"github.com/yaoapp/gou/query"
	"github.com/yaoapp/gou/query/gou"
	agentContext "github.com/yaoapp/yao/agent/context"
	"github.com/yaoapp/yao/agent/search/nlp/querydsl"
	"github.com/yaoapp/yao/agent/search/types"
)

// Handler implements DB search
type Handler struct {
	usesQueryDSL string          // "builtin", "<assistant-id>", "mcp:<server>.<tool>"
	config       *types.DBConfig // DB search configuration
}

// NewHandler creates a new DB search handler
func NewHandler(usesQueryDSL string, cfg *types.DBConfig) *Handler {
	return &Handler{usesQueryDSL: usesQueryDSL, config: cfg}
}

// Type returns the search type this handler supports
func (h *Handler) Type() types.SearchType {
	return types.SearchTypeDB
}

// Search converts NL to QueryDSL and executes
// Note: This method doesn't have context, use SearchWithContext for full functionality
func (h *Handler) Search(req *types.Request) (*types.Result, error) {
	return h.SearchWithContext(nil, req)
}

// SearchWithContext executes DB search with context (required for QueryDSL generation)
func (h *Handler) SearchWithContext(ctx *agentContext.Context, req *types.Request) (*types.Result, error) {
	start := time.Now()

	// Validate request
	if req.Query == "" {
		return &types.Result{
			Type:     types.SearchTypeDB,
			Query:    req.Query,
			Source:   req.Source,
			Items:    []*types.ResultItem{},
			Total:    0,
			Duration: time.Since(start).Milliseconds(),
			Error:    "query is required",
		}, nil
	}

	// Get models from request or config
	modelIDs := req.Models
	if len(modelIDs) == 0 && h.config != nil {
		modelIDs = h.config.Models
	}

	// If no models specified, return empty result
	if len(modelIDs) == 0 {
		return &types.Result{
			Type:     types.SearchTypeDB,
			Query:    req.Query,
			Source:   req.Source,
			Items:    []*types.ResultItem{},
			Total:    0,
			Duration: time.Since(start).Milliseconds(),
			Error:    "no models specified",
		}, nil
	}

	// Get max results
	maxResults := req.Limit
	if maxResults == 0 && h.config != nil && h.config.MaxResults > 0 {
		maxResults = h.config.MaxResults
	}
	if maxResults == 0 {
		maxResults = 20 // default
	}

	// Context is required for QueryDSL generation
	if ctx == nil {
		return &types.Result{
			Type:     types.SearchTypeDB,
			Query:    req.Query,
			Source:   req.Source,
			Items:    []*types.ResultItem{},
			Total:    0,
			Duration: time.Since(start).Milliseconds(),
			Error:    "context is required for DB search",
		}, nil
	}

	// 1. Load all models and build combined schema
	models := make(map[string]*model.Model)
	schemas := make([]map[string]interface{}, 0, len(modelIDs))

	for _, modelID := range modelIDs {
		mod, err := model.Get(modelID)
		if err != nil {
			continue // Skip non-existent models
		}
		models[modelID] = mod
		schemas = append(schemas, h.buildModelSchema(mod))
	}

	if len(schemas) == 0 {
		return &types.Result{
			Type:     types.SearchTypeDB,
			Query:    req.Query,
			Source:   req.Source,
			Items:    []*types.ResultItem{},
			Total:    0,
			Duration: time.Since(start).Milliseconds(),
			Error:    "no valid models found",
		}, nil
	}

	// 2. Generate QueryDSL with all schemas
	generator := querydsl.NewGenerator(h.usesQueryDSL, nil)
	input := &querydsl.Input{
		Query:    req.Query,
		ModelIDs: modelIDs,
		Scenario: req.Scenario, // Pass scenario: filter, aggregation, join, complex
		Limit:    maxResults,
	}

	// Build schema input: single schema or array of schemas
	var schemaInput interface{}
	if len(schemas) == 1 {
		schemaInput = schemas[0]
	} else {
		schemaInput = schemas
	}

	input.ExtraParams = map[string]interface{}{
		"schema": schemaInput,
	}

	result, err := generator.Generate(ctx, input)
	if err != nil {
		return &types.Result{
			Type:     types.SearchTypeDB,
			Query:    req.Query,
			Source:   req.Source,
			Items:    []*types.ResultItem{},
			Total:    0,
			Duration: time.Since(start).Milliseconds(),
			Error:    fmt.Sprintf("QueryDSL generation failed: %v", err),
		}, nil
	}

	if result == nil || result.DSL == nil {
		return &types.Result{
			Type:     types.SearchTypeDB,
			Query:    req.Query,
			Source:   req.Source,
			Items:    []*types.ResultItem{},
			Total:    0,
			Duration: time.Since(start).Milliseconds(),
			Error:    "no QueryDSL generated",
		}, nil
	}

	// 3. Merge preset conditions into generated DSL
	h.mergeDSLConditions(result.DSL, req)

	// 4. Execute QueryDSL using gou query engine
	records, err := h.executeDSL(result.DSL)
	if err != nil {
		return &types.Result{
			Type:     types.SearchTypeDB,
			Query:    req.Query,
			Source:   req.Source,
			Items:    []*types.ResultItem{},
			Total:    0,
			Duration: time.Since(start).Milliseconds(),
			Error:    fmt.Sprintf("query execution failed: %v", err),
		}, nil
	}

	// 5. Determine the primary model for result formatting
	// Use the "from" table from DSL, or first model
	primaryModelID := modelIDs[0]
	if result.DSL.From != nil && result.DSL.From.Name != "" {
		// Find model by table name
		for id, mod := range models {
			if mod.MetaData.Table.Name == result.DSL.From.Name {
				primaryModelID = id
				break
			}
		}
	}

	primaryModel := models[primaryModelID]
	if primaryModel == nil {
		primaryModel, _ = model.Get(primaryModelID) // May be nil, that's ok
	}

	// 6. Convert records to ResultItems
	items := h.convertToResultItems(records, primaryModelID, primaryModel, req.Source)

	// Apply limit
	if len(items) > maxResults {
		items = items[:maxResults]
	}

	// 7. Convert DSL to map for storage
	dslMap := h.dslToMap(result.DSL)

	return &types.Result{
		Type:     types.SearchTypeDB,
		Query:    req.Query,
		Source:   req.Source,
		Items:    items,
		Total:    len(items),
		Duration: time.Since(start).Milliseconds(),
		DSL:      dslMap,
	}, nil
}

// mergeDSLConditions merges preset conditions from request into generated DSL
func (h *Handler) mergeDSLConditions(dsl *gou.QueryDSL, req *types.Request) {
	if dsl == nil {
		return
	}

	// Merge preset Wheres (prepend to ensure they take priority)
	if len(req.Wheres) > 0 {
		dsl.Wheres = append(req.Wheres, dsl.Wheres...)
	}

	// Merge preset Orders (prepend to ensure they take priority)
	if len(req.Orders) > 0 {
		dsl.Orders = append(req.Orders, dsl.Orders...)
	}

	// Merge preset Select fields
	if len(req.Select) > 0 {
		// Convert string fields to Expression
		selectExprs := make([]gou.Expression, 0, len(req.Select))
		for _, field := range req.Select {
			selectExprs = append(selectExprs, gou.Expression{Field: field})
		}
		// If DSL has no select, use preset; otherwise merge
		if len(dsl.Select) == 0 {
			dsl.Select = selectExprs
		} else {
			// Prepend preset fields
			dsl.Select = append(selectExprs, dsl.Select...)
		}
	}

	// Ensure limit is set
	if dsl.Limit == 0 && req.Limit > 0 {
		dsl.Limit = req.Limit
	}
}

// buildModelSchema builds a simplified schema for QueryDSL generator
func (h *Handler) buildModelSchema(mod *model.Model) map[string]interface{} {
	columns := make([]map[string]interface{}, 0, len(mod.Columns))
	for _, col := range mod.Columns {
		colInfo := map[string]interface{}{
			"name": col.Name,
			"type": col.Type,
		}
		if col.Label != "" {
			colInfo["label"] = col.Label
		}
		if col.Description != "" {
			colInfo["description"] = col.Description
		}
		columns = append(columns, colInfo)
	}

	return map[string]interface{}{
		"name":    mod.MetaData.Table.Name,
		"columns": columns,
	}
}

// executeDSL executes the QueryDSL and returns records
func (h *Handler) executeDSL(dsl interface{}) ([]map[string]interface{}, error) {
	// Get the default query engine
	engine, err := query.Select("default")
	if err != nil {
		return nil, fmt.Errorf("query engine not found: %w", err)
	}

	// Marshal DSL to JSON
	dslJSON, err := json.Marshal(dsl)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal DSL: %w", err)
	}

	// Load and execute the query
	q, err := engine.Load(json.RawMessage(dslJSON))
	if err != nil {
		return nil, fmt.Errorf("failed to load DSL: %w", err)
	}

	// Execute query
	rawRecords := q.Get(nil)

	// Convert to map[string]interface{}
	records := make([]map[string]interface{}, 0, len(rawRecords))
	for _, rec := range rawRecords {
		records = append(records, map[string]interface{}(rec))
	}

	return records, nil
}

// convertToResultItems converts query results to ResultItems
func (h *Handler) convertToResultItems(records []map[string]interface{}, modelID string, mod *model.Model, source types.SourceType) []*types.ResultItem {
	items := make([]*types.ResultItem, 0, len(records))

	primaryKey := "id"
	if mod != nil && mod.PrimaryKey != "" {
		primaryKey = mod.PrimaryKey
	}

	for _, rec := range records {
		item := &types.ResultItem{
			Type:   types.SearchTypeDB,
			Source: source,
			Model:  modelID,
			Data:   rec,
		}

		// Try to extract title from common fields
		item.Title = h.extractTitle(rec, mod)

		// Try to extract content/description
		item.Content = h.extractContent(rec, mod)

		// Try to extract record ID
		if id, ok := rec[primaryKey]; ok {
			item.RecordID = id
		}

		items = append(items, item)
	}

	return items
}

// extractTitle tries to extract a title from the record
func (h *Handler) extractTitle(rec map[string]interface{}, mod *model.Model) string {
	// Common title fields
	titleFields := []string{"title", "name", "subject", "label"}
	for _, field := range titleFields {
		if val, ok := rec[field]; ok {
			if str, ok := val.(string); ok && str != "" {
				return str
			}
		}
	}
	return ""
}

// extractContent tries to extract content from the record
func (h *Handler) extractContent(rec map[string]interface{}, mod *model.Model) string {
	// Common content fields
	contentFields := []string{"content", "description", "summary", "text", "body"}
	for _, field := range contentFields {
		if val, ok := rec[field]; ok {
			if str, ok := val.(string); ok && str != "" {
				return str
			}
		}
	}

	// Fallback: serialize first few fields as content
	content, _ := json.Marshal(rec)
	if len(content) > 500 {
		content = content[:500]
	}
	return string(content)
}

// dslToMap converts QueryDSL to map for storage
func (h *Handler) dslToMap(dsl *gou.QueryDSL) map[string]interface{} {
	if dsl == nil {
		return nil
	}

	// Marshal and unmarshal to get a clean map
	data, err := json.Marshal(dsl)
	if err != nil {
		return nil
	}

	var result map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		return nil
	}

	return result
}
