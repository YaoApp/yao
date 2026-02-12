package sitemap

import (
	"compress/gzip"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// testSitemapXML is a small urlset for fetch testing.
const testSitemapXML = `<?xml version="1.0" encoding="UTF-8"?>
<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">
  <url><loc>https://example.com/page1</loc><lastmod>2025-01-01</lastmod></url>
  <url><loc>https://example.com/page2</loc><lastmod>2025-02-01</lastmod></url>
  <url><loc>https://example.com/page3</loc><lastmod>2025-03-01</lastmod></url>
  <url><loc>https://example.com/page4</loc></url>
  <url><loc>https://example.com/page5</loc></url>
</urlset>`

// buildSitemapIndex returns a sitemapindex XML referencing the given sitemap URLs.
func buildSitemapIndex(urls ...string) string {
	var sb strings.Builder
	sb.WriteString(`<?xml version="1.0" encoding="UTF-8"?>`)
	sb.WriteString(`<sitemapindex xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">`)
	for _, u := range urls {
		sb.WriteString(fmt.Sprintf(`<sitemap><loc>%s</loc></sitemap>`, u))
	}
	sb.WriteString(`</sitemapindex>`)
	return sb.String()
}

// ==================== streamParseURLs tests ====================

func TestStreamParseURLs_Basic(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/xml")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(testSitemapXML))
	}))
	defer server.Close()

	client := server.Client()
	link := SitemapLink{URL: server.URL + "/sitemap.xml"}

	urls, total, err := streamParseURLs(client, DefaultUserAgent, link, 0, 100)
	if err != nil {
		t.Fatalf("streamParseURLs failed: %s", err.Error())
	}
	if total != 5 {
		t.Errorf("expected total=5, got %d", total)
	}
	if len(urls) != 5 {
		t.Errorf("expected 5 URLs, got %d", len(urls))
	}
	if urls[0].Loc != "https://example.com/page1" {
		t.Errorf("unexpected first URL: %s", urls[0].Loc)
	}
}

func TestStreamParseURLs_WithSkip(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(testSitemapXML))
	}))
	defer server.Close()

	client := server.Client()
	link := SitemapLink{URL: server.URL + "/sitemap.xml"}

	// Skip 2, take up to 100
	urls, total, err := streamParseURLs(client, DefaultUserAgent, link, 2, 100)
	if err != nil {
		t.Fatalf("streamParseURLs failed: %s", err.Error())
	}
	if total != 5 {
		t.Errorf("expected total=5, got %d", total)
	}
	if len(urls) != 3 {
		t.Errorf("expected 3 URLs (5 - 2 skipped), got %d", len(urls))
	}
	if urls[0].Loc != "https://example.com/page3" {
		t.Errorf("expected page3 as first result after skip, got %s", urls[0].Loc)
	}
}

func TestStreamParseURLs_WithLimit(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(testSitemapXML))
	}))
	defer server.Close()

	client := server.Client()
	link := SitemapLink{URL: server.URL + "/sitemap.xml"}

	// No skip, limit 2
	urls, _, err := streamParseURLs(client, DefaultUserAgent, link, 0, 2)
	if err != nil {
		t.Fatalf("streamParseURLs failed: %s", err.Error())
	}
	if len(urls) != 2 {
		t.Errorf("expected 2 URLs (limit=2), got %d", len(urls))
	}
	if urls[0].Loc != "https://example.com/page1" {
		t.Errorf("unexpected first URL: %s", urls[0].Loc)
	}
	if urls[1].Loc != "https://example.com/page2" {
		t.Errorf("unexpected second URL: %s", urls[1].Loc)
	}
}

func TestStreamParseURLs_SkipAndLimit(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(testSitemapXML))
	}))
	defer server.Close()

	client := server.Client()
	link := SitemapLink{URL: server.URL + "/sitemap.xml"}

	// Skip 1, limit 2
	urls, _, err := streamParseURLs(client, DefaultUserAgent, link, 1, 2)
	if err != nil {
		t.Fatalf("streamParseURLs failed: %s", err.Error())
	}
	if len(urls) != 2 {
		t.Errorf("expected 2 URLs, got %d", len(urls))
	}
	if urls[0].Loc != "https://example.com/page2" {
		t.Errorf("expected page2, got %s", urls[0].Loc)
	}
	if urls[1].Loc != "https://example.com/page3" {
		t.Errorf("expected page3, got %s", urls[1].Loc)
	}
}

func TestStreamParseURLs_SkipAll(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(testSitemapXML))
	}))
	defer server.Close()

	client := server.Client()
	link := SitemapLink{URL: server.URL + "/sitemap.xml"}

	// Skip more than total
	urls, total, err := streamParseURLs(client, DefaultUserAgent, link, 100, 50)
	if err != nil {
		t.Fatalf("streamParseURLs failed: %s", err.Error())
	}
	if len(urls) != 0 {
		t.Errorf("expected 0 URLs when skip > total, got %d", len(urls))
	}
	if total != 5 {
		t.Errorf("expected total=5, got %d", total)
	}
}

func TestStreamParseURLs_Gzip(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Go's default transport auto-decompresses gzip when Content-Encoding
		// is set. To test our manual gzip handling, we use a custom content type
		// and set the Encoding on the SitemapLink instead. Here we just serve
		// raw gzip bytes without Content-Encoding header so Go won't auto-decompress.
		w.Header().Set("Content-Type", "application/x-gzip")
		w.WriteHeader(http.StatusOK)
		gz := gzip.NewWriter(w)
		gz.Write([]byte(testSitemapXML))
		gz.Close()
	}))
	defer server.Close()

	client := server.Client()
	// SitemapLink.Encoding = "gzip" triggers our manual decompression path
	link := SitemapLink{URL: server.URL + "/sitemap.xml.gz", Encoding: "gzip"}

	urls, total, err := streamParseURLs(client, DefaultUserAgent, link, 0, 100)
	if err != nil {
		t.Fatalf("streamParseURLs with gzip failed: %s", err.Error())
	}
	if total != 5 {
		t.Errorf("expected total=5, got %d", total)
	}
	if len(urls) != 5 {
		t.Errorf("expected 5 URLs, got %d", len(urls))
	}
}

func TestStreamParseURLs_HTTP404(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	client := server.Client()
	link := SitemapLink{URL: server.URL + "/missing.xml"}

	_, _, err := streamParseURLs(client, DefaultUserAgent, link, 0, 100)
	if err == nil {
		t.Error("expected error for 404")
	}
}

// ==================== Discover tests (with httptest mock site) ====================

func TestDiscover_SingleURLSet(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/sitemap.xml":
			w.Header().Set("Content-Type", "application/xml")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(testSitemapXML))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	// Discover hardcodes https:// prefix, but we need to use the httptest server.
	// Test classifyAndExpand directly instead.
	client := server.Client()
	links, err := classifyAndExpand(client, DefaultUserAgent, server.URL+"/sitemap.xml", "well-known", 0)
	if err != nil {
		t.Fatalf("classifyAndExpand failed: %s", err.Error())
	}
	if len(links) != 1 {
		t.Fatalf("expected 1 link, got %d", len(links))
	}
	if links[0].URL != server.URL+"/sitemap.xml" {
		t.Errorf("unexpected URL: %s", links[0].URL)
	}
	if links[0].Source != "well-known" {
		t.Errorf("unexpected source: %s", links[0].Source)
	}
}

func TestDiscover_SitemapIndex(t *testing.T) {
	var serverURL string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/sitemap_index.xml":
			indexXML := buildSitemapIndex(
				serverURL+"/sitemap1.xml",
				serverURL+"/sitemap2.xml",
			)
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(indexXML))
		case "/sitemap1.xml":
			w.Header().Set("Content-Length", "900")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(testSitemapXML))
		case "/sitemap2.xml":
			w.Header().Set("Content-Length", "900")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(testSitemapXML))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()
	serverURL = server.URL

	client := server.Client()
	links, err := classifyAndExpand(client, DefaultUserAgent, server.URL+"/sitemap_index.xml", "robots.txt", 0)
	if err != nil {
		t.Fatalf("classifyAndExpand for index failed: %s", err.Error())
	}
	if len(links) != 2 {
		t.Fatalf("expected 2 leaf sitemaps, got %d", len(links))
	}
	if links[0].URL != server.URL+"/sitemap1.xml" {
		t.Errorf("unexpected first link: %s", links[0].URL)
	}
	if links[1].URL != server.URL+"/sitemap2.xml" {
		t.Errorf("unexpected second link: %s", links[1].URL)
	}
}

func TestDiscover_UnreachableSitemap(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	client := server.Client()
	_, err := classifyAndExpand(client, DefaultUserAgent, server.URL+"/sitemap.xml", "test", 0)
	if err == nil {
		t.Error("expected error for 500 response")
	}
}

// ==================== estimateURLCount tests ====================

func TestEstimateURLCount(t *testing.T) {
	tests := []struct {
		size     int64
		encoding string
		expected int
	}{
		{0, "", 0},
		{-1, "", 0},
		{300, "", 1},
		{3000, "", 10},
		{150, "", 1},
		{600, "gzip", 10},  // 600 * 5 / 300 = 10
		{600, "br", 10},    // same ratio
		{3000, "gzip", 50}, // 3000 * 5 / 300 = 50
	}

	for _, tt := range tests {
		got := estimateURLCount(tt.size, tt.encoding)
		if got != tt.expected {
			t.Errorf("estimateURLCount(%d, %q) = %d, want %d", tt.size, tt.encoding, got, tt.expected)
		}
	}
}

// ==================== httpGetBody tests ====================

func TestHTTPGetBody(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("hello world"))
	}))
	defer server.Close()

	client := server.Client()
	body, err := httpGetBody(client, server.URL, DefaultUserAgent)
	if err != nil {
		t.Fatalf("httpGetBody failed: %s", err.Error())
	}
	if body != "hello world" {
		t.Errorf("expected 'hello world', got '%s'", body)
	}
}

func TestHTTPGetBody_404(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	client := server.Client()
	_, err := httpGetBody(client, server.URL, DefaultUserAgent)
	if err == nil {
		t.Error("expected error for 404")
	}
}

// ==================== End-to-end Fetch via streamParseURLs ====================

func TestFetchEndToEnd_Pagination(t *testing.T) {
	// Build a sitemap with 10 URLs
	var sb strings.Builder
	sb.WriteString(`<?xml version="1.0" encoding="UTF-8"?>`)
	sb.WriteString(`<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">`)
	for i := 1; i <= 10; i++ {
		sb.WriteString(fmt.Sprintf(`<url><loc>https://example.com/p%d</loc></url>`, i))
	}
	sb.WriteString(`</urlset>`)
	tenURLsSitemap := sb.String()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(tenURLsSitemap))
	}))
	defer server.Close()

	client := server.Client()
	link := SitemapLink{URL: server.URL + "/sitemap.xml"}

	// Page 1: offset=0, limit=3
	urls1, _, err := streamParseURLs(client, DefaultUserAgent, link, 0, 3)
	if err != nil {
		t.Fatalf("page 1 failed: %s", err.Error())
	}
	if len(urls1) != 3 {
		t.Fatalf("page 1: expected 3, got %d", len(urls1))
	}
	if urls1[0].Loc != "https://example.com/p1" {
		t.Errorf("page 1 first: expected p1, got %s", urls1[0].Loc)
	}
	if urls1[2].Loc != "https://example.com/p3" {
		t.Errorf("page 1 last: expected p3, got %s", urls1[2].Loc)
	}

	// Page 2: offset=3, limit=3
	urls2, _, err := streamParseURLs(client, DefaultUserAgent, link, 3, 3)
	if err != nil {
		t.Fatalf("page 2 failed: %s", err.Error())
	}
	if len(urls2) != 3 {
		t.Fatalf("page 2: expected 3, got %d", len(urls2))
	}
	if urls2[0].Loc != "https://example.com/p4" {
		t.Errorf("page 2 first: expected p4, got %s", urls2[0].Loc)
	}

	// Page 4: offset=9, limit=3 — should get only 1 URL
	urls4, _, err := streamParseURLs(client, DefaultUserAgent, link, 9, 3)
	if err != nil {
		t.Fatalf("page 4 failed: %s", err.Error())
	}
	if len(urls4) != 1 {
		t.Fatalf("page 4: expected 1, got %d", len(urls4))
	}
	if urls4[0].Loc != "https://example.com/p10" {
		t.Errorf("page 4: expected p10, got %s", urls4[0].Loc)
	}
}

// ==================== fillMetadataFromHeaders test ====================

func TestFillMetadataFromHeaders(t *testing.T) {
	body := "hello world test body"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Last-Modified", "Tue, 01 Jan 2025 00:00:00 GMT")
		w.Header().Set("ETag", `"etag123"`)
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(body))
	}))
	defer server.Close()

	client := server.Client()
	resp, err := client.Get(server.URL)
	if err != nil {
		t.Fatalf("GET failed: %s", err.Error())
	}
	defer resp.Body.Close()

	link := SitemapLink{URL: server.URL}
	fillMetadataFromHeaders(&link, resp)

	// Content-Length is set automatically by httptest when body is written
	if link.ContentSize < 0 {
		t.Errorf("expected non-negative ContentSize, got %d", link.ContentSize)
	}
	if link.LastModified != "Tue, 01 Jan 2025 00:00:00 GMT" {
		t.Errorf("unexpected LastModified: %s", link.LastModified)
	}
	if link.ETag != `"etag123"` {
		t.Errorf("unexpected ETag: %s", link.ETag)
	}
}
