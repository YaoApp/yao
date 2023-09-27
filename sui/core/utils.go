package core

import (
	"bytes"
	"strings"

	"github.com/PuerkitoBio/goquery"
)

// NewDocument create a new document
func NewDocument(html []byte) (*goquery.Document, error) {
	return goquery.NewDocumentFromReader(bytes.NewReader(html))
}

// NewDocumentString create a new document
func NewDocumentString(html string) (*goquery.Document, error) {
	return goquery.NewDocumentFromReader(strings.NewReader(html))
}
