package websearch

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

func tavilySearch(apiKey, query string, limit int) []SearchResult {
	if apiKey == "" {
		return nil
	}

	payload, _ := json.Marshal(map[string]interface{}{
		"api_key":             apiKey,
		"query":               query,
		"max_results":         limit,
		"include_raw_content": false,
	})

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Post("https://api.tavily.com/search", "application/json", bytes.NewReader(payload))
	if err != nil {
		return []SearchResult{{Title: "Error", Content: fmt.Sprintf("tavily request failed: %s", err.Error())}}
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return []SearchResult{{Title: "Error", Content: fmt.Sprintf("tavily read body failed: %s", err.Error())}}
	}

	if resp.StatusCode != http.StatusOK {
		return []SearchResult{{Title: "Error", Content: fmt.Sprintf("tavily HTTP %d: %s", resp.StatusCode, string(body))}}
	}

	var result struct {
		Results []struct {
			Title   string  `json:"title"`
			URL     string  `json:"url"`
			Content string  `json:"content"`
			Score   float64 `json:"score"`
		} `json:"results"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return []SearchResult{{Title: "Error", Content: fmt.Sprintf("tavily parse failed: %s", err.Error())}}
	}

	out := make([]SearchResult, 0, len(result.Results))
	for _, r := range result.Results {
		out = append(out, SearchResult{
			Title:   r.Title,
			URL:     r.URL,
			Content: r.Content,
			Score:   r.Score,
		})
	}
	return out
}
