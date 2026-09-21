package webfetch

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// taoFetch calls the Tao Fetch API.
// Tao endpoint: POST /v1/fetch
func taoFetch(cfg *fetchConfig, targetURL, format string) *FetchResponse {
	if cfg.APIURL == "" || cfg.APIKey == "" {
		return &FetchResponse{
			URL:     targetURL,
			Content: "tao service not configured",
			Format:  format,
		}
	}

	if format != "markdown" && format != "html" {
		format = "markdown"
	}

	payload, _ := json.Marshal(map[string]interface{}{
		"url":    targetURL,
		"format": format,
	})

	endpoint := strings.TrimRight(cfg.APIURL, "/") + "/v1/fetch"
	req, err := http.NewRequest("POST", endpoint, bytes.NewReader(payload))
	if err != nil {
		return &FetchResponse{
			URL:     targetURL,
			Content: fmt.Sprintf("tao fetch request build failed: %s", err.Error()),
			Format:  format,
		}
	}
	req.Header.Set("Authorization", "Bearer "+cfg.APIKey)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 60 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return &FetchResponse{
			URL:     targetURL,
			Content: fmt.Sprintf("tao fetch request failed: %s", err.Error()),
			Format:  format,
		}
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, maxBodySize))
	if err != nil {
		return &FetchResponse{
			URL:     targetURL,
			Content: fmt.Sprintf("tao fetch read body failed: %s", err.Error()),
			Format:  format,
		}
	}

	if resp.StatusCode != http.StatusOK {
		return &FetchResponse{
			URL:     targetURL,
			Content: fmt.Sprintf("tao fetch HTTP %d: %s", resp.StatusCode, truncate(string(body), 200)),
			Format:  format,
		}
	}

	return parseTaoFetchResponse(body, targetURL, format)
}

// parseTaoFetchResponse handles the Tao fetch response.
// Tao returns plain text (HTML or Markdown) for fetch; tries JSON first for structured responses.
func parseTaoFetchResponse(body []byte, targetURL, format string) *FetchResponse {
	var result struct {
		Title   string `json:"title"`
		Content string `json:"content"`
	}
	if json.Unmarshal(body, &result) == nil && result.Content != "" {
		return &FetchResponse{
			URL:     targetURL,
			Title:   result.Title,
			Content: result.Content,
			Format:  format,
		}
	}

	// Plain text response (HTML or Markdown)
	return &FetchResponse{
		URL:     targetURL,
		Content: string(body),
		Format:  format,
	}
}
