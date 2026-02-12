package sitemap

import (
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"time"
)

// Discover finds all sitemap files for a given domain.
// It checks robots.txt first, falls back to the well-known /sitemap.xml,
// then recursively expands sitemapindex files (up to MaxDiscoverDepth).
func Discover(domain string, opts *DiscoverOptions) (*DiscoverResult, error) {
	if opts == nil {
		opts = &DiscoverOptions{}
	}
	userAgent := opts.UserAgent
	if userAgent == "" {
		userAgent = DefaultUserAgent
	}
	timeout := opts.Timeout
	if timeout <= 0 {
		timeout = DefaultTimeout
	}

	client := &http.Client{Timeout: time.Duration(timeout) * time.Second}

	// Step 1: GET robots.txt
	robotsURL := fmt.Sprintf("https://%s/robots.txt", domain)
	robotsBody, _ := httpGetBody(client, robotsURL, userAgent)
	candidates := ParseRobots(robotsBody)

	// Step 2: fallback to well-known path
	if len(candidates) == 0 {
		candidates = []string{fmt.Sprintf("https://%s/sitemap.xml", domain)}
	}

	// Step 3-4: classify each candidate, expand indexes
	var leafLinks []SitemapLink
	for _, url := range candidates {
		links, err := classifyAndExpand(client, userAgent, url, "robots.txt", 0)
		if err != nil {
			continue // skip unreachable sitemaps
		}
		leafLinks = append(leafLinks, links...)
	}

	// Calculate total estimated URLs
	totalURLs := 0
	for _, link := range leafLinks {
		totalURLs += link.URLCount
	}

	if leafLinks == nil {
		leafLinks = []SitemapLink{}
	}

	return &DiscoverResult{
		Sitemaps:  leafLinks,
		TotalURLs: totalURLs,
	}, nil
}

// classifyAndExpand fetches a sitemap URL to determine its type (urlset or sitemapindex).
// It uses io.TeeReader to buffer the response while detecting the root element,
// so we can re-parse sitemapindex content without a second GET request.
// For urlset at Level 0: metadata comes from GET response headers (no extra HEAD).
// For urlset at Level 1+: streaming detect then HEAD for metadata.
// Recursively expands sitemapindex files up to MaxDiscoverDepth.
func classifyAndExpand(client *http.Client, userAgent, url, source string, depth int) ([]SitemapLink, error) {
	if depth > MaxDiscoverDepth {
		return nil, fmt.Errorf("max discover depth exceeded")
	}

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", userAgent)

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d for %s", resp.StatusCode, url)
	}

	if depth == 0 {
		// Level 0: read full body (usually small: sitemapindex or small urlset).
		// We need the body for sitemapindex parsing; for urlset we just need the type.
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return nil, fmt.Errorf("failed to read %s: %s", url, err.Error())
		}

		rootName, err := detectFormat(body)
		if err != nil {
			return nil, fmt.Errorf("failed to detect format for %s: %s", url, err.Error())
		}

		switch rootName {
		case "urlset":
			link := SitemapLink{URL: url, Source: source}
			fillMetadataFromHeaders(&link, resp)
			link.URLCount = estimateURLCount(link.ContentSize, link.Encoding)
			return []SitemapLink{link}, nil

		case "sitemapindex":
			result, err := parseSitemapIndex(body)
			if err != nil {
				return nil, err
			}
			var allLinks []SitemapLink
			for _, entry := range result.Sitemaps {
				childLinks, err := classifyAndExpand(client, userAgent, entry.Loc, "index", depth+1)
				if err != nil {
					continue
				}
				allLinks = append(allLinks, childLinks...)
			}
			return allLinks, nil

		default:
			return nil, fmt.Errorf("unexpected root element <%s> in %s", rootName, url)
		}
	}

	// Level 1+: streaming detect — read only until root element is found, then close.
	// This avoids downloading multi-MB urlset files just to classify them.
	decoder := xml.NewDecoder(resp.Body)
	rootName, err := detectRootElement(decoder)
	if err != nil {
		return nil, fmt.Errorf("failed to detect format for %s: %s", url, err.Error())
	}
	// Close the body immediately to stop downloading
	resp.Body.Close()

	switch rootName {
	case "urlset":
		link := SitemapLink{URL: url, Source: source}
		// Use HEAD to get accurate metadata without re-downloading
		fillMetadataFromHEAD(client, userAgent, &link)
		link.URLCount = estimateURLCount(link.ContentSize, link.Encoding)
		return []SitemapLink{link}, nil

	case "sitemapindex":
		// Rare: nested sitemapindex. Need to re-fetch full body to parse children.
		fullBody, err := httpGetBody(client, url, userAgent)
		if err != nil {
			return nil, fmt.Errorf("failed to re-fetch sitemapindex %s: %s", url, err.Error())
		}
		result, err := parseSitemapIndex([]byte(fullBody))
		if err != nil {
			return nil, err
		}
		var allLinks []SitemapLink
		for _, entry := range result.Sitemaps {
			childLinks, err := classifyAndExpand(client, userAgent, entry.Loc, "index", depth+1)
			if err != nil {
				continue
			}
			allLinks = append(allLinks, childLinks...)
		}
		return allLinks, nil

	default:
		return nil, fmt.Errorf("unexpected root element <%s> in %s", rootName, url)
	}
}

// detectRootElement reads XML tokens until it finds the first StartElement
// and returns its local name.
func detectRootElement(decoder *xml.Decoder) (string, error) {
	for {
		tok, err := decoder.Token()
		if err != nil {
			return "", fmt.Errorf("failed to read XML token: %s", err.Error())
		}
		if se, ok := tok.(xml.StartElement); ok {
			return se.Name.Local, nil
		}
	}
}

// fillMetadataFromHeaders extracts sitemap metadata from HTTP response headers.
func fillMetadataFromHeaders(link *SitemapLink, resp *http.Response) {
	link.ContentSize = resp.ContentLength
	link.Encoding = resp.Header.Get("Content-Encoding")
	link.LastModified = resp.Header.Get("Last-Modified")
	link.ETag = resp.Header.Get("ETag")
}

// fillMetadataFromHEAD performs a HEAD request and fills metadata into the SitemapLink.
func fillMetadataFromHEAD(client *http.Client, userAgent string, link *SitemapLink) {
	req, err := http.NewRequest("HEAD", link.URL, nil)
	if err != nil {
		return
	}
	req.Header.Set("User-Agent", userAgent)

	resp, err := client.Do(req)
	if err != nil {
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		fillMetadataFromHeaders(link, resp)
	}
}

// estimateURLCount estimates the number of URLs in a sitemap file based on
// Content-Length. Assumes ~300 bytes per URL for uncompressed XML,
// and a 5x compression ratio for gzip.
func estimateURLCount(contentSize int64, encoding string) int {
	if contentSize <= 0 {
		return 0
	}

	bytesPerURL := int64(300)
	effectiveSize := contentSize

	if encoding == "gzip" || encoding == "br" {
		effectiveSize = contentSize * 5 // assume 5x decompression ratio
	}

	count := int(effectiveSize / bytesPerURL)
	if count < 1 {
		count = 1
	}
	return count
}

// httpGetBody performs a GET request and returns the response body as a string.
// Returns empty string and error on failure.
func httpGetBody(client *http.Client, url, userAgent string) (string, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", userAgent)

	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("HTTP %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	return string(body), nil
}
