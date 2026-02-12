package rss

// Feed represents a unified feed structure for RSS 2.0, Atom 1.0, and Podcast feeds.
// The Format field indicates the source format detected during parsing.
type Feed struct {
	Format      string     `json:"format"`             // "rss2.0" or "atom1.0"
	Title       string     `json:"title"`              // Feed title
	Link        string     `json:"link"`               // Primary feed link (website URL)
	Description string     `json:"description"`        // Feed description or subtitle
	Language    string     `json:"language,omitempty"` // Language code (e.g. "en", "zh-CN")
	Updated     string     `json:"updated,omitempty"`  // Last build date / updated timestamp
	Items       []FeedItem `json:"items"`              // Feed entries
	Podcast     *Podcast   `json:"podcast,omitempty"`  // iTunes/Podcast metadata (nil for non-podcast feeds)
}

// Podcast holds iTunes namespace channel-level metadata.
// Populated only when the feed contains itunes:* extensions.
type Podcast struct {
	Author   string   `json:"author,omitempty"`   // itunes:author
	Summary  string   `json:"summary,omitempty"`  // itunes:summary
	Image    string   `json:"image,omitempty"`    // itunes:image href (cover art)
	Owner    *Owner   `json:"owner,omitempty"`    // itunes:owner
	Category []string `json:"category,omitempty"` // itunes:category text values (may be nested)
	Explicit bool     `json:"explicit"`           // itunes:explicit
	Type     string   `json:"type,omitempty"`     // itunes:type ("episodic" or "serial")
}

// Owner represents the podcast owner information from the iTunes namespace.
type Owner struct {
	Name  string `json:"name,omitempty"`  // itunes:name
	Email string `json:"email,omitempty"` // itunes:email
}

// FeedItem represents a single entry in a feed.
type FeedItem struct {
	Title       string      `json:"title"`                 // Item title
	Link        string      `json:"link"`                  // Item permalink
	Description string      `json:"description,omitempty"` // Short description or summary
	Content     string      `json:"content,omitempty"`     // Full content (content:encoded for RSS, content for Atom)
	Author      string      `json:"author,omitempty"`      // Author name
	Published   string      `json:"published,omitempty"`   // Publication date
	Updated     string      `json:"updated,omitempty"`     // Last updated date
	GUID        string      `json:"guid,omitempty"`        // Globally unique identifier
	Categories  []string    `json:"categories,omitempty"`  // Category tags
	Enclosures  []Enclosure `json:"enclosures,omitempty"`  // Attached media files
	Episode     *Episode    `json:"episode,omitempty"`     // iTunes/Podcast episode metadata (nil for non-podcast items)
}

// Episode holds iTunes namespace item-level metadata for podcast episodes.
// Populated only when the item contains itunes:* extensions.
type Episode struct {
	Duration string `json:"duration,omitempty"` // itunes:duration (HH:MM:SS or seconds)
	Season   int    `json:"season,omitempty"`   // itunes:season
	Number   int    `json:"number,omitempty"`   // itunes:episode
	Type     string `json:"type,omitempty"`     // itunes:episodeType ("full", "trailer", or "bonus")
	Explicit bool   `json:"explicit"`           // itunes:explicit
	Image    string `json:"image,omitempty"`    // itunes:image href (episode-specific cover art)
	Summary  string `json:"summary,omitempty"`  // itunes:summary
}

// Enclosure represents an attached media file in a feed item.
type Enclosure struct {
	URL    string `json:"url"`              // Media file URL
	Type   string `json:"type,omitempty"`   // MIME type (e.g. "audio/mpeg")
	Length string `json:"length,omitempty"` // File size in bytes
}

// FeedLink represents a discovered feed URL extracted from HTML, Markdown, or plain text.
type FeedLink struct {
	URL   string `json:"url"`             // Feed URL
	Title string `json:"title,omitempty"` // Feed title (if available from context)
	Type  string `json:"type,omitempty"`  // "rss" or "atom" (if determinable)
}

// FetchResult holds the result of an rss.Fetch call.
// When the server responds with 304, Feed is nil and NotModified is true.
type FetchResult struct {
	Feed         *Feed  `json:"feed"`                    // Parsed feed (nil on 304)
	StatusCode   int    `json:"status_code"`             // HTTP status code (200, 304, etc.)
	ETag         string `json:"etag,omitempty"`          // ETag response header (for conditional requests)
	LastModified string `json:"last_modified,omitempty"` // Last-Modified response header
	NotModified  bool   `json:"not_modified"`            // True when server returned 304
}

// FetchOptions configures the rss.Fetch request behavior.
type FetchOptions struct {
	UserAgent    string `json:"user_agent"`    // Custom User-Agent (default: "Yao-Robot/1.0")
	Timeout      int    `json:"timeout"`       // Per-request timeout in seconds (default: 30)
	ETag         string `json:"etag"`          // ETag from a previous fetch (for If-None-Match)
	LastModified string `json:"last_modified"` // Last-Modified from a previous fetch (for If-Modified-Since)
}
