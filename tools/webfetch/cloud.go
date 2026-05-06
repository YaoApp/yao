package webfetch

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

func cloudFetch(cfg *fetchConfig, url, format string) *FetchResponse {
	if cfg.APIURL == "" || cfg.APIKey == "" {
		return &FetchResponse{
			URL:     url,
			Content: "cloud service not configured",
			Format:  format,
		}
	}

	if format != "markdown" && format != "html" {
		format = "markdown"
	}

	payload, _ := json.Marshal(map[string]string{
		"url": url,
	})

	endpoint := cfg.APIURL + "/v1/scrape/" + format
	req, err := http.NewRequest("POST", endpoint, bytes.NewReader(payload))
	if err != nil {
		return &FetchResponse{
			URL:     url,
			Content: fmt.Sprintf("cloud request build failed: %s", err.Error()),
			Format:  format,
		}
	}
	req.Header.Set("Authorization", "Bearer "+cfg.APIKey)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 60 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return &FetchResponse{
			URL:     url,
			Content: fmt.Sprintf("cloud request failed: %s", err.Error()),
			Format:  format,
		}
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, maxBodySize))
	if err != nil {
		return &FetchResponse{
			URL:     url,
			Content: fmt.Sprintf("cloud read body failed: %s", err.Error()),
			Format:  format,
		}
	}

	if resp.StatusCode != http.StatusOK {
		return &FetchResponse{
			URL:     url,
			Content: fmt.Sprintf("cloud HTTP %d: %s", resp.StatusCode, truncate(string(body), 200)),
			Format:  format,
		}
	}

	var result struct {
		Title   string `json:"title"`
		Content string `json:"content"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return &FetchResponse{
			URL:     url,
			Title:   "",
			Content: string(body),
			Format:  format,
		}
	}

	return &FetchResponse{
		URL:     url,
		Title:   result.Title,
		Content: result.Content,
		Format:  format,
	}
}
