package websearch

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// taoSearch calls the Tao Search API.
// Tao endpoint: POST /v1/search
func taoSearch(cfg *searchConfig, query string, limit int) []SearchResult {
	if cfg.APIURL == "" || cfg.APIKey == "" {
		return nil
	}

	payload, _ := json.Marshal(map[string]interface{}{
		"query": query,
		"num":   limit,
	})

	url := strings.TrimRight(cfg.APIURL, "/") + "/v1/search"
	req, err := http.NewRequest("POST", url, bytes.NewReader(payload))
	if err != nil {
		return []SearchResult{{Title: "Error", Content: fmt.Sprintf("tao search request build failed: %s", err.Error())}}
	}
	req.Header.Set("Authorization", "Bearer "+cfg.APIKey)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return []SearchResult{{Title: "Error", Content: fmt.Sprintf("tao search request failed: %s", err.Error())}}
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return []SearchResult{{Title: "Error", Content: fmt.Sprintf("tao search read body failed: %s", err.Error())}}
	}

	if resp.StatusCode != http.StatusOK {
		return []SearchResult{{Title: "Error", Content: fmt.Sprintf("tao search HTTP %d: %s", resp.StatusCode, string(body))}}
	}

	return parseTaoSearchResponse(body)
}

// parseTaoSearchResponse handles the Serper-compatible response from Tao.
// Primary format: {"organic":[{"title":"...","link":"...","snippet":"..."},...]}
// Fallback: {"results":[{"title":"...","url":"...","snippet":"..."},...]}
func parseTaoSearchResponse(body []byte) []SearchResult {
	var raw map[string]json.RawMessage
	if json.Unmarshal(body, &raw) != nil {
		return []SearchResult{{Title: "Error", Content: "tao search parse failed"}}
	}

	// Try organic[] (Serper format — Tao default)
	if organic, ok := raw["organic"]; ok {
		var items []struct {
			Title   string  `json:"title"`
			Link    string  `json:"link"`
			Snippet string  `json:"snippet"`
			Score   float64 `json:"score"`
		}
		if json.Unmarshal(organic, &items) == nil && len(items) > 0 {
			out := make([]SearchResult, 0, len(items))
			for _, r := range items {
				out = append(out, SearchResult{
					Title:   r.Title,
					URL:     r.Link,
					Content: r.Snippet,
					Score:   r.Score,
				})
			}
			return out
		}
	}

	// Fallback: results[] (legacy format)
	if results, ok := raw["results"]; ok {
		var items []struct {
			Title   string  `json:"title"`
			URL     string  `json:"url"`
			Snippet string  `json:"snippet"`
			Content string  `json:"content"`
			Score   float64 `json:"score"`
		}
		if json.Unmarshal(results, &items) == nil {
			out := make([]SearchResult, 0, len(items))
			for _, r := range items {
				text := r.Snippet
				if text == "" {
					text = r.Content
				}
				out = append(out, SearchResult{
					Title:   r.Title,
					URL:     r.URL,
					Content: text,
					Score:   r.Score,
				})
			}
			return out
		}
	}

	return []SearchResult{{Title: "Error", Content: "tao search: unexpected response format"}}
}
