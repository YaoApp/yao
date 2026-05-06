package websearch

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

func serperSearch(apiKey, query string, limit int) []SearchResult {
	if apiKey == "" {
		return nil
	}

	payload, _ := json.Marshal(map[string]interface{}{
		"q":   query,
		"num": limit,
	})

	req, err := http.NewRequest("POST", "https://google.serper.dev/search", bytes.NewReader(payload))
	if err != nil {
		return []SearchResult{{Title: "Error", Content: fmt.Sprintf("serper request build failed: %s", err.Error())}}
	}
	req.Header.Set("X-API-KEY", apiKey)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return []SearchResult{{Title: "Error", Content: fmt.Sprintf("serper request failed: %s", err.Error())}}
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return []SearchResult{{Title: "Error", Content: fmt.Sprintf("serper read body failed: %s", err.Error())}}
	}

	if resp.StatusCode != http.StatusOK {
		return []SearchResult{{Title: "Error", Content: fmt.Sprintf("serper HTTP %d: %s", resp.StatusCode, string(body))}}
	}

	var result struct {
		Organic []struct {
			Title   string `json:"title"`
			Link    string `json:"link"`
			Snippet string `json:"snippet"`
		} `json:"organic"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return []SearchResult{{Title: "Error", Content: fmt.Sprintf("serper parse failed: %s", err.Error())}}
	}

	out := make([]SearchResult, 0, len(result.Organic))
	for _, r := range result.Organic {
		out = append(out, SearchResult{
			Title:   r.Title,
			URL:     r.Link,
			Content: r.Snippet,
		})
	}
	return out
}
