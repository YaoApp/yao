package search

import (
	"sync"

	"github.com/yaoapp/yao/agent/search/handlers/db"
	"github.com/yaoapp/yao/agent/search/handlers/kb"
	"github.com/yaoapp/yao/agent/search/handlers/web"
	"github.com/yaoapp/yao/agent/search/interfaces"
	"github.com/yaoapp/yao/agent/search/types"
)

// Searcher is the main search implementation
type Searcher struct {
	config   *types.Config // Merged config (global + assistant)
	handlers map[types.SearchType]interfaces.Handler
	reranker interfaces.Reranker
	citation *CitationGenerator
}

// Uses contains the search-specific uses configuration
// These are extracted from context.Uses and search config
type Uses struct {
	Search   string // "builtin", "disabled", "<assistant-id>", "mcp:<server>.<tool>"
	Web      string // "builtin", "<assistant-id>", "mcp:<server>.<tool>"
	Keyword  string // "builtin", "<assistant-id>", "mcp:<server>.<tool>"
	QueryDSL string // "builtin", "<assistant-id>", "mcp:<server>.<tool>"
	Rerank   string // "builtin", "<assistant-id>", "mcp:<server>.<tool>"
}

// New creates a new Searcher instance
// cfg: merged config from agent/load.go + assistant config
// uses: merged uses configuration (global → assistant → hook)
func New(cfg *types.Config, uses *Uses) *Searcher {
	if uses == nil {
		uses = &Uses{}
	}
	if cfg == nil {
		cfg = &types.Config{}
	}

	return &Searcher{
		config: cfg,
		handlers: map[types.SearchType]interfaces.Handler{
			types.SearchTypeWeb: web.NewHandler(uses.Web, cfg.Web),
			types.SearchTypeKB:  kb.NewHandler(cfg.KB),
			types.SearchTypeDB:  db.NewHandler(uses.QueryDSL, cfg.DB),
		},
		reranker: newBuiltinReranker(), // TODO: use uses.Rerank to select reranker
		citation: NewCitationGenerator(),
	}
}

// Search executes a single search request
func (s *Searcher) Search(req *types.Request) (*types.Result, error) {
	handler, ok := s.handlers[req.Type]
	if !ok {
		return &types.Result{Error: "unsupported search type"}, nil
	}

	// Execute search
	result, err := handler.Search(req)
	if err != nil {
		return &types.Result{Error: err.Error()}, nil
	}

	// Assign weights based on source
	for _, item := range result.Items {
		item.Weight = s.config.GetWeight(req.Source)
	}

	// Rerank if requested
	if req.Rerank != nil && s.reranker != nil {
		result.Items, _ = s.reranker.Rerank(req.Query, result.Items, req.Rerank)
	}

	// Generate citation IDs
	for _, item := range result.Items {
		item.CitationID = s.citation.Next()
	}

	return result, nil
}

// SearchMultiple executes multiple searches in parallel
func (s *Searcher) SearchMultiple(reqs []*types.Request) ([]*types.Result, error) {
	results := make([]*types.Result, len(reqs))
	var wg sync.WaitGroup
	var mu sync.Mutex

	for i, req := range reqs {
		wg.Add(1)
		go func(idx int, r *types.Request) {
			defer wg.Done()
			result, _ := s.Search(r)
			mu.Lock()
			results[idx] = result
			mu.Unlock()
		}(i, req)
	}

	wg.Wait()
	return results, nil
}

// BuildReferences converts search results to unified Reference format
func (s *Searcher) BuildReferences(results []*types.Result) []*types.Reference {
	return BuildReferences(results)
}

// builtinReranker is a simple score-based reranker
type builtinReranker struct{}

func newBuiltinReranker() *builtinReranker {
	return &builtinReranker{}
}

func (r *builtinReranker) Rerank(query string, items []*types.ResultItem, opts *types.RerankOptions) ([]*types.ResultItem, error) {
	// Simple implementation: sort by score (already sorted in most cases)
	// TODO: Implement proper reranking logic
	if opts != nil && opts.TopN > 0 && opts.TopN < len(items) {
		return items[:opts.TopN], nil
	}
	return items, nil
}
