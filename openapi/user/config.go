package user

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/yaoapp/gou/application"
	"github.com/yaoapp/kun/log"
	"github.com/yaoapp/yao/config"
	"github.com/yaoapp/yao/openapi/oauth"
	"github.com/yaoapp/yao/openapi/oauth/types"
)

// Global variables to store loaded configurations
var (
	// Client config
	yaoClientConfig *YaoClientConfig

	// Global providers map (decoupled from locale-specific configs)
	providers = make(map[string]*Provider)
	// Team configurations by locale
	teamConfigs = make(map[string]*TeamConfig)
	// Entry configurations by locale (unified login + register)
	entryConfigs = make(map[string]*EntryConfig)
	// Mutex for thread safety
	configMutex sync.RWMutex
)

// Load loads all signin configurations from the openapi/user directory
func Load(appConfig config.Config) error {
	configMutex.Lock()
	defer configMutex.Unlock()

	// Clear existing configurations
	providers = make(map[string]*Provider)
	teamConfigs = make(map[string]*TeamConfig)
	entryConfigs = make(map[string]*EntryConfig)

	// Load entry configurations from openapi/user/entry directory
	err := loadEntryConfigs(appConfig.Root)
	if err != nil {
		return fmt.Errorf("failed to load entry configs: %v", err)
	}

	// Load team configurations from openapi/user/team directory
	err = loadTeamConfigs(appConfig.Root)
	if err != nil {
		return fmt.Errorf("failed to load team configs: %v", err)
	}

	// Load providers first
	err = loadProviders(appConfig.Root)
	if err != nil {
		return fmt.Errorf("failed to load providers: %v", err)
	}

	// Load client config
	err = loadClientConfig()
	if err != nil {
		return fmt.Errorf("failed to load client config: %v", err)
	}

	return nil
}

// loadClientConfig loads the client config from the openapi/user/client.yao file
func loadClientConfig() error {
	// Check if client config exists
	exists, err := application.App.Exists("openapi/user/client.yao")
	if err != nil {
		return fmt.Errorf("failed to check if client config exists: %v", err)
	}

	if !exists {
		return fmt.Errorf("client config not found")
	}

	// Read client config
	clientConfigRaw, err := application.App.Read("openapi/user/client.yao")
	if err != nil {
		return fmt.Errorf("failed to read client config: %v", err)
	}

	var clientConfig YaoClientConfig
	err = application.Parse("openapi/user/client.yao", clientConfigRaw, &clientConfig)
	if err != nil {
		return fmt.Errorf("failed to parse client config: %v", err)
	}

	// Process ENV variables in client config
	clientConfig.ClientID = replaceENVVar(clientConfig.ClientID)
	clientConfig.ClientSecret = replaceENVVar(clientConfig.ClientSecret)

	// Check if required values are missing or unresolved
	if clientConfig.ClientID == "" {
		return fmt.Errorf("client_id is required but not set")
	}

	// Check if ClientID still contains unresolved environment variable references
	if strings.HasPrefix(clientConfig.ClientID, "$ENV.") || strings.HasPrefix(clientConfig.ClientID, "${") || strings.HasPrefix(clientConfig.ClientID, "$") {
		envVarName := extractEnvVarName(clientConfig.ClientID)
		return fmt.Errorf("environment variable '%s' is required but not set", envVarName)
	}

	// ClientSecret is optional - if it contains unresolved environment variable references, set it to empty
	// This allows the system to generate a new secret during client registration
	if clientConfig.ClientSecret != "" && (strings.HasPrefix(clientConfig.ClientSecret, "$ENV.") || strings.HasPrefix(clientConfig.ClientSecret, "${") || strings.HasPrefix(clientConfig.ClientSecret, "$")) {
		// Log a warning but don't fail - the system will generate a new secret
		log.Warn("Client secret environment variable not set, will generate new secret during registration")
		clientConfig.ClientSecret = ""
	}

	// Validate client config
	err = validateClientConfig(&clientConfig)
	if err != nil {
		return fmt.Errorf("failed to validate client config: %v", err)
	}

	yaoClientConfig = &clientConfig
	return nil
}

// validateClientConfig validates the client config
func validateClientConfig(clientConfig *YaoClientConfig) error {

	// Validate client ID
	err := oauth.OAuth.ValidateClientID(clientConfig.ClientID)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Validate client is registered
	c := oauth.OAuth.GetClientProvider()
	_, err = c.GetClientByID(ctx, clientConfig.ClientID)
	if err != nil {
		// If client is not registered, register it
		if strings.Contains(err.Error(), "Client not found") {
			yaoClientConfig, err = registerClient(clientConfig.ClientID)
			if err != nil {
				return fmt.Errorf("failed to register client: %v", err)
			}
			return nil
		}
		return fmt.Errorf("failed to get client: %v", err)
	}

	return nil
}

// registerClient registers the client config with the OAuth server
func registerClient(clientID string) (*YaoClientConfig, error) {

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Register client
	response, err := oauth.OAuth.DynamicClientRegistration(ctx, &types.DynamicClientRegistrationRequest{
		ClientID:        clientID,
		ClientName:      "Yao OpenAPI Client",
		ResponseTypes:   []string{"code"},
		GrantTypes:      []string{"client_credentials"},
		ApplicationType: types.ApplicationTypeWeb,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create client: %v", err)
	}

	var clientConfig *YaoClientConfig = &YaoClientConfig{}
	clientConfig.ClientID = response.ClientID
	clientConfig.ClientSecret = response.ClientSecret
	clientConfig.ExpiresIn = 3600 * 24                  // 24 hours
	clientConfig.RefreshTokenExpiresIn = 3600 * 24 * 30 // 30 days
	clientConfig.Scopes = []string{"openid", "profile", "email"}
	return clientConfig, nil
}

// loadProviders loads all provider configurations from the openapi/user/providers directory
func loadProviders(_ string) error {
	// Use Walk to find all provider files in the signin/providers directory
	err := application.App.Walk("openapi/user/providers", func(root, filename string, isdir bool) error {
		if isdir {
			return nil
		}

		// Only process .yao files
		if !strings.HasSuffix(filename, ".yao") {
			return nil
		}

		// Skip client.yao file
		if filename == "client.yao" {
			return nil
		}

		// Extract provider ID from filename (basename without extension)
		baseName := filepath.Base(filename)
		providerID := strings.TrimSuffix(baseName, ".yao")

		// Read provider configuration
		configRaw, err := application.App.Read(filename)
		if err != nil {
			return fmt.Errorf("failed to read provider config %s: %v", filename, err)
		}

		// Parse the provider configuration
		var provider Provider
		err = application.Parse(filename, configRaw, &provider)
		if err != nil {
			return fmt.Errorf("failed to parse provider config %s: %v", filename, err)
		}

		// Set the provider ID
		provider.ID = providerID

		// Process ENV variables in the provider configuration
		provider.ClientID = replaceENVVar(provider.ClientID)
		provider.ClientSecret = replaceENVVar(provider.ClientSecret)

		// Store the provider globally
		providers[providerID] = &provider

		return nil
	})

	if err != nil {
		return fmt.Errorf("failed to walk providers directory: %v", err)
	}

	return nil
}

// loadTeamConfigs loads all team configurations from the openapi/user/team directory
func loadTeamConfigs(_ string) error {
	// Use Walk to find all configuration files in the team directory
	err := application.App.Walk("openapi/user/team", func(root, filename string, isdir bool) error {
		if isdir {
			return nil
		}

		// Only process .yao files
		if !strings.HasSuffix(filename, ".yao") {
			return nil
		}

		// Extract locale from filename (basename without extension)
		baseName := filepath.Base(filename)
		locale := strings.ToLower(strings.TrimSuffix(baseName, ".yao"))

		// Read configuration
		configRaw, err := application.App.Read(filename)
		if err != nil {
			return fmt.Errorf("failed to read team config %s: %v", filename, err)
		}

		// Parse the configuration
		var teamConfig TeamConfig
		err = application.Parse(filename, configRaw, &teamConfig)
		if err != nil {
			return fmt.Errorf("failed to parse team config %s: %v", filename, err)
		}

		// Store team configuration
		teamConfigs[locale] = &teamConfig

		return nil
	})

	if err != nil {
		return fmt.Errorf("failed to walk team directory: %v", err)
	}

	return nil
}

// GetProvider returns a provider by ID
func GetProvider(providerID string) (*Provider, error) {
	configMutex.RLock()
	defer configMutex.RUnlock()

	provider, exists := providers[providerID]
	if !exists {
		return nil, fmt.Errorf("provider '%s' not found", providerID)
	}

	return provider, nil
}

// GetYaoClientConfig returns the current yaoClientConfig
func GetYaoClientConfig() *YaoClientConfig {
	configMutex.RLock()
	defer configMutex.RUnlock()
	return yaoClientConfig
}

// GetTeamConfig returns the team configuration for a given locale
func GetTeamConfig(locale string) *TeamConfig {
	configMutex.RLock()
	defer configMutex.RUnlock()

	// Normalize language code to lowercase
	if locale != "" {
		locale = strings.TrimSpace(strings.ToLower(locale))
	}

	// Try to get the specific locale configuration
	if config, exists := teamConfigs[locale]; exists {
		return config
	}

	// If no specific locale, try to get "en" as default
	if config, exists := teamConfigs["en"]; exists {
		return config
	}

	// If "en" is not available, try to get any available configuration
	for _, config := range teamConfigs {
		return config
	}

	return nil
}

// extractEnvVarName extracts the environment variable name from a string like "$ENV.VAR_NAME"
func extractEnvVarName(value string) string {
	if value == "" {
		return "unknown"
	}

	// Handle $ENV.VAR_NAME format
	if strings.HasPrefix(value, "$ENV.") {
		return strings.TrimPrefix(value, "$ENV.")
	}

	// Handle ${VAR_NAME} format
	if strings.HasPrefix(value, "${") && strings.HasSuffix(value, "}") {
		return value[2 : len(value)-1]
	}

	// Handle $VAR_NAME format
	if strings.HasPrefix(value, "$") {
		return value[1:]
	}

	return "unknown"
}

// replaceENVVar replaces environment variables in a string
func replaceENVVar(value string) string {
	if value == "" {
		return value
	}

	// Replace ${ENV_VAR} or $ENV.VAR patterns
	re := regexp.MustCompile(`\$\{([^}]+)\}|\$([A-Za-z_][A-Za-z0-9_.]*)`)
	return re.ReplaceAllStringFunc(value, func(match string) string {
		var envVar string
		if strings.HasPrefix(match, "${") {
			// Extract from ${VAR} format
			envVar = match[2 : len(match)-1]
		} else {
			// Extract from $VAR format, remove $ENV. prefix if present
			envVar = match[1:]
			envVar = strings.TrimPrefix(envVar, "ENV.")
		}

		if envValue := os.Getenv(envVar); envValue != "" {
			return envValue
		}
		return match // Return original if env var not found
	})
}

// normalizeDuration normalizes various duration formats to Go's time.ParseDuration format
// Supports: s (seconds), m (minutes), h (hours), d (days)
func normalizeDuration(expiresIn string) (string, error) {
	if expiresIn == "" {
		return "", fmt.Errorf("empty duration")
	}

	// Common patterns and their conversions
	patterns := map[string]func(int) string{
		"s": func(n int) string { return fmt.Sprintf("%ds", n) },    // seconds
		"m": func(n int) string { return fmt.Sprintf("%dm", n) },    // minutes
		"h": func(n int) string { return fmt.Sprintf("%dh", n) },    // hours
		"d": func(n int) string { return fmt.Sprintf("%dh", n*24) }, // days -> hours
	}

	// Extract number and unit using regex
	re := regexp.MustCompile(`^(\d+)(\w+)$`)
	matches := re.FindStringSubmatch(expiresIn)

	if len(matches) != 3 {
		return "", fmt.Errorf("invalid duration format: %s", expiresIn)
	}

	number, err := strconv.Atoi(matches[1])
	if err != nil {
		return "", fmt.Errorf("invalid number in duration: %s", matches[1])
	}

	unit := matches[2]
	converter, exists := patterns[unit]
	if !exists {
		return "", fmt.Errorf("unsupported time unit: %s (supported: s, m, h, d)", unit)
	}

	normalized := converter(number)

	// Validate the normalized duration
	if _, err := time.ParseDuration(normalized); err != nil {
		return "", fmt.Errorf("failed to create valid duration: %v", err)
	}

	return normalized, nil
}

// processFormConfigENVVariables processes environment variables in the form configuration
func processFormConfigENVVariables(form *FormConfig) []string {
	var missingEnvVars []string

	if form == nil {
		return missingEnvVars
	}

	// Process form captcha options
	if form.Captcha != nil && form.Captcha.Options != nil {
		for key, value := range form.Captcha.Options {
			if strValue, ok := value.(string); ok {
				// Check if ENV variable exists before replacement
				if strings.HasPrefix(strValue, "$ENV.") {
					envVar := strings.TrimPrefix(strValue, "$ENV.")
					if _, exists := os.LookupEnv(envVar); !exists {
						missingEnvVars = append(missingEnvVars, envVar)
					}
				}
				form.Captcha.Options[key] = replaceENVVar(strValue)
			}
		}
	}

	return missingEnvVars
}

// loadEntryConfigs loads all entry configurations from the openapi/user/entry directory
// Entry config merges signin and register configurations
func loadEntryConfigs(_ string) error {
	// Use Walk to find all configuration files in the entry directory
	err := application.App.Walk("openapi/user/entry", func(root, filename string, isdir bool) error {
		if isdir {
			return nil
		}

		// Only process .yao files
		if !strings.HasSuffix(filename, ".yao") {
			return nil
		}

		// Extract locale from filename (basename without extension)
		baseName := filepath.Base(filename)
		locale := strings.ToLower(strings.TrimSuffix(baseName, ".yao"))

		// Read configuration
		configRaw, err := application.App.Read(filename)
		if err != nil {
			return fmt.Errorf("failed to read entry config %s: %v", filename, err)
		}

		// Parse the configuration
		var config EntryConfig
		err = application.Parse(filename, configRaw, &config)
		if err != nil {
			return fmt.Errorf("failed to parse entry config %s: %v", filename, err)
		}

		// Process ENV variables in the configuration
		processEntryConfigENVVariables(&config)

		// Store entry configuration
		entryConfigs[locale] = &config

		return nil
	})

	if err != nil {
		return fmt.Errorf("failed to walk entry directory: %v", err)
	}

	return nil
}

// processEntryConfigENVVariables processes environment variables in the entry configuration
func processEntryConfigENVVariables(config *EntryConfig) {
	var missingEnvVars []string

	// Process client_id and client_secret (from signin config)
	if strings.HasPrefix(config.ClientID, "$ENV.") {
		envVar := strings.TrimPrefix(config.ClientID, "$ENV.")
		if _, exists := os.LookupEnv(envVar); !exists {
			missingEnvVars = append(missingEnvVars, envVar)
		}
	}
	config.ClientID = replaceENVVar(config.ClientID)

	if strings.HasPrefix(config.ClientSecret, "$ENV.") {
		envVar := strings.TrimPrefix(config.ClientSecret, "$ENV.")
		if _, exists := os.LookupEnv(envVar); !exists {
			missingEnvVars = append(missingEnvVars, envVar)
		}
	}
	config.ClientSecret = replaceENVVar(config.ClientSecret)

	// Process form configuration
	formMissingVars := processFormConfigENVVariables(config.Form)
	missingEnvVars = append(missingEnvVars, formMissingVars...)

	// Log warning for missing environment variables
	if len(missingEnvVars) > 0 {
		fmt.Printf("Warning: The following environment variables are not set in entry configuration: %v\n", missingEnvVars)
	}
}

// GetEntryConfig returns the entry configuration for a given locale
func GetEntryConfig(locale string) *EntryConfig {
	configMutex.RLock()
	defer configMutex.RUnlock()

	// Normalize language code to lowercase
	if locale != "" {
		locale = strings.TrimSpace(strings.ToLower(locale))
	}

	// Try to get the specific locale configuration
	if config, exists := entryConfigs[locale]; exists {
		return config
	}

	// If no specific locale, try to get "en" as default
	if config, exists := entryConfigs["en"]; exists {
		return config
	}

	// If "en" is not available, try to get any available configuration
	for _, config := range entryConfigs {
		return config
	}

	return nil
}
