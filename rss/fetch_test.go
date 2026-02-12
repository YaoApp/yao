package rss

import (
	"compress/gzip"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// testFeedXML is a minimal RSS feed for fetch testing.
const testFeedXML = `<?xml version="1.0" encoding="UTF-8"?>
<rss version="2.0">
  <channel>
    <title>Test Feed</title>
    <link>https://example.com</link>
    <description>A test feed</description>
    <item>
      <title>Post 1</title>
      <link>https://example.com/post1</link>
    </item>
  </channel>
</rss>`

func TestFetchBasic(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/rss+xml")
		w.Header().Set("ETag", `"abc123"`)
		w.Header().Set("Last-Modified", "Wed, 01 Jan 2025 00:00:00 GMT")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(testFeedXML))
	}))
	defer server.Close()

	result, err := Fetch(server.URL, nil)
	require.NoError(t, err)

	assert.Equal(t, 200, result.StatusCode)
	assert.False(t, result.NotModified)
	assert.Equal(t, `"abc123"`, result.ETag)
	assert.Equal(t, "Wed, 01 Jan 2025 00:00:00 GMT", result.LastModified)

	require.NotNil(t, result.Feed)
	assert.Equal(t, "Test Feed", result.Feed.Title)
	assert.Equal(t, "rss2.0", result.Feed.Format)
	assert.Len(t, result.Feed.Items, 1)
	assert.Equal(t, "Post 1", result.Feed.Items[0].Title)
}

func TestFetchConditional304(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("If-None-Match") == `"abc123"` {
			w.WriteHeader(http.StatusNotModified)
			return
		}
		w.Header().Set("ETag", `"abc123"`)
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(testFeedXML))
	}))
	defer server.Close()

	result, err := Fetch(server.URL, &FetchOptions{ETag: `"abc123"`})
	require.NoError(t, err)

	assert.Equal(t, 304, result.StatusCode)
	assert.True(t, result.NotModified)
	assert.Nil(t, result.Feed)
}

func TestFetchConditionalLastModified(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("If-Modified-Since") == "Wed, 01 Jan 2025 00:00:00 GMT" {
			w.WriteHeader(http.StatusNotModified)
			return
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(testFeedXML))
	}))
	defer server.Close()

	result, err := Fetch(server.URL, &FetchOptions{
		LastModified: "Wed, 01 Jan 2025 00:00:00 GMT",
	})
	require.NoError(t, err)

	assert.Equal(t, 304, result.StatusCode)
	assert.True(t, result.NotModified)
	assert.Nil(t, result.Feed)
}

func TestFetchGzip(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Only serve gzip if client accepts it
		if r.Header.Get("Accept-Encoding") != "gzip" {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(testFeedXML))
			return
		}
		w.Header().Set("Content-Encoding", "gzip")
		w.Header().Set("Content-Type", "application/rss+xml")
		w.WriteHeader(http.StatusOK)

		gz := gzip.NewWriter(w)
		gz.Write([]byte(testFeedXML))
		gz.Close()
	}))
	defer server.Close()

	result, err := Fetch(server.URL, nil)
	require.NoError(t, err)

	assert.Equal(t, 200, result.StatusCode)
	require.NotNil(t, result.Feed)
	assert.Equal(t, "Test Feed", result.Feed.Title)
}

func TestFetchCustomUserAgent(t *testing.T) {
	var receivedUA string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedUA = r.Header.Get("User-Agent")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(testFeedXML))
	}))
	defer server.Close()

	_, err := Fetch(server.URL, &FetchOptions{UserAgent: "MyBot/2.0"})
	require.NoError(t, err)

	assert.Equal(t, "MyBot/2.0", receivedUA)
}

func TestFetchDefaultUserAgent(t *testing.T) {
	var receivedUA string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedUA = r.Header.Get("User-Agent")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(testFeedXML))
	}))
	defer server.Close()

	_, err := Fetch(server.URL, nil)
	require.NoError(t, err)

	assert.Equal(t, "Yao-Robot/1.0", receivedUA)
}

func TestFetchAcceptHeader(t *testing.T) {
	var receivedAccept string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedAccept = r.Header.Get("Accept")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(testFeedXML))
	}))
	defer server.Close()

	_, err := Fetch(server.URL, nil)
	require.NoError(t, err)

	assert.Equal(t, "application/rss+xml, application/atom+xml, application/xml, text/xml", receivedAccept)
}

func TestFetch404(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	_, err := Fetch(server.URL, nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "HTTP 404")
}

func TestFetch500(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	_, err := Fetch(server.URL, nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "HTTP 500")
}

func TestFetchEmptyURL(t *testing.T) {
	_, err := Fetch("", nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "url is required")
}

func TestFetchInvalidURL(t *testing.T) {
	_, err := Fetch("http://localhost:99999/nonexistent", &FetchOptions{Timeout: 1})
	assert.Error(t, err)
}

func TestFetchAtomFeed(t *testing.T) {
	atomXML := `<?xml version="1.0" encoding="UTF-8"?>
<feed xmlns="http://www.w3.org/2005/Atom">
  <title>Atom Test</title>
  <link href="https://example.com"/>
  <updated>2025-01-01T00:00:00Z</updated>
  <entry>
    <title>Atom Entry</title>
    <link href="https://example.com/entry1"/>
    <id>urn:uuid:1</id>
    <updated>2025-01-01T00:00:00Z</updated>
  </entry>
</feed>`

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/atom+xml")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(atomXML))
	}))
	defer server.Close()

	result, err := Fetch(server.URL, nil)
	require.NoError(t, err)

	require.NotNil(t, result.Feed)
	assert.Equal(t, "atom1.0", result.Feed.Format)
	assert.Equal(t, "Atom Test", result.Feed.Title)
	assert.Len(t, result.Feed.Items, 1)
}

func TestMapToFetchOptions(t *testing.T) {
	input := map[string]interface{}{
		"user_agent":    "TestBot/1.0",
		"timeout":       float64(60),
		"etag":          `"xyz"`,
		"last_modified": "Wed, 01 Jan 2025 00:00:00 GMT",
	}

	opts, err := mapToFetchOptions(input)
	require.NoError(t, err)

	assert.Equal(t, "TestBot/1.0", opts.UserAgent)
	assert.Equal(t, 60, opts.Timeout)
	assert.Equal(t, `"xyz"`, opts.ETag)
	assert.Equal(t, "Wed, 01 Jan 2025 00:00:00 GMT", opts.LastModified)
}

func TestMapToFetchOptionsNil(t *testing.T) {
	opts, err := mapToFetchOptions(nil)
	require.NoError(t, err)
	assert.NotNil(t, opts)
}
