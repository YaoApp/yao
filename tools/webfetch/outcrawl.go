package webfetch

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

const outcrawlTimeout = 90 * time.Second

// outcrawlEndpoint is the Outcrawl scrape endpoint (https://outcrawl.ai).
// It is a variable so tests can point it at an httptest server.
var outcrawlEndpoint = "https://api.outcrawl.ai/v1/scrape"

// outcrawlFetch loads a page through Outcrawl and returns its content in the
// requested format ("markdown" or "html"). An error means the caller should
// fall back to the local fetch path.
func outcrawlFetch(targetURL, apiKey, format string) (*FetchResponse, error) {
	if format != "html" {
		format = "markdown"
	}

	payload, _ := json.Marshal(map[string]interface{}{
		"url":     targetURL,
		"formats": []string{format},
	})

	req, err := http.NewRequest(http.MethodPost, outcrawlEndpoint, bytes.NewReader(payload))
	if err != nil {
		return nil, fmt.Errorf("build outcrawl request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: outcrawlTimeout}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("outcrawl request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, maxBodySize))
	if err != nil {
		return nil, fmt.Errorf("read outcrawl response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("outcrawl HTTP %d: %s", resp.StatusCode, truncate(string(body), 200))
	}

	var result struct {
		Markdown string `json:"markdown"`
		HTML     string `json:"html"`
		Metadata struct {
			Title string `json:"title"`
		} `json:"metadata"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("parse outcrawl response: %w", err)
	}

	content := result.Markdown
	if format == "html" {
		content = result.HTML
	}
	if content == "" {
		return nil, fmt.Errorf("outcrawl returned no %s content", format)
	}

	return &FetchResponse{
		URL:     targetURL,
		Title:   result.Metadata.Title,
		Content: content,
		Format:  format,
	}, nil
}
