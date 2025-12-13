package kb

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/yaoapp/yao/agent/search/types"
)

func TestNewHandler(t *testing.T) {
	t.Run("with nil config", func(t *testing.T) {
		h := NewHandler(nil)
		assert.NotNil(t, h)
		assert.Nil(t, h.config)
	})

	t.Run("with config", func(t *testing.T) {
		cfg := &types.KBConfig{
			Collections: []string{"docs", "faq"},
			Threshold:   0.8,
			Graph:       true,
		}
		h := NewHandler(cfg)
		assert.NotNil(t, h)
		assert.Equal(t, cfg, h.config)
	})
}

func TestHandler_Type(t *testing.T) {
	h := NewHandler(nil)
	assert.Equal(t, types.SearchTypeKB, h.Type())
}

func TestHandler_Search(t *testing.T) {
	tests := []struct {
		name        string
		config      *types.KBConfig
		req         *types.Request
		expectError string
		expectItems int
	}{
		{
			name:   "empty query",
			config: nil,
			req: &types.Request{
				Type:  types.SearchTypeKB,
				Query: "",
			},
			expectError: "query is required",
			expectItems: 0,
		},
		{
			name:   "no collections in request or config",
			config: nil,
			req: &types.Request{
				Type:  types.SearchTypeKB,
				Query: "test query",
			},
			expectError: "",
			expectItems: 0,
		},
		{
			name: "collections from config",
			config: &types.KBConfig{
				Collections: []string{"docs"},
				Threshold:   0.7,
			},
			req: &types.Request{
				Type:  types.SearchTypeKB,
				Query: "test query",
			},
			expectError: "",
			expectItems: 0, // skeleton returns empty
		},
		{
			name:   "collections from request",
			config: nil,
			req: &types.Request{
				Type:        types.SearchTypeKB,
				Query:       "test query",
				Collections: []string{"docs", "faq"},
			},
			expectError: "",
			expectItems: 0, // skeleton returns empty
		},
		{
			name: "with threshold from request",
			config: &types.KBConfig{
				Collections: []string{"docs"},
				Threshold:   0.7,
			},
			req: &types.Request{
				Type:        types.SearchTypeKB,
				Query:       "test query",
				Threshold:   0.9,
				Collections: []string{"docs"},
			},
			expectError: "",
			expectItems: 0, // skeleton returns empty
		},
		{
			name: "with graph enabled",
			config: &types.KBConfig{
				Collections: []string{"docs"},
				Graph:       true,
			},
			req: &types.Request{
				Type:        types.SearchTypeKB,
				Query:       "test query",
				Collections: []string{"docs"},
				Graph:       true,
			},
			expectError: "",
			expectItems: 0, // skeleton returns empty
		},
		{
			name: "with limit",
			config: &types.KBConfig{
				Collections: []string{"docs"},
			},
			req: &types.Request{
				Type:        types.SearchTypeKB,
				Query:       "test query",
				Collections: []string{"docs"},
				Limit:       5,
			},
			expectError: "",
			expectItems: 0, // skeleton returns empty
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := NewHandler(tt.config)
			result, err := h.Search(tt.req)

			assert.NoError(t, err)
			assert.NotNil(t, result)
			assert.Equal(t, types.SearchTypeKB, result.Type)
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
	h := NewHandler(&types.KBConfig{Collections: []string{"docs"}})

	sources := []types.SourceType{types.SourceUser, types.SourceHook, types.SourceAuto}
	for _, source := range sources {
		req := &types.Request{
			Type:        types.SearchTypeKB,
			Query:       "test",
			Source:      source,
			Collections: []string{"docs"},
		}
		result, err := h.Search(req)
		assert.NoError(t, err)
		assert.Equal(t, source, result.Source)
	}
}
