package robot

import (
	"github.com/gin-gonic/gin"
	"github.com/yaoapp/yao/openapi/oauth/authorized"
	"github.com/yaoapp/yao/openapi/oauth/types"
)

// Permission check functions for robot access control
//
// Permission Rules:
// 1. No auth info or no constraints: allow all
// 2. OwnerOnly: user can only access resources they created (__yao_created_by == userID)
// 3. TeamOnly: user can access resources in their team (__yao_team_id == teamID)
// 4. For personal users (no team): __yao_team_id should be empty or equal to user_id
//
// Read vs Write:
// - Read: team members can read team resources
// - Write: only creator or team owner can write (update/delete)

// CanRead checks if the user has read permission for a robot
// Read permission is granted if:
// - No auth info (public access)
// - No constraints (admin/system)
// - User is the creator (__yao_created_by == userID)
// - TeamOnly: robot belongs to user's team (__yao_team_id == teamID)
func CanRead(c *gin.Context, authInfo *types.AuthorizedInfo, robotTeamID, robotCreatedBy string) bool {
	// No auth info, allow access (handled by OAuth guard)
	if authInfo == nil {
		return true
	}

	// No constraints, allow access (admin/system user)
	if !authInfo.Constraints.TeamOnly && !authInfo.Constraints.OwnerOnly {
		return true
	}

	// User is the creator - always allow
	if robotCreatedBy != "" && robotCreatedBy == authInfo.UserID {
		return true
	}

	// TeamOnly constraint: check team membership
	if authInfo.Constraints.TeamOnly && authorized.IsTeamMember(c) {
		// Robot belongs to user's team
		if robotTeamID != "" && robotTeamID == authInfo.TeamID {
			return true
		}
	}

	// OwnerOnly constraint: only creator can access (already checked above)
	// If we reach here with OwnerOnly, user is not the creator
	if authInfo.Constraints.OwnerOnly {
		return false
	}

	return false
}

// CanWrite checks if the user has write permission for a robot (update/delete)
// Write permission is more restrictive:
// - No auth info: deny (should not happen, OAuth guard will block)
// - No constraints: allow (admin/system)
// - User is the creator: allow
// - TeamOnly + OwnerOnly: user must be creator AND in the same team
func CanWrite(c *gin.Context, authInfo *types.AuthorizedInfo, robotTeamID, robotCreatedBy string) bool {
	// No auth info, deny write access
	if authInfo == nil {
		return false
	}

	// No constraints, allow access (admin/system user)
	if !authInfo.Constraints.TeamOnly && !authInfo.Constraints.OwnerOnly {
		return true
	}

	// User is the creator - allow write
	if robotCreatedBy != "" && robotCreatedBy == authInfo.UserID {
		// If TeamOnly is also set, verify team membership
		if authInfo.Constraints.TeamOnly {
			if robotTeamID == "" || robotTeamID == authInfo.TeamID {
				return true
			}
			return false
		}
		return true
	}

	// Not the creator - deny write access
	// (In the future, we could add team admin/owner check here)
	return false
}

// GetEffectiveTeamID returns the effective team_id for a robot
// For personal users (no team selected), returns user_id as team_id
// For team users, returns the selected team_id
func GetEffectiveTeamID(authInfo *types.AuthorizedInfo) string {
	if authInfo == nil {
		return ""
	}

	// If user has a team selected, use it
	if authInfo.TeamID != "" {
		return authInfo.TeamID
	}

	// For personal users, use user_id as team_id
	// This ensures resources are scoped to the individual user
	return authInfo.UserID
}

// BuildListFilter builds filter conditions for listing robots based on permissions
// Returns teamID filter to apply to the query
func BuildListFilter(c *gin.Context, authInfo *types.AuthorizedInfo, requestedTeamID string) string {
	if authInfo == nil {
		return requestedTeamID
	}

	// No constraints - use requested filter or no filter
	if !authInfo.Constraints.TeamOnly && !authInfo.Constraints.OwnerOnly {
		return requestedTeamID
	}

	// TeamOnly constraint: force filter to user's team
	if authInfo.Constraints.TeamOnly && authorized.IsTeamMember(c) {
		return authInfo.TeamID
	}

	// OwnerOnly constraint: filter by user_id as team_id (personal resources)
	if authInfo.Constraints.OwnerOnly {
		return authInfo.UserID
	}

	return requestedTeamID
}
