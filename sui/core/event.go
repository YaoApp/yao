package core

import (
	"fmt"
	"strings"

	"github.com/PuerkitoBio/goquery"
	jsoniter "github.com/json-iterator/go"
	"golang.org/x/net/html"
)

var eventMatcher = NewAttrPrefixMatcher(`s:on-`)

// BindEvent is a method that binds events to the page.
func (page *Page) BindEvent(ctx *BuildContext, sel *goquery.Selection, cn string, ispage bool) {

	sel.FindMatcher(eventMatcher).Each(func(i int, s *goquery.Selection) {
		if comp, has := s.Attr("is"); has && ctx.isJitComponent(comp) {
			return
		}
		id := fmt.Sprintf("%s-%d", page.namespace, ctx.sequence)
		s.SetAttr("s:event", id)
		ReplaceEventData(s)
		ctx.sequence++
		if ispage {
			s.SetAttr("s:event-cn", "__page")
			return
		}
		s.SetAttr("s:event-cn", cn)
	})
}

// BindEvent is a method that binds events to the component in just-in-time mode.
// This is temporarily used in the JIT mode. It will be refectored in the future.
func (parser *TemplateParser) BindEvent(sel *goquery.Selection, ns string, cn string) {

	hasEvent := false
	sel.FindMatcher(eventMatcher).Each(func(i int, s *goquery.Selection) {
		if _, has := s.Attr("s:event-cn"); has {
			return
		}
		id := fmt.Sprintf("%s-%d-%d", ns, parser.sequence, i+1)
		s.SetAttr("s:event", id)
		ReplaceEventData(s)
		s.SetAttr("s:event-cn", cn)
		parser.sequence++
		hasEvent = true
	})

	if !hasEvent {
		return
	}

	// Bind page event
	compSel := sel.Children().First()
	id := fmt.Sprintf("%s-%d", ns, parser.sequence)
	compSel.SetAttr("s:event-cn", "__page")
	compSel.SetAttr("s:event-jit", id)
}

// GetEventScript the event script
func GetEventScript(sequence int, sel *goquery.Selection, ns string, cn string, prefix string, ispage bool) *ScriptNode {

	if len(sel.Nodes) == 0 {
		return nil
	}

	// Page events
	events := map[string]string{}
	dataUnique := map[string]string{}
	jsonUnique := map[string]string{}
	id := fmt.Sprintf("%s-%d", prefix, sequence)
	for _, attr := range sel.Nodes[0].Attr {

		if strings.HasPrefix(attr.Key, "s:on-") {
			name := strings.TrimPrefix(attr.Key, "s:on-")
			handler := attr.Val
			events[name] = handler
			continue
		}

		if strings.HasPrefix(attr.Key, "s:data-") {
			name := strings.TrimPrefix(attr.Key, "s:data-")
			dataUnique[name] = attr.Val
			sel.SetAttr(fmt.Sprintf("data:%s", name), attr.Val)
			continue
		}

		if strings.HasPrefix(attr.Key, "s:json-") {
			name := strings.TrimPrefix(attr.Key, "s:json-")
			jsonUnique[name] = attr.Val
			sel.SetAttr(fmt.Sprintf("json:%s", name), attr.Val)
			continue
		}
	}

	data := []string{}
	for name := range dataUnique {
		data = append(data, name)
		sel.RemoveAttr(fmt.Sprintf("s:data-%s", name))
	}

	json := []string{}
	for name := range jsonUnique {
		json = append(json, name)
		sel.RemoveAttr(fmt.Sprintf("s:json-%s", name))
	}

	dataRaw, _ := jsoniter.MarshalToString(data)
	jsonRaw, _ := jsoniter.MarshalToString(json)

	source := ""
	for name, handler := range events {
		if ispage {
			source += pageEventInjectScript(id, name, dataRaw, jsonRaw, handler) + "\n"
			sel.SetAttr("s:event-cn", "__page")
		} else {
			source += compEventInjectScript(id, name, cn, dataRaw, jsonRaw, handler) + "\n"
			sel.SetAttr("s:event-cn", cn)
		}
		// sel.RemoveAttr(fmt.Sprintf("s:on-%s", name))
	}

	sel.SetAttr("s:event", id)

	return &ScriptNode{
		Source:    source,
		Namespace: ns,
		Component: cn,
		Attrs:     []html.Attribute{{Key: "event", Val: id}},
	}
}

// ReplaceEventData is a method that replaces the data- and json- attributes.
func ReplaceEventData(sel *goquery.Selection) {
	// Replace the data- and json- attributes
	for _, attr := range sel.Nodes[0].Attr {

		if strings.HasPrefix(attr.Key, "s:data-") {
			name := strings.TrimPrefix(attr.Key, "s:data-")
			sel.SetAttr(fmt.Sprintf("data:%s", name), attr.Val)
			sel.RemoveAttr(fmt.Sprintf("s:data-%s", name))
			continue
		}

		if strings.HasPrefix(attr.Key, "s:json-") {
			name := strings.TrimPrefix(attr.Key, "s:json-")
			sel.SetAttr(fmt.Sprintf("json:%s", name), attr.Val)
			sel.RemoveAttr(fmt.Sprintf("s:json-%s", name))
			continue
		}
	}
}
