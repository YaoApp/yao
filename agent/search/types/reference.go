package types

// Reference is the unified structure for all data sources
// Used to build LLM context from search results
type Reference struct {
	ID      string                 `json:"id"`      // Unique citation ID: "ref_001", "ref_002"
	Type    SearchType             `json:"type"`    // Data type: "web", "kb", "db"
	Source  SourceType             `json:"source"`  // Origin: "user", "hook", "auto"
	Weight  float64                `json:"weight"`  // Relevance weight (1.0=highest, 0.6=lowest)
	Score   float64                `json:"score"`   // Relevance score (0-1)
	Title   string                 `json:"title"`   // Optional title
	Content string                 `json:"content"` // Main content
	URL     string                 `json:"url"`     // Optional URL
	Meta    map[string]interface{} `json:"meta"`    // Additional metadata
}

// ReferenceContext holds the formatted references for LLM input
type ReferenceContext struct {
	References []*Reference `json:"references"` // All references
	XML        string       `json:"xml"`        // Formatted <references> XML
	Prompt     string       `json:"prompt"`     // Citation instruction prompt
}
