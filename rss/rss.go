package rss

import (
	"encoding/xml"
	"strconv"
	"strings"
)

// iTunes namespace URI
const itunesNS = "http://www.itunes.com/dtds/podcast-1.0.dtd"

// Content namespace URI (for content:encoded)
const contentNS = "http://purl.org/rss/1.0/modules/content/"

// --- Internal XML mapping structs for RSS 2.0 ---

type rssDoc struct {
	XMLName xml.Name   `xml:"rss"`
	Channel rssChannel `xml:"channel"`
}

type rssChannel struct {
	Title       string `xml:"title"`
	Link        string `xml:"link"`
	Description string `xml:"description"`
	Language    string `xml:"language"`
	LastBuild   string `xml:"lastBuildDate"`
	PubDate     string `xml:"pubDate"`

	// iTunes namespace (podcast extensions) — channel level
	ItunesAuthor   string           `xml:"http://www.itunes.com/dtds/podcast-1.0.dtd author"`
	ItunesSummary  string           `xml:"http://www.itunes.com/dtds/podcast-1.0.dtd summary"`
	ItunesImage    itunesImage      `xml:"http://www.itunes.com/dtds/podcast-1.0.dtd image"`
	ItunesOwner    itunesOwner      `xml:"http://www.itunes.com/dtds/podcast-1.0.dtd owner"`
	ItunesCategory []itunesCategory `xml:"http://www.itunes.com/dtds/podcast-1.0.dtd category"`
	ItunesExplicit string           `xml:"http://www.itunes.com/dtds/podcast-1.0.dtd explicit"`
	ItunesType     string           `xml:"http://www.itunes.com/dtds/podcast-1.0.dtd type"`

	Items []rssItem `xml:"item"`
}

type rssItem struct {
	Title       string         `xml:"title"`
	Link        string         `xml:"link"`
	Description string         `xml:"description"`
	Content     string         `xml:"http://purl.org/rss/1.0/modules/content/ encoded"`
	Author      string         `xml:"author"`
	DcCreator   string         `xml:"http://purl.org/dc/elements/1.1/ creator"`
	PubDate     string         `xml:"pubDate"`
	GUID        rssGUID        `xml:"guid"`
	Categories  []rssCategory  `xml:"category"`
	Enclosures  []rssEnclosure `xml:"enclosure"`

	// iTunes namespace (podcast extensions) — item level
	ItunesDuration    string      `xml:"http://www.itunes.com/dtds/podcast-1.0.dtd duration"`
	ItunesSeason      string      `xml:"http://www.itunes.com/dtds/podcast-1.0.dtd season"`
	ItunesEpisode     string      `xml:"http://www.itunes.com/dtds/podcast-1.0.dtd episode"`
	ItunesEpisodeType string      `xml:"http://www.itunes.com/dtds/podcast-1.0.dtd episodeType"`
	ItunesExplicit    string      `xml:"http://www.itunes.com/dtds/podcast-1.0.dtd explicit"`
	ItunesImage       itunesImage `xml:"http://www.itunes.com/dtds/podcast-1.0.dtd image"`
	ItunesSummary     string      `xml:"http://www.itunes.com/dtds/podcast-1.0.dtd summary"`
	ItunesAuthor      string      `xml:"http://www.itunes.com/dtds/podcast-1.0.dtd author"`
}

type rssGUID struct {
	Value       string `xml:",chardata"`
	IsPermaLink string `xml:"isPermaLink,attr"`
}

type rssCategory struct {
	Value string `xml:",chardata"`
}

type rssEnclosure struct {
	URL    string `xml:"url,attr"`
	Type   string `xml:"type,attr"`
	Length string `xml:"length,attr"`
}

type itunesImage struct {
	Href string `xml:"href,attr"`
}

type itunesOwner struct {
	Name  string `xml:"http://www.itunes.com/dtds/podcast-1.0.dtd name"`
	Email string `xml:"http://www.itunes.com/dtds/podcast-1.0.dtd email"`
}

type itunesCategory struct {
	Text string           `xml:"text,attr"`
	Sub  []itunesCategory `xml:"http://www.itunes.com/dtds/podcast-1.0.dtd category"`
}

// parseRSS parses an RSS 2.0 XML document into a Feed struct.
func parseRSS(data []byte) (*Feed, error) {
	var doc rssDoc
	if err := xml.Unmarshal(data, &doc); err != nil {
		return nil, err
	}

	ch := doc.Channel
	feed := &Feed{
		Format:      "rss2.0",
		Title:       strings.TrimSpace(ch.Title),
		Link:        strings.TrimSpace(ch.Link),
		Description: strings.TrimSpace(ch.Description),
		Language:    strings.TrimSpace(ch.Language),
		Updated:     firstNonEmpty(ch.LastBuild, ch.PubDate),
		Items:       make([]FeedItem, 0, len(ch.Items)),
	}

	// Build podcast metadata if any iTunes fields are present
	feed.Podcast = buildPodcast(&ch)

	for i := range ch.Items {
		feed.Items = append(feed.Items, convertRSSItem(&ch.Items[i]))
	}

	return feed, nil
}

// convertRSSItem converts an internal rssItem to a public FeedItem.
func convertRSSItem(item *rssItem) FeedItem {
	fi := FeedItem{
		Title:       strings.TrimSpace(item.Title),
		Link:        strings.TrimSpace(item.Link),
		Description: strings.TrimSpace(item.Description),
		Content:     strings.TrimSpace(item.Content),
		Published:   strings.TrimSpace(item.PubDate),
		GUID:        strings.TrimSpace(item.GUID.Value),
	}

	// Author: prefer dc:creator over rss author (which is often an email)
	fi.Author = strings.TrimSpace(item.DcCreator)
	if fi.Author == "" {
		fi.Author = strings.TrimSpace(item.Author)
	}

	// Categories
	if len(item.Categories) > 0 {
		fi.Categories = make([]string, 0, len(item.Categories))
		for _, c := range item.Categories {
			v := strings.TrimSpace(c.Value)
			if v != "" {
				fi.Categories = append(fi.Categories, v)
			}
		}
	}

	// Enclosures
	if len(item.Enclosures) > 0 {
		fi.Enclosures = make([]Enclosure, 0, len(item.Enclosures))
		for _, e := range item.Enclosures {
			if e.URL != "" {
				fi.Enclosures = append(fi.Enclosures, Enclosure{
					URL:    e.URL,
					Type:   e.Type,
					Length: e.Length,
				})
			}
		}
	}

	// Podcast episode metadata
	fi.Episode = buildEpisode(item)

	return fi
}

// buildPodcast constructs Podcast metadata from iTunes namespace fields.
// Returns nil if no iTunes fields are populated.
func buildPodcast(ch *rssChannel) *Podcast {
	hasContent := ch.ItunesAuthor != "" ||
		ch.ItunesSummary != "" ||
		ch.ItunesImage.Href != "" ||
		ch.ItunesExplicit != "" ||
		ch.ItunesType != "" ||
		ch.ItunesOwner.Name != "" ||
		ch.ItunesOwner.Email != "" ||
		len(ch.ItunesCategory) > 0

	if !hasContent {
		return nil
	}

	p := &Podcast{
		Author:   strings.TrimSpace(ch.ItunesAuthor),
		Summary:  strings.TrimSpace(ch.ItunesSummary),
		Image:    strings.TrimSpace(ch.ItunesImage.Href),
		Explicit: isExplicit(ch.ItunesExplicit),
		Type:     strings.TrimSpace(ch.ItunesType),
	}

	// Owner
	if ch.ItunesOwner.Name != "" || ch.ItunesOwner.Email != "" {
		p.Owner = &Owner{
			Name:  strings.TrimSpace(ch.ItunesOwner.Name),
			Email: strings.TrimSpace(ch.ItunesOwner.Email),
		}
	}

	// Categories (flatten nested categories)
	p.Category = flattenCategories(ch.ItunesCategory)

	return p
}

// buildEpisode constructs Episode metadata from iTunes namespace item fields.
// Returns nil if no iTunes episode fields are populated.
func buildEpisode(item *rssItem) *Episode {
	hasContent := item.ItunesDuration != "" ||
		item.ItunesSeason != "" ||
		item.ItunesEpisode != "" ||
		item.ItunesEpisodeType != "" ||
		item.ItunesExplicit != "" ||
		item.ItunesImage.Href != "" ||
		item.ItunesSummary != ""

	if !hasContent {
		return nil
	}

	ep := &Episode{
		Duration: strings.TrimSpace(item.ItunesDuration),
		Type:     strings.TrimSpace(item.ItunesEpisodeType),
		Explicit: isExplicit(item.ItunesExplicit),
		Image:    strings.TrimSpace(item.ItunesImage.Href),
		Summary:  strings.TrimSpace(item.ItunesSummary),
	}

	if s, err := strconv.Atoi(strings.TrimSpace(item.ItunesSeason)); err == nil {
		ep.Season = s
	}
	if n, err := strconv.Atoi(strings.TrimSpace(item.ItunesEpisode)); err == nil {
		ep.Number = n
	}

	return ep
}

// flattenCategories extracts category text values, including nested subcategories.
// Example: <itunes:category text="Technology"><itunes:category text="Podcasting"/></itunes:category>
// produces ["Technology", "Technology > Podcasting"]
func flattenCategories(cats []itunesCategory) []string {
	var result []string
	for _, c := range cats {
		text := strings.TrimSpace(c.Text)
		if text == "" {
			continue
		}
		result = append(result, text)
		for _, sub := range c.Sub {
			subText := strings.TrimSpace(sub.Text)
			if subText != "" {
				result = append(result, text+" > "+subText)
			}
		}
	}
	return result
}

// isExplicit interprets the iTunes explicit flag.
// "yes", "true", "explicit" → true; everything else → false.
func isExplicit(val string) bool {
	v := strings.ToLower(strings.TrimSpace(val))
	return v == "yes" || v == "true" || v == "explicit"
}

// firstNonEmpty returns the first non-empty string from the arguments.
func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		v = strings.TrimSpace(v)
		if v != "" {
			return v
		}
	}
	return ""
}
