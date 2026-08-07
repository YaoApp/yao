package webfetch

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"time"
)

const defaultDownloadMax = 20 << 20 // 20 MB

// DownloadBytes fetches binary content from a URL with browser simulation.
// Strategy: direct fetch with browser UA -> Brightdata proxy fallback (if configured) -> error.
// maxBytes limits response size; 0 means 20 MB default.
func DownloadBytes(rawURL string, maxBytes int64) ([]byte, error) {
	if maxBytes <= 0 {
		maxBytes = defaultDownloadMax
	}

	data, statusCode, err := directDownload(rawURL, maxBytes)
	if err == nil && statusCode == http.StatusOK {
		return data, nil
	}

	if statusCode == http.StatusForbidden || statusCode == http.StatusUnauthorized || statusCode == http.StatusTooManyRequests {
		apiKey := os.Getenv("BRIGHTDATA_API_KEY")
		zone := os.Getenv("BRIGHTDATA_ZONE")
		if apiKey != "" {
			bdData, bdErr := brightdataFetch(rawURL, apiKey, zone)
			if bdErr == nil {
				return bdData, nil
			}
		}
	}

	if err != nil {
		return nil, err
	}
	return nil, fmt.Errorf("fetch %s: HTTP %d", rawURL, statusCode)
}

func directDownload(rawURL string, maxBytes int64) ([]byte, int, error) {
	client := &http.Client{Timeout: 30 * time.Second}
	req, err := http.NewRequest(http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, 0, err
	}
	req.Header.Set("User-Agent", browserUserAgent)
	req.Header.Set("Accept", "*/*")
	req.Header.Set("Accept-Language", "en-US,en;q=0.9")

	resp, err := client.Do(req)
	if err != nil {
		return nil, 0, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, resp.StatusCode, nil
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, maxBytes))
	if err != nil {
		return nil, resp.StatusCode, fmt.Errorf("read body: %w", err)
	}
	return body, resp.StatusCode, nil
}
