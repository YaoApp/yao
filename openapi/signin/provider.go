package signin

import (
	"crypto/ecdsa"
	"crypto/hmac"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"math/big"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v4"
	"github.com/yaoapp/gou/application"
	"github.com/yaoapp/gou/http"
	"github.com/yaoapp/kun/log"
	oauthtypes "github.com/yaoapp/yao/openapi/oauth/types"
)

// convertToString converts various types to string, avoiding scientific notation for numbers
func (p *Provider) convertToString(value interface{}) string {
	// Handle nil values
	if value == nil {
		return ""
	}

	switch v := value.(type) {
	case string:
		return v
	case int:
		return strconv.Itoa(v)
	case int64:
		return strconv.FormatInt(v, 10)
	case float64:
		// Check if it's actually an integer value
		if v == float64(int64(v)) {
			return strconv.FormatInt(int64(v), 10)
		}
		return strconv.FormatFloat(v, 'f', -1, 64)
	case float32:
		// Check if it's actually an integer value
		if v == float32(int64(v)) {
			return strconv.FormatInt(int64(v), 10)
		}
		return strconv.FormatFloat(float64(v), 'f', -1, 32)
	case bool:
		return strconv.FormatBool(v)
	case []interface{}:
		// Handle empty arrays
		if len(v) == 0 {
			return ""
		}
		return fmt.Sprintf("%v", v)
	default:
		return fmt.Sprintf("%v", v)
	}
}

// getPresetMappings returns built-in field mappings for different providers
func getPresetMappings() map[string]map[string]string {
	return map[string]map[string]string{
		MappingGoogle: {
			"sub":            "sub",
			"id":             "sub", // fallback
			"name":           "name",
			"given_name":     "given_name",
			"family_name":    "family_name",
			"email":          "email",
			"email_verified": "email_verified",
			"picture":        "picture",
			"locale":         "locale",
		},
		MappingGitHub: {
			"id":         "sub",
			"login":      "preferred_username",
			"name":       "name",
			"email":      "email",
			"avatar_url": "picture",
			"blog":       "website",
			"html_url":   "profile",
			"location":   "address.formatted",
			"updated_at": "updated_at",
		},
		MappingMicrosoft: {
			"id":                "sub",
			"displayName":       "name",
			"givenName":         "given_name",
			"surname":           "family_name",
			"mail":              "email",
			"userPrincipalName": "preferred_username",
			"mobilePhone":       "phone_number", // Priority 1: Personal mobile phone
			"businessPhones[0]": "phone_number", // Priority 2: First business phone using array access
			"preferredLanguage": "locale",
			"officeLocation":    "address.locality",
			// jobTitle will remain in raw data as OIDC has no direct equivalent
		},
		MappingApple: {
			"sub":                "sub",
			"email":              "email",
			"email_verified":     "email_verified",
			"preferred_username": "preferred_username",
			// form_post provides name information in nested structure - using generic nested access
			"name.firstName": "given_name",
			"name.lastName":  "family_name",
			"name":           "name", // Full name object will be handled by mapping logic
		},
		MappingWeChat: {
			"openid":     "sub",
			"nickname":   "nickname",
			"headimgurl": "picture",
			"sex":        "gender",
			"country":    "address.country",
			"province":   "address.region",
			"city":       "address.locality",
		},
		MappingGeneric: {
			"sub":                "sub",
			"id":                 "sub",
			"user_id":            "sub",
			"openid":             "sub",
			"name":               "name",
			"display_name":       "name",
			"displayName":        "name",
			"full_name":          "name",
			"fullName":           "name",
			"given_name":         "given_name",
			"first_name":         "given_name",
			"firstName":          "given_name",
			"family_name":        "family_name",
			"last_name":          "family_name",
			"lastName":           "family_name",
			"surname":            "family_name",
			"middle_name":        "middle_name",
			"middleName":         "middle_name",
			"nickname":           "nickname",
			"nick":               "nickname",
			"preferred_username": "preferred_username",
			"username":           "preferred_username",
			"login":              "preferred_username",
			"screen_name":        "preferred_username",
			"user_name":          "preferred_username",
			"profile":            "profile",
			"profile_url":        "profile",
			"picture":            "picture",
			"avatar":             "picture",
			"avatar_url":         "picture",
			"profile_image_url":  "picture",
			"headimgurl":         "picture",
			"website":            "website",
			"blog":               "website",
			"url":                "website",
			"email":              "email",
			"mail":               "email",
			"email_address":      "email",
			"email_verified":     "email_verified",
			"verified_email":     "email_verified",
			"gender":             "gender",
			"sex":                "gender",
			"birthdate":          "birthdate",
			"birthday":           "birthdate",
			"birth_date":         "birthdate",
			"zoneinfo":           "zoneinfo",
			"timezone":           "zoneinfo",
			"time_zone":          "zoneinfo",
			"locale":             "locale",
			"language":           "locale",
			"lang":               "locale",
			"phone_number":       "phone_number",
			"phone":              "phone_number",
			"mobile":             "phone_number",
			"mobile_phone":       "phone_number",
			"mobilePhone":        "phone_number",
			"updated_at":         "updated_at",
			"last_modified":      "updated_at",
			"modified_at":        "updated_at",
		},
	}
}

// getFieldMapping resolves the mapping configuration and returns the actual field mapping
func (p *Provider) getFieldMapping() map[string]string {
	if p.Mapping == nil {
		// Case 3: nil/empty - use generic mapping
		return getPresetMappings()[MappingGeneric]
	}

	switch mapping := p.Mapping.(type) {
	case string:
		// Case 1: string (preset enum)
		if presetMapping, exists := getPresetMappings()[mapping]; exists {
			return presetMapping
		}
		// If preset not found, fallback to generic
		log.Warn("Unknown preset mapping '%s', falling back to generic mapping", mapping)
		return getPresetMappings()[MappingGeneric]

	case map[string]interface{}:
		// Convert map[string]interface{} to map[string]string
		result := make(map[string]string)
		for k, v := range mapping {
			if strVal, ok := v.(string); ok {
				result[k] = strVal
			}
		}
		return result

	case map[string]string:
		// Case 2: map[string]string (custom mapping)
		return mapping

	default:
		// Invalid type, fallback to generic
		log.Warn("Invalid mapping type %T, falling back to generic mapping", mapping)
		return getPresetMappings()[MappingGeneric]
	}
}

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
func (p *Provider) GetUserInfo(accessToken string, tokenType string) (*oauthtypes.OIDCUserInfo, error) {
	if accessToken == "" {
		return nil, fmt.Errorf("access_token is required")
	}

	// Set default token type if not provided
	if tokenType == "" {
		tokenType = "Bearer"
	}

	// Determine user info source (default to "endpoint")
	userInfoSource := p.UserInfoSource
	if userInfoSource == "" {
		userInfoSource = UserInfoSourceEndpoint
	}

	// Handle different user info sources
	switch userInfoSource {
	case UserInfoSourceEndpoint:
		return p.getUserInfoFromEndpoint(accessToken, tokenType)
	case UserInfoSourceIDToken:
		// For id_token source, we need a different approach since we need the token response
		return nil, fmt.Errorf("id_token source requires GetUserInfoFromTokenResponse method instead")
	case UserInfoSourceAccessToken:
		return p.getUserInfoFromAccessToken(accessToken)
	default:
		return nil, fmt.Errorf("unsupported user_info_source: %s", userInfoSource)
	}
}

// GetUserInfoFromTokenResponse gets user info from complete token response (for Apple OAuth with id_token)
func (p *Provider) GetUserInfoFromTokenResponse(tokenResponse *OAuthTokenResponse, mergeUserInfo ...string) (*oauthtypes.OIDCUserInfo, error) {
	if tokenResponse == nil {
		return nil, fmt.Errorf("token response is required")
	}

	// Determine user info source (default to "endpoint")
	userInfoSource := p.UserInfoSource
	if userInfoSource == "" {
		userInfoSource = UserInfoSourceEndpoint
	}

	// Get user info from different sources
	var userInfo *oauthtypes.OIDCUserInfo
	var err error

	switch userInfoSource {
	case UserInfoSourceEndpoint:
		userInfo, err = p.getUserInfoFromEndpoint(tokenResponse.AccessToken, tokenResponse.TokenType)
	case UserInfoSourceIDToken:
		if tokenResponse.IDToken == "" {
			return nil, fmt.Errorf("id_token not found in token response")
		}
		// Get raw claims from ID token
		rawClaims, err := p.verifyIDTokenAndGetClaims(tokenResponse.IDToken)
		if err != nil {
			return nil, fmt.Errorf("failed to verify ID token: %w", err)
		}

		// Merge cached user info into raw claims before mapping
		if len(mergeUserInfo) > 0 && mergeUserInfo[0] != "" {
			p.mergeFormPostDataIntoClaims(rawClaims, mergeUserInfo[0])
		}

		// Map the merged claims to our standard user info structure
		userInfo = p.mapUserInfoResponse(rawClaims)
	case UserInfoSourceAccessToken:
		userInfo, err = p.getUserInfoFromAccessToken(tokenResponse.AccessToken)
	default:
		return nil, fmt.Errorf("unsupported user_info_source: %s", userInfoSource)
	}

	if err != nil {
		return nil, err
	}

	return userInfo, nil
}

// mergeFormPostDataIntoClaims merges user info from form_post data into raw claims before mapping
func (p *Provider) mergeFormPostDataIntoClaims(rawClaims map[string]interface{}, cachedUserInfo string) {
	var userData map[string]interface{}
	if err := json.Unmarshal([]byte(cachedUserInfo), &userData); err != nil {
		log.Warn("Failed to parse cached user info: %v", err)
		return
	}

	// Merge cached data into raw claims, but preserve existing claims (ID Token data is more reliable)
	// The mapping logic will handle all field conversions
	for key, value := range userData {
		if _, exists := rawClaims[key]; !exists {
			rawClaims[key] = value
		}
	}
}

// getUserInfoFromEndpoint gets user info from a dedicated endpoint (default behavior)
func (p *Provider) getUserInfoFromEndpoint(accessToken string, tokenType string) (*oauthtypes.OIDCUserInfo, error) {
	if p.Endpoints == nil {
		return nil, fmt.Errorf("endpoints not found, set endpoints at least one")
	}

	if p.Endpoints.UserInfo == "" {
		return nil, fmt.Errorf("user_info endpoint not found, set user_info endpoint at least one")
	}

	// Create HTTP request with authorization header
	req := http.New(p.Endpoints.UserInfo).
		SetHeader("Authorization", fmt.Sprintf("%s %s", tokenType, accessToken)).
		SetHeader("Accept", "application/json").
		SetHeader("User-Agent", "Yao-OAuth-Client/1.0")

	// Make the GET request
	resp := req.Get()
	if resp == nil {
		return nil, fmt.Errorf("failed to make user info request: no response")
	}

	// Check for HTTP errors
	if resp.Code != 200 {
		if resp.Data != nil {

			// === Parse the response data ===
			if data, ok := resp.Data.(map[string]interface{}); ok {
				// Handle standard OAuth error format
				if err, ok := data["error_description"]; ok {
					return nil, fmt.Errorf("%v", err)
				}
				if err, ok := data["error"]; ok {
					// Handle Microsoft Graph nested error format
					if errorObj, isMap := err.(map[string]interface{}); isMap {
						if code, hasCode := errorObj["code"]; hasCode {
							if message, hasMessage := errorObj["message"]; hasMessage && message != "" {
								return nil, fmt.Errorf("Microsoft Graph error %v: %v", code, message)
							}
							return nil, fmt.Errorf("Microsoft Graph error: %v", code)
						}
					}
					return nil, fmt.Errorf("%v", err)
				}
			}
		}

		if resp.Message != "" {
			return nil, fmt.Errorf("user info request failed with status %d: %s", resp.Code, resp.Message)
		}
		return nil, fmt.Errorf("user info request failed with status %d", resp.Code)
	}

	// Parse the response data
	var rawData map[string]interface{}
	switch data := resp.Data.(type) {
	case map[string]interface{}:
		rawData = data
	case []byte:
		if err := json.Unmarshal(data, &rawData); err != nil {
			return nil, fmt.Errorf("failed to parse user info response from bytes: %w", err)
		}
	case string:
		if err := json.Unmarshal([]byte(data), &rawData); err != nil {
			return nil, fmt.Errorf("failed to parse user info response from string: %w", err)
		}
	default:
		return nil, fmt.Errorf("unexpected response data type: %T", data)
	}

	// Map the raw response to our standard structure
	userInfo := p.mapUserInfoResponse(rawData)

	return userInfo, nil
}

// verifyIDTokenAndGetClaims verifies ID token signature and returns raw claims for user info mapping
func (p *Provider) verifyIDTokenAndGetClaims(idToken string) (map[string]interface{}, error) {
	// Parse token to get header for key ID
	token, err := jwt.Parse(idToken, func(token *jwt.Token) (interface{}, error) {
		// Verify signing method
		if _, ok := token.Method.(*jwt.SigningMethodRSA); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}

		// Get key ID from token header
		kid, ok := token.Header["kid"].(string)
		if !ok {
			return nil, fmt.Errorf("missing key ID in token header")
		}

		// Get public key from JWKS endpoint for verification
		publicKey, err := p.getJWKSPublicKey(kid)
		if err != nil {
			return nil, fmt.Errorf("failed to get JWKS public key: %w", err)
		}

		return publicKey, nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to parse/verify JWT: %w", err)
	}

	if !token.Valid {
		return nil, fmt.Errorf("invalid JWT token")
	}

	// Extract claims
	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return nil, fmt.Errorf("failed to extract JWT claims")
	}

	// Basic validation
	if aud, ok := claims["aud"].(string); ok && aud != p.ClientID {
		return nil, fmt.Errorf("invalid audience: %s", aud)
	}
	if exp, ok := claims["exp"].(float64); ok && time.Now().Unix() > int64(exp) {
		return nil, fmt.Errorf("token expired")
	}

	// Convert jwt.MapClaims to map[string]interface{}
	rawClaims := make(map[string]interface{})
	for key, value := range claims {
		rawClaims[key] = value
	}

	return rawClaims, nil
}

// getJWKSPublicKey fetches public key from provider's JWKS endpoint
func (p *Provider) getJWKSPublicKey(keyID string) (interface{}, error) {
	// Check if JWKS endpoint is configured
	if p.Endpoints == nil || p.Endpoints.JWKS == "" {
		return nil, fmt.Errorf("JWKS endpoint not configured")
	}

	jwksURL := p.Endpoints.JWKS

	// Make HTTP request to get JWKS
	req := http.New(jwksURL).
		SetHeader("Accept", "application/json").
		SetHeader("User-Agent", "Yao-OAuth-Client/1.0")

	resp := req.Get()
	if resp == nil {
		return nil, fmt.Errorf("failed to fetch JWKS from %s: no response", jwksURL)
	}

	if resp.Code != 200 {
		return nil, fmt.Errorf("failed to fetch JWKS from %s: status %d", jwksURL, resp.Code)
	}

	// Parse JWKS response
	var jwks struct {
		Keys []struct {
			Kid string `json:"kid"`
			Kty string `json:"kty"`
			Use string `json:"use"`
			Alg string `json:"alg"`
			N   string `json:"n"`
			E   string `json:"e"`
		} `json:"keys"`
	}

	// Handle different response data types
	switch data := resp.Data.(type) {
	case map[string]interface{}:
		jsonBytes, err := json.Marshal(data)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal JWKS response: %w", err)
		}
		if err := json.Unmarshal(jsonBytes, &jwks); err != nil {
			return nil, fmt.Errorf("failed to parse JWKS response: %w", err)
		}
	case []byte:
		if err := json.Unmarshal(data, &jwks); err != nil {
			return nil, fmt.Errorf("failed to parse JWKS response: %w", err)
		}
	case string:
		if err := json.Unmarshal([]byte(data), &jwks); err != nil {
			return nil, fmt.Errorf("failed to parse JWKS response: %w", err)
		}
	default:
		return nil, fmt.Errorf("unexpected JWKS response data type: %T", data)
	}

	// Find the key with matching kid
	for _, key := range jwks.Keys {
		if key.Kid == keyID && key.Kty == "RSA" {
			// Decode RSA public key components
			nBytes, err := base64.RawURLEncoding.DecodeString(key.N)
			if err != nil {
				return nil, fmt.Errorf("failed to decode RSA modulus: %w", err)
			}
			eBytes, err := base64.RawURLEncoding.DecodeString(key.E)
			if err != nil {
				return nil, fmt.Errorf("failed to decode RSA exponent: %w", err)
			}

			// Convert exponent bytes to int
			var eInt int64
			for _, b := range eBytes {
				eInt = eInt<<8 + int64(b)
			}

			// Create RSA public key
			rsaKey := &rsa.PublicKey{
				N: big.NewInt(0).SetBytes(nBytes),
				E: int(eInt),
			}

			return rsaKey, nil
		}
	}

	return nil, fmt.Errorf("public key not found for key ID: %s", keyID)
}

// getUserInfoFromAccessToken gets user info from access token response
func (p *Provider) getUserInfoFromAccessToken(accessToken string) (*oauthtypes.OIDCUserInfo, error) {
	// This is a placeholder implementation
	// In a real implementation, this would parse structured data from the access token response
	// or decode a JWT access token if the provider uses JWT access tokens
	return &oauthtypes.OIDCUserInfo{
		Sub: "access_token_user", // Placeholder
		Raw: map[string]interface{}{
			"note":         "User info extracted from access token",
			"access_token": accessToken,
		},
	}, nil
}

// mapUserInfoResponse maps raw OAuth user info response to our standard structure
func (p *Provider) mapUserInfoResponse(rawData map[string]interface{}) *oauthtypes.OIDCUserInfo {
	userInfo := &oauthtypes.OIDCUserInfo{
		Raw: rawData, // Keep raw data for debugging/custom processing
	}

	// Get the appropriate field mapping (preset, custom, or generic)
	fieldMapping := p.getFieldMapping()

	// Apply field mappings with support for nested field access
	for sourceField, targetField := range fieldMapping {
		var value interface{}
		var exists bool

		// Check if it's a nested field (contains dots or array notation)
		if strings.Contains(sourceField, ".") || strings.Contains(sourceField, "[") {
			value = p.getNestedValue(rawData, sourceField)
			exists = (value != nil)
		} else {
			// Simple field access
			value, exists = rawData[sourceField]
		}

		if exists {
			p.setUserInfoField(userInfo, targetField, value)
		}
	}

	// Post-processing: set fallback values
	p.applyFallbackValues(userInfo, rawData)

	return userInfo
}

// getNestedValue retrieves a value from nested object/array using dot notation and array indexing
// Supports: "name.firstName", "address.country", "businessPhones[0]", "roles[1].name"
func (p *Provider) getNestedValue(data map[string]interface{}, path string) interface{} {
	// Split path by dots
	parts := strings.Split(path, ".")
	current := interface{}(data)

	for _, part := range parts {
		// Handle array indexing: fieldName[index]
		if strings.Contains(part, "[") && strings.HasSuffix(part, "]") {
			// Extract field name and index
			openBracket := strings.Index(part, "[")
			fieldName := part[:openBracket]
			indexStr := part[openBracket+1 : len(part)-1]

			// Get the field first
			if currentMap, ok := current.(map[string]interface{}); ok {
				if field, exists := currentMap[fieldName]; exists {
					current = field
				} else {
					return nil
				}
			} else {
				return nil
			}

			// Handle array access
			if currentArray, ok := current.([]interface{}); ok {
				if index, err := strconv.Atoi(indexStr); err == nil && index >= 0 && index < len(currentArray) {
					current = currentArray[index]
				} else {
					return nil
				}
			} else {
				return nil
			}
		} else {
			// Handle simple field access
			if currentMap, ok := current.(map[string]interface{}); ok {
				if field, exists := currentMap[part]; exists {
					current = field
				} else {
					return nil
				}
			} else {
				return nil
			}
		}
	}

	return current
}

// setUserInfoField sets a field in the user info structure
func (p *Provider) setUserInfoField(userInfo *oauthtypes.OIDCUserInfo, fieldName string, value interface{}) {
	// Handle nested address fields
	if strings.HasPrefix(fieldName, "address.") {
		stringValue := p.convertToString(value)
		// Skip empty values
		if stringValue == "" {
			return
		}

		if userInfo.Address == nil {
			userInfo.Address = &oauthtypes.OIDCAddress{}
		}

		addressField := strings.TrimPrefix(fieldName, "address.")

		switch addressField {
		case "formatted":
			userInfo.Address.Formatted = stringValue
		case "street_address":
			userInfo.Address.StreetAddress = stringValue
		case "locality":
			userInfo.Address.Locality = stringValue
		case "region":
			userInfo.Address.Region = stringValue
		case "postal_code":
			userInfo.Address.PostalCode = stringValue
		case "country":
			userInfo.Address.Country = stringValue
		}
		return
	}

	stringValue := p.convertToString(value)

	// Skip empty values for most fields
	if stringValue == "" && fieldName != "phone_number" {
		return
	}

	switch fieldName {
	// OIDC Standard Claims
	case "sub":
		userInfo.Sub = stringValue
	case "name":
		// Handle name as object (e.g., Apple form_post: {"firstName": "John", "lastName": "Doe"})
		if nameObj, ok := value.(map[string]interface{}); ok {
			var nameParts []string
			if firstName, exists := nameObj["firstName"]; exists {
				if firstNameStr := p.convertToString(firstName); firstNameStr != "" {
					nameParts = append(nameParts, firstNameStr)
					if userInfo.GivenName == "" {
						userInfo.GivenName = firstNameStr
					}
				}
			}
			if lastName, exists := nameObj["lastName"]; exists {
				if lastNameStr := p.convertToString(lastName); lastNameStr != "" {
					nameParts = append(nameParts, lastNameStr)
					if userInfo.FamilyName == "" {
						userInfo.FamilyName = lastNameStr
					}
				}
			}
			if len(nameParts) > 0 {
				userInfo.Name = strings.Join(nameParts, " ")
			}
		} else {
			// Handle name as string
			userInfo.Name = stringValue
		}
	case "given_name":
		userInfo.GivenName = stringValue
	case "family_name":
		userInfo.FamilyName = stringValue
	case "middle_name":
		userInfo.MiddleName = stringValue
	case "nickname":
		userInfo.Nickname = stringValue
	case "preferred_username":
		userInfo.PreferredUsername = stringValue
	case "profile":
		userInfo.Profile = stringValue
	case "picture":
		userInfo.Picture = stringValue
	case "website":
		userInfo.Website = stringValue
	case "email":
		userInfo.Email = stringValue
	case "email_verified":
		if boolValue, ok := value.(bool); ok {
			userInfo.EmailVerified = &boolValue
		}
	case "gender":
		// Handle special gender conversion for WeChat
		if floatValue, ok := value.(float64); ok {
			switch int(floatValue) {
			case 1:
				userInfo.Gender = "male"
			case 2:
				userInfo.Gender = "female"
			default:
				userInfo.Gender = "unknown"
			}
		} else {
			userInfo.Gender = stringValue
		}
	case "birthdate":
		userInfo.Birthdate = stringValue
	case "zoneinfo":
		userInfo.Zoneinfo = stringValue
	case "locale":
		userInfo.Locale = stringValue
	case "phone_number":
		// Only set if we don't already have a phone number
		if userInfo.PhoneNumber != "" {
			return
		}

		// Handle array type for Microsoft businessPhones
		if phoneArray, ok := value.([]interface{}); ok && len(phoneArray) > 0 {
			// Take the first non-empty phone number from the array
			for _, phone := range phoneArray {
				if phoneStr := p.convertToString(phone); phoneStr != "" {
					userInfo.PhoneNumber = phoneStr
					break
				}
			}
		} else {
			// Handle single phone number (mobilePhone)
			if stringValue != "" {
				userInfo.PhoneNumber = stringValue
			}
		}
	case "phone_number_verified":
		if boolValue, ok := value.(bool); ok {
			userInfo.PhoneNumberVerified = &boolValue
		}
	case "updated_at":
		if intValue, ok := value.(int64); ok {
			userInfo.UpdatedAt = &intValue
		} else if stringValue, ok := value.(string); ok {
			// Handle ISO 8601 time strings (e.g., from GitHub)
			if parsedTime, err := time.Parse(time.RFC3339, stringValue); err == nil {
				timestamp := parsedTime.Unix()
				userInfo.UpdatedAt = &timestamp
			}
		}
	}
}

// applyFallbackValues applies fallback values and data cleanup
func (p *Provider) applyFallbackValues(userInfo *oauthtypes.OIDCUserInfo, rawData map[string]interface{}) {
	// OIDC Standard: If no name but have given_name/family_name, combine them
	if userInfo.Name == "" && (userInfo.GivenName != "" || userInfo.FamilyName != "") {
		parts := []string{}
		if userInfo.GivenName != "" {
			parts = append(parts, userInfo.GivenName)
		}
		if userInfo.MiddleName != "" {
			parts = append(parts, userInfo.MiddleName)
		}
		if userInfo.FamilyName != "" {
			parts = append(parts, userInfo.FamilyName)
		}
		userInfo.Name = strings.Join(parts, " ")
	}

	// Set preferred_username fallbacks
	if userInfo.PreferredUsername == "" && userInfo.Email != "" {
		if atIndex := strings.Index(userInfo.Email, "@"); atIndex > 0 {
			userInfo.PreferredUsername = userInfo.Email[:atIndex]
		}
	}

	// OIDC requires Sub to be always set
	if userInfo.Sub == "" {
		log.Error("Subject identifier (sub) not found in OAuth response for provider '%s'", p.ID)
	}
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

// GetProvider gets the provider by ID from the global providers map
func GetProvider(providerID string) (*Provider, error) {
	// Get provider from global providers map
	provider, exists := providers[providerID]
	if !exists {
		return nil, fmt.Errorf("OAuth provider '%s' not found", providerID)
	}

	return provider, nil
}
