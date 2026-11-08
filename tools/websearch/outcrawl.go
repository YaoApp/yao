package websearch

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// outcrawlSearchURL is the Outcrawl search endpoint (https://outcrawl.ai).
// It is a variable so tests can point it at an httptest server.
var outcrawlSearchURL = "https://api.outcrawl.ai/v1/search"

func outcrawlSearch(apiKey, query string, limit int) []SearchResult {
	if apiKey == "" {
		return nil
	}

	payload, _ := json.Marshal(map[string]interface{}{
		"query": query,
		"limit": limit,
	})

	req, err := http.NewRequest("POST", outcrawlSearchURL, bytes.NewReader(payload))
	if err != nil {
		return []SearchResult{{Title: "Error", Content: fmt.Sprintf("outcrawl request build failed: %s", err.Error())}}
	}
	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 60 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return []SearchResult{{Title: "Error", Content: fmt.Sprintf("outcrawl request failed: %s", err.Error())}}
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return []SearchResult{{Title: "Error", Content: fmt.Sprintf("outcrawl read body failed: %s", err.Error())}}
	}

	if resp.StatusCode != http.StatusOK {
		return []SearchResult{{Title: "Error", Content: fmt.Sprintf("outcrawl HTTP %d: %s", resp.StatusCode, string(body))}}
	}

	var result struct {
		Hits []struct {
			Title   string `json:"title"`
			URL     string `json:"url"`
			Snippet string `json:"snippet"`
		} `json:"hits"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return []SearchResult{{Title: "Error", Content: fmt.Sprintf("outcrawl parse failed: %s", err.Error())}}
	}

	out := make([]SearchResult, 0, len(result.Hits))
	for _, h := range result.Hits {
		out = append(out, SearchResult{
			Title:   h.Title,
			URL:     h.URL,
			Content: h.Snippet,
		})
	}
	return out
}
