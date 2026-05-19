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

func TestHandler_Search_Validation(t *testing.T) {
	tests := []struct {
		name        string
		config      *types.KBConfig
		req         *types.Request
		expectError string
	}{
		{
			name:   "empty query",
			config: nil,
			req: &types.Request{
				Type:  types.SearchTypeKB,
				Query: "",
			},
			expectError: "query is required",
		},
		{
			name:   "no collections - KB not initialized",
			config: nil,
			req: &types.Request{
				Type:  types.SearchTypeKB,
				Query: "test query",
			},
			expectError: "knowledge base not initialized",
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

func TestHandler_Search_CollectionsFromConfig(t *testing.T) {
	cfg := &types.KBConfig{
		Collections: []string{"docs", "faq"},
		Threshold:   0.7,
	}
	h := NewHandler(cfg)

	// Request without collections should use config collections
	req := &types.Request{
		Type:  types.SearchTypeKB,
		Query: "test query",
	}
	result, err := h.Search(req)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	// Without KB initialized, we get "knowledge base not initialized" error
	assert.Equal(t, "knowledge base not initialized", result.Error)
}

func TestHandler_Search_CollectionsFromRequest(t *testing.T) {
	h := NewHandler(nil)

	// Request with collections
	req := &types.Request{
		Type:        types.SearchTypeKB,
		Query:       "test query",
		Collections: []string{"docs", "faq"},
	}
	result, err := h.Search(req)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	// Without KB initialized, we get "knowledge base not initialized" error
	assert.Equal(t, "knowledge base not initialized", result.Error)
}

func TestHandler_Search_ThresholdHandling(t *testing.T) {
	tests := []struct {
		name            string
		configThreshold float64
		reqThreshold    float64
	}{
		{
			name:            "threshold from request",
			configThreshold: 0.7,
			reqThreshold:    0.9,
		},
		{
			name:            "threshold from config",
			configThreshold: 0.8,
			reqThreshold:    0,
		},
		{
			name:            "default threshold",
			configThreshold: 0,
			reqThreshold:    0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var cfg *types.KBConfig
			if tt.configThreshold > 0 {
				cfg = &types.KBConfig{
					Collections: []string{"docs"},
					Threshold:   tt.configThreshold,
				}
			}
			h := NewHandler(cfg)

			req := &types.Request{
				Type:        types.SearchTypeKB,
				Query:       "test query",
				Threshold:   tt.reqThreshold,
				Collections: []string{"docs"},
			}
			result, err := h.Search(req)

			assert.NoError(t, err)
			assert.NotNil(t, result)
		})
	}
}

func TestHandler_Search_GraphMode(t *testing.T) {
	tests := []struct {
		name        string
		configGraph bool
		reqGraph    bool
	}{
		{
			name:        "graph from request",
			configGraph: false,
			reqGraph:    true,
		},
		{
			name:        "graph from config",
			configGraph: true,
			reqGraph:    false,
		},
		{
			name:        "no graph",
			configGraph: false,
			reqGraph:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &types.KBConfig{
				Collections: []string{"docs"},
				Graph:       tt.configGraph,
			}
			h := NewHandler(cfg)

			req := &types.Request{
				Type:        types.SearchTypeKB,
				Query:       "test query",
				Collections: []string{"docs"},
				Graph:       tt.reqGraph,
			}
			result, err := h.Search(req)

			assert.NoError(t, err)
			assert.NotNil(t, result)
		})
	}
}

func TestHandler_Search_LimitHandling(t *testing.T) {
	h := NewHandler(&types.KBConfig{Collections: []string{"docs"}})

	tests := []struct {
		name  string
		limit int
	}{
		{
			name:  "custom limit",
			limit: 5,
		},
		{
			name:  "default limit",
			limit: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := &types.Request{
				Type:        types.SearchTypeKB,
				Query:       "test query",
				Collections: []string{"docs"},
				Limit:       tt.limit,
			}
			result, err := h.Search(req)

			assert.NoError(t, err)
			assert.NotNil(t, result)
		})
	}
}
