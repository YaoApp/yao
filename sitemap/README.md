# sitemap

Parse, validate, discover, fetch, and build XML sitemaps. Supports Google Image, Video, and News extensions.

## Processes

### sitemap.Parse

Parse a sitemap XML string. Auto-detects `<urlset>` or `<sitemapindex>`.

```javascript
var result = Process("sitemap.Parse", xmlString);
// result.type     → "urlset" or "sitemapindex"
// result.urls     → [{loc, lastmod, changefreq, priority, images, videos, news}]
// result.sitemaps → [{loc, lastmod}]  (when type = "sitemapindex")
```

### sitemap.Validate

Check if a string is valid sitemap XML. Returns `true` on success, or an error description string.

```javascript
var result = Process("sitemap.Validate", xmlString);
if (result !== true) {
  console.log("Invalid: " + result);
}
```

### sitemap.ParseRobo

Extract sitemap URLs from robots.txt content. Pure text parsing, no HTTP.

```javascript
var urls = Process("sitemap.ParseRobo", robotsTxtContent);
// urls → ["https://example.com/sitemap.xml", "https://example.com/sitemap2.xml"]
```

### sitemap.Discover

Discover sitemap files for a domain. Checks robots.txt, falls back to `/sitemap.xml`, recursively expands sitemapindex files.

```javascript
var result = Process("sitemap.Discover", "example.com");
// result.sitemaps  → [{url, source, url_count, content_size, encoding, last_modified, etag}]
// result.total_urls → 15000 (estimated)

// With options
var result = Process("sitemap.Discover", "example.com", {
  user_agent: "MyBot/1.0",
  timeout: 60,
});
```

### sitemap.Fetch

Fetch and parse URLs from a domain's sitemaps. Supports offset/limit pagination for large sites.

```javascript
// First page
var page1 = Process("sitemap.Fetch", "example.com", { limit: 100 });
// page1.urls  → [{loc, lastmod, images, ...}, ...]
// page1.total → 50000 (estimated)

// Next page
var page2 = Process("sitemap.Fetch", "example.com", {
  offset: 100,
  limit: 100,
});
```

**Options** (second argument, optional):

| Field      | Type   | Default          | Description                |
| ---------- | ------ | ---------------- | -------------------------- |
| offset     | int    | 0                | Skip first N URLs          |
| limit      | int    | 50000            | Max URLs to return         |
| user_agent | string | "Yao-Robot/1.0"  | Custom User-Agent          |
| timeout    | int    | 30               | Request timeout in seconds |

### sitemap.Build.Open

Open a new sitemap writer. Returns a UUID handle.

```javascript
var handle = Process("sitemap.Build.Open", {
  dir: "/data/sitemaps",
  base_url: "https://example.com",
});
```

### sitemap.Build.Write

Write a batch of URLs. Call multiple times. Auto-splits into new files at 50,000 URLs per file.

```javascript
Process("sitemap.Build.Write", handle, [
  { loc: "https://example.com/page1", lastmod: "2025-01-01", priority: "0.8" },
  { loc: "https://example.com/page2", changefreq: "daily" },
  {
    loc: "https://example.com/gallery",
    images: [{ loc: "https://example.com/img/1.jpg", caption: "Photo" }],
  },
]);
```

### sitemap.Build.Close

Finalize output. Generates a sitemap index if multiple files were created.

```javascript
var result = Process("sitemap.Build.Close", handle);
// result.files → ["/data/sitemaps/sitemap_1.xml"]
// result.index → ""  (empty if single file)
// result.total → 3

// With many URLs (auto-split):
// result.files → ["/data/sitemaps/sitemap_1.xml", "/data/sitemaps/sitemap_2.xml"]
// result.index → "/data/sitemaps/sitemap_index.xml"
// result.total → 75000
```
