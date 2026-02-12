package rss

import (
	"encoding/xml"
	"strings"
	"time"
)

// buildRSSXML generates an RSS 2.0 XML document from a Feed struct.
// If the Feed contains Podcast metadata, iTunes namespace extensions are included.
func buildRSSXML(feed *Feed) (string, error) {
	doc := rssBuildDoc{
		Version: "2.0",
	}

	// Add namespace declarations based on content
	doc.ContentNS = contentNS
	hasPodcast := feed.Podcast != nil
	if hasPodcast {
		doc.ItunesNS = itunesNS
	}

	ch := &doc.Channel
	ch.Title = feed.Title
	ch.Link = feed.Link
	ch.Description = feed.Description
	ch.Language = feed.Language

	if feed.Updated != "" {
		ch.LastBuild = feed.Updated
	} else {
		ch.LastBuild = time.Now().UTC().Format(time.RFC1123Z)
	}

	// Podcast channel-level metadata
	if hasPodcast {
		p := feed.Podcast
		ch.ItunesAuthor = p.Author
		ch.ItunesSummary = p.Summary
		if p.Image != "" {
			ch.ItunesImage = &rssBuildItunesImage{Href: p.Image}
		}
		if p.Owner != nil {
			ch.ItunesOwner = &rssBuildItunesOwner{
				Name:  p.Owner.Name,
				Email: p.Owner.Email,
			}
		}
		for _, cat := range p.Category {
			// Handle "Parent > Child" format
			parts := strings.SplitN(cat, " > ", 2)
			if len(parts) == 2 {
				// This is a subcategory — skip, it will be included under its parent
				continue
			}
			bc := rssBuildItunesCategory{Text: cat}
			// Check if there are subcategories
			for _, sub := range p.Category {
				subParts := strings.SplitN(sub, " > ", 2)
				if len(subParts) == 2 && subParts[0] == cat {
					bc.Sub = append(bc.Sub, rssBuildItunesCategory{Text: subParts[1]})
				}
			}
			ch.ItunesCategory = append(ch.ItunesCategory, bc)
		}
		if p.Explicit {
			ch.ItunesExplicit = "yes"
		} else if p.Author != "" || p.Summary != "" {
			// Only include explicit=no if podcast metadata is present
			ch.ItunesExplicit = "no"
		}
		ch.ItunesType = p.Type
	}

	// Items
	for _, item := range feed.Items {
		ri := rssBuildItem{
			Title:       item.Title,
			Link:        item.Link,
			Description: item.Description,
			Author:      item.Author,
			PubDate:     item.Published,
			GUID:        item.GUID,
		}

		if item.Content != "" {
			ri.Content = &rssBuildCDATA{Value: item.Content}
		}

		for _, cat := range item.Categories {
			ri.Categories = append(ri.Categories, cat)
		}

		for _, enc := range item.Enclosures {
			ri.Enclosures = append(ri.Enclosures, rssBuildEnclosure{
				URL:    enc.URL,
				Type:   enc.Type,
				Length: enc.Length,
			})
		}

		// Podcast episode metadata
		if item.Episode != nil {
			ep := item.Episode
			ri.ItunesDuration = ep.Duration
			if ep.Season > 0 {
				ri.ItunesSeason = intToStr(ep.Season)
			}
			if ep.Number > 0 {
				ri.ItunesEpisode = intToStr(ep.Number)
			}
			ri.ItunesEpisodeType = ep.Type
			if ep.Explicit {
				ri.ItunesExplicit = "yes"
			} else if ep.Duration != "" {
				ri.ItunesExplicit = "no"
			}
			if ep.Image != "" {
				ri.ItunesImage = &rssBuildItunesImage{Href: ep.Image}
			}
			ri.ItunesSummary = ep.Summary
		}

		ch.Items = append(ch.Items, ri)
	}

	output, err := xml.MarshalIndent(doc, "", "  ")
	if err != nil {
		return "", err
	}

	return xml.Header + string(output), nil
}

// intToStr converts an int to string without importing strconv (to avoid duplication).
func intToStr(n int) string {
	if n == 0 {
		return ""
	}
	// Simple int to string for small positive numbers
	s := ""
	for n > 0 {
		s = string(rune('0'+n%10)) + s
		n /= 10
	}
	return s
}

// --- Build XML structs for RSS 2.0 output ---

type rssBuildDoc struct {
	XMLName   xml.Name        `xml:"rss"`
	Version   string          `xml:"version,attr"`
	ContentNS string          `xml:"xmlns:content,attr,omitempty"`
	ItunesNS  string          `xml:"xmlns:itunes,attr,omitempty"`
	Channel   rssBuildChannel `xml:"channel"`
}

type rssBuildChannel struct {
	Title       string `xml:"title"`
	Link        string `xml:"link"`
	Description string `xml:"description"`
	Language    string `xml:"language,omitempty"`
	LastBuild   string `xml:"lastBuildDate,omitempty"`

	// iTunes namespace
	ItunesAuthor   string                   `xml:"itunes:author,omitempty"`
	ItunesSummary  string                   `xml:"itunes:summary,omitempty"`
	ItunesImage    *rssBuildItunesImage     `xml:"itunes:image,omitempty"`
	ItunesOwner    *rssBuildItunesOwner     `xml:"itunes:owner,omitempty"`
	ItunesCategory []rssBuildItunesCategory `xml:"itunes:category,omitempty"`
	ItunesExplicit string                   `xml:"itunes:explicit,omitempty"`
	ItunesType     string                   `xml:"itunes:type,omitempty"`

	Items []rssBuildItem `xml:"item"`
}

type rssBuildItem struct {
	Title       string              `xml:"title"`
	Link        string              `xml:"link,omitempty"`
	Description string              `xml:"description,omitempty"`
	Content     *rssBuildCDATA      `xml:"content:encoded,omitempty"`
	Author      string              `xml:"author,omitempty"`
	PubDate     string              `xml:"pubDate,omitempty"`
	GUID        string              `xml:"guid,omitempty"`
	Categories  []string            `xml:"category,omitempty"`
	Enclosures  []rssBuildEnclosure `xml:"enclosure,omitempty"`

	// iTunes namespace
	ItunesDuration    string               `xml:"itunes:duration,omitempty"`
	ItunesSeason      string               `xml:"itunes:season,omitempty"`
	ItunesEpisode     string               `xml:"itunes:episode,omitempty"`
	ItunesEpisodeType string               `xml:"itunes:episodeType,omitempty"`
	ItunesExplicit    string               `xml:"itunes:explicit,omitempty"`
	ItunesImage       *rssBuildItunesImage `xml:"itunes:image,omitempty"`
	ItunesSummary     string               `xml:"itunes:summary,omitempty"`
}

type rssBuildCDATA struct {
	Value string `xml:",cdata"`
}

type rssBuildEnclosure struct {
	XMLName xml.Name `xml:"enclosure"`
	URL     string   `xml:"url,attr"`
	Type    string   `xml:"type,attr,omitempty"`
	Length  string   `xml:"length,attr,omitempty"`
}

type rssBuildItunesImage struct {
	XMLName xml.Name `xml:"itunes:image"`
	Href    string   `xml:"href,attr"`
}

type rssBuildItunesOwner struct {
	XMLName xml.Name `xml:"itunes:owner"`
	Name    string   `xml:"itunes:name,omitempty"`
	Email   string   `xml:"itunes:email,omitempty"`
}

type rssBuildItunesCategory struct {
	XMLName xml.Name                 `xml:"itunes:category"`
	Text    string                   `xml:"text,attr"`
	Sub     []rssBuildItunesCategory `xml:"itunes:category,omitempty"`
}
