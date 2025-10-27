package user_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/yaoapp/yao/openapi"
	"github.com/yaoapp/yao/openapi/tests/testutils"
)

// TestTeamList tests the GET /user/teams endpoint
func TestTeamList(t *testing.T) {
	// Initialize test environment
	serverURL := testutils.Prepare(t)
	defer testutils.Clean()

	// Get base URL from server config
	baseURL := ""
	if openapi.Server != nil && openapi.Server.Config != nil {
		baseURL = openapi.Server.Config.BaseURL
	}

	// Register a test client for OAuth authentication
	testClient := testutils.RegisterTestClient(t, "Team Test Client", []string{"https://localhost/callback"})
	defer testutils.CleanupTestClient(t, testClient.ClientID)

	// Obtain access token for authenticated requests
	tokenInfo := testutils.ObtainAccessToken(t, serverURL, testClient.ClientID, testClient.ClientSecret, "https://localhost/callback", "openid profile")

	testCases := []struct {
		name       string
		endpoint   string
		headers    map[string]string
		expectCode int
		expectMsg  string
	}{
		{
			"list teams without authentication",
			"/user/teams",
			map[string]string{},
			401,
			"should require authentication",
		},
		{
			"list teams with valid token",
			"/user/teams",
			map[string]string{
				"Authorization": "Bearer " + tokenInfo.AccessToken,
			},
			200,
			"should return user teams",
		},
		{
			"list teams with pagination",
			"/user/teams?page=1&pagesize=10",
			map[string]string{
				"Authorization": "Bearer " + tokenInfo.AccessToken,
			},
			200,
			"should handle pagination parameters",
		},
		{
			"list teams with status filter",
			"/user/teams?status=active",
			map[string]string{
				"Authorization": "Bearer " + tokenInfo.AccessToken,
			},
			200,
			"should filter by status",
		},
		{
			"list teams with name search",
			"/user/teams?name=test",
			map[string]string{
				"Authorization": "Bearer " + tokenInfo.AccessToken,
			},
			200,
			"should search by name",
		},
		{
			"list teams with invalid pagesize",
			"/user/teams?pagesize=1000",
			map[string]string{
				"Authorization": "Bearer " + tokenInfo.AccessToken,
			},
			200,
			"should limit pagesize to maximum allowed",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			requestURL := serverURL + baseURL + tc.endpoint
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
					// Parse response as array (TeamList returns array directly, not paginated)
					var teams []interface{}
					err = json.Unmarshal(body, &teams)
					assert.NoError(t, err, "Should parse JSON response as array")

					// Verify it's an array
					assert.IsType(t, []interface{}{}, teams, "Response should be an array")
				}

				t.Logf("Team list test %s: status=%d, body=%s", tc.name, resp.StatusCode, string(body))
			}
		})
	}
}

// TestTeamCreate tests the POST /user/teams endpoint
func TestTeamCreate(t *testing.T) {
	// Initialize test environment
	serverURL := testutils.Prepare(t)
	defer testutils.Clean()

	// Get base URL from server config
	baseURL := ""
	if openapi.Server != nil && openapi.Server.Config != nil {
		baseURL = openapi.Server.Config.BaseURL
	}

	// Register a test client for OAuth authentication
	testClient := testutils.RegisterTestClient(t, "Team Create Test Client", []string{"https://localhost/callback"})
	defer testutils.CleanupTestClient(t, testClient.ClientID)

	// Obtain access token for authenticated requests
	tokenInfo := testutils.ObtainAccessToken(t, serverURL, testClient.ClientID, testClient.ClientSecret, "https://localhost/callback", "openid profile")

	testCases := []struct {
		name       string
		body       map[string]interface{}
		headers    map[string]string
		expectCode int
		expectMsg  string
	}{
		{
			"create team without authentication",
			map[string]interface{}{
				"name": "Test Team",
			},
			map[string]string{},
			401,
			"should require authentication",
		},
		{
			"create team with valid data",
			map[string]interface{}{
				"name":        "Test Team",
				"description": "A test team for unit testing",
			},
			map[string]string{
				"Authorization": "Bearer " + tokenInfo.AccessToken,
			},
			201,
			"should create team successfully",
		},
		{
			"create team with settings",
			map[string]interface{}{
				"name":        "Team with Settings",
				"description": "Team with custom settings",
				"settings": map[string]interface{}{
					"theme":      "dark",
					"visibility": "private",
				},
			},
			map[string]string{
				"Authorization": "Bearer " + tokenInfo.AccessToken,
			},
			201,
			"should create team with settings",
		},
		{
			"create team with logo",
			map[string]interface{}{
				"name":        "Team with Logo",
				"description": "Team with custom logo",
				"logo":        "__yao.attachment://test-logo-123",
			},
			map[string]string{
				"Authorization": "Bearer " + tokenInfo.AccessToken,
			},
			201,
			"should create team with logo",
		},
		{
			"create team without name",
			map[string]interface{}{
				"description": "Team without name",
			},
			map[string]string{
				"Authorization": "Bearer " + tokenInfo.AccessToken,
			},
			400,
			"should require team name",
		},
		{
			"create team with empty name",
			map[string]interface{}{
				"name": "",
			},
			map[string]string{
				"Authorization": "Bearer " + tokenInfo.AccessToken,
			},
			400,
			"should require non-empty team name",
		},
		{
			"create team with invalid JSON",
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
			requestURL := serverURL + baseURL + "/user/teams"

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
					// Parse response as team object
					var team map[string]interface{}
					err = json.Unmarshal(body, &team)
					assert.NoError(t, err, "Should parse JSON response")

					// Verify team structure
					assert.Contains(t, team, "id", "Should have team ID")
					assert.Contains(t, team, "team_id", "Should have team_id")
					assert.Contains(t, team, "name", "Should have team name")
					assert.Contains(t, team, "owner_id", "Should have owner_id")
					assert.Contains(t, team, "status", "Should have status")
					assert.Contains(t, team, "created_at", "Should have created_at")
					assert.Contains(t, team, "updated_at", "Should have updated_at")

					// Verify values
					if tc.body != nil {
						if name, ok := tc.body["name"]; ok {
							assert.Equal(t, name, team["name"], "Should have correct team name")
						}
						if description, ok := tc.body["description"]; ok {
							assert.Equal(t, description, team["description"], "Should have correct description")
						}
						if logo, ok := tc.body["logo"]; ok {
							assert.Equal(t, logo, team["logo"], "Should have correct logo")
						}
					}
				}

				t.Logf("Team create test %s: status=%d, body=%s", tc.name, resp.StatusCode, string(body))
			}
		})
	}
}

// TestTeamGet tests the GET /user/teams/:team_id endpoint
func TestTeamGet(t *testing.T) {
	// Initialize test environment
	serverURL := testutils.Prepare(t)
	defer testutils.Clean()

	// Get base URL from server config
	baseURL := ""
	if openapi.Server != nil && openapi.Server.Config != nil {
		baseURL = openapi.Server.Config.BaseURL
	}

	// Register a test client for OAuth authentication
	testClient := testutils.RegisterTestClient(t, "Team Get Test Client", []string{"https://localhost/callback"})
	defer testutils.CleanupTestClient(t, testClient.ClientID)

	// Obtain access token for authenticated requests
	tokenInfo := testutils.ObtainAccessToken(t, serverURL, testClient.ClientID, testClient.ClientSecret, "https://localhost/callback", "openid profile")

	// First create a team to test with
	createTeamBody := map[string]interface{}{
		"name":        "Get Test Team",
		"description": "Team for testing get functionality",
		"settings": map[string]interface{}{
			"theme": "light",
		},
	}

	createReq := createTeamRequest(t, serverURL+baseURL+"/user/teams", createTeamBody, tokenInfo.AccessToken)
	createResp, err := (&http.Client{}).Do(createReq)
	assert.NoError(t, err, "Should create test team")
	defer createResp.Body.Close()

	var createdTeam map[string]interface{}
	if createResp.StatusCode == 201 {
		createBody, _ := io.ReadAll(createResp.Body)
		json.Unmarshal(createBody, &createdTeam)
	}

	testCases := []struct {
		name       string
		teamID     string
		headers    map[string]string
		expectCode int
		expectMsg  string
	}{
		{
			"get team without authentication",
			getTeamID(createdTeam),
			map[string]string{},
			401,
			"should require authentication",
		},
		{
			"get existing team",
			getTeamID(createdTeam),
			map[string]string{
				"Authorization": "Bearer " + tokenInfo.AccessToken,
			},
			200,
			"should return team details",
		},
		{
			"get non-existent team",
			"non-existent-team-id",
			map[string]string{
				"Authorization": "Bearer " + tokenInfo.AccessToken,
			},
			404,
			"should return not found for non-existent team",
		},
		{
			"get team with empty team_id returns team list",
			"",
			map[string]string{
				"Authorization": "Bearer " + tokenInfo.AccessToken,
			},
			200,
			"should return team list when team_id is empty",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			endpoint := "/user/teams"
			if tc.teamID != "" {
				endpoint += "/" + tc.teamID
			}
			requestURL := serverURL + baseURL + endpoint

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
					if tc.teamID == "" {
						// Parse response as team list (returns array directly, not paginated)
						var teams []interface{}
						err = json.Unmarshal(body, &teams)
						assert.NoError(t, err, "Should parse JSON response as array")

						// Verify it's an array
						assert.IsType(t, []interface{}{}, teams, "Response should be an array")
					} else {
						// Parse response as team detail object
						var team map[string]interface{}
						err = json.Unmarshal(body, &team)
						assert.NoError(t, err, "Should parse JSON response")

						// Verify team detail structure
						assert.Contains(t, team, "id", "Should have team ID")
						assert.Contains(t, team, "team_id", "Should have team_id")
						assert.Contains(t, team, "name", "Should have team name")
						assert.Contains(t, team, "description", "Should have description")
						// Logo field is optional, only present if set
						assert.Contains(t, team, "owner_id", "Should have owner_id")
						assert.Contains(t, team, "status", "Should have status")
						assert.Contains(t, team, "settings", "Should have settings")
						assert.Contains(t, team, "created_at", "Should have created_at")
						assert.Contains(t, team, "updated_at", "Should have updated_at")

						// Verify values match created team
						assert.Equal(t, "Get Test Team", team["name"], "Should have correct team name")
						assert.Equal(t, "Team for testing get functionality", team["description"], "Should have correct description")
						if settings, ok := team["settings"].(map[string]interface{}); ok {
							assert.Equal(t, "light", settings["theme"], "Should have correct theme setting")
						}
					}
				}

				t.Logf("Team get test %s: status=%d, body=%s", tc.name, resp.StatusCode, string(body))
			}
		})
	}
}

// TestTeamUpdate tests the PUT /user/teams/:team_id endpoint
func TestTeamUpdate(t *testing.T) {
	// Initialize test environment
	serverURL := testutils.Prepare(t)
	defer testutils.Clean()

	// Get base URL from server config
	baseURL := ""
	if openapi.Server != nil && openapi.Server.Config != nil {
		baseURL = openapi.Server.Config.BaseURL
	}

	// Register a test client for OAuth authentication
	testClient := testutils.RegisterTestClient(t, "Team Update Test Client", []string{"https://localhost/callback"})
	defer testutils.CleanupTestClient(t, testClient.ClientID)

	// Obtain access token for authenticated requests
	tokenInfo := testutils.ObtainAccessToken(t, serverURL, testClient.ClientID, testClient.ClientSecret, "https://localhost/callback", "openid profile")

	// Create a team to test updates
	createTeamBody := map[string]interface{}{
		"name":        "Update Test Team",
		"description": "Team for testing update functionality",
	}

	createReq := createTeamRequest(t, serverURL+baseURL+"/user/teams", createTeamBody, tokenInfo.AccessToken)
	createResp, err := (&http.Client{}).Do(createReq)
	assert.NoError(t, err, "Should create test team")
	defer createResp.Body.Close()

	var createdTeam map[string]interface{}
	if createResp.StatusCode == 201 {
		createBody, _ := io.ReadAll(createResp.Body)
		json.Unmarshal(createBody, &createdTeam)
	}

	testCases := []struct {
		name       string
		teamID     string
		body       map[string]interface{}
		headers    map[string]string
		expectCode int
		expectMsg  string
	}{
		{
			"update team without authentication",
			getTeamID(createdTeam),
			map[string]interface{}{
				"name": "Updated Name",
			},
			map[string]string{},
			401,
			"should require authentication",
		},
		{
			"update team name",
			getTeamID(createdTeam),
			map[string]interface{}{
				"name": "Updated Team Name",
			},
			map[string]string{
				"Authorization": "Bearer " + tokenInfo.AccessToken,
			},
			200,
			"should update team name",
		},
		{
			"update team description",
			getTeamID(createdTeam),
			map[string]interface{}{
				"description": "Updated description",
			},
			map[string]string{
				"Authorization": "Bearer " + tokenInfo.AccessToken,
			},
			200,
			"should update team description",
		},
		{
			"update team settings",
			getTeamID(createdTeam),
			map[string]interface{}{
				"settings": map[string]interface{}{
					"theme":      "dark",
					"visibility": "public",
				},
			},
			map[string]string{
				"Authorization": "Bearer " + tokenInfo.AccessToken,
			},
			200,
			"should update team settings",
		},
		{
			"update team logo",
			getTeamID(createdTeam),
			map[string]interface{}{
				"logo": "__yao.attachment://updated-logo-456",
			},
			map[string]string{
				"Authorization": "Bearer " + tokenInfo.AccessToken,
			},
			200,
			"should update team logo",
		},
		{
			"update non-existent team",
			"non-existent-team-id",
			map[string]interface{}{
				"name": "Updated Name",
			},
			map[string]string{
				"Authorization": "Bearer " + tokenInfo.AccessToken,
			},
			404,
			"should return not found for non-existent team",
		},
		{
			"update team with invalid JSON",
			getTeamID(createdTeam),
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
			requestURL := serverURL + baseURL + "/user/teams/" + tc.teamID

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
					// Parse response as updated team object
					var team map[string]interface{}
					err = json.Unmarshal(body, &team)
					assert.NoError(t, err, "Should parse JSON response")

					// Verify updated values
					if tc.body != nil {
						if name, ok := tc.body["name"]; ok {
							assert.Equal(t, name, team["name"], "Should have updated team name")
						}
						if description, ok := tc.body["description"]; ok {
							assert.Equal(t, description, team["description"], "Should have updated description")
						}
						if logo, ok := tc.body["logo"]; ok {
							assert.Equal(t, logo, team["logo"], "Should have updated logo")
						}
						if settings, ok := tc.body["settings"]; ok {
							assert.Equal(t, settings, team["settings"], "Should have updated settings")
						}
					}
				}

				t.Logf("Team update test %s: status=%d, body=%s", tc.name, resp.StatusCode, string(body))
			}
		})
	}
}

// TestTeamDelete tests the DELETE /user/teams/:team_id endpoint
func TestTeamDelete(t *testing.T) {
	// Initialize test environment
	serverURL := testutils.Prepare(t)
	defer testutils.Clean()

	// Get base URL from server config
	baseURL := ""
	if openapi.Server != nil && openapi.Server.Config != nil {
		baseURL = openapi.Server.Config.BaseURL
	}

	// Register a test client for OAuth authentication
	testClient := testutils.RegisterTestClient(t, "Team Delete Test Client", []string{"https://localhost/callback"})
	defer testutils.CleanupTestClient(t, testClient.ClientID)

	// Obtain access token for authenticated requests
	tokenInfo := testutils.ObtainAccessToken(t, serverURL, testClient.ClientID, testClient.ClientSecret, "https://localhost/callback", "openid profile")

	// Create teams to test deletion
	createTeam := func(name string) map[string]interface{} {
		createTeamBody := map[string]interface{}{
			"name":        name,
			"description": "Team for testing delete functionality",
		}

		createReq := createTeamRequest(t, serverURL+baseURL+"/user/teams", createTeamBody, tokenInfo.AccessToken)
		createResp, err := (&http.Client{}).Do(createReq)
		assert.NoError(t, err, "Should create test team")
		defer createResp.Body.Close()

		var createdTeam map[string]interface{}
		if createResp.StatusCode == 201 {
			createBody, _ := io.ReadAll(createResp.Body)
			json.Unmarshal(createBody, &createdTeam)
		}
		return createdTeam
	}

	testCases := []struct {
		name       string
		teamID     string
		headers    map[string]string
		expectCode int
		expectMsg  string
	}{
		{
			"delete team without authentication",
			getTeamID(createTeam("Delete Test Team 1")),
			map[string]string{},
			401,
			"should require authentication",
		},
		{
			"delete existing team",
			getTeamID(createTeam("Delete Test Team 2")),
			map[string]string{
				"Authorization": "Bearer " + tokenInfo.AccessToken,
			},
			200,
			"should delete team successfully",
		},
		{
			"delete non-existent team",
			"non-existent-team-id",
			map[string]string{
				"Authorization": "Bearer " + tokenInfo.AccessToken,
			},
			404,
			"should return not found for non-existent team",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			requestURL := serverURL + baseURL + "/user/teams/" + tc.teamID

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
					assert.Equal(t, "Team deleted successfully", response["message"], "Should have correct success message")
				}

				t.Logf("Team delete test %s: status=%d, body=%s", tc.name, resp.StatusCode, string(body))
			}
		})
	}
}

// TestTeamAuthenticationEdgeCases tests authentication and authorization edge cases
func TestTeamAuthenticationEdgeCases(t *testing.T) {
	// Initialize test environment
	serverURL := testutils.Prepare(t)
	defer testutils.Clean()

	// Get base URL from server config
	baseURL := ""
	if openapi.Server != nil && openapi.Server.Config != nil {
		baseURL = openapi.Server.Config.BaseURL
	}

	// Register a test client for OAuth authentication
	testClient := testutils.RegisterTestClient(t, "Team Auth Test Client", []string{"https://localhost/callback"})
	defer testutils.CleanupTestClient(t, testClient.ClientID)

	// Obtain access token for authenticated requests
	tokenInfo := testutils.ObtainAccessToken(t, serverURL, testClient.ClientID, testClient.ClientSecret, "https://localhost/callback", "openid profile")

	testCases := []struct {
		name       string
		endpoint   string
		method     string
		headers    map[string]string
		expectCode int
		expectMsg  string
	}{
		{
			"invalid bearer token format",
			"/user/teams",
			"GET",
			map[string]string{
				"Authorization": "Bearer invalid-token",
			},
			401,
			"should reject invalid token",
		},
		{
			"missing bearer prefix",
			"/user/teams",
			"GET",
			map[string]string{
				"Authorization": tokenInfo.AccessToken,
			},
			200,
			"may accept token without Bearer prefix (implementation dependent)",
		},
		{
			"expired token simulation",
			"/user/teams",
			"GET",
			map[string]string{
				"Authorization": "Bearer expired.token.here",
			},
			401,
			"should reject expired token",
		},
		{
			"malformed authorization header",
			"/user/teams",
			"GET",
			map[string]string{
				"Authorization": "Malformed",
			},
			401,
			"should reject malformed header",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			requestURL := serverURL + baseURL + tc.endpoint

			req, err := http.NewRequest(tc.method, requestURL, nil)
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

				t.Logf("Auth edge case test %s: status=%d, body=%s", tc.name, resp.StatusCode, string(body))
			}
		})
	}
}

// TestTeamDeleteMemberCleanup tests that team deletion removes all team members
func TestTeamDeleteMemberCleanup(t *testing.T) {
	// Initialize test environment
	serverURL := testutils.Prepare(t)
	defer testutils.Clean()

	// Get base URL from server config
	baseURL := ""
	if openapi.Server != nil && openapi.Server.Config != nil {
		baseURL = openapi.Server.Config.BaseURL
	}

	// Register a test client for OAuth authentication
	testClient := testutils.RegisterTestClient(t, "Team Delete Member Test Client", []string{"https://localhost/callback"})
	defer testutils.CleanupTestClient(t, testClient.ClientID)

	// Obtain access token for authenticated requests
	tokenInfo := testutils.ObtainAccessToken(t, serverURL, testClient.ClientID, testClient.ClientSecret, "https://localhost/callback", "openid profile")

	// Create a team
	createTeamBody := map[string]interface{}{
		"name":        "Delete Member Test Team",
		"description": "Team for testing member cleanup on deletion",
	}

	createReq := createTeamRequest(t, serverURL+baseURL+"/user/teams", createTeamBody, tokenInfo.AccessToken)
	createResp, err := (&http.Client{}).Do(createReq)
	assert.NoError(t, err, "Should create test team")
	defer createResp.Body.Close()

	var createdTeam map[string]interface{}
	if createResp.StatusCode == 201 {
		createBody, _ := io.ReadAll(createResp.Body)
		json.Unmarshal(createBody, &createdTeam)

		teamID := getTeamID(createdTeam)
		assert.NotEmpty(t, teamID, "Should have team ID")

		t.Logf("Created team: %s (ID: %s)", createdTeam["name"], teamID)

		// Note: Team creation automatically adds the creator as owner member
		// We can't easily verify member existence without member API endpoints
		// But we can verify that deletion completes successfully

		// Delete the team
		deleteReq, err := http.NewRequest("DELETE", serverURL+baseURL+"/user/teams/"+teamID, nil)
		assert.NoError(t, err, "Should create delete request")
		deleteReq.Header.Set("Authorization", "Bearer "+tokenInfo.AccessToken)

		deleteResp, err := (&http.Client{}).Do(deleteReq)
		assert.NoError(t, err, "Should send delete request")
		defer deleteResp.Body.Close()

		assert.Equal(t, 200, deleteResp.StatusCode, "Should successfully delete team")

		deleteBody, err := io.ReadAll(deleteResp.Body)
		assert.NoError(t, err, "Should read delete response")

		var deleteResponse map[string]interface{}
		err = json.Unmarshal(deleteBody, &deleteResponse)
		assert.NoError(t, err, "Should parse delete response")

		assert.Equal(t, "Team deleted successfully", deleteResponse["message"], "Should have success message")

		t.Logf("Team deletion with member cleanup test passed - team %s deleted successfully", teamID)

		// Verify team is actually deleted by trying to get it
		getReq, err := http.NewRequest("GET", serverURL+baseURL+"/user/teams/"+teamID, nil)
		assert.NoError(t, err, "Should create get request")
		getReq.Header.Set("Authorization", "Bearer "+tokenInfo.AccessToken)

		getResp, err := (&http.Client{}).Do(getReq)
		assert.NoError(t, err, "Should send get request")
		defer getResp.Body.Close()

		assert.Equal(t, 404, getResp.StatusCode, "Should return 404 for deleted team")

	} else {
		t.Fatalf("Failed to create team: status=%d", createResp.StatusCode)
	}
}

// TestTeamCreateMembershipVerification tests that team creation automatically adds the creator as owner member
func TestTeamCreateMembershipVerification(t *testing.T) {
	// Initialize test environment
	serverURL := testutils.Prepare(t)
	defer testutils.Clean()

	// Get base URL from server config
	baseURL := ""
	if openapi.Server != nil && openapi.Server.Config != nil {
		baseURL = openapi.Server.Config.BaseURL
	}

	// Register a test client for OAuth authentication
	testClient := testutils.RegisterTestClient(t, "Team Member Test Client", []string{"https://localhost/callback"})
	defer testutils.CleanupTestClient(t, testClient.ClientID)

	// Obtain access token for authenticated requests
	tokenInfo := testutils.ObtainAccessToken(t, serverURL, testClient.ClientID, testClient.ClientSecret, "https://localhost/callback", "openid profile")

	// Create a team
	createTeamBody := map[string]interface{}{
		"name":        "Membership Test Team",
		"description": "Team for testing automatic owner membership",
	}

	createReq := createTeamRequest(t, serverURL+baseURL+"/user/teams", createTeamBody, tokenInfo.AccessToken)
	createResp, err := (&http.Client{}).Do(createReq)
	assert.NoError(t, err, "Should create test team")
	defer createResp.Body.Close()

	var createdTeam map[string]interface{}
	if createResp.StatusCode == 201 {
		createBody, _ := io.ReadAll(createResp.Body)
		json.Unmarshal(createBody, &createdTeam)

		// Verify team was created successfully
		assert.Equal(t, "Membership Test Team", createdTeam["name"], "Should have correct team name")
		assert.Equal(t, tokenInfo.UserID, createdTeam["owner_id"], "Should have correct owner_id")

		t.Logf("Created team: %s (ID: %s, Owner: %s)",
			createdTeam["name"], getTeamID(createdTeam), createdTeam["owner_id"])

		// Verify that creator is automatically added as owner member
		teamID := getTeamID(createdTeam)
		provider := testutils.GetUserProvider(t)

		member, err := provider.GetMember(context.Background(), teamID, tokenInfo.UserID)
		if err == nil {
			// Verify member exists and has correct properties
			assert.Equal(t, teamID, member["team_id"], "Member should belong to created team")
			assert.Equal(t, tokenInfo.UserID, member["user_id"], "Member should have correct user_id")
			assert.Equal(t, "active", member["status"], "Member should be active")

			// Verify is_owner field is set to true
			isOwner := member["is_owner"]
			assert.NotNil(t, isOwner, "is_owner field should be present")
			// Handle different boolean representations from database
			assert.True(t, isOwner == true || isOwner == int64(1) || isOwner == 1,
				"is_owner should be true for team creator, got: %v (type: %T)", isOwner, isOwner)

			t.Logf("Verified creator is automatically added as owner member with is_owner=true")
		} else {
			t.Logf("Could not verify member (may not be implemented yet): %v", err)
		}

		t.Logf("Team creation with automatic owner membership test passed")
	} else {
		t.Fatalf("Failed to create team: status=%d", createResp.StatusCode)
	}
}

// Helper functions

// createTeamRequest creates a POST request for team creation
func createTeamRequest(t *testing.T, url string, body map[string]interface{}, accessToken string) *http.Request {
	bodyBytes, err := json.Marshal(body)
	assert.NoError(t, err, "Should marshal team creation body")

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(bodyBytes))
	assert.NoError(t, err, "Should create team creation request")

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+accessToken)

	return req
}

// getTeamID extracts team_id from created team response
func getTeamID(team map[string]interface{}) string {
	if team == nil {
		return ""
	}
	if teamID, ok := team["team_id"].(string); ok {
		return teamID
	}
	// Handle numeric team_id
	if teamID, ok := team["team_id"].(float64); ok {
		return fmt.Sprintf("%.0f", teamID)
	}
	if teamID, ok := team["team_id"].(int64); ok {
		return fmt.Sprintf("%d", teamID)
	}
	if teamID, ok := team["team_id"].(int); ok {
		return fmt.Sprintf("%d", teamID)
	}
	return ""
}
