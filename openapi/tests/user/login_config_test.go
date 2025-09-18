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

func TestUserLoginConfig(t *testing.T) {
	// Initialize test environment
	serverURL := testutils.Prepare(t)
	defer testutils.Clean()

	// Get base URL from server config
	baseURL := ""
	if openapi.Server != nil && openapi.Server.Config != nil {
		baseURL = openapi.Server.Config.BaseURL
	}

	// Register a test client first (needed for user.Load validation)
	testClient := testutils.RegisterTestClient(t, "User Config Test Client", []string{"https://localhost/callback"})
	defer testutils.CleanupTestClient(t, testClient.ClientID)

	// Note: user.Load is automatically called by openapi.Load in testutils.Prepare

	// Test API endpoints
	testCases := []struct {
		name       string
		endpoint   string
		expectCode int
	}{
		{"get config without locale", "/user/login", 200},
		{"get config with en locale", "/user/login?locale=en", 200},
		{"get config with zh-cn locale", "/user/login?locale=zh-cn", 200},
		{"get config with invalid locale", "/user/login?locale=invalid", 200}, // should fallback to default
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			requestURL := serverURL + baseURL + tc.endpoint
			resp, err := http.Get(requestURL)
			assert.NoError(t, err, "HTTP request should succeed")

			if resp != nil {
				defer resp.Body.Close()
				assert.Equal(t, tc.expectCode, resp.StatusCode, "Expected status code %d", tc.expectCode)

				if resp.StatusCode == 200 {
					// Parse response body
					body, err := io.ReadAll(resp.Body)
					assert.NoError(t, err, "Should read response body")

					var config user.Config
					err = json.Unmarshal(body, &config)
					assert.NoError(t, err, "Should parse JSON response")

					t.Logf("API response for %s: %s", tc.endpoint, config.Title)

					// Verify it's public config (no sensitive data)
					if config.ThirdParty != nil && config.ThirdParty.Providers != nil {
						for _, provider := range config.ThirdParty.Providers {
							// Check that sensitive OAuth fields are removed from API response
							assert.Empty(t, provider.ClientID, "Client ID should be empty in API response")
							assert.Empty(t, provider.ClientSecret, "Client secret should be empty in API response")
							assert.Nil(t, provider.ClientSecretGenerator, "Client secret generator should be nil in API response")
							assert.Empty(t, provider.Scopes, "Scopes should be empty in API response")
							assert.Nil(t, provider.Endpoints, "Endpoints should be nil in API response")
							assert.Empty(t, provider.Mapping, "Mapping should be empty in API response")

							// Check that display fields are preserved in API response
							assert.NotEmpty(t, provider.ID, "Provider ID should be preserved in API response")
							assert.NotEmpty(t, provider.Title, "Provider title should be preserved in API response")
						}
					}

					// Verify captcha sensitive data is removed from API response
					if config.Form != nil && config.Form.Captcha != nil && config.Form.Captcha.Options != nil {
						_, hasSecret := config.Form.Captcha.Options["secret"]
						assert.False(t, hasSecret, "Captcha secret should be removed from API response")
					}
				}
			}
		})
	}
}

func TestUserLoginConfigLoad(t *testing.T) {
	// Initialize test environment
	serverURL := testutils.Prepare(t)
	defer testutils.Clean()

	_ = serverURL // Server URL not needed for this test

	// Test loading user configurations
	err := user.Load(config.Conf)
	assert.NoError(t, err, "user.Load should succeed")

	// Test that we can get public config
	publicConfig := user.GetPublicConfig("")
	if publicConfig != nil {
		t.Logf("Public config loaded with title: %s", publicConfig.Title)
		assert.IsType(t, &user.Config{}, publicConfig, "Should return correct config type")
	} else {
		t.Log("No public config found")
	}
}

func TestUserLoginConfigStructure(t *testing.T) {
	// Initialize test environment
	serverURL := testutils.Prepare(t)
	defer testutils.Clean()

	_ = serverURL // Server URL not needed for this test

	// Note: user.Load is automatically called by openapi.Load in testutils.Prepare

	// Get a config to test structure
	config := user.GetPublicConfig("")
	if config != nil {
		t.Logf("Config loaded successfully with title: %s", config.Title)

		// Verify config structure is valid
		assert.IsType(t, &user.Config{}, config, "Should return correct config type")

		// Test new configuration fields
		assert.IsType(t, "", config.ClientID, "ClientID should be string")
		assert.IsType(t, "", config.ClientSecret, "ClientSecret should be string")
		assert.IsType(t, false, config.Default, "Default should be boolean")
		t.Logf("Config has ClientID: %t, ClientSecret: %t, Default: %t",
			config.ClientID != "", config.ClientSecret != "", config.Default)

		// Test form configuration
		if config.Form != nil {
			t.Logf("Form configuration found")
			if config.Form.Username != nil {
				assert.IsType(t, []string{}, config.Form.Username.Fields, "Username fields should be string slice")
			}
			if config.Form.Captcha != nil {
				assert.IsType(t, map[string]interface{}{}, config.Form.Captcha.Options, "Captcha options should be map")
			}
		}

		// Test third party configuration
		if config.ThirdParty != nil {
			t.Logf("Third party configuration found with %d providers", len(config.ThirdParty.Providers))
			if config.ThirdParty.Providers != nil {
				assert.IsType(t, []*user.Provider{}, config.ThirdParty.Providers, "Providers should be slice of Provider pointers")
				for i, provider := range config.ThirdParty.Providers {
					t.Logf("Provider %d: %s", i, provider.ID)

					// In the new structure, ThirdParty providers only contain display information
					// Sensitive configuration data is stored separately in the global providers map
					assert.NotEmpty(t, provider.ID, "Provider ID should not be empty")
					assert.NotEmpty(t, provider.Title, "Provider title should not be empty")
				}
			}
		}
	} else {
		t.Log("No user configuration found")
	}
}
