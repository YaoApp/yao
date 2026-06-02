package setting

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/yaoapp/yao/openapi/oauth"
	"github.com/yaoapp/yao/openapi/oauth/authorized"
	oauthTypes "github.com/yaoapp/yao/openapi/oauth/types"
	"github.com/yaoapp/yao/openapi/response"
	"github.com/yaoapp/yao/setting"
)

const (
	apiKeyNS          = "api_keys"
	apiKeyIndexNS     = "api_keys_index"
	apiKeyRandLen     = 32
	apiKeyMaxPerScope = 5
)

var apiKeyMu sync.Mutex

func apiKeyScope(info *oauthTypes.AuthorizedInfo) setting.ScopeID {
	if info.TeamID != "" {
		return setting.ScopeID{Scope: setting.ScopeTeam, TeamID: info.TeamID}
	}
	return setting.ScopeID{Scope: setting.ScopeUser, UserID: info.UserID}
}

// handleAPIKeyList returns all API keys for the current scope.
// GET /setting/api-keys
func handleAPIKeyList(c *gin.Context) {
	info := authorized.GetInfo(c)
	scope := apiKeyScope(info)

	if setting.Global == nil {
		response.RespondWithSuccess(c, http.StatusOK, []APIKeyResponse{})
		return
	}

	data, _ := setting.Global.Get(scope, apiKeyNS)
	keys := apiKeysFromMap(data)

	result := make([]APIKeyResponse, 0, len(keys))
	for _, k := range keys {
		result = append(result, APIKeyResponse{
			ID:        k.ID,
			Name:      k.Name,
			KeyPrefix: k.KeyPrefix,
			Status:    k.Status,
			ExpiresAt: k.ExpiresAt,
			CreatedAt: k.CreatedAt,
			LastUsed:  k.LastUsed,
		})
	}

	response.RespondWithSuccess(c, http.StatusOK, result)
}

// handleAPIKeyCreate creates a new API key.
// POST /setting/api-keys
func handleAPIKeyCreate(c *gin.Context) {
	if !guardOwner(c) {
		return
	}
	info := authorized.GetInfo(c)
	scope := apiKeyScope(info)

	var body APIKeyCreateRequest
	if err := c.ShouldBindJSON(&body); err != nil {
		respondError(c, http.StatusBadRequest, "invalid request body")
		return
	}

	if body.Name == "" {
		respondError(c, http.StatusBadRequest, "name is required")
		return
	}

	if setting.Global == nil {
		respondError(c, http.StatusInternalServerError, "setting registry not initialized")
		return
	}

	apiKeyMu.Lock()
	defer apiKeyMu.Unlock()

	data, _ := setting.Global.Get(scope, apiKeyNS)
	keys := apiKeysFromMap(data)

	activeCount := 0
	for _, k := range keys {
		if k.Status == "active" {
			activeCount++
		}
	}
	if activeCount >= apiKeyMaxPerScope {
		respondError(c, http.StatusBadRequest, fmt.Sprintf("maximum %d active API keys per scope", apiKeyMaxPerScope))
		return
	}

	plainKey, err := generateAPIKey()
	if err != nil {
		respondError(c, http.StatusInternalServerError, "failed to generate API key")
		return
	}

	keyHash := setting.HashAPIKey(plainKey)
	keyID := generateKeyID()
	now := time.Now().UTC().Format(time.RFC3339)

	entry := APIKeyEntry{
		ID:        keyID,
		Name:      body.Name,
		KeyPrefix: plainKey[:7],
		KeyHash:   keyHash,
		UserID:    info.UserID,
		Status:    "active",
		ExpiresAt: body.ExpiresAt,
		CreatedAt: now,
	}

	if data == nil {
		data = make(map[string]interface{})
	}
	data[keyID] = entryToMap(entry)

	if _, err := setting.Global.Set(scope, apiKeyNS, data); err != nil {
		respondError(c, http.StatusInternalServerError, err.Error())
		return
	}

	if err := apiKeyIndexAdd(keyHash, info.UserID, info.TeamID, keyID); err != nil {
		respondError(c, http.StatusInternalServerError, "failed to update index")
		return
	}

	response.RespondWithSuccess(c, http.StatusOK, APIKeyCreateResponse{
		ID:        keyID,
		Name:      body.Name,
		Key:       plainKey,
		KeyPrefix: plainKey[:7],
		Status:    "active",
		ExpiresAt: body.ExpiresAt,
		CreatedAt: now,
	})
}

// handleAPIKeyDelete deletes an API key.
// DELETE /setting/api-keys/:key_id
func handleAPIKeyDelete(c *gin.Context) {
	if !guardOwner(c) {
		return
	}
	info := authorized.GetInfo(c)
	scope := apiKeyScope(info)
	keyID := c.Param("key_id")

	if setting.Global == nil {
		respondError(c, http.StatusInternalServerError, "setting registry not initialized")
		return
	}

	apiKeyMu.Lock()
	defer apiKeyMu.Unlock()

	data, _ := setting.Global.Get(scope, apiKeyNS)
	if data == nil {
		respondError(c, http.StatusNotFound, "API key not found")
		return
	}

	entryRaw, ok := data[keyID]
	if !ok {
		respondError(c, http.StatusNotFound, "API key not found")
		return
	}

	entry := mapToEntry(entryRaw)
	if entry.KeyHash != "" {
		apiKeyIndexRemove(entry.KeyHash)
	}

	delete(data, keyID)
	if _, err := setting.Global.Set(scope, apiKeyNS, data); err != nil {
		respondError(c, http.StatusInternalServerError, err.Error())
		return
	}

	oauth.InvalidateAPIKeyToken(keyID)
	response.RespondWithSuccess(c, http.StatusOK, map[string]interface{}{"success": true})
}

// handleAPIKeyRegenerate regenerates an API key (deletes old, creates new with same name).
// POST /setting/api-keys/:key_id/regenerate
func handleAPIKeyRegenerate(c *gin.Context) {
	if !guardOwner(c) {
		return
	}
	info := authorized.GetInfo(c)
	scope := apiKeyScope(info)
	keyID := c.Param("key_id")

	if setting.Global == nil {
		respondError(c, http.StatusInternalServerError, "setting registry not initialized")
		return
	}

	apiKeyMu.Lock()
	defer apiKeyMu.Unlock()

	data, _ := setting.Global.Get(scope, apiKeyNS)
	if data == nil {
		respondError(c, http.StatusNotFound, "API key not found")
		return
	}

	entryRaw, ok := data[keyID]
	if !ok {
		respondError(c, http.StatusNotFound, "API key not found")
		return
	}

	oldEntry := mapToEntry(entryRaw)
	if oldEntry.KeyHash != "" {
		apiKeyIndexRemove(oldEntry.KeyHash)
	}

	plainKey, err := generateAPIKey()
	if err != nil {
		respondError(c, http.StatusInternalServerError, "failed to generate API key")
		return
	}

	keyHash := setting.HashAPIKey(plainKey)
	now := time.Now().UTC().Format(time.RFC3339)

	newEntry := APIKeyEntry{
		ID:        keyID,
		Name:      oldEntry.Name,
		KeyPrefix: plainKey[:7],
		KeyHash:   keyHash,
		UserID:    info.UserID,
		Status:    "active",
		ExpiresAt: oldEntry.ExpiresAt,
		CreatedAt: now,
	}

	data[keyID] = entryToMap(newEntry)
	if _, err := setting.Global.Set(scope, apiKeyNS, data); err != nil {
		respondError(c, http.StatusInternalServerError, err.Error())
		return
	}

	if err := apiKeyIndexAdd(keyHash, info.UserID, info.TeamID, keyID); err != nil {
		respondError(c, http.StatusInternalServerError, "failed to update index")
		return
	}

	oauth.InvalidateAPIKeyToken(keyID)
	response.RespondWithSuccess(c, http.StatusOK, APIKeyCreateResponse{
		ID:        keyID,
		Name:      oldEntry.Name,
		Key:       plainKey,
		KeyPrefix: plainKey[:7],
		Status:    "active",
		ExpiresAt: oldEntry.ExpiresAt,
		CreatedAt: now,
	})
}

// ---------------------------------------------------------------------------
// Index helpers
// ---------------------------------------------------------------------------

func apiKeyIndexAdd(keyHash, userID, teamID, keyID string) error {
	scope := setting.ScopeID{Scope: setting.ScopeSystem}
	data, _ := setting.Global.Get(scope, apiKeyIndexNS)
	if data == nil {
		data = make(map[string]interface{})
	}

	data[keyHash] = map[string]interface{}{
		"user_id": userID,
		"team_id": teamID,
		"key_id":  keyID,
	}

	_, err := setting.Global.Set(scope, apiKeyIndexNS, data)
	return err
}

func apiKeyIndexRemove(keyHash string) {
	scope := setting.ScopeID{Scope: setting.ScopeSystem}
	data, _ := setting.Global.Get(scope, apiKeyIndexNS)
	if data == nil {
		return
	}
	delete(data, keyHash)
	setting.Global.Set(scope, apiKeyIndexNS, data)
}

// ---------------------------------------------------------------------------
// Generation helpers
// ---------------------------------------------------------------------------

func generateAPIKey() (string, error) {
	b := make([]byte, apiKeyRandLen)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return "yao-" + hex.EncodeToString(b), nil
}

func generateKeyID() string {
	b := make([]byte, 8)
	rand.Read(b)
	return hex.EncodeToString(b)
}

// ---------------------------------------------------------------------------
// Map conversion helpers
// ---------------------------------------------------------------------------

func apiKeysFromMap(data map[string]interface{}) []APIKeyEntry {
	if data == nil {
		return nil
	}
	result := make([]APIKeyEntry, 0, len(data))
	for _, v := range data {
		entry := mapToEntry(v)
		if entry.ID != "" {
			result = append(result, entry)
		}
	}
	return result
}

func mapToEntry(v interface{}) APIKeyEntry {
	m := toInterfaceMap(v)
	if m == nil {
		return APIKeyEntry{}
	}
	return APIKeyEntry{
		ID:        strVal(m, "id"),
		Name:      strVal(m, "name"),
		KeyPrefix: strVal(m, "key_prefix"),
		KeyHash:   strVal(m, "key_hash"),
		UserID:    strVal(m, "user_id"),
		Status:    strVal(m, "status"),
		ExpiresAt: strVal(m, "expires_at"),
		CreatedAt: strVal(m, "created_at"),
		LastUsed:  strVal(m, "last_used"),
	}
}

func entryToMap(e APIKeyEntry) map[string]interface{} {
	m := map[string]interface{}{
		"id":         e.ID,
		"name":       e.Name,
		"key_prefix": e.KeyPrefix,
		"key_hash":   e.KeyHash,
		"user_id":    e.UserID,
		"status":     e.Status,
		"created_at": e.CreatedAt,
	}
	if e.ExpiresAt != "" {
		m["expires_at"] = e.ExpiresAt
	}
	if e.LastUsed != "" {
		m["last_used"] = e.LastUsed
	}
	return m
}

func toInterfaceMap(v interface{}) map[string]interface{} {
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

func strVal(m map[string]interface{}, key string) string {
	if v, ok := m[key].(string); ok {
		return v
	}
	return ""
}
