package openapi_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/yaoapp/yao/openapi"
	"github.com/yaoapp/yao/openapi/tests/testutils"
)

// TestListResults tests the results listing endpoint
// GET /v1/agent/robots/:id/results
func TestListResults(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping results tests in short mode")
	}

	serverURL := testutils.Prepare(t)
	defer testutils.Clean()

	baseURL := ""
	if openapi.Server != nil && openapi.Server.Config != nil {
		baseURL = openapi.Server.Config.BaseURL
	}

	// Register test client and get token
	client := testutils.RegisterTestClient(t, "Results List Test Client", []string{"https://localhost/callback"})
	defer testutils.CleanupTestClient(t, client.ClientID)
	tokenInfo := testutils.ObtainAccessToken(t, serverURL, client.ClientID, client.ClientSecret, "https://localhost/callback", "openid profile")

	// Create a test robot
	robotID := fmt.Sprintf("test_results_list_%d", time.Now().UnixNano())
	createRobot(t, serverURL, baseURL, tokenInfo.AccessToken, robotID, "Results List Robot")
	defer deleteRobot(t, serverURL, baseURL, tokenInfo.AccessToken, robotID)

	t.Run("ListResultsSuccess", func(t *testing.T) {
		req, err := http.NewRequest("GET", serverURL+baseURL+"/agent/robots/"+robotID+"/results", nil)
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

		// Verify result items structure (if any results exist)
		data, ok := response["data"].([]interface{})
		if ok && len(data) > 0 {
			result := data[0].(map[string]interface{})
			// Basic fields should exist
			assert.Contains(t, result, "id")
			assert.Contains(t, result, "member_id")
			assert.Contains(t, result, "trigger_type")
			assert.Contains(t, result, "status")
			assert.Contains(t, result, "name")
			assert.Contains(t, result, "summary")
			assert.Contains(t, result, "has_attachments")
			t.Logf("Result response fields: id=%v, name=%v, summary=%v",
				result["id"], result["name"], result["summary"])
		}
	})

	t.Run("ListResultsWithPagination", func(t *testing.T) {
		req, err := http.NewRequest("GET", serverURL+baseURL+"/agent/robots/"+robotID+"/results?page=1&pagesize=5", nil)
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

	t.Run("ListResultsWithTriggerTypeFilter", func(t *testing.T) {
		req, err := http.NewRequest("GET", serverURL+baseURL+"/agent/robots/"+robotID+"/results?trigger_type=clock", nil)
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

	t.Run("ListResultsWithKeywordFilter", func(t *testing.T) {
		req, err := http.NewRequest("GET", serverURL+baseURL+"/agent/robots/"+robotID+"/results?keyword=test", nil)
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

	t.Run("ListResultsRobotNotFound", func(t *testing.T) {
		req, err := http.NewRequest("GET", serverURL+baseURL+"/agent/robots/non_existent_robot/results", nil)
		require.NoError(t, err)

		req.Header.Set("Authorization", "Bearer "+tokenInfo.AccessToken)

		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusNotFound, resp.StatusCode)
	})

	t.Run("ListResultsUnauthorized", func(t *testing.T) {
		req, err := http.NewRequest("GET", serverURL+baseURL+"/agent/robots/"+robotID+"/results", nil)
		require.NoError(t, err)
		// No Authorization header

		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	})
}

// TestGetResult tests the result detail endpoint
// GET /v1/agent/robots/:id/results/:result_id
func TestGetResult(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping result tests in short mode")
	}

	serverURL := testutils.Prepare(t)
	defer testutils.Clean()

	baseURL := ""
	if openapi.Server != nil && openapi.Server.Config != nil {
		baseURL = openapi.Server.Config.BaseURL
	}

	// Register test client and get token
	client := testutils.RegisterTestClient(t, "Result Get Test Client", []string{"https://localhost/callback"})
	defer testutils.CleanupTestClient(t, client.ClientID)
	tokenInfo := testutils.ObtainAccessToken(t, serverURL, client.ClientID, client.ClientSecret, "https://localhost/callback", "openid profile")

	// Create a test robot
	robotID := fmt.Sprintf("test_result_get_%d", time.Now().UnixNano())
	createRobot(t, serverURL, baseURL, tokenInfo.AccessToken, robotID, "Result Get Robot")
	defer deleteRobot(t, serverURL, baseURL, tokenInfo.AccessToken, robotID)

	t.Run("GetResultNotFound", func(t *testing.T) {
		req, err := http.NewRequest("GET", serverURL+baseURL+"/agent/robots/"+robotID+"/results/non_existent_result", nil)
		require.NoError(t, err)

		req.Header.Set("Authorization", "Bearer "+tokenInfo.AccessToken)

		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusNotFound, resp.StatusCode)
	})

	t.Run("GetResultRobotNotFound", func(t *testing.T) {
		req, err := http.NewRequest("GET", serverURL+baseURL+"/agent/robots/non_existent_robot/results/some_result", nil)
		require.NoError(t, err)

		req.Header.Set("Authorization", "Bearer "+tokenInfo.AccessToken)

		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusNotFound, resp.StatusCode)
	})

	t.Run("GetResultUnauthorized", func(t *testing.T) {
		req, err := http.NewRequest("GET", serverURL+baseURL+"/agent/robots/"+robotID+"/results/some_result", nil)
		require.NoError(t, err)
		// No Authorization header

		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	})
}

// TestListActivities tests the activities listing endpoint
// GET /v1/agent/robots/activities
func TestListActivities(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping activities tests in short mode")
	}

	serverURL := testutils.Prepare(t)
	defer testutils.Clean()

	baseURL := ""
	if openapi.Server != nil && openapi.Server.Config != nil {
		baseURL = openapi.Server.Config.BaseURL
	}

	// Register test client and get token
	client := testutils.RegisterTestClient(t, "Activities List Test Client", []string{"https://localhost/callback"})
	defer testutils.CleanupTestClient(t, client.ClientID)
	tokenInfo := testutils.ObtainAccessToken(t, serverURL, client.ClientID, client.ClientSecret, "https://localhost/callback", "openid profile")

	t.Run("ListActivitiesSuccess", func(t *testing.T) {
		req, err := http.NewRequest("GET", serverURL+baseURL+"/agent/robots/activities", nil)
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

		// Verify data field exists
		assert.Contains(t, response, "data")

		// Verify activity items structure (if any activities exist)
		data, ok := response["data"].([]interface{})
		if ok && len(data) > 0 {
			activity := data[0].(map[string]interface{})
			// Basic fields should exist
			assert.Contains(t, activity, "type")
			assert.Contains(t, activity, "robot_id")
			assert.Contains(t, activity, "execution_id")
			assert.Contains(t, activity, "message")
			assert.Contains(t, activity, "timestamp")
			t.Logf("Activity response fields: type=%v, robot_id=%v, message=%v",
				activity["type"], activity["robot_id"], activity["message"])
		}
	})

	t.Run("ListActivitiesWithLimit", func(t *testing.T) {
		req, err := http.NewRequest("GET", serverURL+baseURL+"/agent/robots/activities?limit=10", nil)
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

	t.Run("ListActivitiesWithTypeFilter", func(t *testing.T) {
		// Test filtering by type: execution.completed
		req, err := http.NewRequest("GET", serverURL+baseURL+"/agent/robots/activities?type=execution.completed", nil)
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

		// Verify all returned activities are of the specified type
		data, ok := response["data"].([]interface{})
		if ok && len(data) > 0 {
			for _, item := range data {
				activity := item.(map[string]interface{})
				assert.Equal(t, "execution.completed", activity["type"])
			}
		}
	})

	t.Run("ListActivitiesWithTypeStarted", func(t *testing.T) {
		// Test filtering by type: execution.started
		req, err := http.NewRequest("GET", serverURL+baseURL+"/agent/robots/activities?type=execution.started", nil)
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

		// Verify all returned activities are of the specified type
		data, ok := response["data"].([]interface{})
		if ok && len(data) > 0 {
			for _, item := range data {
				activity := item.(map[string]interface{})
				assert.Equal(t, "execution.started", activity["type"])
			}
		}
	})

	t.Run("ListActivitiesWithInvalidType", func(t *testing.T) {
		// Test with an invalid/unknown type - should return empty data (not error)
		req, err := http.NewRequest("GET", serverURL+baseURL+"/agent/robots/activities?type=invalid.type", nil)
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
		// Invalid type should return empty array
		data, ok := response["data"].([]interface{})
		assert.True(t, ok)
		assert.Empty(t, data)
	})

	t.Run("ListActivitiesWithSince", func(t *testing.T) {
		// Use a timestamp in the past - URL encode properly
		since := time.Now().Add(-24 * time.Hour).UTC().Format(time.RFC3339)
		// URL encode the since parameter (+ becomes %2B, : stays)
		encodedSince := url.QueryEscape(since)
		req, err := http.NewRequest("GET", serverURL+baseURL+"/agent/robots/activities?since="+encodedSince, nil)
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

	t.Run("ListActivitiesWithInvalidSince", func(t *testing.T) {
		req, err := http.NewRequest("GET", serverURL+baseURL+"/agent/robots/activities?since=invalid_date", nil)
		require.NoError(t, err)

		req.Header.Set("Authorization", "Bearer "+tokenInfo.AccessToken)

		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		// Should return 400 Bad Request for invalid date format
		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})

	t.Run("ListActivitiesUnauthorized", func(t *testing.T) {
		req, err := http.NewRequest("GET", serverURL+baseURL+"/agent/robots/activities", nil)
		require.NoError(t, err)
		// No Authorization header

		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	})
}

// TestResultsPermissions tests result permission inheritance from robot
func TestResultsPermissions(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping results permission tests in short mode")
	}

	serverURL := testutils.Prepare(t)
	defer testutils.Clean()

	baseURL := ""
	if openapi.Server != nil && openapi.Server.Config != nil {
		baseURL = openapi.Server.Config.BaseURL
	}

	// Register test client
	client := testutils.RegisterTestClient(t, "Results Permission Test Client", []string{"https://localhost/callback"})
	defer testutils.CleanupTestClient(t, client.ClientID)

	// Create User 1
	token1 := testutils.ObtainAccessToken(t, serverURL, client.ClientID, client.ClientSecret, "https://localhost/callback", "openid profile")
	user1ID := token1.UserID

	// Create User 2
	token2 := testutils.ObtainAccessToken(t, serverURL, client.ClientID, client.ClientSecret, "https://localhost/callback", "openid profile")

	// User 1 creates a robot
	robotID := fmt.Sprintf("test_results_perm_%d", time.Now().UnixNano())
	createRobotWithTeam(t, serverURL, baseURL, token1.AccessToken, robotID, "Results Permission Test Robot", user1ID)
	defer deleteRobot(t, serverURL, baseURL, token1.AccessToken, robotID)

	t.Run("OwnerCanListResults", func(t *testing.T) {
		req, err := http.NewRequest("GET", serverURL+baseURL+"/agent/robots/"+robotID+"/results", nil)
		require.NoError(t, err)

		req.Header.Set("Authorization", "Bearer "+token1.AccessToken)

		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)
		t.Logf("Owner (User 1) successfully listed results for their robot")
	})

	t.Run("OtherUserResultsAccess", func(t *testing.T) {
		// User 2 attempts to list results for User 1's robot
		// With system:root scope this might succeed
		req, err := http.NewRequest("GET", serverURL+baseURL+"/agent/robots/"+robotID+"/results", nil)
		require.NoError(t, err)

		req.Header.Set("Authorization", "Bearer "+token2.AccessToken)

		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		t.Logf("User 2 results list attempt status: %d (with system:root scope)", resp.StatusCode)
	})
}
