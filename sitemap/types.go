package sitemap

import (
	"encoding/xml"
	"os"
	"sync"
)

// ==================== Sitemap URL & Extensions ====================

// URL represents a single page entry in a sitemap <urlset>.
type URL struct {
	XMLName    xml.Name `json:"-"                    xml:"url"`
	Loc        string   `json:"loc"                  xml:"loc"`
	LastMod    string   `json:"lastmod,omitempty"     xml:"lastmod,omitempty"`
	ChangeFreq string   `json:"changefreq,omitempty"  xml:"changefreq,omitempty"`
	Priority   string   `json:"priority,omitempty"    xml:"priority,omitempty"`
	Images     []Image  `json:"images,omitempty"      xml:"http://www.google.com/schemas/sitemap-image/1.1 image,omitempty"`
	Videos     []Video  `json:"videos,omitempty"      xml:"http://www.google.com/schemas/sitemap-video/1.1 video,omitempty"`
	News       *News    `json:"news,omitempty"        xml:"http://www.google.com/schemas/sitemap-news/0.9 news,omitempty"`
}

// Image represents a Google image sitemap extension entry.
// Namespace: http://www.google.com/schemas/sitemap-image/1.1
type Image struct {
	XMLName xml.Name `json:"-"                    xml:"http://www.google.com/schemas/sitemap-image/1.1 image"`
	Loc     string   `json:"loc"                  xml:"http://www.google.com/schemas/sitemap-image/1.1 loc"`
	Caption string   `json:"caption,omitempty"     xml:"http://www.google.com/schemas/sitemap-image/1.1 caption,omitempty"`
	Title   string   `json:"title,omitempty"       xml:"http://www.google.com/schemas/sitemap-image/1.1 title,omitempty"`
	License string   `json:"license,omitempty"     xml:"http://www.google.com/schemas/sitemap-image/1.1 license,omitempty"`
}

// Video represents a Google video sitemap extension entry.
// Namespace: http://www.google.com/schemas/sitemap-video/1.1
type Video struct {
	XMLName         xml.Name `json:"-"                          xml:"http://www.google.com/schemas/sitemap-video/1.1 video"`
	ThumbnailLoc    string   `json:"thumbnail_loc"              xml:"http://www.google.com/schemas/sitemap-video/1.1 thumbnail_loc"`
	Title           string   `json:"title"                      xml:"http://www.google.com/schemas/sitemap-video/1.1 title"`
	Description     string   `json:"description"                xml:"http://www.google.com/schemas/sitemap-video/1.1 description"`
	ContentLoc      string   `json:"content_loc,omitempty"      xml:"http://www.google.com/schemas/sitemap-video/1.1 content_loc,omitempty"`
	PlayerLoc       string   `json:"player_loc,omitempty"       xml:"http://www.google.com/schemas/sitemap-video/1.1 player_loc,omitempty"`
	Duration        int      `json:"duration,omitempty"         xml:"http://www.google.com/schemas/sitemap-video/1.1 duration,omitempty"`
	PublicationDate string   `json:"publication_date,omitempty" xml:"http://www.google.com/schemas/sitemap-video/1.1 publication_date,omitempty"`
}

// News represents a Google news sitemap extension entry.
// Namespace: http://www.google.com/schemas/sitemap-news/0.9
type News struct {
	XMLName         xml.Name    `json:"-"                          xml:"http://www.google.com/schemas/sitemap-news/0.9 news"`
	Publication     Publication `json:"publication"                xml:"http://www.google.com/schemas/sitemap-news/0.9 publication"`
	PublicationDate string      `json:"publication_date"           xml:"http://www.google.com/schemas/sitemap-news/0.9 publication_date"`
	Title           string      `json:"title"                      xml:"http://www.google.com/schemas/sitemap-news/0.9 title"`
	Keywords        string      `json:"keywords,omitempty"         xml:"http://www.google.com/schemas/sitemap-news/0.9 keywords,omitempty"`
}

// Publication identifies the news publication for a news sitemap entry.
type Publication struct {
	Name     string `json:"name"     xml:"name"`
	Language string `json:"language" xml:"language"`
}

// ==================== XML Document Structs (for parsing) ====================

// xmlURLSet is the internal XML mapping for a <urlset> document.
type xmlURLSet struct {
	XMLName xml.Name `xml:"urlset"`
	URLs    []URL    `xml:"url"`
}

// xmlSitemapIndex is the internal XML mapping for a <sitemapindex> document.
type xmlSitemapIndex struct {
	XMLName  xml.Name       `xml:"sitemapindex"`
	Sitemaps []SitemapEntry `xml:"sitemap"`
}

// SitemapEntry represents a single <sitemap> element inside a sitemapindex.
type SitemapEntry struct {
	Loc     string `json:"loc"               xml:"loc"`
	LastMod string `json:"lastmod,omitempty"  xml:"lastmod,omitempty"`
}

// ==================== Parse Result ====================

// ParseResult is the unified return type for sitemap.Parse.
// Type is "urlset" or "sitemapindex". Only the corresponding field is populated.
type ParseResult struct {
	Type     string         `json:"type"`               // "urlset" or "sitemapindex"
	URLs     []URL          `json:"urls,omitempty"`     // populated when type="urlset"
	Sitemaps []SitemapEntry `json:"sitemaps,omitempty"` // populated when type="sitemapindex"
}

// ==================== Discover ====================

// DiscoverResult holds the result of sitemap.Discover.
type DiscoverResult struct {
	Sitemaps  []SitemapLink `json:"sitemaps"`
	TotalURLs int           `json:"total_urls"` // estimated total across all sitemaps
}

// SitemapLink describes a discovered sitemap file and its metadata.
type SitemapLink struct {
	URL          string `json:"url"`
	Source       string `json:"source"`        // "robots.txt", "well-known", or "index"
	URLCount     int    `json:"url_count"`     // estimated URL count (from Content-Length)
	ContentSize  int64  `json:"content_size"`  // Content-Length in bytes (0 if unknown)
	Encoding     string `json:"encoding"`      // "gzip", "br", or "" (from Content-Encoding)
	LastModified string `json:"last_modified"` // Last-Modified header
	ETag         string `json:"etag"`          // ETag header
}

// DiscoverOptions configures the Discover request behavior.
type DiscoverOptions struct {
	UserAgent string `json:"user_agent"` // custom User-Agent (default: "Yao-Robot/1.0")
	Timeout   int    `json:"timeout"`    // per-request timeout in seconds (default: 30)
}

// ==================== Fetch ====================

// FetchResult holds the result of sitemap.Fetch.
type FetchResult struct {
	URLs  []URL `json:"urls"`
	Total int   `json:"total"` // total URL count across all sitemaps (estimated for un-fetched files)
}

// FetchOptions configures the Fetch request behavior.
type FetchOptions struct {
	Offset    int    `json:"offset"`     // skip first N URLs (default: 0)
	Limit     int    `json:"limit"`      // max URLs to return (default/max: 50000)
	UserAgent string `json:"user_agent"` // custom User-Agent (default: "Yao-Robot/1.0")
	Timeout   int    `json:"timeout"`    // per-request timeout in seconds (default: 30)
}

// ==================== Build (Open/Write/Close) ====================

// sitemapWriter manages streaming sitemap file generation.
// Not exported — external callers interact via UUID handle only.
// Stored in openWriters (sync.Map), same pattern as the excel package.
type sitemapWriter struct {
	id          string
	dir         string       // output directory (absolute path)
	baseURL     string       // URL prefix for sitemap index references
	count       int          // URLs written to current file
	total       int          // total URLs written across all files
	fileIndex   int          // current file number (1-based)
	files       []string     // completed file paths
	currentFile *os.File     // current file handle
	encoder     *xml.Encoder // current xml encoder (token-level control)
	create      int64        // creation timestamp (unix seconds)
}

// openWriters stores active sitemapWriter handles.
// Key: UUID string, Value: *sitemapWriter.
var openWriters = sync.Map{}

// BuildResult holds the result returned by Build.Close.
type BuildResult struct {
	Index string   `json:"index"` // sitemap_index.xml path (empty string if single file)
	Files []string `json:"files"` // list of sitemap file paths
	Total int      `json:"total"` // total URLs written
}

// BuildOptions configures the Build.Open call.
type BuildOptions struct {
	Dir     string `json:"dir"`      // output directory (required)
	BaseURL string `json:"base_url"` // base URL for index references (required if multiple files)
}

// ==================== Constants ====================

const (
	// MaxURLsPerFile is the maximum number of URLs per sitemap file (per sitemaps.org spec).
	MaxURLsPerFile = 50000

	// DefaultUserAgent is the default User-Agent for HTTP requests.
	DefaultUserAgent = "Yao-Robot/1.0"

	// DefaultTimeout is the default per-request timeout in seconds.
	DefaultTimeout = 30

	// MaxDiscoverDepth is the maximum recursion depth for sitemapindex traversal.
	MaxDiscoverDepth = 3

	// Sitemap XML namespaces
	NSSitemap = "http://www.sitemaps.org/schemas/sitemap/0.9"
	NSImage   = "http://www.google.com/schemas/sitemap-image/1.1"
	NSVideo   = "http://www.google.com/schemas/sitemap-video/1.1"
	NSNews    = "http://www.google.com/schemas/sitemap-news/0.9"
)
