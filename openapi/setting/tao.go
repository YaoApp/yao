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
	"github.com/yaoapp/yao/openapi/response"
	"github.com/yaoapp/yao/setting"
	"github.com/yaoapp/yao/tools/ocr"
)

const (
	taoNS       = "tao"
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

	scope := scopeFromAuth(info)
	owner := llmOwner(info)

	// Step 2: persist credentials
	m := map[string]interface{}{
		"base_url": baseURL,
		"api_key":  encryptValue(body.Key),
		"status":   "connected",
		"locale":   locale,
	}
	if _, err := setting.Global.Set(scope, taoNS, m); err != nil {
		respondError(c, http.StatusInternalServerError, err.Error())
		return
	}

	// Step 3: fetch models (returns []ModelInfo, populates remoteReasoningCache)
	invalidateRemoteModelCache()
	modelInfos := fetchRemoteModels(baseURL, body.Key)
	modelCount := len(modelInfos)
	for i := range modelInfos {
		modelInfos[i].Enabled = true
	}

	// Step 4: register preset + create provider
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
	roleDefaults := taoFetchModelDefaults(baseURL)
	roleMap := make(map[string]interface{})

	for roleName, rd := range roleDefaults {
		reasoning := remoteReasoningCache[rd.Model]
		connID := resolveConnectorID(rd.Model, rd.Params, reasoning)
		roleMap[roleName] = map[string]interface{}{
			"provider": provKey,
			"model":    connID,
		}
	}

	if _, ok := roleMap["default"]; !ok && modelCount > 0 {
		roleMap["default"] = map[string]interface{}{
			"provider": provKey,
			"model":    modelInfos[0].ID,
		}
	}

	// Merge: preserve existing valid role assignments, only fill empty slots
	existingRoles, _ := setting.Global.Get(scope, llmRolesNS)
	mergeRoleAssignments(roleMap, existingRoles)
	rolesForResponse := buildRolesForResponse(roleMap)
	setting.Global.Set(scope, llmRolesNS, roleMap)

	// Step 6: assign search + OCR tools to tao preset (merge: keep existing non-empty values)
	existingSearch, _ := setting.Global.Get(scope, searchAssignmentNS)
	searchAssign := mergeToolAssignment(existingSearch, map[string]string{
		"web_search": "tao",
		"web_scrape": "tao",
	})
	setting.Global.Set(scope, searchAssignmentNS, searchAssign)

	existingOCR, _ := setting.Global.Get(scope, ocrAssignmentNS)
	ocrAssign := mergeToolAssignment(existingOCR, map[string]string{
		"ocr_recognize": "tao",
	})
	setting.Global.Set(scope, ocrAssignmentNS, ocrAssign)

	// Step 7: pre-fetch OCR types from Tao (populates cache for handler_tao)
	ocr.FetchTaoOCRTypes(baseURL, body.Key)

	// Step 8: determine available services via API
	services := taoFetchServices(baseURL)
	if !services.LLM && modelCount > 0 {
		services.LLM = true
	}

	// Step 9: update stored services
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
				Roles:        rolesForResponse,
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
		data.Key = maskKey(decryptValue(v))
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
			apiKey := decryptValue(encKey)
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
	scope := scopeFromAuth(info)

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

		m["api_key"] = encryptValue(body.Key)
		m["base_url"] = baseURL
		m["status"] = "connected"
	}

	if _, err := setting.Global.Set(scope, taoNS, m); err != nil {
		respondError(c, http.StatusInternalServerError, err.Error())
		return
	}

	if body.Key != "" {
		invalidateRemoteModelCache()
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
		data.Key = maskKey(decryptValue(v))
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

	apiKey := decryptValue(encKey)
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

// TaoRoleDefault holds a parsed role default from /v1/model-defaults (object format).
type TaoRoleDefault struct {
	Model  string
	Params map[string]interface{}
}

// taoFetchModelDefaults calls GET /v1/model-defaults (no auth) and returns role→default mappings.
func taoFetchModelDefaults(baseURL string) map[string]*TaoRoleDefault {
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
		Defaults map[string]interface{} `json:"defaults"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil
	}

	roles := make(map[string]*TaoRoleDefault, len(result.Defaults))
	for roleName, v := range result.Defaults {
		switch val := v.(type) {
		case string:
			if val != "" {
				roles[roleName] = &TaoRoleDefault{Model: val, Params: map[string]interface{}{}}
			}
		case map[string]interface{}:
			rd := &TaoRoleDefault{Params: make(map[string]interface{})}
			rd.Model, _ = val["model"].(string)
			for k, pv := range val {
				if k != "model" {
					rd.Params[k] = pv
				}
			}
			roles[roleName] = rd
		}
	}
	return roles
}

// taoFetchServices calls GET /v1/services and maps the result to TaoServices.
func taoFetchServices(baseURL string) TaoServices {
	url := strings.TrimRight(baseURL, "/") + "/v1/services"
	client := &http.Client{Timeout: 10 * time.Second}
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return TaoServices{}
	}

	resp, err := client.Do(req)
	if err != nil {
		return TaoServices{}
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return TaoServices{}
	}

	body, _ := io.ReadAll(resp.Body)
	var result struct {
		Data []struct {
			Service string `json:"service"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return TaoServices{}
	}

	svc := TaoServices{}
	for _, item := range result.Data {
		switch item.Service {
		case "chat":
			svc.LLM = true
		case "search":
			svc.Search = true
		case "fetch":
			svc.Scrape = true
		case "ocr":
			svc.OCR = true
		case "image":
			svc.Image = true
		case "audio":
			svc.Audio = true
		case "embedding":
			svc.Embedding = true
		}
	}
	return svc
}

// taoDetectServices determines available Tao services from the model list.
// Deprecated: use taoFetchServices instead.
func taoDetectServices(models []llmprovider.ModelInfo) TaoServices {
	svc := TaoServices{
		LLM:    len(models) > 0,
		Search: true,
		Scrape: true,
	}
	for _, m := range models {
		for _, c := range m.Capabilities {
			switch c {
			case "embedding":
				svc.Embedding = true
			case "audio":
				svc.Audio = true
			case "image_generation":
				svc.Image = true
			case "vision":
				svc.OCR = true
			}
		}
	}
	return svc
}

// isValidRoleAssignment checks that a stored role value is a map with
// non-empty "provider" and "model" strings.
func isValidRoleAssignment(v interface{}) bool {
	m, ok := v.(map[string]interface{})
	if !ok {
		return false
	}
	p, _ := m["provider"].(string)
	md, _ := m["model"].(string)
	return p != "" && md != ""
}

// mergeRoleAssignments merges Tao role defaults with existing user role assignments.
// Existing valid assignments (non-nil with non-empty provider+model) take precedence
// over defaults. Invalid existing entries are silently ignored so the Tao default fills in.
func mergeRoleAssignments(defaults, existing map[string]interface{}) {
	for k, v := range existing {
		if isValidRoleAssignment(v) {
			defaults[k] = v
		}
	}
}

// buildRolesForResponse extracts model IDs from a merged role assignment map.
// Each entry is expected to be map[string]interface{}{"provider":…, "model":…}.
func buildRolesForResponse(roleMap map[string]interface{}) map[string]string {
	result := make(map[string]string, len(roleMap))
	for roleName, val := range roleMap {
		if tm, ok := val.(map[string]interface{}); ok {
			if modelID, _ := tm["model"].(string); modelID != "" {
				result[roleName] = modelID
			}
		}
	}
	return result
}

// mergeToolAssignment copies existing assignment values and fills empty/missing
// slots from defaults. A slot is "empty" when the key is absent or its value
// is not a non-empty string.
func mergeToolAssignment(existing map[string]interface{}, defaults map[string]string) map[string]interface{} {
	result := make(map[string]interface{})
	for k, v := range existing {
		result[k] = v
	}
	for k, defVal := range defaults {
		if v, _ := result[k].(string); v == "" {
			result[k] = defVal
		}
	}
	return result
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
