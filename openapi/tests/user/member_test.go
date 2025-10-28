package user_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/yaoapp/kun/maps"
	"github.com/yaoapp/yao/openapi"
	"github.com/yaoapp/yao/openapi/tests/testutils"
)

// TestMemberList tests the GET /user/teams/:team_id/members endpoint
func TestMemberList(t *testing.T) {
	// Initialize test environment
	serverURL := testutils.Prepare(t)
	defer testutils.Clean()

	// Get base URL from server config
	baseURL := ""
	if openapi.Server != nil && openapi.Server.Config != nil {
		baseURL = openapi.Server.Config.BaseURL
	}

	// Register a test client for OAuth authentication
	testClient := testutils.RegisterTestClient(t, "Member List Test Client", []string{"https://localhost/callback"})
	defer testutils.CleanupTestClient(t, testClient.ClientID)

	// Obtain access token for authenticated requests
	tokenInfo := testutils.ObtainAccessToken(t, serverURL, testClient.ClientID, testClient.ClientSecret, "https://localhost/callback", "openid profile")

	// Create a test team first
	createdTeam := createTestTeam(t, serverURL, baseURL, tokenInfo.AccessToken, "Member List Test Team")
	teamID := getTeamID(createdTeam)

	// Create some test members and robots for filtering tests
	testUUID := strings.ReplaceAll(uuid.New().String(), "-", "")[:8]

	// Create a robot member for member_type filtering
	robotBody := map[string]interface{}{
		"name":   "Test Robot " + testUUID,
		"email":  fmt.Sprintf("test-robot-%s@test.com", testUUID),
		"role":   "member",
		"prompt": "You are a test robot for filtering",
	}
	robotBodyBytes, _ := json.Marshal(robotBody)
	robotReq, _ := http.NewRequest("POST", serverURL+baseURL+"/user/teams/"+teamID+"/members/robots", bytes.NewBuffer(robotBodyBytes))
	robotReq.Header.Set("Content-Type", "application/json")
	robotReq.Header.Set("Authorization", "Bearer "+tokenInfo.AccessToken)
	client := &http.Client{}
	robotResp, err := client.Do(robotReq)
	if err == nil && robotResp != nil {
		robotResp.Body.Close()
		if robotResp.StatusCode != 201 {
			t.Logf("Warning: Failed to create robot member for testing (status=%d)", robotResp.StatusCode)
		}
	}

	testCases := []struct {
		name       string
		teamID     string
		query      string
		headers    map[string]string
		expectCode int
		expectMsg  string
		validateFn func(*testing.T, map[string]interface{}) // Optional validation function
	}{
		{
			"list members without authentication",
			teamID,
			"",
			map[string]string{},
			401,
			"should require authentication",
			nil,
		},
		{
			"list members with valid token",
			teamID,
			"",
			map[string]string{
				"Authorization": "Bearer " + tokenInfo.AccessToken,
			},
			200,
			"should return team members with default sorting (is_owner desc, status desc, created_at desc)",
			func(t *testing.T, response map[string]interface{}) {
				// Verify default sorting: is_owner desc first, then status desc
				if data, ok := response["data"].([]interface{}); ok && len(data) > 1 {
					foundNonOwner := false
					foundActive := false

					for _, item := range data {
						member := item.(map[string]interface{})

						// Check is_owner sorting (owners first)
						isOwner := false
						if ownerVal, ok := member["is_owner"]; ok {
							switch v := ownerVal.(type) {
							case float64:
								isOwner = v == 1
							case int:
								isOwner = v == 1
							case bool:
								isOwner = v
							}
						}

						if isOwner {
							assert.False(t, foundNonOwner, "Owners should come before non-owners")
						} else {
							foundNonOwner = true
						}

						// Check status sorting (pending before active) among non-owners
						if !isOwner {
							status := member["status"].(string)
							if status == "pending" {
								assert.False(t, foundActive, "Pending members should come before active members")
							} else if status == "active" {
								foundActive = true
							}
						}
					}
				}
			},
		},
		{
			"list members with pagination",
			teamID,
			"?page=1&pagesize=10",
			map[string]string{
				"Authorization": "Bearer " + tokenInfo.AccessToken,
			},
			200,
			"should handle pagination parameters",
			func(t *testing.T, response map[string]interface{}) {
				assert.Equal(t, float64(1), response["page"], "Should have correct page number")
				assert.Equal(t, float64(10), response["pagesize"], "Should have correct pagesize")
			},
		},
		{
			"list members with status filter",
			teamID,
			"?status=active",
			map[string]string{
				"Authorization": "Bearer " + tokenInfo.AccessToken,
			},
			200,
			"should filter by status",
			func(t *testing.T, response map[string]interface{}) {
				if data, ok := response["data"].([]interface{}); ok {
					for _, item := range data {
						member := item.(map[string]interface{})
						assert.Equal(t, "active", member["status"], "All members should have active status")
					}
				}
			},
		},
		{
			"list members filtered by member_type user",
			teamID,
			"?member_type=user",
			map[string]string{
				"Authorization": "Bearer " + tokenInfo.AccessToken,
			},
			200,
			"should filter by member_type=user",
			func(t *testing.T, response map[string]interface{}) {
				if data, ok := response["data"].([]interface{}); ok {
					for _, item := range data {
						member := item.(map[string]interface{})
						assert.Equal(t, "user", member["member_type"], "All members should be user type")
					}
				}
			},
		},
		{
			"list members filtered by member_type robot",
			teamID,
			"?member_type=robot",
			map[string]string{
				"Authorization": "Bearer " + tokenInfo.AccessToken,
			},
			200,
			"should filter by member_type=robot",
			func(t *testing.T, response map[string]interface{}) {
				if data, ok := response["data"].([]interface{}); ok {
					for _, item := range data {
						member := item.(map[string]interface{})
						assert.Equal(t, "robot", member["member_type"], "All members should be robot type")
					}
				}
			},
		},
		{
			"list members filtered by role_id",
			teamID,
			"?role_id=owner:free",
			map[string]string{
				"Authorization": "Bearer " + tokenInfo.AccessToken,
			},
			200,
			"should filter by role_id",
			func(t *testing.T, response map[string]interface{}) {
				if data, ok := response["data"].([]interface{}); ok {
					for _, item := range data {
						member := item.(map[string]interface{})
						assert.Equal(t, "owner:free", member["role_id"], "All members should have owner:free role")
					}
				}
			},
		},
		{
			"list members with order by created_at asc",
			teamID,
			"?order=created_at+asc",
			map[string]string{
				"Authorization": "Bearer " + tokenInfo.AccessToken,
			},
			200,
			"should sort by is_owner desc, status desc, then created_at ascending",
			func(t *testing.T, response map[string]interface{}) {
				// Verify owner and status sorting priority
				if data, ok := response["data"].([]interface{}); ok && len(data) > 1 {
					foundNonOwner := false
					for _, item := range data {
						member := item.(map[string]interface{})

						isOwner := false
						if ownerVal, ok := member["is_owner"]; ok {
							switch v := ownerVal.(type) {
							case float64:
								isOwner = v == 1
							case int:
								isOwner = v == 1
							case bool:
								isOwner = v
							}
						}

						if isOwner {
							assert.False(t, foundNonOwner, "Owners should come before non-owners even with custom sorting")
						} else {
							foundNonOwner = true
						}
					}
				}
			},
		},
		{
			"list members with order by joined_at desc",
			teamID,
			"?order=joined_at+desc",
			map[string]string{
				"Authorization": "Bearer " + tokenInfo.AccessToken,
			},
			200,
			"should sort by is_owner desc, status desc, then joined_at descending",
			nil,
		},
		{
			"list members with order by joined_at (default desc)",
			teamID,
			"?order=joined_at",
			map[string]string{
				"Authorization": "Bearer " + tokenInfo.AccessToken,
			},
			200,
			"should sort by is_owner desc, status desc, then joined_at with default desc direction",
			nil,
		},
		{
			"list members with field selection",
			teamID,
			"?fields=id,user_id,member_type,role_id,status",
			map[string]string{
				"Authorization": "Bearer " + tokenInfo.AccessToken,
			},
			200,
			"should return only selected fields",
			func(t *testing.T, response map[string]interface{}) {
				if data, ok := response["data"].([]interface{}); ok && len(data) > 0 {
					member := data[0].(map[string]interface{})
					// Should have selected fields
					assert.Contains(t, member, "id", "Should have id field")
					assert.Contains(t, member, "user_id", "Should have user_id field")
					assert.Contains(t, member, "member_type", "Should have member_type field")
					assert.Contains(t, member, "role_id", "Should have role_id field")
					assert.Contains(t, member, "status", "Should have status field")
				}
			},
		},
		{
			"list members with invalid status value",
			teamID,
			"?status=invalid_status",
			map[string]string{
				"Authorization": "Bearer " + tokenInfo.AccessToken,
			},
			400,
			"should reject invalid status value",
			nil,
		},
		{
			"list members with invalid member_type value",
			teamID,
			"?member_type=invalid_type",
			map[string]string{
				"Authorization": "Bearer " + tokenInfo.AccessToken,
			},
			400,
			"should reject invalid member_type value",
			nil,
		},
		{
			"list members with invalid order field",
			teamID,
			"?order=invalid_field+desc",
			map[string]string{
				"Authorization": "Bearer " + tokenInfo.AccessToken,
			},
			400,
			"should reject invalid order field",
			nil,
		},
		{
			"list members with invalid order direction",
			teamID,
			"?order=created_at+invalid",
			map[string]string{
				"Authorization": "Bearer " + tokenInfo.AccessToken,
			},
			400,
			"should reject invalid order direction",
			nil,
		},
		{
			"list members with combined filters",
			teamID,
			"?status=active&member_type=user&order=created_at+asc&page=1&pagesize=5",
			map[string]string{
				"Authorization": "Bearer " + tokenInfo.AccessToken,
			},
			200,
			"should handle combined filters and sorting",
			func(t *testing.T, response map[string]interface{}) {
				assert.Equal(t, float64(1), response["page"], "Should have correct page number")
				assert.Equal(t, float64(5), response["pagesize"], "Should have correct pagesize")
				if data, ok := response["data"].([]interface{}); ok {
					for _, item := range data {
						member := item.(map[string]interface{})
						assert.Equal(t, "active", member["status"], "All members should have active status")
						assert.Equal(t, "user", member["member_type"], "All members should be user type")
					}
				}
			},
		},
		{
			"list members of non-existent team",
			"non-existent-team-id",
			"",
			map[string]string{
				"Authorization": "Bearer " + tokenInfo.AccessToken,
			},
			404,
			"should return not found for non-existent team",
			nil,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			requestURL := serverURL + baseURL + "/user/teams/" + tc.teamID + "/members" + tc.query
			req, err := http.NewRequest("GET", requestURL, nil)
			assert.NoError(t, err, "Should create HTTP request")

			// Add headers
			for key, value := range tc.headers {
				req.Header.Set(key, value)
			}

			httpClient := &http.Client{}
			resp, err := httpClient.Do(req)
			assert.NoError(t, err, "HTTP request should succeed")

			if resp != nil {
				defer resp.Body.Close()
				assert.Equal(t, tc.expectCode, resp.StatusCode, "Expected status code %d for %s", tc.expectCode, tc.name)

				body, err := io.ReadAll(resp.Body)
				assert.NoError(t, err, "Should read response body")

				if resp.StatusCode == 200 {
					// Parse response as pagination result
					var response map[string]interface{}
					err = json.Unmarshal(body, &response)
					assert.NoError(t, err, "Should parse JSON response")

					// Check pagination structure
					assert.Contains(t, response, "data", "Should have data array")
					assert.Contains(t, response, "total", "Should have total count")
					assert.Contains(t, response, "page", "Should have page number")
					assert.Contains(t, response, "pagesize", "Should have pagesize")

					// Run custom validation if provided
					if tc.validateFn != nil {
						tc.validateFn(t, response)
					}
				}

				t.Logf("Member list test %s: status=%d, body=%s", tc.name, resp.StatusCode, string(body))
			}
		})
	}
}

// TestMemberGet tests the GET /user/teams/:team_id/members/:member_id endpoint
func TestMemberGet(t *testing.T) {
	// Initialize test environment
	serverURL := testutils.Prepare(t)
	defer testutils.Clean()

	// Get base URL from server config
	baseURL := ""
	if openapi.Server != nil && openapi.Server.Config != nil {
		baseURL = openapi.Server.Config.BaseURL
	}

	// Register a test client for OAuth authentication
	testClient := testutils.RegisterTestClient(t, "Member Get Test Client", []string{"https://localhost/callback"})
	defer testutils.CleanupTestClient(t, testClient.ClientID)

	// Obtain access token for authenticated requests
	tokenInfo := testutils.ObtainAccessToken(t, serverURL, testClient.ClientID, testClient.ClientSecret, "https://localhost/callback", "openid profile")

	// Create a test team and get owner member ID
	createdTeam := createTestTeam(t, serverURL, baseURL, tokenInfo.AccessToken, "Member Get Test Team")
	teamID := getTeamID(createdTeam)

	// Get member list to find the owner member ID
	ownerMemberID := getOwnerMemberID(t, serverURL, baseURL, teamID, tokenInfo.AccessToken)

	testCases := []struct {
		name       string
		teamID     string
		memberID   string
		headers    map[string]string
		expectCode int
		expectMsg  string
	}{
		{
			"get member without authentication",
			teamID,
			ownerMemberID,
			map[string]string{},
			401,
			"should require authentication",
		},
		{
			"get existing member",
			teamID,
			ownerMemberID,
			map[string]string{
				"Authorization": "Bearer " + tokenInfo.AccessToken,
			},
			200,
			"should return member details",
		},
		{
			"get non-existent member",
			teamID,
			"999999",
			map[string]string{
				"Authorization": "Bearer " + tokenInfo.AccessToken,
			},
			404,
			"should return not found for non-existent member",
		},
		{
			"get member from non-existent team",
			"non-existent-team-id",
			ownerMemberID,
			map[string]string{
				"Authorization": "Bearer " + tokenInfo.AccessToken,
			},
			404,
			"should return not found for non-existent team",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			requestURL := serverURL + baseURL + "/user/teams/" + tc.teamID + "/members/" + tc.memberID
			req, err := http.NewRequest("GET", requestURL, nil)
			assert.NoError(t, err, "Should create HTTP request")

			// Add headers
			for key, value := range tc.headers {
				req.Header.Set(key, value)
			}

			client := &http.Client{}
			resp, err := client.Do(req)
			assert.NoError(t, err, "HTTP request should succeed")

			if resp != nil {
				defer resp.Body.Close()
				assert.Equal(t, tc.expectCode, resp.StatusCode, "Expected status code %d for %s", tc.expectCode, tc.name)

				body, err := io.ReadAll(resp.Body)
				assert.NoError(t, err, "Should read response body")

				if resp.StatusCode == 200 {
					// Parse response as member object
					var member map[string]interface{}
					err = json.Unmarshal(body, &member)
					assert.NoError(t, err, "Should parse JSON response")

					// Verify member structure
					assert.Contains(t, member, "id", "Should have member ID")
					assert.Contains(t, member, "team_id", "Should have team_id")
					assert.Contains(t, member, "user_id", "Should have user_id")
					assert.Contains(t, member, "role_id", "Should have role_id")
					assert.Contains(t, member, "status", "Should have status")
					assert.Contains(t, member, "created_at", "Should have created_at")
					assert.Contains(t, member, "updated_at", "Should have updated_at")

					// Verify values
					assert.Equal(t, teamID, member["team_id"], "Should have correct team_id")
					assert.Equal(t, tokenInfo.UserID, member["user_id"], "Should have correct user_id")
				}

				t.Logf("Member get test %s: status=%d, body=%s", tc.name, resp.StatusCode, string(body))
			}
		})
	}
}

// TestMemberUpdate tests the PUT /user/teams/:team_id/members/:member_id endpoint
func TestMemberUpdate(t *testing.T) {
	// Initialize test environment
	serverURL := testutils.Prepare(t)
	defer testutils.Clean()

	// Get base URL from server config
	baseURL := ""
	if openapi.Server != nil && openapi.Server.Config != nil {
		baseURL = openapi.Server.Config.BaseURL
	}

	// Register a test client for OAuth authentication
	testClient := testutils.RegisterTestClient(t, "Member Update Test Client", []string{"https://localhost/callback"})
	defer testutils.CleanupTestClient(t, testClient.ClientID)

	// Obtain access token for authenticated requests
	tokenInfo := testutils.ObtainAccessToken(t, serverURL, testClient.ClientID, testClient.ClientSecret, "https://localhost/callback", "openid profile")

	testCases := []struct {
		name       string
		setupFunc  func() (string, string) // Returns (teamID, memberID)
		body       map[string]interface{}
		headers    map[string]string
		expectCode int
		expectMsg  string
	}{
		{
			"update member without authentication",
			func() (string, string) {
				team := createTestTeam(t, serverURL, baseURL, tokenInfo.AccessToken, "Update Test Team 1")
				teamID := getTeamID(team)
				memberID := createTestMember(t, serverURL, baseURL, teamID, tokenInfo.AccessToken, "test-update-user-1")
				return teamID, memberID
			},
			map[string]interface{}{
				"role_id": "admin",
			},
			map[string]string{},
			401,
			"should require authentication",
		},
		{
			"update member role",
			func() (string, string) {
				team := createTestTeam(t, serverURL, baseURL, tokenInfo.AccessToken, "Update Test Team 2")
				teamID := getTeamID(team)
				memberID := createTestMember(t, serverURL, baseURL, teamID, tokenInfo.AccessToken, "test-update-user-2")
				return teamID, memberID
			},
			map[string]interface{}{
				"role_id": "admin",
			},
			map[string]string{
				"Authorization": "Bearer " + tokenInfo.AccessToken,
			},
			200,
			"should update member role",
		},
		{
			"update member status",
			func() (string, string) {
				team := createTestTeam(t, serverURL, baseURL, tokenInfo.AccessToken, "Update Test Team 3")
				teamID := getTeamID(team)
				memberID := createTestMember(t, serverURL, baseURL, teamID, tokenInfo.AccessToken, "test-update-user-3")
				return teamID, memberID
			},
			map[string]interface{}{
				"status": "inactive",
			},
			map[string]string{
				"Authorization": "Bearer " + tokenInfo.AccessToken,
			},
			200,
			"should update member status",
		},
		{
			"update non-existent member",
			func() (string, string) {
				team := createTestTeam(t, serverURL, baseURL, tokenInfo.AccessToken, "Update Test Team 5")
				teamID := getTeamID(team)
				return teamID, "999999"
			},
			map[string]interface{}{
				"role_id": "admin",
			},
			map[string]string{
				"Authorization": "Bearer " + tokenInfo.AccessToken,
			},
			404,
			"should return not found for non-existent member",
		},
		{
			"update member in non-existent team",
			func() (string, string) {
				team := createTestTeam(t, serverURL, baseURL, tokenInfo.AccessToken, "Update Test Team 6")
				teamID := getTeamID(team)
				memberID := createTestMember(t, serverURL, baseURL, teamID, tokenInfo.AccessToken, "test-update-user-5")
				return "non-existent-team-id", memberID
			},
			map[string]interface{}{
				"role_id": "admin",
			},
			map[string]string{
				"Authorization": "Bearer " + tokenInfo.AccessToken,
			},
			404,
			"should return not found for non-existent team",
		},
		{
			"update member with invalid JSON",
			func() (string, string) {
				team := createTestTeam(t, serverURL, baseURL, tokenInfo.AccessToken, "Update Test Team 7")
				teamID := getTeamID(team)
				memberID := createTestMember(t, serverURL, baseURL, teamID, tokenInfo.AccessToken, "test-update-user-6")
				return teamID, memberID
			},
			nil, // Will send invalid JSON
			map[string]string{
				"Authorization": "Bearer " + tokenInfo.AccessToken,
			},
			400,
			"should handle invalid JSON",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			teamID, memberID := tc.setupFunc()
			requestURL := serverURL + baseURL + "/user/teams/" + teamID + "/members/" + memberID

			var req *http.Request
			var err error

			if tc.body == nil {
				// Send invalid JSON for invalid JSON test case
				req, err = http.NewRequest("PUT", requestURL, bytes.NewBufferString("invalid json"))
			} else {
				bodyBytes, _ := json.Marshal(tc.body)
				req, err = http.NewRequest("PUT", requestURL, bytes.NewBuffer(bodyBytes))
			}
			assert.NoError(t, err, "Should create HTTP request")

			req.Header.Set("Content-Type", "application/json")

			// Add headers
			for key, value := range tc.headers {
				req.Header.Set(key, value)
			}

			client := &http.Client{}
			resp, err := client.Do(req)
			assert.NoError(t, err, "HTTP request should succeed")

			if resp != nil {
				defer resp.Body.Close()
				assert.Equal(t, tc.expectCode, resp.StatusCode, "Expected status code %d for %s", tc.expectCode, tc.name)

				body, err := io.ReadAll(resp.Body)
				assert.NoError(t, err, "Should read response body")

				if resp.StatusCode == 200 {
					// Parse response as success message
					var response map[string]interface{}
					err = json.Unmarshal(body, &response)
					assert.NoError(t, err, "Should parse JSON response")

					assert.Contains(t, response, "message", "Should have success message")
					assert.Equal(t, "Member updated successfully", response["message"], "Should have correct success message")
				}

				t.Logf("Member update test %s: status=%d, body=%s", tc.name, resp.StatusCode, string(body))
			}
		})
	}
}

// TestMemberDelete tests the DELETE /user/teams/:team_id/members/:member_id endpoint
func TestMemberDelete(t *testing.T) {
	// Initialize test environment
	serverURL := testutils.Prepare(t)
	defer testutils.Clean()

	// Get base URL from server config
	baseURL := ""
	if openapi.Server != nil && openapi.Server.Config != nil {
		baseURL = openapi.Server.Config.BaseURL
	}

	// Register a test client for OAuth authentication
	testClient := testutils.RegisterTestClient(t, "Member Delete Test Client", []string{"https://localhost/callback"})
	defer testutils.CleanupTestClient(t, testClient.ClientID)

	// Obtain access token for authenticated requests
	tokenInfo := testutils.ObtainAccessToken(t, serverURL, testClient.ClientID, testClient.ClientSecret, "https://localhost/callback", "openid profile")

	// Create teams and members for testing deletion
	createMemberForDeletion := func(name string) (string, string) {
		team := createTestTeam(t, serverURL, baseURL, tokenInfo.AccessToken, "Delete Test Team "+name)
		teamID := getTeamID(team)
		memberID := createTestMember(t, serverURL, baseURL, teamID, tokenInfo.AccessToken, "test-delete-user-"+name)
		return teamID, memberID
	}

	testCases := []struct {
		name       string
		setupFunc  func() (string, string) // Returns (teamID, memberID)
		headers    map[string]string
		expectCode int
		expectMsg  string
	}{
		{
			"delete member without authentication",
			func() (string, string) { return createMemberForDeletion("1") },
			map[string]string{},
			401,
			"should require authentication",
		},
		{
			"delete existing member",
			func() (string, string) { return createMemberForDeletion("2") },
			map[string]string{
				"Authorization": "Bearer " + tokenInfo.AccessToken,
			},
			200,
			"should delete member successfully",
		},
		{
			"delete non-existent member",
			func() (string, string) {
				team := createTestTeam(t, serverURL, baseURL, tokenInfo.AccessToken, "Delete Test Team 3")
				return getTeamID(team), "999999"
			},
			map[string]string{
				"Authorization": "Bearer " + tokenInfo.AccessToken,
			},
			404,
			"should return not found for non-existent member",
		},
		{
			"delete member from non-existent team",
			func() (string, string) {
				_, memberID := createMemberForDeletion("4")
				return "non-existent-team-id", memberID
			},
			map[string]string{
				"Authorization": "Bearer " + tokenInfo.AccessToken,
			},
			404,
			"should return not found for non-existent team",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			teamID, memberID := tc.setupFunc()
			requestURL := serverURL + baseURL + "/user/teams/" + teamID + "/members/" + memberID

			req, err := http.NewRequest("DELETE", requestURL, nil)
			assert.NoError(t, err, "Should create HTTP request")

			// Add headers
			for key, value := range tc.headers {
				req.Header.Set(key, value)
			}

			client := &http.Client{}
			resp, err := client.Do(req)
			assert.NoError(t, err, "HTTP request should succeed")

			if resp != nil {
				defer resp.Body.Close()
				assert.Equal(t, tc.expectCode, resp.StatusCode, "Expected status code %d for %s", tc.expectCode, tc.name)

				body, err := io.ReadAll(resp.Body)
				assert.NoError(t, err, "Should read response body")

				if resp.StatusCode == 200 {
					// Parse response as success message
					var response map[string]interface{}
					err = json.Unmarshal(body, &response)
					assert.NoError(t, err, "Should parse JSON response")

					assert.Contains(t, response, "message", "Should have success message")
					assert.Equal(t, "Member removed successfully", response["message"], "Should have correct success message")
				}

				t.Logf("Member delete test %s: status=%d, body=%s", tc.name, resp.StatusCode, string(body))
			}
		})
	}
}

// TestMemberPermissionVerification tests permission verification for member operations
func TestMemberPermissionVerification(t *testing.T) {
	// Initialize test environment
	serverURL := testutils.Prepare(t)
	defer testutils.Clean()

	// Get base URL from server config
	baseURL := ""
	if openapi.Server != nil && openapi.Server.Config != nil {
		baseURL = openapi.Server.Config.BaseURL
	}

	// Register test clients for different users
	ownerClient := testutils.RegisterTestClient(t, "Owner Test Client", []string{"https://localhost/callback"})
	defer testutils.CleanupTestClient(t, ownerClient.ClientID)

	nonOwnerClient := testutils.RegisterTestClient(t, "Non-Owner Test Client", []string{"https://localhost/callback"})
	defer testutils.CleanupTestClient(t, nonOwnerClient.ClientID)

	// Obtain access tokens
	ownerToken := testutils.ObtainAccessToken(t, serverURL, ownerClient.ClientID, ownerClient.ClientSecret, "https://localhost/callback", "openid profile")
	nonOwnerToken := testutils.ObtainAccessToken(t, serverURL, nonOwnerClient.ClientID, nonOwnerClient.ClientSecret, "https://localhost/callback", "openid profile")

	// Create a team with owner
	createdTeam := createTestTeam(t, serverURL, baseURL, ownerToken.AccessToken, "Permission Test Team")
	teamID := getTeamID(createdTeam)

	// Add non-owner as a member
	memberID := createTestMember(t, serverURL, baseURL, teamID, ownerToken.AccessToken, nonOwnerToken.UserID)

	testCases := []struct {
		name       string
		endpoint   string
		method     string
		token      string
		expectCode int
		expectMsg  string
	}{
		{
			"owner can list members",
			"/user/teams/" + teamID + "/members",
			"GET",
			ownerToken.AccessToken,
			200,
			"owner should be able to list members",
		},
		{
			"member can list members",
			"/user/teams/" + teamID + "/members",
			"GET",
			nonOwnerToken.AccessToken,
			200,
			"member should be able to list members",
		},
		{
			"owner can get member details",
			"/user/teams/" + teamID + "/members/" + memberID,
			"GET",
			ownerToken.AccessToken,
			200,
			"owner should be able to get member details",
		},
		{
			"member can get member details",
			"/user/teams/" + teamID + "/members/" + memberID,
			"GET",
			nonOwnerToken.AccessToken,
			200,
			"member should be able to get member details",
		},
		{
			"owner can update members",
			"/user/teams/" + teamID + "/members/" + memberID,
			"PUT",
			ownerToken.AccessToken,
			200,
			"owner should be able to update members",
		},
		{
			"member cannot update members",
			"/user/teams/" + teamID + "/members/" + memberID,
			"PUT",
			nonOwnerToken.AccessToken,
			403,
			"member should not be able to update members",
		},
		{
			"owner can delete members",
			"/user/teams/" + teamID + "/members/" + memberID,
			"DELETE",
			ownerToken.AccessToken,
			200,
			"owner should be able to delete members",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			requestURL := serverURL + baseURL + tc.endpoint

			var req *http.Request
			var err error

			// Create request body for POST/PUT methods
			if tc.method == "POST" {
				body := map[string]interface{}{
					"user_id": "test-permission-user",
					"role_id": "member",
				}
				bodyBytes, _ := json.Marshal(body)
				req, err = http.NewRequest(tc.method, requestURL, bytes.NewBuffer(bodyBytes))
				req.Header.Set("Content-Type", "application/json")
			} else if tc.method == "PUT" {
				body := map[string]interface{}{
					"role_id": "admin",
				}
				bodyBytes, _ := json.Marshal(body)
				req, err = http.NewRequest(tc.method, requestURL, bytes.NewBuffer(bodyBytes))
				req.Header.Set("Content-Type", "application/json")
			} else {
				req, err = http.NewRequest(tc.method, requestURL, nil)
			}

			assert.NoError(t, err, "Should create HTTP request")

			req.Header.Set("Authorization", "Bearer "+tc.token)

			client := &http.Client{}
			resp, err := client.Do(req)
			assert.NoError(t, err, "HTTP request should succeed")

			if resp != nil {
				defer resp.Body.Close()
				assert.Equal(t, tc.expectCode, resp.StatusCode, "Expected status code %d for %s", tc.expectCode, tc.name)

				body, err := io.ReadAll(resp.Body)
				assert.NoError(t, err, "Should read response body")

				t.Logf("Permission test %s: status=%d, body=%s", tc.name, resp.StatusCode, string(body))
			}
		})
	}
}

// Helper functions

// createTestTeam creates a team for testing and returns the team data
func createTestTeam(t *testing.T, serverURL, baseURL, accessToken, teamName string) map[string]interface{} {
	createTeamBody := map[string]interface{}{
		"name":        teamName,
		"description": "Team created for testing purposes",
		"role_id":     "system:root", // Use system:root role which includes all scopes
	}

	bodyBytes, err := json.Marshal(createTeamBody)
	assert.NoError(t, err, "Should marshal team creation body")

	req, err := http.NewRequest("POST", serverURL+baseURL+"/user/teams", bytes.NewBuffer(bodyBytes))
	assert.NoError(t, err, "Should create team creation request")

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+accessToken)

	client := &http.Client{}
	resp, err := client.Do(req)
	assert.NoError(t, err, "Should send team creation request")
	defer resp.Body.Close()

	assert.Equal(t, 201, resp.StatusCode, "Should create team successfully")

	body, err := io.ReadAll(resp.Body)
	assert.NoError(t, err, "Should read team creation response")

	var team map[string]interface{}
	err = json.Unmarshal(body, &team)
	assert.NoError(t, err, "Should parse team creation response")

	return team
}

// createTestMember creates a member for testing using provider directly (no API call).
// This is the recommended approach since direct member creation endpoint was removed.
// Members should normally be added via invitation flow or robot creation endpoint.
// Returns the member_id (global unique identifier).
func createTestMember(t *testing.T, serverURL, baseURL, teamID, accessToken, userID string) string {
	// Get user provider for direct database operations
	provider := testutils.GetUserProvider(t)
	ctx := context.Background()

	// Create member data using maps.MapStrAny (required by UserProvider interface)
	memberData := maps.MapStrAny{
		"team_id":     teamID,
		"user_id":     userID,
		"member_type": "user",
		"role_id":     "team:member",
		"status":      "active",
	}

	// Create member directly in database
	memberID, err := provider.CreateMember(ctx, memberData)
	assert.NoError(t, err, "Should create member in database")
	assert.NotEmpty(t, memberID, "Member ID should not be empty")

	t.Logf("Created test member directly in database: user_id=%s, member_id=%s, team_id=%s", userID, memberID, teamID)

	// Return member_id (global unique identifier used in API)
	return memberID
}

// getOwnerMemberID gets the member_id of the team owner (global unique identifier)
func getOwnerMemberID(t *testing.T, serverURL, baseURL, teamID, accessToken string) string {
	req, err := http.NewRequest("GET", serverURL+baseURL+"/user/teams/"+teamID+"/members", nil)
	assert.NoError(t, err, "Should create member list request")

	req.Header.Set("Authorization", "Bearer "+accessToken)

	client := &http.Client{}
	resp, err := client.Do(req)
	assert.NoError(t, err, "Should send member list request")
	defer resp.Body.Close()

	assert.Equal(t, 200, resp.StatusCode, "Should get members successfully")

	body, err := io.ReadAll(resp.Body)
	assert.NoError(t, err, "Should read member list response")

	var response map[string]interface{}
	err = json.Unmarshal(body, &response)
	assert.NoError(t, err, "Should parse member list response")

	data, ok := response["data"].([]interface{})
	assert.True(t, ok, "Should have data array")
	assert.Greater(t, len(data), 0, "Should have at least one member")

	// Find the owner member and return their member_id
	for _, item := range data {
		member := item.(map[string]interface{})
		if role, ok := member["role_id"].(string); ok && strings.HasPrefix(role, "owner") {
			memberID, ok := member["member_id"].(string)
			if !ok {
				t.Fatal("Owner member missing member_id")
			}
			return memberID
		}
	}

	t.Fatal("Could not find owner member")
	return ""
}

// TestMemberCreateRobot tests the POST /user/teams/:team_id/members/robots endpoint
func TestMemberCreateRobot(t *testing.T) {
	// Initialize test environment
	serverURL := testutils.Prepare(t)
	defer testutils.Clean()

	// Get base URL from server config
	baseURL := ""
	if openapi.Server != nil && openapi.Server.Config != nil {
		baseURL = openapi.Server.Config.BaseURL
	}

	// Register a test client for OAuth authentication
	testClient := testutils.RegisterTestClient(t, "Robot Member Test Client", []string{"https://localhost/callback"})
	defer testutils.CleanupTestClient(t, testClient.ClientID)

	// Obtain access token with root permissions (required for creating robot members)
	tokenInfo := testutils.ObtainAccessTokenWithRootPermission(t, serverURL, testClient.ClientID, testClient.ClientSecret, "https://localhost/callback", "openid profile")

	// Use UUID to ensure unique team name
	testUUID := strings.ReplaceAll(uuid.New().String(), "-", "")[:8]

	// Create a test team
	createdTeam := createTestTeam(t, serverURL, baseURL, tokenInfo.AccessToken, "Robot Member Test Team "+testUUID)
	teamID := getTeamID(createdTeam)

	testCases := []struct {
		name       string
		teamID     string
		body       map[string]interface{}
		headers    map[string]string
		expectCode int
		expectMsg  string
	}{
		{
			"create robot without authentication",
			teamID,
			map[string]interface{}{
				"name":   "Test Robot",
				"email":  "robot@test.com",
				"role":   "member",
				"prompt": "You are a helpful assistant",
			},
			map[string]string{},
			401,
			"should require authentication",
		},
		{
			"create robot with all fields",
			teamID,
			map[string]interface{}{
				"name":               "AI Assistant Full",
				"avatar":             fmt.Sprintf("https://example.com/avatars/ai-full-%s.png", testUUID),
				"email":              fmt.Sprintf("ai-full-%s@test.com", testUUID),
				"robot_email":        fmt.Sprintf("robot-full-%s@robot.test.com", testUUID),
				"authorized_senders": []string{"user1@test.com", "user2@test.com"},
				"email_filter_rules": []string{".*@company\\.com", ".*@partner\\.com"},
				"bio":                "A comprehensive AI assistant",
				"role":               "member",
				"report_to":          tokenInfo.UserID,
				"prompt":             "You are a helpful AI assistant with full capabilities",
				"llm":                "gpt-4",
				"agents":             []string{"data-analyst", "code-reviewer"},
				"mcp_tools":          []string{"filesystem", "database"},
				"autonomous_mode":    "enabled",
				"cost_limit":         100.50,
			},
			map[string]string{
				"Authorization": "Bearer " + tokenInfo.AccessToken,
			},
			201,
			"should create robot with all fields successfully",
		},
		{
			"create robot with required fields only",
			teamID,
			map[string]interface{}{
				"name":        "AI Assistant Min",
				"robot_email": fmt.Sprintf("ai-min-%s@test.com", testUUID),
				"role":        "member",
				"prompt":      "You are a basic assistant",
			},
			map[string]string{
				"Authorization": "Bearer " + tokenInfo.AccessToken,
			},
			201,
			"should create robot with required fields only",
		},
		{
			"create robot with autonomous_mode variations",
			teamID,
			map[string]interface{}{
				"name":            "AI Assistant Auto",
				"robot_email":     fmt.Sprintf("ai-auto-%s@test.com", testUUID),
				"role":            "member",
				"prompt":          "You are an autonomous assistant",
				"autonomous_mode": "1", // Test numeric string
			},
			map[string]string{
				"Authorization": "Bearer " + tokenInfo.AccessToken,
			},
			201,
			"should handle autonomous_mode=1",
		},
		{
			"create robot with disabled autonomous_mode",
			teamID,
			map[string]interface{}{
				"name":            "AI Assistant Manual",
				"robot_email":     fmt.Sprintf("ai-manual-%s@test.com", testUUID),
				"role":            "member",
				"prompt":          "You are a manual assistant",
				"autonomous_mode": "disabled",
			},
			map[string]string{
				"Authorization": "Bearer " + tokenInfo.AccessToken,
			},
			201,
			"should handle autonomous_mode=disabled",
		},
		{
			"create robot without name",
			teamID,
			map[string]interface{}{
				"robot_email": "no-name@test.com",
				"role":        "member",
				"prompt":      "You are an assistant",
			},
			map[string]string{
				"Authorization": "Bearer " + tokenInfo.AccessToken,
			},
			400,
			"should require name",
		},
		{
			"create robot without robot_email",
			teamID,
			map[string]interface{}{
				"name":   "No Robot Email Robot",
				"role":   "member",
				"prompt": "You are an assistant",
			},
			map[string]string{
				"Authorization": "Bearer " + tokenInfo.AccessToken,
			},
			400,
			"should require robot_email",
		},
		{
			"create robot without role",
			teamID,
			map[string]interface{}{
				"name":        "No Role Robot",
				"robot_email": "no-role@test.com",
				"prompt":      "You are an assistant",
			},
			map[string]string{
				"Authorization": "Bearer " + tokenInfo.AccessToken,
			},
			400,
			"should require role",
		},
		{
			"create robot without prompt",
			teamID,
			map[string]interface{}{
				"name":        "No Prompt Robot",
				"robot_email": "no-prompt@test.com",
				"role":        "member",
			},
			map[string]string{
				"Authorization": "Bearer " + tokenInfo.AccessToken,
			},
			400,
			"should require prompt",
		},
		{
			"create robot with duplicate robot_email",
			teamID,
			map[string]interface{}{
				"name":        "Duplicate Robot Email Robot",
				"email":       fmt.Sprintf("duplicate-robot-%s@test.com", testUUID),  // Different email
				"robot_email": fmt.Sprintf("robot-full-%s@robot.test.com", testUUID), // Same robot_email as first successful case
				"role":        "member",
				"prompt":      "You are an assistant",
			},
			map[string]string{
				"Authorization": "Bearer " + tokenInfo.AccessToken,
			},
			409,
			"should reject duplicate robot_email globally",
		},
		{
			"create robot in non-existent team",
			"non-existent-team-id",
			map[string]interface{}{
				"name":        "Robot in Void",
				"robot_email": "void@test.com",
				"role":        "member",
				"prompt":      "You are lost",
			},
			map[string]string{
				"Authorization": "Bearer " + tokenInfo.AccessToken,
			},
			404,
			"should return not found for non-existent team",
		},
		{
			"create robot with invalid JSON",
			teamID,
			nil, // Will send invalid JSON
			map[string]string{
				"Authorization": "Bearer " + tokenInfo.AccessToken,
			},
			400,
			"should handle invalid JSON",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			requestURL := serverURL + baseURL + "/user/teams/" + tc.teamID + "/members/robots"

			var req *http.Request
			var err error

			if tc.body == nil {
				// Send invalid JSON for invalid JSON test case
				req, err = http.NewRequest("POST", requestURL, bytes.NewBufferString("invalid json"))
			} else {
				bodyBytes, _ := json.Marshal(tc.body)
				req, err = http.NewRequest("POST", requestURL, bytes.NewBuffer(bodyBytes))
			}
			assert.NoError(t, err, "Should create HTTP request")

			req.Header.Set("Content-Type", "application/json")

			// Add headers
			for key, value := range tc.headers {
				req.Header.Set(key, value)
			}

			client := &http.Client{}
			resp, err := client.Do(req)
			assert.NoError(t, err, "HTTP request should succeed")

			if resp != nil {
				defer resp.Body.Close()
				assert.Equal(t, tc.expectCode, resp.StatusCode, "Expected status code %d for %s", tc.expectCode, tc.name)

				body, err := io.ReadAll(resp.Body)
				assert.NoError(t, err, "Should read response body")

				if resp.StatusCode == 201 {
					// Parse response as created member
					var response map[string]interface{}
					err = json.Unmarshal(body, &response)
					assert.NoError(t, err, "Should parse JSON response")

					// Verify response structure
					assert.Contains(t, response, "member_id", "Should have member_id")
					assert.NotEmpty(t, response["member_id"], "Member ID should not be empty")

					// Verify the member was created with correct type
					memberID := toString(response["member_id"])
					getMemberURL := serverURL + baseURL + "/user/teams/" + tc.teamID + "/members/" + memberID
					getReq, _ := http.NewRequest("GET", getMemberURL, nil)
					getReq.Header.Set("Authorization", "Bearer "+tokenInfo.AccessToken)

					getResp, err := client.Do(getReq)
					if err == nil && getResp != nil {
						defer getResp.Body.Close()
						if getResp.StatusCode == 200 {
							var member map[string]interface{}
							getBody, _ := io.ReadAll(getResp.Body)
							json.Unmarshal(getBody, &member)

							// Verify robot member fields
							assert.Equal(t, "robot", member["member_type"], "Should be robot member type")
							if tc.body["name"] != nil {
								assert.Equal(t, tc.body["name"], member["display_name"], "Should have correct display_name")
							}
							if tc.body["email"] != nil {
								assert.Equal(t, tc.body["email"], member["email"], "Should have correct email")
							}
							if tc.body["prompt"] != nil {
								assert.Equal(t, tc.body["prompt"], member["system_prompt"], "Should have correct system_prompt")
							}
						}
					}
				}

				t.Logf("Robot member create test %s: status=%d, body=%s", tc.name, resp.StatusCode, string(body))
			}
		})
	}
}

// toString converts interface{} to string for test assertions
func toString(v interface{}) string {
	switch val := v.(type) {
	case string:
		return val
	case float64:
		return fmt.Sprintf("%.0f", val)
	case int:
		return fmt.Sprintf("%d", val)
	case int64:
		return fmt.Sprintf("%d", val)
	default:
		return fmt.Sprintf("%v", val)
	}
}

// TestMemberCheckRobotEmail tests the GET /user/teams/:team_id/members/check-robot-email endpoint
func TestMemberCheckRobotEmail(t *testing.T) {
	// Initialize test environment
	serverURL := testutils.Prepare(t)
	defer testutils.Clean()

	// Get base URL from server config
	baseURL := ""
	if openapi.Server != nil && openapi.Server.Config != nil {
		baseURL = openapi.Server.Config.BaseURL
	}

	// Register a test client for OAuth authentication
	testClient := testutils.RegisterTestClient(t, "Member Check Robot Email Test Client", []string{"https://localhost/callback"})
	defer testutils.CleanupTestClient(t, testClient.ClientID)

	// Obtain access token for authenticated requests
	tokenInfo := testutils.ObtainAccessToken(t, serverURL, testClient.ClientID, testClient.ClientSecret, "https://localhost/callback", "openid profile")

	// Use UUID to ensure unique test data
	testUUID := strings.ReplaceAll(uuid.New().String(), "-", "")[:8]

	// Create a test team
	createdTeam := createTestTeam(t, serverURL, baseURL, tokenInfo.AccessToken, "Robot Email Check Test Team "+testUUID)
	teamID := getTeamID(createdTeam)

	// Create a robot member with a known robot_email (globally unique)
	existingRobotEmail := fmt.Sprintf("existing-robot-%s@robot.test.com", testUUID)
	robotBody := map[string]interface{}{
		"name":        "Existing Robot",
		"email":       fmt.Sprintf("display-%s@test.com", testUUID), // Display email (can be non-unique)
		"robot_email": existingRobotEmail,                           // Globally unique robot email
		"role":        "member",
		"prompt":      "You are a test robot",
	}
	robotBodyBytes, _ := json.Marshal(robotBody)
	robotReq, _ := http.NewRequest("POST", serverURL+baseURL+"/user/teams/"+teamID+"/members/robots", bytes.NewBuffer(robotBodyBytes))
	robotReq.Header.Set("Content-Type", "application/json")
	robotReq.Header.Set("Authorization", "Bearer "+tokenInfo.AccessToken)
	client := &http.Client{}
	robotResp, err := client.Do(robotReq)
	assert.NoError(t, err)
	if robotResp != nil {
		robotResp.Body.Close()
		assert.Equal(t, 201, robotResp.StatusCode, "Should create robot member successfully")
	}

	testCases := []struct {
		name         string
		teamID       string
		robotEmail   string
		headers      map[string]string
		expectCode   int
		expectExists bool
		expectMsg    string
	}{
		{
			"check robot email without authentication",
			teamID,
			existingRobotEmail,
			map[string]string{},
			401,
			false,
			"should require authentication",
		},
		{
			"check existing robot email",
			teamID,
			existingRobotEmail,
			map[string]string{
				"Authorization": "Bearer " + tokenInfo.AccessToken,
			},
			200,
			true,
			"should return exists=true for existing robot email",
		},
		{
			"check non-existing robot email",
			teamID,
			fmt.Sprintf("nonexistent-%s@robot.test.com", testUUID),
			map[string]string{
				"Authorization": "Bearer " + tokenInfo.AccessToken,
			},
			200,
			false,
			"should return exists=false for non-existing robot email",
		},
		{
			"check robot email without robot_email parameter",
			teamID,
			"",
			map[string]string{
				"Authorization": "Bearer " + tokenInfo.AccessToken,
			},
			400,
			false,
			"should require robot_email parameter",
		},
		{
			"check robot email in non-existent team",
			"non-existent-team-id",
			"test@robot.example.com",
			map[string]string{
				"Authorization": "Bearer " + tokenInfo.AccessToken,
			},
			404,
			false,
			"should return not found for non-existent team",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			requestURL := serverURL + baseURL + "/user/teams/" + tc.teamID + "/members/check-robot-email"
			if tc.robotEmail != "" {
				requestURL += "?robot_email=" + tc.robotEmail
			}

			req, err := http.NewRequest("GET", requestURL, nil)
			assert.NoError(t, err, "Should create HTTP request")

			// Add headers
			for key, value := range tc.headers {
				req.Header.Set(key, value)
			}

			resp, err := client.Do(req)
			assert.NoError(t, err, "HTTP request should succeed")

			if resp != nil {
				defer resp.Body.Close()
				assert.Equal(t, tc.expectCode, resp.StatusCode, "Expected status code %d for %s", tc.expectCode, tc.name)

				body, err := io.ReadAll(resp.Body)
				assert.NoError(t, err, "Should read response body")

				if resp.StatusCode == 200 {
					// Parse response
					var response map[string]interface{}
					err = json.Unmarshal(body, &response)
					assert.NoError(t, err, "Should parse JSON response")

					// Verify response structure (global check, no team_id in response)
					assert.Contains(t, response, "exists", "Should have exists field")
					assert.Contains(t, response, "robot_email", "Should have robot_email field")

					// Verify values
					assert.Equal(t, tc.expectExists, response["exists"], "Should have correct exists value")
					assert.Equal(t, tc.robotEmail, response["robot_email"], "Should have correct robot_email")
				}

				t.Logf("Member check email test %s: status=%d, body=%s", tc.name, resp.StatusCode, string(body))
			}
		})
	}
}

// TestMemberUpdateRobot tests the PUT /user/teams/:team_id/members/robots/:member_id endpoint
func TestMemberUpdateRobot(t *testing.T) {
	// Initialize test environment
	serverURL := testutils.Prepare(t)
	defer testutils.Clean()

	// Get base URL from server config
	baseURL := ""
	if openapi.Server != nil && openapi.Server.Config != nil {
		baseURL = openapi.Server.Config.BaseURL
	}

	// Register a test client for OAuth authentication
	testClient := testutils.RegisterTestClient(t, "Robot Member Update Test Client", []string{"https://localhost/callback"})
	defer testutils.CleanupTestClient(t, testClient.ClientID)

	// Obtain access token with root permissions (required for robot operations)
	tokenInfo := testutils.ObtainAccessTokenWithRootPermission(t, serverURL, testClient.ClientID, testClient.ClientSecret, "https://localhost/callback", "openid profile")

	// Use UUID to ensure unique test data
	testUUID := strings.ReplaceAll(uuid.New().String(), "-", "")[:8]

	// Create a test team
	createdTeam := createTestTeam(t, serverURL, baseURL, tokenInfo.AccessToken, "Robot Update Test Team "+testUUID)
	teamID := getTeamID(createdTeam)

	// Helper function to create a robot member for testing
	createTestRobot := func(suffix string) (string, string) {
		robotEmail := fmt.Sprintf("test-robot-%s-%s@robot.test.com", testUUID, suffix)
		robotBody := map[string]interface{}{
			"name":            "Test Robot " + suffix,
			"robot_email":     robotEmail,
			"email":           fmt.Sprintf("display-%s-%s@test.com", testUUID, suffix),
			"role":            "member",
			"prompt":          "Original prompt for " + suffix,
			"llm":             "gpt-3.5-turbo",
			"autonomous_mode": "disabled",
			"cost_limit":      50.0,
		}
		robotBodyBytes, _ := json.Marshal(robotBody)
		robotReq, _ := http.NewRequest("POST", serverURL+baseURL+"/user/teams/"+teamID+"/members/robots", bytes.NewBuffer(robotBodyBytes))
		robotReq.Header.Set("Content-Type", "application/json")
		robotReq.Header.Set("Authorization", "Bearer "+tokenInfo.AccessToken)
		client := &http.Client{}
		robotResp, err := client.Do(robotReq)
		assert.NoError(t, err)
		if robotResp != nil {
			defer robotResp.Body.Close()
			assert.Equal(t, 201, robotResp.StatusCode, "Should create robot member successfully")
			body, _ := io.ReadAll(robotResp.Body)
			var response map[string]interface{}
			json.Unmarshal(body, &response)
			return toString(response["member_id"]), robotEmail
		}
		return "", ""
	}

	testCases := []struct {
		name       string
		setupFunc  func() (string, string) // Returns (memberID, originalRobotEmail)
		body       map[string]interface{}
		headers    map[string]string
		expectCode int
		expectMsg  string
		validateFn func(*testing.T, string) // Optional validation function with memberID
	}{
		{
			"update robot without authentication",
			func() (string, string) { return createTestRobot("1") },
			map[string]interface{}{
				"name": "Updated Name",
			},
			map[string]string{},
			401,
			"should require authentication",
			nil,
		},
		{
			"update robot with all fields",
			func() (string, string) { return createTestRobot("2") },
			map[string]interface{}{
				"name":               "Updated Robot Full",
				"avatar":             fmt.Sprintf("https://example.com/avatars/full-%s.png", testUUID),
				"email":              fmt.Sprintf("updated-display-%s@test.com", testUUID),
				"robot_email":        fmt.Sprintf("updated-robot-%s@robot.test.com", testUUID),
				"bio":                "Updated comprehensive description",
				"role":               "admin",
				"report_to":          tokenInfo.UserID,
				"prompt":             "Updated system prompt",
				"llm":                "gpt-4",
				"agents":             []string{"agent1", "agent2"},
				"mcp_tools":          []string{"tool1", "tool2"},
				"authorized_senders": []string{"admin@test.com"},
				"email_filter_rules": []string{".*@test\\.com$"},
				"autonomous_mode":    "enabled",
				"cost_limit":         100.0,
				"status":             "active",
				"robot_status":       "working",
			},
			map[string]string{
				"Authorization": "Bearer " + tokenInfo.AccessToken,
			},
			200,
			"should update robot with all fields successfully",
			func(t *testing.T, memberID string) {
				// Verify the update
				getMemberURL := serverURL + baseURL + "/user/teams/" + teamID + "/members/" + memberID
				getReq, _ := http.NewRequest("GET", getMemberURL, nil)
				getReq.Header.Set("Authorization", "Bearer "+tokenInfo.AccessToken)
				client := &http.Client{}
				getResp, err := client.Do(getReq)
				assert.NoError(t, err)
				if getResp != nil {
					defer getResp.Body.Close()
					if getResp.StatusCode == 200 {
						var member map[string]interface{}
						body, _ := io.ReadAll(getResp.Body)
						json.Unmarshal(body, &member)
						assert.Equal(t, "Updated Robot Full", member["display_name"])
						assert.Equal(t, fmt.Sprintf("https://example.com/avatars/full-%s.png", testUUID), member["avatar"])
						assert.Equal(t, "Updated system prompt", member["system_prompt"])
						assert.Equal(t, "gpt-4", member["language_model"])
					}
				}
			},
		},
		{
			"update robot with partial fields",
			func() (string, string) { return createTestRobot("3") },
			map[string]interface{}{
				"name":   "Partially Updated Robot",
				"prompt": "Partially updated prompt",
			},
			map[string]string{
				"Authorization": "Bearer " + tokenInfo.AccessToken,
			},
			200,
			"should update robot with partial fields",
			func(t *testing.T, memberID string) {
				getMemberURL := serverURL + baseURL + "/user/teams/" + teamID + "/members/" + memberID
				getReq, _ := http.NewRequest("GET", getMemberURL, nil)
				getReq.Header.Set("Authorization", "Bearer "+tokenInfo.AccessToken)
				client := &http.Client{}
				getResp, err := client.Do(getReq)
				assert.NoError(t, err)
				if getResp != nil {
					defer getResp.Body.Close()
					if getResp.StatusCode == 200 {
						var member map[string]interface{}
						body, _ := io.ReadAll(getResp.Body)
						json.Unmarshal(body, &member)
						assert.Equal(t, "Partially Updated Robot", member["display_name"])
						assert.Equal(t, "Partially updated prompt", member["system_prompt"])
						// Original fields should remain
						assert.Equal(t, "gpt-3.5-turbo", member["language_model"])
					}
				}
			},
		},
		{
			"update robot_email to new unique email",
			func() (string, string) { return createTestRobot("4") },
			map[string]interface{}{
				"robot_email": fmt.Sprintf("new-unique-%s@robot.test.com", testUUID),
			},
			map[string]string{
				"Authorization": "Bearer " + tokenInfo.AccessToken,
			},
			200,
			"should update robot_email to new unique email",
			func(t *testing.T, memberID string) {
				getMemberURL := serverURL + baseURL + "/user/teams/" + teamID + "/members/" + memberID
				getReq, _ := http.NewRequest("GET", getMemberURL, nil)
				getReq.Header.Set("Authorization", "Bearer "+tokenInfo.AccessToken)
				client := &http.Client{}
				getResp, err := client.Do(getReq)
				assert.NoError(t, err)
				if getResp != nil {
					defer getResp.Body.Close()
					if getResp.StatusCode == 200 {
						var member map[string]interface{}
						body, _ := io.ReadAll(getResp.Body)
						json.Unmarshal(body, &member)
						assert.Equal(t, fmt.Sprintf("new-unique-%s@robot.test.com", testUUID), member["robot_email"])
					}
				}
			},
		},
		{
			"update robot_email to duplicate email",
			func() (string, string) {
				// Create two robots
				memberID1, email1 := createTestRobot("5a")
				_, _ = createTestRobot("5b")
				return memberID1, email1
			},
			map[string]interface{}{
				"robot_email": fmt.Sprintf("test-robot-%s-5b@robot.test.com", testUUID),
			},
			map[string]string{
				"Authorization": "Bearer " + tokenInfo.AccessToken,
			},
			409,
			"should reject duplicate robot_email",
			nil,
		},
		{
			"update autonomous_mode variations",
			func() (string, string) { return createTestRobot("6") },
			map[string]interface{}{
				"autonomous_mode": "1",
			},
			map[string]string{
				"Authorization": "Bearer " + tokenInfo.AccessToken,
			},
			200,
			"should handle autonomous_mode=1",
			func(t *testing.T, memberID string) {
				getMemberURL := serverURL + baseURL + "/user/teams/" + teamID + "/members/" + memberID
				getReq, _ := http.NewRequest("GET", getMemberURL, nil)
				getReq.Header.Set("Authorization", "Bearer "+tokenInfo.AccessToken)
				client := &http.Client{}
				getResp, err := client.Do(getReq)
				assert.NoError(t, err)
				if getResp != nil {
					defer getResp.Body.Close()
					if getResp.StatusCode == 200 {
						var member map[string]interface{}
						body, _ := io.ReadAll(getResp.Body)
						json.Unmarshal(body, &member)
						// autonomous_mode should be enabled
						autonomousMode := member["autonomous_mode"]
						assert.True(t, autonomousMode == true || autonomousMode == float64(1) || autonomousMode == int64(1))
					}
				}
			},
		},
		{
			"update robot status",
			func() (string, string) { return createTestRobot("7") },
			map[string]interface{}{
				"status":       "inactive",
				"robot_status": "error",
			},
			map[string]string{
				"Authorization": "Bearer " + tokenInfo.AccessToken,
			},
			200,
			"should update robot status fields",
			func(t *testing.T, memberID string) {
				getMemberURL := serverURL + baseURL + "/user/teams/" + teamID + "/members/" + memberID
				getReq, _ := http.NewRequest("GET", getMemberURL, nil)
				getReq.Header.Set("Authorization", "Bearer "+tokenInfo.AccessToken)
				client := &http.Client{}
				getResp, err := client.Do(getReq)
				assert.NoError(t, err)
				if getResp != nil {
					defer getResp.Body.Close()
					if getResp.StatusCode == 200 {
						var member map[string]interface{}
						body, _ := io.ReadAll(getResp.Body)
						json.Unmarshal(body, &member)
						assert.Equal(t, "inactive", member["status"])
						assert.Equal(t, "error", member["robot_status"])
					}
				}
			},
		},
		{
			"update array fields",
			func() (string, string) { return createTestRobot("8") },
			map[string]interface{}{
				"agents":             []string{"new-agent1", "new-agent2", "new-agent3"},
				"mcp_tools":          []string{"new-tool1"},
				"authorized_senders": []string{"sender1@test.com", "sender2@test.com"},
				"email_filter_rules": []string{".*@allowed\\.com$"},
			},
			map[string]string{
				"Authorization": "Bearer " + tokenInfo.AccessToken,
			},
			200,
			"should update array fields",
			nil,
		},
		{
			"update non-existent robot",
			func() (string, string) { return "non-existent-member-id", "" },
			map[string]interface{}{
				"name": "Should Fail",
			},
			map[string]string{
				"Authorization": "Bearer " + tokenInfo.AccessToken,
			},
			404,
			"should return not found for non-existent robot",
			nil,
		},
		{
			"update regular user member as robot",
			func() (string, string) {
				// Create a regular user member instead of robot
				memberID := createTestMember(t, serverURL, baseURL, teamID, tokenInfo.AccessToken, "regular-user-"+testUUID)
				return memberID, ""
			},
			map[string]interface{}{
				"name": "Should Fail",
			},
			map[string]string{
				"Authorization": "Bearer " + tokenInfo.AccessToken,
			},
			400,
			"should reject updating non-robot member",
			nil,
		},
		{
			"update robot in non-existent team",
			func() (string, string) { return createTestRobot("10") },
			map[string]interface{}{
				"name": "Should Fail",
			},
			map[string]string{
				"Authorization": "Bearer " + tokenInfo.AccessToken,
			},
			404,
			"should return not found for non-existent team",
			nil,
		},
		{
			"update robot with invalid JSON",
			func() (string, string) { return createTestRobot("11") },
			nil, // Will send invalid JSON
			map[string]string{
				"Authorization": "Bearer " + tokenInfo.AccessToken,
			},
			400,
			"should handle invalid JSON",
			nil,
		},
		{
			"update robot with empty body",
			func() (string, string) { return createTestRobot("12") },
			map[string]interface{}{},
			map[string]string{
				"Authorization": "Bearer " + tokenInfo.AccessToken,
			},
			200,
			"should handle empty update (no-op)",
			nil,
		},
		{
			"update robot avatar",
			func() (string, string) { return createTestRobot("13") },
			map[string]interface{}{
				"name":        "Robot with Avatar",
				"robot_email": fmt.Sprintf("robot-avatar-%s@robot.test.com", testUUID),
				"avatar":      fmt.Sprintf("https://example.com/avatars/robot-%s.png", testUUID),
			},
			map[string]string{
				"Authorization": "Bearer " + tokenInfo.AccessToken,
			},
			200,
			"should update robot avatar successfully",
			func(t *testing.T, memberID string) {
				// Verify the avatar was updated
				getMemberURL := serverURL + baseURL + "/user/teams/" + teamID + "/members/" + memberID
				getReq, _ := http.NewRequest("GET", getMemberURL, nil)
				getReq.Header.Set("Authorization", "Bearer "+tokenInfo.AccessToken)
				client := &http.Client{}
				getResp, err := client.Do(getReq)
				assert.NoError(t, err)
				if getResp != nil {
					defer getResp.Body.Close()
					if getResp.StatusCode == 200 {
						var member map[string]interface{}
						body, _ := io.ReadAll(getResp.Body)
						json.Unmarshal(body, &member)
						assert.Equal(t, "Robot with Avatar", member["display_name"])
						assert.Equal(t, fmt.Sprintf("https://example.com/avatars/robot-%s.png", testUUID), member["avatar"], "Should have correct avatar URL")
					}
				}
			},
		},
		{
			"update only robot avatar",
			func() (string, string) { return createTestRobot("14") },
			map[string]interface{}{
				"avatar": fmt.Sprintf("https://example.com/avatars/updated-%s.png", testUUID),
			},
			map[string]string{
				"Authorization": "Bearer " + tokenInfo.AccessToken,
			},
			200,
			"should update only avatar without affecting other fields",
			func(t *testing.T, memberID string) {
				// Verify only avatar was updated
				getMemberURL := serverURL + baseURL + "/user/teams/" + teamID + "/members/" + memberID
				getReq, _ := http.NewRequest("GET", getMemberURL, nil)
				getReq.Header.Set("Authorization", "Bearer "+tokenInfo.AccessToken)
				client := &http.Client{}
				getResp, err := client.Do(getReq)
				assert.NoError(t, err)
				if getResp != nil {
					defer getResp.Body.Close()
					if getResp.StatusCode == 200 {
						var member map[string]interface{}
						body, _ := io.ReadAll(getResp.Body)
						json.Unmarshal(body, &member)
						// Avatar should be updated
						assert.Equal(t, fmt.Sprintf("https://example.com/avatars/updated-%s.png", testUUID), member["avatar"], "Should have updated avatar URL")
						// Original fields should remain
						assert.Equal(t, "Test Robot 14", member["display_name"], "Name should remain unchanged")
						assert.Equal(t, "gpt-3.5-turbo", member["language_model"], "LLM should remain unchanged")
					}
				}
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			memberID, _ := tc.setupFunc()

			// Use non-existent team ID for the specific test case
			targetTeamID := teamID
			if tc.name == "update robot in non-existent team" {
				targetTeamID = "non-existent-team-id"
			}

			requestURL := serverURL + baseURL + "/user/teams/" + targetTeamID + "/members/robots/" + memberID

			var req *http.Request
			var err error

			if tc.body == nil {
				// Send invalid JSON for invalid JSON test case
				req, err = http.NewRequest("PUT", requestURL, bytes.NewBufferString("invalid json"))
			} else {
				bodyBytes, _ := json.Marshal(tc.body)
				req, err = http.NewRequest("PUT", requestURL, bytes.NewBuffer(bodyBytes))
			}
			assert.NoError(t, err, "Should create HTTP request")

			req.Header.Set("Content-Type", "application/json")

			// Add headers
			for key, value := range tc.headers {
				req.Header.Set(key, value)
			}

			client := &http.Client{}
			resp, err := client.Do(req)
			assert.NoError(t, err, "HTTP request should succeed")

			if resp != nil {
				defer resp.Body.Close()
				assert.Equal(t, tc.expectCode, resp.StatusCode, "Expected status code %d for %s", tc.expectCode, tc.name)

				body, err := io.ReadAll(resp.Body)
				assert.NoError(t, err, "Should read response body")

				if resp.StatusCode == 200 {
					// Parse response as success message
					var response map[string]interface{}
					err = json.Unmarshal(body, &response)
					assert.NoError(t, err, "Should parse JSON response")

					assert.Contains(t, response, "message", "Should have success message")
					assert.Equal(t, "Robot member updated successfully", response["message"], "Should have correct success message")

					// Run custom validation if provided
					if tc.validateFn != nil {
						tc.validateFn(t, memberID)
					}
				}

				t.Logf("Robot member update test %s: status=%d, body=%s", tc.name, resp.StatusCode, string(body))
			}
		})
	}
}

// TestMemberProfileGet tests the GET /user/teams/:team_id/members/:user_id/profile endpoint
func TestMemberProfileGet(t *testing.T) {
	// Initialize test environment
	serverURL := testutils.Prepare(t)
	defer testutils.Clean()

	// Get base URL from server config
	baseURL := ""
	if openapi.Server != nil && openapi.Server.Config != nil {
		baseURL = openapi.Server.Config.BaseURL
	}

	// Register a test client for OAuth authentication
	testClient := testutils.RegisterTestClient(t, "Member Profile Get Test Client", []string{"https://localhost/callback"})
	defer testutils.CleanupTestClient(t, testClient.ClientID)

	// Obtain access token with root permission
	tokenInfo := testutils.ObtainAccessTokenWithRootPermission(t, serverURL, testClient.ClientID, testClient.ClientSecret, "https://localhost/callback", "openid profile system:root")

	// Create a test team
	createdTeam := createTestTeam(t, serverURL, baseURL, tokenInfo.AccessToken, "Member Profile Get Test Team")
	teamID := getTeamID(createdTeam)

	// The creator is automatically a member, so we can use their user_id
	userID := tokenInfo.UserID

	// Update the member profile first to have test data
	provider := testutils.GetUserProvider(t)
	ctx := context.Background()
	updateData := maps.MapStrAny{
		"display_name": "Test Display Name",
		"bio":          "Test bio description",
		"avatar":       "https://example.com/test-avatar.png",
		"email":        "test-member@example.com",
	}
	err := provider.UpdateMember(ctx, teamID, userID, updateData)
	assert.NoError(t, err, "Should update member profile for testing")

	testCases := []struct {
		name       string
		teamID     string
		userID     string
		headers    map[string]string
		expectCode int
		expectMsg  string
		validateFn func(*testing.T, map[string]interface{}) // Optional validation function
	}{
		{
			"get profile without authentication",
			teamID,
			userID,
			map[string]string{},
			401,
			"should require authentication",
			nil,
		},
		{
			"get own profile successfully",
			teamID,
			userID,
			map[string]string{
				"Authorization": "Bearer " + tokenInfo.AccessToken,
			},
			200,
			"should return own profile successfully",
			func(t *testing.T, profile map[string]interface{}) {
				// Verify profile structure
				assert.Contains(t, profile, "user_id", "Should have user_id")
				assert.Contains(t, profile, "team_id", "Should have team_id")
				assert.Contains(t, profile, "display_name", "Should have display_name")
				assert.Contains(t, profile, "bio", "Should have bio")
				assert.Contains(t, profile, "avatar", "Should have avatar")
				assert.Contains(t, profile, "email", "Should have email")

				// Verify values
				assert.Equal(t, userID, profile["user_id"], "Should have correct user_id")
				assert.Equal(t, teamID, profile["team_id"], "Should have correct team_id")
				assert.Equal(t, "Test Display Name", profile["display_name"], "Should have correct display_name")
				assert.Equal(t, "Test bio description", profile["bio"], "Should have correct bio")
				assert.Equal(t, "https://example.com/test-avatar.png", profile["avatar"], "Should have correct avatar")
				assert.Equal(t, "test-member@example.com", profile["email"], "Should have correct email")
			},
		},
		{
			"get profile from non-existent team",
			"non-existent-team-id",
			userID,
			map[string]string{
				"Authorization": "Bearer " + tokenInfo.AccessToken,
			},
			404,
			"should return not found for non-existent team",
			nil,
		},
		{
			"get profile for non-existent user",
			teamID,
			"non-existent-user-id",
			map[string]string{
				"Authorization": "Bearer " + tokenInfo.AccessToken,
			},
			404,
			"should return not found for non-existent user",
			nil,
		},
		{
			"get profile with minimal data",
			teamID,
			userID,
			map[string]string{
				"Authorization": "Bearer " + tokenInfo.AccessToken,
			},
			200,
			"should return profile even with minimal data",
			func(t *testing.T, profile map[string]interface{}) {
				// Should always have these fields, even if empty
				assert.Contains(t, profile, "user_id", "Should have user_id field")
				assert.Contains(t, profile, "team_id", "Should have team_id field")
				assert.Contains(t, profile, "display_name", "Should have display_name field")
				assert.Contains(t, profile, "bio", "Should have bio field")
				assert.Contains(t, profile, "avatar", "Should have avatar field")
				assert.Contains(t, profile, "email", "Should have email field")
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			requestURL := serverURL + baseURL + "/user/teams/" + tc.teamID + "/members/" + tc.userID + "/profile"

			req, err := http.NewRequest("GET", requestURL, nil)
			assert.NoError(t, err, "Should create HTTP request")

			// Add headers
			for key, value := range tc.headers {
				req.Header.Set(key, value)
			}

			client := &http.Client{}
			resp, err := client.Do(req)
			assert.NoError(t, err, "HTTP request should succeed")

			if resp != nil {
				defer resp.Body.Close()
				assert.Equal(t, tc.expectCode, resp.StatusCode, "Expected status code %d for %s", tc.expectCode, tc.name)

				body, err := io.ReadAll(resp.Body)
				assert.NoError(t, err, "Should read response body")

				if resp.StatusCode == 200 {
					// Parse response as profile object
					var profile map[string]interface{}
					err = json.Unmarshal(body, &profile)
					assert.NoError(t, err, "Should parse JSON response")

					// Run custom validation if provided
					if tc.validateFn != nil {
						tc.validateFn(t, profile)
					}
				}

				t.Logf("Member profile get test %s: status=%d, body=%s", tc.name, resp.StatusCode, string(body))
			}
		})
	}
}

// TestMemberProfileUpdate tests the PUT /user/teams/:team_id/members/:user_id/profile endpoint
func TestMemberProfileUpdate(t *testing.T) {
	// Initialize test environment
	serverURL := testutils.Prepare(t)
	defer testutils.Clean()

	// Get base URL from server config
	baseURL := ""
	if openapi.Server != nil && openapi.Server.Config != nil {
		baseURL = openapi.Server.Config.BaseURL
	}

	// Register a test client for OAuth authentication
	testClient := testutils.RegisterTestClient(t, "Member Profile Update Test Client", []string{"https://localhost/callback"})
	defer testutils.CleanupTestClient(t, testClient.ClientID)

	// Obtain access token with root permission and explicit member profile scope
	tokenInfo := testutils.ObtainAccessTokenWithRootPermission(t, serverURL, testClient.ClientID, testClient.ClientSecret, "https://localhost/callback", "openid profile system:root member:profile:update:own")

	// Create a test team
	createdTeam := createTestTeam(t, serverURL, baseURL, tokenInfo.AccessToken, "Member Profile Update Test Team")
	teamID := getTeamID(createdTeam)

	// The creator is automatically a member, so we can use their user_id
	userID := tokenInfo.UserID

	testCases := []struct {
		name       string
		teamID     string
		userID     string
		body       map[string]interface{}
		headers    map[string]string
		expectCode int
		expectMsg  string
		validateFn func(*testing.T, string) // Optional validation function with userID
	}{
		{
			"update profile without authentication",
			teamID,
			userID,
			map[string]interface{}{
				"display_name": "New Name",
			},
			map[string]string{},
			401,
			"should require authentication",
			nil,
		},
		{
			"update display_name",
			teamID,
			userID,
			map[string]interface{}{
				"display_name": "Updated Display Name",
			},
			map[string]string{
				"Authorization": "Bearer " + tokenInfo.AccessToken,
			},
			200,
			"should update display_name successfully",
			func(t *testing.T, uid string) {
				// Verify the update by getting member details
				provider := testutils.GetUserProvider(t)
				member, err := provider.GetMember(context.Background(), teamID, uid)
				assert.NoError(t, err)
				assert.Equal(t, "Updated Display Name", member["display_name"])
			},
		},
		{
			"update bio",
			teamID,
			userID,
			map[string]interface{}{
				"bio": "This is my updated bio",
			},
			map[string]string{
				"Authorization": "Bearer " + tokenInfo.AccessToken,
			},
			200,
			"should update bio successfully",
			func(t *testing.T, uid string) {
				provider := testutils.GetUserProvider(t)
				member, err := provider.GetMember(context.Background(), teamID, uid)
				assert.NoError(t, err)
				assert.Equal(t, "This is my updated bio", member["bio"])
			},
		},
		{
			"update avatar",
			teamID,
			userID,
			map[string]interface{}{
				"avatar": "https://example.com/avatar.png",
			},
			map[string]string{
				"Authorization": "Bearer " + tokenInfo.AccessToken,
			},
			200,
			"should update avatar successfully",
			func(t *testing.T, uid string) {
				provider := testutils.GetUserProvider(t)
				member, err := provider.GetMember(context.Background(), teamID, uid)
				assert.NoError(t, err)
				assert.Equal(t, "https://example.com/avatar.png", member["avatar"])
			},
		},
		{
			"update email",
			teamID,
			userID,
			map[string]interface{}{
				"email": "newemail@example.com",
			},
			map[string]string{
				"Authorization": "Bearer " + tokenInfo.AccessToken,
			},
			200,
			"should update email successfully",
			func(t *testing.T, uid string) {
				provider := testutils.GetUserProvider(t)
				member, err := provider.GetMember(context.Background(), teamID, uid)
				assert.NoError(t, err)
				assert.Equal(t, "newemail@example.com", member["email"])
			},
		},
		{
			"update all fields at once",
			teamID,
			userID,
			map[string]interface{}{
				"display_name": "Complete Update",
				"bio":          "All fields updated",
				"avatar":       "https://example.com/complete.png",
				"email":        "complete@example.com",
			},
			map[string]string{
				"Authorization": "Bearer " + tokenInfo.AccessToken,
			},
			200,
			"should update all fields successfully",
			func(t *testing.T, uid string) {
				provider := testutils.GetUserProvider(t)
				member, err := provider.GetMember(context.Background(), teamID, uid)
				assert.NoError(t, err)
				assert.Equal(t, "Complete Update", member["display_name"])
				assert.Equal(t, "All fields updated", member["bio"])
				assert.Equal(t, "https://example.com/complete.png", member["avatar"])
				assert.Equal(t, "complete@example.com", member["email"])
			},
		},
		{
			"update with empty body",
			teamID,
			userID,
			map[string]interface{}{},
			map[string]string{
				"Authorization": "Bearer " + tokenInfo.AccessToken,
			},
			400,
			"should reject empty update",
			nil,
		},
		{
			"update other user's profile should fail",
			teamID,
			"other-user-id",
			map[string]interface{}{
				"display_name": "Should Fail",
			},
			map[string]string{
				"Authorization": "Bearer " + tokenInfo.AccessToken,
			},
			404,
			"should return not found for non-existent user (member not found)",
			nil,
		},
		{
			"update profile in non-existent team",
			"non-existent-team-id",
			userID,
			map[string]interface{}{
				"display_name": "Should Fail",
			},
			map[string]string{
				"Authorization": "Bearer " + tokenInfo.AccessToken,
			},
			404,
			"should return not found for non-existent team",
			nil,
		},
		{
			"update with invalid JSON",
			teamID,
			userID,
			nil, // Will send invalid JSON
			map[string]string{
				"Authorization": "Bearer " + tokenInfo.AccessToken,
			},
			400,
			"should handle invalid JSON",
			nil,
		},
		{
			"partial update - single field",
			teamID,
			userID,
			map[string]interface{}{
				"display_name": "Partial Update",
			},
			map[string]string{
				"Authorization": "Bearer " + tokenInfo.AccessToken,
			},
			200,
			"should handle partial update with single field",
			func(t *testing.T, uid string) {
				provider := testutils.GetUserProvider(t)
				member, err := provider.GetMember(context.Background(), teamID, uid)
				assert.NoError(t, err)
				assert.Equal(t, "Partial Update", member["display_name"])
				// Other fields should remain unchanged
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			requestURL := serverURL + baseURL + "/user/teams/" + tc.teamID + "/members/" + tc.userID + "/profile"

			var req *http.Request
			var err error

			if tc.body == nil {
				// Send invalid JSON for invalid JSON test case
				req, err = http.NewRequest("PUT", requestURL, bytes.NewBufferString("invalid json"))
			} else {
				bodyBytes, _ := json.Marshal(tc.body)
				req, err = http.NewRequest("PUT", requestURL, bytes.NewBuffer(bodyBytes))
			}
			assert.NoError(t, err, "Should create HTTP request")

			req.Header.Set("Content-Type", "application/json")

			// Add headers
			for key, value := range tc.headers {
				req.Header.Set(key, value)
			}

			client := &http.Client{}
			resp, err := client.Do(req)
			assert.NoError(t, err, "HTTP request should succeed")

			if resp != nil {
				defer resp.Body.Close()
				assert.Equal(t, tc.expectCode, resp.StatusCode, "Expected status code %d for %s", tc.expectCode, tc.name)

				body, err := io.ReadAll(resp.Body)
				assert.NoError(t, err, "Should read response body")

				if resp.StatusCode == 200 {
					// Parse response as success message
					var response map[string]interface{}
					err = json.Unmarshal(body, &response)
					assert.NoError(t, err, "Should parse JSON response")

					assert.Contains(t, response, "user_id", "Should have user_id")
					assert.Contains(t, response, "message", "Should have success message")
					assert.Equal(t, tc.userID, response["user_id"], "Should have correct user_id")
					assert.Equal(t, "Member profile updated successfully", response["message"], "Should have correct success message")

					// Run custom validation if provided
					if tc.validateFn != nil {
						tc.validateFn(t, tc.userID)
					}
				}

				t.Logf("Member profile update test %s: status=%d, body=%s", tc.name, resp.StatusCode, string(body))
			}
		})
	}
}

// Note: getTeamID function is already defined in team_test.go
