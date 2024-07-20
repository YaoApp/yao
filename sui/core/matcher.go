package core

import (
	"regexp"
	"strings"

	"golang.org/x/net/html"
)

// AttrMatcher is a matcher that matches attribute keys
type AttrMatcher struct {
	prefix string
	re     *regexp.Regexp
}

// NewAttrPrefixMatcher creates a new attribute matcher that matches attribute keys with the given prefix
func NewAttrPrefixMatcher(prefix string) *AttrMatcher {
	return &AttrMatcher{prefix: prefix}
}

// NewAttrRegexpMatcher creates a new attribute matcher that matches attribute keys with the given regexp
func NewAttrRegexpMatcher(re *regexp.Regexp) *AttrMatcher {
	return &AttrMatcher{re: re}
}

// Match returns true if the node has an attribute key that matches the matcher
func (m *AttrMatcher) Match(n *html.Node) bool {
	if m.re == nil {
		return m.prefixMatch(n)
	}
	return m.regexpMatch(n)
}

func (m *AttrMatcher) regexpMatch(n *html.Node) bool {
	for _, attr := range n.Attr {
		if m.re.MatchString(attr.Key) {
			return true
		}
	}
	return false
}

func (m *AttrMatcher) prefixMatch(n *html.Node) bool {
	for _, attr := range n.Attr {
		if strings.HasPrefix(attr.Key, m.prefix) {
			return true
		}
	}
	return false
}

// MatchAll returns all the nodes that have an attribute key that matches the matcher
func (m *AttrMatcher) MatchAll(n *html.Node) []*html.Node {
	var nodes []*html.Node
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		if m.Match(c) {
			nodes = append(nodes, c)
		}
		nodes = append(nodes, m.MatchAll(c)...)
	}
	return nodes

}

// Filter returns all the nodes that have an attribute key that matches the matcher
func (m *AttrMatcher) Filter(ns []*html.Node) []*html.Node {
	var nodes []*html.Node
	for _, n := range ns {
		if m.Match(n) {
			nodes = append(nodes, n)
		}
	}
	return nodes
}
