package websearch

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
)

func withOutcrawlURL(t *testing.T, url string) {
	t.Helper()
	orig := outcrawlSearchURL
	outcrawlSearchURL = url
	t.Cleanup(func() { outcrawlSearchURL = orig })
}

func TestOutcrawlSearch_Success(t *testing.T) {
	var gotAuth, gotType, gotMethod string
	var gotBody map[string]interface{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		gotType = r.Header.Get("Content-Type")
		gotMethod = r.Method
		json.NewDecoder(r.Body).Decode(&gotBody)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"hits": []map[string]interface{}{
				{"title": "Yao", "url": "https://yaoapps.com", "snippet": "Yao application engine."},
				{"title": "Docs", "url": "https://yaoapps.com/docs", "snippet": "Documentation."},
			},
		})
	}))
	defer srv.Close()
	withOutcrawlURL(t, srv.URL)

	results := outcrawlSearch("test-key", "yao engine", 2)
	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}
	if results[0].Title != "Yao" || results[0].URL != "https://yaoapps.com" || results[0].Content != "Yao application engine." {
		t.Errorf("unexpected first result: %+v", results[0])
	}
	if gotMethod != http.MethodPost {
		t.Errorf("expected POST, got %s", gotMethod)
	}
	if gotAuth != "Bearer test-key" {
		t.Errorf("expected bearer auth, got %q", gotAuth)
	}
	if gotType != "application/json" {
		t.Errorf("expected JSON content type, got %q", gotType)
	}
	if gotBody["query"] != "yao engine" {
		t.Errorf("expected query in body, got %v", gotBody["query"])
	}
	if gotBody["limit"] != float64(2) {
		t.Errorf("expected limit 2 in body, got %v", gotBody["limit"])
	}
}

func TestOutcrawlSearch_NoKey(t *testing.T) {
	if results := outcrawlSearch("", "test", 5); results != nil {
		t.Error("expected nil results with empty key")
	}
}

func TestOutcrawlSearch_HTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"code":"unauthorized","message":"invalid key"}`))
	}))
	defer srv.Close()
	withOutcrawlURL(t, srv.URL)

	results := outcrawlSearch("bad-key", "test", 5)
	if len(results) != 1 || results[0].Title != "Error" {
		t.Fatalf("expected a single error result, got %+v", results)
	}
	if !strings.Contains(results[0].Content, "HTTP 401") {
		t.Errorf("expected status in error content, got %q", results[0].Content)
	}
}

func TestOutcrawlSearch_InvalidJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("not json"))
	}))
	defer srv.Close()
	withOutcrawlURL(t, srv.URL)

	results := outcrawlSearch("test-key", "test", 5)
	if len(results) != 1 || results[0].Title != "Error" {
		t.Fatalf("expected a single error result, got %+v", results)
	}
}

func TestSearchWithConfig_OutcrawlProvider(t *testing.T) {
	var gotAuth string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		w.Write([]byte(`{"hits":[{"title":"A","url":"https://a.example","snippet":"a"}]}`))
	}))
	defer srv.Close()
	withOutcrawlURL(t, srv.URL)

	cfg := &searchConfig{Provider: "outcrawl", APIKey: "cfg-key"}
	results := searchWithConfig(cfg, "q", 1)
	if len(results) != 1 || results[0].URL != "https://a.example" {
		t.Fatalf("unexpected results: %+v", results)
	}
	if gotAuth != "Bearer cfg-key" {
		t.Errorf("expected bearer cfg-key, got %q", gotAuth)
	}
}

func TestOutcrawlSearch_Live(t *testing.T) {
	key := os.Getenv("OUTCRAWL_API_KEY")
	if key == "" {
		t.Skip("OUTCRAWL_API_KEY not set")
	}

	results := outcrawlSearch(key, "Yao application engine", 3)
	if len(results) == 0 {
		t.Fatal("expected at least one result from outcrawl")
	}
	for _, r := range results {
		if r.Title == "Error" {
			t.Fatalf("outcrawl returned error: %s", r.Content)
		}
		if r.URL == "" {
			t.Error("expected non-empty URL")
		}
	}
	t.Logf("got %d results", len(results))
}
