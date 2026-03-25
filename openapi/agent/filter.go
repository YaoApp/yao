package agent

import (
	"encoding/json"

	"github.com/gin-gonic/gin"
	"github.com/yaoapp/gou/model"
	"github.com/yaoapp/xun/dbal/query"
	agenttypes "github.com/yaoapp/yao/agent/store/types"
	"github.com/yaoapp/yao/openapi/oauth/authorized"
	"github.com/yaoapp/yao/openapi/oauth/types"
)

// AuthFilter applies permission-based filtering to query wheres for assistants
// This function builds where clauses based on the user's authorization constraints
// It supports TeamOnly and OwnerOnly constraints for data access control
//
// Parameters:
//   - c: gin.Context containing authorization information
//   - authInfo: authorized information extracted from the context
//
// Returns:
//   - []model.QueryWhere: array of where clauses to apply to the query
func AuthFilter(c *gin.Context, authInfo *types.AuthorizedInfo) []model.QueryWhere {
	if authInfo == nil {
		return []model.QueryWhere{}
	}

	var wheres []model.QueryWhere
	scope := authInfo.AccessScope()

	// Team only - User can access:
	// 1. Public records (public = true)
	// 2. Records in their team where:
	//    - They created the record (__yao_created_by matches)
	//    - OR the record is shared with team (share = "team")
	if authInfo.Constraints.TeamOnly && authorized.IsTeamMember(c) {
		wheres = append(wheres, model.QueryWhere{
			Wheres: []model.QueryWhere{
				{Column: "public", Value: true, Method: "orwhere"},
				{Wheres: []model.QueryWhere{
					{Column: "__yao_team_id", Value: scope.TeamID},
					{Wheres: []model.QueryWhere{
						{Column: "__yao_created_by", Value: scope.CreatedBy},
						{Column: "share", Value: "team", Method: "orwhere"},
					}},
				}, Method: "orwhere"},
			},
		})
		return wheres
	}

	// Owner only - User can access:
	// 1. Public records (public = true)
	// 2. Records they created where:
	//    - __yao_team_id is null (not team records)
	//    - __yao_created_by matches their user ID
	if authInfo.Constraints.OwnerOnly && authInfo.UserID != "" {
		wheres = append(wheres, model.QueryWhere{
			Wheres: []model.QueryWhere{
				{Column: "public", Value: true, Method: "orwhere"},
				{Wheres: []model.QueryWhere{
					{Column: "__yao_team_id", OP: "null"},
					{Column: "__yao_created_by", Value: scope.CreatedBy},
				}, Method: "orwhere"},
			},
		})
		return wheres
	}

	return wheres
}

// AuthQueryFilter returns a Query function for easy permission filtering
// This is a convenience function that can be directly used with query.Where()
//
// Usage:
//
//	if filter := AuthQueryFilter(c, authInfo); filter != nil {
//	    qb.Where(filter)
//	}
func AuthQueryFilter(c *gin.Context, authInfo *types.AuthorizedInfo) func(query.Query) {
	if authInfo == nil {
		return nil
	}

	scope := authInfo.AccessScope()

	// Team only - User can access:
	// 1. Public records (public = true)
	// 2. Records in their team where:
	//    - They created the record (__yao_created_by matches)
	//    - OR the record is shared with team (share = "team")
	if authInfo.Constraints.TeamOnly && authorized.IsTeamMember(c) {
		return func(qb query.Query) {
			qb.Where(func(qb query.Query) {
				// Public records
				qb.Where("public", true)
			}).OrWhere(func(qb query.Query) {
				// Team records where user is creator or share is team
				qb.Where("__yao_team_id", scope.TeamID).Where(func(qb query.Query) {
					qb.Where("__yao_created_by", scope.CreatedBy).
						OrWhere("share", "team")
				})
			})
		}
	}

	// Owner only - User can access:
	// 1. Public records (public = true)
	// 2. Records they created where:
	//    - __yao_team_id is null (not team records)
	//    - __yao_created_by matches their user ID
	if authInfo.Constraints.OwnerOnly && authInfo.UserID != "" {
		return func(qb query.Query) {
			qb.Where(func(qb query.Query) {
				// Public records
				qb.Where("public", true)
			}).OrWhere(func(qb query.Query) {
				// Owner records (team_id is null and created by user)
				qb.WhereNull("__yao_team_id").
					Where("__yao_created_by", scope.CreatedBy)
			})
		}
	}

	return nil
}

// FilterBuiltInFields filters sensitive fields for built-in assistants in a list
// For built-in assistants, code-level fields (prompts, prompt_presets, workflow, kb, mcp, options, source) should be cleared
func FilterBuiltInFields(assistants []*agenttypes.AssistantModel) {
	if assistants == nil {
		return
	}

	for _, assistant := range assistants {
		FilterBuiltInAssistant(assistant)
	}
}

// FilterBuiltInAssistant filters sensitive fields for a single built-in assistant
// For built-in assistants, code-level fields (prompts, prompt_presets, workflow, kb, mcp, options, source) should be cleared
// This function can be used for both single assistant and list of assistants
func FilterBuiltInAssistant(assistant *agenttypes.AssistantModel) {
	if assistant == nil {
		return
	}

	if assistant.BuiltIn {
		// Clear code-level sensitive fields for built-in assistants
		assistant.Prompts = nil
		assistant.PromptPresets = nil
		assistant.Workflow = nil
		assistant.Sandbox = nil
		assistant.KB = nil
		assistant.MCP = nil
		assistant.Options = nil
		assistant.Source = ""
	}
}

// AssistantToResponse converts an AssistantModel to a response map,
// replacing the sandbox JSON object with a boolean indicating whether sandbox is configured.
// hasSandbox must be captured before FilterBuiltInAssistant clears the Sandbox field.
func AssistantToResponse(assistant *agenttypes.AssistantModel, hasSandbox bool) map[string]interface{} {
	if assistant == nil {
		return nil
	}

	raw, err := json.Marshal(assistant)
	if err != nil {
		return nil
	}

	var result map[string]interface{}
	if err := json.Unmarshal(raw, &result); err != nil {
		return nil
	}

	result["sandbox"] = hasSandbox
	if assistant.ComputerFilter != nil {
		result["computer_filter"] = assistant.ComputerFilter
	}
	return result
}

// AssistantsToResponse converts a slice of AssistantModel to response maps,
// replacing sandbox with a boolean for each assistant.
// Captures sandbox state before filtering, then applies FilterBuiltInAssistant.
func AssistantsToResponse(assistants []*agenttypes.AssistantModel) []map[string]interface{} {
	if assistants == nil {
		return nil
	}

	result := make([]map[string]interface{}, 0, len(assistants))
	for _, a := range assistants {
		hasSandbox := a.Sandbox != nil || a.IsSandbox
		FilterBuiltInAssistant(a)
		result = append(result, AssistantToResponse(a, hasSandbox))
	}
	return result
}
