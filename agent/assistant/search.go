package assistant

import (
	"encoding/json"
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
// Returns nil if search should be skipped, otherwise returns SearchIntent with types to search
// Search is skipped if:
// - opts.Skip.Search is true
// - createResponse.Search is false
// - uses.search is "disabled"
// - assistant has no search configuration
// - needsearch intent detection returns false
func (ast *Assistant) shouldAutoSearch(ctx *context.Context, messages []context.Message, createResponse *context.HookCreateResponse, opts *context.Options) *SearchIntent {
	// Check if search is skipped via options
	if opts != nil && opts.Skip != nil && opts.Skip.Search {
		ctx.Logger.Debug("Auto search skipped by opts.Skip.Search")
		return nil
	}

	// Check if search is skipped via ctx.Metadata["__disable_search"]
	if ctx != nil && ctx.Metadata != nil {
		disableSearch := getBool(ctx.Metadata, "__disable_search")
		if disableSearch {
			ctx.Logger.Debug("Auto search skipped by ctx.Metadata['__disable_search']")
			return nil
		}
	}

	// Check createResponse.Search field (highest priority from Create hook)
	// Supports: bool | SearchIntent | nil
	if createResponse != nil && createResponse.Search != nil {
		intent := parseSearchField(createResponse.Search)
		if intent != nil {
			if !intent.NeedSearch {
				ctx.Logger.Info("Auto search disabled by createResponse.Search")
				return nil
			}
			ctx.Logger.Info("Auto search controlled by createResponse.Search: types=%v", intent.SearchTypes)
			return intent
		}
	}

	// Get merged uses configuration
	uses := ast.getMergedSearchUses(createResponse, opts)

	// Check if search is explicitly disabled
	if uses != nil && uses.Search == "disabled" {
		ctx.Logger.Info("Auto search disabled by uses.search=disabled")
		return nil
	}

	// Check if assistant has search configuration
	if ast.Search == nil && (uses == nil || uses.Search == "") {
		return nil
	}

	// Check search intent using __yao.needsearch agent
	intent := ast.checkSearchIntent(ctx, messages)
	if intent == nil || !intent.NeedSearch {
		ctx.Logger.Info("Auto search skipped: intent detection returned false")
		return nil
	}

	return intent
}

// parseSearchField parses the Search field from HookCreateResponse
// Supports: bool | SearchIntent | map[string]any | nil
func parseSearchField(search any) *SearchIntent {
	if search == nil {
		return nil
	}

	switch v := search.(type) {
	case bool:
		// bool: true = enable all, false = disable all
		if v {
			return &SearchIntent{
				NeedSearch:  true,
				SearchTypes: []string{"web", "kb", "db"},
				Confidence:  1.0,
				Reason:      "enabled by hook",
			}
		}
		return &SearchIntent{
			NeedSearch:  false,
			SearchTypes: []string{},
			Confidence:  1.0,
			Reason:      "disabled by hook",
		}

	case *SearchIntent:
		// SearchIntent is alias for context.SearchIntent, so this covers both
		return v

	case SearchIntent:
		return &v

	case map[string]any:
		// Parse from map (e.g., from JSON)
		intent := &SearchIntent{
			NeedSearch:  false,
			SearchTypes: []string{},
			Confidence:  0.5,
		}

		if needSearch, ok := v["need_search"].(bool); ok {
			intent.NeedSearch = needSearch
		}

		if types, ok := v["search_types"].([]any); ok {
			for _, t := range types {
				if typeStr, ok := t.(string); ok {
					intent.SearchTypes = append(intent.SearchTypes, typeStr)
				}
			}
		}

		if confidence, ok := v["confidence"].(float64); ok {
			intent.Confidence = confidence
		}

		if reason, ok := v["reason"].(string); ok {
			intent.Reason = reason
		}

		return intent

	default:
		return nil
	}
}

// checkSearchIntent uses __yao.needsearch agent to determine if search is needed
// Returns SearchIntent with search types and confidence
func (ast *Assistant) checkSearchIntent(ctx *context.Context, messages []context.Message) *SearchIntent {
	// Default intent: no search needed (fallback when agent unavailable or fails)
	defaultIntent := &SearchIntent{
		NeedSearch:  false,
		SearchTypes: []string{},
		Confidence:  0,
	}

	// Build a single text message with conversation context
	intentMessages := buildContextMessage(messages)
	if len(intentMessages) == 0 {
		return defaultIntent // No messages, skip search
	}

	// Try to get __yao.needsearch agent
	needsearchAst, err := Get("__yao.needsearch")
	if err != nil {
		ctx.Logger.Debug("__yao.needsearch agent not available: %v, skipping search", err)
		return defaultIntent // Agent not available, skip search
	}

	// === Output: Send loading message ===
	loadingID := ast.sendIntentLoading(ctx)

	// Call the needsearch agent (Stack will auto-track)
	// IMPORTANT: Skip search to prevent infinite loop, skip output to prevent JSON showing in UI
	opts := &context.Options{
		Skip: &context.Skip{
			History: true, // Don't save to history
			Search:  true, // Skip search to prevent infinite loop
			Output:  true, // Skip output to prevent JSON showing in UI
		},
	}

	result, err := needsearchAst.Stream(ctx, intentMessages, opts)
	if err != nil {
		ctx.Logger.Debug("__yao.needsearch failed: %v, skipping search", err)
		// === Output: Send done (error case, skip search) ===
		ast.sendIntentDone(ctx, loadingID, false, "")
		return defaultIntent // On error, skip search
	}

	// Parse the result
	// Next hook returns {data: {need_search: bool, search_types: [], confidence: float}}
	// First try to get from Next hook response
	if result.Next != nil {
		if nextData, ok := result.Next.(map[string]interface{}); ok {
			// Check for data field (from Next hook's {data: result})
			var intentData map[string]interface{}
			if data, ok := nextData["data"].(map[string]interface{}); ok {
				intentData = data
			} else {
				intentData = nextData
			}

			intent := parseSearchIntent(intentData)
			if intent != nil {
				ctx.Logger.Debug("Search intent (from Next): need_search=%v, types=%v, confidence=%.2f, reason=%s",
					intent.NeedSearch, intent.SearchTypes, intent.Confidence, intent.Reason)
				ast.sendIntentDone(ctx, loadingID, intent.NeedSearch, intent.Reason)
				return intent
			}
		}
	}

	// Fallback: parse from Completion.Content if Next hook didn't process
	if result.Completion != nil {
		content, ok := result.Completion.Content.(string)
		if !ok || content == "" {
			ast.sendIntentDone(ctx, loadingID, false, "")
			return defaultIntent
		}
		intent := parseSearchIntentFromContent(content)
		ctx.Logger.Debug("Search intent (from Content): need_search=%v, types=%v, confidence=%.2f, reason=%s",
			intent.NeedSearch, intent.SearchTypes, intent.Confidence, intent.Reason)
		ast.sendIntentDone(ctx, loadingID, intent.NeedSearch, intent.Reason)
		return intent
	}

	// Default: skip search if we can't parse the result
	// === Output: Send done (default case) ===
	ast.sendIntentDone(ctx, loadingID, false, "")
	return defaultIntent
}

// parseSearchIntent parses SearchIntent from intent data map
func parseSearchIntent(intentData map[string]interface{}) *SearchIntent {
	if intentData == nil {
		return nil
	}

	needSearch, ok := intentData["need_search"].(bool)
	if !ok {
		return nil
	}

	intent := &SearchIntent{
		NeedSearch:  needSearch,
		SearchTypes: []string{},
		Confidence:  0.5, // Default confidence
	}

	// Parse search_types
	if types, ok := intentData["search_types"].([]interface{}); ok {
		for _, t := range types {
			if typeStr, ok := t.(string); ok {
				// Validate type
				typeStr = strings.ToLower(typeStr)
				if typeStr == "web" || typeStr == "kb" || typeStr == "db" {
					intent.SearchTypes = append(intent.SearchTypes, typeStr)
				}
			}
		}
	}

	// Parse confidence
	if confidence, ok := intentData["confidence"].(float64); ok {
		intent.Confidence = confidence
	}

	// Parse reason
	if reason, ok := intentData["reason"].(string); ok {
		intent.Reason = reason
	}

	return intent
}

// parseSearchIntentFromContent parses SearchIntent from LLM completion content
// Handles JSON wrapped in markdown code blocks
func parseSearchIntentFromContent(content string) *SearchIntent {
	// Default intent: no search needed
	defaultIntent := &SearchIntent{
		NeedSearch:  false,
		SearchTypes: []string{},
		Confidence:  0,
	}

	// Remove markdown code block if present
	content = strings.TrimSpace(content)
	if strings.HasPrefix(content, "```json") {
		content = strings.TrimPrefix(content, "```json")
		content = strings.TrimSuffix(content, "```")
		content = strings.TrimSpace(content)
	} else if strings.HasPrefix(content, "```") {
		content = strings.TrimPrefix(content, "```")
		content = strings.TrimSuffix(content, "```")
		content = strings.TrimSpace(content)
	}

	// Try to parse JSON
	var result map[string]interface{}
	if err := json.Unmarshal([]byte(content), &result); err != nil {
		// Failed to parse, default to no search
		return defaultIntent
	}

	intent := parseSearchIntent(result)
	if intent == nil {
		return defaultIntent
	}

	return intent
}

// sendIntentLoading sends the initial intent detection loading message
// Returns the message ID for later replacement
func (ast *Assistant) sendIntentLoading(ctx *context.Context) string {
	loadingMsg := i18n.T(ctx.Locale, "search.intent.loading")

	msg := &message.Message{
		Type: "loading",
		Props: map[string]any{
			"message": loadingMsg,
		},
	}

	// Send and get message ID
	msgID, err := ctx.SendStream(msg)
	if err != nil {
		ctx.Logger.Warn("Failed to send intent loading message: %v", err)
		return ""
	}

	return msgID
}

// sendIntentDone replaces loading with result
// Only marks as done when needSearch is false (no further loading will follow)
// When needSearch is true, the search loading will continue
func (ast *Assistant) sendIntentDone(ctx *context.Context, loadingID string, needSearch bool, reason string) {
	if loadingID == "" {
		return
	}

	var resultMsg string
	if needSearch {
		resultMsg = i18n.T(ctx.Locale, "search.intent.need_search")
	} else {
		resultMsg = i18n.T(ctx.Locale, "search.intent.no_search")
	}

	msg := &message.Message{
		MessageID:   loadingID,
		Delta:       true,
		DeltaAction: message.DeltaReplace,
		Type:        "loading",
		Props: map[string]any{
			"message": resultMsg,
			"done":    true, // Intent detection loading is independent, always close it
		},
	}

	if err := ctx.Send(msg); err != nil {
		ctx.Logger.Warn("Failed to send intent done message: %v", err)
	}
}

// getMergedSearchUses returns the merged uses configuration for search
// Priority:  createResponse > options.Uses > assistant
func (ast *Assistant) getMergedSearchUses(createResponse *context.HookCreateResponse, opts ...*context.Options) *context.Uses {

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

	// Override with options.Uses if provided (highest priority)
	if len(opts) > 0 && opts[0] != nil && opts[0].Uses != nil {

		if uses == nil {
			uses = &context.Uses{}
		}

		if opts[0].Uses.Search != "" {
			uses.Search = opts[0].Uses.Search
		}
		if opts[0].Uses.Web != "" {
			uses.Web = opts[0].Uses.Web
		}

		if opts[0].Uses.Keyword != "" {
			uses.Keyword = opts[0].Uses.Keyword
		}
		if opts[0].Uses.QueryDSL != "" {
			uses.QueryDSL = opts[0].Uses.QueryDSL
		}
		if opts[0].Uses.Rerank != "" {
			uses.Rerank = opts[0].Uses.Rerank
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

// executeAutoSearch executes auto search based on configuration and intent
// Returns ReferenceContext with results and formatted context
// intent specifies which search types to execute (from needsearch agent)
// opts is optional, used to check Skip.Keyword
func (ast *Assistant) executeAutoSearch(ctx *context.Context, messages []context.Message, createResponse *context.HookCreateResponse, intent *SearchIntent, opts ...*context.Options) *searchTypes.ReferenceContext {
	ctx.Logger.Phase("Search")
	defer ctx.Logger.PhaseComplete("Search")

	// Get merged uses configuration
	uses := ast.getMergedSearchUses(createResponse, opts...)

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

	// Build query with conversation context for better keyword extraction
	// This helps the keyword extractor understand the full context
	contextMessages := buildContextMessage(messages)
	query := originalQuery
	if len(contextMessages) > 0 {
		if contextStr, ok := contextMessages[0].Content.(string); ok {
			query = contextStr
		}
	}

	// Check if keyword extraction should be skipped
	skipKeyword := false
	if len(opts) > 0 && opts[0] != nil && opts[0].Skip != nil {
		skipKeyword = opts[0].Skip.Keyword
	}

	// Build search requests based on configuration and intent
	// Keyword extraction is done inside buildSearchRequests for web search
	buildOpts := &buildSearchRequestsOptions{
		skipKeyword: skipKeyword,
		usesKeyword: searchUses.Keyword,
	}
	requests, extractedKeywords := ast.buildSearchRequests(ctx, query, searchConfig, intent, buildOpts)
	if len(requests) == 0 {
		ctx.Logger.Info("No search requests to execute")
		return nil
	}

	// Update query if keywords were extracted (for web search)
	if len(extractedKeywords) > 0 {
		query = keywordsToQuery(extractedKeywords)
	}

	// === Output: Send loading message ===
	loadingID := ast.sendSearchLoading(ctx)

	// === Trace: Create search trace node ===
	searchNode := ast.createSearchTrace(ctx, query, requests)

	// Execute searches in parallel
	// Build provider info for logging
	providerInfo := ast.getSearchProviderInfo(searchConfig, searchUses)
	ctx.Logger.Info("Executing %d search requests via %s for query: %s", len(requests), providerInfo, truncateString(query, 50))

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
		Results:    results,
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

// sendKeywordLoading sends the keyword extraction loading message
// Returns the message ID for later replacement
func (ast *Assistant) sendKeywordLoading(ctx *context.Context) string {
	loadingMsg := i18n.T(ctx.Locale, "search.keyword.loading")

	msg := &message.Message{
		Type: "loading",
		Props: map[string]any{
			"message": loadingMsg,
		},
	}

	// Send and get message ID
	msgID, err := ctx.SendStream(msg)
	if err != nil {
		ctx.Logger.Warn("Failed to send keyword loading message: %v", err)
		return ""
	}

	return msgID
}

// sendKeywordDone replaces keyword loading with done message
func (ast *Assistant) sendKeywordDone(ctx *context.Context, loadingID string, success bool) {
	if loadingID == "" {
		return
	}

	resultMsg := i18n.T(ctx.Locale, "search.keyword.done")

	msg := &message.Message{
		MessageID:   loadingID,
		Delta:       true,
		DeltaAction: message.DeltaReplace,
		Type:        "loading",
		Props: map[string]any{
			"message": resultMsg,
			"done":    true,
		},
	}

	if err := ctx.Send(msg); err != nil {
		ctx.Logger.Warn("Failed to send keyword done message: %v", err)
	}
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

// buildSearchRequestsOptions contains options for building search requests
type buildSearchRequestsOptions struct {
	skipKeyword bool   // Skip keyword extraction
	usesKeyword string // Keyword extractor config: "builtin", "<assistant-id>", "mcp:<server>.<tool>"
}

// buildSearchRequests builds search requests based on assistant configuration and intent
// intent specifies which search types to execute (from needsearch agent)
// Returns requests and extracted keywords (if any)
func (ast *Assistant) buildSearchRequests(ctx *context.Context, query string, config *searchTypes.Config, intent *SearchIntent, opts *buildSearchRequestsOptions) ([]*searchTypes.Request, []searchTypes.Keyword) {
	var requests []*searchTypes.Request
	var extractedKeywords []searchTypes.Keyword

	// Helper to check if a search type is allowed by intent
	isTypeAllowed := func(searchType string) bool {
		if intent == nil || len(intent.SearchTypes) == 0 {
			return true // No intent or empty types means all types allowed
		}
		for _, t := range intent.SearchTypes {
			if t == searchType {
				return true
			}
		}
		return false
	}

	// Web search - check if web search is configured and allowed by intent
	if config != nil && config.Web != nil && isTypeAllowed("web") {
		webQuery := query

		// Extract keywords for web search if configured
		if opts != nil && !opts.skipKeyword && opts.usesKeyword != "" {
			// === Output: Send keyword extraction loading ===
			keywordLoadingID := ast.sendKeywordLoading(ctx)

			extractor := keyword.NewExtractor(opts.usesKeyword, config.Keyword)
			keywords, err := extractor.Extract(ctx, query, nil)
			if err != nil {
				ctx.Logger.Warn("Keyword extraction failed, using original query: %v", err)
				ast.sendKeywordDone(ctx, keywordLoadingID, false)
			} else if len(keywords) > 0 {
				extractedKeywords = keywords
				// Use extracted keywords as the search query for web search
				webQuery = keywordsToQuery(keywords)
				ctx.Logger.Info("Extracted keywords for web search: %s -> %s", truncateString(query, 30), webQuery)
				ast.sendKeywordDone(ctx, keywordLoadingID, true)
			} else {
				ast.sendKeywordDone(ctx, keywordLoadingID, true)
			}
		}

		requests = append(requests, &searchTypes.Request{
			Type:   searchTypes.SearchTypeWeb,
			Query:  webQuery,
			Source: searchTypes.SourceAuto,
			Limit:  config.Web.MaxResults,
		})
	}

	// KB search - check if KB is configured and allowed by intent
	if ast.KB != nil && len(ast.KB.Collections) > 0 && isTypeAllowed("kb") {
		limit := 10
		threshold := 0.7
		if config != nil && config.KB != nil {
			if config.KB.Threshold > 0 {
				threshold = config.KB.Threshold
			}
		}

		// Filter collections by authorization (Collection-level permission check)
		allowedCollections := FilterKBCollectionsByAuth(ctx, ast.KB.Collections)
		if len(allowedCollections) == 0 {
			ctx.Logger.Info("No accessible KB collections after auth filter")
		} else {
			// Build KB request
			kbReq := &searchTypes.Request{
				Type:        searchTypes.SearchTypeKB,
				Query:       query, // KB uses original query for semantic search
				Source:      searchTypes.SourceAuto,
				Limit:       limit,
				Collections: allowedCollections,
				Threshold:   threshold,
				Graph:       config != nil && config.KB != nil && config.KB.Graph,
			}

			requests = append(requests, kbReq)
		}
	}

	// DB search - check if DB is configured and allowed by intent
	if ast.DB != nil && len(ast.DB.Models) > 0 && isTypeAllowed("db") {
		limit := 20
		if config != nil && config.DB != nil && config.DB.MaxResults > 0 {
			limit = config.DB.MaxResults
		}

		// Build DB request with auth where clauses
		dbReq := &searchTypes.Request{
			Type:   searchTypes.SearchTypeDB,
			Query:  query, // DB uses original query for QueryDSL generation
			Source: searchTypes.SourceAuto,
			Limit:  limit,
			Models: ast.DB.Models,
		}

		// Apply authorization where clauses
		if authWheres := BuildDBAuthWheres(ctx); authWheres != nil {
			dbReq.Wheres = authWheres
		}

		requests = append(requests, dbReq)
	}

	return requests, extractedKeywords
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

// extractTextContent extracts text-only content from a message
// For multimodal messages, concatenates all text parts
// Returns empty string if no text content found
func extractTextContent(msg context.Message) string {
	content := msg.Content
	// Handle string content
	if str, ok := content.(string); ok {
		return str
	}
	// Handle content parts (array of objects) - extract only text parts
	if parts, ok := content.([]interface{}); ok {
		var texts []string
		for _, part := range parts {
			if partMap, ok := part.(map[string]interface{}); ok {
				if partMap["type"] == "text" {
					if text, ok := partMap["text"].(string); ok {
						texts = append(texts, text)
					}
				}
			}
		}
		if len(texts) > 0 {
			return strings.Join(texts, "\n")
		}
	}
	// Handle []context.ContentPart
	if parts, ok := content.([]context.ContentPart); ok {
		var texts []string
		for _, part := range parts {
			if part.Type == context.ContentText && part.Text != "" {
				texts = append(texts, part.Text)
			}
		}
		if len(texts) > 0 {
			return strings.Join(texts, "\n")
		}
	}
	return ""
}

// buildContextMessage builds a single user message with conversation context
// Filters out system messages and extracts text-only content
// Only takes the last 5 messages for efficiency
// Returns a slice with one message containing the full context, or empty slice if no content
func buildContextMessage(messages []context.Message) []context.Message {
	const maxMessages = 5

	// Take only the last maxMessages (excluding system messages)
	var recentMessages []context.Message
	for i := len(messages) - 1; i >= 0 && len(recentMessages) < maxMessages; i-- {
		if messages[i].Role != "system" {
			recentMessages = append(recentMessages, messages[i])
		}
	}
	// Reverse to maintain chronological order
	for i, j := 0, len(recentMessages)-1; i < j; i, j = i+1, j-1 {
		recentMessages[i], recentMessages[j] = recentMessages[j], recentMessages[i]
	}

	var contextParts []string
	var lastUserMessage string

	for _, msg := range recentMessages {
		textContent := extractTextContent(msg)
		if textContent == "" {
			continue
		}

		// Format message with role label
		switch msg.Role {
		case "user":
			contextParts = append(contextParts, "[User]: "+textContent)
			lastUserMessage = textContent
		case "assistant":
			contextParts = append(contextParts, "[Assistant]: "+textContent)
		default:
			contextParts = append(contextParts, "["+string(msg.Role)+"]: "+textContent)
		}
	}

	// Build single message with context
	var result []context.Message
	if len(contextParts) > 1 {
		// Multiple messages: include conversation context
		fullContext := "=== Conversation Context ===\n" + strings.Join(contextParts, "\n\n") + "\n=== End Context ===\n\nCurrent user request: " + lastUserMessage
		result = append(result, context.Message{
			Role:    "user",
			Content: fullContext,
		})
	} else if lastUserMessage != "" {
		// Single user message: just use it directly
		result = append(result, context.Message{
			Role:    "user",
			Content: lastUserMessage,
		})
	}
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
	Keywords   []searchTypes.Keyword         // Extracted keywords with weights
	Config     map[string]any                // Search config used
	RefCtx     *searchTypes.ReferenceContext // Reference context with results
	Results    []*searchTypes.Result         // Raw search results (for extracting DSL, etc.)
	Duration   int64                         // Search duration in ms
	Error      error                         // Error if failed
	SearchType string                        // "auto", "web", "kb", "db"
}

// keywordsToQuery converts keywords with weights to a search query string
// Keywords are sorted by weight (descending) and joined with spaces
func keywordsToQuery(keywords []searchTypes.Keyword) string {
	if len(keywords) == 0 {
		return ""
	}

	// Sort by weight descending (higher weight first)
	sorted := make([]searchTypes.Keyword, len(keywords))
	copy(sorted, keywords)
	for i := 0; i < len(sorted)-1; i++ {
		for j := i + 1; j < len(sorted); j++ {
			if sorted[j].W > sorted[i].W {
				sorted[i], sorted[j] = sorted[j], sorted[i]
			}
		}
	}

	// Join keywords
	parts := make([]string, len(sorted))
	for i, kw := range sorted {
		parts[i] = kw.K
	}
	return strings.Join(parts, " ")
}

// keywordsToStrings converts keywords to string slice for storage
func keywordsToStrings(keywords []searchTypes.Keyword) []string {
	if len(keywords) == 0 {
		return nil
	}
	result := make([]string, len(keywords))
	for i, kw := range keywords {
		result[i] = kw.K
	}
	return result
}

// containsSearchType checks if a search type is in the list
func containsSearchType(types []string, searchType string) bool {
	for _, t := range types {
		if t == searchType {
			return true
		}
	}
	return false
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
		Keywords:  keywordsToStrings(execResult.Keywords),
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

	// Extract DSL from DB search results
	if execResult.Results != nil {
		for _, result := range execResult.Results {
			if result != nil && result.Type == searchTypes.SearchTypeDB && result.DSL != nil {
				searchRecord.DSL = result.DSL
				break // Only store the first DSL (usually there's only one DB search)
			}
		}
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

// getSearchProviderInfo returns a human-readable string describing the search provider(s)
func (ast *Assistant) getSearchProviderInfo(config *searchTypes.Config, uses *search.Uses) string {
	var parts []string

	// Web search provider - always show when web search is being executed
	webMode := ""
	if uses != nil {
		webMode = uses.Web
	}

	if webMode == "" || webMode == "builtin" {
		// Builtin mode: show the actual provider (tavily/serper/serpapi)
		provider := "tavily" // default
		if config != nil && config.Web != nil && config.Web.Provider != "" {
			provider = config.Web.Provider
		}
		parts = append(parts, "web:"+provider)
	} else if strings.HasPrefix(webMode, "mcp:") {
		parts = append(parts, "web:"+webMode)
	} else {
		parts = append(parts, "web:agent:"+webMode)
	}

	// KB search
	if config != nil && config.KB != nil && len(config.KB.Collections) > 0 {
		parts = append(parts, "kb")
	}

	// DB search
	if config != nil && config.DB != nil && len(config.DB.Models) > 0 {
		parts = append(parts, "db")
	}

	return strings.Join(parts, ", ")
}
