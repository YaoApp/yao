package search

import (
	"fmt"
	"strings"

	"github.com/yaoapp/yao/agent/search/types"
)

// DefaultCitationPrompt is the default prompt for citation instructions
const DefaultCitationPrompt = `You have access to reference data in <references> tags. Each <ref> has:
- id: Citation identifier (integer)
- type: Data type (web/kb/db)
- weight: Relevance weight (1.0=highest priority, 0.6=lowest)
- source: Origin (user=user-provided, hook=assistant-searched, auto=auto-searched)

Prioritize higher-weight references when answering.

When citing a reference, use this exact HTML format:
<a class="ref" data-ref-id="{id}" data-ref-type="{type}" href="#ref:{id}">[{id}]</a>

Example: According to the product data<a class="ref" data-ref-id="1" data-ref-type="db" href="#ref:1">[1]</a>, the price is $999.`

// BuildReferences converts search results to unified Reference format
func BuildReferences(results []*types.Result) []*types.Reference {
	var refs []*types.Reference
	for _, result := range results {
		if result == nil {
			continue
		}
		for _, item := range result.Items {
			if item == nil {
				continue
			}
			refs = append(refs, &types.Reference{
				ID:      item.CitationID,
				Type:    item.Type,
				Source:  item.Source,
				Weight:  item.Weight,
				Score:   item.Score,
				Title:   item.Title,
				Content: item.Content,
				URL:     item.URL,
			})
		}
	}
	return refs
}

// FormatReferencesXML formats references as XML for LLM context
func FormatReferencesXML(refs []*types.Reference) string {
	if len(refs) == 0 {
		return ""
	}

	var sb strings.Builder
	sb.WriteString("<references>\n")

	for _, ref := range refs {
		if ref == nil {
			continue
		}
		sb.WriteString(fmt.Sprintf(`<ref id="%s" type="%s" weight="%.1f" source="%s">`,
			ref.ID, ref.Type, ref.Weight, ref.Source))
		sb.WriteString("\n")

		if ref.Title != "" {
			sb.WriteString(ref.Title)
			sb.WriteString("\n")
		}
		sb.WriteString(ref.Content)
		if ref.URL != "" {
			sb.WriteString("\nURL: ")
			sb.WriteString(ref.URL)
		}
		sb.WriteString("\n</ref>\n")
	}

	sb.WriteString("</references>")
	return sb.String()
}

// GetCitationPrompt returns the citation instruction prompt
func GetCitationPrompt(cfg *types.CitationConfig) string {
	if cfg == nil {
		return DefaultCitationPrompt
	}
	if cfg.CustomPrompt != "" {
		return cfg.CustomPrompt
	}
	return DefaultCitationPrompt
}

// BuildReferenceContext builds the complete reference context for LLM
func BuildReferenceContext(results []*types.Result, cfg *types.CitationConfig) *types.ReferenceContext {
	refs := BuildReferences(results)
	return &types.ReferenceContext{
		References: refs,
		XML:        FormatReferencesXML(refs),
		Prompt:     GetCitationPrompt(cfg),
	}
}
