package websearch

import (
	"os"
	"testing"
)

func TestTavilySearch(t *testing.T) {
	key := os.Getenv("TAVILY_API_KEY")
	if key == "" {
		t.Skip("TAVILY_API_KEY not set")
	}

	results := tavilySearch(key, "Yao application engine", 3)
	if len(results) == 0 {
		t.Fatal("expected at least one result from tavily")
	}
	for _, r := range results {
		if r.Title == "Error" {
			t.Fatalf("tavily returned error: %s", r.Content)
		}
		if r.URL == "" {
			t.Error("expected non-empty URL")
		}
	}
	t.Logf("got %d results", len(results))
}

func TestSerperSearch(t *testing.T) {
	key := os.Getenv("SERPER_API_KEY")
	if key == "" {
		t.Skip("SERPER_API_KEY not set")
	}

	results := serperSearch(key, "Yao application engine", 3)
	if len(results) == 0 {
		t.Fatal("expected at least one result from serper")
	}
	for _, r := range results {
		if r.Title == "Error" {
			t.Fatalf("serper returned error: %s", r.Content)
		}
		if r.URL == "" {
			t.Error("expected non-empty URL")
		}
	}
	t.Logf("got %d results", len(results))
}

func TestTavilySearch_NoKey(t *testing.T) {
	results := tavilySearch("", "test", 5)
	if results != nil {
		t.Error("expected nil results with empty key")
	}
}

func TestSerperSearch_NoKey(t *testing.T) {
	results := serperSearch("", "test", 5)
	if results != nil {
		t.Error("expected nil results with empty key")
	}
}

func TestGetConfig_Defaults(t *testing.T) {
	cfg := getConfig("", "")
	if cfg.Provider != "tavily" {
		t.Errorf("expected default provider 'tavily', got '%s'", cfg.Provider)
	}
}
