package assistant

import (
	"fmt"
	"strings"
	"time"

	"github.com/yaoapp/yao/agent/context"
	"github.com/yaoapp/yao/agent/i18n"
	"github.com/yaoapp/yao/agent/output/message"
	"github.com/yaoapp/yao/agent/search"
	"github.com/yaoapp/yao/agent/search/nlp/keyword"
	searchTypes "github.com/yaoapp/yao/agent/search/types"
	storeTypes "github.com/yaoapp/yao/agent/store/types"
	traceTypes "github.com/yaoapp/yao/trace/types"
)

// shouldAutoSearch determines if auto search should be executed
// Returns false if:
// - uses.search is "disabled"
// - assistant has no search configuration
func (ast *Assistant) shouldAutoSearch(ctx *context.Context, createResponse *context.HookCreateResponse) bool {
	// Get merged uses configuration
	uses := ast.getMergedSearchUses(createResponse)

	// Check if search is explicitly disabled
	if uses != nil && uses.Search == "disabled" {
		ctx.Logger.Info("Auto search disabled by uses.search=disabled")
		return false
	}

	// Check if assistant has search configuration
	if ast.Search == nil && (uses == nil || uses.Search == "") {
		return false
	}

	// Check if search is enabled (builtin, agent, mcp, or empty means builtin)
	return true
}

// getMergedSearchUses returns the merged uses configuration for search
// Priority: createResponse > assistant
func (ast *Assistant) getMergedSearchUses(createResponse *context.HookCreateResponse) *context.Uses {
	// Start with assistant uses
	var uses *context.Uses
	if ast.Uses != nil {
		uses = &context.Uses{
			Search:   ast.Uses.Search,
			Web:      ast.Uses.Web,
			Keyword:  ast.Uses.Keyword,
			QueryDSL: ast.Uses.QueryDSL,
			Rerank:   ast.Uses.Rerank,
		}
	}

	// Override with createResponse.Uses if provided (highest priority)
	if createResponse != nil && createResponse.Uses != nil {
		if uses == nil {
			uses = &context.Uses{}
		}
		if createResponse.Uses.Search != "" {
			uses.Search = createResponse.Uses.Search
		}
		if createResponse.Uses.Web != "" {
			uses.Web = createResponse.Uses.Web
		}
		if createResponse.Uses.Keyword != "" {
			uses.Keyword = createResponse.Uses.Keyword
		}
		if createResponse.Uses.QueryDSL != "" {
			uses.QueryDSL = createResponse.Uses.QueryDSL
		}
		if createResponse.Uses.Rerank != "" {
			uses.Rerank = createResponse.Uses.Rerank
		}
	}

	return uses
}

// executeAutoSearch executes auto search based on configuration
// Returns ReferenceContext with results and formatted context
// opts is optional, used to check Skip.Keyword
func (ast *Assistant) executeAutoSearch(ctx *context.Context, messages []context.Message, createResponse *context.HookCreateResponse, opts ...*context.Options) *searchTypes.ReferenceContext {
	ctx.Logger.Phase("Search")
	defer ctx.Logger.PhaseComplete("Search")

	// Get merged uses configuration
	uses := ast.getMergedSearchUses(createResponse)

	// Convert to search.Uses
	searchUses := &search.Uses{}
	if uses != nil {
		searchUses.Search = uses.Search
		searchUses.Web = uses.Web
		searchUses.Keyword = uses.Keyword
		searchUses.QueryDSL = uses.QueryDSL
		searchUses.Rerank = uses.Rerank
	}

	// Get merged search config
	searchConfig := ast.GetMergedSearchConfig()

	// Create searcher
	searcher := search.New(searchConfig, searchUses)

	// Extract query from messages (save original for storage)
	originalQuery := extractQueryFromMessages(messages)
	if originalQuery == "" {
		ctx.Logger.Info("No query found in messages, skipping auto search")
		return nil
	}
	query := originalQuery

	// Check if keyword extraction should be skipped
	skipKeyword := false
	if len(opts) > 0 && opts[0] != nil && opts[0].Skip != nil {
		skipKeyword = opts[0].Skip.Keyword
	}

	// Extract keywords for web search if:
	// 1. uses.keyword is configured (not empty)
	// 2. Skip.Keyword is not true
	// 3. Web search is enabled
	var extractedKeywords []string
	webSearchEnabled := searchConfig != nil && searchConfig.Web != nil
	if webSearchEnabled && !skipKeyword && searchUses.Keyword != "" {
		extractor := keyword.NewExtractor(searchUses.Keyword, searchConfig.Keyword)
		keywords, err := extractor.Extract(ctx, query, nil)
		if err != nil {
			ctx.Logger.Warn("Keyword extraction failed, using original query: %v", err)
		} else if len(keywords) > 0 {
			extractedKeywords = keywords
			// Use extracted keywords as the search query for web search
			optimizedQuery := strings.Join(keywords, " ")
			ctx.Logger.Info("Extracted keywords for web search: %s -> %s", truncateString(query, 30), optimizedQuery)
			query = optimizedQuery
		}
	}

	// Build search requests based on configuration
	requests := ast.buildSearchRequests(query, searchConfig)
	if len(requests) == 0 {
		ctx.Logger.Info("No search requests to execute")
		return nil
	}

	// === Output: Send loading message ===
	loadingID := ast.sendSearchLoading(ctx)

	// === Trace: Create search trace node ===
	searchNode := ast.createSearchTrace(ctx, query, requests)

	// Execute searches in parallel
	ctx.Logger.Info("Executing %d search requests for query: %s", len(requests), truncateString(query, 50))

	startTime := time.Now()
	results, err := searcher.All(ctx, requests)
	duration := time.Since(startTime).Milliseconds()

	if err != nil {
		// Log error but don't fail - search errors shouldn't block the main flow
		ctx.Logger.Error("Auto search failed: %v", err)

		// === Output: Send failed message ===
		ast.sendSearchDone(ctx, loadingID, 0, true)

		// === Trace: Mark as failed ===
		ast.completeSearchTrace(searchNode, 0, err)

		// === Storage: Save failed search ===
		ast.saveSearch(ctx, &SearchExecutionResult{
			Query:      originalQuery,
			Keywords:   extractedKeywords,
			Config:     ast.configToMap(searchConfig),
			Duration:   duration,
			Error:      err,
			SearchType: "auto",
		})

		return nil
	}

	// Build reference context (includes references, XML, and prompt)
	var citationConfig *searchTypes.CitationConfig
	if searchConfig != nil {
		citationConfig = searchConfig.Citation
	}
	refCtx := search.BuildReferenceContext(results, citationConfig)

	resultCount := len(refCtx.References)

	// === Output: Send result message, then done ===
	ast.sendSearchResult(ctx, loadingID, resultCount)
	ast.sendSearchDone(ctx, loadingID, resultCount, false)

	// === Trace: Mark as completed ===
	ast.completeSearchTrace(searchNode, resultCount, nil)

	// === Storage: Save successful search ===
	ast.saveSearch(ctx, &SearchExecutionResult{
		Query:      originalQuery,
		Keywords:   extractedKeywords,
		Config:     ast.configToMap(searchConfig),
		RefCtx:     refCtx,
		Duration:   duration,
		SearchType: "auto",
	})

	if resultCount == 0 {
		ctx.Logger.Info("No search results found")
		return nil
	}

	ctx.Logger.Info("Auto search completed: %d references", resultCount)
	return refCtx
}

// ============================================================================
// Output: Loading Replace Pattern
// ============================================================================

// sendSearchLoading sends the initial loading message
// Returns the message ID for later replacement
func (ast *Assistant) sendSearchLoading(ctx *context.Context) string {
	loadingMsg := i18n.T(ctx.Locale, "search.loading")

	msg := &message.Message{
		Type: "loading",
		Props: map[string]any{
			"message": loadingMsg,
		},
	}

	// Send and get message ID
	msgID, err := ctx.SendStream(msg)
	if err != nil {
		ctx.Logger.Warn("Failed to send search loading message: %v", err)
		return ""
	}

	return msgID
}

// sendSearchResult replaces loading with result message (without done flag)
func (ast *Assistant) sendSearchResult(ctx *context.Context, loadingID string, count int) {
	if loadingID == "" {
		return
	}

	var resultMsg string
	if count == 0 {
		resultMsg = i18n.T(ctx.Locale, "search.no_results")
	} else if count == 1 {
		resultMsg = i18n.T(ctx.Locale, "search.success.one")
	} else {
		resultMsg = fmt.Sprintf(i18n.T(ctx.Locale, "search.success"), count)
	}

	msg := &message.Message{
		MessageID:   loadingID,
		Delta:       true,
		DeltaAction: message.DeltaReplace,
		Type:        "loading",
		Props: map[string]any{
			"message": resultMsg,
		},
	}

	if err := ctx.Send(msg); err != nil {
		ctx.Logger.Warn("Failed to send search result message: %v", err)
	}
}

// sendSearchDone sends the final done message (removes loading indicator)
func (ast *Assistant) sendSearchDone(ctx *context.Context, loadingID string, count int, failed bool) {
	if loadingID == "" {
		return
	}

	var resultMsg string
	if failed {
		resultMsg = i18n.T(ctx.Locale, "search.failed")
	} else if count == 0 {
		resultMsg = i18n.T(ctx.Locale, "search.no_results")
	} else if count == 1 {
		resultMsg = i18n.T(ctx.Locale, "search.success.one")
	} else {
		resultMsg = fmt.Sprintf(i18n.T(ctx.Locale, "search.success"), count)
	}

	msg := &message.Message{
		MessageID:   loadingID,
		Delta:       true,
		DeltaAction: message.DeltaReplace,
		Type:        "loading",
		Props: map[string]any{
			"message": resultMsg,
			"done":    true, // Frontend will remove loading indicator
		},
	}

	if err := ctx.Send(msg); err != nil {
		ctx.Logger.Warn("Failed to send search done message: %v", err)
	}
}

// ============================================================================
// Trace: Search Node
// ============================================================================

// createSearchTrace creates a trace node for search operation
func (ast *Assistant) createSearchTrace(ctx *context.Context, query string, requests []*searchTypes.Request) traceTypes.Node {
	trace, _ := ctx.Trace()
	if trace == nil {
		return nil
	}

	// Build search types list
	var searchTypes []string
	for _, req := range requests {
		searchTypes = append(searchTypes, string(req.Type))
	}

	input := map[string]any{
		"query": query,
		"types": searchTypes,
	}

	node, err := trace.Add(input, traceTypes.TraceNodeOption{
		Label:       i18n.T(ctx.Locale, "search.trace.label"),
		Type:        "search",
		Icon:        "search",
		Description: i18n.T(ctx.Locale, "search.trace.description"),
	})

	if err != nil {
		ctx.Logger.Warn("Failed to create search trace node: %v", err)
		return nil
	}

	// Log search start
	node.Info("Starting search", map[string]any{
		"query": query,
		"types": searchTypes,
	})

	return node
}

// completeSearchTrace marks the search trace node as completed or failed
func (ast *Assistant) completeSearchTrace(node traceTypes.Node, resultCount int, err error) {
	if node == nil {
		return
	}

	if err != nil {
		node.Warn("Search failed", map[string]any{"error": err.Error()})
		node.Fail(err)
		return
	}

	// Log completion
	node.Info("Search completed", map[string]any{
		"result_count": resultCount,
	})

	// Complete with output
	node.Complete(map[string]any{
		"result_count": resultCount,
	})
}

// buildSearchRequests builds search requests based on assistant configuration
func (ast *Assistant) buildSearchRequests(query string, config *searchTypes.Config) []*searchTypes.Request {
	var requests []*searchTypes.Request

	// Web search - check if web search is configured
	if config != nil && config.Web != nil {
		requests = append(requests, &searchTypes.Request{
			Type:   searchTypes.SearchTypeWeb,
			Query:  query,
			Source: searchTypes.SourceAuto,
			Limit:  config.Web.MaxResults,
		})
	}

	// KB search - check if KB is configured
	if ast.KB != nil && len(ast.KB.Collections) > 0 {
		limit := 10
		threshold := 0.7
		if config != nil && config.KB != nil {
			if config.KB.Threshold > 0 {
				threshold = config.KB.Threshold
			}
		}
		requests = append(requests, &searchTypes.Request{
			Type:        searchTypes.SearchTypeKB,
			Query:       query,
			Source:      searchTypes.SourceAuto,
			Limit:       limit,
			Collections: ast.KB.Collections,
			Threshold:   threshold,
			Graph:       config != nil && config.KB != nil && config.KB.Graph,
		})
	}

	// DB search - check if DB is configured
	if ast.DB != nil && len(ast.DB.Models) > 0 {
		limit := 20
		if config != nil && config.DB != nil && config.DB.MaxResults > 0 {
			limit = config.DB.MaxResults
		}
		requests = append(requests, &searchTypes.Request{
			Type:   searchTypes.SearchTypeDB,
			Query:  query,
			Source: searchTypes.SourceAuto,
			Limit:  limit,
			Models: ast.DB.Models,
		})
	}

	return requests
}

// injectSearchContext injects search results into messages
// Adds search context as a system message after existing system messages
func (ast *Assistant) injectSearchContext(messages []context.Message, refCtx *searchTypes.ReferenceContext) []context.Message {
	if refCtx == nil || len(refCtx.References) == 0 {
		return messages
	}

	// Build the search context message
	var contentParts []string

	// Add citation prompt
	if refCtx.Prompt != "" {
		contentParts = append(contentParts, refCtx.Prompt)
	}

	// Add XML context
	if refCtx.XML != "" {
		contentParts = append(contentParts, refCtx.XML)
	}

	if len(contentParts) == 0 {
		return messages
	}

	// Create system message with search context
	searchMessage := context.Message{
		Role:    "system",
		Content: strings.Join(contentParts, "\n\n"),
	}

	// Find the position to insert the search message
	// Insert after any existing system messages but before user messages
	insertIndex := 0
	for i, msg := range messages {
		if msg.Role == "system" {
			insertIndex = i + 1
		} else {
			break
		}
	}

	// Insert the search message
	result := make([]context.Message, 0, len(messages)+1)
	result = append(result, messages[:insertIndex]...)
	result = append(result, searchMessage)
	result = append(result, messages[insertIndex:]...)

	return result
}

// extractQueryFromMessages extracts the search query from messages
// Uses the last user message as the query
func extractQueryFromMessages(messages []context.Message) string {
	// Find the last user message
	for i := len(messages) - 1; i >= 0; i-- {
		if messages[i].Role == "user" {
			content := messages[i].Content
			// Handle string content
			if str, ok := content.(string); ok {
				return str
			}
			// Handle content parts (array of objects)
			if parts, ok := content.([]interface{}); ok {
				for _, part := range parts {
					if partMap, ok := part.(map[string]interface{}); ok {
						if partMap["type"] == "text" {
							if text, ok := partMap["text"].(string); ok {
								return text
							}
						}
					}
				}
			}
		}
	}
	return ""
}

// truncateString truncates a string to maxLen characters
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

// ============================================================================
// Storage: Save Search Results
// ============================================================================

// SearchExecutionResult holds all data from search execution for storage
type SearchExecutionResult struct {
	Query      string                        // Original query (before keyword optimization)
	Keywords   []string                      // Extracted keywords
	Config     map[string]any                // Search config used
	RefCtx     *searchTypes.ReferenceContext // Reference context with results
	Duration   int64                         // Search duration in ms
	Error      error                         // Error if failed
	SearchType string                        // "auto", "web", "kb", "db"
}

// saveSearch saves search results to storage
// Called after search execution completes (success or failure)
func (ast *Assistant) saveSearch(ctx *context.Context, execResult *SearchExecutionResult) {
	// Get store
	store := GetStore()
	if store == nil {
		ctx.Logger.Debug("Storage not configured, skipping search save")
		return
	}

	// Build search record
	searchRecord := &storeTypes.Search{
		RequestID: ctx.RequestID(),
		ChatID:    ctx.ChatID,
		Query:     execResult.Query,
		Keywords:  execResult.Keywords,
		Config:    execResult.Config,
		Source:    execResult.SearchType,
		Duration:  execResult.Duration,
		CreatedAt: time.Now(),
	}

	// Set error if present
	if execResult.Error != nil {
		searchRecord.Error = execResult.Error.Error()
	}

	// Convert references if available
	if execResult.RefCtx != nil {
		searchRecord.References = convertToStoreReferences(execResult.RefCtx.References)
		searchRecord.XML = execResult.RefCtx.XML
		searchRecord.Prompt = execResult.RefCtx.Prompt
	}

	// Save to store
	if err := store.SaveSearch(searchRecord); err != nil {
		ctx.Logger.Warn("Failed to save search record: %v", err)
		return
	}

	ctx.Logger.Debug("Search record saved: request_id=%s, refs=%d",
		searchRecord.RequestID, len(searchRecord.References))
}

// convertToStoreReferences converts search References to store References
func convertToStoreReferences(refs []*searchTypes.Reference) []storeTypes.Reference {
	if len(refs) == 0 {
		return nil
	}

	storeRefs := make([]storeTypes.Reference, len(refs))
	for i, ref := range refs {
		if ref == nil {
			continue
		}

		// Parse citation ID as integer (e.g., "1", "2", "3")
		index := i + 1 // Default to position-based index
		if ref.ID != "" {
			if n, err := fmt.Sscanf(ref.ID, "%d", &index); n != 1 || err != nil {
				index = i + 1
			}
		}

		storeRefs[i] = storeTypes.Reference{
			Index:   index,
			Type:    string(ref.Type),
			Title:   ref.Title,
			URL:     ref.URL,
			Snippet: truncateString(ref.Content, 200), // Short snippet
			Content: ref.Content,
			Metadata: map[string]any{
				"weight": ref.Weight,
				"score":  ref.Score,
				"source": string(ref.Source),
			},
		}
	}

	return storeRefs
}

// configToMap converts search config to map for storage
func (ast *Assistant) configToMap(config *searchTypes.Config) map[string]any {
	if config == nil {
		return nil
	}

	result := make(map[string]any)

	if config.Web != nil {
		result["web"] = map[string]any{
			"provider":    config.Web.Provider,
			"max_results": config.Web.MaxResults,
		}
	}

	if config.KB != nil {
		result["kb"] = map[string]any{
			"threshold": config.KB.Threshold,
			"graph":     config.KB.Graph,
		}
	}

	if config.DB != nil {
		result["db"] = map[string]any{
			"max_results": config.DB.MaxResults,
		}
	}

	if config.Weights != nil {
		result["weights"] = map[string]any{
			"user": config.Weights.User,
			"hook": config.Weights.Hook,
			"auto": config.Weights.Auto,
		}
	}

	return result
}
