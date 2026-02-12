package rss

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDiscover_HTMLLinkTags(t *testing.T) {
	html := `<html>
<head>
  <title>My Site</title>
  <link rel="alternate" type="application/rss+xml" href="https://example.com/feed.xml" title="RSS Feed"/>
  <link rel="alternate" type="application/atom+xml" href="https://example.com/atom.xml" title="Atom Feed"/>
  <link rel="stylesheet" href="/style.css"/>
</head>
<body>Hello</body>
</html>`

	links := Discover(html)
	require.Len(t, links, 2)

	assert.Equal(t, "https://example.com/feed.xml", links[0].URL)
	assert.Equal(t, "RSS Feed", links[0].Title)
	assert.Equal(t, "rss", links[0].Type)

	assert.Equal(t, "https://example.com/atom.xml", links[1].URL)
	assert.Equal(t, "Atom Feed", links[1].Title)
	assert.Equal(t, "atom", links[1].Type)
}

func TestDiscover_HTMLLinkTags_AttributeOrder(t *testing.T) {
	// Attributes in different order, single quotes
	html := `<link href='https://blog.example.com/rss' title='Blog' type='application/rss+xml' rel='alternate'>`
	links := Discover(html)
	require.Len(t, links, 1)
	assert.Equal(t, "https://blog.example.com/rss", links[0].URL)
	assert.Equal(t, "Blog", links[0].Title)
	assert.Equal(t, "rss", links[0].Type)
}

func TestDiscover_MarkdownLinks(t *testing.T) {
	md := `# My Bookmarks

Here are some feeds:
- [Tech News](https://news.example.com/rss.xml)
- [Go Blog](https://go.dev/blog/feed.atom)
- [Not a feed](https://example.com/about)
`
	links := Discover(md)
	require.Len(t, links, 2)

	assert.Equal(t, "https://news.example.com/rss.xml", links[0].URL)
	assert.Equal(t, "Tech News", links[0].Title)
	assert.Equal(t, "rss", links[0].Type)

	assert.Equal(t, "https://go.dev/blog/feed.atom", links[1].URL)
	assert.Equal(t, "Go Blog", links[1].Title)
	assert.Equal(t, "atom", links[1].Type)
}

func TestDiscover_BareURLs(t *testing.T) {
	text := `Check out these feeds:
https://example.com/feed.xml
https://blog.example.com/rss
https://news.example.com/atom.xml
https://example.com/about (not a feed)
`
	links := Discover(text)
	require.Len(t, links, 3)

	assert.Equal(t, "https://example.com/feed.xml", links[0].URL)
	assert.Equal(t, "https://blog.example.com/rss", links[1].URL)
	assert.Equal(t, "https://news.example.com/atom.xml", links[2].URL)
}

func TestDiscover_BareURLs_QueryParams(t *testing.T) {
	text := `Feed URL: https://example.com/api?feed=rss&lang=en`
	links := Discover(text)
	require.Len(t, links, 1)
	assert.Equal(t, "https://example.com/api?feed=rss&lang=en", links[0].URL)
}

func TestDiscover_Deduplication(t *testing.T) {
	// Same URL appears in HTML link tag and as bare URL
	text := `<link rel="alternate" type="application/rss+xml" href="https://example.com/feed.xml" title="Feed"/>
Check out: https://example.com/feed.xml`

	links := Discover(text)
	require.Len(t, links, 1) // deduplicated
	assert.Equal(t, "https://example.com/feed.xml", links[0].URL)
	assert.Equal(t, "Feed", links[0].Title) // from HTML tag (higher priority)
}

func TestDiscover_PartialHTML(t *testing.T) {
	// Incomplete HTML fragment
	fragment := `<div>Some content</div>
<link rel="alternate" type="application/rss+xml" href="https://example.com/feed" title="My Feed">
<p>More broken`

	links := Discover(fragment)
	require.Len(t, links, 1)
	assert.Equal(t, "https://example.com/feed", links[0].URL)
}

func TestDiscover_Empty(t *testing.T) {
	assert.Nil(t, Discover(""))
	assert.Nil(t, Discover("   "))
}

func TestDiscover_NoFeeds(t *testing.T) {
	links := Discover("Hello world! Visit https://example.com for more info.")
	assert.Empty(t, links)
}

func TestDiscover_MixedContent(t *testing.T) {
	// HTML link + Markdown link + bare URL, all different
	mixed := `<link rel="alternate" type="application/atom+xml" href="https://a.com/atom.xml" title="A"/>
Some text with [B Feed](https://b.com/feed.xml) and also
https://c.com/rss.xml is available.`

	links := Discover(mixed)
	require.Len(t, links, 3)
	assert.Equal(t, "https://a.com/atom.xml", links[0].URL)
	assert.Equal(t, "atom", links[0].Type)
	assert.Equal(t, "https://b.com/feed.xml", links[1].URL)
	assert.Equal(t, "https://c.com/rss.xml", links[2].URL)
	assert.Equal(t, "rss", links[2].Type)
}

func TestDiscover_TrailingPunctuation(t *testing.T) {
	text := `Check https://example.com/feed.xml.`
	links := Discover(text)
	require.Len(t, links, 1)
	assert.Equal(t, "https://example.com/feed.xml", links[0].URL)
}

func TestLooksLikeFeedURL(t *testing.T) {
	assert.True(t, looksLikeFeedURL("https://example.com/feed.xml"))
	assert.True(t, looksLikeFeedURL("https://example.com/rss"))
	assert.True(t, looksLikeFeedURL("https://example.com/atom.xml"))
	assert.True(t, looksLikeFeedURL("https://example.com/index.xml"))
	assert.True(t, looksLikeFeedURL("https://example.com/feed/"))
	assert.True(t, looksLikeFeedURL("https://example.com/api?feed=rss"))
	assert.True(t, looksLikeFeedURL("https://example.com/blog.rss"))
	assert.False(t, looksLikeFeedURL("https://example.com/about"))
	assert.False(t, looksLikeFeedURL("https://example.com/image.png"))
}

func TestGuessTypeFromURL(t *testing.T) {
	assert.Equal(t, "rss", guessTypeFromURL("https://example.com/rss.xml"))
	assert.Equal(t, "atom", guessTypeFromURL("https://example.com/atom.xml"))
	assert.Equal(t, "", guessTypeFromURL("https://example.com/feed.xml"))
	assert.Equal(t, "", guessTypeFromURL("https://example.com/index.xml"))
}
