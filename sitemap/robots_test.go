package sitemap

import (
	"testing"
)

func TestParseRobotsBasic(t *testing.T) {
	text := `User-agent: *
Disallow: /private/

Sitemap: https://example.com/sitemap.xml
Sitemap: https://example.com/sitemap-news.xml
`
	urls := ParseRobots(text)
	if len(urls) != 2 {
		t.Fatalf("expected 2 URLs, got %d", len(urls))
	}
	if urls[0] != "https://example.com/sitemap.xml" {
		t.Errorf("unexpected URL: %s", urls[0])
	}
	if urls[1] != "https://example.com/sitemap-news.xml" {
		t.Errorf("unexpected URL: %s", urls[1])
	}
}

func TestParseRobotsCaseInsensitive(t *testing.T) {
	text := `sitemap: https://example.com/sitemap1.xml
SITEMAP: https://example.com/sitemap2.xml
SiteMap: https://example.com/sitemap3.xml
`
	urls := ParseRobots(text)
	if len(urls) != 3 {
		t.Fatalf("expected 3 URLs, got %d", len(urls))
	}
}

func TestParseRobotsDuplicate(t *testing.T) {
	text := `Sitemap: https://example.com/sitemap.xml
Sitemap: https://example.com/sitemap.xml
Sitemap: https://example.com/other.xml
`
	urls := ParseRobots(text)
	if len(urls) != 2 {
		t.Fatalf("expected 2 URLs (deduplicated), got %d", len(urls))
	}
}

func TestParseRobotsEmpty(t *testing.T) {
	urls := ParseRobots("")
	if len(urls) != 0 {
		t.Errorf("expected 0 URLs, got %d", len(urls))
	}
}

func TestParseRobotsNoSitemapDirective(t *testing.T) {
	text := `User-agent: *
Disallow: /
`
	urls := ParseRobots(text)
	if len(urls) != 0 {
		t.Errorf("expected 0 URLs, got %d", len(urls))
	}
}

func TestParseRobotsWithWhitespace(t *testing.T) {
	text := `  Sitemap:   https://example.com/sitemap.xml  
	Sitemap:	https://example.com/other.xml	
`
	urls := ParseRobots(text)
	if len(urls) != 2 {
		t.Fatalf("expected 2 URLs, got %d", len(urls))
	}
	if urls[0] != "https://example.com/sitemap.xml" {
		t.Errorf("unexpected URL (whitespace not trimmed): '%s'", urls[0])
	}
}
