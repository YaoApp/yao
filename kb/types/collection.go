package types

import (
	"fmt"

	"github.com/yaoapp/gou/model"
	"github.com/yaoapp/kun/maps"
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
