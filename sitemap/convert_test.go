package sitemap

import (
	"testing"
)

func TestMapToURLs(t *testing.T) {
	// Simulate JS-side data: []interface{} of map[string]interface{}
	input := []interface{}{
		map[string]interface{}{
			"loc":        "https://example.com/page1",
			"lastmod":    "2025-01-01",
			"changefreq": "daily",
			"priority":   "0.8",
		},
		map[string]interface{}{
			"loc": "https://example.com/page2",
		},
	}

	urls, err := mapToURLs(input)
	if err != nil {
		t.Fatalf("mapToURLs failed: %s", err.Error())
	}

	if len(urls) != 2 {
		t.Fatalf("expected 2 URLs, got %d", len(urls))
	}
	if urls[0].Loc != "https://example.com/page1" {
		t.Errorf("expected loc, got '%s'", urls[0].Loc)
	}
	if urls[0].ChangeFreq != "daily" {
		t.Errorf("expected changefreq 'daily', got '%s'", urls[0].ChangeFreq)
	}
}

func TestMapToURLsWithImages(t *testing.T) {
	input := []interface{}{
		map[string]interface{}{
			"loc": "https://example.com/gallery",
			"images": []interface{}{
				map[string]interface{}{
					"loc":     "https://example.com/img/1.jpg",
					"caption": "Photo 1",
				},
			},
		},
	}

	urls, err := mapToURLs(input)
	if err != nil {
		t.Fatalf("mapToURLs failed: %s", err.Error())
	}

	if len(urls) != 1 {
		t.Fatalf("expected 1 URL, got %d", len(urls))
	}
	if len(urls[0].Images) != 1 {
		t.Fatalf("expected 1 image, got %d", len(urls[0].Images))
	}
	if urls[0].Images[0].Loc != "https://example.com/img/1.jpg" {
		t.Errorf("unexpected image loc: %s", urls[0].Images[0].Loc)
	}
}

func TestMapToURLsNil(t *testing.T) {
	_, err := mapToURLs(nil)
	if err == nil {
		t.Error("expected error for nil input")
	}
}

func TestMapToURLsAlreadyTyped(t *testing.T) {
	input := []URL{
		{Loc: "https://example.com/typed"},
	}
	urls, err := mapToURLs(input)
	if err != nil {
		t.Fatalf("mapToURLs failed: %s", err.Error())
	}
	if len(urls) != 1 || urls[0].Loc != "https://example.com/typed" {
		t.Error("expected passthrough for already-typed input")
	}
}

func TestMapToBuildOptions(t *testing.T) {
	input := map[string]interface{}{
		"dir":      "/tmp/sitemaps",
		"base_url": "https://example.com",
	}

	opts, err := mapToBuildOptions(input)
	if err != nil {
		t.Fatalf("mapToBuildOptions failed: %s", err.Error())
	}
	if opts.Dir != "/tmp/sitemaps" {
		t.Errorf("expected dir '/tmp/sitemaps', got '%s'", opts.Dir)
	}
	if opts.BaseURL != "https://example.com" {
		t.Errorf("expected base_url 'https://example.com', got '%s'", opts.BaseURL)
	}
}

func TestMapToBuildOptionsNil(t *testing.T) {
	_, err := mapToBuildOptions(nil)
	if err == nil {
		t.Error("expected error for nil input")
	}
}

func TestMapToDiscoverOptions(t *testing.T) {
	input := map[string]interface{}{
		"user_agent": "TestBot/1.0",
		"timeout":    float64(60),
	}

	opts, err := mapToDiscoverOptions(input)
	if err != nil {
		t.Fatalf("mapToDiscoverOptions failed: %s", err.Error())
	}
	if opts.UserAgent != "TestBot/1.0" {
		t.Errorf("expected user_agent 'TestBot/1.0', got '%s'", opts.UserAgent)
	}
	if opts.Timeout != 60 {
		t.Errorf("expected timeout 60, got %d", opts.Timeout)
	}
}

func TestMapToDiscoverOptionsNil(t *testing.T) {
	opts, err := mapToDiscoverOptions(nil)
	if err != nil {
		t.Fatalf("expected nil to return empty options, got error: %s", err.Error())
	}
	if opts == nil {
		t.Error("expected non-nil options")
	}
}

func TestMapToFetchOptions(t *testing.T) {
	input := map[string]interface{}{
		"offset": float64(100),
		"limit":  float64(50),
	}

	opts, err := mapToFetchOptions(input)
	if err != nil {
		t.Fatalf("mapToFetchOptions failed: %s", err.Error())
	}
	if opts.Offset != 100 {
		t.Errorf("expected offset 100, got %d", opts.Offset)
	}
	if opts.Limit != 50 {
		t.Errorf("expected limit 50, got %d", opts.Limit)
	}
}

func TestMapToFetchOptionsNil(t *testing.T) {
	opts, err := mapToFetchOptions(nil)
	if err != nil {
		t.Fatalf("expected nil to return empty options, got error: %s", err.Error())
	}
	if opts == nil {
		t.Error("expected non-nil options")
	}
}
