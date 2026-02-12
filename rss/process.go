package rss

import (
	"github.com/yaoapp/gou/process"
	"github.com/yaoapp/kun/exception"
)

func init() {
	process.RegisterGroup("rss", map[string]process.Handler{
		"parse":    ProcessParse,
		"validate": ProcessValidate,
		"discover": ProcessDiscover,
		"build":    ProcessBuild,
		"fetch":    ProcessFetch,
	})
}

// ProcessParse handles the rss.Parse process.
// Parses an RSS 2.0 or Atom 1.0 XML string into a unified Feed object.
// Auto-detects format (RSS 2.0, Atom 1.0) and extracts Podcast/iTunes metadata if present.
//
// Args:
//   - data string - The feed XML string to parse
//
// Returns: Feed object (map representation)
//
// Usage:
//
//	var feed = Process("rss.Parse", xmlString)
//	// feed.format → "rss2.0" or "atom1.0"
//	// feed.title → "My Blog"
//	// feed.items → [{title: "Post 1", ...}, ...]
//	// feed.podcast → {author: "...", ...} (nil for non-podcast feeds)
func ProcessParse(p *process.Process) interface{} {
	p.ValidateArgNums(1)
	data := p.ArgsString(0)

	feed, err := Parse(data)
	if err != nil {
		exception.New("rss.parse error: %s", 500, err).Throw()
	}
	return feed
}

// ProcessValidate handles the rss.Validate process.
// Checks whether the input string is a valid RSS 2.0 or Atom 1.0 feed.
//
// Args:
//   - data string - The feed XML string to validate
//
// Returns:
//   - true (bool) if the feed is valid
//   - error description string if invalid (AI-friendly message)
//
// Usage:
//
//	var result = Process("rss.Validate", xmlString)
//	if (result !== true) {
//	    console.log("Invalid feed: " + result)
//	}
func ProcessValidate(p *process.Process) interface{} {
	p.ValidateArgNums(1)
	data := p.ArgsString(0)

	err := Validate(data)
	if err != nil {
		return err.Error()
	}
	return true
}

// ProcessBuild handles the rss.Build process.
// Generates an XML feed document from a Feed object.
//
// Args:
//   - feed map - Feed object (same structure as rss.Parse output)
//   - format string (optional) - Output format: "rss" (default) or "atom"
//
// Returns: XML string
//
// Usage:
//
//	// Build RSS 2.0 (default)
//	var xml = Process("rss.Build", feedObj)
//
//	// Build Atom 1.0
//	var xml = Process("rss.Build", feedObj, "atom")
//
//	// Round-trip: parse then rebuild
//	var feed = Process("rss.Parse", originalXML)
//	var rebuilt = Process("rss.Build", feed, "rss")
func ProcessBuild(p *process.Process) interface{} {
	p.ValidateArgNums(1)

	feedData := p.Args[0]
	feed, err := mapToFeed(feedData)
	if err != nil {
		exception.New("rss.build error: %s", 500, err).Throw()
	}

	format := ""
	if len(p.Args) > 1 {
		format = p.ArgsString(1)
	}

	result, err := Build(feed, format)
	if err != nil {
		exception.New("rss.build error: %s", 500, err).Throw()
	}
	return result
}

// ProcessDiscover handles the rss.Discover process.
// Extracts feed URLs from HTML, Markdown, or plain text content using regex-based detection.
// Does not perform any network requests.
//
// Args:
//   - text string - The text content to scan for feed URLs
//
// Returns: array of FeedLink objects [{url, title, type}, ...]
//
// Usage:
//
//	var links = Process("rss.Discover", htmlString)
//	// links → [{url: "https://example.com/feed.xml", title: "My Blog", type: "rss"}, ...]
func ProcessDiscover(p *process.Process) interface{} {
	p.ValidateArgNums(1)
	text := p.ArgsString(0)

	links := Discover(text)
	if links == nil {
		return []FeedLink{}
	}
	return links
}

// ProcessFetch handles the rss.Fetch process.
// Fetches a remote RSS/Atom feed by URL and returns the parsed Feed along with
// HTTP metadata (ETag, Last-Modified) for conditional polling.
// Supports gzip decompression and conditional requests (If-None-Match / If-Modified-Since).
//
// Args:
//   - url string - The feed URL to fetch
//   - options map (optional) - {user_agent, timeout, etag, last_modified}
//
// Returns: FetchResult {feed, status_code, etag, last_modified, not_modified}
//
// Usage:
//
//	// First fetch
//	var result = Process("rss.Fetch", "https://example.com/feed.xml")
//	// result.feed.title → "My Blog"
//	// result.etag → "abc123"
//	// result.last_modified → "Wed, 01 Jan 2025 00:00:00 GMT"
//
//	// Subsequent polling with conditional request (saves bandwidth)
//	var result2 = Process("rss.Fetch", "https://example.com/feed.xml", {
//	    etag: result.etag,
//	    last_modified: result.last_modified
//	})
//	if (result2.not_modified) {
//	    // Feed unchanged, skip processing
//	}
func ProcessFetch(p *process.Process) interface{} {
	p.ValidateArgNums(1)
	url := p.ArgsString(0)

	var opts *FetchOptions
	if len(p.Args) > 1 {
		o, err := mapToFetchOptions(p.Args[1])
		if err != nil {
			exception.New("rss.fetch error: %s", 500, err).Throw()
		}
		opts = o
	}

	result, err := Fetch(url, opts)
	if err != nil {
		exception.New("rss.fetch error: %s", 500, err).Throw()
	}
	return result
}
