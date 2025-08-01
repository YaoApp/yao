package signin

import (
	"crypto/ecdsa"
	"crypto/hmac"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v4"
	"github.com/yaoapp/gou/application"
	"github.com/yaoapp/gou/http"
	"github.com/yaoapp/kun/log"
)

// GetClientSecret gets the client secret for the provider
func (p *Provider) GetClientSecret() (string, error) {
	if p.ClientSecret != "" {
		return p.ClientSecret, nil
	}

	if p.ClientSecretGenerator == nil {
		return "", fmt.Errorf("client secret generator not found, set client_secret or client_secret_generator at least one")
	}

	// Generate the client secret using the configured generator
	return p.GenerateClientSecret()
}

// GetUserInfo gets the user information from the provider
func (p *Provider) GetUserInfo(accessToken string) (*OAuthUserInfoResponse, error) {
	if p.Endpoints == nil {
		return nil, fmt.Errorf("endpoints not found, set endpoints at least one")
	}

	return nil, nil
}

// GenerateClientSecret generates client secret based on the configured generator type
func (p *Provider) GenerateClientSecret() (string, error) {
	if p.ClientSecretGenerator == nil {
		return "", fmt.Errorf("client secret generator not configured")
	}

	switch p.ClientSecretGenerator.Type {
	case "JWT_ES256", "JWT_APPLE": // Apple JWT is the same as JWT_ES256
		return p.generateJWTES256()
	case "BASIC_CONCAT":
		return p.generateBasicConcat()
	case "HMAC_SHA256":
		return p.generateHMACSignature()
	default:
		return "", fmt.Errorf("unsupported client secret generator type: %s", p.ClientSecretGenerator.Type)
	}
}

// generateJWTES256 generates JWT client secret using ES256 algorithm
func (p *Provider) generateJWTES256() (string, error) {
	gen := p.ClientSecretGenerator

	// Validate required fields
	if gen.PrivateKey == "" {
		return "", fmt.Errorf("private_key is required for JWT ES256 generation")
	}

	if gen.Header == nil {
		return "", fmt.Errorf("header is required for JWT ES256 generation")
	}

	if gen.Payload == nil {
		return "", fmt.Errorf("payload is required for JWT ES256 generation")
	}

	// Read private key
	privateKey, err := p.loadPrivateKey(gen.PrivateKey)
	if err != nil {
		return "", fmt.Errorf("failed to load private key: %w", err)
	}

	// Parse expiration time (already normalized during config loading)
	expiresIn := time.Hour * 24 * 90 // Default 90 days
	if gen.ExpiresIn != "" {
		duration, err := time.ParseDuration(gen.ExpiresIn)
		if err != nil {
			// This should not happen since it's normalized during config loading
			log.Error("Failed to parse normalized expires_in '%s': %v", gen.ExpiresIn, err)
			// Use default duration
		} else {
			expiresIn = duration
		}
	}

	// Create JWT token
	now := time.Now()
	token := jwt.New(jwt.SigningMethodES256)

	// Set header claims
	for key, value := range gen.Header {
		token.Header[key] = value
	}

	// Set payload claims
	claims := token.Claims.(jwt.MapClaims)
	for key, value := range gen.Payload {
		claims[key] = value
	}

	// Set standard claims
	claims["iat"] = now.Unix()
	claims["exp"] = now.Add(expiresIn).Unix()

	// Sign the token
	tokenString, err := token.SignedString(privateKey)
	if err != nil {
		return "", fmt.Errorf("failed to sign JWT: %w", err)
	}

	return tokenString, nil
}

// loadPrivateKey loads and parses the ES256 private key
func (p *Provider) loadPrivateKey(keyPath string) (*ecdsa.PrivateKey, error) {
	var keyData []byte
	var err error

	// Check if keyPath is absolute or relative to openapi/certs
	if filepath.IsAbs(keyPath) {
		keyData, err = os.ReadFile(keyPath)
	} else {
		// Try relative to openapi/certs directory
		certPath := filepath.Join("openapi", "certs", keyPath)
		keyData, err = application.App.Read(certPath)
	}

	if err != nil {
		log.Error("failed to read private key file: %v", err)
		return nil, fmt.Errorf("failed to read private key file: %w", err)
	}

	// Parse PEM block
	block, _ := pem.Decode(keyData)
	if block == nil {
		return nil, fmt.Errorf("failed to decode PEM block")
	}

	// Parse private key
	switch block.Type {
	case "EC PRIVATE KEY":
		return x509.ParseECPrivateKey(block.Bytes)
	case "PRIVATE KEY":
		key, err := x509.ParsePKCS8PrivateKey(block.Bytes)
		if err != nil {
			return nil, err
		}
		ecKey, ok := key.(*ecdsa.PrivateKey)
		if !ok {
			return nil, fmt.Errorf("not an ECDSA private key")
		}
		return ecKey, nil
	default:
		return nil, fmt.Errorf("unsupported private key type: %s", block.Type)
	}
}

// generateBasicConcat generates client secret by concatenating client_id and other values
func (p *Provider) generateBasicConcat() (string, error) {
	gen := p.ClientSecretGenerator

	// Default pattern: client_id:timestamp
	parts := []string{p.ClientID}

	// Add custom parts from payload
	if gen.Payload != nil {
		for key, value := range gen.Payload {
			if key == "separator" {
				continue // Skip separator key
			}
			parts = append(parts, fmt.Sprintf("%v", value))
		}
	} else {
		// Add timestamp if no custom payload
		parts = append(parts, fmt.Sprintf("%d", time.Now().Unix()))
	}

	// Get separator from payload, default to ":"
	separator := ":"
	if gen.Payload != nil {
		if sep, ok := gen.Payload["separator"].(string); ok {
			separator = sep
		}
	}

	return strings.Join(parts, separator), nil
}

// generateHMACSignature generates client secret using HMAC-SHA256 signature
func (p *Provider) generateHMACSignature() (string, error) {
	gen := p.ClientSecretGenerator

	// Get the secret key for HMAC
	secretKey := ""
	if gen.PrivateKey != "" {
		secretKey = gen.PrivateKey
	} else if gen.Payload != nil {
		if key, ok := gen.Payload["secret_key"].(string); ok {
			secretKey = key
		}
	}

	if secretKey == "" {
		return "", fmt.Errorf("secret_key is required for HMAC_SHA256 generation")
	}

	// Build message to sign
	message := p.ClientID
	if gen.Payload != nil {
		if msg, ok := gen.Payload["message"].(string); ok {
			message = msg
		} else if msg, ok := gen.Payload["data"].(string); ok {
			message = msg
		}
	}

	// Add timestamp if configured
	if gen.Payload != nil {
		if addTimestamp, ok := gen.Payload["add_timestamp"].(bool); ok && addTimestamp {
			message += fmt.Sprintf(":%d", time.Now().Unix())
		}
	}

	// Create HMAC signature
	h := hmac.New(sha256.New, []byte(secretKey))
	h.Write([]byte(message))
	signature := h.Sum(nil)

	// Return as hex or base64 based on configuration
	encoding := "hex" // default
	if gen.Payload != nil {
		if enc, ok := gen.Payload["encoding"].(string); ok {
			encoding = enc
		}
	}

	switch encoding {
	case "base64":
		return base64.StdEncoding.EncodeToString(signature), nil
	case "hex":
		return hex.EncodeToString(signature), nil
	default:
		return hex.EncodeToString(signature), nil
	}
}

// AccessToken gets the access token for the provider using OAuth 2.0 authorization code flow
func (p *Provider) AccessToken(code, redirectURI string) (*OAuthTokenResponse, error) {
	if code == "" {
		return nil, fmt.Errorf("authorization code is required")
	}

	// Get the access token endpoint
	if p.Endpoints == nil {
		return nil, fmt.Errorf("endpoints not found, set endpoints at least one")
	}

	if p.Endpoints.Token == "" {
		return nil, fmt.Errorf("token endpoint not found, set token endpoint at least one")
	}

	// Get client secret (handles both ClientSecret and ClientSecretGenerator cases)
	secret, err := p.GetClientSecret()
	if err != nil {
		return nil, fmt.Errorf("failed to get client secret: %w", err)
	}

	// Prepare the request parameters according to OAuth 2.0 spec
	params := map[string]string{
		"grant_type":    "authorization_code",
		"code":          code,
		"client_id":     p.ClientID,
		"client_secret": secret,
		"redirect_uri":  redirectURI,
	}

	// Create HTTP request using gou/http package (with DNS optimization)
	req := http.New(p.Endpoints.Token).
		SetHeader("Content-Type", "application/x-www-form-urlencoded").
		SetHeader("Accept", "application/json").
		SetHeader("User-Agent", "Yao-OAuth-Client/1.0")

	// Make the POST request
	resp := req.Post(params)
	if resp == nil {
		return nil, fmt.Errorf("failed to make token request: no response")
	}

	// Check for HTTP errors
	if resp.Code != 200 {
		if resp.Data != nil {
			if data, ok := resp.Data.(map[string]interface{}); ok {
				if err, ok := data["error_description"]; ok {
					return nil, fmt.Errorf("%v", err)
				}
				if err, ok := data["error"]; ok {
					return nil, fmt.Errorf("%v", err)
				}
			}
		}

		if resp.Message != "" {
			return nil, fmt.Errorf("token request failed with status %d: %s", resp.Code, resp.Message)
		}

		return nil, fmt.Errorf("token request failed with status %d", resp.Code)
	}

	// Parse the JSON response
	var tokenResponse OAuthTokenResponse

	// Handle the response data - it could be already parsed JSON or raw bytes
	switch data := resp.Data.(type) {
	case map[string]interface{}:
		// Already parsed JSON, convert to our struct
		jsonBytes, err := json.Marshal(data)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal response data: %w", err)
		}
		if err := json.Unmarshal(jsonBytes, &tokenResponse); err != nil {
			return nil, fmt.Errorf("failed to parse token response from parsed JSON: %w", err)
		}
	case []byte:
		// Raw bytes, parse as JSON
		if err := json.Unmarshal(data, &tokenResponse); err != nil {
			return nil, fmt.Errorf("failed to parse token response from bytes: %w", err)
		}
	case string:
		// String response, parse as JSON
		if err := json.Unmarshal([]byte(data), &tokenResponse); err != nil {
			return nil, fmt.Errorf("failed to parse token response from string: %w", err)
		}
	default:
		return nil, fmt.Errorf("unexpected response data type: %T", data)
	}

	// Check for OAuth error response
	if tokenResponse.Error != "" {
		errorMsg := tokenResponse.Error
		if tokenResponse.ErrorDesc != "" {
			errorMsg += ": " + tokenResponse.ErrorDesc
		}
		return nil, fmt.Errorf("OAuth error: %s", errorMsg)
	}

	// Validate that we got an access token
	if tokenResponse.AccessToken == "" {
		return nil, fmt.Errorf("no access token in response")
	}

	return &tokenResponse, nil
}

// GetProvider gets the provider by ID
func GetProvider(locale, providerID string) (*Provider, error) {

	// Get the signin configuration
	config := GetFullConfig(locale)
	if config == nil {
		return nil, fmt.Errorf("no signin configuration found")
	}

	// Find the provider
	var provider *Provider
	if config.ThirdParty != nil && config.ThirdParty.Providers != nil {
		for _, p := range config.ThirdParty.Providers {
			if p.ID == providerID {
				provider = p
				break
			}
		}
	}

	if provider == nil {
		return nil, fmt.Errorf("OAuth provider '%s' not found", providerID)
	}

	return provider, nil
}
