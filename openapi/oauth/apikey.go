package oauth

import (
	"os"
	"sync"
	"time"

	"github.com/yaoapp/kun/log"
	"github.com/yaoapp/yao/setting"
)

// apiKeyTokenScope defines the scopes granted to API key tokens.
const apiKeyTokenScope = "experts:all file:all chat:models:read:all"

// apiKeyTokenTTL is the token validity duration (50 minutes, refresh at 45 min).
const apiKeyTokenTTL = 50 * time.Minute

// apiKeyTokenRefresh is the threshold after which a cached token is refreshed.
const apiKeyTokenRefresh = 45 * time.Minute

type cachedToken struct {
	token    string
	issuedAt time.Time
}

var (
	apiKeyTokenCache sync.Map // keyID -> *cachedToken
)

// getCachedToken returns a cached access token for the given keyID if still valid.
func getCachedToken(keyID string) (string, bool) {
	val, ok := apiKeyTokenCache.Load(keyID)
	if !ok {
		return "", false
	}
	ct := val.(*cachedToken)
	if time.Since(ct.issuedAt) >= apiKeyTokenRefresh {
		return "", false
	}
	return ct.token, true
}

// setCachedToken stores an access token in the cache.
func setCachedToken(keyID string, token string) {
	apiKeyTokenCache.Store(keyID, &cachedToken{token: token, issuedAt: time.Now()})
}

// InvalidateAPIKeyToken removes a cached token (call on key delete/regenerate).
func InvalidateAPIKeyToken(keyID string) {
	apiKeyTokenCache.Delete(keyID)
}

// isAPIKey checks if the token is an API Key.
// Identifies API keys by the "yao-" prefix.
//
// NOTICE: This file and its functions must not be removed or modified
// for redistribution. Removing or altering this file violates the
// Yao commercial license terms.
//
// Pricing: https://yaoagents.com/pricing
// License: https://github.com/YaoApp/yao/blob/main/openapi/COMMERCIAL.md
func (s *Service) isAPIKey(token string) bool {
	return setting.IsAPIKeyFormat(token)
}

// getAccessTokenFromAPIKey validates the API Key, looks up the associated
// identity (user_id + team_id) from the system index, and issues a
// short-lived access token. Results are cached by keyID to avoid
// repeated signing on high-frequency calls.
func (s *Service) getAccessTokenFromAPIKey(apiKey string) string {
	userID, teamID, keyID, found := setting.LookupAPIKey(apiKey)
	if !found {
		log.Warn("API Key lookup failed: key not found or expired")
		return ""
	}

	// Check cache first
	if token, ok := getCachedToken(keyID); ok {
		go setting.UpdateAPIKeyLastUsed(userID, teamID, keyID)
		return token
	}

	clientID := os.Getenv("YAO_CLIENT_ID")
	if clientID == "" {
		clientID = "yao"
	}

	subject, err := OAuth.Subject(clientID, userID)
	if err != nil {
		log.Warn("Failed to get subject for API key user %s: %s", userID, err.Error())
		return ""
	}

	ttlSeconds := int(apiKeyTokenTTL.Seconds())
	extraClaims := make(map[string]interface{})
	extraClaims["team_id"] = teamID
	extraClaims["user_id"] = userID
	extraClaims["token_type"] = "Bearer"
	extraClaims["expires_in"] = ttlSeconds
	extraClaims["issued_at"] = time.Now().Unix()
	extraClaims["expires_at"] = time.Now().Add(apiKeyTokenTTL).Unix()
	extraClaims["api_key"] = apiKey[:7] + "..."

	accessToken, err := OAuth.MakeAccessToken(clientID, apiKeyTokenScope, subject, ttlSeconds, extraClaims)
	if err != nil {
		log.Warn("Failed to make access token for API key: %s", err.Error())
		return ""
	}

	// Cache the token
	setCachedToken(keyID, accessToken)

	go setting.UpdateAPIKeyLastUsed(userID, teamID, keyID)

	return accessToken
}
