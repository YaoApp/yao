package rss

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Helper feed for build tests
func newTestFeed() *Feed {
	return &Feed{
		Format:      "rss2.0",
		Title:       "Test Blog",
		Link:        "https://example.com",
		Description: "A test blog",
		Language:    "en",
		Updated:     "Mon, 01 Jan 2024 00:00:00 +0000",
		Items: []FeedItem{
			{
				Title:       "First Post",
				Link:        "https://example.com/post-1",
				Description: "Summary of first post",
				Content:     "<p>Full content</p>",
				Author:      "Alice",
				Published:   "Sun, 31 Dec 2023 12:00:00 +0000",
				GUID:        "https://example.com/post-1",
				Categories:  []string{"Tech", "Go"},
				Enclosures: []Enclosure{
					{URL: "https://example.com/audio.mp3", Type: "audio/mpeg", Length: "12345678"},
				},
			},
			{
				Title:     "Second Post",
				Link:      "https://example.com/post-2",
				Published: "Mon, 01 Jan 2024 00:00:00 +0000",
				GUID:      "https://example.com/post-2",
			},
		},
	}
}

func newTestPodcastFeed() *Feed {
	feed := &Feed{
		Format:      "rss2.0",
		Title:       "My Podcast",
		Link:        "https://podcast.example.com",
		Description: "A tech podcast",
		Language:    "en",
		Updated:     "Wed, 15 Nov 2023 08:00:00 +0000",
		Podcast: &Podcast{
			Author:   "Jane Doe",
			Summary:  "Weekly tech discussions",
			Image:    "https://podcast.example.com/cover.jpg",
			Explicit: false,
			Type:     "episodic",
			Owner:    &Owner{Name: "Jane Doe", Email: "jane@example.com"},
			Category: []string{"Technology", "Technology > Podcasting", "Education"},
		},
		Items: []FeedItem{
			{
				Title:       "Episode 1",
				Link:        "https://podcast.example.com/ep1",
				Description: "Our first episode",
				Published:   "Wed, 15 Nov 2023 08:00:00 +0000",
				GUID:        "https://podcast.example.com/ep1",
				Enclosures: []Enclosure{
					{URL: "https://podcast.example.com/ep1.mp3", Type: "audio/mpeg", Length: "50000000"},
				},
				Episode: &Episode{
					Duration: "01:23:45",
					Season:   1,
					Number:   1,
					Type:     "full",
					Explicit: false,
					Image:    "https://podcast.example.com/ep1-cover.jpg",
					Summary:  "Getting started with podcasting",
				},
			},
		},
	}
	return feed
}

// --- RSS Build tests ---

func TestBuild_RSS_Basic(t *testing.T) {
	feed := newTestFeed()
	xml, err := Build(feed, "rss")
	require.NoError(t, err)

	assert.Contains(t, xml, `<?xml version="1.0" encoding="UTF-8"?>`)
	assert.Contains(t, xml, `<rss version="2.0"`)
	assert.Contains(t, xml, `<title>Test Blog</title>`)
	assert.Contains(t, xml, `<link>https://example.com</link>`)
	assert.Contains(t, xml, `<description>A test blog</description>`)
	assert.Contains(t, xml, `<language>en</language>`)
	assert.Contains(t, xml, `<title>First Post</title>`)
	assert.Contains(t, xml, `<title>Second Post</title>`)
	assert.Contains(t, xml, `<author>Alice</author>`)
	assert.Contains(t, xml, `<category>Tech</category>`)
	assert.Contains(t, xml, `<category>Go</category>`)
	assert.Contains(t, xml, `url="https://example.com/audio.mp3"`)
	assert.Contains(t, xml, `type="audio/mpeg"`)

	// Should NOT contain itunes namespace since no podcast data
	assert.NotContains(t, xml, "itunes")
}

func TestBuild_RSS_DefaultFormat(t *testing.T) {
	feed := newTestFeed()
	xml, err := Build(feed, "")
	require.NoError(t, err)
	assert.Contains(t, xml, `<rss version="2.0"`)
}

func TestBuild_RSS_Podcast(t *testing.T) {
	feed := newTestPodcastFeed()
	xml, err := Build(feed, "rss")
	require.NoError(t, err)

	assert.Contains(t, xml, `xmlns:itunes=`)
	assert.Contains(t, xml, `<itunes:author>Jane Doe</itunes:author>`)
	assert.Contains(t, xml, `<itunes:summary>Weekly tech discussions</itunes:summary>`)
	assert.Contains(t, xml, `href="https://podcast.example.com/cover.jpg"`)
	assert.Contains(t, xml, `<itunes:name>Jane Doe</itunes:name>`)
	assert.Contains(t, xml, `<itunes:email>jane@example.com</itunes:email>`)
	assert.Contains(t, xml, `<itunes:explicit>no</itunes:explicit>`)
	assert.Contains(t, xml, `<itunes:type>episodic</itunes:type>`)

	// Categories
	assert.Contains(t, xml, `text="Technology"`)
	assert.Contains(t, xml, `text="Podcasting"`)
	assert.Contains(t, xml, `text="Education"`)

	// Episode metadata
	assert.Contains(t, xml, `<itunes:duration>01:23:45</itunes:duration>`)
	assert.Contains(t, xml, `<itunes:season>1</itunes:season>`)
	assert.Contains(t, xml, `<itunes:episode>1</itunes:episode>`)
	assert.Contains(t, xml, `<itunes:episodeType>full</itunes:episodeType>`)
}

func TestBuild_RSS_ContentEncoded(t *testing.T) {
	feed := newTestFeed()
	xml, err := Build(feed, "rss")
	require.NoError(t, err)

	assert.Contains(t, xml, `xmlns:content=`)
	assert.Contains(t, xml, `<content:encoded>`)
	assert.Contains(t, xml, `<p>Full content</p>`)
}

// --- Atom Build tests ---

func TestBuild_Atom_Basic(t *testing.T) {
	feed := newTestFeed()
	xml, err := Build(feed, "atom")
	require.NoError(t, err)

	assert.Contains(t, xml, `<?xml version="1.0" encoding="UTF-8"?>`)
	assert.Contains(t, xml, `<feed xmlns="http://www.w3.org/2005/Atom"`)
	assert.Contains(t, xml, `<title>Test Blog</title>`)
	assert.Contains(t, xml, `<subtitle>A test blog</subtitle>`)
	assert.Contains(t, xml, `href="https://example.com"`)
	assert.Contains(t, xml, `<title>First Post</title>`)
	assert.Contains(t, xml, `<name>Alice</name>`)
	assert.Contains(t, xml, `term="Tech"`)
	assert.Contains(t, xml, `term="Go"`)

	// Atom should NOT have iTunes namespace
	assert.NotContains(t, xml, "itunes")
}

func TestBuild_Atom_PodcastIgnored(t *testing.T) {
	feed := newTestPodcastFeed()
	xml, err := Build(feed, "atom")
	require.NoError(t, err)

	// Podcast metadata should be ignored in Atom output
	assert.NotContains(t, xml, "itunes")
	assert.Contains(t, xml, `<title>My Podcast</title>`)
}

func TestBuild_Atom_MultipleAuthors(t *testing.T) {
	feed := &Feed{
		Title: "Multi Author",
		Items: []FeedItem{
			{
				Title:  "Post",
				Author: "Alice, Bob",
				GUID:   "1",
			},
		},
	}
	xml, err := Build(feed, "atom")
	require.NoError(t, err)
	assert.Contains(t, xml, `<name>Alice</name>`)
	assert.Contains(t, xml, `<name>Bob</name>`)
}

// --- Round-trip test ---

func TestBuild_RoundTrip_RSS(t *testing.T) {
	// Parse → Build → Parse and verify consistency
	feed1, err := Parse(testRSS)
	require.NoError(t, err)

	xmlOut, err := Build(feed1, "rss")
	require.NoError(t, err)

	feed2, err := Parse(xmlOut)
	require.NoError(t, err)

	assert.Equal(t, feed1.Title, feed2.Title)
	assert.Equal(t, feed1.Link, feed2.Link)
	assert.Equal(t, feed1.Description, feed2.Description)
	assert.Len(t, feed2.Items, len(feed1.Items))
	assert.Equal(t, feed1.Items[0].Title, feed2.Items[0].Title)
}

func TestBuild_RoundTrip_Podcast(t *testing.T) {
	feed1, err := Parse(testPodcast)
	require.NoError(t, err)

	xmlOut, err := Build(feed1, "rss")
	require.NoError(t, err)

	feed2, err := Parse(xmlOut)
	require.NoError(t, err)

	assert.Equal(t, feed1.Title, feed2.Title)
	require.NotNil(t, feed2.Podcast)
	assert.Equal(t, feed1.Podcast.Author, feed2.Podcast.Author)
	assert.Equal(t, feed1.Podcast.Summary, feed2.Podcast.Summary)
	assert.Len(t, feed2.Items, len(feed1.Items))

	require.NotNil(t, feed2.Items[0].Episode)
	assert.Equal(t, feed1.Items[0].Episode.Duration, feed2.Items[0].Episode.Duration)
}

func TestBuild_RoundTrip_Atom(t *testing.T) {
	feed1, err := Parse(testAtom)
	require.NoError(t, err)

	xmlOut, err := Build(feed1, "atom")
	require.NoError(t, err)

	feed2, err := Parse(xmlOut)
	require.NoError(t, err)

	assert.Equal(t, feed1.Title, feed2.Title)
	assert.Len(t, feed2.Items, len(feed1.Items))
	assert.Equal(t, feed1.Items[0].Title, feed2.Items[0].Title)
}

// --- Edge cases ---

func TestBuild_NilFeed(t *testing.T) {
	_, err := Build(nil, "rss")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "nil")
}

func TestBuild_UnsupportedFormat(t *testing.T) {
	feed := newTestFeed()
	_, err := Build(feed, "json")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported")
}

func TestBuild_EmptyFeed(t *testing.T) {
	feed := &Feed{Title: "Empty"}
	xml, err := Build(feed, "rss")
	require.NoError(t, err)
	assert.Contains(t, xml, `<title>Empty</title>`)
	assert.NotContains(t, xml, "<item>")
}

// --- mapToFeed tests ---

func TestMapToFeed_DirectFeed(t *testing.T) {
	original := newTestFeed()
	result, err := mapToFeed(original)
	require.NoError(t, err)
	assert.Equal(t, original, result)
}

func TestMapToFeed_FromMap(t *testing.T) {
	m := map[string]interface{}{
		"title":       "From Map",
		"link":        "https://example.com",
		"description": "Converted from map",
		"items": []interface{}{
			map[string]interface{}{
				"title": "Item 1",
				"link":  "https://example.com/1",
			},
		},
	}
	feed, err := mapToFeed(m)
	require.NoError(t, err)
	assert.Equal(t, "From Map", feed.Title)
	require.Len(t, feed.Items, 1)
	assert.Equal(t, "Item 1", feed.Items[0].Title)
}

func TestMapToFeed_Nil(t *testing.T) {
	_, err := mapToFeed(nil)
	assert.Error(t, err)
}

// --- intToStr test ---

func TestIntToStr(t *testing.T) {
	assert.Equal(t, "1", intToStr(1))
	assert.Equal(t, "42", intToStr(42))
	assert.Equal(t, "123", intToStr(123))
	assert.Equal(t, "", intToStr(0))
}

// --- Build output well-formedness ---

func TestBuild_RSS_WellFormedXML(t *testing.T) {
	feed := newTestPodcastFeed()
	xmlStr, err := Build(feed, "rss")
	require.NoError(t, err)

	// Should start with XML declaration
	assert.True(t, strings.HasPrefix(xmlStr, "<?xml"))

	// Should be valid — re-parseable
	err = Validate(xmlStr)
	assert.NoError(t, err)
}

func TestBuild_Atom_WellFormedXML(t *testing.T) {
	feed := newTestFeed()
	xmlStr, err := Build(feed, "atom")
	require.NoError(t, err)

	assert.True(t, strings.HasPrefix(xmlStr, "<?xml"))

	// Should be valid Atom — re-parseable
	err = Validate(xmlStr)
	assert.NoError(t, err)
}
