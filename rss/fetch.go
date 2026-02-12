package rss

import (
	"compress/gzip"
	"fmt"
	"io"
	"net/http"
	"time"
)

const (
	defaultUserAgent = "Yao-Robot/1.0"
	defaultTimeout   = 30
	acceptHeader     = "application/rss+xml, application/atom+xml, application/xml, text/xml"
)

// Fetch retrieves a remote RSS/Atom feed by URL and parses it into a Feed.
// Supports gzip decompression and conditional requests (ETag / Last-Modified)
// for bandwidth-efficient polling.
func Fetch(url string, opts *FetchOptions) (*FetchResult, error) {
	if url == "" {
		return nil, fmt.Errorf("url is required")
	}
	if opts == nil {
		opts = &FetchOptions{}
	}

	userAgent := opts.UserAgent
	if userAgent == "" {
		userAgent = defaultUserAgent
	}
	timeout := opts.Timeout
	if timeout <= 0 {
		timeout = defaultTimeout
	}

	client := &http.Client{Timeout: time.Duration(timeout) * time.Second}

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %s", err.Error())
	}

	// Standard headers
	req.Header.Set("User-Agent", userAgent)
	req.Header.Set("Accept", acceptHeader)
	req.Header.Set("Accept-Encoding", "gzip")

	// Conditional request headers
	if opts.ETag != "" {
		req.Header.Set("If-None-Match", opts.ETag)
	}
	if opts.LastModified != "" {
		req.Header.Set("If-Modified-Since", opts.LastModified)
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("HTTP request failed: %s", err.Error())
	}
	defer resp.Body.Close()

	result := &FetchResult{
		StatusCode:   resp.StatusCode,
		ETag:         resp.Header.Get("ETag"),
		LastModified: resp.Header.Get("Last-Modified"),
	}

	// Handle 304 Not Modified
	if resp.StatusCode == http.StatusNotModified {
		result.NotModified = true
		return result, nil
	}

	// Reject non-200 responses
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d for %s", resp.StatusCode, url)
	}

	// Handle gzip decompression.
	// Because we manually set Accept-Encoding: "gzip" above, Go's default transport
	// does NOT auto-decompress — the Content-Encoding header is preserved, and we
	// must decompress ourselves. This is correct and intentional.
	var reader io.Reader = resp.Body
	if resp.Header.Get("Content-Encoding") == "gzip" {
		gz, err := gzip.NewReader(resp.Body)
		if err != nil {
			return nil, fmt.Errorf("failed to create gzip reader: %s", err.Error())
		}
		defer gz.Close()
		reader = gz
	}

	// Read body
	body, err := io.ReadAll(reader)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %s", err.Error())
	}

	// Parse feed using existing Parse function
	feed, err := Parse(string(body))
	if err != nil {
		return nil, fmt.Errorf("failed to parse feed: %s", err.Error())
	}

	result.Feed = feed
	return result, nil
}
