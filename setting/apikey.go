package setting

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"time"
)

const (
	apiKeyNS      = "api_keys"
	apiKeyIndexNS = "api_keys_index"
	apiKeyPrefix  = "yao-"
)

// IsAPIKeyFormat checks if the token has the API key prefix.
func IsAPIKeyFormat(token string) bool {
	return len(token) > len(apiKeyPrefix) && token[:len(apiKeyPrefix)] == apiKeyPrefix
}

// HashAPIKey returns the SHA-256 hex digest of the key.
func HashAPIKey(key string) string {
	h := sha256.Sum256([]byte(key))
	return hex.EncodeToString(h[:])
}

// LookupAPIKey looks up an API key by its plaintext value using the global Registry.
// Returns user_id, team_id, key_id, and whether it was found.
// Returns found=false if key is expired.
func LookupAPIKey(plainKey string) (userID, teamID, keyID string, found bool) {
	if Global == nil {
		return
	}

	keyHash := HashAPIKey(plainKey)
	scope := ScopeID{Scope: ScopeSystem}
	data, _ := Global.Get(scope, apiKeyIndexNS)
	if data == nil {
		return
	}

	entryRaw, ok := data[keyHash]
	if !ok {
		return
	}

	m := normalizeMap(entryRaw)
	if m == nil {
		return
	}

	userID, _ = m["user_id"].(string)
	teamID, _ = m["team_id"].(string)
	keyID, _ = m["key_id"].(string)

	if userID == "" {
		return "", "", "", false
	}

	if expired := isKeyExpired(userID, teamID, keyID); expired {
		return "", "", "", false
	}

	found = true
	return
}

// UpdateAPIKeyLastUsed updates last_used with 5-minute throttling.
func UpdateAPIKeyLastUsed(userID, teamID, keyID string) {
	if Global == nil {
		return
	}

	var scope ScopeID
	if teamID != "" {
		scope = ScopeID{Scope: ScopeTeam, TeamID: teamID}
	} else {
		scope = ScopeID{Scope: ScopeUser, UserID: userID}
	}

	data, _ := Global.Get(scope, apiKeyNS)
	if data == nil {
		return
	}

	entryRaw, ok := data[keyID]
	if !ok {
		return
	}

	m := normalizeMap(entryRaw)
	if m == nil {
		return
	}

	if lastUsed, ok := m["last_used"].(string); ok && lastUsed != "" {
		if t, err := time.Parse(time.RFC3339, lastUsed); err == nil {
			if time.Since(t) < 5*time.Minute {
				return
			}
		}
	}

	m["last_used"] = time.Now().UTC().Format(time.RFC3339)
	data[keyID] = m
	Global.Set(scope, apiKeyNS, data)
}

// isKeyExpired checks if a specific key has expired.
func isKeyExpired(userID, teamID, keyID string) bool {
	if Global == nil {
		return false
	}

	var scope ScopeID
	if teamID != "" {
		scope = ScopeID{Scope: ScopeTeam, TeamID: teamID}
	} else {
		scope = ScopeID{Scope: ScopeUser, UserID: userID}
	}

	data, _ := Global.Get(scope, apiKeyNS)
	if data == nil {
		return false
	}

	entryRaw, ok := data[keyID]
	if !ok {
		return true
	}

	m := normalizeMap(entryRaw)
	if m == nil {
		return false
	}

	expiresAt, ok := m["expires_at"].(string)
	if !ok || expiresAt == "" {
		return false
	}

	t, err := time.Parse(time.RFC3339, expiresAt)
	if err != nil {
		return false
	}
	return time.Now().After(t)
}

// normalizeMap attempts to convert an interface{} to map[string]interface{}.
func normalizeMap(v interface{}) map[string]interface{} {
	if m, ok := v.(map[string]interface{}); ok {
		return m
	}
	raw, err := json.Marshal(v)
	if err != nil {
		return nil
	}
	var m map[string]interface{}
	if err := json.Unmarshal(raw, &m); err != nil {
		return nil
	}
	return m
}
