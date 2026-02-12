package rss

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const testRSS = `<?xml version="1.0" encoding="UTF-8"?>
<rss version="2.0" xmlns:content="http://purl.org/rss/1.0/modules/content/" xmlns:dc="http://purl.org/dc/elements/1.1/">
  <channel>
    <title>Example Blog</title>
    <link>https://example.com</link>
    <description>An example blog feed</description>
    <language>en-us</language>
    <lastBuildDate>Mon, 01 Jan 2024 00:00:00 GMT</lastBuildDate>
    <item>
      <title>First Post</title>
      <link>https://example.com/first</link>
      <description>A short summary</description>
      <content:encoded><![CDATA[<p>Full content of the first post</p>]]></content:encoded>
      <dc:creator>Alice</dc:creator>
      <pubDate>Sun, 31 Dec 2023 12:00:00 GMT</pubDate>
      <guid isPermaLink="true">https://example.com/first</guid>
      <category>Tech</category>
      <category>Go</category>
      <enclosure url="https://example.com/audio.mp3" type="audio/mpeg" length="12345678"/>
    </item>
    <item>
      <title>Second Post</title>
      <link>https://example.com/second</link>
      <description>Another summary</description>
      <author>bob@example.com</author>
      <pubDate>Mon, 01 Jan 2024 00:00:00 GMT</pubDate>
      <guid>https://example.com/second</guid>
    </item>
  </channel>
</rss>`

func TestParseRSS_Basic(t *testing.T) {
	feed, err := parseRSS([]byte(testRSS))
	require.NoError(t, err)

	assert.Equal(t, "rss2.0", feed.Format)
	assert.Equal(t, "Example Blog", feed.Title)
	assert.Equal(t, "https://example.com", feed.Link)
	assert.Equal(t, "An example blog feed", feed.Description)
	assert.Equal(t, "en-us", feed.Language)
	assert.Equal(t, "Mon, 01 Jan 2024 00:00:00 GMT", feed.Updated)
	assert.Nil(t, feed.Podcast, "non-podcast feed should have nil Podcast")

	require.Len(t, feed.Items, 2)

	// First item
	item0 := feed.Items[0]
	assert.Equal(t, "First Post", item0.Title)
	assert.Equal(t, "https://example.com/first", item0.Link)
	assert.Equal(t, "A short summary", item0.Description)
	assert.Equal(t, "<p>Full content of the first post</p>", item0.Content)
	assert.Equal(t, "Alice", item0.Author) // dc:creator preferred
	assert.Equal(t, "Sun, 31 Dec 2023 12:00:00 GMT", item0.Published)
	assert.Equal(t, "https://example.com/first", item0.GUID)
	assert.Equal(t, []string{"Tech", "Go"}, item0.Categories)
	require.Len(t, item0.Enclosures, 1)
	assert.Equal(t, "https://example.com/audio.mp3", item0.Enclosures[0].URL)
	assert.Equal(t, "audio/mpeg", item0.Enclosures[0].Type)
	assert.Equal(t, "12345678", item0.Enclosures[0].Length)
	assert.Nil(t, item0.Episode)

	// Second item
	item1 := feed.Items[1]
	assert.Equal(t, "Second Post", item1.Title)
	assert.Equal(t, "bob@example.com", item1.Author) // fallback to <author>
	assert.Empty(t, item1.Categories)
	assert.Empty(t, item1.Enclosures)
}

const testPodcast = `<?xml version="1.0" encoding="UTF-8"?>
<rss version="2.0"
  xmlns:itunes="http://www.itunes.com/dtds/podcast-1.0.dtd"
  xmlns:content="http://purl.org/rss/1.0/modules/content/">
  <channel>
    <title>My Awesome Podcast</title>
    <link>https://podcast.example.com</link>
    <description>A podcast about technology</description>
    <language>en</language>
    <itunes:author>Jane Doe</itunes:author>
    <itunes:summary>Weekly tech discussions</itunes:summary>
    <itunes:image href="https://podcast.example.com/cover.jpg"/>
    <itunes:owner>
      <itunes:name>Jane Doe</itunes:name>
      <itunes:email>jane@example.com</itunes:email>
    </itunes:owner>
    <itunes:category text="Technology">
      <itunes:category text="Podcasting"/>
    </itunes:category>
    <itunes:category text="Education"/>
    <itunes:explicit>no</itunes:explicit>
    <itunes:type>episodic</itunes:type>
    <item>
      <title>Episode 1: Getting Started</title>
      <link>https://podcast.example.com/ep1</link>
      <description>Our first episode</description>
      <enclosure url="https://podcast.example.com/ep1.mp3" type="audio/mpeg" length="50000000"/>
      <pubDate>Wed, 15 Nov 2023 08:00:00 GMT</pubDate>
      <guid>https://podcast.example.com/ep1</guid>
      <itunes:duration>01:23:45</itunes:duration>
      <itunes:season>1</itunes:season>
      <itunes:episode>1</itunes:episode>
      <itunes:episodeType>full</itunes:episodeType>
      <itunes:explicit>no</itunes:explicit>
      <itunes:image href="https://podcast.example.com/ep1-cover.jpg"/>
      <itunes:summary>In this episode we discuss getting started with podcasting</itunes:summary>
    </item>
    <item>
      <title>Trailer</title>
      <link>https://podcast.example.com/trailer</link>
      <description>Preview of the show</description>
      <enclosure url="https://podcast.example.com/trailer.mp3" type="audio/mpeg" length="5000000"/>
      <itunes:duration>120</itunes:duration>
      <itunes:episodeType>trailer</itunes:episodeType>
      <itunes:explicit>yes</itunes:explicit>
    </item>
  </channel>
</rss>`

func TestParseRSS_Podcast(t *testing.T) {
	feed, err := parseRSS([]byte(testPodcast))
	require.NoError(t, err)

	assert.Equal(t, "rss2.0", feed.Format)
	assert.Equal(t, "My Awesome Podcast", feed.Title)

	// Podcast metadata
	require.NotNil(t, feed.Podcast)
	p := feed.Podcast
	assert.Equal(t, "Jane Doe", p.Author)
	assert.Equal(t, "Weekly tech discussions", p.Summary)
	assert.Equal(t, "https://podcast.example.com/cover.jpg", p.Image)
	assert.Equal(t, false, p.Explicit)
	assert.Equal(t, "episodic", p.Type)

	require.NotNil(t, p.Owner)
	assert.Equal(t, "Jane Doe", p.Owner.Name)
	assert.Equal(t, "jane@example.com", p.Owner.Email)

	// Categories: "Technology", "Technology > Podcasting", "Education"
	require.Len(t, p.Category, 3)
	assert.Equal(t, "Technology", p.Category[0])
	assert.Equal(t, "Technology > Podcasting", p.Category[1])
	assert.Equal(t, "Education", p.Category[2])

	// Episodes
	require.Len(t, feed.Items, 2)

	ep0 := feed.Items[0]
	require.NotNil(t, ep0.Episode)
	assert.Equal(t, "01:23:45", ep0.Episode.Duration)
	assert.Equal(t, 1, ep0.Episode.Season)
	assert.Equal(t, 1, ep0.Episode.Number)
	assert.Equal(t, "full", ep0.Episode.Type)
	assert.Equal(t, false, ep0.Episode.Explicit)
	assert.Equal(t, "https://podcast.example.com/ep1-cover.jpg", ep0.Episode.Image)
	assert.Equal(t, "In this episode we discuss getting started with podcasting", ep0.Episode.Summary)

	ep1 := feed.Items[1]
	require.NotNil(t, ep1.Episode)
	assert.Equal(t, "120", ep1.Episode.Duration)
	assert.Equal(t, "trailer", ep1.Episode.Type)
	assert.Equal(t, true, ep1.Episode.Explicit)
	assert.Equal(t, 0, ep1.Episode.Season) // not set
	assert.Equal(t, 0, ep1.Episode.Number) // not set
}

func TestParseRSS_MinimalFeed(t *testing.T) {
	xml := `<?xml version="1.0"?>
<rss version="2.0">
  <channel>
    <title>Minimal</title>
    <link>https://example.com</link>
    <description>Bare minimum</description>
  </channel>
</rss>`

	feed, err := parseRSS([]byte(xml))
	require.NoError(t, err)
	assert.Equal(t, "Minimal", feed.Title)
	assert.Empty(t, feed.Items)
	assert.Nil(t, feed.Podcast)
}

func TestParseRSS_InvalidXML(t *testing.T) {
	_, err := parseRSS([]byte(`<rss><channel><title>broken`))
	assert.Error(t, err)
}

func TestIsExplicit(t *testing.T) {
	assert.True(t, isExplicit("yes"))
	assert.True(t, isExplicit("Yes"))
	assert.True(t, isExplicit("true"))
	assert.True(t, isExplicit("explicit"))
	assert.False(t, isExplicit("no"))
	assert.False(t, isExplicit("false"))
	assert.False(t, isExplicit("clean"))
	assert.False(t, isExplicit(""))
}

func TestFirstNonEmpty(t *testing.T) {
	assert.Equal(t, "a", firstNonEmpty("a", "b"))
	assert.Equal(t, "b", firstNonEmpty("", "b"))
	assert.Equal(t, "c", firstNonEmpty("", "", "c"))
	assert.Equal(t, "", firstNonEmpty("", ""))
	assert.Equal(t, "x", firstNonEmpty("  ", " x "))
}

func TestFlattenCategories(t *testing.T) {
	cats := []itunesCategory{
		{Text: "Technology", Sub: []itunesCategory{{Text: "Podcasting"}}},
		{Text: "Education"},
	}
	result := flattenCategories(cats)
	assert.Equal(t, []string{"Technology", "Technology > Podcasting", "Education"}, result)
}

func TestFlattenCategories_Empty(t *testing.T) {
	result := flattenCategories(nil)
	assert.Nil(t, result)
}
