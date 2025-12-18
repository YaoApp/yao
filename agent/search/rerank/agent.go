package rerank

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/yaoapp/yao/agent/caller"
	"github.com/yaoapp/yao/agent/context"
	"github.com/yaoapp/yao/agent/search/types"
)

// AgentProvider implements reranking by delegating to another agent
// The agent should have a Next Hook that accepts rerank request and returns reordered items
type AgentProvider struct {
	agentID string // Assistant ID to delegate to
}

// NewAgentProvider creates a new agent reranker
func NewAgentProvider(agentID string) *AgentProvider {
	return &AgentProvider{agentID: agentID}
}

// Rerank delegates reranking to an LLM-powered assistant
// The assistant receives items and query, returns reordered item IDs or items
func (p *AgentProvider) Rerank(ctx *context.Context, query string, items []*types.ResultItem, opts *types.RerankOptions) ([]*types.ResultItem, error) {
	if ctx == nil {
		return nil, fmt.Errorf("context is required for agent rerank")
	}

	// Get agent via caller interface (avoids circular dependency)
	agent, err := caller.AgentGetterFunc(p.agentID)
	if err != nil {
		return nil, fmt.Errorf("failed to get agent %s: %w", p.agentID, err)
	}

	// Build request message with items to rerank
	requestData := map[string]interface{}{
		"query":  query,
		"items":  items,
		"top_n":  opts.TopN,
		"action": "rerank",
	}
	requestJSON, _ := json.Marshal(requestData)

	// Create messages for agent
	messages := []context.Message{
		{
			Role:    "user",
			Content: string(requestJSON),
		},
	}

	// Call agent's Stream method with skip options (no history, no output)
	options := &context.Options{
		Skip: &context.Skip{
			History: true,
			Output:  true,
		},
	}

	response, err := agent.Stream(ctx, messages, options)
	if err != nil {
		return nil, fmt.Errorf("agent stream failed: %w", err)
	}

	// Parse response from response.Next
	return p.parseAgentResponse(response, items, opts)
}

// parseAgentResponse extracts reranked items from agent's *context.Response
// Now that agent.Stream() returns *context.Response directly,
// we can access fields without type assertions.
//
// Expected response.Next format:
// { "order": ["ref_001", "ref_003", "ref_002"] }
// Or: { "items": [{ "citation_id": "ref_001", ... }, ...] }
func (p *AgentProvider) parseAgentResponse(response *context.Response, originalItems []*types.ResultItem, opts *types.RerankOptions) ([]*types.ResultItem, error) {
	if response == nil || response.Next == nil {
		return originalItems, nil
	}

	// Build index map for quick lookup
	itemMap := make(map[string]*types.ResultItem)
	for _, item := range originalItems {
		if item.CitationID != "" {
			itemMap[item.CitationID] = item
		}
	}

	// Extract response data from Next field
	data := extractNextData(response.Next)
	if data == nil {
		return originalItems, nil
	}

	// Try to get reranked order from data
	// Expected format: { "order": ["ref_001", "ref_003", "ref_002"] }
	// Or: { "items": [{ "citation_id": "ref_001", ... }, ...] }

	var reranked []*types.ResultItem

	// Try "order" field (list of citation IDs)
	if order, ok := data["order"]; ok {
		if orderList := toStringSlice(order); len(orderList) > 0 {
			for _, id := range orderList {
				if item, exists := itemMap[id]; exists {
					reranked = append(reranked, item)
					delete(itemMap, id) // Avoid duplicates
				}
			}
			// Append remaining items not in order
			for _, item := range originalItems {
				if _, exists := itemMap[item.CitationID]; exists {
					reranked = append(reranked, item)
				}
			}
		}
	}

	// Try "items" field (full items or items with citation_id)
	if len(reranked) == 0 {
		if items, ok := data["items"]; ok {
			if itemsList := toItemsList(items); len(itemsList) > 0 {
				for _, respItem := range itemsList {
					// Check if it's just a reference or full item
					if citationID, ok := respItem["citation_id"].(string); ok {
						if item, exists := itemMap[citationID]; exists {
							reranked = append(reranked, item)
							delete(itemMap, citationID)
						}
					}
				}
				// Append remaining items
				for _, item := range originalItems {
					if _, exists := itemMap[item.CitationID]; exists {
						reranked = append(reranked, item)
					}
				}
			}
		}
	}

	// If no valid response, return original items
	if len(reranked) == 0 {
		reranked = originalItems
	}

	// Apply top N
	if opts.TopN > 0 && opts.TopN < len(reranked) {
		reranked = reranked[:opts.TopN]
	}

	return reranked, nil
}

// extractNextData extracts the actual data from response.Next field
// Handles nested structures like { "data": { ... } }
func extractNextData(next interface{}) map[string]interface{} {
	if next == nil {
		return nil
	}

	switch v := next.(type) {
	case map[string]interface{}:
		// Check for "data" wrapper
		if data, ok := v["data"].(map[string]interface{}); ok {
			return data
		}
		return v
	case string:
		// Try to parse as JSON
		var data map[string]interface{}
		if err := json.Unmarshal([]byte(v), &data); err == nil {
			return extractNextData(data)
		}
	}
	// Try to handle other types by converting to JSON and back
	if bytes, err := json.Marshal(next); err == nil {
		var data map[string]interface{}
		if err := json.Unmarshal(bytes, &data); err == nil {
			return extractNextData(data)
		}
	}
	return nil
}

// toStringSlice converts interface to string slice
func toStringSlice(v interface{}) []string {
	switch val := v.(type) {
	case []string:
		return val
	case []interface{}:
		result := make([]string, 0, len(val))
		for _, item := range val {
			if s, ok := item.(string); ok {
				result = append(result, s)
			}
		}
		return result
	}
	return nil
}

// toItemsList converts interface to list of maps
func toItemsList(v interface{}) []map[string]interface{} {
	switch val := v.(type) {
	case []map[string]interface{}:
		return val
	case []interface{}:
		result := make([]map[string]interface{}, 0, len(val))
		for _, item := range val {
			if m, ok := item.(map[string]interface{}); ok {
				result = append(result, m)
			}
		}
		return result
	}
	return nil
}

// extractAgentID extracts assistant ID from uses.rerank value
// For backward compatibility, strips any prefix if present
func extractAgentID(usesRerank string) string {
	// Remove any prefix like "agent:" if present
	if strings.HasPrefix(usesRerank, "agent:") {
		return strings.TrimPrefix(usesRerank, "agent:")
	}
	return usesRerank
}
