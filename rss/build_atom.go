package rss

import (
	"encoding/xml"
	"strings"
	"time"
)

// Atom namespace
const atomNS = "http://www.w3.org/2005/Atom"

// buildAtomXML generates an Atom 1.0 XML document from a Feed struct.
// Podcast/iTunes extensions are not included in Atom output (they are RSS-specific).
func buildAtomXML(feed *Feed) (string, error) {
	doc := atomBuildFeed{
		NS: atomNS,
	}

	doc.Title = feed.Title
	if feed.Description != "" {
		doc.Subtitle = feed.Description
	}
	if feed.Language != "" {
		doc.Lang = feed.Language
	}

	// Links
	if feed.Link != "" {
		doc.Links = append(doc.Links, atomBuildLink{
			Href: feed.Link,
			Rel:  "alternate",
			Type: "text/html",
		})
	}

	// Updated
	if feed.Updated != "" {
		doc.Updated = feed.Updated
	} else {
		doc.Updated = time.Now().UTC().Format(time.RFC3339)
	}

	// Entries
	for _, item := range feed.Items {
		entry := atomBuildEntry{
			Title: item.Title,
			ID:    item.GUID,
		}

		if entry.ID == "" {
			entry.ID = item.Link
		}

		if item.Link != "" {
			entry.Links = append(entry.Links, atomBuildLink{
				Href: item.Link,
				Rel:  "alternate",
				Type: "text/html",
			})
		}

		if item.Description != "" {
			entry.Summary = &atomBuildText{
				Type:  "html",
				Value: item.Description,
			}
		}

		if item.Content != "" {
			entry.Content = &atomBuildText{
				Type:  "html",
				Value: item.Content,
			}
		}

		// Authors
		if item.Author != "" {
			// Split on ", " to handle multiple authors
			names := strings.Split(item.Author, ", ")
			for _, name := range names {
				name = strings.TrimSpace(name)
				if name != "" {
					entry.Authors = append(entry.Authors, atomBuildPerson{Name: name})
				}
			}
		}

		entry.Published = item.Published
		entry.Updated = item.Updated
		if entry.Updated == "" {
			entry.Updated = item.Published
		}

		// Categories
		for _, cat := range item.Categories {
			entry.Categories = append(entry.Categories, atomBuildCategory{Term: cat, Label: cat})
		}

		doc.Entries = append(doc.Entries, entry)
	}

	output, err := xml.MarshalIndent(doc, "", "  ")
	if err != nil {
		return "", err
	}

	return xml.Header + string(output), nil
}

// --- Build XML structs for Atom 1.0 output ---

type atomBuildFeed struct {
	XMLName  xml.Name         `xml:"feed"`
	NS       string           `xml:"xmlns,attr"`
	Lang     string           `xml:"xml:lang,attr,omitempty"`
	Title    string           `xml:"title"`
	Subtitle string           `xml:"subtitle,omitempty"`
	Links    []atomBuildLink  `xml:"link"`
	Updated  string           `xml:"updated"`
	Entries  []atomBuildEntry `xml:"entry"`
}

type atomBuildLink struct {
	XMLName xml.Name `xml:"link"`
	Href    string   `xml:"href,attr"`
	Rel     string   `xml:"rel,attr,omitempty"`
	Type    string   `xml:"type,attr,omitempty"`
}

type atomBuildEntry struct {
	Title      string              `xml:"title"`
	Links      []atomBuildLink     `xml:"link"`
	ID         string              `xml:"id"`
	Published  string              `xml:"published,omitempty"`
	Updated    string              `xml:"updated,omitempty"`
	Summary    *atomBuildText      `xml:"summary,omitempty"`
	Content    *atomBuildText      `xml:"content,omitempty"`
	Authors    []atomBuildPerson   `xml:"author,omitempty"`
	Categories []atomBuildCategory `xml:"category,omitempty"`
}

type atomBuildText struct {
	Type  string `xml:"type,attr,omitempty"`
	Value string `xml:",chardata"`
}

type atomBuildPerson struct {
	XMLName xml.Name `xml:"author"`
	Name    string   `xml:"name"`
}

type atomBuildCategory struct {
	XMLName xml.Name `xml:"category"`
	Term    string   `xml:"term,attr"`
	Label   string   `xml:"label,attr,omitempty"`
}
