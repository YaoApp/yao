package openapi_test

import (
	"encoding/json"
	"io"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/yaoapp/yao/config"
	"github.com/yaoapp/yao/openapi"
	"github.com/yaoapp/yao/openapi/signin"
	"github.com/yaoapp/yao/openapi/tests/testutils"
)

func TestSigninLoad(t *testing.T) {
	// Initialize test environment
	serverURL := testutils.Prepare(t)
	defer testutils.Clean()

	_ = serverURL // Server URL not needed for this test

	// Test loading signin configurations
	err := signin.Load(config.Conf)
	assert.NoError(t, err, "signin.Load should succeed")

	// Test that we can get available languages
	languages := signin.GetAvailableLanguages()
	assert.IsType(t, []string{}, languages, "Should return string slice")
	t.Logf("Available languages: %v", languages)

	// Test default language
	defaultLang := signin.GetDefaultLanguage()
	assert.IsType(t, "", defaultLang, "Should return string")
	t.Logf("Default language: %s", defaultLang)
}

func TestSigninGetConfigs(t *testing.T) {
	// Initialize test environment
	serverURL := testutils.Prepare(t)
	defer testutils.Clean()

	_ = serverURL // Server URL not needed for this test

	// Load signin configurations
	err := signin.Load(config.Conf)
	assert.NoError(t, err, "signin.Load should succeed")

	// Test getting configs for different languages
	testCases := []string{"", "en", "zh-cn", "fr"}

	for _, lang := range testCases {
		t.Run("lang_"+lang, func(t *testing.T) {
			fullConfig := signin.GetFullConfig(lang)
			publicConfig := signin.GetPublicConfig(lang)

			if fullConfig != nil {
				t.Logf("Full config for '%s': %+v", lang, fullConfig.Title)
				assert.NotNil(t, publicConfig, "Public config should exist if full config exists")

				// Test that public config removes sensitive data
				if fullConfig.ThirdParty != nil && fullConfig.ThirdParty.Providers != nil {
					for i := range fullConfig.ThirdParty.Providers {
						if publicConfig.ThirdParty != nil && i < len(publicConfig.ThirdParty.Providers) {
							publicProvider := publicConfig.ThirdParty.Providers[i]
							assert.Empty(t, publicProvider.ClientSecret, "Client secret should be empty in public config")
							assert.Nil(t, publicProvider.ClientSecretGenerator, "Client secret generator should be nil in public config")
						}
					}
				}
			} else {
				t.Logf("No config found for language: %s", lang)
			}
		})
	}
}

func TestSigninLanguageNormalization(t *testing.T) {
	// Initialize test environment
	serverURL := testutils.Prepare(t)
	defer testutils.Clean()

	_ = serverURL // Server URL not needed for this test

	// Load signin configurations
	err := signin.Load(config.Conf)
	assert.NoError(t, err, "signin.Load should succeed")

	// Test that language codes are normalized to lowercase
	config1 := signin.GetFullConfig("EN")
	config2 := signin.GetFullConfig("en")
	assert.Equal(t, config1, config2, "Language codes should be normalized to lowercase")

	config3 := signin.GetPublicConfig("ZH-CN")
	config4 := signin.GetPublicConfig("zh-cn")
	assert.Equal(t, config3, config4, "Language codes should be normalized to lowercase")
}

func TestSigninConfigStructure(t *testing.T) {
	// Initialize test environment
	serverURL := testutils.Prepare(t)
	defer testutils.Clean()

	_ = serverURL // Server URL not needed for this test

	// Load signin configurations
	err := signin.Load(config.Conf)
	assert.NoError(t, err, "signin.Load should succeed")

	// Get a config to test structure
	config := signin.GetFullConfig("")
	if config != nil {
		t.Logf("Config loaded successfully with title: %s", config.Title)

		// Verify config structure is valid
		assert.IsType(t, &signin.Config{}, config, "Should return correct config type")

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
				assert.IsType(t, []*signin.Provider{}, config.ThirdParty.Providers, "Providers should be slice of Provider pointers")
				for i, provider := range config.ThirdParty.Providers {
					t.Logf("Provider %d: %s", i, provider.ID)
					assert.IsType(t, []string{}, provider.Scopes, "Provider scopes should be string slice")
					assert.IsType(t, map[string]string{}, provider.Mapping, "Provider mapping should be string map")
				}
			}
		}
	} else {
		t.Log("No signin configuration found")
	}
}

func TestSigninAPI(t *testing.T) {
	// Initialize test environment
	serverURL := testutils.Prepare(t)
	defer testutils.Clean()

	// Get base URL from server config
	baseURL := ""
	if openapi.Server != nil && openapi.Server.Config != nil {
		baseURL = openapi.Server.Config.BaseURL
	}

	// Test API endpoints
	testCases := []struct {
		name       string
		endpoint   string
		expectCode int
	}{
		{"get config without locale", "/signin", 200},
		{"get config with en locale", "/signin?locale=en", 200},
		{"get config with zh-cn locale", "/signin?locale=zh-cn", 200},
		{"get config with invalid locale", "/signin?locale=invalid", 200}, // should fallback to default
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			url := serverURL + baseURL + tc.endpoint
			resp, err := http.Get(url)
			assert.NoError(t, err, "HTTP request should succeed")

			if resp != nil {
				defer resp.Body.Close()
				assert.Equal(t, tc.expectCode, resp.StatusCode, "Expected status code %d", tc.expectCode)

				if resp.StatusCode == 200 {
					// Parse response body
					body, err := io.ReadAll(resp.Body)
					assert.NoError(t, err, "Should read response body")

					var config signin.Config
					err = json.Unmarshal(body, &config)
					assert.NoError(t, err, "Should parse JSON response")

					t.Logf("API response for %s: %s", tc.endpoint, config.Title)

					// Verify it's public config (no sensitive data)
					if config.ThirdParty != nil && config.ThirdParty.Providers != nil {
						for _, provider := range config.ThirdParty.Providers {
							assert.Empty(t, provider.ClientSecret, "Client secret should be empty in API response")
							assert.Nil(t, provider.ClientSecretGenerator, "Client secret generator should be nil in API response")
						}
					}
				}
			}
		})
	}
}
