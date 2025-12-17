// Package querydsl provides QueryDSL generation from natural language for DB search
// Supports three modes via uses.querydsl configuration:
//   - "builtin" or "": Uses __yao.querydsl system agent (LLM-powered)
//   - "<assistant-id>": Delegate to a custom LLM-powered assistant
//   - "mcp:<server>.<tool>": Call external MCP tool
package querydsl

import (
	"fmt"
	"strings"

	"github.com/yaoapp/gou/query/gou"
	"github.com/yaoapp/yao/agent/context"
	"github.com/yaoapp/yao/agent/search/types"
)

// SystemQueryDSLAgent is the default system agent for QueryDSL generation
const SystemQueryDSLAgent = "__yao.querydsl"

// Generator generates QueryDSL from natural language
// Mode is determined by uses.querydsl configuration
type Generator struct {
	usesQueryDSL string                // "builtin", "<assistant-id>", "mcp:<server>.<tool>"
	config       *types.QueryDSLConfig // QueryDSL generation options
}

// NewGenerator creates a new QueryDSL generator
// usesQueryDSL: value from uses.querydsl config
// cfg: QueryDSL generation options from search config
func NewGenerator(usesQueryDSL string, cfg *types.QueryDSLConfig) *Generator {
	return &Generator{
		usesQueryDSL: usesQueryDSL,
		config:       cfg,
	}
}

// Generate generates QueryDSL from natural language based on configured mode
// Returns a QueryDSL ready for execution
func (g *Generator) Generate(ctx *context.Context, input *Input) (*Result, error) {
	var result *Result
	var err error

	switch {
	case g.usesQueryDSL == "builtin" || g.usesQueryDSL == "":
		// Use system querydsl agent
		result, err = g.agentGenerate(ctx, input, SystemQueryDSLAgent)
	case strings.HasPrefix(g.usesQueryDSL, "mcp:"):
		result, err = g.mcpGenerate(ctx, input)
	default:
		// Assume it's an assistant ID for Agent mode
		result, err = g.agentGenerate(ctx, input, g.usesQueryDSL)
	}

	if err != nil {
		return nil, err
	}

	// Validate generated DSL against allowed fields whitelist
	if result != nil && result.DSL != nil && len(input.AllowedFields) > 0 {
		result = g.validateFields(result, input.AllowedFields)
	}

	return result, nil
}

// agentGenerate delegates to an LLM-powered assistant
// The assistant can understand context and generate semantically correct QueryDSL
func (g *Generator) agentGenerate(ctx *context.Context, input *Input, agentID string) (*Result, error) {
	if ctx == nil {
		return nil, fmt.Errorf("context is required for QueryDSL generation")
	}
	provider := NewAgentProvider(agentID)
	return provider.Generate(ctx, input)
}

// mcpGenerate calls an external MCP tool
// Format: "mcp:<server>.<tool>"
func (g *Generator) mcpGenerate(ctx *context.Context, input *Input) (*Result, error) {
	mcpRef := strings.TrimPrefix(g.usesQueryDSL, "mcp:")
	provider, err := NewMCPProvider(mcpRef)
	if err != nil {
		// Fallback to system agent on invalid MCP format
		return g.agentGenerate(ctx, input, SystemQueryDSLAgent)
	}
	return provider.Generate(ctx, input)
}

// validateFields validates that all fields in the generated DSL are in the allowed list
// If a field is not allowed, it's removed and a warning is added
func (g *Generator) validateFields(result *Result, allowedFields []string) *Result {
	if result.DSL == nil {
		return result
	}

	// Build allowed fields set for fast lookup
	allowed := make(map[string]bool)
	for _, f := range allowedFields {
		allowed[f] = true
	}

	var removedFields []string

	// Validate Select fields
	if len(result.DSL.Select) > 0 {
		validSelect := make([]gou.Expression, 0, len(result.DSL.Select))
		for _, expr := range result.DSL.Select {
			if allowed[expr.Field] {
				validSelect = append(validSelect, expr)
			} else if expr.Field != "" {
				removedFields = append(removedFields, "select:"+expr.Field)
			}
		}
		result.DSL.Select = validSelect
	}

	// Validate Where fields (recursive)
	result.DSL.Wheres = g.validateWheres(result.DSL.Wheres, allowed, &removedFields)

	// Validate Order fields
	if len(result.DSL.Orders) > 0 {
		validOrders := make(gou.Orders, 0, len(result.DSL.Orders))
		for _, order := range result.DSL.Orders {
			if order.Field != nil && allowed[order.Field.Field] {
				validOrders = append(validOrders, order)
			} else if order.Field != nil && order.Field.Field != "" {
				removedFields = append(removedFields, "order:"+order.Field.Field)
			}
		}
		result.DSL.Orders = validOrders
	}

	// Add warnings for removed fields
	if len(removedFields) > 0 {
		warning := "removed fields not in allowed list: " + strings.Join(removedFields, ", ")
		result.Warnings = append(result.Warnings, warning)
	}

	return result
}

// validateWheres recursively validates where conditions
func (g *Generator) validateWheres(wheres []gou.Where, allowed map[string]bool, removedFields *[]string) []gou.Where {
	if len(wheres) == 0 {
		return wheres
	}

	validWheres := make([]gou.Where, 0, len(wheres))
	for _, w := range wheres {
		// Check if the field is allowed
		fieldAllowed := true
		if w.Field != nil && w.Field.Field != "" {
			if !allowed[w.Field.Field] {
				*removedFields = append(*removedFields, "where:"+w.Field.Field)
				fieldAllowed = false
			}
		}

		if fieldAllowed {
			// Recursively validate nested wheres
			if len(w.Wheres) > 0 {
				w.Wheres = g.validateWheres(w.Wheres, allowed, removedFields)
			}
			validWheres = append(validWheres, w)
		}
	}

	return validWheres
}
