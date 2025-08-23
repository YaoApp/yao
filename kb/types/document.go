package types

import (
	"fmt"

	"github.com/yaoapp/gou/model"
	"github.com/yaoapp/kun/maps"
)

// SearchDocuments searches documents with pagination
func (c *Config) SearchDocuments(param model.QueryParam, page int, pagesize int) (maps.MapStr, error) {
	modelName := c.DocumentModel
	if modelName == "" {
		modelName = "__yao.kb.document"
	}

	mod := model.Select(modelName)
	if mod == nil {
		return nil, fmt.Errorf("document model not found: %s", modelName)
	}
	return mod.Paginate(param, page, pagesize)
}

// FindDocument finds a single document by document_id
func (c *Config) FindDocument(documentID string, param model.QueryParam) (maps.MapStr, error) {
	modelName := c.DocumentModel
	if modelName == "" {
		modelName = "__yao.kb.document"
	}

	mod := model.Select(modelName)
	if mod == nil {
		return nil, fmt.Errorf("document model not found: %s", modelName)
	}

	param.Wheres = append(param.Wheres, model.QueryWhere{
		Column: "document_id",
		Value:  documentID,
	})
	param.Limit = 1

	res, err := mod.Get(param)
	if err != nil {
		return nil, err
	}
	if len(res) == 0 {
		return nil, fmt.Errorf("document not found: %s", documentID)
	}
	return res[0], nil
}

// CreateDocument creates a new document record
func (c *Config) CreateDocument(data maps.MapStrAny) (int, error) {
	modelName := c.DocumentModel
	if modelName == "" {
		modelName = "__yao.kb.document"
	}

	mod := model.Select(modelName)
	if mod == nil {
		return 0, fmt.Errorf("document model not found: %s", modelName)
	}
	return mod.Create(data)
}

// UpdateDocument updates a document by document_id
func (c *Config) UpdateDocument(documentID string, data maps.MapStrAny) error {
	modelName := c.DocumentModel
	if modelName == "" {
		modelName = "__yao.kb.document"
	}

	mod := model.Select(modelName)
	if mod == nil {
		return fmt.Errorf("document model not found: %s", modelName)
	}

	param := model.QueryParam{
		Wheres: []model.QueryWhere{
			{Column: "document_id", Value: documentID},
		},
		Limit: 1,
	}

	_, err := mod.UpdateWhere(param, data)
	return err
}

// RemoveDocument removes a document by document_id
func (c *Config) RemoveDocument(documentID string) error {
	modelName := c.DocumentModel
	if modelName == "" {
		modelName = "__yao.kb.document"
	}

	mod := model.Select(modelName)
	if mod == nil {
		return fmt.Errorf("document model not found: %s", modelName)
	}

	param := model.QueryParam{
		Wheres: []model.QueryWhere{
			{Column: "document_id", Value: documentID},
		},
		Limit: 1,
	}

	_, err := mod.DeleteWhere(param)
	return err
}
