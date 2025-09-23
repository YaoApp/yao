package user_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/yaoapp/yao/openapi"
	"github.com/yaoapp/yao/openapi/tests/testutils"
	"github.com/yaoapp/yao/openapi/user"
)

// TestInvitationCreate tests the POST /user/teams/:team_id/invitations endpoint
func TestInvitationCreate(t *testing.T) {
	serverURL := testutils.Prepare(t)
	defer testutils.Clean()

	// Get base URL from server config
	baseURL := ""
	if openapi.Server != nil && openapi.Server.Config != nil {
		baseURL = openapi.Server.Config.BaseURL
	}

	// Register test client
	client := testutils.RegisterTestClient(t, "Invitation Test Client", []string{"https://localhost/callback"})
	defer testutils.CleanupTestClient(t, client.ClientID)

	// Get access token
	tokenInfo := testutils.ObtainAccessToken(t, serverURL, client.ClientID, client.ClientSecret, "https://localhost/callback", "openid profile")

	// Use UUID to ensure unique identifiers
	testUUID := strings.ReplaceAll(uuid.New().String(), "-", "")[:8]

	// Create test team first
	createdTeam := createTestTeam(t, serverURL, baseURL, tokenInfo.AccessToken, "Invitation Test Team "+testUUID)
	teamID := getTeamID(createdTeam)

	// Test successful invitation creation
	t.Run("CreateInvitation_Success", func(t *testing.T) {
		invitationData := map[string]interface{}{
			"user_id":     nil, // Invite unregistered user
			"member_type": "user",
			"role_id":     "user",
			"message":     "Welcome to our team!",
			"settings": map[string]interface{}{
				"send_email": true,
			},
		}

		jsonData, _ := json.Marshal(invitationData)
		url := fmt.Sprintf("%s%s/user/teams/%s/invitations", serverURL, baseURL, teamID)

		req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
		assert.NoError(t, err)
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+tokenInfo.AccessToken)

		client := &http.Client{Timeout: 10 * time.Second}
		resp, err := client.Do(req)
		assert.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusCreated, resp.StatusCode)

		var result map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&result)
		assert.NoError(t, err)

		// Should return invitation_id
		assert.Contains(t, result, "invitation_id")
		assert.NotEmpty(t, result["invitation_id"])

		invitationID := result["invitation_id"].(string)
		assert.True(t, strings.HasPrefix(invitationID, "inv_"), "invitation_id should have inv_ prefix")
	})

	// Test invitation creation with registered user
	t.Run("CreateInvitation_RegisteredUser", func(t *testing.T) {
		// Create another user to invite
		anotherTokenInfo := testutils.ObtainAccessToken(t, serverURL, client.ClientID, client.ClientSecret, "https://localhost/callback", "openid profile")

		invitationData := map[string]interface{}{
			"user_id":     anotherTokenInfo.UserID,
			"member_type": "user",
			"role_id":     "admin",
			"message":     "Join as admin!",
		}

		jsonData, _ := json.Marshal(invitationData)
		url := fmt.Sprintf("%s%s/user/teams/%s/invitations", serverURL, baseURL, teamID)

		req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
		assert.NoError(t, err)
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+tokenInfo.AccessToken)

		client := &http.Client{Timeout: 10 * time.Second}
		resp, err := client.Do(req)
		assert.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusCreated, resp.StatusCode)

		var result map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&result)
		assert.NoError(t, err)

		assert.Contains(t, result, "invitation_id")
		assert.NotEmpty(t, result["invitation_id"])
	})

	// Test missing required fields
	t.Run("CreateInvitation_MissingRoleID", func(t *testing.T) {
		invitationData := map[string]interface{}{
			"user_id":     nil,
			"member_type": "user",
			// Missing role_id
		}

		jsonData, _ := json.Marshal(invitationData)
		url := fmt.Sprintf("%s%s/user/teams/%s/invitations", serverURL, baseURL, teamID)

		req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
		assert.NoError(t, err)
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+tokenInfo.AccessToken)

		client := &http.Client{Timeout: 10 * time.Second}
		resp, err := client.Do(req)
		assert.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})

	// Test non-existent team
	t.Run("CreateInvitation_NonExistentTeam", func(t *testing.T) {
		invitationData := map[string]interface{}{
			"user_id":     nil,
			"member_type": "user",
			"role_id":     "user",
		}

		jsonData, _ := json.Marshal(invitationData)
		url := fmt.Sprintf("%s%s/user/teams/non-existent-team/invitations", serverURL, baseURL)

		req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
		assert.NoError(t, err)
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+tokenInfo.AccessToken)

		client := &http.Client{Timeout: 10 * time.Second}
		resp, err := client.Do(req)
		assert.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusNotFound, resp.StatusCode)
	})

	// Test unauthorized access
	t.Run("CreateInvitation_Unauthorized", func(t *testing.T) {
		invitationData := map[string]interface{}{
			"user_id":     nil,
			"member_type": "user",
			"role_id":     "user",
		}

		jsonData, _ := json.Marshal(invitationData)
		url := fmt.Sprintf("%s%s/user/teams/%s/invitations", serverURL, baseURL, teamID)

		req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
		assert.NoError(t, err)
		req.Header.Set("Content-Type", "application/json")
		// No Authorization header

		client := &http.Client{Timeout: 10 * time.Second}
		resp, err := client.Do(req)
		assert.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	})
}

// TestInvitationList tests the GET /user/teams/:team_id/invitations endpoint
func TestInvitationList(t *testing.T) {
	serverURL := testutils.Prepare(t)
	defer testutils.Clean()

	// Get base URL from server config
	baseURL := ""
	if openapi.Server != nil && openapi.Server.Config != nil {
		baseURL = openapi.Server.Config.BaseURL
	}

	// Register test client
	client := testutils.RegisterTestClient(t, "Invitation List Test Client", []string{"https://localhost/callback"})
	defer testutils.CleanupTestClient(t, client.ClientID)

	// Get access token
	tokenInfo := testutils.ObtainAccessToken(t, serverURL, client.ClientID, client.ClientSecret, "https://localhost/callback", "openid profile")

	// Use UUID to ensure unique identifiers
	testUUID := strings.ReplaceAll(uuid.New().String(), "-", "")[:8]

	// Create test team
	createdTeam := createTestTeam(t, serverURL, baseURL, tokenInfo.AccessToken, "Invitation List Test Team "+testUUID)
	teamID := getTeamID(createdTeam)

	// Create some test invitations
	invitationIDs := make([]string, 0)
	for i := 0; i < 3; i++ {
		invitationID := createTestInvitationWithMessage(t, serverURL, baseURL, tokenInfo.AccessToken, teamID, "", fmt.Sprintf("Test invitation %d", i+1))
		invitationIDs = append(invitationIDs, invitationID)
	}

	// Test successful list
	t.Run("ListInvitations_Success", func(t *testing.T) {
		url := fmt.Sprintf("%s%s/user/teams/%s/invitations", serverURL, baseURL, teamID)

		req, err := http.NewRequest("GET", url, nil)
		assert.NoError(t, err)
		req.Header.Set("Authorization", "Bearer "+tokenInfo.AccessToken)

		client := &http.Client{Timeout: 10 * time.Second}
		resp, err := client.Do(req)
		assert.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var result map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&result)
		assert.NoError(t, err)

		// Should contain pagination info
		assert.Contains(t, result, "data")
		assert.Contains(t, result, "total")

		data := result["data"].([]interface{})
		assert.True(t, len(data) >= 3) // At least our 3 test invitations
	})

	// Test with pagination
	t.Run("ListInvitations_Pagination", func(t *testing.T) {
		url := fmt.Sprintf("%s%s/user/teams/%s/invitations?page=1&pagesize=2", serverURL, baseURL, teamID)

		req, err := http.NewRequest("GET", url, nil)
		assert.NoError(t, err)
		req.Header.Set("Authorization", "Bearer "+tokenInfo.AccessToken)

		client := &http.Client{Timeout: 10 * time.Second}
		resp, err := client.Do(req)
		assert.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var result map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&result)
		assert.NoError(t, err)

		data := result["data"].([]interface{})
		assert.True(t, len(data) <= 2) // Should respect pagesize
	})

	// Test status filter
	t.Run("ListInvitations_StatusFilter", func(t *testing.T) {
		url := fmt.Sprintf("%s%s/user/teams/%s/invitations?status=pending", serverURL, baseURL, teamID)

		req, err := http.NewRequest("GET", url, nil)
		assert.NoError(t, err)
		req.Header.Set("Authorization", "Bearer "+tokenInfo.AccessToken)

		client := &http.Client{Timeout: 10 * time.Second}
		resp, err := client.Do(req)
		assert.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var result map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&result)
		assert.NoError(t, err)

		data := result["data"].([]interface{})
		// All returned invitations should be pending
		for _, item := range data {
			invitation := item.(map[string]interface{})
			assert.Equal(t, "pending", invitation["status"])
		}
	})

	// Test unauthorized access
	t.Run("ListInvitations_Unauthorized", func(t *testing.T) {
		url := fmt.Sprintf("%s%s/user/teams/%s/invitations", serverURL, baseURL, teamID)

		req, err := http.NewRequest("GET", url, nil)
		assert.NoError(t, err)
		// No Authorization header

		client := &http.Client{Timeout: 10 * time.Second}
		resp, err := client.Do(req)
		assert.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	})

	// Test non-existent team
	t.Run("ListInvitations_NonExistentTeam", func(t *testing.T) {
		url := fmt.Sprintf("%s%s/user/teams/non-existent-team/invitations", serverURL, baseURL)

		req, err := http.NewRequest("GET", url, nil)
		assert.NoError(t, err)
		req.Header.Set("Authorization", "Bearer "+tokenInfo.AccessToken)

		client := &http.Client{Timeout: 10 * time.Second}
		resp, err := client.Do(req)
		assert.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusNotFound, resp.StatusCode)
	})
}

// TestInvitationGet tests the GET /user/teams/:team_id/invitations/:invitation_id endpoint
func TestInvitationGet(t *testing.T) {
	serverURL := testutils.Prepare(t)
	defer testutils.Clean()

	// Get base URL from server config
	baseURL := ""
	if openapi.Server != nil && openapi.Server.Config != nil {
		baseURL = openapi.Server.Config.BaseURL
	}

	// Register test client
	client := testutils.RegisterTestClient(t, "Invitation Get Test Client", []string{"https://localhost/callback"})
	defer testutils.CleanupTestClient(t, client.ClientID)

	// Get access token
	tokenInfo := testutils.ObtainAccessToken(t, serverURL, client.ClientID, client.ClientSecret, "https://localhost/callback", "openid profile")

	// Use UUID to ensure unique identifiers
	testUUID := strings.ReplaceAll(uuid.New().String(), "-", "")[:8]

	// Create test team
	createdTeam := createTestTeam(t, serverURL, baseURL, tokenInfo.AccessToken, "Invitation Get Test Team "+testUUID)
	teamID := getTeamID(createdTeam)

	// Create test invitation
	invitationID := createTestInvitation(t, serverURL, baseURL, tokenInfo.AccessToken, teamID, "") // Unregistered user

	// Test successful get
	t.Run("GetInvitation_Success", func(t *testing.T) {
		url := fmt.Sprintf("%s%s/user/teams/%s/invitations/%s", serverURL, baseURL, teamID, invitationID)

		req, err := http.NewRequest("GET", url, nil)
		assert.NoError(t, err)
		req.Header.Set("Authorization", "Bearer "+tokenInfo.AccessToken)

		client := &http.Client{Timeout: 10 * time.Second}
		resp, err := client.Do(req)
		assert.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var result user.InvitationDetailResponse
		err = json.NewDecoder(resp.Body).Decode(&result)
		assert.NoError(t, err)

		assert.Equal(t, teamID, result.TeamID)
		assert.Equal(t, "pending", result.Status)
		assert.NotEmpty(t, result.InvitationToken)
		assert.NotEmpty(t, result.InvitedAt)
	})

	// Test non-existent invitation
	t.Run("GetInvitation_NotFound", func(t *testing.T) {
		url := fmt.Sprintf("%s%s/user/teams/%s/invitations/non-existent-invitation", serverURL, baseURL, teamID)

		req, err := http.NewRequest("GET", url, nil)
		assert.NoError(t, err)
		req.Header.Set("Authorization", "Bearer "+tokenInfo.AccessToken)

		client := &http.Client{Timeout: 10 * time.Second}
		resp, err := client.Do(req)
		assert.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusNotFound, resp.StatusCode)
	})

	// Test unauthorized access
	t.Run("GetInvitation_Unauthorized", func(t *testing.T) {
		url := fmt.Sprintf("%s%s/user/teams/%s/invitations/%s", serverURL, baseURL, teamID, invitationID)

		req, err := http.NewRequest("GET", url, nil)
		assert.NoError(t, err)
		// No Authorization header

		client := &http.Client{Timeout: 10 * time.Second}
		resp, err := client.Do(req)
		assert.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	})

	// Test wrong team
	t.Run("GetInvitation_WrongTeam", func(t *testing.T) {
		// Create another team
		anotherCreatedTeam := createTestTeam(t, serverURL, baseURL, tokenInfo.AccessToken, "Another Team "+testUUID)
		anotherTeamID := getTeamID(anotherCreatedTeam)

		url := fmt.Sprintf("%s%s/user/teams/%s/invitations/%s", serverURL, baseURL, anotherTeamID, invitationID)

		req, err := http.NewRequest("GET", url, nil)
		assert.NoError(t, err)
		req.Header.Set("Authorization", "Bearer "+tokenInfo.AccessToken)

		client := &http.Client{Timeout: 10 * time.Second}
		resp, err := client.Do(req)
		assert.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusNotFound, resp.StatusCode)
	})
}

// TestInvitationResend tests the PUT /user/teams/:team_id/invitations/:invitation_id/resend endpoint
func TestInvitationResend(t *testing.T) {
	serverURL := testutils.Prepare(t)
	defer testutils.Clean()

	// Get base URL from server config
	baseURL := ""
	if openapi.Server != nil && openapi.Server.Config != nil {
		baseURL = openapi.Server.Config.BaseURL
	}

	// Register test client
	client := testutils.RegisterTestClient(t, "Invitation Resend Test Client", []string{"https://localhost/callback"})
	defer testutils.CleanupTestClient(t, client.ClientID)

	// Get access token
	tokenInfo := testutils.ObtainAccessToken(t, serverURL, client.ClientID, client.ClientSecret, "https://localhost/callback", "openid profile")

	// Use UUID to ensure unique identifiers
	testUUID := strings.ReplaceAll(uuid.New().String(), "-", "")[:8]

	// Create test team
	createdTeam := createTestTeam(t, serverURL, baseURL, tokenInfo.AccessToken, "Invitation Resend Test Team "+testUUID)
	teamID := getTeamID(createdTeam)

	// Create test invitation
	invitationID := createTestInvitation(t, serverURL, baseURL, tokenInfo.AccessToken, teamID, "") // Unregistered user

	// Test successful resend
	t.Run("ResendInvitation_Success", func(t *testing.T) {
		url := fmt.Sprintf("%s%s/user/teams/%s/invitations/%s/resend", serverURL, baseURL, teamID, invitationID)

		req, err := http.NewRequest("PUT", url, nil)
		assert.NoError(t, err)
		req.Header.Set("Authorization", "Bearer "+tokenInfo.AccessToken)

		client := &http.Client{Timeout: 10 * time.Second}
		resp, err := client.Do(req)
		assert.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var result map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&result)
		assert.NoError(t, err)

		assert.Equal(t, "Invitation resent successfully", result["message"])
	})

	// Test non-existent invitation
	t.Run("ResendInvitation_NotFound", func(t *testing.T) {
		url := fmt.Sprintf("%s%s/user/teams/%s/invitations/non-existent-invitation/resend", serverURL, baseURL, teamID)

		req, err := http.NewRequest("PUT", url, nil)
		assert.NoError(t, err)
		req.Header.Set("Authorization", "Bearer "+tokenInfo.AccessToken)

		client := &http.Client{Timeout: 10 * time.Second}
		resp, err := client.Do(req)
		assert.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusNotFound, resp.StatusCode)
	})

	// Test unauthorized access
	t.Run("ResendInvitation_Unauthorized", func(t *testing.T) {
		url := fmt.Sprintf("%s%s/user/teams/%s/invitations/%s/resend", serverURL, baseURL, teamID, invitationID)

		req, err := http.NewRequest("PUT", url, nil)
		assert.NoError(t, err)
		// No Authorization header

		client := &http.Client{Timeout: 10 * time.Second}
		resp, err := client.Do(req)
		assert.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	})
}

// TestMultipleInvitationCreation tests creating multiple invitations for unregistered users
func TestMultipleInvitationCreation(t *testing.T) {
	serverURL := testutils.Prepare(t)
	defer testutils.Clean()

	// Get base URL from server config
	baseURL := ""
	if openapi.Server != nil && openapi.Server.Config != nil {
		baseURL = openapi.Server.Config.BaseURL
	}

	// Register test client
	client := testutils.RegisterTestClient(t, "Multiple Invitation Test Client", []string{"https://localhost/callback"})
	defer testutils.CleanupTestClient(t, client.ClientID)

	// Get access token
	tokenInfo := testutils.ObtainAccessToken(t, serverURL, client.ClientID, client.ClientSecret, "https://localhost/callback", "openid profile")

	// Use UUID to ensure unique identifiers
	testUUID := strings.ReplaceAll(uuid.New().String(), "-", "")[:8]

	// Create test team
	createdTeam := createTestTeam(t, serverURL, baseURL, tokenInfo.AccessToken, "Multiple Invitation Test Team "+testUUID)
	teamID := getTeamID(createdTeam)

	// Test creating multiple invitations for unregistered users
	t.Run("CreateMultipleInvitations", func(t *testing.T) {
		invitationIDs := make([]string, 0)

		for i := 0; i < 3; i++ {
			t.Logf("Creating invitation %d", i+1)

			// Create invitation data with different messages to ensure uniqueness
			invitationData := map[string]interface{}{
				"member_type": "user",
				"role_id":     "user",
				"message":     fmt.Sprintf("Test invitation %d - %s", i+1, testUUID),
				"user_id":     nil, // Explicitly set to nil for unregistered users
			}

			// Convert to JSON
			jsonData, err := json.Marshal(invitationData)
			assert.NoError(t, err)

			// Make API call
			url := fmt.Sprintf("%s%s/user/teams/%s/invitations", serverURL, baseURL, teamID)
			t.Logf("Request URL: %s", url)
			t.Logf("Request data: %s", string(jsonData))

			req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
			assert.NoError(t, err)
			req.Header.Set("Authorization", "Bearer "+tokenInfo.AccessToken)
			req.Header.Set("Content-Type", "application/json")

			client := &http.Client{Timeout: 10 * time.Second}
			resp, err := client.Do(req)
			assert.NoError(t, err)
			defer resp.Body.Close()

			// Read response
			bodyBytes, err := io.ReadAll(resp.Body)
			assert.NoError(t, err)

			t.Logf("Response status: %d", resp.StatusCode)
			t.Logf("Response body: %s", string(bodyBytes))

			if resp.StatusCode != http.StatusCreated {
				t.Errorf("Expected 201 but got %d for invitation %d: %s", resp.StatusCode, i+1, string(bodyBytes))
				continue
			}

			var result map[string]interface{}
			err = json.Unmarshal(bodyBytes, &result)
			assert.NoError(t, err)

			invitationID, ok := result["invitation_id"].(string)
			assert.True(t, ok, "invitation_id should be a string")
			assert.NotEmpty(t, invitationID, "invitation_id should not be empty")

			invitationIDs = append(invitationIDs, invitationID)
			t.Logf("Successfully created invitation %d with ID: %s", i+1, invitationID)
		}

		// Verify we created all 3 invitations
		assert.Equal(t, 3, len(invitationIDs), "Should have created 3 invitations")

		// Verify all invitation IDs are unique
		uniqueIDs := make(map[string]bool)
		for _, id := range invitationIDs {
			assert.False(t, uniqueIDs[id], "Invitation ID should be unique: %s", id)
			uniqueIDs[id] = true
		}
	})
}

// TestInvitationDelete tests the DELETE /user/teams/:team_id/invitations/:invitation_id endpoint
func TestInvitationDelete(t *testing.T) {
	serverURL := testutils.Prepare(t)
	defer testutils.Clean()

	// Get base URL from server config
	baseURL := ""
	if openapi.Server != nil && openapi.Server.Config != nil {
		baseURL = openapi.Server.Config.BaseURL
	}

	// Register test client
	client := testutils.RegisterTestClient(t, "Invitation Delete Test Client", []string{"https://localhost/callback"})
	defer testutils.CleanupTestClient(t, client.ClientID)

	// Get access token
	tokenInfo := testutils.ObtainAccessToken(t, serverURL, client.ClientID, client.ClientSecret, "https://localhost/callback", "openid profile")

	// Use UUID to ensure unique identifiers
	testUUID := strings.ReplaceAll(uuid.New().String(), "-", "")[:8]

	// Create test team
	createdTeam := createTestTeam(t, serverURL, baseURL, tokenInfo.AccessToken, "Invitation Delete Test Team "+testUUID)
	teamID := getTeamID(createdTeam)

	// Create test invitation
	invitationID := createTestInvitation(t, serverURL, baseURL, tokenInfo.AccessToken, teamID, "") // Unregistered user

	// Test successful delete
	t.Run("DeleteInvitation_Success", func(t *testing.T) {
		url := fmt.Sprintf("%s%s/user/teams/%s/invitations/%s", serverURL, baseURL, teamID, invitationID)

		req, err := http.NewRequest("DELETE", url, nil)
		assert.NoError(t, err)
		req.Header.Set("Authorization", "Bearer "+tokenInfo.AccessToken)

		client := &http.Client{Timeout: 10 * time.Second}
		resp, err := client.Do(req)
		assert.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var result map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&result)
		assert.NoError(t, err)

		assert.Equal(t, "Invitation cancelled successfully", result["message"])

		// Verify invitation is deleted by trying to get it
		getURL := fmt.Sprintf("%s%s/user/teams/%s/invitations/%s", serverURL, baseURL, teamID, invitationID)
		getReq, err := http.NewRequest("GET", getURL, nil)
		assert.NoError(t, err)
		getReq.Header.Set("Authorization", "Bearer "+tokenInfo.AccessToken)

		getResp, err := client.Do(getReq)
		assert.NoError(t, err)
		defer getResp.Body.Close()

		assert.Equal(t, http.StatusNotFound, getResp.StatusCode)
	})

	// Test non-existent invitation
	t.Run("DeleteInvitation_NotFound", func(t *testing.T) {
		url := fmt.Sprintf("%s%s/user/teams/%s/invitations/non-existent-invitation", serverURL, baseURL, teamID)

		req, err := http.NewRequest("DELETE", url, nil)
		assert.NoError(t, err)
		req.Header.Set("Authorization", "Bearer "+tokenInfo.AccessToken)

		client := &http.Client{Timeout: 10 * time.Second}
		resp, err := client.Do(req)
		assert.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusNotFound, resp.StatusCode)
	})

	// Test unauthorized access
	t.Run("DeleteInvitation_Unauthorized", func(t *testing.T) {
		// Create another invitation for this test
		anotherInvitationID := createTestInvitation(t, serverURL, baseURL, tokenInfo.AccessToken, teamID, "") // Unregistered user

		url := fmt.Sprintf("%s%s/user/teams/%s/invitations/%s", serverURL, baseURL, teamID, anotherInvitationID)

		req, err := http.NewRequest("DELETE", url, nil)
		assert.NoError(t, err)
		// No Authorization header

		client := &http.Client{Timeout: 10 * time.Second}
		resp, err := client.Do(req)
		assert.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	})
}

// Helper functions

// createTestInvitation creates a test invitation and returns its ID
func createTestInvitation(t *testing.T, serverURL, baseURL, accessToken, teamID, userID string) string {
	return createTestInvitationWithMessage(t, serverURL, baseURL, accessToken, teamID, userID, "Test invitation")
}

// createTestInvitationWithMessage creates a test invitation with custom message and returns its ID
func createTestInvitationWithMessage(t *testing.T, serverURL, baseURL, accessToken, teamID, userID, message string) string {
	return createTestInvitationWithRoleAndMessage(t, serverURL, baseURL, accessToken, teamID, userID, "user", message)
}

// createTestInvitationWithRoleAndMessage creates a test invitation with custom role and message and returns its ID
func createTestInvitationWithRoleAndMessage(t *testing.T, serverURL, baseURL, accessToken, teamID, userID, roleID, message string) string {
	invitationData := map[string]interface{}{
		"member_type": "user",
		"role_id":     roleID,
		"message":     message,
	}

	// Set user_id - nil for unregistered users, actual ID for registered users
	if userID != "" {
		invitationData["user_id"] = userID
	} else {
		invitationData["user_id"] = nil // Explicitly set to nil for unregistered users
	}

	jsonData, _ := json.Marshal(invitationData)
	url := fmt.Sprintf("%s%s/user/teams/%s/invitations", serverURL, baseURL, teamID)

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	assert.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+accessToken)

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	assert.NoError(t, err)
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		var body map[string]interface{}
		json.NewDecoder(resp.Body).Decode(&body)
		t.Logf("Request data: %s", string(jsonData))
		t.Logf("Request URL: %s", url)
		t.Logf("Response status: %d", resp.StatusCode)
		t.Logf("Response body: %v", body)
		t.Fatalf("Expected 201 but got %d: %v", resp.StatusCode, body)
	}

	var result map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&result)
	assert.NoError(t, err)

	invitationID, ok := result["invitation_id"].(string)
	if !ok {
		t.Fatalf("invitation_id not found in response: %v", result)
	}
	assert.NotEmpty(t, invitationID)

	return invitationID
}
