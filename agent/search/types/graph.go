package types

// GraphNode represents a related entity from knowledge graph
type GraphNode struct {
	ID          string                 `json:"id"`
	Type        string                 `json:"type"`                  // Entity type
	Name        string                 `json:"name"`                  // Entity name
	Description string                 `json:"description,omitempty"` // Entity description
	Relation    string                 `json:"relation,omitempty"`    // Relationship to query
	Score       float64                `json:"score,omitempty"`       // Relevance score
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}
