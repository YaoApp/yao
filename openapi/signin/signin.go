package signin

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/yaoapp/gou/application"
	"github.com/yaoapp/yao/config"
)

// Global variables to store loaded configurations
var (
	// Full configurations with sensitive data (for backend use)
	fullConfigs = make(map[string]*Config)
	// Public configurations without sensitive data (for frontend use)
	publicConfigs = make(map[string]*Config)
	// Default language code
	defaultLang = ""
	// Mutex for thread safety
	configMutex sync.RWMutex
)

// Load loads all signin configurations from the openapi directory
func Load(appConfig config.Config) error {
	configMutex.Lock()
	defer configMutex.Unlock()

	// Clear existing configurations
	fullConfigs = make(map[string]*Config)
	publicConfigs = make(map[string]*Config)
	defaultLang = ""

	// Find all signin configuration files
	files, err := findSigninFiles()
	if err != nil {
		return fmt.Errorf("failed to find signin files: %v", err)
	}

	// If no signin files found, that's not necessarily an error
	// Some applications might not have signin configurations
	if len(files) == 0 {
		return nil
	}

	// Load each configuration file
	for _, file := range files {
		lang := extractLanguageFromFilename(file)

		configPath := filepath.Join("openapi", file)
		configRaw, err := application.App.Read(configPath)
		if err != nil {
			return fmt.Errorf("failed to read signin config %s: %v", file, err)
		}

		// Parse the configuration
		var signinConfig Config
		err = application.Parse(configPath, configRaw, &signinConfig)
		if err != nil {
			return fmt.Errorf("failed to parse signin config %s: %v", file, err)
		}

		// Process ENV variables in full config
		fullConfig := signinConfig
		processENVVariables(&fullConfig, appConfig.Root)

		// Create public config (without sensitive data)
		publicConfig := createPublicConfig(&fullConfig)

		// Store configurations
		fullConfigs[lang] = &fullConfig
		publicConfigs[lang] = &publicConfig

		// Set default language
		if defaultLang == "" || lang == "en" || file == "signin.yao" {
			defaultLang = lang
		}
	}

	return nil
}

// findSigninFiles finds all signin configuration files in the openapi directory
func findSigninFiles() ([]string, error) {
	var files []string
	signinFilePattern := regexp.MustCompile(`^signin(\.[a-z]{2}(-[a-z]{2})?)?\.yao$`)

	// Use Walk to find all signin files in the openapi directory
	err := application.App.Walk("openapi", func(root, filename string, isdir bool) error {
		if isdir {
			return nil
		}

		baseName := filepath.Base(filename)
		if signinFilePattern.MatchString(baseName) {
			files = append(files, baseName)
		}

		return nil
	}, "*.yao")

	if err != nil {
		return nil, err
	}

	return files, nil
}

// extractLanguageFromFilename extracts language code from filename
func extractLanguageFromFilename(filename string) string {
	// signin.yao -> ""
	// signin.en.yao -> "en"
	// signin.zh-cn.yao -> "zh-cn"

	if filename == "signin.yao" {
		return ""
	}

	parts := strings.Split(filename, ".")
	if len(parts) >= 3 {
		return parts[1]
	}

	return ""
}

// processENVVariables processes environment variables in the configuration
func processENVVariables(config *Config, rootPath string) {
	var missingEnvVars []string

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

	// Process third party providers
	if config.ThirdParty != nil && config.ThirdParty.Providers != nil {
		for _, provider := range config.ThirdParty.Providers {
			// Check ClientID
			if strings.HasPrefix(provider.ClientID, "$ENV.") {
				envVar := strings.TrimPrefix(provider.ClientID, "$ENV.")
				if _, exists := os.LookupEnv(envVar); !exists {
					missingEnvVars = append(missingEnvVars, envVar)
				}
			}
			provider.ClientID = replaceENVVar(provider.ClientID)

			// Check ClientSecret
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
						log.Printf("Warning: Invalid expires_in format '%s' for provider '%s': %v",
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
		}
	}

	// Log warning for missing environment variables
	if len(missingEnvVars) > 0 {
		log.Printf("Warning: The following environment variables are not set and may cause configuration issues: %v", missingEnvVars)
		log.Printf("Please set these environment variables to avoid exposing placeholder values in configuration")
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

		// Deep copy Register configuration
		if fullConfig.ThirdParty.Register != nil {
			publicConfig.ThirdParty.Register = &RegisterConfig{
				Auto: fullConfig.ThirdParty.Register.Auto,
				Role: fullConfig.ThirdParty.Register.Role,
			}
		}

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
					// Remove sensitive fields: ClientID, ClientSecret, ClientSecretGenerator, Scopes, Endpoints, Mapping
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

	// Fallback to default language
	if defaultLang != "" {
		if config, exists := fullConfigs[defaultLang]; exists {
			return config
		}
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

	// Fallback to default language
	if defaultLang != "" {
		if config, exists := publicConfigs[defaultLang]; exists {
			return config
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
		if lang != "" {
			languages = append(languages, lang)
		}
	}

	// Add default language if it exists and is empty string
	if defaultLang == "" && len(fullConfigs) > 0 {
		languages = append(languages, "default")
	}

	return languages
}

// GetDefaultLanguage returns the default language code
func GetDefaultLanguage() string {
	configMutex.RLock()
	defer configMutex.RUnlock()

	if defaultLang == "" {
		return "default"
	}
	return defaultLang
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
