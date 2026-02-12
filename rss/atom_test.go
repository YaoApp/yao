package rss

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const testAtom = `<?xml version="1.0" encoding="UTF-8"?>
<feed xmlns="http://www.w3.org/2005/Atom" xml:lang="en">
  <title>Example Atom Feed</title>
  <subtitle>An example Atom feed for testing</subtitle>
  <link href="https://example.com" rel="alternate"/>
  <link href="https://example.com/feed.atom" rel="self"/>
  <updated>2024-01-15T10:30:00Z</updated>
  <entry>
    <title>First Entry</title>
    <link href="https://example.com/entry-1" rel="alternate"/>
    <id>urn:uuid:entry-1</id>
    <published>2024-01-14T08:00:00Z</published>
    <updated>2024-01-14T10:00:00Z</updated>
    <summary>Summary of the first entry</summary>
    <content type="html">Full HTML content of entry 1</content>
    <author>
      <name>Alice</name>
      <email>alice@example.com</email>
    </author>
    <category term="tech" label="Technology"/>
    <category term="go"/>
  </entry>
  <entry>
    <title>Second Entry</title>
    <link href="https://example.com/entry-2"/>
    <id>urn:uuid:entry-2</id>
    <updated>2024-01-15T10:30:00Z</updated>
    <author>
      <name>Bob</name>
    </author>
    <author>
      <name>Charlie</name>
    </author>
  </entry>
</feed>`

func TestParseAtom_Basic(t *testing.T) {
	feed, err := parseAtom([]byte(testAtom))
	require.NoError(t, err)

	assert.Equal(t, "atom1.0", feed.Format)
	assert.Equal(t, "Example Atom Feed", feed.Title)
	assert.Equal(t, "https://example.com", feed.Link) // rel="alternate" preferred
	assert.Equal(t, "An example Atom feed for testing", feed.Description)
	assert.Equal(t, "en", feed.Language)
	assert.Equal(t, "2024-01-15T10:30:00Z", feed.Updated)
	assert.Nil(t, feed.Podcast)

	require.Len(t, feed.Items, 2)

	// First entry
	e0 := feed.Items[0]
	assert.Equal(t, "First Entry", e0.Title)
	assert.Equal(t, "https://example.com/entry-1", e0.Link)
	assert.Equal(t, "urn:uuid:entry-1", e0.GUID)
	assert.Equal(t, "2024-01-14T08:00:00Z", e0.Published)
	assert.Equal(t, "2024-01-14T10:00:00Z", e0.Updated)
	assert.Equal(t, "Summary of the first entry", e0.Description)
	assert.Equal(t, "Full HTML content of entry 1", e0.Content)
	assert.Equal(t, "Alice", e0.Author)
	require.Len(t, e0.Categories, 2)
	assert.Equal(t, "Technology", e0.Categories[0]) // label preferred
	assert.Equal(t, "go", e0.Categories[1])         // fallback to term

	// Second entry — multiple authors
	e1 := feed.Items[1]
	assert.Equal(t, "Second Entry", e1.Title)
	assert.Equal(t, "Bob, Charlie", e1.Author) // joined
	assert.Empty(t, e1.Published)
}

func TestParseAtom_MinimalFeed(t *testing.T) {
	xml := `<?xml version="1.0"?>
<feed xmlns="http://www.w3.org/2005/Atom">
  <title>Minimal Atom</title>
</feed>`

	feed, err := parseAtom([]byte(xml))
	require.NoError(t, err)
	assert.Equal(t, "atom1.0", feed.Format)
	assert.Equal(t, "Minimal Atom", feed.Title)
	assert.Empty(t, feed.Link)
	assert.Empty(t, feed.Items)
}

func TestParseAtom_InvalidXML(t *testing.T) {
	_, err := parseAtom([]byte(`<feed><title>broken`))
	assert.Error(t, err)
}

func TestExtractAtomLink(t *testing.T) {
	links := []atomLink{
		{Href: "https://example.com/self", Rel: "self"},
		{Href: "https://example.com", Rel: "alternate"},
		{Href: "https://example.com/other", Rel: "related"},
	}
	assert.Equal(t, "https://example.com", extractAtomLink(links))
}

func TestExtractAtomLink_NoAlternate(t *testing.T) {
	links := []atomLink{
		{Href: "https://example.com/self", Rel: "self"},
	}
	assert.Equal(t, "https://example.com/self", extractAtomLink(links))
}

func TestExtractAtomLink_EmptyRel(t *testing.T) {
	// Empty rel should be treated as "alternate"
	links := []atomLink{
		{Href: "https://example.com", Rel: ""},
	}
	assert.Equal(t, "https://example.com", extractAtomLink(links))
}

func TestExtractAtomLink_Empty(t *testing.T) {
	assert.Equal(t, "", extractAtomLink(nil))
	assert.Equal(t, "", extractAtomLink([]atomLink{}))
}
