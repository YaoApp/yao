package querydsl

import (
	"github.com/yaoapp/gou/query/gou"
	"github.com/yaoapp/yao/agent/search/types"
)

// Input contains all information needed to generate QueryDSL
type Input struct {
	Query         string                 // Natural language query
	ModelIDs      []string               // Target model IDs (e.g., ["user", "order", "product"])
	Scenario      types.ScenarioType     // QueryDSL scenario: "filter", "aggregation", "join", "complex"
	Wheres        []gou.Where            // Pre-defined filters (optional)
	Orders        gou.Orders             // Sort orders (optional)
	AllowedFields []string               // Allowed fields whitelist (optional, for security validation)
	Limit         int                    // Max results
	ExtraParams   map[string]interface{} // Additional parameters
}

// Result represents the result of QueryDSL generation
type Result struct {
	DSL      *gou.QueryDSL `json:"dsl"`                // Generated QueryDSL (supports joins)
	Explain  string        `json:"explain,omitempty"`  // Human-readable explanation
	Warnings []string      `json:"warnings,omitempty"` // Any warnings during generation
}
