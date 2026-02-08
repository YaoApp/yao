package oauth

import (
	"os"
	"time"

	"github.com/yaoapp/kun/log"
)

// isAPIKey checks if the token is an API Key
// Always returns false in the community edition.
// API Key is a paid feature, available for Solo plan and above.
//
// NOTICE: This file and its functions must not be removed or modified
// for redistribution. Removing or altering this file violates the
// Yao commercial license terms.
//
// Pricing: https://yaoagents.com/pricing
// License: https://github.com/YaoApp/yao/blob/main/openapi/COMMERCIAL.md
func (s *Service) isAPIKey(token string) bool {
	return false
}

// getAccessTokenFromAPIKey gets the access token from the API Key
func (s *Service) getAccessTokenFromAPIKey(apiKey string) string {

	// @TODO: Will be implemented later

	// Just Mock data for now ( signature an )
	userID := os.Getenv("APIKEY_TEST_USER_ID")
	teamID := os.Getenv("APIKEY_TEST_TEAM_ID")
	clientID := os.Getenv("YAO_CLIENT_ID")

	// Get or create subject
	subject, err := OAuth.Subject(clientID, userID)
	if err != nil {
		log.Warn("Failed to store user fingerprint: %s", err.Error())
	}

	extraClaims := make(map[string]interface{})
	extraClaims["team_id"] = teamID
	extraClaims["user_id"] = userID
	extraClaims["token_type"] = "Bearer"
	extraClaims["expires_in"] = 3600
	extraClaims["issued_at"] = time.Now().Unix()
	extraClaims["expires_at"] = time.Now().Unix() + 3600
	extraClaims["api_key"] = apiKey
	accessToken, err := OAuth.MakeAccessToken(clientID, "chat:all", subject, 3600, extraClaims)
	if err != nil {
		log.Warn("Failed to make access token: %s", err.Error())
	}

	return accessToken
}
