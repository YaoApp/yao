package user

import (
	"context"
	"fmt"
	"time"

	"github.com/yaoapp/gou/model"
	"github.com/yaoapp/kun/maps"
)

// Team Resource

// GetTeam retrieves team information by team_id
func (u *DefaultUser) GetTeam(ctx context.Context, teamID string) (maps.MapStrAny, error) {
	m := model.Select(u.teamModel)
	teams, err := m.Get(model.QueryParam{
		Select: u.teamFields,
		Wheres: []model.QueryWhere{
			{Column: "team_id", Value: teamID},
		},
		Limit: 1,
	})

	if err != nil {
		return nil, fmt.Errorf(ErrFailedToGetTeam, err)
	}

	if len(teams) == 0 {
		return nil, fmt.Errorf(ErrTeamNotFound)
	}

	return teams[0], nil
}

// GetTeamDetail retrieves detailed team information by team_id
func (u *DefaultUser) GetTeamDetail(ctx context.Context, teamID string) (maps.MapStrAny, error) {
	m := model.Select(u.teamModel)
	teams, err := m.Get(model.QueryParam{
		Select: u.teamDetailFields,
		Wheres: []model.QueryWhere{
			{Column: "team_id", Value: teamID},
		},
		Limit: 1,
	})

	if err != nil {
		return nil, fmt.Errorf(ErrFailedToGetTeam, err)
	}

	if len(teams) == 0 {
		return nil, fmt.Errorf(ErrTeamNotFound)
	}

	return teams[0], nil
}

// TeamExists checks if a team exists by team_id (lightweight query)
func (u *DefaultUser) TeamExists(ctx context.Context, teamID string) (bool, error) {
	m := model.Select(u.teamModel)
	teams, err := m.Get(model.QueryParam{
		Select: []interface{}{"id"}, // Only select ID for existence check
		Wheres: []model.QueryWhere{
			{Column: "team_id", Value: teamID},
		},
		Limit: 1, // Only need to know if at least one exists
	})

	if err != nil {
		return false, fmt.Errorf(ErrFailedToGetTeam, err)
	}

	return len(teams) > 0, nil
}

// CreateTeam creates a new team
func (u *DefaultUser) CreateTeam(ctx context.Context, teamData maps.MapStrAny) (string, error) {
	// Generate team_id if not provided
	if _, exists := teamData["team_id"]; !exists {
		teamID, err := u.GenerateUserID(ctx, true) // Reuse user ID generation logic for team ID
		if err != nil {
			return "", fmt.Errorf("failed to generate team_id: %w", err)
		}
		teamData["team_id"] = teamID
		teamData["__yao_team_id"] = teamID // Add __yao_team_id to the team data
	}

	// Validate required fields
	if _, exists := teamData["name"]; !exists {
		return "", fmt.Errorf("name is required in teamData")
	}
	if _, exists := teamData["owner_id"]; !exists {
		return "", fmt.Errorf("owner_id is required in teamData")
	}

	// Set default values if not provided
	if _, exists := teamData["status"]; !exists {
		teamData["status"] = "pending"
	}
	if _, exists := teamData["is_verified"]; !exists {
		teamData["is_verified"] = false
	}

	m := model.Select(u.teamModel)
	id, err := m.Create(teamData)
	if err != nil {
		return "", fmt.Errorf(ErrFailedToCreateTeam, err)
	}

	// Return the team_id as string (preferred approach)
	if teamID, ok := teamData["team_id"].(string); ok {
		return teamID, nil
	}

	// Fallback: convert the returned int id to string
	return fmt.Sprintf("%d", id), nil
}

// UpdateTeam updates an existing team
func (u *DefaultUser) UpdateTeam(ctx context.Context, teamID string, teamData maps.MapStrAny) error {
	// Remove sensitive fields that should not be updated directly
	sensitiveFields := []string{"id", "team_id", "created_at", "verified_at", "verified_by"}
	for _, field := range sensitiveFields {
		delete(teamData, field)
	}

	// Skip update if no valid fields remain
	if len(teamData) == 0 {
		return nil
	}

	m := model.Select(u.teamModel)
	affected, err := m.UpdateWhere(model.QueryParam{
		Wheres: []model.QueryWhere{
			{Column: "team_id", Value: teamID},
		},
		Limit: 1, // Safety: ensure only one record is updated
	}, teamData)

	if err != nil {
		return fmt.Errorf(ErrFailedToUpdateTeam, err)
	}

	if affected == 0 {
		return fmt.Errorf(ErrTeamNotFound)
	}

	return nil
}

// DeleteTeam soft deletes a team
func (u *DefaultUser) DeleteTeam(ctx context.Context, teamID string) error {
	// First check if team exists
	m := model.Select(u.teamModel)
	teams, err := m.Get(model.QueryParam{
		Select: []interface{}{"id", "team_id"},
		Wheres: []model.QueryWhere{
			{Column: "team_id", Value: teamID},
		},
		Limit: 1,
	})

	if err != nil {
		return fmt.Errorf(ErrFailedToGetTeam, err)
	}

	if len(teams) == 0 {
		return fmt.Errorf(ErrTeamNotFound)
	}

	// Proceed with soft delete
	affected, err := m.DeleteWhere(model.QueryParam{
		Wheres: []model.QueryWhere{
			{Column: "team_id", Value: teamID},
		},
		Limit: 1, // Safety: ensure only one record is deleted
	})

	if err != nil {
		return fmt.Errorf(ErrFailedToDeleteTeam, err)
	}

	if affected == 0 {
		return fmt.Errorf(ErrTeamNotFound)
	}

	return nil
}

// GetTeams retrieves teams by query parameters
func (u *DefaultUser) GetTeams(ctx context.Context, param model.QueryParam) ([]maps.MapStr, error) {
	// Set default select fields if not provided
	if param.Select == nil {
		param.Select = u.teamFields
	}

	m := model.Select(u.teamModel)
	teams, err := m.Get(param)
	if err != nil {
		return nil, fmt.Errorf(ErrFailedToGetTeam, err)
	}

	return teams, nil
}

// PaginateTeams retrieves paginated list of teams
func (u *DefaultUser) PaginateTeams(ctx context.Context, param model.QueryParam, page int, pagesize int) (maps.MapStr, error) {
	// Set default select fields if not provided
	if param.Select == nil {
		param.Select = u.teamFields
	}

	m := model.Select(u.teamModel)
	result, err := m.Paginate(param, page, pagesize)
	if err != nil {
		return nil, fmt.Errorf(ErrFailedToGetTeam, err)
	}

	return result, nil
}

// CountTeams returns total count of teams with optional filters
func (u *DefaultUser) CountTeams(ctx context.Context, param model.QueryParam) (int64, error) {
	// Use Paginate with a small page size to get the total count
	// This is more reliable than manual COUNT(*) queries
	m := model.Select(u.teamModel)
	result, err := m.Paginate(param, 1, 1) // Get first page with 1 item to get total
	if err != nil {
		return 0, fmt.Errorf(ErrFailedToGetTeam, err)
	}

	// Extract total from pagination result using utility function
	if totalInterface, ok := result["total"]; ok {
		return parseIntFromDB(totalInterface)
	}

	return 0, fmt.Errorf("total not found in pagination result")
}

// GetTeamsByOwner retrieves teams owned by a specific user
func (u *DefaultUser) GetTeamsByOwner(ctx context.Context, ownerID string) ([]maps.MapStr, error) {
	param := model.QueryParam{
		Select: u.teamFields,
		Wheres: []model.QueryWhere{
			{Column: "owner_id", Value: ownerID},
		},
		Orders: []model.QueryOrder{
			{Column: "created_at", Option: "desc"},
		},
	}

	return u.GetTeams(ctx, param)
}

// GetTeamsByMember retrieves teams by member_id (includes role information and owner status)
func (u *DefaultUser) GetTeamsByMember(ctx context.Context, memberID string) ([]maps.MapStr, error) {

	// Query member records to get team_id and role_id
	param := model.QueryParam{
		Select: []interface{}{"team_id", "user_id", "member_type", "role_id"},
		Wheres: []model.QueryWhere{
			{Column: "user_id", Value: memberID},
			{Column: "member_type", Value: "user"},
			{Column: "status", Value: "active"},
		},
	}

	m := model.Select(u.memberModel)
	members, err := m.Get(param)
	if err != nil {
		return nil, fmt.Errorf(ErrFailedToGetTeam, err)
	}

	if len(members) == 0 {
		return []maps.MapStr{}, nil
	}

	// Build team_id to role_id mapping
	teamRoleMap := make(map[string]string)
	teamIDs := []string{}
	for _, member := range members {
		teamID := member["team_id"].(string)
		roleID := ""
		if role, ok := member["role_id"]; ok && role != nil {
			roleID = fmt.Sprintf("%v", role)
		}
		teamRoleMap[teamID] = roleID
		teamIDs = append(teamIDs, teamID)
	}

	// Get teams
	teams, err := u.GetTeams(ctx, model.QueryParam{
		Select: u.teamFields,
		Wheres: []model.QueryWhere{
			{Column: "team_id", Value: teamIDs, OP: "in"},
		},
	})
	if err != nil {
		return nil, fmt.Errorf(ErrFailedToGetTeam, err)
	}

	// Append role_id and is_owner to each team
	for i := range teams {
		teamID := teams[i]["team_id"].(string)
		if roleID, exists := teamRoleMap[teamID]; exists {
			teams[i]["role_id"] = roleID
		}

		// Check if user is the owner of this team
		ownerID := ""
		if owner, ok := teams[i]["owner_id"]; ok && owner != nil {
			ownerID = fmt.Sprintf("%v", owner)
		}
		teams[i]["is_owner"] = (ownerID == memberID)
	}

	return teams, nil
}

// GetTeamByMember retrieves a specific team by team_id and member_id, verifying membership
// Returns the team with role information if the user is a member, or error if not
func (u *DefaultUser) GetTeamByMember(ctx context.Context, teamID string, memberID string) (maps.MapStrAny, error) {
	// First, verify the user is a member of this team
	memberParam := model.QueryParam{
		Select: []interface{}{"team_id", "user_id", "member_type", "role_id"},
		Wheres: []model.QueryWhere{
			{Column: "team_id", Value: teamID},
			{Column: "user_id", Value: memberID},
			{Column: "member_type", Value: "user"},
			{Column: "status", Value: "active"},
		},
		Limit: 1,
	}

	memberModel := model.Select(u.memberModel)
	members, err := memberModel.Get(memberParam)
	if err != nil {
		return nil, fmt.Errorf("failed to verify team membership: %w", err)
	}

	if len(members) == 0 {
		return nil, fmt.Errorf("user is not a member of the team")
	}

	// Get role_id from member record
	roleID := ""
	if role, ok := members[0]["role_id"]; ok && role != nil {
		roleID = fmt.Sprintf("%v", role)
	}

	// Get team details
	teamData, err := u.GetTeamDetail(ctx, teamID)
	if err != nil {
		return nil, fmt.Errorf("failed to get team details: %w", err)
	}

	// Add role_id to team data
	teamData["role_id"] = roleID

	// Check if user is the owner
	ownerID := ""
	if owner, ok := teamData["owner_id"]; ok && owner != nil {
		ownerID = fmt.Sprintf("%v", owner)
	}
	teamData["is_owner"] = (ownerID == memberID)

	return teamData, nil
}

// CountTeamsByMember returns total count of teams by member_id
func (u *DefaultUser) CountTeamsByMember(ctx context.Context, memberID string) (int64, error) {

	param := model.QueryParam{
		Select: []interface{}{"team_id", "user_id", "member_type"},
		Wheres: []model.QueryWhere{
			{Column: "user_id", Value: memberID},
			{Column: "member_type", Value: "user"},
			{Column: "status", Value: "active"},
		},
	}
	// Use Paginate with a small page size to get the total count
	// This is more reliable than manual COUNT(*) queries
	m := model.Select(u.memberModel)
	result, err := m.Paginate(param, 1, 1) // Get first page with 1 item to get total
	if err != nil {
		return 0, fmt.Errorf(ErrFailedToGetTeam, err)
	}

	// Extract total from pagination result using utility function
	if totalInterface, ok := result["total"]; ok {
		return parseIntFromDB(totalInterface)
	}

	return 0, fmt.Errorf("total not found in pagination result")
}

// GetTeamsByStatus retrieves teams by status
func (u *DefaultUser) GetTeamsByStatus(ctx context.Context, status string) ([]maps.MapStr, error) {
	param := model.QueryParam{
		Select: u.teamFields,
		Wheres: []model.QueryWhere{
			{Column: "status", Value: status},
		},
		Orders: []model.QueryOrder{
			{Column: "created_at", Option: "desc"},
		},
	}

	return u.GetTeams(ctx, param)
}

// UpdateTeamStatus updates team status
func (u *DefaultUser) UpdateTeamStatus(ctx context.Context, teamID string, status string) error {
	updateData := maps.MapStrAny{
		"status": status,
	}

	return u.UpdateTeam(ctx, teamID, updateData)
}

// VerifyTeam marks a team as verified
func (u *DefaultUser) VerifyTeam(ctx context.Context, teamID string, verifiedBy string) error {
	updateData := maps.MapStrAny{
		"is_verified": true,
		"verified_by": verifiedBy,
		"verified_at": time.Now(), // Set current timestamp explicitly
	}

	// Direct model update to bypass sensitive field filtering
	m := model.Select(u.teamModel)
	affected, err := m.UpdateWhere(model.QueryParam{
		Wheres: []model.QueryWhere{
			{Column: "team_id", Value: teamID},
		},
		Limit: 1,
	}, updateData)

	if err != nil {
		return fmt.Errorf(ErrFailedToUpdateTeam, err)
	}

	if affected == 0 {
		return fmt.Errorf(ErrTeamNotFound)
	}

	return nil
}

// UnverifyTeam removes verification from a team
func (u *DefaultUser) UnverifyTeam(ctx context.Context, teamID string) error {
	updateData := maps.MapStrAny{
		"is_verified": false,
		"verified_by": nil,
		"verified_at": nil,
	}

	// Direct model update to bypass sensitive field filtering
	m := model.Select(u.teamModel)
	affected, err := m.UpdateWhere(model.QueryParam{
		Wheres: []model.QueryWhere{
			{Column: "team_id", Value: teamID},
		},
		Limit: 1,
	}, updateData)

	if err != nil {
		return fmt.Errorf(ErrFailedToUpdateTeam, err)
	}

	if affected == 0 {
		return fmt.Errorf(ErrTeamNotFound)
	}

	return nil
}

// TransferTeamOwnership transfers team ownership to another user
func (u *DefaultUser) TransferTeamOwnership(ctx context.Context, teamID string, newOwnerID string) error {
	// First verify the new owner exists
	exists, err := u.UserExists(ctx, newOwnerID)
	if err != nil {
		return fmt.Errorf("failed to verify new owner: %w", err)
	}
	if !exists {
		return fmt.Errorf("new owner user not found: %s", newOwnerID)
	}

	updateData := maps.MapStrAny{
		"owner_id": newOwnerID,
	}

	return u.UpdateTeam(ctx, teamID, updateData)
}

// IsTeamOwner checks if a user is the owner of a team
func (u *DefaultUser) IsTeamOwner(ctx context.Context, teamID string, userID string) (bool, error) {
	teamData, err := u.GetTeam(ctx, teamID)
	if err != nil {
		return false, fmt.Errorf("failed to get team: %w", err)
	}

	ownerID, ok := teamData["owner_id"].(string)
	if !ok {
		return false, fmt.Errorf("invalid owner_id type in team data")
	}

	return ownerID == userID, nil
}

// IsTeamMember checks if a user is a member of a team (includes owner)
func (u *DefaultUser) IsTeamMember(ctx context.Context, teamID string, userID string) (bool, error) {
	// First check if user is the owner
	isOwner, err := u.IsTeamOwner(ctx, teamID, userID)
	if err != nil {
		return false, err
	}
	if isOwner {
		return true, nil
	}

	// Then check if user is a member
	return u.MemberExists(ctx, teamID, userID)
}

// CheckTeamAccess checks user's access level to a team
// Returns: (isOwner bool, isMember bool, error)
func (u *DefaultUser) CheckTeamAccess(ctx context.Context, teamID string, userID string) (bool, bool, error) {
	// Check if user is the owner
	isOwner, err := u.IsTeamOwner(ctx, teamID, userID)
	if err != nil {
		return false, false, err
	}

	// Check if user is a member (this will return true for owner as well, but we already know that)
	isMember, err := u.IsTeamMember(ctx, teamID, userID)
	if err != nil {
		return false, false, err
	}

	return isOwner, isMember, nil
}
