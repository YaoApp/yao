package user_test

import (
	"encoding/json"
	"io"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/yaoapp/yao/openapi"
	"github.com/yaoapp/yao/openapi/tests/testutils"
	"github.com/yaoapp/yao/openapi/user"
)

// TestTeamConfigRobotLoad tests loading team configuration with robot field
func TestTeamConfigRobotLoad(t *testing.T) {
	serverURL := testutils.Prepare(t)
	defer testutils.Clean()

	_ = serverURL // Server URL not needed for this test

	// Get team config with robot configuration
	teamConfig := user.GetTeamConfig("en")
	assert.NotNil(t, teamConfig, "Team config should not be nil")

	if teamConfig != nil && teamConfig.Robot != nil {
		t.Logf("Robot config loaded successfully")

		// Test robot roles
		assert.NotNil(t, teamConfig.Robot.Roles, "Robot roles should not be nil")
		if teamConfig.Robot.Roles != nil {
			t.Logf("Robot roles: %v", teamConfig.Robot.Roles)
			assert.Greater(t, len(teamConfig.Robot.Roles), 0, "Robot should have at least one role")
		}

		// Test robot agents
		assert.NotNil(t, teamConfig.Robot.Agents, "Robot agents should not be nil")
		if teamConfig.Robot.Agents != nil {
			t.Logf("Robot agents - Executor: %s, Planner: %s, Profiler: %s",
				teamConfig.Robot.Agents.Executor,
				teamConfig.Robot.Agents.Planner,
				teamConfig.Robot.Agents.Profiler)
			assert.NotEmpty(t, teamConfig.Robot.Agents.Executor, "Executor agent should not be empty")
			assert.NotEmpty(t, teamConfig.Robot.Agents.Planner, "Planner agent should not be empty")
			assert.NotEmpty(t, teamConfig.Robot.Agents.Profiler, "Profiler agent should not be empty")
		}

		// Test robot email domains
		assert.NotNil(t, teamConfig.Robot.EmailDomains, "Robot email domains should not be nil")
		if teamConfig.Robot.EmailDomains != nil {
			t.Logf("Robot has %d email domain(s)", len(teamConfig.Robot.EmailDomains))
			assert.Greater(t, len(teamConfig.Robot.EmailDomains), 0, "Robot should have at least one email domain")

			for i, domain := range teamConfig.Robot.EmailDomains {
				t.Logf("Email domain %d: %s (%s)", i, domain.Name, domain.Domain)
				assert.NotEmpty(t, domain.Name, "Email domain name should not be empty")
				assert.NotEmpty(t, domain.Domain, "Email domain should not be empty")
				assert.NotEmpty(t, domain.Messenger, "Email messenger should not be empty")
				assert.Greater(t, domain.PrefixMinLength, 0, "PrefixMinLength should be greater than 0")
				assert.Greater(t, domain.PrefixMaxLength, domain.PrefixMinLength, "PrefixMaxLength should be greater than PrefixMinLength")

				// Test whitelist
				assert.NotNil(t, domain.Whitelist, "Whitelist should not be nil")
				if domain.Whitelist != nil {
					t.Logf("  Whitelist - Domains: %v, Senders: %v, IPs: %v",
						domain.Whitelist.Domains,
						domain.Whitelist.Senders,
						domain.Whitelist.IPs)
				}
			}
		}

		// Test robot defaults
		assert.NotNil(t, teamConfig.Robot.Defaults, "Robot defaults should not be nil")
		if teamConfig.Robot.Defaults != nil {
			t.Logf("Robot defaults - LLM: %s, AutonomousMode: %v, CostLimit: %d",
				teamConfig.Robot.Defaults.LLM,
				teamConfig.Robot.Defaults.AutonomousMode,
				teamConfig.Robot.Defaults.CostLimit)
			assert.NotEmpty(t, teamConfig.Robot.Defaults.LLM, "Default LLM should not be empty")
		}
	} else {
		t.Log("No robot configuration found in team config")
	}
}

// TestGetTeamConfigPublic tests that GetTeamConfigPublic hides sensitive fields
func TestGetTeamConfigPublic(t *testing.T) {
	serverURL := testutils.Prepare(t)
	defer testutils.Clean()

	_ = serverURL // Server URL not needed for this test

	// Get original config
	originalConfig := user.GetTeamConfig("en")
	assert.NotNil(t, originalConfig, "Original config should not be nil")

	// Get public config
	publicConfig := user.GetTeamConfigPublic("en")
	assert.NotNil(t, publicConfig, "Public config should not be nil")

	// Test that basic fields are preserved
	assert.Equal(t, originalConfig.Type, publicConfig.Type, "Type should be preserved")
	assert.Equal(t, originalConfig.Role, publicConfig.Role, "Role should be preserved")
	assert.Equal(t, originalConfig.Roles, publicConfig.Roles, "Roles should be preserved")
	assert.Equal(t, originalConfig.Invite, publicConfig.Invite, "Invite config should be preserved")

	// Test robot configuration
	if originalConfig.Robot != nil {
		t.Log("Testing robot config sanitization")

		assert.NotNil(t, publicConfig.Robot, "Public config should have robot config")

		// Test that roles are preserved
		assert.Equal(t, originalConfig.Robot.Roles, publicConfig.Robot.Roles, "Robot roles should be preserved")

		// Test that agents are hidden (SENSITIVE)
		assert.Nil(t, publicConfig.Robot.Agents, "Robot agents should be hidden in public config")
		if originalConfig.Robot.Agents != nil {
			t.Logf("Original agents (hidden in public): Executor=%s, Planner=%s, Profiler=%s",
				originalConfig.Robot.Agents.Executor,
				originalConfig.Robot.Agents.Planner,
				originalConfig.Robot.Agents.Profiler)
		}

		// Test that defaults are preserved
		assert.Equal(t, originalConfig.Robot.Defaults, publicConfig.Robot.Defaults, "Robot defaults should be preserved")

		// Test email domains
		if originalConfig.Robot.EmailDomains != nil {
			assert.NotNil(t, publicConfig.Robot.EmailDomains, "Public config should have email domains")
			assert.Equal(t, len(originalConfig.Robot.EmailDomains), len(publicConfig.Robot.EmailDomains),
				"Email domains count should match")

			for i := 0; i < len(originalConfig.Robot.EmailDomains); i++ {
				origDomain := originalConfig.Robot.EmailDomains[i]
				pubDomain := publicConfig.Robot.EmailDomains[i]

				// Test that basic fields are preserved
				assert.Equal(t, origDomain.Name, pubDomain.Name, "Domain name should be preserved")
				assert.Equal(t, origDomain.Domain, pubDomain.Domain, "Domain should be preserved")
				assert.Equal(t, origDomain.Messenger, pubDomain.Messenger, "Messenger should be preserved")
				assert.Equal(t, origDomain.PrefixMinLength, pubDomain.PrefixMinLength, "PrefixMinLength should be preserved")
				assert.Equal(t, origDomain.PrefixMaxLength, pubDomain.PrefixMaxLength, "PrefixMaxLength should be preserved")
				assert.Equal(t, origDomain.ReservedWords, pubDomain.ReservedWords, "ReservedWords should be preserved")

				// Test that whitelist is hidden (SENSITIVE)
				assert.Nil(t, pubDomain.Whitelist, "Whitelist should be hidden in public config")
				if origDomain.Whitelist != nil {
					t.Logf("Domain %s whitelist (hidden in public): Domains=%v, Senders=%v, IPs=%v",
						origDomain.Name,
						origDomain.Whitelist.Domains,
						origDomain.Whitelist.Senders,
						origDomain.Whitelist.IPs)
				}
			}
		}
	}
}

// TestGetTeamConfigPublicNoMutation tests that GetTeamConfigPublic doesn't mutate original data
func TestGetTeamConfigPublicNoMutation(t *testing.T) {
	serverURL := testutils.Prepare(t)
	defer testutils.Clean()

	_ = serverURL // Server URL not needed for this test

	// Get original config
	originalConfig := user.GetTeamConfig("en")
	if originalConfig == nil || originalConfig.Robot == nil {
		t.Skip("No robot config available for this test")
	}

	// Store original values for comparison
	var originalAgentsPresent bool
	var originalAgents *user.RobotAgents
	if originalConfig.Robot.Agents != nil {
		originalAgentsPresent = true
		originalAgents = &user.RobotAgents{
			Executor: originalConfig.Robot.Agents.Executor,
			Planner:  originalConfig.Robot.Agents.Planner,
			Profiler: originalConfig.Robot.Agents.Profiler,
		}
	}

	var originalWhitelists []*user.EmailDomainWhitelist
	if originalConfig.Robot.EmailDomains != nil {
		for _, domain := range originalConfig.Robot.EmailDomains {
			if domain.Whitelist != nil {
				originalWhitelists = append(originalWhitelists, &user.EmailDomainWhitelist{
					Domains: domain.Whitelist.Domains,
					Senders: domain.Whitelist.Senders,
					IPs:     domain.Whitelist.IPs,
				})
			}
		}
	}

	// Get public config (should create a copy, not mutate original)
	publicConfig := user.GetTeamConfigPublic("en")
	assert.NotNil(t, publicConfig, "Public config should not be nil")

	// Verify original config is unchanged
	originalConfigAfter := user.GetTeamConfig("en")
	assert.NotNil(t, originalConfigAfter, "Original config should still exist")

	if originalAgentsPresent {
		assert.NotNil(t, originalConfigAfter.Robot.Agents, "Original agents should still be present")
		assert.Equal(t, originalAgents.Executor, originalConfigAfter.Robot.Agents.Executor, "Original executor should be unchanged")
		assert.Equal(t, originalAgents.Planner, originalConfigAfter.Robot.Agents.Planner, "Original planner should be unchanged")
		assert.Equal(t, originalAgents.Profiler, originalConfigAfter.Robot.Agents.Profiler, "Original profiler should be unchanged")
	}

	if len(originalWhitelists) > 0 {
		for i, domain := range originalConfigAfter.Robot.EmailDomains {
			if i < len(originalWhitelists) {
				assert.NotNil(t, domain.Whitelist, "Original whitelist should still be present")
				assert.Equal(t, originalWhitelists[i].Domains, domain.Whitelist.Domains, "Original whitelist domains should be unchanged")
				assert.Equal(t, originalWhitelists[i].Senders, domain.Whitelist.Senders, "Original whitelist senders should be unchanged")
				assert.Equal(t, originalWhitelists[i].IPs, domain.Whitelist.IPs, "Original whitelist IPs should be unchanged")
			}
		}
	}

	t.Log("Original config remains intact after calling GetTeamConfigPublic")
}

// TestTeamConfigAPIPublic tests that the API endpoint returns public config
func TestTeamConfigAPIPublic(t *testing.T) {
	serverURL := testutils.Prepare(t)
	defer testutils.Clean()

	// Get base URL from server config
	baseURL := ""
	if openapi.Server != nil && openapi.Server.Config != nil {
		baseURL = openapi.Server.Config.BaseURL
	}

	// Register a test client and get access token
	testClient := testutils.RegisterTestClient(t, "Robot Config Test Client", []string{"https://localhost/callback"})
	defer testutils.CleanupTestClient(t, testClient.ClientID)

	// Obtain access token for authentication
	tokenInfo := testutils.ObtainAccessToken(t, serverURL, testClient.ClientID, testClient.ClientSecret, "https://localhost/callback", "openid profile")

	// Test API endpoint
	requestURL := serverURL + baseURL + "/user/teams/config?locale=en"

	// Create request with Authorization header
	req, err := http.NewRequest("GET", requestURL, nil)
	assert.NoError(t, err, "Should create HTTP request")
	req.Header.Set("Authorization", "Bearer "+tokenInfo.AccessToken)

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	assert.NoError(t, err, "HTTP request should succeed")
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode, "Expected status code 200")

	// Parse response body
	body, err := io.ReadAll(resp.Body)
	assert.NoError(t, err, "Should read response body")

	var teamConfig user.TeamConfig
	err = json.Unmarshal(body, &teamConfig)
	assert.NoError(t, err, "Should parse JSON response")

	t.Logf("API response has %d roles", len(teamConfig.Roles))

	// Test that robot config is present (if available in config files)
	if teamConfig.Robot != nil {
		t.Log("Robot config present in API response")

		// Test that public fields are present
		if teamConfig.Robot.Roles != nil {
			t.Logf("Robot roles: %v", teamConfig.Robot.Roles)
			assert.Greater(t, len(teamConfig.Robot.Roles), 0, "Robot should have at least one role")
		}

		if teamConfig.Robot.Defaults != nil {
			t.Logf("Robot defaults - LLM: %s, AutonomousMode: %v, CostLimit: %d",
				teamConfig.Robot.Defaults.LLM,
				teamConfig.Robot.Defaults.AutonomousMode,
				teamConfig.Robot.Defaults.CostLimit)
		}

		// Test that sensitive fields are hidden
		assert.Nil(t, teamConfig.Robot.Agents, "Robot agents should be hidden in API response (SENSITIVE)")

		if teamConfig.Robot.EmailDomains != nil {
			t.Logf("Robot has %d email domain(s)", len(teamConfig.Robot.EmailDomains))
			for i, domain := range teamConfig.Robot.EmailDomains {
				t.Logf("Email domain %d: %s (%s)", i, domain.Name, domain.Domain)
				assert.NotEmpty(t, domain.Name, "Email domain name should be present")
				assert.NotEmpty(t, domain.Domain, "Email domain should be present")

				// Test that whitelist is hidden
				assert.Nil(t, domain.Whitelist, "Whitelist should be hidden in API response (SENSITIVE)")
			}
		}
	} else {
		t.Log("No robot configuration in API response")
	}
}

// TestTeamConfigAPILocales tests that API returns correct locale-specific robot config
func TestTeamConfigAPILocales(t *testing.T) {
	serverURL := testutils.Prepare(t)
	defer testutils.Clean()

	// Get base URL from server config
	baseURL := ""
	if openapi.Server != nil && openapi.Server.Config != nil {
		baseURL = openapi.Server.Config.BaseURL
	}

	// Register a test client and get access token
	testClient := testutils.RegisterTestClient(t, "Robot Config Locale Test Client", []string{"https://localhost/callback"})
	defer testutils.CleanupTestClient(t, testClient.ClientID)

	// Obtain access token for authentication
	tokenInfo := testutils.ObtainAccessToken(t, serverURL, testClient.ClientID, testClient.ClientSecret, "https://localhost/callback", "openid profile")

	// Test different locales
	locales := []string{"en", "zh-cn", ""}

	client := &http.Client{Timeout: 10 * time.Second}

	for _, locale := range locales {
		t.Run("locale_"+locale, func(t *testing.T) {
			requestURL := serverURL + baseURL + "/user/teams/config"
			if locale != "" {
				requestURL += "?locale=" + locale
			}

			req, err := http.NewRequest("GET", requestURL, nil)
			assert.NoError(t, err, "Should create HTTP request")
			req.Header.Set("Authorization", "Bearer "+tokenInfo.AccessToken)

			resp, err := client.Do(req)
			assert.NoError(t, err, "HTTP request should succeed")
			defer resp.Body.Close()

			assert.Equal(t, http.StatusOK, resp.StatusCode, "Expected status code 200")

			body, err := io.ReadAll(resp.Body)
			assert.NoError(t, err, "Should read response body")

			var teamConfig user.TeamConfig
			err = json.Unmarshal(body, &teamConfig)
			assert.NoError(t, err, "Should parse JSON response")

			t.Logf("Locale '%s': %d roles", locale, len(teamConfig.Roles))

			// Verify sensitive fields are hidden
			if teamConfig.Robot != nil {
				assert.Nil(t, teamConfig.Robot.Agents, "Agents should be hidden for locale: "+locale)
				if teamConfig.Robot.EmailDomains != nil {
					for _, domain := range teamConfig.Robot.EmailDomains {
						assert.Nil(t, domain.Whitelist, "Whitelist should be hidden for locale: "+locale)
					}
				}
			}
		})
	}
}
