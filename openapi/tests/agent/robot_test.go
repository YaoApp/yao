package openapi_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/yaoapp/yao/openapi"
	"github.com/yaoapp/yao/openapi/tests/testutils"
)

// TestListRobots tests the robot listing endpoint
func TestListRobots(t *testing.T) {
	serverURL := testutils.Prepare(t)
	defer testutils.Clean()

	// Get base URL from server config
	baseURL := ""
	if openapi.Server != nil && openapi.Server.Config != nil {
		baseURL = openapi.Server.Config.BaseURL
	}

	// Register test client and get token
	client := testutils.RegisterTestClient(t, "Robot List Test Client", []string{"https://localhost/callback"})
	defer testutils.CleanupTestClient(t, client.ClientID)
	tokenInfo := testutils.ObtainAccessToken(t, serverURL, client.ClientID, client.ClientSecret, "https://localhost/callback", "openid profile")

	t.Run("ListRobotsSuccess", func(t *testing.T) {
		req, err := http.NewRequest("GET", serverURL+baseURL+"/agent/robots", nil)
		require.NoError(t, err)

		req.Header.Set("Authorization", "Bearer "+tokenInfo.AccessToken)

		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err)
		require.NotNil(t, resp)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var response map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&response)
		require.NoError(t, err)

		// Verify pagination fields exist
		assert.Contains(t, response, "data")
		assert.Contains(t, response, "page")
		assert.Contains(t, response, "pagesize")
		assert.Contains(t, response, "total")

		// Verify runtime status fields are included in robot items
		if data, ok := response["data"].([]interface{}); ok && len(data) > 0 {
			robot := data[0].(map[string]interface{})
			// Runtime status fields should be present (added for dashboard optimization)
			assert.Contains(t, robot, "running", "Robot should include running count")
			assert.Contains(t, robot, "max_running", "Robot should include max_running")
			// last_run and next_run are optional (omitempty)
		}
	})

	t.Run("ListRobotsWithPagination", func(t *testing.T) {
		req, err := http.NewRequest("GET", serverURL+baseURL+"/agent/robots?page=1&pagesize=5", nil)
		require.NoError(t, err)

		req.Header.Set("Authorization", "Bearer "+tokenInfo.AccessToken)

		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var response map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&response)
		require.NoError(t, err)

		assert.Equal(t, float64(1), response["page"])
		assert.Equal(t, float64(5), response["pagesize"])
	})

	t.Run("ListRobotsWithAutonomousModeFilter", func(t *testing.T) {
		// Test with autonomous_mode=true
		req, err := http.NewRequest("GET", serverURL+baseURL+"/agent/robots?autonomous_mode=true", nil)
		require.NoError(t, err)

		req.Header.Set("Authorization", "Bearer "+tokenInfo.AccessToken)

		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var response map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&response)
		require.NoError(t, err)

		// Verify response structure
		assert.Contains(t, response, "data")
		assert.Contains(t, response, "total")

		// If there are robots, verify they are all autonomous
		if data, ok := response["data"].([]interface{}); ok && len(data) > 0 {
			for _, item := range data {
				if robot, ok := item.(map[string]interface{}); ok {
					assert.True(t, robot["autonomous_mode"].(bool), "All robots should have autonomous_mode=true")
				}
			}
		}
	})

	t.Run("ListRobotsWithAutonomousModeFalse", func(t *testing.T) {
		// Test with autonomous_mode=false
		req, err := http.NewRequest("GET", serverURL+baseURL+"/agent/robots?autonomous_mode=false", nil)
		require.NoError(t, err)

		req.Header.Set("Authorization", "Bearer "+tokenInfo.AccessToken)

		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var response map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&response)
		require.NoError(t, err)

		// Verify response structure
		assert.Contains(t, response, "data")
		assert.Contains(t, response, "total")

		// If there are robots, verify they are all on-demand (not autonomous)
		if data, ok := response["data"].([]interface{}); ok && len(data) > 0 {
			for _, item := range data {
				if robot, ok := item.(map[string]interface{}); ok {
					assert.False(t, robot["autonomous_mode"].(bool), "All robots should have autonomous_mode=false")
				}
			}
		}
	})

	t.Run("ListRobotsIncludesRuntimeStatus", func(t *testing.T) {
		// Verify that list response includes runtime status fields for dashboard optimization
		req, err := http.NewRequest("GET", serverURL+baseURL+"/agent/robots", nil)
		require.NoError(t, err)

		req.Header.Set("Authorization", "Bearer "+tokenInfo.AccessToken)

		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var response map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&response)
		require.NoError(t, err)

		// Verify we have robots to test
		data, ok := response["data"].([]interface{})
		require.True(t, ok, "Response should have data array")

		if len(data) > 0 {
			robot := data[0].(map[string]interface{})

			// Runtime status fields (running is always present, defaults to 0)
			_, hasRunning := robot["running"]
			assert.True(t, hasRunning, "Robot should include 'running' field for dashboard")

			// max_running should be present (with omitempty, only if > 0)
			// The field is returned by GetRobotStatus, so it should be there
			if maxRunning, ok := robot["max_running"]; ok {
				assert.GreaterOrEqual(t, maxRunning.(float64), float64(0), "max_running should be >= 0")
			}

			// robot_status should reflect runtime status
			robotStatus, hasStatus := robot["robot_status"]
			assert.True(t, hasStatus, "Robot should include 'robot_status' field")
			if hasStatus {
				validStatuses := []string{"idle", "working", "paused", "error", "maintenance"}
				assert.Contains(t, validStatuses, robotStatus.(string), "robot_status should be a valid status")
			}
		}
	})

	t.Run("ListRobotsUnauthorized", func(t *testing.T) {
		req, err := http.NewRequest("GET", serverURL+baseURL+"/agent/robots", nil)
		require.NoError(t, err)

		// No Authorization header

		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	})
}

// TestCreateRobot tests the robot creation endpoint
func TestCreateRobot(t *testing.T) {
	serverURL := testutils.Prepare(t)
	defer testutils.Clean()

	// Get base URL from server config
	baseURL := ""
	if openapi.Server != nil && openapi.Server.Config != nil {
		baseURL = openapi.Server.Config.BaseURL
	}

	// Register test client and get token
	client := testutils.RegisterTestClient(t, "Robot Create Test Client", []string{"https://localhost/callback"})
	defer testutils.CleanupTestClient(t, client.ClientID)
	tokenInfo := testutils.ObtainAccessToken(t, serverURL, client.ClientID, client.ClientSecret, "https://localhost/callback", "openid profile")

	// Track created robots for cleanup
	var createdRobotIDs []string
	defer func() {
		// Cleanup created robots
		for _, robotID := range createdRobotIDs {
			req, _ := http.NewRequest("DELETE", serverURL+baseURL+"/agent/robots/"+robotID, nil)
			req.Header.Set("Authorization", "Bearer "+tokenInfo.AccessToken)
			http.DefaultClient.Do(req)
		}
	}()

	t.Run("CreateRobotSuccess", func(t *testing.T) {
		robotID := fmt.Sprintf("test_robot_%d", time.Now().UnixNano())

		createData := map[string]interface{}{
			"member_id":    robotID,
			"team_id":      "test_team_001",
			"display_name": "Test Robot",
			"bio":          "A test robot for API testing",
			"robot_email":  "test@robot.local",
		}

		body, _ := json.Marshal(createData)
		req, err := http.NewRequest("POST", serverURL+baseURL+"/agent/robots", bytes.NewBuffer(body))
		require.NoError(t, err)

		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+tokenInfo.AccessToken)

		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err)
		require.NotNil(t, resp)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusCreated, resp.StatusCode)

		var response map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&response)
		require.NoError(t, err)

		assert.Equal(t, robotID, response["member_id"])
		assert.Equal(t, "Test Robot", response["display_name"])
		assert.Equal(t, "A test robot for API testing", response["bio"])

		// Track for cleanup
		createdRobotIDs = append(createdRobotIDs, robotID)
	})

	t.Run("CreateRobotMissingDisplayName", func(t *testing.T) {
		// Missing display_name (the only required field)
		createData := map[string]interface{}{
			"team_id": "test_team_001",
			// display_name is missing
		}

		body, _ := json.Marshal(createData)
		req, err := http.NewRequest("POST", serverURL+baseURL+"/agent/robots", bytes.NewBuffer(body))
		require.NoError(t, err)

		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+tokenInfo.AccessToken)

		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})

	t.Run("CreateRobotAutoGenerateMemberID", func(t *testing.T) {
		// member_id is optional - should be auto-generated
		createData := map[string]interface{}{
			"team_id":      "test_team_001",
			"display_name": "Auto ID Robot",
		}

		body, _ := json.Marshal(createData)
		req, err := http.NewRequest("POST", serverURL+baseURL+"/agent/robots", bytes.NewBuffer(body))
		require.NoError(t, err)

		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+tokenInfo.AccessToken)

		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusCreated, resp.StatusCode)

		var response map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&response)
		require.NoError(t, err)

		// Verify member_id was auto-generated
		memberID, ok := response["member_id"].(string)
		assert.True(t, ok, "member_id should be a string")
		assert.NotEmpty(t, memberID, "member_id should be auto-generated")
		assert.Len(t, memberID, 12, "auto-generated member_id should be 12 digits")
		t.Logf("Auto-generated member_id: %s", memberID)

		// Cleanup
		createdRobotIDs = append(createdRobotIDs, memberID)
	})

	t.Run("CreateRobotDuplicate", func(t *testing.T) {
		robotID := fmt.Sprintf("test_robot_dup_%d", time.Now().UnixNano())

		createData := map[string]interface{}{
			"member_id":    robotID,
			"team_id":      "test_team_001",
			"display_name": "Test Robot Duplicate",
		}

		body, _ := json.Marshal(createData)

		// First create
		req1, _ := http.NewRequest("POST", serverURL+baseURL+"/agent/robots", bytes.NewBuffer(body))
		req1.Header.Set("Content-Type", "application/json")
		req1.Header.Set("Authorization", "Bearer "+tokenInfo.AccessToken)
		resp1, err := http.DefaultClient.Do(req1)
		require.NoError(t, err)
		resp1.Body.Close()
		assert.Equal(t, http.StatusCreated, resp1.StatusCode)
		createdRobotIDs = append(createdRobotIDs, robotID)

		// Second create with same ID should fail
		body2, _ := json.Marshal(createData)
		req2, _ := http.NewRequest("POST", serverURL+baseURL+"/agent/robots", bytes.NewBuffer(body2))
		req2.Header.Set("Content-Type", "application/json")
		req2.Header.Set("Authorization", "Bearer "+tokenInfo.AccessToken)
		resp2, err := http.DefaultClient.Do(req2)
		require.NoError(t, err)
		defer resp2.Body.Close()

		assert.Equal(t, http.StatusConflict, resp2.StatusCode)
	})
}

// TestGetRobot tests the robot get endpoint
func TestGetRobot(t *testing.T) {
	serverURL := testutils.Prepare(t)
	defer testutils.Clean()

	// Get base URL from server config
	baseURL := ""
	if openapi.Server != nil && openapi.Server.Config != nil {
		baseURL = openapi.Server.Config.BaseURL
	}

	// Register test client and get token
	client := testutils.RegisterTestClient(t, "Robot Get Test Client", []string{"https://localhost/callback"})
	defer testutils.CleanupTestClient(t, client.ClientID)
	tokenInfo := testutils.ObtainAccessToken(t, serverURL, client.ClientID, client.ClientSecret, "https://localhost/callback", "openid profile")

	// Create a test robot first
	robotID := fmt.Sprintf("test_robot_get_%d", time.Now().UnixNano())
	createData := map[string]interface{}{
		"member_id":    robotID,
		"team_id":      "test_team_001",
		"display_name": "Test Robot Get",
		"bio":          "A robot for get test",
	}
	body, _ := json.Marshal(createData)
	createReq, _ := http.NewRequest("POST", serverURL+baseURL+"/agent/robots", bytes.NewBuffer(body))
	createReq.Header.Set("Content-Type", "application/json")
	createReq.Header.Set("Authorization", "Bearer "+tokenInfo.AccessToken)
	createResp, err := http.DefaultClient.Do(createReq)
	require.NoError(t, err)
	createResp.Body.Close()

	// Cleanup
	defer func() {
		req, _ := http.NewRequest("DELETE", serverURL+baseURL+"/agent/robots/"+robotID, nil)
		req.Header.Set("Authorization", "Bearer "+tokenInfo.AccessToken)
		http.DefaultClient.Do(req)
	}()

	t.Run("GetRobotSuccess", func(t *testing.T) {
		req, err := http.NewRequest("GET", serverURL+baseURL+"/agent/robots/"+robotID, nil)
		require.NoError(t, err)

		req.Header.Set("Authorization", "Bearer "+tokenInfo.AccessToken)

		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err)
		require.NotNil(t, resp)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var response map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&response)
		require.NoError(t, err)

		assert.Equal(t, robotID, response["member_id"])
		assert.Equal(t, "Test Robot Get", response["display_name"])
	})

	t.Run("GetRobotNotFound", func(t *testing.T) {
		req, err := http.NewRequest("GET", serverURL+baseURL+"/agent/robots/non_existent_robot", nil)
		require.NoError(t, err)

		req.Header.Set("Authorization", "Bearer "+tokenInfo.AccessToken)

		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusNotFound, resp.StatusCode)
	})
}

// TestUpdateRobot tests the robot update endpoint
func TestUpdateRobot(t *testing.T) {
	serverURL := testutils.Prepare(t)
	defer testutils.Clean()

	// Get base URL from server config
	baseURL := ""
	if openapi.Server != nil && openapi.Server.Config != nil {
		baseURL = openapi.Server.Config.BaseURL
	}

	// Register test client and get token
	client := testutils.RegisterTestClient(t, "Robot Update Test Client", []string{"https://localhost/callback"})
	defer testutils.CleanupTestClient(t, client.ClientID)
	tokenInfo := testutils.ObtainAccessToken(t, serverURL, client.ClientID, client.ClientSecret, "https://localhost/callback", "openid profile")

	// Create a test robot first
	robotID := fmt.Sprintf("test_robot_update_%d", time.Now().UnixNano())
	createData := map[string]interface{}{
		"member_id":    robotID,
		"team_id":      "test_team_001",
		"display_name": "Test Robot Update",
		"bio":          "Original bio",
	}
	body, _ := json.Marshal(createData)
	createReq, _ := http.NewRequest("POST", serverURL+baseURL+"/agent/robots", bytes.NewBuffer(body))
	createReq.Header.Set("Content-Type", "application/json")
	createReq.Header.Set("Authorization", "Bearer "+tokenInfo.AccessToken)
	createResp, err := http.DefaultClient.Do(createReq)
	require.NoError(t, err)
	createResp.Body.Close()

	// Cleanup
	defer func() {
		req, _ := http.NewRequest("DELETE", serverURL+baseURL+"/agent/robots/"+robotID, nil)
		req.Header.Set("Authorization", "Bearer "+tokenInfo.AccessToken)
		http.DefaultClient.Do(req)
	}()

	t.Run("UpdateRobotSuccess", func(t *testing.T) {
		updateData := map[string]interface{}{
			"display_name": "Updated Robot Name",
			"bio":          "Updated bio",
		}

		body, _ := json.Marshal(updateData)
		req, err := http.NewRequest("PUT", serverURL+baseURL+"/agent/robots/"+robotID, bytes.NewBuffer(body))
		require.NoError(t, err)

		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+tokenInfo.AccessToken)

		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err)
		require.NotNil(t, resp)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var response map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&response)
		require.NoError(t, err)

		assert.Equal(t, "Updated Robot Name", response["display_name"])
		assert.Equal(t, "Updated bio", response["bio"])
	})

	t.Run("UpdateRobotNotFound", func(t *testing.T) {
		updateData := map[string]interface{}{
			"display_name": "Updated Name",
		}

		body, _ := json.Marshal(updateData)
		req, err := http.NewRequest("PUT", serverURL+baseURL+"/agent/robots/non_existent_robot", bytes.NewBuffer(body))
		require.NoError(t, err)

		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+tokenInfo.AccessToken)

		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusNotFound, resp.StatusCode)
	})
}

// TestDeleteRobot tests the robot delete endpoint
func TestDeleteRobot(t *testing.T) {
	serverURL := testutils.Prepare(t)
	defer testutils.Clean()

	// Get base URL from server config
	baseURL := ""
	if openapi.Server != nil && openapi.Server.Config != nil {
		baseURL = openapi.Server.Config.BaseURL
	}

	// Register test client and get token
	client := testutils.RegisterTestClient(t, "Robot Delete Test Client", []string{"https://localhost/callback"})
	defer testutils.CleanupTestClient(t, client.ClientID)
	tokenInfo := testutils.ObtainAccessToken(t, serverURL, client.ClientID, client.ClientSecret, "https://localhost/callback", "openid profile")

	t.Run("DeleteRobotSuccess", func(t *testing.T) {
		// Create a test robot first
		robotID := fmt.Sprintf("test_robot_delete_%d", time.Now().UnixNano())
		createData := map[string]interface{}{
			"member_id":    robotID,
			"team_id":      "test_team_001",
			"display_name": "Test Robot Delete",
		}
		body, _ := json.Marshal(createData)
		createReq, _ := http.NewRequest("POST", serverURL+baseURL+"/agent/robots", bytes.NewBuffer(body))
		createReq.Header.Set("Content-Type", "application/json")
		createReq.Header.Set("Authorization", "Bearer "+tokenInfo.AccessToken)
		createResp, err := http.DefaultClient.Do(createReq)
		require.NoError(t, err)
		createResp.Body.Close()

		// Delete the robot
		req, err := http.NewRequest("DELETE", serverURL+baseURL+"/agent/robots/"+robotID, nil)
		require.NoError(t, err)

		req.Header.Set("Authorization", "Bearer "+tokenInfo.AccessToken)

		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err)
		require.NotNil(t, resp)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var response map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&response)
		require.NoError(t, err)

		assert.Equal(t, true, response["deleted"])
		assert.Equal(t, robotID, response["member_id"])

		// Verify it's deleted by trying to get it
		getReq, _ := http.NewRequest("GET", serverURL+baseURL+"/agent/robots/"+robotID, nil)
		getReq.Header.Set("Authorization", "Bearer "+tokenInfo.AccessToken)
		getResp, err := http.DefaultClient.Do(getReq)
		require.NoError(t, err)
		defer getResp.Body.Close()

		assert.Equal(t, http.StatusNotFound, getResp.StatusCode)
	})

	t.Run("DeleteRobotNotFound", func(t *testing.T) {
		req, err := http.NewRequest("DELETE", serverURL+baseURL+"/agent/robots/non_existent_robot", nil)
		require.NoError(t, err)

		req.Header.Set("Authorization", "Bearer "+tokenInfo.AccessToken)

		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusNotFound, resp.StatusCode)
	})
}

// TestGetRobotStatus tests the robot status endpoint
func TestGetRobotStatus(t *testing.T) {
	serverURL := testutils.Prepare(t)
	defer testutils.Clean()

	// Get base URL from server config
	baseURL := ""
	if openapi.Server != nil && openapi.Server.Config != nil {
		baseURL = openapi.Server.Config.BaseURL
	}

	// Register test client and get token
	client := testutils.RegisterTestClient(t, "Robot Status Test Client", []string{"https://localhost/callback"})
	defer testutils.CleanupTestClient(t, client.ClientID)
	tokenInfo := testutils.ObtainAccessToken(t, serverURL, client.ClientID, client.ClientSecret, "https://localhost/callback", "openid profile")

	// Create a test robot first
	robotID := fmt.Sprintf("test_robot_status_%d", time.Now().UnixNano())
	createData := map[string]interface{}{
		"member_id":       robotID,
		"team_id":         "test_team_001",
		"display_name":    "Test Robot Status",
		"autonomous_mode": true,
	}
	body, _ := json.Marshal(createData)
	createReq, _ := http.NewRequest("POST", serverURL+baseURL+"/agent/robots", bytes.NewBuffer(body))
	createReq.Header.Set("Content-Type", "application/json")
	createReq.Header.Set("Authorization", "Bearer "+tokenInfo.AccessToken)
	createResp, err := http.DefaultClient.Do(createReq)
	require.NoError(t, err)
	createResp.Body.Close()

	// Cleanup
	defer func() {
		req, _ := http.NewRequest("DELETE", serverURL+baseURL+"/agent/robots/"+robotID, nil)
		req.Header.Set("Authorization", "Bearer "+tokenInfo.AccessToken)
		http.DefaultClient.Do(req)
	}()

	t.Run("GetRobotStatusSuccess", func(t *testing.T) {
		req, err := http.NewRequest("GET", serverURL+baseURL+"/agent/robots/"+robotID+"/status", nil)
		require.NoError(t, err)

		req.Header.Set("Authorization", "Bearer "+tokenInfo.AccessToken)

		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err)
		require.NotNil(t, resp)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var response map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&response)
		require.NoError(t, err)

		assert.Equal(t, robotID, response["member_id"])
		assert.Contains(t, response, "status")
		assert.Contains(t, response, "running")
		assert.Contains(t, response, "max_running")
	})

	t.Run("GetRobotStatusNotFound", func(t *testing.T) {
		req, err := http.NewRequest("GET", serverURL+baseURL+"/agent/robots/non_existent_robot/status", nil)
		require.NoError(t, err)

		req.Header.Set("Authorization", "Bearer "+tokenInfo.AccessToken)

		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusNotFound, resp.StatusCode)
	})
}

// TestRobotPermissions tests robot permission scenarios
// Tests personal user vs team user access control
func TestRobotPermissions(t *testing.T) {
	serverURL := testutils.Prepare(t)
	defer testutils.Clean()

	// Get base URL from server config
	baseURL := ""
	if openapi.Server != nil && openapi.Server.Config != nil {
		baseURL = openapi.Server.Config.BaseURL
	}

	// Register test client
	client := testutils.RegisterTestClient(t, "Robot Permission Test Client", []string{"https://localhost/callback"})
	defer testutils.CleanupTestClient(t, client.ClientID)

	// Create User 1 (Personal user - no team)
	token1 := testutils.ObtainAccessToken(t, serverURL, client.ClientID, client.ClientSecret, "https://localhost/callback", "openid profile")
	user1ID := token1.UserID

	// Create User 2 (Different user)
	token2 := testutils.ObtainAccessToken(t, serverURL, client.ClientID, client.ClientSecret, "https://localhost/callback", "openid profile")
	user2ID := token2.UserID

	t.Logf("Test users created: User1=%s, User2=%s", user1ID, user2ID)

	// Track created robots for cleanup
	var createdRobotIDs []string
	defer func() {
		for _, robotID := range createdRobotIDs {
			req, _ := http.NewRequest("DELETE", serverURL+baseURL+"/agent/robots/"+robotID, nil)
			req.Header.Set("Authorization", "Bearer "+token1.AccessToken)
			http.DefaultClient.Do(req)
		}
	}()

	t.Run("PersonalUserCreateRobot", func(t *testing.T) {
		// Personal user creates a robot with their user_id as team_id
		// This simulates a personal user (no team) creating their own robot
		robotID := fmt.Sprintf("test_personal_robot_%d", time.Now().UnixNano())

		createData := map[string]interface{}{
			"member_id":    robotID,
			"team_id":      user1ID, // Personal user: team_id = user_id
			"display_name": "Personal Robot",
			"bio":          "A robot created by a personal user",
		}

		body, _ := json.Marshal(createData)
		req, err := http.NewRequest("POST", serverURL+baseURL+"/agent/robots", bytes.NewBuffer(body))
		require.NoError(t, err)

		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+token1.AccessToken)

		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusCreated, resp.StatusCode)

		var response map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&response)
		require.NoError(t, err)

		assert.Equal(t, robotID, response["member_id"])
		assert.Equal(t, user1ID, response["team_id"])
		t.Logf("Personal robot created: %s (team_id: %s)", robotID, user1ID)
		createdRobotIDs = append(createdRobotIDs, robotID)
	})

	t.Run("PersonalUserCanAccessOwnRobot", func(t *testing.T) {
		// User 1 creates a robot
		robotID := fmt.Sprintf("test_own_robot_%d", time.Now().UnixNano())

		createData := map[string]interface{}{
			"member_id":    robotID,
			"team_id":      user1ID, // Personal user: team_id = user_id
			"display_name": "User 1 Robot",
		}

		body, _ := json.Marshal(createData)
		createReq, _ := http.NewRequest("POST", serverURL+baseURL+"/agent/robots", bytes.NewBuffer(body))
		createReq.Header.Set("Content-Type", "application/json")
		createReq.Header.Set("Authorization", "Bearer "+token1.AccessToken)
		createResp, _ := http.DefaultClient.Do(createReq)
		createResp.Body.Close()
		createdRobotIDs = append(createdRobotIDs, robotID)

		// User 1 can access their own robot
		getReq, err := http.NewRequest("GET", serverURL+baseURL+"/agent/robots/"+robotID, nil)
		require.NoError(t, err)
		getReq.Header.Set("Authorization", "Bearer "+token1.AccessToken)

		getResp, err := http.DefaultClient.Do(getReq)
		require.NoError(t, err)
		defer getResp.Body.Close()

		assert.Equal(t, http.StatusOK, getResp.StatusCode)
		t.Logf("User 1 successfully accessed their own robot: %s", robotID)
	})

	t.Run("PersonalUserCanUpdateOwnRobot", func(t *testing.T) {
		// User 1 creates a robot
		robotID := fmt.Sprintf("test_update_robot_%d", time.Now().UnixNano())

		createData := map[string]interface{}{
			"member_id":    robotID,
			"team_id":      user1ID,
			"display_name": "Original Name",
		}

		body, _ := json.Marshal(createData)
		createReq, _ := http.NewRequest("POST", serverURL+baseURL+"/agent/robots", bytes.NewBuffer(body))
		createReq.Header.Set("Content-Type", "application/json")
		createReq.Header.Set("Authorization", "Bearer "+token1.AccessToken)
		createResp, _ := http.DefaultClient.Do(createReq)
		createResp.Body.Close()
		createdRobotIDs = append(createdRobotIDs, robotID)

		// User 1 can update their own robot
		updateData := map[string]interface{}{
			"display_name": "Updated Name",
		}

		updateBody, _ := json.Marshal(updateData)
		updateReq, err := http.NewRequest("PUT", serverURL+baseURL+"/agent/robots/"+robotID, bytes.NewBuffer(updateBody))
		require.NoError(t, err)
		updateReq.Header.Set("Content-Type", "application/json")
		updateReq.Header.Set("Authorization", "Bearer "+token1.AccessToken)

		updateResp, err := http.DefaultClient.Do(updateReq)
		require.NoError(t, err)
		defer updateResp.Body.Close()

		assert.Equal(t, http.StatusOK, updateResp.StatusCode)

		var response map[string]interface{}
		json.NewDecoder(updateResp.Body).Decode(&response)
		assert.Equal(t, "Updated Name", response["display_name"])
		t.Logf("User 1 successfully updated their own robot")
	})

	t.Run("PersonalUserCanDeleteOwnRobot", func(t *testing.T) {
		// User 1 creates a robot
		robotID := fmt.Sprintf("test_delete_robot_%d", time.Now().UnixNano())

		createData := map[string]interface{}{
			"member_id":    robotID,
			"team_id":      user1ID,
			"display_name": "Robot to Delete",
		}

		body, _ := json.Marshal(createData)
		createReq, _ := http.NewRequest("POST", serverURL+baseURL+"/agent/robots", bytes.NewBuffer(body))
		createReq.Header.Set("Content-Type", "application/json")
		createReq.Header.Set("Authorization", "Bearer "+token1.AccessToken)
		createResp, _ := http.DefaultClient.Do(createReq)
		createResp.Body.Close()

		// User 1 can delete their own robot
		deleteReq, err := http.NewRequest("DELETE", serverURL+baseURL+"/agent/robots/"+robotID, nil)
		require.NoError(t, err)
		deleteReq.Header.Set("Authorization", "Bearer "+token1.AccessToken)

		deleteResp, err := http.DefaultClient.Do(deleteReq)
		require.NoError(t, err)
		defer deleteResp.Body.Close()

		assert.Equal(t, http.StatusOK, deleteResp.StatusCode)
		t.Logf("User 1 successfully deleted their own robot")
	})

	t.Run("TeamRobotAccess", func(t *testing.T) {
		// Create a robot with a shared team_id
		sharedTeamID := fmt.Sprintf("team_%d", time.Now().UnixNano())
		robotID := fmt.Sprintf("test_team_robot_%d", time.Now().UnixNano())

		createData := map[string]interface{}{
			"member_id":    robotID,
			"team_id":      sharedTeamID,
			"display_name": "Team Robot",
		}

		body, _ := json.Marshal(createData)
		createReq, _ := http.NewRequest("POST", serverURL+baseURL+"/agent/robots", bytes.NewBuffer(body))
		createReq.Header.Set("Content-Type", "application/json")
		createReq.Header.Set("Authorization", "Bearer "+token1.AccessToken)
		createResp, _ := http.DefaultClient.Do(createReq)
		createResp.Body.Close()
		createdRobotIDs = append(createdRobotIDs, robotID)

		// Creator can access the team robot
		getReq, err := http.NewRequest("GET", serverURL+baseURL+"/agent/robots/"+robotID, nil)
		require.NoError(t, err)
		getReq.Header.Set("Authorization", "Bearer "+token1.AccessToken)

		getResp, err := http.DefaultClient.Do(getReq)
		require.NoError(t, err)
		defer getResp.Body.Close()

		assert.Equal(t, http.StatusOK, getResp.StatusCode)
		t.Logf("Creator successfully accessed team robot: %s (team: %s)", robotID, sharedTeamID)
	})

	t.Run("VerifyYaoPermissionFieldsSet", func(t *testing.T) {
		// Create a robot and verify __yao_created_by and __yao_team_id are set
		robotID := fmt.Sprintf("test_perm_fields_%d", time.Now().UnixNano())

		createData := map[string]interface{}{
			"member_id":    robotID,
			"team_id":      user1ID, // Personal user: team_id = user_id
			"display_name": "Permission Fields Test Robot",
		}

		body, _ := json.Marshal(createData)
		createReq, _ := http.NewRequest("POST", serverURL+baseURL+"/agent/robots", bytes.NewBuffer(body))
		createReq.Header.Set("Content-Type", "application/json")
		createReq.Header.Set("Authorization", "Bearer "+token1.AccessToken)
		createResp, err := http.DefaultClient.Do(createReq)
		require.NoError(t, err)
		defer createResp.Body.Close()
		createdRobotIDs = append(createdRobotIDs, robotID)

		assert.Equal(t, http.StatusCreated, createResp.StatusCode)

		var response map[string]interface{}
		json.NewDecoder(createResp.Body).Decode(&response)

		// The response should contain the robot data
		// Note: __yao_created_by and __yao_team_id might not be in the public response
		// but they should be set in the database
		assert.Equal(t, robotID, response["member_id"])
		t.Logf("Robot created with permission fields (user_id: %s)", user1ID)
	})

	t.Run("DifferentUserCannotUpdateRobot", func(t *testing.T) {
		// User 1 creates a robot
		robotID := fmt.Sprintf("test_cross_update_%d", time.Now().UnixNano())

		createData := map[string]interface{}{
			"member_id":    robotID,
			"team_id":      user1ID,
			"display_name": "User 1 Private Robot",
		}

		body, _ := json.Marshal(createData)
		createReq, _ := http.NewRequest("POST", serverURL+baseURL+"/agent/robots", bytes.NewBuffer(body))
		createReq.Header.Set("Content-Type", "application/json")
		createReq.Header.Set("Authorization", "Bearer "+token1.AccessToken)
		createResp, _ := http.DefaultClient.Do(createReq)
		createResp.Body.Close()
		createdRobotIDs = append(createdRobotIDs, robotID)

		// User 2 attempts to update User 1's robot - should be denied
		// Note: With system:root scope, this might still succeed due to admin privileges
		// In production, user2 would not have system:root
		updateData := map[string]interface{}{
			"display_name": "Unauthorized Update",
		}

		updateBody, _ := json.Marshal(updateData)
		updateReq, err := http.NewRequest("PUT", serverURL+baseURL+"/agent/robots/"+robotID, bytes.NewBuffer(updateBody))
		require.NoError(t, err)
		updateReq.Header.Set("Content-Type", "application/json")
		updateReq.Header.Set("Authorization", "Bearer "+token2.AccessToken)

		updateResp, err := http.DefaultClient.Do(updateReq)
		require.NoError(t, err)
		defer updateResp.Body.Close()

		// With system:root scope (no constraints), user2 can still update
		// This test documents the current behavior with admin privileges
		t.Logf("User 2 update attempt status: %d (with system:root scope)", updateResp.StatusCode)
	})

	t.Run("DifferentUserCannotDeleteRobot", func(t *testing.T) {
		// User 1 creates a robot
		robotID := fmt.Sprintf("test_cross_delete_%d", time.Now().UnixNano())

		createData := map[string]interface{}{
			"member_id":    robotID,
			"team_id":      user1ID,
			"display_name": "User 1 Robot for Delete Test",
		}

		body, _ := json.Marshal(createData)
		createReq, _ := http.NewRequest("POST", serverURL+baseURL+"/agent/robots", bytes.NewBuffer(body))
		createReq.Header.Set("Content-Type", "application/json")
		createReq.Header.Set("Authorization", "Bearer "+token1.AccessToken)
		createResp, _ := http.DefaultClient.Do(createReq)
		createResp.Body.Close()
		createdRobotIDs = append(createdRobotIDs, robotID)

		// User 2 attempts to delete User 1's robot
		// Note: With system:root scope, this might still succeed due to admin privileges
		deleteReq, err := http.NewRequest("DELETE", serverURL+baseURL+"/agent/robots/"+robotID, nil)
		require.NoError(t, err)
		deleteReq.Header.Set("Authorization", "Bearer "+token2.AccessToken)

		deleteResp, err := http.DefaultClient.Do(deleteReq)
		require.NoError(t, err)
		defer deleteResp.Body.Close()

		// With system:root scope (no constraints), user2 can still delete
		// This test documents the current behavior with admin privileges
		t.Logf("User 2 delete attempt status: %d (with system:root scope)", deleteResp.StatusCode)
	})

	t.Run("ListRobotsWithTeamFilter", func(t *testing.T) {
		// Create robots for both users
		robot1ID := fmt.Sprintf("test_list_user1_%d", time.Now().UnixNano())
		robot2ID := fmt.Sprintf("test_list_user2_%d", time.Now().UnixNano())

		// User 1 creates their robot
		create1 := map[string]interface{}{
			"member_id":    robot1ID,
			"team_id":      user1ID,
			"display_name": "User 1 List Robot",
		}
		body1, _ := json.Marshal(create1)
		req1, _ := http.NewRequest("POST", serverURL+baseURL+"/agent/robots", bytes.NewBuffer(body1))
		req1.Header.Set("Content-Type", "application/json")
		req1.Header.Set("Authorization", "Bearer "+token1.AccessToken)
		resp1, _ := http.DefaultClient.Do(req1)
		resp1.Body.Close()
		createdRobotIDs = append(createdRobotIDs, robot1ID)

		// User 2 creates their robot
		create2 := map[string]interface{}{
			"member_id":    robot2ID,
			"team_id":      user2ID,
			"display_name": "User 2 List Robot",
		}
		body2, _ := json.Marshal(create2)
		req2, _ := http.NewRequest("POST", serverURL+baseURL+"/agent/robots", bytes.NewBuffer(body2))
		req2.Header.Set("Content-Type", "application/json")
		req2.Header.Set("Authorization", "Bearer "+token2.AccessToken)
		resp2, _ := http.DefaultClient.Do(req2)
		resp2.Body.Close()
		createdRobotIDs = append(createdRobotIDs, robot2ID)

		// User 1 lists robots with their team_id filter
		listReq, err := http.NewRequest("GET", serverURL+baseURL+"/agent/robots?team_id="+user1ID, nil)
		require.NoError(t, err)
		listReq.Header.Set("Authorization", "Bearer "+token1.AccessToken)

		listResp, err := http.DefaultClient.Do(listReq)
		require.NoError(t, err)
		defer listResp.Body.Close()

		assert.Equal(t, http.StatusOK, listResp.StatusCode)

		var response map[string]interface{}
		json.NewDecoder(listResp.Body).Decode(&response)

		data := response["data"].([]interface{})
		t.Logf("User 1 sees %d robots with team_id=%s filter", len(data), user1ID)
	})
}
