package ocr

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

// BaiduHandler implements ProviderHandler for Baidu Cloud OCR.
type BaiduHandler struct {
	APIKey    string
	SecretKey string
}

var baiduSupportedTypes = map[string]bool{
	"general":         true,
	"table":           true,
	"handwriting":     true,
	"invoice":         true,
	"receipt":         true,
	"id_card":         true,
	"bank_card":       true,
	"license":         true,
	"vehicle_license": true,
	"passport":        true,
	"license_plate":   true,
}

func (h *BaiduHandler) SupportedTypes() map[string]bool {
	return baiduSupportedTypes
}

func (h *BaiduHandler) Recognize(ctx context.Context, req *OCRRequest) (*OCRResponse, error) {
	if h.APIKey == "" || h.SecretKey == "" {
		return nil, fmt.Errorf("Baidu OCR api_key and secret_key are required")
	}

	token, err := getBaiduAccessToken(ctx, h.APIKey, h.SecretKey)
	if err != nil {
		return nil, fmt.Errorf("Baidu auth: %w", err)
	}

	endpoint := baiduEndpoint(req.Type, req.Mode, req.OutputFormat)
	reqURL := fmt.Sprintf("https://aip.baidubce.com/rest/2.0/ocr/v1/%s?access_token=%s", endpoint, url.QueryEscape(token))

	form := url.Values{}
	b64 := base64.StdEncoding.EncodeToString(req.Source)
	if req.MimeType == "application/pdf" {
		form.Set("pdf_file", b64)
	} else {
		form.Set("image", b64)
	}
	if req.Language != "" {
		form.Set("language_type", baiduLanguage(req.Language))
	}

	// Extra pass-through
	for k, v := range req.Extra {
		if s, ok := v.(string); ok {
			form.Set(k, s)
		}
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", reqURL, strings.NewReader(form.Encode()))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	client := &http.Client{Timeout: 30 * time.Second}
	httpResp, err := client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("Baidu OCR request failed: %w", err)
	}
	defer httpResp.Body.Close()

	respBody, err := io.ReadAll(httpResp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	if httpResp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Baidu OCR returned HTTP %d: %s", httpResp.StatusCode, string(respBody))
	}

	return parseBaiduResponse(respBody, req.Type)
}

// baiduEndpoint selects the Baidu OCR API endpoint based on type, mode, and output_format.
func baiduEndpoint(ocrType, mode, outputFormat string) string {
	switch ocrType {
	case "table":
		return "table"
	case "handwriting":
		return "handwriting"
	case "invoice":
		return "vat_invoice"
	case "receipt":
		return "receipt"
	case "id_card":
		return "idcard"
	case "bank_card":
		return "bankcard"
	case "license":
		return "business_license"
	case "vehicle_license":
		return "vehicle_license"
	case "passport":
		return "passport"
	case "license_plate":
		return "license_plate"
	default: // "general"
		needLocation := outputFormat == "json" || outputFormat == "markdown"
		if mode == "accurate" {
			if needLocation {
				return "accurate" // with location info
			}
			return "accurate_basic"
		}
		if needLocation {
			return "general" // with location info
		}
		return "general_basic"
	}
}

// baiduLanguage maps ISO 639-1 codes to Baidu's language_type values.
func baiduLanguage(lang string) string {
	m := map[string]string{
		"zh": "CHN_ENG",
		"en": "ENG",
		"ja": "JAP",
		"ko": "KOR",
		"fr": "FRE",
		"de": "GER",
		"ru": "RUS",
		"es": "SPA",
		"pt": "POR",
		"it": "ITA",
	}
	if v, ok := m[lang]; ok {
		return v
	}
	return "CHN_ENG"
}

// parseBaiduResponse converts Baidu OCR JSON response to unified OCRResponse.
func parseBaiduResponse(data []byte, ocrType string) (*OCRResponse, error) {
	var raw map[string]interface{}
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("parse Baidu response: %w", err)
	}

	// Check for API error
	if errMsg, ok := raw["error_msg"].(string); ok && errMsg != "" {
		errCode, _ := raw["error_code"].(float64)
		return nil, fmt.Errorf("Baidu OCR error %d: %s", int(errCode), errMsg)
	}

	resp := &OCRResponse{Pages: 1}

	// Structured types → Fields
	switch ocrType {
	case "invoice", "receipt", "id_card", "bank_card", "license",
		"vehicle_license", "passport", "license_plate":
		resp.Fields = extractBaiduFields(raw)
		resp.Text = fieldsToText(resp.Fields)
		return resp, nil
	}

	// General/table/handwriting → words_result
	wordsResult, _ := raw["words_result"].([]interface{})
	var textParts []string
	for _, item := range wordsResult {
		m, ok := item.(map[string]interface{})
		if !ok {
			continue
		}
		words, _ := m["words"].(string)
		if words == "" {
			continue
		}
		textParts = append(textParts, words)

		block := OCRBlock{Text: words, Page: 1}
		if loc, ok := m["location"].(map[string]interface{}); ok {
			block.BBox = baiduLocationToBBox(loc)
		}
		if prob, ok := m["probability"].(map[string]interface{}); ok {
			if avg, ok := prob["average"].(float64); ok {
				block.Confidence = avg
			}
		}
		resp.Blocks = append(resp.Blocks, block)
	}

	resp.Text = strings.Join(textParts, "\n")

	if num, ok := raw["pages_number"].(float64); ok {
		resp.Pages = int(num)
	}

	return resp, nil
}

// extractBaiduFields extracts key-value pairs from Baidu structured OCR responses.
func extractBaiduFields(raw map[string]interface{}) map[string]interface{} {
	fields := map[string]interface{}{}

	// invoice / receipt: words_result is a map of field_name → {words: "value"}
	if wr, ok := raw["words_result"].(map[string]interface{}); ok {
		for k, v := range wr {
			if m, ok := v.(map[string]interface{}); ok {
				if words, ok := m["words"].(string); ok {
					fields[k] = words
				}
			} else if s, ok := v.(string); ok {
				fields[k] = s
			}
		}
	}

	// id_card / bank_card: result field
	if result, ok := raw["result"].(map[string]interface{}); ok {
		for k, v := range result {
			if m, ok := v.(map[string]interface{}); ok {
				if words, ok := m["words"].(string); ok {
					fields[k] = words
				}
			} else {
				fields[k] = v
			}
		}
	}

	return fields
}

// baiduLocationToBBox converts Baidu's location format to [x1, y1, x2, y2].
func baiduLocationToBBox(loc map[string]interface{}) []float64 {
	left, _ := loc["left"].(float64)
	top, _ := loc["top"].(float64)
	width, _ := loc["width"].(float64)
	height, _ := loc["height"].(float64)
	return []float64{left, top, left + width, top + height}
}

// fieldsToText converts a Fields map to readable text.
func fieldsToText(fields map[string]interface{}) string {
	var parts []string
	for k, v := range fields {
		parts = append(parts, fmt.Sprintf("%s: %v", k, v))
	}
	return strings.Join(parts, "\n")
}

// --- Baidu access_token cache ---

var (
	baiduTokenCache   = map[string]*baiduToken{}
	baiduTokenCacheMu sync.Mutex
)

type baiduToken struct {
	Token     string
	ExpiresAt time.Time
}

// getBaiduAccessToken returns a cached or fresh access_token.
func getBaiduAccessToken(ctx context.Context, apiKey, secretKey string) (string, error) {
	cacheKey := apiKey + ":" + secretKey

	baiduTokenCacheMu.Lock()
	if cached, ok := baiduTokenCache[cacheKey]; ok && time.Now().Before(cached.ExpiresAt) {
		baiduTokenCacheMu.Unlock()
		return cached.Token, nil
	}
	baiduTokenCacheMu.Unlock()

	tokenURL := fmt.Sprintf(
		"https://aip.baidubce.com/oauth/2.0/token?grant_type=client_credentials&client_id=%s&client_secret=%s",
		url.QueryEscape(apiKey), url.QueryEscape(secretKey),
	)

	httpReq, err := http.NewRequestWithContext(ctx, "POST", tokenURL, nil)
	if err != nil {
		return "", fmt.Errorf("create token request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(httpReq)
	if err != nil {
		return "", fmt.Errorf("token request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("read token response: %w", err)
	}

	var tokenResp struct {
		AccessToken string `json:"access_token"`
		ExpiresIn   int64  `json:"expires_in"`
		Error       string `json:"error"`
		ErrorDesc   string `json:"error_description"`
	}
	if err := json.Unmarshal(body, &tokenResp); err != nil {
		return "", fmt.Errorf("parse token response: %w", err)
	}
	if tokenResp.Error != "" {
		return "", fmt.Errorf("Baidu auth failed: %s (%s)", tokenResp.ErrorDesc, tokenResp.Error)
	}

	// Cache with 1-hour safety margin
	expiresAt := time.Now().Add(time.Duration(tokenResp.ExpiresIn-3600) * time.Second)

	baiduTokenCacheMu.Lock()
	baiduTokenCache[cacheKey] = &baiduToken{
		Token:     tokenResp.AccessToken,
		ExpiresAt: expiresAt,
	}
	baiduTokenCacheMu.Unlock()

	return tokenResp.AccessToken, nil
}

// clearBaiduTokenCache removes all cached tokens (used in tests).
func clearBaiduTokenCache() {
	baiduTokenCacheMu.Lock()
	baiduTokenCache = map[string]*baiduToken{}
	baiduTokenCacheMu.Unlock()
}

// parseBaiduResponseBytes is a convenience wrapper for external callers
// that provides a consistent bytes → OCRResponse path.
func parseBaiduResponseBytes(data []byte) (*OCRResponse, error) {
	return parseBaiduResponse(data, "general")
}
