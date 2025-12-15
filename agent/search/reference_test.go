package search

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/yaoapp/yao/agent/search/types"
)

func TestBuildReferences(t *testing.T) {
	tests := []struct {
		name     string
		results  []*types.Result
		expected int
	}{
		{
			name:     "nil results",
			results:  nil,
			expected: 0,
		},
		{
			name:     "empty results",
			results:  []*types.Result{},
			expected: 0,
		},
		{
			name: "single result with items",
			results: []*types.Result{
				{
					Type:  types.SearchTypeWeb,
					Query: "test query",
					Items: []*types.ResultItem{
						{
							CitationID: "1",
							Type:       types.SearchTypeWeb,
							Source:     types.SourceAuto,
							Weight:     0.6,
							Score:      0.9,
							Title:      "Test Title",
							Content:    "Test content",
							URL:        "https://example.com",
						},
						{
							CitationID: "2",
							Type:       types.SearchTypeWeb,
							Source:     types.SourceAuto,
							Weight:     0.6,
							Score:      0.8,
							Title:      "Test Title 2",
							Content:    "Test content 2",
							URL:        "https://example2.com",
						},
					},
				},
			},
			expected: 2,
		},
		{
			name: "multiple results",
			results: []*types.Result{
				{
					Type: types.SearchTypeWeb,
					Items: []*types.ResultItem{
						{CitationID: "1", Type: types.SearchTypeWeb, Content: "Web content"},
					},
				},
				{
					Type: types.SearchTypeKB,
					Items: []*types.ResultItem{
						{CitationID: "2", Type: types.SearchTypeKB, Content: "KB content"},
					},
				},
				{
					Type: types.SearchTypeDB,
					Items: []*types.ResultItem{
						{CitationID: "3", Type: types.SearchTypeDB, Content: "DB content"},
					},
				},
			},
			expected: 3,
		},
		{
			name: "result with nil items",
			results: []*types.Result{
				{
					Type: types.SearchTypeWeb,
					Items: []*types.ResultItem{
						{CitationID: "1", Content: "Content 1"},
						nil,
						{CitationID: "2", Content: "Content 2"},
					},
				},
			},
			expected: 2,
		},
		{
			name: "nil result in slice",
			results: []*types.Result{
				{
					Type: types.SearchTypeWeb,
					Items: []*types.ResultItem{
						{CitationID: "1", Content: "Content"},
					},
				},
				nil,
				{
					Type: types.SearchTypeKB,
					Items: []*types.ResultItem{
						{CitationID: "2", Content: "Content 2"},
					},
				},
			},
			expected: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			refs := BuildReferences(tt.results)
			assert.Equal(t, tt.expected, len(refs))
		})
	}
}

func TestBuildReferences_FieldMapping(t *testing.T) {
	item := &types.ResultItem{
		CitationID: "1",
		Type:       types.SearchTypeWeb,
		Source:     types.SourceHook,
		Weight:     0.8,
		Score:      0.95,
		Title:      "Test Title",
		Content:    "Test Content",
		URL:        "https://example.com",
	}

	results := []*types.Result{
		{Items: []*types.ResultItem{item}},
	}

	refs := BuildReferences(results)
	assert.Equal(t, 1, len(refs))

	ref := refs[0]
	assert.Equal(t, "1", ref.ID)
	assert.Equal(t, types.SearchTypeWeb, ref.Type)
	assert.Equal(t, types.SourceHook, ref.Source)
	assert.Equal(t, 0.8, ref.Weight)
	assert.Equal(t, 0.95, ref.Score)
	assert.Equal(t, "Test Title", ref.Title)
	assert.Equal(t, "Test Content", ref.Content)
	assert.Equal(t, "https://example.com", ref.URL)
}

func TestFormatReferencesXML(t *testing.T) {
	tests := []struct {
		name     string
		refs     []*types.Reference
		contains []string
		excludes []string
	}{
		{
			name:     "nil refs",
			refs:     nil,
			contains: []string{},
			excludes: []string{"<references>"},
		},
		{
			name:     "empty refs",
			refs:     []*types.Reference{},
			contains: []string{},
			excludes: []string{"<references>"},
		},
		{
			name: "single ref with all fields",
			refs: []*types.Reference{
				{
					ID:      "1",
					Type:    types.SearchTypeWeb,
					Source:  types.SourceUser,
					Weight:  1.0,
					Score:   0.9,
					Title:   "Test Title",
					Content: "Test Content",
					URL:     "https://example.com",
				},
			},
			contains: []string{
				"<references>",
				"</references>",
				`<ref id="1" type="web" weight="1.0" source="user">`,
				"</ref>",
				"Test Title",
				"Test Content",
				"URL: https://example.com",
			},
		},
		{
			name: "ref without title",
			refs: []*types.Reference{
				{
					ID:      "1",
					Type:    types.SearchTypeKB,
					Source:  types.SourceHook,
					Weight:  0.8,
					Content: "Content without title",
				},
			},
			contains: []string{
				`<ref id="1" type="kb" weight="0.8" source="hook">`,
				"Content without title",
			},
			excludes: []string{
				"URL:",
			},
		},
		{
			name: "ref without URL",
			refs: []*types.Reference{
				{
					ID:      "1",
					Type:    types.SearchTypeDB,
					Source:  types.SourceAuto,
					Weight:  0.6,
					Title:   "DB Record",
					Content: "Database content",
				},
			},
			contains: []string{
				`<ref id="1" type="db" weight="0.6" source="auto">`,
				"DB Record",
				"Database content",
			},
			excludes: []string{
				"URL:",
			},
		},
		{
			name: "multiple refs",
			refs: []*types.Reference{
				{ID: "1", Type: types.SearchTypeWeb, Source: types.SourceUser, Weight: 1.0, Content: "Content 1"},
				{ID: "2", Type: types.SearchTypeKB, Source: types.SourceHook, Weight: 0.8, Content: "Content 2"},
				{ID: "3", Type: types.SearchTypeDB, Source: types.SourceAuto, Weight: 0.6, Content: "Content 3"},
			},
			contains: []string{
				"<references>",
				"</references>",
				`id="1"`,
				`id="2"`,
				`id="3"`,
				"Content 1",
				"Content 2",
				"Content 3",
			},
		},
		{
			name: "nil ref in slice",
			refs: []*types.Reference{
				{ID: "1", Type: types.SearchTypeWeb, Weight: 1.0, Content: "Content 1"},
				nil,
				{ID: "2", Type: types.SearchTypeKB, Weight: 0.8, Content: "Content 2"},
			},
			contains: []string{
				`id="1"`,
				`id="2"`,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			xml := FormatReferencesXML(tt.refs)

			for _, s := range tt.contains {
				assert.Contains(t, xml, s, "expected XML to contain: %s", s)
			}

			for _, s := range tt.excludes {
				assert.NotContains(t, xml, s, "expected XML to not contain: %s", s)
			}
		})
	}
}

func TestFormatReferencesXML_Structure(t *testing.T) {
	refs := []*types.Reference{
		{
			ID:      "1",
			Type:    types.SearchTypeWeb,
			Source:  types.SourceUser,
			Weight:  1.0,
			Title:   "Title",
			Content: "Content",
			URL:     "https://example.com",
		},
	}

	xml := FormatReferencesXML(refs)

	// Check structure
	assert.True(t, strings.HasPrefix(xml, "<references>\n"))
	assert.True(t, strings.HasSuffix(xml, "</references>"))
	assert.Contains(t, xml, "</ref>\n")
}

func TestGetCitationPrompt(t *testing.T) {
	tests := []struct {
		name     string
		cfg      *types.CitationConfig
		expected string
	}{
		{
			name:     "nil config",
			cfg:      nil,
			expected: DefaultCitationPrompt,
		},
		{
			name:     "empty config",
			cfg:      &types.CitationConfig{},
			expected: DefaultCitationPrompt,
		},
		{
			name: "config with custom prompt",
			cfg: &types.CitationConfig{
				CustomPrompt: "Custom citation instructions",
			},
			expected: "Custom citation instructions",
		},
		{
			name: "config with empty custom prompt",
			cfg: &types.CitationConfig{
				CustomPrompt: "",
			},
			expected: DefaultCitationPrompt,
		},
		{
			name: "config with format but no custom prompt",
			cfg: &types.CitationConfig{
				Format: "[{id}]",
			},
			expected: DefaultCitationPrompt,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			prompt := GetCitationPrompt(tt.cfg)
			assert.Equal(t, tt.expected, prompt)
		})
	}
}

func TestDefaultCitationPrompt(t *testing.T) {
	// Verify default prompt contains key instructions
	assert.Contains(t, DefaultCitationPrompt, "<references>")
	assert.Contains(t, DefaultCitationPrompt, "id: Citation identifier")
	assert.Contains(t, DefaultCitationPrompt, "type: Data type")
	assert.Contains(t, DefaultCitationPrompt, "weight: Relevance weight")
	assert.Contains(t, DefaultCitationPrompt, "source: Origin")
	assert.Contains(t, DefaultCitationPrompt, `<a class="ref"`)
	assert.Contains(t, DefaultCitationPrompt, "data-ref-id")
	assert.Contains(t, DefaultCitationPrompt, "data-ref-type")
	// Verify example uses simple integer ID
	assert.Contains(t, DefaultCitationPrompt, `data-ref-id="1"`)
}

func TestBuildReferenceContext(t *testing.T) {
	results := []*types.Result{
		{
			Type: types.SearchTypeWeb,
			Items: []*types.ResultItem{
				{
					CitationID: "1",
					Type:       types.SearchTypeWeb,
					Source:     types.SourceAuto,
					Weight:     0.6,
					Title:      "Test",
					Content:    "Content",
					URL:        "https://example.com",
				},
			},
		},
	}

	t.Run("with nil config", func(t *testing.T) {
		ctx := BuildReferenceContext(results, nil)

		assert.NotNil(t, ctx)
		assert.Equal(t, 1, len(ctx.References))
		assert.Contains(t, ctx.XML, "<references>")
		assert.Contains(t, ctx.XML, `id="1"`)
		assert.Equal(t, DefaultCitationPrompt, ctx.Prompt)
	})

	t.Run("with custom prompt config", func(t *testing.T) {
		cfg := &types.CitationConfig{
			CustomPrompt: "Custom prompt",
		}
		ctx := BuildReferenceContext(results, cfg)

		assert.NotNil(t, ctx)
		assert.Equal(t, "Custom prompt", ctx.Prompt)
	})

	t.Run("with empty results", func(t *testing.T) {
		ctx := BuildReferenceContext([]*types.Result{}, nil)

		assert.NotNil(t, ctx)
		assert.Equal(t, 0, len(ctx.References))
		assert.Equal(t, "", ctx.XML)
		assert.Equal(t, DefaultCitationPrompt, ctx.Prompt)
	})
}

func TestBuildReferenceContext_Integration(t *testing.T) {
	// Simulate a real-world scenario with multiple search types
	results := []*types.Result{
		{
			Type:  types.SearchTypeWeb,
			Query: "AI developments",
			Items: []*types.ResultItem{
				{
					CitationID: "1",
					Type:       types.SearchTypeWeb,
					Source:     types.SourceAuto,
					Weight:     0.6,
					Score:      0.95,
					Title:      "OpenAI Announces GPT-5",
					Content:    "OpenAI has announced the development of GPT-5...",
					URL:        "https://news.example.com/gpt5",
				},
			},
		},
		{
			Type:  types.SearchTypeKB,
			Query: "AI developments",
			Items: []*types.ResultItem{
				{
					CitationID: "2",
					Type:       types.SearchTypeKB,
					Source:     types.SourceHook,
					Weight:     0.8,
					Score:      0.88,
					Title:      "Internal AI Research Notes",
					Content:    "Our internal research on AI capabilities...",
				},
			},
		},
		{
			Type:  types.SearchTypeDB,
			Query: "AI developments",
			Items: []*types.ResultItem{
				{
					CitationID: "3",
					Type:       types.SearchTypeDB,
					Source:     types.SourceUser,
					Weight:     1.0,
					Score:      0.92,
					Title:      "Product: AI Assistant",
					Content:    "Name: AI Assistant\nPrice: $99\nCategory: Software",
				},
			},
		},
	}

	ctx := BuildReferenceContext(results, nil)

	// Verify all references are included
	assert.Equal(t, 3, len(ctx.References))

	// Verify XML contains all references
	assert.Contains(t, ctx.XML, `id="1"`)
	assert.Contains(t, ctx.XML, `id="2"`)
	assert.Contains(t, ctx.XML, `id="3"`)

	// Verify different source types are represented
	assert.Contains(t, ctx.XML, `source="auto"`)
	assert.Contains(t, ctx.XML, `source="hook"`)
	assert.Contains(t, ctx.XML, `source="user"`)

	// Verify different search types are represented
	assert.Contains(t, ctx.XML, `type="web"`)
	assert.Contains(t, ctx.XML, `type="kb"`)
	assert.Contains(t, ctx.XML, `type="db"`)
}
