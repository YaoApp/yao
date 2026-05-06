package websearch

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

func cloudSearch(cfg *searchConfig, query string, limit int) []SearchResult {
	if cfg.APIURL == "" || cfg.APIKey == "" {
		return nil
	}

	payload, _ := json.Marshal(map[string]interface{}{
		"query":       query,
		"max_results": limit,
	})

	tool := cfg.CloudTool
	if tool == "" {
		tool = "serper-search"
	}
	url := cfg.APIURL + "/v1/search/" + tool
	req, err := http.NewRequest("POST", url, bytes.NewReader(payload))
	if err != nil {
		return []SearchResult{{Title: "Error", Content: fmt.Sprintf("cloud request build failed: %s", err.Error())}}
	}
	req.Header.Set("Authorization", "Bearer "+cfg.APIKey)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return []SearchResult{{Title: "Error", Content: fmt.Sprintf("cloud request failed: %s", err.Error())}}
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return []SearchResult{{Title: "Error", Content: fmt.Sprintf("cloud read body failed: %s", err.Error())}}
	}

	if resp.StatusCode != http.StatusOK {
		return []SearchResult{{Title: "Error", Content: fmt.Sprintf("cloud HTTP %d: %s", resp.StatusCode, string(body))}}
	}

	var result struct {
		Results []struct {
			Title   string  `json:"title"`
			URL     string  `json:"url"`
			Snippet string  `json:"snippet"`
			Content string  `json:"content"`
			Score   float64 `json:"score"`
		} `json:"results"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return []SearchResult{{Title: "Error", Content: fmt.Sprintf("cloud parse failed: %s", err.Error())}}
	}

	out := make([]SearchResult, 0, len(result.Results))
	for _, r := range result.Results {
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
