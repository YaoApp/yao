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

// TestListExecutions tests the execution listing endpoint
// GET /v1/agent/robots/:id/executions
func TestListExecutions(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping execution tests in short mode")
	}

	serverURL := testutils.Prepare(t)
	defer testutils.Clean()

	baseURL := ""
	if openapi.Server != nil && openapi.Server.Config != nil {
		baseURL = openapi.Server.Config.BaseURL
	}

	// Register test client and get token
	client := testutils.RegisterTestClient(t, "Execution List Test Client", []string{"https://localhost/callback"})
	defer testutils.CleanupTestClient(t, client.ClientID)
	tokenInfo := testutils.ObtainAccessToken(t, serverURL, client.ClientID, client.ClientSecret, "https://localhost/callback", "openid profile")

	// Create a test robot
	robotID := fmt.Sprintf("test_exec_list_%d", time.Now().UnixNano())
	createRobot(t, serverURL, baseURL, tokenInfo.AccessToken, robotID, "Execution List Robot")
	defer deleteRobot(t, serverURL, baseURL, tokenInfo.AccessToken, robotID)

	t.Run("ListExecutionsSuccess", func(t *testing.T) {
		req, err := http.NewRequest("GET", serverURL+baseURL+"/agent/robots/"+robotID+"/executions", nil)
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

		// Verify execution items structure (if any executions exist)
		data, ok := response["data"].([]interface{})
		if ok && len(data) > 0 {
			exec := data[0].(map[string]interface{})
			// Basic fields should exist
			assert.Contains(t, exec, "id")
			assert.Contains(t, exec, "status")
			assert.Contains(t, exec, "phase")
			// UI display fields (may be empty string, but field should be present in response)
			// These are new fields added for frontend display
			t.Logf("Execution response fields: id=%v, name=%v, current_task_name=%v",
				exec["id"], exec["name"], exec["current_task_name"])
		}
	})

	t.Run("ListExecutionsWithPagination", func(t *testing.T) {
		req, err := http.NewRequest("GET", serverURL+baseURL+"/agent/robots/"+robotID+"/executions?page=1&pagesize=5", nil)
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

	t.Run("ListExecutionsWithStatusFilter", func(t *testing.T) {
		req, err := http.NewRequest("GET", serverURL+baseURL+"/agent/robots/"+robotID+"/executions?status=completed", nil)
		require.NoError(t, err)

		req.Header.Set("Authorization", "Bearer "+tokenInfo.AccessToken)

		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var response map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&response)
		require.NoError(t, err)

		assert.Contains(t, response, "data")
	})

	t.Run("ListExecutionsWithTriggerTypeFilter", func(t *testing.T) {
		req, err := http.NewRequest("GET", serverURL+baseURL+"/agent/robots/"+robotID+"/executions?trigger_type=human", nil)
		require.NoError(t, err)

		req.Header.Set("Authorization", "Bearer "+tokenInfo.AccessToken)

		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var response map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&response)
		require.NoError(t, err)

		assert.Contains(t, response, "data")
	})

	t.Run("ListExecutionsRobotNotFound", func(t *testing.T) {
		req, err := http.NewRequest("GET", serverURL+baseURL+"/agent/robots/non_existent_robot/executions", nil)
		require.NoError(t, err)

		req.Header.Set("Authorization", "Bearer "+tokenInfo.AccessToken)

		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusNotFound, resp.StatusCode)
	})

	t.Run("ListExecutionsUnauthorized", func(t *testing.T) {
		req, err := http.NewRequest("GET", serverURL+baseURL+"/agent/robots/"+robotID+"/executions", nil)
		require.NoError(t, err)
		// No Authorization header

		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	})
}

// TestGetExecution tests the execution detail endpoint
// GET /v1/agent/robots/:id/executions/:exec_id
func TestGetExecution(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping execution tests in short mode")
	}

	serverURL := testutils.Prepare(t)
	defer testutils.Clean()

	baseURL := ""
	if openapi.Server != nil && openapi.Server.Config != nil {
		baseURL = openapi.Server.Config.BaseURL
	}

	// Register test client and get token
	client := testutils.RegisterTestClient(t, "Execution Get Test Client", []string{"https://localhost/callback"})
	defer testutils.CleanupTestClient(t, client.ClientID)
	tokenInfo := testutils.ObtainAccessToken(t, serverURL, client.ClientID, client.ClientSecret, "https://localhost/callback", "openid profile")

	// Create a test robot
	robotID := fmt.Sprintf("test_exec_get_%d", time.Now().UnixNano())
	createRobot(t, serverURL, baseURL, tokenInfo.AccessToken, robotID, "Execution Get Robot")
	defer deleteRobot(t, serverURL, baseURL, tokenInfo.AccessToken, robotID)

	t.Run("GetExecutionNotFound", func(t *testing.T) {
		req, err := http.NewRequest("GET", serverURL+baseURL+"/agent/robots/"+robotID+"/executions/non_existent_exec", nil)
		require.NoError(t, err)

		req.Header.Set("Authorization", "Bearer "+tokenInfo.AccessToken)

		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusNotFound, resp.StatusCode)
	})

	t.Run("GetExecutionRobotNotFound", func(t *testing.T) {
		req, err := http.NewRequest("GET", serverURL+baseURL+"/agent/robots/non_existent_robot/executions/some_exec", nil)
		require.NoError(t, err)

		req.Header.Set("Authorization", "Bearer "+tokenInfo.AccessToken)

		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusNotFound, resp.StatusCode)
	})
}

// TestExecutionControl tests the execution control endpoints
// POST /v1/agent/robots/:id/executions/:exec_id/pause
// POST /v1/agent/robots/:id/executions/:exec_id/resume
// POST /v1/agent/robots/:id/executions/:exec_id/cancel
func TestExecutionControl(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping execution control tests in short mode")
	}

	serverURL := testutils.Prepare(t)
	defer testutils.Clean()

	baseURL := ""
	if openapi.Server != nil && openapi.Server.Config != nil {
		baseURL = openapi.Server.Config.BaseURL
	}

	// Register test client and get token
	client := testutils.RegisterTestClient(t, "Execution Control Test Client", []string{"https://localhost/callback"})
	defer testutils.CleanupTestClient(t, client.ClientID)
	tokenInfo := testutils.ObtainAccessToken(t, serverURL, client.ClientID, client.ClientSecret, "https://localhost/callback", "openid profile")

	// Create a test robot
	robotID := fmt.Sprintf("test_exec_ctrl_%d", time.Now().UnixNano())
	createRobot(t, serverURL, baseURL, tokenInfo.AccessToken, robotID, "Execution Control Robot")
	defer deleteRobot(t, serverURL, baseURL, tokenInfo.AccessToken, robotID)

	t.Run("PauseExecutionNotFound", func(t *testing.T) {
		req, err := http.NewRequest("POST", serverURL+baseURL+"/agent/robots/"+robotID+"/executions/non_existent_exec/pause", nil)
		require.NoError(t, err)

		req.Header.Set("Authorization", "Bearer "+tokenInfo.AccessToken)

		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		// Should return error (404 or 500 depending on implementation)
		assert.True(t, resp.StatusCode >= 400, "Expected error status code")
	})

	t.Run("ResumeExecutionNotFound", func(t *testing.T) {
		req, err := http.NewRequest("POST", serverURL+baseURL+"/agent/robots/"+robotID+"/executions/non_existent_exec/resume", nil)
		require.NoError(t, err)

		req.Header.Set("Authorization", "Bearer "+tokenInfo.AccessToken)

		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.True(t, resp.StatusCode >= 400, "Expected error status code")
	})

	t.Run("CancelExecutionNotFound", func(t *testing.T) {
		req, err := http.NewRequest("POST", serverURL+baseURL+"/agent/robots/"+robotID+"/executions/non_existent_exec/cancel", nil)
		require.NoError(t, err)

		req.Header.Set("Authorization", "Bearer "+tokenInfo.AccessToken)

		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.True(t, resp.StatusCode >= 400, "Expected error status code")
	})

	t.Run("PauseExecutionRobotNotFound", func(t *testing.T) {
		req, err := http.NewRequest("POST", serverURL+baseURL+"/agent/robots/non_existent_robot/executions/some_exec/pause", nil)
		require.NoError(t, err)

		req.Header.Set("Authorization", "Bearer "+tokenInfo.AccessToken)

		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusNotFound, resp.StatusCode)
	})

	t.Run("ControlExecutionUnauthorized", func(t *testing.T) {
		req, err := http.NewRequest("POST", serverURL+baseURL+"/agent/robots/"+robotID+"/executions/some_exec/pause", nil)
		require.NoError(t, err)
		// No Authorization header

		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	})
}

// TestExecutionPermissions tests execution permission inheritance from robot
func TestExecutionPermissions(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping execution permission tests in short mode")
	}

	serverURL := testutils.Prepare(t)
	defer testutils.Clean()

	baseURL := ""
	if openapi.Server != nil && openapi.Server.Config != nil {
		baseURL = openapi.Server.Config.BaseURL
	}

	// Register test client
	client := testutils.RegisterTestClient(t, "Execution Permission Test Client", []string{"https://localhost/callback"})
	defer testutils.CleanupTestClient(t, client.ClientID)

	// Create User 1
	token1 := testutils.ObtainAccessToken(t, serverURL, client.ClientID, client.ClientSecret, "https://localhost/callback", "openid profile")
	user1ID := token1.UserID

	// Create User 2
	token2 := testutils.ObtainAccessToken(t, serverURL, client.ClientID, client.ClientSecret, "https://localhost/callback", "openid profile")

	// User 1 creates a robot
	robotID := fmt.Sprintf("test_exec_perm_%d", time.Now().UnixNano())
	createRobotWithTeam(t, serverURL, baseURL, token1.AccessToken, robotID, "Permission Test Robot", user1ID)
	defer deleteRobot(t, serverURL, baseURL, token1.AccessToken, robotID)

	t.Run("OwnerCanListExecutions", func(t *testing.T) {
		req, err := http.NewRequest("GET", serverURL+baseURL+"/agent/robots/"+robotID+"/executions", nil)
		require.NoError(t, err)

		req.Header.Set("Authorization", "Bearer "+token1.AccessToken)

		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)
		t.Logf("Owner (User 1) successfully listed executions for their robot")
	})

	t.Run("OtherUserExecutionAccess", func(t *testing.T) {
		// User 2 attempts to list executions for User 1's robot
		// With system:root scope this might succeed
		req, err := http.NewRequest("GET", serverURL+baseURL+"/agent/robots/"+robotID+"/executions", nil)
		require.NoError(t, err)

		req.Header.Set("Authorization", "Bearer "+token2.AccessToken)

		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		t.Logf("User 2 execution list attempt status: %d (with system:root scope)", resp.StatusCode)
	})
}

// ==================== Helper Functions ====================

func createRobot(t *testing.T, serverURL, baseURL, token, robotID, displayName string) {
	createData := map[string]interface{}{
		"member_id":    robotID,
		"team_id":      "test_team_001",
		"display_name": displayName,
	}

	body, _ := json.Marshal(createData)
	req, err := http.NewRequest("POST", serverURL+baseURL+"/agent/robots", bytes.NewBuffer(body))
	require.NoError(t, err)

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	resp.Body.Close()
}

func createRobotWithTeam(t *testing.T, serverURL, baseURL, token, robotID, displayName, teamID string) {
	createData := map[string]interface{}{
		"member_id":    robotID,
		"team_id":      teamID,
		"display_name": displayName,
	}

	body, _ := json.Marshal(createData)
	req, err := http.NewRequest("POST", serverURL+baseURL+"/agent/robots", bytes.NewBuffer(body))
	require.NoError(t, err)

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	resp.Body.Close()
}

func deleteRobot(t *testing.T, serverURL, baseURL, token, robotID string) {
	req, _ := http.NewRequest("DELETE", serverURL+baseURL+"/agent/robots/"+robotID, nil)
	req.Header.Set("Authorization", "Bearer "+token)
	resp, _ := http.DefaultClient.Do(req)
	if resp != nil {
		resp.Body.Close()
	}
}
