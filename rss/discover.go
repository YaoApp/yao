package rss

import (
	"regexp"
	"strings"
)

// Common feed URL path patterns used for heuristic URL detection.
var feedPathPatterns = []string{
	"/feed", "/rss", "/atom",
	"/feed.xml", "/rss.xml", "/atom.xml", "/index.xml",
	"/feed/", "/rss/",
	"/feed.json", // JSON Feed (for completeness)
	".rss", ".atom",
}

// Feed URL query parameter patterns.
var feedQueryPatterns = []string{
	"feed=rss", "feed=atom", "format=rss", "format=atom", "format=feed",
}

// Compiled regex patterns (initialized once).
var (
	// Pattern 1: HTML <link> tags with RSS/Atom type
	// Matches <link rel="alternate" type="application/rss+xml" href="..." title="...">
	// Handles attributes in any order, single or double quotes, and self-closing tags.
	reLinkTag = regexp.MustCompile(
		`(?i)<link\b[^>]*\btype\s*=\s*["']application/(rss|atom)\+xml["'][^>]*>`,
	)
	reLinkHref  = regexp.MustCompile(`(?i)\bhref\s*=\s*["']([^"']+)["']`)
	reLinkTitle = regexp.MustCompile(`(?i)\btitle\s*=\s*["']([^"']+)["']`)
	reLinkType  = regexp.MustCompile(`(?i)\btype\s*=\s*["']application/(rss|atom)\+xml["']`)

	// Pattern 2: Markdown links [text](url)
	reMarkdownLink = regexp.MustCompile(`\[([^\]]*)\]\((https?://[^)\s]+)\)`)

	// Pattern 3: Bare URLs in text
	reURL = regexp.MustCompile(`https?://[^\s<>"'\)\]]+`)
)

// Discover extracts feed URLs from the given text content.
// The input can be HTML (complete or partial), Markdown, or plain text.
// It uses regex-based detection (not HTML parsing) to handle all input types robustly.
//
// Detection is performed in priority order:
//  1. HTML <link> tags with RSS/Atom type attributes
//  2. Markdown links [text](url) matching feed URL patterns
//  3. Bare URLs matching common feed path/query patterns
//
// Results are deduplicated by URL and ordered by detection priority.
func Discover(text string) []FeedLink {
	if strings.TrimSpace(text) == "" {
		return nil
	}

	seen := make(map[string]bool)
	var results []FeedLink

	// Priority 1: HTML <link> tags
	linkMatches := reLinkTag.FindAllString(text, -1)
	for _, tag := range linkMatches {
		href := extractAttr(reLinkHref, tag)
		if href == "" {
			continue
		}
		if seen[href] {
			continue
		}
		seen[href] = true

		fl := FeedLink{URL: href}
		fl.Title = extractAttr(reLinkTitle, tag)

		typeMatch := reLinkType.FindStringSubmatch(tag)
		if len(typeMatch) > 1 {
			fl.Type = strings.ToLower(typeMatch[1]) // "rss" or "atom"
		}

		results = append(results, fl)
	}

	// Priority 2: Markdown links with feed-like URLs
	mdMatches := reMarkdownLink.FindAllStringSubmatch(text, -1)
	for _, m := range mdMatches {
		if len(m) < 3 {
			continue
		}
		title, url := m[1], m[2]
		if seen[url] {
			continue
		}
		if !looksLikeFeedURL(url) {
			continue
		}
		seen[url] = true
		results = append(results, FeedLink{
			URL:   url,
			Title: strings.TrimSpace(title),
			Type:  guessTypeFromURL(url),
		})
	}

	// Priority 3: Bare URLs matching feed patterns
	urlMatches := reURL.FindAllString(text, -1)
	for _, url := range urlMatches {
		// Clean trailing punctuation that may be part of surrounding text
		url = strings.TrimRight(url, ".,;:!?")
		if seen[url] {
			continue
		}
		if !looksLikeFeedURL(url) {
			continue
		}
		seen[url] = true
		results = append(results, FeedLink{
			URL:  url,
			Type: guessTypeFromURL(url),
		})
	}

	return results
}

// looksLikeFeedURL checks whether a URL matches common feed path or query patterns.
func looksLikeFeedURL(url string) bool {
	lower := strings.ToLower(url)

	for _, p := range feedPathPatterns {
		if strings.Contains(lower, p) {
			return true
		}
	}

	for _, p := range feedQueryPatterns {
		if strings.Contains(lower, p) {
			return true
		}
	}

	return false
}

// guessTypeFromURL attempts to determine the feed type from URL patterns.
// Returns "rss", "atom", or empty string if undetermined.
func guessTypeFromURL(url string) string {
	lower := strings.ToLower(url)

	if strings.Contains(lower, "atom") {
		return "atom"
	}
	if strings.Contains(lower, "rss") {
		return "rss"
	}

	// Generic feed paths — cannot determine type
	return ""
}

// extractAttr extracts the first capture group from a regex match on the input string.
func extractAttr(re *regexp.Regexp, input string) string {
	m := re.FindStringSubmatch(input)
	if len(m) > 1 {
		return strings.TrimSpace(m[1])
	}
	return ""
}
