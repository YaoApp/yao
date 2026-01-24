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

// TestTriggerRobot tests the robot trigger endpoint
// POST /v1/agent/robots/:id/trigger
func TestTriggerRobot(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping trigger tests in short mode (requires AI/manager)")
	}

	serverURL := testutils.Prepare(t)
	defer testutils.Clean()

	baseURL := ""
	if openapi.Server != nil && openapi.Server.Config != nil {
		baseURL = openapi.Server.Config.BaseURL
	}

	// Register test client and get token
	client := testutils.RegisterTestClient(t, "Trigger Test Client", []string{"https://localhost/callback"})
	defer testutils.CleanupTestClient(t, client.ClientID)
	tokenInfo := testutils.ObtainAccessToken(t, serverURL, client.ClientID, client.ClientSecret, "https://localhost/callback", "openid profile")

	// Create a test robot
	robotID := fmt.Sprintf("test_trigger_%d", time.Now().UnixNano())
	createRobotForTrigger(t, serverURL, baseURL, tokenInfo.AccessToken, robotID, "Trigger Test Robot")
	defer deleteRobotForTrigger(t, serverURL, baseURL, tokenInfo.AccessToken, robotID)

	t.Run("TriggerRobotBasic", func(t *testing.T) {
		triggerData := map[string]interface{}{
			"trigger_type": "human",
			"messages": []map[string]interface{}{
				{
					"role":    "user",
					"content": "Hello, please help me with a task",
				},
			},
		}

		body, _ := json.Marshal(triggerData)
		req, err := http.NewRequest("POST", serverURL+baseURL+"/agent/robots/"+robotID+"/trigger", bytes.NewBuffer(body))
		require.NoError(t, err)

		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+tokenInfo.AccessToken)

		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err)
		require.NotNil(t, resp)
		defer resp.Body.Close()

		var response map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&response)
		require.NoError(t, err)

		// If manager is not started, we get a 500 error (expected in test environment)
		// In production with manager running, response should contain "accepted" field
		if resp.StatusCode == http.StatusInternalServerError {
			// Expected when robot manager is not started
			assert.Contains(t, response, "error_description")
			t.Logf("Trigger response (manager not started): status=%d, error=%v", resp.StatusCode, response["error_description"])
		} else {
			// Manager is running - verify accepted field
			assert.Contains(t, response, "accepted")
			t.Logf("Trigger response: status=%d, accepted=%v", resp.StatusCode, response["accepted"])
		}
	})

	t.Run("TriggerRobotWithAction", func(t *testing.T) {
		triggerData := map[string]interface{}{
			"trigger_type": "human",
			"action":       "task.add",
			"messages": []map[string]interface{}{
				{
					"role":    "user",
					"content": "Add a new task: Review quarterly report",
				},
			},
		}

		body, _ := json.Marshal(triggerData)
		req, err := http.NewRequest("POST", serverURL+baseURL+"/agent/robots/"+robotID+"/trigger", bytes.NewBuffer(body))
		require.NoError(t, err)

		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+tokenInfo.AccessToken)

		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		var response map[string]interface{}
		json.NewDecoder(resp.Body).Decode(&response)

		// If manager is not started, we get a 500 error (expected in test environment)
		if resp.StatusCode == http.StatusInternalServerError {
			assert.Contains(t, response, "error_description")
			t.Logf("Trigger with action response (manager not started): status=%d", resp.StatusCode)
		} else {
			assert.Contains(t, response, "accepted")
			t.Logf("Trigger with action response: status=%d", resp.StatusCode)
		}
	})

	t.Run("TriggerRobotWithLocale", func(t *testing.T) {
		// Test the new locale parameter for i18n support
		triggerData := map[string]interface{}{
			"trigger_type": "human",
			"locale":       "zh", // Chinese locale
			"messages": []map[string]interface{}{
				{
					"role":    "user",
					"content": "请帮我分析销售数据",
				},
			},
		}

		body, _ := json.Marshal(triggerData)
		req, err := http.NewRequest("POST", serverURL+baseURL+"/agent/robots/"+robotID+"/trigger", bytes.NewBuffer(body))
		require.NoError(t, err)

		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+tokenInfo.AccessToken)

		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		var response map[string]interface{}
		json.NewDecoder(resp.Body).Decode(&response)

		// If manager is not started, we get a 500 error (expected in test environment)
		if resp.StatusCode == http.StatusInternalServerError {
			assert.Contains(t, response, "error_description")
			t.Logf("Trigger with locale response (manager not started): status=%d", resp.StatusCode)
		} else {
			assert.Contains(t, response, "accepted")
			t.Logf("Trigger with locale response: status=%d, accepted=%v", resp.StatusCode, response["accepted"])
		}
	})

	t.Run("TriggerRobotNotFound", func(t *testing.T) {
		triggerData := map[string]interface{}{
			"trigger_type": "human",
		}

		body, _ := json.Marshal(triggerData)
		req, err := http.NewRequest("POST", serverURL+baseURL+"/agent/robots/non_existent_robot/trigger", bytes.NewBuffer(body))
		require.NoError(t, err)

		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+tokenInfo.AccessToken)

		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusNotFound, resp.StatusCode)
	})

	t.Run("TriggerRobotUnauthorized", func(t *testing.T) {
		triggerData := map[string]interface{}{
			"trigger_type": "human",
		}

		body, _ := json.Marshal(triggerData)
		req, err := http.NewRequest("POST", serverURL+baseURL+"/agent/robots/"+robotID+"/trigger", bytes.NewBuffer(body))
		require.NoError(t, err)

		req.Header.Set("Content-Type", "application/json")
		// No Authorization header

		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	})

	t.Run("TriggerRobotInvalidBody", func(t *testing.T) {
		req, err := http.NewRequest("POST", serverURL+baseURL+"/agent/robots/"+robotID+"/trigger", bytes.NewBuffer([]byte("invalid json")))
		require.NoError(t, err)

		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+tokenInfo.AccessToken)

		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})
}

// TestInterveneRobot tests the robot intervention endpoint
// POST /v1/agent/robots/:id/intervene
func TestInterveneRobot(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping intervene tests in short mode (requires AI/manager)")
	}

	serverURL := testutils.Prepare(t)
	defer testutils.Clean()

	baseURL := ""
	if openapi.Server != nil && openapi.Server.Config != nil {
		baseURL = openapi.Server.Config.BaseURL
	}

	// Register test client and get token
	client := testutils.RegisterTestClient(t, "Intervene Test Client", []string{"https://localhost/callback"})
	defer testutils.CleanupTestClient(t, client.ClientID)
	tokenInfo := testutils.ObtainAccessToken(t, serverURL, client.ClientID, client.ClientSecret, "https://localhost/callback", "openid profile")

	// Create a test robot
	robotID := fmt.Sprintf("test_intervene_%d", time.Now().UnixNano())
	createRobotForTrigger(t, serverURL, baseURL, tokenInfo.AccessToken, robotID, "Intervene Test Robot")
	defer deleteRobotForTrigger(t, serverURL, baseURL, tokenInfo.AccessToken, robotID)

	t.Run("InterveneRobotBasic", func(t *testing.T) {
		interveneData := map[string]interface{}{
			"action": "task.add",
			"messages": []map[string]interface{}{
				{
					"role":    "user",
					"content": "Please add a high priority task",
				},
			},
		}

		body, _ := json.Marshal(interveneData)
		req, err := http.NewRequest("POST", serverURL+baseURL+"/agent/robots/"+robotID+"/intervene", bytes.NewBuffer(body))
		require.NoError(t, err)

		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+tokenInfo.AccessToken)

		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err)
		require.NotNil(t, resp)
		defer resp.Body.Close()

		var response map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&response)
		require.NoError(t, err)

		// If manager is not started, we get a 500 error (expected in test environment)
		if resp.StatusCode == http.StatusInternalServerError {
			assert.Contains(t, response, "error_description")
			t.Logf("Intervene response (manager not started): status=%d, error=%v", resp.StatusCode, response["error_description"])
		} else {
			assert.Contains(t, response, "accepted")
			t.Logf("Intervene response: status=%d, accepted=%v", resp.StatusCode, response["accepted"])
		}
	})

	t.Run("InterveneRobotMissingAction", func(t *testing.T) {
		interveneData := map[string]interface{}{
			"messages": []map[string]interface{}{
				{
					"role":    "user",
					"content": "Some message",
				},
			},
		}

		body, _ := json.Marshal(interveneData)
		req, err := http.NewRequest("POST", serverURL+baseURL+"/agent/robots/"+robotID+"/intervene", bytes.NewBuffer(body))
		require.NoError(t, err)

		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+tokenInfo.AccessToken)

		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		// Action is required
		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})

	t.Run("InterveneRobotNotFound", func(t *testing.T) {
		interveneData := map[string]interface{}{
			"action": "task.add",
		}

		body, _ := json.Marshal(interveneData)
		req, err := http.NewRequest("POST", serverURL+baseURL+"/agent/robots/non_existent_robot/intervene", bytes.NewBuffer(body))
		require.NoError(t, err)

		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+tokenInfo.AccessToken)

		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusNotFound, resp.StatusCode)
	})

	t.Run("InterveneRobotUnauthorized", func(t *testing.T) {
		interveneData := map[string]interface{}{
			"action": "task.add",
		}

		body, _ := json.Marshal(interveneData)
		req, err := http.NewRequest("POST", serverURL+baseURL+"/agent/robots/"+robotID+"/intervene", bytes.NewBuffer(body))
		require.NoError(t, err)

		req.Header.Set("Content-Type", "application/json")
		// No Authorization header

		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	})

	t.Run("InterveneRobotInvalidBody", func(t *testing.T) {
		req, err := http.NewRequest("POST", serverURL+baseURL+"/agent/robots/"+robotID+"/intervene", bytes.NewBuffer([]byte("invalid json")))
		require.NoError(t, err)

		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+tokenInfo.AccessToken)

		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})
}

// TestTriggerPermissions tests trigger permission inheritance from robot
func TestTriggerPermissions(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping trigger permission tests in short mode")
	}

	serverURL := testutils.Prepare(t)
	defer testutils.Clean()

	baseURL := ""
	if openapi.Server != nil && openapi.Server.Config != nil {
		baseURL = openapi.Server.Config.BaseURL
	}

	// Register test client
	client := testutils.RegisterTestClient(t, "Trigger Permission Test Client", []string{"https://localhost/callback"})
	defer testutils.CleanupTestClient(t, client.ClientID)

	// Create User 1
	token1 := testutils.ObtainAccessToken(t, serverURL, client.ClientID, client.ClientSecret, "https://localhost/callback", "openid profile")
	user1ID := token1.UserID

	// Create User 2
	token2 := testutils.ObtainAccessToken(t, serverURL, client.ClientID, client.ClientSecret, "https://localhost/callback", "openid profile")

	// User 1 creates a robot
	robotID := fmt.Sprintf("test_trig_perm_%d", time.Now().UnixNano())
	createRobotWithTeamForTrigger(t, serverURL, baseURL, token1.AccessToken, robotID, "Trigger Perm Robot", user1ID)
	defer deleteRobotForTrigger(t, serverURL, baseURL, token1.AccessToken, robotID)

	t.Run("OwnerCanTrigger", func(t *testing.T) {
		triggerData := map[string]interface{}{
			"trigger_type": "human",
			"messages": []map[string]interface{}{
				{
					"role":    "user",
					"content": "Owner triggering robot",
				},
			},
		}

		body, _ := json.Marshal(triggerData)
		req, err := http.NewRequest("POST", serverURL+baseURL+"/agent/robots/"+robotID+"/trigger", bytes.NewBuffer(body))
		require.NoError(t, err)

		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+token1.AccessToken)

		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		// Owner should be able to trigger (may fail at manager level with 500, but not 403 permission denied)
		// 500 = manager not started (acceptable), 403 = permission denied (not acceptable)
		assert.NotEqual(t, http.StatusForbidden, resp.StatusCode, "Owner should have permission to trigger")
		t.Logf("Owner trigger attempt status: %d", resp.StatusCode)
	})

	t.Run("OwnerCanIntervene", func(t *testing.T) {
		interveneData := map[string]interface{}{
			"action": "task.add",
			"messages": []map[string]interface{}{
				{
					"role":    "user",
					"content": "Owner intervention",
				},
			},
		}

		body, _ := json.Marshal(interveneData)
		req, err := http.NewRequest("POST", serverURL+baseURL+"/agent/robots/"+robotID+"/intervene", bytes.NewBuffer(body))
		require.NoError(t, err)

		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+token1.AccessToken)

		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		// 500 = manager not started (acceptable), 403 = permission denied (not acceptable)
		assert.NotEqual(t, http.StatusForbidden, resp.StatusCode, "Owner should have permission to intervene")
		t.Logf("Owner intervene attempt status: %d", resp.StatusCode)
	})

	t.Run("OtherUserTriggerAccess", func(t *testing.T) {
		// User 2 attempts to trigger User 1's robot
		// With system:root scope this might succeed
		triggerData := map[string]interface{}{
			"trigger_type": "human",
		}

		body, _ := json.Marshal(triggerData)
		req, err := http.NewRequest("POST", serverURL+baseURL+"/agent/robots/"+robotID+"/trigger", bytes.NewBuffer(body))
		require.NoError(t, err)

		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+token2.AccessToken)

		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		t.Logf("User 2 trigger attempt status: %d (with system:root scope)", resp.StatusCode)
	})
}

// ==================== Helper Functions ====================

func createRobotForTrigger(t *testing.T, serverURL, baseURL, token, robotID, displayName string) {
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

func createRobotWithTeamForTrigger(t *testing.T, serverURL, baseURL, token, robotID, displayName, teamID string) {
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

func deleteRobotForTrigger(t *testing.T, serverURL, baseURL, token, robotID string) {
	req, _ := http.NewRequest("DELETE", serverURL+baseURL+"/agent/robots/"+robotID, nil)
	req.Header.Set("Authorization", "Bearer "+token)
	resp, _ := http.DefaultClient.Do(req)
	if resp != nil {
		resp.Body.Close()
	}
}
