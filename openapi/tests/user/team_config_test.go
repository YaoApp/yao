package user_test

import (
	"encoding/json"
	"io"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/yaoapp/yao/config"
	"github.com/yaoapp/yao/openapi"
	"github.com/yaoapp/yao/openapi/tests/testutils"
	"github.com/yaoapp/yao/openapi/user"
)

func TestTeamConfigLoad(t *testing.T) {
	// Initialize test environment
	serverURL := testutils.Prepare(t)
	defer testutils.Clean()

	_ = serverURL // Server URL not needed for this test

	// Test loading team configurations
	err := user.Load(config.Conf)
	assert.NoError(t, err, "user.Load should succeed")

	// Test that we can get team config
	teamConfig := user.GetTeamConfig("")
	if teamConfig != nil {
		t.Logf("Team config loaded with %d roles", len(teamConfig.Roles))
		assert.IsType(t, &user.TeamConfig{}, teamConfig, "Should return correct team config type")
	} else {
		t.Log("No team config found")
	}
}

func TestTeamConfigStructure(t *testing.T) {
	// Initialize test environment
	serverURL := testutils.Prepare(t)
	defer testutils.Clean()

	_ = serverURL // Server URL not needed for this test

	// Note: user.Load is automatically called by openapi.Load in testutils.Prepare

	// Get a team config to test structure
	teamConfig := user.GetTeamConfig("")
	if teamConfig != nil {
		t.Logf("Team config loaded successfully with %d roles", len(teamConfig.Roles))

		// Verify team config structure is valid
		assert.IsType(t, &user.TeamConfig{}, teamConfig, "Should return correct team config type")

		// Test roles configuration
		if teamConfig.Roles != nil {
			assert.IsType(t, []*user.TeamRole{}, teamConfig.Roles, "Roles should be slice of TeamRole pointers")
			for i, role := range teamConfig.Roles {
				t.Logf("Role %d: %s (%s)", i, role.RoleID, role.Label)
				assert.NotEmpty(t, role.RoleID, "Role ID should not be empty")
				assert.NotEmpty(t, role.Label, "Role label should not be empty")
				assert.NotEmpty(t, role.Description, "Role description should not be empty")
			}
		}

		// Test invite configuration
		if teamConfig.Invite != nil {
			t.Logf("Invite config found: channel=%s, expiry=%s", teamConfig.Invite.Channel, teamConfig.Invite.Expiry)
			assert.IsType(t, &user.InviteConfig{}, teamConfig.Invite, "Invite should be InviteConfig type")

			if teamConfig.Invite.Templates != nil {
				assert.IsType(t, map[string]string{}, teamConfig.Invite.Templates, "Templates should be map[string]string")
				for templateType, templateName := range teamConfig.Invite.Templates {
					t.Logf("Template %s: %s", templateType, templateName)
					assert.NotEmpty(t, templateName, "Template name should not be empty")
				}
			}
		}
	} else {
		t.Log("No team configuration found")
	}
}

func TestTeamConfigByLocale(t *testing.T) {
	// Initialize test environment
	serverURL := testutils.Prepare(t)
	defer testutils.Clean()

	_ = serverURL // Server URL not needed for this test

	// Test different locales
	locales := []string{"en", "zh-cn", "invalid", ""}

	for _, locale := range locales {
		t.Run("locale_"+locale, func(t *testing.T) {
			teamConfig := user.GetTeamConfig(locale)
			if teamConfig != nil {
				t.Logf("Team config for locale '%s' loaded with %d roles", locale, len(teamConfig.Roles))
				assert.IsType(t, &user.TeamConfig{}, teamConfig, "Should return correct team config type")
			} else {
				t.Logf("No team config found for locale '%s'", locale)
			}
		})
	}
}

func TestTeamConfigAPI(t *testing.T) {
	// Initialize test environment
	serverURL := testutils.Prepare(t)
	defer testutils.Clean()

	// Get base URL from server config
	baseURL := ""
	if openapi.Server != nil && openapi.Server.Config != nil {
		baseURL = openapi.Server.Config.BaseURL
	}

	// Register a test client and get access token (team config endpoint now requires authentication)
	testClient := testutils.RegisterTestClient(t, "Team Config Test Client", []string{"https://localhost/callback"})
	defer testutils.CleanupTestClient(t, testClient.ClientID)

	// Obtain access token for authentication
	tokenInfo := testutils.ObtainAccessToken(t, serverURL, testClient.ClientID, testClient.ClientSecret, "https://localhost/callback", "openid profile")

	// Test API endpoints for team configuration
	testCases := []struct {
		name       string
		endpoint   string
		expectCode int
	}{
		{"get team config without locale", "/user/teams/config", 200},
		{"get team config with en locale", "/user/teams/config?locale=en", 200},
		{"get team config with zh-cn locale", "/user/teams/config?locale=zh-cn", 200},
		{"get team config with invalid locale", "/user/teams/config?locale=invalid", 200}, // should fallback to default
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			requestURL := serverURL + baseURL + tc.endpoint

			// Create request with Authorization header
			req, err := http.NewRequest("GET", requestURL, nil)
			assert.NoError(t, err, "Should create HTTP request")
			req.Header.Set("Authorization", "Bearer "+tokenInfo.AccessToken)

			resp, err := http.DefaultClient.Do(req)
			assert.NoError(t, err, "HTTP request should succeed")

			if resp != nil {
				defer resp.Body.Close()
				assert.Equal(t, tc.expectCode, resp.StatusCode, "Expected status code %d", tc.expectCode)

				if resp.StatusCode == 200 {
					// Parse response body
					body, err := io.ReadAll(resp.Body)
					assert.NoError(t, err, "Should read response body")

					var teamConfig user.TeamConfig
					err = json.Unmarshal(body, &teamConfig)
					assert.NoError(t, err, "Should parse JSON response")

					t.Logf("API response for %s: %d roles", tc.endpoint, len(teamConfig.Roles))

					// Verify team config structure
					assert.IsType(t, &user.TeamConfig{}, &teamConfig, "Should return correct team config type")

					// Test roles if present
					if teamConfig.Roles != nil {
						assert.IsType(t, []*user.TeamRole{}, teamConfig.Roles, "Roles should be slice of TeamRole pointers")
						for i, role := range teamConfig.Roles {
							t.Logf("Role %d: %s (%s)", i, role.RoleID, role.Label)
							assert.NotEmpty(t, role.RoleID, "Role ID should not be empty")
							assert.NotEmpty(t, role.Label, "Role label should not be empty")
						}
					}

					// Test invite config if present
					if teamConfig.Invite != nil {
						t.Logf("Invite config: channel=%s, expiry=%s", teamConfig.Invite.Channel, teamConfig.Invite.Expiry)
						assert.IsType(t, &user.InviteConfig{}, teamConfig.Invite, "Invite should be InviteConfig type")
					}
				}
			}
		})
	}
}
