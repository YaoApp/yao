package webfetch

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestTaoFetch_Endpoint(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/v1/fetch" {
			t.Errorf("expected /v1/fetch, got %s", r.URL.Path)
		}
		if r.Header.Get("Authorization") != "Bearer test-key" {
			t.Errorf("expected Bearer test-key, got %s", r.Header.Get("Authorization"))
		}

		var body map[string]interface{}
		json.NewDecoder(r.Body).Decode(&body)
		if body["url"] != "https://example.com" {
			t.Errorf("expected url 'https://example.com', got %v", body["url"])
		}
		if body["format"] != "markdown" {
			t.Errorf("expected format 'markdown', got %v", body["format"])
		}

		w.Write([]byte("# Example\n\nHello world"))
	}))
	defer srv.Close()

	cfg := &fetchConfig{
		Provider: "tao",
		APIKey:   "test-key",
		APIURL:   srv.URL,
	}
	resp := taoFetch(cfg, "https://example.com", "markdown")
	if resp.Content != "# Example\n\nHello world" {
		t.Errorf("expected markdown content, got %q", resp.Content)
	}
	if resp.Format != "markdown" {
		t.Errorf("expected format 'markdown', got %q", resp.Format)
	}
}

func TestTaoFetch_JSONResponse(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		json.NewEncoder(w).Encode(map[string]interface{}{
			"title":   "Example Domain",
			"content": "<html><body>hello</body></html>",
		})
	}))
	defer srv.Close()

	cfg := &fetchConfig{Provider: "tao", APIKey: "k", APIURL: srv.URL}
	resp := taoFetch(cfg, "https://example.com", "html")
	if resp.Title != "Example Domain" {
		t.Errorf("expected title 'Example Domain', got %q", resp.Title)
	}
	if resp.Content != "<html><body>hello</body></html>" {
		t.Errorf("expected HTML content, got %q", resp.Content)
	}
}

func TestTaoFetch_MissingCredentials(t *testing.T) {
	cfg := &fetchConfig{Provider: "tao", APIKey: "", APIURL: ""}
	resp := taoFetch(cfg, "https://example.com", "markdown")
	if resp.Content != "tao service not configured" {
		t.Errorf("expected not-configured message, got %q", resp.Content)
	}
}

func TestTaoFetch_ServerError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadGateway)
		w.Write([]byte("target unreachable"))
	}))
	defer srv.Close()

	cfg := &fetchConfig{Provider: "tao", APIKey: "k", APIURL: srv.URL}
	resp := taoFetch(cfg, "https://down.example.com", "markdown")
	if resp.URL != "https://down.example.com" {
		t.Errorf("expected original URL preserved, got %q", resp.URL)
	}
	if resp.Content == "" {
		t.Error("expected error content, got empty")
	}
}
