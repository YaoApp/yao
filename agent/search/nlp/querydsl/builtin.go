package querydsl

import (
	"github.com/yaoapp/gou/model"
	"github.com/yaoapp/gou/query/gou"
)

// BuiltinGenerator implements template-based QueryDSL generation
// This is a placeholder implementation that returns a basic QueryDSL.
//
// TODO: Implement actual template-based generation:
//   - Parse natural language query
//   - Match against model schema
//   - Generate appropriate where clauses
//   - Handle common query patterns (search, filter, sort)
//
// For production use cases requiring high accuracy, use Agent or MCP mode.
type BuiltinGenerator struct{}

// NewBuiltinGenerator creates a new builtin QueryDSL generator
func NewBuiltinGenerator() *BuiltinGenerator {
	return &BuiltinGenerator{}
}

// Generate generates QueryDSL from natural language
// Currently returns a placeholder QueryDSL that searches all searchable fields
func (g *BuiltinGenerator) Generate(input *Input) (*Result, error) {
	if input == nil || input.Query == "" {
		return &Result{
			Warnings: []string{"empty query, returning empty DSL"},
		}, nil
	}

	// Build a basic QueryDSL
	dsl := &gou.QueryDSL{}

	// Set limit
	limit := input.Limit
	if limit <= 0 {
		limit = 20
	}
	dsl.Limit = limit

	// Apply pre-defined wheres if provided
	if len(input.Wheres) > 0 {
		dsl.Wheres = input.Wheres
	}

	// Apply orders if provided
	if len(input.Orders) > 0 {
		dsl.Orders = input.Orders
	}

	// Load models and try to generate basic search conditions
	// Use the first model as the primary table, others can be joined
	if len(input.ModelIDs) > 0 {
		primaryModelID := input.ModelIDs[0]

		// Check if model exists before selecting
		if !model.Exists(primaryModelID) {
			return &Result{
				DSL:     dsl,
				Explain: "Generated basic QueryDSL (model not found)",
				Warnings: []string{
					"model '" + primaryModelID + "' not found, returning basic DSL without search conditions",
				},
			}, nil
		}

		primaryModel := model.Select(primaryModelID)
		if primaryModel != nil && len(primaryModel.MetaData.Columns) > 0 {
			// Find searchable text columns (string/text types with index)
			var searchableColumns []string
			for _, col := range primaryModel.MetaData.Columns {
				// Use Index as a proxy for searchable, and check for text types
				if col.Index && (col.Type == "string" || col.Type == "text" || col.Type == "longText") {
					searchableColumns = append(searchableColumns, col.Name)
				}
			}

			// If we have searchable columns and no pre-defined wheres, add a basic search
			if len(searchableColumns) > 0 && len(input.Wheres) == 0 {
				// Build OR conditions for searchable columns
				orWheres := make([]gou.Where, 0, len(searchableColumns))
				for _, col := range searchableColumns {
					orWheres = append(orWheres, gou.Where{
						Condition: gou.Condition{
							Field: &gou.Expression{Field: col},
							OP:    "match",
							Value: input.Query,
						},
					})
				}

				// Wrap in OR group if multiple columns
				if len(orWheres) > 1 {
					// Mark all but the first as OR conditions
					for i := 1; i < len(orWheres); i++ {
						orWheres[i].OR = true
					}
					dsl.Wheres = []gou.Where{
						{
							Wheres: orWheres,
						},
					}
				} else if len(orWheres) == 1 {
					dsl.Wheres = orWheres
				}
			}
		}

		// TODO: For multi-model queries, generate joins based on model relations
		// This requires analyzing the relations between models and generating
		// appropriate JOIN clauses in the QueryDSL
	}

	return &Result{
		DSL:     dsl,
		Explain: "Generated basic search QueryDSL using builtin template (placeholder implementation)",
		Warnings: []string{
			"builtin generator is a placeholder, consider using Agent or MCP mode for production",
		},
	}, nil
}
