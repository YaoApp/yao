package rss

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"io"
	"strings"
)

// Parse parses an RSS 2.0 or Atom 1.0 XML string into a unified Feed struct.
// It auto-detects the feed format by inspecting the root XML element.
func Parse(data string) (*Feed, error) {
	trimmed := strings.TrimSpace(data)
	if trimmed == "" {
		return nil, fmt.Errorf("empty input")
	}

	format, err := detectFormat([]byte(trimmed))
	if err != nil {
		return nil, err
	}

	b := []byte(trimmed)
	switch format {
	case "rss":
		return parseRSS(b)
	case "atom":
		return parseAtom(b)
	default:
		return nil, fmt.Errorf("unsupported feed format: %s", format)
	}
}

// Validate checks whether the input string is a valid RSS 2.0 or Atom 1.0 feed.
// Returns nil on success, or a descriptive error explaining what is wrong.
//
// Process convention:
//   - success → true (bool)
//   - failure → error description string
func Validate(data string) error {
	trimmed := strings.TrimSpace(data)
	if trimmed == "" {
		return fmt.Errorf("empty input: expected an XML document containing an RSS or Atom feed")
	}

	// Step 1: check if it is valid XML at all
	decoder := xml.NewDecoder(bytes.NewReader([]byte(trimmed)))
	var rootFound bool
	for {
		tok, err := decoder.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("not valid XML: %s", err.Error())
		}
		if _, ok := tok.(xml.StartElement); ok {
			rootFound = true
			break
		}
	}
	if !rootFound {
		return fmt.Errorf("not valid XML: no root element found")
	}

	// Step 2: detect format
	format, err := detectFormat([]byte(trimmed))
	if err != nil {
		return err
	}

	// Step 3: attempt a full parse to verify structural integrity
	b := []byte(trimmed)
	switch format {
	case "rss":
		feed, err := parseRSS(b)
		if err != nil {
			return fmt.Errorf("RSS 2.0 parse error: %s", err.Error())
		}
		if feed.Title == "" {
			return fmt.Errorf("RSS 2.0 feed is missing required <title> element in <channel>")
		}
	case "atom":
		feed, err := parseAtom(b)
		if err != nil {
			return fmt.Errorf("Atom 1.0 parse error: %s", err.Error())
		}
		if feed.Title == "" {
			return fmt.Errorf("Atom feed is missing required <title> element")
		}
	default:
		return fmt.Errorf("unsupported feed format: %s", format)
	}

	return nil
}

// detectFormat inspects the root XML element to determine the feed format.
// Returns "rss" for RSS 2.0, "atom" for Atom 1.0, or an error.
func detectFormat(data []byte) (string, error) {
	decoder := xml.NewDecoder(bytes.NewReader(data))
	for {
		tok, err := decoder.Token()
		if err == io.EOF {
			return "", fmt.Errorf("not valid XML: unexpected end of document before root element")
		}
		if err != nil {
			return "", fmt.Errorf("not valid XML: %s", err.Error())
		}

		se, ok := tok.(xml.StartElement)
		if !ok {
			continue
		}

		local := strings.ToLower(se.Name.Local)

		switch {
		case local == "rss":
			return "rss", nil

		case local == "rdf":
			// RDF-based RSS 1.0 (root element is <rdf:RDF>)
			// We treat it as RSS for parsing purposes
			return "rss", nil

		case local == "feed":
			return "atom", nil

		default:
			return "", fmt.Errorf(
				"unrecognized feed format: root element is <%s>, expected <rss>, <feed>, or <rdf:RDF>",
				se.Name.Local,
			)
		}
	}
}
