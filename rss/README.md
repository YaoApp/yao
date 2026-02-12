# rss

Parse, validate, discover, fetch, and build RSS 2.0 / Atom 1.0 feeds. Includes iTunes/Podcast extension support.

## Processes

### rss.Parse

Parse an RSS/Atom XML string into a Feed object. Auto-detects format.

```javascript
var feed = Process("rss.Parse", xmlString);
// feed.format  → "rss2.0" or "atom1.0"
// feed.title   → "My Blog"
// feed.items   → [{title, link, description, content, author, published, ...}]
// feed.podcast → {author, summary, image, ...} (nil for non-podcast feeds)
```

### rss.Validate

Check if a string is valid RSS/Atom XML. Returns `true` on success, or an error description string.

```javascript
var result = Process("rss.Validate", xmlString);
if (result !== true) {
  console.log("Invalid: " + result);
}
```

### rss.Fetch

Fetch a remote feed by URL. Supports gzip and conditional requests (ETag / Last-Modified).

```javascript
// First fetch
var result = Process("rss.Fetch", "https://example.com/feed.xml");
// result.feed          → parsed Feed object
// result.status_code   → 200
// result.etag          → "abc123"
// result.last_modified → "Wed, 01 Jan 2025 00:00:00 GMT"

// Conditional polling (saves bandwidth)
var result2 = Process("rss.Fetch", "https://example.com/feed.xml", {
  etag: result.etag,
  last_modified: result.last_modified,
});
// result2.not_modified → true (when 304)
// result2.feed         → nil  (when 304)
```

**Options** (second argument, optional):

| Field          | Type   | Default          | Description                     |
| -------------- | ------ | ---------------- | ------------------------------- |
| user_agent     | string | "Yao-Robot/1.0"  | Custom User-Agent               |
| timeout        | int    | 30               | Request timeout in seconds      |
| etag           | string |                  | ETag for If-None-Match          |
| last_modified  | string |                  | Value for If-Modified-Since     |

### rss.Discover

Extract feed URLs from HTML, Markdown, or plain text. No HTTP requests.

```javascript
var links = Process("rss.Discover", htmlString);
// links → [{url: "https://example.com/feed.xml", title: "Blog", type: "rss"}]
```

### rss.Build

Generate RSS or Atom XML from a Feed object.

```javascript
// Build RSS 2.0 (default)
var xml = Process("rss.Build", feedObj);

// Build Atom 1.0
var xml = Process("rss.Build", feedObj, "atom");
```
