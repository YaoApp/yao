package rss

import (
	"encoding/xml"
	"strings"
)

// --- Internal XML mapping structs for Atom 1.0 ---

type atomFeed struct {
	XMLName  xml.Name    `xml:"feed"`
	Title    string      `xml:"title"`
	Subtitle string      `xml:"subtitle"`
	Links    []atomLink  `xml:"link"`
	Updated  string      `xml:"updated"`
	Language string      `xml:"http://www.w3.org/XML/1998/namespace lang,attr"`
	Entries  []atomEntry `xml:"entry"`
}

type atomLink struct {
	Href string `xml:"href,attr"`
	Rel  string `xml:"rel,attr"`
	Type string `xml:"type,attr"`
}

type atomEntry struct {
	Title      string         `xml:"title"`
	Links      []atomLink     `xml:"link"`
	Summary    string         `xml:"summary"`
	Content    atomContent    `xml:"content"`
	Authors    []atomPerson   `xml:"author"`
	Published  string         `xml:"published"`
	Updated    string         `xml:"updated"`
	ID         string         `xml:"id"`
	Categories []atomCategory `xml:"category"`
}

type atomContent struct {
	Type  string `xml:"type,attr"`
	Value string `xml:",chardata"`
}

type atomPerson struct {
	Name  string `xml:"name"`
	Email string `xml:"email"`
}

type atomCategory struct {
	Term  string `xml:"term,attr"`
	Label string `xml:"label,attr"`
}

// parseAtom parses an Atom 1.0 XML document into a Feed struct.
func parseAtom(data []byte) (*Feed, error) {
	var doc atomFeed
	if err := xml.Unmarshal(data, &doc); err != nil {
		return nil, err
	}

	feed := &Feed{
		Format:      "atom1.0",
		Title:       strings.TrimSpace(doc.Title),
		Description: strings.TrimSpace(doc.Subtitle),
		Language:    strings.TrimSpace(doc.Language),
		Updated:     strings.TrimSpace(doc.Updated),
		Items:       make([]FeedItem, 0, len(doc.Entries)),
	}

	// Extract primary link: prefer rel="alternate", fallback to first link
	feed.Link = extractAtomLink(doc.Links)

	for i := range doc.Entries {
		feed.Items = append(feed.Items, convertAtomEntry(&doc.Entries[i]))
	}

	return feed, nil
}

// convertAtomEntry converts an internal atomEntry to a public FeedItem.
func convertAtomEntry(entry *atomEntry) FeedItem {
	fi := FeedItem{
		Title:     strings.TrimSpace(entry.Title),
		Link:      extractAtomLink(entry.Links),
		Published: strings.TrimSpace(entry.Published),
		Updated:   strings.TrimSpace(entry.Updated),
		GUID:      strings.TrimSpace(entry.ID),
	}

	// Summary and content
	fi.Description = strings.TrimSpace(entry.Summary)
	fi.Content = strings.TrimSpace(entry.Content.Value)

	// Author: join multiple authors with ", "
	if len(entry.Authors) > 0 {
		names := make([]string, 0, len(entry.Authors))
		for _, a := range entry.Authors {
			n := strings.TrimSpace(a.Name)
			if n != "" {
				names = append(names, n)
			}
		}
		fi.Author = strings.Join(names, ", ")
	}

	// Categories: prefer label, fallback to term
	if len(entry.Categories) > 0 {
		fi.Categories = make([]string, 0, len(entry.Categories))
		for _, c := range entry.Categories {
			v := strings.TrimSpace(c.Label)
			if v == "" {
				v = strings.TrimSpace(c.Term)
			}
			if v != "" {
				fi.Categories = append(fi.Categories, v)
			}
		}
	}

	return fi
}

// extractAtomLink returns the href of the first "alternate" link,
// or the first link if no alternate is found.
func extractAtomLink(links []atomLink) string {
	var fallback string
	for _, l := range links {
		href := strings.TrimSpace(l.Href)
		if href == "" {
			continue
		}
		if l.Rel == "alternate" || l.Rel == "" {
			return href
		}
		if fallback == "" {
			fallback = href
		}
	}
	return fallback
}
