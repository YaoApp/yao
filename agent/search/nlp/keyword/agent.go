package keyword

import (
	"encoding/json"
	"fmt"

	"github.com/yaoapp/yao/agent/caller"
	agentContext "github.com/yaoapp/yao/agent/context"
	"github.com/yaoapp/yao/agent/search/types"
)

// AgentProvider delegates keyword extraction to an LLM-powered assistant
// The assistant can understand context and extract semantically relevant keywords
type AgentProvider struct {
	agentID string // Assistant ID to delegate to
}

// NewAgentProvider creates a new agent-based keyword extractor
func NewAgentProvider(agentID string) *AgentProvider {
	return &AgentProvider{
		agentID: agentID,
	}
}

// Extract extracts keywords by calling the target agent
// The agent receives the content and returns extracted keywords
func (p *AgentProvider) Extract(ctx *agentContext.Context, content string, opts *types.KeywordOptions) ([]string, error) {
	if ctx == nil {
		return nil, fmt.Errorf("context is required for agent keyword extraction")
	}

	// Check if AgentGetterFunc is initialized
	if caller.AgentGetterFunc == nil {
		return nil, fmt.Errorf("AgentGetterFunc not initialized")
	}

	// Get the agent
	agent, err := caller.AgentGetterFunc(p.agentID)
	if err != nil {
		return nil, fmt.Errorf("failed to get agent %s: %w", p.agentID, err)
	}

	// Build the request message
	requestData := map[string]interface{}{
		"content":      content,
		"max_keywords": opts.MaxKeywords,
		"language":     opts.Language,
	}
	requestJSON, _ := json.Marshal(requestData)

	// Create message for the agent
	messages := []agentContext.Message{
		{
			Role:    "user",
			Content: string(requestJSON),
		},
	}

	// Call the agent with skip options (no history, no output)
	options := &agentContext.Options{
		Skip: &agentContext.Skip{
			History: true,
			Output:  true,
		},
	}

	response, err := agent.Stream(ctx, messages, options)
	if err != nil {
		return nil, fmt.Errorf("agent call failed: %w", err)
	}

	// Parse the result from response.Next
	return p.parseResponse(response)
}

// parseResponse extracts keywords from the agent's *context.Response
// Now that agent.Stream() returns *context.Response directly,
// we can access fields without type assertions.
//
// The agent returns keywords in response.Next field
func (p *AgentProvider) parseResponse(response *agentContext.Response) ([]string, error) {
	if response == nil || response.Next == nil {
		return []string{}, nil
	}

	return p.parseNextData(response.Next)
}

// parseNextData extracts keywords from Next hook data
func (p *AgentProvider) parseNextData(next interface{}) ([]string, error) {
	if next == nil {
		return []string{}, nil
	}

	// Try to convert to map first (most common case)
	var data map[string]interface{}

	switch v := next.(type) {
	case map[string]interface{}:
		data = v
	case string:
		// Try to parse as JSON
		if err := json.Unmarshal([]byte(v), &data); err != nil {
			// Not a JSON object, try as array
			var keywords []string
			if err := json.Unmarshal([]byte(v), &keywords); err == nil {
				return keywords, nil
			}
			// Return as single keyword
			return []string{v}, nil
		}
	case []string:
		return v, nil
	case []interface{}:
		keywords := make([]string, 0, len(v))
		for _, item := range v {
			if s, ok := item.(string); ok {
				keywords = append(keywords, s)
			}
		}
		return keywords, nil
	default:
		// Try to marshal and unmarshal
		jsonBytes, err := json.Marshal(next)
		if err != nil {
			return []string{}, nil
		}
		if err := json.Unmarshal(jsonBytes, &data); err != nil {
			return []string{}, nil
		}
	}

	// Extract keywords from data
	// Try common field names: "keywords", "data", "data.keywords"
	if kw, ok := data["keywords"]; ok {
		return p.extractKeywordsFromValue(kw)
	}
	if d, ok := data["data"]; ok {
		if dm, ok := d.(map[string]interface{}); ok {
			if kw, ok := dm["keywords"]; ok {
				return p.extractKeywordsFromValue(kw)
			}
		}
		return p.extractKeywordsFromValue(d)
	}

	return []string{}, nil
}

// extractKeywordsFromValue extracts string array from various types
func (p *AgentProvider) extractKeywordsFromValue(v interface{}) ([]string, error) {
	switch kw := v.(type) {
	case []string:
		return kw, nil
	case []interface{}:
		keywords := make([]string, 0, len(kw))
		for _, item := range kw {
			if s, ok := item.(string); ok {
				keywords = append(keywords, s)
			}
		}
		return keywords, nil
	case string:
		var keywords []string
		if err := json.Unmarshal([]byte(kw), &keywords); err == nil {
			return keywords, nil
		}
		return []string{kw}, nil
	}
	return []string{}, nil
}
