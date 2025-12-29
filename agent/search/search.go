package search

import (
	"sync"

	"github.com/yaoapp/yao/agent/context"
	"github.com/yaoapp/yao/agent/search/handlers/db"
	"github.com/yaoapp/yao/agent/search/handlers/kb"
	"github.com/yaoapp/yao/agent/search/handlers/web"
	"github.com/yaoapp/yao/agent/search/interfaces"
	"github.com/yaoapp/yao/agent/search/rerank"
	"github.com/yaoapp/yao/agent/search/types"
)

// Searcher is the main search implementation
type Searcher struct {
	config   *types.Config // Merged config (global + assistant)
	handlers map[types.SearchType]interfaces.Handler
	reranker *rerank.Reranker
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
		reranker: rerank.NewReranker(uses.Rerank, cfg.Rerank),
		citation: NewCitationGenerator(),
	}
}

// Search executes a single search request
func (s *Searcher) Search(ctx *context.Context, req *types.Request) (*types.Result, error) {
	handler, ok := s.handlers[req.Type]
	if !ok {
		return &types.Result{Error: "unsupported search type"}, nil
	}

	// Execute search - use context if handler supports it
	var result *types.Result
	var err error
	if ctxHandler, ok := handler.(interfaces.ContextHandler); ok {
		result, err = ctxHandler.SearchWithContext(ctx, req)
	} else {
		result, err = handler.Search(req)
	}
	if err != nil {
		return &types.Result{Error: err.Error()}, nil
	}

	// Assign weights based on source
	for _, item := range result.Items {
		item.Weight = s.config.GetWeight(req.Source)
	}

	// Rerank if requested
	if req.Rerank != nil && s.reranker != nil {
		result.Items, _ = s.reranker.Rerank(ctx, req.Query, result.Items, req.Rerank)
	}

	// Generate citation IDs
	for _, item := range result.Items {
		item.CitationID = s.citation.Next()
	}

	return result, nil
}

// All executes all searches and waits for all to complete (like Promise.all)
func (s *Searcher) All(ctx *context.Context, reqs []*types.Request) ([]*types.Result, error) {
	if len(reqs) == 0 {
		return []*types.Result{}, nil
	}
	return s.parallelAll(ctx, reqs)
}

// Any returns as soon as any search succeeds with results (like Promise.any)
func (s *Searcher) Any(ctx *context.Context, reqs []*types.Request) ([]*types.Result, error) {
	if len(reqs) == 0 {
		return []*types.Result{}, nil
	}
	return s.parallelAny(ctx, reqs)
}

// Race returns as soon as any search completes (like Promise.race)
func (s *Searcher) Race(ctx *context.Context, reqs []*types.Request) ([]*types.Result, error) {
	if len(reqs) == 0 {
		return []*types.Result{}, nil
	}
	return s.parallelRace(ctx, reqs)
}

// parallelAll executes all searches and waits for all to complete (like Promise.all)
func (s *Searcher) parallelAll(ctx *context.Context, reqs []*types.Request) ([]*types.Result, error) {
	results := make([]*types.Result, len(reqs))
	var wg sync.WaitGroup
	var mu sync.Mutex

	for i, req := range reqs {
		wg.Add(1)
		go func(idx int, r *types.Request) {
			defer wg.Done()
			defer func() {
				if err := recover(); err != nil {
					mu.Lock()
					results[idx] = &types.Result{Error: "search panic recovered"}
					mu.Unlock()
				}
			}()

			result, err := s.Search(ctx, r)
			mu.Lock()
			if err != nil {
				results[idx] = &types.Result{Error: err.Error()}
			} else if result == nil {
				results[idx] = &types.Result{Error: "empty result"}
			} else {
				results[idx] = result
			}
			mu.Unlock()
		}(i, req)
	}

	wg.Wait()
	return results, nil
}

// parallelAny returns as soon as any search succeeds (has results) (like Promise.any)
// Other searches continue in background but results are discarded
func (s *Searcher) parallelAny(ctx *context.Context, reqs []*types.Request) ([]*types.Result, error) {
	results := make([]*types.Result, len(reqs))
	resultChan := make(chan struct {
		idx    int
		result *types.Result
	}, len(reqs))

	var wg sync.WaitGroup
	done := make(chan struct{})

	for i, req := range reqs {
		wg.Add(1)
		go func(idx int, r *types.Request) {
			defer wg.Done()

			// Check if done before starting
			select {
			case <-done:
				return
			default:
			}

			result, _ := s.Search(ctx, r)

			// Try to send result
			select {
			case <-done:
				// Already found a successful result
			case resultChan <- struct {
				idx    int
				result *types.Result
			}{idx, result}:
			}
		}(i, req)
	}

	// Close channel when all goroutines complete
	go func() {
		wg.Wait()
		close(resultChan)
	}()

	// Collect results until we find one with items (success)
	var foundSuccess bool
	for res := range resultChan {
		results[res.idx] = res.result
		// Check if this result has items (success = has results and no error)
		if !foundSuccess && res.result != nil && len(res.result.Items) > 0 && res.result.Error == "" {
			foundSuccess = true
			close(done) // Signal other goroutines to stop
		}
	}

	// All goroutines have completed (resultChan is closed)
	return results, nil
}

// parallelRace returns as soon as any search completes (like Promise.race)
// Returns immediately when first result arrives, regardless of success/failure
// Note: Still waits for all goroutines to complete before returning to avoid resource leaks
func (s *Searcher) parallelRace(ctx *context.Context, reqs []*types.Request) ([]*types.Result, error) {
	results := make([]*types.Result, len(reqs))
	resultChan := make(chan struct {
		idx    int
		result *types.Result
	}, len(reqs))

	var wg sync.WaitGroup
	done := make(chan struct{})

	for i, req := range reqs {
		wg.Add(1)
		go func(idx int, r *types.Request) {
			defer wg.Done()

			// Check if done before starting
			select {
			case <-done:
				return
			default:
			}

			result, _ := s.Search(ctx, r)

			// Try to send result
			select {
			case <-done:
				// Already got first result
			case resultChan <- struct {
				idx    int
				result *types.Result
			}{idx, result}:
			}
		}(i, req)
	}

	// Close channel when all goroutines complete
	go func() {
		wg.Wait()
		close(resultChan)
	}()

	// Get first result and signal others to stop
	var gotFirst bool
	for res := range resultChan {
		results[res.idx] = res.result
		if !gotFirst {
			gotFirst = true
			close(done) // Signal other goroutines to stop
		}
	}

	// All goroutines have completed (resultChan is closed)
	return results, nil
}

// BuildReferences converts search results to unified Reference format
func (s *Searcher) BuildReferences(results []*types.Result) []*types.Reference {
	return BuildReferences(results)
}
