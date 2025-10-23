package user

import (
	"context"
	"fmt"
	"time"

	"github.com/yaoapp/gou/model"
	"github.com/yaoapp/kun/maps"
)

// Member Resource

// GetMember retrieves member information by team_id and user_id
func (u *DefaultUser) GetMember(ctx context.Context, teamID string, userID string) (maps.MapStrAny, error) {
	m := model.Select(u.memberModel)
	members, err := m.Get(model.QueryParam{
		Select: u.memberFields,
		Wheres: []model.QueryWhere{
			{Column: "team_id", Value: teamID},
			{Column: "user_id", Value: userID},
		},
		Limit: 1,
	})

	if err != nil {
		return nil, fmt.Errorf(ErrFailedToGetMember, err)
	}

	if len(members) == 0 {
		return nil, fmt.Errorf(ErrMemberNotFound)
	}

	return members[0], nil
}

// GetMemberDetail retrieves detailed member information
func (u *DefaultUser) GetMemberDetail(ctx context.Context, teamID string, userID string) (maps.MapStrAny, error) {
	m := model.Select(u.memberModel)
	members, err := m.Get(model.QueryParam{
		Select: u.memberDetailFields,
		Wheres: []model.QueryWhere{
			{Column: "team_id", Value: teamID},
			{Column: "user_id", Value: userID},
		},
		Limit: 1,
	})

	if err != nil {
		return nil, fmt.Errorf(ErrFailedToGetMember, err)
	}

	if len(members) == 0 {
		return nil, fmt.Errorf(ErrMemberNotFound)
	}

	return members[0], nil
}

// GetMemberByID retrieves member information by internal ID
func (u *DefaultUser) GetMemberByID(ctx context.Context, memberID int64) (maps.MapStrAny, error) {
	m := model.Select(u.memberModel)
	members, err := m.Get(model.QueryParam{
		Select: u.memberFields,
		Wheres: []model.QueryWhere{
			{Column: "id", Value: memberID},
		},
		Limit: 1,
	})

	if err != nil {
		return nil, fmt.Errorf(ErrFailedToGetMember, err)
	}

	if len(members) == 0 {
		return nil, fmt.Errorf(ErrMemberNotFound)
	}

	return members[0], nil
}

// GetMemberByInvitationID retrieves member information by invitation_id
func (u *DefaultUser) GetMemberByInvitationID(ctx context.Context, invitationID string) (maps.MapStrAny, error) {
	m := model.Select(u.memberModel)
	members, err := m.Get(model.QueryParam{
		Select: u.memberFields,
		Wheres: []model.QueryWhere{
			{Column: "invitation_id", Value: invitationID},
		},
		Limit: 1,
	})

	if err != nil {
		return nil, fmt.Errorf(ErrFailedToGetMember, err)
	}

	if len(members) == 0 {
		return nil, fmt.Errorf(ErrMemberNotFound)
	}

	return members[0], nil
}

// MemberExists checks if a member exists by team_id and user_id
func (u *DefaultUser) MemberExists(ctx context.Context, teamID string, userID string) (bool, error) {
	m := model.Select(u.memberModel)
	members, err := m.Get(model.QueryParam{
		Select: []interface{}{"id"}, // Only select ID for existence check
		Wheres: []model.QueryWhere{
			{Column: "team_id", Value: teamID},
			{Column: "user_id", Value: userID},
		},
		Limit: 1,
	})

	if err != nil {
		return false, fmt.Errorf(ErrFailedToGetMember, err)
	}

	return len(members) > 0, nil
}

// CreateMember creates a new team member (user type)
func (u *DefaultUser) CreateMember(ctx context.Context, memberData maps.MapStrAny) (int64, error) {
	// Validate required fields for user members
	if _, exists := memberData["team_id"]; !exists {
		return 0, fmt.Errorf("team_id is required in memberData")
	}
	if _, exists := memberData["role_id"]; !exists {
		return 0, fmt.Errorf("role_id is required in memberData")
	}

	// Add __yao_team_id to the member data
	memberData["__yao_team_id"] = memberData["team_id"]

	// Set default values if not provided
	if _, exists := memberData["member_type"]; !exists {
		memberData["member_type"] = "user"
	}
	if _, exists := memberData["status"]; !exists {
		memberData["status"] = "pending"
	}

	// For user members, user_id is required unless it's an invitation (status=pending)
	memberType := memberData["member_type"].(string)
	status, _ := memberData["status"].(string)

	if memberType == "user" && status != "pending" {
		if _, exists := memberData["user_id"]; !exists {
			return 0, fmt.Errorf("user_id is required for active user members")
		}
	}

	// Generate invitation_id for pending invitations
	if status == "pending" && memberData["invitation_id"] == nil {
		invitationID, err := u.generateInvitationID()
		if err != nil {
			return 0, fmt.Errorf("failed to generate invitation ID: %w", err)
		}
		memberData["invitation_id"] = invitationID
	}

	m := model.Select(u.memberModel)
	id, err := m.Create(memberData)
	if err != nil {
		return 0, fmt.Errorf(ErrFailedToCreateMember, err)
	}

	return int64(id), nil
}

// CreateRobotMember creates a new robot member
func (u *DefaultUser) CreateRobotMember(ctx context.Context, teamID string, robotData maps.MapStrAny) (int64, error) {
	// Validate required fields for robot members
	if _, exists := robotData["robot_name"]; !exists {
		return 0, fmt.Errorf("robot_name is required for robot members")
	}
	if _, exists := robotData["role_id"]; !exists {
		return 0, fmt.Errorf("role_id is required for robot members")
	}

	memberData := maps.MapStrAny{
		"team_id":     teamID,
		"member_type": "robot",
		"status":      "active", // Robots are typically active by default
		"user_id":     nil,      // Robots don't have user_id
	}

	// Copy robot-specific fields
	robotFields := []string{
		"role_id", "robot_name", "robot_description", "robot_avatar",
		"robot_config", "agents", "tools", "mcp_servers", "data_access_permissions",
		"system_prompt", "is_active_robot", "schedule_config", "random_activity",
		"activity_frequency", "robot_status",
	}

	for _, field := range robotFields {
		if value, exists := robotData[field]; exists {
			memberData[field] = value
		}
	}

	// Set default robot status if not provided
	if _, exists := memberData["robot_status"]; !exists {
		memberData["robot_status"] = "idle"
	}

	return u.CreateMember(ctx, memberData)
}

// AddMember adds a user to a team (invitation-based)
func (u *DefaultUser) AddMember(ctx context.Context, teamID string, userID string, roleID string, invitedBy string) (int64, error) {
	// Check if member already exists
	exists, err := u.MemberExists(ctx, teamID, userID)
	if err != nil {
		return 0, fmt.Errorf("failed to check member existence: %w", err)
	}
	if exists {
		return 0, fmt.Errorf("user is already a member of this team")
	}

	// Generate invitation token
	token, err := generateRandomPassword(32) // Use existing password generation for token
	if err != nil {
		return 0, fmt.Errorf("failed to generate invitation token: %w", err)
	}

	memberData := maps.MapStrAny{
		"team_id":               teamID,
		"user_id":               userID,
		"member_type":           "user",
		"role_id":               roleID,
		"status":                "pending",
		"invited_by":            invitedBy,
		"invited_at":            time.Now(),
		"invitation_token":      token,
		"invitation_expires_at": time.Now().Add(7 * 24 * time.Hour), // 7 days expiry
	}

	return u.CreateMember(ctx, memberData)
}

// AcceptInvitation accepts a team invitation
// userID can be empty - if provided and invitation doesn't have user_id, it will be updated
func (u *DefaultUser) AcceptInvitation(ctx context.Context, invitationID string, invitationToken string, userID string) error {
	// Find member by invitation_id and token
	m := model.Select(u.memberModel)
	members, err := m.Get(model.QueryParam{
		Select: []interface{}{"id", "team_id", "user_id", "status", "invitation_expires_at"},
		Wheres: []model.QueryWhere{
			{Column: "invitation_id", Value: invitationID},
			{Column: "invitation_token", Value: invitationToken},
			{Column: "status", Value: "pending"},
		},
		Limit: 1,
	})

	if err != nil {
		return fmt.Errorf(ErrFailedToGetMember, err)
	}

	if len(members) == 0 {
		return fmt.Errorf("invitation not found or already accepted")
	}

	member := members[0]

	// Check if invitation has expired
	if expired, err := checkTimeExpired(member["invitation_expires_at"]); err == nil && expired {
		return fmt.Errorf("invitation has expired")
	}

	// Update member status to active
	memberID, err := parseIntFromDB(member["id"])
	if err != nil {
		return fmt.Errorf("invalid member ID: %w", err)
	}
	updateData := maps.MapStrAny{
		"status":           "active",
		"joined_at":        time.Now(),
		"invitation_token": nil,    // Clear the token
		"__yao_updated_by": userID, // Set the updated by user ID
	}

	// If invitation doesn't have a user_id (unregistered user invitation), update it with provided userID
	if userID != "" && (member["user_id"] == nil || member["user_id"] == "") {
		updateData["user_id"] = userID
	}

	affected, err := m.UpdateWhere(model.QueryParam{
		Wheres: []model.QueryWhere{
			{Column: "id", Value: memberID},
		},
		Limit: 1,
	}, updateData)

	if err != nil {
		return fmt.Errorf(ErrFailedToUpdateMember, err)
	}

	if affected == 0 {
		return fmt.Errorf(ErrMemberNotFound)
	}

	return nil
}

// UpdateMember updates an existing member
func (u *DefaultUser) UpdateMember(ctx context.Context, teamID string, userID string, memberData maps.MapStrAny) error {
	// Remove sensitive fields that should not be updated directly
	sensitiveFields := []string{"id", "team_id", "user_id", "created_at", "invitation_token"}
	for _, field := range sensitiveFields {
		delete(memberData, field)
	}

	// Skip update if no valid fields remain
	if len(memberData) == 0 {
		return nil
	}

	m := model.Select(u.memberModel)
	affected, err := m.UpdateWhere(model.QueryParam{
		Wheres: []model.QueryWhere{
			{Column: "team_id", Value: teamID},
			{Column: "user_id", Value: userID},
		},
		Limit: 1,
	}, memberData)

	if err != nil {
		return fmt.Errorf(ErrFailedToUpdateMember, err)
	}

	if affected == 0 {
		return fmt.Errorf(ErrMemberNotFound)
	}

	return nil
}

// UpdateMemberByID updates a member by internal ID
func (u *DefaultUser) UpdateMemberByID(ctx context.Context, memberID int64, memberData maps.MapStrAny) error {
	// Remove sensitive fields that should not be updated directly
	sensitiveFields := []string{"id", "team_id", "user_id", "created_at", "invitation_token"}
	for _, field := range sensitiveFields {
		delete(memberData, field)
	}

	// Skip update if no valid fields remain
	if len(memberData) == 0 {
		return nil
	}

	m := model.Select(u.memberModel)
	affected, err := m.UpdateWhere(model.QueryParam{
		Wheres: []model.QueryWhere{
			{Column: "id", Value: memberID},
		},
		Limit: 1,
	}, memberData)

	if err != nil {
		return fmt.Errorf(ErrFailedToUpdateMember, err)
	}

	if affected == 0 {
		return fmt.Errorf(ErrMemberNotFound)
	}

	return nil
}

// RemoveMember removes a member from a team (soft delete)
func (u *DefaultUser) RemoveMember(ctx context.Context, teamID string, userID string) error {
	m := model.Select(u.memberModel)
	affected, err := m.DeleteWhere(model.QueryParam{
		Wheres: []model.QueryWhere{
			{Column: "team_id", Value: teamID},
			{Column: "user_id", Value: userID},
		},
		Limit: 1,
	})

	if err != nil {
		return fmt.Errorf(ErrFailedToDeleteMember, err)
	}

	if affected == 0 {
		return fmt.Errorf(ErrMemberNotFound)
	}

	return nil
}

// RemoveAllTeamMembers removes all members from a team (used when deleting team)
func (u *DefaultUser) RemoveAllTeamMembers(ctx context.Context, teamID string) error {
	m := model.Select(u.memberModel)
	_, err := m.DeleteWhere(model.QueryParam{
		Wheres: []model.QueryWhere{
			{Column: "team_id", Value: teamID},
		},
	})

	if err != nil {
		return fmt.Errorf("failed to delete all team members: %w", err)
	}

	return nil
}

// GetTeamMembers retrieves all members of a team
func (u *DefaultUser) GetTeamMembers(ctx context.Context, teamID string) ([]maps.MapStr, error) {
	param := model.QueryParam{
		Select: u.memberFields,
		Wheres: []model.QueryWhere{
			{Column: "team_id", Value: teamID},
		},
		Orders: []model.QueryOrder{
			{Column: "joined_at", Option: "desc"},
			{Column: "invited_at", Option: "desc"},
		},
	}

	m := model.Select(u.memberModel)
	members, err := m.Get(param)
	if err != nil {
		return nil, fmt.Errorf(ErrFailedToGetMember, err)
	}

	return members, nil
}

// GetUserTeams retrieves all teams a user is a member of
func (u *DefaultUser) GetUserTeams(ctx context.Context, userID string) ([]maps.MapStr, error) {
	param := model.QueryParam{
		Select: u.memberFields,
		Wheres: []model.QueryWhere{
			{Column: "user_id", Value: userID},
		},
		Orders: []model.QueryOrder{
			{Column: "joined_at", Option: "desc"},
		},
	}

	m := model.Select(u.memberModel)
	members, err := m.Get(param)
	if err != nil {
		return nil, fmt.Errorf(ErrFailedToGetMember, err)
	}

	return members, nil
}

// GetTeamMembersByStatus retrieves team members by status
func (u *DefaultUser) GetTeamMembersByStatus(ctx context.Context, teamID string, status string) ([]maps.MapStr, error) {
	param := model.QueryParam{
		Select: u.memberFields,
		Wheres: []model.QueryWhere{
			{Column: "team_id", Value: teamID},
			{Column: "status", Value: status},
		},
		Orders: []model.QueryOrder{
			{Column: "invited_at", Option: "desc"},
		},
	}

	m := model.Select(u.memberModel)
	members, err := m.Get(param)
	if err != nil {
		return nil, fmt.Errorf(ErrFailedToGetMember, err)
	}

	return members, nil
}

// GetTeamRobotMembers retrieves all robot members of a team
func (u *DefaultUser) GetTeamRobotMembers(ctx context.Context, teamID string) ([]maps.MapStr, error) {
	param := model.QueryParam{
		Select: u.memberDetailFields,
		Wheres: []model.QueryWhere{
			{Column: "team_id", Value: teamID},
			{Column: "member_type", Value: "robot"},
		},
		Orders: []model.QueryOrder{
			{Column: "robot_name", Option: "asc"},
		},
	}

	m := model.Select(u.memberModel)
	members, err := m.Get(param)
	if err != nil {
		return nil, fmt.Errorf(ErrFailedToGetMember, err)
	}

	return members, nil
}

// GetActiveRobotMembers retrieves all active robot members across all teams
func (u *DefaultUser) GetActiveRobotMembers(ctx context.Context) ([]maps.MapStr, error) {
	param := model.QueryParam{
		Select: u.memberDetailFields,
		Wheres: []model.QueryWhere{
			{Column: "member_type", Value: "robot"},
			{Column: "is_active_robot", Value: true},
			{Column: "status", Value: "active"},
		},
		Orders: []model.QueryOrder{
			{Column: "last_robot_activity", Option: "asc"}, // Oldest activity first
		},
	}

	m := model.Select(u.memberModel)
	members, err := m.Get(param)
	if err != nil {
		return nil, fmt.Errorf(ErrFailedToGetMember, err)
	}

	return members, nil
}

// UpdateMemberRole updates a member's role
func (u *DefaultUser) UpdateMemberRole(ctx context.Context, teamID string, userID string, roleID string) error {
	updateData := maps.MapStrAny{
		"role_id": roleID,
	}

	return u.UpdateMember(ctx, teamID, userID, updateData)
}

// UpdateMemberStatus updates a member's status
func (u *DefaultUser) UpdateMemberStatus(ctx context.Context, teamID string, userID string, status string) error {
	updateData := maps.MapStrAny{
		"status": status,
	}

	return u.UpdateMember(ctx, teamID, userID, updateData)
}

// UpdateMemberLastActivity updates a member's last activity time
func (u *DefaultUser) UpdateMemberLastActivity(ctx context.Context, teamID string, userID string) error {
	updateData := maps.MapStrAny{
		"last_active_at": time.Now(),
	}

	// Also increment login count
	member, err := u.GetMember(ctx, teamID, userID)
	if err != nil {
		return err
	}

	loginCount := int64(0)
	if count := member["login_count"]; count != nil {
		if parsedCount, err := parseIntFromDB(count); err == nil {
			loginCount = parsedCount
		}
	}
	updateData["login_count"] = loginCount + 1

	return u.UpdateMember(ctx, teamID, userID, updateData)
}

// UpdateRobotActivity updates robot member's last activity and status
func (u *DefaultUser) UpdateRobotActivity(ctx context.Context, memberID int64, robotStatus string) error {
	updateData := maps.MapStrAny{
		"last_robot_activity": time.Now(),
		"robot_status":        robotStatus,
	}

	return u.UpdateMemberByID(ctx, memberID, updateData)
}

// UpdateMemberByInvitationID updates a member by invitation_id
func (u *DefaultUser) UpdateMemberByInvitationID(ctx context.Context, invitationID string, memberData maps.MapStrAny) error {
	// Remove sensitive fields that should not be updated directly
	// Note: user_id is allowed for invitation acceptance (pending -> active transition)
	sensitiveFields := []string{"id", "team_id", "created_at", "invitation_id"}
	for _, field := range sensitiveFields {
		delete(memberData, field)
	}

	// Skip update if no valid fields remain
	if len(memberData) == 0 {
		return nil
	}

	m := model.Select(u.memberModel)
	affected, err := m.UpdateWhere(model.QueryParam{
		Wheres: []model.QueryWhere{
			{Column: "invitation_id", Value: invitationID},
		},
		Limit: 1,
	}, memberData)

	if err != nil {
		return fmt.Errorf(ErrFailedToUpdateMember, err)
	}

	if affected == 0 {
		return fmt.Errorf(ErrMemberNotFound)
	}

	return nil
}

// RemoveMemberByInvitationID removes a member by invitation_id
func (u *DefaultUser) RemoveMemberByInvitationID(ctx context.Context, invitationID string) error {
	m := model.Select(u.memberModel)
	affected, err := m.DeleteWhere(model.QueryParam{
		Wheres: []model.QueryWhere{
			{Column: "invitation_id", Value: invitationID},
		},
		Limit: 1,
	})

	if err != nil {
		return fmt.Errorf(ErrFailedToDeleteMember, err)
	}

	if affected == 0 {
		return fmt.Errorf(ErrMemberNotFound)
	}

	return nil
}

// PaginateMembers retrieves paginated list of members
func (u *DefaultUser) PaginateMembers(ctx context.Context, param model.QueryParam, page int, pagesize int) (maps.MapStr, error) {
	// Set default select fields if not provided
	if param.Select == nil {
		param.Select = u.memberFields
	}

	m := model.Select(u.memberModel)
	result, err := m.Paginate(param, page, pagesize)
	if err != nil {
		return nil, fmt.Errorf(ErrFailedToGetMember, err)
	}

	return result, nil
}
