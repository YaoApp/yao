package api

import (
	"context"
	"fmt"
	"sort"
	"sync"

	graphragtypes "github.com/yaoapp/gou/graphrag/types"
	"github.com/yaoapp/kun/log"
	"github.com/yaoapp/yao/kb/providers/factory"
)

// Default search parameters
const (
	DefaultSearchK        = 10
	DefaultMaxDepth       = 2
	DefaultThreshold      = 0.0
	MaxSearchK            = 100
	DefaultSearchPageSize = 20
)

// Search performs batch search operations on the knowledge base
// Queries can span multiple collections; implementation groups by CollectionID
// Mode, providers (embedding/extraction/reranker) are read from each collection's config
// All results are merged and deduplicated
func (kb *KBInstance) Search(ctx context.Context, queries []Query) (*SearchResult, error) {
	if len(queries) == 0 {
		return &SearchResult{
			Segments: []graphragtypes.Segment{},
			Total:    0,
		}, nil
	}

	// 1. Validate queries
	if err := kb.validateQueries(queries); err != nil {
		return nil, err
	}

	// 2. Group queries by CollectionID
	groupedQueries := kb.groupQueriesByCollection(queries)

	// 3. Process each collection group in parallel
	var (
		allSegments []graphragtypes.Segment
		allGraph    *GraphData
		mu          sync.Mutex
		wg          sync.WaitGroup
		errChan     = make(chan error, len(groupedQueries))
	)

	for collectionID, collQueries := range groupedQueries {
		wg.Add(1)
		go func(collID string, qs []Query) {
			defer wg.Done()

			segments, graph, err := kb.searchCollection(ctx, collID, qs)
			if err != nil {
				errChan <- fmt.Errorf("search in collection %s failed: %w", collID, err)
				return
			}

			mu.Lock()
			allSegments = append(allSegments, segments...)
			if graph != nil {
				allGraph = mergeGraphData(allGraph, graph)
			}
			mu.Unlock()
		}(collectionID, collQueries)
	}

	wg.Wait()
	close(errChan)

	// Collect errors
	var errs []error
	for err := range errChan {
		errs = append(errs, err)
	}
	if len(errs) > 0 {
		// Combine all error messages
		errMsgs := make([]string, len(errs))
		for i, e := range errs {
			errMsgs[i] = e.Error()
		}
		return nil, fmt.Errorf("search failed: %v", errMsgs)
	}

	// 4. Merge and deduplicate results
	mergedSegments := kb.deduplicateSegments(allSegments)

	// 5. Sort by score (descending)
	sort.Slice(mergedSegments, func(i, j int) bool {
		return mergedSegments[i].Score > mergedSegments[j].Score
	})

	// 6. Apply pagination from first query (if specified)
	result := kb.applyPagination(mergedSegments, queries[0])
	result.Graph = allGraph

	return result, nil
}

// ========== Validation ==========

// validateQueries validates all queries
func (kb *KBInstance) validateQueries(queries []Query) error {
	for i, q := range queries {
		if q.CollectionID == "" {
			return fmt.Errorf("query %d: collection_id is required", i)
		}
		if q.Input == "" && len(q.Messages) == 0 {
			return fmt.Errorf("query %d: either input or messages is required", i)
		}
	}
	return nil
}

// ========== Query Grouping ==========

// groupQueriesByCollection groups queries by their CollectionID
func (kb *KBInstance) groupQueriesByCollection(queries []Query) map[string][]Query {
	grouped := make(map[string][]Query)
	for _, q := range queries {
		grouped[q.CollectionID] = append(grouped[q.CollectionID], q)
	}
	return grouped
}

// ========== Collection Search ==========

// searchCollection processes all queries for a single collection
func (kb *KBInstance) searchCollection(ctx context.Context, collectionID string, queries []Query) ([]graphragtypes.Segment, *GraphData, error) {
	// Get collection config
	collection, err := kb.GetCollection(ctx, collectionID)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get collection: %w", err)
	}

	// Get embedding provider from collection config
	embeddingProviderID, _ := collection["embedding_provider_id"].(string)
	embeddingOptionID, _ := collection["embedding_option_id"].(string)
	if embeddingProviderID == "" || embeddingOptionID == "" {
		return nil, nil, fmt.Errorf("collection %s missing embedding configuration", collectionID)
	}

	// Create embedding function
	embedding, err := kb.createEmbedding(embeddingProviderID, embeddingOptionID, "en")
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create embedding: %w", err)
	}

	var (
		allSegments []graphragtypes.Segment
		allGraph    *GraphData
		mu          sync.Mutex
		wg          sync.WaitGroup
		errChan     = make(chan error, len(queries))
	)

	// Process queries in parallel
	for _, query := range queries {
		wg.Add(1)
		go func(q Query) {
			defer wg.Done()

			segments, graph, err := kb.executeQuery(ctx, collectionID, q, embedding, collection)
			if err != nil {
				errChan <- err
				return
			}

			mu.Lock()
			allSegments = append(allSegments, segments...)
			if graph != nil {
				allGraph = mergeGraphData(allGraph, graph)
			}
			mu.Unlock()
		}(query)
	}

	wg.Wait()
	close(errChan)

	// Collect errors
	var errors []error
	for err := range errChan {
		errors = append(errors, err)
	}
	if len(errors) > 0 {
		return allSegments, allGraph, errors[0]
	}

	return allSegments, allGraph, nil
}

// executeQuery executes a single query based on its mode
func (kb *KBInstance) executeQuery(ctx context.Context, collectionID string, query Query, embedding graphragtypes.Embedding, collection map[string]interface{}) ([]graphragtypes.Segment, *GraphData, error) {
	// Determine search mode
	mode := query.Mode
	if mode == "" {
		// Default to expand mode
		mode = SearchModeExpand
	}

	// Get query text
	queryText := kb.getQueryText(query)
	if queryText == "" {
		return nil, nil, fmt.Errorf("no query text found")
	}

	// Execute based on mode
	switch mode {
	case SearchModeVector:
		return kb.searchVector(ctx, collectionID, queryText, query, embedding)
	case SearchModeGraph:
		return kb.searchGraph(ctx, collectionID, queryText, query, collection)
	case SearchModeExpand:
		return kb.searchExpand(ctx, collectionID, queryText, query, embedding, collection)
	default:
		return nil, nil, fmt.Errorf("unknown search mode: %s", mode)
	}
}

// getQueryText extracts query text from Input or Messages
func (kb *KBInstance) getQueryText(query Query) string {
	// Input takes precedence
	if query.Input != "" {
		return query.Input
	}

	// Extract from last user message
	for i := len(query.Messages) - 1; i >= 0; i-- {
		if query.Messages[i].Role == "user" {
			return query.Messages[i].Content
		}
	}

	return ""
}

// ========== Vector Search ==========

// searchVector performs pure vector similarity search
func (kb *KBInstance) searchVector(ctx context.Context, collectionID string, queryText string, query Query, embedding graphragtypes.Embedding) ([]graphragtypes.Segment, *GraphData, error) {
	// Build search options
	k := query.PageSize
	if k <= 0 {
		k = DefaultSearchK
	}
	if k > MaxSearchK {
		k = MaxSearchK
	}

	options := &graphragtypes.VectorSearchOptions{
		CollectionID: collectionID,
		DocumentID:   query.DocumentID,
		Query:        queryText,
		K:            k,
		MinScore:     query.Threshold,
		Embedding:    embedding,
	}

	// Add metadata filter
	if len(query.Metadata) > 0 {
		options.Filter = query.Metadata
	}

	// Execute search
	result, err := kb.GraphRag.SearchVector(ctx, options)
	if err != nil {
		return nil, nil, fmt.Errorf("vector search failed: %w", err)
	}
	return result.Segments, nil, nil
}

// ========== Graph Search ==========

// searchGraph performs pure graph traversal search
func (kb *KBInstance) searchGraph(ctx context.Context, collectionID string, queryText string, query Query, collection map[string]interface{}) ([]graphragtypes.Segment, *GraphData, error) {
	// Get extraction provider for entity extraction
	extraction, err := kb.createExtraction(collection)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create extraction: %w", err)
	}

	// Build graph search options
	maxDepth := query.MaxDepth
	if maxDepth <= 0 {
		maxDepth = DefaultMaxDepth
	}

	options := &graphragtypes.GraphSearchOptions{
		CollectionID: collectionID,
		DocumentID:   query.DocumentID,
		Query:        queryText,
		MaxDepth:     maxDepth,
		Extraction:   extraction,
	}

	// Execute search
	result, err := kb.GraphRag.SearchGraph(ctx, options)
	if err != nil {
		return nil, nil, fmt.Errorf("graph search failed: %w", err)
	}

	// Convert to GraphData
	graph := &GraphData{
		Nodes:         result.Nodes,
		Relationships: result.Relationships,
	}

	return result.Segments, graph, nil
}

// ========== Expand Search (Graph + Vector) ==========

// searchExpand performs graph-based entity expansion + vector search
// This mode uses graph to find related entities, then enhances vector search
func (kb *KBInstance) searchExpand(ctx context.Context, collectionID string, queryText string, query Query, embedding graphragtypes.Embedding, collection map[string]interface{}) ([]graphragtypes.Segment, *GraphData, error) {
	// Step 1: Extract entities from query using graph search
	extraction, err := kb.createExtraction(collection)
	if err != nil {
		// Fall back to pure vector search if extraction is not available
		log.Warn("Extraction not available, falling back to vector search: %v", err)
		return kb.searchVector(ctx, collectionID, queryText, query, embedding)
	}

	maxDepth := query.MaxDepth
	if maxDepth <= 0 {
		maxDepth = DefaultMaxDepth
	}

	graphOptions := &graphragtypes.GraphSearchOptions{
		CollectionID: collectionID,
		DocumentID:   query.DocumentID,
		Query:        queryText,
		MaxDepth:     maxDepth,
		Extraction:   extraction,
	}

	// Execute graph search to find related entities
	graphResult, graphErr := kb.GraphRag.SearchGraph(ctx, graphOptions)

	// Step 2: Perform vector search
	k := query.PageSize
	if k <= 0 {
		k = DefaultSearchK
	}
	if k > MaxSearchK {
		k = MaxSearchK
	}

	vectorOptions := &graphragtypes.VectorSearchOptions{
		CollectionID: collectionID,
		DocumentID:   query.DocumentID,
		Query:        queryText,
		K:            k,
		MinScore:     query.Threshold,
		Embedding:    embedding,
	}

	if len(query.Metadata) > 0 {
		vectorOptions.Filter = query.Metadata
	}

	vectorResult, err := kb.GraphRag.SearchVector(ctx, vectorOptions)
	if err != nil {
		return nil, nil, fmt.Errorf("vector search failed: %w", err)
	}

	// Step 3: Merge results
	segments := vectorResult.Segments

	var graph *GraphData
	if graphErr == nil && graphResult != nil {
		// Add graph segments (deduplicated later)
		segments = append(segments, graphResult.Segments...)

		// Include graph data
		graph = &GraphData{
			Nodes:         graphResult.Nodes,
			Relationships: graphResult.Relationships,
		}
	}

	return segments, graph, nil
}

// ========== Helper Functions ==========

// createEmbedding creates an embedding function from provider config
func (kb *KBInstance) createEmbedding(providerID, optionID, locale string) (graphragtypes.Embedding, error) {
	if locale == "" {
		locale = "en"
	}

	// Get provider option
	option, err := kb.getProviderOption("embedding", providerID, optionID, locale)
	if err != nil {
		return nil, fmt.Errorf("failed to get embedding option: %w", err)
	}

	// Create embedding provider
	return factory.MakeEmbedding(providerID, option)
}

// createExtraction creates an extraction function from collection config
func (kb *KBInstance) createExtraction(collection map[string]interface{}) (graphragtypes.Extraction, error) {
	// Try to get extraction provider from collection metadata
	metadata, _ := collection["metadata"].(map[string]interface{})
	if metadata == nil {
		metadata = collection
	}

	extractionProviderID, _ := metadata["__extraction_provider"].(string)
	extractionOptionID, _ := metadata["__extraction_option"].(string)

	// Fall back to default extraction provider
	if extractionProviderID == "" {
		extractionProviderID = "__yao.openai"
		extractionOptionID = "gpt-4o-mini"
	}

	// Get provider option
	option, err := kb.getProviderOption("extraction", extractionProviderID, extractionOptionID, "en")
	if err != nil {
		return nil, fmt.Errorf("failed to get extraction option: %w", err)
	}

	// Create extraction provider
	return factory.MakeExtraction(extractionProviderID, option)
}

// deduplicateSegments removes duplicate segments by ID, keeping highest score
func (kb *KBInstance) deduplicateSegments(segments []graphragtypes.Segment) []graphragtypes.Segment {
	seen := make(map[string]int) // ID -> index in result
	result := make([]graphragtypes.Segment, 0, len(segments))

	for _, seg := range segments {
		if idx, exists := seen[seg.ID]; exists {
			// Keep the one with higher score
			if seg.Score > result[idx].Score {
				result[idx] = seg
			}
		} else {
			seen[seg.ID] = len(result)
			result = append(result, seg)
		}
	}

	return result
}

// mergeGraphData merges two GraphData objects
func mergeGraphData(a, b *GraphData) *GraphData {
	if a == nil {
		return b
	}
	if b == nil {
		return a
	}

	// Merge nodes (deduplicate by ID)
	nodeMap := make(map[string]graphragtypes.GraphNode)
	for _, n := range a.Nodes {
		nodeMap[n.ID] = n
	}
	for _, n := range b.Nodes {
		nodeMap[n.ID] = n
	}

	nodes := make([]graphragtypes.GraphNode, 0, len(nodeMap))
	for _, n := range nodeMap {
		nodes = append(nodes, n)
	}

	// Merge relationships (deduplicate by ID)
	relMap := make(map[string]graphragtypes.GraphRelationship)
	for _, r := range a.Relationships {
		relMap[r.ID] = r
	}
	for _, r := range b.Relationships {
		relMap[r.ID] = r
	}

	relationships := make([]graphragtypes.GraphRelationship, 0, len(relMap))
	for _, r := range relMap {
		relationships = append(relationships, r)
	}

	return &GraphData{
		Nodes:         nodes,
		Relationships: relationships,
	}
}

// applyPagination applies pagination to segments
func (kb *KBInstance) applyPagination(segments []graphragtypes.Segment, query Query) *SearchResult {
	total := len(segments)

	// If no pagination requested, return all
	if query.Page <= 0 && query.PageSize <= 0 {
		return &SearchResult{
			Segments: segments,
			Total:    total,
		}
	}

	page := query.Page
	if page <= 0 {
		page = 1
	}

	pageSize := query.PageSize
	if pageSize <= 0 {
		pageSize = DefaultSearchPageSize
	}

	// Calculate pagination
	totalPages := (total + pageSize - 1) / pageSize
	start := (page - 1) * pageSize
	end := start + pageSize

	if start >= total {
		return &SearchResult{
			Segments:   []graphragtypes.Segment{},
			Total:      total,
			Page:       page,
			PageSize:   pageSize,
			TotalPages: totalPages,
		}
	}

	if end > total {
		end = total
	}

	result := &SearchResult{
		Segments:   segments[start:end],
		Total:      total,
		Page:       page,
		PageSize:   pageSize,
		TotalPages: totalPages,
	}

	// Set next/prev page
	if page < totalPages {
		result.Next = page + 1
	}
	if page > 1 {
		result.Prev = page - 1
	}

	return result
}
