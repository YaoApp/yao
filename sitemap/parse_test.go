package sitemap

import (
	"testing"
)

const testURLSetXML = `<?xml version="1.0" encoding="UTF-8"?>
<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9"
        xmlns:image="http://www.google.com/schemas/sitemap-image/1.1"
        xmlns:video="http://www.google.com/schemas/sitemap-video/1.1"
        xmlns:news="http://www.google.com/schemas/sitemap-news/0.9">
  <url>
    <loc>https://example.com/page1</loc>
    <lastmod>2025-01-01</lastmod>
    <changefreq>daily</changefreq>
    <priority>0.8</priority>
  </url>
  <url>
    <loc>https://example.com/page2</loc>
    <lastmod>2025-06-15</lastmod>
    <priority>0.5</priority>
  </url>
  <url>
    <loc>https://example.com/gallery</loc>
    <image:image>
      <image:loc>https://example.com/img/photo1.jpg</image:loc>
      <image:caption>A beautiful photo</image:caption>
    </image:image>
    <image:image>
      <image:loc>https://example.com/img/photo2.jpg</image:loc>
    </image:image>
  </url>
</urlset>`

const testSitemapIndexXML = `<?xml version="1.0" encoding="UTF-8"?>
<sitemapindex xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">
  <sitemap>
    <loc>https://example.com/sitemap1.xml</loc>
    <lastmod>2025-01-01</lastmod>
  </sitemap>
  <sitemap>
    <loc>https://example.com/sitemap2.xml</loc>
    <lastmod>2025-06-15</lastmod>
  </sitemap>
</sitemapindex>`

func TestParseURLSet(t *testing.T) {
	result, err := Parse(testURLSetXML)
	if err != nil {
		t.Fatalf("Parse urlset failed: %s", err.Error())
	}

	if result.Type != "urlset" {
		t.Errorf("expected type 'urlset', got '%s'", result.Type)
	}

	if len(result.URLs) != 3 {
		t.Fatalf("expected 3 URLs, got %d", len(result.URLs))
	}

	// Check first URL
	u := result.URLs[0]
	if u.Loc != "https://example.com/page1" {
		t.Errorf("expected loc 'https://example.com/page1', got '%s'", u.Loc)
	}
	if u.LastMod != "2025-01-01" {
		t.Errorf("expected lastmod '2025-01-01', got '%s'", u.LastMod)
	}
	if u.ChangeFreq != "daily" {
		t.Errorf("expected changefreq 'daily', got '%s'", u.ChangeFreq)
	}
	if u.Priority != "0.8" {
		t.Errorf("expected priority '0.8', got '%s'", u.Priority)
	}

	// Check third URL with images
	u3 := result.URLs[2]
	if len(u3.Images) != 2 {
		t.Fatalf("expected 2 images, got %d", len(u3.Images))
	}
	if u3.Images[0].Loc != "https://example.com/img/photo1.jpg" {
		t.Errorf("expected image loc, got '%s'", u3.Images[0].Loc)
	}
	if u3.Images[0].Caption != "A beautiful photo" {
		t.Errorf("expected caption 'A beautiful photo', got '%s'", u3.Images[0].Caption)
	}

	// Sitemaps should be nil/empty
	if len(result.Sitemaps) != 0 {
		t.Errorf("expected empty sitemaps, got %d", len(result.Sitemaps))
	}
}

func TestParseSitemapIndex(t *testing.T) {
	result, err := Parse(testSitemapIndexXML)
	if err != nil {
		t.Fatalf("Parse sitemapindex failed: %s", err.Error())
	}

	if result.Type != "sitemapindex" {
		t.Errorf("expected type 'sitemapindex', got '%s'", result.Type)
	}

	if len(result.Sitemaps) != 2 {
		t.Fatalf("expected 2 sitemaps, got %d", len(result.Sitemaps))
	}

	if result.Sitemaps[0].Loc != "https://example.com/sitemap1.xml" {
		t.Errorf("unexpected sitemap loc: %s", result.Sitemaps[0].Loc)
	}
	if result.Sitemaps[0].LastMod != "2025-01-01" {
		t.Errorf("unexpected lastmod: %s", result.Sitemaps[0].LastMod)
	}

	// URLs should be nil/empty
	if len(result.URLs) != 0 {
		t.Errorf("expected empty URLs, got %d", len(result.URLs))
	}
}

func TestParseEmpty(t *testing.T) {
	_, err := Parse("")
	if err == nil {
		t.Error("expected error for empty input")
	}
}

func TestParseInvalidXML(t *testing.T) {
	_, err := Parse("<not-a-sitemap><foo></foo></not-a-sitemap>")
	if err == nil {
		t.Error("expected error for non-sitemap XML")
	}
}

func TestValidateURLSet(t *testing.T) {
	err := Validate(testURLSetXML)
	if err != nil {
		t.Errorf("expected valid urlset, got: %s", err.Error())
	}
}

func TestValidateSitemapIndex(t *testing.T) {
	err := Validate(testSitemapIndexXML)
	if err != nil {
		t.Errorf("expected valid sitemapindex, got: %s", err.Error())
	}
}

func TestValidateEmpty(t *testing.T) {
	err := Validate("")
	if err == nil {
		t.Error("expected error for empty input")
	}
}

func TestValidateMissingLoc(t *testing.T) {
	xml := `<?xml version="1.0" encoding="UTF-8"?>
<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">
  <url>
    <lastmod>2025-01-01</lastmod>
  </url>
</urlset>`
	err := Validate(xml)
	if err == nil {
		t.Error("expected error for missing <loc>")
	}
}

func TestValidateInvalidXML(t *testing.T) {
	err := Validate("<urlset><url><loc>test</url></urlset>")
	if err == nil {
		t.Error("expected error for malformed XML")
	}
}
