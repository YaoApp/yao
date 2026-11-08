package webfetch

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
)

func withOutcrawlEndpoint(t *testing.T, url string) {
	t.Helper()
	orig := outcrawlEndpoint
	outcrawlEndpoint = url
	t.Cleanup(func() { outcrawlEndpoint = orig })
}

func TestOutcrawlFetch_Markdown(t *testing.T) {
	var gotAuth, gotMethod string
	var gotBody map[string]interface{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		gotMethod = r.Method
		json.NewDecoder(r.Body).Decode(&gotBody)
		w.Write([]byte(`{"url":"https://example.com/","markdown":"# Hello\n\nWorld","metadata":{"title":"Example"}}`))
	}))
	defer srv.Close()
	withOutcrawlEndpoint(t, srv.URL)

	resp, err := outcrawlFetch("https://example.com", "test-key", "markdown")
	if err != nil {
		t.Fatalf("outcrawlFetch failed: %v", err)
	}
	if resp.Content != "# Hello\n\nWorld" || resp.Title != "Example" || resp.Format != "markdown" {
		t.Errorf("unexpected response: %+v", resp)
	}
	if resp.URL != "https://example.com" {
		t.Errorf("expected requested URL, got %q", resp.URL)
	}
	if gotMethod != http.MethodPost || gotAuth != "Bearer test-key" {
		t.Errorf("unexpected method/auth: %s %q", gotMethod, gotAuth)
	}
	if gotBody["url"] != "https://example.com" {
		t.Errorf("expected url in body, got %v", gotBody["url"])
	}
	formats, _ := gotBody["formats"].([]interface{})
	if len(formats) != 1 || formats[0] != "markdown" {
		t.Errorf("expected formats [markdown], got %v", gotBody["formats"])
	}
}

func TestOutcrawlFetch_HTML(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]interface{}
		json.NewDecoder(r.Body).Decode(&body)
		formats, _ := body["formats"].([]interface{})
		if len(formats) != 1 || formats[0] != "html" {
			t.Errorf("expected formats [html], got %v", body["formats"])
		}
		w.Write([]byte(`{"html":"<p>Hi</p>"}`))
	}))
	defer srv.Close()
	withOutcrawlEndpoint(t, srv.URL)

	resp, err := outcrawlFetch("https://example.com", "test-key", "html")
	if err != nil {
		t.Fatalf("outcrawlFetch failed: %v", err)
	}
	if resp.Content != "<p>Hi</p>" || resp.Format != "html" {
		t.Errorf("unexpected response: %+v", resp)
	}
}

func TestOutcrawlFetch_Errors(t *testing.T) {
	tests := []struct {
		name   string
		status int
		body   string
		want   string
	}{
		{"http error", http.StatusUnauthorized, `{"code":"unauthorized","message":"bad key"}`, "HTTP 401"},
		{"invalid json", http.StatusOK, "not json", "parse"},
		{"empty content", http.StatusOK, `{"markdown":""}`, "no markdown content"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.status)
				w.Write([]byte(tt.body))
			}))
			defer srv.Close()
			withOutcrawlEndpoint(t, srv.URL)

			resp, err := outcrawlFetch("https://example.com", "test-key", "markdown")
			if err == nil || resp != nil {
				t.Fatalf("expected error, got resp=%+v err=%v", resp, err)
			}
			if !strings.Contains(err.Error(), tt.want) {
				t.Errorf("expected error containing %q, got %v", tt.want, err)
			}
		})
	}
}

func TestFetchWithConfig_OutcrawlProvider(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"markdown":"from outcrawl","metadata":{"title":"OC"}}`))
	}))
	defer srv.Close()
	withOutcrawlEndpoint(t, srv.URL)

	directSrv := newDirectServer(directContent)
	defer directSrv.Close()

	cfg := &fetchConfig{Provider: "outcrawl", OutcrawlKey: "test-key"}
	resp := fetchWithConfig(cfg, directSrv.URL, "markdown")
	if resp.Content != "from outcrawl" || resp.Title != "OC" {
		t.Errorf("expected Outcrawl content, got %+v", resp)
	}
}

func TestFetchWithConfig_OutcrawlProvider_FallsBackToDirect(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()
	withOutcrawlEndpoint(t, srv.URL)

	directSrv := newDirectServer(directContent)
	defer directSrv.Close()

	cfg := &fetchConfig{Provider: "outcrawl", OutcrawlKey: "test-key"}
	resp := fetchWithConfig(cfg, directSrv.URL, "html")
	if resp.Title != "Direct" {
		t.Errorf("expected fallback to direct fetch, got %+v", resp)
	}
}

func TestFetchWithConfig_OutcrawlProvider_NoKeyUsesDirect(t *testing.T) {
	var called bool
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
	}))
	defer srv.Close()
	withOutcrawlEndpoint(t, srv.URL)

	directSrv := newDirectServer(directContent)
	defer directSrv.Close()

	cfg := &fetchConfig{Provider: "outcrawl"}
	resp := fetchWithConfig(cfg, directSrv.URL, "html")
	if called {
		t.Error("outcrawl must not be called without a key")
	}
	if resp.Title != "Direct" {
		t.Errorf("expected direct fetch, got %+v", resp)
	}
}

func TestOutcrawlFetch_Live(t *testing.T) {
	key := os.Getenv("OUTCRAWL_API_KEY")
	if key == "" {
		t.Skip("OUTCRAWL_API_KEY not set")
	}

	resp, err := outcrawlFetch("https://example.com", key, "markdown")
	if err != nil {
		t.Fatalf("outcrawlFetch failed: %v", err)
	}
	if resp.Content == "" {
		t.Error("expected non-empty content")
	}
}
