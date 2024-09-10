package core

import (
	"bytes"
	"fmt"
	"hash/fnv"
	"strings"

	"github.com/PuerkitoBio/goquery"
	jsoniter "github.com/json-iterator/go"
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

// NewDocumentStringWithWrapper create a new document with a wrapper
func NewDocumentStringWithWrapper(htmlContent string) (*goquery.Document, error) {
	doc, err := NewDocumentString(htmlContent)
	if err != nil {
		return nil, err
	}

	// Check if the doc has root element add a div wrapper
	nodes := doc.Find("Body *").Nodes
	if len(nodes) == 1 {
		sel := goquery.NewDocumentFromNode(nodes[0])
		if _, has := sel.Attr("is"); has {
			doc, err := NewDocumentString(fmt.Sprintf("<div>\n%s\n</div>", htmlContent))
			if err != nil {
				return nil, err
			}
			return doc, nil
		}
	}
	return doc, nil
}

// Namespace convert the name to namespace
func Namespace(name string, idx int, hash ...bool) string {
	name = strings.ReplaceAll(name, "/", "_")
	name = strings.ReplaceAll(name, "[", "_")
	name = strings.ReplaceAll(name, "]", "_")
	ns := fmt.Sprintf("page_%s_%d", name, idx)
	if len(hash) > 0 && hash[0] {
		h := fnv.New64a()
		h.Write([]byte(ns))
		return fmt.Sprintf("ns_%x", h.Sum64())
	}
	return ns
}

// ComponentName convert the name to component name
func ComponentName(name string, hash ...bool) string {
	name = strings.ReplaceAll(name, "/", "_")
	name = strings.ReplaceAll(name, "[", "_")
	name = strings.ReplaceAll(name, "]", "_")
	cn := fmt.Sprintf("comp_%s", name)
	// Keep the component name | hash will be supported later
	// if len(hash) > 0 && hash[0] {
	// 	h := fnv.New64a()
	// 	h.Write([]byte(cn))
	// 	return fmt.Sprintf("cn_%x", h.Sum64())
	// }
	return cn
}

// TranslationKey convert the name to translation key
func TranslationKey(name string, sequence int) string {
	prefix := TranslationKeyPrefix(name)
	return fmt.Sprintf("%s_%d", prefix, sequence)
}

// TranslationKeyPrefix convert the name to translation key prefix
func TranslationKeyPrefix(name string) string {
	name = strings.ReplaceAll(name, "/", "_")
	name = strings.ReplaceAll(name, "[", "_")
	name = strings.ReplaceAll(name, "]", "_")
	return fmt.Sprintf("trans_%s", name)
}

// ToCamelCase convert the string to camel case
func ToCamelCase(s string, split ...string) string {
	splitter := "-"
	if len(split) > 0 {
		splitter = split[0]
	}

	s = strings.ToLower(s)
	parts := strings.Split(s, splitter)
	for i, part := range parts {
		if i == 0 {
			continue
		}
		parts[i] = strings.ToUpper(part[:1]) + part[1:]
	}

	return strings.Join(parts, "")
}

// ValueJSON parse the value to a json value
func ValueJSON(value string) interface{} {
	var v interface{}
	err := jsoniter.UnmarshalFromString(value, &v)
	if err != nil {
		return fmt.Sprintf("json error: %s", err.Error())
	}
	return v
}

// HasJSON check if the values has json value
func HasJSON(values []StringValue) bool {
	if values == nil {
		return false
	}

	for _, value := range values {
		if value.JSON {
			return true
		}
	}
	return false
}
