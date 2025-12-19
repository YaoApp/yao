package web

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"time"

	"github.com/yaoapp/yao/agent/search/types"
)

const (
	serpAPIURL     = "https://serpapi.com/search.json"
	serpAPITimeout = 30 * time.Second
)

// SerpAPIProvider implements web search using SerpAPI (supports multiple search engines)
type SerpAPIProvider struct {
	apiKey     string
	maxResults int
	engine     string // Search engine: "google", "bing", "baidu", "yandex", "duckduckgo", etc.
}

// NewSerpAPIProvider creates a new SerpAPI provider
func NewSerpAPIProvider(cfg *types.WebConfig) *SerpAPIProvider {
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

	engine := "google" // Default to Google
	if cfg != nil && cfg.Engine != "" {
		engine = cfg.Engine
	}

	return &SerpAPIProvider{
		apiKey:     apiKey,
		maxResults: maxResults,
		engine:     engine,
	}
}

// serpAPIResponse represents the response from SerpAPI
type serpAPIResponse struct {
	SearchMetadata    serpAPIMetadata   `json:"search_metadata"`
	SearchParameters  serpAPIParams     `json:"search_parameters"`
	SearchInformation serpAPIInfo       `json:"search_information"`
	OrganicResults    []serpAPIResult   `json:"organic_results"`
	AnswerBox         *serpAPIAnswerBox `json:"answer_box,omitempty"`
	KnowledgeGraph    *serpAPIKnowledge `json:"knowledge_graph,omitempty"`
	RelatedSearches   []serpAPIRelated  `json:"related_searches,omitempty"`
	RelatedQuestions  []serpAPIQuestion `json:"related_questions,omitempty"`
}

// serpAPIMetadata contains metadata from response
type serpAPIMetadata struct {
	ID             string  `json:"id"`
	Status         string  `json:"status"`
	CreatedAt      string  `json:"created_at"`
	ProcessedAt    string  `json:"processed_at"`
	TotalTimeTaken float64 `json:"total_time_taken"`
}

// serpAPIParams contains search parameters from response
type serpAPIParams struct {
	Engine       string `json:"engine"`
	Q            string `json:"q"`
	Location     string `json:"location_used"`
	GoogleDomain string `json:"google_domain"`
	HL           string `json:"hl"`
	GL           string `json:"gl"`
	Device       string `json:"device"`
}

// serpAPIInfo contains search information
type serpAPIInfo struct {
	QueryDisplayed      string  `json:"query_displayed"`
	TotalResults        int64   `json:"total_results"`
	TimeTakenDisplayed  float64 `json:"time_taken_displayed"`
	OrganicResultsState string  `json:"organic_results_state"`
}

// serpAPIResult represents a single organic search result
type serpAPIResult struct {
	Position       int    `json:"position"`
	Title          string `json:"title"`
	Link           string `json:"link"`
	RedirectLink   string `json:"redirect_link,omitempty"`
	DisplayedLink  string `json:"displayed_link"`
	Snippet        string `json:"snippet"`
	Date           string `json:"date,omitempty"`
	CachedPageLink string `json:"cached_page_link,omitempty"`
}

// serpAPIAnswerBox represents the answer box (featured snippet)
type serpAPIAnswerBox struct {
	Type    string `json:"type,omitempty"`
	Title   string `json:"title,omitempty"`
	Snippet string `json:"snippet,omitempty"`
	Link    string `json:"link,omitempty"`
}

// serpAPIKnowledge represents knowledge graph data
type serpAPIKnowledge struct {
	Title       string      `json:"title,omitempty"`
	Type        interface{} `json:"type,omitempty"` // Can be string or object depending on query
	Description string      `json:"description,omitempty"`
}

// serpAPIRelated represents related searches
type serpAPIRelated struct {
	Query string `json:"query"`
	Link  string `json:"link"`
}

// serpAPIQuestion represents related questions (People Also Ask)
type serpAPIQuestion struct {
	Question string `json:"question"`
	Snippet  string `json:"snippet,omitempty"`
	Title    string `json:"title,omitempty"`
	Link     string `json:"link,omitempty"`
}

// Search executes a web search using SerpAPI
func (p *SerpAPIProvider) Search(req *types.Request) (*types.Result, error) {
	startTime := time.Now()

	// Validate API key
	if p.apiKey == "" {
		return &types.Result{
			Type:   types.SearchTypeWeb,
			Query:  req.Query,
			Source: req.Source,
			Items:  []*types.ResultItem{},
			Total:  0,
			Error:  "SerpAPI API key not configured",
		}, nil
	}

	// Determine max results
	maxResults := p.maxResults
	if req.Limit > 0 {
		maxResults = req.Limit
	}

	// Build query parameters
	params := url.Values{}
	params.Set("engine", p.engine)
	params.Set("api_key", p.apiKey)
	params.Set("num", fmt.Sprintf("%d", maxResults))

	// Build search query with site restrictions if specified
	query := req.Query
	if len(req.Sites) > 0 {
		siteQuery := ""
		for i, site := range req.Sites {
			if i > 0 {
				siteQuery += " OR "
			}
			siteQuery += "site:" + site
		}
		query = "(" + siteQuery + ") " + req.Query
	}
	params.Set("q", query)

	// Add time range if specified (tbs parameter)
	if req.TimeRange != "" {
		tbs := convertSerpAPITimeRange(req.TimeRange)
		if tbs != "" {
			params.Set("tbs", tbs)
		}
	}

	// Execute API call
	serpResp, err := p.callAPI(params)
	if err != nil {
		return &types.Result{
			Type:     types.SearchTypeWeb,
			Query:    req.Query,
			Source:   req.Source,
			Items:    []*types.ResultItem{},
			Total:    0,
			Duration: time.Since(startTime).Milliseconds(),
			Error:    fmt.Sprintf("SerpAPI error: %v", err),
		}, nil
	}

	// Convert results
	items := make([]*types.ResultItem, 0, len(serpResp.OrganicResults))

	// Add answer box as first result if available
	if serpResp.AnswerBox != nil && serpResp.AnswerBox.Snippet != "" {
		items = append(items, &types.ResultItem{
			Type:    types.SearchTypeWeb,
			Title:   serpResp.AnswerBox.Title,
			Content: serpResp.AnswerBox.Snippet,
			URL:     serpResp.AnswerBox.Link,
			Score:   1.0, // Featured snippet gets highest score
			Source:  req.Source,
			Metadata: map[string]interface{}{
				"type": "answer_box",
			},
		})
	}

	// Add organic results
	for _, r := range serpResp.OrganicResults {
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

// callAPI makes the HTTP GET request to SerpAPI
func (p *SerpAPIProvider) callAPI(params url.Values) (*serpAPIResponse, error) {
	// Build URL with query parameters
	reqURL := serpAPIURL + "?" + params.Encode()

	// Create HTTP request
	httpReq, err := http.NewRequest(http.MethodGet, reqURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Execute request
	client := &http.Client{Timeout: serpAPITimeout}
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
	var serpResp serpAPIResponse
	if err := json.Unmarshal(respBody, &serpResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &serpResp, nil
}

// convertSerpAPITimeRange converts time range to SerpAPI tbs format
func convertSerpAPITimeRange(timeRange string) string {
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
