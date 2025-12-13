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
	serperAPIURL     = "https://google.serper.dev/search"
	serperAPITimeout = 30 * time.Second
)

// SerperProvider implements web search using Serper API (serper.dev)
type SerperProvider struct {
	apiKey     string
	maxResults int
}

// NewSerperProvider creates a new Serper provider
func NewSerperProvider(cfg *types.WebConfig) *SerperProvider {
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

	return &SerperProvider{
		apiKey:     apiKey,
		maxResults: maxResults,
	}
}

// serperRequest represents the request body for Serper API
type serperRequest struct {
	Q       string `json:"q"`              // Search query
	Num     int    `json:"num,omitempty"`  // Number of results (default: 10, max: 100)
	GL      string `json:"gl,omitempty"`   // Country code (e.g., "us", "cn")
	HL      string `json:"hl,omitempty"`   // Language code (e.g., "en", "zh-cn")
	TBS     string `json:"tbs,omitempty"`  // Time-based search (qdr:h, qdr:d, qdr:w, qdr:m, qdr:y)
	Page    int    `json:"page,omitempty"` // Page number (default: 1)
	AutoCor bool   `json:"autocorrect"`    // Auto-correct spelling
}

// serperResponse represents the response from Serper API
type serperResponse struct {
	SearchParameters serperSearchParams `json:"searchParameters"`
	Organic          []serperResult     `json:"organic"`
	AnswerBox        *serperAnswerBox   `json:"answerBox,omitempty"`
	KnowledgeGraph   *serperKnowledge   `json:"knowledgeGraph,omitempty"`
	RelatedSearches  []serperRelated    `json:"relatedSearches,omitempty"`
}

// serperSearchParams contains search parameters from response
type serperSearchParams struct {
	Q    string `json:"q"`
	Type string `json:"type"`
	GL   string `json:"gl"`
	HL   string `json:"hl"`
	Num  int    `json:"num"`
}

// serperResult represents a single organic search result
type serperResult struct {
	Title    string `json:"title"`
	Link     string `json:"link"`
	Snippet  string `json:"snippet"`
	Position int    `json:"position"`
	Date     string `json:"date,omitempty"`
}

// serperAnswerBox represents the answer box (featured snippet)
type serperAnswerBox struct {
	Title   string `json:"title,omitempty"`
	Snippet string `json:"snippet,omitempty"`
	Link    string `json:"link,omitempty"`
}

// serperKnowledge represents knowledge graph data
type serperKnowledge struct {
	Title       string `json:"title,omitempty"`
	Type        string `json:"type,omitempty"`
	Description string `json:"description,omitempty"`
}

// serperRelated represents related searches
type serperRelated struct {
	Query string `json:"query"`
}

// Search executes a web search using Serper API
func (p *SerperProvider) Search(req *types.Request) (*types.Result, error) {
	startTime := time.Now()

	// Validate API key
	if p.apiKey == "" {
		return &types.Result{
			Type:   types.SearchTypeWeb,
			Query:  req.Query,
			Source: req.Source,
			Items:  []*types.ResultItem{},
			Total:  0,
			Error:  "Serper API key not configured",
		}, nil
	}

	// Determine max results
	maxResults := p.maxResults
	if req.Limit > 0 {
		maxResults = req.Limit
	}

	// Build search query with site restrictions if specified
	query := req.Query
	if len(req.Sites) > 0 {
		// Serper uses "site:domain" syntax in query
		if len(req.Sites) == 1 {
			query = "site:" + req.Sites[0] + " " + req.Query
		} else {
			// Multiple sites: (site:domain1 OR site:domain2) query
			siteQuery := ""
			for i, site := range req.Sites {
				if i > 0 {
					siteQuery += " OR "
				}
				siteQuery += "site:" + site
			}
			query = "(" + siteQuery + ") " + req.Query
		}
	}

	// Build request body
	serperReq := serperRequest{
		Q:       query,
		Num:     maxResults,
		AutoCor: true,
	}

	// Add time range if specified
	if req.TimeRange != "" {
		serperReq.TBS = convertSerperTimeRange(req.TimeRange)
	}

	// Execute API call
	serperResp, err := p.callAPI(&serperReq)
	if err != nil {
		return &types.Result{
			Type:     types.SearchTypeWeb,
			Query:    req.Query,
			Source:   req.Source,
			Items:    []*types.ResultItem{},
			Total:    0,
			Duration: time.Since(startTime).Milliseconds(),
			Error:    fmt.Sprintf("Serper API error: %v", err),
		}, nil
	}

	// Convert results
	items := make([]*types.ResultItem, 0, len(serperResp.Organic))

	// Add answer box as first result if available
	if serperResp.AnswerBox != nil && serperResp.AnswerBox.Snippet != "" {
		items = append(items, &types.ResultItem{
			Type:    types.SearchTypeWeb,
			Title:   serperResp.AnswerBox.Title,
			Content: serperResp.AnswerBox.Snippet,
			URL:     serperResp.AnswerBox.Link,
			Score:   1.0, // Featured snippet gets highest score
			Source:  req.Source,
			Metadata: map[string]interface{}{
				"type": "answer_box",
			},
		})
	}

	// Add organic results
	for _, r := range serperResp.Organic {
		// Calculate score based on position (1st = 0.95, 2nd = 0.90, etc.)
		score := 1.0 - float64(r.Position)*0.05
		if score < 0.1 {
			score = 0.1
		}

		items = append(items, &types.ResultItem{
			Type:    types.SearchTypeWeb,
			Title:   r.Title,
			Content: r.Snippet,
			URL:     r.Link,
			Score:   score,
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

// callAPI makes the HTTP POST request to Serper API
func (p *SerperProvider) callAPI(req *serperRequest) (*serperResponse, error) {
	// Serialize request body
	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Create HTTP request
	httpReq, err := http.NewRequest(http.MethodPost, serperAPIURL, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("X-API-KEY", p.apiKey)

	// Execute request
	client := &http.Client{Timeout: serperAPITimeout}
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
	var serperResp serperResponse
	if err := json.Unmarshal(respBody, &serperResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &serperResp, nil
}

// convertSerperTimeRange converts time range to Serper tbs format
func convertSerperTimeRange(timeRange string) string {
	switch timeRange {
	case "hour":
		return "qdr:h"
	case "day":
		return "qdr:d"
	case "week":
		return "qdr:w"
	case "month":
		return "qdr:m"
	case "year":
		return "qdr:y"
	default:
		return ""
	}
}
