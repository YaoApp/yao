package job

import (
	"github.com/gin-gonic/gin"
	"github.com/yaoapp/gou/model"
	"github.com/yaoapp/yao/job"
	"github.com/yaoapp/yao/openapi/oauth/authorized"
	"github.com/yaoapp/yao/openapi/oauth/types"
)

// AuthFilter applies permission-based filtering to query wheres
// This function builds where clauses based on the user's authorization constraints
// It supports TeamOnly and OwnerOnly constraints for data access control
//
// Note: Unlike the kb module, job doesn't have 'public' and 'share' fields,
// so the filtering is simpler and based only on __yao_team_id and __yao_created_by
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
	// 1. Records in their team where __yao_team_id matches
	if authInfo.Constraints.TeamOnly && authorized.IsTeamMember(c) {
		wheres = append(wheres, model.QueryWhere{
			Column: "__yao_team_id",
			Value:  scope.TeamID,
		})
		return wheres
	}

	// Owner only - User can access:
	// 1. Records they created where:
	//    - __yao_team_id is null (not team records)
	//    - __yao_created_by matches their user ID
	if authInfo.Constraints.OwnerOnly && authInfo.UserID != "" {
		wheres = append(wheres, model.QueryWhere{
			Wheres: []model.QueryWhere{
				{Column: "__yao_team_id", OP: "null"},
				{Column: "__yao_created_by", Value: scope.CreatedBy},
			},
		})
		return wheres
	}

	return wheres
}

// HasJobAccess checks if the current user has access to a specific job
// This is useful for checking access to job-related resources like executions and logs
//
// Parameters:
//   - c: gin.Context containing authorization information
//   - authInfo: authorized information extracted from the context
//   - jobInstance: the job instance to check access for
//
// Returns:
//   - bool: true if the user has access to the job, false otherwise
func HasJobAccess(c *gin.Context, authInfo *types.AuthorizedInfo, jobInstance *job.Job) bool {
	if authInfo == nil {
		// No auth info means public access (or no auth required)
		return true
	}

	scope := authInfo.AccessScope()

	// Team only - Check if job belongs to user's team
	if authInfo.Constraints.TeamOnly && authorized.IsTeamMember(c) {
		return jobInstance.YaoTeamID == scope.TeamID
	}

	// Owner only - Check if job was created by user and not in a team
	if authInfo.Constraints.OwnerOnly && authInfo.UserID != "" {
		return jobInstance.YaoTeamID == "" && jobInstance.YaoCreatedBy == scope.CreatedBy
	}

	// No constraints means access is allowed
	return true
}
