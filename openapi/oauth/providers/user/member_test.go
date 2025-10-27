package user_test

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/yaoapp/gou/model"
	"github.com/yaoapp/kun/maps"
)

func TestMemberBasicOperations(t *testing.T) {
	prepare(t)
	defer clean()

	ctx := context.Background()

	// Use UUID to ensure unique identifiers
	testUUID := strings.ReplaceAll(uuid.New().String(), "-", "")[:8]

	// Create test users
	ownerUser := createTestUser(ctx, t, "owner"+testUUID)
	memberUser := createTestUser(ctx, t, "member"+testUUID)

	// Create test team
	teamMap := maps.MapStrAny{
		"name":         "Test Team " + testUUID,
		"display_name": "Test Display " + testUUID,
		"description":  "A test team for member testing",
		"owner_id":     ownerUser,
		"status":       "active",
		"type":         "corporation",
		"type_id":      "business",
		"metadata":     map[string]interface{}{"test": true},
	}

	teamID, err := testProvider.CreateTeam(ctx, teamMap)
	assert.NoError(t, err)

	var businessMemberID string

	// Test CreateMember
	t.Run("CreateMember", func(t *testing.T) {
		memberData := maps.MapStrAny{
			"team_id":     teamID,
			"user_id":     memberUser,
			"member_type": "user",
			"role_id":     "user",
			"status":      "active",
		}

		memberID, err := testProvider.CreateMember(ctx, memberData)
		assert.NoError(t, err)
		assert.NotEmpty(t, memberID)
		businessMemberID = memberID
	})

	// Test GetMember
	t.Run("GetMember", func(t *testing.T) {
		member, err := testProvider.GetMember(ctx, teamID, memberUser)
		assert.NoError(t, err)
		assert.NotNil(t, member)
		assert.Equal(t, teamID, member["team_id"])
		assert.Equal(t, memberUser, member["user_id"])
		assert.Equal(t, "user", member["member_type"])
		assert.Equal(t, "user", member["role_id"])
	})

	// Test GetMemberDetail
	t.Run("GetMemberDetail", func(t *testing.T) {
		member, err := testProvider.GetMemberDetail(ctx, teamID, memberUser)
		assert.NoError(t, err)
		assert.NotNil(t, member)
		assert.Equal(t, teamID, member["team_id"])
		assert.Equal(t, memberUser, member["user_id"])
		// Should contain more detailed fields
		assert.Contains(t, member, "created_at")
		assert.Contains(t, member, "updated_at")
	})

	// Test GetMemberByMemberID
	t.Run("GetMemberByMemberID", func(t *testing.T) {
		member, err := testProvider.GetMemberByMemberID(ctx, businessMemberID)
		assert.NoError(t, err)
		assert.NotNil(t, member)
		assert.Equal(t, teamID, member["team_id"])
		assert.Equal(t, memberUser, member["user_id"])
		// Verify member_id is returned
		assert.Equal(t, businessMemberID, member["member_id"])
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

	// Test UpdateMember
	t.Run("UpdateMember", func(t *testing.T) {
		updateData := maps.MapStrAny{
			"role_id": "admin",
			"notes":   "Promoted to admin",
		}

		err := testProvider.UpdateMember(ctx, teamID, memberUser, updateData)
		assert.NoError(t, err)

		// Verify update
		member, err := testProvider.GetMember(ctx, teamID, memberUser)
		assert.NoError(t, err)
		assert.Equal(t, "admin", member["role_id"])

		// Test updating sensitive fields (should be ignored)
		sensitiveData := maps.MapStrAny{
			"id":               999,
			"team_id":          "new-team",
			"user_id":          "new-user",
			"invitation_token": "fake-token",
		}

		err = testProvider.UpdateMember(ctx, teamID, memberUser, sensitiveData)
		assert.NoError(t, err) // Should not error, just ignore sensitive fields
	})

	// Test UpdateMemberByMemberID
	t.Run("UpdateMemberByMemberID", func(t *testing.T) {
		updateData := maps.MapStrAny{
			"status": "inactive",
		}

		err := testProvider.UpdateMemberByMemberID(ctx, businessMemberID, updateData)
		assert.NoError(t, err)

		// Verify update
		member, err := testProvider.GetMemberByMemberID(ctx, businessMemberID)
		assert.NoError(t, err)
		assert.Equal(t, "inactive", member["status"])

		// Change back to active for other tests
		err = testProvider.UpdateMemberByMemberID(ctx, businessMemberID, maps.MapStrAny{"status": "active"})
		assert.NoError(t, err)
	})

	// Test UpdateMemberRole
	t.Run("UpdateMemberRole", func(t *testing.T) {
		err := testProvider.UpdateMemberRole(ctx, teamID, memberUser, "moderator")
		assert.NoError(t, err)

		// Verify role was updated
		member, err := testProvider.GetMember(ctx, teamID, memberUser)
		assert.NoError(t, err)
		assert.Equal(t, "moderator", member["role_id"])
	})

	// Test UpdateMemberStatus
	t.Run("UpdateMemberStatus", func(t *testing.T) {
		err := testProvider.UpdateMemberStatus(ctx, teamID, memberUser, "suspended")
		assert.NoError(t, err)

		// Verify status was updated
		member, err := testProvider.GetMember(ctx, teamID, memberUser)
		assert.NoError(t, err)
		assert.Equal(t, "suspended", member["status"])

		// Change back to active
		err = testProvider.UpdateMemberStatus(ctx, teamID, memberUser, "active")
		assert.NoError(t, err)
	})

	// Test UpdateMemberLastActivity
	t.Run("UpdateMemberLastActivity", func(t *testing.T) {
		err := testProvider.UpdateMemberLastActivity(ctx, teamID, memberUser)
		assert.NoError(t, err)

		// Verify last_active_at was updated and login_count incremented
		member, err := testProvider.GetMember(ctx, teamID, memberUser)
		assert.NoError(t, err)
		assert.NotNil(t, member["last_active_at"])
		// login_count should be at least 1 (handle different integer types)
		loginCount := member["login_count"]
		if loginCount != nil {
			switch v := loginCount.(type) {
			case int:
				assert.True(t, v >= 1, "login_count should be at least 1")
			case int64:
				assert.True(t, v >= 1, "login_count should be at least 1")
			case int32:
				assert.True(t, v >= 1, "login_count should be at least 1")
			default:
				t.Logf("Unexpected login_count type: %T, value: %v", loginCount, loginCount)
				assert.True(t, false, "login_count should be a numeric type")
			}
		} else {
			assert.True(t, false, "login_count should not be nil")
		}
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

func TestMemberInvitationFlow(t *testing.T) {
	prepare(t)
	defer clean()

	ctx := context.Background()

	// Use UUID to ensure unique identifiers
	testUUID := strings.ReplaceAll(uuid.New().String(), "-", "")[:8]

	// Create test users
	ownerUser := createTestUser(ctx, t, "owner"+testUUID)
	inviteeUser := createTestUser(ctx, t, "invitee"+testUUID)

	// Create test team
	teamMap := maps.MapStrAny{
		"name":         "Invitation Test Team " + testUUID,
		"display_name": "Invitation Test " + testUUID,
		"description":  "A test team for invitation testing",
		"owner_id":     ownerUser,
		"status":       "active",
		"type":         "corporation",
		"type_id":      "business",
		"metadata":     map[string]interface{}{"test": true},
	}

	teamID, err := testProvider.CreateTeam(ctx, teamMap)
	assert.NoError(t, err)

	var invitationToken string
	var invitationID string

	// Test AddMember (invitation-based)
	t.Run("AddMember", func(t *testing.T) {
		memberID, err := testProvider.AddMember(ctx, teamID, inviteeUser, "user", ownerUser)
		assert.NoError(t, err)
		assert.NotEmpty(t, memberID)

		// Verify member was created with pending status
		member, err := testProvider.GetMember(ctx, teamID, inviteeUser)
		assert.NoError(t, err)
		assert.Equal(t, "pending", member["status"])
		assert.Equal(t, ownerUser, member["invited_by"])

		// Get invitation token and invitation_id for acceptance test
		memberDetail, err := testProvider.GetMemberDetail(ctx, teamID, inviteeUser)
		assert.NoError(t, err)
		invitationToken = memberDetail["invitation_token"].(string)
		assert.NotEmpty(t, invitationToken)
		invitationID = memberDetail["invitation_id"].(string)
		assert.NotEmpty(t, invitationID)

		// Verify invitation expiry is set
		assert.NotNil(t, memberDetail["invitation_expires_at"])
	})

	// Test duplicate invitation prevention
	t.Run("AddMember_DuplicatePrevention", func(t *testing.T) {
		_, err := testProvider.AddMember(ctx, teamID, inviteeUser, "user", ownerUser)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "already a member")
	})

	// Test AcceptInvitation
	t.Run("AcceptInvitation", func(t *testing.T) {
		err := testProvider.AcceptInvitation(ctx, invitationID, invitationToken, "")
		assert.NoError(t, err)

		// Verify member status changed to active
		member, err := testProvider.GetMember(ctx, teamID, inviteeUser)
		assert.NoError(t, err)
		assert.Equal(t, "active", member["status"])
		assert.NotNil(t, member["joined_at"])

		// Verify invitation token was cleared
		memberDetail, err := testProvider.GetMemberDetail(ctx, teamID, inviteeUser)
		assert.NoError(t, err)
		assert.Nil(t, memberDetail["invitation_token"])
	})

	// Test AcceptInvitation with invalid token
	t.Run("AcceptInvitation_InvalidToken", func(t *testing.T) {
		err := testProvider.AcceptInvitation(ctx, invitationID, "invalid-token", "")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invitation not found")
	})

	// Test AcceptInvitation with already accepted token
	t.Run("AcceptInvitation_AlreadyAccepted", func(t *testing.T) {
		err := testProvider.AcceptInvitation(ctx, invitationID, invitationToken, "")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invitation not found")
	})
}

func TestRobotMemberOperations(t *testing.T) {
	prepare(t)
	defer clean()

	ctx := context.Background()

	// Use UUID to ensure unique identifiers
	testUUID := strings.ReplaceAll(uuid.New().String(), "-", "")[:8]

	// Create test user (team owner)
	ownerUser := createTestUser(ctx, t, "owner"+testUUID)

	// Create test team
	teamMap := maps.MapStrAny{
		"name":         "Robot Test Team " + testUUID,
		"display_name": "Robot Test " + testUUID,
		"description":  "A test team for robot testing",
		"owner_id":     ownerUser,
		"status":       "active",
		"type":         "corporation",
		"type_id":      "business",
		"metadata":     map[string]interface{}{"test": true},
	}

	teamID, err := testProvider.CreateTeam(ctx, teamMap)
	assert.NoError(t, err)

	var robotBusinessMemberID string

	// Test CreateRobotMember
	t.Run("CreateRobotMember", func(t *testing.T) {
		robotData := maps.MapStrAny{
			"display_name":    "TestBot" + testUUID,
			"bio":             "A test robot for unit testing",
			"avatar":          "https://example.com/robot.png",
			"role_id":         "bot",
			"autonomous_mode": true,
			"robot_status":    "idle",
			"system_prompt":   "You are a helpful test robot",
			"language_model":  "gpt-4",
			"cost_limit":      100.00,
			"manager_id":      ownerUser,
			"robot_email":     "testbot" + testUUID + "@robot.example.com",
			"authorized_senders": []string{
				"admin@example.com",
				"manager@example.com",
			},
			"email_filter_rules": []string{
				".*@example\\.com$",
				".*@test\\.com$",
			},
			"robot_config": map[string]interface{}{
				"max_tokens": 1000,
			},
		}

		memberID, err := testProvider.CreateRobotMember(ctx, teamID, robotData)
		assert.NoError(t, err)
		assert.NotEmpty(t, memberID)
		robotBusinessMemberID = memberID

		// Verify robot member was created
		member, err := testProvider.GetMemberByMemberID(ctx, robotBusinessMemberID)
		assert.NoError(t, err)
		assert.Equal(t, "robot", member["member_type"])
		assert.Equal(t, "active", member["status"]) // Robots are active by default
		assert.Nil(t, member["user_id"])            // Robots don't have user_id

		// Verify new email fields in detail view
		memberDetail, err := testProvider.GetMemberDetailByMemberID(ctx, robotBusinessMemberID)
		assert.NoError(t, err)
		assert.Equal(t, "testbot"+testUUID+"@robot.example.com", memberDetail["robot_email"])
		assert.NotNil(t, memberDetail["authorized_senders"])
		assert.NotNil(t, memberDetail["email_filter_rules"])
	})

	// Test GetTeamRobotMembers
	t.Run("GetTeamRobotMembers", func(t *testing.T) {
		robots, err := testProvider.GetTeamRobotMembers(ctx, teamID)
		assert.NoError(t, err)
		assert.Len(t, robots, 1)
		assert.Equal(t, "robot", robots[0]["member_type"])
		assert.Equal(t, "TestBot"+testUUID, robots[0]["display_name"])
		assert.Equal(t, "A test robot for unit testing", robots[0]["bio"])
	})

	// Test UpdateRobotActivity - needs internal database ID
	t.Run("UpdateRobotActivity", func(t *testing.T) {
		// Get internal database ID from member_id
		member, err := testProvider.GetMemberByMemberID(ctx, robotBusinessMemberID)
		assert.NoError(t, err)
		robotTeamID := member["team_id"].(string)

		// UpdateRobotActivity still uses internal database ID
		// We need to query the database to get it
		m := model.Select("__yao.member")
		result, err := m.Get(model.QueryParam{
			Select: []interface{}{"id"},
			Wheres: []model.QueryWhere{
				{Column: "member_id", Value: robotBusinessMemberID},
			},
			Limit: 1,
		})
		assert.NoError(t, err)
		assert.Len(t, result, 1)
		robotDBID := result[0]["id"].(int64)

		err = testProvider.UpdateRobotActivity(ctx, robotDBID, "working")
		assert.NoError(t, err)

		// Get robot members to verify status (robot members don't have user_id)
		robots, err := testProvider.GetTeamRobotMembers(ctx, robotTeamID)
		assert.NoError(t, err)
		assert.Len(t, robots, 1)
		robot := robots[0]
		assert.Equal(t, "working", robot["robot_status"])
		assert.NotNil(t, robot["last_robot_activity"])
	})

	// Test GetActiveRobotMembers
	t.Run("GetActiveRobotMembers", func(t *testing.T) {
		// First make sure our robot is active
		err := testProvider.UpdateMemberByMemberID(ctx, robotBusinessMemberID, maps.MapStrAny{
			"autonomous_mode": true,
			"status":          "active",
		})
		assert.NoError(t, err)

		robots, err := testProvider.GetActiveRobotMembers(ctx)
		assert.NoError(t, err)
		assert.True(t, len(robots) >= 1) // At least our test robot

		// Find our test robot in the results
		found := false
		for _, robot := range robots {
			if robot["display_name"] == "TestBot"+testUUID {
				found = true
				assert.Equal(t, "robot", robot["member_type"])
				// Handle different boolean types from database
				autonomousMode := robot["autonomous_mode"]
				assert.True(t, autonomousMode == true || autonomousMode == int64(1) || autonomousMode == 1, "Robot should be autonomous")
				break
			}
		}
		assert.True(t, found, "Test robot should be found in active robots")
	})

	// Test robot member validation
	t.Run("CreateRobotMember_ValidationErrors", func(t *testing.T) {
		// Missing display_name
		robotData := maps.MapStrAny{
			"role_id": "bot",
		}
		_, err := testProvider.CreateRobotMember(ctx, teamID, robotData)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "display_name is required")

		// Missing role_id
		robotData = maps.MapStrAny{
			"display_name": "TestBot2",
		}
		_, err = testProvider.CreateRobotMember(ctx, teamID, robotData)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "role_id is required")
	})
}

func TestMemberQueryOperations(t *testing.T) {
	prepare(t)
	defer clean()

	ctx := context.Background()

	// Use UUID to ensure unique identifiers
	testUUID := strings.ReplaceAll(uuid.New().String(), "-", "")[:8]

	// Create test users
	ownerUser := createTestUser(ctx, t, "owner"+testUUID)
	member1User := createTestUser(ctx, t, "member1"+testUUID)
	member2User := createTestUser(ctx, t, "member2"+testUUID)

	// Create test teams
	team1Map := maps.MapStrAny{
		"name":         "Query Test Team 1 " + testUUID,
		"display_name": "Query Test 1 " + testUUID,
		"description":  "First test team for query testing",
		"owner_id":     ownerUser,
		"status":       "active",
		"type":         "corporation",
		"type_id":      "business",
		"metadata":     map[string]interface{}{"test": true},
	}

	team1ID, err := testProvider.CreateTeam(ctx, team1Map)
	assert.NoError(t, err)

	team2Map := maps.MapStrAny{
		"name":         "Query Test Team 2 " + testUUID,
		"display_name": "Query Test 2 " + testUUID,
		"description":  "Second test team for query testing",
		"owner_id":     ownerUser,
		"status":       "active",
		"type":         "corporation",
		"type_id":      "business",
		"metadata":     map[string]interface{}{"test": true},
	}

	team2ID, err := testProvider.CreateTeam(ctx, team2Map)
	assert.NoError(t, err)

	// Add members to teams
	_, err = testProvider.CreateMember(ctx, maps.MapStrAny{
		"team_id":     team1ID,
		"user_id":     member1User,
		"member_type": "user",
		"role_id":     "user",
		"status":      "active",
	})
	assert.NoError(t, err)

	_, err = testProvider.CreateMember(ctx, maps.MapStrAny{
		"team_id":     team1ID,
		"user_id":     member2User,
		"member_type": "user",
		"role_id":     "admin",
		"status":      "pending",
	})
	assert.NoError(t, err)

	_, err = testProvider.CreateMember(ctx, maps.MapStrAny{
		"team_id":     team2ID,
		"user_id":     member1User,
		"member_type": "user",
		"role_id":     "moderator",
		"status":      "active",
	})
	assert.NoError(t, err)

	// Test GetTeamMembers
	t.Run("GetTeamMembers", func(t *testing.T) {
		members, err := testProvider.GetTeamMembers(ctx, team1ID)
		assert.NoError(t, err)
		assert.Len(t, members, 2) // member1 and member2

		// Verify members are ordered by joined_at desc, invited_at desc
		userIDs := []string{members[0]["user_id"].(string), members[1]["user_id"].(string)}
		assert.Contains(t, userIDs, member1User)
		assert.Contains(t, userIDs, member2User)
	})

	// Test GetUserTeams
	t.Run("GetUserTeams", func(t *testing.T) {
		teams, err := testProvider.GetUserTeams(ctx, member1User)
		assert.NoError(t, err)
		assert.Len(t, teams, 2) // member1 is in both teams

		teamIDs := []string{teams[0]["team_id"].(string), teams[1]["team_id"].(string)}
		assert.Contains(t, teamIDs, team1ID)
		assert.Contains(t, teamIDs, team2ID)
	})

	// Test GetTeamMembersByStatus
	t.Run("GetTeamMembersByStatus", func(t *testing.T) {
		// Get active members
		activeMembers, err := testProvider.GetTeamMembersByStatus(ctx, team1ID, "active")
		assert.NoError(t, err)
		assert.Len(t, activeMembers, 1) // Only member1 is active
		assert.Equal(t, member1User, activeMembers[0]["user_id"])

		// Get pending members
		pendingMembers, err := testProvider.GetTeamMembersByStatus(ctx, team1ID, "pending")
		assert.NoError(t, err)
		assert.Len(t, pendingMembers, 1) // Only member2 is pending
		assert.Equal(t, member2User, pendingMembers[0]["user_id"])

		// Get inactive members (should be empty)
		inactiveMembers, err := testProvider.GetTeamMembersByStatus(ctx, team1ID, "inactive")
		assert.NoError(t, err)
		assert.Len(t, inactiveMembers, 0)
	})

	// Test PaginateMembers
	t.Run("PaginateMembers", func(t *testing.T) {
		param := model.QueryParam{
			Wheres: []model.QueryWhere{
				{Column: "team_id", Value: team1ID},
			},
		}

		result, err := testProvider.PaginateMembers(ctx, param, 1, 10)
		assert.NoError(t, err)
		assert.NotNil(t, result)
		// Pagination result may use "data" instead of "items"
		assert.True(t, result["data"] != nil || result["items"] != nil)
		assert.Contains(t, result, "total")

		// Total should be 2 (member1 and member2)
		total := result["total"]
		assert.True(t, total == 2 || total == int64(2))
	})
}

func TestMemberErrorHandling(t *testing.T) {
	prepare(t)
	defer clean()

	ctx := context.Background()
	testUUID := strings.ReplaceAll(uuid.New().String(), "-", "")[:8]
	nonExistentTeamID := "non-existent-team-" + testUUID
	nonExistentUserID := "non-existent-user-" + testUUID
	nonExistentMemberID := int64(999999)

	t.Run("GetMember_NotFound", func(t *testing.T) {
		_, err := testProvider.GetMember(ctx, nonExistentTeamID, nonExistentUserID)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "member not found")
	})

	t.Run("GetMemberDetail_NotFound", func(t *testing.T) {
		_, err := testProvider.GetMemberDetail(ctx, nonExistentTeamID, nonExistentUserID)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "member not found")
	})

	t.Run("GetMemberByID_NotFound", func(t *testing.T) {
		_, err := testProvider.GetMemberByID(ctx, nonExistentMemberID)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "member not found")
	})

	t.Run("UpdateMember_NotFound", func(t *testing.T) {
		updateData := maps.MapStrAny{"role_id": "admin"}
		err := testProvider.UpdateMember(ctx, nonExistentTeamID, nonExistentUserID, updateData)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "member not found")
	})

	t.Run("UpdateMemberByID_NotFound", func(t *testing.T) {
		updateData := maps.MapStrAny{"role_id": "admin"}
		err := testProvider.UpdateMemberByID(ctx, nonExistentMemberID, updateData)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "member not found")
	})

	t.Run("RemoveMember_NotFound", func(t *testing.T) {
		err := testProvider.RemoveMember(ctx, nonExistentTeamID, nonExistentUserID)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "member not found")
	})

	t.Run("CreateMember_MissingRequiredFields", func(t *testing.T) {
		// Missing team_id
		memberData := maps.MapStrAny{
			"user_id": "test-user",
			"role_id": "user",
		}
		_, err := testProvider.CreateMember(ctx, memberData)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "team_id is required")

		// Missing role_id
		memberData = maps.MapStrAny{
			"team_id": "test-team",
			"user_id": "test-user",
		}
		_, err = testProvider.CreateMember(ctx, memberData)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "role_id is required")

		// Missing user_id for active user member
		memberData = maps.MapStrAny{
			"team_id":     "test-team",
			"role_id":     "user",
			"member_type": "user",
			"status":      "active", // Explicitly set to active to trigger validation
		}
		_, err = testProvider.CreateMember(ctx, memberData)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "user_id is required for active user members")
	})

	t.Run("UpdateMember_EmptyData", func(t *testing.T) {
		// Create a test member first
		ownerUser := createTestUser(ctx, t, "owner"+testUUID)
		memberUser := createTestUser(ctx, t, "member"+testUUID)

		teamMap := maps.MapStrAny{
			"name":         "Error Test Team " + testUUID,
			"display_name": "Error Test " + testUUID,
			"description":  "A test team for error testing",
			"owner_id":     ownerUser,
			"status":       "active",
		}
		teamID, err := testProvider.CreateTeam(ctx, teamMap)
		assert.NoError(t, err)

		_, err = testProvider.CreateMember(ctx, maps.MapStrAny{
			"team_id":     teamID,
			"user_id":     memberUser,
			"member_type": "user",
			"role_id":     "user",
			"status":      "active",
		})
		assert.NoError(t, err)

		// Test update with empty data (should not error, just do nothing)
		err = testProvider.UpdateMember(ctx, teamID, memberUser, maps.MapStrAny{})
		assert.NoError(t, err)

		// Test update with only sensitive fields (should not error, just ignore them)
		err = testProvider.UpdateMember(ctx, teamID, memberUser, maps.MapStrAny{
			"id":      999,
			"team_id": "new-team",
		})
		assert.NoError(t, err)
	})
}

func TestMemberInvitationExpiry(t *testing.T) {
	prepare(t)
	defer clean()

	ctx := context.Background()

	// Use UUID to ensure unique identifiers
	testUUID := strings.ReplaceAll(uuid.New().String(), "-", "")[:8]

	// Create test users
	ownerUser := createTestUser(ctx, t, "owner"+testUUID)
	inviteeUser := createTestUser(ctx, t, "invitee"+testUUID)

	// Create test team
	teamMap := maps.MapStrAny{
		"name":         "Expiry Test Team " + testUUID,
		"display_name": "Expiry Test " + testUUID,
		"description":  "A test team for invitation expiry testing",
		"owner_id":     ownerUser,
		"status":       "active",
	}

	teamID, err := testProvider.CreateTeam(ctx, teamMap)
	assert.NoError(t, err)

	// Create member with expired invitation
	expiredTime := time.Now().Add(-2 * time.Hour) // Expired 2 hours ago to be safe
	memberData := maps.MapStrAny{
		"team_id":               teamID,
		"user_id":               inviteeUser,
		"member_type":           "user",
		"role_id":               "user",
		"status":                "pending",
		"invited_by":            ownerUser,
		"invited_at":            expiredTime.Add(-1 * time.Hour), // Invited 3 hours ago
		"invitation_token":      "expired-token-" + testUUID,
		"invitation_expires_at": expiredTime, // Expired 2 hours ago
	}

	businessMemberID, err := testProvider.CreateMember(ctx, memberData)
	assert.NoError(t, err)

	// Get the invitation_id
	member, err := testProvider.GetMemberByMemberID(ctx, businessMemberID)
	assert.NoError(t, err)
	invitationID := member["invitation_id"].(string)
	assert.NotEmpty(t, invitationID)

	// Test AcceptInvitation with expired token
	t.Run("AcceptInvitation_ExpiredToken", func(t *testing.T) {
		err := testProvider.AcceptInvitation(ctx, invitationID, "expired-token-"+testUUID, "")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invitation has expired")
	})
}

func TestMemberInvitationIDOperations(t *testing.T) {
	prepare(t)
	defer clean()

	ctx := context.Background()

	// Use UUID to ensure unique identifiers
	testUUID := strings.ReplaceAll(uuid.New().String(), "-", "")[:8]

	// Create test users
	ownerUser := createTestUser(ctx, t, "owner"+testUUID)
	inviteeUser := createTestUser(ctx, t, "invitee"+testUUID)

	// Create test team
	teamMap := maps.MapStrAny{
		"name":         "Invitation ID Test Team " + testUUID,
		"display_name": "Invitation ID Test " + testUUID,
		"description":  "A test team for invitation_id testing",
		"owner_id":     ownerUser,
		"status":       "active",
		"type":         "corporation",
		"type_id":      "business",
		"metadata":     map[string]interface{}{"test": true},
	}

	teamID, err := testProvider.CreateTeam(ctx, teamMap)
	assert.NoError(t, err)

	var invitationID string

	// Test CreateMember with pending status (should generate invitation_id)
	t.Run("CreateMember_GeneratesInvitationID", func(t *testing.T) {
		memberData := maps.MapStrAny{
			"team_id":     teamID,
			"user_id":     nil, // Simulate invitation to unregistered user
			"member_type": "user",
			"role_id":     "user",
			"status":      "pending",
			"invited_by":  ownerUser,
		}

		businessMemberID, err := testProvider.CreateMember(ctx, memberData)
		assert.NoError(t, err)
		assert.NotEmpty(t, businessMemberID)

		// Get the created member to verify invitation_id was generated
		member, err := testProvider.GetMemberByMemberID(ctx, businessMemberID)
		assert.NoError(t, err)
		assert.NotNil(t, member["invitation_id"])
		assert.NotEmpty(t, member["invitation_id"])

		invitationID = member["invitation_id"].(string)
		t.Logf("Generated invitation_id: %s", invitationID)
		assert.True(t, strings.Contains(invitationID, "inv_"), "invitation_id should contain inv_ prefix, got: "+invitationID)
	})

	// Test GetMemberByInvitationID
	t.Run("GetMemberByInvitationID", func(t *testing.T) {
		member, err := testProvider.GetMemberByInvitationID(ctx, invitationID)
		assert.NoError(t, err)
		assert.NotNil(t, member)
		assert.Equal(t, invitationID, member["invitation_id"])
		assert.Equal(t, teamID, member["team_id"])
		assert.Equal(t, "pending", member["status"])
		assert.Equal(t, ownerUser, member["invited_by"])
	})

	// Test GetMemberByInvitationID with non-existent invitation
	t.Run("GetMemberByInvitationID_NotFound", func(t *testing.T) {
		_, err := testProvider.GetMemberByInvitationID(ctx, "non-existent-invitation-id")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "member not found")
	})

	// Test UpdateMemberByInvitationID
	t.Run("UpdateMemberByInvitationID", func(t *testing.T) {
		updateData := maps.MapStrAny{
			"user_id":   inviteeUser, // Now associate with a user
			"status":    "active",
			"joined_at": time.Now(),
			"notes":     "Invitation accepted",
		}

		err := testProvider.UpdateMemberByInvitationID(ctx, invitationID, updateData)
		assert.NoError(t, err)

		// Verify update
		member, err := testProvider.GetMemberByInvitationID(ctx, invitationID)
		assert.NoError(t, err)
		assert.Equal(t, inviteeUser, member["user_id"])
		assert.Equal(t, "active", member["status"])
		assert.NotNil(t, member["joined_at"])

		// Test updating sensitive fields (should be ignored except user_id which is allowed)
		sensitiveData := maps.MapStrAny{
			"id":            999,
			"team_id":       "new-team",
			"invitation_id": "new-invitation-id",
		}

		err = testProvider.UpdateMemberByInvitationID(ctx, invitationID, sensitiveData)
		assert.NoError(t, err) // Should not error, just ignore sensitive fields

		// Verify sensitive fields were not updated
		member, err = testProvider.GetMemberByInvitationID(ctx, invitationID)
		assert.NoError(t, err)
		assert.Equal(t, invitationID, member["invitation_id"]) // Should remain unchanged
		assert.Equal(t, teamID, member["team_id"])             // Should remain unchanged
		assert.Equal(t, inviteeUser, member["user_id"])        // Should remain as updated value
	})

	// Test UpdateMemberByInvitationID with non-existent invitation
	t.Run("UpdateMemberByInvitationID_NotFound", func(t *testing.T) {
		updateData := maps.MapStrAny{"notes": "test"}
		err := testProvider.UpdateMemberByInvitationID(ctx, "non-existent-invitation-id", updateData)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "member not found")
	})

	// Test UpdateMemberByInvitationID with empty data (should not error)
	t.Run("UpdateMemberByInvitationID_EmptyData", func(t *testing.T) {
		err := testProvider.UpdateMemberByInvitationID(ctx, invitationID, maps.MapStrAny{})
		assert.NoError(t, err) // Should not error, just do nothing
	})

	// Test RemoveMemberByInvitationID (at the end)
	t.Run("RemoveMemberByInvitationID", func(t *testing.T) {
		err := testProvider.RemoveMemberByInvitationID(ctx, invitationID)
		assert.NoError(t, err)

		// Verify member was removed
		_, err = testProvider.GetMemberByInvitationID(ctx, invitationID)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "member not found")
	})

	// Test RemoveMemberByInvitationID with non-existent invitation
	t.Run("RemoveMemberByInvitationID_NotFound", func(t *testing.T) {
		err := testProvider.RemoveMemberByInvitationID(ctx, "non-existent-invitation-id")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "member not found")
	})
}

func TestCreateMemberInvitationIDGeneration(t *testing.T) {
	prepare(t)
	defer clean()

	ctx := context.Background()

	// Use UUID to ensure unique identifiers
	testUUID := strings.ReplaceAll(uuid.New().String(), "-", "")[:8]

	// Create test user
	ownerUser := createTestUser(ctx, t, "owner"+testUUID)

	// Create test team
	teamMap := maps.MapStrAny{
		"name":         "ID Generation Test Team " + testUUID,
		"display_name": "ID Generation Test " + testUUID,
		"description":  "A test team for invitation_id generation testing",
		"owner_id":     ownerUser,
		"status":       "active",
	}

	teamID, err := testProvider.CreateTeam(ctx, teamMap)
	assert.NoError(t, err)

	// Test invitation_id generation for pending members
	t.Run("CreateMember_PendingStatus_GeneratesInvitationID", func(t *testing.T) {
		memberData := maps.MapStrAny{
			"team_id":     teamID,
			"user_id":     nil, // No user_id for pending invitation
			"member_type": "user",
			"role_id":     "user",
			"status":      "pending",
			"invited_by":  ownerUser,
		}

		businessMemberID, err := testProvider.CreateMember(ctx, memberData)
		assert.NoError(t, err)

		// Get the created member
		member, err := testProvider.GetMemberByMemberID(ctx, businessMemberID)
		assert.NoError(t, err)

		// Verify invitation_id was generated
		assert.NotNil(t, member["invitation_id"])
		assert.NotEmpty(t, member["invitation_id"])

		invitationID := member["invitation_id"].(string)
		t.Logf("Generated invitation_id: %s", invitationID)
		assert.True(t, strings.Contains(invitationID, "inv_"), "invitation_id should contain inv_ prefix, got: "+invitationID)
		assert.True(t, len(invitationID) > 4, "invitation_id should be longer than just the prefix")
	})

	// Test that active members don't get invitation_id
	t.Run("CreateMember_ActiveStatus_NoInvitationID", func(t *testing.T) {
		activeUser := createTestUser(ctx, t, "active"+testUUID)

		memberData := maps.MapStrAny{
			"team_id":     teamID,
			"user_id":     activeUser,
			"member_type": "user",
			"role_id":     "user",
			"status":      "active",
		}

		businessMemberID, err := testProvider.CreateMember(ctx, memberData)
		assert.NoError(t, err)

		// Get the created member
		member, err := testProvider.GetMemberByMemberID(ctx, businessMemberID)
		assert.NoError(t, err)

		// Verify invitation_id is nil for active members
		assert.Nil(t, member["invitation_id"])
	})

	// Test explicit invitation_id is preserved
	t.Run("CreateMember_ExplicitInvitationID_Preserved", func(t *testing.T) {
		explicitInvitationID := "inv_explicit_test_" + testUUID

		memberData := maps.MapStrAny{
			"team_id":       teamID,
			"user_id":       nil,
			"member_type":   "user",
			"role_id":       "user",
			"status":        "pending",
			"invited_by":    ownerUser,
			"invitation_id": explicitInvitationID,
		}

		businessMemberID, err := testProvider.CreateMember(ctx, memberData)
		assert.NoError(t, err)

		// Get the created member
		member, err := testProvider.GetMemberByMemberID(ctx, businessMemberID)
		assert.NoError(t, err)

		// Verify explicit invitation_id was preserved
		assert.Equal(t, explicitInvitationID, member["invitation_id"])
	})
}

func TestMemberIDOperations(t *testing.T) {
	prepare(t)
	defer clean()

	ctx := context.Background()

	// Use UUID to ensure unique identifiers
	testUUID := strings.ReplaceAll(uuid.New().String(), "-", "")[:8]

	// Create test users
	ownerUser := createTestUser(ctx, t, "owner"+testUUID)
	memberUser := createTestUser(ctx, t, "member"+testUUID)

	// Create test team
	teamMap := maps.MapStrAny{
		"name":         "Member ID Test Team " + testUUID,
		"display_name": "Member ID Test " + testUUID,
		"description":  "A test team for member_id testing",
		"owner_id":     ownerUser,
		"status":       "active",
		"type":         "corporation",
		"type_id":      "business",
		"metadata":     map[string]interface{}{"test": true},
	}

	teamID, err := testProvider.CreateTeam(ctx, teamMap)
	assert.NoError(t, err)

	var businessMemberID string

	// Test CreateMember generates member_id
	t.Run("CreateMember_GeneratesMemberID", func(t *testing.T) {
		memberData := maps.MapStrAny{
			"team_id":     teamID,
			"user_id":     memberUser,
			"member_type": "user",
			"role_id":     "user",
			"status":      "active",
		}

		_, err := testProvider.CreateMember(ctx, memberData)
		assert.NoError(t, err)

		// Get the created member to verify member_id was generated
		member, err := testProvider.GetMember(ctx, teamID, memberUser)
		assert.NoError(t, err)
		assert.NotNil(t, member["member_id"])
		assert.NotEmpty(t, member["member_id"])

		businessMemberID = member["member_id"].(string)
		t.Logf("Generated member_id: %s", businessMemberID)
	})

	// Test GetMemberByMemberID
	t.Run("GetMemberByMemberID", func(t *testing.T) {
		member, err := testProvider.GetMemberByMemberID(ctx, businessMemberID)
		assert.NoError(t, err)
		assert.NotNil(t, member)
		assert.Equal(t, businessMemberID, member["member_id"])
		assert.Equal(t, teamID, member["team_id"])
		assert.Equal(t, memberUser, member["user_id"])
	})

	// Test GetMemberDetailByMemberID
	t.Run("GetMemberDetailByMemberID", func(t *testing.T) {
		member, err := testProvider.GetMemberDetailByMemberID(ctx, businessMemberID)
		assert.NoError(t, err)
		assert.NotNil(t, member)
		assert.Equal(t, businessMemberID, member["member_id"])
		assert.Equal(t, teamID, member["team_id"])
		// Should contain detailed fields
		assert.Contains(t, member, "created_at")
		assert.Contains(t, member, "updated_at")
	})

	// Test UpdateMemberByMemberID
	t.Run("UpdateMemberByMemberID", func(t *testing.T) {
		updateData := maps.MapStrAny{
			"role_id": "admin",
			"notes":   "Updated via member_id",
		}

		err := testProvider.UpdateMemberByMemberID(ctx, businessMemberID, updateData)
		assert.NoError(t, err)

		// Verify update
		member, err := testProvider.GetMemberByMemberID(ctx, businessMemberID)
		assert.NoError(t, err)
		assert.Equal(t, "admin", member["role_id"])
	})

	// Test UpdateMemberRoleByMemberID
	t.Run("UpdateMemberRoleByMemberID", func(t *testing.T) {
		err := testProvider.UpdateMemberRoleByMemberID(ctx, businessMemberID, "moderator")
		assert.NoError(t, err)

		// Verify role was updated
		member, err := testProvider.GetMemberByMemberID(ctx, businessMemberID)
		assert.NoError(t, err)
		assert.Equal(t, "moderator", member["role_id"])
	})

	// Test UpdateMemberStatusByMemberID
	t.Run("UpdateMemberStatusByMemberID", func(t *testing.T) {
		err := testProvider.UpdateMemberStatusByMemberID(ctx, businessMemberID, "suspended")
		assert.NoError(t, err)

		// Verify status was updated
		member, err := testProvider.GetMemberByMemberID(ctx, businessMemberID)
		assert.NoError(t, err)
		assert.Equal(t, "suspended", member["status"])

		// Change back to active
		err = testProvider.UpdateMemberStatusByMemberID(ctx, businessMemberID, "active")
		assert.NoError(t, err)
	})

	// Test UpdateMemberLastActivityByMemberID
	t.Run("UpdateMemberLastActivityByMemberID", func(t *testing.T) {
		err := testProvider.UpdateMemberLastActivityByMemberID(ctx, businessMemberID)
		assert.NoError(t, err)

		// Verify last_active_at was updated and login_count incremented
		member, err := testProvider.GetMemberByMemberID(ctx, businessMemberID)
		assert.NoError(t, err)
		assert.NotNil(t, member["last_active_at"])
		// login_count should be at least 1
		loginCount := member["login_count"]
		if loginCount != nil {
			switch v := loginCount.(type) {
			case int:
				assert.True(t, v >= 1, "login_count should be at least 1")
			case int64:
				assert.True(t, v >= 1, "login_count should be at least 1")
			case int32:
				assert.True(t, v >= 1, "login_count should be at least 1")
			default:
				t.Logf("Unexpected login_count type: %T, value: %v", loginCount, loginCount)
				assert.True(t, false, "login_count should be a numeric type")
			}
		} else {
			assert.True(t, false, "login_count should not be nil")
		}
	})

	// Test RemoveMemberByMemberID (at the end)
	t.Run("RemoveMemberByMemberID", func(t *testing.T) {
		err := testProvider.RemoveMemberByMemberID(ctx, businessMemberID)
		assert.NoError(t, err)

		// Verify member was removed
		_, err = testProvider.GetMemberByMemberID(ctx, businessMemberID)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "member not found")
	})
}

func TestMemberExistsByRobotEmail(t *testing.T) {
	prepare(t)
	defer clean()

	ctx := context.Background()

	// Use UUID to ensure unique identifiers
	testUUID := strings.ReplaceAll(uuid.New().String(), "-", "")[:8]

	// Create test users
	ownerUser := createTestUser(ctx, t, "owner"+testUUID)

	// Create test team
	teamMap := maps.MapStrAny{
		"name":         "Robot Email Test Team " + testUUID,
		"display_name": "Robot Email Test " + testUUID,
		"description":  "A test team for robot email testing",
		"owner_id":     ownerUser,
		"status":       "active",
	}

	teamID, err := testProvider.CreateTeam(ctx, teamMap)
	assert.NoError(t, err)

	testRobotEmail := "testrobot" + testUUID + "@robot.example.com"

	// Create robot member with robot_email
	t.Run("CreateRobotMemberWithRobotEmail", func(t *testing.T) {
		robotData := maps.MapStrAny{
			"display_name": "TestBot" + testUUID,
			"robot_email":  testRobotEmail,
			"role_id":      "bot",
		}

		businessMemberID, err := testProvider.CreateRobotMember(ctx, teamID, robotData)
		assert.NoError(t, err)
		assert.NotEmpty(t, businessMemberID)
	})

	// Test MemberExistsByRobotEmail
	t.Run("MemberExistsByRobotEmail_Exists", func(t *testing.T) {
		exists, err := testProvider.MemberExistsByRobotEmail(ctx, testRobotEmail)
		assert.NoError(t, err)
		assert.True(t, exists)
	})

	// Test with non-existent robot email
	t.Run("MemberExistsByRobotEmail_NotExists", func(t *testing.T) {
		exists, err := testProvider.MemberExistsByRobotEmail(ctx, "nonexistent@robot.example.com")
		assert.NoError(t, err)
		assert.False(t, exists)
	})
}

func TestUpdateRobotMember(t *testing.T) {
	prepare(t)
	defer clean()

	ctx := context.Background()

	// Use UUID to ensure unique identifiers
	testUUID := strings.ReplaceAll(uuid.New().String(), "-", "")[:8]

	// Create test user (team owner)
	ownerUser := createTestUser(ctx, t, "owner"+testUUID)
	regularUser := createTestUser(ctx, t, "user"+testUUID)

	// Create test team
	teamMap := maps.MapStrAny{
		"name":         "Update Robot Test Team " + testUUID,
		"display_name": "Update Robot Test " + testUUID,
		"description":  "A test team for robot update testing",
		"owner_id":     ownerUser,
		"status":       "active",
	}

	teamID, err := testProvider.CreateTeam(ctx, teamMap)
	assert.NoError(t, err)

	var robotMemberID string
	var regularMemberID string

	// Create a robot member
	t.Run("Setup_CreateRobotMember", func(t *testing.T) {
		robotData := maps.MapStrAny{
			"display_name":    "TestBot" + testUUID,
			"bio":             "Original bio",
			"role_id":         "bot",
			"autonomous_mode": false,
			"robot_status":    "idle",
			"system_prompt":   "Original prompt",
			"language_model":  "gpt-3.5-turbo",
			"cost_limit":      50.00,
			"robot_email":     "testbot" + testUUID + "@robot.example.com",
		}

		memberID, err := testProvider.CreateRobotMember(ctx, teamID, robotData)
		assert.NoError(t, err)
		assert.NotEmpty(t, memberID)
		robotMemberID = memberID
	})

	// Create a regular user member for testing
	t.Run("Setup_CreateRegularMember", func(t *testing.T) {
		memberData := maps.MapStrAny{
			"team_id":     teamID,
			"user_id":     regularUser,
			"member_type": "user",
			"role_id":     "user",
			"status":      "active",
		}

		memberID, err := testProvider.CreateMember(ctx, memberData)
		assert.NoError(t, err)
		assert.NotEmpty(t, memberID)
		regularMemberID = memberID
	})

	// Test successful update of robot member
	t.Run("UpdateRobotMember_Success", func(t *testing.T) {
		updateData := maps.MapStrAny{
			"display_name":    "UpdatedBot" + testUUID,
			"bio":             "Updated bio",
			"autonomous_mode": true,
			"robot_status":    "working",
			"system_prompt":   "Updated prompt",
			"language_model":  "gpt-4",
			"cost_limit":      100.00,
		}

		err := testProvider.UpdateRobotMember(ctx, robotMemberID, updateData)
		assert.NoError(t, err)

		// Verify updates
		member, err := testProvider.GetMemberDetailByMemberID(ctx, robotMemberID)
		assert.NoError(t, err)
		assert.Equal(t, "UpdatedBot"+testUUID, member["display_name"])
		assert.Equal(t, "Updated bio", member["bio"])
		assert.Equal(t, "working", member["robot_status"])
		assert.Equal(t, "Updated prompt", member["system_prompt"])
		assert.Equal(t, "gpt-4", member["language_model"])
	})

	// Test updating robot_email
	t.Run("UpdateRobotMember_UpdateEmail", func(t *testing.T) {
		newEmail := "updated-testbot" + testUUID + "@robot.example.com"
		updateData := maps.MapStrAny{
			"robot_email": newEmail,
		}

		err := testProvider.UpdateRobotMember(ctx, robotMemberID, updateData)
		assert.NoError(t, err)

		// Verify update
		member, err := testProvider.GetMemberDetailByMemberID(ctx, robotMemberID)
		assert.NoError(t, err)
		assert.Equal(t, newEmail, member["robot_email"])
	})

	// Test updating robot configuration fields
	t.Run("UpdateRobotMember_UpdateConfiguration", func(t *testing.T) {
		updateData := maps.MapStrAny{
			"authorized_senders": []string{
				"admin@example.com",
				"manager@example.com",
			},
			"email_filter_rules": []string{
				".*@example\\.com$",
				".*@test\\.com$",
			},
			"robot_config": map[string]interface{}{
				"max_tokens":  2000,
				"temperature": 0.7,
			},
			"agents": []string{
				"agent1",
				"agent2",
			},
			"mcp_servers": []string{
				"mcp://server1",
				"mcp://server2",
			},
		}

		err := testProvider.UpdateRobotMember(ctx, robotMemberID, updateData)
		assert.NoError(t, err)

		// Verify update
		member, err := testProvider.GetMemberDetailByMemberID(ctx, robotMemberID)
		assert.NoError(t, err)
		assert.NotNil(t, member["authorized_senders"])
		assert.NotNil(t, member["email_filter_rules"])
		assert.NotNil(t, member["robot_config"])
	})

	// Test error when trying to update non-robot member
	t.Run("UpdateRobotMember_NotRobotMember", func(t *testing.T) {
		updateData := maps.MapStrAny{
			"display_name": "Should Fail",
		}

		err := testProvider.UpdateRobotMember(ctx, regularMemberID, updateData)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not a robot member")
	})

	// Test error when member_id doesn't exist
	t.Run("UpdateRobotMember_NotFound", func(t *testing.T) {
		updateData := maps.MapStrAny{
			"display_name": "Should Fail",
		}

		err := testProvider.UpdateRobotMember(ctx, "non-existent-member-id", updateData)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to get member")
	})

	// Test robot_email uniqueness validation during update
	t.Run("UpdateRobotMember_EmailUniqueness", func(t *testing.T) {
		// Create another robot with a different email
		anotherRobotData := maps.MapStrAny{
			"display_name": "AnotherBot" + testUUID,
			"role_id":      "bot",
			"robot_email":  "anotherbot" + testUUID + "@robot.example.com",
		}

		_, err := testProvider.CreateRobotMember(ctx, teamID, anotherRobotData)
		assert.NoError(t, err)

		// Try to update the first robot's email to match the second robot's email
		updateData := maps.MapStrAny{
			"robot_email": "anotherbot" + testUUID + "@robot.example.com",
		}

		err = testProvider.UpdateRobotMember(ctx, robotMemberID, updateData)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "already exists")
	})

	// Test updating with same email (should succeed - no actual change)
	t.Run("UpdateRobotMember_SameEmail", func(t *testing.T) {
		// Get current email
		member, err := testProvider.GetMemberDetailByMemberID(ctx, robotMemberID)
		assert.NoError(t, err)
		currentEmail := member["robot_email"]

		// Update with same email
		updateData := maps.MapStrAny{
			"robot_email": currentEmail,
		}

		err = testProvider.UpdateRobotMember(ctx, robotMemberID, updateData)
		assert.NoError(t, err)
	})

	// Test update with empty data (should not error)
	t.Run("UpdateRobotMember_EmptyData", func(t *testing.T) {
		err := testProvider.UpdateRobotMember(ctx, robotMemberID, maps.MapStrAny{})
		assert.NoError(t, err) // Should not error, just do nothing
	})

	// Test updating status
	t.Run("UpdateRobotMember_UpdateStatus", func(t *testing.T) {
		updateData := maps.MapStrAny{
			"status": "inactive",
		}

		err := testProvider.UpdateRobotMember(ctx, robotMemberID, updateData)
		assert.NoError(t, err)

		// Verify update
		member, err := testProvider.GetMemberByMemberID(ctx, robotMemberID)
		assert.NoError(t, err)
		assert.Equal(t, "inactive", member["status"])

		// Restore to active
		err = testProvider.UpdateRobotMember(ctx, robotMemberID, maps.MapStrAny{"status": "active"})
		assert.NoError(t, err)
	})
}

func TestRobotEmailUniqueness(t *testing.T) {
	prepare(t)
	defer clean()

	ctx := context.Background()

	// Use UUID to ensure unique identifiers
	testUUID := strings.ReplaceAll(uuid.New().String(), "-", "")[:8]

	// Create test user (team owner)
	ownerUser := createTestUser(ctx, t, "owner"+testUUID)

	// Create two test teams
	team1Map := maps.MapStrAny{
		"name":         "Robot Email Test Team 1 " + testUUID,
		"display_name": "Robot Email Test 1 " + testUUID,
		"description":  "First test team for robot email testing",
		"owner_id":     ownerUser,
		"status":       "active",
	}

	team1ID, err := testProvider.CreateTeam(ctx, team1Map)
	assert.NoError(t, err)

	team2Map := maps.MapStrAny{
		"name":         "Robot Email Test Team 2 " + testUUID,
		"display_name": "Robot Email Test 2 " + testUUID,
		"description":  "Second test team for robot email testing",
		"owner_id":     ownerUser,
		"status":       "active",
	}

	team2ID, err := testProvider.CreateTeam(ctx, team2Map)
	assert.NoError(t, err)

	testEmail := "unique-robot" + testUUID + "@robot.example.com"

	// Test creating first robot with robot_email
	t.Run("CreateFirstRobotWithEmail", func(t *testing.T) {
		robotData := maps.MapStrAny{
			"display_name": "Robot1" + testUUID,
			"role_id":      "bot",
			"robot_email":  testEmail,
			"authorized_senders": []string{
				"admin@example.com",
			},
			"email_filter_rules": []string{
				".*@example\\.com$",
			},
		}

		memberID, err := testProvider.CreateRobotMember(ctx, team1ID, robotData)
		assert.NoError(t, err)
		assert.NotEmpty(t, memberID)

		// Verify robot_email was set
		member, err := testProvider.GetMemberDetailByMemberID(ctx, memberID)
		assert.NoError(t, err)
		assert.Equal(t, testEmail, member["robot_email"])
	})

	// Test creating second robot with same robot_email should fail (global uniqueness)
	t.Run("CreateSecondRobotWithSameEmail_ShouldFail", func(t *testing.T) {
		robotData := maps.MapStrAny{
			"display_name": "Robot2" + testUUID,
			"role_id":      "bot",
			"robot_email":  testEmail, // Same email as first robot
		}

		_, err := testProvider.CreateRobotMember(ctx, team2ID, robotData)
		assert.Error(t, err)
		// The error should indicate uniqueness constraint violation
		// Note: The exact error message may vary depending on the database driver
	})

	// Test creating robot with different robot_email should succeed
	t.Run("CreateRobotWithDifferentEmail_ShouldSucceed", func(t *testing.T) {
		differentEmail := "another-robot" + testUUID + "@robot.example.com"
		robotData := maps.MapStrAny{
			"display_name": "Robot3" + testUUID,
			"role_id":      "bot",
			"robot_email":  differentEmail,
		}

		memberID, err := testProvider.CreateRobotMember(ctx, team2ID, robotData)
		assert.NoError(t, err)
		assert.NotEmpty(t, memberID)

		// Verify robot_email was set
		member, err := testProvider.GetMemberDetailByMemberID(ctx, memberID)
		assert.NoError(t, err)
		assert.Equal(t, differentEmail, member["robot_email"])
	})

	// Test updating robot_email
	t.Run("UpdateRobotEmail", func(t *testing.T) {
		// Create a new robot
		newEmail := "updatable-robot" + testUUID + "@robot.example.com"
		robotData := maps.MapStrAny{
			"display_name": "Robot4" + testUUID,
			"role_id":      "bot",
			"robot_email":  newEmail,
		}

		memberID, err := testProvider.CreateRobotMember(ctx, team1ID, robotData)
		assert.NoError(t, err)

		// Update robot_email
		updatedEmail := "updated-robot" + testUUID + "@robot.example.com"
		updateData := maps.MapStrAny{
			"robot_email": updatedEmail,
		}

		err = testProvider.UpdateMemberByMemberID(ctx, memberID, updateData)
		assert.NoError(t, err)

		// Verify update
		member, err := testProvider.GetMemberDetailByMemberID(ctx, memberID)
		assert.NoError(t, err)
		assert.Equal(t, updatedEmail, member["robot_email"])
	})

	// Test updating authorized_senders and email_filter_rules
	t.Run("UpdateRobotEmailConfiguration", func(t *testing.T) {
		// Create a new robot
		robotData := maps.MapStrAny{
			"display_name": "Robot5" + testUUID,
			"role_id":      "bot",
			"robot_email":  "config-robot" + testUUID + "@robot.example.com",
			"authorized_senders": []string{
				"initial@example.com",
			},
			"email_filter_rules": []string{
				".*@initial\\.com$",
			},
		}

		memberID, err := testProvider.CreateRobotMember(ctx, team1ID, robotData)
		assert.NoError(t, err)

		// Update email configuration
		updateData := maps.MapStrAny{
			"authorized_senders": []string{
				"admin@example.com",
				"manager@example.com",
				"owner@example.com",
			},
			"email_filter_rules": []string{
				".*@example\\.com$",
				".*@test\\.com$",
				".*@company\\.com$",
			},
		}

		err = testProvider.UpdateMemberByMemberID(ctx, memberID, updateData)
		assert.NoError(t, err)

		// Verify update
		member, err := testProvider.GetMemberDetailByMemberID(ctx, memberID)
		assert.NoError(t, err)
		assert.NotNil(t, member["authorized_senders"])
		assert.NotNil(t, member["email_filter_rules"])
	})
}

// Helper function createTestUser is defined in team_test.go
