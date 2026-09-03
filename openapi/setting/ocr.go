package setting

import (
	_ "embed"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/yaoapp/gou/connector"
	agentllm "github.com/yaoapp/yao/agent/llm"
	"github.com/yaoapp/yao/llmprovider"
	"github.com/yaoapp/yao/openapi/oauth/authorized"
	oauthTypes "github.com/yaoapp/yao/openapi/oauth/types"
	"github.com/yaoapp/yao/openapi/response"
	"github.com/yaoapp/yao/setting"
	"gopkg.in/yaml.v3"
)

//go:embed ocr_presets.yml
var ocrPresetsYML []byte

var ocrPresets []OCRProviderPreset

func init() {
	if err := yaml.Unmarshal(ocrPresetsYML, &ocrPresets); err != nil {
		ocrPresets = nil
	}
}

func ocrFindPreset(key string) *OCRProviderPreset {
	for i := range ocrPresets {
		if ocrPresets[i].Key == key {
			return &ocrPresets[i]
		}
	}
	return nil
}

func ocrScope(info *oauthTypes.AuthorizedInfo) setting.ScopeID {
	if info.TeamID != "" {
		return setting.ScopeID{Scope: setting.ScopeTeam, TeamID: info.TeamID}
	}
	return setting.ScopeID{Scope: setting.ScopeUser, UserID: info.UserID}
}

func ocrProviderNS(key string) string {
	return "ocr.providers." + key
}

const ocrAssignmentNS = "ocr.tool_assignment"

func ocrPasswordFields(preset *OCRProviderPreset) map[string]bool {
	m := make(map[string]bool)
	for _, f := range preset.Fields {
		if f.Type == "password" {
			m[f.Key] = true
		}
	}
	return m
}

// ocrSelectConnector resolves an LLM connector by ID.
// Uses llmprovider.Global.GetModel when available (handles providerKey:modelID format),
// falls back to connector.Select for raw connector IDs.
func ocrSelectConnector(connID string) (connector.Connector, error) {
	if llmprovider.Global != nil {
		return llmprovider.Global.GetModel(connID)
	}
	return connector.Select(connID)
}

// ---------------------------------------------------------------------------
// GET /setting/ocr
// ---------------------------------------------------------------------------

func handleOCRGet(c *gin.Context) {
	info := authorized.GetInfo(c)
	scope := ocrScope(info)

	providers := make([]OCRProviderConfig, 0, len(ocrPresets))
	for _, preset := range ocrPresets {
		cfg := OCRProviderConfig{
			PresetKey:   preset.Key,
			Enabled:     false,
			FieldValues: map[string]string{},
			Status:      "unconfigured",
		}

		if setting.Global != nil {
			saved, _ := setting.Global.Get(scope, ocrProviderNS(preset.Key))
			if saved != nil {
				if v, ok := saved["enabled"].(bool); ok {
					cfg.Enabled = v
				}
				if v, ok := saved["status"].(string); ok && v != "" {
					cfg.Status = v
				}
				pwFields := ocrPasswordFields(&preset)
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

	var assignment OCRToolAssignment
	if setting.Global != nil {
		saved, _ := setting.Global.Get(scope, ocrAssignmentNS)
		if saved != nil {
			if v, ok := saved["ocr_recognize"].(string); ok && v != "" {
				assignment.OCRRecognize = &v
			}
		}
	}

	response.RespondWithSuccess(c, http.StatusOK, OCRPageData{
		Presets:        ocrPresets,
		Providers:      providers,
		ToolAssignment: assignment,
	})
}

// ---------------------------------------------------------------------------
// PUT /setting/ocr/providers/:key
// ---------------------------------------------------------------------------

func handleOCRProviderUpdate(c *gin.Context) {
	if !guardOwner(c) {
		return
	}

	key := c.Param("key")
	preset := ocrFindPreset(key)
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
	scope := ocrScope(info)

	existing, _ := setting.Global.Get(scope, ocrProviderNS(key))
	m := make(map[string]interface{})
	for k, v := range existing {
		m[k] = v
	}

	validFields := make(map[string]bool)
	for _, f := range preset.Fields {
		validFields[f.Key] = true
	}

	pwFields := ocrPasswordFields(preset)
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
				continue
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

	if _, err := setting.Global.Set(scope, ocrProviderNS(key), m); err != nil {
		respondError(c, http.StatusInternalServerError, err.Error())
		return
	}

	cfg := OCRProviderConfig{
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
// PUT /setting/ocr/providers/:key/toggle
// ---------------------------------------------------------------------------

func handleOCRProviderToggle(c *gin.Context) {
	if !guardOwner(c) {
		return
	}

	key := c.Param("key")
	preset := ocrFindPreset(key)
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
	scope := ocrScope(info)

	existing, _ := setting.Global.Get(scope, ocrProviderNS(key))
	m := make(map[string]interface{})
	for k, v := range existing {
		m[k] = v
	}
	m["enabled"] = body.Enabled

	if _, err := setting.Global.Set(scope, ocrProviderNS(key), m); err != nil {
		respondError(c, http.StatusInternalServerError, err.Error())
		return
	}

	if !body.Enabled {
		assignData, _ := setting.Global.Get(scope, ocrAssignmentNS)
		if assignData != nil {
			if v, ok := assignData["ocr_recognize"].(string); ok && v == key {
				assignData["ocr_recognize"] = ""
				setting.Global.Set(scope, ocrAssignmentNS, assignData)
			}
		}
	}

	cfg := OCRProviderConfig{
		PresetKey:   key,
		Enabled:     body.Enabled,
		FieldValues: map[string]string{},
		Status:      "unconfigured",
	}
	if v, ok := m["status"].(string); ok && v != "" {
		cfg.Status = v
	}
	pwFields := ocrPasswordFields(preset)
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
// POST /setting/ocr/providers/:key/test
// ---------------------------------------------------------------------------

func handleOCRProviderTest(c *gin.Context) {
	if !guardOwner(c) {
		return
	}

	key := c.Param("key")
	preset := ocrFindPreset(key)
	if preset == nil {
		respondError(c, http.StatusBadRequest, fmt.Sprintf("unknown provider: %s", key))
		return
	}

	var body struct {
		FieldValues map[string]string `json:"field_values"`
	}
	c.ShouldBindJSON(&body)

	info := authorized.GetInfo(c)
	scope := ocrScope(info)

	resolveField := func(fieldKey string) string {
		if body.FieldValues != nil {
			if v := body.FieldValues[fieldKey]; v != "" {
				return v
			}
		}
		if setting.Global != nil {
			saved, _ := setting.Global.Get(scope, ocrProviderNS(key))
			if saved != nil {
				if fv, ok := saved["field_values"].(map[string]interface{}); ok {
					if v, ok := fv[fieldKey].(string); ok {
						return cloudDecrypt(v)
					}
				}
			}
		}
		return ""
	}

	start := time.Now()
	var testErr error

	switch key {
	case "paddleocr":
		baseURL := resolveField("base_url")
		if baseURL == "" {
			response.RespondWithSuccess(c, http.StatusOK, OCRTestResult{
				Success: false,
				Message: "Service URL is required",
			})
			return
		}
		testErr = ocrTestPaddleOCR(baseURL)

	case "baidu":
		apiKey := resolveField("api_key")
		secretKey := resolveField("secret_key")
		if apiKey == "" || secretKey == "" {
			response.RespondWithSuccess(c, http.StatusOK, OCRTestResult{
				Success: false,
				Message: "API Key and Secret Key are required",
			})
			return
		}
		testErr = ocrTestBaidu(apiKey, secretKey)

	case "google":
		apiKey := resolveField("api_key")
		if apiKey == "" {
			response.RespondWithSuccess(c, http.StatusOK, OCRTestResult{
				Success: false,
				Message: "API Key is required",
			})
			return
		}
		testErr = ocrTestGoogle(apiKey)

	case "azure":
		apiKey := resolveField("api_key")
		endpoint := resolveField("endpoint")
		if apiKey == "" || endpoint == "" {
			response.RespondWithSuccess(c, http.StatusOK, OCRTestResult{
				Success: false,
				Message: "API Key and Endpoint are required",
			})
			return
		}
		testErr = ocrTestAzure(apiKey, endpoint)

	default:
		respondError(c, http.StatusBadRequest, fmt.Sprintf("test not supported for provider: %s", key))
		return
	}

	latency := time.Since(start).Milliseconds()

	if testErr != nil {
		if setting.Global != nil {
			saved, _ := setting.Global.Get(scope, ocrProviderNS(key))
			if saved == nil {
				saved = map[string]interface{}{}
			}
			saved["status"] = "disconnected"
			setting.Global.Set(scope, ocrProviderNS(key), saved)
		}
		response.RespondWithSuccess(c, http.StatusOK, OCRTestResult{
			Success: false,
			Message: testErr.Error(),
		})
		return
	}

	if setting.Global != nil {
		saved, _ := setting.Global.Get(scope, ocrProviderNS(key))
		if saved == nil {
			saved = map[string]interface{}{}
		}
		saved["status"] = "connected"
		setting.Global.Set(scope, ocrProviderNS(key), saved)
	}

	response.RespondWithSuccess(c, http.StatusOK, OCRTestResult{
		Success:   true,
		Message:   "Connection successful",
		LatencyMs: latency,
	})
}

func ocrTestPaddleOCR(baseURL string) error {
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Get(baseURL + "/health")
	if err != nil {
		return fmt.Errorf("connection failed: %s", err.Error())
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("health check returned HTTP %d", resp.StatusCode)
	}
	return nil
}

func ocrTestBaidu(apiKey, secretKey string) error {
	client := &http.Client{Timeout: 15 * time.Second}
	tokenURL := fmt.Sprintf(
		"https://aip.baidubce.com/oauth/2.0/token?grant_type=client_credentials&client_id=%s&client_secret=%s",
		url.QueryEscape(apiKey), url.QueryEscape(secretKey),
	)
	resp, err := client.Post(tokenURL, "application/x-www-form-urlencoded", nil)
	if err != nil {
		return fmt.Errorf("connection failed: %s", err.Error())
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("authentication failed (HTTP %d)", resp.StatusCode)
	}
	return nil
}

func ocrTestGoogle(apiKey string) error {
	client := &http.Client{Timeout: 15 * time.Second}
	testURL := "https://vision.googleapis.com/v1/images:annotate?key=" + url.QueryEscape(apiKey)
	resp, err := client.Post(testURL, "application/json", nil)
	if err != nil {
		return fmt.Errorf("connection failed: %s", err.Error())
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		return fmt.Errorf("invalid API key (HTTP %d)", resp.StatusCode)
	}
	// 400 INVALID_ARGUMENT is expected with empty body — proves key is valid
	return nil
}

func ocrTestAzure(apiKey, endpoint string) error {
	client := &http.Client{Timeout: 15 * time.Second}
	infoURL := endpoint + "/documentintelligence/info?api-version=2024-11-30"
	req, err := http.NewRequest("GET", infoURL, nil)
	if err != nil {
		return fmt.Errorf("failed to build request: %s", err.Error())
	}
	req.Header.Set("Ocp-Apim-Subscription-Key", apiKey)
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

// ---------------------------------------------------------------------------
// PUT /setting/ocr/tool-assignment
// ---------------------------------------------------------------------------

func handleOCRToolAssignment(c *gin.Context) {
	if !guardOwner(c) {
		return
	}

	var body struct {
		OCRRecognize *string `json:"ocr_recognize"`
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
	scope := ocrScope(info)

	if body.OCRRecognize != nil && *body.OCRRecognize != "" {
		val := *body.OCRRecognize
		if strings.HasPrefix(val, "llm:") {
			connID := strings.TrimPrefix(val, "llm:")
			conn, err := ocrSelectConnector(connID)
			if err != nil {
				respondError(c, http.StatusBadRequest, fmt.Sprintf("LLM connector not found: %s", connID))
				return
			}
			caps := agentllm.GetCapabilitiesFromConn(conn)
			if caps == nil || !caps.OCR {
				respondError(c, http.StatusBadRequest, fmt.Sprintf("LLM connector %s does not have OCR capability", connID))
				return
			}
		} else {
			preset := ocrFindPreset(val)
			if preset == nil {
				respondError(c, http.StatusBadRequest, fmt.Sprintf("unknown provider: %s", val))
				return
			}

			hasOCR := false
			for _, t := range preset.Tools {
				if t == "ocr_recognize" {
					hasOCR = true
					break
				}
			}
			if !hasOCR {
				respondError(c, http.StatusBadRequest, fmt.Sprintf("provider %s does not support ocr_recognize", val))
				return
			}

			if len(preset.Fields) > 0 {
				saved, _ := setting.Global.Get(scope, ocrProviderNS(val))
				if saved == nil {
					respondError(c, http.StatusBadRequest, fmt.Sprintf("provider %s is not configured", val))
					return
				}
				if v, ok := saved["enabled"].(bool); !ok || !v {
					respondError(c, http.StatusBadRequest, fmt.Sprintf("provider %s is not enabled", val))
					return
				}
			}
		}
	}

	m := make(map[string]interface{})
	if body.OCRRecognize != nil {
		m["ocr_recognize"] = *body.OCRRecognize
	} else {
		m["ocr_recognize"] = ""
	}

	if _, err := setting.Global.Set(scope, ocrAssignmentNS, m); err != nil {
		respondError(c, http.StatusInternalServerError, err.Error())
		return
	}

	result := OCRToolAssignment{}
	if v, ok := m["ocr_recognize"].(string); ok && v != "" {
		result.OCRRecognize = &v
	}

	response.RespondWithSuccess(c, http.StatusOK, result)
}
