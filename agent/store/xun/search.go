package xun

import (
	"fmt"
	"time"

	jsoniter "github.com/json-iterator/go"
	"github.com/yaoapp/gou/model"
	"github.com/yaoapp/xun/dbal/query"
	"github.com/yaoapp/yao/agent/store/types"
)

// =============================================================================
// Search Management
// =============================================================================

// SaveSearch saves a search record for a request
func (store *Xun) SaveSearch(search *types.Search) error {
	if search == nil {
		return fmt.Errorf("search is nil")
	}
	if search.RequestID == "" {
		return fmt.Errorf("request_id is required")
	}
	if search.ChatID == "" {
		return fmt.Errorf("chat_id is required")
	}
	if search.Source == "" {
		return fmt.Errorf("source is required")
	}

	now := time.Now()

	// Build row data
	row := map[string]interface{}{
		"request_id": search.RequestID,
		"chat_id":    search.ChatID,
		"query":      search.Query,
		"source":     search.Source,
		"duration":   search.Duration,
		"created_at": now,
		"updated_at": now,
	}

	// Handle JSON fields
	if search.Config != nil {
		configJSON, err := jsoniter.MarshalToString(search.Config)
		if err != nil {
			return fmt.Errorf("failed to marshal config: %w", err)
		}
		row["config"] = configJSON
	}

	if len(search.Keywords) > 0 {
		keywordsJSON, err := jsoniter.MarshalToString(search.Keywords)
		if err != nil {
			return fmt.Errorf("failed to marshal keywords: %w", err)
		}
		row["keywords"] = keywordsJSON
	}

	if len(search.Entities) > 0 {
		entitiesJSON, err := jsoniter.MarshalToString(search.Entities)
		if err != nil {
			return fmt.Errorf("failed to marshal entities: %w", err)
		}
		row["entities"] = entitiesJSON
	}

	if len(search.Relations) > 0 {
		relationsJSON, err := jsoniter.MarshalToString(search.Relations)
		if err != nil {
			return fmt.Errorf("failed to marshal relations: %w", err)
		}
		row["relations"] = relationsJSON
	}

	if search.DSL != nil {
		dslJSON, err := jsoniter.MarshalToString(search.DSL)
		if err != nil {
			return fmt.Errorf("failed to marshal dsl: %w", err)
		}
		row["dsl"] = dslJSON
	}

	if len(search.References) > 0 {
		refsJSON, err := jsoniter.MarshalToString(search.References)
		if err != nil {
			return fmt.Errorf("failed to marshal references: %w", err)
		}
		row["references"] = refsJSON
	}

	if len(search.Graph) > 0 {
		graphJSON, err := jsoniter.MarshalToString(search.Graph)
		if err != nil {
			return fmt.Errorf("failed to marshal graph: %w", err)
		}
		row["graph"] = graphJSON
	}

	if search.XML != "" {
		row["xml"] = search.XML
	}

	if search.Prompt != "" {
		row["prompt"] = search.Prompt
	}

	if search.Error != "" {
		row["error"] = search.Error
	}

	return store.newQuerySearch().Insert(row)
}

// GetSearches retrieves all search records for a request
func (store *Xun) GetSearches(requestID string) ([]*types.Search, error) {
	if requestID == "" {
		return nil, fmt.Errorf("request_id is required")
	}

	rows, err := store.newQuerySearch().
		Where("request_id", requestID).
		WhereNull("deleted_at").
		OrderBy("created_at", "asc").
		Get()
	if err != nil {
		return nil, err
	}

	searches := make([]*types.Search, 0, len(rows))
	for _, row := range rows {
		data := row.ToMap()
		if data == nil {
			continue
		}

		search, err := store.rowToSearch(data)
		if err != nil {
			continue
		}
		searches = append(searches, search)
	}

	return searches, nil
}

// GetReference retrieves a single reference by request ID and index
func (store *Xun) GetReference(requestID string, index int) (*types.Reference, error) {
	if requestID == "" {
		return nil, fmt.Errorf("request_id is required")
	}
	if index < 1 {
		return nil, fmt.Errorf("index must be >= 1")
	}

	// Get all searches for this request
	searches, err := store.GetSearches(requestID)
	if err != nil {
		return nil, err
	}

	// Find the reference with matching index
	for _, search := range searches {
		for _, ref := range search.References {
			if ref.Index == index {
				return &ref, nil
			}
		}
	}

	return nil, fmt.Errorf("reference not found: request_id=%s, index=%d", requestID, index)
}

// DeleteSearches deletes all search records for a chat (soft delete)
func (store *Xun) DeleteSearches(chatID string) error {
	if chatID == "" {
		return fmt.Errorf("chat_id is required")
	}

	_, err := store.newQuerySearch().
		Where("chat_id", chatID).
		WhereNull("deleted_at").
		Update(map[string]interface{}{
			"deleted_at": time.Now(),
			"updated_at": time.Now(),
		})

	return err
}

// =============================================================================
// Query Builder
// =============================================================================

// newQuerySearch creates a new query builder for the search table
func (store *Xun) newQuerySearch() query.Query {
	qb := store.query.New()
	qb.Table(store.getSearchTable())
	return qb
}

// getSearchTable returns the search table name
func (store *Xun) getSearchTable() string {
	m := model.Select("__yao.agent.search")
	if m != nil && m.MetaData.Table.Name != "" {
		return m.MetaData.Table.Name
	}
	return "agent_search"
}

// =============================================================================
// Helper Functions
// =============================================================================

// rowToSearch converts a database row to a Search struct
func (store *Xun) rowToSearch(data map[string]interface{}) (*types.Search, error) {
	search := &types.Search{
		ID:        getInt64(data, "id"),
		RequestID: getString(data, "request_id"),
		ChatID:    getString(data, "chat_id"),
		Query:     getString(data, "query"),
		Source:    getString(data, "source"),
		XML:       getString(data, "xml"),
		Prompt:    getString(data, "prompt"),
		Duration:  getInt64(data, "duration"),
		Error:     getString(data, "error"),
	}

	// Handle timestamps
	if createdAt := getTime(data, "created_at"); createdAt != nil {
		search.CreatedAt = *createdAt
	}

	// Parse JSON fields
	if config := data["config"]; config != nil {
		if configStr, ok := config.(string); ok && configStr != "" {
			var configMap map[string]any
			if err := jsoniter.UnmarshalFromString(configStr, &configMap); err == nil {
				search.Config = configMap
			}
		}
	}

	if keywords := data["keywords"]; keywords != nil {
		if keywordsStr, ok := keywords.(string); ok && keywordsStr != "" {
			var keywordsList []string
			if err := jsoniter.UnmarshalFromString(keywordsStr, &keywordsList); err == nil {
				search.Keywords = keywordsList
			}
		}
	}

	if entities := data["entities"]; entities != nil {
		if entitiesStr, ok := entities.(string); ok && entitiesStr != "" {
			var entitiesList []types.Entity
			if err := jsoniter.UnmarshalFromString(entitiesStr, &entitiesList); err == nil {
				search.Entities = entitiesList
			}
		}
	}

	if relations := data["relations"]; relations != nil {
		if relationsStr, ok := relations.(string); ok && relationsStr != "" {
			var relationsList []types.Relation
			if err := jsoniter.UnmarshalFromString(relationsStr, &relationsList); err == nil {
				search.Relations = relationsList
			}
		}
	}

	if dsl := data["dsl"]; dsl != nil {
		if dslStr, ok := dsl.(string); ok && dslStr != "" {
			var dslMap map[string]any
			if err := jsoniter.UnmarshalFromString(dslStr, &dslMap); err == nil {
				search.DSL = dslMap
			}
		}
	}

	if refs := data["references"]; refs != nil {
		if refsStr, ok := refs.(string); ok && refsStr != "" {
			var refsList []types.Reference
			if err := jsoniter.UnmarshalFromString(refsStr, &refsList); err == nil {
				search.References = refsList
			}
		}
	}

	if graph := data["graph"]; graph != nil {
		if graphStr, ok := graph.(string); ok && graphStr != "" {
			var graphList []types.GraphNode
			if err := jsoniter.UnmarshalFromString(graphStr, &graphList); err == nil {
				search.Graph = graphList
			}
		}
	}

	return search, nil
}
