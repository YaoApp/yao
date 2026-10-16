package llm

import (
	goumodel "github.com/yaoapp/gou/model"
)

// buildScopeWheres constructs query where clauses for scope filtering.
// Empty scope means no scope filter (list all).
func buildScopeWheres(scope Scope, teamID, userID string) []goumodel.QueryWhere {
	var wheres []goumodel.QueryWhere
	if scope != "" {
		wheres = append(wheres, goumodel.QueryWhere{Column: "scope", Value: string(scope)})
	}
	if teamID != "" {
		wheres = append(wheres, goumodel.QueryWhere{Column: "team_id", Value: teamID})
	}
	if userID != "" {
		wheres = append(wheres, goumodel.QueryWhere{Column: "user_id", Value: userID})
	}
	return wheres
}
