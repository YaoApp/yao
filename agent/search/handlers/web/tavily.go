package web

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/yaoapp/yao/agent/search/types"
)

const (
	tavilyAPIURL     = "https://api.tavily.com/search"
	tavilyAPITimeout = 30 * time.Second
)

// TavilyProvider implements web search using Tavily API
type TavilyProvider struct {
	apiKey     string
	maxResults int
}

// NewTavilyProvider creates a new Tavily provider
func NewTavilyProvider(cfg *types.WebConfig) *TavilyProvider {
	apiKey := ""
	if cfg != nil && cfg.APIKeyEnv != "" {
		// Support both "$ENV.VAR_NAME" and "VAR_NAME" formats
		envName := cfg.APIKeyEnv
		if len(envName) > 5 && envName[:5] == "$ENV." {
			envName = envName[5:]
		}
		apiKey = os.Getenv(envName)
	}

	maxResults := 10
	if cfg != nil && cfg.MaxResults > 0 {
		maxResults = cfg.MaxResults
	}

	return &TavilyProvider{
		apiKey:     apiKey,
		maxResults: maxResults,
	}
}

// tavilyRequest represents the request body for Tavily API
type tavilyRequest struct {
	APIKey            string   `json:"api_key"`
	Query             string   `json:"query"`
	SearchDepth       string   `json:"search_depth,omitempty"`        // "basic" or "advanced"
	IncludeAnswer     bool     `json:"include_answer,omitempty"`      // Include AI-generated answer
	IncludeRawContent bool     `json:"include_raw_content,omitempty"` // Include raw HTML content
	MaxResults        int      `json:"max_results,omitempty"`         // Max number of results
	IncludeDomains    []string `json:"include_domains,omitempty"`     // Limit to specific domains
	ExcludeDomains    []string `json:"exclude_domains,omitempty"`     // Exclude specific domains
}

// tavilyResponse represents the response from Tavily API
type tavilyResponse struct {
	Query   string         `json:"query"`
	Answer  string         `json:"answer,omitempty"`
	Results []tavilyResult `json:"results"`
}

// tavilyResult represents a single search result from Tavily
type tavilyResult struct {
	Title      string  `json:"title"`
	URL        string  `json:"url"`
	Content    string  `json:"content"`
	Score      float64 `json:"score"`
	RawContent string  `json:"raw_content,omitempty"`
}

// Search executes a web search using Tavily API
func (p *TavilyProvider) Search(req *types.Request) (*types.Result, error) {
	startTime := time.Now()

	// Validate API key
	if p.apiKey == "" {
		return &types.Result{
			Type:   types.SearchTypeWeb,
			Query:  req.Query,
			Source: req.Source,
			Items:  []*types.ResultItem{},
			Total:  0,
			Error:  "Tavily API key not configured",
		}, nil
	}

	// Determine max results
	maxResults := p.maxResults
	if req.Limit > 0 {
		maxResults = req.Limit
	}

	// Build request body
	tavilyReq := tavilyRequest{
		APIKey:        p.apiKey,
		Query:         req.Query,
		SearchDepth:   "basic",
		IncludeAnswer: false,
		MaxResults:    maxResults,
	}

	// Add domain restrictions if specified
	if len(req.Sites) > 0 {
		tavilyReq.IncludeDomains = req.Sites
	}

	// Execute API call
	tavilyResp, err := p.callAPI(&tavilyReq)
	if err != nil {
		return &types.Result{
			Type:     types.SearchTypeWeb,
			Query:    req.Query,
			Source:   req.Source,
			Items:    []*types.ResultItem{},
			Total:    0,
			Duration: time.Since(startTime).Milliseconds(),
			Error:    fmt.Sprintf("Tavily API error: %v", err),
		}, nil
	}

	// Convert results
	items := make([]*types.ResultItem, 0, len(tavilyResp.Results))
	for _, r := range tavilyResp.Results {
		items = append(items, &types.ResultItem{
			Type:    types.SearchTypeWeb,
			Title:   r.Title,
			Content: r.Content,
			URL:     r.URL,
			Score:   r.Score,
			Source:  req.Source,
		})
	}

	return &types.Result{
		Type:     types.SearchTypeWeb,
		Query:    req.Query,
		Source:   req.Source,
		Items:    items,
		Total:    len(items),
		Duration: time.Since(startTime).Milliseconds(),
	}, nil
}

// callAPI makes the HTTP request to Tavily API
func (p *TavilyProvider) callAPI(req *tavilyRequest) (*tavilyResponse, error) {
	// Serialize request body
	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Create HTTP request
	httpReq, err := http.NewRequest(http.MethodPost, tavilyAPIURL, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")

	// Execute request
	client := &http.Client{Timeout: tavilyAPITimeout}
	resp, err := client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	// Read response body
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	// Check status code
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(respBody))
	}

	// Parse response
	var tavilyResp tavilyResponse
	if err := json.Unmarshal(respBody, &tavilyResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &tavilyResp, nil
}
