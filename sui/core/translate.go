package core

import (
	"fmt"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"golang.org/x/net/html"
)

// TranslateDocument translates the document
func (page *Page) TranslateDocument(doc *goquery.Document) error {

	if doc.Length() == 0 {
		return nil
	}

	if page.transCtx == nil {
		return fmt.Errorf("TranslateMarks: context is nil")
	}

	if page.transCtx.translations == nil {
		page.transCtx.translations = []Translation{}
	}

	root := doc.First()
	return page.TranslateSelection(root)
}

// TranslateSelection translates the selection
func (page *Page) TranslateSelection(sel *goquery.Selection) error {

	if sel.Length() == 0 {
		return nil
	}

	if page.transCtx == nil {
		return fmt.Errorf("TranslateMarks: context is nil")
	}

	if page.transCtx.translations == nil {
		page.transCtx.translations = []Translation{}
	}

	translations, err := page.translateNode(sel.Nodes[0])
	if err != nil {
		return err
	}

	if translations != nil {
		page.transCtx.translations = append(page.transCtx.translations, translations...)
	}

	return nil

}

func (page *Page) translateNode(node *html.Node) ([]Translation, error) {

	translations := []Translation{}

	switch node.Type {
	case html.DocumentNode:
		for child := node.FirstChild; child != nil; child = child.NextSibling {
			trans, err := page.translateNode(child)
			if err != nil {
				return nil, err
			}
			translations = append(translations, trans...)
		}
		break

	case html.ElementNode:

		sel := goquery.NewDocumentFromNode(node)
		// Script
		if node.Data == "script" {
			if _, has := sel.Attr("s:trans-script"); has {
				break
			}
			code := goquery.NewDocumentFromNode(node).Text()
			trans, keys, err := page.translateScript(code)
			if err != nil {
				return nil, err
			}
			if len(keys) > 0 {
				raw := strings.Join(keys, ",")
				sel.SetAttr("s:trans-script", raw)
				translations = append(translations, trans...)
			}
			break
		}

		for _, attr := range node.Attr {

			if _, has := sel.Attr("s:trans-attr-" + attr.Key); has {
				continue
			}

			trans, keys, err := page.translateText(attr.Val, "attr")
			if err != nil {
				return nil, err
			}
			if len(keys) > 0 {
				raw := strings.Join(keys, ",")
				sel.SetAttr("s:trans-attr-"+attr.Key, raw)
				translations = append(translations, trans...)
			}

		}

		// Node Attributes
		for child := node.FirstChild; child != nil; child = child.NextSibling {
			trans, err := page.translateNode(child)
			if err != nil {
				return nil, err
			}
			translations = append(translations, trans...)
		}
		break

	case html.TextNode:
		parentSel := goquery.NewDocumentFromNode(node.Parent)
		if _, has := parentSel.Attr("s:trans"); has {
			if _, has := parentSel.Attr("s:trans-node"); has {
				break
			}

			key := TranslationKey(page.Route, page.transCtx.sequence)
			message := strings.TrimSpace(node.Data)
			if message != "" {
				translations = append(translations, Translation{
					Key:     key,
					Message: message,
					Type:    "text",
				})
				parentSel.SetAttr("s:trans-node", key)
				page.transCtx.sequence = page.transCtx.sequence + 1
			}
			parentSel.SetAttr("s:trans-escape", "true")
		}

		if _, has := parentSel.Attr("s:trans-text"); has {
			break
		}
		trans, keys, err := page.translateText(node.Data, "text")
		if err != nil {
			return nil, err
		}
		if len(keys) > 0 {
			raw := strings.Join(keys, ",")
			parentSel.SetAttr("s:trans-text", raw)
			translations = append(translations, trans...)
		}
		break
	}

	return translations, nil
}

func (page *Page) translateText(text string, transType string) ([]Translation, []string, error) {
	translations := []Translation{}
	matches := dataTokens.FindAllStringSubmatch(text, -1)
	keys := []string{}
	for _, match := range matches {
		text := strings.TrimSpace(match[1])
		transMatches := transStmtReSingle.FindAllStringSubmatch(text, -1)
		if len(transMatches) == 0 {
			transMatches = transStmtReDouble.FindAllStringSubmatch(text, -1)
		}
		for _, transMatch := range transMatches {
			message := strings.TrimSpace(transMatch[1])
			key := TranslationKey(page.Route, page.transCtx.sequence)
			keys = append(keys, key)
			translations = append(translations, Translation{
				Key:     key,
				Message: message,
				Type:    transType,
			})
			page.transCtx.sequence = page.transCtx.sequence + 1
		}
	}
	return translations, keys, nil
}

func (page *Page) translateScript(code string) ([]Translation, []string, error) {

	translations := []Translation{}
	keys := []string{}
	if code == "" {
		return translations, keys, nil
	}
	matches := transFuncRe.FindAllStringSubmatch(code, -1)
	for _, match := range matches {
		key := TranslationKey(page.Route, page.transCtx.sequence)
		translations = append(translations, Translation{
			Key:     key,
			Message: match[1],
			Type:    "script",
		})
		page.transCtx.sequence = page.transCtx.sequence + 1
		keys = append(keys, key)
	}
	return translations, keys, nil
}
