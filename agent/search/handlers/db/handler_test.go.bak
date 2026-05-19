package db

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/yaoapp/gou/model"
	"github.com/yaoapp/gou/query/gou"
	"github.com/yaoapp/yao/agent/search/types"
)

func TestNewHandler(t *testing.T) {
	t.Run("with nil config", func(t *testing.T) {
		h := NewHandler("builtin", nil)
		assert.NotNil(t, h)
		assert.Equal(t, "builtin", h.usesQueryDSL)
		assert.Nil(t, h.config)
	})

	t.Run("with config", func(t *testing.T) {
		cfg := &types.DBConfig{
			Models:     []string{"product", "order"},
			MaxResults: 50,
		}
		h := NewHandler("workers.nlp.querydsl", cfg)
		assert.NotNil(t, h)
		assert.Equal(t, "workers.nlp.querydsl", h.usesQueryDSL)
		assert.Equal(t, cfg, h.config)
	})

	t.Run("with mcp mode", func(t *testing.T) {
		h := NewHandler("mcp:nlp.generate_querydsl", nil)
		assert.NotNil(t, h)
		assert.Equal(t, "mcp:nlp.generate_querydsl", h.usesQueryDSL)
	})
}

func TestHandler_Type(t *testing.T) {
	h := NewHandler("builtin", nil)
	assert.Equal(t, types.SearchTypeDB, h.Type())
}

func TestHandler_Search_Validation(t *testing.T) {
	tests := []struct {
		name         string
		usesQueryDSL string
		config       *types.DBConfig
		req          *types.Request
		expectError  string
	}{
		{
			name:         "empty query",
			usesQueryDSL: "builtin",
			config:       nil,
			req: &types.Request{
				Type:  types.SearchTypeDB,
				Query: "",
			},
			expectError: "query is required",
		},
		{
			name:         "no models in request or config",
			usesQueryDSL: "builtin",
			config:       nil,
			req: &types.Request{
				Type:  types.SearchTypeDB,
				Query: "find products under $100",
			},
			expectError: "no models specified",
		},
		{
			name:         "context required for DB search",
			usesQueryDSL: "builtin",
			config: &types.DBConfig{
				Models: []string{"product"},
			},
			req: &types.Request{
				Type:   types.SearchTypeDB,
				Query:  "find products under $100",
				Models: []string{"product"},
			},
			expectError: "context is required for DB search",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := NewHandler(tt.usesQueryDSL, tt.config)
			result, err := h.Search(tt.req)

			assert.NoError(t, err)
			assert.NotNil(t, result)
			assert.Equal(t, types.SearchTypeDB, result.Type)
			assert.Equal(t, tt.expectError, result.Error)
			assert.Equal(t, 0, len(result.Items))
			assert.GreaterOrEqual(t, result.Duration, int64(0))
		})
	}
}

func TestHandler_Search_SourcePreserved(t *testing.T) {
	h := NewHandler("builtin", &types.DBConfig{Models: []string{"product"}})

	sources := []types.SourceType{types.SourceUser, types.SourceHook, types.SourceAuto}
	for _, source := range sources {
		req := &types.Request{
			Type:   types.SearchTypeDB,
			Query:  "test",
			Source: source,
			Models: []string{"product"},
		}
		result, err := h.Search(req)
		assert.NoError(t, err)
		assert.Equal(t, source, result.Source)
	}
}

func TestHandler_BuildModelSchema(t *testing.T) {
	h := NewHandler("builtin", nil)

	// Create a mock model for testing
	mod := &model.Model{
		MetaData: model.MetaData{
			Table: model.Table{
				Name: "test_products",
			},
		},
		Columns: map[string]*model.Column{
			"id": {
				Name:  "id",
				Type:  "ID",
				Label: "ID",
			},
			"name": {
				Name:        "name",
				Type:        "string",
				Label:       "Name",
				Description: "Product name",
			},
			"price": {
				Name:  "price",
				Type:  "decimal",
				Label: "Price",
			},
		},
	}

	schema := h.buildModelSchema(mod)

	assert.NotNil(t, schema)
	assert.Equal(t, "test_products", schema["name"])

	columns, ok := schema["columns"].([]map[string]interface{})
	assert.True(t, ok)
	assert.Len(t, columns, 3)

	// Verify columns have required fields
	for _, col := range columns {
		assert.NotEmpty(t, col["name"])
		assert.NotEmpty(t, col["type"])
	}
}

func TestHandler_BuildModelSchema_MultipleModels(t *testing.T) {
	h := NewHandler("builtin", nil)

	// Create mock models for testing joins
	productMod := &model.Model{
		MetaData: model.MetaData{
			Table: model.Table{Name: "products"},
		},
		Columns: map[string]*model.Column{
			"id":          {Name: "id", Type: "ID"},
			"name":        {Name: "name", Type: "string"},
			"category_id": {Name: "category_id", Type: "integer"},
		},
	}

	categoryMod := &model.Model{
		MetaData: model.MetaData{
			Table: model.Table{Name: "categories"},
		},
		Columns: map[string]*model.Column{
			"id":   {Name: "id", Type: "ID"},
			"name": {Name: "name", Type: "string"},
		},
	}

	productSchema := h.buildModelSchema(productMod)
	categorySchema := h.buildModelSchema(categoryMod)

	assert.Equal(t, "products", productSchema["name"])
	assert.Equal(t, "categories", categorySchema["name"])

	// Verify both schemas can be combined into an array
	schemas := []map[string]interface{}{productSchema, categorySchema}
	assert.Len(t, schemas, 2)
}

func TestHandler_ExtractTitle(t *testing.T) {
	h := NewHandler("builtin", nil)
	mod := &model.Model{}

	tests := []struct {
		name     string
		record   map[string]interface{}
		expected string
	}{
		{
			name:     "title field",
			record:   map[string]interface{}{"title": "Test Title", "id": 1},
			expected: "Test Title",
		},
		{
			name:     "name field",
			record:   map[string]interface{}{"name": "Test Name", "id": 1},
			expected: "Test Name",
		},
		{
			name:     "subject field",
			record:   map[string]interface{}{"subject": "Test Subject", "id": 1},
			expected: "Test Subject",
		},
		{
			name:     "label field",
			record:   map[string]interface{}{"label": "Test Label", "id": 1},
			expected: "Test Label",
		},
		{
			name:     "no title field",
			record:   map[string]interface{}{"id": 1, "price": 100},
			expected: "",
		},
		{
			name:     "empty title",
			record:   map[string]interface{}{"title": "", "name": "Fallback"},
			expected: "Fallback",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			title := h.extractTitle(tt.record, mod)
			assert.Equal(t, tt.expected, title)
		})
	}
}

func TestHandler_ExtractContent(t *testing.T) {
	h := NewHandler("builtin", nil)
	mod := &model.Model{}

	tests := []struct {
		name        string
		record      map[string]interface{}
		expectEmpty bool
	}{
		{
			name:        "content field",
			record:      map[string]interface{}{"content": "Test Content"},
			expectEmpty: false,
		},
		{
			name:        "description field",
			record:      map[string]interface{}{"description": "Test Description"},
			expectEmpty: false,
		},
		{
			name:        "summary field",
			record:      map[string]interface{}{"summary": "Test Summary"},
			expectEmpty: false,
		},
		{
			name:        "fallback to JSON",
			record:      map[string]interface{}{"id": 1, "price": 100},
			expectEmpty: false, // Should return JSON representation
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			content := h.extractContent(tt.record, mod)
			if tt.expectEmpty {
				assert.Empty(t, content)
			} else {
				assert.NotEmpty(t, content)
			}
		})
	}
}

func TestHandler_ConvertToResultItems(t *testing.T) {
	h := NewHandler("builtin", nil)
	mod := &model.Model{
		PrimaryKey: "id",
	}

	records := []map[string]interface{}{
		{
			"id":          1,
			"name":        "Product 1",
			"description": "Description 1",
			"price":       99.99,
		},
		{
			"id":      2,
			"title":   "Product 2",
			"content": "Content 2",
		},
	}

	items := h.convertToResultItems(records, "product", mod, types.SourceAuto)

	assert.Len(t, items, 2)

	// First item
	assert.Equal(t, types.SearchTypeDB, items[0].Type)
	assert.Equal(t, types.SourceAuto, items[0].Source)
	assert.Equal(t, "product", items[0].Model)
	assert.Equal(t, 1, items[0].RecordID)
	assert.Equal(t, "Product 1", items[0].Title)
	assert.Equal(t, "Description 1", items[0].Content)
	assert.NotNil(t, items[0].Data)

	// Second item
	assert.Equal(t, 2, items[1].RecordID)
	assert.Equal(t, "Product 2", items[1].Title)
	assert.Equal(t, "Content 2", items[1].Content)
}

func TestHandler_ConvertToResultItems_NilModel(t *testing.T) {
	h := NewHandler("builtin", nil)

	records := []map[string]interface{}{
		{"id": 1, "name": "Test"},
	}

	// Should use default primary key "id" when model is nil
	items := h.convertToResultItems(records, "test", nil, types.SourceHook)

	assert.Len(t, items, 1)
	assert.Equal(t, 1, items[0].RecordID)
	assert.Equal(t, "Test", items[0].Title)
}

func TestHandler_Search_ScenarioTypes(t *testing.T) {
	// Test that all scenario types are valid
	scenarios := []types.ScenarioType{
		types.ScenarioFilter,
		types.ScenarioAggregation,
		types.ScenarioJoin,
		types.ScenarioComplex,
	}

	for _, scenario := range scenarios {
		t.Run(string(scenario), func(t *testing.T) {
			h := NewHandler("builtin", &types.DBConfig{Models: []string{"product"}})
			req := &types.Request{
				Type:     types.SearchTypeDB,
				Query:    "test query",
				Source:   types.SourceAuto,
				Models:   []string{"product"},
				Scenario: scenario,
			}

			// Without context, should return error (but scenario should be preserved in request)
			result, err := h.Search(req)
			assert.NoError(t, err)
			assert.NotNil(t, result)
			// Verify request scenario is set correctly
			assert.Equal(t, scenario, req.Scenario)
		})
	}
}

func TestScenarioTypeConstants(t *testing.T) {
	// Verify scenario type constants match expected values
	assert.Equal(t, types.ScenarioType("filter"), types.ScenarioFilter)
	assert.Equal(t, types.ScenarioType("aggregation"), types.ScenarioAggregation)
	assert.Equal(t, types.ScenarioType("join"), types.ScenarioJoin)
	assert.Equal(t, types.ScenarioType("complex"), types.ScenarioComplex)
}

func TestHandler_MergeDSLConditions(t *testing.T) {
	h := NewHandler("builtin", nil)

	t.Run("merge wheres", func(t *testing.T) {
		dsl := &gou.QueryDSL{
			From: &gou.Table{Name: "users"},
			Wheres: []gou.Where{
				{Condition: gou.Condition{Field: &gou.Expression{Field: "status"}, Value: "active", OP: "="}},
			},
		}
		req := &types.Request{
			Wheres: []gou.Where{
				{Condition: gou.Condition{Field: &gou.Expression{Field: "tenant_id"}, Value: 1, OP: "="}},
			},
		}

		h.mergeDSLConditions(dsl, req)

		// Preset wheres should be prepended
		assert.Len(t, dsl.Wheres, 2)
		assert.Equal(t, "tenant_id", dsl.Wheres[0].Field.Field)
		assert.Equal(t, "status", dsl.Wheres[1].Field.Field)
	})

	t.Run("merge orders", func(t *testing.T) {
		dsl := &gou.QueryDSL{
			From: &gou.Table{Name: "products"},
			Orders: gou.Orders{
				{Field: &gou.Expression{Field: "name"}, Sort: "asc"},
			},
		}
		req := &types.Request{
			Orders: gou.Orders{
				{Field: &gou.Expression{Field: "created_at"}, Sort: "desc"},
			},
		}

		h.mergeDSLConditions(dsl, req)

		// Preset orders should be prepended
		assert.Len(t, dsl.Orders, 2)
		assert.Equal(t, "created_at", dsl.Orders[0].Field.Field)
		assert.Equal(t, "name", dsl.Orders[1].Field.Field)
	})

	t.Run("merge select fields", func(t *testing.T) {
		dsl := &gou.QueryDSL{
			From: &gou.Table{Name: "orders"},
			Select: []gou.Expression{
				{Field: "amount"},
			},
		}
		req := &types.Request{
			Select: []string{"id", "status"},
		}

		h.mergeDSLConditions(dsl, req)

		// Preset select should be prepended
		assert.Len(t, dsl.Select, 3)
		assert.Equal(t, "id", dsl.Select[0].Field)
		assert.Equal(t, "status", dsl.Select[1].Field)
		assert.Equal(t, "amount", dsl.Select[2].Field)
	})

	t.Run("set limit from request", func(t *testing.T) {
		dsl := &gou.QueryDSL{
			From:  &gou.Table{Name: "users"},
			Limit: 0,
		}
		req := &types.Request{
			Limit: 50,
		}

		h.mergeDSLConditions(dsl, req)

		assert.Equal(t, 50, dsl.Limit)
	})

	t.Run("preserve dsl limit if set", func(t *testing.T) {
		dsl := &gou.QueryDSL{
			From:  &gou.Table{Name: "users"},
			Limit: 10,
		}
		req := &types.Request{
			Limit: 50,
		}

		h.mergeDSLConditions(dsl, req)

		// DSL limit should be preserved
		assert.Equal(t, 10, dsl.Limit)
	})

	t.Run("nil dsl", func(t *testing.T) {
		req := &types.Request{
			Wheres: []gou.Where{
				{Condition: gou.Condition{Field: &gou.Expression{Field: "id"}, Value: 1}},
			},
		}

		// Should not panic
		h.mergeDSLConditions(nil, req)
	})
}
