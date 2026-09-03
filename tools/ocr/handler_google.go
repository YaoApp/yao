package ocr

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

// GoogleHandler implements ProviderHandler for Google Cloud Vision OCR.
type GoogleHandler struct {
	APIKey string
}

var googleSupportedTypes = map[string]bool{
	"general":     true,
	"document":    true,
	"handwriting": true,
}

func (h *GoogleHandler) SupportedTypes() map[string]bool {
	return googleSupportedTypes
}

func (h *GoogleHandler) Recognize(ctx context.Context, req *OCRRequest) (*OCRResponse, error) {
	if req.MimeType == "application/pdf" {
		return nil, fmt.Errorf("Google Cloud Vision does not support PDF; use Baidu, Azure, or PaddleOCR")
	}
	if h.APIKey == "" {
		return nil, fmt.Errorf("Google Vision api_key is required")
	}

	featureType := "DOCUMENT_TEXT_DETECTION"
	if req.Mode == "standard" {
		featureType = "TEXT_DETECTION"
	}

	body := map[string]interface{}{
		"requests": []map[string]interface{}{
			{
				"image": map[string]string{
					"content": base64.StdEncoding.EncodeToString(req.Source),
				},
				"features": []map[string]interface{}{
					{"type": featureType, "maxResults": 50},
				},
			},
		},
	}

	// Add language hints
	if req.Language != "" {
		body["requests"].([]map[string]interface{})[0]["imageContext"] = map[string]interface{}{
			"languageHints": []string{req.Language},
		}
	}

	reqBody, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	apiURL := "https://vision.googleapis.com/v1/images:annotate?key=" + url.QueryEscape(h.APIKey)
	httpReq, err := http.NewRequestWithContext(ctx, "POST", apiURL, bytes.NewReader(reqBody))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 30 * time.Second}
	httpResp, err := client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("Google Vision request failed: %w", err)
	}
	defer httpResp.Body.Close()

	respBody, err := io.ReadAll(httpResp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	if httpResp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Google Vision returned HTTP %d: %s", httpResp.StatusCode, string(respBody))
	}

	return parseGoogleResponse(respBody)
}

// parseGoogleResponse converts Google Vision API response to unified OCRResponse.
func parseGoogleResponse(data []byte) (*OCRResponse, error) {
	var raw struct {
		Responses []struct {
			FullTextAnnotation *struct {
				Text  string                   `json:"text"`
				Pages []map[string]interface{} `json:"pages"`
			} `json:"fullTextAnnotation"`
			TextAnnotations []struct {
				Description  string `json:"description"`
				BoundingPoly struct {
					Vertices []struct {
						X float64 `json:"x"`
						Y float64 `json:"y"`
					} `json:"vertices"`
				} `json:"boundingPoly"`
			} `json:"textAnnotations"`
			Error *struct {
				Code    int    `json:"code"`
				Message string `json:"message"`
			} `json:"error"`
		} `json:"responses"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("parse Google Vision response: %w", err)
	}

	if len(raw.Responses) == 0 {
		return &OCRResponse{Text: "", Pages: 1}, nil
	}

	r := raw.Responses[0]

	if r.Error != nil {
		return nil, fmt.Errorf("Google Vision error %d: %s", r.Error.Code, r.Error.Message)
	}

	resp := &OCRResponse{Pages: 1}

	// Full text from fullTextAnnotation
	if r.FullTextAnnotation != nil {
		resp.Text = r.FullTextAnnotation.Text
		resp.Pages = max(1, len(r.FullTextAnnotation.Pages))
	}

	// Individual text annotations → blocks (skip first which is the full text)
	for i, ann := range r.TextAnnotations {
		if i == 0 {
			if resp.Text == "" {
				resp.Text = ann.Description
			}
			continue
		}
		block := OCRBlock{Text: ann.Description, Page: 1}
		if len(ann.BoundingPoly.Vertices) >= 4 {
			v := ann.BoundingPoly.Vertices
			block.BBox = []float64{v[0].X, v[0].Y, v[2].X, v[2].Y}
		}
		resp.Blocks = append(resp.Blocks, block)
	}

	return resp, nil
}
