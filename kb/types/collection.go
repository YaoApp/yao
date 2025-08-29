package types

import (
	"fmt"

	"github.com/yaoapp/gou/model"
	"github.com/yaoapp/kun/maps"
	"github.com/yaoapp/xun/dbal"
)

// SearchCollections searches collections with pagination
func (c *Config) SearchCollections(param model.QueryParam, page int, pagesize int) (maps.MapStr, error) {
	modelName := c.CollectionModel
	if modelName == "" {
		modelName = "__yao.kb.collection"
	}

	mod := model.Select(modelName)
	if mod == nil {
		return nil, fmt.Errorf("collection model not found: %s", modelName)
	}
	return mod.Paginate(param, page, pagesize)
}

// FindCollection finds a single collection by collection_id
func (c *Config) FindCollection(collectionID string, param model.QueryParam) (maps.MapStr, error) {
	modelName := c.CollectionModel
	if modelName == "" {
		modelName = "__yao.kb.collection"
	}

	mod := model.Select(modelName)
	if mod == nil {
		return nil, fmt.Errorf("collection model not found: %s", modelName)
	}

	param.Wheres = append(param.Wheres, model.QueryWhere{
		Column: "collection_id",
		Value:  collectionID,
	})
	param.Limit = 1

	res, err := mod.Get(param)
	if err != nil {
		return nil, err
	}
	if len(res) == 0 {
		return nil, fmt.Errorf("collection not found: %s", collectionID)
	}
	return res[0], nil
}

// CreateCollection creates a new collection record
func (c *Config) CreateCollection(data maps.MapStrAny) (int, error) {
	modelName := c.CollectionModel
	if modelName == "" {
		modelName = "__yao.kb.collection"
	}

	mod := model.Select(modelName)
	if mod == nil {
		return 0, fmt.Errorf("collection model not found: %s", modelName)
	}
	return mod.Create(data)
}

// UpdateCollection updates a collection by collection_id
func (c *Config) UpdateCollection(collectionID string, data maps.MapStrAny) error {
	modelName := c.CollectionModel
	if modelName == "" {
		modelName = "__yao.kb.collection"
	}

	mod := model.Select(modelName)
	if mod == nil {
		return fmt.Errorf("collection model not found: %s", modelName)
	}

	param := model.QueryParam{
		Wheres: []model.QueryWhere{
			{Column: "collection_id", Value: collectionID},
		},
		Limit: 1,
	}

	_, err := mod.UpdateWhere(param, data)
	return err
}

// RemoveCollection removes a collection by collection_id
func (c *Config) RemoveCollection(collectionID string) error {
	modelName := c.CollectionModel
	if modelName == "" {
		modelName = "__yao.kb.collection"
	}

	mod := model.Select(modelName)
	if mod == nil {
		return fmt.Errorf("collection model not found: %s", modelName)
	}

	param := model.QueryParam{
		Wheres: []model.QueryWhere{
			{Column: "collection_id", Value: collectionID},
		},
		Limit: 1,
	}

	_, err := mod.DeleteWhere(param)
	return err
}

// DocumentCount returns the number of documents in a collection
func (c *Config) DocumentCount(collectionID string) (int, error) {
	modelName := c.DocumentModel
	if modelName == "" {
		modelName = "__yao.kb.document"
	}

	mod := model.Select(modelName)
	if mod == nil {
		return 0, fmt.Errorf("document model not found: %s", modelName)
	}

	// Use dbal.Raw to count documents in the collection
	param := model.QueryParam{
		Select: []interface{}{dbal.Raw("COUNT(*) as count")},
		Wheres: []model.QueryWhere{
			{Column: "collection_id", Value: collectionID},
		},
	}

	result, err := mod.Get(param)
	if err != nil {
		return 0, fmt.Errorf("failed to count documents: %w", err)
	}

	if len(result) == 0 {
		return 0, nil
	}

	// Extract count from result
	countValue, exists := result[0]["count"]
	if !exists {
		return 0, fmt.Errorf("count field not found in result")
	}

	// Convert to int
	switch v := countValue.(type) {
	case int:
		return v, nil
	case int64:
		return int(v), nil
	case float64:
		return int(v), nil
	default:
		return 0, fmt.Errorf("unexpected count type: %T", v)
	}
}

// UpdateDocumentCount updates the document_count field in collection metadata
func (c *Config) UpdateDocumentCount(collectionID string) error {
	// Get current document count
	count, err := c.DocumentCount(collectionID)
	if err != nil {
		return fmt.Errorf("failed to get document count: %w", err)
	}

	// Update collection metadata with the new count
	data := maps.MapStrAny{
		"document_count": count,
	}

	return c.UpdateCollection(collectionID, data)
}
