package webfetch

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
)

func newDirectServer(body string) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(body))
	}))
}

func newBrightdataServer(body string) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(body))
	}))
}

var (
	directContent  = "<html><head><title>Direct</title></head><body><p>" + strings.Repeat("direct-content ", 40) + "</p></body></html>"
	brightdataHTML = "<html><head><title>Brightdata</title></head><body><p>" + strings.Repeat("brightdata-content ", 40) + "</p></body></html>"
)

func TestFetchHTML_BrightdataProvider_PrefersProxy(t *testing.T) {
	directSrv := newDirectServer(directContent)
	defer directSrv.Close()

	bdSrv := newBrightdataServer(brightdataHTML)
	defer bdSrv.Close()

	origEndpoint := brightdataEndpoint
	brightdataEndpoint = bdSrv.URL
	defer func() { brightdataEndpoint = origEndpoint }()

	cfg := &fetchConfig{
		Provider:       "brightdata",
		BrightdataKey:  "test-key",
		BrightdataZone: "test-zone",
	}

	resp := fetchHTML(cfg, directSrv.URL)
	if resp == nil {
		t.Fatal("expected non-nil response")
	}
	if resp.Title != "Brightdata" {
		t.Errorf("expected Brightdata content first, got title=%q", resp.Title)
	}
}

func TestFetchHTML_DefaultProvider_PrefersDirect(t *testing.T) {
	directSrv := newDirectServer(directContent)
	defer directSrv.Close()

	bdSrv := newBrightdataServer(brightdataHTML)
	defer bdSrv.Close()

	origEndpoint := brightdataEndpoint
	brightdataEndpoint = bdSrv.URL
	defer func() { brightdataEndpoint = origEndpoint }()

	cfg := &fetchConfig{
		Provider:       "",
		BrightdataKey:  "test-key",
		BrightdataZone: "test-zone",
	}

	resp := fetchHTML(cfg, directSrv.URL)
	if resp == nil {
		t.Fatal("expected non-nil response")
	}
	if resp.Title != "Direct" {
		t.Errorf("expected direct content first, got title=%q", resp.Title)
	}
}

func TestFetchHTML_BrightdataProvider_FallsBackToDirect(t *testing.T) {
	directSrv := newDirectServer(directContent)
	defer directSrv.Close()

	// Brightdata returns 500 → should fall back to direct
	bdSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("error"))
	}))
	defer bdSrv.Close()

	origEndpoint := brightdataEndpoint
	brightdataEndpoint = bdSrv.URL
	defer func() { brightdataEndpoint = origEndpoint }()

	cfg := &fetchConfig{
		Provider:       "brightdata",
		BrightdataKey:  "test-key",
		BrightdataZone: "test-zone",
	}

	resp := fetchHTML(cfg, directSrv.URL)
	if resp == nil {
		t.Fatal("expected non-nil response")
	}
	if resp.Title != "Direct" {
		t.Errorf("expected fallback to direct, got title=%q", resp.Title)
	}
}

func TestFetchHTML_DefaultProvider_FallsBackToBrightdata(t *testing.T) {
	// Direct returns 403 → should fall back to Brightdata
	directSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		w.Write([]byte("forbidden"))
	}))
	defer directSrv.Close()

	bdSrv := newBrightdataServer(brightdataHTML)
	defer bdSrv.Close()

	origEndpoint := brightdataEndpoint
	brightdataEndpoint = bdSrv.URL
	defer func() { brightdataEndpoint = origEndpoint }()

	cfg := &fetchConfig{
		Provider:       "",
		BrightdataKey:  "test-key",
		BrightdataZone: "test-zone",
	}

	resp := fetchHTML(cfg, directSrv.URL)
	if resp == nil {
		t.Fatal("expected non-nil response")
	}
	if resp.Title != "Brightdata" {
		t.Errorf("expected fallback to brightdata, got title=%q", resp.Title)
	}
}

func TestFetchRawHTML_BrightdataProvider_PrefersProxy(t *testing.T) {
	directSrv := newDirectServer(directContent)
	defer directSrv.Close()

	bdSrv := newBrightdataServer(brightdataHTML)
	defer bdSrv.Close()

	origEndpoint := brightdataEndpoint
	brightdataEndpoint = bdSrv.URL
	defer func() { brightdataEndpoint = origEndpoint }()

	cfg := &fetchConfig{
		Provider:       "brightdata",
		BrightdataKey:  "test-key",
		BrightdataZone: "test-zone",
	}

	body := fetchRawHTML(cfg, directSrv.URL)
	if body == nil {
		t.Fatal("expected non-nil body")
	}
	if !strings.Contains(string(body), "Brightdata") {
		t.Error("expected Brightdata content when provider is brightdata")
	}
}

func TestFetchRawHTML_DefaultProvider_PrefersDirect(t *testing.T) {
	directSrv := newDirectServer(directContent)
	defer directSrv.Close()

	bdSrv := newBrightdataServer(brightdataHTML)
	defer bdSrv.Close()

	origEndpoint := brightdataEndpoint
	brightdataEndpoint = bdSrv.URL
	defer func() { brightdataEndpoint = origEndpoint }()

	cfg := &fetchConfig{
		Provider:       "",
		BrightdataKey:  "test-key",
		BrightdataZone: "test-zone",
	}

	body := fetchRawHTML(cfg, directSrv.URL)
	if body == nil {
		t.Fatal("expected non-nil body")
	}
	if !strings.Contains(string(body), "Direct") {
		t.Error("expected direct content when provider is empty")
	}
}

func TestFetchRawHTML_BrightdataProvider_FallsBackToDirect(t *testing.T) {
	directSrv := newDirectServer(directContent)
	defer directSrv.Close()

	bdSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer bdSrv.Close()

	origEndpoint := brightdataEndpoint
	brightdataEndpoint = bdSrv.URL
	defer func() { brightdataEndpoint = origEndpoint }()

	cfg := &fetchConfig{
		Provider:       "brightdata",
		BrightdataKey:  "test-key",
		BrightdataZone: "test-zone",
	}

	body := fetchRawHTML(cfg, directSrv.URL)
	if body == nil {
		t.Fatal("expected non-nil body from direct fallback")
	}
	if !strings.Contains(string(body), "Direct") {
		t.Error("expected direct content as fallback")
	}
}

func TestFetchHTML_BrightdataProvider_NeverCallsDirect_WhenProxySucceeds(t *testing.T) {
	var directCalls atomic.Int32
	directSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		directCalls.Add(1)
		w.Write([]byte(directContent))
	}))
	defer directSrv.Close()

	bdSrv := newBrightdataServer(brightdataHTML)
	defer bdSrv.Close()

	origEndpoint := brightdataEndpoint
	brightdataEndpoint = bdSrv.URL
	defer func() { brightdataEndpoint = origEndpoint }()

	cfg := &fetchConfig{
		Provider:       "brightdata",
		BrightdataKey:  "test-key",
		BrightdataZone: "test-zone",
	}

	resp := fetchHTML(cfg, directSrv.URL)
	if resp == nil {
		t.Fatal("expected non-nil response")
	}
	if directCalls.Load() != 0 {
		t.Errorf("direct server should not be called when brightdata succeeds, got %d calls", directCalls.Load())
	}
}

func TestFetchMarkdown_BrightdataProvider_UsesProxy(t *testing.T) {
	directSrv := newDirectServer(directContent)
	defer directSrv.Close()

	bdSrv := newBrightdataServer(brightdataHTML)
	defer bdSrv.Close()

	origEndpoint := brightdataEndpoint
	brightdataEndpoint = bdSrv.URL
	defer func() { brightdataEndpoint = origEndpoint }()

	cfg := &fetchConfig{
		Provider:       "brightdata",
		BrightdataKey:  "test-key",
		BrightdataZone: "test-zone",
	}

	resp := fetchMarkdown(cfg, directSrv.URL)
	if resp == nil {
		t.Fatal("expected non-nil response")
	}
	if resp.Format != "markdown" {
		t.Errorf("expected format 'markdown', got '%s'", resp.Format)
	}
	if !strings.Contains(resp.Content, "Brightdata") {
		t.Errorf("expected Brightdata content in markdown, got: %s", resp.Content[:min(len(resp.Content), 200)])
	}
}
