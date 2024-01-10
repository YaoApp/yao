package core

import (
	"bytes"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"golang.org/x/net/html"
)

// NewDocument create a new document
func NewDocument(htmlContent []byte) (*goquery.Document, error) {
	docNode, err := html.Parse(bytes.NewReader(htmlContent))
	if err != nil {
		return nil, err
	}
	return goquery.NewDocumentFromNode(docNode), nil
}

// NewDocumentString create a new document
func NewDocumentString(htmlContent string) (*goquery.Document, error) {
	docNode, err := html.Parse(strings.NewReader(htmlContent))
	if err != nil {
		return nil, err
	}
	return goquery.NewDocumentFromNode(docNode), nil
}
