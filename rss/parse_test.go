package rss

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- Parse tests ---

func TestParse_RSS(t *testing.T) {
	feed, err := Parse(testRSS)
	require.NoError(t, err)
	assert.Equal(t, "rss2.0", feed.Format)
	assert.Equal(t, "Example Blog", feed.Title)
	assert.Len(t, feed.Items, 2)
}

func TestParse_Atom(t *testing.T) {
	feed, err := Parse(testAtom)
	require.NoError(t, err)
	assert.Equal(t, "atom1.0", feed.Format)
	assert.Equal(t, "Example Atom Feed", feed.Title)
	assert.Len(t, feed.Items, 2)
}

func TestParse_Podcast(t *testing.T) {
	feed, err := Parse(testPodcast)
	require.NoError(t, err)
	assert.Equal(t, "rss2.0", feed.Format)
	assert.NotNil(t, feed.Podcast)
	assert.Equal(t, "Jane Doe", feed.Podcast.Author)
}

func TestParse_Empty(t *testing.T) {
	_, err := Parse("")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "empty input")
}

func TestParse_Whitespace(t *testing.T) {
	_, err := Parse("   \n\t  ")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "empty input")
}

func TestParse_NotXML(t *testing.T) {
	_, err := Parse("This is not XML at all")
	assert.Error(t, err)
}

func TestParse_HTML(t *testing.T) {
	_, err := Parse(`<html><head><title>Not a feed</title></head></html>`)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unrecognized feed format")
	assert.Contains(t, err.Error(), "<html>")
}

func TestParse_UnknownRoot(t *testing.T) {
	_, err := Parse(`<?xml version="1.0"?><document><data>test</data></document>`)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unrecognized feed format")
}

// --- Validate tests ---

func TestValidate_ValidRSS(t *testing.T) {
	err := Validate(testRSS)
	assert.NoError(t, err)
}

func TestValidate_ValidAtom(t *testing.T) {
	err := Validate(testAtom)
	assert.NoError(t, err)
}

func TestValidate_ValidPodcast(t *testing.T) {
	err := Validate(testPodcast)
	assert.NoError(t, err)
}

func TestValidate_Empty(t *testing.T) {
	err := Validate("")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "empty input")
}

func TestValidate_NotXML(t *testing.T) {
	err := Validate("just some random text")
	assert.Error(t, err)
}

func TestValidate_BrokenXML(t *testing.T) {
	err := Validate(`<?xml version="1.0"?><rss><channel><title>oops`)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "RSS 2.0 parse error")
}

func TestValidate_HTML(t *testing.T) {
	err := Validate(`<html><body>Hello</body></html>`)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unrecognized feed format")
}

func TestValidate_MissingTitle_RSS(t *testing.T) {
	xml := `<?xml version="1.0"?>
<rss version="2.0">
  <channel>
    <link>https://example.com</link>
    <description>No title</description>
  </channel>
</rss>`
	err := Validate(xml)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "missing required <title>")
}

func TestValidate_MissingTitle_Atom(t *testing.T) {
	xml := `<?xml version="1.0"?>
<feed xmlns="http://www.w3.org/2005/Atom">
  <link href="https://example.com"/>
</feed>`
	err := Validate(xml)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "missing required <title>")
}

// --- detectFormat tests ---

func TestDetectFormat_RSS(t *testing.T) {
	f, err := detectFormat([]byte(`<?xml version="1.0"?><rss version="2.0"><channel></channel></rss>`))
	require.NoError(t, err)
	assert.Equal(t, "rss", f)
}

func TestDetectFormat_Atom(t *testing.T) {
	f, err := detectFormat([]byte(`<?xml version="1.0"?><feed xmlns="http://www.w3.org/2005/Atom"></feed>`))
	require.NoError(t, err)
	assert.Equal(t, "atom", f)
}

func TestDetectFormat_RDF(t *testing.T) {
	f, err := detectFormat([]byte(`<?xml version="1.0"?><rdf:RDF xmlns:rdf="http://www.w3.org/1999/02/22-rdf-syntax-ns#"></rdf:RDF>`))
	require.NoError(t, err)
	assert.Equal(t, "rss", f) // RDF treated as RSS
}

func TestDetectFormat_Unknown(t *testing.T) {
	_, err := detectFormat([]byte(`<html><head></head></html>`))
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unrecognized")
}

func TestDetectFormat_EmptyDoc(t *testing.T) {
	_, err := detectFormat([]byte(`<?xml version="1.0"?>`))
	assert.Error(t, err)
}
