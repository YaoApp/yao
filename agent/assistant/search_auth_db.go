package assistant

import (
	"github.com/yaoapp/gou/query/gou"
	"github.com/yaoapp/yao/agent/context"
)

// BuildDBAuthWheres builds where clauses for DB search based on authorization
// This applies permission-based filtering to database queries
// Returns gou.Where clauses to filter records by authorization scope
func BuildDBAuthWheres(ctx *context.Context) []gou.Where {
	if ctx == nil || ctx.Authorized == nil {
		return nil
	}

	authInfo := ctx.Authorized

	// No constraints, no filter needed
	if !authInfo.Constraints.TeamOnly && !authInfo.Constraints.OwnerOnly {
		return nil
	}

	var wheres []gou.Where

	// Team only - User can access:
	// 1. Public records (public = true)
	// 2. Records in their team where:
	//    - They created the record (__yao_created_by matches)
	//    - OR the record is shared with team (share = "team")
	if authInfo.Constraints.TeamOnly && authInfo.TeamID != "" {
		wheres = append(wheres, gou.Where{
			Wheres: []gou.Where{
				// Public records
				{Condition: gou.Condition{
					Field: &gou.Expression{Field: "public"},
					Value: true,
					OP:    "=",
					OR:    true,
				}},
				// Team records
				{
					Wheres: []gou.Where{
						{Condition: gou.Condition{
							Field: &gou.Expression{Field: "__yao_team_id"},
							Value: authInfo.TeamID,
							OP:    "=",
						}},
						{Wheres: []gou.Where{
							{Condition: gou.Condition{
								Field: &gou.Expression{Field: "__yao_created_by"},
								Value: authInfo.UserID,
								OP:    "=",
							}},
							{Condition: gou.Condition{
								Field: &gou.Expression{Field: "share"},
								Value: "team",
								OP:    "=",
								OR:    true,
							}},
						}},
					},
				},
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
		wheres = append(wheres, gou.Where{
			Wheres: []gou.Where{
				// Public records
				{Condition: gou.Condition{
					Field: &gou.Expression{Field: "public"},
					Value: true,
					OP:    "=",
					OR:    true,
				}},
				// Owner records
				{
					Wheres: []gou.Where{
						{Condition: gou.Condition{
							Field: &gou.Expression{Field: "__yao_team_id"},
							OP:    "null",
						}},
						{Condition: gou.Condition{
							Field: &gou.Expression{Field: "__yao_created_by"},
							Value: authInfo.UserID,
							OP:    "=",
						}},
					},
				},
			},
		})
		return wheres
	}

	return wheres
}
