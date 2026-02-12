package sitemap

import (
	"regexp"
	"strings"
)

// reSitemapLine matches "Sitemap:" directives in robots.txt (case-insensitive).
var reSitemapLine = regexp.MustCompile(`(?im)^\s*Sitemap:\s*(.+?)\s*$`)

// ParseRobots extracts sitemap URLs from a robots.txt text content.
// It looks for lines matching "Sitemap: <url>" (case-insensitive).
// Returns a deduplicated list of sitemap URLs.
func ParseRobots(text string) []string {
	matches := reSitemapLine.FindAllStringSubmatch(text, -1)
	if len(matches) == 0 {
		return []string{}
	}

	seen := make(map[string]bool, len(matches))
	var urls []string
	for _, m := range matches {
		u := strings.TrimSpace(m[1])
		if u == "" {
			continue
		}
		if !seen[u] {
			seen[u] = true
			urls = append(urls, u)
		}
	}

	if urls == nil {
		return []string{}
	}
	return urls
}
