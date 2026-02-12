package sitemap

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"strings"
)

// Parse parses a sitemap XML string and returns a ParseResult.
// It auto-detects whether the input is a <urlset> or <sitemapindex>.
func Parse(data string) (*ParseResult, error) {
	trimmed := strings.TrimSpace(data)
	if trimmed == "" {
		return nil, fmt.Errorf("input is empty")
	}

	format, err := detectFormat([]byte(trimmed))
	if err != nil {
		return nil, err
	}

	switch format {
	case "urlset":
		return parseURLSet([]byte(trimmed))
	case "sitemapindex":
		return parseSitemapIndex([]byte(trimmed))
	default:
		return nil, fmt.Errorf("unknown sitemap format: root element is <%s>, expected <urlset> or <sitemapindex>", format)
	}
}

// Validate checks whether the input string is a valid sitemap XML.
// Returns nil on success, or a descriptive error explaining what is wrong.
func Validate(data string) error {
	trimmed := strings.TrimSpace(data)
	if trimmed == "" {
		return fmt.Errorf("input is empty")
	}

	// Check basic XML validity
	decoder := xml.NewDecoder(bytes.NewReader([]byte(trimmed)))
	for {
		_, err := decoder.Token()
		if err != nil {
			if err.Error() == "EOF" {
				break
			}
			return fmt.Errorf("not valid XML: %s", err.Error())
		}
	}

	// Detect format
	format, err := detectFormat([]byte(trimmed))
	if err != nil {
		return err
	}

	// Full parse to check structure
	switch format {
	case "urlset":
		result, err := parseURLSet([]byte(trimmed))
		if err != nil {
			return err
		}
		// Check that every <url> has a <loc>
		for i, u := range result.URLs {
			if strings.TrimSpace(u.Loc) == "" {
				return fmt.Errorf("urlset <url> at index %d is missing required <loc> element", i)
			}
		}

	case "sitemapindex":
		result, err := parseSitemapIndex([]byte(trimmed))
		if err != nil {
			return err
		}
		// Check that every <sitemap> has a <loc>
		for i, s := range result.Sitemaps {
			if strings.TrimSpace(s.Loc) == "" {
				return fmt.Errorf("sitemapindex <sitemap> at index %d is missing required <loc> element", i)
			}
		}

	default:
		return fmt.Errorf("root element is <%s>, expected <urlset> or <sitemapindex>", format)
	}

	return nil
}

// detectFormat reads the XML to find the root element name.
func detectFormat(data []byte) (string, error) {
	decoder := xml.NewDecoder(bytes.NewReader(data))
	for {
		tok, err := decoder.Token()
		if err != nil {
			return "", fmt.Errorf("failed to detect sitemap format: %s", err.Error())
		}
		if se, ok := tok.(xml.StartElement); ok {
			return se.Name.Local, nil
		}
	}
}

// parseURLSet parses a <urlset> XML document.
func parseURLSet(data []byte) (*ParseResult, error) {
	var urlset xmlURLSet
	if err := xml.Unmarshal(data, &urlset); err != nil {
		return nil, fmt.Errorf("failed to parse urlset: %s", err.Error())
	}
	return &ParseResult{
		Type: "urlset",
		URLs: urlset.URLs,
	}, nil
}

// parseSitemapIndex parses a <sitemapindex> XML document.
func parseSitemapIndex(data []byte) (*ParseResult, error) {
	var idx xmlSitemapIndex
	if err := xml.Unmarshal(data, &idx); err != nil {
		return nil, fmt.Errorf("failed to parse sitemapindex: %s", err.Error())
	}
	return &ParseResult{
		Type:     "sitemapindex",
		Sitemaps: idx.Sitemaps,
	}, nil
}
