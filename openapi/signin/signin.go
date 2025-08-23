package signin

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

	// Full configurations with sensitive data (for backend use)
	fullConfigs = make(map[string]*Config)
	// Public configurations without sensitive data (for frontend use)
	publicConfigs = make(map[string]*Config)
	// Global providers map (decoupled from locale-specific configs)
	providers = make(map[string]*Provider)
	// Default configuration (marked with default: true)
	defaultConfig *Config
	// Mutex for thread safety
	configMutex sync.RWMutex
)

// Load loads all signin configurations from the openapi/signin directory
func Load(appConfig config.Config) error {
	configMutex.Lock()
	defer configMutex.Unlock()

	// Clear existing configurations
	fullConfigs = make(map[string]*Config)
	publicConfigs = make(map[string]*Config)
	providers = make(map[string]*Provider)
	defaultConfig = nil

	// Load signin configurations
	err := loadSigninConfigs(appConfig.Root)
	if err != nil {
		return fmt.Errorf("failed to load signin configs: %v", err)
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

// loadClientConfig loads the client config from the openapi/signin/client.yao file
func loadClientConfig() error {
	// Check if client config exists
	exists, err := application.App.Exists("openapi/signin/client.yao")
	if err != nil {
		return fmt.Errorf("failed to check if client config exists: %v", err)
	}

	if !exists {
		return fmt.Errorf("client config not found")
	}

	// Read client config
	clientConfigRaw, err := application.App.Read("openapi/signin/client.yao")
	if err != nil {
		return fmt.Errorf("failed to read client config: %v", err)
	}

	var clientConfig YaoClientConfig
	err = application.Parse("openapi/signin/client.yao", clientConfigRaw, &clientConfig)
	if err != nil {
		return fmt.Errorf("failed to parse client config: %v", err)
	}

	// Process ENV variables in client config
	clientConfig.ClientID = replaceENVVar(clientConfig.ClientID)
	clientConfig.ClientSecret = replaceENVVar(clientConfig.ClientSecret)

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

// loadProviders loads all provider configurations from the openapi/signin/providers directory
func loadProviders(rootPath string) error {
	// Use Walk to find all provider files in the signin/providers directory
	err := application.App.Walk("openapi/signin/providers", func(root, filename string, isdir bool) error {
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

		// Process ENV variables in provider config
		processProviderENVVariables(&provider, rootPath)

		// Store the provider
		providers[providerID] = &provider

		return nil
	}, "*.yao")

	return err
}

// loadSigninConfigs loads all signin configurations from the openapi/signin directory
func loadSigninConfigs(rootPath string) error {
	// Use Walk to find all signin config files in the signin directory (but not subdirectories)
	err := application.App.Walk("openapi/signin", func(root, filename string, isdir bool) error {
		if isdir {
			return nil
		}

		// Skip files in subdirectories (like providers/)
		if filepath.Dir(filename) != "openapi/signin" {
			return nil
		}

		// Only process .yao files
		if !strings.HasSuffix(filename, ".yao") {
			return nil
		}

		// Extract language code from filename
		baseName := filepath.Base(filename)
		lang := extractLanguageFromFilename(baseName)

		// Read signin configuration
		configRaw, err := application.App.Read(filename)
		if err != nil {
			return fmt.Errorf("failed to read signin config %s: %v", filename, err)
		}

		// Parse the configuration
		var signinConfig Config
		err = application.Parse(filename, configRaw, &signinConfig)
		if err != nil {
			return fmt.Errorf("failed to parse signin config %s: %v", filename, err)
		}

		// Process ENV variables in full config
		fullConfig := signinConfig
		processConfigENVVariables(&fullConfig, rootPath)

		// Set as default config if marked as default
		if fullConfig.Default {
			defaultConfig = &fullConfig
		}

		// Create public config (without sensitive data)
		publicConfig := createPublicConfig(&fullConfig)

		// Store configurations
		fullConfigs[lang] = &fullConfig
		publicConfigs[lang] = &publicConfig

		return nil
	}, "*.yao")

	return err
}

// extractLanguageFromFilename extracts language code from filename
func extractLanguageFromFilename(filename string) string {
	// New naming convention:
	// en.yao -> "en"
	// zh-cn.yao -> "zh-cn"
	// default.yao -> "default"

	baseName := strings.TrimSuffix(filename, ".yao")
	return strings.ToLower(baseName)
}

// processProviderENVVariables processes environment variables in the provider configuration
func processProviderENVVariables(provider *Provider, rootPath string) {
	var missingEnvVars []string

	// Process ClientID
	if strings.HasPrefix(provider.ClientID, "$ENV.") {
		envVar := strings.TrimPrefix(provider.ClientID, "$ENV.")
		if _, exists := os.LookupEnv(envVar); !exists {
			missingEnvVars = append(missingEnvVars, envVar)
		}
	}
	provider.ClientID = replaceENVVar(provider.ClientID)

	// Process ClientSecret
	if strings.HasPrefix(provider.ClientSecret, "$ENV.") {
		envVar := strings.TrimPrefix(provider.ClientSecret, "$ENV.")
		if _, exists := os.LookupEnv(envVar); !exists {
			missingEnvVars = append(missingEnvVars, envVar)
		}
	}
	provider.ClientSecret = replaceENVVar(provider.ClientSecret)

	// Process client secret generator
	if provider.ClientSecretGenerator != nil {
		// Check PrivateKey
		if strings.HasPrefix(provider.ClientSecretGenerator.PrivateKey, "$ENV.") {
			envVar := strings.TrimPrefix(provider.ClientSecretGenerator.PrivateKey, "$ENV.")
			if _, exists := os.LookupEnv(envVar); !exists {
				missingEnvVars = append(missingEnvVars, envVar)
			}
		}
		provider.ClientSecretGenerator.PrivateKey = replaceENVVar(provider.ClientSecretGenerator.PrivateKey)

		// Convert relative path to absolute path for private key
		if provider.ClientSecretGenerator.PrivateKey != "" && !filepath.IsAbs(provider.ClientSecretGenerator.PrivateKey) {
			provider.ClientSecretGenerator.PrivateKey = filepath.Join(rootPath, "openapi", "certs", provider.ClientSecretGenerator.PrivateKey)
		}

		// Process and normalize expires_in format
		if provider.ClientSecretGenerator.ExpiresIn != "" {
			normalizedDuration, err := normalizeExpiresIn(provider.ClientSecretGenerator.ExpiresIn)
			if err != nil {
				log.Warn("Invalid expires_in format '%s' for provider '%s': %v",
					provider.ClientSecretGenerator.ExpiresIn, provider.ID, err)
				// Set default to 90 days
				provider.ClientSecretGenerator.ExpiresIn = "2160h" // 90 * 24 hours
			} else {
				provider.ClientSecretGenerator.ExpiresIn = normalizedDuration
			}
		}

		// Process header values
		if provider.ClientSecretGenerator.Header != nil {
			for key, value := range provider.ClientSecretGenerator.Header {
				if strValue, ok := value.(string); ok {
					if strings.HasPrefix(strValue, "$ENV.") {
						envVar := strings.TrimPrefix(strValue, "$ENV.")
						if _, exists := os.LookupEnv(envVar); !exists {
							missingEnvVars = append(missingEnvVars, envVar)
						}
					}
					provider.ClientSecretGenerator.Header[key] = replaceENVVar(strValue)
				}
			}
		}

		// Process payload values
		if provider.ClientSecretGenerator.Payload != nil {
			for key, value := range provider.ClientSecretGenerator.Payload {
				if strValue, ok := value.(string); ok {
					if strings.HasPrefix(strValue, "$ENV.") {
						envVar := strings.TrimPrefix(strValue, "$ENV.")
						if _, exists := os.LookupEnv(envVar); !exists {
							missingEnvVars = append(missingEnvVars, envVar)
						}
					}
					provider.ClientSecretGenerator.Payload[key] = replaceENVVar(strValue)
				}
			}
		}
	}

	// Log warning for missing environment variables
	if len(missingEnvVars) > 0 {
		log.Warn("The following environment variables are not set for provider '%s': %v", provider.ID, missingEnvVars)
	}
}

// processConfigENVVariables processes environment variables in the signin configuration
func processConfigENVVariables(config *Config, rootPath string) {
	var missingEnvVars []string

	// Process client_id and client_secret
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

	// Process form captcha options
	if config.Form != nil && config.Form.Captcha != nil && config.Form.Captcha.Options != nil {
		for key, value := range config.Form.Captcha.Options {
			if strValue, ok := value.(string); ok {
				// Check if ENV variable exists before replacement
				if strings.HasPrefix(strValue, "$ENV.") {
					envVar := strings.TrimPrefix(strValue, "$ENV.")
					if _, exists := os.LookupEnv(envVar); !exists {
						missingEnvVars = append(missingEnvVars, envVar)
					}
				}
				config.Form.Captcha.Options[key] = replaceENVVar(strValue)
			}
		}
	}

	// Note: Third party providers are now handled separately in loadProviders()
	// No need to process provider configurations here anymore

	// Log warning for missing environment variables
	if len(missingEnvVars) > 0 {
		log.Warn("The following environment variables are not set in signin configuration: %v", missingEnvVars)
		log.Warn("Please set these environment variables to avoid exposing placeholder values in configuration")
	}
}

// replaceENVVar replaces environment variables in the format $ENV.VAR_NAME
func replaceENVVar(value string) string {
	if strings.HasPrefix(value, "$ENV.") {
		envVar := strings.TrimPrefix(value, "$ENV.")
		envValue, exists := os.LookupEnv(envVar)
		if exists {
			return envValue
		}
		// If environment variable doesn't exist, return empty string for security
		// Never expose ENV placeholder values to prevent configuration leakage
		return ""
	}
	return value
}

// createPublicConfig creates a public version of the configuration without sensitive data
func createPublicConfig(fullConfig *Config) Config {
	// Perform deep copy to avoid modifying the original fullConfig
	publicConfig := Config{
		Title:       fullConfig.Title,
		Description: fullConfig.Description,
		SuccessURL:  fullConfig.SuccessURL,
		FailureURL:  fullConfig.FailureURL,
	}

	// Deep copy Form configuration
	if fullConfig.Form != nil {
		publicConfig.Form = &FormConfig{
			ForgotPasswordLink: fullConfig.Form.ForgotPasswordLink,
			RememberMe:         fullConfig.Form.RememberMe,
			RegisterLink:       fullConfig.Form.RegisterLink,
			TermsOfServiceLink: fullConfig.Form.TermsOfServiceLink,
			PrivacyPolicyLink:  fullConfig.Form.PrivacyPolicyLink,
		}

		// Deep copy Username configuration
		if fullConfig.Form.Username != nil {
			publicConfig.Form.Username = &UsernameConfig{
				Placeholder: fullConfig.Form.Username.Placeholder,
				Fields:      append([]string(nil), fullConfig.Form.Username.Fields...),
			}
		}

		// Deep copy Password configuration
		if fullConfig.Form.Password != nil {
			publicConfig.Form.Password = &PasswordConfig{
				Placeholder: fullConfig.Form.Password.Placeholder,
			}
		}

		// Deep copy Captcha configuration with sensitive data removal
		if fullConfig.Form.Captcha != nil {
			publicConfig.Form.Captcha = &CaptchaConfig{
				Type: fullConfig.Form.Captcha.Type,
			}

			if fullConfig.Form.Captcha.Options != nil {
				// Create a new options map without sensitive fields
				publicOptions := make(map[string]interface{})
				for key, value := range fullConfig.Form.Captcha.Options {
					// Only include non-sensitive fields
					switch key {
					case "sitekey", "theme", "size", "action", "cdata", "response_mode":
						// These are safe to expose to frontend
						publicOptions[key] = value
					case "secret":
						// Remove secret field - this should never be exposed to frontend
						continue
					default:
						// For unknown fields, be conservative and exclude them
						continue
					}
				}
				publicConfig.Form.Captcha.Options = publicOptions
			}
		}
	}

	// Deep copy Token configuration
	if fullConfig.Token != nil {
		publicConfig.Token = &TokenConfig{
			ExpiresIn:           fullConfig.Token.ExpiresIn,
			RememberMeExpiresIn: fullConfig.Token.RememberMeExpiresIn,
		}
	}

	// Deep copy ThirdParty configuration with sensitive data removal
	if fullConfig.ThirdParty != nil {
		publicConfig.ThirdParty = &ThirdParty{}

		// Deep copy Providers with sensitive data removal
		if fullConfig.ThirdParty.Providers != nil {
			publicProviders := make([]*Provider, len(fullConfig.ThirdParty.Providers))
			for i, provider := range fullConfig.ThirdParty.Providers {
				publicProvider := Provider{
					ID:        provider.ID,
					Title:     provider.Title,
					Logo:      provider.Logo,
					Color:     provider.Color,
					TextColor: provider.TextColor,
					// Only expose display fields for frontend
					// Remove sensitive fields: ClientID, ClientSecret, ClientSecretGenerator, Scopes, Endpoints, Mapping, Register
				}

				publicProviders[i] = &publicProvider
			}
			publicConfig.ThirdParty.Providers = publicProviders
		}
	}

	return publicConfig
}

// GetFullConfig returns the full configuration for a given language
func GetFullConfig(lang string) *Config {
	configMutex.RLock()
	defer configMutex.RUnlock()

	// Normalize language code to lowercase
	if lang != "" {
		lang = strings.ToLower(lang)
	}

	// Try to get specific language config
	if config, exists := fullConfigs[lang]; exists {
		return config
	}

	// Fallback to default config (marked with default: true)
	if defaultConfig != nil {
		return defaultConfig
	}

	// Return any available config as last resort
	for _, config := range fullConfigs {
		return config
	}

	return nil
}

// GetPublicConfig returns the public configuration for a given language
func GetPublicConfig(lang string) *Config {
	configMutex.RLock()
	defer configMutex.RUnlock()

	// Normalize language code to lowercase
	if lang != "" {
		lang = strings.ToLower(lang)
	}

	// Try to get specific language config
	if config, exists := publicConfigs[lang]; exists {
		return config
	}

	// Fallback to default config's public version
	if defaultConfig != nil {
		// Find the public version of the default config
		for lang, fullConfig := range fullConfigs {
			if fullConfig == defaultConfig {
				if publicConfig, exists := publicConfigs[lang]; exists {
					return publicConfig
				}
			}
		}
	}

	// Return any available config as last resort
	for _, config := range publicConfigs {
		return config
	}

	return nil
}

// GetAvailableLanguages returns all available language codes
func GetAvailableLanguages() []string {
	configMutex.RLock()
	defer configMutex.RUnlock()

	var languages []string
	for lang := range fullConfigs {
		languages = append(languages, lang)
	}

	return languages
}

// GetDefaultLanguage returns the default language code
func GetDefaultLanguage() string {
	configMutex.RLock()
	defer configMutex.RUnlock()

	// Find the language code for the default config
	if defaultConfig != nil {
		for lang, config := range fullConfigs {
			if config == defaultConfig {
				return lang
			}
		}
	}

	// Return the first available language as fallback
	for lang := range fullConfigs {
		return lang
	}

	return ""
}

// normalizeExpiresIn converts custom time units to Go standard duration format
func normalizeExpiresIn(expiresIn string) (string, error) {
	if expiresIn == "" {
		return "", nil
	}

	// Try parsing as standard Go duration first
	if _, err := time.ParseDuration(expiresIn); err == nil {
		return expiresIn, nil
	}

	// Custom unit conversion patterns
	patterns := map[string]func(int) string{
		"d":  func(n int) string { return fmt.Sprintf("%dh", n*24) },     // days to hours
		"w":  func(n int) string { return fmt.Sprintf("%dh", n*24*7) },   // weeks to hours
		"M":  func(n int) string { return fmt.Sprintf("%dh", n*24*30) },  // months to hours (approximate)
		"y":  func(n int) string { return fmt.Sprintf("%dh", n*24*365) }, // years to hours (approximate)
		"ms": func(n int) string { return fmt.Sprintf("%dms", n) },       // milliseconds
		"s":  func(n int) string { return fmt.Sprintf("%ds", n) },        // seconds
		"m":  func(n int) string { return fmt.Sprintf("%dm", n) },        // minutes
		"h":  func(n int) string { return fmt.Sprintf("%dh", n) },        // hours
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
		return "", fmt.Errorf("unsupported time unit: %s", unit)
	}

	normalized := converter(number)

	// Validate the normalized duration
	if _, err := time.ParseDuration(normalized); err != nil {
		return "", fmt.Errorf("failed to create valid duration: %v", err)
	}

	return normalized, nil
}
