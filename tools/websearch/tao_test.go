package websearch

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestTaoSearch_Endpoint(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/v1/search" {
			t.Errorf("expected /v1/search, got %s", r.URL.Path)
		}
		if r.Header.Get("Authorization") != "Bearer test-key" {
			t.Errorf("expected Bearer test-key, got %s", r.Header.Get("Authorization"))
		}

		var body map[string]interface{}
		json.NewDecoder(r.Body).Decode(&body)
		if body["query"] != "golang generics" {
			t.Errorf("expected query 'golang generics', got %v", body["query"])
		}
		if body["num"] != float64(5) {
			t.Errorf("expected num=5, got %v", body["num"])
		}
		if _, hasMaxResults := body["max_results"]; hasMaxResults {
			t.Error("body should not contain max_results; Tao uses 'num'")
		}

		json.NewEncoder(w).Encode(map[string]interface{}{
			"organic": []interface{}{
				map[string]interface{}{
					"title":   "Go Generics Tutorial",
					"link":    "https://go.dev/doc/tutorial/generics",
					"snippet": "Introduction to generics in Go.",
				},
				map[string]interface{}{
					"title":   "Go Blog: Generics",
					"link":    "https://go.dev/blog/generics",
					"snippet": "Generics are coming.",
				},
			},
		})
	}))
	defer srv.Close()

	cfg := &searchConfig{
		Provider: "tao",
		APIKey:   "test-key",
		APIURL:   srv.URL,
	}
	results := taoSearch(cfg, "golang generics", 5)
	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}
	if results[0].URL != "https://go.dev/doc/tutorial/generics" {
		t.Errorf("expected URL from 'link' field, got %q", results[0].URL)
	}
	if results[0].Content != "Introduction to generics in Go." {
		t.Errorf("expected snippet content, got %q", results[0].Content)
	}
}

func TestTaoSearch_MissingCredentials(t *testing.T) {
	cfg := &searchConfig{Provider: "tao", APIKey: "", APIURL: ""}
	results := taoSearch(cfg, "test", 5)
	if results != nil {
		t.Error("expected nil results with empty credentials")
	}
}

func TestParseTaoSearchResponse_Organic(t *testing.T) {
	body, _ := json.Marshal(map[string]interface{}{
		"organic": []interface{}{
			map[string]interface{}{"title": "A", "link": "https://a.com", "snippet": "snippet a"},
			map[string]interface{}{"title": "B", "link": "https://b.com", "snippet": "snippet b"},
		},
	})
	results := parseTaoSearchResponse(body)
	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}
	if results[1].URL != "https://b.com" {
		t.Errorf("expected https://b.com, got %q", results[1].URL)
	}
}

func TestParseTaoSearchResponse_LegacyResults(t *testing.T) {
	body, _ := json.Marshal(map[string]interface{}{
		"results": []interface{}{
			map[string]interface{}{"title": "Legacy", "url": "https://legacy.com", "snippet": "old format"},
		},
	})
	results := parseTaoSearchResponse(body)
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].URL != "https://legacy.com" {
		t.Errorf("expected https://legacy.com, got %q", results[0].URL)
	}
}

func TestParseTaoSearchResponse_Invalid(t *testing.T) {
	results := parseTaoSearchResponse([]byte("not json"))
	if len(results) != 1 || results[0].Title != "Error" {
		t.Error("expected error result for invalid JSON")
	}
}
