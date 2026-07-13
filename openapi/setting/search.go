package setting

import (
	"bytes"
	_ "embed"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/yaoapp/yao/openapi/oauth/authorized"
	oauthTypes "github.com/yaoapp/yao/openapi/oauth/types"
	"github.com/yaoapp/yao/openapi/response"
	"github.com/yaoapp/yao/setting"
	"gopkg.in/yaml.v3"
)

//go:embed search_presets.yml
var searchPresetsYML []byte

var searchPresets []SearchProviderPreset

func init() {
	if err := yaml.Unmarshal(searchPresetsYML, &searchPresets); err != nil {
		searchPresets = nil
	}
}

func searchFindPreset(key string) *SearchProviderPreset {
	for i := range searchPresets {
		if searchPresets[i].Key == key {
			return &searchPresets[i]
		}
	}
	return nil
}

func searchScope(info *oauthTypes.AuthorizedInfo) setting.ScopeID {
	if info.TeamID != "" {
		return setting.ScopeID{Scope: setting.ScopeTeam, TeamID: info.TeamID}
	}
	return setting.ScopeID{Scope: setting.ScopeUser, UserID: info.UserID}
}

func searchProviderNS(key string) string {
	return "search.providers." + key
}

const searchAssignmentNS = "search.tool_assignment"

func searchPasswordFields(preset *SearchProviderPreset) map[string]bool {
	m := make(map[string]bool)
	for _, f := range preset.Fields {
		if f.Type == "password" {
			m[f.Key] = true
		}
	}
	return m
}

// ---------------------------------------------------------------------------
// GET /setting/search
// ---------------------------------------------------------------------------

func handleSearchGet(c *gin.Context) {
	info := authorized.GetInfo(c)
	scope := searchScope(info)

	providers := make([]SearchProviderConfig, 0, len(searchPresets))
	for _, preset := range searchPresets {
		cfg := SearchProviderConfig{
			PresetKey:   preset.Key,
			Enabled:     false,
			FieldValues: map[string]string{},
			Status:      "unconfigured",
		}

		if preset.IsCloud {
			var cloudSaved map[string]interface{}
			if setting.Global != nil {
				cloudSaved, _ = setting.Global.GetMerged(info.UserID, info.TeamID, cloudNS)
			}
			if cloudSaved != nil {
				if st, ok := cloudSaved["status"].(string); ok && st == "connected" {
					cfg.Enabled = true
					cfg.Status = "connected"
				}
			}
		} else if setting.Global != nil {
			saved, _ := setting.Global.Get(scope, searchProviderNS(preset.Key))
			if saved != nil {
				if v, ok := saved["enabled"].(bool); ok {
					cfg.Enabled = v
				}
				if v, ok := saved["status"].(string); ok && v != "" {
					cfg.Status = v
				}
				pwFields := searchPasswordFields(&preset)
				if fv, ok := saved["field_values"].(map[string]interface{}); ok {
					for k, v := range fv {
						s, _ := v.(string)
						if pwFields[k] && s != "" {
							cfg.FieldValues[k] = cloudMaskKey(cloudDecrypt(s))
						} else {
							cfg.FieldValues[k] = s
						}
					}
				}
			}
		}
		providers = append(providers, cfg)
	}

	var assignment SearchToolAssignment
	if setting.Global != nil {
		saved, _ := setting.Global.Get(scope, searchAssignmentNS)
		if saved != nil {
			if v, ok := saved["web_search"].(string); ok && v != "" {
				assignment.WebSearch = &v
			}
			if v, ok := saved["web_scrape"].(string); ok && v != "" {
				assignment.WebScrape = &v
			}
		}
	}

	response.RespondWithSuccess(c, http.StatusOK, SearchPageData{
		Presets:        searchPresets,
		Providers:      providers,
		ToolAssignment: assignment,
	})
}

// ---------------------------------------------------------------------------
// PUT /setting/search/providers/:key
// ---------------------------------------------------------------------------

func handleSearchProviderUpdate(c *gin.Context) {
	if !guardOwner(c) {
		return
	}

	key := c.Param("key")
	if key == "cloud" {
		respondError(c, http.StatusBadRequest, "cloud provider is managed by cloud service settings")
		return
	}

	preset := searchFindPreset(key)
	if preset == nil {
		respondError(c, http.StatusBadRequest, fmt.Sprintf("unknown provider: %s", key))
		return
	}

	var body struct {
		FieldValues map[string]string `json:"field_values"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		respondError(c, http.StatusBadRequest, "invalid request body")
		return
	}

	if setting.Global == nil {
		respondError(c, http.StatusInternalServerError, "setting registry not initialized")
		return
	}

	info := authorized.GetInfo(c)
	scope := searchScope(info)

	existing, _ := setting.Global.Get(scope, searchProviderNS(key))
	m := make(map[string]interface{})
	for k, v := range existing {
		m[k] = v
	}

	validFields := make(map[string]bool)
	for _, f := range preset.Fields {
		validFields[f.Key] = true
	}

	pwFields := searchPasswordFields(preset)
	existingFV := map[string]interface{}{}
	if fv, ok := m["field_values"].(map[string]interface{}); ok {
		existingFV = fv
	}

	newFV := make(map[string]interface{})
	for k, v := range existingFV {
		newFV[k] = v
	}

	for k, v := range body.FieldValues {
		if !validFields[k] {
			continue
		}
		if pwFields[k] {
			if v == "" {
				continue // keep existing
			}
			newFV[k] = cloudEncrypt(v)
		} else {
			newFV[k] = v
		}
	}

	m["field_values"] = newFV
	if _, ok := m["enabled"]; !ok {
		m["enabled"] = false
	}
	if _, ok := m["status"]; !ok {
		m["status"] = "unconfigured"
	}

	if _, err := setting.Global.Set(scope, searchProviderNS(key), m); err != nil {
		respondError(c, http.StatusInternalServerError, err.Error())
		return
	}

	cfg := SearchProviderConfig{
		PresetKey:   key,
		Enabled:     false,
		FieldValues: map[string]string{},
		Status:      "unconfigured",
	}
	if v, ok := m["enabled"].(bool); ok {
		cfg.Enabled = v
	}
	if v, ok := m["status"].(string); ok && v != "" {
		cfg.Status = v
	}
	if fv, ok := m["field_values"].(map[string]interface{}); ok {
		for k, v := range fv {
			s, _ := v.(string)
			if pwFields[k] && s != "" {
				cfg.FieldValues[k] = cloudMaskKey(cloudDecrypt(s))
			} else {
				cfg.FieldValues[k] = s
			}
		}
	}

	response.RespondWithSuccess(c, http.StatusOK, cfg)
}

// ---------------------------------------------------------------------------
// PUT /setting/search/providers/:key/toggle
// ---------------------------------------------------------------------------

func handleSearchProviderToggle(c *gin.Context) {
	if !guardOwner(c) {
		return
	}

	key := c.Param("key")
	if key == "cloud" {
		respondError(c, http.StatusBadRequest, "cloud provider is managed by cloud service settings")
		return
	}

	preset := searchFindPreset(key)
	if preset == nil {
		respondError(c, http.StatusBadRequest, fmt.Sprintf("unknown provider: %s", key))
		return
	}

	var body struct {
		Enabled bool `json:"enabled"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		respondError(c, http.StatusBadRequest, "invalid request body")
		return
	}

	if setting.Global == nil {
		respondError(c, http.StatusInternalServerError, "setting registry not initialized")
		return
	}

	info := authorized.GetInfo(c)
	scope := searchScope(info)

	existing, _ := setting.Global.Get(scope, searchProviderNS(key))
	m := make(map[string]interface{})
	for k, v := range existing {
		m[k] = v
	}
	m["enabled"] = body.Enabled

	if _, err := setting.Global.Set(scope, searchProviderNS(key), m); err != nil {
		respondError(c, http.StatusInternalServerError, err.Error())
		return
	}

	// When disabling, clear tool_assignment references
	if !body.Enabled {
		assignData, _ := setting.Global.Get(scope, searchAssignmentNS)
		if assignData != nil {
			changed := false
			if v, ok := assignData["web_search"].(string); ok && v == key {
				assignData["web_search"] = ""
				changed = true
			}
			if v, ok := assignData["web_scrape"].(string); ok && v == key {
				assignData["web_scrape"] = ""
				changed = true
			}
			if changed {
				setting.Global.Set(scope, searchAssignmentNS, assignData)
			}
		}
	}

	cfg := SearchProviderConfig{
		PresetKey:   key,
		Enabled:     body.Enabled,
		FieldValues: map[string]string{},
		Status:      "unconfigured",
	}
	if v, ok := m["status"].(string); ok && v != "" {
		cfg.Status = v
	}
	pwFields := searchPasswordFields(preset)
	if fv, ok := m["field_values"].(map[string]interface{}); ok {
		for k, v := range fv {
			s, _ := v.(string)
			if pwFields[k] && s != "" {
				cfg.FieldValues[k] = cloudMaskKey(cloudDecrypt(s))
			} else {
				cfg.FieldValues[k] = s
			}
		}
	}

	response.RespondWithSuccess(c, http.StatusOK, cfg)
}

// ---------------------------------------------------------------------------
// POST /setting/search/providers/:key/test
// ---------------------------------------------------------------------------

func handleSearchProviderTest(c *gin.Context) {
	if !guardOwner(c) {
		return
	}

	key := c.Param("key")
	if key == "cloud" {
		respondError(c, http.StatusBadRequest, "cloud provider status is determined by cloud service configuration")
		return
	}

	preset := searchFindPreset(key)
	if preset == nil {
		respondError(c, http.StatusBadRequest, fmt.Sprintf("unknown provider: %s", key))
		return
	}

	var body struct {
		FieldValues map[string]string `json:"field_values"`
	}
	c.ShouldBindJSON(&body)

	info := authorized.GetInfo(c)
	scope := searchScope(info)

	// Resolve API key: prefer body, fall back to saved
	apiKey := ""
	if body.FieldValues != nil {
		apiKey = body.FieldValues["api_key"]
	}
	if apiKey == "" && setting.Global != nil {
		saved, _ := setting.Global.Get(scope, searchProviderNS(key))
		if saved != nil {
			if fv, ok := saved["field_values"].(map[string]interface{}); ok {
				if v, ok := fv["api_key"].(string); ok {
					apiKey = cloudDecrypt(v)
				}
			}
		}
	}

	if apiKey == "" {
		response.RespondWithSuccess(c, http.StatusOK, SearchTestResult{
			Success: false,
			Message: "API key is required",
		})
		return
	}

	start := time.Now()
	var testErr error

	zone := ""
	if body.FieldValues != nil {
		zone = body.FieldValues["zone"]
	}
	if zone == "" && setting.Global != nil {
		saved, _ := setting.Global.Get(scope, searchProviderNS(key))
		if saved != nil {
			if fv, ok := saved["field_values"].(map[string]interface{}); ok {
				if v, ok := fv["zone"].(string); ok {
					zone = v
				}
			}
		}
	}

	switch key {
	case "tavily":
		testErr = searchTestTavily(apiKey)
	case "serper":
		testErr = searchTestSerper(apiKey)
	case "brightdata":
		testErr = searchTestBrightdata(apiKey, zone)
	default:
		respondError(c, http.StatusBadRequest, fmt.Sprintf("test not supported for provider: %s", key))
		return
	}

	latency := time.Since(start).Milliseconds()

	if testErr != nil {
		// Update status to disconnected
		if setting.Global != nil {
			saved, _ := setting.Global.Get(scope, searchProviderNS(key))
			if saved == nil {
				saved = map[string]interface{}{}
			}
			saved["status"] = "disconnected"
			setting.Global.Set(scope, searchProviderNS(key), saved)
		}
		response.RespondWithSuccess(c, http.StatusOK, SearchTestResult{
			Success: false,
			Message: testErr.Error(),
		})
		return
	}

	// Update status to connected
	if setting.Global != nil {
		saved, _ := setting.Global.Get(scope, searchProviderNS(key))
		if saved == nil {
			saved = map[string]interface{}{}
		}
		saved["status"] = "connected"
		setting.Global.Set(scope, searchProviderNS(key), saved)
	}

	response.RespondWithSuccess(c, http.StatusOK, SearchTestResult{
		Success:   true,
		Message:   "Connection successful",
		LatencyMs: latency,
	})
}

func searchTestTavily(apiKey string) error {
	payload, _ := json.Marshal(map[string]interface{}{
		"api_key": apiKey,
		"query":   "test",
	})
	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Post("https://api.tavily.com/search", "application/json", bytes.NewReader(payload))
	if err != nil {
		return fmt.Errorf("connection failed: %s", err.Error())
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		return fmt.Errorf("invalid API key (HTTP %d)", resp.StatusCode)
	}
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("server returned HTTP %d", resp.StatusCode)
	}
	return nil
}

func searchTestSerper(apiKey string) error {
	payload, _ := json.Marshal(map[string]string{"q": "test"})
	client := &http.Client{Timeout: 15 * time.Second}
	req, err := http.NewRequest("POST", "https://google.serper.dev/search", bytes.NewReader(payload))
	if err != nil {
		return fmt.Errorf("failed to build request: %s", err.Error())
	}
	req.Header.Set("X-API-KEY", apiKey)
	req.Header.Set("Content-Type", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("connection failed: %s", err.Error())
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		return fmt.Errorf("invalid API key (HTTP %d)", resp.StatusCode)
	}
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("server returned HTTP %d", resp.StatusCode)
	}
	return nil
}

func searchTestBrightdata(apiKey, zone string) error {
	if zone == "" {
		return fmt.Errorf("Zone is required")
	}
	client := &http.Client{Timeout: 10 * time.Second}
	req, err := http.NewRequest("GET", "https://api.brightdata.com/zone/status?zone="+zone, nil)
	if err != nil {
		return fmt.Errorf("failed to build request: %s", err.Error())
	}
	req.Header.Set("Authorization", "Bearer "+apiKey)
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("connection failed: %s", err.Error())
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		return fmt.Errorf("invalid API key (HTTP %d)", resp.StatusCode)
	}
	if resp.StatusCode == http.StatusNotFound {
		return fmt.Errorf("zone '%s' not found", zone)
	}
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("server returned HTTP %d", resp.StatusCode)
	}
	return nil
}

// ---------------------------------------------------------------------------
// PUT /setting/search/tool-assignment
// ---------------------------------------------------------------------------

func handleSearchToolAssignment(c *gin.Context) {
	if !guardOwner(c) {
		return
	}

	var body struct {
		WebSearch *string `json:"web_search"`
		WebScrape *string `json:"web_scrape"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		respondError(c, http.StatusBadRequest, "invalid request body")
		return
	}

	if setting.Global == nil {
		respondError(c, http.StatusInternalServerError, "setting registry not initialized")
		return
	}

	info := authorized.GetInfo(c)
	scope := searchScope(info)

	// Validate: provider must be enabled and support the tool
	validateAssignment := func(providerKey *string, toolType string) error {
		if providerKey == nil || *providerKey == "" {
			return nil
		}
		preset := searchFindPreset(*providerKey)
		if preset == nil {
			return fmt.Errorf("unknown provider: %s", *providerKey)
		}

		hasTools := false
		for _, t := range preset.Tools {
			if t == toolType {
				hasTools = true
				break
			}
		}
		if !hasTools {
			return fmt.Errorf("provider %s does not support %s", *providerKey, toolType)
		}

		if preset.IsCloud || len(preset.Fields) == 0 {
			return nil
		}

		saved, _ := setting.Global.Get(scope, searchProviderNS(*providerKey))
		if saved != nil {
			if v, ok := saved["enabled"].(bool); ok && v {
				return nil
			}
		}
		return fmt.Errorf("provider %s is not enabled", *providerKey)
	}

	if err := validateAssignment(body.WebSearch, "web_search"); err != nil {
		respondError(c, http.StatusBadRequest, err.Error())
		return
	}
	if err := validateAssignment(body.WebScrape, "web_scrape"); err != nil {
		respondError(c, http.StatusBadRequest, err.Error())
		return
	}

	m := make(map[string]interface{})
	if body.WebSearch != nil {
		m["web_search"] = *body.WebSearch
	} else {
		m["web_search"] = ""
	}
	if body.WebScrape != nil {
		m["web_scrape"] = *body.WebScrape
	} else {
		m["web_scrape"] = ""
	}

	if _, err := setting.Global.Set(scope, searchAssignmentNS, m); err != nil {
		respondError(c, http.StatusInternalServerError, err.Error())
		return
	}

	result := SearchToolAssignment{}
	if v, ok := m["web_search"].(string); ok && v != "" {
		result.WebSearch = &v
	}
	if v, ok := m["web_scrape"].(string); ok && v != "" {
		result.WebScrape = &v
	}

	response.RespondWithSuccess(c, http.StatusOK, result)
}
