package ocr

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// AzureHandler implements ProviderHandler for Azure Document Intelligence.
type AzureHandler struct {
	APIKey   string
	Endpoint string
}

var azureSupportedTypes = map[string]bool{
	"general":  true,
	"table":    true,
	"invoice":  true,
	"receipt":  true,
	"id_card":  true,
	"passport": true,
	"document": true,
}

func (h *AzureHandler) SupportedTypes() map[string]bool {
	return azureSupportedTypes
}

func (h *AzureHandler) Recognize(ctx context.Context, req *OCRRequest) (*OCRResponse, error) {
	if h.APIKey == "" || h.Endpoint == "" {
		return nil, fmt.Errorf("Azure Document Intelligence api_key and endpoint are required")
	}

	modelID := azureModelID(req.Type)
	apiVersion := "2024-11-30"
	if v, ok := req.Extra["api_version"].(string); ok && v != "" {
		apiVersion = v
	}
	if m, ok := req.Extra["model_id"].(string); ok && m != "" {
		modelID = m
	}

	outputContentFormat := "text"
	if req.OutputFormat == "markdown" {
		outputContentFormat = "markdown"
	}

	analyzeURL := fmt.Sprintf(
		"%s/documentintelligence/documentModels/%s:analyze?api-version=%s&outputContentFormat=%s",
		strings.TrimRight(h.Endpoint, "/"), modelID, apiVersion, outputContentFormat,
	)
	if req.Pages != "" {
		analyzeURL += "&pages=" + req.Pages
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", analyzeURL, bytes.NewReader(req.Source))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	httpReq.Header.Set("Ocp-Apim-Subscription-Key", h.APIKey)
	httpReq.Header.Set("Content-Type", azureContentType(req.MimeType))

	client := &http.Client{Timeout: 120 * time.Second}
	httpResp, err := client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("Azure request failed: %w", err)
	}
	defer httpResp.Body.Close()

	// Azure returns 202 Accepted with Operation-Location for async polling
	if httpResp.StatusCode == http.StatusAccepted {
		opURL := httpResp.Header.Get("Operation-Location")
		if opURL == "" {
			return nil, fmt.Errorf("Azure returned 202 but no Operation-Location header")
		}
		return h.pollResult(ctx, opURL, req.Type)
	}

	respBody, err := io.ReadAll(httpResp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	if httpResp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Azure returned HTTP %d: %s", httpResp.StatusCode, string(respBody))
	}

	return parseAzureResponse(respBody, req.Type)
}

// pollResult polls the Azure async operation until completion.
func (h *AzureHandler) pollResult(ctx context.Context, opURL string, ocrType string) (*OCRResponse, error) {
	client := &http.Client{Timeout: 30 * time.Second}

	for i := 0; i < 60; i++ {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(2 * time.Second):
		}

		req, err := http.NewRequestWithContext(ctx, "GET", opURL, nil)
		if err != nil {
			return nil, fmt.Errorf("create poll request: %w", err)
		}
		req.Header.Set("Ocp-Apim-Subscription-Key", h.APIKey)

		resp, err := client.Do(req)
		if err != nil {
			return nil, fmt.Errorf("poll request failed: %w", err)
		}

		body, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			return nil, fmt.Errorf("read poll response: %w", err)
		}

		status := extractJSONStr(body, "status")
		switch status {
		case "succeeded":
			return parseAzureResponse(body, ocrType)
		case "failed":
			errMsg := extractJSONStr(body, "error")
			return nil, fmt.Errorf("Azure analysis failed: %s", errMsg)
		case "running", "notStarted":
			continue
		default:
			return nil, fmt.Errorf("Azure unknown status: %s", status)
		}
	}

	return nil, fmt.Errorf("Azure analysis timed out after 120 seconds")
}

// azureModelID maps OCR type to Azure model ID.
func azureModelID(ocrType string) string {
	switch ocrType {
	case "invoice":
		return "prebuilt-invoice"
	case "receipt":
		return "prebuilt-receipt"
	case "id_card", "passport":
		return "prebuilt-idDocument"
	case "table", "document":
		return "prebuilt-layout"
	default:
		return "prebuilt-read"
	}
}

// azureContentType maps MIME types to Azure-accepted content types.
func azureContentType(mimeType string) string {
	switch {
	case strings.HasPrefix(mimeType, "image/"):
		return mimeType
	case mimeType == "application/pdf":
		return "application/pdf"
	default:
		return "application/octet-stream"
	}
}

// parseAzureResponse converts Azure Document Intelligence response to unified OCRResponse.
func parseAzureResponse(data []byte, ocrType string) (*OCRResponse, error) {
	var raw map[string]interface{}
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("parse Azure response: %w", err)
	}

	// Azure wraps result in "analyzeResult"
	analyzeResult, _ := raw["analyzeResult"].(map[string]interface{})
	if analyzeResult == nil {
		analyzeResult = raw
	}

	resp := &OCRResponse{Pages: 1}

	// Extract content (text or markdown depending on request)
	if content, ok := analyzeResult["content"].(string); ok {
		resp.Text = content
	}

	// Extract pages count
	if pages, ok := analyzeResult["pages"].([]interface{}); ok {
		resp.Pages = len(pages)
	}

	// Extract structured fields for invoice/receipt/id_card
	if documents, ok := analyzeResult["documents"].([]interface{}); ok && len(documents) > 0 {
		if doc, ok := documents[0].(map[string]interface{}); ok {
			if fields, ok := doc["fields"].(map[string]interface{}); ok {
				resp.Fields = extractAzureFields(fields)
			}
		}
	}

	// Extract text blocks from pages → lines
	if pages, ok := analyzeResult["pages"].([]interface{}); ok {
		for pageIdx, page := range pages {
			if p, ok := page.(map[string]interface{}); ok {
				pageW, _ := p["width"].(float64)
				pageH, _ := p["height"].(float64)
				if lines, ok := p["lines"].([]interface{}); ok {
					for _, line := range lines {
						if l, ok := line.(map[string]interface{}); ok {
							text, _ := l["content"].(string)
							block := OCRBlock{
								Text: text,
								Page: pageIdx + 1,
							}
							if polygon, ok := l["polygon"].([]interface{}); ok && len(polygon) >= 8 && pageW > 0 && pageH > 0 {
								block.BBox = azurePolygonToBBox(polygon, pageW, pageH)
							}
							resp.Blocks = append(resp.Blocks, block)
						}
					}
				}
			}
		}
	}

	return resp, nil
}

// extractAzureFields extracts key-value pairs from Azure document fields.
func extractAzureFields(fields map[string]interface{}) map[string]interface{} {
	result := map[string]interface{}{}
	for k, v := range fields {
		if m, ok := v.(map[string]interface{}); ok {
			if content, ok := m["content"].(string); ok {
				result[k] = content
			} else if val, ok := m["valueString"].(string); ok {
				result[k] = val
			} else if val, ok := m["valueNumber"].(float64); ok {
				result[k] = val
			} else if val, ok := m["valueCurrency"].(map[string]interface{}); ok {
				result[k] = val
			} else {
				result[k] = m["value"]
			}
		}
	}
	return result
}

// azurePolygonToBBox converts Azure polygon [x1,y1,x2,y2,...,x4,y4] to normalized bbox.
func azurePolygonToBBox(polygon []interface{}, pageW, pageH float64) []float64 {
	if len(polygon) < 8 {
		return nil
	}
	x1, _ := polygon[0].(float64)
	y1, _ := polygon[1].(float64)
	x3, _ := polygon[4].(float64)
	y3, _ := polygon[5].(float64)
	return []float64{x1 / pageW, y1 / pageH, x3 / pageW, y3 / pageH}
}
