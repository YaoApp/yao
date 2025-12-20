package kb

import (
	"context"
	"fmt"
	"time"

	"github.com/yaoapp/yao/agent/search/types"
	"github.com/yaoapp/yao/kb"
	kbapi "github.com/yaoapp/yao/kb/api"
)

// Handler implements KB search using the KB API
type Handler struct {
	config *types.KBConfig // KB search configuration
}

// NewHandler creates a new KB search handler
func NewHandler(cfg *types.KBConfig) *Handler {
	return &Handler{config: cfg}
}

// Type returns the search type this handler supports
func (h *Handler) Type() types.SearchType {
	return types.SearchTypeKB
}

// Search executes vector search and optional graph association
func (h *Handler) Search(req *types.Request) (*types.Result, error) {
	start := time.Now()

	// Validate request
	if req.Query == "" {
		return &types.Result{
			Type:     types.SearchTypeKB,
			Query:    req.Query,
			Source:   req.Source,
			Items:    []*types.ResultItem{},
			Total:    0,
			Duration: time.Since(start).Milliseconds(),
			Error:    "query is required",
		}, nil
	}

	// Check if KB API is available
	if kb.API == nil {
		return &types.Result{
			Type:     types.SearchTypeKB,
			Query:    req.Query,
			Source:   req.Source,
			Items:    []*types.ResultItem{},
			Total:    0,
			Duration: time.Since(start).Milliseconds(),
			Error:    "knowledge base not initialized",
		}, nil
	}

	// Get collections from request or config
	collections := req.Collections
	if len(collections) == 0 && h.config != nil {
		collections = h.config.Collections
	}

	// If no collections specified, return empty result
	if len(collections) == 0 {
		return &types.Result{
			Type:     types.SearchTypeKB,
			Query:    req.Query,
			Source:   req.Source,
			Items:    []*types.ResultItem{},
			Total:    0,
			Duration: time.Since(start).Milliseconds(),
		}, nil
	}

	// Get threshold from request or config
	threshold := req.Threshold
	if threshold == 0 && h.config != nil && h.config.Threshold > 0 {
		threshold = h.config.Threshold
	}
	if threshold == 0 {
		threshold = 0.7 // default
	}

	// Get limit
	limit := req.Limit
	if limit == 0 {
		limit = 10 // default
	}

	// Determine search mode
	mode := kbapi.SearchModeVector
	if req.Graph {
		mode = kbapi.SearchModeExpand
	}
	if h.config != nil && h.config.Graph {
		mode = kbapi.SearchModeExpand
	}

	// Build KB API queries - one per collection
	var queries []kbapi.Query
	for _, collectionID := range collections {
		queries = append(queries, kbapi.Query{
			CollectionID: collectionID,
			Input:        req.Query,
			Mode:         mode,
			Threshold:    threshold,
			PageSize:     limit,
			Metadata:     req.Metadata,
		})
	}

	// Execute search using KB API
	ctx := context.Background()
	searchResult, err := kb.API.Search(ctx, queries)
	if err != nil {
		return &types.Result{
			Type:     types.SearchTypeKB,
			Query:    req.Query,
			Source:   req.Source,
			Items:    []*types.ResultItem{},
			Total:    0,
			Duration: time.Since(start).Milliseconds(),
			Error:    fmt.Sprintf("search failed: %v", err),
		}, nil
	}

	// Convert segments to result items
	// Note: MinScore filtering is already done by KB API, no need to filter again
	items := make([]*types.ResultItem, 0, len(searchResult.Segments))
	for _, seg := range searchResult.Segments {
		item := &types.ResultItem{
			Type:       types.SearchTypeKB,
			Source:     req.Source,
			Score:      seg.Score,
			Content:    seg.Text,
			DocumentID: seg.DocumentID,
			Collection: seg.CollectionID,
			Metadata:   seg.Metadata,
		}

		// Extract title from metadata if available
		if seg.Metadata != nil {
			if title, ok := seg.Metadata["title"].(string); ok {
				item.Title = title
			}
		}

		items = append(items, item)
	}

	// Convert graph data if available
	var graphNodes []*types.GraphNode
	if searchResult.Graph != nil {
		for _, node := range searchResult.Graph.Nodes {
			// Extract name from properties if available
			name := ""
			if node.Properties != nil {
				if n, ok := node.Properties["name"].(string); ok {
					name = n
				}
			}
			graphNodes = append(graphNodes, &types.GraphNode{
				ID:       node.ID,
				Type:     node.EntityType,
				Name:     name,
				Metadata: node.Properties,
			})
		}
	}

	result := &types.Result{
		Type:       types.SearchTypeKB,
		Query:      req.Query,
		Source:     req.Source,
		Items:      items,
		Total:      len(items),
		Duration:   time.Since(start).Milliseconds(),
		GraphNodes: graphNodes,
	}

	return result, nil
}
