package web

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/yaoapp/yao/agent/caller"
	agentContext "github.com/yaoapp/yao/agent/context"
	"github.com/yaoapp/yao/agent/search/types"
)

// AgentProvider implements web search using another agent (AI Search)
type AgentProvider struct {
	agentID string // Agent/Assistant ID (e.g., "workers.search.web")
}

// NewAgentProvider creates a new Agent provider
func NewAgentProvider(agentID string) *AgentProvider {
	return &AgentProvider{
		agentID: agentID,
	}
}

// Search executes web search via agent delegation
// The agent can understand intent, generate optimized queries, and return structured results
func (p *AgentProvider) Search(ctx *agentContext.Context, req *types.Request) (*types.Result, error) {
	startTime := time.Now()

	// Check if context is provided
	if ctx == nil {
		return &types.Result{
			Type:     types.SearchTypeWeb,
			Query:    req.Query,
			Source:   req.Source,
			Items:    []*types.ResultItem{},
			Total:    0,
			Duration: time.Since(startTime).Milliseconds(),
			Error:    "Agent mode requires context",
		}, nil
	}

	// Check if AgentGetterFunc is initialized
	if caller.AgentGetterFunc == nil {
		return &types.Result{
			Type:     types.SearchTypeWeb,
			Query:    req.Query,
			Source:   req.Source,
			Items:    []*types.ResultItem{},
			Total:    0,
			Duration: time.Since(startTime).Milliseconds(),
			Error:    "AgentGetterFunc not initialized",
		}, nil
	}

	// Get the agent
	agent, err := caller.AgentGetterFunc(p.agentID)
	if err != nil {
		return &types.Result{
			Type:     types.SearchTypeWeb,
			Query:    req.Query,
			Source:   req.Source,
			Items:    []*types.ResultItem{},
			Total:    0,
			Duration: time.Since(startTime).Milliseconds(),
			Error:    fmt.Sprintf("Agent '%s' not found: %v", p.agentID, err),
		}, nil
	}

	// Build message for the agent
	// Include search parameters in the message content
	searchParams := map[string]interface{}{
		"query":  req.Query,
		"type":   "web",
		"source": string(req.Source),
	}

	if req.Limit > 0 {
		searchParams["limit"] = req.Limit
	}
	if len(req.Sites) > 0 {
		searchParams["sites"] = req.Sites
	}
	if req.TimeRange != "" {
		searchParams["time_range"] = req.TimeRange
	}

	// Convert to JSON for the message
	paramsJSON, err := json.Marshal(searchParams)
	if err != nil {
		return &types.Result{
			Type:     types.SearchTypeWeb,
			Query:    req.Query,
			Source:   req.Source,
			Items:    []*types.ResultItem{},
			Total:    0,
			Duration: time.Since(startTime).Milliseconds(),
			Error:    fmt.Sprintf("Failed to serialize search params: %v", err),
		}, nil
	}

	// Create message for the agent
	message := agentContext.Message{
		Role:    "user",
		Content: string(paramsJSON),
	}

	// Call the agent with skip options (no history, no output)
	opts := &agentContext.Options{
		Skip: &agentContext.Skip{
			History: true,
			Output:  true,
		},
	}

	response, err := agent.Stream(ctx, []agentContext.Message{message}, opts)
	if err != nil {
		return &types.Result{
			Type:     types.SearchTypeWeb,
			Query:    req.Query,
			Source:   req.Source,
			Items:    []*types.ResultItem{},
			Total:    0,
			Duration: time.Since(startTime).Milliseconds(),
			Error:    fmt.Sprintf("Agent call failed: %v", err),
		}, nil
	}

	// Parse the agent response
	items, total, parseErr := p.parseAgentResponse(response, req.Source)
	if parseErr != "" {
		return &types.Result{
			Type:     types.SearchTypeWeb,
			Query:    req.Query,
			Source:   req.Source,
			Items:    []*types.ResultItem{},
			Total:    0,
			Duration: time.Since(startTime).Milliseconds(),
			Error:    parseErr,
		}, nil
	}

	return &types.Result{
		Type:     types.SearchTypeWeb,
		Query:    req.Query,
		Source:   req.Source,
		Items:    items,
		Total:    total,
		Duration: time.Since(startTime).Milliseconds(),
	}, nil
}

// parseAgentResponse parses the agent's *context.Response into search result items
// Now that agent.Stream() returns *context.Response directly,
// we can access fields without type assertions.
//
// The agent returns search results in response.Next field
func (p *AgentProvider) parseAgentResponse(response *agentContext.Response, source types.SourceType) ([]*types.ResultItem, int, string) {
	if response == nil || response.Next == nil {
		return nil, 0, "Agent returned nil response"
	}

	// Extract data from Next field
	data := extractNextData(response.Next)
	if data == nil {
		return nil, 0, "Failed to extract data from agent response"
	}

	// Extract items from data
	items := []*types.ResultItem{}
	total := 0

	if itemsData, ok := data["items"].([]interface{}); ok {
		for _, itemData := range itemsData {
			if item, ok := itemData.(map[string]interface{}); ok {
				resultItem := &types.ResultItem{
					Type:   types.SearchTypeWeb,
					Source: source,
				}

				if title, ok := item["title"].(string); ok {
					resultItem.Title = title
				}
				if content, ok := item["content"].(string); ok {
					resultItem.Content = content
				}
				if url, ok := item["url"].(string); ok {
					resultItem.URL = url
				}
				if score, ok := item["score"].(float64); ok {
					resultItem.Score = score
				}

				items = append(items, resultItem)
			}
		}
	}

	if totalVal, ok := data["total"].(float64); ok {
		total = int(totalVal)
	} else {
		total = len(items)
	}

	return items, total, ""
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
