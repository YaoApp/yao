package user_test

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
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

	testCases := []struct {
		name       string
		teamID     string
		query      string
		headers    map[string]string
		expectCode int
		expectMsg  string
	}{
		{
			"list members without authentication",
			teamID,
			"",
			map[string]string{},
			401,
			"should require authentication",
		},
		{
			"list members with valid token",
			teamID,
			"",
			map[string]string{
				"Authorization": "Bearer " + tokenInfo.AccessToken,
			},
			200,
			"should return team members",
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

			client := &http.Client{}
			resp, err := client.Do(req)
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

					// Verify that creator is automatically added as member
					if data, ok := response["data"].([]interface{}); ok {
						assert.GreaterOrEqual(t, len(data), 1, "Should have at least the owner as member")
						if len(data) > 0 {
							member := data[0].(map[string]interface{})
							assert.Equal(t, tokenInfo.UserID, member["user_id"], "Owner should be in member list")
							assert.Equal(t, "owner", member["role_id"], "Creator should have owner role")
						}
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

// TestMemberCreateDirect tests the POST /user/teams/:team_id/members/direct endpoint
func TestMemberCreateDirect(t *testing.T) {
	// Initialize test environment
	serverURL := testutils.Prepare(t)
	defer testutils.Clean()

	// Get base URL from server config
	baseURL := ""
	if openapi.Server != nil && openapi.Server.Config != nil {
		baseURL = openapi.Server.Config.BaseURL
	}

	// Register a test client for OAuth authentication
	testClient := testutils.RegisterTestClient(t, "Member Create Test Client", []string{"https://localhost/callback"})
	defer testutils.CleanupTestClient(t, testClient.ClientID)

	// Obtain access token for authenticated requests
	tokenInfo := testutils.ObtainAccessToken(t, serverURL, testClient.ClientID, testClient.ClientSecret, "https://localhost/callback", "openid profile")

	// Create a test team
	createdTeam := createTestTeam(t, serverURL, baseURL, tokenInfo.AccessToken, "Member Create Test Team")
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
			"create member without authentication",
			teamID,
			map[string]interface{}{
				"user_id": "test-user-123",
				"role_id": "member",
			},
			map[string]string{},
			401,
			"should require authentication",
		},
		{
			"create member with valid data",
			teamID,
			map[string]interface{}{
				"user_id":     "test-user-123",
				"member_type": "user",
				"role_id":     "member",
			},
			map[string]string{
				"Authorization": "Bearer " + tokenInfo.AccessToken,
			},
			201,
			"should create member successfully",
		},
		{
			"create member with settings",
			teamID,
			map[string]interface{}{
				"user_id":     "test-user-456",
				"member_type": "user",
				"role_id":     "admin",
				"settings": map[string]interface{}{
					"notifications": true,
					"permissions":   []string{"read", "write"},
				},
			},
			map[string]string{
				"Authorization": "Bearer " + tokenInfo.AccessToken,
			},
			201,
			"should create member with settings",
		},
		{
			"create member without user_id",
			teamID,
			map[string]interface{}{
				"role_id": "member",
			},
			map[string]string{
				"Authorization": "Bearer " + tokenInfo.AccessToken,
			},
			400,
			"should require user_id",
		},
		{
			"create member without role_id",
			teamID,
			map[string]interface{}{
				"user_id": "test-user-789",
			},
			map[string]string{
				"Authorization": "Bearer " + tokenInfo.AccessToken,
			},
			400,
			"should require role_id",
		},
		{
			"create duplicate member",
			teamID,
			map[string]interface{}{
				"user_id": "test-user-123", // Same as first successful case
				"role_id": "member",
			},
			map[string]string{
				"Authorization": "Bearer " + tokenInfo.AccessToken,
			},
			409,
			"should reject duplicate member",
		},
		{
			"create member in non-existent team",
			"non-existent-team-id",
			map[string]interface{}{
				"user_id": "test-user-999",
				"role_id": "member",
			},
			map[string]string{
				"Authorization": "Bearer " + tokenInfo.AccessToken,
			},
			404,
			"should return not found for non-existent team",
		},
		{
			"create member with invalid JSON",
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
			requestURL := serverURL + baseURL + "/user/teams/" + tc.teamID + "/members/direct"

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
				}

				t.Logf("Member create test %s: status=%d, body=%s", tc.name, resp.StatusCode, string(body))
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
			"owner can create members",
			"/user/teams/" + teamID + "/members/direct",
			"POST",
			ownerToken.AccessToken,
			201, // Will create successfully
			"owner should be able to create members",
		},
		{
			"member cannot create members",
			"/user/teams/" + teamID + "/members/direct",
			"POST",
			nonOwnerToken.AccessToken,
			403,
			"member should not be able to create members",
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

// createTestMember creates a member for testing and returns the user_id (which serves as member_id in API context)
func createTestMember(t *testing.T, serverURL, baseURL, teamID, accessToken, userID string) string {
	createMemberBody := map[string]interface{}{
		"user_id":     userID,
		"member_type": "user",
		"role_id":     "member",
	}

	bodyBytes, err := json.Marshal(createMemberBody)
	assert.NoError(t, err, "Should marshal member creation body")

	req, err := http.NewRequest("POST", serverURL+baseURL+"/user/teams/"+teamID+"/members/direct", bytes.NewBuffer(bodyBytes))
	assert.NoError(t, err, "Should create member creation request")

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+accessToken)

	client := &http.Client{}
	resp, err := client.Do(req)
	assert.NoError(t, err, "Should send member creation request")
	defer resp.Body.Close()

	assert.Equal(t, 201, resp.StatusCode, "Should create member successfully")

	body, err := io.ReadAll(resp.Body)
	assert.NoError(t, err, "Should read member creation response")

	var response map[string]interface{}
	err = json.Unmarshal(body, &response)
	assert.NoError(t, err, "Should parse member creation response")

	_, ok := response["member_id"]
	assert.True(t, ok, "Should have member_id in response")

	// For API purposes, the member_id is the user_id in the context of team_id
	// So we return the user_id that was used to create the member
	return userID
}

// getOwnerMemberID gets the user_id of the team owner (which serves as member_id in API context)
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

	// Find the owner member and return their user_id
	for _, item := range data {
		member := item.(map[string]interface{})
		if role, ok := member["role_id"].(string); ok && role == "owner" {
			userID, ok := member["user_id"].(string)
			if !ok {
				t.Fatal("Owner member missing user_id")
			}
			return userID
		}
	}

	t.Fatal("Could not find owner member")
	return ""
}

// Note: getTeamID function is already defined in team_test.go
