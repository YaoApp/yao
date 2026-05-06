package webfetch

import (
	"testing"
)

func TestDirectFetch_Success(t *testing.T) {
	res, err := directFetch("https://example.com", false)
	if err != nil {
		t.Fatalf("directFetch failed: %v", err)
	}
	if res.StatusCode != 200 {
		t.Errorf("expected 200, got %d", res.StatusCode)
	}
	if len(res.Body) < 100 {
		t.Error("expected body with at least 100 bytes")
	}
}

func TestDirectFetch_Bot(t *testing.T) {
	res, err := directFetch("https://example.com", true)
	if err != nil {
		t.Fatalf("directFetch with bot failed: %v", err)
	}
	if res.StatusCode != 200 {
		t.Errorf("expected 200, got %d", res.StatusCode)
	}
}

func TestFetchHTML_Local(t *testing.T) {
	cfg := &fetchConfig{}
	resp := fetchHTML(cfg, "https://example.com")
	if resp == nil {
		t.Fatal("expected non-nil response")
	}
	if resp.Format != "html" {
		t.Errorf("expected format 'html', got '%s'", resp.Format)
	}
	if resp.Content == "" {
		t.Error("expected non-empty content")
	}
}

func TestFetchMarkdown_Local(t *testing.T) {
	cfg := &fetchConfig{}
	resp := fetchMarkdown(cfg, "https://example.com")
	if resp == nil {
		t.Fatal("expected non-nil response")
	}
	if resp.Format != "markdown" {
		t.Errorf("expected format 'markdown', got '%s'", resp.Format)
	}
	if resp.Content == "" {
		t.Error("expected non-empty content")
	}
}

func TestBuildMdURL(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"https://example.com/docs/page", "https://example.com/docs/page.md"},
		{"https://example.com/docs/page/", "https://example.com/docs/page.md"},
		{"https://example.com/docs/page.md", ""},
		{"https://example.com/docs/page.MDX", ""},
	}
	for _, tt := range tests {
		got := buildMdURL(tt.input)
		if got != tt.expected {
			t.Errorf("buildMdURL(%q) = %q, want %q", tt.input, got, tt.expected)
		}
	}
}
