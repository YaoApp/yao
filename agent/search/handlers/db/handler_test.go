package db

import (
	"testing"

	"github.com/stretchr/testify/assert"
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

func TestHandler_Search(t *testing.T) {
	tests := []struct {
		name         string
		usesQueryDSL string
		config       *types.DBConfig
		req          *types.Request
		expectError  string
		expectItems  int
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
			expectItems: 0,
		},
		{
			name:         "no models in request or config",
			usesQueryDSL: "builtin",
			config:       nil,
			req: &types.Request{
				Type:  types.SearchTypeDB,
				Query: "find products under $100",
			},
			expectError: "",
			expectItems: 0,
		},
		{
			name:         "models from config",
			usesQueryDSL: "builtin",
			config: &types.DBConfig{
				Models:     []string{"product"},
				MaxResults: 20,
			},
			req: &types.Request{
				Type:  types.SearchTypeDB,
				Query: "find products under $100",
			},
			expectError: "",
			expectItems: 0, // skeleton returns empty
		},
		{
			name:         "models from request",
			usesQueryDSL: "builtin",
			config:       nil,
			req: &types.Request{
				Type:   types.SearchTypeDB,
				Query:  "find products under $100",
				Models: []string{"product", "order"},
			},
			expectError: "",
			expectItems: 0, // skeleton returns empty
		},
		{
			name:         "with limit",
			usesQueryDSL: "builtin",
			config: &types.DBConfig{
				Models: []string{"product"},
			},
			req: &types.Request{
				Type:   types.SearchTypeDB,
				Query:  "find products",
				Models: []string{"product"},
				Limit:  5,
			},
			expectError: "",
			expectItems: 0, // skeleton returns empty
		},
		{
			name:         "with wheres",
			usesQueryDSL: "builtin",
			config: &types.DBConfig{
				Models: []string{"product"},
			},
			req: &types.Request{
				Type:   types.SearchTypeDB,
				Query:  "find products",
				Models: []string{"product"},
				// Wheres would be set here in real usage
			},
			expectError: "",
			expectItems: 0, // skeleton returns empty
		},
		{
			name:         "agent mode",
			usesQueryDSL: "workers.nlp.querydsl",
			config: &types.DBConfig{
				Models: []string{"product"},
			},
			req: &types.Request{
				Type:   types.SearchTypeDB,
				Query:  "find products",
				Models: []string{"product"},
			},
			expectError: "",
			expectItems: 0, // skeleton returns empty
		},
		{
			name:         "mcp mode",
			usesQueryDSL: "mcp:nlp.generate_querydsl",
			config: &types.DBConfig{
				Models: []string{"product"},
			},
			req: &types.Request{
				Type:   types.SearchTypeDB,
				Query:  "find products",
				Models: []string{"product"},
			},
			expectError: "",
			expectItems: 0, // skeleton returns empty
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := NewHandler(tt.usesQueryDSL, tt.config)
			result, err := h.Search(tt.req)

			assert.NoError(t, err)
			assert.NotNil(t, result)
			assert.Equal(t, types.SearchTypeDB, result.Type)
			assert.Equal(t, tt.req.Query, result.Query)
			assert.Equal(t, tt.expectItems, len(result.Items))

			if tt.expectError != "" {
				assert.Equal(t, tt.expectError, result.Error)
			} else {
				assert.Empty(t, result.Error)
			}

			// Duration should be set
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

func TestHandler_Search_MaxResultsFromConfig(t *testing.T) {
	cfg := &types.DBConfig{
		Models:     []string{"product"},
		MaxResults: 50,
	}
	h := NewHandler("builtin", cfg)

	req := &types.Request{
		Type:   types.SearchTypeDB,
		Query:  "test",
		Models: []string{"product"},
		// No limit in request, should use config's MaxResults
	}
	result, err := h.Search(req)
	assert.NoError(t, err)
	assert.NotNil(t, result)
	// Skeleton doesn't actually use maxResults yet, but the test ensures the handler runs
}
