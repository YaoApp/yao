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

// parseAgentResponse parses the agent response into search result items
// The agent should return a JSON structure with search results
func (p *AgentProvider) parseAgentResponse(response interface{}, source types.SourceType) ([]*types.ResultItem, int, string) {
	if response == nil {
		return nil, 0, "Agent returned nil response"
	}

	// Try to extract data from response
	var data map[string]interface{}

	// Handle different response types
	switch v := response.(type) {
	case map[string]interface{}:
		data = v
	case string:
		// Try to parse as JSON
		if err := json.Unmarshal([]byte(v), &data); err != nil {
			return nil, 0, fmt.Sprintf("Failed to parse agent response as JSON: %v", err)
		}
	default:
		// Try to marshal and unmarshal
		jsonBytes, err := json.Marshal(response)
		if err != nil {
			return nil, 0, fmt.Sprintf("Failed to serialize agent response: %v", err)
		}
		if err := json.Unmarshal(jsonBytes, &data); err != nil {
			return nil, 0, fmt.Sprintf("Failed to parse agent response: %v", err)
		}
	}

	// Check for "next" field (custom hook data)
	if next, hasNext := data["next"]; hasNext && next != nil {
		if nextMap, ok := next.(map[string]interface{}); ok {
			data = nextMap
		} else if nextStr, ok := next.(string); ok {
			// Try to parse as JSON
			if err := json.Unmarshal([]byte(nextStr), &data); err != nil {
				return nil, 0, fmt.Sprintf("Failed to parse next hook data: %v", err)
			}
		}
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
