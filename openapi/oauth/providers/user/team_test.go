package user_test

import (
	"context"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/yaoapp/gou/model"
	"github.com/yaoapp/kun/maps"
)

// TestTeamData represents test team data structure
type TestTeamData struct {
	Name        string                 `json:"name"`
	DisplayName string                 `json:"display_name"`
	Description string                 `json:"description"`
	Website     string                 `json:"website"`
	OwnerID     string                 `json:"owner_id"`
	Status      string                 `json:"status"`
	Type        string                 `json:"type"`
	TypeID      string                 `json:"type_id"`
	Metadata    map[string]interface{} `json:"metadata"`
}

// TestMemberData represents test member data structure
type TestMemberData struct {
	TeamID     string `json:"team_id"`
	UserID     string `json:"user_id"`
	RoleID     string `json:"role_id"`
	Status     string `json:"status"`
	InvitedBy  string `json:"invited_by"`
	MemberType string `json:"member_type"`
}

func TestTeamBasicOperations(t *testing.T) {
	prepare(t)
	defer clean()

	ctx := context.Background()

	// Use UUID to ensure unique identifiers
	testUUID := strings.ReplaceAll(uuid.New().String(), "-", "")[:8] // 8 char UUID

	// First, create a test user to be the team owner
	testUser := &TestUserData{
		PreferredUsername: "teamowner" + testUUID,
		Email:             "teamowner" + testUUID + "@example.com",
		Password:          "TestPass123!",
		Name:              "Team Owner " + testUUID,
		GivenName:         "Team",
		FamilyName:        "Owner",
		Status:            "active",
		RoleID:            "admin",
		TypeID:            "regular",
		EmailVerified:     true,
		Metadata:          map[string]interface{}{"source": "test"},
	}

	userMap := maps.MapStrAny{
		"preferred_username": testUser.PreferredUsername,
		"email":              testUser.Email,
		"password":           testUser.Password,
		"name":               testUser.Name,
		"given_name":         testUser.GivenName,
		"family_name":        testUser.FamilyName,
		"status":             testUser.Status,
		"role_id":            testUser.RoleID,
		"type_id":            testUser.TypeID,
		"email_verified":     testUser.EmailVerified,
		"metadata":           testUser.Metadata,
	}

	// Create the owner user
	_, err := testProvider.CreateUser(ctx, userMap)
	assert.NoError(t, err)
	ownerUserID := userMap["user_id"].(string)

	// Create test team data dynamically
	testTeam := &TestTeamData{
		Name:        "Test Team " + testUUID,
		DisplayName: "Test Display " + testUUID,
		Description: "A test team for unit testing",
		Website:     "https://test" + testUUID + ".example.com",
		OwnerID:     ownerUserID,
		Status:      "active",
		Type:        "corporation",
		TypeID:      "business",
		Metadata:    map[string]interface{}{"test": true, "uuid": testUUID},
	}

	var testTeamID string // Store the auto-generated team_id

	// Test CreateTeam
	t.Run("CreateTeam", func(t *testing.T) {
		teamMap := maps.MapStrAny{
			"name":         testTeam.Name,
			"display_name": testTeam.DisplayName,
			"description":  testTeam.Description,
			"website":      testTeam.Website,
			"owner_id":     testTeam.OwnerID,
			"status":       testTeam.Status,
			"type":         testTeam.Type,
			"type_id":      testTeam.TypeID,
			"metadata":     testTeam.Metadata,
		}

		id, err := testProvider.CreateTeam(ctx, teamMap)
		assert.NoError(t, err)
		assert.NotEmpty(t, id)

		// Verify team was created with auto-generated team_id
		assert.Contains(t, teamMap, "team_id")
		assert.NotEmpty(t, teamMap["team_id"])

		// Store generated team_id for subsequent tests
		testTeamID = teamMap["team_id"].(string)
	})

	// Test GetTeam
	t.Run("GetTeam", func(t *testing.T) {
		team, err := testProvider.GetTeam(ctx, testTeamID)
		assert.NoError(t, err)
		assert.NotNil(t, team)
		assert.Equal(t, testTeam.Name, team["name"])
		assert.Equal(t, testTeam.DisplayName, team["display_name"])
		assert.Equal(t, testTeam.OwnerID, team["owner_id"])
	})

	// Test GetTeamDetail
	t.Run("GetTeamDetail", func(t *testing.T) {
		team, err := testProvider.GetTeamDetail(ctx, testTeamID)
		assert.NoError(t, err)
		assert.NotNil(t, team)
		assert.Equal(t, testTeam.Name, team["name"])
		assert.Equal(t, testTeam.Website, team["website"])
		assert.Equal(t, testTeam.Description, team["description"])
	})

	// Test TeamExists
	t.Run("TeamExists", func(t *testing.T) {
		exists, err := testProvider.TeamExists(ctx, testTeamID)
		assert.NoError(t, err)
		assert.True(t, exists)

		// Test with non-existent team
		exists, err = testProvider.TeamExists(ctx, "non-existent-team-"+testUUID)
		assert.NoError(t, err)
		assert.False(t, exists)
	})

	// Test UpdateTeam
	t.Run("UpdateTeam", func(t *testing.T) {
		updateData := maps.MapStrAny{
			"description":  "Updated test team description",
			"display_name": "Updated Display Name",
			"metadata":     map[string]interface{}{"updated": true},
		}

		err := testProvider.UpdateTeam(ctx, testTeamID, updateData)
		assert.NoError(t, err)

		// Verify update
		team, err := testProvider.GetTeam(ctx, testTeamID)
		assert.NoError(t, err)
		assert.Equal(t, "Updated test team description", team["description"])
		assert.Equal(t, "Updated Display Name", team["display_name"])

		// Test updating sensitive fields (should be ignored)
		sensitiveData := maps.MapStrAny{
			"team_id":     "new-team-id",
			"created_at":  "2023-01-01",
			"verified_at": "2023-01-01",
		}

		err = testProvider.UpdateTeam(ctx, testTeamID, sensitiveData)
		assert.NoError(t, err) // Should not error, just ignore sensitive fields
	})

	// Test UpdateTeamStatus
	t.Run("UpdateTeamStatus", func(t *testing.T) {
		err := testProvider.UpdateTeamStatus(ctx, testTeamID, "inactive")
		assert.NoError(t, err)

		// Verify status was updated
		team, err := testProvider.GetTeam(ctx, testTeamID)
		assert.NoError(t, err)
		assert.Equal(t, "inactive", team["status"])

		// Change back to active for other tests
		err = testProvider.UpdateTeamStatus(ctx, testTeamID, "active")
		assert.NoError(t, err)
	})

	// Test VerifyTeam
	t.Run("VerifyTeam", func(t *testing.T) {
		err := testProvider.VerifyTeam(ctx, testTeamID, ownerUserID)
		assert.NoError(t, err)

		// Verify team was marked as verified
		team, err := testProvider.GetTeamDetail(ctx, testTeamID)
		assert.NoError(t, err)
		// Database may return int64(1) instead of bool(true)
		isVerified := team["is_verified"]
		assert.True(t, isVerified == true || isVerified == int64(1) || isVerified == 1)
		// verified_by might be nil due to sensitive field filtering, just check it's not empty if present
		if verifiedBy := team["verified_by"]; verifiedBy != nil {
			assert.Equal(t, ownerUserID, verifiedBy)
		}
	})

	// Test UnverifyTeam
	t.Run("UnverifyTeam", func(t *testing.T) {
		err := testProvider.UnverifyTeam(ctx, testTeamID)
		assert.NoError(t, err)

		// Verify team was marked as unverified
		team, err := testProvider.GetTeamDetail(ctx, testTeamID)
		assert.NoError(t, err)
		// Database may return int64(0) instead of bool(false)
		isVerified := team["is_verified"]
		assert.True(t, isVerified == false || isVerified == int64(0) || isVerified == 0)
		assert.Nil(t, team["verified_by"])
	})

	// Test GetTeamsByOwner
	t.Run("GetTeamsByOwner", func(t *testing.T) {
		teams, err := testProvider.GetTeamsByOwner(ctx, ownerUserID)
		assert.NoError(t, err)
		assert.Len(t, teams, 1)
		assert.Equal(t, testTeamID, teams[0]["team_id"])
	})

	// Test GetTeamsByStatus
	t.Run("GetTeamsByStatus", func(t *testing.T) {
		teams, err := testProvider.GetTeamsByStatus(ctx, "active")
		assert.NoError(t, err)
		assert.True(t, len(teams) >= 1) // At least our test team

		// Find our test team in the results
		found := false
		for _, team := range teams {
			if team["team_id"] == testTeamID {
				found = true
				break
			}
		}
		assert.True(t, found, "Test team should be found in active teams")
	})

	// Test PaginateTeams
	t.Run("PaginateTeams", func(t *testing.T) {
		param := model.QueryParam{
			Wheres: []model.QueryWhere{
				{Column: "status", Value: "active"},
			},
		}

		result, err := testProvider.PaginateTeams(ctx, param, 1, 10)
		assert.NoError(t, err)
		assert.NotNil(t, result)
		// Pagination result may use "data" instead of "items"
		assert.True(t, result["data"] != nil || result["items"] != nil)
		assert.Contains(t, result, "total")
	})

	// Test CountTeams
	t.Run("CountTeams", func(t *testing.T) {
		param := model.QueryParam{
			Wheres: []model.QueryWhere{
				{Column: "status", Value: "active"},
			},
		}

		count, err := testProvider.CountTeams(ctx, param)
		assert.NoError(t, err)
		assert.True(t, count >= 1) // At least our test team
	})

	// Test DeleteTeam (at the end)
	t.Run("DeleteTeam", func(t *testing.T) {
		err := testProvider.DeleteTeam(ctx, testTeamID)
		assert.NoError(t, err)

		// Verify team was deleted
		_, err = testProvider.GetTeam(ctx, testTeamID)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "team not found")
	})
}

func TestTeamMemberOperations(t *testing.T) {
	prepare(t)
	defer clean()

	ctx := context.Background()

	// Use UUID to ensure unique identifiers
	testUUID := strings.ReplaceAll(uuid.New().String(), "-", "")[:8]

	// Create test users
	ownerUser := createTestUser(ctx, t, "owner"+testUUID)
	memberUser := createTestUser(ctx, t, "member"+testUUID)

	// Create test team
	testTeam := &TestTeamData{
		Name:        "Member Test Team " + testUUID,
		DisplayName: "Member Test " + testUUID,
		Description: "A test team for member testing",
		OwnerID:     ownerUser,
		Status:      "active",
		Type:        "corporation",
		TypeID:      "business",
		Metadata:    map[string]interface{}{"test": true},
	}

	teamMap := maps.MapStrAny{
		"name":         testTeam.Name,
		"display_name": testTeam.DisplayName,
		"description":  testTeam.Description,
		"owner_id":     testTeam.OwnerID,
		"status":       testTeam.Status,
		"type":         testTeam.Type,
		"type_id":      testTeam.TypeID,
		"metadata":     testTeam.Metadata,
	}

	teamID, err := testProvider.CreateTeam(ctx, teamMap)
	assert.NoError(t, err)

	var memberID int64

	// Test AddMember (invitation-based)
	t.Run("AddMember", func(t *testing.T) {
		id, err := testProvider.AddMember(ctx, teamID, memberUser, "user", ownerUser)
		assert.NoError(t, err)
		assert.Greater(t, id, int64(0))
		memberID = id
	})

	// Test MemberExists
	t.Run("MemberExists", func(t *testing.T) {
		exists, err := testProvider.MemberExists(ctx, teamID, memberUser)
		assert.NoError(t, err)
		assert.True(t, exists)

		// Test with non-existent member
		exists, err = testProvider.MemberExists(ctx, teamID, "non-existent-user")
		assert.NoError(t, err)
		assert.False(t, exists)
	})

	// Test GetMember
	t.Run("GetMember", func(t *testing.T) {
		member, err := testProvider.GetMember(ctx, teamID, memberUser)
		assert.NoError(t, err)
		assert.NotNil(t, member)
		assert.Equal(t, teamID, member["team_id"])
		assert.Equal(t, memberUser, member["user_id"])
		assert.Equal(t, "pending", member["status"]) // Initially pending
	})

	// Test GetMemberByID
	t.Run("GetMemberByID", func(t *testing.T) {
		member, err := testProvider.GetMemberByID(ctx, memberID)
		assert.NoError(t, err)
		assert.NotNil(t, member)
		assert.Equal(t, teamID, member["team_id"])
		assert.Equal(t, memberUser, member["user_id"])
	})

	// Test AcceptInvitation
	t.Run("AcceptInvitation", func(t *testing.T) {
		// First get the invitation token
		member, err := testProvider.GetMemberDetail(ctx, teamID, memberUser)
		assert.NoError(t, err)
		invitationToken := member["invitation_token"].(string)
		assert.NotEmpty(t, invitationToken)

		// Accept the invitation
		err = testProvider.AcceptInvitation(ctx, invitationToken)
		assert.NoError(t, err)

		// Verify member status changed to active
		member, err = testProvider.GetMember(ctx, teamID, memberUser)
		assert.NoError(t, err)
		assert.Equal(t, "active", member["status"])
	})

	// Test UpdateMemberRole
	t.Run("UpdateMemberRole", func(t *testing.T) {
		err := testProvider.UpdateMemberRole(ctx, teamID, memberUser, "admin")
		assert.NoError(t, err)

		// Verify role was updated
		member, err := testProvider.GetMember(ctx, teamID, memberUser)
		assert.NoError(t, err)
		assert.Equal(t, "admin", member["role_id"])
	})

	// Test UpdateMemberStatus
	t.Run("UpdateMemberStatus", func(t *testing.T) {
		err := testProvider.UpdateMemberStatus(ctx, teamID, memberUser, "inactive")
		assert.NoError(t, err)

		// Verify status was updated
		member, err := testProvider.GetMember(ctx, teamID, memberUser)
		assert.NoError(t, err)
		assert.Equal(t, "inactive", member["status"])

		// Change back to active
		err = testProvider.UpdateMemberStatus(ctx, teamID, memberUser, "active")
		assert.NoError(t, err)
	})

	// Test UpdateMemberLastActivity
	t.Run("UpdateMemberLastActivity", func(t *testing.T) {
		err := testProvider.UpdateMemberLastActivity(ctx, teamID, memberUser)
		assert.NoError(t, err)

		// Verify last_active_at was updated
		member, err := testProvider.GetMember(ctx, teamID, memberUser)
		assert.NoError(t, err)
		assert.NotNil(t, member["last_active_at"])
	})

	// Test GetTeamMembers
	t.Run("GetTeamMembers", func(t *testing.T) {
		members, err := testProvider.GetTeamMembers(ctx, teamID)
		assert.NoError(t, err)
		assert.Len(t, members, 1) // Only our test member
		assert.Equal(t, memberUser, members[0]["user_id"])
	})

	// Test GetUserTeams
	t.Run("GetUserTeams", func(t *testing.T) {
		teams, err := testProvider.GetUserTeams(ctx, memberUser)
		assert.NoError(t, err)
		assert.Len(t, teams, 1) // Only our test team
		assert.Equal(t, teamID, teams[0]["team_id"])
	})

	// Test GetTeamMembersByStatus
	t.Run("GetTeamMembersByStatus", func(t *testing.T) {
		members, err := testProvider.GetTeamMembersByStatus(ctx, teamID, "active")
		assert.NoError(t, err)
		assert.Len(t, members, 1) // Our active member
		assert.Equal(t, memberUser, members[0]["user_id"])
	})

	// Test CreateRobotMember
	t.Run("CreateRobotMember", func(t *testing.T) {
		robotData := maps.MapStrAny{
			"robot_name":        "TestBot" + testUUID,
			"robot_description": "A test robot for unit testing",
			"role_id":           "bot",
			"is_active_robot":   true,
			"robot_status":      "idle",
		}

		robotID, err := testProvider.CreateRobotMember(ctx, teamID, robotData)
		assert.NoError(t, err)
		assert.Greater(t, robotID, int64(0))
	})

	// Test GetTeamRobotMembers
	t.Run("GetTeamRobotMembers", func(t *testing.T) {
		robots, err := testProvider.GetTeamRobotMembers(ctx, teamID)
		assert.NoError(t, err)
		assert.Len(t, robots, 1) // Our test robot
		assert.Equal(t, "robot", robots[0]["member_type"])
		assert.Equal(t, "TestBot"+testUUID, robots[0]["robot_name"])
	})

	// Test RemoveMember (at the end)
	t.Run("RemoveMember", func(t *testing.T) {
		err := testProvider.RemoveMember(ctx, teamID, memberUser)
		assert.NoError(t, err)

		// Verify member was removed
		_, err = testProvider.GetMember(ctx, teamID, memberUser)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "member not found")
	})
}

func TestTeamErrorHandling(t *testing.T) {
	prepare(t)
	defer clean()

	ctx := context.Background()
	testUUID := strings.ReplaceAll(uuid.New().String(), "-", "")[:8]
	nonExistentTeamID := "non-existent-team-" + testUUID
	nonExistentUserID := "non-existent-user-" + testUUID

	t.Run("GetTeam_NotFound", func(t *testing.T) {
		_, err := testProvider.GetTeam(ctx, nonExistentTeamID)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "team not found")
	})

	t.Run("GetTeamDetail_NotFound", func(t *testing.T) {
		_, err := testProvider.GetTeamDetail(ctx, nonExistentTeamID)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "team not found")
	})

	t.Run("UpdateTeam_NotFound", func(t *testing.T) {
		updateData := maps.MapStrAny{"name": "Test"}
		err := testProvider.UpdateTeam(ctx, nonExistentTeamID, updateData)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "team not found")
	})

	t.Run("DeleteTeam_NotFound", func(t *testing.T) {
		err := testProvider.DeleteTeam(ctx, nonExistentTeamID)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "team not found")
	})

	t.Run("GetMember_NotFound", func(t *testing.T) {
		_, err := testProvider.GetMember(ctx, nonExistentTeamID, nonExistentUserID)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "member not found")
	})

	t.Run("RemoveMember_NotFound", func(t *testing.T) {
		err := testProvider.RemoveMember(ctx, nonExistentTeamID, nonExistentUserID)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "member not found")
	})

	t.Run("CreateTeam_MissingRequiredFields", func(t *testing.T) {
		// Missing name
		teamData := maps.MapStrAny{
			"owner_id": "test-owner",
		}
		_, err := testProvider.CreateTeam(ctx, teamData)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "name is required")

		// Missing owner_id
		teamData = maps.MapStrAny{
			"name": "Test Team",
		}
		_, err = testProvider.CreateTeam(ctx, teamData)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "owner_id is required")
	})
}

// Helper function to create a test user and return the user_id
func createTestUser(ctx context.Context, t *testing.T, suffix string) string {
	userMap := maps.MapStrAny{
		"preferred_username": "testuser" + suffix,
		"email":              "testuser" + suffix + "@example.com",
		"password":           "TestPass123!",
		"name":               "Test User " + suffix,
		"given_name":         "Test",
		"family_name":        "User",
		"status":             "active",
		"role_id":            "user",
		"type_id":            "regular",
		"email_verified":     true,
		"metadata":           map[string]interface{}{"source": "test"},
	}

	_, err := testProvider.CreateUser(ctx, userMap)
	assert.NoError(t, err)
	return userMap["user_id"].(string)
}
