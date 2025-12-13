package assistant

import (
	"strings"

	"github.com/yaoapp/yao/agent/context"
	"github.com/yaoapp/yao/agent/search"
	"github.com/yaoapp/yao/agent/search/nlp/keyword"
	searchTypes "github.com/yaoapp/yao/agent/search/types"
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

	// Extract query from messages
	query := extractQueryFromMessages(messages)
	if query == "" {
		ctx.Logger.Info("No query found in messages, skipping auto search")
		return nil
	}

	// Check if keyword extraction should be skipped
	skipKeyword := false
	if len(opts) > 0 && opts[0] != nil && opts[0].Skip != nil {
		skipKeyword = opts[0].Skip.Keyword
	}

	// Extract keywords for web search if:
	// 1. uses.keyword is configured (not empty)
	// 2. Skip.Keyword is not true
	// 3. Web search is enabled
	webSearchEnabled := searchConfig != nil && searchConfig.Web != nil
	if webSearchEnabled && !skipKeyword && searchUses.Keyword != "" {
		extractor := keyword.NewExtractor(searchUses.Keyword, searchConfig.Keyword)
		keywords, err := extractor.Extract(ctx, query, nil)
		if err != nil {
			ctx.Logger.Warn("Keyword extraction failed, using original query: %v", err)
		} else if len(keywords) > 0 {
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

	// Execute searches in parallel
	ctx.Logger.Info("Executing %d search requests for query: %s", len(requests), truncateString(query, 50))

	results, err := searcher.All(ctx, requests)
	if err != nil {
		// Log error but don't fail - search errors shouldn't block the main flow
		ctx.Logger.Error("Auto search failed: %v", err)
		return nil
	}

	// Build reference context (includes references, XML, and prompt)
	var citationConfig *searchTypes.CitationConfig
	if searchConfig != nil {
		citationConfig = searchConfig.Citation
	}
	refCtx := search.BuildReferenceContext(results, citationConfig)

	if len(refCtx.References) == 0 {
		ctx.Logger.Info("No search results found")
		return nil
	}

	ctx.Logger.Info("Auto search completed: %d references", len(refCtx.References))
	return refCtx
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
