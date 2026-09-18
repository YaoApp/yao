package setting

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/yaoapp/yao/llmprovider"
	"github.com/yaoapp/yao/openapi/oauth/authorized"
	oauthTypes "github.com/yaoapp/yao/openapi/oauth/types"
	"github.com/yaoapp/yao/openapi/response"
	"github.com/yaoapp/yao/setting"
)

const (
	taoNS       = "tao"
	taoMaskLen  = 4
	taoPresetNS = "taoservice"
)

const (
	defaultTaoBaseURLCN = "https://tao-api.yaoagents.cn"
	defaultTaoBaseURLEN = "https://us.yao.run"
)

// resolveTaoBaseURL returns the Tao API base URL for the given locale.
// Locales starting with "zh" resolve to the Chinese endpoint; all others to the international one.
// ENV overrides: TAO_BASE_URL_CN, TAO_BASE_URL_EN.
func resolveTaoBaseURL(locale string) string {
	if strings.HasPrefix(strings.ToLower(locale), "zh") {
		if v := os.Getenv("TAO_BASE_URL_CN"); v != "" {
			return strings.TrimRight(v, "/")
		}
		return defaultTaoBaseURLCN
	}
	if v := os.Getenv("TAO_BASE_URL_EN"); v != "" {
		return strings.TrimRight(v, "/")
	}
	return defaultTaoBaseURLEN
}

func taoScope(info *oauthTypes.AuthorizedInfo) setting.ScopeID {
	if info.TeamID != "" {
		return setting.ScopeID{Scope: setting.ScopeTeam, TeamID: info.TeamID}
	}
	return setting.ScopeID{Scope: setting.ScopeUser, UserID: info.UserID}
}

func taoMaskKey(key string) string {
	if key == "" {
		return ""
	}
	if len(key) <= taoMaskLen {
		return strings.Repeat("*", len(key))
	}
	prefix := key[:3]
	suffix := key[len(key)-taoMaskLen:]
	return prefix + "..." + suffix
}

// ---------------------------------------------------------------------------
// Handlers
// ---------------------------------------------------------------------------

// handleTaoVerify validates a Tao Service key without persisting it.
// POST /setting/tao/verify
func handleTaoVerify(c *gin.Context) {
	if !guardOwner(c) {
		return
	}

	var body struct {
		Key string `json:"key"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		respondError(c, http.StatusBadRequest, "invalid request body")
		return
	}
	if body.Key == "" {
		respondError(c, http.StatusBadRequest, "key is required")
		return
	}

	locale := c.DefaultQuery("locale", "en-us")
	baseURL := resolveTaoBaseURL(locale)

	result := taoCallVerify(baseURL, body.Key)
	response.RespondWithSuccess(c, http.StatusOK, result)
}

// handleTaoSetup verifies, persists, and auto-configures LLM + search for Tao Service.
// POST /setting/tao/setup
func handleTaoSetup(c *gin.Context) {
	if !guardOwner(c) {
		return
	}

	var body struct {
		Key string `json:"key"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		respondError(c, http.StatusBadRequest, "invalid request body")
		return
	}
	if body.Key == "" {
		respondError(c, http.StatusBadRequest, "key is required")
		return
	}

	info := authorized.GetInfo(c)
	locale := c.DefaultQuery("locale", "en-us")
	baseURL := resolveTaoBaseURL(locale)

	// Step 1: verify key
	verify := taoCallVerify(baseURL, body.Key)
	if !verify.Valid {
		response.RespondWithSuccess(c, http.StatusOK, TaoSetupResult{
			Success:          false,
			Message:          verify.Message,
			BalanceAvailable: verify.BalanceAvailable,
		})
		return
	}

	if setting.Global == nil {
		respondError(c, http.StatusInternalServerError, "setting registry not initialized")
		return
	}

	scope := taoScope(info)
	owner := llmOwner(info)

	// Step 2: persist credentials
	m := map[string]interface{}{
		"base_url": baseURL,
		"api_key":  cloudEncrypt(body.Key),
		"status":   "connected",
		"locale":   locale,
	}
	if _, err := setting.Global.Set(scope, taoNS, m); err != nil {
		respondError(c, http.StatusInternalServerError, err.Error())
		return
	}

	// Step 3: fetch models
	models := fetchCloudModels(baseURL, body.Key)
	modelCount := len(models)

	// Step 4: register preset + create provider
	var modelInfos []llmprovider.ModelInfo
	if len(models) > 0 {
		rawJSON, _ := json.Marshal(models)
		json.Unmarshal(rawJSON, &modelInfos)
		for i := range modelInfos {
			modelInfos[i].Enabled = true
		}
	}

	llmprovider.RegisterPreset(llmprovider.ProviderPreset{
		Key:           taoPresetNS,
		Name:          "Tao Service",
		Type:          "openai",
		APIURL:        baseURL,
		RequireKey:    false,
		IsCloud:       true,
		DefaultModels: modelInfos,
	})

	provKey := llmprovider.ScopedKey(owner, taoPresetNS)
	provider := llmprovider.Provider{
		Key:        provKey,
		Name:       "Tao Service",
		Type:       "openai",
		APIURL:     baseURL,
		APIKey:     body.Key,
		Models:     modelInfos,
		Enabled:    true,
		Status:     "connected",
		PresetKey:  taoPresetNS,
		RequireKey: false,
		Source:     llmprovider.ProviderSourceDynamic,
		Owner:      *owner,
	}

	if llmprovider.Global != nil {
		existing, _ := llmprovider.Global.Get(provKey)
		if existing != nil {
			llmprovider.Global.Update(provKey, &provider)
		} else {
			llmprovider.Global.Create(&provider)
		}
	}

	// Step 5: fetch model-defaults and assign roles
	roles := taoFetchModelDefaults(baseURL)
	if roles == nil {
		roles = map[string]string{}
	}

	// Ensure "default" role has a value
	if _, ok := roles["default"]; !ok && modelCount > 0 {
		roles["default"] = modelInfos[0].ID
	}

	roleMap := make(map[string]interface{})
	for roleName, modelID := range roles {
		roleMap[roleName] = map[string]interface{}{
			"provider": provKey,
			"model":    modelID,
		}
	}
	setting.Global.Set(scope, llmRolesNS, roleMap)

	// Step 6: assign search tools to tao preset
	searchAssign := map[string]interface{}{
		"web_search": "tao",
		"web_scrape": "tao",
	}
	setting.Global.Set(scope, searchAssignmentNS, searchAssign)

	// Step 7: determine available services from model capabilities
	services := taoDetectServices(models)

	// Step 8: update stored services
	m["services"] = services
	setting.Global.Set(scope, taoNS, m)

	response.RespondWithSuccess(c, http.StatusOK, TaoSetupResult{
		Success:          true,
		Balance:          verify.Balance,
		BalanceAvailable: verify.BalanceAvailable,
		Configured: &TaoSetupDetails{
			LLM: &TaoSetupLLM{
				ProviderName: "Tao Service",
				ModelCount:   modelCount,
				Roles:        roles,
			},
			Search: &TaoSetupSearch{
				ProviderName: "Tao Service",
				Tools:        []string{"web_search", "web_scrape"},
			},
		},
	})
}

// handleTaoGet returns the stored Tao configuration with a live balance probe.
// GET /setting/tao/config
func handleTaoGet(c *gin.Context) {
	info := authorized.GetInfo(c)

	var saved map[string]interface{}
	if setting.Global != nil {
		saved, _ = setting.Global.GetMerged(info.UserID, info.TeamID, taoNS)
	}

	data := TaoConfig{
		Status:   "unconfigured",
		Services: TaoServices{},
	}

	if saved == nil {
		response.RespondWithSuccess(c, http.StatusOK, data)
		return
	}

	if v, ok := saved["base_url"].(string); ok {
		data.BaseURL = v
	}
	if v, ok := saved["api_key"].(string); ok && v != "" {
		data.Key = taoMaskKey(cloudDecrypt(v))
	}
	if v, ok := saved["status"].(string); ok && v != "" {
		data.Status = v
	}
	if svc, ok := saved["services"].(map[string]interface{}); ok {
		data.Services = taoServicesFromMap(svc)
	}

	// Live balance probe with 3s timeout
	if data.Status == "connected" {
		if encKey, _ := saved["api_key"].(string); encKey != "" {
			apiKey := cloudDecrypt(encKey)
			bal, balAvail := taoFetchBalance(data.BaseURL, apiKey, 3*time.Second)
			data.Balance = bal
			data.BalanceAvailable = balAvail
		}
	}

	response.RespondWithSuccess(c, http.StatusOK, data)
}

// handleTaoUpdate updates the Tao key. Validates the new key before saving.
// PUT /setting/tao/config
func handleTaoUpdate(c *gin.Context) {
	if !guardOwner(c) {
		return
	}

	info := authorized.GetInfo(c)
	scope := taoScope(info)

	var body struct {
		Key string `json:"key"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		respondError(c, http.StatusBadRequest, "invalid request body")
		return
	}

	if setting.Global == nil {
		respondError(c, http.StatusInternalServerError, "setting registry not initialized")
		return
	}

	existing, _ := setting.Global.Get(scope, taoNS)
	m := make(map[string]interface{})
	for k, v := range existing {
		m[k] = v
	}

	if body.Key != "" {
		locale := c.DefaultQuery("locale", "en-us")
		baseURL := resolveTaoBaseURL(locale)

		verify := taoCallVerify(baseURL, body.Key)
		if !verify.Valid {
			respondError(c, http.StatusBadRequest, fmt.Sprintf("API key validation failed: %s", verify.Message))
			return
		}

		m["api_key"] = cloudEncrypt(body.Key)
		m["base_url"] = baseURL
		m["status"] = "connected"
	}

	if _, err := setting.Global.Set(scope, taoNS, m); err != nil {
		respondError(c, http.StatusInternalServerError, err.Error())
		return
	}

	// Return updated config
	data := TaoConfig{
		Status:   "unconfigured",
		Services: TaoServices{},
	}
	if v, ok := m["base_url"].(string); ok {
		data.BaseURL = v
	}
	if v, ok := m["api_key"].(string); ok && v != "" {
		data.Key = taoMaskKey(cloudDecrypt(v))
	}
	if v, ok := m["status"].(string); ok && v != "" {
		data.Status = v
	}
	if svc, ok := m["services"].(map[string]interface{}); ok {
		data.Services = taoServicesFromMap(svc)
	}

	response.RespondWithSuccess(c, http.StatusOK, data)
}

// handleTaoBalance refreshes the balance by calling the upstream key endpoint.
// POST /setting/tao/balance
func handleTaoBalance(c *gin.Context) {
	if !guardOwner(c) {
		return
	}

	info := authorized.GetInfo(c)

	var saved map[string]interface{}
	if setting.Global != nil {
		saved, _ = setting.Global.GetMerged(info.UserID, info.TeamID, taoNS)
	}
	if saved == nil {
		respondError(c, http.StatusBadRequest, "tao service not configured")
		return
	}

	status, _ := saved["status"].(string)
	if status != "connected" {
		respondError(c, http.StatusBadRequest, "tao service not connected")
		return
	}

	encKey, _ := saved["api_key"].(string)
	if encKey == "" {
		respondError(c, http.StatusBadRequest, "no API key configured")
		return
	}

	baseURL, _ := saved["base_url"].(string)
	if baseURL == "" {
		baseURL = resolveTaoBaseURL("en-us")
	}

	apiKey := cloudDecrypt(encKey)
	bal, balAvail := taoFetchBalance(baseURL, apiKey, 10*time.Second)
	result := map[string]interface{}{
		"balance_available": balAvail,
	}
	if bal != nil {
		result["balance"] = *bal
	}
	response.RespondWithSuccess(c, http.StatusOK, result)
}

// handleTaoSignupGift returns the signup gift amount from the Tao platform.
// GET /setting/tao/signup-gift
func handleTaoSignupGift(c *gin.Context) {
	locale := c.DefaultQuery("locale", "en-us")
	baseURL := resolveTaoBaseURL(locale)
	url := strings.TrimRight(baseURL, "/") + "/api/v1/invite/campaigns/default"

	client := &http.Client{Timeout: 10 * time.Second}
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		respondError(c, http.StatusInternalServerError, "failed to build request")
		return
	}

	resp, err := client.Do(req)
	if err != nil {
		respondError(c, http.StatusBadGateway, "failed to reach Tao service")
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respondError(c, http.StatusBadGateway, fmt.Sprintf("Tao service returned HTTP %d", resp.StatusCode))
		return
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		respondError(c, http.StatusInternalServerError, "failed to read response")
		return
	}

	var campaign struct {
		SignupGift int64 `json:"signup_gift"`
	}
	if err := json.Unmarshal(body, &campaign); err != nil {
		respondError(c, http.StatusInternalServerError, "invalid response from Tao service")
		return
	}

	response.RespondWithSuccess(c, http.StatusOK, TaoSignupGift{
		SignupGift: campaign.SignupGift,
	})
}

// ---------------------------------------------------------------------------
// Upstream helpers
// ---------------------------------------------------------------------------

// taoCallVerify calls GET /v1/key with Bearer auth and returns a TaoVerifyResult.
func taoCallVerify(baseURL, apiKey string) TaoVerifyResult {
	url := strings.TrimRight(baseURL, "/") + "/v1/key"
	client := &http.Client{Timeout: 10 * time.Second}
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return TaoVerifyResult{Valid: false, Message: "failed to build request"}
	}
	req.Header.Set("Authorization", "Bearer "+apiKey)

	resp, err := client.Do(req)
	if err != nil {
		return TaoVerifyResult{Valid: false, Message: fmt.Sprintf("connection failed: %s", err.Error())}
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return TaoVerifyResult{Valid: false, Message: "failed to read response"}
	}

	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		var errBody struct {
			Error struct {
				Type    string `json:"type"`
				Message string `json:"message"`
			} `json:"error"`
		}
		json.Unmarshal(body, &errBody)
		errType := errBody.Error.Type
		msg := errBody.Error.Message
		if msg == "" {
			msg = fmt.Sprintf("invalid API key (HTTP %d)", resp.StatusCode)
		}
		return TaoVerifyResult{Valid: false, ErrorType: errType, Message: msg}
	}

	if resp.StatusCode != http.StatusOK {
		return TaoVerifyResult{Valid: false, Message: fmt.Sprintf("server returned HTTP %d", resp.StatusCode)}
	}

	var result struct {
		Valid            bool  `json:"valid"`
		Balance          int64 `json:"balance"`
		BalanceAvailable bool  `json:"balance_available"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return TaoVerifyResult{Valid: false, Message: "invalid response"}
	}

	if !result.Valid {
		return TaoVerifyResult{Valid: false, Message: "key is not valid"}
	}

	balance := result.Balance
	return TaoVerifyResult{
		Valid:            true,
		Balance:          &balance,
		BalanceAvailable: result.BalanceAvailable,
		Message:          "ok",
	}
}

// taoFetchBalance calls GET /v1/key and returns balance info.
// On timeout or error, returns (nil, false).
func taoFetchBalance(baseURL, apiKey string, timeout time.Duration) (*int64, bool) {
	url := strings.TrimRight(baseURL, "/") + "/v1/key"
	client := &http.Client{Timeout: timeout}
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, false
	}
	req.Header.Set("Authorization", "Bearer "+apiKey)

	resp, err := client.Do(req)
	if err != nil {
		return nil, false
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, false
	}

	body, _ := io.ReadAll(resp.Body)
	var result struct {
		Balance          int64 `json:"balance"`
		BalanceAvailable bool  `json:"balance_available"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, false
	}

	return &result.Balance, result.BalanceAvailable
}

// taoFetchModelDefaults calls GET /v1/model-defaults (no auth) and returns role→model mappings.
func taoFetchModelDefaults(baseURL string) map[string]string {
	url := strings.TrimRight(baseURL, "/") + "/v1/model-defaults"
	client := &http.Client{Timeout: 10 * time.Second}
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil
	}

	body, _ := io.ReadAll(resp.Body)
	var result struct {
		Data map[string]string `json:"data"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil
	}
	return result.Data
}

// taoDetectServices determines available Tao services from the model list.
func taoDetectServices(models []map[string]interface{}) TaoServices {
	svc := TaoServices{
		LLM:    len(models) > 0,
		Search: true,
		Scrape: true,
	}
	for _, m := range models {
		mode, _ := m["mode"].(string)
		switch mode {
		case "embedding":
			svc.Embedding = true
		case "audio_transcription", "audio_speech":
			svc.Audio = true
		case "image_generation":
			svc.Image = true
		}
		if getBool(m, "supports_vision") {
			svc.OCR = true
		}
	}
	return svc
}

// taoServicesFromMap deserializes a services map from the settings store.
func taoServicesFromMap(m map[string]interface{}) TaoServices {
	return TaoServices{
		LLM:       getBool(m, "llm"),
		Search:    getBool(m, "search"),
		Scrape:    getBool(m, "scrape"),
		OCR:       getBool(m, "ocr"),
		Image:     getBool(m, "image"),
		Audio:     getBool(m, "audio"),
		Embedding: getBool(m, "embedding"),
	}
}
