package sitemap

import (
	"compress/gzip"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"time"
)

// Fetch retrieves and parses sitemap URLs for a domain, supporting offset/limit pagination.
// It first calls Discover to get the sitemap file list, then uses smart offset/limit
// to determine which files to actually download. Stream parsing ensures low memory usage.
func Fetch(domain string, opts *FetchOptions) (*FetchResult, error) {
	if opts == nil {
		opts = &FetchOptions{}
	}

	// Apply defaults
	userAgent := opts.UserAgent
	if userAgent == "" {
		userAgent = DefaultUserAgent
	}
	timeout := opts.Timeout
	if timeout <= 0 {
		timeout = DefaultTimeout
	}
	limit := opts.Limit
	if limit <= 0 || limit > MaxURLsPerFile {
		limit = MaxURLsPerFile
	}
	offset := opts.Offset
	if offset < 0 {
		offset = 0
	}

	// Step 1: Discover sitemap files
	discoverOpts := &DiscoverOptions{
		UserAgent: userAgent,
		Timeout:   timeout,
	}
	discovered, err := Discover(domain, discoverOpts)
	if err != nil {
		return nil, fmt.Errorf("discover failed: %s", err.Error())
	}

	if len(discovered.Sitemaps) == 0 {
		return &FetchResult{URLs: []URL{}, Total: 0}, nil
	}

	client := &http.Client{Timeout: time.Duration(timeout) * time.Second}

	// Step 2: Use estimated URL counts to skip files before offset
	var collected []URL
	remaining := limit
	skipped := 0 // total URLs skipped so far (via file skipping + stream skipping)
	totalPrecise := 0
	totalEstimated := 0

	for i, sitemapLink := range discovered.Sitemaps {
		if remaining <= 0 {
			// We have enough URLs. Add estimated totals for remaining files.
			for j := i; j < len(discovered.Sitemaps); j++ {
				totalEstimated += discovered.Sitemaps[j].URLCount
			}
			break
		}

		estimatedCount := sitemapLink.URLCount
		if estimatedCount <= 0 {
			estimatedCount = 1 // at least try to fetch
		}

		// Can we skip this entire file?
		if skipped+estimatedCount <= offset {
			skipped += estimatedCount
			totalEstimated += estimatedCount
			continue
		}

		// We need to stream-parse this file
		skipInFile := 0
		if skipped < offset {
			skipInFile = offset - skipped
		}

		urls, fileTotal, err := streamParseURLs(client, userAgent, sitemapLink, skipInFile, remaining)
		if err != nil {
			// Skip this file on error, use estimate for total
			totalEstimated += estimatedCount
			skipped += estimatedCount
			continue
		}

		collected = append(collected, urls...)
		remaining -= len(urls)
		skipped += skipInFile + len(urls)
		totalPrecise += fileTotal
	}

	total := totalPrecise + totalEstimated

	if collected == nil {
		collected = []URL{}
	}

	return &FetchResult{
		URLs:  collected,
		Total: total,
	}, nil
}

// streamParseURLs streams a sitemap file via HTTP GET, skipping `skip` URLs
// and collecting up to `limit` URLs. Returns the collected URLs and the actual
// total number of URLs in the file (for precise counting).
func streamParseURLs(client *http.Client, userAgent string, link SitemapLink, skip, limit int) ([]URL, int, error) {
	req, err := http.NewRequest("GET", link.URL, nil)
	if err != nil {
		return nil, 0, err
	}
	req.Header.Set("User-Agent", userAgent)

	resp, err := client.Do(req)
	if err != nil {
		return nil, 0, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, 0, fmt.Errorf("HTTP %d for %s", resp.StatusCode, link.URL)
	}

	// Handle gzip decompression.
	// Go's default HTTP transport auto-decompresses Content-Encoding: gzip and strips
	// the header. We only need manual decompression in two cases:
	// 1. The response still has Content-Encoding: gzip (transport did not handle it).
	// 2. The response body is raw gzip (e.g. .xml.gz file served without Content-Encoding),
	//    indicated by resp.Uncompressed == false AND link.Encoding hints gzip.
	var reader io.Reader = resp.Body
	needGzip := resp.Header.Get("Content-Encoding") == "gzip"
	if !needGzip && link.Encoding == "gzip" && !resp.Uncompressed {
		needGzip = true
	}
	if needGzip {
		gz, err := gzip.NewReader(resp.Body)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to create gzip reader: %s", err.Error())
		}
		defer gz.Close()
		reader = gz
	}

	// Stream parse with xml.Decoder
	decoder := xml.NewDecoder(reader)
	var collected []URL
	count := 0    // total URLs seen in this file
	skipped := 0  // URLs skipped so far
	gathered := 0 // URLs collected so far

	for {
		tok, err := decoder.Token()
		if err != nil {
			if err == io.EOF {
				break
			}
			// Tolerate partial reads if we already have enough URLs
			if gathered >= limit {
				break
			}
			return collected, count, fmt.Errorf("XML decode error: %s", err.Error())
		}

		se, ok := tok.(xml.StartElement)
		if !ok || se.Name.Local != "url" {
			continue
		}

		// Decode the <url> element
		var u URL
		if err := decoder.DecodeElement(&u, &se); err != nil {
			continue // skip malformed entries
		}
		count++

		// Skip phase
		if skipped < skip {
			skipped++
			continue
		}

		// Collect phase
		if gathered < limit {
			collected = append(collected, u)
			gathered++

			// We have enough — close the connection to stop downloading
			if gathered >= limit {
				resp.Body.Close()
				break
			}
		}
	}

	return collected, count, nil
}
