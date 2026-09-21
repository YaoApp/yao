package ocr

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"
)

// TaoHandler implements ProviderHandler for Tao Service OCR.
type TaoHandler struct {
	BaseURL string
	APIKey  string
}

// ---------------------------------------------------------------------------
// Dynamic OCR type cache — fetched from Tao, hardcoded fallback
// ---------------------------------------------------------------------------

// fallbackTaoTypes is the static seed used when the Tao API is unreachable.
var fallbackTaoTypes = map[string]bool{
	"general_basic":  true,
	"accurate_basic": true,
	"table":          true,
	"handwriting":    true,
	"idcard":         true,
	"bankcard":       true,
}

// taoTypeAliases maps internal type names to Tao API equivalents.
var taoTypeAliases = map[string]string{
	"general":   "general_basic",
	"id_card":   "idcard",
	"bank_card": "bankcard",
}

var (
	taoOCRTypeCacheMu sync.Mutex
	taoOCRTypeCache   map[string]bool // nil = not fetched yet
)

// SetTaoOCRTypes replaces the cached type set (called during Tao setup).
func SetTaoOCRTypes(types map[string]bool) {
	taoOCRTypeCacheMu.Lock()
	defer taoOCRTypeCacheMu.Unlock()
	taoOCRTypeCache = types
}

// InvalidateTaoOCRTypes clears the cache so the next request re-fetches.
func InvalidateTaoOCRTypes() {
	taoOCRTypeCacheMu.Lock()
	defer taoOCRTypeCacheMu.Unlock()
	taoOCRTypeCache = nil
}

// FetchTaoOCRTypes fetches OCR types from the Tao API and caches them.
// Returns the fetched set, or nil if the fetch failed (caller should use fallback).
func FetchTaoOCRTypes(baseURL, apiKey string) map[string]bool {
	types := fetchTaoOCRTypesFromAPI(baseURL, apiKey)
	if types != nil && len(types) > 0 {
		taoOCRTypeCacheMu.Lock()
		taoOCRTypeCache = types
		taoOCRTypeCacheMu.Unlock()
		return types
	}
	return nil
}

// fetchTaoOCRTypesFromAPI probes GET /v1/ocr for available types.
// Expected response: {"types":["general_basic","accurate_basic",...]}
// Returns nil on error or unexpected format (caller falls back to hardcoded).
func fetchTaoOCRTypesFromAPI(baseURL, apiKey string) map[string]bool {
	url := strings.TrimRight(baseURL, "/") + "/v1/ocr"
	client := &http.Client{Timeout: 10 * time.Second}
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil
	}
	req.Header.Set("Authorization", "Bearer "+apiKey)

	resp, err := client.Do(req)
	if err != nil {
		return nil
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil
	}

	var result struct {
		Types []string `json:"types"`
	}
	if json.Unmarshal(body, &result) != nil || len(result.Types) == 0 {
		return nil
	}

	m := make(map[string]bool, len(result.Types))
	for _, t := range result.Types {
		m[t] = true
	}
	return m
}

// getTaoNativeTypes returns the cached type set, falling back to the hardcoded seed.
func getTaoNativeTypes() map[string]bool {
	taoOCRTypeCacheMu.Lock()
	defer taoOCRTypeCacheMu.Unlock()
	if taoOCRTypeCache != nil {
		return taoOCRTypeCache
	}
	return fallbackTaoTypes
}

func (h *TaoHandler) SupportedTypes() map[string]bool {
	native := getTaoNativeTypes()
	m := make(map[string]bool, len(native)+len(taoTypeAliases))
	for k := range native {
		m[k] = true
	}
	for k := range taoTypeAliases {
		m[k] = true
	}
	return m
}

func (h *TaoHandler) Recognize(ctx context.Context, req *OCRRequest) (*OCRResponse, error) {
	if h.BaseURL == "" || h.APIKey == "" {
		return nil, fmt.Errorf("Tao OCR base_url and api_key are required")
	}

	// Lazy-fetch OCR types on first request if cache is empty.
	taoOCRTypeCacheMu.Lock()
	needsFetch := taoOCRTypeCache == nil
	taoOCRTypeCacheMu.Unlock()
	if needsFetch {
		FetchTaoOCRTypes(h.BaseURL, h.APIKey)
	}

	taoType := resolveTaoOCRType(req.Type)

	body := map[string]interface{}{
		"image": base64.StdEncoding.EncodeToString(req.Source),
	}
	if taoType == "idcard" {
		side, _ := req.Extra["id_card_side"].(string)
		if side == "" {
			side = "front"
		}
		body["id_card_side"] = side
	}

	bodyBytes, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	endpoint := strings.TrimRight(h.BaseURL, "/") + "/v1/ocr/" + taoType
	httpReq, err := http.NewRequestWithContext(ctx, "POST", endpoint, bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+h.APIKey)

	client := &http.Client{Timeout: 60 * time.Second}
	httpResp, err := client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("Tao OCR request failed: %w", err)
	}
	defer httpResp.Body.Close()

	respBody, err := io.ReadAll(httpResp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	if httpResp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Tao OCR returned HTTP %d: %s", httpResp.StatusCode, string(respBody))
	}

	return parseTaoOCRResponse(respBody, taoType)
}

// resolveTaoOCRType resolves a type name to the Tao API path segment.
// Accepts Tao native names (e.g. "idcard"), internal aliases (e.g. "id_card"),
// and dynamically fetched types.
func resolveTaoOCRType(ocrType string) string {
	native := getTaoNativeTypes()
	if native[ocrType] {
		return ocrType
	}
	if alias, ok := taoTypeAliases[ocrType]; ok {
		return alias
	}
	return "general_basic"
}

// parseTaoOCRResponse converts Tao OCR JSON response to unified OCRResponse.
func parseTaoOCRResponse(data []byte, ocrType string) (*OCRResponse, error) {
	var raw map[string]interface{}
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("parse Tao OCR response: %w", err)
	}

	if errMsg, _ := raw["error"].(string); errMsg != "" {
		return nil, fmt.Errorf("Tao OCR error: %s", errMsg)
	}
	if errObj, ok := raw["error"].(map[string]interface{}); ok {
		msg, _ := errObj["message"].(string)
		return nil, fmt.Errorf("Tao OCR error: %s", msg)
	}

	// idcard / bankcard: upstream returns HTTP 200 with image_status on content mismatch.
	// These are deterministic failures — do not retry.
	if err := checkCardStatus(raw, ocrType); err != nil {
		return nil, err
	}

	resp := &OCRResponse{Pages: 1}

	if text, ok := raw["text"].(string); ok {
		resp.Text = text
	}

	// Table-specific: parse tables_result / table_num
	if ocrType == "table" {
		return parseTaoTableResponse(raw, resp)
	}

	if results, ok := raw["words_result"].([]interface{}); ok {
		parseWordsResult(results, resp)
	}

	if num, ok := raw["pages_number"].(float64); ok {
		resp.Pages = int(num)
	}

	return resp, nil
}

// checkCardStatus detects card-type content mismatches that the upstream returns
// as HTTP 200 with image_status / error_code in the response body.
// These are deterministic failures (wrong card content) and must not be retried.
func checkCardStatus(raw map[string]interface{}, ocrType string) error {
	// image_status: "other_type_card" — idcard/bankcard image doesn't match expected type
	if imgStatus, ok := raw["image_status"].(string); ok && imgStatus != "normal" && imgStatus != "" {
		wordsNum, _ := raw["words_result_num"].(float64)
		if wordsNum == 0 {
			return fmt.Errorf("card content mismatch (image_status: %s): the image does not match the requested card type; do not retry", imgStatus)
		}
	}

	// error_code 216630: bankcard-specific recognition failure
	if errCode, ok := raw["error_code"].(float64); ok && errCode != 0 {
		errMsg, _ := raw["error_msg"].(string)
		if errMsg == "" {
			errMsg = "recognition failed"
		}
		return fmt.Errorf("Tao OCR error %d: %s; do not retry", int(errCode), errMsg)
	}

	return nil
}

// parseTaoTableResponse handles the table OCR response format.
func parseTaoTableResponse(raw map[string]interface{}, resp *OCRResponse) (*OCRResponse, error) {
	tableNum, _ := raw["table_num"].(float64)
	if tableNum == 0 {
		resp.Text = "未检出表格"
		return resp, nil
	}

	tables, ok := raw["tables_result"].([]interface{})
	if !ok || len(tables) == 0 {
		// Fallback: try words_result for table data
		if results, ok := raw["words_result"].([]interface{}); ok {
			parseWordsResult(results, resp)
		}
		if resp.Text == "" {
			resp.Text = "未检出表格"
		}
		return resp, nil
	}

	var sb strings.Builder
	for i, tbl := range tables {
		m, ok := tbl.(map[string]interface{})
		if !ok {
			continue
		}
		if i > 0 {
			sb.WriteString("\n\n")
		}
		// tables_result[].body contains rows→cells
		if body, ok := m["body"].([]interface{}); ok {
			for _, row := range body {
				rowMap, ok := row.(map[string]interface{})
				if !ok {
					continue
				}
				cellText, _ := rowMap["words"].(string)
				if cellText != "" {
					sb.WriteString(cellText)
					sb.WriteString("\t")
				}
			}
		}
		// Fallback: header field
		if header, ok := m["header"].([]interface{}); ok {
			for _, h := range header {
				hm, ok := h.(map[string]interface{})
				if !ok {
					continue
				}
				words, _ := hm["words"].(string)
				if words != "" {
					sb.WriteString(words)
					sb.WriteString("\t")
				}
			}
		}
	}

	resp.Text = strings.TrimSpace(sb.String())
	if resp.Text == "" {
		resp.Text = "未检出表格"
	}

	if num, ok := raw["pages_number"].(float64); ok {
		resp.Pages = int(num)
	}
	return resp, nil
}

// parseWordsResult extracts text blocks from the standard words_result array.
func parseWordsResult(results []interface{}, resp *OCRResponse) {
	var textParts []string
	for _, item := range results {
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
			left, _ := loc["left"].(float64)
			top, _ := loc["top"].(float64)
			width, _ := loc["width"].(float64)
			height, _ := loc["height"].(float64)
			block.BBox = []float64{left, top, left + width, top + height}
		}
		if prob, ok := m["probability"].(map[string]interface{}); ok {
			if avg, ok := prob["average"].(float64); ok {
				block.Confidence = avg
			}
		}
		resp.Blocks = append(resp.Blocks, block)
	}
	if resp.Text == "" {
		resp.Text = strings.Join(textParts, "\n")
	}
}
