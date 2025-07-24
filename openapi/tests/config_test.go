package openapi_test

import (
	"path/filepath"
	"strings"
	"testing"
	"time"

	jsoniter "github.com/json-iterator/go"
	"github.com/stretchr/testify/assert"
	"github.com/yaoapp/yao/openapi"
	"github.com/yaoapp/yao/openapi/oauth/types"
)

func TestConfigUnmarshalJSON_TimeParsingCorrect(t *testing.T) {
	jsonData := `{
		"baseurl": "/v1",
		"store": "__yao.oauth.store",
		"cache": "__yao.oauth.cache",
		"oauth": {
			"issuer_url": "https://localhost:5099",
			"signing": {
				"cert_rotation_interval": "24h"
			},
			"token": {
				"access_token_lifetime": "1h",
				"refresh_token_lifetime": "24h",
				"authorization_code_lifetime": "10m",
				"device_code_lifetime": "15m",
				"device_code_interval": "5s"
			},
			"security": {
				"state_parameter_lifetime": "10m",
				"rate_limit_window": "1m",
				"lockout_duration": "15m"
			},
			"client": {
				"client_secret_lifetime": "0s"
			}
		}
	}`

	var config openapi.Config
	err := jsoniter.Unmarshal([]byte(jsonData), &config)
	assert.NoError(t, err, "JSON unmarshaling should succeed")

	// Test that duration strings are correctly parsed
	assert.Equal(t, 24*time.Hour, config.OAuth.Signing.CertRotationInterval)
	assert.Equal(t, time.Hour, config.OAuth.Token.AccessTokenLifetime)
	assert.Equal(t, 24*time.Hour, config.OAuth.Token.RefreshTokenLifetime)
	assert.Equal(t, 10*time.Minute, config.OAuth.Token.AuthorizationCodeLifetime)
	assert.Equal(t, 15*time.Minute, config.OAuth.Token.DeviceCodeLifetime)
	assert.Equal(t, 5*time.Second, config.OAuth.Token.DeviceCodeInterval)
	assert.Equal(t, 10*time.Minute, config.OAuth.Security.StateParameterLifetime)
	assert.Equal(t, time.Minute, config.OAuth.Security.RateLimitWindow)
	assert.Equal(t, 15*time.Minute, config.OAuth.Security.LockoutDuration)
	assert.Equal(t, time.Duration(0), config.OAuth.Client.ClientSecretLifetime)

	// Test other fields are correctly parsed
	assert.Equal(t, "/v1", config.BaseURL)
	assert.Equal(t, "__yao.oauth.store", config.Store)
	assert.Equal(t, "__yao.oauth.cache", config.Cache)
	assert.Equal(t, "https://localhost:5099", config.OAuth.IssuerURL)
}

func TestParseDuration(t *testing.T) {
	tests := []struct {
		input    string
		expected time.Duration
		hasError bool
	}{
		{"24h", 24 * time.Hour, false},
		{"1h", time.Hour, false},
		{"10m", 10 * time.Minute, false},
		{"5s", 5 * time.Second, false},
		{"0s", 0, false},
		{"0", 0, false},
		{"", 0, false},
		{"invalid", 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result, err := parseDuration(tt.input)
			if tt.hasError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}

func TestFormatDuration(t *testing.T) {
	tests := []struct {
		input    time.Duration
		expected string
	}{
		{24 * time.Hour, "24h0m0s"},
		{time.Hour, "1h0m0s"},
		{10 * time.Minute, "10m0s"},
		{5 * time.Second, "5s"},
		{0, "0s"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			result := formatDuration(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestConfigMarshalUnmarshalRoundTrip(t *testing.T) {
	// Create a config with duration fields
	originalConfig := &openapi.Config{
		BaseURL: "/v1",
		Store:   "__yao.oauth.store",
		Cache:   "__yao.oauth.cache",
		OAuth: &openapi.OAuth{
			IssuerURL: "https://localhost:5099",
			Signing: types.SigningConfig{
				SigningCertPath:      "/path/to/cert.pem",
				SigningKeyPath:       "/path/to/key.pem",
				CertRotationInterval: 24 * time.Hour,
			},
			Token: types.TokenConfig{
				AccessTokenLifetime:       time.Hour,
				RefreshTokenLifetime:      24 * time.Hour,
				AuthorizationCodeLifetime: 10 * time.Minute,
				DeviceCodeLifetime:        15 * time.Minute,
				DeviceCodeInterval:        5 * time.Second,
			},
			Security: types.SecurityConfig{
				StateParameterLifetime: 10 * time.Minute,
				RateLimitWindow:        time.Minute,
				LockoutDuration:        15 * time.Minute,
			},
			Client: types.ClientConfig{
				ClientSecretLifetime: 0, // No expiration
			},
		},
	}

	// Marshal to JSON
	jsonData, err := jsoniter.Marshal(originalConfig)
	assert.NoError(t, err, "Marshal should succeed")

	// Unmarshal back to config
	var unmarshaledConfig openapi.Config
	err = jsoniter.Unmarshal(jsonData, &unmarshaledConfig)
	assert.NoError(t, err, "Unmarshal should succeed")

	// Compare the original and unmarshaled configs
	assert.Equal(t, originalConfig.BaseURL, unmarshaledConfig.BaseURL)
	assert.Equal(t, originalConfig.Store, unmarshaledConfig.Store)
	assert.Equal(t, originalConfig.Cache, unmarshaledConfig.Cache)
	assert.Equal(t, originalConfig.OAuth.IssuerURL, unmarshaledConfig.OAuth.IssuerURL)

	// Compare duration fields
	assert.Equal(t, originalConfig.OAuth.Signing.CertRotationInterval, unmarshaledConfig.OAuth.Signing.CertRotationInterval)
	assert.Equal(t, originalConfig.OAuth.Token.AccessTokenLifetime, unmarshaledConfig.OAuth.Token.AccessTokenLifetime)
	assert.Equal(t, originalConfig.OAuth.Token.RefreshTokenLifetime, unmarshaledConfig.OAuth.Token.RefreshTokenLifetime)
	assert.Equal(t, originalConfig.OAuth.Token.AuthorizationCodeLifetime, unmarshaledConfig.OAuth.Token.AuthorizationCodeLifetime)
	assert.Equal(t, originalConfig.OAuth.Token.DeviceCodeLifetime, unmarshaledConfig.OAuth.Token.DeviceCodeLifetime)
	assert.Equal(t, originalConfig.OAuth.Token.DeviceCodeInterval, unmarshaledConfig.OAuth.Token.DeviceCodeInterval)
	assert.Equal(t, originalConfig.OAuth.Security.StateParameterLifetime, unmarshaledConfig.OAuth.Security.StateParameterLifetime)
	assert.Equal(t, originalConfig.OAuth.Security.RateLimitWindow, unmarshaledConfig.OAuth.Security.RateLimitWindow)
	assert.Equal(t, originalConfig.OAuth.Security.LockoutDuration, unmarshaledConfig.OAuth.Security.LockoutDuration)
	assert.Equal(t, originalConfig.OAuth.Client.ClientSecretLifetime, unmarshaledConfig.OAuth.Client.ClientSecretLifetime)

	// Verify that the JSON contains human-readable duration strings
	jsonString := string(jsonData)
	assert.Contains(t, jsonString, `"cert_rotation_interval":"24h0m0s"`)
	assert.Contains(t, jsonString, `"access_token_lifetime":"1h0m0s"`)
	assert.Contains(t, jsonString, `"authorization_code_lifetime":"10m0s"`)
	assert.Contains(t, jsonString, `"device_code_interval":"5s"`)
	assert.Contains(t, jsonString, `"client_secret_lifetime":"0s"`)
}

// TestConfigJSONOutputDemo demonstrates the human-readable JSON output format
func TestConfigJSONOutputDemo(t *testing.T) {
	config := &openapi.Config{
		BaseURL: "/v1",
		Store:   "__yao.oauth.store",
		Cache:   "__yao.oauth.cache",
		OAuth: &openapi.OAuth{
			IssuerURL: "https://localhost:5099",
			Signing: types.SigningConfig{
				SigningCertPath:      "openapi/certs/signing-cert.pem",
				SigningKeyPath:       "openapi/certs/signing-key.pem",
				CertRotationInterval: 24 * time.Hour,
			},
			Token: types.TokenConfig{
				AccessTokenLifetime:       time.Hour,
				RefreshTokenLifetime:      24 * time.Hour,
				AuthorizationCodeLifetime: 10 * time.Minute,
				DeviceCodeLifetime:        15 * time.Minute,
				DeviceCodeInterval:        5 * time.Second,
				AccessTokenFormat:         "jwt",
				RefreshTokenFormat:        "opaque",
			},
		},
	}

	jsonData, err := jsoniter.MarshalIndent(config, "", "  ")
	assert.NoError(t, err)

	t.Logf("Human-readable JSON output:\n%s", string(jsonData))

	// Verify key duration fields are formatted as strings
	jsonString := string(jsonData)
	assert.Contains(t, jsonString, `"cert_rotation_interval":"24h0m0s"`)
	assert.Contains(t, jsonString, `"access_token_lifetime":"1h0m0s"`)
	assert.Contains(t, jsonString, `"device_code_interval":"5s"`)
}

func TestConvertRelativeToAbsolutePath(t *testing.T) {
	tests := []struct {
		name         string
		relativePath string
		rootPath     string
		expected     string
	}{
		{
			name:         "basic relative path",
			relativePath: "signing-cert.pem",
			rootPath:     "/app",
			expected:     "/app/openapi/certs/signing-cert.pem",
		},
		{
			name:         "relative path with subdirectory",
			relativePath: "ssl/signing-cert.pem",
			rootPath:     "/app",
			expected:     "/app/openapi/certs/ssl/signing-cert.pem",
		},
		{
			name:         "empty relative path",
			relativePath: "",
			rootPath:     "/app",
			expected:     "",
		},
		{
			name:         "already absolute path",
			relativePath: "/absolute/path/cert.pem",
			rootPath:     "/app",
			expected:     "/absolute/path/cert.pem",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := convertRelativeToAbsolutePath(tt.relativePath, tt.rootPath)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestConvertAbsoluteToRelativePath(t *testing.T) {
	tests := []struct {
		name         string
		absolutePath string
		rootPath     string
		expected     string
	}{
		{
			name:         "basic absolute path",
			absolutePath: "/app/openapi/certs/signing-cert.pem",
			rootPath:     "/app",
			expected:     "signing-cert.pem",
		},
		{
			name:         "absolute path with subdirectory",
			absolutePath: "/app/openapi/certs/ssl/signing-cert.pem",
			rootPath:     "/app",
			expected:     "ssl/signing-cert.pem",
		},
		{
			name:         "empty absolute path",
			absolutePath: "",
			rootPath:     "/app",
			expected:     "",
		},
		{
			name:         "already relative path",
			absolutePath: "relative/path/cert.pem",
			rootPath:     "/app",
			expected:     "relative/path/cert.pem",
		},
		{
			name:         "path not matching pattern",
			absolutePath: "/other/path/cert.pem",
			rootPath:     "/app",
			expected:     "/other/path/cert.pem",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := convertAbsoluteToRelativePath(tt.absolutePath, tt.rootPath)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestCertificatePathConversion(t *testing.T) {
	t.Run("complete path conversion cycle", func(t *testing.T) {
		rootPath := "/app"
		originalRelativePath := "ssl/signing-cert.pem"

		// Convert relative to absolute
		absolutePath := convertRelativeToAbsolutePath(originalRelativePath, rootPath)
		expected := "/app/openapi/certs/ssl/signing-cert.pem"
		assert.Equal(t, expected, absolutePath)

		// Convert absolute back to relative
		convertedRelativePath := convertAbsoluteToRelativePath(absolutePath, rootPath)
		assert.Equal(t, originalRelativePath, convertedRelativePath)
	})

	t.Run("path conversion with different scenarios", func(t *testing.T) {
		testCases := []struct {
			name     string
			relative string
			root     string
			absolute string
		}{
			{
				name:     "simple certificate",
				relative: "cert.pem",
				root:     "/app",
				absolute: "/app/openapi/certs/cert.pem",
			},
			{
				name:     "nested directory",
				relative: "ssl/prod/cert.pem",
				root:     "/production",
				absolute: "/production/openapi/certs/ssl/prod/cert.pem",
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				// Test conversion to absolute
				absolute := convertRelativeToAbsolutePath(tc.relative, tc.root)
				assert.Equal(t, tc.absolute, absolute)

				// Test conversion back to relative
				relative := convertAbsoluteToRelativePath(tc.absolute, tc.root)
				assert.Equal(t, tc.relative, relative)
			})
		}
	})
}

// parseDuration parses a time duration string (e.g., "24h", "1h", "10m") into time.Duration
func parseDuration(durationStr string) (time.Duration, error) {
	if durationStr == "" || durationStr == "0" || durationStr == "0s" {
		return 0, nil
	}
	return time.ParseDuration(durationStr)
}

// formatDuration converts time.Duration to human-readable string format
func formatDuration(duration time.Duration) string {
	if duration == 0 {
		return "0s"
	}
	return duration.String()
}

// convertRelativeToAbsolutePath converts relative certificate path to absolute path
func convertRelativeToAbsolutePath(relativePath, rootPath string) string {
	if relativePath == "" {
		return ""
	}
	// If already absolute path, return as is
	if filepath.IsAbs(relativePath) {
		return relativePath
	}
	// Convert relative path to absolute: Root + "openapi" + "certs" + relativePath
	return filepath.Join(rootPath, "openapi", "certs", relativePath)
}

// convertAbsoluteToRelativePath converts absolute certificate path to relative path
func convertAbsoluteToRelativePath(absolutePath, rootPath string) string {
	if absolutePath == "" {
		return ""
	}
	// If not absolute path, return as is
	if !filepath.IsAbs(absolutePath) {
		return absolutePath
	}

	// Remove Root + "openapi" + "certs" prefix
	certBasePath := filepath.Join(rootPath, "openapi", "certs")
	if strings.HasPrefix(absolutePath, certBasePath) {
		relativePath := strings.TrimPrefix(absolutePath, certBasePath)
		// Remove leading separator
		relativePath = strings.TrimPrefix(relativePath, string(filepath.Separator))
		return relativePath
	}

	// If path doesn't match expected pattern, return as is
	return absolutePath
}
