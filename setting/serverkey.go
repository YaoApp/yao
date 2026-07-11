package setting

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"time"
)

const (
	serverKeyNS      = "server_keys"
	serverKeyIndexNS = "server_keys_index"
	serverKeyPrefix  = "yao-sk:"
)

// ServerKeyInfo holds non-secret metadata about a server key.
type ServerKeyInfo struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	KeyHash   string `json:"key_hash"`
	CreatedAt string `json:"created_at"`
	ExpiresAt string `json:"expires_at,omitempty"`
	Revoked   bool   `json:"revoked"`
	LastUsed  string `json:"last_used,omitempty"`
}

// IsServerKeyFormat checks if the token has the server key prefix.
func IsServerKeyFormat(token string) bool {
	return len(token) > len(serverKeyPrefix) && token[:len(serverKeyPrefix)] == serverKeyPrefix
}

// HashServerKey returns the SHA-256 hex digest of a server key.
func HashServerKey(key string) string {
	h := sha256.Sum256([]byte(key))
	return hex.EncodeToString(h[:])
}

// CreateServerKey generates a new server key with the given name and optional TTL.
// Returns the plaintext key (shown once) and the key ID.
// If ttl is 0, the key never expires.
func CreateServerKey(name string, ttl ...time.Duration) (plainKey, keyID string, err error) {
	if Global == nil {
		return "", "", fmt.Errorf("setting registry not initialized")
	}

	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", "", fmt.Errorf("generate random key: %w", err)
	}

	plainKey = serverKeyPrefix + hex.EncodeToString(b)
	keyHash := HashServerKey(plainKey)
	keyID = "sk-" + hex.EncodeToString(b[:6])

	scope := ScopeID{Scope: ScopeSystem}
	now := time.Now().UTC().Format(time.RFC3339)

	keys, _ := Global.Get(scope, serverKeyNS)
	if keys == nil {
		keys = make(map[string]interface{})
	}
	entry := map[string]interface{}{
		"name":       name,
		"key_hash":   keyHash,
		"created_at": now,
		"revoked":    false,
	}
	if len(ttl) > 0 && ttl[0] > 0 {
		entry["expires_at"] = time.Now().UTC().Add(ttl[0]).Format(time.RFC3339)
	}
	keys[keyID] = entry
	if _, err := Global.Set(scope, serverKeyNS, keys); err != nil {
		return "", "", fmt.Errorf("store server key: %w", err)
	}

	index, _ := Global.Get(scope, serverKeyIndexNS)
	if index == nil {
		index = make(map[string]interface{})
	}
	index[keyHash] = keyID
	if _, err := Global.Set(scope, serverKeyIndexNS, index); err != nil {
		return "", "", fmt.Errorf("store server key index: %w", err)
	}

	return plainKey, keyID, nil
}

// ValidateServerKey validates a plaintext server key.
// Returns the key ID if valid, or an error if invalid/revoked.
func ValidateServerKey(plainKey string) (keyID string, err error) {
	if Global == nil {
		return "", fmt.Errorf("setting registry not initialized")
	}

	keyHash := HashServerKey(plainKey)
	scope := ScopeID{Scope: ScopeSystem}

	index, _ := Global.Get(scope, serverKeyIndexNS)
	if index == nil {
		return "", fmt.Errorf("invalid server key")
	}

	idRaw, ok := index[keyHash]
	if !ok {
		return "", fmt.Errorf("invalid server key")
	}
	keyID, _ = idRaw.(string)
	if keyID == "" {
		return "", fmt.Errorf("invalid server key")
	}

	keys, _ := Global.Get(scope, serverKeyNS)
	if keys == nil {
		return "", fmt.Errorf("invalid server key")
	}

	entryRaw, ok := keys[keyID]
	if !ok {
		return "", fmt.Errorf("invalid server key")
	}

	m := normalizeMap(entryRaw)
	if m == nil {
		return "", fmt.Errorf("invalid server key")
	}

	if revoked, _ := m["revoked"].(bool); revoked {
		return "", fmt.Errorf("server key has been revoked")
	}

	if expiresAt, _ := m["expires_at"].(string); expiresAt != "" {
		if t, err := time.Parse(time.RFC3339, expiresAt); err == nil && time.Now().After(t) {
			return "", fmt.Errorf("server key has expired")
		}
	}

	return keyID, nil
}

// ListServerKeys returns metadata for all server keys.
func ListServerKeys() ([]ServerKeyInfo, error) {
	if Global == nil {
		return nil, fmt.Errorf("setting registry not initialized")
	}

	scope := ScopeID{Scope: ScopeSystem}
	keys, _ := Global.Get(scope, serverKeyNS)
	if keys == nil {
		return nil, nil
	}

	var result []ServerKeyInfo
	for id, entryRaw := range keys {
		m := normalizeMap(entryRaw)
		if m == nil {
			continue
		}

		info := ServerKeyInfo{ID: id}
		info.Name, _ = m["name"].(string)
		info.KeyHash, _ = m["key_hash"].(string)
		info.CreatedAt, _ = m["created_at"].(string)
		info.ExpiresAt, _ = m["expires_at"].(string)
		info.Revoked, _ = m["revoked"].(bool)
		info.LastUsed, _ = m["last_used"].(string)
		result = append(result, info)
	}
	return result, nil
}

// RevokeServerKey marks a server key as revoked and removes it from the index.
func RevokeServerKey(keyID string) error {
	if Global == nil {
		return fmt.Errorf("setting registry not initialized")
	}

	scope := ScopeID{Scope: ScopeSystem}
	keys, _ := Global.Get(scope, serverKeyNS)
	if keys == nil {
		return fmt.Errorf("server key %q not found", keyID)
	}

	entryRaw, ok := keys[keyID]
	if !ok {
		return fmt.Errorf("server key %q not found", keyID)
	}

	m := normalizeMap(entryRaw)
	if m == nil {
		return fmt.Errorf("server key %q not found", keyID)
	}

	m["revoked"] = true
	keys[keyID] = m
	if _, err := Global.Set(scope, serverKeyNS, keys); err != nil {
		return fmt.Errorf("revoke server key: %w", err)
	}

	keyHash, _ := m["key_hash"].(string)
	if keyHash != "" {
		index, _ := Global.Get(scope, serverKeyIndexNS)
		if index != nil {
			delete(index, keyHash)
			Global.Set(scope, serverKeyIndexNS, index)
		}
	}

	return nil
}

// UpdateServerKeyLastUsed updates last_used with 5-minute throttling.
func UpdateServerKeyLastUsed(keyID string) {
	if Global == nil {
		return
	}

	scope := ScopeID{Scope: ScopeSystem}
	keys, _ := Global.Get(scope, serverKeyNS)
	if keys == nil {
		return
	}

	entryRaw, ok := keys[keyID]
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
	keys[keyID] = m
	Global.Set(scope, serverKeyNS, keys)
}
