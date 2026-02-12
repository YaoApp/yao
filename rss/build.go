package rss

import "fmt"

// Build generates an XML feed document from a Feed struct.
// The format parameter specifies the output format: "rss" (default) or "atom".
// If format is empty, RSS 2.0 is used.
//
// When the Feed contains Podcast metadata, the RSS 2.0 output will include
// iTunes namespace extensions automatically. Atom output ignores Podcast
// extensions as they are RSS-specific.
func Build(feed *Feed, format string) (string, error) {
	if feed == nil {
		return "", fmt.Errorf("feed is nil")
	}

	switch format {
	case "", "rss", "rss2.0":
		return buildRSSXML(feed)
	case "atom", "atom1.0":
		return buildAtomXML(feed)
	default:
		return "", fmt.Errorf("unsupported output format: %q, expected \"rss\" or \"atom\"", format)
	}
}
