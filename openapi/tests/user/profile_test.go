package user_test

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/yaoapp/yao/openapi"
	"github.com/yaoapp/yao/openapi/oauth"
	"github.com/yaoapp/yao/openapi/tests/testutils"
)

// TestProfileGet tests the GET /user/profile endpoint
func TestProfileGet(t *testing.T) {
	serverURL := testutils.Prepare(t)
	defer testutils.Clean()

	// Get base URL from server config
	baseURL := ""
	if openapi.Server != nil && openapi.Server.Config != nil {
		baseURL = openapi.Server.Config.BaseURL
	}

	// Register a test client
	client := testutils.RegisterTestClient(t, "Profile Test Client", []string{"https://localhost/callback"})
	defer testutils.CleanupTestClient(t, client.ClientID)

	// Create a test user with root permissions
	tokenInfo := testutils.ObtainAccessTokenWithRootPermission(t, serverURL, client.ClientID, client.ClientSecret, "https://localhost/callback", "openid profile email")

	// Test 1: Get basic profile without optional parameters
	t.Run("GetBasicProfile", func(t *testing.T) {
		fullURL := serverURL + baseURL + "/user/profile"
		t.Logf("Requesting URL: %s", fullURL)

		req, err := http.NewRequest("GET", fullURL, nil)
		assert.NoError(t, err)

		// Add authorization header
		req.Header.Set("Authorization", "Bearer "+tokenInfo.AccessToken)

		resp, err := http.DefaultClient.Do(req)
		assert.NoError(t, err)
		defer resp.Body.Close()

		t.Logf("Response status: %d", resp.StatusCode)
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var result map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&result)
		assert.NoError(t, err)

		// Verify basic OIDC fields
		assert.NotEmpty(t, result["sub"], "sub field should be present")
		assert.NotEmpty(t, result["yao:user_id"], "yao:user_id field should be present")

		// Team, member, and type should NOT be present in basic profile
		assert.Nil(t, result["yao:team"], "team should not be present without team=true")
		assert.Nil(t, result["member"], "member should not be present without member=true")
		assert.Nil(t, result["yao:type"], "type should not be present without type=true")

		t.Logf("Basic profile retrieved successfully for user: %s", result["yao:user_id"])
	})

	// Test 2: Get profile with team parameter (but no team context in token)
	t.Run("GetProfileWithTeamParameter", func(t *testing.T) {
		// Request profile with team=true but without team context
		// This should return profile without team info (since no team context)
		req, err := http.NewRequest("GET", serverURL+baseURL+"/user/profile?team=true", nil)
		assert.NoError(t, err)
		req.Header.Set("Authorization", "Bearer "+tokenInfo.AccessToken)

		resp, err := http.DefaultClient.Do(req)
		assert.NoError(t, err)
		defer resp.Body.Close()

		bodyBytes, _ := io.ReadAll(resp.Body)
		t.Logf("Response status: %d", resp.StatusCode)

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var result map[string]interface{}
		err = json.Unmarshal(bodyBytes, &result)
		assert.NoError(t, err)

		// Without team context in token, team info should not be present
		assert.Nil(t, result["yao:team"], "team info should not be present without team context")
		assert.Empty(t, result["yao:team_id"], "team_id should not be present without team context")

		t.Logf("Profile request with team parameter (but no team context) handled correctly")
	})

	// Test 3: Get profile with member parameter (but no team context)
	t.Run("GetProfileWithMemberParameter", func(t *testing.T) {
		// Request profile with member=true but without team context
		req, err := http.NewRequest("GET", serverURL+baseURL+"/user/profile?member=true", nil)
		assert.NoError(t, err)
		req.Header.Set("Authorization", "Bearer "+tokenInfo.AccessToken)

		resp, err := http.DefaultClient.Do(req)
		assert.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var result map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&result)
		assert.NoError(t, err)

		// Without team context, member info should not be present
		assert.Nil(t, result["member"], "member info should not be present without team context")

		t.Logf("Profile request with member parameter (but no team context) handled correctly")
	})

	// Test 4: Get profile with type information
	t.Run("GetProfileWithType", func(t *testing.T) {
		// Create a user type
		provider := testutils.GetUserProvider(t)
		ctx := context.Background()

		typeData := map[string]interface{}{
			"type_id":     fmt.Sprintf("test_type_%d", time.Now().UnixNano()),
			"name":        "Test User Type",
			"locale":      "en",
			"description": "Type for profile testing",
			"is_active":   true,
			"created_at":  time.Now(),
			"updated_at":  time.Now(),
		}

		typeID, err := provider.CreateType(ctx, typeData)
		assert.NoError(t, err)
		t.Logf("Created test type: %s", typeID)

		// Update user with type_id
		err = provider.UpdateUser(ctx, tokenInfo.UserID, map[string]interface{}{
			"type_id": typeID,
		})
		assert.NoError(t, err)
		t.Logf("Updated user with type: %s", typeID)

		// Get profile with type=true
		req, err := http.NewRequest("GET", serverURL+baseURL+"/user/profile?type=true", nil)
		assert.NoError(t, err)
		req.Header.Set("Authorization", "Bearer "+tokenInfo.AccessToken)

		resp, err := http.DefaultClient.Do(req)
		assert.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var result map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&result)
		assert.NoError(t, err)

		// Verify type fields are present
		assert.NotEmpty(t, result["yao:type_id"], "type_id should be present")
		assert.NotNil(t, result["yao:type"], "type info should be present")

		if typeInfo, ok := result["yao:type"].(map[string]interface{}); ok {
			assert.Equal(t, typeID, typeInfo["type_id"])
			assert.Equal(t, "Test User Type", typeInfo["name"])
			assert.Equal(t, "en", typeInfo["locale"])
		}

		t.Logf("Profile with type info retrieved successfully")

		// Cleanup
		err = provider.DeleteType(ctx, typeID)
		assert.NoError(t, err)
	})

	// Test 5: Get profile with all optional parameters (without team context)
	t.Run("GetProfileWithAllOptions", func(t *testing.T) {
		// Create type and update user
		provider := testutils.GetUserProvider(t)
		ctx := context.Background()

		// Create type
		typeData := map[string]interface{}{
			"type_id":     fmt.Sprintf("test_type_all_%d", time.Now().UnixNano()),
			"name":        "Complete Test Type",
			"locale":      "en-US",
			"description": "Type for complete testing",
			"is_active":   true,
			"created_at":  time.Now(),
			"updated_at":  time.Now(),
		}

		typeID, err := provider.CreateType(ctx, typeData)
		assert.NoError(t, err)

		// Update user with type
		err = provider.UpdateUser(ctx, tokenInfo.UserID, map[string]interface{}{
			"type_id": typeID,
		})
		assert.NoError(t, err)

		// Get profile with all options (but without team context)
		req, err := http.NewRequest("GET", serverURL+baseURL+"/user/profile?team=true&member=true&type=true", nil)
		assert.NoError(t, err)
		req.Header.Set("Authorization", "Bearer "+tokenInfo.AccessToken)

		resp, err := http.DefaultClient.Do(req)
		assert.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var result map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&result)
		assert.NoError(t, err)

		// Verify user and type fields are present
		assert.NotEmpty(t, result["yao:user_id"], "user_id should be present")
		assert.NotEmpty(t, result["yao:type_id"], "type_id should be present")
		assert.NotNil(t, result["yao:type"], "type info should be present")

		// Without team context, team and member should not be present
		assert.Nil(t, result["yao:team"], "team info should not be present without team context")
		assert.Nil(t, result["member"], "member info should not be present without team context")

		t.Logf("Profile with type info retrieved successfully (without team context)")

		// Cleanup
		err = provider.DeleteType(ctx, typeID)
		assert.NoError(t, err)
	})

	// Test 6: Get profile with REAL team context (create team, member, and properly signed token)
	t.Run("GetProfileWithRealTeamContext", func(t *testing.T) {
		provider := testutils.GetUserProvider(t)
		ctx := context.Background()

		// Step 1: Create a team type first
		teamTypeData := map[string]interface{}{
			"type_id":     fmt.Sprintf("team_type_%d", time.Now().UnixNano()),
			"name":        "Pro Team Type",
			"locale":      "en-US",
			"description": "Professional team type",
			"is_active":   true,
			"created_at":  time.Now(),
			"updated_at":  time.Now(),
		}
		teamTypeID, err := provider.CreateType(ctx, teamTypeData)
		assert.NoError(t, err)
		t.Logf("Created team type: %s", teamTypeID)

		// Step 2: Create a team with role_id (required for ACL)
		teamData := map[string]interface{}{
			"team_id":     fmt.Sprintf("test_team_real_%d", time.Now().UnixNano()),
			"name":        "Real Test Team",
			"description": "Team with proper context",
			"logo":        "https://example.com/logo.png",
			"owner_id":    tokenInfo.UserID,
			"type_id":     teamTypeID,
			"role_id":     "system:root", // Required for ACL verification
			"status":      "active",
			"is_verified": true,
			"created_at":  time.Now(),
			"updated_at":  time.Now(),
		}

		teamID, err := provider.CreateTeam(ctx, teamData)
		assert.NoError(t, err)
		assert.NotEmpty(t, teamID)
		t.Logf("Created test team: %s", teamID)

		// Step 3: Add user as team member with system:root role (to bypass ACL)
		memberData := map[string]interface{}{
			"team_id":     teamID,
			"user_id":     tokenInfo.UserID,
			"member_type": "user",
			"role_id":     "system:root", // Use system:root for testing to bypass ACL
			"is_owner":    true,
			"status":      "active",
			"joined_at":   time.Now(),
			"created_at":  time.Now(),
			"updated_at":  time.Now(),
		}

		memberID, err := provider.CreateMember(ctx, memberData)
		assert.NoError(t, err)
		assert.NotEmpty(t, memberID)
		t.Logf("Added user as team member with system:root role: %s", memberID)

		// Step 4: Get team details for token creation
		team, err := provider.GetTeamByMember(ctx, teamID, tokenInfo.UserID)
		assert.NoError(t, err)
		assert.NotNil(t, team)

		// Step 5: Create a properly signed token with team context (like issueTokens does)
		oauthService := oauth.OAuth
		assert.NotNil(t, oauthService, "OAuth service should be initialized")

		// Get or create subject
		subject, err := oauthService.Subject(client.ClientID, tokenInfo.UserID)
		assert.NoError(t, err)

		// Prepare extra claims with team context (matching login.go issueTokens)
		extraClaims := map[string]interface{}{
			"user_id": tokenInfo.UserID, // Add user_id to claims
			"team_id": teamID,
		}

		// Add tenant_id if available from team
		if tenantID, ok := team["tenant_id"].(string); ok && tenantID != "" {
			extraClaims["tenant_id"] = tenantID
		}

		// Add owner_id
		if ownerID, ok := team["owner_id"].(string); ok && ownerID != "" {
			extraClaims["owner_id"] = ownerID
		}

		// Add type_id from team
		if typeID, ok := team["type_id"].(string); ok && typeID != "" {
			extraClaims["type_id"] = typeID
		}

		// Create access token with team context
		accessToken, err := oauthService.MakeAccessToken(
			client.ClientID,
			"openid profile email system:root",
			subject,
			3600,
			extraClaims,
		)
		assert.NoError(t, err)
		assert.NotEmpty(t, accessToken)
		t.Logf("Created token with team context: team_id=%s", teamID)

		// Step 6: Request profile with team=true, member=true, type=true
		req, err := http.NewRequest("GET", serverURL+baseURL+"/user/profile?team=true&member=true&type=true", nil)
		assert.NoError(t, err)
		req.Header.Set("Authorization", "Bearer "+accessToken)

		resp, err := http.DefaultClient.Do(req)
		assert.NoError(t, err)
		defer resp.Body.Close()

		bodyBytes, _ := io.ReadAll(resp.Body)
		t.Logf("Response status: %d, body: %s", resp.StatusCode, string(bodyBytes))

		assert.Equal(t, http.StatusOK, resp.StatusCode, "Response body: %s", string(bodyBytes))

		var result map[string]interface{}
		err = json.Unmarshal(bodyBytes, &result)
		assert.NoError(t, err)

		// Step 7: Verify all fields are present
		assert.NotEmpty(t, result["yao:user_id"], "user_id should be present")
		assert.NotEmpty(t, result["yao:team_id"], "team_id should be present")
		assert.Equal(t, teamID, result["yao:team_id"], "team_id should match")

		// Verify team info
		assert.NotNil(t, result["yao:team"], "team info should be present")
		if teamInfo, ok := result["yao:team"].(map[string]interface{}); ok {
			assert.Equal(t, teamID, teamInfo["team_id"], "team.team_id should match")
			assert.Equal(t, "Real Test Team", teamInfo["name"], "team.name should match")
			assert.Equal(t, "Team with proper context", teamInfo["description"], "team.description should match")
		}

		// Verify member info
		assert.NotNil(t, result["member"], "member info should be present")
		if member, ok := result["member"].(map[string]interface{}); ok {
			assert.Equal(t, teamID, member["team_id"], "member.team_id should match")
			assert.Equal(t, tokenInfo.UserID, member["user_id"], "member.user_id should match")
			assert.Equal(t, "system:root", member["role_id"], "member.role_id should match")
			assert.Equal(t, "active", member["status"], "member.status should match")
		}

		// Verify type info (should use team's type)
		assert.NotEmpty(t, result["yao:type_id"], "type_id should be present")
		assert.Equal(t, teamTypeID, result["yao:type_id"], "type_id should match team type")
		assert.NotNil(t, result["yao:type"], "type info should be present")
		if typeInfo, ok := result["yao:type"].(map[string]interface{}); ok {
			assert.Equal(t, teamTypeID, typeInfo["type_id"], "type.type_id should match")
			assert.Equal(t, "Pro Team Type", typeInfo["name"], "type.name should match")
		}

		// Verify is_owner flag
		if isOwner, ok := result["yao:is_owner"].(bool); ok {
			assert.True(t, isOwner, "user should be team owner")
		}

		t.Logf("✅ Profile with REAL team context retrieved successfully")
		t.Logf("   - Team: %s", result["yao:team_id"])
		t.Logf("   - Member role: %s", result["member"].(map[string]interface{})["role_id"])
		t.Logf("   - Type: %s", result["yao:type_id"])

		// Cleanup
		err = provider.DeleteTeam(ctx, teamID)
		assert.NoError(t, err)
		err = provider.DeleteType(ctx, teamTypeID)
		assert.NoError(t, err)
	})

	// Test 6: Unauthorized access
	t.Run("UnauthorizedAccess", func(t *testing.T) {
		req, err := http.NewRequest("GET", serverURL+baseURL+"/user/profile", nil)
		assert.NoError(t, err)

		// No authorization header
		resp, err := http.DefaultClient.Do(req)
		assert.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)

		var result map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&result)
		assert.NoError(t, err)

		assert.NotEmpty(t, result["error"], "error field should be present")
		t.Logf("Unauthorized access correctly rejected")
	})

	// Test 7: Invalid token
	t.Run("InvalidToken", func(t *testing.T) {
		req, err := http.NewRequest("GET", serverURL+baseURL+"/user/profile", nil)
		assert.NoError(t, err)

		// Invalid token
		req.Header.Set("Authorization", "Bearer invalid_token_12345")

		resp, err := http.DefaultClient.Do(req)
		assert.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)

		var result map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&result)
		assert.NoError(t, err)

		assert.NotEmpty(t, result["error"], "error field should be present")
		t.Logf("Invalid token correctly rejected")
	})
}

// TestProfileUpdate tests the PUT /user/profile endpoint
func TestProfileUpdate(t *testing.T) {
	serverURL := testutils.Prepare(t)
	defer testutils.Clean()

	// Get base URL from server config
	baseURL := ""
	if openapi.Server != nil && openapi.Server.Config != nil {
		baseURL = openapi.Server.Config.BaseURL
	}

	// Register a test client
	client := testutils.RegisterTestClient(t, "Profile Update Test Client", []string{"https://localhost/callback"})
	defer testutils.CleanupTestClient(t, client.ClientID)

	// Create a test user with root permissions
	tokenInfo := testutils.ObtainAccessTokenWithRootPermission(t, serverURL, client.ClientID, client.ClientSecret, "https://localhost/callback", "openid profile email")

	// Test 1: Update basic profile fields
	t.Run("UpdateBasicProfile", func(t *testing.T) {
		updateData := map[string]interface{}{
			"name":        "Updated Name",
			"given_name":  "Updated",
			"family_name": "Name",
			"nickname":    "UpdatedNick",
			"gender":      "male",
			"birthdate":   "1990-05-15",
			"locale":      "zh-CN",
			"zoneinfo":    "Asia/Shanghai",
		}

		body, err := json.Marshal(updateData)
		assert.NoError(t, err)

		req, err := http.NewRequest("PUT", serverURL+baseURL+"/user/profile", strings.NewReader(string(body)))
		assert.NoError(t, err)

		req.Header.Set("Authorization", "Bearer "+tokenInfo.AccessToken)
		req.Header.Set("Content-Type", "application/json")

		resp, err := http.DefaultClient.Do(req)
		assert.NoError(t, err)
		defer resp.Body.Close()

		bodyBytes, _ := io.ReadAll(resp.Body)
		t.Logf("Response status: %d, body: %s", resp.StatusCode, string(bodyBytes))

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var result map[string]interface{}
		err = json.Unmarshal(bodyBytes, &result)
		assert.NoError(t, err)

		// Verify response contains user_id and message
		assert.Equal(t, tokenInfo.UserID, result["user_id"], "user_id should match")
		assert.Equal(t, "Profile updated successfully", result["message"], "message should be present")

		t.Logf("✅ Profile updated successfully")

		// Verify the update by getting the profile
		getReq, err := http.NewRequest("GET", serverURL+baseURL+"/user/profile", nil)
		assert.NoError(t, err)
		getReq.Header.Set("Authorization", "Bearer "+tokenInfo.AccessToken)

		getResp, err := http.DefaultClient.Do(getReq)
		assert.NoError(t, err)
		defer getResp.Body.Close()

		var profile map[string]interface{}
		err = json.NewDecoder(getResp.Body).Decode(&profile)
		assert.NoError(t, err)

		// Verify updated fields
		assert.Equal(t, "Updated Name", profile["name"])
		assert.Equal(t, "Updated", profile["given_name"])
		assert.Equal(t, "Name", profile["family_name"])
		assert.Equal(t, "UpdatedNick", profile["nickname"])
		assert.Equal(t, "male", profile["gender"])
		assert.Equal(t, "1990-05-15", profile["birthdate"])
		assert.Equal(t, "zh-CN", profile["locale"])
		assert.Equal(t, "Asia/Shanghai", profile["zoneinfo"])

		t.Logf("✅ Profile fields verified after update")
	})

	// Test 2: Update profile with picture and website
	t.Run("UpdateProfileWithLinks", func(t *testing.T) {
		updateData := map[string]interface{}{
			"picture": "https://example.com/avatar-new.jpg",
			"website": "https://mynewsite.com",
			"profile": "https://mynewsite.com/profile",
		}

		body, err := json.Marshal(updateData)
		assert.NoError(t, err)

		req, err := http.NewRequest("PUT", serverURL+baseURL+"/user/profile", strings.NewReader(string(body)))
		assert.NoError(t, err)
		req.Header.Set("Authorization", "Bearer "+tokenInfo.AccessToken)
		req.Header.Set("Content-Type", "application/json")

		resp, err := http.DefaultClient.Do(req)
		assert.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var result map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&result)
		assert.NoError(t, err)
		assert.Equal(t, tokenInfo.UserID, result["user_id"])
		assert.Equal(t, "Profile updated successfully", result["message"])

		t.Logf("✅ Profile links updated successfully")
	})

	// Test 3: Update with address and metadata
	t.Run("UpdateWithAddressAndMetadata", func(t *testing.T) {
		updateData := map[string]interface{}{
			"address": map[string]interface{}{
				"formatted":      "北京市朝阳区xxx街道",
				"street_address": "xxx街道123号",
				"locality":       "北京",
				"region":         "北京市",
				"postal_code":    "100000",
				"country":        "中国",
			},
			"metadata": map[string]interface{}{
				"bio":     "全栈开发工程师",
				"company": "示例科技公司",
				"skills":  []string{"Go", "React", "TypeScript"},
			},
		}

		body, err := json.Marshal(updateData)
		assert.NoError(t, err)

		req, err := http.NewRequest("PUT", serverURL+baseURL+"/user/profile", strings.NewReader(string(body)))
		assert.NoError(t, err)
		req.Header.Set("Authorization", "Bearer "+tokenInfo.AccessToken)
		req.Header.Set("Content-Type", "application/json")

		resp, err := http.DefaultClient.Do(req)
		assert.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var result map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&result)
		assert.NoError(t, err)
		assert.Equal(t, tokenInfo.UserID, result["user_id"])
		assert.Equal(t, "Profile updated successfully", result["message"])

		t.Logf("✅ Address and metadata updated successfully")

		// Verify the update
		getReq, err := http.NewRequest("GET", serverURL+baseURL+"/user/profile", nil)
		assert.NoError(t, err)
		getReq.Header.Set("Authorization", "Bearer "+tokenInfo.AccessToken)

		getResp, err := http.DefaultClient.Do(getReq)
		assert.NoError(t, err)
		defer getResp.Body.Close()

		var profile map[string]interface{}
		err = json.NewDecoder(getResp.Body).Decode(&profile)
		assert.NoError(t, err)

		// Verify address
		if address, ok := profile["address"].(map[string]interface{}); ok {
			assert.Equal(t, "北京市朝阳区xxx街道", address["formatted"])
			assert.Equal(t, "中国", address["country"])
		}

		// Verify metadata
		if metadata, ok := profile["yao:metadata"].(map[string]interface{}); ok {
			assert.Equal(t, "全栈开发工程师", metadata["bio"])
			assert.Equal(t, "示例科技公司", metadata["company"])
		}

		t.Logf("✅ Address and metadata verified")
	})

	// Test 4: Update theme preference
	t.Run("UpdateTheme", func(t *testing.T) {
		updateData := map[string]interface{}{
			"theme": "dark",
		}

		body, err := json.Marshal(updateData)
		assert.NoError(t, err)

		req, err := http.NewRequest("PUT", serverURL+baseURL+"/user/profile", strings.NewReader(string(body)))
		assert.NoError(t, err)
		req.Header.Set("Authorization", "Bearer "+tokenInfo.AccessToken)
		req.Header.Set("Content-Type", "application/json")

		resp, err := http.DefaultClient.Do(req)
		assert.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var result map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&result)
		assert.NoError(t, err)
		assert.Equal(t, tokenInfo.UserID, result["user_id"])
		assert.Equal(t, "Profile updated successfully", result["message"])

		t.Logf("✅ Theme preference updated successfully")
	})

	// Test 5: Empty update should fail
	t.Run("EmptyUpdateShouldFail", func(t *testing.T) {
		updateData := map[string]interface{}{}

		body, err := json.Marshal(updateData)
		assert.NoError(t, err)

		req, err := http.NewRequest("PUT", serverURL+baseURL+"/user/profile", strings.NewReader(string(body)))
		assert.NoError(t, err)
		req.Header.Set("Authorization", "Bearer "+tokenInfo.AccessToken)
		req.Header.Set("Content-Type", "application/json")

		resp, err := http.DefaultClient.Do(req)
		assert.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusInternalServerError, resp.StatusCode)

		var result map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&result)
		assert.NoError(t, err)
		assert.NotEmpty(t, result["error"], "error should be present for empty update")

		t.Logf("✅ Empty update correctly rejected")
	})

	// Test 6: Invalid JSON should fail
	t.Run("InvalidJSONShouldFail", func(t *testing.T) {
		req, err := http.NewRequest("PUT", serverURL+baseURL+"/user/profile", strings.NewReader("{invalid json}"))
		assert.NoError(t, err)
		req.Header.Set("Authorization", "Bearer "+tokenInfo.AccessToken)
		req.Header.Set("Content-Type", "application/json")

		resp, err := http.DefaultClient.Do(req)
		assert.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)

		var result map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&result)
		assert.NoError(t, err)
		assert.NotEmpty(t, result["error"], "error should be present for invalid JSON")

		t.Logf("✅ Invalid JSON correctly rejected")
	})

	// Test 7: Unauthorized access
	t.Run("UnauthorizedUpdate", func(t *testing.T) {
		updateData := map[string]interface{}{
			"name": "Unauthorized Update",
		}

		body, err := json.Marshal(updateData)
		assert.NoError(t, err)

		req, err := http.NewRequest("PUT", serverURL+baseURL+"/user/profile", strings.NewReader(string(body)))
		assert.NoError(t, err)
		req.Header.Set("Content-Type", "application/json")
		// No authorization header

		resp, err := http.DefaultClient.Do(req)
		assert.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)

		var result map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&result)
		assert.NoError(t, err)
		assert.NotEmpty(t, result["error"], "error should be present")

		t.Logf("✅ Unauthorized update correctly rejected")
	})

	// Test 8: Invalid token
	t.Run("InvalidTokenUpdate", func(t *testing.T) {
		updateData := map[string]interface{}{
			"name": "Invalid Token Update",
		}

		body, err := json.Marshal(updateData)
		assert.NoError(t, err)

		req, err := http.NewRequest("PUT", serverURL+baseURL+"/user/profile", strings.NewReader(string(body)))
		assert.NoError(t, err)
		req.Header.Set("Authorization", "Bearer invalid_token_xyz")
		req.Header.Set("Content-Type", "application/json")

		resp, err := http.DefaultClient.Do(req)
		assert.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)

		var result map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&result)
		assert.NoError(t, err)
		assert.NotEmpty(t, result["error"], "error should be present")

		t.Logf("✅ Invalid token update correctly rejected")
	})

	// Test 9: Partial update (only one field)
	t.Run("PartialUpdate", func(t *testing.T) {
		updateData := map[string]interface{}{
			"nickname": "PartialNick",
		}

		body, err := json.Marshal(updateData)
		assert.NoError(t, err)

		req, err := http.NewRequest("PUT", serverURL+baseURL+"/user/profile", strings.NewReader(string(body)))
		assert.NoError(t, err)
		req.Header.Set("Authorization", "Bearer "+tokenInfo.AccessToken)
		req.Header.Set("Content-Type", "application/json")

		resp, err := http.DefaultClient.Do(req)
		assert.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var result map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&result)
		assert.NoError(t, err)
		assert.Equal(t, tokenInfo.UserID, result["user_id"])
		assert.Equal(t, "Profile updated successfully", result["message"])

		t.Logf("✅ Partial update (single field) successful")

		// Verify only nickname was updated
		getReq, err := http.NewRequest("GET", serverURL+baseURL+"/user/profile", nil)
		assert.NoError(t, err)
		getReq.Header.Set("Authorization", "Bearer "+tokenInfo.AccessToken)

		getResp, err := http.DefaultClient.Do(getReq)
		assert.NoError(t, err)
		defer getResp.Body.Close()

		var profile map[string]interface{}
		err = json.NewDecoder(getResp.Body).Decode(&profile)
		assert.NoError(t, err)

		assert.Equal(t, "PartialNick", profile["nickname"])

		t.Logf("✅ Partial update verified")
	})
}
