package core

import (
	"bytes"
	"fmt"
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

// Namespace convert the name to namespace
func Namespace(name string, idx int) string {
	name = strings.ReplaceAll(name, "/", "_")
	name = strings.ReplaceAll(name, "[", "_")
	name = strings.ReplaceAll(name, "]", "_")
	namespace := fmt.Sprintf("__page_%s_%d", name, idx)
	return namespace
}
