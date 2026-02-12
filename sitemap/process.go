package sitemap

import (
	"github.com/yaoapp/gou/process"
	"github.com/yaoapp/kun/exception"
)

func init() {
	process.RegisterGroup("sitemap", map[string]process.Handler{
		"parse":       processParse,
		"validate":    processValidate,
		"parserobo":   processParseRobots,
		"discover":    processDiscover,
		"fetch":       processFetch,
		"build.open":  processBuildOpen,
		"build.write": processBuildWrite,
		"build.close": processBuildClose,
	})
}

// processParse handles the sitemap.Parse process.
// Parses a sitemap XML string and returns a unified ParseResult.
// Auto-detects <urlset> or <sitemapindex> format.
//
// Args:
//   - data string - The sitemap XML string to parse
//
// Returns: ParseResult {type, urls, sitemaps}
//
// Usage:
//
//	var result = Process("sitemap.Parse", xmlString)
//	// result.type → "urlset" or "sitemapindex"
//	// result.urls → [{loc: "https://example.com/page1", ...}, ...]
//	// result.sitemaps → [{loc: "https://example.com/sitemap1.xml", ...}, ...]
func processParse(p *process.Process) interface{} {
	p.ValidateArgNums(1)
	data := p.ArgsString(0)

	result, err := Parse(data)
	if err != nil {
		exception.New("sitemap.parse error: %s", 500, err).Throw()
	}
	return result
}

// processValidate handles the sitemap.Validate process.
// Checks whether the input string is a valid sitemap XML.
//
// Args:
//   - data string - The sitemap XML string to validate
//
// Returns:
//   - true (bool) if the sitemap is valid
//   - error description string if invalid (AI-friendly message)
//
// Usage:
//
//	var result = Process("sitemap.Validate", xmlString)
//	if (result !== true) {
//	    console.log("Invalid sitemap: " + result)
//	}
func processValidate(p *process.Process) interface{} {
	p.ValidateArgNums(1)
	data := p.ArgsString(0)

	err := Validate(data)
	if err != nil {
		return err.Error()
	}
	return true
}

// processParseRobots handles the sitemap.ParseRobo process.
// Extracts sitemap URLs from robots.txt content. Pure text parsing, no HTTP.
//
// Args:
//   - text string - The robots.txt content
//
// Returns: array of sitemap URL strings
//
// Usage:
//
//	var urls = Process("sitemap.ParseRobo", robotsTxtContent)
//	// urls → ["https://example.com/sitemap.xml", "https://example.com/sitemap2.xml"]
func processParseRobots(p *process.Process) interface{} {
	p.ValidateArgNums(1)
	text := p.ArgsString(0)

	urls := ParseRobots(text)
	return urls
}

// processDiscover handles the sitemap.Discover process.
// Discovers sitemap files for a given domain via robots.txt and well-known paths.
// Recursively expands sitemapindex files. Minimizes bandwidth usage.
//
// Args:
//   - domain string - The domain to discover sitemaps for (e.g. "example.com")
//   - options map (optional) - {user_agent, timeout}
//
// Returns: DiscoverResult {sitemaps: [{url, source, url_count, ...}], total_urls}
//
// Usage:
//
//	var result = Process("sitemap.Discover", "example.com")
//	// result.sitemaps → [{url: "https://example.com/sitemap.xml", url_count: 500, ...}]
//	// result.total_urls → 500
//
//	// With options
//	var result = Process("sitemap.Discover", "example.com", {user_agent: "MyBot/1.0", timeout: 60})
func processDiscover(p *process.Process) interface{} {
	p.ValidateArgNums(1)
	domain := p.ArgsString(0)

	var opts *DiscoverOptions
	if len(p.Args) > 1 {
		o, err := mapToDiscoverOptions(p.Args[1])
		if err != nil {
			exception.New("sitemap.discover error: %s", 500, err).Throw()
		}
		opts = o
	}

	result, err := Discover(domain, opts)
	if err != nil {
		exception.New("sitemap.discover error: %s", 500, err).Throw()
	}
	return result
}

// processFetch handles the sitemap.Fetch process.
// Fetches and parses URLs from sitemaps for a domain with pagination support.
// Uses Discover internally and supports offset/limit for large sitemaps.
//
// Args:
//   - domain string - The domain to fetch sitemaps for (e.g. "example.com")
//   - options map (optional) - {offset, limit, user_agent, timeout}
//
// Returns: FetchResult {urls: [{loc, lastmod, ...}], total}
//
// Usage:
//
//	// Fetch first page
//	var page1 = Process("sitemap.Fetch", "example.com", {limit: 100})
//	// page1.urls → [{loc: "https://example.com/page1", ...}, ...]
//	// page1.total → 5000
//
//	// Fetch second page
//	var page2 = Process("sitemap.Fetch", "example.com", {offset: 100, limit: 100})
func processFetch(p *process.Process) interface{} {
	p.ValidateArgNums(1)
	domain := p.ArgsString(0)

	var opts *FetchOptions
	if len(p.Args) > 1 {
		o, err := mapToFetchOptions(p.Args[1])
		if err != nil {
			exception.New("sitemap.fetch error: %s", 500, err).Throw()
		}
		opts = o
	}

	result, err := Fetch(domain, opts)
	if err != nil {
		exception.New("sitemap.fetch error: %s", 500, err).Throw()
	}
	return result
}

// processBuildOpen handles the sitemap.Build.Open process.
// Opens a new sitemap writer and returns a UUID handle.
//
// Args:
//   - options map - {dir: "/path/to/output", base_url: "https://example.com"}
//
// Returns: handle string (UUID)
//
// Usage:
//
//	var handle = Process("sitemap.Build.Open", {
//	    dir: "/data/sitemaps",
//	    base_url: "https://example.com"
//	})
func processBuildOpen(p *process.Process) interface{} {
	p.ValidateArgNums(1)

	opts, err := mapToBuildOptions(p.Args[0])
	if err != nil {
		exception.New("sitemap.build.open error: %s", 500, err).Throw()
	}

	handle, err := BuildOpen(opts)
	if err != nil {
		exception.New("sitemap.build.open error: %s", 500, err).Throw()
	}
	return handle
}

// processBuildWrite handles the sitemap.Build.Write process.
// Writes a batch of URLs to the open sitemap writer.
// Automatically splits into new files when 50,000 URLs per file is reached.
//
// Args:
//   - handle string - The UUID handle from Build.Open
//   - urls array - [{loc: "...", lastmod: "...", images: [...], ...}, ...]
//
// Returns: nil
//
// Usage:
//
//	Process("sitemap.Build.Write", handle, [
//	    {loc: "https://example.com/page1", lastmod: "2025-01-01", priority: "0.8"},
//	    {loc: "https://example.com/page2", changefreq: "daily"},
//	])
func processBuildWrite(p *process.Process) interface{} {
	p.ValidateArgNums(2)
	handle := p.ArgsString(0)

	urls, err := mapToURLs(p.Args[1])
	if err != nil {
		exception.New("sitemap.build.write error: %s", 500, err).Throw()
	}

	if err := BuildWrite(handle, urls); err != nil {
		exception.New("sitemap.build.write error: %s", 500, err).Throw()
	}
	return nil
}

// processBuildClose handles the sitemap.Build.Close process.
// Finalizes the sitemap output, generates index if needed, cleans up the handle.
//
// Args:
//   - handle string - The UUID handle from Build.Open
//
// Returns: BuildResult {index, files, total}
//
// Usage:
//
//	var result = Process("sitemap.Build.Close", handle)
//	// result.index → "/data/sitemaps/sitemap_index.xml" (empty if single file)
//	// result.files → ["/data/sitemaps/sitemap_1.xml", "/data/sitemaps/sitemap_2.xml"]
//	// result.total → 75000
func processBuildClose(p *process.Process) interface{} {
	p.ValidateArgNums(1)
	handle := p.ArgsString(0)

	result, err := BuildClose(handle)
	if err != nil {
		exception.New("sitemap.build.close error: %s", 500, err).Throw()
	}
	return result
}
